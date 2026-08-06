package desktopstore

import "encoding/json"

type localBackend struct {
	Type string `json:"type"`
}

// LocalBackend returns the local backend marker used by Buzz Desktop records.
func LocalBackend() localBackend {
	return localBackend{Type: "local"}
}

// ManagedAgentRecord mirrors managed_agents/types.rs::ManagedAgentRecord
// (desktop/src-tauri/src, reference repo, read-only). Fields with NO
// `omitempty` correspond to Rust Option<T> fields WITHOUT #[serde(default)]:
// the JSON key must always be present (null is fine for the value) or the
// real desktop app fails to parse the file at boot. Slices tagged this way
// (AgentArgs, RespondToAllowlist) must never be left nil - always assign
// []string{} when empty; encoding/json marshals a nil slice as `null`.
type ManagedAgentRecord struct {
	PubKey                       string            `json:"pubkey"`
	Name                         string            `json:"name"`
	PersonaID                    *string           `json:"persona_id"`
	TeamID                       *string           `json:"team_id,omitempty"`
	PrivateKeyNsec               string            `json:"private_key_nsec,omitempty"`
	AuthTag                      *string           `json:"auth_tag"`
	RelayURL                     string            `json:"relay_url"`
	AvatarURL                    *string           `json:"avatar_url"`
	ACPCommand                   string            `json:"acp_command"`
	AgentCommand                 string            `json:"agent_command"`
	AgentCommandOverride         *string           `json:"agent_command_override"`
	AgentArgs                    []string          `json:"agent_args"`
	MCPCommand                   string            `json:"mcp_command"`
	TurnTimeoutSeconds           uint64            `json:"turn_timeout_seconds"`
	IdleTimeoutSeconds           *uint64           `json:"idle_timeout_seconds"`
	MaxTurnDurationSeconds       *uint64           `json:"max_turn_duration_seconds"`
	Parallelism                  uint32            `json:"parallelism"`
	SystemPrompt                 *string           `json:"system_prompt"`
	Model                        *string           `json:"model"`
	Provider                     *string           `json:"provider"`
	PersonaSourceVersion         *string           `json:"persona_source_version"`
	EnvVars                      map[string]string `json:"env_vars,omitempty"`
	StartOnAppLaunch             bool              `json:"start_on_app_launch"`
	AutoRestartOnConfigChange    bool              `json:"auto_restart_on_config_change"`
	RuntimePID                   *uint32           `json:"runtime_pid"`
	Backend                      localBackend      `json:"backend"`
	BackendAgentID               *string           `json:"backend_agent_id"`
	ProviderBinaryPath           *string           `json:"provider_binary_path"`
	PersonaTeamDir               *string           `json:"persona_team_dir,omitempty"`
	PersonaNameInTeam            *string           `json:"persona_name_in_team,omitempty"`
	CreatedAt                    string            `json:"created_at"`
	UpdatedAt                    string            `json:"updated_at"`
	LastStartedAt                *string           `json:"last_started_at"`
	LastStoppedAt                *string           `json:"last_stopped_at"`
	LastExitCode                 *int32            `json:"last_exit_code"`
	LastError                    *string           `json:"last_error"`
	LastErrorCode                *int64            `json:"last_error_code"`
	RespondTo                    string            `json:"respond_to"`
	RespondToAllowlist           []string          `json:"respond_to_allowlist"`
	DisplayName                  *string           `json:"display_name,omitempty"`
	Slug                         *string           `json:"slug,omitempty"`
	Runtime                      *string           `json:"runtime,omitempty"`
	NamePool                     []string          `json:"name_pool,omitempty"`
	IsBuiltin                    bool              `json:"is_builtin"`
	IsActive                     bool              `json:"is_active"`
	SourceTeam                   *string           `json:"source_team,omitempty"`
	SourceTeamPersonaSlug        *string           `json:"source_team_persona_slug,omitempty"`
	CatalogSource                json.RawMessage   `json:"catalog_source,omitempty"`
	DefinitionRespondTo          *string           `json:"definition_respond_to,omitempty"`
	DefinitionRespondToAllowlist []string          `json:"definition_respond_to_allowlist,omitempty"`
	DefinitionParallelism        *uint32           `json:"definition_parallelism,omitempty"`
	RelayMesh                    json.RawMessage   `json:"relay_mesh,omitempty"`
}
