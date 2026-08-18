---
phase: 12-filesystem-source
plan: 04
subsystem: ui
tags: [svelte5, shadcn-svelte, bits-ui, checkbox, connection-form, e2e, playwright]

requires:
  - phase: 12-filesystem-source
    provides: "12-01's file://-scheme deep-link convention and kernel-mediated open route; 12-02's document scope/preview shapes; 12-03's typed recursive config key threaded end to end into the plugin subprocess, which this plan's checkbox writes to from the UI"

provides:
  - "web/src/lib/components/ui/checkbox/ — the shadcn-svelte official Checkbox primitive wrapper, the first new ui/* block since Phase 11"
  - "ConnectionField.kind ('text'|'checkbox', defaults to text) and ConnectionField.helperText — a reusable, plugin-agnostic boolean field kind any future plugin's connection row can use with zero new component code"
  - "CONNECTION_FIELDS['topos-plugin-filesystem'] — the filesystem plugin's complete connection row (Display Name, required Local Path with no defaultValue, checkbox-kind Include subfolders with helper text, advanced Sync Interval Override)"
  - "ConnectionForm.svelte's third field.kind === 'checkbox' render branch, boolFieldValue's unset/non-boolean-to-false coercion, and setField widened to string | boolean"
  - "web/e2e/specs/12-filesystem-add-source.spec.ts — success criterion 1 driven end to end through the real UI against a real kernel and a real topos-plugin-filesystem binary"

affects: [12-05]

actuals:
  tokens: 7300
  tasks: 3
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Reusable ConnectionField.kind extension (defaults to 'text') — any future plugin's boolean-shaped connection field reuses this branch and the Checkbox primitive with zero new ConnectionForm.svelte code"
    - "A plugin-agnostic KNOWN_UI_PRIMITIVES allowlist test (save-state.test.ts) exists specifically to catch an unreviewed new ui/ primitive directory — a legitimate new primitive must be added there explicitly, which this plan's Task 2 did"

key-files:
  created:
    - web/src/lib/components/ui/checkbox/checkbox.svelte
    - web/src/lib/components/ui/checkbox/index.ts
    - web/src/lib/components/connection-checkbox.test.ts
    - web/e2e/specs/12-filesystem-add-source.spec.ts
  modified:
    - web/src/lib/plugin-fields.ts
    - web/src/lib/plugin-fields.test.ts
    - web/src/lib/api.ts
    - web/src/lib/components/ConnectionForm.svelte
    - web/src/lib/components/save-state.test.ts

key-decisions:
  - "Task 2's TDD RED/GREEN split followed the plan's <behavior> block literally: plugin-fields.test.ts's filesystem-row assertions and connection-checkbox.test.ts's not-yet-written boolFieldValue/setField/branch assertions were committed failing before any implementation, then made to pass in a single GREEN commit."
  - "The e2e spec seeds one pre-existing topos-plugin-mock instance (never attached to this webspace's match block) rather than a genuinely zero-source starting config: WebspaceHeader.svelte's chip row — and the '+' Add-source trigger living inside it — only renders once at least one source is configured system-wide (shouldShowSourceRows). A truly empty config has no chip row to click '+' from at all; this is a pre-existing app property, not something this plan changed, and the plan's own acceptance criterion (\"no source of plugin type filesystem\") is satisfied without it."

requirements-completed: [SRC-04]

coverage:
  - id: D1
    description: "The Checkbox primitive is installed under web/src/lib/components/ui/checkbox/, wrapping bits-ui's Checkbox with the stock unchecked-border/checked-fill mapping onto the existing --border/--primary tokens, and components.json is unchanged"
    requirement: SRC-04
    verification:
      - kind: unit
        ref: "npm --prefix web run check (0 errors) — checkbox.svelte compiles and type-checks against bits-ui's CheckboxRootProps"
        status: pass
      - kind: other
        ref: "git diff --stat web/components.json shows no change"
        status: pass
    human_judgment: false

  - id: D2
    description: "ConnectionField gains optional kind ('text'|'checkbox', defaulting to text) and helperText properties; every pre-existing plugin row reports an absent (undefined) kind — no existing plugin type's form changed"
    requirement: SRC-04
    verification:
      - kind: unit
        ref: "web/src/lib/plugin-fields.test.ts — 'ConnectionField.kind — every pre-existing plugin row reports an absent field kind'"
        status: pass
    human_judgment: false

  - id: D3
    description: "The topos-plugin-filesystem connection row lists Display Name, required Local Path (two-example placeholder, no defaultValue), checkbox-kind Include subfolders (not required, exact helper text), and the shared advanced Sync Interval Override, in that order — required flags derived from plugins/filesystem/main.go's own fatal guard (path only)"
    requirement: SRC-04
    verification:
      - kind: unit
        ref: "web/src/lib/plugin-fields.test.ts — 'topos-plugin-filesystem connection row (12-04-PLAN.md Task 2)' (4 cases, incl. table-truth)"
        status: pass
      - kind: unit
        ref: "web/src/lib/plugin-fields.test.ts — 'defaultConnectionValues: topos-plugin-filesystem returns only the plugin key' and 'missingRequiredFields: topos-plugin-filesystem'"
        status: pass
    human_judgment: false

  - id: D4
    description: "ConnectionForm.svelte's checkbox branch: an unset/non-boolean stored value renders unchecked (never indeterminate), toggling emits a boolean under the field's own key via a boolean-widened setField, helper text renders only when declared, and the whole min-h-11 label row (not only the 16px control) is the clickable target"
    requirement: SRC-04
    verification:
      - kind: unit
        ref: "web/src/lib/components/connection-checkbox.test.ts (11 cases)"
        status: pass
    human_judgment: false

  - id: D5
    description: "A user picks the filesystem plugin type from the source picker, fills in a local path, chooses whether to include subfolders, saves, and the resulting source syncs its documents into the matching webspace — driven entirely from the UI, proving the checkbox's value actually reached the launched plugin subprocess (a nested document appears only because Include subfolders was ticked)"
    requirement: SRC-04
    verification:
      - kind: e2e
        ref: "web/e2e/specs/12-filesystem-add-source.spec.ts (make e2e E2E_ARGS='e2e/specs/12-filesystem-add-source.spec.ts')"
        status: pass
    human_judgment: false

  - id: D6
    description: "The extras block has no independent scroll or height cap beyond the enclosing dialog's own — fine at realistic key counts, unverified at pathological ones, and not filesystem-specific"
    verification: []
    human_judgment: true
    rationale: "12-UI-SPEC.md itself scopes this as a backstop truth, carried forward unresolved from 12-01/12-02/12-03 — not newly introduced by this plan."

duration: ~55min
completed: 2026-08-14
status: complete
---

# Phase 12 Plan 04: The Filesystem Connection Form Summary

**The filesystem plugin type now has a complete connection form — a required Local Path field with no seeded default and an "Include subfolders" checkbox (a new, reusable boolean field kind in the shared connection-field vocabulary) — proven end to end by an e2e spec that adds a folder as a source through the real UI and watches a nested document reach the stream because the checkbox was ticked.**

## Performance

- **Duration:** ~55 min
- **Tasks:** 3
- **Files modified:** 9 (4 created, 5 modified)

## Accomplishments

- Installed the shadcn-svelte official `checkbox` block under `web/src/lib/components/ui/checkbox/`, wrapping `bits-ui`'s `Checkbox` primitive with the stock unchecked-border/checked-fill mapping onto the existing `--border`/`--primary` tokens — no colour override, `components.json` unchanged.
- Extended `ConnectionField` (`plugin-fields.ts`) with optional `kind` (`'text' | 'checkbox'`, defaulting to `'text'`) and `helperText` — every existing plugin row omits both and keeps rendering byte-identically, proven by a regression test walking every pre-existing plugin binary.
- Added the `topos-plugin-filesystem` row to `CONNECTION_FIELDS`: Display Name, a required `path` field labelled "Local Path" with the two-example placeholder and deliberately no `defaultValue` (an arbitrary folder has no universally correct one), a checkbox-kind `recursive` field labelled "Include subfolders" with the exact helper text, and the shared advanced Sync Interval Override — required flags derived from `plugins/filesystem/main.go`'s own fatal guard (path only; `recursive` has none).
- `SourceConfig` gained `recursive?: boolean`, the frontend half of `config.Source.Recursive`.
- `ConnectionForm.svelte` gained a third `field.kind === 'checkbox'` branch (ordered before the plain-text `{:else}` so Svelte's `{:else if}` chain is valid): `boolFieldValue` coerces an unset/non-boolean value to unchecked, `setField` widened to `string | boolean`, a `min-h-11` wrapping `<label>` keeps the whole row — not just the 16px control — the clickable target, and a muted helper paragraph renders only when `field.helperText` is set.
- New `web/e2e/specs/12-filesystem-add-source.spec.ts`: opens the add-source picker, selects "Filesystem folder," asserts the empty-path block and the initially-unchecked checkbox, fills in a real corpus path, ticks "Include subfolders," saves through the Match step, and asserts a nested document (one level below the configured root) reaches the webspace stream while its top-level sibling — outside the webspace's own folder-keyword match — does not, proving the checkbox's value reached the launched plugin subprocess rather than only the form.

## Task Commits

1. **Task 1: The Checkbox primitive and the reusable checkbox field kind** — `3dc2734` (feat)
2. **Task 2 (RED): failing tests for the filesystem connection row and checkbox branch** — `4fc98b8` (test)
3. **Task 2 (GREEN): the filesystem connection row and ConnectionForm checkbox branch** — `fe5ceed` (feat)
4. **Task 3: Add a folder as a source through the real UI, end to end** — `82288df` (test)

**Plan metadata:** this SUMMARY's own commit (pending, see below)

## Files Created/Modified

- `web/src/lib/components/ui/checkbox/checkbox.svelte`, `index.ts` — the new Checkbox primitive
- `web/src/lib/plugin-fields.ts` — `ConnectionField.kind`/`helperText`, the `topos-plugin-filesystem` row, `PLUGIN_TYPE_LABELS` entry
- `web/src/lib/plugin-fields.test.ts` — the absent-field-kind regression test plus the filesystem row's full behavior coverage
- `web/src/lib/api.ts` — `SourceConfig.recursive?: boolean`
- `web/src/lib/components/ConnectionForm.svelte` — the checkbox render branch, `boolFieldValue`, boolean-widened `setField`
- `web/src/lib/components/connection-checkbox.test.ts` — the new branch's full behavior contract (11 cases)
- `web/src/lib/components/save-state.test.ts` — `'checkbox'` added to `KNOWN_UI_PRIMITIVES` (deviation, see below)
- `web/e2e/specs/12-filesystem-add-source.spec.ts` — the new end-to-end spec

## Decisions Made

- **TDD RED/GREEN split for Task 2:** wrote `plugin-fields.test.ts`'s filesystem-row assertions and `connection-checkbox.test.ts` in full before any implementation, confirmed both suites failed for the expected reasons (missing row, missing `boolFieldValue`/branch), committed that as the RED commit, then implemented the row/branch and confirmed GREEN in a second commit — matching the plan's own `<behavior>` block verbatim.
- **e2e spec seeds one pre-existing `topos-plugin-mock` instance:** `WebspaceHeader.svelte`'s chip row (and the "+" Add-source trigger inside it) only renders once at least one source is configured system-wide (`shouldShowSourceRows`, `web/src/lib/format.ts`) — a genuinely empty starting config has no chip row to click "+" from. The seeded mock instance is never attached to the spec's webspace match block and its fixed corpus carries no "archive"-labelled item, so it contributes nothing to the assertions; the plan's own acceptance criterion ("no source of plugin type filesystem is present in the spec's starting config") is unaffected.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] Added `'checkbox'` to `save-state.test.ts`'s `KNOWN_UI_PRIMITIVES` allowlist**
- **Found during:** Task 2, running `npm --prefix web run test` after adding the Checkbox primitive and wiring it into `ConnectionForm.svelte`
- **Issue:** `save-state.test.ts` carries a deliberate allowlist test (`'no toast anywhere...'`) that fails the build on any import from an unrecognised `ui/` primitive directory — it correctly flagged the new `ui/checkbox` import from both `ConnectionForm.svelte` and the new test file the moment they landed.
- **Fix:** Added `'checkbox'` to `KNOWN_UI_PRIMITIVES`, in alphabetical position alongside the existing entries.
- **Files modified:** `web/src/lib/components/save-state.test.ts`
- **Verification:** `npm --prefix web run test` passes in full (955/955) after the update.
- **Committed in:** `fe5ceed` (Task 2 GREEN commit)

---

**Total deviations:** 1 auto-fixed (Rule 3 — an existing guard test correctly caught the new, legitimate `ui/` primitive and needed its allowlist updated; not a bug in the new code itself)
**Impact on plan:** Necessary maintenance of an existing guard, not scope creep — the guard's own design intent (force a reviewed decision on any new `ui/` primitive) was honored, not bypassed.

## Issues Encountered

- `npm run build` (invoked by `make e2e`) again overwrote the gitignored `kernel/webui/build/.gitkeep` placeholder — restored via `git checkout -- kernel/webui/build/.gitkeep` before committing Task 3, same as every prior plan in this phase.
- The e2e spec's first draft used a genuinely zero-source starting config, which left no way to reach the "Add source" trigger in the browser at all (`WebspaceHeader.svelte`'s chip row is gated on at least one source existing system-wide) — diagnosed via the failure screenshot and fixed by seeding one unrelated `topos-plugin-mock` instance (see Decisions Made above).

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- The filesystem plugin type now has a complete, correct, UI-proven connection form; the checkbox field kind is reusable by any future plugin with zero new component code.
- 12-05 (docs, and this phase's remaining wave) can build on top without touching this plan's own files again.
- D6 (the extras block's unverified-at-pathological-key-counts overflow backstop, carried forward from 12-01/12-02/12-03) remains genuinely unverified — untouched by this plan's scope.

---
*Phase: 12-filesystem-source*
*Completed: 2026-08-14*
