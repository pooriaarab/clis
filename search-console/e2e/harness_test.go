// Package e2e runs the real compiled binary against local fake servers.
//
// No unit tests exist on purpose. Each test lists the ways the feature can fail
// before the code is written, then runs the binary and asserts exit codes and
// output.
package e2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "search-console-e2e-")
	if err != nil {
		panic(err)
	}
	binPath = filepath.Join(dir, "search-console")
	build := exec.Command("go", "build", "-o", binPath, "../cmd/search-console")
	if out, err := build.CombinedOutput(); err != nil {
		os.Stderr.Write(out)
		os.RemoveAll(dir)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// Result is what one run of the binary produced.
type Result struct {
	Stdout, Stderr string
	Code           int
}

// JSON decodes stdout as one JSON object.
func (r Result) JSON(t *testing.T) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal([]byte(r.Stdout), &v); err != nil {
		t.Fatalf("stdout is not one JSON object: %v\n%s", err, r.Stdout)
	}
	return v
}

// Sandbox is one isolated environment: a clean env and its own config dir.
type Sandbox struct {
	t         *testing.T
	Env       map[string]string
	ConfigDir string
	aliases   map[string]string
}

func newSandbox(t *testing.T) *Sandbox {
	t.Helper()
	dir := t.TempDir()
	sb := &Sandbox{t: t, ConfigDir: dir, aliases: map[string]string{dir: "$CONFIG"}}
	sb.Env = map[string]string{
		"PATH":                      os.Getenv("PATH"),
		"HOME":                      dir,
		"SEARCH_CONSOLE_CONFIG_DIR": dir,
	}
	return sb
}

// Alias replaces a fake server URL with a stable name in the saved transcript.
func (sb *Sandbox) Alias(url, name string) { sb.aliases[url] = name }

func (sb *Sandbox) redact(s string) string {
	for from, to := range sb.aliases {
		s = strings.ReplaceAll(s, from, to)
	}
	return s
}

// Run executes the binary and logs a transcript of the call.
func (sb *Sandbox) Run(args ...string) Result {
	sb.t.Helper()
	cmd := exec.Command(binPath, args...)
	for k, v := range sb.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	code := 0
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		code = exit.ExitCode()
	} else if err != nil {
		sb.t.Fatalf("cannot run binary: %v", err)
	}
	r := Result{Stdout: stdout.String(), Stderr: stderr.String(), Code: code}
	sb.t.Logf("$ search-console %s", sb.redact(strings.Join(args, " ")))
	sb.t.Logf("%s", sb.redact(strings.TrimRight(r.Stdout, "\n")))
	if r.Stderr != "" {
		sb.t.Logf("stderr: %s", sb.redact(strings.TrimRight(r.Stderr, "\n")))
	}
	sb.t.Logf("-> exit %d", r.Code)
	return r
}

func wantExit(t *testing.T, r Result, code int) {
	t.Helper()
	if r.Code != code {
		t.Fatalf("exit code = %d, want %d\nstdout: %s\nstderr: %s", r.Code, code, r.Stdout, r.Stderr)
	}
}

func wantContains(t *testing.T, label, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("%s does not contain %q:\n%s", label, want, got)
	}
}
