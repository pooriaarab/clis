package e2e

import (
	"testing"

	"github.com/pooriaarab/clis/search-console/e2e/fakes"
)

// Ways `bing verify` can fail, listed before the code was written:
//  1. The Bing key or the Cloudflare token is missing: stop before any request.
//  2. The site is already verified: no AddSite, no DNS change, exit 0 on every rerun.
//  3. The site is added but not verified, and the CLI calls AddSite again (Bing refuses).
//  4. The CNAME is created proxied, so Bing sees Cloudflare and never verifies.
//  5. An old proxied CNAME exists and the CLI leaves it proxied.
//  6. Bing answers false until DNS is ready: retry with backoff, then succeed.
//  7. DNS never becomes ready: exit 4, and the message names the CNAME.
//  8. AddSite or VerifySite returns an error (HTTP 400 or 200 body): exit 1, no retry.
//  9. Without --cloudflare-zone the record to add is not printed.
// 10. The key or the token leaks, or --dry-run sends a request.
// 11. Bing answers error 81058 (already present) and the CLI fails. It is success.
// 12. The account lists the site as http://, in other case, or without the slash, the
//     lookup misses it, and AddSite runs for a site that is already there.
// 13. Bing sends the CNAME name as "<code>.<domain>", and the CLI appends the domain again.

const bingCode = "0123456789abcdef0123456789abcdef"

func bingDNSSandbox(t *testing.T) (*Sandbox, *fakes.Bing, *fakes.Cloudflare) {
	sb, b := bingSandbox(t)
	cf := fakes.NewCloudflare(t)
	sb.Alias(cf.URL, "http://cloudflare.fake")
	sb.Env["CLOUDFLARE_API_BASE"] = cf.URL
	sb.Env["CLOUDFLARE_API_TOKEN"] = "cf-test"
	cf.AddZone("example.com", "zone-1")
	b.Configure(func(f *fakes.Bing) { f.Ready = cf.HasCNAME }) // Bing sees the CNAME only when it is in DNS and not proxied
	return sb, b, cf
}

func TestBingVerifyCreatesUnproxiedCNAME(t *testing.T) {
	sb, b, cf := bingDNSSandbox(t)
	r := sb.Run("bing", "verify", "example.com", "--cloudflare-zone", "auto", "--interval", "20ms", "--json")
	wantExit(t, r, 0)
	if !cf.HasCNAME(bingCode+".example.com", "verify.bing.com") || !b.Verified("https://example.com/") {
		t.Fatal("the CNAME is missing, proxied, or the site is not verified")
	}
	if v := r.JSON(t); v["attempts"] != float64(1) || v["cloudflare_changed"] != true {
		t.Fatalf("unexpected result: %s", r.Stdout)
	}
	noLeak(t, r, bingKey)
	noLeak(t, r, "cf-test")
}

func TestBingVerifyTreats81058AsAlreadyPresent(t *testing.T) {
	for _, as200 := range []bool{false, true} {
		sb, b, cf := bingDNSSandbox(t)
		b.ErrorsAs200 = as200
		b.AddSite("https://example.com/", fakes.BingSite{DNSCode: bingCode})
		b.HideSitesCalls = 1 // the first listing lags: the CLI sees no site and calls AddSite
		r := sb.Run("bing", "verify", "example.com", "--cloudflare-zone", "auto", "--interval", "20ms", "--json")
		wantExit(t, r, 0)
		if r.JSON(t)["already_present"] != true || !b.Verified("https://example.com/") || !cf.HasCNAME(bingCode+".example.com", "verify.bing.com") {
			t.Fatalf("as200=%t: unexpected result: %s", as200, r.Stdout)
		}
		if b.Calls("AddSite") != 1 {
			t.Fatalf("AddSite calls = %d, want 1", b.Calls("AddSite"))
		}
	}
}

func TestBingVerifyFindsTheSiteWhateverItsURLLooksLike(t *testing.T) {
	for _, held := range []string{"http://example.com/", "https://Example.COM/", "https://example.com", "HTTP://EXAMPLE.COM"} {
		sb, b, _ := bingDNSSandbox(t)
		b.AddSite(held, fakes.BingSite{Verified: true, DNSCode: bingCode})
		r := sb.Run("bing", "verify", "example.com", "--json")
		wantExit(t, r, 0)
		if r.JSON(t)["already_verified"] != true || b.Calls("AddSite") != 0 {
			t.Fatalf("held as %q: not found: %s", held, r.Stdout)
		}
	}
}

func TestBingVerifyFixesProxiedCNAME(t *testing.T) {
	sb, b, cf := bingDNSSandbox(t)
	b.AddSite("https://example.com/", fakes.BingSite{DNSCode: bingCode})
	cf.SeedCNAME("zone-1", bingCode+".example.com", "verify.bing.com", true)
	wantExit(t, sb.Run("bing", "verify", "example.com", "--cloudflare-zone", "auto", "--interval", "20ms"), 0)
	if !cf.HasCNAME(bingCode+".example.com", "verify.bing.com") {
		t.Fatal("the CNAME is still proxied")
	}
	if b.Calls("AddSite") != 0 {
		t.Fatal("AddSite ran for a site that was already added")
	}
	if len(cf.Records("zone-1")) != 1 {
		t.Fatal("the CLI made a second record")
	}
}

func TestBingVerifyIsIdempotent(t *testing.T) {
	sb, b, cf := bingDNSSandbox(t)
	b.AddSite("https://example.com/", fakes.BingSite{Verified: true, DNSCode: bingCode})
	for i := 0; i < 2; i++ {
		r := sb.Run("bing", "verify", "example.com", "--cloudflare-zone", "auto", "--json")
		wantExit(t, r, 0)
		if r.JSON(t)["already_verified"] != true {
			t.Fatalf("not reported as verified: %s", r.Stdout)
		}
	}
	if b.Calls("AddSite")+b.Calls("VerifySite") != 0 || len(cf.Records("zone-1")) != 0 {
		t.Fatal("a verified site caused a change")
	}
}

func TestBingVerifyRetriesUntilDNSIsReady(t *testing.T) {
	sb, b, cf := bingDNSSandbox(t)
	n := 0
	b.Configure(func(f *fakes.Bing) {
		f.Ready = func(name, target string) bool { n++; return n > 2 && cf.HasCNAME(name, target) }
	})
	r := sb.Run("bing", "verify", "example.com", "--cloudflare-zone", "auto", "--interval", "20ms")
	wantExit(t, r, 0)
	wantContains(t, "stderr", r.Stderr, "Retrying in")
	wantContains(t, "stdout", r.Stdout, "after 3 attempt(s)")
}

func TestBingVerifyTimesOut(t *testing.T) {
	sb, b, _ := bingDNSSandbox(t)
	b.Configure(func(f *fakes.Bing) { f.Ready = func(string, string) bool { return false } })
	r := sb.Run("bing", "verify", "example.com", "--cloudflare-zone", "auto", "--interval", "20ms", "--wait", "300ms")
	wantExit(t, r, 4)
	wantContains(t, "stderr", r.Stderr, bingCode+".example.com")
	wantContains(t, "stderr", r.Stderr, "run the command again")
}

func TestBingVerifyErrorsAreNotRetried(t *testing.T) {
	for _, as200 := range []bool{false, true} {
		sb, b, _ := bingDNSSandbox(t)
		b.Configure(func(f *fakes.Bing) { f.ErrorsAs200 = as200 })
		b.Configure(func(f *fakes.Bing) { f.VerifyError = "verification refused" })
		r := sb.Run("bing", "verify", "example.com", "--cloudflare-zone", "auto", "--interval", "20ms")
		wantExit(t, r, 1)
		wantContains(t, "stderr", r.Stderr, "verification refused")
		if b.Calls("VerifySite") != 1 {
			t.Fatalf("VerifySite calls = %d, want 1", b.Calls("VerifySite"))
		}
	}
}

func TestBingVerifyPrintsRecordWithoutCloudflare(t *testing.T) {
	sb, b, _ := bingDNSSandbox(t)
	b.Configure(func(f *fakes.Bing) { f.Ready = func(string, string) bool { return false } })
	r := sb.Run("bing", "verify", "example.com", "--interval", "20ms", "--wait", "100ms")
	wantExit(t, r, 4)
	wantContains(t, "stderr", r.Stderr, bingCode+".example.com -> verify.bing.com")
	wantContains(t, "stderr", r.Stderr, "not proxied")
}

func TestBingVerifyMissingSecrets(t *testing.T) {
	sb, b, cf := bingDNSSandbox(t)
	delete(sb.Env, "CLOUDFLARE_API_TOKEN")
	r := sb.Run("bing", "verify", "example.com", "--cloudflare-zone", "auto")
	wantExit(t, r, 1)
	wantContains(t, "stderr", r.Stderr, "CLOUDFLARE_API_TOKEN")
	delete(sb.Env, "BING_WEBMASTER_API_KEY")
	wantExit(t, sb.Run("bing", "verify", "example.com"), 1)
	if b.Calls("GetUserSites") != 0 || len(cf.Records("zone-1")) != 0 {
		t.Fatal("a request was sent without credentials")
	}
}

func TestBingVerifyDryRun(t *testing.T) {
	sb, b, cf := bingDNSSandbox(t)
	r := sb.Run("bing", "verify", "example.com", "--cloudflare-zone", "auto", "--dry-run")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "VerifySite")
	noLeak(t, r, bingKey)
	noLeak(t, r, "cf-test")
	if b.Calls("AddSite") != 0 || b.Calls("VerifySite") != 0 || len(cf.Records("zone-1")) != 0 {
		t.Fatal("dry run sent a request")
	}
}
