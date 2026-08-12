# Snapchat Ads CLI

This printed CLI provides a non-interactive token workflow for Snapchat Ads.

## Install

Run go build ./... from this directory. The binary is snapchat-ads-pp-cli.

## Get a token

Create or approve an app in the official developer console. Follow the platform's access steps in the links below. Then run:

    snapchat-ads-pp-cli auth set-token YOUR_ACCESS_TOKEN

You can also set SNAPCHAT_ADS_ACCESS_TOKEN. Check the local setup with:

    snapchat-ads-pp-cli doctor

For a token check, use the platform's validate-token or first read endpoint from the official documentation. This CLI never logs token values.

## Validate a token

Use the documented organization discovery endpoint:

    curl --fail-with-body -H "Authorization: Bearer $SNAPCHAT_ADS_ACCESS_TOKEN" "https://adsapi.snapchat.com/v1/me/organizations?with_ad_accounts=true"

## Commands

The available commands are doctor, auth status, auth login, auth set-token, accounts list (when documented), campaigns list/create, audiences list/upload (when documented), and reporting get (when documented).

Use --help for command details. JSON output is the default.

## API coverage

Reporting and audience writes stay TODO until each current Ads API resource is added from its official reference. Campaign list and create use the paths shown in the official Ads API docs.

## Official references checked 2026-08-11

- https://developers.snap.com/marketing-api/Ads-API/authentication
- https://developers.snap.com/marketing-api/Ads-API/quick-start
- https://developers.snap.com/marketing-api/Ads-API/campaigns
