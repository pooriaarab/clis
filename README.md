# pooriaarab/clis

Working, source-available CLIs for APIs that don't ship a good one of their own. Sibling repo to [pooriaarab/skills](https://github.com/pooriaarab/skills) — skills are specs, these are runnable code.

Each CLI is generated from the platform's real API spec via [cli-printing-press](https://github.com/mvanhorn/cli-printing-press), then hand-patched where the generator's coverage falls short (see each project's `.printing-press-patches/` for exactly what and why). All three are flag-driven, JSON-output-capable, and built for both terminal use and AI-agent consumption (`--json`, `--dry-run`, `--select`, MCP server binary alongside the CLI binary).

## CLIs

| CLI | What it covers | Auth | Developer access |
|---|---|---|---|
| [`google-ads/`](google-ads/README.md) | Full read (GAQL) and near-full write coverage of the Google Ads API — campaigns, ad groups, keywords, experiments, and every other mutable resource via a generic escape hatch | OAuth2 (Desktop client + refresh token) | — |
| [`reddit-ads/`](reddit-ads/README.md) | Reddit's Ads API v3 — ad accounts, campaigns, ad groups, ads, custom audiences, funding instruments, reporting, forecasting | OAuth2 (interactive browser login built in) | — |
| [`meta-ads/`](meta-ads/README.md) | Meta Marketing API — campaigns, ad sets, ads, creatives, custom audiences, insights, plus a generic node/edge escape hatch for the rest of the Graph API | Access token (no built-in login flow — see that CLI's README for why) | — |
| [`microsoft-ads/`](microsoft-ads/README.md) | Auth-only stub (`doctor`, `auth` only — verified via `go build` + `--help`); pending the SOAP-to-REST migration and developer-token access | OAuth access token + developer token TODO | [Developer Portal — Request Token](https://developers.ads.microsoft.com/account) |
| [`tiktok-ads/`](tiktok-ads/README.md) | Auth-only stub (`doctor`, `auth` only); pending Marketing API app approval | OAuth access token TODO | [Developer portal](https://business-api.tiktok.com/portal) |
| [`linkedin-ads/`](linkedin-ads/README.md) | Auth-only stub (`doctor`, `auth` only); pending Marketing Developer Platform approval | OAuth 2.0 access token TODO | [Developer Portal — My Apps](https://www.linkedin.com/developers/apps) |
| [`pinterest-ads/`](pinterest-ads/README.md) | Thin unverified print, never checked against a real account: `accounts list`, `campaigns list/create`, `audiences list/upload`, `reporting get` against documented `/v5` paths (generic `--body` JSON passthrough) | OAuth 2.0 access token | [My Apps](https://developers.pinterest.com/apps/) |
| [`snapchat-ads/`](snapchat-ads/README.md) | Thin unverified print: `accounts list` (org discovery) and `campaigns list/create` hit documented paths; `audiences` and `reporting` subcommands are listed in `--help` but exit with a not-implemented TODO (no verified paths) | OAuth 2.0 access token | [Apply for Ads API access](https://businesshelp.snapchat.com/s/article/api-apply) |
| [`x-ads/`](x-ads/README.md) | Auth-only stub (`doctor`, `auth` only); pending approved Ads API access and OAuth 1.0a signing | OAuth 1.0a TODO | [Ads API getting started](https://docs.x.com/x-ads-api/getting-started) |
| [`amazon-ads/`](amazon-ads/README.md) | Auth-only stub (`doctor`, `auth` only); pending application approval | OAuth 2.0 access token TODO | [API onboarding](https://advertising.amazon.com/API/docs/en-us/guides/onboarding/overview) |
| [`apple-search-ads/`](apple-search-ads/README.md) | Thin unverified print: `campaigns list/create` and `reporting get` hit documented `/api/v5` paths; `accounts`/`audiences` subcommands are listed in `--help` but exit with a not-implemented TODO | OAuth 2.0 access token | [OAuth guide — API user setup](https://developer.apple.com/documentation/apple_ads/implementing-oauth-for-the-apple-search-ads-api) |
| [`criteo/`](criteo/README.md) | Auth-only stub (`doctor`, `auth` only); pending product and permission selection | OAuth 2.0 TODO | [Onboarding checklist](https://developers.criteo.com/marketing-solutions/docs/onboarding-checklist) |
| [`taboola/`](taboola/README.md) | Thin unverified print: `campaigns list/create` and `reporting get` hit documented Backstage paths; `accounts`/`audiences` subcommands are listed in `--help` but exit with a not-implemented TODO | OAuth 2.0 access token | [Backstage API welcome — credentials via account manager](https://developers.taboola.com/backstage-api/reference/welcome) |
| [`outbrain/`](outbrain/README.md) | Auth-only stub (`doctor`, `auth` only); the public Amplify endpoint reference is not complete enough for safe generation | Token TODO | [Amplify API application](https://developer.outbrain.com/home-page/amplify-api/apply/) |
| [`quora/`](quora/README.md) | Auth-only stub (`doctor`, `auth` only); no public advertiser API reference found | Token TODO | no public application URL found |
| [`trade-desk/`](trade-desk/README.md) | Auth-only stub (`doctor`, `auth` only); pending partner access (provisioned by an account representative, no self-serve registration) | Token TODO | [Partner portal](https://partner.thetradedesk.com/) |
| [`stackadapt/`](stackadapt/README.md) | Auth-only stub (`doctor`, `auth` only); no public advertiser API reference (API key is issued inside the platform to customers) | Token TODO | no public application URL found |
| [`nextdoor/`](nextdoor/README.md) | Auth-only stub (`doctor`, `auth` only); pending partner access | Token TODO | [Developer site — request access](https://developer.nextdoor.com/) |
| [`yandex-direct/`](yandex-direct/README.md) | Thin unverified print: `reporting get` (POST JSON to the documented v501 reports endpoint) only; all other groups are listed in `--help` but exit with a not-implemented TODO | OAuth 2.0 access token | [Yandex Direct developer docs](https://yandex.com/dev/direct/) |
| [`vk-ads/`](vk-ads/README.md) | Auth-only stub (`doctor`, `auth` only); no stable public API contract (access is granted per-request, not self-serve) | Token TODO | [VK Ads cabinet — request access](https://ads.vk.com/) |
| [`tencent-ads/`](tencent-ads/README.md) | Auth-only stub (`doctor`, `auth` only); pending regional account access | Token TODO | [Developer portal](https://developers.e.qq.com/) |
| [`baidu-ads/`](baidu-ads/README.md) | Auth-only stub (`doctor`, `auth` only); pending partner endpoint access | OAuth 2.0 TODO | [Commercial open platform](https://dev2.baidu.com/) |
| [`naver-ads/`](naver-ads/README.md) | Auth-only stub (`doctor`, `auth` only); pending request-signing coverage (API license is issued in the advertiser center) | API license and secret TODO | [Advertiser center](https://searchad.naver.com/) |
| [`kakao-ads/`](kakao-ads/README.md) | Thin unverified print: `reporting get` against the documented `/openapi/v4/campaigns/report` path only; all other groups are listed in `--help` but exit with a not-implemented TODO | Business access token | [Kakao Developers — request the Moment feature](https://developers.kakao.com/) |
| [`line-ads/`](line-ads/README.md) | Auth-only stub (`doctor`, `auth` only); pending the documented request signer (corporate users apply in the Ads Manager) | Access key and secret TODO | [LINE Ads API docs](https://developers.line.biz/en/docs/line-ads-api/) |
| [`yahoo-japan-ads/`](yahoo-japan-ads/README.md) | Auth-only stub (`doctor`, `auth` only); pending service-specific endpoint generation | OAuth 2.0 TODO | [App registration](https://ads-developers.yahoo.co.jp/developercenter/en/startup-guide/app-registration.html) |
| [`mercadolibre-ads/`](mercadolibre-ads/README.md) | Auth-only stub (`doctor`, `auth` only); pending product access | OAuth 2.0 TODO | [Developer portal — create an application](https://developers.mercadolibre.com/) |

"Auth-only stub" means the built binary exposes only `doctor` and `auth` (`status`, `login`, `set-token`) — the `accounts`/`campaigns`/`audiences`/`reporting` functions exist in source as an unwired template and are not reachable. "Thin unverified print" means only the commands named as implemented in the row above hit documented endpoint paths, as generic `--body` JSON passthrough, and even those were never verified end-to-end against a real account, unlike the big three. The remaining subcommands in the same CLI appear in `--help` but call no endpoint at all — they exit with a not-implemented error.

Each subdirectory's README has the exact setup steps for that platform: where to register a developer app, which environment variables the CLI needs, and how to get your first real credential.

## Why generate instead of hand-write

Most ad-platform APIs are wide (dozens to hundreds of resource types) but structurally uniform — Google Ads' ~65 mutable resources all share one request shape, Meta's Graph API is uniformly node/edge shaped. Generating the bulk of the CLI from the platform's own API spec, then closing the remaining gap with one hand-written generic command instead of dozens of near-duplicate generated wrappers, gets full coverage without the hallucination risk of generating 60+ near-identical commands one at a time. See each CLI's `.printing-press-patches/` directory for the specific hand-written additions and the reasoning behind each one.

## Install

Every CLI here is a standalone Go module — no shared build step, no monorepo tooling required:

```bash
git clone https://github.com/pooriaarab/clis.git
cd clis/<cli-name>
go build -o <cli-name>-pp-cli ./cmd/<cli-name>-pp-cli
```

Requires [Go](https://go.dev/dl/) 1.22+. See each CLI's own README for the exact binary name and setup steps.

## Contributing

Found a gap, a bug, or want to add another platform's CLI? PRs welcome. If you're adding a new CLI generated via `cli-printing-press`, please:
1. Verify auth end-to-end against a real account before opening the PR (a `--dry-run` pass alone doesn't catch auth-shape bugs — several were found this way during the Reddit CLI's initial verification, see its `.printing-press-patches/`).
2. Note any hand-written additions (generic escape hatches, auth fixes) in a `.printing-press-patches/*.md` file in that CLI's directory, per the convention the other three already follow.
3. Keep the README's Setup section grounded in the platform's real registration flow, not a generic template — a wrong env var name in the docs is worse than no docs.

## License

Apache-2.0 — see [LICENSE](LICENSE). Each CLI's source files carry the same license header.
