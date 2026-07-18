# Generic node/edge commands

Added `internal/cli/generic_node.go` (hand-written, not printed) and two
`rootCmd.AddCommand(...)` lines in `root.go` (`newGenericNodeCmd`,
`newGenericEdgeCmd`).

Why: Meta's Graph API is node/edge shaped — every object is reachable at
GET/POST/DELETE /{node-id}, every relationship at GET/POST /{node-id}/{edge}.
The spec's generic `/{node_id}` and `/{node_id}/{edge}` paths have no
derivable resource name, so cli-printing-press's generator skips them (same
failure mode the Google Ads CLI hit on ~59 of its ~65 mutable resources —
see mutate_resource.go in that project). `node get/update/delete <id>` and
`edge get/create <id> <edgeName>` generalize the escape hatch so every Graph
API object and edge this CLI doesn't have a dedicated command for (pages,
posts, page insights, lead-gen forms, product catalogs, ...) stays reachable
without printing one wrapper per object type.

On regen: re-add the two `AddCommand` lines if root.go is regenerated from
scratch instead of merged; the command file itself doesn't depend on any
generated symbol beyond `rootFlags`, `classifyAPIError`,
`writeMutationResponseToStore`, `attachFreshness`, `DataProvenance`,
`printProvenance`, `wantsHumanTable`, `isTerminal`, `filterFields`,
`compactFields`, `wrapWithProvenance`, `printOutput`, `printOutputWithFlags`,
`printAutoTable` — all present in `internal/cli` since the first print.
