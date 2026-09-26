# walmart-ca CLI

Read-only walmart.ca order history via session replay. No login flow, no bot-check solving.

- Auth = `auth import` of a Chrome "Copy as cURL" capture; session in `~/.walmart-ca/session.json` (0600).
- `doctor` verifies session state without printing secrets.
- API: `GET /orchestra/cph/graphql/PurchaseHistoryV2/<hash>?variables=...`.
- Default persisted-query hash came from the live en-CA web build; imports override it when Walmart rotates builds.

Run `gofmt`, `go vet`, and `go build ./...` before a PR. Stay inside `walmart-ca/`.
