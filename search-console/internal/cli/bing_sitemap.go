package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pooriaarab/clis/search-console/internal/bing"
)

func bingSitemapCmd(env *Env) *cobra.Command {
	cmd := &cobra.Command{Use: "sitemap", Short: "Submit a sitemap to Bing and read its status"}
	submit := &cobra.Command{
		Use:   "submit <domain> <sitemap-url>",
		Short: "Check a sitemap and submit it to Bing",
		Long: `Fetch and check the sitemap first. A sitemap with problems is not submitted (exit 3).
The site must be verified in Bing. Bing reads the sitemap later, so the status can
be missing for a while. Run "bing sitemap status" to read it again.`,
		Args: cobra.ExactArgs(2),
		RunE: run(func(args []string) error {
			domain, err := normalizeDomain(args[0])
			if err != nil {
				return err
			}
			ctx := context.Background()
			if err := checkSitemap(ctx, env, args[1]); err != nil {
				return err
			}
			api, err := env.bingAPI()
			if err != nil {
				return err
			}
			site, err := api.FindSite(ctx, domain)
			if err != nil {
				return err
			}
			if !env.DryRun && (site == nil || !site.IsVerified) {
				return fmt.Errorf("%s is not verified in Bing (run `search-console bing verify %s` first)", bing.SiteURL(domain), domain)
			}
			if err := api.SubmitFeed(ctx, domain, args[1]); err != nil {
				return err
			}
			f, err := api.Feed(ctx, domain, args[1])
			if err != nil {
				return err
			}
			return reportFeed(env, args[1], f, true)
		}),
	}
	status := &cobra.Command{
		Use:   "status <domain> <sitemap-url>",
		Short: "Show what Bing knows about a submitted sitemap",
		Long:  "Show the status, the last crawl and the URL count. Exit code 3 means Bing gave the sitemap an error status.",
		Args:  cobra.ExactArgs(2),
		RunE: run(func(args []string) error {
			domain, err := normalizeDomain(args[0])
			if err != nil {
				return err
			}
			api, err := env.bingAPI()
			if err != nil {
				return err
			}
			f, err := api.Feed(context.Background(), domain, args[1])
			if err != nil {
				return err
			}
			if f == nil && !env.DryRun {
				return fmt.Errorf("Bing does not list %s for %s (submit it with `search-console bing sitemap submit`)", args[1], bing.SiteURL(domain))
			}
			return reportFeed(env, args[1], f, false)
		}),
	}
	cmd.AddCommand(submit, status)
	return cmd
}

// reportFeed prints the feed. A missing feed after a submit is normal, because
// Bing lists it later. An error status exits with code 3.
func reportFeed(env *Env, feedURL string, f *bing.Feed, submitted bool) error {
	out := map[string]any{"ok": true, "sitemap": feedURL, "submitted": submitted, "listed": f != nil}
	if f != nil {
		out["ok"] = !f.HasError()
		out["status"], out["type"] = f.Status, f.Type
		out["last_crawled"], out["url_count"] = f.LastCrawled, f.URLCount
	}
	if err := env.Emit(out, func() {
		if submitted {
			env.P.Linef("Submitted %s to Bing.", feedURL)
		}
		if f == nil {
			env.P.Linef("Bing does not list the sitemap yet. Bing reads it later.")
			return
		}
		env.P.Linef("%s\n  status: %s\n  last crawled: %s\n  urls: %d", f.URL, f.Status, orNever(f.LastCrawled), f.URLCount)
	}); err != nil {
		return err
	}
	if f != nil && f.HasError() {
		return &ExitError{Code: ExitSitemapError, Err: errors.New("Bing reported an error for the sitemap"), Printed: true}
	}
	return nil
}
