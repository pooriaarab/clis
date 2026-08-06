package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"buzz-cli/internal/client"
	"buzz-cli/internal/nostr"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// diffTruncationNotice mirrors buzz-cli/src/validate.rs truncate_diff's
// TRUNCATION_NOTICE. maxContentBytes/maxDiffBytes and validateContentSize/
// readOrStdin are shared with the NIP-34 git commands (git_common.go).
const diffTruncationNotice = "\n\n[diff truncated — exceeded size limit]"

// resolveChannelForEvent looks up an existing event by id and returns its
// "h" tag (channel UUID). Used by edit/delete/vote, which take only an
// event id and must recover which channel to scope the follow-up event to
// (buzz-cli/src/commands/messages.rs resolve_channel_id).
func (opts *rootOptions) resolveChannelForEvent(ctx context.Context, eventID string) (string, error) {
	events, err := opts.queryEvents(ctx, []client.Filter{{"ids": []string{eventID}}})
	if err != nil {
		return "", err
	}
	if len(events) == 0 {
		return "", otherWrap("resolve channel", fmt.Errorf("event %s not found", eventID))
	}
	for _, tag := range events[0].Tags {
		if len(tag) >= 2 && tag[0] == "h" {
			channel := tag[1]
			if _, err := uuid.Parse(channel); err != nil {
				return "", otherWrap("resolve channel", fmt.Errorf("event h-tag is not a valid UUID: %s", channel))
			}
			return channel, nil
		}
	}
	return "", otherWrap("resolve channel", fmt.Errorf("event %s has no h-tag — cannot determine channel", eventID))
}

// findRootFromTags extracts the NIP-10 thread root from a parent event's
// tags: an explicit "root"-marked e-tag wins, otherwise a "reply"-marked
// e-tag (a direct reply's parent IS the root), otherwise "" (parent is
// itself a top-level message). Mirrors messages.rs find_root_from_tags.
func findRootFromTags(tags nostr.Tags) string {
	var root, reply string
	for _, tag := range tags {
		if len(tag) >= 4 && tag[0] == "e" && isHex64(tag[1]) {
			switch tag[3] {
			case "root":
				root = tag[1]
			case "reply":
				reply = tag[1]
			}
		}
	}
	if root != "" {
		return root
	}
	return reply
}

// resolveThreadRefTags fetches the immediate parent event and builds the
// NIP-10 e-tags for a reply to it: a direct reply to a top-level message
// emits a single ["e", root, "", "reply"] tag; a nested reply also emits
// the root as ["e", root, "", "root"]. Mirrors messages.rs resolve_thread_ref
// + buzz-sdk's thread_tags.
func (opts *rootOptions) resolveThreadRefTags(ctx context.Context, parentEventID string) (nostr.Tags, error) {
	events, err := opts.queryEvents(ctx, []client.Filter{{"ids": []string{parentEventID}, "limit": 1}})
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, otherWrap("resolve thread ref", fmt.Errorf("parent event %s not found", parentEventID))
	}
	root := findRootFromTags(events[0].Tags)
	if root == "" || root == parentEventID {
		return nostr.Tags{{"e", parentEventID, "", "reply"}}, nil
	}
	return nostr.Tags{{"e", root, "", "root"}, {"e", parentEventID, "", "reply"}}, nil
}

// truncateDiff caps diff at maxBytes, cutting on a hunk ("\n@@") or line
// boundary when possible so the truncated diff stays readable, and appends
// a notice. Mirrors buzz-cli/src/validate.rs truncate_diff exactly.
func truncateDiff(diff string, maxBytes int) (string, bool) {
	if len(diff) <= maxBytes {
		return diff, false
	}
	effectiveLimit := maxBytes - len(diffTruncationNotice)
	if effectiveLimit < 0 {
		effectiveLimit = 0
	}
	boundary := 0
	for i := range diff {
		if i > effectiveLimit {
			break
		}
		boundary = i
	}
	safePrefix := diff[:boundary]
	cutPoint := strings.LastIndex(safePrefix, "\n@@")
	if cutPoint <= 0 {
		cutPoint = strings.LastIndex(safePrefix, "\n")
		if cutPoint < 0 {
			cutPoint = boundary
		}
	}
	return diff[:cutPoint] + diffTruncationNotice, true
}

// inferLanguage maps a file extension to a language hint. Mirrors
// buzz-cli/src/validate.rs infer_language (case-sensitive on the extension,
// same as the Rust match).
func inferLanguage(filePath string) string {
	ext := filePath
	if i := strings.LastIndex(filePath, "."); i >= 0 {
		ext = filePath[i+1:]
	}
	switch ext {
	case "rs":
		return "rust"
	case "ts", "tsx":
		return "typescript"
	case "js", "jsx":
		return "javascript"
	case "py":
		return "python"
	case "go":
		return "go"
	case "java":
		return "java"
	case "rb":
		return "ruby"
	case "c", "h":
		return "c"
	case "cpp", "cc", "cxx", "hpp":
		return "cpp"
	case "cs":
		return "csharp"
	case "swift":
		return "swift"
	case "kt", "kts":
		return "kotlin"
	case "scala":
		return "scala"
	case "sh", "bash", "zsh":
		return "bash"
	case "sql":
		return "sql"
	case "html", "htm":
		return "html"
	case "css", "scss", "sass":
		return "css"
	case "json":
		return "json"
	case "yaml", "yml":
		return "yaml"
	case "toml":
		return "toml"
	case "xml":
		return "xml"
	case "md", "markdown":
		return "markdown"
	case "dockerfile":
		return "dockerfile"
	default:
		return ""
	}
}

// matchProfilesByName does an exact, case-insensitive match of a name
// against kind:0 display_name/name content, returning deduped (pubkey,
// shown-name) pairs sorted for stable output. Mirrors messages.rs
// match_profiles_by_name.
func matchProfilesByName(events []nostr.Event, name string) [][2]string {
	lower := strings.ToLower(name)
	seen := map[string]bool{}
	matches := make([][2]string, 0)
	for _, e := range events {
		var content struct {
			DisplayName string `json:"display_name"`
			Name        string `json:"name"`
		}
		if json.Unmarshal([]byte(e.Content), &content) != nil {
			continue
		}
		if strings.ToLower(content.DisplayName) != lower && strings.ToLower(content.Name) != lower {
			continue
		}
		shown := content.Name
		if content.DisplayName != "" {
			shown = content.DisplayName
		}
		key := e.PubKey + "\x00" + shown
		if seen[key] {
			continue
		}
		seen[key] = true
		matches = append(matches, [2]string{e.PubKey, shown})
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i][0] != matches[j][0] {
			return matches[i][0] < matches[j][0]
		}
		return matches[i][1] < matches[j][1]
	})
	return matches
}

// resolveAuthor resolves a `--author` value to a lowercase hex pubkey:
// 64-char hex, an npub, or a display name resolved via NIP-50 kind:0
// search (must match exactly one profile). Mirrors messages.rs
// resolve_author.
func (opts *rootOptions) resolveAuthor(ctx context.Context, author string) (string, error) {
	author = strings.TrimSpace(author)
	if isHex64(author) {
		return strings.ToLower(author), nil
	}
	if strings.HasPrefix(author, "npub1") {
		hex, err := nostr.ParseNpub(author)
		if err != nil {
			return "", inputWrap("invalid npub", err)
		}
		return hex, nil
	}
	events, err := opts.queryEvents(ctx, []client.Filter{{"kinds": []int{nostr.KindProfile}, "search": author, "limit": 100}})
	if err != nil {
		return "", err
	}
	matches := matchProfilesByName(events, author)
	switch len(matches) {
	case 0:
		return "", inputError(fmt.Sprintf("no user found with name '%s' — pass a hex pubkey or npub instead", author))
	case 1:
		return matches[0][0], nil
	default:
		shown := 5
		if len(matches) < shown {
			shown = len(matches)
		}
		listing := make([]string, 0, shown+1)
		for _, m := range matches[:shown] {
			listing = append(listing, fmt.Sprintf("%s (%s)", m[1], m[0]))
		}
		if len(matches) > shown {
			listing = append(listing, fmt.Sprintf("… and %d more", len(matches)-shown))
		}
		return "", inputError(fmt.Sprintf("name '%s' is ambiguous — matches: %s. Pass a pubkey instead", author, strings.Join(listing, ", ")))
	}
}

func messagesSearchCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Full-text search across messages",
		RunE: func(cmd *cobra.Command, args []string) error {
			query, _ := cmd.Flags().GetString("query")
			author, _ := cmd.Flags().GetString("author")
			since, _ := cmd.Flags().GetInt64("since")
			limit, _ := cmd.Flags().GetInt("limit")
			hasQuery := cmd.Flags().Changed("query")
			hasAuthor := cmd.Flags().Changed("author")
			if !hasQuery && !hasAuthor {
				return inputError("at least one of --query or --author is required")
			}
			if limit <= 0 {
				limit = 20
			}
			if limit > 100 {
				limit = 100
			}

			filter := client.Filter{
				"kinds": []int{nostr.KindChannelMessage, nostr.KindStreamMessageV2, nostr.KindForumPost, nostr.KindForumComment},
				"limit": limit,
			}
			if hasQuery {
				filter["search"] = query
			}
			if hasAuthor {
				authorHex, err := opts.resolveAuthor(cmd.Context(), author)
				if err != nil {
					return err
				}
				filter["authors"] = []string{authorHex}
			}
			if cmd.Flags().Changed("since") {
				filter["since"] = since
			}

			events, err := opts.queryEvents(cmd.Context(), []client.Filter{filter})
			if err != nil {
				return err
			}
			// The full-text path returns relevance order; a pure author/time query has
			// no relevance, so present newest-first like `messages get`.
			if !hasQuery {
				sort.SliceStable(events, func(i, j int) bool { return events[i].CreatedAt > events[j].CreatedAt })
			}
			if events == nil {
				events = []nostr.Event{}
			}
			return opts.writeJSON(events)
		},
	}
	cmd.Flags().String("query", "", "Search query string (optional when --author is given)")
	cmd.Flags().String("author", "", "Filter by author: 64-char hex pubkey, npub, or display name")
	cmd.Flags().Int64("since", 0, "Unix timestamp — return messages after this time")
	cmd.Flags().Int("limit", 20, "Maximum number of results to return")
	return cmd
}

func messagesSendDiffCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send-diff",
		Short: "Send a code diff / patch to a channel",
		RunE: func(cmd *cobra.Command, args []string) error {
			channel, err := requiredFlag(cmd, "channel")
			if err != nil {
				return err
			}
			if _, err := uuid.Parse(channel); err != nil {
				return inputWrap("invalid channel UUID", err)
			}
			diffFlag, err := requiredFlag(cmd, "diff")
			if err != nil {
				return err
			}
			repo, err := requiredFlag(cmd, "repo")
			if err != nil {
				return err
			}
			if !strings.HasPrefix(repo, "http://") && !strings.HasPrefix(repo, "https://") {
				return inputError("--repo must start with http:// or https://")
			}
			commit, err := requiredFlag(cmd, "commit")
			if err != nil {
				return err
			}
			if len(commit) < 7 || !isLowerHexOrUpper(commit) {
				return inputError("--commit must be at least 7 hex characters")
			}
			file, _ := cmd.Flags().GetString("file")
			parentCommit, _ := cmd.Flags().GetString("parent-commit")
			if parentCommit != "" && (len(parentCommit) < 7 || !isLowerHexOrUpper(parentCommit)) {
				return inputError("--parent-commit must be at least 7 hex characters")
			}
			sourceBranch, _ := cmd.Flags().GetString("source-branch")
			targetBranch, _ := cmd.Flags().GetString("target-branch")
			if (sourceBranch == "") != (targetBranch == "") {
				return inputError("--source-branch and --target-branch must both be provided or both omitted")
			}
			pr, _ := cmd.Flags().GetInt("pr")
			hasPR := cmd.Flags().Changed("pr")
			if hasPR && pr <= 0 {
				return inputError("--pr must be positive")
			}
			lang, _ := cmd.Flags().GetString("lang")
			description, _ := cmd.Flags().GetString("description")
			replyTo, _ := cmd.Flags().GetString("reply-to")
			if replyTo != "" && !isHex64(replyTo) {
				return inputError("--reply-to must be a 64-character hex string")
			}

			diffContent, err := readOrStdin(diffFlag)
			if err != nil {
				return err
			}
			truncatedDiff, wasTruncated := truncateDiff(diffContent, maxDiffBytes)

			language := lang
			if language == "" && file != "" {
				language = inferLanguage(file)
			}

			var alt string
			switch {
			case file != "" && description != "":
				alt = fmt.Sprintf("Diff: %s — %s", file, description)
			case file != "":
				alt = fmt.Sprintf("Diff: %s", file)
			default:
				alt = "Diff"
			}

			tags := nostr.Tags{{"h", channel}, {"repo", repo}, {"commit", strings.ToLower(commit)}}
			if file != "" {
				tags = append(tags, nostr.Tag{"file", file})
			}
			if parentCommit != "" {
				tags = append(tags, nostr.Tag{"parent-commit", strings.ToLower(parentCommit)})
			}
			if sourceBranch != "" && targetBranch != "" {
				tags = append(tags, nostr.Tag{"branch", sourceBranch, targetBranch})
			}
			if hasPR {
				tags = append(tags, nostr.Tag{"pr", strconv.Itoa(pr)})
			}
			if language != "" {
				tags = append(tags, nostr.Tag{"l", language})
			}
			if description != "" {
				tags = append(tags, nostr.Tag{"description", description})
			}
			if wasTruncated {
				tags = append(tags, nostr.Tag{"truncated", "true"})
			}
			tags = append(tags, nostr.Tag{"alt", alt})

			if replyTo != "" {
				threadTags, err := opts.resolveThreadRefTags(cmd.Context(), replyTo)
				if err != nil {
					return err
				}
				tags = append(tags, threadTags...)
			}

			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			event := nostr.NewUnsignedEvent(nostr.KindMessageDiff, keys.PublicHex(), truncatedDiff, tags, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign diff message event", err)
			}
			return opts.publish(cmd.Context(), resolved, keys, event)
		},
	}
	cmd.Flags().String("channel", "", "Channel UUID")
	cmd.Flags().String("diff", "", "Diff/patch content (use '-' to read from stdin)")
	cmd.Flags().String("repo", "", "Repository URL (e.g. https://github.com/org/repo)")
	cmd.Flags().String("commit", "", "Commit SHA")
	cmd.Flags().String("file", "", "Single file path within the repo")
	cmd.Flags().String("parent-commit", "", "Parent commit SHA for three-way diff context")
	cmd.Flags().String("source-branch", "", "Source branch name")
	cmd.Flags().String("target-branch", "", "Target branch name")
	cmd.Flags().Int("pr", 0, "Pull request number")
	cmd.Flags().String("lang", "", "Language hint (auto-detected from file extension if omitted)")
	cmd.Flags().String("description", "", "Human-readable description of the change")
	cmd.Flags().String("reply-to", "", "Event ID to reply to (creates a thread)")
	return cmd
}

func messagesEditCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit a previously sent message",
		RunE: func(cmd *cobra.Command, args []string) error {
			eventID, err := requiredFlag(cmd, "event")
			if err != nil {
				return err
			}
			if !isHex64(eventID) {
				return inputError("--event must be a 64-character hex string")
			}
			content, err := requiredFlag(cmd, "content")
			if err != nil {
				return err
			}
			if err := validateContentSize(content, maxContentBytes, "content"); err != nil {
				return err
			}
			channel, err := opts.resolveChannelForEvent(cmd.Context(), eventID)
			if err != nil {
				return err
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			tags := nostr.Tags{{"h", channel}, {"e", strings.ToLower(eventID)}}
			event := nostr.NewUnsignedEvent(nostr.KindMessageEdit, keys.PublicHex(), content, tags, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign edit event", err)
			}
			return opts.publish(cmd.Context(), resolved, keys, event)
		},
	}
	cmd.Flags().String("event", "", "Event ID of the message to edit (64-char hex)")
	cmd.Flags().String("content", "", "New message content")
	return cmd
}

func messagesDeleteCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a message by event ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			eventID, err := requiredFlag(cmd, "event")
			if err != nil {
				return err
			}
			if !isHex64(eventID) {
				return inputError("--event must be a 64-character hex string")
			}
			actionID, _ := cmd.Flags().GetString("action-id")
			reasonCode, _ := cmd.Flags().GetString("reason-code")
			publicReason, _ := cmd.Flags().GetString("public-reason")
			if strings.TrimSpace(actionID) != "" {
				if _, err := uuid.Parse(strings.TrimSpace(actionID)); err != nil {
					return inputWrap("invalid --action-id", err)
				}
			}
			channel, err := opts.resolveChannelForEvent(cmd.Context(), eventID)
			if err != nil {
				return err
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			tags := nostr.Tags{{"h", channel}, {"e", strings.ToLower(eventID)}}
			if strings.TrimSpace(actionID) != "" {
				tags = append(tags, nostr.Tag{"action_id", strings.TrimSpace(actionID)})
			}
			if reasonCode != "" {
				tags = append(tags, nostr.Tag{"reason_code", reasonCode})
			}
			if publicReason != "" {
				tags = append(tags, nostr.Tag{"public_reason", publicReason})
			}
			event := nostr.NewUnsignedEvent(nostr.KindNIP29DeleteEvent, keys.PublicHex(), "", tags, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign delete event", err)
			}
			return opts.publish(cmd.Context(), resolved, keys, event)
		},
	}
	cmd.Flags().String("event", "", "Event ID to delete (64-char hex)")
	cmd.Flags().String("action-id", "", "Optional moderation audit action UUID for the public tombstone")
	cmd.Flags().String("reason-code", "", "Optional machine-readable public reason code for the tombstone")
	cmd.Flags().String("public-reason", "", "Optional human-readable public reason for the tombstone")
	return cmd
}

func messagesVoteCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vote",
		Short: "Upvote or downvote a forum post",
		RunE: func(cmd *cobra.Command, args []string) error {
			eventID, err := requiredFlag(cmd, "event")
			if err != nil {
				return err
			}
			if !isHex64(eventID) {
				return inputError("--event must be a 64-character hex string")
			}
			direction, err := requiredFlag(cmd, "direction")
			if err != nil {
				return err
			}
			var content string
			switch direction {
			case "up":
				content = "+"
			case "down":
				content = "-"
			default:
				return inputError(fmt.Sprintf("--direction must be 'up' or 'down' (got: %s)", direction))
			}
			channel, err := opts.resolveChannelForEvent(cmd.Context(), eventID)
			if err != nil {
				return err
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			tags := nostr.Tags{{"h", channel}, {"e", strings.ToLower(eventID)}}
			event := nostr.NewUnsignedEvent(nostr.KindForumVote, keys.PublicHex(), content, tags, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign vote event", err)
			}
			return opts.publish(cmd.Context(), resolved, keys, event)
		},
	}
	cmd.Flags().String("event", "", "Event ID of the post to vote on (64-char hex)")
	cmd.Flags().String("direction", "", `Vote direction: "up" or "down"`)
	return cmd
}
