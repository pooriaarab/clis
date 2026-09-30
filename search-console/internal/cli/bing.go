package cli

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	"github.com/pooriaarab/clis/search-console/internal/bing"
)

const bingKeyEnv = "BING_WEBMASTER_API_KEY"

func bingCmd(env *Env) *cobra.Command {
	cmd := &cobra.Command{Use: "bing", Short: "Verify a domain and submit URLs in Bing Webmaster Tools"}
	cmd.AddCommand(bingVerifyCmd(env), bingQuotaCmd(env))
	return cmd
}

// bingAPI reads the API key from the environment. The key is never saved.
func (e *Env) bingAPI() (*bing.API, error) {
	key := e.Getenv(bingKeyEnv)
	if key == "" {
		return nil, errors.New(bingKeyEnv + " is not set: create a key in Bing Webmaster Tools under Settings, API access")
	}
	return &bing.API{HTTP: e.Client, Base: envOr(e.Getenv, "BING_API_BASE", bing.DefaultBase), Key: key}, nil
}

func authBingCmd(env *Env) *cobra.Command {
	return &cobra.Command{
		Use:   "bing",
		Short: "Check the Bing Webmaster API key from " + bingKeyEnv,
		Long: `Bing has no login flow. Create an API key in Bing Webmaster Tools under
Settings, API access, and set it in ` + bingKeyEnv + `. This command calls Bing once
to prove the key works. It never saves the key.`,
		Args: cobra.NoArgs,
		RunE: run(func([]string) error {
			api, err := env.bingAPI()
			if err != nil {
				return err
			}
			sites, err := api.Sites(context.Background())
			if err != nil {
				return err
			}
			return env.Emit(map[string]any{"ok": true, "sites": len(sites)}, func() {
				env.P.Linef("Bing key works. The account has %d site(s).", len(sites))
			})
		}),
	}
}

func bingQuotaCmd(env *Env) *cobra.Command {
	return &cobra.Command{
		Use:   "quota <domain>",
		Short: "Show how many URLs the domain can still submit to Bing",
		Args:  cobra.ExactArgs(1),
		RunE: run(func(args []string) error {
			domain, err := normalizeDomain(args[0])
			if err != nil {
				return err
			}
			api, err := env.bingAPI()
			if err != nil {
				return err
			}
			q, err := api.Quota(context.Background(), domain)
			if err != nil {
				return err
			}
			return env.Emit(map[string]any{"domain": domain, "daily": q.Daily, "monthly": q.Monthly}, func() {
				env.P.Linef("%s can submit %d URLs today and %d this month.", domain, q.Daily, q.Monthly)
			})
		}),
	}
}
