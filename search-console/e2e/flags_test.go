package e2e

import (
	"strings"
	"testing"
)

// Ways the timing flags can fail, listed before the code was written:
//  1. --interval 0 or a negative value is accepted, and the retry loop spins with no delay.
//  2. A negative --wait is accepted and the command gives up at once, as if it timed out (exit 4).
//  3. --timeout 0 or a negative value is accepted, so `auth google` times out before the user can act.
//  4. The check runs after a login or a request, so a bad flag still reaches the network or reports
//     "not logged in" (exit 1) instead of a usage error.
//  5. The message does not name the flag or the value.
//  6. One command validates and a sibling with the same flag does not (google verify, google sitemap
//     submit, bing verify, launch, auth google).
//
// The sandbox has no login and no fake server, so a command that reaches the network or the
// credential check exits 1, not 2. That proves the check runs first.

func TestTimingFlagsAreValidated(t *testing.T) {
	commands := map[string][]string{
		"google verify":        {"google", "verify", "example.com"},
		"google sitemap":       {"google", "sitemap", "submit", "example.com", "https://example.com/sitemap.xml"},
		"bing verify":          {"bing", "verify", "example.com"},
		"launch":               {"launch", "example.com", "--sitemap", "https://example.com/sitemap.xml"},
		"launch --dry-run":     {"launch", "example.com", "--sitemap", "https://example.com/sitemap.xml", "--dry-run"},
		"google verify --json": {"google", "verify", "example.com", "--json"},
	}
	flags := []struct{ name, value, want string }{
		{"--interval", "0", "--interval must be greater than 0"},
		{"--interval", "-5s", "--interval must be greater than 0"},
		{"--wait", "-1s", "--wait must not be negative"},
	}
	for name, base := range commands {
		for _, f := range flags {
			t.Run(name+" "+f.name+"="+f.value, func(t *testing.T) {
				sb := newSandbox(t)
				args := append(append([]string{}, base...), f.name+"="+f.value)
				r := sb.Run(args...)
				wantExit(t, r, 2)
				wantContains(t, "output", r.Stderr+r.Stdout, f.want)
				wantContains(t, "output", r.Stderr+r.Stdout, f.value)
			})
		}
	}
}

func TestAuthGoogleRejectsBadTimeout(t *testing.T) {
	for _, v := range []string{"0", "-1s"} {
		t.Run(v, func(t *testing.T) {
			sb, oauth := authSandbox(t)
			r := sb.Run("auth", "google", "--client-id", "cid", "--client-secret", "csecret", "--timeout="+v)
			wantExit(t, r, 2)
			wantContains(t, "stderr", r.Stderr, "--timeout must be greater than 0")
			if oauth.TokenHits() != 0 || strings.Contains(r.Stderr, "Open this URL") {
				t.Fatal("the login started before the flag was checked")
			}
		})
	}
}

func TestZeroWaitIsAllowed(t *testing.T) {
	// --wait 0 means one try and no waiting. It is not a usage error.
	sb := newSandbox(t)
	r := sb.Run("google", "verify", "example.com", "--wait=0")
	wantExit(t, r, 1) // not logged in: the flag passed, the credential check failed
	wantContains(t, "stderr", r.Stderr, "not logged in")
}
