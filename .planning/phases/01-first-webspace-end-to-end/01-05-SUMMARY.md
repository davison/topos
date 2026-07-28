---
phase: 01-first-webspace-end-to-end
plan: 05
subsystem: ui
tags: [sveltekit, tailwindcss-v4, vite, css, smoke-test, gap-closure]

# Dependency graph
requires:
  - phase: 01-first-webspace-end-to-end (01-01, 01-02, 01-03, 01-04)
    provides: SvelteKit scaffold, DetailPane, StreamRow/StreamList components, kernel HTTP API, e2e smoke test
provides:
  - Root layout imports src/app.css, restoring the entire Tailwind v4 + UI-SPEC design-token stylesheet to the production SPA build
  - scripts/e2e-smoke.sh recurrence guard: stale-listener pre-check + stylesheet assertion, so a future CSS-less build fails the smoke test instead of passing vacuously
  - Freshly built kernel serving styled house-move webspace at 127.0.0.1:7777, handed off for UAT re-run
affects: [any future frontend phase touching web/src/routes/+layout.svelte or scripts/e2e-smoke.sh]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "SvelteKit root layout must import the global stylesheet (../app.css) — an orphan app.css is valid to svelte-check/npm run build but silently excluded from the Vite module graph, producing a build with zero CSS assets"
    - "Smoke tests that assert against 'the server' must first prove no stale process already owns that port — otherwise a passing assertion can be answering an old binary"

key-files:
  created: []
  modified:
    - web/src/routes/+layout.svelte
    - scripts/e2e-smoke.sh

key-decisions:
  - "Root cause fixed at its single source (one import line) with no compensating changes to vite.config.ts, svelte.config.js, app.css, embed.go, the Makefile, or any component — matches the plan's own root-cause investigation (.planning/debug/stream-ui-unstyled.md)"
  - "The @source contingency in app.css was not needed — Tailwind v4 automatic source detection covered web/src once app.css entered the build graph"
  - "Smoke-test stylesheet assertion checks three things (link exists, asset fetches non-empty, body contains the #020617 token) because any two alone still pass on a broken build (dead link, empty file, or placeholder CSS)"

patterns-established:
  - "Pattern 1: Recurrence guards for silent build defects belong in the smoke test that already exercises the full build→serve→request path, not as a separate unit test"

requirements-completed: [UI-01, UI-03, UI-04]

coverage:
  - id: D1
    description: "web/src/routes/+layout.svelte imports ../app.css; production build emits a real CSS asset linked from 200.html"
    requirement: "UI-01"
    verification:
      - kind: integration
        ref: "automated verify command in 01-05-PLAN.md Task 1 (npm run build + grep for .line-clamp-2/.truncate/.stream-row-surface/.stream-row-meta/#020617/display:flex in emitted CSS)"
        status: pass
    human_judgment: false
  - id: D2
    description: "scripts/e2e-smoke.sh aborts with a named FAIL if 127.0.0.1:7777 is already occupied before starting the server under test"
    verification:
      - kind: e2e
        ref: "./scripts/e2e-smoke.sh full run against live paperless-ngx, plus manual reproduction (killed a stale listener from a previous session and observed the new pre-check would have caught it)"
        status: pass
    human_judgment: false
  - id: D3
    description: "scripts/e2e-smoke.sh asserts the served SPA HTML links a stylesheet, that stylesheet fetches non-empty, and its body contains the #020617 design token"
    verification:
      - kind: e2e
        ref: "./scripts/e2e-smoke.sh full run (assertion executed and passed)"
        status: pass
      - kind: other
        ref: "negative control: HTML with no stylesheet link correctly rejected by the matcher"
        status: pass
    human_judgment: false
  - id: D4
    description: "Fresh kernel build serves the styled house-move webspace at http://127.0.0.1:7777/ with a populated stream, left running for user UAT re-run"
    requirement: "UI-03"
    verification:
      - kind: integration
        ref: "curl http://127.0.0.1:7777/w/house-move (links stylesheet with #020617), curl .../api/webspaces/house-move/stream (35 items)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Stream rows render as designed fixed-height cards and the detail pane sits beside the stream (UAT tests 2, 3, 4 visual re-verification)"
    requirement: "UI-04"
    verification: []
    human_judgment: true
    rationale: "Visual rendering fidelity (row height, thumbnail sizing, ellipsis/clamp behavior, two-pane layout, independent scroll regions) requires human eyes in a real browser; automated checks in this plan only prove the CSS ships and contains the expected selectors/tokens, not that the browser renders them as designed"

# Metrics
duration: 4min
completed: 2026-07-28
status: complete
---

# Phase 01 Plan 05: Restore SPA styling (gap G-01-2) Summary

**Added the single missing `import '../app.css'` line to the SvelteKit root layout, restoring all Tailwind v4 + UI-SPEC styling to the production build, and hardened scripts/e2e-smoke.sh with a stale-listener pre-check and a three-part stylesheet assertion so the defect can never silently reappear.**

## Performance

- **Duration:** 4 min
- **Started:** 2026-07-28T11:57:58Z
- **Completed:** 2026-07-28T12:01:38Z
- **Tasks:** 3
- **Files modified:** 2 (source), plus regenerated gitignored build artifacts (kernel/webui/build/, bin/webspaces, bin/plugins/webspaces-plugin-paperless)

## Accomplishments
- Root-caused-and-fixed gap G-01-2: `web/src/routes/+layout.svelte` now imports `../app.css`, so Tailwind v4, the UI-SPEC dark-theme design tokens, the `@layer base` body rules, and the hand-authored `.stream-row-surface`/`.stream-row-meta` classes all enter the Vite module graph and are emitted as a real stylesheet (33,334 bytes) linked from `200.html`
- Added a recurrence guard to `scripts/e2e-smoke.sh`: a stale-listener pre-check (Edit A) and a stylesheet assertion with three independent checks (Edit B) — link exists, asset fetches non-empty, body contains the `#020617` token
- Rebuilt and left a fresh kernel serving the styled house-move webspace at `http://127.0.0.1:7777/`, populated with 35 items, ready for the user's UAT re-run of tests 2–4

## Task Commits

Each task was committed atomically:

1. **Task 1: Import the stylesheet entry point into the root layout** - `5d2b119` (fix)
2. **Task 2: Recurrence guard — smoke test asserts the served HTML links a real stylesheet** - `85ef4a9` (test)
3. **Task 3: Rebuild, serve, and hand off for UAT re-run of tests 2–4** - no commit (all outputs are gitignored build artifacts: `bin/webspaces`, `bin/plugins/webspaces-plugin-paperless`, `kernel/webui/build/`)

**Plan metadata:** committed below (docs: complete plan)

## Files Created/Modified
- `web/src/routes/+layout.svelte` - Added `import '../app.css';` as the first statement in the script block, above the favicon import. No other line changed.
- `scripts/e2e-smoke.sh` - Edit A: moved `BASE` assignment above server start, added a pre-check that FAILs if `127.0.0.1:7777` already answers `/api/webspaces` before the script starts its own server. Edit B: after the readiness loop, fetches `$BASE/`, extracts the first `.css` stylesheet href, normalizes a relative (`./...`) href to an absolute path, fetches it, and asserts it is non-empty and contains `#020617`.

## Decisions Made
- The `@source` contingency in `web/src/app.css` (Task 1's fallback for Tailwind v4 source-detection failures) was not needed — the verify command's full selector list (`.line-clamp-2`, `.truncate`, `.stream-row-surface`, `.stream-row-meta`, `#020617`, `display:flex`) all appeared in the emitted CSS on the first build after the import was added.
- No changes made to `vite.config.ts`, `svelte.config.js`, `app.css`, `kernel/webui/embed.go`, the Makefile, or any component — confirming the debug session's root-cause diagnosis was complete and correct (single missing import, nothing else).

## Deviations from Plan

None - plan executed exactly as written.

One operational note (not a plan deviation): a stale `./bin/webspaces serve` process from a prior session (started 11:50, before this plan began at 11:57) was already listening on `127.0.0.1:7777` when execution started. It was stopped before running the new smoke-test pre-check and again before Task 3's fresh serve, exactly the scenario the plan's Task 2 Edit A and Task 3 precondition were written to guard against. Its child plugin subprocess (`webspaces-plugin-paperless`) did not exit with its parent on `kill` and required a separate `kill` both times — this is pre-existing subprocess-lifecycle behavior outside this plan's scope (files_modified did not include kernel process-management code) and is not tracked as a stub or deferred item since it doesn't affect any of this plan's must-haves.

## Issues Encountered

The plan's own Task 3 automated verify command (`"http://127.0.0.1:7777/${HREF#./}"`) produces a double-slash URL when the SPA's stylesheet href is already absolute (e.g. `/_app/immutable/assets/0.CaDOf_tl.css`, which does not start with `./`). The double-slash request was still answered with HTTP 200 by the kernel's SPA fallback handler, but with the fallback HTML (`200.html`), not the CSS file — an easy false-negative trap if run verbatim. `scripts/e2e-smoke.sh`'s own Edit B does not have this bug (it normalizes any leading `.` off first, then re-adds a single `/` only if the path doesn't already start with one). Verified manually with correct single-slash URL construction that the stylesheet fetches correctly and contains `#020617`; no plan file needed changing since `scripts/e2e-smoke.sh` (the committed artifact) was already correct — only the plan's inline one-off verify snippet had the edge case, and it is not part of the shipped codebase.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The rebuilt kernel is running at `http://127.0.0.1:7777/` (PID via `./bin/webspaces serve`, started 2026-07-28T13:00 local time) and has NOT been stopped — left running per the plan's explicit instruction for the user's UAT re-run.
- **UAT tests 2, 3, and 4 are pending user confirmation** in the browser (hard-reload with Ctrl+Shift+R to bypass any cached CSS-less HTML). Test steps are documented in full in `01-05-PLAN.md` Task 3's `<human-check>` block: detail pane beside the stream with independent scroll containment, fixed-height 152px stream rows with correct thumbnail/title/tag/date/snippet formatting, and the "Nothing here yet" / truncated-name-tooltip / sync-error states.
- Plan 01-06 (the phase's remaining plan) can proceed once the user confirms tests 2–4 pass.

---
*Phase: 01-first-webspace-end-to-end*
*Completed: 2026-07-28*
