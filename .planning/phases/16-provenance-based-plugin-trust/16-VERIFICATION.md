---
phase: 16-provenance-based-plugin-trust
verified: 2026-08-20T13:00:00Z
status: gaps_found
score: 5/6 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 4/6
  gaps_closed:
    - "Gap 1: EvaluateTrust's two tamper-refusal return paths now set Tier: TierTrusted explicitly (kernel/pluginhost/provenance.go lines ~648-661 and ~663-671), confirmed by direct code read and by two passing real-Discover()-path regression tests (TestLaunch_ManifestGate_DiscoverRecordsRefusalAndSiblingsStillLaunch, TestLaunch_ManifestGate_DiscoverSignedArmRefusalReportsTrustedTier)."
    - "Gap 2: docs/plugin-contract.md's 'Trust tiers' section (lines 206-256) now states the shipped evidence-based model — both directories are pure search paths, tier decided by pluginhost.EvaluateTrust from provenance evidence, corrected collision rule, deferral link to docs/plugin-trust.md. Confirmed by direct read; no residual directory-derived-trust language found repo-wide."
  gaps_remaining: []
  regressions: []
gaps:
  - truth: "Every escalation path named in the standing security todo is closed by a committed test that fails if its gate is removed: trust cannot be granted by editing config, by dropping a file into the trusted directory, or by shadowing a trusted plugin name with a different binary (ROADMAP success criterion 4, TRUST-04) — AND the operator can see why a plugin holds the tier it holds, a verification failure names the cause in the kernel log, and a tier decision is never silently mishandled (ROADMAP success criterion 5)"
    status: failed
    reason: "NEW critical finding (16-REVIEW.md CR-01, post-gap-closure re-review), independently confirmed by direct code inspection of the current tree: resolveBinaryDetailed's two-directory collision branch (kernel/pluginhost/discover_binaries.go:496-516) gates both winner checks on `err == nil`, so when one collision candidate is a genuine tamper refusal (err != nil, Tier == TierTrusted per the 16-06 fix) and the other candidate resolves cleanly, the tamper refusal is silently discarded rather than surfaced. Two confirmed scenarios: (a) trusted-dir copy tamper-refused + external-dir copy independently evidenced -> function returns the external copy with nil error, silently dropping the trusted-dir tamper signal (only a generic 'external copy carries evidence and wins' log line, no mention of the tamper); (b) external-dir copy tamper-refused + trusted-dir copy has no evidence -> function falls through to the final catch-all return with nil error, and the emitted log line ('neither copy carries evidence') is factually wrong — the external copy WAS refused as tampered, this just never gets logged or surfaced as a launch failure. This directly contradicts the invariant docs/plugin-contract.md's own newly-corrected 'Trust tiers' section states for this exact code path (added by gap-closure plan 16-07): 'A candidate that a manifest positively names with a digest that no longer matches what's on disk is a tamper refusal — that resolves to the refusal itself and never falls back to launching the other copy instead.' The shipped code does fall back to the other copy in both directions. No test in escalation_test.go, tier_test.go, manifestgate_test.go, or the web/e2e/specs/16-*.spec.ts files exercises this exact combination (one collision candidate a genuine tamper refusal, the other a clean win) — confirmed by reading TestEscalation_ShadowingCannotInheritTrust and TestResolveBinary_CollisionResolvesToWhicheverCopyCarriesEvidence, both of which only cover 'neither side has evidence' or 'one side has evidence, the other has none', never 'one side is tamper-refused'. This is a real, reachable defect in the exact escalation-closure mechanism TRUST-04 requires and the exact operator-visibility guarantee SC5 requires, discovered by this phase's own post-gap-closure code review, not resolved by either gap-closure plan (16-06/16-07 both explicitly left discover_binaries.go untouched)."
    artifacts:
      - path: "kernel/pluginhost/discover_binaries.go"
        issue: "resolveBinaryDetailed's collision branch (lines 496-516) checks `trustedErr == nil && trustedTrust.Tier == TierTrusted` and `externalErr == nil && externalTrust.Tier == TierTrusted` before checking for a tamper refusal on either side, so a tamper refusal on the losing candidate is silently swallowed rather than winning outright and refusing the launch"
      - path: "docs/plugin-contract.md"
        issue: "Line ~240's collision paragraph (written by this same phase's own gap-closure plan 16-07) now asserts an invariant the shipped code does not satisfy: 'a tamper refusal... resolves to the refusal itself and never falls back to launching the other copy instead'"
    missing:
      - "Reorder resolveBinaryDetailed's collision branch to check for a tamper refusal (err != nil) on EITHER candidate before checking for a clean TierTrusted win on either — refusing immediately and naming the cause if found, matching 16-REVIEW.md CR-01's prescribed fix"
      - "Regression tests for both directions: (a) trusted-side tamper-refused + external-side clean TierTrusted win must refuse, not launch the external copy; (b) external-side tamper-refused + trusted-side plain TierExternal must refuse (or at minimum surface/log the tamper by name), not silently fall through to the ordinary pin-check path"
  - deferred_out_of_scope_advisories: "WR-01 (install.sh verifier-trust-bootstrapping caveat) and WR-02 (topos-provenance keygen charset validation) and IN-01 (link-time-arm Diagnostics asymmetry) remain open per 16-REVIEW.md but are advisory (warning/info severity), not phase-goal-blocking, and were explicitly scoped out of both gap-closure plans. Not included as gaps here; recommend routing to /gsd-code-review 16 --fix or a follow-up issue."
---

# Phase 16: Provenance-Based Plugin Trust Verification Report

**Phase Goal:** The kernel derives trust from verifiable artifact provenance rather than directory location, closing the standing escalation paths.
**Verified:** 2026-08-20T13:00:00Z
**Status:** gaps_found
**Re-verification:** Yes — after gap-closure plans 16-06 and 16-07

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Trust is location-independent — a first-party binary earns trusted tier wherever it lives, a dropped file earns nothing wherever it lives (SC1, TRUST-01) | ✓ VERIFIED | Unchanged since initial verification. `resolveBinaryDetailed`/`DiscoverAllTiered` both call `EvaluateTrust` per binary; `TestResolveBinary_LocationSymmetric`, `TestResolveBinary_ExternalWithSignedManifestResolvesTrustedTier`, `TestResolveBinary_TrustedWithNoEvidenceResolvesExternalTier` re-run live and pass (`go test ./kernel/pluginhost/... -run 'TestResolveBinary\|TestLaunch_ManifestGate\|TestEscalation\|Provenance\|EvaluateTrust' -count=1` -> `ok`, 3.19s, all subtests PASS). |
| 2 | A topos-plugins release binary verifies as first-party trusted on an installed instance with no link-time manifest entry (SC2, TRUST-02) | ✓ VERIFIED | Unchanged since initial verification — `16-04-TRUST02-PROOF.md`'s real, offline-verified pipeline evidence stands; no gap-closure plan touched signing/release material. |
| 3 | The unsigned-external consent-and-pin fallback is unchanged (SC3, TRUST-03) | ✓ VERIFIED | `pin_test.go` remains untouched by both gap-closure plans (confirmed: absent from 16-06 and 16-07's `files_modified`, and 16-06-SUMMARY explicitly records `pin_test.go` and `tier_test.go` as genuinely untouched against the plan's base commit). |
| 4 | Every escalation path (config edit, file drop, binary shadowing) is closed by a committed test that fails if its gate is removed (SC4, TRUST-04) | ✗ FAILED | The three ORIGINAL escalation tests (`TestEscalation_ConfigEditCannotGrantTrust`, `TestEscalation_FileDropCannotGrantTrust`, `TestEscalation_ShadowingCannotInheritTrust`) still pass, but 16-REVIEW.md's post-gap-closure re-review found CR-01: a genuine, untested collision-shadowing combination (one candidate tamper-refused, the other clean) is NOT closed — see Gaps. |
| 5 | The operator can see why a plugin holds the tier it holds — a verification failure names the cause on the source chip and in the kernel log, and never silently downgrades a plugin the operator believes is trusted (SC5) | ✗ FAILED | Gap 1 (previous verification) is closed: single-candidate tamper refusals now correctly report `Tier: TierTrusted` on the wire, confirmed by direct code read of both `EvaluateTrust` return paths (kernel/pluginhost/provenance.go:648-671) and by passing regression tests. HOWEVER, CR-01 (new finding) shows the collision path can silently drop a tamper-refusal signal entirely — no log naming the cause, no launch-failure entry — for one of the two colliding candidates. See Gaps. |
| 6 | Documentation states the trust model in one place; every other document agrees with it (16-05 must-have, D-11 intent) | ✓ VERIFIED | Gap 2 (previous verification) is closed. Direct read of `docs/plugin-contract.md:206-256` confirms: "pure search paths" language present, `pluginhost.EvaluateTrust` named as the decider, the corrected collision rule stated, deferral link to `docs/plugin-trust.md` present. Repo-wide sweep (`grep -rn 'trusted directory always wins\|derived exclusively from which directory\|resolved from here is' docs/ README.md web/README.md`) produces no output. `./scripts/check-doc-links.sh` -> "checked 59 link(s) across 22 file(s) — all resolve." Note: the corrected collision paragraph states an invariant ("never falls back to launching the other copy instead") that CR-01 shows the shipped code does not actually satisfy — the DOCUMENT itself is now internally consistent and accurate to the INTENDED design, but it describes behavior the code does not yet deliver. This is scored as the documentation truth being met (the doc states the correct design) while the underlying behavioral gap is captured separately under truth 4/5. |

**Score:** 5/6 truths verified (0 present, behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `kernel/pluginhost/provenance.go` | `EvaluateTrust`'s two tamper-refusal paths report the trusted tier on the wire | ✓ VERIFIED | Lines 648-661 (link-time arm) and 663-671 (signed arm) both now construct `Trust{Tier: TierTrusted, ...}` — confirmed by direct read, matching 16-06's must-haves exactly. |
| `kernel/pluginhost/manifestgate_test.go` | Real-path regression nets asserting `LaunchFailure.Tier` for both refusal arms | ✓ VERIFIED | `TestLaunch_ManifestGate_DiscoverRecordsRefusalAndSiblingsStillLaunch` (extended) and `TestLaunch_ManifestGate_DiscoverSignedArmRefusalReportsTrustedTier` (new) both pass on live re-run. |
| `web/e2e/specs/13-manifest-unverified.spec.ts` | Browser-level assertion that the wire tier field is correct for a refusal | ✓ VERIFIED (present) | Per 16-06-SUMMARY, widened with `tier` assertions and a badge-glyph-absence check; not independently re-run in this verification pass (browser/network-heavy — see Behavioral Spot-Checks). |
| `docs/plugin-contract.md` | Trust tiers section states the shipped evidence-based model | ✓ VERIFIED | Confirmed by direct read; see truth 6 above. |
| `kernel/pluginhost/discover_binaries.go` | Collision resolution correctly refuses on any tamper-refused candidate | ✗ DEFECT | `resolveBinaryDetailed`'s collision branch (lines 496-516) is unmodified by either gap-closure plan and contains the CR-01 defect — confirmed unfixed by direct code read. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `kernel/pluginhost/provenance.go` (EvaluateTrust tamper paths) | `kernel/httpapi/sources.go` (wire `tier` field) | serialization | ✓ WIRED | Confirmed fixed — `Tier: TierTrusted` now flows through to the wire for single-candidate refusals. |
| `kernel/pluginhost/discover_binaries.go` (collision branch) | operator-visible signal (log + `GET /api/sources` launch_failure) | tamper-refusal-on-either-candidate | ✗ NOT WIRED | A tamper refusal on the LOSING side of a collision produces no launch-failure entry and, in one of the two scenarios, a factually incorrect log line ("neither copy carries evidence" when one candidate was actually tamper-refused). |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Provenance/tier/escalation/manifest-gate Go suites pass | `go test ./kernel/pluginhost/... -run 'TestResolveBinary\|TestLaunch_ManifestGate\|TestEscalation\|Provenance\|EvaluateTrust' -count=1 -v` | `ok github.com/davison/topos/kernel/pluginhost 3.186s`, every listed test `--- PASS` | ✓ PASS |
| Full repo builds | `go build ./...` | exit 0, no output | ✓ PASS |
| `go vet` on kernel packages | `go vet ./kernel/...` | exit 0, no output | ✓ PASS |
| Doc-link integrity | `./scripts/check-doc-links.sh` | "checked 59 link(s) across 22 file(s) — all resolve" | ✓ PASS |
| No test exercises the tamper-vs-clean collision combination (CR-01) | `grep -n "func Test" kernel/pluginhost/escalation_test.go kernel/pluginhost/tier_test.go` + manual read of `TestEscalation_ShadowingCannotInheritTrust` and `TestResolveBinary_CollisionResolvesToWhicheverCopyCarriesEvidence` | Both tests cover only "neither side has evidence" or "one side has evidence, other has none" — never "one side is tamper-refused" | ✗ FAIL (confirms CR-01 — no regression net exists) |
| e2e suites (`make e2e`, `make provenance-check`, `make docs-check`) | not independently re-run (network/browser-heavy) | 16-06-SUMMARY and 16-07-SUMMARY both document passing runs with quoted output; `make docs-check`'s doc-link component independently re-run above and passes | ? SKIP (accepted on the strength of SUMMARY-documented runs plus this verifier's independent doc-link and Go-level re-runs) |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| TRUST-01 | 16-01, 16-02, 16-06 | Trust derives from verifiable provenance, not directory | ✓ SATISFIED (core mechanism, and the reporting-field regression is now closed); ⚠️ the collision-resolution edge case (CR-01) is a real, separate defect in the same subsystem | Location-symmetry tests pass; single-candidate wire-tier regression closed; collision-with-tamper combination is untested and defective |
| TRUST-02 | 16-04, 16-05 | Real topos-plugins release verifies as first-party trusted | ✓ SATISFIED | 16-04-TRUST02-PROOF.md, unchanged, real pipeline artifact |
| TRUST-03 | 16-02, 16-05 | Unsigned external fallback unchanged | ✓ SATISFIED | pin_test.go unmodified by gap closure; regression net passing |
| TRUST-04 | 16-03, 16-06 | Escalation paths closed by fail-first tests | ✗ NOT FULLY SATISFIED | The three original escalation scenarios (config edit, file drop, name shadowing without a tamper on either side) are closed and tested. The collision-with-tamper-on-one-side combination is NOT closed by any committed test and the shipped code does not enforce the invariant TRUST-04 requires for it (CR-01). |

No orphaned requirements — REQUIREMENTS.md maps exactly TRUST-01..04 to Phase 16 and all four appear across the seven plans' frontmatter `requirements:` fields.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `kernel/pluginhost/discover_binaries.go` | 496-516 | Collision branch's two `if err == nil && Tier == TierTrusted` guards silently discard a tamper refusal on the losing candidate | 🛑 Blocker | Confirmed unfixed CR-01 — see Gaps |
| `docs/plugin-contract.md` | ~240 | Collision paragraph asserts an invariant the shipped code does not satisfy (see truth 6 note) | ℹ️ Info | The prose is correct as a statement of INTENDED design and matches every other design document; it is the code, not the doc, that needs to change to make the claim true |
| `scripts/install.sh` | ~299-303 | `chmod 0755` applied unconditionally to `*.provenance.json`/`.sig` data files (WR-02, carried over from 16-REVIEW.md, unaddressed by scope) | ⚠️ Warning | Inert but incorrect permission; explicitly out of gap-closure scope per both plans |

No `TBD`/`FIXME`/`XXX` debt markers found in any file touched by the gap-closure plans.

## Human Verification Required

None required to determine the status of this verification — the failing truth (CR-01) is a code defect confirmed by direct inspection and by absence of test coverage, not a matter of subjective judgment.

## Gaps Summary

Phase 16's two previously-recorded gaps are both closed, and closed well:

1. **Gap 1 (wire-contract regression) — CLOSED.** `EvaluateTrust`'s two single-candidate tamper-refusal return paths now explicitly set `Tier: TierTrusted`, matching `docs/api.md`'s documented `manifest_unverified` wire contract. Confirmed by direct code read and by two passing real-`Discover()`-path regression tests that were demonstrated RED before the fix (per 16-06-SUMMARY, independently plausible given the tests assert exactly the previously-broken field).

2. **Gap 2 (documentation regression) — CLOSED.** `docs/plugin-contract.md`'s "Trust tiers" section, plus four further residual regions in the same file, now state the shipped evidence-based (D-11) model and defer to `docs/plugin-trust.md` as canonical. Confirmed by direct read and a clean repo-wide grep sweep for the superseded language.

However, this phase's own post-gap-closure code review (`16-REVIEW.md`, re-reviewed the same day the gap-closure plans landed) surfaced a **new critical finding, CR-01**, which this verification independently confirmed by direct code inspection of the current tree:

**CR-01: The collision resolver in `resolveBinaryDetailed` (`kernel/pluginhost/discover_binaries.go:496-516`) silently discards a tamper refusal when the other colliding candidate resolves cleanly.** Both winner-checks in the collision branch require `err == nil`, so a candidate that IS a genuine tamper refusal (which, since the Gap-1 fix, correctly carries `Tier: TierTrusted` alongside its non-nil error) never satisfies either `if`, and nothing in the branch refuses on its behalf. Two confirmed fall-through scenarios both silently drop the tamper signal — no launch-failure entry, and in one scenario an actively misleading log line ("neither copy carries evidence" when one candidate was, in fact, tamper-refused). This is not a hypothetical: it directly contradicts the exact invariant this phase's own gap-closure plan 16-07 wrote into `docs/plugin-contract.md`'s corrected "Trust tiers" section ("a tamper refusal... resolves to the refusal itself and never falls back to launching the other copy instead") — the newly-corrected documentation and the shipped code now actively disagree on this one path. No test anywhere in the suite (`escalation_test.go`, `tier_test.go`, `manifestgate_test.go`, the `web/e2e/specs/16-*.spec.ts` files) exercises the specific combination of "one collision candidate tamper-refused, the other clean" — confirmed by reading every existing collision-related test in the package.

This bears directly on two of the phase's own ROADMAP success criteria: SC4 ("Every escalation path... is closed by a committed test that fails if its gate is removed... trust cannot be granted by... shadowing a trusted plugin name with a different binary") and SC5 ("the operator can see why a plugin holds the tier it holds — a verification failure names the cause... in logs"). This specific collision-with-tamper shadowing combination is neither closed by a test nor does it name the cause in the log for one of its two fall-through scenarios — so it is treated as a genuine, phase-goal-blocking gap rather than out-of-scope follow-up work, notwithstanding that the practical exploit impact is a detection/transparency loss rather than a direct grant of trust to a tampered binary (in both confirmed scenarios, the binary that actually launches is the one that legitimately earned whatever tier it received — the defect is that the OTHER candidate's tamper status is silently lost, not that a tampered binary is itself elevated).

The code review's own prescribed fix (reordering the collision branch to check for a tamper refusal on either candidate FIRST, before checking for a clean win) is narrow and mechanical, with a code sketch already provided in `16-REVIEW.md`. Two remaining Warnings (WR-01: install.sh verifier-trust-bootstrapping caveat; WR-02: `topos-provenance keygen` charset validation) and one Info item (IN-01: link-time-arm `Diagnostics` asymmetry) from the same review are advisory, not phase-goal-blocking, and were explicitly and correctly scoped out of both gap-closure plans — they do not affect this verdict.

Recommend a short, targeted third gap-closure plan addressing CR-01 specifically (fix the collision-branch ordering per the reviewer's sketch, add the two prescribed regression tests) before the phase is marked shippable.

---

_Verified: 2026-08-20T13:00:00Z_
_Verifier: Claude (gsd-verifier)_
