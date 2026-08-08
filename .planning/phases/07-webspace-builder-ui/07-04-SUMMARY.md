---
phase: 07-webspace-builder-ui
plan: 04
subsystem: ui
tags: [svelte5, sveltekit, bits-ui, popover, dialog, dropdown-menu, config, source, secrets]

# Dependency graph
requires:
  - phase: 07-webspace-builder-ui
    provides: "07-01's PUT /api/config (base_hash lock) + 07-02's live apply-on-save, POST /api/config/reload, GET /api/config/plugin-types, POST /api/config/describe-plugin — this plan's every write and every vocabulary lookup rides these seams"
  - phase: 07-webspace-builder-ui
    provides: "07-03's dialog/dropdown-menu/alert-dialog overlay primitives (child({ props }) trigger composition) and config-edit.ts's cloneConfig/addWebspace/removeWebspace/setWebspaceFilter pure-edit discipline"
provides:
  - "web/src/lib/plugin-fields.ts: the static per-plugin-type connection field table (connectionFieldsFor), the comma-separated match-value parser (parseMatchValues), and label helpers (titleCaseField, pluginTypeLabel) — the one place a new plugin type's connection fields get declared"
  - "config-edit.ts's setMatchBlock/addSourceToWebspace/removeSourceFromWebspace/upsertSourceInstance: every remaining pure config-document edit this phase's builder surfaces need, all non-mutating and unit-proven"
  - "MatchFieldsForm.svelte + ConnectionForm.svelte + SecretField.svelte: the three shared form bodies every add/edit flow in this phase (and 07-05) composes from"
  - "AddSourceModal.svelte: the '+' picker (Popover) plus the one-step existing-instance and two-step new-instance add flows, embedded inline in WebspaceHeader's chip row"
  - "EditSourceModal.svelte + SourceChip.svelte's third ⋮ control: edit connection/edit match settings/remove from this webspace, D-12's chip-menu escape hatch"
affects: [07-05]

# Actuals (#2632) — pairs with the plan's estimate to calibrate future estimates.
actuals:
  tokens: 24000
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "AddSourceModal.svelte is mounted DIRECTLY INLINE inside WebspaceHeader.svelte's chip row (not at the route level with an externally-toggled open boolean, unlike CreateWebspaceModal) — bits-ui's Popover.Root/Trigger/Content must live in one component subtree, and the picker's trigger must physically be the '+' button the row's own visibleChipCount overflow math measures. The two Dialog flows inside the same component are unaffected (DialogContent already portals to <body>). Documented in the component's own header comment; within CONTEXT.md's 'Claude's Discretion: picker presentation' grant."
    - "A component that must seed local editable state once from an initial prop (MatchFieldsForm's text, EditSourceModal's connectionValues/matchBlock) does so at $state() declaration time, never reactively re-derived in an $effect keyed on the prop — a reactive re-seed would fight the user's typing every time an onchange round trip updates the parent's own state object. Callers key these components ({#key instance-mode}) so a genuinely different target always mounts fresh."
    - "Every describePlugin call in this plan doubles as a live 'read this plugin's declared match_vocabulary' lookup, not only the two-step flow's Step-1-to-Step-2 handoff it was originally built for (07-02) — both the one-step existing-instance add (AddSourceModal.selectExisting) and the chip menu's 'Edit match settings…' (the route's handleChipEdit) trial-launch the instance's OWN already-stored connection config through this same RPC, since GET /api/sources carries no match_vocabulary field and this plan's files_modified scope carries no kernel/ files to add one."

key-files:
  created:
    - web/src/lib/plugin-fields.ts
    - web/src/lib/plugin-fields.test.ts
    - web/src/lib/components/MatchFieldsForm.svelte
    - web/src/lib/components/AddSourceModal.svelte
    - web/src/lib/components/add-source.test.ts
    - web/src/lib/components/SecretField.svelte
    - web/src/lib/components/secret-field.test.ts
    - web/src/lib/components/ConnectionForm.svelte
    - web/src/lib/components/EditSourceModal.svelte
    - web/src/lib/components/chip-edit-menu.test.ts
  modified:
    - web/src/lib/api.ts
    - web/src/lib/config-edit.ts
    - web/src/lib/config-edit.test.ts
    - web/src/lib/components/WebspaceHeader.svelte
    - web/src/lib/components/SourceChip.svelte
    - web/src/routes/w/[webspace]/+page.svelte

key-decisions:
  - "AddSourceModal architecture (Claude's Discretion, CONTEXT.md): self-contained (owns Popover-picker + both Dialog-flow state internally), mounted inline in WebspaceHeader's chip row rather than route-level with open/onclose props — required by bits-ui's trigger/content co-location constraint plus the row's own overflow-measurement need for the '+' button's real DOM position."
  - "SecretField's set/unset badge reads the caller-supplied envVars prop (the last GET/PUT /api/config response's own env_vars presence map) synchronously, not a per-keystroke debounced network lookup as 07-UI-SPEC.md's prose literally describes — kernel/httpapi/config.go's envVarsIn only ever reports on variable names already referenced in the PERSISTED config document; there is no endpoint that checks an arbitrary not-yet-saved name, and this plan's files_modified scope carries no kernel/ files to add one. A brand-new variable name shows 'Not set' until the instance is actually saved — a conservative, never-falsely-reassuring degradation consistent with D-15's 'informational, never a submit blocker' framing."
  - "The one-step existing-instance add flow and the chip menu's 'Edit match settings…' both resolve match vocabulary via describePlugin trial-launched against the instance's own already-stored Source, not by reading GET /api/sources (which carries no match_vocabulary field today) — same underlying gap and same in-scope substitute as the SecretField decision above."
  - "'Save anyway' (Step 1 Describe failure) transitions to a 'connect-saved' sub-step inside the SAME two-step Dialog (showing the fixed follow-up copy plus a Done button) rather than surfacing the notice in the header region as 07-UI-SPEC.md's prose describes — keeps the notice self-contained in AddSourceModal without new cross-component prop plumbing; the required copy is still shown to the user in full."
  - "Instance id derivation (Step 1 -> id) rejects only what the kernel structurally cannot express (blank id, or one already present in config.sources) — display-name uniqueness and every other rule is left entirely to the kernel's own load-time validator, per the plan's own instruction, so there is exactly one rule set."

requirements-completed: [UI-12]

coverage:
  - id: D1
    description: "The '+' picker offers every configured instance not already in this webspace plus a 'New {plugin type}…' row per discovered-but-unconfigured plugin binary, shows the exact empty-state copy when nothing is left to add, and is height-capped/scrollable"
    requirement: "UI-12"
    verification:
      - kind: unit
        ref: "web/src/lib/components/add-source.test.ts (trigger/picker/one-step-modal describes)"
        status: pass
    human_judgment: true
    rationale: "The structural guard proves the template shape (aria-label, dashed border, empty-copy branch, height-cap classes); the actual open/select/populate interaction against a running kernel was not exercised live — no make dev session was available in this execution environment (same limitation 07-01/07-02/07-03-SUMMARY.md each recorded)."
  - id: D2
    description: "config-edit.ts's setMatchBlock/addSourceToWebspace/removeSourceFromWebspace/upsertSourceInstance are pure, non-mutating, and addSourceToWebspace seeds a previously-absent sources allowlist from every currently participating instance (D-14) without ever being triggered by an unrelated setWebspaceFilter save"
    requirement: "UI-12"
    verification:
      - kind: unit
        ref: "web/src/lib/config-edit.test.ts (setMatchBlock/addSourceToWebspace/removeSourceFromWebspace/upsertSourceInstance describes)"
        status: pass
    human_judgment: false
  - id: D3
    description: "SecretField renders a plain text input holding a variable NAME only (never a password input, never autofill-eligible), and both the Set/Not-set badges render their exact copy gated on a non-blank name"
    requirement: "UI-12"
    verification:
      - kind: unit
        ref: "web/src/lib/components/secret-field.test.ts"
        status: pass
    human_judgment: false
  - id: D4
    description: "The two-step new-instance flow verifies via describePlugin before persisting anything, offers 'Save anyway' with the exact failure copy on a Describe failure, and Step 2 issues exactly one PUT /api/config writing the source block, match block and allowlist together"
    requirement: "UI-12"
    verification:
      - kind: unit
        ref: "web/src/lib/components/add-source.test.ts (two-step-modal/Step-1-failure/Step-2-submit describes)"
        status: pass
    human_judgment: true
    rationale: "The structural guard proves the step-indicator copy, the failure-branch copy/Save-anyway gating, and the single-putConfig-call shape of submitMatch's own source; the actual Describe round trip against a real plugin subprocess (success and failure) was not exercised live in this environment."
  - id: D5
    description: "Every source chip's ⋮ menu offers exactly Edit connection…/Edit match settings…/Remove from this webspace (no instance-deletion item), opening it never toggles the chip's filter state (stopPropagation runs before the trigger's own click handling), and the remove item is destructive-tinted"
    requirement: "UI-12"
    verification:
      - kind: unit
        ref: "web/src/lib/components/chip-edit-menu.test.ts"
        status: pass
      - kind: unit
        ref: "web/src/lib/components/source-chip-pill.test.ts"
        status: pass
      - kind: unit
        ref: "web/src/lib/components/source-chip-selected.test.ts"
        status: pass
    human_judgment: false
  - id: D6
    description: "End-to-end live flows (add an existing instance, connect a new plugin type through both steps, edit a connection, edit match settings, remove a source, confirm no secret value ever reaches the network tab or DOM) against a real running kernel"
    verification: []
    human_judgment: true
    rationale: "No live make dev session was available in this execution environment — every unit-testable seam (plugin-fields.ts, config-edit.ts's four new helpers, all five new/modified components' structural guards) is proven by a passing automated test, but the actual browser<->kernel round trip (real PUT /api/config writes, real describePlugin trial-launches, real env-var badge truth) is not. Same limitation 07-01/07-02/07-03-SUMMARY.md each recorded — recommend a live-kernel pass (via /gsd-verify-work or a manual make dev session) before shipping this phase's UI behavior end to end."

duration: ~1h (three tasks, no checkpoints — autonomous plan)
completed: 2026-08-08
status: complete
---

# Phase 7 Plan 4: Add-Source Picker, Two-Step Connect Flow, and Chip Edit Menu Summary

**A "+" affordance in the chip row that composes a webspace's source set end to end — pick an already-configured instance (one-step match-only modal) or connect a brand-new plugin-type instance (two-step Connect/Match modal with Describe-verify-then-save and a Save-anyway fallback) — plus a "⋮" menu on every chip for editing its connection, editing its match settings, or removing it from the webspace, all built on plugin-fields.ts's static connection-field table, four new pure config-edit.ts helpers, and the env-var-name-only SecretField contract.**

## Performance

- **Duration:** ~1h across three tasks (autonomous execution, no checkpoints)
- **Tasks:** 3
- **Files touched:** 16 (10 created, 6 modified)

## Accomplishments

- `web/src/lib/plugin-fields.ts` is the one honest place a plugin type's connection-field SHAPE lives (`connectionFieldsFor`), since Describe carries match vocabulary only — no connection-field schema exists on the wire. `parseMatchValues` and `titleCaseField` round out the three exported helpers the plan named, plus a `pluginTypeLabel` fallback the picker/Step-1 title needed.
- `config-edit.ts` gains `setMatchBlock`, `addSourceToWebspace` (D-14: seeds a previously-absent `sources` allowlist from every currently-participating instance, never triggered by an unrelated `setWebspaceFilter` save), `removeSourceFromWebspace`, and `upsertSourceInstance` — every one pure, non-mutating, and covered by dedicated unit tests proving the input document is never touched.
- `MatchFieldsForm.svelte` renders one labelled comma-separated input per declared vocabulary field, structurally incapable of emitting an empty value list (a blank field is omitted from the block entirely) — shared unforked by the one-step add, the two-step Match step, and the chip menu's "Edit match settings…".
- `AddSourceModal.svelte` is the "+" picker (a Popover listing unparticipating instances plus "New {plugin type}…" rows, or the exact "All available sources are already in this webspace." copy when nothing is left) and both add flows: a one-step existing-instance modal (match fields only), and a two-step new-instance modal (`1. Connect` / `2. Match` step indicator) that trial-launches via `describePlugin` before persisting anything, offers "Save anyway" on a Describe failure, and writes the source block + match block + allowlist together in exactly one `PUT /api/config` on Step 2 submit.
- `SecretField.svelte` (D-15/T-07-19) is a plain text input holding only an environment variable NAME — never `type="password"`, never autofill-eligible — with a live Set/Not-set badge; `ConnectionForm.svelte` renders `connectionFieldsFor`'s table in order, routing secret fields through `SecretField` and non-advanced/non-secret fields as plain labelled inputs, with an Advanced-options disclosure for the sync-interval override.
- `SourceChip.svelte` gains a third `size-8` `⋮` control (stopPropagation before the trigger's own click handling — the single most important line, since opening the menu must never also toggle the chip's filter) offering `Edit connection…`/`Edit match settings…`/`Remove from this webspace`, with no instance-deletion item (that stays the sole province of 07-05's Manage Sources modal). `EditSourceModal.svelte` reuses `ConnectionForm`/`MatchFieldsForm` unforked for the two edit flows; "Remove from this webspace" is a modal-less write reusing the header's existing `filterBusy`/`filterError` surface.

## Task Commits

Each task was committed atomically:

1. **Task 1: The "+" picker and adding an already-configured instance to this webspace** — `41abecd` (feat)
2. **Task 2: Connect a brand-new instance — two-step modal with the secret-field contract** — `e3030d5` (feat)
3. **Task 3: The chip's ⋮ menu — edit connection, edit match settings, remove from this webspace** — `6700d49` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified

- `web/src/lib/plugin-fields.ts` / `plugin-fields.test.ts` (new) - `connectionFieldsFor`, `parseMatchValues`, `titleCaseField`, `pluginTypeLabel`
- `web/src/lib/config-edit.ts` / `config-edit.test.ts` - `setMatchBlock`, `addSourceToWebspace`, `removeSourceFromWebspace`, `upsertSourceInstance` added to the existing pure-edit module
- `web/src/lib/components/MatchFieldsForm.svelte` (new) - the shared vocabulary-driven match form
- `web/src/lib/components/AddSourceModal.svelte` / `add-source.test.ts` (new) - the picker plus both add flows
- `web/src/lib/components/SecretField.svelte` / `secret-field.test.ts` (new) - env-var-name-only secret input with set/unset badge
- `web/src/lib/components/ConnectionForm.svelte` (new) - static per-plugin-type connection field form
- `web/src/lib/components/EditSourceModal.svelte` (new) - chip menu's Edit connection/Edit match settings modals
- `web/src/lib/components/SourceChip.svelte` - third `⋮` edit-menu control, `onedit` prop
- `web/src/lib/components/chip-edit-menu.test.ts` (new) - the chip's edit-menu structural guard
- `web/src/lib/components/WebspaceHeader.svelte` - `+` add-source trigger wired into the chip row's own overflow measurement, `onedit` threaded to every real `SourceChip`
- `web/src/lib/api.ts` - `listPluginTypes`, `describePlugin` client functions + interfaces
- `web/src/routes/w/[webspace]/+page.svelte` - `pluginTypes` fetch, add-source/chip-edit/remove-source wiring

## Decisions Made

- **AddSourceModal architecture** (Claude's Discretion, CONTEXT.md "picker presentation"): self-contained, mounted inline in WebspaceHeader's chip row rather than route-level with `open`/`onclose` props — bits-ui's Popover trigger/content must be co-located in one component subtree, and the trigger must physically be the row's own measured `+` button.
- **SecretField's badge reads the caller-supplied `envVars` snapshot synchronously**, not a per-keystroke debounced network lookup — `kernel/httpapi/config.go`'s `envVarsIn` only ever reports on names already referenced in the PERSISTED config, and this plan's `files_modified` scope carries no `kernel/` files to add a live-arbitrary-name lookup endpoint. A brand-new variable name conservatively shows "Not set" until the instance is saved.
- **The one-step existing-instance add and the chip menu's "Edit match settings…" both resolve vocabulary via `describePlugin`** against the instance's own stored config, not by reading `GET /api/sources` (no `match_vocabulary` field there today) — same substitute, same underlying gap.
- **"Save anyway" transitions to an in-Dialog "connect-saved" confirmation** (fixed copy + Done button) rather than a header-region notice — keeps the required copy visible without new cross-component prop plumbing.
- **Instance-id derivation rejects only what the kernel structurally cannot express** (blank id, or one already present) — every other rule, including display-name uniqueness, is left to the kernel's own load-time validator.

## Deviations from Plan

### Auto-fixed Issues

None — every adaptation above was a structural/architectural necessity flagged and reasoned through at design time (all within CONTEXT.md's pre-granted "Claude's Discretion" for modal/form layout and picker presentation), not a bug fixed after the fact. No Rule 1/2/3 auto-fixes were needed.

---

**Total deviations:** 0 auto-fixed. Three deliberate, documented architecture adaptations (see Decisions Made) made necessary by (a) bits-ui's Popover trigger/content co-location constraint and (b) `GET /api/sources` lacking a `match_vocabulary` field — both real gaps between the plan's prose and the actual available surface, resolved within this plan's own frontend-only `files_modified` scope rather than reaching into `kernel/` files not listed there.

## Known Stubs

None — every flow this plan builds (picker, one-step add, two-step add, edit connection, edit match settings, remove-from-webspace) is a real, working implementation against the actual kernel API surface, exercised by its own test.

## Issues Encountered

- No live `make dev` session was available in this execution environment to perform the plan's own `<verification>` section's live-kernel checks (add an existing instance and confirm the chip appears without a restart; connect a new plugin type through both steps and confirm `config.toml` gains the three blocks in one write; confirm a secret field never leaks a value into the network tab or DOM; confirm the chip's `⋮` menu never toggles its filter). Every unit-testable seam is proven by a passing automated test against the real component source (structural guards) and pure logic (config-edit.ts); the live browser<->kernel round trip is not — same environment limitation 07-01/07-02/07-03-SUMMARY.md each recorded. Recorded in `coverage` above (D1, D4, D6) as needing human/live-kernel verification.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `MatchFieldsForm.svelte`, `ConnectionForm.svelte`, and `plugin-fields.ts`'s connection-field table are exactly what 07-05's Manage Sources modal (instance deletion, webspace deletion, Reload config) needs underneath it for its own "Edit" affordance — no further primitive work needed before that plan starts.
- `EditSourceModal.svelte`'s 'connection' mode is already the exact component 07-05's Manage Sources "Edit" button should open (same props shape) — reuse directly rather than forking.
- Recommend a live-kernel pass (via `/gsd-verify-work` or a manual `make dev` session) before shipping this plan's UI behavior end to end, given the environment limitation noted above — no blocker for continuing to 07-05, since it depends only on this plan's shipped code shape, not its live-kernel outcome.

---
*Phase: 07-webspace-builder-ui*
*Completed: 2026-08-08*

## Self-Check: PASSED
