// hand-authored novel command; preserve across regenerate

package cli

import (
	"fmt"
	"strings"

	"cineplex-pp-cli/internal/config"
	"github.com/spf13/cobra"
)

func newAuthSetSceneCookieCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "set-cookie <cookie>",
		Short: "Save your SCENE+ session cookie for authed ticketing calls",
		Long: strings.Trim(`
Authenticated ticketing endpoints (seat availability, cart, reserve-seats)
authenticate with your SCENE+ web session COOKIE plus the subscription key —
not a bearer token. Log in to cineplex.com in a browser, copy the Cookie
request header value from any apis.cineplex.com request in DevTools, and pass
it here. Stored privately in the config file; never logged.
`, "\n"),
		Annotations: map[string]string{"mcp:local-write": "true", "pp:sensitive": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !flags.dryRun {
				return cmd.Help()
			}
			if len(args) < 1 || strings.TrimSpace(args[0]) == "" {
				return usageErr(fmt.Errorf("cookie is required\nUsage: %s <cookie>", cmd.CommandPath()))
			}
			if dryRunOK(flags) {
				return nil
			}
			cfg, err := config.Load(flags.configPath)
			if err != nil {
				return configErr(err)
			}
			if err := config.SaveSceneSessionCookie(cfg.Path, args[0]); err != nil {
				return configErr(fmt.Errorf("saving session cookie: %w", err))
			}
			if flags.asJSON {
				return printJSONFiltered(cmd.OutOrStdout(), map[string]any{"saved": true, "config_path": cfg.Path}, flags)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "SCENE+ session cookie saved to %s\n", cfg.Path)
			return nil
		},
	}
}
