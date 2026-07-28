---
status: testing
phase: 01-first-webspace-end-to-end
source: [01-VERIFICATION.md]
round: 2
started: 2026-07-28T12:36:00Z
updated: 2026-07-28T12:36:00Z
---

## Current Test

number: 1
name: Re-run UAT tests 2, 3, 4 in the browser now that G-01-2 (CSS delivery) is fixed
expected: |
  Detail pane sits BESIDE the stream (never stacked below), title/date/tags appear
  instantly with a skeleton-then-fill preview, scroll containment is per-region,
  stream rows are fixed-height (152px) small-thumbnail cards with ellipsised titles
  and two-line clamped snippets, and the three original UAT symptoms (full-size
  centered images, unformatted stacked text, whole-page scrolling) are gone.
awaiting: user response

## Tests

### 1. Re-run UAT tests 2, 3, 4 — browser rendering after CSS fix
test: Hard-reload (Ctrl+Shift+R) http://127.0.0.1:7777/ (server already running with the styled build). Click a document row to open the detail pane; scroll the PDF/extracted text while watching the stream list; click "Open in paperless-ngx"; open a webspace row with several tags and a long title; check a no-OCR-text document; scroll the stream with the detail pane open; try an 80+ character webspace name; point base_url at an unreachable host and confirm the sync-error state.
expected: Per 01-05-PLAN.md Task 3's human-check block — detail pane beside the stream, instant metadata with skeleton-then-fill preview, per-region scroll containment, fixed-height small-thumbnail cards with one-line ellipsised titles and two-line clamped snippets, "Nothing here yet" / truncated-name-tooltip / sync-error states render correctly; the three original symptoms (full-size centered images, unformatted stacked text, whole-page scrolling) are gone.
result: [pending]

### 2. AGENT-02 / concurrency backstop (carried forward)
test: Issue GET /api/webspaces/house-move/stream repeatedly in a tight loop while `webspaces sync` runs concurrently; diff each response against the two known-good pre/post item sets.
expected: Every response matches either the pre-sync or post-sync set exactly, never a partial mix (SQLite WAL snapshot isolation).
result: [pending]

### 3. UI / error stream-list backstop (carried forward)
test: Stop `webspaces serve` (or block port 7777) and load a webspace route in the browser; confirm the error state, then restart the kernel and retry.
expected: "Couldn't load this webspace" copy with a working retry control that recovers the stream once the kernel is back.
result: [pending]

## Summary

total: 3
passed: 0
issues: 0
pending: 3
skipped: 0
blocked: 0

## Gaps

Prior-round gaps (round 1: 2 passed, 4 issues) — both closed by gap-closure plans and independently re-verified in 01-VERIFICATION.md:

- gap_id: G-01-2
  truth: "Detail pane opens beside the stream; stream rows render as designed fixed-height cards with small thumbnails, formatted title/date/tags, two-line clamped snippet"
  status: resolved
  severity: major
  test: 2, 3, 4
  resolved_by: 01-05 (import '../app.css' in +layout.svelte; smoke-test stylesheet + stale-listener guards)
  note: "CSS provably ships (33,334-byte asset, all tokens/selectors confirmed, served live); visual confirmation queued as round-2 test 1"
- gap_id: G-01-6
  truth: "A committed, wired test enforces that no code path transmits data to any host other than the configured paperless-ngx base_url and loopback"
  status: resolved
  severity: minor
  test: 6
  resolved_by: 01-06 (host-pinned CheckRedirect/DialContext with ErrForeignHost; internal/audit repo-wide AST egress test, non-vacuous via fixture)
  note: "7 committed tests across two files re-run and passing; 01-01-PLAN.md prohibition now carries verified_by"
