package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"buzz-cli/internal/client"
	"buzz-cli/internal/config"
	"buzz-cli/internal/nostr"
	"buzz-cli/internal/types"
	"github.com/google/uuid"
	"github.com/nbd-wtf/go-nostr/nip44"
	"github.com/spf13/cobra"
)

// ── agents update (headless CRUD; no bundled-CLI equivalent — see SPEC.md
// "Agents & fleet") ─────────────────────────────────────────────────────────

func ptrOrNil(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

func agentsUpdateCommand(opts *rootOptions) *cobra.Command {
	var systemPrompt, systemPromptFile, model, provider, respondTo string
	var respondToAllowlist []string
	cmd := &cobra.Command{
		Use:   "update <name|pubkey>",
		Short: "Update a managed agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			resolved, owner, err := opts.resolveOwnerKey(true)
			if err != nil {
				return err
			}
			if key, ok := resolved.File.Identities[target]; ok {
				keyPair, err := nostr.ParsePrivateKey(key)
				if err != nil {
					return inputWrap("parse configured identity", err)
				}
				target = keyPair.PublicHex()
			}
			target = strings.ToLower(strings.TrimSpace(target))
			if !isHex64(target) {
				return inputError("target must be a known identity name or a 64-character hex pubkey")
			}

			hasPrompt := cmd.Flags().Changed("system-prompt") || cmd.Flags().Changed("system-prompt-file")
			hasModel := cmd.Flags().Changed("model")
			hasProvider := cmd.Flags().Changed("provider")
			hasRespondTo := cmd.Flags().Changed("respond-to")
			hasAllowlist := cmd.Flags().Changed("respond-to-allowlist")
			if !hasPrompt && !hasModel && !hasProvider && !hasRespondTo && !hasAllowlist {
				return inputError("at least one field required (--system-prompt/-file, --model, --provider, --respond-to, --respond-to-allowlist)")
			}

			events, err := opts.queryEvents(cmd.Context(), []client.Filter{{
				"kinds":   []int{nostr.KindManagedAgent},
				"#d":      []string{target},
				"authors": []string{owner.PublicHex()},
				"limit":   1,
			}})
			if err != nil {
				return err
			}
			if len(events) == 0 {
				return ExitError{Code: ExitOther, Message: "managed agent not found: " + target}
			}

			var content types.ManagedAgentEventContent
			if err := json.Unmarshal([]byte(events[0].Content), &content); err != nil {
				return otherWrap("parse managed agent content", err)
			}

			if hasPrompt {
				prompt := systemPrompt
				if systemPromptFile != "" {
					b, err := os.ReadFile(systemPromptFile)
					if err != nil {
						return inputWrap("read system prompt file", err)
					}
					prompt = string(b)
				}
				if strings.TrimSpace(prompt) == "" {
					return inputError("system prompt is required")
				}
				content.SystemPrompt = ptrOrNil(prompt)
			}
			if hasModel {
				content.Model = ptrOrNil(model)
			}
			if hasProvider {
				content.Provider = ptrOrNil(provider)
			}
			if hasRespondTo {
				if respondTo != types.RespondToOwnerOnly && respondTo != types.RespondToAllowlist && respondTo != types.RespondToAnyone {
					return inputError("--respond-to must be owner-only, allowlist, or anyone")
				}
				content.RespondTo = respondTo
			}
			if hasAllowlist {
				content.RespondToAllowlist = compactStrings(respondToAllowlist)
			}

			body, err := json.Marshal(content)
			if err != nil {
				return otherWrap("encode managed agent content", err)
			}
			event := nostr.NewUnsignedEvent(nostr.KindManagedAgent, owner.PublicHex(), string(body), nostr.Tags{{"d", target}}, 0)
			if err := event.Sign(owner); err != nil {
				return otherWrap("sign managed agent event", err)
			}
			return opts.publish(cmd.Context(), resolved, owner, event)
		},
	}
	cmd.Flags().StringVar(&systemPrompt, "system-prompt", "", "new system prompt")
	cmd.Flags().StringVar(&systemPromptFile, "system-prompt-file", "", "new system prompt file")
	cmd.Flags().StringVar(&model, "model", "", "new model")
	cmd.Flags().StringVar(&provider, "provider", "", "new provider")
	cmd.Flags().StringVar(&respondTo, "respond-to", "", "respond-to mode: owner-only, allowlist, anyone")
	cmd.Flags().StringSliceVar(&respondToAllowlist, "respond-to-allowlist", nil, "allowlisted pubkeys (used when --respond-to allowlist)")
	return cmd
}

// ── agents draft-create / draft-update ──────────────────────────────────────
//
// These publish an ephemeral kind:24200 agent-observer-frame event whose
// content is a NIP-44-encrypted JSON "agent_management_request" payload,
// exactly as buzz-cli/src/agent_management.rs + commands/agents.rs. Buzz
// Desktop decrypts it and shows the owner a prefilled create/edit form;
// nothing is created or changed until the owner explicitly saves it.

const (
	agentManagementRequestKind = "agent_management_request"
	maxDraftNameChars          = 120
	maxDraftPromptChars        = 20000
	maxDraftOptionalChars      = 300
	observerFrameTelemetry     = "telemetry"
)

type agentManagementRequest struct {
	Type      string `json:"type"`
	Action    string `json:"action"`
	RequestID string `json:"requestId"`
	Request   any    `json:"request"`
}

type observerEventPayload struct {
	Seq       uint64                 `json:"seq"`
	Timestamp string                 `json:"timestamp"`
	Kind      string                 `json:"kind"`
	ChannelID *string                `json:"channelId"`
	Payload   agentManagementRequest `json:"payload"`
}

type createAgentDraftRequest struct {
	ChannelID    string `json:"channelId"`
	DisplayName  string `json:"displayName"`
	SystemPrompt string `json:"systemPrompt"`
}

type updateAgentDraftRequest struct {
	ChannelID    string  `json:"channelId"`
	AgentName    string  `json:"agentName"`
	DisplayName  *string `json:"displayName,omitempty"`
	SystemPrompt *string `json:"systemPrompt,omitempty"`
	Runtime      *string `json:"runtime,omitempty"`
	Provider     *string `json:"provider,omitempty"`
	Model        *string `json:"model,omitempty"`
	RespondTo    *string `json:"respondTo,omitempty"`
}

func requiredDraftField(value, label string, maxChars int) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", inputError(label + " is required")
	}
	if utf8.RuneCountInString(trimmed) > maxChars {
		return "", inputError(fmt.Sprintf("%s is too long (max %d characters)", label, maxChars))
	}
	return trimmed, nil
}

// requireDraftOwner extracts the owner pubkey from the configured NIP-OA
// auth tag — draft requests are always encrypted to that owner, matching
// agents.rs require_owner.
func (opts *rootOptions) requireDraftOwner(resolved config.Resolved) (string, error) {
	tag, err := nostr.ParseAuthTagJSON(resolved.AuthTag)
	if err != nil {
		return "", inputWrap("parse auth tag", err)
	}
	if len(tag) < 2 || tag[0] != "auth" {
		return "", ExitError{Code: ExitAuth, Message: "agent draft requests require BUZZ_AUTH_TAG"}
	}
	owner := strings.ToLower(strings.TrimSpace(tag[1]))
	if !isHex64(owner) {
		return "", authWrap("invalid owner attestation", fmt.Errorf("owner pubkey %q is not 64-char hex", tag[1]))
	}
	return owner, nil
}

// buildDraftRequestEvent serializes and NIP-44 encrypts the draft request
// payload for ownerHex, builds the kind:24200 observer frame, and signs it
// with the agent's own keys. Mirrors agent_management.rs `build`.
func buildDraftRequestEvent(keys *nostr.KeyPair, ownerHex, channelID, action string, request any) (nostr.Event, string, error) {
	requestID := uuid.NewString()
	payload := observerEventPayload{
		Seq:       0,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Kind:      agentManagementRequestKind,
		ChannelID: &channelID,
		Payload: agentManagementRequest{
			Type:      agentManagementRequestKind,
			Action:    action,
			RequestID: requestID,
			Request:   request,
		},
	}
	plaintext, err := json.Marshal(payload)
	if err != nil {
		return nostr.Event{}, "", otherWrap("encode draft request", err)
	}
	kC, err := nip44.GenerateConversationKey(ownerHex, keys.SecretHex())
	if err != nil {
		return nostr.Event{}, "", otherWrap("derive conversation key", err)
	}
	ciphertext, err := nip44.Encrypt(string(plaintext), kC)
	if err != nil {
		return nostr.Event{}, "", otherWrap("encrypt draft request", err)
	}
	tags := nostr.Tags{{"p", ownerHex}, {"agent", keys.PublicHex()}, {"frame", observerFrameTelemetry}}
	event := nostr.NewUnsignedEvent(nostr.KindAgentObserverFrame, keys.PublicHex(), ciphertext, tags, 0)
	if err := event.Sign(keys); err != nil {
		return nostr.Event{}, "", otherWrap("sign draft request", err)
	}
	return event, requestID, nil
}

// publishEphemeralEvent publishes an ephemeral (non-stored) event over the
// relay WebSocket. Like opts.publishWS, this fires the publish without
// waiting for the relay's OK frame — consistent with the rest of this CLI's
// WS publish path (see users set-presence).
func (opts *rootOptions) publishEphemeralEvent(ctx context.Context, resolved config.Resolved, keys *nostr.KeyPair, event nostr.Event) error {
	if resolved.RelayURL == "" {
		return inputError("relay URL is required")
	}
	tag, err := nostr.ParseAuthTagJSON(resolved.AuthTag)
	if err != nil {
		return inputWrap("parse auth tag", err)
	}
	ws, err := client.DialWS(ctx, resolved.RelayURL, keys, tag)
	if err != nil {
		return ExitError{Code: ExitRelay, Message: "connect relay websocket failed", Err: err}
	}
	defer ws.Close(1000, "done")
	if err := ws.Publish(ctx, event); err != nil {
		return ExitError{Code: ExitRelay, Message: "publish websocket event failed", Err: err}
	}
	return nil
}

func (opts *rootOptions) writeDraftResponse(eventID, requestID, action string) error {
	return opts.writeJSON(map[string]any{
		"ok":         true,
		"event_id":   eventID,
		"request_id": requestID,
		"action":     action,
		"saved":      false,
		"message":    "Draft sent to Buzz Desktop for owner review. Nothing changes until the owner saves it.",
	})
}

func agentsDraftCreateCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "draft-create",
		Short: "Open a prefilled create-agent form in the owner's Buzz Desktop",
		RunE: func(cmd *cobra.Command, args []string) error {
			channel, err := requiredFlag(cmd, "channel")
			if err != nil {
				return err
			}
			if _, err := uuid.Parse(channel); err != nil {
				return inputWrap("invalid channel UUID", err)
			}
			displayNameFlag, err := requiredFlag(cmd, "display-name")
			if err != nil {
				return err
			}
			systemPromptFlag, err := requiredFlag(cmd, "system-prompt")
			if err != nil {
				return err
			}
			systemPromptRaw, err := readOrStdin(systemPromptFlag)
			if err != nil {
				return err
			}
			displayName, err := requiredDraftField(displayNameFlag, "display name", maxDraftNameChars)
			if err != nil {
				return err
			}
			systemPrompt, err := requiredDraftField(systemPromptRaw, "system prompt", maxDraftPromptChars)
			if err != nil {
				return err
			}

			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			ownerHex, err := opts.requireDraftOwner(resolved)
			if err != nil {
				return err
			}

			event, requestID, err := buildDraftRequestEvent(keys, ownerHex, channel, "create", createAgentDraftRequest{
				ChannelID:    channel,
				DisplayName:  displayName,
				SystemPrompt: systemPrompt,
			})
			if err != nil {
				return err
			}
			if err := opts.publishEphemeralEvent(cmd.Context(), resolved, keys, event); err != nil {
				return err
			}
			return opts.writeDraftResponse(event.ID, requestID, "create")
		},
	}
	cmd.Flags().String("channel", "", "Current channel UUID; the new agent is added here after save")
	cmd.Flags().String("display-name", "", "Proposed agent name")
	cmd.Flags().String("system-prompt", "", "Proposed instructions; use '-' to read from stdin")
	return cmd
}

func agentsDraftUpdateCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "draft-update",
		Short: "Open a prefilled edit-agent form in the owner's Buzz Desktop",
		RunE: func(cmd *cobra.Command, args []string) error {
			channel, err := requiredFlag(cmd, "channel")
			if err != nil {
				return err
			}
			if _, err := uuid.Parse(channel); err != nil {
				return inputWrap("invalid channel UUID", err)
			}
			agentNameFlag, err := requiredFlag(cmd, "agent-name")
			if err != nil {
				return err
			}
			agentName, err := requiredDraftField(agentNameFlag, "agent name", maxDraftNameChars)
			if err != nil {
				return err
			}

			request := updateAgentDraftRequest{ChannelID: channel, AgentName: agentName}
			hasUpdate := false

			if cmd.Flags().Changed("display-name") {
				v, _ := cmd.Flags().GetString("display-name")
				field, err := requiredDraftField(v, "display name", maxDraftOptionalChars)
				if err != nil {
					return err
				}
				request.DisplayName = &field
				hasUpdate = true
			}
			if cmd.Flags().Changed("system-prompt") {
				v, _ := cmd.Flags().GetString("system-prompt")
				raw, err := readOrStdin(v)
				if err != nil {
					return err
				}
				field, err := requiredDraftField(raw, "system prompt", maxDraftPromptChars)
				if err != nil {
					return err
				}
				request.SystemPrompt = &field
				hasUpdate = true
			}
			if cmd.Flags().Changed("runtime") {
				v, _ := cmd.Flags().GetString("runtime")
				field, err := requiredDraftField(v, "runtime", maxDraftOptionalChars)
				if err != nil {
					return err
				}
				request.Runtime = &field
				hasUpdate = true
			}
			if cmd.Flags().Changed("provider") {
				v, _ := cmd.Flags().GetString("provider")
				field, err := requiredDraftField(v, "provider", maxDraftOptionalChars)
				if err != nil {
					return err
				}
				request.Provider = &field
				hasUpdate = true
			}
			if cmd.Flags().Changed("model") {
				v, _ := cmd.Flags().GetString("model")
				field, err := requiredDraftField(v, "model", maxDraftOptionalChars)
				if err != nil {
					return err
				}
				request.Model = &field
				hasUpdate = true
			}
			if cmd.Flags().Changed("respond-to") {
				v, _ := cmd.Flags().GetString("respond-to")
				if v != "owner-only" && v != "anyone" {
					return inputError("respond-to must be owner-only or anyone")
				}
				request.RespondTo = &v
				hasUpdate = true
			}
			if !hasUpdate {
				return inputError("include at least one field to update")
			}

			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			ownerHex, err := opts.requireDraftOwner(resolved)
			if err != nil {
				return err
			}

			event, requestID, err := buildDraftRequestEvent(keys, ownerHex, channel, "update", request)
			if err != nil {
				return err
			}
			if err := opts.publishEphemeralEvent(cmd.Context(), resolved, keys, event); err != nil {
				return err
			}
			return opts.writeDraftResponse(event.ID, requestID, "update")
		},
	}
	cmd.Flags().String("channel", "", "Current channel UUID")
	cmd.Flags().String("agent-name", "", "Current name of the personal agent to update")
	cmd.Flags().String("display-name", "", "")
	cmd.Flags().String("system-prompt", "", "Replacement instructions; use '-' to read from stdin")
	cmd.Flags().String("runtime", "", "")
	cmd.Flags().String("provider", "", "")
	cmd.Flags().String("model", "", "")
	cmd.Flags().String("respond-to", "", "owner-only or anyone")
	return cmd
}

// ── agents archive / unarchive / archived (NIP-IA) ──────────────────────────

const maxReasonBytes = 64 // buzz-sdk/src/builders.rs MAX_REASON_BYTES

func validateReasonCode(reason string) error {
	if len(reason) > maxReasonBytes {
		return inputError(fmt.Sprintf("reason code exceeds maximum length of %d UTF-8 bytes (got %d)", maxReasonBytes, len(reason)))
	}
	for _, r := range reason {
		if unicode.IsControl(r) {
			return inputError("reason code must not contain control characters")
		}
	}
	return nil
}

// resolveOwnerAuthTag resolves the optional NIP-OA auth tag proving the
// signer is the target identity's owner, needed for the owner-of-agent
// archive/unarchive consent path. Self-targeting needs no auth. Mirrors
// agents.rs resolve_auth + extract_owner_auth_tag.
func (opts *rootOptions) resolveOwnerAuthTag(ctx context.Context, targetHex, signerHex string) (nostr.Tag, error) {
	if strings.EqualFold(targetHex, signerHex) {
		return nil, nil
	}
	events, err := opts.queryEvents(ctx, []client.Filter{{"kinds": []int{nostr.KindProfile}, "authors": []string{targetHex}, "limit": 1}})
	if err != nil {
		return nil, otherWrap("fetch target profile", err)
	}
	if len(events) == 0 {
		return nil, nil
	}
	return extractOwnerAuthTag(events[0].Tags, signerHex), nil
}

func extractOwnerAuthTag(tags nostr.Tags, signerHex string) nostr.Tag {
	var candidate nostr.Tag
	count := 0
	for _, tag := range tags {
		if len(tag) >= 1 && tag[0] == "auth" {
			count++
			candidate = tag
		}
	}
	if count != 1 || len(candidate) != 4 {
		return nil
	}
	owner := candidate[1]
	if !strings.EqualFold(owner, signerHex) {
		return nil
	}
	ownerLower := strings.ToLower(owner)
	sig := candidate[3]
	if !isHex64(ownerLower) || len(sig) != 128 || !isLowerHexOrUpper(sig) {
		return nil
	}
	return nostr.Tag{"auth", ownerLower, candidate[2], sig}
}

// submitEvent posts a signed event to the relay without producing its own
// JSON output, so callers (archive/unarchive) can shape a custom response.
func (opts *rootOptions) submitEvent(ctx context.Context, resolved config.Resolved, keys *nostr.KeyPair, event nostr.Event) error {
	if resolved.RelayURL == "" {
		return inputError("relay URL is required")
	}
	tag, err := nostr.ParseAuthTagJSON(resolved.AuthTag)
	if err != nil {
		return inputWrap("parse auth tag", err)
	}
	relayClient := client.New(resolved.RelayURL, keys, tag)
	if _, err := relayClient.PostEvent(ctx, event); err != nil {
		return ExitError{Code: ExitRelay, Message: "publish event failed", Err: err}
	}
	return nil
}

func identityArchiveTags(target string, reason, replacedBy string, auth nostr.Tag) (nostr.Tags, error) {
	// NIP-70: mark as protected administrative state.
	tags := nostr.Tags{{"-"}, {"p", target}}
	if reason != "" {
		if err := validateReasonCode(reason); err != nil {
			return nil, err
		}
		tags = append(tags, nostr.Tag{"reason", reason})
	}
	if replacedBy != "" {
		replacedByLower := strings.ToLower(strings.TrimSpace(replacedBy))
		if !isHex64(replacedByLower) {
			return nil, inputError("--replaced-by must be a 64-character hex string")
		}
		if replacedByLower == target {
			return nil, inputError("--replaced-by must differ from the target")
		}
		tags = append(tags, nostr.Tag{"replaced-by", replacedByLower})
	}
	if len(auth) == 4 {
		tags = append(tags, nostr.Tag{"auth", auth[1], auth[2], auth[3]})
	}
	return tags, nil
}

func agentsArchiveCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive <target-pubkey>",
		Short: "Submit a NIP-IA archive request for an identity (kind 9035)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := strings.ToLower(strings.TrimSpace(args[0]))
			if !isHex64(target) {
				return inputError("target pubkey must be a 64-character hex string")
			}
			reason, _ := cmd.Flags().GetString("reason")
			replacedBy, _ := cmd.Flags().GetString("replaced-by")
			content, _ := cmd.Flags().GetString("content")
			if err := validateContentSize(content, maxContentBytes, "content"); err != nil {
				return err
			}

			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			auth, err := opts.resolveOwnerAuthTag(cmd.Context(), target, keys.PublicHex())
			if err != nil {
				return err
			}
			tags, err := identityArchiveTags(target, reason, replacedBy, auth)
			if err != nil {
				return err
			}

			event := nostr.NewUnsignedEvent(nostr.KindIAArchiveRequest, keys.PublicHex(), content, tags, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign archive request", err)
			}
			if err := opts.submitEvent(cmd.Context(), resolved, keys, event); err != nil {
				return err
			}
			return opts.writeJSON(map[string]any{"ok": true, "event_id": event.ID, "action": "archive", "target": target})
		},
	}
	cmd.Flags().String("reason", "", "Machine-readable reason code, max 64 UTF-8 bytes")
	cmd.Flags().String("replaced-by", "", "Rotation pointer pubkey (hex); must differ from the target")
	cmd.Flags().String("content", "", "Optional human-readable note (not parsed for authorization)")
	return cmd
}

func agentsUnarchiveCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unarchive <target-pubkey>",
		Short: "Submit a NIP-IA unarchive request for an identity (kind 9036)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := strings.ToLower(strings.TrimSpace(args[0]))
			if !isHex64(target) {
				return inputError("target pubkey must be a 64-character hex string")
			}
			reason, _ := cmd.Flags().GetString("reason")
			content, _ := cmd.Flags().GetString("content")
			if err := validateContentSize(content, maxContentBytes, "content"); err != nil {
				return err
			}

			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			auth, err := opts.resolveOwnerAuthTag(cmd.Context(), target, keys.PublicHex())
			if err != nil {
				return err
			}
			tags, err := identityArchiveTags(target, reason, "", auth)
			if err != nil {
				return err
			}

			event := nostr.NewUnsignedEvent(nostr.KindIAUnarchiveRequest, keys.PublicHex(), content, tags, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign unarchive request", err)
			}
			if err := opts.submitEvent(cmd.Context(), resolved, keys, event); err != nil {
				return err
			}
			return opts.writeJSON(map[string]any{"ok": true, "event_id": event.ID, "action": "unarchive", "target": target})
		},
	}
	cmd.Flags().String("reason", "", "Machine-readable reason code, max 64 UTF-8 bytes")
	cmd.Flags().String("content", "", "Optional human-readable note (not parsed for authorization)")
	return cmd
}

// fetchArchivedSnapshot fetches the relay's NIP-11 `self` pubkey, queries
// its kind:13535 archived-identities snapshot, and verifies it before
// trusting the contents. Mirrors agents.rs fetch_archived_snapshot.
func (opts *rootOptions) fetchArchivedSnapshot(ctx context.Context) ([]string, error) {
	resolved, err := config.Resolve(config.Options{
		ConfigPath: opts.ConfigPath,
		RelayURL:   opts.RelayURL,
		Identity:   opts.Identity,
		PrivateKey: opts.PrivateKey,
		AuthTag:    opts.AuthTag,
		OwnerKey:   opts.OwnerKey,
	})
	if err != nil {
		return nil, otherWrap("resolve config", err)
	}
	if resolved.RelayURL == "" {
		return nil, inputError("relay URL is required")
	}
	relayClient := client.New(resolved.RelayURL, nil, nil)
	raw, err := relayClient.GetRelayInfo(ctx)
	if err != nil {
		return nil, otherWrap("fetch relay info document", err)
	}
	var info struct {
		Self string `json:"self"`
	}
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil, otherWrap("parse relay info document", err)
	}
	if strings.TrimSpace(info.Self) == "" {
		return nil, otherWrap("parse relay info document", errors.New("relay info document missing 'self' field"))
	}
	selfHex := strings.ToLower(strings.TrimSpace(info.Self))
	if !isHex64(selfHex) {
		return nil, otherWrap("parse relay info document", fmt.Errorf("relay 'self' field is not a valid 64-hex pubkey: %s", info.Self))
	}

	events, err := opts.queryEvents(ctx, []client.Filter{{"kinds": []int{nostr.KindIAArchivedList}, "authors": []string{selfHex}, "limit": 1}})
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return []string{}, nil
	}
	return verifyArchivedEvent(events[0], selfHex)
}

// verifyArchivedEvent applies the NIP-IA trust checks to a kind:13535
// snapshot event: correct kind, author == relay self, exactly one
// well-formed NIP-70 "-" protection tag, and a valid signature. Any
// failure is an error, never a silent empty result. Mirrors agents.rs
// verify_archived_event.
func verifyArchivedEvent(event nostr.Event, relaySelfHex string) ([]string, error) {
	if event.Kind != nostr.KindIAArchivedList {
		return nil, otherWrap("verify archived snapshot", fmt.Errorf("archived-identities event has wrong kind: %d", event.Kind))
	}
	if !strings.EqualFold(event.PubKey, relaySelfHex) {
		return nil, otherWrap("verify archived snapshot", fmt.Errorf("archived-identities event author %s does not match relay self %s", event.PubKey, relaySelfHex))
	}

	nip70Count := 0
	for _, tag := range event.Tags {
		if len(tag) == 0 || tag[0] != "-" {
			continue
		}
		if len(tag) != 1 {
			return nil, otherWrap("verify archived snapshot", errors.New("archived-identities event has a malformed NIP-70 '-' tag (expected arity 1)"))
		}
		nip70Count++
	}
	if nip70Count != 1 {
		return nil, otherWrap("verify archived snapshot", fmt.Errorf("archived-identities event must have exactly one NIP-70 '-' tag, found %d", nip70Count))
	}

	ok, err := event.Verify()
	if err != nil {
		return nil, otherWrap("verify archived snapshot", fmt.Errorf("archived-identities event failed cryptographic verification: %w", err))
	}
	if !ok {
		return nil, otherWrap("verify archived snapshot", errors.New("archived-identities event failed cryptographic verification: signature mismatch"))
	}

	archived := make([]string, 0)
	for _, tag := range event.Tags {
		if len(tag) >= 2 && tag[0] == "p" {
			pk := tag[1]
			if isHex64(pk) {
				archived = append(archived, pk)
			}
		}
	}
	return archived, nil
}

func agentsArchivedCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archived",
		Short: "Read the relay's current NIP-IA archive snapshot (kind 13535)",
		RunE: func(cmd *cobra.Command, args []string) error {
			archived, err := opts.fetchArchivedSnapshot(cmd.Context())
			if err != nil {
				return err
			}
			return opts.writeJSON(map[string]any{"archived": archived})
		},
	}
	return cmd
}
