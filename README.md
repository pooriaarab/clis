# pooriaarab/clis

Working, source-available CLIs for APIs that don't ship a good one of their own. Sibling repo to [pooriaarab/skills](https://github.com/pooriaarab/skills) — skills are specs, these are runnable code.

Each CLI is generated from the platform's real API spec via [cli-printing-press](https://github.com/mvanhorn/cli-printing-press), then hand-patched where the generator's coverage falls short (see each project's `.printing-press-patches/` for exactly what and why). All three are flag-driven, JSON-output-capable, and built for both terminal use and AI-agent consumption (`--json`, `--dry-run`, `--select`, MCP server binary alongside the CLI binary).

## CLIs

| CLI | What it covers | Auth |
|---|---|---|
| [`google-ads/`](google-ads/README.md) | Full read (GAQL) and near-full write coverage of the Google Ads API — campaigns, ad groups, keywords, experiments, and every other mutable resource via a generic escape hatch | OAuth2 (Desktop client + refresh token) |
| [`reddit-ads/`](reddit-ads/README.md) | Reddit's Ads API v3 — ad accounts, campaigns, ad groups, ads, custom audiences, funding instruments, reporting, forecasting | OAuth2 (interactive browser login built in) |
| [`meta-ads/`](meta-ads/README.md) | Meta Marketing API — campaigns, ad sets, ads, creatives, custom audiences, insights, plus a generic node/edge escape hatch for the rest of the Graph API | Access token (no built-in login flow — see that CLI's README for why) |

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
