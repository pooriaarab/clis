# Semantic sites section commands

## Why this patch exists

The generated Soloist CLI exposes raw Firestore document CRUD for
`DraftWebsites`, but common website edits need semantic operations over
`websiteSettings.sections` and `websiteSettings.pages`. The website builder
stores renderable sections in `websiteSettings.sections` and attaches them to
pages by adding section ids to each page's `sectionIds` array, so a safe CLI
edit must update both arrays together.

This generated tree can be reprinted, so all new command and codec code lives
in hand-authored files without the CLI Printing Press generated-file header.
The generated `root.go` already has the preserved `novelCommandHooks`
registration point; `internal/cli/sites.go` uses that hook to add the `sites`
command without hand-editing generated command families.

## What changed

- `internal/cli/firestore_value.go`: adds a small Firestore REST typed-value
  codec for `stringValue`, `integerValue`, `doubleValue`, `booleanValue`,
  `nullValue`, `timestampValue`, `mapValue`, and `arrayValue`.
- `internal/cli/firestore_value_test.go`: covers codec round-tripping for a
  nested map/array/string/int/bool/null payload.
- `internal/cli/sites.go`: registers the hand-authored `sites` command group
  through `registerNovelCommand`.
- `internal/cli/sites_sections.go`: implements `sites sections list`,
  `sites sections add`, and `sites sections rm` against the
  `DraftWebsites/{siteId}` Firestore REST document. Mutations PATCH only
  `websiteSettings.sections` and `websiteSettings.pages`.

## Reapply notes

After a future CLI Printing Press regeneration:

1. Restore the hand-authored files listed above.
2. Confirm `root.go` still calls every `novelCommandHooks` registration during
   root construction; if that hook is lost, re-add the hook loop or register
   `newSitesCmd(flags)` through the current preserved-extension mechanism.
3. Verify `sites sections add <siteId> --type CodeEmbed --code '<b>hi</b>'
   --dry-run` prints a CodeEmbed section and a PATCH body without fetching the
   Firestore document.
