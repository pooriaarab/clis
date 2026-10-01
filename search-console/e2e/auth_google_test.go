package e2e

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
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
// 10. A reply with no refresh token, or an empty one, overwrites or adds a file under
//     the default ~/.config/search-console, not only under SEARCH_CONSOLE_CONFIG_DIR.
// 11. A login with a wrong client id is accepted by the fake, so the CLI never sees invalid_client.
// 12. A refresh with a wrong client id or secret is accepted by the fake.

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
			oauth.Configure(func(f *fakes.OAuth) { f.Mode = c.mode })
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

// treeState lists every file under root with its mode and content.
func treeState(t *testing.T, root string) map[string]string {
	t.Helper()
	state := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		content := ""
		if d.Type().IsRegular() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			content = string(data)
		}
		state[path] = info.Mode().String() + " " + content
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func TestAuthGoogleNoRefreshTokenWritesNothing(t *testing.T) {
	for _, mode := range []string{"norefresh", "emptyrefresh"} {
		for _, existing := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/existing=%t", mode, existing), func(t *testing.T) {
				sb, oauth := authSandbox(t)
				delete(sb.Env, "SEARCH_CONSOLE_CONFIG_DIR") // use the default path under the temp HOME
				oauth.Configure(func(f *fakes.OAuth) { f.Mode = mode })
				if existing {
					dir := filepath.Join(sb.Env["HOME"], ".config", "search-console")
					if err := os.MkdirAll(dir, 0o700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(dir, "google.json"), []byte(`{"refresh_token":"rt-keep"}`), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				before := treeState(t, sb.Env["HOME"])
				r := login(sb)
				wantExit(t, r, 1)
				wantContains(t, "stderr", r.Stderr, "refresh token")
				after := treeState(t, sb.Env["HOME"])
				if !reflect.DeepEqual(before, after) {
					t.Fatalf("a failed login changed the home directory\nbefore: %v\nafter:  %v", before, after)
				}
			})
		}
	}
}

func TestAuthGoogleRejectsWrongClientID(t *testing.T) {
	sb, _ := authSandbox(t)
	r := sb.Run("auth", "google", "--client-id", "other", "--client-secret", "csecret", "--timeout", "10s")
	wantExit(t, r, 1)
	wantContains(t, "stderr", r.Stderr, "invalid_client")
	if _, err := os.Stat(filepath.Join(sb.ConfigDir, "google.json")); err == nil {
		t.Fatal("config file was saved after a refused client")
	}
}

func TestAuthRefreshRejectsWrongClient(t *testing.T) {
	for _, c := range []struct{ id, secret string }{{"cid", "wrong"}, {"wrong", "csecret"}} {
		t.Run(c.id+"/"+c.secret, func(t *testing.T) {
			sb, _ := authSandbox(t)
			sb.Env["GOOGLE_CLIENT_ID"], sb.Env["GOOGLE_CLIENT_SECRET"] = c.id, c.secret
			sb.Env["GOOGLE_REFRESH_TOKEN"] = "rt-env"
			r := sb.Run("auth", "status", "--check")
			wantExit(t, r, 1)
			wantContains(t, "stderr", r.Stderr, "invalid_client")
		})
	}
}
