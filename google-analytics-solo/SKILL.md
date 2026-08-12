---
name: pp-google-analytics-solo
description: "Flag-driven GA4 reports, one-command fan-out across every property, and a local cache for offline trends — none of which the raw Data API gives you. Trigger phrases: `run a GA4 report`, `active users for solo-prod`, `compare properties in analytics`, `GA4 realtime`, `use google-analytics`, `run google-analytics`."
author: "Pooria Arab"
license: "Apache-2.0"
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - google-analytics-solo-pp-cli
    install:
      - kind: go
        bins: [google-analytics-solo-pp-cli]
        module: github.com/mvanhorn/printing-press-library/library/marketing/google-analytics-solo/cmd/google-analytics-solo-pp-cli
---

# Google Analytics — Printing Press CLI

## Prerequisites: Install the CLI

This skill drives the `google-analytics-solo-pp-cli` binary. **You must verify the CLI is installed before invoking any command from this skill.** If it is missing, install it first:

1. Install via the Printing Press installer. It defaults binaries to `$HOME/.local/bin` on macOS/Linux and `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows:
   ```bash
   npx -y @mvanhorn/printing-press-library install google-analytics-solo --cli-only
   ```
2. Verify: `google-analytics-solo-pp-cli --version`
3. Ensure the reported install directory is on `$PATH` for the agent/runtime that will invoke this skill.

If the `npx` install fails (no Node, offline, etc.), fall back to a direct Go install (requires Go 1.26.5 or newer). This installs into `$GOPATH/bin` (default `$HOME/go/bin`), so add that directory to `$PATH` instead:

```bash
go install github.com/mvanhorn/printing-press-library/library/marketing/google-analytics-solo/cmd/google-analytics-solo-pp-cli@latest
```

If `--version` reports "command not found" after install, the runtime cannot see the binary directory on `$PATH`. Do not proceed with skill commands until verification succeeds.

The GA4 Data API is POST-with-hand-written-JSON report bodies against one numeric property at a time. This CLI turns reports into flags (--dims, --metrics, --since), fans a single report out across all your registered properties, caches results in SQLite for offline diffing, covers the full GA4 Admin API (accounts, properties, data streams, custom dimensions, audiences), and adds Measurement Protocol event ingestion. Mutating Admin/send operations are gated behind --confirm. It speaks --json/--select for agents.

## When to Use This CLI

Use this CLI when an agent or human needs GA4 programmatically: report data (single or fanned out across properties), realtime snapshots, Admin API config reads/writes, or Measurement Protocol event ingestion. Reads run freely; mutations require --confirm.

## Anti-triggers

Do not use this CLI for:
- Do not use for Universal Analytics (ga: views) — GA4 only.
- Do not run Admin deletes or 'mp send' casually — they mutate live work data and require --confirm.

## Unique Capabilities

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

## Command Reference

**account-summaries** — Manage account summaries

- `google-analytics-solo-pp-cli account-summaries` — Returns summaries of all accounts accessible by the caller.

**accounts** — Manage accounts

- `google-analytics-solo-pp-cli accounts list` — Returns all accounts accessible by the caller. Note that these accounts might not currently have GA4 properties.
- `google-analytics-solo-pp-cli accounts search-change-history-events` — Searches through all changes to an account or its children given the specified set of filters.

**accounts-provision-account-ticket** — Manage accounts provision account ticket

- `google-analytics-solo-pp-cli accounts-provision-account-ticket` — Requests a ticket for creating an account.

**data-streams** — Manage data streams


**google-analytics-admin-properties** — Manage google analytics admin properties

- `google-analytics-solo-pp-cli google-analytics-admin-properties acknowledge-user-data-collection` — Acknowledges the terms of user data collection for the specified property.
- `google-analytics-solo-pp-cli google-analytics-admin-properties create` — Creates an 'GA4' property with the specified location and attributes.
- `google-analytics-solo-pp-cli google-analytics-admin-properties list` — Returns child Properties under the specified parent Account. Only 'GA4' properties will be returned.
- `google-analytics-solo-pp-cli google-analytics-admin-properties run-access-report` — Returns a customized report of data access records.

**properties** — Manage properties

- `google-analytics-solo-pp-cli properties batch-run-pivot-reports` — Returns multiple pivot reports in a batch. All reports must be for the same GA4 Property.
- `google-analytics-solo-pp-cli properties batch-run-reports` — Returns multiple reports in a batch. All reports must be for the same GA4 Property.
- `google-analytics-solo-pp-cli properties check-compatibility` — This compatibility method lists dimensions and metrics that can be added to a report request and maintain compatibility.
- `google-analytics-solo-pp-cli properties get-metadata` — Returns metadata for dimensions and metrics available in reporting methods. Used to explore the dimensions and metrics.
- `google-analytics-solo-pp-cli properties run-pivot-report` — Returns a customized pivot report of your Google Analytics event data.
- `google-analytics-solo-pp-cli properties run-realtime-report` — Returns a customized report of realtime event data for your property.
- `google-analytics-solo-pp-cli properties run-report` — Returns a customized report of your Google Analytics event data.

**properties-create-connected-site-tag** — Manage properties create connected site tag

- `google-analytics-solo-pp-cli properties-create-connected-site-tag` — Creates a connected site tag for a Universal Analytics property.

**properties-delete-connected-site-tag** — Manage properties delete connected site tag

- `google-analytics-solo-pp-cli properties-delete-connected-site-tag` — Deletes a connected site tag for a Universal Analytics property. Note: this has no effect on GA4 properties.

**properties-fetch-automated-ga4-configuration-opt-out** — Manage properties fetch automated ga4 configuration opt out

- `google-analytics-solo-pp-cli properties-fetch-automated-ga4-configuration-opt-out` — Fetches the opt out status for the automated GA4 setup process for a UA property.

**properties-fetch-connected-ga4-property** — Manage properties fetch connected ga4 property

- `google-analytics-solo-pp-cli properties-fetch-connected-ga4-property` — Given a specified UA property, looks up the GA4 property connected to it. Note: this cannot be used with GA4 properties.

**properties-list-connected-site-tags** — Manage properties list connected site tags

- `google-analytics-solo-pp-cli properties-list-connected-site-tags` — Lists the connected site tags for a Universal Analytics property. A maximum of 20 connected site tags will be returned.

**properties-set-automated-ga4-configuration-opt-out** — Manage properties set automated ga4 configuration opt out

- `google-analytics-solo-pp-cli properties-set-automated-ga4-configuration-opt-out` — Sets the opt out status for the automated GA4 setup process for a UA property. Note: this has no effect on GA4 property.


### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
google-analytics-solo-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

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

## Auth Setup

Authenticates to Google through Application Default Credentials, which unifies three identity modes: a service-account key (GOOGLE_APPLICATION_CREDENTIALS), a gcloud user login (run 'gcloud auth application-default login'), or workload identity on GCP. A stored OAuth refresh token is a fallback. Scope is read-only analytics; write operations still require the --confirm flag. Measurement Protocol commands use a separate GA4_MP_API_SECRET, not OAuth.

Run `google-analytics-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  google-analytics-solo-pp-cli account-summaries --agent --select id,name,status
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

- Use `--home <dir>` for one invocation, or set `GOOGLE_ANALYTICS_SOLO_HOME=<dir>` to relocate all four path kinds under one root.
- Use per-kind env vars only when a specific kind must diverge: `GOOGLE_ANALYTICS_SOLO_CONFIG_DIR`, `GOOGLE_ANALYTICS_SOLO_DATA_DIR`, `GOOGLE_ANALYTICS_SOLO_STATE_DIR`, `GOOGLE_ANALYTICS_SOLO_CACHE_DIR`.
- Resolution order is per-kind env var, `--home`, `GOOGLE_ANALYTICS_SOLO_HOME`, XDG (`XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME`), then platform defaults.
- `config` contains settings like `config.toml` and profiles. `data` contains `credentials.toml`, `data.db`, cookies, and auth sidecars. `state` contains persisted queries, jobs, and `teach.log`. `cache` contains regenerable HTTP/cache files.
- Stored secrets live in `credentials.toml` under the data dir. Existing legacy `config.toml` secrets are read for compatibility and leave `config.toml` on the first auth write.
- Run `google-analytics-solo-pp-cli doctor --fail-on warn` to surface path and credential-location warnings. `agent-context` exposes a schema v4 `paths` block for agents that need the resolved dirs.
- For MCP, pass relocation through the MCP host config. The MCP binary does not inherit CLI flags:

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

Fleet precedence: an inherited per-kind env var overrides an explicit `--home` for that kind. Use `GOOGLE_ANALYTICS_SOLO_HOME` or per-kind vars as durable fleet levers, and use `--home` only for a single invocation. Relocation is not reversible by unsetting env vars; move files manually before clearing `GOOGLE_ANALYTICS_SOLO_HOME`, or `doctor` will not find credentials left under the former root.

## Automatic learning

This CLI ships a self-capturing learning loop. The CLI does its own bookkeeping: every invocation is journaled locally, a failed flag followed by a corrected retry auto-derives a `flag_alias` candidate, and a `teach` on a query family without a playbook auto-synthesizes a `playbook_candidate` from the session's journal. Your job is judgment only: `recall` first, act on surfaced candidates, `teach` the final answer, `playbook amend` when you observe a correction. You never record failures by hand.

### Step 1: `recall` before any discovery

Before list/search/drill commands on a new user question, run:

```bash
google-analytics-solo-pp-cli recall "<user's question>" --agent
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
      "next_action": ["<trial command>", "google-analytics-solo-pp-cli learnings confirm 12"] }
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
       materially more, record the divergence via `google-analytics-solo-pp-cli playbook amend`
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

Candidate judgment details: `learnings confirm <id>` prints the candidate's full payload before materializing it - check that the printed payload matches the behavior you verified. `learnings reject <id>` tombstones the derivation signature so the same candidate does not resurface. The envelope carries only the few candidates worth acting on now; `google-analytics-solo-pp-cli learnings candidates` lists the full open set.

Graceful degradation: if `learnings confirm` is an unknown command, you are driving an older binary - ignore the candidates guidance and follow the rest of the protocol.

### Step 3: always read `warnings`

- `low_confidence`: row exists at `confidence<2`. Treat as a hint, not a skip-discovery hit.
- `resource_not_in_store`: the local store doesn't have the resource the learning points at. The match validator couldn't classify entities — direct-fetch and re-evaluate.
- `cross_alias_match` (per-result): the row was taught under a different alias and matched the live query's canonical via `entity_lookups` (e.g., a "USA" teach satisfying a "United States" recall). Trust the resource_id.
- `similar_shape_different_entity:<canonical>` (top-level): a structurally matching row exists but its canonical entity differs from the live query's. Treated as cold start; the warning carries the conflicting canonical as a hint, but the row is NOT promoted into Results.
- `ambiguous_alias` (top-level): a single query entity resolved to multiple canonicals (e.g., "Cards" → Arizona Cardinals + St. Louis Cardinals). Surface the ambiguity from context before committing to a resource.
- `candidates_present` (top-level): the envelope carries a `candidates` section. Handle it via the candidates branch in Step 2 before anything else.
- `lookup_refresh_available` (top-level): an entity in the query has no lookup row yet, but synced data could provide one. Run `google-analytics-solo-pp-cli sync` to refresh entity lookups.
- Top-level `no_learnings_for_query_family`: the table had no rows above the Jaccard floor. Pure cold start.

### Step 4: `teach &` after finalizing your response - always

Teaching is unconditional. After resolving a query the store could not answer, background-teach the final resource mapping - no call-count threshold, no judging whether it was "worth" learning. The teach is the anchor of the loop: it triggers playbook synthesis for a family without a playbook, and same-referent phrasings fold into one family so near-duplicate teaches do not fragment the store. Fire it after assembling your user-facing response but BEFORE emitting it, with a shell `&` so the call returns immediately:

```bash
google-analytics-solo-pp-cli teach --query "<user's question>" --resource-type <type> --resource <id1> --resource <id2>
# (append shell `&` to background it)
```

Silent on success. Errors only land in `teach.log` under the resolved state dir. Teach the **most specific** resource - if the user asked a broad question and you walked through parent records to find the specific answer, teach the leaf id, not the parent. The CLI uses seeded `entity_lookups` for cross-alias resolution at recall time, so a teach under one alias (e.g., "Niners") satisfies future queries under another alias (e.g., "49ers", "San Francisco") automatically.

PII rule: teach the structural question with identifiers stripped - never include names, emails, phone numbers, account ids, or other personal identifiers in taught queries or notes. The CLI scans teach queries for obvious email/phone shapes and warns, but does not block; strip before teaching rather than relying on the warning.

### Step 5: playbooks - optional flags, automatic synthesis

You do not need to decide whether a session "deserves" a playbook: a teach on a family without one auto-synthesizes a `playbook_candidate` from the session's journal, and the next session judges it via confirm/reject. Attach explicit playbook flags only when you already hold choreography worth recording verbatim - workarounds the CLI didn't surface (silently-dropped flags, undocumented params, pagination tricks, payload gotchas). Prefer the **integrated one-call form** - record the resource learning and the playbook in the same `teach` invocation:

```bash
# Common case: record both the resource learning AND the playbook in one call.
google-analytics-solo-pp-cli teach \
  --query "<user's question>" \
  --resource <id> \
  --playbook-file ~/playbooks/<shape>.json \
  --playbook-notes-file ~/playbooks/<shape>-notes.md
# (append shell `&` to background it)

# Alternate: playbook-only (no resource to record alongside).
google-analytics-solo-pp-cli teach-playbook \
  --query "<user's question>" \
  --playbook-file ~/playbooks/<shape>.json \
  --notes-file ~/playbooks/<shape>-notes.md
```

Playbook files are JSON with `steps`, `entity_slots`, `expected_tool_calls`. Notes files are markdown carrying the gotchas verbatim. File-free callers (MCP-only agents) pass the same content inline: `--playbook-json` and `--playbook-notes` on the integrated `teach` form, `--playbook-json` and `--notes` on `teach-playbook`. On the integrated `teach` form, the playbook flags are optional - omit them entirely for a resource-only teach. On the standalone `teach-playbook` form, at least one of the playbook and notes flags must be set; both empty is rejected. Playbooks are keyed on the structural query family (entities stripped) so a recipe taught from one entity-shaped query applies to every other query of the same shape, with `slots_resolved` binding the live query's canonical at recall time.

When you DO find a playbook on a future recall, treat it as ground truth: replay the steps with `slots_resolved` substitutions, skip the discovery that the choreography already documents, and read `notes` before any step.

### Step 6: `playbook amend &` when your debug response identifies a correction

If your debug-protocol response identifies a concrete correction the notes or playbook should know — a workaround, an undocumented endpoint shape, a stale field name, observed schema drift, an empty-payload fallback — fire `playbook amend` BEFORE emitting your user-facing response. Same fire-and-forget posture as `teach`.

```bash
google-analytics-solo-pp-cli playbook amend \
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

`google-analytics-solo-pp-cli learnings stats` reports recall hit rate, teach-to-reuse, playbook resolution rate, and candidate confirm/reject counts from the local `learn_events` table. Rates are null until they have a denominator; everything stays on this machine. Use it to check whether the loop is earning its keep for this CLI.

### Disabling learning

- `--no-learn` on a single command short-circuits both `recall` and the `teach` write path. Use for deterministic agent flows or tests that must not be affected by accumulated learnings.
- `GOOGLE_ANALYTICS_SOLO_NO_LEARN=true` in the environment globally disables the pipeline.

## Agent Feedback

When you (or the agent) notice something off about this CLI, record it:

```
google-analytics-solo-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
google-analytics-solo-pp-cli feedback --stdin < notes.txt
google-analytics-solo-pp-cli feedback list --json --limit 10
```

Entries are stored locally as `feedback.jsonl` under the resolved data dir. They are never POSTed unless `GOOGLE_ANALYTICS_SOLO_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `GOOGLE_ANALYTICS_SOLO_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

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
google-analytics-solo-pp-cli profile save briefing --json
google-analytics-solo-pp-cli --profile briefing account-summaries
google-analytics-solo-pp-cli profile list --json
google-analytics-solo-pp-cli profile show briefing
google-analytics-solo-pp-cli profile delete briefing --yes
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

1. **Empty, `help`, or `--help`** → show `google-analytics-solo-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → see Prerequisites above
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## MCP Server Installation

1. Install the MCP server:
   ```bash
   go install github.com/mvanhorn/printing-press-library/library/marketing/google-analytics-solo/cmd/google-analytics-solo-pp-mcp@latest
   ```
2. Register with Claude Code:
   ```bash
   claude mcp add google-analytics-solo-pp-mcp -- google-analytics-solo-pp-mcp
   ```
3. Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which google-analytics-solo-pp-cli`
   If not found, offer to install (see Prerequisites at the top of this skill).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   google-analytics-solo-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `google-analytics-solo-pp-cli <command> --help`.
