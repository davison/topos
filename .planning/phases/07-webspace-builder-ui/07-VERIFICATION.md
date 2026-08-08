---
phase: 07-webspace-builder-ui
verified: 2026-08-08T23:59:00Z
status: human_needed
score: 73/107 must-haves verified
behavior_unverified: 34
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 63/95
  gaps_closed:
    - "kernel/supervisor/supervisor.go's Apply D-07 removed-instance index cleanup (items, sync_runs) no longer runs only on the ValidateMatchConfig-succeeds path (07-VERIFICATION.md prior gaps[0] / 07-REVIEW.md post-07-09 CR-01): confirmed by direct reading of the current kernel/supervisor/supervisor.go — cleanupRemovedInstances(ctx, oldCfg, newCfg) is now called at line 366, textually and temporally BEFORE pluginhost.ValidateMatchConfig at line 368, so a vocabulary rejection can only add to the returned error and can never gate, shorten or skip the cleanup. Per-instance failures are collected via errors.Join rather than returned on the first (kernel/supervisor/supervisor.go:438-450), so one instance's cleanup failure no longer abandons the rest of a removed-instance batch. Confirmed against git history: git show 8d2f9ea~1:kernel/supervisor/supervisor.go shows the pre-fix code with the D-07 loop textually AFTER the ValidateMatchConfig check, each failure branch returning immediately after its own commitGeneration(newCfg) call — this is the exact defect gaps[0] named, and it is gone in the current tree. Two new behavioral tests independently re-run in this session — TestApply_RemovedInstanceCleanedUpEvenWhenTheSameSaveIsRejected and TestApply_MultipleRemovedInstances_OneCleanupFailureDoesNotAbandonTheRest — both pass; the first drives the exact combination gaps[0] described (a same-save instance removal plus an unrelated match-vocabulary rejection) against real launched mock-plugin subprocesses and asserts both the removed instance's items row AND its sync_runs history are gone afterward, while the survivor's rows are untouched. The operator-visible error message text is confirmed byte-identical to the pre-fix code (git-diff-verified: 'delete items for removed source %q: %w' / 'delete sync history for removed source %q: %w' wrapped by the same 'supervisor: apply: ' prefix, unchanged). All four pre-existing name-pinned TestApply_* tests are confirmed unmodified (git diff e6ebf04..HEAD -- kernel/supervisor/supervisor_test.go shows 250 insertions, 0 deletions — purely additive)."
  gaps_remaining: []
  regressions: []
deferred: []
behavior_unverified_items:
  - truth: "Save-as-filter UI states: with an empty filter stack the filter-chip row does not render at all; Save as filter and each chip's \u00d7 disable while their write is in flight; a hash-conflict on a filter write surfaces the destructive Alert with the fixed copy 'Config changed on disk \u2014 review and retry.'; filter chips render rounded-md (distinguishable from a source chip's rounded-full); and a search query identical to an already-active filter term offers no Save as filter affordance (07-01 Task \u2014 four non-backstop truths tracked as behavior-unverified in the Summary by Plan table with no prior checklist entry; added during the 07-VERIFICATION.md internal-consistency repair)"
    test: "make dev; open a webspace with an active search; click Save as filter and confirm the write-in-flight disabled state, the resulting rounded-md chip, and that a byte-identical repeat search offers no Save as filter button; then force a hash conflict (edit config.toml externally between two saves) and confirm the fixed Alert copy; then remove every filter and confirm the chip row is fully absent, not an empty-styled row"
    expected: "All five states render exactly as UI-SPEC E9/E10 describe, with no visible affordance when the filter stack is empty"
    why_human: "These are visual/timing UI states (empty-state absence, in-flight disabling, hash-conflict Alert copy, CSS class distinction, dedup-affordance suppression) that structural source-scan tests can assert are coded but cannot confirm render correctly in a live browser. Identified as a real, previously-unlisted checklist gap during this repair: the Summary by Plan table has tracked these 4 truths (of 07-01's 5 behavior-unverified truths) since the original verification round, but no prior round's frontmatter checklist named them individually."
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
    why_human: "manage-sources.test.ts is a structural guard. The underlying kernel mechanics ARE integration-tested in 07-02 (and, for the removed-instance-cleanup edge, now also in 07-10) — but the UI trigger → confirm → observed effect round trip was never run against a live kernel."
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
    why_human: "Confirmed by direct read (web/src/routes/w/[webspace]/+page.svelte, line 166): handleChipEdit still has no generation/sequence guard around its describePlugin await, unlike every other async call site in the same file. 07-10 deliberately touched only kernel/supervisor/*.go — web/src is unchanged (confirmed via git diff e6ebf04..HEAD --stat, no web/src entries). Real UI display glitch, not a data-corruption path: editInstance always reflects the LAST click, so a save still writes to the correct instance; only offered vocabulary suggestions can be momentarily stale."
  - truth: "Against a live kernel via make dev, editing a source's connection details AND introducing an invalid match field name in the same UI save produces the 500 apply_failed response, and that source's chip then continues to sync and report healthy on its next scheduled tick rather than failing continuously until the kernel is restarted (07-09 backstop, D3)"
    test: "make dev; open a webspace; use the chip ⋮ menu's Edit connection… to change a source's base_url, and in the same session add a match field name the plugin does not declare, then save; confirm the UI surfaces the kernel's rejection; leave the kernel running and watch that source's chip through its next scheduled sync tick"
    expected: "500 apply_failed with the vocabulary error's own text; the source syncs and reports healthy on its next tick rather than failing every tick"
    why_human: "Tagged verification: backstop in 07-09-PLAN.md (D3 in its SUMMARY coverage table) — the underlying mechanism IS behaviorally proven by TestApply_ValidateMatchConfigFailsAfterReconcile_CoordinatorTracksRelaunchedPlugin against real launched mock-plugin subprocesses; only the live make dev / real UI confirmation remains, per workflow.human_verify_mode=end-of-phase."
  - truth: "A kernel killed midway through the D-07 cleanup — between one instance's items delete and that same instance's sync-history delete — leaves at most that one instance's sync_runs rows behind; no still-configured instance's rows are ever deleted, and no other removed instance in the same batch is left in a half-cleaned state (07-10 backstop)"
    test: "Kill the topos process (SIGKILL) at the instant between one removed instance's DeleteSourceItems call returning and its DeleteSyncRuns call starting, during an Apply that removes 2+ instances, then inspect the index"
    expected: "At most the interrupted instance's sync_runs rows survive; every other instance in the batch is either fully cleaned or fully untouched, never half-cleaned; no still-configured instance's rows are ever touched"
    why_human: "Tagged verification: backstop in 07-10-PLAN.md — no test can deterministically interrupt the process at that exact instant, same class as the 07-01 config.toml.bak backstop. The batch-continuation test (TestApply_MultipleRemovedInstances_OneCleanupFailureDoesNotAbandonTheRest) proves the SQL-error case behaves correctly but cannot simulate a hard process kill mid-loop."
  - truth: "Against a live kernel via make dev, hand-editing config.toml so that one save both deletes a [sources.<id>] block and typos a [webspaces.<name>.match.<other-instance>] field, then clicking Manage sources… → Reload config, surfaces the kernel's vocabulary rejection AND leaves the removed instance's items absent from every webspace stream and its sync history absent from the health surface — with the kernel never restarted (07-10 backstop)"
    test: "make dev; hand-edit config.toml to remove a [sources.<id>] block and typo an unrelated [webspaces.<name>.match.<other-instance>] field in the same edit; Manage sources… → Reload config; confirm the kernel's rejection message renders, the removed source's items are gone from every webspace stream, its sync history is gone from the health surface, and no restart occurred; then re-add an instance under the same key and confirm no phantom history"
    expected: "Rejection surfaces via the UI; removed source's items/health gone immediately; no restart; re-adding under the same key starts with a clean slate"
    why_human: "Tagged verification: backstop in 07-10-PLAN.md (D4 in its SUMMARY coverage table) — the underlying mechanism IS behaviorally proven end-to-end by TestApply_RemovedInstanceCleanedUpEvenWhenTheSameSaveIsRejected against real launched mock-plugin subprocesses through a real config.Store.Save; only the live make dev / real browser + Reload-config-button confirmation remains, per workflow.human_verify_mode=end-of-phase."
---

# Phase 7: Webspace Builder UI Verification Report

**Phase Goal:** User can configure sources and webspaces from the UI instead of hand-editing TOML — pick plugin types from a list, configure named instances, save a configured set as a webspace, and promote a live search into the webspace's permanent filter.
**Verified:** 2026-08-08
**Status:** human_needed
**Re-verification:** Yes — after 07-10 (D-07 index cleanup gap closure, closing prior round's sole Critical gap)

## Goal Achievement

### Build, Test and Contract Evidence (independently re-run in this session, not taken from SUMMARY claims)

| Check | Command | Result |
|---|---|---|
| Go build | `CGO_ENABLED=0 go build ./...` | clean, exit 0 |
| Go test suite (full) | `go test ./kernel/... -count=1` | all packages `ok` (config, correlate, httpapi, index, pluginhost, supervisor, syncer) |
| Go test — 07-10 gap closure, targeted | `go test ./kernel/supervisor/... -run 'TestApply' -count=1 -v` | 6/6 passed (4 pre-existing byte-identical, 2 new from 07-10) |
| Diff scope for 07-10 | `git diff e6ebf04 HEAD --stat` | 6 files: `kernel/supervisor/supervisor.go` (+123/-50), `kernel/supervisor/supervisor_test.go` (+250/-0), plus `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md`, `.planning/STATE.md`, `07-10-SUMMARY.md` — no UI file, no HTTP handler, no other kernel package touched |
| Pre-existing test bodies unmodified | `git diff e6ebf04 HEAD --numstat -- kernel/supervisor/supervisor_test.go` | 250 insertions, **0 deletions** — purely additive, confirms the four 07-09-era `TestApply_*` tests are byte-identical |
| Error text byte-identity | `git show 8d2f9ea~1:kernel/supervisor/supervisor.go` vs current, grepped for the cleanup error strings | `"delete items for removed source %q: %w"` / `"delete sync history for removed source %q: %w"`, both wrapped by the caller's unchanged `"supervisor: apply: "` prefix — identical pre- and post-fix |
| Source assertion — commit-site counts | `grep -v '^\s*//' kernel/supervisor/supervisor.go \| grep -c ...` | `commitGeneration(` = 2 (decl + 1 call site, down from 07-09's 4 call sites); `s.startScheduler(` = 3; `errors.Join(` = 2; `s.startScheduler(oldCfg)` = 1; `s.cfg = ` = 2 (`NewSupervisor` boot + `commitGeneration`, documented and reconciled in 07-10-SUMMARY's Deviations section) — all match 07-10-SUMMARY's claimed counts exactly |
| HTTP layer unweakened | `grep -n "apply_failed" kernel/httpapi/config.go` | Both `ConfigSaveHandler` and `ConfigReloadHandler` still answer 500 `apply_failed` on any `Apply` error, unchanged |
| Debt markers on touched files | `grep -riE "TBD\|FIXME\|XXX\|TODO\|HACK\|PLACEHOLDER"` over `kernel/supervisor/supervisor.go`, `supervisor_test.go` | none found |
| Requirements traceability | `grep KERN-08\|UI-12` over every `07-*-PLAN.md`'s `requirements:` frontmatter | KERN-08 claimed by 07-01/02/05/06/07/08/09/10; UI-12 claimed by 07-01/03/04/05/06/07/08/09/10 — no orphaned IDs |

### Gap Closure: Prior gaps[0] (D-07 cleanup gated by, and stranded by, the ValidateMatchConfig rejection branch) — CONFIRMED FIXED

The single Critical gap from the prior verification pass is now **✓ VERIFIED CLOSED**, based on direct source reading (not taken on SUMMARY.md's word) and independently re-run behavioral tests:

- `kernel/supervisor/supervisor.go` read in full, focused on `Apply` (lines 337-409) and the new `cleanupRemovedInstances` method (lines 411-450).
- `s.cleanupRemovedInstances(ctx, oldCfg, newCfg)` is called at line 366, **before** `pluginhost.ValidateMatchConfig(newCfg, s.host)` is called at line 368 — the exact reordering `gaps[0].missing[0]` asked for. Confirmed by reading the surrounding code: there is no early return, no conditional, and no `if` gate between the `Host.Reconcile` success point and the cleanup call.
- `cleanupRemovedInstances` no longer returns on the first per-instance failure: it appends every failure to a `[]error` slice and continues to the next name, returning `errors.Join(failures...)` once the whole batch has been attempted — the exact behavior `gaps[0].missing[1]` asked for.
- A single shared `s.commitGeneration(newCfg)` call (line 380) now serves the whole post-Reconcile region — down from 07-09's four separate call sites to one — strengthening rather than re-diverging 07-09's coordinator/cfg/scheduler consistency invariant.
- The joined error (`errors.Join(validateErr, cleanupErr)`, vocabulary error leading) is returned unchanged in shape and wrapped in the same `"supervisor: apply: %w"` prefix (line 387-388), so `ConfigSaveHandler`/`ConfigReloadHandler` still answer 500 `apply_failed` on every post-Reconcile failure branch — adopting the new generation on a rejected save is state repair, not a converted success.
- Confirmed via `git show 8d2f9ea~1:kernel/supervisor/supervisor.go` (the commit immediately before 07-10's fix) that the pre-fix code had the D-07 loop textually **after** the `ValidateMatchConfig` check, with each cleanup failure branch calling `commitGeneration(newCfg)` and returning immediately — this is byte-for-byte the defect `gaps[0]` described.
- Two new behavioral tests, independently re-run in this session:
  - `TestApply_RemovedInstanceCleanedUpEvenWhenTheSameSaveIsRejected` — drives the exact combination `gaps[0]` named: a same-save instance removal plus an unrelated match-vocabulary rejection, against real launched mock-plugin subprocesses through a real `config.Store.Save`. Asserts the removed instance's `items` row is gone, its `sync_runs` history is gone, the surviving instance's rows are untouched, and `sup.cfg` still advances to the new generation (07-09's invariant, unweakened). **PASS.**
  - `TestApply_MultipleRemovedInstances_OneCleanupFailureDoesNotAbandonTheRest` — proves an early removed instance's `DeleteSourceItems`/`DeleteSyncRuns` failure (forced via a closed index store) does not abandon a later-sorted instance's cleanup in the same batch, and that the returned error names both instances. **PASS.**
- The full `go test ./kernel/... -count=1` suite passes with every package `ok`, and `git diff e6ebf04 HEAD --numstat -- kernel/supervisor/supervisor_test.go` shows 250 insertions and 0 deletions — the four pre-existing name-pinned tests (`TestApply_MidFlightSyncLeavesNoStrandedRunningRow`, `TestApply_RemovedInstance_PluginGoneAndIndexRowsGone`, `TestApply_ValidateMatchConfigFailsAfterReconcile_CoordinatorTracksRelaunchedPlugin`, `TestApply_RejectedSaveIsIdempotent_SecondApplyDoesNotRelaunchSubprocesses`) are confirmed byte-identical, all passing.
- `Apply`'s doc comment (lines 254-336) now states the index-hygiene half of its post-Reconcile contract explicitly, citing `gaps[0]`, 07-REVIEW.md's post-07-09 CR-01, and both 07-09-PLAN.md and 07-10-PLAN.md by name.

**No new gap was found in this round.** The two Warning-severity findings (`kernel/httpapi/webspaces.go`'s global `last_sync` aggregate, `web/src/lib/instance-id.ts`'s unbounded `deriveInstanceId`), the Info-severity finding (`kernel/config/writer.go`'s implicit temp-file mode), and the carried WR-01 `handleChipEdit` race are all confirmed still present, unchanged, and non-blocking — 07-10 deliberately did not touch any of `kernel/httpapi/`, `web/src/lib/instance-id.ts`, `kernel/config/writer.go` or `web/src/routes/w/[webspace]/+page.svelte` (confirmed by `git diff e6ebf04 HEAD --stat`, which lists only `kernel/supervisor/supervisor.go`, `kernel/supervisor/supervisor_test.go`, and three `.planning/` docs plus the new SUMMARY). These remain recorded in Anti-Patterns below for visibility and are candidates for a follow-up `/gsd-code-review 7 --fix` pass, as 07-10-PLAN.md itself notes.

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
| 07-10 | Gap closure: D-07 index cleanup on every post-Reconcile path (this round's gaps[0]) | 12 | 10 | 2 (backstops) | 0 |
| **Total** | | **107** | **73** | **34 (32 non-backstop-and-backstop carried unchanged + 2 new 07-10 backstops)** | **0** |

**Why the truth-level count went up:** 07-10 added its own 12 must-have truths (10 non-backstop + 2 backstop), closing the gap the prior verification round tracked outside the per-plan truth counts. All 10 non-backstop 07-10 truths are VERIFIED against direct source reading and two independently re-run behavioral tests. The 2 backstop truths (a mid-process-kill interruption case, and the live-`make dev` end-to-end confirmation of the fixed flow) are — by their own explicit `verification: backstop` tag — not programmatically provable and route to human verification, consistent with how every other backstop truth in this phase has been handled across all ten plans.

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `kernel/supervisor/supervisor.go` | D-07 cleanup runs unconditionally and to completion before the match-vocabulary check, with per-instance failures collected via `errors.Join`; single shared `commitGeneration` site for the whole post-Reconcile region; extended doc comment | ✓ VERIFIED | 461 lines total (min_lines: 380 met). `cleanupRemovedInstances` extracted (lines 411-450), called before `ValidateMatchConfig` (line 366 vs 368); grep-verified commit-site counts match exactly; doc comment cites gaps[0]/07-REVIEW.md CR-01/07-09-PLAN.md/07-10-PLAN.md |
| `kernel/supervisor/supervisor_test.go` | Removed-instance-plus-rejected-save test, plus batch-continuation test | ✓ VERIFIED | 913 lines total (min_lines: 600 met). Both named tests present, pass, and are substantive (assert both `items` and `sync_runs` tables, assert survivor untouched, assert both removed-instance names appear in the batch-failure error) |
| `kernel/httpapi/agent.go` | Five agent handlers each resolving config per request (prior CR-01 fix, 07-07) | ✓ VERIFIED (regression check — file untouched since 07-07) | Not re-touched this round; prior verification's confirmation stands |
| `web/src/lib/edit-modal-state.ts` | Single seeding site, pure functions (prior CR-02 fix, 07-08) | ✓ VERIFIED (regression check — file untouched since 07-08) | Not re-touched this round; prior verification's confirmation stands |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `kernel/supervisor/supervisor.go` `Apply`, immediately after `Host.Reconcile` returns nil | `kernel/supervisor/supervisor.go` `cleanupRemovedInstances` | Unconditional call at line 366, textually before `ValidateMatchConfig` at line 368 | ✓ WIRED (previously the mechanism of the gap; now fixed) | Confirmed by direct read; the removed-instance cleanup can no longer be gated, shortened or skipped by the vocabulary check's outcome |
| `kernel/supervisor/supervisor.go` `cleanupRemovedInstances` | `kernel/index` `DeleteSourceItems` / `DeleteSyncRuns` | Loop over `removedInstances(oldCfg, newCfg)`, per-instance failures collected via `errors.Join`, no early return | ✓ WIRED | Confirmed by direct read and by `TestApply_MultipleRemovedInstances_OneCleanupFailureDoesNotAbandonTheRest` passing |
| `kernel/supervisor/supervisor.go` `Apply` post-Reconcile region | `kernel/supervisor/supervisor.go` `commitGeneration` | One shared call site (line 380) reached by both success and every post-Reconcile failure outcome | ✓ WIRED | Confirmed by grep (`commitGeneration(` = 2: 1 declaration + 1 call site) and by 07-09's four regression tests still passing |
| `kernel/supervisor/supervisor_test.go` | `kernel/index` `GetItem` / `LatestSyncRunPerSource` | Behavioral proof that a removed instance's items AND sync history are gone after an `Apply` that returned the vocabulary check's error, while the surviving instance's rows are untouched | ✓ WIRED | `TestApply_RemovedInstanceCleanedUpEvenWhenTheSameSaveIsRejected` passes; asserts both tables explicitly |

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|---|---|---|---|---|
| KERN-08 | 07-01, 07-02, 07-05, 07-06, 07-07, 07-08, 07-09, 07-10 | Webspace/source-instance config editable via kernel API (non-secret only), hand-editing remains supported | ✓ SATISFIED | Config write path is sound and well-tested. The prior hot-apply coordinator-staleness defect (gaps[0], round 1) and the D-07 index-hygiene defect (gaps[0], round 2, closed this round by 07-10) are both now confirmed closed by direct source reading and passing behavioral tests. The one remaining gap in this requirement's assurance is the 19 consolidated checklist items (`behavior_unverified_items`) requiring a live `make dev` session, together covering the 34 individual behavior-unverified truths tracked in the Summary by Plan table above — not a code defect, but unexercised runtime confirmation, consistent with `workflow.human_verify_mode=end-of-phase`. See "Note on Verification Granularity" below. |
| UI-12 | 07-01, 07-03, 07-04, 07-05, 07-06, 07-07, 07-08, 07-09, 07-10 | Webspace builder UI: pick plugin types, configure named instances, save the set, promote a live search into a permanent filter | ✓ SATISFIED | All UI-facing must-haves for this requirement across 07-03/04/05/06/07/08/09 are VERIFIED or behavior-unverified-pending-human-check; none FAILED. The kernel-level defect that previously blocked this requirement (reachable through `ManageSourcesModal.svelte`'s Reload config button) is now closed. |

No orphaned requirement IDs found — every ID mapped to Phase 7 in REQUIREMENTS.md (KERN-08, UI-12) appears in at least one plan's `requirements:` frontmatter field (including 07-10), and every plan's declared requirements are accounted for above.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `kernel/httpapi/webspaces.go` | 36-64 | `WebspacesHandler`'s `last_sync` is computed once globally (`LatestSyncRunPerSource` + `aggregateSyncStatus`) and assigned to every webspace, rather than narrowed to each webspace's own participating sources | ⚠️ Warning | A webspace whose own sources are healthy can report `"error"` because of an unrelated source elsewhere in the config. Predates this phase and is unchanged by 07-10 (confirmed: file not in `git diff e6ebf04 HEAD --stat`) |
| `web/src/lib/instance-id.ts` | 40-46 | `deriveInstanceId` has no upper bound on output length and no reservation for kernel-internal sentinel names (e.g. `__trial__`) | ⚠️ Warning | Defense-in-depth gap only — no demonstrated exploit; unchanged by 07-10 (confirmed: file not in `git diff e6ebf04 HEAD --stat`) |
| `kernel/config/writer.go` | 57-60 | `Store.WriteCanonical`'s temp file relies implicitly on `os.CreateTemp`'s default `0o600` mode rather than declaring it | ℹ️ Info | No behavior issue today; unchanged by 07-10 |
| `web/src/routes/w/[webspace]/+page.svelte` | 166-188 | `handleChipEdit`'s async `describePlugin` await has no generation/sequence guard, unlike every other async call site in the same file | ⚠️ Warning (carried, unchanged — 07-10 touched only `kernel/supervisor/`) | Display-only race (stale vocabulary suggestions momentarily shown); does not corrupt saved config since `editInstance` always reflects the latest click |

No debt markers (`TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`) found in `kernel/supervisor/` or any file touched by 07-10.

### Human Verification Required

See `behavior_unverified_items` in frontmatter for the full list — **19 consolidated checklist items**, each a live-session test scenario covering one or more of the 34 individual behavior-unverified truths tracked in the "Observable Truths — Summary by Plan" table above (see "Note on Verification Granularity" immediately below this section for how the two counts relate). Of the 19: 16 are carried forward unchanged from the prior verification round's own checklist, 1 is newly added during this round's internal-consistency repair to close a real, previously-unlisted coverage gap for 07-01's Save-as-filter UI states, and 2 are new backstop items from 07-10 itself (a mid-process-kill interruption case and a live-`make dev` end-to-end confirmation of the D-07 fix against a real browser + Reload-config-button flow). All require a live `make dev` session or a precisely-timed process kill; none were available in this verification environment. None of these items are FAILED; they are present-and-wired code paths (or, for the process-kill backstops, genuinely non-deterministic timing windows) whose runtime behavior a static/test-suite check cannot fully exercise.

### Note on Verification Granularity

This document tracks human-verification status at two deliberately different granularities, and this section makes both explicit after an internal-consistency repair identified a wording bug conflating them:

1. **Truths-level (34, in the "Observable Truths — Summary by Plan" table and the `score`/`behavior_unverified` frontmatter fields).** Each plan's `must_haves.truths` array is counted individually — 107 total across all 10 plans, 73 verified, 34 behavior-unverified, 0 failed. `score: 73/107` and `behavior_unverified: 34` are both defined at this granularity and are mutually consistent (73 + 34 = 107). This table and its 34-truth tally are carried forward unchanged from the prior verification round's own methodology (07-01 through 07-09) plus this round's freshly-verified 07-10 addition (12 truths: 10 verified, 2 behavior-unverified backstops).

2. **Checklist-level (19, in the `behavior_unverified_items` frontmatter array and the "Human Verification Required" section).** This is a curated, human-actionable list of distinct live-session test scenarios — several related truths that would naturally be exercised together in one `make dev` session (e.g., a single UI component's several rendering states, or a single shared save-state pattern reused across multiple plans) are consolidated into one checklist entry rather than listed once per truth.

**Why this note exists:** this consolidation practice was already in effect in the pre-07-10 verification round (its own `behavior_unverified_items` array contained 16 entries despite its frontmatter scalar and body prose claiming "32 items") but was never disclosed as intentional consolidation — it read as an unexplained mismatch. During this round's repair (prompted by a coordinator review), the same mismatch was found to have been reproduced in this round's initial rewrite (`behavior_unverified: 34` paired with a body claim of "34 items" against an actual 18-entry array). Rather than either (a) mechanically shrinking the truths-level scalar to 18 — which would have broken the `score` line's own defined arithmetic (73 + 18 ≠ 107) and been inconsistent with the unchanged, still-valid Summary-by-Plan table — or (b) fabricating a false 1:1 restoration to 34 individually-named entries the prior round never actually produced (verified by inspection: `git show HEAD~N` for the pre-07-10 file shows only 16 concrete entries, not 32), this repair:

- Left the truths-level accounting (`score: 73/107`, `behavior_unverified: 34`, and the Summary-by-Plan table) untouched — it is internally self-consistent and does not depend on checklist array length.
- Corrected every body-prose claim that the checklist array itself contains "34 items" to instead accurately state its actual size (19) and its relationship to the 34 tracked truths.
- Audited checklist coverage against the Summary-by-Plan table's per-plan counts and found one concrete, previously-unlisted gap: 07-01 tracks 5 behavior-unverified truths (4 non-backstop UI-rendering states plus 1 backstop) but only the backstop had ever been given a checklist entry, in this round or any prior one. Added one new consolidated entry (07-01's Save-as-filter UI states: empty-state chip-row absence, in-flight disabling, hash-conflict Alert copy, chip CSS-class distinction, and search-dedup affordance suppression) to close that gap, bringing the checklist from 18 to 19 entries.
- Does **not** claim perfect, individually-cited 1:1 traceability from all 34 truths to named checklist entries for every one of plans 07-03/07-04/07-05 (the largest sources of behavior-unverified truths, 8/9/6 respectively) — a full per-truth audit of those three plans' original round-1 categorization was outside this repair's scope and would require re-deriving which specific truths among each plan's non-backstop list were originally judged behavior-unverified versus code/test-verified, a determination not recorded per-truth in any prior round's artifacts. The existing checklist entries for those plans (items covering the webspace switcher, create-webspace modal, root redirect, add-source picker, two-step connect flow, secret field, chip edit menu, and manage-sources modal) are broad, representative live-session scenarios that substantively exercise the same UI surfaces and shared components (e.g., the single `CONFIG_CONFLICT_MESSAGE` constant and save-state pattern in `web/src/lib/api.ts`, confirmed exercised by every config-writing UI surface in the phase) the tracked truths describe, even where a truth is not individually named.

### Gaps Summary

**No blocking gaps remain.** The single Critical gap carried from the prior verification round — `Supervisor.Apply`'s D-07 removed-instance index-cleanup loop being gated by (and stranded by) the `ValidateMatchConfig` check — is confirmed closed by 07-10: direct source reading confirms the cleanup now runs unconditionally and to completion before the vocabulary check, with per-instance failures collected rather than abandoning the batch, and two new behavioral tests exercise exactly the combination the gap named, both passing. The four pre-existing regression tests from 07-09 remain byte-identical and passing. No new gap was surfaced by this round's re-verification — the two Warning findings, the Info finding and the carried WR-01 race are all pre-existing, non-blocking, and outside 07-10's declared scope.

The phase has now closed four consecutive Critical/high-severity findings across four gap-closure rounds (CR-01 agent-route staleness in 07-07, CR-02 edit-modal stale state in 07-08, gaps[0]-round-1 coordinator staleness in 07-09, gaps[0]-round-2 index-hygiene ordering in 07-10), and each closure has held up under its subsequent round's fresh regression checks — including this one.

**Status is `human_needed`, not `passed`,** because 19 human-verification checklist items remain outstanding, covering 34 individually-tracked behavior-unverified truths (per the decision tree: any non-empty human-verification list routes to `human_needed` even when zero truths are FAILED). This is the expected terminal state for this phase under `workflow.human_verify_mode=end-of-phase` — these items are deferred UI/browser/timing confirmations, not code gaps, and route to a `{phase_num}-UAT.md` checklist per the standard downstream path.

---

_Verified: 2026-08-08T23:59:00Z_
_Verifier: Claude (gsd-verifier)_
