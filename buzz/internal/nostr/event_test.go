package nostr

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
)

func TestEventIDAndSignRoundTrip(t *testing.T) {
	secret := testSecret("event id signing")
	keys, err := ParsePrivateKey(secret)
	if err != nil {
		t.Fatalf("ParsePrivateKey() error = %v", err)
	}
	targetPub := testPubKey(t, "target pubkey")

	event := Event{
		PubKey:    keys.PublicHex(),
		CreatedAt: 1234567890,
		Kind:      KindTextNote,
		Tags:      Tags{{"p", targetPub}, {"client", "buzz-cli"}},
		Content:   "hello",
	}

	canonical, err := event.CanonicalJSON()
	if err != nil {
		t.Fatalf("CanonicalJSON() error = %v", err)
	}
	wantCanonical := `[0,"` + keys.PublicHex() + `",1234567890,1,[["p","` + targetPub + `"],["client","buzz-cli"]],"hello"]`
	if string(canonical) != wantCanonical {
		t.Fatalf("canonical JSON mismatch\nwant %s\n got %s", wantCanonical, canonical)
	}

	if err := event.Sign(keys); err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	if event.ID == "" || event.Sig == "" {
		t.Fatalf("Sign() did not populate id and signature: %#v", event)
	}
	if ok, err := event.Verify(); err != nil || !ok {
		t.Fatalf("Verify() = %v, %v", ok, err)
	}

	var encoded map[string]any
	if err := json.Unmarshal(event.MustJSON(), &encoded); err != nil {
		t.Fatalf("signed event JSON did not unmarshal: %v", err)
	}
	if encoded["pubkey"] != keys.PublicHex() {
		t.Fatalf("pubkey mismatch in JSON")
	}
}

func TestBuildNIP42AuthEventShape(t *testing.T) {
	keys, err := ParsePrivateKey(testSecret("nip42 signer"))
	if err != nil {
		t.Fatalf("ParsePrivateKey() error = %v", err)
	}
	owner := testPubKey(t, "nip42 owner")
	authTag := Tag{"auth", owner, "", testSignatureHex("nip42 sig")}
	relay := "wss" + "://" + "relay" + ".invalid"

	event, err := BuildAuthEvent("challenge-value", relay, keys, authTag)
	if err != nil {
		t.Fatalf("BuildAuthEvent() error = %v", err)
	}

	if event.Kind != KindAuth {
		t.Fatalf("kind = %d, want %d", event.Kind, KindAuth)
	}
	wantTags := Tags{{"relay", relay}, {"challenge", "challenge-value"}, authTag}
	if !event.Tags.Equal(wantTags) {
		t.Fatalf("tags = %#v, want %#v", event.Tags, wantTags)
	}
	if ok, err := event.Verify(); err != nil || !ok {
		t.Fatalf("Verify() = %v, %v", ok, err)
	}
}

func TestMintAndVerifyAuthTag(t *testing.T) {
	owner, err := ParsePrivateKey(testSecret("owner"))
	if err != nil {
		t.Fatalf("owner ParsePrivateKey() error = %v", err)
	}
	agent, err := ParsePrivateKey(testSecret("agent"))
	if err != nil {
		t.Fatalf("agent ParsePrivateKey() error = %v", err)
	}

	tag, err := MintAuthTag(owner, agent.PublicHex(), "kind=9&created_at<1713957000")
	if err != nil {
		t.Fatalf("MintAuthTag() error = %v", err)
	}

	if len(tag) != 4 || tag[0] != "auth" || tag[1] != owner.PublicHex() || tag[2] != "kind=9&created_at<1713957000" {
		t.Fatalf("unexpected auth tag: %#v", tag)
	}
	if _, err := VerifyAuthTag(tag, agent.PublicHex()); err != nil {
		t.Fatalf("VerifyAuthTag() error = %v", err)
	}
}

func testSecret(label string) string {
	sum := sha256.Sum256([]byte(label))
	if sum[31] == 0 {
		sum[31] = 1
	}
	return hex.EncodeToString(sum[:])
}

func testPubKey(t *testing.T, label string) string {
	t.Helper()
	keys, err := ParsePrivateKey(testSecret(label))
	if err != nil {
		t.Fatalf("ParsePrivateKey(%q) error = %v", label, err)
	}
	return keys.PublicHex()
}

func testSignatureHex(label string) string {
	first := sha256.Sum256([]byte(label + ":1"))
	second := sha256.Sum256([]byte(label + ":2"))
	return hex.EncodeToString(append(first[:], second[:]...))
}
