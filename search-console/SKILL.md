---
name: search-console
description: "Launch a new domain into Google Search Console, Bing Webmaster Tools and IndexNow: verify ownership over DNS, submit the sitemap, send URLs. Trigger phrases: `launch this domain in search console`, `verify a domain in Google and Bing`, `submit a sitemap to Bing`, `send URLs to IndexNow`, `index a new site`."
author: "Pooria Arab"
license: "Apache-2.0"
argument-hint: "<command> [args]"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - search-console
---

# search-console

Use this skill to put a new domain into Google Search Console, Bing Webmaster Tools and IndexNow. The binary is `search-console`. Build it with `make build` in the `search-console/` directory.

Run `search-console --help` for the full command list. Every command accepts `--json` and `--dry-run`.

## Before you run anything

Run `search-console auth status --check`. If it reports a missing credential, stop and ask the owner to finish the one-time setup below. Do not try to create credentials yourself.

The owner does this once:

1. Create a Google Cloud OAuth client of type Desktop app. Enable the Search Console API and the Site Verification API. Set the consent screen to In production.
2. Run `search-console auth google --client-id <id> --client-secret <secret>` and sign in.
3. Create a Bing API key under Settings, API access, in Bing Webmaster Tools. Export `BING_WEBMASTER_API_KEY`.
4. Optional. Create a Cloudflare token with Zone Read and DNS Edit. Export `CLOUDFLARE_API_TOKEN`. The CLI reads it from the environment only.
5. Serve the IndexNow key file at `https://<domain>/<key>.txt`. The first `indexnow submit` prints the file path.

## Launch a domain

Always dry-run first, then run for real:

```bash
search-console launch example.com --sitemap https://example.com/sitemap.xml --cloudflare-zone auto --dry-run
search-console launch example.com --sitemap https://example.com/sitemap.xml --cloudflare-zone auto --json
```

The result is a table of five steps: `google verify`, `google sitemap`, `bing verify`, `bing sitemap` and `indexnow`. Each row is `pass`, `fail` or `skip`. Run the command again to retry. A step that is already done passes.

## Single steps

| Goal | Command |
|---|---|
| Check a sitemap | `search-console sitemap check <url>` |
| Verify in Google | `search-console google verify <domain> --cloudflare-zone auto` |
| Verify in Bing | `search-console bing verify <domain> --cloudflare-zone auto` |
| Submit a sitemap | `search-console google sitemap submit <domain> <url>` or `bing sitemap submit` |
| Read sitemap status | `search-console google sitemap status <domain> <url>` or `bing sitemap status` |
| Bing URL quota | `search-console bing quota <domain>` |
| Send URLs to IndexNow | `search-console indexnow submit <domain> --from-sitemap` |

Without `--cloudflare-zone`, the verify commands print the DNS record and wait. Tell the owner to add it.

## Exit codes

- `0` success.
- `1` a request or a step failed. Read the `error` field.
- `2` usage error. Fix the arguments.
- `3` the sitemap has problems, or a service reported sitemap errors.
- `4` DNS was not ready in time. Wait, then run the command again.

## Rules

- Never print, log or store a secret. The CLI hides secrets in `--dry-run` output.
- Never add a DNS record by hand when `--cloudflare-zone` works.
- The Bing CNAME must not be proxied. The CLI already sets this.
- The CLI has been tested against fake servers only. Report any difference from the live APIs.
