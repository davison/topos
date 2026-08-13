---
phase: 11-external-plugins-the-trust-boundary
verified: 2026-08-13T16:10:00Z
status: passed
score: 5/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 4/5
  gaps_closed:
    - "Trust is decided by the kernel from where the binary lives plus a content hash pinned when the source was added; swapping the binary afterwards is caught at the next launch and fails loudly by name instead of inheriting stale trust (ROADMAP success criterion 3)."
  gaps_remaining: []
  regressions: []
deferred: []
human_verification: []
---

# Phase 11: External Plugins — the Trust Boundary Verification Report

**Phase Goal:** The user can install and run plugin binaries that topos did not build, and always knows which of their sources come from code the project can't vouch for.
**Verified:** 2026-08-13T16:10:00Z
**Status:** passed
**Re-verification:** Yes — after gap closure (plan 11-07, CR-01)

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | User drops a plugin binary into the external directory and it appears in the install catalog, configurable and launchable/syncing like any other source. | ✓ VERIFIED | `pluginhost.DiscoverAllTiered`/`DiscoverTiered` (kernel/pluginhost/discover_binaries.go); `GET /api/config/plugin-types` publishes `plugin_type_tiers`; `web/e2e/specs/11-external-tier-badge.spec.ts` re-run live during this verification — 3/3 pass — boots a real kernel with a binary present only in the external directory and asserts it launches, syncs and reports `tier: "external"`. |
| 2 | Adding a source from an external plugin shows an explicit warning before confirmation; the resulting source carries a persistent untrusted badge everywhere it appears (picker, chip). | ✓ VERIFIED | `AddSourceModal.svelte`'s `'untrusted-confirm'` step (type-to-confirm gate); `TrustBadge.svelte` at chip and picker scale; `web/e2e/specs/11-untrusted-add.spec.ts` re-run live — 1/1 pass — drives the full picker→confirm→save→chip journey against the real out-of-repo `topos-plugin-external-demo` binary. |
| 3 | Trust is decided by the kernel from where the binary lives plus a content hash pinned when the source was added; swapping the binary is caught at the next launch and fails loudly by name instead of inheriting stale trust. | ✓ VERIFIED (gap closed) | **Confinement half (newly closed):** `pluginhost.validatePluginBinaryName` (kernel/pluginhost/discover_binaries.go:326) is `ResolveBinary`'s first statement, rejecting any name containing `/`/`\`, `.`/`..`, or `name != filepath.Base(name)` — before either directory is stat'd; all three `os.Stat` sites now require `info.Mode().IsRegular()`. `config.Validate` gained the hand-kept twin `validateSourcePlugins` (kernel/config/config.go:451), called from `Validate` at both `config.Load` and every `PUT /api/config` save. `host.go:785` is the sole production call site of `ResolveBinary` — confirmed by grep, no bypass exists. 13/13 `TestResolveBinary_*` pass (live re-run), 3/3 `TestValidate_SourcePlugin*` pass, 2/2 `TestConfigSaveHandler_{Traversal,Empty}Plugin*` pass (all re-run live, not read from SUMMARY). **Swap-detection half (previously proven, re-confirmed):** `kernel/supervisor/pinmismatch_test.go`, `kernel/supervisor/externalproof_test.go`, `web/e2e/specs/11-binary-changed-repin.spec.ts` all pass (live re-run, 1/1 e2e). |
| 4 | An external plugin receives provider-specific config keys the kernel has never heard of, with no kernel change required to add a new key. | ✓ VERIFIED | `sourceConfigEnvelope.Extras` (kernel/pluginhost/host.go); `[sources.<id>.extras]` round-trips through `WriteCanonical`; `TestExternalProof_OutOfRepoBinaryEndToEnd` (live re-run, pass) asserts extras passthrough including a `${VAR}`-expanded value against the real out-of-repo proof binary. |
| 5 | A real binary built outside the in-repo plugin set is discovered, marked untrusted, and synced end to end — proven before out-of-repo source work starts. | ✓ VERIFIED | `testdata/external-plugin/` (module `example.com/acme/topos-plugin-external-demo`, outside `github.com/davison/topos`, outside `plugins/`); `TestExternalProof_OutOfRepoBinaryEndToEnd` (live re-run, pass) covers discovery, tier, extras passthrough, environment scrubbing and pin refusal against this real binary. |

**Score:** 5/5 truths verified (0 present, behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `kernel/pluginhost/discover_binaries.go` (`validatePluginBinaryName`, `ResolveBinary` confinement + IsRegular guards) | Confined, launch-time trust authority | ✓ VERIFIED (exists, substantive, wired) | `go build ./...` clean; 13/13 `TestResolveBinary_*` pass including 7 new adversarial cases (traversal, absolute path, Windows separator, empty, `.`/`..`, directory-shadowing, symlink-still-resolves) |
| `kernel/config/config.go` (`validateSourcePlugins`, called from `Validate`) | Config-side confinement twin | ✓ VERIFIED | `grep` confirms single definition, single call site inside `Validate` before `validatePins`; 3/3 new tests pass |
| `kernel/pluginhost/tier_test.go`, `kernel/config/config_test.go`, `kernel/httpapi/config_test.go` | 12 new regression tests | ✓ VERIFIED | All 12 present and passing, live re-run (7 + 3 + 2) |
| `docs/plugin-contract.md` (bare-filename rule) | Published contract update | ✓ VERIFIED | `grep -n 'bare binary filename'` finds the new paragraph in the Trust tiers section; `make docs-check` exits 0 |
| `kernel/pluginhost/binaryhash.go`, `kernel/pluginhost/host.go` (pin gate, launch-failure, extras envelope) | Pre-exec pin gate + soft failure + extras | ✓ VERIFIED (unchanged since prior pass, regression-checked) | `TestHashBinary_*`, `TestLaunch_Pin_*`, `TestAllowedEnv` pass |
| `web/src/lib/components/TrustBadge.svelte`, `TrustUpdateDialog.svelte` | Reusable trust UI | ✓ VERIFIED (unchanged, regression-checked) | wired into `SourceChip`/`AddSourceModal`/`+page.svelte`; all 3 phase e2e specs pass live |
| `testdata/external-plugin/` | Out-of-repo proof binary | ✓ VERIFIED (unchanged, regression-checked) | own `go.mod`, own Makefile target, builds clean |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `PUT /api/config` (`Source.Plugin` value) → `Config.Validate` → `validateSourcePlugins` → 422 `config_invalid` naming the source, file on disk untouched | WIRED (newly closed) | `TestConfigSaveHandler_TraversalPluginValueReturns422AndLeavesFileUnchanged`, `TestConfigSaveHandler_EmptyPluginValueReturns422NamingTheSource` — both re-run live, pass; byte-identical-file assertion included |
| `supervisor.Apply` → `Host.Reconcile` → `launch` → `pluginhost.ResolveBinary` → `validatePluginBinaryName` + `IsRegular` gates → named error instead of an out-of-directory `TierTrusted` resolution | WIRED (newly closed) | `ResolveBinary`'s only production caller is `host.go:785` (confirmed by grep — no bypass); 7 new adversarial tests pass live |
| `cmd/topos` dirs → supervisor → `pluginhost.launch` → `SourceHealth.Tier` → `GET /api/sources` → `SourceChip` `TrustBadge` | WIRED (regression-checked) | `11-external-tier-badge.spec.ts` passes live |
| `[plugins.pins]` → raw config → launch's pre-exec SHA-256 compare → `ErrPinMismatch` → `Host.LaunchFailures` → `GET /api/sources` | WIRED (regression-checked) | `pinmismatch_test.go`, `11-binary-changed-repin.spec.ts` pass live |
| describePlugin response (tier, binary_hash, env_var_names, extras) → confirm interstitial → `submitMatch`'s single `putConfig` | WIRED (regression-checked) | `11-untrusted-add.spec.ts` passes live |
| `GET /api/sources` `launch_failure` + `current_hash` → `SourceChip` mismatch flag → `TrustUpdateDialog` → `setPluginPin` + `putConfig` → supervisor `Apply` relaunches | WIRED (regression-checked) | `11-binary-changed-repin.spec.ts` passes live |

### Behavioral Spot-Checks / Test Execution (all live, re-run during this verification — not read from SUMMARY.md)

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full Go build | `CGO_ENABLED=0 go build ./...` | exit 0 | PASS |
| `ResolveBinary` unit tests (incl. 7 new adversarial cases) | `go test ./kernel/pluginhost/ -run TestResolveBinary -v` | 13/13 pass | PASS |
| `config.Validate` source-plugin tests (3 new) | `go test ./kernel/config/... -run TestValidate_SourcePlugin -v` | 3/3 pass | PASS |
| HTTP-boundary tests (2 new + 5 pre-existing) | `go test ./kernel/httpapi/... -run TestConfigSaveHandler -v` | 7/7 pass | PASS |
| Phase-11 package regression sweep | `go test ./kernel/pluginhost/... ./kernel/supervisor/... ./kernel/httpapi/... ./kernel/config/... ./cmd/topos/... ./sdk/...` | all `ok` | PASS |
| Full portable gate (root + every workspace plugin module) | `make test-portable` | all `ok`, exit 0 | PASS |
| Docs link check | `make docs-check` | exit 0, 35/35 links resolve | PASS |
| e2e: two-tier discovery + badge | `make e2e E2E_ARGS=specs/11-external-tier-badge.spec.ts` | 3/3 pass | PASS |
| e2e: untrusted add + extras passthrough | `make e2e E2E_ARGS=specs/11-untrusted-add.spec.ts` | 1/1 pass | PASS |
| e2e: binary swap / catch / re-pin / recover | `make e2e E2E_ARGS=specs/11-binary-changed-repin.spec.ts` | 1/1 pass | PASS |
| Single production call site of `ResolveBinary` | `grep -rn "ResolveBinary(" kernel/ cmd/ \| grep -v _test.go` | exactly `host.go:785` | PASS (no bypass path) |

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|-----------------|-------------|--------|----------|
| PLUG-06 | 11-01, 11-04 | Kernel discovers/launches plugin binaries from a configured external directory, distinct from trusted | ✓ SATISFIED | two-tier discovery + real out-of-repo proof binary, both re-proven live |
| PLUG-07 | 11-02, 11-03, 11-06, 11-07 | Kernel derives trusted/untrusted status from provenance (directory tier, content hash pinned at add-time, re-verified at every launch) — never from anything the plugin declares | ✓ SATISFIED (gap closed by 11-07) | Confinement guard (`validatePluginBinaryName`) + regular-file gate now make tier genuinely provenance-derived; pin re-verification remains proven; 12 new regression tests plus live re-runs of all prior tests confirm no regression |
| PLUG-08 | 11-01, 11-03, 11-05, 11-06 | Explicit warning before adding an untrusted source; persistent untrusted badge everywhere the source appears | ✓ SATISFIED (regression-checked) | confirm interstitial, chip badge, picker badge/label, all re-proven live via passing e2e specs |
| PLUG-09 | 11-02, 11-03, 11-04, 11-05 | Plugin host passes arbitrary per-instance config keys through to plugins with no kernel change | ✓ SATISFIED (regression-checked) | extras envelope, UI extras editor, real out-of-repo binary observably receiving and echoing extras keys |

No orphaned requirements — REQUIREMENTS.md maps exactly PLUG-06/07/08/09 to Phase 11, and all four are claimed across the seven plans (11-01 through 11-07). REQUIREMENTS.md's checkbox/status column for PLUG-06/08/09 (`Pending`/`Gaps Found`) and PLUG-07's status text ("Gaps Found" despite its checkbox being ticked by the 11-07 executor) are stale tracking-file state — per this task's instructions, the orchestrator updates STATE.md/ROADMAP.md/REQUIREMENTS.md centrally after verification passes, so this is not counted as a gap.

### Anti-Patterns Found

No `TODO`/`FIXME`/`XXX`/`TBD`/placeholder markers found in the phase's key artifact files, including the newly modified `discover_binaries.go`, `config.go`, `tier_test.go`, `config_test.go`, `httpapi/config_test.go`, and `docs/plugin-contract.md`. No stub/empty-implementation patterns found. No fixture was weakened to make a test pass — `git show --stat` on all three 11-07 task commits shows additions only (238, 197, and further lines added; zero deletions in the fix commits).

Two pre-existing (non-blocking) **Warnings** surfaced by `11-REVIEW.md`'s re-review, confirmed present in the current codebase during this verification, neither of which touches the fixed confinement logic and neither of which is part of 11-07's declared scope:

| File | Pattern | Severity | Impact |
|------|---------|----------|--------|
| `config.example.toml` (5 occurrences, e.g. line 177) | Stale comment `# Validation: none at load time; a missing file fails at startup, by path.` on every `[sources.<name>] plugin` field, no longer accurate now that `validateSourcePlugins` rejects a malformed value at load/save time | ⚠️ Warning | Documentation-only; undersells the new load-time confinement guard to an operator or plugin author reading the example config, but has no functional or security effect (it doesn't grant permission the code doesn't actually reject) |
| `web/src/lib/components/TrustUpdateDialog.svelte:74` | `setPluginPin(config, source.plugin, source.current_hash ?? '')` silently falls back to an empty-string pin if `current_hash` is ever absent, rather than surfacing an error | ⚠️ Warning | Currently unreachable in practice (`launch()` always populates `current_hash` for every `TierExternal` pin-mismatch entry that opens this dialog), but is a latent defensive gap pre-dating 11-07 |

Both are recommended follow-up items (not phase blockers); confirmed still present by direct inspection during this verification, not merely carried over from the review report.

### Human Verification Required

None outstanding. Phase 11's single human checkpoint (11-06 Task 4, all six UI elements E1-E6) was already run and approved during execution, per `11-06-SUMMARY.md`'s `coverage[D6]` entry (`human_judgment: true`, approved). No new human-verification-worthy behavior was introduced by 11-07's gap closure (kernel-side input validation with a fully automatable HTTP/unit-test boundary, deliberately without a new Playwright spec per the plan's own documented rationale — confirmed reasonable: the three existing browser-drivable behaviors of this phase are unchanged and all pass live).

### Gaps Summary

None. The single gap the prior verification found — `pluginhost.ResolveBinary`'s missing path-confinement/regular-file check and `config.Validate`'s missing `Source.Plugin` shape check (CR-01) — is closed. This was independently re-derived from first principles during this verification, not read from SUMMARY.md claims:

- Read `validatePluginBinaryName` and the updated `ResolveBinary` directly; confirmed the guard runs as the function's first statement, before either directory is touched, and rejects empty/whitespace, `/`, `\`, `.`, `..`, and any non-`filepath.Base`-equal name.
- Read `validateSourcePlugins` directly; confirmed it is called from `Config.Validate` (which runs at both `config.Load` and every `PUT /api/config` save via `dryRunExpand`) and enforces the identical four rules, naming the offending source.
- Confirmed by `grep` that `host.go:785` is `ResolveBinary`'s only production call site — no code path can reach a plugin launch while bypassing the new guard.
- Re-ran all 12 new regression tests live (not from SUMMARY): 7 `TestResolveBinary_*`, 3 `TestValidate_SourcePlugin*`, 2 `TestConfigSaveHandler_*` — all pass.
- Re-ran the full portable test gate (`make test-portable`, root + 6 plugin modules) and `make docs-check` live — both exit 0.
- Re-ran all three Phase 11 Playwright e2e specs live (not trusted from SUMMARY) — 3+1+1 = 5/5 pass, including the symlink-dependent discovery spec that the new `IsRegular()` gate could plausibly have broken.
- Confirmed via `git show --stat` on the three 11-07 fix/test commits that no existing test, assertion, or fixture was weakened or deleted to make the new checks pass — only additions.

Two non-blocking documentation/defensive-code Warnings remain (see Anti-Patterns), recommended as follow-up but not gating this phase's goal achievement.

---

_Verified: 2026-08-13T16:10:00Z_
_Verifier: Claude (gsd-verifier)_
