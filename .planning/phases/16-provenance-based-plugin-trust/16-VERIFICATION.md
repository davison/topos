---
phase: 16-provenance-based-plugin-trust
verified: 2026-08-20T12:00:00Z
status: gaps_found
score: 4/6 must-haves verified
behavior_unverified: 0
overrides_applied: 0
gaps:
  - truth: "The operator can see why a plugin holds the tier it holds — a verification failure names the cause on the source chip and in the kernel log, and never silently downgrades a plugin the operator believes is trusted (ROADMAP success criterion 5)"
    status: failed
    reason: "EvaluateTrust's two tamper-refusal return paths (kernel/pluginhost/provenance.go, link-time-digest-mismatch branch and signed-arm-error branch) construct Trust{} without setting Tier, leaving the wire tier field at its zero value \"\" instead of the documented \"trusted\" for every manifest_unverified LaunchFailure. This is a confirmed regression from pre-Phase-16 behavior (where tier was always TierTrusted in this branch) and contradicts docs/api.md's own worked example (tier: \"trusted\" for the dropped-binary case). It was independently reproduced live during this same phase's own TRUST-02 proof session: 16-04-TRUST02-PROOF.md's negative case 4 (tampered binary) records the actual GET /api/sources payload with \"tier\": \"\" verbatim. No test in the suite asserts LaunchFailures()[0].Tier for a real (non-hand-built) manifest_unverified refusal — confirmed by inspection of manifestgate_test.go and escalation_test.go, matching the code review's WR-01 finding."
    artifacts:
      - path: "kernel/pluginhost/provenance.go"
        issue: "EvaluateTrust's two tamper-refusal return statements omit Tier: TierTrusted, so the zero-value \"\" flows through resolveBinaryDetailed -> launch -> manifestUnverifiedError -> toLaunchFailure -> GET /api/sources' tier field"
      - path: "web/src/lib/components/SourceChip.svelte"
        issue: "isExternal = source.tier === 'external' (line 141) and the file's own doc comment (line 269, 'a manifest-unverified or shadowed source is always tier: \"trusted\"') both assume the wire contract this bug breaks"
    missing:
      - "Set Tier: TierTrusted on both tamper-refusal return paths in EvaluateTrust (kernel/pluginhost/provenance.go)"
      - "A regression test exercising the real Discover()/launch() path (not a hand-built fakeProber fixture) asserting LaunchFailures()[0].Tier == TierTrusted for a genuine digest-mismatch refusal"
  - truth: "Project documentation states the trust model in one place, and every other document (and code comment) that references the trust model agrees with it (16-05 must-have; D-11 intent)"
    status: failed
    reason: "docs/plugin-contract.md's 'Trust tiers' section (lines 206-241) still describes the pre-Phase-16 directory-derived model almost verbatim: it states a binary resolved from the trusted directory 'is pluginhost.TierTrusted', that 'Tier is derived exclusively from which directory a binary resolved from, launch time — never from anything the plugin itself declares', and that on a name collision 'the trusted directory always wins'. All three claims are false under the shipped D-11 model and are directly contradicted by this same phase's own tests (TestResolveBinary_ExternalWithSignedManifestResolvesTrustedTier, TestResolveBinary_LocationSymmetric, TestResolveBinary_CollisionResolvesToWhicheverCopyCarriesEvidence) and by web/e2e/specs/16-signed-provenance-tier.spec.ts's whole premise, plus by docs/plugin-trust.md — this phase's own canonical document — which explicitly states every other trust-touching document 'links back here rather than restating the model.' 16-05-SUMMARY.md self-flags this staleness under 'Issues Encountered' as known but out of the plan's narrowly-scoped Task 3 edit (which only touched the 'integrity control, not publisher authentication' paragraph), leaving the reader-facing Trust tiers section uncorrected."
    artifacts:
      - path: "docs/plugin-contract.md"
        issue: "Lines 206-241 ('Trust tiers' section) restate the superseded directory-derived model, including a shadow-rule paragraph that contradicts the shipped evidence-based collision rule"
    missing:
      - "Rewrite docs/plugin-contract.md:206-241 to state both directories are pure search paths, remove the per-directory TierTrusted/TierExternal assertions, correct the shadow-rule paragraph ('whichever candidate carries valid evidence wins...'), and point the reader at docs/plugin-trust.md for the authoritative model per that document's own stated intent"
---

# Phase 16: Provenance-Based Plugin Trust Verification Report

**Phase Goal:** The kernel derives trust from verifiable artifact provenance rather than directory location, closing the standing escalation paths.
**Verified:** 2026-08-20T12:00:00Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Trust is location-independent — a first-party binary earns trusted tier wherever it lives, a dropped file earns nothing wherever it lives (SC1, TRUST-01) | ✓ VERIFIED | `kernel/pluginhost/discover_binaries.go`'s `resolveBinaryDetailed`/`DiscoverAllTiered` call `EvaluateTrust` per binary (2 call sites, confirmed by grep); `TestResolveBinary_LocationSymmetric`, `TestResolveBinary_ExternalWithSignedManifestResolvesTrustedTier`, `TestResolveBinary_TrustedWithNoEvidenceResolvesExternalTier` pass (re-run live: `go test ./kernel/pluginhost/... -run 'TestResolveBinary\|Provenance\|EvaluateTrust\|ManifestGate\|Escalation\|EmbeddedProvenanceKeys' -count=1` → ok, 3.6s); browser proof in `web/e2e/specs/16-signed-provenance-tier.spec.ts` and `web/e2e/specs/16-file-drop-external-tier.spec.ts` (both present, non-trivial, per SUMMARY claims of `make e2e` passing 148/5 skipped) |
| 2 | A topos-plugins release binary verifies as first-party trusted on an installed instance with no link-time manifest entry (SC2, TRUST-02) | ✓ VERIFIED | `.planning/phases/16-provenance-based-plugin-trust/16-04-TRUST02-PROOF.md` records a real release (`davison/topos-plugins` v0.0.1), a real tag-triggered GitHub Actions run (`success`), `topos-provenance verify` exiting 0 and naming the manifest, `GET /api/sources` reporting `tier: "trusted"` with the kernel's link-time manifest structurally excluding `topos-plugin-demo` (only 6 in-repo plugin names), and fully offline verification inside a `unshare --net` namespace. This is concrete recorded evidence, not an assertion. |
| 3 | The unsigned-external consent-and-pin fallback is unchanged (SC3, TRUST-03) | ✓ VERIFIED | `kernel/pluginhost/pin_test.go` unmodified (`git diff --stat` reported empty in 16-02-SUMMARY, and this file is absent from every plan's `files_modified` list); `TestSources_UnsignedExternalBinaryConsentAndPinPathUnchanged` added as an explicit end-to-end regression net (16-02); `docs/api.md`'s `pin_mismatch`/`manifest_unverified` vocabularies preserved per grep acceptance criteria in 16-05 |
| 4 | Every escalation path (config edit, file drop, binary shadowing) is closed by a committed test that fails if its gate is removed (SC4, TRUST-04) | ✓ VERIFIED | `kernel/pluginhost/escalation_test.go` exists (330 lines) with `TestEscalation_ConfigEditCannotGrantTrust`, `TestEscalation_FileDropCannotGrantTrust`, `TestEscalation_ShadowingCannotInheritTrust` — all pass on live re-run; 16-03-SUMMARY documents a LIVE fail-first demonstration (temporary weakened-gate edit flipped 4 subtests to FAIL, suite exited non-zero, then reverted with `git diff` clean) — the falsifiability property was actually exercised, not merely asserted |
| 5 | The operator can see why a plugin holds the tier it holds — a refusal names the cause on the source chip and in the kernel log, and a tier change is never silent (SC5) | ✗ FAILED | CR-01 (code review): `EvaluateTrust`'s two tamper-refusal return paths never set `Trust.Tier`, so `GET /api/sources` reports `tier: ""` (not `"trusted"`) for a `manifest_unverified` entry — confirmed by direct code read and independently reproduced live in this phase's own `16-04-TRUST02-PROOF.md` (negative case 4 shows the actual `"tier": ""` payload). The kernel log DOES name the cause correctly, but the wire `tier` field — the field `SourceChip.svelte`'s `TrustBadge` branches on — is wrong. See Gaps. |
| 6 | Documentation states the trust model in one place; every other document agrees with it (16-05 must-have, D-11 intent) | ✗ FAILED | CR-02 (code review), confirmed by direct file read: `docs/plugin-contract.md:206-241` still describes the pre-Phase-16 directory-derived model verbatim, contradicting the shipped code and this phase's own canonical `docs/plugin-trust.md`. Self-flagged as known-but-unfixed in 16-05-SUMMARY's "Issues Encountered." See Gaps. |

**Score:** 4/6 truths verified (0 present, behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `kernel/pluginhost/provenance.go` | Signed release-manifest verifier, key set, `EvaluateTrust` authority | ✓ VERIFIED | 653 lines, contains `ed25519`, `HashBinary(`; but see Gap 1 — `EvaluateTrust`'s tamper-refusal paths are a substantive defect, not a stub |
| `kernel/pluginhost/provenance_test.go` | RED-first coverage of every named provenance failure cause | ✓ VERIFIED | 556 lines, exercises all named causes per 16-01-SUMMARY's coverage table |
| `cmd/topos-provenance/main.go` | keygen/sign/verify CLI | ✓ VERIFIED | 299 lines, wraps `BuildProvenanceManifest`/`SignProvenanceManifest`/`VerifySignedProvenance` |
| `scripts/provenance-smoke.sh` | Committed round-trip gate | ✓ VERIFIED | 168 lines, executable, wired as `make provenance-check` |
| `kernel/pluginhost/discover_binaries.go` | Provenance-driven tier derivation | ✓ VERIFIED | `EvaluateTrust(` appears 7×; `resolveBinaryDetailed` and `DiscoverAllTiered` both call it |
| `kernel/config/types.go` | `PluginsConfig` documented as pure search paths | ✓ VERIFIED | "search path" language present, directory-derived-trust sentence removed (per 16-02 grep acceptance criteria) |
| `kernel/pluginhost/tier_test.go` | Location-independent tier coverage | ✓ VERIFIED | 502 lines, includes `TestResolveBinary_LocationSymmetric` and collision-evidence cases |
| `kernel/pluginhost/escalation_test.go` | TRUST-04's three escalation tests + fail-first proof | ✓ VERIFIED | 330 lines, cites the todo and TRUST-04, 3 `TestEscalation_*` functions confirmed passing |
| `web/e2e/specs/16-file-drop-external-tier.spec.ts` | Browser proof, dropped binary lands untrusted | ✓ VERIFIED | 131 lines, present |
| `.planning/phases/16-provenance-based-plugin-trust/16-04-TRUST02-PROOF.md` | Recorded end-to-end TRUST-02 proof | ✓ VERIFIED | 266 lines; also the artifact that independently documents Gap 1 in its own case 4 output |
| `scripts/install.sh` | Provenance verification wired into verify step | ✓ VERIFIED | contains "provenance" 30×, "topos-provenance" 16× — but WR-02 (chmod 0755 on `.provenance.json`/`.sig` data files) is a real, confirmed minor defect (warning, not blocker) |
| `web/e2e/specs/16-signed-provenance-tier.spec.ts` | Browser proof of location-independent signed trust | ✓ VERIFIED | 224 lines, present |
| `docs/plugin-trust.md` | Canonical trust-model statement | ✓ VERIFIED | 123 lines (≥60 required), contains "external-tier by construction" and `topos-provenance verify` — but see Gap 2: this document is correct, the documents that are supposed to defer to it are not all consistent |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `kernel/pluginhost/discover_binaries.go` | `kernel/pluginhost/provenance.go` | `EvaluateTrust(` | ✓ WIRED | 7 call sites |
| `kernel/pluginhost/host.go` | resolver's `Trust` value | `launch` gates directly on `resolveBinaryDetailed`'s result | ✓ WIRED | `EvaluateTrust(` count in host.go is 0 by design (16-02 moved the call into the resolver; `launch` consumes the already-produced `Trust`), and `VerifyTrustedBinary(` count in host.go is 0 — confirmed live |
| `cmd/topos-provenance/main.go` | `kernel/pluginhost/provenance.go` | `pluginhost.` | ✓ WIRED | CLI wraps kernel producers/verifier, not a reimplementation |
| `scripts/install.sh` | `cmd/topos-provenance` | verify step | ✓ WIRED | resolution order staged→PREFIX/bin→PATH per 16-05 |
| `kernel/pluginhost/provenance.go` (EvaluateTrust tamper paths) | `kernel/httpapi/sources.go` (`Tier: string(f.Tier)`) | wire serialization | ✗ HOLLOW | Trust.Tier zero-value flows unmodified through `resolveBinaryDetailed`, `launch`, `manifestUnverifiedError`, `toLaunchFailure()` to the wire — confirmed by code trace and by the live-recorded `"tier": ""` payload in 16-04-TRUST02-PROOF.md |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Provenance/tier/escalation/manifest-gate Go suites pass | `go test ./kernel/pluginhost/... -run 'TestResolveBinary\|Provenance\|EvaluateTrust\|ManifestGate\|Escalation\|EmbeddedProvenanceKeys' -count=1` | `ok  github.com/davison/topos/kernel/pluginhost  3.594s` | ✓ PASS |
| Full repo builds | `go build ./...` | exit 0 | ✓ PASS |
| No test asserts the real wire `Tier` for a manifest_unverified `LaunchFailure` | `grep -n '\.Tier\b' kernel/pluginhost/manifestgate_test.go kernel/pluginhost/escalation_test.go` | Only `Tier()`/`desc.Tier` assertions on success paths and `tb.Tier` in a listing-tag case; no assertion on `LaunchFailures()[0].Tier` for a real refusal | ✗ FAIL (confirms Gap 1 / WR-01 — no regression net exists for CR-01) |
| e2e/e2e-only suites (`make e2e`, `make provenance-check`, `make install-check`, `make docs-check`) | not re-run (network/browser-heavy, long-running; SUMMARY claims and code review's independent 31-file review both report green) | not independently re-executed | ? SKIP — accepted on the strength of the code review's own independent pass over these files plus this verifier's own Go-level re-runs |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| TRUST-01 | 16-01, 16-02 | Trust derives from verifiable provenance, not directory | ✓ SATISFIED (core mechanism); ⚠️ the SC5 visibility clause tied to this same tier-derivation code is broken (Gap 1) | Location-symmetry tests pass; wire-tier regression is a defect in the same subsystem, not a failure of the core TRUST-01 mechanism (launch still correctly refuses) |
| TRUST-02 | 16-04, 16-05 | Real topos-plugins release verifies as first-party trusted | ✓ SATISFIED | 16-04-TRUST02-PROOF.md, real pipeline artifact, offline verification |
| TRUST-03 | 16-02, 16-05 | Unsigned external fallback unchanged | ✓ SATISFIED | pin_test.go unmodified, explicit regression net added and passing |
| TRUST-04 | 16-03 | Escalation paths closed by fail-first tests | ✓ SATISFIED | escalation_test.go, live fail-first demonstration recorded |

No orphaned requirements — REQUIREMENTS.md maps exactly TRUST-01..04 to Phase 16 and all four appear in plan frontmatter `requirements:` fields.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `kernel/pluginhost/provenance.go` | ~635-647 | `EvaluateTrust`'s tamper-refusal branches construct `Trust{}` without `Tier` | 🛑 Blocker | Wire `tier` field silently reports `""` instead of `"trusted"` for a tampered/hijacked binary — see Gap 1 |
| `docs/plugin-contract.md` | 206-241 | Stale directory-derived trust model description | 🛑 Blocker | Misleads an operator/reader about the actual (evidence-based) collision and tier rules — see Gap 2 |
| `scripts/install.sh` | ~299-303 | `chmod 0755` applied unconditionally to `*.provenance.json`/`.sig` data files | ⚠️ Warning | Inert but incorrect permission on plain data files (WR-02, not independently confirmed to be blocking; matches code review) |
| `kernel/pluginhost/discover_binaries.go` | ~495-515 | Collision-fallback log line can misname a tamper refusal as "neither copy carries evidence" | ℹ️ Info | Log wording imprecision only; behavior is correct (IN-01) |

No `TBD`/`FIXME`/`XXX` debt markers found in any file touched by this phase (checked across all 22 files listed in the code review's file list).

## Human Verification Required

None required to determine the status of this verification — the two failing truths (Gap 1, Gap 2) are code/documentation defects confirmed by direct inspection, not items needing subjective human judgment.

### Deferred, non-blocking item (surfaced per orchestrator instruction, not a gap)

**Live-instance trusted-chip re-check on the operator's real installed kernel** — 16-04-SUMMARY.md records (coverage item D6, `human_judgment: true`) that the operator approved plan 16-04's checkpoint on the strength of the throwaway-prefix end-to-end proof (`16-04-TRUST02-PROOF.md`), explicitly deferring the checkpoint's steps 3-5 (add a source on the operator's own live installed instance; remove/restore `.provenance.sig` there; confirm the chip renders trusted/untrusted correctly) to post-merge/rebuild as part of phase UAT, because the operator's currently-installed kernel predates the embedded `topos-plugins-2026a` key. This is a legitimate, already-tracked deferred item, not a new gap raised by this verification — it should be exercised once a kernel built from this merged work is actually installed.

## Gaps Summary

Phase 16 lands a substantial, well-tested provenance-trust mechanism: the signed release-manifest arm (16-01), the location-independent tier rewrite (16-02), the fail-first-demonstrated escalation suite (16-03), a real end-to-end proof against a genuine `topos-plugins` release (16-04), and install-time verification plus a canonical trust-model document (16-05). Success criteria 1-4 are solidly verified with concrete, re-run evidence (Go suites re-executed live by this verifier; the TRUST-02 proof is a real artifact, not an assertion).

However, the phase's own code review (`16-REVIEW.md`) surfaced two CRITICAL findings that remain unfixed in the current tree, and this verification independently confirmed both by direct code inspection:

1. **CR-01 (wire-contract regression):** `EvaluateTrust`'s tamper-refusal paths never set `Trust.Tier`, so `GET /api/sources` reports `tier: ""` instead of the documented `"trusted"` for a `manifest_unverified` entry. This is not hypothetical — the phase's own `16-04-TRUST02-PROOF.md` independently captured this exact `"tier": ""` payload while proving TRUST-02's negative case. No test in the suite guards against it (confirmed: no assertion on `LaunchFailures()[0].Tier` exists anywhere in `manifestgate_test.go` or `escalation_test.go`). This directly undermines ROADMAP success criterion 5 ("the operator can see why a plugin holds the tier it holds ... on the source chip") since the wire field the chip's `TrustBadge` reads is wrong, even though the chip's other launch_failure-driven signal channels still show the failure.

2. **CR-02 (documentation regression):** `docs/plugin-contract.md`'s "Trust tiers" section still documents the pre-Phase-16 directory-derived model almost verbatim — directly contradicting the shipped code, this phase's own tests, and its own canonical `docs/plugin-trust.md`. 16-05's own SUMMARY self-flags this as known but left out of that plan's narrowly-scoped documentation edit.

Both are narrow, mechanically-fixable defects (a two-line code fix for CR-01; a section rewrite for CR-02, both already scoped with exact fix guidance in `16-REVIEW.md`) rather than architectural problems — the underlying security mechanism (refuse-and-create-no-subprocess) is correct and well-tested throughout. Recommend a short closure plan addressing both before the phase is marked shippable.

---

_Verified: 2026-08-20T12:00:00Z_
_Verifier: Claude (gsd-verifier)_
