# sites theme / settings / pages

Hand-authored `internal/cli/sites_settings.go` and `internal/cli/sites_pages.go`
(+ three `AddCommand` lines in `sites.go`). No generated header, so they
survive regen; re-add the `AddCommand` lines if `sites.go` is regenerated.

All of a website's content lives under `DraftWebsites/{id}.websiteSettings`.
These commands reuse the `fsDecode`/`fsEncode` codec and share two helpers in
`sites_settings.go`:

- `loadWebsiteSettings` — GET the draft, decode `websiteSettings` to a plain map.
- `patchWebsiteSettingsFields` — PATCH one-or-more top-level `websiteSettings`
  sub-fields with an `updateMask` scoped to exactly those paths (everything
  else is left untouched).

Commands:
- `sites theme get|colors|fonts|style` — `colorScheme`, `fonts`,
  `buttonCornerStyle`/`imageCornerStyle`/`bottomBorderStyle`.
- `sites settings get|business|language|ga4|seo|head-code|social` —
  `businessName`/`businessActivities`/`businessLocation`, `language`,
  `gaMeasurementId`, `indexForSearch`, `codeContent` ({enabled,code}),
  `socialMediaLinks`.
- `sites pages list|create|rm|set` — `websiteSettings.pages` (each
  `{id,title,path,keywords,sectionIds}`).

Every subcommand supports `--dry-run` (renders the PATCH body, no network) and
`--json`.
