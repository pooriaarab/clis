# Google Ads CLI

Run Google Ads from a terminal or an AI agent, with the same commands either way.

## Why this exists

The Google Ads API means gRPC clients, GAQL, and OAuth boilerplate before you can list a single campaign. This CLI wraps that in flag-driven commands with JSON output, so a script or an agent can call it directly — no SDK, no client library, no query-building by hand.

It also learns. Ask it something once and it caches the resource lookup; ask something structurally similar later and it reuses the answer instead of re-discovering it. Point an agent at it and the loop tightens itself over time.

## Install

Build from source (requires [Go](https://go.dev/dl/) 1.22+):

```bash
git clone https://github.com/pooriaarab/clis.git
cd clis/google-ads
go build -o google-ads-pp-cli ./cmd/google-ads-pp-cli
sudo mv google-ads-pp-cli /usr/local/bin/   # or anywhere on your PATH
```

This also builds an MCP server binary (`google-ads-pp-mcp`) from the same source, if you want to use this CLI's tools from an MCP-compatible agent instead of the terminal:

```bash
go build -o google-ads-pp-mcp ./cmd/google-ads-pp-mcp
```

Add it to your MCP client's config (e.g. Claude Desktop's `claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "google-ads": {
      "command": "/path/to/google-ads-pp-mcp",
      "env": {
        "GOOGLE_ADS_CLIENT_ID": "<your-client-id>",
        "GOOGLE_ADS_CLIENT_SECRET": "<your-client-secret>",
        "GOOGLE_ADS_REFRESH_TOKEN": "<your-refresh-token>",
        "GOOGLE_ADS_DEVELOPER_TOKEN": "<your-developer-token>",
        "GOOGLE_ADS_LOGIN_CUSTOMER_ID": "<your-customer-id>"
      }
    }
  }
}
```

## Setup — getting real credentials

The Google Ads API needs five separate values before any command works. None of these come from the same place — budget for a bit of one-time setup.

1. **Create a Google Cloud project** (or use an existing one) at [console.cloud.google.com](https://console.cloud.google.com).
2. **Enable the Google Ads API** on that project: APIs & Services → Library → search "Google Ads API" → Enable.
3. **Create an OAuth client** for this CLI: APIs & Services → Credentials → Create Credentials → OAuth client ID → application type **Desktop app** (not Web application — Desktop clients get an automatic loopback redirect with no redirect URI to register). This gives you `GOOGLE_ADS_CLIENT_ID` and `GOOGLE_ADS_CLIENT_SECRET`.
4. **Get a developer token**: sign in at [ads.google.com](https://ads.google.com) with a **manager (MCC) account** — developer tokens are only issuable from the manager tier, not a regular ad account. Tools & Settings → Setup → API Center. New tokens start in test-account-only access; production access requires Google's review (budget a few days).
5. **Get your login customer ID**: the 10-digit ID (shown like `123-456-7890` in the UI — strip the dashes) of the Google Ads account you want to operate on. This does *not* need to be the manager account — the developer token can come from a manager account while `GOOGLE_ADS_LOGIN_CUSTOMER_ID` points at any account that manager has access to, including the manager's own linked regular accounts.
6. **Get a refresh token** — this is the one interactive step. Run:
   ```bash
   export GOOGLE_ADS_CLIENT_ID=<from step 3>
   export GOOGLE_ADS_CLIENT_SECRET=<from step 3>
   google-ads-pp-cli auth login
   ```
   This opens a browser, you approve access once, and the CLI stores a refresh token that's good indefinitely (no re-login needed after this).

Set all five as environment variables (or in this CLI's config file — run `doctor` to see where) before running any command.

## Quick start

```bash
go build -o google-ads-pp-cli ./cmd/google-ads-pp-cli
export GOOGLE_ADS_CLIENT_ID=<...>
export GOOGLE_ADS_CLIENT_SECRET=<...>
export GOOGLE_ADS_DEVELOPER_TOKEN=<...>
export GOOGLE_ADS_LOGIN_CUSTOMER_ID=<...>
google-ads-pp-cli auth login
google-ads-pp-cli customers get <your-customer-id>
```

Check setup any time with `google-ads-pp-cli doctor`.

## Usage

Run `google-ads-pp-cli --help` for the full command reference and flag list.

## Paths & environment variables

This CLI separates local files into four path kinds:

| Kind | Contents |
|------|----------|
| `config` | User-editable settings such as `config.toml` and saved profiles |
| `data` | Durable local data: `credentials.toml`, `data.db`, cookies, browser-session proof files, and other auth sidecars |
| `state` | Runtime state such as persisted queries, jobs, and `teach.log` |
| `cache` | Regenerable HTTP/cache files |

Each kind resolves independently. The order is:

1. Per-kind env var: `GOOGLE_ADS_CONFIG_DIR`, `GOOGLE_ADS_DATA_DIR`, `GOOGLE_ADS_STATE_DIR`, or `GOOGLE_ADS_CACHE_DIR`
2. `--home <dir>` for this invocation
3. `GOOGLE_ADS_HOME` for a flat relocated root
4. XDG env vars: `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME`
5. Platform defaults matching existing installs

For containers and agent sandboxes, prefer a single relocated root:

```bash
export GOOGLE_ADS_HOME=/srv/google-ads
google-ads-pp-cli doctor
```

Under `GOOGLE_ADS_HOME=/srv/google-ads`, the four dirs resolve to `/srv/google-ads/config`, `/srv/google-ads/data`, `/srv/google-ads/state`, and `/srv/google-ads/cache`.

MCP servers don't receive CLI flags from the host. Put relocation in the host `env` block:

```json
{
  "mcpServers": {
    "google-ads": {
      "command": "google-ads-pp-mcp",
      "env": {
        "GOOGLE_ADS_HOME": "/srv/google-ads"
      }
    }
  }
}
```

Precedence matters in fleets: an ambient per-kind variable such as `GOOGLE_ADS_DATA_DIR` overrides an explicit `--home` for that kind. Use `GOOGLE_ADS_HOME` or the per-kind variables for durable fleet relocation; treat `--home` as the weaker per-invocation lever.

Relocation is one-way. Unsetting `GOOGLE_ADS_HOME` doesn't move files back to platform defaults, and `doctor` won't find credentials left under a former root. Move the files manually before unsetting relocation variables.

Existing installs keep working because the platform-default rung matches the legacy layout. On the first auth write, stored secrets leave `config.toml` and are consolidated into `credentials.toml` under the data directory. Run `google-ads-pp-cli doctor --fail-on warn` to check path and credential-location warnings in automation.

## Commands

### adGroupAds

Operations on adGroupAds

- **`google-ads-pp-cli ad-group-ads <customerId>`** - Create, update, or remove ad group ads

### adGroups

Operations on adGroups

- **`google-ads-pp-cli ad-groups <customerId>`** - Create, update, or remove ad groups

### assets

Operations on assets

- **`google-ads-pp-cli assets <customerId>`** - Create assets

### campaignBudgets

Operations on campaignBudgets

- **`google-ads-pp-cli campaign-budgets <customerId>`** - Create, update, or remove campaign budgets

### campaigns

Operations on campaigns

- **`google-ads-pp-cli campaigns <customerId>`** - Create, update, or remove campaigns

### conversionActions

Operations on conversionActions

- **`google-ads-pp-cli conversion-actions <customerId>`** - Create, update, or remove conversion actions

### customers

Operations on customers

- **`google-ads-pp-cli customers get`** - Get a customer by ID
- **`google-ads-pp-cli customers list-accessible-customers`** - List customer resource names accessible to the authenticated user
- **`google-ads-pp-cli customers mutate`** - Mutate a customer

### mutate (generic escape hatch)

The commands above wrap 6 of the ~65 mutable Google Ads REST resources with dedicated flags. Every one of those resources shares the same request shape (`POST /v{ver}/customers/{id}/{resourcePlural}:mutate` with a `{"operations":[...]}` body), so one generic command reaches all of them — including resources with no dedicated command above, like `experiments`, `experimentArms`, `biddingStrategies`, `audiences`, `userLists`, `labels`, and the rest of the [full REST resource list](https://developers.google.com/google-ads/api/rest/reference/rest):

```bash
# add a keyword to an ad group (AdGroupCriterion create)
google-ads-pp-cli mutate adGroupCriteria 1234567890 --operations '[{"create":{"adGroup":"customers/1234567890/adGroups/222","keyword":{"text":"running shoes","matchType":"PHRASE"}}}]'

# create an experiment (the same thing Google Ads' web UI Experiments tab manages)
google-ads-pp-cli mutate experiments 1234567890 --operations '[{"create":{"name":"Q3 bid test","suffix":"-exp","type":"SEARCH_CUSTOM"}}]'

# pipe generated operations JSON for bulk mutate on any resource
cat ops.json | google-ads-pp-cli mutate audiences 1234567890 --stdin
```

### googleAds

Operations on googleAds (search/reporting)

- **`google-ads-pp-cli google-ads search`** - Run a GAQL search query and return paginated results
- **`google-ads-pp-cli google-ads search-stream`** - Run a GAQL search query and stream results

### Self-learning loop

This CLI caches per-question discovery so repeat queries skip the walk and structurally similar queries get answered via entity substitution. The loop also self-captures: every invocation is journaled locally, and failed-flag corrections plus fresh teaches surface as candidates on the next `recall` for confirm/reject judgment. Agents call `recall` before discovery and fire `teach &` after answering. See the `## Automatic learning` section in `SKILL.md` for the full protocol.

- **`google-ads-pp-cli recall <query>`** - Look up cached resources for a query before running discovery
- **`google-ads-pp-cli teach`** - Record a query -> resource mapping (silent on success, safe to background with `&`)
- **`google-ads-pp-cli learnings list`** - Inspect taught rows
- **`google-ads-pp-cli learnings forget <query>`** - Undo a teach
- **`google-ads-pp-cli learnings candidates`** - List auto-captured candidates awaiting confirm/reject
- **`google-ads-pp-cli learnings stats`** - Local loop metrics: recall hit rate, teach-to-reuse, playbook resolution, candidate counts
- **`google-ads-pp-cli teach-pattern`** - Install a query/resource template up front
- **`google-ads-pp-cli teach-lookup`** - Add an entity mapping (e.g. country code, team alias) for pattern substitution

Pass `--no-learn` or set `GOOGLE_ADS_NO_LEARN=true` to disable the loop for deterministic flows.

The local store's schema version stamp is one-way: once this version of `google-ads-pp-cli` opens the database, older binaries refuse it with a version error — upgrade the binary rather than downgrading.

## Output formats

```bash
# Human-readable table (default in terminal, JSON when piped)
google-ads-pp-cli customers get mock-value

# JSON for scripting and agents
google-ads-pp-cli customers get mock-value --json

# Filter to specific fields
google-ads-pp-cli customers get mock-value --json --select id,name,status

# Dry run — show the request without sending
google-ads-pp-cli customers get mock-value --dry-run

# Agent mode — JSON + compact + no prompts in one flag
google-ads-pp-cli customers get mock-value --agent
```

## Built for agents, not just humans

Most Google Ads tooling assumes a person reading a terminal. This one assumes the caller might be a script:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select id,name` returns only the fields you asked for
- **Previewable** - `--dry-run` shows the request without sending it
- **Explicit retries** - add `--idempotent` when a no-op success on retry is fine
- **Confirmable** - `--yes` for explicit confirmation of destructive actions
- **Piped input** - write commands accept structured input when their help lists `--stdin`
- **Offline-friendly** - sync/search commands can use the local SQLite store when available
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `4` auth error, `5` API error, `7` rate limited, `10` config error.

## Health check

```bash
google-ads-pp-cli doctor
```

Checks configuration, credentials, and connectivity to the API.

## Configuration

Run `google-ads-pp-cli doctor` to see the resolved config, data, state, and cache directories. The platform-default config path is `~/.config/google-ads-cli/config.toml`; `--home`, `GOOGLE_ADS_HOME`, and per-kind env vars can relocate it.

Static request headers go under `headers`; per-command header overrides take precedence.

Environment variables:

| Name | Required | Description |
| --- | --- | --- |
| `GOOGLE_ADS_CLIENT_ID` | Yes | OAuth client ID (Desktop app type — see Setup above) |
| `GOOGLE_ADS_CLIENT_SECRET` | Yes | OAuth client secret, paired with the client ID |
| `GOOGLE_ADS_REFRESH_TOKEN` | Yes | Long-lived token from `auth login`; saved to this CLI's credentials file automatically after login, so you usually don't need to set this by hand |
| `GOOGLE_ADS_DEVELOPER_TOKEN` | Yes | From your manager account's API Center |
| `GOOGLE_ADS_LOGIN_CUSTOMER_ID` | Yes | The 10-digit account ID to operate on (digits only, no dashes) |

### agentcookie (optional)

If you use agentcookie to sync secrets across machines, this CLI auto-adopts agentcookie-managed credentials with no extra setup. When the daemon writes to this CLI's config, `google-ads-pp-cli doctor` reports `agentcookie: detected` and `auth-status` labels the source as `agentcookie`. Skip this section if you don't use agentcookie — the CLI works the same either way.

## Troubleshooting

**Authentication errors (exit code 4)**
- Run `google-ads-pp-cli doctor` to check credentials
- Verify all five environment variables from Setup are set: `google-ads-pp-cli auth status`
- `USER_PERMISSION_DENIED` on an otherwise-valid request usually means `GOOGLE_ADS_LOGIN_CUSTOMER_ID` points at an account your OAuth login doesn't have access to — point it at an account you can actually see in the Google Ads UI

**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

---

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
