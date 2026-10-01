package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/pooriaarab/clis/search-console/internal/backoff"
	"github.com/pooriaarab/clis/search-console/internal/bing"
	"github.com/pooriaarab/clis/search-console/internal/cloudflare"
)

func bingVerifyCmd(env *Env) *cobra.Command {
	var wait, interval time.Duration
	var zone string
	cmd := &cobra.Command{
		Use:   "verify <domain>",
		Short: "Add a domain to Bing Webmaster Tools and verify it with a DNS CNAME",
		Long: `Add the domain to Bing, read its verification code, and create a CNAME record
from <code>.<domain> to verify.bing.com. Bing cannot see a proxied record, so the
record is never proxied. Then check with exponential backoff until Bing finds it.
Exit code 4 means DNS was not ready in time.`,
		Args: cobra.ExactArgs(1),
		RunE: run(func(args []string) error {
			if err := checkPolling(wait, interval); err != nil {
				return err
			}
			domain, err := normalizeDomain(args[0])
			if err != nil {
				return err
			}
			ctx := context.Background()
			api, err := env.bingAPI()
			if err != nil {
				return err
			}
			var cf *cloudflare.API
			if zone != "" {
				if cf, err = env.cloudflareAPI(); err != nil {
					return err
				}
			}
			site, err := api.FindSite(ctx, domain)
			if err != nil {
				return err
			}
			result := map[string]any{"ok": true, "domain": domain, "site": bing.SiteURL(domain), "already_verified": site != nil && site.IsVerified}
			if site != nil && site.IsVerified {
				return env.Emit(result, func() { env.P.Linef("%s was already verified in Bing.", domain) })
			}
			if site == nil {
				present, err := api.AddSite(ctx, domain)
				if err != nil {
					return err
				}
				result["already_present"] = present
				if site, err = api.FindSite(ctx, domain); err != nil {
					return err
				}
			}
			if site == nil && env.DryRun {
				site = &bing.Site{DNSRecord: "CODE." + domain}
			}
			if site == nil || site.DNSRecord == "" {
				return fmt.Errorf("Bing gave no DNS verification code for %s", bing.SiteURL(domain))
			}
			name := site.DNSRecord
			result["cname"] = map[string]string{"name": name, "target": bing.VerifyTarget}
			if cf == nil {
				env.P.Warnf("Add this CNAME record, not proxied:\n  %s -> %s", name, bing.VerifyTarget)
			} else {
				proxied := false
				changed, err := publishRecord(ctx, cf, zone, domain, cloudflare.Record{Type: "CNAME", Name: name, Content: bing.VerifyTarget, TTL: 120, Proxied: &proxied})
				if err != nil {
					return err
				}
				result["cloudflare_changed"] = changed
				env.P.Warnf("Cloudflare CNAME %s -> %s, not proxied (changed: %t).", name, bing.VerifyTarget, changed)
			}
			attempts, err := api.VerifyWithBackoff(ctx, domain, backoff.Opts{
				Wait: wait, Interval: interval,
				OnRetry: func(n int, d time.Duration) {
					env.P.Warnf("Bing cannot see the record yet (attempt %d). Retrying in %s.", n, d)
				},
			})
			if errors.Is(err, bing.ErrVerifyTimeout) {
				return &ExitError{Code: ExitTimeout, Err: errors.New(err.Error() + ". Add the CNAME record " + name + " and run the command again")}
			}
			if err != nil {
				return err
			}
			result["attempts"] = attempts
			return env.Emit(result, func() { env.P.Linef("Verified %s in Bing after %d attempt(s).", domain, attempts) })
		}),
	}
	cmd.Flags().StringVar(&zone, "cloudflare-zone", "", "create the CNAME record on Cloudflare: a zone id, or auto to find it (needs CLOUDFLARE_API_TOKEN)")
	cmd.Flags().DurationVar(&wait, "wait", 10*time.Minute, "how long to wait for DNS before giving up")
	cmd.Flags().DurationVar(&interval, "interval", 5*time.Second, "first delay between checks; it doubles up to 60s")
	return cmd
}
