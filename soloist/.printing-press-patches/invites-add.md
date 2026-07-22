# invites add

Hand-authored `internal/cli/invites_add.go` (+ one `AddCommand` line in the
generated `invites.go`). Semantic wrapper over the raw `invites create`.

`invites add <websiteId> <email> [--role Viewer|Collaborator|Admin|Owner]`
builds a correct `WebsiteInvite` document — `{id, websiteId, invitedUserEmail,
role, accepted:false, createdAt}` — and createDocument-POSTs it to the
`WebsiteInvites` collection (creating the doc is the invite). Role defaults to
`Collaborator`. `--dry-run`/`--json` supported.

## Note on domains

There is deliberately no `domains connect` semantic wrapper. A `WebsiteDomain`
is not a plain doc create: connecting a custom domain runs a server-side
ownership-verification + certificate pipeline (states
PendingOwnershipVerification → OwnershipVerified → Connected). Writing a raw
`WebsiteDomains` doc would not verify or connect anything. Real connection
needs the Solo `/api` domain route (same server-feature bucket as publish's
backend). The raw `domains list/get/create/delete` commands remain available
for inspecting/removing domain records.
