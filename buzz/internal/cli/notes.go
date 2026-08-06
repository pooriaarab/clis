package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"buzz-cli/internal/client"
	"buzz-cli/internal/config"
	"buzz-cli/internal/nostr"
	"github.com/spf13/cobra"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9._-]+$`)

type noteSnapshot struct {
	ID          string
	PubKey      string
	Slug        string
	Title       string
	Summary     *string
	Tags        []string
	PublishedAt *uint64
	UpdatedAt   uint64
	Content     string
}

type noteOutput struct {
	ID          string   `json:"id"`
	PubKey      string   `json:"pubkey"`
	Naddr       string   `json:"naddr"`
	Coordinate  string   `json:"coordinate"`
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Summary     *string  `json:"summary"`
	Tags        []string `json:"tags"`
	PublishedAt *uint64  `json:"published_at"`
	UpdatedAt   uint64   `json:"updated_at"`
	Content     string   `json:"content"`
}

func notesCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "notes", Short: "Long-form note commands"}

	set := &cobra.Command{
		Use:   "set",
		Short: "Publish a long-form note",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := requiredFlag(cmd, "name")
			if err != nil {
				return err
			}
			slug, err := parseSlug(name)
			if err != nil {
				return err
			}
			clearTags, _ := cmd.Flags().GetBool("clear-tags")
			if clearTags && cmd.Flags().Changed("tag") {
				return inputError("--clear-tags is mutually exclusive with --tag; pick one")
			}
			if !cmd.Flags().Changed("content") {
				return inputError("--content is required")
			}
			body, _ := cmd.Flags().GetString("content")
			allowEmpty, _ := cmd.Flags().GetBool("allow-empty")
			if body == "-" {
				body, err = readNoteBodyFromStdin(allowEmpty)
				if err != nil {
					return err
				}
			} else if body == "" && !allowEmpty {
				return inputError("refusing to publish an empty body; pass --allow-empty to confirm")
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			me := keys.PublicHex()
			prior, err := fetchLatestNote(cmd.Context(), opts, resolved, keys, client.Filter{"kinds": []int{nostr.KindLongForm}, "authors": []string{me}, "#d": []string{slug}, "limit": 1})
			if err != nil {
				return err
			}

			title, _ := cmd.Flags().GetString("title")
			if !cmd.Flags().Changed("title") {
				if prior != nil {
					title = prior.Title
				} else {
					return inputError("--title is required on first publish (NIP-23)")
				}
			}
			var summary *string
			if cmd.Flags().Changed("summary") {
				value, _ := cmd.Flags().GetString("summary")
				summary = &value
			} else if prior != nil {
				summary = prior.Summary
			}
			var topics []string
			switch {
			case clearTags:
				topics = []string{}
			case cmd.Flags().Changed("tag"):
				values, _ := cmd.Flags().GetStringSlice("tag")
				topics = compactStrings(values)
			case prior != nil:
				topics = append([]string{}, prior.Tags...)
			default:
				topics = []string{}
			}
			now := time.Now().Unix()
			publishedAt := uint64(now)
			if prior != nil && prior.PublishedAt != nil {
				publishedAt = *prior.PublishedAt
			}
			tags := nostr.Tags{{"d", slug}, {"title", title}}
			if summary != nil {
				tags = append(tags, nostr.Tag{"summary", *summary})
			}
			for _, topic := range topics {
				tags = append(tags, nostr.Tag{"t", topic})
			}
			tags = append(tags, nostr.Tag{"published_at", strconv.FormatUint(publishedAt, 10)})
			event := nostr.NewUnsignedEvent(nostr.KindLongForm, me, body, tags, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign note event", err)
			}
			relayClient, err := restClientFromResolved(resolved, keys)
			if err != nil {
				return err
			}
			raw, err := relayClient.PostEvent(cmd.Context(), event)
			if err != nil {
				return ExitError{Code: ExitRelay, Message: "publish note failed", Err: err}
			}
			if err := relayPublishError(raw); err != nil {
				return err
			}
			naddr, err := nostr.EncodeNaddr(nostr.KindLongForm, me, slug)
			if err != nil {
				return otherWrap("encode note naddr", err)
			}
			return opts.writeJSON(map[string]any{
				"event_id":   event.ID,
				"naddr":      naddr,
				"coordinate": fmt.Sprintf("%d:%s:%s", nostr.KindLongForm, me, slug),
				"slug":       slug,
				"title":      title,
			})
		},
	}
	set.Flags().String("name", "", "note slug")
	set.Flags().String("title", "", "note title")
	set.Flags().String("summary", "", "note summary")
	set.Flags().StringSlice("tag", nil, "topic tag")
	set.Flags().Bool("clear-tags", false, "clear topic tags")
	set.Flags().String("content", "", "note content, or - for stdin")
	set.Flags().Bool("allow-empty", false, "allow empty content")

	get := &cobra.Command{
		Use:   "get",
		Short: "Get a long-form note",
		RunE: func(cmd *cobra.Command, args []string) error {
			naddrRaw, _ := cmd.Flags().GetString("naddr")
			nameRaw, _ := cmd.Flags().GetString("name")
			authorRaw, _ := cmd.Flags().GetString("author")
			latest, _ := cmd.Flags().GetBool("latest")
			contentOnly, _ := cmd.Flags().GetBool("content-only")
			if (naddrRaw == "") == (nameRaw == "") {
				return inputError("exactly one of --naddr or --name is required")
			}
			if naddrRaw != "" && authorRaw != "" {
				return inputError("--author is only valid with --name")
			}
			if naddrRaw != "" && latest {
				return inputError("--latest is only valid with --name")
			}
			if authorRaw != "" && latest {
				return inputError("--author and --latest are mutually exclusive")
			}
			resolved, keys, err := opts.resolveKeys(false)
			if err != nil {
				return err
			}
			var snapshot noteSnapshot
			if naddrRaw != "" {
				kind, pubkey, identifier, err := decodeNoteAddress(naddrRaw)
				if err != nil {
					return err
				}
				if kind != nostr.KindLongForm {
					return inputError("note address kind must be 30023")
				}
				if identifier == "" {
					return inputError("note address identifier is required")
				}
				found, err := fetchLatestNote(cmd.Context(), opts, resolved, keys, client.Filter{"kinds": []int{nostr.KindLongForm}, "authors": []string{pubkey}, "#d": []string{identifier}, "limit": 1})
				if err != nil {
					return err
				}
				if found == nil {
					return ExitError{Code: ExitOther, Message: fmt.Sprintf("note not found: %s", naddrRaw)}
				}
				snapshot = *found
			} else {
				slug, err := parseSlug(nameRaw)
				if err != nil {
					return err
				}
				if authorRaw != "" {
					pubkey, err := resolveAuthor(cmd.Context(), opts, resolved, keys, authorRaw)
					if err != nil {
						return err
					}
					found, err := fetchLatestNote(cmd.Context(), opts, resolved, keys, client.Filter{"kinds": []int{nostr.KindLongForm}, "authors": []string{pubkey}, "#d": []string{slug}, "limit": 1})
					if err != nil {
						return err
					}
					if found == nil {
						return ExitError{Code: ExitOther, Message: fmt.Sprintf("note not found: %s", slug)}
					}
					snapshot = *found
				} else {
					snapshots, err := fetchNoteSnapshots(cmd.Context(), opts, resolved, keys, client.Filter{"kinds": []int{nostr.KindLongForm}, "#d": []string{slug}, "limit": 50})
					if err != nil {
						return err
					}
					if len(snapshots) == 0 {
						return ExitError{Code: ExitOther, Message: fmt.Sprintf("note not found: %s", slug)}
					}
					sortNoteSnapshots(snapshots)
					if len(snapshots) == 1 || latest {
						snapshot = snapshots[0]
					} else {
						lines := make([]string, 0, len(snapshots))
						for _, candidate := range snapshots {
							title := candidate.Title
							if title == "" {
								title = "(untitled)"
							}
							lines = append(lines, fmt.Sprintf("%s %d %s", candidate.PubKey, candidate.UpdatedAt, title))
						}
						return inputError(fmt.Sprintf("note name %q is ambiguous; pass --author <pubkey> or --latest\n%s", slug, strings.Join(lines, "\n")))
					}
				}
			}
			if contentOnly {
				if _, err := fmt.Fprint(opts.stdout(), snapshot.Content); err != nil {
					return err
				}
				if !strings.HasSuffix(snapshot.Content, "\n") {
					_, err := fmt.Fprintln(opts.stdout())
					return err
				}
				return nil
			}
			out, err := outputForNote(snapshot)
			if err != nil {
				return err
			}
			return opts.writeJSON(out)
		},
	}
	get.Flags().String("naddr", "", "naddr or coordinate")
	get.Flags().String("name", "", "note slug")
	get.Flags().String("author", "", "author pubkey, me, or display name")
	get.Flags().Bool("latest", false, "pick latest matching note")
	get.Flags().Bool("content-only", false, "print only note content")

	ls := &cobra.Command{
		Use:   "ls",
		Short: "List long-form notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			authorRaw, _ := cmd.Flags().GetString("author")
			tag, _ := cmd.Flags().GetString("tag")
			limit, _ := cmd.Flags().GetInt("limit")
			if limit <= 0 {
				return inputError("limit must be greater than zero")
			}
			if limit > 200 {
				limit = 200
			}
			if cmd.Flags().Changed("tag") && tag == "" {
				return inputError("--tag must be non-empty")
			}
			resolved, keys, err := opts.resolveKeys(false)
			if err != nil {
				return err
			}
			filter := client.Filter{"kinds": []int{nostr.KindLongForm}, "limit": limit}
			if authorRaw != "all" {
				pubkey, err := resolveAuthor(cmd.Context(), opts, resolved, keys, authorRaw)
				if err != nil {
					return err
				}
				filter["authors"] = []string{pubkey}
			}
			if tag != "" {
				filter["#t"] = []string{tag}
			}
			snapshots, err := fetchNoteSnapshots(cmd.Context(), opts, resolved, keys, filter)
			if err != nil {
				return err
			}
			sortNoteSnapshots(snapshots)
			out := make([]noteOutput, 0, len(snapshots))
			for _, snapshot := range snapshots {
				item, err := outputForNote(snapshot)
				if err != nil {
					return err
				}
				out = append(out, item)
			}
			return opts.writeJSON(out)
		},
	}
	ls.Flags().String("author", "me", "author pubkey, me, display name, or all")
	ls.Flags().String("tag", "", "topic tag")
	ls.Flags().Int("limit", 50, "max results")

	rm := &cobra.Command{
		Use:   "rm",
		Short: "Delete one of your long-form notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := requiredFlag(cmd, "name")
			if err != nil {
				return err
			}
			slug, err := parseSlug(name)
			if err != nil {
				return err
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			me := keys.PublicHex()
			found, err := fetchLatestNote(cmd.Context(), opts, resolved, keys, client.Filter{"kinds": []int{nostr.KindLongForm}, "authors": []string{me}, "#d": []string{slug}, "limit": 1})
			if err != nil {
				return err
			}
			if found == nil {
				return ExitError{Code: ExitOther, Message: fmt.Sprintf("no note %q found for you (%s); nothing to delete", slug, me)}
			}
			coordinate := fmt.Sprintf("%d:%s:%s", nostr.KindLongForm, me, slug)
			event := nostr.NewUnsignedEvent(nostr.KindDeletion, me, "", nostr.Tags{{"a", coordinate}}, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign note deletion event", err)
			}
			relayClient, err := restClientFromResolved(resolved, keys)
			if err != nil {
				return err
			}
			raw, err := relayClient.PostEvent(cmd.Context(), event)
			if err != nil {
				return ExitError{Code: ExitRelay, Message: "publish note deletion failed", Err: err}
			}
			if err := relayPublishError(raw); err != nil {
				return err
			}
			return opts.writeJSON(map[string]any{"deleted": coordinate, "deletion": event.ID})
		},
	}
	rm.Flags().String("name", "", "note slug")

	cmd.AddCommand(set, get, ls, rm)
	return cmd
}

func parseSlug(raw string) (string, error) {
	if raw == "" {
		return "", inputError("slug is required")
	}
	if len([]rune(raw)) > 80 {
		return "", inputError("slug must be 80 characters or fewer")
	}
	if !slugPattern.MatchString(raw) {
		for _, r := range raw {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
				continue
			}
			return "", inputError(fmt.Sprintf("slug contains invalid character %q", r))
		}
		return "", inputError("slug must contain only lowercase letters, digits, '.', '_' and '-'")
	}
	return raw, nil
}

func noteSnapshotFromEvent(ev map[string]any) (noteSnapshot, error) {
	kind, ok := intFromAny(ev["kind"])
	if !ok || kind != nostr.KindLongForm {
		return noteSnapshot{}, fmt.Errorf("event kind is %d, want %d", kind, nostr.KindLongForm)
	}
	tags := tagsFromAny(ev["tags"])
	snapshot := noteSnapshot{
		ID:        stringFromAny(ev["id"]),
		PubKey:    stringFromAny(ev["pubkey"]),
		Title:     "",
		Tags:      []string{},
		UpdatedAt: uint64FromAny(ev["created_at"]),
		Content:   stringFromAny(ev["content"]),
	}
	for _, tag := range tags {
		if len(tag) < 2 {
			continue
		}
		switch tag[0] {
		case "d":
			if snapshot.Slug == "" {
				snapshot.Slug = tag[1]
			}
		case "title":
			snapshot.Title = tag[1]
		case "summary":
			value := tag[1]
			snapshot.Summary = &value
		case "t":
			if tag[1] != "" {
				snapshot.Tags = append(snapshot.Tags, tag[1])
			}
		case "published_at":
			if parsed, err := strconv.ParseUint(tag[1], 10, 64); err == nil {
				snapshot.PublishedAt = &parsed
			}
		}
	}
	if snapshot.Slug == "" {
		return noteSnapshot{}, fmt.Errorf("kind:30023 event is missing the required `d` tag")
	}
	return snapshot, nil
}

func readNoteBodyFromStdin(allowEmpty bool) (string, error) {
	const maxNoteBytes = 1 << 20
	b, err := io.ReadAll(io.LimitReader(os.Stdin, maxNoteBytes+1))
	if err != nil {
		return "", inputWrap("read note body from stdin", err)
	}
	if len(b) > maxNoteBytes {
		return "", inputError("note body from stdin must be 1MiB or smaller")
	}
	if len(b) == 0 && !allowEmpty {
		return "", inputError("refusing to publish an empty body from stdin; pass --allow-empty to confirm")
	}
	return string(b), nil
}

func decodeNoteAddress(raw string) (uint32, string, string, error) {
	kind, pubkey, identifier, err := nostr.DecodeNaddr(raw)
	if err == nil {
		return kind, pubkey, identifier, nil
	}
	parts := strings.SplitN(raw, ":", 3)
	if len(parts) != 3 {
		return 0, "", "", inputError("note address must be naddr or kind:pubkey:identifier")
	}
	parsedKind, parseErr := strconv.ParseUint(parts[0], 10, 32)
	if parseErr != nil {
		return 0, "", "", inputWrap("parse note coordinate kind", parseErr)
	}
	pubkey, validateErr := validateHex64(parts[1])
	if validateErr != nil {
		return 0, "", "", validateErr
	}
	if parts[2] == "" {
		return 0, "", "", inputError("note coordinate identifier is required")
	}
	return uint32(parsedKind), pubkey, parts[2], nil
}

func resolveAuthor(ctx context.Context, opts *rootOptions, resolved config.Resolved, keys *nostr.KeyPair, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", inputError("author is required")
	}
	if ref == "me" {
		if keys == nil {
			return "", inputError("private key is required to resolve --author me")
		}
		return keys.PublicHex(), nil
	}
	if len(ref) == 64 {
		return validateHex64(ref)
	}
	raw, err := opts.fetchQuery(ctx, resolved, keys, []client.Filter{{"kinds": []int{nostr.KindProfile}, "search": ref, "limit": 100}})
	if err != nil {
		return "", err
	}
	var events []struct {
		PubKey  string `json:"pubkey"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(raw, &events); err != nil {
		return "", otherWrap("parse profile search", err)
	}
	matches := []string{}
	seen := map[string]struct{}{}
	for _, event := range events {
		var profile struct {
			DisplayName string `json:"display_name"`
			Name        string `json:"name"`
		}
		if err := json.Unmarshal([]byte(event.Content), &profile); err != nil {
			continue
		}
		if !strings.EqualFold(profile.DisplayName, ref) && !strings.EqualFold(profile.Name, ref) {
			continue
		}
		pubkey := strings.ToLower(event.PubKey)
		if _, ok := seen[pubkey]; ok {
			continue
		}
		matches = append(matches, pubkey)
		seen[pubkey] = struct{}{}
	}
	switch len(matches) {
	case 0:
		return "", inputError(fmt.Sprintf("no user found for %q", ref))
	case 1:
		return matches[0], nil
	default:
		return "", inputError(fmt.Sprintf("multiple users found for %q; disambiguate with --author <hex-pubkey>", ref))
	}
}

func fetchLatestNote(ctx context.Context, opts *rootOptions, resolved config.Resolved, keys *nostr.KeyPair, filter client.Filter) (*noteSnapshot, error) {
	snapshots, err := fetchNoteSnapshots(ctx, opts, resolved, keys, filter)
	if err != nil {
		return nil, err
	}
	if len(snapshots) == 0 {
		return nil, nil
	}
	sortNoteSnapshots(snapshots)
	return &snapshots[0], nil
}

func fetchNoteSnapshots(ctx context.Context, opts *rootOptions, resolved config.Resolved, keys *nostr.KeyPair, filter client.Filter) ([]noteSnapshot, error) {
	raw, err := opts.fetchQuery(ctx, resolved, keys, []client.Filter{filter})
	if err != nil {
		return nil, err
	}
	var events []map[string]any
	if len(strings.TrimSpace(string(raw))) == 0 {
		return []noteSnapshot{}, nil
	}
	if err := json.Unmarshal(raw, &events); err != nil {
		return nil, otherWrap("parse note query", err)
	}
	snapshots := make([]noteSnapshot, 0, len(events))
	for _, event := range events {
		snapshot, err := noteSnapshotFromEvent(event)
		if err != nil {
			return nil, otherWrap("parse note event", err)
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, nil
}

func sortNoteSnapshots(snapshots []noteSnapshot) {
	sort.SliceStable(snapshots, func(i, j int) bool {
		return snapshots[i].UpdatedAt > snapshots[j].UpdatedAt
	})
}

func outputForNote(snapshot noteSnapshot) (noteOutput, error) {
	naddr, err := nostr.EncodeNaddr(nostr.KindLongForm, snapshot.PubKey, snapshot.Slug)
	if err != nil {
		return noteOutput{}, otherWrap("encode note naddr", err)
	}
	return noteOutput{
		ID:          snapshot.ID,
		PubKey:      snapshot.PubKey,
		Naddr:       naddr,
		Coordinate:  fmt.Sprintf("%d:%s:%s", nostr.KindLongForm, snapshot.PubKey, snapshot.Slug),
		Slug:        snapshot.Slug,
		Title:       snapshot.Title,
		Summary:     snapshot.Summary,
		Tags:        snapshot.Tags,
		PublishedAt: snapshot.PublishedAt,
		UpdatedAt:   snapshot.UpdatedAt,
		Content:     snapshot.Content,
	}, nil
}

func relayPublishError(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var response struct {
		Accepted *bool  `json:"accepted"`
		Message  string `json:"message"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return otherWrap("parse relay publish response", err)
	}
	if response.Message == "duplicate" || strings.HasPrefix(response.Message, "duplicate:") {
		return ExitError{Code: ExitConflict, Message: "relay reported event as duplicate / dominated by a newer head"}
	}
	if response.Accepted != nil && !*response.Accepted {
		message := response.Message
		if message == "" {
			message = "relay rejected event"
		}
		return ExitError{Code: ExitRelay, Message: message}
	}
	return nil
}

func restClientFromResolved(resolved config.Resolved, keys *nostr.KeyPair) (*client.Client, error) {
	if resolved.RelayURL == "" {
		return nil, inputError("relay URL is required")
	}
	tag, err := nostr.ParseAuthTagJSON(resolved.AuthTag)
	if err != nil {
		return nil, inputWrap("parse auth tag", err)
	}
	return client.New(resolved.RelayURL, keys, tag), nil
}

func intFromAny(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case int64:
		return int(v), true
	case json.Number:
		n, err := v.Int64()
		return int(n), err == nil
	default:
		return 0, false
	}
}

func uint64FromAny(value any) uint64 {
	switch v := value.(type) {
	case float64:
		if v < 0 {
			return 0
		}
		return uint64(v)
	case int:
		if v < 0 {
			return 0
		}
		return uint64(v)
	case int64:
		if v < 0 {
			return 0
		}
		return uint64(v)
	case json.Number:
		n, err := strconv.ParseUint(v.String(), 10, 64)
		if err == nil {
			return n
		}
	}
	return 0
}

func stringFromAny(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func tagsFromAny(value any) nostr.Tags {
	switch tags := value.(type) {
	case nostr.Tags:
		return tags
	case []any:
		out := nostr.Tags{}
		for _, rawTag := range tags {
			rawParts, ok := rawTag.([]any)
			if !ok {
				continue
			}
			tag := nostr.Tag{}
			for _, rawPart := range rawParts {
				part, ok := rawPart.(string)
				if !ok {
					continue
				}
				tag = append(tag, part)
			}
			if len(tag) > 0 {
				out = append(out, tag)
			}
		}
		return out
	default:
		return nil
	}
}
