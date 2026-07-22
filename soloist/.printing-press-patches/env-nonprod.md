# Environment selection (prod / nonprod)

Hand-authored `internal/cli/env.go` (+ call-site swaps in `sites_publish.go`,
`auth_login.go`, `auth_refresh.go` from the hardcoded prod origin to
`soloistWebBase()`).

soloist.ai = prod (`moz-ocho-solo-prod`); main.soloist.ai = nonprod
(`moz-ocho-solo-nonprod`). The generated command paths hardcode the prod
project, so rather than regenerate them, a `registerClientHook` installs a
`projectRewriteTransport` (a `RoundTripper` on `client.HTTPClient`) that swaps
the prod project id for the target in every outgoing Firestore URL — covering
generated and hand-written commands alike, no path edits.

Enable nonprod via env:
- `SOLOIST_ENV=nonprod` (or `SOLOIST_PROJECT=<id>` to set the project directly)
- `SOLOIST_WEB_BASE_URL=https://main.soloist.ai` (publish + Referer origin)
- `SOLOIST_FIREBASE_API_KEY=<nonprod web key>` (login/refresh)

Verified: with `SOLOIST_ENV=nonprod` a request hits `moz-ocho-solo-nonprod`
(prod token → 403) vs prod (`moz-ocho-solo-prod` → 404), proving the rewrite.
