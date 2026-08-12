# Pinterest Ads CLI

This printed CLI provides a non-interactive token workflow for Pinterest Ads.

## Install

Run go build ./... from this directory. The binary is pinterest-ads-pp-cli.

## Get a token

Create or approve an app in the official developer console. Follow the platform's access steps in the links below. Then run:

    pinterest-ads-pp-cli auth set-token YOUR_ACCESS_TOKEN

You can also set PINTEREST_ADS_ACCESS_TOKEN. Check the local setup with:

    pinterest-ads-pp-cli doctor

For a token check, use the platform's validate-token or first read endpoint from the official documentation. This CLI never logs token values.

## Validate a token

Use the documented ad-account read endpoint:

    curl --fail-with-body -H "Authorization: Bearer $PINTEREST_ADS_ACCESS_TOKEN" https://api.pinterest.com/v5/ad_accounts

## Validate a token

Use the documented ad-account read endpoint:

    curl --fail-with-body -H "Authorization: Bearer $PINTEREST_ADS_ACCESS_TOKEN" https://api.pinterest.com/v5/ad_accounts

## Commands

The available commands are doctor, auth status, auth login, auth set-token, accounts list (when documented), campaigns list/create, audiences list/upload (when documented), and reporting get (when documented).

Use --help for command details. JSON output is the default.

## API coverage

Audience uploads use the documented customer_lists endpoint. The audience list command uses the documented audiences resource.

## Official references checked 2026-08-11

- https://developers.pinterest.com/docs/api/v5/introduction/?query=authentication
- https://developers.pinterest.com/docs/work-with-ads/create-campaigns-and-ad-groups/
- https://developers.pinterest.com/docs/work-with-targets-and-audiences/create-audiences/
- https://developers.pinterest.com/docs/analytics-and-reports/ads-reporting/
