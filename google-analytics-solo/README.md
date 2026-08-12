# Google Analytics CLI

**Flag-driven GA4 reports, one-command fan-out across every property, and a local cache for offline trends — none of which the raw Data API gives you.**

The GA4 Data API is POST-with-hand-written-JSON report bodies against one numeric property at a time. This CLI turns reports into flags (--dims, --metrics, --since), fans a single report out across all your registered properties, caches results in SQLite for offline diffing, covers the full GA4 Admin API (accounts, properties, data streams, custom dimensions, audiences), and adds Measurement Protocol event ingestion. Mutating Admin/send operations are gated behind --confirm. It speaks --json/--select for agents.

## Install

The recommended path installs both the `google-analytics-solo-pp-cli` binary and the `pp-google-analytics-solo` agent skill (Claude Code, Codex, Cursor, Gemini CLI, GitHub Copilot, and other agents supported by the upstream [`skills`](https://github.com/vercel-labs/skills) CLI) in one shot:

```bash
npx -y @mvanhorn/printing-press-library install google-analytics-solo
```

For CLI only (no skill):

```bash
npx -y @mvanhorn/printing-press-library install google-analytics-solo --cli-only
```

For skill only — installs the skill into the same agents as the default command above, but skips the CLI binary (use this to update or reinstall just the skill):

```bash
npx -y @mvanhorn/printing-press-library install google-analytics-solo --skill-only
```

To constrain the skill install to one or more specific agents (repeatable — agent names match the [`skills`](https://github.com/vercel-labs/skills) CLI):

```bash
npx -y @mvanhorn/printing-press-library install google-analytics-solo --agent claude-code
npx -y @mvanhorn/printing-press-library install google-analytics-solo --agent claude-code --agent codex
```

### Without Node (Go fallback)

If `npx` isn't available (no Node, offline), install the CLI directly via Go (requires Go 1.26.5 or newer):

```bash
go install github.com/mvanhorn/printing-press-library/library/marketing/google-analytics-solo/cmd/google-analytics-solo-pp-cli@latest
```

This installs the CLI only — no skill.

### Pre-built binary

Download a pre-built binary for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/google-analytics-solo-current). On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine <binary>`. On Unix, mark it executable: `chmod +x <binary>`.

<!-- pp-hermes-install-anchor -->
## Install for Hermes

Install the CLI binary first. The installer writes binaries to a per-user managed bin directory by default: `$HOME/.local/bin` on macOS/Linux and `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows.

```bash
npx -y @mvanhorn/printing-press-library install google-analytics-solo --cli-only
```

Then install the focused Hermes skill.

From the Hermes CLI:

```bash
hermes skills install mvanhorn/printing-press-library/cli-skills/pp-google-analytics-solo --force
```

Inside a Hermes chat session:

```bash
/skills install mvanhorn/printing-press-library/cli-skills/pp-google-analytics-solo --force
```

Restart the Hermes session or gateway if the newly installed skill is not visible immediately.

## Install for OpenClaw
Install both the CLI binary and the focused OpenClaw skill. The installer defaults binaries to a per-user bin directory (`$HOME/.local/bin` on macOS/Linux, `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows):

```bash
npx -y @mvanhorn/printing-press-library install google-analytics-solo --agent openclaw
```

Restart the OpenClaw session or gateway if the newly installed skill is not visible immediately.

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/google-analytics-solo-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.
3. Fill in `GOOGLE_APPLICATION_CREDENTIALS` when Claude Desktop prompts you.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.


```bash
go install github.com/mvanhorn/printing-press-library/library/marketing/google-analytics-solo/cmd/google-analytics-solo-pp-mcp@latest
```

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "google-analytics-solo": {
      "command": "google-analytics-solo-pp-mcp",
      "env": {
        "GOOGLE_ANALYTICS_SOLO_PARENT": "<parent>",
        "GOOGLE_APPLICATION_CREDENTIALS": "<your-key>"
      }
    }
  }
}
```

</details>

## Authentication

Authenticates to Google through Application Default Credentials, which unifies three identity modes: a service-account key (GOOGLE_APPLICATION_CREDENTIALS), a gcloud user login (run 'gcloud auth application-default login'), or workload identity on GCP. A stored OAuth refresh token is a fallback. Scope is read-only analytics; write operations still require the --confirm flag. Measurement Protocol commands use a separate GA4_MP_API_SECRET, not OAuth.

## Quick Start

```bash
# check credentials and config before anything else
google-analytics-solo-pp-cli doctor --dry-run

# auto-register every property your account can see
google-analytics-solo-pp-cli alias discover

# run your first report by friendly name
google-analytics-solo-pp-cli report --property solo-prod --dims date --metrics activeUsers --since 7d --json

```

## Unique Features

These capabilities aren't available in any other tool for this API.

### Multi-property
- **`report`** — Run one report against every registered property at once and get per-property results.

  _Reach for this when an agent needs the same metric across every Solo property in one call instead of six._

  ```bash
  google-analytics-solo-pp-cli report --all-properties --dims date --metrics activeUsers --since 7d --json
  ```
- **`compare`** — Rank all registered properties by a single metric over a date range.

  _Use when the question is 'which property leads on metric X', not single-property detail._

  ```bash
  google-analytics-solo-pp-cli compare --metric activeUsers --since 30d --json
  ```
- **`properties add`** — Register numeric property IDs under friendly names once, then reference by name everywhere.

  _Use once at setup so every later command can say --property solo-prod._

  ```bash
  google-analytics-solo-pp-cli properties add solo-prod 123456789
  ```

### Local state that compounds
- **`trend`** — Show how a metric moved over time using locally cached report runs, no re-query.

  _Use for offline trend analysis after reports have been cached locally._

  ```bash
  google-analytics-solo-pp-cli trend activeUsers --property solo-prod --since 90d --json
  ```
- **`report`** — Save a report spec by name and replay it without retyping dimensions and metrics.

  _Use to replay common reports; agents can reference a named report instead of a full spec._

  ```bash
  google-analytics-solo-pp-cli report --run weekly-actives --property solo-prod --json
  ```
- **`metadata search`** — Full-text search the cached dimension/metric catalog to find exact API names.

  _Use to find the exact dimension/metric API name before building a report._

  ```bash
  google-analytics-solo-pp-cli metadata search revenue --property solo-prod --json
  ```

## Recipes

### Fan-out active users across all Solo properties

```bash
google-analytics-solo-pp-cli report --all-properties --dims date --metrics activeUsers --since 7d --json --select rows
```

One report, every registered property, narrowed to just the rows array.

### Rank properties by a metric

```bash
google-analytics-solo-pp-cli compare --metric screenPageViews --since 30d --json
```

Cross-property leaderboard for the last 30 days.

### Find the right metric name

```bash
google-analytics-solo-pp-cli metadata search conversion --property solo-prod --json
```

Search the cached catalog before writing a report.

## Usage

Run `google-analytics-solo-pp-cli --help` for the full command reference and flag list.

## Paths & environment variables

This CLI separates local files into four path kinds:

| Kind | Contents |
|------|----------|
| `config` | User-editable settings such as `config.toml` and saved profiles |
| `data` | Durable local data: `credentials.toml`, `data.db`, cookies, browser-session proof files, and other auth sidecars |
| `state` | Runtime state such as persisted queries, jobs, and `teach.log` |
| `cache` | Regenerable HTTP/cache files |

Each kind resolves independently. The ladder is:

1. Per-kind env var: `GOOGLE_ANALYTICS_SOLO_CONFIG_DIR`, `GOOGLE_ANALYTICS_SOLO_DATA_DIR`, `GOOGLE_ANALYTICS_SOLO_STATE_DIR`, or `GOOGLE_ANALYTICS_SOLO_CACHE_DIR`
2. `--home <dir>` for this invocation
3. `GOOGLE_ANALYTICS_SOLO_HOME` for a flat relocated root
4. XDG env vars: `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME`
5. Platform defaults matching existing installs

For containers and agent sandboxes, prefer a single relocated root:

```bash
export GOOGLE_ANALYTICS_SOLO_HOME=/srv/google-analytics-solo
google-analytics-solo-pp-cli doctor
```

Under `GOOGLE_ANALYTICS_SOLO_HOME=/srv/google-analytics-solo`, the four dirs resolve to `/srv/google-analytics-solo/config`, `/srv/google-analytics-solo/data`, `/srv/google-analytics-solo/state`, and `/srv/google-analytics-solo/cache`.

MCP servers do not receive CLI flags from the host. Put relocation in the host `env` block:

```json
{
  "mcpServers": {
    "google-analytics-solo": {
      "command": "google-analytics-solo-pp-mcp",
      "env": {
        "GOOGLE_ANALYTICS_SOLO_HOME": "/srv/google-analytics-solo"
      }
    }
  }
}
```

Precedence matters in fleets: an ambient per-kind variable such as `GOOGLE_ANALYTICS_SOLO_DATA_DIR` overrides an explicit `--home` for that kind. Use `GOOGLE_ANALYTICS_SOLO_HOME` or the per-kind variables for durable fleet relocation; treat `--home` as the weaker per-invocation lever.

Relocation is one-way. Unsetting `GOOGLE_ANALYTICS_SOLO_HOME` does not move files back to platform defaults, and `doctor` cannot find credentials left under a former root. Move the files manually before unsetting relocation variables.

Existing installs keep working because the platform-default rung matches the legacy layout. On the first auth write, stored secrets leave `config.toml` and are consolidated into `credentials.toml` under the data directory. Run `google-analytics-solo-pp-cli doctor --fail-on warn` to check path and credential-location warnings in automation.

## Commands

### account-summaries

Manage account summaries

- **`google-analytics-solo-pp-cli account-summaries`** - Returns summaries of all accounts accessible by the caller.

### accounts

Manage accounts

- **`google-analytics-solo-pp-cli accounts list`** - Returns all accounts accessible by the caller. Note that these accounts might not currently have GA4 properties. Soft-deleted (ie: "trashed") accounts are excluded by default. Returns an empty list if no relevant accounts are found.
- **`google-analytics-solo-pp-cli accounts search-change-history-events`** - Searches through all changes to an account or its children given the specified set of filters.

### accounts-provision-account-ticket

Manage accounts provision account ticket

- **`google-analytics-solo-pp-cli accounts-provision-account-ticket`** - Requests a ticket for creating an account.

### data-streams

Manage data streams


### google-analytics-admin-properties

Manage google analytics admin properties

- **`google-analytics-solo-pp-cli google-analytics-admin-properties acknowledge-user-data-collection`** - Acknowledges the terms of user data collection for the specified property. This acknowledgement must be completed (either in the Google Analytics UI or through this API) before MeasurementProtocolSecret resources may be created.
- **`google-analytics-solo-pp-cli google-analytics-admin-properties create`** - Creates an "GA4" property with the specified location and attributes.
- **`google-analytics-solo-pp-cli google-analytics-admin-properties list`** - Returns child Properties under the specified parent Account. Only "GA4" properties will be returned. Properties will be excluded if the caller does not have access. Soft-deleted (ie: "trashed") properties are excluded by default. Returns an empty list if no relevant properties are found.
- **`google-analytics-solo-pp-cli google-analytics-admin-properties run-access-report`** - Returns a customized report of data access records. The report provides records of each time a user reads Google Analytics reporting data. Access records are retained for up to 2 years. Data Access Reports can be requested for a property. The property must be in Google Analytics 360. This method is only available to Administrators. These data access records include GA4 UI Reporting, GA4 UI Explorations, GA4 Data API, and other products like Firebase & Admob that can retrieve data from Google Analytics through a linkage. These records don't include property configuration changes like adding a stream or changing a property's time zone. For configuration change history, see [searchChangeHistoryEvents](https://developers.google.com/analytics/devguides/config/admin/v1/rest/v1alpha/accounts/searchChangeHistoryEvents).

### properties

Manage properties

- **`google-analytics-solo-pp-cli properties batch-run-pivot-reports`** - Returns multiple pivot reports in a batch. All reports must be for the same GA4 Property.
- **`google-analytics-solo-pp-cli properties batch-run-reports`** - Returns multiple reports in a batch. All reports must be for the same GA4 Property.
- **`google-analytics-solo-pp-cli properties check-compatibility`** - This compatibility method lists dimensions and metrics that can be added to a report request and maintain compatibility. This method fails if the request's dimensions and metrics are incompatible. In Google Analytics, reports fail if they request incompatible dimensions and/or metrics; in that case, you will need to remove dimensions and/or metrics from the incompatible report until the report is compatible. The Realtime and Core reports have different compatibility rules. This method checks compatibility for Core reports.
- **`google-analytics-solo-pp-cli properties get-metadata`** - Returns metadata for dimensions and metrics available in reporting methods. Used to explore the dimensions and metrics. In this method, a Google Analytics GA4 Property Identifier is specified in the request, and the metadata response includes Custom dimensions and metrics as well as Universal metadata. For example if a custom metric with parameter name `levels_unlocked` is registered to a property, the Metadata response will contain `customEvent:levels_unlocked`. Universal metadata are dimensions and metrics applicable to any property such as `country` and `totalUsers`.
- **`google-analytics-solo-pp-cli properties run-pivot-report`** - Returns a customized pivot report of your Google Analytics event data. Pivot reports are more advanced and expressive formats than regular reports. In a pivot report, dimensions are only visible if they are included in a pivot. Multiple pivots can be specified to further dissect your data.
- **`google-analytics-solo-pp-cli properties run-realtime-report`** - Returns a customized report of realtime event data for your property. Events appear in realtime reports seconds after they have been sent to the Google Analytics. Realtime reports show events and usage data for the periods of time ranging from the present moment to 30 minutes ago (up to 60 minutes for Google Analytics 360 properties). For a guide to constructing realtime requests & understanding responses, see [Creating a Realtime Report](https://developers.google.com/analytics/devguides/reporting/data/v1/realtime-basics).
- **`google-analytics-solo-pp-cli properties run-report`** - Returns a customized report of your Google Analytics event data. Reports contain statistics derived from data collected by the Google Analytics tracking code. The data returned from the API is as a table with columns for the requested dimensions and metrics. Metrics are individual measurements of user activity on your property, such as active users or event count. Dimensions break down metrics across some common criteria, such as country or event name. For a guide to constructing requests & understanding responses, see [Creating a Report](https://developers.google.com/analytics/devguides/reporting/data/v1/basics).

### properties-create-connected-site-tag

Manage properties create connected site tag

- **`google-analytics-solo-pp-cli properties-create-connected-site-tag`** - Creates a connected site tag for a Universal Analytics property. You can create a maximum of 20 connected site tags per property. Note: This API cannot be used on GA4 properties.

### properties-delete-connected-site-tag

Manage properties delete connected site tag

- **`google-analytics-solo-pp-cli properties-delete-connected-site-tag`** - Deletes a connected site tag for a Universal Analytics property. Note: this has no effect on GA4 properties.

### properties-fetch-automated-ga4-configuration-opt-out

Manage properties fetch automated ga4 configuration opt out

- **`google-analytics-solo-pp-cli properties-fetch-automated-ga4-configuration-opt-out`** - Fetches the opt out status for the automated GA4 setup process for a UA property. Note: this has no effect on GA4 property.

### properties-fetch-connected-ga4-property

Manage properties fetch connected ga4 property

- **`google-analytics-solo-pp-cli properties-fetch-connected-ga4-property`** - Given a specified UA property, looks up the GA4 property connected to it. Note: this cannot be used with GA4 properties.

### properties-list-connected-site-tags

Manage properties list connected site tags

- **`google-analytics-solo-pp-cli properties-list-connected-site-tags`** - Lists the connected site tags for a Universal Analytics property. A maximum of 20 connected site tags will be returned. Note: this has no effect on GA4 property.

### properties-set-automated-ga4-configuration-opt-out

Manage properties set automated ga4 configuration opt out

- **`google-analytics-solo-pp-cli properties-set-automated-ga4-configuration-opt-out`** - Sets the opt out status for the automated GA4 setup process for a UA property. Note: this has no effect on GA4 property.


### Self-learning loop

This CLI caches per-question discovery so repeat queries skip the walk and structurally similar queries get answered via entity substitution. The loop also self-captures: every invocation is journaled locally, and failed-flag corrections plus fresh teaches surface as candidates on the next `recall` for confirm/reject judgment. Agents call `recall` before discovery and fire `teach &` after answering. See the `## Automatic learning` section in `SKILL.md` for the full protocol.

- **`google-analytics-solo-pp-cli recall <query>`** - Look up cached resources for a query before running discovery
- **`google-analytics-solo-pp-cli teach`** - Record a query -> resource mapping (silent on success, safe to background with `&`)
- **`google-analytics-solo-pp-cli learnings list`** - Inspect taught rows
- **`google-analytics-solo-pp-cli learnings forget <query>`** - Undo a teach
- **`google-analytics-solo-pp-cli learnings candidates`** - List auto-captured candidates awaiting confirm/reject
- **`google-analytics-solo-pp-cli learnings stats`** - Local loop metrics: recall hit rate, teach-to-reuse, playbook resolution, candidate counts
- **`google-analytics-solo-pp-cli teach-pattern`** - Install a query/resource template up front
- **`google-analytics-solo-pp-cli teach-lookup`** - Add an entity mapping (e.g. country code, team alias) for pattern substitution

Pass `--no-learn` or set `GOOGLE_ANALYTICS_SOLO_NO_LEARN=true` to disable the loop for deterministic flows.

The local store's schema version stamp is one-way: once this version of `google-analytics-solo-pp-cli` opens the database, older binaries refuse it with a version error — upgrade the binary rather than downgrading.

## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
google-analytics-solo-pp-cli account-summaries

# JSON for scripting and agents
google-analytics-solo-pp-cli account-summaries --json

# Filter to specific fields
google-analytics-solo-pp-cli account-summaries --json --select id,name,status

# Dry run — show the request without sending
google-analytics-solo-pp-cli account-summaries --dry-run

# Agent mode — JSON + compact + no prompts in one flag
google-analytics-solo-pp-cli account-summaries --agent
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

## Runtime Endpoint

This CLI resolves endpoint placeholders at runtime, so one installed binary can target different tenants or API versions without regeneration.

Endpoint environment variables:
- `GOOGLE_ANALYTICS_SOLO_PARENT` resolves `{parent}`

Base URL: `https://analyticsdata.googleapis.com`

## Health Check

```bash
google-analytics-solo-pp-cli doctor
```

Verifies configuration, credentials, and connectivity to the API.

## Configuration

Run `google-analytics-solo-pp-cli doctor` to see the resolved config, data, state, and cache directories. The platform-default config path is `~/.config/google-analytics-solo-pp-cli/config.toml`; `--home`, `GOOGLE_ANALYTICS_SOLO_HOME`, and per-kind env vars can relocate it.

Static request headers can be configured under `headers`; per-command header overrides take precedence.

Environment variables:

| Name | Kind | Required | Description |
| --- | --- | --- | --- |
| `GOOGLE_ANALYTICS_SOLO_PARENT` | endpoint | Yes |  |
| `GOOGLE_APPLICATION_CREDENTIALS` | per_call | No | Set to your API credential. |
| `GOOGLE_ANALYTICS_DATA_OAUTH2C` | per_call | No | Set to your API credential. |

### agentcookie (optional)

If you use agentcookie to sync secrets across machines, this CLI auto-adopts agentcookie-managed credentials with no extra setup. When the daemon writes to this CLI's config, `google-analytics-solo-pp-cli doctor` reports `agentcookie: detected` and `auth-status` labels the source as `agentcookie`. Skip this section if you don't use agentcookie - the CLI works the same as any other.

## Troubleshooting
**Authentication errors (exit code 4)**
- Run `google-analytics-solo-pp-cli doctor` to check credentials
- Verify the environment variable is set: `echo $GOOGLE_APPLICATION_CREDENTIALS`
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

### API-specific
- **401 / PERMISSION_DENIED** — Run 'gcloud auth application-default login', or set GOOGLE_APPLICATION_CREDENTIALS to a service-account key with GA4 Viewer on the property.
- **unknown property alias** — Register it: 'alias add <name> <numeric-id>', or auto-discover with 'alias discover'.
- **mutation did nothing** — Admin writes/deletes and 'mp send' preview only; re-run with --confirm to execute.

## Sources & Inspiration

This CLI was built by studying these projects and resources:

- [**googleapis/google-analytics-data (Go)**](https://github.com/googleapis/google-cloud-go) — Go
- [**googleapis/python-analytics-data**](https://github.com/googleapis/python-analytics-data) — Python

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
