---
phase: 06-ui-scalable-source-surface
plan: 05
subsystem: ui
tags: [svelte, tailwind, search-highlighting, design-tokens, source-scan-test]

requires:
  - phase: 06-01
    provides: highlightText/SnippetSegment client-side term derivation and the DetailPane body-variant UI-09 highlighting this plan extends to the title
  - phase: 03-04
    provides: parseSnippet-based search-result snippet rendering in StreamRow.svelte, whose weight-only match treatment this plan retires
provides:
  - A single project-wide .search-highlight class (web/src/app.css @layer components), the one declaration every SPA match surface now resolves through
  - Detail-pane title highlighting (previously only the body was highlighted)
  - Stream-row title highlighting, threaded via a new optional searchQuery prop
  - Search-result snippet match treatment unified from font-semibold weight to the shared amber class
  - A comment-stripped source-scan recurrence guard (search-emphasis.test.ts) proving the wiring across all four surfaces
  - Dated supersession of 03-UI-SPEC.md's weight-only match-emphasis rule, extended UI-09 surface set in 06-UI-SPEC.md, and disambiguated UI-09 wording in REQUIREMENTS.md
affects: [phase-07-webspace-builder-ui, any-future-detail-pane-or-stream-row-work]

actuals:
  tokens: 7800
  tasks: 3
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Shared match-emphasis component class (.search-highlight) declared once in app.css, never in a component-scoped <style> block, so it can be reused across sibling Svelte components"
    - "Comment-stripped source-scan guard (strip <!-- -->, /* */, and // before asserting) as the recurrence-proof instrument for a component-mount-free (environment: 'node') test runner"

key-files:
  created:
    - web/src/lib/components/search-emphasis.test.ts
  modified:
    - web/src/app.css
    - web/src/lib/components/DetailPane.svelte
    - web/src/lib/components/StreamRow.svelte
    - web/src/lib/components/SearchResults.svelte
    - .planning/REQUIREMENTS.md
    - .planning/phases/03-email-in-the-webspace/03-UI-SPEC.md
    - .planning/phases/06-ui-scalable-source-surface/06-UI-SPEC.md

key-decisions:
  - "Promoted .search-highlight from DetailPane.svelte's component-scoped <style> block to web/src/app.css's @layer components — the mechanical fix for why the detail pane and stream row could never share one class"
  - "Retired 03-UI-SPEC.md's Phase 3 weight-only match-emphasis rule with a dated, sourced supersession note that explicitly addresses its own two-weight rationale, rather than silently deleting it"
  - "Fixed 03-UI-SPEC.md's Usage-mapping table and accent-reserved paragraph for consistency with the superseded rule, beyond the plan's two explicitly named edit sites (Rule 1 — leaving them stale would contradict the paragraph immediately above them in the same document)"

patterns-established:
  - "One shared, token-derived component class per cross-component visual concept, declared exactly once in app.css — enforced going forward by a comment-stripped source-scan test counting rule-selector occurrences"

requirements-completed: [UI-09]

coverage:
  - id: D1
    description: "A term matching only an item's title renders a visible amber highlight in both the search-results row title and the detail-pane title"
    requirement: "UI-09"
    verification:
      - kind: unit
        ref: "web/src/lib/components/search-emphasis.test.ts#the title surfaces are wired"
        status: pass
      - kind: manual_procedural
        ref: "06-05-PLAN.md verify human-check: search a webspace for a title-only word, confirm amber highlight in both the search-results row title and the opened detail-pane title"
        status: unknown
    human_judgment: true
    rationale: "The source-scan guard proves the wiring (highlightText applied to item.title, shared class named) but cannot render a live component or a live search result — actual pixel-level visibility in a running browser against real data needs a human or a screenshot-driving UAT pass."
  - id: D2
    description: "The search-result snippet, detail-pane body and kernel rendition iframe all paint the same amber pair; no surface uses a weight change as its match treatment"
    requirement: "UI-09"
    verification:
      - kind: unit
        ref: "web/src/lib/components/search-emphasis.test.ts#the snippet surface is unified"
        status: pass
      - kind: unit
        ref: "web/src/lib/components/search-emphasis.test.ts#one declaration, one place"
        status: pass
    human_judgment: false
  - id: D3
    description: "With an empty search query every touched surface renders as it did before this plan (searchQuery defaults to '', StreamRow unfiltered stream never passes it)"
    requirement: "UI-09"
    verification:
      - kind: unit
        ref: "web/src/lib/components/search-emphasis.test.ts#the query reaches the row"
        status: pass
      - kind: unit
        ref: "cd web && npx vitest run (full 203-test suite, all green)"
        status: pass
    human_judgment: false
  - id: D4
    description: "search-emphasis.test.ts fails if any one of those surfaces is reverted"
    verification:
      - kind: manual_procedural
        ref: "Manually reverted StreamRow.svelte's snippet class back to font-semibold during execution; confirmed 2 of 15 guard assertions failed with the expected messages, then restored"
        status: pass
    human_judgment: false
  - id: D5
    description: "03-UI-SPEC.md, 06-UI-SPEC.md and REQUIREMENTS.md agree with the shipped behaviour"
    verification:
      - kind: other
        ref: "grep -q 'G-06-1' .planning/phases/03-email-in-the-webspace/03-UI-SPEC.md && grep -qi 'title' .planning/REQUIREMENTS.md && sed -n '/UI-09/p' .planning/REQUIREMENTS.md | grep -qi 'title'"
        status: pass
      - kind: manual_procedural
        ref: "Read the amended 03-UI-SPEC.md paragraph and confirm it reads as a deliberate, reasoned supersession rather than a silent deletion"
        status: unknown
    human_judgment: true
    rationale: "The automated grep confirms the marker strings exist; whether the amendment 'reads as' a deliberate, well-reasoned supersession is a prose-quality judgment the plan itself calls out as a human-check."

duration: ~15min
completed: 2026-08-07
status: complete
---

# Phase 06 Plan 05: Unify Search-Match Emphasis Summary

**Closed gap G-06-1 by promoting search-match highlighting to one shared `.search-highlight` class in app.css, applying it to the previously-unhighlighted detail-pane and stream-row titles, retiring the stream/search-row's Phase 3 weight-only treatment, and adding a comment-stripped recurrence guard plus dated spec amendments.**

## Performance

- **Duration:** ~15 min
- **Completed:** 2026-08-07
- **Tasks:** 3
- **Files modified:** 8 (4 source, 1 new test, 3 planning docs)

## Accomplishments

- One project-wide `.search-highlight` class, declared exactly once in `web/src/app.css`'s `@layer components`, resolving through the existing `--warning`/`--background` tokens — no new color introduced.
- Detail-pane title (`DetailPane.svelte`) now highlights matched search terms, closing the "title-only FTS match renders zero highlight anywhere" functional gap.
- Stream-row title (`StreamRow.svelte`) highlights matched terms via a new optional `searchQuery` prop (default `''`, so the unfiltered stream stays byte-identical).
- Search-result snippet's match treatment changed from a barely-visible `font-semibold` weight to the same amber class the detail pane and kernel iframe already use.
- `SearchResults.svelte` threads the live query down to each row as `searchQuery`.
- New `search-emphasis.test.ts`: a comment-stripped source-scan guard (15 assertions across 5 groups) proving one declaration, no component-scoped redeclaration, both title surfaces wired, the snippet surface unified, and the query reaching the row — manually verified to fail loudly when the snippet treatment is reverted.
- `03-UI-SPEC.md`'s Phase 3 weight-only match-emphasis rule superseded with a dated, sourced rationale that explicitly addresses (and improves on) its own "exactly 2 weights" justification.
- `06-UI-SPEC.md`'s UI-09 section extended to name the item title as a highlight surface and explain why (title/preview are both FTS-indexed).
- `REQUIREMENTS.md`'s UI-09 wording disambiguated to name both the title and the rendered content, across both the detail pane and the search-results row.

## Task Commits

1. **Task 1: Promote the match treatment to one shared class and apply it to every SPA match surface** - `65407cc` (feat)
2. **Task 2: Recurrence guard — a title-only match must render a visible highlight somewhere** - `d2a998c` (test)
3. **Task 3: Amend the design contracts this change overturns** - `8cdd7de` (docs)

_No separate plan-metadata commit beyond the final STATE/ROADMAP commit below (`commit_docs` handled by the SDK's final-commit step)._

## Files Created/Modified

- `web/src/app.css` - Added the shared `.search-highlight` component class inside `@layer components`
- `web/src/lib/components/DetailPane.svelte` - Deleted the component-scoped `<style>` block; title now renders through `highlightText`/shared class
- `web/src/lib/components/StreamRow.svelte` - New `searchQuery` prop; title highlighted; snippet's match class swapped from `font-semibold` to `search-highlight`
- `web/src/lib/components/SearchResults.svelte` - Passes `searchQuery={query}` down to `StreamRow`
- `web/src/lib/components/search-emphasis.test.ts` - New comment-stripped source-scan recurrence guard (created)
- `.planning/REQUIREMENTS.md` - UI-09 wording disambiguated
- `.planning/phases/03-email-in-the-webspace/03-UI-SPEC.md` - Weight-only match-emphasis rule superseded; E2 table row and accent-usage paragraph updated for consistency
- `.planning/phases/06-ui-scalable-source-surface/06-UI-SPEC.md` - UI-09 surface set extended to the item title, with rationale

## Decisions Made

- Promoted the class to `app.css` rather than duplicating it in both components — a Svelte scoped `<style>` class cannot be shared across sibling components, which is the mechanical root cause of the original drift.
- Superseded (not deleted) the Phase 3 weight-only rule, with a dated note naming G-06-1 and this plan, and an explicit argument that the new colour-only treatment *more purely* satisfies the original "exactly 2 weights" rationale than the retired weight-based exception did.
- Also updated 03-UI-SPEC.md's Usage-mapping table and accent-reserved paragraph (not explicitly named by the plan's two edit sites) for internal consistency — leaving them stale would have contradicted the just-amended paragraph in the same document (Rule 1).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug/doc inconsistency] Updated two additional 03-UI-SPEC.md locations beyond the plan's named edit sites**
- **Found during:** Task 3
- **Issue:** The plan named exactly two edit locations (the Typography paragraph and the E2 table row). Two more locations in the same document — the "Usage mapping" table's Body row and the Accent-reserved paragraph's "uses weight, not color" parenthetical — restated the same retired rule and would have been left contradicting the just-superseded paragraph directly above them.
- **Fix:** Updated both to reflect the shared amber treatment, cross-referencing the superseded-rule note.
- **Files modified:** `.planning/phases/03-email-in-the-webspace/03-UI-SPEC.md`
- **Verification:** `grep -q 'G-06-1'` passes; full document re-read for internal consistency.
- **Committed in:** `8cdd7de` (Task 3 commit)

---

**Total deviations:** 1 auto-fixed (1 doc-consistency bug)
**Impact on plan:** Documentation-only, same file already in scope. No scope creep into code or architecture.

## Issues Encountered

None — the debug session's root-cause diagnosis (`.planning/debug/search-highlight-title-and-stream.md`) was accurate and complete going in, so no new investigation was needed during execution.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- G-06-1 is closed: both RC-1 (missing title highlight) and RC-2 (weight/colour vocabulary drift) are fixed, with an automated recurrence guard in place.
- Remaining verification: the plan's `<verify><human-check>` (live browser check of a title-only match) and the Task 3 human-check (prose-quality read of the superseded-rule note) are recorded as `human_judgment: true` coverage items (D1, D5) for the phase's end-of-phase UAT pass — not run in this execution.
- No blockers for Phase 7 (Webspace Builder UI) or any future work touching `DetailPane.svelte`/`StreamRow.svelte`.

---
*Phase: 06-ui-scalable-source-surface*
*Completed: 2026-08-07*

## Self-Check: PASSED

All 9 claimed files found on disk; all 4 claimed commit hashes (`65407cc`, `d2a998c`, `8cdd7de`, `cb5e9a0`) found in git history.
