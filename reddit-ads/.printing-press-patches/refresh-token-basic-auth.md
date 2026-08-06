# Patch: refresh_token request must use HTTP Basic auth

**File:** internal/client/client.go — `refreshAccessToken`

**Bug:** the OAuth refresh request sent `client_id`/`client_secret` as form
fields. Reddit's `/api/v1/access_token` endpoint requires the client
credentials via HTTP Basic auth, so refresh returned HTTP 401. Auto-refresh
silently failed and the user had to re-run `login` every ~24h.

**Fix:** drop the client_id/client_secret form fields; set
`req.SetBasicAuth(ClientID, ClientSecret)` on the refresh request.

**Verified:** after the fix, an expired access token auto-refreshes and a live
`reports get-a` call succeeds with no re-login. Preserve on reprint (ideally
fold upstream into the Printing Press auth template).
