package cli

import (
	"encoding/hex"
	"testing"

	"buzz-cli/internal/nostr"
)

// Cross-check vectors from the Rust oracle
// (buzz-core/src/engram.rs test module: SECKEY_A/SECKEY_O/K_C_HEX/D_CORE/
// D_EXAMPLE/D_NOTES) — pins our NIP-44 conversation-key derivation and HMAC
// d-tag against the reference implementation.
const (
	memTestSeckeyA = "0000000000000000000000000000000000000000000000000000000000000001"
	memTestSeckeyO = "0000000000000000000000000000000000000000000000000000000000000002"
	memTestPubkeyO = "c6047f9441ed7d6d3045406e95c07cd85c778e4b8cef3ca7abac09b95c709ee5"
	memTestKCHex   = "c41c775356fd92eadc63ff5a0dc1da211b268cbea22316767095b2871ea1412d"
	memTestDCore   = "bdc233238ffe52e272b44cc233c8f33a2bc510b08be04495b225964283be4a90"
	memTestDEx     = "72d4f9629106451505d7d341ea85bb3ebad4f654fcfd2aad100d5a35f8a85cba"
	memTestDNotes  = "31651571a312780cfdc1f0b706b682ac9f3f51a053e8dca76fe57710bae5a4d4"
)

func TestMemConversationKeyMatchesRustOracle(t *testing.T) {
	kC, err := memConversationKey(memTestSeckeyA, memTestPubkeyO)
	if err != nil {
		t.Fatalf("conversation key derivation failed: %v", err)
	}
	got := hex.EncodeToString(kC[:])
	if got != memTestKCHex {
		t.Fatalf("K_c mismatch: got %s want %s", got, memTestKCHex)
	}
}

func TestMemDTagMatchesRustOracle(t *testing.T) {
	kC, err := memConversationKey(memTestSeckeyA, memTestPubkeyO)
	if err != nil {
		t.Fatalf("conversation key derivation failed: %v", err)
	}
	cases := []struct {
		slug string
		want string
	}{
		{memCoreSlug, memTestDCore},
		{"mem/example", memTestDEx},
		{"mem/notes/2026-05-12", memTestDNotes},
	}
	for _, c := range cases {
		if got := memDTag(kC, c.slug); got != c.want {
			t.Errorf("d_tag(%q) = %s, want %s", c.slug, got, c.want)
		}
	}
}

func TestMemValidateSlug(t *testing.T) {
	valid := []string{"core", "mem/x", "mem/x-y_z", "mem/0", "mem/notes/2026-05-12", "mem/a/b/c"}
	for _, s := range valid {
		if err := memValidateSlug(s); err != nil {
			t.Errorf("expected %q valid, got %v", s, err)
		}
	}
	invalid := []string{"", "notmem", "mem/", "mem//x", "mem/X"}
	for _, s := range invalid {
		if err := memValidateSlug(s); err == nil {
			t.Errorf("expected %q invalid", s)
		}
	}
}

func TestMemNormalizeSlug(t *testing.T) {
	got, err := memNormalizeSlug("foo")
	if err != nil || got != "mem/foo" {
		t.Fatalf("got %q, %v", got, err)
	}
	got, err = memNormalizeSlug("core")
	if err != nil || got != "core" {
		t.Fatalf("got %q, %v", got, err)
	}
	got, err = memNormalizeSlug("mem/foo")
	if err != nil || got != "mem/foo" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestMemBodyRoundTrip(t *testing.T) {
	value := "hello, agent memory"
	plaintext, err := memEncodeMemoryBody("mem/example", &value)
	if err != nil {
		t.Fatal(err)
	}
	body, err := memDecodeBody(plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if body.IsCore || body.Value == nil || *body.Value != value || body.Slug != "mem/example" {
		t.Fatalf("round trip mismatch: %+v", body)
	}

	// Tombstone.
	plaintext, err = memEncodeMemoryBody("mem/example", nil)
	if err != nil {
		t.Fatal(err)
	}
	body, err = memDecodeBody(plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if body.Value != nil {
		t.Fatalf("expected tombstone, got %+v", body)
	}

	// Core.
	plaintext, err = memEncodeCoreBody("test agent profile")
	if err != nil {
		t.Fatal(err)
	}
	body, err = memDecodeBody(plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if !body.IsCore || body.Profile != "test agent profile" {
		t.Fatalf("core round trip mismatch: %+v", body)
	}
}

func TestMemSelectHead(t *testing.T) {
	events := []nostr.Event{
		{ID: "bb", CreatedAt: 100},
		{ID: "aa", CreatedAt: 100},
		{ID: "cc", CreatedAt: 50},
	}
	head := memSelectHead(events)
	if head == nil || head.ID != "aa" {
		t.Fatalf("expected tie-break to lowest id, got %+v", head)
	}

	events = []nostr.Event{{ID: "x", CreatedAt: 1}, {ID: "y", CreatedAt: 5}}
	head = memSelectHead(events)
	if head == nil || head.ID != "y" {
		t.Fatalf("expected greatest created_at to win, got %+v", head)
	}

	if memSelectHead(nil) != nil {
		t.Fatalf("expected nil for empty input")
	}
}

func TestMemMonotonicCreatedAt(t *testing.T) {
	if got := memMonotonicCreatedAt(100, nil); got != 100 {
		t.Fatalf("got %d, want 100", got)
	}
	prior := &nostr.Event{CreatedAt: 100}
	if got := memMonotonicCreatedAt(50, prior); got != 101 {
		t.Fatalf("got %d, want 101 (monotonic bump)", got)
	}
	if got := memMonotonicCreatedAt(200, prior); got != 200 {
		t.Fatalf("got %d, want 200 (now already ahead)", got)
	}
}

// ── diff engine (mirrors the Rust oracle's unit tests in mem.rs) ───────────

func TestMemDiffApplySimple(t *testing.T) {
	current := "alpha\nbeta\ngamma\n"
	patch := "--- a/x\n+++ b/x\n@@ -1,3 +1,3 @@\n alpha\n-beta\n+delta\n gamma\n"
	hunks, err := memParsePatch(patch)
	if err != nil {
		t.Fatal(err)
	}
	got, err := memApplyPatch(current, hunks)
	if err != nil {
		t.Fatal(err)
	}
	if want := "alpha\ndelta\ngamma\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestMemDiffRejectsOffsetSlide(t *testing.T) {
	// A hunk declaring "@@ -1,3 @@" against "zero\nalpha\nbeta\ngamma\n"
	// must be rejected at the declared position rather than sliding to
	// where the context happens to match (line 2).
	current := "zero\nalpha\nbeta\ngamma\n"
	patch := "--- a/x\n+++ b/x\n@@ -1,3 +1,3 @@\n alpha\n-beta\n+delta\n gamma\n"
	hunks, err := memParsePatch(patch)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := memApplyPatch(current, hunks); err == nil {
		t.Fatal("expected strict-position rejection, got success")
	}
}

func TestMemDiffPureInsertionIntoEmpty(t *testing.T) {
	patch := "--- a/x\n+++ b/x\n@@ -0,0 +1,2 @@\n+first\n+second\n"
	hunks, err := memParsePatch(patch)
	if err != nil {
		t.Fatal(err)
	}
	got, err := memApplyPatch("", hunks)
	if err != nil {
		t.Fatal(err)
	}
	if want := "first\nsecond\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestMemDiffMultiHunkAgainstOriginalPositions(t *testing.T) {
	current := "a\nb\nc\nd\ne\nf\ng\nh\ni\nj\nk\nl\n"
	patch := "--- a/x\n+++ b/x\n@@ -1,3 +1,3 @@\n a\n-b\n+B\n c\n@@ -10,3 +10,3 @@\n j\n-k\n+K\n l\n"
	hunks, err := memParsePatch(patch)
	if err != nil {
		t.Fatal(err)
	}
	got, err := memApplyPatch(current, hunks)
	if err != nil {
		t.Fatal(err)
	}
	if want := "a\nB\nc\nd\ne\nf\ng\nh\ni\nj\nK\nl\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestMemDiffNoTrailingNewline(t *testing.T) {
	current := "alpha\nbeta\ngamma"
	patch := "--- a/x\n+++ b/x\n@@ -1,3 +1,3 @@\n alpha\n-beta\n+delta\n gamma\n\\ No newline at end of file\n"
	hunks, err := memParsePatch(patch)
	if err != nil {
		t.Fatal(err)
	}
	got, err := memApplyPatch(current, hunks)
	if err != nil {
		t.Fatal(err)
	}
	if want := "alpha\ndelta\ngamma"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestMemDiffMismatchedContextRejected(t *testing.T) {
	current := "alpha\nbeta\ngamma\n"
	patch := "--- a/x\n+++ b/x\n@@ -1,3 +1,3 @@\n alpha\n-BETA\n+delta\n gamma\n"
	hunks, err := memParsePatch(patch)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := memApplyPatch(current, hunks); err == nil {
		t.Fatal("expected mismatch rejection")
	}
}
