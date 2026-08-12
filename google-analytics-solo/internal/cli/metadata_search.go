// Copyright 2026 Pooria Arab and contributors. Licensed under Apache-2.0. See LICENSE.
// Hand-authored novel command. generate --force preserves implemented bodies.
//
// pp:data-source live — reads the GA4 metadata catalog from the live API.

package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newNovelMetadataSearchCmd(flags *rootFlags) *cobra.Command {
	var flagProperty string

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Full-text search the GA4 dimension/metric catalog to find exact API names.",
		Long: "Fetch a property's metadata (available dimensions and metrics) and filter it by a query " +
			"across API name, UI name, and description. Use to find the exact API name before building a report.",
		Example:     "  google-analytics-solo-pp-cli metadata search revenue --property solo-prod --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would fetch and filter GA4 metadata")
				return nil
			}
			if len(args) < 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("a search query is required, e.g. 'metadata search revenue --property solo-prod'"))
			}
			query := strings.ToLower(args[0])
			prop, err := ResolveProperty(flagProperty)
			if err != nil {
				return err
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			raw, err := c.Get(ctx, "/v1beta/"+prop+"/metadata", nil)
			if err != nil {
				return fmt.Errorf("fetching metadata: %w", err)
			}
			var md struct {
				Dimensions []struct {
					APIName     string `json:"apiName"`
					UIName      string `json:"uiName"`
					Description string `json:"description"`
				} `json:"dimensions"`
				Metrics []struct {
					APIName     string `json:"apiName"`
					UIName      string `json:"uiName"`
					Description string `json:"description"`
				} `json:"metrics"`
			}
			if err := json.Unmarshal(raw, &md); err != nil {
				return fmt.Errorf("parsing metadata: %w", err)
			}

			type match struct {
				Kind        string `json:"kind"`
				APIName     string `json:"apiName"`
				UIName      string `json:"uiName"`
				Description string `json:"description"`
			}
			hit := func(a, u, d string) bool {
				return strings.Contains(strings.ToLower(a), query) ||
					strings.Contains(strings.ToLower(u), query) ||
					strings.Contains(strings.ToLower(d), query)
			}
			matches := make([]match, 0)
			for _, d := range md.Dimensions {
				if hit(d.APIName, d.UIName, d.Description) {
					matches = append(matches, match{"dimension", d.APIName, d.UIName, d.Description})
				}
			}
			for _, m := range md.Metrics {
				if hit(m.APIName, m.UIName, m.Description) {
					matches = append(matches, match{"metric", m.APIName, m.UIName, m.Description})
				}
			}
			if flags.asJSON || flags.agent {
				return printJSONFiltered(cmd.OutOrStdout(), matches, flags)
			}
			if len(matches) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "no dimensions or metrics match %q\n", args[0])
				return nil
			}
			for _, m := range matches {
				fmt.Fprintf(cmd.OutOrStdout(), "%-9s %-28s %s\n", m.Kind, m.APIName, m.UIName)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&flagProperty, "property", "", "Property alias or numeric id")
	return cmd
}
