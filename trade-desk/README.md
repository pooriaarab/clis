# The Trade Desk CLI

This printed CLI provides a non-interactive token workflow for The Trade Desk.

## Install

Run go build ./... from this directory. The binary is trade-desk-pp-cli.

## Get a token

Create or approve an app in the official developer console. Follow the platform's access steps in the links below. Then run:

    trade-desk-pp-cli auth set-token YOUR_ACCESS_TOKEN

You can also set TRADE_DESK_ACCESS_TOKEN. Check the local setup with:

    trade-desk-pp-cli doctor

For a token check, use the platform's validate-token or first read endpoint from the official documentation. This CLI never logs token values.

## Validate a token

The current official references do not publish a safe, general token-validation endpoint for this gated or incomplete integration. After access is approved, use the first endpoint in the official reference. This stub does not guess a URL.

## Commands

The available commands are doctor, auth status, auth login, and auth set-token. API commands stay disabled until the documented access gate or endpoint contract is available.

Use --help for command details. JSON output is the default.

## API coverage

undefined

## Official references checked 2026-08-11

- The Trade Desk APIs are partner-gated. Add verified account, campaign, audience, and reporting resources after partner access.
