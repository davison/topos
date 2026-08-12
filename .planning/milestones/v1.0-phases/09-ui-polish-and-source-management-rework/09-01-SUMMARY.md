---
phase: 09-ui-polish-and-source-management-rework
plan: 01
subsystem: plugin-contract
tags: [protobuf, grpc, go-plugin, sveltekit, svelte5, csp, xml, playwright]

requires: []
provides:
  - "DescribeResponse.icon/icon_mime additive proto fields (5, 6), regenerated Go stubs"
  - "kernel/pluginhost.Plugin icon capture at the existing launch()-time Describe call, validated (size/mime allowlist)"
  - "GET /api/plugins/{plugin}/icon: cached, hardened, read-only kernel route keyed on plugin binary name"
  - "PluginIcon.svelte: the one <img> rendering path + mandatory Puzzle fallback chain, used by SourceChip.svelte"
  - "SourceStatus.plugin (binary name) now present on every /api/sources row"
affects: [09-02, 09-03, 09-04, 09-05, 09-06, 09-07]

actuals:
  tokens: 18700
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Plugin-declared identity icon via an additive proto3 field, captured at an existing RPC call site (no new RPC, no handshake bump)"
    - "Kernel-served, hardened (CSP/nosniff/immutable-cache/ETag), read-only per-plugin-binary asset route"
    - "Frontend mandatory three-step fallback chain: unknown-plugin -> Puzzle; kernel 404/malformed -> <img> onerror -> Puzzle; else the real <img>"

key-files:
  created:
    - proto/topos/v1/plugin.proto (icon/icon_mime fields)
    - plugins/mock/assets/icon.svg
    - kernel/httpapi/pluginicon.go
    - kernel/httpapi/pluginicon_test.go
    - kernel/pluginhost/icon_test.go
    - web/src/lib/components/PluginIcon.svelte
    - web/src/lib/components/plugin-icon.test.ts
    - web/e2e/specs/09-plugin-icon.spec.ts
  modified:
    - sdk/gen/topos/v1/plugin.pb.go
    - sdk/contract_test.go
    - docs/plugin-contract.md
    - docs/api.md
    - plugins/mock/plugin.go
    - kernel/pluginhost/host.go
    - kernel/httpapi/sources.go
    - kernel/httpapi/routes.go
    - kernel/supervisor/supervisor.go
    - cmd/topos/main.go
    - web/src/lib/api.ts
    - web/src/lib/components/SourceChip.svelte
    - kernel/httpapi/agent_test.go
    - kernel/httpapi/agent_live_config_test.go
    - kernel/httpapi/live_config_test.go
    - kernel/httpapi/contract_test.go
    - web/src/lib/format.test.ts
    - web/src/lib/components/sources.test.ts
    - web/src/lib/components/staleness.test.ts

key-decisions:
  - "SourceStatus.plugin (binary name) added to GET /api/sources — the plan's key_links assumed this field already existed; it did not, so this was added as a Rule 2 fix, threading pluginhost.SourceHealth.Plugin -> httpapi sourceStatus.Plugin -> web SourceStatus.plugin"
  - "Fixed a literal '--' inside plugins/mock/assets/icon.svg's XML comment (Rule 1 bug, caught live by the tracer's own e2e run): Chromium's strict XML parser for an <img>-loaded SVG fails the whole document silently on this, always falling back to Puzzle with no console error naming the cause. The plan's exact CSP (including 'sandbox') was empirically NOT the cause and is unchanged."

patterns-established:
  - "A new plugin-contract field is captured at the plugin's existing Describe call site, never a second RPC — kept load-bearing across future icon consumers (09-02 expands to six more plugins)"
  - "Any future embedded-SVG asset with a hand-authored provenance comment must avoid a literal double-hyphen inside the comment body — noted directly in icon.svg's own comment for the next editor"

requirements-completed: []

coverage:
  - id: D1
    description: "Additive DescribeResponse.icon/icon_mime proto fields, regenerated stubs, unchanged RPC allowlist and handshake ProtocolVersion"
    verification:
      - kind: unit
        ref: "sdk/contract_test.go#TestContractDeclaresIconFields"
        status: pass
      - kind: unit
        ref: "sdk/contract_test.go#TestContractIconFieldsAreAdditive"
        status: pass
      - kind: unit
        ref: "sdk/contract_test.go#TestDescribeResponseIconFieldsZeroValue"
        status: pass
    human_judgment: false
  - id: D2
    description: "Mock plugin declares an icon; kernel captures/validates it at launch and serves it on GET /api/plugins/{plugin}/icon with hardened headers, 404-as-normal for undescribed plugins"
    verification:
      - kind: unit
        ref: "kernel/pluginhost/icon_test.go (TestCaptureIcon_*, TestHostPluginIcon_*, TestPluginIcon_AccessorMirrorsCaptureIcon)"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/pluginicon_test.go (TestPluginIconHandler_*)"
        status: pass
      - kind: integration
        ref: "make test-portable (all 7 workspace modules)"
        status: pass
    human_judgment: false
  - id: D3
    description: "PluginIcon.svelte renders the mandatory fallback chain and SourceChip shows the mock's real icon in a real browser, decoded"
    verification:
      - kind: unit
        ref: "web/src/lib/components/plugin-icon.test.ts (13 structural assertions)"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/09-plugin-icon.spec.ts (3 specs: decoded <img>, direct 200 with headers, unknown-plugin 404)"
        status: pass
    human_judgment: false

duration: ~50min
completed: 2026-08-11
status: complete
---

# Phase 9 Plan 1: Plugin-Provided Icon Contract (Tracer) Summary

**Additive `DescribeResponse.icon`/`icon_mime` proto fields, a cached/hardened `GET /api/plugins/{plugin}/icon` kernel route, and `PluginIcon.svelte`'s mandatory fallback chain — proven end to end with the mock plugin's own Lucide-derived, provenance-documented SVG rendering as a decoded `<img>` in a real browser.**

## Performance

- **Duration:** ~50min
- **Completed:** 2026-08-11
- **Tasks:** 3
- **Files modified:** 27 (12 created, 15 modified across proto/sdk/kernel/plugins/web)

## Accomplishments

- `proto/topos/v1/plugin.proto`'s `DescribeResponse` gained `icon = 5` / `icon_mime = 6`, additive and wire-compatible — no `sdk.Handshake.ProtocolVersion` bump, no new RPC — regenerated via `make proto` (buf), guarded by three new `sdk/contract_test.go` assertions plus the pre-existing RPC-allowlist test.
- `plugins/mock` embeds a provenance-documented (source, file, version, license), baked-color (`#94a3b8`, not `currentColor`) Lucide `FlaskConical` SVG and returns it from `Describe`.
- `kernel/pluginhost.Plugin` captures and validates the icon at the *existing* launch-time `Describe` call — no second RPC, no subprocess round trip. `MaxIconBytes = 65536` and a two-value mime allowlist (`image/svg+xml`, `image/png`) are enforced kernel-side; any failure is treated as "no icon declared," never a launch failure. `Host.PluginIcon(binary)` resolves by plugin BINARY name over launched instances.
- `kernel/httpapi/pluginicon.go` serves `GET /api/plugins/{plugin}/icon`: exact bytes, declared `Content-Type`, `Cache-Control: public, max-age=31536000, immutable`, a SHA-256 `ETag` with `304` support, `X-Content-Type-Options: nosniff`, `Content-Disposition: inline`, and a CSP (`default-src 'none'; style-src 'unsafe-inline'; sandbox`) neutralising a directly-navigated malicious SVG. A `{plugin}` value with a path separator or `..` 404s before any lookup. An undescribed plugin binary 404s in the standard error envelope — documented as the routine, expected case.
- `PluginIcon.svelte` implements the mandatory three-step fallback chain (empty/unknown plugin -> `Puzzle`; kernel 404/malformed bytes via the `<img>`'s own `onerror` -> `Puzzle`; otherwise the real, kernel-served `<img>`), decorative `alt=""`, `object-contain`, and a caller-supplied fixed-size box. `SourceChip.svelte` renders it at `size-3.5` between the health dot and the display name.
- `web/e2e/specs/09-plugin-icon.spec.ts` proves the whole path in a real (Chromium) browser: the chip's `<img>` actually decodes (`naturalWidth > 0`, not merely present in the DOM), a direct `GET` returns `200` with an `image/*` `Content-Type`, a non-empty `ETag`, and an immutable `Cache-Control`, and an unrecognised plugin binary `404`s.

## Task Commits

1. **Task 1: Additive icon fields on the plugin contract, guarded** - `d135668` (feat)
2. **Task 2: Mock plugin declares an icon; kernel caches and serves it** - `cd63328` (feat)
3. **Task 3: PluginIcon.svelte with the mandatory fallback chain, rendered in the source chip** - `bf08fe2` (feat)

_No separate plan-metadata commit — worktree mode; the orchestrator commits SUMMARY.md/STATE.md centrally after the wave._

## Files Created/Modified

- `proto/topos/v1/plugin.proto` - additive `icon`/`icon_mime` fields on `DescribeResponse`
- `sdk/gen/topos/v1/plugin.pb.go` - regenerated stubs (`make proto`, buf)
- `sdk/contract_test.go` - field presence, additive-compatibility (unchanged RPC set/handshake version), zero-value behavior
- `docs/plugin-contract.md`, `docs/api.md` - Describe/icon-route documentation
- `plugins/mock/assets/icon.svg` - embedded, provenance-documented, baked-color icon asset
- `plugins/mock/plugin.go` - `//go:embed`s and returns the icon from `Describe`
- `kernel/pluginhost/host.go` - `captureIcon`, `Plugin.iconBytes/iconMIME`/`Icon()`, `Host.PluginIcon`, `MaxIconBytes`, `allowedIconMIME`, `SourceHealth.Plugin`
- `kernel/pluginhost/icon_test.go` - capture/validation/resolution behavior, via a fake `sdk.SourcePlugin` and direct in-package `Plugin`/`Host` construction (no subprocess)
- `kernel/httpapi/pluginicon.go`, `pluginicon_test.go` - the icon route and its handler tests
- `kernel/httpapi/routes.go`, `kernel/httpapi/sources.go`, `kernel/supervisor/supervisor.go`, `cmd/topos/main.go` - route registration, `sourceStatus.Plugin`, `Supervisor.PluginIcon`, and the `Router` call sites (main + 4 test files) threaded with the new `PluginIconProvider` dependency
- `web/src/lib/components/PluginIcon.svelte`, `plugin-icon.test.ts` - the fallback-chain component and its structural guard
- `web/src/lib/components/SourceChip.svelte` - renders `PluginIcon` between the dot and the name
- `web/src/lib/api.ts` - `SourceStatus.plugin`
- `web/e2e/specs/09-plugin-icon.spec.ts` - the real-browser tracer proof
- `web/src/lib/format.test.ts`, `sources.test.ts`, `staleness.test.ts` - fixture `makeSource()` helpers updated with the new required `plugin` field

## Decisions Made

- **`SourceStatus.plugin` added (Rule 2, missing critical functionality).** The plan's `key_links` block asserted `SourceStatus.plugin` was "already on every `/api/sources` row" — it was not. Without it, `PluginIcon.svelte` has no key to build `GET /api/plugins/{plugin}/icon` from, so this was load-bearing, not optional. Threaded from `pluginhost.SourceHealth.Plugin` (new field, populated in `ProbeSources` from `p.src.Plugin`) through `kernel/httpapi/sources.go`'s `sourceStatus.Plugin` (`json:"plugin"`) to `web/src/lib/api.ts`'s `SourceStatus.plugin`.
- **Fixed a malformed XML comment in `plugins/mock/assets/icon.svg` (Rule 1, bug — found via the tracer's own e2e run, not a unit test).** The provenance comment's prose included a literal `(--muted-foreground)`. Chromium parses an `<img>`-loaded SVG with a strict XML parser; a `--` sequence inside a comment body (legal only as the closing delimiter) silently fails the whole document's parse, and the `<img>`'s `onerror` fires with **no console error naming plugin.svg** — the failure surfaced only as the frontend correctly, but misleadingly, falling back to `Puzzle`. Diagnosed by directly inspecting network responses/headers via the e2e harness and testing hypotheses (first suspected the plan's own `sandbox` CSP directive, confirmed by elimination it was NOT the cause — removed it, bug persisted; restored it once the real cause was found and fixed). The plan's exact CSP header, including `sandbox`, ships unchanged.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added `SourceStatus.plugin`/`sourceStatus.Plugin`/`SourceHealth.Plugin`**
- **Found during:** Task 3 (writing `PluginIcon.svelte`'s call site in `SourceChip.svelte`)
- **Issue:** The plan's `key_links` assumed `SourceStatus.plugin` already existed on every `/api/sources` row; `kernel/httpapi/sources.go`'s `sourceStatus` struct had no such field, and neither did `pluginhost.SourceHealth`.
- **Fix:** Added `Plugin string` to `pluginhost.SourceHealth` (populated in `ProbeSources`), `Plugin string \`json:"plugin"\`` to `httpapi.sourceStatus` (populated in `sourceStatusesFrom`), and `plugin: string` to `web/src/lib/api.ts`'s `SourceStatus`. Updated `docs/api.md`'s `/api/sources` example and prose, and three frontend test fixture `makeSource()` helpers.
- **Files modified:** `kernel/pluginhost/host.go`, `kernel/httpapi/sources.go`, `docs/api.md`, `web/src/lib/api.ts`, `web/src/lib/format.test.ts`, `web/src/lib/components/sources.test.ts`, `web/src/lib/components/staleness.test.ts`
- **Verification:** `go test ./kernel/...`, `cd web && npm run check && npm test` all green; the e2e spec's direct `GET /api/sources` assertion (via a temporary debug log during diagnosis, since removed) confirmed the field's presence and correct value.
- **Committed in:** `bf08fe2` (Task 3 commit)

**2. [Rule 1 - Bug] Removed a literal `--` from `plugins/mock/assets/icon.svg`'s XML comment**
- **Found during:** Task 3, running the tracer's own `web/e2e/specs/09-plugin-icon.spec.ts` for the first time — the chip always rendered `Puzzle`, never the real icon, with zero console errors identifying the cause.
- **Issue:** The provenance comment's prose parenthetical `(--muted-foreground)` put a literal double-hyphen inside an XML comment body — illegal per the XML spec (a comment's content must never contain `--` except as its closing delimiter). Chromium's strict XML parser for an `<img>`-loaded SVG failed the entire document's parse on this, silently triggering the `<img>`'s own `onerror` and falling back to `Puzzle` — the intended, correct behavior for a genuinely broken icon, but here caused by an authoring mistake in the icon's own comment, not by anything in the kernel/contract/component code.
- **Fix:** Reworded the comment to avoid any `--` sequence in its body, and added a note directly in the file warning future editors of the same asset about this exact failure mode (silent, no console error naming the file).
- **Files modified:** `plugins/mock/assets/icon.svg`
- **Verification:** `web/e2e/specs/09-plugin-icon.spec.ts`'s "decoded `<img>`" assertion (`naturalWidth > 0`) passes; confirmed the fix (not the plan's `sandbox` CSP directive, which was tested and ruled out first) by isolating each hypothesis independently before landing the real fix.
- **Files modified:** `plugins/mock/assets/icon.svg`
- **Committed in:** `bf08fe2` (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (1 missing critical functionality, 1 bug — both essential; the tracer task's whole purpose is to catch exactly this class of surprise before 09-02 expands to six more plugins).
**Impact on plan:** No scope creep — both fixes are strictly load-bearing for the plan's own must-have truths ("a configured, successfully-launched mock source renders its plugin-declared icon as an `<img>`... sourced from `GET /api/plugins/topos-plugin-mock/icon`"). The plan's exact CSP header (including `sandbox`) is unchanged; it was investigated and empirically ruled out as the cause during diagnosis.

## Issues Encountered

- Diagnosing the "always renders `Puzzle`" failure required isolating three independent hypotheses in sequence against a live e2e run (CSP `sandbox` directive, CSP entirely absent, then the SVG asset itself) before finding the real cause. Resolved; documented above and in the icon asset's own comment as a guard against recurrence.
- No node_modules were present in `web/` at session start (`npm install` required before `npm run check`/`npm test` could run) — a one-time setup cost, not a plan defect.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- The icon contract, kernel route, and frontend component are all production-quality and unchanged for every future consumer — 09-02 expands to the other six plugins by embedding an asset and returning it from `Describe`; no kernel or frontend code changes required per plugin.
- `SourceStatus.plugin` is now a stable, documented field any later plan can rely on (e.g., 09-11's add-source picker, which the phase's own `09-UI-SPEC.md` already plans to key on plugin icons/binary names).
- No blockers. The `sandbox` CSP directive is confirmed compatible with `<img>`-loaded SVG rendering in Chromium once the SVG itself is well-formed XML — a fact worth carrying into 09-02's real-logo assets (paperless-ngx, SilverBullet), which should be validated as well-formed XML before landing.

---
*Phase: 09-ui-polish-and-source-management-rework*
*Completed: 2026-08-11*

## Self-Check: PASSED

All 8 created files verified present on disk; all 3 task commit hashes (`d135668`, `cd63328`, `bf08fe2`) verified present in `git log --oneline --all`.
