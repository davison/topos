---
phase: 06-ui-scalable-source-surface
plan: 02
subsystem: ui
tags: [svelte5, sveltekit, bits-ui, popover, resize-observer, source-filter]

# Dependency graph
requires:
  - phase: 06-ui-scalable-source-surface plan 01
    provides: detail-pane search highlighting groundwork (parallel, not a hard dependency of this plan's chip work)
  - phase: 05-source-instances-per-type-matching
    provides: "SourceStatus.display_name and per-instance identity (D-08/D-09) this chip renders from"
provides:
  - "One merged SourceChip.svelte per configured source instance, replacing the SourceHealthChip + SourceFilterChips two-row header"
  - "Multi-select source filtering (D-02) via a Set<string>, persisted in the URL as ?sources= (plural), replacing Phase 2's single-select ?source="
  - "A local shadcn-svelte-style popover wrapper over the already-installed bits-ui popover primitive"
  - "Measured single-line chip row with overflow into a popover at any instance count, worst-of health tone surfaced on the overflow trigger"
affects: [07-webspace-builder-ui]

# Actuals (#2632)
actuals:
  tokens: 12539
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "shadcn-svelte-style local wrapper over an already-installed bits-ui primitive (web/src/lib/components/ui/popover/), mirroring the existing tooltip wrapper's file layout exactly — no npm install, no shadcn CLI invocation"
    - "ResizeObserver-driven overflow measurement: a visible clipped row plus an off-screen unclipped measurement clone, diffed before writing state to avoid a resize-then-measure feedback loop"
    - "Set<string> as the canonical multi-select filter representation, serialized to a comma-joined URL query value with per-member degrade on resolve"

key-files:
  created:
    - web/src/lib/components/SourceChip.svelte
    - web/src/lib/components/ui/popover/index.ts
    - web/src/lib/components/ui/popover/popover.svelte
    - web/src/lib/components/ui/popover/popover-trigger.svelte
    - web/src/lib/components/ui/popover/popover-content.svelte
  modified:
    - web/src/lib/format.ts
    - web/src/lib/components/WebspaceHeader.svelte
    - web/src/lib/components/StreamList.svelte
    - web/src/routes/w/[webspace]/+page.svelte
    - web/src/lib/components/sources.test.ts
    - web/src/lib/components/staleness.test.ts

key-decisions:
  - "?sources= replaces ?source= outright, no backward-compatible read of the singular key (plan's own recorded decision, single-user desktop tool, no bookmark-compatibility audience)"
  - "visibleChipCount implemented per the plan's own <action> algorithm text (subtract reservedWidth, check full-fit before charging the overflow trigger's width) rather than the plan's own numeric acceptance-criteria example, which was internally inconsistent — see Deviations"

patterns-established:
  - "Reduction-over-a-set health tone helpers (worstHealthTone) seed the accumulator with the least-alarming rank and special-case the empty-input result separately, rather than seeding with the 'no data' rank — seeding with 'unknown' silently discards every all-success/all-warning input's true worst tone"

requirements-completed: [UI-07]

coverage:
  - id: D1
    description: "One merged SourceChip per configured instance (health dot, name, filter toggle, hover-revealed refresh, health tooltip) replaces the two retired rows"
    requirement: UI-07
    verification:
      - kind: unit
        ref: "web/src/lib/components/sources.test.ts — resolveSourceFilters/toggleSourceFilter/filterItemsBySource/streamVariant describe blocks"
        status: pass
      - kind: manual_procedural
        ref: "06-02-PLAN.md Task 1 <verify><human-check> — one chip per source, click-to-filter, hover-reveal refresh, tooltip copy parity"
        status: unknown
    human_judgment: true
    rationale: "Visual/interaction confirmation (hover reveal, tooltip rendering, click behavior in a live browser) was not performed this session — no browser/dev-server access in this execution environment. Logic underneath (filter resolution, toggle, tone mapping) is unit-tested; the rendered result needs a human or automated-UI pass."
  - id: D2
    description: "Multi-select filtering (Set<string>) persisted in the URL as ?sources=, degrading unrecognised members individually rather than all-or-nothing"
    requirement: UI-07
    verification:
      - kind: unit
        ref: "web/src/lib/components/sources.test.ts — 'resolveSourceFilters' describe block (partially-stale, whitespace-tolerant, round-trip via serializeSourceFilters)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Chip row stays a single line at any instance count via ResizeObserver-driven measurement, moving overflow chips into a popover whose trigger carries the worst-of hidden health tone"
    requirement: UI-07
    verification:
      - kind: unit
        ref: "web/src/lib/components/sources.test.ts — 'visibleChipCount' and 'worstHealthTone' describe blocks"
        status: pass
      - kind: manual_procedural
        ref: "06-02-PLAN.md Task 2 <verify><human-check> — 10+ instances / narrow-window overflow behavior, destructive-tone trigger"
        status: unknown
    human_judgment: true
    rationale: "Real-DOM ResizeObserver measurement behavior (actual pixel widths, the popover opening/closing, the destructive-tone trigger rendering) needs a live browser pass; the pure fit-computation logic underneath is unit-tested exhaustively."

# Metrics
duration: ~20min
completed: 2026-08-06
status: complete
---

# Phase 6 Plan 2: Scalable Source Chip Row Summary

**Merged the two-row source health/filter header into one `SourceChip` per instance with multi-select `Set<string>` filtering and a `ResizeObserver`-driven overflow popover that keeps the row on one line at any instance count.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-08-06 (session start)
- **Completed:** 2026-08-06T20:54:41Z
- **Tasks:** 3
- **Files modified:** 13 (5 created, 6 modified, 2 deleted)

## Accomplishments
- One `SourceChip.svelte` per configured instance — health dot, truncated display name, real-button filter toggle (`aria-pressed`), hover/focus-revealed refresh control, and the Phase 2 tooltip copy carried forward verbatim plus a new syncing branch — replaces `SourceHealthChip.svelte` + `SourceFilterChips.svelte` outright.
- `format.ts` filter helpers rewritten around `Set<string>`: `resolveSourceFilters` degrades unrecognised members individually (not all-or-nothing), `toggleSourceFilter` is a non-mutating toggle, `serializeSourceFilters` round-trips to the URL; `?sources=` (plural) replaces `?source=` with no compatibility shim.
- New local `web/src/lib/components/ui/popover/` wrapper over the already-installed `bits-ui` popover primitive (no npm install, no dependency diff) mirroring the existing tooltip wrapper's file/export shape.
- `WebspaceHeader.svelte` measures the chip row via `ResizeObserver` against an off-screen unclipped clone, renders only the chips that fit inline (`visibleChipCount`), and moves the remainder into a popover whose trigger carries the worst-of health tone (`worstHealthTone`) over the hidden set — a failing hidden source is never silently invisible. `Clear filters` (shown only while the selection is non-empty) and `Refresh all` stay pinned and reserved out of the fit computation.
- 41 new/migrated unit tests in `sources.test.ts` covering per-member filter degrade, toggle idempotency, round-trip serialization, worst-of tone precedence, and the overflow fit boundary (exact fit, one-more-chip overflow, reserved-width reduction, zero-width floor, determinism).

## Task Commits

Each task was committed atomically:

1. **Task 1: One merged chip per instance, multi-select filtering end-to-end** - `7687dd6` (feat)
2. **Task 2: Keep the row on one line at any instance count — measured overflow into a popover** - `7745e50` (feat)
3. **Task 3: Unit tests for the multi-select filter, worst-of tone and overflow fit boundary** - `3847b72` (test)

**Plan metadata:** (this commit)

## Files Created/Modified
- `web/src/lib/components/SourceChip.svelte` - the single merged per-instance affordance
- `web/src/lib/components/ui/popover/{index.ts,popover.svelte,popover-trigger.svelte,popover-content.svelte}` - local shadcn-style wrapper over bits-ui's popover
- `web/src/lib/format.ts` - `resolveSourceFilters`/`toggleSourceFilter`/`serializeSourceFilters`/`worstHealthTone`/`visibleChipCount`; `filterItemsBySource`/`streamVariant` moved to `Set<string>`
- `web/src/lib/components/WebspaceHeader.svelte` - single measured chip row, overflow popover, Clear filters, pinned Refresh all
- `web/src/lib/components/StreamList.svelte` - `selectedSources: Set<string>`, filtered-empty copy names a source only when exactly one is selected
- `web/src/routes/w/[webspace]/+page.svelte` - `?sources=` URL round trip, `toggleFilter`/`clearFilters` handlers
- `web/src/lib/components/sources.test.ts` - migrated + expanded unit coverage
- `web/src/lib/components/staleness.test.ts` - migrated one `filterItemsBySource` call site to the `Set` signature (Rule 3, same-package compile dependency)
- `web/src/lib/components/SourceHealthChip.svelte`, `web/src/lib/components/SourceFilterChips.svelte` - deleted, absorbed into `SourceChip.svelte`

## Decisions Made
- `?sources=` replaces `?source=` outright per the plan's own recorded decision (single-user desktop tool, no external-bookmark audience to preserve compatibility for).
- Implemented `visibleChipCount` per the plan's own `<action>` algorithm description (full-fit check before charging the overflow trigger's width) rather than the plan's literal `visibleChipCount([10,10,10], 35, 0, 8)` → `2` acceptance-criteria example — see Deviations.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `worstHealthTone`'s reduction seed produced the wrong tone for an all-success input**
- **Found during:** Task 3 (unit-testing `worstHealthTone`)
- **Issue:** The reducer initialized its running `worst` value to `'unknown'` (rank 2 of 4) rather than the least-alarming rank. A set containing only `success`-tone sources compared `success` (rank 3) against the seed (rank 2), never won the "more alarming" comparison, and the function incorrectly returned `'unknown'` instead of `'success'`.
- **Fix:** Seeded the accumulator with `'success'` (the least-alarming rank, always beaten by any real source's tone) and handled the genuinely-empty-input case with an explicit early return of `'unknown'` before the loop, matching the doc comment's stated contract.
- **Files modified:** `web/src/lib/format.ts`
- **Verification:** `worstHealthTone`'s new "maps a single success source to success" test failed before the fix and passes after; full suite (146 tests) green.
- **Committed in:** `3847b72` (Task 3 commit — the buggy version was introduced in Task 2's `7745e50` and corrected here since Task 3's own tests caught it before Task 2's commit was superseded)

**2. [Rule 1 - Bug] Plan's own `visibleChipCount` numeric acceptance-criteria example was internally inconsistent**
- **Found during:** Task 2 implementation, before writing any test
- **Issue:** The plan's acceptance criteria asserted `visibleChipCount([10,10,10], 35, 0, 8)` returns `2` and `visibleChipCount([10,10,10], 30, 0, 8)` returns `3`. For a fixed set of chip widths summing to 30, no monotonic width-fitting algorithm can return *fewer* visible chips at a *larger* available width (35) than at a smaller one (30) — the two examples are mathematically incompatible with each other and with the function's own documented algorithm ("if every chip fits, return the full count").
- **Fix:** Implemented `visibleChipCount` exactly per the plan's own `<action>` prose (subtract `reservedWidth`; if the full set fits, return it uncharged; only when it does not fit, reduce the budget by `overflowTriggerWidth` and accumulate). This satisfies the `30 → 3` example verbatim and is internally consistent (monotonic in `availableWidth`) at every boundary. `sources.test.ts`'s own fit-boundary tests use self-consistent parameter sets (documented inline) rather than the plan's contradictory `35 → 2` example.
- **Files modified:** `web/src/lib/format.ts`, `web/src/lib/components/sources.test.ts`
- **Verification:** All `visibleChipCount` unit tests pass, including the plan's own `30 → 3` exact-fit example and a directly-derived "one chip more than fits hides exactly one" case built from consistent numbers.
- **Committed in:** `7745e50` (Task 2 commit, implementation) / `3847b72` (Task 3 commit, tests)

---

**Total deviations:** 2 auto-fixed (both Rule 1 — bugs; one in this plan's own written code, one in the plan's own acceptance-criteria prose)
**Impact on plan:** Both fixes are necessary for correctness (a monotonic, side-effect-free fit function and an accurate health-tone reduction are load-bearing for the phase's "no source silently invisible" prohibition). No scope creep.

## Issues Encountered
- No browser/dev-server access in this execution session — both tasks' `<human-check>` verification steps (visual chip-row behavior, overflow-trigger tone, popover interaction under `make dev`) were not performed live. All underlying pure-function logic is unit-tested (146 passing tests); the rendered/interactive result is recorded as `human_judgment: true` in this SUMMARY's `coverage:` block for a follow-up UAT pass.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- UI-07 is code-complete and unit-tested; a live `make dev` pass against the operator's real multi-instance config (verifying the 10+-instance overflow behavior and the destructive-tone trigger specifically) is the recommended first UAT step before Phase 7 (Webspace Builder UI) builds further chip-row-adjacent surface.
- No blockers for Phase 6 Plan 3 or Phase 7 — this plan's file surface (`SourceChip.svelte`, `format.ts` filter helpers, the new popover wrapper) is self-contained and does not touch this phase's other plans' declared files.

---
*Phase: 06-ui-scalable-source-surface*
*Completed: 2026-08-06*

## Self-Check: PASSED

- FOUND: `web/src/lib/components/SourceChip.svelte`
- FOUND: `web/src/lib/components/ui/popover/index.ts`
- CONFIRMED ABSENT: `web/src/lib/components/SourceHealthChip.svelte`, `web/src/lib/components/SourceFilterChips.svelte`
- FOUND commit: `7687dd6` (Task 1)
- FOUND commit: `7745e50` (Task 2)
- FOUND commit: `3847b72` (Task 3)
