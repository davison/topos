---
phase: 11-external-plugins-the-trust-boundary
verified: 2026-08-13T13:27:15Z
status: gaps_found
score: 4/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
gaps:
  - truth: "Trust is decided by the kernel from where the binary lives plus a content hash pinned when the source was added; swapping the binary afterwards is caught at the next launch and fails loudly by name instead of inheriting stale trust (ROADMAP success criterion 3)."
    status: failed
    reason: "The pin-mismatch/swap-detection mechanism itself works and is proven end to end (kernel/supervisor/externalproof_test.go, web/e2e/specs/11-binary-changed-repin.spec.ts) — but the criterion's own premise, 'trust is decided by the kernel from where the binary lives,' is false as implemented. pluginhost.ResolveBinary (kernel/pluginhost/discover_binaries.go:317-343) joins the caller-supplied plugin name onto dirs.Trusted/dirs.External with filepath.Join and never checks the resolved path stays inside that directory, and config.Validate (kernel/config/config.go) never validates Source.Plugin's shape or presence at all. A `[sources.<id>].plugin` value containing `../` segments, written through PUT /api/config (kernel/httpapi/config.go's ConfigSaveHandler -> config.Store.Save -> dryRunExpand -> Config.Validate — no membership or shape check anywhere in that path; only POST /api/config/describe-plugin's DescribePluginHandler checks against DiscoverAllTiered), can resolve to a file outside both configured directories and be misclassified TierTrusted, skipping the pin-verification gate entirely (launch's pin check is gated on tier == TierExternal). This is exactly the scenario 11-01-PLAN.md's own must_haves.prohibitions declares 'kept': 'The external tier MUST NOT be reachable by a caller-supplied path — directory listing across the two configured directories stays the only authority over what may be launched (T-07-09, extended to two tiers).' It is not kept. Confirmed by direct code reading during this verification (matches 11-REVIEW.md's CR-01, Critical severity, still present and unpatched at verification time — no commit since the review landed touches discover_binaries.go or config.go's Validate)."
    artifacts:
      - path: "kernel/pluginhost/discover_binaries.go"
        issue: "ResolveBinary (lines 317-343) performs no name-shape/confinement check before filepath.Join+os.Stat, and no regular-file check (accepts a directory or device file) — unlike DiscoverAllBinaries' isRegularFileFollowingSymlinks discipline"
      - path: "kernel/config/config.go"
        issue: "Validate iterates cfg.Sources and checks base_url/token/path/sync_interval, but never checks src.Plugin for emptiness or shape (no bare-filename regex) — confirmed by reading the full Validate loop, which contains no reference to src.Plugin at all"
    missing:
      - "Reject any Source.Plugin value that is not a bare filename (name == filepath.Base(name), no '..' segment, non-empty) inside config.Validate, naming the offending source"
      - "Add the identical confinement + os.Stat-regular-file check as defense-in-depth directly inside ResolveBinary before either directory's os.Stat call (the review's suggested fix mirrors isRegularFileFollowingSymlinks' IsRegular() discipline)"
      - "A regression test exercising a `../`-containing (and an absolute-path, and an empty) Source.Plugin value through both config.Validate and ResolveBinary, asserting rejection by name in both places — no such adversarial case exists today in tier_test.go, discover_binaries_test.go or config_test.go (confirmed by grep during this verification)"
deferred: []
human_verification: []
---

# Phase 11: External Plugins — the Trust Boundary Verification Report

**Phase Goal:** The user can install and run plugin binaries that topos did not build, and always knows which of their sources come from code the project can't vouch for.
**Verified:** 2026-08-13T13:27:15Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | User drops a plugin binary into the external directory and it appears in the install catalog, configurable and launchable/syncing like any other source. | VERIFIED | `kernel/pluginhost.DiscoverAllTiered`/`DiscoverTiered` (kernel/pluginhost/discover_binaries.go); `GET /api/config/plugin-types` publishes `plugin_type_tiers`; e2e `web/e2e/specs/11-external-tier-badge.spec.ts` (3/3 tests pass, re-run live during this verification) boots a real kernel with a binary present only in the external directory and asserts it launches, syncs and reports `tier: "external"`. |
| 2 | Adding a source from an external plugin shows an explicit warning before confirmation; the resulting source carries a persistent untrusted badge everywhere it appears (picker, chip). | VERIFIED | `AddSourceModal.svelte`'s `'untrusted-confirm'` step (type-to-confirm gate); `TrustBadge.svelte` at chip and picker scale; `web/e2e/specs/11-untrusted-add.spec.ts` (1/1 pass, re-run live) drives the full picker→confirm→save→chip journey through a real browser against the real out-of-repo `topos-plugin-external-demo` binary and asserts the badge, the label, the disabled/enabled confirm gate, and the persisted pin. |
| 3 | Trust is decided by the kernel from where the binary lives plus a content hash pinned when the source was added; swapping the binary is caught at the next launch and fails loudly by name instead of inheriting stale trust. | **FAILED** | The swap-detection half is proven (`kernel/supervisor/pinmismatch_test.go`, `kernel/supervisor/externalproof_test.go`, `web/e2e/specs/11-binary-changed-repin.spec.ts` all pass). The "decided ... from where the binary lives" half is false as implemented: `pluginhost.ResolveBinary` performs no path-confinement check, and `config.Validate` performs no shape/presence check on `Source.Plugin` — a `../`-containing value written through `PUT /api/config` can resolve outside both configured directories and be misclassified `TierTrusted`, skipping pin verification entirely. Confirmed present in the current codebase (11-REVIEW.md CR-01, Critical, unpatched). See Gaps Summary. |
| 4 | An external plugin receives provider-specific config keys the kernel has never heard of, with no kernel change required to add a new key. | VERIFIED | `sourceConfigEnvelope.Extras` (kernel/pluginhost/host.go); `[sources.<id>.extras]` round-trips through `WriteCanonical`; `kernel/supervisor/externalproof_test.go`'s `TestExternalProof_OutOfRepoBinaryEndToEnd` (re-run live, pass) asserts an index item exists for each configured extras key on the real out-of-repo proof binary, including a `${VAR}`-expanded value. |
| 5 | A real binary built outside the in-repo plugin set is discovered, marked untrusted, and synced end to end — proven before out-of-repo source work starts. | VERIFIED | `testdata/external-plugin/` (module path `example.com/acme/topos-plugin-external-demo`, outside `github.com/davison/topos`, outside `plugins/`, outside `internal/audit`'s scan scope — `go test ./internal/audit/...` passes); `TestExternalProof_OutOfRepoBinaryEndToEnd` (re-run live, pass) covers discovery, tier, extras passthrough, environment scrubbing and pin refusal against this real binary. |

**Score:** 4/5 truths verified, 1 failed (behavior_unverified: 0)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `kernel/pluginhost/discover_binaries.go` (Tier, Dirs, TieredBinary, ResolveBinary, DiscoverTiered, DiscoverAllTiered) | Two-tier discovery + resolution authority | VERIFIED (exists, substantive, wired) but **the resolution authority itself has an unconfined-path defect** — see gap above | `go build ./...` clean; `TestResolveBinary_*` pass but cover only well-formed names, no adversarial/traversal case |
| `kernel/config/types.go` (PluginsConfig.ExternalDir/Pins, Source.Extras) | Declared config surface | VERIFIED | present with toml+json tags |
| `kernel/pluginhost/binaryhash.go` (HashBinary) | SHA-256 content hash | VERIFIED | `TestHashBinary_*` pass |
| `kernel/pluginhost/host.go` (ErrPinMismatch, LaunchFailure, Host.LaunchFailures, allowedEnv, sourceConfigEnvelope) | Pre-exec pin gate + soft failure + env allowlist + extras envelope | VERIFIED | `TestLaunch_Pin_*`, `TestAllowedEnv` pass |
| `proto/topos/v1/plugin.proto` (ExtrasField, DescribeResponse.extras field 7) | Additive contract extension | VERIFIED | `go test ./sdk/ -run TestContract` passes; field 7, fields 1-6 unchanged |
| `kernel/httpapi/sources.go` (launch-failure merge, sourceStatus fields) | Trust facts published to browser | VERIFIED | `go test ./kernel/httpapi/...` passes |
| `web/src/lib/components/TrustBadge.svelte` | Reusable trust badge | VERIFIED | exists, wired into SourceChip and AddSourceModal picker rows |
| `web/src/lib/components/TrustUpdateDialog.svelte` | Re-pin confirmation | VERIFIED | exists, wired via `+page.svelte`'s `trustUpdateInstance` slot |
| `testdata/external-plugin/` (standalone module) | Out-of-repo proof binary | VERIFIED | own go.mod (`example.com/acme/...`), own Makefile target (`external-demo`), builds to `bin/plugins-external/` |
| `docs/plugin-contract.md`, `docs/api.md`, `config.example.toml` | Republished contract/docs | VERIFIED | `make docs-check` implied clean (grep confirms Trust tiers/Pinning/launch-environment sections, `tier`/`pinned_hash`/`launch_failure`/`plugin_type_tiers` documented, `external_dir`/`[plugins.pins]`/`extras` example in config.example.toml) |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| cmd/topos dirs -> supervisor -> pluginhost.launch -> SourceHealth.Tier -> GET /api/sources -> SourceChip TrustBadge | WIRED | verified by passing e2e `11-external-tier-badge.spec.ts` |
| `[plugins.pins]` -> raw config -> launch's pre-exec SHA-256 compare -> ErrPinMismatch -> Host.LaunchFailures -> GET /api/sources | WIRED | verified by passing `pinmismatch_test.go`, `11-binary-changed-repin.spec.ts` |
| `PUT /api/config` (Source.Plugin value) -> `Config.Validate` -> `Supervisor.Apply` -> `Host.Reconcile` -> `launch` -> `ResolveBinary` | **WIRED BUT UNGUARDED** | this is the CR-01 path: the link exists and functions for well-formed values, but neither endpoint of the chain validates or confines the value in transit — see gap above |
| describePlugin response (tier, binary_hash, env_var_names, extras) -> confirm interstitial -> submitMatch's single putConfig | WIRED | verified by passing `11-untrusted-add.spec.ts` |
| GET /api/sources launch_failure + current_hash -> SourceChip mismatch flag -> TrustUpdateDialog -> setPluginPin + putConfig -> supervisor Apply relaunches | WIRED | verified by passing `11-binary-changed-repin.spec.ts` |

### Behavioral Spot-Checks / Test Execution (live, re-run during this verification)

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full Go build | `CGO_ENABLED=0 go build ./...` | exit 0 | PASS |
| Go test suite (phase-11 packages) | `go test ./kernel/pluginhost/... ./kernel/supervisor/... ./kernel/httpapi/... ./kernel/config/... ./cmd/topos/... ./sdk/...` | all `ok` | PASS |
| `ResolveBinary` unit tests | `go test ./kernel/pluginhost/ -run TestResolveBinary -v` | 6/6 pass, **no traversal/confinement case present** | PASS (incomplete coverage) |
| Pin verification unit tests | `go test ./kernel/pluginhost/ -run TestLaunch_Pin -v` | 5/5 pass | PASS |
| External-tier supervisor boot | `go test ./kernel/supervisor/ -run TestExternalTier -v` | 2/2 pass | PASS |
| Boot-time pin-mismatch soft-failure (checkpoint-fix regression) | `go test ./kernel/supervisor/ -run TestPinMismatch -v` | 3/3 pass | PASS |
| Out-of-repo proof end-to-end | `go test ./kernel/supervisor/ -run TestExternalProof -v` | 1/1 pass | PASS |
| Frontend type-check | `npm --prefix web run check` | 0 errors, 10 pre-existing unrelated warnings | PASS |
| Frontend unit tests (phase 11) | `npm --prefix web run test -- trust-badge.test.ts untrusted-add.test.ts extras-form.test.ts repin.test.ts` | 126/126 pass | PASS |
| e2e: two-tier discovery + badge | `make e2e E2E_ARGS=specs/11-external-tier-badge.spec.ts` | 3/3 pass | PASS |
| e2e: untrusted add + extras passthrough | `make e2e E2E_ARGS=specs/11-untrusted-add.spec.ts` | 1/1 pass | PASS |
| e2e: binary swap / catch / re-pin / recover | `make e2e E2E_ARGS=specs/11-binary-changed-repin.spec.ts` | 1/1 pass | PASS |

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|-----------------|-------------|--------|----------|
| PLUG-06 | 11-01, 11-04 | Kernel discovers/launches plugin binaries from a configured external directory, distinct from trusted | SATISFIED | two-tier discovery + real out-of-repo proof binary, both proven live |
| PLUG-07 | 11-02, 11-03, 11-06 | Kernel derives trusted/untrusted status from provenance (directory tier, content hash pinned at add-time, re-verified at every launch) — never from anything the plugin declares | **BLOCKED** | Pin re-verification is real and proven, but "derives ... from provenance (directory tier)" is not reliably true — `ResolveBinary`'s unconfined path join means tier can be a caller-supplied path's accident rather than genuine directory provenance (CR-01) |
| PLUG-08 | 11-01, 11-03, 11-05, 11-06 | Explicit warning before adding an untrusted source; persistent untrusted badge everywhere the source appears | SATISFIED | confirm interstitial, chip badge, picker badge/label, all proven live via passing e2e specs |
| PLUG-09 | 11-02, 11-03, 11-04, 11-05 | Plugin host passes arbitrary per-instance config keys through to plugins with no kernel change | SATISFIED | extras envelope, UI extras editor, real out-of-repo binary observably receiving and echoing extras keys |

No orphaned requirements — REQUIREMENTS.md maps exactly PLUG-06/07/08/09 to Phase 11 and all four are claimed across the six plans.

### Anti-Patterns Found

No `TODO`/`FIXME`/`XXX`/`TBD`/placeholder markers found in the phase's key artifact files (`TrustBadge.svelte`, `TrustUpdateDialog.svelte`, `SourceChip.svelte`, `AddSourceModal.svelte`, `discover_binaries.go`, `host.go`, `config.go`). No stub/empty-implementation patterns found in the same set.

The one substantive finding is CR-01, already detailed above and in `11-REVIEW.md` — this is a missing-validation defect, not a stub or placeholder, but it is a security-relevant gap directly contradicting a `status: kept` prohibition in `11-01-PLAN.md`'s own must_haves frontmatter.

### Human Verification Required

None outstanding. Phase 11's single human checkpoint (11-06 Task 4, all six UI elements E1-E6) was already run and approved during execution (two rounds — round 1 found a boot-time defect that round 2's fix and an independent live re-verification closed), per `11-06-SUMMARY.md`'s `coverage[D6]` entry (`human_judgment: true`, approved).

### Gaps Summary

Five of the five ROADMAP success criteria are functionally implemented and independently re-proven live during this verification (build, full relevant Go test packages, targeted frontend tests, and all three phase-11 Playwright specs all pass). The mechanism for detecting a swapped binary — the headline feature of success criterion 3 — genuinely works and is proven end to end against a real out-of-repo binary, including the boot-time soft-failure parity fix that a checkpoint round caught.

However, criterion 3's own premise — "Trust is decided by the kernel from where the binary lives" — does not hold universally: `pluginhost.ResolveBinary` (the single documented authority the whole trust-tier system is built on, per its own doc comment) performs `filepath.Join(dir, name)` + `os.Stat` with no check that the result stays inside `dir`, and `config.Validate` never validates `Source.Plugin`'s shape at all. A `Source.Plugin` value containing `../` segments, saved through `PUT /api/config` (which performs no membership or shape check on this field anywhere in its validate/save/apply path), can resolve to an arbitrary file outside both configured plugin directories, get misclassified `TierTrusted`, and skip pin verification and exec.Command's usual regular-file assumptions entirely. This is the exact scenario `11-01-PLAN.md`'s own must-have prohibition promises is prevented ("The external tier MUST NOT be reachable by a caller-supplied path... T-07-09") — the promise is not kept in the current codebase. It matches `11-REVIEW.md`'s CR-01 (Critical) precisely, and remains unpatched as of this verification (no commit since the review's timestamp touches either file).

This does not appear intentional or covered by a later phase — neither Phase 12 (Filesystem Source) nor Phase 14 (Google Drive) revisits `ResolveBinary` or `config.Validate`; both simply depend on Phase 11's trust mechanism being sound.

**Recommended fix** (mirrors the review's own suggested patch): add a bare-filename shape check (`name == filepath.Base(name)`, no `..`, non-empty) to `config.Validate` for every `Source.Plugin`, and add the identical confinement check plus an `IsRegular()` check directly inside `ResolveBinary` before either directory's `os.Stat` — with a regression test exercising a `../`-containing value through both. This is a small, well-scoped fix; it does not require replanning the phase's UI or wire-contract work, only closing the one unvalidated input path.

---

_Verified: 2026-08-13T13:27:15Z_
_Verifier: Claude (gsd-verifier)_
