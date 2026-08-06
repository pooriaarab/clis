package mcp

import (
	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// flagKind is the JSON-schema shape a CLI flag maps to on the MCP side.
type flagKind int

const (
	flagString flagKind = iota
	flagBool
	flagInt
	flagStringSlice
)

// flagDef mirrors one buzz-cli flag. Name is used verbatim as both the MCP
// tool's input-schema property name and the `--<name>` CLI flag, so tool
// inputs read exactly like the CLI's own flags.
type flagDef struct {
	Name        string
	Kind        flagKind
	Required    bool
	Description string
}

// toolDef is one MCP tool: a buzz-cli subcommand path, its own flags, and
// (for the handful of commands that take one) its single positional arg.
type toolDef struct {
	Name               string
	Path               []string
	Description        string
	Flags              []flagDef
	Positional         string
	PositionalRequired bool
	PositionalDesc     string
	ReadOnly           bool
}

// globalFlags are buzz-cli's persistent root flags. They're appended to
// every tool so a caller can target a specific relay/identity/key per call
// instead of relying only on the MCP server process's ambient env/config
// file — the same precedence buzz-cli itself uses (flags > env > file).
var globalFlags = []flagDef{
	{Name: "relay", Kind: flagString, Description: "Relay URL; overrides BUZZ_RELAY_URL / config file"},
	{Name: "identity", Kind: flagString, Description: "Named identity from the config file"},
	{Name: "key", Kind: flagString, Description: "Private key as nsec or 64-char hex; overrides BUZZ_PRIVATE_KEY"},
	{Name: "auth-tag", Kind: flagString, Description: "NIP-OA auth tag JSON; overrides BUZZ_AUTH_TAG"},
	{Name: "owner-key", Kind: flagString, Description: "Owner private key as nsec or 64-char hex; overrides BUZZ_OWNER_KEY"},
	{Name: "config", Kind: flagString, Description: "Config file path; overrides BUZZ_CONFIG"},
}

// toolDefs is the MCP tool surface: buzz-cli's high-value operations,
// mirroring the CLI's own subcommands and flags one-for-one. See
// buzz-cli/SPEC.md for the full command tree; this is a subset — the
// operations an agent needs most — not an auto-generated mirror of every
// subcommand.
var toolDefs = []toolDef{
	// messages
	{
		Name: "messages_send", Path: []string{"messages", "send"},
		Description: "Send a channel message.",
		Flags: []flagDef{
			{Name: "channel", Kind: flagString, Required: true, Description: "Channel id"},
			{Name: "content", Kind: flagString, Description: "Message content"},
			{Name: "file", Kind: flagString, Description: "Read message content from this file instead of --content"},
			{Name: "kind", Kind: flagInt, Description: "Event kind (default: channel message)"},
			{Name: "reply-to", Kind: flagString, Description: "Parent event id"},
			{Name: "mention", Kind: flagStringSlice, Description: "Pubkey(s) to mention"},
			{Name: "broadcast", Kind: flagBool, Description: "Broadcast message"},
		},
	},
	{
		Name: "messages_get", Path: []string{"messages", "get"}, ReadOnly: true,
		Description: "Get a message by event id.",
		Flags:       []flagDef{{Name: "id", Kind: flagString, Required: true, Description: "Event id"}},
	},
	{
		Name: "messages_thread", Path: []string{"messages", "thread"}, ReadOnly: true,
		Description: "Get a message thread by root event id.",
		Flags:       []flagDef{{Name: "root", Kind: flagString, Required: true, Description: "Root event id"}},
	},

	// channels
	{
		Name: "channels_list", Path: []string{"channels", "list"}, ReadOnly: true,
		Description: "List channels.",
		Flags: []flagDef{
			{Name: "visibility", Kind: flagString, Description: "Visibility filter"},
			{Name: "member", Kind: flagString, Description: "Member pubkey filter"},
			{Name: "limit", Kind: flagInt, Description: "Max results"},
		},
	},
	{
		Name: "channels_get", Path: []string{"channels", "get"}, ReadOnly: true,
		Description: "Get a channel by id.",
		Flags:       []flagDef{{Name: "channel", Kind: flagString, Required: true, Description: "Channel id"}},
	},
	{
		Name: "channels_create", Path: []string{"channels", "create"},
		Description: "Create a channel.",
		Flags: []flagDef{
			{Name: "name", Kind: flagString, Required: true, Description: "Channel name"},
			{Name: "type", Kind: flagString, Description: "Channel type"},
			{Name: "visibility", Kind: flagString, Description: "Channel visibility"},
			{Name: "description", Kind: flagString, Description: "Channel description"},
			{Name: "ttl", Kind: flagInt, Description: "TTL seconds"},
		},
	},
	{
		Name: "channels_add_member", Path: []string{"channels", "add-member"},
		Description: "Add a member to a channel.",
		Flags: []flagDef{
			{Name: "channel", Kind: flagString, Required: true, Description: "Channel id"},
			{Name: "pubkey", Kind: flagString, Required: true, Description: "Member pubkey"},
			{Name: "role", Kind: flagString, Description: "Member role"},
		},
	},

	// agents & fleet
	{
		Name: "agents_create", Path: []string{"agents", "create"},
		Description: "Create a managed agent (keygen + profile + persona + managed-agent projection + auth tag + channel memberships).",
		Flags:       agentCreateFlags(),
	},
	{
		Name: "agents_list", Path: []string{"agents", "list"}, ReadOnly: true,
		Description: "List managed agents.",
	},
	{
		Name: "agents_get", Path: []string{"agents", "get"}, ReadOnly: true,
		Description: "Get a managed agent.", Positional: "target", PositionalRequired: true,
		PositionalDesc: "Agent name (from the config file) or hex pubkey",
	},
	{
		Name: "agents_run", Path: []string{"agents", "run"},
		Description: "Run an agent's buzz-acp runtime. Blocks unless --detach.",
		Positional:  "target", PositionalRequired: true,
		PositionalDesc: "Agent name (from the config file) or hex pubkey",
		Flags: []flagDef{
			{Name: "detach", Kind: flagBool, Description: "Run in the background and return immediately"},
			{Name: "pidfile", Kind: flagString, Description: "Pidfile path (used with --detach)"},
			{Name: "acp-command", Kind: flagString, Description: "ACP runner command (default: buzz-acp)"},
			{Name: "harness-command", Kind: flagString, Description: "Agent harness command"},
		},
	},
	{
		Name: "agents_stop", Path: []string{"agents", "stop"},
		Description: "Stop a detached agent runtime.",
		Flags: []flagDef{
			{Name: "pid", Kind: flagInt, Description: "Process id"},
			{Name: "pidfile", Kind: flagString, Description: "Pidfile path"},
		},
	},
	{
		Name: "agents_delete", Path: []string{"agents", "delete"},
		Description: "Delete a managed agent projection.", Positional: "target", PositionalRequired: true,
		PositionalDesc: "Agent name (from the config file) or hex pubkey",
	},
	{
		Name: "fleet_create", Path: []string{"fleet"},
		Description: "Create N managed agents sharing one runtime, optionally launching and supervising all of them (--run).",
		Flags: append([]flagDef{
			{Name: "count", Kind: flagInt, Required: true, Description: "Number of agents to create"},
			{Name: "name-prefix", Kind: flagString, Required: true, Description: "Agent name prefix (agents are named <prefix>-1, <prefix>-2, ...)"},
			{Name: "run", Kind: flagBool, Description: "Launch and supervise all created agents after creation"},
			{Name: "max-concurrent", Kind: flagInt, Description: "Cap on simultaneously-running runtimes (default: count; excess agents queue)"},
			{Name: "acp-command", Kind: flagString, Description: "ACP runner command (default: buzz-acp)"},
			{Name: "harness-command", Kind: flagString, Description: "Agent harness command"},
			{Name: "log-dir", Kind: flagString, Description: "Directory for per-agent runtime logs (default: a generated temp directory)"},
		}, agentCreateFlags()...),
	},

	// users
	{
		Name: "users_get", Path: []string{"users", "get"}, ReadOnly: true,
		Description: "Get user profiles.",
		Flags: []flagDef{
			{Name: "pubkey", Kind: flagString, Description: "User pubkey"},
			{Name: "name", Kind: flagString, Description: "Profile search text"},
			{Name: "owner", Kind: flagString, Description: "Owner pubkey"},
		},
	},
	{
		Name: "users_set_profile", Path: []string{"users", "set-profile"},
		Description: "Publish a kind:0 profile.",
		Flags: []flagDef{
			{Name: "name", Kind: flagString, Description: "Profile name"},
			{Name: "display-name", Kind: flagString, Description: "Profile display name"},
			{Name: "avatar", Kind: flagString, Description: "Avatar URL"},
			{Name: "about", Kind: flagString, Description: "Profile about text"},
			{Name: "nip05", Kind: flagString, Description: "NIP-05 identifier"},
		},
	},

	// reactions
	{
		Name: "reactions_add", Path: []string{"reactions", "add"},
		Description: "Add a reaction to an event.",
		Flags: []flagDef{
			{Name: "event", Kind: flagString, Required: true, Description: "Event id"},
			{Name: "emoji", Kind: flagString, Required: true, Description: "Emoji or shortcode"},
			{Name: "emoji-url", Kind: flagString, Description: "Custom emoji URL"},
		},
	},
	{
		Name: "reactions_remove", Path: []string{"reactions", "remove"},
		Description: "Remove your reaction from an event.",
		Flags: []flagDef{
			{Name: "event", Kind: flagString, Required: true, Description: "Event id"},
			{Name: "emoji", Kind: flagString, Required: true, Description: "Emoji"},
		},
	},
	{
		Name: "reactions_get", Path: []string{"reactions", "get"}, ReadOnly: true,
		Description: "Get reactions on an event, grouped by emoji.",
		Flags:       []flagDef{{Name: "event", Kind: flagString, Required: true, Description: "Event id"}},
	},

	// dms
	{
		Name: "dms_list", Path: []string{"dms", "list"}, ReadOnly: true,
		Description: "List direct messages.",
		Flags:       []flagDef{{Name: "limit", Kind: flagInt, Description: "Max results"}},
	},
	{
		Name: "dms_open", Path: []string{"dms", "open"},
		Description: "Open a direct message with 1-8 participants.",
		Flags:       []flagDef{{Name: "pubkey", Kind: flagStringSlice, Required: true, Description: "Participant pubkey(s), 1 to 8"}},
	},
	{
		Name: "dms_add_member", Path: []string{"dms", "add-member"},
		Description: "Add a member to a direct message.",
		Flags: []flagDef{
			{Name: "channel", Kind: flagString, Required: true, Description: "DM channel id"},
			{Name: "pubkey", Kind: flagString, Required: true, Description: "Member pubkey"},
		},
	},
	{
		Name: "dms_hide", Path: []string{"dms", "hide"},
		Description: "Hide a direct message.",
		Flags:       []flagDef{{Name: "channel", Kind: flagString, Required: true, Description: "DM channel id"}},
	},

	// feed
	{
		Name: "feed_get", Path: []string{"feed", "get"}, ReadOnly: true,
		Description: "Get your feed.",
		Flags: []flagDef{
			{Name: "since", Kind: flagInt, Description: "Minimum created_at timestamp"},
			{Name: "limit", Kind: flagInt, Description: "Max results"},
			{Name: "types", Kind: flagString, Description: "Feed types CSV: mentions, needs_action, activity, agent_activity"},
		},
	},

	// notes
	{
		Name: "notes_set", Path: []string{"notes", "set"},
		Description: "Publish a long-form note (NIP-23).",
		Flags: []flagDef{
			{Name: "name", Kind: flagString, Required: true, Description: "Note slug"},
			{Name: "title", Kind: flagString, Description: "Note title (required on first publish)"},
			{Name: "summary", Kind: flagString, Description: "Note summary"},
			{Name: "tag", Kind: flagStringSlice, Description: "Topic tag(s)"},
			{Name: "clear-tags", Kind: flagBool, Description: "Clear topic tags (mutually exclusive with --tag)"},
			{Name: "content", Kind: flagString, Required: true, Description: "Note content, or '-' to read stdin (not supported over MCP; pass content directly)"},
			{Name: "allow-empty", Kind: flagBool, Description: "Allow empty content"},
		},
	},
	{
		Name: "notes_get", Path: []string{"notes", "get"}, ReadOnly: true,
		Description: "Get a long-form note by naddr or name.",
		Flags: []flagDef{
			{Name: "naddr", Kind: flagString, Description: "naddr or kind:pubkey:identifier coordinate"},
			{Name: "name", Kind: flagString, Description: "Note slug"},
			{Name: "author", Kind: flagString, Description: "Author pubkey, 'me', or display name (only with --name)"},
			{Name: "latest", Kind: flagBool, Description: "Pick the latest matching note (only with --name)"},
			{Name: "content-only", Kind: flagBool, Description: "Return only the note content"},
		},
	},
	{
		Name: "notes_ls", Path: []string{"notes", "ls"}, ReadOnly: true,
		Description: "List long-form notes.",
		Flags: []flagDef{
			{Name: "author", Kind: flagString, Description: "Author pubkey, 'me', display name, or 'all' (default: me)"},
			{Name: "tag", Kind: flagString, Description: "Topic tag filter"},
			{Name: "limit", Kind: flagInt, Description: "Max results"},
		},
	},
	{
		Name: "notes_rm", Path: []string{"notes", "rm"},
		Description: "Delete one of your long-form notes.",
		Flags:       []flagDef{{Name: "name", Kind: flagString, Required: true, Description: "Note slug"}},
	},

	// invite
	{
		Name: "invite_create", Path: []string{"invite", "create"},
		Description: "Create a relay invite.",
		Flags: []flagDef{
			{Name: "ttl-secs", Kind: flagInt, Description: "Invite TTL in seconds"},
			{Name: "max-uses", Kind: flagInt, Description: "Maximum invite uses"},
		},
	},
	{
		Name: "invite_claim", Path: []string{"invite", "claim"},
		Description: "Claim a relay invite.",
		Flags: []flagDef{
			{Name: "code", Kind: flagString, Required: true, Description: "Invite code"},
			{Name: "policy-receipt", Kind: flagString, Description: "Policy receipt"},
		},
	},
	{
		Name: "invite_list", Path: []string{"invite", "list"}, ReadOnly: true,
		Description: "List locally-created invites (from the config file).",
	},

	// workflows
	{
		Name: "workflows_list", Path: []string{"workflows", "list"}, ReadOnly: true,
		Description: "List workflows in a channel.",
		Flags:       []flagDef{{Name: "channel", Kind: flagString, Required: true, Description: "Channel UUID"}},
	},
	{
		Name: "workflows_get", Path: []string{"workflows", "get"}, ReadOnly: true,
		Description: "Get a workflow.",
		Flags:       []flagDef{{Name: "workflow", Kind: flagString, Required: true, Description: "Workflow UUID"}},
	},
	{
		Name: "workflows_create", Path: []string{"workflows", "create"},
		Description: "Create a workflow from a YAML definition.",
		Flags: []flagDef{
			{Name: "channel", Kind: flagString, Required: true, Description: "Channel UUID"},
			{Name: "yaml", Kind: flagString, Required: true, Description: "Workflow YAML definition"},
		},
	},
	{
		Name: "workflows_update", Path: []string{"workflows", "update"},
		Description: "Update a workflow's YAML definition.",
		Flags: []flagDef{
			{Name: "channel", Kind: flagString, Required: true, Description: "Channel UUID the workflow belongs to"},
			{Name: "workflow", Kind: flagString, Required: true, Description: "Workflow UUID"},
			{Name: "yaml", Kind: flagString, Required: true, Description: "Updated workflow YAML definition"},
		},
	},
	{
		Name: "workflows_trigger", Path: []string{"workflows", "trigger"},
		Description: "Trigger a workflow run.",
		Flags: []flagDef{
			{Name: "workflow", Kind: flagString, Required: true, Description: "Workflow UUID"},
			{Name: "inputs", Kind: flagString, Description: "JSON object of input variables"},
		},
	},
	{
		Name: "workflows_runs", Path: []string{"workflows", "runs"}, ReadOnly: true,
		Description: "List runs for a workflow.",
		Flags: []flagDef{
			{Name: "workflow", Kind: flagString, Required: true, Description: "Workflow UUID"},
			{Name: "limit", Kind: flagInt, Description: "Max results"},
		},
	},
	{
		Name: "workflows_approve", Path: []string{"workflows", "approve"},
		Description: "Approve or deny a workflow step.",
		Flags: []flagDef{
			{Name: "token", Kind: flagString, Required: true, Description: "Approval token UUID"},
			{Name: "approved", Kind: flagBool, Description: "Approve (true) or deny (false); default true"},
			{Name: "note", Kind: flagString, Description: "Optional note"},
		},
	},

	// moderation (reads only — mutations are owner/admin-gated relay-side
	// and intentionally not exposed as MCP tools in this increment)
	{
		Name: "moderation_reports", Path: []string{"moderation", "reports"}, ReadOnly: true,
		Description: "List reports in the moderation queue (newest first).",
		Flags: []flagDef{
			{Name: "status", Kind: flagString, Description: "Filter by status: open | resolved | dismissed | escalated"},
			{Name: "limit", Kind: flagInt, Description: "Max results"},
		},
	},
	{
		Name: "moderation_restricted", Path: []string{"moderation", "restricted"}, ReadOnly: true,
		Description: "List currently-restricted members (active ban or timeout).",
	},
	{
		Name: "moderation_audit", Path: []string{"moderation", "audit"}, ReadOnly: true,
		Description: "Read the moderation audit trail (newest first).",
		Flags:       []flagDef{{Name: "limit", Kind: flagInt, Description: "Max rows"}},
	},
}

// agentCreateFlags mirrors internal/cli.addAgentCreateFlags (name is
// excluded: fleet derives it from --name-prefix, and agents_create takes it
// as its own required flag below).
func agentCreateFlags() []flagDef {
	return []flagDef{
		{Name: "name", Kind: flagString, Description: "Agent name (required for agents_create; ignored for fleet_create, which derives names from --name-prefix)"},
		{Name: "system-prompt", Kind: flagString, Description: "System prompt"},
		{Name: "system-prompt-file", Kind: flagString, Description: "System prompt file path"},
		{Name: "avatar", Kind: flagString, Description: "Avatar URL"},
		{Name: "runtime", Kind: flagString, Description: "Runtime command"},
		{Name: "runtime-args", Kind: flagString, Description: "Runtime args"},
		{Name: "model", Kind: flagString, Description: "Model"},
		{Name: "provider", Kind: flagString, Description: "Provider"},
		{Name: "channels", Kind: flagStringSlice, Description: "Channel id(s) to add the agent to"},
		{Name: "respond-to", Kind: flagString, Description: "respond-to mode: owner-only, allowlist, or anyone"},
		{Name: "persona-dir", Kind: flagString, Description: "Persona directory (fleet_create only: per-agent system-prompt files named <name>.txt)"},
	}
}

// toolOptions builds the mcp-go tool options (description, positional arg,
// own flags, then the shared global flags) for one toolDef.
func toolOptions(def toolDef) []mcplib.ToolOption {
	opts := []mcplib.ToolOption{mcplib.WithDescription(def.Description)}
	if def.ReadOnly {
		opts = append(opts, mcplib.WithReadOnlyHintAnnotation(true), mcplib.WithDestructiveHintAnnotation(false))
	}
	if def.Positional != "" {
		opts = append(opts, flagOption(flagDef{
			Name:        def.Positional,
			Kind:        flagString,
			Required:    def.PositionalRequired,
			Description: def.PositionalDesc,
		}))
	}
	for _, f := range def.Flags {
		opts = append(opts, flagOption(f))
	}
	for _, g := range globalFlags {
		opts = append(opts, flagOption(g))
	}
	return opts
}

func flagOption(f flagDef) mcplib.ToolOption {
	var props []mcplib.PropertyOption
	if f.Description != "" {
		props = append(props, mcplib.Description(f.Description))
	}
	if f.Required {
		props = append(props, mcplib.Required())
	}
	switch f.Kind {
	case flagBool:
		return mcplib.WithBoolean(f.Name, props...)
	case flagInt:
		return mcplib.WithNumber(f.Name, props...)
	case flagStringSlice:
		props = append(props, mcplib.WithStringItems())
		return mcplib.WithArray(f.Name, props...)
	default:
		return mcplib.WithString(f.Name, props...)
	}
}

// RegisterTools registers every buzz-cli operation in toolDefs as an MCP
// tool backed by runCLI, so a fresh RegisterTools call is the entire MCP
// surface — see cmd/buzz-mcp/main.go.
func RegisterTools(s *server.MCPServer) {
	for _, def := range toolDefs {
		s.AddTool(mcplib.NewTool(def.Name, toolOptions(def)...), makeHandler(def))
	}
}
