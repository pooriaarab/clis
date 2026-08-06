# buzz-cli

Standalone Go CLI for the Buzz relay protocol.

This first increment provides the protocol foundation, config resolution, relay clients, and a real Cobra command tree for the Buzz desktop CLI surface. Relay URL, identity key, auth tag, and owner key are always supplied at runtime through flags, environment, or config.

## Configuration

Default config path:

```text
~/.config/buzz-cli/config.toml
```

Supported values:

```toml
relay_url = "<relay-url>"
owner_key = "<owner-private-key>"

[identities]
agent = "<agent-private-key>"

[auth_tags]
agent = "<auth-tag-json>"

[[invites]]
code = "<invite-code>"
url = "<invite-url>"
expires_at = 0
max_uses = 0
uses_remaining = 0
created_at = 0
```

The `[[invites]]` entries are written automatically by `buzz invite create` and read back by `buzz invite list` — the relay has no endpoint to list previously-minted invites, so this is CLI-local bookkeeping only.

Precedence is:

```text
flags > environment > config file
```

Environment variables:

```text
BUZZ_CONFIG
BUZZ_RELAY_URL
BUZZ_PRIVATE_KEY
BUZZ_AUTH_TAG
BUZZ_OWNER_KEY
```

Private keys may be `nsec` bech32 values or 64-character hex values.

## Commands

Implemented groups in this increment:

```text
buzz users get
buzz users set-profile
buzz users presence
buzz channels list
buzz channels get
buzz channels create
buzz channels join
buzz channels add-member
buzz channels remove-member
buzz channels members
buzz messages send
buzz messages get
buzz messages thread
buzz agents create
buzz agents list
buzz agents get
buzz agents run
buzz agents stop
buzz agents delete
buzz agents fleet
buzz fleet
buzz canvas get
buzz canvas set
buzz reactions add
buzz reactions remove
buzz reactions get
buzz emoji list
buzz emoji set
buzz emoji rm
buzz emoji export
buzz emoji import
buzz dms list
buzz dms open
buzz dms add-member
buzz dms hide
buzz feed get
buzz social publish
buzz social set-contacts
buzz social event
buzz social notes
buzz social contacts
buzz social set-list
buzz social list
buzz notes set
buzz notes get
buzz notes ls
buzz notes rm
buzz invite create
buzz invite claim
buzz invite list
buzz settings get
buzz settings set
```

The full top-level command tree from the spec is wired. Commands whose schemas are not implemented yet return a structured JSON error instead of guessing an event shape.

`buzz invite list` and `buzz notes get --content-only` are the two intentional deviations from strict "everything is JSON" output: `invite list` reads local config state (the relay has no invite-listing endpoint), and `notes get --content-only` prints the raw markdown body instead of a JSON envelope.

## Verification

Run:

```sh
gofmt -l .
go vet ./...
go build ./...
go test ./...
```

Live relay checks should be gated outside unit tests by environment-provided runtime config.
