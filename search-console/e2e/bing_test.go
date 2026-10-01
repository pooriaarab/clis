package e2e

import (
	"strings"
	"testing"

	"github.com/pooriaarab/clis/search-console/e2e/fakes"
)

// Ways `auth bing` and `bing quota` can fail, listed before the code was written:
//  1. BING_WEBMASTER_API_KEY is not set: stop with the env name, send no request.
//  2. Bing rejects the key (HTTP 400 InvalidApiKey) and the CLI reports success.
//  3. Bing sends the same error as HTTP 200 with an ErrorCode body, and the CLI reads it as data.
//  4. The API key leaks in stdout, stderr, the dry-run calls or a transport error.
//  5. The domain is not a site of the account: show Bing's message, exit 1.
//  6. The quota is not shown as numbers.
//  7. --dry-run sends a request.
//  8. `auth status` shows the key or hides that it is set.
//  9. Bing repeats the request URL in its error text, so the key reaches the screen.
// 10. A long or HTML error body floods the terminal.
// 11. The fake answers a shape the live API never sends. The live shapes, seen with
//     read-only calls: a bad key is HTTP 400 code 3 "InvalidApiKey"; a site that is not
//     in the account is HTTP 400 code 14 "NotAuthorized"; GetUserSites sends
//     DnsVerificationCode as the whole record name "<code>.<domain>".

const bingKey = "bing-key"

func bingSandbox(t *testing.T) (*Sandbox, *fakes.Bing) {
	sb := newSandbox(t)
	b := fakes.NewBing(t)
	sb.Alias(b.URL, "http://bing.fake")
	sb.Env["BING_API_BASE"] = b.URL
	sb.Env["BING_WEBMASTER_API_KEY"] = bingKey
	return sb, b
}

func noLeak(t *testing.T, r Result, secret string) {
	t.Helper()
	if strings.Contains(r.Stdout+r.Stderr, secret) {
		t.Fatalf("output leaks %q:\n%s%s", secret, r.Stdout, r.Stderr)
	}
}

func TestAuthBingChecksTheKey(t *testing.T) {
	sb, b := bingSandbox(t)
	b.AddSite("https://example.com/", fakes.BingSite{})
	r := sb.Run("auth", "bing", "--json")
	wantExit(t, r, 0)
	if v := r.JSON(t); v["ok"] != true || v["sites"] != float64(1) {
		t.Fatalf("unexpected result: %s", r.Stdout)
	}
	noLeak(t, r, bingKey)
}

func TestAuthBingKeyMissing(t *testing.T) {
	sb, b := bingSandbox(t)
	delete(sb.Env, "BING_WEBMASTER_API_KEY")
	r := sb.Run("auth", "bing")
	wantExit(t, r, 1)
	wantContains(t, "stderr", r.Stderr, "BING_WEBMASTER_API_KEY")
	if b.Calls("GetUserSites") != 0 {
		t.Fatal("the CLI called Bing without a key")
	}
}

func TestBingRejectsKeyInBothErrorShapes(t *testing.T) {
	for _, as200 := range []bool{false, true} {
		sb, b := bingSandbox(t)
		b.Configure(func(f *fakes.Bing) { f.ErrorsAs200 = as200 })
		sb.Env["BING_WEBMASTER_API_KEY"] = "wrong-key"
		for _, args := range [][]string{{"auth", "bing"}, {"bing", "quota", "example.com"}} {
			r := sb.Run(args...)
			wantExit(t, r, 1)
			wantContains(t, "stderr", r.Stderr, "InvalidApiKey")
			noLeak(t, r, "wrong-key")
		}
	}
}

func TestBingErrorTextHidesKeyAndIsShort(t *testing.T) {
	for _, mode := range []string{"echo", "long", "html", "all"} {
		for _, key := range []string{bingKey, "wrong-key"} {
			sb, b := bingSandbox(t)
			sb.Env["BING_WEBMASTER_API_KEY"] = key
			b.Configure(func(f *fakes.Bing) {
				f.EchoRequest = mode == "echo" || mode == "all"
				f.LongError = map[bool]int{true: 5000}[mode == "long" || mode == "all"]
				f.RawErrors = mode == "html" || mode == "all"
			})
			cmds := [][]string{{"bing", "quota", "example.com"}} // the site is not in the account
			if key != bingKey {
				cmds = append(cmds, []string{"auth", "bing"})
			}
			for _, args := range cmds {
				r := sb.Run(args...)
				wantExit(t, r, 1)
				noLeak(t, r, key)
				if len(r.Stderr) > 600 {
					t.Fatalf("%s: the error is %d bytes, want at most 600:\n%s", mode, len(r.Stderr), r.Stderr)
				}
			}
		}
	}
}

func TestBingTransportErrorHidesKey(t *testing.T) {
	sb, b := bingSandbox(t)
	b.Close() // the address now refuses connections
	r := sb.Run("auth", "bing")
	wantExit(t, r, 1)
	noLeak(t, r, bingKey)
}

func TestBingQuota(t *testing.T) {
	sb, b := bingSandbox(t)
	b.AddSite("https://example.com/", fakes.BingSite{})
	b.Configure(func(f *fakes.Bing) { f.Daily, f.Monthly = 9990, 299000 })
	r := sb.Run("bing", "quota", "example.com", "--json")
	wantExit(t, r, 0)
	if v := r.JSON(t); v["daily"] != float64(9990) || v["monthly"] != float64(299000) {
		t.Fatalf("unexpected quota: %s", r.Stdout)
	}
	r = sb.Run("bing", "quota", "example.com")
	wantContains(t, "stdout", r.Stdout, "9990 URLs today and 299000 this month")
}

func TestBingQuotaUnknownSite(t *testing.T) {
	sb, _ := bingSandbox(t)
	r := sb.Run("bing", "quota", "example.com")
	wantExit(t, r, 1)
	wantContains(t, "stderr", r.Stderr, "NotAuthorized")
	wantExit(t, sb.Run("bing", "quota", "https://example.com"), 2)
}

func TestBingDryRunHidesKey(t *testing.T) {
	sb, b := bingSandbox(t)
	r := sb.Run("bing", "quota", "example.com", "--dry-run", "--json")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "apikey=REDACTED")
	noLeak(t, r, bingKey)
	if b.Calls("GetUrlSubmissionQuota") != 0 {
		t.Fatal("dry run sent a request")
	}
}

func TestAuthStatusShowsBing(t *testing.T) {
	sb, _ := bingSandbox(t)
	r := sb.Run("auth", "status")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "bing: configured (env:BING_WEBMASTER_API_KEY)")
	noLeak(t, r, bingKey)
	delete(sb.Env, "BING_WEBMASTER_API_KEY")
	wantContains(t, "stdout", sb.Run("auth", "status").Stdout, "bing: not configured")
}
