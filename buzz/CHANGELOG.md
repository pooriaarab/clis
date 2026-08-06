# Changelog

## Unreleased

- Added the initial `buzz-cli` Go module with Cobra entrypoint at `cmd/buzz`.
- Added NIP-01 event serialization, event id hashing, BIP-340 Schnorr signing and verification, nsec/hex key parsing, NIP-42 auth events, and owner-signed NIP-OA auth tags.
- Added TOML config resolution with flags, environment, and file precedence.
- Added REST and WebSocket relay clients for event publishing, query/count requests, subscriptions, and relay auth challenges.
- Added Buzz schema builders for kind 0 profiles, kind 30175 personas, kind 30177 managed-agent projections, and kind 9000 channel membership events.
- Wired the first command groups for users, channels, messages, agents, and fleet, plus explicit stubs for the remaining spec command tree.
- Added unit tests for event signing/verification, NIP-42 auth shape, NIP-OA auth tag round trip, config precedence, and managed-agent event field order.
- Implemented `canvas`, `reactions`, `emoji`, `dms`, `feed`, `social`, `notes`, `invite`, and `settings` command groups against real relay schemas (kinds 7, 30023, 30030, 40100, 41001/41010/41011/41012, 9033, plus `POST /api/invites`, `POST /api/invites/claim`, and the NIP-11 relay info document).
- Added NIP-19 `naddr` encode/decode (`internal/nostr/naddr.go`) for `notes` addressable-coordinate references.
- Added local (CLI-side) invite bookkeeping in the config file, since the relay exposes no invite-listing endpoint.
