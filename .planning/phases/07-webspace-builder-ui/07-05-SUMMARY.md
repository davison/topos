---
phase: 07-webspace-builder-ui
plan: 05
subsystem: ui
tags: [svelte5, sveltekit, bits-ui, alert-dialog, config, delete, reload, go-ast]

# Dependency graph
requires:
  - phase: 07-webspace-builder-ui
    provides: "07-01's PUT /api/config (base_hash lock, canonical writer) + 07-02's POST /api/config/reload and live apply-on-save — every delete/reload in this plan rides these same seams"
  - phase: 07-webspace-builder-ui
    provides: "07-03's dialog/alert-dialog overlay primitives and config-edit.ts's pure edit-then-return-new-document discipline"
  - phase: 07-webspace-builder-ui
    provides: "07-04's EditSourceModal (reused unforked for Manage Sources' own Edit action), plugin-fields.ts, and SourceChip's chip-menu shape"
provides:
  - "config-edit.ts's removeSourceInstance: the one function that deletes a [sources.<id>] block AND clears every match-block/allowlist reference to it across every webspace in the same document"
  - "api.ts's reloadConfig() and CONFIG_CONFLICT_MESSAGE — the ONE exported hash-conflict copy constant every config-writing surface in the app now references by name instead of re-typing the literal"
  - "ManageSourcesModal.svelte — D-13's single escape hatch: instance/webspace delete behind AlertDialog confirmations, Reload config"
  - "save-state.test.ts — the cross-component guard proving the shared save/reload state pattern holds across every writing surface in the app, not by convention"
  - "kernel/httpapi/contract_test.go's TestContract_MutatingRoutesAreConfigScoped — the mechanical AST guard over the kernel's whole mutating HTTP surface, including the /agent/v1 GET-only guarantee"
affects: []

# Actuals (#2632) — pairs with the plan's estimate to calibrate future estimates.
actuals:
  tokens: 42000
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "ManageSourcesModal holds its own local (localConfig/localHash) snapshot, updated directly from each putConfig/reloadConfig/getConfig response rather than waiting on the parent route's async onchanged-triggered refresh — so a rapid sequence of deletes/edits/reloads within one modal session never races itself into a spurious hash-conflict against its own prior write."
    - "The fixed hash-conflict copy is now a single exported constant (api.ts's CONFIG_CONFLICT_MESSAGE), referenced by name from all five config-writing surfaces (CreateWebspaceModal, AddSourceModal, EditSourceModal, ManageSourcesModal, and the route's own modal-less filter/remove writes) rather than duplicated as a literal string per call site — pinned by save-state.test.ts's exactly-one-occurrence scan."
    - "kernel/httpapi/contract_test.go's nonGetRoutesInFile helper applies the identical AST-walk shape to both routes.go (explicit allowlist) and agent.go (must be empty) — factored once so the /agent/v1 GET-only guarantee and the /api/* mutating-surface allowlist are proven by the same mechanism, not two independently-maintained scans."

key-files:
  created:
    - web/src/lib/components/ManageSourcesModal.svelte
    - web/src/lib/components/manage-sources.test.ts
    - web/src/lib/components/save-state.test.ts
  modified:
    - web/src/lib/config-edit.ts
    - web/src/lib/config-edit.test.ts
    - web/src/lib/api.ts
    - web/src/lib/components/AddSourceModal.svelte
    - web/src/lib/components/CreateWebspaceModal.svelte
    - web/src/lib/components/EditSourceModal.svelte
    - web/src/lib/components/SourceChip.svelte
    - web/src/lib/components/WebspaceHeader.svelte
    - web/src/lib/components/save-filter-clone.test.ts
    - web/src/lib/node-builtins.d.ts
    - web/src/routes/w/[webspace]/+page.svelte
    - kernel/httpapi/contract_test.go
    - docs/api.md (audited only — no edit needed; see decisions)
    - config.example.toml
    - README.md

key-decisions:
  - "removeSourceInstance writes sources = [] (never omits the key) when a delete leaves a webspace's allowlist empty — kernel/config/types.go's Webspace.Participates treats a zero-length Sources slice identically to an absent one (all-instances-participate default), and KernelConfig's sources field is not optional on the wire, so [] IS the kernel's own encoding of that default, not a lesser substitute for dropping the key."
  - "CONFIG_CONFLICT_MESSAGE (api.ts) promoted and threaded through ahead of Task 2's own declared scope — pulled forward into Task 1's commit because ManageSourcesModal (Task 1's own deliverable) needed the shared constant immediately; introducing an 8th duplicate of the literal string only to delete it again in Task 2 would have been pure churn. All five writing surfaces (four modals + the route's modal-less writes) were converged onto the constant in the same commit."
  - "Audit gap closed (Task 2): SourceChip's 'Remove from this webspace' menu item had no in-flight disabled state, the one write that menu can trigger directly. Added a `busy` prop wired from WebspaceHeader's existing filterBusy (the same flag the route's handleRemoveSource already sets) — Edit connection…/Edit match settings… stay enabled regardless, since opening a modal isn't itself a write in flight."
  - "TestContract_MutatingRoutesAreConfigScoped in contract_test.go duplicates config_test.go's pre-existing TestRoutesGuard_NonGetRoutesScopedToConfig scan over routes.go by design — the plan names this exact test in this exact file, and the pre-existing guard stays unmodified. The new test's only genuinely new coverage is the second assertion: agent.go (a separate file, not scanned by the routes.go-only guard) registers zero non-GET routes, closing the /agent/v1 GET-only gap a routes.go-only scan structurally cannot see."
  - "docs/api.md required no edit: audited against every Task 3 criterion (raw-form guarantee, base_hash lock, save-equals-apply, reload-not-watch, filter parity, every error code across the phase's five plans, the read-only-over-source-data-vs-config-is-mutating distinction) and found already fully satisfied by 07-01/07-02's own prior work, presented as one coherent sequence of route sections rather than three appended fragments."

requirements-completed: [KERN-08, UI-12]

coverage:
  - id: D1
    description: "removeSourceInstance clears [sources.<id>] AND every match-block/allowlist reference to it across every webspace in the same document, so the kernel's own load-time validators never see a dangling reference (T-07-26)"
    requirement: "KERN-08"
    verification:
      - kind: unit
        ref: "web/src/lib/config-edit.test.ts (removeSourceInstance describe block, 6 tests)"
        status: pass
    human_judgment: false
  - id: D2
    description: "ManageSourcesModal is the single escape hatch (D-13): two height-capped/scrollable lists, both deletes behind AlertDialog confirmations naming their real blast radius with no type-to-confirm field, Reload config with the kernel's verbatim failure message"
    requirement: "UI-12"
    verification:
      - kind: unit
        ref: "web/src/lib/components/manage-sources.test.ts"
        status: pass
    human_judgment: true
    rationale: "The structural guard proves the template shape (section headings, truncate+title precedent, exact AlertDialog copy, no text input, distinct RotateCw/RefreshCw icons, single-entry-point label-set comparison against SourceChip.svelte); the actual delete/reload round trip against a live kernel (chip disappearing from every webspace, items disappearing from the stream, the subprocess actually stopping, a hand-edited file actually applying) was not exercised live — no make dev session was available in this execution environment, the same limitation every prior plan in this phase (07-01 through 07-04) recorded."
  - id: D3
    description: "One shared save/reload state pattern governs every config-writing surface: disabled submit in flight, the kernel's verbatim message in a destructive Alert on validation failure, the fixed CONFIG_CONFLICT_MESSAGE copy on hash conflict, a closed modal over an updated header as the only success signal — no toast anywhere in the app"
    requirement: "UI-12"
    verification:
      - kind: unit
        ref: "web/src/lib/components/save-state.test.ts (19 tests)"
        status: pass
    human_judgment: false
  - id: D4
    description: "The kernel's mutating HTTP surface stays scoped to configuration, mechanically pinned: routes.go's non-GET route set equals an explicit five-route allowlist, and agent.go registers zero non-GET routes under /agent/v1"
    requirement: "KERN-08"
    verification:
      - kind: unit
        ref: "kernel/httpapi/contract_test.go::TestContract_MutatingRoutesAreConfigScoped"
        status: pass
    human_judgment: false

duration: ~16 min (three tasks, no checkpoints — autonomous plan)
completed: 2026-08-08
status: complete
---

# Phase 7 Plan 5: Manage Sources Modal, Save/Reload State Guard, and Contract Publication Summary

**D-13's deliberately minimal escape hatch (instance/webspace delete behind AlertDialog confirmations, Reload config) closes the phase's builder surface; a cross-component guard pins the one shared save/reload state pattern every modal in Phase 7 now provably follows; and a Go AST test mechanically pins the kernel's whole mutating HTTP surface — including the /agent/v1 GET-only guarantee — while the published docs (`docs/api.md`, `config.example.toml`, `README.md`) tell a hand-editing operator exactly what the UI does to their file.**

## Performance

- **Duration:** ~16 min total plan span (three tasks, fully autonomous)
- **Tasks:** 3
- **Files touched:** 17 (3 created, 14 modified — one of the 14, `docs/api.md`, was audited with no edit needed)
- **Commits:** 3

## Accomplishments

- `config-edit.ts` gains `removeSourceInstance` — the one function that removes a `[sources.<id>]` block and clears every reference to it (match blocks + allowlist entries) across every webspace in the same document, so the kernel's own `validateMatchBlocks`/`validateSourcesAllowlist` never rejects a partial delete with a confusing "unknown instance" message. An allowlist left empty by the removal is written as `[]` deliberately — `Webspace.Participates` treats that identically to an absent key (all-instances-participate default), so this is the kernel's own encoding of the correct post-delete semantics, not a lesser substitute.
- `api.ts` gains `reloadConfig()` (`POST /api/config/reload`) and `CONFIG_CONFLICT_MESSAGE` — the fixed hash-conflict copy promoted to one exported constant. All five config-writing surfaces in the app (`CreateWebspaceModal`, `AddSourceModal`, `EditSourceModal`, the new `ManageSourcesModal`, and the route's own modal-less filter/remove writes) now reference it by name; `save-state.test.ts` proves the literal string appears in exactly one file (`api.ts`, its own definition).
- `ManageSourcesModal.svelte` — the single place in the app that deletes a source instance or a webspace outright: two divided, height-capped/scrollable lists (`Source instances`, `Webspaces`), instance `Edit` reusing `EditSourceModal`'s `connection` mode unforked, both deletes behind `AlertDialog` confirmations naming their real blast radius with no type-to-confirm field, and a `Reload config` footer that renders the kernel's verbatim failure message. Deleting the currently-open webspace navigates to another remaining one or to the root route's zero-webspaces empty state — never leaving the user on a route the kernel no longer knows. Wired from `WebspaceSwitcher`'s pre-existing `Manage sources…` item, replacing 07-03/07-04's no-op placeholder.
- `save-state.test.ts` — a 19-assertion cross-component guard proving the shared save/reload state pattern (in-flight disabled, verbatim validation failure, fixed hash-conflict copy, no-toast success) holds across every writing surface, plus a `ui/` primitive import allowlist scan proving no toast/sonner/snackbar primitive has crept in anywhere under `web/src/lib/components/`.
- `kernel/httpapi/contract_test.go` gains `TestContract_MutatingRoutesAreConfigScoped`: an AST scan over the real `Router`'s `routes.go` asserting its non-GET route set equals an explicit five-route allowlist, plus a second scan proving `agent.go` registers zero non-GET routes — the mechanical half of ROADMAP success criterion 4, verified red-on-addition live (a temporarily injected bogus mutating route failed the test with a clear diagnostic, then was reverted).
- `docs/api.md` was audited against every Task 3 criterion and found already fully compliant from 07-01/07-02's own prior work; `config.example.toml`'s header gains a paragraph on the UI write path (canonical-minimal rewrite, comment flattening, `config.toml.bak`, Reload as the no-restart hand-edit path) plus D-14's own `sources`-allowlist interaction rule; `README.md`'s stale "Phases 1–5" framing is corrected to "Phases 1–7", naming the webspace builder, the permanent filter, and Reload config.

## Task Commits

Each task was committed atomically:

1. **Task 1: Manage sources — instance and webspace lists, both destructive confirmations** — `9db2aeb` (feat)
2. **Task 2: One save/reload state pattern across every surface, pinned by a cross-component guard** — `3d2c1a1` (feat)
3. **Task 3: Publish the contracts and pin the config-scoped mutation guarantee** — `4fa5839` (docs)

## Files Created/Modified

- `web/src/lib/components/ManageSourcesModal.svelte` / `manage-sources.test.ts` (new) — the Manage Sources modal and its structural guard
- `web/src/lib/components/save-state.test.ts` (new) — the cross-component save/reload state guard
- `web/src/lib/config-edit.ts` / `config-edit.test.ts` — `removeSourceInstance` added
- `web/src/lib/api.ts` — `reloadConfig`, `CONFIG_CONFLICT_MESSAGE`
- `web/src/lib/components/{AddSourceModal,CreateWebspaceModal,EditSourceModal}.svelte` — converged onto `CONFIG_CONFLICT_MESSAGE`
- `web/src/lib/components/SourceChip.svelte` / `WebspaceHeader.svelte` — `busy` prop disabling "Remove from this webspace" in flight
- `web/src/lib/components/save-filter-clone.test.ts` — assertion updated to match the `CONFIG_CONFLICT_MESSAGE` identifier (was: a literal-string occurrence count)
- `web/src/lib/node-builtins.d.ts` — `statSync`/`relative` ambient declarations added for `save-state.test.ts`'s directory walk
- `web/src/routes/w/[webspace]/+page.svelte` — `manageOpen` state, `ManageSourcesModal` wiring, `CONFIG_CONFLICT_MESSAGE` adoption
- `kernel/httpapi/contract_test.go` — `TestContract_MutatingRoutesAreConfigScoped`, `nonGetRoutesInFile` helper
- `config.example.toml`, `README.md` — contract publication (see Decisions)

## Decisions Made

- **`removeSourceInstance` writes `sources: []`, never omits the key**, when a delete empties a webspace's allowlist — `Webspace.Participates` treats a zero-length slice identically to an absent one, and the field isn't optional on the wire, so `[]` is the kernel's own correct encoding of the all-instances-participate default, not a compromise.
- **`CONFIG_CONFLICT_MESSAGE` pulled forward into Task 1's commit** (ahead of its declared home in Task 2) because `ManageSourcesModal` needed it immediately; converging all five writing surfaces onto it happened in the same commit rather than introducing then deleting an eighth duplicate literal.
- **Audit gap closed in Task 2**: `SourceChip`'s "Remove from this webspace" item had no in-flight disabled state — added a `busy` prop wired from the existing `filterBusy` flag.
- **`TestContract_MutatingRoutesAreConfigScoped` duplicates an existing scan by design** — the plan names this exact test in this exact file; the pre-existing `config_test.go` guard stays untouched. Its only new coverage is the `agent.go` zero-non-GET-routes assertion.
- **`docs/api.md` needed no edit** — audited against every Task 3 criterion and found already fully compliant.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `SourceChip`'s "Remove from this webspace" menu item had no in-flight disabled state**
- **Found during:** Task 2's audit-and-converge pass over every config-writing surface
- **Issue:** E6 of 07-UI-SPEC.md's own UI Considerations table states "the initiating control disables in flight per the shared Save/Reload pattern" for remove-from-webspace, but `SourceChip.svelte`'s `DropdownMenuItem` for it carried no `disabled` binding — a user could trigger the same remove-from-webspace write twice in quick succession.
- **Fix:** Added a `busy: boolean` prop to `SourceChip.svelte` (default `false`), applied only to the "Remove from this webspace" item; wired from `WebspaceHeader.svelte`'s existing `filterBusy` state (the same flag the route's `handleRemoveSource` already sets) on both the visible-row and overflow-popover `SourceChip` usages.
- **Files modified:** `web/src/lib/components/SourceChip.svelte`, `web/src/lib/components/WebspaceHeader.svelte`
- **Verification:** `save-state.test.ts`'s modal-submit-disabled assertions plus the pre-existing `chip-edit-menu.test.ts` suite (unaffected — its label-set extraction strips tags before comparing) both pass; `npm run test` (448/448) and `npm run check` (0 errors) both green.
- **Committed in:** `3d2c1a1` (Task 2 commit)

**2. [Rule 1 - Bug/regression] `save-filter-clone.test.ts`'s literal disk-conflict-copy count broke when the copy moved to `CONFIG_CONFLICT_MESSAGE`**
- **Found during:** Task 1, after converging `+page.svelte`'s `writeFilter` onto the new shared constant
- **Issue:** A pre-existing 07-01 regression test counted literal occurrences of the disk-conflict copy string inside `writeFilter`'s catch block, to prove it was never reused as the fallback for an unrelated error. Once the literal was replaced by the `CONFIG_CONFLICT_MESSAGE` identifier, the literal-string count dropped to zero even though the invariant (never conflate a generic error with a hash conflict) still held.
- **Fix:** Updated the test to count occurrences of the identifier `CONFIG_CONFLICT_MESSAGE` instead of the literal string, preserving the test's original purpose.
- **Files modified:** `web/src/lib/components/save-filter-clone.test.ts`
- **Verification:** Full suite passes (448/448) after the update.
- **Committed in:** `9db2aeb` (Task 1 commit)

**3. [Rule 3 - Blocking compile dependency] `node-builtins.d.ts` missing `statSync`/`relative` ambient declarations**
- **Found during:** Task 2, writing `save-state.test.ts`'s recursive directory walk
- **Issue:** `save-state.test.ts` needed `node:fs`'s `statSync` and `node:path`'s `relative`, neither declared in the project's existing minimal ambient `node-builtins.d.ts` (no `@types/node` package is installed, per that file's own documented threat-model decision) — `npm run check` failed with two `TS2305` errors.
- **Fix:** Extended `node-builtins.d.ts` with narrow declarations for exactly these two functions, matching the file's existing "nothing more than what's actually imported" discipline.
- **Files modified:** `web/src/lib/node-builtins.d.ts`
- **Verification:** `npm run check` returns to 0 errors.
- **Committed in:** `3d2c1a1` (Task 2 commit)

---

**Total deviations:** 3 auto-fixed (1 Rule 1 gap closure, 1 Rule 1 regression-test update, 1 Rule 3 blocking compile dependency). None required an architectural decision or user input.

## Known Stubs

None — every surface this plan builds (instance/webspace delete, Reload config, the cross-component save-state guard, the mutation-scope AST guard) is a real, working implementation against the actual kernel API surface, exercised by its own test.

## Issues Encountered

- No live `make dev` session (a real running kernel) was available in this execution environment to perform this plan's own `<verification>` section's live-kernel checks: opening `Manage sources…` against real configured instances and webspaces, deleting an instance and confirming its chip/items/subprocess actually disappear without a restart, deleting a webspace and confirming every other instance/webspace survives, and hand-editing `config.toml` then clicking `Reload config` to confirm both the success and failure paths. Every unit-testable seam (`removeSourceInstance`, the two structural component guards, the AST route guard) is proven by a passing automated test; the live browser↔kernel round trip is not. Same limitation every prior plan in this phase (07-01 through 07-04) recorded — recorded in `coverage` above (D2) as needing a human/live-kernel pass before being trusted end to end.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- This is the final plan of Phase 7 (webspace-builder-ui). Every builder surface named in the phase's `07-UI-SPEC.md` is now shipped: the webspace switcher and create-webspace modal (07-03), the add-source picker and two-step connect flow plus the chip edit menu (07-04), and this plan's Manage Sources modal, cross-component save-state guard, and published contracts.
- Recommend a live-kernel pass (via `/gsd-verify-work` or a manual `make dev` session) covering this plan's own delete/reload flows, plus the phase's full end-of-phase human-check walkthrough named in `07-05-PLAN.md`'s `<verification>` block, before the phase is considered fully trusted end to end — no blocker for closing out the phase's plan sequence, since every shipped surface's own unit-testable seam is green.

---
*Phase: 07-webspace-builder-ui*
*Completed: 2026-08-08*

## Self-Check: PASSED
