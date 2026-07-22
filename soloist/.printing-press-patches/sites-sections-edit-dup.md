# sites sections edit / dup

Hand-authored `internal/cli/sites_sections_edit.go` (+ two `AddCommand` lines
in `sites_sections.go`). Reuses the section helpers already there
(`loadDraftWebsiteState`, `findSectionIndex`, `buildSitesSectionsPatchBody`,
`draftWebsitePatchPath`) and the codec.

- `sites sections edit <siteId> <sectionId> --set prop=value` — merges the
  given keys into that section's `props` map (e.g. `--set title=... --set
  code=...`), then PATCHes `websiteSettings.sections`.
- `sites sections dup <siteId> <sectionId>` — deep-copies the section with a
  new UUID, inserts it right after the original in `websiteSettings.sections`,
  and inserts the new id right after the original in every page's `sectionIds`.

Both support `--dry-run` and `--json`. `edit` verified live.
