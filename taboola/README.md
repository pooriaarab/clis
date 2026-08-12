# Taboola Backstage CLI

This printed CLI provides a non-interactive token workflow for Taboola Backstage.

## Install

Run go build ./... from this directory. The binary is taboola-pp-cli.

## Get a token

Create or approve an app in the official developer console. Follow the platform's access steps in the links below. Then run:

    taboola-pp-cli auth set-token YOUR_ACCESS_TOKEN

You can also set TABOOLA_ACCESS_TOKEN. Check the local setup with:

    taboola-pp-cli doctor

For a token check, use the platform's validate-token or first read endpoint from the official documentation. This CLI never logs token values.

## Validate a token

Set TABOOLA_ACCOUNT_ID to an advertiser account from the official setup flow:

    curl --fail-with-body -H "Authorization: Bearer $TABOOLA_ACCESS_TOKEN" "https://backstage.taboola.com/backstage/api/1.0/$TABOOLA_ACCOUNT_ID/campaigns/"

## Commands

The available commands are doctor, auth status, auth login, auth set-token, accounts list (when documented), campaigns list/create, audiences list/upload (when documented), and reporting get (when documented).

Use --help for command details. JSON output is the default.

## API coverage

Taboola's public advertiser docs cover campaigns and standard reports. Audience uploads are not added without a verified current endpoint.

## Official references checked 2026-08-11

- https://developers.taboola.com/backstage-api/reference/getting-an-access-token
- https://developers.taboola.com/backstage-api/reference/get-all-campaigns
- https://developers.taboola.com/backstage-api/reference/create-a-campaign
- https://developers.taboola.com/backstage-api/docs/reporting-overview
