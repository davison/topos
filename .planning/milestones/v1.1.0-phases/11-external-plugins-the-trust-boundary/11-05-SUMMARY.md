---
phase: 11-external-plugins-the-trust-boundary
plan: 05
subsystem: ui
tags: [svelte5, trust-boundary, plugin-picker, extras-form, playwright]

requires:
  - phase: 11-01
    provides: "kernel/pluginhost.Tier/Dirs/ResolveBinary — launch-time tier provenance; TrustBadge.svelte; the complete Phase 11 TypeScript wire surface in web/src/lib/api.ts; the e2e harness's two-tier fixture support"
  - phase: 11-03
    provides: "GET /api/config/plugin-types plugin_type_tiers; POST /api/config/describe-plugin's tier/binary_hash/env_var_names/extras — the kernel-derived facts this plan's UI consumes"
  - phase: 11-04
    provides: "testdata/external-plugin/ (topos-plugin-external-demo) — the genuinely out-of-repo proof binary this plan's e2e spec drives through the browser"
provides:
  - "AddSourceModal.svelte's 'untrusted-confirm' step (E1) — the type-the-binary-name-to-confirm interstitial that writes [plugins.pins] in the same save submitMatch already issues"
  - "Tier-aware picker rows (E2/E3): TrustBadge + an 'Untrusted' text label on every external-tier row in both picker groups"
  - "ConnectionForm.svelte's E6 extras section (declared fields + always-visible free-form editor), reused unforked by AddSourceModal's Connect step and EditSourceModal's Edit connection… flow"
  - "plugin-fields.ts: isExternalTier, UNTRUSTED_LABEL, ExtrasRow, extrasToRows, rowsToExtras, extrasKeyError"
  - "config-edit.ts: setPluginPin — the one function that ever writes [plugins.pins], always echoing back a kernel-computed hash"
  - "web/e2e/fixtures: FixtureConfigSpec.externalPluginBinariesSrcDir + plugin-binaries.ts's EXTERNAL_DEMO_BIN_DIR — lets a fixture pin/link an external binary that has no bin/plugins trusted-dir copy"
affects: [11-06]

actuals:
  tokens: 22700
  tasks: 3
  commits: 2

tech-stack:
  added: []
  patterns:
    - "A best-effort, silent describePlugin call fires at plugin-TYPE SELECTION time (before any field is filled) purely to learn declared extras keys — refreshed by the real Next-click describe response, never a second persisted call; a describe response missing `extras` entirely degrades to [] rather than propagating undefined into a function that assumes an array"
    - "ConnectionForm.svelte's free-form extras row list is $bindable — the one deviation from this file's 'parent owns all state, child only renders' discipline, needed so the caller (AddSourceModal/EditSourceModal) can validate the EXACT rows the form renders (extrasKeyError) without holding a second, possibly-divergent copy"
    - "A trust-consequential UI step (the untrusted-confirm interstitial) is a pure state transition with no network call of its own — the pin write rides the SAME putConfig call the following step already issues, never a second round trip"

key-files:
  created:
    - web/src/lib/components/untrusted-add.test.ts
    - web/src/lib/components/extras-form.test.ts
    - web/e2e/specs/11-untrusted-add.spec.ts
  modified:
    - web/src/lib/components/AddSourceModal.svelte
    - web/src/lib/components/ConnectionForm.svelte
    - web/src/lib/components/EditSourceModal.svelte
    - web/src/lib/components/WebspaceHeader.svelte
    - web/src/lib/components/ManageSourcesModal.svelte
    - web/src/lib/components/add-source.test.ts
    - web/src/lib/plugin-fields.ts
    - web/src/lib/config-edit.ts
    - web/src/routes/w/[webspace]/+page.svelte
    - web/e2e/fixtures/config-builder.ts
    - web/e2e/fixtures/kernel.ts
    - web/e2e/fixtures/plugin-binaries.ts
    - web/e2e/specs/uat-05-two-step-connect.spec.ts
    - web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts

key-decisions:
  - "The untrusted-confirm step's binary-name text IS the raw plugin binary (e.g. topos-plugin-external-demo), never a prettified pluginTypeLabel() form — matches D-05's own framing (accidental-click protection, not ceremony) and lets the confirm box compare byte-for-byte against the exact string the picker/kernel both use as this plugin's identity."
  - "ConnectionForm.svelte's extrasRows prop is $bindable (a deliberate, narrow exception to this file's otherwise strict 'no owned state, parent controls everything via values/onchange' discipline) — the alternative (recomputing a row-equivalent from the composed values.extras map at validation time) cannot represent an in-progress empty or duplicate key, since JS object keys collapse duplicates and filter empties before extrasKeyError would ever see them."
  - "EditSourceModal.svelte's 'connection' mode gained its own describePlugin call (previously only 'match' mode ever called it), split into a non-blocking loadEditExtrasFields — 'connection' mode has always opened synchronously and this plan does not delay that; the extras declarations simply arrive whenever the fire-and-forget call resolves, picked up by ConnectionForm's own reactive effect."
  - "The e2e spec fills topos-plugin-external-demo's declared, REQUIRED 'workspace_id' extras field via its labeled input, not the free-form editor — the plan's own action text says 'free-form extras row', but the real proof plugin (11-04) actually declares this exact key as required+non-secret, so the declared-field path is what a real browser session reaches; the underlying config write and the plugin's own observed item are identical either way."

requirements-completed: [PLUG-08, PLUG-09]

coverage:
  - id: D1
    description: "Adding a source from an external-tier plugin routes through a new untrusted-confirm interstitial (E1): binary name, full 64-hex-char kernel-computed hash, env-var disclosure (zero vs. one-or-many referenced vars), and a type-the-binary-name-to-confirm gate whose primary action stays disabled until the typed value exactly matches — confirming writes the pin in the SAME putConfig call the match step already issues"
    requirement: "PLUG-08"
    verification:
      - kind: unit
        ref: "web/src/lib/components/untrusted-add.test.ts (35 cases: step wiring, copy strings, disabled-until-exact-match, setPluginPin's single call site inside submitMatch)"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/11-untrusted-add.spec.ts#picker labels, the confirm-step gate, the written pin, and the extras item all prove out against the real out-of-repo binary"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every picker row backed by an external-tier plugin (both the existing-instance group and the install-catalog group) carries the TrustBadge + an 'Untrusted' text label; a trusted-tier row is unchanged and no third, untrusted-only picker section exists (D-07)"
    requirement: "PLUG-08"
    verification:
      - kind: unit
        ref: "web/src/lib/components/untrusted-add.test.ts (TrustBadge usage in both groups, exactly two group headers, Group 2's justify-between/ml-auto layout)"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/11-untrusted-add.spec.ts (Group 1 trusted row carries no 'Untrusted' text, Group 2 external tile does)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Save anyway is never offered for an external-tier plugin type — a connection-only save has no kernel-computed hash to pin, so it would create an unstartable, never-warned-about source"
    requirement: "PLUG-08"
    verification:
      - kind: unit
        ref: "web/src/lib/components/untrusted-add.test.ts#Save anyway is excluded for an external-tier plugin type (T-11-27)"
        status: pass
    human_judgment: false
  - id: D4
    description: "ConnectionForm.svelte's E6 extras section — a plugin's Describe-declared expected extras keys render as labeled inputs whose declared default is placeholder-only, never pre-filled into the bound value (D-14); an always-visible free-form key/value editor covers undeclared keys; extrasKeyError blocks an empty/duplicate/declared-colliding free-form key at the same submit-time point missingRequiredFields already runs; reused unforked by both AddSourceModal and EditSourceModal"
    requirement: "PLUG-09"
    verification:
      - kind: unit
        ref: "web/src/lib/components/extras-form.test.ts (31 cases: extrasToRows/rowsToExtras/extrasKeyError pure-function behavior, placeholder-vs-value binding, both consumers passing extrasFields into the same ConnectionForm)"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/11-untrusted-add.spec.ts (declared Workspace ID field round-trips into [sources.<id>.extras] and reaches the real out-of-repo plugin process, observed as a synced item)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Arbitrary provider-specific extras keys, entered in the UI, round-trip through config.toml to a genuinely out-of-repo plugin process unmodified, and the pin written equals the exact kernel-computed hash the confirm dialog displayed — proven by observation (the synced stream), not by a test's own mock"
    requirement: "PLUG-09"
    verification:
      - kind: e2e
        ref: "web/e2e/specs/11-untrusted-add.spec.ts (GET /api/config's plugins.pins compared against the hash string read from the dialog; the demo-proof instance's synced stream carries 'extras workspace_id=acme-42')"
        status: pass
    human_judgment: false

duration: ~1h active (session start through final commit; the two-task combined commit and the fixture/e2e debugging pass each took a meaningful share)
completed: 2026-08-13
status: complete
---

# Phase 11 Plan 05: The Untrusted-Source Confirm Interstitial, Tier-Aware Picker Rows, and the Extras Form Summary

**A new `'untrusted-confirm'` step in `AddSourceModal.svelte` gates adding any external-tier plugin behind a type-the-binary-name confirmation that names the binary, shows the full kernel-computed SHA-256 hash, and discloses referenced env vars — writing the pin in the same save the match step already issues — while every picker row for an external-tier plugin now carries a `TrustBadge` and an "Untrusted" label, and `ConnectionForm.svelte` gained a declared-plus-free-form extras editor proven, through a real browser session against the genuinely out-of-repo `topos-plugin-external-demo` binary, to round-trip a provider-specific key end to end.**

## Performance

- **Duration:** ~1h active engineering time (git commit span `b9ea270`→`47248a3`, 09:51–10:05 UTC on 2026-08-13, plus the reading/context-gathering pass before the first commit and the fixture/e2e debugging pass after the second)
- **Started:** 2026-08-13 (session start, worktree base `da25bd5f`)
- **Completed:** 2026-08-13T11:04:48+01:00
- **Tasks:** 3/3
- **Files modified:** 20 (3 new, 17 modified — 3 of those pre-existing e2e specs updated for behavior this plan's own Task 1/2 changes correctly introduced)

## Accomplishments

- `+page.svelte` threads `GET /api/config/plugin-types`' `plugin_type_tiers` lookup table through `WebspaceHeader` into `AddSourceModal`, alongside the existing `pluginTypes` fetch — no second network round trip
- `AddSourceModal.svelte`'s picker rows (both the existing-instance group and the install-catalog group) wrap their `PluginIcon` in `TrustBadge` and render an "Untrusted" text label whenever that row's plugin is external-tier — a trusted-tier row is byte-identical to before this phase, and no third, untrusted-only picker section was added (D-07)
- Selecting an external-tier plugin type and passing the connect step routes to a new `'untrusted-confirm'` step (E1) instead of straight to Match: binary name, full 64-hex-char kernel-computed hash (`break-all`), an env-var disclosure with two copy branches (zero vs. one-or-many referenced vars), and a type-the-binary-name-to-confirm `Input` whose primary action stays disabled until the typed value exactly, case-sensitively matches
- Confirming is a pure state transition — the pin write (`config-edit.setPluginPin`) rides the SAME `putConfig` call `submitMatch` already issues; `Save anyway` is hidden entirely for an external-tier plugin type, since a connection-only save has no kernel-computed hash to pin
- `ConnectionForm.svelte` gained the E6 extras section: a plugin's Describe-declared extras keys render as labeled inputs (secret-ish ones through `SecretField`) whose declared default binds only to the input's `placeholder`, never the value (D-14); an always-visible free-form key/value editor covers undeclared keys, with a bindable `extrasRows` list so the caller can validate the exact rows the form renders
- `AddSourceModal.svelte` learns a plugin type's declared extras via a best-effort, silent describe call fired at plugin-type SELECTION time (before any field is filled), refreshed by the real Next-click describe response — a response missing `extras` entirely degrades to `[]` rather than crashing a downstream validator
- `EditSourceModal.svelte`'s 'connection' mode (previously the only mode with no describe call at all) now fires a non-blocking describe to learn extras declarations too, without delaying that mode's existing synchronous open
- A real browser session (`web/e2e/specs/11-untrusted-add.spec.ts`) drives the entire journey against the genuinely out-of-repo `topos-plugin-external-demo` binary (11-04): picker labels, the confirm-step gate (wrong value stays disabled, exact value enables), cancelling and returning to add the declared "Workspace ID" extras value, confirming, saving, and — via `GET /api/config` and the synced stream — proving the written pin equals the exact hash string read from the dialog and the extras value reached the real out-of-repo process unmodified

## Task Commits

Each task was committed (Tasks 1-2 combined into one commit; see Deviations for why):

1. **Tasks 1-2: tier-aware picker, untrusted-source confirm step, and the extras form** - `b9ea270` (feat)
2. **Task 3: browser proof of the untrusted add journey and the extras round trip** - `47248a3` (test)

_No TDD tasks in this plan's execution — Tasks 1-2 were declared `tdd="true"`; their test files (`untrusted-add.test.ts`, `extras-form.test.ts`) were written and verified alongside the implementation within the single combined commit, matching this phase's own established single-commit-per-task convention for `type="auto" tdd="true"` tasks whose behavior is proven by a real test suite run before commit, not a separate RED/GREEN pair._

## Files Created/Modified

**Tasks 1-2 (`b9ea270`):**
- `web/src/lib/components/AddSourceModal.svelte` - `pluginTypeTiers` prop; `'untrusted-confirm'` step + `pendingBinaryHash`/`pendingEnvVarNames`/`confirmTyped` state; `cancelUntrustedConfirm`/`confirmUntrusted`; `declaredExtras`/`extrasRows` state + `loadDeclaredExtras`; tier-aware picker markup; Save-anyway exclusion; `setPluginPin` call inside `submitMatch`
- `web/src/lib/components/ConnectionForm.svelte` - `extrasFields`/`extrasRows` props; declared-field + free-form E6 markup; `declaredExtrasValues` state, the `extrasFields`-keyed re-sync effect, `commitExtras`/`setDeclaredExtra`/`setExtraRow*`/`addExtraRow`/`removeExtraRow`
- `web/src/lib/components/EditSourceModal.svelte` - `extrasFields` prop; `extrasRows` state; `extrasKeyError` check in `submitConnection`
- `web/src/lib/components/WebspaceHeader.svelte` - `pluginTypeTiers` prop, threaded to `AddSourceModal`
- `web/src/lib/components/ManageSourcesModal.svelte` - `extrasFields={[]}` on its own `EditSourceModal` call site (Rule 3 fix, see Deviations)
- `web/src/lib/components/add-source.test.ts` - widened for this plan's own required structural changes (Dialog `open` condition, Save-anyway guard, a search-window fix)
- `web/src/lib/plugin-fields.ts` - `isExternalTier`, `UNTRUSTED_LABEL`, `ExtrasRow`, `extrasToRows`, `rowsToExtras`, `extrasKeyError`; a new `topos-plugin-external-demo` `CONNECTION_FIELDS` row (Task 3 deviation, see below)
- `web/src/lib/config-edit.ts` - `setPluginPin`
- `web/src/routes/w/[webspace]/+page.svelte` - `pluginTypeTiers` state/fetch; `editExtrasFields` state; widened `handleChipEdit`/new `loadEditExtrasFields`
- `web/src/lib/components/untrusted-add.test.ts` - new (35 cases)
- `web/src/lib/components/extras-form.test.ts` - new (31 cases)

**Task 3 (`47248a3`):**
- `web/e2e/specs/11-untrusted-add.spec.ts` - new
- `web/e2e/fixtures/config-builder.ts` / `kernel.ts` / `plugin-binaries.ts` - `FixtureConfigSpec.externalPluginBinariesSrcDir`, `EXTERNAL_DEMO_BIN_DIR`
- `web/src/lib/plugin-fields.ts` - `topos-plugin-external-demo` connection field row (moved to this task's commit boundary conceptually, landed in the combined Task 1-2 commit since both files were touched together)
- `web/src/lib/components/AddSourceModal.svelte` / `web/src/routes/w/[webspace]/+page.svelte` - `resp.extras ?? []` guard (Rule 1 fix, see Deviations)
- `web/e2e/specs/uat-05-two-step-connect.spec.ts` / `uat-08-whatsapp-qr-link.spec.ts` - updated for the new selection-time describe call's effect on request counts

## Decisions Made

- **The untrusted-confirm step's binary-name text is the raw plugin binary**, never `pluginTypeLabel()`'s prettified form — matches D-05's "accidental-click protection, not ceremony" framing, and the confirm box compares byte-for-byte against the same string the picker/kernel both use as this plugin's identity.
- **`ConnectionForm.svelte`'s `extrasRows` is `$bindable`** — the one deliberate exception to this component's otherwise strict "owns no state, parent controls everything via values/onchange" discipline. Recomputing an equivalent row list from the composed `values.extras` map at validation time cannot represent an in-progress empty or duplicate key (plain JS object keys collapse duplicates and drop empties before `extrasKeyError` would ever see them), so the caller needs the exact, pre-composition row list the form is rendering.
- **`EditSourceModal.svelte`'s 'connection' mode gained its own describe call**, split into a non-blocking `loadEditExtrasFields` so 'connection' mode's pre-existing synchronous open is unaffected — extras declarations simply arrive whenever the fire-and-forget call resolves.
- **The e2e spec fills the real proof plugin's declared "Workspace ID" field**, not a free-form row, despite the plan's own action text saying "free-form extras row" — `testdata/external-plugin/plugin.go`'s actual `Describe` response declares `workspace_id` as required+non-secret, so a real browser session reaches the declared-field UI path; the underlying config write and the plugin's own observed item are identical either way, and this path additionally exercises E6's declared-field rendering the free-form-only interpretation would have skipped.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `ManageSourcesModal.svelte`'s own `EditSourceModal` call site needed `extrasFields={[]}`**
- **Found during:** Task 2, `npm run check` after widening `EditSourceModal`'s props
- **Issue:** `extrasFields` is a required prop on `EditSourceModal`; `ManageSourcesModal.svelte`'s own, second entry point to the edit-connection flow (distinct from the chip menu) didn't know about it, failing `svelte-check`.
- **Fix:** Passed `extrasFields={[]}` — this entry point has no describe call of its own, so declared fields never render there, but the always-visible free-form editor still works fully.
- **Files modified:** `web/src/lib/components/ManageSourcesModal.svelte`
- **Verification:** `npm run check` — 0 errors.
- **Committed in:** `b9ea270`

**2. [Rule 3 - Blocking] `add-source.test.ts` broke on this plan's own required structural changes**
- **Found during:** Task 1, `npm run test` after editing the Dialog `open` condition and the Save-anyway guard
- **Issue:** Three pre-existing literal/window-based assertions in `add-source.test.ts` (the Dialog's `open` expression, the Save-anyway `{#if describeFailed}` regex, and a fixed-width search window around a picker row) no longer matched after this plan's own instructed markup changes (adding `'untrusted-confirm'` to the open condition, widening the Save-anyway guard, and inserting `TrustBadge`/label markup between two previously-adjacent strings).
- **Fix:** Widened the three assertions to match the new, correct shape (prefix regex for the Save-anyway guard, a wider search window, an updated literal string) without weakening what they actually verify.
- **Files modified:** `web/src/lib/components/add-source.test.ts`
- **Verification:** `npm run test` — full suite green.
- **Committed in:** `b9ea270`

**3. [Rule 3 - Blocking] `topos-plugin-external-demo` had no `plugin-fields.ts` connection-field row**
- **Found during:** Task 3, writing the e2e spec — the connect step rendered no field for the plugin's required `path` key at all
- **Issue:** `connectionFieldsFor` falls back to Display Name + Sync Interval Override for any plugin binary with no declared row — `topos-plugin-external-demo`'s own required `path` field (mirroring `testdata/external-plugin/main.go`'s fatal guard) was structurally unreachable through the UI.
- **Fix:** Added a `topos-plugin-external-demo` row (required `path`, no seeded default — this plugin has no "standard install" location, unlike Signal/WhatsApp/mockstrict).
- **Files modified:** `web/src/lib/plugin-fields.ts`
- **Verification:** the e2e spec fills and submits this field successfully.
- **Committed in:** `b9ea270` (landed alongside Task 1-2's other `plugin-fields.ts` edits)

**4. [Rule 3 - Blocking] `linkPluginBinaries`'s external-directory call had no source-directory seam**
- **Found during:** Task 3, wiring the fixture — `hashPluginBinary`/`linkPluginBinaries` both defaulted to `PLUGIN_BIN_DIR`, which has no copy of `topos-plugin-external-demo` (built only into `bin/plugins-external/`)
- **Issue:** Every prior `externalPluginBinaries` fixture (e.g. `topos-plugin-mockstrict`) assumed a trusted-dir copy existed to hash/symlink from; a genuinely out-of-repo binary has none, so the existing call would throw "binary not found."
- **Fix:** Added `FixtureConfigSpec.externalPluginBinariesSrcDir` (threaded through `buildConfig`'s pin-hashing and `kernel.ts`'s `linkPluginBinaries` call) and `plugin-binaries.ts`'s `EXTERNAL_DEMO_BIN_DIR` constant — explicitly directed by this plan's own Task 3 action text ("extend the fixture call with the source-directory argument rather than copying binaries around").
- **Files modified:** `web/e2e/fixtures/config-builder.ts`, `web/e2e/fixtures/kernel.ts`, `web/e2e/fixtures/plugin-binaries.ts`
- **Verification:** `make e2e E2E_ARGS=specs/11-untrusted-add.spec.ts` passes; the written pin matches the real `bin/plugins-external/topos-plugin-external-demo` binary's hash.
- **Committed in:** `47248a3`

**5. [Rule 1 - Bug] `resp.extras` being `undefined` crashed `extrasKeyError`, silently stranding the WhatsApp connect flow**
- **Found during:** Task 3, running the full `make e2e` suite (not just the new spec) — every case in `uat-08-whatsapp-qr-link.spec.ts` broke
- **Issue:** That spec's scripted `describe-plugin` mock predates this phase and never declares an `extras` field. `declaredExtras = resp.extras` therefore set `declaredExtras` to `undefined`; `extrasKeyError(declaredExtras, extrasRows)` then threw `Cannot read properties of undefined (reading 'map')` — an uncaught exception inside `handleConnectNext`, outside its own try/catch, which silently halted the function before any `step = ...` assignment ran, leaving the dialog stuck on the connect step with no visible error.
- **Fix:** `declaredExtras = resp.extras ?? [];` at all four assignment sites (`AddSourceModal.svelte`'s `loadDeclaredExtras`/`handleConnectNext`, `+page.svelte`'s two `editExtrasFields` assignments) — a describe response that omits the field now degrades to an empty declarations list, exactly like a genuine describe failure already does.
- **Files modified:** `web/src/lib/components/AddSourceModal.svelte`, `web/src/routes/w/[webspace]/+page.svelte`
- **Verification:** `make e2e` — full suite, 105/105 pass (was 14 failing in `uat-08-whatsapp-qr-link.spec.ts` alone before this fix).
- **Committed in:** `47248a3`

**6. [Rule 1 - Bug] Two pre-existing e2e specs' request-count assertions didn't account for the new selection-time describe call**
- **Found during:** Task 3, running the full suite — `uat-05-two-step-connect.spec.ts` and `uat-08-whatsapp-qr-link.spec.ts`'s case 13 both failed on stale call-count expectations
- **Issue:** Task 2's best-effort describe call at plugin-type SELECTION time (not just at Next) is new, correct behavior this plan itself introduces (D-15) — but it shifts every subsequent describe call's ordinal position by one, breaking `uat-05`'s literal "zero requests on a blank field" assertion and `uat-08` case 13's `failFromCall: 2` (which now needed to fail the THIRD call, not the second, to still land on the intended post-decline retry).
- **Fix:** `uat-05` now captures a post-selection baseline and asserts no ADDITIONAL request on the blank-field click (rather than a literal zero), and on the real advancing click (baseline + 1); `uat-08` case 13's `failFromCall` moved from `2` to `3`.
- **Files modified:** `web/e2e/specs/uat-05-two-step-connect.spec.ts`, `web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts`
- **Verification:** both specs pass individually and as part of the full `make e2e` run.
- **Committed in:** `47248a3`

---

**Total deviations:** 6 auto-fixed (4 Rule 3 blocking, 2 Rule 1 bugs)
**Impact on plan:** Every deviation was a direct, necessary consequence of correctly implementing this plan's own instructions and then verifying that implementation against the FULL suite rather than only the new files — no scope creep beyond what following the plan and its own testing convention required. The two Rule 1 fixes in particular (the `undefined`-extras crash and the shifted request counts) would otherwise have shipped a regression invisible to this plan's own two new test files, since neither exercises a plugin type whose describe response predates Phase 11.

## Issues Encountered

- **`kernel/webui/build/.gitkeep` was overwritten by `make e2e`'s build step** (the same pre-existing gap 11-01-SUMMARY.md already documented) — restored via `git checkout -- kernel/webui/build/.gitkeep` before each commit, since that placeholder is deliberately the only tracked file under the gitignored `kernel/webui/build/*`.
- **Tasks 1 and 2 landed in one combined commit rather than two.** Both tasks' implementation is tightly interleaved within the same files (`AddSourceModal.svelte`'s `declaredExtras`/`extrasRows` state feeds both the untrusted-confirm flow and the extras form; `plugin-fields.ts` gained helpers for both in the same edit pass) — splitting them after the fact risked landing a broken intermediate commit. Both tasks' own `<verify>` commands pass against the combined commit; their test files (`untrusted-add.test.ts` for Task 1, `extras-form.test.ts` for Task 2) remain independently runnable and independently verify each task's own scope.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `PLUG-08` and `PLUG-09` are now fully delivered at the UI layer: the picker discloses tier everywhere it renders a plugin, adding an external-tier source requires an explicit, informed, typed confirmation before anything is persisted, and arbitrary provider-specific extras keys can be entered, saved, and observed arriving at a genuinely out-of-repo plugin process unmodified.
- Plan 11-06 (per the roadmap's own dependency note) can now build the re-pin flow (E4: "Trust updated binary…", the chip menu action + confirmation dialog) and the pinned-hash chip-menu footer (E5) on top of this plan's `TrustBadge`/picker/confirm-step foundation and `config-edit.setPluginPin` — no further wiring in `AddSourceModal.svelte`'s tier-detection path should be needed there.
- `EXTERNAL_DEMO_BIN_DIR`/`FixtureConfigSpec.externalPluginBinariesSrcDir` are available for plan 11-06's own e2e coverage of the re-pin/binary-changed flow without any further fixture-layer changes.
- Full local verification green: `npm --prefix web run check` (0 errors), `npm --prefix web run test` (868/868), `npm --prefix web run check:e2e` (0 errors), `make e2e` (105/105, full suite including the new spec and every pre-existing one), `CGO_ENABLED=0 go build ./...` (repo root, unaffected by this plan's frontend-only + fixture-only changes).

---
*Phase: 11-external-plugins-the-trust-boundary*
*Completed: 2026-08-13*

## Self-Check: PASSED

- Verified files exist: `web/src/lib/components/AddSourceModal.svelte`, `ConnectionForm.svelte`, `EditSourceModal.svelte`, `WebspaceHeader.svelte`, `ManageSourcesModal.svelte`, `add-source.test.ts`, `untrusted-add.test.ts`, `extras-form.test.ts`, `web/src/lib/plugin-fields.ts`, `web/src/lib/config-edit.ts`, `web/src/routes/w/[webspace]/+page.svelte`, `web/e2e/specs/11-untrusted-add.spec.ts`, `web/e2e/fixtures/config-builder.ts`, `kernel.ts`, `plugin-binaries.ts`, `web/e2e/specs/uat-05-two-step-connect.spec.ts`, `uat-08-whatsapp-qr-link.spec.ts`
- Verified commits exist in `git log --oneline`: `b9ea270`, `47248a3`
