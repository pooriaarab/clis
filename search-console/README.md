# search-console

Launch a new domain into Google Search Console, Bing Webmaster Tools and IndexNow with one command.

The CLI proves that you own the domain, adds it to each service, submits the sitemap and tells IndexNow about your URLs. Every step is safe to run again.

## Status

The E2E tests run the real binary against fake Google, Bing, Cloudflare and IndexNow servers. The CLI has not run against the live APIs yet. The first live run is the real test. Use `--dry-run` first.

## Install

```bash
git clone https://github.com/pooriaarab/clis.git
cd clis/search-console
make build
```

The binary is `bin/search-console`. You need [Go](https://go.dev/dl/) 1.22 or later.

## One-time setup

You do these steps once, by hand. The CLI cannot do them for you.

1. **Google.** In Google Cloud, create a project and enable the Search Console API and the Site Verification API. Create an OAuth client of type Desktop app. Set the consent screen to In production. A refresh token from a Testing consent screen expires after 7 days.
2. **Sign in to Google.** Run `search-console auth google --client-id <id> --client-secret <secret>`. A browser opens. Sign in with the Google account that should own the domains. The CLI saves the refresh token in `~/.config/search-console/google.json` with mode 0600.
3. **Bing.** In Bing Webmaster Tools, open Settings, then API access, and create an API key. Set `BING_WEBMASTER_API_KEY`. The CLI never saves this key.
4. **Cloudflare (optional).** Create an API token with Zone Read and DNS Edit for your zone. Set `CLOUDFLARE_API_TOKEN`. The CLI reads the token from the environment only. Without it, the CLI prints each DNS record and waits while you add it.
5. **IndexNow.** The first run makes a key and writes `<key>.txt` to `--key-dir`. Serve that file at `https://<domain>/<key>.txt` before the CLI pings IndexNow. See [First launch and the IndexNow key file](#first-launch-and-the-indexnow-key-file).
6. **Sitemap.** Publish the sitemap at a public URL before you launch.

Check your setup with `search-console auth status --check`.

## Launch a domain

```bash
export BING_WEBMASTER_API_KEY=...
export CLOUDFLARE_API_TOKEN=...
search-console launch example.com --sitemap https://example.com/sitemap.xml --cloudflare-zone auto
```

The command prints one table:

```text
STEP            STATUS  DETAIL
google verify   pass    verified after 1 attempt(s)
google sitemap  pass    done
bing verify     pass    verified after 1 attempt(s)
bing sitemap    pass    status Pending
indexnow        pass    2 URL(s) sent
```

A failed step does not stop the others. A sitemap step is skipped when its verify step failed. The exit code is the code of the first failed step.

`launch` checks every flag before it changes anything. A bad flag exits 2 with no DNS record, site or feed made. A dry run checks the same flags and exits the same way.

| Flag | Default | Use |
|---|---|---|
| `--sitemap <url>` | required | Absolute URL of the sitemap. Google and Bing get this URL. IndexNow gets the URLs inside it. |
| `--cloudflare-zone <id\|auto>` | none | Publish the DNS records to Cloudflare. Without it, the CLI prints each record and waits. |
| `--wait <duration>` | `10m` | How long each verify step waits for DNS. Must not be negative. |
| `--interval <duration>` | `5s` | Delay between checks. Must be more than zero. |
| `--key-dir <dir>` | `.` | Existing directory where the IndexNow key file is written. |
| `--key-location <url>` | `https://<domain>/<key>.txt` | URL where the key file is served. |
| `--indexnow-urls <file>` | none | File with one URL per line. Use it instead of the sitemap for IndexNow. |
| `--skip <groups>` | none | Leave out `google`, `bing` or `indexnow`. A skipped group is not checked. |

### First launch and the IndexNow key file

On the first launch, the `indexnow` step makes a key and writes `<key>.txt` to `--key-dir`. IndexNow reads that file from your site before it accepts URLs. Serve the file at `https://<domain>/<key>.txt` before the CLI pings IndexNow. Until then, the `indexnow` step fails with `the key file is not reachable`. The other steps still pass.

1. Run `launch`. The log shows the key file path.
2. Deploy that file with your site, so `https://<domain>/<key>.txt` answers 200 with the key.
3. Run `launch` again. The finished steps pass again. The `indexnow` step sends the URLs.

### Dry run

`launch --dry-run` runs no step. Every step shows `skipped (dry-run)` and what it would do. The last line is `dry run: nothing was changed or verified`. In `--json`, each step has `"status": "skipped"`, and the top level has `"verified": false`.

A dry run still runs the read-only checks for real: the Google login, the Bing key, the Cloudflare token, a request for the sitemap, and a request for the IndexNow key file. A failed check shows `fail` for its step and exits with the code of the real run: 1, or 3 for a sitemap with problems. The key file check waits for the first real run, because the key does not exist before then.

## Commands

| Command | What it does |
|---|---|
| `auth google` | Log in with the browser loopback flow. |
| `auth bing` | Check the Bing API key. |
| `auth status [--check]` | Show which credentials are set. |
| `google verify <domain> [--cloudflare-zone <id\|auto>]` | Verify the domain over a DNS TXT record, then add it to Search Console. |
| `google sitemap submit\|status <domain> <sitemap-url>` | Submit a sitemap and read its status. |
| `bing verify <domain> [--cloudflare-zone <id\|auto>]` | Verify the domain over a DNS CNAME record. |
| `bing sitemap submit\|status <domain> <sitemap-url>` | Submit a sitemap and read its status. |
| `bing quota <domain>` | Show the URL submission quota. |
| `indexnow submit <domain> (--urls <file>\|--from-sitemap[=url])` | Send URLs in batches of up to 10,000. |
| `sitemap check <sitemap-url>` | Check a sitemap before you submit it. |
| `launch <domain> --sitemap <url> [flags]` | Run all the steps above in order. See the flag table above. |

Every command accepts `--json` and `--dry-run`. With `--dry-run`, the CLI prints the HTTP calls and sends none of them. A dry run never reports a pass for work it did not do. Secrets do not appear in the output.

## Environment

| Variable | Use |
|---|---|
| `BING_WEBMASTER_API_KEY` | Bing API key. |
| `CLOUDFLARE_API_TOKEN` | Cloudflare token. Read from the environment only. |
| `GOOGLE_ACCESS_TOKEN`, `GOOGLE_REFRESH_TOKEN`, `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` | Google credentials. They win over `google.json`. |
| `SEARCH_CONSOLE_CONFIG_DIR` | Config directory. The default is `~/.config/search-console`. |
| `GOOGLE_API_BASE`, `BING_API_BASE`, `CLOUDFLARE_API_BASE`, `INDEXNOW_API_BASE` | Point a service at another server, for example a test server. |
| `GOOGLE_OAUTH_AUTH_URL`, `GOOGLE_OAUTH_TOKEN_URL` | Point the Google login at another server. |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success. |
| 1 | A request or a step failed. |
| 2 | Usage error, such as a bad domain or a missing flag. |
| 3 | The sitemap has problems, or a service reported sitemap errors. |
| 4 | DNS was not ready before `--wait` ended. Run the command again. |

## Test

```bash
make e2e
```

The target runs `go vet`, runs every E2E test and writes `e2e-report.txt`. The report lists each test and one full launch with its rerun. Two runs give the same file.
