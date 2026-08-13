---
phase: 11-external-plugins-the-trust-boundary
plan: 03
subsystem: api
tags: [go-plugin, trust-boundary, http-api, sha256, protobuf, documentation]

requires:
  - phase: 11-01
    provides: "kernel/pluginhost.Tier/Dirs/TieredBinary/ResolveBinary/DiscoverTiered/DiscoverAllTiered — the two-tier plugin discovery/launch-time provenance authority; GET /api/sources tier field and GET /api/config/plugin-types plugin_type_tiers"
  - phase: 11-02
    provides: "kernel/pluginhost.HashBinary/ErrPinMismatch/LaunchFailure/Host.LaunchFailures — pre-exec pin verification and the soft per-instance launch-failure channel; kernel/config.EnvRefNames; kernel/supervisor.Supervisor.LaunchFailures; topos.v1.ExtrasField/DescribeResponse.extras"
provides:
  - "kernel/httpapi.HealthProber widened with LaunchFailures() — *pluginhost.Host and *supervisor.Supervisor both satisfy it structurally via plan 11-02's delegation"
  - "kernel/httpapi.sourceStatus.PinnedHash/CurrentHash/LaunchFailure — GET /api/sources now carries a real, named entry for a source that never launched, plus the pin the browser needs to render for a healthy external source"
  - "kernel/pluginhost.Plugin.pinnedHash/binaryHash/extras and DescribeInfo.BinaryHash/Extras — the launch-time hash/extras facts the describe-plugin flow needed but plan 11-02 didn't yet expose"
  - "kernel/httpapi.describePluginResponse.Tier/BinaryHash/EnvVarNames/Extras — the add-source confirm interstitial's complete kernel-derived fact set, before anything is persisted or pinned"
  - "docs/plugin-contract.md's 'Trust tiers', 'Pinning' and 'The launch environment' sections, plus extras coverage in Configuration/Describe — the published third-party contract is coherent with Phase 11's shipped behavior again"
  - "docs/api.md and config.example.toml fully documenting every Phase 11 field and config key introduced across plans 11-01/11-02/11-03"
affects: [11-05, 11-06]

actuals:
  tokens: 16982
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "sourceStatusesFrom merges two independently-sourced sets (live probe results, soft launch failures) into one sorted response, with the probe result always winning a name collision — the same 'kernel-derived facts, client only renders' discipline extended from a single source to a merge of two"
    - "launch() computes an external-tier binary's SHA-256 unconditionally (both real and trial launches), reusing the one HashBinary call for both the pin-comparison (real launches) and the describe-plugin flow's reported binary_hash (trial launches) — never hashing the same file twice within one call"
    - "A plugin's Describe-declared extras are filtered (empty key dropped) once, at launch time inside pluginhost, so every consumer (DescribePluginType, a future real-launch consumer) sees an already-safe list rather than re-validating"

key-files:
  created: []
  modified:
    - kernel/httpapi/sources.go
    - kernel/httpapi/sources_test.go
    - kernel/httpapi/config.go
    - kernel/httpapi/config_test.go
    - kernel/pluginhost/host.go
    - kernel/pluginhost/describe_test.go
    - docs/plugin-contract.md
    - docs/api.md
    - config.example.toml

key-decisions:
  - "binaryHash is computed for EVERY TierExternal launch, not only trial (describeOnly) ones — the pin-comparison block already existed for real launches (plan 11-02); this plan widened it to run (hash-only, no comparison) on trial launches too, so DescribePluginType and a real launch share the exact same code path and hash computation rather than duplicating the HashBinary call in two places."
  - "PinnedHash flows onto SourceHealth (not just LaunchFailure) — a healthy external-tier source's GET /api/sources entry now also carries pinned_hash, sourced from the same launchPinnedHash the pin-check block already computes on a successful match, so the chip menu's pinned-hash footer works for a source that never failed a pin check, not only a broken one."
  - "extrasFieldResponse is a hand-written mirror struct (not a direct reuse of the generated toposv1.ExtrasField), matching this package's existing convention of never leaking a generated proto type across the HTTP JSON boundary — five fields, snake_case json tags, no proto import in the response shape."
  - "docs/plugin-contract.md's new 'Trust tiers'/'Pinning'/'The launch environment' sections are placed as top-level `##` sections directly after 'Discovery and launch' (before 'Configuration') — the reading order a third-party author needs: discover -> understand what launched it -> understand what environment it runs in -> understand its own config surface."

requirements-completed: [PLUG-07, PLUG-08, PLUG-09]

coverage:
  - id: D1
    description: "A source refused launch on a pin mismatch still appears in GET /api/sources as a real, named entry — one entry per instance whether it launched or failed, sorted by instance id, with the probe result winning any name collision"
    requirement: "PLUG-07"
    verification:
      - kind: unit
        ref: "kernel/httpapi/sources_test.go#TestSourcesHandler_MergesLaunchFailureIntoSourcesResponse"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/sources_test.go#TestSourcesHandler_ProbeResultWinsOverLaunchFailureForSameInstance"
        status: pass
      - kind: integration
        ref: "kernel/supervisor/pinmismatch_test.go#TestPinMismatch_BootSucceedsHealthySourceSyncsFailureRecorded (real supervisor boot proving LaunchFailures() records the mismatched instance by name while every other source boots/syncs normally)"
        status: pass
    human_judgment: false
  - id: D2
    description: "launch_failure is a closed-vocabulary, machine-readable field (empty or pin_mismatch), never gated on parsing last_error text; a launch-failed source's pinned_hash and current_hash are both populated; a healthy external-tier source also reports pinned_hash"
    requirement: "PLUG-08"
    verification:
      - kind: unit
        ref: "kernel/httpapi/sources_test.go#TestSourcesHandler_MergesLaunchFailureIntoSourcesResponse"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/sources_test.go#TestSourcesHandler_HealthyExternalSourceReportsPinnedHash"
        status: pass
    human_judgment: false
  - id: D3
    description: "The agent route's grant filter is unaffected by the launch-failure merge: an ungranted launch-failed source is structurally absent from GET /agent/v1/sources, and present with capabilities.read=true once granted"
    requirement: "PLUG-07"
    verification:
      - kind: unit
        ref: "kernel/httpapi/sources_test.go#TestAgentSourcesHandler_LaunchFailedSourceRespectsGrant"
        status: pass
    human_judgment: false
  - id: D4
    description: "POST /api/config/describe-plugin publishes tier, the kernel-computed binary_hash (external tier only, empty for trusted), env_var_names (names referenced in the submitted source including extras, never values) and the plugin's declared extras fields (empty key dropped) — extras and env_var_names both serialize as [] not null when empty"
    requirement: "PLUG-08"
    verification:
      - kind: unit
        ref: "kernel/httpapi/config_test.go#TestDescribePluginHandler_ExternalBinary_ReturnsTierAndKernelComputedHash"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/config_test.go#TestDescribePluginHandler_TrustedBinary_ReturnsTierTrustedAndEmptyHash"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/config_test.go#TestDescribePluginHandler_EnvVarNamesReferenceTokenAndExtras_NeverLeaksValues"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/config_test.go#TestDescribePluginHandler_ExtrasAndEnvVarNamesSerializeAsEmptyArrayNotNull"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/describe_test.go#TestFilterExtras_DropsEntryWithEmptyKey"
        status: pass
    human_judgment: false
  - id: D5
    description: "Unknown plugin binary still refused 404 plugin_binary_not_found before anything executes, unaffected by the widened response shape"
    requirement: "PLUG-08"
    verification:
      - kind: unit
        ref: "kernel/httpapi/config_test.go#TestDescribePluginHandler_UnknownBinaryStillReturns404AlongsidePhase11Fields"
        status: pass
    human_judgment: false
  - id: D6
    description: "docs/plugin-contract.md, docs/api.md and config.example.toml republished to document trust tiers, pinning, the exact nine-variable launch-environment allowlist, the extras config shape/Describe declaration, and every Phase 11 wire field/config key — with no claim of sandboxing/containment/isolation anywhere"
    requirement: "PLUG-09"
    verification:
      - kind: automated_ui
        ref: "make docs-check (35 links across 18 files, all resolve) && grep -c external_dir docs/api.md docs/plugin-contract.md config.example.toml (1/2/2)"
        status: pass
    human_judgment: false

duration: ~11min (git commit span, 10:13:34-10:24:06 UTC+1)
completed: 2026-08-13
status: complete
---

# Phase 11 Plan 03: Publishing the Trust Boundary — Sources API, Describe-Plugin, and the Republished Contract Summary

**A pin-mismatched external source now appears as a real, named `launch_failure: "pin_mismatch"` entry on `GET /api/sources` instead of silently vanishing; `POST /api/config/describe-plugin` publishes tier, the kernel-computed binary hash, referenced env-var names and declared extras before anything is saved; and `docs/plugin-contract.md`/`docs/api.md`/`config.example.toml` are republished to match.**

## Performance

- **Duration:** ~11 min by git commit span (`7d32039` → `b17f8e3`); active engineering time (reading plans 11-01/11-02's summaries and the existing sources.go/host.go/config.go code before writing) was materially longer
- **Started:** 2026-08-13T10:13:34+01:00 (first task commit)
- **Completed:** 2026-08-13T10:24:06+01:00
- **Tasks:** 3/3
- **Files modified:** 9 (0 new, 9 modified)

## Accomplishments

- `kernel/httpapi.HealthProber` gained `LaunchFailures() []pluginhost.LaunchFailure`; `*pluginhost.Host` and `*supervisor.Supervisor` already satisfied it structurally via plan 11-02's own delegation, so no third implementer needed updating
- `sourceStatusesFrom` now merges TWO sets — live probe results and soft launch failures — into one sorted `GET /api/sources` response: a configured, never-launched source (pin mismatch) produces a real entry naming its instance id, display name, plugin binary, tier, `launch_failure: "pin_mismatch"`, both hashes, and the kernel's own named refusal message as `last_error`; a name present in both sets never produces two entries (the probe result wins)
- `pluginhost.Plugin` and `SourceHealth` both gained `pinnedHash`/`PinnedHash`, populated for every launched external-tier instance (not only a mismatched one) — the chip menu's pinned-hash footer now has this fact for a healthy source too
- `pluginhost.launch()` computes an external-tier binary's SHA-256 for EVERY launch including a `describeOnly` trial launch (previously only real, non-trial launches hashed the binary) — `DescribeInfo.BinaryHash` and `Plugin.Extras()` (Describe-declared extras, empty-key entries dropped) are now available before any source using that binary is ever persisted or pinned
- `describePluginResponse` gained `tier`, `binary_hash`, `env_var_names` (scanned via the existing shared `config.EnvRefNames`, names only, never a value) and `extras` — the add-source confirm interstitial's complete kernel-derived fact set
- `docs/plugin-contract.md` gained "Trust tiers", "Pinning" and "The launch environment" sections (the exact nine-variable allowlist, named), extended `WEBSPACES_SOURCE_CONFIG` and `Describe` for extras, and states plainly that topos does not sandbox plugins
- `docs/api.md` documents every Phase 11 field this plan and its predecessors introduced (`tier`/`pinned_hash`/`current_hash`/`launch_failure`, `plugin_type_tiers`, `tier`/`binary_hash`/`env_var_names`/`extras` on describe-plugin) and the three new config keys; `config.example.toml` documents `external_dir`, a `[plugins.pins]` example and a `[sources.paperless.extras]` example at the existing comment density

## Task Commits

Each task was committed atomically:

1. **Task 1: Merge launch-failed sources into GET /api/sources with a machine-readable failure field** - `7d32039` (feat)
2. **Task 2: describe-plugin publishes tier, the kernel-computed hash, referenced env var names and declared extras** - `8bc37b6` (feat)
3. **Task 3: Republish the plugin contract, the API reference and the example config** - `b17f8e3` (docs)

_No TDD tasks in this plan's execution — `tdd="true"` was declared on Tasks 1-2; tests were written and verified alongside each task's implementation within a single commit per task, matching plans 11-01/11-02's own established single-commit-per-task convention._

## Files Created/Modified

**Task 1 (kernel/httpapi, kernel/pluginhost):**
- `kernel/httpapi/sources.go` - `HealthProber.LaunchFailures`; `sourceStatus.PinnedHash/CurrentHash/LaunchFailure`; `sourceStatusesFrom` rewritten to merge probe + failure sets
- `kernel/httpapi/sources_test.go` - `fakeProber.LaunchFailures`; 5 new tests (merge shape, probe-wins-collision, pinned-hash-on-healthy-external, agent-route grant filter)
- `kernel/pluginhost/host.go` - `Plugin.pinnedHash`; `SourceHealth.PinnedHash`; `launch()` carries the matched pin out via `launchPinnedHash`; `ProbeSources` populates it

**Task 2 (kernel/httpapi, kernel/pluginhost):**
- `kernel/pluginhost/host.go` - `Plugin.binaryHash/extras` + `BinaryHash()/Extras()` accessors; `DescribeInfo.BinaryHash/Extras`; `launch()`'s pin block widened to compute the hash unconditionally for `TierExternal` (both real and trial launches); new `filterExtras` helper
- `kernel/pluginhost/describe_test.go` - `TestFilterExtras_DropsEntryWithEmptyKey`
- `kernel/httpapi/config.go` - `extrasFieldResponse`; `describePluginResponse.Tier/BinaryHash/EnvVarNames/Extras`; `DescribePluginHandler` populates all four
- `kernel/httpapi/config_test.go` - `newExternalTierDescribeRouter` helper; 5 new tests (external tier + hash, trusted tier + empty hash, env-var-names-never-leaks-values, empty-array-not-null, 404 unaffected)

**Task 3 (docs):**
- `docs/plugin-contract.md` - "Trust tiers", "Pinning", "The launch environment" sections; extras additions to `WEBSPACES_SOURCE_CONFIG` and `Describe`; trial-launch honesty note
- `docs/api.md` - `GET /api/sources`, `GET /api/config/plugin-types`, `POST /api/config/describe-plugin` and the `GET /api/config` config-document section all extended
- `config.example.toml` - `external_dir`, `[plugins.pins]`, `[sources.paperless.extras]` commented examples

## Decisions Made

- **`binaryHash` is computed unconditionally for `TierExternal`, not only on trial launches.** Plan 11-02's pin-check block only ran `hashAndLookupPin` for real (non-trial) launches; this plan widened the SAME block to always compute the hash (skipping only the pin-comparison/refusal for `describeOnly`), so `DescribePluginType`'s trial launch and a real launch share one code path and one `HashBinary` call rather than two independent hashing sites.
- **`PinnedHash` flows onto `SourceHealth`, not only `LaunchFailure`.** The plan's own must-haves required a healthy external-tier source to also report `pinned_hash` (the chip menu's footer needs it either way) — sourced from the same `launchPinnedHash` the pin-check block already computes on a match.
- **`extrasFieldResponse` is a hand-written mirror, not a reuse of `toposv1.ExtrasField`** — this package never leaks a generated proto type across the HTTP JSON boundary; five fields, `snake_case` json tags, matching every other response shape in `kernel/httpapi/config.go`.
- **Documentation section placement**: "Trust tiers"/"Pinning"/"The launch environment" land as top-level `##` sections directly after "Discovery and launch" and before "Configuration" — the reading order a third-party plugin author actually needs (discover → what launched it → what environment it gets → its own config surface).

## Deviations from Plan

None — plan executed exactly as written. The one design choice not explicitly spelled out in the plan text (computing `binaryHash` unconditionally rather than only inside a new, separate code path) is a direct, minimal reading of the plan's own instruction to "reuse the existing unexported `launch()`... rather than duplicating any of it" (echoing `DescribePluginType`'s own established convention) — not a deviation from the stated behavior, just the natural implementation of it.

## Issues Encountered

None. All three tasks' `<verify>` commands passed on the first attempt: `go test ./kernel/httpapi/... ./kernel/pluginhost/...` (Tasks 1-2) and `make docs-check && grep -c 'external_dir' docs/api.md docs/plugin-contract.md config.example.toml` (Task 3, output `1`/`2`/`2` matching the plan's own three-file requirement).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `PLUG-07`/`PLUG-08` are fully delivered at the kernel HTTP boundary: a refused-launch source is now visible and actionable through the API (the re-pin remedial action plans 11-05/11-06 build has a real, closed-vocabulary field to gate on), and the add-source flow has every fact it needs (tier, hash, referenced env vars, extras) to warn honestly and pin correctly before persisting anything.
- `PLUG-09`'s documentation half is delivered: the published contract now accurately describes the trust boundary, the launch environment, and extras — coherent with what plans 11-01/11-02 actually shipped.
- Full local verification green: `CGO_ENABLED=0 go build ./...` and `go test ./...` (whole repo, all workspace modules) both pass; `make docs-check` passes (35 links, 18 files).
- **`make e2e` was NOT run for this plan.** No frontend (`web/`) files were touched by any of this plan's three tasks — only Go source under `kernel/`/`kernel/pluginhost` and three documentation files. Given the plan's own `<verification>` block states "no UI change in this plan; the additive fields must not disturb any existing spec," and this plan ran as a parallel worktree executor alongside plan 11-04 (which owns `docs/testing.md`/`Makefile`/`go.work`/e2e fixtures), the full Playwright suite was left for the orchestrator's post-merge verification pass rather than run redundantly here. Worth confirming the full e2e suite still passes 104/104 post-merge, though nothing in this plan's diff has a plausible path to breaking it (no `web/` file changed, and every new Go field is additive/`omitempty`).
- The re-pin UI action (11-UI-SPEC.md E4), the picker's "Untrusted" label (E3), the confirm interstitial (E1), the pinned-hash footer (E5), and the extras form (E6) are all still unbuilt frontend work — this plan's job was exclusively publishing the kernel-side facts those UI pieces will consume; nothing here renders anything.

---
*Phase: 11-external-plugins-the-trust-boundary*
*Completed: 2026-08-13*
