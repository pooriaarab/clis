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

func TestSitemapCheckDryRunSendsNothing(t *testing.T) {
	sb, site := checkSandbox(t)
	site.Serve("/sitemap.xml", 200, "application/xml", fakes.URLSet(site.URL+"/"))
	r := sb.Run("sitemap", "check", site.URL+"/sitemap.xml", "--dry-run", "--json")
	wantExit(t, r, 0)
	v := r.JSON(t)
	calls, _ := v["calls"].([]any)
	if v["dry_run"] != true || len(calls) != 1 {
		t.Fatalf("dry run must list one call: %s", r.Stdout)
	}
	if site.Hits("/sitemap.xml") != 0 {
		t.Fatal("dry run sent a request")
	}
}
