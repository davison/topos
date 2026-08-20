---
phase: 16-provenance-based-plugin-trust
plan: 06
subsystem: security
tags: [go, provenance, trust-tier, playwright, gap-closure]

requires:
  - phase: 16-provenance-based-plugin-trust
    provides: "EvaluateTrust's dual-arm (link-time + signed) tamper-refusal gate (16-01/16-02/16-03/16-04)"
provides:
  - "EvaluateTrust's two tamper-refusal return paths (link-time arm, signed arm) both set Trust.Tier=TierTrusted, never the empty zero value"
  - "Two real-Discover()-path Go regression tests pinning LaunchFailures()[0].Tier for both refusal arms"
  - "Browser-level proof (widened 13-manifest-unverified.spec.ts) that GET /api/sources reports the trusted tier for a manifest_unverified entry and the chip renders no untrusted badge"
affects: [17-plugin-repo-split, 16-VERIFICATION, security-review]

actuals:
  tokens: 3400
  tasks: 2
  commits: 5

tech-stack:
  added: []
  patterns:
    - "Trust is a reporting-field-only struct on a refusal path: Tier reflects WHY a binary is being refused (evidence-named-it), while the returned error is the sole thing that stops the launch — never conflate the two."

key-files:
  created: []
  modified:
    - kernel/pluginhost/provenance.go
    - kernel/pluginhost/manifestgate_test.go
    - kernel/pluginhost/escalation_test.go
    - web/e2e/specs/13-manifest-unverified.spec.ts

key-decisions:
  - "Both of EvaluateTrust's tamper-refusal return paths now set Tier: TierTrusted explicitly — a refusal only ever fires when a trust arm positively named the binary, so the wire tier field must never be the empty zero value (closes 16-VERIFICATION.md gap 1 / CR-01 / WR-01)."
  - "Deviation from plan: updated one stale assertion in kernel/pluginhost/escalation_test.go (TestEscalation_ShadowingCannotInheritTrust) that asserted tier != TierTrusted on a digest-mismatched refusal — the exact defect this plan corrects. The plan listed this file as 'Explicitly NOT touched' with a zero-diff acceptance criterion, which could not hold given ResolveBinary returns EvaluateTrust's Trust.Tier unconditionally regardless of error."

patterns-established: []

requirements-completed: [TRUST-01, TRUST-04]

coverage:
  - id: D1
    description: "A genuine tamper refusal from EvaluateTrust's link-time arm reports Tier=TierTrusted on the real Discover()/launch() path"
    requirement: TRUST-01
    verification:
      - kind: unit
        ref: "kernel/pluginhost/manifestgate_test.go#TestLaunch_ManifestGate_DiscoverRecordsRefusalAndSiblingsStillLaunch"
        status: pass
    human_judgment: false
  - id: D2
    description: "A genuine tamper refusal from EvaluateTrust's signed arm reports Tier=TierTrusted on the real Discover() path (new test)"
    requirement: TRUST-01
    verification:
      - kind: unit
        ref: "kernel/pluginhost/manifestgate_test.go#TestLaunch_ManifestGate_DiscoverSignedArmRefusalReportsTrustedTier"
        status: pass
    human_judgment: false
  - id: D3
    description: "GET /api/sources reports tier='trusted' for a manifest_unverified entry, and the chip renders no untrusted trust-badge glyph, in a real browser"
    requirement: TRUST-01
    verification:
      - kind: e2e
        ref: "web/e2e/specs/13-manifest-unverified.spec.ts#destructive chip, contract-exact tooltip, no reachable probe, no re-pin action, and the refusal is named in the kernel log"
        status: pass
    human_judgment: false
  - id: D4
    description: "No launch decision changed: TRUST-04 escalation suite, pin_test.go, and tier_test.go all still refuse/pass with the same launch behavior (only one stale assertion in escalation_test.go needed updating, documented as a deviation)"
    requirement: TRUST-04
    verification:
      - kind: unit
        ref: "go test ./kernel/pluginhost/... -count=1"
        status: pass
    human_judgment: false

duration: ~40min
completed: 2026-08-20
status: complete
---

# Phase 16 Plan 06: Tamper-Refusal Wire Tier Summary

**EvaluateTrust's link-time and signed tamper-refusal paths now set `Tier: TierTrusted` explicitly, closing 16-VERIFICATION.md gap 1 (empty wire tier on `manifest_unverified` entries) with two real-Discover()-path Go tests and a widened browser spec — no launch decision changed.**

## Performance

- **Duration:** ~40min
- **Completed:** 2026-08-20T16:35:22Z
- **Tasks:** 2 completed
- **Files modified:** 4 (3 Go, 1 Playwright spec)

## Accomplishments

- `EvaluateTrust`'s two tamper-refusal return statements (link-time-arm digest mismatch, signed-arm digest mismatch) each now return `Trust{Tier: TierTrusted, ...}` instead of leaving `Tier` at its empty zero value — matching `docs/api.md`'s documented `manifest_unverified` wire contract.
- Two Go regression tests drive the REAL `Discover()` path (not a hand-built `LaunchFailure` fixture) and assert `LaunchFailures()[0].Tier == TierTrusted`: one for the link-time arm (extended an existing test), one new test for the signed arm (`TestLaunch_ManifestGate_DiscoverSignedArmRefusalReportsTrustedTier`). Both were observed failing (`expected "trusted", got ""`) against unmodified production code before the fix.
- `web/e2e/specs/13-manifest-unverified.spec.ts` widened additively: its typed `GET /api/sources` response shape now declares `tier`; new assertions prove the tampered entry's tier is `'trusted'` (with a control-entry tier assertion so the check can't pass vacuously) and that the tampered chip renders no untrusted trust-badge glyph. Confirmed RED against a temporarily reverted `provenance.go`, then GREEN again with the fix restored.
- Doc comments on `EvaluateTrust` and the `Trust` struct updated to state explicitly that a refusal carries `Tier=TierTrusted` and that Tier is a reporting field only — the returned error is what actually refuses the launch.

## Task Commits

Each task was committed atomically (Task 1 followed the plan's explicit RED/GREEN split):

1. **Task 1 (RED): assert real wire tier for both tamper-refusal arms** - `a55194b` (test)
2. **Task 1 (GREEN): set trusted tier on both tamper-refusal return paths** - `ec51328` (feat)
3. **Task 1 (deviation fix): update stale tier assertion in escalation_test.go** - `0a3e5f0` (fix)
4. **Task 2: prove the corrected tier at the wire and in the browser** - `d234eb5` (test)

**Plan metadata:** committed as part of this SUMMARY commit (worktree mode — orchestrator handles STATE.md/ROADMAP.md).

## Files Created/Modified

- `kernel/pluginhost/provenance.go` — `EvaluateTrust`'s two tamper-refusal returns now set `Tier: TierTrusted`; doc comments on `EvaluateTrust` and the `Trust` struct updated.
- `kernel/pluginhost/manifestgate_test.go` — extended `TestLaunch_ManifestGate_DiscoverRecordsRefusalAndSiblingsStillLaunch` with a `Tier` assertion; added `TestLaunch_ManifestGate_DiscoverSignedArmRefusalReportsTrustedTier`.
- `kernel/pluginhost/escalation_test.go` — updated one stale assertion in `TestEscalation_ShadowingCannotInheritTrust` (see Deviations below).
- `web/e2e/specs/13-manifest-unverified.spec.ts` — typed response shape gained `tier`; added tampered-entry tier assertion, control-entry tier assertion, and badge-glyph-absence assertion; header comment updated.

## Decisions Made

- Both tamper-refusal return paths in `EvaluateTrust` set `Tier: TierTrusted` (not left empty, not set to some third "refused" value) — a refusal by definition means a trust arm positively named this binary, so trusted-tier is the semantically correct reporting value, and it is what `docs/api.md`'s worked example already documented.
- Tier remains strictly a reporting field: the returned error (`buildErr`/`provErr`, both left untouched) is what actually gates the launch. No guard condition, branch order, or error value was changed anywhere in `EvaluateTrust`, `resolveBinaryDetailed`, or `launch`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Stale assertion in `escalation_test.go` encoded the exact defect this plan fixes**

- **Found during:** Task 1, STEP 3 (whole-package verification run)
- **Issue:** `TestEscalation_ShadowingCannotInheritTrust`'s "digest mismatch under a legitimately-named manifest entry" subtest called `ResolveBinary` directly and asserted `tier != TierTrusted` on a digest-mismatched refusal. `ResolveBinary` returns `EvaluateTrust`'s `Trust.Tier` unconditionally (regardless of error) — the exact same value Task 1's fix changes. Before this plan, that tier was the empty zero value, which incidentally satisfied `tier != TierTrusted`; the plan's own must-haves establish that this is now deliberately `TierTrusted`, so the pre-existing assertion could not pass without either this test update or silently reverting the fix for this one code path.
- **Fix:** Removed the `if tier == TierTrusted { t.Fatalf(...) }` check. Kept every other invariant in the subtest: `err != nil` (refusal), `errors.Is(err, ErrProvenanceUnverified)`, and `tier != TierExternal` (no demotion). Added a doc comment explaining why the removed check is stale under the new, intentional contract.
- **Files modified:** `kernel/pluginhost/escalation_test.go`
- **Verification:** `go test ./kernel/pluginhost/... -count=1` passes in full (previously this one subtest failed: `must never resolve "trusted" for a digest-mismatched shadow`). `go build ./...` and `make test-portable` both pass.
- **Committed in:** `0a3e5f0`

**Note on the plan's own acceptance criteria:** 16-06-PLAN.md's Task 1 acceptance criteria and its plan-level `<verification>` section both required `git diff --stat -- kernel/pluginhost/escalation_test.go kernel/pluginhost/pin_test.go kernel/pluginhost/tier_test.go` to produce no output, and listed `escalation_test.go` under "Explicitly NOT touched." That requirement could not be satisfied together with a passing full test suite once this specific coupling was discovered — `pin_test.go` and `tier_test.go` remain genuinely untouched (verified: `git diff --stat` against the plan's base commit shows zero changes to either), but `escalation_test.go` required the one-line fix above. This is recorded here rather than silently honored, per Rule 1 (auto-fix bug) — the alternative (leaving the suite red, or reverting Task 1's fix for this one shadow-collision code path) would have been worse on every axis.

---

**Total deviations:** 1 auto-fixed (1 bug fix)
**Impact on plan:** Necessary for correctness — the plan's own must-have truths (both refusal arms set `Tier` explicitly) directly implied this test needed updating; the plan's assumption that the file was untouchable was the part that didn't hold up against the actual codebase.

## Issues Encountered

- The plan's acceptance criterion `grep -c 'failures\[0\].Tier' kernel/pluginhost/manifestgate_test.go` expected `2` ("one assertion per refusal arm"); the actual count is `4`, because this file's established convention (matching `.Reason`, `.Instance`) writes each assertion as two lines — the `if` condition plus a `t.Errorf` message that also references the field — for every field, in every test in this file. The substantive requirement (assert `LaunchFailures()[0].Tier` for both refusal arms, following the file's existing style) is fully satisfied; the grep count itself was a minor miscount in the plan's acceptance criteria, not a defect in the implementation.
- `gsd-tools windows append` (broken-windows ledger) returned an unrelated pre-existing error ("Ledger entry 5 has invalid status: resolved") when attempting to record the escalation_test.go deviation. Per the executor's windows-ledger contract this is best-effort and non-blocking; the deviation is instead fully documented above and in this plan's own commit messages.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- 16-VERIFICATION.md gap 1 is closed: both tamper-refusal return paths report the documented trusted tier on the wire, pinned by two real-path Go tests plus a widened browser spec.
- `kernel/pluginhost/pin_test.go` and `kernel/pluginhost/tier_test.go` remain genuinely untouched (verified against the plan's base commit); `kernel/httpapi/` was not touched.
- No blockers for the next phase. The out-of-scope advisory findings WR-02 (`scripts/install.sh` executable bits) and IN-01 (collision-fallback log wording), noted in the plan as explicitly out of scope for this gap-closure round, remain open for `/gsd-code-review 16 --fix`.

---

*Phase: 16-provenance-based-plugin-trust*
*Completed: 2026-08-20*

## Self-Check: PASSED

- FOUND: kernel/pluginhost/provenance.go
- FOUND: kernel/pluginhost/manifestgate_test.go
- FOUND: kernel/pluginhost/escalation_test.go
- FOUND: web/e2e/specs/13-manifest-unverified.spec.ts
- FOUND commit: a55194b
- FOUND commit: ec51328
- FOUND commit: 0a3e5f0
- FOUND commit: d234eb5
