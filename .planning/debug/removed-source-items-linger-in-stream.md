---
status: diagnosed
trigger: "G-07-7: Removing a source from a webspace (chip ⋮ menu → 'Remove from this webspace') removes the chip immediately, but the removed instance's items remain visible in the stream until a manual page refresh."
created: 2026-08-09T00:00:00Z
updated: 2026-08-09T00:00:00Z
---

## Current Focus
<!-- OVERWRITE on each update - reflects NOW -->

hypothesis: CONFIRMED — see Resolution. The stale webspace_items purge for a de-allowlisted (webspace, source) pair happens only inside the fire-and-forget eager re-sync goroutine Supervisor.Apply dispatches (go coord.Refresh), which completes long after PUT /api/config's 200; the client's immediate stream refetch therefore always reads the pre-purge join rows.
test: complete — full causal chain read directly from source (config.go handler → supervisor.Apply → correlate.SyncSource → index.ReplaceWebspaceSourceItems → stream.go join → +page.svelte handlers)
expecting: n/a — diagnosis complete
next_action: return ROOT CAUSE FOUND to orchestrator (goal: find_root_cause_only — no fix applied)
known_pattern_candidate: "KB-002 (weak match, borne out in spirit): chip row and stream list are backed by two different state sources with different update timing — chip from the synchronously-swapped config, stream from asynchronously-purged webspace_items rows"
bug_class: Bohrbug in practice — the async purge window (≥1 plugin Match RPC round trip per webspace) is always orders of magnitude longer than the ~ms gap between the PUT 200 and the client's refetch, so the stale read wins every time

reasoning_checkpoint:
  hypothesis: "The (webspace, removed-source) webspace_items rows are purged only by the eager re-sync goroutine (supervisor.go:399-406, `go coord.Refresh(...)`) via correlate.SyncSource's participates==false branch; PUT /api/config returns 200 before that goroutine finishes, so handleRemoveSource's immediate getStream() reads the stale join rows."
  confirming_evidence:
    - "ConfigSaveHandler (kernel/httpapi/config.go:153-193) is synchronous through Apply — but Apply itself (supervisor.go:337-409) returns nil after dispatching the webspace-changed re-syncs as detached goroutines, so the 200 does NOT wait for the purge."
    - "removedInstances (supervisor.go:454-463) diffs cfg.Sources only — a webspace-narrowing leaves the [sources.<id>] block intact, so cleanupRemovedInstances deletes nothing synchronously (correct globally, but means no sync-path purge either)."
    - "correlate.SyncSource (correlate.go:88-103) is the ONLY purge site for de-participated pairs: participates==false → ReplaceWebspaceSourceItems(ctx, name, src, nil) clears the rows — reachable only during a sync."
    - "StreamHandler (kernel/httpapi/stream.go:66-104) serves items purely from the webspace_items join (store.go:333-359); it applies the saved keyword filter live from config but NEVER the participation/allowlist — so read-time config freshness cannot mask stale join rows."
    - "handleRemoveSource (+page.svelte:204-224) DOES refetch the stream immediately after the 200 (await Promise.all([loadSources(), load(navGeneration)])) — the refetch happens, it just lands before the purge."
    - "ensurePolling (+page.svelte:453-462) — the WR-03 sync-completion poll — reloads ONLY loadSources() on each tick and simply stops when syncing goes false; it never calls load(), so even when the poll observes the eager re-sync finishing, the stream is never refetched. The self-healing path is foreclosed."
  falsification_test: "Instrument or observe: if a fresh GET /api/webspaces/{ws}/stream issued immediately after the PUT 200 returned the correct (purged) item set, the hypothesis would be false. Code reading shows this is impossible: the purge DELETE runs inside the goroutine's later transaction; nothing on the synchronous PUT path touches webspace_items."
  fix_rationale: "n/a — diagnose-only session; fix direction recorded in Resolution."
  blind_spots: "Not live-reproduced with timing logs (code-reading diagnosis); if the eager re-sync FAILS (source unreachable) the rows linger until the next successful scheduled sync, which is a worse variant of the same cause, not a different cause. Did not verify Coordinator.Refresh's StartSyncRun timing vs the client's loadSources — irrelevant to the outcome since the poll never reloads the stream anyway."
  candidate_causes:
    - "code (kernel): purge of de-participated webspace_items deferred to an async goroutine past the PUT response — PRIMARY, sufficient alone"
    - "code (client): WR-03 poll reloads sources only, never the stream — contributing; forecloses the only mechanism that could have self-healed the stale view without a manual refresh"
    - "config: ruled out — config.toml narrows correctly per UAT; the config swap itself is synchronous (Store.Save before Apply)"
    - "data: ruled out — index rows are correct per the hybrid model (instance still configured globally, items rightly retained in `items`); only the (webspace, source) join rows are momentarily stale, by design of the sync-time correlation model"
  and_gate: "yes, for the full user-visible symptom: the primary cause creates the stale window; the contributing client cause makes the window permanent-until-manual-refresh. The primary alone explains 'items linger at remove time'; without the contributing cause the items would still linger ~seconds (until the poll noticed sync completion). Both are worth naming; the kernel-side deferral is the root."

## Symptoms
<!-- Written during gathering, then IMMUTABLE -->

expected: When the source is removed from the webspace, its items disappear from the visible stream at the same time the chip disappears — no manual refresh needed.
actual: "when the source is removed, its items remain visible in the stream until a refresh occurs. This is counter-intuitive, they should be removed as the chip is removed. Otherwise it's a pass" (verbatim user report). Chip removal, config.toml narrowing, other webspaces, and the re-add round-trip all behave correctly.
errors: None reported.
reproduction: Test 4 in .planning/phases/07-webspace-builder-ui/07-UAT.md — `make dev`, open a chip's ⋮ menu → "Remove from this webspace"; observe the stream list keeps showing the removed instance's items until reload.
started: Discovered during round-2 UAT (2026-08-09), first live run after gap-closure plan 07-14 landed (removeSourceFromWebspace participant-set seeding + chip row derived from shared participatesIn predicate). The chip now updates correctly; the stream item list does not.

## Eliminated
<!-- APPEND only - prevents re-investigating -->

- hypothesis: "Client never refetches the stream after a successful remove (stale client state, missing invalidation)"
  evidence: "handleRemoveSource (+page.svelte:204-224) explicitly awaits Promise.all([loadSources(), load(navGeneration)]) after putConfig succeeds — the stream IS refetched immediately. The refetch simply returns stale data."
  timestamp: 2026-08-09

- hypothesis: "Kernel stream handler reads a stale config snapshot captured at Router construction"
  evidence: "StreamHandler (stream.go:66-68) reads cfgStore.Expanded() fresh per request, and the config swap (Store.Save) completes before Apply is even called. Config freshness is not the issue — the handler never applies participation to the query at all; membership comes solely from webspace_items rows."
  timestamp: 2026-08-09

- hypothesis: "PUT /api/config returns before Supervisor.Apply runs at all (fully async apply)"
  evidence: "ConfigSaveHandler (config.go:185-191) calls applier.Apply(r.Context()) synchronously and returns 500 apply_failed on error — Apply as a whole IS synchronous. Only the eager re-sync dispatch inside it (supervisor.go:399-406) is fire-and-forget."
  timestamp: 2026-08-09

## Evidence
<!-- APPEND only - facts discovered -->

- timestamp: 2026-08-09
  checked: .planning/debug/knowledge-base.md
  found: No direct pattern match. KB-002 (two queries backing two UI fields disagree) is a weak analogue — chip row and stream list are backed by different state.
  implication: Treat as hypothesis flavor only; investigate normally.

- timestamp: 2026-08-09
  checked: web/src/routes/w/[webspace]/+page.svelte handleRemoveSource (204-224)
  found: "After putConfig 200: configResponse = res (chip row updates instantly via participatesIn over new config), then await Promise.all([loadSources(), load(navGeneration)]) — a genuine immediate stream refetch."
  implication: "Client behavior is correct-in-shape; the bug must be that the refetched data is itself stale. Shifts investigation kernel-side."

- timestamp: 2026-08-09
  checked: kernel/httpapi/config.go ConfigSaveHandler (153-193)
  found: "Save (validate + write + in-memory swap) then applier.Apply(r.Context()) run synchronously before the 200 is written."
  implication: "Any staleness must come from work Apply defers rather than completes."

- timestamp: 2026-08-09
  checked: kernel/httpapi/stream.go StreamHandler (66-104) + kernel/index/store.go StreamItems (333-359)
  found: "Stream = SELECT items JOIN webspace_items WHERE webspace_name = ?. Webspace membership is a sync-time correlation artifact (webspace_items rows), not a read-time predicate. Config is only consulted for display names and the saved keyword filter."
  implication: "A config change removing a source from a webspace changes NOTHING in stream output until the webspace_items rows for that (webspace, source) pair are physically deleted."

- timestamp: 2026-08-09
  checked: kernel/supervisor/supervisor.go Apply (337-409), removedInstances (454-463), cleanupRemovedInstances (438-450)
  found: "For a webspace-narrowing, removedInstances is EMPTY ([sources.<id>] intact) → no synchronous index cleanup. Because oldCfg.Webspaces != newCfg.Webspaces, Apply dispatches `go coord.Refresh(context.Background(), name)` for every connection-unchanged instance and returns nil WITHOUT waiting."
  implication: "The only path that will purge the stale rows is dispatched fire-and-forget; PUT 200 races it and always wins the race from the client's perspective."

- timestamp: 2026-08-09
  checked: kernel/correlate/correlate.go SyncSource (84-137) + matchFieldsFor (170-188), kernel/index/store.go ReplaceWebspaceSourceItems (191-241)
  found: "The de-allowlisted purge exists and is correct: participates==false → ReplaceWebspaceSourceItems(ctx, ws, src, nil) deletes the (webspace, source) join rows ('a de-allowlisted instance leaves no orphaned rows behind', ROADMAP criterion 3). But it is reachable ONLY inside a sync — i.e., only when the async eager Refresh (or a later scheduled sync) actually runs and completes, which involves ≥1 Match RPC per webspace to the plugin subprocess (seconds, not ms)."
  implication: "Purge timing = async re-sync completion. Explains why the user's manual refresh (seconds later) shows the clean stream: it lands after the goroutine finished. This is NOT the DeleteSourceItems path (that's for whole-instance removal only)."

- timestamp: 2026-08-09
  checked: web/src/routes/w/[webspace]/+page.svelte ensurePolling (452-462), loadSources (426-444)
  found: "The WR-03 syncing poll ticks loadSources() only and stops when no source reports syncing — it never calls load(). So even when the poll observes the eager re-sync completing, the stream list is never refetched."
  implication: "Contributing cause: the one existing mechanism that notices sync completion cannot heal the stale stream. Without a manual refresh the stale items persist indefinitely on screen."

## Resolution
<!-- OVERWRITE as understanding evolves -->

root_cause: "Two contributing causes, kernel-side primary: (1) PRIMARY — webspace membership in the stream is a sync-time artifact (webspace_items rows, stream.go:82/store.go:348), and for a source removed from a webspace's allowlist the only purge path is correlate.SyncSource's participates==false branch (correlate.go:96), reachable only during a sync; Supervisor.Apply dispatches that sync fire-and-forget (`go coord.Refresh(...)`, supervisor.go:403) and returns before it completes, so PUT /api/config's 200 arrives while the stale rows still exist — the client's immediate, correctly-implemented stream refetch (handleRemoveSource, +page.svelte:212) reads them every time. removedInstances/cleanupRemovedInstances covers only whole-instance removal (cfg.Sources diff) and is rightly empty here. (2) CONTRIBUTING — the client's WR-03 sync poll (ensurePolling, +page.svelte:453-462) reloads only the source list on each tick and never reloads the stream, so when the eager re-sync does finish, nothing refetches; the stale items persist until a manual refresh."
fix: "(not applied — diagnose-only) Suggested direction: in Supervisor.Apply, synchronously purge de-participated pairs before returning — diff old vs new participation per (webspace, source) (Webspace.Participates + the D-20 no-match-input rule, i.e. the same predicate matchFieldsFor applies) and call idx.ReplaceWebspaceSourceItems(ctx, ws, src, nil) for each pair that flipped true→false. This is a pure local DB write (no plugin RPC), preserves D-06's 'save = apply immediately' at the stream boundary, and makes the client's existing refetch just work. Optionally also make the WR-03 poll reload the stream when syncing transitions true→false (general freshness win, and covers the purge-on-next-sync fallback when an eager re-sync fails)."
verification: "n/a — no fix applied"
files_changed: []
oracle_type: n/a (diagnose-only)
