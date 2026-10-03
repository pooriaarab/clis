package cli

import (
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/pooriaarab/clis/submit/internal/launchingnext"
	"github.com/pooriaarab/clis/submit/internal/sites"
	"github.com/pooriaarab/clis/submit/internal/store"
)

func launchingNextCmd(env *Env) *cobra.Command {
	var siteIDs []string
	var gap time.Duration
	cmd := &cobra.Command{
		Use:   "launching-next --sites sites.yaml",
		Short: "Submit every site to Launching Next, the one scriptable target",
		Long: `Submit every site to Launching Next (plain form POST + math check).

This is the only directory the CLI posts to. With --dry-run the command
shows the exact POST bodies without posting. Without it the command
re-reads the form fresh per site, aborts when the shape changed, waits
--gap between posts, and records each result in the tracker.`,
		Args: cobra.NoArgs,
	}
	load := sitesFlag(env, cmd)
	cmd.Flags().StringSliceVar(&siteIDs, "site", nil, "only these site ids (repeatable, default all)")
	cmd.Flags().DurationVar(&gap, "gap", launchingnext.MinGap, "delay between posts (minimum 25s on the real site)")
	cmd.RunE = run(func([]string) error {
		list, _, err := load()
		if err != nil {
			return err
		}
		picked, err := filterSites(list, siteIDs)
		if err != nil {
			return err
		}
		base, overridden := launchingnext.BaseURL(env.Getenv)
		if !overridden && gap < launchingnext.MinGap {
			return usagef("--gap is %s; the real site needs at least %s between posts", gap, launchingnext.MinGap)
		}
		if gap < 0 {
			return usagef("--gap must not be negative (got %s)", gap)
		}
		if env.DryRun {
			return launchingNextDryRun(env, base, picked)
		}
		return launchingNextRun(env, base, picked, gap)
	})
	return cmd
}

// postBody builds one site's exact POST body for a solved math answer.
func postBody(s sites.Site, answer string) url.Values {
	return launchingnext.Body(launchingnext.Input{
		Name: s.Name, URL: s.URL, Headline: s.Headline, Full: s.Full2200,
		Tags: strings.Join(s.Tags, ", "), Funding: s.Funding,
		Marketing: s.Marketing, Founder: s.Founder, Email: s.Email,
	}, answer)
}

func launchingNextDryRun(env *Env, base string, picked []sites.Site) error {
	page, err := launchingnext.Fetch(base)
	if err != nil {
		return err
	}
	form, err := launchingnext.Parse(page)
	if err != nil {
		return err
	}
	type sub struct {
		Site     string `json:"site"`
		Question string `json:"question"`
		Body     string `json:"body"`
	}
	var subs []sub
	for _, s := range picked {
		subs = append(subs, sub{Site: s.ID, Question: form.Question, Body: postBody(s, form.Answer).Encode()})
	}
	if env.P.JSON {
		return env.P.Object(map[string]any{"ok": true, "dry_run": true, "submit_url": base + launchingnext.SubmitPath, "submissions": subs})
	}
	env.P.Linef("POST %s", base+launchingnext.SubmitPath)
	for _, s := range subs {
		env.P.Linef("site %s (%s): %s", s.Site, s.Question, s.Body)
	}
	env.P.Linef("dry run: nothing was posted")
	return nil
}

func launchingNextRun(env *Env, base string, picked []sites.Site, gap time.Duration) error {
	path, err := storePath(env)
	if err != nil {
		return err
	}
	f, err := store.Load(path)
	if err != nil {
		return err
	}
	today := time.Now().Format("2006-01-02")
	if n := f.SubmittedOn(today, ""); n+len(picked) > store.DailyLimit {
		return usagef("daily limit: %d submission(s) already dated %s, %d more would pass the %d/day max; narrow --site or wait a day",
			n, today, len(picked), store.DailyLimit)
	}
	type outcome struct {
		Site       string `json:"site"`
		Status     string `json:"status"`
		HTTPStatus int    `json:"http_status"`
		Detail     string `json:"detail"`
	}
	var outcomes []outcome
	for i, s := range picked {
		if i > 0 {
			time.Sleep(gap)
		}
		// Fresh form per submit: the shape check must see what the POST hits.
		page, err := launchingnext.Fetch(base)
		if err != nil {
			return err
		}
		form, err := launchingnext.Parse(page)
		if err != nil {
			return err // form shape changed: abort, record nothing further
		}
		code, respPage, err := launchingnext.Post(base, postBody(s, form.Answer))
		if err != nil {
			return err
		}
		oc := outcome{Site: s.ID, HTTPStatus: code}
		if code == 200 && launchingnext.LooksLikeSuccess(respPage) {
			if _, err := f.Set(store.SetInput{
				Site: s.ID, SiteURL: s.URL, Directory: "launching-next",
				Status: store.Submitted, Date: today,
				Note: "launching-next auto-submit, http 200, thank-you page",
				Now:  time.Now(),
			}); err != nil {
				return usagef("%s", err)
			}
			oc.Status = store.Submitted
			oc.Detail = "accepted (thank-you page)"
		} else {
			oc.Status = "failed"
			oc.Detail = "response did not read as an acceptance; left untracked, check by hand"
		}
		outcomes = append(outcomes, oc)
	}
	if err := f.Save(path); err != nil {
		return err
	}
	if env.P.JSON {
		return env.P.Object(map[string]any{"ok": true, "submit_url": base + launchingnext.SubmitPath, "results": outcomes})
	}
	for _, o := range outcomes {
		env.P.Linef("%s: %s (http %d) %s", o.Site, o.Status, o.HTTPStatus, o.Detail)
	}
	return nil
}
