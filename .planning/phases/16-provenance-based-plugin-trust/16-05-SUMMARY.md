---
phase: 16-provenance-based-plugin-trust
plan: 05
subsystem: security
tags: [ed25519, plugin-trust, provenance, install, e2e, documentation]

# Dependency graph
requires:
  - phase: 16-02
    provides: "EvaluateTrust/resolveBinaryDetailed as the pure provenance-driven tier authority — Dirs.Trusted/Dirs.External as pure search paths"
  - phase: 16-04
    provides: "The real davison/topos-plugins signing pipeline, the topos-plugins-2026a embedded key, and TRUST-02's throwaway-prefix proof this plan's install-time verification now covers on the real install surface"
provides:
  - "D-09's install-time arm: scripts/install.sh verifies provenance inside its existing verify stage, before any placement — a provenance-free release (every one published to date) is an unchanged no-op; a validly-signed release verifies and installs; a binary altered after signing aborts naming it with $PREFIX left byte-unchanged"
  - "The e2e harness signs its own fixture manifests through the real topos-provenance CLI, via the link-time provenanceKeysExtra key seam make e2e injects (D-12) — never a runtime-readable trust input"
  - "web/e2e/specs/16-signed-provenance-tier.spec.ts: the browser proof that a signed binary in the EXTERNAL plugin directory launches trusted with no badge, and the identical binary with its signature removed resolves the untrusted path instead"
  - "docs/plugin-trust.md: the one canonical statement of the trust model, cross-linked from docs/plugin-contract.md, docs/api.md, and docs/install.md"
affects: [17]

# Actuals (#2632)
actuals:
  tokens: 16419
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Install-time provenance verification as a second, independent check layered on top of SHA-256 transport-integrity verification — same verify stage, same before-any-placement discipline, resolved via a three-tier verifier lookup (staged payload -> $PREFIX/bin -> PATH) that never silently skips evidence it cannot check"
    - "One signed manifest per binary in the e2e fixture harness (never a shared batch manifest) — the seam that lets a single kernel boot carry both the positive (valid signature) and negative (signature removed) provenance cases side by side, since Playwright forbids varying a worker-scoped fixture option via test.use() inside a describe block"

key-files:
  created:
    - docs/plugin-trust.md
    - web/e2e/specs/16-signed-provenance-tier.spec.ts
  modified:
    - scripts/install.sh
    - scripts/install-smoke.sh
    - scripts/smoke-lib.sh
    - Makefile
    - web/e2e/fixtures/plugin-binaries.ts
    - web/e2e/fixtures/config-builder.ts
    - web/e2e/fixtures/kernel.ts
    - docs/plugin-contract.md
    - docs/api.md
    - docs/install.md
    - docs/testing.md
    - kernel/pluginhost/binaryhash.go

key-decisions:
  - "Extended scripts/smoke-lib.sh (not in Task 1's declared <files> list) with smoke_build_fixture_release_with_provenance — the shared fixture-release builder file's own stated purpose is hosting exactly this kind of helper for install-smoke.sh, and duplicating the existing smoke_build_fixture_release pattern inline would have re-created the same one-place-only drift risk the file already exists to prevent."
  - "Redesigned signedProvenanceBinaries from the plan's own sketch (string[], one shared manifest) to {name, removeSignature?}[] with ONE independent sign() call per entry — discovered live that Playwright refuses test.use({configSpec}) inside a test.describe() block for a worker-scoped fixture option ('Cannot use({ configSpec }) in a describe group, because it forces a new worker'), so the plan's literal 'second test in the same file with a different configSpec' could not run. Both the positive and negative provenance cases now share ONE kernel boot, proven via two independently-signed renamed copies of the same real binary (topos-plugin-mockstrict) rather than two separate kernels."
  - "cmd/topos-provenance sign's --tag default in the e2e fixture helper (signProvenanceFixture) is derived from the signed binary name(s) rather than a fixed literal, so two independent sign() calls into the SAME external directory never collide on the manifest's own <repo>-<tag> basename."

patterns-established:
  - "Per-binary independent provenance manifests in e2e fixtures: signProvenanceFixture is called once per binary that needs its own revocable evidence, never batched, so a future spec proving a THIRD state can add a third signed entry without touching the first two."

requirements-completed: [TRUST-01, TRUST-02, TRUST-03]

coverage:
  - id: D1
    description: "Install-time provenance verification (D-09's second arm): scripts/install.sh verifies signed provenance inside its existing verify stage, before any placement; a provenance-free release (every real release published to date) is an unchanged no-op; a validly-signed release verifies and installs; a binary altered after signing (with checksums.txt regenerated over the tampered bytes, so only provenance — not SHA-256 — catches it) aborts naming the binary with the target prefix byte-identical before and after"
    requirement: "TRUST-02"
    verification:
      - kind: integration
        ref: "scripts/install-smoke.sh#Case: provenance — no provenance files installs exactly as before"
        status: pass
      - kind: integration
        ref: "scripts/install-smoke.sh#Case: provenance — valid signed manifest verifies and installs"
        status: pass
      - kind: integration
        ref: "scripts/install-smoke.sh#Case: provenance — binary altered after signing aborts, naming it"
        status: pass
      - kind: other
        ref: "make install-check (full suite, all cases including the pre-existing ones)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The e2e harness signs its own fixture manifests through the real topos-provenance CLI via the link-time provenanceKeysExtra seam (Makefile), and the browser proves a signed binary in the EXTERNAL plugin directory launches trusted with no badge while the identical binary with its signature removed resolves the untrusted path — location-independent trust, visible to the operator"
    requirement: "TRUST-01"
    verification:
      - kind: e2e
        ref: "web/e2e/specs/16-signed-provenance-tier.spec.ts#a validly signed external-directory binary renders healthy and trusted (no untrusted badge, no re-pin remedy), and its items sync with no pin recorded"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/16-signed-provenance-tier.spec.ts#the same binary, signed into the same directory, with its signature removed before boot, resolves untrusted: badge, no launch, no synced items"
        status: pass
      - kind: e2e
        ref: "make e2e (full suite: 148 passed, 5 skipped, 0 failed)"
        status: pass
    human_judgment: false
  - id: D3
    description: "docs/plugin-trust.md states the trust model once — the two evidence sources, the on-disk manifest/signature format, what earns trust and what does not, key rotation, why the external tier is not second-class, and the topos-provenance verify invocation — and every document/comment that previously described the superseded model now agrees with it or links to it"
    verification:
      - kind: other
        ref: "make docs-check"
        status: pass
      - kind: other
        ref: "grep-based acceptance criteria: plugin-trust.md line count, boundary-sentence preservation, cross-links from plugin-contract.md/install.md/testing.md, superseded-claim removal from binaryhash.go and plugin-contract.md, manifest_unverified/pin_mismatch vocabulary survival in api.md, 16-signed-provenance-tier indexed in testing.md — all pass"
        status: pass
    human_judgment: false

# Metrics
duration: 55min
completed: 2026-08-20
status: complete
---

# Phase 16 Plan 5: Provenance-Based Plugin Trust — Install-Time Verification, Location-Independence Proof, and Trust Model Docs Summary

**`scripts/install.sh` now verifies signed provenance before placement (D-09's install-time arm), the e2e harness signs its own fixtures through the real `topos-provenance` CLI and a browser spec proves location-independent trust, and `docs/plugin-trust.md` becomes the one canonical statement of the model — closing Phase 16.**

## Performance

- **Duration:** 55 min
- **Started:** 2026-08-20T09:10:00+01:00 (approx.)
- **Completed:** 2026-08-20T10:05:00+01:00 (approx.)
- **Tasks:** 3
- **Files modified:** 14 (2 created, 12 modified)

## Accomplishments

- `scripts/install.sh`: a provenance-verification step runs inside the existing verify stage, immediately after `sha256sum -c` and before any placement — a payload carrying at least one `*.provenance.json` is verified via `topos-provenance verify` (resolved from the staged payload, then `$PREFIX/bin`, then `PATH`, in that order); a payload with no provenance files at all is a documented no-op, preserving today's install behavior exactly (TRUST-03). The path allowlist now also accepts `*.provenance.json`/`.sig` and an optional top-level `topos-provenance` asset.
- `scripts/install-smoke.sh` + `scripts/smoke-lib.sh`: three new cases pin the no-provenance no-op, the valid-signed-manifest happy path, and a binary altered *after* signing (with `checksums.txt` regenerated over the tampered bytes, so the pre-existing SHA-256 pass alone would succeed) aborting with the binary named and the target prefix byte-identical before/after. `smoke_build_fixture_release_with_provenance` builds this fixture by mirroring `scripts/provenance-smoke.sh`'s own keygen → relink → sign sequence, so this is also an executable proof the link-time `provenanceKeysExtra` seam works.
- `Makefile`: the `e2e` and `gdrive-external-rehearsal` targets build `bin/topos-provenance`, generate an ephemeral `e2e-fixture` signing keypair into `bin/` (gitignored), and inject the public half into the kernel build via a second `-X` (`PROVENANCE_LDFLAGS_VAR`, written in exactly one place, mirroring `MANIFEST_LDFLAGS_VAR`) alongside the existing link-time manifest one.
- `web/e2e/fixtures/{plugin-binaries,config-builder,kernel}.ts`: `signProvenanceFixture` signs a fixture manifest by executing the real `topos-provenance` CLI (never reimplementing the format); `signedProvenanceBinaries` on `FixtureConfigSpec` signs each named external-directory binary into its own independent manifest, with no `[plugins.pins]` entry, so trust comes from provenance alone.
- `web/e2e/specs/16-signed-provenance-tier.spec.ts` (new): proves a validly signed binary in the external directory launches trusted with no badge and no re-pin remedy, its items sync, and no pin was recorded — then proves the identical binary type, signed independently and with its own signature removed before boot, resolves the untrusted path (badge, no launch, no items).
- `docs/plugin-trust.md` (new, 123 lines): the canonical trust-model statement — the two evidence sources (D-10), the on-disk format, what earns trust and what does not, key rotation (D-03), why the external tier is not second-class, and the `topos-provenance verify` hand-verification invocation. `docs/plugin-contract.md`, `docs/api.md`, `docs/install.md`, and `docs/testing.md` are corrected/extended and cross-link to it; `kernel/pluginhost/binaryhash.go`'s stale "no signature or publisher verification anywhere in this design" doc comment is corrected.

## Task Commits

Each task was committed atomically:

1. **Task 1: Verify provenance before placement in the installer** — `3e23d5f` (feat)
2. **Task 2: Sign e2e fixtures through the link-time key seam and prove location-independent trust in the browser** — `303d6dc` (feat)
3. **Task 3: State the trust model once, and update the docs that reference it** — `1a9ac7f` (docs)

**Plan metadata:** commit pending (this SUMMARY, parallel/worktree mode — STATE.md/ROADMAP.md are updated centrally by the orchestrator)

## Files Created/Modified

- `scripts/install.sh` — provenance verification wired into the existing verify stage, before placement
- `scripts/install-smoke.sh` — three new provenance smoke cases
- `scripts/smoke-lib.sh` — `smoke_build_fixture_release_with_provenance`
- `Makefile` — `PROVENANCE_LDFLAGS_VAR`/`E2E_PROVENANCE_KEY_ID`, e2e/gdrive-external-rehearsal key injection
- `web/e2e/fixtures/plugin-binaries.ts` — `PROVENANCE_BIN`, `PROVENANCE_KEY_FILE`, `signProvenanceFixture`
- `web/e2e/fixtures/config-builder.ts` — `signedProvenanceBinaries` field, pin-skip logic
- `web/e2e/fixtures/kernel.ts` — signs each entry after linking, before kernel spawn
- `web/e2e/specs/16-signed-provenance-tier.spec.ts` — the location-independence browser proof
- `docs/plugin-trust.md` — the canonical trust-model statement (new)
- `docs/plugin-contract.md` — corrected integrity-vs-authentication paragraph, links to plugin-trust.md
- `docs/api.md` — `manifest_unverified` documented as covering both evidence arms
- `docs/install.md` — new "Provenance verification" section, extended troubleshooting entry
- `docs/testing.md` — `16-signed-provenance-tier.spec.ts` indexed, install-check gate description extended
- `kernel/pluginhost/binaryhash.go` — corrected doc comment

## Decisions Made

- Extended `scripts/smoke-lib.sh` beyond Task 1's declared file list — the shared fixture-builder file's own purpose is hosting exactly this kind of helper; inlining it into `install-smoke.sh` would have duplicated the existing `smoke_build_fixture_release` pattern.
- Redesigned `signedProvenanceBinaries` from a flat `string[]` to `{name, removeSignature?}[]` with one independent `sign()` call per entry, after discovering live that Playwright forbids `test.use({configSpec})` inside a `test.describe()` block for a worker-scoped option. Both the positive and negative provenance cases now share one kernel boot via two independently-signed renamed copies of `topos-plugin-mockstrict`.
- `signProvenanceFixture`'s default `--tag` is derived from the signed binary name(s) so two independent signing calls into the same external directory never collide on the manifest's `<repo>-<tag>` basename.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `scripts/smoke-lib.sh` extended beyond Task 1's declared file list**
- **Found during:** Task 1, designing the three new install-smoke.sh provenance cases
- **Issue:** The plan's Task 1 `<files>` list names only `scripts/install.sh, scripts/install-smoke.sh`. Building a signed-provenance fixture release requires the same keygen → relink → sign sequence `scripts/provenance-smoke.sh` already uses, and the existing shared fixture builder (`smoke_build_fixture_release`) already lives in `scripts/smoke-lib.sh` specifically to avoid duplicating that logic per-caller.
- **Fix:** Added `smoke_build_fixture_release_with_provenance` to `scripts/smoke-lib.sh`, following the existing function's exact shape and doc-comment style.
- **Files modified:** `scripts/smoke-lib.sh`
- **Verification:** `make install-check` (all cases, including the three new ones) passes.
- **Committed in:** `3e23d5f` (Task 1 commit)

**2. [Rule 3 - Blocking] `signedProvenanceBinaries` redesigned from the plan's sketched `string[]` shape**
- **Found during:** Task 2, first `make e2e` run of the new spec
- **Issue:** The plan's Task 2 action text describes one spec file with "a second test... with the `.provenance.sig` file removed" implying a second kernel boot with a different `configSpec`. Playwright's real error: `Cannot use({ configSpec }) in a describe group, because it forces a new worker. Make it top-level in the test file or put in the configuration file.` — a worker-scoped fixture option cannot vary within one spec file via a nested `test.describe()`'s own `test.use()`.
- **Fix:** Redesigned `signedProvenanceBinaries` to `{name, removeSignature?}[]`, signing each entry independently (one manifest per binary, never batched) so one kernel boot carries both a validly-signed binary and a second, independently-signed-then-revoked binary side by side. Both tests in `16-signed-provenance-tier.spec.ts` now share one top-level `configSpec`.
- **Files modified:** `web/e2e/fixtures/config-builder.ts`, `web/e2e/fixtures/kernel.ts`, `web/e2e/fixtures/plugin-binaries.ts`, `web/e2e/specs/16-signed-provenance-tier.spec.ts`
- **Verification:** `make e2e E2E_ARGS="e2e/specs/16-signed-provenance-tier.spec.ts"` (2/2 pass) and the full `make e2e` (148 passed, 5 skipped, 0 failed).
- **Committed in:** `303d6dc` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 3 — blocking issues discovered while satisfying the plan's own required verification commands)
**Impact on plan:** Both were necessary to make the plan's own acceptance criteria (`make install-check`, `make e2e`) actually pass; neither changed the plan's scope or intent — the location-independence proof and the install-time verification behavior are exactly what was asked for, delivered through a shape Playwright's real constraints and this repo's existing conventions actually support.

## Issues Encountered

- `kernel/webui/build/.gitkeep` was repeatedly deleted from disk by the SvelteKit adapter-static build step (`npm run build`, invoked by every `make e2e`/`make build` run during this session) — pre-existing behavior of the build tooling, unrelated to this plan's own changes. Restored via `git checkout -- kernel/webui/build/.gitkeep` before each commit; not itself a code change.
- `docs/plugin-contract.md`'s "Trust tiers" section still describes some directory-derived-tier and shadow-rule language superseded by 16-02's D-11 rewrite (e.g. "Tier is derived exclusively from which directory a binary resolved from" and the shadow-rule paragraph). This plan's Task 3 action text scoped the correction narrowly to the "integrity control, not publisher authentication" paragraph and the stale-signature-claim removal — both done. The broader D-11 staleness in that section predates this plan (introduced by 16-02, not corrected there) and is out of this plan's declared scope; worth a follow-up documentation pass, not blocking this phase's own success criteria.

## User Setup Required

None — no external service configuration required. The e2e-only signing key is generated ephemerally by `make e2e` itself and is gitignored (`bin/` is wholesale-ignored).

## Next Phase Readiness

- D-09 is now fully satisfied: verification runs at install time (this plan) and at every launch (16-01/16-02).
- Success criterion 1 (location-independence) is proven both at the Go level (16-02's `TestResolveBinary_LocationSymmetric`) and now in the browser, for both trust arms and both directions (trusted, and — this plan — the signed-provenance arm specifically).
- The e2e harness's link-time key-injection seam (D-12) keeps "a dev kernel trusts its own local builds" open for Phase 17 without adding any runtime-readable trust input.
- `docs/plugin-trust.md` is the settled single source of truth Phase 17 can extend when the link-time arm is retired — that document's own "two evidence sources" section is written to shrink to one at that point.
- The `TRUST-02` requirement, shared with 16-04-PLAN.md per the shared-ID gate (#2388), is now ready to flip to complete: both plans declaring it have finished.
- `go build ./...`, `go vet ./kernel/... ./cmd/...`, `go test ./... -count=1`, `make test`, `make install-check`, `make provenance-check`, `make docs-check`, and `make e2e` (148 passed, 5 skipped, 0 failed) all pass locally.
- Phase 16 is complete: all five plans finished, all four success criteria met.

---
*Phase: 16-provenance-based-plugin-trust*
*Completed: 2026-08-20*

## Self-Check: PASSED

- `docs/plugin-trust.md` — FOUND
- `web/e2e/specs/16-signed-provenance-tier.spec.ts` — FOUND
- `scripts/install.sh`, `scripts/install-smoke.sh`, `scripts/smoke-lib.sh` — FOUND, modified
- `Makefile`, `web/e2e/fixtures/{plugin-binaries,config-builder,kernel}.ts` — FOUND, modified
- `docs/{plugin-contract,api,install,testing}.md`, `kernel/pluginhost/binaryhash.go` — FOUND, modified
- Commit `3e23d5f` (Task 1) — FOUND in `git log --oneline --all`
- Commit `303d6dc` (Task 2) — FOUND in `git log --oneline --all`
- Commit `1a9ac7f` (Task 3) — FOUND in `git log --oneline --all`
- All plan-level `<verification>` commands re-run and green: `make test`, `make install-check`, `make provenance-check`, `make docs-check`, `make e2e` (148 passed, 5 skipped, 0 failed), `go test ./... -count=1`
- `git ls-files | grep -c 'e2e-fixture\.key'` returns 0 — no key file tracked
- `git diff web/src | wc -l` returns 0 — no application source changed
