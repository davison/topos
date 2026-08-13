---
phase: 11-external-plugins-the-trust-boundary
plan: 01
subsystem: plugin-host
tags: [go-plugin, svelte5, playwright, trust-boundary, plugin-discovery]

requires: []
provides:
  - "kernel/pluginhost.Tier/Dirs/TieredBinary/ResolveBinary/DiscoverTiered/DiscoverAllTiered — the two-tier plugin discovery/launch-time provenance authority"
  - "kernel/config.PluginsConfig.ExternalDir/Pins, kernel/config.Source.Extras — the Phase 11 config surface (declared, not yet behaviorally wired)"
  - "GET /api/sources tier field and GET /api/config/plugin-types plugin_type_tiers — the kernel-derived provenance published to the browser"
  - "web/src/lib/api.ts's complete Phase 11 TypeScript wire surface (tier, pinned_hash, current_hash, launch_failure, extras, external_dir, pins, plugin_type_tiers, ExtrasFieldDecl, and DescribePluginResponse's tier/binary_hash/env_var_names/extras)"
  - "web/src/lib/components/TrustBadge.svelte — the reusable chip/picker trust-badge overlay"
  - "web/e2e/fixtures's two-tier support (externalPluginBinaries, hashPluginBinary, plugins.external_dir/pins) for every later Phase 11 e2e spec"
affects: [11-02, 11-03, 11-04, 11-05, 11-06]

actuals:
  tokens: 33048
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Trust tier is derived exclusively from launch-time directory resolution (pluginhost.ResolveBinary), never from plugin-asserted data — set once at launch and read everywhere downstream (Plugin.tier, SourceHealth.Tier, DescribeInfo.Tier)"
    - "DiscoverXxx/DiscoverAllXxx split (UI-policy catalog vs. security-authority listing) extended from one directory to two tiers without changing which callers use which function"
    - "TrustBadge.svelte: a snippet-wrapping overlay component, conditionally rendering nothing at all for the common (trusted) case — zero visual/DOM cost when a feature does not apply"

key-files:
  created:
    - kernel/pluginhost/tier_test.go
    - kernel/supervisor/externaltier_test.go
    - cmd/topos/externaldir_test.go
    - web/src/lib/components/TrustBadge.svelte
    - web/src/lib/components/trust-badge.test.ts
    - web/e2e/specs/11-external-tier-badge.spec.ts
  modified:
    - kernel/pluginhost/discover_binaries.go
    - kernel/pluginhost/host.go
    - kernel/config/types.go
    - kernel/supervisor/supervisor.go
    - kernel/httpapi/config.go
    - kernel/httpapi/sources.go
    - kernel/httpapi/routes.go
    - cmd/topos/main.go
    - web/src/lib/api.ts
    - web/src/lib/components/SourceChip.svelte
    - web/e2e/fixtures/plugin-binaries.ts
    - web/e2e/fixtures/config-builder.ts
    - web/e2e/fixtures/kernel.ts
    - web/e2e/e2e-builtins.d.ts

key-decisions:
  - "ExcludedPluginBinaries (mock/mockstrict) applies uniformly across both tiers in DiscoverTiered — a fixture binary in the external directory is excluded from the picker exactly like one in the trusted directory (11-CONTEXT.md left this to planner discretion)."
  - "kernel/httpapi/whatsapplink.go deliberately left untouched, resolving only against the trusted directory — the QR link flow is an in-repo-plugin-only path per this plan's own prohibition, not a leftover single-directory assumption."
  - "SourceStatus.tier and DescribePluginResponse.tier/binary_hash/env_var_names/extras are declared as REQUIRED TypeScript fields (no `?`) even though the Go backend does not populate the DescribePluginResponse quartet yet — the plan's own text specified this exact non-optional shape so no later Phase 11 plan needs to re-edit api.ts; pinned_hash/current_hash/launch_failure stayed optional per the plan's explicit `?` on those three."
  - "trust-badge e2e assertions scope past WebspaceHeader's own invisible (visibility:hidden) measurement-clone chip row via Playwright's `:visible` pseudo-class — an unscoped page-wide count would double every real badge."

requirements-completed: [PLUG-06]

coverage:
  - id: D1
    description: "Two-tier plugin discovery/launch: a binary present only in the external directory is discovered, launched, and syncs exactly like an in-repo binary; a name collision resolves to the trusted copy with a named warning log line (D-11)"
    requirement: "PLUG-06"
    verification:
      - kind: unit
        ref: "kernel/pluginhost/tier_test.go#TestResolveBinary_* (6 cases)"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/discover_binaries_test.go#TestDiscoverAllTiered_*/TestDiscoverTiered_* (6 cases)"
        status: pass
      - kind: integration
        ref: "kernel/supervisor/externaltier_test.go#TestExternalTier_* (2 cases, real supervisor boot + real subprocess launch)"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/11-external-tier-badge.spec.ts#GET /api/sources reports tier \"trusted\"...\"external\""
        status: pass
    human_judgment: false
  - id: D2
    description: "Tier published on GET /api/sources (per-instance) and GET /api/config/plugin-types (plugin_type_tiers, additive, no schema_version bump); a missing external directory is a legitimate empty tier"
    requirement: "PLUG-06"
    verification:
      - kind: unit
        ref: "kernel/httpapi/sources_test.go#TestSourcesHandler_ReportsTierPerInstance"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/config_test.go#TestPluginTypesHandler_ReturnsPluginTypeTiersAlongsidePluginTypes"
        status: pass
      - kind: unit
        ref: "cmd/topos/externaldir_test.go#TestDefaultExternalPluginsDir_*/TestExternalPluginsDir_* (5 cases)"
        status: pass
    human_judgment: false
  - id: D3
    description: "TrustBadge overlay renders a CircleAlert glyph on an external-tier source's chip icon at the specified chip/picker scales; a trusted-tier chip's pill/markup/tooltip stay byte-identical to before this phase (D-06)"
    requirement: "PLUG-08"
    verification:
      - kind: unit
        ref: "web/src/lib/components/trust-badge.test.ts (16 cases)"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/11-external-tier-badge.spec.ts#exactly one chip renders the trust badge"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/11-external-tier-badge.spec.ts#tooltip discloses \"untrusted external plugin\""
        status: pass
    human_judgment: false
  - id: D4
    description: "The complete Phase 11 TypeScript wire surface declared once in web/src/lib/api.ts (SourceStatus, SourceConfig, KernelConfig, PluginTypesResponse, DescribePluginResponse, ExtrasFieldDecl) so no later plan needs to re-edit this file"
    verification:
      - kind: unit
        ref: "npm --prefix web run check (svelte-check, 0 errors)"
        status: pass
      - kind: unit
        ref: "npm --prefix web run test (802/802 pass)"
        status: pass
    human_judgment: false
  - id: D5
    description: "e2e fixture support for a second plugin directory (externalPluginBinaries, hashPluginBinary, plugins.external_dir/pins) so every later Phase 11 e2e spec can populate a two-tier fixture without touching the harness core again"
    verification:
      - kind: e2e
        ref: "make e2e (full suite, 104/104 pass, including the 3 new specs.11-external-tier-badge.spec.ts cases)"
        status: pass
    human_judgment: false

duration: ~40min active (plus a human-verification checkpoint pause between Task 1 and Tasks 2-3)
completed: 2026-08-13
status: complete
---

# Phase 11 Plan 01: External Plugins & the Trust Boundary — Tracer Slice Summary

**Two-tier plugin discovery (`pluginhost.Dirs`/`Tier`/`ResolveBinary`) with launch-time-only provenance published as `tier` on `GET /api/sources`, a `TrustBadge.svelte` overlay wired into `SourceChip`, and a real-browser Playwright spec proving a trusted and an external plugin sync side by side with exactly one badged chip.**

## Performance

- **Duration:** ~40min of active engineering work, plus a checkpoint pause between Task 1's commit and Task 2/3's continuation (auto mode was inactive — `workflow._auto_chain_active`/`workflow.auto_advance` both `false` — so the tracer feedback gate required an explicit human-verify approval before expansion, per this plan's own `type="tracer"` protocol)
- **Started:** 2026-08-13T00:13:09Z (worktree base)
- **Completed:** 2026-08-13T08:26:47Z
- **Tasks:** 3/3
- **Files modified:** 38 (14 modified in kernel/cmd for Task 1's signature widening, 24 across web for Tasks 2-3, net of test-fixup files touched by the Discover/NewSupervisor/Router signature change)

## Accomplishments
- A plugin binary present only in a second, external directory is discovered, launched, and syncs exactly like an in-repo binary — proven both by a real supervisor boot in Go (`externaltier_test.go`) and a real browser session (`11-external-tier-badge.spec.ts`)
- Trust tier is derived exclusively from which directory a binary resolved from at launch time (`pluginhost.ResolveBinary`), never from anything the plugin declares about itself — set once on `*Plugin` and read everywhere downstream (`SourceHealth.Tier`, `DescribeInfo.Tier`, the HTTP `tier` field)
- A name collision between the two directories resolves to the trusted copy and emits a named `hclog.Warn` line carrying the colliding binary's name (D-11) — never a silent shadow
- `GET /api/sources` publishes `tier` per instance; `GET /api/config/plugin-types` gains the additive `plugin_type_tiers` lookup table spanning both tiers, with no `schema_version` bump
- The complete Phase 11 TypeScript wire surface is declared once in `web/src/lib/api.ts` — every later Phase 11 plan consumes these types without re-editing this file
- `TrustBadge.svelte` renders a warning glyph on an external-tier source's chip icon at two declared scales (chip/picker); a trusted-tier chip's pill, border, background, and tooltip text stay byte-identical to before this phase
- The e2e harness gained a second plugin directory end to end (`externalPluginBinaries`, `hashPluginBinary`, `plugins.external_dir`/`pins`), proven by a real kernel booting one trusted and one external instance and the browser showing the badge on exactly the external chip

## Task Commits

Each task was committed atomically:

1. **Task 1: End-to-end "a binary from the external directory launches and reports its tier" — one path only** - `9aee73f` (feat)
2. **Task 2: Phase 11 wire types and the reusable trust badge on the source chip** - `97a86ac` (feat)
3. **Task 3: e2e fixture support for a second plugin tier, and the end-to-end badged-chip spec** - `760cb6d` (feat)

_No TDD tasks in this plan; Task 1 is `type="tracer"` (real implementation + real `<verify>`, single commit), Tasks 2-3 are `type="auto"`._

## Files Created/Modified

**Task 1 (kernel/cmd):**
- `kernel/pluginhost/discover_binaries.go` - `Tier`, `Dirs`, `TieredBinary`, `DiscoverAllTiered`, `DiscoverTiered`, `ResolveBinary`
- `kernel/pluginhost/host.go` - `Host.dirs`, `Discover`/`Reconcile`/`launch`/`DescribePluginType` take `Dirs`; `Plugin.tier`, `SourceHealth.Tier`, `DescribeInfo.Tier`
- `kernel/config/types.go` - `PluginsConfig.ExternalDir`/`Pins`, `Source.Extras` (config surface only)
- `kernel/supervisor/supervisor.go` - `Supervisor.dirs`, `NewSupervisor` takes `Dirs`
- `kernel/httpapi/config.go` - `PluginTypesHandler`/`DescribePluginHandler` take `Dirs`; `pluginTypesResponse.PluginTypeTiers`
- `kernel/httpapi/sources.go` - `sourceStatus.Tier`
- `kernel/httpapi/routes.go` - `Router` takes `Dirs`; `whatsapplink` route pinned to `dirs.Trusted` only
- `cmd/topos/main.go` - `defaultExternalPluginsDir`/`externalPluginsDir` (per-OS D-09 resolution)
- New tests: `kernel/pluginhost/tier_test.go`, `kernel/supervisor/externaltier_test.go`, `cmd/topos/externaldir_test.go`
- Fixed up every pre-existing test call site the signature widening broke (`reconcile_test.go`, `describe_test.go`, `describe_whatsapp_test.go`, `stderr_test.go`, `readiness_test.go`, `launchlatency_test.go`, `suspend_test.go`, `supervisor_test.go`, `agent_test.go`, `live_config_test.go`, `agent_live_config_test.go`, `contract_test.go`, `config_test.go`, `sources_test.go`)

**Task 2 (web/src):**
- `web/src/lib/api.ts` - the complete Phase 11 wire surface
- `web/src/lib/components/TrustBadge.svelte` - new
- `web/src/lib/components/SourceChip.svelte` - wraps `PluginIcon` in `TrustBadge`; tooltip gains the untrusted clause
- `web/src/lib/components/trust-badge.test.ts` - new (16 cases)
- Fixed up three pre-existing `SourceStatus` test fixtures the now-required `tier` field broke (`format.test.ts`, `sources.test.ts`, `staleness.test.ts`)

**Task 3 (web/e2e):**
- `web/e2e/fixtures/plugin-binaries.ts` - `linkPluginBinaries` gains `srcDir`; new `hashPluginBinary`
- `web/e2e/fixtures/config-builder.ts` - `FixtureConfigSpec.externalPluginBinaries`, `FixtureSourceSpec.extras`; `buildConfig` always writes `plugins.external_dir`/`pins`
- `web/e2e/fixtures/kernel.ts` - creates `plugins-external` and links into it
- `web/e2e/e2e-builtins.d.ts` - narrow ambient declarations for `node:crypto`'s `createHash` and a raw-bytes `readFileSync` overload (no `@types/node` in this tree)
- `web/e2e/specs/11-external-tier-badge.spec.ts` - new

## Decisions Made

- **ExcludedPluginBinaries applies uniformly across tiers.** `DiscoverTiered` filters the fixture names (`topos-plugin-mock`/`-mockstrict`) identically regardless of which directory they sit in — 11-CONTEXT.md left this to planner discretion; decided here for consistency with the exclusion's own stated purpose (keep dev fixtures out of the picker, not just out of one directory).
- **`kernel/httpapi/whatsapplink.go` left untouched, on the trusted directory only.** This is a deliberate narrowing (the QR link flow is in-repo-plugin-only), not a leftover single-directory assumption — recorded as a prohibition in this plan's `must_haves` and held.
- **`DescribePluginResponse`'s `tier`/`binary_hash`/`env_var_names`/`extras` are declared non-optional in TypeScript** even though the Go handler doesn't populate them yet — matching the plan's own literal spec so a later plan (pin verification, extras form) never needs to widen this shape; `pinned_hash`/`current_hash`/`launch_failure` on `SourceStatus` stayed optional per the plan's explicit `?`.
- **e2e badge-count assertions scope past the invisible measurement-clone chip row** (`WebspaceHeader.svelte`'s off-screen width-measurement clones, pre-existing since 09.1-01) via Playwright's `:visible` pseudo-class — discovered live when an unscoped page-wide count returned 2 instead of 1.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed every pre-existing Go test call site broken by the Discover/DescribePluginType/NewSupervisor/Router signature widening**
- **Found during:** Task 1 (implementing the `Dirs`-widened signatures)
- **Issue:** Changing `Discover(ctx, pluginsDir string, ...)` to `Discover(ctx, dirs Dirs, ...)` (and the same shape change to `DescribePluginType`/`NewSupervisor`/`Router`) is exactly what the plan's `<action>` text specifies ("No call site keeps a single-directory parameter"), but this broke ~14 pre-existing test files across `kernel/pluginhost`, `kernel/supervisor`, and `kernel/httpapi` that were not in this task's `files_modified` list.
- **Fix:** Updated every call site to wrap the existing string in `pluginhost.Dirs{Trusted: dir}` (mechanical, behavior-preserving for every existing trusted-only test).
- **Files modified:** `kernel/pluginhost/{reconcile,describe,describe_whatsapp,stderr}_test.go`, `kernel/supervisor/{readiness,launchlatency,suspend,supervisor}_test.go`, `kernel/httpapi/{agent,live_config,agent_live_config,contract,config,sources}_test.go`
- **Verification:** `go build ./... && go test ./...` (full repo) passes clean.
- **Committed in:** `9aee73f` (Task 1 commit)

**2. [Rule 3 - Blocking] Fixed three pre-existing `SourceStatus` test fixtures broken by the new required `tier` field**
- **Found during:** Task 2 (`svelte-check` after widening `SourceStatus`)
- **Issue:** `SourceStatus.tier` is declared non-optional per the plan's exact spec; three pre-existing `makeSource()` test factories (`format.test.ts`, `sources.test.ts`, `staleness.test.ts`) constructed `SourceStatus` literals with no `tier` field, failing type-check.
- **Fix:** Added `tier: 'trusted'` as each factory's default.
- **Files modified:** `web/src/lib/format.test.ts`, `web/src/lib/components/sources.test.ts`, `web/src/lib/components/staleness.test.ts`
- **Verification:** `npm run check` — 0 errors (down from 3).
- **Committed in:** `97a86ac` (Task 2 commit)

**3. [Rule 1 - Bug] Fixed the e2e badge-count assertion double-counting the invisible measurement-clone chip**
- **Found during:** Task 3 (running the new spec against a real browser)
- **Issue:** `page.locator('svg.lucide-circle-alert')` matched 2 elements, not 1 — `WebspaceHeader.svelte` renders an off-screen, `visibility: hidden` measurement clone of every chip (pre-existing since 09.1-01-PLAN.md, for overflow-width calculation), so an unscoped page-wide count doubled the real badge.
- **Fix:** Scoped the assertion with Playwright's `:visible` pseudo-class (`svg.lucide-circle-alert:visible`).
- **Files modified:** `web/e2e/specs/11-external-tier-badge.spec.ts`
- **Verification:** `make e2e E2E_ARGS=specs/11-external-tier-badge.spec.ts` — 3/3 pass.
- **Committed in:** `760cb6d` (Task 3 commit)

---

**Total deviations:** 3 auto-fixed (2 Rule 3 blocking, 1 Rule 1 bug)
**Impact on plan:** All three were necessary consequences of following the plan's own explicit instructions (the `Dirs`-widening, the non-optional `tier` field) or a genuine test-authoring bug caught by running the spec for real. No scope creep — no file outside this plan's stated intent was touched.

## Issues Encountered

- **No `@types/node` in `web/e2e/`'s TypeScript project** (by design — Task 1's package-legitimacy gate scoped its approval to exactly three packages). `hashPluginBinary`'s `node:crypto`/raw-bytes `readFileSync` usage needed narrow ambient declarations added to `web/e2e/e2e-builtins.d.ts`, matching that file's own established "declare exactly what's imported" discipline rather than installing a fourth package.
- **`node_modules` was not present in this worktree** at plan start (fresh worktree checkout) — ran `npm install` once at the start of Task 2 to unblock `svelte-check`/vitest/Playwright; no `package.json`/`package-lock.json` changes resulted (all needed packages were already declared).
- **`make e2e`'s build step overwrote `kernel/webui/build/.gitkeep`** with real SPA output — restored via `git checkout -- kernel/webui/build/.gitkeep` before committing Task 3, since that placeholder is deliberately the only tracked file under the gitignored `kernel/webui/build/*`.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The tracer slice this whole plan exists to prove is proven end to end: two-tier discovery, launch-time tier resolution, `tier`/`plugin_type_tiers` published over HTTP, and the chip badge — all with real subprocess/browser coverage, not merely unit-test inference.
- `PLUG-06` is fully delivered by this plan. `PLUG-08` is only the **chip half** delivered here (the badge + tooltip) — the picker "Untrusted" label (E3), the warning-before-confirm interstitial (E1), the "binary changed" re-pin flow (E4), the pinned-hash footer (E5), and the extras form (E6) are explicitly carried forward to plans 11-02/11-05/11-06 per this plan's own "Flagged assumptions" section; `requirements-completed` above lists `PLUG-06` only for that reason, even though the plan's frontmatter names both.
- `kernel/config.PluginsConfig.Pins`/`Source.Extras` are declared but structurally inert (no hash verification, no launch-env scrubbing yet) — plan 11-02 is expected to wire the behavior D-01/D-02/D-03/D-14 describe.
- The e2e fixture's `externalPluginBinaries`/`hashPluginBinary`/`plugins.pins` machinery is ready for a pin-verification spec without any further harness changes.
- No blockers. Full verification suite green: `go build ./... && go test ./...` (root + all Go workspace modules), `npm run check` (0 errors), `npm run test` (802/802), `make e2e` (104/104, full suite).

---
*Phase: 11-external-plugins-the-trust-boundary*
*Completed: 2026-08-13*
