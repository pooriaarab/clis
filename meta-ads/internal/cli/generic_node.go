// Copyright 2026 Pooria Arab and contributors. Licensed under Apache-2.0.
//
// Hand-written, not printed. Meta's Graph API is node/edge shaped, not
// path-per-resource shaped: every object (campaign, ad set, ad, creative,
// audience, page, ...) is reachable at GET/POST/DELETE /{node-id}, and every
// relationship (an ad account's campaigns, a campaign's ad sets, a page's
// posts, ...) is reachable at GET/POST /{node-id}/{edge}. The spec this CLI
// was generated from models a handful of ad-management edges by name
// (act_{ad_account_id}/campaigns, /adsets, /ads, /adcreatives,
// /customaudiences, /insights) but the generic /{node_id} and
// /{node_id}/{edge} paths themselves don't carry a derivable resource name,
// so cli-printing-press's doc-to-command generator skips them (same failure
// mode the Google Ads CLI hit on ~59 of its ~65 mutable resources — see
// mutate_resource.go in that project). This file is the same fix applied to
// Meta's node/edge shape instead of Google's REST-plural :mutate shape: two
// generic commands that reach every Graph API object and edge this CLI
// doesn't have a dedicated command for yet (pages, posts, page insights,
// lead-gen forms, product catalogs, business assets, ...).
//
// See .printing-press-patches/ for the reprint-guard note per AGENTS.md.
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

func newGenericNodeCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Read, update, or delete any Graph API node by id (escape hatch beyond campaigns/adsets/ads/creatives/audiences)",
	}
	cmd.AddCommand(newGenericNodeGetCmd(flags))
	cmd.AddCommand(newGenericNodeUpdateCmd(flags))
	cmd.AddCommand(newGenericNodeDeleteCmd(flags))
	return cmd
}

func newGenericEdgeCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edge",
		Short: "List or create on any Graph API edge by name (escape hatch for edges this CLI has no dedicated command for)",
	}
	cmd.AddCommand(newGenericEdgeGetCmd(flags))
	cmd.AddCommand(newGenericEdgeCreateCmd(flags))
	return cmd
}

func newGenericNodeGetCmd(flags *rootFlags) *cobra.Command {
	var flagFields string

	cmd := &cobra.Command{
		Use:   "get <nodeId>",
		Short: "Get any Graph API node by id",
		Example: `  meta-ads-pp-cli node get 120210000000000001 --fields name,status,objective
  meta-ads-pp-cli node get me --fields id,name`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			params := map[string]string{}
			if flagFields != "" {
				params["fields"] = flagFields
			}
			data, err := c.Get(cmd.Context(), "/"+args[0], params)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			prov := attachFreshness(DataProvenance{Source: "live"}, flags)
			if wantsHumanTable(cmd.OutOrStdout(), flags) {
				printProvenance(cmd, 1, prov)
			}
			if flags.asJSON || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
				filtered := data
				if flags.selectFields != "" {
					filtered = filterFields(filtered, flags.selectFields)
				} else if flags.compact {
					filtered = compactFields(filtered)
				}
				wrapped, wrapErr := wrapWithProvenance(filtered, prov)
				if wrapErr != nil {
					return wrapErr
				}
				return printOutput(cmd.OutOrStdout(), wrapped, true)
			}
			return printOutputWithFlags(cmd.OutOrStdout(), data, flags)
		},
	}
	cmd.Flags().StringVar(&flagFields, "fields", "", "Comma-separated list of fields to return.")
	return cmd
}

func newGenericNodeUpdateCmd(flags *rootFlags) *cobra.Command {
	var bodyJSON string
	var stdinBody bool

	cmd := &cobra.Command{
		Use:   "update <nodeId>",
		Short: "Update any Graph API node by id",
		Example: `  meta-ads-pp-cli node update 120210000000000001 --body '{"status":"PAUSED"}'
  echo '{"name":"New name"}' | meta-ads-pp-cli node update 120210000000000001 --stdin`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			body, err := readGenericMutateBody(bodyJSON, stdinBody)
			if err != nil {
				return err
			}
			data, statusCode, err := c.PostWithParams(cmd.Context(), "/"+args[0], map[string]string{}, body)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			if !flags.dryRun && statusCode >= 200 && statusCode < 300 {
				writeMutationResponseToStore(cmd.Context(), "node", data, "")
			}
			return printGenericMutateResult(cmd, flags, data, statusCode)
		},
	}
	cmd.Flags().StringVar(&bodyJSON, "body", "", "JSON object of fields to update, e.g. '{\"status\":\"PAUSED\"}'.")
	cmd.Flags().BoolVar(&stdinBody, "stdin", false, "Read the JSON body from stdin instead of --body.")
	return cmd
}

func newGenericNodeDeleteCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <nodeId>",
		Short:   "Delete any Graph API node by id",
		Example: `  meta-ads-pp-cli node delete 120210000000000001`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			data, statusCode, err := c.Delete(cmd.Context(), "/"+args[0])
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return printGenericMutateResult(cmd, flags, data, statusCode)
		},
	}
	return cmd
}

func newGenericEdgeGetCmd(flags *rootFlags) *cobra.Command {
	var flagFields string
	var flagLimit string
	var flagAfter string

	cmd := &cobra.Command{
		Use:   "get <nodeId> <edgeName>",
		Short: "List any Graph API edge off any node by name",
		Example: `  meta-ads-pp-cli edge get act_123456789 activities --fields event_type,event_time
  meta-ads-pp-cli edge get 120210000000000001 leadgen_forms`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			params := map[string]string{}
			if flagFields != "" {
				params["fields"] = flagFields
			}
			if flagLimit != "" {
				params["limit"] = flagLimit
			}
			if flagAfter != "" {
				params["after"] = flagAfter
			}
			data, err := c.Get(cmd.Context(), "/"+args[0]+"/"+args[1], params)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			prov := attachFreshness(DataProvenance{Source: "live"}, flags)
			if wantsHumanTable(cmd.OutOrStdout(), flags) {
				var wrapped struct {
					Data []map[string]any `json:"data"`
				}
				if json.Unmarshal(data, &wrapped) == nil && len(wrapped.Data) > 0 {
					printProvenance(cmd, len(wrapped.Data), prov)
					if err := printAutoTable(cmd.OutOrStdout(), wrapped.Data); err == nil {
						return nil
					}
				}
			}
			if flags.asJSON || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
				filtered := data
				if flags.selectFields != "" {
					filtered = filterFields(filtered, flags.selectFields)
				} else if flags.compact {
					filtered = compactFields(filtered)
				}
				wrapped, wrapErr := wrapWithProvenance(filtered, prov)
				if wrapErr != nil {
					return wrapErr
				}
				return printOutput(cmd.OutOrStdout(), wrapped, true)
			}
			return printOutputWithFlags(cmd.OutOrStdout(), data, flags)
		},
	}
	cmd.Flags().StringVar(&flagFields, "fields", "", "Comma-separated list of fields to return.")
	cmd.Flags().StringVar(&flagLimit, "limit", "", "Max results per page.")
	cmd.Flags().StringVar(&flagAfter, "after", "", "Cursor for the next page (from paging.cursors.after).")
	return cmd
}

func newGenericEdgeCreateCmd(flags *rootFlags) *cobra.Command {
	var bodyJSON string
	var stdinBody bool

	cmd := &cobra.Command{
		Use:   "create <nodeId> <edgeName>",
		Short: "Create a child object on any Graph API edge by name",
		Example: `  meta-ads-pp-cli edge create act_123456789 customaudiences --body '{"name":"Site visitors","subtype":"WEBSITE"}'
  echo '{"message":"Hello"}' | meta-ads-pp-cli edge create 120210000000000002 feed --stdin`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			body, err := readGenericMutateBody(bodyJSON, stdinBody)
			if err != nil {
				return err
			}
			data, statusCode, err := c.PostWithParams(cmd.Context(), "/"+args[0]+"/"+args[1], map[string]string{}, body)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			if !flags.dryRun && statusCode >= 200 && statusCode < 300 {
				writeMutationResponseToStore(cmd.Context(), "edge", data, "")
			}
			return printGenericMutateResult(cmd, flags, data, statusCode)
		},
	}
	cmd.Flags().StringVar(&bodyJSON, "body", "", "JSON object of fields for the new object.")
	cmd.Flags().BoolVar(&stdinBody, "stdin", false, "Read the JSON body from stdin instead of --body.")
	return cmd
}

// readGenericMutateBody resolves the JSON body for a generic node/edge
// mutate command from either --body or --stdin (mutually exclusive; --stdin
// wins if both happen to be set since it's the more explicit ask).
func readGenericMutateBody(bodyJSON string, stdinBody bool) (map[string]any, error) {
	body := map[string]any{}
	if stdinBody {
		stdinData, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("reading stdin: %w", err)
		}
		if err := json.Unmarshal(stdinData, &body); err != nil {
			return nil, fmt.Errorf("parsing stdin JSON: %w", err)
		}
		return body, nil
	}
	if bodyJSON != "" {
		if err := json.Unmarshal([]byte(bodyJSON), &body); err != nil {
			return nil, fmt.Errorf("parsing --body JSON: %w", err)
		}
	}
	return body, nil
}

func printGenericMutateResult(cmd *cobra.Command, flags *rootFlags, data json.RawMessage, statusCode int) error {
	if flags.asJSON || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
		envelope := map[string]any{
			"status":  statusCode,
			"success": statusCode >= 200 && statusCode < 300,
		}
		if flags.dryRun {
			envelope["dry_run"] = true
			envelope["status"] = 0
			envelope["success"] = false
		}
		filtered := data
		if flags.selectFields != "" {
			filtered = filterFields(filtered, flags.selectFields)
		} else if flags.compact {
			filtered = compactFields(filtered)
		}
		if len(filtered) > 0 {
			var parsed any
			if err := json.Unmarshal(filtered, &parsed); err == nil {
				envelope["data"] = parsed
			}
		}
		envelopeJSON, err := json.Marshal(envelope)
		if err != nil {
			return err
		}
		return printOutput(cmd.OutOrStdout(), json.RawMessage(envelopeJSON), true)
	}
	return printOutputWithFlags(cmd.OutOrStdout(), data, flags)
}
