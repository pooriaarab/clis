# Kakao Moment CLI

This printed CLI provides a non-interactive token workflow for Kakao Moment.

## Install

Run go build ./... from this directory. The binary is kakao-ads-pp-cli.

## Get a token

Create or approve an app in the official developer console. Follow the platform's access steps in the links below. Then run:

    kakao-ads-pp-cli auth set-token YOUR_ACCESS_TOKEN

You can also set KAKAO_ADS_ACCESS_TOKEN. Check the local setup with:

    kakao-ads-pp-cli doctor

For a token check, use the platform's validate-token or first read endpoint from the official documentation. This CLI never logs token values.

## Validate a token

Set KAKAO_ADS_ACCOUNT_ID to an ad account allowed by the business token:

    curl --fail-with-body -H "Authorization: Bearer $KAKAO_ADS_ACCESS_TOKEN" -H "adAccountId: $KAKAO_ADS_ACCOUNT_ID" "https://apis.moment.kakao.com/openapi/v4/campaigns/report?datePreset=TODAY&metricsGroup=BASIC&level=CAMPAIGN"

## Commands

The available commands are doctor, auth status, auth login, auth set-token, accounts list (when documented), campaigns list/create, audiences list/upload (when documented), and reporting get (when documented).

Use --help for command details. JSON output is the default.

## API coverage

The verified command covers campaign reports with a business token. Campaign writes, audience uploads, and account reads remain TODO.

## Official references checked 2026-08-11

- https://developers.kakao.com/docs/en/kakaomoment/report
- https://developers.kakao.com/docs/latest/en/kakaomoment/auth
