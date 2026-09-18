# MERX CLI

Public MERX lists. `search` parses HTML; there is no JSON API. Authenticated
access uses SAML SSO via `merx login`, which reads `MERX_USERNAME` /
`MERX_PASSWORD`.

Commands: `doctor`, `search`, `login`, `logout`, `auth status`. Cache is
`$XDG_CACHE_HOME/merx` or `~/.cache/merx`; the session cookie jar lives there too.
Queries that report more than 1,000 results cannot be fully retrieved.

Run `gofmt`, `go vet`, and `go build ./...` before a PR. Stay inside `merx/`.
