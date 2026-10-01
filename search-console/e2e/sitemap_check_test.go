package e2e

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pooriaarab/clis/search-console/e2e/fakes"
)

// Ways `sitemap check` can fail, listed before the code was written:
//  1. The URL is not absolute http(s): the CLI must refuse before any request.
//  2. The sitemap answers 404 and the CLI reports success.
//  3. The site answers 200 with an HTML page (a soft 404) and the CLI accepts it.
//  4. The XML is cut off (a failed deploy) and the CLI accepts the part it read.
//  5. The root element is neither urlset nor sitemapindex.
//  6. The sitemap holds 50,000 or more URLs, which Google rejects.
//  7. The sitemap holds no URL at all.
//  8. The server is down: exit 1 with the address in the message, not exit 3.
//  9. --dry-run sends the request.
// 10. Problems must exit 3 and --json must list every problem, not only the first.

func checkSandbox(t *testing.T) (*Sandbox, *fakes.Site) {
	sb := newSandbox(t)
	site := fakes.NewSite(t)
	sb.Alias(site.URL, "https://site.fake")
	return sb, site
}

func TestSitemapCheckGood(t *testing.T) {
	sb, site := checkSandbox(t)
	site.Serve("/sitemap.xml", 200, "application/xml", fakes.URLSet(site.URL+"/", site.URL+"/a"))
	r := sb.Run("sitemap", "check", site.URL+"/sitemap.xml", "--json")
	wantExit(t, r, 0)
	v := r.JSON(t)
	if v["ok"] != true || v["urls"] != float64(2) || v["kind"] != "urlset" {
		t.Fatalf("unexpected report: %v", v)
	}
}

func TestSitemapCheckProblems(t *testing.T) {
	var many strings.Builder
	many.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	for i := 0; i < 50000; i++ {
		fmt.Fprintf(&many, "<url><loc>https://site.fake/%d</loc></url>", i)
	}
	many.WriteString("</urlset>")

	cases := []struct {
		name        string
		status      int
		contentType string
		body        string
		wantProblem string
	}{
		{"not found", 404, "text/html", "missing", "HTTP 404"},
		{"soft 404 html", 200, "text/html", "<html><body>Not found</body></html>", "urlset or sitemapindex"},
		{"truncated xml", 200, "application/xml", `<urlset><url><loc>https://site.fake/a</loc>`, "XML"},
		{"wrong root", 200, "application/xml", `<feed><entry/></feed>`, "urlset or sitemapindex"},
		{"too many urls", 200, "application/xml", many.String(), "50,000"},
		{"empty", 200, "application/xml", `<urlset></urlset>`, "no URLs"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sb, site := checkSandbox(t)
			site.Serve("/sitemap.xml", c.status, c.contentType, c.body)
			r := sb.Run("sitemap", "check", site.URL+"/sitemap.xml", "--json")
			wantExit(t, r, 3)
			v := r.JSON(t)
			problems := fmt.Sprint(v["problems"])
			if v["ok"] != false || !strings.Contains(problems, c.wantProblem) {
				t.Fatalf("want problem %q, got %v", c.wantProblem, v)
			}
		})
	}
}

func TestSitemapCheckRelativeURL(t *testing.T) {
	sb, _ := checkSandbox(t)
	r := sb.Run("sitemap", "check", "/sitemap.xml")
	wantExit(t, r, 2)
	wantContains(t, "stderr", r.Stderr, "absolute")
}

func TestSitemapCheckServerDown(t *testing.T) {
	sb, site := checkSandbox(t)
	url := site.URL + "/sitemap.xml"
	site.Close()
	r := sb.Run("sitemap", "check", url)
	wantExit(t, r, 1)
}

// Ways --dry-run can fail, listed before the code was written:
// 17. The dry run skips the fetch, so it exits 0 for a sitemap the real run rejects.
// 18. The dry run reads a different set of problems than the real run.
// 19. The dry run changes something remote. A GET changes nothing, so it must go out.

func TestSitemapCheckDryRunMatchesRealRun(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		exit   int
	}{
		{"good", 200, fakes.URLSet("SITE/", "SITE/a"), 0},
		{"not found", 404, "missing", 3},
		{"wrong root", 200, `<feed><entry/></feed>`, 3},
		{"other host", 200, fakes.URLSet("SITE/", "https://other.test/a"), 3},
		{"empty", 200, `<urlset></urlset>`, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sb, site := checkSandbox(t)
			site.Serve("/sitemap.xml", c.status, "application/xml", strings.ReplaceAll(c.body, "SITE", site.URL))
			real := sb.Run("sitemap", "check", site.URL+"/sitemap.xml", "--json")
			dry := sb.Run("sitemap", "check", site.URL+"/sitemap.xml", "--dry-run", "--json")
			wantExit(t, real, c.exit)
			wantExit(t, dry, c.exit)
			rv, dv := real.JSON(t), dry.JSON(t)
			for _, k := range []string{"ok", "kind", "urls", "problems"} {
				if fmt.Sprint(rv[k]) != fmt.Sprint(dv[k]) {
					t.Fatalf("%s differs: real %v, dry run %v", k, rv[k], dv[k])
				}
			}
			if calls, _ := dv["calls"].([]any); dv["dry_run"] != true || len(calls) != 0 {
				t.Fatalf("a dry run of a read must list no unsent call: %s", dry.Stdout)
			}
			if site.Hits("/sitemap.xml") != 2 {
				t.Fatalf("the sitemap was fetched %d times, want 2 (one per run)", site.Hits("/sitemap.xml"))
			}
		})
	}
}

// More ways `sitemap check` can fail, listed before the code was written:
// 11. A <loc> is relative (/a) or not http(s), and the CLI accepts it.
// 12. A <loc> is on another host (or www vs apex), which search engines drop.
// 13. A sitemap index lists a child that answers 404 and the CLI never fetches it.
// 14. A child of an index is itself an index (nesting is not allowed).
// 15. A child holds bad URLs and the report does not say which child.
// 16. The URL count of an index is the number of children, not of pages.

func TestSitemapCheckURLRules(t *testing.T) {
	cases := []struct {
		name string
		locs func(site string) []string
		want string
	}{
		{"relative", func(s string) []string { return []string{s + "/", "/a"} }, "not an absolute http(s) URL"},
		{"ftp scheme", func(s string) []string { return []string{"ftp://x.test/a"} }, "not an absolute http(s) URL"},
		{"other host", func(s string) []string { return []string{s + "/", "https://other.test/a"} }, "other.test"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sb, site := checkSandbox(t)
			site.Serve("/sitemap.xml", 200, "application/xml", fakes.URLSet(c.locs(site.URL)...))
			r := sb.Run("sitemap", "check", site.URL+"/sitemap.xml", "--json")
			wantExit(t, r, 3)
			wantContains(t, "problems", fmt.Sprint(r.JSON(t)["problems"]), c.want)
		})
	}
}

func indexXML(children ...string) string {
	var b strings.Builder
	b.WriteString(`<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	for _, c := range children {
		fmt.Fprintf(&b, "<sitemap><loc>%s</loc></sitemap>", c)
	}
	b.WriteString("</sitemapindex>")
	return b.String()
}

func TestSitemapCheckIndexFollowsChildren(t *testing.T) {
	sb, site := checkSandbox(t)
	site.Serve("/index.xml", 200, "application/xml", indexXML(site.URL+"/a.xml", site.URL+"/b.xml"))
	site.Serve("/a.xml", 200, "application/xml", fakes.URLSet(site.URL+"/1", site.URL+"/2"))
	site.Serve("/b.xml", 200, "application/xml", fakes.URLSet(site.URL+"/3"))
	r := sb.Run("sitemap", "check", site.URL+"/index.xml", "--json")
	wantExit(t, r, 0)
	v := r.JSON(t)
	if v["kind"] != "sitemapindex" || v["urls"] != float64(3) || v["sitemaps"] != float64(2) {
		t.Fatalf("index totals are wrong: %v", v)
	}
}

func TestSitemapCheckIndexChildProblems(t *testing.T) {
	sb, site := checkSandbox(t)
	site.Serve("/index.xml", 200, "application/xml", indexXML(site.URL+"/missing.xml", site.URL+"/nested.xml", site.URL+"/bad.xml"))
	site.Serve("/nested.xml", 200, "application/xml", indexXML(site.URL+"/a.xml"))
	site.Serve("/bad.xml", 200, "application/xml", fakes.URLSet("/relative"))
	r := sb.Run("sitemap", "check", site.URL+"/index.xml", "--json")
	wantExit(t, r, 3)
	problems := fmt.Sprint(r.JSON(t)["problems"])
	for _, want := range []string{"missing.xml", "HTTP 404", "nested.xml", "index inside an index", "bad.xml", "not an absolute"} {
		wantContains(t, "problems", problems, want)
	}
	// The nested index lists one sitemap. That is not a page, so it must not count.
	if v := r.JSON(t); v["urls"] != float64(1) {
		t.Fatalf("urls = %v, want 1 (the bad.xml entry only)", v["urls"])
	}
}

// Ways an index can hurt the machine that reads it, listed before the code was written:
// 20. A child is on another host and the CLI fetches it anyway (the index decides where we connect).
// 21. An index lists thousands of children and the CLI fetches every one.
// 22. A child that is an index is followed, so nesting has no end.

func TestSitemapCheckIndexDoesNotFetchOtherHosts(t *testing.T) {
	sb, site := checkSandbox(t)
	other := fakes.NewSite(t)
	other.Serve("/c.xml", 200, "application/xml", fakes.URLSet(other.URL+"/1"))
	site.Serve("/index.xml", 200, "application/xml", indexXML(site.URL+"/a.xml", other.URL+"/c.xml"))
	site.Serve("/a.xml", 200, "application/xml", fakes.URLSet(site.URL+"/1"))
	r := sb.Run("sitemap", "check", site.URL+"/index.xml", "--json")
	wantExit(t, r, 3)
	wantContains(t, "problems", fmt.Sprint(r.JSON(t)["problems"]), "not on the sitemap host")
	if n := other.Hits("/c.xml"); n != 0 {
		t.Fatalf("a child on another host was fetched %d times", n)
	}
	if site.Hits("/a.xml") != 1 {
		t.Fatal("the same-host child was not fetched")
	}
}

func TestSitemapCheckIndexHasChildCap(t *testing.T) {
	sb, site := checkSandbox(t)
	var children []string
	for i := 0; i < 1001; i++ {
		children = append(children, fmt.Sprintf("%s/c%d.xml", site.URL, i))
	}
	site.Serve("/index.xml", 200, "application/xml", indexXML(children...))
	r := sb.Run("sitemap", "check", site.URL+"/index.xml", "--json")
	wantExit(t, r, 3)
	wantContains(t, "problems", fmt.Sprint(r.JSON(t)["problems"]), "limit is 1000")
	if n := site.Hits("/c0.xml"); n != 0 {
		t.Fatalf("an index over the cap still had its children fetched (%d)", n)
	}
}

func TestSitemapCheckNestedIndexIsNotFollowed(t *testing.T) {
	sb, site := checkSandbox(t)
	site.Serve("/index.xml", 200, "application/xml", indexXML(site.URL+"/nested.xml"))
	site.Serve("/nested.xml", 200, "application/xml", indexXML(site.URL+"/deep.xml"))
	site.Serve("/deep.xml", 200, "application/xml", fakes.URLSet(site.URL+"/1"))
	r := sb.Run("sitemap", "check", site.URL+"/index.xml", "--json")
	wantExit(t, r, 3)
	if n := site.Hits("/deep.xml"); n != 0 {
		t.Fatalf("a nested index was followed (%d fetches of its child)", n)
	}
}
