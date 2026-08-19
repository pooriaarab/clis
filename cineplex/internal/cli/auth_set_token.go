// hand-authored novel command; preserve across regenerate

package cli

import (
	"fmt"
	"strings"

	"cineplex-pp-cli/internal/config"
	"github.com/spf13/cobra"
)

func newAuthSetSceneTokenCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:         "set-token <token>",
		Short:       "Save a SCENE+ session token to the config file",
		Annotations: map[string]string{"mcp:local-write": "true", "pp:sensitive": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !flags.dryRun {
				return cmd.Help()
			}
			if len(args) < 1 || strings.TrimSpace(args[0]) == "" {
				return usageErr(fmt.Errorf("token is required\nUsage: %s <token>", cmd.CommandPath()))
			}
			if dryRunOK(flags) {
				return nil
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			_ = ctx
			cfg, err := config.Load(flags.configPath)
			if err != nil {
				return configErr(err)
			}
			if err := config.SaveSceneSessionToken(cfg.Path, args[0]); err != nil {
				return configErr(fmt.Errorf("saving session token: %w", err))
			}
			if flags.asJSON {
				return printJSONFiltered(cmd.OutOrStdout(), map[string]any{"saved": true, "config_path": cfg.Path}, flags)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "SCENE+ session token saved to %s\n", cfg.Path)
			return nil
		},
	}
}
