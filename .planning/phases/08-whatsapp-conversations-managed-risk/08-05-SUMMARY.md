---
phase: 08-whatsapp-conversations-managed-risk
plan: 05
subsystem: ui
tags: [svelte, sveltekit, whatsapp, polling, gap-closure]

# Dependency graph
requires:
  - phase: 08-whatsapp-conversations-managed-risk
    provides: QRPanel.svelte (plan 08-04), the whatsapp-link HTTP surface (plan 08-03), 08-UAT.md's G-08-1 diagnosis
provides:
  - Amended docs/api.md wire contract (fixed poll cadence, pairing_accepted/already_linked non-terminal states, stderr routing note)
  - Amended 08-UI-SPEC.md Amendment 2 (Pairing progress state, no-Cancel-while-pairing, countdown floor copy, declined-link notice copy)
  - Widened WhatsAppLinkState union (web/src/lib/api.ts)
  - QRPanel.svelte's fixed-cadence schedulePoll, pairing phase, payload-gated countdown restart
affects: [08-06, 08-07]

# Actuals (#2632)
actuals:
  tokens: 6922
  tasks: 2
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Panel liveness poll runs on a fixed interval decoupled from any server-reported validity window (schedulePoll() takes no delay argument by design)"
    - "Non-terminal wire states get their own switch case that ends in schedulePoll(), including default: — no non-terminal state may fall through to a bare break"

key-files:
  created: []
  modified:
    - docs/api.md
    - .planning/phases/08-whatsapp-conversations-managed-risk/08-UI-SPEC.md
    - web/src/lib/api.ts
    - web/src/lib/components/QRPanel.svelte
    - web/src/lib/components/qr-panel.test.ts

key-decisions:
  - "Consumer tolerance shipped before producer emission (deliberate plan ordering): QRPanel now handles pairing_accepted/already_linked even though the plugin (plan 08-06) doesn't emit them yet — an unrecognised non-terminal state now keeps polling via the new default: schedulePoll() case rather than hanging"
  - "Countdown restart gated on payload equality (incoming png_data_uri !== rendered qrDataUri), not an unconditional restart on every qr poll response — this is the direct fix for the frozen-code-reads-as-live-refresh defect"
  - "pairingMessage and both progress copies are module-level const strings, referenced identically by markup and by the structural test guards"

patterns-established:
  - "schedulePoll() with zero parameters as a structural foreclosure device: a future implementer cannot re-introduce cadence-from-payload coupling without changing the function signature itself"

requirements-completed: [SRC-03]

coverage:
  - id: D1
    description: "docs/api.md and 08-UI-SPEC.md describe the fixed poll cadence and the new pairing progress state that the shipped component actually implements"
    requirement: SRC-03
    verification:
      - kind: other
        ref: "grep counts: pairing_accepted x3, already_linked x3, stale-cadence-parenthetical x0 in docs/api.md; Amendment 2 heading + 2 supersession pointers in 08-UI-SPEC.md"
        status: pass
    human_judgment: false
  - id: D2
    description: "QRPanel.svelte polls on its own fixed 2s clock, independent of expires_in_seconds, and schedulePoll's signature structurally forecloses re-coupling cadence to a code's validity"
    requirement: SRC-03
    verification:
      - kind: unit
        ref: "web/src/lib/components/qr-panel.test.ts#the liveness poll runs on its own fixed cadence, decoupled from any code's validity window (G-08-1)"
        status: pass
    human_judgment: false
  - id: D3
    description: "A distinct Pairing progress state renders between qr and the terminal states, with no Cancel affordance available while it is live"
    requirement: SRC-03
    verification:
      - kind: unit
        ref: "web/src/lib/components/qr-panel.test.ts#a post-pair progress phase sits between qr and the terminal states (G-08-1)"
        status: pass
    human_judgment: false
  - id: D4
    description: "A countdown reaching zero without a replacement code reads as 'Waiting for a new code…', never as a live code about to refresh"
    requirement: SRC-03
    verification:
      - kind: unit
        ref: "web/src/lib/components/qr-panel.test.ts#the countdown line falls back to a waiting copy once it reaches zero without a replacement code (G-08-1)"
        status: pass
    human_judgment: false
  - id: D5
    description: "The genuine end-to-end proof (qr -> poll -> pairing_accepted -> paired walk against a realistic expiry) is deliberately deferred to plan 08-07's Playwright case, per this plan's own objective — no tracer task exists here since the path being fixed already runs end to end"
    verification: []
    human_judgment: true
    rationale: "This plan's own objective explicitly defers the live end-to-end walk to 08-07; nothing in this plan's scope produces an e2e artifact to point at yet."

# Metrics
duration: ~25min
completed: 2026-08-10
status: complete
---

# Phase 08 Plan 05: QR Panel Poll-Cadence Decoupling and Pairing Progress State Summary

**Fixed-cadence `schedulePoll()` and a new Pairing progress phase close the browser half of G-08-1's AND-gate — a successfully-scanned WhatsApp session no longer reads as visually indistinguishable from a dead one.**

## Performance

- **Duration:** ~25 min
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- `docs/api.md` no longer prescribes deriving the browser's poll cadence from `expires_in_seconds`; it now states the fixed short-interval rule plus the `pairing_accepted`/`already_linked` non-terminal wire states and a note on stderr routing
- `08-UI-SPEC.md` gained a dated Amendment 2 (Pairing progress copy, no-Cancel-while-pairing, countdown floor copy, and the plan 08-07 declined-link notice copy locked here), with supersession pointers added to Amendment 1
- `WhatsAppLinkState` (web/src/lib/api.ts) widened with the two new non-terminal values
- `QRPanel.svelte` rewritten: `schedulePoll()` takes no delay argument and runs on a fixed `POLL_INTERVAL_MS` (2s); the `qr` case restarts the countdown only when the incoming code payload differs from the one already rendered; a new `pairing` phase renders between `qr` and the terminal states with no Cancel affordance; `default:` now calls `schedulePoll()` instead of a bare `break`; the countdown line falls back to "Waiting for a new code…" at zero

## Task Commits

Each task was committed atomically:

1. **Task 1: Amend the wire contract and the UI contract, and widen WhatsAppLinkState** - `74bf532` (docs)
2. **Task 2: QRPanel polls on its own clock and renders the post-pair progress state** - `b58caf8` (test, RED) + `bea4c2b` (feat, GREEN)

_TDD task: RED (`b58caf8`) added the new structural guards and confirmed they failed against the pre-existing component; GREEN (`bea4c2b`) implemented the fix and all 23 qr-panel.test.ts assertions pass._

## Files Created/Modified

- `docs/api.md` - Fixed poll-cadence rule, `pairing_accepted`/`already_linked` state rows, non-terminal clarification, stderr-routing note
- `.planning/phases/08-whatsapp-conversations-managed-risk/08-UI-SPEC.md` - Amendment 2 section plus two supersession pointers into Amendment 1
- `web/src/lib/api.ts` - `WhatsAppLinkState` widened with two non-terminal values
- `web/src/lib/components/QRPanel.svelte` - Fixed-cadence `schedulePoll()`, payload-gated countdown restart, new `pairing` phase, `default:` no longer hangs the poll, countdown floor copy
- `web/src/lib/components/qr-panel.test.ts` - New/updated structural guards for all of the above; the old `POLL_FLOOR_MS`-clamp assertion (the one the plan explicitly named as replaced) is gone

## Decisions Made

- Consumer tolerance shipped before producer emission, per the plan's own stated ordering — `applySession`'s `default:` case now calls `schedulePoll()` so plan 08-06 can later emit `pairing_accepted`/`already_linked` without this panel ever hanging on an unrecognised state in the interim
- Countdown restart gated on `png_data_uri !== qrDataUri` equality rather than restarting unconditionally on every `qr` poll response — this is the direct mechanical fix for the frozen-already-scanned-code-looks-live defect G-08-1 diagnosed
- Both progress copies (`Scan accepted — completing login…`, `Already linked — confirming this session…`) declared as module-level `const` strings so the markup and the test guards reference one literal each

## Deviations from Plan

**1. [Rule 1 - Bug] Updated a pre-existing test assertion invalidated by the new markup, beyond the one the plan explicitly excepted**
- **Found during:** Task 2, writing the RED guards
- **Issue:** The plan's `<behavior>` block explicitly excepted only the `POLL_FLOOR_MS`-clamp assertion from "every existing assertion must still pass." Two more existing assertions broke as a direct, unavoidable consequence of the same rewrite: (a) the qr-branch countdown assertion, which matched the literal `Refreshes in {formatCountdown(remainingSeconds)}` template expression — now replaced by the `{countdownLine}` derived-value render the plan's own `<behavior>` requires; (b) the success-branch extraction's end marker, an exact-whitespace string (`'{/if}\n\n\t{#if phase'`) that broke when a load-bearing explanatory comment (which the plan's own `<action>` explicitly requires: "say so in a comment there") was inserted between the two markers.
- **Fix:** Updated the qr-branch assertion to check for `{countdownLine}` instead of the old inline expression (the plan's own behavior spec names this exact derived value). Replaced the success-branch's fragile whitespace-exact end marker with the stable `"{#if phase === 'loading' || phase === 'qr'}"` string.
- **Files modified:** web/src/lib/components/qr-panel.test.ts
- **Verification:** All 23 assertions in qr-panel.test.ts pass; full vitest suite (654 tests, 38 files) passes with no regression in add-source.test.ts or relink.test.ts
- **Committed in:** b58caf8 (Task 2 RED commit)

---

**Total deviations:** 1 auto-fixed (1 bug, both instances directly caused by required behavior changes)
**Impact on plan:** No scope creep — both test updates are mechanical consequences of implementing the plan's own named behavior changes, not independent work.

## Issues Encountered

- `web/node_modules` was absent in this worktree (git worktrees don't carry `node_modules`). Symlinked it from the main repo's `web/node_modules` after confirming `package-lock.json` is byte-identical between the two (same md5sum) — read-only use, no lockfile modification, no `npm install`/`npm ci` run.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The consumer half of G-08-1's AND-gate is closed: `QRPanel.svelte` now tolerates `pairing_accepted`/`already_linked` and polls on its own clock, independent of `expires_in_seconds`.
- Plan 08-06 (the producer half — the plugin emitting `pairing_accepted`/`already_linked`) can now land safely; the panel will not hang on those states even before this plan, and will render the new Pairing progress line once 08-06 ships.
- Plan 08-07 owns the genuine end-to-end Playwright proof (`qr -> poll -> pairing_accepted -> paired`) and the Add-Source declined-link notice implementation, whose copy is already locked in this plan's 08-UI-SPEC.md Amendment 2.
- No blockers.

---
*Phase: 08-whatsapp-conversations-managed-risk*
*Completed: 2026-08-10*

## Self-Check: PASSED

- FOUND: .planning/phases/08-whatsapp-conversations-managed-risk/08-05-SUMMARY.md
- FOUND: 74bf532 (Task 1 commit)
- FOUND: b58caf8 (Task 2 RED commit)
- FOUND: bea4c2b (Task 2 GREEN commit)
- FOUND: c4a963f (SUMMARY.md commit)
