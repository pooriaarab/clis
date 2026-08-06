package cli

import (
	"testing"

	"buzz-cli/internal/nostr"
)

func TestRelayServerTag(t *testing.T) {
	cases := []struct{ url, want string }{
		{"ws://localhost:3000", "localhost:3000"},
		{"wss://relay.example:8443", "relay.example:8443"},
		{"wss://relay.example:443", "relay.example"},
		{"ws://relay.example:80", "relay.example"},
		{"wss://relay.example", "relay.example"},
		{"wss://Relay.Example.", "relay.example"},
	}
	for _, c := range cases {
		if got := relayServerTag(c.url); got != c.want {
			t.Errorf("relayServerTag(%q) = %q, want %q", c.url, got, c.want)
		}
	}
}

func TestIsSafeMediaPathSegment(t *testing.T) {
	hash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	cases := []struct {
		in   string
		want bool
	}{
		{hash, true},
		{hash + ".jpg", true},
		{hash + ".thumb.jpg", true},
		{"abc123.jpg", false},
		{hash + ".JPG", false},
		{hash + ".eviltoolong", false},
		{"../evil", false},
	}
	for _, c := range cases {
		if got := isSafeMediaPathSegment(c.in); got != c.want {
			t.Errorf("isSafeMediaPathSegment(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestMediaURLFromInputSameOriginOnly(t *testing.T) {
	hash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	relay := "https://relay.example:443"

	got, err := mediaURLFromInput(relay, "https://relay.example/media/"+hash+".jpg")
	if err != nil || got != "https://relay.example/media/"+hash+".jpg" {
		t.Fatalf("got %q, %v", got, err)
	}

	if _, err := mediaURLFromInput(relay, "https://evil.example/media/"+hash+".jpg"); err == nil {
		t.Fatal("expected cross-origin rejection")
	}
	if _, err := mediaURLFromInput(relay, "http://relay.example/media/"+hash+".jpg"); err == nil {
		t.Fatal("expected scheme-mismatch rejection")
	}

	got, err = mediaURLFromInput(relay, hash+".jpg")
	if err != nil || got != relay+"/media/"+hash+".jpg" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestSignBlossomGetProducesServerScopedAuth(t *testing.T) {
	keys, err := nostr.NewKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	header, err := signBlossomGet(keys, "https://relay.example:443/media/"+"a"+"a")
	if err != nil {
		t.Fatal(err)
	}
	if header == "" || header[:6] != "Nostr " {
		t.Fatalf("unexpected header: %q", header)
	}
}
