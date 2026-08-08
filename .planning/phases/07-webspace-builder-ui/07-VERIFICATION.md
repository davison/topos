---
phase: 07-webspace-builder-ui
verified: 2026-08-08T20:30:00Z
status: gaps_found
score: 63/95 must-haves verified
behavior_unverified: 32
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 54/85
  gaps_closed:
    - "kernel/supervisor/supervisor.go's Apply never leaves s.host, s.coord and s.cfg disagreeing on which config generation they reflect after a post-Reconcile ValidateMatchConfig rejection (07-VERIFICATION.md prior gaps[0] / 07-REVIEW.md post-07-07/08 CR-01): confirmed by direct reading of kernel/supervisor/supervisor.go — the ValidateMatchConfig failure branch (line ~158 in comment-stripped numbering) now calls the new commitGeneration(newCfg) instead of s.startScheduler(oldCfg), and commitGeneration performs coordinator-install, then s.cfg assignment, then scheduler-start in that fixed order (grep-verified: s.coord = newCoordinator( appears at exactly 2 sites — NewSupervisor's boot sequence and commitGeneration; s.startScheduler( appears at exactly 3 sites — boot, commitGeneration, and the pre-Reconcile old-generation restart; commitGeneration( appears at exactly 5 sites — 1 declaration + 4 call sites covering the vocabulary branch, both D-07 index-cleanup branches, and the success path). Two behavioral tests independently re-run in this session — TestApply_ValidateMatchConfigFailsAfterReconcile_CoordinatorTracksRelaunchedPlugin and TestApply_RejectedSaveIsIdempotent_SecondApplyDoesNotRelaunchSubprocesses — both pass, the first proving via a real sup.Refresh() call that the coordinator dispatches against the live relaunched plugin (Status == \"ok\"), not the killed one. Apply's doc comment now states the two-regime contract (pre-Reconcile: old generation kept; post-Reconcile: new generation adopted) and cites gaps[0], D-06, D-07 and D-08 by name."
  gaps_remaining: []
  regressions:
    - "No regression caused by 07-09 — git diff --stat for 07-09's three commits touches only kernel/supervisor/supervisor.go and kernel/supervisor/supervisor_test.go, and the two pre-existing name-pinned tests (TestApply_MidFlightSyncLeavesNoStrandedRunningRow, TestApply_RemovedInstance_PluginGoneAndIndexRowsGone) are confirmed byte-identical in their own bodies. However, this phase's own fresh code-review re-pass (07-REVIEW.md, committed d016738, dated after 07-09 landed) traced the D-07 removed-instance cleanup loop against Apply's now-corrected control flow and surfaced ONE NEW Critical-severity, previously-undetected defect in the same function, in the same class as the just-closed gaps[0]: the D-07 cleanup loop (deletes a removed instance's index rows and sync history) sits textually AFTER the ValidateMatchConfig check in Apply's body, so when ValidateMatchConfig fails, the function returns before the loop is ever reached — a removed instance's subprocess is already killed by the preceding successful Reconcile, but its items/sync_runs rows are never deleted, and because the ValidateMatchConfig failure branch now correctly calls commitGeneration(newCfg) (the very fix 07-09 just landed), s.cfg advances to newCfg on that very call — so on every later Apply, oldCfg no longer contains the stranded instance either, and removedInstances(oldCfg, newCfg) can never again compute it as removed. There is no retry path; the orphaned rows persist in the index permanently. This predates 07-09 (confirmed via git show f25c4ab:kernel/supervisor/supervisor.go showing the same ordering before 07-09's fix) and is not something 07-09's own must_haves claimed to cover — 07-09's truths address generation consistency (s.host/s.coord/s.cfg agreement) and the four branches' shared commit-site ordering, not whether the D-07 loop actually executes on every post-Reconcile exit path. Independently confirmed here by reading kernel/supervisor/supervisor.go lines 307-377 directly, not taken on 07-REVIEW.md's word — see Critical Issues below."
gaps:
  - truth: "The kernel's hot-apply mechanism (Supervisor.Apply, D-06/D-07) never permanently strands a removed source instance's index rows (items, sync_runs) — the documented T-07-13 guarantee that a removed-then-re-added instance starts with no inherited history holds on every path out of Apply, not only the success path"
    status: failed
    reason: "kernel/supervisor/supervisor.go:307-377 (Apply), confirmed by direct read. The D-07 removed-instance cleanup loop (lines 346-355, iterating removedInstances(oldCfg, newCfg) and calling s.idx.DeleteSourceItems/DeleteSyncRuns) is positioned AFTER the ValidateMatchConfig check (lines 325-336). When ValidateMatchConfig fails, Apply returns at line 335 — the D-07 loop is never reached for that Apply call. Host.Reconcile has already committed by that point (it ran and succeeded before ValidateMatchConfig was even called), so any instance removed from config in that same save has already had its subprocess killed, but DeleteSourceItems/DeleteSyncRuns are never invoked for it. Worse: the ValidateMatchConfig failure branch calls commitGeneration(newCfg) — this phase's own 07-09 fix for the OTHER gap — which sets s.cfg = newCfg. On the very NEXT Apply call, oldCfg is now newCfg, which already lacks the removed instance on both sides of the diff, so removedInstances(oldCfg, newCfg2) can never again compute it as removed. There is no retry path — the orphaned items/sync_runs rows persist in the index forever, or get inherited as phantom history if an instance is later re-added under the same [sources.<id>] key, exactly the outcome DeleteSourceItems/DeleteSyncRuns's own doc comments and T-07-13 name as the guarantee this cleanup exists to provide. A second related failure mode exists in the same loop: if removedInstances names more than one instance and an early one's DeleteSourceItems/DeleteSyncRuns call errors, the loop returns immediately (also via commitGeneration(newCfg) + return), abandoning cleanup for every later-sorted instance in the same batch with the same permanent-stranding consequence. Reachable through the documented, supported hand-edit + POST /api/config/reload flow (ManageSourcesModal.svelte's 'Reload config' button): a single hand-edited config.toml that both removes a [sources.<id>] block and introduces/typos a [webspaces.<name>.match.<other-instance>] field outside that instance's declared vocabulary hits this exact combination in one Apply call. Confirmed via git show f25c4ab:kernel/supervisor/supervisor.go that this ordering (vocabulary check runs before the D-07 loop) predates 07-09 — it is not a regression 07-09 introduced, but it is the same class of defect 07-09 set out to close on the coordinator seam, left unclosed on this adjacent seam. No test exercises this: grepped kernel/supervisor/supervisor_test.go for removedInstances/DeleteSourceItems/DeleteSyncRuns — zero hits outside supervisor.go and kernel/index, confirming the gap is uncovered by the regression suite 07-09 otherwise relies on. First raised as 07-REVIEW.md's (post-07-09) CR-01."
    artifacts:
      - path: "kernel/supervisor/supervisor.go"
        issue: "Apply's control flow runs the D-07 removed-instance cleanup loop only on the path that falls through the ValidateMatchConfig check successfully — a ValidateMatchConfig failure (or a mid-loop DeleteSourceItems/DeleteSyncRuns failure) returns before/during the loop while committing s.cfg forward to newCfg anyway, so the diff that would ever again detect the removed instance is destroyed by the very commit that was supposed to make the failure survivable"
    missing:
      - "Run the D-07 cleanup loop unconditionally on every post-Reconcile exit path — i.e., move it ahead of (or make it independent of) the ValidateMatchConfig check, since Reconcile has already committed and the removed instances are already gone from the host regardless of what any later check decides"
      - "Make the loop itself not abort on a single instance's cleanup failure — collect per-instance errors (e.g. via errors.Join) and continue to the next name, so one SQL failure doesn't strand every other removed instance in the same batch"
      - "A test exercising a removed instance combined with a post-Reconcile ValidateMatchConfig failure in the same Apply call, asserting the removed instance's items/sync_runs rows are gone despite the save being rejected — supervisor_test.go currently has no such case (grepped for removedInstances/DeleteSourceItems/DeleteSyncRuns, no hits)"
deferred: []
behavior_unverified_items:
  - truth: "Webspace switcher: opening the drop-down lists every configured webspace, marks the current one aria-current at weight 600, and clicking a non-current entry navigates to it in one click (07-03 Task 2)"
    test: "make dev; open a webspace with 2+ webspaces configured; open the title drop-down; confirm every webspace is listed, the current one is visually heavier; click another entry"
    expected: "Menu lists all webspaces in GET /api/config order; current one bold; click navigates to /w/{name} with no full page reload artifact"
    why_human: "web/src/lib/components/webspace-switcher.test.ts is a comment-stripped source-scan, not a rendered-component interaction test. No make dev session was available in this verification environment."
  - truth: "Create-webspace modal: submitting a name writes a new [webspaces.<name>] block through PUT /api/config and navigates to it without a kernel restart; a kernel rejection leaves the modal open with the typed name intact (07-03 Task 2)"
    test: "make dev; + New webspace; type a name; submit; confirm config.toml gains the block and the app navigates there with no restart. Then submit a name the kernel rejects and confirm the Alert + retained input."
    expected: "One PUT /api/config call; success navigates; failure keeps modal open with kernel's verbatim message"
    why_human: "CreateWebspaceModal's actual PUT /api/config round trip was never exercised against a live kernel in this or any prior session."
  - truth: "Root redirect: with no webspaces configured, `/` renders 'No webspaces yet' with a working Create webspace CTA and does not navigate; with webspaces, it lands on the remembered/first one (07-03 Task 3)"
    test: "make dev with config.toml carrying zero [webspaces.*] blocks; load /; confirm the empty state and its CTA. Then add webspaces back and confirm redirect behavior."
    expected: "No redirect loop, no blank page, CTA opens the same CreateWebspaceModal"
    why_human: "resolveRedirectTarget() is genuinely unit-tested and VERIFIED separately — but the surrounding +page.svelte component's render/navigate behavior is only structurally scanned, never run in a browser."
  - truth: "Add-source '+' picker: opens a popover offering unparticipating instances plus New {plugin type}… rows, or the exact empty-state copy when none remain; choosing an existing instance opens a match-only modal that writes source+match+allowlist and the new chip appears without reload (07-04 Task 1)"
    test: "make dev; click the dashed '+' chip; add an already-configured instance with match fields; confirm the chip appears and its items sync in"
    expected: "Picker lists correctly, one-step modal round-trips through PUT /api/config, chip appears live"
    why_human: "add-source.test.ts is a structural source-scan for the picker; no live kernel session exercised the picker→modal→PUT→chip-render chain."
  - truth: "Two-step 'New {plugin type}…' flow: Connect step trial-launches via describePlugin, advances to a vocabulary-driven Match step on success, offers 'Save anyway' + the exact failure copy on a Describe failure, and Step 2 issues exactly one PUT /api/config (07-04 Task 2)"
    test: "make dev; + → New {plugin type}…; complete Connect against a real/fake service; confirm the vocabulary-driven Step 2 form and a single config.toml write carrying all three blocks. Additionally (07-06 D5): type a display name that collides with an existing instance at Save anyway and confirm the network tab shows no PUT /api/config, the victim instance's chip/connection/agent grants are byte-identical afterwards, and the rejection message + retry affordance render correctly."
    expected: "Step indicator, describe round trip, single write, chip appears; the collision case refuses the write client-side with no network call"
    why_human: "The Describe round trip against a real plugin subprocess (success and failure paths), and the collision-guard's live browser behavior, were not exercised live. The collision guard itself IS verified at the source-control-flow level plus passing unit/structural tests — this item asks for the remaining live-browser/network confirmation only."
  - truth: "Secret field: shows a live Set/Not-set badge for the typed variable name, never displays or transmits a value, and never blocks submit either way (07-04 Task 2)"
    test: "make dev; type an env var name that IS set in the kernel's environment, confirm 'Set'; type one that is NOT, confirm 'Not set — add it to .env and restart before this source can connect.'; confirm the network tab and DOM never contain a secret value"
    expected: "Badge reflects truth; submit stays enabled either way"
    why_human: "secret-field.test.ts proves the component never renders a password input and never receives a value prop, but the badge's live truthfulness against a running kernel's env_vars map was not exercised."
  - truth: "Chip ⋮ menu: offers exactly Edit connection…/Edit match settings…/Remove from this webspace, opening it never toggles the chip's own filter state, and Edit connection… shows the cross-webspace notice before the fields (07-04 Task 3)"
    test: "make dev; click a chip's ⋮ control and confirm the chip's filter state does NOT change; open each menu item and confirm the notice/pre-filled state."
    expected: "stopPropagation prevents filter toggle; notice visible before fields"
    why_human: "chip-edit-menu.test.ts is a structural source scan; the stopPropagation and cross-webspace-notice behavior at runtime remain unverified. (The previously-open stale-value-on-reopen sub-case, CR-02, is VERIFIED — see prior gaps_closed.)"
  - truth: "Manage sources modal: the sole entry point for instance/webspace deletion; deleting an instance shuts down its subprocess and deletes its indexed items across every webspace; deleting a webspace leaves every instance and other webspace untouched; Reload config applies a hand-edit or shows the kernel's verbatim failure with 'The previous configuration is still running.' (07-05 Task 1)"
    test: "make dev; Manage sources…; delete an instance and confirm its chip/items/subprocess are gone everywhere; delete a webspace and confirm nothing else changed; hand-edit config.toml and click Reload config, both for a valid and an invalid edit"
    expected: "Both AlertDialogs behave as documented; Reload's both branches behave as documented"
    why_human: "manage-sources.test.ts is a structural guard. The underlying kernel mechanics ARE integration-tested in 07-02 — but the UI trigger → confirm → observed effect round trip was never run against a live kernel."
  - truth: "A kernel killed between the config.toml.bak write and the atomic rename leaves config.toml fully intact at its previous content (07-01 backstop)"
    test: "Kill the topos process (SIGKILL) at the instant between the .bak write and the os.Rename call during a config save, then inspect config.toml"
    expected: "config.toml is byte-identical to its pre-save content — never truncated, never half-written"
    why_human: "Explicitly tagged verification: backstop — no test can deterministically interrupt the process at that exact instant."
  - truth: "The webspace switcher drop-down and the add-source picker popover stay usable (height-capped, scrollable) as their list counts reach double digits (07-03/07-04 backstops)"
    test: "Configure 15+ webspaces and 15+ source instances; open the switcher and the '+' picker; confirm both scroll internally rather than growing past the viewport"
    expected: "Fixed max-height with internal scroll"
    why_human: "Tagged verification: backstop — CSS classes are present per structural guards, no test renders at scale to confirm the visual result."
  - truth: "The manage-sources instance and webspace lists stay usable as their counts grow (07-05 backstop)"
    test: "Configure 15+ instances and webspaces; open Manage sources…; confirm both lists scroll internally"
    expected: "Fixed max-height with internal scroll"
    why_human: "Tagged verification: backstop — same class as above."
  - truth: "Against a live kernel via make dev, the two-step New {plugin type}… flow's Save anyway path refuses a colliding display name in the browser exactly as the unit and structural guards assert, and the victim instance's chip, connection and agent grants are unchanged afterwards (07-06 backstop)"
    test: "make dev; open the two-step New {plugin type}… flow; type a display name colliding with an existing instance; fail the connection test; click Save anyway; confirm no PUT /api/config in the network tab, the collision message renders, and the victim instance's chip/connection/agent grants are byte-identical afterwards"
    expected: "Client-side refusal, no network write, victim instance untouched"
    why_human: "Tagged verification: backstop in 07-06-PLAN.md — the guard's synchronous control flow and pure-function behavior ARE genuinely verified; only the live-browser/network confirmation remains."
  - truth: "Against a live kernel via make dev, revoking a source's agent grant in the UI and saving makes that source vanish from an /agent/v1/sources call issued from a second terminal on the next request, with the kernel process never restarted, and the same source's /agent/v1/items/{id} starts answering 404 item_not_found (07-07 backstop)"
    test: "make dev; revoke a source's agent.read grant via the UI's save path; from a second terminal, curl /agent/v1/sources and /agent/v1/items/{id} for an item of that source"
    expected: "Source disappears from /agent/v1/sources, item 404s, no kernel restart"
    why_human: "Tagged verification: backstop in 07-07-PLAN.md — the underlying mechanism IS behaviorally proven by TestAgentLiveConfig_RevokedReadGrantTakesEffectWithoutRestart against a real httptest.Router built from a real temp-file config.Store; only the live make dev / real subprocess confirmation remains."
  - truth: "Against a live kernel via make dev, opening a source's Edit connection…, typing a wrong base_url, clicking Cancel, then reopening the SAME source's Edit connection… shows the value currently in config.toml — and the same holds for Edit match settings… after a Cancelled match edit (07-08 backstop)"
    test: "make dev; Edit connection… on a source; type a wrong base_url; Cancel; reopen Edit connection… on the same source; confirm the field shows the real stored value, not the discarded typed one. Repeat for Edit match settings…"
    expected: "Reopen always shows current config, never a discarded draft"
    why_human: "Tagged verification: backstop in 07-08-PLAN.md — the underlying mechanism IS behaviorally proven by 23 passing unit/structural tests over the extracted seeding module and the route/component wiring; only the live-browser confirmation remains."
  - truth: "handleChipEdit's match-mode describePlugin call resolves without ever letting a slower first request's response overwrite a faster second request's state (WR-01, 07-REVIEW.md prior round)"
    test: "make dev; open 'Edit match settings…' on one chip, then before the vocabulary loads, open 'Edit match settings…' or 'Edit connection…' on a different chip; confirm the modal never briefly shows or reverts to the FIRST chip's vocabulary/open state"
    expected: "The second (current) click's state always wins; the first click's late-resolving describePlugin response is a no-op"
    why_human: "Confirmed by direct read (web/src/routes/w/[webspace]/+page.svelte): handleChipEdit still has no generation/sequence guard around its describePlugin await, unlike every other async call site in the same file — 07-09 deliberately did not touch web/src, so this is unchanged. Real UI display glitch, not a data-corruption path: editInstance always reflects the LAST click, so a save still writes to the correct instance; only offered vocabulary suggestions can be momentarily stale."
  - truth: "Against a live kernel via make dev, editing a source's connection details AND introducing an invalid match field name in the same UI save produces the 500 apply_failed response, and that source's chip then continues to sync and report healthy on its next scheduled tick rather than failing continuously until the kernel is restarted (07-09 backstop, D3)"
    test: "make dev; open a webspace; use the chip ⋮ menu's Edit connection… to change a source's base_url, and in the same session add a match field name the plugin does not declare, then save; confirm the UI surfaces the kernel's rejection; leave the kernel running and watch that source's chip through its next scheduled sync tick"
    expected: "500 apply_failed with the vocabulary error's own text; the source syncs and reports healthy on its next tick rather than failing every tick"
    why_human: "Tagged verification: backstop in 07-09-PLAN.md (D3 in its SUMMARY coverage table) — the underlying mechanism IS behaviorally proven by TestApply_ValidateMatchConfigFailsAfterReconcile_CoordinatorTracksRelaunchedPlugin against real launched mock-plugin subprocesses; only the live make dev / real UI confirmation remains, per workflow.human_verify_mode=end-of-phase."
---

# Phase 7: Webspace Builder UI Verification Report

**Phase Goal:** User can configure sources and webspaces from the UI instead of hand-editing TOML — pick plugin types from a list, configure named instances, save a configured set as a webspace, and promote a live search into the webspace's permanent filter.
**Verified:** 2026-08-08
**Status:** gaps_found
**Re-verification:** Yes — after 07-09 (Supervisor.Apply generation consistency, closing prior round's gaps[0])

## Goal Achievement

### Build, Test and Contract Evidence (independently re-run in this session, not taken from SUMMARY claims)

| Check | Command | Result |
|---|---|---|
| Go build | `CGO_ENABLED=0 go build ./...` | clean, exit 0 |
| Go test suite (full) | `go test ./kernel/... -count=1` | all packages `ok` (config, correlate, httpapi, index, pluginhost, supervisor, syncer) |
| Go test — gaps[0] closure (prior round) | `go test ./kernel/supervisor/... -run 'TestApply' -count=1 -v` | 4/4 passed (2 pre-existing unmodified, 2 new from 07-09) |
| Web test suite (full) | `cd web && npx vitest run` | 31 files, **492/492** passed |
| Diff scope for 07-09 | `git diff --stat ad8fc49~3 ad8fc49 -- kernel/ web/src` (via 07-09-SUMMARY's own commit trail) | Exactly `kernel/supervisor/supervisor.go`, `kernel/supervisor/supervisor_test.go` — no UI file, no HTTP handler touched |
| Source assertion — commit-site counts | `grep -v '^\s*//' kernel/supervisor/supervisor.go` piped through targeted `grep -n` | `s.coord = newCoordinator(` × 2 (boot, commitGeneration); `s.startScheduler(` × 3 (boot, commitGeneration, pre-Reconcile restart); `commitGeneration(` × 5 (1 decl + 4 call sites); `s.cfg = ` × 2 (boot, commitGeneration) — matches 07-09-SUMMARY's documented reconciliation of its own acceptance-criteria numeric self-inconsistency |
| Debt markers on supervisor files | `grep -riE "TBD\|FIXME\|XXX\|TODO\|HACK\|PLACEHOLDER"` over `kernel/supervisor/` | none found |
| Requirements traceability | `grep KERN-08\|UI-12 .planning/REQUIREMENTS.md` | Both `[x]` marked and listed "Complete" in the requirement table — this report's finding below means that marking is not yet fully earned |

### Gap Closure: Prior gaps[0] (Apply left s.coord/s.cfg stale after a rejected post-Reconcile save) — CONFIRMED FIXED

The single failed truth from the prior verification pass is now **✓ VERIFIED**, based on direct source reading and independently re-run behavioral tests:

- `kernel/supervisor/supervisor.go` read in full. A new unexported method `commitGeneration(cfg *config.Config)` performs exactly three steps in a fixed, documented order: `s.coord = newCoordinator(...)`, then `s.cfg = cfg`, then `s.startScheduler(cfg)` — the order is load-bearing because `startScheduler` reads `s.coord` at call time into the `syncer.Scheduler` value it constructs.
- The `ValidateMatchConfig` failure branch (the exact defect gaps[0] named) now calls `s.commitGeneration(newCfg)` instead of `s.startScheduler(oldCfg)`, adopting the new generation while still returning the vocabulary check's own error unchanged (D-09: the HTTP layer still answers 500 `apply_failed`).
- Both D-07 index-cleanup failure branches, and the success path, now route through the same `commitGeneration` call — confirmed by grep count (5 occurrences: 1 declaration + 4 call sites) over comment-stripped source.
- The pre-Reconcile failure branch (`Reconcile` itself fails) is deliberately left asymmetric — it still restarts the scheduler against `oldCfg`, because `Reconcile`'s own T-07-11 guarantee means the previously running plugin set is genuinely untouched on that path. Confirmed unchanged by direct read.
- `Apply`'s doc comment now states the two-regime contract (pre-Reconcile: old generation kept; post-Reconcile: new generation adopted, error still returned) and explicitly cites `gaps[0]`, `D-06`, `D-07` and `D-08`.
- Two new behavioral tests, independently re-run in this session: `TestApply_ValidateMatchConfigFailsAfterReconcile_CoordinatorTracksRelaunchedPlugin` (proves via a real `sup.Refresh()` call that the coordinator dispatches against the live relaunched plugin subprocess, `Status == "ok"`, not the one `Reconcile` killed) and `TestApply_RejectedSaveIsIdempotent_SecondApplyDoesNotRelaunchSubprocesses` (proves a retried rejected save churns no subprocesses). Both **pass**.
- The full `go test ./kernel/... -count=1` suite passes with every package `ok`, and the two pre-existing name-pinned tests (`TestApply_MidFlightSyncLeavesNoStrandedRunningRow`, `TestApply_RemovedInstance_PluginGoneAndIndexRowsGone`) are confirmed unmodified in their own bodies.

### New Finding: One Critical, Previously-Undetected Defect in the Same Function (07-REVIEW.md re-review after 07-09, committed `d016738`)

This phase's own code-review process, re-run after 07-09 landed, found **one new Critical-severity, unresolved issue** in `kernel/supervisor/supervisor.go`'s `Apply` — the same function 07-09 just repaired, on an adjacent seam. Independently confirmed here by reading the actual source at lines 307-377, not taken on the review document's word:

The D-07 removed-instance cleanup loop (deletes a removed instance's `items` and `sync_runs` rows) is positioned textually **after** the `ValidateMatchConfig` check inside `Apply`. When `ValidateMatchConfig` fails, the function returns at that point — the cleanup loop is **never reached** for that `Apply` call. `Host.Reconcile` has already committed by then (it ran and succeeded before `ValidateMatchConfig` was even called), so a removed instance's subprocess is already dead, but its index rows are never deleted. Compounding this: the `ValidateMatchConfig` failure branch calls `commitGeneration(newCfg)` — 07-09's own fix for the coordinator-staleness gap — which sets `s.cfg = newCfg`. On the very next `Apply` call, `oldCfg` is now `newCfg`, which already lacks the removed instance on both sides of the diff, so `removedInstances(oldCfg, newCfg2)` can never again compute it as removed. **There is no retry path** — the orphaned rows persist in the index permanently, or get inherited as phantom history if an instance is later re-added under the same `[sources.<id>]` key. A second, related failure mode exists in the same loop: if more than one instance is removed and an early one's `DeleteSourceItems`/`DeleteSyncRuns` call errors, the loop returns immediately, abandoning cleanup for every later-sorted instance in the same batch, with the same permanent-stranding consequence.

This is reachable through the documented, supported hand-edit + `POST /api/config/reload` flow (`ManageSourcesModal.svelte`'s "Reload config" button): a single hand-edited `config.toml` that both removes a `[sources.<id>]` block and introduces/typos a `[webspaces.<name>.match.<other-instance>]` field outside that instance's declared vocabulary hits this exact combination in one `Apply` call. Confirmed via `git show f25c4ab:kernel/supervisor/supervisor.go` that this ordering (vocabulary check before the D-07 loop) predates 07-09 entirely — **it is not a regression 07-09 introduced**, but it is the same class of defect 07-09 set out to close on the coordinator seam, left unclosed on this adjacent seam, and it directly violates `DeleteSourceItems`/`DeleteSyncRuns`'s own documented `T-07-13` guarantee. No test exercises this ordering: `kernel/supervisor/supervisor_test.go` was grepped for `removedInstances`/`DeleteSourceItems`/`DeleteSyncRuns` and produced zero hits outside `supervisor.go` and `kernel/index` itself — `go test ./kernel/...` passes cleanly precisely because nothing currently covers this path.

None of 07-09's own `must_haves.truths` claim to cover this — they address `s.host`/`s.coord`/`s.cfg` generation agreement and the shared commit-site ordering across all four adopting branches, both of which genuinely hold (independently confirmed above). This is a distinct invariant (D-07's index-hygiene guarantee, `T-07-13`) that the same textual restructuring did not happen to fix, and 07-09's own scope discipline note explicitly says it "touches only what closes gaps[0]" — this new finding is outside that declared scope, which is exactly why it survived 07-09 and was only caught by the review's own re-pass.

Two further lower-severity findings from the same review pass are non-blocking and pre-existing (confirmed via the review's own git-blame-style checks): `kernel/httpapi/webspaces.go`'s per-webspace `last_sync` is actually a global aggregate across every configured source rather than narrowed to each webspace's participating sources (Warning, predates this phase), and `web/src/lib/instance-id.ts`'s `deriveInstanceId` has no length/reserved-word bound (Warning, purely defense-in-depth, no demonstrated exploit). Neither blocks this verification; both are recorded in Anti-Patterns below for visibility.

### Observable Truths — Summary by Plan

| Plan | Focus | Truths | Verified | Behavior-unverified (human_needed) | Failed |
|---|---|---|---|---|---|
| 07-01 | Search→filter tracer, config write path | 17 | 12 | 5 (incl. 1 backstop) | 0 |
| 07-02 | Hot-apply, reload, plugin discovery (kernel-only) | 10 | 10 | 0 | 0 |
| 07-03 | Webspace switcher, create, root redirect | 12 | 4 | 8 (incl. 1 backstop) | 0 |
| 07-04 | Add-source picker, two-step connect, chip edit menu | 15 | 6 | 9 (incl. 1 backstop) | 0 |
| 07-05 | Manage sources, save-state guard, contract publication | 10 | 4 | 6 (incl. 1 backstop) | 0 |
| 07-06 | Gap closure: instance-id collision guard | 5 | 4 | 1 (backstop) | 0 |
| 07-07 | Gap closure: agent-route config staleness (prior CR-01) | 8 | 7 | 1 (backstop) | 0 |
| 07-08 | Gap closure: edit-modal stale-state resurfacing (prior CR-02) | 8 | 7 | 1 (backstop) | 0 |
| 07-09 | Gap closure: Supervisor.Apply generation consistency (prior gaps[0]) | 10 | 9 | 1 (backstop) | 0 |
| **Total** | | **95** | **63** | **32 (30 non-backstop + 7 backstop + 1 WR-01 human item folded into non-backstop human count above)** | **0** |

No individual enumerated must-have truth across any of the 9 plans is worded to cover the new D-07 cleanup-ordering finding — consistent with how gaps[0] itself was handled in the prior two rounds, it is tracked as its own gap (see Gaps Summary) because the decision-tree rule "blocker anti-pattern found → gaps_found" applies regardless of exact truth wording.

**Why the truth-level count went up:** 07-09 added its own 10 must-have truths (9 non-backstop + 1 backstop), closing the gap the prior verification round tracked outside the per-plan truth counts. The previously-failed gap is now folded into the 07-09 truths above rather than double-counted — but a NEW gap (the D-07 cleanup ordering defect) was found by this round's review and is tracked the same way gaps[0] itself was tracked in the prior two rounds: as a standalone anti-pattern-driven gap, not mapped to any specific truth's wording.

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `kernel/supervisor/supervisor.go` | Single shared `commitGeneration` commit site every adopting branch routes through, in the load-bearing order; corrected doc comment | ✓ VERIFIED (for gaps[0]'s scope) / ✗ DEFECTIVE (for the D-07 cleanup-ordering invariant, a distinct concern) | 390 lines (min_lines: 300 met). `commitGeneration` and its 4 call sites confirmed by grep; doc comment confirmed to cite gaps[0]/D-06/D-07/D-08. The NEW gap is a separate control-flow ordering issue in the same file — see Gaps |
| `kernel/supervisor/supervisor_test.go` | Reconcile-succeeds/ValidateMatchConfig-fails ordering test, plus retry-idempotency test | ✓ VERIFIED | 522 lines (min_lines: 380 met). Both named tests present and pass; pre-existing tests unmodified |
| `kernel/httpapi/agent.go` | Five agent handlers each resolving config per request (prior CR-01 fix, 07-07) | ✓ VERIFIED (regression check — file untouched by 07-09) | Not re-touched this round; prior verification's confirmation stands |
| `web/src/lib/edit-modal-state.ts` | Single seeding site, pure functions (prior CR-02 fix, 07-08) | ✓ VERIFIED (regression check — file untouched by 07-09) | Not re-touched this round; prior verification's confirmation stands |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `kernel/supervisor/supervisor.go` `Apply` ValidateMatchConfig-failure branch | `kernel/supervisor/supervisor.go` `commitGeneration` | Branch calls `s.commitGeneration(newCfg)` instead of restarting the scheduler against `oldCfg` | ✓ WIRED | Confirmed by direct read and grep |
| `kernel/supervisor/supervisor.go` `commitGeneration` | `kernel/supervisor/supervisor.go` `startScheduler` | Coordinator assigned before `startScheduler` is called, within `commitGeneration`'s body | ✓ WIRED | Confirmed: `s.coord = ...` precedes `s.startScheduler(cfg)` in the method body |
| `kernel/supervisor/supervisor_test.go` | `kernel/pluginhost` `ValidateMatchConfig` | Test fixture declares a match field outside the mock plugin's vocabulary so `Reconcile` succeeds and the vocabulary check then rejects | ✓ WIRED | Test passes, error message confirmed to name the foreign field and its webspace |
| `kernel/supervisor/supervisor.go` `Apply` ValidateMatchConfig-failure branch | `kernel/supervisor/supervisor.go` D-07 removed-instance cleanup loop | Sequential control flow — the loop is reached only if `ValidateMatchConfig` succeeds | ✗ NOT_WIRED (defect) | This is the mechanism of the new gap: the loop's execution is conditioned on a check whose failure should not gate it, since `Reconcile` (the event the loop responds to) has already committed regardless |

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|---|---|---|---|---|
| KERN-08 | 07-01, 07-02, 07-05, 07-06, 07-07, 07-08, 07-09 | Webspace/source-instance config editable via kernel API (non-secret only), hand-editing remains supported | ⚠️ BLOCKED | Config write path itself is sound and well-tested; the prior hot-apply staleness defect (gaps[0]) is now closed. But a distinct data-hygiene defect in the same hot-apply mechanism — permanently orphaned index rows for a removed instance whenever a removal and a match-vocabulary rejection land in the same save — is newly confirmed and unresolved. The requirement's "editable... while hand-editing remains supported" surface is functionally present; a reliability/data-integrity guarantee behind it is not fully intact. |
| UI-12 | 07-01, 07-03, 07-04, 07-05, 07-06, 07-07, 07-08, 07-09 | Webspace builder UI: pick plugin types, configure named instances, save the set, promote a live search into a permanent filter | ⚠️ BLOCKED (same root cause) | All UI-facing must-haves for this requirement across 07-03/04/05/06/07/08 are VERIFIED or behavior-unverified-pending-human-check; none FAILED. Blocked at the same kernel-level file, since the hand-edit + Reload config flow this UI explicitly supports (ManageSourcesModal.svelte) is the documented path that reaches the new defect. |

No orphaned requirement IDs found — every ID mapped to Phase 7 in REQUIREMENTS.md (KERN-08, UI-12) appears in at least one plan's `requirements:` frontmatter field (including 07-09), and every plan's declared requirements are accounted for above.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `kernel/supervisor/supervisor.go` | 325-355 | D-07 removed-instance index cleanup loop is positioned after (gated by) the `ValidateMatchConfig` check, so a rejection there skips the loop entirely; the loop also aborts on its first per-instance cleanup error rather than continuing to the next instance | 🛑 Blocker | Permanently strands a removed instance's `items`/`sync_runs` rows once `s.cfg` advances past the removal — no retry path exists, and a re-added instance under the same id could inherit phantom history, violating documented `T-07-13` |
| `kernel/httpapi/webspaces.go` | 36-64 | `WebspacesHandler`'s `last_sync` is computed once globally (`LatestSyncRunPerSource` + `aggregateSyncStatus`) and assigned to every webspace, rather than narrowed to each webspace's own participating sources | ⚠️ Warning | A webspace whose own sources are healthy can report `"error"` because of an unrelated source elsewhere in the config. Predates this phase (confirmed via `git diff` scoping); not introduced or touched by 07-01..07-09 |
| `web/src/lib/instance-id.ts` | 40-46 | `deriveInstanceId` has no upper bound on output length and no reservation for kernel-internal sentinel names (e.g. `__trial__`) | ⚠️ Warning | Defense-in-depth gap only — no demonstrated exploit; the one collision that matters today (an actual instance-id clash) is already checked separately |
| `kernel/config/writer.go` | 57-60 | `Store.WriteCanonical`'s temp file relies implicitly on `os.CreateTemp`'s default `0o600` mode rather than declaring it | ℹ️ Info | No behavior issue today; a future temp-file-helper swap could silently regress permissions with nothing to catch it |
| `web/src/routes/w/[webspace]/+page.svelte` | 166-188 | `handleChipEdit`'s async `describePlugin` await has no generation/sequence guard, unlike every other async call site in the same file | ⚠️ Warning (carried from prior round, unchanged — 07-09 did not touch `web/src`) | Display-only race (stale vocabulary suggestions momentarily shown); does not corrupt saved config since `editInstance` always reflects the latest click |

No debt markers (`TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`) found in `kernel/supervisor/` or any file touched by 07-09.

### Human Verification Required

See `behavior_unverified_items` in frontmatter for the full list (32 items: 31 carried forward unchanged from the prior verification round since 07-09 did not touch any of that code, plus 1 new backstop from 07-09 itself, D3). All require a live `make dev` session, most exercising a UI flow end-to-end against a running kernel — none were available in this verification environment. None of these items are FAILED; they are present-and-wired code paths whose runtime behavior a static/test-suite check cannot fully exercise.

### Gaps Summary

One Critical-severity gap blocks this phase: `kernel/supervisor/supervisor.go`'s `Apply` function's D-07 removed-instance index-cleanup loop is positioned after the `ValidateMatchConfig` check, so a rejected save that also removes a source instance in the same call never cleans up that instance's index rows — and because the rejection branch (correctly, per 07-09's own fix) advances `s.cfg` to the new config, there is no later opportunity to detect the removal and retry the cleanup. The orphaned `items`/`sync_runs` rows persist permanently. This is reachable through the documented, supported hand-edit + `POST /api/config/reload` flow, violates the `T-07-13` guarantee `DeleteSourceItems`/`DeleteSyncRuns` exist to provide, and sits in the exact function this phase's hot-apply mechanism (D-06/D-07) is built around. It predates 07-09 (confirmed via `git show`) and is not a regression introduced by this round's gap-closure plan — but it was not covered by 07-09's own `must_haves`, was not caught by the original verification pass or by the prior two gap-closure rounds (07-06/07-07/07-08), and was only surfaced by this phase's own follow-up code review after 07-09 landed. No test currently exercises this ordering.

Prior round's gaps[0] (Apply leaving `s.host`/`s.coord`/`s.cfg` disagreeing after a rejected post-Reconcile save) is now confirmed closed by direct source reading and two independently re-run behavioral tests — this is genuine progress, not a wash: the phase has now closed three consecutive Critical/high-severity findings across three gap-closure rounds (CR-01 agent-route staleness in 07-07, CR-02 edit-modal stale state in 07-08, and gaps[0] coordinator staleness in 07-09), and each closure held up under this round's fresh regression checks. The remaining gap is narrower in surface (one specific interleaving: a same-save removal + match-vocabulary rejection) but no less real, since it silently and permanently violates a documented data-hygiene invariant with no recovery path once triggered.

---

_Verified: 2026-08-08T20:30:00Z_
_Verifier: Claude (gsd-verifier)_
