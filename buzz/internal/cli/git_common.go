package cli

// Shared helpers for the NIP-34 git-collaboration groups (repos, projects,
// patches, issues, pr). Kinds/tags/validation mirror the Rust oracle at
// /Users/parab/code/buzz/crates/buzz-sdk/src/builders.rs,
// /Users/parab/code/buzz/crates/buzz-core/src/git_perms.rs, and the CLI
// commands under /Users/parab/code/buzz/crates/buzz-cli/src/commands/.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"buzz-cli/internal/client"
	"buzz-cli/internal/config"
	"buzz-cli/internal/nostr"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

const (
	maxProtectionRules     = 50
	maxRefPatternLen       = 256
	maxWildcardsPerPattern = 3
	projectDMaxLen         = 1024
	projectNameMax         = 256
	projectDescriptionMax  = 2048
	projectChannelMax      = 256
	projectVisibilityMax   = 256
	projectMemberCap       = 64
)

// ── validation ───────────────────────────────────────────────────────────────

// validateRepoID mirrors Rust validate_repo_id: [a-zA-Z0-9._-]{1,64}, no
// leading dot, no "..".
func validateRepoID(id string) (string, error) {
	if id == "" || len(id) > 64 {
		return "", inputError(fmt.Sprintf("repo ID must be 1-64 characters (got %d)", len(id)))
	}
	if strings.HasPrefix(id, ".") {
		return "", inputError("repo ID must not start with '.'")
	}
	if strings.Contains(id, "..") {
		return "", inputError("repo ID must not contain '..'")
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			continue
		}
		return "", inputError(fmt.Sprintf("repo ID contains invalid characters (allowed: a-z A-Z 0-9 . _ -): %s", id))
	}
	return id, nil
}

// validateCommitHex mirrors check_commit_hex: a full 40-char (SHA-1) or
// 64-char (SHA-256) hex commit id.
func validateCommitHex(raw, field string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if (len(value) != 40 && len(value) != 64) || !isLowerHex(value) {
		return "", inputError(fmt.Sprintf("%s must be a full 40-character (SHA-1) or 64-character (SHA-256) hex commit id (got %q)", field, raw))
	}
	return value, nil
}

// maxContentBytes/maxDiffBytes mirror buzz-cli's validate::MAX_CONTENT_BYTES
// (64KiB, markdown bodies) and MAX_DIFF_BYTES (60KiB, git-patch content).
const (
	maxContentBytes = 65536
	maxDiffBytes    = 61440
)

func validateContentSize(content string, max int, label string) error {
	if len(content) > max {
		return inputError(fmt.Sprintf("%s exceeds maximum size (%d > %d bytes)", label, len(content), max))
	}
	return nil
}

func validateUUIDStr(raw string) (string, error) {
	if _, err := uuid.Parse(raw); err != nil {
		return "", inputError(fmt.Sprintf("invalid UUID: %s", raw))
	}
	return raw, nil
}

// aTagValue renders the kind:30617 coordinate clients use to address a repo
// announcement: 30617:<owner>:<id>.
func aTagValue(owner, repoID string) string {
	return fmt.Sprintf("%d:%s:%s", nostr.KindGitRepoAnnouncement, owner, repoID)
}

// isBareRepoID matches Rust's is_bare_repo_id: guaranteed collision-free with
// a full 30617:<owner>:<d> coordinate since it never contains a colon.
func isBareRepoID(s string) bool {
	if s == "" || len(s) > 64 {
		return false
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

// parseFullRepoCoord validates a full "30617:<owner-hex>:<repo-d>" coordinate
// per ProjectMemberCoord::parse_full: exactly 3 segments after splitting on
// the first two colons, literal kind "30617", lowercase 64-hex owner,
// non-empty repo-d.
func parseFullRepoCoord(coord string) (string, error) {
	parts := strings.SplitN(coord, ":", 3)
	if len(parts) != 3 {
		return "", inputError(fmt.Sprintf("invalid repo coordinate: member coordinate must start with '30617:' (got %q)", coord))
	}
	kindPart, ownerPart, rest := parts[0], parts[1], parts[2]
	if kindPart != "30617" {
		return "", inputError(fmt.Sprintf("invalid repo coordinate: member coordinate must start with '30617:' (got kind %q)", kindPart))
	}
	if len(ownerPart) != 64 || !isLowerHex(ownerPart) {
		return "", inputError(fmt.Sprintf("invalid repo coordinate: member owner must be a 64-character lowercase hex pubkey (got %q)", ownerPart))
	}
	if rest == "" {
		return "", inputError("invalid repo coordinate: member coordinate repo-d must not be empty")
	}
	return fmt.Sprintf("30617:%s:%s", ownerPart, rest), nil
}

func isLowerHex(s string) bool {
	for _, r := range s {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') {
			continue
		}
		return false
	}
	return true
}

// expandRepoCoord expands a `--repo` argument (bare Buzz repo id, or full
// 30617:<owner>:<d> coordinate) into a full coordinate string.
func expandRepoCoord(raw, callerPubkey string) (string, error) {
	if isBareRepoID(raw) {
		return fmt.Sprintf("30617:%s:%s", callerPubkey, raw), nil
	}
	return parseFullRepoCoord(raw)
}

// validateRefPattern mirrors RefPattern::parse's grammar: non-empty, <=256
// chars, must start with "refs/", segments are literal [a-zA-Z0-9._-]+, "*"
// (one segment), or "**" (last segment only); at most 3 wildcard segments.
func validateRefPattern(pattern string) error {
	if pattern == "" {
		return inputError("pattern is empty")
	}
	if len(pattern) > maxRefPatternLen {
		return inputError(fmt.Sprintf("pattern exceeds %d chars", maxRefPatternLen))
	}
	if !strings.HasPrefix(pattern, "refs/") {
		return inputError("pattern must start with 'refs/'")
	}
	parts := strings.Split(pattern, "/")
	wildcards := 0
	for i, part := range parts {
		switch {
		case part == "**":
			if i != len(parts)-1 {
				return inputError("invalid segment: \"** must be the last segment\"")
			}
			wildcards++
		case part == "*":
			wildcards++
		case part == "":
			return inputError("invalid segment: \"\"")
		case strings.ContainsAny(part, "*?[]"):
			return inputError(fmt.Sprintf("invalid segment: %q", part))
		default:
			for _, r := range part {
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
					continue
				}
				return inputError(fmt.Sprintf("invalid segment: %q", part))
			}
		}
		if wildcards > maxWildcardsPerPattern {
			return inputError(fmt.Sprintf("pattern exceeds %d wildcards", maxWildcardsPerPattern))
		}
	}
	return nil
}

// ── buzz-protect tags (repos protect) ───────────────────────────────────────

var validPushRoles = map[string]bool{"owner": true, "admin": true, "member": true}

// buildProtectionTag builds a ["buzz-protect", pattern, ...rules] tag,
// validating the pattern and requiring at least one rule.
func buildProtectionTag(pattern, pushRole string, noForcePush, noDelete, requirePatch bool) (nostr.Tag, error) {
	if err := validateRefPattern(pattern); err != nil {
		return nil, err
	}
	values := []string{pattern}
	if pushRole != "" {
		values = append(values, "push:"+pushRole)
	}
	if noForcePush {
		values = append(values, "no-force-push")
	}
	if noDelete {
		values = append(values, "no-delete")
	}
	if requirePatch {
		values = append(values, "require-patch")
	}
	if _, err := parseProtectionRuleValues(values); err != nil {
		return nil, inputError("invalid protection rule: " + err.Error())
	}
	tag := nostr.Tag{"buzz-protect"}
	return append(tag, values...), nil
}

// parseProtectionRuleValues validates a single buzz-protect tag's values
// (pattern + rules, "buzz-protect" already stripped) and returns any
// unknown rule strings (forward-compat, not an error).
func parseProtectionRuleValues(values []string) ([]string, error) {
	if len(values) < 2 {
		return nil, fmt.Errorf("buzz-protect tag needs pattern + at least one rule")
	}
	if err := validateRefPattern(values[0]); err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}
	var unknown []string
	for _, rule := range values[1:] {
		switch {
		case strings.HasPrefix(rule, "push:"):
			role := strings.TrimPrefix(rule, "push:")
			if !validPushRoles[role] {
				return nil, fmt.Errorf("invalid role in push rule: %q", role)
			}
		case rule == "no-force-push", rule == "no-delete", rule == "require-patch":
			// known rule
		default:
			unknown = append(unknown, rule)
		}
	}
	return unknown, nil
}

// parseProtectionTags parses every buzz-protect tag in an event's tag list,
// enforcing the per-repo rule count limit. Returns unknown rule strings
// collected across all tags.
func parseProtectionTags(tags nostr.Tags) ([]string, error) {
	var unknown []string
	count := 0
	for _, tag := range tags {
		if len(tag) == 0 || tag[0] != "buzz-protect" {
			continue
		}
		if count >= maxProtectionRules {
			return nil, fmt.Errorf("exceeds max %d rules per repo", maxProtectionRules)
		}
		count++
		unknowns, err := parseProtectionRuleValues(tag[1:])
		if err != nil {
			return nil, err
		}
		unknown = append(unknown, unknowns...)
	}
	return unknown, nil
}

func protectionPattern(tag nostr.Tag) (string, bool) {
	if len(tag) < 2 || tag[0] != "buzz-protect" {
		return "", false
	}
	return tag[1], true
}

// ── git status (kind 1630-1633) ─────────────────────────────────────────────

// gitStatusKind maps the CLI's status word to its kind, matching Rust's
// parse_status. "merged" and "resolved" are synonyms for kind 1631.
func gitStatusKind(word string) (int, error) {
	switch word {
	case "open":
		return nostr.KindGitStatusOpen, nil
	case "merged", "resolved":
		return nostr.KindGitStatusMerged, nil
	case "closed":
		return nostr.KindGitStatusClosed, nil
	case "draft":
		return nostr.KindGitStatusDraft, nil
	default:
		return 0, inputError(fmt.Sprintf("invalid status %q — expected one of: open, merged, resolved, closed, draft", word))
	}
}

// gitStatusKindRestricted validates word against the subcommand's own
// enumeration (patches/pr expose "merged"; issues expose "resolved" —
// clap rejects the other synonym at parse time in the Rust oracle) before
// mapping it to a kind via gitStatusKind.
func gitStatusKindRestricted(word string, allowed []string) (int, error) {
	for _, a := range allowed {
		if a == word {
			return gitStatusKind(word)
		}
	}
	return 0, inputError(fmt.Sprintf("invalid status %q — expected one of: %s", word, strings.Join(allowed, ", ")))
}

var (
	patchOrPrStatusWords = []string{"open", "merged", "closed", "draft"}
	issueStatusWords     = []string{"open", "resolved", "closed", "draft"}
)

// ── applied-patch refs (`--q`) ───────────────────────────────────────────────

// parseAppliedPatchRef parses "<id>", "<id>:<relay-url>", or
// "<id>:<relay-url>:<pubkey>". The relay-url segment may itself contain ':'
// (wss://host:port), so only the trailing segment is checked for a 64-hex
// pubkey shape.
func parseAppliedPatchRef(spec string) (id, relay, pubkey string, err error) {
	idx := strings.IndexByte(spec, ':')
	if idx < 0 {
		id, err = validateHex64(spec)
		return id, "", "", err
	}
	id, err = validateHex64(spec[:idx])
	if err != nil {
		return "", "", "", err
	}
	rest := spec[idx+1:]
	if last := strings.LastIndexByte(rest, ':'); last >= 0 {
		candidate := rest[last+1:]
		if len(candidate) == 64 && isLowerHexOrUpper(candidate) {
			pk, perr := validateHex64(candidate)
			if perr == nil {
				return id, rest[:last], pk, nil
			}
		}
	}
	return id, rest, "", nil
}

func isLowerHexOrUpper(s string) bool {
	for _, r := range s {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			continue
		}
		return false
	}
	return true
}

// ── committer identity (`--committer`) ──────────────────────────────────────

func parseCommitter(spec string) (name, email, ts, tz string, err error) {
	parts := strings.Split(spec, "|")
	if len(parts) != 4 {
		return "", "", "", "", inputError("--committer must be 'name|email|timestamp|tz-offset-minutes'")
	}
	return parts[0], parts[1], parts[2], parts[3], nil
}

// ── NIP-MP project envelope (kind:30621) ────────────────────────────────────

var projectSingletonFields = map[string]int{
	"name":            projectNameMax,
	"description":     projectDescriptionMax,
	"buzz-channel":    projectChannelMax,
	"buzz-visibility": projectVisibilityMax,
}

// validateProjectEnvelope enforces the 8 NIP-MP ingest rules Layer A checks
// server-side, so a malformed local build fails fast as a usage error
// instead of round-tripping to the relay.
func validateProjectEnvelope(tags nostr.Tags) error {
	dCount := 0
	var dValue string
	aCount := 0
	seenCoords := map[string]struct{}{}
	singletonCount := map[string]int{}

	for _, tag := range tags {
		if len(tag) == 0 {
			continue
		}
		switch tag[0] {
		case "d":
			dCount++
			if len(tag) > 1 {
				dValue = tag[1]
			}
		case "a":
			aCount++
		}
	}
	if dCount != 1 {
		return inputError("project must have exactly one 'd' tag (rule: d-cardinality)")
	}
	if dValue == "" {
		return inputError("project 'd' tag must not be empty (rule: d-empty)")
	}
	if len(dValue) > projectDMaxLen {
		return inputError(fmt.Sprintf("project 'd' tag exceeds %d bytes (rule: d-empty)", projectDMaxLen))
	}
	if aCount > projectMemberCap {
		return inputError(fmt.Sprintf("project exceeds member cap of %d (got %d) (rule: member-cap)", projectMemberCap, aCount))
	}

	for _, tag := range tags {
		if len(tag) == 0 {
			continue
		}
		switch tag[0] {
		case "a":
			valueLen := len(tag) - 1
			if valueLen < 1 || valueLen > 2 {
				return inputError(fmt.Sprintf("member 'a' tag must have 1 or 2 value elements (got %d) (rule: member-tag-arity)", valueLen))
			}
			coord := tag[1]
			if _, err := parseFullRepoCoord(coord); err != nil {
				return inputError(err.Error() + " (rule: member-coordinate-malformed)")
			}
			if _, ok := seenCoords[coord]; ok {
				return inputError(fmt.Sprintf("duplicate member coordinate %q (rule: member-duplicate)", coord))
			}
			seenCoords[coord] = struct{}{}
		default:
			if maxBytes, ok := projectSingletonFields[tag[0]]; ok {
				singletonCount[tag[0]]++
				if singletonCount[tag[0]] > 1 {
					return inputError(fmt.Sprintf("project must have at most one '%s' tag (rule: metadata-cardinality)", tag[0]))
				}
				value := ""
				if len(tag) > 1 {
					value = tag[1]
				}
				if len(value) > maxBytes {
					return inputError(fmt.Sprintf("'%s' tag exceeds %d bytes (rule: metadata-length)", tag[0], maxBytes))
				}
			}
		}
	}
	return nil
}

// ── entity links (buzz:// deep links, rendered as rich previews in-app) ─────

func repoLink(owner, repoID string) string {
	return fmt.Sprintf("buzz://repo?owner=%s&d=%s", owner, repoID)
}

func issueLink(eventID, owner, repoID string) string {
	return fmt.Sprintf("buzz://issue?id=%s&owner=%s&d=%s", eventID, owner, repoID)
}

func pullRequestLink(eventID, owner, repoID string) string {
	return fmt.Sprintf("buzz://pr?id=%s&owner=%s&d=%s", eventID, owner, repoID)
}

// ── stdin / file reading (--content/--patch-file/--body "-") ───────────────

// readOptionalBody mirrors read_optional_body: --body and --body-file are
// mutually exclusive; body is literal content ('-' reads stdin), body-file
// is always a path ('-' reads stdin). Neither given means an empty body.
func readOptionalBody(body, bodyFile string) (string, error) {
	switch {
	case body != "" && bodyFile != "":
		return "", inputError("--body and --body-file are mutually exclusive")
	case body != "":
		return readOrStdin(body)
	case bodyFile != "":
		return readFileOrStdin(bodyFile)
	default:
		return "", nil
	}
}

func readOrStdin(value string) (string, error) {
	if value == "-" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", otherWrap("read stdin", err)
		}
		return string(b), nil
	}
	return value, nil
}

func readFileOrStdin(value string) (string, error) {
	if value == "-" {
		return readOrStdin(value)
	}
	b, err := os.ReadFile(value)
	if err != nil {
		return "", inputWrap(fmt.Sprintf("read %q", value), err)
	}
	return string(b), nil
}

// ── relay event snapshot (read-modify-write for repos/projects) ────────────

type gitEventSnapshot struct {
	ID        string
	PubKey    string
	CreatedAt int64
	Tags      nostr.Tags
	Content   string
}

func gitEventSnapshotFromAny(ev map[string]any) gitEventSnapshot {
	return gitEventSnapshot{
		ID:        stringFromAny(ev["id"]),
		PubKey:    stringFromAny(ev["pubkey"]),
		CreatedAt: int64(uint64FromAny(ev["created_at"])),
		Tags:      tagsFromAny(ev["tags"]),
		Content:   stringFromAny(ev["content"]),
	}
}

// fetchLatestGitEvent runs filter and returns the newest matching event
// (by created_at), or nil if there were no matches.
func fetchLatestGitEvent(ctx context.Context, opts *rootOptions, resolved config.Resolved, keys *nostr.KeyPair, filter client.Filter) (*gitEventSnapshot, error) {
	raw, err := opts.fetchQuery(ctx, resolved, keys, []client.Filter{filter})
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil, nil
	}
	var events []map[string]any
	if err := json.Unmarshal(raw, &events); err != nil {
		return nil, otherWrap("parse relay response", err)
	}
	if len(events) == 0 {
		return nil, nil
	}
	snapshots := make([]gitEventSnapshot, 0, len(events))
	for _, ev := range events {
		snapshots = append(snapshots, gitEventSnapshotFromAny(ev))
	}
	sort.SliceStable(snapshots, func(i, j int) bool { return snapshots[i].CreatedAt > snapshots[j].CreatedAt })
	return &snapshots[0], nil
}

// rebuildTags drops the existing "d"/"auth" tags (and anything drop()
// flags), prepends a canonical d tag, and appends extra tags — the shared
// read-modify-write shape used by repos bind/protect and projects
// create/update/add-repo/remove-repo.
func rebuildTags(existing nostr.Tags, dTagValue string, drop func(nostr.Tag) bool, extra ...nostr.Tag) nostr.Tags {
	tags := nostr.Tags{{"d", dTagValue}}
	for _, tag := range existing {
		if len(tag) == 0 || tag[0] == "d" || tag[0] == "auth" {
			continue
		}
		if drop != nil && drop(tag) {
			continue
		}
		tags = append(tags, tag)
	}
	return append(tags, extra...)
}

// publishWithLink signs and submits a create event, then injects a
// buzz://... deep link into the relay's write response when accepted --
// matching the bundled CLI's create-command output shape.
func (opts *rootOptions) publishWithLink(ctx context.Context, resolved config.Resolved, keys *nostr.KeyPair, event nostr.Event, linkKey, link string) error {
	relayClient, err := restClientFromResolved(resolved, keys)
	if err != nil {
		return err
	}
	raw, err := relayClient.PostEvent(ctx, event)
	if err != nil {
		return ExitError{Code: ExitRelay, Message: "publish event failed", Err: err}
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil || value == nil {
		value = map[string]any{}
	}
	if accepted, _ := value["accepted"].(bool); accepted {
		value[linkKey] = link
	}
	return opts.writeJSON(value)
}

// submitGitEvent signs and submits a write, surfacing relay-reported
// duplicate/dominated writes as a write-conflict (exit 5).
func (opts *rootOptions) submitGitEvent(ctx context.Context, resolved config.Resolved, keys *nostr.KeyPair, event nostr.Event) error {
	relayClient, err := restClientFromResolved(resolved, keys)
	if err != nil {
		return err
	}
	raw, err := relayClient.PostEvent(ctx, event)
	if err != nil {
		return ExitError{Code: ExitRelay, Message: "publish event failed", Err: err}
	}
	if err := relayPublishError(raw); err != nil {
		return err
	}
	return opts.writeRawJSON(raw)
}

// resolveOwnerOrSelf reads a --<flagName> pubkey flag, defaulting to the
// resolved identity's own pubkey when omitted (requires a private key in
// that case).
func (opts *rootOptions) resolveOwnerOrSelf(cmd *cobra.Command, flagName string) (config.Resolved, *nostr.KeyPair, string, error) {
	resolved, keys, err := opts.resolveKeys(false)
	if err != nil {
		return config.Resolved{}, nil, "", err
	}
	raw, _ := cmd.Flags().GetString(flagName)
	if raw != "" {
		pubkey, err := validateHex64(raw)
		return resolved, keys, pubkey, err
	}
	if keys == nil {
		return config.Resolved{}, nil, "", inputError("private key is required when --" + flagName + " is not given")
	}
	return resolved, keys, keys.PublicHex(), nil
}

// optionalRepoCoord bundles the --repo-owner/--repo-id pairing used by
// patches/issues/pr status commands: both given, or neither.
func optionalRepoCoord(cmd *cobra.Command) (owner, id string, ok bool, err error) {
	owner, _ = cmd.Flags().GetString("repo-owner")
	id, _ = cmd.Flags().GetString("repo-id")
	if owner == "" && id == "" {
		return "", "", false, nil
	}
	if owner == "" || id == "" {
		return "", "", false, inputError("--repo-owner and --repo-id must be given together")
	}
	owner, err = validateHex64(owner)
	if err != nil {
		return "", "", false, err
	}
	id, err = validateRepoID(id)
	if err != nil {
		return "", "", false, err
	}
	return owner, id, true, nil
}

// statusBaseTags builds the tags shared by every kind:1630-1633 status
// event: the root `e` tag, recipient `p` tags (repo owner + --to), the repo
// `a` tag when --repo-owner/--repo-id are given, and the `r` euc tag.
func statusBaseTags(cmd *cobra.Command, root string) (nostr.Tags, error) {
	repoOwner, repoID, hasRepo, err := optionalRepoCoord(cmd)
	if err != nil {
		return nil, err
	}
	to, _ := cmd.Flags().GetStringArray("to")
	recipients, err := statusRecipients(repoOwner, to)
	if err != nil {
		return nil, err
	}
	tags := nostr.Tags{{"e", root, "", "root"}}
	for _, recipient := range recipients {
		tags = append(tags, nostr.Tag{"p", recipient})
	}
	if hasRepo {
		tags = append(tags, nostr.Tag{"a", aTagValue(repoOwner, repoID)})
	}
	if euc, _ := cmd.Flags().GetString("euc"); euc != "" {
		validated, err := validateCommitHex(euc, "euc")
		if err != nil {
			return nil, err
		}
		tags = append(tags, nostr.Tag{"r", validated})
	}
	return tags, nil
}

// appendMergeCommitTag appends the merge-commit + r tags shared by patches
// and pr status (issues never carries one).
func appendMergeCommitTag(tags nostr.Tags, mergeCommit string) (nostr.Tags, error) {
	if mergeCommit == "" {
		return tags, nil
	}
	validated, err := validateCommitHex(mergeCommit, "merge_commit")
	if err != nil {
		return nil, err
	}
	return append(tags, nostr.Tag{"merge-commit", validated}, nostr.Tag{"r", validated}), nil
}

func addStatusCommonFlags(cmd *cobra.Command) {
	cmd.Flags().String("status", "", "new status")
	cmd.Flags().String("repo-owner", "", "repo owner pubkey — requires --repo-id")
	cmd.Flags().String("repo-id", "", "repo identifier (d-tag) — requires --repo-owner")
	cmd.Flags().String("euc", "", "earliest-unique-commit of the repo")
	cmd.Flags().StringArray("to", nil, "additional recipient pubkey(s) for the status event (repeatable)")
}

// statusRecipients builds the `p` tag recipient list for a status change:
// the repo owner (if known) first, then --to values (deduped).
func statusRecipients(repoOwner string, to []string) ([]string, error) {
	var recipients []string
	seen := map[string]struct{}{}
	if repoOwner != "" {
		recipients = append(recipients, repoOwner)
		seen[repoOwner] = struct{}{}
	}
	for _, raw := range to {
		pk, err := validateHex64(raw)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[pk]; ok {
			continue
		}
		seen[pk] = struct{}{}
		recipients = append(recipients, pk)
	}
	return recipients, nil
}
