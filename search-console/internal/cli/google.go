package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/pooriaarab/clis/search-console/internal/backoff"
	"github.com/pooriaarab/clis/search-console/internal/cloudflare"
	"github.com/pooriaarab/clis/search-console/internal/google"
)

func googleCmd(env *Env) *cobra.Command {
	cmd := &cobra.Command{Use: "google", Short: "Verify a domain and submit sitemaps in Google Search Console"}
	cmd.AddCommand(googleVerifyCmd(env), googleSitemapCmd(env))
	return cmd
}

// googleAPI logs in from the saved or env credentials and returns an API client.
func (e *Env) googleAPI(ctx context.Context) (*google.API, error) {
	res, err := google.Resolve(e.Getenv)
	if err != nil {
		return nil, err
	}
	token, err := e.oauth().AccessToken(ctx, res)
	if err != nil {
		return nil, err
	}
	return &google.API{HTTP: e.Client, Base: envOr(e.Getenv, "GOOGLE_API_BASE", google.DefaultAPIBase), Token: token}, nil
}

func googleVerifyCmd(env *Env) *cobra.Command {
	var wait, interval time.Duration
	var zone string
	cmd := &cobra.Command{
		Use:   "verify <domain>",
		Short: "Verify a domain with Google over DNS and add it to Search Console",
		Long: `Ask Google for the TXT record that proves you own the domain, print it, and
check it with exponential backoff until Google sees it. Add the record at your
DNS host while the command waits. Then it adds the domain to Search Console.
Exit code 4 means DNS was not ready in time.`,
		Args: cobra.ExactArgs(1),
		RunE: run(func(args []string) error {
			domain, err := normalizeDomain(args[0])
			if err != nil {
				return err
			}
			ctx := context.Background()
			var cf *cloudflare.API
			if zone != "" {
				if cf, err = env.cloudflareAPI(); err != nil {
					return err
				}
			}
			api, err := env.googleAPI(ctx)
			if err != nil {
				return err
			}
			already, err := api.IsVerified(ctx, domain)
			if err != nil {
				return err
			}
			result := map[string]any{"ok": true, "domain": domain, "site": google.SiteURL(domain), "already_verified": already}
			if !already {
				record, err := api.VerificationToken(ctx, domain)
				if err != nil {
					return err
				}
				if cf == nil {
					env.P.Warnf("Add this TXT record at the root of %s:\n  %s", domain, record)
				} else {
					created, err := publishRecord(ctx, cf, zone, domain, cloudflare.Record{Type: "TXT", Name: domain, Content: record, TTL: 120})
					if err != nil {
						return err
					}
					result["cloudflare_created"] = created
					env.P.Warnf("Cloudflare TXT record for %s (created: %t).", domain, created)
				}
				attempts, err := api.VerifyWithBackoff(ctx, domain, backoff.Opts{
					Wait: wait, Interval: interval,
					OnRetry: func(n int, d time.Duration) {
						env.P.Warnf("Google cannot see the record yet (attempt %d). Retrying in %s.", n, d)
					},
				})
				if errors.Is(err, google.ErrVerifyTimeout) {
					return &ExitError{Code: ExitTimeout, Err: errors.New(err.Error() + ". Add the TXT record to " + domain + " and run the command again")}
				}
				if err != nil {
					return err
				}
				result["txt_record"], result["attempts"] = record, attempts
			}
			if err := api.AddSite(ctx, domain); err != nil {
				return fmt.Errorf("%s is verified but Search Console refused the site: %w", domain, err)
			}
			perm, err := api.Permission(ctx, domain)
			if err != nil {
				return err
			}
			if perm != "siteOwner" && !env.DryRun {
				return fmt.Errorf("%s has permission %q in Search Console, want siteOwner", google.SiteURL(domain), perm)
			}
			result["permission"] = perm
			return env.Emit(result, func() {
				if already {
					env.P.Linef("%s was already verified.", domain)
				} else {
					env.P.Linef("Verified %s after %d attempt(s).", domain, result["attempts"])
				}
				env.P.Linef("Added %s to Search Console.", google.SiteURL(domain))
			})
		}),
	}
	cmd.Flags().StringVar(&zone, "cloudflare-zone", "", "create the TXT record on Cloudflare: a zone id, or auto to find it (needs CLOUDFLARE_API_TOKEN)")
	cmd.Flags().DurationVar(&wait, "wait", 10*time.Minute, "how long to wait for DNS before giving up")
	cmd.Flags().DurationVar(&interval, "interval", 5*time.Second, "first delay between checks; it doubles up to 60s")
	return cmd
}

// cloudflareAPI reads the token from CLOUDFLARE_API_TOKEN. It never reads a file.
func (e *Env) cloudflareAPI() (*cloudflare.API, error) {
	token := e.Getenv("CLOUDFLARE_API_TOKEN")
	if token == "" {
		return nil, errors.New("--cloudflare-zone needs the CLOUDFLARE_API_TOKEN environment variable")
	}
	return &cloudflare.API{HTTP: e.Client, Base: envOr(e.Getenv, "CLOUDFLARE_API_BASE", cloudflare.DefaultBase), Token: token}, nil
}

// publishRecord creates a DNS record on Cloudflare. zone is a zone id or "auto",
// and "auto" finds the zone that holds domain.
func publishRecord(ctx context.Context, cf *cloudflare.API, zone, domain string, rec cloudflare.Record) (bool, error) {
	if zone == "auto" {
		var err error
		if zone, err = cf.ZoneID(ctx, domain); err != nil {
			return false, err
		}
	}
	return cf.EnsureRecord(ctx, zone, rec)
}
