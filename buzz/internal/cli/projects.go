package cli

import (
	"context"
	"fmt"

	"buzz-cli/internal/client"
	"buzz-cli/internal/config"
	"buzz-cli/internal/nostr"
	"github.com/spf13/cobra"
)

func projectsCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "projects", Short: "Create and manage multi-repo projects (NIP-MP)"}

	create := &cobra.Command{
		Use:   "create <SLUG>",
		Short: "Create a new multi-repo project (NIP-MP kind:30621)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if err := validateProjectSlug(slug); err != nil {
				return err
			}
			repos, _ := cmd.Flags().GetStringArray("repo")
			if len(repos) == 0 {
				return inputError("at least one --repo is required")
			}
			name, _ := cmd.Flags().GetString("name")
			description, _ := cmd.Flags().GetString("description")
			channel, _ := cmd.Flags().GetString("channel")
			visibility, _ := cmd.Flags().GetString("visibility")
			if name != "" && len(name) > projectNameMax {
				return inputError(fmt.Sprintf("project name must not exceed %d bytes (got %d)", projectNameMax, len(name)))
			}
			if channel != "" {
				var err error
				if channel, err = validateUUIDStr(channel); err != nil {
					return err
				}
			}
			if visibility != "" {
				if err := validateVisibility(visibility); err != nil {
					return err
				}
			}

			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			caller := keys.PublicHex()
			members, err := expandAndDedupeRepos(repos, caller)
			if err != nil {
				return err
			}

			existing, err := fetchOwnProject(cmd.Context(), opts, resolved, keys, slug)
			if err != nil {
				return err
			}
			if existing != nil {
				return ExitError{Code: ExitConflict, Message: fmt.Sprintf("project %q already exists; use 'buzz projects update' to modify it", slug)}
			}

			tags := nostr.Tags{{"d", slug}}
			if name != "" {
				tags = append(tags, nostr.Tag{"name", name})
			}
			if description != "" {
				tags = append(tags, nostr.Tag{"description", description})
			}
			for _, coord := range members {
				tags = append(tags, nostr.Tag{"a", coord})
			}
			if channel != "" {
				tags = append(tags, nostr.Tag{"buzz-channel", channel})
			}
			if visibility != "" {
				tags = append(tags, nostr.Tag{"buzz-visibility", visibility})
			}
			if err := validateProjectEnvelope(tags); err != nil {
				return err
			}

			event := nostr.NewUnsignedEvent(nostr.KindProject, caller, "", tags, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign project event", err)
			}
			return opts.submitGitEvent(cmd.Context(), resolved, keys, event)
		},
	}
	create.Flags().StringArray("repo", nil, "member repository coordinate (bare id or full 30617:<owner-hex>:<repo-d>)")
	create.Flags().String("name", "", "display name")
	create.Flags().String("description", "", "description")
	create.Flags().String("channel", "", "associated Buzz channel UUID")
	create.Flags().String("visibility", "", "visibility: listed (default) or unlisted")

	get := &cobra.Command{
		Use:   "get <SLUG>",
		Short: "Get a project by slug",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if err := validateProjectSlug(slug); err != nil {
				return err
			}
			resolved, keys, owner, err := opts.resolveOwnerOrSelf(cmd, "owner")
			if err != nil {
				return err
			}
			found, err := fetchProject(cmd.Context(), opts, resolved, keys, slug, owner)
			if err != nil {
				return err
			}
			if found == nil {
				ownerDesc, _ := cmd.Flags().GetString("owner")
				if ownerDesc == "" {
					ownerDesc = "current identity"
				}
				return ExitError{Code: ExitInput, Message: fmt.Sprintf("project %q not found for %s", slug, ownerDesc)}
			}
			return opts.writeJSON(map[string]any{
				"event_id":   found.ID,
				"pubkey":     found.PubKey,
				"created_at": found.CreatedAt,
				"kind":       nostr.KindProject,
				"tags":       found.Tags,
				"content":    found.Content,
			})
		},
	}
	get.Flags().String("owner", "", "owner pubkey (64-char hex); defaults to the current identity")

	list := &cobra.Command{
		Use:   "list",
		Short: "List projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, keys, owner, err := opts.resolveOwnerOrSelf(cmd, "owner")
			if err != nil {
				return err
			}
			limit, _ := cmd.Flags().GetInt("limit")
			filter := client.Filter{"kinds": []int{nostr.KindProject}, "authors": []string{owner}}
			if cmd.Flags().Changed("limit") {
				filter["limit"] = limit
			}
			return opts.queryResolved(cmd.Context(), resolved, keys, []client.Filter{filter})
		},
	}
	list.Flags().String("owner", "", "owner pubkey (64-char hex); defaults to the current identity")
	list.Flags().Int("limit", 0, "maximum number of results")

	addRepo := &cobra.Command{
		Use:   "add-repo <SLUG>",
		Short: "Add one or more member repositories to a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if err := validateProjectSlug(slug); err != nil {
				return err
			}
			repos, _ := cmd.Flags().GetStringArray("repo")
			if len(repos) == 0 {
				return inputError("at least one --repo is required")
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			caller := keys.PublicHex()
			newMembers, err := expandAndDedupeRepos(repos, caller)
			if err != nil {
				return err
			}

			head, err := fetchOwnProject(cmd.Context(), opts, resolved, keys, slug)
			if err != nil {
				return err
			}
			if head == nil {
				return ExitError{Code: ExitInput, Message: fmt.Sprintf("project %q not found", slug)}
			}
			existingCoords := projectMemberCoords(head.Tags)
			var extra []nostr.Tag
			for _, coord := range newMembers {
				if _, ok := existingCoords[coord]; !ok {
					extra = append(extra, nostr.Tag{"a", coord})
				}
			}
			if len(extra) == 0 {
				return ExitError{Code: ExitConflict, Message: fmt.Sprintf("all requested repositories are already members of project %q", slug)}
			}

			tags := rebuildTags(head.Tags, slug, nil, extra...)
			if err := validateProjectEnvelope(tags); err != nil {
				return otherWrap("envelope validation failed", err)
			}
			event := nostr.NewUnsignedEvent(nostr.KindProject, caller, head.Content, tags, head.CreatedAt+1)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign project event", err)
			}
			return opts.submitGitEvent(cmd.Context(), resolved, keys, event)
		},
	}
	addRepo.Flags().StringArray("repo", nil, "member repository coordinate (bare id or full 30617:<owner-hex>:<repo-d>)")

	removeRepo := &cobra.Command{
		Use:   "remove-repo <SLUG>",
		Short: "Remove one or more member repositories from a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if err := validateProjectSlug(slug); err != nil {
				return err
			}
			repos, _ := cmd.Flags().GetStringArray("repo")
			if len(repos) == 0 {
				return inputError("at least one --repo is required")
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			caller := keys.PublicHex()
			toRemove, err := expandRepos(repos, caller)
			if err != nil {
				return err
			}

			head, err := fetchOwnProject(cmd.Context(), opts, resolved, keys, slug)
			if err != nil {
				return err
			}
			if head == nil {
				return ExitError{Code: ExitInput, Message: fmt.Sprintf("project %q not found", slug)}
			}
			existingCoords := projectMemberCoords(head.Tags)
			removeSet := map[string]struct{}{}
			for _, coord := range toRemove {
				if _, ok := existingCoords[coord]; !ok {
					return ExitError{Code: ExitInput, Message: fmt.Sprintf("project %q does not contain member %q", slug, coord)}
				}
				removeSet[coord] = struct{}{}
			}

			tags := rebuildTags(head.Tags, slug, func(t nostr.Tag) bool {
				if t[0] != "a" || len(t) < 2 {
					return false
				}
				_, drop := removeSet[t[1]]
				return drop
			})
			if err := validateProjectEnvelope(tags); err != nil {
				return otherWrap("envelope validation failed", err)
			}
			event := nostr.NewUnsignedEvent(nostr.KindProject, caller, head.Content, tags, head.CreatedAt+1)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign project event", err)
			}
			return opts.submitGitEvent(cmd.Context(), resolved, keys, event)
		},
	}
	removeRepo.Flags().StringArray("repo", nil, "member repository coordinate to remove (bare id or full 30617:<owner-hex>:<repo-d>)")

	update := &cobra.Command{
		Use:   "update <SLUG>",
		Short: "Update project metadata (at least one setter or clearer required)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			name, _ := cmd.Flags().GetString("name")
			clearName, _ := cmd.Flags().GetBool("clear-name")
			description, _ := cmd.Flags().GetString("description")
			clearDescription, _ := cmd.Flags().GetBool("clear-description")
			channel, _ := cmd.Flags().GetString("channel")
			clearChannel, _ := cmd.Flags().GetBool("clear-channel")
			visibility, _ := cmd.Flags().GetString("visibility")
			clearVisibility, _ := cmd.Flags().GetBool("clear-visibility")

			hasMutation := name != "" || clearName || description != "" || clearDescription ||
				channel != "" || clearChannel || visibility != "" || clearVisibility
			if !hasMutation {
				return inputError("buzz projects update requires at least one of: " +
					"--name, --clear-name, --description, --clear-description, " +
					"--channel, --clear-channel, --visibility, --clear-visibility")
			}
			if err := validateProjectSlug(slug); err != nil {
				return err
			}
			if channel != "" {
				var err error
				if channel, err = validateUUIDStr(channel); err != nil {
					return err
				}
			}
			if visibility != "" {
				if err := validateVisibility(visibility); err != nil {
					return err
				}
			}

			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			head, err := fetchOwnProject(cmd.Context(), opts, resolved, keys, slug)
			if err != nil {
				return err
			}
			if head == nil {
				return ExitError{Code: ExitInput, Message: fmt.Sprintf("project %q not found", slug)}
			}

			replacing := map[string]bool{
				"name":            clearName || name != "",
				"description":     clearDescription || description != "",
				"buzz-channel":    clearChannel || channel != "",
				"buzz-visibility": clearVisibility || visibility != "",
			}
			tags := nostr.Tags{}
			for _, tag := range head.Tags {
				if len(tag) == 0 || tag[0] == "auth" {
					continue
				}
				if replacing[tag[0]] {
					continue
				}
				tags = append(tags, tag)
			}
			if name != "" {
				tags = append(tags, nostr.Tag{"name", name})
			}
			if description != "" {
				tags = append(tags, nostr.Tag{"description", description})
			}
			if channel != "" {
				tags = append(tags, nostr.Tag{"buzz-channel", channel})
			}
			if visibility != "" {
				tags = append(tags, nostr.Tag{"buzz-visibility", visibility})
			}
			if err := validateProjectEnvelope(tags); err != nil {
				return otherWrap("envelope validation failed", err)
			}
			event := nostr.NewUnsignedEvent(nostr.KindProject, keys.PublicHex(), head.Content, tags, head.CreatedAt+1)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign project event", err)
			}
			return opts.submitGitEvent(cmd.Context(), resolved, keys, event)
		},
	}
	update.Flags().String("name", "", "set the display name")
	update.Flags().Bool("clear-name", false, "remove the display name")
	update.Flags().String("description", "", "set the description")
	update.Flags().Bool("clear-description", false, "remove the description")
	update.Flags().String("channel", "", "set the associated Buzz channel UUID")
	update.Flags().Bool("clear-channel", false, "remove the associated channel")
	update.Flags().String("visibility", "", "set visibility: listed or unlisted")
	update.Flags().Bool("clear-visibility", false, "remove the visibility tag (absence defaults to listed)")

	del := &cobra.Command{
		Use:   "delete <SLUG>",
		Short: "Delete a project (head-based tombstone; verified after submit)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if err := validateProjectSlug(slug); err != nil {
				return err
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			head, err := fetchOwnProject(cmd.Context(), opts, resolved, keys, slug)
			if err != nil {
				return err
			}
			if head == nil {
				return ExitError{Code: ExitInput, Message: fmt.Sprintf("project %q not found", slug)}
			}
			pubkeyHex := keys.PublicHex()
			coord := fmt.Sprintf("%d:%s:%s", nostr.KindProject, pubkeyHex, slug)
			event := nostr.NewUnsignedEvent(nostr.KindDeletion, pubkeyHex, "", nostr.Tags{{"a", coord}}, head.CreatedAt+1)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign project delete event", err)
			}
			relayClient, err := restClientFromResolved(resolved, keys)
			if err != nil {
				return err
			}
			raw, err := relayClient.PostEvent(cmd.Context(), event)
			if err != nil {
				return ExitError{Code: ExitRelay, Message: "publish delete event failed", Err: err}
			}
			if err := relayPublishError(raw); err != nil {
				return err
			}
			survivor, err := fetchOwnProject(cmd.Context(), opts, resolved, keys, slug)
			if err != nil {
				return err
			}
			if survivor != nil {
				return ExitError{Code: ExitConflict, Message: fmt.Sprintf(
					"project %q still exists (head at %d); a concurrent write raced the delete", slug, survivor.CreatedAt)}
			}
			return opts.writeJSON(map[string]any{"deleted": slug, "status": "ok"})
		},
	}

	cmd.AddCommand(create, get, list, addRepo, removeRepo, update, del)
	return cmd
}

func validateProjectSlug(slug string) error {
	if slug == "" {
		return inputError("project slug must not be empty")
	}
	if len(slug) > projectDMaxLen {
		return inputError(fmt.Sprintf("project slug must not exceed %d bytes (got %d)", projectDMaxLen, len(slug)))
	}
	return nil
}

func validateVisibility(vis string) error {
	if vis != "listed" && vis != "unlisted" {
		return inputError(fmt.Sprintf("visibility must be 'listed' or 'unlisted' (got %q)", vis))
	}
	return nil
}

// expandAndDedupeRepos expands and validates --repo values, rejecting a
// coordinate repeated within the same invocation.
func expandAndDedupeRepos(repos []string, callerPubkey string) ([]string, error) {
	members, err := expandRepos(repos, callerPubkey)
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	for _, coord := range members {
		if _, ok := seen[coord]; ok {
			return nil, inputError(fmt.Sprintf("duplicate --repo coordinate in this invocation: %q", coord))
		}
		seen[coord] = struct{}{}
	}
	return members, nil
}

func expandRepos(repos []string, callerPubkey string) ([]string, error) {
	members := make([]string, 0, len(repos))
	for _, raw := range repos {
		coord, err := expandRepoCoord(raw, callerPubkey)
		if err != nil {
			return nil, err
		}
		members = append(members, coord)
	}
	return members, nil
}

func projectMemberCoords(tags nostr.Tags) map[string]struct{} {
	coords := map[string]struct{}{}
	for _, tag := range tags {
		if len(tag) >= 2 && tag[0] == "a" {
			coords[tag[1]] = struct{}{}
		}
	}
	return coords
}

func fetchOwnProject(ctx context.Context, opts *rootOptions, resolved config.Resolved, keys *nostr.KeyPair, slug string) (*gitEventSnapshot, error) {
	return fetchLatestGitEvent(ctx, opts, resolved, keys, client.Filter{
		"kinds": []int{nostr.KindProject}, "authors": []string{keys.PublicHex()}, "#d": []string{slug}, "limit": 1,
	})
}

func fetchProject(ctx context.Context, opts *rootOptions, resolved config.Resolved, keys *nostr.KeyPair, slug, owner string) (*gitEventSnapshot, error) {
	return fetchLatestGitEvent(ctx, opts, resolved, keys, client.Filter{
		"kinds": []int{nostr.KindProject}, "authors": []string{owner}, "#d": []string{slug}, "limit": 1,
	})
}
