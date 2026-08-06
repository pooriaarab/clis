package nostr

import (
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcutil/bech32"
)

func TestParseNpubRoundTrip(t *testing.T) {
	pubHex := testPubKey(t, "npub round trip")
	raw, err := hex.DecodeString(pubHex)
	if err != nil {
		t.Fatalf("decode test pubkey: %v", err)
	}
	data, err := bech32.ConvertBits(raw, 8, 5, true)
	if err != nil {
		t.Fatalf("ConvertBits: %v", err)
	}
	npub, err := bech32.Encode("npub", data)
	if err != nil {
		t.Fatalf("Encode npub: %v", err)
	}

	got, err := ParseNpub(npub)
	if err != nil {
		t.Fatalf("ParseNpub() error = %v", err)
	}
	if got != pubHex {
		t.Fatalf("ParseNpub() = %s, want %s", got, pubHex)
	}
}

func TestParseNpubRejectsWrongHRP(t *testing.T) {
	nsec, err := EncodeNsec(mustHex(t, testSecret("npub wrong hrp")))
	if err != nil {
		t.Fatalf("EncodeNsec: %v", err)
	}
	if _, err := ParseNpub(nsec); err == nil {
		t.Fatal("ParseNpub() on an nsec value should have failed")
	}
}

func TestParseNpubRejectsGarbage(t *testing.T) {
	if _, err := ParseNpub("not-a-bech32-string"); err == nil {
		t.Fatal("ParseNpub() on garbage input should have failed")
	}
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	raw, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("decode hex: %v", err)
	}
	return raw
}
