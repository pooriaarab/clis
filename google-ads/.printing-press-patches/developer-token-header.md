# developer-token / login-customer-id headers

Fixed `internal/client/client.go`'s request-building loop (right after the
`c.Config.Headers` static-headers loop) to attach `developer-token` and
`login-customer-id` as real HTTP headers when those config fields are set.

Why: Google Ads REST requires both as headers on every single call, in
addition to the OAuth Bearer token. The generator correctly detected and
loaded `GOOGLE_ADS_DEVELOPER_TOKEN` / `GOOGLE_ADS_LOGIN_CUSTOMER_ID` into
`Config.GoogleAdsDeveloperToken` / `Config.GoogleAdsLoginCustomerId`, but
never wired them into an outgoing request — it only used them for the
credential-masking helper (`addCredential` in the same file), which redacts
secrets from log/dry-run output, not the live request path. Every call
failed with `DEVELOPER_TOKEN_PARAMETER_MISSING` until this was added.

This is a generic-single-credential-auth-model gap: the API's "two more
required headers beyond the auth header" shape isn't something the generic
REST client template accounts for. Worth flagging upstream in
mvanhorn/cli-printing-press if this pattern recurs on another
multi-header-auth API.

On regen: re-add the two `req.Header.Set(...)` lines if `client.go` is
regenerated from scratch instead of merged.
