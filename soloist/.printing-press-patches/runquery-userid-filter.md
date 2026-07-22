# runQuery user/email filters for list commands

## Why this patch exists

The generated Soloist CLI list commands for `draft-websites`, `websites`, `domains`, and `invites` call Firestore REST v1 `documents:runQuery`. The generator emitted an empty JSON body (`{}`) for the non-`--stdin` path, but Firestore needs a `structuredQuery` body to match the Soloist website builder traffic.

This is a generated tree, so the hand-authored query logic lives in `internal/cli/soloist_query.go` without the CLI Printing Press generated-file header. Re-apply this patch after a reprint if those list commands revert to posting `{}`.

## What changed

- `internal/cli/soloist_query.go`: adds shared helpers for Firestore single string `EQUAL` filters, WebsiteInvites website/email queries, and uid/email decoding from the loaded bearer JWT claims (`user_id` with `sub` fallback, plus `email`).
- `internal/cli/draft-websites_list.go`: adds `--uid` and sends `DraftWebsites` where `userId == <uid>`.
- `internal/cli/websites_list.go`: adds `--uid` and sends `Websites` where `userId == <uid>`.
- `internal/cli/domains_list.go`: adds `--uid` and sends `WebsiteDomains` where `userId == <uid>`.
- `internal/cli/invites_list.go`: adds `--website-id`, `--email`, and `--include-pending`; sends `WebsiteInvites` where `websiteId == <id>` when `--website-id` is provided, otherwise filters by `invitedUserEmail == <email>` and `accepted == true` unless pending invites are included.

The `--stdin` branches remain the generated behavior: stdin JSON is used as the full request body and is not replaced by these helpers.

## Reapply notes

After a future CLI Printing Press regeneration:

1. Restore `internal/cli/soloist_query.go` as a hand-authored package `cli` file.
2. In each of the four list files, keep the generated stdin branch intact and replace only the non-stdin `bodyMap := map[string]any{}` with the helper call for that collection.
3. Re-add the override flags on the list commands.
4. Verify with dry-runs that the printed request body contains `structuredQuery` and that `echo '{}' | soloist-pp-cli draft-websites list --stdin` still posts `{}`.
