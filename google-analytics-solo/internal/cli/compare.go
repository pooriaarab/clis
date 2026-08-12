// Copyright 2026 Pooria Arab and contributors. Licensed under Apache-2.0. See LICENSE.
// Hand-authored novel command. generate --force preserves implemented bodies.

package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"sync"

	"github.com/spf13/cobra"
)

// gaReportResponse is the subset of runReport we parse for a single metric.
type gaReportResponse struct {
	Rows []struct {
		MetricValues []struct {
			Value string `json:"value"`
		} `json:"metricValues"`
	} `json:"rows"`
	Totals []struct {
		MetricValues []struct {
			Value string `json:"value"`
		} `json:"metricValues"`
	} `json:"totals"`
}

// firstMetricValue returns the metric value from totals, else the first row.
func firstMetricValue(raw json.RawMessage) float64 {
	var r gaReportResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return 0
	}
	if len(r.Totals) > 0 && len(r.Totals[0].MetricValues) > 0 {
		v, _ := strconv.ParseFloat(r.Totals[0].MetricValues[0].Value, 64)
		return v
	}
	if len(r.Rows) > 0 && len(r.Rows[0].MetricValues) > 0 {
		v, _ := strconv.ParseFloat(r.Rows[0].MetricValues[0].Value, 64)
		return v
	}
	return 0
}

func newNovelCompareCmd(flags *rootFlags) *cobra.Command {
	var (
		flagMetric string
		flagSince  string
		flagUntil  string
	)

	cmd := &cobra.Command{
		Use:         "compare",
		Short:       "Rank all registered properties by a single metric over a date range.",
		Long:        "Fan out one metric across every registered property alias and print a ranked leaderboard. Use for 'which property leads on metric X'; use 'report' for single-property detail.",
		Example:     "  google-analytics-solo-pp-cli compare --metric activeUsers --since 30d --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would rank registered properties by metric")
				return nil
			}
			if flagMetric == "" {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--metric is required (e.g. --metric activeUsers)"))
			}
			start, err := parseSince(flagSince)
			if err != nil {
				return err
			}
			end, err := parseUntil(flagUntil)
			if err != nil {
				return err
			}
			st, err := loadAliases()
			if err != nil {
				return err
			}
			if len(st.Aliases) == 0 {
				return usageErr(fmt.Errorf("no registered properties; run 'alias add' or 'alias discover' first"))
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			type row struct {
				Property string  `json:"property"`
				Metric   string  `json:"metric"`
				Value    float64 `json:"value"`
			}
			type res struct {
				row row
				err error
				key string
			}
			names := make([]string, 0, len(st.Aliases))
			for n := range st.Aliases {
				names = append(names, n)
			}
			results := make(chan res, len(names))
			sem := make(chan struct{}, 5)
			var wg sync.WaitGroup
			for _, n := range names {
				wg.Add(1)
				go func(alias, prop string) {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()
					raw, err := runGAReport(ctx, c, prop, "", flagMetric, start, end, 0)
					if err != nil {
						results <- res{key: alias, err: err}
						return
					}
					results <- res{key: alias, row: row{Property: alias, Metric: flagMetric, Value: firstMetricValue(raw)}}
				}(n, st.Aliases[n])
			}
			go func() { wg.Wait(); close(results) }()

			ranked := make([]row, 0, len(names))
			failures := make([]reportFailure, 0)
			for r := range results {
				if r.err != nil {
					failures = append(failures, reportFailure{Property: r.key, Error: r.err.Error()})
					continue
				}
				ranked = append(ranked, r.row)
			}
			sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].Value > ranked[j].Value })
			if len(failures) > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %d of %d property fetches failed; ranking the remaining %d\n", len(failures), len(names), len(ranked))
			}

			if flags.asJSON || flags.agent {
				out := map[string]any{"metric": flagMetric, "ranking": ranked}
				if len(failures) > 0 {
					out["fetch_failures"] = failures
				}
				return printJSONFiltered(cmd.OutOrStdout(), out, flags)
			}
			if len(ranked) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no results")
				return nil
			}
			for _, r := range ranked {
				fmt.Fprintf(cmd.OutOrStdout(), "%-24s %s=%.0f\n", r.Property, r.Metric, r.Value)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&flagMetric, "metric", "", "Single GA4 metric to rank by (e.g. activeUsers) [required]")
	cmd.Flags().StringVar(&flagSince, "since", "30d", "Start of range: Nd, Nw, today, yesterday, or YYYY-MM-DD")
	cmd.Flags().StringVar(&flagUntil, "until", "today", "End of range: today, yesterday, Nd, or YYYY-MM-DD")
	return cmd
}
