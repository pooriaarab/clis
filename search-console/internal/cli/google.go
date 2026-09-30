package cli

import (
	"context"
	"errors"
	"time"

	"github.com/spf13/cobra"

	"github.com/pooriaarab/clis/search-console/internal/google"
)

func googleCmd(env *Env) *cobra.Command {
	cmd := &cobra.Command{Use: "google", Short: "Verify a domain and submit sitemaps in Google Search Console"}
	cmd.AddCommand(googleVerifyCmd(env))
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
	cmd := &cobra.Command{
		Use:   "verify <domain>",
		Short: "Get the DNS TXT record for a domain and verify it with Google",
		Long: `Ask Google for the TXT record that proves you own the domain, print it, and
check it with exponential backoff until Google sees it. Add the record at your
DNS host while the command waits. Exit code 4 means DNS was not ready in time.`,
		Args: cobra.ExactArgs(1),
		RunE: run(func(args []string) error {
			domain, err := normalizeDomain(args[0])
			if err != nil {
				return err
			}
			ctx := context.Background()
			api, err := env.googleAPI(ctx)
			if err != nil {
				return err
			}
			record, err := api.VerificationToken(ctx, domain)
			if err != nil {
				return err
			}
			env.P.Warnf("Add this TXT record at the root of %s:\n  %s", domain, record)
			attempts, err := api.VerifyWithBackoff(ctx, domain, google.BackoffOpts{
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
			return env.Emit(map[string]any{"ok": true, "domain": domain, "txt_record": record, "attempts": attempts}, func() {
				env.P.Linef("Verified %s after %d attempt(s).", domain, attempts)
			})
		}),
	}
	cmd.Flags().DurationVar(&wait, "wait", 10*time.Minute, "how long to wait for DNS before giving up")
	cmd.Flags().DurationVar(&interval, "interval", 5*time.Second, "first delay between checks; it doubles up to 60s")
	return cmd
}
