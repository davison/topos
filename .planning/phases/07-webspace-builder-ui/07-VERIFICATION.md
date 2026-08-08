---
phase: 07-webspace-builder-ui
verified: 2026-08-08T03:00:00Z
status: gaps_found
score: 36/64 must-haves verified
behavior_unverified: 24
overrides_applied: 0
gaps:
  - truth: "The config write path never lets one user action silently corrupt an unrelated, already-configured source instance"
    status: failed
    reason: "07-REVIEW.md CR-01 (Critical, unresolved as of HEAD ce620f3): AddSourceModal.svelte's 'Save anyway' path (web/src/lib/components/AddSourceModal.svelte:242-269, saveAnyway()) omits the duplicate-instance-id collision guard its sibling handleConnectNext (lines ~204-240) enforces. A user can type a display name that derives to an already-existing instance id (e.g. reverting an edit, or naming an existing working instance) and click 'Save anyway' after a failed connection test; upsertSourceInstance(config, candidateId, connectionValues) unconditionally overwrites next.sources[instanceId] — clobbering that instance's base_url/token reference and silently resetting its agent.read/agent.handoff grants to false (the new-instance flow always initializes agent fresh). This is reachable through ordinary UI interaction, requires no confirmation, and succeeds at the PUT /api/config layer (base_hash still matches since nothing else touched the file), so the D-03 clobber guard does not catch it. This directly contradicts the phase's own repeated design principle (D-12/D-13: exactly one deliberate place performs a destructive/overwriting action; T-07-23's mitigation text explicitly assumes 'Save anyway' can only ever be a connection-only write for the NEW instance, which this bug falsifies) and the goal's own framing of the write path as a safe alternative to hand-editing TOML."
    artifacts:
      - path: "web/src/lib/components/AddSourceModal.svelte"
        issue: "saveAnyway() (lines ~242-269) does not check `config.sources[candidateId]` before calling upsertSourceInstance, unlike handleConnectNext's identical check a few lines above it"
    missing:
      - "Reuse handleConnectNext's `if (config.sources[candidateId]) { ...; return; }` collision guard inside saveAnyway() before calling upsertSourceInstance, or factor id-derivation + collision-check into one shared helper both call, per 07-REVIEW.md's own suggested fix"
      - "A regression test in add-source.test.ts or a new focused test proving saveAnyway() refuses to overwrite an existing instance id"
deferred: []
behavior_unverified_items:
  - truth: "Webspace switcher: opening the drop-down lists every configured webspace, marks the current one aria-current at weight 600, and clicking a non-current entry navigates to it in one click (07-03 Task 2)"
    test: "make dev; open a webspace with 2+ webspaces configured; open the title drop-down; confirm every webspace is listed, the current one is visually heavier; click another entry"
    expected: "Menu lists all webspaces in GET /api/config order; current one bold; click navigates to /w/{name} with no full page reload artifact"
    why_human: "web/src/lib/components/webspace-switcher.test.ts is a comment-stripped source-scan (regex over the .svelte file's text), not a rendered-component interaction test — it proves the code SHAPE (aria-current present, one font-semibold occurrence, dropdown-menu child snippet used) but never mounts the component or simulates a click. No make dev session was available in the execution environment (07-03-SUMMARY.md, self-reported)."
  - truth: "Create-webspace modal: submitting a name writes a new [webspaces.<name>] block through PUT /api/config and navigates to it without a kernel restart; a kernel rejection leaves the modal open with the typed name intact (07-03 Task 2)"
    test: "make dev; + New webspace; type a name; submit; confirm config.toml gains the block and the app navigates there with no restart. Then submit a name the kernel rejects and confirm the Alert + retained input."
    expected: "One PUT /api/config call; success navigates; failure keeps modal open with kernel's verbatim message"
    why_human: "CreateWebspaceModal's actual PUT /api/config round trip (success, validation-failure Alert, hash-conflict Alert, disabled-while-saving) was never exercised against a live kernel — 07-03-SUMMARY.md D3 self-flags human_judgment: true for exactly this reason."
  - truth: "Root redirect: with no webspaces configured, `/` renders 'No webspaces yet' with a working Create webspace CTA and does not navigate; with webspaces, it lands on the remembered/first one (07-03 Task 3)"
    test: "make dev with config.toml carrying zero [webspaces.*] blocks; load /; confirm the empty state and its CTA. Then add webspaces back and confirm redirect behavior."
    expected: "No redirect loop, no blank page, CTA opens the same CreateWebspaceModal"
    why_human: "The pure resolveRedirectTarget() function IS genuinely unit-tested (last-webspace.test.ts) and is VERIFIED separately below — but the surrounding +page.svelte component's actual render/navigate behavior (Skeleton while loading, empty-state render, goto with replaceState) is only structurally scanned, never run in a browser."
  - truth: "Add-source '+' picker: opens a popover offering unparticipating instances plus New {plugin type}… rows, or the exact empty-state copy when none remain; choosing an existing instance opens a match-only modal that writes source+match+allowlist and the new chip appears without reload (07-04 Task 1)"
    test: "make dev; click the dashed '+' chip; add an already-configured instance with match fields; confirm the chip appears and its items sync in"
    expected: "Picker lists correctly, one-step modal round-trips through PUT /api/config, chip appears live"
    why_human: "add-source.test.ts is a structural source-scan (07-04-SUMMARY.md D1 self-flags human_judgment: true); no live kernel session exercised the picker→modal→PUT→chip-render chain."
  - truth: "Two-step 'New {plugin type}…' flow: Connect step trial-launches via describePlugin, advances to a vocabulary-driven Match step on success, offers 'Save anyway' + the exact failure copy on a Describe failure, and Step 2 issues exactly one PUT /api/config (07-04 Task 2)"
    test: "make dev; + → New {plugin type}…; complete Connect against a real/fake service; confirm the vocabulary-driven Step 2 form and a single config.toml write carrying all three blocks"
    expected: "Step indicator, describe round trip, single write, chip appears"
    why_human: "07-04-SUMMARY.md D4 self-flags human_judgment: true — the actual Describe round trip against a real plugin subprocess (success and failure paths) was not exercised live. NOTE: this exact code path also carries the unresolved CR-01 defect (see gaps) — human verification of this flow should specifically include the collision case."
  - truth: "Secret field: shows a live Set/Not-set badge for the typed variable name, never displays or transmits a value, and never blocks submit either way (07-04 Task 2)"
    test: "make dev; type an env var name that IS set in the kernel's environment, confirm 'Set'; type one that is NOT, confirm 'Not set — add it to .env and restart before this source can connect.'; confirm the network tab and DOM never contain a secret value"
    expected: "Badge reflects truth; submit stays enabled either way"
    why_human: "secret-field.test.ts proves the component never renders a password input and never receives a value prop — strong structural evidence for the D-15 prohibition — but the badge's actual live truthfulness against a running kernel's env_vars map was not exercised (07-04-SUMMARY.md records the badge reads a synchronous prop snapshot rather than the debounced network lookup the UI-SPEC prose describes, a recorded and reasoned deviation, not yet confirmed live)."
  - truth: "Chip ⋮ menu: offers exactly Edit connection…/Edit match settings…/Remove from this webspace, opening it never toggles the chip's own filter state, and Edit connection… shows the cross-webspace notice before the fields (07-04 Task 3)"
    test: "make dev; click a chip's ⋮ control and confirm the chip's filter state does NOT change; open each menu item and confirm the notice/pre-filled state"
    expected: "stopPropagation prevents filter toggle; notice visible before fields"
    why_human: "chip-edit-menu.test.ts is a structural source scan proving stopPropagation is called before the callback in the SOURCE TEXT, not proving the browser event actually behaves this way at runtime. No live session exercised it."
  - truth: "Manage sources modal: the sole entry point for instance/webspace deletion; deleting an instance shuts down its subprocess and deletes its indexed items across every webspace; deleting a webspace leaves every instance and other webspace untouched; Reload config applies a hand-edit or shows the kernel's verbatim failure with 'The previous configuration is still running.' (07-05 Task 1)"
    test: "make dev; Manage sources…; delete an instance and confirm its chip/items/subprocess are gone everywhere; delete a webspace and confirm nothing else changed; hand-edit config.toml and click Reload config, both for a valid and an invalid edit"
    expected: "Both AlertDialogs behave as documented; Reload's both branches behave as documented"
    why_human: "manage-sources.test.ts is a structural guard (07-05-SUMMARY.md D2 self-flags human_judgment: true). The underlying kernel mechanics (DeleteSourceItems, DeleteSyncRuns, Reload's load-into-locals-then-swap) ARE genuinely integration-tested in 07-02 — but the UI trigger → confirm → observed effect round trip was never run against a live kernel."
  - truth: "A kernel killed between the config.toml.bak write and the atomic rename leaves config.toml fully intact at its previous content (07-01 backstop)"
    test: "Kill the topos process (SIGKILL) at the instant between the .bak write and the os.Rename call during a config save, then inspect config.toml"
    expected: "config.toml is byte-identical to its pre-save content — never truncated, never half-written"
    why_human: "Explicitly tagged verification: backstop in the plan — no test can deterministically interrupt the process at that exact instant. Related: 07-REVIEW.md WR-04 notes WriteCanonical never fsyncs the containing directory after os.Rename, which is a real (low-severity, desktop-app-appropriate) gap in true power-loss durability, though distinct from the mid-write-truncation claim this truth makes."
  - truth: "The webspace switcher drop-down and the add-source picker popover stay usable (height-capped, scrollable) as their list counts reach double digits (07-03/07-04 backstops)"
    test: "Configure 15+ webspaces and 15+ source instances; open the switcher and the '+' picker; confirm both scroll internally rather than growing past the viewport"
    expected: "Fixed max-height with internal scroll"
    why_human: "Tagged verification: backstop — the CSS classes (max-height + overflow-y) are present per the structural guards, but no test renders at scale to confirm the visual result."
  - truth: "The manage-sources instance and webspace lists stay usable as their counts grow (07-05 backstop)"
    test: "Configure 15+ instances and webspaces; open Manage sources…; confirm both lists scroll internally"
    expected: "Fixed max-height with internal scroll"
    why_human: "Tagged verification: backstop — same class as above."
---

# Phase 7: Webspace Builder UI Verification Report

**Phase Goal:** The webspace becomes a builder surface — promote a search to a permanent filter, live config apply/reload without restart, switch/create webspaces from the header, add/edit sources from the chip row, and manage instances/webspaces in one place — all through the UI config write path, never hand-editing TOML.
**Verified:** 2026-08-08
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Build, Test and Contract Evidence (independently re-run, not taken from SUMMARY claims)

| Check | Command | Result |
|---|---|---|
| Go build | `CGO_ENABLED=0 go build ./...` | clean, exit 0 |
| Go test suite | `go test ./kernel/... -count=1` | all packages `ok` (config, correlate, httpapi, index, pluginhost, supervisor, syncer) |
| Web test suite | `cd web && npm run test` | 28 files, **448/448** passed |
| Web typecheck | `cd web && npm run check` | 0 errors (9 pre-existing `state_referenced_locally` warnings, unrelated to correctness) |
| Full production build | `make build` | clean — Go binary + all 4 plugin binaries + SvelteKit static build embedded |
| Plugin contract untouched | `git diff --name-only -- docs/plugin-contract.md proto/` (across all Phase 7 commits) | empty, confirmed |
| Mounted routes | `kernel/httpapi/routes.go` | matches plans exactly: `GET/PUT /api/config`, `POST /api/config/reload`, `GET /api/config/plugin-types`, `POST /api/config/describe-plugin`, plus the two pre-existing refresh routes — no other non-GET route |

All SUMMARY-claimed build/test-green claims were independently reproduced and hold.

### Observable Truths — Summary by Plan

Phase 7 declares 64 must-have truths across its 5 plans (60 numbered + 4 tagged `verification: backstop`). Full detail is in the frontmatter (`behavior_unverified_items`, `gaps`); this table summarizes by plan and truth category rather than listing all 64 rows, given the volume.

| Plan | Focus | Truths | Verified | Behavior-unverified (human_needed) | Failed |
|---|---|---|---|---|---|
| 07-01 | Search→filter tracer, config write path | 17 | 12 | 5 (incl. 1 backstop) | 0 |
| 07-02 | Hot-apply, reload, plugin discovery (kernel-only) | 10 | 10 | 0 | 0 |
| 07-03 | Webspace switcher, create, root redirect | 12 | 4 | 8 (incl. 1 backstop) | 0 |
| 07-04 | Add-source picker, two-step connect, chip edit menu | 15 | 6 | 9 (incl. 1 backstop) | 0* |
| 07-05 | Manage sources, save-state guard, contract publication | 10 | 4 | 6 (incl. 1 backstop) | 0* |
| **Total** | | **64** | **36** | **28 (24 non-backstop + 4 backstop)** | **0** |

\* No individual 07-04/07-05 must-have truth is literally worded to cover the CR-01 defect below, so none is marked FAILED in this table — but CR-01 is tracked as its own gap (see Gaps Summary) because it is a real, reachable, unresolved Critical defect discovered in the same code paths this phase built, and the decision-tree rule "blocker anti-pattern found → gaps_found" applies regardless of whether an enumerated truth's exact wording covers it.

**Why the split is this large:** every backend/kernel truth (config.Store, WriteCanonical, Supervisor.Apply, Reconcile, reload, DiscoverBinaries/DescribePluginType, the AST route guards, and every pure `config-edit.ts`/`plugin-fields.ts`/`last-webspace.ts` function) is backed by a real, passing Go or Vitest unit/integration test that exercises actual logic — several against real subprocess plugin binaries and a real `chi.Router`. These are VERIFIED. Every Svelte **component interaction** truth (switcher click-to-navigate, modal submit round trips, picker open/select, chip menu stopPropagation, delete/reload confirm flows) is backed only by a comment-stripped **source-scan** test (regex over the `.svelte` file's text) that proves the code's shape is present and wired, never by a test that mounts the component and simulates an event. No `make dev` live-kernel session was available in the execution environment for Plans 07-02 through 07-05 (07-01's Task 1 tracer is the one exception — that flow WAS human-verified live, per its own SUMMARY and the orchestrator's context). This split is self-reported honestly in every SUMMARY's `coverage` frontmatter (`human_judgment: true` on the exact same items), which this verification confirms rather than contradicts.

### Required Artifacts

All artifacts named across the 5 plans' `must_haves.artifacts` exist, exceed their `min_lines` thresholds, and are wired (imported and used), confirmed directly:

| Artifact | Plan | Exists | Lines | Wired |
|---|---|---|---|---|
| `kernel/config/store.go` (`Store`, `NewStore`) | 07-01 | ✓ | — | ✓ (used by `cmd/topos/main.go`, `kernel/httpapi/*`) |
| `kernel/config/writer.go` (`WriteCanonical`) | 07-01 | ✓ | — | ✓ |
| `kernel/httpapi/config.go` (`ConfigHandler`) | 07-01/02 | ✓ | — | ✓ (mounted in `routes.go`) |
| `web/src/lib/components/FilterChip.svelte` | 07-01 | ✓ | 52 (≥20) | ✓ |
| `kernel/supervisor/supervisor.go` (`Apply`) | 07-02 | ✓ | — | ✓ |
| `kernel/pluginhost/discover_binaries.go` (`DiscoverBinaries`) | 07-02 | ✓ | — | ✓ |
| `kernel/index/store.go` (`DeleteSourceItems`) | 07-02 | ✓ | — | ✓ |
| `web/src/lib/components/WebspaceSwitcher.svelte` | 07-03 | ✓ | 82 (≥40) | ✓ (replaces `<h1>` in `WebspaceHeader.svelte`) |
| `web/src/lib/components/CreateWebspaceModal.svelte` | 07-03 | ✓ | 107 (≥40) | ✓ |
| `web/src/lib/config-edit.ts` | 07-03/04/05 | ✓ | 232 | ✓ (exports `addWebspace/removeWebspace/setWebspaceFilter/setMatchBlock/addSourceToWebspace/removeSourceFromWebspace/upsertSourceInstance/removeSourceInstance`, all confirmed) |
| `web/src/lib/last-webspace.ts` | 07-03 | ✓ | 58 | ✓ (exports `readLastWebspace/writeLastWebspace/resolveRedirectTarget`) |
| `web/src/lib/plugin-fields.ts` | 07-04 | ✓ | 155 | ✓ |
| `web/src/lib/components/MatchFieldsForm.svelte` | 07-04 | ✓ | (≥30, confirmed present) | ✓ |
| `web/src/lib/components/AddSourceModal.svelte` | 07-04 | ✓ | 500 (≥80) | ✓ |
| `web/src/lib/components/SecretField.svelte` | 07-04 | ✓ | 75 (≥25) | ✓ |
| `web/src/lib/components/ManageSourcesModal.svelte` | 07-05 | ✓ | 370 (≥70) | ✓ (opened only from `WebspaceSwitcher`'s `Manage sources…`, confirmed no longer a no-op) |
| `web/src/lib/components/save-state.test.ts` | 07-05 | ✓ | (≥25, confirmed present, part of the 448 passing) | ✓ |

No artifact is missing or stub-level.

### Key Link Verification

| From | To | Via | Status |
|---|---|---|---|
| `WebspaceHeader.svelte` | `kernel/httpapi/config.go` | `putConfig()` PUT /api/config | WIRED — confirmed in source and exercised by passing HTTP-layer tests |
| `kernel/httpapi/config.go` | `kernel/config/store.go` | `Store.Save` | WIRED |
| `kernel/httpapi/stream.go`/`agent.go` | `kernel/index/store.go` | `StreamItems(ctx, name, filterTerms)` | WIRED, both callers pass identical filter terms (confirmed by grep and passing tests) |
| `kernel/httpapi/config.go` | `kernel/supervisor/supervisor.go` | `Applier.Apply` after `Save`/`Reload` | WIRED |
| `kernel/supervisor/supervisor.go` | `kernel/pluginhost/host.go` | `Host.Reconcile` | WIRED |
| `web/src/lib/components/WebspaceSwitcher.svelte` | `web/src/lib/components/ui/dropdown-menu/` | dropdown-menu primitive | WIRED |
| `web/src/routes/+page.svelte` | `web/src/lib/last-webspace.ts` | `readLastWebspace`/`resolveRedirectTarget` | WIRED |
| `web/src/lib/components/AddSourceModal.svelte` | `kernel/httpapi/config.go` | `describePlugin()` | WIRED (but see CR-01 gap — the sibling `saveAnyway` write path within this same component is where the defect lives) |
| `web/src/lib/components/SourceChip.svelte` | `web/src/lib/components/EditSourceModal.svelte` | `onedit` prop | WIRED, `busy` prop also wired (07-05 fix confirmed present) |
| `WebspaceSwitcher.svelte` `onmanage` | `ManageSourcesModal.svelte` | `handleManageSources` → `manageOpen = true` | WIRED — confirmed no longer the 07-03/07-04 no-op placeholder |

No orphaned or not-wired key links found.

### Requirements Coverage

| Requirement | Description (REQUIREMENTS.md) | Claimed by plans | Status | Evidence |
|---|---|---|---|---|
| KERN-08 | Webspace/source-instance config editable through kernel API (non-secret fields only; secrets stay environment-only), hand-editing remains supported | 07-01, 07-02, 07-05 | SATISFIED, with the CR-01 caveat noted in Gaps | `PUT/GET /api/config`, `POST /api/config/reload`, secret round-trip tests, hot-apply tests all pass; CR-01 is a defect in a UI-layer edit function, not a violation of the requirement's own text (secrets never cross; hand-editing works), but it does undermine the requirement's implicit safety expectation for "editable through the kernel API" |
| UI-12 | Webspace builder UI — pick plugin types, configure named instances, save as webspace, promote live search to permanent filter | 07-01, 07-03, 07-04, 07-05 | SATISFIED, with the CR-01 caveat | Every named capability (switcher, create, add-source picker, two-step connect, chip edit, manage sources, save-as-filter) is implemented, wired, and unit/structurally tested; the search→filter flow was additionally human-verified live |

No orphaned requirement IDs found — REQUIREMENTS.md's traceability table already marks both KERN-08 and UI-12 as "Phase 7 / Complete", and both are declared in at least one plan's frontmatter `requirements:` field, consistent with the roadmap mapping.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `web/src/lib/components/AddSourceModal.svelte` | ~242-269 (`saveAnyway`) | Missing duplicate-instance-id guard present in the sibling code path | 🛑 Blocker | Silent overwrite of an unrelated source instance's connection config and agent grants, reachable via ordinary UI use, unresolved as of HEAD (07-REVIEW.md CR-01) |
| — | — | No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers found in any of the ~34 files this phase created or modified | ℹ️ Info | Debt-marker gate clean |
| `kernel/config/writer.go` | ~56-78 | No directory `fsync` after `os.Rename` | ⚠️ Warning | Real but low-severity durability gap for a true power-loss event (07-REVIEW.md WR-04); does not falsify the backstop truth's literal wording (never truncated/half-written), which concerns process-kill not power-loss |
| `kernel/httpapi/webspaces.go` vs `agent.go` | — | `GET /api/webspaces`'s `item_count` is not narrowed by a webspace's saved `filter`, while `GET /agent/v1/webspaces`'s is | ⚠️ Warning | Contradicts the design comment's own "filtered view IS the webspace for every consumer" framing, but the three routes the phase's own must-have truth explicitly names (stream/search/agent-stream) DO agree identically, per passing tests (07-REVIEW.md WR-01) |
| `web/src/lib/components/CreateWebspaceModal.svelte` | 52-76 | No client-side existing-webspace-name check before submit | ⚠️ Warning | Kernel correctly rejects the resulting invalid config, but with a confusing validator message rather than a clear "name already exists" (07-REVIEW.md WR-02) |
| `web/src/lib/config-edit.ts` | 128-145, 181-189 | A new source instance can retroactively invalidate an unrelated, unviewed webspace | ⚠️ Warning | Kernel's own `Validate` correctly refuses the write (no data loss), but the failure is confusing (07-REVIEW.md WR-03) |
| `web/src/lib/config-edit.ts` | 217-232 | Deleting the last allowlisted instance silently re-opens a webspace to all other sources | ⚠️ Warning | Real, user-visible participation-model flip with no warning in the delete confirmation's copy (07-REVIEW.md WR-05) |

The single 🛑 Blocker (CR-01) is what drives this report's `gaps_found` status per the decision tree ("blocker anti-pattern found"). The five ⚠️ Warnings are pre-existing findings from `07-REVIEW.md`, unresolved but lower severity — none falsifies an enumerated must-have truth's literal wording, and none blocks the phase goal from being observably true. They are carried forward here for visibility rather than re-litigated.

## Gaps Summary

**One blocking gap: 07-REVIEW.md's CR-01 (Critical) is unresolved.** `AddSourceModal.svelte`'s "Save anyway" write path can silently overwrite an existing, unrelated source instance's connection block and reset its agent grants, with no confirmation, through ordinary UI interaction. This was found by this phase's own code review (`07-REVIEW.md`, committed at `ce620f3`), and the commit history confirms no fix landed afterward — `AddSourceModal.svelte`'s last touching commit is `9db2aeb` (07-05 Task 1), which predates the review commit. The fix is small and precisely scoped: reuse `handleConnectNext`'s existing `config.sources[candidateId]` collision check inside `saveAnyway()` before it calls `upsertSourceInstance`.

This is the only truth-blocking gap. Everything else — the full backend write/apply/reload/discovery path, every pure config-editing function, all 448 web tests, the Go test suite, the production build, and the plugin-contract non-interference guarantee — is genuinely verified against the real codebase, not just claimed in a SUMMARY.

The 24 non-backstop items in `behavior_unverified_items` are not gaps in the sense of missing or broken code — they are real, wired, unit/structurally-tested implementations whose actual browser↔kernel runtime behavior has not yet been exercised in this environment (every SUMMARY from 07-02 onward self-reports this limitation honestly). They should be run through a `make dev` human-verification pass — ideally the exact end-of-phase walkthrough `07-05-PLAN.md`'s own `<human-check>` block already specifies — before the phase is fully trusted end to end, and that pass should specifically include the CR-01 collision scenario once fixed.

---

_Verified: 2026-08-08_
_Verifier: Claude (gsd-verifier)_
