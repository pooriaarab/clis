package nostr

import (
	"strings"
	"testing"
)

func TestNaddrRoundTrip(t *testing.T) {
	tests := []struct {
		name       string
		kind       uint32
		identifier string
	}{
		{name: "long form", kind: KindLongForm, identifier: "release-notes"},
		{name: "emoji set", kind: KindEmojiSet, identifier: "buzz:custom-emoji"},
		{name: "hyphens and dots", kind: KindLongForm, identifier: "build.plan-v1.2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys, err := NewKeyPair()
			if err != nil {
				t.Fatalf("NewKeyPair() error = %v", err)
			}
			encoded, err := EncodeNaddr(tt.kind, keys.PublicHex(), tt.identifier)
			if err != nil {
				t.Fatalf("EncodeNaddr() error = %v", err)
			}
			if !strings.HasPrefix(encoded, "naddr1") {
				t.Fatalf("encoded naddr %q does not have naddr1 prefix", encoded)
			}

			kind, pubkey, identifier, err := DecodeNaddr(encoded)
			if err != nil {
				t.Fatalf("DecodeNaddr() error = %v", err)
			}
			if kind != tt.kind {
				t.Fatalf("kind = %d, want %d", kind, tt.kind)
			}
			if pubkey != keys.PublicHex() {
				t.Fatalf("pubkey = %q, want %q", pubkey, keys.PublicHex())
			}
			if identifier != tt.identifier {
				t.Fatalf("identifier = %q, want %q", identifier, tt.identifier)
			}
		})
	}
}
