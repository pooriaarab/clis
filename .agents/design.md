# clis design

## Overview

This document covers the collection's terminal interface. Each top-level directory owns its command tree and runtime contract.

Repository patterns form three families: full printed CLIs, focused CLIs, and setup-only stubs.

Apply a rich feature only where the module implements it. Never infer support from a sibling module.

## Colors

Full printed CLIs keep color off unless `--human-friendly` enables it. `--no-color`, `--agent`, and `NO_COLOR` suppress color.

Those CLIs use four ANSI treatments:

- Bold marks table headers and labels.
- Green marks successful diagnostic states.
- Red marks failed diagnostic states.
- Yellow marks warnings and informational diagnostic states.

Focused CLIs and stubs may return plain text or JSON without these treatments. Do not add color without a tested plain mode.

## Typography

The terminal controls the font. The repository does not bundle or require a typeface.

Use sentence case for descriptions. Use exact command, flag, path, and environment names in code spans.

Keep help text scannable. Put the command purpose first, then inputs, defaults, and side effects.

## Layout

Use a noun-led command tree. Put the action below its resource when the module supports nested commands.

Keep stable output on standard output. Send diagnostics and warnings to standard error when code paths support that split.

Full printed CLIs offer human tables and structured modes. Their `--agent` flag selects JSON, compact, non-interactive, colorless defaults.

Those CLIs can separate config, data, state, and cache paths. Their `--home` flag relocates all four path kinds.

Other modules own different flags and paths. Their README and `--help` output remain authoritative.

## Elevation & Depth

There is no graphical elevation. Command nesting communicates depth.

The root names the product. Resource groups form the next level. Actions and identifiers follow beneath them.

Avoid deep trees when one resource and action are clear. Expose raw API access only as a documented escape hatch.

## Shapes

The interface uses text boundaries, not graphic shapes.

- Code blocks contain commands and examples.
- Tables compare resources or returned fields.
- JSON objects preserve machine-readable response structure.
- Bracketed arguments in help text show optional values.

Do not use decorative boxes, banners, or ASCII art in machine-readable output.

## Components

Every module has a CLI entry point and its own Go module. Common components vary by product family.

- Cobra commands define roots, groups, actions, flags, and help.
- `doctor` reports local setup and connectivity where implemented.
- `--dry-run` previews requests where implemented.
- Full printed CLIs can provide `agent-context`, `which`, profiles, local learning, and MCP tools.
- Stub modules expose setup and auth status while their API command groups remain disabled.
- Module READMEs define credentials, verified coverage, build commands, and known gaps.

Generated modules treat `.printing-press.json` as print metadata. Local changes belong in `.printing-press-patches/` when that directory exists.

## Do's and Don'ts

Do inspect a module's README and `--help` before use.

Do use `--agent` for automation only when that module defines it.

Do prefer `--dry-run` before a supported write command.

Do keep JSON field names stable and output free of terminal styling.

Do document missing access or endpoints as an explicit limit.

Don't claim one module's flags, storage, MCP server, or learning loop for the whole collection.

Don't guess provider endpoints, scopes, resources, or response fields.

Don't expose secrets in output, examples, fixtures, or logs.

Don't edit generated source without recording durable patch intent.
