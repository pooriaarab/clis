# sites publish

Hand-authored `internal/cli/sites_publish.go` (+ one `AddCommand` line in
`sites.go`). Adds `sites publish <siteId>`.

## Why it's not a Firestore write

Every other `sites`/`draft-websites` command mutates Firestore directly.
Publish is different: the live site is produced by a server route, not a
plain document copy. The command:

1. GETs the Firestore-typed `DraftWebsites/{id}` doc.
2. Decodes its `handle` and `websiteSettings` to plain JSON with the shared
   `fsDecode` codec (the publish route wants plain JSON, not Firestore typed
   values).
3. Builds a second client with `BaseURL` overridden to `https://soloist.ai`
   (the generated client's base is the Firestore host) and POSTs
   `{handle, websiteSettings, draftId}` to `/api/websites/publish`.

Auth is the same Firebase ID token the rest of the CLI uses — the publish
route authenticates with the `Authorization: Bearer` header, no session
cookie.

## On regen

`sites_publish.go` has no generated header, so it survives regen. Re-add the
`cmd.AddCommand(newSitesPublishCmd(flags))` line in `sites.go` if root/parent
wiring is regenerated from scratch.
