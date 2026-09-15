# Doctor token capability probe

Modified `internal/cli/doctor.go` so `doctor` performs an authenticated
`GET /me/adaccounts` probe when credentials are present.

Why: `doctor` previously reported a wall of green `OK`s for a Conversions API
token that cannot manage campaigns. The only caveat lived in a grey `INFO`
Credentials line and was easy to miss.

On regen: re-apply these changes to `internal/cli/doctor.go`:

1. Import `encoding/json`.
2. Add helpers near `looksLikeDoctorInterstitial`:
   - `extractMetaErrorCode` parses a Meta Graph error envelope.
   - `countAdAccounts` counts objects in `/me/adaccounts` `data` array.
   - `doctorStatus` maps a prefixed report string to its indicator and strips
     the prefix for display, so `WARN Auth: configured, not verified` renders
     without a duplicated prefix.
3. Replace the "present, not verified" credentials block with a probe of
   `/me/adaccounts` using `c.GetNoCache` and classify the result:
   - HTTP 200 with non-empty `data`: `OK proven for campaign management: N ad account(s)`
   - HTTP 200 with empty `data`: `WARN valid token, no ad account assigned; cannot manage campaigns`
   - Meta error 190: `FAIL token invalid (error 190)`
   - Meta error 100 / 200-series: `WARN valid token, lacks permission (code N); cannot manage campaigns`
   - network/client/init/base_url failures: `WARN present, not verified (reason)`
4. Update the `auth` verdict to `WARN configured, ...` when the token has not
   been proven or cannot manage campaigns, and `OK configured and verified`
   only when `/me/adaccounts` returns accounts.
5. Change the non-2xx `/` reachability message from `reachable (HTTP N at /)`
   to `INFO answered (HTTP N at /)` so a 400 is not rendered as a green OK.
6. Change the missing-token `env_vars` line from `ERROR missing required: ...`
   to `INFO not set: ...` so a fresh machine without credentials does not show
   a red `FAIL`.
7. Add an exact-match special case so `Auth: not configured` renders as `OK`
   (the `doctorStatus` helper handles this).
