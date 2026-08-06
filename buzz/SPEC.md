# buzz-cli — spec (v1: full GUI parity + headless agents/fleet)

A standalone Go CLI that does **everything the Buzz desktop app does**, headlessly,
by speaking the Buzz relay protocol directly. Home: `pooriaarab/clis/buzz`.
Sibling binary: an MCP server (`cmd/buzz-mcp`) exposing the same operations as tools
(mirror the `soloist` layout).

Reference implementation (read these — same machine):
- Protocol + relay + kinds + SDK: `/Users/parab/code/buzz` (Rust). Event kinds in
  `crates/buzz-core/src/kind.rs`. NIP-OA in `crates/buzz-sdk/src/nip_oa.rs`. Event
  builders in `crates/buzz-sdk/src/builders.rs`. Relay HTTP surface in
  `crates/buzz-relay/src/api`. WS client in `crates/buzz-ws-client`.
- **Behavioral oracle:** the bundled CLI `/Applications/Buzz.app/Contents/MacOS/buzz`
  (buzz-cli) already implements most non-agent commands. Match its command tree,
  flags, JSON output shape, and exit codes exactly. `buzz <cmd> --help` is truth.
- ACP harness the agent runtime drives: `/Applications/Buzz.app/Contents/MacOS/buzz-acp`.

## Hard rules
- **Generic. No PII. No hardcoded communities or keys.** Nothing about `mozilla`,
  `beeloud`, `pooria`, or any nsec/pubkey may appear in source, tests, README, or
  fixtures. Relay URL, identity key, and owner key come only from flags / env /
  config file at runtime.
- Go, matching sibling CLIs (`cmd/`, `internal/`, `go.mod module buzz-cli`,
  `agentcookie.toml`, `AGENTS.md`, `CLAUDE.md`, `CHANGELOG.md`, README). Use cobra
  (sibling CLIs do). `go 1.26`.
- Never invent event schemas — derive exact bytes/tags/kinds from `~/code/buzz`.
  Parameterized-replaceable kinds (30000–39999) have content byte-order that pins
  the NIP-01 id; preserve field order from the Rust structs.

## Config & identity
- Config file `~/.config/buzz-cli/config.toml` (path overridable via `--config` /
  `BUZZ_CONFIG`). Holds: default `relay_url`, named identities `{name -> nsec}`,
  and an optional `owner` identity used to sign NIP-OA attestations / owner-only
  events. Never log secrets.
- Precedence: flags > env (`BUZZ_RELAY_URL`, `BUZZ_PRIVATE_KEY`, `BUZZ_AUTH_TAG`,
  `BUZZ_OWNER_KEY`) > config file.
- Support nsec (bech32) and 64-hex keys everywhere.

## Protocol foundation (`internal/nostr`, `internal/client`)
- secp256k1 Schnorr (BIP-340) event signing; NIP-01 event id = sha256 of the
  canonical serialized `[0,pubkey,created_at,kind,tags,content]`.
- WebSocket client: connect, handle relay `AUTH` challenge → send kind **22242**
  auth event (NIP-42); when acting as an agent, attach the NIP-OA `auth` tag so the
  relay grants virtual membership (needs relay `BUZZ_ALLOW_NIP_OA_AUTH`; matches
  desktop). REQ/EVENT/EOSE/CLOSE handling; publish via `EVENT`.
- REST client: `POST /events` (submit signed event), `POST /query` (REQ filters,
  NIP-50 search), `POST /count`, invites `POST /api/invites` + `/api/invites/claim`
  (NIP-98 signed), Blossom media upload/download. Prefer REST for one-shot writes,
  WS for subscriptions/agent runtime.
- **NIP-OA auth tag mint** (owner-signed): `preimage = "nostr:agent-auth:" +
  agentPubHex + ":" + conditions` (conditions default ""); `msg = sha256(preimage)`;
  `sig = schnorr(msg, ownerSecret)`; tag = `["auth", ownerPubHex, conditions, sigHex]`.
  Verified working against the live relay.

## Command tree — full parity (match bundled buzz-cli exactly)
Global flags: `--relay`, `--identity <name>`/`--key`, `--format compact|json`
(global, before subcommand), `--config`. Exit codes: 0 ok, 1 input, 2 relay/net,
3 auth, 4 other, 5 write-conflict. Errors JSON on stderr `{"error","message"}`.

- `agents` — draft-create, draft-update, archive, unarchive, archived **PLUS new
  headless CRUD** (see Agents section): create, list, get, update, run, stop,
  delete, fleet.
- `messages` — send, send-diff, edit, delete, get, thread, search, vote
- `channels` — list, get, search, create, update, topic, purpose, join, leave,
  archive, unarchive, delete, members, add-member, remove-member, set-add-policy
- `canvas` — get, set
- `reactions` — add, remove, get
- `emoji` — list, set, rm, export, import
- `dms` — list, open, add-member, hide
- `users` — get, set-profile, presence, set-presence, set-status
- `workflows` — list, get, create, update, delete, trigger, runs, approve
- `feed` — get
- `social` — publish, set-contacts, event, notes, contacts, set-list, list
- `notes` — set, get, ls, rm
- `repos` — create, get, list, bind, protect
- `projects` — create, get, list, add-repo, remove-repo, update, delete
- `patches` — send, get, list, status
- `issues` — create, get, list, status
- `pr` — open, update, get, list, status
- `media` — get ; `upload` — file
- `mem` — ls, get, hash, set, patch, rm
- `pack` — validate, inspect
- `moderation` — reports, resolve, ban, unban, timeout, untimeout, restricted, audit
- `invite` — create (owner mints `POST /api/invites`), claim, list
- `settings` — get/set community + identity settings the app exposes (relay
  metadata NIP-11, join policy/ToS, agent defaults). Derive exact surface from app.

## Agents & fleet (the headless capability the app gates behind its GUI)
A first-class managed agent = these events, so it renders in-app like a native agent
(name, avatar, Runtime/Instances/Channels/Memories, "Managed by <owner>"):
1. **keygen** — fresh secp256k1 identity (the agent's own nsec).
2. **profile** kind **0** signed by the agent (display name, avatar, about).
3. **persona** kind **30175** (`KIND_PERSONA`) — content struct in
   `desktop/src-tauri/src/managed_agents/persona_events.rs` (`PersonaEventContent`:
   display_name, system_prompt, avatar_url, runtime, model, provider, name_pool;
   field order pins id), d-tag = persona slug. Signed by agent (or owner per app).
4. **managed-agent projection** kind **30177** (`KIND_MANAGED_AGENT`) — owner-signed
   addressable event at coord `30177:<owner>:<agentPub>`; this is what surfaces the
   agent under "Agents / Managed by you". Derive content+tags from the desktop
   `create_managed_agent` path (`desktop/src-tauri/src/commands/agents.rs`) +
   `retain_managed_agent_pending`.
5. **auth tag** — owner NIP-OA attestation (above) so the agent authenticates.
6. **channel membership** — owner signs kind **9000** (`KIND_NIP29_PUT_USER`,
   `channels add-member`) to add the agent to each target channel; or agent self-joins
   an open channel with kind **9021**.
7. **run** — supervise a `buzz-acp` process with env `BUZZ_PRIVATE_KEY`(agent nsec),
   `BUZZ_RELAY_URL`, `BUZZ_AUTH_TAG`, `BUZZ_ACP_AGENT_COMMAND`(harness), plus harness
   env; forward `respond-to`, `subscribe`, `system-prompt`, `model`, `parallelism`.

Commands:
- `agents create` — flags: --name, --system-prompt(/-file), --avatar, --runtime
  (harness command), --runtime-args, --model, --channels (csv of channel ids/names to
  add to), --respond-to, --owner-key. Does steps 1–6, prints the agent identity +
  a saved record path. Store the agent nsec + auth tag in the config identities.
- `agents run <name|pubkey>` — steps 7; `--detach` to background + write a pidfile;
  streams logs otherwise.
- `agents stop`, `agents list`, `agents get`, `agents update`, `agents delete`
  (kind 5 / NIP-IA 9035 archive + remove membership).
- `agents fleet` — `--count N --name-prefix X --runtime R --system-prompt-file F
  [--persona-dir D] --channels C`. Loops create for N distinct identities that share
  one runtime; optional per-agent persona files; `--run` to also launch/supervise all
  (a supervisor managing N buzz-acp children with restart+backoff; log per agent;
  `--max-concurrent` to cap simultaneously-running harness processes). Emit a clear
  summary + a fleet manifest file. NEVER silently cap — log any bound.

## MCP (`cmd/buzz-mcp`)
Mirror the command set as MCP tools (stdio), same as `soloist`'s `*-mcp`. Reuse the
`internal/` layer; do not duplicate protocol logic.

## Quality gates
`go vet`, `go build ./...`, `gofmt`, unit tests for: event id/signing vectors
(cross-check against a known buzz event), NIP-OA tag mint (round-trip verify),
NIP-42 auth event shape, config precedence, and each command's request builder.
Add a `just`/Makefile `shipcheck` mirroring sibling CLIs. Update CHANGELOG + README.
No network in unit tests; gate live E2E behind an env flag.
