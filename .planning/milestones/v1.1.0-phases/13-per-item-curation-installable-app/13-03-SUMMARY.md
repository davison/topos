---
phase: 13-per-item-curation-installable-app
plan: 03
subsystem: curation
tags: [svelte5, bits-ui, playwright, sonner]

requires:
  - phase: 13-01
    provides: "item_marks substrate, POST /api/webspaces/{webspace}/marks, toast.ts (markPhrase/markSuccessToast/markFailureToast), DetailPane single-item exclude/include control"
  - phase: 13-02
    provides: "index.MarkView (ViewIncluded/ViewExcluded), ?view= on GET stream, streamResponse.excluded_count, orphan prune sweep"
provides:
  - "web/src/lib/selection.ts — toggleSelection/selectRange/clearSelection, the pure bulk-selection axis"
  - "StreamRow.svelte bulk-select props (bulkSelected/bulkModeActive/onbulktoggle) and the leading checkbox slot"
  - "SelectionActionBar.svelte — the floating bulk-action bar"
  - "StreamList.svelte threading bulk-select props + rendering SelectionActionBar inside its own scroll container"
  - "WebspaceHeader.svelte excludedCount/view/ontoggleview — the Excluded (N) toggle"
  - "api.ts: StreamResponse.excluded_count, getStream(webspace, view?)"
  - "+page.svelte: bulkSelection/bulkAnchor/view state, handleBulkToggle/handleBulkPrimary/handleToggleView, the excluded-view auto-flip"
  - "web/e2e/specs/13-multi-select-bulk-exclude.spec.ts, 13-excluded-view.spec.ts"
affects: [13-04, 13-05, 13-06]

actuals:
  tokens: 11877
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "selection.ts mirrors format.ts's pure-helper + colocated-vitest convention: no Svelte imports, no current-selection parameter on selectRange (so 'replace, never union' is true by construction, not by caller discipline)"
    - "StreamRow's root element is a <div role=\"button\"> instead of a real <button> — the leading Checkbox slot this plan adds is itself a real interactive <button> (bits-ui's stock recipe), and a <button> may never contain another; handleActivate/handleRowKeydown restore Enter/Space activation manually"
    - "$effect(() => { const ws = webspace; untrack(() => { ... }); }) — the per-webspace effect's ONLY intended dependency (webspace) is now the only tracked read; every other read/write inside untrack() can no longer cause the effect to re-schedule itself"

key-files:
  created:
    - web/src/lib/selection.ts
    - web/src/lib/selection.test.ts
    - web/src/lib/components/SelectionActionBar.svelte
    - web/src/lib/components/selection-action-bar.test.ts
    - web/e2e/specs/13-multi-select-bulk-exclude.spec.ts
    - web/e2e/specs/13-excluded-view.spec.ts
  modified:
    - web/src/lib/components/StreamRow.svelte
    - web/src/lib/components/StreamList.svelte
    - web/src/lib/components/WebspaceHeader.svelte
    - web/src/lib/api.ts
    - web/src/lib/components/sources.test.ts
    - web/src/routes/w/[webspace]/+page.svelte
    - docs/testing.md

key-decisions:
  - "StreamRow's root converted from <button> to <div role=\"button\" tabindex=\"0\"> (deviation, Rule 1) — the new Checkbox slot is a real nested <button>, and nesting a <button> inside a <button> is invalid HTML that also breaks click-event isolation (a checkbox click would bubble into the row's own onclick with no way to distinguish the two). handleActivate/handleRowKeydown restore native-button Enter/Space semantics explicitly; the checkbox's own onclick carries event.stopPropagation() so clicking it can never also open the detail pane."
  - "The per-webspace $effect wraps its body in untrack(), reading only `webspace` as a tracked dependency (deviation, Rule 1, found stabilizing 13-excluded-view.spec.ts) — the pre-existing `const gen = ++navGeneration;` both read and wrote navGeneration inside the effect, which made Svelte track navGeneration too. Confirmed live: after toggling to the excluded view, this effect re-ran 1-2 extra times, each resetting `view` back to 'included' and re-issuing a stale fetch that clobbered the just-toggled 'excluded' result moments after it landed. untrack() makes the effect's own doc comment (\"whenever the webspace route param changes\") true by construction."
  - "handleBulkPrimary/handleToggleView are view-driven from the single `view` state value (never a separate mode flag) — action/verb/undo-action all derive from `view === 'excluded'`, mirroring D-01's \"selection-set size is the only mode indicator\" discipline for the view axis too."
  - "The auto-flip (view resets to 'included' when the excluded bucket empties) is a single, non-looping recursive load() call: `view` is written to 'included' BEFORE the recursive call, so the auto-flip branch (`view === 'excluded' && res.items.length === 0`) is structurally false on that recursive call — no counter or extra flag needed."
  - "SelectionActionBar's count label ({N} selected) needs no pluralization helper, unlike toast.ts's markPhrase — the word \"selected\" doesn't change with N, so the same template string is correct for every count."

patterns-established:
  - "A row-level nested-interactive-control problem (a real button inside what was a real button) is resolved by demoting the OUTER element to a div+role+manual-keydown, never by demoting the inner control to non-interactive decoration — preserves the stock, unmodified primitive (Checkbox) as the plan's own \"reuse, don't reinvent\" instruction requires."
  - "untrack() around everything but the intended dependency is the fix pattern for a self-referential $effect (one that both reads and writes the same piece of state, e.g. `++counter`) — apply this BEFORE adding new state the effect resets, not after a bug surfaces, whenever an effect's own doc comment names a narrower dependency than its literal reads imply."

requirements-completed: [KERN-09, KERN-10]

coverage:
  - id: D1
    description: "Ctrl/Cmd-click toggles a row's bulk selection without opening the detail pane; shift-click replaces the selection with the contiguous range from the anchor to that row; a plain click still opens the detail pane and never clears an existing bulk selection (D-01, UI-SPEC E1)"
    requirement: KERN-09
    verification:
      - kind: unit
        ref: "web/src/lib/selection.test.ts#selectRange (all 6 <behavior> bullets)"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/13-multi-select-bulk-exclude.spec.ts#ctrl/shift-click select a range, a plain click never clears it, Esc clears, and Exclude + Undo round-trip"
        status: pass
    human_judgment: false
  - id: D2
    description: "The action bar is absent at zero selected and renders identically for one and many (plurality lives only in the toast copy); Esc and the bar's own Clear are the only two paths that empty the selection"
    requirement: KERN-09
    verification:
      - kind: unit
        ref: "web/src/lib/components/selection-action-bar.test.ts (count-gated render, fixed-template label)"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/13-multi-select-bulk-exclude.spec.ts (steps 1, 2, 3, 5)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Both action-bar buttons disable for the duration of an in-flight bulk write; the primary button is never variant=\"destructive\"; a bulk Exclude removes the selected rows and fires the contract-exact 'Excluded N items' undo toast, and Undo restores them"
    requirement: KERN-09
    verification:
      - kind: unit
        ref: "web/src/lib/components/selection-action-bar.test.ts (disabled={busy}, no destructive variant)"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/13-multi-select-bulk-exclude.spec.ts (steps 6-7)"
        status: pass
    human_judgment: false
  - id: D4
    description: "The excluded-view toggle renders only when excluded_count > 0, fully absent (not disabled) otherwise, sourced from the same stream response that drives the stream itself"
    requirement: KERN-10
    verification:
      - kind: e2e
        ref: "web/e2e/specs/13-excluded-view.spec.ts (steps 1, 3)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Flipping the toggle reuses StreamList/StreamRow/the detail pane/multi-select wholesale (no modal, no new route) and shows exactly the excluded bucket; the bulk primary label reads Include while viewing it"
    requirement: KERN-10
    verification:
      - kind: e2e
        ref: "web/e2e/specs/13-excluded-view.spec.ts (step 4)"
        status: pass
    human_judgment: false
  - id: D6
    description: "When the excluded count reaches zero while the excluded view is showing, the view auto-flips back to the normal stream and the toggle disappears — no sustained empty-excluded-view state, and the auto-flip cannot loop"
    requirement: KERN-10
    verification:
      - kind: e2e
        ref: "web/e2e/specs/13-excluded-view.spec.ts (step 6)"
        status: pass
    human_judgment: false
  - id: D7
    description: "The detail-pane include mirror is reachable only while viewing the excluded bucket, and fires the exact same setItemMarks/toast path the bulk action uses"
    requirement: KERN-10
    verification:
      - kind: other
        ref: "web/src/lib/components/DetailPane.svelte — no diff required; 13-01 already implemented the excluded-state icon/label/onexclude-oninclude branch this task specifies, and +page.svelte now threads excluded={view === 'excluded'}"
        status: pass
    human_judgment: false
  - id: D8
    description: "Multi-select never renders below the 768px breakpoint; the checkbox slot is a fixed-width leading block that changes no other row element's clamp/truncation behaviour"
    requirement: KERN-09
    verification:
      - kind: other
        ref: "web/src/lib/components/StreamRow.svelte — checkbox slot carries max-md:hidden (grep-verified against the plan's own acceptance criterion) and a fixed size-9 width"
        status: pass
    human_judgment: true
    rationale: "The checkbox slot's fixed width and the max-md:hidden breakpoint are proven by source inspection and the plan's own grep-based acceptance criteria (both passed); no e2e spec drives an actual sub-768px viewport against this specific row, matching 13-UI-SPEC.md's own framing that multi-select is explicitly desktop-scoped and its absence below 768px is untested-by-design, not an oversight."
  - id: D9
    description: "The action bar's three-segment layout has no explicit narrow-width collapse and is expected to fit at very narrow desktop widths on the strength of its short fixed copy"
    verification: []
    human_judgment: true
    rationale: "13-UI-SPEC.md's own backstop item — unverified against an actual narrow-desktop render, per the UI-SPEC's explicit framing (\"re-check only if this proves a real problem, not pre-emptively redesigned\")."

duration: ~50min
completed: 2026-08-14
status: complete
---

# Phase 13 Plan 03: Multi-Select, Bulk Exclude, and the Excluded View Summary

**Desktop multi-select on the stream with a floating bulk-action bar, the excluded-items view toggle with a structurally single-shot auto-flip, and two Playwright specs proving both curation round trips against a real booted kernel — the whole user-facing half of KERN-09/KERN-10.**

## Performance

- **Duration:** ~50 min
- **Completed:** 2026-08-14
- **Tasks:** 3
- **Files modified:** 13 (6 created, 7 modified across three commits)

## Accomplishments

- `web/src/lib/selection.ts` — `toggleSelection`/`selectRange`/`clearSelection`, pure and Svelte-free, unit-tested for all six `<behavior>` bullets including the "replace, never union" guarantee (structural: `selectRange` takes no prior-selection argument at all)
- `StreamRow.svelte` gains ctrl/cmd-click toggle and shift-click range-select, a leading hover/focus-revealed checkbox slot, and a `bg-secondary/60` bulk-selected tint that coexists additively with the pre-existing `border-l-primary` open-item accent
- `SelectionActionBar.svelte` — the floating bulk-action bar (`{N} selected` / primary Exclude-or-Include / Clear), absent entirely at zero selected, rendered by `StreamList.svelte` inside the stream pane's own scroll container so it sticks to the pane, not the viewport
- `WebspaceHeader.svelte` gains the `Excluded (N)` toggle in the existing trailing reserved group, absent entirely (not disabled) at zero exclusions, with an accent-tinted pressed state mirroring the existing selected-chip precedent
- `api.ts`'s `getStream` widens to an optional `view` parameter (`?view=excluded` appended only when requested — the default call is byte-identical to before) and `StreamResponse` gains `excluded_count`
- `+page.svelte` ties it together: `bulkSelection`/`bulkAnchor`/`view` state, the full shift-click anchor fallback chain (most-recent ctrl-click → currently-open item → first row), Esc-to-clear, view-driven bulk exclude/include, and a structurally single-shot auto-flip back to the normal stream when the excluded bucket empties
- `web/e2e/specs/13-multi-select-bulk-exclude.spec.ts` and `13-excluded-view.spec.ts` prove both round trips against a real booted kernel and the mock plugin's four-item corpus — all 7 numbered assertions in the multi-select spec and all 6 in the excluded-view spec pass, and the full 133-spec e2e suite (every prior phase's specs, not only Phase 13's) passes clean after these changes

## Task Commits

Each task was committed atomically:

1. **Task 1: Desktop multi-select on the stream and the floating action bar** - `42f9be0` (feat)
2. **Task 2: The excluded-bucket toggle, the include mirror, and the auto-flip** - `79f02f0` (feat)
3. **Task 3: Playwright specs for both round trips** - `655bfe3` (test)

## Files Created/Modified

- `web/src/lib/selection.ts`, `web/src/lib/selection.test.ts` - the pure bulk-selection helpers and their unit tests
- `web/src/lib/components/StreamRow.svelte` - bulk-select props, the checkbox slot, the `<div role="button">` root conversion (see Deviations)
- `web/src/lib/components/StreamList.svelte` - threads bulk-select props to every row, renders `SelectionActionBar`
- `web/src/lib/components/SelectionActionBar.svelte`, `web/src/lib/components/selection-action-bar.test.ts` - the floating bar and its structural test
- `web/src/lib/components/WebspaceHeader.svelte` - the `Excluded (N)` toggle
- `web/src/lib/api.ts` - `StreamResponse.excluded_count`, `getStream(webspace, view?)`
- `web/src/lib/components/sources.test.ts` - mechanical `excluded_count: 0` fixture fix (Rule 3)
- `web/src/routes/w/[webspace]/+page.svelte` - bulk-select/view state and handlers, the `untrack()` fix (see Deviations)
- `web/e2e/specs/13-multi-select-bulk-exclude.spec.ts`, `web/e2e/specs/13-excluded-view.spec.ts` - the two new curation specs
- `docs/testing.md` - documents both new specs alongside the existing 13-01 tracer entry

## Decisions Made

- `StreamRow`'s root element became `<div role="button" tabindex="0">` instead of a real `<button>` — see Deviations. This preserves the stock, unmodified `Checkbox` primitive (the plan's own instruction) rather than hand-rolling a non-interactive checkbox visual.
- `selectRange` takes no "current selection" parameter, so "replace rather than union" (13-UI-SPEC.md E1) is true by construction — there is no code path through which a prior selection could survive into the computed range.
- `handleBulkPrimary`/`handleToggleView` derive action/verb/undo-action from the single `view` state value, never a second mode flag — mirrors the plan's own key_links ("the selection set's size is the only mode indicator") extended to the view axis.
- The auto-flip is a single non-looping recursive `load()` call: `view` is reset to `'included'` before the recursive call, so the auto-flip's own guard condition is structurally false on that call.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `StreamRow.svelte`'s root element converted from `<button>` to `<div role="button" tabindex="0">`**
- **Found during:** Task 1
- **Issue:** The plan's own action text specifies adding "the stock Checkbox" as a leading slot inside the existing row `<button>`. bits-ui's stock Checkbox recipe always renders as a real `<button role="checkbox">` — nesting a `<button>` inside a `<button>` is invalid HTML and, more concretely, breaks click-event isolation: a click on the checkbox bubbles straight into the row's own `onclick`, which would also fire the plain-click "open the detail pane" branch on every checkbox click.
- **Fix:** Converted the row's root to `<div role="button" tabindex="0">`, added `handleActivate`/`handleRowKeydown` to restore native-button Enter/Space activation explicitly (a `<div>` doesn't get this for free), and added `event.stopPropagation()` on the checkbox's own `onclick` so a checkbox click can never also open the detail pane.
- **Files modified:** `web/src/lib/components/StreamRow.svelte`
- **Verification:** `npm --prefix web run test`/`check` pass; the full 133-spec e2e suite passes, including every pre-existing spec that clicks a stream row (`09-*`, `12-*`, `uat-*`, `smoke-*`).
- **Committed in:** `42f9be0` (Task 1 commit)

**2. [Rule 3 - Blocking] `sources.test.ts` fixture updated for the widened `StreamResponse` type**
- **Found during:** Task 2
- **Issue:** `StreamResponse` gained a required `excluded_count: number` field; `sources.test.ts`'s own `makeResponse` fixture builder didn't set it, failing the type check.
- **Fix:** Added `excluded_count: 0` to the fixture's default object literal.
- **Files modified:** `web/src/lib/components/sources.test.ts`
- **Verification:** `npm --prefix web run check` reports zero errors.
- **Committed in:** `79f02f0` (Task 2 commit)

**3. [Rule 1 - Bug] The per-webspace `$effect` self-triggered via its own `++navGeneration` read-and-write, clobbering the excluded-view toggle**
- **Found during:** Task 3, stabilizing `13-excluded-view.spec.ts`
- **Issue:** `+page.svelte`'s per-webspace `$effect` (its own doc comment: "whenever the webspace route param changes") contains `const gen = ++navGeneration;`, which both reads and writes `navGeneration` inside the effect body — making Svelte's dependency tracker treat `navGeneration` as a tracked source of the effect too. Confirmed live via a Playwright trace (network log + response bodies): after toggling to the excluded view, this effect re-ran 1-2 extra times within milliseconds, each time resetting `view` back to `'included'` and re-issuing a stale-view fetch that landed just after the correct `?view=excluded` fetch, clobbering it. This was a pre-existing latent characteristic of the effect (present before this plan), invisible until now because nothing it reset was previously a piece of state a single user action could put into an OBSERVABLY different value from a fresh mount.
- **Fix:** Wrapped the effect body in `untrack()`, reading only `webspace` as the tracked dependency (`const ws = webspace; untrack(() => { ...all existing resets, load(), loadSources(), loadConfig()... });`) — makes the effect's own documented intent true by construction.
- **Files modified:** `web/src/routes/w/[webspace]/+page.svelte`
- **Verification:** `13-excluded-view.spec.ts` and `13-multi-select-bulk-exclude.spec.ts` pass, re-run twice for stability with no flake; the full 133-spec e2e suite passes; `npm --prefix web run test`/`check` pass.
- **Committed in:** `655bfe3` (Task 3 commit)

---

**Total deviations:** 3 auto-fixed (2 Rule 1 bug fixes, 1 Rule 3 mechanical fixture fix)
**Impact on plan:** All three were necessary consequences of implementing the plan's own deliberate additions (the checkbox slot, the widened `StreamResponse` type, the `view` state) colliding with either the stock primitive's own markup shape or a pre-existing, previously-invisible effect characteristic. No scope creep — no plan behavior changed as a result; both bug fixes are corrections, not new features.

## Issues Encountered

- **Debugging the `$effect` self-trigger (Deviation 3) required a real Playwright trace inspection**, not just re-reading the source: the symptom (the excluded view showing the included bucket's 3 items instead of the excluded bucket's 1) looked exactly like a server-side view-filter bug at first glance. A manual, hermetically-launched kernel plus direct `curl` calls to `?view=excluded` proved the kernel was always correct; extracting the failing Playwright run's `trace.zip` (`0-trace.network` + the cached response bodies in `resources/`) showed FIVE `/stream` requests fired within ~600ms of the toggle click, with two extra bare (`view`-less) requests racing the correct `?view=excluded` one — which is what led to inspecting `navGeneration`'s self-referential increment as the actual cause.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The full user-facing curation surface (multi-select, bulk exclude/include, the excluded-items view with its auto-flip) is complete and proven end to end against a real kernel; 13-04/13-05 (PWA installability, already executed in a parallel wave) and 13-06 build on top of this with no further curation-surface work expected from this plan.
- The `untrack()` fix to the per-webspace `$effect` is a general correctness fix, not scoped narrowly to `view` — any future piece of per-webspace UI state this effect resets is now also protected from the same spurious-retrigger class of bug.
- No blockers. `npm --prefix web run test`/`check`/`check:e2e`, the full 133-spec `make e2e`-equivalent Playwright run, and `make docs-check` all pass clean.

---
*Phase: 13-per-item-curation-installable-app*
*Completed: 2026-08-14*

## Self-Check: PASSED

- FOUND: web/src/lib/selection.ts, web/src/lib/selection.test.ts, web/src/lib/components/SelectionActionBar.svelte, web/src/lib/components/selection-action-bar.test.ts, web/e2e/specs/13-multi-select-bulk-exclude.spec.ts, web/e2e/specs/13-excluded-view.spec.ts
- FOUND: commits 42f9be0, 79f02f0, 655bfe3 in `git log --oneline --all`
