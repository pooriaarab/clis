package e2e

import (
	"os"
	"path/filepath"
	"strings"
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
// 10. A dry-run step shows pass for work it did not do, verify included.
// 11. The dry run exits 0 when the real run would fail: a missing login or key,
//     or an unreachable IndexNow key file.
// 12. The --json output says ok, or status pass, for a step that was skipped.
// 13. The dry run does not say that nothing was verified, or says it before the table ends.
// 14. A read-only check is faked instead of run: the credential check or the key file request.
// 15. A step skipped by --skip is labelled as a dry-run skip.

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
	e.site.Serve("/sitemap.xml", 200, "application/xml", fakes.URLSet("https://example.com/", "https://example.com/a"))
	sb.Env["SITEMAP_FETCH_BASE"] = e.site.URL
	list := e.urlFile(t, "https://example.com/", "https://example.com/a")
	return &launchRig{sb: sb, g: g, cf: cf, b: b, n: e.n, site: e.site, keyDir: e.keyDir, args: []string{
		"launch", "example.com", "--sitemap", "https://example.com/sitemap.xml", "--cloudflare-zone", "auto",
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
		if got[step]["status"] != result {
			t.Fatalf("%s = %v (%v), want %s", step, got[step]["status"], got[step]["detail"], result)
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
	wantSteps(t, results(t, r), map[string]string{"google verify": "fail", "google sitemap": "skipped", "bing verify": "pass", "bing sitemap": "pass", "indexnow": "pass"})
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
	wantSteps(t, got, map[string]string{"google verify": "pass", "google sitemap": "pass", "bing verify": "fail", "bing sitemap": "skipped", "indexnow": "pass"})
	wantContains(t, "detail", got["bing verify"]["detail"].(string), "BING_WEBMASTER_API_KEY")
}

func TestLaunchSkipsAProvider(t *testing.T) {
	l := newLaunchRig(t)
	r := l.run("--json", "--skip", "bing,indexnow")
	wantExit(t, r, 0)
	wantSteps(t, results(t, r), map[string]string{"google verify": "pass", "google sitemap": "pass", "bing verify": "skipped", "bing sitemap": "skipped", "indexnow": "skipped"})
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

// dryRunSteps are the five steps, in order.
var dryRunSteps = []string{"google verify", "google sitemap", "bing verify", "bing sitemap", "indexnow"}

func TestLaunchDryRunNeverReportsAPass(t *testing.T) {
	l := newLaunchRig(t)
	r := l.run("--dry-run", "--json")
	wantExit(t, r, 0)
	got := results(t, r)
	for _, name := range dryRunSteps {
		if got[name]["status"] != "skipped" {
			t.Fatalf("%s = %v, want skipped", name, got[name]["status"])
		}
		if w, _ := got[name]["would"].(string); w == "" {
			t.Fatalf("%s does not say what it would do: %v", name, got[name])
		}
	}
	v := r.JSON(t)
	if v["verified"] != false || v["dry_run"] != true || v["summary"] != "dry run: nothing was changed or verified" {
		t.Fatalf("the JSON does not say that nothing was verified: %s", r.Stdout)
	}

	r = l.run("--dry-run")
	wantExit(t, r, 0)
	for _, name := range dryRunSteps {
		wantLine(t, r.Stdout, name, "skipped (dry-run)")
	}
	if strings.Contains(r.Stdout, "pass") {
		t.Fatalf("a dry run printed pass:\n%s", r.Stdout)
	}
	lines := strings.Split(strings.TrimSpace(r.Stdout), "\n")
	if last := lines[len(lines)-1]; last != "dry run: nothing was changed or verified" {
		t.Fatalf("the last line is %q", last)
	}
}

func TestLaunchDryRunRunsTheReadOnlyChecksForReal(t *testing.T) {
	l := newLaunchRig(t)
	r := l.run("--dry-run", "--json")
	wantExit(t, r, 0)
	if l.site.Hits("/"+inKey+".txt") == 0 {
		t.Fatal("the key file was not requested")
	}
	checks := map[string]string{}
	for _, name := range dryRunSteps {
		list, _ := results(t, r)[name]["checks"].([]any)
		for _, c := range list {
			m := c.(map[string]any)
			checks[name+"/"+m["name"].(string)] = m["status"].(string)
		}
	}
	for _, want := range []string{"google verify/google login", "google verify/cloudflare token", "bing verify/bing key", "indexnow/indexnow key file"} {
		if checks[want] != "pass" {
			t.Fatalf("check %s = %q, want pass\n%v", want, checks[want], checks)
		}
	}
	if len(l.cf.Records("zone-1")) != 0 || l.g.HasSite("example.com") || len(l.n.Posts()) != 0 || l.b.Calls("AddSite") != 0 {
		t.Fatal("dry run changed something")
	}
	if files, _ := filepath.Glob(filepath.Join(l.keyDir, "*")); len(files) != 0 {
		t.Fatalf("dry run wrote %v", files)
	}
	if strings.Contains(r.Stderr, "created: true") {
		t.Fatalf("a dry run says it created a record:\n%s", r.Stderr)
	}
}

func TestLaunchDryRunFailsWhenACheckFails(t *testing.T) {
	cases := []struct {
		name   string
		break_ func(*launchRig)
		want   map[string]string
		detail [2]string // step, text in its detail
	}{
		{"no Bing key", func(l *launchRig) { delete(l.sb.Env, "BING_WEBMASTER_API_KEY") },
			map[string]string{"google verify": "skipped", "bing verify": "fail", "bing sitemap": "skipped", "indexnow": "skipped"},
			[2]string{"bing verify", "BING_WEBMASTER_API_KEY"}},
		{"no Google login", func(l *launchRig) { delete(l.sb.Env, "GOOGLE_ACCESS_TOKEN") },
			map[string]string{"google verify": "fail", "google sitemap": "skipped", "bing verify": "skipped"},
			[2]string{"google verify", "not logged in to Google"}},
		{"no Cloudflare token", func(l *launchRig) { delete(l.sb.Env, "CLOUDFLARE_API_TOKEN") },
			map[string]string{"google verify": "fail", "bing verify": "fail", "indexnow": "skipped"},
			[2]string{"google verify", "CLOUDFLARE_API_TOKEN"}},
		{"key file is missing", func(l *launchRig) { l.site.Serve("/"+inKey+".txt", 404, "text/html", "missing") },
			map[string]string{"google verify": "skipped", "bing verify": "skipped", "indexnow": "fail"},
			[2]string{"indexnow", "not reachable"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := newLaunchRig(t)
			c.break_(l)
			r := l.run("--dry-run", "--json")
			wantExit(t, r, 1)
			got := results(t, r)
			wantSteps(t, got, c.want)
			wantContains(t, "detail", got[c.detail[0]]["detail"].(string), c.detail[1])
			if r.JSON(t)["ok"] != false {
				t.Fatalf("ok is not false: %s", r.Stdout)
			}
			text := l.run("--dry-run")
			wantExit(t, text, 1)
			wantContains(t, "stdout", text.Stdout, "dry run: nothing was changed or verified")
			if len(l.n.Posts()) != 0 || len(l.cf.Records("zone-1")) != 0 {
				t.Fatal("dry run changed something")
			}
		})
	}
}

func TestLaunchDryRunNamesAKeyFileThatDoesNotExistYet(t *testing.T) {
	l := newLaunchRig(t)
	if err := os.Remove(filepath.Join(l.sb.ConfigDir, "indexnow-example.com.json")); err != nil {
		t.Fatal(err)
	}
	r := l.run("--dry-run", "--json")
	wantExit(t, r, 0)
	wantSteps(t, results(t, r), map[string]string{"indexnow": "skipped"})
	wantContains(t, "detail", results(t, r)["indexnow"]["detail"].(string), "no key saved yet")
	if l.site.Hits("/"+inKey+".txt") != 0 {
		t.Fatal("the key file was requested with no saved key")
	}
}

func TestLaunchDryRunLabelsASkipFlagAsASkip(t *testing.T) {
	l := newLaunchRig(t)
	r := l.run("--dry-run", "--skip", "bing")
	wantExit(t, r, 0)
	wantLine(t, r.Stdout, "bing verify", "skipped by --skip bing")
	if strings.Contains(strings.Split(r.Stdout, "\n")[3], "(dry-run)") {
		t.Fatalf("a --skip row is labelled as a dry-run skip:\n%s", r.Stdout)
	}
	wantLine(t, r.Stdout, "google verify", "skipped (dry-run)")
}

// wantLine finds the row that starts with step and checks that it holds text.
func wantLine(t *testing.T, stdout, step, text string) {
	t.Helper()
	for _, line := range strings.Split(stdout, "\n") {
		if strings.HasPrefix(line, step+" ") {
			wantContains(t, "row "+step, line, text)
			return
		}
	}
	t.Fatalf("no row for %s:\n%s", step, stdout)
}
