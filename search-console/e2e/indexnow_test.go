package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pooriaarab/clis/search-console/e2e/fakes"
)

// Ways `indexnow submit` can fail, listed before the code was written:
//  1. Neither or both of --urls and --from-sitemap are given: exit 2, no request.
//  2. Blank lines, comments or duplicates in the list reach IndexNow.
//  3. A URL is on another host: IndexNow refuses the batch, so stop with exit 2.
//  4. The key changes on every run, so the key file on the site goes stale.
//  5. The key file is missing (404) or a soft 404 page: exit 1 before any POST.
//  6. More than 10,000 URLs go in one request.
//  7. IndexNow answers 202: that is success. 400, 403, 422, 429: exit 1 with the reason,
//     and the batches after the failed one are not sent.
//  8. --dry-run writes the key file, saves the key or sends a request.
//  9. The key directory does not exist: fail before any request.
// 10. A sitemap index is not followed, or an unreadable sitemap gives an empty run.

const inKey = "abcdef0123456789abcdef0123456789"

type inEnv struct {
	sb      *Sandbox
	n       *fakes.IndexNow
	site    *fakes.Site
	keyDir  string
	keyLoc  string
	workDir string
}

func indexnowSandbox(t *testing.T) *inEnv {
	sb := newSandbox(t)
	n := fakes.NewIndexNow(t)
	site := fakes.NewSite(t)
	sb.Alias(n.URL, "http://indexnow.fake")
	sb.Alias(site.URL, "http://site.fake")
	sb.Env["INDEXNOW_API_BASE"] = n.URL + "/indexnow"
	e := &inEnv{sb: sb, n: n, site: site, keyDir: t.TempDir(), workDir: t.TempDir(), keyLoc: site.URL + "/" + inKey + ".txt"}
	sb.Alias(e.keyDir, "$KEYDIR")
	sb.Alias(e.workDir, "$WORK")
	return e
}

// seedKey saves a known key and serves its key file, as a deployed site would.
func (e *inEnv) seedKey(t *testing.T) {
	data, _ := json.Marshal(map[string]string{"key": inKey})
	if err := os.WriteFile(filepath.Join(e.sb.ConfigDir, "indexnow-example.com.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	e.site.Serve("/"+inKey+".txt", 200, "text/plain", inKey+"\n")
}

func (e *inEnv) urlFile(t *testing.T, lines ...string) string {
	p := filepath.Join(e.workDir, "urls.txt")
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func (e *inEnv) submit(args ...string) Result {
	base := []string{"indexnow", "submit", "example.com", "--key-dir", e.keyDir, "--key-location", e.keyLoc}
	return e.sb.Run(append(base, args...)...)
}

func TestIndexNowSubmitCleansTheList(t *testing.T) {
	e := indexnowSandbox(t)
	e.seedKey(t)
	f := e.urlFile(t, "# new pages", "https://example.com/", "", "https://example.com/a", "https://example.com/a", "  https://example.com/b  ")
	r := e.submit("--urls", f, "--json")
	wantExit(t, r, 0)
	posts := e.n.Posts()
	if len(posts) != 1 || len(posts[0].URLs) != 3 || posts[0].Host != "example.com" || posts[0].Key != inKey || posts[0].KeyLocation != e.keyLoc {
		t.Fatalf("unexpected posts: %+v", posts)
	}
	if v := r.JSON(t); v["urls"] != float64(3) || v["batches"] != float64(1) {
		t.Fatalf("unexpected result: %s", r.Stdout)
	}
	wantContains(t, "stderr", r.Stderr, filepath.Join(e.keyDir, inKey+".txt"))
	if got, _ := os.ReadFile(filepath.Join(e.keyDir, inKey+".txt")); string(got) != inKey {
		t.Fatalf("key file holds %q", got)
	}
}

func TestIndexNowNeedsOneSource(t *testing.T) {
	e := indexnowSandbox(t)
	e.seedKey(t)
	f := e.urlFile(t, "https://example.com/")
	for _, args := range [][]string{{}, {"--urls", f, "--from-sitemap=" + e.site.URL + "/sitemap.xml"}} {
		wantExit(t, e.submit(args...), 2)
	}
	if len(e.n.Posts()) != 0 {
		t.Fatal("a request was sent")
	}
}

func TestIndexNowRejectsOtherHosts(t *testing.T) {
	e := indexnowSandbox(t)
	e.seedKey(t)
	r := e.submit("--urls", e.urlFile(t, "https://example.com/a", "https://other.example/b"))
	wantExit(t, r, 2)
	wantContains(t, "stderr", r.Stderr, "https://other.example/b")
	if len(e.n.Posts()) != 0 {
		t.Fatal("a request was sent")
	}
}

func TestIndexNowKeepsOneKeyAndStopsWhenUnreachable(t *testing.T) {
	e := indexnowSandbox(t)
	f := e.urlFile(t, "https://example.com/")
	e.keyLoc = e.site.URL + "/missing.txt"
	var first string
	for i := 0; i < 2; i++ {
		r := e.submit("--urls", f)
		wantExit(t, r, 1)
		wantContains(t, "stderr", r.Stderr, "not reachable")
		files, _ := filepath.Glob(filepath.Join(e.keyDir, "*.txt"))
		if len(files) != 1 {
			t.Fatalf("key files = %v, want one", files)
		}
		if i == 0 {
			first = files[0]
		} else if files[0] != first {
			t.Fatalf("the key changed between runs: %s then %s", first, files[0])
		}
	}
	if len(e.n.Posts()) != 0 {
		t.Fatal("a request was sent while the key file was unreachable")
	}
	saved, _ := os.ReadFile(filepath.Join(e.sb.ConfigDir, "indexnow-example.com.json"))
	if !strings.Contains(string(saved), strings.TrimSuffix(filepath.Base(first), ".txt")) {
		t.Fatalf("the saved key differs from the key file: %s", saved)
	}
}

func TestIndexNowSoftNotFoundKeyFile(t *testing.T) {
	e := indexnowSandbox(t)
	e.seedKey(t)
	e.site.Serve("/"+inKey+".txt", 200, "text/html", "<html>Page not found</html>")
	r := e.submit("--urls", e.urlFile(t, "https://example.com/"))
	wantExit(t, r, 1)
	wantContains(t, "stderr", r.Stderr, "does not hold the key")
	if len(e.n.Posts()) != 0 {
		t.Fatal("a request was sent")
	}
}

func numbered(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("https://example.com/p/%d", i)
	}
	return out
}

func TestIndexNowSplitsIntoBatchesOf10000(t *testing.T) {
	e := indexnowSandbox(t)
	e.seedKey(t)
	wantExit(t, e.submit("--urls", e.urlFile(t, numbered(25000)...)), 0)
	var sizes []int
	for _, p := range e.n.Posts() {
		sizes = append(sizes, len(p.URLs))
	}
	if fmt.Sprint(sizes) != "[10000 10000 5000]" {
		t.Fatalf("batch sizes = %v", sizes)
	}
}

func TestIndexNowAcceptedAndRefusedAnswers(t *testing.T) {
	e := indexnowSandbox(t)
	e.seedKey(t)
	e.n.Answer[0] = 202
	wantExit(t, e.submit("--urls", e.urlFile(t, "https://example.com/")), 0)
	for status, want := range map[int]string{400: "malformed", 403: "key is not valid", 422: "does not belong", 429: "too many requests"} {
		e := indexnowSandbox(t)
		e.seedKey(t)
		e.n.Answer[1] = status // the second of three batches fails
		r := e.submit("--urls", e.urlFile(t, numbered(25000)...))
		wantExit(t, r, 1)
		wantContains(t, "stderr", r.Stderr, want)
		wantContains(t, "stderr", r.Stderr, "batch 2 of 3 failed after 1 batch(es)")
		if len(e.n.Posts()) != 2 {
			t.Fatalf("status %d: posts = %d, want 2", status, len(e.n.Posts()))
		}
	}
}

func TestIndexNowDryRunWritesAndSendsNothing(t *testing.T) {
	e := indexnowSandbox(t)
	r := e.submit("--urls", e.urlFile(t, "https://example.com/"), "--dry-run")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "/indexnow")
	if files, _ := filepath.Glob(filepath.Join(e.keyDir, "*")); len(files) != 0 {
		t.Fatalf("dry run wrote %v", files)
	}
	if files, _ := filepath.Glob(filepath.Join(e.sb.ConfigDir, "*")); len(files) != 0 {
		t.Fatalf("dry run saved %v", files)
	}
	if len(e.n.Posts()) != 0 {
		t.Fatal("dry run sent a request")
	}
}

func TestIndexNowMissingKeyDir(t *testing.T) {
	e := indexnowSandbox(t)
	e.seedKey(t)
	e.keyDir = filepath.Join(e.keyDir, "nope")
	wantExit(t, e.submit("--urls", e.urlFile(t, "https://example.com/")), 1)
	if len(e.n.Posts()) != 0 {
		t.Fatal("a request was sent")
	}
}

func TestIndexNowFromSitemapIndex(t *testing.T) {
	e := indexnowSandbox(t)
	e.seedKey(t)
	e.site.Serve("/a.xml", 200, "application/xml", fakes.URLSet("https://example.com/", "https://example.com/a"))
	e.site.Serve("/b.xml", 200, "application/xml", fakes.URLSet("https://example.com/a", "https://example.com/b"))
	e.site.Serve("/index.xml", 200, "application/xml", `<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><sitemap><loc>`+
		e.site.URL+`/a.xml</loc></sitemap><sitemap><loc>`+e.site.URL+`/b.xml</loc></sitemap></sitemapindex>`)
	wantExit(t, e.submit("--from-sitemap="+e.site.URL+"/index.xml"), 0)
	if p := e.n.Posts(); len(p) != 1 || len(p[0].URLs) != 3 {
		t.Fatalf("posts = %+v, want one post with 3 URLs", p)
	}
}

func TestIndexNowUnreadableSitemap(t *testing.T) {
	e := indexnowSandbox(t)
	e.seedKey(t)
	r := e.submit("--from-sitemap=" + e.site.URL + "/missing.xml")
	wantExit(t, r, 1)
	wantContains(t, "stderr", r.Stderr, "cannot read the sitemap")
	if len(e.n.Posts()) != 0 {
		t.Fatal("a request was sent")
	}
}

func TestIndexNowDefaultSitemapInDryRun(t *testing.T) {
	e := indexnowSandbox(t)
	r := e.submit("--from-sitemap", "--dry-run")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "https://example.com/sitemap.xml")
	if len(e.n.Posts()) != 0 || e.site.Hits("/sitemap.xml") != 0 {
		t.Fatal("dry run sent a request")
	}
}
