---
phase: 07-webspace-builder-ui
reviewed: 2026-08-09T16:42:35Z
depth: standard
files_reviewed: 16
files_reviewed_list:
  - docs/api.md
  - kernel/correlate/correlate.go
  - kernel/correlate/correlate_test.go
  - kernel/httpapi/agent.go
  - kernel/httpapi/agent_test.go
  - kernel/httpapi/search.go
  - kernel/httpapi/search_test.go
  - kernel/httpapi/stream.go
  - kernel/httpapi/stream_test.go
  - kernel/index/store.go
  - kernel/supervisor/supervisor.go
  - kernel/supervisor/supervisor_test.go
  - web/src/lib/components/StreamList.svelte
  - web/src/lib/components/StreamMissing.svelte
  - "web/src/routes/w/[webspace]/+page.svelte"
  - web/src/routes/webspace-stream-refresh.test.ts
  - web/src/routes/webspace-stream-states.test.ts
findings:
  critical: 0
  warning: 2
  info: 1
  total: 3
status: issues_found
---

# Phase 07: Code Review Report (gap-closure re-review: 07-15/07-16)

**Reviewed:** 2026-08-09T16:42:35Z
**Depth:** standard
**Files Reviewed:** 16
**Status:** issues_found

## Summary

This re-review scopes to the 16 files that make up the diff of gap-closure
plans 07-15 (`webspaceIsKnown`, the config-first existence gate for
stream/search/agent-stream) and 07-16 (`purgeDeparticipatedWebspaceRows`,
the synchronous webspace_items purge in `Supervisor.Apply`, plus the
client's quiet background-refresh-on-sync-completion), against commit
range `f25ac59..HEAD`. It supersedes the prior `07-REVIEW.md` (26-file
scope, 07-11 through 07-14), which is no longer applicable to the current
tree; that review's WR-01/WR-02/WR-03/IN-01 (frontend null-tolerance gaps)
and WR-04 (`Host.Reconcile` relaunching on an agent-grant-only change) are
outside this diff's scope and were not re-verified here.

The `webspaceIsKnown` change (07-15) is small, correctly scoped, and
consistently applied across `StreamHandler`, `SearchHandler` and
`agentStreamHandler` — the same predicate, the same 404 message, the same
docs update, backed by tests that exercise every combination named in the
plan (config-known/never-synced, zero-configured-sources, index-known/
config-removed, unknown webspace). No defect found in this half.

The `purgeDeparticipatedWebspaceRows` change (07-16) is well-reasoned and
extensively tested (five new `Supervisor` tests covering narrowing, empty-
shell, no-op idempotency, deleted-webspace exclusion, and multi-failure
error joining), and the extraction of `correlate.ParticipatesIn` as a
shared predicate is verified for agreement with `matchFieldsFor` by a
dedicated table test (`TestParticipatesIn_ResolutionShapes`). However, the
synchronous purge only reasons about the *new* coordinator generation's
eager resync (deliberately dispatched asynchronously, per T-07-62) — it
does not account for a manual/scheduled refresh already in flight against
the *old* coordinator generation at the moment `Apply` runs. That refresh
can complete after the purge and silently re-write the rows the purge just
cleared, using stale participation rules. This is a race condition, not a
deterministic failure, but it directly undermines the invariant this phase
exists to establish (G-07-7: "removed source items linger in stream"). See
WR-01.

The two new front-end structural-guard test files
(`webspace-stream-refresh.test.ts`, `webspace-stream-states.test.ts`)
follow this codebase's established convention of comment-stripped
source-text assertions rather than mounting the component — consistent
with sibling tests elsewhere in the tree — but worth flagging because the
feature they guard (quiet background refresh, generation-capture ordering)
is exactly the kind of logic a text-shape check cannot actually exercise.
See WR-02.

`go build ./...`, `go vet ./kernel/...`, `go test ./kernel/correlate/...
./kernel/httpapi/... ./kernel/index/... ./kernel/supervisor/...` (including
`-race`), and the two new vitest files all pass cleanly against the
reviewed tree. No hardcoded secrets, injection vectors, unsafe eval/exec
usage, or authorization gaps were found in the reviewed delta. The
`/agent/v1` existence-gate change (`agentStreamHandler` now calling
`webspaceIsKnown` instead of `store.WebspaceExists` directly) preserves the
no-existence-leak property verified by this file's own tests.

## Warnings

### WR-01: Synchronous webspace-participation purge can be silently undone by a concurrent in-flight refresh from the previous config generation

**File:** `kernel/supervisor/supervisor.go:358-447` (`Apply`), `kernel/supervisor/supervisor.go:499-552` (`purgeDeparticipatedWebspaceRows`)

**Issue:** `Apply` holds `s.mu` for its own duration and calls
`s.stopScheduler()` before Reconcile, which blocks until the *scheduled*
generation's goroutines have exited (`stopScheduler`'s own doc: "nothing
from the OLD generation can still be calling into the plugin host or
coordinator"). That guarantee does not extend to a **manual** refresh
triggered through `POST /api/sources/{name}/refresh` or `POST /api/sync`
(`Supervisor.Refresh`/`RefreshAll`, lines 188-197): those methods resolve
`s.Coordinator()` fresh (grabbing `s.mu` only long enough to read the
pointer, then releasing it) and then call `.Refresh(ctx, sourceName)` on
that snapshot **without holding any lock for the duration of the RPC**.
`Coordinator.Refresh` → `syncOne` → `correlate.Engine.SyncSource` uses the
`*config.Config` the `Coordinator` was built with at construction time
(`newCoordinator`, line ~100) — a config snapshot that does not update
mid-flight.

Concrete race:
1. A source's underlying plugin process is *not* relaunched by `Reconcile`
   when only webspace/match config changes (`Apply`'s own eager-resync
   comment, lines 429-444, confirms this: "A match-only edit ... has no
   relaunch of its own"). The subprocess a manual refresh is talking to is
   therefore the *same* process across the config change.
2. Operator/UI calls `POST /api/sources/{name}/refresh` (e.g. via
   `handleRefreshSource`/`handleRefreshAll` in `+page.svelte`) for an
   instance participating in webspace W. This resolves the *old*
   `Coordinator` (built from `oldCfg`) and begins a real Match RPC.
3. Before that RPC returns, the operator saves a config that narrows W's
   `sources` allowlist to exclude this instance. `Apply` runs, purges
   `(W, instance)`'s `webspace_items` rows synchronously, and returns.
4. The manual refresh from step 2 (still running against `oldCfg`, where
   the instance *did* participate in W) completes and calls
   `ReplaceWebspaceSourceItems(ctx, W, instance, items)` with a non-nil
   `items` slice, re-inserting exactly the rows the purge in step 3 just
   removed.

The result is that a webspace narrowed to exclude an instance can still
show that instance's stale items after the config save returns — the
precise symptom (`07-UAT.md` G-07-7 /
`.planning/debug/removed-source-items-linger-in-stream.md`) this phase's
synchronous purge and the client's quiet-refresh-on-sync-completion were
built to close, reintroduced through a timing window the extensive doc
comments in `Apply` and `purgeDeparticipatedWebspaceRows` do not address
(they reason carefully about the *new* generation's eager resync racing
the scheduler, but not about an *old*-generation manual refresh outliving
the purge).

**Fix:** Have `correlate.Engine.SyncSource` (or its caller) re-validate
participation against the **current** live config immediately before
persisting each webspace's items — e.g. thread the live `*config.Store`
into `Coordinator`/`Engine` instead of a config snapshot captured at
construction, or have `ReplaceWebspaceSourceItems`'s caller re-check
`correlate.ParticipatesIn(currentCfg.Webspaces[ws], instance)` right before
the write and skip the persist (treating it like the "does not
participate" branch) when the pair is no longer live. Alternatively, make
manual/scheduled refreshes take `s.mu` for the full duration of resolving
and executing against a given coordinator generation, so a refresh in
flight blocks (or is fenced from) a concurrent `Apply`.

### WR-02: New structural "guard" tests assert on source text, not runtime behavior, for a concurrency-sensitive feature

**File:** `web/src/routes/webspace-stream-refresh.test.ts:1-269`, `web/src/routes/webspace-stream-states.test.ts:1-207`

**Issue:** Both new test files parse `+page.svelte` (and, in the states
test, `StreamList.svelte`/`StreamMissing.svelte`) as raw text, strip
comments, and assert via regex/substring/brace-matching that certain
tokens appear in a certain relative order (e.g. "`load(gen, { quiet: true
})` appears after `clearInterval(pollHandle)` inside `ensurePolling`'s
body", or "`if (!quiet) loadState = 'loading'`" appears literally). This
is consistent with this codebase's established pattern for Svelte
components (no component-mount harness is configured in this project's
vitest config), but it means these tests can pass while the actual runtime
behavior is wrong — e.g. a stray early return, an inverted boolean, or a
variable-shadowing bug inside the matched region would not be caught,
because the assertions never execute the code. This matters more than
usual here because the feature under guard (`ensurePolling`'s quiet stream
refetch on sync completion, and the `gen`-capture-before-first-`await`
ordering) is precisely a concurrency/staleness concern — the class of bug
a static text match is least equipped to catch (e.g. `gen` being captured
at the textually-correct spot does not verify that `load()` actually
discards a superseded response correctly under real interleaving).

**Fix:** No change required to match this codebase's existing convention,
but consider adding at least one real interaction test (e.g. via
`@testing-library/svelte` or an equivalent mount-based harness) for
`ensurePolling`'s stop-branch refetch and the `webspace_not_found`
classification, so a future refactor that preserves the token shapes but
breaks the actual control flow is still caught.

## Info

### IN-01: Purge marks a webspace as "synced" even when no real sync ever ran for it

**File:** `kernel/supervisor/supervisor.go:499-552` (`purgeDeparticipatedWebspaceRows`), `kernel/index/store.go:191-241` (`ReplaceWebspaceSourceItems`)

**Issue:** `purgeDeparticipatedWebspaceRows` clears a departicipated pair
via `s.idx.ReplaceWebspaceSourceItems(ctx, wsName, instance, nil)`. Per
that method's own doc comment, this call unconditionally marks
`webspaces.synced_unix` for `wsName` "even if this is the only source
configured" — a side effect designed for the case where a source
genuinely attempted (and completed) a sync. When the purge fires for a
webspace that has never had *any* source successfully sync into it yet
(e.g., the operator narrows a brand-new webspace's allowlist before its
first scheduled/eager sync completes), this call marks it "synced" purely
as a byproduct of a participation-narrowing purge that performed no real
sync work. This currently has no observable effect (`webspaceIsKnown`
already treats a config-known webspace as known regardless of
`WebspaceExists`, and `GET /api/webspaces`'s item counts come from a map
keyed by `cfg.Webspaces` names with a zero-value default, not gated on
`WebspaceExists`), but it is a latent inconsistency in what
`Store.WebspaceExists`/`webspaces.synced_unix` is documented to mean
("has completed at least one sync") that a future feature relying on that
column more strictly could trip over.

**Fix:** If this distinction ever needs to matter, consider a dedicated
`Store` method for a pure participation-clear that does not touch
`webspaces.synced_unix` (e.g. a `DELETE FROM webspace_items WHERE
webspace_name = ? AND item_id IN (SELECT id FROM items WHERE source = ?)`
without the paired `webspaces` upsert), rather than reusing
`ReplaceWebspaceSourceItems`'s full contract for this narrower purge.

---

_Reviewed: 2026-08-09T16:42:35Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
