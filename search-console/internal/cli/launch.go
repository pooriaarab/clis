package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/pooriaarab/clis/search-console/internal/httpx"
)

// step is one command that launch runs. A step that needs another is skipped
// when that one did not pass.
type step struct {
	name, group, needs string
	args               []string
	// would and checks are for --dry-run. would says what the step does. checks
	// are the read-only tests a dry run may run for real.
	would  string
	checks []check
}

// stepResult is one row of the table. Status is pass, fail or skipped.
type stepResult struct {
	Step   string        `json:"step"`
	Status string        `json:"status"`
	Exit   int           `json:"exit"`
	Detail string        `json:"detail"`
	Would  string        `json:"would,omitempty"`
	Checks []checkResult `json:"checks,omitempty"`
	// dryRun marks a step that was skipped because of --dry-run.
	dryRun bool
}

// checkResult is the real result of one read-only check in a dry run.
type checkResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

// label is the status as the table shows it.
func (r stepResult) label() string {
	if r.dryRun {
		return r.Status + " (dry-run)"
	}
	return r.Status
}

func launchCmd(env *Env) *cobra.Command {
	var sitemapURL, zone, keyDir, keyLocation, indexnowURLs string
	var wait, interval time.Duration
	var skip []string
	cmd := &cobra.Command{
		Use:   "launch <domain> --sitemap <url>",
		Short: "Run every launch step and print one pass/fail table",
		Long: `Run these steps in order: google verify, google sitemap submit, bing verify,
bing sitemap submit and indexnow submit. A step that fails does not stop the others,
but a sitemap step is skipped when its verify step failed. A step that is already
done passes, so you can run the command again. The exit code is the code of the
first failed step.

With --dry-run, no step runs and every step shows "skipped (dry-run)" with what it
would do. Read-only checks still run for real: the credentials and the IndexNow
key file. A failed check fails its step and the exit
code. A dry run never reports that a step passed.`,
		Args: cobra.ExactArgs(1),
		RunE: run(func(args []string) error {
			if err := checkPolling(wait, interval); err != nil {
				return err
			}
			d, err := normalizeDomain(args[0])
			if err != nil {
				return err
			}
			skipped := map[string]bool{}
			for _, g := range skip {
				if g != "google" && g != "bing" && g != "indexnow" {
					return &ExitError{Code: ExitUsage, Err: fmt.Errorf("--skip %q: use google, bing or indexnow", g)}
				}
				skipped[g] = true
			}
			common := []string{"--interval", interval.String()}
			verify := append([]string{"--wait", wait.String()}, common...)
			if zone != "" {
				verify = append(verify, "--cloudflare-zone", zone)
			}
			source := []string{"--from-sitemap=" + sitemapURL}
			if indexnowURLs != "" {
				source = []string{"--urls", indexnowURLs}
			}
			inArgs := append([]string{"indexnow", "submit", d, "--key-dir", keyDir}, source...)
			if keyLocation != "" {
				inArgs = append(inArgs, "--key-location", keyLocation)
			}
			var verifyChecks []check
			if zone != "" {
				verifyChecks = append(verifyChecks, env.cloudflareTokenCheck())
			}
			record := "print the DNS record and wait for you to add it"
			if zone != "" {
				record = "publish the DNS record to Cloudflare zone " + zone
			}
			steps := []step{
				{"google verify", "google", "", append([]string{"google", "verify", d}, verify...),
					"get a TXT token from Google, " + record + ", wait for Google to see it, then add the site",
					append([]check{env.googleLoginCheck()}, verifyChecks...)},
				{"google sitemap", "google", "google verify", append([]string{"google", "sitemap", "submit", d, sitemapURL}, common...),
					"submit " + sitemapURL + " to Google Search Console", nil},
				{"bing verify", "bing", "", append([]string{"bing", "verify", d}, verify...),
					"add the site to Bing, " + record + ", wait for Bing to verify it",
					append([]check{env.bingKeyCheck()}, verifyChecks...)},
				{"bing sitemap", "bing", "bing verify", []string{"bing", "sitemap", "submit", d, sitemapURL},
					"submit " + sitemapURL + " to Bing Webmaster Tools", nil},
				{"indexnow", "indexnow", "", inArgs,
					"write the key file to " + keyDir + ", check that it is reachable, then post the URLs to IndexNow",
					[]check{env.keyFileCheck(d, keyLocation)}},
			}
			var rows []stepResult
			passed := map[string]bool{}
			failed := map[string]bool{}
			firstFail := 0
			for _, s := range steps {
				r := stepResult{Step: s.name}
				switch {
				case skipped[s.group]:
					r.Status, r.Detail = statusSkipped, "skipped by --skip "+s.group
				case env.DryRun:
					// A dry run calls no step. A sub-command dry run only lists its unsent calls.
					_ = env.runStep(s)
					r = dryRunStep(s, s.needs != "" && failed[s.needs])
				case s.needs != "" && !passed[s.needs]:
					r.Status, r.Detail = statusSkipped, s.needs+" did not pass"
				default:
					r = env.runStep(s)
				}
				passed[s.name] = r.Status == statusPass
				failed[s.name] = r.Status == statusFail
				if r.Status == statusFail && firstFail == 0 {
					firstFail = r.Exit
				}
				rows = append(rows, r)
			}
			result := map[string]any{"ok": firstFail == 0, "domain": d, "steps": rows}
			if env.DryRun {
				// ok means that no check failed. Nothing was verified.
				result["verified"], result["summary"] = false, dryRunSummary
			}
			if err := env.Emit(result, func() {
				table := [][]string{{"STEP", "STATUS", "DETAIL"}}
				for _, r := range rows {
					table = append(table, []string{r.Step, r.label(), r.Detail})
				}
				env.P.Table(table)
			}); err != nil {
				return err
			}
			if env.DryRun && !env.P.JSON {
				env.P.Linef("%s", dryRunSummary)
			}
			if firstFail != 0 {
				return &ExitError{Code: firstFail, Err: errors.New("a launch step failed"), Printed: true}
			}
			return nil
		}),
	}
	cmd.Flags().StringVar(&sitemapURL, "sitemap", "", "absolute URL of the sitemap")
	_ = cmd.MarkFlagRequired("sitemap")
	cmd.Flags().StringVar(&zone, "cloudflare-zone", "", "Cloudflare zone id or auto: publish the DNS records there")
	cmd.Flags().DurationVar(&wait, "wait", 10*time.Minute, "how long each verify step waits for DNS")
	cmd.Flags().DurationVar(&interval, "interval", 5*time.Second, "delay between checks")
	cmd.Flags().StringVar(&keyDir, "key-dir", ".", "directory where the IndexNow key file is written")
	cmd.Flags().StringVar(&keyLocation, "key-location", "", "URL of the IndexNow key file")
	cmd.Flags().StringVar(&indexnowURLs, "indexnow-urls", "", "file with the IndexNow URLs, instead of the sitemap")
	cmd.Flags().StringSliceVar(&skip, "skip", nil, "skip a group: google, bing or indexnow")
	return cmd
}

// runStep runs one command in this process with --json and reads its result.
// The command writes its hints to stderr as it goes.
func (e *Env) runStep(s step) stepResult {
	args := append(append([]string{}, s.args...), "--json")
	if e.DryRun {
		args = append(args, "--dry-run")
	}
	var out bytes.Buffer
	stderr := e.P.Err
	if e.DryRun {
		stderr = io.Discard // the hints of a dry run say "created: true" for records that were not made
	}
	code := Main(args, &out, stderr, e.Getenv)
	var m struct {
		Error           string       `json:"error"`
		AlreadyVerified bool         `json:"already_verified"`
		Attempts        int          `json:"attempts"`
		URLs            int          `json:"urls"`
		Status          string       `json:"status"`
		Calls           []httpx.Call `json:"calls"`
		Pending         *bool        `json:"isPending"`
	}
	_ = json.Unmarshal(out.Bytes(), &m)
	e.Client.Calls = append(e.Client.Calls, m.Calls...)
	r := stepResult{Step: s.name, Status: statusPass, Exit: code}
	switch {
	case code != 0:
		r.Status, r.Detail = statusFail, m.Error
		if r.Detail == "" {
			r.Detail = fmt.Sprintf("exit code %d", code)
		}
		r.Detail, _, _ = strings.Cut(r.Detail, "\n")
	case m.AlreadyVerified:
		r.Detail = "already verified"
	case m.Attempts > 0:
		r.Detail = fmt.Sprintf("verified after %d attempt(s)", m.Attempts)
	case m.URLs > 0 && m.Status == "":
		r.Detail = fmt.Sprintf("%d URL(s) sent", m.URLs)
	case m.Status != "":
		r.Detail = "status " + m.Status
	case m.Pending != nil && *m.Pending:
		r.Detail = "submitted, Google is still reading it"
	default:
		r.Detail = "done"
	}
	return r
}
