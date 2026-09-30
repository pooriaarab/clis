package e2e

import (
	"path/filepath"
	"testing"

	"github.com/pooriaarab/clis/search-console/e2e/fakes"
)

// Ways `launch` can fail, listed before the code was written:
//  1. --sitemap is missing, the domain is bad or --skip is unknown: exit 2, no step runs.
//  2. A rerun fails or repeats work: a second DNS record, a second site, a second feed.
//  3. One failed step stops the others, or a sitemap step runs after its verify failed.
//  4. The exit code hides which kind of failure came first (timeout is 4, sitemap is 3).
//  5. A missing Bing key shows as a silent pass, or does not name the variable.
//  6. --skip still calls the skipped provider.
//  7. The sitemap has problems and a submit step sends it anyway.
//  8. --dry-run changes DNS, adds a site or posts URLs.
//  9. The table does not show each step, or a secret leaks into it.

type launchRig struct {
	sb     *Sandbox
	g      *fakes.Google
	cf     *fakes.Cloudflare
	b      *fakes.Bing
	n      *fakes.IndexNow
	site   *fakes.Site
	keyDir string
	args   []string
}

func newLaunchRig(t *testing.T) *launchRig {
	sb, g, cf := cloudflareSandbox(t)
	b := fakes.NewBing(t)
	sb.Env["BING_API_BASE"], sb.Env["BING_WEBMASTER_API_KEY"] = b.URL, bingKey
	sb.Alias(b.URL, "http://bing.fake")
	b.Ready = cf.HasCNAME
	e := indexnowSandbox(t)
	sb.Env["INDEXNOW_API_BASE"] = e.n.URL + "/indexnow"
	sb.Alias(e.n.URL, "http://indexnow.fake")
	sb.Alias(e.site.URL, "http://site.fake")
	e.sb = sb
	sb.Alias(e.keyDir, "$KEYDIR")
	sb.Alias(e.workDir, "$WORK")
	e.seedKey(t)
	e.site.Serve("/sitemap.xml", 200, "application/xml", fakes.URLSet(e.site.URL+"/", e.site.URL+"/a"))
	list := e.urlFile(t, "https://example.com/", "https://example.com/a")
	return &launchRig{sb: sb, g: g, cf: cf, b: b, n: e.n, site: e.site, keyDir: e.keyDir, args: []string{
		"launch", "example.com", "--sitemap", e.site.URL + "/sitemap.xml", "--cloudflare-zone", "auto",
		"--interval", "20ms", "--wait", "400ms", "--key-dir", e.keyDir, "--key-location", e.keyLoc, "--indexnow-urls", list,
	}}
}

func (l *launchRig) run(extra ...string) Result {
	r := l.sb.Run(append(append([]string{}, l.args...), extra...)...)
	for _, secret := range []string{"cf-test", bingKey, "at-test"} {
		noLeak(l.sb.t, r, secret)
	}
	return r
}

// results maps a step name to its row in the JSON table.
func results(t *testing.T, r Result) map[string]map[string]any {
	t.Helper()
	out := map[string]map[string]any{}
	for _, row := range r.JSON(t)["steps"].([]any) {
		m := row.(map[string]any)
		out[m["step"].(string)] = m
	}
	if len(out) != 5 {
		t.Fatalf("the table has %d rows, want 5:\n%s", len(out), r.Stdout)
	}
	return out
}

func wantSteps(t *testing.T, got map[string]map[string]any, want map[string]string) {
	t.Helper()
	for step, result := range want {
		if got[step]["result"] != result {
			t.Fatalf("%s = %v (%v), want %s", step, got[step]["result"], got[step]["detail"], result)
		}
	}
}

func TestLaunchPassesThenRerunsWithoutRepeating(t *testing.T) {
	l := newLaunchRig(t)
	all := map[string]string{"google verify": "pass", "google sitemap": "pass", "bing verify": "pass", "bing sitemap": "pass", "indexnow": "pass"}
	for i := 0; i < 2; i++ {
		r := l.run("--json")
		wantExit(t, r, 0)
		wantSteps(t, results(t, r), all)
	}
	if got := len(l.cf.Records("zone-1")); got != 2 {
		t.Fatalf("DNS records = %d after two runs, want 2 (one TXT, one CNAME)", got)
	}
	if len(l.g.Sitemaps()) != 1 || len(l.b.Feeds("https://example.com/")) != 1 {
		t.Fatal("a rerun made a second sitemap")
	}
	r := l.run()
	wantExit(t, r, 0)
	for _, want := range []string{"STEP", "google verify", "already verified", "indexnow", "pass"} {
		wantContains(t, "stdout", r.Stdout, want)
	}
}

func TestLaunchKeepsGoingAfterAFailure(t *testing.T) {
	l := newLaunchRig(t)
	l.g.Ready = func(string, string) bool { return false } // DNS never shows the TXT record
	r := l.run("--json")
	wantExit(t, r, 4)
	wantSteps(t, results(t, r), map[string]string{"google verify": "fail", "google sitemap": "skip", "bing verify": "pass", "bing sitemap": "pass", "indexnow": "pass"})
	if len(l.g.Sitemaps()) != 0 {
		t.Fatal("google sitemap ran after its verify step failed")
	}
}

func TestLaunchNamesAMissingBingKey(t *testing.T) {
	l := newLaunchRig(t)
	delete(l.sb.Env, "BING_WEBMASTER_API_KEY")
	r := l.run("--json")
	wantExit(t, r, 1)
	got := results(t, r)
	wantSteps(t, got, map[string]string{"google verify": "pass", "google sitemap": "pass", "bing verify": "fail", "bing sitemap": "skip", "indexnow": "pass"})
	wantContains(t, "detail", got["bing verify"]["detail"].(string), "BING_WEBMASTER_API_KEY")
}

func TestLaunchSkipsAProvider(t *testing.T) {
	l := newLaunchRig(t)
	r := l.run("--json", "--skip", "bing,indexnow")
	wantExit(t, r, 0)
	wantSteps(t, results(t, r), map[string]string{"google verify": "pass", "google sitemap": "pass", "bing verify": "skip", "bing sitemap": "skip", "indexnow": "skip"})
	if l.b.Calls("GetUserSites") != 0 || len(l.n.Posts()) != 0 {
		t.Fatal("a skipped provider was called")
	}
}

func TestLaunchUsageErrorsRunNothing(t *testing.T) {
	l := newLaunchRig(t)
	for _, args := range [][]string{
		{"launch", "example.com"},
		{"launch", "https://example.com", "--sitemap", l.site.URL + "/sitemap.xml"},
		append(append([]string{}, l.args...), "--skip", "yahoo"),
	} {
		wantExit(t, l.sb.Run(args...), 2)
	}
	if l.g.Calls("POST /siteVerification/v1/token") != 0 || l.b.Calls("GetUserSites") != 0 || len(l.cf.Records("zone-1")) != 0 {
		t.Fatal("a step ran after a usage error")
	}
}

func TestLaunchDoesNotSubmitABrokenSitemap(t *testing.T) {
	l := newLaunchRig(t)
	l.site.Serve("/sitemap.xml", 404, "text/html", "missing")
	r := l.run("--json")
	wantExit(t, r, 3)
	wantSteps(t, results(t, r), map[string]string{"google verify": "pass", "google sitemap": "fail", "bing verify": "pass", "bing sitemap": "fail", "indexnow": "pass"})
	if len(l.g.Sitemaps()) != 0 || len(l.b.Feeds("https://example.com/")) != 0 {
		t.Fatal("a broken sitemap was submitted")
	}
}

func TestLaunchShowsAnUnreachableIndexNowKey(t *testing.T) {
	l := newLaunchRig(t)
	l.site.Serve("/"+inKey+".txt", 404, "text/html", "missing")
	r := l.run("--json")
	wantExit(t, r, 1)
	got := results(t, r)
	wantSteps(t, got, map[string]string{"google verify": "pass", "bing sitemap": "pass", "indexnow": "fail"})
	wantContains(t, "detail", got["indexnow"]["detail"].(string), "not reachable")
	if len(l.n.Posts()) != 0 {
		t.Fatal("URLs were posted while the key file was unreachable")
	}
}

func TestLaunchDryRunChangesNothing(t *testing.T) {
	l := newLaunchRig(t)
	r := l.run("--dry-run")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "dry-run:")
	if len(l.cf.Records("zone-1")) != 0 || l.g.HasSite("example.com") || len(l.n.Posts()) != 0 || l.b.Calls("AddSite") != 0 {
		t.Fatal("dry run changed something")
	}
	if files, _ := filepath.Glob(filepath.Join(l.keyDir, "*")); len(files) != 0 {
		t.Fatalf("dry run wrote %v", files)
	}
}
