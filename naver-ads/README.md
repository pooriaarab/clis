# Naver Ads CLI

This printed CLI provides a non-interactive token workflow for Naver Ads.

## Install

Run go build ./... from this directory. The binary is naver-ads-pp-cli.

## Get a token

Create or approve an app in the official developer console. Follow the platform's access steps in the links below. Then run:

    naver-ads-pp-cli auth set-token YOUR_ACCESS_TOKEN

You can also set NAVER_ADS_ACCESS_TOKEN. Check the local setup with:

    naver-ads-pp-cli doctor

For a token check, use the platform's validate-token or first read endpoint from the official documentation. This CLI never logs token values.

## Validate a token

The current official references do not publish a safe, general token-validation endpoint for this gated or incomplete integration. After access is approved, use the first endpoint in the official reference. This stub does not guess a URL.

## Commands

The available commands are doctor, auth status, auth login, and auth set-token. API commands stay disabled until the documented access gate or endpoint contract is available.

Use --help for command details. JSON output is the default.

## API coverage

Naver Search Ads uses an API license, secret key, customer ID, and request signing. This print keeps auth-only until the signer and current method set are generated from the official reference.

## Official references checked 2026-08-11

+- https://naver.github.io/searchad-apidoc/
- https://ads.naver.com/help/faq/302?t=1769857366078
