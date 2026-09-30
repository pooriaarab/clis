package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
}

type stepResult struct {
	Step   string `json:"step"`
	Result string `json:"result"`
	Exit   int    `json:"exit"`
	Detail string `json:"detail"`
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
first failed step.`,
		Args: cobra.ExactArgs(1),
		RunE: run(func(args []string) error {
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
			steps := []step{
				{"google verify", "google", "", append([]string{"google", "verify", d}, verify...)},
				{"google sitemap", "google", "google verify", append([]string{"google", "sitemap", "submit", d, sitemapURL}, common...)},
				{"bing verify", "bing", "", append([]string{"bing", "verify", d}, verify...)},
				{"bing sitemap", "bing", "bing verify", []string{"bing", "sitemap", "submit", d, sitemapURL}},
				{"indexnow", "indexnow", "", inArgs},
			}
			var rows []stepResult
			passed := map[string]bool{}
			firstFail := 0
			for _, s := range steps {
				r := stepResult{Step: s.name}
				switch {
				case skipped[s.group]:
					r.Result, r.Detail = "skip", "skipped by --skip "+s.group
				case s.needs != "" && !passed[s.needs]:
					r.Result, r.Detail = "skip", s.needs+" did not pass"
				default:
					r = env.runStep(s)
				}
				passed[s.name] = r.Result == "pass"
				if r.Result == "fail" && firstFail == 0 {
					firstFail = r.Exit
				}
				rows = append(rows, r)
			}
			if err := env.Emit(map[string]any{"ok": firstFail == 0, "domain": d, "steps": rows}, func() {
				table := [][]string{{"STEP", "RESULT", "DETAIL"}}
				for _, r := range rows {
					table = append(table, []string{r.Step, r.Result, r.Detail})
				}
				env.P.Table(table)
			}); err != nil {
				return err
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
	code := Main(args, &out, e.P.Err, e.Getenv)
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
	r := stepResult{Step: s.name, Result: "pass", Exit: code}
	switch {
	case code != 0:
		r.Result, r.Detail = "fail", m.Error
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
