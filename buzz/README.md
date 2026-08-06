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
```

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
```

The full top-level command tree from the spec is wired. Commands whose schemas are not implemented yet return a structured JSON error instead of guessing an event shape.

## Verification

Run:

```sh
gofmt -l .
go vet ./...
go build ./...
go test ./...
```

Live relay checks should be gated outside unit tests by environment-provided runtime config.
