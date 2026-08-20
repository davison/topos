---
phase: 16-provenance-based-plugin-trust
fixed_at: 2026-08-20T19:40:00Z
review_path: .planning/phases/16-provenance-based-plugin-trust/16-REVIEW.md
iteration: 1
findings_in_scope: 3
fixed: 3
skipped: 0
status: all_fixed
---

# Phase 16: Code Review Fix Report

**Fixed at:** 2026-08-20T19:40:00Z
**Source review:** .planning/phases/16-provenance-based-plugin-trust/16-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope (critical + warning): 3
- Fixed: 3
- Skipped: 0
- Out of scope (fix_scope=critical_warning): IN-01 (info) — not attempted

## Fixed Issues

### CR-01: Collision resolver silently discards a tamper refusal when the colliding candidate resolves cleanly

**Files modified:** `kernel/pluginhost/discover_binaries.go`, `kernel/pluginhost/escalation_test.go`, `kernel/pluginhost/tier_test.go`
**Commit:** `1ab93f6`
**Applied fix:** Rewrote `resolveBinaryDetailed`'s two-directory collision branch to check for a tamper refusal (`trustedErr != nil` / `externalErr != nil`) on *either* candidate first and refuse immediately if found, before ever checking either candidate's `Tier == TierTrusted`. This matches `docs/plugin-contract.md`'s documented invariant that a tamper refusal never falls back to launching the other copy — applied exactly as the review's suggested fix code, adapted only to keep the existing per-branch logging shape.

Added two new regression tests to `escalation_test.go`'s `TestEscalation_ShadowingCannotInheritTrust` covering both directions the review named: (a) trusted-side tamper refusal (via the link-time arm) with an independently clean external-side win must refuse, never launch external; (b) the mirrored direction (external-side tamper refusal, trusted-side clean win) must refuse, never launch trusted while silently dropping evidence that the external copy sharing its name was tampered.

**Adaptation note (intelligent fix application):** While implementing the requested regression tests, discovered that the review's literal scenario 2 ("external-directory copy is tamper-refused; trusted-directory copy has no evidence at all, plain `TierExternal`") is structurally unreachable given how `VerifySignedProvenance`/`VerifyTrustedBinary` actually work: both arms resolve their manifest entry purely by *name*, scanning both `dirs.Trusted` and `dirs.External` exhaustively before deciding (D-08) — so once any manifest names a colliding binary at all, *every* same-named candidate is classified match-or-mismatch against that one entry, never left unevaluated. This also means two **pre-existing** tests in the suite (`TestEscalation_ShadowingCannotInheritTrust`'s "cross-directory shadow resolves..." subtest and `tier_test.go`'s `TestResolveBinary_CollisionResolvesToWhicheverCopyCarriesEvidence`) were unknowingly asserting the exact insecure behavior CR-01 closes: their fixture (one candidate with a valid signed manifest matching its own bytes, the other candidate with unrelated different bytes sharing the same filename) makes the *other* candidate a genuine cross-directory tamper mismatch too, not "no evidence" — and both tests previously expected the old, buggy "external wins cleanly" outcome. Both were updated (with clarifying doc comments explaining the mechanism) to assert the corrected refuse-outright behavior; this was necessary to apply CR-01's fix without leaving the suite red, and is squarely in scope since it demonstrates the exact vulnerability being closed.

Verified with `go test ./kernel/pluginhost/... -count=1` (all pass, run 3x for flake-check), `go build ./...`, `go vet ./...`, and the full `make test` (test-portable + test-signal, cgo Signal plugin) — all green.

### WR-01: install.sh may trust a provenance verifier shipped by the same (potentially compromised) release payload it is verifying

**Files modified:** `scripts/install.sh`, `docs/install.md`, `scripts/install-smoke.sh`
**Commit:** `0ab88a9`
**Applied fix:** Inverted the `topos-provenance` verifier resolution order in `scripts/install.sh` so a previously-installed (`$BIN_DIR/topos-provenance`) or `PATH`-resolved verifier is tried before the staged payload's own copy, which is now the last resort — narrowing the window where a fully-compromised release payload could supply its own (fake, always-succeeding) verifier. Added a "Bootstrap-trust caveat" subsection to `docs/install.md`'s "Provenance verification" section documenting this residual risk explicitly, per the review's "at minimum, document this" fallback ask, in addition to implementing the stronger reordering fix. Updated a stale comment in `scripts/install-smoke.sh` that referenced the old "tier 1" staged-CLI position.

Verified with `./scripts/install-smoke.sh` (all cases pass, including the two provenance-specific cases exercising the staged-CLI fallback and the tamper-refusal path) and `bash -n` syntax checks on both modified shell scripts.

### WR-02: `topos-provenance keygen --key-id` and `ParseProvenanceKeys`/`FormatProvenanceKeys` have no charset validation

**Files modified:** `cmd/topos-provenance/main.go`, `cmd/topos-provenance/main_test.go` (new), `kernel/pluginhost/provenance.go`, `kernel/pluginhost/provenance_test.go`
**Commit:** `b3a2c0b`
**Applied fix:** Added an exported `pluginhost.ValidateProvenanceKeyID` function (restricted to `^[A-Za-z0-9._-]+$`, explicitly excluding `,` and `=`) and wired it in at three points: `runKeygen` and `runSign` in `cmd/topos-provenance/main.go` (per the review's core fix — validate at the point the id is chosen, matching the review's "ideally in runSign... too" note), and inside `ParseProvenanceKeys` itself (defense in depth, replacing its prior bare "empty id" check) so the restriction holds regardless of entry point. Added regression tests: `TestValidateProvenanceKeyID_AcceptsAndRejects` and an extended charset case in `TestParseProvenanceKeys_MalformedSegmentsAreRejected` in `kernel/pluginhost/provenance_test.go`, plus a new `cmd/topos-provenance/main_test.go` exercising `runKeygen`/`runSign` rejecting comma/equals/space key ids with a clear, subcommand-prefixed error (and a positive control proving a valid key id still works end to end).

Verified with `go test ./kernel/pluginhost/... ./cmd/topos-provenance/... -count=1`, `go build ./...`, `go vet ./...`, and `./scripts/provenance-smoke.sh` (exercises `topos-provenance keygen --key-id` end to end) — all pass.

## Skipped Issues

None — all in-scope findings were fixed. (IN-01 is Info-tier and out of `fix_scope=critical_warning`; not attempted.)

## Verification summary

- `go build ./...` — clean
- `go vet ./...` — clean
- `go test ./... -count=1` (root module) — all pass
- `make test-portable` — all pass (root + all 8 cgo-free workspace modules)
- `make test-signal` (cgo Signal plugin) — pass
- `./scripts/install-smoke.sh` — all cases pass
- `./scripts/provenance-smoke.sh` — pass

---

_Fixed: 2026-08-20T19:40:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
