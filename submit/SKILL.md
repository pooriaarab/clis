---
name: submit
description: "Directory-submission copilot and tracker: render the copy kit, track (site x directory) status, verify listings, submit to Launching Next. Trigger phrases: `render the submission kit`, `track a directory submission`, `check a listing`, `submit to launching next`, `submission status`."
author: "Pooria Arab"
license: "Apache-2.0"
argument-hint: "<command> [args]"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - submit
---

# submit

Use this skill to prepare directory submissions, track their status and verify listings. The binary is `submit`. Build it with `make build` in the `submit/` directory.

Run `submit --help` for the full command list. Every command accepts `--json` and `--dry-run`.

## Before you run anything

Run `submit doctor --sites sites.yaml`. If it reports the sites file missing or invalid, stop and ask the owner to fix it. The sites file holds the owner's site names, copy and contact — never commit it, never invent its contents.

## Render the copy kit

```bash
submit kit --sites sites.yaml --out kit/ --dry-run
submit kit --sites sites.yaml --out kit/
```

Writes `<id>.md` + `<id>.json` per site: tagline, one-liner, short, medium, headline, full description, tags, founder, email.

## Track submissions

```bash
submit track init
submit track set --site widgets --directory crunchbase --status draft
submit track set --site widgets --directory crunchbase --status submitted --listing-url https://...
submit track list --status submitted
```

Statuses: `draft`, `submitted`, `live`, `rejected`. `track set` refuses the 11th submission dated one day (max 10/day). See the 32 ranked directories with `submit directories list` (filter: `--wave 1-5|geo|company`).

## Check a listing

```bash
submit check --site widgets --directory crunchbase
```

Fetches the listing URL from the tracker and reports the HTTP status, redirects, whether the page links to the site (follow vs nofollow), and a `site:` query to confirm indexing by hand.

## Launching Next (the one scriptable target)

Always dry-run first, then run for real:

```bash
submit launching-next --sites sites.yaml --dry-run
submit launching-next --sites sites.yaml --json
```

This is the only directory the CLI posts to. It re-reads the form fresh per site, aborts when the form shape changed, waits 25s between posts, and records acceptances in the tracker. Narrow with repeatable `--site <id>`.

## Rules

- Never post to any other directory. Never add auto-submit targets without a new brief.
- Never commit the sites file or paste its contents into the repo.
- Never lower `--gap` below 25s on the real site.
- The CLI was tested against fake servers only. Report any difference from the live form.
