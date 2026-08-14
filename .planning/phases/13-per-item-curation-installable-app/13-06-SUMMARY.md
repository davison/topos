---
phase: 13-per-item-curation-installable-app
plan: 06
subsystem: ui
tags: [svelte, source-chip, trust-tier, playwright, e2e, docs]

requires:
  - phase: 13-per-item-curation-installable-app
    provides: "13-05: kernel/pluginhost link-time build manifest (VerifyTrustedBinary, launch_failure=manifest_unverified, launch_advisory=shadowed) and the docs/api.md wire contract for both fields"
provides:
  - "web/src/lib/api.ts: SourceStatus.launch_failure widened with 'manifest_unverified'; new SourceStatus.launch_advisory ('' | 'shadowed')"
  - "web/src/lib/format.ts: healthTone/isAdvisoryOnly extended with both new trust states as inputs to the existing precedence chain (no parallel gate)"
  - "web/src/lib/components/SourceChip.svelte: isManifestUnverified/isShadowed derivations, two new contract-exact tooltip sentences, chip menu re-pin action unchanged (isPinMismatch-gated only)"
  - "web/e2e/fixtures/plugin-binaries.ts: linkPluginBinaryAs (arbitrary source path -> caller-chosen destination name in a fixture's TRUSTED directory)"
  - "web/e2e/fixtures/config-builder.ts: FixtureConfigSpec.trustedBinaryLinks"
  - "web/e2e/specs/13-manifest-unverified.spec.ts, web/e2e/specs/13-shadowed-advisory.spec.ts"
  - "docs/plugin-contract.md: integrity-control-not-authentication limitation paragraph; docs/plugins/signal.md: D-15 local-build/consent-and-pin section; docs/testing.md: spec index entries"
affects: [14-google-drive]

actuals:
  tokens: 12000
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "linkPluginBinaryAs — the general (arbitrary-source-path, caller-chosen-destination-name) form of the e2e fixture's existing name-preserving linkPluginBinaries, for a spec proving a file-drop-style bypass under a name the trust manifest never covers"

key-files:
  created:
    - web/e2e/specs/13-manifest-unverified.spec.ts
    - web/e2e/specs/13-shadowed-advisory.spec.ts
  modified:
    - web/src/lib/api.ts
    - web/src/lib/format.ts
    - web/src/lib/components/SourceChip.svelte
    - web/src/lib/components/match-advisory.test.ts
    - web/src/lib/components/source-chip-tooltip.test.ts
    - web/e2e/fixtures/plugin-binaries.ts
    - web/e2e/fixtures/config-builder.ts
    - web/e2e/fixtures/kernel.ts
    - docs/plugin-contract.md
    - docs/plugins/signal.md
    - docs/testing.md

key-decisions:
  - "Both new trust states feed the SAME healthTone/isAdvisoryOnly precedence chain as pin_mismatch/last_notice — manifest_unverified sits beside pin_mismatch (destructive, head of chain), shadowed sits beside last_notice (warning, last among problem states, before success) — never a parallel gate (T-13-28)."
  - "isAdvisoryOnly widened to clear BOTH last_notice and launch_advisory together when re-asking healthTone, so neither becomes a dead branch — the identical trap Phase 12's CR-01 fix already documents, now applied to the second advisory input."
  - "Did NOT implement the UAT-flagged native-title-tooltip suppression (see Deviations) — a real conflict with an existing, deliberately-locked touch-accessibility requirement was found and the fix was deferred rather than forced through."
  - "kernel.ts fixture wiring for trustedBinaryLinks was added even though the plan's Task 2 <files> list omitted it (Rule 3: the new config-builder field is inert without it)."

patterns-established:
  - "linkPluginBinaryAs (web/e2e/fixtures/plugin-binaries.ts) — link an arbitrary source path into a fixture directory under a name that does NOT match the source's own basename; use this whenever a spec needs to prove a name-based trust/verification gate rather than a directory-based one."

requirements-completed: [PLUG-07]

coverage:
  - id: D1
    description: "A trusted-directory binary that fails build-manifest verification renders a destructive-tone chip with the contract-exact tooltip, no reachable probe, and no re-pin remedial action; the refusal is also named in the kernel's own log (D-12/D-13)"
    requirement: PLUG-07
    verification:
      - kind: unit
        ref: "web/src/lib/components/match-advisory.test.ts (healthTone: manifest_unverified (D-12/D-13) describe block)"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/13-manifest-unverified.spec.ts"
        status: pass
    human_judgment: false
  - id: D2
    description: "A trusted binary shadowing a same-named pinned external plugin renders a warning-tone chip (never destructive) with the contract-exact shadowing tooltip, and the source still launches and syncs (D-14)"
    requirement: PLUG-07
    verification:
      - kind: unit
        ref: "web/src/lib/components/match-advisory.test.ts (healthTone: shadowed advisory (D-14) describe block)"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/13-shadowed-advisory.spec.ts"
        status: pass
    human_judgment: false
  - id: D3
    description: "Both new states are new inputs to the single healthTone/isAdvisoryOnly precedence chain — a real problem (pin mismatch, manifest-unverified, unreachable, errored, never-synced) always outranks the shadowed advisory, and a launch refusal always outranks it too"
    requirement: PLUG-07
    verification:
      - kind: unit
        ref: "web/src/lib/components/match-advisory.test.ts (precedence coexistence cases: pin_mismatch+shadowed, manifest_unverified+shadowed, unreachable+shadowed, error+shadowed, never-synced+shadowed)"
        status: pass
    human_judgment: false
  - id: D4
    description: "The published plugin contract states manifest membership (not directory location) determines trust, states the refuse-to-load behaviour including the trial launch, and carries an explicit integrity-control-not-authentication limitation; docs/plugins/signal.md explains D-15 with a step-by-step consent-and-pin remedy; both cross-link"
    requirement: PLUG-07
    verification:
      - kind: other
        ref: "make docs-check; grep -q 'build-provenance manifest' docs/plugin-contract.md; grep -n 'manifest_unverified' docs/api.md; grep -n '13-manifest-unverified' docs/testing.md; grep -n '13-shadowed-advisory' docs/testing.md"
        status: pass
    human_judgment: false

duration: ~2h
completed: 2026-08-14
status: complete
---

# Phase 13 Plan 06: Trust-tier chip visibility, real-binary e2e proof, republished docs Summary

**The two new build-manifest trust states (manifest-unverified refusal, shadowed-by-trusted advisory) now render through SourceChip's existing healthTone/isAdvisoryOnly precedence chain, proven end to end in the browser against real binaries (no new plugin built), with the plugin contract and Signal docs republished to describe the trust system that actually ships.**

## Performance

- **Duration:** ~2h
- **Tasks:** 3
- **Files modified:** 14 (2 created, 12 modified — 11 declared in the plan, kernel.ts added as necessary plumbing per Rule 3)

## Accomplishments

- `web/src/lib/api.ts`'s `SourceStatus.launch_failure` widened with `'manifest_unverified'`, and a new `launch_advisory?: '' | 'shadowed'` field added, both documented against the closed-vocabulary/branch-on-field-not-text discipline the neighbouring fields already carry.
- `web/src/lib/format.ts`'s `healthTone` gained the manifest-unverified check beside the existing pin-mismatch check (destructive, head of the chain) and the shadowed-advisory check in the same position `last_notice` already occupies (warning, last among problem states, immediately before success) — `isAdvisoryOnly` widened to a second input (`launch_advisory`), clearing both inputs together when re-asking `healthTone` to avoid the CR-01 dead-branch trap.
- `web/src/lib/components/SourceChip.svelte` gained `isManifestUnverified`/`isShadowed` derivations mirroring `isPinMismatch`'s exact shape, and `tooltipText` gained the two contract-exact sentences from 13-UI-SPEC.md's Copywriting Contract — the chip menu's re-pin action stays gated on `isPinMismatch` alone, with no new menu item, dialog, or interstitial for either new state.
- `web/src/lib/components/match-advisory.test.ts` and `source-chip-tooltip.test.ts` extended with the full precedence matrix from the plan's `<behavior>` bullets, including every coexistence case (pin-mismatch outranks shadowed, manifest-unverified outranks shadowed, unreachable outranks shadowed, errored outranks shadowed, never-synced outranks shadowed) and the branch-selection matrix mirror function.
- `web/e2e/fixtures/plugin-binaries.ts` gained `linkPluginBinaryAs`, the general (arbitrary source path, caller-chosen destination name) form of the existing name-preserving `linkPluginBinaries` — threaded through `config-builder.ts`'s new `trustedBinaryLinks` field and `kernel.ts`'s fixture wiring.
- Two new Playwright specs, both driven entirely by binaries the existing `make e2e` recipe already builds (`git diff Makefile` is empty): `13-manifest-unverified.spec.ts` links the real out-of-repo `topos-plugin-external-demo` binary into the hermetic kernel's trusted directory under a name absent from the build manifest, proving the destructive chip, the contract-exact tooltip, no reachable probe, no re-pin action, AND the kernel's own captured log naming the refusal (D-13's log-AND-UI requirement); `13-shadowed-advisory.spec.ts` links `topos-plugin-mock` into both the trusted and external directories under the same name, proving the warning-tone (explicitly not destructive) chip and contract-exact tooltip on a source that still launches and syncs.
- `docs/plugin-contract.md` gained an explicit "integrity control, not publisher authentication" limitation paragraph (mirroring `binaryhash.go`'s own doc-comment framing) and the literal "build-provenance manifest" phrase; `docs/plugins/signal.md` gained a full D-15 section (why a locally-built Signal binary refuses to load against a release kernel, and the step-by-step consent-and-pin fix); both cross-link; `docs/testing.md` gained index entries for both new specs. `docs/api.md` already carried `launch_advisory`/`manifest_unverified` from 13-05 — verified via grep, no edit needed.

## Task Commits

1. **Task 1: Two new named chip states, fed through the existing precedence chain** - `37ef689` (feat)
2. **Task 2: Browser specs driving both states with real binaries** - `828236c` (test)
3. **Task 3: Republish the plugin contract, the Signal guidance, and the API surface** - `fc038ea` (docs)

## Files Created/Modified

- `web/src/lib/api.ts` - widened `launch_failure`, new `launch_advisory` field, both documented
- `web/src/lib/format.ts` - `healthTone`/`isAdvisoryOnly` extended with the two new states as chain inputs
- `web/src/lib/components/SourceChip.svelte` - `isManifestUnverified`/`isShadowed` derivations, two new tooltip sentences
- `web/src/lib/components/match-advisory.test.ts` - full behavioural/structural/branch-selection matrix for both states
- `web/src/lib/components/source-chip-tooltip.test.ts` - the two contract-exact sentences, byte-for-byte
- `web/e2e/fixtures/plugin-binaries.ts` - `linkPluginBinaryAs`
- `web/e2e/fixtures/config-builder.ts` - `FixtureConfigSpec.trustedBinaryLinks`
- `web/e2e/fixtures/kernel.ts` - wires `trustedBinaryLinks` into `launchKernel` (Rule 3 addition, see Deviations)
- `web/e2e/specs/13-manifest-unverified.spec.ts` - D-12/D-13 proof
- `web/e2e/specs/13-shadowed-advisory.spec.ts` - D-14 proof
- `docs/plugin-contract.md` - limitation paragraph, "build-provenance manifest" phrase, cross-link to Signal
- `docs/plugins/signal.md` - new "Local builds and the build manifest" section (D-15)
- `docs/testing.md` - spec index entries for both new specs

## Decisions Made

- Both new states extend the ONE `healthTone`/`isAdvisoryOnly` chain rather than adding a parallel gate — `manifest_unverified` at the head (beside `pin_mismatch`), `shadowed` at the tail (beside `last_notice`) — matching the plan's explicit instruction and T-13-28's threat mitigation.
- `isAdvisoryOnly`'s early-return and re-ask both widened to a SECOND input (`launch_advisory`) rather than adding a sibling function, so a future health-adjacent surface has exactly one precedence authority to consult, per the CR-01 discipline this plan explicitly carries forward.
- `linkPluginBinaryAs` was added as a genuinely new, general helper (not a parameter bolted onto `linkPluginBinaries`) because the two functions have different contracts: one preserves the source basename across directories, the other lets the destination name diverge from the source path entirely — conflating them would have made the common case (`linkPluginBinaries`) harder to read for the sake of the rare case.
- `kernel.ts` was added to the file set actually touched (not declared in the plan's Task 2 `<files>` list) because `config-builder.ts`'s new `trustedBinaryLinks` field is otherwise dead — nothing reads it. This is a Rule 3 (blocking issue) auto-fix: the plan's declared files omitted the one file that makes the new fixture field functional.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `web/e2e/fixtures/kernel.ts` needed the new `trustedBinaryLinks` field wired into `launchKernel`**

- **Found during:** Task 2, writing `13-manifest-unverified.spec.ts`
- **Issue:** The plan's Task 2 `<files>` list named `plugin-binaries.ts`, `config-builder.ts`, and the two new spec files, but not `kernel.ts` — the file that actually calls `linkPluginBinaries`/`linkPluginBinaryAs` when a kernel fixture boots. Without touching it, a fixture's `trustedBinaryLinks` field would be silently ignored.
- **Fix:** Imported `linkPluginBinaryAs` and added a loop over `configSpec.trustedBinaryLinks ?? []` immediately after the existing `linkPluginBinaries` calls in `launchKernel`.
- **Files modified:** `web/e2e/fixtures/kernel.ts`
- **Verification:** `13-manifest-unverified.spec.ts` passes, proving the field is actually consumed; `npm run check:e2e` reports zero errors.
- **Committed in:** `828236c` (Task 2 commit)

**2. [Rule 3 - Blocking] Both new fixture webspaces needed a `keywords` fallback**

- **Found during:** Task 2, first `make e2e` run against the two new specs
- **Issue:** `FixtureWebspaceSpec { sources: [...] }` alone is insufficient — `kernel/config.Validate` rejects a webspace declaring neither a `keywords` fallback nor a `match` block (`topos: config: webspace "..." declares neither a keywords fallback nor any match block`), which the plan's own action text did not call out.
- **Fix:** Added `keywords: ['demo']` to both fixture webspace specs — matches the mock plugin's own default corpus labels, consistent with how every other mock-backed fixture in this harness resolves membership.
- **Files modified:** `web/e2e/specs/13-manifest-unverified.spec.ts`, `web/e2e/specs/13-shadowed-advisory.spec.ts`
- **Verification:** both specs pass; `make e2e` passes end to end (135/135).
- **Committed in:** `828236c` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 3 — blocking issues necessary to make the plan's own declared scope actually work).
**Impact on plan:** No scope creep — both fixes are plumbing/config corrections required for the plan's own declared deliverables (the new fixture field, the two new specs) to function at all.

## UAT Context Follow-ups (not folded in — see reasoning below)

The orchestrator's `<uat_context>` block asked this plan to weigh three live-UAT observations from the 13-04 checkpoint. Two are documented here rather than acted on:

1. **Scheduler retries a refused plugin forever** (`scheduled sync failed to dispatch: source=X error="syncer: unknown source"`, logged every sync interval). Out of this plan's declared file scope (`kernel/syncer` is untouched by any of the three tasks) — per the orchestrator's own instruction, **not** acted on, only recorded here as a known follow-up. A future plan bounding or quieting this log line should scope it to `kernel/syncer/scheduler.go`.

2. **Native browser `title`/`alt` tooltips duplicating/covering the CSS popover on source chips** (captured todo `.planning/todos/pending/2026-08-14-suppress-native-tooltips-under-chip-popovers.md`, which explicitly named `SourceChip.svelte` as in this plan's scope). **Investigated and deliberately NOT implemented** — a genuine conflict was found with an existing, deliberately-locked design decision:
   - `web/src/lib/components/source-chip-pill.test.ts`'s "touch health detail: the filter button carries a native title" test structurally asserts `title=` is present on the filter button, with an explicit rationale: *"health detail is otherwise unreachable without hover, and a native title is the long-press-accessible touch degrade"* (09.1-04-PLAN.md's own R2 planner resolution).
   - `web/e2e/specs/11-binary-changed-repin.spec.ts` asserts `toHaveAttribute('title', ...)` against the live, rendered chip — proving the `title` attribute must actually be SET at runtime with the exact tooltip text, not merely present in source.
   - Blindly dropping `title` (as the todo's literal wording suggests) would break BOTH of these existing, intentional guarantees — removing a real mobile-accessibility affordance and an already-shipped, tested behaviour — to fix a desktop-only visual overlap. Per deviation-rule discipline ("only auto-fix issues directly caused by the current task's changes" / never weaken an existing test), this was left alone rather than forced through.
   - **What a correct fix looks like** (for whoever picks this up next): a pointer-type-aware suppression — blank `title` on `mouseenter`/`pointerenter` (restoring it on leave) so a desktop mouse hover never triggers the native tooltip alongside the CSS popover, while leaving `title` untouched for a touch long-press (which does not fire a preceding `mouseenter`). This needs new script logic in `SourceChip.svelte`, a new behavioural test proving both the desktop-suppression and touch-preservation halves, and should NOT touch the structural `title=` presence assertions in `source-chip-pill.test.ts` or `11-binary-changed-repin.spec.ts`.
   - **Recommendation to the orchestrator:** leave `.planning/todos/pending/2026-08-14-suppress-native-tooltips-under-chip-popovers.md` OPEN — this plan did not resolve it, and its own "if 13-06 executes first, it may be cheapest to fold in" note turned out not to hold once the existing touch-accessibility test was found.

3. **Trust-state visibility distinct from a mere warning** — this is the core of Task 1 and IS fully addressed: `manifest_unverified` renders the destructive tone (same as `pin_mismatch`), `shadowed` renders the warning tone, and both are visually and textually distinct from each other and from a transient sync warning, proven in both `match-advisory.test.ts` and the two new e2e specs.

## Issues Encountered

None beyond the deviations above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

Both new trust states are visible, tested at the unit and e2e level against real binaries, and the plugin contract/Signal docs describe the trust system that now actually ships. `13-06` was the last plan in Phase 13's wave 4; no blockers for Phase 14. Two items are carried forward as documented follow-ups (not blockers): the scheduler's unbounded retry-and-log for a refused plugin, and the native-tooltip-suppression todo (left open, with a documented reason and a recommended fix shape for whoever picks it up).

## Self-Check: PASSED

- FOUND: web/src/lib/api.ts
- FOUND: web/src/lib/format.ts
- FOUND: web/src/lib/components/SourceChip.svelte
- FOUND: web/src/lib/components/match-advisory.test.ts
- FOUND: web/src/lib/components/source-chip-tooltip.test.ts
- FOUND: web/e2e/fixtures/plugin-binaries.ts
- FOUND: web/e2e/fixtures/config-builder.ts
- FOUND: web/e2e/fixtures/kernel.ts
- FOUND: web/e2e/specs/13-manifest-unverified.spec.ts
- FOUND: web/e2e/specs/13-shadowed-advisory.spec.ts
- FOUND: docs/plugin-contract.md
- FOUND: docs/plugins/signal.md
- FOUND: docs/testing.md
- FOUND commit 37ef689 (Task 1)
- FOUND commit 828236c (Task 2)
- FOUND commit fc038ea (Task 3)

---
*Phase: 13-per-item-curation-installable-app*
*Completed: 2026-08-14*
