---
phase: 07-webspace-builder-ui
verified: 2026-08-08T16:10:00Z
status: gaps_found
score: 40/69 must-haves verified
behavior_unverified: 29
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 36/64
  gaps_closed:
    - "The config write path never lets one user action silently corrupt an unrelated, already-configured source instance (07-REVIEW.md CR-01 — original): saveAnyway() now routes through the shared resolveNewInstanceId guard before any write, confirmed by direct source reading (not just SUMMARY claim) plus 8 passing unit tests in instance-id.test.ts and a structural ordering proof in add-source.test.ts"
  gaps_remaining: []
  regressions:
    - "Two NEW Critical-severity findings surfaced by this phase's own re-review (07-REVIEW.md, committed at ccc9449, dated after the CR-01 fix): a fresh agent-route config-staleness bug and a fresh edit-modal stale-state bug. Neither is a regression caused by the 07-06 fix itself (both live in code untouched by 07-06 — kernel/httpapi/agent.go last touched by 07-01, EditSourceModal.svelte/+page.svelte last touched by 07-04/07-05) — they are newly-discovered, pre-existing defects in this phase's own delivered code that the first verification pass did not catch. Both are independently confirmed against the actual source by this verification, not taken on the review's word."
gaps:
  - truth: "Revoking or granting a source's agent.read/agent.handoff grant through the UI's hot-apply config save (PUT /api/config or POST /api/config/reload) takes effect on the /agent/v1 agent-facing API surface without a kernel restart, matching D-06's 'save = apply immediately' guarantee this phase's config write path is built on"
    status: failed
    reason: "kernel/httpapi/agent.go's MountAgentRoutes (confirmed by direct read, lines 391-399) resolves cfg := cfgStore.Expanded() exactly once at server-boot/router-construction time and closes over that single *config.Config value in agentSourcesHandler, agentWebspacesHandler, agentItemHandler and agentRenditionHandler — only agentStreamHandler takes cfgStore itself and re-resolves per request. A grant revoked via the UI's hot-apply save is reflected immediately on every /api/* route and in the UI (07-02 Task 2's live-config fix, confirmed working) but NOT on /agent/v1/sources, /agent/v1/webspaces, /agent/v1/items/{id}, or the content/thumbnail routes until the kernel process restarts — a live authorization-bypass window on the one surface (AGENT-01's default-deny grant model) whose entire job is gating automated access to personal data. The file's own doc comment (agent.go:384-390) falsely claims parity with WebspacesHandler/ItemHandler/SourceRefreshHandler, which were in fact fixed by 07-02 Task 2 — this claim is stale and misleading. Discovered and documented in 07-REVIEW.md CR-01 (new, post-gap-closure re-review, committed ccc9449) and independently re-confirmed here by reading kernel/httpapi/agent.go directly."
    artifacts:
      - path: "kernel/httpapi/agent.go"
        issue: "agentSourcesHandler, agentWebspacesHandler, agentItemHandler, agentRenditionHandler all take a boot-snapshotted *config.Config instead of *config.Store; MountAgentRoutes (lines 391-399) resolves cfg once and never re-resolves it"
    missing:
      - "Thread cfgStore (not a resolved cfg) into all four boot-snapshotted agent handlers, resolving cfg := cfgStore.Expanded() as the first statement inside each returned closure — the same pattern agentStreamHandler and every /api/* handler already use"
      - "A live_config_test.go-style regression test asserting a grant revoked via Store.Save on the same *Store/Router stops appearing in /agent/v1/sources and starts 404'ing on /agent/v1/items/{id} on the very next request, with no restart"
      - "Correct or remove agent.go's stale doc comment (lines 384-390) claiming parity with handlers that are no longer boot-snapshotted"
  - truth: "Editing an existing source's connection or match settings through the chip ⋮ menu (Edit connection…/Edit match settings…) never silently re-saves a value the user already typed and then discarded via Cancel"
    status: failed
    reason: "web/src/routes/w/[webspace]/+page.svelte's handleEditClose and handleEditSaved (confirmed by direct read, lines 160-166) both only set editOpen = false and never clear editInstance/editMode. EditSourceModal is rendered inside {#key `${editInstance}-${editMode}`} (line 582), so reopening the edit modal for the SAME source in the SAME mode produces an identical key and Svelte does not remount the component — its connectionValues/matchBlock $state (seeded once at mount, per EditSourceModal.svelte:59-65) survive from the previous session, including anything typed and then Cancelled. A user can type an incorrect base_url/token/display name, Cancel, later reopen the same source's edit modal, see the stale discarded value indistinguishable from real data, and click Save — silently corrupting config.toml's real connection config for that source via PUT /api/config. This directly parallels the phase's own repeated safety framing for the original CR-01 (the UI write path must never silently corrupt an existing, unrelated or previously-entered configuration) and sits squarely inside UI-12's 'configure named instances' edit flow, built in 07-04 Task 3. The one place this discipline IS correctly implemented — ManageSourcesModal.svelte:366 (onclose={() => (editInstance = null)}) — is not the primary entry point; the chip ⋮ menu's +page.svelte wiring is. Discovered and documented in 07-REVIEW.md CR-02 (new, post-gap-closure re-review, committed ccc9449) and independently re-confirmed here by reading both files directly."
    artifacts:
      - path: "web/src/routes/w/[webspace]/+page.svelte"
        issue: "handleEditClose and handleEditSaved (lines 160-166) never reset editInstance/editMode, so the {#key} guard at line 582 does not force a remount when the same source's edit modal is reopened"
      - path: "web/src/lib/components/EditSourceModal.svelte"
        issue: "connectionValues/matchBlock (lines 59-65) are seeded exactly once at mount from props and rely entirely on the caller forcing a remount to refresh them — no reset-on-open effect exists as a defensive second layer"
    missing:
      - "Reset editInstance (and editMode) to null in handleEditClose and handleEditSaved, mirroring ManageSourcesModal.svelte's own onclose={() => (editInstance = null)}, so every reopen genuinely remounts EditSourceModal from current props"
      - "Optionally, a defensive $effect(() => { if (open) { connectionValues = ...; matchBlock = ...; } }) reset-on-open inside EditSourceModal itself, matching CreateWebspaceModal.svelte's and ManageSourcesModal.svelte's own documented pattern, so a future caller making the same {#key}-reset mistake doesn't reopen this exact gap"
      - "A regression test proving that closing (Cancel) and reopening the same instance/mode pairing yields fresh, current field values, not the previously-typed-and-discarded ones"
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
    why_human: "The Describe round trip against a real plugin subprocess (success and failure paths), and the collision-guard's live browser behavior, were not exercised live. The collision guard itself IS now verified at the source-control-flow level (direct read of saveAnyway's synchronous return-before-write) and via passing unit/structural tests — this item asks for the remaining live-browser/network confirmation only, not the underlying logic."
  - truth: "Secret field: shows a live Set/Not-set badge for the typed variable name, never displays or transmits a value, and never blocks submit either way (07-04 Task 2)"
    test: "make dev; type an env var name that IS set in the kernel's environment, confirm 'Set'; type one that is NOT, confirm 'Not set — add it to .env and restart before this source can connect.'; confirm the network tab and DOM never contain a secret value"
    expected: "Badge reflects truth; submit stays enabled either way"
    why_human: "secret-field.test.ts proves the component never renders a password input and never receives a value prop, but the badge's live truthfulness against a running kernel's env_vars map was not exercised."
  - truth: "Chip ⋮ menu: offers exactly Edit connection…/Edit match settings…/Remove from this webspace, opening it never toggles the chip's own filter state, and Edit connection… shows the cross-webspace notice before the fields (07-04 Task 3)"
    test: "make dev; click a chip's ⋮ control and confirm the chip's filter state does NOT change; open each menu item and confirm the notice/pre-filled state. Also: Cancel an edit with a changed field, reopen the SAME source's Edit connection… and confirm the fields show the CURRENT config, not the discarded typed value (see gaps — this exact scenario is CR-02, currently a confirmed failure, not merely unverified)."
    expected: "stopPropagation prevents filter toggle; notice visible before fields; reopening after Cancel shows fresh values"
    why_human: "chip-edit-menu.test.ts is a structural source scan; the stopPropagation and cross-webspace-notice behavior at runtime remain unverified. NOTE: the stale-value-on-reopen sub-case is not merely unverified — it is a confirmed failure (CR-02, see gaps) and must be fixed before this item is re-tested."
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
    why_human: "Tagged verification: backstop in 07-06-PLAN.md — the guard's synchronous control flow and pure-function behavior ARE genuinely verified (source-level read plus passing unit tests); only the live-browser/network confirmation remains."
---

# Phase 7: Webspace Builder UI Verification Report

**Phase Goal:** User can configure sources and webspaces from the UI instead of hand-editing TOML — pick plugin types from a list, configure named instances, save a configured set as a webspace, and promote a live search into the webspace's permanent filter
**Verified:** 2026-08-08
**Status:** gaps_found
**Re-verification:** Yes — after 07-06 gap-closure plan execution (closes 07-REVIEW.md's original CR-01)

## Goal Achievement

### Build, Test and Contract Evidence (independently re-run in this session, not taken from SUMMARY claims)

| Check | Command | Result |
|---|---|---|
| Go build | `CGO_ENABLED=0 go build ./...` | clean, exit 0 |
| Go test suite | `go test ./kernel/... -count=1` | all packages `ok` (config, correlate, httpapi, index, pluginhost, supervisor, syncer) |
| Web test suite | `npx vitest run src/lib/instance-id.test.ts src/lib/components/add-source.test.ts` | 2 files, 35/35 passed |
| Web full test suite | `cd web && npm run test` | 29 files, **469/469** passed |
| Web typecheck | `cd web && npm run check` | 0 errors (same 9 pre-existing `state_referenced_locally` warnings as prior verification, unrelated to correctness) |
| Plugin contract untouched | `git diff --name-only ce620f3 HEAD -- docs/plugin-contract.md proto/` | empty, confirmed |
| Scope of code changed since prior verification | `git diff ce620f3 HEAD --stat` | only `web/src/lib/instance-id.ts` (new), `web/src/lib/instance-id.test.ts` (new), `web/src/lib/components/AddSourceModal.svelte`, `web/src/lib/components/add-source.test.ts`, plus planning docs — no other production file touched by the gap-closure plan |
| Requirements traceability | `grep KERN-08\|UI-12 .planning/REQUIREMENTS.md` | Both marked `[x]` and "Phase 7 / Complete" |
| Debt markers on gap-closure files | `grep -riE "TBD\|FIXME\|XXX\|TODO\|HACK\|PLACEHOLDER"` over `instance-id.ts`, `instance-id.test.ts`, `AddSourceModal.svelte`, `add-source.test.ts` | none found |

### Gap Closure: Original CR-01 (instance-id collision guard) — CONFIRMED FIXED

The single failed truth from the prior verification pass ("The config write path never lets one user action silently corrupt an unrelated, already-configured source instance") is now **✓ VERIFIED**, based on direct source reading (not the SUMMARY's claim):

- `web/src/lib/instance-id.ts` exists (68 lines), exports `deriveInstanceId`, `resolveNewInstanceId`, `InstanceIdResult`. Read in full — `resolveNewInstanceId` derives a candidate id, rejects blank (`ok:false, reason:'blank'`) and collision (`ok:false, reason:'collision'`) cases before ever returning `ok:true`, and never mutates the passed `cfg`.
- `AddSourceModal.svelte` read in full: `saveAnyway()` (lines 228-255) now calls `resolveNewInstanceId(config, displayName)` as its first action; on a not-ok result it sets `connectError` and returns — a plain synchronous `return` statement that unconditionally prevents `upsertSourceInstance`/`putConfig` from executing later in the same function body. This is a language-level guarantee, not a heuristic: there is no code path in `saveAnyway` between the `return` and the `upsertSourceInstance` call. `handleConnectNext` (lines 191-222) was refactored to use the same shared helper, with the deliberate asymmetry that it DOES clear `describeFailed` on rejection (a `Next`-time validation failure is not a connection failure) while `saveAnyway` does NOT (preserving the retry affordance) — confirmed directly in both function bodies.
- `web/src/lib/instance-id.test.ts` (125 lines): 14 passing tests, including a named CR-01 regression case asserting that resolving the victim instance's own stored display name never returns `ok`, and a never-mutates-input assertion. Independently re-run in this session: **passed**.
- `web/src/lib/components/add-source.test.ts`: extended with a structural ordering proof (`resolveNewInstanceId(` index strictly before `upsertSourceInstance(` index, with a `return` between) plus an invariant suite (zero local `deriveInstanceId` occurrences, exactly two `upsertSourceInstance(` call sites, `saveAnyway` never clears `describeFailed`, `handleConnectNext` does). Independently re-run in this session: **passed**.
- Commits `54ba33c` (fix) and `2582b47` (test) exist in `git log`, match their stated diffs, and match the SUMMARY's claims.

This gap is closed with high confidence — stronger than typical "structural test" evidence, because the fix is a synchronous, single-threaded control-flow guarantee directly confirmed by reading the actual function bodies, not inferred from a regex scan.

### New Findings: Two Critical, Previously-Undetected Defects (07-REVIEW.md re-review, committed `ccc9449`)

This phase's own code-review process re-reviewed all 74 files after the CR-01 gap closure and found **two new Critical-severity, unresolved issues** neither the original verification pass nor the 07-06 gap-closure plan touched. Both are independently confirmed here by reading the actual source, not taken on the review document's word:

**1. Agent-route config staleness (new CR-01).** `kernel/httpapi/agent.go:391-399` (`MountAgentRoutes`) resolves `cfg := cfgStore.Expanded()` exactly once at router-construction time and passes that single snapshot into `agentSourcesHandler`, `agentWebspacesHandler`, `agentItemHandler`, and `agentRenditionHandler` — confirmed directly by reading the function and each handler's signature (`func agentSourcesHandler(store *index.Store, cfg *config.Config, prober HealthProber)`, etc. — all four take `*config.Config`, not `*config.Store`). Only `agentStreamHandler` takes `cfgStore` and re-resolves per request. A grant revoked through this phase's own hot-apply config write path takes effect on `/api/*` and the UI immediately, but not on four of five `/agent/v1` routes until a kernel restart — contradicting D-06's "save = apply immediately" design decision this phase's write path is built on, and creating a live authorization-bypass window against AGENT-01's default-deny grant model. `kernel/httpapi/agent.go` was last touched by this phase's own 07-01 commit (`08e17aa`).

**2. Edit-source modal stale-state resurfacing (CR-02).** `web/src/routes/w/[webspace]/+page.svelte`'s `handleEditClose`/`handleEditSaved` (lines 160-166, read directly) only set `editOpen = false` and never reset `editInstance`/`editMode`. `EditSourceModal` is gated by `{#key \`${editInstance}-${editMode}\`}` (line 582) — reopening the edit modal for the *same* source in the *same* mode produces an identical key, so Svelte does not remount the component, and its `connectionValues`/`matchBlock` `$state` (seeded once at mount, `EditSourceModal.svelte:59-65`, also read directly) survive from the previous session including anything typed and then Cancelled. A user can type an incorrect value, Cancel, later reopen the same source's edit modal, see the stale discarded value indistinguishably from real data, and save it — silently corrupting `config.toml`'s real connection or match config for that instance. Confirmed by contrast: `ManageSourcesModal.svelte:366` correctly implements `onclose={() => (editInstance = null)}` — the discipline exists in the codebase, just not on the primary chip ⋮ menu entry point (`+page.svelte`, last touched by 07-04/07-05).

Both findings sit inside files this phase created or modified, both are reachable through ordinary UI interaction with no confirmation step, and both directly undermine the same "the UI write path must never silently corrupt existing configuration" principle the phase's own design decisions (D-12/D-13) and the original CR-01 fix were built around. Per the same decision-tree rule the prior verification applied ("blocker anti-pattern found → gaps_found," regardless of whether an individual enumerated must-have truth's exact wording covers it), these findings drive this report's status.

### Observable Truths — Summary by Plan (updated)

| Plan | Focus | Truths | Verified | Behavior-unverified (human_needed) | Failed |
|---|---|---|---|---|---|
| 07-01 | Search→filter tracer, config write path | 17 | 12 | 5 (incl. 1 backstop) | 0 |
| 07-02 | Hot-apply, reload, plugin discovery (kernel-only) | 10 | 10 | 0 | 0 |
| 07-03 | Webspace switcher, create, root redirect | 12 | 4 | 8 (incl. 1 backstop) | 0 |
| 07-04 | Add-source picker, two-step connect, chip edit menu | 15 | 6 | 9 (incl. 1 backstop) | 0* |
| 07-05 | Manage sources, save-state guard, contract publication | 10 | 4 | 6 (incl. 1 backstop) | 0* |
| 07-06 | Gap closure: instance-id collision guard | 5 | 4 | 1 (backstop) | 0 |
| **Total** | | **69** | **40** | **29 (24 non-backstop + 5 backstop)** | **0** |

\* No individual enumerated must-have truth is worded to cover the two NEW Critical findings below (agent-route staleness, edit-modal stale state) — consistent with how the original CR-01 was handled in the prior pass, they are tracked as their own gaps (see Gaps Summary) because the decision-tree rule "blocker anti-pattern found → gaps_found" applies regardless of exact truth wording.

**Why the truth-level count went up:** 07-06 added 5 new must-have truths (4 non-backstop, now all VERIFIED by direct source reading; 1 backstop, still requiring a live `make dev` session). The previously-failed roadmap-level truth is now closed and folded into the 07-06 truths above rather than double-counted.

### Required Artifacts

All artifacts named across the 6 plans' (07-01..07-06) `must_haves.artifacts` exist, meet or exceed their `min_lines` thresholds, and are wired. No artifact is missing or stub-level. New/changed since the prior pass:

| Artifact | Plan | Exists | Lines | Wired |
|---|---|---|---|---|
| `web/src/lib/instance-id.ts` | 07-06 | ✓ | 68 (≥25) | ✓ (imported by `AddSourceModal.svelte`) |
| `web/src/lib/instance-id.test.ts` | 07-06 | ✓ | 125 (≥30) | ✓ (14 tests, all passing) |
| `web/src/lib/components/AddSourceModal.svelte` | 07-04/07-06 | ✓ | 487 (≥80) | ✓ — local `deriveInstanceId` removed, `resolveNewInstanceId` imported and called from both write paths |

All other artifacts from the prior verification pass are unchanged (`git diff ce620f3 HEAD --stat` confirms no other production file was touched) and are carried forward as still VERIFIED without re-scanning line counts individually.

### Key Link Verification

| From | To | Via | Status |
|---|---|---|---|
| `web/src/lib/components/AddSourceModal.svelte` | `web/src/lib/instance-id.ts` | `resolveNewInstanceId` called from both `handleConnectNext` and `saveAnyway` before any write | WIRED — confirmed by direct read of both function bodies |
| `web/src/lib/instance-id.ts` | `web/src/lib/config-edit.ts` | a not-ok result means `upsertSourceInstance` is never reached | WIRED — confirmed: both call sites `return` on a not-ok result before reaching `upsertSourceInstance` |
| All prior-verified key links (config-edit → API, switcher → dropdown, root redirect → last-webspace, etc.) | — | — | Carried forward, unchanged (no touching file modified since prior pass) |

New anti-pattern-driven NOT-WIRED findings (see Anti-Patterns below): `kernel/httpapi/agent.go`'s four boot-snapshotted handlers are wired to a *stale* config snapshot, not the live `cfgStore`; `+page.svelte`'s `handleEditClose`/`handleEditSaved` are wired to close the dialog but NOT wired to reset `editInstance`.

### Requirements Coverage

| Requirement | Description (REQUIREMENTS.md) | Claimed by plans | Status | Evidence |
|---|---|---|---|---|
| KERN-08 | Webspace/source-instance config editable through kernel API (non-secret fields only; secrets stay environment-only), hand-editing remains supported | 07-01, 07-02, 07-05, 07-06 | SATISFIED, with the new CR-02 caveat | `PUT/GET /api/config`, `POST /api/config/reload`, secret round-trip tests, hot-apply tests, and the now-closed instance-id collision guard all pass; CR-02 (edit-modal stale state) is a defect in a UI-layer edit function within this requirement's own "editable through the kernel API" surface — it does not violate the requirement's literal text (secrets never cross; hand-editing works) but does undermine its implicit safety expectation, same framing the original CR-01 received |
| UI-12 | Webspace builder UI — pick plugin types, configure named instances, save as webspace, promote live search to permanent filter | 07-01, 07-03, 07-04, 07-05, 07-06 | SATISFIED, with the new CR-02 caveat | Every named capability (switcher, create, add-source picker, two-step connect with the now-fixed collision guard, chip edit, manage sources, save-as-filter) is implemented and wired; CR-02 is a defect specifically in the "configure named instances" edit flow this requirement names |

No orphaned requirement IDs found — REQUIREMENTS.md's traceability table marks both KERN-08 and UI-12 as "Phase 7 / Complete", consistent with the roadmap mapping. (AGENT-01, the requirement most directly affected by the new agent-route staleness finding, belongs to Phase 2 and is already tracked there as "Gaps Found" — it is not one of Phase 7's declared requirement IDs, so the staleness finding is reported here as a blocker anti-pattern in files this phase touched, not as a Phase-7-requirement failure.)

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `kernel/httpapi/agent.go` | 391-399 (`MountAgentRoutes`), 92/146/253/303 (handler signatures) | Four of five agent handlers close over a boot-snapshotted `*config.Config` instead of the live `*config.Store` | 🛑 Blocker | Agent-grant revocation via this phase's own hot-apply config write path has no effect on `/agent/v1/sources`, `/agent/v1/webspaces`, `/agent/v1/items/{id}`, or content/thumbnail routes until kernel restart — new finding, confirmed directly, unresolved as of HEAD `ccc9449` |
| `web/src/routes/w/[webspace]/+page.svelte` | 160-166 (`handleEditClose`/`handleEditSaved`) | `editInstance`/`editMode` never reset on close, so `{#key}` at line 582 does not force a remount when the same source's edit modal is reopened | 🛑 Blocker | Canceling an edit and reopening the same source's edit modal shows previously-typed-and-discarded values, which can then be silently saved over the real config — new finding, confirmed directly, unresolved as of HEAD `ccc9449` |
| — | — | No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers found in `instance-id.ts`, `instance-id.test.ts`, `AddSourceModal.svelte`, or `add-source.test.ts` | ℹ️ Info | Debt-marker gate clean for the gap-closure plan's files |
| `kernel/config/writer.go` | ~56-78 | No directory `fsync` after `os.Rename` | ⚠️ Warning | Carried forward unchanged from prior pass (07-REVIEW.md WR-04 in the original review); file not touched since |
| `web/src/lib/components/ConnectionForm.svelte` | ~46-51 | `unwrapVar` silently echoes a non-`${VAR}`-shaped stored token verbatim into a plaintext field | ⚠️ Warning | New finding this pass (07-REVIEW.md WR-02, current numbering) — a hand-edited config with a literal (non-reference) token value is rendered and could be silently mangled into a broken reference on save |
| `web/src/lib/components/ManageSourcesModal.svelte` | 174-192 (`handleReload`) | No check for the currently-viewed webspace disappearing after a config reload | ⚠️ Warning | New finding this pass (07-REVIEW.md WR-01, current numbering) — `confirmDeleteWebspace` handles this case, `handleReload` does not |
| Various (`CreateWebspaceModal.svelte`, `config-edit.ts`) | — | No client-side webspace-name collision check; a new source instance can retroactively invalidate an unrelated webspace; deleting the last allowlisted instance silently re-opens participation | ⚠️ Warning (x3) | Carried forward unchanged from prior pass (original WR-02/WR-03/WR-05); files not touched since, kernel correctly refuses the resulting invalid writes in all cases (no data loss), only UX clarity affected |
| `kernel/httpapi/agent.go` | 384-390 | Doc comment claims a parity with `/api/*` handlers that no longer holds | ℹ️ Info | 07-REVIEW.md IN-01 — will mislead future maintainers; fix alongside the staleness blocker above |

Two 🛑 Blockers drive this report's `gaps_found` status. The Warnings/Info are carried forward or newly surfaced at lower severity — none falsifies an enumerated must-have truth's literal wording, and none blocks the search→filter or basic create/configure flows from being observably true.

## Gaps Summary

**Two blocking gaps, both newly discovered by this phase's own re-review after the original CR-01 gap closure, neither addressed by any executed plan:**

1. **Agent-route config staleness.** Four of five `/agent/v1` handlers read a config snapshot resolved once at server boot, so an agent-grant revocation made through this phase's UI write path has no effect on the agent-facing API until a kernel restart — a live authorization-bypass window on the AGENT-01 default-deny grant model. This directly contradicts D-06's "save = apply immediately" guarantee, which this whole phase's config write path is framed around.

2. **Edit-source modal stale-state resurfacing.** The chip ⋮ menu's "Edit connection…"/"Edit match settings…" flow never clears `editInstance`/`editMode` on close, so canceling and reopening the same source's edit modal resurfaces previously-typed-and-discarded field values, which a user can then silently save over the real configuration — a direct parallel to the original CR-01's "silent overwrite" class of bug, this time in the edit (not add) path.

Both are confirmed by direct source reading in this verification session (not taken on the code-review document's word), sit in files this phase itself created or modified, are reachable through ordinary UI interaction with no confirmation step, and are unresolved as of `HEAD` (`ccc9449`). Fixes for both are small and well-scoped per 07-REVIEW.md's own suggested fixes (thread `cfgStore` into the four agent handlers; reset `editInstance`/`editMode` on close, mirroring `ManageSourcesModal.svelte`'s own existing correct pattern).

**What is genuinely closed:** the original CR-01 (instance-id collision guard in `saveAnyway`) is fixed with strong evidence — a synchronous control-flow guarantee confirmed by direct code reading, backed by 14 new passing unit tests and a structural invariant suite, with commits matching the SUMMARY's claims. The full backend write/apply/reload/discovery path, every pure config-editing function, all 469 web tests, the Go test suite, and the plugin-contract non-interference guarantee remain genuinely verified.

The 24 non-backstop and 5 backstop items in `behavior_unverified_items` are unchanged in kind from the prior pass — real, wired, unit/structurally-tested implementations whose actual browser↔kernel runtime behavior has not yet been exercised in this environment. They should be run through a `make dev` human-verification pass once the two new blockers above are fixed — the chip ⋮ menu item in this list should specifically include the CR-02 reopen-after-cancel scenario once patched, and the two-step flow item should specifically include the (now code-verified, still not live-browser-verified) Save-anyway collision case.

---

_Verified: 2026-08-08_
_Verifier: Claude (gsd-verifier)_
