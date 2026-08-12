# Yandex Direct CLI

This printed CLI provides a non-interactive token workflow for Yandex Direct.

## Install

Run go build ./... from this directory. The binary is yandex-direct-pp-cli.

## Get a token

Create or approve an app in the official developer console. Follow the platform's access steps in the links below. Then run:

    yandex-direct-pp-cli auth set-token YOUR_ACCESS_TOKEN

You can also set YANDEX_DIRECT_ACCESS_TOKEN. Check the local setup with:

    yandex-direct-pp-cli doctor

For a token check, use the platform's validate-token or first read endpoint from the official documentation. This CLI never logs token values.

## Validate a token

Create a valid JSON report body from the official Reports specification, then run:

    curl --fail-with-body -H "Authorization: Bearer $YANDEX_DIRECT_ACCESS_TOKEN" -H "Content-Type: application/json" --data @report.json https://api.direct.yandex.com/json/v501/reports

## Commands

The available commands are doctor, auth status, auth login, auth set-token, accounts list (when documented), campaigns list/create, audiences list/upload (when documented), and reporting get (when documented).

Use --help for command details. JSON output is the default.

## API coverage

The reports command posts JSON to the documented v501 reports endpoint. Campaign and audience method coverage remains TODO until generated from the current method reference.

## Official references checked 2026-08-11

- https://yandex.com/dev/direct/doc/en/
- https://www.yandex.ru/dev/direct/doc/ru/reports
- https://m.yandex.ru/support/direct/ru/alternative-interfaces/api
