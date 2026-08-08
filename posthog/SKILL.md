---
name: pp-posthog
description: "Printing Press CLI for Posthog."
author: "Pooria Arab"
license: "Apache-2.0"
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - posthog-pp-cli
    install:
      - kind: go
        bins: [posthog-pp-cli]
        module: github.com/mvanhorn/printing-press-library/library/monitoring/posthog/cmd/posthog-pp-cli
---

# Posthog — Printing Press CLI

## Prerequisites: Install the CLI

This skill drives the `posthog-pp-cli` binary. **You must verify the CLI is installed before invoking any command from this skill.** If it is missing, install it first:

1. Install via the Printing Press installer. It defaults binaries to `$HOME/.local/bin` on macOS/Linux and `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows:
   ```bash
   npx -y @mvanhorn/printing-press-library install posthog --cli-only
   ```
2. Verify: `posthog-pp-cli --version`
3. Ensure the reported install directory is on `$PATH` for the agent/runtime that will invoke this skill.

If the `npx` install fails (no Node, offline, etc.), fall back to a direct Go install (requires Go 1.26.5 or newer). This installs into `$GOPATH/bin` (default `$HOME/go/bin`), so add that directory to `$PATH` instead:

```bash
go install github.com/mvanhorn/printing-press-library/library/monitoring/posthog/cmd/posthog-pp-cli@latest
```

If `--version` reports "command not found" after install, the runtime cannot see the binary directory on `$PATH`. Do not proceed with skill commands until verification succeeds.



## Command Reference

**account-notes** — Manage account notes

- `posthog-pp-cli account-notes <project_id>` — List

**account-relationship-definitions** — Manage account relationship definitions

- `posthog-pp-cli account-relationship-definitions create` — Create
- `posthog-pp-cli account-relationship-definitions destroy` — Destroy
- `posthog-pp-cli account-relationship-definitions list` — List
- `posthog-pp-cli account-relationship-definitions partial-update` — Partial update
- `posthog-pp-cli account-relationship-definitions retrieve` — Retrieve
- `posthog-pp-cli account-relationship-definitions update` — Update

**accounts** — Manage accounts

- `posthog-pp-cli accounts create` — Create
- `posthog-pp-cli accounts destroy` — Destroy
- `posthog-pp-cli accounts list` — List
- `posthog-pp-cli accounts partial-update` — Partial update
- `posthog-pp-cli accounts retrieve` — Retrieve
- `posthog-pp-cli accounts update` — Update

**actions** — Manage actions

- `posthog-pp-cli actions bulk-update-tags-create` — Bulk update tags on multiple objects.
- `posthog-pp-cli actions create` — Create
- `posthog-pp-cli actions destroy` — Hard delete of this model is not allowed. Use a patch API call to set 'deleted' to true
- `posthog-pp-cli actions list` — List
- `posthog-pp-cli actions partial-update` — Partial update
- `posthog-pp-cli actions retrieve` — Retrieve
- `posthog-pp-cli actions update` — Update

**activity-log** — Manage activity log

- `posthog-pp-cli activity-log <project_id>` — List

**advanced-activity-logs** — Manage advanced activity logs

- `posthog-pp-cli advanced-activity-logs available-filters-retrieve` — Available filters retrieve
- `posthog-pp-cli advanced-activity-logs export-create` — Export create
- `posthog-pp-cli advanced-activity-logs list` — List

**alerts** — Manage alerts

- `posthog-pp-cli alerts create` — Create
- `posthog-pp-cli alerts destroy` — Destroy
- `posthog-pp-cli alerts list` — List
- `posthog-pp-cli alerts partial-update` — Partial update
- `posthog-pp-cli alerts retrieve` — Retrieve
- `posthog-pp-cli alerts simulate-create` — Simulate a detector on an insight's historical data. Read-only — no AlertCheck records are created.
- `posthog-pp-cli alerts update` — Update

**annotations** — Manage annotations

- `posthog-pp-cli annotations create` — Create, Read, Update and Delete annotations. [See docs](https://posthog.
- `posthog-pp-cli annotations destroy` — Hard delete of this model is not allowed. Use a patch API call to set 'deleted' to true
- `posthog-pp-cli annotations list` — Create, Read, Update and Delete annotations. [See docs](https://posthog.
- `posthog-pp-cli annotations partial-update` — Create, Read, Update and Delete annotations. [See docs](https://posthog.
- `posthog-pp-cli annotations retrieve` — Create, Read, Update and Delete annotations. [See docs](https://posthog.
- `posthog-pp-cli annotations update` — Create, Read, Update and Delete annotations. [See docs](https://posthog.

**announcements** — Manage announcements

- `posthog-pp-cli announcements channels-list` — Slack channels the SupportHog bot can post to, labeled by customer account name.
- `posthog-pp-cli announcements create` — Create
- `posthog-pp-cli announcements list` — List
- `posthog-pp-cli announcements retrieve` — Retrieve

**approval-policies** — Manage approval policies

- `posthog-pp-cli approval-policies create` — Create
- `posthog-pp-cli approval-policies destroy` — Destroy
- `posthog-pp-cli approval-policies list` — List
- `posthog-pp-cli approval-policies partial-update` — Partial update
- `posthog-pp-cli approval-policies retrieve` — Retrieve
- `posthog-pp-cli approval-policies update` — Update

**batch-exports** — Manage batch exports

- `posthog-pp-cli batch-exports create` — Create
- `posthog-pp-cli batch-exports destroy` — Destroy
- `posthog-pp-cli batch-exports list` — List
- `posthog-pp-cli batch-exports partial-update` — Partial update
- `posthog-pp-cli batch-exports retrieve` — Retrieve
- `posthog-pp-cli batch-exports run-test-step-new-create` — Run test step new create
- `posthog-pp-cli batch-exports test-retrieve` — Test retrieve
- `posthog-pp-cli batch-exports update` — Update

**business-knowledge** — Manage business knowledge

- `posthog-pp-cli business-knowledge documents-search-list` — Read-only access to parsed knowledge documents.
- `posthog-pp-cli business-knowledge documents-window-list` — Read-only access to parsed knowledge documents.
- `posthog-pp-cli business-knowledge gap-suggestions-accept-create` — Surfaces topics the support AI couldn't answer from the knowledge base.
- `posthog-pp-cli business-knowledge gap-suggestions-accept-topic-create` — Accept all pending suggestions for a normalized topic cluster.
- `posthog-pp-cli business-knowledge gap-suggestions-dismiss-create` — Surfaces topics the support AI couldn't answer from the knowledge base.
- `posthog-pp-cli business-knowledge gap-suggestions-dismiss-topic-create` — Dismiss all pending suggestions for a normalized topic cluster.
- `posthog-pp-cli business-knowledge gap-suggestions-list` — Surfaces topics the support AI couldn't answer from the knowledge base.
- `posthog-pp-cli business-knowledge sources-create` — Sources create
- `posthog-pp-cli business-knowledge sources-destroy` — Sources destroy
- `posthog-pp-cli business-knowledge sources-list` — Sources list
- `posthog-pp-cli business-knowledge sources-partial-update` — Sources partial update
- `posthog-pp-cli business-knowledge sources-refresh-create` — Sources refresh create
- `posthog-pp-cli business-knowledge sources-retrieve` — Sources retrieve
- `posthog-pp-cli business-knowledge sources-text-retrieve` — Sources text retrieve

**calendar-sync** — Manage calendar sync

- `posthog-pp-cli calendar-sync list` — Calendar-sync controls for Customer analytics settings.
- `posthog-pp-cli calendar-sync sync-now-create` — Start a sync run for one connected Google Calendar immediately, outside the hourly schedule.

**canvases** — Manage canvases

- `posthog-pp-cli canvases create` — Create a new, empty canvas in a channel; give it source by publishing a project.
- `posthog-pp-cli canvases destroy` — Canvases: agent-built sandboxed browser apps, filed into channels.
- `posthog-pp-cli canvases list` — Canvases: agent-built sandboxed browser apps, filed into channels.
- `posthog-pp-cli canvases partial-update` — Update canvas metadata (name, author context, pin, generation-task pointer).
- `posthog-pp-cli canvases retrieve` — Canvases: agent-built sandboxed browser apps, filed into channels.

**change-requests** — Manage change requests

- `posthog-pp-cli change-requests list` — List
- `posthog-pp-cli change-requests retrieve` — Retrieve

**code** — Manage code

- `posthog-pp-cli code invites-check-access-retrieve` — Check whether the authenticated user has access to PostHog Desktop and to Loops.
- `posthog-pp-cli code invites-redeem-create` — Redeem a PostHog Desktop invite code to enable access.

**cohorts** — Manage cohorts

- `posthog-pp-cli cohorts all-activity-retrieve` — All activity retrieve
- `posthog-pp-cli cohorts create` — Create
- `posthog-pp-cli cohorts destroy` — Hard delete of this model is not allowed. Use a patch API call to set 'deleted' to true
- `posthog-pp-cli cohorts list` — List
- `posthog-pp-cli cohorts partial-update` — Partial update
- `posthog-pp-cli cohorts retrieve` — Retrieve
- `posthog-pp-cli cohorts update` — Update

**comments** — Manage comments

- `posthog-pp-cli comments count-retrieve` — Count retrieve
- `posthog-pp-cli comments create` — Create a comment.
- `posthog-pp-cli comments destroy` — Hard delete of this model is not allowed. Use a patch API call to set 'deleted' to true
- `posthog-pp-cli comments list` — List
- `posthog-pp-cli comments partial-update` — Partial update
- `posthog-pp-cli comments retrieve` — Retrieve
- `posthog-pp-cli comments update` — Update

**conversations** — Manage conversations

- `posthog-pp-cli conversations create` — Unified endpoint that handles both conversation creation and streaming.
- `posthog-pp-cli conversations destroy` — Delete a conversation.
- `posthog-pp-cli conversations list` — List
- `posthog-pp-cli conversations retrieve` — Retrieve
- `posthog-pp-cli conversations tickets-ai-feedback-create` — Record reviewer feedback on an AI reply, captured to the internal analytics project.
- `posthog-pp-cli conversations tickets-bulk-update-status-create` — Update the status of multiple tickets in a single request.
- `posthog-pp-cli conversations tickets-bulk-update-tags-create` — Bulk update tags on multiple objects.
- `posthog-pp-cli conversations tickets-compose-create` — Create a new outbound ticket and send the first message to the customer.
- `posthog-pp-cli conversations tickets-destroy` — Tickets destroy
- `posthog-pp-cli conversations tickets-list` — List tickets with person data attached.
- `posthog-pp-cli conversations tickets-messages-list` — Return the message thread for a ticket, ordered chronologically (paginated).
- `posthog-pp-cli conversations tickets-notes-destroy` — Soft-delete a private note on a ticket. Only the note's author can delete it.
- `posthog-pp-cli conversations tickets-notes-partial-update` — Update a private note on a ticket. Only the note's author can edit it.
- `posthog-pp-cli conversations tickets-partial-update` — Tickets partial update
- `posthog-pp-cli conversations tickets-reply-create` — Post a reply or internal note to a ticket.
- `posthog-pp-cli conversations tickets-retrieve` — Get single ticket and mark as read by team.
- `posthog-pp-cli conversations tickets-unread-count-retrieve` — Get total unread ticket count for the team.
- `posthog-pp-cli conversations tickets-update` — Handle ticket updates including assignee changes.
- `posthog-pp-cli conversations views-create` — Views create
- `posthog-pp-cli conversations views-destroy` — Views destroy
- `posthog-pp-cli conversations views-list` — Views list
- `posthog-pp-cli conversations views-partial-update` — Views partial update
- `posthog-pp-cli conversations views-retrieve` — Views retrieve

**custom-property-definitions** — Manage custom property definitions

- `posthog-pp-cli custom-property-definitions create` — Create
- `posthog-pp-cli custom-property-definitions destroy` — Destroy
- `posthog-pp-cli custom-property-definitions list` — List
- `posthog-pp-cli custom-property-definitions partial-update` — Partial update
- `posthog-pp-cli custom-property-definitions retrieve` — Retrieve
- `posthog-pp-cli custom-property-definitions update` — Update
- `posthog-pp-cli custom-property-definitions values-retrieve` — Values retrieve

**custom-property-sources** — Manage custom property sources

- `posthog-pp-cli custom-property-sources create` — Create
- `posthog-pp-cli custom-property-sources destroy` — Destroy
- `posthog-pp-cli custom-property-sources list` — List
- `posthog-pp-cli custom-property-sources partial-update` — Partial update
- `posthog-pp-cli custom-property-sources retrieve` — Retrieve
- `posthog-pp-cli custom-property-sources update` — Update

**customer-analytics** — Manage customer analytics

- `posthog-pp-cli customer-analytics` — List accounts with external IDs and their active relationship assignments.

**customer-journeys** — Manage customer journeys

- `posthog-pp-cli customer-journeys create` — Create
- `posthog-pp-cli customer-journeys destroy` — Destroy
- `posthog-pp-cli customer-journeys list` — List
- `posthog-pp-cli customer-journeys partial-update` — Partial update
- `posthog-pp-cli customer-journeys retrieve` — Retrieve
- `posthog-pp-cli customer-journeys update` — Update

**customer-profile-configs** — Manage customer profile configs

- `posthog-pp-cli customer-profile-configs create` — Create
- `posthog-pp-cli customer-profile-configs destroy` — Destroy
- `posthog-pp-cli customer-profile-configs list` — List
- `posthog-pp-cli customer-profile-configs partial-update` — Partial update
- `posthog-pp-cli customer-profile-configs retrieve` — Retrieve
- `posthog-pp-cli customer-profile-configs update` — Update

**dashboard-templates** — Manage dashboard templates

- `posthog-pp-cli dashboard-templates copy-between-projects-create` — Creates a new team-scoped template in the **target** project (URL)
- `posthog-pp-cli dashboard-templates create` — Create
- `posthog-pp-cli dashboard-templates destroy` — Hard delete of this model is not allowed. Use a patch API call to set 'deleted' to true
- `posthog-pp-cli dashboard-templates json-schema-retrieve` — Json schema retrieve
- `posthog-pp-cli dashboard-templates list` — List
- `posthog-pp-cli dashboard-templates partial-update` — Partial update
- `posthog-pp-cli dashboard-templates retrieve` — Retrieve
- `posthog-pp-cli dashboard-templates update` — Update

**dashboards** — Manage dashboards

- `posthog-pp-cli dashboards bulk-update-tags-create` — Bulk update tags on multiple objects.
- `posthog-pp-cli dashboards create` — Create
- `posthog-pp-cli dashboards create-from-template-json-create` — Create from template json create
- `posthog-pp-cli dashboards create-unlisted-create` — Creates an unlisted dashboard from template by tag. Enforces uniqueness (one per tag per team).
- `posthog-pp-cli dashboards destroy` — Hard delete of this model is not allowed. Use a patch API call to set 'deleted' to true
- `posthog-pp-cli dashboards list` — List
- `posthog-pp-cli dashboards partial-update` — Partial update
- `posthog-pp-cli dashboards retrieve` — Retrieve
- `posthog-pp-cli dashboards update` — Update
- `posthog-pp-cli dashboards widget-catalog-retrieve` — List registered dashboard widget types and per-type config_schema documentation for agents.

**data-catalog** — Manage data catalog

- `posthog-pp-cli data-catalog certifications-certify-create` — Mark the target as certified (prefer this source).
- `posthog-pp-cli data-catalog certifications-create` — Trust marks on warehouse tables and views. Reads exclude soft-deleted targets.
- `posthog-pp-cli data-catalog certifications-deprecate-create` — Mark the target as deprecated (avoid this source).
- `posthog-pp-cli data-catalog certifications-destroy` — Trust marks on warehouse tables and views. Reads exclude soft-deleted targets.
- `posthog-pp-cli data-catalog certifications-list` — Trust marks on warehouse tables and views. Reads exclude soft-deleted targets.
- `posthog-pp-cli data-catalog certifications-retrieve` — Trust marks on warehouse tables and views. Reads exclude soft-deleted targets.
- `posthog-pp-cli data-catalog metrics-approve-create` — Bless a metric as canonical. Returns 409 while the metric is drifted from its insight.
- `posthog-pp-cli data-catalog metrics-create` — Create a metric, or refine the one already holding this name for the team.
- `posthog-pp-cli data-catalog metrics-destroy` — CRUD for catalog metrics, addressed by their reserved ``name`` (e.g. /metrics/mrr/).
- `posthog-pp-cli data-catalog metrics-list` — CRUD for catalog metrics, addressed by their reserved ``name`` (e.g. /metrics/mrr/).
- `posthog-pp-cli data-catalog metrics-partial-update` — CRUD for catalog metrics, addressed by their reserved ``name`` (e.g. /metrics/mrr/).
- `posthog-pp-cli data-catalog metrics-refresh-from-insight-create` — Re-snapshot the linked insight's current query into the definition.
- `posthog-pp-cli data-catalog metrics-retrieve` — CRUD for catalog metrics, addressed by their reserved ``name`` (e.g. /metrics/mrr/).
- `posthog-pp-cli data-catalog metrics-run-create` — Execute the metric's definition and return the normalized result envelope.
- `posthog-pp-cli data-catalog metrics-update` — CRUD for catalog metrics, addressed by their reserved ``name`` (e.g. /metrics/mrr/).
- `posthog-pp-cli data-catalog relationship-proposals-accept-create` — Promote the proposal to a real warehouse join after re-validating and probing it.
- `posthog-pp-cli data-catalog relationship-proposals-create` — Reviewed join facts. Accepting one promotes it to a real DataWarehouseJoin; rejections persist.
- `posthog-pp-cli data-catalog relationship-proposals-list` — Reviewed join facts. Accepting one promotes it to a real DataWarehouseJoin; rejections persist.
- `posthog-pp-cli data-catalog relationship-proposals-reject-create` — Reject the proposal. Persists forever so the pair is never re-proposed.
- `posthog-pp-cli data-catalog relationship-proposals-retrieve` — Reviewed join facts. Accepting one promotes it to a real DataWarehouseJoin; rejections persist.

**data-color-themes** — Manage data color themes

- `posthog-pp-cli data-color-themes create` — Create
- `posthog-pp-cli data-color-themes destroy` — Destroy
- `posthog-pp-cli data-color-themes list` — List
- `posthog-pp-cli data-color-themes partial-update` — Partial update
- `posthog-pp-cli data-color-themes retrieve` — Retrieve
- `posthog-pp-cli data-color-themes update` — Update

**data-modeling-jobs** — Manage data modeling jobs

- `posthog-pp-cli data-modeling-jobs list` — List data modeling jobs which are 'runs' for our saved queries.
- `posthog-pp-cli data-modeling-jobs recent-retrieve` — Get the most recent non-running job for each saved query from the v2 backend.
- `posthog-pp-cli data-modeling-jobs retrieve` — List data modeling jobs which are 'runs' for our saved queries.
- `posthog-pp-cli data-modeling-jobs running-retrieve` — Get all currently running jobs from the v2 backend.

**data-warehouse** — Manage data warehouse

- `posthog-pp-cli data-warehouse check-database-name-retrieve` — Check if a database name is available.
- `posthog-pp-cli data-warehouse check-schema-name-retrieve` — Check if a schema name is free within the organization's managed warehouse.
- `posthog-pp-cli data-warehouse completed-activity-retrieve` — Returns completed/non-running activities (jobs with status 'Completed'). Supports pagination and cutoff time filtering.
- `posthog-pp-cli data-warehouse data-health-issues-retrieve` — Returns failed/disabled data pipeline items for the Pipeline status side panel.
- `posthog-pp-cli data-warehouse data-ops-dashboard-retrieve` — Returns the data ops overview dashboard ID for this team, creating it if it doesn't exist yet.
- `posthog-pp-cli data-warehouse delete-org-destroy` — Remove the organization's provisioning record after teardown, freeing its warehouse name.
- `posthog-pp-cli data-warehouse deprovision-create` — Start deprovisioning the organization's managed warehouse. Restricted to organization admins.
- `posthog-pp-cli data-warehouse job-stats-retrieve` — Returns success and failed job statistics for the last 1, 7, or 30 days.
- `posthog-pp-cli data-warehouse managed-warehouse-data-status-retrieve` — Get events, persons, and imported source readiness for the managed warehouse.
- `posthog-pp-cli data-warehouse managed-warehouse-source-schemas-retrieve` — Per-schema backfill and live import status for one source
- `posthog-pp-cli data-warehouse onboard-team-create` — Onboard this project onto the organization's existing managed warehouse.
- `posthog-pp-cli data-warehouse property-values-retrieve` — API endpoints for data warehouse aggregate statistics and operations.
- `posthog-pp-cli data-warehouse provision-create` — Start provisioning a managed warehouse for this organization (shared by all its teams).
- `posthog-pp-cli data-warehouse reset-password-create` — Reset the root password for the managed warehouse.
- `posthog-pp-cli data-warehouse running-activity-retrieve` — Returns currently running activities (jobs with status 'Running'). Supports pagination and cutoff time filtering.
- `posthog-pp-cli data-warehouse total-rows-stats-retrieve` — Returns aggregated statistics for the data warehouse total rows processed within the current billing period.
- `posthog-pp-cli data-warehouse warehouse-status-retrieve` — Get the current provisioning status of the managed warehouse, with this project's onboarding state.

**dataset-items** — Manage dataset items

- `posthog-pp-cli dataset-items create` — Create an item and its first immutable version. An identical client item ID retry returns the existing item.
- `posthog-pp-cli dataset-items list` — List a dataset's current items or its exact contents at a prior revision.
- `posthog-pp-cli dataset-items partial-update` — Create a new immutable item version from editable fields.
- `posthog-pp-cli dataset-items retrieve` — Retrieve the current item version or the version visible at an exact dataset revision.

**datasets** — Manage datasets

- `posthog-pp-cli datasets create` — Create an empty dataset. Its first revision is created with its first item.
- `posthog-pp-cli datasets list` — List active datasets by default, or archived datasets when requested.
- `posthog-pp-cli datasets partial-update` — Update descriptive dataset fields without changing its revision.
- `posthog-pp-cli datasets retrieve` — Retrieve an active or archived dataset.

**early-access-feature** — Manage early access feature

- `posthog-pp-cli early-access-feature create` — Create
- `posthog-pp-cli early-access-feature destroy` — Destroy
- `posthog-pp-cli early-access-feature list` — List
- `posthog-pp-cli early-access-feature partial-update` — Partial update
- `posthog-pp-cli early-access-feature retrieve` — Retrieve
- `posthog-pp-cli early-access-feature update` — Update

**elements** — Manage elements

- `posthog-pp-cli elements create` — Create
- `posthog-pp-cli elements destroy` — Destroy
- `posthog-pp-cli elements list` — List
- `posthog-pp-cli elements partial-update` — Partial update
- `posthog-pp-cli elements retrieve` — Retrieve
- `posthog-pp-cli elements stats-retrieve` — Counts of $autocapture, $rageclick, and $dead_click events grouped by the element chain they occurred on
- `posthog-pp-cli elements update` — Update
- `posthog-pp-cli elements values-list` — Values list

**endpoints** — Manage endpoints

- `posthog-pp-cli endpoints create` — Create a new endpoint.
- `posthog-pp-cli endpoints destroy` — Delete an endpoint and clean up materialized query.
- `posthog-pp-cli endpoints last-execution-times-create` — Get the most recent execution time per endpoint (endpoint-level).
- `posthog-pp-cli endpoints list` — List all endpoints for the team.
- `posthog-pp-cli endpoints materialization-conditions-retrieve` — Get the source code of the live materialization checks, plus the rewrite contract.
- `posthog-pp-cli endpoints partial-update` — Update an existing endpoint.
- `posthog-pp-cli endpoints retrieve` — Retrieve an endpoint, or a specific version via ?version=N.
- `posthog-pp-cli endpoints update` — Update an existing endpoint. Parameters are optional. Pass version in body or ?

**engineering-analytics** — Manage engineering analytics

- `posthog-pp-cli engineering-analytics author-workflow-costs` — One author's estimated CI cost split by workflow over a window (date_from default -30d), highest spend first.
- `posthog-pp-cli engineering-analytics broken-tests` — The broken-tests triage panel
- `posthog-pp-cli engineering-analytics ci-cards` — Headline counts for the open-PR backlog: open PRs, distinct repos, stuck PRs (open, non-draft, non-bot
- `posthog-pp-cli engineering-analytics ci-failure-logs` — The thinned CI failure logs for a pull request, grouped by failed job.
- `posthog-pp-cli engineering-analytics ci-signals-config-retrieve` — Return the atomic CI Signals configuration and aggregate GitHub warehouse sync status.
- `posthog-pp-cli engineering-analytics ci-signals-config-update` — Enable or disable all CI signal detectors in one transaction.
- `posthog-pp-cli engineering-analytics current-branch-health` — Current default-branch CI verdict over the fixed last-24-hours window.
- `posthog-pp-cli engineering-analytics flaky-tests` — The active test-health queue: pytest and Jest tests worth acting on now, from the per-test CI spans
- `posthog-pp-cli engineering-analytics job-aggregates` — Per-job aggregates for one workflow over a window (default -30d)
- `posthog-pp-cli engineering-analytics master-failures` — Default-branch failures over a window (default -24h), grouped error-tracking style by (workflow, de-sharded failing job)
- `posthog-pp-cli engineering-analytics pr-cost` — Estimated CI cost for a pull request, summed over the jobs of all its workflow runs.
- `posthog-pp-cli engineering-analytics pr-lifecycle` — The timeline of a single pull request: header plus ordered events (opened, CI started/finished, merged or closed).
- `posthog-pp-cli engineering-analytics pr-runs` — Every workflow run attributed to a pull request, across all its commits (grouped by head SHA client-side), newest first.
- `posthog-pp-cli engineering-analytics pull-requests` — Open pull requests plus any merged or closed since date_from (default -30d), newest first
- `posthog-pp-cli engineering-analytics quarantine` — The repository's checked-in .test_quarantine.
- `posthog-pp-cli engineering-analytics quarantine-request` — Opens a pull request that edits the repository's checked-in .test_quarantine.
- `posthog-pp-cli engineering-analytics repo-overview` — Repo-level headline aggregates over a window (default -30d): run count, success rate, re-run cycles
- `posthog-pp-cli engineering-analytics repo-run-activity` — Default-branch health as compact chart points over a window (default -30d), newest first
- `posthog-pp-cli engineering-analytics resolve-branch` — Resolve a git branch to the pull request(s)
- `posthog-pp-cli engineering-analytics run-failure-logs` — The thinned CI failure logs of one workflow run
- `posthog-pp-cli engineering-analytics sources` — The team's selectable GitHub repositories, oldest source first — one entry per repository a source is configured to sync
- `posthog-pp-cli engineering-analytics team-ci-activity` — One owning team's CI test activity: per-test current-vs-prior signal pairs (the before/after comparison)
- `posthog-pp-cli engineering-analytics team-ci-health` — Per-owning-team rollup of the CI test surfaces each team owns
- `posthog-pp-cli engineering-analytics team-merge-trend` — One team's daily time-to-merge trend
- `posthog-pp-cli engineering-analytics workflow-health` — Per-workflow CI health over a window (default last 24 hours, maximum 366 days): run count, success rate
- `posthog-pp-cli engineering-analytics workflow-jobs` — Jobs of a single workflow run attempt, with per-job duration, runner tier, and estimated cost.
- `posthog-pp-cli engineering-analytics workflow-run` — A single workflow run: status, conclusion, duration, branch, attempt, and the attributed pull request.
- `posthog-pp-cli engineering-analytics workflow-run-activity` — Compact per-run points for a single workflow over a window (date_from default -30d), newest first
- `posthog-pp-cli engineering-analytics workflow-runner-costs` — A workflow's estimated CI cost broken down by runner tier over a window (date_from default -30d), highest spend first.
- `posthog-pp-cli engineering-analytics workflow-runs` — Runs of a single workflow within a repo over a window (date_from default -30d), newest first.

**environments** — Manage environments

- `posthog-pp-cli environments create` — Deprecated: use /api/environments/{id}/ instead.
- `posthog-pp-cli environments destroy` — Deprecated: use /api/environments/{id}/ instead.
- `posthog-pp-cli environments list` — Deprecated: use /api/environments/{id}/ instead.
- `posthog-pp-cli environments partial-update` — Deprecated: use /api/environments/{id}/ instead.
- `posthog-pp-cli environments retrieve` — Deprecated: use /api/environments/{id}/ instead.
- `posthog-pp-cli environments update` — Deprecated: use /api/environments/{id}/ instead.

**error-tracking** — Manage error tracking

- `posthog-pp-cli error-tracking assignment-rules-create` — Assignment rules create
- `posthog-pp-cli error-tracking assignment-rules-destroy` — Assignment rules destroy
- `posthog-pp-cli error-tracking assignment-rules-list` — Assignment rules list
- `posthog-pp-cli error-tracking assignment-rules-partial-update` — Assignment rules partial update
- `posthog-pp-cli error-tracking assignment-rules-reorder-partial-update` — Assignment rules reorder partial update
- `posthog-pp-cli error-tracking assignment-rules-retrieve` — Assignment rules retrieve
- `posthog-pp-cli error-tracking assignment-rules-update` — Assignment rules update
- `posthog-pp-cli error-tracking bypass-rules-create` — Bypass rules create
- `posthog-pp-cli error-tracking bypass-rules-destroy` — Bypass rules destroy
- `posthog-pp-cli error-tracking bypass-rules-list` — Bypass rules list
- `posthog-pp-cli error-tracking bypass-rules-partial-update` — Bypass rules partial update
- `posthog-pp-cli error-tracking bypass-rules-reorder-partial-update` — Bypass rules reorder partial update
- `posthog-pp-cli error-tracking bypass-rules-retrieve` — Bypass rules retrieve
- `posthog-pp-cli error-tracking bypass-rules-update` — Bypass rules update
- `posthog-pp-cli error-tracking external-references-create` — External references create
- `posthog-pp-cli error-tracking external-references-destroy` — Hard delete of this model is not allowed. Use a patch API call to set 'deleted' to true
- `posthog-pp-cli error-tracking external-references-list` — External references list
- `posthog-pp-cli error-tracking external-references-retrieve` — External references retrieve
- `posthog-pp-cli error-tracking fingerprints-destroy` — Hard delete of this model is not allowed. Use a patch API call to set 'deleted' to true
- `posthog-pp-cli error-tracking fingerprints-list` — Fingerprints list
- `posthog-pp-cli error-tracking fingerprints-resolve-retrieve` — Fingerprints resolve retrieve
- `posthog-pp-cli error-tracking fingerprints-retrieve` — Fingerprints retrieve
- `posthog-pp-cli error-tracking git-provider-file-links-resolve-github-retrieve` — Git provider file links resolve github retrieve
- `posthog-pp-cli error-tracking git-provider-file-links-resolve-gitlab-retrieve` — Git provider file links resolve gitlab retrieve
- `posthog-pp-cli error-tracking grouping-rules-create` — Grouping rules create
- `posthog-pp-cli error-tracking grouping-rules-destroy` — Grouping rules destroy
- `posthog-pp-cli error-tracking grouping-rules-list` — Grouping rules list
- `posthog-pp-cli error-tracking grouping-rules-partial-update` — Grouping rules partial update
- `posthog-pp-cli error-tracking grouping-rules-reorder-partial-update` — Grouping rules reorder partial update
- `posthog-pp-cli error-tracking grouping-rules-retrieve` — Grouping rules retrieve
- `posthog-pp-cli error-tracking grouping-rules-update` — Grouping rules update
- `posthog-pp-cli error-tracking issues-activity-retrieve` — Issues activity retrieve
- `posthog-pp-cli error-tracking issues-all-activity-retrieve` — Issues all activity retrieve
- `posthog-pp-cli error-tracking issues-assign-partial-update` — Issues assign partial update
- `posthog-pp-cli error-tracking issues-bulk-create` — Issues bulk create
- `posthog-pp-cli error-tracking issues-cohort-update` — Issues cohort update
- `posthog-pp-cli error-tracking issues-destroy` — Hard delete of this model is not allowed. Use a patch API call to set 'deleted' to true
- `posthog-pp-cli error-tracking issues-exists-retrieve` — Issues exists retrieve
- `posthog-pp-cli error-tracking issues-list` — Issues list
- `posthog-pp-cli error-tracking issues-merge-create` — Issues merge create
- `posthog-pp-cli error-tracking issues-partial-update` — Issues partial update
- `posthog-pp-cli error-tracking issues-retrieve` — Issues retrieve
- `posthog-pp-cli error-tracking issues-split-create` — Issues split create
- `posthog-pp-cli error-tracking issues-update` — Issues update
- `posthog-pp-cli error-tracking issues-values-retrieve` — Issues values retrieve
- `posthog-pp-cli error-tracking query-issue-create` — Fetch one error tracking issue with impact counts, top in_app frame, latest release, and optional sparkline.
- `posthog-pp-cli error-tracking query-issue-events-create` — Fetch sampled exception events, stack traces, browser/SDK context, URL, and $session_id values for one issue.
- `posthog-pp-cli error-tracking query-issues-list-create` — List error tracking issues with typed filters and compact aggregate counts.
- `posthog-pp-cli error-tracking recommendations-dismiss-create` — Recommendations dismiss create
- `posthog-pp-cli error-tracking recommendations-list` — Recommendations list
- `posthog-pp-cli error-tracking recommendations-refresh-create` — Recommendations refresh create
- `posthog-pp-cli error-tracking recommendations-restore-create` — Recommendations restore create
- `posthog-pp-cli error-tracking releases-create` — Releases create
- `posthog-pp-cli error-tracking releases-destroy` — Releases destroy
- `posthog-pp-cli error-tracking releases-hash-retrieve` — Releases hash retrieve
- `posthog-pp-cli error-tracking releases-list` — Releases list
- `posthog-pp-cli error-tracking releases-partial-update` — Releases partial update
- `posthog-pp-cli error-tracking releases-retrieve` — Releases retrieve
- `posthog-pp-cli error-tracking releases-update` — Releases update
- `posthog-pp-cli error-tracking settings-retrieve-settings-retrieve` — Settings retrieve settings retrieve
- `posthog-pp-cli error-tracking settings-update-settings-partial-update` — Settings update settings partial update
- `posthog-pp-cli error-tracking spike-detection-config-list` — Spike detection config list
- `posthog-pp-cli error-tracking spike-detection-config-update-config-partial-update` — Spike detection config update config partial update
- `posthog-pp-cli error-tracking spike-events-list` — Spike events list
- `posthog-pp-cli error-tracking stack-frames-batch-get-create` — Stack frames batch get create
- `posthog-pp-cli error-tracking stack-frames-destroy` — Hard delete of this model is not allowed. Use a patch API call to set 'deleted' to true
- `posthog-pp-cli error-tracking stack-frames-list` — Stack frames list
- `posthog-pp-cli error-tracking stack-frames-retrieve` — Stack frames retrieve
- `posthog-pp-cli error-tracking suppression-rules-create` — Suppression rules create
- `posthog-pp-cli error-tracking suppression-rules-destroy` — Suppression rules destroy
- `posthog-pp-cli error-tracking suppression-rules-list` — Suppression rules list
- `posthog-pp-cli error-tracking suppression-rules-partial-update` — Suppression rules partial update
- `posthog-pp-cli error-tracking suppression-rules-reorder-partial-update` — Suppression rules reorder partial update
- `posthog-pp-cli error-tracking suppression-rules-retrieve` — Suppression rules retrieve
- `posthog-pp-cli error-tracking suppression-rules-update` — Suppression rules update
- `posthog-pp-cli error-tracking symbol-sets-bulk-delete-create` — Symbol sets bulk delete create
- `posthog-pp-cli error-tracking symbol-sets-bulk-finish-upload-create` — Symbol sets bulk finish upload create
- `posthog-pp-cli error-tracking symbol-sets-bulk-start-upload-create` — Symbol sets bulk start upload create
- `posthog-pp-cli error-tracking symbol-sets-destroy` — Symbol sets destroy
- `posthog-pp-cli error-tracking symbol-sets-download-retrieve` — Return a presigned URL for downloading the symbol set's source map.
- `posthog-pp-cli error-tracking symbol-sets-finish-upload-update` — Symbol sets finish upload update
- `posthog-pp-cli error-tracking symbol-sets-list` — Symbol sets list
- `posthog-pp-cli error-tracking symbol-sets-retrieve` — Symbol sets retrieve

**evaluation-directories** — Manage evaluation directories

- `posthog-pp-cli evaluation-directories create` — Create
- `posthog-pp-cli evaluation-directories destroy` — Destroy
- `posthog-pp-cli evaluation-directories list` — List
- `posthog-pp-cli evaluation-directories partial-update` — Partial update
- `posthog-pp-cli evaluation-directories retrieve` — Retrieve

**evaluation-runs** — Manage evaluation runs

- `posthog-pp-cli evaluation-runs <project_id>` — Create a new evaluation run.

**evaluations** — Manage evaluations

- `posthog-pp-cli evaluations create` — Create
- `posthog-pp-cli evaluations destroy` — Hard delete of this model is not allowed. Use a patch API call to set 'deleted' to true
- `posthog-pp-cli evaluations list` — List
- `posthog-pp-cli evaluations partial-update` — Partial update
- `posthog-pp-cli evaluations retrieve` — Retrieve
- `posthog-pp-cli evaluations test-hog-create` — Test Hog evaluation code against sample events without saving.
- `posthog-pp-cli evaluations update` — Update

**event-definitions** — Manage event definitions

- `posthog-pp-cli event-definitions bulk-update-tags-create` — Add, remove, or replace tags across multiple event definitions in one request. Overrides ``TaggedItemViewSetMixin.
- `posthog-pp-cli event-definitions bulk-update-verified-create` — Mark multiple event definitions as verified or unverified in one request.
- `posthog-pp-cli event-definitions by-name-retrieve` — Get event definition by exact name
- `posthog-pp-cli event-definitions create` — Create
- `posthog-pp-cli event-definitions destroy` — Destroy
- `posthog-pp-cli event-definitions golang-retrieve` — Golang retrieve
- `posthog-pp-cli event-definitions list` — List
- `posthog-pp-cli event-definitions partial-update` — Partial update
- `posthog-pp-cli event-definitions primary-properties-retrieve` — Resolve team-configured primary properties for event definitions.
- `posthog-pp-cli event-definitions python-retrieve` — Python retrieve
- `posthog-pp-cli event-definitions retrieve` — Retrieve
- `posthog-pp-cli event-definitions typescript-retrieve` — Typescript retrieve
- `posthog-pp-cli event-definitions update` — Update

**event-filter** — Manage event filter

- `posthog-pp-cli event-filter create` — Create or update the event filter config.
- `posthog-pp-cli event-filter metrics-retrieve` — Single event filter per team.
- `posthog-pp-cli event-filter metrics-totals-retrieve` — Single event filter per team.
- `posthog-pp-cli event-filter retrieve` — Returns the event filter config for the team, or null if not yet created.

**event-schemas** — Manage event schemas

- `posthog-pp-cli event-schemas create` — Create
- `posthog-pp-cli event-schemas destroy` — Destroy
- `posthog-pp-cli event-schemas list` — List
- `posthog-pp-cli event-schemas partial-update` — Partial update
- `posthog-pp-cli event-schemas update` — Update

**event-streams** — Manage event streams

- `posthog-pp-cli event-streams create` — The caller's event stream: a live feed of selected accounts' events posted to a Slack channel of their choice.
- `posthog-pp-cli event-streams destroy` — The caller's event stream: a live feed of selected accounts' events posted to a Slack channel of their choice.
- `posthog-pp-cli event-streams list` — The caller's event stream: a live feed of selected accounts' events posted to a Slack channel of their choice.
- `posthog-pp-cli event-streams partial-update` — The caller's event stream: a live feed of selected accounts' events posted to a Slack channel of their choice.
- `posthog-pp-cli event-streams update` — The caller's event stream: a live feed of selected accounts' events posted to a Slack channel of their choice.

**events** — Manage events

- `posthog-pp-cli events list` — This endpoint allows you to list and filter events.
- `posthog-pp-cli events retrieve` — Retrieve
- `posthog-pp-cli events values-retrieve` — Values retrieve

**experiment-holdouts** — Manage experiment holdouts

- `posthog-pp-cli experiment-holdouts create` — Create
- `posthog-pp-cli experiment-holdouts destroy` — Destroy
- `posthog-pp-cli experiment-holdouts list` — List
- `posthog-pp-cli experiment-holdouts partial-update` — Partial update
- `posthog-pp-cli experiment-holdouts retrieve` — Retrieve
- `posthog-pp-cli experiment-holdouts update` — Update

**experiment-saved-metrics** — Manage experiment saved metrics

- `posthog-pp-cli experiment-saved-metrics create` — Create
- `posthog-pp-cli experiment-saved-metrics destroy` — Destroy
- `posthog-pp-cli experiment-saved-metrics list` — List
- `posthog-pp-cli experiment-saved-metrics partial-update` — Partial update
- `posthog-pp-cli experiment-saved-metrics retrieve` — Retrieve
- `posthog-pp-cli experiment-saved-metrics update` — Update

**experiments** — Manage experiments

- `posthog-pp-cli experiments calculate-running-time-create` — Estimate the recommended sample size and running time for an experiment.
- `posthog-pp-cli experiments create` — Create a new experiment in draft status with optional metrics.
- `posthog-pp-cli experiments create-from-prompt-create` — Create an experiment that compares N versions of an LLM prompt using a metric template.
- `posthog-pp-cli experiments destroy` — Hard delete of this model is not allowed. Use a patch API call to set 'deleted' to true
- `posthog-pp-cli experiments list` — List experiments for the current project. Supports filtering by status and archival state.
- `posthog-pp-cli experiments partial-update` — Update an experiment.
- `posthog-pp-cli experiments prompt-templates-retrieve` — List the LLM metric templates that can be passed to `create_from_prompt`.
- `posthog-pp-cli experiments retrieve` — Retrieve a single experiment by ID, including its current status, metrics, feature flag, and results metadata.
- `posthog-pp-cli experiments session-context-retrieve` — Resolve which experiments (and variants) a session recording saw.
- `posthog-pp-cli experiments session-contexts-create` — Resolve experiment context for a batch of session recordings.
- `posthog-pp-cli experiments stats-retrieve` — Mixin for ViewSets to handle approval-gate exceptions raised from decorated serializers.
- `posthog-pp-cli experiments update` — Mixin for ViewSets to handle approval-gate exceptions raised from decorated serializers.

**exports** — Manage exports

- `posthog-pp-cli exports create` — Create
- `posthog-pp-cli exports list` — List
- `posthog-pp-cli exports retrieve` — Retrieve

**external-data-schemas** — Manage external data schemas

- `posthog-pp-cli external-data-schemas destroy` — Destroy
- `posthog-pp-cli external-data-schemas list` — List
- `posthog-pp-cli external-data-schemas partial-update` — Partial update
- `posthog-pp-cli external-data-schemas retrieve` — Retrieve
- `posthog-pp-cli external-data-schemas update` — Update

**external-data-sources** — Manage external data sources

- `posthog-pp-cli external-data-sources check-cdc-prerequisites-create` — Validate CDC prerequisites against a live Postgres connection.
- `posthog-pp-cli external-data-sources connect-link-retrieve` — Return a secure browser link for connecting a data warehouse source.
- `posthog-pp-cli external-data-sources connections-list` — Create, Read, Update and Delete External data Sources.
- `posthog-pp-cli external-data-sources create` — Create, Read, Update and Delete External data Sources.
- `posthog-pp-cli external-data-sources database-schema-create` — Create, Read, Update and Delete External data Sources.
- `posthog-pp-cli external-data-sources destroy` — Create, Read, Update and Delete External data Sources.
- `posthog-pp-cli external-data-sources direct-connection-options-list` — Source types the user can add as a direct connection
- `posthog-pp-cli external-data-sources draft-custom-manifest-create` — Draft a Custom REST source manifest from API documentation using an LLM.
- `posthog-pp-cli external-data-sources list` — Create, Read, Update and Delete External data Sources.
- `posthog-pp-cli external-data-sources oauth-accounts-retrieve` — List the accounts/properties a connected OAuth integration exposes, in the shared IntegrationAccount shape.
- `posthog-pp-cli external-data-sources partial-update` — Create, Read, Update and Delete External data Sources.
- `posthog-pp-cli external-data-sources preview-resource-create` — Read a bounded sample of rows for one resource of a Custom REST source.
- `posthog-pp-cli external-data-sources retrieve` — Create, Read, Update and Delete External data Sources.
- `posthog-pp-cli external-data-sources setup-create` — One-shot data warehouse source setup.
- `posthog-pp-cli external-data-sources source-prefix-create` — Create, Read, Update and Delete External data Sources.
- `posthog-pp-cli external-data-sources store-credentials-create` — Validate and store credentials for a data warehouse source without creating the source.
- `posthog-pp-cli external-data-sources stored-credentials-list` — List credentials the requesting user stored via the source connect page that haven't been consumed yet.
- `posthog-pp-cli external-data-sources update` — Create, Read, Update and Delete External data Sources.
- `posthog-pp-cli external-data-sources wizard-retrieve` — Create, Read, Update and Delete External data Sources.

**feature-flags** — Manage feature flags

- `posthog-pp-cli feature-flags all-activity-retrieve` — Create, read, update and delete feature flags. [See docs](https://posthog.
- `posthog-pp-cli feature-flags bulk-delete-create` — Bulk delete feature flags by filter criteria or explicit IDs. Accepts either: - {'filters': {...
- `posthog-pp-cli feature-flags bulk-keys-retrieve` — Get feature flag keys by IDs. Accepts a list of feature flag IDs and returns a mapping of ID to key.
- `posthog-pp-cli feature-flags bulk-update-tags-create` — Bulk update tags on multiple objects.
- `posthog-pp-cli feature-flags create` — Create, read, update and delete feature flags. [See docs](https://posthog.
- `posthog-pp-cli feature-flags destroy` — Hard delete of this model is not allowed. Use a patch API call to set 'deleted' to true
- `posthog-pp-cli feature-flags evaluation-reasons-retrieve` — Create, read, update and delete feature flags. [See docs](https://posthog.
- `posthog-pp-cli feature-flags list` — Create, read, update and delete feature flags. [See docs](https://posthog.
- `posthog-pp-cli feature-flags matching-ids-retrieve` — Get IDs of all feature flags matching the current filters. Uses the same filtering logic as the list endpoint.
- `posthog-pp-cli feature-flags my-flags-retrieve` — Create, read, update and delete feature flags. [See docs](https://posthog.
- `posthog-pp-cli feature-flags partial-update` — Create, read, update and delete feature flags. [See docs](https://posthog.
- `posthog-pp-cli feature-flags retrieve` — Create, read, update and delete feature flags. [See docs](https://posthog.
- `posthog-pp-cli feature-flags update` — Create, read, update and delete feature flags. [See docs](https://posthog.
- `posthog-pp-cli feature-flags user-blast-radius-create` — Create, read, update and delete feature flags. [See docs](https://posthog.

**field-notes** — Manage field notes

- `posthog-pp-cli field-notes create` — Create, read, update, and resolve toolbar field notes — UI feedback a user points at on their own site
- `posthog-pp-cli field-notes destroy` — Create, read, update, and resolve toolbar field notes — UI feedback a user points at on their own site
- `posthog-pp-cli field-notes list` — Create, read, update, and resolve toolbar field notes — UI feedback a user points at on their own site
- `posthog-pp-cli field-notes partial-update` — Create, read, update, and resolve toolbar field notes — UI feedback a user points at on their own site
- `posthog-pp-cli field-notes retrieve` — Create, read, update, and resolve toolbar field notes — UI feedback a user points at on their own site
- `posthog-pp-cli field-notes update` — Create, read, update, and resolve toolbar field notes — UI feedback a user points at on their own site

**file-download-batch-exports** — Manage file download batch exports

- `posthog-pp-cli file-download-batch-exports create` — Create and start a batch export on demand run to download a file.
- `posthog-pp-cli file-download-batch-exports list` — List
- `posthog-pp-cli file-download-batch-exports retrieve` — Get a batch export on demand run.

**file-system** — Manage file system

- `posthog-pp-cli file-system count-by-path-create` — Get count of all files in a folder.
- `posthog-pp-cli file-system create` — Create
- `posthog-pp-cli file-system destroy` — Destroy
- `posthog-pp-cli file-system list` — List
- `posthog-pp-cli file-system log-view-create` — Log view create
- `posthog-pp-cli file-system log-view-retrieve` — Log view retrieve
- `posthog-pp-cli file-system partial-update` — Partial update
- `posthog-pp-cli file-system retrieve` — Retrieve
- `posthog-pp-cli file-system undo-delete-create` — Undo delete create
- `posthog-pp-cli file-system unfiled-retrieve` — Unfiled retrieve
- `posthog-pp-cli file-system update` — Update

**file-system-shortcut** — Manage file system shortcut

- `posthog-pp-cli file-system-shortcut create` — Create
- `posthog-pp-cli file-system-shortcut destroy` — Destroy
- `posthog-pp-cli file-system-shortcut list` — List
- `posthog-pp-cli file-system-shortcut partial-update` — Partial update
- `posthog-pp-cli file-system-shortcut reorder-create` — Set the display order of the current user's shortcuts.
- `posthog-pp-cli file-system-shortcut retrieve` — Retrieve
- `posthog-pp-cli file-system-shortcut update` — Update

**flag-value** — Manage flag value

- `posthog-pp-cli flag-value <project_id>` — Get possible values for a feature flag.

**groups** — Manage groups

- `posthog-pp-cli groups activity-retrieve` — Activity retrieve
- `posthog-pp-cli groups create` — Create
- `posthog-pp-cli groups delete-property-create` — Delete property create
- `posthog-pp-cli groups find-retrieve` — Find retrieve
- `posthog-pp-cli groups list` — List all groups of a specific group type. You must pass ?group_type_index= in the URL.
- `posthog-pp-cli groups property-values-retrieve` — Property values retrieve
- `posthog-pp-cli groups related-retrieve` — Related retrieve
- `posthog-pp-cli groups update-property-create` — Update property create

**groups-types** — Manage groups types

- `posthog-pp-cli groups-types create-detail-dashboard-update` — Create detail dashboard update
- `posthog-pp-cli groups-types destroy` — Destroy
- `posthog-pp-cli groups-types list` — List
- `posthog-pp-cli groups-types set-default-columns-update` — Set default columns update
- `posthog-pp-cli groups-types update-metadata-partial-update` — Update metadata partial update

**health-issues** — Manage health issues

- `posthog-pp-cli health-issues list` — Lists health issues detected across all of this project's PostHog health checks (outdated SDKs
- `posthog-pp-cli health-issues partial-update` — Partial update
- `posthog-pp-cli health-issues refresh-create` — Refresh create
- `posthog-pp-cli health-issues retrieve` — Fetches a single health issue, enriched with the owning check's rendered explanation: a title
- `posthog-pp-cli health-issues summary-retrieve` — Returns aggregated counts of active, non-dismissed health issues for the project, broken down by severity and by kind.

**heatmap-screenshots** — Manage heatmap screenshots


**heatmaps** — Manage heatmaps

- `posthog-pp-cli heatmaps events-retrieve` — Drill into the individual session interactions behind one or more heatmap coordinates.
- `posthog-pp-cli heatmaps list` — Aggregated heatmap interactions for a page.

**hog-flows** — Manage hog flows

- `posthog-pp-cli hog-flows bulk-delete-create` — Bulk delete create
- `posthog-pp-cli hog-flows create` — Create
- `posthog-pp-cli hog-flows destroy` — Destroy
- `posthog-pp-cli hog-flows email-sending-suspension-retrieve` — Cheap read for the scene-wide suspension banner: single-row `TeamWorkflowsConfig` lookup with no reputation computation.
- `posthog-pp-cli hog-flows list` — List
- `posthog-pp-cli hog-flows metrics-global-retrieve` — Metrics global retrieve
- `posthog-pp-cli hog-flows partial-update` — Partial update
- `posthog-pp-cli hog-flows reputation-retrieve` — Bounce/complaint rates for this project's workflow email over the last 30 days
- `posthog-pp-cli hog-flows retrieve` — Retrieve
- `posthog-pp-cli hog-flows update` — Update
- `posthog-pp-cli hog-flows user-blast-radius-create` — User blast radius create

**hog-function-templates** — Manage hog function templates

- `posthog-pp-cli hog-function-templates list` — List
- `posthog-pp-cli hog-function-templates retrieve` — Retrieve

**hog-functions** — Manage hog functions

- `posthog-pp-cli hog-functions create` — Create
- `posthog-pp-cli hog-functions destroy` — Hard delete of this model is not allowed. Use a patch API call to set 'deleted' to true
- `posthog-pp-cli hog-functions icon-retrieve` — Icon retrieve
- `posthog-pp-cli hog-functions icons-retrieve` — Icons retrieve
- `posthog-pp-cli hog-functions list` — List
- `posthog-pp-cli hog-functions partial-update` — Partial update
- `posthog-pp-cli hog-functions rearrange-partial-update` — Update the execution order of multiple HogFunctions.
- `posthog-pp-cli hog-functions retrieve` — Retrieve
- `posthog-pp-cli hog-functions update` — Update

**ingestion-warnings-v2** — Manage ingestion warnings v2

- `posthog-pp-cli ingestion-warnings-v2 <project_id>` — Lists this project's ingestion warnings — events or person/group updates that were ingested with problems (oversized

**insight-variables** — Manage insight variables

- `posthog-pp-cli insight-variables create` — Create
- `posthog-pp-cli insight-variables destroy` — Destroy
- `posthog-pp-cli insight-variables list` — List
- `posthog-pp-cli insight-variables partial-update` — Partial update
- `posthog-pp-cli insight-variables retrieve` — Retrieve
- `posthog-pp-cli insight-variables update` — Update

**insights** — Manage insights

- `posthog-pp-cli insights all-activity-retrieve` — Project-wide audit trail across all insights — who created, edited, deleted, or restored insights
- `posthog-pp-cli insights bulk-delete-create` — Soft-delete insights in bulk by ID.
- `posthog-pp-cli insights bulk-restore-create` — Restore soft-deleted insights in bulk by ID — the inverse of bulk_delete.
- `posthog-pp-cli insights bulk-update-tags-create` — Bulk update tags on multiple objects.
- `posthog-pp-cli insights cancel-create` — DRF ViewSet mixin that gates coalesced responses behind permission checks.
- `posthog-pp-cli insights create` — DRF ViewSet mixin that gates coalesced responses behind permission checks.
- `posthog-pp-cli insights destroy` — Hard delete of this model is not allowed. Use a patch API call to set 'deleted' to true
- `posthog-pp-cli insights generate-metadata-create` — Generate an AI-suggested name and description for an insight based on its query configuration.
- `posthog-pp-cli insights list` — DRF ViewSet mixin that gates coalesced responses behind permission checks.
- `posthog-pp-cli insights my-last-viewed-retrieve` — Returns basic details about the last 5 insights viewed by this user. Most recently viewed first.
- `posthog-pp-cli insights partial-update` — DRF ViewSet mixin that gates coalesced responses behind permission checks.
- `posthog-pp-cli insights retrieve` — DRF ViewSet mixin that gates coalesced responses behind permission checks.
- `posthog-pp-cli insights trending-retrieve` — Returns insights ranked by view count over the last N days (default 7), highest first.
- `posthog-pp-cli insights update` — DRF ViewSet mixin that gates coalesced responses behind permission checks.
- `posthog-pp-cli insights viewed-create` — Record that the current user has just viewed one or more insights.

**integrations** — Manage integrations

- `posthog-pp-cli integrations authorize-retrieve` — Authorize retrieve
- `posthog-pp-cli integrations create` — Create
- `posthog-pp-cli integrations destroy` — Destroy
- `posthog-pp-cli integrations domain-connect-apply-url-create` — Unified endpoint for generating Domain Connect apply URLs.
- `posthog-pp-cli integrations domain-connect-check-retrieve` — Domain connect check retrieve
- `posthog-pp-cli integrations github-available-installations-retrieve` — List the org's existing GitHub installations this project can reuse.
- `posthog-pp-cli integrations github-link-existing-create` — Reuse a GitHub installation already linked to a sibling team in the same organization.
- `posthog-pp-cli integrations github-oauth-authorize-create` — Mint a User OAuth URL to bootstrap a fresh `code` when the install flow returns without one.
- `posthog-pp-cli integrations github-prepare-callback-create` — Seed GitHub setup callback state without redirecting to GitHub.
- `posthog-pp-cli integrations list` — List
- `posthog-pp-cli integrations request-access-create` — Notify project admins that a member is requesting an integration be connected.
- `posthog-pp-cli integrations retrieve` — Retrieve

**js-snippet** — Manage js snippet

- `posthog-pp-cli js-snippet resolve-retrieve` — Preview what a given pin would resolve to, without saving it.
- `posthog-pp-cli js-snippet version-partial-update` — Update the team's version pin.
- `posthog-pp-cli js-snippet version-retrieve` — Return the team's current version pin and resolved version.

**live-debugger-breakpoints** — Manage live debugger breakpoints

- `posthog-pp-cli live-debugger-breakpoints active-retrieve` — External API endpoint for client applications to fetch active breakpoints using Project API key.
- `posthog-pp-cli live-debugger-breakpoints breakpoint-hits-retrieve` — Retrieve breakpoint hit events from ClickHouse with optional filtering and pagination.
- `posthog-pp-cli live-debugger-breakpoints create` — Create, Read, Update and Delete breakpoints for live debugging.
- `posthog-pp-cli live-debugger-breakpoints destroy` — Create, Read, Update and Delete breakpoints for live debugging.
- `posthog-pp-cli live-debugger-breakpoints list` — Create, Read, Update and Delete breakpoints for live debugging.
- `posthog-pp-cli live-debugger-breakpoints partial-update` — Create, Read, Update and Delete breakpoints for live debugging.
- `posthog-pp-cli live-debugger-breakpoints retrieve` — Create, Read, Update and Delete breakpoints for live debugging.
- `posthog-pp-cli live-debugger-breakpoints update` — Create, Read, Update and Delete breakpoints for live debugging.

**llm-analytics** — Manage llm analytics

- `posthog-pp-cli llm-analytics clustering-config-list` — Team-level clustering configuration (event filters for automated pipelines).
- `posthog-pp-cli llm-analytics clustering-config-set-event-filters-create` — Team-level clustering configuration (event filters for automated pipelines).
- `posthog-pp-cli llm-analytics clustering-jobs-create` — CRUD for clustering job configurations (max 10 per team).
- `posthog-pp-cli llm-analytics clustering-jobs-destroy` — CRUD for clustering job configurations (max 10 per team).
- `posthog-pp-cli llm-analytics clustering-jobs-list` — CRUD for clustering job configurations (max 10 per team).
- `posthog-pp-cli llm-analytics clustering-jobs-partial-update` — CRUD for clustering job configurations (max 10 per team).
- `posthog-pp-cli llm-analytics clustering-jobs-retrieve` — CRUD for clustering job configurations (max 10 per team).
- `posthog-pp-cli llm-analytics clustering-jobs-update` — CRUD for clustering job configurations (max 10 per team).
- `posthog-pp-cli llm-analytics clustering-runs-create` — Trigger a new clustering workflow run.
- `posthog-pp-cli llm-analytics evaluation-config-retrieve` — Get the evaluation config for this team
- `posthog-pp-cli llm-analytics evaluation-config-set-active-key-create` — Set the active provider key for evaluations
- `posthog-pp-cli llm-analytics evaluation-reports-create` — CRUD for evaluation report configurations + report run history.
- `posthog-pp-cli llm-analytics evaluation-reports-destroy` — Evaluation report configs are deleted only when their evaluation is deleted. Use PATCH enabled=false to stop delivery.
- `posthog-pp-cli llm-analytics evaluation-reports-generate-create` — Trigger immediate report generation.
- `posthog-pp-cli llm-analytics evaluation-reports-list` — CRUD for evaluation report configurations + report run history.
- `posthog-pp-cli llm-analytics evaluation-reports-partial-update` — CRUD for evaluation report configurations + report run history.
- `posthog-pp-cli llm-analytics evaluation-reports-retrieve` — CRUD for evaluation report configurations + report run history.
- `posthog-pp-cli llm-analytics evaluation-reports-runs-list` — List report runs (history) for this report.
- `posthog-pp-cli llm-analytics evaluation-reports-update` — CRUD for evaluation report configurations + report run history.
- `posthog-pp-cli llm-analytics evaluation-summary-create` — Generate an AI-powered summary of evaluation results.
- `posthog-pp-cli llm-analytics models-retrieve` — List available models for a provider.
- `posthog-pp-cli llm-analytics offline-evaluations-experiment-items-create` — Offline evaluations experiment items create
- `posthog-pp-cli llm-analytics parser-recipes-create` — Parser recipes create
- `posthog-pp-cli llm-analytics parser-recipes-destroy` — Parser recipes destroy
- `posthog-pp-cli llm-analytics parser-recipes-list` — Parser recipes list
- `posthog-pp-cli llm-analytics parser-recipes-partial-update` — Parser recipes partial update
- `posthog-pp-cli llm-analytics parser-recipes-retrieve` — Parser recipes retrieve
- `posthog-pp-cli llm-analytics personal-spend-list` — Return a structured personal LLM spend analysis for the requesting user.
- `posthog-pp-cli llm-analytics provider-key-validations-create` — Validate LLM provider API keys without persisting them
- `posthog-pp-cli llm-analytics provider-keys-create` — Provider keys create
- `posthog-pp-cli llm-analytics provider-keys-dependent-configs-retrieve` — Get evaluations using this key and alternative keys for replacement.
- `posthog-pp-cli llm-analytics provider-keys-destroy` — Provider keys destroy
- `posthog-pp-cli llm-analytics provider-keys-list` — Provider keys list
- `posthog-pp-cli llm-analytics provider-keys-partial-update` — Provider keys partial update
- `posthog-pp-cli llm-analytics provider-keys-retrieve` — Provider keys retrieve
- `posthog-pp-cli llm-analytics provider-keys-update` — Provider keys update
- `posthog-pp-cli llm-analytics provider-keys-validate-create` — Provider keys validate create
- `posthog-pp-cli llm-analytics review-queue-items-create` — Review queue items create
- `posthog-pp-cli llm-analytics review-queue-items-destroy` — Review queue items destroy
- `posthog-pp-cli llm-analytics review-queue-items-list` — Review queue items list
- `posthog-pp-cli llm-analytics review-queue-items-partial-update` — Review queue items partial update
- `posthog-pp-cli llm-analytics review-queue-items-retrieve` — Review queue items retrieve
- `posthog-pp-cli llm-analytics review-queues-create` — Review queues create
- `posthog-pp-cli llm-analytics review-queues-destroy` — Review queues destroy
- `posthog-pp-cli llm-analytics review-queues-list` — Review queues list
- `posthog-pp-cli llm-analytics review-queues-partial-update` — Review queues partial update
- `posthog-pp-cli llm-analytics review-queues-retrieve` — Review queues retrieve
- `posthog-pp-cli llm-analytics score-definitions-create` — Score definitions create
- `posthog-pp-cli llm-analytics score-definitions-list` — Score definitions list
- `posthog-pp-cli llm-analytics score-definitions-new-version-create` — Score definitions new version create
- `posthog-pp-cli llm-analytics score-definitions-partial-update` — Score definitions partial update
- `posthog-pp-cli llm-analytics score-definitions-retrieve` — Score definitions retrieve
- `posthog-pp-cli llm-analytics summarization-batch-check-create` — Check which traces have cached summaries available.
- `posthog-pp-cli llm-analytics summarization-create` — Generate an AI-powered summary of an LLM trace or event.
- `posthog-pp-cli llm-analytics text-repr-create` — Generate a human-readable text representation of an LLM trace event.
- `posthog-pp-cli llm-analytics trace-reviews-create` — Trace reviews create
- `posthog-pp-cli llm-analytics trace-reviews-destroy` — Trace reviews destroy
- `posthog-pp-cli llm-analytics trace-reviews-list` — Trace reviews list
- `posthog-pp-cli llm-analytics trace-reviews-partial-update` — Trace reviews partial update
- `posthog-pp-cli llm-analytics trace-reviews-retrieve` — Trace reviews retrieve
- `posthog-pp-cli llm-analytics translate-create` — Translate text to target language.

**llm-prompts** — Manage llm prompts

- `posthog-pp-cli llm-prompts create` — Create
- `posthog-pp-cli llm-prompts list` — List
- `posthog-pp-cli llm-prompts name-archive-create` — Name archive create
- `posthog-pp-cli llm-prompts name-duplicate-create` — Name duplicate create
- `posthog-pp-cli llm-prompts name-labels-destroy` — Name labels destroy
- `posthog-pp-cli llm-prompts name-labels-update` — Name labels update
- `posthog-pp-cli llm-prompts name-partial-update` — Name partial update
- `posthog-pp-cli llm-prompts name-retrieve` — Name retrieve
- `posthog-pp-cli llm-prompts resolve-name-retrieve` — Resolve name retrieve

**llm-skills** — Manage llm skills

- `posthog-pp-cli llm-skills create` — Create
- `posthog-pp-cli llm-skills import-create` — Import create
- `posthog-pp-cli llm-skills list` — List
- `posthog-pp-cli llm-skills marketplace-install-command-create` — Mint the user's read-only marketplace credential (or rotate it) and return the install command.
- `posthog-pp-cli llm-skills marketplace-install-command-retrieve` — Report whether the user already has a marketplace credential, without minting one.
- `posthog-pp-cli llm-skills name-archive-create` — Name archive create
- `posthog-pp-cli llm-skills name-duplicate-create` — Name duplicate create
- `posthog-pp-cli llm-skills name-export-retrieve` — Name export retrieve
- `posthog-pp-cli llm-skills name-files-create` — Name files create
- `posthog-pp-cli llm-skills name-files-destroy` — Name files destroy
- `posthog-pp-cli llm-skills name-files-rename-create` — Name files rename create
- `posthog-pp-cli llm-skills name-files-retrieve` — Name files retrieve
- `posthog-pp-cli llm-skills name-partial-update` — Name partial update
- `posthog-pp-cli llm-skills name-retrieve` — Name retrieve
- `posthog-pp-cli llm-skills resolve-name-retrieve` — Resolve name retrieve

**logs** — Manage logs

- `posthog-pp-cli logs alerts-create` — Alerts create
- `posthog-pp-cli logs alerts-destinations-create` — Create a notification destination for this alert. One HogFunction is created per alert event kind (firing, resolved, ...
- `posthog-pp-cli logs alerts-destinations-delete-create` — Delete a notification destination by deleting its HogFunction group atomically.
- `posthog-pp-cli logs alerts-destroy` — Alerts destroy
- `posthog-pp-cli logs alerts-events-list` — Paginated event history for this alert, newest first.
- `posthog-pp-cli logs alerts-list` — Alerts list
- `posthog-pp-cli logs alerts-partial-update` — Alerts partial update
- `posthog-pp-cli logs alerts-reset-create` — Reset a broken alert. Clears the consecutive-failure counter and schedules an immediate recheck.
- `posthog-pp-cli logs alerts-retrieve` — Alerts retrieve
- `posthog-pp-cli logs alerts-simulate-create` — Simulate a logs alert on historical data using the full state machine. Read-only — no alert check records are created.
- `posthog-pp-cli logs alerts-update` — Alerts update
- `posthog-pp-cli logs anomalies-scan-create` — Runs anomaly detection on demand over one service's log volume for the given window.
- `posthog-pp-cli logs attributes-retrieve` — Attributes retrieve
- `posthog-pp-cli logs count-create` — Count create
- `posthog-pp-cli logs count-ranges-create` — Count ranges create
- `posthog-pp-cli logs explain-with-ai-create` — Explain a log entry using AI. POST /api/environments/:id/logs/explainLogWithAI/
- `posthog-pp-cli logs export-create` — Export create
- `posthog-pp-cli logs facet-values-create` — Facet values create
- `posthog-pp-cli logs group-by-create` — Group by create
- `posthog-pp-cli logs has-retrieve` — Has retrieve
- `posthog-pp-cli logs metric-rules-create` — Metric rules create
- `posthog-pp-cli logs metric-rules-destroy` — Metric rules destroy
- `posthog-pp-cli logs metric-rules-list` — Metric rules list
- `posthog-pp-cli logs metric-rules-partial-update` — Metric rules partial update
- `posthog-pp-cli logs metric-rules-retrieve` — Metric rules retrieve
- `posthog-pp-cli logs metric-rules-update` — Metric rules update
- `posthog-pp-cli logs patterns-create` — Patterns create
- `posthog-pp-cli logs patterns-diff-create` — Patterns diff create
- `posthog-pp-cli logs query-create` — Query create
- `posthog-pp-cli logs retention-rules-create` — Retention rules create
- `posthog-pp-cli logs retention-rules-destroy` — Retention rules destroy
- `posthog-pp-cli logs retention-rules-list` — Retention rules list
- `posthog-pp-cli logs retention-rules-partial-update` — Retention rules partial update
- `posthog-pp-cli logs retention-rules-reorder-create` — Atomically reassign priorities so the given ID order maps to ascending priorities (0..n-1).
- `posthog-pp-cli logs retention-rules-retrieve` — Retention rules retrieve
- `posthog-pp-cli logs retention-rules-suggest-name-create` — Suggest a human-readable name for a retention rule from its retention tier and filter group.
- `posthog-pp-cli logs retention-rules-update` — Retention rules update
- `posthog-pp-cli logs sampling-rules-create` — Sampling rules create
- `posthog-pp-cli logs sampling-rules-destroy` — Sampling rules destroy
- `posthog-pp-cli logs sampling-rules-list` — Sampling rules list
- `posthog-pp-cli logs sampling-rules-partial-update` — Sampling rules partial update
- `posthog-pp-cli logs sampling-rules-reorder-create` — Atomically reassign priorities so the given ID order maps to ascending priorities (0..n-1).
- `posthog-pp-cli logs sampling-rules-retrieve` — Sampling rules retrieve
- `posthog-pp-cli logs sampling-rules-simulate-create` — Dry-run estimate for how much volume this rule would remove (placeholder response until CH-backed simulation is wired).
- `posthog-pp-cli logs sampling-rules-update` — Sampling rules update
- `posthog-pp-cli logs services-create` — Services create
- `posthog-pp-cli logs sparkline-create` — Sparkline create
- `posthog-pp-cli logs values-retrieve` — Values retrieve
- `posthog-pp-cli logs views-create` — Views create
- `posthog-pp-cli logs views-destroy` — Views destroy
- `posthog-pp-cli logs views-list` — Views list
- `posthog-pp-cli logs views-partial-update` — Views partial update
- `posthog-pp-cli logs views-retrieve` — Views retrieve
- `posthog-pp-cli logs views-update` — Views update

**loops** — Manage loops

- `posthog-pp-cli loops create` — API for managing loops — named, cloud-executed agent automations triggered by schedule
- `posthog-pp-cli loops destroy` — Soft delete. Pauses every trigger's schedule. Owner or a project admin only.
- `posthog-pp-cli loops list` — List loops visible to the caller: personal loops they own, plus every team loop.
- `posthog-pp-cli loops partial-update` — Partial update.
- `posthog-pp-cli loops retrieve` — API for managing loops — named, cloud-executed agent automations triggered by schedule

**managed-viewsets** — Manage managed viewsets

- `posthog-pp-cli managed-viewsets retrieve` — Get all views associated with a specific managed viewset. GET /api/environments/{team_id}/managed_viewsets/{kind}/
- `posthog-pp-cli managed-viewsets update` — Enable or disable a managed viewset by kind.

**marketing-analytics** — Manage marketing analytics

- `posthog-pp-cli marketing-analytics conversion-goals-retrieve` — Read the configured conversion goals for the current project — each with its kind, target, last-30d count
- `posthog-pp-cli marketing-analytics data-sources-retrieve` — Check the platform → data-warehouse side of every native marketing integration: connection state, sync recency
- `posthog-pp-cli marketing-analytics diagnose-retrieve` — Aggregate data-source sync health, UTM attribution health
- `posthog-pp-cli marketing-analytics explain-conversion-goal-retrieve` — Break down a single conversion goal's events over a period by event name, utm_source, and matched integration
- `posthog-pp-cli marketing-analytics suggest-conversion-goals-retrieve` — Rank existing custom events as conversion-goal candidates by volume, UTM-tag coverage, and unique users
- `posthog-pp-cli marketing-analytics suggest-utm-mappings-retrieve` — Detect unmatched utm_source values from recent events and propose custom_source_mappings entries
- `posthog-pp-cli marketing-analytics test-mapping-create` — Test mapping create
- `posthog-pp-cli marketing-analytics utm-audit-retrieve` — Cross-reference campaigns with spend from ad platforms against pageview events with UTM parameters to identify tracking

**max-tools** — Manage max tools

- `posthog-pp-cli max-tools <project_id>` — Create and query insight create

**mcp-analytics** — Manage mcp analytics

- `posthog-pp-cli mcp-analytics feedback-create` — Create a new MCP feedback submission for the current project.
- `posthog-pp-cli mcp-analytics feedback-list` — List MCP feedback submissions for the current project, newest first.
- `posthog-pp-cli mcp-analytics intent-clusters-recompute` — Trigger an asynchronous recompute of the intent cluster snapshot.
- `posthog-pp-cli mcp-analytics intent-clusters-retrieve` — Return the most recent intent cluster snapshot for the current project.
- `posthog-pp-cli mcp-analytics missing-capabilities-create` — Create a new missing capability report for the current project.
- `posthog-pp-cli mcp-analytics missing-capabilities-list` — List missing capability reports for the current project, newest first.
- `posthog-pp-cli mcp-analytics sessions-activity-overview` — Aggregate counters, top tools, agent clients, and the most recent tool calls for the last 30 days
- `posthog-pp-cli mcp-analytics sessions-generate-intent` — Generate (or return the cached) LLM summary of the agent's goal for a session, derived from its recorded $mcp_intents.
- `posthog-pp-cli mcp-analytics sessions-intent-digest` — Generate (or return the cached) LLM digest of what agents are trying to do with this MCP server
- `posthog-pp-cli mcp-analytics sessions-list` — List MCP sessions for the current project, derived by grouping $mcp_tool_call events by $mcp_session_id.
- `posthog-pp-cli mcp-analytics sessions-tool-calls` — List a page of the $mcp_tool_call events that belong to a given $session_id, in chronological order.

**mcp-gateway** — Manage mcp gateway

- `posthog-pp-cli mcp-gateway audit-counts-retrieve` — Totals backing the quick-filter chips.
- `posthog-pp-cli mcp-gateway audit-list` — Read-only trail of proxied tool calls. Project admins see all calls.
- `posthog-pp-cli mcp-gateway audit-retrieve` — Read-only trail of proxied tool calls. Project admins see all calls.
- `posthog-pp-cli mcp-gateway config-apply-preset-create` — Set the policy baseline for members or agents (admin-only).
- `posthog-pp-cli mcp-gateway config-list` — The team's gateway settings, plus whether the caller can administer them.
- `posthog-pp-cli mcp-gateway config-set-all-servers-enabled-create` — Enable or disable every MCP server for the team (admin-only)
- `posthog-pp-cli mcp-gateway config-update-settings-create` — Update team gateway settings (admin-only).
- `posthog-pp-cli mcp-gateway members-list` — Admin overview of each member's gateway posture, plus the per-member server kill switch.
- `posthog-pp-cli mcp-gateway members-retrieve` — Admin overview of each member's gateway posture, plus the per-member server kill switch.
- `posthog-pp-cli mcp-gateway members-set-access-create` — Turn one gateway server off (or back on) for one member.
- `posthog-pp-cli mcp-gateway rules-create` — Team guardrails evaluated before any scope policy.
- `posthog-pp-cli mcp-gateway rules-destroy` — Team guardrails evaluated before any scope policy.
- `posthog-pp-cli mcp-gateway rules-list` — Team guardrails evaluated before any scope policy.
- `posthog-pp-cli mcp-gateway rules-partial-update` — Team guardrails evaluated before any scope policy.
- `posthog-pp-cli mcp-gateway rules-retrieve` — Team guardrails evaluated before any scope policy.
- `posthog-pp-cli mcp-gateway rules-update` — Team guardrails evaluated before any scope policy.
- `posthog-pp-cli mcp-gateway servers-destroy` — The team's gateway server registry.
- `posthog-pp-cli mcp-gateway servers-list` — The team's gateway server registry.
- `posthog-pp-cli mcp-gateway servers-partial-update` — The team's gateway server registry.
- `posthog-pp-cli mcp-gateway servers-policies-create` — Upsert per-tool states for a scope, returning the re-resolved catalog.
- `posthog-pp-cli mcp-gateway servers-retrieve` — The team's gateway server registry.
- `posthog-pp-cli mcp-gateway servers-set-template-enabled-create` — Enable or disable a catalog template for the team (admin-only)
- `posthog-pp-cli mcp-gateway servers-tools-retrieve` — Tool catalog with the resolved policy for a scope.
- `posthog-pp-cli mcp-gateway servers-update` — The team's gateway server registry.
- `posthog-pp-cli mcp-gateway service-accounts-access-create` — Grant or revoke this agent's access to one gateway server.
- `posthog-pp-cli mcp-gateway service-accounts-list` — PostHog's built-in agents and their MCP access grants. The catalog is fixed.
- `posthog-pp-cli mcp-gateway service-accounts-partial-update` — PostHog's built-in agents and their MCP access grants. The catalog is fixed.
- `posthog-pp-cli mcp-gateway service-accounts-retrieve` — PostHog's built-in agents and their MCP access grants. The catalog is fixed.
- `posthog-pp-cli mcp-gateway service-accounts-update` — PostHog's built-in agents and their MCP access grants. The catalog is fixed.

**mcp-server-installations** — Manage mcp server installations

- `posthog-pp-cli mcp-server-installations authorize-retrieve` — Start (or re-start) an OAuth flow.
- `posthog-pp-cli mcp-server-installations available-tools-retrieve` — Every tool the caller can currently reach, across all their connections.
- `posthog-pp-cli mcp-server-installations create` — Create
- `posthog-pp-cli mcp-server-installations destroy` — Destroy
- `posthog-pp-cli mcp-server-installations install-custom-create` — Install custom create
- `posthog-pp-cli mcp-server-installations install-template-create` — Install template create
- `posthog-pp-cli mcp-server-installations list` — List
- `posthog-pp-cli mcp-server-installations partial-update` — Partial update
- `posthog-pp-cli mcp-server-installations retrieve` — Retrieve
- `posthog-pp-cli mcp-server-installations update` — Update

**mcp-servers** — Manage mcp servers

- `posthog-pp-cli mcp-servers <project_id>` — Lists curated MCP server templates that users can install with one click.

**mcp-tools** — Manage mcp tools

- `posthog-pp-cli mcp-tools create` — Invoke an MCP tool by name.
- `posthog-pp-cli mcp-tools docs-search` — Run a hybrid (semantic + full-text) RAG search over the PostHog documentation via Inkeep.

**messaging-preferences** — Manage messaging preferences

- `posthog-pp-cli messaging-preferences add-opt-out-create` — Manually add a recipient to the opt-out list for a specific category or all marketing messages.
- `posthog-pp-cli messaging-preferences bulk-add-opt-outs-create` — Opt every recipient in the list out of the category named on their entry, or a default category.
- `posthog-pp-cli messaging-preferences export-opt-outs-csv-retrieve` — Stream the opt-out list for a category as a CSV file that can be re-imported as-is.
- `posthog-pp-cli messaging-preferences generate-link-create` — Generate an unsubscribe link for the current user's email address
- `posthog-pp-cli messaging-preferences opt-outs-retrieve` — Get opt-outs filtered by category or overall opt-outs if no category specified
- `posthog-pp-cli messaging-preferences remove-opt-out-create` — Opt a recipient back in to a specific category, or to all marketing messages.
- `posthog-pp-cli messaging-preferences webhook-url-retrieve` — Return the webhook URL for Customer.io integration setup.

**messaging-suppressions** — Manage messaging suppressions

- `posthog-pp-cli messaging-suppressions add-suppression-create` — Manually suppress an email address so no workflow sends to it.
- `posthog-pp-cli messaging-suppressions remove-suppression-create` — Remove an address from the suppression list so it can receive messages again.
- `posthog-pp-cli messaging-suppressions suppressions-retrieve` — List suppressed recipients for the team, most recently updated first.

**messaging-templates** — Manage messaging templates

- `posthog-pp-cli messaging-templates create` — Create
- `posthog-pp-cli messaging-templates destroy` — Hard delete of this model is not allowed. Use a patch API call to set 'deleted' to true
- `posthog-pp-cli messaging-templates list` — List
- `posthog-pp-cli messaging-templates partial-update` — Partial update
- `posthog-pp-cli messaging-templates retrieve` — Retrieve
- `posthog-pp-cli messaging-templates update` — Update

**metrics** — Manage metrics

- `posthog-pp-cli metrics attribute-values-retrieve` — Observed values for one metric attribute key, most frequent first. Backs the filter bar's value autocomplete.
- `posthog-pp-cli metrics attributes-retrieve` — Distinct attribute keys seen on the team's metrics (datapoint and resource attributes merged), most frequent first.
- `posthog-pp-cli metrics characterize-create` — Characterize a metric anomaly: compare an anomaly window against a baseline, find the onset
- `posthog-pp-cli metrics has-retrieve` — Has retrieve
- `posthog-pp-cli metrics query-create` — Query create
- `posthog-pp-cli metrics samples-create` — Raw individual emissions for a metric (the events model)
- `posthog-pp-cli metrics values-retrieve` — Distinct metric names for the team. Backs the picker UI.

**notebooks** — Manage notebooks

- `posthog-pp-cli notebooks all-activity-retrieve` — The API for interacting with Notebooks.
- `posthog-pp-cli notebooks create` — The API for interacting with Notebooks.
- `posthog-pp-cli notebooks destroy` — Hard delete of this model is not allowed. Use a patch API call to set 'deleted' to true
- `posthog-pp-cli notebooks list` — The API for interacting with Notebooks.
- `posthog-pp-cli notebooks partial-update` — The API for interacting with Notebooks.
- `posthog-pp-cli notebooks recording-comments-retrieve` — The API for interacting with Notebooks.
- `posthog-pp-cli notebooks retrieve` — The API for interacting with Notebooks.
- `posthog-pp-cli notebooks update` — The API for interacting with Notebooks.

**object-media-previews** — Manage object media previews

- `posthog-pp-cli object-media-previews create` — Create
- `posthog-pp-cli object-media-previews destroy` — Destroy
- `posthog-pp-cli object-media-previews list` — List
- `posthog-pp-cli object-media-previews partial-update` — Partial update
- `posthog-pp-cli object-media-previews preferred-for-event-retrieve` — Get the preferred media preview for an event definition. Most recent user-uploaded, then most recent exported asset.
- `posthog-pp-cli object-media-previews retrieve` — Retrieve
- `posthog-pp-cli object-media-previews update` — Update

**organizations** — Manage organizations

- `posthog-pp-cli organizations create` — Create
- `posthog-pp-cli organizations destroy` — Destroy
- `posthog-pp-cli organizations list` — List
- `posthog-pp-cli organizations partial-update` — Partial update
- `posthog-pp-cli organizations retrieve` — Retrieve
- `posthog-pp-cli organizations update` — Update

**paths-v2** — Manage paths v2

- `posthog-pp-cli paths-v2 <project_id>` — Converts a displayed journeys segment into the funnel query that reproduces its unique-actor count exactly.

**persons** — Manage persons

- `posthog-pp-cli persons all-activity-retrieve` — This endpoint is meant for reading and deleting persons.
- `posthog-pp-cli persons batch-by-distinct-ids-create` — This endpoint is meant for reading and deleting persons.
- `posthog-pp-cli persons batch-by-uuids-create` — This endpoint is meant for reading and deleting persons.
- `posthog-pp-cli persons bulk-delete-create` — This endpoint allows you to bulk delete persons, either by the PostHog person IDs or by distinct IDs.
- `posthog-pp-cli persons cohorts-retrieve` — This endpoint is meant for reading and deleting persons.
- `posthog-pp-cli persons deletion-status-list` — List the status of queued event deletions for persons.
- `posthog-pp-cli persons list` — This endpoint is meant for reading and deleting persons.
- `posthog-pp-cli persons partial-update` — This endpoint is meant for reading and deleting persons.
- `posthog-pp-cli persons properties-at-time-retrieve` — Get person properties as they existed at a specific point in time.
- `posthog-pp-cli persons reset-distinct-id-create` — Reset a distinct_id for a deleted person. This allows the distinct_id to be used again.
- `posthog-pp-cli persons retrieve` — This endpoint is meant for reading and deleting persons.
- `posthog-pp-cli persons update` — Only for setting properties on the person. 'properties' from the request data will be updated via a '$set' event.
- `posthog-pp-cli persons values-retrieve` — This endpoint is meant for reading and deleting persons.

**plugin-configs** — Manage plugin configs


**posthog-connections** — Manage posthog connections


**product-enablement** — Manage product enablement

- `posthog-pp-cli product-enablement <project_id>` — Create

**product-tours** — Manage product tours

- `posthog-pp-cli product-tours create` — Create, read, update, and manage product tours and their targeting.
- `posthog-pp-cli product-tours destroy` — Create, read, update, and manage product tours and their targeting.
- `posthog-pp-cli product-tours list` — Create, read, update, and manage product tours and their targeting.
- `posthog-pp-cli product-tours partial-update` — Create, read, update, and manage product tours and their targeting.
- `posthog-pp-cli product-tours retrieve` — Create, read, update, and manage product tours and their targeting.
- `posthog-pp-cli product-tours update` — Create, read, update, and manage product tours and their targeting.

**project-secret-api-keys** — Manage project secret api keys

- `posthog-pp-cli project-secret-api-keys secret-api-keys-create` — Secret api keys create
- `posthog-pp-cli project-secret-api-keys secret-api-keys-destroy` — Secret api keys destroy
- `posthog-pp-cli project-secret-api-keys secret-api-keys-list` — Secret api keys list
- `posthog-pp-cli project-secret-api-keys secret-api-keys-partial-update` — Secret api keys partial update
- `posthog-pp-cli project-secret-api-keys secret-api-keys-retrieve` — Secret api keys retrieve
- `posthog-pp-cli project-secret-api-keys secret-api-keys-update` — Secret api keys update

**property-access-controls** — Manage property access controls

- `posthog-pp-cli property-access-controls create` — Create or update a property access control rule.
- `posthog-pp-cli property-access-controls destroy` — Delete a property access control rule.
- `posthog-pp-cli property-access-controls retrieve` — Get all property access control rules for a property definition.

**property-definitions** — Manage property definitions

- `posthog-pp-cli property-definitions bulk-update-tags-create` — Bulk update tags on multiple objects.
- `posthog-pp-cli property-definitions destroy` — Destroy
- `posthog-pp-cli property-definitions list` — List
- `posthog-pp-cli property-definitions partial-update` — Partial update
- `posthog-pp-cli property-definitions retrieve` — Retrieve
- `posthog-pp-cli property-definitions seen-together-retrieve` — Allows a caller to provide a list of event names and a single property name Returns a map of the event names to a
- `posthog-pp-cli property-definitions update` — Update

**public-hog-function-templates** — Manage public hog function templates

- `posthog-pp-cli public-hog-function-templates` — List

**pulse** — Manage pulse

- `posthog-pp-cli pulse brief-configs-create` — Brief configs create
- `posthog-pp-cli pulse brief-configs-destroy` — Brief configs destroy
- `posthog-pp-cli pulse brief-configs-list` — Brief configs list
- `posthog-pp-cli pulse brief-configs-partial-update` — Brief configs partial update
- `posthog-pp-cli pulse brief-configs-retrieve` — Brief configs retrieve
- `posthog-pp-cli pulse brief-configs-update` — Brief configs update
- `posthog-pp-cli pulse briefs-generate-create` — Briefs generate create
- `posthog-pp-cli pulse briefs-list` — Briefs list
- `posthog-pp-cli pulse briefs-retrieve` — Briefs retrieve

**query** — Manage query

- `posthog-pp-cli query check-auth-for-async-create` — DRF ViewSet mixin that gates coalesced responses behind permission checks.
- `posthog-pp-cli query create` — DRF ViewSet mixin that gates coalesced responses behind permission checks.
- `posthog-pp-cli query create-with-kind` — DRF ViewSet mixin that gates coalesced responses behind permission checks.
- `posthog-pp-cli query destroy` — (Experimental)
- `posthog-pp-cli query draft-sql-retrieve` — DRF ViewSet mixin that gates coalesced responses behind permission checks.
- `posthog-pp-cli query retrieve` — (Experimental)
- `posthog-pp-cli query upgrade-create` — Upgrades a query without executing it. Returns a query with all nodes migrated to the latest version.

**quota-limits** — Manage quota limits

- `posthog-pp-cli quota-limits <project_id>` — Return the current quota-limit state for the team identified in the URL, keyed by `QuotaResource` value.

**reminders** — Manage reminders

- `posthog-pp-cli reminders create` — Create
- `posthog-pp-cli reminders destroy` — Destroy
- `posthog-pp-cli reminders list` — List
- `posthog-pp-cli reminders partial-update` — Partial update
- `posthog-pp-cli reminders retrieve` — Retrieve
- `posthog-pp-cli reminders update` — Update

**review-hog** — Manage review hog

- `posthog-pp-cli review-hog blind-spots-list` — List the `review-hog-blind-spots-*` skills visible to the requesting user — the canonical skill plus the customs they
- `posthog-pp-cli review-hog blind-spots-partial-update` — Make a `review-hog-blind-spots-*` skill the single sweep that runs on the requesting user's PR reviews
- `posthog-pp-cli review-hog perspectives-list` — List the `review-hog-perspective-*` skills visible to the requesting user — the canonical perspectives plus the customs
- `posthog-pp-cli review-hog perspectives-partial-update` — Toggle whether a `review-hog-perspective-*` skill runs on the requesting user's PR reviews.
- `posthog-pp-cli review-hog reviews-list` — Recent ReviewHog reviews on this project: actively running reviews first (with the in-flight turn's stage)
- `posthog-pp-cli review-hog reviews-perspective-stats-retrieve` — How many findings each review skill (perspective or blind-spot sweep)
- `posthog-pp-cli review-hog reviews-retrieve` — One completed ReviewHog review on this project, with the latest turn's validated findings
- `posthog-pp-cli review-hog reviews-trigger-create` — Start a ReviewHog review of any pull request the project's GitHub App installation can access
- `posthog-pp-cli review-hog validators-list` — List the `review-hog-validation-*` skills visible to the requesting user — the canonical validator plus the customs
- `posthog-pp-cli review-hog validators-partial-update` — Make a `review-hog-validation-*` skill the single validator that runs on the requesting user's PR reviews

**sandbox-custom-images** — Manage sandbox custom images

- `posthog-pp-cli sandbox-custom-images create` — Create a draft custom image and start its interactive image-builder agent task.
- `posthog-pp-cli sandbox-custom-images destroy` — API for custom sandbox base images, built on top of the VM sandbox base via an image-builder agent.
- `posthog-pp-cli sandbox-custom-images list` — API for custom sandbox base images, built on top of the VM sandbox base via an image-builder agent.
- `posthog-pp-cli sandbox-custom-images partial-update` — Rename or update the description of a custom image.
- `posthog-pp-cli sandbox-custom-images retrieve` — API for custom sandbox base images, built on top of the VM sandbox base via an image-builder agent.

**sandbox-environments** — Manage sandbox environments

- `posthog-pp-cli sandbox-environments sandbox-create` — API for managing sandbox environments that control network access for task runs.
- `posthog-pp-cli sandbox-environments sandbox-destroy` — API for managing sandbox environments that control network access for task runs.
- `posthog-pp-cli sandbox-environments sandbox-list` — API for managing sandbox environments that control network access for task runs.
- `posthog-pp-cli sandbox-environments sandbox-partial-update` — API for managing sandbox environments that control network access for task runs.
- `posthog-pp-cli sandbox-environments sandbox-retrieve` — API for managing sandbox environments that control network access for task runs.

**saved** — Manage saved

- `posthog-pp-cli saved create` — Create a saved heatmap for a page URL.
- `posthog-pp-cli saved destroy` — Hard delete of this model is not allowed. Use a patch API call to set 'deleted' to true
- `posthog-pp-cli saved list` — List saved heatmaps for the project.
- `posthog-pp-cli saved partial-update` — Update a saved heatmap (e.g. rename, change widths, or soft-delete via 'deleted').
- `posthog-pp-cli saved preflight-create` — Fetch a page URL server-side and report whether it allows being embedded in the live preview iframe
- `posthog-pp-cli saved prewarm-create` — Speculatively render a screenshot for a page URL ahead of heatmap creation, so it's ready (or closer to ready)
- `posthog-pp-cli saved retrieve` — Get a single saved heatmap by its short_id, including per-width render status.

**saved-query-column-annotations** — Manage saved query column annotations

- `posthog-pp-cli saved-query-column-annotations create` — Read and edit semantic descriptions of data-modelling views and columns surfaced to the AI agent.
- `posthog-pp-cli saved-query-column-annotations destroy` — Read and edit semantic descriptions of data-modelling views and columns surfaced to the AI agent.
- `posthog-pp-cli saved-query-column-annotations list` — Read and edit semantic descriptions of data-modelling views and columns surfaced to the AI agent.
- `posthog-pp-cli saved-query-column-annotations partial-update` — Read and edit semantic descriptions of data-modelling views and columns surfaced to the AI agent.
- `posthog-pp-cli saved-query-column-annotations retrieve` — Read and edit semantic descriptions of data-modelling views and columns surfaced to the AI agent.
- `posthog-pp-cli saved-query-column-annotations update` — Read and edit semantic descriptions of data-modelling views and columns surfaced to the AI agent.

**scheduled-changes** — Manage scheduled changes

- `posthog-pp-cli scheduled-changes create` — Create, read, update and delete scheduled changes.
- `posthog-pp-cli scheduled-changes destroy` — Create, read, update and delete scheduled changes.
- `posthog-pp-cli scheduled-changes list` — Create, read, update and delete scheduled changes.
- `posthog-pp-cli scheduled-changes partial-update` — Create, read, update and delete scheduled changes.
- `posthog-pp-cli scheduled-changes retrieve` — Create, read, update and delete scheduled changes.
- `posthog-pp-cli scheduled-changes update` — Create, read, update and delete scheduled changes.

**schema-property-groups** — Manage schema property groups

- `posthog-pp-cli schema-property-groups create` — Create
- `posthog-pp-cli schema-property-groups destroy` — Destroy
- `posthog-pp-cli schema-property-groups list` — List
- `posthog-pp-cli schema-property-groups partial-update` — Partial update
- `posthog-pp-cli schema-property-groups retrieve` — Retrieve
- `posthog-pp-cli schema-property-groups update` — Update

**sdk-health** — Manage sdk health

- `posthog-pp-cli sdk-health <project_id>` — Returns a pre-digested health assessment of the PostHog SDKs the project is using.

**session-group-summaries** — Manage session group summaries

- `posthog-pp-cli session-group-summaries create` — API for retrieving and managing stored group session summaries.
- `posthog-pp-cli session-group-summaries destroy` — API for retrieving and managing stored group session summaries.
- `posthog-pp-cli session-group-summaries list` — API for retrieving and managing stored group session summaries.
- `posthog-pp-cli session-group-summaries partial-update` — API for retrieving and managing stored group session summaries.
- `posthog-pp-cli session-group-summaries retrieve` — API for retrieving and managing stored group session summaries.
- `posthog-pp-cli session-group-summaries update` — API for retrieving and managing stored group session summaries.

**session-recording-playlists** — Manage session recording playlists

- `posthog-pp-cli session-recording-playlists create` — Create
- `posthog-pp-cli session-recording-playlists destroy` — Hard delete of this model is not allowed. Use a patch API call to set 'deleted' to true
- `posthog-pp-cli session-recording-playlists list` — Override list to include synthetic playlists.
- `posthog-pp-cli session-recording-playlists partial-update` — Partial update
- `posthog-pp-cli session-recording-playlists retrieve` — Retrieve
- `posthog-pp-cli session-recording-playlists update` — Update

**session-recordings** — Manage session recordings

- `posthog-pp-cli session-recordings bulk-delete-create` — Delete a batch of session recordings by session ID. Deletion is permanent and cannot be undone.
- `posthog-pp-cli session-recordings destroy` — Destroy
- `posthog-pp-cli session-recordings list` — List
- `posthog-pp-cli session-recordings partial-update` — Partial update
- `posthog-pp-cli session-recordings retrieve` — Retrieve
- `posthog-pp-cli session-recordings update` — Update

**session-summaries** — Manage session summaries

- `posthog-pp-cli session-summaries create` — Generate AI summary for a group of session recordings to find patterns and generate a notebook.
- `posthog-pp-cli session-summaries create-individually` — Generate AI individual summary for each session, without grouping.
- `posthog-pp-cli session-summaries retrieve-config` — Retrieve the team's session summaries configuration (product context used to tailor single-session replay summaries).
- `posthog-pp-cli session-summaries update-config` — Update the team's session summaries configuration (product context used to tailor single-session replay summaries).

**sessions** — Manage sessions

- `posthog-pp-cli sessions property-definitions-retrieve` — Property definitions retrieve
- `posthog-pp-cli sessions values-retrieve` — Values retrieve

**signals** — Manage signals

- `posthog-pp-cli signals processing-list` — Return current processing state including pause status.
- `posthog-pp-cli signals processing-pause-destroy` — View and control signal processing pipeline state for a team.
- `posthog-pp-cli signals processing-pause-update` — View and control signal processing pipeline state for a team.
- `posthog-pp-cli signals report-artefacts-create` — Append an artefact to a report (see artefact_type for the writable types).
- `posthog-pp-cli signals report-artefacts-destroy` — Delete an artefact, addressed by id.
- `posthog-pp-cli signals report-artefacts-diff` — Fetch the unified diff of a `commit` artefact's branch against the repository default branch via the team's GitHub
- `posthog-pp-cli signals report-artefacts-list` — List every artefact on a report — the full work log: signal findings (the evidence behind the report)
- `posthog-pp-cli signals report-artefacts-partial-update` — Replace the content of an existing artefact, addressed by id.
- `posthog-pp-cli signals report-artefacts-retrieve` — Get one artefact by id, content parsed (and reviewers enriched) the same way as the list.
- `posthog-pp-cli signals report-pr-checks` — Fetch the CI status (GitHub Actions check runs and legacy commit statuses)
- `posthog-pp-cli signals report-pr-comments` — Fetch the pull request's conversation comments and inline review comments, merged chronologically
- `posthog-pp-cli signals report-pr-review-comment-destroy` — Delete one of the requesting user's own review comments
- `posthog-pp-cli signals report-pr-review-comment-reaction-destroy` — Remove one of the requesting user's own reactions from a review comment
- `posthog-pp-cli signals report-pr-review-comment-reactions-create` — React to a review comment as the requesting user
- `posthog-pp-cli signals report-pr-review-comment-update` — Edit one of the requesting user's own review comments
- `posthog-pp-cli signals report-pr-review-comments-create` — Post an inline review comment on the report's implementation pull request
- `posthog-pp-cli signals reports-bulk-state-create` — Transition many reports to a new state in one call.
- `posthog-pp-cli signals reports-feedback-create` — Record the thumbs rating at the end of a report, with an optional note.
- `posthog-pp-cli signals reports-list` — Reports list
- `posthog-pp-cli signals reports-partial-update` — Edit the human-facing title and/or summary (description) of a signal report, addressed by id.
- `posthog-pp-cli signals reports-refund-create` — Refund the flat charge for this report's implementation PR and archive the report.
- `posthog-pp-cli signals reports-refund-summary-retrieve` — Aggregate credited-path refunds across the whole organization for the current billing period — counts only
- `posthog-pp-cli signals reports-retrieve` — Reports retrieve
- `posthog-pp-cli signals reports-retrieve-projects` — Fetch all signals for a report from ClickHouse, including full metadata.
- `posthog-pp-cli signals reports-state-create` — Transition a report to a new state. The model validates allowed transitions.
- `posthog-pp-cli signals reports-viewed-create` — Record that the caller opened this report's detail view.
- `posthog-pp-cli signals scout-config-create` — Register the config for a `signals-scout-*` skill immediately, without waiting for the coordinator to auto-register it.
- `posthog-pp-cli signals scout-config-destroy` — Delete one scout config by its `id`, removing the per-(team, skill) schedule/emit row outright.
- `posthog-pp-cli signals scout-config-list` — List the per-(team, skill) scout configs for this project.
- `posthog-pp-cli signals scout-config-run` — Dispatch one on-demand run of this scout immediately, regardless of its schedule.
- `posthog-pp-cli signals scout-config-sync` — Materialize the scout fleet for this project on demand (idempotent): seed the canonical `signals-scout-*` skills
- `posthog-pp-cli signals scout-config-update` — Tune one scout: change its schedule (rolling `run_interval_minutes`
- `posthog-pp-cli signals scout-create` — Create a `signals-scout-*` skill and its runnable config atomically. The skill always receives the report-channel tools.
- `posthog-pp-cli signals scout-edit-report` — Rewrite a report's title/summary, append a note, and/or set its suggested reviewers.
- `posthog-pp-cli signals scout-emit` — Fire `emit_signal` with `source_product = signals_scout`. The `finding_id` is baked into the deterministic `Signal.
- `posthog-pp-cli signals scout-emit-report` — The second emit channel: author a complete `SignalReport` directly instead of emitting a weak signal.
- `posthog-pp-cli signals scout-members-list` — Return the people who can review work on this project — one row per member with access to it
- `posthog-pp-cli signals scout-metadata-get` — Return the project's scout metadata: whether it is enrolled, the current announcement banner (e.g.
- `posthog-pp-cli signals scout-notes-create` — Leave a steering note the scout fleet reads on its next runs.
- `posthog-pp-cli signals scout-notes-destroy` — Delete one note by its `id`, retiring it from every scout's view.
- `posthog-pp-cli signals scout-notes-list` — Return the steering notes left for this project's scouts, newest first.
- `posthog-pp-cli signals scout-project-profile-get` — Return the team's deterministic project profile.
- `posthog-pp-cli signals scout-record-output` — The structured-output channel: record schema-validated records this run produced.
- `posthog-pp-cli signals scout-runs-emission-reports` — Best-effort reverse of the report -> signals link.
- `posthog-pp-cli signals scout-runs-emission-reports-batch` — Batched form of the per-run emission-reports endpoint.
- `posthog-pp-cli signals scout-runs-emissions` — Return the findings a `SignalScoutRun` emitted to the inbox
- `posthog-pp-cli signals scout-runs-emissions-batch` — Batched form of the per-run emissions endpoint: return the findings every requested `SignalScoutRun` emitted
- `posthog-pp-cli signals scout-runs-findings-summary` — Return a cheap fleet-wide tally of the output the scout troop produced in the recent window — the finding count
- `posthog-pp-cli signals scout-runs-list` — Return the most recent `SignalScoutRun` summaries for this project, newest first.
- `posthog-pp-cli signals scout-runs-recent-emissions` — Return the team's recently emitted scout findings across *every* run
- `posthog-pp-cli signals scout-runs-retrieve` — Return the full `SignalScoutRun` row. Status, timing, and error flow from the linked `tasks.TaskRun`.
- `posthog-pp-cli signals scout-scratchpad-forget` — Delete an entry by key. Returns `deleted=false` if no row matched.
- `posthog-pp-cli signals scout-scratchpad-remember` — Upsert a memory keyed on `(team, key)`. Re-using a key updates the existing entry in place.
- `posthog-pp-cli signals scout-scratchpad-search` — Return `SignalScratchpad` entries for this project, newest-first.
- `posthog-pp-cli signals source-configs-create` — Source configs create
- `posthog-pp-cli signals source-configs-destroy` — Source configs destroy
- `posthog-pp-cli signals source-configs-list` — Source configs list
- `posthog-pp-cli signals source-configs-partial-update` — Source configs partial update
- `posthog-pp-cli signals source-configs-retrieve` — Source configs retrieve
- `posthog-pp-cli signals source-configs-update` — Source configs update

**single-session-summaries** — Manage single session summaries

- `posthog-pp-cli single-session-summaries list` — List stored AI-generated session summaries for the team, one row per session (latest summary kept).
- `posthog-pp-cli single-session-summaries retrieve` — Get the latest stored AI summary for a single session by `session_id`.

**stamphog** — Manage stamphog

- `posthog-pp-cli stamphog digest-channels-create` — Per-audience Slack destinations for the daily merged-PR digest.
- `posthog-pp-cli stamphog digest-channels-destroy` — Per-audience Slack destinations for the daily merged-PR digest.
- `posthog-pp-cli stamphog digest-channels-list` — Per-audience Slack destinations for the daily merged-PR digest.
- `posthog-pp-cli stamphog digest-channels-partial-update` — Per-audience Slack destinations for the daily merged-PR digest.
- `posthog-pp-cli stamphog digest-channels-retrieve` — Per-audience Slack destinations for the daily merged-PR digest.
- `posthog-pp-cli stamphog digest-channels-update` — Per-audience Slack destinations for the daily merged-PR digest.
- `posthog-pp-cli stamphog digest-runs-list` — Read-only history of posted (or attempted) digests, filterable by digest channel.
- `posthog-pp-cli stamphog digest-runs-retrieve` — Read-only history of posted (or attempted) digests, filterable by digest channel.
- `posthog-pp-cli stamphog pull-requests-list` — Read-only pull requests stamphog knows about, filterable by PR number and merge state.
- `posthog-pp-cli stamphog pull-requests-retrieve` — Read-only pull requests stamphog knows about, filterable by PR number and merge state.
- `posthog-pp-cli stamphog repo-configs-create` — Per-repo stamphog settings — enable/disable review, GitHub App installation, policy overrides.
- `posthog-pp-cli stamphog repo-configs-destroy` — Per-repo stamphog settings — enable/disable review, GitHub App installation, policy overrides.
- `posthog-pp-cli stamphog repo-configs-install-info-retrieve` — Per-repo stamphog settings — enable/disable review, GitHub App installation, policy overrides.
- `posthog-pp-cli stamphog repo-configs-list` — Per-repo stamphog settings — enable/disable review, GitHub App installation, policy overrides.
- `posthog-pp-cli stamphog repo-configs-partial-update` — Per-repo stamphog settings — enable/disable review, GitHub App installation, policy overrides.
- `posthog-pp-cli stamphog repo-configs-retrieve` — Per-repo stamphog settings — enable/disable review, GitHub App installation, policy overrides.
- `posthog-pp-cli stamphog repo-configs-sync-installation-create` — Per-repo stamphog settings — enable/disable review, GitHub App installation, policy overrides.
- `posthog-pp-cli stamphog repo-configs-update` — Per-repo stamphog settings — enable/disable review, GitHub App installation, policy overrides.
- `posthog-pp-cli stamphog review-runs-list` — Read-only history of stamphog review runs, filterable by repository, PR number, and status.
- `posthog-pp-cli stamphog review-runs-retrieve` — Read-only history of stamphog review runs, filterable by repository, PR number, and status.

**streamlit-apps** — Manage streamlit apps

- `posthog-pp-cli streamlit-apps create` — Create a streamlit app
- `posthog-pp-cli streamlit-apps destroy` — Delete a streamlit app
- `posthog-pp-cli streamlit-apps list` — List streamlit apps
- `posthog-pp-cli streamlit-apps partial-update` — Partially update a streamlit app
- `posthog-pp-cli streamlit-apps retrieve` — Retrieve a streamlit app
- `posthog-pp-cli streamlit-apps update` — Update a streamlit app

**subscriptions** — Manage subscriptions

- `posthog-pp-cli subscriptions create` — Create
- `posthog-pp-cli subscriptions destroy` — Hard delete of this model is not allowed. Use a patch API call to set 'deleted' to true
- `posthog-pp-cli subscriptions list` — List
- `posthog-pp-cli subscriptions partial-update` — Partial update
- `posthog-pp-cli subscriptions retrieve` — Retrieve
- `posthog-pp-cli subscriptions summary-quota-retrieve` — Summary quota retrieve
- `posthog-pp-cli subscriptions update` — Update

**surveys** — Manage surveys

- `posthog-pp-cli surveys all-activity-retrieve` — All activity retrieve
- `posthog-pp-cli surveys create` — Create
- `posthog-pp-cli surveys destroy` — Destroy
- `posthog-pp-cli surveys global-stats-retrieve` — Get aggregated response statistics across all surveys. Args: date_from: Optional ISO timestamp for start date (e.g.
- `posthog-pp-cli surveys list` — List
- `posthog-pp-cli surveys partial-update` — Partial update
- `posthog-pp-cli surveys question-labels` — Return a slim list of question labels for the team's surveys.
- `posthog-pp-cli surveys responses-count-retrieve` — Get response counts for all surveys.
- `posthog-pp-cli surveys retrieve` — Retrieve
- `posthog-pp-cli surveys update` — Update

**taggers** — Manage taggers

- `posthog-pp-cli taggers create` — Create
- `posthog-pp-cli taggers destroy` — Hard delete of this model is not allowed. Use a patch API call to set 'deleted' to true
- `posthog-pp-cli taggers list` — List
- `posthog-pp-cli taggers partial-update` — Partial update
- `posthog-pp-cli taggers retrieve` — Retrieve
- `posthog-pp-cli taggers test-hog-create` — Test Hog tagger code against sample events without saving.
- `posthog-pp-cli taggers update` — Update

**task-activity** — Manage task activity

- `posthog-pp-cli task-activity list` — Task lifecycle rows collapse per task. Comment notifications remain separate.
- `posthog-pp-cli task-activity mark-read-create` — Clear collapsed task activity through task timestamps and individual comment activity through activity IDs.

**task-automations** — Manage task automations

- `posthog-pp-cli task-automations create` — API for managing scheduled task automations.
- `posthog-pp-cli task-automations destroy` — API for managing scheduled task automations.
- `posthog-pp-cli task-automations list` — API for managing scheduled task automations.
- `posthog-pp-cli task-automations partial-update` — API for managing scheduled task automations.
- `posthog-pp-cli task-automations retrieve` — API for managing scheduled task automations.

**task-channels** — Manage task channels

- `posthog-pp-cli task-channels create` — Returns the existing public channel with the (normalized) name, creating it if needed.
- `posthog-pp-cli task-channels destroy` — API for task channels — the shared feeds tasks are kicked off in.
- `posthog-pp-cli task-channels list` — All live public channels plus the requester's personal #me channel (created on first list).
- `posthog-pp-cli task-channels partial-update` — API for task channels — the shared feeds tasks are kicked off in.
- `posthog-pp-cli task-channels retrieve` — API for task channels — the shared feeds tasks are kicked off in.

**task-mentions** — Manage task mentions

- `posthog-pp-cli task-mentions <project_id>` — Thread messages that @-mention the requester, newest first, restricted to tasks they can see.

**tasks** — Manage tasks

- `posthog-pp-cli tasks active-wizard-run-retrieve` — Returns the most recent onboarding wizard cloud run for the current project when it is still running (or completed
- `posthog-pp-cli tasks create` — API for managing tasks within a project. Tasks represent units of work to be performed by an agent.
- `posthog-pp-cli tasks destroy` — API for managing tasks within a project. Tasks represent units of work to be performed by an agent.
- `posthog-pp-cli tasks list` — Get a list of tasks for the current project, with optional filtering by origin product, stage, organization, repository
- `posthog-pp-cli tasks partial-update` — API for managing tasks within a project. Tasks represent units of work to be performed by an agent.
- `posthog-pp-cli tasks pinned-retrieve` — Return the visible tasks pinned by the requester in the current project.
- `posthog-pp-cli tasks repositories-retrieve` — Return the set of repositories referenced by non-deleted, non-internal tasks in the current project.
- `posthog-pp-cli tasks repository-readiness-retrieve` — Get autonomy readiness details for a specific repository in the current project.
- `posthog-pp-cli tasks retrieve` — Retrieve a single task by ID.
- `posthog-pp-cli tasks slack-thread-context-retrieve` — PostHog-internal debug tool.
- `posthog-pp-cli tasks summaries-create` — Returns summary for the requested tasks: `id`, `title`, `repository`, `created_at`, `updated_at`
- `posthog-pp-cli tasks update` — API for managing tasks within a project. Tasks represent units of work to be performed by an agent.
- `posthog-pp-cli tasks warm-create` — Warm a full idling Run for a Code-app cloud task while the user composes: boot a sandbox, clone the repo

**tracing** — Manage tracing

- `posthog-pp-cli tracing spans-aggregate-create` — Spans aggregate create
- `posthog-pp-cli tracing spans-attribute-breakdown-create` — Spans attribute breakdown create
- `posthog-pp-cli tracing spans-attributes-retrieve` — Spans attributes retrieve
- `posthog-pp-cli tracing spans-count-create` — Spans count create
- `posthog-pp-cli tracing spans-duration-histogram-create` — Spans duration histogram create
- `posthog-pp-cli tracing spans-has-spans-retrieve` — Spans has spans retrieve
- `posthog-pp-cli tracing spans-latency-heatmap-create` — Spans latency heatmap create
- `posthog-pp-cli tracing spans-query-create` — Spans query create
- `posthog-pp-cli tracing spans-service-names-retrieve` — Spans service names retrieve
- `posthog-pp-cli tracing spans-sparkline-create` — Spans sparkline create
- `posthog-pp-cli tracing spans-symbol-stats-create` — Spans symbol stats create
- `posthog-pp-cli tracing spans-trace-create` — Spans trace create
- `posthog-pp-cli tracing spans-tree-create` — Spans tree create
- `posthog-pp-cli tracing spans-values-retrieve` — Spans values retrieve
- `posthog-pp-cli tracing views-create` — Views create
- `posthog-pp-cli tracing views-destroy` — Views destroy
- `posthog-pp-cli tracing views-list` — Views list
- `posthog-pp-cli tracing views-partial-update` — Views partial update
- `posthog-pp-cli tracing views-retrieve` — Views retrieve
- `posthog-pp-cli tracing views-update` — Views update

**uploaded-media** — Manage uploaded media

- `posthog-pp-cli uploaded-media <project_id>` — When object storage is available this API allows upload of media which can be used, for example

**user-home-settings** — Manage user home settings

- `posthog-pp-cli user-home-settings partial-update` — Update the authenticated user's pinned sidebar tabs and/or homepage for the current team. Pass `@me` as the UUID.
- `posthog-pp-cli user-home-settings retrieve` — Get the authenticated user's pinned sidebar tabs and configured homepage for the current team. Pass `@me` as the UUID.

**user-interview-topics** — Manage user interview topics

- `posthog-pp-cli user-interview-topics create` — Planned user interview topics: who we want to target and what we want to ask about.
- `posthog-pp-cli user-interview-topics destroy` — Planned user interview topics: who we want to target and what we want to ask about.
- `posthog-pp-cli user-interview-topics list` — Planned user interview topics: who we want to target and what we want to ask about.
- `posthog-pp-cli user-interview-topics partial-update` — Planned user interview topics: who we want to target and what we want to ask about.
- `posthog-pp-cli user-interview-topics retrieve` — Planned user interview topics: who we want to target and what we want to ask about.
- `posthog-pp-cli user-interview-topics update` — Planned user interview topics: who we want to target and what we want to ask about.

**user-interviews** — Manage user interviews

- `posthog-pp-cli user-interviews create` — Create
- `posthog-pp-cli user-interviews destroy` — Destroy
- `posthog-pp-cli user-interviews list` — List
- `posthog-pp-cli user-interviews partial-update` — Partial update
- `posthog-pp-cli user-interviews retrieve` — Retrieve
- `posthog-pp-cli user-interviews search-create` — Embed `query` with the same model used to index interview transcripts and summaries
- `posthog-pp-cli user-interviews update` — Update

**users** — Manage users

- `posthog-pp-cli users cancel-email-change-request-partial-update` — Cancel email change request partial update
- `posthog-pp-cli users destroy` — Destroy
- `posthog-pp-cli users list` — List
- `posthog-pp-cli users partial-update` — Update one or more of the authenticated user's profile fields or settings.
- `posthog-pp-cli users request-email-verification-create` — Request email verification create
- `posthog-pp-cli users retrieve` — Retrieve a user's profile and settings.
- `posthog-pp-cli users update` — Replace the authenticated user's profile and settings. Pass `@me` as the UUID to update the authenticated user.
- `posthog-pp-cli users verify-email-create` — Verify email create

**vision** — Manage vision

- `posthog-pp-cli vision actions-create` — CRUD for Replay Vision actions — scheduled 'and then…' automations over a scanner's observations.
- `posthog-pp-cli vision actions-destroy` — CRUD for Replay Vision actions — scheduled 'and then…' automations over a scanner's observations.
- `posthog-pp-cli vision actions-list` — CRUD for Replay Vision actions — scheduled 'and then…' automations over a scanner's observations.
- `posthog-pp-cli vision actions-partial-update` — CRUD for Replay Vision actions — scheduled 'and then…' automations over a scanner's observations.
- `posthog-pp-cli vision actions-retrieve` — CRUD for Replay Vision actions — scheduled 'and then…' automations over a scanner's observations.
- `posthog-pp-cli vision actions-run-create` — Run this summary now
- `posthog-pp-cli vision actions-runs-list` — Read-only run history for a single vision action (nested under /vision/actions/{action_id}/runs/).
- `posthog-pp-cli vision actions-runs-retrieve` — Read-only run history for a single vision action (nested under /vision/actions/{action_id}/runs/).
- `posthog-pp-cli vision environment-quota-retrieve` — Environment quota retrieve
- `posthog-pp-cli vision observations-create-task-create` — Create a PostHog Task from this observation's finding so it can be triaged and fixed.
- `posthog-pp-cli vision observations-label-create` — Set or update the observation's shared label: whether the scanner scored the session correctly
- `posthog-pp-cli vision observations-label-destroy` — Remove the observation's shared label. Requires editor access to the scanner.
- `posthog-pp-cli vision observations-list` — Read-only access to a session's observations across every scanner the caller can read, for the replay-page dock.
- `posthog-pp-cli vision observations-retrieve` — Retrieve one observation.
- `posthog-pp-cli vision observations-retry-create` — Delete a failed or ineligible observation and re-run its scanner on the same recording.
- `posthog-pp-cli vision scanners-affected-cohort-create` — Save the users this scanner matched as a static cohort, for surveys, funnels, and retention analysis.
- `posthog-pp-cli vision scanners-bulk-observe-create` — Apply this scanner to many sessions on demand.
- `posthog-pp-cli vision scanners-create` — CRUD for Replay Vision scanners.
- `posthog-pp-cli vision scanners-creators-retrieve` — Distinct creators across the team's scanners — feeds the `Created by` filter dropdown.
- `posthog-pp-cli vision scanners-destroy` — CRUD for Replay Vision scanners.
- `posthog-pp-cli vision scanners-estimate-create` — Estimate the observation volume a proposed scanner would generate, for the pre-save cost preview.
- `posthog-pp-cli vision scanners-impact-retrieve` — Affected sessions and users for this scanner over the trailing window.
- `posthog-pp-cli vision scanners-inline-scan-create` — Scan named sessions against a prompt without saving a scanner first, for one-off questions.
- `posthog-pp-cli vision scanners-list` — CRUD for Replay Vision scanners.
- `posthog-pp-cli vision scanners-observations-create-task-create` — Create a PostHog Task from this observation's finding so it can be triaged and fixed.
- `posthog-pp-cli vision scanners-observations-label-create` — Set or update the observation's shared label: whether the scanner scored the session correctly
- `posthog-pp-cli vision scanners-observations-label-destroy` — Remove the observation's shared label. Requires editor access to the scanner.
- `posthog-pp-cli vision scanners-observations-list` — Read-only access to observations produced by a scanner.
- `posthog-pp-cli vision scanners-observations-retrieve` — Retrieve one observation.
- `posthog-pp-cli vision scanners-observations-retry-create` — Delete a failed or ineligible observation and re-run its scanner on the same recording.
- `posthog-pp-cli vision scanners-observations-stats-retrieve` — Aggregate counts and per-scanner-type distributions over the filtered observation set.
- `posthog-pp-cli vision scanners-observe-create` — Apply this scanner to one specific session, on demand. Returns 202 with the workflow handle.
- `posthog-pp-cli vision scanners-partial-update` — CRUD for Replay Vision scanners.
- `posthog-pp-cli vision scanners-prompt-suggestions-apply-create` — Apply this suggestion
- `posthog-pp-cli vision scanners-prompt-suggestions-current-retrieve` — The scanner's newest prompt suggestion plus whether it is stale (the ratings changed since it was generated)
- `posthog-pp-cli vision scanners-prompt-suggestions-dismiss-create` — Dismiss this suggestion without applying it. Only the current pending suggestion can be dismissed.
- `posthog-pp-cli vision scanners-prompt-suggestions-evaluate-create` — Test this suggestion before applying it
- `posthog-pp-cli vision scanners-prompt-suggestions-generate-create` — Generate a fresh prompt suggestion from the team's current ratings.
- `posthog-pp-cli vision scanners-prompt-suggestions-list` — AI prompt-rewrite suggestions for a scanner, generated from the team's thumbs up/down ratings.
- `posthog-pp-cli vision scanners-retrieve` — CRUD for Replay Vision scanners.
- `posthog-pp-cli vision scanners-stats-retrieve` — Team-wide scanner counts — independent of list filters, so the overview stays stable.
- `posthog-pp-cli vision scanners-suggest-tags-create` — Suggest classifier tags grounded in the scanner's own observations and the org's product data.

**visual-review** — Manage visual review

- `posthog-pp-cli visual-review repos-baselines-retrieve` — Snapshots overview for a repo
- `posthog-pp-cli visual-review repos-create` — Create a new repo.
- `posthog-pp-cli visual-review repos-list` — List all projects for the team.
- `posthog-pp-cli visual-review repos-partial-update` — Update a repo's settings.
- `posthog-pp-cli visual-review repos-quarantine-create` — Quarantine a snapshot identifier for a specific run type.
- `posthog-pp-cli visual-review repos-quarantine-expire-create` — Expire all active quarantine entries for an identifier.
- `posthog-pp-cli visual-review repos-quarantine-list` — List quarantined identifiers. Without filter: active only. With identifier: full history.
- `posthog-pp-cli visual-review repos-retrieve` — Get a repo by ID.
- `posthog-pp-cli visual-review repos-runs-counts-retrieve` — Review state counts for runs in this repo.
- `posthog-pp-cli visual-review repos-runs-list` — List runs in this repo, optionally filtered by review state and free-text search.
- `posthog-pp-cli visual-review repos-snapshots-list` — Deduped baseline timeline for a snapshot identity. Newest first.
- `posthog-pp-cli visual-review repos-thumbnails-retrieve` — Serve a snapshot thumbnail by identifier. Returns WebP with ETag caching.
- `posthog-pp-cli visual-review runs-add-snapshots-create` — Add a batch of snapshots to a pending run (shard-based flow).
- `posthog-pp-cli visual-review runs-approve-create` — Mark snapshots reviewed (DB only). Records the per-snapshot 'Accept change' decision.
- `posthog-pp-cli visual-review runs-complete-create` — Complete a run: detect removals, verify uploads, trigger diff processing.
- `posthog-pp-cli visual-review runs-counts-retrieve` — Review state counts for the runs list.
- `posthog-pp-cli visual-review runs-create` — Create a new run from a CI manifest.
- `posthog-pp-cli visual-review runs-finalize-create` — Finalize a fully-reviewed run: commit the approved baseline and green the gate.
- `posthog-pp-cli visual-review runs-list` — List runs for the team, optionally filtered by review state, PR number, commit SHA, branch, or free-text search.
- `posthog-pp-cli visual-review runs-recompute-create` — Re-evaluate quarantine and counts, update commit status, and optionally rerun the CI job.
- `posthog-pp-cli visual-review runs-retrieve` — Get run status and summary.
- `posthog-pp-cli visual-review runs-snapshot-history-list` — Recent change history for a snapshot identifier across runs.
- `posthog-pp-cli visual-review runs-snapshots-list` — Get a run's snapshots with diff results, excluding quarantined ones by default.
- `posthog-pp-cli visual-review runs-tolerate-create` — Mark a changed snapshot as a known tolerated alternate.
- `posthog-pp-cli visual-review runs-tolerated-hashes-list` — List known tolerated hashes for a snapshot identifier.

**warehouse-column-annotations** — Manage warehouse column annotations

- `posthog-pp-cli warehouse-column-annotations create` — Read and edit semantic descriptions of warehouse tables and columns surfaced to the AI agent.
- `posthog-pp-cli warehouse-column-annotations destroy` — Read and edit semantic descriptions of warehouse tables and columns surfaced to the AI agent.
- `posthog-pp-cli warehouse-column-annotations list` — Read and edit semantic descriptions of warehouse tables and columns surfaced to the AI agent.
- `posthog-pp-cli warehouse-column-annotations partial-update` — Read and edit semantic descriptions of warehouse tables and columns surfaced to the AI agent.
- `posthog-pp-cli warehouse-column-annotations retrieve` — Read and edit semantic descriptions of warehouse tables and columns surfaced to the AI agent.
- `posthog-pp-cli warehouse-column-annotations update` — Read and edit semantic descriptions of warehouse tables and columns surfaced to the AI agent.

**warehouse-column-statistics** — Manage warehouse column statistics

- `posthog-pp-cli warehouse-column-statistics list` — Read per-column data statistics (null fraction, min/max, row count) for warehouse tables.
- `posthog-pp-cli warehouse-column-statistics retrieve` — Read per-column data statistics (null fraction, min/max, row count) for warehouse tables.

**warehouse-dag** — Manage warehouse dag

- `posthog-pp-cli warehouse-dag <project_id>` — Return this team's DAG as a set of edges and nodes

**warehouse-expressions** — Manage warehouse expressions

- `posthog-pp-cli warehouse-expressions create` — Create, read, update and delete saved HogQL expressions that appear as virtual fields on tables.
- `posthog-pp-cli warehouse-expressions destroy` — Create, read, update and delete saved HogQL expressions that appear as virtual fields on tables.
- `posthog-pp-cli warehouse-expressions list` — Create, read, update and delete saved HogQL expressions that appear as virtual fields on tables.
- `posthog-pp-cli warehouse-expressions partial-update` — Create, read, update and delete saved HogQL expressions that appear as virtual fields on tables.
- `posthog-pp-cli warehouse-expressions retrieve` — Create, read, update and delete saved HogQL expressions that appear as virtual fields on tables.
- `posthog-pp-cli warehouse-expressions update` — Create, read, update and delete saved HogQL expressions that appear as virtual fields on tables.

**warehouse-model-paths** — Manage warehouse model paths

- `posthog-pp-cli warehouse-model-paths list` — List
- `posthog-pp-cli warehouse-model-paths retrieve` — Retrieve

**warehouse-saved-queries** — Manage warehouse saved queries

- `posthog-pp-cli warehouse-saved-queries create` — Create, Read, Update and Delete Warehouse Tables.
- `posthog-pp-cli warehouse-saved-queries destroy` — Create, Read, Update and Delete Warehouse Tables.
- `posthog-pp-cli warehouse-saved-queries list` — Create, Read, Update and Delete Warehouse Tables.
- `posthog-pp-cli warehouse-saved-queries partial-update` — Create, Read, Update and Delete Warehouse Tables.
- `posthog-pp-cli warehouse-saved-queries resume-schedules-create` — Resume paused materialization schedules for multiple matviews.
- `posthog-pp-cli warehouse-saved-queries retrieve` — Create, Read, Update and Delete Warehouse Tables.
- `posthog-pp-cli warehouse-saved-queries update` — Create, Read, Update and Delete Warehouse Tables.

**warehouse-saved-query-folders** — Manage warehouse saved query folders

- `posthog-pp-cli warehouse-saved-query-folders create` — Create
- `posthog-pp-cli warehouse-saved-query-folders destroy` — Destroy
- `posthog-pp-cli warehouse-saved-query-folders list` — List
- `posthog-pp-cli warehouse-saved-query-folders partial-update` — Partial update
- `posthog-pp-cli warehouse-saved-query-folders retrieve` — Retrieve

**warehouse-tables** — Manage warehouse tables

- `posthog-pp-cli warehouse-tables create` — Create, Read, Update and Delete Warehouse Tables.
- `posthog-pp-cli warehouse-tables create-from-upload-create` — Turn a previously uploaded file into a self-managed warehouse table.
- `posthog-pp-cli warehouse-tables destroy` — Create, Read, Update and Delete Warehouse Tables.
- `posthog-pp-cli warehouse-tables file-create` — Create, Read, Update and Delete Warehouse Tables.
- `posthog-pp-cli warehouse-tables list` — Create, Read, Update and Delete Warehouse Tables.
- `posthog-pp-cli warehouse-tables partial-update` — Create, Read, Update and Delete Warehouse Tables.
- `posthog-pp-cli warehouse-tables retrieve` — Create, Read, Update and Delete Warehouse Tables.
- `posthog-pp-cli warehouse-tables update` — Create, Read, Update and Delete Warehouse Tables.
- `posthog-pp-cli warehouse-tables upload-file-create` — Store an uploaded file in object storage so a self-managed table can be created from it.

**warehouse-view-link** — Manage warehouse view link

- `posthog-pp-cli warehouse-view-link create` — Create, Read, Update and Delete View Columns.
- `posthog-pp-cli warehouse-view-link destroy` — Create, Read, Update and Delete View Columns.
- `posthog-pp-cli warehouse-view-link list` — Create, Read, Update and Delete View Columns.
- `posthog-pp-cli warehouse-view-link partial-update` — Create, Read, Update and Delete View Columns.
- `posthog-pp-cli warehouse-view-link retrieve` — Create, Read, Update and Delete View Columns.
- `posthog-pp-cli warehouse-view-link update` — Create, Read, Update and Delete View Columns.
- `posthog-pp-cli warehouse-view-link validate-create` — Create, Read, Update and Delete View Columns.

**warehouse-view-links** — Manage warehouse view links

- `posthog-pp-cli warehouse-view-links create` — Create, Read, Update and Delete View Columns.
- `posthog-pp-cli warehouse-view-links destroy` — Create, Read, Update and Delete View Columns.
- `posthog-pp-cli warehouse-view-links list` — Create, Read, Update and Delete View Columns.
- `posthog-pp-cli warehouse-view-links partial-update` — Create, Read, Update and Delete View Columns.
- `posthog-pp-cli warehouse-view-links retrieve` — Create, Read, Update and Delete View Columns.
- `posthog-pp-cli warehouse-view-links update` — Create, Read, Update and Delete View Columns.
- `posthog-pp-cli warehouse-view-links validate-create` — Create, Read, Update and Delete View Columns.

**web-analytics** — Manage web analytics

- `posthog-pp-cli web-analytics recap` — The 'Wrapped'-style weekly recap: everything in the weekly digest (visitors, pageviews, sessions, bounce rate
- `posthog-pp-cli web-analytics weekly-digest` — Summarizes a project's web analytics over a lookback window (default 7 days): unique visitors, pageviews, sessions

**web-analytics-achievements** — Manage web analytics achievements

- `posthog-pp-cli web-analytics-achievements acknowledge-celebration` — Clears a pending celebration for the given track and stage once the client has shown it, so it isn't celebrated again.
- `posthog-pp-cli web-analytics-achievements overview` — Returns the achievement track definitions (thresholds resolved for the requesting user's streak-cadence arm)
- `posthog-pp-cli web-analytics-achievements preferences` — Returns the requesting user's per-project Web analytics achievements preferences.
- `posthog-pp-cli web-analytics-achievements record-interaction` — Idempotently increments the requesting user's first-party counter for an in-product Web analytics interaction (slicing
- `posthog-pp-cli web-analytics-achievements record-visit` — Idempotently records that the requesting user opened Web analytics today (team-local date)
- `posthog-pp-cli web-analytics-achievements update-preferences` — Sets the requesting user's per-project Web analytics achievements preferences.

**web-analytics-path-cleaning-suggestions** — Manage web analytics path cleaning suggestions

- `posthog-pp-cli web-analytics-path-cleaning-suggestions <project_id>` — Samples the team's recent paths, asks the LLM for cleaning rules, validates them against the real paths

**web-experiments** — Manage web experiments

- `posthog-pp-cli web-experiments create` — Create
- `posthog-pp-cli web-experiments destroy` — Destroy
- `posthog-pp-cli web-experiments list` — List
- `posthog-pp-cli web-experiments partial-update` — Partial update
- `posthog-pp-cli web-experiments retrieve` — Retrieve
- `posthog-pp-cli web-experiments update` — Update

**web-vitals** — Manage web vitals

- `posthog-pp-cli web-vitals <project_id>` — Get web vitals for a specific pathname. Toolbar accesses this via OAuth (handled by TeamAndOrgViewSetMixin.

**wizard** — Manage wizard

- `posthog-pp-cli wizard sessions-create` — Upsert a wizard session.
- `posthog-pp-cli wizard sessions-latest-retrieve` — Return the single most-recent wizard session for a workflow (and optional skill), or 204 if none exists.
- `posthog-pp-cli wizard sessions-list` — List wizard sessions for the project, ordered by started_at desc. This should only be called by the PostHog Wizard.
- `posthog-pp-cli wizard sessions-retrieve` — Retrieve a single wizard session by its session_id.
- `posthog-pp-cli wizard sessions-stream-retrieve` — Server-Sent Events stream of wizard session updates for a (workflow_id, skill_id) pair.


### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
posthog-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

## Auth Setup

Run `posthog-pp-cli auth setup` for the URL and steps to obtain a token (add `--launch` to open the URL). Then store it:

```bash
posthog-pp-cli auth set-token YOUR_TOKEN_HERE
```

Or set `POSTHOG_PERSONAL_APIKEY_AUTH` as an environment variable.

Run `posthog-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  posthog-pp-cli account-notes mock-value --agent --select id,name,status
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

- Use `--home <dir>` for one invocation, or set `POSTHOG_HOME=<dir>` to relocate all four path kinds under one root.
- Use per-kind env vars only when a specific kind must diverge: `POSTHOG_CONFIG_DIR`, `POSTHOG_DATA_DIR`, `POSTHOG_STATE_DIR`, `POSTHOG_CACHE_DIR`.
- Resolution order is per-kind env var, `--home`, `POSTHOG_HOME`, XDG (`XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME`), then platform defaults.
- `config` contains settings like `config.toml` and profiles. `data` contains `credentials.toml`, `data.db`, cookies, and auth sidecars. `state` contains persisted queries, jobs, and `teach.log`. `cache` contains regenerable HTTP/cache files.
- Stored secrets live in `credentials.toml` under the data dir. Existing legacy `config.toml` secrets are read for compatibility and leave `config.toml` on the first auth write.
- Run `posthog-pp-cli doctor --fail-on warn` to surface path and credential-location warnings. `agent-context` exposes a schema v4 `paths` block for agents that need the resolved dirs.
- For MCP, pass relocation through the MCP host config. The MCP binary does not inherit CLI flags:

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

Fleet precedence: an inherited per-kind env var overrides an explicit `--home` for that kind. Use `POSTHOG_HOME` or per-kind vars as durable fleet levers, and use `--home` only for a single invocation. Relocation is not reversible by unsetting env vars; move files manually before clearing `POSTHOG_HOME`, or `doctor` will not find credentials left under the former root.

## Automatic learning

This CLI ships a self-capturing learning loop. The CLI does its own bookkeeping: every invocation is journaled locally, a failed flag followed by a corrected retry auto-derives a `flag_alias` candidate, and a `teach` on a query family without a playbook auto-synthesizes a `playbook_candidate` from the session's journal. Your job is judgment only: `recall` first, act on surfaced candidates, `teach` the final answer, `playbook amend` when you observe a correction. You never record failures by hand.

### Step 1: `recall` before any discovery

Before list/search/drill commands on a new user question, run:

```bash
posthog-pp-cli recall "<user's question>" --agent
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
      "next_action": ["<trial command>", "posthog-pp-cli learnings confirm 12"] }
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
       materially more, record the divergence via `posthog-pp-cli playbook amend`
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

Candidate judgment details: `learnings confirm <id>` prints the candidate's full payload before materializing it - check that the printed payload matches the behavior you verified. `learnings reject <id>` tombstones the derivation signature so the same candidate does not resurface. The envelope carries only the few candidates worth acting on now; `posthog-pp-cli learnings candidates` lists the full open set.

Graceful degradation: if `learnings confirm` is an unknown command, you are driving an older binary - ignore the candidates guidance and follow the rest of the protocol.

### Step 3: always read `warnings`

- `low_confidence`: row exists at `confidence<2`. Treat as a hint, not a skip-discovery hit.
- `resource_not_in_store`: the local store doesn't have the resource the learning points at. The match validator couldn't classify entities — direct-fetch and re-evaluate.
- `cross_alias_match` (per-result): the row was taught under a different alias and matched the live query's canonical via `entity_lookups` (e.g., a "USA" teach satisfying a "United States" recall). Trust the resource_id.
- `similar_shape_different_entity:<canonical>` (top-level): a structurally matching row exists but its canonical entity differs from the live query's. Treated as cold start; the warning carries the conflicting canonical as a hint, but the row is NOT promoted into Results.
- `ambiguous_alias` (top-level): a single query entity resolved to multiple canonicals (e.g., "Cards" → Arizona Cardinals + St. Louis Cardinals). Surface the ambiguity from context before committing to a resource.
- `candidates_present` (top-level): the envelope carries a `candidates` section. Handle it via the candidates branch in Step 2 before anything else.
- `lookup_refresh_available` (top-level): an entity in the query has no lookup row yet, but synced data could provide one. Run `posthog-pp-cli sync` to refresh entity lookups.
- Top-level `no_learnings_for_query_family`: the table had no rows above the Jaccard floor. Pure cold start.

### Step 4: `teach &` after finalizing your response - always

Teaching is unconditional. After resolving a query the store could not answer, background-teach the final resource mapping - no call-count threshold, no judging whether it was "worth" learning. The teach is the anchor of the loop: it triggers playbook synthesis for a family without a playbook, and same-referent phrasings fold into one family so near-duplicate teaches do not fragment the store. Fire it after assembling your user-facing response but BEFORE emitting it, with a shell `&` so the call returns immediately:

```bash
posthog-pp-cli teach --query "<user's question>" --resource-type <type> --resource <id1> --resource <id2>
# (append shell `&` to background it)
```

Silent on success. Errors only land in `teach.log` under the resolved state dir. Teach the **most specific** resource - if the user asked a broad question and you walked through parent records to find the specific answer, teach the leaf id, not the parent. The CLI uses seeded `entity_lookups` for cross-alias resolution at recall time, so a teach under one alias (e.g., "Niners") satisfies future queries under another alias (e.g., "49ers", "San Francisco") automatically.

PII rule: teach the structural question with identifiers stripped - never include names, emails, phone numbers, account ids, or other personal identifiers in taught queries or notes. The CLI scans teach queries for obvious email/phone shapes and warns, but does not block; strip before teaching rather than relying on the warning.

### Step 5: playbooks - optional flags, automatic synthesis

You do not need to decide whether a session "deserves" a playbook: a teach on a family without one auto-synthesizes a `playbook_candidate` from the session's journal, and the next session judges it via confirm/reject. Attach explicit playbook flags only when you already hold choreography worth recording verbatim - workarounds the CLI didn't surface (silently-dropped flags, undocumented params, pagination tricks, payload gotchas). Prefer the **integrated one-call form** - record the resource learning and the playbook in the same `teach` invocation:

```bash
# Common case: record both the resource learning AND the playbook in one call.
posthog-pp-cli teach \
  --query "<user's question>" \
  --resource <id> \
  --playbook-file ~/playbooks/<shape>.json \
  --playbook-notes-file ~/playbooks/<shape>-notes.md
# (append shell `&` to background it)

# Alternate: playbook-only (no resource to record alongside).
posthog-pp-cli teach-playbook \
  --query "<user's question>" \
  --playbook-file ~/playbooks/<shape>.json \
  --notes-file ~/playbooks/<shape>-notes.md
```

Playbook files are JSON with `steps`, `entity_slots`, `expected_tool_calls`. Notes files are markdown carrying the gotchas verbatim. File-free callers (MCP-only agents) pass the same content inline: `--playbook-json` and `--playbook-notes` on the integrated `teach` form, `--playbook-json` and `--notes` on `teach-playbook`. On the integrated `teach` form, the playbook flags are optional - omit them entirely for a resource-only teach. On the standalone `teach-playbook` form, at least one of the playbook and notes flags must be set; both empty is rejected. Playbooks are keyed on the structural query family (entities stripped) so a recipe taught from one entity-shaped query applies to every other query of the same shape, with `slots_resolved` binding the live query's canonical at recall time.

When you DO find a playbook on a future recall, treat it as ground truth: replay the steps with `slots_resolved` substitutions, skip the discovery that the choreography already documents, and read `notes` before any step.

### Step 6: `playbook amend &` when your debug response identifies a correction

If your debug-protocol response identifies a concrete correction the notes or playbook should know — a workaround, an undocumented endpoint shape, a stale field name, observed schema drift, an empty-payload fallback — fire `playbook amend` BEFORE emitting your user-facing response. Same fire-and-forget posture as `teach`.

```bash
posthog-pp-cli playbook amend \
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

`posthog-pp-cli learnings stats` reports recall hit rate, teach-to-reuse, playbook resolution rate, and candidate confirm/reject counts from the local `learn_events` table. Rates are null until they have a denominator; everything stays on this machine. Use it to check whether the loop is earning its keep for this CLI.

### Disabling learning

- `--no-learn` on a single command short-circuits both `recall` and the `teach` write path. Use for deterministic agent flows or tests that must not be affected by accumulated learnings.
- `POSTHOG_NO_LEARN=true` in the environment globally disables the pipeline.

## Agent Feedback

When you (or the agent) notice something off about this CLI, record it:

```
posthog-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
posthog-pp-cli feedback --stdin < notes.txt
posthog-pp-cli feedback list --json --limit 10
```

Entries are stored locally as `feedback.jsonl` under the resolved data dir. They are never POSTed unless `POSTHOG_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `POSTHOG_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

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
posthog-pp-cli profile save briefing --json
posthog-pp-cli --profile briefing account-notes mock-value
posthog-pp-cli profile list --json
posthog-pp-cli profile show briefing
posthog-pp-cli profile delete briefing --yes
```

Explicit flags always win over profile values; profile values win over defaults. `agent-context` lists all available profiles under `available_profiles` so introspecting agents discover them at runtime.

## Async Jobs

For endpoints that submit long-running work, the generator detects the submit-then-poll pattern (a `job_id`/`task_id`/`operation_id` field in the response plus a sibling status endpoint) and wires up three extra flags on the submitting command:

| Flag | Purpose |
|------|---------|
| `--wait` | Block until the job reaches a terminal status instead of returning the job ID immediately |
| `--wait-timeout` | Maximum wait duration (default 10m, 0 means no timeout) |
| `--wait-interval` | Initial poll interval (default 2s; grows with exponential backoff up to 30s) |

Use async submission without `--wait` when you want to fire-and-forget; use `--wait` when you want one command to return the finished artifact.

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

1. **Empty, `help`, or `--help`** → show `posthog-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → see Prerequisites above
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## MCP Server Installation

1. Install the MCP server:
   ```bash
   go install github.com/mvanhorn/printing-press-library/library/monitoring/posthog/cmd/posthog-pp-mcp@latest
   ```
2. Register with Claude Code:
   ```bash
   claude mcp add posthog-pp-mcp -- posthog-pp-mcp
   ```
3. Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which posthog-pp-cli`
   If not found, offer to install (see Prerequisites at the top of this skill).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   posthog-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `posthog-pp-cli <command> --help`.
