---
status: testing
phase: 01-first-webspace-end-to-end
source: [01-VERIFICATION.md]
started: 2026-07-28T02:45:00Z
updated: 2026-07-28T02:45:00Z
---

## Current Test

number: 1
name: Browser walkthrough — landing page, house-move stream, SPA fallback
expected: |
  Landing page lists house-move with a non-zero item count on a near-black (#020617) dark background with Inter font. Clicking it navigates to /w/house-move and shows your own paperless-ngx documents, newest first. A direct browser reload on /w/house-move still renders (SPA fallback, no 404).
awaiting: user response

## Tests

### 1. Browser walkthrough — landing page, house-move stream, SPA fallback
test: Run `make build && ./bin/webspaces sync && ./bin/webspaces serve`, open http://127.0.0.1:7777/ in a browser, and click the house-move webspace.
expected: Landing page lists house-move with a non-zero item count on a near-black (#020617) dark background with Inter font. Clicking it navigates to /w/house-move and shows your own paperless-ngx documents, newest first. A direct browser reload on /w/house-move still renders (SPA fallback, no 404).
result: [pending]

### 2. Detail pane — instant metadata, scroll containment, deep link, offline degradation
test: Click a document row to open the detail pane; scroll the extracted text and the PDF embed while watching the stream list; click "Open in paperless-ngx"; then make paperless-ngx unreachable and click a different row.
expected: Title/date/tags appear instantly with a skeleton where the preview will be, then fill in. Scrolling the PDF or extracted text moves only that region, never the stream list. The CTA opens the exact correct document in a new tab. With paperless-ngx down, the pane shows "Couldn't load this document" with the offline body copy and a retry button, while title/tags/CTA remain visible and clickable.
result: [pending]

### 3. Stream row visual fidelity and date agreement
test: Open a webspace row with several tags, a very long title, and a document with no OCR text; compare rendered dates against paperless-ngx's own document dates.
expected: Every row is the same height; thumbnails are a consistent small portrait size with a document icon fallback; long titles ellipsis at one line; preview snippets stop at two lines; a document with no extracted text shows title/date/tags with no collapsed row; dates match paperless-ngx's own display.
result: [pending]

### 4. Empty / scroll / long-name / sync-failure states (judgment-tier prohibition)
test: (a) Open a webspace whose keywords match no tags. (b) Open house-move and scroll the stream while the detail pane is open. (c) Add an 80+ character webspace name and open it. (d) Point base_url at an unreachable host, sync, restart, and open that webspace.
expected: (a) "Nothing here yet" centred, no list chrome. (b) Stream scrolls independently, detail pane doesn't move, nothing scrolls sideways. (c) Header truncates with an ellipsis and a hover tooltip shows the full name. (d) "Couldn't load this webspace" with retry and the recorded sync.error text — never "Nothing here yet".
result: [pending]

### 5. README walkthrough and plugin-contract sufficiency
test: Read README.md top to bottom and follow it literally as a fresh clone (env vars, copy config, edit webspace, build, run). Then skim docs/plugin-contract.md and judge whether a second plugin could be written from it alone.
expected: Every documented command works with no undocumented step, ending in a browsable webspace. The plugin contract answers, without repo access — which interface to implement, how the kernel finds/launches the binary, how config reaches it, and what each Item field means.
result: [pending]

### 6. Outbound-host restriction prohibition (unwired test-tier guarantee)
test: Decide how to close the flagged prohibition — no committed test enforces that the kernel/plugin only ever talk to the configured paperless-ngx base_url and loopback (readonly_test.go enforces GET-only methods, not destination hosts). Manual repo-wide search found zero telemetry code and zero foreign hosts.
expected: Either accept the circumstantial evidence and downgrade the item's `verification:` tier to judgment in 01-01-PLAN.md, or request a committed outbound-host allowlist test (candidate for the next phase or a gap-closure plan).
result: [pending]

## Summary

total: 6
passed: 0
issues: 0
pending: 6
skipped: 0
blocked: 0

## Gaps
