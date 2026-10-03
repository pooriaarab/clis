package e2e

import (
	"net/url"
	"testing"

	"github.com/pooriaarab/clis/submit/e2e/fakes"
)

func TestLaunchingNextDryRunShowsBodies(t *testing.T) {
	ln := fakes.NewLaunchingNext("ok")
	defer ln.Server.Close()
	sb := newSandbox(t)
	sb.Alias(ln.Server.URL, "$LN")
	sb.Env["SUBMIT_LAUNCHINGNEXT_BASE"] = ln.Server.URL
	sites := sb.writeSites()
	sb.Run("track", "init")

	r := sb.Run("launching-next", "--sites", sites, "--dry-run")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "POST")
	wantContains(t, "stdout", r.Stdout, "What is 2+3?")
	wantContains(t, "stdout", r.Stdout, "math=5")
	wantContains(t, "stdout", r.Stdout, "startupname=Widget+Directory")
	wantContains(t, "stdout", r.Stdout, "formSubmit=Submit+Startup")
	wantContains(t, "stdout", r.Stdout, "dry run: nothing was posted")
	if n := ln.PostCount(); n != 0 {
		t.Fatalf("dry run posted %d time(s)", n)
	}
	if vals := ln.FirstPost(); vals != nil {
		t.Fatalf("dry run posted: %v", vals)
	}

	// Nothing recorded in the tracker either.
	r = sb.Run("track", "list")
	wantContains(t, "stdout", r.Stdout, "no rows")

	// The newsletter opt-in is never sent.
	r = sb.Run("launching-next", "--sites", sites, "--dry-run", "--json")
	wantExit(t, r, 0)
	v := r.JSON(t)
	subs, _ := v["submissions"].([]any)
	if len(subs) != 2 {
		t.Fatalf("submissions = %d, want 2", len(subs))
	}
	for _, s := range subs {
		body, _ := url.ParseQuery(s.(map[string]any)["body"].(string))
		if body.Get("newsletter_optin") != "" {
			t.Errorf("dry-run body opts into the newsletter: %v", body)
		}
		if body.Get("math") != "5" || body.Get("formSubmit") != "Submit Startup" {
			t.Errorf("dry-run body misses solved math or submit button: %v", body)
		}
	}
}

func TestLaunchingNextPostsAndTracks(t *testing.T) {
	ln := fakes.NewLaunchingNext("ok")
	defer ln.Server.Close()
	sb := newSandbox(t)
	sb.Alias(ln.Server.URL, "$LN")
	sb.Env["SUBMIT_LAUNCHINGNEXT_BASE"] = ln.Server.URL
	sites := sb.writeSites()
	sb.Run("track", "init")

	r := sb.Run("launching-next", "--sites", sites, "--site", "widgets", "--gap", "1s")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "widgets: submitted")
	if n := ln.PostCount(); n != 1 {
		t.Fatalf("posts = %d, want 1", n)
	}
	post := ln.FirstPost()
	if post.Get("startupname") != "Widget Directory" || post.Get("math") != "5" {
		t.Errorf("posted body = %v", post)
	}

	r = sb.Run("track", "list", "--json")
	v := r.JSON(t)
	recs, _ := v["records"].([]any)
	if len(recs) != 1 {
		t.Fatalf("records = %d, want 1", len(recs))
	}
	rec := recs[0].(map[string]any)
	if rec["site"] != "widgets" || rec["directory"] != "launching-next" || rec["status"] != "submitted" {
		t.Errorf("record = %v", rec)
	}
}

func TestLaunchingNextAbortsOnChangedForm(t *testing.T) {
	ln := fakes.NewLaunchingNext("changed")
	defer ln.Server.Close()
	sb := newSandbox(t)
	sb.Alias(ln.Server.URL, "$LN")
	sb.Env["SUBMIT_LAUNCHINGNEXT_BASE"] = ln.Server.URL
	sites := sb.writeSites()
	sb.Run("track", "init")

	r := sb.Run("launching-next", "--sites", sites, "--gap", "1s")
	wantExit(t, r, 1)
	wantContains(t, "output", r.Stdout+r.Stderr, "form shape changed")
	if n := ln.PostCount(); n != 0 {
		t.Fatalf("posted %d time(s) despite changed form", n)
	}
	r = sb.Run("track", "list")
	wantContains(t, "stdout", r.Stdout, "no rows")
}

func TestLaunchingNextRefusesSmallGapOnRealSite(t *testing.T) {
	sb := newSandbox(t)
	sites := sb.writeSites()
	// No SUBMIT_LAUNCHINGNEXT_BASE: the real site, so the 25s floor applies.
	// The gap check runs before any fetch, so this test makes no requests.
	r := sb.Run("launching-next", "--sites", sites, "--gap", "1s")
	wantExit(t, r, 2)
	wantContains(t, "output", r.Stdout+r.Stderr, "25s")
}

func TestLaunchingNextRespectsDailyLimit(t *testing.T) {
	ln := fakes.NewLaunchingNext("ok")
	defer ln.Server.Close()
	sb := newSandbox(t)
	sb.Env["SUBMIT_LAUNCHINGNEXT_BASE"] = ln.Server.URL
	sites := sb.writeSites()
	sb.Run("track", "init")
	for _, d := range []string{"crunchbase", "saashub", "uneed", "startupbase", "peerpush",
		"f6s", "wellfound", "indie-hackers", "linkcentre", "betalist"} {
		r := sb.Run("track", "set", "--site", "widgets", "--directory", d, "--status", "submitted")
		wantExit(t, r, 0)
	}
	r := sb.Run("launching-next", "--sites", sites, "--site", "widgets", "--gap", "1s")
	wantExit(t, r, 2)
	wantContains(t, "output", r.Stdout+r.Stderr, "daily limit")
	if n := ln.PostCount(); n != 0 {
		t.Fatalf("posted %d time(s) despite full day", n)
	}
}
