# Microsoft Advertising CLI

This printed CLI provides a non-interactive token workflow for Microsoft Advertising.

## Install

Run go build ./... from this directory. The binary is microsoft-ads-pp-cli.

## Get a token

Create or approve an app in the official developer console. Follow the platform's access steps in the links below. Then run:

    microsoft-ads-pp-cli auth set-token YOUR_ACCESS_TOKEN

You can also set MICROSOFT_ADS_ACCESS_TOKEN. Check the local setup with:

    microsoft-ads-pp-cli doctor

For a token check, use the platform's validate-token or first read endpoint from the official documentation. This CLI never logs token values.

## Validate a token

The current official references do not publish a safe, general token-validation endpoint for this gated or incomplete integration. After access is approved, use the first endpoint in the official reference. This stub does not guess a URL.

## Commands

The available commands are doctor, auth status, auth login, and auth set-token. API commands stay disabled until the documented access gate or endpoint contract is available.

Use --help for command details. JSON output is the default.

## API coverage

Microsoft is moving from SOAP to REST. This print does not guess a REST resource or hide SOAP envelopes. Add verified campaign, account, reporting, and audience operations after the target account receives the required developer token.

## Official references checked 2026-08-11

- https://learn.microsoft.com/en-us/advertising/guides/authentication-oauth-quick-start?view=bingads-13
- https://learn.microsoft.com/en-us/advertising/guides/authentication-oauth-get-tokens?view=bingads-13
- https://learn.microsoft.com/en-us/advertising/guides/?view=bingads-13
