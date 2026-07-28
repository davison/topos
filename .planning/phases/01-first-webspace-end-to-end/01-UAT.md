---
status: diagnosed
phase: 01-first-webspace-end-to-end
source: [01-VERIFICATION.md]
started: 2026-07-28T02:45:00Z
updated: 2026-07-28T10:10:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Browser walkthrough — landing page, house-move stream, SPA fallback
test: Run `make build && ./bin/webspaces sync && ./bin/webspaces serve`, open http://127.0.0.1:7777/ in a browser, and click the house-move webspace.
expected: Landing page lists house-move with a non-zero item count on a near-black (#020617) dark background with Inter font. Clicking it navigates to /w/house-move and shows your own paperless-ngx documents, newest first. A direct browser reload on /w/house-move still renders (SPA fallback, no 404).
result: pass

### 2. Detail pane — instant metadata, scroll containment, deep link, offline degradation
test: Click a document row to open the detail pane; scroll the extracted text and the PDF embed while watching the stream list; click "Open in paperless-ngx"; then make paperless-ngx unreachable and click a different row.
expected: Title/date/tags appear instantly with a skeleton where the preview will be, then fill in. Scrolling the PDF or extracted text moves only that region, never the stream list. The CTA opens the exact correct document in a new tab. With paperless-ngx down, the pane shows "Couldn't load this document" with the offline body copy and a retry button, while title/tags/CTA remain visible and clickable.
result: issue
reported: "detail list loads. In each row, a very large preview image of the document is shown centrally. Below it, unformatted, the document title on one row, then the date and tags on another row, and finally the abstract (cut off after what looks like a fixed number of characters). There are no links to the source document. Scrolling anywhere in the page scrolls the entire list in the browser viewport. Once the final document is reached, the formatting is different. The row background is black rather than light grey, the title is formatted as a heading, there is a link to open the document in paperless which works, and the preview image is (too) small with its own scroll region."
severity: major

### 3. Stream row visual fidelity and date agreement
test: Open a webspace row with several tags, a very long title, and a document with no OCR text; compare rendered dates against paperless-ngx's own document dates.
expected: Every row is the same height; thumbnails are a consistent small portrait size with a document icon fallback; long titles ellipsis at one line; preview snippets stop at two lines; a document with no extracted text shows title/date/tags with no collapsed row; dates match paperless-ngx's own display.
result: issue
reported: "same issue (as Test 2 — unstyled stream rows / stacked detail pane)"
severity: major
gap_ref: G-01-2

### 4. Empty / scroll / long-name / sync-failure states (judgment-tier prohibition)
test: (a) Open a webspace whose keywords match no tags. (b) Open house-move and scroll the stream while the detail pane is open. (c) Add an 80+ character webspace name and open it. (d) Point base_url at an unreachable host, sync, restart, and open that webspace.
expected: (a) "Nothing here yet" centred, no list chrome. (b) Stream scrolls independently, detail pane doesn't move, nothing scrolls sideways. (c) Header truncates with an ellipsis and a hover tooltip shows the full name. (d) "Couldn't load this webspace" with retry and the recorded sync.error text — never "Nothing here yet".
result: issue
reported: "screenshot evidence (01-uat-test4-evidence.png): stream rows fully unstyled — full-size centered preview images, plain unformatted text, no source links, page-level scrolling; DetailPane renders stacked BELOW the list with its hand-authored dark styling intact (heading, working Open-in-paperless link, small scrollable thumbnail). Utility-class styling absent everywhere; custom CSS works. Parts (a)/(c)/(d) not yet testable — re-test after fix."
severity: major
gap_ref: G-01-2

### 5. README walkthrough and plugin-contract sufficiency
test: Read README.md top to bottom and follow it literally as a fresh clone (env vars, copy config, edit webspace, build, run). Then skim docs/plugin-contract.md and judge whether a second plugin could be written from it alone.
expected: Every documented command works with no undocumented step, ending in a browsable webspace. The plugin contract answers, without repo access — which interface to implement, how the kernel finds/launches the binary, how config reaches it, and what each Item field means.
result: pass
note: "user: 'I think it's a pass'"

### 6. Outbound-host restriction prohibition (unwired test-tier guarantee)
test: Decide how to close the flagged prohibition — no committed test enforces that the kernel/plugin only ever talk to the configured paperless-ngx base_url and loopback (readonly_test.go enforces GET-only methods, not destination hosts). Manual repo-wide search found zero telemetry code and zero foreign hosts.
expected: Either accept the circumstantial evidence and downgrade the item's `verification:` tier to judgment in 01-01-PLAN.md, or request a committed outbound-host allowlist test (candidate for the next phase or a gap-closure plan).
result: issue
reported: "test — user wants a committed outbound-host allowlist test rather than downgrading the verification tier"
severity: minor

## Summary

total: 6
passed: 2
issues: 4
pending: 0
skipped: 0
blocked: 0

## Gaps

- gap_id: G-01-2
  truth: "Detail pane opens beside the stream: instant metadata + skeleton then live preview fill; scroll containment per region; Open-in-paperless CTA on every item; stream rows show small consistent thumbnails, formatted title/date/tags, two-line clamped snippet"
  status: failed
  reason: "User reported: stream rows render unstyled — huge centered preview images, unformatted title/date/tags/abstract, no source links, whole-page scrolling; only the final rendered block (apparently the detail pane, stacked below the list instead of beside it) is dark-themed with heading, working paperless link, and a too-small scrollable preview region"
  severity: major
  test: 2, 3, 4
  root_cause: "web/src/routes/+layout.svelte is missing `import '../app.css';` — app.css (Tailwind v4 entry + all design tokens + hand-authored classes) is orphaned, so Vite emits ZERO CSS in the production build; 200.html links no stylesheet. Present since the 01-01 scaffold (da15f94). All rendering seen was browser UA defaults on semantic HTML. vite.config.ts, svelte.config.js, Makefile, embed.go, and all component markup verified correct."
  artifacts:
    - path: "web/src/routes/+layout.svelte"
      issue: "missing `import '../app.css';` at top of script block (the defect)"
    - path: "web/src/app.css"
      issue: "correct and complete but referenced by nothing"
    - path: ".planning/phases/01-first-webspace-end-to-end/01-uat-test4-evidence.png"
      issue: "screenshot evidence"
  missing:
    - "Add `import '../app.css';` to +layout.svelte, rebuild, confirm _app/immutable/assets/*.css emitted and 200.html links a stylesheet"
    - "Confirm utility selectors used by components (.line-clamp-2, .flex, .truncate) present in emitted CSS (secondary @source coverage check)"
    - "Recurrence guard: build-output assertion (e.g. in e2e-smoke.sh) that built HTML references at least one stylesheet"
  debug_session: .planning/debug/stream-ui-unstyled.md
- gap_id: G-01-6
  truth: "A committed, wired test enforces that no code path in the kernel or paperless plugin transmits data to any host other than the configured paperless-ngx base_url and loopback (the plan's test-tier prohibition)"
  status: failed
  reason: "User reported: test — requested a committed outbound-host allowlist test instead of downgrading the verification tier. readonly_test.go enforces GET-only methods but not destination hosts; no other test asserts an allowlist."
  severity: minor
  test: 6
  root_cause: "Definitional — the plan's test-tier prohibition was never wired to a committed test. readonly_test.go enforces GET-only methods but not destination hosts. No debug investigation needed (verifier established this with evidence)."
  artifacts:
    - path: "plugins/paperless/readonly_test.go"
      issue: "closest analog — enforces methods, not destination hosts"
  missing:
    - "Committed outbound-host allowlist test asserting kernel + paperless plugin only dial the configured base_url host and loopback"
  debug_session: none
