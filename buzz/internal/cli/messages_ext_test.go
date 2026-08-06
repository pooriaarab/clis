package cli

import (
	"strings"
	"testing"

	"buzz-cli/internal/nostr"
)

func TestTruncateDiff(t *testing.T) {
	short := "small diff"
	got, truncated := truncateDiff(short, 100)
	if truncated || got != short {
		t.Fatalf("truncateDiff(short) = (%q, %v), want (%q, false)", got, truncated, short)
	}

	// Build a diff comfortably over a small limit with a hunk boundary so the
	// cut lands on "\n@@" per buzz-cli's truncate_diff.
	diff := "line one\n@@ hunk one @@\n" + strings.Repeat("x", 50) + "\n@@ hunk two @@\n" + strings.Repeat("y", 50)
	limit := 40
	result, truncated := truncateDiff(diff, limit)
	if !truncated {
		t.Fatalf("truncateDiff() truncated = false, want true")
	}
	if !strings.HasSuffix(result, diffTruncationNotice) {
		t.Fatalf("truncateDiff() result missing truncation notice: %q", result)
	}
	if len(result) > limit+len(diffTruncationNotice) {
		t.Fatalf("truncateDiff() result too long: %d bytes", len(result))
	}
}

func TestTruncateDiffUTF8Safe(t *testing.T) {
	// A multi-byte rune ("é", 2 bytes in UTF-8) sitting right at the cut
	// boundary must not be split — the result must still be valid UTF-8.
	diff := strings.Repeat("é", 40)
	result, truncated := truncateDiff(diff, 21)
	if !truncated {
		t.Fatalf("truncateDiff() truncated = false, want true")
	}
	if !isValidUTF8(result) {
		t.Fatalf("truncateDiff() produced invalid UTF-8: %q", result)
	}
}

func isValidUTF8(s string) bool {
	for _, r := range s {
		if r == '�' {
			return false
		}
	}
	return true
}

func TestInferLanguage(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"main.go", "go"},
		{"app/component.tsx", "typescript"},
		{"script.py", "python"},
		{"README.md", "markdown"},
		{"noextension", ""},
		{"Dockerfile", ""}, // case-sensitive: "Dockerfile" != "dockerfile"
		{"build.dockerfile", "dockerfile"},
	}
	for _, tt := range tests {
		if got := inferLanguage(tt.path); got != tt.want {
			t.Errorf("inferLanguage(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestFindRootFromTags(t *testing.T) {
	root := strings.Repeat("a", 64)
	reply := strings.Repeat("b", 64)

	tests := []struct {
		name string
		tags nostr.Tags
		want string
	}{
		{
			name: "root marker wins",
			tags: nostr.Tags{{"e", root, "", "root"}, {"e", reply, "", "reply"}},
			want: root,
		},
		{
			name: "reply-only falls back to reply as root",
			tags: nostr.Tags{{"e", reply, "", "reply"}},
			want: reply,
		},
		{
			name: "no thread markers",
			tags: nostr.Tags{{"h", "channel-id"}},
			want: "",
		},
		{
			name: "malformed event id ignored",
			tags: nostr.Tags{{"e", "not-hex", "", "root"}},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findRootFromTags(tt.tags); got != tt.want {
				t.Errorf("findRootFromTags() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMatchProfilesByName(t *testing.T) {
	events := []nostr.Event{
		{PubKey: strings.Repeat("1", 64), Content: `{"display_name":"Ada Lovelace"}`},
		{PubKey: strings.Repeat("2", 64), Content: `{"name":"ada"}`},
		{PubKey: strings.Repeat("3", 64), Content: `{"display_name":"Bob"}`},
		{PubKey: strings.Repeat("1", 64), Content: `{"display_name":"Ada Lovelace"}`}, // duplicate, deduped
	}

	matches := matchProfilesByName(events, "ada lovelace")
	if len(matches) != 1 || matches[0][0] != strings.Repeat("1", 64) {
		t.Fatalf("matchProfilesByName(ada lovelace) = %v, want single match on pubkey 1", matches)
	}

	matches = matchProfilesByName(events, "ada")
	if len(matches) != 1 || matches[0][0] != strings.Repeat("2", 64) {
		t.Fatalf("matchProfilesByName(ada) = %v, want single match on pubkey 2", matches)
	}

	matches = matchProfilesByName(events, "nobody")
	if len(matches) != 0 {
		t.Fatalf("matchProfilesByName(nobody) = %v, want no matches", matches)
	}
}
