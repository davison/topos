---
status: resolved
trigger: "Proton Mail plugin permanently shows as 'Syncing...' with a spinner, although the messages appear to load fine"
created: 2026-08-05T15:30:00Z
updated: 2026-08-05T17:05:00Z
resolved: 2026-08-05T17:05:00Z
fix_commit: f01df96
---

## Symptoms

DATA_START
- **Expected:** After a sync completes, the Proton header chip shows its settled state (green dot, last-sync info in tooltip) and the manual refresh button is available. "Syncing..." text and the spinner should only appear transiently while a sync run is actually in flight.
- **Actual:** The Proton source chip shows a GREEN dot with the permanent label "Syncing..." and a perpetually spinning icon in place of the refresh button (screenshot confirmed). This appears in the header health chip and the refresh control. Messages themselves load fine — stream rows, detail pane, deep links all work. Other sources are unaffected.
- **Errors:** None in the browser devtools console on page load (user checked earlier today). Kernel-side logs not yet inspected.
- **Timeline:** Used to settle to green normally after Phase 3 shipped. Started sticking at some unknown point; first reported 2026-08-04 in the UI/UX issue list. Present across kernel restarts.
- **Reproduction:** Open any webspace with the Proton source configured — the chip is permanently in this state.
DATA_END

## Context (orchestrator-supplied, from project history)

- Phase 2 D-07 (02-CONTEXT.md): the UI's syncing indicator is driven by `sync_runs.status = "running"`; the stream re-fetches on completion. If the latest Proton `sync_runs` row is stuck at `running`, the UI would show exactly this.
- Phase 2 02-02: the single-flight coordinator (`kernel/syncer`) owns the `sync_runs` rows — it records `running` at start and finalizes to ok/error. `GET /api/sources` `last_error` is sourced exclusively from `sync_runs` history.
- healthTone (02-03): never-synced takes precedence; green dot means recent successful history + reachable — consistent with a stuck `running` row that sits ALONGSIDE an older successful run, or with the UI keying "syncing" off a different field than health.
- Environment note: on 2026-08-05 several kernel processes were force-killed mid-session (stale-process cleanup during unrelated work, incl. SIGKILL on plugin subprocesses). An interrupted sync could plausibly have left an unfinalized `running` row. However the user says the sticking predates today (reported 2026-08-04), so a single orphaned row may not be the whole story — check whether the coordinator EVER finalizes the row on some path (e.g. a Proton-specific error path), whether startup cleans up stale running rows, and whether the UI picks latest-row-by-time vs any-running-row.
- Index DB: ~/.local/share/webspaces/index.db (sqlite; `sync_runs` table per kernel/index/schema.go). Inspect read-only.
- Relevant code: kernel/syncer (coordinator), kernel/httpapi/sources.go (status merge), web/src/lib/components/WebspaceHeader.svelte / SourceHealthChip.svelte (chip + refresh spinner state).

## Current Focus

hypothesis: CONFIRMED (multi-cause, AND-gate) — see Resolution.
test: complete — all three causes confirmed by direct evidence plus a deterministic reproduction experiment.
expecting: n/a
next_action: none — session resolved and archived. Fix shipped in f01df96 and confirmed in the live UI by the user on 2026-08-05.

bug_class: Bohrbug (deterministic — reproduces on every page load, persists across restarts)

## Evidence

- timestamp: 2026-08-05T16:20:00Z
  checked: sync_runs table grouped by source_type/status (read-only)
  found: Exactly ONE row with status='running' in the entire table — id=772, source_type=proton, started_unix=1785801715 (2026-08-04 00:01:55), finished_unix=NULL, error='', item_count=0. No other source has a running row.
  implication: Confirms an orphaned running row exists, and its date matches the user's "first reported 2026-08-04" timeline exactly. Explains why ONLY proton is affected.

- timestamp: 2026-08-05T16:20:00Z
  checked: proton sync_runs history relative to the orphaned row
  found: proton has 244 rows with status='ok' spanning up to 2026-08-05 15:08:41 (id=892) — i.e. 244 successful runs completed AFTER the orphaned row 772. Latest proton row overall is ok, not running.
  implication: The UI/API cannot be selecting the LATEST row per source (that would be 'ok' and would render settled). It must be matching ANY row with status='running'. This also explains the green dot coexisting with "Syncing..." — health reads recent successful history while the syncing flag reads the orphan.

- timestamp: 2026-08-05T16:30:00Z
  checked: kernel/index/store.go SyncingSourceTypes vs LatestSyncRunPerSource
  found: SyncingSourceTypes (store.go:581) = `SELECT DISTINCT source_type FROM sync_runs WHERE status='running' AND finished_unix IS NULL` — ANY row, no per-source latest restriction, no recency bound. LatestSyncRunPerSource (store.go:552) = `WHERE id IN (SELECT MAX(id) ... GROUP BY source_type)` — correctly latest-per-source. httpapi/sources.go:105-106 sets Syncing from the former and LastStatus from the latter.
  implication: Mechanism fully explains BOTH observed symptoms simultaneously — green dot (last_status='ok' from row 892) AND permanent "Syncing..." (syncing=true from orphan row 772). Two different queries over the same table disagree.

- timestamp: 2026-08-05T16:32:00Z
  checked: live GET /api/sources on the running kernel (127.0.0.1:7777)
  found: proton entry is `"reachable":true,"syncing":true,"last_status":"ok","last_sync_unix":1785942521,"last_error":""`. All three other sources report syncing:false.
  implication: Direct confirmation in the live system, not just inferred from the DB. The API — not the frontend — is the origin of the stuck flag. The UI is rendering the API faithfully; no frontend bug.

- timestamp: 2026-08-05T16:35:00Z
  checked: grep for any startup/boot reconciliation of stale running rows across kernel/ and cmd/
  found: StartSyncRun/FinishSyncRun are called ONLY from syncer/coordinator.go syncOne. No reconcile/stale/orphan/interrupted cleanup logic exists anywhere in the codebase.
  implication: Nothing ever heals an orphaned row. Explains "present across kernel restarts" — the row from 2026-08-04 survives every boot indefinitely.

- timestamp: 2026-08-05T16:40:00Z
  checked: EXPERIMENT — ran a temporary test cancelling ctx mid-sync (simulating main.go:156 cancel() on graceful shutdown) and then querying the store with a fresh ctx
  found: `RunResult: status="error" err="finish sync run: index: finish sync run 1: context canceled"` and `syncing[proton]=true latestRun={Status:running FinishedUnix:0}`. ORPHAN REPRODUCED DETERMINISTICALLY.
  implication: The orphan is NOT merely a SIGKILL artifact. syncOne passes the same cancellable ctx to FinishSyncRun that the sync ran under; database/sql rejects the UPDATE outright when that ctx is already canceled, so the finalize write is SKIPPED on every graceful shutdown that lands mid-sync. Proton is the slowest source (IMAP over the network to Bridge) so it has by far the widest window to be mid-sync at shutdown — explains why only proton is affected.

## Reasoning Checkpoint

```yaml
reasoning_checkpoint:
  hypothesis: "The permanent 'Syncing...' state is produced by THREE independent defects acting together: (C1) syncOne finalizes the sync_runs row using the same cancellable ctx the sync ran under, so graceful shutdown mid-sync skips the finalize write and orphans the row; (C2) nothing reconciles orphaned running rows at startup, so the orphan survives restarts forever; (C3) SyncingSourceTypes matches ANY running row rather than each source's latest, so one stale orphan outvotes 244 subsequent successful runs."
  confirming_evidence:
    - "Exactly one running row exists table-wide: id=772 proton, finished_unix NULL, started 2026-08-04 00:01:55 — matching the user's first-report date"
    - "Live GET /api/sources returns proton syncing:true AND last_status:ok simultaneously — the two-query disagreement, observed directly"
    - "Experiment reproduced the orphan deterministically: FinishSyncRun failed with 'context canceled' and left the row running"
    - "grep proves no reconciliation logic exists anywhere in kernel/ or cmd/"
  falsification_test: "If SyncingSourceTypes were restricted to each source's latest row, proton's latest row (id=892, status=ok) would yield syncing:false despite orphan 772 still existing. If that did NOT clear the flag, C3 is not the amplifier."
  fix_rationale: "Each cause is fixed at its own layer rather than papering over the symptom: C1 finalizes via a detached context so an outcome that already happened is always recorded; C2 reconciles orphans at boot (which also self-heals the user's existing row 772 without hand-editing their DB); C3 bounds the blast radius so any future orphan can never permanently pin the indicator. Fixing only C3 would hide orphans while still losing sync outcomes; fixing only C1 would leave row 772 stuck forever."
  blind_spots: "Have not confirmed whether row 772 specifically arose from ctx-cancellation vs the SIGKILL noted in project context — both produce an identical row and both are covered by the C2 boot reconciliation, so the distinction does not change the fix. Have not yet checked whether the frontend caches syncing state independently of /api/sources."
  candidate_causes:
    - "code: FinishSyncRun called with an already-canceled ctx (orphan creation)"
    - "code: SyncingSourceTypes any-running semantics (amplifier that makes it permanent)"
    - "code: absence of boot-time reconciliation (persistence across restarts)"
    - "environment: kernel/plugin processes SIGKILLed mid-sync on 2026-08-05 (an alternative orphan-creation trigger; same remedy as C2)"
    - "data: the single pre-existing orphan row 772 in the user's index.db — must be healed, not just prevented"
  and_gate: "YES — all three code causes are required for the PERMANENT symptom. Without C3 (latest-row semantics) the next successful run's row would become latest and clear the flag, making it transient. Without C2 a restart would clear it, but the user reports it survives restarts. Without C1 no orphan is created on the graceful path at all. This is genuinely multi-cause, not a single fault."
```

## Eliminated

- Frontend state caching — SourceHealthChip.svelte reads `source.syncing` straight from /api/sources
  with no independent local state or caching. The UI rendered the API faithfully; no frontend change
  was needed. (Was an open blind spot at the reasoning checkpoint; closed before the fix.)
- "A single orphaned row from the 2026-08-05 SIGKILL cleanup" as the WHOLE story — correctly doubted
  in the orchestrator-supplied context. The row predates that cleanup (2026-08-04) and the
  graceful-shutdown ctx-cancellation path reproduces it deterministically with no SIGKILL involved.
  Both triggers produce an identical row and both are covered by the C2 boot reconciliation.
- A Proton-plugin-specific error path failing to finalise — ruled out. The defect is in the shared
  coordinator, not in the Proton plugin. Proton is affected only because it has the widest mid-sync
  window (slowest source, IMAP over the network to Bridge); any source can hit this.
- UI keying "syncing" off a different field than health (one of the two original hypotheses) —
  half-right and worth recording precisely: the UI does read a different FIELD (`syncing` vs
  `last_status`), but faithfully. The disagreement originates one layer down, in two different
  SQL queries over the same table.

## Resolution

root_cause: |
  THREE contributing causes (AND-gate fired — all required for the permanent symptom):

  C1 (orphan creation) — kernel/syncer/coordinator.go syncOne finalised the sync_runs row via
  FinishSyncRun(ctx, ...) using the SAME cancellable ctx the sync ran under. When that ctx is
  cancelled mid-sync (graceful shutdown cancels the scheduler ctx at cmd/webspaces/main.go:156,
  or an HTTP client disconnects behind a manual refresh), database/sql rejects the finalising
  UPDATE outright with "context canceled" — so the row is left permanently at status "running".
  Reproduced deterministically by experiment. Proton is the only source affected in practice
  because, as the slowest source (IMAP over the network to Bridge), it has by far the widest
  window to be mid-sync when shutdown lands.

  C2 (persistence across restarts) — nothing anywhere in the kernel reconciled orphaned running
  rows. StartSyncRun/FinishSyncRun are called only from syncOne, so once a row is stranded no
  code path ever finalises it. It survived every restart indefinitely.

  C3 (amplifier making it permanent) — kernel/index/store.go SyncingSourceTypes matched ANY row
  with status='running', with no per-source latest-row restriction and no recency bound. So the
  single orphan (id=772, 2026-08-04 00:01:55) outvoted 244 subsequent successful proton runs.
  Meanwhile LatestSyncRunPerSource correctly returned the latest row (ok) — the two queries
  disagreeing is exactly why the chip showed a GREEN dot AND "Syncing..." simultaneously.

fix: |
  Fixed each cause at its own layer rather than papering over the symptom:

  C1 — syncOne now finalises on context.WithoutCancel(ctx) bounded by a 5s finishRunTimeout. The
  run's outcome has already happened by that point, so recording it must never be skipped because
  the triggering context was cancelled.

  C2 — added Store.ReconcileInterruptedSyncRuns, called at kernel startup in runServe. A freshly
  started kernel has no in-flight runs by definition (the coordinator is the only writer and holds
  run IDs in memory), so any row still "running" at boot was stranded by a previous process. Rows
  are marked "error"/interrupted rather than deleted — the run was genuinely attempted and
  genuinely did not complete. A repair failure logs but never blocks boot.

  C3 — SyncingSourceTypes now restricts to each source's latest row (MAX(id) GROUP BY source_type),
  so an older stranded row can never outvote a completed run. Semantically correct independent of
  the bug: "is this source syncing right now" is a property of its current run.

verification: |
  guardrail_verdict: accepted
  oracle_type: specified (the intended contract — "sources syncing right now" — is explicit in the
  API contract and the method's documented behaviour)

  signal_1_reproduction (test fails pre-fix): PASS. New tests run against pre-fix code:
  TestRefresh_CancelledContextStillFinalisesSyncRun failed with the exact production symptom
  ("Status:running FinishedUnix:0"). Targeted mutation reverting ONLY the SyncingSourceTypes
  WHERE clause to the old any-running form failed both
  TestSyncingSourceTypes_OrphanedRunDoesNotOutvoteLaterCompletedRun and the in-flight boundary
  subtest. Tests bite on the real defect, not on scaffolding.

  signal_2_regression: PASS. Full `go test ./...` green (audit, config, correlate, httpapi, index,
  syncer). `go build ./...`, `go vet ./kernel/... ./cmd/...` and `gofmt -l` all clean. Pre-existing
  sync-lifecycle tests (TestStartAndFinishSyncRun, TestSyncingSourceTypes_UnrelatedSourceUnaffected,
  TestSourcesHandler_RunningRowReportsSyncingTrue) still pass unmodified.

  signal_3_revert: PASS. Bug returns on revert — demonstrated by the mutation above for C3 and by
  the stashed-source run for C1.

  signal_4_diff_review: PASS. Not deletion-only; additive (one new store method, one startup call,
  one detached-context finalise) plus one justified query-semantics change. No test was weakened or
  removed; no assertion loosened.

  signal_5_real_data: PASS. Ran the fix against a COPY of the user's actual index.db (real DB opened
  read-only / never mutated, confirmed still holding its orphan afterwards). Results: with the new
  latest-row query the copy reported syncing=map[] BEFORE reconciliation even ran — i.e. C3 alone
  clears the user's live symptom — and ReconcileInterruptedSyncRuns then repaired exactly 1 stranded
  row, turning id=772 into status=error / "interrupted: kernel stopped before this sync run
  finished", leaving zero running rows.

  blind_spot_closed: SourceHealthChip.svelte reads source.syncing straight from /api/sources with no
  independent caching or local state, so no frontend change is needed — confirming the API fix is
  sufficient.

  signal_6_human_verify (live system): PASS. The self-verified checks could not cover the user's own
  running instance — their kernel still held the OLD binary and their real index.db still contained
  orphan row 772, so the fix only took effect on the next restart. Confirmed by the user on
  2026-08-05: kernel restarted, boot-time reconciliation ran, and the Proton chip settled — no more
  perpetual "Syncing..." text or spinner, refresh control back to normal. No remaining symptoms
  reported. This closes the one gap the agent could not close itself.

fix_commit: f01df96 — "fix(syncer): stop orphaned sync runs pinning the syncing indicator on"
  (source files only; no .planning/ artifacts; atomic on main)

prevention: |
  Blameless postmortem — why this was not caught earlier:

  No gate existed for this class. The kernel had tests for the sync lifecycle's HAPPY path
  (TestStartAndFinishSyncRun) and even for the running-row rendering path
  (TestSourcesHandler_RunningRowReportsSyncingTrue), but nothing exercised the INTERRUPTED path —
  what the persisted state looks like when a run is cancelled partway. The existing tests all
  assumed a run reaches its own finaliser with a live context, which is exactly the assumption
  that breaks at shutdown. C3 (the any-running query) was likewise never wrong on its own in any
  test, because no test ever created a stale row and a later completed row for the same source.

  Guard now in place: TestRefresh_CancelledContextStillFinalisesSyncRun (coordinator_test.go)
  pins the C1 contract — an outcome that has already happened is always recorded, regardless of
  the triggering context's state. TestSyncingSourceTypes_OrphanedRunDoesNotOutvoteLaterCompletedRun
  plus the orphan-boundary and reconciliation tests (store_test.go) pin C2/C3. Boot-time
  ReconcileInterruptedSyncRuns additionally makes the whole class self-healing rather than
  merely prevented — future orphans from any cause (including SIGKILL) are repaired at startup.

  Generalised pattern recorded in .planning/debug/knowledge-base.md: "graceful shutdown cancels
  the context that also finalises the DB write". Directly relevant to the upcoming Phase 8
  WhatsApp plugin's event-stream persistence and any other two-phase write in the Phase 4-5
  plugins.

files_changed:
  - kernel/syncer/coordinator.go: finalise sync_runs on a detached (WithoutCancel) context with a 5s bound; added finishRunTimeout
  - kernel/index/store.go: SyncingSourceTypes restricted to each source's latest run; added ReconcileInterruptedSyncRuns
  - cmd/webspaces/main.go: reconcile interrupted sync runs at kernel startup in runServe
  - kernel/index/store_test.go: regression tests for orphan-does-not-outvote, orphan boundaries, and reconciliation
  - kernel/syncer/coordinator_test.go: regression test proving a cancelled context still finalises the run
