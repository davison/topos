---
phase: 13-per-item-curation-installable-app
plan: 08
subsystem: ui
tags: [svelte, sveltekit, playwright, e2e, undo, race-condition]

# Dependency graph
requires:
  - phase: 13-07
    provides: "The webspace-targeted undo toast (ws/gen snapshotted at mark time), plus the WR-01 regression spec this plan extends"
provides:
  - "A true entry guard in web/src/routes/w/[webspace]/+page.svelte's load() — a stale-generation call is now a no-op with no state write, no request"
  - "A permanent browser-driven regression pinning 13-UAT.md's exact reported reproduction (exclude, switch to an EMPTY second webspace, Undo)"
  - "Rendered-stream (not only kernel-state) assertions on the reversal's target webspace, across all four write paths"
affects: [ui, playwright-e2e-suite]

# Actuals (#2632)
actuals:
  tokens: 5000
  tasks: 2
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "load()-style async fetchers that write loading state guard staleness at ENTRY (before any state write), not only after the await — a post-await-only guard can discard a stale RESPONSE but can never undo a stale-generation WRITE that already happened before the request was even issued."

key-files:
  created: []
  modified:
    - web/src/routes/w/[webspace]/+page.svelte
    - web/e2e/specs/13-undo-across-webspace-switch.spec.ts
    - docs/testing.md

key-decisions:
  - "Declined the gap's optional suggestion (switching the three onUndo closures to load(gen, { quiet: true })) with cause: quiet's failure branch never sets loadState = 'error', which would silently swallow a failed post-undo refetch. The entry guard delivers everything quiet was being considered for here, so taking it would make undo and exclude behave differently for zero gain — recorded as a plan prohibition, not a silent omission."
  - "B's rendered-stream assertions are expressed against server truth (readStream's own items.length) rather than an in-window-captured count, and are always paired with a nonzero-items check — so a test can never pass vacuously against an empty list, matching the plan's explicit non-vacuous requirement."

patterns-established:
  - "clickUndo(page, markWebspace) in 13-undo-across-webspace-switch.spec.ts: registers page.waitForResponse for the reversal's mark POST BEFORE clicking Undo, so every caller assertion after it is sequenced strictly after the browser has the response — the ordering discipline that makes an absence-of-skeleton assertion non-vacuous."

requirements-completed: [KERN-09, KERN-10]

coverage:
  - id: D1
    description: "load()'s entry guard closes G-13-1: a stale-generation call to load() performs no observable work (no loadState write, no getStream request) for any caller, guarded once at the top of the function."
    requirement: KERN-09
    verification:
      - kind: e2e
        ref: "web/e2e/specs/13-undo-across-webspace-switch.spec.ts#exclude in A4, switch to EMPTY B4, Undo — B4 renders no stranded skeleton (G-13-1)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The three pre-existing undo-across-webspace tests (single-item exclude, bulk exclude, detail-pane include) now assert on webspace B's rendered stream (row count + zero skeletons) in addition to the kernel-side proof, and all four tests share the clickUndo ordering discipline."
    requirement: KERN-10
    verification:
      - kind: e2e
        ref: "web/e2e/specs/13-undo-across-webspace-switch.spec.ts (all 4 tests)"
        status: pass
      - kind: e2e
        ref: "make e2e (full Playwright suite, 139 tests, incl. 13-exclude-tracer.spec.ts, 13-multi-select-bulk-exclude.spec.ts, 13-excluded-view.spec.ts, spec-hygiene.spec.ts)"
        status: pass
    human_judgment: false

duration: ~50min
completed: 2026-08-15
status: complete
---

# Phase 13 Plan 08: Cross-webspace Undo skeleton strand (G-13-1) Summary

**Added a true entry guard to `load()` in `web/src/routes/w/[webspace]/+page.svelte` — a stale-generation call is now a genuine no-op — closing the class of bug where clicking Undo after switching webspaces stranded four permanent loading skeletons in the navigated-to webspace.**

## Performance

- **Duration:** ~50 min
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- `load(gen, options)` now returns immediately, as its first statement, when `gen !== navGeneration` — before the `quiet` destructuring and before the `loadState = 'loading'` write. Both existing post-await guards (the in-flight-navigation race) are unchanged.
- A new fourth Playwright test drives the exact UAT reproduction: exclude an item in a populated webspace, switch to a genuinely EMPTY second webspace inside the toast's real 5000ms window, click Undo, and assert the second webspace still shows `Nothing here yet` with zero stream skeletons — plus the reversal is visible on both the kernel (A's `excluded_count` is 0) and on return navigation to A (the restored item renders as a row).
- The three pre-existing tests in the same spec file (single-item exclude, bulk exclude, detail-pane include) were extended with the identical rendered-stream half they had deliberately skipped — B's row count against server truth, plus zero skeletons — and all four tests now share one `clickUndo` helper that sequences every post-undo assertion strictly after the reversal's mark POST reaches the page.
- All three recorded instances of the false "a stale-generation refetch is a design-level no-op" claim (the spec's `readStream` doc comment, the in-test comment folded into the `clickUndo` migration, and `docs/testing.md`'s spec catalogue entry) now state the property as a consequence of `load()`'s entry guard.

## Task Commits

Both tasks were TDD (RED/GREEN), producing three commits total:

1. **Task 1 — RED:** `4f67d32` (test) — new fourth test added, observed FAILING against the unfixed source on B4's `Nothing here yet` assertion (element not found — the pre-fix skeleton branch was rendering instead), never on any A4 assertion.
2. **Task 1 — GREEN:** `d44fd68` (feat) — the `load()` entry guard, plus doc-comment corrections above `load` and above `markBusy`. All 4 tests pass; `npm --prefix web run check` and `check:e2e` clean.
3. **Task 2:** `bcf09dc` (test) — extended the three pre-existing tests with B-side rendered assertions via the shared `clickUndo` helper, and corrected all three recorded instances of the false no-op claim. Full `make e2e` suite (139 tests) passes.

**Plan metadata:** (this commit, following SUMMARY.md creation)

## RED Evidence (Task 1 Step 2, run against the unfixed source)

```
Error: expect(locator).toBeVisible() failed
Locator: getByText('Nothing here yet')
Expected: visible
Timeout: 5000ms
Error: element(s) not found
```

Failing line: `await expect(page.getByText('Nothing here yet')).toBeVisible();` — the assertion on webspace B4's rendered empty-state copy, immediately after the cross-webspace Undo click. The accessibility snapshot captured at failure time showed only the header/banner and an empty `main`, consistent with the pre-fix skeleton branch (four decoration-only rows with no accessible name) rendering in place of the empty-state copy. The failure was on B4's rendered stream, never on any A4 (kernel-side) assertion — the right RED for G-13-1.

## Files Created/Modified

- `web/src/routes/w/[webspace]/+page.svelte` — `load()`'s entry guard (first statement: `if (gen !== navGeneration) return;`), plus extended doc comments above `load` and above `markBusy` naming the guard's contract and why it makes a deferred callback's stale-generation call safe.
- `web/e2e/specs/13-undo-across-webspace-switch.spec.ts` — added `WS_A4`/`WS_B4` fixture pair (B4 seeded `keywords: []`), `clickUndo` and `streamSkeletonLocator` helpers, a fourth test pinning the exact UAT reproduction, B-side rendered-stream assertions on the three pre-existing tests, and corrected doc comments (no instance of the false no-op claim remains).
- `docs/testing.md` — rewrote the spec catalogue entry for `13-undo-across-webspace-switch.spec.ts` to describe the two-layer assertion strategy (kernel for A, rendered page for B) and the empty-second-webspace reproduction, with no trace of the old false claim.

## Decisions Made

- Declined the gap's optional `load(gen, { quiet: true })` suggestion for the three `onUndo` closures, with cause: `quiet`'s failure branch never sets `loadState = 'error'`, which would silently swallow a failed post-undo refetch — a real behavior regression for a foreground user action. The entry guard alone delivers the safety `quiet` was being considered for. Recorded as a plan prohibition (verified: the closures are unchanged in the diff).
- `ensurePolling`'s own `load(gen, { quiet: true })` call site was deliberately left untouched — the entry guard makes it strictly less work when its generation has gone stale (it now returns before issuing a request it would previously have made and discarded), with no behavior change. Not edited; documented per the plan's Step 4 instruction.

## Deviations from Plan

None — plan executed exactly as written. Both tasks' `<done>` criteria and the plan's `<verification>` section were satisfied without any Rule 1-4 auto-fixes.

## Issues Encountered

`web/node_modules` was not present in the worktree at execution start (`npm ci` had never been run there), so `npm --prefix web run check:e2e` initially failed with `tsc: command not found`. Ran `npm ci` in `web/` before continuing — a one-time environment setup, not a deviation from plan content.

## Verification Run (in order, per plan `<verification>`)

1. `npm --prefix web run check` — 0 errors (967 files, 10 pre-existing unrelated warnings).
2. `npm --prefix web run check:e2e` — 0 errors.
3. `npm --prefix web run test` — 1078 tests passed (58 files), no new failures.
4. `make e2e E2E_ARGS="e2e/specs/13-undo-across-webspace-switch.spec.ts"` — 4 tests, 0 failures.
5. `make e2e` — full suite, 139 tests, 0 failures — including `13-exclude-tracer.spec.ts`, `13-multi-select-bulk-exclude.spec.ts`, `13-excluded-view.spec.ts`, and `spec-hygiene.spec.ts`.
6. `go build ./...` — clean.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

G-13-1 is closed: `load()`'s entry guard is a structural fix (guards every caller, present and future), backed by a permanent browser regression pinning the exact reported UAT reproduction. 13-07's kernel-side guarantees, the toast's 5000ms contract, and every same-webspace undo path remain unchanged and passing. No known stubs or deferred items from this plan.

---

*Phase: 13-per-item-curation-installable-app*
*Completed: 2026-08-15*

## Self-Check: PASSED

- FOUND: `.planning/phases/13-per-item-curation-installable-app/13-08-SUMMARY.md`
- FOUND: `web/src/routes/w/[webspace]/+page.svelte`
- FOUND: `web/e2e/specs/13-undo-across-webspace-switch.spec.ts`
- FOUND: `docs/testing.md`
- FOUND: commit `4f67d32` (test: RED)
- FOUND: commit `d44fd68` (feat: GREEN, entry guard)
- FOUND: commit `bcf09dc` (test: B-side assertions + doc corrections)
