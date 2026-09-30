package e2e

import "testing"

// Ways the scaffold can fail, listed before the code was written:
//  1. The binary does not start, or `version` prints nothing.
//  2. An unknown command exits 0, so a typo in CI passes.
//  3. A usage error exits 1, which hides it from real failures (exit 2).
//  4. With --json, an error prints prose, so an agent cannot parse it.
//  5. With --json, the error also leaks to stderr and breaks `2>&1` parsing.

func TestVersion(t *testing.T) {
	sb := newSandbox(t)
	r := sb.Run("version")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "search-console ")

	r = sb.Run("version", "--json")
	wantExit(t, r, 0)
	if r.JSON(t)["version"] == "" {
		t.Fatal("json version is empty")
	}
}

func TestUnknownCommandIsUsageError(t *testing.T) {
	sb := newSandbox(t)
	r := sb.Run("frobnicate")
	wantExit(t, r, 2)
	wantContains(t, "stderr", r.Stderr, "unknown command")
}

func TestJSONErrorIsOneObjectOnStdout(t *testing.T) {
	sb := newSandbox(t)
	r := sb.Run("--json", "frobnicate")
	wantExit(t, r, 2)
	v := r.JSON(t)
	if v["ok"] != false || v["error"] == "" {
		t.Fatalf("bad error object: %v", v)
	}
	if r.Stderr != "" {
		t.Fatalf("stderr must be empty in json mode, got %q", r.Stderr)
	}
}

func TestHelpDoesNotExitNonZero(t *testing.T) {
	sb := newSandbox(t)
	r := sb.Run("--help")
	wantExit(t, r, 0)
	wantContains(t, "help", r.Stdout, "--json")
}
