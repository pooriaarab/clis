package e2e

import (
	"strings"
	"testing"

	"github.com/pooriaarab/clis/search-console/e2e/fakes"
)

// Ways `google verify` can fail, listed before the code was written:
//  1. Not logged in: the command must stop before it sends a request.
//  2. The domain is a URL, a path, a wildcard or junk: exit 2, no request.
//  3. The token call is refused (API not enabled, bad token): exit 1 with Google's message.
//  4. Google answers 400 while DNS propagates: retry with backoff, then succeed.
//  5. DNS never becomes ready: exit 4, and the message still shows the TXT record.
//  6. Verify answers 403 or 500: do not retry, exit 1.
//  7. Upper case or a trailing dot makes a different domain.
//  8. The access token appears in stdout or stderr.
//  9. --dry-run sends a request.
// 10. --json prints more than one object, or logs progress on stdout.

const txtRecord = "google-site-verification=fake-example.com"

func googleSandbox(t *testing.T) (*Sandbox, *fakes.Google) {
	sb := newSandbox(t)
	g := fakes.NewGoogle(t)
	sb.Alias(g.URL, "http://google.fake")
	sb.Env["GOOGLE_API_BASE"] = g.URL
	sb.Env["GOOGLE_ACCESS_TOKEN"] = "at-test"
	return sb, g
}

func TestGoogleVerifyNotLoggedIn(t *testing.T) {
	sb, g := googleSandbox(t)
	delete(sb.Env, "GOOGLE_ACCESS_TOKEN")
	r := sb.Run("google", "verify", "example.com")
	wantExit(t, r, 1)
	wantContains(t, "stderr", r.Stderr, "auth google")
	if g.Calls("POST /siteVerification/v1/token") != 0 {
		t.Fatal("the CLI called Google without a login")
	}
}

func TestGoogleVerifyRejectsBadDomains(t *testing.T) {
	sb, g := googleSandbox(t)
	for _, d := range []string{"https://example.com", "example.com/path", "*.example.com", "localhost", "exa mple.com", "example.com:8080", ""} {
		r := sb.Run("google", "verify", d)
		wantExit(t, r, 2)
	}
	if g.Calls("POST /siteVerification/v1/token") != 0 {
		t.Fatal("a bad domain reached Google")
	}
}

func TestGoogleVerifyRetriesUntilDNSIsReady(t *testing.T) {
	sb, g := googleSandbox(t)
	g.ReadyAfter = 2
	r := sb.Run("google", "verify", "example.com", "--interval", "20ms")
	wantExit(t, r, 0)
	wantContains(t, "stderr", r.Stderr, txtRecord)
	wantContains(t, "stderr", r.Stderr, "Retrying in")
	wantContains(t, "stdout", r.Stdout, "Verified example.com after 3 attempt(s)")
	if got := g.Calls("POST /siteVerification/v1/webResource"); got != 3 {
		t.Fatalf("verify calls = %d, want 3", got)
	}
	if strings.Contains(r.Stdout+r.Stderr, "at-test") {
		t.Fatal("the access token leaked")
	}
}

func TestGoogleVerifyTimesOutWhenDNSNeverReady(t *testing.T) {
	sb, g := googleSandbox(t)
	g.ReadyAfter = -1
	r := sb.Run("google", "verify", "example.com", "--interval", "20ms", "--wait", "300ms")
	wantExit(t, r, 4)
	wantContains(t, "stderr", r.Stderr, txtRecord)
	wantContains(t, "stderr", r.Stderr, "run the command again")
	if g.Calls("POST /siteVerification/v1/webResource") < 2 {
		t.Fatal("the CLI did not retry")
	}
}

func TestGoogleVerifyDoesNotRetryOtherErrors(t *testing.T) {
	for _, status := range []int{403, 500} {
		sb, g := googleSandbox(t)
		g.VerifyStatus = status
		r := sb.Run("google", "verify", "example.com", "--interval", "20ms")
		wantExit(t, r, 1)
		if got := g.Calls("POST /siteVerification/v1/webResource"); got != 1 {
			t.Fatalf("status %d: verify calls = %d, want 1", status, got)
		}
	}
}

func TestGoogleVerifyTokenRefused(t *testing.T) {
	sb, g := googleSandbox(t)
	g.TokenStatus = 403
	r := sb.Run("google", "verify", "example.com")
	wantExit(t, r, 1)
	wantContains(t, "stderr", r.Stderr, "Site Verification API has not been used")
	sb.Env["GOOGLE_ACCESS_TOKEN"] = "at-wrong"
	wantExit(t, sb.Run("google", "verify", "example.com"), 1)
}

func TestGoogleVerifyNormalizesDomain(t *testing.T) {
	sb, _ := googleSandbox(t)
	r := sb.Run("google", "verify", "  Example.COM.", "--json")
	wantExit(t, r, 0)
	v := r.JSON(t)
	if v["domain"] != "example.com" || v["txt_record"] != txtRecord || v["attempts"] != float64(1) {
		t.Fatalf("unexpected result: %s", r.Stdout)
	}
	if strings.Count(r.Stdout, "Add this TXT") != 0 {
		t.Fatal("progress text is on stdout in --json mode")
	}
}

func TestGoogleVerifyDryRun(t *testing.T) {
	sb, g := googleSandbox(t)
	r := sb.Run("google", "verify", "example.com", "--dry-run", "--json")
	wantExit(t, r, 0)
	v := r.JSON(t)
	if v["dry_run"] != true || len(v["calls"].([]any)) != 2 {
		t.Fatalf("dry run must list the token and verify calls: %s", r.Stdout)
	}
	if g.Calls("POST /siteVerification/v1/token") != 0 || g.Calls("POST /siteVerification/v1/webResource") != 0 {
		t.Fatal("dry run sent a request")
	}
}
