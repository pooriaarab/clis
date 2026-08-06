package desktopstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
	"time"
)

func TestLoadRawMissingFileReturnsEmptySlice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "managed-agents.json")

	got, err := LoadRaw(path)
	if err != nil {
		t.Fatalf("LoadRaw() error = %v", err)
	}
	if got == nil {
		t.Fatal("LoadRaw() returned nil slice")
	}
	if len(got) != 0 {
		t.Fatalf("LoadRaw() len = %d, want 0", len(got))
	}
}

func TestSaveRawLoadRawRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "managed-agents.json")
	records := []json.RawMessage{
		json.RawMessage(`{"pubkey":"` + hex64("a") + `","name":"alpha"}`),
		json.RawMessage(`{"pubkey":"` + hex64("b") + `","name":"beta"}`),
	}

	if err := SaveRaw(path, records); err != nil {
		t.Fatalf("SaveRaw() error = %v", err)
	}
	got, err := LoadRaw(path)
	if err != nil {
		t.Fatalf("LoadRaw() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("LoadRaw() len = %d, want 2", len(got))
	}
	assertRawJSONEqual(t, records[0], got[0])
	assertRawJSONEqual(t, records[1], got[1])
}

func TestSaveRawPreservesUnknownFieldsWhenAppending(t *testing.T) {
	path := filepath.Join(t.TempDir(), "managed-agents.json")
	sibling := json.RawMessage(`{"pubkey":"` + hex64("c") + `","name":"sibling","totally_unknown_field_from_a_future_desktop_version":{"nested":true}}`)
	if err := os.WriteFile(path, []byte("["+string(sibling)+"]"), 0o600); err != nil {
		t.Fatalf("write seed: %v", err)
	}

	records, err := LoadRaw(path)
	if err != nil {
		t.Fatalf("LoadRaw() error = %v", err)
	}
	records = append(records, json.RawMessage(`{"pubkey":"`+hex64("d")+`","name":"new"}`))
	if err := SaveRaw(path, records); err != nil {
		t.Fatalf("SaveRaw() error = %v", err)
	}

	got, err := LoadRaw(path)
	if err != nil {
		t.Fatalf("LoadRaw() after append error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("LoadRaw() len = %d, want 2", len(got))
	}
	var first map[string]any
	if err := json.Unmarshal(got[0], &first); err != nil {
		t.Fatalf("unmarshal sibling: %v", err)
	}
	unknown, ok := first["totally_unknown_field_from_a_future_desktop_version"].(map[string]any)
	if !ok {
		t.Fatalf("unknown field missing or wrong type: %#v", first["totally_unknown_field_from_a_future_desktop_version"])
	}
	if unknown["nested"] != true {
		t.Fatalf("unknown nested value = %#v, want true", unknown["nested"])
	}
}

func TestSaveRawCreatesBackupOfPreviousContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "managed-agents.json")
	before := json.RawMessage(`{"pubkey":"` + hex64("e") + `","name":"before"}`)
	if err := SaveRaw(path, []json.RawMessage{before}); err != nil {
		t.Fatalf("initial SaveRaw() error = %v", err)
	}

	after := json.RawMessage(`{"pubkey":"` + hex64("f") + `","name":"after"}`)
	if err := SaveRaw(path, []json.RawMessage{after}); err != nil {
		t.Fatalf("second SaveRaw() error = %v", err)
	}

	backup, err := LoadRaw(path + ".bak")
	if err != nil {
		t.Fatalf("LoadRaw(backup) error = %v", err)
	}
	if len(backup) != 1 {
		t.Fatalf("backup len = %d, want 1", len(backup))
	}
	assertRawJSONEqual(t, before, backup[0])
}

func TestSaveRawLeavesNoTempFileAfterSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "managed-agents.json")
	if err := SaveRaw(path, []json.RawMessage{json.RawMessage(`{"name":"agent"}`)}); err != nil {
		t.Fatalf("SaveRaw() error = %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".managed-agents-*.tmp"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("leftover temp files = %v", matches)
	}
}

// TestWithFileLockFallsBackWhenLockHeldElsewhere simulates the Buzz Desktop
// app holding an exclusive flock on managed-agents.json.lock (e.g. while it
// keeps the store open): the CLI write must not deadlock waiting on that
// lock forever - it should give up after its timeout and fall back to the
// unlocked atomic temp-write+rename, which is safe on its own because
// rename is atomic.
func TestWithFileLockFallsBackWhenLockHeldElsewhere(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "managed-agents.json.lock")

	holder, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatalf("open lock file: %v", err)
	}
	defer holder.Close()
	if err := syscall.Flock(int(holder.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatalf("hold exclusive flock: %v", err)
	}

	storePath := filepath.Join(dir, "managed-agents.json")
	called := false
	start := time.Now()
	err = WithFileLock(lockPath, 200*time.Millisecond, func() error {
		called = true
		return SaveRaw(storePath, []json.RawMessage{json.RawMessage(`{"name":"agent"}`)})
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("WithFileLock() error = %v, want fallback write to succeed", err)
	}
	if !called {
		t.Fatal("WithFileLock() never called fn despite an externally held lock")
	}
	if elapsed > 3*time.Second {
		t.Fatalf("WithFileLock() took %s, want it bounded by its timeout", elapsed)
	}

	got, err := LoadRaw(storePath)
	if err != nil {
		t.Fatalf("LoadRaw() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("LoadRaw() len = %d, want 1 (write succeeded despite contended lock)", len(got))
	}
}

func assertRawJSONEqual(t *testing.T, want, got json.RawMessage) {
	t.Helper()
	var wantValue any
	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatalf("unmarshal want: %v", err)
	}
	var gotValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("json mismatch got %#v, want %#v", gotValue, wantValue)
	}
}

func hex64(ch string) string {
	out := ""
	for i := 0; i < 64; i++ {
		out += ch
	}
	return out
}
