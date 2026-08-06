package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"buzz-cli/internal/client"
	"buzz-cli/internal/desktopstore"
	"buzz-cli/internal/nostr"
	"buzz-cli/internal/types"
	"github.com/spf13/cobra"
)

var newDesktopSecretStore = desktopstore.NewDarwinKeychainStore

// defaultDesktopTimeout bounds the whole `desktop create`/`desktop delete`
// relay-publish phase (--timeout). desktopRelayCallTimeout bounds each
// individual REST publish so one slow/unreachable relay can never hang the
// command - local store+keychain writes always happen regardless of relay
// outcome.
const (
	defaultDesktopTimeout   = 30 * time.Second
	desktopRelayCallTimeout = 15 * time.Second
)

type desktopListProbe struct {
	Name         string `json:"name"`
	PubKey       string `json:"pubkey"`
	AgentCommand string `json:"agent_command"`
	RelayURL     string `json:"relay_url"`
	IsBuiltin    bool   `json:"is_builtin"`
	IsActive     bool   `json:"is_active"`
}

type desktopDeleteProbe struct {
	Name      string `json:"name"`
	PubKey    string `json:"pubkey"`
	RelayURL  string `json:"relay_url"`
	IsBuiltin bool   `json:"is_builtin"`
}

type desktopCreateInput struct {
	Name            string
	Harness         string
	Community       string
	SystemPrompt    string
	Channels        []string
	Avatar          string
	Model           string
	StorePath       string
	KeychainService string
	Timeout         time.Duration
}

type desktopCreateResult struct {
	Name                   string   `json:"name"`
	PubKey                 string   `json:"pubkey"`
	Nsec                   string   `json:"nsec"`
	AuthTag                string   `json:"auth_tag"`
	StorePath              string   `json:"store_path"`
	KeychainService        string   `json:"keychain_service"`
	KeychainStored         bool     `json:"keychain_stored"`
	KeychainFallbackInline bool     `json:"keychain_fallback_inline"`
	EventIDs               []string `json:"event_ids"`
	RelayErrors            []string `json:"relay_errors"`
}

type desktopDeleteInput struct {
	Target          string
	Force           bool
	Community       string
	StorePath       string
	KeychainService string
	Timeout         time.Duration
}

type desktopDeleteResult struct {
	Name                string   `json:"name"`
	PubKey              string   `json:"pubkey"`
	RemovedFromStore    bool     `json:"removed_from_store"`
	KeychainDeleted     bool     `json:"keychain_deleted"`
	KeychainError       *string  `json:"keychain_error"`
	RelayDeleteEventID  *string  `json:"relay_delete_event_id"`
	RelayArchiveEventID *string  `json:"relay_archive_event_id"`
	RelayErrors         []string `json:"relay_errors"`
	RelaySkippedReason  *string  `json:"relay_skipped_reason,omitempty"`
}

func desktopCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "desktop", Short: "Manage Buzz Desktop-native managed agents (managed-agents.json + keychain + relay)"}
	cmd.AddCommand(desktopListCommand(opts))
	cmd.AddCommand(desktopCreateCommand(opts))
	cmd.AddCommand(desktopDeleteCommand(opts))
	return cmd
}

func desktopListCommand(opts *rootOptions) *cobra.Command {
	var storePath string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Buzz Desktop managed agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveDesktopStorePath(storePath)
			if err != nil {
				return otherWrap("resolve desktop store path", err)
			}
			records, err := desktopstore.LoadRaw(path)
			if err != nil {
				return otherWrap("load desktop store", err)
			}
			list := make([]desktopListProbe, 0, len(records))
			for _, raw := range records {
				var probe desktopListProbe
				if err := json.Unmarshal(raw, &probe); err != nil {
					continue
				}
				list = append(list, probe)
			}
			return opts.writeJSON(list)
		},
	}
	cmd.Flags().StringVar(&storePath, "store-path", "", "managed-agents.json path")
	return cmd
}

func desktopCreateCommand(opts *rootOptions) *cobra.Command {
	var input desktopCreateInput
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a Buzz Desktop managed agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := requiredFlag(cmd, "name")
			if err != nil {
				return err
			}
			harness, err := requiredFlag(cmd, "harness")
			if err != nil {
				return err
			}
			systemPrompt, err := requiredFlag(cmd, "system-prompt")
			if err != nil {
				return err
			}
			input.Name = name
			input.Harness = harness
			input.SystemPrompt = systemPrompt
			return opts.desktopCreate(cmd.Context(), input)
		},
	}
	cmd.Flags().StringVar(&input.Name, "name", "", "agent name")
	cmd.Flags().StringVar(&input.Harness, "harness", "", "agent harness command")
	cmd.Flags().StringVar(&input.Community, "community", "", "community relay URL")
	cmd.Flags().StringVar(&input.SystemPrompt, "system-prompt", "", "system prompt")
	cmd.Flags().StringSliceVar(&input.Channels, "channels", nil, "channel ids")
	cmd.Flags().StringVar(&input.Avatar, "avatar", "", "avatar URL")
	cmd.Flags().StringVar(&input.Model, "model", "", "model")
	cmd.Flags().StringVar(&input.StorePath, "store-path", "", "managed-agents.json path")
	cmd.Flags().StringVar(&input.KeychainService, "keychain-service", "", "macOS keychain service")
	cmd.Flags().DurationVar(&input.Timeout, "timeout", defaultDesktopTimeout, "overall relay-publish timeout; local store+keychain writes never wait on this")
	return cmd
}

func desktopDeleteCommand(opts *rootOptions) *cobra.Command {
	var input desktopDeleteInput
	cmd := &cobra.Command{
		Use:   "delete <name|pubkey>",
		Short: "Delete a Buzz Desktop managed agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input.Target = args[0]
			return opts.desktopDelete(cmd.Context(), input)
		},
	}
	cmd.Flags().BoolVar(&input.Force, "force", false, "delete a built-in agent")
	cmd.Flags().StringVar(&input.Community, "community", "", "community relay URL")
	cmd.Flags().StringVar(&input.StorePath, "store-path", "", "managed-agents.json path")
	cmd.Flags().StringVar(&input.KeychainService, "keychain-service", "", "macOS keychain service")
	cmd.Flags().DurationVar(&input.Timeout, "timeout", defaultDesktopTimeout, "overall relay-delete timeout; local store+keychain removal never waits on this")
	return cmd
}

func resolveDesktopStorePath(flagVal string) (string, error) {
	if strings.TrimSpace(flagVal) != "" {
		return strings.TrimSpace(flagVal), nil
	}
	if env := strings.TrimSpace(os.Getenv("BUZZ_DESKTOP_STORE_PATH")); env != "" {
		return env, nil
	}
	return desktopstore.DefaultStorePath()
}

func resolveDesktopKeychainService(flagVal string) string {
	if strings.TrimSpace(flagVal) != "" {
		return strings.TrimSpace(flagVal)
	}
	if env := strings.TrimSpace(os.Getenv("BUZZ_DESKTOP_KEYCHAIN_SERVICE")); env != "" {
		return env
	}
	return "buzz-desktop"
}

func (opts *rootOptions) desktopCreate(ctx context.Context, input desktopCreateInput) error {
	overallTimeout := input.Timeout
	if overallTimeout <= 0 {
		overallTimeout = defaultDesktopTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, overallTimeout)
	defer cancel()

	resolved, owner, err := opts.resolveOwnerKey(true)
	if err != nil {
		return err
	}
	relayURL := resolved.RelayURL
	if strings.TrimSpace(input.Community) != "" {
		relayURL = strings.TrimSpace(input.Community)
	}
	if relayURL == "" {
		return inputError("relay URL is required (--community, --relay, BUZZ_RELAY_URL, or config)")
	}
	storePath, err := resolveDesktopStorePath(input.StorePath)
	if err != nil {
		return otherWrap("resolve desktop store path", err)
	}
	keychainService := resolveDesktopKeychainService(input.KeychainService)

	agentKeys, err := nostr.NewKeyPair()
	if err != nil {
		return otherWrap("generate agent key", err)
	}
	authTag, err := nostr.MintAuthTag(owner, agentKeys.PublicHex(), "")
	if err != nil {
		return authWrap("mint auth tag", err)
	}
	authTagJSON, err := nostr.AuthTagJSON(authTag)
	if err != nil {
		return otherWrap("encode auth tag", err)
	}
	events, err := types.BuildManagedAgentCreateEvents(types.ManagedAgentCreateInput{
		AgentPubKey:  agentKeys.PublicHex(),
		OwnerPubKey:  owner.PublicHex(),
		Name:         input.Name,
		SystemPrompt: input.SystemPrompt,
		AvatarURL:    input.Avatar,
		Runtime:      input.Harness,
		Model:        input.Model,
		Parallelism:  10,
		RespondTo:    types.RespondToOwnerOnly,
		Channels:     input.Channels,
		AuthTag:      authTag,
	})
	if err != nil {
		return inputWrap("build managed agent events", err)
	}

	signed := []nostr.Event{events.Profile, events.Persona, events.ManagedAgent}
	signed = append(signed, events.ChannelMemberships...)
	if err := signed[0].Sign(agentKeys); err != nil {
		return otherWrap("sign profile event", err)
	}
	for i := 1; i < len(signed); i++ {
		if err := signed[i].Sign(owner); err != nil {
			return otherWrap("sign owner event", err)
		}
	}

	// Publish over REST (POST /events), never a blocking WS session. Each
	// publish gets its own hard deadline so one slow/unreachable relay can
	// never hang the whole command; the overall --timeout above is the
	// outer bound. Local store+keychain writes below happen unconditionally,
	// independent of whether any of these publishes succeeded.
	agentClient := client.New(relayURL, agentKeys, authTag)
	ownerClient := client.New(relayURL, owner, nil)
	eventIDs := make([]string, 0, len(signed))
	relayErrors := []string{}
	for i, event := range signed {
		rc := ownerClient
		if i == 0 {
			rc = agentClient
		}
		callCtx, callCancel := context.WithTimeout(ctx, desktopRelayCallTimeout)
		_, err := rc.PostEvent(callCtx, event)
		callCancel()
		if err != nil {
			relayErrors = append(relayErrors, err.Error())
			continue
		}
		eventIDs = append(eventIDs, event.ID)
	}

	nsec, err := agentKeys.Nsec()
	if err != nil {
		return otherWrap("encode agent nsec", err)
	}
	keychainFallbackInline := false
	privateKeyNsec := ""
	secretStore := newDesktopSecretStore(keychainService)
	if err := secretStore.Store("agent:"+strings.ToLower(agentKeys.PublicHex()), nsec); err != nil {
		keychainFallbackInline = true
		privateKeyNsec = nsec
	}

	now := time.Now().UTC().Format(time.RFC3339)
	record := desktopstore.ManagedAgentRecord{
		PubKey:                    agentKeys.PublicHex(),
		Name:                      input.Name,
		PersonaID:                 nil,
		TeamID:                    nil,
		PrivateKeyNsec:            privateKeyNsec,
		AuthTag:                   stringPtr(authTagJSON),
		RelayURL:                  relayURL,
		AvatarURL:                 stringPtrIfNotEmpty(input.Avatar),
		ACPCommand:                "buzz-acp",
		AgentCommand:              input.Harness,
		AgentCommandOverride:      nil,
		AgentArgs:                 []string{},
		MCPCommand:                "",
		TurnTimeoutSeconds:        320,
		IdleTimeoutSeconds:        nil,
		MaxTurnDurationSeconds:    nil,
		Parallelism:               10,
		SystemPrompt:              stringPtr(input.SystemPrompt),
		Model:                     stringPtrIfNotEmpty(input.Model),
		Provider:                  nil,
		PersonaSourceVersion:      nil,
		EnvVars:                   nil,
		StartOnAppLaunch:          true,
		AutoRestartOnConfigChange: true,
		RuntimePID:                nil,
		Backend:                   desktopstore.LocalBackend(),
		BackendAgentID:            nil,
		ProviderBinaryPath:        nil,
		PersonaTeamDir:            nil,
		PersonaNameInTeam:         nil,
		CreatedAt:                 now,
		UpdatedAt:                 now,
		LastStartedAt:             nil,
		LastStoppedAt:             nil,
		LastExitCode:              nil,
		LastError:                 nil,
		LastErrorCode:             nil,
		RespondTo:                 types.RespondToOwnerOnly,
		RespondToAllowlist:        []string{},
		DisplayName:               nil,
		Slug:                      nil,
		Runtime:                   nil,
		NamePool:                  nil,
		IsBuiltin:                 false,
		IsActive:                  true,
		SourceTeam:                nil,
		SourceTeamPersonaSlug:     nil,
		CatalogSource:             nil,
		DefinitionRespondTo:       nil,
		DefinitionParallelism:     nil,
		RelayMesh:                 nil,
	}
	if err := desktopstore.WithFileLock(storePath+".lock", 5*time.Second, func() error {
		raw, err := desktopstore.LoadRaw(storePath)
		if err != nil {
			return err
		}
		recordJSON, err := json.Marshal(record)
		if err != nil {
			return err
		}
		raw = append(raw, json.RawMessage(recordJSON))
		return desktopstore.SaveRaw(storePath, raw)
	}); err != nil {
		return otherWrap("save desktop store", err)
	}

	result := desktopCreateResult{
		Name:                   input.Name,
		PubKey:                 agentKeys.PublicHex(),
		Nsec:                   nsec,
		AuthTag:                authTagJSON,
		StorePath:              storePath,
		KeychainService:        keychainService,
		KeychainStored:         !keychainFallbackInline,
		KeychainFallbackInline: keychainFallbackInline,
		EventIDs:               eventIDs,
		RelayErrors:            relayErrors,
	}
	if err := opts.writeJSON(result); err != nil {
		return err
	}
	if len(relayErrors) > 0 {
		// The local store + keychain already succeeded above - a slow or
		// unreachable relay must never fail the whole command, since the
		// desktop app can re-publish these events later. Warn on stderr
		// (stdout stays pure JSON) and exit 0.
		fmt.Fprintf(opts.stderr(), "warning: buzz desktop create: %d relay publish(es) failed or timed out (recorded in relay_errors); local store+keychain were still written: %s\n", len(relayErrors), strings.Join(relayErrors, "; "))
	}
	return nil
}

// desktopDelete removes every stored record matching input.Target (by exact
// name, or by 64-hex pubkey) in one local-store pass, then does
// keychain+relay cleanup per matched record. A record with no pubkey (a
// template/local-only agent that was never published) never touches the
// relay at all - there is nothing there to delete and no relay/auth
// identity to build a delete event from, so attempting it used to hang on a
// dead connect. Records with a pubkey get a REST relay-delete bounded by
// desktopRelayCallTimeout so a slow/unreachable relay can never hang the
// command; the local store+keychain removal above already happened by the
// time any relay call is attempted.
func (opts *rootOptions) desktopDelete(ctx context.Context, input desktopDeleteInput) error {
	overallTimeout := input.Timeout
	if overallTimeout <= 0 {
		overallTimeout = defaultDesktopTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, overallTimeout)
	defer cancel()

	storePath, err := resolveDesktopStorePath(input.StorePath)
	if err != nil {
		return otherWrap("resolve desktop store path", err)
	}
	keychainService := resolveDesktopKeychainService(input.KeychainService)
	var matches []desktopDeleteProbe
	var owner *nostr.KeyPair
	var fallbackRelayURL string
	if err := desktopstore.WithFileLock(storePath+".lock", 5*time.Second, func() error {
		raw, err := desktopstore.LoadRaw(storePath)
		if err != nil {
			return err
		}
		target := strings.TrimSpace(input.Target)
		targetLower := strings.ToLower(target)
		keep := make([]json.RawMessage, 0, len(raw))
		for _, record := range raw {
			var probe desktopDeleteProbe
			if err := json.Unmarshal(record, &probe); err != nil {
				keep = append(keep, record)
				continue
			}
			probe.PubKey = strings.ToLower(strings.TrimSpace(probe.PubKey))
			if probe.Name == target || (len(targetLower) == 64 && isHex64(targetLower) && probe.PubKey == targetLower) {
				matches = append(matches, probe)
				continue
			}
			keep = append(keep, record)
		}
		if len(matches) == 0 {
			return inputError("agent not found: " + input.Target)
		}
		if !input.Force {
			for _, m := range matches {
				if m.IsBuiltin {
					return inputError("refusing to delete a built-in agent without --force")
				}
			}
		}
		resolved, ownerKeys, err := opts.resolveOwnerKey(true)
		if err != nil {
			return err
		}
		owner = ownerKeys
		fallbackRelayURL = resolved.RelayURL
		return desktopstore.SaveRaw(storePath, keep)
	}); err != nil {
		return err
	}

	secretStore := newDesktopSecretStore(keychainService)
	results := make([]desktopDeleteResult, 0, len(matches))
	relayFailed := false
	for _, matched := range matches {
		result := opts.desktopDeleteOne(ctx, secretStore, owner, fallbackRelayURL, input.Community, matched)
		if len(result.RelayErrors) > 0 {
			relayFailed = true
		}
		results = append(results, result)
	}

	if err := opts.writeJSON(results); err != nil {
		return err
	}
	if relayFailed {
		return ExitError{Code: ExitRelay, Message: "one or more relay publishes failed", Err: errors.New("relay publish failed for one or more deleted agents")}
	}
	return nil
}

// desktopDeleteOne does the keychain + relay cleanup for a single matched
// record; the local store removal already happened before this is called.
func (opts *rootOptions) desktopDeleteOne(ctx context.Context, secretStore desktopstore.SecretStore, owner *nostr.KeyPair, fallbackRelayURL, community string, matched desktopDeleteProbe) desktopDeleteResult {
	result := desktopDeleteResult{
		Name:             matched.Name,
		PubKey:           matched.PubKey,
		RemovedFromStore: true,
		RelayErrors:      []string{},
	}

	if strings.TrimSpace(matched.PubKey) == "" {
		// Template/local-only record: never had a nostr identity, so there
		// is no keychain secret and no relay-side agent to delete.
		result.RelaySkippedReason = stringPtr("record has no pubkey (template/local-only agent); relay delete skipped")
		return result
	}

	keychainErr := secretStore.Delete("agent:" + strings.ToLower(matched.PubKey))
	result.KeychainDeleted = keychainErr == nil
	if keychainErr != nil {
		result.KeychainError = stringPtr(keychainErr.Error())
	}

	relayURL := strings.TrimSpace(matched.RelayURL)
	if relayURL == "" {
		relayURL = fallbackRelayURL
		if strings.TrimSpace(community) != "" {
			relayURL = strings.TrimSpace(community)
		}
	}
	if relayURL == "" {
		result.RelaySkippedReason = stringPtr("no relay URL resolvable")
		return result
	}

	authTag, err := nostr.MintAuthTag(owner, matched.PubKey, "")
	if err != nil {
		authTag = nil
	}
	deletion := nostr.NewUnsignedEvent(
		nostr.KindDeletion,
		owner.PublicHex(),
		"",
		nostr.Tags{{"a", fmt.Sprintf("%d:%s:%s", nostr.KindManagedAgent, owner.PublicHex(), matched.PubKey)}},
		0,
	)
	if err := deletion.Sign(owner); err != nil {
		result.RelayErrors = append(result.RelayErrors, "sign delete event: "+err.Error())
		return result
	}
	var archive nostr.Event
	built, err := types.BuildArchiveIdentityRequest(owner.PublicHex(), matched.PubKey, authTag, time.Now().Unix())
	if err != nil {
		result.RelayErrors = append(result.RelayErrors, err.Error())
	} else if err := built.Sign(owner); err != nil {
		result.RelayErrors = append(result.RelayErrors, "sign archive event: "+err.Error())
	} else {
		archive = built
	}

	// REST publish, bounded per-call - never a blocking WS session, never
	// left to hang on a relay that is slow or never answers.
	ownerClient := client.New(relayURL, owner, nil)
	deleteCtx, deleteCancel := context.WithTimeout(ctx, desktopRelayCallTimeout)
	_, err = ownerClient.PostEvent(deleteCtx, deletion)
	deleteCancel()
	if err != nil {
		result.RelayErrors = append(result.RelayErrors, err.Error())
	} else {
		result.RelayDeleteEventID = stringPtr(deletion.ID)
	}
	if archive.ID != "" {
		archiveCtx, archiveCancel := context.WithTimeout(ctx, desktopRelayCallTimeout)
		_, err = ownerClient.PostEvent(archiveCtx, archive)
		archiveCancel()
		if err != nil {
			result.RelayErrors = append(result.RelayErrors, err.Error())
		} else {
			result.RelayArchiveEventID = stringPtr(archive.ID)
		}
	}
	return result
}

func stringPtr(value string) *string {
	return &value
}

func stringPtrIfNotEmpty(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
