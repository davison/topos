---
phase: 12-filesystem-source
reviewed: 2026-08-14T11:07:43Z
depth: standard
files_reviewed: 6
files_reviewed_list:
  - kernel/httpapi/sources.go
  - kernel/httpapi/sources_test.go
  - web/e2e/specs/12-tooltip-precedence.spec.ts
  - web/src/lib/components/SourceChip.svelte
  - web/src/lib/components/match-advisory.test.ts
  - web/src/lib/format.ts
findings:
  critical: 0
  warning: 0
  info: 1
  total: 1
status: clean
---

# Phase 12: Code Review Report (Gap-Closure Delta — Plan 12-11)

**Reviewed:** 2026-08-14T11:07:43Z
**Depth:** standard
**Files Reviewed:** 6
**Status:** clean

## Summary

This is the delta review for plan 12-11, which was executed specifically to close two findings from the prior phase-wide review (dated 2026-08-14T09:58Z, 21 files): **CR-01** (SourceChip tooltip advisory gate ignored `reachable`) and **WR-01** (launch-failure `last_notice` contract undocumented). This report replaces that prior review at the same path. The six files above are the entire diff since `f8fdf8d`.

I independently verified the diff against `f8fdf8d..HEAD` (not just read the final files) to confirm both gap closures were genuine rather than cosmetic, then ran every test suite this delta touches or added:

- `go test ./kernel/httpapi/... -run 'TestSourcesHandler|TestSources_' -v` — 11/11 pass, including the new `TestSourcesHandler_LaunchFailedEntryCarriesNoLastNotice`.
- `npx vitest run src/lib/components/match-advisory.test.ts` — 36/36 pass.
- `go vet ./kernel/httpapi/...` — clean.
- `npm run check` (svelte-check) — 0 errors across 866 files; no new warnings on `SourceChip.svelte` or `format.ts`.
- `npm run check:e2e` (tsc) — the one pre-existing error reported (`e2e/specs/12-filesystem-recursion.spec.ts`, `unlinkSync` import) is in a file outside this delta's scope; `12-tooltip-precedence.spec.ts` type-checks clean.

**CR-01 — genuinely closed.** The old tooltip gate (`advisory !== '' && source.last_status !== 'error'`) never consulted `source.reachable`, so an unreachable source with a stale `last_status: "ok"` and a leftover `last_notice` rendered a reassuring "synced … — advisory" tooltip while its own dot was red. The fix adds `format.ts`'s `isAdvisoryOnly(source)` — re-invokes `healthTone` with the notice cleared and checks the result is `'success'` — which correctly folds in `reachable`, `launch_failure`, `last_status`, all through `healthTone`'s existing, already-tested precedence chain rather than a hand-rolled parallel condition. I traced the four precedence cases by hand (unreachable+notice → `destructive`/"unreachable since…", pin-mismatch+notice → the binary-changed sentence, never-synced+notice → "not yet synced", errored+notice → the last-error sentence) and confirmed each against both the new unit-test branch-selection matrix (`match-advisory.test.ts`) and the new browser-level proof (`12-tooltip-precedence.spec.ts`, Test A/B). All check out.

**WR-01 — genuinely closed, and confirmed to have been correct all along.** The `kernel/httpapi/sources.go` diff is comment-only (verified via `git diff`): the launch-failure branch of `sourceStatusesFrom` never set `LastNotice` on the synthesized entry even before this plan — the gap was that this omission was undocumented and unpinned by a test, not that the field leaked. `docs/api.md` already states "Empty for an instance that never launched" for `last_notice` (line 571-573), confirming the plan's premise. The new `TestSourcesHandler_LaunchFailedEntryCarriesNoLastNotice` closes the test gap correctly: it records a completed run with a non-empty notice for the same instance name, then proves the merge still reports `last_status`/`last_sync_unix` from that run while omitting `last_notice` — a real assertion (proving the merge saw the run) rather than a vacuous pass.

**Plan constraints, verified byte-for-byte via `git diff`:**
- `healthTone` in `format.ts` is untouched; only `isAdvisoryOnly` was added after it. Confirmed.
- `SourceChip.svelte`'s four switch-arm tooltip templates (success/warning/destructive/unknown) are byte-identical; the only functional change is the advisory branch's gate condition (`source.last_status !== 'error'` → `advisoryOnly`) plus the new import and the new `advisoryOnly` derived declaration. Confirmed.
- `kernel/httpapi/sources.go` is comment-only. Confirmed.

No new Critical or Warning findings in the changed code.

## Info

### IN-02: Whitespace-only `last_notice` produces a malformed "last error: " tooltip on a healthy sync (pre-existing, explicitly out-of-scope, unchanged by this delta)

**File:** `web/src/lib/components/SourceChip.svelte:130,225-232`, `web/src/lib/format.ts:132-146`
**Issue:** `advisory` (the tooltip's own local variable) is the **trimmed** notice; `isAdvisoryOnly`/`healthTone` test the **untrimmed** `source.last_notice`. For a hypothetical whitespace-only `last_notice` on an otherwise-healthy, reachable, non-erroring sync: `advisory === ''` (trimmed empty) so the advisory branch's gate `advisory !== '' && advisoryOnly` is false and the code falls through to `switch (tone)` — but `tone = healthTone(source)` (computed from the *untrimmed* notice) evaluates `(source.last_notice ?? '') !== ''` as true and returns `'warning'`, landing on the `case 'warning'` branch: `` `${display_name} — last error ${relative}: ${source.last_error}` ``. Since this is a healthy sync, `last_error` is empty, producing a tooltip that reads "… — last error 5 minutes ago: " with a dangling colon and no error text — self-contradictory copy for a source that has no error.
This is not introduced by plan 12-11 — the same divergence existed under the old gate (`source.last_status !== 'error'`) since `healthTone` was never touched — and the plan's own code comment (SourceChip.svelte:220-224) explicitly documents this as a known, deliberately-unchanged, out-of-scope inconsistency. Whether the kernel can ever actually emit a whitespace-only (non-empty-but-blank) `last_notice` is unclear from these six files alone (`correlate`'s notice-composition code is out of this delta's scope), so this may be purely theoretical. Flagged for visibility, not as a regression.
**Fix:** If ever revisited, either trim `source.last_notice` once at the `healthTone`/`isAdvisoryOnly` boundary so "whitespace-only" and "empty" are treated identically everywhere, or trim before storing/composing the notice in `kernel/correlate`. Not required for this delta.

## Deferred (carried forward, not re-opened)

**IN-01** (kernel/correlate `zeroMatchNotice`, cosmetic empty-value-list guard) — out of scope for this delta (no file touching `kernel/correlate` is in this review's file list). Per the plan, this was explicitly deferred and is carried forward unchanged rather than re-opened here.

---

_Reviewed: 2026-08-14T11:07:43Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
