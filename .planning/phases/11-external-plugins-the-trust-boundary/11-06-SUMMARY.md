---
phase: 11-external-plugins-the-trust-boundary
plan: 06
subsystem: ui
tags: [svelte5, trust-boundary, plugin-host, sha256, re-pin, playwright, go-plugin]

requires:
  - phase: 11-02
    provides: "kernel/pluginhost.HashBinary/ErrPinMismatch/LaunchFailure/Host.LaunchFailures — pre-exec pin verification and the soft per-instance launch-failure channel"
  - phase: 11-03
    provides: "kernel/httpapi.sourceStatus.PinnedHash/CurrentHash/LaunchFailure — GET /api/sources' published trust-boundary wire surface"
  - phase: 11-05
    provides: "web/src/lib/components/TrustBadge.svelte, web/src/lib/config-edit.ts's setPluginPin, the untrusted-confirm interstitial (E1) and picker labeling (E2/E3) this plan's chip/menu work sits beside"
provides:
  - "SourceChip.svelte's isPinMismatch/isExternal-gated binary-changed tooltip branch, the first-position 'Trust updated binary…' menu item, the Refresh-now mismatch-disable, and the E5 pinned-hash footer with copy-to-clipboard"
  - "web/src/lib/format.ts's shortHash — the shared 12-char-plus-ellipsis short form"
  - "TrustUpdateDialog.svelte — the E4 re-pin confirmation dialog, wired into +page.svelte's trustUpdateInstance state slot"
  - "kernel/pluginhost.launchFailedNames and the widened validateMatchConfig — a pin-mismatched instance is now excused from match-vocabulary validation at BOTH boot (NewSupervisor) and apply (Supervisor.Apply) time, not only the runtime relaunch path"
  - "web/e2e/specs/11-binary-changed-repin.spec.ts — the full swap/catch/re-pin/recover journey proven against the real out-of-repo topos-plugin-external-demo binary"
affects: [12, 14]

actuals:
  tokens: 21350
  tasks: 4
  commits: 4

tech-stack:
  added: []
  patterns:
    - "A third participant class in pluginhost.validateMatchConfig, alongside 'launched' and 'suspended': an instance currently named in Host.LaunchFailures() is excused entirely from its own match-vocabulary check (both the explicit-match-block and keywords-fallback code paths) — its absence is already a separately-reported, machine-readable fact (GET /api/sources' launch_failure field), never a config defect that should block an unrelated save or an entire kernel boot."
    - "The re-pin write is a single-purpose confirmation dialog with no editable form state (unlike EditSourceModal/AddSourceModal) — saving/error state resets for free via the route's own {#key trustUpdateInstance} remount discipline, no defensive reset-on-open effect needed."
    - "shortHash lives in format.ts (not the chip component) specifically because Task 2's dialog needed the identical short-form derivation the chip's own footer (Task 1) already established — one function, two consumers, never two independent slice-and-ellipsis implementations."

key-files:
  created:
    - web/src/lib/components/TrustUpdateDialog.svelte
    - web/src/lib/components/repin.test.ts
    - web/e2e/specs/11-binary-changed-repin.spec.ts
  modified:
    - web/src/lib/format.ts
    - web/src/lib/components/SourceChip.svelte
    - web/src/lib/components/WebspaceHeader.svelte
    - web/src/routes/w/[webspace]/+page.svelte
    - web/src/lib/components/chip-edit-menu.test.ts
    - web/src/lib/components/relink.test.ts
    - web/e2e/e2e-builtins.d.ts
    - kernel/pluginhost/matchconfig.go
    - kernel/pluginhost/matchconfig_test.go
    - kernel/supervisor/pinmismatch_test.go
    - kernel/supervisor/externalproof_test.go

key-decisions:
  - "onedit's kind union widens with 'trust-update' across SourceChip.svelte AND WebspaceHeader.svelte's own separately-typed passthrough prop (Rule 3) — a Svelte prop-callback type is checked contravariantly, so the wider union had to propagate to every call site that types it, including +page.svelte's handleChipEdit and relink.test.ts's literal signature marker, to keep the build green at each task's own commit boundary."
  - "The kernel-side boot-time gap (see Deviations) was found and fixed inside Task 3's own commit — before the Task 4 checkpoint's live walkthrough ever ran — but the walkthrough's manual build predated that fix landing on this branch, which is why round 1 of the checkpoint still reproduced the stale symptom. Round 2 added dedicated NewSupervisor-level test coverage and an independent, outside-the-test-suite live reproduction (real kernel, real binary, real HTTP calls) to close the loop conclusively rather than relying solely on the checkpoint's own manual re-walkthrough."
  - "TrustUpdateDialog reads source.current_hash directly (never re-derives or re-hashes) and setPluginPin only ever echoes that value back — the client-side trust boundary never computes its own notion of 'the new hash' (T-11-30)."

requirements-completed: [PLUG-07, PLUG-08]

coverage:
  - id: D1
    description: "A source whose external binary no longer matches its pin renders a chip in the binary-changed state: a destructive health dot and a tooltip naming the specific cause, taking priority over the generic unreachable wording and yielding only to the syncing branch — healthTone/tooltipText both gated on the kernel-published launch_failure field, never a last_error string match"
    requirement: "PLUG-07"
    verification:
      - kind: unit
        ref: "web/src/lib/components/repin.test.ts (format.ts: healthTone branches on launch_failure before its existing tone logic — 3 cases; SourceChip.svelte: isPinMismatch/isExternal keyed on kernel-published fields — 3 cases)"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/11-binary-changed-repin.spec.ts (external chip alone destructive + binary-changed tooltip; control chip and its stream items unaffected)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The chip menu offers 'Trust updated binary…' as the FIRST item only when the mismatch signal is set (absent, not merely disabled, otherwise); Refresh now disables alongside it; a pinned-hash footer (short display form, full title/copy value) renders for every external-tier chip's menu, absent for trusted tier"
    requirement: "PLUG-08"
    verification:
      - kind: unit
        ref: "web/src/lib/components/repin.test.ts (Trust updated binary… menu item — 4 cases; Refresh now disabled while isPinMismatch; pinned-hash footer — 5 cases); web/src/lib/components/chip-edit-menu.test.ts (superseded assertions updated in place: 6-item label set, 3 separators, 2 aria-labels)"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/11-binary-changed-repin.spec.ts (Trust updated binary… is the first menu item, Refresh now disabled while mismatched; recovered menu shows neither, Refresh now re-enabled)"
        status: pass
    human_judgment: false
  - id: D3
    description: "The re-pin dialog shows the previously pinned hash (short form, or 'not pinned' when absent) and the on-disk hash (full, break-all); confirming calls setPluginPin + putConfig exactly once each, keyed on the binary name (D-02) and echoing the kernel-published current_hash verbatim (T-11-30, never client-computed); failure surfaces through the existing destructive Alert + CONFIG_CONFLICT_MESSAGE pattern"
    requirement: "PLUG-07"
    verification:
      - kind: unit
        ref: "web/src/lib/components/repin.test.ts (TrustUpdateDialog.svelte: E4 Copywriting Contract — 4 cases; hash block — 4 cases; confirm handler — 5 cases; +page.svelte wiring — 6 cases)"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/11-binary-changed-repin.spec.ts (dialog shows the exact independently-computed previously-pinned/currently-on-disk hashes and that they differ; confirming persists the new hash and recovers the chip)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Swapping an external binary is caught, named, visible on the affected chip only, and repairable from that chip's own menu — proven end to end in a real browser against the genuinely out-of-repo topos-plugin-external-demo binary, never mutating the shared bin/plugins-external build output"
    requirement: "PLUG-07"
    verification:
      - kind: e2e
        ref: "web/e2e/specs/11-binary-changed-repin.spec.ts#a tampered external binary refuses to launch by name, an unrelated save still succeeds, and re-pinning recovers the chip"
        status: pass
    human_judgment: false
  - id: D5
    description: "A pin-mismatched instance participating in a webspace's match config (explicit block or keywords fallback) no longer blocks kernel BOOT — pluginhost.ValidateMatchConfig excuses a currently-launch-failed instance from its own vocabulary check, closing a gap 11-02-SUMMARY.md had flagged and left unresolved; every other configured source still boots and syncs"
    requirement: "PLUG-07"
    verification:
      - kind: unit
        ref: "kernel/pluginhost/matchconfig_test.go#TestValidateMatchConfig_PinMismatchedInstanceExcusedFromExplicitMatchBlock, #TestValidateMatchConfig_PinMismatchedInstanceExcusedFromKeywordsFallback, #TestValidateMatchConfig_UnlaunchedInstanceStillFailsWhenNotExcused (negative control)"
        status: pass
      - kind: integration
        ref: "kernel/supervisor/pinmismatch_test.go#TestPinMismatch_BootSucceedsWithMismatchedInstanceParticipatingInWebspace (real NewSupervisor boot, healthy sibling in the same webspace still syncs)"
        status: pass
      - kind: manual_procedural
        ref: "Live reproduction outside the test suite: real kernel binary + real topos-plugin-external-demo, real config.toml, boot (healthy) -> stop -> tamper -> restart (boots, demo soft-fails, control stays healthy) -> PUT repaired pin over HTTP -> demo recovers, recorded in this plan's checkpoint round 2 response"
        status: pass
    human_judgment: false
  - id: D6
    description: "A human confirmed all six Phase 11 UI elements (E1-E6) render as the design contract specifies, including the corrected binary-changed/re-pin walkthrough (E4) and the pinned-hash footer (E5) this plan itself ships"
    verification: []
    human_judgment: true
    rationale: "Pixel/copy/spacing fidelity against 11-UI-SPEC.md cannot be judged by an automated test — approved live by the user across two checkpoint rounds (round 1 found the tamper-script/boot-path issues fixed in round 2; round 2 approved on the strength of the corrected script plus an independent live reproduction). Any residual pixel-level findings route through /gsd-verify-work 11."

duration: ~3h10min (session start through checkpoint round 2 approval, including two live-kernel manual verification passes)
completed: 2026-08-13
status: complete
---

# Phase 11 Plan 06: Binary-Changed Chip State, Re-Pin Flow, and Boot-Time Soft-Failure Parity Summary

**A swapped external plugin binary is now caught, named on its own chip (destructive tone + "binary changed" tooltip), and repairable in two clicks via a new "Trust updated binary…" menu action and confirmation dialog that echoes the kernel-published hash back through the existing hot-apply config save — proven end to end against a real out-of-repo binary in a browser — and a kernel boot-time gap this plan's own checkpoint walkthrough surfaced (a pin-mismatched, webspace-participating instance used to refuse the WHOLE kernel from starting) is now closed, with the mismatch soft-failing at boot exactly as it already did at runtime relaunch.**

## Performance

- **Duration:** ~3h10min (session start through checkpoint round 2 approval — includes two independent live-kernel manual verification passes, one per checkpoint round)
- **Started:** 2026-08-13T11:09:55+01:00 (worktree base)
- **Completed:** 2026-08-13T14:10:42+01:00
- **Tasks:** 4/4 (3 auto tasks + 1 human-verify checkpoint, approved after one fix round)
- **Files modified:** 15 (3 new, 12 modified)

## Accomplishments

- `SourceChip.svelte` gained `isPinMismatch`/`isExternal` derivations (kernel-published fields only, never a `last_error` string match — T-11-32), a binary-changed `tooltipText` branch inserted ahead of the plain destructive wording, a first-position "Trust updated binary…" menu item gated on the mismatch signal, a mismatch-aware `Refresh now` disable, and the E5 pinned-hash footer (`Copy`/`Check` swap on successful copy, silent no-op on clipboard failure) — a trusted-tier chip's menu stays byte-identical to before this phase
- `format.ts` gained `shortHash` (the shared 12-char-plus-ellipsis form) and a leading `launch_failure === 'pin_mismatch'` branch in `healthTone`, extending the existing branch chain rather than introducing a parallel tone system
- `TrustUpdateDialog.svelte` is the new E4 re-pin confirmation dialog — shows the previously-pinned hash (short, or "not pinned") and the on-disk hash (full, `break-all`), and its single confirm action calls `setPluginPin` + `putConfig` exactly once each, keyed on the plugin binary name (D-02: one re-pin repairs every instance sharing that binary) and echoing back `source.current_hash` verbatim — wired into `+page.svelte` via a `trustUpdateInstance` state slot mirroring `relinkInstance`'s own shape
- `web/e2e/specs/11-binary-changed-repin.spec.ts` proves the full journey against the genuinely out-of-repo `topos-plugin-external-demo` binary: tamper only the fixture's own external directory (never `bin/plugins-external`, the shared build output), trigger a relaunch through the real Edit connection… UI flow, assert the unrelated save succeeds, assert the external chip alone goes destructive while the control chip and its stream items are unaffected, assert the menu ordering and disabled state, assert the dialog's two independently-computed hashes differ, and assert the persisted pin equals the new on-disk hash after confirming
- **Deviation, found via the Task 4 checkpoint's live walkthrough:** `pluginhost.ValidateMatchConfig` rejected kernel BOOT (and apply) for any webspace naming a pin-mismatched, participating instance as "has no launched plugin" — closed by threading `Host.LaunchFailures()` into `validateMatchConfig` as a third excused participant class (alongside "launched" and "suspended"), with new unit and supervisor-level test coverage plus an independent live reproduction against a real, running kernel process

## Task Commits

Each task was committed atomically:

1. **Task 1: The binary-changed chip state, the trust-update menu action, and the pinned-hash footer** - `50b635d` (feat)
2. **Task 2: The re-pin confirmation dialog and its hot-apply save** - `6476c5f` (feat)
3. **Task 3: Browser proof — swap a real binary mid-run, see it caught, re-pin, recover** (includes the boot-time `matchconfig.go` fix, discovered running this task's own e2e spec) - `2204ba0` (test)
4. **Task 4 checkpoint follow-up: verification-script fix + dedicated boot-time test coverage** (round 2, after the live walkthrough reproduced the pre-fix build's symptom) - `05b0679` (docs)

**Plan metadata:** commit pending (this SUMMARY.md + PLAN.md, no STATE.md/ROADMAP.md — worktree mode, orchestrator owns those centrally)

_No TDD tasks executed as RED/GREEN pairs — Tasks 1-2 declared `tdd="true"`; tests were written and verified alongside each task's own implementation within a single commit per task, matching this phase's established single-commit-per-task convention._

## Files Created/Modified

**Task 1 (`50b635d`):**
- `web/src/lib/format.ts` - `healthTone`'s leading `launch_failure` branch, `shortHash`
- `web/src/lib/components/SourceChip.svelte` - `isPinMismatch`/`isExternal`, tooltip branch, trust-update menu item, Refresh-now disable, pinned-hash footer, `hashCopied`/`copyPinnedHash`
- `web/src/lib/components/WebspaceHeader.svelte` - `onedit`'s widened kind union (passthrough prop type)
- `web/src/routes/w/[webspace]/+page.svelte` - `handleChipEdit`'s widened kind union + placeholder `'trust-update'` branch (superseded by Task 2)
- `web/src/lib/components/repin.test.ts` - new (Task 1's own guards)
- `web/src/lib/components/chip-edit-menu.test.ts` - superseded assertions updated in place (label set, separator count, aria-label count, Refresh-now-first)
- `web/src/lib/components/relink.test.ts` - `handleChipEdit`'s signature marker widened

**Task 2 (`6476c5f`):**
- `web/src/lib/components/TrustUpdateDialog.svelte` - new
- `web/src/routes/w/[webspace]/+page.svelte` - `trustUpdateInstance`/`trustUpdateSource` state, real `'trust-update'` branch, `handleTrustUpdateClose`/`handleTrustUpdateSaved`, dialog mount
- `web/src/lib/components/repin.test.ts` - extended with Task 2's guards

**Task 3 (`2204ba0`):**
- `web/e2e/specs/11-binary-changed-repin.spec.ts` - new
- `web/e2e/e2e-builtins.d.ts` - `Buffer.from`/`Buffer.concat`, `writeFileSync`'s Buffer overload, `chmodSync`
- `kernel/pluginhost/matchconfig.go` - `launchFailedNames`, `validateMatchConfig`/`validateMatchBlockVocabulary`/`validateFallbackVocabulary` widened with the `excused` set
- `kernel/pluginhost/matchconfig_test.go` - 4 new tests (2 positive exemption cases, 1 negative control, plus the pre-existing suite's own `newTestHostWithLaunchFailure` fixture)
- `kernel/supervisor/externalproof_test.go` - stale comment updated in place
- `kernel/supervisor/pinmismatch_test.go` - stale comment updated in place

**Task 4 checkpoint follow-up (`05b0679`):**
- `kernel/supervisor/pinmismatch_test.go` - `TestPinMismatch_BootSucceedsWithMismatchedInstanceParticipatingInWebspace` (full `NewSupervisor`-level proof)
- `.planning/phases/11-external-plugins-the-trust-boundary/11-06-PLAN.md` - Task 4's tamper step corrected (stop-server-first, ETXTBSY explained), new must-haves truth, T-11-33's mitigation widened, verification section's stale "no kernel change" note corrected

## Decisions Made

- **`onedit`'s kind union widening had to propagate to every typed call site, not just `SourceChip.svelte`** — Svelte prop-callback types are checked contravariantly, so `WebspaceHeader.svelte`'s own separately-declared `onedit` prop type, `+page.svelte`'s `handleChipEdit` signature, and `relink.test.ts`'s literal signature-marker string all needed mechanical updates within Task 1 to keep `npm run check` green at that task's own commit boundary — Task 2 then replaced Task 1's placeholder `'trust-update'` branch with the real `trustUpdateInstance` wiring.
- **`shortHash` lives in `format.ts`, not `SourceChip.svelte`** — Task 2's dialog needed the identical short-form derivation Task 1's footer already established; one shared function, never two independent slice-and-ellipsis implementations.
- **The boot-time kernel fix's test coverage is layered three ways**: pure `pluginhost` unit tests (isolated `validateMatchConfig` behavior, including a negative control proving the exemption is scoped to exactly the excused instance), a full `kernel/supervisor` `NewSupervisor`-level integration test (the real boot sequence, a real webspace, a real healthy sibling), and an independent live reproduction outside both test suites (a real kernel process, a real binary, real HTTP calls) — the checkpoint's own live walkthrough is what surfaced the gap, so closing the loop with an equally live reproduction (not only new unit tests) was the appropriate bar of proof.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `onedit`'s kind union widening cascaded to `WebspaceHeader.svelte`, `+page.svelte`, and `relink.test.ts`**
- **Found during:** Task 1, `npm run check` after widening `SourceChip.svelte`'s `onedit` prop type
- **Issue:** `WebspaceHeader.svelte` independently re-declares the same `onedit` callback type as its own prop (a passthrough, not an inferred type) — TypeScript's contravariant function-parameter checking meant the narrower, pre-existing union was no longer assignable once `SourceChip`'s own prop widened. `+page.svelte`'s `handleChipEdit` (the concrete implementation ultimately passed in) and `relink.test.ts`'s literal string-marker assertion of that function's signature both needed matching updates.
- **Fix:** Widened `WebspaceHeader.svelte`'s `onedit` prop type; widened `handleChipEdit`'s parameter type with a placeholder `'trust-update'` no-op branch (superseded by Task 2's real wiring); updated `relink.test.ts`'s marker to match the two-line-wrapped signature.
- **Files modified:** `web/src/lib/components/WebspaceHeader.svelte`, `web/src/routes/w/[webspace]/+page.svelte`, `web/src/lib/components/relink.test.ts`
- **Verification:** `npm --prefix web run check` — 0 errors; `npm --prefix web run test` — full suite green.
- **Committed in:** `50b635d` (Task 1 commit)

**2. [Rule 1 - Bug] Kernel boot/apply hard-fails for a pin-mismatched, webspace-participating instance**
- **Found during:** Task 3, running the new e2e spec's real config-write step (surfaced immediately; confirmed as a genuine, pre-existing kernel gap rather than a spec bug)
- **Issue:** `pluginhost.ValidateMatchConfig`, called by both `NewSupervisor` (boot) and `Supervisor.Apply` (config save), rejected ANY webspace naming a pin-mismatched instance — via an explicit match block OR the keywords fallback — as "has no launched plugin". This directly contradicted T-11-33's own threat-register mitigation ("a config save succeeds while a source is pin-mismatched") and, at boot, made the re-pin flow this very plan ships structurally unreachable after a real kernel restart (the mismatched instance's chip would never boot into view at all). `11-02-SUMMARY.md`'s own "Issues Encountered" had flagged this exact gap, unresolved, as deferred to "a later plan... if it becomes user-visible" — it did, in this plan's own Task 3 and again in the Task 4 checkpoint's live walkthrough.
- **Fix:** `kernel/pluginhost/matchconfig.go`'s `validateMatchConfig` now accepts a third `excused` participant set (instance names currently present in `Host.LaunchFailures()`), threaded through both `ValidateMatchConfig` and `ValidateMatchConfigWithSuspended`, and checked in both `validateMatchBlockVocabulary` and `validateFallbackVocabulary` before either would otherwise report "has no launched plugin".
- **Files modified:** `kernel/pluginhost/matchconfig.go`, `kernel/pluginhost/matchconfig_test.go` (4 new tests), `kernel/supervisor/externalproof_test.go` and `kernel/supervisor/pinmismatch_test.go` (stale comments updated in place)
- **Verification:** `CGO_ENABLED=0 go build ./... && go test ./...` (whole repo); `make e2e` (106/106); round 2 additionally added `TestPinMismatch_BootSucceedsWithMismatchedInstanceParticipatingInWebspace` and an independent live reproduction (real kernel binary, real `topos-plugin-external-demo`, real `config.toml`, real HTTP boot/tamper/restart/re-pin cycle) proving the fix holds outside both the Go test suite and the e2e harness.
- **Committed in:** `2204ba0` (Task 3 commit, initial fix); `05b0679` (checkpoint round 2, additional supervisor-level test coverage + verification-script correction)

---

**Total deviations:** 2 auto-fixed (1 Rule 3 blocking type-propagation, 1 Rule 1 kernel bug — the latter discovered live twice: once by this plan's own Task 3 e2e spec, once by the Task 4 checkpoint's human walkthrough against a build that predated the fix)
**Impact on plan:** The Rule 1 kernel fix is the most consequential deviation in this plan: without it, T-11-33's own threat mitigation would have shipped broken at boot (though proven at apply-time by the e2e spec), and the repin flow this plan exists to build would have been unreachable after any real-world kernel restart with a tampered, webspace-participating binary — exactly the scenario a real operator would hit. No scope creep beyond fixing what the plan's own threat model required to actually hold.

## Issues Encountered

- **Checkpoint round 1's tamper instruction (`printf '\0' >> ...` against a running kernel) fails with `ETXTBSY`** — the plan's own Task 4 text hadn't accounted for the binary being a live, open executable. Corrected in round 2 to a stop-server-first ordering (simpler for a manual walkthrough than the e2e spec's own temp-file+rename technique).
- **The Task 4 checkpoint's live walkthrough used a kernel build that predated Task 3's own boot-time fix** — the fix had already landed in this branch's history by the time the checkpoint was returned, but the human's manual `make build` run happened against an earlier state. Resolved by adding dedicated `NewSupervisor`-level test coverage and reproducing the full boot/tamper/restart/re-pin cycle live, independently of both the Go test suite and the e2e harness, to give the user a byte-for-byte re-verifiable script and independent confirmation the fix holds against a genuinely running process before they re-ran the walkthrough themselves.
- **`kernel/webui/build/.gitkeep` was overwritten by the SPA build step** (the same pre-existing gap 11-01/11-05-SUMMARY.md already documented) — restored via `git checkout -- kernel/webui/build/.gitkeep` before the Task 3 commit.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- ROADMAP success criterion 3 is fully delivered: swapping an external binary is caught, named, visible on the affected chip only, and repairable from that chip's own menu — proven in a browser against a real out-of-repo binary, and now also proven to survive a real kernel restart rather than only a runtime relaunch.
- PLUG-07/PLUG-08 are fully delivered at both the UI and kernel layers; T-11-33's threat mitigation now genuinely holds at both apply time (proven by 11-05/11-06's e2e specs) and boot time (this plan's own checkpoint-driven fix).
- All six Phase 11 UI elements (E1-E6) are human-approved against 11-UI-SPEC.md, closing this phase's single human-verification checkpoint.
- Full local verification green: `npm --prefix web run check` (0 errors), `npm --prefix web run test` (912/912), `npm --prefix web run check:e2e` (0 errors), `make e2e` (106/106, full suite including the new spec), `CGO_ENABLED=0 go build ./... && go test ./...` (whole repo, including the new boot-time supervisor test).
- Phase 11 (external plugin loading + trust boundary) is now feature-complete pending the orchestrator's own post-wave verification pass; Phase 12 (filesystem plugin) and Phase 14 (Google Drive, out-of-repo) both depend on this phase's external-plugin mechanism being solid, which this plan's checkpoint round closed the last known gap in.

---
*Phase: 11-external-plugins-the-trust-boundary*
*Completed: 2026-08-13*

## Self-Check: PASSED

- Verified files exist: `web/src/lib/components/TrustUpdateDialog.svelte`, `web/src/lib/components/repin.test.ts`, `web/e2e/specs/11-binary-changed-repin.spec.ts`, `kernel/pluginhost/matchconfig.go`, `kernel/pluginhost/matchconfig_test.go`, `kernel/supervisor/pinmismatch_test.go`
- Verified commits exist in `git log --oneline`: `50b635d`, `6476c5f`, `2204ba0`, `05b0679`
