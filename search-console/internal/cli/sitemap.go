package cli

import (
	"context"
	"errors"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pooriaarab/clis/search-console/internal/sitemap"
)

func sitemapCmd(env *Env) *cobra.Command {
	cmd := &cobra.Command{Use: "sitemap", Short: "Check a sitemap before you submit it"}
	cmd.AddCommand(&cobra.Command{
		Use:   "check <sitemap-url>",
		Short: "Fetch a sitemap and check it",
		Long:  "Fetch a sitemap and check it for HTTP 200, valid XML and fewer than 50,000 URLs.\nExit code 3 means the sitemap has problems.",
		Args:  cobra.ExactArgs(1),
		RunE: run(func(args []string) error {
			rep, err := sitemap.Check(context.Background(), env.Client, args[0])
			if errors.Is(err, sitemap.ErrNotAbsolute) {
				return &ExitError{Code: ExitUsage, Err: err}
			}
			if err != nil {
				return err
			}
			if err := env.Emit(rep, func() {
				if rep.OK {
					env.P.Linef("ok: %s has %d URLs (%s)", rep.URL, rep.URLs, rep.Kind)
					return
				}
				env.P.Linef("problems in %s:\n  %s", rep.URL, strings.Join(rep.Problems, "\n  "))
			}); err != nil {
				return err
			}
			if !rep.OK {
				return &ExitError{Code: ExitSitemapError, Err: errors.New("the sitemap has problems"), Printed: true}
			}
			return nil
		}),
	})
	return cmd
}
