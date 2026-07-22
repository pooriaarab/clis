# sites blog + sections move/hide

Hand-authored `internal/cli/sites_blog.go` and
`internal/cli/sites_sections_order.go` (+ AddCommand lines in `sites.go` and
`sites_sections.go`). Reuse the codec + shared websiteSettings/section helpers.

- `sites blog list|add|rm` — `websiteSettings.pregeneratedBlogPosts` (each
  BlogPost `{id,title,body,image,imageKeywords,authorName,date,isAIAssisted}`).
  `add` defaults `date` to today and `isAIAssisted` to false.
- `sites sections move <siteId> <sectionId> --page /p --to N` — reorder the
  section within a page's `sectionIds` (render order is per-page).
- `sites sections hide <siteId> <sectionId>` — remove it from every page's
  `sectionIds` while keeping it in `websiteSettings.sections`.

All support `--dry-run` and `--json`.
