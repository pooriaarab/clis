package e2e

import (
	"testing"

	"github.com/pooriaarab/clis/search-console/e2e/fakes"
)

// Ways `google sitemap` can fail, listed before the code was written:
//  1. The sitemap has problems (404, HTML, cut off) and the CLI submits it anyway.
//  2. The domain is not in Search Console yet: exit 1, name `google verify`, send no PUT.
//  3. The account is not siteOwner and the CLI submits anyway.
//  4. Google is still reading the sitemap: poll until it is done, then print the final status.
//  5. Google never finishes: stop at --wait, exit 0, and say pending is true.
//  6. Google reports errors: exit 3 with the counts. Warnings alone exit 0.
//  7. Google sends the counts as strings and the CLI reads them as 0.
//  8. Google refuses the submit: exit 1 with its message.
//  9. `status` for a sitemap nobody submitted: exit 1, not a zero-error report.
// 10. The sitemap URL is not encoded in the API path, so Google cannot find it.
// 11. --dry-run sends a request to Google.

func sitemapSandbox(t *testing.T) (*Sandbox, *fakes.Google, string) {
	sb, g := googleSandbox(t)
	g.PreAdd("example.com")
	site := fakes.NewSite(t)
	sm := site.URL + "/sitemap.xml"
	site.Serve("/sitemap.xml", 200, "application/xml", fakes.URLSet(site.URL+"/", site.URL+"/a"))
	sb.Alias(site.URL, "https://site.fake")
	return sb, g, sm
}

func TestGoogleSitemapSubmitPollsUntilRead(t *testing.T) {
	sb, g, sm := sitemapSandbox(t)
	g.Configure(func(f *fakes.Google) { f.PendingPolls = 2 })
	r := sb.Run("google", "sitemap", "submit", "example.com", sm, "--interval", "20ms", "--json")
	wantExit(t, r, 0)
	v := r.JSON(t)
	if v["isPending"] != false || v["lastDownloaded"] == "" || v["errors"] != float64(0) || v["warnings"] != float64(0) {
		t.Fatalf("unexpected status: %s", r.Stdout)
	}
	if got := g.Sitemaps(); len(got) != 1 || got[0] != sm {
		t.Fatalf("submitted = %v, want %s", got, sm)
	}
}

func TestGoogleSitemapSubmitStopsWhenStillPending(t *testing.T) {
	sb, g, sm := sitemapSandbox(t)
	g.Configure(func(f *fakes.Google) { f.PendingPolls = 1000 })
	r := sb.Run("google", "sitemap", "submit", "example.com", sm, "--interval", "20ms", "--wait", "200ms")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "pending: true")
}

func TestGoogleSitemapProblemsAreNotSubmitted(t *testing.T) {
	sb, g, _ := sitemapSandbox(t)
	site := fakes.NewSite(t)
	site.Serve("/sitemap.xml", 404, "text/html", "missing")
	r := sb.Run("google", "sitemap", "submit", "example.com", site.URL+"/sitemap.xml")
	wantExit(t, r, 3)
	wantContains(t, "stderr", r.Stderr, "not submitted")
	if len(g.Sitemaps()) != 0 {
		t.Fatal("a sitemap with problems was submitted")
	}
}

func TestGoogleSitemapNeedsVerifiedSite(t *testing.T) {
	sb, g, sm := sitemapSandbox(t)
	r := sb.Run("google", "sitemap", "submit", "other.com", sm)
	wantExit(t, r, 1)
	wantContains(t, "stderr", r.Stderr, "google verify other.com")
	g.Configure(func(f *fakes.Google) { f.Permission = "siteUnverifiedUser" })
	r = sb.Run("google", "sitemap", "submit", "example.com", sm)
	wantExit(t, r, 1)
	wantContains(t, "stderr", r.Stderr, "want siteOwner")
	if len(g.Sitemaps()) != 0 {
		t.Fatal("a sitemap was submitted without ownership")
	}
}

func TestGoogleSitemapErrorsExitThree(t *testing.T) {
	sb, g, sm := sitemapSandbox(t)
	g.Configure(func(f *fakes.Google) { f.SitemapErrors, f.SitemapWarnings = 3, 2 })
	r := sb.Run("google", "sitemap", "submit", "example.com", sm, "--interval", "20ms", "--json")
	wantExit(t, r, 3)
	if v := r.JSON(t); v["errors"] != float64(3) || v["warnings"] != float64(2) {
		t.Fatalf("counts were lost: %s", r.Stdout)
	}
	r = sb.Run("google", "sitemap", "status", "example.com", sm)
	wantExit(t, r, 3)
	wantContains(t, "stdout", r.Stdout, "errors: 3")

	g.Configure(func(f *fakes.Google) { f.SitemapErrors = 0 })
	r = sb.Run("google", "sitemap", "status", "example.com", sm)
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "warnings: 2")
}

func TestGoogleSitemapSubmitRefused(t *testing.T) {
	sb, g, sm := sitemapSandbox(t)
	g.Configure(func(f *fakes.Google) { f.SitemapStatus = 403 })
	r := sb.Run("google", "sitemap", "submit", "example.com", sm)
	wantExit(t, r, 1)
	wantContains(t, "stderr", r.Stderr, "sitemap refused")
}

func TestGoogleSitemapStatusNeverSubmitted(t *testing.T) {
	sb, _, sm := sitemapSandbox(t)
	r := sb.Run("google", "sitemap", "status", "example.com", sm)
	wantExit(t, r, 1)
	wantContains(t, "stderr", r.Stderr, "Sitemap not found")
}

func TestGoogleSitemapDryRun(t *testing.T) {
	sb, g, sm := sitemapSandbox(t)
	r := sb.Run("google", "sitemap", "submit", "example.com", sm, "--dry-run", "--json")
	wantExit(t, r, 0)
	if v := r.JSON(t); v["dry_run"] != true {
		t.Fatalf("not a dry run: %s", r.Stdout)
	}
	if len(g.Sitemaps()) != 0 {
		t.Fatal("dry run submitted a sitemap")
	}
}
