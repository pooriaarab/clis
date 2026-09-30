package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/pooriaarab/clis/search-console/internal/config"
	"github.com/pooriaarab/clis/search-console/internal/google"
	"github.com/pooriaarab/clis/search-console/internal/httpx"
	"github.com/pooriaarab/clis/search-console/internal/indexnow"
)

const (
	statusPass    = "pass"
	statusFail    = "fail"
	statusSkipped = "skipped"

	// dryRunSummary ends every dry run. A dry run verifies nothing.
	dryRunSummary = "dry run: nothing was changed or verified"
)

// check is one read-only test that a dry run may run for real.
type check struct {
	name string
	run  func() (string, error)
}

// dryRunStep reports what a step would do. It never calls the step. It runs only
// the read-only checks, and the step fails when one check fails. A step that
// passes its checks is still skipped, because the action did not run.
func dryRunStep(s step, verifyFailed bool) stepResult {
	r := stepResult{Step: s.name, Status: statusSkipped, dryRun: true}
	var passed, failed []string
	for _, c := range s.checks {
		detail, err := c.run()
		if err != nil {
			first, _, _ := strings.Cut(err.Error(), "\n")
			failed = append(failed, c.name+": "+first)
			r.Checks = append(r.Checks, checkResult{Name: c.name, Status: statusFail, Detail: first})
			continue
		}
		passed = append(passed, c.name+" ("+detail+")")
		r.Checks = append(r.Checks, checkResult{Name: c.name, Status: statusPass, Detail: detail})
	}
	switch {
	case len(failed) > 0:
		r.Status, r.Exit, r.dryRun = statusFail, ExitFailure, false
		r.Detail = strings.Join(failed, "; ")
	case verifyFailed:
		r.Detail = s.needs + " would fail, so this step would not run"
	default:
		r.Detail = "would " + s.would
		if len(passed) > 0 {
			r.Detail += ". Checked: " + strings.Join(passed, ", ")
		}
	}
	r.Would = s.would
	return r
}

// googleLoginCheck proves the Google login is present. It does not call Google.
func (e *Env) googleLoginCheck() check {
	return check{"google login", func() (string, error) {
		res, err := google.Resolve(e.Getenv)
		if err != nil {
			return "", err
		}
		if err := res.Ready(); err != nil {
			return "", err
		}
		return res.Source, nil
	}}
}

func (e *Env) bingKeyCheck() check {
	return check{"bing key", func() (string, error) {
		if _, err := e.bingAPI(); err != nil {
			return "", err
		}
		return "env:" + bingKeyEnv, nil
	}}
}

func (e *Env) cloudflareTokenCheck() check {
	return check{"cloudflare token", func() (string, error) {
		if _, err := e.cloudflareAPI(); err != nil {
			return "", err
		}
		return "env:CLOUDFLARE_API_TOKEN", nil
	}}
}

// keyFileCheck fetches the IndexNow key file with a real request. It needs the
// saved key, because the file must hold that key. Before the first real run no
// key exists, so the check names that and does not guess.
func (e *Env) keyFileCheck(domain, keyLocation string) check {
	return check{"indexnow key file", func() (string, error) {
		cfg, err := config.Dir(e.Getenv)
		if err != nil {
			return "", err
		}
		var saved struct {
			Key string `json:"key"`
		}
		if _, err := config.Load(cfg, "indexnow-"+domain, &saved); err != nil {
			return "", err
		}
		if saved.Key == "" {
			return "no key saved yet, so not checked", nil
		}
		if keyLocation == "" {
			keyLocation = "https://" + domain + "/" + saved.Key + ".txt"
		}
		api := &indexnow.API{HTTP: httpx.New(false)}
		if err := api.CheckKeyFile(context.Background(), keyLocation, saved.Key); err != nil {
			return "", fmt.Errorf("the key file is not reachable: %w", err)
		}
		return "reachable", nil
	}}
}
