# Cineplex CLI

Cineplex Canada theatrical, ticketing, food, and SCENE+ API. Two auth hosts: subscription key for apis.cineplex.com; loyalty bearer for authenticated ticketing; separate SCENE+ cookie session for connect.cineplex.com.

Created by [@pooriaarab](https://github.com/pooriaarab) (Pooria Arab).

## Install

The recommended path installs both the `cineplex-pp-cli` binary and the `pp-cineplex` agent skill (Claude Code, Codex, Cursor, Gemini CLI, GitHub Copilot, and other agents supported by the upstream [`skills`](https://github.com/vercel-labs/skills) CLI) in one shot:

```bash
npx -y @mvanhorn/printing-press-library install cineplex
```

For CLI only (no skill):

```bash
npx -y @mvanhorn/printing-press-library install cineplex --cli-only
```

For skill only — installs the skill into the same agents as the default command above, but skips the CLI binary (use this to update or reinstall just the skill):

```bash
npx -y @mvanhorn/printing-press-library install cineplex --skill-only
```

To constrain the skill install to one or more specific agents (repeatable — agent names match the [`skills`](https://github.com/vercel-labs/skills) CLI):

```bash
npx -y @mvanhorn/printing-press-library install cineplex --agent claude-code
npx -y @mvanhorn/printing-press-library install cineplex --agent claude-code --agent codex
```

### Without Node (Go fallback)

If `npx` isn't available (no Node, offline), install the CLI directly via Go (requires Go 1.26.5 or newer):

```bash
go install github.com/mvanhorn/printing-press-library/library/media-and-entertainment/cineplex/cmd/cineplex-pp-cli@latest
```

This installs the CLI only — no skill.

### Pre-built binary

Download a pre-built binary for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/cineplex-current). On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine <binary>`. On Unix, mark it executable: `chmod +x <binary>`.

<!-- pp-hermes-install-anchor -->
## Install for Hermes

Install the CLI binary first. The installer writes binaries to a per-user managed bin directory by default: `$HOME/.local/bin` on macOS/Linux and `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows.

```bash
npx -y @mvanhorn/printing-press-library install cineplex --cli-only
```

Then install the focused Hermes skill.

From the Hermes CLI:

```bash
hermes skills install mvanhorn/printing-press-library/cli-skills/pp-cineplex --force
```

Inside a Hermes chat session:

```bash
/skills install mvanhorn/printing-press-library/cli-skills/pp-cineplex --force
```

Restart the Hermes session or gateway if the newly installed skill is not visible immediately.

## Install for OpenClaw
Install both the CLI binary and the focused OpenClaw skill. The installer defaults binaries to a per-user bin directory (`$HOME/.local/bin` on macOS/Linux, `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows):

```bash
npx -y @mvanhorn/printing-press-library install cineplex --agent openclaw
```

Restart the OpenClaw session or gateway if the newly installed skill is not visible immediately.

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/cineplex-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.
3. Fill in `CINEPLEX_SUBSCRIPTION_KEY` when Claude Desktop prompts you.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.


```bash
go install github.com/mvanhorn/printing-press-library/library/media-and-entertainment/cineplex/cmd/cineplex-pp-mcp@latest
```

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "cineplex": {
      "command": "cineplex-pp-mcp",
      "env": {
        "CINEPLEX_SUBSCRIPTION_KEY": "<your-key>"
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

Get your API key from your API provider's developer portal. The key typically looks like a long alphanumeric string.

```bash
export CINEPLEX_SUBSCRIPTION_KEY="<paste-your-key>"
```

To persist credentials, use `cineplex-pp-cli auth set-token <token>`. Stored secrets live in `credentials.toml` under the data directory, not in `config.toml`.

### 3. Verify Setup

```bash
cineplex-pp-cli doctor
```

This checks your configuration and credentials.

### 4. Try Your First Command

```bash
cineplex-pp-cli concessions --theatre-id 42
```

## Usage

Run `cineplex-pp-cli --help` for the full command reference and flag list.

## Paths & environment variables

This CLI separates local files into four path kinds:

| Kind | Contents |
|------|----------|
| `config` | User-editable settings such as `config.toml` and saved profiles |
| `data` | Durable local data: `credentials.toml`, `data.db`, cookies, browser-session proof files, and other auth sidecars |
| `state` | Runtime state such as persisted queries, jobs, and `teach.log` |
| `cache` | Regenerable HTTP/cache files |

Each kind resolves independently. The ladder is:

1. Per-kind env var: `CINEPLEX_CONFIG_DIR`, `CINEPLEX_DATA_DIR`, `CINEPLEX_STATE_DIR`, or `CINEPLEX_CACHE_DIR`
2. `--home <dir>` for this invocation
3. `CINEPLEX_HOME` for a flat relocated root
4. XDG env vars: `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME`
5. Platform defaults matching existing installs

For containers and agent sandboxes, prefer a single relocated root:

```bash
export CINEPLEX_HOME=/srv/cineplex
cineplex-pp-cli doctor
```

Under `CINEPLEX_HOME=/srv/cineplex`, the four dirs resolve to `/srv/cineplex/config`, `/srv/cineplex/data`, `/srv/cineplex/state`, and `/srv/cineplex/cache`.

MCP servers do not receive CLI flags from the host. Put relocation in the host `env` block:

```json
{
  "mcpServers": {
    "cineplex": {
      "command": "cineplex-pp-mcp",
      "env": {
        "CINEPLEX_HOME": "/srv/cineplex"
      }
    }
  }
}
```

Precedence matters in fleets: an ambient per-kind variable such as `CINEPLEX_DATA_DIR` overrides an explicit `--home` for that kind. Use `CINEPLEX_HOME` or the per-kind variables for durable fleet relocation; treat `--home` as the weaker per-invocation lever.

Relocation is one-way. Unsetting `CINEPLEX_HOME` does not move files back to platform defaults, and `doctor` cannot find credentials left under a former root. Move the files manually before unsetting relocation variables.

Existing installs keep working because the platform-default rung matches the legacy layout. On the first auth write, stored secrets leave `config.toml` and are consolidated into `credentials.toml` under the data directory. Run `cineplex-pp-cli doctor --fail-on warn` to check path and credential-location warnings in automation.

## Commands

### account

SCENE+ account and subscription status on the APIM connect host

- **`cineplex-pp-cli account`** - Get CineClub subscription status from /prod/connect/v2/subscription

### concessions

Browse Cineplex food, drink, and in-seat menu items

- **`cineplex-pp-cli concessions`** - List concession items for a theatre

### experiences

Browse Cineplex viewing experience filters

- **`cineplex-pp-cli experiences`** - List the Cineplex experience types and filters

### gift-cards

Look up Cineplex gift card balances

- **`cineplex-pp-cli gift-cards`** - Get a Cineplex gift card balance

### loyalty

Mint the loyalty bearer token used by authenticated ticketing calls

- **`cineplex-pp-cli loyalty`** - Mint the ticketing loyalty bearer token from the SCENE+ session

### movies

Find current, upcoming, and detailed Cineplex movies

- **`cineplex-pp-cli movies coming-soon`** - List coming-soon movies with advance tickets
- **`cineplex-pp-cli movies get-by-id`** - Get a movie by Cineplex film ID
- **`cineplex-pp-cli movies get-by-slug`** - Get a movie by Cineplex URL slug
- **`cineplex-pp-cli movies list`** - List the current Cineplex movie catalog
- **`cineplex-pp-cli movies now-playing`** - List now-playing movies that can be booked

### orders

Create and change ticketing orders. These endpoints mutate cart state.

- **`cineplex-pp-cli orders add-concessions`** - WRITE: Add food and drink items to an order
- **`cineplex-pp-cli orders add-tickets`** - WRITE: Add ticket types to an order
- **`cineplex-pp-cli orders apply-voucher`** - WRITE: Apply a ticket voucher to an order
- **`cineplex-pp-cli orders create-session`** - WRITE: Create a ticketing cart
- **`cineplex-pp-cli orders get`** - Get the current ticketing cart
- **`cineplex-pp-cli orders update`** - WRITE: Update the ticketing cart

### payment

Initialize and confirm ticketing payments. These endpoints mutate payment state.

- **`cineplex-pp-cli payment apply-giftcard`** - WRITE: Apply a Cineplex gift card to the ticketing cart
- **`cineplex-pp-cli payment apply-points`** - WRITE: Apply SCENE+ points to the ticketing cart
- **`cineplex-pp-cli payment confirm`** - WRITE: Confirm payment and complete an order
- **`cineplex-pp-cli payment init`** - WRITE: Initialize payment for an order

### scene

SCENE+ account calls using the separate browser cookie session

- **`cineplex-pp-cli scene get-gift-cards`** - Get gift cards linked to the SCENE+ account
- **`cineplex-pp-cli scene get-login-status`** - Get SCENE+ login status from the cookie session
- **`cineplex-pp-cli scene get-pay-pal-information`** - Get saved PayPal information for the SCENE+ account
- **`cineplex-pp-cli scene get-payment-cards`** - Get saved payment cards for the SCENE+ account
- **`cineplex-pp-cli scene get-scene-info`** - Get SCENE+ member information and points
- **`cineplex-pp-cli scene get-user-profile-info`** - Get the authenticated SCENE+ user profile

### seats

Read seat layouts and live seat availability

- **`cineplex-pp-cli seats availability`** - Get current seat availability for a showtime
- **`cineplex-pp-cli seats booking-fees`** - Get online booking fees for a theatre
- **`cineplex-pp-cli seats layout`** - Get the physical seat layout for a showtime
- **`cineplex-pp-cli seats reserve-seats`** - WRITE: Hold seats for a ticketing cart

### showtimes

Find showtimes grouped by theatre, movie, and experience

- **`cineplex-pp-cli showtimes detail`** - Get showtime details for a theatre
- **`cineplex-pp-cli showtimes list`** - List showtimes for a theatre and date

### theatres

Find Cineplex theatres and theatre details

- **`cineplex-pp-cli theatres by-location`** - Resolve a location before finding nearby theatres
- **`cineplex-pp-cli theatres get`** - Get a theatre by location ID
- **`cineplex-pp-cli theatres list`** - List the Cineplex theatre directory
- **`cineplex-pp-cli theatres nearby`** - Find theatres near latitude and longitude

### utilities

Resolve a browser location for Cineplex discovery

- **`cineplex-pp-cli utilities`** - Resolve the current browser location


### Self-learning loop

This CLI caches per-question discovery so repeat queries skip the walk and structurally similar queries get answered via entity substitution. The loop also self-captures: every invocation is journaled locally, and failed-flag corrections plus fresh teaches surface as candidates on the next `recall` for confirm/reject judgment. Agents call `recall` before discovery and fire `teach &` after answering. See the `## Automatic learning` section in `SKILL.md` for the full protocol.

- **`cineplex-pp-cli recall <query>`** - Look up cached resources for a query before running discovery
- **`cineplex-pp-cli teach`** - Record a query -> resource mapping (silent on success, safe to background with `&`)
- **`cineplex-pp-cli learnings list`** - Inspect taught rows
- **`cineplex-pp-cli learnings forget <query>`** - Undo a teach
- **`cineplex-pp-cli learnings candidates`** - List auto-captured candidates awaiting confirm/reject
- **`cineplex-pp-cli learnings stats`** - Local loop metrics: recall hit rate, teach-to-reuse, playbook resolution, candidate counts
- **`cineplex-pp-cli teach-pattern`** - Install a query/resource template up front
- **`cineplex-pp-cli teach-lookup`** - Add an entity mapping (e.g. country code, team alias) for pattern substitution

Pass `--no-learn` or set `CINEPLEX_NO_LEARN=true` to disable the loop for deterministic flows.

The local store's schema version stamp is one-way: once this version of `cineplex-pp-cli` opens the database, older binaries refuse it with a version error — upgrade the binary rather than downgrading.

## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
cineplex-pp-cli concessions --theatre-id 42

# JSON for scripting and agents
cineplex-pp-cli concessions --theatre-id 42 --json

# Filter to specific fields
cineplex-pp-cli concessions --theatre-id 42 --json --select id,name,status

# Dry run — show the request without sending
cineplex-pp-cli concessions --theatre-id 42 --dry-run

# Agent mode — JSON + compact + no prompts in one flag
cineplex-pp-cli concessions --theatre-id 42 --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select id,name` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Explicit retries** - add `--idempotent` to create retries when a no-op success is acceptable
- **Confirmable** - `--yes` for explicit confirmation of destructive actions
- **Piped input** - write commands can accept structured input when their help lists `--stdin`
- **Offline-friendly** - sync/search commands can use the local SQLite store when available
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `4` auth error, `5` API error, `7` rate limited, `10` config error.

## Health Check

```bash
cineplex-pp-cli doctor
```

Verifies configuration, credentials, and connectivity to the API.

## Configuration

Run `cineplex-pp-cli doctor` to see the resolved config, data, state, and cache directories. The platform-default config path is `~/.config/cineplex-cli/config.toml`; `--home`, `CINEPLEX_HOME`, and per-kind env vars can relocate it.

Static request headers can be configured under `headers`; per-command header overrides take precedence.

Environment variables:

| Name | Kind | Required | Description |
| --- | --- | --- | --- |
| `CINEPLEX_SUBSCRIPTION_KEY` | per_call | Yes | Set to your API credential. |

### agentcookie (optional)

If you use agentcookie to sync secrets across machines, this CLI auto-adopts agentcookie-managed credentials with no extra setup. When the daemon writes to this CLI's config, `cineplex-pp-cli doctor` reports `agentcookie: detected` and `auth-status` labels the source as `agentcookie`. Skip this section if you don't use agentcookie - the CLI works the same as any other.

## Troubleshooting
**Authentication errors (exit code 4)**
- Run `cineplex-pp-cli doctor` to check credentials
- Verify the environment variable is set: `echo $CINEPLEX_SUBSCRIPTION_KEY`
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

---

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
