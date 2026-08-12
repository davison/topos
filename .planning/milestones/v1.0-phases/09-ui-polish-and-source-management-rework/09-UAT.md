---
status: complete
phase: 09-ui-polish-and-source-management-rework
source: [09-VERIFICATION.md]
started: 2026-08-11T17:35:00Z
updated: 2026-08-11T18:05:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Plugin icon legibility at shipped sizes (14-16px, dark palette)
expected: With the kernel's `[plugins] dir` pointed at freshly-built plugin binaries from this codebase (rebuild first: `make build`; check `~/.config/topos/config.toml` per README's Development loop note), every plugin's icon is legible and distinguishable at 14px (chips, stream/search rows) and 16px ("+" picker, Manage Sources rows) on the dark palette — paperless-ngx and SilverBullet read as their upstream marks; Proton/Signal/WhatsApp/mock glyphs read as distinct shapes.
result: pass

### 2. Upstream trademark/brand-policy recheck (paperless-ngx, SilverBullet)
expected: Confirmation that no separate trademark/brand-usage policy (beyond the GPL-3.0/MIT code licenses) prohibits embedding the paperless-ngx and SilverBullet logo marks in third-party software — or a decision to swap a restricted mark for a generic glyph. Provenance pointers live in `plugins/paperless/plugin.go` and `plugins/silverbullet/plugin.go`.
result: pass

## Summary

total: 2
passed: 2
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps
