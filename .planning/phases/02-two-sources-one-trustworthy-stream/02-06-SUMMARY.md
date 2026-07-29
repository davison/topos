---
phase: 02-two-sources-one-trustworthy-stream
plan: 06
subsystem: ui
tags: [tailwindcss-v4, css-theme, e2e-smoke-test, gap-closure]

requires:
  - phase: 02-two-sources-one-trustworthy-stream
    provides: "SourceHealthChip tooltip (02-03) and the index page webspace cards (Phase 1) whose max-w-* utilities this plan repairs"
provides:
  - "app.css @theme inline block carrying only color/radius/font entries — no named spacing-scale keys shadowing Tailwind v4's container scale"
  - "scripts/assert-stylesheet.sh: reusable, standalone built-stylesheet assertion (G-01-2 + G-02-1 guards) usable with or without a live server"
  - "scripts/e2e-smoke.sh's stylesheet stage delegating to the shared helper"
affects: [ui, testing]

tech-stack:
  added: []
  patterns:
    - "Named max-w-* Tailwind v4 utilities resolve from the container theme namespace; never declare a same-named key in the spacing theme namespace or it silently shadows the container default"
    - "Built-artifact assertions factored into a standalone script (scripts/assert-stylesheet.sh) that both the plain build and the e2e smoke test can call, so the guard runs with or without a live server / credentials"

key-files:
  created:
    - scripts/assert-stylesheet.sh
  modified:
    - web/src/app.css
    - scripts/e2e-smoke.sh

key-decisions:
  - "Deleted the seven --spacing-<key> theme entries outright rather than renaming/relocating them — zero utilities in web/src reference them, so a renamed copy would be equivalent dead weight one grep further away"
  - "assert-stylesheet.sh accepts either --container-<key> custom property or an inlined var()-free rem value for each named width, since Tailwind v4's @theme inline block inlines resolved values rather than emitting custom properties for its own defaults"

requirements-completed: [UI-05, UI-06]

coverage:
  - id: D1
    description: "web/src/app.css no longer declares named spacing-scale entries that shadow Tailwind v4's container scale"
    requirement: "UI-06"
    verification:
      - kind: unit
        ref: "grep-based structural check: --container-xs/md/3xl resolve to 20rem/28rem/48rem in the built CSS; no .max-w-<key>{max-width:<n>px} rule remains"
        status: pass
      - kind: unit
        ref: "npm --prefix web run test (44/44 passing, unchanged suites)"
        status: pass
    human_judgment: false
  - id: D2
    description: "scripts/assert-stylesheet.sh created, executable, passes against the real build, rejects a collapsed fixture, accepts a var()-based fixture with max-w-px"
    verification:
      - kind: unit
        ref: "scripts/assert-stylesheet.sh (no arg, against real build) — exit 0"
        status: pass
      - kind: unit
        ref: "scripts/assert-stylesheet.sh against synthetic collapsed-pixel fixture — exit 1 (correctly rejected)"
        status: pass
      - kind: unit
        ref: "scripts/assert-stylesheet.sh against synthetic var(--container-*)+max-w-px fixture — exit 0"
        status: pass
      - kind: integration
        ref: "scripts/e2e-smoke.sh full run against live paperless-ngx/SilverBullet instances — stylesheet stage passed via the new helper against the actually-served CSS"
        status: pass
    human_judgment: false
  - id: D3
    description: "Health-chip tooltip and index page webspace link render at normal, readable size in an actual browser (the user-reported G-02-1 symptom)"
    verification: []
    human_judgment: true
    rationale: "No browser/screenshot tooling (Playwright, Chromium) was available in this execution environment to capture a rendered DOM. The built-CSS resolution (D1) proves the mechanism is fixed — max-w-xs/md/3xl now resolve to 20rem/28rem/48rem instead of 4px/16px/64px — but the actual visual re-check must happen against a real browser, per this project's human_verify_mode: end-of-phase config setting."

duration: 15min
completed: 2026-07-29
status: complete
---

# Phase 02 Plan 06: Gap closure for collapsed tooltip and index link (G-02-1) Summary

**Removed seven dead `--spacing-<key>` theme entries that were shadowing Tailwind v4's container scale, restoring the health-chip tooltip and index-page webspace cards to full width, plus a reusable `assert-stylesheet.sh` guard wired into the e2e smoke test.**

## Performance

- **Duration:** 15 min
- **Started:** 2026-07-29T16:27:00Z
- **Completed:** 2026-07-29T16:42:00Z
- **Tasks:** 2 completed
- **Files modified:** 3

## Accomplishments

- Deleted the seven named `--spacing-xs/sm/md/lg/xl/2xl/3xl` entries from `web/src/app.css`'s `@theme inline` block — they were referenced by zero utilities anywhere in `web/src` and existed only as documentation of the 01-UI-SPEC scale, but their presence in Tailwind v4's spacing theme namespace shadowed the default `--container-*` scale that every named `max-w-*` utility resolves from.
- Confirmed the built stylesheet now resolves `max-w-xs`/`max-w-md`/`max-w-3xl` via `var(--container-xs|md|3xl)` at 20rem/28rem/48rem respectively (previously inlined at 4px/16px/64px) — the exact mechanism that collapsed the health-chip tooltip (D-08) and the index page's webspace cards.
- Left the color, radius, and font entries in the `@theme` block untouched — this was a one-group deletion, not a theme rewrite.
- Added `scripts/assert-stylesheet.sh`: a standalone, executable helper that checks any CSS file (or the freshly built app stylesheet, resolved by globbing `kernel/webui/build/_app/immutable/assets/*.css`) against both the G-01-2 guard (non-empty, contains the `#020617` design token) and the new G-02-1 guard (named container widths resolve correctly; no named `max-w-*` utility resolves to a raw pixel value, while `max-w-px` is correctly exempted).
- Rewired `scripts/e2e-smoke.sh`'s stylesheet stage to delegate to the shared helper instead of running its own inline non-empty/token checks, so the served stylesheet (fetched from the live kernel, not just the local build) is checked by the same guard.
- Ran the full `e2e-smoke.sh` end to end against the user's live paperless-ngx and SilverBullet instances; the new stylesheet stage passed against the actually-served CSS.

## Task Commits

Each task was committed atomically:

1. **Task 1: Remove the named spacing entries that shadow Tailwind's container scale** - `3a6d7ec` (fix)
2. **Task 2: Add a built-stylesheet assertion and wire the e2e smoke test to it** - `c34c5e1` (feat)

**Plan metadata:** committed separately below.

## Files Created/Modified

- `web/src/app.css` - Deleted the seven `--spacing-<key>` theme entries; replaced with a comment recording the numeric-utility mapping and the shadowing mechanism, so the mistake is not reintroduced.
- `scripts/assert-stylesheet.sh` (new) - Standalone, reusable built-stylesheet assertion covering both the G-01-2 (non-empty, design-token) and G-02-1 (container-scale resolution) guards.
- `scripts/e2e-smoke.sh` - Stylesheet stage now delegates its non-empty/token checks to `assert-stylesheet.sh`, run against the fetched `$CSS_TMP`.

## Decisions Made

- Deleted the dead spacing tokens outright rather than renaming/relocating them elsewhere in the file — nothing consumes them, and a renamed copy is equally dead weight, just one grep further from discovery.
- `assert-stylesheet.sh`'s container-width check accepts either a `--container-<key>` custom property at the expected rem value or an inlined `.max-w-<key>{max-width:<rem>rem}` rule, per the `<interfaces>` note that `@theme inline` inlines resolved values rather than emitting custom properties the way non-inline theme blocks would.

## Deviations from Plan

None - plan executed exactly as written. Both tasks' `<verify>` blocks passed on the first attempt with no auto-fixes required.

### Note on scope-adjacent action (not a plan deviation, no files changed)

While running `make smoke`'s equivalent (`./scripts/e2e-smoke.sh`) to gather phase-level verification evidence, a stale `./bin/webspaces serve` process (PID 1200224, 34 minutes old, running an already-deleted binary from a prior session) was found listening on 127.0.0.1:7777, causing the smoke test's own "port is free" pre-check to correctly fail. This process was killed so the smoke test could run; it was a leftover background process from earlier work in this repo, not user-owned application state, and killing it is exactly what the smoke test's own pre-check message instructs ("stop it before running this smoke test"). No files were modified as a result.

## Issues Encountered

- After clearing the stale port-7777 listener, `./scripts/e2e-smoke.sh` ran successfully through its new stylesheet stage (`OK: stylesheet ... passes G-01-2 and G-02-1 guards`, checked against the live-served CSS) but failed later at an unrelated, pre-existing check: `FAIL: not every item's link.url starts with $PAPERLESS_URL/documents/`. This is outside this plan's scope (files_modified: `web/src/app.css`, `scripts/assert-stylesheet.sh`, `scripts/e2e-smoke.sh` only) — it concerns paperless-ngx link-URL construction, not the stylesheet/theme change this plan makes, and per the scope-boundary rule it was not investigated or fixed here. Logged to the phase's deferred-items tracking for a future look.

## Verification Evidence (per plan's `<verification>` and `<output>` sections)

1. `npm --prefix web run build` succeeds; `./scripts/assert-stylesheet.sh` (no argument, against the freshly built asset) exits 0.
2. `npm --prefix web run test` passes: 3 suites, 44/44 tests, unchanged from before this plan.
3. `git diff --stat` across both task commits touches exactly three files: `web/src/app.css`, `scripts/assert-stylesheet.sh` (new), `scripts/e2e-smoke.sh` — confirmed via `git diff --stat 3a6d7ec^..c34c5e1`.
4. **Resolved form of the three named widths in the built CSS:** all three resolve via `var(--container-<key>)` referencing a `--container-<key>: <rem>rem` custom property (e.g. `.max-w-3xl{max-width:var(--container-3xl)}` with `--container-3xl:48rem` defined elsewhere in the file) — the `var()`-reference branch of assertion 3, not the inlined-rem branch. Both branches were still implemented in `assert-stylesheet.sh` per the plan's own note that either form is possible.
5. **Human visual re-check:** NOT performed in a real browser in this execution environment — no Playwright/Chromium or other browser-automation tooling was available. The built-CSS evidence in point 4 directly proves the fix (the exact values a browser would apply), and this project's config (`human_verify_mode: end-of-phase`) defers live browser verification to the phase's end-of-phase UAT pass rather than per-plan. The filtered-empty `StreamEmpty` state's reachability was not evaluated for the same reason.
6. **`make smoke` / `scripts/e2e-smoke.sh` runnability:** Runnable in this environment — `PAPERLESS_URL`/`PAPERLESS_TOKEN` were present via `.env`, and a live paperless-ngx + SilverBullet instance were reachable. The stylesheet stage (this plan's concern) passed against the live-served CSS. The overall script failed later at an unrelated, pre-existing, out-of-scope check (see "Issues Encountered" above), which does not implicate this plan's changes.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- G-02-1 is closed at the CSS/build level with a recurrence guard (`scripts/assert-stylesheet.sh`) now exercised by both a standalone run and the live e2e smoke test.
- Recommend a real-browser visual pass (health-chip tooltip, index page cards, filtered-empty state reachability) at the phase's end-of-phase UAT gate, since that could not be performed in this execution environment.
- The unrelated `link.url` prefix failure surfaced during smoke-test verification should be triaged separately — it predates this plan and is out of its file scope.

---
*Phase: 02-two-sources-one-trustworthy-stream*
*Completed: 2026-07-29*

## Self-Check: PASSED

- FOUND: web/src/app.css
- FOUND: scripts/assert-stylesheet.sh
- FOUND: scripts/e2e-smoke.sh
- FOUND: commit 3a6d7ec
- FOUND: commit c34c5e1
