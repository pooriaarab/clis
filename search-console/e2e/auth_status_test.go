package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Ways token refresh and `auth status` can fail, listed before the code was written:
//  1. The refresh token is expired or revoked and the error does not say how to fix it.
//  2. A saved login exists and the env is ignored (env must win, one value at a time).
//  3. An access token in the env still triggers a token call.
//  4. Nothing is configured and --check exits 0.
//  5. google.json is not valid JSON and the CLI crashes or ignores it.
//  6. The login is partial (a refresh token with no client secret) and the CLI sends it anyway.
//  7. Status or the dry-run output prints a token or a secret.

func status(sb *Sandbox, extra ...string) Result {
	return sb.Run(append([]string{"auth", "status", "--check", "--json"}, extra...)...)
}

func googleStatus(t *testing.T, r Result) map[string]any {
	t.Helper()
	return r.JSON(t)["google"].(map[string]any)
}

func TestAuthStatusCheckRefreshesSavedLogin(t *testing.T) {
	sb, oauth := authSandbox(t)
	wantExit(t, login(sb), 0)
	r := status(sb)
	wantExit(t, r, 0)
	if g := googleStatus(t, r); g["source"] != "file" || g["check"] != "ok" {
		t.Fatalf("unexpected status: %v", g)
	}
	if oauth.TokenHits() != 2 {
		t.Fatalf("token calls = %d, want 2 (login and refresh)", oauth.TokenHits())
	}
	for _, secret := range []string{"rt-valid", "csecret", "at-refreshed"} {
		if strings.Contains(r.Stdout+r.Stderr, secret) {
			t.Fatalf("status leaks %q", secret)
		}
	}
}

func TestAuthExpiredRefreshTokenExplainsFix(t *testing.T) {
	sb, _ := authSandbox(t)
	sb.Env["GOOGLE_CLIENT_ID"], sb.Env["GOOGLE_CLIENT_SECRET"] = "cid", "csecret"
	sb.Env["GOOGLE_REFRESH_TOKEN"] = "rt-expired"
	r := sb.Run("auth", "status", "--check")
	wantExit(t, r, 1)
	wantContains(t, "stderr", r.Stderr, "In production")
}

func TestAuthEnvWinsOverSavedLogin(t *testing.T) {
	sb, oauth := authSandbox(t)
	wantExit(t, login(sb), 0)
	before := oauth.TokenHits()

	sb.Env["GOOGLE_ACCESS_TOKEN"] = "at-env"
	r := status(sb)
	wantExit(t, r, 0)
	if g := googleStatus(t, r); g["source"] != "env:GOOGLE_ACCESS_TOKEN" {
		t.Fatalf("access token env was ignored: %v", g)
	}
	if oauth.TokenHits() != before {
		t.Fatal("an access token in the env must not call the token endpoint")
	}

	delete(sb.Env, "GOOGLE_ACCESS_TOKEN")
	sb.Env["GOOGLE_REFRESH_TOKEN"] = "rt-env"
	r = status(sb)
	wantExit(t, r, 0)
	if g := googleStatus(t, r); g["source"] != "env:GOOGLE_REFRESH_TOKEN" || g["check"] != "ok" {
		t.Fatalf("refresh token env was ignored: %v", g)
	}
}

func TestAuthStatusNotConfigured(t *testing.T) {
	sb, _ := authSandbox(t)
	r := sb.Run("auth", "status")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "not configured")
	r = sb.Run("auth", "status", "--check")
	wantExit(t, r, 1)
	wantContains(t, "stderr", r.Stderr, "auth google")
}

func TestAuthStatusBadConfigFile(t *testing.T) {
	sb, _ := authSandbox(t)
	if err := os.WriteFile(filepath.Join(sb.ConfigDir, "google.json"), []byte("{oops"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := sb.Run("auth", "status")
	wantExit(t, r, 1)
	wantContains(t, "stderr", r.Stderr, "google.json is not valid JSON")
}

func TestAuthStatusPartialLogin(t *testing.T) {
	sb, oauth := authSandbox(t)
	sb.Env["GOOGLE_REFRESH_TOKEN"] = "rt-env"
	r := sb.Run("auth", "status", "--check")
	wantExit(t, r, 1)
	wantContains(t, "stderr", r.Stderr, "GOOGLE_CLIENT_SECRET")
	if oauth.TokenHits() != 0 {
		t.Fatal("the CLI sent a refresh without a client secret")
	}
}

func TestAuthStatusDryRunHidesSecrets(t *testing.T) {
	sb, oauth := authSandbox(t)
	sb.Env["GOOGLE_CLIENT_ID"], sb.Env["GOOGLE_CLIENT_SECRET"] = "cid", "csecret"
	sb.Env["GOOGLE_REFRESH_TOKEN"] = "rt-env"
	r := status(sb, "--dry-run")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, `"dry_run": true`)
	for _, secret := range []string{"csecret", "rt-env"} {
		if strings.Contains(r.Stdout+r.Stderr, secret) {
			t.Fatalf("dry run leaks %q", secret)
		}
	}
	if oauth.TokenHits() != 0 {
		t.Fatal("dry run sent a request")
	}
}
