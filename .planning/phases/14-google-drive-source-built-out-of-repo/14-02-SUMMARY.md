---
phase: 14-google-drive-source-built-out-of-repo
plan: 02
subsystem: ui
tags: [svelte5, bits-ui, playwright, accessibility, aria-describedby, sr-only]

# Dependency graph
requires:
  - phase: 13-per-item-marks-and-pwa
    provides: SourceChip.svelte's tooltip precedence chain (12-11-PLAN.md CR-01) and the manifest-unverified/shadowed branches (13-06-PLAN.md) this plan's suppression fix layers on top of, unchanged
provides:
  - SourceChip.svelte's outer filter button and inner display-name span with no native-tooltip title attribute
  - A visually-hidden sr-only description span (id via $props.id()) wired to the button through aria-describedby, carrying the identical tooltipText sentence the app's own styled Tooltip renders
  - source-chip-tooltip.test.ts structural coverage pinning the suppression and the chosen replacement surface
  - Playwright specs repointed onto toHaveAccessibleDescription for every site that used to read the removed title attribute
affects: [14-03, 14-04, any future SourceChip.svelte edit, any future chip Playwright locator]

# Actuals (#2632)
actuals:
  tokens: 6269
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "aria-describedby + visually-hidden sr-only sibling span, id from Svelte 5's $props.id() (not a data-derived id), for a component that renders multiple simultaneous DOM instances of the same logical entity (visible row + aria-hidden measurement clone + overflow-popover clone)"
    - "Playwright toHaveAccessibleDescription for asserting an aria-describedby-exposed sentence, replacing toHaveAttribute('title', ...) reads"

key-files:
  created: []
  modified:
    - web/src/lib/components/SourceChip.svelte
    - web/src/lib/components/source-chip-tooltip.test.ts
    - web/src/lib/components/source-chip-pill.test.ts
    - web/e2e/specs/09-1-header-touch.spec.ts
    - web/e2e/specs/11-binary-changed-repin.spec.ts
    - web/e2e/specs/11-external-tier-badge.spec.ts
    - web/e2e/specs/12-external-rehearsal.spec.ts
    - web/e2e/specs/12-tooltip-precedence.spec.ts
    - web/e2e/specs/12-zero-match-diagnostic.spec.ts
    - web/e2e/specs/13-manifest-unverified.spec.ts
    - web/e2e/specs/13-shadowed-advisory.spec.ts

key-decisions:
  - "Task 1 checkpoint: option-b selected verbatim — preserve current accessible-name semantics via a visually-hidden sr-only description span plus aria-describedby on the chip filter button, rather than moving the health sentence into aria-label. This is a deliberate deviation from 14-UI-SPEC.md § G1 point 1's literal prescription (aria-label={tooltipText}). Reason: accessible-name preservation — G1's stated rationale that aria-label 'preserves the exact same accessible-name text' does not hold against the current markup, where the button's accessible name is computed from its own text content (the display name alone) and the removed title attribute only ever contributed an accessible DESCRIPTION. Moving the sentence to aria-label would have replaced the name 'Household Docs (external)' with the full sentence 'Household Docs (external) — synced 5 minutes ago — untrusted external plugin', breaking ~24 Playwright locators that match chips by exact display name and every future chip locator with them."
  - "chipDescId uses Svelte 5's $props.id() rather than an id derived from source.name (as the plan's Task 2 <action> literally specified). SourceChip.svelte renders more than one simultaneous DOM instance of the same source (WebspaceHeader.svelte's visible row, its aria-hidden measurement clone, and the overflow-popover clone), so a name-derived id would produce duplicate DOM ids across those instances — $props.id() is SSR-safe and unique per component instance, the correct primitive for exactly this case."
  - "Class 1 spec assertions (7 sites reading the removed title attribute directly) were repointed to Playwright's toHaveAccessibleDescription matcher rather than manually resolving aria-describedby -> element id -> textContent — it reads the computed accessible description the same way assistive tech does, is a retrying web-first assertion, and needed no per-site id-lookup helper."
  - "Class 2 locators (getByRole('button', { name: displayName, exact: true })) required NO changes under option-b, confirmed by running the full 46-file/139-test Playwright suite rather than relying on grep alone — the button's accessible name is unaffected by this plan's change."

patterns-established:
  - "When a Svelte component instance needs a stable per-instance DOM id for aria-* wiring and the component is known or suspected to render more than once simultaneously for the same logical entity, use $props.id() over any props-derived string."

requirements-completed: [SRC-06]

coverage:
  - id: D1
    description: "SourceChip.svelte renders exactly one popover on hover (the app's own styled Tooltip) — the native-tooltip title attribute is removed from both the outer filter button and the inner truncated display-name span, leaving the dropdown-footer pinned-hash span untouched"
    requirement: SRC-06
    verification:
      - kind: unit
        ref: "web/src/lib/components/source-chip-tooltip.test.ts#native-tooltip suppression (14-02-PLAN.md, option-b): no title on popover-bearing elements"
        status: pass
    human_judgment: false
  - id: D2
    description: "The chip's full health sentence (including the untrusted-external-plugin clause) stays reachable by assistive technology via a visually-hidden sr-only span wired through aria-describedby, without changing the button's accessible name"
    requirement: SRC-06
    verification:
      - kind: unit
        ref: "web/src/lib/components/source-chip-tooltip.test.ts#the outer chip button is wired to the replacement description via aria-describedby={chipDescId}"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/11-external-tier-badge.spec.ts#the mockstrict chip health description discloses \"untrusted external plugin\"; the mock chip health description does not"
        status: pass
    human_judgment: false
  - id: D3
    description: "The full Playwright suite passes against the changed chip markup — every site that used to read the removed title attribute is repointed to the accessible-description surface, and no locator was left matching an attribute the component no longer renders"
    verification:
      - kind: e2e
        ref: "make e2e (139/139 passed, chromium project)"
        status: pass
      - kind: other
        ref: "npm run check:e2e (tsc over e2e project, 0 errors)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Known, deliberate regression: a touch user on a source chip below 768px who is not running a screen reader can no longer reach chip health detail at all (09.1-04-PLAN.md R2's long-press native-title touch degrade is removed under 14-UI-SPEC.md G1 and neither Task 1 option restores an equivalent touch-only affordance) — a human should confirm this tradeoff is acceptable or route a follow-up"
    verification: []
    human_judgment: true
    rationale: "This is a product/UX tradeoff (losing a previously-shipped touch accessibility guarantee), not something a passing or failing automated check can validate as 'correct' — a human needs to decide whether the regression is acceptable as-is or needs a follow-up fix (e.g., a tap-to-reveal affordance for touch)."

duration: ~20min (continuation session, resuming after Task 1's checkpoint was resolved)
completed: 2026-08-15
status: complete
---

# Phase 14 Plan 02: Suppress Native Tooltips on SourceChip Summary

**Removed the browser-native tooltip that duplicated SourceChip's own styled popover, replacing it with an aria-describedby-wired sr-only description (option-b) so the chip's accessible name stays just the display name — no Playwright locator broke on that axis — while repointing the 9 spec assertions that read the removed `title` attribute directly onto Playwright's `toHaveAccessibleDescription` matcher.**

## Performance

- **Duration:** ~20min (continuation session; Task 1's checkpoint had already been resolved by the orchestrator before this session started)
- **Tasks:** 3/3 (Task 1 checkpoint decision recorded as resolved; Tasks 2 and 3 executed and committed)
- **Files modified:** 11 (2 component files + 2 component test files + 7 Playwright spec files)

## Accomplishments

- `SourceChip.svelte`'s outer filter button and inner truncated display-name span no longer carry a native `title` attribute — hovering a chip now shows exactly one popover (the app's own `Tooltip`), closing the Phase 13 UAT-reported defect this plan's folded todo exists for.
- The chip's health sentence (including the untrusted-external-plugin disclosure and every named health cause) stays reachable to assistive technology as the button's accessible DESCRIPTION via a new visually-hidden `sr-only` sibling span, wired through `aria-describedby={chipDescId}` — `chipDescId` comes from Svelte 5's `$props.id()`, not a `source.name`-derived string, because this component renders more than one simultaneous DOM instance of the same source (the visible chip row, `WebspaceHeader.svelte`'s `aria-hidden` measurement clone, and the overflow-popover clone) and a name-derived id would have collided across them.
- The chip button's accessible NAME is unchanged (still just `source.display_name`'s own text content) — confirmed by running the full 46-file, 139-test Playwright suite unmodified on that axis, not merely by grep.
- `source-chip-tooltip.test.ts` gained a new `describe` block (6 assertions) pinning: no `title` on either popover-bearing element; the dropdown-footer pinned-hash `title` is untouched; `aria-describedby={chipDescId}` is present; the `sr-only` span renders the exact `tooltipText` expression; and no `aria-label={tooltipText}` was introduced (proving option-a was NOT taken).
- 7 Playwright spec sites across 5 files that read the removed `title` attribute directly (a defect present under either Task 1 option) were repointed to `toHaveAccessibleDescription`, preserving each assertion's original polarity, expected substring/regex, and exact-string contract sentences.

## Task Commits

1. **Task 1: Decide the chip's accessible-name semantics before touching the markup** — checkpoint, resolved by the user (option-b) before this session started; no commit of its own (decision recorded in this SUMMARY per the plan's own acceptance criteria).
2. **Task 2: Remove the native tooltips from the chip and pin it with the component test** — `95ee38a` (feat)
3. **Task 3: Bring the Playwright suite onto the chip's new locator surface** — `9b03e7b` (test)

**Plan metadata:** (this commit, docs: complete plan)

## Files Created/Modified

- `web/src/lib/components/SourceChip.svelte` — outer button: `title={tooltipText}` and the inner span's `title={source.display_name}` removed; `aria-describedby={chipDescId}` added to the button; a new `sr-only` sibling `<span id={chipDescId}>{tooltipText}</span>` added; R2 comment block replaced with the new arrangement's rationale, including a note that bits-ui's own transient `aria-describedby` (set while the Tooltip is open) is harmlessly overridden by this explicit one since both reference the identical sentence.
- `web/src/lib/components/source-chip-tooltip.test.ts` — new `describe` block, 6 structural assertions, all existing branch-sentence assertions untouched.
- `web/src/lib/components/source-chip-pill.test.ts` — the pre-existing "touch health detail" test (09.1-04-PLAN.md R2) updated from asserting a native `title` IS present to asserting it is NOT present, with a comment documenting the regression this represents.
- `web/e2e/specs/09-1-header-touch.spec.ts` — test 5 ("chip health detail is reachable without hover, via a native title") rewritten to assert the new behaviour and document the same regression; renamed to make the new contract explicit.
- `web/e2e/specs/11-binary-changed-repin.spec.ts` — 2 sites: the binary-changed positive-tooltip assertion and the recovered-chip negative assertion, both repointed to `toHaveAccessibleDescription`.
- `web/e2e/specs/11-external-tier-badge.spec.ts` — the positive/negative untrusted-clause pair repointed.
- `web/e2e/specs/12-external-rehearsal.spec.ts` — the untrusted-clause assertion repointed.
- `web/e2e/specs/12-tooltip-precedence.spec.ts` — both CR-01 precedence assertions (unreachable-since, synced-plus-advisory) repointed; added a local `escapeRegExp` helper mirroring `12-zero-match-diagnostic.spec.ts`'s existing inline escape; `describe` header comment updated to name the new surface.
- `web/e2e/specs/12-zero-match-diagnostic.spec.ts` — the dynamically-built advisory-substring regex assertion repointed.
- `web/e2e/specs/13-manifest-unverified.spec.ts` — the contract-exact manifest-unverified sentence assertion repointed.
- `web/e2e/specs/13-shadowed-advisory.spec.ts` — the contract-exact shadowed sentence assertion repointed.

Untouched, as the plan and its option-b resume instructions required: `09-chip-menu.spec.ts`, `09-plugin-icon.spec.ts`, `11-untrusted-add.spec.ts`, `header-branding.spec.ts`, `smoke-search-filter.spec.ts`, and `09-picker-groups.spec.ts` (its `title` assertion is on a picker catalog tile, explicitly out of G1's scope).

## Decisions Made

See `key-decisions` in frontmatter — summarized:

1. **Task 1 checkpoint: option-b**, verbatim, recorded as required by the plan's acceptance criteria — deviates deliberately from 14-UI-SPEC.md § G1 point 1's literal `aria-label` prescription, for accessible-name preservation (G1's own stated rationale for `aria-label` does not hold against the current markup).
2. **`chipDescId` via `$props.id()`**, not `source.name` — a correctness fix over the plan's literal `<action>` text (which specified "a stable id derived from `source.name`"), because this component's own file-header comment documents that it renders more than one simultaneous instance per source. A name-derived id would have produced duplicate DOM ids across the visible row, the `aria-hidden` measurement clone, and the overflow-popover clone.
3. **Class 1 assertions repointed to `toHaveAccessibleDescription`** rather than manual `aria-describedby` → element-id → `textContent` resolution — simpler, retrying, and matches how assistive tech actually computes the description.
4. **Class 2 locators required no change**, confirmed by running the full suite rather than trusting the plan's own file-list grep alone.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 — Bug] `chipDescId` derived from `$props.id()`, not `source.name`**
- **Found during:** Task 2 (writing the sr-only description span)
- **Issue:** The plan's `<action>` literally specified "a stable id derived from `source.name`". `SourceChip.svelte`'s own file-header comment documents that it is reused unforked in three simultaneous DOM locations per source (visible chip row, `WebspaceHeader.svelte`'s `aria-hidden` measurement clone, and the overflow-popover clone) — a name-derived id would produce duplicate DOM ids across those instances, which is invalid HTML and an unreliable `aria-describedby` target.
- **Fix:** Used Svelte 5's `$props.id()` instead — SSR-safe, unique per component instance, and exactly the primitive Svelte ships for this case.
- **Files modified:** `web/src/lib/components/SourceChip.svelte`
- **Verification:** `npm --prefix web run test -- source-chip-tooltip` (20/20 pass), `npm --prefix web run check` (0 errors), `make e2e` (139/139 pass).
- **Committed in:** `95ee38a` (Task 2 commit)

**2. [Rule 1 — Bug, scope-adjacent] Repointed 3 pre-existing test sites the plan did not declare, all directly broken by Task 2's `title` removal**
- **Found during:** Task 2 verification (`source-chip-pill.test.ts`, not in the plan's declared `files_modified`) and Task 3's full-suite run (`09-1-header-touch.spec.ts` test 5 and two multi-line `toHaveAttribute('title', …)` calls in `11-binary-changed-repin.spec.ts` / `13-manifest-unverified.spec.ts` / `13-shadowed-advisory.spec.ts` that an earlier single-line grep had missed).
- **Issue:** `source-chip-pill.test.ts` and `09-1-header-touch.spec.ts` test 5 both encode 09.1-04-PLAN.md R2's requirement that the filter button carry a native `title` as the long-press touch degrade for chip health detail (RESEARCH Pitfall 2: "chip health detail is otherwise unreachable without hover" below 768px). 14-UI-SPEC.md G1 (which this plan implements) removes that `title` under **either** Task 1 option — this was never discussed in G1 or Task 1's checkpoint, and both options break this test identically, so it was not an option-a/option-b distinguishing factor the checkpoint could have surfaced.
- **Fix:** Updated both tests to assert the new, approved (G1) behaviour, with an explicit comment on each documenting this as a **known, deliberate regression, not an oversight**: a touch user on a source chip below 768px who is not running a screen reader can no longer reach the health sentence at all — neither `aria-label` (option-a) nor `aria-describedby` (option-b) is exposed to a plain touch tap, only to assistive technology. The remaining 3 sites (in `11-binary-changed-repin.spec.ts`, `13-manifest-unverified.spec.ts`, `13-shadowed-advisory.spec.ts`) were genuine Class 1 sites the plan's own file list already declared in scope, just written as multi-line `toHaveAttribute('title', …)` calls that a naive single-line grep missed on the first pass — repointed identically to the other Class 1 sites.
- **Files modified:** `web/src/lib/components/source-chip-pill.test.ts`, `web/e2e/specs/09-1-header-touch.spec.ts`, `web/e2e/specs/11-binary-changed-repin.spec.ts`, `web/e2e/specs/13-manifest-unverified.spec.ts`, `web/e2e/specs/13-shadowed-advisory.spec.ts`
- **Verification:** `npm --prefix web run test` (1084/1084 pass), `make e2e` (139/139 pass).
- **Committed in:** `95ee38a` (source-chip-pill.test.ts, part of Task 2) and `9b03e7b` (the four spec files, part of Task 3)

---

**Total deviations:** 2 auto-fixed (1 Rule 1 correctness fix to the plan's own literal instruction, 1 Rule 1 scope-adjacent fix directly caused by Task 2's change).
**Impact on plan:** Both necessary for the plan's own hard acceptance criteria (whole suite green) to be met at all. No scope creep beyond what Task 2's change directly broke.

## Known Regression (flagged for human review — see coverage D4)

**A touch user on a source chip below 768px who is not running a screen reader can no longer reach the chip's health detail sentence at all.** Before this plan, the outer filter button's native `title` attribute served as a long-press-accessible touch degrade (09.1-04-PLAN.md R2, RESEARCH Pitfall 2 — added specifically because chip health detail is otherwise unreachable without hover, and a touchscreen has neither hover nor keyboard focus). 14-UI-SPEC.md G1 (the approved design contract this plan implements) removes that `title` in favour of an aria attribute (`aria-label` under option-a, `aria-describedby` under option-b) — but an aria attribute is exposed only through the accessibility tree, to assistive technology, never to a plain touch tap. Neither Task 1 option restores an equivalent touch-only affordance, and G1 itself never discusses touch reachability, so this was not a distinguishing factor the checkpoint could have surfaced.

This is real, deliberate, and openly regressed (not silently dropped) — both `source-chip-pill.test.ts` and `09-1-header-touch.spec.ts` test 5 now assert and document the new behaviour explicitly. A human should confirm this tradeoff is acceptable, or route a follow-up (e.g., a tap-to-reveal affordance, or a `Popover` fallback on touch) as a new todo.

**WINDOWS.md ledger:** population was attempted via `gsd-tools windows append` but the installed `gsd-tools`/`gsd-sdk` binary in this environment does not implement a `windows` subcommand (`Error: Unknown command: windows`) — recording here and via this section instead, since ledger population is documented as best-effort and non-blocking. Recommend a follow-up `gsd-tools windows append --kind deviation --phase 14 --file web/src/lib/components/SourceChip.svelte --description "..."` (see this section's text) once a working `gsd-tools` build is available, or manual entry by the orchestrator.

## Issues Encountered

- `web/node_modules` was not present in this worktree at session start; ran `npm install` before any vitest/svelte-check/Playwright command could run.
- An early single-line `grep -rn "toHaveAttribute('title'" web/e2e/specs/` missed 3 multi-line `toHaveAttribute('title', <newline> …)` call sites and 2 tests using `getAttribute('title')` directly — caught by running the actual full `make e2e` suite (139 tests) rather than trusting the grep-only check, surfacing 4 real failures the first pass missed. All 4 are documented above.

## Next Phase Readiness

- `SourceChip.svelte` renders exactly one popover on hover; the health sentence (including the untrusted-external-plugin disclosure) remains reachable to assistive technology via `aria-describedby`; the whole Playwright suite (139/139) and vitest suite (1084/1084) are green.
- Plan 14-04 (live UAT) should include a manual spot check that hovering a chip shows exactly one popover, per this plan's own `<verification>` requirement.
- The known touch-reachability regression (see above) should be surfaced to the operator before/during 14-04's UAT — it is not blocking (G1 is an approved contract and both Task 1 options share the regression identically), but it is a real capability loss worth an explicit "accept" or a follow-up todo.

---
*Phase: 14-google-drive-source-built-out-of-repo*
*Completed: 2026-08-15*
