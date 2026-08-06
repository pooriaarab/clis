package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writePackFixture(t *testing.T, manifestJSON string, personas map[string]string, mcpJSON string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".plugin"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".plugin", "plugin.json"), []byte(manifestJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "personas"), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, content := range personas {
		if err := os.WriteFile(filepath.Join(dir, "personas", name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if mcpJSON != "" {
		if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(mcpJSON), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestPackValidateMinimal(t *testing.T) {
	dir := writePackFixture(t,
		`{"id":"t","name":"T","version":"0.1.0","personas":["personas/a.persona.md"]}`,
		map[string]string{"a.persona.md": "---\nname: a\ndisplay_name: A\ndescription: desc\n---\nBody.\n"},
		"")
	errs, warns := validatePack(dir)
	if len(errs) != 0 || len(warns) != 0 {
		t.Fatalf("expected clean validation, got errs=%v warns=%v", errs, warns)
	}
}

func TestPackValidateZeroPersonas(t *testing.T) {
	dir := writePackFixture(t, `{"id":"t","name":"T","version":"0.1.0","personas":[]}`, nil, "")
	errs, _ := validatePack(dir)
	if len(errs) == 0 {
		t.Fatal("expected zero-persona error")
	}
}

func TestPackValidateDuplicateNames(t *testing.T) {
	dir := writePackFixture(t,
		`{"id":"t","name":"T","version":"0.1.0","personas":["personas/a.persona.md","personas/b.persona.md"]}`,
		map[string]string{
			"a.persona.md": "---\nname: bot\ndisplay_name: A\ndescription: d\n---\n",
			"b.persona.md": "---\nname: bot\ndisplay_name: B\ndescription: d\n---\n",
		}, "")
	errs, _ := validatePack(dir)
	if len(errs) == 0 {
		t.Fatal("expected duplicate name error")
	}
}

func TestPackValidateUnknownFrontmatterKeyErrors(t *testing.T) {
	dir := writePackFixture(t,
		`{"id":"t","name":"T","version":"0.1.0","personas":["personas/a.persona.md"]}`,
		map[string]string{"a.persona.md": "---\nname: a\ndisplay_name: A\ndescription: d\nzomg_unknown: true\n---\n"}, "")
	errs, _ := validatePack(dir)
	if len(errs) == 0 {
		t.Fatal("expected unknown-key error")
	}
}

func TestPackValidateUnknownManifestKeyWarns(t *testing.T) {
	dir := writePackFixture(t,
		`{"id":"t","name":"T","version":"0.1.0","personas":["personas/a.persona.md"],"totally_made_up":true}`,
		map[string]string{"a.persona.md": "---\nname: a\ndisplay_name: A\ndescription: d\n---\n"}, "")
	errs, warns := validatePack(dir)
	if len(errs) != 0 {
		t.Fatalf("advisory checks should not be errors, got %v", errs)
	}
	if len(warns) == 0 {
		t.Fatal("expected unknown-key warning")
	}
}

func TestPackNameValidation(t *testing.T) {
	if isValidPersonaName("my bot") {
		t.Fatal("spaces should be invalid")
	}
	if !isValidPersonaName("my-bot_2") {
		t.Fatal("expected valid name")
	}
}

func TestSplitFrontmatterEdgeCases(t *testing.T) {
	fm, body, err := splitFrontmatter("---\nname: bot\n---\nBody.\n")
	if err != nil || fm != "name: bot" || body != "Body.\n" {
		t.Fatalf("got fm=%q body=%q err=%v", fm, body, err)
	}

	// "---junk" must not be treated as a closing delimiter.
	_, _, err = splitFrontmatter("---\nname: bot\n---junk\n")
	if err == nil {
		t.Fatal("expected missing-delimiter error")
	}

	// A "---junk" inside a literal block scalar must be skipped, finding
	// the real closing delimiter.
	fm, body, err = splitFrontmatter("---\nname: bot\ndescription: |\n  some text\n  ---junk\n---\nBody here.\n")
	if err != nil {
		t.Fatal(err)
	}
	if body != "Body here.\n" {
		t.Fatalf("got body=%q", body)
	}
	_ = fm

	// No frontmatter at all.
	if _, _, err := splitFrontmatter("Just plain markdown."); err == nil {
		t.Fatal("expected NoFrontmatter error")
	}
}

func TestRuntimeEnvVars(t *testing.T) {
	temp := 0.7
	ctx := uint64(8192)
	vars := runtimeEnvVars("anthropic:claude-sonnet-4-20250514", "", &temp, &ctx)
	want := [][2]string{
		{"GOOSE_PROVIDER", "anthropic"},
		{"GOOSE_MODEL", "claude-sonnet-4-20250514"},
		{"GOOSE_TEMPERATURE", "0.7"},
		{"GOOSE_CONTEXT_LIMIT", "8192"},
	}
	if len(vars) != len(want) {
		t.Fatalf("got %v, want %v", vars, want)
	}
	for i := range want {
		if vars[i] != want[i] {
			t.Errorf("index %d: got %v, want %v", i, vars[i], want[i])
		}
	}
}

func TestRuntimeEnvVarsBuzzAgent(t *testing.T) {
	vars := runtimeEnvVars("databricks:goose-claude-4-6-opus", "buzz-agent", nil, nil)
	want := [][2]string{
		{"BUZZ_AGENT_MODEL", "goose-claude-4-6-opus"},
		{"BUZZ_AGENT_PROVIDER", "databricks"},
	}
	if len(vars) != len(want) {
		t.Fatalf("got %v, want %v", vars, want)
	}
	for i := range want {
		if vars[i] != want[i] {
			t.Errorf("index %d: got %v, want %v", i, vars[i], want[i])
		}
	}
}

func TestRuntimeEnvVarsNoModel(t *testing.T) {
	if vars := runtimeEnvVars("", "", nil, nil); len(vars) != 0 {
		t.Fatalf("expected no vars, got %v", vars)
	}
}

// TestResolvePackFailsFastOnDuplicateNames pins a behavior confirmed
// against the live oracle (/Applications/Buzz.app buzz pack inspect):
// `resolve_pack` (used by `inspect`) hard-fails on the FIRST semantic
// violation, whereas `validate_pack` (used by `validate`) collects it as
// one diagnostic among others and keeps going. loadPack itself must stay
// semantics-free so validatePack's diagnostic collection still works.
func TestResolvePackFailsFastOnDuplicateNames(t *testing.T) {
	dir := writePackFixture(t,
		`{"id":"t","name":"T","version":"0.1.0","personas":["personas/a.persona.md","personas/b.persona.md"]}`,
		map[string]string{
			"a.persona.md": "---\nname: bot\ndisplay_name: A\ndescription: d\n---\n",
			"b.persona.md": "---\nname: bot\ndisplay_name: B\ndescription: d\n---\n",
		}, "")

	if _, err := resolvePack(dir); err == nil {
		t.Fatal("expected resolvePack to fail fast on duplicate names")
	}

	if _, err := loadPack(dir); err != nil {
		t.Fatalf("loadPack must not itself enforce semantics: %v", err)
	}

	errs, _ := validatePack(dir)
	found := false
	for _, e := range errs {
		if strings.Contains(e, "duplicate persona name") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected validatePack to report the duplicate as a diagnostic, got %v", errs)
	}
}

func TestMergedMCPServerCount(t *testing.T) {
	shared := map[string]any{"search": map[string]any{"command": "npx"}}
	personaServers := []any{map[string]any{"name": "other", "command": "npx"}}
	if got := mergedMCPServerCount(shared, personaServers); got != 2 {
		t.Fatalf("got %d, want 2", got)
	}
	// Name collision: still counts once.
	personaServers = []any{map[string]any{"name": "search", "command": "npx"}}
	if got := mergedMCPServerCount(shared, personaServers); got != 1 {
		t.Fatalf("got %d, want 1 (dedup by name)", got)
	}
}
