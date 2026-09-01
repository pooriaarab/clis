# clis brand

## Purpose

`pooriaarab/clis` collects runnable command-line clients for APIs that lack suitable official tools.

The collection complements `pooriaarab/skills`. Skills explain workflows, while this repository contains executable source.

Each top-level product directory is an independent Go module. Its README defines setup, credentials, coverage, and current limits.

## Audience

The tools serve terminal users, scripts, and software agents. Output must remain useful without a graphical interface.

Several full printed CLIs also ship MCP binaries. Smaller modules may provide only a CLI binary.

## Promise

- Ground API behavior in an official specification or documented service contract.
- Expose missing coverage honestly instead of guessing endpoints.
- Make automated use explicit through machine-readable output and non-interactive controls.
- Keep each module independently buildable.
- Record changes to generated trees in that module's `.printing-press-patches/` directory.

## Voice

Use direct, factual language. Name the command, required input, side effect, and result.

Use the platform's own resource names. Explain access gates and incomplete coverage without promotional claims.

Error text must state what failed and the next safe action. Help text must not promise unimplemented behavior.

## Product families

Full printed CLIs provide broad command trees and richer agent controls. Examples include `google-ads`, `meta-ads`, and `posthog`.

Focused CLIs implement narrower service workflows. Examples include `google-analytics` and `buzz`.

Modules whose `.printing-press.json` sets `mode` to `stub` expose local setup and auth commands only. They do not imply API coverage.

## Boundaries

This repository has no website, shared graphical interface, or custom visual mark.

Do not transfer a service provider's brand into this collection. Keep provider names descriptive and subordinate to each tool's function.
