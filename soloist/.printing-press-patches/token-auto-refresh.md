# Token auto-refresh (+ Referer fix)

Hand-authored `internal/cli/auth_refresh.go`, plus:
- one call inserted in `rootFlags.newClient` (`internal/cli/root.go`):
  `maybeRefreshToken(cfg, f.timeout)` right after `config.Load`, before
  `client.New` — so the fresh token is what the client uses.
- `auth set-token` grew optional `--refresh-token` / `--expires-in`
  (`internal/cli/auth.go`) so the browser-capture path can store a refresh
  token + expiry.
- `auth login` (`internal/cli/auth_login.go`) now sends its requests via
  `PostWithHeaders` with a `Referer` header.

## How refresh works

`maybeRefreshToken` runs before every authenticated request. If a refresh
token is stored and the ID token is expired (or within 60s of expiry), it
POSTs `securetoken.googleapis.com/v1/token` (form-encoded,
`grant_type=refresh_token`) and persists the new ID token + refresh token +
expiry via `SaveTokens`. Best-effort: any failure leaves the stored token in
place so a genuine 401 still surfaces.

## The Referer requirement (important)

The Firebase **web** API key is HTTP-referrer-restricted to the app origin.
A CLI request has no referer and gets `403 API_KEY_HTTP_REFERRER_BLOCKED` from
both `securetoken` and `identitytoolkit`. Every CLI call that uses the web key
therefore sets `Referer: https://soloist.ai/`. Verified: refresh advances the
stored expiry ~1h and the next request authenticates with the new token.

Override the key/origin for other envs with `SOLOIST_FIREBASE_API_KEY` /
`--firebase-key` (nonprod).
