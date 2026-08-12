# Apple Search Ads CLI

This printed CLI provides a non-interactive token workflow for Apple Search Ads.

## Install

Run go build ./... from this directory. The binary is apple-search-ads-pp-cli.

## Get a token

Create or approve an app in the official developer console. Follow the platform's access steps in the links below. Then run:

    apple-search-ads-pp-cli auth set-token YOUR_ACCESS_TOKEN

You can also set APPLE_SEARCH_ADS_ACCESS_TOKEN. Check the local setup with:

    apple-search-ads-pp-cli doctor

For a token check, use the platform's validate-token or first read endpoint from the official documentation. This CLI never logs token values.

## Validate a token

Use the documented campaign read endpoint. Set APPLE_SEARCH_ADS_ORG_ID first:

    curl --fail-with-body -H "Authorization: Bearer $APPLE_SEARCH_ADS_ACCESS_TOKEN" -H "orgId: $APPLE_SEARCH_ADS_ORG_ID" https://api.searchads.apple.com/api/v5/campaigns

## Commands

The available commands are doctor, auth status, auth login, auth set-token, accounts list (when documented), campaigns list/create, audiences list/upload (when documented), and reporting get (when documented).

Use --help for command details. JSON output is the default.

## API coverage

Apple Search Ads does not expose a customer-match upload operation in the verified API pages used for this print. Audience upload remains TODO.

## Official references checked 2026-08-11

- https://developer.apple.com/documentation/apple_ads/implementing-oauth-for-the-apple-search-ads-api
- https://developer.apple.com/documentation/apple_ads/get-a-campaign?changes=l_6
- https://developer.apple.com/documentation/apple_ads/get-campaign-level-reports?changes=_3
