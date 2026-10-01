package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/pooriaarab/clis/search-console/internal/google"
	"github.com/pooriaarab/clis/search-console/internal/sitemap"
)

func googleSitemapCmd(env *Env) *cobra.Command {
	cmd := &cobra.Command{Use: "sitemap", Short: "Submit a sitemap to Search Console and read its status"}
	var wait, interval time.Duration
	submit := &cobra.Command{
		Use:   "submit <domain> <sitemap-url>",
		Short: "Check a sitemap, submit it and wait for Google to read it",
		Long: `Fetch and check the sitemap first. A sitemap with problems is not submitted (exit 3).
Then submit it and poll until Google has read it or --wait ends. Exit code 3 also
means Google reported errors for the sitemap.`,
		Args: cobra.ExactArgs(2),
		RunE: run(func(args []string) error {
			if err := checkPolling(wait, interval); err != nil {
				return err
			}
			domain, err := normalizeDomain(args[0])
			if err != nil {
				return err
			}
			ctx := context.Background()
			if err := checkSitemap(ctx, env, args[1]); err != nil {
				return err
			}
			api, err := env.googleAPI(ctx)
			if err != nil {
				return err
			}
			perm, err := api.Permission(ctx, domain)
			if err != nil {
				return fmt.Errorf("read %s: %w (run `search-console google verify %s` first)", google.SiteURL(domain), err, domain)
			}
			if perm != "siteOwner" && !env.DryRun {
				return fmt.Errorf("%s has permission %q, want siteOwner", google.SiteURL(domain), perm)
			}
			if err := api.SubmitSitemap(ctx, domain, args[1]); err != nil {
				return err
			}
			st, err := api.WaitSitemap(ctx, domain, args[1], wait, interval)
			if err != nil {
				return err
			}
			return reportSitemap(env, st)
		}),
	}
	submit.Flags().DurationVar(&wait, "wait", time.Minute, "how long to wait for Google to read the sitemap")
	submit.Flags().DurationVar(&interval, "interval", 5*time.Second, "delay between status checks")
	status := &cobra.Command{
		Use:   "status <domain> <sitemap-url>",
		Short: "Show what Search Console knows about a submitted sitemap",
		Long:  "Show isPending, lastDownloaded and the error and warning counts. Exit code 3 means errors.",
		Args:  cobra.ExactArgs(2),
		RunE: run(func(args []string) error {
			domain, err := normalizeDomain(args[0])
			if err != nil {
				return err
			}
			api, err := env.googleAPI(context.Background())
			if err != nil {
				return err
			}
			st, err := api.Sitemap(context.Background(), domain, args[1])
			if err != nil {
				return err
			}
			return reportSitemap(env, st)
		}),
	}
	cmd.AddCommand(submit, status)
	return cmd
}

// checkSitemap fetches the sitemap. It fails with exit code 3 when the sitemap
// has problems, so nobody submits a broken sitemap.
func checkSitemap(ctx context.Context, env *Env, url string) error {
	rep, err := sitemap.Check(ctx, env.Client, url)
	if errors.Is(err, sitemap.ErrNotAbsolute) {
		return &ExitError{Code: ExitUsage, Err: err}
	}
	if err != nil {
		return err
	}
	if !rep.OK && !env.DryRun {
		env.P.Warnf("problems in %s:\n  %s", rep.URL, strings.Join(rep.Problems, "\n  "))
		return &ExitError{Code: ExitSitemapError, Err: errors.New("the sitemap has problems, so it was not submitted")}
	}
	return nil
}

// reportSitemap prints the status. Errors from Google exit with code 3.
func reportSitemap(env *Env, st *google.SitemapStatus) error {
	if err := env.Emit(st, func() {
		env.P.Linef("%s\n  pending: %t\n  last downloaded: %s\n  errors: %d\n  warnings: %d",
			st.Path, st.IsPending, orNever(st.LastDownloaded), st.Errors, st.Warnings)
	}); err != nil {
		return err
	}
	if st.Errors > 0 {
		return &ExitError{Code: ExitSitemapError, Err: errors.New("Google reported errors for the sitemap"), Printed: true}
	}
	return nil
}

func orNever(s string) string {
	if s == "" {
		return "never"
	}
	return s
}
