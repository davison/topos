---
phase: 07-webspace-builder-ui
plan: 03
subsystem: ui
tags: [svelte5, sveltekit, bits-ui, dialog, dropdown-menu, alert-dialog, config, webspace]

# Dependency graph
requires:
  - phase: 07-webspace-builder-ui
    provides: "07-01's config.Store PUT /api/config (optimistic hash lock, canonical writer) and 07-02's hot-apply Supervisor.Apply — the create-webspace flow writes through this same guarded seam and the kernel reconfigures live, no restart"
provides:
  - "Three hand-adapted bits-ui overlay primitives (dialog, dropdown-menu, alert-dialog) in the repo's own house style — the trigger/content/title/footer decomposition, app.css-token-only styling, child({ props }) trigger composition — that every remaining builder surface in 07-04/07-05 is built from"
  - "web/src/lib/config-edit.ts: cloneConfig/addWebspace/removeWebspace/setWebspaceFilter, the single place any builder surface edits a KernelConfig document"
  - "WebspaceSwitcher.svelte + CreateWebspaceModal.svelte: the header's title becomes a drop-down webspace switcher that can create a new webspace through PUT /api/config, replacing the standalone home page (D-10)"
  - "web/src/lib/last-webspace.ts: readLastWebspace/writeLastWebspace/resolveRedirectTarget — the root route's redirect-only behavior and its zero-webspaces empty state"
affects: [07-04, 07-05]

# Actuals (#2632) — pairs with the plan's estimate to calibrate future estimates.
actuals:
  tokens: 19600
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Overlay primitive wrappers over bits-ui (dialog/dropdown-menu/alert-dialog): scaffolded by hand rather than via network CLI (no confirmed network access in this execution environment), matching popover/tooltip's existing shape exactly — ref = $bindable(null) + ...restProps spread, never destructuring child/children itself, so a caller's own {#snippet child({ props })} passes through to the bits-ui primitive unmolested"
    - "config-edit.ts's pure edit-then-return-new-document helpers: every builder write in this phase and the next (source add/edit, webspace delete) goes through one of these functions rather than an inline mutation at the call site"
    - "cloneConfig clones via a JSON round trip (JSON.parse(JSON.stringify(cfg))), not structuredClone or $state.snapshot() — config-edit.ts is a plain .ts module (not .svelte.ts), so the $state.snapshot() rune compiler macro isn't callable there; JSON.stringify reads through a Svelte 5 reactive Proxy exactly as a plain property access would, sidestepping 07-01's structuredClone-throws-on-Proxy bug without needing a rune"

key-files:
  created:
    - web/src/lib/components/ui/dialog/dialog.svelte
    - web/src/lib/components/ui/dialog/dialog-trigger.svelte
    - web/src/lib/components/ui/dialog/dialog-content.svelte
    - web/src/lib/components/ui/dialog/dialog-header.svelte
    - web/src/lib/components/ui/dialog/dialog-title.svelte
    - web/src/lib/components/ui/dialog/dialog-footer.svelte
    - web/src/lib/components/ui/dialog/index.ts
    - web/src/lib/components/ui/dropdown-menu/dropdown-menu.svelte
    - web/src/lib/components/ui/dropdown-menu/dropdown-menu-trigger.svelte
    - web/src/lib/components/ui/dropdown-menu/dropdown-menu-content.svelte
    - web/src/lib/components/ui/dropdown-menu/dropdown-menu-item.svelte
    - web/src/lib/components/ui/dropdown-menu/dropdown-menu-separator.svelte
    - web/src/lib/components/ui/dropdown-menu/index.ts
    - web/src/lib/components/ui/alert-dialog/alert-dialog.svelte
    - web/src/lib/components/ui/alert-dialog/alert-dialog-content.svelte
    - web/src/lib/components/ui/alert-dialog/alert-dialog-title.svelte
    - web/src/lib/components/ui/alert-dialog/alert-dialog-description.svelte
    - web/src/lib/components/ui/alert-dialog/alert-dialog-action.svelte
    - web/src/lib/components/ui/alert-dialog/alert-dialog-cancel.svelte
    - web/src/lib/components/ui/alert-dialog/index.ts
    - web/src/lib/components/ui/overlay-primitives.test.ts
    - web/src/lib/config-edit.ts
    - web/src/lib/config-edit.test.ts
    - web/src/lib/last-webspace.ts
    - web/src/lib/last-webspace.test.ts
    - web/src/lib/components/WebspaceSwitcher.svelte
    - web/src/lib/components/CreateWebspaceModal.svelte
    - web/src/lib/components/webspace-switcher.test.ts
  modified:
    - web/src/lib/components/WebspaceHeader.svelte
    - web/src/lib/components/save-filter-clone.test.ts
    - web/src/routes/w/[webspace]/+page.svelte
    - web/src/routes/+page.svelte

key-decisions:
  - "Hand-scaffolded the three bits-ui wrappers directly against node_modules/bits-ui's own type/export shape rather than running npx shadcn-svelte add against the live network registry — no confirmed network access in this execution environment, and the plan itself treats CLI output as a structural starting point only (hand-adapted to app.css tokens regardless of source), so the resulting wrapper API shape is what matters, not how it was typed"
  - "UI-SPEC ordering deviation, as pre-recorded by the plan itself: webspaces render in the order GET /api/config's JSON response lists them (Object.keys on the parsed document), never re-sorted client-side — the actual stability guarantee (never re-sorted per render, never reordered by state) holds; literal TOML file-declaration order does not, since a Go map cannot preserve it"
  - "web/src/lib/last-webspace.ts (formally a Task 3 deliverable) pulled forward into Task 2's own commit (Rule 3, blocking compile dependency) — Task 2's create-webspace flow calls writeLastWebspace per the plan's own action text, so the file had to exist and compile before Task 2's own npm run check/test could pass. Task 3 added no further exports to it, only last-webspace.test.ts"
  - "last-webspace.test.ts added as a sibling suite to last-webspace.ts rather than folded into config-edit.test.ts — the plan itself offers this as an alternative ('or add a sibling suite if it grows past a screen'), and it matches every other module+test-file pairing already established in this repo (config-edit.ts/config-edit.test.ts, FilterChip.svelte/filter-chip.test.ts)"

patterns-established:
  - "Overlay-primitive guard (overlay-primitives.test.ts): proves barrel exports, the trigger pass-through shape (destructures only ref, spreads ...restProps, never destructures child/children itself), the dropdown-menu content's max-height+overflow-y scroll cap, zero raw hex colours across the three new directories, and web/package.json's dependency set unchanged — a single guard file covering every acceptance criterion Task 1 names"
  - "Comment-stripped source-scan component guards (webspace-switcher.test.ts) continue the house pattern (filter-chip.test.ts, source-chip-pill.test.ts): scoped extractBetween blocks so an assertion can only be satisfied by the actual template region it claims to cover, never by prose elsewhere in the file"

requirements-completed: [UI-12]

coverage:
  - id: D1
    description: "Three overlay primitives (dialog, dropdown-menu, alert-dialog) hand-adapted from the already-installed bits-ui, no new npm dependency, with the child({ props }) trigger composition API every builder surface needs"
    requirement: "UI-12"
    verification:
      - kind: unit
        ref: "web/src/lib/components/ui/overlay-primitives.test.ts"
        status: pass
    human_judgment: false
  - id: D2
    description: "The header title is a working webspace switcher: lists every configured webspace with the current one weight-emphasized (aria-current + font-semibold), navigates on click, offers + New webspace and Manage sources… and nothing else"
    requirement: "UI-12"
    verification:
      - kind: unit
        ref: "web/src/lib/components/webspace-switcher.test.ts"
        status: pass
    human_judgment: true
    rationale: "The structural guard proves the template shape (aria-current, weight budget, exact menu-item set, trigger truncation), but the actual drop-down open/close/navigate/create interaction against a running kernel was not exercised live in this execution environment — no make dev session was available. Needs a human/live-kernel pass before this is trusted end to end."
  - id: D3
    description: "Creating a webspace from the switcher (or the empty-state CTA) writes a new [webspaces.<name>] block through PUT /api/config and navigates to it without a kernel restart; a kernel rejection leaves the modal open with the typed name intact and the kernel's own message verbatim"
    requirement: "UI-12"
    verification:
      - kind: unit
        ref: "web/src/lib/config-edit.test.ts"
        status: pass
    human_judgment: true
    rationale: "config-edit.ts's pure document-edit functions are unit-tested directly; CreateWebspaceModal's actual PUT /api/config round trip against a live kernel (success, validation-failure Alert, hash-conflict Alert, disabled-while-saving) was not exercised live — same environment limitation as D2."
  - id: D4
    description: "The root URL is redirect-only (remembered webspace, else first, else the zero-webspaces empty state), never a redirect loop, with a working Create webspace CTA when no webspaces exist"
    requirement: "UI-12"
    verification:
      - kind: unit
        ref: "web/src/lib/last-webspace.test.ts"
        status: pass
    human_judgment: false

duration: ~8 min (three tasks, no checkpoints — autonomous plan)
completed: 2026-08-08
status: complete
---

# Phase 7 Plan 3: Webspace Switcher, Create Webspace, and Root Redirect Summary

**Three hand-adapted bits-ui overlay primitives (dialog, dropdown-menu, alert-dialog), a webspace switcher replacing the static header title, a create-webspace modal writing through PUT /api/config, and a redirect-only root route with a real zero-webspaces empty state — retiring the standalone home page.**

## Performance

- **Duration:** ~8 min total plan span (three tasks, fully autonomous)
- **Started:** 2026-08-08T01:39:54+01:00 (Task 1's commit)
- **Completed:** 2026-08-08T01:47:49+01:00 (Task 3's commit)
- **Tasks:** 3
- **Files modified:** 32 (28 created, 4 modified)

## Accomplishments

- Three overlay primitives (`Dialog`, `DropdownMenu`, `AlertDialog`) now exist in the repo's own house style — same `child({ props })` trigger-composition API as the existing `popover`/`tooltip` wrappers, every colour resolved through an `app.css` token, zero new npm dependency (`git diff web/package.json` is empty) — and every remaining builder surface in 07-04/07-05 has what it needs.
- The header's static `<h1>{webspace}</h1>` is a working drop-down switcher: every configured webspace listed in the kernel's own `GET /api/config` order, the active one marked `aria-current` at weight 600 (the only emphasis cue — no separate icon), `+ New webspace` and `Manage sources…` and nothing else, matching D-13's "no other global settings surface" rule.
- `CreateWebspaceModal.svelte` writes a new empty `[webspaces.<name>]` block through the same `PUT /api/config` seam 07-01/07-02 built and hot-applies it — no `sources` allowlist yet (D-14), submit disables while in flight, a kernel rejection renders verbatim (or the fixed hash-conflict copy) in a destructive `Alert` with the typed name intact.
- `config-edit.ts` centralizes every config-document edit this phase makes (`cloneConfig`/`addWebspace`/`removeWebspace`/`setWebspaceFilter`) behind pure, unit-tested, non-mutating functions — 07-01's inline filter mutation in the webspace route now goes through the same path.
- `/` is redirect-only: remembered webspace (if the kernel still reports it) → first webspace in kernel order → the zero-webspaces empty state, never a redirect loop, never a blank page. The retired card-list home page and its `Card` import are gone.

## Task Commits

Each task was committed atomically:

1. **Task 1: Dialog, dropdown-menu and alert-dialog wrappers on the already-installed bits-ui** — `441176f` (feat)
2. **Task 2: The header title becomes a webspace switcher that can create a webspace** — `424c242` (feat)
3. **Task 3: Root becomes a redirect, and a first-run user has somewhere to start** — `915980d` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified

- `web/src/lib/components/ui/dialog/`, `ui/dropdown-menu/`, `ui/alert-dialog/` (new) - the three overlay primitive wrapper directories, barrel-exported, hand-adapted from bits-ui
- `web/src/lib/components/ui/overlay-primitives.test.ts` (new) - barrel-export, trigger-shape, height-cap, no-raw-hex, and package.json-unchanged guards
- `web/src/lib/config-edit.ts` / `config-edit.test.ts` (new) - `cloneConfig`, `addWebspace`, `removeWebspace`, `setWebspaceFilter`
- `web/src/lib/last-webspace.ts` / `last-webspace.test.ts` (new) - `LAST_WEBSPACE_KEY`, `readLastWebspace`, `writeLastWebspace`, `resolveRedirectTarget`
- `web/src/lib/components/WebspaceSwitcher.svelte` / `webspace-switcher.test.ts` (new) - the header's title drop-down
- `web/src/lib/components/CreateWebspaceModal.svelte` (new) - single-field create-webspace dialog, reused unchanged by the root empty state
- `web/src/lib/components/WebspaceHeader.svelte` - `<h1>` replaced with `<WebspaceSwitcher>`, new `webspaces`/`oncreatewebspace`/`onmanagesources` props
- `web/src/lib/components/save-filter-clone.test.ts` - assertion updated to match `writeFilter`'s delegation to `setWebspaceFilter` (was: literal `$state.snapshot` call)
- `web/src/routes/w/[webspace]/+page.svelte` - webspace list/create-modal wiring, filter writes moved onto `setWebspaceFilter`, `writeLastWebspace(webspace)` called on every visit
- `web/src/routes/+page.svelte` - rewritten as redirect-only with the zero-webspaces empty state; retired card-list markup and `Card` import removed

## Decisions Made

- **Hand-scaffolded the three overlay primitives** directly against `node_modules/bits-ui`'s own type/export shape rather than running `npx shadcn-svelte add` against the live registry — no confirmed network access in this execution environment. The plan itself treats CLI output as "a structural starting point only," hand-adapted to `app.css` tokens regardless of source, so the resulting wrapper API shape (verified against `popover`/`tooltip`'s established pattern) is what actually matters.
- **UI-SPEC ordering deviation** (pre-recorded by the plan itself): the switcher renders webspaces in `GET /api/config`'s own JSON key order, never re-sorted — the real stability guarantee (never re-sorted per render, never reordered by state) holds; literal TOML file-declaration order does not, since a Go map cannot preserve it.
- **`web/src/lib/last-webspace.ts` pulled forward into Task 2's commit** (Rule 3, blocking compile dependency): Task 2's create-webspace flow calls `writeLastWebspace` per the plan's own action text, so the file had to exist before Task 2's `npm run check`/`test` could pass. Built to its full Task 3 spec immediately; Task 3 added only `last-webspace.test.ts`.
- **`cloneConfig` clones via a JSON round trip**, not `structuredClone` or `$state.snapshot()` — `config-edit.ts` is a plain `.ts` module, so Svelte's `$state.snapshot()` rune compiler macro isn't callable there. `JSON.stringify` reads through a Svelte 5 reactive Proxy exactly as a plain property access would, sidestepping 07-01's `structuredClone`-throws-on-Proxy bug without needing a rune-capable file.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking compile dependency] `writeLastWebspace` (Task 3's own file) needed by Task 2**
- **Found during:** Task 2, wiring `CreateWebspaceModal`'s success path in `web/src/routes/w/[webspace]/+page.svelte`
- **Issue:** The plan's own Task 2 action text instructs "Record the created name through `writeLastWebspace` (Task 3's helper)" — but `web/src/lib/last-webspace.ts` is declared as a Task 3 file. Without it, Task 2 cannot compile or pass its own `npm run check`/`test` verification.
- **Fix:** Built `web/src/lib/last-webspace.ts` to its full Task 3 specification (`LAST_WEBSPACE_KEY`, `readLastWebspace`, `writeLastWebspace`, `resolveRedirectTarget`) within Task 2's own commit. Task 3 then only added `last-webspace.test.ts` against the already-complete file — no further implementation changes were needed.
- **Files modified:** `web/src/lib/last-webspace.ts` (created in Task 2 instead of Task 3)
- **Verification:** `npm run check` (0 errors) and `npm run test` both passed at the end of Task 2; Task 3's `last-webspace.test.ts` (10 tests) passes against the unmodified file.
- **Committed in:** `424c242` (Task 2 commit)

**2. [Rule 1 - Bug/regression] `save-filter-clone.test.ts`'s literal `$state.snapshot` assertion broke when `writeFilter` moved to `setWebspaceFilter`**
- **Found during:** Task 2, after moving the filter-write path onto `config-edit.ts`'s `setWebspaceFilter` per the plan's own instruction ("Move 07-01's inline filter mutation in the route onto `setWebspaceFilter`")
- **Issue:** A pre-existing 07-01 regression test asserted `writeFilter`'s source literally contained `$state.snapshot(configResponse.config)`. That call moved into `config-edit.ts`'s `cloneConfig` (as a JSON round trip, not `$state.snapshot`, since `config-edit.ts` is a plain `.ts` module where the rune isn't callable) — the literal assertion now failed even though the invariant it protects (never `structuredClone` the reactive config document) still holds.
- **Fix:** Updated the test's second assertion to check that `writeFilter` delegates to `setWebspaceFilter(configResponse.config, ...)`, and added a new assertion proving `config-edit.ts` itself contains no `structuredClone` call — preserving the file's original purpose (guard the invariant) while matching where the clone mechanism now actually lives.
- **Files modified:** `web/src/lib/components/save-filter-clone.test.ts`
- **Verification:** Full test suite passes (342/342) after the update.
- **Committed in:** `424c242` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (1 blocking compile dependency, 1 regression-test update following an instructed refactor). Both were structurally required by the plan's own action text; no scope creep beyond what Task 2/3 already asked for.

## Known Stubs

- `handleManageSources` in `web/src/routes/w/[webspace]/+page.svelte` is an intentional no-op placeholder for the `Manage sources…` menu item — explicitly specified by the plan ("pass a no-op-safe handler from the header until then, and do not render the item as disabled"). Resolved when 07-05 builds the Manage Sources modal.

## Issues Encountered

- No live `make dev` session (real running kernel) was available in this execution environment to perform the plan's own `<verification>` section's live checks: opening `/` against a real kernel, exercising the switcher against real configured webspaces, creating a webspace and confirming it lands in `config.toml`/the switcher/the current route without a restart, and the "rename every `[webspaces.*]` block, confirm the empty state" check. Every unit-testable seam (overlay primitives, `config-edit.ts`, `last-webspace.ts`, the two structural component guards) is proven by a passing automated test; the live end-to-end interaction against a running kernel is not. Recorded in `coverage` above (D2/D3) as needing human/live-kernel verification before being trusted end to end — same limitation 07-02-SUMMARY.md recorded for its own live-kernel checks.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- The three overlay primitives (`Dialog`, `DropdownMenu`, `AlertDialog`) and `config-edit.ts`'s pure edit helpers are exactly what 07-04 (the "+"-picker / two-step new-instance modal) and 07-05 (chip edit menu, Manage Sources modal, destructive confirmations) build on next — no further primitive work needed before either plan starts.
- `handleManageSources` in the webspace route is a placeholder 07-05 must replace with the real Manage Sources modal open handler.
- Recommend a live-kernel pass (via `/gsd-verify-work` or a manual `make dev` session) before shipping this plan's UI behavior end to end, given the environment limitation noted above — no blocker for continuing to 07-04/07-05, since neither depends on this plan's live-kernel outcome, only on its shipped code shape.

---
*Phase: 07-webspace-builder-ui*
*Completed: 2026-08-08*

## Self-Check: PASSED
