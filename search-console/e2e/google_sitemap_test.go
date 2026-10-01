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
	site.Serve("/sitemap.xml", 200, "application/xml", fakes.URLSet("https://example.com/", "https://example.com/a"))
	sb.Env["SITEMAP_FETCH_BASE"] = site.URL
	sb.Alias(site.URL, "https://site.fake")
	return sb, g, "https://example.com/sitemap.xml"
}

func TestGoogleSitemapSubmitPollsUntilRead(t *testing.T) {
	sb, g, sm := sitemapSandbox(t)
	g.PendingPolls = 2
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
	g.PendingPolls = 1000
	r := sb.Run("google", "sitemap", "submit", "example.com", sm, "--interval", "20ms", "--wait", "200ms")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "pending: true")
}

func TestGoogleSitemapProblemsAreNotSubmitted(t *testing.T) {
	sb, g, sm := sitemapSandbox(t)
	site := fakes.NewSite(t)
	site.Serve("/sitemap.xml", 404, "text/html", "missing")
	sb.Env["SITEMAP_FETCH_BASE"] = site.URL
	r := sb.Run("google", "sitemap", "submit", "example.com", sm)
	wantExit(t, r, 3)
	wantContains(t, "stderr", r.Stderr, "not submitted")
	if len(g.Sitemaps()) != 0 {
		t.Fatal("a sitemap with problems was submitted")
	}
}

func TestGoogleSitemapNeedsVerifiedSite(t *testing.T) {
	sb, g, sm := sitemapSandbox(t)
	exampleBase := sb.Env["SITEMAP_FETCH_BASE"]
	other := fakes.NewSite(t)
	other.Serve("/sitemap.xml", 200, "application/xml", fakes.URLSet("https://other.com/"))
	sb.Env["SITEMAP_FETCH_BASE"] = other.URL
	r := sb.Run("google", "sitemap", "submit", "other.com", "https://other.com/sitemap.xml")
	wantExit(t, r, 1)
	sb.Env["SITEMAP_FETCH_BASE"] = exampleBase
	wantContains(t, "stderr", r.Stderr, "google verify other.com")
	g.Permission = "siteUnverifiedUser"
	r = sb.Run("google", "sitemap", "submit", "example.com", sm)
	wantExit(t, r, 1)
	wantContains(t, "stderr", r.Stderr, "want siteOwner")
	if len(g.Sitemaps()) != 0 {
		t.Fatal("a sitemap was submitted without ownership")
	}
}

func TestGoogleSitemapErrorsExitThree(t *testing.T) {
	sb, g, sm := sitemapSandbox(t)
	g.SitemapErrors, g.SitemapWarnings = 3, 2
	r := sb.Run("google", "sitemap", "submit", "example.com", sm, "--interval", "20ms", "--json")
	wantExit(t, r, 3)
	if v := r.JSON(t); v["errors"] != float64(3) || v["warnings"] != float64(2) {
		t.Fatalf("counts were lost: %s", r.Stdout)
	}
	r = sb.Run("google", "sitemap", "status", "example.com", sm)
	wantExit(t, r, 3)
	wantContains(t, "stdout", r.Stdout, "errors: 3")

	g.SitemapErrors = 0
	r = sb.Run("google", "sitemap", "status", "example.com", sm)
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "warnings: 2")
}

func TestGoogleSitemapSubmitRefused(t *testing.T) {
	sb, g, sm := sitemapSandbox(t)
	g.SitemapStatus = 403
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

// Ways the sitemap URL can be wrong for the property, listed before the code was written:
// 12. The URL is http, so Google gets a sitemap it will not trust: refuse before any request.
// 13. The URL is on another host than the property, or only looks like it
//     (notexample.com, example.com.evil.test, user@host): refuse before any request.
// 14. A subdomain of the property (www.example.com) is a valid sitemap host.
// 15. --dry-run accepts a sitemap that the real run refuses (bad URL or a bad sitemap).

func TestGoogleSitemapRejectsWrongURL(t *testing.T) {
	for _, bad := range []string{
		"http://example.com/sitemap.xml",
		"https://other.com/sitemap.xml",
		"https://notexample.com/sitemap.xml",
		"https://example.com.evil.test/sitemap.xml",
		"https://example.com@evil.test/sitemap.xml",
		"ftp://example.com/sitemap.xml",
	} {
		for _, extra := range [][]string{nil, {"--dry-run"}} {
			sb, g, _ := sitemapSandbox(t)
			r := sb.Run(append([]string{"google", "sitemap", "submit", "example.com", bad}, extra...)...)
			wantExit(t, r, 2)
			wantContains(t, "stderr", r.Stderr, "sitemap URL")
			if len(g.Sitemaps()) != 0 {
				t.Fatalf("%s was submitted", bad)
			}
		}
	}
}

func TestGoogleSitemapAcceptsSubdomain(t *testing.T) {
	sb, g, _ := sitemapSandbox(t)
	site := fakes.NewSite(t)
	site.Serve("/sitemap.xml", 200, "application/xml", fakes.URLSet("https://www.example.com/", "https://www.example.com/a"))
	sb.Env["SITEMAP_FETCH_BASE"] = site.URL
	sm := "https://www.example.com/sitemap.xml"
	r := sb.Run("google", "sitemap", "submit", "example.com", sm, "--interval", "20ms")
	wantExit(t, r, 0)
	if got := g.Sitemaps(); len(got) != 1 || got[0] != sm {
		t.Fatalf("submitted = %v, want %s", got, sm)
	}
}

func TestGoogleSitemapDryRunRefusesBadSitemap(t *testing.T) {
	sb, g, sm := sitemapSandbox(t)
	site := fakes.NewSite(t)
	site.Serve("/sitemap.xml", 404, "text/html", "missing")
	sb.Env["SITEMAP_FETCH_BASE"] = site.URL
	real := sb.Run("google", "sitemap", "submit", "example.com", sm)
	dry := sb.Run("google", "sitemap", "submit", "example.com", sm, "--dry-run")
	wantExit(t, real, 3)
	wantExit(t, dry, real.Code)
	if len(g.Sitemaps()) != 0 {
		t.Fatal("a sitemap with problems was submitted")
	}
}
