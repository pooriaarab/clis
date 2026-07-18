# Generic resource mutate command

Added `internal/cli/mutate_resource.go` (hand-written, not printed) and one
`rootCmd.AddCommand(newMutateResourceCmd(flags))` line in `root.go`.

Why: the promoted commands (campaigns, ad-groups, ad-group-ads,
campaign-budgets, assets, conversion-actions) each hardcode one resource
against `POST /v24/customers/{customerId}/{resourcePlural}:mutate` with a
`{"operations":[...]}` body. That shape is identical for every one of the
~65 mutable Google Ads REST resources. `mutate <resourcePlural> <customerId>`
generalizes it so every mutable resource is reachable without printing 65
near-duplicate wrapper commands from doc-scraping (each an LLM pass and a
schema-hallucination risk on a money-touching API).

On regen: re-add the `AddCommand` line if root.go is regenerated from
scratch instead of merged; the command file itself doesn't depend on any
generated symbol beyond `rootFlags`, `replacePathParam`, `classifyAPIError`,
`detectPartialFailure`, `writeMutationResponseToStore`, `attachFreshness`,
`DataProvenance`, `printProvenance`, `wantsHumanTable`, `isTerminal`,
`filterFields`, `compactFields`, `wrapWithProvenance`, `printOutput`,
`printOutputWithFlags`, `partialFailureErr` — all present in `internal/cli`
since the first print.
