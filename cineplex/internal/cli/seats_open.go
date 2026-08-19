// hand-authored novel command; preserve across regenerate

package cli

import (
	"fmt"
	"os/exec"
	"runtime"

	"cineplex-pp-cli/internal/cliutil"
	"github.com/spf13/cobra"
)

func newSeatsOpenCmd(flags *rootFlags) *cobra.Command {
	var launch bool

	cmd := &cobra.Command{
		Use:         "open <theatreId> <showtimeId>",
		Short:       "Print or open the seat map for a showtime",
		Annotations: map[string]string{"mcp:read-only": "true", "pp:endpoint": "seats.open"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !hasChangedLocalFlags(cmd) && !flags.dryRun {
				return cmd.Help()
			}
			if len(args) < 2 {
				return usageErr(fmt.Errorf("missing required arguments\nUsage: %s <theatreId> <showtimeId>", cmd.CommandPath()))
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			url := seatMapURLFor(args[0], args[1])
			if flags.asJSON || flags.agent {
				if err := printJSONFiltered(cmd.OutOrStdout(), map[string]string{"seat_map_url": url}, flags); err != nil {
					return err
				}
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), url)
			}
			if !launch || cliutil.IsVerifyEnv() || dryRunOK(flags) {
				return nil
			}
			name, commandArgs := seatMapOpener(url)
			if err := exec.CommandContext(ctx, name, commandArgs...).Run(); err != nil {
				return fmt.Errorf("opening seat map: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&launch, "launch", false, "Open the seat map in your default browser")
	return cmd
}

func seatMapOpener(url string) (string, []string) {
	switch runtime.GOOS {
	case "darwin":
		return "open", []string{url}
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		return "xdg-open", []string{url}
	}
}
