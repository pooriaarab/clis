// Copyright 2026 Pooria Arab and contributors. Licensed under Apache-2.0.
//
// Hand-written, not printed. The generated promoted_*.go commands (campaigns,
// ad-groups, ad-group-ads, campaign-budgets, assets, conversion-actions) each
// hardcode one resource's plural REST segment against the exact same shape:
// POST /v24/customers/{customerId}/{resourcePlural}:mutate with a JSON
// {"operations": [...]} body. That shape is uniform across every one of the
// ~65 mutable Google Ads REST resources (AdGroupCriterion, BiddingStrategy,
// Experiment, ExperimentArm, Audience, UserList, KeywordPlan*, Label,
// SharedSet, ... see developers.google.com/google-ads/api/rest/reference/rest
// for the full resource list), so instead of doc-scraping and generating one
// wrapper per resource (~65 more LLM passes, each risking a hallucinated
// field/path), this one command takes the REST plural segment as an argument
// and reaches every mutable resource through the same path template.
//
// A few Ads API write operations are NOT plain :mutate and this command does
// NOT cover them: BatchJob (run), Recommendation (apply/dismiss),
// CampaignDraft (promote), OfflineUserDataJob (add-operations/run). Those
// need their own verb-specific endpoint, not operations:mutate.
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

func newMutateResourceCmd(flags *rootFlags) *cobra.Command {
	var bodyOperations string
	var stdinBody bool

	cmd := &cobra.Command{
		Use:   "mutate <resourcePlural> <customerId>",
		Short: "Create, update, or remove any mutable resource by its REST plural name",
		Long: `Generic escape hatch covering every mutable Google Ads REST resource, not
just the ones with a dedicated command (campaigns, ad-groups, ad-group-ads,
campaign-budgets, assets, conversion-actions).

<resourcePlural> is the exact REST path segment from the API reference, e.g.
adGroupCriteria, biddingStrategies, experiments, experimentArms, audiences,
userLists, labels, sharedSets, keywordPlanCampaignKeywords, customerLabels.
Full list: https://developers.google.com/google-ads/api/rest/reference/rest`,
		Example: `# add a keyword to an ad group (AdGroupCriterion create)
google-ads mutate adGroupCriteria 1234567890 --operations '[{"create":{"adGroup":"customers/1234567890/adGroups/222","keyword":{"text":"running shoes","matchType":"PHRASE"}}}]'

# create an experiment (the same thing Google Ads' web UI Experiments tab manages)
google-ads mutate experiments 1234567890 --operations '[{"create":{"name":"Q3 bid test","suffix":"-exp","type":"SEARCH_CUSTOM"}}]'

# pipe generated operations JSON for bulk mutate on any resource
cat ops.json | google-ads mutate audiences 1234567890 --stdin`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			resource := args[0]
			customerID := args[1]
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			path := replacePathParam("/v24/customers/{customerId}/"+resource+":mutate", "customerId", customerID)
			var body any
			if stdinBody {
				stdinData, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("reading stdin: %w", err)
				}
				var parsed any
				if err := json.Unmarshal(stdinData, &parsed); err != nil {
					return fmt.Errorf("parsing stdin JSON: %w", err)
				}
				if arr, ok := parsed.([]any); ok {
					body = map[string]any{"operations": arr}
				} else {
					body = parsed // already a full {"operations": [...]} envelope
				}
			} else {
				bodyMap := map[string]any{}
				body = bodyMap
				if bodyOperations != "" {
					var parsedOperations any
					if err := json.Unmarshal([]byte(bodyOperations), &parsedOperations); err != nil {
						return fmt.Errorf("parsing --operations JSON: %w", err)
					}
					asArray, ok := parsedOperations.([]any)
					if !ok {
						return fmt.Errorf("--operations must be a JSON array, got JSON %T", parsedOperations)
					}
					bodyMap["operations"] = asArray
				}
			}

			data, statusCode, err := c.PostWithParams(cmd.Context(), path, map[string]string{}, body)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			var partialFailure *partialFailureReport
			if !flags.dryRun && statusCode >= 200 && statusCode < 300 {
				partialFailure = detectPartialFailure(data)
				if partialFailure != nil {
					fmt.Fprintf(os.Stderr, "warning: partial failure detected in %s response: %s\n", resource, partialFailure.Message)
				}
			}
			if !flags.dryRun && statusCode >= 200 && statusCode < 300 && (partialFailure == nil || flags.allowPartialFailure) {
				writeMutationResponseToStore(cmd.Context(), resource, data, "")
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
				if perr := printOutput(cmd.OutOrStdout(), wrapped, true); perr != nil {
					return perr
				}
				if partialFailure != nil && !flags.allowPartialFailure {
					return partialFailureErr(fmt.Errorf("partial failure in %s response: %s", resource, partialFailure.Message))
				}
				return nil
			}
			if perr := printOutputWithFlags(cmd.OutOrStdout(), data, flags); perr != nil {
				return perr
			}
			if partialFailure != nil && !flags.allowPartialFailure {
				return partialFailureErr(fmt.Errorf("partial failure in %s response: %s", resource, partialFailure.Message))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&bodyOperations, "operations", "", "JSON array of mutate operations")
	cmd.Flags().BoolVar(&stdinBody, "stdin", false, "Read operations JSON from stdin (array, or full {\"operations\":[...]} envelope)")

	return cmd
}
