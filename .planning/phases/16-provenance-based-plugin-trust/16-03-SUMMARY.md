---
phase: 16-provenance-based-plugin-trust
plan: 03
subsystem: security
tags: [plugin-trust, provenance, pluginhost, trust-04, e2e, escalation-tests]

# Dependency graph
requires:
  - phase: 16-01
    provides: "EvaluateTrust (the single provenance authority), Trust struct, VerifySignedProvenance, ErrProvenanceUnverified, OverrideProvenanceKeys test seam"
  - phase: 16-02
    provides: "Provenance-driven resolveBinaryDetailed/DiscoverAllTiered, ErrPinMismatch/manifestUnverifiedError distinguishing not-found from tamper-refusal, Dirs as pure search paths"
provides:
  - "kernel/pluginhost/escalation_test.go: TRUST-04's three committed escalation tests (config edit, file drop, name shadowing), each driving the real EvaluateTrust/resolveBinaryDetailed/launch gate, plus a live-demonstrated fail-first falsifiability proof"
  - "web/e2e/specs/16-file-drop-external-tier.spec.ts: the browser-visible proof that a dropped, unsigned binary lands untrusted (destructive chip, trust badge, no synced items), never a silent trusted launch"
  - "web/e2e/specs/13-manifest-unverified.spec.ts repointed to the tampered-provenance case D-11 left it covering, distinct from the file-drop case"
  - "web/e2e fixtures: externalBinaryLinks (config-builder.ts/kernel.ts/plugin-binaries.ts) — a renamed-destination link mechanism for the external directory, the sibling of the existing trustedBinaryLinks seam, needed to keep 11-external-tier-badge.spec.ts and 12-external-rehearsal.spec.ts meaningful under D-11's location-independent trust"
  - "docs/testing.md: make provenance-check indexed as the eighth gate; the trust-tier spec section and escalation_test.go pointer kept current"
affects: [16-04, 16-05]

# Actuals (#2632)
actuals:
  tokens: 13095
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Falsifiability proof without a new production seam: a skipped-by-default Go test (env-var gated) asserting the OPPOSITE outcome of the real gate's own tests, documented alongside the exact temporary edit a maintainer applies to see the suite go red — performed live in this plan (edit -> 4 subtests fail, suite exits non-zero -> revert -> git diff clean) rather than merely asserted in prose."
    - "Renamed-destination binary linking (externalBinaryLinks, mirroring the existing trustedBinaryLinks/linkPluginBinaryAs seam) as the e2e-harness answer to D-11's location-independence: a fixture proving 'this real binary resolves external tier' must place it under a name the kernel's link-time build manifest does not cover, never under its own name, once tier stopped being directory-derived."

key-files:
  created:
    - kernel/pluginhost/escalation_test.go
    - web/e2e/specs/16-file-drop-external-tier.spec.ts
  modified:
    - web/e2e/specs/13-manifest-unverified.spec.ts
    - docs/testing.md
    - web/e2e/specs/11-external-tier-badge.spec.ts
    - web/e2e/specs/12-external-rehearsal.spec.ts
    - web/e2e/fixtures/config-builder.ts
    - web/e2e/fixtures/kernel.ts
    - web/e2e/fixtures/plugin-binaries.ts

key-decisions:
  - "13-manifest-unverified.spec.ts's new control instance is topos-plugin-mockstrict (not topos-plugin-mock) so a genuinely tampered topos-plugin-mock copy can occupy that name in the SAME trusted directory without a file-path collision — both names are MANIFEST_E2E_BINARIES entries, so the fixture needed no new build target."
  - "16-file-drop-external-tier.spec.ts reuses the OLD 13-manifest-unverified.spec.ts fixture shape verbatim (trustedBinaryLinks, no pin) — that exact shape is now the file-drop case under D-11, which is why the repoint and the new spec are one task."
  - "The pre-existing make e2e regression in 11-external-tier-badge.spec.ts/12-external-rehearsal.spec.ts (Rule 3 deviation, see below) is fixed via a NEW renamed-destination fixture primitive (externalBinaryLinks) rather than swapping either spec's plugin type — preserves each spec's original intent (mockstrict's two-tier tracer proof; topos-plugin-filesystem's real-source-plugin external rehearsal) while restoring the genuine 'no evidence' precondition each depends on."

patterns-established:
  - "externalBinaryLinks as the external-directory sibling of trustedBinaryLinks: any future e2e fixture needing 'a real, manifest-covered binary's bytes, proven genuinely evidence-free' links it under a renamed destination via this field, never by re-adding it to a build manifest exclusion list."

requirements-completed: [TRUST-04, TRUST-03]

coverage:
  - id: D1
    description: "Config-edit escalation closed: pointing Dirs.Trusted or Dirs.External at an attacker-chosen directory holding an unsigned binary grants nothing — TierExternal from both directions, launch refuses with ErrPinMismatch, no subprocess"
    requirement: "TRUST-04"
    verification:
      - kind: unit
        ref: "kernel/pluginhost/escalation_test.go#TestEscalation_ConfigEditCannotGrantTrust"
        status: pass
    human_judgment: false
  - id: D2
    description: "File-drop escalation closed (Go level): a covered and an uncovered binary in the SAME directory receive different tiers from the same Dirs value; DiscoverAllTiered tags both correctly; the dropped binary's launch refuses"
    requirement: "TRUST-04"
    verification:
      - kind: unit
        ref: "kernel/pluginhost/escalation_test.go#TestEscalation_FileDropCannotGrantTrust"
        status: pass
    human_judgment: false
  - id: D3
    description: "Name-shadowing escalation closed: a digest-mismatched binary under a legitimately-named manifest entry is a REFUSAL (ErrProvenanceUnverified), never a demotion to external; the not-named-at-all sibling is genuinely external; a cross-directory shadow resolves to whichever copy carries evidence and logs the collision by name"
    requirement: "TRUST-04"
    verification:
      - kind: unit
        ref: "kernel/pluginhost/escalation_test.go#TestEscalation_ShadowingCannotInheritTrust (3 subtests)"
        status: pass
    human_judgment: false
  - id: D4
    description: "The escalation suite is demonstrated fail-first, not merely asserted to be: a temporary local edit weakening resolveBinaryDetailed's trusted-branch evaluation to an unconditional TierTrusted grant flips 4 subtests to FAIL and the suite exits non-zero; reverted cleanly (git diff empty)"
    requirement: "TRUST-04"
    verification:
      - kind: other
        ref: "live executor session: TOPOS_ESCALATION_FAILFIRST_PROOF=1 go test ./kernel/pluginhost/ -run TestEscalation -count=1 (exit 1) against the temporary edit, then reverted and go test ./kernel/pluginhost/... -count=1 (exit 0)"
        status: pass
    human_judgment: false
  - id: D5
    description: "The browser proves a dropped, unsigned binary with no pin recorded lands untrusted/consent-required (destructive chip, trust badge, a Trust updated binary… remedy, zero synced items), never a silent trusted launch, and the kernel log names the refused binary"
    requirement: "TRUST-04"
    verification:
      - kind: e2e
        ref: "web/e2e/specs/16-file-drop-external-tier.spec.ts"
        status: pass
    human_judgment: false
  - id: D6
    description: "The tampered-provenance surface (a binary whose bytes no longer match a legitimately-named build-manifest entry) keeps its destructive-chip, no-reachable-probe, no-re-pin-action, named-kernel-log regression net after being repointed for D-11"
    requirement: "TRUST-03"
    verification:
      - kind: e2e
        ref: "web/e2e/specs/13-manifest-unverified.spec.ts"
        status: pass
    human_judgment: false
  - id: D7
    description: "The whole Playwright suite is green, including TRUST-03's own browser regression net (11-external-tier-badge, 11-untrusted-add, 11-binary-changed-repin, 13-shadowed-advisory) — no pre-existing spec regressed"
    requirement: "TRUST-03"
    verification:
      - kind: e2e
        ref: "make e2e (full suite)"
        status: pass
    human_judgment: false
  - id: D8
    description: "docs/testing.md names every gate and spec this phase added — make provenance-check indexed, 16-file-drop-external-tier.spec.ts and the repointed 13-manifest-unverified.spec.ts entries current, escalation_test.go pointed at"
    verification:
      - kind: other
        ref: "make docs-check"
        status: pass
    human_judgment: false

# Metrics
duration: 74min
completed: 2026-08-20
status: complete
---

# Phase 16 Plan 3: Provenance-Based Plugin Trust — TRUST-04 Escalation Suite Summary

**Three committed Go tests (config edit, file drop, name shadowing) close TRUST-04 by driving the real `EvaluateTrust`/`launch` gate directly, demonstrated fail-first live against a temporary weakened-gate edit, paired with the browser-visible file-drop proof and a repointed tampered-provenance spec — closing the standing 2026-08-13 security todo by evidence, not assertion.**

## Performance

- **Duration:** 74 min
- **Started:** 2026-08-20T02:15:00+01:00 (approx.)
- **Completed:** 2026-08-20T03:29:00+01:00 (approx.)
- **Tasks:** 3
- **Files modified:** 9 (2 created, 7 modified)

## Accomplishments

- `kernel/pluginhost/escalation_test.go`: three named tests — `TestEscalation_ConfigEditCannotGrantTrust`, `TestEscalation_FileDropCannotGrantTrust`, `TestEscalation_ShadowingCannotInheritTrust` (3 subtests) — each drives the real resolver (`ResolveBinary`/`resolveBinaryDetailed`) and the real `launch` gate, never a stand-in. The suite's fail-first property was demonstrated LIVE in this session: temporarily replacing `resolveBinaryDetailed`'s trusted-branch `EvaluateTrust` call with an unconditional `TierTrusted` grant flipped 4 subtests to FAIL and the suite exited non-zero; reverted cleanly (`git diff` empty afterward). A fourth, skipped-by-default test documents the exact edit and command for a future maintainer to reproduce this without needing a new production-code seam.
- `web/e2e/specs/16-file-drop-external-tier.spec.ts` (new): proves a dropped, unsigned `topos-plugin-external-demo` binary with no pin recorded lands untrusted — destructive chip, trust badge, a `Trust updated binary…` remedy (the same consent path any never-pinned external add presents), zero synced stream items, and the kernel log naming the refused instance by name.
- `web/e2e/specs/13-manifest-unverified.spec.ts` repointed for D-11: its fixture now links a tampered copy of `topos-plugin-mock` (one byte appended) under the name `topos-plugin-mock` — a name `MANIFEST_E2E_BINARIES` DOES cover, so the kernel's link-time manifest positively vouches for it with a digest that no longer matches. Every prior assertion (destructive chip, contract-exact tooltip, no reachable probe, no re-pin action, named kernel log) survives unchanged.
- `docs/testing.md`: `make provenance-check` indexed as the eighth gate; the trust-tier spec section renamed and extended to cover all three specs plus a pointer to `escalation_test.go` as the Go-level home of TRUST-04's coverage.

## Task Commits

Each task was committed atomically:

1. **Task 1: The three escalation tests** — `33ba531` (test)
2. **Task 2: Repoint the dropped-binary spec and add the file-drop tier spec** — `7e6c7df` (test)
3. **Task 3: Keep the testing map honest** — `ef78aab` (docs)

**Plan metadata:** commit pending (this SUMMARY + REQUIREMENTS.md, parallel/worktree mode — STATE.md/ROADMAP.md are updated centrally by the orchestrator)

## Files Created/Modified

- `kernel/pluginhost/escalation_test.go` — TRUST-04's three escalation tests plus the fail-first falsifiability proof
- `web/e2e/specs/16-file-drop-external-tier.spec.ts` — the file-drop path's browser proof
- `web/e2e/specs/13-manifest-unverified.spec.ts` — repointed to the tampered-provenance case
- `docs/testing.md` — gate table + spec index kept current
- `web/e2e/specs/11-external-tier-badge.spec.ts` — gap-closure: external-tier participant linked under a renamed, manifest-uncovered destination
- `web/e2e/specs/12-external-rehearsal.spec.ts` — gap-closure: same fix for the filesystem-plugin external rehearsal
- `web/e2e/fixtures/config-builder.ts` — new `externalBinaryLinks` field + its pin computation
- `web/e2e/fixtures/kernel.ts` — wires `externalBinaryLinks` into the external directory at boot
- `web/e2e/fixtures/plugin-binaries.ts` — new `hashPluginBinaryAtPath` helper

## Decisions Made

- `13-manifest-unverified.spec.ts`'s control instance moved from `topos-plugin-mock` to `topos-plugin-mockstrict` so the tampered `topos-plugin-mock` copy can occupy that exact name in the same trusted directory without a file collision — both names are already `MANIFEST_E2E_BINARIES` entries, so no new build target was needed.
- The new file-drop spec deliberately reuses the OLD `13-manifest-unverified.spec.ts` fixture shape (`trustedBinaryLinks`, no pin) verbatim — that exact shape IS the file-drop case under D-11, which is why repointing 13 and adding 16 are one task, not two unrelated changes.
- The pre-existing `make e2e` regression in `11-external-tier-badge.spec.ts`/`12-external-rehearsal.spec.ts` (see Deviations) is fixed via a new renamed-destination fixture primitive (`externalBinaryLinks`) rather than swapping either spec's plugin type — this preserves each spec's original intent (mockstrict's two-tier tracer proof; the real filesystem plugin's external-tier rehearsal, Phase 12 criterion 5) while restoring the genuine "no evidence" precondition each spec's assertions depend on.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `spec-hygiene.spec.ts`'s mechanical `mkdtempSync` substring check tripped on my own doc comment**
- **Found during:** Task 2, first `make e2e` run against the repointed `13-manifest-unverified.spec.ts`
- **Issue:** My header comment for the tampered-scratch-directory setup literally contained the string "never mkdtempSync directly" as prose explaining the convention — `spec-hygiene.spec.ts`'s gate does a plain substring search over the whole file text (not code-aware), so the comment itself tripped the check even though the actual code correctly used `mkdtempCorpus`.
- **Fix:** Reworded the comment to avoid the literal substring while preserving the same explanation.
- **Files modified:** `web/e2e/specs/13-manifest-unverified.spec.ts`
- **Verification:** `spec-hygiene.spec.ts`'s both cases pass; `grep -c mkdtempSync` over the file returns 0.
- **Committed in:** `7e6c7df` (Task 2 commit)

**2. [Rule 3 - Blocking] Pre-existing `make e2e` regression in two Phase 11/12 specs, introduced by 16-02's D-11 tier rewrite and never caught (16-02's own verification never ran `make e2e`)**
- **Found during:** Task 2, first full `make e2e` run (required by this plan's own `<verification>`/acceptance criteria)
- **Issue:** `11-external-tier-badge.spec.ts` and `12-external-rehearsal.spec.ts` each placed a link-time-manifest-covered binary (`topos-plugin-mockstrict`, `topos-plugin-filesystem` — both in `MANIFEST_E2E_BINARIES`) into the EXTERNAL directory under its OWN name, expecting it to resolve `TierExternal`. Under 16-02's D-11 rewrite, tier is a pure function of provenance: a manifest-covered binary now resolves `TierTrusted` from ANY directory, including the external one (success criterion 1, "trust is no longer a property of location"). Both specs' whole premise — "same binary, external directory, therefore untrusted" — is exactly the bug Phase 16 fixes, so both broke: `mockstrict?.tier` returned `"trusted"` instead of `"external"`, the trust badge never rendered, and (for `12-external-rehearsal.spec.ts`) the kernel refused to even boot (bare port-bind failure surfaced instead, since the "external" instance never actually needed a pin under the OLD behavior and had none under the new one).
- **Fix:** Added `externalBinaryLinks` — a new `FixtureConfigSpec` field mirroring the existing `trustedBinaryLinks`/`linkPluginBinaryAs` seam, but targeting the external directory — plus `hashPluginBinaryAtPath` so the renamed destination can be pinned correctly. Both specs now link their real, unmodified binary under a RENAMED destination the manifest does not cover (`topos-plugin-mockstrict-untrusted`, `topos-plugin-filesystem-untrusted`), restoring the genuine "no evidence, external tier" precondition each spec's assertions depend on — the underlying plugin behavior is unaffected (neither plugin's Go code reads its own `argv[0]`/filename).
- **Files modified:** `web/e2e/specs/11-external-tier-badge.spec.ts`, `web/e2e/specs/12-external-rehearsal.spec.ts`, `web/e2e/fixtures/config-builder.ts`, `web/e2e/fixtures/kernel.ts`, `web/e2e/fixtures/plugin-binaries.ts`
- **Verification:** `make e2e`'s full suite: 146 passed, 5 skipped, 0 failed (confirmed across three consecutive clean runs after the fix; two earlier failing runs — one with a transient ephemeral-port flake in an unrelated spec, one before this fix — are not the current state).
- **Committed in:** `7e6c7df` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 3 — blocking issues preventing this plan's own required `make e2e` verification from passing)
**Impact on plan:** Deviation 1 was a one-line wording fix. Deviation 2 touched 5 files outside this plan's declared `files_modified` list, but was strictly necessary: without it, `make e2e` could not exit 0 (an explicit acceptance criterion of Task 2 and this plan's own `<verification>` block), and the underlying cause was a genuine, unintentional regression from Wave 2 (16-02), not a deliberate architectural choice needing a human decision — the fix mechanically restores each affected spec's ORIGINAL intent under the new provenance model, changing no application source (`web/src`) and no plugin behavior.

## Issues Encountered

- A `disk quota exceeded` error from the Go linker interrupted `make provenance-check` and one `make e2e` run mid-session — traced to `/tmp` (tmpfs) sitting at 80% capacity, largely from ~5GB of leaked `topos-shutdown-signal-test-*` temp directories accumulated by an unrelated, pre-existing test leak in `kernel/supervisor`'s own test suite across many prior sessions (dated back to 2026-08-12), not something this plan's changes caused. Removed the stale ones older than the current session to free space; not a code change, not committed, and out of this plan's scope to fix at the source (logged here for visibility, not filed as a todo since it is a resource-hygiene concern on this specific machine, not a product defect).
- One `make e2e` run reported a single flaky failure (`16-file-drop-external-tier.spec.ts`, ephemeral-port bind collision under full-parallel load) that did not reproduce on the next two consecutive runs — the harness's own documented TOCTOU risk in `allocateEphemeralPort`, not a defect in this plan's spec.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- TRUST-04 is closed by a committed, fail-first-demonstrated Go test suite and a browser proof; TRUST-03's full regression net (Go + browser) is green.
- The standing todo `2026-08-13-plugin-trust-tier-is-directory-location-not-provenance.md` is closed by evidence: all three named escalation paths (config edit, file drop, shadowing) are proven closed, and the suite is proven falsifiable, not merely written.
- `go build ./...`, `go vet ./...`, `go test ./... -count=1`, `make test` (including the cgo `test-signal` target), `make provenance-check`, `make e2e` (146 passed, 5 skipped), and `make docs-check` all pass locally.
- No blockers for 16-04 (real `topos-plugins` signing key + `embeddedProvenanceKeys` populated) or 16-05 (install-time verification via `topos-provenance verify`, D-09) — neither depends on this plan's surface beyond what 16-01/16-02 already provide.
- Phase 16's own success criterion 4 ("every escalation path... closed by a committed test that fails if its gate is removed") is satisfied and demonstrated live, not merely asserted.

---
*Phase: 16-provenance-based-plugin-trust*
*Completed: 2026-08-20*

## Self-Check: PASSED

- `kernel/pluginhost/escalation_test.go` — FOUND
- `web/e2e/specs/16-file-drop-external-tier.spec.ts` — FOUND
- `web/e2e/specs/13-manifest-unverified.spec.ts` — FOUND
- `docs/testing.md` — FOUND
- `web/e2e/fixtures/config-builder.ts` — FOUND
- `web/e2e/fixtures/kernel.ts` — FOUND
- `web/e2e/fixtures/plugin-binaries.ts` — FOUND
- Commit `33ba531` (Task 1) — FOUND in `git log --oneline --all`
- Commit `7e6c7df` (Task 2) — FOUND in `git log --oneline --all`
- Commit `ef78aab` (Task 3) — FOUND in `git log --oneline --all`
- All plan-level `<verification>` commands re-run and green: `make test`, `go test ./kernel/pluginhost/ -run TestEscalation -count=1` (3+ named tests pass), fail-first proof executed live (failing test names recorded above), `make e2e` (146 passed, 5 skipped, 0 failed), `make docs-check`
- `go build ./...`, `go vet ./...` — clean
