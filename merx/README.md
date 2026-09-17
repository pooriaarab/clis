# MERX CLI

Skeleton for the [MERX](https://www.merx.com/) procurement portal. No auth. Do
not scrape the HTML site; commands here only check reachability.

No search, no login, no harvest yet. Those land on their own issues.

## Install

You need [Go](https://go.dev/dl/) 1.22 or later.

```bash
cd merx
go build -o merx ./cmd/merx
```

## Cache directory

Resolved in this order: `--cache-dir`, then `MERX_CACHE_DIR`, then
`$XDG_CACHE_HOME/merx`, else `~/.cache/merx`.

## Crawl delay

MERX's `robots.txt` asks for `Crawl-delay: 5`. Every request through
`internal/httpx` is spaced by at least 5 seconds. `MERX_CRAWL_DELAY` can raise
that interval but is rejected below 5, so the polite floor cannot be tuned
away by accident.

```bash
$ MERX_CRAWL_DELAY=1 go run ./cmd/merx doctor
MERX_CRAWL_DELAY must be >= 5 (robots.txt Crawl-delay: 5), got 1
```

## doctor

Reports the cache location, bytes on disk, and portal reachability.

```bash
go run ./cmd/merx doctor
```

```
cache: /home/you/.cache/merx
disk: 0 bytes
reachable: true
```

Add `--json` for machine-readable output:

```bash
go run ./cmd/merx doctor --json
```

```json
{
  "bytes": 0,
  "cache_dir": "/home/you/.cache/merx",
  "host": "https://www.merx.com/",
  "ok": true,
  "reachable": true
}
```

## Development

Run `gofmt`, `go vet`, and `go build ./...` before a PR. Changes here stay
inside `merx/`.
