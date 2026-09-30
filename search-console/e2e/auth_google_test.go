package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pooriaarab/clis/search-console/e2e/fakes"
)

// Ways `auth google` can fail, listed before the code was written:
//  1. No client id or secret: the flow must stop with a message, not open a browser.
//  2. The user denies consent (error=access_denied) and the CLI waits forever.
//  3. The callback state differs from ours (CSRF) and the CLI accepts the code.
//  4. Google returns no refresh token, and the CLI saves an unusable login.
//  5. Nobody completes the consent and the CLI never exits (exit 4 on timeout).
//  6. The saved file is readable by other users (must be 0600, dir 0700).
//  7. The token or secret leaks to stdout or stderr.
//  8. --dry-run opens a browser or sends a request (it is refused, exit 2).
//  9. The PKCE verifier or redirect URI is wrong, so Google refuses the code.

func authSandbox(t *testing.T) (*Sandbox, *fakes.OAuth) {
	sb := newSandbox(t)
	oauth := fakes.NewOAuth(t)
	sb.Alias(oauth.URL, "http://google.fake")
	sb.Env["GOOGLE_OAUTH_AUTH_URL"] = oauth.URL + "/auth"
	sb.Env["GOOGLE_OAUTH_TOKEN_URL"] = oauth.URL + "/token"
	sb.Browser()
	return sb, oauth
}

func login(sb *Sandbox, extra ...string) Result {
	args := append([]string{"auth", "google", "--client-id", "cid", "--client-secret", "csecret", "--timeout", "10s"}, extra...)
	return sb.Run(args...)
}

func TestAuthGoogleLoginStoresRefreshToken(t *testing.T) {
	sb, _ := authSandbox(t)
	sb.Env["SEARCH_CONSOLE_CONFIG_DIR"] = filepath.Join(sb.ConfigDir, "nested", "search-console")
	r := login(sb, "--json")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, `"ok": true`)
	dir := sb.Env["SEARCH_CONSOLE_CONFIG_DIR"]
	file, err := os.Stat(filepath.Join(dir, "google.json"))
	d, _ := os.Stat(dir)
	if err != nil || file.Mode().Perm() != 0o600 || d.Mode().Perm() != 0o700 {
		t.Fatalf("modes: file %v dir %v (err %v), want 0600 and 0700", file, d, err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "google.json"))
	wantContains(t, "google.json", string(data), "rt-valid")
	for _, secret := range []string{"rt-valid", "csecret", "at-login"} {
		if strings.Contains(r.Stdout+r.Stderr, secret) {
			t.Fatalf("output leaks %q", secret)
		}
	}
}

func TestAuthGoogleMissingClient(t *testing.T) {
	sb, _ := authSandbox(t)
	r := sb.Run("auth", "google")
	wantExit(t, r, 1)
	wantContains(t, "stderr", r.Stderr, "GOOGLE_CLIENT_ID")
}

func TestAuthGoogleFailedConsent(t *testing.T) {
	cases := []struct{ mode, want string }{
		{"deny", "access_denied"},
		{"badstate", "state"},
		{"norefresh", "refresh token"},
	}
	for _, c := range cases {
		t.Run(c.mode, func(t *testing.T) {
			sb, oauth := authSandbox(t)
			oauth.Mode = c.mode
			r := login(sb)
			wantExit(t, r, 1)
			wantContains(t, "stderr", r.Stderr, c.want)
			if c.mode == "deny" || c.mode == "badstate" {
				if oauth.TokenHits() != 0 {
					t.Fatal("the CLI exchanged a code it should have refused")
				}
			}
			if _, err := os.Stat(filepath.Join(sb.ConfigDir, "google.json")); err == nil {
				t.Fatal("config file was saved after a failed login")
			}
		})
	}
}

func TestAuthGoogleTimeout(t *testing.T) {
	sb, _ := authSandbox(t)
	sb.Env["BROWSER"] = "true" // a browser that never completes the consent
	r := login(sb, "--timeout", "1s")
	wantExit(t, r, 4)
	wantContains(t, "stderr", r.Stderr, "timed out")
}

func TestAuthGoogleRejectsDryRun(t *testing.T) {
	sb, oauth := authSandbox(t)
	r := login(sb, "--dry-run")
	wantExit(t, r, 2)
	if oauth.TokenHits() != 0 {
		t.Fatal("dry run sent a request")
	}
}
