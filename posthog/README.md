# Posthog CLI



Created by [@pooriaarab](https://github.com/pooriaarab) (Pooria Arab).

## Install

The recommended path installs both the `posthog-pp-cli` binary and the `pp-posthog` agent skill (Claude Code, Codex, Cursor, Gemini CLI, GitHub Copilot, and other agents supported by the upstream [`skills`](https://github.com/vercel-labs/skills) CLI) in one shot:

```bash
npx -y @mvanhorn/printing-press-library install posthog
```

For CLI only (no skill):

```bash
npx -y @mvanhorn/printing-press-library install posthog --cli-only
```

For skill only — installs the skill into the same agents as the default command above, but skips the CLI binary (use this to update or reinstall just the skill):

```bash
npx -y @mvanhorn/printing-press-library install posthog --skill-only
```

To constrain the skill install to one or more specific agents (repeatable — agent names match the [`skills`](https://github.com/vercel-labs/skills) CLI):

```bash
npx -y @mvanhorn/printing-press-library install posthog --agent claude-code
npx -y @mvanhorn/printing-press-library install posthog --agent claude-code --agent codex
```

### Without Node (Go fallback)

If `npx` isn't available (no Node, offline), install the CLI directly via Go (requires Go 1.26.5 or newer):

```bash
go install github.com/mvanhorn/printing-press-library/library/monitoring/posthog/cmd/posthog-pp-cli@latest
```

This installs the CLI only — no skill.

### Pre-built binary

Download a pre-built binary for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/posthog-current). On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine <binary>`. On Unix, mark it executable: `chmod +x <binary>`.

<!-- pp-hermes-install-anchor -->
## Install for Hermes

Install the CLI binary first. The installer writes binaries to a per-user managed bin directory by default: `$HOME/.local/bin` on macOS/Linux and `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows.

```bash
npx -y @mvanhorn/printing-press-library install posthog --cli-only
```

Then install the focused Hermes skill.

From the Hermes CLI:

```bash
hermes skills install mvanhorn/printing-press-library/cli-skills/pp-posthog --force
```

Inside a Hermes chat session:

```bash
/skills install mvanhorn/printing-press-library/cli-skills/pp-posthog --force
```

Restart the Hermes session or gateway if the newly installed skill is not visible immediately.

## Install for OpenClaw
Install both the CLI binary and the focused OpenClaw skill. The installer defaults binaries to a per-user bin directory (`$HOME/.local/bin` on macOS/Linux, `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows):

```bash
npx -y @mvanhorn/printing-press-library install posthog --agent openclaw
```

Restart the OpenClaw session or gateway if the newly installed skill is not visible immediately.

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/posthog-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.
3. Fill in `POSTHOG_PERSONAL_APIKEY_AUTH` when Claude Desktop prompts you.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.


```bash
go install github.com/mvanhorn/printing-press-library/library/monitoring/posthog/cmd/posthog-pp-mcp@latest
```

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "posthog": {
      "command": "posthog-pp-mcp",
      "env": {
        "POSTHOG_PERSONAL_APIKEY_AUTH": "<your-key>"
      }
    }
  }
}
```

</details>

## Quick Start

### 1. Install

See [Install](#install) above.

### 2. Set Up Credentials

Get your access token from your API provider's developer portal, then store it:

```bash
posthog-pp-cli auth set-token YOUR_TOKEN_HERE
```

Or set it via environment variable:

```bash
export POSTHOG_PERSONAL_APIKEY_AUTH="your-token-here"
```

### 3. Verify Setup

```bash
posthog-pp-cli doctor
```

This checks your configuration and credentials.

### 4. Try Your First Command

```bash
posthog-pp-cli account-notes mock-value
```

## Usage

Run `posthog-pp-cli --help` for the full command reference and flag list.

## Paths & environment variables

This CLI separates local files into four path kinds:

| Kind | Contents |
|------|----------|
| `config` | User-editable settings such as `config.toml` and saved profiles |
| `data` | Durable local data: `credentials.toml`, `data.db`, cookies, browser-session proof files, and other auth sidecars |
| `state` | Runtime state such as persisted queries, jobs, and `teach.log` |
| `cache` | Regenerable HTTP/cache files |

Each kind resolves independently. The ladder is:

1. Per-kind env var: `POSTHOG_CONFIG_DIR`, `POSTHOG_DATA_DIR`, `POSTHOG_STATE_DIR`, or `POSTHOG_CACHE_DIR`
2. `--home <dir>` for this invocation
3. `POSTHOG_HOME` for a flat relocated root
4. XDG env vars: `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME`
5. Platform defaults matching existing installs

For containers and agent sandboxes, prefer a single relocated root:

```bash
export POSTHOG_HOME=/srv/posthog
posthog-pp-cli doctor
```

Under `POSTHOG_HOME=/srv/posthog`, the four dirs resolve to `/srv/posthog/config`, `/srv/posthog/data`, `/srv/posthog/state`, and `/srv/posthog/cache`.

MCP servers do not receive CLI flags from the host. Put relocation in the host `env` block:

```json
{
  "mcpServers": {
    "posthog": {
      "command": "posthog-pp-mcp",
      "env": {
        "POSTHOG_HOME": "/srv/posthog"
      }
    }
  }
}
```

Precedence matters in fleets: an ambient per-kind variable such as `POSTHOG_DATA_DIR` overrides an explicit `--home` for that kind. Use `POSTHOG_HOME` or the per-kind variables for durable fleet relocation; treat `--home` as the weaker per-invocation lever.

Relocation is one-way. Unsetting `POSTHOG_HOME` does not move files back to platform defaults, and `doctor` cannot find credentials left under a former root. Move the files manually before unsetting relocation variables.

Existing installs keep working because the platform-default rung matches the legacy layout. On the first auth write, stored secrets leave `config.toml` and are consolidated into `credentials.toml` under the data directory. Run `posthog-pp-cli doctor --fail-on warn` to check path and credential-location warnings in automation.

## Commands

### account-notes

Manage account notes

- **`posthog-pp-cli account-notes <project_id>`** - List

### account-relationship-definitions

Manage account relationship definitions

- **`posthog-pp-cli account-relationship-definitions create`** - Create
- **`posthog-pp-cli account-relationship-definitions destroy`** - Destroy
- **`posthog-pp-cli account-relationship-definitions list`** - List
- **`posthog-pp-cli account-relationship-definitions partial-update`** - Partial update
- **`posthog-pp-cli account-relationship-definitions retrieve`** - Retrieve
- **`posthog-pp-cli account-relationship-definitions update`** - Update

### accounts

Manage accounts

- **`posthog-pp-cli accounts create`** - Create
- **`posthog-pp-cli accounts destroy`** - Destroy
- **`posthog-pp-cli accounts list`** - List
- **`posthog-pp-cli accounts partial-update`** - Partial update
- **`posthog-pp-cli accounts retrieve`** - Retrieve
- **`posthog-pp-cli accounts update`** - Update

### actions

Manage actions

- **`posthog-pp-cli actions bulk-update-tags-create`** - Bulk update tags on multiple objects.

PAT access: this action has no ``required_scopes=`` on the decorator —
inheriting viewsets must add ``"bulk_update_tags"`` to their
``scope_object_write_actions`` list to accept personal API keys.
Without that opt-in, ``APIScopePermission`` rejects PAT requests with
"This action does not support personal API key access". Done per-viewset
so granting ``<scope>:write`` for one resource doesn't leak access to
sibling resources that share this mixin.

Accepts:
- {"ids": [...], "action": "add"|"remove"|"set", "tags": ["tag1", "tag2"]}

Actions:
- "add": Add tags to existing tags on each object
- "remove": Remove specific tags from each object
- "set": Replace all tags on each object with the provided list
- **`posthog-pp-cli actions create`** - Create
- **`posthog-pp-cli actions destroy`** - Hard delete of this model is not allowed. Use a patch API call to set "deleted" to true
- **`posthog-pp-cli actions list`** - List
- **`posthog-pp-cli actions partial-update`** - Partial update
- **`posthog-pp-cli actions retrieve`** - Retrieve
- **`posthog-pp-cli actions update`** - Update

### activity-log

Manage activity log

- **`posthog-pp-cli activity-log <project_id>`** - List

### advanced-activity-logs

Manage advanced activity logs

- **`posthog-pp-cli advanced-activity-logs available-filters-retrieve`** - Available filters retrieve
- **`posthog-pp-cli advanced-activity-logs export-create`** - Export create
- **`posthog-pp-cli advanced-activity-logs list`** - List

### alerts

Manage alerts

- **`posthog-pp-cli alerts create`** - Create
- **`posthog-pp-cli alerts destroy`** - Destroy
- **`posthog-pp-cli alerts list`** - List
- **`posthog-pp-cli alerts partial-update`** - Partial update
- **`posthog-pp-cli alerts retrieve`** - Retrieve
- **`posthog-pp-cli alerts simulate-create`** - Simulate a detector on an insight's historical data. Read-only — no AlertCheck records are created.
- **`posthog-pp-cli alerts update`** - Update

### annotations

Manage annotations

- **`posthog-pp-cli annotations create`** - Create, Read, Update and Delete annotations. [See docs](https://posthog.com/docs/data/annotations) for more information on annotations.
- **`posthog-pp-cli annotations destroy`** - Hard delete of this model is not allowed. Use a patch API call to set "deleted" to true
- **`posthog-pp-cli annotations list`** - Create, Read, Update and Delete annotations. [See docs](https://posthog.com/docs/data/annotations) for more information on annotations.
- **`posthog-pp-cli annotations partial-update`** - Create, Read, Update and Delete annotations. [See docs](https://posthog.com/docs/data/annotations) for more information on annotations.
- **`posthog-pp-cli annotations retrieve`** - Create, Read, Update and Delete annotations. [See docs](https://posthog.com/docs/data/annotations) for more information on annotations.
- **`posthog-pp-cli annotations update`** - Create, Read, Update and Delete annotations. [See docs](https://posthog.com/docs/data/annotations) for more information on annotations.

### announcements

Manage announcements

- **`posthog-pp-cli announcements channels-list`** - Slack channels the SupportHog bot can post to, labeled by customer account name.
- **`posthog-pp-cli announcements create`** - Create
- **`posthog-pp-cli announcements list`** - List
- **`posthog-pp-cli announcements retrieve`** - Retrieve

### approval-policies

Manage approval policies

- **`posthog-pp-cli approval-policies create`** - Create
- **`posthog-pp-cli approval-policies destroy`** - Destroy
- **`posthog-pp-cli approval-policies list`** - List
- **`posthog-pp-cli approval-policies partial-update`** - Partial update
- **`posthog-pp-cli approval-policies retrieve`** - Retrieve
- **`posthog-pp-cli approval-policies update`** - Update

### batch-exports

Manage batch exports

- **`posthog-pp-cli batch-exports create`** - Create
- **`posthog-pp-cli batch-exports destroy`** - Destroy
- **`posthog-pp-cli batch-exports list`** - List
- **`posthog-pp-cli batch-exports partial-update`** - Partial update
- **`posthog-pp-cli batch-exports retrieve`** - Retrieve
- **`posthog-pp-cli batch-exports run-test-step-new-create`** - Run test step new create
- **`posthog-pp-cli batch-exports test-retrieve`** - Test retrieve
- **`posthog-pp-cli batch-exports update`** - Update

### business-knowledge

Manage business knowledge

- **`posthog-pp-cli business-knowledge documents-search-list`** - Read-only access to parsed knowledge documents. Exposes hybrid search
(``search``) and a drill-down window (``window``) so an agent (PHAI or
MCP) can find and explore business knowledge chunks.
- **`posthog-pp-cli business-knowledge documents-window-list`** - Read-only access to parsed knowledge documents. Exposes hybrid search
(``search``) and a drill-down window (``window``) so an agent (PHAI or
MCP) can find and explore business knowledge chunks.
- **`posthog-pp-cli business-knowledge gap-suggestions-accept-create`** - Surfaces topics the support AI couldn't answer from the knowledge base.

Two list shapes controlled by the ``ticket_id`` query param:
- **per-ticket** (``?ticket_id=<uuid>``): individual gap rows for that ticket.
- **aggregated** (no ``ticket_id``): gaps grouped by normalized topic with counts,
  for the Business knowledge suggestions panel.
- **`posthog-pp-cli business-knowledge gap-suggestions-accept-topic-create`** - Accept all pending suggestions for a normalized topic cluster.
- **`posthog-pp-cli business-knowledge gap-suggestions-dismiss-create`** - Surfaces topics the support AI couldn't answer from the knowledge base.

Two list shapes controlled by the ``ticket_id`` query param:
- **per-ticket** (``?ticket_id=<uuid>``): individual gap rows for that ticket.
- **aggregated** (no ``ticket_id``): gaps grouped by normalized topic with counts,
  for the Business knowledge suggestions panel.
- **`posthog-pp-cli business-knowledge gap-suggestions-dismiss-topic-create`** - Dismiss all pending suggestions for a normalized topic cluster.
- **`posthog-pp-cli business-knowledge gap-suggestions-list`** - Surfaces topics the support AI couldn't answer from the knowledge base.

Two list shapes controlled by the ``ticket_id`` query param:
- **per-ticket** (``?ticket_id=<uuid>``): individual gap rows for that ticket.
- **aggregated** (no ``ticket_id``): gaps grouped by normalized topic with counts,
  for the Business knowledge suggestions panel.
- **`posthog-pp-cli business-knowledge sources-create`** - Sources create
- **`posthog-pp-cli business-knowledge sources-destroy`** - Sources destroy
- **`posthog-pp-cli business-knowledge sources-list`** - Sources list
- **`posthog-pp-cli business-knowledge sources-partial-update`** - Sources partial update
- **`posthog-pp-cli business-knowledge sources-refresh-create`** - Sources refresh create
- **`posthog-pp-cli business-knowledge sources-retrieve`** - Sources retrieve
- **`posthog-pp-cli business-knowledge sources-text-retrieve`** - Sources text retrieve

### calendar-sync

Manage calendar sync

- **`posthog-pp-cli calendar-sync list`** - Calendar-sync controls for Customer analytics settings. Sync runs on an hourly
Temporal schedule; this surface only offers the manual "sync now" escape hatch.
- **`posthog-pp-cli calendar-sync sync-now-create`** - Start a sync run for one connected Google Calendar immediately, outside the hourly schedule.

### canvases

Manage canvases

- **`posthog-pp-cli canvases create`** - Create a new, empty canvas in a channel; give it source by publishing a project.
- **`posthog-pp-cli canvases destroy`** - Canvases: agent-built sandboxed browser apps, filed into channels.

Source is versioned per publish and built server-side; the canvas app
renders the published build's artifact from the isolated artifact origin.
- **`posthog-pp-cli canvases list`** - Canvases: agent-built sandboxed browser apps, filed into channels.

Source is versioned per publish and built server-side; the canvas app
renders the published build's artifact from the isolated artifact origin.
- **`posthog-pp-cli canvases partial-update`** - Update canvas metadata (name, author context, pin, generation-task pointer).
- **`posthog-pp-cli canvases retrieve`** - Canvases: agent-built sandboxed browser apps, filed into channels.

Source is versioned per publish and built server-side; the canvas app
renders the published build's artifact from the isolated artifact origin.

### change-requests

Manage change requests

- **`posthog-pp-cli change-requests list`** - List
- **`posthog-pp-cli change-requests retrieve`** - Retrieve

### code

Manage code

- **`posthog-pp-cli code invites-check-access-retrieve`** - Check whether the authenticated user has access to PostHog Desktop and to Loops.
- **`posthog-pp-cli code invites-redeem-create`** - Redeem a PostHog Desktop invite code to enable access.

### cohorts

Manage cohorts

- **`posthog-pp-cli cohorts all-activity-retrieve`** - All activity retrieve
- **`posthog-pp-cli cohorts create`** - Create
- **`posthog-pp-cli cohorts destroy`** - Hard delete of this model is not allowed. Use a patch API call to set "deleted" to true
- **`posthog-pp-cli cohorts list`** - List
- **`posthog-pp-cli cohorts partial-update`** - Partial update
- **`posthog-pp-cli cohorts retrieve`** - Retrieve
- **`posthog-pp-cli cohorts update`** - Update

### comments

Manage comments

- **`posthog-pp-cli comments count-retrieve`** - Count retrieve
- **`posthog-pp-cli comments create`** - Create a comment.

Support messages are deduplicated: an identical message from the same author on the same
ticket within a short window returns the original comment with a 200 instead of creating a
second one, and a 409 while a concurrent request is still creating it.
- **`posthog-pp-cli comments destroy`** - Hard delete of this model is not allowed. Use a patch API call to set "deleted" to true
- **`posthog-pp-cli comments list`** - List
- **`posthog-pp-cli comments partial-update`** - Partial update
- **`posthog-pp-cli comments retrieve`** - Retrieve
- **`posthog-pp-cli comments update`** - Update

### conversations

Manage conversations

- **`posthog-pp-cli conversations create`** - Unified endpoint that handles both conversation creation and streaming.

- If message is provided: Start new conversation processing
- If no message: Stream from existing conversation
- **`posthog-pp-cli conversations destroy`** - Delete a conversation.
- **`posthog-pp-cli conversations list`** - List
- **`posthog-pp-cli conversations retrieve`** - Retrieve
- **`posthog-pp-cli conversations tickets-ai-feedback-create`** - Record reviewer feedback on an AI reply, captured to the internal analytics project.
- **`posthog-pp-cli conversations tickets-bulk-update-status-create`** - Update the status of multiple tickets in a single request.

Only tickets belonging to the current team are affected; other-team UUIDs
are silently ignored. Tickets the caller lacks editor-level access to (denied
or view-only via object-level access control) are silently skipped too, the
same way single-ticket updates enforce object-level access via get_object().
Tickets already in the requested status are skipped.
- **`posthog-pp-cli conversations tickets-bulk-update-tags-create`** - Bulk update tags on multiple objects.

PAT access: this action has no ``required_scopes=`` on the decorator —
inheriting viewsets must add ``"bulk_update_tags"`` to their
``scope_object_write_actions`` list to accept personal API keys.
Without that opt-in, ``APIScopePermission`` rejects PAT requests with
"This action does not support personal API key access". Done per-viewset
so granting ``<scope>:write`` for one resource doesn't leak access to
sibling resources that share this mixin.

Accepts:
- {"ids": [...], "action": "add"|"remove"|"set", "tags": ["tag1", "tag2"]}

Actions:
- "add": Add tags to existing tags on each object
- "remove": Remove specific tags from each object
- "set": Replace all tags on each object with the provided list
- **`posthog-pp-cli conversations tickets-compose-create`** - Create a new outbound ticket and send the first message to the customer.
- **`posthog-pp-cli conversations tickets-destroy`** - Tickets destroy
- **`posthog-pp-cli conversations tickets-list`** - List tickets with person data attached.
- **`posthog-pp-cli conversations tickets-messages-list`** - Return the message thread for a ticket, ordered chronologically (paginated).
- **`posthog-pp-cli conversations tickets-notes-destroy`** - Soft-delete a private note on a ticket.

Only the note's author can delete it. Customer-facing replies cannot be
deleted via this endpoint.
- **`posthog-pp-cli conversations tickets-notes-partial-update`** - Update a private note on a ticket.

Only the note's author can edit it. Customer-facing replies cannot be
edited (outbound delivery only runs on create).
- **`posthog-pp-cli conversations tickets-partial-update`** - Tickets partial update
- **`posthog-pp-cli conversations tickets-reply-create`** - Post a reply or internal note to a ticket.

With is_private=false, the reply is delivered to the customer via the
ticket's channel (email, Slack, Teams, GitHub). With is_private=true,
the message is stored as an internal note only visible to team members.

Retrying an identical message from the same author within a short window returns the
original message with a 200 rather than posting it twice, and a 409 while a concurrent
request is still creating it.
- **`posthog-pp-cli conversations tickets-retrieve`** - Get single ticket and mark as read by team.
- **`posthog-pp-cli conversations tickets-unread-count-retrieve`** - Get total unread ticket count for the team.

Returns the sum of unread_team_count for all non-resolved tickets visible to the
caller. The team-wide Redis cache (30s TTL, invalidated on changes) is only used for
callers without object-level ticket restrictions, since it holds one unscoped total
per team - serving it to a restricted member would leak counts for tickets they can't
see.
- **`posthog-pp-cli conversations tickets-update`** - Handle ticket updates including assignee changes.
- **`posthog-pp-cli conversations views-create`** - Views create
- **`posthog-pp-cli conversations views-destroy`** - Views destroy
- **`posthog-pp-cli conversations views-list`** - Views list
- **`posthog-pp-cli conversations views-partial-update`** - Views partial update
- **`posthog-pp-cli conversations views-retrieve`** - Views retrieve

### custom-property-definitions

Manage custom property definitions

- **`posthog-pp-cli custom-property-definitions create`** - Create
- **`posthog-pp-cli custom-property-definitions destroy`** - Destroy
- **`posthog-pp-cli custom-property-definitions list`** - List
- **`posthog-pp-cli custom-property-definitions partial-update`** - Partial update
- **`posthog-pp-cli custom-property-definitions retrieve`** - Retrieve
- **`posthog-pp-cli custom-property-definitions update`** - Update
- **`posthog-pp-cli custom-property-definitions values-retrieve`** - Values retrieve

### custom-property-sources

Manage custom property sources

- **`posthog-pp-cli custom-property-sources create`** - Create
- **`posthog-pp-cli custom-property-sources destroy`** - Destroy
- **`posthog-pp-cli custom-property-sources list`** - List
- **`posthog-pp-cli custom-property-sources partial-update`** - Partial update
- **`posthog-pp-cli custom-property-sources retrieve`** - Retrieve
- **`posthog-pp-cli custom-property-sources update`** - Update

### customer-analytics

Manage customer analytics

- **`posthog-pp-cli customer-analytics`** - List accounts with external IDs and their active relationship assignments. Requires a project secret API key with the `account:read` scope.

### customer-journeys

Manage customer journeys

- **`posthog-pp-cli customer-journeys create`** - Create
- **`posthog-pp-cli customer-journeys destroy`** - Destroy
- **`posthog-pp-cli customer-journeys list`** - List
- **`posthog-pp-cli customer-journeys partial-update`** - Partial update
- **`posthog-pp-cli customer-journeys retrieve`** - Retrieve
- **`posthog-pp-cli customer-journeys update`** - Update

### customer-profile-configs

Manage customer profile configs

- **`posthog-pp-cli customer-profile-configs create`** - Create
- **`posthog-pp-cli customer-profile-configs destroy`** - Destroy
- **`posthog-pp-cli customer-profile-configs list`** - List
- **`posthog-pp-cli customer-profile-configs partial-update`** - Partial update
- **`posthog-pp-cli customer-profile-configs retrieve`** - Retrieve
- **`posthog-pp-cli customer-profile-configs update`** - Update

### dashboard-templates

Manage dashboard templates

- **`posthog-pp-cli dashboard-templates copy-between-projects-create`** - Creates a new team-scoped template in the **target** project (URL) from a **team-scoped** source template in the same organization. Global and feature-flag templates return 400. Cross-organization or inaccessible sources return 404. Source and destination projects must differ (400 if equal). Conflicting `template_name` values on the destination are auto-suffixed with `(copy)`, `(copy 2)`, …
- **`posthog-pp-cli dashboard-templates create`** - Create
- **`posthog-pp-cli dashboard-templates destroy`** - Hard delete of this model is not allowed. Use a patch API call to set "deleted" to true
- **`posthog-pp-cli dashboard-templates json-schema-retrieve`** - Json schema retrieve
- **`posthog-pp-cli dashboard-templates list`** - List
- **`posthog-pp-cli dashboard-templates partial-update`** - Partial update
- **`posthog-pp-cli dashboard-templates retrieve`** - Retrieve
- **`posthog-pp-cli dashboard-templates update`** - Update

### dashboards

Manage dashboards

- **`posthog-pp-cli dashboards bulk-update-tags-create`** - Bulk update tags on multiple objects.

PAT access: this action has no ``required_scopes=`` on the decorator —
inheriting viewsets must add ``"bulk_update_tags"`` to their
``scope_object_write_actions`` list to accept personal API keys.
Without that opt-in, ``APIScopePermission`` rejects PAT requests with
"This action does not support personal API key access". Done per-viewset
so granting ``<scope>:write`` for one resource doesn't leak access to
sibling resources that share this mixin.

Accepts:
- {"ids": [...], "action": "add"|"remove"|"set", "tags": ["tag1", "tag2"]}

Actions:
- "add": Add tags to existing tags on each object
- "remove": Remove specific tags from each object
- "set": Replace all tags on each object with the provided list
- **`posthog-pp-cli dashboards create`** - Create
- **`posthog-pp-cli dashboards create-from-template-json-create`** - Create from template json create
- **`posthog-pp-cli dashboards create-unlisted-create`** - Creates an unlisted dashboard from template by tag.
Enforces uniqueness (one per tag per team).
Returns 409 if unlisted dashboard with this tag already exists.
- **`posthog-pp-cli dashboards destroy`** - Hard delete of this model is not allowed. Use a patch API call to set "deleted" to true
- **`posthog-pp-cli dashboards list`** - List
- **`posthog-pp-cli dashboards partial-update`** - Partial update
- **`posthog-pp-cli dashboards retrieve`** - Retrieve
- **`posthog-pp-cli dashboards update`** - Update
- **`posthog-pp-cli dashboards widget-catalog-retrieve`** - List registered dashboard widget types and per-type config_schema documentation for agents.

### data-catalog

Manage data catalog

- **`posthog-pp-cli data-catalog certifications-certify-create`** - Mark the target as certified (prefer this source).
- **`posthog-pp-cli data-catalog certifications-create`** - Trust marks on warehouse tables and views. Reads exclude soft-deleted targets.
- **`posthog-pp-cli data-catalog certifications-deprecate-create`** - Mark the target as deprecated (avoid this source).
- **`posthog-pp-cli data-catalog certifications-destroy`** - Trust marks on warehouse tables and views. Reads exclude soft-deleted targets.
- **`posthog-pp-cli data-catalog certifications-list`** - Trust marks on warehouse tables and views. Reads exclude soft-deleted targets.
- **`posthog-pp-cli data-catalog certifications-retrieve`** - Trust marks on warehouse tables and views. Reads exclude soft-deleted targets.
- **`posthog-pp-cli data-catalog metrics-approve-create`** - Bless a metric as canonical. Returns 409 while the metric is drifted from its insight.
- **`posthog-pp-cli data-catalog metrics-create`** - Create a metric, or refine the one already holding this name for the team.
- **`posthog-pp-cli data-catalog metrics-destroy`** - CRUD for catalog metrics, addressed by their reserved ``name`` (e.g. /metrics/mrr/).
- **`posthog-pp-cli data-catalog metrics-list`** - CRUD for catalog metrics, addressed by their reserved ``name`` (e.g. /metrics/mrr/).
- **`posthog-pp-cli data-catalog metrics-partial-update`** - CRUD for catalog metrics, addressed by their reserved ``name`` (e.g. /metrics/mrr/).
- **`posthog-pp-cli data-catalog metrics-refresh-from-insight-create`** - Re-snapshot the linked insight's current query into the definition.
- **`posthog-pp-cli data-catalog metrics-retrieve`** - CRUD for catalog metrics, addressed by their reserved ``name`` (e.g. /metrics/mrr/).
- **`posthog-pp-cli data-catalog metrics-run-create`** - Execute the metric's definition and return the normalized result envelope.
- **`posthog-pp-cli data-catalog metrics-update`** - CRUD for catalog metrics, addressed by their reserved ``name`` (e.g. /metrics/mrr/).
- **`posthog-pp-cli data-catalog relationship-proposals-accept-create`** - Promote the proposal to a real warehouse join after re-validating and probing it.
- **`posthog-pp-cli data-catalog relationship-proposals-create`** - Reviewed join facts. Accepting one promotes it to a real DataWarehouseJoin; rejections persist.
- **`posthog-pp-cli data-catalog relationship-proposals-list`** - Reviewed join facts. Accepting one promotes it to a real DataWarehouseJoin; rejections persist.
- **`posthog-pp-cli data-catalog relationship-proposals-reject-create`** - Reject the proposal. Persists forever so the pair is never re-proposed.
- **`posthog-pp-cli data-catalog relationship-proposals-retrieve`** - Reviewed join facts. Accepting one promotes it to a real DataWarehouseJoin; rejections persist.

### data-color-themes

Manage data color themes

- **`posthog-pp-cli data-color-themes create`** - Create
- **`posthog-pp-cli data-color-themes destroy`** - Destroy
- **`posthog-pp-cli data-color-themes list`** - List
- **`posthog-pp-cli data-color-themes partial-update`** - Partial update
- **`posthog-pp-cli data-color-themes retrieve`** - Retrieve
- **`posthog-pp-cli data-color-themes update`** - Update

### data-modeling-jobs

Manage data modeling jobs

- **`posthog-pp-cli data-modeling-jobs list`** - List data modeling jobs which are "runs" for our saved queries.
- **`posthog-pp-cli data-modeling-jobs recent-retrieve`** - Get the most recent non-running job for each saved query from the v2 backend.
- **`posthog-pp-cli data-modeling-jobs retrieve`** - List data modeling jobs which are "runs" for our saved queries.
- **`posthog-pp-cli data-modeling-jobs running-retrieve`** - Get all currently running jobs from the v2 backend.

### data-warehouse

Manage data warehouse

- **`posthog-pp-cli data-warehouse check-database-name-retrieve`** - Check if a database name is available.
- **`posthog-pp-cli data-warehouse check-schema-name-retrieve`** - Check if a schema name is free within the organization's managed warehouse.
- **`posthog-pp-cli data-warehouse completed-activity-retrieve`** - Returns completed/non-running activities (jobs with status 'Completed').
Supports pagination and cutoff time filtering.
- **`posthog-pp-cli data-warehouse data-health-issues-retrieve`** - Returns failed/disabled data pipeline items for the Pipeline status side panel.
Includes: materializations, syncs, sources, destinations, and transformations.
- **`posthog-pp-cli data-warehouse data-ops-dashboard-retrieve`** - Returns the data ops overview dashboard ID for this team, creating it if it doesn't exist yet.
- **`posthog-pp-cli data-warehouse delete-org-destroy`** - Remove the organization's provisioning record after teardown, freeing its warehouse name.

Called once the warehouse status reports `deleted`: deprovision tears the warehouse
down, this removes the now-empty org row so the database_name can be reused. Restricted
to organization admins.
- **`posthog-pp-cli data-warehouse deprovision-create`** - Start deprovisioning the organization's managed warehouse. Restricted to organization admins.
- **`posthog-pp-cli data-warehouse job-stats-retrieve`** - Returns success and failed job statistics for the last 1, 7, or 30 days.
Query parameter 'days' can be 1, 7, or 30 (default: 7).
- **`posthog-pp-cli data-warehouse managed-warehouse-data-status-retrieve`** - Get events, persons, and imported source readiness for the managed warehouse.
- **`posthog-pp-cli data-warehouse managed-warehouse-source-schemas-retrieve`** - Per-schema backfill and live import status for one source, for the Overview tab's drill-down modal — the main status endpoint only returns a per-source rollup.
- **`posthog-pp-cli data-warehouse onboard-team-create`** - Onboard this project onto the organization's existing managed warehouse.

Requires a schema name and records the project's membership in the Duckgres control plane.
Restricted to organization admins.
- **`posthog-pp-cli data-warehouse property-values-retrieve`** - API endpoints for data warehouse aggregate statistics and operations.
- **`posthog-pp-cli data-warehouse provision-create`** - Start provisioning a managed warehouse for this organization (shared by all its teams).
- **`posthog-pp-cli data-warehouse reset-password-create`** - Reset the root password for the managed warehouse.
- **`posthog-pp-cli data-warehouse running-activity-retrieve`** - Returns currently running activities (jobs with status 'Running').
Supports pagination and cutoff time filtering.
- **`posthog-pp-cli data-warehouse total-rows-stats-retrieve`** - Returns aggregated statistics for the data warehouse total rows processed within the current billing period.
Used by the frontend data warehouse scene to display usage information.
- **`posthog-pp-cli data-warehouse warehouse-status-retrieve`** - Get the current provisioning status of the managed warehouse, with this project's onboarding state.

### dataset-items

Manage dataset items

- **`posthog-pp-cli dataset-items create`** - Create an item and its first immutable version. An identical client item ID retry returns the existing item. A different payload or an archived match returns a conflict.
- **`posthog-pp-cli dataset-items list`** - List a dataset's current items or its exact contents at a prior revision.
- **`posthog-pp-cli dataset-items partial-update`** - Create a new immutable item version from editable fields.
- **`posthog-pp-cli dataset-items retrieve`** - Retrieve the current item version or the version visible at an exact dataset revision.

### datasets

Manage datasets

- **`posthog-pp-cli datasets create`** - Create an empty dataset. Its first revision is created with its first item.
- **`posthog-pp-cli datasets list`** - List active datasets by default, or archived datasets when requested.
- **`posthog-pp-cli datasets partial-update`** - Update descriptive dataset fields without changing its revision.
- **`posthog-pp-cli datasets retrieve`** - Retrieve an active or archived dataset.

### early-access-feature

Manage early access feature

- **`posthog-pp-cli early-access-feature create`** - Create
- **`posthog-pp-cli early-access-feature destroy`** - Destroy
- **`posthog-pp-cli early-access-feature list`** - List
- **`posthog-pp-cli early-access-feature partial-update`** - Partial update
- **`posthog-pp-cli early-access-feature retrieve`** - Retrieve
- **`posthog-pp-cli early-access-feature update`** - Update

### elements

Manage elements

- **`posthog-pp-cli elements create`** - Create
- **`posthog-pp-cli elements destroy`** - Destroy
- **`posthog-pp-cli elements list`** - List
- **`posthog-pp-cli elements partial-update`** - Partial update
- **`posthog-pp-cli elements retrieve`** - Retrieve
- **`posthog-pp-cli elements stats-retrieve`** - Counts of $autocapture, $rageclick, and $dead_click events grouped by the element chain
they occurred on, ordered by count. Defaults to all three event types; narrow with the
include parameter.
- **`posthog-pp-cli elements update`** - Update
- **`posthog-pp-cli elements values-list`** - Values list

### endpoints

Manage endpoints

- **`posthog-pp-cli endpoints create`** - Create a new endpoint.
- **`posthog-pp-cli endpoints destroy`** - Delete an endpoint and clean up materialized query.
- **`posthog-pp-cli endpoints last-execution-times-create`** - Get the most recent execution time per endpoint (endpoint-level). Timestamps are recorded by the run path for personal-API-key calls. For per-version usage, query the query_log table directly.
- **`posthog-pp-cli endpoints list`** - List all endpoints for the team.
- **`posthog-pp-cli endpoints materialization-conditions-retrieve`** - Get the source code of the live materialization checks, plus the rewrite contract. Lets an agent rewrite a rejected endpoint query itself: fetch these conditions, produce a semantically equivalent query that passes every check, update the endpoint with it, then confirm via materialization_status. The source is read from the running system, so it always matches the checks this instance enforces.
- **`posthog-pp-cli endpoints partial-update`** - Update an existing endpoint.
- **`posthog-pp-cli endpoints retrieve`** - Retrieve an endpoint, or a specific version via ?version=N.
- **`posthog-pp-cli endpoints update`** - Update an existing endpoint. Parameters are optional. Pass version in body or ?version=N query param to target a specific version.

### engineering-analytics

Manage engineering analytics

- **`posthog-pp-cli engineering-analytics author-workflow-costs`** - One author's estimated CI cost split by workflow over a window (date_from default -30d), highest spend first. Runs are attributed to the author through their pull requests (attribution is by PR number). Returns an empty list when the job-level source isn't synced.
- **`posthog-pp-cli engineering-analytics broken-tests`** - The broken-tests triage panel: live CI failures over the last 2 days grouped into distinct failures (by test id + normalized error signature) and classified by how each is behaving right now — breaking trunk, blocking the merge queue, a new failure spreading across branches, probably-resolved, flaky, or one PR's own problem — ranked with the most urgent first. A blocking_merge_queue row is a failure on a merge-queue gate branch that never hit trunk: the commit had already passed the PR's own CI, so it is the semantic conflict the queue exists to catch, and it is holding up landings. Also returns breaking_master_jobs, the default-branch jobs whose latest run is red. Reach for this to answer 'what CI failures should I care about right now'; expand a row's latest_run_id via run_failure_logs for the failing lines. Fingerprinting is pytest-only for now (jest/playwright/cargo failures aren't grouped yet), and the breaking/resolved distinction needs the job-level source synced — without it those failures fall through to flaky/pr_only rather than being misreported.
- **`posthog-pp-cli engineering-analytics ci-cards`** - Headline counts for the open-PR backlog: open PRs, distinct repos, stuck PRs (open, non-draft, non-bot, older than 7 days), and PRs with failing CI. The failing-CI count rests on the head-SHA join and can lag until late CI completions settle.
- **`posthog-pp-cli engineering-analytics ci-failure-logs`** - The thinned CI failure logs for a pull request, grouped by failed job. Resolves the PR to its workflow runs via the pull_requests association (all of the PR's pushes, not just the latest commit), then reads the Logs product joined on run_id. Returns failed jobs only (the worker fetches logs for failures); logs_available is false when CI hasn't failed, the logs aged out of the short Logs retention, or a fork PR has no run association. Each line carries its original 1-based line number in the full pre-thinning log; lines are the failure region (errors plus surrounding context, with omission markers), capped per job and overall.
- **`posthog-pp-cli engineering-analytics ci-signals-config-retrieve`** - Return the atomic CI Signals configuration and aggregate GitHub warehouse sync status.
- **`posthog-pp-cli engineering-analytics ci-signals-config-update`** - Enable or disable all CI signal detectors in one transaction.
- **`posthog-pp-cli engineering-analytics current-branch-health`** - Current default-branch CI verdict over the fixed last-24-hours window. Counts every workflow whose latest completed run failed or timed out; failing workflow names are a bounded preview. The default branch is detected from the same window, independently of analytics date filters.
- **`posthog-pp-cli engineering-analytics flaky-tests`** - The active test-health queue: pytest and Jest tests worth acting on now, from the per-test CI spans, over a window (default -7d, maximum 30 days). Evidence is counted per CI run, never per span or run attempt. A test is a 'confirmed_flake' when one commit both failed and passed it (a 'Re-run failed jobs' attempt went green, or an in-job retry recovered it); 'quarantined' when a tolerated failure is recorded while it is masked; otherwise 'suspected_regression'. It qualifies on any same-commit recovery, any master/main failure, a quarantined failure, or failures on at least min_failed_prs distinct PRs. Counts are absolute, never rates: CI emits every failure but omits ordinary passing spans, so there is no execution denominator. 'suspected_regression' means no recovery was recorded in this data, not that the test never flakes.
- **`posthog-pp-cli engineering-analytics job-aggregates`** - Per-job aggregates for one workflow over a window (default -30d), one row per de-sharded job name (matrix shards aggregate together), busiest first: queue p50, duration p50/p95, failure rate, retry pressure, run share (below 1.0 = conditional job), and billable cost. Jobs always need their run as context — this is the aggregate view; use workflow_jobs for one run's jobs. Empty when the job-level source isn't synced.
- **`posthog-pp-cli engineering-analytics master-failures`** - Default-branch failures over a window (default -24h), grouped error-tracking style by (workflow, de-sharded failing job) with a run count and first/last seen, newest group first. `branch` overrides the detected default branch. PR-branch failures are deliberately excluded — at monorepo volume a flat feed is a firehose; those surface per PR. Groups degrade to workflow level (failed_job '') when the job-level source isn't synced.
- **`posthog-pp-cli engineering-analytics pr-cost`** - Estimated CI cost for a pull request, summed over the jobs of all its workflow runs. Billable self-hosted Linux runners only — provider-hosted (free GitHub-hosted) and non-Linux jobs are excluded. Every figure is zero/null with `jobs_available` false when the job-level source isn't synced yet. `llm_spend` carries the agent LLM token spend attributed to the PR by git branch, or null when no `$ai_generation` event matched.
- **`posthog-pp-cli engineering-analytics pr-lifecycle`** - The timeline of a single pull request: header plus ordered events (opened, CI started/finished, merged or closed). Use this to answer 'where is this PR stuck and what happened to it'. This is a partial view: review and comment events are not yet available.
- **`posthog-pp-cli engineering-analytics pr-runs`** - Every workflow run attributed to a pull request, across all its commits (grouped by head SHA client-side), newest first. Run-level only.
- **`posthog-pp-cli engineering-analytics pull-requests`** - Open pull requests plus any merged or closed since date_from (default -30d), newest first, each with its head-SHA CI rollup. The list is capped; when more match, `truncated` is true and the ci_cards counts can exceed it. open_to_merge_seconds is coarse — it fuses draft and ready-for-review time; CI counts can lag until late completions settle.
- **`posthog-pp-cli engineering-analytics quarantine`** - The repository's checked-in .test_quarantine.json: flaky tests temporarily quarantined with a hard expiry, classified by urgency (overdue, in grace, expiring soon, active). `available` is false when the repo has no quarantine file — that is not an error. Parsing is fail-open: malformed entries are reported in parse_errors while well-formed ones are kept.
- **`posthog-pp-cli engineering-analytics quarantine-request`** - Opens a pull request that edits the repository's checked-in .test_quarantine.json — and, for a new quarantine, a tracking issue the PR links but does not close. The file stays the source of truth that CI enforces; this never bypasses it. A quarantine only affects CI runs that start after the PR merges.
- **`posthog-pp-cli engineering-analytics repo-overview`** - Repo-level headline aggregates over a window (default -30d): run count, success rate, re-run cycles, merged-PR count (bots included), median PR open-to-merge (bots and drafts excluded; coarse — draft and ready time fused), and billable minutes + estimated cost (with the merge-queue slice of billable minutes broken out) — each with its equal-length previous-window twin so a caller can render honest deltas. Also carries the detected default branch and its completed-run history series (skippable via include_series=false). Cost figures are null until the job-level source is synced.
- **`posthog-pp-cli engineering-analytics repo-run-activity`** - Default-branch health as compact chart points over a window (default -30d), newest first, for the repo-hub run-activity chart. All of a commit's workflow runs are collapsed into ONE point per commit (head SHA): its earliest workflow start, wall-clock duration until the last workflow settled (null while any is still running), and an overall conclusion that is 'failure' if any workflow decisively failed, else 'success' when at least one passed, else 'neutral'. `branch` overrides the detected default branch. `truncated` is true when more commits matched than the cap, so the chart covers only the most recent commits.
- **`posthog-pp-cli engineering-analytics resolve-branch`** - Resolve a git branch to the pull request(s) it belongs to — the cross-product link seam so another product (the LLM analytics UI) can turn a git branch into a PR detail link. Matches the PR's head ref, open PRs first then most recently updated. Pass `timestamp` (the trace's capture time) to prefer the PR that was active at that moment when a branch name has been reused across PRs. `branch` is required. Returns a possibly-empty, possibly-multi list — an empty list is a valid 200 (the caller renders a plain chip).
- **`posthog-pp-cli engineering-analytics run-failure-logs`** - The thinned CI failure logs of one workflow run, grouped by failed job — the run-scoped twin of ci_failure_logs for surfaces that aren't PR-scoped (default-branch failures, the run page). logs_available is false when the run didn't fail or its logs aged out of the short Logs retention.
- **`posthog-pp-cli engineering-analytics sources`** - The team's selectable GitHub repositories, oldest source first — one entry per repository a source is configured to sync, so a source syncing several repositories appears once per repo. Populate a repo picker from this and pass a chosen entry's `id` back as `source_id` and its `repo` back as `repo` to the other endpoints. Includes repositories whose tables aren't fully synced yet.
- **`posthog-pp-cli engineering-analytics team-ci-activity`** - One owning team's CI test activity: per-test current-vs-prior signal pairs (the before/after comparison) over the window and its equal-length prior twin. Signal = runs where an owned test failed, errored, or a retry recovered it. Counts are absolute, never rates: CI emits every failure but omits ordinary passing spans, so there is no execution denominator. 'suspected_regression' means no recovery was recorded in this data, not that the test never flakes.
- **`posthog-pp-cli engineering-analytics team-ci-health`** - Per-owning-team rollup of the CI test surfaces each team owns, over the same run evidence as flaky_tests and with the same meaning of flaky: flaky_test_count is owned tests one commit was seen both failing and passing in the window, regression_test_count is owned tests that failed with no such proof and still hit the blast-radius bar, plus failed/recovery/quarantined run counts. Each has an equal-length previous-window twin for honest deltas. Ownership is stamped on the spans at CI emission time from the repo's ownership map (products/*/product.yaml + CODEOWNERS); unstamped spans aggregate under the literal team 'unowned', and a re-stamped test lands under its latest owner only. Teams are organizational owners of code surfaces, never authors. Counts are absolute, never rates: CI emits every failure but omits ordinary passing spans, so there is no execution denominator. 'suspected_regression' means no recovery was recorded in this data, not that the test never flakes.
- **`posthog-pp-cli engineering-analytics team-merge-trend`** - One team's daily time-to-merge trend: the median and average open→merge seconds over the PRs the team's members merged each day (PR author login → GitHub org team membership). Team-level aggregates only, never per-member figures or cross-team rankings. Timing is the coarse open→merge (draft + review time combined); bots are excluded. Requires the GitHub source's team_members snapshot; has_membership_data is false without it.
- **`posthog-pp-cli engineering-analytics workflow-health`** - Per-workflow CI health over a window (default last 24 hours, maximum 366 days): run count, success rate, p50/p95 duration, last failure time, latest-run status, and a zero-filled run history bucketed by hour/day/week to fit the window. p50/p95 are over successful runs only, so cancelled (superseded) and failed runs never bias the duration trend. Optionally scope to a single git branch via `branch`, or to attributed pull-request runs via `run_scope=pull_request`. Use this for 'is CI getting slower' and 'which workflow is the long pole'; compare two windows to get a trend.
- **`posthog-pp-cli engineering-analytics workflow-jobs`** - Jobs of a single workflow run attempt, with per-job duration, runner tier, and estimated cost. Scoped to one run_attempt (the latest unless specified) so a re-run's attempts don't merge. Returns an empty list when the job-level source isn't synced yet.
- **`posthog-pp-cli engineering-analytics workflow-run`** - A single workflow run: status, conclusion, duration, branch, attempt, and the attributed pull request. Run-level only — per-job and per-step detail are not tracked yet.
- **`posthog-pp-cli engineering-analytics workflow-run-activity`** - Compact per-run points for a single workflow over a window (date_from default -30d), newest first, for the run-activity chart: each run's start time, duration, conclusion, branch, and attributed PR. Optionally scope to a single git branch via `branch`, matching workflow_runs. Leaner and higher-capped than workflow_runs so the chart spans the full window even on busy workflows; `truncated` is true when the cap is hit, so the chart covers only the most recent runs.
- **`posthog-pp-cli engineering-analytics workflow-runner-costs`** - A workflow's estimated CI cost broken down by runner tier over a window (date_from default -30d), highest spend first. Optionally scope to a single git branch via `branch`. Returns an empty list when the job-level source isn't synced.
- **`posthog-pp-cli engineering-analytics workflow-runs`** - Runs of a single workflow within a repo over a window (date_from default -30d), newest first. Optionally scope to a single git branch via `branch`. Each row is run-level — per-job and per-step detail are not tracked yet. Use this as the GitHub 'workflow' page between the workflow list and a single run.

### environments

Manage environments

- **`posthog-pp-cli environments create`** - Deprecated: use /api/environments/{id}/ instead.
- **`posthog-pp-cli environments destroy`** - Deprecated: use /api/environments/{id}/ instead.
- **`posthog-pp-cli environments list`** - Deprecated: use /api/environments/{id}/ instead.
- **`posthog-pp-cli environments partial-update`** - Deprecated: use /api/environments/{id}/ instead.
- **`posthog-pp-cli environments retrieve`** - Deprecated: use /api/environments/{id}/ instead.
- **`posthog-pp-cli environments update`** - Deprecated: use /api/environments/{id}/ instead.

### error-tracking

Manage error tracking

- **`posthog-pp-cli error-tracking assignment-rules-create`** - Assignment rules create
- **`posthog-pp-cli error-tracking assignment-rules-destroy`** - Assignment rules destroy
- **`posthog-pp-cli error-tracking assignment-rules-list`** - Assignment rules list
- **`posthog-pp-cli error-tracking assignment-rules-partial-update`** - Assignment rules partial update
- **`posthog-pp-cli error-tracking assignment-rules-reorder-partial-update`** - Assignment rules reorder partial update
- **`posthog-pp-cli error-tracking assignment-rules-retrieve`** - Assignment rules retrieve
- **`posthog-pp-cli error-tracking assignment-rules-update`** - Assignment rules update
- **`posthog-pp-cli error-tracking bypass-rules-create`** - Bypass rules create
- **`posthog-pp-cli error-tracking bypass-rules-destroy`** - Bypass rules destroy
- **`posthog-pp-cli error-tracking bypass-rules-list`** - Bypass rules list
- **`posthog-pp-cli error-tracking bypass-rules-partial-update`** - Bypass rules partial update
- **`posthog-pp-cli error-tracking bypass-rules-reorder-partial-update`** - Bypass rules reorder partial update
- **`posthog-pp-cli error-tracking bypass-rules-retrieve`** - Bypass rules retrieve
- **`posthog-pp-cli error-tracking bypass-rules-update`** - Bypass rules update
- **`posthog-pp-cli error-tracking external-references-create`** - External references create
- **`posthog-pp-cli error-tracking external-references-destroy`** - Hard delete of this model is not allowed. Use a patch API call to set "deleted" to true
- **`posthog-pp-cli error-tracking external-references-list`** - External references list
- **`posthog-pp-cli error-tracking external-references-retrieve`** - External references retrieve
- **`posthog-pp-cli error-tracking fingerprints-destroy`** - Hard delete of this model is not allowed. Use a patch API call to set "deleted" to true
- **`posthog-pp-cli error-tracking fingerprints-list`** - Fingerprints list
- **`posthog-pp-cli error-tracking fingerprints-resolve-retrieve`** - Fingerprints resolve retrieve
- **`posthog-pp-cli error-tracking fingerprints-retrieve`** - Fingerprints retrieve
- **`posthog-pp-cli error-tracking git-provider-file-links-resolve-github-retrieve`** - Git provider file links resolve github retrieve
- **`posthog-pp-cli error-tracking git-provider-file-links-resolve-gitlab-retrieve`** - Git provider file links resolve gitlab retrieve
- **`posthog-pp-cli error-tracking grouping-rules-create`** - Grouping rules create
- **`posthog-pp-cli error-tracking grouping-rules-destroy`** - Grouping rules destroy
- **`posthog-pp-cli error-tracking grouping-rules-list`** - Grouping rules list
- **`posthog-pp-cli error-tracking grouping-rules-partial-update`** - Grouping rules partial update
- **`posthog-pp-cli error-tracking grouping-rules-reorder-partial-update`** - Grouping rules reorder partial update
- **`posthog-pp-cli error-tracking grouping-rules-retrieve`** - Grouping rules retrieve
- **`posthog-pp-cli error-tracking grouping-rules-update`** - Grouping rules update
- **`posthog-pp-cli error-tracking issues-activity-retrieve`** - Issues activity retrieve
- **`posthog-pp-cli error-tracking issues-all-activity-retrieve`** - Issues all activity retrieve
- **`posthog-pp-cli error-tracking issues-assign-partial-update`** - Issues assign partial update
- **`posthog-pp-cli error-tracking issues-bulk-create`** - Issues bulk create
- **`posthog-pp-cli error-tracking issues-cohort-update`** - Issues cohort update
- **`posthog-pp-cli error-tracking issues-destroy`** - Hard delete of this model is not allowed. Use a patch API call to set "deleted" to true
- **`posthog-pp-cli error-tracking issues-exists-retrieve`** - Issues exists retrieve
- **`posthog-pp-cli error-tracking issues-list`** - Issues list
- **`posthog-pp-cli error-tracking issues-merge-create`** - Issues merge create
- **`posthog-pp-cli error-tracking issues-partial-update`** - Issues partial update
- **`posthog-pp-cli error-tracking issues-retrieve`** - Issues retrieve
- **`posthog-pp-cli error-tracking issues-split-create`** - Issues split create
- **`posthog-pp-cli error-tracking issues-update`** - Issues update
- **`posthog-pp-cli error-tracking issues-values-retrieve`** - Issues values retrieve
- **`posthog-pp-cli error-tracking query-issue-create`** - Fetch one error tracking issue with impact counts, top in_app frame, latest release, and optional sparkline.
- **`posthog-pp-cli error-tracking query-issue-events-create`** - Fetch sampled exception events, stack traces, browser/SDK context, URL, and $session_id values for one issue.
- **`posthog-pp-cli error-tracking query-issues-list-create`** - List error tracking issues with typed filters and compact aggregate counts.
- **`posthog-pp-cli error-tracking recommendations-dismiss-create`** - Recommendations dismiss create
- **`posthog-pp-cli error-tracking recommendations-list`** - Recommendations list
- **`posthog-pp-cli error-tracking recommendations-refresh-create`** - Recommendations refresh create
- **`posthog-pp-cli error-tracking recommendations-restore-create`** - Recommendations restore create
- **`posthog-pp-cli error-tracking releases-create`** - Releases create
- **`posthog-pp-cli error-tracking releases-destroy`** - Releases destroy
- **`posthog-pp-cli error-tracking releases-hash-retrieve`** - Releases hash retrieve
- **`posthog-pp-cli error-tracking releases-list`** - Releases list
- **`posthog-pp-cli error-tracking releases-partial-update`** - Releases partial update
- **`posthog-pp-cli error-tracking releases-retrieve`** - Releases retrieve
- **`posthog-pp-cli error-tracking releases-update`** - Releases update
- **`posthog-pp-cli error-tracking settings-retrieve-settings-retrieve`** - Settings retrieve settings retrieve
- **`posthog-pp-cli error-tracking settings-update-settings-partial-update`** - Settings update settings partial update
- **`posthog-pp-cli error-tracking spike-detection-config-list`** - Spike detection config list
- **`posthog-pp-cli error-tracking spike-detection-config-update-config-partial-update`** - Spike detection config update config partial update
- **`posthog-pp-cli error-tracking spike-events-list`** - Spike events list
- **`posthog-pp-cli error-tracking stack-frames-batch-get-create`** - Stack frames batch get create
- **`posthog-pp-cli error-tracking stack-frames-destroy`** - Hard delete of this model is not allowed. Use a patch API call to set "deleted" to true
- **`posthog-pp-cli error-tracking stack-frames-list`** - Stack frames list
- **`posthog-pp-cli error-tracking stack-frames-retrieve`** - Stack frames retrieve
- **`posthog-pp-cli error-tracking suppression-rules-create`** - Suppression rules create
- **`posthog-pp-cli error-tracking suppression-rules-destroy`** - Suppression rules destroy
- **`posthog-pp-cli error-tracking suppression-rules-list`** - Suppression rules list
- **`posthog-pp-cli error-tracking suppression-rules-partial-update`** - Suppression rules partial update
- **`posthog-pp-cli error-tracking suppression-rules-reorder-partial-update`** - Suppression rules reorder partial update
- **`posthog-pp-cli error-tracking suppression-rules-retrieve`** - Suppression rules retrieve
- **`posthog-pp-cli error-tracking suppression-rules-update`** - Suppression rules update
- **`posthog-pp-cli error-tracking symbol-sets-bulk-delete-create`** - Symbol sets bulk delete create
- **`posthog-pp-cli error-tracking symbol-sets-bulk-finish-upload-create`** - Symbol sets bulk finish upload create
- **`posthog-pp-cli error-tracking symbol-sets-bulk-start-upload-create`** - Symbol sets bulk start upload create
- **`posthog-pp-cli error-tracking symbol-sets-destroy`** - Symbol sets destroy
- **`posthog-pp-cli error-tracking symbol-sets-download-retrieve`** - Return a presigned URL for downloading the symbol set's source map.
- **`posthog-pp-cli error-tracking symbol-sets-finish-upload-update`** - Symbol sets finish upload update
- **`posthog-pp-cli error-tracking symbol-sets-list`** - Symbol sets list
- **`posthog-pp-cli error-tracking symbol-sets-retrieve`** - Symbol sets retrieve

### evaluation-directories

Manage evaluation directories

- **`posthog-pp-cli evaluation-directories create`** - Create
- **`posthog-pp-cli evaluation-directories destroy`** - Destroy
- **`posthog-pp-cli evaluation-directories list`** - List
- **`posthog-pp-cli evaluation-directories partial-update`** - Partial update
- **`posthog-pp-cli evaluation-directories retrieve`** - Retrieve

### evaluation-runs

Manage evaluation runs

- **`posthog-pp-cli evaluation-runs <project_id>`** - Create a new evaluation run.

This endpoint validates the request and enqueues a Temporal workflow
to asynchronously execute the evaluation.

### evaluations

Manage evaluations

- **`posthog-pp-cli evaluations create`** - Create
- **`posthog-pp-cli evaluations destroy`** - Hard delete of this model is not allowed. Use a patch API call to set "deleted" to true
- **`posthog-pp-cli evaluations list`** - List
- **`posthog-pp-cli evaluations partial-update`** - Partial update
- **`posthog-pp-cli evaluations retrieve`** - Retrieve
- **`posthog-pp-cli evaluations test-hog-create`** - Test Hog evaluation code against sample events without saving.
- **`posthog-pp-cli evaluations update`** - Update

### event-definitions

Manage event definitions

- **`posthog-pp-cli event-definitions bulk-update-tags-create`** - Add, remove, or replace tags across multiple event definitions in one request.

Overrides ``TaggedItemViewSetMixin.bulk_update_tags``, which assumes integer PKs and runs
object-level access-control filtering. Event definitions use UUID PKs and are not an
object-level access-controlled resource — project membership (enforced by the viewset) is
the only boundary, matching the single-object update path — so this scopes by project and
skips the per-object editor check. Tags live on the base ``EventDefinition`` row, so it
operates there regardless of the enterprise extension.
- **`posthog-pp-cli event-definitions bulk-update-verified-create`** - Mark multiple event definitions as verified or unverified in one request.

In the same vein as ``bulk_update_tags``, but ``verified`` lives on the enterprise
``EnterpriseEventDefinition`` extension rather than the base row, so this action:
- requires an enterprise license;
- scopes by project (``team__project_id``) and relies on project membership — the same
  boundary the single-object update path uses — rather than object-level RBAC;
- lazily promotes ingestion-created base rows to ``EnterpriseEventDefinition`` (mirroring
  ``_get_event_definition``) before setting ``verified``;
- mirrors the single-object semantics: verifying stamps ``verified_by``/``verified_at`` and
  unhides the event (an event cannot be both hidden and verified); unverifying clears them;
- logs a "changed" activity per event so the History tab matches the single-object path.

Events already in the target state are skipped (not re-written, not logged).
- **`posthog-pp-cli event-definitions by-name-retrieve`** - Get event definition by exact name
- **`posthog-pp-cli event-definitions create`** - Create
- **`posthog-pp-cli event-definitions destroy`** - Destroy
- **`posthog-pp-cli event-definitions golang-retrieve`** - Golang retrieve
- **`posthog-pp-cli event-definitions list`** - List
- **`posthog-pp-cli event-definitions partial-update`** - Partial update
- **`posthog-pp-cli event-definitions primary-properties-retrieve`** - Resolve team-configured primary properties for event definitions.

The response only contains entries where a non-null primary_property is set on the
EventDefinition. Callers should fall back to the core taxonomy defaults client-side
for names not present in the response.
- **`posthog-pp-cli event-definitions python-retrieve`** - Python retrieve
- **`posthog-pp-cli event-definitions retrieve`** - Retrieve
- **`posthog-pp-cli event-definitions typescript-retrieve`** - Typescript retrieve
- **`posthog-pp-cli event-definitions update`** - Update

### event-filter

Manage event filter

- **`posthog-pp-cli event-filter create`** - Create or update the event filter config.
- **`posthog-pp-cli event-filter metrics-retrieve`** - Single event filter per team.
GET  /event_filter/ — returns the config (or null if not yet created)
POST /event_filter/ — creates or updates the config (upsert)
GET  /event_filter/metrics/ — time-series metrics
GET  /event_filter/metrics/totals/ — aggregate totals
- **`posthog-pp-cli event-filter metrics-totals-retrieve`** - Single event filter per team.
GET  /event_filter/ — returns the config (or null if not yet created)
POST /event_filter/ — creates or updates the config (upsert)
GET  /event_filter/metrics/ — time-series metrics
GET  /event_filter/metrics/totals/ — aggregate totals
- **`posthog-pp-cli event-filter retrieve`** - Returns the event filter config for the team, or null if not yet created.

### event-schemas

Manage event schemas

- **`posthog-pp-cli event-schemas create`** - Create
- **`posthog-pp-cli event-schemas destroy`** - Destroy
- **`posthog-pp-cli event-schemas list`** - List
- **`posthog-pp-cli event-schemas partial-update`** - Partial update
- **`posthog-pp-cli event-schemas update`** - Update

### event-streams

Manage event streams

- **`posthog-pp-cli event-streams create`** - The caller's event stream: a live feed of selected accounts' events posted to a
Slack channel of their choice. Per-user — each team member owns at most one stream, and
every endpoint is scoped to the caller's own. Delivery runs through a managed CDP
destination that is re-provisioned inside the same transaction as every write, so
config and delivery can't drift apart.
- **`posthog-pp-cli event-streams destroy`** - The caller's event stream: a live feed of selected accounts' events posted to a
Slack channel of their choice. Per-user — each team member owns at most one stream, and
every endpoint is scoped to the caller's own. Delivery runs through a managed CDP
destination that is re-provisioned inside the same transaction as every write, so
config and delivery can't drift apart.
- **`posthog-pp-cli event-streams list`** - The caller's event stream: a live feed of selected accounts' events posted to a
Slack channel of their choice. Per-user — each team member owns at most one stream, and
every endpoint is scoped to the caller's own. Delivery runs through a managed CDP
destination that is re-provisioned inside the same transaction as every write, so
config and delivery can't drift apart.
- **`posthog-pp-cli event-streams partial-update`** - The caller's event stream: a live feed of selected accounts' events posted to a
Slack channel of their choice. Per-user — each team member owns at most one stream, and
every endpoint is scoped to the caller's own. Delivery runs through a managed CDP
destination that is re-provisioned inside the same transaction as every write, so
config and delivery can't drift apart.
- **`posthog-pp-cli event-streams update`** - The caller's event stream: a live feed of selected accounts' events posted to a
Slack channel of their choice. Per-user — each team member owns at most one stream, and
every endpoint is scoped to the caller's own. Delivery runs through a managed CDP
destination that is re-provisioned inside the same transaction as every write, so
config and delivery can't drift apart.

### events

Manage events

- **`posthog-pp-cli events list`** - This endpoint allows you to list and filter events.
        It is effectively deprecated and is kept only for backwards compatibility.
        If you ever ask about it you will be advised to not use it...
        If you want to ad-hoc list or aggregate events, use the Query endpoint instead.
        If you want to export all events or many pages of events you should use our CDP/Batch Exports products instead.
- **`posthog-pp-cli events retrieve`** - Retrieve
- **`posthog-pp-cli events values-retrieve`** - Values retrieve

### experiment-holdouts

Manage experiment holdouts

- **`posthog-pp-cli experiment-holdouts create`** - Create
- **`posthog-pp-cli experiment-holdouts destroy`** - Destroy
- **`posthog-pp-cli experiment-holdouts list`** - List
- **`posthog-pp-cli experiment-holdouts partial-update`** - Partial update
- **`posthog-pp-cli experiment-holdouts retrieve`** - Retrieve
- **`posthog-pp-cli experiment-holdouts update`** - Update

### experiment-saved-metrics

Manage experiment saved metrics

- **`posthog-pp-cli experiment-saved-metrics create`** - Create
- **`posthog-pp-cli experiment-saved-metrics destroy`** - Destroy
- **`posthog-pp-cli experiment-saved-metrics list`** - List
- **`posthog-pp-cli experiment-saved-metrics partial-update`** - Partial update
- **`posthog-pp-cli experiment-saved-metrics retrieve`** - Retrieve
- **`posthog-pp-cli experiment-saved-metrics update`** - Update

### experiments

Manage experiments

- **`posthog-pp-cli experiments calculate-running-time-create`** - Estimate the recommended sample size and running time for an experiment.

Pure statistical calculation — does not read or write any experiment. Pass the metric type, a
minimum detectable effect, and either a baseline value or raw baseline statistics. When
`exposure_rate_per_day` is provided, the response also includes the estimated running time in days.
- **`posthog-pp-cli experiments create`** - Create a new experiment in draft status with optional metrics.
- **`posthog-pp-cli experiments create-from-prompt-create`** - Create an experiment that compares N versions of an LLM prompt using a metric template.

The user picks 2+ versions of an existing LLMPrompt and 1+ metric templates
(cost / latency / eval_pass_rate). The endpoint builds the matching variants
(control + test-N, each named after its prompt version) and attaches one
metric per selected template, each scoped to the prompt's $ai_prompt_name.
Resulting experiment is in draft state.
- **`posthog-pp-cli experiments destroy`** - Hard delete of this model is not allowed. Use a patch API call to set "deleted" to true
- **`posthog-pp-cli experiments list`** - List experiments for the current project. Supports filtering by status and archival state.
- **`posthog-pp-cli experiments partial-update`** - Update an experiment. Use this to modify experiment properties such as name, description, metrics, variants, and configuration. Metrics can be added, changed and removed at any time. Feature-flag config (variants, rollout, payloads) is sent via the feature_flag object.
- **`posthog-pp-cli experiments prompt-templates-retrieve`** - List the LLM metric templates that can be passed to `create_from_prompt`.
- **`posthog-pp-cli experiments retrieve`** - Retrieve a single experiment by ID, including its current status, metrics, feature flag, and results metadata.
- **`posthog-pp-cli experiments session-context-retrieve`** - Resolve which experiments (and variants) a session recording saw. Variants come from the session's $feature_flag_called events and stamped $feature/<key> event properties — flag evaluation, which may differ from an experiment's exposure criteria.
- **`posthog-pp-cli experiments session-contexts-create`** - Resolve experiment context for a batch of session recordings.

Batch variant of `session_context`, used to prefetch the replay player's experiments
box for a whole recordings list in one request. POST because the id list doesn't fit a
query string; the endpoint only reads. Already-computed sessions are served from (and
cold ones written to) the same short-lived per-viewer cache the single-session endpoint
uses, so opening any prefetched recording renders its context instantly. Sessions whose
recording metadata doesn't exist yet are omitted from the response, as are recordings
the caller can't access and sessions beyond the batch's recording-day budget (each
distinct recording day costs its own set of ClickHouse scans, so only the most recent
days are computed per request).
- **`posthog-pp-cli experiments stats-retrieve`** - Mixin for ViewSets to handle approval-gate exceptions raised from decorated serializers.

Intercepts ApprovalRequired (409) and PolicyConflict (400) raised by the @approval_gate
decorator on serializer methods and converts them into the same responses the viewset path
produces (see decorators._result_to_response), so both paths share one contract.
- **`posthog-pp-cli experiments update`** - Mixin for ViewSets to handle approval-gate exceptions raised from decorated serializers.

Intercepts ApprovalRequired (409) and PolicyConflict (400) raised by the @approval_gate
decorator on serializer methods and converts them into the same responses the viewset path
produces (see decorators._result_to_response), so both paths share one contract.

### exports

Manage exports

- **`posthog-pp-cli exports create`** - Create
- **`posthog-pp-cli exports list`** - List
- **`posthog-pp-cli exports retrieve`** - Retrieve

### external-data-schemas

Manage external data schemas

- **`posthog-pp-cli external-data-schemas destroy`** - Destroy
- **`posthog-pp-cli external-data-schemas list`** - List
- **`posthog-pp-cli external-data-schemas partial-update`** - Partial update
- **`posthog-pp-cli external-data-schemas retrieve`** - Retrieve
- **`posthog-pp-cli external-data-schemas update`** - Update

### external-data-sources

Manage external data sources

- **`posthog-pp-cli external-data-sources check-cdc-prerequisites-create`** - Validate CDC prerequisites against a live Postgres connection.

Used by the source wizard to surface ✅/❌ checks before source creation,
and by the self-managed setup popup to verify user-created publications.
- **`posthog-pp-cli external-data-sources connect-link-retrieve`** - Return a secure browser link for connecting a data warehouse source.

The link opens a minimal connect page rendering the source's full connection form — OAuth options
included — with no table selection and no source creation. The user authenticates in their browser,
secrets never pass through the agent, and the agent finishes setup afterwards by passing the stored
credential id to data-warehouse-source-setup.
- **`posthog-pp-cli external-data-sources connections-list`** - Create, Read, Update and Delete External data Sources.
- **`posthog-pp-cli external-data-sources create`** - Create, Read, Update and Delete External data Sources.
- **`posthog-pp-cli external-data-sources database-schema-create`** - Create, Read, Update and Delete External data Sources.
- **`posthog-pp-cli external-data-sources destroy`** - Create, Read, Update and Delete External data Sources.
- **`posthog-pp-cli external-data-sources direct-connection-options-list`** - Source types the user can add as a direct connection, driven by the direct-SQL capability
surface so the picker never drifts from the engines we actually support.
- **`posthog-pp-cli external-data-sources draft-custom-manifest-create`** - Draft a Custom REST source manifest from API documentation using an LLM.

Reads the docs (a URL fetched server-side, or pasted text / OpenAPI spec), asks the model to
author a RESTAPIConfig manifest, and validates it against the create-path checks — repairing
against validation errors up to a small budget. Returns the manifest for the user to review
and tweak in the builder before creating the source; it does NOT create anything. Gated by the
`dwh-custom-source-ai-builder` flag, and requires the org to have approved AI data processing,
since the docs are sent to the LLM gateway.
- **`posthog-pp-cli external-data-sources list`** - Create, Read, Update and Delete External data Sources.
- **`posthog-pp-cli external-data-sources oauth-accounts-retrieve`** - List the accounts/properties a connected OAuth integration exposes, in the shared
IntegrationAccount shape. The logic lives in each source (via OAuthMixin.get_oauth_accounts);
this endpoint just routes by source type, applies the optional search filter, and serializes.
- **`posthog-pp-cli external-data-sources partial-update`** - Create, Read, Update and Delete External data Sources.
- **`posthog-pp-cli external-data-sources preview-resource-create`** - Read a bounded sample of rows for one resource of a Custom REST source.

Lets a manifest author verify `data_selector`, `primary_key`, and the incremental
`cursor_path` against live data before creating the source. Only `source_type: "Custom"`
is supported — other source types return 400. The read is bounded (single page per
resource, capped row count, short timeouts, no redirects). Manifest, validation, and SSRF
problems return 400; a live fetch failure returns 200 with `error` set and empty `rows`.
- **`posthog-pp-cli external-data-sources retrieve`** - Create, Read, Update and Delete External data Sources.
- **`posthog-pp-cli external-data-sources setup-create`** - One-shot data warehouse source setup.

Validate credentials, discover available tables, enable them all with sensible sync defaults
(incremental where supported, else append, else full refresh), and create the source in a single
call — the caller never has to assemble a `schemas` array. For sources that support webhooks
(e.g. Stripe), a webhook is auto-registered after creation: on success webhook-capable tables
switch to real-time webhook sync (unlocking webhook-only tables); on failure the polling
defaults stay in place. For fine-grained table/sync control, use the lower-level
`database_schema` + `create` flow instead.
- **`posthog-pp-cli external-data-sources source-prefix-create`** - Create, Read, Update and Delete External data Sources.
- **`posthog-pp-cli external-data-sources store-credentials-create`** - Validate and store credentials for a data warehouse source without creating the source.

Backs the source connect page: the user enters credentials directly in PostHog, they are
checked against a live connection, then stashed encrypted in a temporary store. The returned
credential id can be passed to `setup` as {'credential_id': <id>} to create the source — so
secrets never travel through an agent conversation. The stash is single-use: it is deleted
as soon as `setup` consumes it, and expires after 24 hours if never consumed.
- **`posthog-pp-cli external-data-sources stored-credentials-list`** - List credentials the requesting user stored via the source connect page that haven't been consumed yet.

Returns metadata only (id, source type, timestamps) — never the secrets themselves. Stored
credentials are scoped to their creator: only the user who filled the connect page can list
or consume them. They are temporary too: they disappear once consumed by `setup` or when
they expire. Newest first, so after a user confirms they've finished the connect page, the
first entry for the source type is the one to pass to `setup`.
- **`posthog-pp-cli external-data-sources update`** - Create, Read, Update and Delete External data Sources.
- **`posthog-pp-cli external-data-sources wizard-retrieve`** - Create, Read, Update and Delete External data Sources.

### feature-flags

Manage feature flags

- **`posthog-pp-cli feature-flags all-activity-retrieve`** - Create, read, update and delete feature flags. [See docs](https://posthog.com/docs/feature-flags) for more information on feature flags.

If you're looking to use feature flags on your application, you can either use our JavaScript Library or our dedicated endpoint to check if feature flags are enabled for a given user.
- **`posthog-pp-cli feature-flags bulk-delete-create`** - Bulk delete feature flags by filter criteria or explicit IDs.

Accepts either:
- {"filters": {...}} - Same filter params as list endpoint (search, active, type, etc.)
- {"ids": [...]} - Explicit list of flag IDs (no limit)

Returns same format as bulk_delete for UI compatibility.

Uses bulk operations for efficiency: database updates are batched and cache
invalidation happens once at the end rather than per-flag.
- **`posthog-pp-cli feature-flags bulk-keys-retrieve`** - Get feature flag keys by IDs.
Accepts a list of feature flag IDs and returns a mapping of ID to key.
- **`posthog-pp-cli feature-flags bulk-update-tags-create`** - Bulk update tags on multiple objects.

PAT access: this action has no ``required_scopes=`` on the decorator —
inheriting viewsets must add ``"bulk_update_tags"`` to their
``scope_object_write_actions`` list to accept personal API keys.
Without that opt-in, ``APIScopePermission`` rejects PAT requests with
"This action does not support personal API key access". Done per-viewset
so granting ``<scope>:write`` for one resource doesn't leak access to
sibling resources that share this mixin.

Accepts:
- {"ids": [...], "action": "add"|"remove"|"set", "tags": ["tag1", "tag2"]}

Actions:
- "add": Add tags to existing tags on each object
- "remove": Remove specific tags from each object
- "set": Replace all tags on each object with the provided list
- **`posthog-pp-cli feature-flags create`** - Create, read, update and delete feature flags. [See docs](https://posthog.com/docs/feature-flags) for more information on feature flags.

If you're looking to use feature flags on your application, you can either use our JavaScript Library or our dedicated endpoint to check if feature flags are enabled for a given user.
- **`posthog-pp-cli feature-flags destroy`** - Hard delete of this model is not allowed. Use a patch API call to set "deleted" to true
- **`posthog-pp-cli feature-flags evaluation-reasons-retrieve`** - Create, read, update and delete feature flags. [See docs](https://posthog.com/docs/feature-flags) for more information on feature flags.

If you're looking to use feature flags on your application, you can either use our JavaScript Library or our dedicated endpoint to check if feature flags are enabled for a given user.
- **`posthog-pp-cli feature-flags list`** - Create, read, update and delete feature flags. [See docs](https://posthog.com/docs/feature-flags) for more information on feature flags.

If you're looking to use feature flags on your application, you can either use our JavaScript Library or our dedicated endpoint to check if feature flags are enabled for a given user.
- **`posthog-pp-cli feature-flags matching-ids-retrieve`** - Get IDs of all feature flags matching the current filters.
Uses the same filtering logic as the list endpoint.
Returns only IDs that the user has permission to edit.
- **`posthog-pp-cli feature-flags my-flags-retrieve`** - Create, read, update and delete feature flags. [See docs](https://posthog.com/docs/feature-flags) for more information on feature flags.

If you're looking to use feature flags on your application, you can either use our JavaScript Library or our dedicated endpoint to check if feature flags are enabled for a given user.
- **`posthog-pp-cli feature-flags partial-update`** - Create, read, update and delete feature flags. [See docs](https://posthog.com/docs/feature-flags) for more information on feature flags.

If you're looking to use feature flags on your application, you can either use our JavaScript Library or our dedicated endpoint to check if feature flags are enabled for a given user.
- **`posthog-pp-cli feature-flags retrieve`** - Create, read, update and delete feature flags. [See docs](https://posthog.com/docs/feature-flags) for more information on feature flags.

If you're looking to use feature flags on your application, you can either use our JavaScript Library or our dedicated endpoint to check if feature flags are enabled for a given user.
- **`posthog-pp-cli feature-flags update`** - Create, read, update and delete feature flags. [See docs](https://posthog.com/docs/feature-flags) for more information on feature flags.

If you're looking to use feature flags on your application, you can either use our JavaScript Library or our dedicated endpoint to check if feature flags are enabled for a given user.
- **`posthog-pp-cli feature-flags user-blast-radius-create`** - Create, read, update and delete feature flags. [See docs](https://posthog.com/docs/feature-flags) for more information on feature flags.

If you're looking to use feature flags on your application, you can either use our JavaScript Library or our dedicated endpoint to check if feature flags are enabled for a given user.

### field-notes

Manage field notes

- **`posthog-pp-cli field-notes create`** - Create, read, update, and resolve toolbar field notes — UI feedback a user
points at on their own site, surfaced to coding agents over MCP.
- **`posthog-pp-cli field-notes destroy`** - Create, read, update, and resolve toolbar field notes — UI feedback a user
points at on their own site, surfaced to coding agents over MCP.
- **`posthog-pp-cli field-notes list`** - Create, read, update, and resolve toolbar field notes — UI feedback a user
points at on their own site, surfaced to coding agents over MCP.
- **`posthog-pp-cli field-notes partial-update`** - Create, read, update, and resolve toolbar field notes — UI feedback a user
points at on their own site, surfaced to coding agents over MCP.
- **`posthog-pp-cli field-notes retrieve`** - Create, read, update, and resolve toolbar field notes — UI feedback a user
points at on their own site, surfaced to coding agents over MCP.
- **`posthog-pp-cli field-notes update`** - Create, read, update, and resolve toolbar field notes — UI feedback a user
points at on their own site, surfaced to coding agents over MCP.

### file-download-batch-exports

Manage file download batch exports

- **`posthog-pp-cli file-download-batch-exports create`** - Create and start a batch export on demand run to download a file.
- **`posthog-pp-cli file-download-batch-exports list`** - List
- **`posthog-pp-cli file-download-batch-exports retrieve`** - Get a batch export on demand run.

If the underlying batch export run has completed, we return keys to the
generated file downloads so that users may download them by making a request
to /download.

### file-system

Manage file system

- **`posthog-pp-cli file-system count-by-path-create`** - Get count of all files in a folder.
- **`posthog-pp-cli file-system create`** - Create
- **`posthog-pp-cli file-system destroy`** - Destroy
- **`posthog-pp-cli file-system list`** - List
- **`posthog-pp-cli file-system log-view-create`** - Log view create
- **`posthog-pp-cli file-system log-view-retrieve`** - Log view retrieve
- **`posthog-pp-cli file-system partial-update`** - Partial update
- **`posthog-pp-cli file-system retrieve`** - Retrieve
- **`posthog-pp-cli file-system undo-delete-create`** - Undo delete create
- **`posthog-pp-cli file-system unfiled-retrieve`** - Unfiled retrieve
- **`posthog-pp-cli file-system update`** - Update

### file-system-shortcut

Manage file system shortcut

- **`posthog-pp-cli file-system-shortcut create`** - Create
- **`posthog-pp-cli file-system-shortcut destroy`** - Destroy
- **`posthog-pp-cli file-system-shortcut list`** - List
- **`posthog-pp-cli file-system-shortcut partial-update`** - Partial update
- **`posthog-pp-cli file-system-shortcut reorder-create`** - Set the display order of the current user's shortcuts. `ordered_ids` becomes the new top-to-bottom order; any unknown IDs are rejected.
- **`posthog-pp-cli file-system-shortcut retrieve`** - Retrieve
- **`posthog-pp-cli file-system-shortcut update`** - Update

### flag-value

Manage flag value

- **`posthog-pp-cli flag-value <project_id>`** - Get possible values for a feature flag.

Query parameters:
- key: The flag ID (required)
Returns:

- Array of objects with 'name' field containing possible values

### groups

Manage groups

- **`posthog-pp-cli groups activity-retrieve`** - Activity retrieve
- **`posthog-pp-cli groups create`** - Create
- **`posthog-pp-cli groups delete-property-create`** - Delete property create
- **`posthog-pp-cli groups find-retrieve`** - Find retrieve
- **`posthog-pp-cli groups list`** - List all groups of a specific group type. You must pass ?group_type_index= in the URL.
To get a list of valid group types, call /api/:project_id/groups_types/.

Uses forward-only keyset pagination via the `cursor` parameter.
The `previous` field in the response envelope is always null.
- **`posthog-pp-cli groups property-values-retrieve`** - Property values retrieve
- **`posthog-pp-cli groups related-retrieve`** - Related retrieve
- **`posthog-pp-cli groups update-property-create`** - Update property create

### groups-types

Manage groups types

- **`posthog-pp-cli groups-types create-detail-dashboard-update`** - Create detail dashboard update
- **`posthog-pp-cli groups-types destroy`** - Destroy
- **`posthog-pp-cli groups-types list`** - List
- **`posthog-pp-cli groups-types set-default-columns-update`** - Set default columns update
- **`posthog-pp-cli groups-types update-metadata-partial-update`** - Update metadata partial update

### health-issues

Manage health issues

- **`posthog-pp-cli health-issues list`** - Lists health issues detected across all of this project's PostHog health checks (outdated SDKs, data warehouse sync failures, missing web analytics events, ingestion warnings, and more). Filter by status, severity, kind, or dismissed state.
- **`posthog-pp-cli health-issues partial-update`** - Partial update
- **`posthog-pp-cli health-issues refresh-create`** - Refresh create
- **`posthog-pp-cli health-issues retrieve`** - Fetches a single health issue, enriched with the owning check's rendered explanation: a title, a one-line summary of what's wrong, a deep link to the relevant page, and remediation guidance for how to fix it.
- **`posthog-pp-cli health-issues summary-retrieve`** - Returns aggregated counts of active, non-dismissed health issues for the project, broken down by severity and by kind. Use for a quick overview of overall project health before drilling in with the list endpoint.

### heatmap-screenshots

Manage heatmap screenshots


### heatmaps

Manage heatmaps

- **`posthog-pp-cli heatmaps events-retrieve`** - Drill into the individual session interactions behind one or more heatmap coordinates. Pass the 'points' you want to inspect (from the heatmaps list response) to get the underlying per-session events, so you can jump to the session recordings that produced a hotspot.
- **`posthog-pp-cli heatmaps list`** - Aggregated heatmap interactions for a page. For type 'click'/'rageclick'/'mousemove' each result is a point with relative x, absolute client-y, and a count. For type 'scrolldepth' the response is scroll-depth buckets instead (cumulative reach down the page).

### hog-flows

Manage hog flows

- **`posthog-pp-cli hog-flows bulk-delete-create`** - Bulk delete create
- **`posthog-pp-cli hog-flows create`** - Create
- **`posthog-pp-cli hog-flows destroy`** - Destroy
- **`posthog-pp-cli hog-flows email-sending-suspension-retrieve`** - Cheap read for the scene-wide suspension banner: single-row `TeamWorkflowsConfig` lookup
with no reputation computation. Every project member sees this — a suspension stops
everyone's email, so hiding it would leave silent send failures unexplained.
- **`posthog-pp-cli hog-flows list`** - List
- **`posthog-pp-cli hog-flows metrics-global-retrieve`** - Metrics global retrieve
- **`posthog-pp-cli hog-flows partial-update`** - Partial update
- **`posthog-pp-cli hog-flows reputation-retrieve`** - Bounce/complaint rates for this project's workflow email over the last 30 days, computed on
the fly from app metrics (a project-wide aggregate plus per-workflow rows, worst first,
capped), together with the authoritative AWS SES tenant verdict — sending status and open
reputation findings. Our rates are the per-workflow diagnosis; AWS judges and enforces.
- **`posthog-pp-cli hog-flows retrieve`** - Retrieve
- **`posthog-pp-cli hog-flows update`** - Update
- **`posthog-pp-cli hog-flows user-blast-radius-create`** - User blast radius create

### hog-function-templates

Manage hog function templates

- **`posthog-pp-cli hog-function-templates list`** - List
- **`posthog-pp-cli hog-function-templates retrieve`** - Retrieve

### hog-functions

Manage hog functions

- **`posthog-pp-cli hog-functions create`** - Create
- **`posthog-pp-cli hog-functions destroy`** - Hard delete of this model is not allowed. Use a patch API call to set "deleted" to true
- **`posthog-pp-cli hog-functions icon-retrieve`** - Icon retrieve
- **`posthog-pp-cli hog-functions icons-retrieve`** - Icons retrieve
- **`posthog-pp-cli hog-functions list`** - List
- **`posthog-pp-cli hog-functions partial-update`** - Partial update
- **`posthog-pp-cli hog-functions rearrange-partial-update`** - Update the execution order of multiple HogFunctions.
- **`posthog-pp-cli hog-functions retrieve`** - Retrieve
- **`posthog-pp-cli hog-functions update`** - Update

### ingestion-warnings-v2

Manage ingestion warnings v2

- **`posthog-pp-cli ingestion-warnings-v2 <project_id>`** - Lists this project's ingestion warnings — events or person/group updates that were ingested with problems (oversized messages, rejected person merges, invalid data) — grouped by warning type. Each entry carries the warning's category and severity, the total count and a sparkline over the requested time range, and the most recent sample warnings with the affected event/person/group. Filter by category, type, severity or time range to drill into a specific problem.

### insight-variables

Manage insight variables

- **`posthog-pp-cli insight-variables create`** - Create
- **`posthog-pp-cli insight-variables destroy`** - Destroy
- **`posthog-pp-cli insight-variables list`** - List
- **`posthog-pp-cli insight-variables partial-update`** - Partial update
- **`posthog-pp-cli insight-variables retrieve`** - Retrieve
- **`posthog-pp-cli insight-variables update`** - Update

### insights

Manage insights

- **`posthog-pp-cli insights all-activity-retrieve`** - Project-wide audit trail across all insights — who created, edited, deleted, or restored insights, what changed (with before/after diffs), and when. Useful for surfacing what people (or agents) have been working on recently.
- **`posthog-pp-cli insights bulk-delete-create`** - Soft-delete insights in bulk by ID. Mirrors the single-insight delete: sets deleted=True, soft-deletes the insights' dashboard tiles, and removes their linked alerts. Insights the requester cannot edit are skipped and reported in `skipped`. Reversible via the bulk_restore endpoint.
- **`posthog-pp-cli insights bulk-restore-create`** - Restore soft-deleted insights in bulk by ID — the inverse of bulk_delete. Sets deleted=False and re-activates the insights' dashboard tiles on dashboards that still exist. Linked alerts are not restored (they are removed on delete). Insights the requester cannot edit are reported in `skipped`.
- **`posthog-pp-cli insights bulk-update-tags-create`** - Bulk update tags on multiple objects.

PAT access: this action has no ``required_scopes=`` on the decorator —
inheriting viewsets must add ``"bulk_update_tags"`` to their
``scope_object_write_actions`` list to accept personal API keys.
Without that opt-in, ``APIScopePermission`` rejects PAT requests with
"This action does not support personal API key access". Done per-viewset
so granting ``<scope>:write`` for one resource doesn't leak access to
sibling resources that share this mixin.

Accepts:
- {"ids": [...], "action": "add"|"remove"|"set", "tags": ["tag1", "tag2"]}

Actions:
- "add": Add tags to existing tags on each object
- "remove": Remove specific tags from each object
- "set": Replace all tags on each object with the provided list
- **`posthog-pp-cli insights cancel-create`** - DRF ViewSet mixin that gates coalesced responses behind permission checks.

The QueryCoalescingMiddleware attaches cached response data to
request.META["_coalesced_response"] for followers. This mixin runs DRF's
initial() (auth + permissions + throttling) before returning the
cached response, ensuring the request is authorized.
- **`posthog-pp-cli insights create`** - DRF ViewSet mixin that gates coalesced responses behind permission checks.

The QueryCoalescingMiddleware attaches cached response data to
request.META["_coalesced_response"] for followers. This mixin runs DRF's
initial() (auth + permissions + throttling) before returning the
cached response, ensuring the request is authorized.
- **`posthog-pp-cli insights destroy`** - Hard delete of this model is not allowed. Use a patch API call to set "deleted" to true
- **`posthog-pp-cli insights generate-metadata-create`** - Generate an AI-suggested name and description for an insight based on its query configuration.
- **`posthog-pp-cli insights list`** - DRF ViewSet mixin that gates coalesced responses behind permission checks.

The QueryCoalescingMiddleware attaches cached response data to
request.META["_coalesced_response"] for followers. This mixin runs DRF's
initial() (auth + permissions + throttling) before returning the
cached response, ensuring the request is authorized.
- **`posthog-pp-cli insights my-last-viewed-retrieve`** - Returns basic details about the last 5 insights viewed by this user. Most recently viewed first.
- **`posthog-pp-cli insights partial-update`** - DRF ViewSet mixin that gates coalesced responses behind permission checks.

The QueryCoalescingMiddleware attaches cached response data to
request.META["_coalesced_response"] for followers. This mixin runs DRF's
initial() (auth + permissions + throttling) before returning the
cached response, ensuring the request is authorized.
- **`posthog-pp-cli insights retrieve`** - DRF ViewSet mixin that gates coalesced responses behind permission checks.

The QueryCoalescingMiddleware attaches cached response data to
request.META["_coalesced_response"] for followers. This mixin runs DRF's
initial() (auth + permissions + throttling) before returning the
cached response, ensuring the request is authorized.
- **`posthog-pp-cli insights trending-retrieve`** - Returns insights ranked by view count over the last N days (default 7), highest first. Each result includes the same metadata as the standard insights list, plus a `view_count` and up to 3 recent `viewers`. Useful for surfacing the most-used insights in a project.
- **`posthog-pp-cli insights update`** - DRF ViewSet mixin that gates coalesced responses behind permission checks.

The QueryCoalescingMiddleware attaches cached response data to
request.META["_coalesced_response"] for followers. This mixin runs DRF's
initial() (auth + permissions + throttling) before returning the
cached response, ensuring the request is authorized.
- **`posthog-pp-cli insights viewed-create`** - Record that the current user has just viewed one or more insights. Submitted ids that do not belong to the current project or that point at deleted insights are silently dropped. Returns 201 on success regardless of how many ids were retained.

### integrations

Manage integrations

- **`posthog-pp-cli integrations authorize-retrieve`** - Authorize retrieve
- **`posthog-pp-cli integrations create`** - Create
- **`posthog-pp-cli integrations destroy`** - Destroy
- **`posthog-pp-cli integrations domain-connect-apply-url-create`** - Unified endpoint for generating Domain Connect apply URLs.

Accepts a context ("email" or "proxy") and the relevant resource ID.
The backend resolves the domain, template variables, and service ID
based on context, then builds the signed apply URL.
- **`posthog-pp-cli integrations domain-connect-check-retrieve`** - Domain connect check retrieve
- **`posthog-pp-cli integrations github-available-installations-retrieve`** - List the org's existing GitHub installations this project can reuse.

A GitHub App installs once per organization, so a second project links an existing
installation rather than reinstalling. This backs the picker: when the org has more than
one installation, the client passes the chosen installation_id to github/link_existing.
- **`posthog-pp-cli integrations github-link-existing-create`** - Reuse a GitHub installation already linked to a sibling team in the same organization.
- **`posthog-pp-cli integrations github-oauth-authorize-create`** - Mint a User OAuth URL to bootstrap a fresh `code` when the install flow returns without one.
- **`posthog-pp-cli integrations github-prepare-callback-create`** - Seed GitHub setup callback state without redirecting to GitHub.

Used when the user opens an existing installation's settings on github.com (e.g. PostHog
Code "Update in GitHub") so the subsequent Setup URL redirect can be validated.
- **`posthog-pp-cli integrations list`** - List
- **`posthog-pp-cli integrations request-access-create`** - Notify project admins that a member is requesting an integration be connected.
- **`posthog-pp-cli integrations retrieve`** - Retrieve

### js-snippet

Manage js snippet

- **`posthog-pp-cli js-snippet resolve-retrieve`** - Preview what a given pin would resolve to, without saving it.
- **`posthog-pp-cli js-snippet version-partial-update`** - Update the team's version pin.
- **`posthog-pp-cli js-snippet version-retrieve`** - Return the team's current version pin and resolved version.

### live-debugger-breakpoints

Manage live debugger breakpoints

- **`posthog-pp-cli live-debugger-breakpoints active-retrieve`** - External API endpoint for client applications to fetch active breakpoints using Project API key. This endpoint allows external client applications (like Python scripts, Node.js apps, etc.) to fetch the list of active breakpoints so they can instrument their code accordingly. 

Authentication: Requires a Project API Key in the Authorization header: `Authorization: Bearer phs_<your-project-api-key>`. You can find your Project API Key in PostHog at: Settings → Project → Project API Key
- **`posthog-pp-cli live-debugger-breakpoints breakpoint-hits-retrieve`** - Retrieve breakpoint hit events from ClickHouse with optional filtering and pagination. Returns hit events containing stack traces, local variables, and execution context from your application's runtime. 

Security: Breakpoint IDs are filtered to only include those belonging to the current team.
- **`posthog-pp-cli live-debugger-breakpoints create`** - Create, Read, Update and Delete breakpoints for live debugging.
- **`posthog-pp-cli live-debugger-breakpoints destroy`** - Create, Read, Update and Delete breakpoints for live debugging.
- **`posthog-pp-cli live-debugger-breakpoints list`** - Create, Read, Update and Delete breakpoints for live debugging.
- **`posthog-pp-cli live-debugger-breakpoints partial-update`** - Create, Read, Update and Delete breakpoints for live debugging.
- **`posthog-pp-cli live-debugger-breakpoints retrieve`** - Create, Read, Update and Delete breakpoints for live debugging.
- **`posthog-pp-cli live-debugger-breakpoints update`** - Create, Read, Update and Delete breakpoints for live debugging.

### llm-analytics

Manage llm analytics

- **`posthog-pp-cli llm-analytics clustering-config-list`** - Team-level clustering configuration (event filters for automated pipelines).
- **`posthog-pp-cli llm-analytics clustering-config-set-event-filters-create`** - Team-level clustering configuration (event filters for automated pipelines).
- **`posthog-pp-cli llm-analytics clustering-jobs-create`** - CRUD for clustering job configurations (max 10 per team).
- **`posthog-pp-cli llm-analytics clustering-jobs-destroy`** - CRUD for clustering job configurations (max 10 per team).
- **`posthog-pp-cli llm-analytics clustering-jobs-list`** - CRUD for clustering job configurations (max 10 per team).
- **`posthog-pp-cli llm-analytics clustering-jobs-partial-update`** - CRUD for clustering job configurations (max 10 per team).
- **`posthog-pp-cli llm-analytics clustering-jobs-retrieve`** - CRUD for clustering job configurations (max 10 per team).
- **`posthog-pp-cli llm-analytics clustering-jobs-update`** - CRUD for clustering job configurations (max 10 per team).
- **`posthog-pp-cli llm-analytics clustering-runs-create`** - Trigger a new clustering workflow run.

This endpoint validates the request parameters and starts a Temporal workflow
to perform trace clustering with the specified configuration.
- **`posthog-pp-cli llm-analytics evaluation-config-retrieve`** - Get the evaluation config for this team
- **`posthog-pp-cli llm-analytics evaluation-config-set-active-key-create`** - Set the active provider key for evaluations
- **`posthog-pp-cli llm-analytics evaluation-reports-create`** - CRUD for evaluation report configurations + report run history.
- **`posthog-pp-cli llm-analytics evaluation-reports-destroy`** - Evaluation report configs are deleted only when their evaluation is deleted. Use PATCH enabled=false to stop delivery.
- **`posthog-pp-cli llm-analytics evaluation-reports-generate-create`** - Trigger immediate report generation.
- **`posthog-pp-cli llm-analytics evaluation-reports-list`** - CRUD for evaluation report configurations + report run history.
- **`posthog-pp-cli llm-analytics evaluation-reports-partial-update`** - CRUD for evaluation report configurations + report run history.
- **`posthog-pp-cli llm-analytics evaluation-reports-retrieve`** - CRUD for evaluation report configurations + report run history.
- **`posthog-pp-cli llm-analytics evaluation-reports-runs-list`** - List report runs (history) for this report.
- **`posthog-pp-cli llm-analytics evaluation-reports-update`** - CRUD for evaluation report configurations + report run history.
- **`posthog-pp-cli llm-analytics evaluation-summary-create`** - Generate an AI-powered summary of evaluation results.

This endpoint analyzes evaluation runs and identifies patterns in passing
and failing evaluations, providing actionable recommendations.

Data is fetched server-side by evaluation ID to ensure data integrity.

**Use Cases:**
- Understand why evaluations are passing or failing
- Identify systematic issues in LLM responses
- Get recommendations for improving response quality
- Review patterns across many evaluation runs at once
- **`posthog-pp-cli llm-analytics models-retrieve`** - List available models for a provider.
- **`posthog-pp-cli llm-analytics offline-evaluations-experiment-items-create`** - Offline evaluations experiment items create
- **`posthog-pp-cli llm-analytics parser-recipes-create`** - Parser recipes create
- **`posthog-pp-cli llm-analytics parser-recipes-destroy`** - Parser recipes destroy
- **`posthog-pp-cli llm-analytics parser-recipes-list`** - Parser recipes list
- **`posthog-pp-cli llm-analytics parser-recipes-partial-update`** - Parser recipes partial update
- **`posthog-pp-cli llm-analytics parser-recipes-retrieve`** - Parser recipes retrieve
- **`posthog-pp-cli llm-analytics personal-spend-list`** - Return a structured personal LLM spend analysis for the requesting user. Pass `date_from` / `date_to` (absolute like `2026-04-23` or relative like `-7d`) to bound the window — defaults to the last 30 days, max 90 days. The `product=<ai_product>` query param is required and scopes the tool / model / day / trace breakdowns to a single product; supported values: posthog_code. `by_product` is always returned for cross-product visibility. `by_day` returns a day-ascending spend series for the scoped product. Pass `bucket_minutes` (5, 15, 30, or 60; the window may span at most 600 buckets) to additionally get `by_bucket`, a time-ascending series with per-bucket cost split into uncached input / output / cache read / cache creation components. Use `refresh=true` to bypass the 5-minute response cache.
- **`posthog-pp-cli llm-analytics provider-key-validations-create`** - Validate LLM provider API keys without persisting them
- **`posthog-pp-cli llm-analytics provider-keys-create`** - Provider keys create
- **`posthog-pp-cli llm-analytics provider-keys-dependent-configs-retrieve`** - Get evaluations using this key and alternative keys for replacement.
- **`posthog-pp-cli llm-analytics provider-keys-destroy`** - Provider keys destroy
- **`posthog-pp-cli llm-analytics provider-keys-list`** - Provider keys list
- **`posthog-pp-cli llm-analytics provider-keys-partial-update`** - Provider keys partial update
- **`posthog-pp-cli llm-analytics provider-keys-retrieve`** - Provider keys retrieve
- **`posthog-pp-cli llm-analytics provider-keys-update`** - Provider keys update
- **`posthog-pp-cli llm-analytics provider-keys-validate-create`** - Provider keys validate create
- **`posthog-pp-cli llm-analytics review-queue-items-create`** - Review queue items create
- **`posthog-pp-cli llm-analytics review-queue-items-destroy`** - Review queue items destroy
- **`posthog-pp-cli llm-analytics review-queue-items-list`** - Review queue items list
- **`posthog-pp-cli llm-analytics review-queue-items-partial-update`** - Review queue items partial update
- **`posthog-pp-cli llm-analytics review-queue-items-retrieve`** - Review queue items retrieve
- **`posthog-pp-cli llm-analytics review-queues-create`** - Review queues create
- **`posthog-pp-cli llm-analytics review-queues-destroy`** - Review queues destroy
- **`posthog-pp-cli llm-analytics review-queues-list`** - Review queues list
- **`posthog-pp-cli llm-analytics review-queues-partial-update`** - Review queues partial update
- **`posthog-pp-cli llm-analytics review-queues-retrieve`** - Review queues retrieve
- **`posthog-pp-cli llm-analytics score-definitions-create`** - Score definitions create
- **`posthog-pp-cli llm-analytics score-definitions-list`** - Score definitions list
- **`posthog-pp-cli llm-analytics score-definitions-new-version-create`** - Score definitions new version create
- **`posthog-pp-cli llm-analytics score-definitions-partial-update`** - Score definitions partial update
- **`posthog-pp-cli llm-analytics score-definitions-retrieve`** - Score definitions retrieve
- **`posthog-pp-cli llm-analytics summarization-batch-check-create`** - Check which traces have cached summaries available.

This endpoint allows batch checking of multiple trace IDs to see which ones
have cached summaries. Returns only the traces that have cached summaries
with their titles.

**Use Cases:**
- Load cached summaries on session view load
- Avoid unnecessary LLM calls for already-summarized traces
- Display summary previews without generating new summaries
- **`posthog-pp-cli llm-analytics summarization-create`** - Generate an AI-powered summary of an LLM trace or event.

This endpoint analyzes the provided trace/event, generates a line-numbered text
representation, and uses an LLM to create a concise summary with line references.

**Two ways to use this endpoint:**

1. **By ID (recommended):** Pass `trace_id` or `generation_id` with an optional `date_from`/`date_to`.
   The backend fetches the data automatically. `summarize_type` is inferred.
2. **By data:** Pass the full trace/event data blob in `data` with `summarize_type`.
   This is how the frontend uses it.

**Summary Format:**
- Title (concise, max 10 words)
- Mermaid flow diagram showing the main flow
- 3-10 summary bullets with line references
- "Interesting Notes" section for failures, successes, or unusual patterns
- Line references in [L45] or [L45-52] format pointing to relevant sections

The response includes the structured summary, the text representation, and metadata.
- **`posthog-pp-cli llm-analytics text-repr-create`** - Generate a human-readable text representation of an LLM trace event.

This endpoint converts AI observability events ($ai_generation, $ai_span, $ai_embedding, or $ai_trace)
into formatted text representations suitable for display, logging, or analysis.

**Supported Event Types:**
- `$ai_generation`: Individual LLM API calls with input/output messages
- `$ai_span`: Logical spans with state transitions
- `$ai_embedding`: Embedding generation events (text input → vector)
- `$ai_trace`: Full traces with hierarchical structure

**Options:**
- `max_length`: Maximum character count (default: 2000000)
- `truncated`: Enable middle-content truncation within events (default: true)
- `truncate_buffer`: Characters at start/end when truncating (default: 1000)
- `include_markers`: Use interactive markers vs plain text indicators (default: true)
  - Frontend: set true for `<<<TRUNCATED|base64|...>>>` markers
  - Backend/LLM: set false for `... (X chars truncated) ...` text
- `collapsed`: Show summary vs full trace tree (default: false)
- `include_hierarchy`: Include tree structure for traces (default: true)
- `max_depth`: Maximum depth for hierarchical rendering (default: unlimited)
- `tools_collapse_threshold`: Number of tools before auto-collapsing list (default: 5)
  - Tool lists >5 items show `<<<TOOLS_EXPANDABLE|...>>>` marker for frontend
  - Or `[+] AVAILABLE TOOLS: N` for backend when `include_markers: false`
- `include_line_numbers`: Prefix each line with line number like L001:, L010: (default: false)

**Use Cases:**
- Frontend display: `truncated: true, include_markers: true, include_line_numbers: true`
- Backend LLM context (summary): `truncated: true, include_markers: false, collapsed: true`
- Backend LLM context (full): `truncated: false`

The response includes the formatted text and metadata about the rendering.
- **`posthog-pp-cli llm-analytics trace-reviews-create`** - Trace reviews create
- **`posthog-pp-cli llm-analytics trace-reviews-destroy`** - Trace reviews destroy
- **`posthog-pp-cli llm-analytics trace-reviews-list`** - Trace reviews list
- **`posthog-pp-cli llm-analytics trace-reviews-partial-update`** - Trace reviews partial update
- **`posthog-pp-cli llm-analytics trace-reviews-retrieve`** - Trace reviews retrieve
- **`posthog-pp-cli llm-analytics translate-create`** - Translate text to target language.

### llm-prompts

Manage llm prompts

- **`posthog-pp-cli llm-prompts create`** - Create
- **`posthog-pp-cli llm-prompts list`** - List
- **`posthog-pp-cli llm-prompts name-archive-create`** - Name archive create
- **`posthog-pp-cli llm-prompts name-duplicate-create`** - Name duplicate create
- **`posthog-pp-cli llm-prompts name-labels-destroy`** - Name labels destroy
- **`posthog-pp-cli llm-prompts name-labels-update`** - Name labels update
- **`posthog-pp-cli llm-prompts name-partial-update`** - Name partial update
- **`posthog-pp-cli llm-prompts name-retrieve`** - Name retrieve
- **`posthog-pp-cli llm-prompts resolve-name-retrieve`** - Resolve name retrieve

### llm-skills

Manage llm skills

- **`posthog-pp-cli llm-skills create`** - Create
- **`posthog-pp-cli llm-skills import-create`** - Import create
- **`posthog-pp-cli llm-skills list`** - List
- **`posthog-pp-cli llm-skills marketplace-install-command-create`** - Mint the user's read-only marketplace credential (or rotate it) and return the install command.

Per-user: rotating only ever invalidates this user's own credential, never a teammate's.
- **`posthog-pp-cli llm-skills marketplace-install-command-retrieve`** - Report whether the user already has a marketplace credential, without minting one.

The token is unrecoverable, so an existing credential returns its mask only — the UI shows
"already connected, existing setups keep working" and offers an explicit rotate.
- **`posthog-pp-cli llm-skills name-archive-create`** - Name archive create
- **`posthog-pp-cli llm-skills name-duplicate-create`** - Name duplicate create
- **`posthog-pp-cli llm-skills name-export-retrieve`** - Name export retrieve
- **`posthog-pp-cli llm-skills name-files-create`** - Name files create
- **`posthog-pp-cli llm-skills name-files-destroy`** - Name files destroy
- **`posthog-pp-cli llm-skills name-files-rename-create`** - Name files rename create
- **`posthog-pp-cli llm-skills name-files-retrieve`** - Name files retrieve
- **`posthog-pp-cli llm-skills name-partial-update`** - Name partial update
- **`posthog-pp-cli llm-skills name-retrieve`** - Name retrieve
- **`posthog-pp-cli llm-skills resolve-name-retrieve`** - Resolve name retrieve

### logs

Manage logs

- **`posthog-pp-cli logs alerts-create`** - Alerts create
- **`posthog-pp-cli logs alerts-destinations-create`** - Create a notification destination for this alert. One HogFunction is created per alert event kind (firing, resolved, ...) atomically.
- **`posthog-pp-cli logs alerts-destinations-delete-create`** - Delete a notification destination by deleting its HogFunction group atomically.
- **`posthog-pp-cli logs alerts-destroy`** - Alerts destroy
- **`posthog-pp-cli logs alerts-events-list`** - Paginated event history for this alert, newest first. Returns state transitions, errored checks, and user-initiated control-plane rows (reset, enable/disable, snooze/unsnooze, threshold change) — quiet no-op check rows (where state didn't change and there was no error) are filtered out since only the last 10 are kept and they carry no forensic value. Optional `?kind=...` narrows to a single kind.
- **`posthog-pp-cli logs alerts-list`** - Alerts list
- **`posthog-pp-cli logs alerts-partial-update`** - Alerts partial update
- **`posthog-pp-cli logs alerts-reset-create`** - Reset a broken alert. Clears the consecutive-failure counter and schedules an immediate recheck.
- **`posthog-pp-cli logs alerts-retrieve`** - Alerts retrieve
- **`posthog-pp-cli logs alerts-simulate-create`** - Simulate a logs alert on historical data using the full state machine. Read-only — no alert check records are created.
- **`posthog-pp-cli logs alerts-update`** - Alerts update
- **`posthog-pp-cli logs anomalies-scan-create`** - Runs anomaly detection on demand over one service's log volume for the given window. Learns per severity baselines from up to 6 weeks of history and returns per bucket expected bands plus any spike, drop, or silence issues. Synchronous and read only.
- **`posthog-pp-cli logs attributes-retrieve`** - Attributes retrieve
- **`posthog-pp-cli logs count-create`** - Count create
- **`posthog-pp-cli logs count-ranges-create`** - Count ranges create
- **`posthog-pp-cli logs explain-with-ai-create`** - Explain a log entry using AI.

POST /api/environments/:id/logs/explainLogWithAI/
- **`posthog-pp-cli logs export-create`** - Export create
- **`posthog-pp-cli logs facet-values-create`** - Facet values create
- **`posthog-pp-cli logs group-by-create`** - Group by create
- **`posthog-pp-cli logs has-retrieve`** - Has retrieve
- **`posthog-pp-cli logs metric-rules-create`** - Metric rules create
- **`posthog-pp-cli logs metric-rules-destroy`** - Metric rules destroy
- **`posthog-pp-cli logs metric-rules-list`** - Metric rules list
- **`posthog-pp-cli logs metric-rules-partial-update`** - Metric rules partial update
- **`posthog-pp-cli logs metric-rules-retrieve`** - Metric rules retrieve
- **`posthog-pp-cli logs metric-rules-update`** - Metric rules update
- **`posthog-pp-cli logs patterns-create`** - Patterns create
- **`posthog-pp-cli logs patterns-diff-create`** - Patterns diff create
- **`posthog-pp-cli logs query-create`** - Query create
- **`posthog-pp-cli logs retention-rules-create`** - Retention rules create
- **`posthog-pp-cli logs retention-rules-destroy`** - Retention rules destroy
- **`posthog-pp-cli logs retention-rules-list`** - Retention rules list
- **`posthog-pp-cli logs retention-rules-partial-update`** - Retention rules partial update
- **`posthog-pp-cli logs retention-rules-reorder-create`** - Atomically reassign priorities so the given ID order maps to ascending priorities (0..n-1).
- **`posthog-pp-cli logs retention-rules-retrieve`** - Retention rules retrieve
- **`posthog-pp-cli logs retention-rules-suggest-name-create`** - Suggest a human-readable name for a retention rule from its retention tier and filter group. Used by the create form as an auto-suggest; nothing is persisted. Returns an empty name when a suggestion can't be generated.
- **`posthog-pp-cli logs retention-rules-update`** - Retention rules update
- **`posthog-pp-cli logs sampling-rules-create`** - Sampling rules create
- **`posthog-pp-cli logs sampling-rules-destroy`** - Sampling rules destroy
- **`posthog-pp-cli logs sampling-rules-list`** - Sampling rules list
- **`posthog-pp-cli logs sampling-rules-partial-update`** - Sampling rules partial update
- **`posthog-pp-cli logs sampling-rules-reorder-create`** - Atomically reassign priorities so the given ID order maps to ascending priorities (0..n-1).
- **`posthog-pp-cli logs sampling-rules-retrieve`** - Sampling rules retrieve
- **`posthog-pp-cli logs sampling-rules-simulate-create`** - Dry-run estimate for how much volume this rule would remove (placeholder response until CH-backed simulation is wired).
- **`posthog-pp-cli logs sampling-rules-update`** - Sampling rules update
- **`posthog-pp-cli logs services-create`** - Services create
- **`posthog-pp-cli logs sparkline-create`** - Sparkline create
- **`posthog-pp-cli logs values-retrieve`** - Values retrieve
- **`posthog-pp-cli logs views-create`** - Views create
- **`posthog-pp-cli logs views-destroy`** - Views destroy
- **`posthog-pp-cli logs views-list`** - Views list
- **`posthog-pp-cli logs views-partial-update`** - Views partial update
- **`posthog-pp-cli logs views-retrieve`** - Views retrieve
- **`posthog-pp-cli logs views-update`** - Views update

### loops

Manage loops

- **`posthog-pp-cli loops create`** - API for managing loops — named, cloud-executed agent automations triggered by
schedule, GitHub events or authenticated API calls. See `products/tasks/docs/LOOPS.md`.
- **`posthog-pp-cli loops destroy`** - Soft delete. Pauses every trigger's schedule. Owner or a project admin only.
- **`posthog-pp-cli loops list`** - List loops visible to the caller: personal loops they own, plus every team loop. The response also carries `max_loops_per_team` and `total_loop_count` so a client can show remaining capacity and disable creation at the cap without hardcoding the limit.
- **`posthog-pp-cli loops partial-update`** - Partial update. Identity-bearing fields (instructions, repositories, connectors, behaviors, model config, triggers) are owner-only on team loops; name, description, notifications and enable/pause are editable by any team member.
- **`posthog-pp-cli loops retrieve`** - API for managing loops — named, cloud-executed agent automations triggered by
schedule, GitHub events or authenticated API calls. See `products/tasks/docs/LOOPS.md`.

### managed-viewsets

Manage managed viewsets

- **`posthog-pp-cli managed-viewsets retrieve`** - Get all views associated with a specific managed viewset.
GET /api/environments/{team_id}/managed_viewsets/{kind}/
- **`posthog-pp-cli managed-viewsets update`** - Enable or disable a managed viewset by kind.
PUT /api/environments/{team_id}/managed_viewsets/{kind}/ with body {"enabled": true/false}

### marketing-analytics

Manage marketing analytics

- **`posthog-pp-cli marketing-analytics conversion-goals-retrieve`** - Read the configured conversion goals for the current project — each with its kind, target, last-30d count, integrated vs non-integrated split, and a misconfiguration flag. Read-only.
- **`posthog-pp-cli marketing-analytics data-sources-retrieve`** - Check the platform → data-warehouse side of every native marketing integration: connection state, sync recency, row counts, required-table status, and schema-mapping coverage. Read-only.
- **`posthog-pp-cli marketing-analytics diagnose-retrieve`** - Aggregate data-source sync health, UTM attribution health, and conversion-goal config into a single per-integration diagnostic with recommended actions. Read-only.
- **`posthog-pp-cli marketing-analytics explain-conversion-goal-retrieve`** - Break down a single conversion goal's events over a period by event name, utm_source, and matched integration, with a small sample of events. Read-only.
- **`posthog-pp-cli marketing-analytics suggest-conversion-goals-retrieve`** - Rank existing custom events as conversion-goal candidates by volume, UTM-tag coverage, and unique users, excluding system/autocaptured events. Read-only.
- **`posthog-pp-cli marketing-analytics suggest-utm-mappings-retrieve`** - Detect unmatched utm_source values from recent events and propose custom_source_mappings entries, alongside the full utm_source catalogue and current mappings. Read-only.
- **`posthog-pp-cli marketing-analytics test-mapping-create`** - Test mapping create
- **`posthog-pp-cli marketing-analytics utm-audit-retrieve`** - Cross-reference campaigns with spend from ad platforms against pageview events with UTM parameters to identify tracking issues.

### max-tools

Manage max tools

- **`posthog-pp-cli max-tools <project_id>`** - Create and query insight create

### mcp-analytics

Manage mcp analytics

- **`posthog-pp-cli mcp-analytics feedback-create`** - Create a new MCP feedback submission for the current project.
- **`posthog-pp-cli mcp-analytics feedback-list`** - List MCP feedback submissions for the current project, newest first.
- **`posthog-pp-cli mcp-analytics intent-clusters-recompute`** - Trigger an asynchronous recompute of the intent cluster snapshot. The task runs in the background; poll the GET endpoint for progress (status transitions to 'idle' or 'error').
- **`posthog-pp-cli mcp-analytics intent-clusters-retrieve`** - Return the most recent intent cluster snapshot for the current project. Returns an empty IDLE snapshot when no clustering run has happened yet.
- **`posthog-pp-cli mcp-analytics missing-capabilities-create`** - Create a new missing capability report for the current project.
- **`posthog-pp-cli mcp-analytics missing-capabilities-list`** - List missing capability reports for the current project, newest first.
- **`posthog-pp-cli mcp-analytics sessions-activity-overview`** - Aggregate counters, top tools, agent clients, and the most recent tool calls for the last 30 days, computed in one request. Powers the dashboard's activity view; always computed fresh so polling callers watch data arrive.
- **`posthog-pp-cli mcp-analytics sessions-generate-intent`** - Generate (or return the cached) LLM summary of the agent's goal for a session, derived from its recorded $mcp_intents. The first call summarises and persists the result; subsequent calls return the stored summary.
- **`posthog-pp-cli mcp-analytics sessions-intent-digest`** - Generate (or return the cached) LLM digest of what agents are trying to do with this MCP server, derived from the most recent recorded $mcp_intents across all sessions: a one-sentence summary plus semantic themes, each sized and attributed to tools from the intents themselves. Cached by intent corpus and by recency, so repeated calls are cheap and a busy server regenerates at a bounded rate. Powers the dashboard's activity tab.
- **`posthog-pp-cli mcp-analytics sessions-list`** - List MCP sessions for the current project, derived by grouping $mcp_tool_call events by $mcp_session_id. Ordered by newest session start first by default.
- **`posthog-pp-cli mcp-analytics sessions-tool-calls`** - List a page of the $mcp_tool_call events that belong to a given $session_id, in chronological order.

### mcp-gateway

Manage mcp gateway

- **`posthog-pp-cli mcp-gateway audit-counts-retrieve`** - Totals backing the quick-filter chips.
- **`posthog-pp-cli mcp-gateway audit-list`** - Read-only trail of proxied tool calls. Project admins see all calls.
Members see calls made through their connections, including calls made by
agents using connections they shared.
- **`posthog-pp-cli mcp-gateway audit-retrieve`** - Read-only trail of proxied tool calls. Project admins see all calls.
Members see calls made through their connections, including calls made by
agents using connections they shared.
- **`posthog-pp-cli mcp-gateway config-apply-preset-create`** - Set the policy baseline for members or agents (admin-only).
- **`posthog-pp-cli mcp-gateway config-list`** - The team's gateway settings, plus whether the caller can administer them.
- **`posthog-pp-cli mcp-gateway config-set-all-servers-enabled-create`** - Enable or disable every MCP server for the team (admin-only): flips
each registered server and the default for untouched catalog servers,
so newly published templates follow the same posture.
- **`posthog-pp-cli mcp-gateway config-update-settings-create`** - Update team gateway settings (admin-only).
- **`posthog-pp-cli mcp-gateway members-list`** - Admin overview of each member's gateway posture, plus the per-member
server kill switch.
- **`posthog-pp-cli mcp-gateway members-retrieve`** - Admin overview of each member's gateway posture, plus the per-member
server kill switch.
- **`posthog-pp-cli mcp-gateway members-set-access-create`** - Turn one gateway server off (or back on) for one member.
- **`posthog-pp-cli mcp-gateway rules-create`** - Team guardrails evaluated before any scope policy.
- **`posthog-pp-cli mcp-gateway rules-destroy`** - Team guardrails evaluated before any scope policy.
- **`posthog-pp-cli mcp-gateway rules-list`** - Team guardrails evaluated before any scope policy.
- **`posthog-pp-cli mcp-gateway rules-partial-update`** - Team guardrails evaluated before any scope policy.
- **`posthog-pp-cli mcp-gateway rules-retrieve`** - Team guardrails evaluated before any scope policy.
- **`posthog-pp-cli mcp-gateway rules-update`** - Team guardrails evaluated before any scope policy.
- **`posthog-pp-cli mcp-gateway servers-destroy`** - The team's gateway server registry. The registry is sparse: rows appear
through the install/share/OAuth-start flows in views.py, or when an admin
toggles an untouched catalog template here (`set_template_enabled`).
Servers with no row follow the team config's `default_servers_enabled`.
- **`posthog-pp-cli mcp-gateway servers-list`** - The team's gateway server registry. The registry is sparse: rows appear
through the install/share/OAuth-start flows in views.py, or when an admin
toggles an untouched catalog template here (`set_template_enabled`).
Servers with no row follow the team config's `default_servers_enabled`.
- **`posthog-pp-cli mcp-gateway servers-partial-update`** - The team's gateway server registry. The registry is sparse: rows appear
through the install/share/OAuth-start flows in views.py, or when an admin
toggles an untouched catalog template here (`set_template_enabled`).
Servers with no row follow the team config's `default_servers_enabled`.
- **`posthog-pp-cli mcp-gateway servers-policies-create`** - Upsert per-tool states for a scope, returning the re-resolved catalog.
- **`posthog-pp-cli mcp-gateway servers-retrieve`** - The team's gateway server registry. The registry is sparse: rows appear
through the install/share/OAuth-start flows in views.py, or when an admin
toggles an untouched catalog template here (`set_template_enabled`).
Servers with no row follow the team config's `default_servers_enabled`.
- **`posthog-pp-cli mcp-gateway servers-set-template-enabled-create`** - Enable or disable a catalog template for the team (admin-only),
materializing its gateway registration on the first toggle.
- **`posthog-pp-cli mcp-gateway servers-tools-retrieve`** - Tool catalog with the resolved policy for a scope.
- **`posthog-pp-cli mcp-gateway servers-update`** - The team's gateway server registry. The registry is sparse: rows appear
through the install/share/OAuth-start flows in views.py, or when an admin
toggles an untouched catalog template here (`set_template_enabled`).
Servers with no row follow the team config's `default_servers_enabled`.
- **`posthog-pp-cli mcp-gateway service-accounts-access-create`** - Grant or revoke this agent's access to one gateway server.
- **`posthog-pp-cli mcp-gateway service-accounts-list`** - PostHog's built-in agents and their MCP access grants.

The catalog is fixed. Projects can pause an agent's MCP access and grant or
revoke servers, but cannot create, rename, rotate, or delete agents.
- **`posthog-pp-cli mcp-gateway service-accounts-partial-update`** - PostHog's built-in agents and their MCP access grants.

The catalog is fixed. Projects can pause an agent's MCP access and grant or
revoke servers, but cannot create, rename, rotate, or delete agents.
- **`posthog-pp-cli mcp-gateway service-accounts-retrieve`** - PostHog's built-in agents and their MCP access grants.

The catalog is fixed. Projects can pause an agent's MCP access and grant or
revoke servers, but cannot create, rename, rotate, or delete agents.
- **`posthog-pp-cli mcp-gateway service-accounts-update`** - PostHog's built-in agents and their MCP access grants.

The catalog is fixed. Projects can pause an agent's MCP access and grant or
revoke servers, but cannot create, rename, rotate, or delete agents.

### mcp-server-installations

Manage mcp server installations

- **`posthog-pp-cli mcp-server-installations authorize-retrieve`** - Start (or re-start) an OAuth flow.

Pass ``template_id`` to (re)connect a catalog template, or
``installation_id`` to reconnect an existing custom install using its
cached metadata and per-user DCR creds.
- **`posthog-pp-cli mcp-server-installations available-tools-retrieve`** - Every tool the caller can currently reach, across all their connections.

One request instead of one per connection: an agent surface resolving its
tool list on each session cannot afford a fan-out. `do_not_use` and removed
tools are omitted — an agent should not see what it cannot call — while
`needs_approval` tools are listed with their state so the caller can explain
the block rather than report the capability as missing.
- **`posthog-pp-cli mcp-server-installations create`** - Create
- **`posthog-pp-cli mcp-server-installations destroy`** - Destroy
- **`posthog-pp-cli mcp-server-installations install-custom-create`** - Install custom create
- **`posthog-pp-cli mcp-server-installations install-template-create`** - Install template create
- **`posthog-pp-cli mcp-server-installations list`** - List
- **`posthog-pp-cli mcp-server-installations partial-update`** - Partial update
- **`posthog-pp-cli mcp-server-installations retrieve`** - Retrieve
- **`posthog-pp-cli mcp-server-installations update`** - Update

### mcp-servers

Manage mcp servers

- **`posthog-pp-cli mcp-servers <project_id>`** - Lists curated MCP server templates that users can install with one click.

Templates are seeded by PostHog operators and carry shared, encrypted
OAuth client credentials. Inactive templates are hidden from the catalog.

### mcp-tools

Manage mcp tools

- **`posthog-pp-cli mcp-tools create`** - Invoke an MCP tool by name.

This endpoint allows MCP callers to invoke Max AI tools directly
without going through the full LangChain conversation flow.

Scopes are resolved dynamically per tool via dangerously_get_required_scopes.
- **`posthog-pp-cli mcp-tools docs-search`** - Run a hybrid (semantic + full-text) RAG search over the PostHog documentation via Inkeep. Returns a markdown body with title, URL, and excerpt for each match for the agent to cite back to the user.

### messaging-preferences

Manage messaging preferences

- **`posthog-pp-cli messaging-preferences add-opt-out-create`** - Manually add a recipient to the opt-out list for a specific category or all marketing messages.
- **`posthog-pp-cli messaging-preferences bulk-add-opt-outs-create`** - Opt every recipient in the list out of the category named on their entry, or a default category.
- **`posthog-pp-cli messaging-preferences export-opt-outs-csv-retrieve`** - Stream the opt-out list for a category as a CSV file that can be re-imported as-is.
- **`posthog-pp-cli messaging-preferences generate-link-create`** - Generate an unsubscribe link for the current user's email address
- **`posthog-pp-cli messaging-preferences opt-outs-retrieve`** - Get opt-outs filtered by category or overall opt-outs if no category specified
- **`posthog-pp-cli messaging-preferences remove-opt-out-create`** - Opt a recipient back in to a specific category, or to all marketing messages.
- **`posthog-pp-cli messaging-preferences webhook-url-retrieve`** - Return the webhook URL for Customer.io integration setup.

### messaging-suppressions

Manage messaging suppressions

- **`posthog-pp-cli messaging-suppressions add-suppression-create`** - Manually suppress an email address so no workflow sends to it.
- **`posthog-pp-cli messaging-suppressions remove-suppression-create`** - Remove an address from the suppression list so it can receive messages again.
- **`posthog-pp-cli messaging-suppressions suppressions-retrieve`** - List suppressed recipients for the team, most recently updated first.

### messaging-templates

Manage messaging templates

- **`posthog-pp-cli messaging-templates create`** - Create
- **`posthog-pp-cli messaging-templates destroy`** - Hard delete of this model is not allowed. Use a patch API call to set "deleted" to true
- **`posthog-pp-cli messaging-templates list`** - List
- **`posthog-pp-cli messaging-templates partial-update`** - Partial update
- **`posthog-pp-cli messaging-templates retrieve`** - Retrieve
- **`posthog-pp-cli messaging-templates update`** - Update

### metrics

Manage metrics

- **`posthog-pp-cli metrics attribute-values-retrieve`** - Observed values for one metric attribute key, most frequent first.
Backs the filter bar's value autocomplete.
- **`posthog-pp-cli metrics attributes-retrieve`** - Distinct attribute keys seen on the team's metrics (datapoint and
resource attributes merged), most frequent first. Backs the filter
bar's key autocomplete.
- **`posthog-pp-cli metrics characterize-create`** - Characterize a metric anomaly: compare an anomaly window against a
baseline, find the onset, and rank which label values moved.
- **`posthog-pp-cli metrics has-retrieve`** - Has retrieve
- **`posthog-pp-cli metrics query-create`** - Query create
- **`posthog-pp-cli metrics samples-create`** - Raw individual emissions for a metric (the events model), newest
first — backs the Samples view and the metric->trace pivot.
- **`posthog-pp-cli metrics values-retrieve`** - Distinct metric names for the team. Backs the picker UI.

### notebooks

Manage notebooks

- **`posthog-pp-cli notebooks all-activity-retrieve`** - The API for interacting with Notebooks. This feature is in early access and the API can have breaking changes without announcement.
- **`posthog-pp-cli notebooks create`** - The API for interacting with Notebooks. This feature is in early access and the API can have breaking changes without announcement.
- **`posthog-pp-cli notebooks destroy`** - Hard delete of this model is not allowed. Use a patch API call to set "deleted" to true
- **`posthog-pp-cli notebooks list`** - The API for interacting with Notebooks. This feature is in early access and the API can have breaking changes without announcement.
- **`posthog-pp-cli notebooks partial-update`** - The API for interacting with Notebooks. This feature is in early access and the API can have breaking changes without announcement.
- **`posthog-pp-cli notebooks recording-comments-retrieve`** - The API for interacting with Notebooks. This feature is in early access and the API can have breaking changes without announcement.
- **`posthog-pp-cli notebooks retrieve`** - The API for interacting with Notebooks. This feature is in early access and the API can have breaking changes without announcement.
- **`posthog-pp-cli notebooks update`** - The API for interacting with Notebooks. This feature is in early access and the API can have breaking changes without announcement.

### object-media-previews

Manage object media previews

- **`posthog-pp-cli object-media-previews create`** - Create
- **`posthog-pp-cli object-media-previews destroy`** - Destroy
- **`posthog-pp-cli object-media-previews list`** - List
- **`posthog-pp-cli object-media-previews partial-update`** - Partial update
- **`posthog-pp-cli object-media-previews preferred-for-event-retrieve`** - Get the preferred media preview for an event definition.
Most recent user-uploaded, then most recent exported asset.
Requires event_definition (query param).
- **`posthog-pp-cli object-media-previews retrieve`** - Retrieve
- **`posthog-pp-cli object-media-previews update`** - Update

### organizations

Manage organizations

- **`posthog-pp-cli organizations create`** - Create
- **`posthog-pp-cli organizations destroy`** - Destroy
- **`posthog-pp-cli organizations list`** - List
- **`posthog-pp-cli organizations partial-update`** - Partial update
- **`posthog-pp-cli organizations retrieve`** - Retrieve
- **`posthog-pp-cli organizations update`** - Update

### paths-v2

Manage paths v2

- **`posthog-pp-cli paths-v2 <project_id>`** - Converts a displayed journeys segment into the funnel query that reproduces its unique-actor count exactly. In open mode only a single edge converts (a two-step funnel with the inactivity gap as conversion window); in anchored mode any anchor-rooted chain converts (window W). The funnel is returned as JSON and is not executed or persisted here.

### persons

Manage persons

- **`posthog-pp-cli persons all-activity-retrieve`** - This endpoint is meant for reading and deleting persons. To create or update persons, we recommend using the [capture API](https://posthog.com/docs/api/capture), the `$set` and `$unset` [properties](https://posthog.com/docs/product-analytics/user-properties), or one of our SDKs.
- **`posthog-pp-cli persons batch-by-distinct-ids-create`** - This endpoint is meant for reading and deleting persons. To create or update persons, we recommend using the [capture API](https://posthog.com/docs/api/capture), the `$set` and `$unset` [properties](https://posthog.com/docs/product-analytics/user-properties), or one of our SDKs.
- **`posthog-pp-cli persons batch-by-uuids-create`** - This endpoint is meant for reading and deleting persons. To create or update persons, we recommend using the [capture API](https://posthog.com/docs/api/capture), the `$set` and `$unset` [properties](https://posthog.com/docs/product-analytics/user-properties), or one of our SDKs.
- **`posthog-pp-cli persons bulk-delete-create`** - This endpoint allows you to bulk delete persons, either by the PostHog person IDs or by distinct IDs. You can pass in a maximum of 1000 IDs per call. Only events captured before the request will be deleted.
- **`posthog-pp-cli persons cohorts-retrieve`** - This endpoint is meant for reading and deleting persons. To create or update persons, we recommend using the [capture API](https://posthog.com/docs/api/capture), the `$set` and `$unset` [properties](https://posthog.com/docs/product-analytics/user-properties), or one of our SDKs.
- **`posthog-pp-cli persons deletion-status-list`** - List the status of queued event deletions for persons. When you delete a person with `delete_events=true`, an async deletion is queued. Use this endpoint to check whether those deletions are still pending or have been completed.
- **`posthog-pp-cli persons list`** - This endpoint is meant for reading and deleting persons. To create or update persons, we recommend using the [capture API](https://posthog.com/docs/api/capture), the `$set` and `$unset` [properties](https://posthog.com/docs/product-analytics/user-properties), or one of our SDKs.
- **`posthog-pp-cli persons partial-update`** - This endpoint is meant for reading and deleting persons. To create or update persons, we recommend using the [capture API](https://posthog.com/docs/api/capture), the `$set` and `$unset` [properties](https://posthog.com/docs/product-analytics/user-properties), or one of our SDKs.
- **`posthog-pp-cli persons properties-at-time-retrieve`** - Get person properties as they existed at a specific point in time.

This endpoint reconstructs person properties by querying ClickHouse events
for $set and $set_once operations up to the specified timestamp.

Query parameters:
- distinct_id: The distinct_id of the person
- timestamp: ISO datetime string for the point in time (e.g., "2023-06-15T14:30:00Z")
- include_set_once: Whether to handle $set_once operations (default: false)
- **`posthog-pp-cli persons reset-distinct-id-create`** - Reset a distinct_id for a deleted person. This allows the distinct_id to be used again.
- **`posthog-pp-cli persons retrieve`** - This endpoint is meant for reading and deleting persons. To create or update persons, we recommend using the [capture API](https://posthog.com/docs/api/capture), the `$set` and `$unset` [properties](https://posthog.com/docs/product-analytics/user-properties), or one of our SDKs.
- **`posthog-pp-cli persons update`** - Only for setting properties on the person. "properties" from the request data will be updated via a "$set" event.
This means that only the properties listed will be updated, but other properties won't be removed nor updated.
If you would like to remove a property use the `delete_property` endpoint.
- **`posthog-pp-cli persons values-retrieve`** - This endpoint is meant for reading and deleting persons. To create or update persons, we recommend using the [capture API](https://posthog.com/docs/api/capture), the `$set` and `$unset` [properties](https://posthog.com/docs/product-analytics/user-properties), or one of our SDKs.

### plugin-configs

Manage plugin configs


### posthog-connections

Manage posthog connections


### product-enablement

Manage product enablement

- **`posthog-pp-cli product-enablement <project_id>`** - Create

### product-tours

Manage product tours

- **`posthog-pp-cli product-tours create`** - Create, read, update, and manage product tours and their targeting.
- **`posthog-pp-cli product-tours destroy`** - Create, read, update, and manage product tours and their targeting.
- **`posthog-pp-cli product-tours list`** - Create, read, update, and manage product tours and their targeting.
- **`posthog-pp-cli product-tours partial-update`** - Create, read, update, and manage product tours and their targeting.
- **`posthog-pp-cli product-tours retrieve`** - Create, read, update, and manage product tours and their targeting.
- **`posthog-pp-cli product-tours update`** - Create, read, update, and manage product tours and their targeting.

### project-secret-api-keys

Manage project secret api keys

- **`posthog-pp-cli project-secret-api-keys secret-api-keys-create`** - Secret api keys create
- **`posthog-pp-cli project-secret-api-keys secret-api-keys-destroy`** - Secret api keys destroy
- **`posthog-pp-cli project-secret-api-keys secret-api-keys-list`** - Secret api keys list
- **`posthog-pp-cli project-secret-api-keys secret-api-keys-partial-update`** - Secret api keys partial update
- **`posthog-pp-cli project-secret-api-keys secret-api-keys-retrieve`** - Secret api keys retrieve
- **`posthog-pp-cli project-secret-api-keys secret-api-keys-update`** - Secret api keys update

### property-access-controls

Manage property access controls

- **`posthog-pp-cli property-access-controls create`** - Create or update a property access control rule.
- **`posthog-pp-cli property-access-controls destroy`** - Delete a property access control rule. The rule is identified by `property_definition_id` plus an optional `organization_member` or `role` query parameter. Omitting both targets deletes the default rule.
- **`posthog-pp-cli property-access-controls retrieve`** - Get all property access control rules for a property definition.

### property-definitions

Manage property definitions

- **`posthog-pp-cli property-definitions bulk-update-tags-create`** - Bulk update tags on multiple objects.

PAT access: this action has no ``required_scopes=`` on the decorator —
inheriting viewsets must add ``"bulk_update_tags"`` to their
``scope_object_write_actions`` list to accept personal API keys.
Without that opt-in, ``APIScopePermission`` rejects PAT requests with
"This action does not support personal API key access". Done per-viewset
so granting ``<scope>:write`` for one resource doesn't leak access to
sibling resources that share this mixin.

Accepts:
- {"ids": [...], "action": "add"|"remove"|"set", "tags": ["tag1", "tag2"]}

Actions:
- "add": Add tags to existing tags on each object
- "remove": Remove specific tags from each object
- "set": Replace all tags on each object with the provided list
- **`posthog-pp-cli property-definitions destroy`** - Destroy
- **`posthog-pp-cli property-definitions list`** - List
- **`posthog-pp-cli property-definitions partial-update`** - Partial update
- **`posthog-pp-cli property-definitions retrieve`** - Retrieve
- **`posthog-pp-cli property-definitions seen-together-retrieve`** - Allows a caller to provide a list of event names and a single property name
Returns a map of the event names to a boolean representing whether that property has ever been seen with that event_name
- **`posthog-pp-cli property-definitions update`** - Update

### public-hog-function-templates

Manage public hog function templates

- **`posthog-pp-cli public-hog-function-templates`** - List

### pulse

Manage pulse

- **`posthog-pp-cli pulse brief-configs-create`** - Brief configs create
- **`posthog-pp-cli pulse brief-configs-destroy`** - Brief configs destroy
- **`posthog-pp-cli pulse brief-configs-list`** - Brief configs list
- **`posthog-pp-cli pulse brief-configs-partial-update`** - Brief configs partial update
- **`posthog-pp-cli pulse brief-configs-retrieve`** - Brief configs retrieve
- **`posthog-pp-cli pulse brief-configs-update`** - Brief configs update
- **`posthog-pp-cli pulse briefs-generate-create`** - Briefs generate create
- **`posthog-pp-cli pulse briefs-list`** - Briefs list
- **`posthog-pp-cli pulse briefs-retrieve`** - Briefs retrieve

### query

Manage query

- **`posthog-pp-cli query check-auth-for-async-create`** - DRF ViewSet mixin that gates coalesced responses behind permission checks.

The QueryCoalescingMiddleware attaches cached response data to
request.META["_coalesced_response"] for followers. This mixin runs DRF's
initial() (auth + permissions + throttling) before returning the
cached response, ensuring the request is authorized.
- **`posthog-pp-cli query create`** - DRF ViewSet mixin that gates coalesced responses behind permission checks.

The QueryCoalescingMiddleware attaches cached response data to
request.META["_coalesced_response"] for followers. This mixin runs DRF's
initial() (auth + permissions + throttling) before returning the
cached response, ensuring the request is authorized.
- **`posthog-pp-cli query create-with-kind`** - DRF ViewSet mixin that gates coalesced responses behind permission checks.

The QueryCoalescingMiddleware attaches cached response data to
request.META["_coalesced_response"] for followers. This mixin runs DRF's
initial() (auth + permissions + throttling) before returning the
cached response, ensuring the request is authorized.
- **`posthog-pp-cli query destroy`** - (Experimental)
- **`posthog-pp-cli query draft-sql-retrieve`** - DRF ViewSet mixin that gates coalesced responses behind permission checks.

The QueryCoalescingMiddleware attaches cached response data to
request.META["_coalesced_response"] for followers. This mixin runs DRF's
initial() (auth + permissions + throttling) before returning the
cached response, ensuring the request is authorized.
- **`posthog-pp-cli query retrieve`** - (Experimental)
- **`posthog-pp-cli query upgrade-create`** - Upgrades a query without executing it. Returns a query with all nodes migrated to the latest version.

### quota-limits

Manage quota limits

- **`posthog-pp-cli quota-limits <project_id>`** - Return the current quota-limit state for the team identified in the URL, keyed by `QuotaResource` value. Used by the LLM gateway to gate billable products on AI credits exhaustion.

### reminders

Manage reminders

- **`posthog-pp-cli reminders create`** - Create
- **`posthog-pp-cli reminders destroy`** - Destroy
- **`posthog-pp-cli reminders list`** - List
- **`posthog-pp-cli reminders partial-update`** - Partial update
- **`posthog-pp-cli reminders retrieve`** - Retrieve
- **`posthog-pp-cli reminders update`** - Update

### review-hog

Manage review hog

- **`posthog-pp-cli review-hog blind-spots-list`** - List the `review-hog-blind-spots-*` skills visible to the requesting user — the canonical skill plus the customs they authored — flagging the one active for them. The canonical skill is auto-seeded active on the first read; a custom skill the user has not selected shows as inactive.
- **`posthog-pp-cli review-hog blind-spots-partial-update`** - Make a `review-hog-blind-spots-*` skill the single sweep that runs on the requesting user's PR reviews, switching the user's other blind-spots skills off in the same call. Only skills visible to the user — the canonical plus the customs they authored — can be selected; anything else 404s. Upserts the per-user config row, so selecting a freshly authored custom skill works in one call.
- **`posthog-pp-cli review-hog perspectives-list`** - List the `review-hog-perspective-*` skills visible to the requesting user — the canonical perspectives plus the customs they authored — joined with their enable state. The 3 canonical perspectives are auto-seeded enabled on the first read; a custom perspective the user has not switched on shows as disabled.
- **`posthog-pp-cli review-hog perspectives-partial-update`** - Toggle whether a `review-hog-perspective-*` skill runs on the requesting user's PR reviews. Only skills visible to the user — the canonicals plus the customs they authored — can be toggled; anything else 404s. Upserts the per-user config row, so enabling a freshly authored custom perspective works in one call. Rejected if it would leave the user with no enabled perspective.
- **`posthog-pp-cli review-hog reviews-list`** - Recent ReviewHog reviews on this project: actively running reviews first (with the in-flight turn's stage), then the most recent completed ones — at most `limit` rows (default 5), plus `has_more` for whether a larger `limit` would reveal more. By default only the requesting user's reviews; `scope=everyone` lists every review on the project.
- **`posthog-pp-cli review-hog reviews-perspective-stats-retrieve`** - How many findings each review skill (perspective or blind-spot sweep) raised across the recent completed reviews in scope — the requesting user's by default, every review on this project with `scope=everyone` — and how many of those the validator kept vs dismissed.
- **`posthog-pp-cli review-hog reviews-retrieve`** - One completed ReviewHog review on this project, with the latest turn's validated findings, the findings the validator dismissed (and why), and the review body published to GitHub. Project-wide, so reviews listed under `scope=everyone` can be opened too.
- **`posthog-pp-cli review-hog reviews-trigger-create`** - Start a ReviewHog review of any pull request the project's GitHub App installation can access, and publish it back to the PR. The requesting user is the review's acting user: their enabled perspectives, blind-spot check, validator, and urgency threshold drive the run, and it appears under their recent reviews. Nonexistent, closed, and fork PRs are rejected synchronously; a PR whose current commit already has a published review returns 'already_reviewed' without starting a run, and triggering a PR whose review is currently running joins the in-flight run. Otherwise non-blocking: returns the Temporal workflow id immediately while the review runs in the worker.
- **`posthog-pp-cli review-hog validators-list`** - List the `review-hog-validation-*` skills visible to the requesting user — the canonical validator plus the customs they authored — flagging the one active for them. The canonical validator is auto-seeded active on the first read; a custom validator the user has not selected shows as inactive.
- **`posthog-pp-cli review-hog validators-partial-update`** - Make a `review-hog-validation-*` skill the single validator that runs on the requesting user's PR reviews, switching the user's other validators off in the same call. Only skills visible to the user — the canonical plus the customs they authored — can be selected; anything else 404s. Upserts the per-user config row, so selecting a freshly authored custom validator works in one call.

### sandbox-custom-images

Manage sandbox custom images

- **`posthog-pp-cli sandbox-custom-images create`** - Create a draft custom image and start its interactive image-builder agent task. The returned builder_task_id points at the conversation.
- **`posthog-pp-cli sandbox-custom-images destroy`** - API for custom sandbox base images, built on top of the VM sandbox base via an image-builder agent.

Custom images only run on the Modal VM runtime, so every action is gated on the
`tasks-modal-vm-sandbox` flag (org-enabled with `user_created` in its origin_products payload).
- **`posthog-pp-cli sandbox-custom-images list`** - API for custom sandbox base images, built on top of the VM sandbox base via an image-builder agent.

Custom images only run on the Modal VM runtime, so every action is gated on the
`tasks-modal-vm-sandbox` flag (org-enabled with `user_created` in its origin_products payload).
- **`posthog-pp-cli sandbox-custom-images partial-update`** - Rename or update the description of a custom image. Only mutable metadata (name, description) is editable; the build spec and status are managed by the build flow.
- **`posthog-pp-cli sandbox-custom-images retrieve`** - API for custom sandbox base images, built on top of the VM sandbox base via an image-builder agent.

Custom images only run on the Modal VM runtime, so every action is gated on the
`tasks-modal-vm-sandbox` flag (org-enabled with `user_created` in its origin_products payload).

### sandbox-environments

Manage sandbox environments

- **`posthog-pp-cli sandbox-environments sandbox-create`** - API for managing sandbox environments that control network access for task runs.
- **`posthog-pp-cli sandbox-environments sandbox-destroy`** - API for managing sandbox environments that control network access for task runs.
- **`posthog-pp-cli sandbox-environments sandbox-list`** - API for managing sandbox environments that control network access for task runs.
- **`posthog-pp-cli sandbox-environments sandbox-partial-update`** - API for managing sandbox environments that control network access for task runs.
- **`posthog-pp-cli sandbox-environments sandbox-retrieve`** - API for managing sandbox environments that control network access for task runs.

### saved

Manage saved

- **`posthog-pp-cli saved create`** - Create a saved heatmap for a page URL. For type 'screenshot' (the default) this enqueues a headless render of the page at each target width; poll the saved heatmap or its content endpoint until status is 'completed'. Provide 'widths' to control which viewport widths are rendered.
- **`posthog-pp-cli saved destroy`** - Hard delete of this model is not allowed. Use a patch API call to set "deleted" to true
- **`posthog-pp-cli saved list`** - List saved heatmaps for the project. A saved heatmap pins a page URL and a set of viewport widths, and (for type 'screenshot') renders the page so heatmap data can be overlaid on it.
- **`posthog-pp-cli saved partial-update`** - Update a saved heatmap (e.g. rename, change widths, or soft-delete via 'deleted'). Changing the URL of a 'screenshot' heatmap triggers a re-render.
- **`posthog-pp-cli saved preflight-create`** - Fetch a page URL server-side and report whether it allows being embedded in the live preview iframe, plus the HTTP status it returned. The live preview loads the customer's site directly in their browser, so a site that sends X-Frame-Options or a restrictive frame-ancestors will never render, and a 4xx or 5xx from the site's own host or CDN leaves an empty frame with no explanation. This endpoint makes both cases explainable. The fetch comes from PostHog's own network rather than from the screenshot renderer, so a host that varies its response by IP or user agent can answer this differently than it answers a screenshot render. Settled verdicts are cached briefly, so repeat checks for the same URL do not refetch it.
- **`posthog-pp-cli saved prewarm-create`** - Speculatively render a screenshot for a page URL ahead of heatmap creation, so it's ready (or closer to ready) by the time the user reaches the generation screen. Renders a single preview width. Idempotent within a short window: returns the existing in-flight or completed prewarm render for the same URL and consent setting if one exists (200), otherwise starts a new one (201). The result is reused when a heatmap is later created for the same URL.
- **`posthog-pp-cli saved retrieve`** - Get a single saved heatmap by its short_id, including per-width render status.

### saved-query-column-annotations

Manage saved query column annotations

- **`posthog-pp-cli saved-query-column-annotations create`** - Read and edit semantic descriptions of data-modelling views and columns surfaced to the AI agent.

List can be filtered to one view with `?saved_query_id=<uuid>`. Any create or update is treated as a
user edit (`is_user_edited=True`), which protects the row from being overwritten by automatic
enrichment. Create upserts on `(saved_query, column_name)`; the view cannot be changed after creation.
- **`posthog-pp-cli saved-query-column-annotations destroy`** - Read and edit semantic descriptions of data-modelling views and columns surfaced to the AI agent.

List can be filtered to one view with `?saved_query_id=<uuid>`. Any create or update is treated as a
user edit (`is_user_edited=True`), which protects the row from being overwritten by automatic
enrichment. Create upserts on `(saved_query, column_name)`; the view cannot be changed after creation.
- **`posthog-pp-cli saved-query-column-annotations list`** - Read and edit semantic descriptions of data-modelling views and columns surfaced to the AI agent.

List can be filtered to one view with `?saved_query_id=<uuid>`. Any create or update is treated as a
user edit (`is_user_edited=True`), which protects the row from being overwritten by automatic
enrichment. Create upserts on `(saved_query, column_name)`; the view cannot be changed after creation.
- **`posthog-pp-cli saved-query-column-annotations partial-update`** - Read and edit semantic descriptions of data-modelling views and columns surfaced to the AI agent.

List can be filtered to one view with `?saved_query_id=<uuid>`. Any create or update is treated as a
user edit (`is_user_edited=True`), which protects the row from being overwritten by automatic
enrichment. Create upserts on `(saved_query, column_name)`; the view cannot be changed after creation.
- **`posthog-pp-cli saved-query-column-annotations retrieve`** - Read and edit semantic descriptions of data-modelling views and columns surfaced to the AI agent.

List can be filtered to one view with `?saved_query_id=<uuid>`. Any create or update is treated as a
user edit (`is_user_edited=True`), which protects the row from being overwritten by automatic
enrichment. Create upserts on `(saved_query, column_name)`; the view cannot be changed after creation.
- **`posthog-pp-cli saved-query-column-annotations update`** - Read and edit semantic descriptions of data-modelling views and columns surfaced to the AI agent.

List can be filtered to one view with `?saved_query_id=<uuid>`. Any create or update is treated as a
user edit (`is_user_edited=True`), which protects the row from being overwritten by automatic
enrichment. Create upserts on `(saved_query, column_name)`; the view cannot be changed after creation.

### scheduled-changes

Manage scheduled changes

- **`posthog-pp-cli scheduled-changes create`** - Create, read, update and delete scheduled changes.
- **`posthog-pp-cli scheduled-changes destroy`** - Create, read, update and delete scheduled changes.
- **`posthog-pp-cli scheduled-changes list`** - Create, read, update and delete scheduled changes.
- **`posthog-pp-cli scheduled-changes partial-update`** - Create, read, update and delete scheduled changes.
- **`posthog-pp-cli scheduled-changes retrieve`** - Create, read, update and delete scheduled changes.
- **`posthog-pp-cli scheduled-changes update`** - Create, read, update and delete scheduled changes.

### schema-property-groups

Manage schema property groups

- **`posthog-pp-cli schema-property-groups create`** - Create
- **`posthog-pp-cli schema-property-groups destroy`** - Destroy
- **`posthog-pp-cli schema-property-groups list`** - List
- **`posthog-pp-cli schema-property-groups partial-update`** - Partial update
- **`posthog-pp-cli schema-property-groups retrieve`** - Retrieve
- **`posthog-pp-cli schema-property-groups update`** - Update

### sdk-health

Manage sdk health

- **`posthog-pp-cli sdk-health <project_id>`** - Returns a pre-digested health assessment of the PostHog SDKs the project is using. Covers which SDKs are current vs outdated (smart-semver rules with grace periods and traffic-percentage thresholds), per-version breakdown, and a human-readable reason for each assessment. Use this to diagnose SDK version issues, surface upgrade recommendations, or check overall SDK health.

### session-group-summaries

Manage session group summaries

- **`posthog-pp-cli session-group-summaries create`** - API for retrieving and managing stored group session summaries.
- **`posthog-pp-cli session-group-summaries destroy`** - API for retrieving and managing stored group session summaries.
- **`posthog-pp-cli session-group-summaries list`** - API for retrieving and managing stored group session summaries.
- **`posthog-pp-cli session-group-summaries partial-update`** - API for retrieving and managing stored group session summaries.
- **`posthog-pp-cli session-group-summaries retrieve`** - API for retrieving and managing stored group session summaries.
- **`posthog-pp-cli session-group-summaries update`** - API for retrieving and managing stored group session summaries.

### session-recording-playlists

Manage session recording playlists

- **`posthog-pp-cli session-recording-playlists create`** - Create
- **`posthog-pp-cli session-recording-playlists destroy`** - Hard delete of this model is not allowed. Use a patch API call to set "deleted" to true
- **`posthog-pp-cli session-recording-playlists list`** - Override list to include synthetic playlists.

Synthetics have no DB row, so we compute each one's position in the merged
sort and split the requested page between synthetics and a DB queryset slice.
The merge/rank/sort is all in-memory, so each phase is wrapped in a span and
the input sizes are recorded as span attributes — a slow response on a team
with many playlists then shows up as a wide span against a large db_count.
- **`posthog-pp-cli session-recording-playlists partial-update`** - Partial update
- **`posthog-pp-cli session-recording-playlists retrieve`** - Retrieve
- **`posthog-pp-cli session-recording-playlists update`** - Update

### session-recordings

Manage session recordings

- **`posthog-pp-cli session-recordings bulk-delete-create`** - Delete a batch of session recordings by session ID. Deletion is permanent and cannot be undone. IDs that don't match an existing recording are skipped and counted in `total_requested` but not `deleted_count`.
- **`posthog-pp-cli session-recordings destroy`** - Destroy
- **`posthog-pp-cli session-recordings list`** - List
- **`posthog-pp-cli session-recordings partial-update`** - Partial update
- **`posthog-pp-cli session-recordings retrieve`** - Retrieve
- **`posthog-pp-cli session-recordings update`** - Update

### session-summaries

Manage session summaries

- **`posthog-pp-cli session-summaries create`** - Generate AI summary for a group of session recordings to find patterns and generate a notebook.
- **`posthog-pp-cli session-summaries create-individually`** - Generate AI individual summary for each session, without grouping.
- **`posthog-pp-cli session-summaries retrieve-config`** - Retrieve the team's session summaries configuration (product context used to tailor single-session replay summaries).
- **`posthog-pp-cli session-summaries update-config`** - Update the team's session summaries configuration (product context used to tailor single-session replay summaries).

### sessions

Manage sessions

- **`posthog-pp-cli sessions property-definitions-retrieve`** - Property definitions retrieve
- **`posthog-pp-cli sessions values-retrieve`** - Values retrieve

### signals

Manage signals

- **`posthog-pp-cli signals processing-list`** - Return current processing state including pause status.
- **`posthog-pp-cli signals processing-pause-destroy`** - View and control signal processing pipeline state for a team.
- **`posthog-pp-cli signals processing-pause-update`** - View and control signal processing pipeline state for a team.
- **`posthog-pp-cli signals report-artefacts-create`** - Append an artefact to a report (see artefact_type for the writable types). Everything is append-only: log entries (code reference, commit, task run, note) accumulate, while status types (safety / actionability / priority judgments, repo selection, suggested reviewers) are latest-wins — appending a new version supersedes the previous one as the report's canonical status. Content is validated against the type's schema.
- **`posthog-pp-cli signals report-artefacts-destroy`** - Delete an artefact, addressed by id. Deleting the latest row of a status type reverts the report's canonical status to the previous version (latest-wins over what remains).
- **`posthog-pp-cli signals report-artefacts-diff`** - Fetch the unified diff of a `commit` artefact's branch against the repository default branch via the team's GitHub integration — using the branch's current tip so the diff reflects the latest state of the work, not just the single recorded commit.
- **`posthog-pp-cli signals report-artefacts-list`** - List every artefact on a report — the full work log: signal findings (the evidence behind the report), status judgments (safety / actionability / priority, repo selection, suggested reviewers — the newest row of each status type is canonical), and log entries (code references, commits, task runs, notes). `suggested_reviewers` content is enriched with PostHog user info at read time.
- **`posthog-pp-cli signals report-artefacts-partial-update`** - Replace the content of an existing artefact, addressed by id. The new content is validated against the artefact's type schema. Editing the latest row of a status type changes the report's canonical status (latest-wins); to re-assess while keeping history, append a new artefact instead. Attribution is creation-time only — edits don't reassign it.
- **`posthog-pp-cli signals report-artefacts-retrieve`** - Get one artefact by id, content parsed (and reviewers enriched) the same way as the list.
- **`posthog-pp-cli signals report-pr-checks`** - Fetch the CI status (GitHub Actions check runs and legacy commit statuses) of the pull request the report's implementation task opened, via the team's GitHub integration.
- **`posthog-pp-cli signals report-pr-comments`** - Fetch the pull request's conversation comments and inline review comments, merged chronologically, via the team's GitHub integration.
- **`posthog-pp-cli signals report-pr-review-comment-destroy`** - Delete one of the requesting user's own review comments
- **`posthog-pp-cli signals report-pr-review-comment-reaction-destroy`** - Remove one of the requesting user's own reactions from a review comment
- **`posthog-pp-cli signals report-pr-review-comment-reactions-create`** - React to a review comment as the requesting user
- **`posthog-pp-cli signals report-pr-review-comment-update`** - Edit one of the requesting user's own review comments
- **`posthog-pp-cli signals report-pr-review-comments-create`** - Post an inline review comment on the report's implementation pull request, attributed to the requesting user's own GitHub identity via their personal GitHub connection. Either replies to an existing thread (`in_reply_to`) or starts a new thread on a diff line (`path` + `line`).
- **`posthog-pp-cli signals reports-bulk-state-create`** - Transition many reports to a new state in one call.

Each id is processed independently: a report whose transition isn't allowed from its
current status is reported as `skipped` (a 409 on the single-report endpoint) and the
rest still go through. Returns one result per requested id (in request order, after
de-duplication) plus per-outcome counts. The whole call is 200 even on partial failure —
inspect `results` / the counts to see what happened.
- **`posthog-pp-cli signals reports-feedback-create`** - Record the thumbs rating at the end of a report, with an optional note. For browser-session requests the rating is persisted as a per-person report action, which counts as consumption evidence for the scout that authored the report (scouts whose output nobody consumes are eventually paused); requests authenticated any other way record no action. When a note is present and the report was authored by a scout, the note is also forwarded to that scout as a steering note it reads on its next run; for any other report there is nothing to steer. The report's state is never changed.
- **`posthog-pp-cli signals reports-list`** - Reports list
- **`posthog-pp-cli signals reports-partial-update`** - Edit the human-facing title and/or summary (description) of a signal report, addressed by id. Both fields are optional — supply only the ones you want to change; at least one is required. Every other report field (status, weights, judgments) is managed by the signals pipeline and cannot be set here. Returns the full updated report.
- **`posthog-pp-cli signals reports-refund-create`** - Refund the flat charge for this report's implementation PR and archive the report. Refunds auto-approve: the charge is either excluded from usage before it is ever reported to billing (refund on the same UTC day as the PR run) or returned as a Stripe customer-balance credit on the next invoice. A refunded PR does not count toward the free monthly PR allowance. One refund per report, ever — repeat calls return the existing refund with already_refunded=true. The report is archived as part of the refund (a resolved report stays resolved) and can't be restored afterwards.
- **`posthog-pp-cli signals reports-refund-summary-retrieve`** - Aggregate credited-path refunds across the whole organization for the current billing period — counts only, no per-team detail. The billing usage widget needs this because billing usage is org-wide while reports (and their refunds) are team-scoped: subtract the refunded credits from billing usage to show the net PR count. Excluded-path refunds never reach billing usage, so no adjustment is needed for them. Also carries the org's live billable credits for the period (billing's recorded usage lags by up to a day), so the widget can count just-created PRs and react to same-day refunds.
- **`posthog-pp-cli signals reports-retrieve`** - Reports retrieve
- **`posthog-pp-cli signals reports-retrieve-projects`** - Fetch all signals for a report from ClickHouse, including full metadata.
- **`posthog-pp-cli signals reports-state-create`** - Transition a report to a new state. The model validates allowed transitions.

The request body is validated by SignalReportStateRequestSerializer — only the
fields it declares (state, dismissal_reason, dismissal_note, snooze_for) are read,
and only snooze_for is ever forwarded to transition_to. Any other key is ignored,
so internal transition_to kwargs (reset_weight, error, ...) can't be injected.

Body: {
    "state": "suppressed" | "potential" | "resolved",
    # Optional dismissal feedback (honored when state == "suppressed", "potential", or "resolved"):
    "dismissal_reason": "<canonical reason code, see SIGNAL_REPORT_DISMISSAL_REASON_CHOICES>",
    "dismissal_note": "free-form text",
    # Optional, only honored for state == "potential":
    "snooze_for": <number of additional signals before re-promotion>,
}
- **`posthog-pp-cli signals reports-viewed-create`** - Record that the caller opened this report's detail view. One row per person per report is kept (repeat views bump a counter), and the record counts as consumption evidence for the scout that authored the report — scouts whose reports nobody consumes are eventually paused. Intended as fire-and-forget from the inbox UI when a person opens a report. Only browser-session requests leave a record; a call with any other credential (personal API key, OAuth token) returns 204 but records nothing.
- **`posthog-pp-cli signals scout-config-create`** - Register the config for a `signals-scout-*` skill immediately, without waiting for the coordinator to auto-register it. The same call can optionally set `run_interval_minutes`, a cron `run_cron_schedule`, `enabled`, `emit`, `network_access`, and output destinations. The skill must already exist on this project. Upsert: if a config already exists for the skill, the provided fields are applied to it.
- **`posthog-pp-cli signals scout-config-destroy`** - Delete one scout config by its `id`, removing the per-(team, skill) schedule/emit row outright. The point is cleaning up an orphaned config whose `signals-scout-*` skill was archived or deleted — it lingers in `list` with an empty `description`, never runs (the coordinator skips it and the skill can't load), but can't otherwise be removed over the API. Deletion is activity-logged. Note: if the skill still exists, the coordinator re-creates a default-schedule config on its next tick — to retire a live scout, archive its skill (or set `enabled=false` to make it inert) rather than deleting the config.
- **`posthog-pp-cli signals scout-config-list`** - List the per-(team, skill) scout configs for this project. Each row includes its schedule (rolling `run_interval_minutes`, or a project-local `run_cron_schedule` when set), `enabled`, `emit` posture, and `tags`. A freshly authored scout skill appears here once its config is registered, either explicitly via create or by the coordinator's next tick. Pass `tags` to narrow the fleet to the scouts carrying at least one of the given labels.
- **`posthog-pp-cli signals scout-config-run`** - Dispatch one on-demand run of this scout immediately, regardless of its schedule. Useful to test a scout right after authoring it, or to refresh its findings on demand. The run executes asynchronously on the worker and inherits every guard the scheduled path has: it is forbidden if scouts are not enabled for the project (403), and skipped if the project is over its Signals credits quota or daily run budget (429) or a run for this scout is already in progress (409). A manual run counts against the same daily run budget as scheduled runs, so repeated manual runs of the same scout can exhaust the project's daily allowance. A manual run does not change the scout's schedule or `last_run_at`. A disabled scout can still be run this way (to test before enabling). Returns immediately with the workflow id — poll the scout's runs for the result.
- **`posthog-pp-cli signals scout-config-sync`** - Materialize the scout fleet for this project on demand (idempotent): seed the canonical `signals-scout-*` skills, create a default-schedule config for any scout lacking one, and return all scout configs. Normally the Temporal coordinator does this on its next tick; this action exists so setup flows (e.g. the wizard's self-driving program) can hand the user a tunable fleet immediately.
- **`posthog-pp-cli signals scout-config-update`** - Tune one scout: change its schedule (rolling `run_interval_minutes`, or a cron `run_cron_schedule` that takes precedence when set), `enabled`, `emit` (dry-run) posture, `network_access` (trusted-domain allowlist vs full access for the scout's sandbox), or output destinations. `skill_name` is fixed. Enabling records `enabled_by` and is activity-logged since it drives spend.
- **`posthog-pp-cli signals scout-create`** - Create a `signals-scout-*` skill and its runnable config atomically. The skill always receives the report-channel tools. The optional config controls schedule, enablement, dry-run posture, network access, and typed destinations such as Slack. Repeating the same definition is safe and applies any supplied config fields; reusing its name for a different definition returns 409.
- **`posthog-pp-cli signals scout-edit-report`** - Rewrite a report's title/summary, append a note, and/or set its suggested reviewers. Can target ANY of the project's inbox reports, not just scout-authored ones — so the edit is attributed to this scout. Setting reviewers is how you rescue a report that surfaced routed to no one: it replaces the reviewer list and re-runs autostart, so a report missing a qualifying reviewer can open a draft PR. Title/summary edits are best-effort: the pipeline may later re-research them.
- **`posthog-pp-cli signals scout-emit`** - Fire `emit_signal` with `source_product = signals_scout`. The `finding_id` is baked into the deterministic `Signal.source_id = run:<id>:finding:<id>` for traceability, but this is NOT idempotent — a second call with the same `finding_id` emits a second signal, so do not retry an emit that may have already succeeded.
- **`posthog-pp-cli signals scout-emit-report`** - The second emit channel: author a complete `SignalReport` directly instead of emitting a weak signal. The report passes the safety judge, then surfaces at the status the scout's `actionability` call implies (or is suppressed). Backing `evidence` is written as bound signals so the report behaves like a pipeline report. NOT idempotent — a retry authors a second report; use `reports` to find a prior report and `edit-report` to update it instead.
- **`posthog-pp-cli signals scout-members-list`** - Return the people who can review work on this project — one row per member with access to it, each with their `user_uuid`, `email`, `first_name`/`last_name`, and resolved GitHub `login` (null when they have no linked GitHub identity). The cold-start reviewer-routing path: when a finding's owner can't be read off a fetched entity's `created_by` and there's no cached `reviewer:<area>` memory or inbox precedent, list members, match the owner by email/name, then put their resolved `github_login` in `suggested_reviewers` on `emit-report` / `edit-report`. Pass `search` to narrow a large roster; the result is capped at 200. Strictly team-scoped.
- **`posthog-pp-cli signals scout-metadata-get`** - Return the project's scout metadata: whether it is enrolled, the current announcement banner (e.g. an alpha run-limit notice, or null when unset), and the enforced run limits with current usage. Limits reflect what the coordinator actually applies at dispatch, so a user can see the real throttle rather than what they assume they set. All values come from the `signals-scout` flag payload, so the banner and caps can change with no deploy.
- **`posthog-pp-cli signals scout-notes-create`** - Leave a steering note the scout fleet reads on its next runs. Address it to one scout via `skill_name` (`signals-scout-*`), or omit it for a general note every scout sees. Each call creates a new note (no upsert); delete retires one. Attributed to the authenticated user.
- **`posthog-pp-cli signals scout-notes-destroy`** - Delete one note by its `id`, retiring it from every scout's view. Use this when a note has been acted on or no longer applies; time-boxed notes can instead carry an `expires_at` and retire themselves.
- **`posthog-pp-cli signals scout-notes-list`** - Return the steering notes left for this project's scouts, newest first. Pass `skill_name` to get the notes addressed to one scout plus the general (blank-target) fleet-wide notes — the shape a scout run reads at cold start. Omit `skill_name` to browse every note. Expired notes are excluded unless `include_expired=true`. `date_from` / `date_to` are a half-open window on `created_at` (`>= date_from`, `< date_to`); pass `date_to` (the `created_at` of the oldest note seen) to walk past the cap. Results capped at 500.
- **`posthog-pp-cli signals scout-project-profile-get`** - Return the team's deterministic project profile. For the internal scout token the response reflects the newest non-expired cached row or a freshly-built one (lazy compute on cache miss); `force_refresh=true` skips the cache and rebuilds from authoritative sources. Public read callers (session auth or a `signal_scout:read` PAK) get the newest cached profile, or 404 if none has been built yet — they never trigger a rebuild. Read this at the start of a run to orient on the team's product mix, integrations, warehouse sources, signal coverage, and existing inbox surface.
- **`posthog-pp-cli signals scout-record-output`** - The structured-output channel: record schema-validated records this run produced. Opt-in via the scout config's `structured_output_schema` (a JSON Schema describing one record) — without it the call fails closed, as it does for a dry-run scout (emit off). All-or-nothing: any invalid record fails the whole call with nothing written, so fix and resubmit the batch. Each accepted record lands in the project's event stream as a `$scout_structured_output` event — query them like any event (insights, SQL over `events`). Recording is idempotent: event ids are deterministic, so resubmitting an identical batch (e.g. retrying after a 503) cannot double-count.
- **`posthog-pp-cli signals scout-runs-emission-reports`** - Best-effort reverse of the report -> signals link. For each finding the run emitted, resolve the inbox `SignalReport` (if any) its underlying signal grouped into by walking the deterministic `source_id` back through the signal store. `report` is null when the finding hasn't grouped into a report yet, was de-duplicated away, or its signal was deleted. Lets the scout UI surface which inbox report a finding contributed to — the reverse of the report's evidence list. Strictly team-scoped — a run UUID belonging to another team returns 404.
- **`posthog-pp-cli signals scout-runs-emission-reports-batch`** - Batched form of the per-run emission-reports endpoint. For every finding the requested runs emitted, resolve the inbox `SignalReport` (if any) its signal grouped into — all in a single ClickHouse round-trip rather than one query per run, which is what made the findings page slow to open. `report` is null when a finding hasn't grouped yet, was de-duplicated, or its signal was deleted. Strictly team-scoped — run ids belonging to another team contribute no rows.
- **`posthog-pp-cli signals scout-runs-emissions`** - Return the findings a `SignalScoutRun` emitted to the inbox, newest first — one row per emit with its `description` (the finding text as surfaced), `weight`, `confidence`, `severity`, and the deterministic `source_id` that joins back to the underlying signal. Lets a team and its agents see *what* a run surfaced without parsing `emitted_finding_ids` or scanning the signal store. Strictly team-scoped — a run UUID belonging to another team returns 404.
- **`posthog-pp-cli signals scout-runs-emissions-batch`** - Batched form of the per-run emissions endpoint: return the findings every requested `SignalScoutRun` emitted, flattened newest-first, in a single request. Each row carries its `run_id`, so the caller can regroup by run. The findings UI uses this to load the whole recent window in one round-trip instead of one request per run. Strictly team-scoped — run ids belonging to another team contribute no rows (no per-run 404; one stale id never fails the batch).
- **`posthog-pp-cli signals scout-runs-findings-summary`** - Return a cheap fleet-wide tally of the output the scout troop produced in the recent window — the finding count, the distinct reports authored/edited via the report channel, the number of distinct scouts behind them, and the latest output time. Backs the 'Scout findings' callout so it renders from one query instead of the client paging through the whole runs window. Counts runs that emitted at least one finding (`emitted_count > 0`) or authored/edited an inbox report within the last `window_hours` (default 72), capped to the most recent 120 such runs so the count matches what the findings list renders. Strictly team-scoped.
- **`posthog-pp-cli signals scout-runs-list`** - Return the most recent `SignalScoutRun` summaries for this project, newest first. Used by the headless scout to dedupe against work other runs already covered. ILIKE matches on `summary`. `date_from` / `date_to` are a half-open window on `created_at` (`>= date_from`, `< date_to`); pass `date_to` on subsequent calls to walk past the 100-row cap. Pass `emitted=true` to see only runs that surfaced at least one finding. Pass `skill_name` (optionally with `skill_version`) to scope to a single scout. Results capped at 100.
- **`posthog-pp-cli signals scout-runs-recent-emissions`** - Return the team's recently emitted scout findings across *every* run, newest first — the cross-run counterpart to the per-run `emissions` action. Each row carries its `run_id`, so you can regroup by run without first listing runs and fanning out one `emissions` call each. Pass `skill_name` to scope to a single scout, and `date_from` / `date_to` (a half-open window on `emitted_at`) to bound or paginate — set `date_to` to the oldest emission's `emitted_at` to walk back past the limit. Pure Postgres, no ClickHouse round-trip. Capped at 200 rows (default 50).
- **`posthog-pp-cli signals scout-runs-retrieve`** - Return the full `SignalScoutRun` row. Status, timing, and error flow from the linked `tasks.TaskRun`. Strictly team-scoped — a UUID belonging to another team returns 404.
- **`posthog-pp-cli signals scout-scratchpad-forget`** - Delete an entry by key. Returns `deleted=false` if no row matched.
- **`posthog-pp-cli signals scout-scratchpad-remember`** - Upsert a memory keyed on `(team, key)`. Re-using a key updates the existing entry in place.
- **`posthog-pp-cli signals scout-scratchpad-search`** - Return `SignalScratchpad` entries for this project, newest-first. ILIKE matches on `content` and `key`; pass `key` instead for an exact single-entry lookup. `date_from` / `date_to` are a half-open window on `updated_at` (`>= date_from`, `< date_to`); pass `date_to` (the `updated_at` of the oldest entry seen) on subsequent calls to walk past the cap. Pass `keys_only=true` to scan keys without pulling entry bodies, or `content_max_chars` to cap each `content` to a preview — both keep a wide orientation scan from returning every entry's full prose. Results capped at 1000.
- **`posthog-pp-cli signals source-configs-create`** - Source configs create
- **`posthog-pp-cli signals source-configs-destroy`** - Source configs destroy
- **`posthog-pp-cli signals source-configs-list`** - Source configs list
- **`posthog-pp-cli signals source-configs-partial-update`** - Source configs partial update
- **`posthog-pp-cli signals source-configs-retrieve`** - Source configs retrieve
- **`posthog-pp-cli signals source-configs-update`** - Source configs update

### single-session-summaries

Manage single session summaries

- **`posthog-pp-cli single-session-summaries list`** - List stored AI-generated session summaries for the team, one row per session (latest summary kept). Use to discover which sessions have been summarized and to filter for sessions with specific problems — `has_exceptions=true`, `outcome=failure`, or a custom `session_ids` narrowing. Returns lightweight rows without the full summary JSON; use the retrieve endpoint for the per-segment / per-action detail.
- **`posthog-pp-cli single-session-summaries retrieve`** - Get the latest stored AI summary for a single session by `session_id`. Returns the full `summary` JSON (segments with named timeline, per-action `abandonment` / `confusion` / `exception` flags, segment outcomes, headline `session_outcome`, optional `sentiment`), the `exception_event_ids` array, the `extra_summary_context` (e.g. `focus_area`) used at generation time, and the `run_metadata` (LLM model used, whether visual confirmation was applied). 404 if no summary has been generated for this session yet — to trigger generation, use the existing `session-recording-summarize` flow rather than this endpoint.

### stamphog

Manage stamphog

- **`posthog-pp-cli stamphog digest-channels-create`** - Per-audience Slack destinations for the daily merged-PR digest.
- **`posthog-pp-cli stamphog digest-channels-destroy`** - Per-audience Slack destinations for the daily merged-PR digest.
- **`posthog-pp-cli stamphog digest-channels-list`** - Per-audience Slack destinations for the daily merged-PR digest.
- **`posthog-pp-cli stamphog digest-channels-partial-update`** - Per-audience Slack destinations for the daily merged-PR digest.
- **`posthog-pp-cli stamphog digest-channels-retrieve`** - Per-audience Slack destinations for the daily merged-PR digest.
- **`posthog-pp-cli stamphog digest-channels-update`** - Per-audience Slack destinations for the daily merged-PR digest.
- **`posthog-pp-cli stamphog digest-runs-list`** - Read-only history of posted (or attempted) digests, filterable by digest channel.
- **`posthog-pp-cli stamphog digest-runs-retrieve`** - Read-only history of posted (or attempted) digests, filterable by digest channel.
- **`posthog-pp-cli stamphog pull-requests-list`** - Read-only pull requests stamphog knows about, filterable by PR number and merge state.
- **`posthog-pp-cli stamphog pull-requests-retrieve`** - Read-only pull requests stamphog knows about, filterable by PR number and merge state.
- **`posthog-pp-cli stamphog repo-configs-create`** - Per-repo stamphog settings — enable/disable review, GitHub App installation, policy overrides.
- **`posthog-pp-cli stamphog repo-configs-destroy`** - Per-repo stamphog settings — enable/disable review, GitHub App installation, policy overrides.
- **`posthog-pp-cli stamphog repo-configs-install-info-retrieve`** - Per-repo stamphog settings — enable/disable review, GitHub App installation, policy overrides.
- **`posthog-pp-cli stamphog repo-configs-list`** - Per-repo stamphog settings — enable/disable review, GitHub App installation, policy overrides.
- **`posthog-pp-cli stamphog repo-configs-partial-update`** - Per-repo stamphog settings — enable/disable review, GitHub App installation, policy overrides.
- **`posthog-pp-cli stamphog repo-configs-retrieve`** - Per-repo stamphog settings — enable/disable review, GitHub App installation, policy overrides.
- **`posthog-pp-cli stamphog repo-configs-sync-installation-create`** - Per-repo stamphog settings — enable/disable review, GitHub App installation, policy overrides.
- **`posthog-pp-cli stamphog repo-configs-update`** - Per-repo stamphog settings — enable/disable review, GitHub App installation, policy overrides.
- **`posthog-pp-cli stamphog review-runs-list`** - Read-only history of stamphog review runs, filterable by repository, PR number, and status.
- **`posthog-pp-cli stamphog review-runs-retrieve`** - Read-only history of stamphog review runs, filterable by repository, PR number, and status.

### streamlit-apps

Manage streamlit apps

- **`posthog-pp-cli streamlit-apps create`** - Create a streamlit app
- **`posthog-pp-cli streamlit-apps destroy`** - Delete a streamlit app
- **`posthog-pp-cli streamlit-apps list`** - List streamlit apps
- **`posthog-pp-cli streamlit-apps partial-update`** - Partially update a streamlit app
- **`posthog-pp-cli streamlit-apps retrieve`** - Retrieve a streamlit app
- **`posthog-pp-cli streamlit-apps update`** - Update a streamlit app

### subscriptions

Manage subscriptions

- **`posthog-pp-cli subscriptions create`** - Create
- **`posthog-pp-cli subscriptions destroy`** - Hard delete of this model is not allowed. Use a patch API call to set "deleted" to true
- **`posthog-pp-cli subscriptions list`** - List
- **`posthog-pp-cli subscriptions partial-update`** - Partial update
- **`posthog-pp-cli subscriptions retrieve`** - Retrieve
- **`posthog-pp-cli subscriptions summary-quota-retrieve`** - Summary quota retrieve
- **`posthog-pp-cli subscriptions update`** - Update

### surveys

Manage surveys

- **`posthog-pp-cli surveys all-activity-retrieve`** - All activity retrieve
- **`posthog-pp-cli surveys create`** - Create
- **`posthog-pp-cli surveys destroy`** - Destroy
- **`posthog-pp-cli surveys global-stats-retrieve`** - Get aggregated response statistics across all surveys.

Args:
    date_from: Optional ISO timestamp for start date (e.g. 2024-01-01T00:00:00Z)
    date_to: Optional ISO timestamp for end date (e.g. 2024-01-31T23:59:59Z)

Returns:
    Aggregated statistics across all surveys including total counts and rates
- **`posthog-pp-cli surveys list`** - List
- **`posthog-pp-cli surveys partial-update`** - Partial update
- **`posthog-pp-cli surveys question-labels`** - Return a slim list of question labels for the team's surveys. Used by the frontend to resolve `$survey_response_<question_id>` property keys into human-readable question text without loading the full survey payload.
- **`posthog-pp-cli surveys responses-count-retrieve`** - Get response counts for all surveys.

Args:
    exclude_archived: Optional boolean to exclude archived responses (default: false, includes archived)
    survey_ids: Optional comma-separated list of survey IDs to filter by

Returns:
    Dictionary mapping survey IDs to response counts
- **`posthog-pp-cli surveys retrieve`** - Retrieve
- **`posthog-pp-cli surveys update`** - Update

### taggers

Manage taggers

- **`posthog-pp-cli taggers create`** - Create
- **`posthog-pp-cli taggers destroy`** - Hard delete of this model is not allowed. Use a patch API call to set "deleted" to true
- **`posthog-pp-cli taggers list`** - List
- **`posthog-pp-cli taggers partial-update`** - Partial update
- **`posthog-pp-cli taggers retrieve`** - Retrieve
- **`posthog-pp-cli taggers test-hog-create`** - Test Hog tagger code against sample events without saving.
- **`posthog-pp-cli taggers update`** - Update

### task-activity

Manage task activity

- **`posthog-pp-cli task-activity list`** - Task lifecycle rows collapse per task. Comment notifications remain separate. Results are most-recent first and restricted to tasks the requester can see.
- **`posthog-pp-cli task-activity mark-read-create`** - Clear collapsed task activity through task timestamps and individual comment activity through activity IDs.

### task-automations

Manage task automations

- **`posthog-pp-cli task-automations create`** - API for managing scheduled task automations.
- **`posthog-pp-cli task-automations destroy`** - API for managing scheduled task automations.
- **`posthog-pp-cli task-automations list`** - API for managing scheduled task automations.
- **`posthog-pp-cli task-automations partial-update`** - API for managing scheduled task automations.
- **`posthog-pp-cli task-automations retrieve`** - API for managing scheduled task automations.

### task-channels

Manage task channels

- **`posthog-pp-cli task-channels create`** - Returns the existing public channel with the (normalized) name, creating it if needed.
- **`posthog-pp-cli task-channels destroy`** - API for task channels — the shared feeds tasks are kicked off in. Listing lazily
provisions the requester's personal "#me" channel; creation is resolve-or-create
by normalized name so clients can map channel-like surfaces onto backend channels.
- **`posthog-pp-cli task-channels list`** - All live public channels plus the requester's personal #me channel (created on first list).
- **`posthog-pp-cli task-channels partial-update`** - API for task channels — the shared feeds tasks are kicked off in. Listing lazily
provisions the requester's personal "#me" channel; creation is resolve-or-create
by normalized name so clients can map channel-like surfaces onto backend channels.
- **`posthog-pp-cli task-channels retrieve`** - API for task channels — the shared feeds tasks are kicked off in. Listing lazily
provisions the requester's personal "#me" channel; creation is resolve-or-create
by normalized name so clients can map channel-like surfaces onto backend channels.

### task-mentions

Manage task mentions

- **`posthog-pp-cli task-mentions <project_id>`** - Thread messages that @-mention the requester, newest first, restricted to tasks they can see.

### tasks

Manage tasks

- **`posthog-pp-cli tasks active-wizard-run-retrieve`** - Returns the most recent onboarding wizard cloud run for the current project when it is still running (or completed within the last day), so the setup-progress FAB can rehydrate after a drop-flow signup that started the run server-side. Returns 204 when there is none.
- **`posthog-pp-cli tasks create`** - API for managing tasks within a project. Tasks represent units of work to be performed by an agent.
- **`posthog-pp-cli tasks destroy`** - API for managing tasks within a project. Tasks represent units of work to be performed by an agent.
- **`posthog-pp-cli tasks list`** - Get a list of tasks for the current project, with optional filtering by origin product, stage, organization, repository, and created_by.
- **`posthog-pp-cli tasks partial-update`** - API for managing tasks within a project. Tasks represent units of work to be performed by an agent.
- **`posthog-pp-cli tasks pinned-retrieve`** - Return the visible tasks pinned by the requester in the current project.
- **`posthog-pp-cli tasks repositories-retrieve`** - Return the set of repositories referenced by non-deleted, non-internal tasks in the current project. Used to populate repository filter pickers without being constrained by task list pagination.
- **`posthog-pp-cli tasks repository-readiness-retrieve`** - Get autonomy readiness details for a specific repository in the current project.
- **`posthog-pp-cli tasks retrieve`** - Retrieve a single task by ID.
- **`posthog-pp-cli tasks slack-thread-context-retrieve`** - PostHog-internal debug tool. Resolves a Slack permalink to the linked task, its runs, the task-processing and mention-dispatch Temporal workflow ids/URLs, and presigned log URLs.
- **`posthog-pp-cli tasks summaries-create`** - Returns summary for the requested tasks: `id`, `title`, `repository`, `created_at`, `updated_at`, and the latest run's `status` and `environment`.
- **`posthog-pp-cli tasks update`** - API for managing tasks within a project. Tasks represent units of work to be performed by an agent.
- **`posthog-pp-cli tasks warm-create`** - Warm a full idling Run for a Code-app cloud task while the user composes: boot a sandbox, clone the repo, check out the branch, and start the agent, then idle awaiting the first message. On submit the normal create+run path transparently reuses and activates this Run; abandoned warms are reaped by the Run's inactivity timeout. Best-effort: returns an empty body when the feature flag is off, the warm pool is full, or the GitHub integration doesn't belong to the team.

### tracing

Manage tracing

- **`posthog-pp-cli tracing spans-aggregate-create`** - Spans aggregate create
- **`posthog-pp-cli tracing spans-attribute-breakdown-create`** - Spans attribute breakdown create
- **`posthog-pp-cli tracing spans-attributes-retrieve`** - Spans attributes retrieve
- **`posthog-pp-cli tracing spans-count-create`** - Spans count create
- **`posthog-pp-cli tracing spans-duration-histogram-create`** - Spans duration histogram create
- **`posthog-pp-cli tracing spans-has-spans-retrieve`** - Spans has spans retrieve
- **`posthog-pp-cli tracing spans-latency-heatmap-create`** - Spans latency heatmap create
- **`posthog-pp-cli tracing spans-query-create`** - Spans query create
- **`posthog-pp-cli tracing spans-service-names-retrieve`** - Spans service names retrieve
- **`posthog-pp-cli tracing spans-sparkline-create`** - Spans sparkline create
- **`posthog-pp-cli tracing spans-symbol-stats-create`** - Spans symbol stats create
- **`posthog-pp-cli tracing spans-trace-create`** - Spans trace create
- **`posthog-pp-cli tracing spans-tree-create`** - Spans tree create
- **`posthog-pp-cli tracing spans-values-retrieve`** - Spans values retrieve
- **`posthog-pp-cli tracing views-create`** - Views create
- **`posthog-pp-cli tracing views-destroy`** - Views destroy
- **`posthog-pp-cli tracing views-list`** - Views list
- **`posthog-pp-cli tracing views-partial-update`** - Views partial update
- **`posthog-pp-cli tracing views-retrieve`** - Views retrieve
- **`posthog-pp-cli tracing views-update`** - Views update

### uploaded-media

Manage uploaded media

- **`posthog-pp-cli uploaded-media <project_id>`** - When object storage is available this API allows upload of media which can be used, for example, in text cards on dashboards.

    Uploaded media must have a content type beginning with 'image/' and be less than 4MB.

### user-home-settings

Manage user home settings

- **`posthog-pp-cli user-home-settings partial-update`** - Update the authenticated user's pinned sidebar tabs and/or homepage for the current team. Pass `@me` as the UUID. Send `tabs` to replace the pinned tab list, `homepage` to set the home destination (any PostHog URL — dashboard, insight, search results, scene). Either field may be omitted to leave it unchanged; sending `homepage: null` or `{}` clears the homepage.
- **`posthog-pp-cli user-home-settings retrieve`** - Get the authenticated user's pinned sidebar tabs and configured homepage for the current team. Pass `@me` as the UUID.

### user-interview-topics

Manage user interview topics

- **`posthog-pp-cli user-interview-topics create`** - Planned user interview topics: who we want to target and what we want to ask about.
- **`posthog-pp-cli user-interview-topics destroy`** - Planned user interview topics: who we want to target and what we want to ask about.
- **`posthog-pp-cli user-interview-topics list`** - Planned user interview topics: who we want to target and what we want to ask about.
- **`posthog-pp-cli user-interview-topics partial-update`** - Planned user interview topics: who we want to target and what we want to ask about.
- **`posthog-pp-cli user-interview-topics retrieve`** - Planned user interview topics: who we want to target and what we want to ask about.
- **`posthog-pp-cli user-interview-topics update`** - Planned user interview topics: who we want to target and what we want to ask about.

### user-interviews

Manage user interviews

- **`posthog-pp-cli user-interviews create`** - Create
- **`posthog-pp-cli user-interviews destroy`** - Destroy
- **`posthog-pp-cli user-interviews list`** - List
- **`posthog-pp-cli user-interviews partial-update`** - Partial update
- **`posthog-pp-cli user-interviews retrieve`** - Retrieve
- **`posthog-pp-cli user-interviews search-create`** - Embed `query` with the same model used to index interview transcripts and summaries, then return the top matches by cosine distance. Each match is a single (interview, document_type) pair — an interview can appear up to twice if both its transcript and summary score above other interviews. Useful for surfacing relevant interview snippets in natural language, without exact keyword matches.
- **`posthog-pp-cli user-interviews update`** - Update

### users

Manage users

- **`posthog-pp-cli users cancel-email-change-request-partial-update`** - Cancel email change request partial update
- **`posthog-pp-cli users destroy`** - Destroy
- **`posthog-pp-cli users list`** - List
- **`posthog-pp-cli users partial-update`** - Update one or more of the authenticated user's profile fields or settings.
- **`posthog-pp-cli users request-email-verification-create`** - Request email verification create
- **`posthog-pp-cli users retrieve`** - Retrieve a user's profile and settings. Pass `@me` as the UUID to fetch the authenticated user; non-staff callers may only access their own account.
- **`posthog-pp-cli users update`** - Replace the authenticated user's profile and settings. Pass `@me` as the UUID to update the authenticated user. Prefer the PATCH endpoint for partial updates — PUT requires every writable field to be provided.
- **`posthog-pp-cli users verify-email-create`** - Verify email create

### vision

Manage vision

- **`posthog-pp-cli vision actions-create`** - CRUD for Replay Vision actions — scheduled "and then…" automations over a scanner's observations.
- **`posthog-pp-cli vision actions-destroy`** - CRUD for Replay Vision actions — scheduled "and then…" automations over a scanner's observations.
- **`posthog-pp-cli vision actions-list`** - CRUD for Replay Vision actions — scheduled "and then…" automations over a scanner's observations.
- **`posthog-pp-cli vision actions-partial-update`** - CRUD for Replay Vision actions — scheduled "and then…" automations over a scanner's observations.
- **`posthog-pp-cli vision actions-retrieve`** - CRUD for Replay Vision actions — scheduled "and then…" automations over a scanner's observations.
- **`posthog-pp-cli vision actions-run-create`** - Run this summary now, without waiting for its schedule — synthesizes a group summary over the
observations since the last summary (or the last 24h). The recurring schedule is untouched: the
engine advances next_run_at only at scheduled claim time, never in the run itself.
- **`posthog-pp-cli vision actions-runs-list`** - Read-only run history for a single vision action (nested under /vision/actions/{action_id}/runs/).
- **`posthog-pp-cli vision actions-runs-retrieve`** - Read-only run history for a single vision action (nested under /vision/actions/{action_id}/runs/).
- **`posthog-pp-cli vision environment-quota-retrieve`** - Environment quota retrieve
- **`posthog-pp-cli vision observations-create-task-create`** - Create a PostHog Task from this observation's finding so it can be triaged and fixed. Title and description are derived from the scanner and its result. Record-only: this does not start the coding agent. Idempotent per observation: once a task exists, repeat calls return its id with a 200 instead of creating a duplicate.
- **`posthog-pp-cli vision observations-label-create`** - Set or update the observation's shared label: whether the scanner scored the session correctly, plus optional feedback on what it got wrong. One label per observation, shared across the team; these labels feed prompt improvement. Requires editor access to the scanner.
- **`posthog-pp-cli vision observations-label-destroy`** - Remove the observation's shared label. Requires editor access to the scanner.
- **`posthog-pp-cli vision observations-list`** - Read-only access to a session's observations across every scanner the caller can read, for the replay-page dock.
- **`posthog-pp-cli vision observations-retrieve`** - Retrieve one observation. Any list filters passed along (status, tags, order_by, …) scope the `previous_observation_id`/`next_observation_id` navigation to the matching, identically-ordered set — so prev/next from a filtered table stays within that filtered list.
- **`posthog-pp-cli vision observations-retry-create`** - Delete a failed or ineligible observation and re-run its scanner on the same recording. Returns 202 with the workflow handle.
- **`posthog-pp-cli vision scanners-affected-cohort-create`** - Save the users this scanner matched as a static cohort, for surveys, funnels, and retention analysis.
- **`posthog-pp-cli vision scanners-bulk-observe-create`** - Apply this scanner to many sessions on demand. Starts as many as fit under the in-flight
caps and monthly credit quota, reporting the rest as skipped rather than failing the batch.
- **`posthog-pp-cli vision scanners-create`** - CRUD for Replay Vision scanners.
- **`posthog-pp-cli vision scanners-creators-retrieve`** - Distinct creators across the team's scanners — feeds the `Created by` filter dropdown.
- **`posthog-pp-cli vision scanners-destroy`** - CRUD for Replay Vision scanners.
- **`posthog-pp-cli vision scanners-estimate-create`** - Estimate the observation volume a proposed scanner would generate, for the pre-save cost preview.
- **`posthog-pp-cli vision scanners-impact-retrieve`** - Affected sessions and users for this scanner over the trailing window.
- **`posthog-pp-cli vision scanners-inline-scan-create`** - Scan named sessions against a prompt without saving a scanner first, for one-off questions.

The config resolves to a scanner minted on first use, so asking the same question twice reuses
the observations it already has, while a different question about the same session gets its own.
- **`posthog-pp-cli vision scanners-list`** - CRUD for Replay Vision scanners.
- **`posthog-pp-cli vision scanners-observations-create-task-create`** - Create a PostHog Task from this observation's finding so it can be triaged and fixed. Title and description are derived from the scanner and its result. Record-only: this does not start the coding agent. Idempotent per observation: once a task exists, repeat calls return its id with a 200 instead of creating a duplicate.
- **`posthog-pp-cli vision scanners-observations-label-create`** - Set or update the observation's shared label: whether the scanner scored the session correctly, plus optional feedback on what it got wrong. One label per observation, shared across the team; these labels feed prompt improvement. Requires editor access to the scanner.
- **`posthog-pp-cli vision scanners-observations-label-destroy`** - Remove the observation's shared label. Requires editor access to the scanner.
- **`posthog-pp-cli vision scanners-observations-list`** - Read-only access to observations produced by a scanner.
- **`posthog-pp-cli vision scanners-observations-retrieve`** - Retrieve one observation. Any list filters passed along (status, tags, order_by, …) scope the `previous_observation_id`/`next_observation_id` navigation to the matching, identically-ordered set — so prev/next from a filtered table stays within that filtered list.
- **`posthog-pp-cli vision scanners-observations-retry-create`** - Delete a failed or ineligible observation and re-run its scanner on the same recording. Returns 202 with the workflow handle.
- **`posthog-pp-cli vision scanners-observations-stats-retrieve`** - Aggregate counts and per-scanner-type distributions over the filtered observation set. Same filters as the list endpoint apply.
- **`posthog-pp-cli vision scanners-observe-create`** - Apply this scanner to one specific session, on demand. Returns 202 with the workflow handle.
- **`posthog-pp-cli vision scanners-partial-update`** - CRUD for Replay Vision scanners.
- **`posthog-pp-cli vision scanners-prompt-suggestions-apply-create`** - Apply this suggestion: write a config to the scanner (the prompt plus any type-specific config such as classifier tags or the monitor allow_inconclusive flag), bumping the scanner version, and mark the suggestion applied. Pass `config` to apply an edited subset of the recommendation; omit it to apply the full suggested config. Only the current pending suggestion can be applied. Requires session recording edit access.
- **`posthog-pp-cli vision scanners-prompt-suggestions-current-retrieve`** - The scanner's newest prompt suggestion plus whether it is stale (the ratings changed since it was generated) and how many rated observations are available.
- **`posthog-pp-cli vision scanners-prompt-suggestions-dismiss-create`** - Dismiss this suggestion without applying it. Only the current pending suggestion can be dismissed. Requires editor access to the scanner.
- **`posthog-pp-cli vision scanners-prompt-suggestions-evaluate-create`** - Test this suggestion before applying it: re-run the scanner with the suggested prompt against already-rated sessions in the background and compare each fresh output with the stored one. Results land on the suggestion's `evaluation` field. Poll `current` while status is running. `session_limit` controls how many rated sessions are re-run (thumbs-down prioritized, up to `evaluation_session_cap`). Each successful re-run charges credits like a normal observation of the same model. The request is refused with 402 when the planned credits exceed what is left of the monthly limit. Monitor and classifier scanners get a kept/fixed/regressed classification, while scorer and summarizer scanners show the raw before and after output. Requires session recording edit access.
- **`posthog-pp-cli vision scanners-prompt-suggestions-generate-create`** - Generate a fresh prompt suggestion from the team's current ratings. The previous pending suggestion becomes history (superseded). Requires at least one rated observation and editor access to the scanner.
- **`posthog-pp-cli vision scanners-prompt-suggestions-list`** - AI prompt-rewrite suggestions for a scanner, generated from the team's thumbs up/down ratings.
- **`posthog-pp-cli vision scanners-retrieve`** - CRUD for Replay Vision scanners.
- **`posthog-pp-cli vision scanners-stats-retrieve`** - Team-wide scanner counts — independent of list filters, so the overview stays stable.
- **`posthog-pp-cli vision scanners-suggest-tags-create`** - Suggest classifier tags grounded in the scanner's own observations and the org's product data.

### visual-review

Manage visual review

- **`posthog-pp-cli visual-review repos-baselines-retrieve`** - Snapshots overview for a repo: every identifier with a current baseline (latest non-superseded master/main run per run_type), plus tolerate counts, active quarantine state, and a 30-day stability sparkline. Capped at 5000 entries — sets `truncated` and returns the most recently active when exceeded. Filtering / faceting / search are all done client-side; this endpoint takes no filter query params.
- **`posthog-pp-cli visual-review repos-create`** - Create a new repo.
- **`posthog-pp-cli visual-review repos-list`** - List all projects for the team.
- **`posthog-pp-cli visual-review repos-partial-update`** - Update a repo's settings.
- **`posthog-pp-cli visual-review repos-quarantine-create`** - Quarantine a snapshot identifier for a specific run type.
- **`posthog-pp-cli visual-review repos-quarantine-expire-create`** - Expire all active quarantine entries for an identifier.
- **`posthog-pp-cli visual-review repos-quarantine-list`** - List quarantined identifiers. Without filter: active only. With identifier: full history.
- **`posthog-pp-cli visual-review repos-retrieve`** - Get a repo by ID.
- **`posthog-pp-cli visual-review repos-runs-counts-retrieve`** - Review state counts for runs in this repo.
- **`posthog-pp-cli visual-review repos-runs-list`** - List runs in this repo, optionally filtered by review state and free-text search.
- **`posthog-pp-cli visual-review repos-snapshots-list`** - Deduped baseline timeline for a snapshot identity. Newest first.
- **`posthog-pp-cli visual-review repos-thumbnails-retrieve`** - Serve a snapshot thumbnail by identifier. Returns WebP with ETag caching.
- **`posthog-pp-cli visual-review runs-add-snapshots-create`** - Add a batch of snapshots to a pending run (shard-based flow).
- **`posthog-pp-cli visual-review runs-approve-create`** - Mark snapshots reviewed (DB only).

Records the per-snapshot "Accept change" decision. Does not commit the baseline
or change the GitHub gate — call finalize to ship the run.
- **`posthog-pp-cli visual-review runs-complete-create`** - Complete a run: detect removals, verify uploads, trigger diff processing.
- **`posthog-pp-cli visual-review runs-counts-retrieve`** - Review state counts for the runs list.
- **`posthog-pp-cli visual-review runs-create`** - Create a new run from a CI manifest.
- **`posthog-pp-cli visual-review runs-finalize-create`** - Finalize a fully-reviewed run: commit the approved baseline and green the gate.

Commits exactly the snapshots approved in the DB (tolerated ones keep their baseline)
and only succeeds once every changed/new snapshot is resolved. With approve_all=true,
any still-pending changed/new snapshot is approved first. With commit_to_github=false
the server returns the signed baseline YAML instead of committing it.
- **`posthog-pp-cli visual-review runs-list`** - List runs for the team, optionally filtered by review state, PR number, commit SHA, branch, or free-text search.
- **`posthog-pp-cli visual-review runs-recompute-create`** - Re-evaluate quarantine and counts, update commit status, and optionally rerun the CI job.
- **`posthog-pp-cli visual-review runs-retrieve`** - Get run status and summary.
- **`posthog-pp-cli visual-review runs-snapshot-history-list`** - Recent change history for a snapshot identifier across runs.
- **`posthog-pp-cli visual-review runs-snapshots-list`** - Get a run's snapshots with diff results, excluding quarantined ones by default.
- **`posthog-pp-cli visual-review runs-tolerate-create`** - Mark a changed snapshot as a known tolerated alternate.
- **`posthog-pp-cli visual-review runs-tolerated-hashes-list`** - List known tolerated hashes for a snapshot identifier.

### warehouse-column-annotations

Manage warehouse column annotations

- **`posthog-pp-cli warehouse-column-annotations create`** - Read and edit semantic descriptions of warehouse tables and columns surfaced to the AI agent.

List can be filtered to one table with `?table_id=<uuid>`. Any create or update is treated as a
user edit (`is_user_edited=True`), which protects the row from being overwritten by automatic
enrichment. Create upserts on `(table, column_name)`; the table cannot be changed after creation.
- **`posthog-pp-cli warehouse-column-annotations destroy`** - Read and edit semantic descriptions of warehouse tables and columns surfaced to the AI agent.

List can be filtered to one table with `?table_id=<uuid>`. Any create or update is treated as a
user edit (`is_user_edited=True`), which protects the row from being overwritten by automatic
enrichment. Create upserts on `(table, column_name)`; the table cannot be changed after creation.
- **`posthog-pp-cli warehouse-column-annotations list`** - Read and edit semantic descriptions of warehouse tables and columns surfaced to the AI agent.

List can be filtered to one table with `?table_id=<uuid>`. Any create or update is treated as a
user edit (`is_user_edited=True`), which protects the row from being overwritten by automatic
enrichment. Create upserts on `(table, column_name)`; the table cannot be changed after creation.
- **`posthog-pp-cli warehouse-column-annotations partial-update`** - Read and edit semantic descriptions of warehouse tables and columns surfaced to the AI agent.

List can be filtered to one table with `?table_id=<uuid>`. Any create or update is treated as a
user edit (`is_user_edited=True`), which protects the row from being overwritten by automatic
enrichment. Create upserts on `(table, column_name)`; the table cannot be changed after creation.
- **`posthog-pp-cli warehouse-column-annotations retrieve`** - Read and edit semantic descriptions of warehouse tables and columns surfaced to the AI agent.

List can be filtered to one table with `?table_id=<uuid>`. Any create or update is treated as a
user edit (`is_user_edited=True`), which protects the row from being overwritten by automatic
enrichment. Create upserts on `(table, column_name)`; the table cannot be changed after creation.
- **`posthog-pp-cli warehouse-column-annotations update`** - Read and edit semantic descriptions of warehouse tables and columns surfaced to the AI agent.

List can be filtered to one table with `?table_id=<uuid>`. Any create or update is treated as a
user edit (`is_user_edited=True`), which protects the row from being overwritten by automatic
enrichment. Create upserts on `(table, column_name)`; the table cannot be changed after creation.

### warehouse-column-statistics

Manage warehouse column statistics

- **`posthog-pp-cli warehouse-column-statistics list`** - Read per-column data statistics (null fraction, min/max, row count) for warehouse tables.

Statistics are computed automatically after a sync and surfaced to the AI agent so it can write
better queries. They are system-owned and read-only here. List can be filtered to one table with
`?table_id=<uuid>`.
- **`posthog-pp-cli warehouse-column-statistics retrieve`** - Read per-column data statistics (null fraction, min/max, row count) for warehouse tables.

Statistics are computed automatically after a sync and surfaced to the AI agent so it can write
better queries. They are system-owned and read-only here. List can be filtered to one table with
`?table_id=<uuid>`.

### warehouse-dag

Manage warehouse dag

- **`posthog-pp-cli warehouse-dag <project_id>`** - Return this team's DAG as a set of edges and nodes

### warehouse-expressions

Manage warehouse expressions

- **`posthog-pp-cli warehouse-expressions create`** - Create, read, update and delete saved HogQL expressions that appear as virtual fields on tables.
- **`posthog-pp-cli warehouse-expressions destroy`** - Create, read, update and delete saved HogQL expressions that appear as virtual fields on tables.
- **`posthog-pp-cli warehouse-expressions list`** - Create, read, update and delete saved HogQL expressions that appear as virtual fields on tables.
- **`posthog-pp-cli warehouse-expressions partial-update`** - Create, read, update and delete saved HogQL expressions that appear as virtual fields on tables.
- **`posthog-pp-cli warehouse-expressions retrieve`** - Create, read, update and delete saved HogQL expressions that appear as virtual fields on tables.
- **`posthog-pp-cli warehouse-expressions update`** - Create, read, update and delete saved HogQL expressions that appear as virtual fields on tables.

### warehouse-model-paths

Manage warehouse model paths

- **`posthog-pp-cli warehouse-model-paths list`** - List
- **`posthog-pp-cli warehouse-model-paths retrieve`** - Retrieve

### warehouse-saved-queries

Manage warehouse saved queries

- **`posthog-pp-cli warehouse-saved-queries create`** - Create, Read, Update and Delete Warehouse Tables.
- **`posthog-pp-cli warehouse-saved-queries destroy`** - Create, Read, Update and Delete Warehouse Tables.
- **`posthog-pp-cli warehouse-saved-queries list`** - Create, Read, Update and Delete Warehouse Tables.
- **`posthog-pp-cli warehouse-saved-queries partial-update`** - Create, Read, Update and Delete Warehouse Tables.
- **`posthog-pp-cli warehouse-saved-queries resume-schedules-create`** - Resume paused materialization schedules for multiple matviews.

Accepts a list of view IDs in the request body: {"view_ids": ["id1", "id2", ...]}
This endpoint is idempotent - calling it on already running or non-existent schedules is safe.
- **`posthog-pp-cli warehouse-saved-queries retrieve`** - Create, Read, Update and Delete Warehouse Tables.
- **`posthog-pp-cli warehouse-saved-queries update`** - Create, Read, Update and Delete Warehouse Tables.

### warehouse-saved-query-folders

Manage warehouse saved query folders

- **`posthog-pp-cli warehouse-saved-query-folders create`** - Create
- **`posthog-pp-cli warehouse-saved-query-folders destroy`** - Destroy
- **`posthog-pp-cli warehouse-saved-query-folders list`** - List
- **`posthog-pp-cli warehouse-saved-query-folders partial-update`** - Partial update
- **`posthog-pp-cli warehouse-saved-query-folders retrieve`** - Retrieve

### warehouse-tables

Manage warehouse tables

- **`posthog-pp-cli warehouse-tables create`** - Create, Read, Update and Delete Warehouse Tables.
- **`posthog-pp-cli warehouse-tables create-from-upload-create`** - Turn a previously uploaded file into a self-managed warehouse table.

The file already sits in PostHog's own bucket (see `upload_file`), so the table points straight
at it and is read in place — no import pipeline and no recurring sync, the same shape as a linked
S3/GCS bucket. The read location is always derived from the caller's own team, so a client-supplied
`upload_id` can only resolve inside that team's folder, and the table carries no credential (reads
fall back to the node role, never a user-supplied key).
- **`posthog-pp-cli warehouse-tables destroy`** - Create, Read, Update and Delete Warehouse Tables.
- **`posthog-pp-cli warehouse-tables file-create`** - Create, Read, Update and Delete Warehouse Tables.
- **`posthog-pp-cli warehouse-tables list`** - Create, Read, Update and Delete Warehouse Tables.
- **`posthog-pp-cli warehouse-tables partial-update`** - Create, Read, Update and Delete Warehouse Tables.
- **`posthog-pp-cli warehouse-tables retrieve`** - Create, Read, Update and Delete Warehouse Tables.
- **`posthog-pp-cli warehouse-tables update`** - Create, Read, Update and Delete Warehouse Tables.
- **`posthog-pp-cli warehouse-tables upload-file-create`** - Store an uploaded file in object storage so a self-managed table can be created from it.

Uploading is a separate first step from `create_from_upload` so the create call stays JSON-only:
this returns an `upload_id` the caller passes back to build the table. The file is written under
a team-scoped prefix, so a table can only ever read back its own team's uploads.

### warehouse-view-link

Manage warehouse view link

- **`posthog-pp-cli warehouse-view-link create`** - Create, Read, Update and Delete View Columns.
- **`posthog-pp-cli warehouse-view-link destroy`** - Create, Read, Update and Delete View Columns.
- **`posthog-pp-cli warehouse-view-link list`** - Create, Read, Update and Delete View Columns.
- **`posthog-pp-cli warehouse-view-link partial-update`** - Create, Read, Update and Delete View Columns.
- **`posthog-pp-cli warehouse-view-link retrieve`** - Create, Read, Update and Delete View Columns.
- **`posthog-pp-cli warehouse-view-link update`** - Create, Read, Update and Delete View Columns.
- **`posthog-pp-cli warehouse-view-link validate-create`** - Create, Read, Update and Delete View Columns.

### warehouse-view-links

Manage warehouse view links

- **`posthog-pp-cli warehouse-view-links create`** - Create, Read, Update and Delete View Columns.
- **`posthog-pp-cli warehouse-view-links destroy`** - Create, Read, Update and Delete View Columns.
- **`posthog-pp-cli warehouse-view-links list`** - Create, Read, Update and Delete View Columns.
- **`posthog-pp-cli warehouse-view-links partial-update`** - Create, Read, Update and Delete View Columns.
- **`posthog-pp-cli warehouse-view-links retrieve`** - Create, Read, Update and Delete View Columns.
- **`posthog-pp-cli warehouse-view-links update`** - Create, Read, Update and Delete View Columns.
- **`posthog-pp-cli warehouse-view-links validate-create`** - Create, Read, Update and Delete View Columns.

### web-analytics

Manage web analytics

- **`posthog-pp-cli web-analytics recap`** - The 'Wrapped'-style weekly recap: everything in the weekly digest (visitors, pageviews, sessions, bounce rate, average session duration with period-over-period comparisons, top pages, top sources, and goals) plus a single derived weekly persona and a short list of screenshot-worthy highlights for the period.
- **`posthog-pp-cli web-analytics weekly-digest`** - Summarizes a project's web analytics over a lookback window (default 7 days): unique visitors, pageviews, sessions, bounce rate, and average session duration with period-over-period comparisons, plus the top 5 pages, top 5 traffic sources, and goal conversions.

### web-analytics-achievements

Manage web analytics achievements

- **`posthog-pp-cli web-analytics-achievements acknowledge-celebration`** - Clears a pending celebration for the given track and stage once the client has shown it, so it isn't celebrated again. Idempotent.
- **`posthog-pp-cli web-analytics-achievements overview`** - Returns the achievement track definitions (thresholds resolved for the requesting user's streak-cadence arm), the user's and team's progress, and any newly unlocked stages awaiting an in-session celebration.
- **`posthog-pp-cli web-analytics-achievements preferences`** - Returns the requesting user's per-project Web analytics achievements preferences.
- **`posthog-pp-cli web-analytics-achievements record-interaction`** - Idempotently increments the requesting user's first-party counter for an in-product Web analytics interaction (slicing data, or opening a session recording), which drives the Explorer and Detective achievement tracks.
- **`posthog-pp-cli web-analytics-achievements record-visit`** - Idempotently records that the requesting user opened Web analytics today (team-local date) and schedules a debounced achievement recompute. Intended to be called once per session.
- **`posthog-pp-cli web-analytics-achievements update-preferences`** - Sets the requesting user's per-project Web analytics achievements preferences.

### web-analytics-path-cleaning-suggestions

Manage web analytics path cleaning suggestions

- **`posthog-pp-cli web-analytics-path-cleaning-suggestions <project_id>`** - Samples the team's recent paths, asks the LLM for cleaning rules, validates them against the real paths, and stores the result as a `path_cleaning_suggestions` health issue (replacing any previous active one). Runs even if the team already has rules. Returns the suggestion (or a skip status when there aren't enough paths to suggest from).

### web-experiments

Manage web experiments

- **`posthog-pp-cli web-experiments create`** - Create
- **`posthog-pp-cli web-experiments destroy`** - Destroy
- **`posthog-pp-cli web-experiments list`** - List
- **`posthog-pp-cli web-experiments partial-update`** - Partial update
- **`posthog-pp-cli web-experiments retrieve`** - Retrieve
- **`posthog-pp-cli web-experiments update`** - Update

### web-vitals

Manage web vitals

- **`posthog-pp-cli web-vitals <project_id>`** - Get web vitals for a specific pathname.
Toolbar accesses this via OAuth (handled by TeamAndOrgViewSetMixin.get_authenticators).

### wizard

Manage wizard

- **`posthog-pp-cli wizard sessions-create`** - Upsert a wizard session. The `session_id` key is the idempotency anchor — reposting the same `session_id` replaces the existing row. Returns 201 on create, 200 on update.
- **`posthog-pp-cli wizard sessions-latest-retrieve`** - Return the single most-recent wizard session for a workflow (and optional skill), or 204 if none exists. Unlike `list`, this is a point lookup the app shell uses to decide whether to open the live SSE stream — it never returns a collection, and 'no run' is a 204 rather than a 404 so clients don't conflate it with a missing endpoint.
- **`posthog-pp-cli wizard sessions-list`** - List wizard sessions for the project, ordered by started_at desc. This should only be called by the PostHog Wizard. Optional filters: ?workflow_id=<id> and ?skill_id=<id>.
- **`posthog-pp-cli wizard sessions-retrieve`** - Retrieve a single wizard session by its session_id.
- **`posthog-pp-cli wizard sessions-stream-retrieve`** - Server-Sent Events stream of wizard session updates for a (workflow_id, skill_id) pair. On connect, the current latest session (if any) is emitted as the first event; subsequent upserts are streamed in real time. The server closes the connection after 900 seconds with an `event: end` line so the client (EventSource) can reconnect.

**SDK consumers**: do not call the generated fetch wrapper for this path — it will buffer the entire infinite stream. Use the URL builder (`getWizardSessionsStreamRetrieveUrl`) with the browser's `EventSource` API instead.


### Self-learning loop

This CLI caches per-question discovery so repeat queries skip the walk and structurally similar queries get answered via entity substitution. The loop also self-captures: every invocation is journaled locally, and failed-flag corrections plus fresh teaches surface as candidates on the next `recall` for confirm/reject judgment. Agents call `recall` before discovery and fire `teach &` after answering. See the `## Automatic learning` section in `SKILL.md` for the full protocol.

- **`posthog-pp-cli recall <query>`** - Look up cached resources for a query before running discovery
- **`posthog-pp-cli teach`** - Record a query -> resource mapping (silent on success, safe to background with `&`)
- **`posthog-pp-cli learnings list`** - Inspect taught rows
- **`posthog-pp-cli learnings forget <query>`** - Undo a teach
- **`posthog-pp-cli learnings candidates`** - List auto-captured candidates awaiting confirm/reject
- **`posthog-pp-cli learnings stats`** - Local loop metrics: recall hit rate, teach-to-reuse, playbook resolution, candidate counts
- **`posthog-pp-cli teach-pattern`** - Install a query/resource template up front
- **`posthog-pp-cli teach-lookup`** - Add an entity mapping (e.g. country code, team alias) for pattern substitution

Pass `--no-learn` or set `POSTHOG_NO_LEARN=true` to disable the loop for deterministic flows.

The local store's schema version stamp is one-way: once this version of `posthog-pp-cli` opens the database, older binaries refuse it with a version error — upgrade the binary rather than downgrading.

## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
posthog-pp-cli account-notes mock-value

# JSON for scripting and agents
posthog-pp-cli account-notes mock-value --json

# Filter to specific fields
posthog-pp-cli account-notes mock-value --json --select id,name,status

# Dry run — show the request without sending
posthog-pp-cli account-notes mock-value --dry-run

# Agent mode — JSON + compact + no prompts in one flag
posthog-pp-cli account-notes mock-value --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select id,name` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Explicit retries** - add `--idempotent` to create retries and add `--ignore-missing` to delete retries when a no-op success is acceptable
- **Confirmable** - `--yes` for explicit confirmation of destructive actions
- **Piped input** - write commands can accept structured input when their help lists `--stdin`
- **Offline-friendly** - sync/search commands can use the local SQLite store when available
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `4` auth error, `5` API error, `7` rate limited, `10` config error.

## Health Check

```bash
posthog-pp-cli doctor
```

Verifies configuration, credentials, and connectivity to the API.

## Configuration

Run `posthog-pp-cli doctor` to see the resolved config, data, state, and cache directories. The platform-default config path is `~/.config/posthog-pp-cli/config.toml`; `--home`, `POSTHOG_HOME`, and per-kind env vars can relocate it.

Static request headers can be configured under `headers`; per-command header overrides take precedence.

Environment variables:

| Name | Kind | Required | Description |
| --- | --- | --- | --- |
| `POSTHOG_PERSONAL_APIKEY_AUTH` | per_call | Yes | Set to your API credential. |

### agentcookie (optional)

If you use agentcookie to sync secrets across machines, this CLI auto-adopts agentcookie-managed credentials with no extra setup. When the daemon writes to this CLI's config, `posthog-pp-cli doctor` reports `agentcookie: detected` and `auth-status` labels the source as `agentcookie`. Skip this section if you don't use agentcookie - the CLI works the same as any other.

## Troubleshooting
**Authentication errors (exit code 4)**
- Run `posthog-pp-cli doctor` to check credentials
- Verify the environment variable is set: `echo $POSTHOG_PERSONAL_APIKEY_AUTH`
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

---

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
