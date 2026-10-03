package e2e

import (
	"testing"

	"github.com/pooriaarab/clis/submit/e2e/fakes"
)

const checkSite = "https://widgets.example.com"

func seedCheckRow(t *testing.T, sb *Sandbox, listingURL string) {
	t.Helper()
	sb.Run("track", "init")
	r := sb.Run("track", "set", "--site", "widgets", "--site-url", checkSite,
		"--directory", "crunchbase", "--status", "submitted", "--listing-url", listingURL)
	wantExit(t, r, 0)
}

func TestCheckReportsFollow(t *testing.T) {
	l := fakes.NewListing(checkSite)
	defer l.Server.Close()
	sb := newSandbox(t)
	sb.Alias(l.Server.URL, "$LISTING")
	seedCheckRow(t, sb, l.URL("/follow"))

	r := sb.Run("check", "--site", "widgets", "--directory", "crunchbase")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "status: 200")
	wantContains(t, "stdout", r.Stdout, "link: found (follow)")
	wantContains(t, "stdout", r.Stdout, "hint: page is reachable and links (follow) to the site; confirm with: site:127.0.0.1 widgets")

	r = sb.Run("check", "--site", checkSite, "--directory", "Crunchbase", "--json")
	wantExit(t, r, 0)
	v := r.JSON(t)
	c := v["check"].(map[string]any)
	if c["link_rel"] != "follow" || c["link_found"] != true {
		t.Errorf("check = %v", c)
	}
}

func TestCheckReportsNofollow(t *testing.T) {
	l := fakes.NewListing(checkSite)
	defer l.Server.Close()
	sb := newSandbox(t)
	sb.Alias(l.Server.URL, "$LISTING")
	seedCheckRow(t, sb, l.URL("/nofollow"))

	r := sb.Run("check", "--site", "widgets", "--directory", "crunchbase")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "link: found (nofollow)")
}

func TestCheckFollowsRedirects(t *testing.T) {
	l := fakes.NewListing(checkSite)
	defer l.Server.Close()
	sb := newSandbox(t)
	sb.Alias(l.Server.URL, "$LISTING")
	seedCheckRow(t, sb, l.URL("/redir"))

	r := sb.Run("check", "--site", "widgets", "--directory", "crunchbase")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "redirects: 1")
	wantContains(t, "stdout", r.Stdout, "link: found (follow)")
}

func TestCheckReportsMissingLink(t *testing.T) {
	l := fakes.NewListing(checkSite)
	defer l.Server.Close()
	sb := newSandbox(t)
	sb.Alias(l.Server.URL, "$LISTING")
	seedCheckRow(t, sb, l.URL("/missing"))

	r := sb.Run("check", "--site", "widgets", "--directory", "crunchbase")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "link: not found")
}

func TestCheckNeedsARow(t *testing.T) {
	sb := newSandbox(t)
	sb.Run("track", "init")
	r := sb.Run("check", "--site", "widgets", "--directory", "crunchbase")
	wantExit(t, r, 2)
	wantContains(t, "output", r.Stdout+r.Stderr, "no tracker row")
}

func TestCheckDryRunFetchesNothing(t *testing.T) {
	l := fakes.NewListing(checkSite)
	defer l.Server.Close()
	sb := newSandbox(t)
	seedCheckRow(t, sb, l.URL("/follow"))

	r := sb.Run("check", "--site", "widgets", "--directory", "crunchbase", "--dry-run")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "would fetch")
}
