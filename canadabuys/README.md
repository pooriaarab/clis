# CanadaBuys CLI

Query Government of Canada procurement data from your terminal. No auth, no API
key. You download the open CanadaBuys CSV files, then search them locally.

The web UI shows only notices that are open right now. That is 913 notices out
of 101,212. This CLI exists so you can search all of them, plus award notices.

## Install

You need [Go](https://go.dev/dl/) 1.22 or later.

```bash
export CANADABUYS_CACHE_DIR=/tmp/cb-cache
cd canadabuys
go build -o canadabuys ./cmd/canadabuys
```

Set `CANADABUYS_CACHE_DIR` before every command in this file. The default cache
is `$XDG_CACHE_HOME/canadabuys`, else `~/.cache/canadabuys`. You can also pass
`--cache-dir` per command.

## Fetch the data

List the dataset registry first. It shows what is cached and how old it is.

```bash
export CANADABUYS_CACHE_DIR=/tmp/cb-cache
go run ./cmd/canadabuys datasets list
```

```
ID                KIND       CACHED  AGE       BYTES      TITLE
open-tenders      tenders    true    2h32m20s  6573338    Open tender notices
new-tenders       tenders    false   -         -          New tender notices, published today
tenders           tenders    true    2h23m54s  181723440  Tender notices, 2022-08 onward
tenders-legacy    tenders    true    2h23m36s  558413231  Tender notices, 2009 to 2022
awards            awards     true    2h0m33s   99559059   Award notices, 2022-08 onward
awards-legacy     awards     false   -         -          Award notices, 2012 to 2022-08
contracts         contracts  true    2h0m31s   57669819   Contract history, 2023-06 onward
contracts-legacy  contracts  false   -         -          Contract history, 2009-01 to 2023-05
gsin-unspsc       reference  true    2h0m30s   1546841    Mapping of GSIN to UNSPSC codes
```

Download what you need. `fetch` sends `If-Modified-Since`, so it skips files
that did not change, and it writes to a temp file with a SHA-256 hash before it
renames into place. It does not resume partial downloads; an interrupted fetch
starts over.

```bash
go run ./cmd/canadabuys fetch tenders awards
go run ./cmd/canadabuys fetch --kind reference
go run ./cmd/canadabuys fetch gsin-unspsc --force
```

Add `--all` for every dataset. Without `--force`, `fetch` skips unchanged files.

Check cache state and reachability with `doctor`:

```bash
go run ./cmd/canadabuys doctor
```

```
cache: /tmp/cb-cache
disk: 905487676 bytes
reachable: true
open-tenders  cached  2h32m20s  6573338
new-tenders  missing
tenders  cached  2h23m54s  181723440
tenders-legacy  cached  2h23m36s  558413231
awards  cached  2h0m33s   99559059
awards-legacy  missing
contracts  cached  2h0m31s  57669819
contracts-legacy  missing
gsin-unspsc  cached  2h0m30s  1546841
```

`reachable` is a HEAD request to `https://canadabuys.canada.ca/`. Add `--json`
to any command for machine output on stdout. Progress goes to stderr.

## What is in the cache

The two archive files hold 101,212 tender notices: 29,902 from 2022-08 onward
and 71,310 from 2009 to 2022. The open file holds 913 notices, and 894 of them
also appear in the archive. Award notices start in 2022-08: 27,669 rows, of
which 12,637 carry a positive amount and 7,670 of those sit under $250,000
(60.7%). Audit these counts yourself with plain Python. Note that `wc -l`
overcounts: multi-valued cells hold embedded newlines, so parse the CSV.

```bash
export CANADABUYS_CACHE_DIR=/tmp/cb-cache
python3 - <<'EOF'
import csv, os
d = os.environ['CANADABUYS_CACHE_DIR']
def rd(f):
    return open(d + '/' + f, newline='', encoding='utf-8-sig')
for f in ['open-tenders.csv', 'tenders.csv', 'tenders-legacy.csv', 'awards.csv']:
    print(f, sum(1 for _ in csv.reader(rd(f))) - 1)
n = mu = mg = 0
for f in ['tenders.csv', 'tenders-legacy.csv']:
    r = csv.DictReader(rd(f))
    u = [k for k in r.fieldnames if k == 'unspsc'][0]
    g = [k for k in r.fieldnames if 'gsin-nibs' in k][0]
    for row in r:
        n += 1
        mu += '\n' in (row[u] or '')
        mg += '\n' in (row[g] or '')
print('archive rows:', n, 'multi-UNSPSC:', mu, 'multi-GSIN:', mg)
def refs(f):
    r = csv.DictReader(rd(f))
    c = [k for k in r.fieldnames if 'referenceNumber' in k][0]
    return set(row[c] for row in r)
print('open notices also in tenders.csv:', len(refs('open-tenders.csv') & refs('tenders.csv')))
r = csv.DictReader(rd('awards.csv'))
a = [k for k in r.fieldnames if 'mount' in k.lower()][0]
amt = [float(row[a].replace(',', '')) for row in r if (row[a] or '').replace(',', '').replace('.', '', 1).lstrip('-').isdigit()]
priced = [x for x in amt if x > 0]
print('priced:', len(priced), 'under $250k:', sum(x < 250000 for x in priced))
EOF
```

```
open-tenders.csv 913
tenders.csv 29902
tenders-legacy.csv 71310
awards.csv 27669
archive rows: 101212 multi-UNSPSC: 8045 multi-GSIN: 10760
open notices also in tenders.csv: 894
priced: 12637 under $250k: 7670
```

## Query tenders

`tenders list` reads `tenders` and `tenders-legacy` by default. It streams, so
`--limit` stops the scan early. Filters match any member of a multi-valued
cell (see below).

```bash
go run ./cmd/canadabuys tenders list --status Open --limit 5
```

```
REFERENCE        STATUS  PUBLISHED   CLOSING     ORG                                            TITLE
MX-443841357513  Open    2026-01-23  2029-03-31  Defence Construction Canada - Atlantic Region  Open Construction Source List for CFB Halifax
MX-443841357514  Open    2026-01-23  2029-03-31  Defence Construction Canada - Atlantic Region  Open Construction Source List for 9 Wing Gander
MX-443841357515  Open    2026-01-23  2029-03-31  Defence Construction Canada - Atlantic Region  Open Construction Source List for 5 Wing Goose Bay
MX-443841876264  Open    2026-01-23  2029-03-31  Defence Construction Canada - Atlantic Region  Open Construction Source List for CDSB Gagetown and P.E.I.
MX-443841876266  Open    2026-01-23  2029-03-31  Defence Construction Canada - Atlantic Region  Open Construction Source List for 14 Wing Greenwood
```

More filters:

```bash
go run ./cmd/canadabuys tenders list --unspsc "*77121608" --limit 3
go run ./cmd/canadabuys tenders list --q bridge --since 2024-01-01 --limit 10
go run ./cmd/canadabuys tenders list --org "Shared Services" --category "*SRV" --limit 10
go run ./cmd/canadabuys tenders list --dataset open-tenders --status Open --limit 10
go run ./cmd/canadabuys tenders list --sort -publication --limit 10
```

Show one notice in full:

```bash
go run ./cmd/canadabuys tenders show cb-223-38486320
```

```
Reference:         cb-223-38486320
Solicitation:      CSC2627-0278
Title:             AAFC REQUEST FOR SUPPLY ARRANGEMENT - PMC Analysis of Pesticide Residues
Status:            Open
Categories:        *SRV
UNSPSC:            *10191500; *77121608
```

The `--unspsc "*77121608"` filter above finds this row even though its UNSPSC
cell holds two codes. That is match-any behavior.

## Query awards

`awards list` reads the `awards` dataset. Filter by amount band, supplier, org,
code, currency, or date.

```bash
go run ./cmd/canadabuys awards list --min-amount 1000000 --limit 5
```

```
AWARDED     REFERENCE       SUPPLIER                                             ORG                       CURR  AMOUNT
2024-02-29  MX-43049679057  Lightning Tree Consulting Inc.                       Farm Credit Canada (FCC)  CAD   $1,500,000
2024-02-29  MX-43049887844  Oona Stock                                           Farm Credit Canada (FCC)  CAD   $1,500,000
2024-04-01  MX-43089535758  Deloitte MAIN ACCOUNT                                Farm Credit Canada (FCC)  CAD   $1,500,000
2024-04-01  MX-43089535759  KPMG LLP                                             Farm Credit Canada (FCC)  CAD   $1,500,000
2024-04-01  MX-43089535760  PricewaterhouseCoopers LLP - MAIN PwC NATIONAL ACCT  Farm Credit Canada (FCC)  CAD   $1,500,000
```

```bash
go run ./cmd/canadabuys awards list --currency CAD --max-amount 100000 --limit 3
go run ./cmd/canadabuys awards list --supplier "Chantier Davie" --limit 5
```

`awards-legacy` is fetchable but not queryable yet.

## Award statistics

```bash
go run ./cmd/canadabuys stats awards --json | head -20
```

```json
{
  "rows": 27669,
  "priced": 12637,
  "blank": 2924,
  "zero": 11599,
  "negative": 509,
  "invalid": 0,
  "total": "$36.54B",
  "total_cents": 3653584807389,
  "currency": "mixed",
  "mean": "$2,891,180",
  "mean_cents": 289118051,
  "median": "$152,465",
  "median_cents": 15246578,
  "p25": "$50,887",
  "p25_cents": 5088778,
  "p75": "$600,000",
  "p75_cents": 60000000,
  "p90": "$2,232,951",
  "p90_cents": 223295155,
```

The full output adds p95 ($5,300,375), p99 ($25,571,256), max
($13,044,299,191), and a per-currency breakdown. The total mixes currencies;
pass `--currency CAD` to `awards list` to isolate one. Group sizes with
`--by category|unspsc|gsin|org|supplier|year`:

```bash
go run ./cmd/canadabuys stats awards --by category
```

```
GROUP    NOTICES  TOTAL
*GD      7073     $25,838,927,856
*SRV     5174     $9,853,949,476
*CNST    333      $806,344,609
*SRVTGD  188      $342,628,042
```

The NOTICES column sums to 12,768, more than the 12,637 priced awards. That is
correct: one award with two categories counts toward both. See below.

Rank buyers and suppliers:

```bash
go run ./cmd/canadabuys stats buyers --limit 5
go run ./cmd/canadabuys stats suppliers --limit 5
```

```
SUPPLIER                              NOTICES  TOTAL            TOP CATEGORY  CATEGORY TOTAL   SHARE
Chantier Davie Canada Inc.            4        $13,052,236,789  *GD           $25,838,927,856  50.5%
LEONARDO UK LTD                       2        $1,171,876,633   *GD           $25,838,927,856  4.5%
Sun Life Assurance Company of Canada  1        $746,698,598     *SRV          $9,853,949,476   7.6%
Vancouver Shipyards Co. Ltd.          4        $740,888,786     *GD           $25,838,927,856  2.3%
KNDS Deutschland GmbH & Co. KG        1        $653,212,475     *GD           $25,838,927,856  2.5%
```

SHARE is the supplier's fraction of its top category total. `stats awards`
over the 99 MB award file holds about 18 MB of RAM:

```bash
go build -o /tmp/cb ./cmd/canadabuys
/usr/bin/time -l /tmp/cb stats awards
```

`time -l` is macOS syntax. On Linux use `time -v` and read "Maximum resident".

## Find opportunities

`opportunities` scores notices for whether a small team could win the work. It
excludes staffing supply arrangements by default and reports how many it saw.

```bash
go run ./cmd/canadabuys opportunities --limit 10
```

```
18201 staffing vehicles seen (--exclude-staffing=false to include)
SCORE  REFERENCE                   BUYER                                                      BAND           TITLE
92.0   cb-7396-70784855            Elections Canada                                           no award data  Field Service Supply Solution (FSSS)
89.0   cb-999-99027572             Elections Canada (Elections)                               no award data  Field Service Supply Solution (FSSS)
85.7   PW-__XE-668-28944           Public Works and Government Services Canada                no award data  MPMCT Project - Projet TCGPM
85.7   PW-__XE-670-29347           Public Works and Government Services Canada                no award data  MPMCT Project - Projet TCGPM
84.8   WS4312512507-Doc4312547453  Department of Public Works and Government Services (PSPC)  no award data  RFI T8086-212558- Transport Canada's CIVIL AVIATION OVERSIGHT APPLICATION RATIONALIZATION (CAOAR)
84.8   cb-0-11990877               Department of Public Works and Government Services (PSPC)  no award data  Reliable AI Sensor Fusion for Real-World Missions
84.8   cb-237-59677578             Department of Employment and Social Development (ESDC)     no award data  REQUEST FOR INFORMATION  FOR A USER TESTING AND RESEARCH PLATFORM FOR EMPLOYMENT AND SOCIAL DEVELOPMENT CANADA
84.8   cb-355-84944279             Destination Canada (DC)                                    no award data  Canadian Specialist Program Provider
84.8   cb-4391-98358496            Innovation, Science and Economic Development Canada        no award data  Specialized Investigative Management Software
```

Refine the ranking with `--min-score`, `--category`, `--min-award`,
`--max-award`, or `--since`. Ask why a notice scored the way it did:

```bash
go run ./cmd/canadabuys opportunities --explain cb-7396-70784855
```

```
reference: cb-7396-70784855
title: Field Service Supply Solution (FSSS)
buyer: Elections Canada
category: *SRV
award band: no award data
staffing: false
score: 92.0
SIGNAL        RAW                         WEIGHT  SCORE  CONTRIBUTION
category-fit  unspsc *43230000            3       1.00   0.273
keyword       +5 positive, -0 negative    3       1.00   0.273
award-band    no award data               2       0.00   0.000
recurrence    2 distinct year(s)          2       0.60   0.109
competition   Competitive - Open bidding  2       1.00   0.182
incumbency    top share 8%                1       0.92   0.084
```

`--llm` enriches the already-ranked shortlist, never the full corpus. It needs
`CANADABUYS_LLM_API_KEY` or `CEREBRAS_API_KEY`. Override the OpenAI-compatible
endpoint with `CANADABUYS_LLM_BASE_URL` (default
`https://api.cerebras.ai/v1`). `--llm-concurrency` (default 8, 1 is serial)
runs ten-notice batches in a bounded worker pool. Replies are cached under
`$CANADABUYS_CACHE_DIR/llm`.

`--lens default|horizontal|displace|bootstrap` re-ranks the same tenders with
different weights. Every lens keeps `category-fit` and `keyword` so the
result stays software, not just cheap and open. Lenses issue no model
requests, so each is free once the cache exists:

- `horizontal` favors notices solicited by many distinct departments.
- `displace` favors categories where one supplier holds most of the value;
  its table adds an INCUMBENT column, computed at the notice's own UNSPSC
  granularity rather than the whole procurement category.
- `bootstrap` favors open competitive work in the $50k-$250k band.

`--lens` cannot combine with `--llm`: a lens re-ranks cached enrichment, and
`--llm` issues new model requests, so run `--llm` on the default lens first
and re-rank with `--lens` afterward.

## Four things that surprise readers

1. The web UI shows only open notices: 913 out of 101,212. The archive files
   hold the rest. Query the archives when you research history.
2. Several columns are multi-valued. Cells hold newline-separated values, and
   8,045 archive rows carry several UNSPSC codes while 10,760 carry several
   GSINs. `tradeAgreements`, `regionsOfDelivery`, and `procurementCategory`
   work the same way. Filters match any member, and group totals count a
   multi-valued row once per member, so they sum to more than the corpus
   total. That is expected, not a bug.
3. Award notices start in 2022-08. Tender notices reach back to 2009. A
   tender-to-award join is therefore partial by construction. A missing award
   is missing, not zero: `--explain` shows "no award data", and that signal
   stays out of the score instead of counting as zero.
4. 18,201 notices are task-based staffing supply arrangements (TBIPS, SBIPS,
   and similar). They dominate any naive ranking by repeat purchasing, and
   they are rate cards rather than products. `opportunities` classifies them
   separately and excludes them by default, while still reporting the count.

One honest limitation, recorded on #55: because a missing award is excluded
rather than penalised, a notice with no award data is scored over a smaller
denominator than a notice with a mediocre award band. The ranking mixes two
populations that are not strictly comparable. Treat high scores with "no
award data" as leads, not verdicts.
