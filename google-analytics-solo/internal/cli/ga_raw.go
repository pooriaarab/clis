// Copyright 2026 Pooria Arab and contributors. Licensed under Apache-2.0. See LICENSE.
// Hand-authored generic escape hatch. Not generated; safe across regen.
//
// GA4's Admin API is a resource-name REST API (~166 methods that collapse to a
// handful of {name}/{parent} URL templates), so friendly per-resource commands
// can only cover the common surface. `raw` closes the gap: it calls ANY GA4
// Data or Admin endpoint by method + path, giving literal 100% public-API
// reachability, present or future. Mutating calls are confirm-gated.

package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

const (
	gaAdminBase = "https://analyticsadmin.googleapis.com"
	gaDataBase  = "https://analyticsdata.googleapis.com"
)

// gaDataVerbs are the Data API request suffixes; everything else is Admin.
var gaDataVerbs = []string{":runReport", ":runPivotReport", ":batchRunReports", ":batchRunPivotReports", ":runRealtimeReport", ":checkCompatibility", "/metadata", "audienceExports"}

// resolveRawURL builds an absolute URL from a path, choosing the host from an
// explicit override or by inferring Data vs Admin from the path.
func resolveRawURL(path, host string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	var base string
	switch strings.ToLower(host) {
	case "admin":
		base = gaAdminBase
	case "data":
		base = gaDataBase
	default:
		base = gaAdminBase
		for _, v := range gaDataVerbs {
			if strings.Contains(path, v) {
				base = gaDataBase
				break
			}
		}
	}
	return base + path
}

func newRawCmd(flags *rootFlags) *cobra.Command {
	var host, body string
	var query []string

	cmd := &cobra.Command{
		Use:   "raw <METHOD> <path>",
		Short: "Call any GA4 Data or Admin API endpoint directly (100% coverage escape hatch)",
		Long: "Invoke any GA4 endpoint by HTTP method and path — the escape hatch for the full public API " +
			"surface the friendly commands don't wrap. The host (Admin vs Data) is inferred from the path or " +
			"forced with --host. Mutating methods (POST/PATCH/PUT/DELETE, except report reads) require --confirm.",
		Example: "  # read: list custom dimensions\n" +
			"  google-analytics-solo-pp-cli raw GET /v1alpha/properties/123456789/customDimensions\n" +
			"  # write: create a custom dimension (needs --confirm + edit access)\n" +
			"  google-analytics-solo-pp-cli raw POST /v1alpha/properties/123456789/customDimensions \\\n" +
			"    --body '{\"parameterName\":\"plan\",\"displayName\":\"Plan\",\"scope\":\"EVENT\"}' --confirm\n" +
			"  # update with a field mask\n" +
			"  google-analytics-solo-pp-cli raw PATCH /v1alpha/properties/123456789 \\\n" +
			"    --query updateMask=displayName --body '{\"displayName\":\"New name\"}' --confirm",
		Annotations: map[string]string{"mcp:read-only": "false"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if len(args) < 2 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("usage: raw <METHOD> <path> [--host admin|data] [--body JSON] [--query k=v]"))
			}
			method := strings.ToUpper(args[0])
			url := resolveRawURL(args[1], host)

			if dryRunOK(flags) {
				fmt.Fprintf(cmd.OutOrStdout(), "would %s %s\n", method, url)
				return nil
			}
			// Gate mutations (reuses the shared classifier: report-style POSTs
			// are reads, everything else that writes needs --confirm).
			if isMutationEndpoint(method, url) {
				proceed, err := requireConfirm(cmd, flags, fmt.Sprintf("%s %s", method, url))
				if err != nil {
					return err
				}
				if !proceed {
					return nil
				}
			}

			params := map[string]string{}
			for _, q := range query {
				k, v, ok := strings.Cut(q, "=")
				if !ok {
					return usageErr(fmt.Errorf("--query must be k=v, got %q", q))
				}
				params[k] = v
			}
			var bodyVal any
			if strings.TrimSpace(body) != "" {
				if err := json.Unmarshal([]byte(body), &bodyVal); err != nil {
					return usageErr(fmt.Errorf("--body must be valid JSON: %w", err))
				}
			}

			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			var raw json.RawMessage
			switch method {
			case "GET":
				raw, err = c.Get(ctx, url, params)
			case "POST":
				raw, _, err = c.PostWithParams(ctx, url, params, bodyVal)
			case "PATCH":
				raw, _, err = c.PatchWithParams(ctx, url, params, bodyVal)
			case "PUT":
				raw, _, err = c.PutWithParams(ctx, url, params, bodyVal)
			case "DELETE":
				raw, _, err = c.DeleteWithParams(ctx, url, params)
			default:
				return usageErr(fmt.Errorf("unsupported method %q (use GET, POST, PATCH, PUT, or DELETE)", method))
			}
			if err != nil {
				return fmt.Errorf("%s %s: %w", method, url, err)
			}
			if len(raw) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "%s %s: ok (empty response)\n", method, url)
				return nil
			}
			var v any
			if json.Unmarshal(raw, &v) == nil {
				return printJSONFiltered(cmd.OutOrStdout(), v, flags)
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			return nil
		},
	}
	cmd.Flags().StringVar(&host, "host", "", "Force API host: admin or data (default: inferred from path)")
	cmd.Flags().StringVar(&body, "body", "", "Request body as JSON (POST/PATCH/PUT)")
	cmd.Flags().StringArrayVar(&query, "query", nil, "Query params as k=v (repeatable), e.g. --query updateMask=displayName")
	return cmd
}
