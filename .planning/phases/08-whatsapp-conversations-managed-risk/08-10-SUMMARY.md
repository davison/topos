---
phase: 08-whatsapp-conversations-managed-risk
plan: 10
subsystem: api
tags: [svelte, go, stream-state, sync-status, participation-scoping, e2e, uat-gap-closure]

# Dependency graph
requires:
  - phase: 07.1-browser-e2e-harness
    provides: the hermetic Playwright harness (kernel fixture, config-builder, route interception discipline) this plan's new spec builds on
  - phase: 01-first-webspace-end-to-end
    provides: the original StreamError/StreamMissing stream-state contract and its Copywriting Contract row, now split by this plan
  - phase: 05-source-instances-and-per-type-matching
    provides: correlate.ParticipatesIn, the participation predicate this plan's kernel-side scoping reuses verbatim
provides:
  - A neutral, source-scoped degraded stream state (StreamSyncDegraded.svelte) distinct from the genuine-outage state
  - A webspace's sync aggregate scoped to its own participating sources at all four call sites (StreamHandler, WebspacesHandler, agentWebspacesHandler, agentStreamHandler)
  - Browser and structural regression armor for both halves
affects: [09, any future phase touching StreamList.svelte's variant branches or the sync aggregate's callers]

# Actuals (#2632)
actuals:
  tokens: 13216
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Per-cause stream state components (StreamEmpty/StreamMissing/StreamError/StreamSyncDegraded), each with exactly one render site and a doc comment naming its distinguishing cause"
    - "filterRunsByGrant/filterRunsByParticipation as composable, orthogonal map filters over LatestSyncRunPerSource, composed (never merged) at each call site"

key-files:
  created:
    - web/src/lib/components/StreamSyncDegraded.svelte
    - web/src/lib/components/stream-degraded.test.ts
    - web/e2e/specs/g-08-3-degraded-source-not-outage.spec.ts
  modified:
    - web/src/lib/components/StreamList.svelte
    - web/src/lib/components/StreamError.svelte
    - kernel/httpapi/sources.go
    - kernel/httpapi/stream.go
    - kernel/httpapi/webspaces.go
    - kernel/httpapi/agent.go
    - kernel/httpapi/stream_test.go
    - kernel/httpapi/sources_test.go
    - kernel/httpapi/agent_test.go
    - docs/api.md
    - .planning/phases/08-whatsapp-conversations-managed-risk/08-UI-SPEC.md
    - .planning/phases/01-first-webspace-end-to-end/01-UI-SPEC.md

key-decisions:
  - "Composed filterRunsByGrant(filterRunsByParticipation(...)) rather than merging the two predicates into one function — grant filtering stays outermost in meaning per the plan's own instruction, and the two concerns (participation, authorization) stay independently testable"
  - "TestStreamHandler_SyncStatusErrorWhenOneOfTwoSourcesFailed rebuilt on newTestRouterWithConfig with a real config — under participation scoping its original empty-config fixture would fail for the wrong reason (no participants at all), not the two-source precedence rule it exists to guard"

patterns-established:
  - "A stream-state component's doc comment must name its one cause and distinguish itself explicitly from every sibling state (StreamEmpty/StreamMissing/StreamError/StreamSyncDegraded) — established by StreamMissing (07-15), extended here for the fourth branch"

requirements-completed: [SRC-03]

coverage:
  - id: D1
    description: "A 200 stream response with a per-source sync failure and zero items renders the neutral degraded state (StreamSyncDegraded), never the service-unreachable copy"
    requirement: "SRC-03"
    verification:
      - kind: e2e
        ref: "web/e2e/specs/g-08-3-degraded-source-not-outage.spec.ts#a per-source sync failure with zero items degrades, and never claims the service is down"
        status: pass
      - kind: unit
        ref: "web/src/lib/components/stream-degraded.test.ts#the sync-failed branch renders StreamSyncDegraded, not StreamError"
        status: pass
    human_judgment: false
  - id: D2
    description: "A genuine stream fetch failure still renders the outage state (StreamError), and the two causes stay distinguishable"
    requirement: "SRC-03"
    verification:
      - kind: e2e
        ref: "web/e2e/specs/g-08-3-degraded-source-not-outage.spec.ts#a genuine fetch failure still renders the outage state, and never the degraded state"
        status: pass
    human_judgment: false
  - id: D3
    description: "A sync failure alongside items renders the stream normally — the degraded state never hijacks a populated view"
    requirement: "SRC-03"
    verification:
      - kind: e2e
        ref: "web/e2e/specs/g-08-3-degraded-source-not-outage.spec.ts#a sync failure alongside items renders the stream, not either error state"
        status: pass
    human_judgment: false
  - id: D4
    description: "A webspace's sync status reflects only its own participating sources at all four aggregate call sites (stream, webspaces list, and both agent mirrors)"
    requirement: "SRC-03"
    verification:
      - kind: unit
        ref: "kernel/httpapi/stream_test.go#TestStreamHandler_NonParticipatingSourceFailureDoesNotEscalate"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/stream_test.go#TestStreamHandler_ParticipatingSourceFailureStillEscalates"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/stream_test.go#TestStreamHandler_IndexOnlyWebspaceReportsZeroValueSyncDespiteOtherFailure"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/sources_test.go#TestFilterRunsByParticipation_FourBranches"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/agent_test.go#TestAgentStreamHandler_SyncStatusComposesGrantAndParticipation"
        status: pass
    human_judgment: false
  - id: D5
    description: "Structural armor pins the sync-failed branch to StreamSyncDegraded and StreamError to exactly one render site, demonstrated red before green"
    verification:
      - kind: unit
        ref: "web/src/lib/components/stream-degraded.test.ts"
        status: pass
    human_judgment: false

duration: ~45min
completed: 2026-08-10
status: complete
---

# Phase 8 Plan 10: Degraded stream state + participation-scoped sync status Summary

**Split a false total-outage message into an honest per-source degradation, and scoped the kernel's sync-status aggregate so an unrelated source's failure can never leak into a webspace it doesn't feed.**

## Performance

- **Duration:** ~45 min (approximate — precise dispatch timestamp not captured; commit range spans 2026-08-10T22:51:44Z–22:59:07Z UTC for the three code commits, preceded by a substantial read/context phase)
- **Tasks:** 3 completed
- **Files modified:** 15 (3 created, 12 modified)

## Accomplishments

- A webspace whose only failing source is one of several now shows "A source couldn't sync" naming the actual recorded error, not "The topos service didn't respond — check that it's running, then retry."
- The service-unreachable copy (`StreamError.svelte`) now has exactly one cause — a genuine stream-fetch failure — and one render site.
- `kernel/httpapi/sources.go`'s new `filterRunsByParticipation` scopes the sync aggregate to a webspace's actual participating sources (via `correlate.ParticipatesIn`, the exact predicate the sync path itself uses) at all four call sites: `StreamHandler`, `WebspacesHandler` (moved inside its per-webspace loop), `agentWebspacesHandler`, and `agentStreamHandler` (composed with the pre-existing grant filter).
- New hermetic e2e spec (`g-08-3-degraded-source-not-outage.spec.ts`, 3 cases) and a new structural guard (`stream-degraded.test.ts`, 6 assertions) armor both halves against regression; both RED demonstrations were performed and are quoted below.
- `08-UI-SPEC.md` gained Amendment 3 (the degraded state's locked copy contract); `01-UI-SPEC.md`'s original Copywriting Contract row was annotated with a dated superseding note rather than rewritten.

## Task Commits

Each task was committed atomically:

1. **Task 1: End-to-end — a failing source degrades its webspace instead of faking an outage** — `1f429ef` (feat, tracer/tdd)
2. **Task 2: A webspace's sync status reflects only the sources that participate in it** — `803e52a` (fix)
3. **Task 3: Structural armor, and the superseded Phase-1 copy row says so** — `d28dad8` (test)

## Files Created/Modified

- `web/src/lib/components/StreamSyncDegraded.svelte` — the new degraded stream state (default-variant Alert, `syncError` rendered verbatim)
- `web/src/lib/components/StreamList.svelte` — sync-failed branch retargeted from `StreamError` to `StreamSyncDegraded`; precedence over both empty branches unchanged
- `web/src/lib/components/StreamError.svelte` — narrowed to one cause; dead `syncError` prop removed
- `web/src/lib/components/stream-degraded.test.ts` — structural guard (6 assertions) over the sync-failed branch
- `web/e2e/specs/g-08-3-degraded-source-not-outage.spec.ts` — 3 hermetic browser cases (degrade / outage / adjacency)
- `kernel/httpapi/sources.go` — `filterRunsByParticipation` (new helper); corrected `SourceRefreshHandler`'s now-stale `ErrUnknownSource` comment (08-09-PLAN.md landed in this same wave)
- `kernel/httpapi/stream.go` — `StreamHandler` aggregates over the participation-filtered map
- `kernel/httpapi/webspaces.go` — `WebspacesHandler`'s aggregate moved inside the per-webspace loop, scoped per entry
- `kernel/httpapi/agent.go` — `agentWebspacesHandler`/`agentStreamHandler` compose participation with the existing grant filter
- `kernel/httpapi/stream_test.go` — 3 new cases (non-participant excluded, participant still escalates, index-only webspace)
- `kernel/httpapi/sources_test.go` — `TestFilterRunsByParticipation_FourBranches`; rebuilt `TestStreamHandler_SyncStatusErrorWhenOneOfTwoSourcesFailed` on a real config
- `kernel/httpapi/agent_test.go` — `TestAgentStreamHandler_SyncStatusComposesGrantAndParticipation`
- `docs/api.md` — corrected the aggregate's documented scope; corrected the agent-mirror table's two `last_sync`/`sync` rows
- `.planning/phases/08-whatsapp-conversations-managed-risk/08-UI-SPEC.md` — Amendment 3
- `.planning/phases/01-first-webspace-end-to-end/01-UI-SPEC.md` — dated superseding note on the Stream-load-error row

## Decisions Made

- Composed `filterRunsByGrant(filterRunsByParticipation(...))` at both agent call sites rather than merging the two predicates into one function — the plan explicitly requires grant filtering to stay outermost in meaning (an ungranted source must remain invisible regardless of participation), and keeping them as two composable functions keeps each independently unit-tested (see `TestFilterRunsByParticipation_FourBranches` and `TestAgentStreamHandler_SyncStatusComposesGrantAndParticipation`).
- Rebuilt `TestStreamHandler_SyncStatusErrorWhenOneOfTwoSourcesFailed` on `newTestRouterWithConfig` with a real config (both instances configured, webspace carrying a `Keywords` fallback so both participate) rather than its original empty-config fixture — the plan's own `key_links` flagged this test would otherwise fail under participation scoping for the wrong reason (no participants at all, not the two-source error-precedence rule it was written to guard). Documented here per the plan's explicit instruction to make this visible rather than buried in a diff.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug, self-corrected during the plan's own TDD flow] First e2e case ordering caused RED to fail at a misleading assertion**
- **Found during:** Task 1's RED demonstration
- **Issue:** The e2e spec's first written assertion ordering (checking `DEGRADED_TITLE` visibility before the outage-sentence absence) caused the pre-fix run to time out waiting for the degraded title rather than failing cleanly on the outage sentence's presence — a race against the initial loading skeleton (Playwright's `toHaveCount(0)` resolved transiently true before the intercepted fetch settled).
- **Fix:** Added `await expect(page.getByRole('alert')).toBeVisible()` as a settle-wait (valid on both sides of the fix, since both `StreamError` and `StreamSyncDegraded` share the `Alert` primitive's `role="alert"`), then reordered the outage-sentence-absence assertion first.
- **Files modified:** web/e2e/specs/g-08-3-degraded-source-not-outage.spec.ts (part of the same in-progress file before Task 1's commit — not a separate deviation commit)
- **Verification:** Re-ran; RED now fails cleanly on `expect(page.getByText(OUTAGE_SENTENCE)).toHaveCount(0)` with `Received: 1`
- **Committed in:** 1f429ef (Task 1 commit — this was resolved before the task's own commit, not a follow-up fix)

---

**Total deviations:** 1 self-corrected during the plan's own TDD flow (not a post-hoc fix to committed code).
**Impact on plan:** None — resolved before Task 1's commit; no scope creep, no follow-up commit needed.

## RED Demonstrations (quoted, per the plan's own acceptance criteria)

### Task 1 — pre-fix e2e RED (case 1, before `StreamSyncDegraded.svelte` existed)

```
Error: expect(locator).toHaveCount(expected) failed

Locator:  getByText('The topos service didn\'t respond — check that it\'s running, then retry.')
Expected: 0
Received: 1
Timeout:  5000ms
```

The accessibility snapshot captured at that failure showed the outage `Alert` rendering both the fixed outage sentence AND the scripted per-source error text in the same alert — proof that, pre-fix, a per-source sync failure was rendering as a total outage.

### Task 3 — pre-fix structural-guard RED (StreamList.svelte's sync-failed branch temporarily reverted to `<StreamError {onretry} />`)

```
FAIL src/lib/components/stream-degraded.test.ts > the sync-failed branch renders
StreamSyncDegraded, not StreamError (08-UAT.md G-08-3) > renders StreamSyncDegraded
AssertionError: expected the sync-failed branch to render StreamSyncDegraded — reverting
this puts the false "The topos service didn't respond" outage copy back in front of a
user whose kernel answered 200 and whose only real problem is one source's failed sync:
expected false to be true

FAIL src/lib/components/stream-degraded.test.ts > the sync-failed branch renders
StreamSyncDegraded, not StreamError (08-UAT.md G-08-3) > passes the response's recorded
sync error through
AssertionError: expected the sync-failed branch to forward response.sync.error...
expected false to be true

FAIL src/lib/components/stream-degraded.test.ts > StreamError renders in exactly one
branch — the fetch-failure state, never the sync-failed one > StreamError appears
exactly once in the whole file
AssertionError: expected exactly one <StreamError render site...: expected 2 to be 1

Test Files  1 failed (1)
     Tests  3 failed | 3 passed (6)
```

After restoring, `git diff --stat -- web/src/lib/components/StreamList.svelte` produced no output — byte-identical to Task 1's committed state.

## Issues Encountered

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- The presentation half of `08-UAT.md` G-08-3 is closed. The co-cause (K) — the suspend/resume coordinator-rebuild lifecycle bug that also contributed to the reported gap — is out of scope for this plan and tracked separately as `08-09-PLAN.md` in the same wave.
- `git diff --stat b969465 HEAD` touches exactly the 15 files declared in this plan's `files_modified` frontmatter — no lockfile, no supervisor file, no plugin file.
- Full verification suite green: `npm --prefix web run test -- --run` (673/673), `npm --prefix web run check` (0 errors, 9 pre-existing warnings), `npm --prefix web run check:e2e` (0 errors), `make e2e` (42/42, +3 over the pre-plan baseline, every pre-existing spec including `uat-04-zero-webspace-vs-outage.spec.ts` unaffected), `make test-portable` (all modules ok).

---
*Phase: 08-whatsapp-conversations-managed-risk*
*Completed: 2026-08-10*
