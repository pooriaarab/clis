package cli

import (
	"strings"
	"testing"

	"buzz-cli/internal/nostr"
)

func TestValidateRepoID(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "valid", raw: "my-repo_1.0"},
		{name: "empty", raw: "", wantErr: true},
		{name: "too long", raw: strings.Repeat("a", 65), wantErr: true},
		{name: "leading dot", raw: ".hidden", wantErr: true},
		{name: "double dot", raw: "a..b", wantErr: true},
		{name: "invalid char", raw: "has:colon", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateRepoID(tt.raw)
			if tt.wantErr != (err != nil) {
				t.Fatalf("validateRepoID(%q) error = %v, wantErr %v", tt.raw, err, tt.wantErr)
			}
		})
	}
}

func TestValidateRefPattern(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "exact", raw: "refs/heads/main"},
		{name: "single wildcard", raw: "refs/heads/*"},
		{name: "recursive wildcard last", raw: "refs/heads/**"},
		{name: "missing prefix", raw: "heads/main", wantErr: true},
		{name: "empty", raw: "", wantErr: true},
		{name: "partial glob", raw: "refs/heads/v*", wantErr: true},
		{name: "recursive not last", raw: "refs/**/main", wantErr: true},
		{name: "too many wildcards", raw: "refs/*/*/*/*", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRefPattern(tt.raw)
			if tt.wantErr != (err != nil) {
				t.Fatalf("validateRefPattern(%q) error = %v, wantErr %v", tt.raw, err, tt.wantErr)
			}
		})
	}
}

func TestBuildProtectionTag(t *testing.T) {
	tag, err := buildProtectionTag("refs/heads/main", "admin", true, true, false)
	if err != nil {
		t.Fatalf("buildProtectionTag() error = %v", err)
	}
	want := nostr.Tag{"buzz-protect", "refs/heads/main", "push:admin", "no-force-push", "no-delete"}
	if !(nostr.Tags{tag}).Equal(nostr.Tags{want}) {
		t.Fatalf("buildProtectionTag() = %#v, want %#v", tag, want)
	}

	if _, err := buildProtectionTag("refs/heads/main", "", false, false, false); err == nil {
		t.Fatalf("buildProtectionTag() with no rules should error")
	}
	if _, err := buildProtectionTag("refs/heads/main", "bot", false, false, false); err == nil {
		t.Fatalf("buildProtectionTag() with invalid role should error")
	}
}

func TestParseProtectionTags(t *testing.T) {
	tags := nostr.Tags{
		{"d", "demo"},
		{"buzz-protect", "refs/heads/main", "push:admin", "future-rule"},
	}
	unknown, err := parseProtectionTags(tags)
	if err != nil {
		t.Fatalf("parseProtectionTags() error = %v", err)
	}
	if len(unknown) != 1 || unknown[0] != "future-rule" {
		t.Fatalf("parseProtectionTags() unknown = %#v", unknown)
	}

	malformed := nostr.Tags{{"buzz-protect", "refs/heads/main"}}
	if _, err := parseProtectionTags(malformed); err == nil {
		t.Fatalf("parseProtectionTags() with pattern-only tag should error")
	}

	var many nostr.Tags
	for i := 0; i < 51; i++ {
		many = append(many, nostr.Tag{"buzz-protect", "refs/heads/main", "no-delete"})
	}
	if _, err := parseProtectionTags(many); err == nil {
		t.Fatalf("parseProtectionTags() should reject more than 50 rules")
	}
}

func TestExpandRepoCoord(t *testing.T) {
	owner := strings.Repeat("a", 64)
	other := strings.Repeat("b", 64)

	coord, err := expandRepoCoord("my-repo", owner)
	if err != nil || coord != "30617:"+owner+":my-repo" {
		t.Fatalf("expandRepoCoord(bare) = %q, %v", coord, err)
	}

	full := "30617:" + other + ":infra"
	coord, err = expandRepoCoord(full, owner)
	if err != nil || coord != full {
		t.Fatalf("expandRepoCoord(full) = %q, %v", coord, err)
	}

	if _, err := expandRepoCoord("nope:bad", owner); err == nil {
		t.Fatalf("expandRepoCoord() with malformed coordinate should error")
	}
	upper := "30617:" + strings.ToUpper(other) + ":infra"
	if _, err := expandRepoCoord(upper, owner); err == nil {
		t.Fatalf("expandRepoCoord() should reject uppercase owner hex")
	}
}

func TestValidateProjectEnvelope(t *testing.T) {
	owner := strings.Repeat("a", 64)
	valid := nostr.Tags{{"d", "platform"}, {"a", "30617:" + owner + ":buzz"}}
	if err := validateProjectEnvelope(valid); err != nil {
		t.Fatalf("validateProjectEnvelope(valid) error = %v", err)
	}

	noD := nostr.Tags{{"a", "30617:" + owner + ":buzz"}}
	if err := validateProjectEnvelope(noD); err == nil {
		t.Fatalf("validateProjectEnvelope() should require exactly one d tag")
	}

	dup := nostr.Tags{
		{"d", "platform"},
		{"a", "30617:" + owner + ":buzz"},
		{"a", "30617:" + owner + ":buzz"},
	}
	if err := validateProjectEnvelope(dup); err == nil {
		t.Fatalf("validateProjectEnvelope() should reject duplicate member coordinates")
	}

	twoNames := nostr.Tags{{"d", "platform"}, {"name", "a"}, {"name", "b"}}
	if err := validateProjectEnvelope(twoNames); err == nil {
		t.Fatalf("validateProjectEnvelope() should reject a second 'name' tag")
	}

	var wide nostr.Tags = nostr.Tags{{"d", "wide"}}
	for i := 0; i < 65; i++ {
		wide = append(wide, nostr.Tag{"a", "30617:" + owner + ":repo-" + strings.Repeat("x", 1) + string(rune('a'+i%26))})
	}
	if err := validateProjectEnvelope(wide); err == nil {
		t.Fatalf("validateProjectEnvelope() should enforce the 64-member cap")
	}
}

func TestGitStatusKindRestricted(t *testing.T) {
	if _, err := gitStatusKindRestricted("resolved", patchOrPrStatusWords); err == nil {
		t.Fatalf("patch/pr status should reject 'resolved'")
	}
	if _, err := gitStatusKindRestricted("merged", issueStatusWords); err == nil {
		t.Fatalf("issue status should reject 'merged'")
	}
	kind, err := gitStatusKindRestricted("merged", patchOrPrStatusWords)
	if err != nil || kind != nostr.KindGitStatusMerged {
		t.Fatalf("gitStatusKindRestricted(merged) = %d, %v", kind, err)
	}
	kind, err = gitStatusKindRestricted("resolved", issueStatusWords)
	if err != nil || kind != nostr.KindGitStatusMerged {
		t.Fatalf("gitStatusKindRestricted(resolved) = %d, %v", kind, err)
	}
}

func TestParseAppliedPatchRef(t *testing.T) {
	id64 := strings.Repeat("c", 64)
	pk64 := strings.Repeat("d", 64)

	gotID, relay, pubkey, err := parseAppliedPatchRef(id64)
	if err != nil || gotID != id64 || relay != "" || pubkey != "" {
		t.Fatalf("parseAppliedPatchRef(id) = %q,%q,%q,%v", gotID, relay, pubkey, err)
	}

	spec := id64 + ":wss://relay.example.com:4443:" + pk64
	gotID, relay, pubkey, err = parseAppliedPatchRef(spec)
	if err != nil || gotID != id64 || relay != "wss://relay.example.com:4443" || pubkey != pk64 {
		t.Fatalf("parseAppliedPatchRef(id:relay:pubkey) = %q,%q,%q,%v", gotID, relay, pubkey, err)
	}

	spec = id64 + ":wss://relay.example.com"
	gotID, relay, pubkey, err = parseAppliedPatchRef(spec)
	if err != nil || gotID != id64 || relay != "wss://relay.example.com" || pubkey != "" {
		t.Fatalf("parseAppliedPatchRef(id:relay) = %q,%q,%q,%v", gotID, relay, pubkey, err)
	}
}

func TestParseCommitter(t *testing.T) {
	name, email, ts, tz, err := parseCommitter("Jane Doe|jane@example.com|1700000000|-480")
	if err != nil || name != "Jane Doe" || email != "jane@example.com" || ts != "1700000000" || tz != "-480" {
		t.Fatalf("parseCommitter() = %q,%q,%q,%q,%v", name, email, ts, tz, err)
	}
	if _, _, _, _, err := parseCommitter("a|b|c"); err == nil {
		t.Fatalf("parseCommitter() with wrong field count should error")
	}
}

func TestReadOptionalBody(t *testing.T) {
	if _, err := readOptionalBody("x", "file.md"); err == nil {
		t.Fatalf("readOptionalBody() with both body and body-file should error")
	}
	got, err := readOptionalBody("", "")
	if err != nil || got != "" {
		t.Fatalf("readOptionalBody() default = %q, %v", got, err)
	}
	got, err = readOptionalBody("hello", "")
	if err != nil || got != "hello" {
		t.Fatalf("readOptionalBody(body) = %q, %v", got, err)
	}
}

func TestEntityLinks(t *testing.T) {
	owner := strings.Repeat("a", 64)
	eventID := strings.Repeat("b", 64)
	if got := repoLink(owner, "buzz-world"); got != "buzz://repo?owner="+owner+"&d=buzz-world" {
		t.Fatalf("repoLink() = %q", got)
	}
	if got := issueLink(eventID, owner, "buzz-world"); got != "buzz://issue?id="+eventID+"&owner="+owner+"&d=buzz-world" {
		t.Fatalf("issueLink() = %q", got)
	}
	if got := pullRequestLink(eventID, owner, "buzz-world"); got != "buzz://pr?id="+eventID+"&owner="+owner+"&d=buzz-world" {
		t.Fatalf("pullRequestLink() = %q", got)
	}
}
