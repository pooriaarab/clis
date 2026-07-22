# auth login

Hand-authored `internal/cli/auth_login.go` (+ one `AddCommand` line in
`auth.go`). Interactive email-OTP login that mints and stores a Firebase ID
token, so users don't have to paste a token by hand.

Flow (all replayable HTTP):
1. `POST soloist.ai/api/users/otp/generate {email}` -> `{sessionId, status}`.
2. Prompt for the emailed code (or `--code`).
3. `POST soloist.ai/api/login/verify {email, sessionId, code}` ->
   `{status, loginLink}` (a Firebase email sign-in link).
4. Extract the `oobCode` from the link, then
   `POST identitytoolkit.googleapis.com/v1/accounts:signInWithEmailLink?key=<webKey> {email, oobCode}`
   -> `{idToken, refreshToken, expiresIn}`.
5. Persist via `cfg.SaveTokens("", "", idToken, refreshToken, expiry)` — the
   ID token is stored as the access token, which the client sends as
   `Authorization: Bearer`.

`--email` / `--code` run it non-interactively. The Firebase **web** API key
(public client config, not a secret) defaults to the prod project's key and is
overridable with `--firebase-key` or `SOLOIST_FIREBASE_API_KEY` (e.g. for the
nonprod project behind main.soloist.ai).
