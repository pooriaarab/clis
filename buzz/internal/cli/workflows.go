package cli

// buzz workflows — YAML-as-code workflow definitions (kind:30620) and their
// runtime events. Kinds/tags mirror the Rust oracle at
// /Users/parab/code/buzz/crates/buzz-sdk/src/builders.rs (build_workflow_def,
// build_workflow_update, build_workflow_delete, build_workflow_trigger,
// build_workflow_approval) and the CLI at
// /Users/parab/code/buzz/crates/buzz-cli/src/commands/workflows.rs.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"buzz-cli/internal/client"
	"buzz-cli/internal/config"
	"buzz-cli/internal/nostr"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func workflowsCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "workflows", Short: "Workflow commands"}

	list := &cobra.Command{
		Use:   "list",
		Short: "List workflows in a channel",
		RunE: func(cmd *cobra.Command, args []string) error {
			channel, err := requiredFlag(cmd, "channel")
			if err != nil {
				return err
			}
			if _, err := validateUUIDStr(channel); err != nil {
				return err
			}
			resolved, err := resolveNoKeys(opts)
			if err != nil {
				return err
			}
			raw, err := opts.fetchQuery(cmd.Context(), resolved, nil, []client.Filter{{"kinds": []int{nostr.KindWorkflowDef}, "#h": []string{channel}}})
			if err != nil {
				return err
			}
			return opts.writeRawJSON(normalizeWorkflowList(raw))
		},
	}
	list.Flags().String("channel", "", "channel UUID")

	get := &cobra.Command{
		Use:   "get",
		Short: "Get a workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			workflow, err := requiredFlag(cmd, "workflow")
			if err != nil {
				return err
			}
			if _, err := validateUUIDStr(workflow); err != nil {
				return err
			}
			resolved, err := resolveNoKeys(opts)
			if err != nil {
				return err
			}
			raw, err := opts.fetchQuery(cmd.Context(), resolved, nil, []client.Filter{{"kinds": []int{nostr.KindWorkflowDef}, "#d": []string{workflow}}})
			if err != nil {
				return err
			}
			return opts.writeJSON(firstNormalizedWorkflowOrNull(raw))
		},
	}
	get.Flags().String("workflow", "", "workflow UUID")

	create := &cobra.Command{
		Use:   "create",
		Short: "Create a workflow from a YAML definition",
		RunE: func(cmd *cobra.Command, args []string) error {
			channel, err := requiredFlag(cmd, "channel")
			if err != nil {
				return err
			}
			if _, err := validateUUIDStr(channel); err != nil {
				return err
			}
			yamlFlag, err := requiredFlag(cmd, "yaml")
			if err != nil {
				return err
			}
			yamlDef, err := readOrStdin(yamlFlag)
			if err != nil {
				return err
			}
			if err := validateContentSize(yamlDef, maxContentBytes, "yaml"); err != nil {
				return err
			}
			workflowID := newUUID()
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			event := nostr.NewUnsignedEvent(nostr.KindWorkflowDef, keys.PublicHex(), yamlDef, nostr.Tags{{"d", workflowID}, {"h", channel}}, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign workflow event", err)
			}
			return opts.publishWithField(cmd.Context(), resolved, keys, event, "workflow_id", workflowID)
		},
	}
	create.Flags().String("channel", "", "channel UUID")
	create.Flags().String("yaml", "", "workflow YAML definition ('-' reads stdin)")

	update := &cobra.Command{
		Use:   "update",
		Short: "Update a workflow's YAML definition",
		RunE: func(cmd *cobra.Command, args []string) error {
			channel, err := requiredFlag(cmd, "channel")
			if err != nil {
				return err
			}
			if _, err := validateUUIDStr(channel); err != nil {
				return err
			}
			workflow, err := requiredFlag(cmd, "workflow")
			if err != nil {
				return err
			}
			if _, err := validateUUIDStr(workflow); err != nil {
				return err
			}
			yamlFlag, err := requiredFlag(cmd, "yaml")
			if err != nil {
				return err
			}
			yamlDef, err := readOrStdin(yamlFlag)
			if err != nil {
				return err
			}
			if err := validateContentSize(yamlDef, maxContentBytes, "yaml"); err != nil {
				return err
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			event := nostr.NewUnsignedEvent(nostr.KindWorkflowDef, keys.PublicHex(), yamlDef, nostr.Tags{{"d", workflow}, {"h", channel}}, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign workflow event", err)
			}
			return opts.submitAndNormalize(cmd.Context(), resolved, keys, event)
		},
	}
	update.Flags().String("channel", "", "channel UUID the workflow belongs to")
	update.Flags().String("workflow", "", "workflow UUID")
	update.Flags().String("yaml", "", "updated workflow YAML definition ('-' reads stdin)")

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			workflow, err := requiredFlag(cmd, "workflow")
			if err != nil {
				return err
			}
			if _, err := validateUUIDStr(workflow); err != nil {
				return err
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			event := nostr.NewUnsignedEvent(nostr.KindDeletion, keys.PublicHex(), "", nostr.Tags{{"a", fmt.Sprintf("%d:%s:%s", nostr.KindWorkflowDef, keys.PublicHex(), workflow)}}, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign workflow deletion event", err)
			}
			return opts.submitAndNormalize(cmd.Context(), resolved, keys, event)
		},
	}
	deleteCmd.Flags().String("workflow", "", "workflow UUID")

	trigger := &cobra.Command{
		Use:   "trigger",
		Short: "Trigger a workflow run",
		RunE: func(cmd *cobra.Command, args []string) error {
			workflow, err := requiredFlag(cmd, "workflow")
			if err != nil {
				return err
			}
			if _, err := validateUUIDStr(workflow); err != nil {
				return err
			}
			inputs, _ := cmd.Flags().GetString("inputs")
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			content := ""
			if cmd.Flags().Changed("inputs") {
				var parsed map[string]any
				if err := json.Unmarshal([]byte(inputs), &parsed); err != nil {
					return inputError("--inputs is not valid JSON: " + err.Error())
				}
				content = inputs
			}
			event := nostr.NewUnsignedEvent(nostr.KindWorkflowTrigger, keys.PublicHex(), content, nostr.Tags{{"d", workflow}}, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign workflow trigger event", err)
			}
			return opts.submitAndNormalize(cmd.Context(), resolved, keys, event)
		},
	}
	trigger.Flags().String("workflow", "", "workflow UUID")
	trigger.Flags().String("inputs", "", "JSON object of input variables passed to the workflow as event content")

	runs := &cobra.Command{
		Use:   "runs",
		Short: "List runs for a workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			workflow, err := requiredFlag(cmd, "workflow")
			if err != nil {
				return err
			}
			if _, err := validateUUIDStr(workflow); err != nil {
				return err
			}
			limit, _ := cmd.Flags().GetInt("limit")
			limit = positiveOr(limit, 20)
			if limit > 100 {
				limit = 100
			}
			resolved, err := resolveNoKeys(opts)
			if err != nil {
				return err
			}
			filter := client.Filter{
				"kinds": []int{
					nostr.KindWorkflowTriggered,
					nostr.KindWorkflowStepStarted,
					nostr.KindWorkflowStepCompleted,
				},
				"#d":    []string{workflow},
				"limit": limit,
			}
			raw, err := opts.fetchQuery(cmd.Context(), resolved, nil, []client.Filter{filter})
			if err != nil {
				return err
			}
			return opts.writeRawJSON(raw)
		},
	}
	runs.Flags().String("workflow", "", "workflow UUID")
	runs.Flags().Int("limit", 20, "maximum number of results to return")

	approve := &cobra.Command{
		Use:   "approve",
		Short: "Approve or deny a workflow step",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := requiredFlag(cmd, "token")
			if err != nil {
				return err
			}
			if _, err := validateUUIDStr(token); err != nil {
				return err
			}
			approved, _ := cmd.Flags().GetBool("approved")
			note, _ := cmd.Flags().GetString("note")
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			sum := sha256.Sum256([]byte(token))
			tokenHash := hex.EncodeToString(sum[:])
			kind := nostr.KindApprovalGrant
			if !approved {
				kind = nostr.KindApprovalDeny
			}
			event := nostr.NewUnsignedEvent(kind, keys.PublicHex(), note, nostr.Tags{{"d", tokenHash}}, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign workflow approval event", err)
			}
			return opts.submitAndNormalize(cmd.Context(), resolved, keys, event)
		},
	}
	approve.Flags().String("token", "", "the approval token UUID (from the approval request)")
	approve.Flags().Bool("approved", true, "approve (true) or deny (false) the step")
	approve.Flags().String("note", "", "optional note to include with the approval/denial")

	cmd.AddCommand(list, get, create, update, deleteCmd, trigger, runs, approve)
	return cmd
}

// normalizeWorkflowList reshapes a raw relay event array into the bundled
// CLI's `{workflow_id, content, created_at, pubkey}` projection (mirrors
// cmd_list_workflows in workflows.rs).
func normalizeWorkflowList(raw json.RawMessage) json.RawMessage {
	var events []map[string]any
	if err := json.Unmarshal(raw, &events); err != nil {
		return raw
	}
	out := make([]map[string]any, 0, len(events))
	for _, ev := range events {
		out = append(out, normalizeWorkflowEvent(ev))
	}
	b, err := json.Marshal(out)
	if err != nil {
		return raw
	}
	return b
}

func firstNormalizedWorkflowOrNull(raw json.RawMessage) any {
	var events []map[string]any
	if err := json.Unmarshal(raw, &events); err != nil || len(events) == 0 {
		return nil
	}
	return normalizeWorkflowEvent(events[0])
}

func normalizeWorkflowEvent(ev map[string]any) map[string]any {
	dTag := ""
	for _, tag := range tagsFromAny(ev["tags"]) {
		if len(tag) >= 2 && tag[0] == "d" {
			dTag = tag[1]
			break
		}
	}
	return map[string]any{
		"workflow_id": dTag,
		"content":     stringFromAny(ev["content"]),
		"created_at":  uint64FromAny(ev["created_at"]),
		"pubkey":      stringFromAny(ev["pubkey"]),
	}
}

// submitAndNormalize signs and submits an event, surfacing relay-reported
// write conflicts, and prints the normalized `{event_id, accepted, message}`
// write response (mirrors normalize_write_response in client.rs).
func (opts *rootOptions) submitAndNormalize(ctx context.Context, resolved config.Resolved, keys *nostr.KeyPair, event nostr.Event) error {
	relayClient, err := restClientFromResolved(resolved, keys)
	if err != nil {
		return err
	}
	raw, err := relayClient.PostEvent(ctx, event)
	if err != nil {
		return ExitError{Code: ExitRelay, Message: "publish event failed", Err: err}
	}
	if err := relayPublishError(raw); err != nil {
		return err
	}
	return opts.writeRawJSON(normalizeWriteResponse(raw))
}

// publishWithField signs and submits a create event, injecting idKey: idVal
// into the write response when the relay accepted it (mirrors
// print_create_response / create_response_with_id_if_accepted).
func (opts *rootOptions) publishWithField(ctx context.Context, resolved config.Resolved, keys *nostr.KeyPair, event nostr.Event, idKey, idVal string) error {
	relayClient, err := restClientFromResolved(resolved, keys)
	if err != nil {
		return err
	}
	raw, err := relayClient.PostEvent(ctx, event)
	if err != nil {
		return ExitError{Code: ExitRelay, Message: "publish event failed", Err: err}
	}
	var value map[string]any
	if err := json.Unmarshal(normalizeWriteResponse(raw), &value); err != nil || value == nil {
		value = map[string]any{}
	}
	if accepted, _ := value["accepted"].(bool); accepted {
		value[idKey] = idVal
	}
	return opts.writeJSON(value)
}

// normalizeWriteResponse mirrors normalize_write_response: reshape a relay
// write response to `{event_id, accepted, message}` when recognizable,
// otherwise pass the raw bytes through unchanged.
func normalizeWriteResponse(raw json.RawMessage) json.RawMessage {
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return raw
	}
	_, hasEventID := value["event_id"]
	_, hasAccepted := value["accepted"]
	if !hasEventID && !hasAccepted {
		return raw
	}
	out, err := json.Marshal(map[string]any{
		"event_id": stringFromAny(value["event_id"]),
		"accepted": boolFromAny(value["accepted"]),
		"message":  stringFromAny(value["message"]),
	})
	if err != nil {
		return raw
	}
	return out
}

func boolFromAny(value any) bool {
	b, _ := value.(bool)
	return b
}

func newUUID() string {
	return uuid.New().String()
}
