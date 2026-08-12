// Copyright 2026 Pooria Arab and contributors. Licensed under Apache-2.0. See LICENSE.
// Hand-authored novel command. generate --force preserves implemented bodies.

package cli

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"google-analytics-solo-pp-cli/internal/client"
	"google-analytics-solo-pp-cli/internal/store"
)

// runGAReport POSTs a single runReport and returns the raw response.
func runGAReport(ctx context.Context, c *client.Client, property, dims, metrics, start, end string, limit int) (json.RawMessage, error) {
	body := map[string]any{
		"dateRanges": []map[string]string{{"startDate": start, "endDate": end}},
		"metrics":    nameObjs(metrics),
	}
	if dims != "" {
		body["dimensions"] = nameObjs(dims)
	}
	if limit > 0 {
		body["limit"] = strconv.Itoa(limit)
	}
	path := "/v1beta/" + property + ":runReport"
	raw, _, err := c.Post(ctx, path, body)
	return raw, err
}

func specHash(property, dims, metrics, since, until string) string {
	sum := sha256.Sum256([]byte(property + "|" + dims + "|" + metrics + "|" + since + "|" + until))
	return fmt.Sprintf("%x", sum[:8])
}

// cacheReport best-effort records a run for offline `trend`. Cache failures are
// non-fatal — they must never fail a successful report.
func cacheReport(ctx context.Context, property, dims, metrics, since, until string, raw json.RawMessage) {
	dbPath := defaultDBPath("google-analytics-solo-pp-cli")
	st, err := store.Open(dbPath)
	if err != nil {
		return
	}
	defer st.Close()
	_ = st.InsertGAReport(ctx, store.GAReportRow{
		Property: property,
		SpecHash: specHash(property, dims, metrics, since, until),
		Dims:     dims,
		Metrics:  metrics,
		Since:    since,
		Until:    until,
		RowsJSON: string(raw),
		RanAt:    time.Now().UTC().Format(time.RFC3339),
	})
}

type reportFailure struct {
	Property string `json:"property"`
	Error    string `json:"error"`
}

func newNovelReportCmd(flags *rootFlags) *cobra.Command {
	var (
		flagProperty      string
		flagAllProperties bool
		flagDims          string
		flagMetrics       string
		flagSince         string
		flagUntil         string
		flagLimit         int
		flagSave          string
		flagRun           string
	)

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Run a GA4 report by flags, across one or every registered property.",
		Long: "Build a GA4 runReport from flags (--dims, --metrics, --since) instead of hand-written JSON. " +
			"Use --all-properties to fan out across every registered alias, --save/--run for reusable report specs. " +
			"Each successful run is cached locally for offline 'trend'.",
		Example: "  google-analytics-solo-pp-cli report --property solo-prod --dims date --metrics activeUsers --since 7d --json\n" +
			"  google-analytics-solo-pp-cli report --all-properties --metrics activeUsers --since 30d --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would run GA4 report")
				return nil
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			// --run: load a saved spec, letting explicit flags override it.
			if flagRun != "" {
				st, err := loadAliases()
				if err != nil {
					return err
				}
				sr, ok := st.Reports[flagRun]
				if !ok {
					return usageErr(fmt.Errorf("no saved report %q; save one with 'report --save %s ...'", flagRun, flagRun))
				}
				if flagDims == "" {
					flagDims = sr.Dims
				}
				if flagMetrics == "" {
					flagMetrics = sr.Metrics
				}
				if flagSince == "" {
					flagSince = sr.Since
				}
				if flagUntil == "" {
					flagUntil = sr.Until
				}
				if flagLimit == 0 {
					flagLimit = sr.Limit
				}
				if flagProperty == "" && !flagAllProperties {
					flagProperty = sr.Property
				}
			}

			if flagMetrics == "" {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--metrics is required (e.g. --metrics activeUsers)"))
			}
			start, err := parseSince(flagSince)
			if err != nil {
				return err
			}
			end, err := parseUntil(flagUntil)
			if err != nil {
				return err
			}

			// --save: persist the spec before running.
			if flagSave != "" {
				st, err := loadAliases()
				if err != nil {
					return err
				}
				st.Reports[flagSave] = savedReport{
					Property: flagProperty, Dims: flagDims, Metrics: flagMetrics,
					Since: flagSince, Until: flagUntil, Limit: flagLimit,
				}
				if err := saveAliases(st); err != nil {
					return err
				}
				fmt.Fprintf(cmd.ErrOrStderr(), "saved report %q\n", flagSave)
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			// Build the target property set.
			type target struct{ alias, property string }
			var targets []target
			if flagAllProperties {
				st, err := loadAliases()
				if err != nil {
					return err
				}
				if len(st.Aliases) == 0 {
					return usageErr(fmt.Errorf("no registered properties; run 'alias add' or 'alias discover' first"))
				}
				names := make([]string, 0, len(st.Aliases))
				for n := range st.Aliases {
					names = append(names, n)
				}
				sort.Strings(names)
				for _, n := range names {
					targets = append(targets, target{alias: n, property: st.Aliases[n]})
				}
			} else {
				prop, err := ResolveProperty(flagProperty)
				if err != nil {
					return err
				}
				targets = append(targets, target{alias: prop, property: prop})
			}

			// Single property: emit the raw response directly.
			if len(targets) == 1 {
				raw, err := runGAReport(ctx, c, targets[0].property, flagDims, flagMetrics, start, end, flagLimit)
				if err != nil {
					return fmt.Errorf("running report for %s: %w", targets[0].property, err)
				}
				cacheReport(ctx, targets[0].property, flagDims, flagMetrics, flagSince, flagUntil, raw)
				var v any
				_ = json.Unmarshal(raw, &v)
				return printJSONFiltered(cmd.OutOrStdout(), v, flags)
			}

			// Fan-out: run concurrently, keep per-property errors separate.
			type res struct {
				alias string
				raw   json.RawMessage
				err   error
			}
			results := make(chan res, len(targets))
			sem := make(chan struct{}, 5)
			var wg sync.WaitGroup
			for _, t := range targets {
				wg.Add(1)
				go func(t target) {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()
					raw, err := runGAReport(ctx, c, t.property, flagDims, flagMetrics, start, end, flagLimit)
					if err == nil {
						cacheReport(ctx, t.property, flagDims, flagMetrics, flagSince, flagUntil, raw)
					}
					results <- res{alias: t.alias, raw: raw, err: err}
				}(t)
			}
			go func() { wg.Wait(); close(results) }()

			ok := map[string]any{}
			failures := make([]reportFailure, 0)
			for r := range results {
				if r.err != nil {
					failures = append(failures, reportFailure{Property: r.alias, Error: r.err.Error()})
					continue
				}
				var v any
				_ = json.Unmarshal(r.raw, &v)
				ok[r.alias] = v
			}
			if len(failures) > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %d of %d property reports failed\n", len(failures), len(targets))
			}
			out := map[string]any{"results": ok}
			if len(failures) > 0 {
				out["fetch_failures"] = failures
			}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	cmd.Flags().StringVar(&flagProperty, "property", "", "Property alias or numeric id (e.g. solo-prod or 123456789)")
	cmd.Flags().BoolVar(&flagAllProperties, "all-properties", false, "Run against every registered property alias and return per-property results")
	cmd.Flags().StringVar(&flagDims, "dims", "", "Comma-separated GA4 dimensions (e.g. date,country)")
	cmd.Flags().StringVar(&flagMetrics, "metrics", "", "Comma-separated GA4 metrics (e.g. activeUsers,sessions) [required]")
	cmd.Flags().StringVar(&flagSince, "since", "7d", "Start of range: Nd, Nw, today, yesterday, or YYYY-MM-DD")
	cmd.Flags().StringVar(&flagUntil, "until", "today", "End of range: today, yesterday, Nd, or YYYY-MM-DD")
	cmd.Flags().IntVar(&flagLimit, "limit", 0, "Max rows to return (0 = API default)")
	cmd.Flags().StringVar(&flagSave, "save", "", "Save this report spec under a name for later --run")
	cmd.Flags().StringVar(&flagRun, "run", "", "Run a previously --save'd report spec by name")
	return cmd
}
