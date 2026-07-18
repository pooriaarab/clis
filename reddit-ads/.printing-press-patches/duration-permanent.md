# Force duration=permanent on login

Added `"duration": {"permanent"}` to the authorize-request params in
`internal/cli/auth.go`'s `runOAuthLogin` (hand-added, one line, not from the
generic OAuth2 template the generator uses).

Why: Reddit's authorization endpoint issues a short-lived (1 hour) access
token with **no refresh_token at all** unless `duration=permanent` is
present on the authorize request. Without this, every `reddit-ads-pp-cli
login` silently produces a token that stops working in an hour with no way
to renew it short of logging in again — a real functional bug, not a
preference. Confirmed against Reddit's real OAuth2 docs
(components.securitySchemes in the downloaded openapi.json only documents
the scopes, not this parameter, since it's an OAuth2-spec-standard param
Reddit repurposes, not an API-specific field the spec would capture).

On regen: re-add this line if `auth.go`'s `runOAuthLogin` function is
regenerated from scratch instead of merged.
