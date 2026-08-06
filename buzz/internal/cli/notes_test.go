package cli

import (
	"strings"
	"testing"
)

func TestParseSlug(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "lowercase", raw: "release-notes"},
		{name: "digits", raw: "note123"},
		{name: "dots and underscores", raw: "daily.note_v2"},
		{name: "empty", raw: "", wantErr: true},
		{name: "uppercase", raw: "Release", wantErr: true},
		{name: "space", raw: "release notes", wantErr: true},
		{name: "overlong", raw: strings.Repeat("a", 81), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSlug(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseSlug(%q) error = nil", tt.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseSlug(%q) error = %v", tt.raw, err)
			}
			if got != tt.raw {
				t.Fatalf("parseSlug(%q) = %q", tt.raw, got)
			}
		})
	}
}

func TestNoteSnapshotFromEvent(t *testing.T) {
	published := "1713957000"
	ev := map[string]any{
		"id":         "event-id",
		"pubkey":     "author",
		"kind":       float64(30023),
		"created_at": float64(1713958000),
		"content":    "body",
		"tags": []any{
			[]any{"d", "release-notes"},
			[]any{"title", "Release Notes"},
			[]any{"summary", "Summary"},
			[]any{"t", "release"},
			[]any{"t", "cli"},
			[]any{"published_at", published},
		},
	}

	snapshot, err := noteSnapshotFromEvent(ev)
	if err != nil {
		t.Fatalf("noteSnapshotFromEvent() error = %v", err)
	}
	if snapshot.ID != "event-id" || snapshot.PubKey != "author" || snapshot.Slug != "release-notes" {
		t.Fatalf("snapshot identity fields = %#v", snapshot)
	}
	if snapshot.Title != "Release Notes" {
		t.Fatalf("Title = %q", snapshot.Title)
	}
	if snapshot.Summary == nil || *snapshot.Summary != "Summary" {
		t.Fatalf("Summary = %#v", snapshot.Summary)
	}
	if len(snapshot.Tags) != 2 || snapshot.Tags[0] != "release" || snapshot.Tags[1] != "cli" {
		t.Fatalf("Tags = %#v", snapshot.Tags)
	}
	if snapshot.PublishedAt == nil || *snapshot.PublishedAt != 1713957000 {
		t.Fatalf("PublishedAt = %#v", snapshot.PublishedAt)
	}
	if snapshot.UpdatedAt != 1713958000 || snapshot.Content != "body" {
		t.Fatalf("snapshot content fields = %#v", snapshot)
	}
}

func TestNoteSnapshotFromEventMissingDTagErrors(t *testing.T) {
	_, err := noteSnapshotFromEvent(map[string]any{
		"kind": float64(30023),
		"tags": []any{[]any{"title", "Title"}},
	})
	if err == nil {
		t.Fatalf("noteSnapshotFromEvent() error = nil")
	}
}

func TestNoteSnapshotFromEventGarbagePublishedAtIsIgnored(t *testing.T) {
	snapshot, err := noteSnapshotFromEvent(map[string]any{
		"kind": float64(30023),
		"tags": []any{
			[]any{"d", "release-notes"},
			[]any{"published_at", "not-a-number"},
		},
	})
	if err != nil {
		t.Fatalf("noteSnapshotFromEvent() error = %v", err)
	}
	if snapshot.PublishedAt != nil {
		t.Fatalf("PublishedAt = %#v, want nil", snapshot.PublishedAt)
	}
}
