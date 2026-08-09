---
phase: 07-webspace-builder-ui
plan: 14
subsystem: ui
tags: [svelte, svelte5-runes, config-edit, participation, gap-closure, uat]

# Dependency graph
requires:
  - phase: 07-webspace-builder-ui
    provides: "07-11's participation.ts module (null-tolerant readers, isEmptyWebspaceShell, D-20 has-match-input rule in matchFieldsFor) and 07-13's AddSourceModal.svelte"
provides:
  - "participatingInstances / participatesIn — the single client-side mirror of the kernel's effective participation (allowlist gate AND has-match-input), added to participation.ts"
  - "removeSourceFromWebspace seeds the current participant set before filtering, mirroring addSourceToWebspace"
  - "WebspaceHeader.svelte's chip content (visible/hidden slices, measurement clones, overflow count) filtered through the shared predicate; row visibility deliberately still unfiltered"
  - "AddSourceModal.svelte's inline participant-set derivation replaced by the shared helper"
affects: [07-webspace-builder-ui, any-future-webspace-participation-consumer]

actuals:
  tokens: 8683
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Shared client-side predicate module (participation.ts) mirrors two kernel functions (Webspace.Participates + matchFieldsFor's has-match-input rule) — every UI surface reading webspace participation calls the one helper instead of re-deriving it"
    - "Seed-then-filter allowlist mutation: both addSourceToWebspace and removeSourceFromWebspace materialise the current participant set from the input document before mutating, so an implicit all-participate default never round-trips through an empty-array no-op"
    - "Deliberate two-gate split in a single component: row VISIBILITY (shouldShowSourceRows) and row CONTENT (participatingSources) read different lists on purpose, each pinned by a positive source-scan assertion so a future edit can't silently merge them"

key-files:
  created:
    - web/src/lib/components/webspace-participation.test.ts
  modified:
    - web/src/lib/participation.ts
    - web/src/lib/participation.test.ts
    - web/src/lib/config-edit.ts
    - web/src/lib/config-edit.test.ts
    - web/src/lib/components/WebspaceHeader.svelte
    - web/src/lib/components/AddSourceModal.svelte

key-decisions:
  - "Both defects (the config-write no-op and the unfiltered chip row) were fixed in one plan because either alone leaves the reported symptom unchanged — fixing only the write still resolves to all-participate under the corrected filter's own logic once the header consults it; fixing only the header still shows a stale chip because the write never actually narrowed participation."
  - "The shared predicate mirrors 07-11's post-fix kernel semantics (allowlist gate AND has-match-input), not the pre-07-11 allowlist-gate-only semantics AddSourceModal's old inline participatingSet implemented — a 07-11 empty shell would otherwise show every configured instance as a participant."
  - "Row visibility (shouldShowSourceRows) and chip content (participatingSources) are deliberately driven by different lists in WebspaceHeader.svelte — filtering the visibility gate would hide the \"+\" add-source trigger for any webspace with zero participants, replacing one dead end with another."
  - "The last-named-entry-removed boundary is pinned, not redesigned: an explicit allowlist narrowed to empty is indistinguishable from the all-participate default in the current config format, so a webspace with a keywords fallback sees its remaining instances rejoin. Documented in removeSourceFromWebspace's doc comment and pinned by a test, per the plan's own planning choice 5."

patterns-established:
  - "New pure predicates that mirror kernel logic get their tests written FIRST against a fixture with enough distinct instances (3, not 2) to distinguish every branch of the logic — a 2-instance fixture cannot tell 'some but not all participate' apart from either extreme."

requirements-completed: [KERN-08, UI-12]

coverage:
  - id: D1
    description: "participatingInstances/participatesIn added to participation.ts as the single client-side mirror of kernel participation semantics (allowlist gate AND has-match-input)"
    requirement: "UI-12"
    verification:
      - kind: unit
        ref: "web/src/lib/participation.test.ts#describe('participatingInstances / participatesIn')"
        status: pass
    human_judgment: false
  - id: D2
    description: "removeSourceFromWebspace seeds the current participant set (existing allowlist, or every configured instance) before filtering, so removing a source from an all-participate webspace actually narrows participation"
    requirement: "KERN-08"
    verification:
      - kind: unit
        ref: "web/src/lib/config-edit.test.ts#describe('removeSourceFromWebspace') — seeding cases"
        status: pass
    human_judgment: false
  - id: D3
    description: "WebspaceHeader.svelte's chip row (visible/hidden slices, measurement clones, overflow-trigger count) filters through the shared predicate; row visibility (the \"+\" trigger) deliberately keeps reading the unfiltered sources prop"
    requirement: "UI-12"
    verification:
      - kind: unit
        ref: "web/src/lib/components/webspace-participation.test.ts"
        status: pass
    human_judgment: true
    rationale: "The source-scan guard proves the derivations are wired correctly, but the actual live behaviour — the chip visually disappearing with no reload after 'Remove from this webspace', config.toml correctly narrowed, and the '+' picker remaining usable when a webspace's last source is removed — is exactly what 07-UAT.md G-07-6 originally reported as broken from live use, and can only be confirmed against a running kernel via `make dev` per the plan's own human-check."
  - id: D4
    description: "AddSourceModal.svelte's inline participatingSet replaced by the shared participatingInstances helper, so the add-picker and the header's chip row can never disagree"
    requirement: "UI-12"
    verification:
      - kind: unit
        ref: "web/src/lib/components/webspace-participation.test.ts#describe('AddSourceModal.svelte: no longer computes its own participant set')"
        status: pass
    human_judgment: false

duration: 8min
completed: 2026-08-09
status: complete
---

# Phase 7 Plan 14: Remove-from-webspace gap closure (G-07-6) Summary

**Fixed both G-07-6 defects together — `removeSourceFromWebspace` now seeds the current participant set before filtering (mirroring `addSourceToWebspace`), and `WebspaceHeader.svelte`'s chip row now filters through a shared `participation.ts` predicate instead of rendering the kernel-wide, unfiltered `GET /api/sources` list verbatim.**

## Performance

- **Duration:** ~8 min (git-log span across the three task commits; reading/context time not separately tracked)
- **Started:** 2026-08-09T13:37:18+01:00 (first task commit)
- **Completed:** 2026-08-09T13:44:27+01:00 (final task commit)
- **Tasks:** 3
- **Files modified:** 7 (6 modified, 1 created)

## Accomplishments

- `participation.ts` gained `participatingInstances`/`participatesIn` — the single client-side mirror of the kernel's effective participation (Phase 5 D-03's allowlist gate AND 07-11's D-20 has-match-input rule), built on the module's existing null-tolerant readers.
- `removeSourceFromWebspace` now seeds the current participant set (the existing allowlist when non-empty, else every configured instance) from its own `cfg` input *before* filtering — mirroring `addSourceToWebspace`'s long-standing pattern. The empty-allowlist "all-participate" case, previously a silent no-op, was confirmed RED and is now fixed and covered.
- `WebspaceHeader.svelte`'s chip content (the visible slice, the hidden/overflow slice, the off-screen measurement clones, and the overflow-trigger's count label) now derives from `participatingSources` — the `sources` prop filtered through `participatesIn` — rather than the raw prop. Row visibility (`shouldShowSourceRows`, and by extension the "+" add-source trigger) deliberately keeps reading the unfiltered prop, so a webspace with zero participants (every freshly created one, or one whose last source was just removed) never loses its only way to add a source back.
- `AddSourceModal.svelte`'s inline `participatingSet` derivation (which implemented only the allowlist-gate half of participation, and was the second half of `G-07-6`'s divergence) was deleted and replaced by the shared `participatingInstances` helper.
- Added `webspace-participation.test.ts`, a source-scan guard over both components, following the house comment-stripped-source pattern (`add-source.test.ts`). It pins: the filtered derivations in the header, the deliberate row-visibility split, the removal of `AddSourceModal`'s local participant set, and that each shared-helper call site appears exactly once.

## Task Commits

Each task was committed atomically:

1. **Task 1: One client-side definition of what it means to participate in a webspace** - `68bb5a0` (feat)
2. **Task 2: Removing a source from a webspace writes a document that really removes it** - `66534cf` (fix)
3. **Task 3: The chip row shows this webspace's sources, and there is only one implementation of that idea** - `154cc83` (fix)

_No separate plan-metadata commit was made prior to this SUMMARY; the metadata commit follows this file per the execute-plan workflow._

## Files Created/Modified

- `web/src/lib/participation.ts` - Added `participatingInstances`/`participatesIn`, the shared effective-participation predicate (6 exported functions total; the 4 from 07-11 unchanged)
- `web/src/lib/participation.test.ts` - Added a `describe('participatingInstances / participatesIn')` block with a three-instance fixture covering every case in the plan's `<behavior>` table, plus a no-throw-on-null assertion
- `web/src/lib/config-edit.ts` - `removeSourceFromWebspace` now seeds from `cfg.webspaces[webspace]`/`cfg.sources` (the pre-mutation input) before filtering; doc comment rewritten to explain why filtering directly was wrong and to name the pinned last-entry boundary
- `web/src/lib/config-edit.test.ts` - Added a `carsConfig` three-instance fixture and 8 new cases to the existing `removeSourceFromWebspace` describe block (existing 3 cases untouched)
- `web/src/lib/components/WebspaceHeader.svelte` - Added the `participatingSources` derivation; pointed the visible slice, hidden slice, the sources-keyed effect, the measurement clone loop, and the overflow-trigger label at it; left `shouldShowSourceRows` reading the raw `sources` prop
- `web/src/lib/components/AddSourceModal.svelte` - Deleted the inline `participatingSet` derivation; `availableInstances` now filters against `participatingInstances(config, webspace)`
- `web/src/lib/components/webspace-participation.test.ts` - New source-scan guard (15 assertions) over both components, confirmed RED against the pre-change files

## Decisions Made

- Both defects fixed in one plan, not split across two — either fix alone leaves `G-07-6`'s reported symptom unchanged (see plan objective and this file's `key-decisions` frontmatter).
- The shared predicate mirrors 07-11's *post-fix* kernel semantics (allowlist gate AND has-match-input), deliberately more restrictive than `AddSourceModal`'s old allowlist-gate-only logic, so a 07-11 empty shell correctly shows zero participants instead of "every configured instance."
- Row visibility and chip content stay on two different lists in `WebspaceHeader.svelte` by design — filtering the visibility gate would hide the "+" trigger for the exact webspaces (zero-participant ones) that most need it.
- The last-named-entry-removed boundary (an explicit allowlist narrowed to empty reverting to the all-participate default when a `keywords` fallback exists) is pinned by a test and documented as a known, intentional limitation of the current config format — not something this gap-closure plan redesigns.
- Config-null fallback in `WebspaceHeader.svelte`: `participatingSources` falls back to the unfiltered `sources` prop when `config` is still `null` (the first render, before `loadConfig`/`loadSources`'s independent, unsequenced fetches resolve) rather than filtering to an empty list — this avoids a flash of "no chips" before the correct filtered set appears a moment later. Not explicitly specified in the plan; a minimal, conservative choice consistent with the existing `{#if config}`-gated "+" trigger pattern already in the file.

## Deviations from Plan

None - plan executed exactly as written. The config-null fallback noted above under "Decisions Made" is a small implementation detail filling a gap the plan's `<action>` text did not explicitly specify (how to handle the header rendering before `config` resolves); it does not change any documented behavior, prohibition, or acceptance criterion.

## RED Confirmations (recorded per plan's `<output>` requirement)

**Task 2 — empty-allowlist `removeSourceFromWebspace` case**, run against the unmodified (pre-fix) function:

```
FAIL  src/lib/config-edit.test.ts > removeSourceFromWebspace > the reported G-07-6 case: an all-participate webspace (no explicit allowlist) loses exactly the removed instance, not nothing …
AssertionError: expected [] to deeply equal [ 'a', 'c' ]
- Expected: ["a", "c"]
+ Received: []

FAIL  src/lib/config-edit.test.ts > removeSourceFromWebspace > seeds every configured instance without throwing when the allowlist arrives as null
TypeError: Cannot read properties of null (reading 'filter')
 ❯ removeSourceFromWebspace src/lib/config-edit.ts:190:29
```

**Task 3 — component source-scan test**, run against the unmodified (pre-fix) components: 11 of 15 assertions failed, including:

```
AssertionError: the hidden overflow-trigger clone must report the filtered count … expected false to be true
AssertionError: AddSourceModal must not keep its own copy of the participation predicate … expected 2 to be +0
AssertionError: the picker's "not yet in this webspace" list must be computed via the same shared predicate … expected false to be true
AssertionError: a second call site (or a hand-rolled equivalent) would let the header's filter drift … expected +0 to be 1
AssertionError: a second call site would risk the picker and its own internal logic disagreeing … expected +0 to be 1
```

(The remaining 4 assertions passed against the pre-change files by design — the non-empty-source guards, and `shouldShowSourceRows` being called with the raw `sources` prop, which was already correct before this plan.)

Both RED runs were reproduced by temporarily stashing the fix (`git stash push -- <file>`, run tests, `git stash pop`), not by writing a separate broken commit.

## Test/Build Results (all run from the fully-fixed tree)

- `cd web && npm test` — **559 → 574 tests, 34 test files, all passed** (every pre-existing header/chip/add-source test passed with an unmodified body)
- `cd web && npm run check` — **0 errors**, 9 pre-existing warnings (unrelated `$state`-referenced-locally notices in files this plan didn't touch)
- `cd web && npm run build` — **exits 0**
- `CGO_ENABLED=0 go build ./...` — clean (this plan touches no kernel file)
- `go test ./kernel/... -count=1` — **all packages ok**
- `git diff --stat kernel/ plugins/ proto/ web/package.json web/package-lock.json` — no output attributable to this plan (a pre-existing, unrelated `kernel/webui/build/.gitkeep` deletion predates this session and was not touched by any task here)
- `git diff web/src/routes/w/[webspace]/+page.svelte web/src/lib/format.ts` — empty
- `grep -c 'participatingSet' web/src/lib/components/AddSourceModal.svelte` — `0`
- `git diff --stat` across the three task commits touches exactly the 7 files in the plan's `files_modified` — no more, no less

## Row-Visibility Gate Confirmation

`WebspaceHeader.svelte`'s `showSourceRows` derivation still reads the raw, unfiltered `sources` prop:
```ts
let showSourceRows = $derived(shouldShowSourceRows(sourcesState, sources));
```
This is deliberate (planning choice 3 in 07-14-PLAN.md): row visibility answers "does this installation have any configured source instances at all", not "which of them belong to this webspace." The `{#if showSourceRows}` block encloses the "+" `AddSourceModal` trigger and Refresh all — filtering the gate's own input would hide the "+" for exactly the webspaces (zero-participant ones, including every freshly-created 07-11 shell and any webspace whose last source this plan's Task 2 lets you remove) that most need it. Pinned by `webspace-participation.test.ts`'s positive assertion that `shouldShowSourceRows(sourcesState, sources)` is still called with the raw prop, and by the assertion that the "+" mount is gated only on `config`, never on `participatingSources.length`.

## URL `?sources=` Selection Case (planning choice 6) — Deliberately Not Changed

A source filter selected via the URL (`?sources=`) can name an instance that no longer participates in the current webspace after a removal. `resolveSourceFilters` (`web/src/lib/format.ts`, untouched by this plan — confirmed via the empty `git diff`) already degrades per-member against the *configured*-instance list, so nothing throws; the stale selection simply matches no visible chip. Narrowing that resolution to *participating* instances would change URL-persistence semantics settled in Phase 6 (D-02) and is explicitly out of scope for this gap-closure plan.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `07-UAT.md` `G-07-6` is closed: the config write genuinely narrows participation for an all-participate webspace, and the header's chip row genuinely reflects that narrowing.
- Exactly one client-side implementation of webspace participation now exists (`participation.ts`'s `participatingInstances`/`participatesIn`), consumed identically by `config-edit.ts`, `AddSourceModal.svelte`, and `WebspaceHeader.svelte`.
- The plan's `<verification>` `<human-check>` (live `make dev` confirmation that the chip disappears immediately, `config.toml` narrows correctly, the instance's own `[sources.<id>]` block survives, other webspaces are unchanged, the "+" picker re-offers the removed instance, and the "+" trigger survives removing every source one by one) has **not** been run in this execution environment — no live kernel session was available. This is the same category of gap 07-04-SUMMARY.md flagged for the original (now-fixed) implementation; it should be exercised at the next opportunity to have a running `make dev` session, per the plan's own `<human-check>` block.
- No further work is scoped for `G-07-6`; the phase's remaining open item is the URL `?sources=`-naming-a-non-participant case, explicitly flagged as deliberately out of scope (planning choice 6) rather than deferred as a gap.

---
*Phase: 07-webspace-builder-ui*
*Completed: 2026-08-09*
