// Copyright 2026 Pooria Arab and contributors. Licensed under Apache-2.0. See LICENSE.
// Hand-authored novel command. generate --force preserves implemented bodies.
//
// pp:data-source local — reads cached report runs from the local store only.

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"google-analytics-solo-pp-cli/internal/store"
)

func newNovelTrendCmd(flags *rootFlags) *cobra.Command {
	var flagProperty string
	var flagSince string

	cmd := &cobra.Command{
		Use:   "trend <metric>",
		Short: "Show how a metric moved over time using locally cached report runs, no re-query.",
		Long: "Read cached report runs from the local SQLite mirror and show a metric over time. " +
			"Populate the cache by running 'report' first. Offline and free — no API calls.",
		Example:     "  google-analytics-solo-pp-cli trend activeUsers --property solo-prod --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would read cached report runs")
				return nil
			}
			if len(args) < 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("a metric is required, e.g. 'trend activeUsers --property solo-prod'"))
			}
			metric := args[0]
			prop, err := ResolveProperty(flagProperty)
			if err != nil {
				return err
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			dbPath := defaultDBPath("google-analytics-solo-pp-cli")
			if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
				fmt.Fprintf(cmd.ErrOrStderr(), "no local mirror at %s\nrun: google-analytics-solo-pp-cli report --property %s --metrics %s --since 30d\n", dbPath, flagProperty, metric)
				if flags.asJSON || flags.agent {
					fmt.Fprintln(cmd.OutOrStdout(), "[]")
				}
				return nil
			}
			st, err := store.Open(dbPath)
			if err != nil {
				return fmt.Errorf("opening cache: %w", err)
			}
			defer st.Close()
			runs, err := st.QueryGAReports(ctx, prop, metric)
			if err != nil {
				return err
			}

			type point struct {
				RanAt string  `json:"ran_at"`
				Since string  `json:"since"`
				Until string  `json:"until"`
				Value float64 `json:"value"`
			}
			var cutoff time.Time
			if flagSince != "" {
				d, derr := sinceToDuration(flagSince)
				if derr != nil {
					return derr
				}
				cutoff = time.Now().Add(-d)
			}
			points := make([]point, 0, len(runs))
			for _, r := range runs {
				if !cutoff.IsZero() {
					if t, perr := time.Parse(time.RFC3339, r.RanAt); perr == nil && t.Before(cutoff) {
						continue
					}
				}
				points = append(points, point{
					RanAt: r.RanAt, Since: r.Since, Until: r.Until,
					Value: firstMetricValue(json.RawMessage(r.RowsJSON)),
				})
			}
			if flags.asJSON || flags.agent {
				return printJSONFiltered(cmd.OutOrStdout(), map[string]any{"property": prop, "metric": metric, "points": points}, flags)
			}
			if len(points) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "no cached runs for %s on %s; run 'report' first\n", metric, prop)
				return nil
			}
			for _, p := range points {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s=%.0f\t(%s..%s)\n", p.RanAt, metric, p.Value, p.Since, p.Until)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&flagProperty, "property", "", "Property alias or numeric id")
	cmd.Flags().StringVar(&flagSince, "since", "", "Only include cached runs newer than this window (e.g. 90d, 4w)")
	return cmd
}
