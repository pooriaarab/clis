// Hand-authored (not generated). Publishes a draft website live via the Solo
// publish API. See .printing-press-patches/sites-publish.md.

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"soloist-pp-cli/internal/client"
	"soloist-pp-cli/internal/config"
)

const (
	soloistPublishBaseURL = "https://soloist.ai"
	soloistPublishPath    = "/api/websites/publish"
)

// newSitesPublishCmd builds `sites publish <siteId>`.
//
// Publish is the one write that is not a plain Firestore mutation: the app
// POSTs the whole (plain-JSON) websiteSettings to a server route that copies
// the draft into the live Websites collection and provisions the live site.
// We GET the Firestore-typed draft, decode websiteSettings to plain JSON with
// the shared codec, and POST {handle, websiteSettings, draftId} to
// soloist.ai/api/websites/publish using the same bearer token the CLI already
// holds (the route authenticates with the Firebase ID token, no cookie).
func newSitesPublishCmd(flags *rootFlags) *cobra.Command {
	var force, waitLive bool
	var waitTimeout int
	var expect string
	cmd := &cobra.Command{
		Use:     "publish <siteId>",
		Short:   "Publish a draft website live (soloist.ai/<handle>).",
		Long:    "GET the draft, then POST it to the Solo publish API so it goes live at soloist.ai/<handle>.",
		Example: "  soloist-pp-cli sites publish 87c951ef-dd78-4074-9c06-b330af60845a",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if len(args) < 1 {
				return missingSitesArgument(cmd, flags, "<siteId>")
			}
			siteID := args[0]

			if dryRunOK(flags) {
				return writeSitesMutationResult(cmd, flags, map[string]any{
					"action":   "publish",
					"dry_run":  true,
					"draftId":  siteID,
					"endpoint": soloistWebBase() + soloistPublishPath,
					"note":     "dry-run does not fetch or publish; live mode GETs the draft, decodes websiteSettings to plain JSON, and POSTs {handle, websiteSettings, draftId}.",
				}, fmt.Sprintf("would publish draft %s", siteID))
			}

			// 1. Fetch the Firestore-typed draft document.
			fc, err := flags.newClient()
			if err != nil {
				return err
			}
			data, err := fc.Get(cmd.Context(), draftWebsitePath(siteID), map[string]string{})
			if err != nil {
				return classifyAPIError(err, flags)
			}
			var doc struct {
				Fields map[string]map[string]any `json:"fields"`
			}
			if err := json.Unmarshal(data, &doc); err != nil {
				return fmt.Errorf("parsing draft document: %w", err)
			}
			if doc.Fields == nil {
				return fmt.Errorf("draft %s has no fields (not found, or not yours)", siteID)
			}
			handle := ""
			if hv, ok := doc.Fields["handle"]; ok {
				handle, _ = fsDecode(hv).(string)
			}
			wsVal, ok := doc.Fields["websiteSettings"]
			if !ok {
				return fmt.Errorf("draft %s has no websiteSettings", siteID)
			}
			websiteSettings := fsDecode(wsVal) // Firestore-typed -> plain JSON

			// Readiness guard: refuse to publish a site whose websiteSettings is
			// missing the structure Solo's renderer needs (a Header + Footer
			// section, at least one page, and a theme). Publishing an incomplete
			// site produces a live 500. --force overrides.
			if issues := publishReadinessIssues(websiteSettings); len(issues) > 0 && !force {
				return usageErr(fmt.Errorf("site not ready to publish (would likely 500):\n  - %s\nfix these or re-run with --force", strings.Join(issues, "\n  - ")))
			}

			// 2. POST to the publish route on soloist.ai (same bearer token).
			cfg, err := config.Load(flags.configPath)
			if err != nil {
				return configErr(err)
			}
			cfg.BaseURL = soloistWebBase()
			pc := client.New(cfg, flags.timeout, flags.rateLimit)
			body := map[string]any{
				"handle":          handle,
				"websiteSettings": websiteSettings,
				"draftId":         siteID,
			}
			resp, status, err := pc.Post(cmd.Context(), soloistPublishPath, body)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			if status < 200 || status >= 300 {
				return fmt.Errorf("publish failed: HTTP %d: %s", status, string(resp))
			}

			url := "https://soloist.ai/" + handle

			result := map[string]any{
				"action":  "publish",
				"draftId": siteID,
				"handle":  handle,
				"status":  status,
				"url":     url,
			}
			msg := fmt.Sprintf("published: %s", url)

			// Optionally wait until the change is actually live-served. Publishing
			// returns before the CDN/edge reflects the new content, so a fetch
			// immediately after publish can serve a stale page. --wait-live polls
			// the live URL (cache-busted) until it returns 2xx and, if --expect is
			// given, until the served HTML contains that marker.
			if waitLive {
				live, elapsed, detail := waitForLive(cmd.Context(), url, expect, waitTimeout)
				result["waited"] = true
				result["live"] = live
				result["wait_seconds"] = elapsed
				if expect != "" {
					result["expect"] = expect
					result["expect_seen"] = detail == "matched"
				}
				if live {
					msg += fmt.Sprintf(" (live after %ds%s)", elapsed, map[bool]string{true: ", marker seen", false: ""}[detail == "matched"])
				} else {
					msg += fmt.Sprintf(" (WARNING: not confirmed live after %ds: %s)", elapsed, detail)
				}
			}

			return writeSitesMutationResult(cmd, flags, result, msg)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "publish even if the readiness check fails")
	cmd.Flags().BoolVar(&waitLive, "wait-live", false, "after publishing, poll the live URL until it serves the new content (beats CDN lag)")
	cmd.Flags().IntVar(&waitTimeout, "wait-timeout", 120, "max seconds to wait for --wait-live")
	cmd.Flags().StringVar(&expect, "expect", "", "with --wait-live, keep polling until the served HTML contains this substring")
	return cmd
}

// waitForLive polls url (cache-busted) until it returns 2xx and, when expect is
// non-empty, until the body contains expect. Returns whether it went live, the
// seconds elapsed, and a detail string ("matched", "ok", or the last error/
// status on timeout).
func waitForLive(ctx context.Context, url, expect string, timeoutSec int) (bool, int, string) {
	if timeoutSec <= 0 {
		timeoutSec = 120
	}
	deadline := timeoutSec
	elapsed := 0
	last := "no response"
	attempt := 0
	for elapsed <= deadline {
		attempt++
		// Base liveness polls the normal URL (what a visitor sees). Only when a
		// content marker is required do we cache-bust to force an origin-fresh
		// render — that hits a slower, rebuild-sensitive path, so it's reserved
		// for the --expect case.
		target := url
		if expect != "" {
			target = url + "?_pp=" + strconv.Itoa(attempt) + "-" + strconv.FormatInt(time.Now().UnixNano(), 10)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err == nil {
			// Browser-like headers: the app edge returns 406 to requests without
			// a User-Agent / Accept (bot protection), which would make every poll
			// look like a failure.
			req.Header.Set("User-Agent", "Mozilla/5.0")
			req.Header.Set("Accept", "*/*")
			req.Header.Set("Accept-Language", "en-US,en;q=0.9")
			resp, err := http.DefaultClient.Do(req)
			if err == nil {
				body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
				resp.Body.Close()
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					if expect == "" {
						return true, elapsed, "ok"
					}
					if strings.Contains(string(body), expect) {
						return true, elapsed, "matched"
					}
					last = "200 but marker not present yet"
				} else {
					last = "HTTP " + strconv.Itoa(resp.StatusCode)
				}
			} else {
				last = err.Error()
			}
		}
		time.Sleep(4 * time.Second)
		elapsed += 4
	}
	return false, elapsed, last
}

// publishReadinessIssues returns human-readable reasons the site is not safe to
// publish. Empty means ready. Solo's renderer 500s without a Header + Footer,
// at least one page, and a theme (colorScheme).
func publishReadinessIssues(ws any) []string {
	var issues []string
	m, ok := ws.(map[string]any)
	if !ok {
		return []string{"websiteSettings is missing or malformed"}
	}
	sections, _ := m["sections"].([]any)
	if len(sections) == 0 {
		issues = append(issues, "no sections")
	}
	types := map[string]bool{}
	for _, s := range sections {
		if sm, ok := s.(map[string]any); ok {
			if t, ok := sm["type"].(string); ok {
				types[t] = true
			}
		}
	}
	if !types["Header"] {
		issues = append(issues, "no Header section")
	}
	if !types["Footer"] {
		issues = append(issues, "no Footer section")
	}
	if pages, _ := m["pages"].([]any); len(pages) == 0 {
		issues = append(issues, "no pages")
	}
	if cs, ok := m["colorScheme"].(map[string]any); !ok || len(cs) == 0 {
		issues = append(issues, "no theme (colorScheme)")
	}
	return issues
}
