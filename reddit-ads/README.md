{{Target platform}}:{{Unique app ID}}:{{Version string}} (by /u/{{Your Reddit username}})
```

> **Note:** Including your version number helps us block old buggy or broken versions of your app safely. Just remember to update this string when you update your version number.

<details class="sl-border sl-border-solid sl-border-canvas-200 sl-rounded-lg sl-p-4"><summary>Example</summary>

```
User-Agent: android:com.example.myredditapp:v1.2.3 (by /u/kemitche)
```

</details>

> **Important:** Never lie about your user agent! Don't pretend to be popular browsers or other bots. We'll ban anyone caught in the act.

### Pagination

Paginated endpoints respond with a `pagination` object containing up to 2 fields:

> **Important:** The URLs for these fields should be followed directly. Don't assume pagination based on the query parameters in the provided URLs.

- **`next_url`:** The full URL to access the next page of the response. If not available, the current page is the final page.
- **`previous_url`:** The full URL to access the previous page of the response. If not available, the current page is the first page.

All endpoints that return a list, such as `List Campaigns` and `Get a Report`, will be paginated.

### Response types

Response | Description
--- | ---
200 | Successfully processed the request.
400 | Request was invalid. See the client error message for specifics.
401 | No bearer token or a bad bearer token was provided. Check your application authentication.
403 | Insufficient authentication scopes or the user doesn't have permission. Check the client error message for specifics. If it's a permission issue, ensure the user has the proper permissions assigned to take action.
404 | Specified resource was not found. Check that correct permissions have been given to the source and that the resource exists.
429 | Request has exceeded rate limits. For more information on rate limiting and best practices for handling errors, see our [rate limiting reference](/docs/v3/guides/quick-start/rate-limiting).
500 | Server error while processing events. Try again later or [use our chat agent](https://business.reddithelp.com/s/) if the problem persists.

### Limitations

> **Note:** Future changes may introduce new limitations.

- **Rate limits:** Exceeding request limits may result in temporary throttling.
- **Functionality restrictions:** Unless explicitly stated, alpha or beta products aren't supported in the Reddit Ads API.

Created by [@pooriaarab](https://github.com/pooriaarab) (Pooria Arab).

## Install

Build from source (requires [Go](https://go.dev/dl/) 1.22+):

```bash
git clone https://github.com/pooriaarab/clis.git
cd clis/reddit-ads
go build -o reddit-ads-pp-cli ./cmd/reddit-ads-pp-cli
sudo mv reddit-ads-pp-cli /usr/local/bin/   # or anywhere on your PATH
```

This also builds an MCP server binary (`reddit-ads-pp-mcp`) from the same source, for using this CLI's tools from an MCP-compatible agent instead of the terminal:

```bash
go build -o reddit-ads-pp-mcp ./cmd/reddit-ads-pp-mcp
```

Add it to your MCP client's config (e.g. Claude Desktop's `claude_desktop_config.json`) — authenticate with `auth login` first (see Setup below), since the MCP server reuses the token that command stores locally:

```json
{
  "mcpServers": {
    "reddit-ads": {
      "command": "/path/to/reddit-ads-pp-mcp"
    }
  }
}
```

## Setup — getting real credentials

Reddit's Ads API doesn't require allowlisting or approval — any developer can register an app and start immediately.

1. Go to [ads.reddit.com](https://ads.reddit.com) → your Business → Developer Portal → Developer Applications → create a new app.
2. Set the **Redirect URI** to exactly `http://127.0.0.1:8085/callback` — this must match the CLI's login command's default callback port byte-for-byte, or the token exchange fails.
3. Copy the App ID and Secret — these are `REDDIT_ADS_CLIENT_ID` and `REDDIT_ADS_CLIENT_SECRET`.
4. Run the interactive login:
   ```bash
   export REDDIT_ADS_CLIENT_ID=<app id>
   export REDDIT_ADS_CLIENT_SECRET=<app secret>
   reddit-ads-pp-cli login
   ```
   This opens a browser, you approve access once, and the CLI stores a permanent (non-expiring-without-revoke) access token.
5. You'll also want your **ad account ID** (looks like `a2_xxxxxxxx`, not the business ID from step 1's URL) to point commands at — find it via `reddit-ads-pp-cli businesses ad-accounts list-by-business <business-id>`, or in the Ads Manager URL when viewing the account.

If `--port 8085` is already taken on your machine, pass a different port to `login` and update the app's registered Redirect URI to match.

## Quick Start

```bash
go build -o reddit-ads-pp-cli ./cmd/reddit-ads-pp-cli
export REDDIT_ADS_CLIENT_ID=<...>
export REDDIT_ADS_CLIENT_SECRET=<...>
reddit-ads-pp-cli login
reddit-ads-pp-cli ad-accounts get <your-ad-account-id>
```

## Usage

Run `reddit-ads-pp-cli --help` for the full command reference and flag list.

## Paths & environment variables

This CLI separates local files into four path kinds:

| Kind | Contents |
|------|----------|
| `config` | User-editable settings such as `config.toml` and saved profiles |
| `data` | Durable local data: `credentials.toml`, `data.db`, cookies, browser-session proof files, and other auth sidecars |
| `state` | Runtime state such as persisted queries, jobs, and `teach.log` |
| `cache` | Regenerable HTTP/cache files |

Each kind resolves independently. The ladder is:

1. Per-kind env var: `REDDIT_ADS_CONFIG_DIR`, `REDDIT_ADS_DATA_DIR`, `REDDIT_ADS_STATE_DIR`, or `REDDIT_ADS_CACHE_DIR`
2. `--home <dir>` for this invocation
3. `REDDIT_ADS_HOME` for a flat relocated root
4. XDG env vars: `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME`
5. Platform defaults matching existing installs

For containers and agent sandboxes, prefer a single relocated root:

```bash
export REDDIT_ADS_HOME=/srv/reddit-ads
reddit-ads-pp-cli doctor
```

Under `REDDIT_ADS_HOME=/srv/reddit-ads`, the four dirs resolve to `/srv/reddit-ads/config`, `/srv/reddit-ads/data`, `/srv/reddit-ads/state`, and `/srv/reddit-ads/cache`.

MCP servers do not receive CLI flags from the host. Put relocation in the host `env` block:

```json
{
  "mcpServers": {
    "reddit-ads": {
      "command": "reddit-ads-pp-mcp",
      "env": {
        "REDDIT_ADS_HOME": "/srv/reddit-ads"
      }
    }
  }
}
```

Precedence matters in fleets: an ambient per-kind variable such as `REDDIT_ADS_DATA_DIR` overrides an explicit `--home` for that kind. Use `REDDIT_ADS_HOME` or the per-kind variables for durable fleet relocation; treat `--home` as the weaker per-invocation lever.

Relocation is one-way. Unsetting `REDDIT_ADS_HOME` does not move files back to platform defaults, and `doctor` cannot find credentials left under a former root. Move the files manually before unsetting relocation variables.

Existing installs keep working because the platform-default rung matches the legacy layout. On the first auth write, stored secrets leave `config.toml` and are consolidated into `credentials.toml` under the data directory. Run `reddit-ads-pp-cli doctor --fail-on warn` to check path and credential-location warnings in automation.

## Commands

Each command below lists its rate limit as `quota / window`.

### ad-accounts

Information about ad accounts.

- **`ad-accounts get`** — Retrieve an ad account by ID. *(400 req / 60s)*
- **`ad-accounts update`** — Update an ad account. *(200 req / 60s)*

### ad-groups

Information about ad groups.

- **`ad-groups get`** — Retrieve an ad group. *(400 req / 60s)*
- **`ad-groups update`** — Update an ad group. When `shopping_type` is `DYNAMIC`, see the `targeting` field description for which fields are applied. *(200 req / 60s)*

### ads

Information about ads.

- **`ads get`** — Retrieve an ad. *(400 req / 60s)*
- **`ads update`** — Update an ad. For catalog sales ads using `shopping_creative`, see `click_url` for how it is handled. *(200 req / 60s)*

### apps

Information about apps.

### businesses

Information about businesses.

- **`businesses get-business`** — Retrieve business by ID. *(100 req / 60s)*
- **`businesses update-business`** — Update business by ID. *(100 req / 60s)*

### campaigns

Information about campaigns.

- **`campaigns get`** — Retrieve a campaign. *(400 req / 60s)*
- **`campaigns update`** — Update a campaign. *(200 req / 60s)*

### catalog-imports

Manage catalog imports.

### channel-planning

Manage channel planning.

- **`channel-planning`** — Retrieve a 10-point reach curve at different impression levels based on targeting. *(30 req / 60s)*

### creative-assets

Information about creative assets stored in the asset library.

- **`creative-assets get`** — Fetch metadata for your creative asset. See the [asset library](https://business.reddithelp.com/s/article/asset-library). Requires the Max campaign type (beta) — [reach out to a Reddit Ads expert](https://www.business.reddit.com/speak-with-a-reddit-ads-expert) for access. *(200 req / 60s)*
- **`creative-assets get-upload`** — Poll the processing status of a single creative asset upload by upload ID. Use for one upload at a time; use `List Creative Asset Uploads` when checking multiple together. Returns 404 when the upload ID is unavailable, deleted, or not pollable.

### custom-audiences

Information about custom audiences.

- **`custom-audiences delete`** — Delete a custom audience. *(3000 req / 900s, burst 500 req / 60s)*
- **`custom-audiences get`** — Retrieve a custom audience. *(100 req / 60s)*

### data-deletion-jobs

Manage data deletion jobs.

- **`data-deletion-jobs <job_id>`** — Retrieve the current status of a data deletion job. Deletions are processed daily — poll once per day until the job reaches `COMPLETED` or `FAILED`. See the [polling guidance](/docs/v3/guides/programs/data-deletion/delete-user-data#best-practices). *(60 req / 60s)*

### feature-access

Information about feature access.

- **`feature-access`** — Retrieve the features accessible for a particular context. *(100 req / 60s)*

  | ID | Description |
  | --- | --- |
  | `landing_page_optimization` | `LANDING_PAGE_VISIT` is a selectable goal for traffic campaigns. |
  | `reddit_max` | Allow creating [Max Campaigns](https://business.reddithelp.com/s/article/max-campaigns). Only conversions and traffic campaigns are eligible by default. |
  | `rmax_app_installs` | App install campaigns can be created with Max Campaigns. |
  | `ad_group_layout_view_ui` | `supplementary_text` can be provided for ads. |
  | `cbo_shopping` | Campaign budget optimization is selectable for catalog sales campaigns. |
  | `cbo_conversions` | Campaign budget optimization is selectable for conversions campaigns. |
  | `cbo_app_installs` | Campaign budget optimization is selectable for app installs campaigns. |
  | `keyword_targeting_in_feed` | Ads can be shown in Home and Community feeds, prioritized for users who interacted with related content. |
  | `third_party_audiences` | Enables third-party audience targeting with applicable data partners. |

  Contact your [Reddit Ads expert](https://www.business.reddit.com/speak-with-a-reddit-ads-expert) to request access to a feature.

### forecasting

Information about forecasting.

- **`forecasting`** — Retrieve bid suggestions based on recent auction outcomes for a given set of targeting, budget, and bidding parameters. *(30 req / 60s)*

### funding-instruments

Information about funding instruments.

### industries

Manage industries.

- **`industries`** — List supported industries. *(100 req / 60s)*

### lead-gen-forms

Manage lead gen forms.

- **`lead-gen-forms <lead_gen_form_id>`** — Retrieve a lead generation form. *(20 req / 60s)*

### me

Manage me.

- **`me get`** — Retrieve the authenticated user. *(100 req / 60s)*
- **`me list-my-businesses`** — Retrieve all businesses associated with the authenticated user; filter to narrow by access. *(100 req / 60s)*

### pixels

Information about Pixels.

### posts

Information about posts. Legacy API — use the Structured Posts API instead.

- **`posts get`** — Retrieve a post. Legacy — use `structured-posts get`. *(200 req / 60s)*
- **`posts update`** — Update a post. Legacy — create a structured post instead. *(200 req / 60s)*

### product-catalogs

Information about product catalogs.

- **`product-catalogs delete`** — Delete a catalog. Can't be undone. *(7000 req / 300s, burst 3000 req / 60s)*
- **`product-catalogs get`** — Retrieve a catalog. *(7000 req / 300s, burst 3000 req / 60s)*
- **`product-catalogs update`** — Change a catalog's name or attached Pixel ID. *(7000 req / 300s, burst 3000 req / 60s)*

### product-feeds

Manage product feeds.

- **`product-feeds delete`** — Delete a feed in a catalog. Can't be undone. *(7000 req / 300s, burst 3000 req / 60s)*
- **`product-feeds get`** — Retrieve metadata for a specific feed. *(7000 req / 300s, burst 3000 req / 60s)*
- **`product-feeds update`** — Change a feed's metadata. *(7000 req / 300s, burst 3000 req / 60s)*

### product-sets

Manage product sets.

- **`product-sets delete`** — Delete your product set. Can't be undone. *(7000 req / 300s, burst 3000 req / 60s)*
- **`product-sets get`** — Retrieve metadata for a specific product set. *(7000 req / 300s, burst 3000 req / 60s)*
- **`product-sets update`** — Change a specific product set's name and filters. *(7000 req / 300s, burst 3000 req / 60s)*

### profiles

Information about profiles.

- **`profiles <profile_id>`** — Retrieve profile by ID. *(100 req / 60s)*

### saved-audiences

Information about saved audiences.

- **`saved-audiences get`** — Retrieve a saved audience. *(400 req / 60s)*
- **`saved-audiences update`** — Update a saved audience. *(200 req / 60s)*

### structured-posts

Information about structured posts.

- **`structured-posts get`** — Retrieve a structured post. *(200 req / 60s)*
- **`structured-posts get-creation-job`** — Retrieve a structured post creation job. *(200 req / 60s)*
- **`structured-posts update`** — Modify `allow_comments` on an existing structured post. Create another structured post to change the headline, assets, or other fields. *(200 req / 60s)*

### targeting

Information about targeting.

- **`targeting do-geolocation-validations`** — Verify whether geolocations are targetable; invalid ones return an error. *(100 req / 60s)*
- **`targeting do-keyword-validations`** — Validate whether keywords are targetable. *(100 req / 60s)*
- **`targeting list-3rd-party-audiences`** — Retrieve details on supported third-party audiences. *(100 req / 60s)*
- **`targeting list-carriers`** — Retrieve targetable carriers. *(100 req / 60s)*
- **`targeting list-communities`** — Retrieve targetable communities. *(100 req / 60s)*
- **`targeting list-communities-suggestions`** — Retrieve suggested communities based on keywords and/or a website. *(100 req / 60s)*
- **`targeting list-devices`** — Retrieve targetable devices. *(100 req / 60s)*
- **`targeting list-geolocations`** — Retrieve targetable geolocations. *(100 req / 60s)*
- **`targeting list-interests`** — Retrieve targetable interests. *(100 req / 60s)*
- **`targeting list-keyword-suggestions`** — Retrieve keyword suggestions seeded from input terms, each with a monthly view count. *(100 req / 60s)*
- **`targeting list-languages`** — Retrieve targetable languages. *(100 req / 60s)*
- **`targeting search-communities`** — Find targetable communities by name or topic. *(100 req / 60s)*

### time-zones

Information about time zones.

- **`time-zones`** — List supported time zones. *(400 req / 60s)*

### Self-learning loop

This CLI caches per-question discovery so repeat queries skip the walk and structurally similar queries get answered via entity substitution. The loop also self-captures: every invocation is journaled locally, and failed-flag corrections plus fresh teaches surface as candidates on the next `recall` for confirm/reject judgment. Agents call `recall` before discovery and fire `teach &` after answering. See the `## Automatic learning` section in `SKILL.md` for the full protocol.

- **`recall <query>`** — Look up cached resources for a query before running discovery.
- **`teach`** — Record a query → resource mapping (silent on success, safe to background with `&`).
- **`learnings list`** — Inspect taught rows.
- **`learnings forget <query>`** — Undo a teach.
- **`learnings candidates`** — List auto-captured candidates awaiting confirm/reject.
- **`learnings stats`** — Local loop metrics: recall hit rate, teach-to-reuse, playbook resolution, candidate counts.
- **`teach-pattern`** — Install a query/resource template up front.
- **`teach-lookup`** — Add an entity mapping (e.g. country code, team alias) for pattern substitution.

Pass `--no-learn` or set `REDDIT_ADS_NO_LEARN=true` to disable the loop for deterministic flows.

The local store's schema version stamp is one-way: once this version of `reddit-ads-pp-cli` opens the database, older binaries refuse it with a version error — upgrade the binary rather than downgrading.

## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
reddit-ads-pp-cli ad-accounts get mock-value

# JSON for scripting and agents
reddit-ads-pp-cli ad-accounts get mock-value --json

# Filter to specific fields
reddit-ads-pp-cli ad-accounts get mock-value --json --select id,name,status

# Dry run — show the request without sending
reddit-ads-pp-cli ad-accounts get mock-value --dry-run

# Agent mode — JSON + compact + no prompts in one flag
reddit-ads-pp-cli ad-accounts get mock-value --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** — never prompts, every input is a flag
- **Pipeable** — `--json` output to stdout, errors to stderr
- **Filterable** — `--select id,name` returns only fields you need
- **Previewable** — `--dry-run` shows the request without sending
- **Explicit retries** — add `--idempotent` to create retries and add `--ignore-missing` to delete retries when a no-op success is acceptable
- **Confirmable** — `--yes` for explicit confirmation of destructive actions
- **Piped input** — write commands can accept structured input when their help lists `--stdin`
- **Offline-friendly** — sync/search commands can use the local SQLite store when available
- **Agent-safe by default** — no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `4` auth error, `5` API error, `7` rate limited, `10` config error.

## Health Check

```bash
reddit-ads-pp-cli doctor
