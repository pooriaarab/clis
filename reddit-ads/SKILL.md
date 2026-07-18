---
name: pp-reddit-ads
description: "Printing Press CLI for Reddit Ads. By accessing or using the Ads API and/or associated Reddit Data, you are agreeing that you have read"
author: "Pooria Arab"
license: "Apache-2.0"
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - reddit-ads-pp-cli
---

# Reddit Ads — Printing Press CLI

## Prerequisites: Install the CLI

This skill drives the `reddit-ads-pp-cli` binary. **You must verify the CLI is installed before invoking any command from this skill.** If it is missing, install it first:

1. Install via the Printing Press installer. It defaults binaries to `$HOME/.local/bin` on macOS/Linux and `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows:
   ```bash
   npx -y @mvanhorn/printing-press-library install reddit-ads --cli-only
   ```
2. Verify: `reddit-ads-pp-cli --version`
3. Ensure the reported install directory is on `$PATH` for the agent/runtime that will invoke this skill.

If the `npx` install fails before this CLI has a public-library category, install Node or use the category-specific Go fallback after publish.

If `--version` reports "command not found" after install, the runtime cannot see the binary directory on `$PATH`. Do not proceed with skill commands until verification succeeds.

## Ads API terms

By accessing or using the Ads API and/or associated Reddit Data, you are agreeing that you have read, and that you Agree to comply with and to be bound by, the [Ads API Terms](https://business.reddithelp.com/s/article/Reddit-Ads-API-Terms) and all applicable laws and regulations in their entirety without limitation or qualification.

The Ads API Terms apply to and govern your access to and use of the Ads API and Reddit Data, constitute a legally binding agreement between you and Reddit, and include certain terms that are defined in the [Advertising Platform Terms](https://business.reddithelp.com/s/article/REDDIT-ADVERTISING-PLATFORM-TERMS), [Developer Terms](https://www.redditinc.com/policies/developer-terms), and [User Agreement](https://redditinc.com/policies/user-agreement).

You may use the Ads API and Reddit Data only in accordance with the Ads API Terms. Use of the Conversions API is subject to the [Advertiser Measurement Program Terms](https://business.reddithelp.com/s/article/Reddit-Business-Tool-Terms) in addition to the same agreements as the Ads API. If you do not agree to the Ads API Terms, and Advertiser Measurement Program Terms as applicable, then you must not access or use the Ads API, Conversions API, or Reddit Data.

<blockquote>

**Note:** The Reddit Ads API is open to all developers and **does not require allowlisting or approval from Reddit to access**. [Get started with the API](https://ads-api.reddit.com/docs/v3/create-a-developer-application).

API partners working on behalf of advertisers can request support by [contacting us](https://support.reddithelp.com/hc/en-us/requests/new?ticket_form_id=50388520825236). [Managed advertisers](https://www.business.reddit.com/speak-with-a-reddit-ads-expert) can request partnership approval by contacting their Reddit Ads expert. Self-service developers interested in joining the Ads API ecosystem can apply to become a [partner](https://www.redditforbusiness.com/api-partnership). Learn more about Reddit's [advertising ecosystem partner program](https://www.business.reddit.com/solutions/advertising-agency/ecosystem-partner).
</blockquote>

## Overview

The Reddit Ads API lets advertisers tap into the Reddit Ads Platform to build and manage advertising campaigns and accounts programmatically. By integrating Reddit Ads API into your tooling, you can streamline workflows, reduce operational overhead, and accelerate the creation, optimization, and reporting of Reddit campaigns without needing to step into the Reddit Ads Manager directly.

All API applications must be authenticated with OAuth2 by [creating a developer application](https://ads-api.reddit.com/docs/v3/create-a-developer-application) and [obtaining access and refresh tokens](https://ads-api.reddit.com/docs/v3/authenticate-your-developer-application).

![v3 ERD diagram](/common/v3/erd-diagram.png)

Here are two popular ways to use the Reddit Ads API:

- **Postman:** A visual client that makes it simpler to set up and start testing requests and responses. This is the fastest way to familiarize yourself with the API if you prefer a point-and-click approach. You can explore our [Postman collections](https://www.postman.com/reddit-ads-api "https://www.postman.com/reddit-ads-api").
- **Programmatic access:** Use your preferred programming language or command-line tools to create automations and integrations. This approach is ideal if you want to embed API calls into your workflows or build custom tools.

### User agents

Rate-limiting issues often occur if you haven't set your user agent to a unique descriptor. Many default user agents (like `“Python/urllib”` or `“Java”`) are drastically limited to encourage unique and descriptive user-agent strings. We recommend following this format for your client's user agent string:

```
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

## Command Reference

**ad-accounts** — Information about ad accounts.

- `reddit-ads-pp-cli ad-accounts get` — Retrieve ad account by ID.
- `reddit-ads-pp-cli ad-accounts update` — Update an ad account.

**ad-groups** — Information about ad groups.

- `reddit-ads-pp-cli ad-groups get` — Retrieve an ad group.
- `reddit-ads-pp-cli ad-groups update` — Update an ad group.

**ads** — Information about ads.

- `reddit-ads-pp-cli ads get` — Retrieve an ad.
- `reddit-ads-pp-cli ads update` — Update an ad. For catalog sales ads using `shopping_creative`, see `click_url` for how it is handled.

**apps** — Information about apps.


**businesses** — Information about businesses.

- `reddit-ads-pp-cli businesses get-business` — Retrieve business by ID.
- `reddit-ads-pp-cli businesses update-business` — Update business by ID.

**campaigns** — Information about campaigns.

- `reddit-ads-pp-cli campaigns get` — Retrieve a campaign.
- `reddit-ads-pp-cli campaigns update` — Update a campaign.

**catalog-imports** — Manage catalog imports


**channel-planning** — Manage channel planning

- `reddit-ads-pp-cli channel-planning` — Retrieve a 10-point reach curve at different impression levels based on targeting.

**creative-assets** — Information about creative assets stored in the asset library.

- `reddit-ads-pp-cli creative-assets get` — Fetch metadata for your creative asset. Learn about the [asset library](https://business.reddithelp.
- `reddit-ads-pp-cli creative-assets get-upload` — Poll the processing status of a single creative asset upload by upload ID.

**custom-audiences** — Information about custom audiences.

- `reddit-ads-pp-cli custom-audiences delete` — Delete a custom audience.
- `reddit-ads-pp-cli custom-audiences get` — Retrieve a custom audience.

**data-deletion-jobs** — Manage data deletion jobs

- `reddit-ads-pp-cli data-deletion-jobs <job_id>` — Retrieve the current status of a data deletion job.

**feature-access** — Information about feature access.

- `reddit-ads-pp-cli feature-access` — Retrieve a list of the features accessible for a particular context.

**forecasting** — Information about forecasting.

- `reddit-ads-pp-cli forecasting` — Retrieve bid suggestions based on recent auction outcomes for a given set of targeting, budget, and bidding parameters.

**funding-instruments** — Information about funding instruments.


**industries** — Manage industries

- `reddit-ads-pp-cli industries` — List supported industries.

**lead-gen-forms** — Manage lead gen forms

- `reddit-ads-pp-cli lead-gen-forms <lead_gen_form_id>` — Retrieve a lead generation form.

**me** — Manage me

- `reddit-ads-pp-cli me get` — Retrieve the authenticated user.
- `reddit-ads-pp-cli me list-my-businesses` — Retrieve all businesses associated with the authenticated user. Apply filters to narrow results by the user's access.

**pixels** — Information about Pixels.


**posts** — Information about posts.
> **Note:** This is a legacy API. Use [Structured Posts API](/docs/v3/api/structured-posts) instead.

- `reddit-ads-pp-cli posts get` — Retrieve a post. > **Note:** This is a legacy endpoint. Use [`Get Structured Post`](https://ads-api.reddit.
- `reddit-ads-pp-cli posts update` — Update a post. > **Note:** This is a legacy endpoint. Create a [structured post](https://ads-api.reddit.

**product-catalogs** — Information about product catalogs.

- `reddit-ads-pp-cli product-catalogs delete` — Delete a catalog. > **Important:** This action can't be undone.
- `reddit-ads-pp-cli product-catalogs get` — Retrieve a catalog.
- `reddit-ads-pp-cli product-catalogs update` — Change a catalog's name or attached Pixel ID.

**product-feeds** — Manage product feeds

- `reddit-ads-pp-cli product-feeds delete` — Delete a feed in a catalog. > **Important:** This action can't be undone.
- `reddit-ads-pp-cli product-feeds get` — Retrieve metadata for a specific feed.
- `reddit-ads-pp-cli product-feeds update` — Change a feed's metadata.

**product-sets** — Manage product sets

- `reddit-ads-pp-cli product-sets delete` — Delete your product set. > **Important:** This action can't be undone.
- `reddit-ads-pp-cli product-sets get` — Retrieve metadata for a specific product set.
- `reddit-ads-pp-cli product-sets update` — Change a specific product set's name and filters.

**profiles** — Information about profiles.

- `reddit-ads-pp-cli profiles <profile_id>` — Retrieve profile by ID.

**saved-audiences** — Information about saved audiences.

- `reddit-ads-pp-cli saved-audiences get` — Retrieve a saved audience.
- `reddit-ads-pp-cli saved-audiences update` — Update a saved audience.

**structured-posts** — Information about structured posts.

- `reddit-ads-pp-cli structured-posts get` — Retrieve a structured post.
- `reddit-ads-pp-cli structured-posts get-creation-job` — Retrieve a structured post creation job.
- `reddit-ads-pp-cli structured-posts update` — Modify `allow_comments` on an existing structured post. [Create another structured post](https://ads-api.reddit.

**targeting** — Information about targeting.

- `reddit-ads-pp-cli targeting do-geolocation-validations` — Verify whether certain geolocations are targetable. Invalid geolocations will return an error message.
- `reddit-ads-pp-cli targeting do-keyword-validations` — Validate whether certain [keywords are targetable](https://business.reddithelp.
- `reddit-ads-pp-cli targeting list-3rd-party-audiences` — Retrieve details on supported third-party audiences.
- `reddit-ads-pp-cli targeting list-carriers` — Retrieve details on [targetable carriers](https://business.reddithelp.
- `reddit-ads-pp-cli targeting list-communities` — Retrieve details on [targetable communities](https://business.reddithelp.
- `reddit-ads-pp-cli targeting list-communities-suggestions` — Retrieve suggested communities based on keywords and/or a website.
- `reddit-ads-pp-cli targeting list-devices` — Retrieve details on [targetable devices](https://business.reddithelp.
- `reddit-ads-pp-cli targeting list-geolocations` — Retrieve details on [targetable geolocations](https://business.reddithelp.
- `reddit-ads-pp-cli targeting list-interests` — Retrieve details on [targetable interests](https://business.reddithelp.
- `reddit-ads-pp-cli targeting list-keyword-suggestions` — Retrieve keyword suggestions seeded from a list of input terms. Each suggestion includes its monthly view count.
- `reddit-ads-pp-cli targeting list-languages` — Retrieve details on [targetable languages](https://business.reddithelp.com/s/article/demographics#language-targeting).
- `reddit-ads-pp-cli targeting search-communities` — Find [targetable communities](https://business.reddithelp.

**time-zones** — Information about time zones.

- `reddit-ads-pp-cli time-zones` — List supported time zones.


### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
reddit-ads-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

## Auth Setup

Run `reddit-ads-pp-cli auth setup` for the URL and steps to obtain a token (add `--launch` to open the URL). Then store it:

```bash
reddit-ads-pp-cli auth set-token YOUR_TOKEN_HERE
```

Or set `REDDIT_ADS_REDDIT_APIKEY` as an environment variable.

Run `reddit-ads-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  reddit-ads-pp-cli ad-accounts get mock-value --agent --select id,name,status
  ```
- **Previewable** — `--dry-run` shows the request without sending
- **Offline-friendly** — sync/search commands can use the local SQLite store when available
- **Non-interactive** — never prompts, every input is a flag
- **Explicit retries** — use `--idempotent` only when an already-existing create should count as success, and use `--ignore-missing` only when a missing delete target should count as success

### Response envelope

Commands that read from the local store or the API wrap output in a provenance envelope:

```json
{
  "meta": {"source": "live" | "local", "synced_at": "...", "reason": "..."},
  "results": <data>
}
```

Parse `.results` for data and `.meta.source` to know whether it's live or local. A human-readable `N results (live)` summary is printed to stderr only when stdout is a terminal AND no machine-format flag (`--json`, `--csv`, `--compact`, `--quiet`, `--plain`, `--select`) is set — piped/agent consumers and explicit-format runs get pure JSON on stdout.

## Paths and state

Agents should treat the CLI's path resolver as part of the runtime contract:

- Use `--home <dir>` for one invocation, or set `REDDIT_ADS_HOME=<dir>` to relocate all four path kinds under one root.
- Use per-kind env vars only when a specific kind must diverge: `REDDIT_ADS_CONFIG_DIR`, `REDDIT_ADS_DATA_DIR`, `REDDIT_ADS_STATE_DIR`, `REDDIT_ADS_CACHE_DIR`.
- Resolution order is per-kind env var, `--home`, `REDDIT_ADS_HOME`, XDG (`XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME`), then platform defaults.
- `config` contains settings like `config.toml` and profiles. `data` contains `credentials.toml`, `data.db`, cookies, and auth sidecars. `state` contains persisted queries, jobs, and `teach.log`. `cache` contains regenerable HTTP/cache files.
- Stored secrets live in `credentials.toml` under the data dir. Existing legacy `config.toml` secrets are read for compatibility and leave `config.toml` on the first auth write.
- Run `reddit-ads-pp-cli doctor --fail-on warn` to surface path and credential-location warnings. `agent-context` exposes a schema v4 `paths` block for agents that need the resolved dirs.
- For MCP, pass relocation through the MCP host config. The MCP binary does not inherit CLI flags:

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

Fleet precedence: an inherited per-kind env var overrides an explicit `--home` for that kind. Use `REDDIT_ADS_HOME` or per-kind vars as durable fleet levers, and use `--home` only for a single invocation. Relocation is not reversible by unsetting env vars; move files manually before clearing `REDDIT_ADS_HOME`, or `doctor` will not find credentials left under the former root.

## Automatic learning

This CLI ships a self-capturing learning loop. The CLI does its own bookkeeping: every invocation is journaled locally, a failed flag followed by a corrected retry auto-derives a `flag_alias` candidate, and a `teach` on a query family without a playbook auto-synthesizes a `playbook_candidate` from the session's journal. Your job is judgment only: `recall` first, act on surfaced candidates, `teach` the final answer, `playbook amend` when you observe a correction. You never record failures by hand.

### Step 1: `recall` before any discovery

Before list/search/drill commands on a new user question, run:

```bash
reddit-ads-pp-cli recall "<user's question>" --agent
```

The response envelope:

```json
{
  "query": "...",
  "normalized": "<normalized form>",
  "query_entities": ["..."],
  "found": true | false,
  "match_score": 0.0,
  "results": [
    { "resource_id": "...", "resource_type": "...", "venue": "...",
      "confidence": 2, "entity_match": "exact|partial|unknown",
      "source": "taught|preseed|pattern", "warnings": ["..."] }
  ],
  "mismatches": [ /* only when --debug-mismatches */ ],
  "warnings": [ /* top-level */ ],
  "candidates": [
    { "id": 12, "class": "flag_alias | playbook_candidate",
      "summary": "...", "sightings": 3, "last_seen": "...",
      "rationale": "...",
      "next_action": ["<trial command>", "reddit-ads-pp-cli learnings confirm 12"] }
  ],
  "playbook": {
    "query_family": "...",
    "playbook": {
      "steps": [ { "cmd": "<command with {slot} substitution>", "purpose": "..." } ],
      "entity_slots": ["$ENTITY"],
      "expected_tool_calls": 3
    },
    "slots_resolved": { "$ENTITY": { "token": "<live token>", "canonical": "<canonical>" } },
    "notes": "<workarounds + gotchas for this query family>"
  },
  "notes": "<duplicate surface for non-playbook callers>"
}
```

Empty-store short-circuit: if the store has no learnings, playbooks, or candidates yet (recall finds nothing and `learnings list` and `learnings candidates` are both empty), skip recall for the rest of this session instead of taxing every query; resume recall-first once something has been taught.

### Step 2: decision tree

Read `candidates`, `playbook`, `notes`, `results[0]`, and warnings in that order:

```
if Candidates present (warnings include "candidates_present"):
    -> candidates are try-then-confirm, never facts. Follow each candidate's
       two-step next_action verbatim: run the trial command first, then run
       `learnings confirm <id>` only after the trial verified the behavior.
       Reject a wrong candidate with `learnings reject <id>`.
    -> NEVER re-teach something recall surfaced as a candidate; confirm or
       reject that candidate instead of teaching a duplicate.
    -> candidates ride alongside playbooks and resource hits, not instead of
       them; continue with the branches below after acting on them.

if Playbook present:
    -> READ Playbook.notes verbatim FIRST (workarounds + gotchas the CLI surface doesn't expose)
    -> replay Playbook.steps in order, substituting Playbook.slots_resolved entries
       for the entity slot tokens. If a step's slot is unresolved, fall back to
       discovery for that step only.
    -> the Playbook's expected_tool_calls is a budget; if you find yourself running
       materially more, record the divergence via `reddit-ads-pp-cli playbook amend`
       at end-of-session.

elif Notes present (no Playbook):
    -> read Notes verbatim before any discovery step; they carry known gotchas
       for this query family even when no structured choreography exists yet.

elif Found AND Results[0].EntityMatch == "exact" AND Results[0].Confidence >= 2:
    -> skip discovery; fetch live data for Results[*].ResourceID in parallel

elif Found AND Results[0].EntityMatch == "partial":
    -> candidate hint, NOT a hit; read the resource title to validate before trusting

elif (any row in Mismatches[] when --debug-mismatches was passed):
    -> treat as cold start; the stored learning is for a different entity
       (different canonical resolved from query_entities)

else:  // Found == false, no playbook, no notes
    -> cold start; run discovery normally; teach the answer afterward (Step 4).
       If the family has no playbook yet, that teach auto-synthesizes a
       playbook candidate from this session's journal - you do not need to
       record one by hand.
```

Playbook and Notes are orthogonal to the per-resource path. A recall response can carry both a Playbook AND a `Results[]` hit - use both: the Playbook tells you which choreography to run; the resource hits short-circuit specific steps. Default to skipping `mismatches`; pass `--debug-mismatches` only when investigating cold-start surprises.

Candidate judgment details: `learnings confirm <id>` prints the candidate's full payload before materializing it - check that the printed payload matches the behavior you verified. `learnings reject <id>` tombstones the derivation signature so the same candidate does not resurface. The envelope carries only the few candidates worth acting on now; `reddit-ads-pp-cli learnings candidates` lists the full open set.

Graceful degradation: if `learnings confirm` is an unknown command, you are driving an older binary - ignore the candidates guidance and follow the rest of the protocol.

### Step 3: always read `warnings`

- `low_confidence`: row exists at `confidence<2`. Treat as a hint, not a skip-discovery hit.
- `resource_not_in_store`: the local store doesn't have the resource the learning points at. The match validator couldn't classify entities — direct-fetch and re-evaluate.
- `cross_alias_match` (per-result): the row was taught under a different alias and matched the live query's canonical via `entity_lookups` (e.g., a "USA" teach satisfying a "United States" recall). Trust the resource_id.
- `similar_shape_different_entity:<canonical>` (top-level): a structurally matching row exists but its canonical entity differs from the live query's. Treated as cold start; the warning carries the conflicting canonical as a hint, but the row is NOT promoted into Results.
- `ambiguous_alias` (top-level): a single query entity resolved to multiple canonicals (e.g., "Cards" → Arizona Cardinals + St. Louis Cardinals). Surface the ambiguity from context before committing to a resource.
- `candidates_present` (top-level): the envelope carries a `candidates` section. Handle it via the candidates branch in Step 2 before anything else.
- `lookup_refresh_available` (top-level): an entity in the query has no lookup row yet, but synced data could provide one. Run `reddit-ads-pp-cli sync` to refresh entity lookups.
- Top-level `no_learnings_for_query_family`: the table had no rows above the Jaccard floor. Pure cold start.

### Step 4: `teach &` after finalizing your response - always

Teaching is unconditional. After resolving a query the store could not answer, background-teach the final resource mapping - no call-count threshold, no judging whether it was "worth" learning. The teach is the anchor of the loop: it triggers playbook synthesis for a family without a playbook, and same-referent phrasings fold into one family so near-duplicate teaches do not fragment the store. Fire it after assembling your user-facing response but BEFORE emitting it, with a shell `&` so the call returns immediately:

```bash
reddit-ads-pp-cli teach --query "<user's question>" --resource-type <type> --resource <id1> --resource <id2>
# (append shell `&` to background it)
```

Silent on success. Errors only land in `teach.log` under the resolved state dir. Teach the **most specific** resource - if the user asked a broad question and you walked through parent records to find the specific answer, teach the leaf id, not the parent. The CLI uses seeded `entity_lookups` for cross-alias resolution at recall time, so a teach under one alias (e.g., "Niners") satisfies future queries under another alias (e.g., "49ers", "San Francisco") automatically.

PII rule: teach the structural question with identifiers stripped - never include names, emails, phone numbers, account ids, or other personal identifiers in taught queries or notes. The CLI scans teach queries for obvious email/phone shapes and warns, but does not block; strip before teaching rather than relying on the warning.

### Step 5: playbooks - optional flags, automatic synthesis

You do not need to decide whether a session "deserves" a playbook: a teach on a family without one auto-synthesizes a `playbook_candidate` from the session's journal, and the next session judges it via confirm/reject. Attach explicit playbook flags only when you already hold choreography worth recording verbatim - workarounds the CLI didn't surface (silently-dropped flags, undocumented params, pagination tricks, payload gotchas). Prefer the **integrated one-call form** - record the resource learning and the playbook in the same `teach` invocation:

```bash
# Common case: record both the resource learning AND the playbook in one call.
reddit-ads-pp-cli teach \
  --query "<user's question>" \
  --resource <id> \
  --playbook-file ~/playbooks/<shape>.json \
  --playbook-notes-file ~/playbooks/<shape>-notes.md
# (append shell `&` to background it)

# Alternate: playbook-only (no resource to record alongside).
reddit-ads-pp-cli teach-playbook \
  --query "<user's question>" \
  --playbook-file ~/playbooks/<shape>.json \
  --notes-file ~/playbooks/<shape>-notes.md
```

Playbook files are JSON with `steps`, `entity_slots`, `expected_tool_calls`. Notes files are markdown carrying the gotchas verbatim. File-free callers (MCP-only agents) pass the same content inline: `--playbook-json` and `--playbook-notes` on the integrated `teach` form, `--playbook-json` and `--notes` on `teach-playbook`. On the integrated `teach` form, the playbook flags are optional - omit them entirely for a resource-only teach. On the standalone `teach-playbook` form, at least one of the playbook and notes flags must be set; both empty is rejected. Playbooks are keyed on the structural query family (entities stripped) so a recipe taught from one entity-shaped query applies to every other query of the same shape, with `slots_resolved` binding the live query's canonical at recall time.

When you DO find a playbook on a future recall, treat it as ground truth: replay the steps with `slots_resolved` substitutions, skip the discovery that the choreography already documents, and read `notes` before any step.

### Step 6: `playbook amend &` when your debug response identifies a correction

If your debug-protocol response identifies a concrete correction the notes or playbook should know — a workaround, an undocumented endpoint shape, a stale field name, observed schema drift, an empty-payload fallback — fire `playbook amend` BEFORE emitting your user-facing response. Same fire-and-forget posture as `teach`.

```bash
reddit-ads-pp-cli playbook amend \
  --query "<exact recall query string>" \
  --add-note "<your concrete correction>"
# (append shell `&` to background it)
```

What counts as worth amending: a behavior you OBSERVED this session that future-you would benefit from knowing. Examples worth amending:

- A workaround for a CLI surface that silently drops or misorders a flag.
- An undocumented endpoint shape (response wrapped in `{meta, results}`, payload nested two levels deeper than the docs claim).
- Observed schema drift (a field renamed, an index that shifted between seasons, a category label that the API now returns lower-cased).

What does NOT belong in notes:

- The year-specific or entity-specific answer to the user's question. That's the response, not a learning.
- Per-team / per-athlete / per-row data the playbook already retrieves at runtime.
- Statements that paraphrase what the existing notes already say.

The amend command appends to the family's existing notes with a timestamped marker (`[amend YYYY-MM-DDTHH:MMZ]: <text>`). Multiple amends accumulate; the audit trail is visible. If no playbook exists yet for the family, amend creates a notes-only one (so cold-start corrections still land).

#### PII discipline for amend notes

`playbook amend` notes are designed to potentially flow upstream as shared knowledge in future versions of the Printing Press. Keep them clean of user-identifying content so the upstream-contribution path stays open without retroactive scrubbing:

- **Do NOT embed** paths to user filesystems, personal API keys or tokens, user email addresses, user GitHub handles, or specific query histories tied to a single user.
- **Acceptable**: endpoint shapes, undocumented field names, API gotchas, observed schema drift, workarounds for CLI surfaces, generalizable pagination or retry tactics.

If a correction is only meaningful with user-specific context, it belongs in a personal note, not in the playbook amend.

### Measuring the loop

`reddit-ads-pp-cli learnings stats` reports recall hit rate, teach-to-reuse, playbook resolution rate, and candidate confirm/reject counts from the local `learn_events` table. Rates are null until they have a denominator; everything stays on this machine. Use it to check whether the loop is earning its keep for this CLI.

### Disabling learning

- `--no-learn` on a single command short-circuits both `recall` and the `teach` write path. Use for deterministic agent flows or tests that must not be affected by accumulated learnings.
- `REDDIT_ADS_NO_LEARN=true` in the environment globally disables the pipeline.

## Agent Feedback

When you (or the agent) notice something off about this CLI, record it:

```
reddit-ads-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
reddit-ads-pp-cli feedback --stdin < notes.txt
reddit-ads-pp-cli feedback list --json --limit 10
```

Entries are stored locally as `feedback.jsonl` under the resolved data dir. They are never POSTed unless `REDDIT_ADS_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `REDDIT_ADS_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

Write what *surprised* you, not a bug report. Short, specific, one line: that is the part that compounds.

## Output Delivery

Every command accepts `--deliver <sink>`. The output goes to the named sink in addition to (or instead of) stdout, so agents can route command results without hand-piping. Three sinks are supported:

| Sink | Effect |
|------|--------|
| `stdout` | Default; write to stdout only |
| `file:<path>` | Atomically write output to `<path>` (tmp + rename) |
| `webhook:<url>` | POST the output body to the URL (`application/json` or `application/x-ndjson` when `--compact`) |

Unknown schemes are refused with a structured error naming the supported set. Webhook failures return non-zero and log the URL + HTTP status on stderr.

## Named Profiles

A profile is a saved set of flag values, reused across invocations. Use it when a scheduled or recurring agent reuses the same saved flags while providing different input each run.

```
reddit-ads-pp-cli profile save briefing --json
reddit-ads-pp-cli --profile briefing ad-accounts get mock-value
reddit-ads-pp-cli profile list --json
reddit-ads-pp-cli profile show briefing
reddit-ads-pp-cli profile delete briefing --yes
```

Explicit flags always win over profile values; profile values win over defaults. `agent-context` lists all available profiles under `available_profiles` so introspecting agents discover them at runtime.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error (wrong arguments) |
| 3 | Resource not found |
| 4 | Authentication required |
| 5 | API error (upstream issue) |
| 7 | Rate limited (wait and retry) |
| 10 | Config error |

## Argument Parsing

Parse `$ARGUMENTS`:

1. **Empty, `help`, or `--help`** → show `reddit-ads-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → see Prerequisites above
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## MCP Server Installation

Install the MCP binary from this CLI's published public-library entry or pre-built release, then register it:

```bash
claude mcp add reddit-ads-pp-mcp -- reddit-ads-pp-mcp
```

Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which reddit-ads-pp-cli`
   If not found, offer to install (see Prerequisites at the top of this skill).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   reddit-ads-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `reddit-ads-pp-cli <command> --help`.
