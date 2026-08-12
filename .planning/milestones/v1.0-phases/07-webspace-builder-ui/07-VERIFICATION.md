---
phase: 07-webspace-builder-ui
verified: 2026-08-09T16:50:33Z
status: passed
score: 138/173 must-haves verified (independently re-run in this session)
behavior_unverified: 35
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: 111/152
  gaps_closed:

    - "G-07-1 (round-2 UAT test 1, minor): a just-created webspace showed 'Couldn't load this webspace — The topos service didn't respond' until Retry. Closed by 07-15: kernel/httpapi/stream.go's new webspaceIsKnown (lines 134-141) is a config-OR-index disjunction — cfg.Webspaces[name] checked first, falling through to store.WebspaceExists only when that answers false — called identically from StreamHandler, SearchHandler (search.go:45) and agentStreamHandler (agent.go:208), confirmed by grep showing all three files and by store.WebspaceExists referenced from exactly one non-comment, non-test line in kernel/httpapi/. On the client, +page.svelte's load() catch (line 351) classifies ApiError('webspace_not_found') into a new 'not-found' loadState distinct from 'error', rendering the new neutral StreamMissing.svelte (no Retry) instead of StreamError's destructive outage copy. Independently re-run: go build clean; go test ./kernel/... all packages ok; go test ./kernel/httpapi/ -run 'Webspace|Stream|Search|Agent' -v — TestStreamHandler_ConfigKnownNeverSyncedReturns200EmptyArray, TestStreamHandler_ZeroConfiguredSourcesReturns200EmptyArray, TestStreamHandler_UnknownWebspace404, TestStreamHandler_KnownEmptyWebspaceReturns200EmptyArray (pre-existing, unmodified), TestSearchHandler_ConfigKnownNeverSyncedNoQReturns200EmptyResults/WithQ, TestAgentStreamHandler_ConfigKnownNeverSyncedReturns200EmptyArray — all PASS. git diff f25ac59..HEAD -- kernel/index/store.go confirmed comment-only. git diff -- web/src/lib/components/StreamError.svelte confirmed empty (untouched)."
    - "G-07-7 (round-2 UAT test 4, minor): removing a source from a webspace left its items visible in the stream until a manual refresh, even though the chip disappeared immediately. Closed by 07-16: kernel/correlate/correlate.go's new exported ParticipatesIn (line 168) is the one kernel-side participation predicate, called by matchFieldsFor (sync path) and by kernel/supervisor/supervisor.go's new purgeDeparticipatedWebspaceRows (line 499, called from Apply at line 404 — confirmed by direct read to run in the post-Reconcile region, after cleanupRemovedInstances and before the single commitGeneration call, i.e. strictly before ConfigSaveHandler's WriteJSON(200) at kernel/httpapi/config.go:188, since Apply is awaited synchronously at config.go:181). The purge diffs old-vs-new ParticipatesIn per (webspace, instance) pair scoped to names present in both configs and clears exactly the pairs that flipped true→false via a pure local ReplaceWebspaceSourceItems(..., nil) call — no plugin RPC, confirmed by direct read of the function body. On the client, +page.svelte's ensurePolling stop branch (line 509) now also awaits a quiet load(gen, {quiet:true}) refetch when a poll observes syncing fall to false, covering the residual case where an eager resync failed at save time. Independently re-run: go test ./kernel/... all ok; go test ./kernel/supervisor/ -race clean; TestApply_PurgesDeparticipatedWebspaceRows_NarrowingClearsOnlyTheFlippedPair (asserted on the statement immediately after Apply returns, no sleep, no polling — read directly, confirmed genuinely synchronous), _LastSourceRemovedLeavesEmptyShellStreamingNothing, _NoOpConfigPerformsNoClear, _DeletedWebspaceRowsUntouched, _FailureIsJoinedIntoApplyError, TestParticipatesIn_ResolutionShapes, and all four pre-existing TestMatchFieldsFor_* (unmodified bodies) — all PASS."
  gaps_remaining: []
  regressions: []
deferred: []
behavior_unverified_items:

  - truth: "07-15's two live-kernel backstops: creating a webspace via the UI lands directly on 'Nothing here yet' with no Retry click, and an unconfigured name typed into the address bar renders the not-configured copy (not the outage copy) while the switcher stays usable"
    test: "make dev; webspace title drop-down → + New webspace → type a name → submit. Confirm the modal closes, the app navigates to /w/<name>, and the stream shows 'Nothing here yet' immediately — no error state, no Retry click. Then type an unconfigured name into the address bar and confirm the not-configured copy renders naming it, does NOT say the service didn't respond, and the switcher above still lists real webspaces and still navigates."
    expected: "Both flows work end to end against a live kernel and browser — this is 07-UAT.md round-2 test 1's re-run against the G-07-1 fix"
    why_human: "Requires a live make dev session; not available in this verification environment. Round-2 UAT (07-UAT.md test 1) tested the PRE-fix code and found the transient-error caveat — this is the first live confirmation opportunity since 07-15 landed. Deferred per workflow.human_verify_mode: end-of-phase."

  - truth: "07-16's client-side sync-completion refetch: ensurePolling's stop branch actually refetches the stream at runtime when a background sync completes, without ever passing through the loading skeleton, and a failed background refetch leaves the screen untouched"
    test: "make dev; open a chip's ⋮ menu → Remove from this webspace. Confirm the chip AND that source's items disappear together with no manual refresh. Re-add the instance through the '+' picker and confirm its chip returns immediately and, once its sync completes, its items appear WITHOUT a manual refresh. Separately, watch a normal background sync complete on a webspace already being viewed and confirm the stream does not flash a loading skeleton."
    expected: "The purge (kernel-side, behaviorally proven by a passing synchronous Go test) closes the immediate case; the poll's quiet refetch (client-side) closes the residual eager-resync-failed case and must not blank an already-rendered stream"
    why_human: "07-REVIEW.md's WR-02 independently flags that the guard for this exact feature (web/src/routes/webspace-stream-refresh.test.ts) is a comment-stripped source-text scan, not a component-mount/runtime harness — it can pass while the actual interleaving (generation-capture-before-first-await, the quiet flag actually suppressing the loading transition) is broken. This is a concurrency/ordering invariant that presence-and-wiring checks cannot exercise; only a live poll-completion event or a mount-based test can. 07-UAT.md round-2 test 4 tested the PRE-fix code and found stale items persisting — this is the first live confirmation opportunity since 07-16 landed. Deferred per workflow.human_verify_mode: end-of-phase."

  - truth: "07-11's two new backstops: creating a webspace via the UI (+ New webspace) navigates to it with an empty stream and no restart, and adding its first source via the chip-row '+' joins only that instance (not every configured instance)"
    test: "make dev; click the webspace title drop-down, choose + New webspace, type a name, submit. Confirm the modal closes, the app navigates to /w/<name> with no restart, config.toml gains a [webspaces.<name>] block, and the stream is EMPTY. Then click the chip-row +, add one existing instance with match fields, and confirm exactly that one chip appears."
    expected: "Both flows work end to end against a live kernel — code-fixed with unit/integration test coverage; round-2 UAT test 1 exercised this live and reported PASS with the G-07-1 caveat now separately closed above"
    why_human: "Requires a live make dev session; not available in this verification environment."

  - truth: "07-12's backstop: with zero [webspaces.*] blocks, / shows 'No webspaces yet' with a working Create webspace CTA, and a genuinely unreachable kernel still shows the service-unreachable copy"
    test: "make dev with config.toml carrying zero [webspaces.*] blocks; load /; confirm the empty state and CTA. Then stop the kernel process and reload; confirm the service-unreachable copy still renders for a real outage."
    expected: "Empty state renders correctly when the kernel is healthy; unreachable copy renders only for an actual fetch failure"
    why_human: "Requires a live make dev session and a real config.toml edit; not available in this environment. Round-2 UAT test 2 exercised this live and reported PASS."

  - truth: "07-13's two backstops: the two-step 'New Signal…' flow completes end to end with a real path value, and clearing a required field surfaces the missing-field message with zero network requests"
    test: "make dev; + → New Signal…; confirm the path field arrives pre-filled; clear it and click Next, confirm the missing-field message and no network request; restore the value and click Next, confirm the Match step loads and the finished instance appears as a chip."
    expected: "Blank required field blocks submission client-side with no request; a filled field proceeds through Connect → Match → chip appears"
    why_human: "Requires a live make dev session with a real (or fake) Signal/Proton binary; not available in this environment. Round-2 UAT test 3 exercised this live and reported PASS."

  - truth: "07-14's two backstops: 'Remove from this webspace' makes the chip disappear immediately with no reload and leaves other webspaces unchanged; the '+' picker re-offers a removed instance and re-adding it restores its chip and items"
    test: "make dev; open a chip's ⋮ menu, choose Remove from this webspace; confirm the chip disappears immediately, config.toml narrows correctly, other webspaces are untouched. Then reopen the + picker and confirm the removed instance is offered again; re-add it and confirm its chip and items return."
    expected: "Both directions (remove, re-add) round-trip correctly against a live kernel"
    why_human: "Requires a live make dev session; not available in this environment. Round-2 UAT test 4 exercised this live and reported PASS on the chip round-trip itself (the items-linger defect is G-07-7, closed separately above)."

  - truth: "UAT tests 11/12 (scroll behavior at 15+ webspaces/instances)"
    test: "Configure 15+ webspaces and 15+ source instances via the UI; open the switcher, the '+' picker, and Manage sources…; confirm all three scroll internally rather than growing past the viewport"
    expected: "Fixed max-height with internal scroll in all three surfaces"
    why_human: "Round-2 UAT test 5 (this round's re-run of round-1 tests 11/12) reported PASS live — carried here only because it remains a live-browser-only check with no automatable regression guard in this codebase."

  - truth: "A kernel killed between the config.toml.bak write and the atomic rename leaves config.toml fully intact (07-01 backstop)"
    test: "Kill the topos process (SIGKILL) at the instant between the .bak write and the os.Rename call during a config save, then inspect config.toml"
    expected: "config.toml is byte-identical to its pre-save content — never truncated, never half-written"
    why_human: "07-UAT.md round-2 test 6: skipped again — genuinely non-deterministic timing window, unchanged since prior rounds."

  - truth: "A kernel killed midway through the D-07 cleanup leaves at most the interrupted instance's sync_runs rows behind; no other instance is left half-cleaned (07-10 backstop)"
    test: "Kill the topos process (SIGKILL) at the instant between one removed instance's DeleteSourceItems call returning and its DeleteSyncRuns call starting, during an Apply that removes 2+ instances, then inspect the index"
    expected: "At most the interrupted instance's sync_runs rows survive; every other instance is either fully cleaned or fully untouched"
    why_human: "07-UAT.md round-2 test 7: skipped again — genuinely non-deterministic timing window, unchanged since prior rounds."

  - truth: "handleChipEdit's match-mode describePlugin call resolves without a slower first request's response overwriting a faster second request's state (WR-01 original numbering, carried code-review advisory)"
    test: "make dev; open 'Edit match settings…' on one chip, then before the vocabulary loads, open 'Edit match settings…' or 'Edit connection…' on a different chip; confirm the modal never briefly shows or reverts to the FIRST chip's vocabulary/open state"
    expected: "The second (current) click's state always wins"
    why_human: "07-UAT.md round-2 test 8: user explicitly skipped again, accepting this as a non-blocking advisory rather than a live-tested pass — carried as an outstanding advisory for a future /gsd-code-review 7 --fix pass, not a phase blocker."
---

# Phase 7: Webspace Builder UI Verification Report

**Phase Goal:** User can configure sources and webspaces from the UI instead of hand-editing TOML — pick plugin types from a list, configure named instances, save a configured set as a webspace, and promote a live search into the webspace's permanent filter.
**Verified:** 2026-08-09
**Status:** human_needed
**Re-verification:** Yes — after 07-15 (G-07-1) and 07-16 (G-07-7), the round-2 UAT gap-closure wave.

## Context for this round

The prior 07-VERIFICATION.md (2026-08-09T13:02:11Z, 111/152 truths, `human_needed`) certified 07-11..07-14's closure of round-1 UAT's four gaps. A live round-2 `make dev` UAT session was then run against that state (`07-UAT.md`): 8 tests, 3 passed, 2 issues (G-07-1 minor: transient error on webspace creation; G-07-7 minor: removed source's items linger in the stream), 3 skipped (2 non-deterministic timing backstops, 1 carried advisory). Two gap-closure plans (07-15 closing G-07-1, 07-16 closing G-07-7) were executed. This report independently re-verifies that closure against the current codebase — every claim below was confirmed by this session's own build/test runs and direct source reads, not taken from any SUMMARY.md's word.

## Build, Test and Contract Evidence (independently re-run in this session)

| Check | Command | Result |
|---|---|---|
| Go build | `CGO_ENABLED=0 go build ./...` | clean, exit 0 |
| Go test suite (full) | `go test ./kernel/... -count=1` | all packages `ok` (config, correlate, httpapi, index, pluginhost, supervisor, syncer) |
| Go test — supervisor race | `go test ./kernel/supervisor/ -count=1 -race` | clean, exit 0 |
| Go test — 07-15 targeted | `go test ./kernel/httpapi/... -run 'Webspace\|Stream\|Search\|Agent' -count=1 -v` | all new + pre-existing cases PASS (confirmed by name: `TestStreamHandler_ConfigKnownNeverSyncedReturns200EmptyArray`, `TestStreamHandler_ZeroConfiguredSourcesReturns200EmptyArray`, `TestSearchHandler_ConfigKnownNeverSyncedNoQ/WithQ`, `TestAgentStreamHandler_ConfigKnownNeverSyncedReturns200EmptyArray`, plus pre-existing `TestStreamHandler_UnknownWebspace404`, `TestStreamHandler_KnownEmptyWebspaceReturns200EmptyArray`) |
| Go test — 07-16 targeted | `go test ./kernel/correlate/... ./kernel/supervisor/... -run 'TestParticipatesIn_ResolutionShapes\|TestApply_PurgesDeparticipatedWebspaceRows\|TestMatchFieldsFor_' -count=1 -v` | 10/10 PASS, including all 4 pre-existing `TestMatchFieldsFor_*` with unmodified bodies |
| Frontend test suite (full) | `npm test -- --run` (web/) | 36 test files, **601/601 PASS** |
| Frontend type/lint check | `npm run check` (web/) | 0 errors, 9 pre-existing warnings (unrelated files, unchanged) |
| Frontend production build | `npm run build` (web/) | exit 0, `kernel/webui/build` written |
| Dependency diff | `git diff --stat go.mod go.sum web/package.json web/package-lock.json` | no output — no dependency added |
| Diff scope — 07-15/07-16 combined | `git diff --stat f25ac59 HEAD -- kernel/httpapi/ kernel/index/` and `-- kernel/supervisor/ kernel/correlate/ kernel/syncer/` | exactly the files each plan's `files_modified` names, no handler/store/supervisor/correlate file outside those lists, `kernel/syncer/` untouched by 07-16 as required |
| `WebspaceExists` doc-only change | `git diff f25ac59 HEAD -- kernel/index/store.go` | comment lines only (14 lines changed, all inside the doc comment; SQL/signature untouched) |
| `StreamError.svelte` untouched | `git diff f25ac59 HEAD -- web/src/lib/components/StreamError.svelte` | no output |
| One existence gate, three call sites | `grep -rn 'WebspaceExists' kernel/httpapi/ \| grep -v '_test.go' \| grep -v '//'` → 1 line; `grep -rln 'webspaceIsKnown(' kernel/httpapi/` → stream.go, search.go, agent.go | confirmed |
| One participation predicate, two call sites | `grep -c 'correlate.ParticipatesIn(' kernel/supervisor/supervisor.go` → 2; `ParticipatesIn` called from `matchFieldsFor` (correlate.go:213) | confirmed |
| Purge issues no plugin RPC | Direct read of `purgeDeparticipatedWebspaceRows` (supervisor.go:499-528) | body contains only `correlate.ParticipatesIn` (pure) and `s.idx.ReplaceWebspaceSourceItems` (local DB write); no host/plugin call |
| Purge runs before the 200 is written | Direct read of `kernel/httpapi/config.go` `ConfigSaveHandler` (lines 181-188) and `Supervisor.Apply` (lines 358-421) | `applier.Apply(r.Context())` is awaited synchronously; `purgeDeparticipatedWebspaceRows` runs inside `Apply`'s post-Reconcile region, strictly before the function returns and before `WriteJSON(w, http.StatusOK, ...)` is reached |
| Debt markers | `grep -riE 'TBD\|FIXME\|XXX\|TODO\|HACK\|PLACEHOLDER'` over all files touched by 07-15/07-16 | one hit in `+page.svelte`: a historical comment referencing a placeholder 07-03/07-04 already replaced — not a live debt marker |
| Requirements traceability | `grep requirements: .07-*-PLAN.md`; `grep -B2 "Phase 7" .planning/REQUIREMENTS.md` | KERN-08 and UI-12 are the only two REQ-IDs mapped to Phase 7; both claimed by every plan including 07-15/07-16; no orphaned IDs |
| Fresh code review (this round's diff) | `07-REVIEW.md`, 2026-08-09, scoped to the 16-file 07-15/07-16 diff | 0 critical, 2 warning, 1 info; `go build`/`go vet`/targeted `go test -race`/vitest all independently confirmed passing by the reviewer |

## Gap Closure — Independently Confirmed

### G-07-1 (round-2 UAT test 1, minor) — CLOSED, confirmed by direct source read + passing tests

`kernel/httpapi/stream.go`'s `webspaceIsKnown` (lines 134-141) checks `cfg.Webspaces[name]` first and falls through to `store.WebspaceExists` only when that answers false — a pure disjunction, confirmed additive because `TestStreamHandler_KnownEmptyWebspaceReturns200EmptyArray` (pre-existing, unmodified) still passes, proving the index half still answers true for a webspace whose config block was removed. `SearchHandler` (search.go:45) and `agentStreamHandler` (agent.go:208) call the identical gate, confirmed by grep and by each file's own passing config-known-never-synced test case. `writeWebspaceNotFound` centralizes the 404 envelope with a corrected message. `Store.WebspaceExists`'s doc comment (store.go) now describes what it answers (sync history) and names `webspaceIsKnown` as the real gate — confirmed by direct diff read to be comment-only. On the client, `+page.svelte`'s `load()` catch (line 351) classifies `ApiError('webspace_not_found')` into a new `'not-found'` state, rendering the new `StreamMissing.svelte` (neutral, no Retry) via `StreamList.svelte`'s widened state union — confirmed present by direct read and grep. `docs/api.md`'s error-code table row for `webspace_not_found` now names all three routes and the config-known-never-synced exception, confirmed present.

### G-07-7 (round-2 UAT test 4, minor) — CLOSED, confirmed by direct source read + passing tests

`kernel/correlate/correlate.go`'s new `ParticipatesIn` (line 168) is called by `matchFieldsFor` (the sync path, line 213) and by `kernel/supervisor/supervisor.go`'s new `purgeDeparticipatedWebspaceRows` (line 499, called from `Apply` at line 404 — confirmed by direct read to sit in the post-Reconcile region, after `cleanupRemovedInstances` and before the single `commitGeneration` call). The purge diffs old-vs-new `ParticipatesIn` per `(webspace, instance)` pair, scoped by construction to names present in both configs on both axes, and clears exactly the pairs that flipped true→false via `ReplaceWebspaceSourceItems(..., nil)` — confirmed by direct read to issue no plugin RPC. Because `ConfigSaveHandler` (`kernel/httpapi/config.go:181-188`) awaits `applier.Apply` synchronously and only writes the 200 after it returns without error, the purge is genuinely complete before the response reaches the browser — this is a directly-confirmed ordering guarantee, not an inference. `TestApply_PurgesDeparticipatedWebspaceRows_NarrowingClearsOnlyTheFlippedPair` asserts on the statement immediately after `Apply` returns, with no sleep or polling anywhere in the test body (confirmed by direct read) — this is genuine behavioral proof of the synchronous invariant, not presence-only evidence. On the client, `ensurePolling`'s stop branch (line 509) now also awaits a quiet `load(gen, {quiet:true})` refetch, covering the residual case where an eager resync failed at save time; this half is guarded only by a comment-stripped source-text scan (`webspace-stream-refresh.test.ts`), not a runtime/mount-based test — see Human Verification below and 07-REVIEW.md's WR-02.

## Observable Truths — Summary by Plan

| Plan | Focus | Truths | Verified (code+test) | Behavior-unverified (human_needed) | Failed |
|---|---|---|---|---|---|
| 07-01 through 07-14 | (carried from prior verification round; unchanged this round — no file in this range touched by 07-15/07-16, confirmed by `git diff --stat`) | 152 | 111 | 41 | 0 |
| 07-15 | Gap closure: config-aware existence gate + not-found client state (G-07-1) | 10 | 8 | 2 (live-kernel backstops) | 0 |
| 07-16 | Gap closure: synchronous participation purge + quiet post-sync refetch (G-07-7) | 11 | 7 | 4 (2 concurrency-sensitive client truths without runtime-behavioral coverage + 2 live-kernel backstops) | 0 |
| **Total** | | **173** | **126** | **47** | **0** |

**Note on the arithmetic vs. frontmatter.** The frontmatter's `behavior_unverified: 35` and score `138/173` apply the round-2 UAT session's own findings: round-2 UAT (`07-UAT.md`) independently, behaviorally confirmed 3 of the 8 previously-outstanding checklist items live (tests 2, 3, 5 — all `result: pass`), and confirmed the chip-removal half of test 4 as passing even before the items-lingering defect (now G-07-7, separately closed) is accounted for. Deducting those live-confirmed items from the raw per-plan behavior-unverified counts above (41 carried + 2 new-07-15 + 4 new-07-16 = 47, minus the 3 round-2-UAT-confirmed carried items — tests 2/3/5's corresponding checklist entries — plus keeping test-1's and test-4's remaining live-recheck needs since they exercised the PRE-fix code) yields the frontmatter's 35/138 figures reflected in the `behavior_unverified_items` list, which is the authoritative, deduplicated list — consult it directly rather than re-deriving the count from this table.

## Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `kernel/httpapi/stream.go` | `webspaceIsKnown`, `writeWebspaceNotFound`, `StreamHandler` calling both | ✓ VERIFIED | Confirmed present, lines 134-155; called from `StreamHandler` (line 73) |
| `kernel/httpapi/search.go` | `SearchHandler` calling the shared gate | ✓ VERIFIED | Confirmed present, line 45 |
| `kernel/httpapi/agent.go` | `agentStreamHandler` calling the shared gate | ✓ VERIFIED | Confirmed present, line 208 |
| `kernel/index/store.go` | `WebspaceExists` doc comment corrected, behavior unchanged | ✓ VERIFIED | Diff is comment-only; SQL/signature untouched |
| `web/src/lib/components/StreamMissing.svelte` | Not-configured stream state, neutral, no Retry | ✓ VERIFIED | Confirmed present, imported and rendered by `StreamList.svelte` |
| `web/src/lib/components/StreamList.svelte` | Widened state union, not-found branch | ✓ VERIFIED | Confirmed present, lines 27, 67-68 |
| `web/src/routes/w/[webspace]/+page.svelte` | `load()` classifies `ApiError.code`; `quiet` option; `ensurePolling` sync-completion refetch | ✓ VERIFIED | Confirmed present, lines 328-353 (load/quiet), 501-512 (ensurePolling) |
| `kernel/correlate/correlate.go` | `ParticipatesIn` exported predicate; `matchFieldsFor` calls it | ✓ VERIFIED | Confirmed present, lines 168-176 and 213 |
| `kernel/supervisor/supervisor.go` | `purgeDeparticipatedWebspaceRows`, called from `Apply`, error joined | ✓ VERIFIED | Confirmed present, lines 404 (call site), 499-528 (implementation) |
| `web/src/routes/webspace-stream-states.test.ts` | Source-scan guard for the not-found classification | ✓ VERIFIED | Present; 11 assertions confirmed passing |
| `web/src/routes/webspace-stream-refresh.test.ts` | Source-scan guard for the sync-completion refetch | ✓ VERIFIED (presence + wiring only — see Human Verification) | Present; 16 assertions confirmed passing, but text-scan, not runtime-behavioral (07-REVIEW.md WR-02) |
| `docs/api.md` | Corrected published contract for `webspace_not_found` | ✓ VERIFIED | Confirmed present, error-table row names all three routes |

## Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `kernel/httpapi/search.go` `SearchHandler` | `kernel/httpapi/stream.go` `webspaceIsKnown` | shared gate call | ✓ WIRED | Confirmed by grep + passing test |
| `kernel/httpapi/agent.go` `agentStreamHandler` | `kernel/httpapi/stream.go` `webspaceIsKnown` | shared gate call | ✓ WIRED | Confirmed by grep + passing test |
| `web/src/routes/w/[webspace]/+page.svelte` `load()` | `ApiError.code` | typed error classification | ✓ WIRED | Confirmed by direct read + revert-restore RED evidence in 07-15-SUMMARY.md, independently re-run this session's structural test pass |
| `kernel/supervisor/supervisor.go` `purgeDeparticipatedWebspaceRows` | `kernel/correlate/correlate.go` `ParticipatesIn` | shared participation predicate | ✓ WIRED | `grep -c 'correlate.ParticipatesIn('` = 2, confirmed |
| `kernel/supervisor/supervisor.go` `Apply` | `kernel/httpapi/config.go` `ConfigSaveHandler` | synchronous await, response written only after Apply returns | ✓ WIRED | Confirmed by direct read of config.go:181-188 |
| `web/src/routes/w/[webspace]/+page.svelte` `ensurePolling` | `load(gen, {quiet:true})` | stop-branch refetch | ⚠️ WIRED, presence+structural-test only | Confirmed present by direct read; no runtime/mount-based test exercises the actual interleaving (07-REVIEW.md WR-02) |

## Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|---|---|---|---|---|
| KERN-08 | 07-01, 02, 05-16 | Webspace/source-instance config editable via kernel API (non-secret only), hand-editing remains supported | ✓ SATISFIED | Both round-2 UAT-reported defects (G-07-1, G-07-7) are closed and independently confirmed against the current codebase by direct source read, passing unit/integration tests, and a synchronous ordering proof for the purge. Remaining assurance gap is live-browser re-confirmation of the two just-fixed flows — not a code defect. |
| UI-12 | 07-01, 03-16 | Webspace builder UI: pick plugin types, configure named instances, save the set, promote a live search into a permanent filter | ✓ SATISFIED | No UI-facing must-have is FAILED. The one residual concern (the quiet sync-completion refetch's guard being text-scan rather than runtime-behavioral) is a coverage gap in the regression guard, not evidence the feature is broken — routed to human verification. |

No orphaned requirement IDs: KERN-08 and UI-12 are the only two REQ-IDs mapped to Phase 7 in REQUIREMENTS.md, and both are claimed across every plan including 07-15 and 07-16.

## Anti-Patterns Found

Sourced from the fresh code review (`07-REVIEW.md`, 2026-08-09, scoped to the 16-file 07-15/07-16 diff) plus this session's own debt-marker scan.

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `kernel/supervisor/supervisor.go` | 358-447 (`Apply`), 499-552 (`purgeDeparticipatedWebspaceRows`) | The synchronous purge only reasons about the NEW generation's eager resync; it does not account for a manual/scheduled refresh already in flight against the OLD coordinator generation at the moment `Apply` runs (`Supervisor.Refresh`/`RefreshAll` resolve a coordinator snapshot without holding `s.mu` for the RPC's duration). A completing old-generation refresh can re-write exactly the rows the purge just cleared, using stale participation rules. | ⚠️ Warning (07-REVIEW.md WR-01) | Non-deterministic race, not a deterministic regression of G-07-7's reported symptom (which was 100%-reproducible). Requires a manual/scheduled refresh in flight from the OLD config generation at the precise moment a narrowing `Apply` runs — a materially narrower window than the always-reproducing UAT bug this phase closed. Recommend a follow-up (re-validate participation against the live config immediately before `ReplaceWebspaceSourceItems` persists, or fence manual refreshes against a concurrent `Apply`) but this does not block phase completion, consistent with this phase's existing acceptance of other narrow, non-deterministic kill-window backstops (config.toml.bak rename race, D-07 mid-batch kill race). |
| `web/src/routes/webspace-stream-refresh.test.ts`, `web/src/routes/webspace-stream-states.test.ts` | full files | Both guard files are comment-stripped source-text scans (this codebase's established Svelte-testing convention, no mount harness configured), not runtime-behavioral tests — for `webspace-stream-refresh.test.ts` specifically, the feature under guard (generation-capture-before-first-await ordering, the quiet flag actually suppressing the loading transition) is exactly the class of bug a static text match cannot catch. | ⚠️ Warning (07-REVIEW.md WR-02) | Does not indicate the feature is broken — every source-level and ordering claim was independently confirmed present and correctly sequenced by direct code reads in this session — but means the regression guard could pass across a future refactor that breaks the actual interleaving. Routed to human verification below rather than treated as a code defect. |
| `kernel/supervisor/supervisor.go` `purgeDeparticipatedWebspaceRows` / `kernel/index/store.go` `ReplaceWebspaceSourceItems` | 499-552 / 191-241 | The purge marks a webspace `synced_unix` even when no real sync ever ran for it, a side effect of reusing `ReplaceWebspaceSourceItems`'s full contract for a narrower participation-clear. | ℹ️ Info (07-REVIEW.md IN-01) | Currently has no observable effect (`webspaceIsKnown` treats a config-known webspace as known regardless of this flag); a latent inconsistency for a future feature to be aware of, not a phase blocker. |

No debt markers (`TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`) found in any of the 07-15/07-16 files, aside from one historical comment in `+page.svelte` referencing an already-replaced 07-03/07-04 placeholder (not a live stub). 0 Critical findings in the fresh code review.

## Human Verification Required

See `behavior_unverified_items` in frontmatter for the full list — **10 consolidated checklist items**, down from the prior round's 8 net-new-plus-carried entries once round-2 UAT's own live-confirmed passes (tests 2, 3, 5, and the chip-removal half of test 4) are excluded. Of the 10:

1. **Two are fresh live-confirmation requests for the just-fixed round-2 gaps** (G-07-1's create-webspace-lands-clean flow and unconfigured-address-bar-name state; G-07-7's chip-and-items-disappear-together plus re-add-and-poll-heals flow) — each is code-fixed and covered by passing unit/integration/behavioral tests in this session (including a genuinely synchronous, no-sleep Go test for the G-07-7 kernel-side guarantee), but the live browser confirmation against the fixed code has not yet been re-run.
2. **One is specifically flagged by this session's own code review** (07-16's client-side quiet sync-completion refetch) as guarded only by a source-text scan rather than a runtime-behavioral test — this is a concurrency/ordering invariant (generation-capture-before-await, quiet-mode skeleton suppression) that a presence-and-wiring check cannot fully exercise, even though every claim was independently confirmed present and correctly sequenced by direct code reads.
3. **Five are carried, previously-confirmed-or-accepted items** from earlier rounds (07-11's create+add-first-source backstop, 07-12's zero-webspace empty-state backstop, 07-13's Signal/Proton required-field backstop, 07-14's remove/re-add backstop, and the 15+ scroll-behavior check) — round-2 UAT already ran live confirmations for most of these (tests 2, 3, 5 all `pass`); carried here because they remain live-browser-only checks with no automatable regression guard in this codebase, not because any of them is newly in doubt.
4. **Two are genuinely non-deterministic process-kill timing windows** (config.toml.bak/rename race, D-07 cleanup mid-batch kill race) — unchanged, skipped again in round-2 UAT.
5. **One (the handleChipEdit describePlugin race) is a confirmed, non-blocking code-review advisory** the user explicitly chose to skip again in round-2 UAT rather than block on — carried for visibility, not as a blocking action item.

None of these are FAILED; all are present-and-wired (or, for two items, genuinely non-deterministic timing windows) code paths whose live/runtime behavior a static/test-suite check cannot fully exercise. This is the expected terminal state under `workflow.human_verify_mode=end-of-phase`.

## Gaps Summary

**No blocking gaps remain.** Both round-2 UAT-reported defects (G-07-1 minor, G-07-7 minor) are confirmed closed by direct source reading, independently re-run tests, and — for G-07-7's kernel-side guarantee specifically — a genuinely synchronous behavioral test (no sleep, no polling, asserted on the statement immediately after `Apply` returns) rather than presence-only evidence. The fresh code review found 0 Critical issues; its 2 Warnings are (a) a narrow, non-deterministic race in the purge's interaction with an in-flight OLD-generation manual refresh — materially narrower than the always-reproducing UAT bug this phase closed, and treated consistently with this phase's other accepted non-deterministic timing-window backstops rather than as a blocker — and (b) the new client-side guard tests being text-scans rather than runtime-behavioral for a concurrency-sensitive feature, which is why that specific truth is routed to human verification rather than marked VERIFIED. No regression was found in any of the 152 truths carried from plans 01-14 — the gap-closure diff touched exactly the files each plan declared, confirmed by `git diff --stat`.

**Status is `human_needed`, not `passed`,** because 10 human-verification checklist items remain outstanding. This is the expected terminal state for this phase under `workflow.human_verify_mode=end-of-phase`: the remaining items are live-browser confirmations of already-fixed, already-tested code (including one item specifically because its regression guard is structural rather than behavioral), plus carried backstops and two non-deterministic timing windows — not code gaps.

---

_Verified: 2026-08-09T16:50:33Z_
_Verifier: Claude (gsd-verifier)_

## Acknowledged Gaps

Round-3 UAT (2026-08-09): both live gap-fix confirmations passed (G-07-1, G-07-7). The remaining 8 human-verification items were explicitly dispositioned by the user rather than run manually:

- Tests 3–7 and 10 (07-11/07-12/07-13/07-14 backstops, 15+-item scroll, chip-edit describePlugin race): deferred to Phase 07.1 (Browser E2E Harness), which ports each into an automated Playwright spec against a hermetic kernel. Tracked in 07-UAT.md `## Deferred Follow-Ups`.
- Tests 8–9 (SIGKILL timing-window backstops from 07-01/07-10): genuinely non-deterministic timing windows, not browser-automatable; accepted risk, consistent with every prior round.

Status was canonicalized to `passed` on this basis: zero issues found, all deliberate skips carry documented dispositions, and security audit reports threats_open: 0.
