# Meta Ads CLI

Run Meta's Marketing API (campaigns, ad sets, ads, creatives, custom audiences, insights) from a terminal or an AI agent.

Meta's own [official Ads MCP server](https://developers.facebook.com/documentation/ads-commerce/ads-ai-connectors/ads-mcp-server/ads-mcp-server-overview) (`https://mcp.facebook.com/ads`) and official Ads CLI need zero developer-app setup for interactive use in an MCP client. This CLI is for a different case: programmatic/scripted access, or reaching parts of the Graph API neither official tool wraps — it includes a generic `node`/`edge` escape hatch that reaches any Graph API object or edge by ID, not just the ~7 ad-management resources with dedicated commands.

## Install

Build from source (requires [Go](https://go.dev/dl/) 1.22+):

```bash
git clone https://github.com/pooriaarab/clis.git
cd clis/meta-ads
go build -o meta-ads-pp-cli ./cmd/meta-ads-pp-cli
sudo mv meta-ads-pp-cli /usr/local/bin/   # or anywhere on your PATH
```

This also builds an MCP server binary (`meta-ads-pp-mcp`) from the same source:

```bash
go build -o meta-ads-pp-mcp ./cmd/meta-ads-pp-mcp
```

Add it to your MCP client's config (e.g. Claude Desktop's `claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "meta-ads": {
      "command": "/path/to/meta-ads-pp-mcp",
      "env": {
        "META_ADS_BEARER_AUTH": "<your access token>"
      }
    }
  }
}
```

## Setup — getting a real access token

Unlike Google Ads and Reddit Ads, this CLI has no built-in OAuth login flow — no discoverable, machine-readable OAuth flow doc existed for the Graph API to generate one from, so you get a token yourself and hand it to the CLI directly.

1. Create a Meta App at [developers.facebook.com/apps](https://developers.facebook.com/apps) → Create App → type **Business**.
2. Add the **Marketing API** product to the app (App Dashboard → Add Product).
3. Get a token with `ads_management`/`ads_read` permission. Easiest path for your own ad accounts: [Graph API Explorer](https://developers.facebook.com/tools/explorer) → select your app → select permissions → Generate Access Token. For a longer-lived token, exchange the short-lived one for a long-lived one, or create a System User token in Business Manager (Business Settings → Users → System Users) if you need something that doesn't expire in a couple hours.
4. Store it:
   ```bash
   meta-ads-pp-cli auth set-token YOUR_TOKEN_HERE
   ```
   Or set `META_ADS_BEARER_AUTH` as an environment variable instead of persisting it to disk.
5. You'll also need your **ad account ID** (format `act_<numbers>`) — found in Ads Manager's URL or account settings.

Note: `ads_management`/`ads_read` are permissions Meta reviews for public apps serving other people's ad accounts. For your own account(s) as the app's admin/developer, a token generated via Graph API Explorer or a System User works without needing to submit for App Review.

## Quick Start

```bash
go build -o meta-ads-pp-cli ./cmd/meta-ads-pp-cli
meta-ads-pp-cli auth set-token YOUR_TOKEN_HERE
meta-ads-pp-cli doctor
meta-ads-pp-cli act-ad-account-id get-ad-account <your-ad-account-id> --fields name,status
```

## Usage

Run `meta-ads-pp-cli --help` for the full command reference and flag list.

## Paths & environment variables

This CLI separates local files into four path kinds:

| Kind | Contents |
|------|----------|
| `config` | User-editable settings such as `config.toml` and saved profiles |
| `data` | Durable local data: `credentials.toml`, `data.db`, cookies, browser-session proof files, and other auth sidecars |
| `state` | Runtime state such as persisted queries, jobs, and `teach.log` |
| `cache` | Regenerable HTTP/cache files |

Each kind resolves independently. The ladder is:

1. Per-kind env var: `META_ADS_CONFIG_DIR`, `META_ADS_DATA_DIR`, `META_ADS_STATE_DIR`, or `META_ADS_CACHE_DIR`
2. `--home <dir>` for this invocation
3. `META_ADS_HOME` for a flat relocated root
4. XDG env vars: `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME`
5. Platform defaults matching existing installs

For containers and agent sandboxes, prefer a single relocated root:

```bash
export META_ADS_HOME=/srv/meta-ads
meta-ads-pp-cli doctor
```

Under `META_ADS_HOME=/srv/meta-ads`, the four dirs resolve to `/srv/meta-ads/config`, `/srv/meta-ads/data`, `/srv/meta-ads/state`, and `/srv/meta-ads/cache`.

MCP servers do not receive CLI flags from the host. Put relocation in the host `env` block:

```json
{
  "mcpServers": {
    "meta-ads": {
      "command": "meta-ads-pp-mcp",
      "env": {
        "META_ADS_HOME": "/srv/meta-ads"
      }
    }
  }
}
```

Precedence matters in fleets: an ambient per-kind variable such as `META_ADS_DATA_DIR` overrides an explicit `--home` for that kind. Use `META_ADS_HOME` or the per-kind variables for durable fleet relocation; treat `--home` as the weaker per-invocation lever.

Relocation is one-way. Unsetting `META_ADS_HOME` does not move files back to platform defaults, and `doctor` cannot find credentials left under a former root. Move the files manually before unsetting relocation variables.

Existing installs keep working because the platform-default rung matches the legacy layout. On the first auth write, stored secrets leave `config.toml` and are consolidated into `credentials.toml` under the data directory. Run `meta-ads-pp-cli doctor --fail-on warn` to check path and credential-location warnings in automation.

## Commands

### act-ad-account-id

Manage act ad account id

- **`meta-ads-pp-cli act-ad-account-id create-ad`** - Create an ad (adset_id, name, creative={creative_id}, status, etc.)
- **`meta-ads-pp-cli act-ad-account-id create-ad-creative`** - Create an ad creative (name, object_story_spec or asset_feed_spec, etc.)
- **`meta-ads-pp-cli act-ad-account-id create-ad-set`** - Create an ad set (campaign_id, name, targeting, daily_budget or lifetime_budget, billing_event, optimization_goal, bid_amount, status, etc.)
- **`meta-ads-pp-cli act-ad-account-id create-campaign`** - Create a campaign (name, objective, status, special_ad_categories, buying_type, etc.)
- **`meta-ads-pp-cli act-ad-account-id create-custom-audience`** - Create a custom audience (name, subtype, description, customer_file_source, etc.)
- **`meta-ads-pp-cli act-ad-account-id get-ad-account`** - Get ad account details
- **`meta-ads-pp-cli act-ad-account-id get-ad-account-insights`** - Get performance insights for an ad account (reporting)
- **`meta-ads-pp-cli act-ad-account-id list-ad-creatives`** - List ad creatives in an ad account
- **`meta-ads-pp-cli act-ad-account-id list-ad-sets`** - List ad sets in an ad account
- **`meta-ads-pp-cli act-ad-account-id list-ads`** - List ads in an ad account
- **`meta-ads-pp-cli act-ad-account-id list-campaigns`** - List campaigns in an ad account
- **`meta-ads-pp-cli act-ad-account-id list-custom-audiences`** - List custom audiences in an ad account

The `create-*` commands above take their fields as a JSON object via `--body` or piped through `--stdin`, not individual per-field flags — Graph API objects accept too wide a field set to enumerate as flags one by one:

```bash
meta-ads-pp-cli act-ad-account-id create-campaign act_123456789 --body '{"name":"Q3 test","objective":"OUTCOME_SALES","status":"PAUSED"}'
```

### node / edge (generic escape hatch)

Meta's Graph API is node/edge shaped: every object (page, post, lead-gen form, product catalog, business asset, ...) is reachable at `GET/POST/DELETE /{node-id}`, and every relationship is reachable at `GET/POST /{node-id}/{edge}`. The commands above wrap the handful of ad-management objects/edges with dedicated names; `node` and `edge` reach everything else Meta's Graph API exposes:

```bash
# get any node by id
meta-ads-pp-cli node get act_123456789 --fields name,id

# update any node (e.g. pause a campaign)
meta-ads-pp-cli node update 120210000000000001 --body '{"status":"PAUSED"}'

# delete any node
meta-ads-pp-cli node delete 120210000000000001

# list any edge by name off any node
meta-ads-pp-cli edge get act_123456789 activities --fields event_type,event_time

# create a child object on any edge by name
meta-ads-pp-cli edge create 120210000000000002 feed --body '{"message":"Hello"}'
```

### Self-learning loop

This CLI caches per-question discovery so repeat queries skip the walk and structurally similar queries get answered via entity substitution. The loop also self-captures: every invocation is journaled locally, and failed-flag corrections plus fresh teaches surface as candidates on the next `recall` for confirm/reject judgment. Agents call `recall` before discovery and fire `teach &` after answering. See the `## Automatic learning` section in `SKILL.md` for the full protocol.

- **`meta-ads-pp-cli recall <query>`** - Look up cached resources for a query before running discovery
- **`meta-ads-pp-cli teach`** - Record a query -> resource mapping (silent on success, safe to background with `&`)
- **`meta-ads-pp-cli learnings list`** - Inspect taught rows
- **`meta-ads-pp-cli learnings forget <query>`** - Undo a teach
- **`meta-ads-pp-cli learnings candidates`** - List auto-captured candidates awaiting confirm/reject
- **`meta-ads-pp-cli learnings stats`** - Local loop metrics: recall hit rate, teach-to-reuse, playbook resolution, candidate counts
- **`meta-ads-pp-cli teach-pattern`** - Install a query/resource template up front
- **`meta-ads-pp-cli teach-lookup`** - Add an entity mapping (e.g. country code, team alias) for pattern substitution

Pass `--no-learn` or set `META_ADS_NO_LEARN=true` to disable the loop for deterministic flows.

The local store's schema version stamp is one-way: once this version of `meta-ads-pp-cli` opens the database, older binaries refuse it with a version error — upgrade the binary rather than downgrading.

## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
meta-ads-pp-cli act-ad-account-id create-ad mock-value

# JSON for scripting and agents
meta-ads-pp-cli act-ad-account-id create-ad mock-value --json

# Filter to specific fields
meta-ads-pp-cli act-ad-account-id create-ad mock-value --json --select id,name,status

# Dry run — show the request without sending
meta-ads-pp-cli act-ad-account-id create-ad mock-value --dry-run

# Agent mode — JSON + compact + no prompts in one flag
meta-ads-pp-cli act-ad-account-id create-ad mock-value --agent
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

## Runtime Endpoint

This CLI resolves endpoint placeholders at runtime, so one installed binary can target different tenants or API versions without regeneration.

Endpoint environment variables:
- `META_ADS_AD_ACCOUNT_ID` resolves `{ad_account_id}`

Base URL: `https://graph.facebook.com/v21.0`

## Health Check

```bash
meta-ads-pp-cli doctor
