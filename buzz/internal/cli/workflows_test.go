package cli

import "testing"

func TestNormalizeWriteResponsePassesThroughRawWhenUnrecognized(t *testing.T) {
	raw := []byte(`{"foo":"bar"}`)
	got := normalizeWriteResponse(raw)
	if string(got) != string(raw) {
		t.Fatalf("expected pass-through, got %s", got)
	}
}

func TestNormalizeWriteResponseReshapesKnownFields(t *testing.T) {
	raw := []byte(`{"event_id":"abc","accepted":true,"message":"ok","extra":"dropped"}`)
	got := normalizeWriteResponse(raw)
	want := `{"accepted":true,"event_id":"abc","message":"ok"}`
	if string(got) != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestNormalizeWorkflowEvent(t *testing.T) {
	ev := map[string]any{
		"pubkey":     "abc123",
		"content":    "yaml: true",
		"created_at": float64(1700000000),
		"tags":       []any{[]any{"d", "wf-1"}, []any{"h", "chan-1"}},
	}
	got := normalizeWorkflowEvent(ev)
	if got["workflow_id"] != "wf-1" || got["pubkey"] != "abc123" || got["content"] != "yaml: true" {
		t.Fatalf("got %+v", got)
	}
	if got["created_at"] != uint64(1700000000) {
		t.Fatalf("got created_at=%v", got["created_at"])
	}
}
