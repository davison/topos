---
phase: 260805-irt
plan: 01
subsystem: ui
tags: [svelte, tailwind, flexbox, vitest]

requires: []
provides:
  - "Region-scoped source-scan guard (web/src/lib/pane-layout.test.ts) enforcing the stream-fixed/detail-flex sizing invariant"
  - "Inverted pane sizing in web/src/routes/w/[webspace]/+page.svelte — detail pane flexes (flex-1, min-w-0), stream pane holds a conditional fixed width (w-[480px], shrink-0) driven by selectedItem"
affects: [ui, webspace-layout]

actuals:
  tokens: 1726
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Source-scan invariant tests over Svelte markup (extended from date-format.test.ts's pattern) — scope every assertion to a uniquely-matched class attribute substring, never the whole file, so future prose changes can't silently satisfy or invalidate the guard."
    - "Conditional Tailwind class interpolation driven by the same reactive value that gates a sibling's rendering ({selectedItem ? '...' : '...'}) — keeps two dependent layout decisions from ever disagreeing."

key-files:
  created:
    - web/src/lib/pane-layout.test.ts
  modified:
    - web/src/routes/w/[webspace]/+page.svelte

key-decisions:
  - "Reused the existing 480px constant, moved from the detail wrapper to the stream wrapper, rather than introducing a new value or theme token (per plan instruction and app.css's recorded regression on named spacing keys)."
  - "Stream wrapper's fixed-vs-flex sizing is conditional on selectedItem (not unconditionally fixed) to preserve today's full-width single-pane behavior when nothing is selected."

patterns-established:
  - "Pattern: Region-scoped source-scan guard — see web/src/lib/pane-layout.test.ts for the helper shape (findUniqueClassAttr) other layout-invariant tests should follow."

requirements-completed: []

coverage:
  - id: D1
    description: "Failing region-scoped guard added asserting the stream-fixed/detail-flex inversion (Task 1, RED)"
    verification:
      - kind: unit
        ref: "web/src/lib/pane-layout.test.ts#pane-layout source-scan guard"
        status: pass
    human_judgment: false
  - id: D2
    description: "Pane wrapper classes inverted so the detail pane flexes and the stream pane holds a fixed width when an item is open; guard now passes (Task 2, GREEN); svelte-check and production build both succeed"
    verification:
      - kind: unit
        ref: "web/src/lib/pane-layout.test.ts#pane-layout source-scan guard"
        status: pass
    human_judgment: false
  - id: D3
    description: "Real-browser resize behavior confirms the stream pane's width stays constant while the detail pane absorbs viewport width changes, with the single-pane full-width fallback, independent scroll, row truncation, and wide-viewport rendering all preserved"
    verification: []
    human_judgment: true
    rationale: "vitest runs in Node and jsdom performs no layout, so computed flexbox widths do not exist outside a real browser — Task 3's blocking human-verify walkthrough was performed by the user and approved 2026-08-05 (all seven checks passed)."

duration: ~12min
completed: 2026-08-05
status: complete
---

# Quick Task 260805-irt: Invert Pane Flex/Fixed Widths Summary

**Detail pane now flexes to absorb viewport width changes; stream pane holds a fixed 480px width whenever an item is open — inverted from today's flex-1 stream / fixed-480px detail layout, guarded by a new region-scoped source-scan test.**

## Performance

- **Duration:** ~12 min (Tasks 1-2; Task 3 resolved by user walkthrough)
- **Completed:** 2026-08-05
- **Tasks:** 3 of 3 completed (Task 3 human-verify walkthrough approved by user 2026-08-05)
- **Files modified:** 2

## Accomplishments

- Added `web/src/lib/pane-layout.test.ts`, a region-scoped source-scan guard that extracts each pane wrapper's own `class="..."` attribute (via a helper asserting exactly-one-match) and asserts the fixed/flex inversion against that scoped string — immune to unrelated text elsewhere in the route file.
- Confirmed RED: against the pre-fix route source, 3 of the new tests failed (detail-pane-not-fixed-width, detail-pane-flex-1, stream-pane-fixed-width), all passing pre-existing suites untouched (88 passed, 3 failed).
- Inverted the two pane wrappers in `web/src/routes/w/[webspace]/+page.svelte`:
  - Detail wrapper: removed `w-[480px] shrink-0`, added `flex-1 min-w-0` (kept `flex flex-col overflow-hidden border-l border-border pl-8`).
  - Stream wrapper: now `min-h-0 min-w-0 overflow-x-hidden overflow-y-auto` plus a `{selectedItem ? 'w-[480px] shrink-0' : 'flex-1'}` interpolation — fixed width only when an item is open, flexing to fill full width otherwise.
  - Updated the explanatory comment above the stream wrapper to describe the new sizing relationship alongside the existing scroll-independence note.
- Confirmed GREEN: full vitest suite passes (91/91, 6 files), `svelte-check` reports 0 errors (1 pre-existing unrelated warning in `SearchBox.svelte`, out of scope), and `npm run build` succeeds with all four required utilities (`w-[480px]`, `shrink-0`, `flex-1`, `min-w-0`) present in the emitted stylesheet.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add a failing region-scoped guard for the pane fixed/flex inversion** - `9966e9f` (test)
2. **Task 2: Invert the pane sizing so the stream is fixed and the detail pane flexes** - `4e51006` (fix)

**Task 3 (checkpoint:human-verify, gate="blocking") has NOT been executed** — see "Next Steps: What Remains" below. No plan-metadata commit has been made; the orchestrator will handle docs commit once the checkpoint is resolved.

## Files Created/Modified

- `web/src/lib/pane-layout.test.ts` - New region-scoped source-scan guard (5 tests) enforcing the stream-fixed/detail-flex invariant.
- `web/src/routes/w/[webspace]/+page.svelte` - Detail wrapper now `flex-1 min-w-0` (was `w-[480px] shrink-0`); stream wrapper now conditionally fixed via `{selectedItem ? 'w-[480px] shrink-0' : 'flex-1'}` (was unconditionally `flex-1`); comment above the stream wrapper extended.

## Decisions Made

- Reused the existing `480px` value, simply relocated from the detail side to the stream side, per the plan's explicit instruction and `web/src/app.css`'s recorded regression on named spacing-namespace keys — no new theme token or container key introduced.
- Made the stream wrapper's width conditional on `selectedItem` (the same value gating the detail pane's own rendering) rather than unconditionally fixed, so the full-width single-pane state is preserved exactly as it behaves today.

## Deviations from Plan

None - plan executed exactly as written for Tasks 1 and 2.

## Issues Encountered

None. The Edit tool required a couple of retries to match exact tab-indentation in the route file's `<div class="...">` line, but this was a mechanical string-matching issue, not a plan or code deviation.

## Known Stubs

None.

## Task 3 Resolution (blocking human-verify checkpoint — APPROVED)

**Checkpoint resolved 2026-08-05:** the user ran the seven-step browser walkthrough below and replied "approved" — all checks behaved as described. The walkthrough is preserved for reference:

1. Run `make dev` from the repo root (starts the kernel on 127.0.0.1:7777 and opens the Vite dev server).
2. Open a webspace route (`/w/<name>`) that has items in its stream.
3. **Two-pane resize (the fix):** click a stream item so the detail pane opens. Drag the window edge to widen and then narrow the browser, or use devtools to drag the viewport divider.
   - Expect: the stream pane's width does NOT change. All extra (or lost) width goes to the detail pane on the right.
4. **Single-pane regression check:** deselect (reload the route, or navigate to another webspace) so no item is open.
   - Expect: the stream fills the full content width edge to edge, and resizes the window to flex normally.
5. **Independent scroll (preserved):** with the detail pane open on an item with long text or a PDF/image preview, scroll the stream list.
   - Expect: the detail pane's own scroll position does not move, and vice versa.
6. **Stream row integrity at the narrower fixed width:** with the detail pane open, check a row with a long title, group label, date, and tag pills.
   - Expect: the title truncates to one line, the snippet clamps to 2 lines, nothing overflows horizontally.
7. **Detail pane at a wide viewport:** maximize the window with an item open.
   - Expect: embeds and text fill the wider pane without pushing past the window edge or introducing a horizontal scrollbar.

## Next Phase Readiness

Not applicable — this is a standalone quick task, not a phase. No further follow-up work is anticipated; the layout inversion is self-contained to the two pane wrappers.

---
*Quick task: 260805-irt*
*Status: complete — Task 3 human-verify approved by user 2026-08-05*

## Self-Check: PASSED

- FOUND: web/src/lib/pane-layout.test.ts
- FOUND: web/src/routes/w/[webspace]/+page.svelte
- FOUND: .planning/quick/260805-irt-fix-pane-flex-stream-pane-fixed-width-de/260805-irt-SUMMARY.md
- FOUND commit: 9966e9f
- FOUND commit: 4e51006
