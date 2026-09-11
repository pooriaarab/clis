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
canadabuys tenders list [--json] [--status S] [--category C] [--unspsc U] [--gsin G]
                         [--org O] [--region R] [--notice-type N] [--q Q]
                         [--since D] [--until D] [--closing-after D] [--closing-before D]
                         [--sort FIELD] [--limit N] [--dataset ID]
canadabuys tenders show <referenceNumber> [--json] [--dataset ID]
canadabuys awards list [--json] [--supplier S] [--org O] [--unspsc U] [--category C]
                        [--currency C] [--since D] [--until D]
                        [--min-amount N] [--max-amount N] [--limit N]
```

`awards list` reads only the `awards` dataset; `awards-legacy` is fetchable but not yet queryable.

Kinds: `tenders`, `awards`, `contracts`, `reference`.

Cache: `$XDG_CACHE_HOME/canadabuys`, else `~/.cache/canadabuys`. Override with `--cache-dir` or `CANADABUYS_CACHE_DIR`.

`--json` prints machine output on stdout only. Progress goes to stderr.
