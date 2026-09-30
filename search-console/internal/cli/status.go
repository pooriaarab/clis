package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/pooriaarab/clis/search-console/internal/google"
)

func authStatusCmd(env *Env) *cobra.Command {
	var check bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show which credentials are set, without printing them",
		Args:  cobra.NoArgs,
		RunE: run(func([]string) error {
			res, err := google.Resolve(env.Getenv)
			if err != nil {
				return err
			}
			g := map[string]any{"configured": res.Source != "", "source": res.Source}
			if check {
				if _, err := env.oauth().AccessToken(context.Background(), res); err != nil {
					return err
				}
				g["check"] = "ok"
			}
			return env.Emit(map[string]any{"google": g}, func() {
				if res.Source == "" {
					env.P.Linef("google: not configured")
					return
				}
				line := "google: configured (" + res.Source + ")"
				if check {
					line += ", token check ok"
				}
				env.P.Linef("%s", line)
			})
		}),
	}
	cmd.Flags().BoolVar(&check, "check", false, "get an access token to prove the login works")
	return cmd
}
