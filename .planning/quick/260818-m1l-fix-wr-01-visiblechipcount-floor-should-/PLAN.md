---
quick_id: 260818-m1l
type: quick
autonomous: true
files_modified:
  - web/src/lib/format.ts
  - web/src/lib/components/sources.test.ts
---

<objective>
Fix WR-01 from 14-REVIEW-GAPS.md: `visibleChipCount`'s G-14-2 floor
unconditionally reserves the overflow trigger's width when testing whether
a chip can be forced inline — but when the forced chip is the ONLY
participating chip, forcing it leaves nothing to relegate and no trigger
ever renders. The floor therefore under-estimates the true available room
by `overflowTriggerWidth + gapWidth` (~81px with live constants) and can
decline to fire for a single-source webspace whose budget lands in that
band — reproducing the exact G-14-2 symptom the floor exists to close.
</objective>

<tasks>

<task type="auto">
  <name>Task 1: Single-chip forced-budget special case + regression test</name>
  <files>web/src/lib/format.ts, web/src/lib/components/sources.test.ts</files>
  <action>
    In `visibleChipCount`, when the floor is considered and
    `chipWidths.length === 1`, test the minimum against the true
    single-chip ceiling (`budget - gapWidth` — the plain budget minus the
    one trailing gap, since no trigger will render) instead of the
    multi-chip `overflowBudget`. Correct the docstring sentence that
    states the false premise ("a forced chip coexists with the overflow
    trigger in every case where anything was relegated"). Add the review's
    boundary-pinning regression case to the existing describe block.
  </action>
  <verify>
    <automated>npm --prefix web run test; npm --prefix web run check; make e2e E2E_ARGS="e2e/specs/14-chip-health-narrow-viewport.spec.ts"; make e2e</automated>
  </verify>
  <acceptance_criteria>
    - visibleChipCount([230], 375, 150, 40, 8, 200) === 1 (was 0): the sole chip is forced when the single-chip ceiling clears the minimum even though the multi-chip formula would not.
    - All pre-existing unit cases (incl. the six 14-06 floor cases) pass unmodified.
    - Full e2e suite green at zero retries.
  </acceptance_criteria>
</task>

</tasks>
