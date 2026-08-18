---
quick_id: 260818-m1l
status: complete
date: 2026-08-18
commits: [af172e5]
---

# Quick Task Summary: fix WR-01 (single-source floor over-reservation)

Fixed `14-REVIEW-GAPS.md` WR-01: `visibleChipCount`'s G-14-2 floor
unconditionally charged the "+N" overflow trigger's width (~81px with
live constants) when deciding whether to force a chip inline — but a
single-source webspace whose only chip is forced renders no trigger at
all, so the floor could decline inside a real width band and reproduce
the exact G-14-2 symptom (chip relegated into a "+1" pill, nothing to
hover) for lone-source webspaces.

## Changes

- `web/src/lib/format.ts` — the floor now tests a single candidate chip
  against the trigger-free ceiling (`budget - gapWidth`); two or more
  candidates keep the overflow-budget gate unchanged. The docstring's
  false premise ("a forced chip coexists with the overflow trigger in
  every case") corrected to state the two-case rule.
- `web/src/lib/components/sources.test.ts` — two boundary-pinning cases:
  the review's regression (sole chip forced where the multi-chip formula
  alone lacked room) and the negative guard (still declines when even the
  trigger-free ceiling cannot seat the minimum).

## Verification

- `npm --prefix web run test`: 1092/1092 (58 files) — all 14-06 floor
  cases and the eleven original cases unmodified and green
- `npm --prefix web run check`: 0 errors
- `make e2e`: 145 passed, 5 skipped, zero retries (includes the six
  14-chip-health-narrow-viewport cases)
