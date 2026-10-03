# submit

Directory-submission copilot and tracker: render the copy kit, track each (site x directory) submission, verify listings, and submit to the one scriptable directory.

The CLI never posts to third parties — except Launching Next, a plain form POST with a static math check. Every other directory needs an account, OAuth, CAPTCHA, checkout or editorial review, so those stay manual: the CLI prepares the copy, tracks the status and checks the listing afterwards.

## Status

Tested against local fake servers only (`make e2e`). The Launching Next form shape was verified on 2026-10-03; the command re-reads the form fresh per submit and aborts when the shape changed. Use `--dry-run` first.

## Install

```bash
git clone https://github.com/pooriaarab/clis.git
cd clis/submit
make build
```

The binary is `bin/submit`. You need [Go](https://go.dev/dl/) 1.22 or later.

## The sites file

Your sites live in a YAML file you supply. It holds your names, copy and contact, so it is never committed here. Every site needs:

| Field | Use |
|---|---|
| `id` | Short unique id, used by `track` and `--site` |
| `name` | Startup name (max 100 chars) |
| `url` | Absolute https URL |
| `tagline` | 5-8 words; defaults the headline |
| `one_liner` | One sentence |
| `short` | Max 280 chars |
| `medium` | Max 500 chars |
| `headline` | Directory headline, max 100 chars (default: tagline) |
| `full2200` | Full description, max 2500 chars |
| `tags` | 5-10 tags |
| `funding` | `side`, `bootstrapped`, `funded` or `none` (default `none`) |
| `marketing_budget` | One of `$0`, `Less than $1,000`, `$1,000 – $5,000`, `$5,000 – $15,000`, `More than $15,000` (default `$0`) |
| `founder` | Your name (sent as the Launching Next contact) |
| `email` | Contact email (sent as the Launching Next contact) |

Example:

```yaml
sites:
  - id: widgets
    name: Widget Directory
    url: https://widgets.example.com
    tagline: Find a widget maker near you
    one_liner: Widget Directory lists 120 widget makers with addresses and reviews.
    short: Widget Directory lists 120 widget makers with addresses, hours and reviews.
    medium: Widget Directory lists 120 widget makers with addresses, hours and verified reviews.
    full2200: Widget Directory is a free directory of 120 independent widget makers.
    tags: [widgets, makers, directory, shopping, local]
    funding: bootstrapped
    founder: Jane Founder
    email: jane@example.com
```

Resolution: `--sites` flag, `SUBMIT_SITES` env, `./sites.yaml`.

## Commands

| Command | What it does |
|---|---|
| `kit --sites sites.yaml --out kit/` | Render `<id>.md` + `<id>.json` per site |
| `track init` | Create the tracker store (`~/.config/submit/store.json`) |
| `track set --site ID --directory DIR --status STATUS` | Set one row; refuses past 10 submissions/day |
| `track list [--site ID] [--directory DIR] [--status STATUS]` | List rows |
| `directories list [--wave 1-5\|geo\|company]` | Show the built-in 32 ranked directories |
| `check --site SITE --directory DIR [--url URL]` | Fetch the listing, report follow vs nofollow, index hint |
| `launching-next --sites sites.yaml [--site ID] [--gap 25s]` | Post to Launching Next, the one scriptable target |
| `doctor [--sites sites.yaml]` | Config + store health |

Every command accepts `--json` and `--dry-run`. With `--dry-run` the CLI writes nothing and posts nothing.

### Tracker

Statuses are `draft`, `submitted`, `live`, `rejected`. `track set` also takes `--site-url` (stored for `check`), `--listing-url`, `--date YYYY-MM-DD` (default today) and `--note`. Empty fields keep their previous value, so a bare `--status` flip keeps the URLs.

The daily limit counts non-draft rows per date: the 11th submission dated one day is refused with a clear error. Drafts never count.

### Listing checks

`check` reads the listing URL from the tracker (or `--url`), fetches it, and reports the HTTP status, the redirect chain, whether the page links to the site, and whether that link is follow or nofollow (link-level `rel` or page-level meta robots). The index hint is a `site:` query to confirm by hand — the CLI cannot ask Google. `--site` accepts the site id or the site URL.

### Launching Next

The only directory the CLI posts to. The form (verified 2026-10-03) takes the startup name, URL, headline, full description (2500 chars max), tags, funding, marketing budget, contact name/email, a submit button and a `What is A+B?` math check. The newsletter opt-in is never sent.

- `--dry-run` fetches the form once, solves the math check, and shows the exact POST body per site without posting.
- A real run re-reads the form fresh per site and aborts the whole run when the shape changed (missing fields, no math question): a redesign never gets a blind POST.
- Posts wait `--gap` apart, minimum 25s on the real site. The floor is lifted only when `SUBMIT_LAUNCHINGNEXT_BASE` points the command at another server (tests).
- Success is a heuristic: HTTP 200 plus a thank-you/received/under-review page that does not redisplay the form. Accepted posts are recorded as `submitted` in the tracker; anything else is reported as failed and left untracked for a hand check.
- The run refuses upfront when it would pass the 10/day limit.

Never add other auto-submit targets without a new brief.

## Environment

| Variable | Use |
|---|---|
| `SUBMIT_SITES` | Sites YAML path (default `./sites.yaml`) |
| `SUBMIT_CONFIG_DIR` | Config directory (default `~/.config/submit`) |
| `SUBMIT_LAUNCHINGNEXT_BASE` | Override the Launching Next host (tests only) |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success. |
| 1 | A request or a step failed (fetch error, changed form, failed POST). |
| 2 | Usage error: bad flags, bad sites file, unknown site or directory, daily limit. |

## Test

```bash
make e2e
```

The target runs `go vet`, runs every E2E test (real binary against local fake servers, no real network) and writes `e2e-report.txt`. The report lists each test and one full launching-next dry run. Two runs give the same file.
