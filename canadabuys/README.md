# CanadaBuys CLI

Fetch and cache Government of Canada procurement CSVs. No auth, no API key.

Files live at https://canadabuys.canada.ca/opendata/pub/. Catalog metadata is on the CKAN API at https://open.canada.ca/data/en/api/3/action/package_show?id=<id>. Do not scrape the JavaScript tender site.

## Install

```
go build -o canadabuys ./cmd/canadabuys
```

## Commands

```
canadabuys datasets list [--json] [--kind K]
canadabuys fetch [id...] [--all] [--force] [--kind K]
canadabuys doctor [--json]
```

Kinds: `tenders`, `awards`, `contracts`, `reference`.

Cache: `$XDG_CACHE_HOME/canadabuys`, else `~/.cache/canadabuys`. Override with `--cache-dir` or `CANADABUYS_CACHE_DIR`.

`--json` prints machine output on stdout only. Progress goes to stderr.
