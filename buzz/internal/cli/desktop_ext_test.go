package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"buzz-cli/internal/desktopstore"
	"buzz-cli/internal/nostr"
)

func TestDesktopListReadsStore(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "managed-agents.json")
	seed := `[
		{"name":"builtin","pubkey":"` + strings.Repeat("a", 64) + `","agent_command":"harness-a","relay_url":"https://relay.example/a","is_builtin":true,"is_active":true},
		{"name":"custom","pubkey":"` + strings.Repeat("b", 64) + `","agent_command":"harness-b","relay_url":"https://relay.example/b","is_builtin":false,"is_active":false}
	]`
	if err := os.WriteFile(storePath, []byte(seed), 0o600); err != nil {
		t.Fatalf("write seed: %v", err)
	}

	stdout, _, err := executeBuzz(t, "desktop", "list", "--store-path", storePath)
	if err != nil {
		t.Fatalf("execute desktop list error = %v", err)
	}
	var got []desktopListProbe
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, stdout.String())
	}
	if len(got) != 2 {
		t.Fatalf("list len = %d, want 2: %#v", len(got), got)
	}
	if got[0].Name != "builtin" || got[0].PubKey != strings.Repeat("a", 64) || got[0].AgentCommand != "harness-a" || got[0].RelayURL != "https://relay.example/a" || !got[0].IsBuiltin || !got[0].IsActive {
		t.Fatalf("first record = %#v", got[0])
	}
	if got[1].Name != "custom" || got[1].PubKey != strings.Repeat("b", 64) || got[1].AgentCommand != "harness-b" || got[1].RelayURL != "https://relay.example/b" || got[1].IsBuiltin || got[1].IsActive {
		t.Fatalf("second record = %#v", got[1])
	}
}

func TestDesktopListMissingStorePrintsEmptyArray(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "missing.json")

	stdout, _, err := executeBuzz(t, "desktop", "list", "--store-path", storePath)
	if err != nil {
		t.Fatalf("execute desktop list error = %v", err)
	}
	var got []desktopListProbe
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, stdout.String())
	}
	if got == nil {
		t.Fatal("list output decoded as nil")
	}
	if len(got) != 0 {
		t.Fatalf("list len = %d, want 0", len(got))
	}
}

func TestDesktopCreateWritesStoreAndKeychain(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "managed-agents.json")
	relay := newFakeRelay(t, http.StatusOK)
	patchDesktopSecretStore(t, desktopstore.NewMemorySecretStore)

	stdout, _, err := executeBuzz(t,
		"--config", tempConfigPath(t),
		"--owner-key", newNsec(t),
		"desktop", "create",
		"--name", "desk-agent",
		"--harness", "codex --serve",
		"--community", relay.URL,
		"--system-prompt", "You are useful.",
		"--channels", "general,engineering",
		"--avatar", "https://example.test/avatar.png",
		"--model", "gpt-5",
		"--store-path", storePath,
		"--keychain-service", "test-service",
	)
	if err != nil {
		t.Fatalf("execute desktop create error = %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, stdout.String())
	}
	if result["name"] != "desk-agent" || result["store_path"] != storePath || result["keychain_service"] != "test-service" {
		t.Fatalf("unexpected result identity fields: %#v", result)
	}
	if result["keychain_stored"] != true || result["keychain_fallback_inline"] != false {
		t.Fatalf("unexpected keychain fields: %#v", result)
	}
	if relayErrors, ok := result["relay_errors"].([]any); !ok || len(relayErrors) != 0 {
		t.Fatalf("relay_errors = %#v, want empty array", result["relay_errors"])
	}
	if eventIDs, ok := result["event_ids"].([]any); !ok || len(eventIDs) != 5 {
		t.Fatalf("event_ids = %#v, want 5 ids", result["event_ids"])
	}

	records, err := desktopstore.LoadRaw(storePath)
	if err != nil {
		t.Fatalf("LoadRaw() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("records len = %d, want 1", len(records))
	}
	var record desktopstore.ManagedAgentRecord
	if err := json.Unmarshal(records[0], &record); err != nil {
		t.Fatalf("unmarshal record: %v\n%s", err, string(records[0]))
	}
	if record.Name != "desk-agent" || record.ACPCommand != "buzz-acp" || record.AgentCommand != "codex --serve" || record.RelayURL != relay.URL {
		t.Fatalf("unexpected record core fields: %#v", record)
	}
	if record.TurnTimeoutSeconds != 320 || record.Parallelism != 10 || record.RespondTo != "owner-only" || record.IsBuiltin || !record.IsActive {
		t.Fatalf("unexpected record defaults: %#v", record)
	}
	assertJSONKeyIsArray(t, records[0], "agent_args")
	assertJSONKeyIsArray(t, records[0], "respond_to_allowlist")
	if record.PrivateKeyNsec != "" {
		t.Fatalf("PrivateKeyNsec = %q, want omitted/empty when keychain store succeeds", record.PrivateKeyNsec)
	}
	for _, key := range []string{
		"persona_id",
		"agent_command_override",
		"idle_timeout_seconds",
		"max_turn_duration_seconds",
		"provider",
		"persona_source_version",
		"runtime_pid",
		"backend_agent_id",
		"provider_binary_path",
		"last_started_at",
		"last_stopped_at",
		"last_exit_code",
		"last_error",
		"last_error_code",
	} {
		assertJSONKeyNull(t, records[0], key)
	}
}

func TestDesktopCreateWritesLocalStoreWhenRelayFails(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "managed-agents.json")
	patchDesktopSecretStore(t, desktopstore.NewMemorySecretStore)
	relay := newFakeRelay(t, http.StatusInternalServerError)

	stdout, stderr, err := executeBuzz(t,
		"--config", tempConfigPath(t),
		"--owner-key", newNsec(t),
		"desktop", "create",
		"--name", "relay-fail",
		"--harness", "codex",
		"--community", relay.URL,
		"--system-prompt", "Keep going.",
		"--store-path", storePath,
	)
	// A relay failure must never fail the whole command: local store +
	// keychain already succeeded, and the desktop app can re-publish later.
	if err != nil {
		t.Fatalf("execute desktop create error = %v, want exit 0 despite relay failure", err)
	}
	if !strings.Contains(stderr.String(), "relay publish") {
		t.Fatalf("stderr = %q, want a relay-publish warning", stderr.String())
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, stdout.String())
	}
	if relayErrors, ok := result["relay_errors"].([]any); !ok || len(relayErrors) == 0 {
		t.Fatalf("relay_errors = %#v, want non-empty array", result["relay_errors"])
	}
	records, err := desktopstore.LoadRaw(storePath)
	if err != nil {
		t.Fatalf("LoadRaw() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("records len = %d, want local record despite relay failure", len(records))
	}
}

func TestDesktopCreateSlowRelayDoesNotHang(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "managed-agents.json")
	patchDesktopSecretStore(t, desktopstore.NewMemorySecretStore)
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // slower than the test's --timeout below
	}))
	t.Cleanup(relay.Close)

	start := time.Now()
	stdout, stderr, err := executeBuzz(t,
		"--config", tempConfigPath(t),
		"--owner-key", newNsec(t),
		"desktop", "create",
		"--name", "slow-relay",
		"--harness", "codex",
		"--community", relay.URL,
		"--system-prompt", "Keep going.",
		"--store-path", storePath,
		"--timeout", "300ms",
	)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("execute desktop create error = %v, want exit 0 despite a hanging relay", err)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("desktop create took %s, want it bounded by --timeout", elapsed)
	}
	if !strings.Contains(stderr.String(), "relay publish") {
		t.Fatalf("stderr = %q, want a relay-publish warning", stderr.String())
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, stdout.String())
	}
	if relayErrors, ok := result["relay_errors"].([]any); !ok || len(relayErrors) == 0 {
		t.Fatalf("relay_errors = %#v, want non-empty array", result["relay_errors"])
	}
	records, err := desktopstore.LoadRaw(storePath)
	if err != nil {
		t.Fatalf("LoadRaw() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("records len = %d, want local record despite hanging relay", len(records))
	}
}

func TestDesktopDeleteRemovesTargetAndPreservesSiblingUnknownFields(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "managed-agents.json")
	targetPub := strings.Repeat("a", 64)
	siblingPub := strings.Repeat("b", 64)
	relay, events := newRecordingRelay(t, http.StatusOK)
	writeDesktopRawRecords(t, storePath,
		desktopTestRawRecord("target", targetPub, false, relay.URL, nil),
		desktopTestRawRecord("sibling", siblingPub, false, "", map[string]any{"catalog_source": map[string]any{"future": true}}),
	)
	patchDesktopSecretStore(t, desktopstore.NewMemorySecretStore)
	ownerKeys, ownerNsec := newKeyAndNsec(t)

	stdout, _, err := executeBuzz(t,
		"--config", tempConfigPath(t),
		"--owner-key", ownerNsec,
		"--relay", relay.URL,
		"desktop", "delete", "target",
		"--store-path", storePath,
	)
	if err != nil {
		t.Fatalf("execute desktop delete error = %v", err)
	}
	var results []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, stdout.String())
	}
	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}
	result := results[0]
	if result["name"] != "target" || result["pubkey"] != targetPub || result["removed_from_store"] != true {
		t.Fatalf("unexpected delete result: %#v", result)
	}

	records, err := desktopstore.LoadRaw(storePath)
	if err != nil {
		t.Fatalf("LoadRaw() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("records len = %d, want 1 sibling", len(records))
	}
	var kept map[string]any
	if err := json.Unmarshal(records[0], &kept); err != nil {
		t.Fatalf("unmarshal kept sibling: %v", err)
	}
	if kept["name"] != "sibling" || kept["pubkey"] != siblingPub {
		t.Fatalf("kept record = %#v", kept)
	}
	assertNestedJSONBool(t, kept, "catalog_source", "future", true)

	published := events()
	if len(published) != 2 {
		t.Fatalf("published events len = %d, want 2: %#v", len(published), published)
	}
	if published[0].Kind != nostr.KindDeletion {
		t.Fatalf("first event kind = %d, want %d", published[0].Kind, nostr.KindDeletion)
	}
	wantDeleteTag := strings.Join([]string{"a", fmt.Sprintf("%d:%s:%s", nostr.KindManagedAgent, ownerKeys.PublicHex(), targetPub)}, "\x00")
	if len(published[0].Tags) != 1 || strings.Join(published[0].Tags[0], "\x00") != wantDeleteTag {
		t.Fatalf("delete tags = %#v", published[0].Tags)
	}
	if published[1].Kind != nostr.KindIAArchiveRequest {
		t.Fatalf("second event kind = %d, want %d", published[1].Kind, nostr.KindIAArchiveRequest)
	}
	if len(published[1].Tags) != 4 {
		t.Fatalf("archive tags len = %d, want 4: %#v", len(published[1].Tags), published[1].Tags)
	}
	wantArchiveTags := nostr.Tags{{"-"}, {"p", targetPub}, {"reason", "retired"}}
	for i, want := range wantArchiveTags {
		if strings.Join(published[1].Tags[i], "\x00") != strings.Join(want, "\x00") {
			t.Fatalf("archive tag %d = %#v, want %#v", i, published[1].Tags[i], want)
		}
	}
	if tag := published[1].Tags[3]; len(tag) != 4 || tag[0] != "auth" || tag[1] != ownerKeys.PublicHex() {
		t.Fatalf("archive auth tag = %#v", tag)
	}
}

func TestDesktopDeleteSkipsRelayForPubkeylessTemplateRecord(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "managed-agents.json")
	// A relay that would fail the test if the command ever dialed it - a
	// pubkey-less record has no nostr identity, so there is nothing to
	// delete on the relay and it must never be contacted.
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected relay request for pubkey-less record: %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(relay.Close)
	writeDesktopRawRecords(t, storePath, desktopTestRawRecord("template", "", false, "", nil))
	patchDesktopSecretStore(t, desktopstore.NewMemorySecretStore)

	start := time.Now()
	stdout, _, err := executeBuzz(t,
		"--config", tempConfigPath(t),
		"--owner-key", newNsec(t),
		"--relay", relay.URL,
		"desktop", "delete", "template",
		"--store-path", storePath,
	)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("execute desktop delete error = %v", err)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("desktop delete of a pubkey-less record took %s, want near-instant (no relay dial)", elapsed)
	}
	var results []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, stdout.String())
	}
	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}
	result := results[0]
	if result["removed_from_store"] != true {
		t.Fatalf("removed_from_store = %#v, want true", result["removed_from_store"])
	}
	if result["relay_skipped_reason"] == nil {
		t.Fatalf("relay_skipped_reason = %#v, want a reason", result["relay_skipped_reason"])
	}

	records, err := desktopstore.LoadRaw(storePath)
	if err != nil {
		t.Fatalf("LoadRaw() error = %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("records len = %d, want 0", len(records))
	}
}

func TestDesktopDeleteHangingRelayDoesNotHang(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "managed-agents.json")
	targetPub := strings.Repeat("e", 64)
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // slower than the test's --timeout below
	}))
	t.Cleanup(relay.Close)
	writeDesktopRawRecords(t, storePath, desktopTestRawRecord("target", targetPub, false, relay.URL, nil))
	patchDesktopSecretStore(t, desktopstore.NewMemorySecretStore)

	start := time.Now()
	stdout, _, err := executeBuzz(t,
		"--config", tempConfigPath(t),
		"--owner-key", newNsec(t),
		"desktop", "delete", "target",
		"--store-path", storePath,
		"--timeout", "300ms",
	)
	elapsed := time.Since(start)
	// A hanging relay must still return a relay-publish error (unlike
	// create, delete keeps the ExitRelay signal since callers may want to
	// retry the delete), but it must never block the command itself.
	var exitErr ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != ExitRelay {
		t.Fatalf("error = %v, want ExitRelay", err)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("desktop delete took %s, want it bounded by --timeout", elapsed)
	}

	var results []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, stdout.String())
	}
	if len(results) != 1 || results[0]["removed_from_store"] != true {
		t.Fatalf("results = %#v, want local removal despite hanging relay", results)
	}
	relayErrors, ok := results[0]["relay_errors"].([]any)
	if !ok || len(relayErrors) == 0 {
		t.Fatalf("relay_errors = %#v, want non-empty array", results[0]["relay_errors"])
	}

	records, err := desktopstore.LoadRaw(storePath)
	if err != nil {
		t.Fatalf("LoadRaw() error = %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("records len = %d, want 0 (local removal happens before relay call)", len(records))
	}
}

func TestDesktopDeleteHandlesMultipleRecordsWithSameName(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "managed-agents.json")
	pubA := strings.Repeat("1", 64)
	pubB := strings.Repeat("2", 64)
	relay := newFakeRelay(t, http.StatusOK)
	writeDesktopRawRecords(t, storePath,
		desktopTestRawRecord("dupe", pubA, false, relay.URL, nil),
		desktopTestRawRecord("dupe", pubB, false, relay.URL, nil),
		desktopTestRawRecord("keep-me", strings.Repeat("3", 64), false, "", nil),
	)
	patchDesktopSecretStore(t, desktopstore.NewMemorySecretStore)

	stdout, _, err := executeBuzz(t,
		"--config", tempConfigPath(t),
		"--owner-key", newNsec(t),
		"desktop", "delete", "dupe",
		"--store-path", storePath,
	)
	if err != nil {
		t.Fatalf("execute desktop delete error = %v", err)
	}
	var results []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, stdout.String())
	}
	if len(results) != 2 {
		t.Fatalf("results len = %d, want 2 (both records named dupe)", len(results))
	}
	gotPubs := map[string]bool{}
	for _, r := range results {
		if r["removed_from_store"] != true {
			t.Fatalf("removed_from_store = %#v, want true: %#v", r["removed_from_store"], r)
		}
		gotPubs[r["pubkey"].(string)] = true
	}
	if !gotPubs[pubA] || !gotPubs[pubB] {
		t.Fatalf("results pubkeys = %#v, want both %s and %s", gotPubs, pubA, pubB)
	}

	records, err := desktopstore.LoadRaw(storePath)
	if err != nil {
		t.Fatalf("LoadRaw() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("records len = %d, want 1 (only keep-me survives)", len(records))
	}
	var kept map[string]any
	if err := json.Unmarshal(records[0], &kept); err != nil {
		t.Fatalf("unmarshal kept record: %v", err)
	}
	if kept["name"] != "keep-me" {
		t.Fatalf("kept record = %#v, want keep-me", kept)
	}
}

func TestDesktopDeleteBuiltinRequiresForceAndDoesNotTouchStore(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "managed-agents.json")
	writeDesktopRawRecords(t, storePath, desktopTestRawRecord("builtin", strings.Repeat("c", 64), true, "https://relay.example", nil))
	before := decodeJSONFile(t, storePath)

	_, _, err := executeBuzz(t,
		"desktop", "delete", "builtin",
		"--store-path", storePath,
	)
	if err == nil {
		t.Fatal("expected input error")
	}
	var exitErr ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != ExitInput {
		t.Fatalf("error = %v, want ExitInput", err)
	}
	after := decodeJSONFile(t, storePath)
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("store changed after refused delete: got %#v want %#v", after, before)
	}
}

func TestDesktopDeleteUnknownAgent(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "managed-agents.json")
	writeDesktopRawRecords(t, storePath, desktopTestRawRecord("known", strings.Repeat("d", 64), false, "", nil))

	_, _, err := executeBuzz(t,
		"desktop", "delete", "unknown",
		"--store-path", storePath,
	)
	if err == nil {
		t.Fatal("expected input error")
	}
	var exitErr ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != ExitInput {
		t.Fatalf("error = %v, want ExitInput", err)
	}
}

func executeBuzz(t *testing.T, args ...string) (*bytes.Buffer, *bytes.Buffer, error) {
	t.Helper()
	root, opts := NewRootCommand()
	var stdout, stderr bytes.Buffer
	opts.Stdout = &stdout
	opts.Stderr = &stderr
	root.SetArgs(args)
	err := root.Execute()
	return &stdout, &stderr, err
}

func newFakeRelay(t *testing.T, status int) *httptest.Server {
	t.Helper()
	server, _ := newRecordingRelay(t, status)
	return server
}

func newRecordingRelay(t *testing.T, status int) (*httptest.Server, func() []nostr.Event) {
	t.Helper()
	var mu sync.Mutex
	var events []nostr.Event
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/events" {
			t.Fatalf("unexpected relay request %s %s", r.Method, r.URL.Path)
		}
		var event nostr.Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Fatalf("decode relay event: %v", err)
		}
		mu.Lock()
		events = append(events, event)
		mu.Unlock()
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(status)
		if status >= 200 && status < 300 {
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":false,"message":"fail"}`))
	}))
	t.Cleanup(server.Close)
	return server, func() []nostr.Event {
		mu.Lock()
		defer mu.Unlock()
		out := make([]nostr.Event, len(events))
		copy(out, events)
		return out
	}
}

func patchDesktopSecretStore(t *testing.T, fn func() *desktopstore.MemorySecretStore) {
	t.Helper()
	old := newDesktopSecretStore
	store := fn()
	newDesktopSecretStore = func(string) desktopstore.SecretStore { return store }
	t.Cleanup(func() { newDesktopSecretStore = old })
}

func newNsec(t *testing.T) string {
	t.Helper()
	_, nsec := newKeyAndNsec(t)
	return nsec
}

func newKeyAndNsec(t *testing.T) (*nostr.KeyPair, string) {
	t.Helper()
	keys, err := nostr.NewKeyPair()
	if err != nil {
		t.Fatalf("NewKeyPair() error = %v", err)
	}
	nsec, err := keys.Nsec()
	if err != nil {
		t.Fatalf("Nsec() error = %v", err)
	}
	return keys, nsec
}

func tempConfigPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "config.toml")
}

func desktopTestRawRecord(name, pubkey string, builtin bool, relayURL string, extra map[string]any) json.RawMessage {
	now := "2026-08-06T00:00:00Z"
	record := map[string]any{
		"pubkey":                        pubkey,
		"name":                          name,
		"persona_id":                    nil,
		"auth_tag":                      nil,
		"relay_url":                     relayURL,
		"avatar_url":                    nil,
		"acp_command":                   "buzz-acp",
		"agent_command":                 "codex",
		"agent_command_override":        nil,
		"agent_args":                    []string{},
		"mcp_command":                   "",
		"turn_timeout_seconds":          320,
		"idle_timeout_seconds":          nil,
		"max_turn_duration_seconds":     nil,
		"parallelism":                   10,
		"system_prompt":                 "prompt",
		"model":                         nil,
		"provider":                      nil,
		"persona_source_version":        nil,
		"start_on_app_launch":           true,
		"auto_restart_on_config_change": true,
		"runtime_pid":                   nil,
		"backend":                       map[string]any{"type": "local"},
		"backend_agent_id":              nil,
		"provider_binary_path":          nil,
		"created_at":                    now,
		"updated_at":                    now,
		"last_started_at":               nil,
		"last_stopped_at":               nil,
		"last_exit_code":                nil,
		"last_error":                    nil,
		"last_error_code":               nil,
		"respond_to":                    "owner-only",
		"respond_to_allowlist":          []string{},
		"is_builtin":                    builtin,
		"is_active":                     true,
	}
	for key, value := range extra {
		record[key] = value
	}
	b, err := json.Marshal(record)
	if err != nil {
		panic(err)
	}
	return b
}

func writeDesktopRawRecords(t *testing.T, path string, records ...json.RawMessage) {
	t.Helper()
	if err := desktopstore.SaveRaw(path, records); err != nil {
		t.Fatalf("SaveRaw() error = %v", err)
	}
}

func decodeJSONFile(t *testing.T, path string) any {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	var value any
	if err := json.Unmarshal(b, &value); err != nil {
		t.Fatalf("Unmarshal(%s): %v", path, err)
	}
	return value
}

func assertJSONKeyIsArray(t *testing.T, raw json.RawMessage, key string) {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if _, ok := value[key].([]any); !ok {
		t.Fatalf("%s = %#v, want JSON array", key, value[key])
	}
}

func assertJSONKeyNull(t *testing.T, raw json.RawMessage, key string) {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	got, ok := value[key]
	if !ok {
		t.Fatalf("%s key missing from JSON object", key)
	}
	if got != nil {
		t.Fatalf("%s = %#v, want null", key, got)
	}
}

func assertNestedJSONBool(t *testing.T, value map[string]any, outer, inner string, want bool) {
	t.Helper()
	nested, ok := value[outer].(map[string]any)
	if !ok {
		t.Fatalf("%s = %#v, want object", outer, value[outer])
	}
	if nested[inner] != want {
		t.Fatalf("%s.%s = %#v, want %v", outer, inner, nested[inner], want)
	}
}
