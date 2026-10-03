package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKitRendersMarkdownAndJSON(t *testing.T) {
	sb := newSandbox(t)
	sites := sb.writeSites()
	out := filepath.Join(t.TempDir(), "kit")

	r := sb.Run("kit", "--sites", sites, "--out", out)
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "wrote")

	for _, id := range []string{"widgets", "gadgets"} {
		md, err := os.ReadFile(filepath.Join(out, id+".md"))
		if err != nil {
			t.Fatalf("missing %s.md: %v", id, err)
		}
		for _, want := range []string{"## Tagline", "## One-liner", "## Short", "## Medium", "## Full", "Tags:"} {
			if !strings.Contains(string(md), want) {
				t.Errorf("%s.md misses section %q", id, want)
			}
		}
		js, err := os.ReadFile(filepath.Join(out, id+".json"))
		if err != nil {
			t.Fatalf("missing %s.json: %v", id, err)
		}
		var v map[string]any
		if err := json.Unmarshal(js, &v); err != nil {
			t.Fatalf("%s.json is not JSON: %v", id, err)
		}
		if v["id"] != id || v["tagline"] == "" || v["full2200"] == "" {
			t.Errorf("%s.json misses copy fields: %v", id, v)
		}
	}

	r = sb.Run("kit", "--sites", sites, "--out", out, "--json")
	wantExit(t, r, 0)
	v := r.JSON(t)
	if v["ok"] != true {
		t.Errorf("json ok = %v", v["ok"])
	}
}

func TestKitRejectsBadYAML(t *testing.T) {
	sb := newSandbox(t)
	path := filepath.Join(t.TempDir(), "sites.yaml")
	full := strings.Repeat("x", 2501)
	body := "sites:\n  - id: bad\n    name: Bad\n    url: https://bad.example.com\n" +
		"    tagline: t\n    one_liner: o\n    short: s\n    medium: m\n" +
		"    full2200: " + full + "\n" +
		"    tags: [a, b, c, d, e]\n    founder: F\n    email: f@example.com\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	r := sb.Run("kit", "--sites", path, "--out", t.TempDir())
	wantExit(t, r, 2)
	wantContains(t, "output", r.Stdout+r.Stderr, "full2200")
}

func TestKitRejectsTraversalID(t *testing.T) {
	sb := newSandbox(t)
	path := filepath.Join(t.TempDir(), "sites.yaml")
	body := "sites:\n  - id: ../evil\n    name: Evil\n    url: https://evil.example.com\n" +
		"    tagline: t\n    one_liner: o\n    short: s\n    medium: m\n    full2200: f\n" +
		"    tags: [a, b, c, d, e]\n    founder: F\n    email: f@example.com\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	r := sb.Run("kit", "--sites", path, "--out", t.TempDir())
	wantExit(t, r, 2)
	wantContains(t, "output", r.Stdout+r.Stderr, "must be a slug")
}

func TestKitDryRunWritesNothing(t *testing.T) {
	sb := newSandbox(t)
	sites := sb.writeSites()
	out := filepath.Join(t.TempDir(), "kit")

	r := sb.Run("kit", "--sites", sites, "--out", out, "--dry-run")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "would write")
	wantContains(t, "stdout", r.Stdout, "dry run: nothing was written")
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Errorf("dry run created %s", out)
	}
}
