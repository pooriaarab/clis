package cli

import (
	"strings"
	"testing"

	"buzz-cli/internal/nostr"
)

func TestExtractOwnerAuthTag(t *testing.T) {
	owner := strings.Repeat("a", 64)
	sig := strings.Repeat("b", 128)

	tests := []struct {
		name string
		tags nostr.Tags
		want bool
	}{
		{
			name: "well-formed owner-matching tag",
			tags: nostr.Tags{{"auth", owner, "conditions", sig}},
			want: true,
		},
		{
			name: "no auth tag",
			tags: nostr.Tags{{"p", owner}},
			want: false,
		},
		{
			name: "wrong owner",
			tags: nostr.Tags{{"auth", strings.Repeat("c", 64), "", sig}},
			want: false,
		},
		{
			name: "malformed arity",
			tags: nostr.Tags{{"auth", owner, "conditions"}},
			want: false,
		},
		{
			name: "duplicate auth tags poison the whole set",
			tags: nostr.Tags{{"auth", owner, "", sig}, {"auth", owner, "", sig}},
			want: false,
		},
		{
			name: "short signature rejected",
			tags: nostr.Tags{{"auth", owner, "", sig[:64]}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractOwnerAuthTag(tt.tags, owner)
			if (got != nil) != tt.want {
				t.Fatalf("extractOwnerAuthTag() = %v, want non-nil=%v", got, tt.want)
			}
		})
	}
}

func TestValidateReasonCode(t *testing.T) {
	if err := validateReasonCode(""); err != nil {
		t.Fatalf("validateReasonCode(empty) error = %v", err)
	}
	if err := validateReasonCode("retired"); err != nil {
		t.Fatalf("validateReasonCode(retired) error = %v", err)
	}
	if err := validateReasonCode(strings.Repeat("x", 65)); err == nil {
		t.Fatal("validateReasonCode(65 bytes) should have failed")
	}
	if err := validateReasonCode("bad\x00reason"); err == nil {
		t.Fatal("validateReasonCode(control char) should have failed")
	}
}

func signedArchivedEvent(t *testing.T, keys *nostr.KeyPair, kind int, pTags []string, includeNIP70 bool) nostr.Event {
	t.Helper()
	tags := nostr.Tags{}
	if includeNIP70 {
		tags = append(tags, nostr.Tag{"-"})
	}
	for _, pk := range pTags {
		tags = append(tags, nostr.Tag{"p", pk})
	}
	event := nostr.NewUnsignedEvent(kind, keys.PublicHex(), "", tags, 1700000000)
	if err := event.Sign(keys); err != nil {
		t.Fatalf("sign archived event: %v", err)
	}
	return event
}

func TestVerifyArchivedEventValid(t *testing.T) {
	keys, err := nostr.NewKeyPair()
	if err != nil {
		t.Fatalf("NewKeyPair() error = %v", err)
	}
	pk1 := strings.Repeat("a", 64)
	pk2 := strings.Repeat("b", 64)
	event := signedArchivedEvent(t, keys, nostr.KindIAArchivedList, []string{pk1, pk2}, true)

	got, err := verifyArchivedEvent(event, keys.PublicHex())
	if err != nil {
		t.Fatalf("verifyArchivedEvent() error = %v", err)
	}
	if len(got) != 2 || got[0] != pk1 || got[1] != pk2 {
		t.Fatalf("verifyArchivedEvent() = %v, want [%s %s]", got, pk1, pk2)
	}
}

func TestVerifyArchivedEventTrustFailures(t *testing.T) {
	keys, err := nostr.NewKeyPair()
	if err != nil {
		t.Fatalf("NewKeyPair() error = %v", err)
	}

	t.Run("wrong kind", func(t *testing.T) {
		event := signedArchivedEvent(t, keys, 9999, nil, true)
		if _, err := verifyArchivedEvent(event, keys.PublicHex()); err == nil {
			t.Fatal("expected error for wrong kind")
		}
	})

	t.Run("author mismatch", func(t *testing.T) {
		event := signedArchivedEvent(t, keys, nostr.KindIAArchivedList, nil, true)
		if _, err := verifyArchivedEvent(event, strings.Repeat("f", 64)); err == nil {
			t.Fatal("expected error for author mismatch")
		}
	})

	t.Run("missing NIP-70 tag", func(t *testing.T) {
		event := signedArchivedEvent(t, keys, nostr.KindIAArchivedList, nil, false)
		if _, err := verifyArchivedEvent(event, keys.PublicHex()); err == nil {
			t.Fatal("expected error for missing NIP-70 tag")
		}
	})

	t.Run("duplicate NIP-70 tags", func(t *testing.T) {
		event := nostr.NewUnsignedEvent(nostr.KindIAArchivedList, keys.PublicHex(), "", nostr.Tags{{"-"}, {"-"}}, 1700000000)
		if err := event.Sign(keys); err != nil {
			t.Fatalf("sign: %v", err)
		}
		if _, err := verifyArchivedEvent(event, keys.PublicHex()); err == nil {
			t.Fatal("expected error for duplicate NIP-70 tags")
		}
	})

	t.Run("non-hex p tags dropped, not errored", func(t *testing.T) {
		event := signedArchivedEvent(t, keys, nostr.KindIAArchivedList, []string{"not-hex-at-all"}, true)
		got, err := verifyArchivedEvent(event, keys.PublicHex())
		if err != nil {
			t.Fatalf("verifyArchivedEvent() error = %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("verifyArchivedEvent() = %v, want empty (malformed p tag dropped)", got)
		}
	})
}
