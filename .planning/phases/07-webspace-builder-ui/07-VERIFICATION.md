---
phase: 07-webspace-builder-ui
verified: 2026-08-08T16:56:33Z
status: gaps_found
score: 54/85 must-haves verified
behavior_unverified: 31
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 40/69
  gaps_closed:
    - "Revoking or granting a source's agent.read/agent.handoff grant through the UI's hot-apply config save takes effect on /agent/v1 without a kernel restart (07-VERIFICATION.md prior gaps[0] / 07-REVIEW.md original CR-01): confirmed by direct reading of kernel/httpapi/agent.go — every one of the five /agent/v1 handlers (agentSourcesHandler, agentWebspacesHandler, agentStreamHandler, agentItemHandler, agentRenditionHandler) resolves cfg := cfgStore.Expanded() as the first statement of its own per-request closure, and MountAgentRoutes holds no config value at all — plus two independently re-run behavioral tests (TestAgentLiveConfig_RevokedReadGrantTakesEffectWithoutRestart, TestAgentLiveConfig_NewlyGrantedSourceIsVisibleWithoutRestart) that round-trip a revoke/grant through the SAME already-constructed router with no restart, and a structural AST guard (TestAgentGuard_EveryHandlerResolvesConfigPerRequest) pinning the invariant. All three pass."
    - "Editing an existing source through the chip menu never silently re-saves a value the user already typed and then discarded via Cancel (07-VERIFICATION.md prior gaps[1] / 07-REVIEW.md CR-02): confirmed by direct reading of web/src/routes/w/[webspace]/+page.svelte — handleEditClose and handleEditSaved both now call a single resetEditSession() that nulls editInstance (and editMode/editVocabulary), which drops the {#if configResponse && editInstance}-gated EditSourceModal subtree entirely on every close path (not relying on the {#key} value happening to change). EditSourceModal additionally seeds via two pure, exported helpers (web/src/lib/edit-modal-state.ts: seedConnectionValues, seedMatchBlock) used both at $state initialisation and inside a defensive reset-on-open $effect that tracks only the open flag (every config read wrapped in untrack, confirmed by direct read). ManageSourcesModal's own independent edit-modal entry point still nulls editInstance in its own close handler. 23 tests across edit-modal-state.test.ts and edit-modal-reset.test.ts pass, including named CR-02 regression cases."
  gaps_remaining: []
  regressions:
    - "One NEW Critical-severity, previously-undetected correctness defect surfaced by this phase's own code-review re-pass (07-REVIEW.md, committed 5bb9eae, dated after 07-07/07-08 landed) in kernel/supervisor/supervisor.go's Apply — the exact function this phase's 07-02 plan wrote as the hot-apply-without-restart mechanism the whole phase goal depends on. Not a regression caused by 07-07 or 07-08 (neither plan touches kernel/supervisor/supervisor.go — confirmed via git diff scope below); it is a pre-existing defect from 07-02 that neither the original verification pass nor the 07-06/07-07/07-08 gap-closure rounds caught. Independently confirmed here by reading kernel/supervisor/supervisor.go directly, not taken on the review's word — see Critical Issues below. Two further Warning-severity findings from the same review pass (a missing async-generation guard in +page.svelte's handleChipEdit, and an incomplete claim in EditSourceModal's defensive reset-on-open effect's own doc comment) are recorded as human-verification items; neither is currently reachable given how every existing caller behaves, so neither blocks this verification on its own."
gaps:
  - truth: "The kernel's hot-apply mechanism (Supervisor.Apply — the single seam every config save/reload goes through per D-06, and the concrete mechanism behind roadmap success criterion 1's 'the kernel loads the result without a restart') never leaves the running kernel in an inconsistent state on a rejected save — s.host, s.coord and s.cfg always agree on which config generation they reflect, whether the save succeeds or is rejected"
    status: failed
    reason: "kernel/supervisor/supervisor.go:240-299 (Apply), confirmed by direct read. Apply's error handling is asymmetric across its two early-return branches in a way that is unsound given what Host.Reconcile actually does. Host.Reconcile (kernel/pluginhost/host.go) mutates s.host.plugins IN PLACE on success — any instance whose connection config changed is relaunched (old subprocess Kill()ed, new *Plugin installed) — and that mutation is already committed by the time Reconcile returns nil; there is no undo. When Reconcile succeeds but the immediately-following pluginhost.ValidateMatchConfig(newCfg, s.host) call then fails (a live, post-launch check that cannot be caught by config.Store.Save's own pre-launch Validate, per ValidateMatchConfig's own doc comment), the code at lines 254-257 restarts the scheduler against oldCfg but leaves s.coord and s.cfg untouched at their PREVIOUS values — while s.host has already moved to newCfg. For any instance whose connection changed in that Apply call, s.coord still holds a correlate.Source pointing at the OLD *Plugin object Reconcile just Kill()ed, while s.host now holds a NEW *Plugin for that same instance that s.coord has no reference to at all. The restarted (oldCfg-generation) scheduler goroutine for that source goes on calling Coordinator.Refresh against the now-dead old *Plugin's gRPC client indefinitely — that source's syncs fail continuously with a transport error, with no signal connecting the failure back to the earlier rejected save — until some LATER Apply call happens to succeed all the way through and rebuild s.coord from s.host.Plugins(). Reachable through ordinary use: a single PUT /api/config that both edits a source's connection details and introduces an invalid match field name for any webspace (a validation config.Store.Save's own struct-level check cannot catch, by design) hits exactly this ordering. Apply's own doc comment (lines 231-239) explicitly promises the opposite of this behavior: 'A failed apply restarts the scheduler against the previously running (unchanged) host and coordinator pairing, so periodic sync does not stall indefinitely.' The host is not unchanged on this path — the doc comment's own invariant is violated by the code beneath it. No test exercises this ordering: kernel/supervisor/supervisor_test.go's TestApply_MidFlightSyncLeavesNoStrandedRunningRow deliberately makes Reconcile itself fail (a missing binary), never reaching the ValidateMatchConfig branch; go test ./kernel/... passes cleanly precisely because nothing currently covers this path. Independently confirmed by reading kernel/supervisor/supervisor.go lines 240-299 directly, not taken on 07-REVIEW.md's word. First raised as 07-REVIEW.md's (post-07-07/08) CR-01 — renumbered here to avoid confusion with the now-closed original CR-01 (agent-route staleness)."
    artifacts:
      - path: "kernel/supervisor/supervisor.go"
        issue: "Apply's ValidateMatchConfig failure branch (lines 254-257) calls s.startScheduler(oldCfg) but never updates s.coord or s.cfg, even though s.host was already mutated to newCfg by the preceding successful Reconcile call — the three fields go out of sync on this one failure path, and stay out of sync until a later Apply succeeds all the way through"
    missing:
      - "Either (a) roll s.coord and s.cfg forward to newCfg in the ValidateMatchConfig failure branch too (since s.host already reflects newCfg regardless of what the caller is told), restarting the scheduler against newCfg so its goroutine set matches what s.coord actually holds; or (b) if leaving the running kernel on the last-known-good match configuration is the intended behavior, re-invoke Host.Reconcile (or add a symmetric rollback method) to relaunch the reconciled instances back against oldCfg.Sources before restarting the scheduler with oldCfg, so s.host, s.coord and the scheduler generation always agree on which config they are running"
      - "A test exercising exactly this ordering (Host.Reconcile succeeds, ValidateMatchConfig fails on the same Apply call) — supervisor_test.go currently has no such case"
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
    why_human: "chip-edit-menu.test.ts is a structural source scan; the stopPropagation and cross-webspace-notice behavior at runtime remain unverified. (The previously-open stale-value-on-reopen sub-case, CR-02, is now VERIFIED — see gaps_closed.)"
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
  - truth: "handleChipEdit's match-mode describePlugin call resolves without ever letting a slower first request's response overwrite a faster second request's state (WR-01, 07-REVIEW.md)"
    test: "make dev; open 'Edit match settings…' on one chip, then before the vocabulary loads, open 'Edit match settings…' or 'Edit connection…' on a different chip; confirm the modal never briefly shows or reverts to the FIRST chip's vocabulary/open state"
    expected: "The second (current) click's state always wins; the first click's late-resolving describePlugin response is a no-op"
    why_human: "Confirmed by direct read (web/src/routes/w/[webspace]/+page.svelte:166-188): handleChipEdit has no generation/sequence guard around its describePlugin await, unlike every other async call site in the same file (load, handleSearch, loadConfig all use navGeneration/searchRequestSeq). This is a real race (07-REVIEW.md WR-01, Warning severity) — the visible effect is a UI display glitch (wrong instance's match-field vocabulary momentarily shown), not a data-corruption path: editInstance itself is always the LAST click's value, so a save still writes to the correct instance; only the offered vocabulary suggestions can be stale. Kept as a human-verification item rather than a gaps blocker because it does not corrupt saved config and requires a live double-click race to observe."
---

# Phase 7: Webspace Builder UI Verification Report

**Phase Goal:** User can configure sources and webspaces from the UI instead of hand-editing TOML — pick plugin types from a list, configure named instances, save a configured set as a webspace, and promote a live search into the webspace's permanent filter.
**Verified:** 2026-08-08
**Status:** gaps_found
**Re-verification:** Yes — after 07-07 (agent-route staleness) and 07-08 (edit-modal stale state) gap-closure plan execution

## Goal Achievement

### Build, Test and Contract Evidence (independently re-run in this session, not taken from SUMMARY claims)

| Check | Command | Result |
|---|---|---|
| Go build | `CGO_ENABLED=0 go build ./...` | clean, exit 0 |
| Go test suite (full) | `go test ./kernel/... -count=1` | all packages `ok` (config, correlate, httpapi, index, pluginhost, supervisor, syncer) |
| Go test — CR-01 closure | `go test ./kernel/httpapi/... -run TestAgentLiveConfig -v` and `-run TestAgentGuard` | 3/3 passed |
| Web test suite (full) | `cd web && npm run test` | 31 files, **492/492** passed |
| Web test — CR-02 closure | `npx vitest run src/lib/edit-modal-state.test.ts src/lib/components/edit-modal-reset.test.ts` | 2 files, 23/23 passed |
| Diff scope since prior verification | `git diff --stat ccc9449 HEAD -- kernel/ web/src` | Exactly `kernel/httpapi/agent.go`, `kernel/httpapi/agent_live_config_test.go`, `web/src/lib/components/EditSourceModal.svelte`, `web/src/lib/components/edit-modal-reset.test.ts`, `web/src/lib/edit-modal-state.ts`, `web/src/lib/edit-modal-state.test.ts`, `web/src/routes/w/[webspace]/+page.svelte` — exactly what 07-07/07-08 claim to touch, no other production file |
| Debt markers on gap-closure files | `grep -riE "TBD|FIXME|XXX"` over the 7 changed files plus `kernel/supervisor/supervisor.go` | none found |
| Requirements traceability | `grep KERN-08\|UI-12 .planning/REQUIREMENTS.md` | Both `[x]`, Phase 7, currently "Gaps Found" (correctly reflects this report's outstanding blocker below) |

### Gap Closure: Prior CR-01 (agent-route config staleness) — CONFIRMED FIXED

The first failed truth from the prior verification pass is now **✓ VERIFIED**, based on direct source reading:

- `kernel/httpapi/agent.go` read in full: `agentSourcesHandler`, `agentWebspacesHandler`, `agentStreamHandler`, `agentItemHandler`, `agentRenditionHandler` all resolve `cfg := cfgStore.Expanded()` as the first statement inside their returned closures. `MountAgentRoutes` (lines 391-399) takes `store *index.Store, cfgStore *config.Store, fetcher Fetcher, prober HealthProber` and forwards `cfgStore` unchanged into all six route registrations — it resolves no config value itself.
- `agentWebspacesHandler`'s `item_count` is computed per request via `agentGrantedItemCount`, which re-reads `store.StreamItems` and filters by the freshly-resolved `granted` set on every call — never cached across requests.
- `agentItemHandler`/`agentRenditionHandler` route both a genuinely-missing item and an ungranted item through the identical `agentItemNotFound` call, preserving the no-existence-leak guarantee.
- `kernel/httpapi/agent_live_config_test.go` (563 new lines): `TestAgentLiveConfig_RevokedReadGrantTakesEffectWithoutRestart` and `TestAgentLiveConfig_NewlyGrantedSourceIsVisibleWithoutRestart` each build one real `httptest`-backed router from a real temp-file `*config.Store`, save a config change through the production `Store.Save` path, and re-issue requests against the SAME already-constructed router — confirming `/agent/v1/sources`, `/agent/v1/items/{id}` (+ `/content`, `/thumbnail`), and `/agent/v1/webspaces`' `item_count` all reflect the change with no restart. `TestAgentGuard_EveryHandlerResolvesConfigPerRequest` is a structural AST guard pinning the invariant against regression. All three independently re-run in this session: **passed**.
- `MountAgentRoutes`'s doc comment (lines 384-390) now accurately describes the per-request resolution and explicitly references the CR-01 fix, replacing the prior stale parity claim.

### Gap Closure: Prior CR-02 (edit-modal stale-state resurfacing) — CONFIRMED FIXED

The second failed truth from the prior verification pass is now **✓ VERIFIED**, based on direct source reading:

- `web/src/routes/w/[webspace]/+page.svelte` read in full: `handleEditClose` (line 190) and `handleEditSaved` (line 194) both call the single `resetEditSession()` (lines 159-163), which sets `editOpen = false; editInstance = null; editMode = 'connection'; editVocabulary = []`. Since `EditSourceModal` is rendered inside `{#if configResponse && editInstance}` (a render *guard*, not merely a `{#key}` value), nulling `editInstance` destroys the component subtree outright on every close path — the fix no longer depends on the `{#key}` expression's value happening to differ across reopens.
- `web/src/lib/edit-modal-state.ts` (68 lines, new): exports `seedConnectionValues`/`seedMatchBlock`, both pure functions that always return a fresh object/array (never the config document's own object), confirmed by direct read.
- `web/src/lib/components/EditSourceModal.svelte` read in full: `connectionValues`/`matchBlock` `$state` are seeded from these two helpers at mount (lines 64-65) AND inside a defensive reset-on-open `$effect` (lines 71-93) that tracks only the `open` flag — every config/instance/webspace read inside is wrapped in `untrack(...)`, confirmed directly, so a parent config refresh landing mid-edit cannot wipe in-progress typing, and reopening cannot resurface a discarded session.
- `ManageSourcesModal.svelte`'s own independent edit-modal entry point (line 366: `onclose={() => (editInstance = null)}`) is confirmed still correct and untouched.
- `web/src/lib/edit-modal-state.test.ts` (140 lines, new, 13 tests) and `web/src/lib/components/edit-modal-reset.test.ts` (186 lines, new, 10 tests) — including named "CR-02 regression" describe blocks proving a re-seed has no memory of a discarded session, and that exactly one `editInstance = null` assignment exists in the route file (inside `resetEditSession`). Independently re-run in this session: **23/23 passed**.

### New Finding: One Critical, Previously-Undetected Defect (07-REVIEW.md re-review, committed `5bb9eae`)

This phase's own code-review process, re-run after 07-07/07-08 landed, found **one new Critical-severity, unresolved issue** in `kernel/supervisor/supervisor.go`'s `Apply` — the exact hot-apply mechanism roadmap success criterion 1 ("the kernel loads the result without a restart") depends on. Independently confirmed here by reading the actual source, not taken on the review document's word:

`Apply` (`kernel/supervisor/supervisor.go:240-299`) stops the scheduler, calls `s.host.Reconcile(ctx, newCfg.Sources, s.logger)` (which mutates `s.host.plugins` **in place** on success — relaunching any instance whose connection config changed, killing its old subprocess), then calls `pluginhost.ValidateMatchConfig(newCfg, s.host)`. If `Reconcile` succeeds but `ValidateMatchConfig` then fails, the code (lines 254-257) restarts the scheduler against `oldCfg` but leaves `s.coord` and `s.cfg` at their **previous** values — while `s.host` has already moved to `newCfg`. For any instance whose connection changed in that `Apply` call, `s.coord` keeps a reference to the OLD `*Plugin` object `Reconcile` just killed, while `s.host` now holds a NEW `*Plugin` for that instance that `s.coord` has no reference to at all. The restarted (old-generation) scheduler goroutine for that source calls `Coordinator.Refresh` against the dead old `*Plugin`'s gRPC client indefinitely — continuous sync failures with no signal connecting them back to the earlier rejected save — until a *later* `Apply` succeeds all the way through.

This is reachable through ordinary use (`config.Store.Save`'s pre-launch validation cannot catch an invalid match field name against a plugin's live vocabulary — that check is deliberately deferred to `ValidateMatchConfig`, by its own doc comment — so one `PUT /api/config` editing a source's connection AND introducing a bad match field name hits exactly this ordering), and it directly contradicts `Apply`'s own doc comment's promise ("a failed apply restarts the scheduler against the previously running (unchanged) host and coordinator pairing" — the host is not unchanged on this path). No test exercises this ordering; `go test ./kernel/...` passes cleanly precisely because nothing currently covers it (`TestApply_MidFlightSyncLeavesNoStrandedRunningRow` deliberately fails `Reconcile` itself, never reaching the `ValidateMatchConfig` branch). `kernel/supervisor/supervisor.go` was last touched by this phase's own `07-02` commits (`f25c4ab`, `894ab20`) and is untouched by 07-07/07-08 (confirmed via `git diff --stat ccc9449 HEAD -- kernel/ web/src`, which does not include this file).

This sits inside the phase's own hot-apply mechanism, is reachable through ordinary UI use with no confirmation step, and directly undermines the same "the kernel loads the result without a restart, and never leaves the running system in a broken state" guarantee both roadmap success criterion 1 and the phase's D-06/D-12/D-13 design decisions are built around — the same family of concern the two now-closed CR-01/CR-02 findings belonged to. Per the same decision-tree rule the prior verification pass applied ("blocker anti-pattern found → gaps_found," regardless of whether an individual enumerated must-have truth's exact wording covers it), this finding drives this report's status.

Two further Warning-severity findings from the same review pass are recorded as human-verification items below (WR-01: `handleChipEdit`'s async `describePlugin` call has no request-generation guard, a display-only race not a data-corruption path; WR-02: `EditSourceModal`'s defensive reset-on-open effect's doc comment overstates its own coverage for match-mode, but is not currently reachable given every existing caller's remount discipline). Neither independently blocks this verification.

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
| **Total** | | **85** | **54** | **31 (28 non-backstop + 5 backstop + 1 WR-01 human item folded into non-backstop human count above)** | **0** |

No individual enumerated must-have truth across any of the 8 plans is worded to cover the new supervisor.go Apply finding — consistent with how the original CR-01/CR-02 were handled in the prior pass, it is tracked as its own gap (see Gaps Summary) because the decision-tree rule "blocker anti-pattern found → gaps_found" applies regardless of exact truth wording.

**Why the truth-level count went up:** 07-07 and 07-08 each added their own must-have truths (7 non-backstop + 1 backstop apiece), closing the two gaps the prior verification round tracked outside the per-plan truth counts. The previously-failed gaps are now folded into the 07-07/07-08 truths above rather than double-counted.

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `kernel/httpapi/agent.go` | Five agent handlers each resolving config per request; `MountAgentRoutes` holding no config value | ✓ VERIFIED | Confirmed by direct read; `gsd_run query verify.artifacts` also reports pass |
| `kernel/httpapi/agent_live_config_test.go` | Live-config regression coverage + AST guard | ✓ VERIFIED | 563 lines, 3 named tests, all pass |
| `web/src/lib/edit-modal-state.ts` | Single seeding site, pure functions | ✓ VERIFIED | 68 lines, exports confirmed |
| `web/src/lib/edit-modal-state.test.ts` | Behavioral unit tests incl. CR-02 regression | ✓ VERIFIED | 140 lines, 13 tests pass |
| `web/src/lib/components/edit-modal-reset.test.ts` | Structural guard over route + component reset | ✓ VERIFIED | 186 lines, 10 tests pass |
| `kernel/supervisor/supervisor.go` | Hot-apply mechanism keeps `s.host`/`s.coord`/`s.cfg` consistent on every Apply outcome | ✗ DEFECTIVE | ValidateMatchConfig failure branch (lines 254-257) leaves `s.coord`/`s.cfg` stale while `s.host` has already moved forward — see Gaps |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `kernel/httpapi/agent.go` | `kernel/config/store.go` | Per-handler `cfgStore.Expanded()` | ✓ WIRED | Confirmed in all 5 handlers |
| `kernel/httpapi/routes.go` | `kernel/httpapi/agent.go` | `MountAgentRoutes(r, store, cfgStore, fetcher, prober)` | ✓ WIRED | `grep` confirms exact call site (tool's regex check false-negatived on the literal parens; manually confirmed) |
| `kernel/httpapi/agent_live_config_test.go` | `kernel/httpapi/live_config_test.go` | Shared `saveConfig` helper | ✓ WIRED | `grep` confirms two call sites (tool's regex check false-negatived on literal parens; manually confirmed) |
| `web/src/routes/w/[webspace]/+page.svelte` | `{#if configResponse && editInstance}` render guard | `resetEditSession` nulls `editInstance` | ✓ WIRED | Confirmed |
| `web/src/lib/components/EditSourceModal.svelte` | `web/src/lib/edit-modal-state.ts` | `$state` init + reset-on-open effect both call the shared helpers | ✓ WIRED | Confirmed |
| `web/src/lib/components/EditSourceModal.svelte` | svelte's `untrack` | Reset-on-open effect wraps config reads | ✓ WIRED | `grep -n "untrack("` confirms (tool's regex check false-negatived on literal parens; manually confirmed) |
| `kernel/supervisor/supervisor.go` `Apply` | `kernel/pluginhost/host.go` `Reconcile` | Mutation committed to `s.host` before `ValidateMatchConfig` runs, with no corresponding update to `s.coord`/`s.cfg` on that branch's failure | ✗ NOT_WIRED (defect) | This is the mechanism of the new gap, not a missing link — the link exists but propagates inconsistent state |

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|---|---|---|---|---|
| KERN-08 | 07-01, 07-02, 07-05, 07-06, 07-07, 07-08 | Webspace/source-instance config editable via kernel API (non-secret only), hand-editing remains supported | ⚠️ BLOCKED | Config write path itself is sound and well-tested (07-01/02/05/06/07/08 all verified) — but the hot-apply mechanism underneath it (Supervisor.Apply) has a confirmed defect that can leave the kernel serving a stale/broken plugin reference for an instance whose connection change was part of a rejected save. The requirement's "editable... while hand-editing remains supported" surface is functionally present; the reliability guarantee behind it is not fully intact. |
| UI-12 | 07-01, 07-03, 07-04, 07-05, 07-06, 07-07, 07-08 | Webspace builder UI: pick plugin types, configure named instances, save the set, promote a live search into a permanent filter | ⚠️ BLOCKED (same root cause) | All UI-facing must-haves for this requirement across 07-03/04/05/06/07/08 are VERIFIED or behavior-unverified-pending-human-check; none FAILED. Blocked at the same kernel-level gap above, since a UI save can trigger the defective Apply path. |

No orphaned requirement IDs found — every ID mapped to Phase 7 in REQUIREMENTS.md (KERN-08, UI-12) appears in at least one plan's `requirements:` frontmatter field, and every plan's declared requirements are accounted for above.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `kernel/supervisor/supervisor.go` | 254-257 | Asymmetric error-handling branch that mutates one piece of shared state (`s.host` via the preceding `Reconcile`) without updating the sibling state (`s.coord`, `s.cfg`) it must stay consistent with | 🛑 Blocker | Leaves the kernel silently serving a stale/dead plugin reference for an affected instance until a later successful Apply; contradicts the function's own doc comment |
| `web/src/routes/w/[webspace]/+page.svelte` | 166-188 | `handleChipEdit`'s async `describePlugin` await has no generation/sequence guard, unlike every other async call site in the same file | ⚠️ Warning | Display-only race (stale vocabulary suggestions momentarily shown); does not corrupt saved config since `editInstance` always reflects the latest click |
| `web/src/lib/components/EditSourceModal.svelte` | 70-94 | Reset-on-open effect's doc comment claims full defensive coverage but does not re-seed `MatchFieldsForm`'s own internal `text` state for match mode | ℹ️ Info | Not currently reachable — every existing caller destroys/remounts the modal on close — but the doc comment overstates the guarantee for a future caller |

No debt markers (`TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`) found in any file touched by 07-07, 07-08, or in `kernel/supervisor/supervisor.go`.

### Human Verification Required

See `behavior_unverified_items` in frontmatter for the full list (15 items: 13 carried forward unchanged from the prior verification round since no plan re-touched that code, plus 1 new backstop each from 07-07 and 07-08, plus WR-01). All require a live `make dev` session, most exercising a UI flow end-to-end against a running kernel — none were available in this verification environment. None of these items are FAILED; they are present-and-wired code paths whose runtime behavior a static/test-suite check cannot fully exercise.

### Gaps Summary

One Critical-severity gap blocks this phase: `kernel/supervisor/supervisor.go`'s `Apply` function — the concrete mechanism behind roadmap success criterion 1's "the kernel loads the result without a restart" — has an asymmetric error-handling branch that can leave the running kernel's `s.coord` (and the scheduler generation restarted alongside it) referencing a plugin instance `s.host` has already killed and replaced, whenever `Host.Reconcile` succeeds but the immediately-following `ValidateMatchConfig` check rejects the save. This is reachable through ordinary use (an edit that changes a source's connection details and, in the same save, introduces an invalid match field name), leaves that source's periodic sync silently and continuously broken until a later successful save, and directly contradicts the function's own documented invariant. It sits inside code this phase wrote (07-02) as the core hot-apply mechanism the entire phase goal depends on, was not caught by the original verification pass, the 07-06 gap closure, or the 07-07/07-08 gap closures — only surfaced by this phase's own follow-up code review. No test currently exercises this failure ordering.

Both previously-open gaps (agent-route config staleness, edit-modal stale-state resurfacing) are now confirmed closed by direct source reading and passing behavioral/structural tests — this is genuine progress, not a wash. The phase is one focused fix away from a clean pass: either roll `s.coord`/`s.cfg` forward to `newCfg` in the `ValidateMatchConfig` failure branch (since `s.host` already reflects it), or re-invoke `Reconcile` to roll the host back to `oldCfg.Sources` before restarting the scheduler — paired with a test exercising the `Reconcile`-succeeds/`ValidateMatchConfig`-fails ordering, which `supervisor_test.go` does not currently cover.

---

_Verified: 2026-08-08T16:56:33Z_
_Verifier: Claude (gsd-verifier)_
