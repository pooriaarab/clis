package llm

import (
	"canadabuys-cli/internal/score"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoKeyNamesBothVars(t *testing.T) {
	t.Setenv("CANADABUYS_LLM_API_KEY", "")
	t.Setenv("CEREBRAS_API_KEY", "")
	_, _, err := (&Client{CacheDir: t.TempDir()}).Enrich(nil, "m")
	if err == nil || !strings.Contains(err.Error(), "CANADABUYS_LLM_API_KEY") || !strings.Contains(err.Error(), "CEREBRAS_API_KEY") {
		t.Fatalf("empty key: %v", err)
	}
}

func TestPrefersGeneralKeyEnv(t *testing.T) {
	t.Setenv("CANADABUYS_LLM_API_KEY", "general")
	t.Setenv("CEREBRAS_API_KEY", "cerebras")
	c := &Client{CacheDir: t.TempDir()}
	if _, _, err := c.Enrich(nil, "m"); err != nil {
		t.Fatal(err)
	}
	if c.Key != "general" {
		t.Fatalf("key=%q", c.Key)
	}
}

func TestCacheHitIssuesNoRequest(t *testing.T) {
	dir := t.TempDir()
	o := score.Opportunity{Reference: "r1", Title: "t", Buyer: "b", Description: "d"}
	c := &Client{Key: "dummy", CacheDir: dir}
	want := Enrichment{Reference: "r1", Buildability: 60, Thesis: "x", Category: "y"}
	raw, _ := json.Marshal(want)
	if err := os.WriteFile(filepath.Join(dir, c.key(o, "m")), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	got, skip, err := c.Enrich([]score.Opportunity{o}, "m")
	if err != nil || skip != 0 || len(got) != 1 || got[0] == nil || *got[0] != want {
		t.Fatalf("got=%v skip=%d err=%v", got, skip, err)
	}
}
