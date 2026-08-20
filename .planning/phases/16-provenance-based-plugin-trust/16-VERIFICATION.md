---
phase: 16-provenance-based-plugin-trust
verified: 2026-08-20T20:15:00Z
status: passed
score: 6/6 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 5/6
  gaps_closed:
    - "CR-01: resolveBinaryDetailed's two-directory collision branch (kernel/pluginhost/discover_binaries.go:496-531) now checks trustedErr != nil / externalErr != nil on EITHER candidate FIRST, before ever checking either candidate's Tier == TierTrusted, and refuses outright (returning the tamper-refused candidate's path, Trust, and non-nil error) if found — exactly the reordering 16-REVIEW.md prescribed. Confirmed by direct read of the current code (commit 1ab93f6) and by two new passing regression tests in kernel/pluginhost/escalation_test.go's TestEscalation_ShadowingCannotInheritTrust: 'cross-directory: trusted-side tamper refusal wins even though the external copy independently resolves clean' and 'cross-directory: external-side tamper refusal wins even though the trusted copy independently resolves clean' — both exercise the exact untested combination CR-01 identified and both pass on a live re-run (go test ./kernel/pluginhost/... -run TestEscalation -v -count=1 -> PASS). Two pre-existing tests that were unknowingly asserting the old insecure fall-through behavior (TestEscalation_ShadowingCannotInheritTrust's 'cross-directory shadow' subtest and tier_test.go's TestResolveBinary_CollisionResolvesToWhicheverCopyCarriesEvidence) were correctly updated to assert the corrected refuse-outright behavior, with doc comments explaining why the old expectation was itself the bug's fingerprint — read directly and confirmed this is a legitimate strengthening, not a weakening, of the assertions."
  gaps_remaining: []
  regressions: []
---

# Phase 16: Provenance-Based Plugin Trust Verification Report

**Phase Goal:** The kernel decides a plugin's trust tier from verifiable provenance carried by the artifact itself, so a first-party plugin earns trust wherever it lives on disk — and no config edit, file drop, or shadow binary can forge it.
**Verified:** 2026-08-20T20:15:00Z
**Status:** passed
**Re-verification:** Yes — third pass, after gap-closure plans 16-06/16-07 and a code-review fix pass (16-REVIEW-FIX.md, commits 1ab93f6/0ab88a9/b3a2c0b) closing CR-01 (blocking) plus WR-01/WR-02 (advisory, fixed as a bonus)

## History Across All Three Verification Passes

1. **Pass 1 (initial):** score 4/6. Gap 1 (EvaluateTrust's tamper-refusal paths left `Tier` at the zero value on the wire) and Gap 2 (docs/plugin-contract.md still described directory-derived trust) — both closed by gap-closure plans 16-06/16-07.
2. **Pass 2 (re-verification #1):** score 5/6. Gaps 1 and 2 confirmed closed. A NEW finding surfaced by this phase's own post-gap-closure code review (16-REVIEW.md CR-01): `resolveBinaryDetailed`'s collision branch silently discarded a tamper refusal when the other colliding candidate resolved cleanly — untested, and contradicting the very invariant 16-07's doc fix had just written into `docs/plugin-contract.md`.
3. **Pass 3 (this verification):** CR-01 independently confirmed CLOSED by direct code inspection and live test re-run — see below. Score 6/6. No gaps remain.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Trust is location-independent — a first-party binary earns trusted tier wherever it lives, a dropped file earns nothing wherever it lives (SC1, TRUST-01) | ✓ VERIFIED | Unchanged since pass 1/2, no regression: `TestResolveBinary_LocationSymmetric`, `TestResolveBinary_ExternalWithSignedManifestResolvesTrustedTier`, `TestResolveBinary_TrustedWithNoEvidenceResolvesExternalTier` re-run live and pass (`go test ./kernel/pluginhost/... -run 'TestResolveBinary_LocationSymmetric|TestResolveBinary_ExternalWithSignedManifestResolvesTrustedTier|TestResolveBinary_TrustedWithNoEvidenceResolvesExternalTier' -v -count=1` -> all PASS). `git diff --stat a3901a9 HEAD` confirms the three fix commits touched only `discover_binaries.go`, `escalation_test.go`, `tier_test.go`, `provenance.go`/`provenance_test.go`, `cmd/topos-provenance/*`, `scripts/install.sh`/`install-smoke.sh`, `docs/install.md` — no change to this truth's supporting code paths beyond the collision branch itself. |
| 2 | A topos-plugins release binary verifies as first-party trusted on an installed instance with no link-time manifest entry (SC2, TRUST-02) | ✓ VERIFIED | Unchanged — `16-04-TRUST02-PROOF.md` untouched by any commit since pass 1 (`git diff --stat a3901a9 HEAD -- .planning/phases/16-provenance-based-plugin-trust/16-04-TRUST02-PROOF.md` -> empty diff). |
| 3 | The unsigned-external consent-and-pin fallback is unchanged (SC3, TRUST-03) | ✓ VERIFIED | `pin_test.go` untouched by the fix commits (`git diff --stat a3901a9 HEAD -- kernel/pluginhost/pin_test.go` -> empty diff); full `go test ./kernel/pluginhost/...` still green. |
| 4 | Every escalation path (config edit, file drop, binary shadowing — including the collision-with-tamper combination) is closed by a committed test that fails if its gate is removed (SC4, TRUST-04) | ✓ VERIFIED | The three original escalation tests still pass. CR-01's previously-untested combination is now covered by two new subtests in `TestEscalation_ShadowingCannotInheritTrust` — verified by direct read (both subtests construct a genuine tamper on one side via `OverrideBuildManifest` plus mismatched bytes, and a genuine clean win on the other, then assert `err != nil`, `errors.Is(err, ErrManifestUnverified)`, and that the returned `path` names the tamper-refused candidate, never the clean one) and by a live re-run: `go test ./kernel/pluginhost/... -run TestEscalation -v -count=1` -> all 5 top-level tests and all subtests `PASS`, `ok ... 3.3s`. |
| 5 | The operator can see why a plugin holds the tier it holds — a verification failure names the cause on the source chip and in the kernel log, and never silently downgrades a plugin the operator believes is trusted (SC5) | ✓ VERIFIED | Gap 1 remains closed (wire `Tier: TierTrusted` on single-candidate refusals). CR-01's collision-branch silent-drop is now closed: the fixed code logs `hclog.Warn` naming which side refused ("trusted copy is a tamper refusal, refusing regardless of the external copy" / the mirrored external-side message) BEFORE returning the refusal, and returns the tamper-refused candidate's own non-nil error rather than falling through — confirmed by direct read of `discover_binaries.go:503-517`. |
| 6 | Documentation states the trust model in one place; every other document agrees with it, including the code (16-05 must-have, D-11 intent) | ✓ VERIFIED | `docs/plugin-contract.md:245`'s collision paragraph ("...resolves to the refusal itself and never falls back to launching the other copy instead") now MATCHES the shipped code — pass 2 had flagged this as internally consistent-but-inaccurate-to-the-code; that discrepancy is now resolved because the code was fixed to match the doc, not vice versa. `./scripts/check-doc-links.sh` -> "checked 59 link(s) across 22 file(s) — all resolve" (re-run live). |

**Score:** 6/6 truths verified (0 present, behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `kernel/pluginhost/discover_binaries.go` | Collision resolution refuses on any tamper-refused candidate, regardless of the other candidate's outcome | ✓ VERIFIED | Lines 495-531: tamper-refusal checks (`if trustedErr != nil` / `if externalErr != nil`) now precede the clean-win checks, exactly matching 16-REVIEW.md's prescribed fix code. Confirmed by direct read of the current tree. |
| `kernel/pluginhost/escalation_test.go` | Regression coverage for both collision-with-tamper directions | ✓ VERIFIED | Two new subtests present and passing: "cross-directory: trusted-side tamper refusal wins even though the external copy independently resolves clean" and "cross-directory: external-side tamper refusal wins even though the trusted copy independently resolves clean." A third, pre-existing subtest ("cross-directory shadow...") was corrected in place with an explanatory comment, not weakened — confirmed by direct read. |
| `kernel/pluginhost/tier_test.go` | Pre-existing collision test no longer asserts the insecure fall-through outcome | ✓ VERIFIED | `TestResolveBinary_CollisionResolvesToWhicheverCopyCarriesEvidence` now asserts `err != nil`, `errors.Is(err, ErrProvenanceUnverified)`, and `tier == TierTrusted` (a refusal, not a clean external win) — confirmed by direct read, and this reflects a genuinely reachable cross-directory tamper (per 16-REVIEW-FIX.md's documented reasoning about `VerifySignedProvenance`'s exhaustive by-name scan, independently plausible given `provenance.go`'s scan implementation). |
| `docs/plugin-contract.md` | Collision invariant text matches shipped behavior | ✓ VERIFIED | Confirmed by direct read; see truth 6. |
| `scripts/install.sh` (WR-01, non-blocking bonus fix) | Verifier resolution prefers a prior install / PATH verifier over the staged payload's own copy | ✓ VERIFIED (present) | Confirmed by direct read of the reordered resolution block (`$BIN_DIR/topos-provenance` -> `PATH` -> staged payload, in that order) plus a documented bootstrap-trust caveat added to `docs/install.md`. |
| `kernel/pluginhost/provenance.go` + `cmd/topos-provenance/main.go` (WR-02, non-blocking bonus fix) | `--key-id` charset validated at keygen/sign and at parse time | ✓ VERIFIED (present) | `ValidateProvenanceKeyID` (restricted to `^[A-Za-z0-9._-]+$`) wired into `runKeygen`, `runSign`, and `ParseProvenanceKeys`; confirmed by direct read and passing `TestValidateProvenanceKeyID_AcceptsAndRejects`. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `kernel/pluginhost/provenance.go` (EvaluateTrust tamper paths) | `kernel/httpapi/sources.go` (wire `tier` field) | serialization | ✓ WIRED | Unchanged since pass 2 — confirmed still fixed. |
| `kernel/pluginhost/discover_binaries.go` (collision branch) | operator-visible signal (log + refused launch, never a silent fallback) | tamper-refusal-on-either-candidate | ✓ WIRED | CR-01 closed — a tamper refusal on either colliding candidate now returns that candidate's own non-nil error and logs the cause by name before any clean-win check runs, confirmed by direct code read and by two live-passing regression tests exercising both directions. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Provenance/tier/escalation/manifest-gate Go suites pass | `go test ./kernel/pluginhost/... -run 'TestResolveBinary\|TestLaunch_ManifestGate\|TestEscalation\|Provenance\|EvaluateTrust' -count=1 -v` | `ok github.com/davison/topos/kernel/pluginhost 3.338s`, every listed test `--- PASS` (including both new CR-01 regression subtests) | ✓ PASS |
| `cmd/topos-provenance` suite passes (WR-02 regression net) | `go test ./kernel/pluginhost/... ./cmd/topos-provenance/... -count=1` | `ok github.com/davison/topos/kernel/pluginhost 4.736s`, `ok github.com/davison/topos/cmd/topos-provenance 0.005s` | ✓ PASS |
| Full workspace test suite passes (regression sweep) | `go test ./... -count=1` | All 14 packages `ok` (or `[no test files]`), no failures | ✓ PASS |
| Full repo builds | `go build ./...` | exit 0, no output | ✓ PASS |
| `go vet` on kernel packages | `go vet ./kernel/...` | exit 0, no output | ✓ PASS |
| Doc-link integrity | `./scripts/check-doc-links.sh` | "checked 59 link(s) across 22 file(s) — all resolve" | ✓ PASS |
| Install smoke suite, incl. both provenance cases (WR-01 regression net) | `./scripts/install-smoke.sh` | All 17 cases PASS, incl. "provenance — valid signed manifest verifies and installs" and "provenance — binary altered after signing aborts, naming it" | ✓ PASS |
| Provenance CLI smoke test | `./scripts/provenance-smoke.sh` | "provenance-smoke: OK" | ✓ PASS |
| CR-01 collision-with-tamper combination is now covered | Direct read of `TestEscalation_ShadowingCannotInheritTrust`'s two new subtests + live run of `go test ./kernel/pluginhost/... -run TestEscalation -v -count=1` | Both new subtests present, both construct the exact combination CR-01 named, both PASS | ✓ PASS (CR-01 confirmed closed — regression net now exists) |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| TRUST-01 | 16-01, 16-02, 16-06 | Trust derives from verifiable provenance, not directory | ✓ SATISFIED | Location-symmetry tests pass; single-candidate wire-tier regression closed (pass 2); collision-resolution branch now correctly refuses on tamper regardless of the other candidate (CR-01, closed this pass). |
| TRUST-02 | 16-04, 16-05 | Real topos-plugins release verifies as first-party trusted | ✓ SATISFIED | 16-04-TRUST02-PROOF.md, unchanged, real pipeline artifact. |
| TRUST-03 | 16-02, 16-05 | Unsigned external fallback unchanged | ✓ SATISFIED | pin_test.go unmodified across all three verification passes; regression net passing. |
| TRUST-04 | 16-03, 16-06 | Escalation paths closed by fail-first tests | ✓ SATISFIED | All named escalation scenarios — config edit, file drop, name shadowing, AND the collision-with-tamper-on-one-side combination CR-01 identified — are now closed by committed, passing regression tests. |

No orphaned requirements — REQUIREMENTS.md maps exactly TRUST-01..04 to Phase 16 and all four appear across the seven plans' frontmatter `requirements:` fields.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `scripts/install.sh` | ~313-324 | `chmod 0755` applied unconditionally in the generic "place" loop to every asset named in `checksums.txt`, including `*.provenance.json`/`.sig` data files (not just the `topos`/`topos-provenance`/plugin binaries) | ℹ️ Info | Inert but semantically wrong permission bit on two small JSON/signature data files; pre-existing, outside TRUST-01..04 scope, not touched by any of the three CR-01/WR-01/WR-02 fixes; does not affect provenance verification correctness (verification reads file contents, not the executable bit) |

No `TBD`/`FIXME`/`XXX` debt markers found in any file touched by the fix commits (the one `XXX` grep hit in `scripts/install.sh` is `mktemp`'s `.topos-install.XXXXXX` template placeholder, not a debt marker).

## Human Verification Required

None. CR-01's closure is a code defect resolved by a mechanical, reviewer-prescribed reordering fix, confirmed by direct code inspection and by live-passing regression tests exercising both previously-untested directions — not a matter of subjective judgment. WR-01 and WR-02 (advisory, non-blocking) were also independently confirmed fixed by direct code read and live smoke-test runs.

## Gaps Summary

No gaps remain. All three verification passes are now reconciled:

- **Gap 1 (wire-contract regression)** — closed in gap-closure plan 16-06, confirmed in passes 2 and 3.
- **Gap 2 (documentation regression)** — closed in gap-closure plan 16-07, confirmed in passes 2 and 3.
- **CR-01 (collision-branch tamper-refusal silently discarded)** — the sole remaining blocker recorded in pass 2, closed by commit `1ab93f6` (code-review fix pass). Independently re-verified this pass by: (a) direct read of the current `resolveBinaryDetailed` collision branch, confirming it matches 16-REVIEW.md's prescribed reordering exactly; (b) direct read of the two new regression subtests covering both previously-untested directions; (c) a live re-run of the full `kernel/pluginhost` and `cmd/topos-provenance` suites, `go build ./...`, `go vet ./kernel/...`, `./scripts/check-doc-links.sh`, `./scripts/install-smoke.sh`, and `./scripts/provenance-smoke.sh` — all green with no failures; (d) confirming no regression in the four previously-verified truths (1, 2, 3, 6), whose supporting files (`pin_test.go`, `16-04-TRUST02-PROOF.md`) were untouched by the three fix commits and whose supporting tests still pass unchanged.

Two of the fix commits (WR-01, WR-02) address advisory findings from the same code review that were explicitly out of both gap-closure plans' scope and non-blocking for this phase's goal; they were fixed anyway as part of the same review-fix pass and are confirmed working, but their absence would not have changed this verdict.

One residual Info-severity observation (unconditional `chmod 0755` on provenance data files in `scripts/install.sh`'s generic "place" loop) remains unaddressed — it is cosmetic (does not affect verification correctness), pre-existing, and outside the TRUST-01..04 requirement set; recommend a follow-up issue if it is to be tracked at all.

**Phase 16's goal is achieved: the kernel decides a plugin's trust tier from verifiable provenance carried by the artifact itself, wherever the artifact lives on disk, and no config edit, file drop, or shadow binary — including the collision-with-tamper edge case surfaced by this phase's own review — can forge it.**

---

_Verified: 2026-08-20T20:15:00Z_
_Verifier: Claude (gsd-verifier)_
