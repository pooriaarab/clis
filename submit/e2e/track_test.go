package e2e

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTrackInitSetList(t *testing.T) {
	sb := newSandbox(t)

	r := sb.Run("track", "init")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "store ready")
	if _, err := os.Stat(filepath.Join(sb.ConfigDir, "store.json")); err != nil {
		t.Fatalf("store file missing: %v", err)
	}

	r = sb.Run("track", "set", "--site", "widgets", "--site-url", "https://widgets.example.com",
		"--directory", "crunchbase", "--status", "submitted",
		"--listing-url", "https://www.crunchbase.com/widgets")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "set widgets x crunchbase to submitted")

	r = sb.Run("track", "list")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "widgets")
	wantContains(t, "stdout", r.Stdout, "crunchbase")

	r = sb.Run("track", "list", "--json")
	wantExit(t, r, 0)
	v := r.JSON(t)
	if v["ok"] != true {
		t.Errorf("json ok = %v", v["ok"])
	}
	recs, _ := v["records"].([]any)
	if len(recs) != 1 {
		t.Fatalf("records = %d, want 1", len(recs))
	}
	rec := recs[0].(map[string]any)
	if rec["status"] != "submitted" || rec["listing_url"] != "https://www.crunchbase.com/widgets" {
		t.Errorf("record = %v", rec)
	}

	// A status-only set keeps the listing URL.
	r = sb.Run("track", "set", "--site", "widgets", "--directory", "crunchbase", "--status", "live")
	wantExit(t, r, 0)
	r = sb.Run("track", "list", "--status", "live", "--json")
	v = r.JSON(t)
	recs, _ = v["records"].([]any)
	if len(recs) != 1 || recs[0].(map[string]any)["listing_url"] != "https://www.crunchbase.com/widgets" {
		t.Errorf("listing URL lost on status-only set: %v", recs)
	}
}

func TestTrackSetEnforcesDailyLimit(t *testing.T) {
	sb := newSandbox(t)
	sb.Run("track", "init")
	today := time.Now().Format("2006-01-02")
	dirs := []string{"crunchbase", "launching-next", "saashub", "uneed", "startupbase",
		"peerpush", "f6s", "wellfound", "indie-hackers", "linkcentre"}
	for _, d := range dirs {
		r := sb.Run("track", "set", "--site", "widgets", "--directory", d,
			"--status", "submitted", "--date", today)
		wantExit(t, r, 0)
	}
	r := sb.Run("track", "set", "--site", "widgets", "--directory", "betalist",
		"--status", "submitted", "--date", today)
	wantExit(t, r, 2)
	wantContains(t, "output", r.Stdout+r.Stderr, "daily limit")

	// Drafts do not count, and another day is fine.
	r = sb.Run("track", "set", "--site", "widgets", "--directory", "betalist", "--status", "draft")
	wantExit(t, r, 0)
	r = sb.Run("track", "set", "--site", "widgets", "--directory", "betalist",
		"--status", "submitted", "--date", "2000-01-01")
	wantExit(t, r, 0)
}

func TestTrackSetRejectsBadInput(t *testing.T) {
	sb := newSandbox(t)
	sb.Run("track", "init")

	r := sb.Run("track", "set", "--site", "widgets", "--directory", "nosuch", "--status", "submitted")
	wantExit(t, r, 2)
	wantContains(t, "output", r.Stdout+r.Stderr, "unknown directory")

	r = sb.Run("track", "set", "--site", "widgets", "--directory", "crunchbase", "--status", "maybe")
	wantExit(t, r, 2)
	wantContains(t, "output", r.Stdout+r.Stderr, "status must be")

	r = sb.Run("track", "set", "--site", "widgets", "--directory", "crunchbase")
	wantExit(t, r, 2)
}

func TestTrackSetDryRunWritesNothing(t *testing.T) {
	sb := newSandbox(t)
	sb.Run("track", "init")

	r := sb.Run("track", "set", "--site", "widgets", "--directory", "crunchbase",
		"--status", "submitted", "--dry-run")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "would set")

	r = sb.Run("track", "list")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "no rows")
}
