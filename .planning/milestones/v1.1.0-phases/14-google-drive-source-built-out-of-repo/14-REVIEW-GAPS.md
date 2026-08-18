---
phase: 14-google-drive-source-built-out-of-repo
reviewed: 2026-08-18T14:16:01Z
depth: standard
files_reviewed: 7
files_reviewed_list:
  - web/src/lib/format.ts
  - web/src/lib/components/WebspaceHeader.svelte
  - web/src/lib/components/SourceChip.svelte
  - web/src/lib/components/sources.test.ts
  - web/e2e/specs/14-chip-health-narrow-viewport.spec.ts
  - web/e2e/specs/09-1-header-touch.spec.ts
  - web/e2e/specs/09-1-mobile-takeover.spec.ts
findings:
  critical: 0
  warning: 1
  info: 0
  total: 1
status: issues_found
---

# Phase 14: Code Review Report (14-06 gap-closure diff, base 45e3305)

**Reviewed:** 2026-08-18T14:16:01Z
**Depth:** standard
**Files Reviewed:** 7
**Status:** issues_found

## Summary

Reviewed the 14-06 gap-closure diff against `45e3305`: `visibleChipCount`'s
new budget-gated minimum-inline-chip floor (`format.ts`), the
`MIN_INLINE_CHIP_PX` constant and `shrinkable` wiring in
`WebspaceHeader.svelte`, the opt-in `shrinkable` prop and mutually-exclusive
shrink classes in `SourceChip.svelte`, the six new `visibleChipCount` unit
cases, the new `14-chip-health-narrow-viewport.spec.ts`, and the
comment/locator updates in the two `09-1-*` specs.

All three locked constraints hold:
- **14-02 option-b** (health sentence stays an `aria-describedby`
  description; no native `title`; accessible name stays the display name)
  is untouched by this diff — `SourceChip.svelte`'s tooltip/description
  wiring is unmodified, and `14-chip-health-narrow-viewport.spec.ts` case 5
  and `09-1-header-touch.spec.ts` case 5 both re-assert it.
- **06-UI-SPEC D-01** (chips never reordered by health) holds — the floor
  only ever changes *how many* leading, config-ordered chips are sliced
  into `visibleSources`; it never reorders `participatingSources`.
- **The floor only ever raises a zero, and is inert at the default
  parameter value** — verified by code inspection and by re-running the
  arithmetic on all 6 new unit cases (all pass; `npx vitest run
  src/lib/components/sources.test.ts` → 63/63 green) and by `svelte-check`
  (0 errors across the repo, no new warnings in the reviewed files).

One real, provable behavioral gap remains in the floor's own budget
accounting for the specific case the phase's headline scenario actually
exercises (a webspace with exactly one participating source) — see WR-01.
It doesn't violate any of the three locked constraints above, and the
committed tests happen not to land on the numbers that expose it, but it
means G-14-2 ("a long-named chip is relegated wholesale into the '+N'
pill") can still reproduce for a lone-source webspace at some narrow
widths, which is precisely the class of regression this phase exists to
close.

## Warnings

### WR-01: The floor's overflow-budget accounting over-reserves space for a trigger that will never render when the forced chip is the *only* participating source, so the floor can still fail to fire for exactly the single-source scenario 14-06 targets

**File:** `web/src/lib/format.ts:398-433` (`visibleChipCount`), consumed by `web/src/lib/components/WebspaceHeader.svelte:376-385`

**Issue:**

When `visibleChipCount` falls into the overflow branch, it always computes
`overflowBudget = budget - overflowTriggerWidth - gapWidth * 2` before
testing the G-14-2 floor — i.e. it always reserves space for the "+N"
overflow trigger plus its two flanking gaps, on the theory (stated in the
function's own doc comment) that "a forced chip coexists with the overflow
trigger in every case where anything was relegated."

That premise is false in exactly one case: when `chipWidths.length === 1`
(a webspace with a single participating source, `LONG_WEBSPACE` in the new
e2e spec — `mockInstances`/`attachedWebspace(LONG_WEBSPACE, [LONG_ID], …)`).
If the floor fires there, `count` becomes `1 === chipWidths.length`, so
`hiddenSources` is empty and `hasOverflow` is `false` — **no overflow
trigger ever renders** in that case. The true ceiling for the forced chip
is therefore `budget - gapWidth` (the row's available width minus the one
trailing gap before the reserved trailing group), not
`budget - overflowTriggerWidth - gapWidth * 2`. The current code
under-estimates the truly available room by
`overflowTriggerWidth + gapWidth` (≈ 73 + 8 = 81px using the values the
`MIN_INLINE_CHIP_PX` doc comment itself measured live), purely because it
reuses the multi-chip overflow-budget formula unconditionally.

Concretely, with production's real `MIN_INLINE_CHIP_PX = 88`,
`overflowTriggerWidth ≈ 73`, `gapWidth = 8` (values taken directly from the
`WebspaceHeader.svelte` doc comment's own live probe), any single-source
webspace whose `budget` (`availableWidth - reservedWidth`) lands in
`[96, 177)` reproduces the *exact* bug this phase is closing: the sole
chip's natural width doesn't fit, the floor's conservative
`overflowBudget` (`budget - 89`) is below 88 so it declines to fire, and
the chip is relegated wholesale into a "+1" popover — with nothing left in
the row to hover — even though the *true* available room
(`budget - gapWidth`, up to 169px in that range) is comfortably above the
88px floor and the chip would have rendered fine, shrunk, with no trigger
needed at all.

This is a narrow window (exactly one participating source, and a specific
40–80px width band), which is why none of the six new unit tests in
`sources.test.ts` land on it — test 1
(`visibleChipCount([230], 375, 150, 40, 8, 112)`) is a single-chip case,
but its `overflowBudget` (169) already clears `minInlineChipWidth` (112),
so it doesn't probe the boundary where the discrepancy actually flips the
answer. It's also why the new `14-chip-health-narrow-viewport.spec.ts`
(which specifically exercises `LONG_WEBSPACE`, a single-source webspace)
doesn't catch it: the live-probed numbers happen to leave `overflowBudget`
at 99px, comfortably above the 88px floor, so the extra 81px of
unnecessarily-reserved slack never mattered for that particular fixture.
A webspace with a merely-somewhat-narrower row (fewer trailing controls,
or a slightly longer display name) would land inside the failing band.

**Fix:** Special-case the single-remaining-chip situation so the floor is
tested against the budget that will actually apply once nothing is
relegated:

```ts
// One more gap for the trigger itself, plus the gap before it.
const overflowBudget = budget - overflowTriggerWidth - gapWidth * 2;
let used = 0;
let count = 0;
for (const width of chipWidths) {
	const candidateGap = count > 0 ? gapWidth : 0;
	if (used + candidateGap + width > overflowBudget) break;
	used += candidateGap + width;
	count += 1;
}
if (count === 0 && minInlineChipWidth > 0 && chipWidths.length > 0) {
	// Forcing the ONLY candidate chip leaves nothing to relegate, so no
	// overflow trigger will ever render for it — the true ceiling is the
	// plain budget minus one trailing gap, not the multi-chip
	// overflow-budget formula (which needlessly reserves the trigger's
	// width + an extra gap that will never be spent).
	const forcedBudget = chipWidths.length === 1 ? budget - gapWidth : overflowBudget;
	if (forcedBudget >= minInlineChipWidth) return 1;
}
return count;
```

and add a regression case to `sources.test.ts` that pins the boundary,
e.g. (using the production constants directly so it can't silently drift):

```ts
it('forces the sole participating chip inline even when the multi-chip overflow-budget formula alone would not have room for it (single-source WR-01 regression)', () => {
	// budget = 375-150=225; overflowBudget = 225-40-16=169 (< 200, the
	// floor as currently written declines); true single-chip ceiling =
	// budget-gapWidth = 217 (>= 200) — the chip should still be forced.
	expect(visibleChipCount([230], 375, 150, 40, 8, 200)).toBe(1);
});
```

---

_Reviewed: 2026-08-18T14:16:01Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
