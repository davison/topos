---
status: testing
phase: 09-ui-polish-and-source-management-rework
source: [09-VERIFICATION.md]
started: 2026-08-11T17:35:00Z
updated: 2026-08-11T17:35:00Z
---

## Current Test

number: 1
name: Plugin icon legibility at shipped sizes (14-16px, dark palette)
expected: |
  With the kernel's `[plugins] dir` pointed at freshly-built plugin binaries from
  this codebase (not a stale external bin/plugins/), open a webspace with at least
  one instance of each plugin type configured and view the chip row, the "+"
  picker, the Manage Sources rows, and the stream/search row icons. Every plugin's
  icon is legible and visually distinguishable from the others at 14-16px — the
  real paperless-ngx and SilverBullet marks read as their upstream logos, and the
  Lucide-derived Mail/MessageCircle/MessageSquare/FlaskConical glyphs read as
  recognizably distinct shapes, not a blur.
awaiting: user response

## Tests

### 1. Plugin icon legibility at shipped sizes (14-16px, dark palette)
expected: With the kernel's `[plugins] dir` pointed at freshly-built plugin binaries from this codebase (rebuild first: `make build`; check `~/.config/topos/config.toml` per README's Development loop note), every plugin's icon is legible and distinguishable at 14px (chips, stream/search rows) and 16px ("+" picker, Manage Sources rows) on the dark palette — paperless-ngx and SilverBullet read as their upstream marks; Proton/Signal/WhatsApp/mock glyphs read as distinct shapes.
result: [pending]

### 2. Upstream trademark/brand-policy recheck (paperless-ngx, SilverBullet)
expected: Confirmation that no separate trademark/brand-usage policy (beyond the GPL-3.0/MIT code licenses) prohibits embedding the paperless-ngx and SilverBullet logo marks in third-party software — or a decision to swap a restricted mark for a generic glyph. Provenance pointers live in `plugins/paperless/plugin.go` and `plugins/silverbullet/plugin.go`.
result: [pending]

## Summary

total: 2
passed: 0
issues: 0
pending: 2
skipped: 0
blocked: 0

## Gaps
