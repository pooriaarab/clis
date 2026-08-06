# buzz-cli — contributor guide (AI agents)

Read `SPEC.md` first. It is the source of truth for scope, protocol, command tree,
and the agents/fleet design.

## Non-negotiables
- **Generic, no PII.** Never hardcode a relay URL, community name, key, or pubkey.
  Everything comes from flags / env / `~/.config/buzz-cli/config.toml` at runtime.
- Match the bundled reference CLI `/Applications/Buzz.app/Contents/MacOS/buzz`
  command tree, flags, JSON output, and exit codes exactly (it is the oracle).
- Derive every event kind / content / tag from the Rust source at
  `/Users/parab/code/buzz` — never invent schemas. Preserve struct field order for
  parameterized-replaceable kinds (their content pins the NIP-01 id).

## Layout (mirror the sibling `soloist` CLI)
- `cmd/buzz/` — CLI entrypoint (cobra). `cmd/buzz-mcp/` — MCP server (stdio).
- `internal/nostr` — event build/sign (BIP-340), NIP-01 id, NIP-42 auth, NIP-OA tag.
- `internal/client` — WebSocket + REST relay client.
- `internal/config` — config file + identity store + flag/env precedence.
- `internal/cli` — command groups. `internal/types` — shared types.

## Gates
`gofmt`, `go vet`, `go build ./...`, unit tests (no network; live E2E behind an env
flag). Keep secrets out of logs. Update CHANGELOG + README.
