package types

import (
	"strings"
	"testing"

	"buzz-cli/internal/nostr"
)

func TestBuildArchiveIdentityRequestTags(t *testing.T) {
	owner := strings.Repeat("a", 64)
	target := strings.Repeat("B", 64)
	authTag := nostr.Tag{"auth", owner, "", strings.Repeat("c", 128)}

	event, err := BuildArchiveIdentityRequest(owner, target, authTag, 123)
	if err != nil {
		t.Fatalf("BuildArchiveIdentityRequest() error = %v", err)
	}
	if event.Kind != nostr.KindIAArchiveRequest {
		t.Fatalf("kind = %d, want %d", event.Kind, nostr.KindIAArchiveRequest)
	}
	if event.PubKey != owner {
		t.Fatalf("pubkey = %q, want %q", event.PubKey, owner)
	}
	if len(event.Tags) != 4 {
		t.Fatalf("tags len = %d, want 4: %#v", len(event.Tags), event.Tags)
	}
	wantTags := nostr.Tags{
		{"-"},
		{"p", strings.ToLower(target)},
		{"reason", "retired"},
		authTag,
	}
	for i := range wantTags {
		if strings.Join(event.Tags[i], "\x00") != strings.Join(wantTags[i], "\x00") {
			t.Fatalf("tag %d = %#v, want %#v", i, event.Tags[i], wantTags[i])
		}
	}
}

func TestBuildArchiveIdentityRequestValidatesTarget(t *testing.T) {
	if _, err := BuildArchiveIdentityRequest(strings.Repeat("a", 64), "not-hex", nil, 0); err == nil {
		t.Fatal("expected invalid target error")
	}
}
