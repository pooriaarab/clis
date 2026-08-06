package cli

import (
	"context"
	"fmt"
	"strings"

	"buzz-cli/internal/client"
	"buzz-cli/internal/config"
	"buzz-cli/internal/nostr"
	"github.com/spf13/cobra"
)

func reposCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "repos", Short: "Announce and discover git repositories (NIP-34)"}

	create := &cobra.Command{
		Use:   "create",
		Short: "Announce a git repository (NIP-34)",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := requiredFlag(cmd, "id")
			if err != nil {
				return err
			}
			if id, err = validateRepoID(id); err != nil {
				return err
			}
			name, _ := cmd.Flags().GetString("name")
			description, _ := cmd.Flags().GetString("description")
			cloneURLs, _ := cmd.Flags().GetStringArray("clone")
			web, _ := cmd.Flags().GetString("web")
			relays, _ := cmd.Flags().GetStringArray("nostr-relay")
			channel, _ := cmd.Flags().GetString("channel")

			if len(name) > 128 {
				return inputError(fmt.Sprintf("name exceeds 128 characters (got %d)", len(name)))
			}
			if len(description) > 1024 {
				return inputError(fmt.Sprintf("description exceeds 1024 characters (got %d)", len(description)))
			}
			if len(cloneURLs) > 5 {
				return inputError(fmt.Sprintf("too many clone_urls (max 5, got %d)", len(cloneURLs)))
			}
			for _, url := range cloneURLs {
				if url == "" {
					return inputError("clone_url must not be empty")
				}
				if len(url) > 512 {
					return inputError(fmt.Sprintf("clone_url exceeds 512 characters (got %d)", len(url)))
				}
			}
			if web != "" {
				if !hasHTTPPrefix(web) {
					return inputError(fmt.Sprintf("web_url must start with http:// or https:// (got %q)", web))
				}
				if len(web) > 512 {
					return inputError(fmt.Sprintf("web_url exceeds 512 characters (got %d)", len(web)))
				}
			}
			if len(relays) > 10 {
				return inputError(fmt.Sprintf("too many relays (max 10, got %d)", len(relays)))
			}
			for _, relay := range relays {
				if !hasWSPrefix(relay) {
					return inputError(fmt.Sprintf("relay must start with ws:// or wss:// (got %q)", relay))
				}
				if len(relay) > 256 {
					return inputError(fmt.Sprintf("relay exceeds 256 characters (got %d)", len(relay)))
				}
			}

			tags := nostr.Tags{{"d", id}}
			if name != "" {
				tags = append(tags, nostr.Tag{"name", name})
			}
			if description != "" {
				tags = append(tags, nostr.Tag{"description", description})
			}
			if len(cloneURLs) > 0 {
				tags = append(tags, append(nostr.Tag{"clone"}, cloneURLs...))
			}
			if web != "" {
				tags = append(tags, nostr.Tag{"web", web})
			}
			if len(relays) > 0 {
				tags = append(tags, append(nostr.Tag{"relays"}, relays...))
			}
			if channel != "" {
				if channel, err = validateUUIDStr(channel); err != nil {
					return err
				}
				tags = append(tags, nostr.Tag{"buzz-channel", channel})
			}

			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			event := nostr.NewUnsignedEvent(nostr.KindGitRepoAnnouncement, keys.PublicHex(), "", tags, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign repo announcement event", err)
			}
			return opts.publishWithLink(cmd.Context(), resolved, keys, event, "link", repoLink(keys.PublicHex(), id))
		},
	}
	create.Flags().String("id", "", "repository identifier")
	create.Flags().String("name", "", "human-readable display name")
	create.Flags().String("description", "", "repository description")
	create.Flags().StringArray("clone", nil, "clone URL (repeatable)")
	create.Flags().String("web", "", "web browsing URL")
	create.Flags().StringArray("nostr-relay", nil, "preferred relay for repo discovery (repeatable)")
	create.Flags().String("channel", "", "channel UUID to bind the repo to")

	get := &cobra.Command{
		Use:   "get",
		Short: "Get a repository announcement",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := requiredFlag(cmd, "id")
			if err != nil {
				return err
			}
			if id, err = validateRepoID(id); err != nil {
				return err
			}
			owner, _ := cmd.Flags().GetString("owner")
			filter := client.Filter{"kinds": []int{nostr.KindGitRepoAnnouncement}, "#d": []string{id}}
			if owner != "" {
				if owner, err = validateHex64(owner); err != nil {
					return err
				}
				filter["authors"] = []string{owner}
			}
			return opts.query(cmd.Context(), []client.Filter{filter})
		},
	}
	get.Flags().String("id", "", "repository identifier (d-tag)")
	get.Flags().String("owner", "", "owner pubkey (64-char hex); omit to match any owner")

	list := &cobra.Command{
		Use:   "list",
		Short: "List repository announcements",
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, keys, owner, err := opts.resolveOwnerOrSelf(cmd, "owner")
			if err != nil {
				return err
			}
			limit, _ := cmd.Flags().GetInt("limit")
			filter := client.Filter{"kinds": []int{nostr.KindGitRepoAnnouncement}, "authors": []string{owner}}
			if cmd.Flags().Changed("limit") {
				filter["limit"] = limit
			}
			return opts.queryResolved(cmd.Context(), resolved, keys, []client.Filter{filter})
		},
	}
	list.Flags().String("owner", "", "owner pubkey (64-char hex); omit for your repos")
	list.Flags().Int("limit", 0, "maximum number of results")

	bind := &cobra.Command{
		Use:   "bind",
		Short: "Bind (or rebind) one of your repositories to a channel",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := requiredFlag(cmd, "id")
			if err != nil {
				return err
			}
			if id, err = validateRepoID(id); err != nil {
				return err
			}
			channel, err := requiredFlag(cmd, "channel")
			if err != nil {
				return err
			}
			if channel, err = validateUUIDStr(channel); err != nil {
				return err
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			existing, err := currentRepo(cmd.Context(), opts, resolved, keys, id)
			if err != nil {
				return err
			}
			tags := rebuildTags(existing.Tags, id, func(t nostr.Tag) bool { return t[0] == "buzz-channel" }, nostr.Tag{"buzz-channel", channel})
			event := nostr.NewUnsignedEvent(nostr.KindGitRepoAnnouncement, keys.PublicHex(), existing.Content, tags, existing.CreatedAt+1)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign repo announcement event", err)
			}
			return opts.submitGitEvent(cmd.Context(), resolved, keys, event)
		},
	}
	bind.Flags().String("id", "", "repository identifier (d-tag)")
	bind.Flags().String("channel", "", "channel UUID to bind; replaces any existing binding")

	cmd.AddCommand(create, get, list, bind, reposProtectCommand(opts))
	return cmd
}

// currentRepo fetches the caller's own live kind:30617 head for repoID.
func currentRepo(ctx context.Context, opts *rootOptions, resolved config.Resolved, keys *nostr.KeyPair, repoID string) (*gitEventSnapshot, error) {
	filter := client.Filter{"kinds": []int{nostr.KindGitRepoAnnouncement}, "authors": []string{keys.PublicHex()}, "#d": []string{repoID}, "limit": 1}
	found, err := fetchLatestGitEvent(ctx, opts, resolved, keys, filter)
	if err != nil {
		return nil, err
	}
	if found == nil {
		return nil, ExitError{Code: ExitInput, Message: fmt.Sprintf("repository %q was not found for the current identity", repoID)}
	}
	return found, nil
}

func reposProtectCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "protect", Short: "Manage branch and tag protection rules on one of your repositories"}

	list := &cobra.Command{
		Use:   "list",
		Short: "List the repository's protection rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := requiredFlag(cmd, "id")
			if err != nil {
				return err
			}
			if id, err = validateRepoID(id); err != nil {
				return err
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			existing, err := currentRepo(cmd.Context(), opts, resolved, keys, id)
			if err != nil {
				return err
			}
			unknown, parseErr := parseProtectionTags(existing.Tags)
			var validationError any
			if parseErr != nil {
				validationError = parseErr.Error()
				unknown = nil
			}
			protections := []map[string]any{}
			for _, tag := range existing.Tags {
				pattern, ok := protectionPattern(tag)
				if !ok {
					continue
				}
				rules := []string{}
				if len(tag) > 2 {
					rules = tag[2:]
				}
				protections = append(protections, map[string]any{"ref": pattern, "rules": rules})
			}
			if unknown == nil {
				unknown = []string{}
			}
			return opts.writeJSON(map[string]any{
				"repo_id":          id,
				"protections":      protections,
				"unknown_rules":    unknown,
				"validation_error": validationError,
			})
		},
	}
	list.Flags().String("id", "", "repository identifier (d-tag)")

	set := &cobra.Command{
		Use:   "set",
		Short: "Create or replace the rule for an exact ref pattern",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := requiredFlag(cmd, "id")
			if err != nil {
				return err
			}
			if id, err = validateRepoID(id); err != nil {
				return err
			}
			refPattern, err := requiredFlag(cmd, "ref")
			if err != nil {
				return err
			}
			push, _ := cmd.Flags().GetString("push")
			if push != "" && !validPushRoles[push] {
				return inputError("--push must be one of: owner, admin, member")
			}
			noForcePush, _ := cmd.Flags().GetBool("no-force-push")
			noDelete, _ := cmd.Flags().GetBool("no-delete")
			requirePatch, _ := cmd.Flags().GetBool("require-patch")
			tag, err := buildProtectionTag(refPattern, push, noForcePush, noDelete, requirePatch)
			if err != nil {
				return err
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			existing, err := currentRepo(cmd.Context(), opts, resolved, keys, id)
			if err != nil {
				return err
			}
			tags := rebuildTags(existing.Tags, id, func(t nostr.Tag) bool {
				pattern, ok := protectionPattern(t)
				return ok && pattern == refPattern
			}, tag)
			if _, err := parseProtectionTags(tags); err != nil {
				return otherWrap("repository contains invalid protection rules; refusing update", err)
			}
			event := nostr.NewUnsignedEvent(nostr.KindGitRepoAnnouncement, keys.PublicHex(), existing.Content, tags, existing.CreatedAt+1)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign repo announcement event", err)
			}
			return opts.submitGitEvent(cmd.Context(), resolved, keys, event)
		},
	}
	set.Flags().String("id", "", "repository identifier (d-tag)")
	set.Flags().String("ref", "", "full ref pattern, such as refs/heads/main or refs/heads/*")
	set.Flags().String("push", "", "minimum role allowed to push: owner, admin, or member")
	set.Flags().Bool("no-force-push", false, "reject non-fast-forward updates")
	set.Flags().Bool("no-delete", false, "reject deletion of matching refs")
	set.Flags().Bool("require-patch", false, "require the NIP-34 patch workflow instead of direct pushes")

	remove := &cobra.Command{
		Use:   "remove",
		Short: "Remove every protection rule for an exact ref pattern",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := requiredFlag(cmd, "id")
			if err != nil {
				return err
			}
			if id, err = validateRepoID(id); err != nil {
				return err
			}
			refPattern, err := requiredFlag(cmd, "ref")
			if err != nil {
				return err
			}
			if err := validateRefPattern(refPattern); err != nil {
				return inputError("invalid ref pattern: " + err.Error())
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			existing, err := currentRepo(cmd.Context(), opts, resolved, keys, id)
			if err != nil {
				return err
			}
			found := false
			for _, tag := range existing.Tags {
				if pattern, ok := protectionPattern(tag); ok && pattern == refPattern {
					found = true
					break
				}
			}
			if !found {
				return ExitError{Code: ExitInput, Message: fmt.Sprintf("repository %q has no protection rule for %q", id, refPattern)}
			}
			tags := rebuildTags(existing.Tags, id, func(t nostr.Tag) bool {
				pattern, ok := protectionPattern(t)
				return ok && pattern == refPattern
			})
			event := nostr.NewUnsignedEvent(nostr.KindGitRepoAnnouncement, keys.PublicHex(), existing.Content, tags, existing.CreatedAt+1)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign repo announcement event", err)
			}
			return opts.submitGitEvent(cmd.Context(), resolved, keys, event)
		},
	}
	remove.Flags().String("id", "", "repository identifier (d-tag)")
	remove.Flags().String("ref", "", "full ref pattern to remove")

	cmd.AddCommand(list, set, remove)
	return cmd
}

func hasHTTPPrefix(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func hasWSPrefix(s string) bool {
	return strings.HasPrefix(s, "ws://") || strings.HasPrefix(s, "wss://")
}
