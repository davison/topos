---
phase: 07-webspace-builder-ui
plan: 15
subsystem: api
tags: [existence-gate, error-classification, go, sveltekit, gap-closure, webspace-builder]

# Dependency graph
requires:
  - phase: 07-webspace-builder-ui (07-11, 07-12)
    provides: D-20 empty webspace shell (Webspace.IsEmptyShell), config API null-safety, root-route catch isolation pattern this plan's client-side classification mirrors
provides:
  - "webspaceIsKnown (kernel/httpapi/stream.go) — the single config-OR-index existence gate every surface (stream, search, agent stream) calls"
  - "writeWebspaceNotFound (kernel/httpapi/stream.go) — the one 404 envelope writer"
  - "StreamMissing.svelte — the not-configured stream state, neutral treatment, no Retry"
  - "load()'s typed-error classification in web/src/routes/w/[webspace]/+page.svelte"
affects: [07-16 (edits the same stream.go route file; G-07-7's async window is explicitly out of this plan's scope)]

actuals:
  tokens: 8864
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "One shared existence gate (webspaceIsKnown) called from three handler files, replacing three inline copies of the same check — closes the exact drift class (three copies of one decision) that produced the gap"
    - "Config-first, index-second disjunction: the gate is additive by construction, since the index half is untouched and the config half only ever turns a prior 404 into a 200"
    - "Client-side ApiError.code classification narrows an existing catch rather than widening it — the request stays the only thing wrapped in try/catch, mirroring 07-12's isolated-catch lesson"

key-files:
  created:
    - web/src/lib/components/StreamMissing.svelte
    - web/src/routes/webspace-stream-states.test.ts
  modified:
    - kernel/httpapi/stream.go
    - kernel/httpapi/search.go
    - kernel/httpapi/agent.go
    - kernel/httpapi/stream_test.go
    - kernel/httpapi/search_test.go
    - kernel/httpapi/agent_test.go
    - kernel/index/store.go
    - docs/api.md
    - web/src/lib/components/StreamList.svelte
    - web/src/routes/w/[webspace]/+page.svelte
    - .planning/phases/07-webspace-builder-ui/07-UI-SPEC.md

key-decisions:
  - "The gate is config OR index, never config alone — TestStreamHandler_KnownEmptyWebspaceReturns200EmptyArray (pre-existing, seeds the index over an empty config) still depends on and pins the index half."
  - "Existence is answered from configuration, not by writing an index row synchronously from the apply path — the debug session's rejected alternative; the gate moves, sync timing does not need to change."
  - "One gate, three call sites (stream, search, agent stream), fixed in a single task — closes G-07-1.missing[2]'s explicit audit of search.go and agent.go."
  - "The not-found client state is neutral (StreamEmpty's treatment), never StreamError's destructive Alert — nothing is broken when a webspace is not configured."
  - "No Retry button on the not-found state — after the kernel fix, webspace_not_found is definitive, not transient; the webspace switcher in the header is the recovery affordance instead."

patterns-established:
  - "webspaceIsKnown/writeWebspaceNotFound as the one place a cross-cutting existence decision and its error envelope both live, callable from any handler needing the same answer."

requirements-completed: [KERN-08, UI-12]

coverage:
  - id: D1
    description: "A webspace named in the running config with no index rows returns 200 with a non-nil empty items array, on the stream, search and agent-stream routes"
    requirement: KERN-08
    verification:
      - kind: unit
        ref: "kernel/httpapi/stream_test.go#TestStreamHandler_ConfigKnownNeverSyncedReturns200EmptyArray"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/search_test.go#TestSearchHandler_ConfigKnownNeverSyncedNoQReturns200EmptyResults / TestSearchHandler_ConfigKnownNeverSyncedWithQReturns200EmptyResults"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/agent_test.go#TestAgentStreamHandler_ConfigKnownNeverSyncedReturns200EmptyArray"
        status: pass
    human_judgment: false
  - id: D2
    description: "An install with zero configured sources at all still serves a config-known webspace with 200 (the permanent-404 corollary)"
    requirement: KERN-08
    verification:
      - kind: unit
        ref: "kernel/httpapi/stream_test.go#TestStreamHandler_ZeroConfiguredSourcesReturns200EmptyArray"
        status: pass
    human_judgment: false
  - id: D3
    description: "A name in neither config nor index still 404s webspace_not_found; a webspace present only in the index (config block removed) still 200s — the disjunction holds both directions"
    requirement: KERN-08
    verification:
      - kind: unit
        ref: "kernel/httpapi/stream_test.go#TestStreamHandler_UnknownWebspace404, TestStreamHandler_KnownEmptyWebspaceReturns200EmptyArray (pre-existing, unmodified body)"
        status: pass
    human_judgment: false
  - id: D4
    description: "A typed webspace_not_found ApiError renders the distinct StreamMissing state naming the webspace, with no Retry control; every other failure still renders StreamError with Retry; the generation (stale-navigation) check still runs first"
    requirement: UI-12
    verification:
      - kind: unit
        ref: "web/src/routes/webspace-stream-states.test.ts (11 assertions across 5 describe blocks)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Live-kernel verification (make dev): create-webspace flow lands directly on 'Nothing here yet' with no Retry click; an unconfigured name in the address bar shows the not-configured copy naming it, with the switcher still working; a genuine outage still shows the service-unreachable copy and Retry"
    verification: []
    human_judgment: true
    rationale: "Plan's own <human-check> block — requires a live make dev kernel/browser session; not automatable from this execution environment. Deferred to the phase's end-of-phase human-verify pass (workflow.human_verify_mode: end-of-phase)."

duration: ~35min
completed: 2026-08-09
status: complete
---

# Phase 07 Plan 15: Config-Aware Webspace Existence Gate; Not-Found State Distinct From Outage Summary

**A webspace is now servable the instant its `PUT /api/config` returns — before any sync has run, and even on an install with zero configured sources — through one shared existence gate called by the stream, search and agent-stream routes; on the client, a typed `webspace_not_found` answer renders a neutral "not configured" state instead of the fixed service-unreachable copy. Closes 07-UAT.md `G-07-1` on both co-causes.**

## Performance

- **Duration:** ~35 min
- **Completed:** 2026-08-09
- **Tasks:** 2/2
- **Files modified:** 13 (2 new, 11 modified)

## Accomplishments

- `kernel/httpapi/stream.go`'s new `webspaceIsKnown` is the single existence gate: config checked first (a name in `cfg.Webspaces` is servable immediately), falling through to `store.WebspaceExists` only when the config half answers false — a pure disjunction, additive by construction.
- `writeWebspaceNotFound` centralizes the 404 envelope, with its message corrected: after this change a configured webspace is always servable, so a 404 means exactly "not configured," never "might just be unsynced."
- `SearchHandler` and `agentStreamHandler` now call the same gate — closing `G-07-1.missing[2]`'s explicit audit of `search.go` and `agent.go`, so the three surfaces can no longer drift apart.
- `Store.WebspaceExists`'s doc comment corrected to describe what it actually answers (sync history) rather than claiming to be the definition of existence, naming `webspaceIsKnown` as the real gate.
- `docs/api.md`'s stream/search known-unknown paragraphs and the `webspace_not_found` error-code table row (now naming all three routes, including the agent mirror) rewritten to match.
- New `StreamMissing.svelte`: the not-configured stream state, mirroring `StreamEmpty`'s neutral centred treatment, no Retry control — the webspace switcher in the header is the recovery affordance.
- `StreamList.svelte`'s state union widened with `'not-found'`, branch placed with the other state-driven branches ahead of the response-derived variants.
- `+page.svelte`'s `load()` catch now classifies `ApiError('webspace_not_found')` into the new state and everything else into the existing outage state — the generation (stale-navigation) check still runs first, and the catch still wraps only the request.
- `webspace-stream-states.test.ts`: an 11-assertion, comment-stripped source-scan guard pinning the classification, the render branch, and the approved copy with no Retry.

## Task Commits

Each task was committed atomically:

1. **Task 1: A webspace exists because it is configured, not because it has been synced** - `2cfe4bc` (feat, tdd)
2. **Task 2: A definitive answer from a healthy kernel never reads as an outage** - `e8ee516` (feat)

## Files Created/Modified

- `kernel/httpapi/stream.go` - `webspaceIsKnown`, `writeWebspaceNotFound`, `StreamHandler` calling both
- `kernel/httpapi/search.go` - `SearchHandler` calling the shared gate
- `kernel/httpapi/agent.go` - `agentStreamHandler` calling the shared gate; doc comment extended
- `kernel/httpapi/stream_test.go` - 2 new RED-then-GREEN cases (config-known-never-synced, zero-configured-sources)
- `kernel/httpapi/search_test.go` - 2 new mirror cases (no `q`, with `q`)
- `kernel/httpapi/agent_test.go` - 1 new mirror case
- `kernel/index/store.go` - `WebspaceExists` doc comment only (no SQL/signature change)
- `docs/api.md` - known/unknown paragraphs for stream/search, `webspace_not_found` error-table row
- `web/src/lib/components/StreamMissing.svelte` - new: the not-configured state
- `web/src/lib/components/StreamList.svelte` - widened state union, new branch, `webspace` prop
- `web/src/routes/w/[webspace]/+page.svelte` - `load()`'s catch classifies on `ApiError.code`
- `web/src/routes/webspace-stream-states.test.ts` - new source-scan guard (11 assertions)
- `.planning/phases/07-webspace-builder-ui/07-UI-SPEC.md` - Copywriting Contract row for the new state

## RED Confirmations (recorded per plan's `<output>` requirement)

**Task 1 — all five new kernel cases, run against the pre-fix (index-only) gate:**

```
=== RUN   TestAgentStreamHandler_ConfigKnownNeverSyncedReturns200EmptyArray
    agent_test.go:220: expected 200 for a config-known-never-synced webspace through the agent mirror,
    got 404: {"schema_version":1,"error":{"code":"webspace_not_found","message":"webspace \"new-project\"
    is not configured or has not been synced"}}
--- FAIL: TestAgentStreamHandler_ConfigKnownNeverSyncedReturns200EmptyArray (0.00s)
=== RUN   TestSearchHandler_ConfigKnownNeverSyncedNoQReturns200EmptyResults
    search_test.go:81: expected 200 for a config-known-never-synced webspace, got 404: ...
--- FAIL: TestSearchHandler_ConfigKnownNeverSyncedNoQReturns200EmptyResults (0.00s)
=== RUN   TestSearchHandler_ConfigKnownNeverSyncedWithQReturns200EmptyResults
    search_test.go:105: expected 200 for a config-known-never-synced webspace with a query, got 404: ...
--- FAIL: TestSearchHandler_ConfigKnownNeverSyncedWithQReturns200EmptyResults (0.00s)
=== RUN   TestStreamHandler_ConfigKnownNeverSyncedReturns200EmptyArray
    stream_test.go:113: expected 200 for a config-known-never-synced webspace, got 404: ...
--- FAIL: TestStreamHandler_ConfigKnownNeverSyncedReturns200EmptyArray (0.00s)
=== RUN   TestStreamHandler_ZeroConfiguredSourcesReturns200EmptyArray
    stream_test.go:145: expected 200 on a zero-configured-sources install (the permanent-404 corollary),
    got 404: ...
--- FAIL: TestStreamHandler_ZeroConfiguredSourcesReturns200EmptyArray (0.00s)
FAIL
```

All five failed for the identical reason: the gate consulted only sync history. All five passed after `webspaceIsKnown` landed, and the pre-existing `TestStreamHandler_KnownEmptyWebspaceReturns200EmptyArray` and `TestStreamHandler_UnknownWebspace404` passed both before and after, unmodified.

**Task 2 — the client guard's catch-block assertion, run against a temporary, git-diff-confirmed-clean revert of the classification line (`loadState = err instanceof ApiError && err.code === 'webspace_not_found' ? 'not-found' : 'error';` reverted to `loadState = 'error';`):**

```
 FAIL  webspace-stream-states.test.ts > load()'s catch classifies a typed not-found answer apart
       from every other failure > classifies on the caught error's code, producing both the
       not-found and the error state
AssertionError: expected the catch block to branch on the webspace_not_found code: expected false to be true
 Test Files  1 failed (1)
      Tests  1 failed | 10 passed (11)
```

The fix was restored immediately after capturing this output; `git diff --stat web/src/routes/w/[webspace]/+page.svelte` before and after the temporary revert/restore was confirmed identical (18 insertions, 3 deletions unchanged). All 11 assertions passed after restoration.

## Verification Results

- `CGO_ENABLED=0 go build ./...` — exit 0
- `go test ./kernel/... -count=1` — every package `ok` (config, correlate, httpapi, index, pluginhost, supervisor, syncer)
- `go test ./kernel/httpapi/ -run 'Webspace|Stream|Search|Agent' -count=1 -v` — all new and pre-existing cases `PASS`
- `git diff --stat go.mod go.sum web/package.json web/package-lock.json` — no output (no dependency added)
- `git diff --stat kernel/supervisor/ kernel/correlate/ kernel/syncer/` — no output (no sync timing touched)
- `git diff kernel/index/store.go` — comment lines only (11 insertions/3 deletions, all inside the doc comment)
- `git diff web/src/lib/components/StreamError.svelte` — no output (untouched)
- `grep -rn 'WebspaceExists' kernel/httpapi/ | grep -v '_test.go' | grep -v '//'` — exactly one line, inside `webspaceIsKnown`
- `grep -rln 'webspaceIsKnown(' kernel/httpapi/` — lists `stream.go`, `search.go`, `agent.go`
- `grep -c 'StreamMissing' web/src/lib/components/StreamList.svelte` — 3 (import, branch, and the guard test's own regex both count the tag)
- `grep -c "loadState === 'ready'" "web/src/routes/w/[webspace]/+page.svelte"` — 1 (marker-overlay gate unchanged)
- `cd web && npm test` — 35 test files, 585 tests, all pass
- `cd web && npm run check` — 0 errors, 9 pre-existing warnings in unrelated files (unchanged by this plan)
- `cd web && npm run build` — exit 0
- `git diff --stat f25ac59 HEAD` (both task commits combined) — lists exactly the 13 files in the plan's `files_modified`, nothing else
- `grep -riE 'TODO|FIXME|XXX|HACK'` over every new/modified handler and component file in this plan — no matches

## Decisions Made

- Implemented the gate as a disjunction (config OR index), never config-only — the pre-existing `TestStreamHandler_KnownEmptyWebspaceReturns200EmptyArray` (seeded index, empty config) still passes unmodified and still depends on the index half.
- `writeWebspaceNotFound`'s message text changed from "is not configured or has not been synced" to "is not configured" — after this plan a configured webspace is always servable regardless of sync history, so the old message's second clause was no longer ever the true reason for a 404.
- The not-found client state renders no Retry — a `webspace_not_found` response is now definitive, not transient; the webspace switcher (already mounted in every stream state) is the recovery affordance, per planning choice 5.
- Deliberately did NOT change the kernel-wide `sync` aggregate's cross-webspace scoping (planning choice 7) or `handleSearch`'s error classification (planning choice 8) — both are recorded as known, out-of-scope boundaries, not silent gaps.

## Deviations from Plan

None — plan executed exactly as written. Both tasks' `git diff --stat` outputs matched their declared `<files>` lists exactly, and every prohibition held: the gate stayed a disjunction, `WebspaceExists`'s SQL/signature were untouched, no supervisor/correlate/syncer file was touched, `StreamError.svelte` was not modified, and no existing test body in `kernel/httpapi/*_test.go` or `web/src/**/*.test.ts` was altered.

## Issues Encountered

- The client guard's whitespace-sensitive body-copy assertion initially failed against the source-scanned (comment-stripped but not whitespace-normalized) `StreamMissing.svelte`, because the approved copy's line-wraps in the markup put a newline where the assertion's regex expected a single space. Fixed by collapsing whitespace runs before that one assertion, matching how `root-empty-state.test.ts`'s own scanning tolerates markup formatting. Not a Rule 1-4 deviation — the underlying copy and component were correct from the first write; only the test's own string-matching needed adjusting, caught and fixed before any commit.

## Known Stubs

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- 07-UAT.md `G-07-1` is closed on both co-causes: a configured webspace serves 200 before its first sync (including the zero-configured-sources permanent-404 corollary), and a typed not-found answer is no longer reported to the user as a service outage.
- `07-16` (this wave's other gap-closure plan, `G-07-7`) depends on this plan only because both edit `kernel/httpapi/stream.go` — no other overlap. The async eager-resync window `07-16` addresses was deliberately left untouched here (planning choice / prohibition: this plan makes the read boundary correct regardless of when any sync runs).
- Live-kernel human verification (`make dev`: create-webspace flow lands directly on "Nothing here yet"; an unconfigured address-bar name shows the not-configured copy with a working switcher; a genuine outage still shows the service-unreachable copy and Retry) has NOT been run in this execution environment — recorded as coverage item D5, deferred to the phase's end-of-phase human-verify pass per `workflow.human_verify_mode: end-of-phase`.

---
*Phase: 07-webspace-builder-ui*
*Completed: 2026-08-09*

## Self-Check: PASSED

All created/modified files (kernel/httpapi/stream.go, search.go, agent.go and their test files, kernel/index/store.go, docs/api.md, web/src/lib/components/StreamMissing.svelte, StreamList.svelte, web/src/routes/w/[webspace]/+page.svelte, web/src/routes/webspace-stream-states.test.ts, 07-UI-SPEC.md) confirmed present on disk. Both commits (2cfe4bc, e8ee516) confirmed present in `git log`.
