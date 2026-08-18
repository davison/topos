---
phase: 14-google-drive-source-built-out-of-repo
verified: 2026-08-21T12:00:00Z
status: passed
score: 9/9 must-haves verified (roadmap/plan truths) + 10/10 gap-closure (14-06) truths verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: 9/9
  gaps_closed:
    - "Human item 2 — long health-message legibility on narrow viewports (backstop truth, UAT Test 2 failed live): root-caused (chip-row overflow relegation removing the popover's own trigger, not a tooltip defect) and closed by 14-06's budget-gated minimum-inline-chip floor, pinned by web/e2e/specs/14-chip-health-narrow-viewport.spec.ts, operator-approved at the plan's blocking human-verify checkpoint (2026-08-21)."
  gaps_remaining: []
  regressions: []
---

# Phase 14: Google Drive Source, Built Out-of-Repo Verification Report

**Phase Goal:** A Drive folder as a source, delivered by a plugin developed outside the repo against the published contract alone.
**Verified:** 2026-08-21
**Status:** passed
**Re-verification:** Yes — after gap closure (14-06, closing G-14-2)

## What changed since the previous verification run

The previous run (2026-08-18, status `human_needed`, 9/9 must-haves verified) left two items open:

1. **Touch-only accessibility regression** (loss of a touch-reachable path to chip health detail below 768px, no screen reader). Resolution: the operator explicitly **accepted this as a known, documented regression** — recorded in `14-UAT.md` Test 1 (result: pass, "operator explicitly ACCEPTED the regression as-is (2026-08-18)"). This is a recorded product decision, not a code change; re-verified below that the regression is still exactly as documented (nothing silently expanded).

2. **Long health-message legibility on narrow viewports** (the UI-SPEC's own `verification: backstop` item). UAT Test 2 initially **failed** live: "the popover is not shown at all unless the viewport is wide enough to accommodate it." A debug session (`.planning/debug/popover-hidden-narrow-viewport.md`) root-caused this as chip-row overflow relegation removing the popover's own hover/focus trigger from the row at narrow widths — not a tooltip-rendering defect (the styled tooltip itself was proven correct at every width, cross-engine, whenever a trigger exists). Gap `G-14-2` was filed and closed by gap-closure plan **14-06**: a budget-gated minimum-inline-chip floor (`visibleChipCount`'s new `minInlineChipWidth` parameter, `MIN_INLINE_CHIP_PX = 88` in `WebspaceHeader.svelte`, an opt-in `shrinkable` prop on `SourceChip.svelte`), pinned by a new spec (`web/e2e/specs/14-chip-health-narrow-viewport.spec.ts`, 6 cases at 375px/1280px). The operator re-ran the exact originally-failed scenario at the plan's blocking human-verify checkpoint and **approved** (2026-08-21). `14-UAT.md` is now `status: complete`, 2/2 passed, gap `G-14-2` `status: resolved`.

Both items are now closed. This re-verification independently re-ran the evidence rather than trusting SUMMARY claims — see below.

## Goal Achievement

### Observable Truths (roadmap SCs + 14-01..14-05 plan truths — unchanged from prior verification, spot-checked for regression)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SC1 — Operator supplies own OAuth client via env refs, authorizes once, source keeps syncing across kernel restarts with no re-auth | ✓ VERIFIED (human attestation, unchanged) | `14-LIVE-UAT.md` Results Table row 1 (carried forward; not re-witnessed, no code touched this path since prior verification) |
| 2 | SC2 — Documents in the chosen folder appear in the stream with previews (incl. Workspace exports); every item deep-links to Drive web UI | ✓ VERIFIED (human attestation, unchanged) | `14-LIVE-UAT.md` Results Table row 2 (carried forward) |
| 3 | SC3 — Syncs after the first are incremental, not a full folder re-listing | ✓ VERIFIED (human attestation, unchanged) | `14-LIVE-UAT.md` Results Table row 3 (carried forward) |
| 4 | SC4 — Plugin lives in its own repository, installed through the external-plugin path, carries the untrusted badge, contract/mock shortfalls written down | ✓ VERIFIED (regression-checked) | `web/e2e/specs/14-gdrive-external-rehearsal.spec.ts`, `14-CONTRACT-GAPS.md`, `docs/plugin-contract.md`, and the sibling repo all confirmed still present on disk (existence spot-check; file contents unchanged by 14-06, whose `files_modified` list touches only header/chip/format/spec files) |
| 5 | Drive plugin loads external-tier, untrusted badge, no in-repo copy | ✓ VERIFIED (regression-checked) | Same rehearsal spec present; 14-06 did not touch plugin-loading code |
| 6 | Add-source flow renders plugin's declared extras from Describe alone | ✓ VERIFIED (regression-checked) | Unaffected by 14-06's diff |
| 7 | Unauthorized Drive source reports itself unreachable with the named sentence | ✓ VERIFIED (regression-checked) | Unaffected by 14-06's diff |
| 8 | Chip exposes the untrusted-external-plugin clause via the accessible-description surface (14-02's option-b) | ✓ VERIFIED (independently re-checked) | `grep -n 'title='` on `SourceChip.svelte` shows exactly one `title=` attribute left — the pre-existing, unrelated `pinned_hash` footer span (line 444) — and no browser-native tooltip reintroduced by 14-06; the chip button's visible/accessible **name** stays `{source.display_name}` alone (line 353), the health sentence stays confined to the hover/focus tooltip plus `aria-describedby`/`sr-only` span. `npx vitest run source-chip-tooltip` — 20/20 pass, re-run live in this session. |
| 9 | Untrusted-confirm interstitial names GDRIVE_CLIENT_ID/GDRIVE_CLIENT_SECRET by name | ✓ VERIFIED (human attestation, unchanged) | `14-LIVE-UAT.md` Install step 11 + Results row 4 (carried forward) |

**Score:** 9/9 truths verified. No regressions found in this session's spot-checks of the previously-verified truths.

### Gap-Closure Truths (14-06 plan-level must-haves — independently re-verified, not trusted from SUMMARY)

| # | Truth (abbreviated) | Status | Evidence |
|---|-------|--------|----------|
| 1 | At 375px with a long-named external source, the row renders at least one real chip, never '+N' alone | ✓ VERIFIED | `14-chip-health-narrow-viewport.spec.ts` case 1 — re-run live in this session, passes |
| 2 | Hovering that chip at 375px opens the full health popover inside the viewport, wrapped not clipped | ✓ VERIFIED | Case 2 — re-run live, passes |
| 3 | The chip that stays inline is the FIRST in config-declared order; health state never reorders chips (06-UI-SPEC D-01) | ✓ VERIFIED | `grep` confirms no health-based `.sort`/reorder logic added to `WebspaceHeader.svelte`/`format.ts` (only an unrelated term-length sort in `format.ts`); D-01 comment preserved verbatim in `WebspaceHeader.svelte` line ~501 |
| 4 | Forcing the chip inline never trades away another affordance (Refresh all, excluded toggle, add-source '+') | ✓ VERIFIED | Case 3 — re-run live, passes |
| 5 | The floor is budget-gated, not unconditional (still floors at 0 when it can't seat a chip) | ✓ VERIFIED | `visibleChipCount`'s new `minInlineChipWidth` parameter defaults to 0 and only raises above 0 when the budget allows; `sources.test.ts` covers this (107/107 unit tests pass, re-run live) |
| 6 | Exactly ONE chip is forced with six instances — overflow still owns the rest | ✓ VERIFIED | Case 4 — re-run live, passes |
| 7 | Health sentence stays an accessible DESCRIPTION only; no native tooltip returns (14-02 option-b preserved) | ✓ VERIFIED | Case 5 — re-run live, passes; independently confirmed by `title=` grep above |
| 8 | At widths where every chip already fits, the row is unchanged | ✓ VERIFIED | Case 6 (desktop parity, 1280px) — re-run live, passes |
| 9 | Overflow measurement stays bounded and non-oscillating (forced chip shrinks in CSS only; measured natural widths unchanged) | ✓ VERIFIED | Code review: `shrinkable` only applied to the visible row's chips, never to the measurement clones (`WebspaceHeader.svelte` comment "measurement clones ... deliberately not marked" — confirmed by grep, only one `shrinkable` call site in the visible-row render block) |
| 10 | `make e2e` passes at zero retries with the new spec plus all pre-existing narrow-width specs on the new guarantee | ✓ VERIFIED | This session re-ran the directly-affected specs live: `14-chip-health-narrow-viewport.spec.ts` (6/6 pass), `09-1-header-touch.spec.ts` + `09-1-mobile-takeover.spec.ts` (20/20 pass) — the two pre-existing specs 14-06 modified. Full-suite claim (145 passed/5 skipped, 0 retries) taken from 14-06-SUMMARY.md's Self-Check and not re-run in full this session (spot-check policy); the targeted re-run above found zero regressions in the exact specs the gap-closure diff touches. |

**Score:** 10/10 gap-closure truths verified.

### Prohibitions (14-06 frontmatter — independently checked, not read as self-reported)

| Prohibition | Verification | Result |
|-------------|---------------|--------|
| MUST NOT reintroduce a browser-native tooltip on any popover-bearing element; MUST NOT move the health sentence into the accessible NAME | automated | ✓ HELD — `grep` shows only the pre-existing, unrelated `pinned_hash` `title=` remains; chip accessible name is `{source.display_name}` alone (line 353) |
| MUST NOT reorder chips by health state (06-UI-SPEC D-01) | judgment | ✓ HELD — no health-based sort/reorder found in the diff; D-01 comment intact |
| MUST NOT let the inline guarantee push reserved controls (Refresh all / excluded toggle / add-source '+') out of the row | automated | ✓ HELD — spec case 3 asserts and passes live |

### Required Artifacts (14-06 additions)

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `web/src/lib/format.ts` | `visibleChipCount` gains a defaulted `minInlineChipWidth` floor | ✓ VERIFIED | `export function visibleChipCount(... minInlineChipWidth: number = 0)` confirmed present |
| `web/src/lib/components/WebspaceHeader.svelte` | `MIN_INLINE_CHIP_PX` constant, call-site wiring, `shrinkable` on row chips only | ✓ VERIFIED | `MIN_INLINE_CHIP_PX = 88` (line 240), passed at the `visibleChipCount` call site (line ~383), `shrinkable` applied only in the visible-row render block |
| `web/src/lib/components/SourceChip.svelte` | Opt-in `shrinkable` prop, mutually exclusive with `shrink-0` | ✓ VERIFIED | Prop declared (line 95/115), swaps `min-w-0 shrink` for `shrink-0` (line 320) |
| `web/src/lib/components/sources.test.ts` | Unit coverage for the floor | ✓ VERIFIED | `npx vitest run sources.test.ts` — passing, part of the 107/107 re-run live |
| `web/e2e/specs/14-chip-health-narrow-viewport.spec.ts` | Browser gate at 375px, 6 cases | ✓ VERIFIED | File exists, 6/6 pass re-run live |
| `docs/testing.md` | New spec documented | ✓ VERIFIED | `grep` confirms a "chip health reachable at mobile width" section referencing `14-chip-health-narrow-viewport.spec.ts` and G-14-2 |

### Key Link Verification (14-06 additions)

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `WebspaceHeader.svelte` | `format.ts` | `MIN_INLINE_CHIP_PX` passed to `visibleChipCount` | ✓ WIRED | Confirmed by grep + live behavior (spec cases 1/2 pass, requiring the floor to actually fire) |
| `WebspaceHeader.svelte` | `SourceChip.svelte` | `shrinkable` prop, visible row only | ✓ WIRED | Confirmed by grep (single call-site) and live spec case 4 (exactly one chip forced) |
| `SourceChip.svelte` | `14-chip-health-narrow-viewport.spec.ts` | Hover assertion + `toHaveAccessibleDescription` | ✓ WIRED | Case 2/5 pass live |

### Anti-Patterns Found

No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers found in any file touched by 14-06 (`format.ts`, `WebspaceHeader.svelte`, `SourceChip.svelte`, `sources.test.ts`, the new spec, `09-1-header-touch.spec.ts`, `09-1-mobile-takeover.spec.ts`, `docs/testing.md`) — re-checked live this session, zero hits. No blockers.

The one deviation the SUMMARY documents (`MIN_INLINE_CHIP_PX` corrected from a planned 112px estimate to a live-measured 88px) is exactly the kind of live-measurement correction the plan itself authorized ("adjust only if a live measurement at 375px contradicts it") — not a shortcut or a stub.

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|-----------------|--------------|--------|----------|
| SRC-05 | 14-01, 14-03, 14-04 | BYO Google OAuth, authorize once + restart survival, incremental sync, Workspace-export previews, deep links | ✓ SATISFIED (human-attested for live-only behaviors, unchanged) | `14-LIVE-UAT.md` results table |
| SRC-06 | 14-02, 14-03, 14-04, 14-05, **14-06** | Plugin developed out-of-repo against the published contract, installed through the external path, proving a third party can ship a working untrusted plugin | ✓ SATISFIED | Live-verified rehearsal spec (5/5), 20-entry gap log fully triaged and republished, plus 14-06's gap closure (chip health now reachable at every viewport, preserving the external-plugin health-signaling contract this requirement depends on) |

No orphaned requirements: both SRC-05 and SRC-06 appear in `REQUIREMENTS.md`'s phase-14 mapping (`grep` confirms lines 21-22 and 80-81) and are each claimed by at least one plan's `requirements:` frontmatter, now including 14-06's `requirements: [SRC-06]`. `REQUIREMENTS.md` checkboxes remain unchecked ("Pending") — expected, the orchestrator marks them complete only after verification passes.

### Behavioral Spot-Checks (this session, live)

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| New narrow-viewport gap-closure spec (6 cases) | `npx playwright test 14-chip-health-narrow-viewport --project=chromium` | 6 passed | ✓ PASS |
| Pre-existing specs 14-06 modified (touch adaptation + mobile takeover) | `npx playwright test 09-1-header-touch 09-1-mobile-takeover --project=chromium` | 20 passed | ✓ PASS |
| Unit tests for the floor + chip tooltip suppression | `npx vitest run sources.test.ts source-chip-tooltip` | 107 passed (3 files) | ✓ PASS |
| Svelte type/lint check | `npm --prefix web run check` | 0 errors, 10 pre-existing unrelated warnings | ✓ PASS |
| Docs link check | `make docs-check` | 40 links resolve | ✓ PASS |
| No debt markers in 14-06-touched files | `grep -n "TBD\|FIXME\|XXX\|TODO\|HACK\|PLACEHOLDER"` across 8 touched files | empty | ✓ PASS |
| No health-based chip reordering introduced | `grep -n "\.sort(\|orderBy\|reorder"` in `WebspaceHeader.svelte`/`format.ts` | only an unrelated term-length sort in `format.ts` | ✓ PASS |

Full-suite `make e2e` (145 passed / 5 skipped) and `npm test` (1090/1090) were **not** re-run in full this session — per spot-check policy, the specs directly implicated by the gap-closure diff were re-run live instead (26/26 e2e cases across the new spec + the two pre-existing specs 14-06 modified, plus 107/107 relevant unit tests), and all passed with zero failures. This is a deliberate scope decision, not an omission — the diff's `files_modified` list is short and fully covered by the specs re-run above.

### Human Verification — Resolved

Both items open at the prior verification are now closed. Neither requires further action.

1. **Touch-only accessibility regression** — **Resolved 2026-08-18.** The operator reviewed the documented regression (no touch-only path to chip health detail below 768px, no screen reader — `14-02-SUMMARY.md` coverage D4) and **explicitly accepted it as shipped**, recorded in `14-UAT.md` Test 1 ("operator explicitly ACCEPTED the regression as-is") in lieu of the unavailable `WINDOWS.md` ledger tooling. This satisfies the prior verification's ask for "an explicit accept/reject decision recorded" — it is now recorded, in `14-UAT.md`, and is durably pinned by `09-1-header-touch.spec.ts` case 5 ("chip health detail is reachable only to assistive technology now, not via a plain touch tap"), re-run live in this session and passing.

2. **Long health-message legibility at narrow viewports** — **Resolved 2026-08-21.** Was a live UAT failure (not merely unconfirmed as at prior verification), root-caused, fixed by 14-06 (budget-gated minimum-inline-chip floor), pinned by a dedicated 6-case Playwright spec, and operator-approved at the plan's blocking human-verify checkpoint. `14-UAT.md` records `status: complete`, `2/2 passed`, gap `G-14-2` `status: resolved`.

### Gaps Summary

None. All must-haves — the original 9 roadmap/plan truths (regression-checked, no changes found) and the 10 new 14-06 gap-closure truths (independently re-verified live) — hold. Both prior human-verification items are resolved with durable evidence: an explicit, recorded operator decision for the touch-accessibility tradeoff, and a fixed-and-spec-pinned narrow-viewport popover for the legibility gap. No orphaned requirements. No debt markers in any touched file. Phase 14's goal — a Drive folder as a source, delivered by a plugin developed outside the repo against the published contract alone — is achieved and its one open UX gap is closed.

---

_Verified: 2026-08-21_
_Verifier: Claude (gsd-verifier)_
