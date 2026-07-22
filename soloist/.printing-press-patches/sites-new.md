# sites new

Hand-authored `internal/cli/sites_new.go` (+ one `AddCommand` line in
`sites.go`). Also adds `newSoloWebClient`, a shared helper for calling the
soloist.ai `/api/*` server routes with the stored bearer token (auto-refreshed)
and a Referer/Content-Type header.

`sites new --from <siteId> [--handle H] [--business-name N]` creates a new
draft by cloning an existing site's `websiteSettings` (decoded via the codec,
re-keyed with a fresh UUID) and POSTing it to `/api/websites/create`. Cloning a
real site guarantees valid, publishable content. Verified live (cloned a real
site, status 200).

## Why AI generation + domains connect are NOT here

- **AI website generation:** `/api/ai` is a single-schema prompt call; the
  designer's "create a website" onboarding orchestrates ~15 of these and
  assembles section props client-side. More importantly, the **deployed
  `/api/ai` contract differs from the repo source** (prod is version-skewed
  from main — a bare `{docId, prompt}` returns HTTP 400), so a repo-derived
  implementation is unreliable. Correct implementation needs a live capture of
  the real deployed request shape.
- **domains connect:** connecting an external domain is a multi-step
  server-side ownership-verification + certificate flow (Entri / DNS TXT token
  / verification polling), not a plain doc write. Needs a live capture of the
  real Manage-Domains flow.

Both are deferred rather than shipped guessed/broken. `sites new --from` is the
reliable "create a new website" path today.
