package e2e

import (
	"testing"

	"github.com/pooriaarab/clis/search-console/e2e/fakes"
)

// Ways `bing sitemap` can fail, listed before the code was written:
//  1. The Bing key is missing: stop before any request.
//  2. The sitemap has problems and the CLI submits it anyway.
//  3. The site is not added, or added but not verified: exit 1, name `bing verify`, no SubmitFeed.
//  4. Bing refuses SubmitFeed (HTTP 400 or a 200 error body): exit 1 with its message.
//  5. Bing lists no feed right after the submit: that is normal, so exit 0 and say so.
//  6. A resubmit makes a second feed, or fails.
//  7. `status` for a sitemap nobody submitted: exit 1, not an empty report.
//  8. Bing gives an error status: exit 3. A pending status exits 0.
//  9. The .NET date "/Date(ms)/" reaches the user as raw text.
// 10. --dry-run sends a request, or the key leaks.

func bingSitemapSandbox(t *testing.T) (*Sandbox, *fakes.Bing, string) {
	sb, b := bingSandbox(t)
	b.AddSite("https://example.com/", fakes.BingSite{Verified: true, DNSCode: bingCode})
	site := fakes.NewSite(t)
	site.Serve("/sitemap.xml", 200, "application/xml", fakes.URLSet(site.URL+"/", site.URL+"/a"))
	sb.Alias(site.URL, "https://site.fake")
	return sb, b, site.URL + "/sitemap.xml"
}

func TestBingSitemapSubmitThenStatus(t *testing.T) {
	sb, b, sm := bingSitemapSandbox(t)
	r := sb.Run("bing", "sitemap", "submit", "example.com", sm, "--json")
	wantExit(t, r, 0)
	if v := r.JSON(t); v["submitted"] != true || v["listed"] != true || v["status"] != "Success" {
		t.Fatalf("unexpected result: %s", r.Stdout)
	}
	if got := b.Feeds("https://example.com/"); len(got) != 1 || got[0] != sm {
		t.Fatalf("feeds = %v, want %s", got, sm)
	}
	r = sb.Run("bing", "sitemap", "status", "example.com", sm, "--json")
	wantExit(t, r, 0)
	v := r.JSON(t)
	if v["submitted"] != false || v["last_crawled"] != "" {
		t.Fatalf("Bing's date for never (1601-01-01) was not read as never: %s", r.Stdout)
	}
	wantContains(t, "stdout", sb.Run("bing", "sitemap", "status", "example.com", sm).Stdout, "last crawled: never")
	noLeak(t, r, bingKey)
}

func TestBingSitemapSubmit81058IsAlreadyPresent(t *testing.T) {
	for _, as200 := range []bool{false, true} {
		sb, b, sm := bingSitemapSandbox(t)
		b.Configure(func(f *fakes.Bing) { f.ErrorsAs200 = as200 })
		wantExit(t, sb.Run("bing", "sitemap", "submit", "example.com", sm), 0)
		b.Configure(func(f *fakes.Bing) { f.DuplicateFeedIs81058 = true })
		r := sb.Run("bing", "sitemap", "submit", "example.com", sm, "--json")
		wantExit(t, r, 0)
		if v := r.JSON(t); v["already_present"] != true || v["listed"] != true || v["ok"] != true {
			t.Fatalf("as200=%t: unexpected result: %s", as200, r.Stdout)
		}
		wantContains(t, "stdout", sb.Run("bing", "sitemap", "submit", "example.com", sm).Stdout, "already")
	}
}

func TestBingSitemapSubmitFindsSiteWhateverItsURLLooksLike(t *testing.T) {
	sb, b, sm := bingSitemapSandbox(t)
	b.AddSite("http://Example.com", fakes.BingSite{Verified: true, DNSCode: bingCode})
	wantExit(t, sb.Run("bing", "sitemap", "submit", "example.com", sm), 0)
}

func TestBingSitemapResubmitKeepsOneFeed(t *testing.T) {
	sb, b, sm := bingSitemapSandbox(t)
	for i := 0; i < 2; i++ {
		wantExit(t, sb.Run("bing", "sitemap", "submit", "example.com", sm), 0)
	}
	if got := b.Feeds("https://example.com/"); len(got) != 1 {
		t.Fatalf("feeds = %v, want one", got)
	}
}

func TestBingSitemapSubmitBeforeBingListsIt(t *testing.T) {
	sb, b, sm := bingSitemapSandbox(t)
	b.Configure(func(f *fakes.Bing) { f.HideFeeds = true })
	r := sb.Run("bing", "sitemap", "submit", "example.com", sm)
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "does not list the sitemap yet")
}

func TestBingSitemapProblemsAreNotSubmitted(t *testing.T) {
	sb, b, _ := bingSitemapSandbox(t)
	site := fakes.NewSite(t)
	site.Serve("/sitemap.xml", 404, "text/html", "missing")
	r := sb.Run("bing", "sitemap", "submit", "example.com", site.URL+"/sitemap.xml")
	wantExit(t, r, 3)
	wantContains(t, "stderr", r.Stderr, "not submitted")
	if b.Calls("SubmitFeed") != 0 {
		t.Fatal("a broken sitemap was submitted")
	}
}

func TestBingSitemapNeedsVerifiedSite(t *testing.T) {
	sb, b, sm := bingSitemapSandbox(t)
	b.AddSite("https://pending.example/", fakes.BingSite{DNSCode: bingCode})
	for _, d := range []string{"missing.example", "pending.example"} {
		r := sb.Run("bing", "sitemap", "submit", d, sm)
		wantExit(t, r, 1)
		wantContains(t, "stderr", r.Stderr, "bing verify "+d)
	}
	if b.Calls("SubmitFeed") != 0 {
		t.Fatal("SubmitFeed ran for a site that is not verified")
	}
}

func TestBingSitemapSubmitErrorInBothShapes(t *testing.T) {
	for _, as200 := range []bool{false, true} {
		sb, b, sm := bingSitemapSandbox(t)
		b.Configure(func(f *fakes.Bing) { f.ErrorsAs200 = as200 })
		sb.Env["BING_WEBMASTER_API_KEY"] = "wrong-key"
		r := sb.Run("bing", "sitemap", "submit", "example.com", sm)
		wantExit(t, r, 1)
		wantContains(t, "stderr", r.Stderr, "InvalidApiKey")
		noLeak(t, r, "wrong-key")
	}
}

func TestBingSitemapStatusOfUnknownSitemap(t *testing.T) {
	sb, _, sm := bingSitemapSandbox(t)
	r := sb.Run("bing", "sitemap", "status", "example.com", sm)
	wantExit(t, r, 1)
	wantContains(t, "stderr", r.Stderr, "bing sitemap submit")
}

func TestBingSitemapErrorStatusExits3(t *testing.T) {
	sb, b, sm := bingSitemapSandbox(t)
	b.Configure(func(f *fakes.Bing) { f.FeedStatus = "Error" })
	wantExit(t, sb.Run("bing", "sitemap", "submit", "example.com", sm), 3)
	r := sb.Run("bing", "sitemap", "status", "example.com", sm, "--json")
	wantExit(t, r, 3)
	if r.JSON(t)["ok"] != false {
		t.Fatalf("ok should be false: %s", r.Stdout)
	}
}

func TestBingSitemapMissingKey(t *testing.T) {
	sb, b, sm := bingSitemapSandbox(t)
	delete(sb.Env, "BING_WEBMASTER_API_KEY")
	for _, c := range []string{"submit", "status"} {
		r := sb.Run("bing", "sitemap", c, "example.com", sm)
		wantExit(t, r, 1)
		wantContains(t, "stderr", r.Stderr, "BING_WEBMASTER_API_KEY")
	}
	if b.Calls("GetUserSites")+b.Calls("GetFeeds")+b.Calls("SubmitFeed") != 0 {
		t.Fatal("a request was sent without a key")
	}
}

func TestBingSitemapDryRun(t *testing.T) {
	sb, b, sm := bingSitemapSandbox(t)
	r := sb.Run("bing", "sitemap", "submit", "example.com", sm, "--dry-run")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "SubmitFeed")
	noLeak(t, r, bingKey)
	if b.Calls("SubmitFeed") != 0 || b.Calls("GetUserSites") != 0 {
		t.Fatal("dry run sent a request")
	}
}
