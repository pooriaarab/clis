package cli

// buzz pack validate/inspect — LOCAL persona pack operations. No relay
// connection. Mirrors the directory layout and checks in
// /Users/parab/code/buzz/crates/buzz-persona/src/{pack,persona,manifest,merge,validate}.rs
// and the CLI at crates/buzz-cli/src/commands/pack.rs.
//
// Directory layout:
//
//	<pack_root>/
//	  .plugin/plugin.json     ← manifest
//	  personas/<name>.persona.md
//	  instructions.md         ← optional
//	  .mcp.json               ← optional
//	  skills/                 ← optional

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	packMaxFrontmatterBytes = 1_048_576
	packMaxBodyBytes        = 262_144
	packMaxPersonaNameLen   = 64
)

var knownManifestKeys = map[string]bool{
	"$schema": true, "id": true, "name": true, "version": true, "description": true,
	"author": true, "license": true, "homepage": true, "repository": true,
	"keywords": true, "engines": true,
	"personas": true, "defaults": true, "pack_instructions": true,
	"hooks_config": true, "mcp_config": true,
}

var knownBehavioralKeys = map[string]bool{
	"subscribe": true, "triggers": true, "respond_to": true, "model": true,
	"temperature": true, "max_context_tokens": true, "thread_replies": true,
	"broadcast_replies": true,
}

var knownRespondToKeys = map[string]bool{"mentions": true, "keywords": true, "all_messages": true}

var knownPersonaKeys = map[string]bool{
	"name": true, "display_name": true, "avatar": true, "description": true,
	"version": true, "author": true, "skills": true, "mcp_servers": true,
	"subscribe": true, "triggers": true, "respond_to": true, "model": true,
	"runtime": true, "temperature": true, "max_context_tokens": true,
	"thread_replies": true, "broadcast_replies": true, "hooks": true,
}

var knownHooksKeys = map[string]bool{"on_start": true, "on_stop": true, "on_message": true}

type packManifest struct {
	ID               string
	Name             string
	Version          string
	Personas         []string
	PackInstructions string
	Defaults         map[string]any
}

type persona struct {
	SourcePath       string
	Name             string
	DisplayName      string
	Description      string
	Avatar           string
	Model            string
	Runtime          string
	Temperature      *float64
	MaxContextTokens *uint64
	Subscribe        []string
	Triggers         respondTo
	ThreadReplies    bool
	BroadcastReplies bool
	Skills           []string
	MCPServerCount   int
	EnvVars          [][2]string
	Prompt           string
}

type respondTo struct {
	Mentions    bool
	Keywords    []string
	AllMessages bool
}

type loadedPack struct {
	Manifest  packManifest
	Personas  []persona
	SkillsDir string
}

func packCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "pack", Short: "Persona pack commands"}

	validate := &cobra.Command{
		Use:   "validate <path>",
		Short: "Validate a persona pack directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			if err := requirePackDir(path); err != nil {
				return err
			}
			errs, warns := validatePack(path)
			for _, e := range errs {
				fmt.Fprintln(opts.stderr(), "  ERROR: "+e)
			}
			for _, w := range warns {
				fmt.Fprintln(opts.stderr(), "  WARN:  "+w)
			}
			if len(errs) > 0 {
				return inputError("Validation failed.")
			}
			if len(warns) > 0 {
				fmt.Fprintln(opts.stdout(), "Valid (with warnings).")
			} else {
				fmt.Fprintln(opts.stdout(), "Valid.")
			}
			return nil
		},
	}

	inspect := &cobra.Command{
		Use:   "inspect <path>",
		Short: "Inspect a persona pack — show metadata and effective config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			if err := requirePackDir(path); err != nil {
				return err
			}
			pack, err := resolvePack(path)
			if err != nil {
				return otherWrap("failed to resolve pack", err)
			}
			printPackInspect(opts, pack)
			return nil
		},
	}

	cmd.AddCommand(validate, inspect)
	return cmd
}

// resolvePack mirrors resolve_pack/resolve_loaded_pack: load, then run the
// same semantic checks validatePack applies as soft diagnostics — but here
// the FIRST violation is a hard failure, matching `pack inspect`'s
// fail-fast behavior (distinct from `pack validate`, which collects every
// diagnostic before deciding pass/fail).
func resolvePack(dir string) (loadedPack, error) {
	pack, err := loadPack(dir)
	if err != nil {
		return loadedPack{}, err
	}
	if err := checkPersonaSemantics(pack); err != nil {
		return loadedPack{}, err
	}
	return pack, nil
}

// checkPersonaSemantics mirrors resolve_loaded_pack: fails fast on the
// first violation, wrapped as "invalid file <source path>: <reason>" per
// PackError::FileParse's Display impl (except the zero-personas check,
// which carries no single offending file).
func checkPersonaSemantics(pack loadedPack) error {
	if len(pack.Personas) == 0 {
		return errors.New("pack contains zero personas")
	}
	seen := map[string]bool{}
	for _, p := range pack.Personas {
		if seen[p.Name] {
			return fmt.Errorf("invalid file %s: duplicate persona name %q", p.SourcePath, p.Name)
		}
		seen[p.Name] = true
		if !isValidPersonaName(p.Name) {
			return fmt.Errorf("invalid file %s: persona name %q contains invalid characters (allowed: [a-zA-Z0-9_-])", p.SourcePath, p.Name)
		}
		if len(p.Name) > packMaxPersonaNameLen {
			return fmt.Errorf("invalid file %s: persona name %q exceeds %d characters (got %d)", p.SourcePath, p.Name, packMaxPersonaNameLen, len(p.Name))
		}
	}
	return nil
}

func requirePackDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return inputError("path does not exist: " + path)
	}
	if !info.IsDir() {
		return inputError("not a directory: " + path)
	}
	return nil
}

// validatePack mirrors validate_pack: load, then semantic + advisory checks.
func validatePack(dir string) (errs, warns []string) {
	pack, err := loadPack(dir)
	if err != nil {
		return []string{"pack failed to load: " + err.Error()}, nil
	}

	if len(pack.Personas) == 0 {
		return []string{"pack contains zero personas"}, nil
	}
	seen := map[string]bool{}
	for _, p := range pack.Personas {
		if seen[p.Name] {
			errs = append(errs, fmt.Sprintf("duplicate persona name %q", p.Name))
		}
		seen[p.Name] = true
		if len(p.Name) > packMaxPersonaNameLen {
			errs = append(errs, fmt.Sprintf("persona name %q exceeds %d characters (got %d)", p.Name, packMaxPersonaNameLen, len(p.Name)))
		}
		if !isValidPersonaName(p.Name) {
			errs = append(errs, fmt.Sprintf("persona name %q contains invalid characters (allowed: [a-zA-Z0-9_-])", p.Name))
		}
	}

	warns = append(warns, advisoryManifestKeyWarnings(dir)...)
	warns = append(warns, advisorySkillNameWarnings(pack)...)
	return errs, warns
}

func isValidPersonaName(name string) bool {
	for _, r := range name {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

// advisoryManifestKeyWarnings re-reads the raw plugin.json for unknown
// top-level / defaults / defaults.triggers keys (typos), mirroring
// advisory_check_manifest_keys.
func advisoryManifestKeyWarnings(dir string) []string {
	var warns []string
	raw, err := os.ReadFile(filepath.Join(dir, ".plugin", "plugin.json"))
	if err != nil {
		return nil
	}
	var top map[string]any
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil
	}
	for _, key := range sortedKeys(top) {
		if !knownManifestKeys[key] {
			warns = append(warns, fmt.Sprintf("plugin.json unknown key %q", key))
		}
	}
	if defaults, ok := top["defaults"].(map[string]any); ok {
		for _, key := range sortedKeys(defaults) {
			if !knownBehavioralKeys[key] {
				warns = append(warns, fmt.Sprintf("plugin.json defaults: unknown key %q", key))
			}
		}
		if triggers, ok := aliasedField(defaults, "triggers", "respond_to"); ok {
			if obj, ok := triggers.(map[string]any); ok {
				for _, key := range sortedKeys(obj) {
					if !knownRespondToKeys[key] {
						warns = append(warns, fmt.Sprintf("plugin.json defaults.triggers: unknown key %q", key))
					}
				}
			}
		}
	}
	return warns
}

// advisorySkillNameWarnings checks that each skill directory's SKILL.md
// `name:` field matches its directory name.
func advisorySkillNameWarnings(pack loadedPack) []string {
	if pack.SkillsDir == "" {
		return nil
	}
	entries, err := os.ReadDir(pack.SkillsDir)
	if err != nil {
		return nil
	}
	var warns []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(pack.SkillsDir, entry.Name(), "SKILL.md"))
		if err != nil {
			continue
		}
		fmStr, _, err := splitFrontmatter(string(content))
		if err != nil {
			continue
		}
		var fm map[string]any
		if err := yaml.Unmarshal([]byte(fmStr), &fm); err != nil {
			continue
		}
		if name, _ := fm["name"].(string); name != "" && name != entry.Name() {
			warns = append(warns, fmt.Sprintf("skill skills/%s: name %q differs from directory name %q", entry.Name(), name, entry.Name()))
		}
	}
	return warns
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// loadPack reads .plugin/plugin.json and every referenced .persona.md file,
// using os.Root to confine all reads within dir (rejects path traversal
// and symlink escapes natively — the Go equivalent of buzz-persona's
// safe_resolve).
func loadPack(dir string) (loadedPack, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return loadedPack{}, err
	}
	defer root.Close()

	manifestRaw, err := readRootFile(root, filepath.Join(".plugin", "plugin.json"))
	if err != nil {
		return loadedPack{}, fmt.Errorf("manifest not found at %s: %w", filepath.Join(dir, ".plugin", "plugin.json"), err)
	}
	manifest, err := parsePackManifest(manifestRaw)
	if err != nil {
		return loadedPack{}, fmt.Errorf("failed to parse manifest: %w", err)
	}

	var sharedMCP map[string]any
	if raw, err := readRootFile(root, ".mcp.json"); err == nil {
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			return loadedPack{}, fmt.Errorf("failed to parse .mcp.json: %w", err)
		}
		sharedMCP, _ = doc["mcpServers"].(map[string]any)
	}

	personas := make([]persona, 0, len(manifest.Personas))
	for _, relPath := range manifest.Personas {
		content, err := readRootFile(root, relPath)
		if err != nil {
			return loadedPack{}, fmt.Errorf("persona file not found: %s", filepath.Join(dir, relPath))
		}
		p, err := parsePersonaFile(string(content), manifest.Defaults, sharedMCP)
		if err != nil {
			return loadedPack{}, fmt.Errorf("invalid file %s: %w", relPath, err)
		}
		p.SourcePath = filepath.Join(dir, relPath)
		personas = append(personas, p)
	}

	skillsDir := ""
	if info, err := os.Stat(filepath.Join(dir, "skills")); err == nil && info.IsDir() {
		skillsDir = filepath.Join(dir, "skills")
	}

	return loadedPack{Manifest: manifest, Personas: personas, SkillsDir: skillsDir}, nil
}

// readRootFile opens and reads relPath through root, rejecting absolute
// paths and any ".." component before root.Open even gets a chance to
// reject a genuine escape (root.Open already refuses those too — this is
// belt-and-suspenders for a clearer error on the common typo).
func readRootFile(root *os.Root, relPath string) ([]byte, error) {
	if strings.HasPrefix(relPath, "/") {
		return nil, fmt.Errorf("path traversal rejected: %s", relPath)
	}
	f, err := root.Open(relPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

// parsePackManifest parses .plugin/plugin.json permissively (unknown keys
// are ignored here — the validator reports them as advisory warnings) and
// enforces the OPS-required id/name/version fields.
func parsePackManifest(raw []byte) (packManifest, error) {
	var fields struct {
		ID               *string        `json:"id"`
		Name             *string        `json:"name"`
		Version          *string        `json:"version"`
		Personas         []string       `json:"personas"`
		PackInstructions string         `json:"pack_instructions"`
		Defaults         map[string]any `json:"defaults"`
	}
	if err := json.Unmarshal(raw, &fields); err != nil {
		return packManifest{}, err
	}
	id, err := requireNonEmpty(fields.ID, "id")
	if err != nil {
		return packManifest{}, err
	}
	name, err := requireNonEmpty(fields.Name, "name")
	if err != nil {
		return packManifest{}, err
	}
	version, err := requireNonEmpty(fields.Version, "version")
	if err != nil {
		return packManifest{}, err
	}
	// Type-check defaults.triggers/respond_to eagerly so a malformed pack
	// fails to load here — mirrors serde's typed BehavioralDefaults
	// deserialize failing before the advisory checks ever run.
	if fields.Defaults != nil {
		if _, err := decodeAliasedRespondTo(fields.Defaults); err != nil {
			return packManifest{}, fmt.Errorf("defaults.triggers: %w", err)
		}
	}
	return packManifest{
		ID: id, Name: name, Version: version,
		Personas:         fields.Personas,
		PackInstructions: fields.PackInstructions,
		Defaults:         fields.Defaults,
	}, nil
}

func requireNonEmpty(value *string, field string) (string, error) {
	if value == nil {
		return "", fmt.Errorf("missing required field: %s", field)
	}
	if strings.TrimSpace(*value) == "" {
		return "", fmt.Errorf("missing required field: %s (empty)", field)
	}
	return *value, nil
}

// parsePersonaFile parses a .persona.md file: YAML frontmatter (strict —
// unknown keys are a hard error, mirroring serde deny_unknown_fields) plus a
// markdown body, then applies pack-level defaults per the 5-level
// precedence model (persona wins, pack defaults fill gaps, null falls
// through, empty array/object is a present override).
func parsePersonaFile(content string, packDefaults, sharedMCP map[string]any) (persona, error) {
	fmStr, body, err := splitFrontmatter(content)
	if err != nil {
		return persona{}, err
	}
	if len(fmStr) > packMaxFrontmatterBytes {
		return persona{}, fmt.Errorf("frontmatter exceeds %d bytes", packMaxFrontmatterBytes)
	}
	if len(body) > packMaxBodyBytes {
		return persona{}, fmt.Errorf("body exceeds %d bytes", packMaxBodyBytes)
	}

	var fm map[string]any
	if err := yaml.Unmarshal([]byte(fmStr), &fm); err != nil {
		return persona{}, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}
	for key := range fm {
		if !knownPersonaKeys[key] {
			return persona{}, fmt.Errorf("failed to parse YAML frontmatter: unknown field %q", key)
		}
	}
	if hooks, ok := fm["hooks"].(map[string]any); ok {
		for key := range hooks {
			if !knownHooksKeys[key] {
				return persona{}, fmt.Errorf("failed to parse YAML frontmatter: unknown field %q in hooks", key)
			}
		}
	}

	name, err := requiredStringField(fm, "name")
	if err != nil {
		return persona{}, err
	}
	displayName, err := requiredStringField(fm, "display_name")
	if err != nil {
		return persona{}, err
	}
	description, err := requiredStringField(fm, "description")
	if err != nil {
		return persona{}, err
	}

	// resolveTriggers is a shallow replacement (merge.rs): a non-null persona
	// triggers/respond_to block wins entirely, else the pack default, else
	// the built-in default (mentions=true) — resolve_triggers in resolve.rs
	// always yields a value, never omitted, so the persona.Triggers field is
	// never nil.
	triggersPtr, err := resolveTriggers(fm, packDefaults)
	if err != nil {
		return persona{}, fmt.Errorf("triggers: %w", err)
	}
	triggers := respondTo{Mentions: true}
	if triggersPtr != nil {
		triggers = *triggersPtr
	}

	merged := mergeBehavioral(fm, packDefaults)
	skills := stringSliceField(fm, "skills")
	model := stringOrEmpty(merged["model"])
	runtime := stringOrEmpty(fm["runtime"])
	temperature := float64Field(merged, "temperature")
	maxContextTokens := uint64PtrField(merged, "max_context_tokens")

	return persona{
		Name:             name,
		DisplayName:      displayName,
		Description:      description,
		Avatar:           stringOrEmpty(fm["avatar"]),
		Model:            model,
		Runtime:          runtime,
		Temperature:      temperature,
		MaxContextTokens: maxContextTokens,
		Subscribe:        resolveSubscribe(fm, packDefaults),
		Triggers:         triggers,
		ThreadReplies:    boolFieldOr(merged, "thread_replies", true),
		BroadcastReplies: boolFieldOr(merged, "broadcast_replies", false),
		Skills:           skills,
		MCPServerCount:   mergedMCPServerCount(sharedMCP, fm["mcp_servers"]),
		EnvVars:          runtimeEnvVars(model, runtime, temperature, maxContextTokens),
		Prompt:           body,
	}, nil
}

// mergedMCPServerCount mirrors merge_mcp_servers: shared .mcp.json servers
// plus per-persona servers, deduped by name (persona wins on collision).
func mergedMCPServerCount(sharedMCP map[string]any, personaServers any) int {
	names := map[string]bool{}
	for name := range sharedMCP {
		names[name] = true
	}
	if servers, ok := personaServers.([]any); ok {
		for _, s := range servers {
			entry, ok := s.(map[string]any)
			if !ok {
				continue
			}
			if name, ok := entry["name"].(string); ok && name != "" {
				names[name] = true
			}
		}
	}
	return len(names)
}

// runtimeEnvVars projects the resolved model/runtime/temperature/
// max_context_tokens into agent-subprocess env vars, mirroring
// resolve.rs::runtime_env_vars exactly (including its GOOSE_*/BUZZ_AGENT_*
// runtime branch and insertion order).
func runtimeEnvVars(model, runtime string, temperature *float64, maxContextTokens *uint64) [][2]string {
	var vars [][2]string
	if model != "" {
		provider, modelID := splitModelString(model)
		if runtime == "buzz-agent" {
			vars = append(vars, [2]string{"BUZZ_AGENT_MODEL", modelID})
			if provider != "" {
				vars = append(vars, [2]string{"BUZZ_AGENT_PROVIDER", provider})
			}
		} else {
			if provider != "" {
				vars = append(vars, [2]string{"GOOSE_PROVIDER", provider})
			}
			vars = append(vars, [2]string{"GOOSE_MODEL", modelID})
		}
	}
	if temperature != nil {
		vars = append(vars, [2]string{"GOOSE_TEMPERATURE", strconv.FormatFloat(*temperature, 'g', -1, 64)})
	}
	if maxContextTokens != nil {
		vars = append(vars, [2]string{"GOOSE_CONTEXT_LIMIT", strconv.FormatUint(*maxContextTokens, 10)})
	}
	return vars
}

// splitModelString mirrors persona::split_model: "provider:model-id" splits
// on the first colon; no colon means no provider.
func splitModelString(model string) (provider, id string) {
	if idx := strings.IndexByte(model, ':'); idx >= 0 {
		return model[:idx], model[idx+1:]
	}
	return "", model
}

func requiredStringField(fm map[string]any, key string) (string, error) {
	raw, ok := fm[key]
	if !ok {
		return "", fmt.Errorf("missing required field: %s", key)
	}
	s, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("field %q must be a string", key)
	}
	if strings.TrimSpace(s) == "" {
		return "", fmt.Errorf("missing required field: %s (empty)", key)
	}
	return s, nil
}

func stringOrEmpty(v any) string {
	s, _ := v.(string)
	return s
}

func stringSliceField(fm map[string]any, key string) []string {
	raw, ok := fm[key].([]any)
	if !ok {
		return nil
	}
	return anySliceToStrings(raw)
}

func boolFieldOr(fm map[string]any, key string, fallback bool) bool {
	if b, ok := fm[key].(bool); ok {
		return b
	}
	return fallback
}

func float64Field(fm map[string]any, key string) *float64 {
	switch v := fm[key].(type) {
	case float64:
		return &v
	case int:
		f := float64(v)
		return &f
	}
	return nil
}

func uint64PtrField(fm map[string]any, key string) *uint64 {
	switch v := fm[key].(type) {
	case int:
		u := uint64(v)
		return &u
	case float64:
		u := uint64(v)
		return &u
	}
	return nil
}

// mergeBehavioral mirrors merge_behavioral_config: pack defaults first,
// non-null persona values override; a persona key absent from defaults
// passes through as-is.
func mergeBehavioral(persona, defaults map[string]any) map[string]any {
	merged := map[string]any{}
	for k, v := range defaults {
		if pv, ok := persona[k]; ok && pv != nil {
			merged[k] = pv
		} else {
			merged[k] = v
		}
	}
	for k, v := range persona {
		if _, exists := merged[k]; !exists && v != nil {
			merged[k] = v
		}
	}
	return merged
}

// resolveSubscribe mirrors the subscribe precedence in merge.rs: a
// non-null persona array wins outright (may be empty); otherwise fall
// through to the pack default array; otherwise nil (absent).
func resolveSubscribe(persona, defaults map[string]any) []string {
	if arr, ok := persona["subscribe"].([]any); ok {
		return anySliceToStrings(arr)
	}
	if persona["subscribe"] == nil {
		if arr, ok := defaults["subscribe"].([]any); ok {
			return anySliceToStrings(arr)
		}
	}
	return nil
}

func anySliceToStrings(arr []any) []string {
	out := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// resolveTriggers mirrors merge.rs's shallow-replacement rule: a
// non-null persona triggers/respond_to block wins entirely (pack default
// ignored); a null/absent persona block falls through to the pack default;
// otherwise nil.
func resolveTriggers(fm, defaults map[string]any) (*respondTo, error) {
	if val, ok := aliasedField(fm, "triggers", "respond_to"); ok && val != nil {
		return decodeRespondTo(val)
	}
	if val, ok := aliasedField(defaults, "triggers", "respond_to"); ok && val != nil {
		return decodeRespondTo(val)
	}
	return nil, nil
}

func aliasedField(m map[string]any, primary, alias string) (any, bool) {
	if m == nil {
		return nil, false
	}
	if v, ok := m[primary]; ok {
		return v, true
	}
	if v, ok := m[alias]; ok {
		return v, true
	}
	return nil, false
}

func decodeAliasedRespondTo(m map[string]any) (*respondTo, error) {
	val, ok := aliasedField(m, "triggers", "respond_to")
	if !ok || val == nil {
		return nil, nil
	}
	return decodeRespondTo(val)
}

// decodeRespondTo type-checks and extracts a triggers/respond_to block.
// Missing sub-fields fall back to the built-in defaults (mentions=true,
// all_messages=false, keywords=[]) per merge.rs::parse_triggers.
func decodeRespondTo(raw any) (*respondTo, error) {
	obj, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected object, got %T", raw)
	}
	rt := &respondTo{Mentions: true, AllMessages: false}
	if v, ok := obj["mentions"]; ok {
		b, ok := v.(bool)
		if !ok {
			return nil, fmt.Errorf("mentions: expected bool, got %T", v)
		}
		rt.Mentions = b
	}
	if v, ok := obj["all_messages"]; ok {
		b, ok := v.(bool)
		if !ok {
			return nil, fmt.Errorf("all_messages: expected bool, got %T", v)
		}
		rt.AllMessages = b
	}
	if v, ok := obj["keywords"]; ok {
		arr, ok := v.([]any)
		if !ok {
			return nil, fmt.Errorf("keywords: expected array, got %T", v)
		}
		for i, item := range arr {
			if _, ok := item.(string); !ok {
				return nil, fmt.Errorf("keywords[%d]: expected string, got %T", i, item)
			}
		}
		rt.Keywords = anySliceToStrings(arr)
	}
	return rt, nil
}

// splitFrontmatter mirrors buzz_persona::persona::split_frontmatter: content
// must start with "---\n" (or "---\r\n") and contain a closing "---" on its
// own line (followed by \n, \r\n, or EOF) — a line like "---junk" is not a
// valid delimiter.
func splitFrontmatter(content string) (fm, body string, err error) {
	rest, ok := strings.CutPrefix(content, "---")
	if !ok {
		return "", "", fmt.Errorf("missing `---` frontmatter delimiters")
	}
	rest = strings.TrimPrefix(rest, "\r")
	rest, ok = strings.CutPrefix(rest, "\n")
	if !ok {
		return "", "", fmt.Errorf("missing `---` frontmatter delimiters")
	}

	searchFrom := 0
	for {
		idx := strings.Index(rest[searchFrom:], "\n---")
		if idx < 0 {
			return "", "", fmt.Errorf("missing `---` frontmatter delimiters")
		}
		pos := idx + searchFrom
		afterDashes := pos + 4
		if afterDashes >= len(rest) {
			return rest[:pos], "", nil
		}
		switch rest[afterDashes] {
		case '\n', '\r':
			after := rest[afterDashes:]
			after = strings.TrimPrefix(after, "\r\n")
			after = strings.TrimPrefix(after, "\n")
			return rest[:pos], after, nil
		default:
			searchFrom = afterDashes
		}
	}
}

func printPackInspect(opts *rootOptions, pack loadedPack) {
	out := opts.stdout()
	fmt.Fprintf(out, "Pack: %s (%s)\n", pack.Manifest.Name, pack.Manifest.ID)
	fmt.Fprintf(out, "Version: %s\n", pack.Manifest.Version)
	fmt.Fprintf(out, "Personas: %d\n\n", len(pack.Personas))
	for _, p := range pack.Personas {
		fmt.Fprintf(out, "  %s\n", p.Name)
		fmt.Fprintf(out, "    Display: %s\n", p.DisplayName)
		fmt.Fprintf(out, "    Description: %s\n", p.Description)
		if p.Model != "" {
			fmt.Fprintf(out, "    Model: %s\n", p.Model)
		}
		if p.Temperature != nil {
			fmt.Fprintf(out, "    Temperature: %s\n", strconv.FormatFloat(*p.Temperature, 'g', -1, 64))
		}
		if p.MaxContextTokens != nil {
			fmt.Fprintf(out, "    Max context tokens: %d\n", *p.MaxContextTokens)
		}
		if len(p.Subscribe) > 0 {
			fmt.Fprintf(out, "    Subscribe: %s\n", strings.Join(p.Subscribe, ", "))
		}
		var parts []string
		if p.Triggers.Mentions {
			parts = append(parts, "mentions")
		}
		if len(p.Triggers.Keywords) > 0 {
			parts = append(parts, fmt.Sprintf("keywords %s", debugStringSlice(p.Triggers.Keywords)))
		}
		if p.Triggers.AllMessages {
			parts = append(parts, "all_messages")
		}
		if len(parts) > 0 {
			fmt.Fprintf(out, "    Triggers: %s\n", strings.Join(parts, " + "))
		}
		fmt.Fprintf(out, "    Thread replies: %t\n", p.ThreadReplies)
		fmt.Fprintf(out, "    Broadcast replies: %t\n", p.BroadcastReplies)
		if p.MCPServerCount > 0 {
			fmt.Fprintf(out, "    MCP servers: %d\n", p.MCPServerCount)
		}
		if len(p.Skills) > 0 {
			fmt.Fprintf(out, "    Skills: %s\n", strings.Join(p.Skills, ", "))
		}
		if p.Avatar != "" {
			fmt.Fprintf(out, "    Avatar: %s\n", p.Avatar)
		}
		prompt := strings.ReplaceAll(p.Prompt, "\n", " ")
		runes := []rune(prompt)
		if len(runes) > 80 {
			prompt = string(runes[:77]) + "..."
		}
		fmt.Fprintf(out, "    System prompt: %d chars (%s)\n", len(p.Prompt), prompt)
		if len(p.EnvVars) > 0 {
			pairs := make([]string, 0, len(p.EnvVars))
			for _, kv := range p.EnvVars {
				pairs = append(pairs, kv[0]+"="+kv[1])
			}
			fmt.Fprintf(out, "    Env vars: %s\n", strings.Join(pairs, ", "))
		}
		fmt.Fprintln(out)
	}
}

// debugStringSlice mirrors Rust's `{:?}` Debug format for Vec<String>:
// `["a", "b"]`.
func debugStringSlice(values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = strconv.Quote(v)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
