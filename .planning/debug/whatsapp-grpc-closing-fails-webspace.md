---
status: diagnosed
trigger: "G-08-3 (whatsapp-grpc-closing-fails-webspace): Right after successfully linking a WhatsApp source (real device, in-app QR pairing), opening a webspace failed entirely with 'Couldn't load this webspace / The topos service didn't respond — check that it's running, then retry.' Kernel logged: whatsapp: match against source \"whatsapp\": rpc error: code = Canceled desc = grpc: the client connection is closing"
created: 2026-08-10T22:04:19+01:00
updated: 2026-08-10T22:55:00+01:00
---

## Current Focus
<!-- OVERWRITE on each update - reflects NOW -->

bug_class: Bohrbug once the state exists (stale coordinator handle -> every sync fails; latest error run + zero items -> full-page error, deterministically). The state-creating event (suspend/resume without generation rebuild) is a lifecycle-design gap, not a probabilistic race.
known_pattern_candidate: "new-webspace-transient-service-error (G-07-1) — same UI copy misclassification family (confirmed relevant: co-cause P below)"

hypothesis: CONFIRMED — see reasoning_checkpoint and Resolution
next_action: return ROOT CAUSE FOUND to orchestrator (goal: find_root_cause_only; no fix applied)

reasoning_checkpoint:
  hypothesis: "AND-gate, two co-causes. (K) kernel/supervisor's WhatsApp link suspend/resume lifecycle operates on the plugin HOST only: SuspendInstance kills the instance's go-plugin client (Host.Reconcile -> Plugin.Kill -> local ClientConn.Close) without stopping the scheduler or rebuilding the coordinator, and its resume closure relaunches the instance in the host but also never rebuilds the coordinator/scheduler; since syncer.Coordinator captures *pluginhost.Plugin handles once at construction and Host.Reconcile creates NEW *Plugin values, every sync path (scheduled tick, manual refresh, eager refresh — all resolve through the stale coordinator) calls Match on the killed client from suspension onward, failing with grpc-go's ErrClientConnClosing ('rpc error: code = Canceled desc = grpc: the client connection is closing') until the next config save (Apply -> commitGeneration) or kernel restart; a re-link never saves config, so the failure persists indefinitely. (P) That failure is escalated to an apparent total outage by the presentation path: aggregateSyncStatus folds ANY configured source's latest errored run into the stream response's single sync object (errParts 'source: run.Error' = the user's exact quoted string), and the client's streamVariant maps sync.status==='error' && items.length===0 to a full-page StreamError whose fixed copy falsely claims 'The topos service didn't respond'."
  confirming_evidence:
    - "Error string forensics: grpc-go emits 'grpc: the client connection is closing' (codes.Canceled) ONLY for RPCs on a locally-Close()d ClientConn — kernel-side Kill(), not plugin crash (Unavailable) and not ctx cancellation ('context canceled')"
    - "supervisor.go:245-295 — SuspendInstance kills via Host.Reconcile, doc-comment-explicitly does NOT stop the scheduler; resume closure (285-293) calls only s.host.Reconcile — no commitGeneration, no coordinator rebuild, no scheduler restart"
    - "coordinator.go:63-79 — Coordinator captures sources map at construction, no update seam; supervisor.go Fetch/Refresh doc comments themselves state 'Apply replaces the *syncer.Coordinator outright' because reconciled hosts produce NEW *Plugin values (host.go:190-219 launch/Kill)"
    - "sources.go:214-249 aggregateSyncStatus — status 'error' if ANY source's latest run errored; errParts = source+': '+run.Error reproduces the user's quoted 'whatsapp: match against source \"whatsapp\": rpc error: ...' byte-for-byte, proving the user saw the sync field of a 200 response (rendered by StreamError's syncError prop) — the kernel did not fail the request"
    - "format.ts:216-222 + StreamList.svelte:69-70 + StreamError.svelte — sync error + zero items renders the full-page 'Couldn't load this webspace / The topos service didn't respond' copy"
    - "RelinkModal.svelte passes instance (triggers suspend); onrelinked only refetches health — no config save, so nothing rebuilds the coordinator post-re-link; link.go's already_linked->paired no-QR path is the natural in-app 'no second QR' confirmation used in UAT test 1"
    - "plugins/whatsapp Match (plugin.go:193-245) is a fast local store read — rules out mid-flight-kill as the probable mechanism; the handle was dead before Match was issued"
  falsification_test: "Would be disproven if (a) resume rebuilt the coordinator (it doesn't — single Reconcile call, grep-verified no commitGeneration outside Apply), (b) the Coordinator resolved sources through the live host per call (it doesn't — captured map), or (c) the stream endpoint had returned non-200 (ruled out: the quoted string exists only in the 200 response's sync.error / scheduler log). Live falsification: on a dev kernel, SuspendInstance a mock source, resume it, call POST /api/sources/{name}/refresh -> expect exactly 'rpc error: code = Canceled desc = grpc: the client connection is closing'; then PUT any config change (Apply) -> refresh succeeds again."
  fix_rationale: "n/a — diagnose-only mode; fix direction recorded in Resolution"
  blind_spots: "The operator's exact click path (re-link vs add-flow) is not reconstructible from artifacts — UAT test 1 note says 'details pending'. This does not weaken the mechanism: the Add-Source flow passes no instance (no suspend) and cannot produce a locally-closed conn, so SOME instance-named link session (re-link entry, most plausibly the 'no second QR' reconnect confirmation) must have run in the same kernel process. Not measured live: no ephemeral-kernel repro this session (static analysis only); timing of the failing sync run (tick vs manual refresh) unknown but immaterial — every sync path shares the stale coordinator."
  candidate_causes:
    - "code (kernel lifecycle): suspend/resume bypasses the coordinator/scheduler generation contract — CONFIRMED primary (K)"
    - "code (API+client presentation): global any-source sync-error aggregation + zero-items escalation to full-page service-unreachable copy — CONFIRMED co-cause (P), same family as resolved G-07-1/G-07-4"
    - "environment (WhatsApp service / phone side): remote de-link or connect failure — ELIMINATED (error is a local ClientConn close, not a transport/Unavailable failure)"
    - "data (index corruption / stale rows): ELIMINATED (sync_runs row accurately records a real RPC failure; index reads served the 200 normally)"
  and_gate: "YES — the reported symptom needs BOTH K (whatsapp's latest sync_run pinned at the closing-connection error) AND P (that error escalated to a full-page 'service didn't respond' for a zero-item webspace). Fixing only K removes this trigger but leaves P latent for every future single-source failure (violating Phase 8's degradation guarantee generally); fixing only P leaves whatsapp sync silently broken after every re-link until a config save or restart."

## Symptoms
<!-- Written during gathering, then IMMUTABLE -->

expected: After linking a WhatsApp source, opening a webspace loads the stream; if the whatsapp plugin's gRPC connection is unavailable (e.g. mid-restart right after pairing, or its plugin instance being resumed/replaced), that source degrades — health error surfaced on its chip — without failing the entire webspace load.
actual: The whole webspace failed to load with "Couldn't load this webspace / The topos service didn't respond — check that it's running, then retry."
errors: 'kernel log: whatsapp: match against source "whatsapp": rpc error: code = Canceled desc = grpc: the client connection is closing'
reproduction: Real-device UAT (test 1 of 08-UAT.md): make dev; Add Source → New WhatsApp… with display name + seeded local path; scan QR with a real phone; pairing succeeds; then open/load a webspace containing (or while the kernel holds) the whatsapp source. Error observed immediately after the pairing flow completed.
started: Discovered during Phase 8 UAT on 2026-08-10, immediately after the first successful real-device pairing via the new in-app QR flow (plans 08-03..08-08).

## Eliminated
<!-- APPEND only - prevents re-investigating -->

- hypothesis: "The kernel failed/hung/crashed on the webspace stream request itself ('service didn't respond' taken literally)"
  evidence: "StreamHandler (kernel/httpapi/stream.go) reads only the index + config store — structurally unable to reach a plugin; the user's quoted 'kernel error' string is byte-identical to aggregateSyncStatus's errParts format (source+': '+run.Error), which only appears in the sync field of a SUCCESSFUL 200 stream/webspaces response (and in scheduler logs). StreamError additionally renders syncError only in the sync-failed-with-zero-items branch, which requires a 200 response. The kernel answered normally."
  timestamp: 2026-08-10T22:30:00+01:00

- hypothesis: "The whatsapp plugin subprocess crashed/exited on its own (e.g. store-lock loss, whatsmeow failure), breaking the connection"
  evidence: "A self-exiting plugin yields codes.Unavailable transport errors ('transport is closing'/'connection refused'), never grpc-go's ErrClientConnClosing (codes.Canceled, 'grpc: the client connection is closing'), which is produced exclusively by a LOCAL ClientConn.Close() — i.e. the kernel's own goplugin.Client.Kill(). Also the whatsapp plugin's Match guard would return codes.Unavailable with a 'whatsapp: <state message>' prefix, not this transport-level error."
  timestamp: 2026-08-10T22:38:00+01:00

- hypothesis: "A scheduler-generation sync was cancelled mid-flight by Apply/Shutdown (stopScheduler), producing the error"
  evidence: "stopScheduler cancels the generation ctx and blocks until Run returns; an in-flight Match aborted via ctx cancellation records 'context canceled', not 'the client connection is closing'. Apply's cancel-and-block discipline (07-02) prevents the scheduler generation from ever calling into a plugin set Reconcile is tearing down. The observed message requires the conn's local Close, not ctx cancellation."
  timestamp: 2026-08-10T22:42:00+01:00

- hypothesis: "A kill landed while a whatsapp Match was in flight (mid-RPC Close)"
  evidence: "Not impossible in general (pending RPCs on Close also get ErrClientConnClosing, and the untracked context.Background eager-refresh goroutines are a real exposure — kept as a latent sibling), but improbable as THIS incident's mechanism: whatsapp's Match is a fast local scratch-store read (plugins/whatsapp/plugin.go:193-245, no network), giving a millisecond-scale window, while the stale-handle window after suspend/resume is minutes-to-indefinite and produces the error on EVERY sync attempt — matching the persistent whole-webspace failure observed."
  timestamp: 2026-08-10T22:47:00+01:00

## Evidence
<!-- APPEND only - facts discovered -->

- timestamp: 2026-08-10T22:10:00+01:00
  checked: .planning/debug/knowledge-base.md + resolved sessions list
  found: "No direct KB match for 'grpc client connection is closing'. Adjacent: new-webspace-transient-service-error (G-07-1) diagnosed the SAME UI copy ('The topos service didn't respond') as a client misclassification; 07-15 fixed the webspace_not_found half (StreamMissing state). KB-001 warns a one-plugin-looking bug can be shared-path filtered by timing."
  implication: The UI copy being identical to a previously-diagnosed misclassification strongly suggests the 'service didn't respond' framing is again NOT a real outage.

- timestamp: 2026-08-10T22:15:00+01:00
  checked: kernel/correlate/correlate.go SyncSource (lines 84-137)
  found: "'match against source %q: %w' is produced ONLY at correlate.go:107, wrapping src.Match error into a per-(webspace,source) WebspaceResult.Err. Design is degradation-correct: the error skips only this source's persistence; other sources unaffected. Caller (syncer.Coordinator) records it into sync_runs."
  implication: The Match error itself never fails an HTTP request directly — it lands in sync_runs history.

- timestamp: 2026-08-10T22:20:00+01:00
  checked: kernel/httpapi/stream.go StreamHandler + sources.go aggregateSyncStatus (202-249)
  found: "StreamHandler is structurally unable to call Match (reads index only); it embeds aggregateSyncStatus(LatestSyncRunPerSource()) into the 200 response's sync object. aggregateSyncStatus: status='error' if ANY configured source's LATEST run errored — regardless of whether that source participates in the requested webspace; errParts format is source+': '+run.Error."
  implication: "errParts for instance id 'whatsapp' with run.Error='match against source \"whatsapp\": rpc error: code = Canceled desc = grpc: the client connection is closing' reproduces the user's quoted string BYTE-FOR-BYTE. The user's 'kernel error' quote is the stream response's sync.error field — the kernel answered 200 OK."

- timestamp: 2026-08-10T22:22:00+01:00
  checked: web/src/lib/format.ts streamVariant (216-222), StreamList.svelte (60-92), StreamError.svelte
  found: "streamVariant: sync.status==='error' && items.length===0 -> 'sync-failed'; StreamList renders <StreamError syncError={response.sync.error}/> for that variant; StreamError's FIXED copy is 'Couldn't load this webspace / The topos service didn't respond — check that it's running, then retry.' with syncError shown underneath. StreamError.svelte's own doc comment admits both causes (fetch failed vs sync failed with zero items) map to the same copy — a Phase 1 (01-UI-SPEC) decision."
  implication: "CONFIRMED presentation mechanism: kernel served 200; webspace had zero items; ANY-source sync error -> full-page 'service didn't respond' error. Phase 8's per-source degradation intent (health error on chip, stream keeps working) is structurally violated by this pre-existing Phase 1 branch."

- timestamp: 2026-08-10T22:35:00+01:00
  checked: grpc-go error taxonomy for the exact message
  found: "'rpc error: code = Canceled desc = grpc: the client connection is closing' is grpc-go's ErrClientConnClosing — returned ONLY for RPCs issued on (or in flight over) a ClientConn whose LOCAL Close() has been invoked. A plugin subprocess dying on its own yields codes.Unavailable ('transport is closing'/'connection refused'); a cancelled ctx yields 'context canceled'. In this kernel, ClientConn.Close happens exclusively inside goplugin.Client.Kill() — called from Host.Reconcile (replace/remove), Host.Shutdown, launch-failure cleanup, DescribePluginType's trial defer, and (via Reconcile) supervisor.SuspendInstance."
  implication: The kernel itself Kill()ed the whatsapp plugin client, and the sync path then issued Match through that same killed handle.

- timestamp: 2026-08-10T22:40:00+01:00
  checked: kernel/supervisor/supervisor.go SuspendInstance (245-295), resume closure (285-293), commitGeneration (326-330), Apply (470-614); kernel/pluginhost/host.go Reconcile (167-221), Plugin struct (44-59), Kill (116-118); kernel/syncer/coordinator.go (63-108); kernel/syncer/scheduler.go
  found: "(1) syncer.Coordinator captures its sources map ([]correlate.Source = *pluginhost.Plugin values) ONCE at construction; it has no update seam — supervisor doc comments state 'Apply replaces the *syncer.Coordinator outright'. (2) Host.Reconcile creates NEW *Plugin values on relaunch and Kill()s the old ones. (3) SuspendInstance kills the named instance via Host.Reconcile but is 'deliberately narrow ... no scheduler generation is stopped/restarted' — it does NOT rebuild the coordinator; the running scheduler generation and its coordinator keep the KILLED *Plugin handle. (4) The resume closure only calls s.host.Reconcile(s.cfg.Sources) — relaunches the instance in the HOST but again does NOT rebuild the coordinator or restart the scheduler. (5) Supervisor.Refresh/RefreshAll delegate to s.coord — the same stale coordinator — so manual refresh hits the dead handle too. (6) Only commitGeneration (reached solely from Apply, i.e. a config save) rebuilds the coordinator."
  implication: "From the moment SuspendInstance runs until the next config save (Apply) or kernel restart, EVERY sync of the suspended-then-resumed instance — scheduled tick, manual refresh, eager refresh — calls Match through a Kill()ed client and fails with exactly the observed error. Resume does not heal it."

- timestamp: 2026-08-10T22:45:00+01:00
  checked: web/src/lib/components/AddSourceModal.svelte (254-405, 597-627), QRPanel.svelte props, RelinkModal.svelte, plugins/whatsapp/link.go (already_linked path), plugins/whatsapp/match.go + plugin.go Match (193-245)
  found: "(1) Add-Source flow passes NO instance to QRPanel -> no suspension; resolveNewInstanceId blocks on id collision; submitMatch performs ONE save (source+match+allowlist). (2) RelinkModal passes instance -> WhatsAppLinkStartHandler calls SuspendInstance. (3) RelinkModal.onrelinked only refetches source health — NO config save ever follows a re-link, so nothing triggers Apply/commitGeneration and the stale coordinator persists indefinitely. (4) link.go emits already_linked then paired WITHOUT a QR for an already-paired store — the in-app affordance for test 1's 'restart and confirm it reconnects with no second QR' check. (5) whatsapp's Match is a fast local scratch-store read (no network) — a kill landing mid-Match is improbable; the handle was dead BEFORE Match was issued."
  implication: "The suspend/resume (re-link) lifecycle is the only construction that leaves a killed handle reachable by the sync path. Most probable operator flow: a re-link session (chip menu Re-link…, or the no-second-QR confirmation) suspended instance 'whatsapp'; after pairing/confirmation, resume relaunched it in the host only; the next scheduled tick (<=15m default interval) or manual refresh recorded the closing-connection error as whatsapp's latest sync_run."

- timestamp: 2026-08-10T22:50:00+01:00
  checked: kernel/supervisor/suspend_test.go
  found: "TestSuspendInstance_StopsThenResumeRestarts asserts ONLY sup.Host().Plugins() membership before/after suspend/resume — no test drives Coordinator.Refresh (or a scheduler tick) through a suspend/resume cycle. TestApply_UnrelatedSaveSucceedsWhileAnInstanceIsSuspended covers the Apply-during-suspend interaction but also never exercises the sync path."
  implication: Gate gap — the exact defect (sync path holding a dead handle across suspend/resume) is invisible to the existing suspend tests; a regression test that calls sup.Refresh after resume would have caught it.

- timestamp: 2026-08-10T22:52:00+01:00
  checked: kernel/httpapi/whatsapplink.go poll/reap/shutdown resume call sites; kernel/supervisor/supervisor.go Apply eager-refresh dispatch (595-611)
  found: "Latent siblings in the same defect family: (a) WhatsAppLinkPollHandler runs retired.resume(r.Context()) — a client disconnect mid-resume cancels the relaunch, leaving the instance absent from the host AND stale in the coordinator (reaper/Shutdown correctly use context.Background()). (b) Apply's eager resync goroutines (`go coord.Refresh(context.Background(), name)`) are untracked and uncancellable — stopScheduler never waits for them, and singleflight coalescing means a scheduler goroutine can block in group.Do on a background-ctx sync it cannot cancel; a later Apply/Shutdown killing plugins while such a sync is mid-Match produces the same closing-connection error. (c) During the suspension window itself (link session up to 5 min), scheduled ticks for the suspended instance hit the killed handle — transient variant of the same failure even if resume were fixed."
  implication: The fix should treat suspend/resume as a generation change (or make the coordinator resolve sources by name through the live host), not merely patch the resume closure.

## Resolution
<!-- OVERWRITE as understanding evolves -->

root_cause: "Two co-causes (AND): (K) The WhatsApp link-session suspend/resume lifecycle bypasses the kernel's generation contract. supervisor.SuspendInstance kills the named instance's go-plugin client (Host.Reconcile -> Plugin.Kill -> local gRPC ClientConn.Close) while deliberately leaving the scheduler generation running and the syncer.Coordinator un-rebuilt; the resume closure it returns relaunches the instance in the HOST only (a fresh *pluginhost.Plugin value) and likewise never rebuilds the coordinator or restarts the scheduler. Because the Coordinator captures its *pluginhost.Plugin handles once at construction (no update seam — the codebase's own doc comments require Apply to 'replace the *syncer.Coordinator outright' after any Reconcile), every sync path — scheduled tick, POST refresh (Supervisor.Refresh -> stale coordinator), eager resync — issues Match through the killed client from the moment of suspension onward, failing with grpc-go's ErrClientConnClosing: 'rpc error: code = Canceled desc = grpc: the client connection is closing'. Nothing heals this until the next config save (Apply -> commitGeneration) or a kernel restart — and the re-link flow never saves config (RelinkModal.onrelinked only refetches health), so after an in-app re-link/reconnect confirmation the whatsapp source's syncs fail indefinitely, recorded as its latest sync_runs row. (P) That single-source failure is escalated into an apparent total outage: kernel/httpapi aggregateSyncStatus folds ANY configured source's latest errored run into the one global sync object embedded in every webspace's 200 stream response (errParts 'source: run.Error' — the user's quoted error verbatim), and the SPA's streamVariant maps sync.status==='error' && items.length===0 to the full-page StreamError whose fixed Phase-1 copy falsely claims 'The topos service didn't respond' — so a zero-item webspace renders as a service outage because one source (participating or not) has a failing latest sync. Latent siblings noted in Evidence: poll-handler resume runs under r.Context() (client disconnect can cancel the relaunch); Apply's eager `go coord.Refresh(context.Background(), ...)` goroutines are untracked/uncancellable (mid-Match kills at a later Apply/Shutdown, and singleflight coalescing can block stopScheduler on them); scheduled ticks during the suspension window itself hit the killed handle."
fix: "(not applied — diagnose-only) Direction: (K) make suspend/resume participate in the generation lifecycle — either stop the scheduler + commitGeneration (minus/plus the instance) inside SuspendInstance and its resume closure, or give the coordinator live by-name source resolution through the host (Host.byInstance) so a reconciled host is immediately visible to the sync path; run resume on a detached context in the poll handler (match reaper/Shutdown); track or scope the eager-resync goroutines. Add the missing regression test: Refresh a source after suspend+resume (suspend_test.go currently asserts host membership only). (P) scope the stream response's sync status to the webspace's PARTICIPATING sources (correlate.ParticipatesIn) and/or stop escalating a per-source sync error with zero items into the full-page service-unreachable state — degrade to chip-level health error + empty/partial stream per Phase 8's intent; reserve the 'didn't respond' copy for genuine fetch failures (same fix shape as G-07-1's 07-15 client half)."
verification: "n/a — no fix applied this session. Live falsification path documented in reasoning_checkpoint (suspend/resume a mock instance, refresh, observe exact error; any config save heals it)."
files_changed: []
