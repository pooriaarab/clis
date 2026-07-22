# sites ai

Hand-authored `internal/cli/sites_ai.go` (+ one `AddCommand` line in
`sites.go`). Wrapper over the Solo `/api/ai` content-generation route.

**Contract captured from the live deployed API** (the repo source is
version-skewed, so this was taken from a real onboarding generation, not
`solo-main`):

```
POST /api/ai  (bearer + Referer)
  { "docId": "<uuid>", "prompt": "<text>", "schema": "<lowercase>", "genLanguage": "english" }
  -> 200 { "result": "<string; JSON when a schema was given>" }
```

`schema` values are lowercase short names — `intro`, `services`, `unsplash`,
`quotes`, `about`, `faq`, `reviews`, `team`, `contact`, `colors`, `fonts`,
`video`, `scheduling`, `newsletter`, `blog`. `genLanguage` is a lowercase
language name (default `english`). Omit `--schema` for freeform text.

`sites ai --prompt "..." [--schema intro] [--doc-id] [--language]` prints the
result (structured JSON in schema mode). Verified live (schema `intro` returned
a headline + description). This is the single-generation building block; the
designer's full "create a website" onboarding chains ~5-15 of these across
schemas and assembles the section props client-side — a `sites generate`
orchestration on top of this remains future work.
