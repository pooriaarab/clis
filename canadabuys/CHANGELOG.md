# Changelog

## [Unreleased]

### Features

- Fetch and cache CanadaBuys open CSVs, with conditional re-download,
  SHA-256 check, and a dataset registry (`datasets list`, `fetch`, `doctor`)
  (bbe5b5c)
- Query cached tender notices by status, category, code, org, region, date,
  and text (`tenders list`, `tenders show`) (9d07273)
- Query cached award notices by amount band, supplier, org, code, currency,
  and date (`awards list`) (9417592)
- Award size distribution and buyer/supplier rankings (`stats awards`,
  `stats buyers`, `stats suppliers`) (b1b3cb6)
- Score tender notices for winnability by a small team, with per-signal
  explanations (`opportunities --explain`) (8d04fe9)
- Enrich the ranked shortlist with an LLM (`opportunities --llm`) (8d2124e)

---

Generated from 6 commits since the repo root.
