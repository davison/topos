# Debug Knowledge Base

Cross-session patterns worth remembering. Each entry records a defect *class* — not a
one-off bug — so future work can recognise the shape before shipping it again.

Add an entry when a resolved session surfaces a pattern that could plausibly recur in a
different subsystem. Link back to the resolved session for full evidence.

---

## KB-001 — Graceful shutdown cancels the context that also finalises the DB write

**Discovered:** 2026-08-05 · [proton-stuck-syncing](resolved/proton-stuck-syncing.md) · fix `f01df96`
**Class:** two-phase write / lifecycle · **Severity:** silent data loss (no error surfaced to the user)

### The pattern

Code opens a unit of work, records a "started" row, does the work, then writes the outcome —
all under a single `ctx`:

```go
runID, _ := store.StartSyncRun(ctx, sourceType)   // phase 1: "running"
res := doTheWork(ctx)                             // cancelled here on shutdown
store.FinishSyncRun(ctx, runID, res)              // phase 2: NEVER RUNS
```

When that `ctx` is cancelled mid-work — graceful shutdown, an HTTP client disconnecting behind a
manual-trigger endpoint, a parent errgroup failing — `database/sql` rejects the phase-2 UPDATE
outright with `context canceled`. The write is not retried and not queued; it is *skipped*. The
row is stranded in its intermediate state forever.

The insidious part: **the outcome has already happened**. The work either succeeded or failed, and
that fact is real. Cancelling the context that merely *records* it destroys information about
something that already occurred. Cancellation should stop future work, never the recording of past
work.

### Why it hides

- No error reaches the user. The process is shutting down; the log line scrolls past or is lost.
- It only fires on a cancellation that lands *inside* the work window, so it is timing-dependent
  and will not reproduce under normal test runs.
- It preferentially hits the **slowest** component, because that component has the widest window
  to be mid-work at shutdown. In this session only the Proton source was ever affected (IMAP over
  the network to Bridge), which made it look Proton-specific when the defect was in shared code.
  **A bug that appears to affect exactly one plugin may be a shared-path bug filtered by latency.**

### The fix shape

Finalise on a detached context with its own bound:

```go
finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), finishRunTimeout)
defer cancel()
store.FinishSyncRun(finishCtx, runID, res)
```

`context.WithoutCancel` (Go 1.21+) keeps values while dropping cancellation. The explicit timeout
stops a detached write from hanging shutdown indefinitely.

Then add the two companions, because the fix above only prevents *new* damage:

1. **Boot-time reconciliation.** A freshly started process has no in-flight work by definition, so
   anything still in an intermediate state at startup was stranded by a previous process. Mark it
   terminal-with-cause (`error` / "interrupted"), don't delete it — the work was genuinely attempted
   and genuinely did not complete, and that is worth knowing. Never let a repair failure block boot.
2. **Bound the blast radius in the read path.** See KB-002 below — it is what turned this from
   transient to permanent.

### Where to watch for it in this codebase

Any two-phase write, i.e. anywhere a "started" marker precedes a separate "finished" write:

- **Phase 8 WhatsApp plugin event-stream persistence** — a long-lived WebSocket whose events are
  checkpointed as they arrive is exactly this shape, and whatsmeow's connection ctx is cancelled on
  shutdown. Highest-risk upcoming instance.
- Any Phase 4-5 plugin that records sync progress or a resume cursor before/after fetching.
- Anything writing a "last successful position" after a long streaming read (large mailbox backfill,
  chat-history import).

### Checklist for review

- [ ] Does any write record something that has *already happened*? If so, it must not be cancellable.
- [ ] Is there a boot-time path that reconciles intermediate states left by a dead process?
- [ ] Is there a test that cancels the context *mid-work* and asserts on the persisted state?
      (Happy-path lifecycle tests will not catch this — see KB-002's note on gate gaps.)

---

## KB-002 — "Any row" vs "latest row" turns a transient glitch into a permanent one

**Discovered:** 2026-08-05 · [proton-stuck-syncing](resolved/proton-stuck-syncing.md) · fix `f01df96`
**Class:** query semantics · **Severity:** stale state pinned indefinitely

### The pattern

A status query asks "is anything in state X?" when the real question is "is this entity's *current*
record in state X?":

```sql
-- wrong: any row, unbounded in time
SELECT DISTINCT source_type FROM sync_runs WHERE status='running' AND finished_unix IS NULL

-- right: each entity's latest row
WHERE id IN (SELECT MAX(id) FROM sync_runs GROUP BY source_type) AND status='running'
```

With any-row semantics a single stale row outvotes **every** subsequent success — in this case one
orphan from 2026-08-04 beat 244 later successful runs. With latest-row semantics the next completed
run naturally supersedes it and the state self-corrects.

"Is this thing happening *right now*" is a property of the entity's current record, not a property
of the entire history table.

### The tell

Two queries over the same table disagreeing produces contradictory UI that looks impossible:
here, a **green health dot** (from `LatestSyncRunPerSource` → `ok`, correct) rendered simultaneously
with **"Syncing..."** (from `SyncingSourceTypes` → any-running, wrong). When a UI shows two states
that cannot both be true, suspect two different queries backing two different fields — not a
rendering bug. The frontend was never at fault in this session.

### Gate gap this exposed

Happy-path lifecycle tests do not catch either KB-001 or KB-002. The codebase already had
`TestStartAndFinishSyncRun` and even `TestSourcesHandler_RunningRowReportsSyncingTrue`, and both
passed throughout. What was missing:

- no test drove the **interrupted** path (cancel mid-work, then assert persisted state)
- no test set up a **stale row plus a later completed row for the same entity**, which is the only
  arrangement where any-row and latest-row semantics differ

When writing tests for a status/lifecycle table, include an aged-stale-row fixture. It is the case
that distinguishes correct from incorrect aggregation, and it is almost never written by default.
