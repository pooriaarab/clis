# LINE Ads CLI

This printed CLI provides a non-interactive token workflow for LINE Ads.

## Install

Run go build ./... from this directory. The binary is line-ads-pp-cli.

## Get a token

Create or approve an app in the official developer console. Follow the platform's access steps in the links below. Then run:

    line-ads-pp-cli auth set-token YOUR_ACCESS_TOKEN

You can also set LINE_ADS_ACCESS_TOKEN. Check the local setup with:

    line-ads-pp-cli doctor

For a token check, use the platform's validate-token or first read endpoint from the official documentation. This CLI never logs token values.

## Validate a token

The current official references do not publish a safe, general token-validation endpoint for this gated or incomplete integration. After access is approved, use the first endpoint in the official reference. This stub does not guess a URL.

## Commands

The available commands are doctor, auth status, auth login, and auth set-token. API commands stay disabled until the documented access gate or endpoint contract is available.

Use --help for command details. JSON output is the default.

## API coverage

LINE Ads uses request signatures with an access key and secret. This print avoids a partial signer and keeps campaign, audience, and reporting operations TODO.

## Official references checked 2026-08-11

- https://developers.line.biz/en/docs/line-ads-api/
- https://ads.line.me/public-docs/pages/v3/3.11.4/certificated-ad-tech-general-partner/
