package e2e

import (
	"strings"
	"testing"

	"github.com/pooriaarab/clis/search-console/e2e/fakes"
)

// Ways `google verify --cloudflare-zone` can fail, listed before the code was written:
//  1. CLOUDFLARE_API_TOKEN is not set: stop before any request, and name only the env var.
//  2. --cloudflare-zone auto finds no zone: exit 1 with a clear message.
//  3. An explicit zone id still triggers a zone lookup.
//  4. A rerun creates the TXT record again (Cloudflare refuses an identical one).
//  5. Another TXT record at the same name blocks the new one, or gets replaced.
//  6. Cloudflare refuses the call (bad token, no permission) and the CLI carries on.
//  7. A subdomain does not find the parent zone.
//  8. The Cloudflare token leaks to stdout or stderr.
//  9. --dry-run sends a request or needs a real zone id.
// 10. Google is asked to verify before the record exists.

const cfRecord = "google-site-verification=fake-example.com"

func cloudflareSandbox(t *testing.T) (*Sandbox, *fakes.Google, *fakes.Cloudflare) {
	sb, g := googleSandbox(t)
	cf := fakes.NewCloudflare(t)
	sb.Alias(cf.URL, "http://cloudflare.fake")
	sb.Env["CLOUDFLARE_API_BASE"] = cf.URL
	sb.Env["CLOUDFLARE_API_TOKEN"] = "cf-test"
	cf.AddZone("example.com", "zone-1")
	// Google sees the TXT record only after Cloudflare holds it.
	g.Configure(func(f *fakes.Google) { f.Ready = cf.HasTXT })
	return sb, g, cf
}

func TestCloudflareZoneAutoCreatesTXTThenVerifies(t *testing.T) {
	sb, g, cf := cloudflareSandbox(t)
	r := sb.Run("google", "verify", "example.com", "--cloudflare-zone", "auto", "--interval", "20ms", "--json")
	wantExit(t, r, 0)
	if !cf.HasTXT("example.com", cfRecord) {
		t.Fatal("the TXT record is not on Cloudflare")
	}
	if v := r.JSON(t); v["cloudflare_created"] != true || v["attempts"] != float64(1) {
		t.Fatalf("unexpected result: %s", r.Stdout)
	}
	if !g.HasSite("example.com") {
		t.Fatal("the site was not added")
	}
	if strings.Contains(r.Stdout+r.Stderr, "cf-test") {
		t.Fatal("the Cloudflare token leaked")
	}
}

func TestCloudflareExplicitZoneSkipsLookup(t *testing.T) {
	sb, _, cf := cloudflareSandbox(t)
	wantExit(t, sb.Run("google", "verify", "example.com", "--cloudflare-zone", "zone-1", "--interval", "20ms"), 0)
	if cf.Calls("GET /zones") != 0 {
		t.Fatal("an explicit zone id triggered a zone lookup")
	}
	if len(cf.Records("zone-1")) != 1 {
		t.Fatalf("records = %v, want one", cf.Records("zone-1"))
	}
}

func TestCloudflareRerunDoesNotDuplicateTXT(t *testing.T) {
	sb, g, cf := cloudflareSandbox(t)
	g.Configure(func(f *fakes.Google) { f.Ready = func(string, string) bool { return false } }) // verify never works, so the record stays unverified
	for i := 0; i < 2; i++ {
		r := sb.Run("google", "verify", "example.com", "--cloudflare-zone", "auto", "--interval", "20ms", "--wait", "100ms")
		wantExit(t, r, 4)
	}
	if got := len(cf.Records("zone-1")); got != 1 {
		t.Fatalf("records = %d after two runs, want 1", got)
	}
}

func TestCloudflareKeepsOtherTXTRecords(t *testing.T) {
	sb, _, cf := cloudflareSandbox(t)
	cf.SeedRecord("zone-1", "TXT", "example.com", "v=spf1 -all")
	wantExit(t, sb.Run("google", "verify", "example.com", "--cloudflare-zone", "auto", "--interval", "20ms"), 0)
	if got := len(cf.Records("zone-1")); got != 2 {
		t.Fatalf("records = %v, want the old one and the new one", cf.Records("zone-1"))
	}
}

func TestCloudflareTokenMissing(t *testing.T) {
	sb, g, _ := cloudflareSandbox(t)
	delete(sb.Env, "CLOUDFLARE_API_TOKEN")
	r := sb.Run("google", "verify", "example.com", "--cloudflare-zone", "auto")
	wantExit(t, r, 1)
	wantContains(t, "stderr", r.Stderr, "CLOUDFLARE_API_TOKEN")
	if g.Calls("GET /siteVerification/v1/webResource") != 0 {
		t.Fatal("the CLI called Google before it checked the Cloudflare token")
	}
}

func TestCloudflareNoZoneForDomain(t *testing.T) {
	sb, g, _ := cloudflareSandbox(t)
	r := sb.Run("google", "verify", "other.org", "--cloudflare-zone", "auto")
	wantExit(t, r, 1)
	wantContains(t, "stderr", r.Stderr, "no Cloudflare zone found for the domain: other.org")
	if g.Calls("POST /siteVerification/v1/webResource") != 0 {
		t.Fatal("the CLI verified without a record")
	}
}

func TestCloudflareSubdomainUsesParentZone(t *testing.T) {
	sb, _, cf := cloudflareSandbox(t)
	wantExit(t, sb.Run("google", "verify", "shop.example.com", "--cloudflare-zone", "auto", "--interval", "20ms"), 0)
	if !cf.HasTXT("shop.example.com", "google-site-verification=fake-shop.example.com") {
		t.Fatal("the record for the subdomain is missing")
	}
}

func TestCloudflareRefused(t *testing.T) {
	sb, _, cf := cloudflareSandbox(t)
	cf.Configure(func(f *fakes.Cloudflare) { f.CreateStatus = 403 })
	r := sb.Run("google", "verify", "example.com", "--cloudflare-zone", "auto", "--interval", "20ms")
	wantExit(t, r, 1)
	wantContains(t, "stderr", r.Stderr, "not allowed for this zone")
	sb.Env["CLOUDFLARE_API_TOKEN"] = "cf-wrong"
	r = sb.Run("google", "verify", "example.com", "--cloudflare-zone", "auto")
	wantExit(t, r, 1)
	wantContains(t, "stderr", r.Stderr, "Authentication error")
}

func TestCloudflareDryRun(t *testing.T) {
	sb, _, cf := cloudflareSandbox(t)
	r := sb.Run("google", "verify", "example.com", "--cloudflare-zone", "auto", "--dry-run")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "/zones/ZONE_ID/dns_records")
	if cf.Calls("GET /zones") != 0 || len(cf.Records("zone-1")) != 0 {
		t.Fatal("dry run sent a request")
	}
	if strings.Contains(r.Stdout+r.Stderr, "cf-test") {
		t.Fatal("dry run printed the Cloudflare token")
	}
}
