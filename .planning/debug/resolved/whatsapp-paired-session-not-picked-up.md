---
status: resolved
trigger: "G-08-4 — after a successful real-device WhatsApp pairing, the running whatsapp source instance still answers Match with its pre-pairing 'Not linked' health error"
created: 2026-08-11T00:50:00Z
updated: 2026-08-11T01:40:00Z
mode: find_root_cause_only
gap: G-08-4
---

## Current Focus

bug_class: Bohrbug (deterministic ordering defect — NOT a flaky race; the kernel's
  post-launch Match beats the WhatsApp login RTT by ~3 orders of magnitude)

hypothesis: |
  The fresh session IS picked up. The relaunched serve-mode instance reads the paired
  device row fine. The defect is that plugins/whatsapp/connect.go's success path
  (device.ID != nil && client.Connect() == nil) calls setHealthState with NOTHING —
  so p.state keeps its Go ZERO VALUE, which health.go deliberately defines as
  healthStateNotLinked. The plugin therefore reports the literal, factually FALSE
  "Not linked — pair this device…" message during the entire post-Connect() /
  pre-*events.Connected login window, and Match returns codes.Unavailable for it.
  supervisor.commitGeneration -> startScheduler fires that source's eager refresh
  within milliseconds of Reconcile returning, so the very first Match after every
  (re)launch lands inside that window and persists an errored sync_runs row.

test: |
  1. Prove the message identity: a SourcePlugin that has not observed *events.Connected
     emits EXACTLY the string the user reported, with no parenthetical detail.
  2. Prove the ordering: Connect() returns after the noise handshake, *events.Connected
     only after the server's <success> node — and NewSourcePlugin runs BEFORE
     goplugin.Serve, so go-plugin's handshake (and therefore Reconcile's return)
     completes strictly inside that window.

expecting: |
  Confirmed if (1) the string matches byte-for-byte with detail == "" and (2) no code
  path sets healthStateLinked other than eventhandler.go's *events.Connected case.

next_action: CONFIRMED — root cause established; hand back to /gsd-plan-phase --gaps (diagnose-only mode, no fix applied)

reasoning_checkpoint:
  hypothesis: |
    plugins/whatsapp has no "linked but still logging in" health state. connect.go's
    success path (device.ID != nil, client.Connect() returns nil) sets NO state at all,
    leaving p.state at its Go zero value — which health.go defines as
    healthStateNotLinked. Only *events.Connected (eventhandler.go) ever sets
    healthStateLinked. Because main.go runs NewSourcePlugin BEFORE goplugin.Serve, the
    kernel's go-plugin handshake completes at the moment Connect() returns — i.e.
    strictly INSIDE that window — and supervisor.commitGeneration -> startScheduler
    fires that source's eager Match within milliseconds. The first Match after every
    (re)launch therefore reads a NotLinked state and returns codes.Unavailable carrying
    the factually false "Not linked — pair this device with WhatsApp" message, which is
    persisted as the source's latest sync_runs row and rendered by the stream banner.
  confirming_evidence:
    - "E-04: a zero-value *SourcePlugin emits byte-for-byte the reported gRPC message (empirically run, not inferred)"
    - "E-02: grep proves exactly one assignment of healthStateLinked, in eventhandler.go's *events.Connected case"
    - "E-06: whatsmeow Connect() returns after doHandshake; events.Connected is dispatched only from handleConnectSuccess (server <success> node) — a separate network round trip"
    - "E-07: main.go calls NewSourcePlugin before goplugin.Serve, so Reconcile returns inside the window"
    - "E-08: commitGeneration -> startScheduler -> runSource refreshes immediately (scheduler.go:74)"
    - "E-11: the symptom appeared exactly when 08-09 turned resume into a generation change that fires an eager refresh — a trigger that did not exist before that wave"
  falsification_test: |
    Launch the serve-mode plugin against an already-paired store and call Health/Match in a
    tight loop from launch. If the first responses were healthy — or carried the
    "(initial connect failed, retrying: …)" detail rather than the bare template — this
    hypothesis is wrong and the "no device row in the store the instance reads" branch
    would be back in play.
  fix_rationale: |
    Addresses the root cause, not the symptom: the plugin currently cannot distinguish
    "never paired" from "paired, socket up, awaiting login", and the kernel has no
    readiness gate between launching a plugin and issuing its first Match. Fixing either
    one alone leaves a wrong-but-differently-worded error; fixing the plugin's state
    machine AND removing the guaranteed-losing race is what makes the first post-pairing
    sync honest.
  blind_spots: |
    No real-device run was possible in this worktree — the ordering argument is built
    from source reading of whatsmeow + the kernel, plus an empirical message-identity
    test. Not directly observed: the actual wall-clock gap between Reconcile returning
    and *events.Connected on the user's network; whether the user's specific run was a
    re-link (resume path) or an Add-Source (config-save/Apply path) — both funnel into
    the same eager-refresh-after-launch window, so the diagnosis holds either way.
  candidate_causes:
    - "code (plugin): no healthStateConnecting; connect.go's success path sets no state, so the zero value NotLinked is reported for a device that IS linked"
    - "code (kernel): supervisor.commitGeneration -> startScheduler issues a source's first Match with no plugin-readiness gate — at boot, on every Apply, and (new in 08-09) on every link suspend/resume"
    - "config: DefaultSyncInterval = 15m — decides how long the wrong error stays the latest sync_runs row"
    - "environment: real WhatsApp login RTT is non-zero; the hermetic mock plugin's is exactly zero, which is why every automated gate passes"
  and_gate: |
    YES — multiple conditions must hold simultaneously. (1) the plugin reports non-healthy
    during its normal login window AND (2) the kernel issues a Match inside that window
    AND (3) nothing re-syncs before the user looks (15m default) AND (4) the stream banner
    reads the persisted sync_runs error rather than the live health probe. (1)+(2) create
    the false error; (3)+(4) are what make it "pinned" rather than a blink. root_cause is
    therefore recorded as a set, not a single line.

## Symptoms

expected: |
  Immediately after a successful real-device WhatsApp pairing, the running whatsapp source
  instance uses the fresh session: the source syncs (or reports a transient connecting state)
  rather than staying pinned on the pre-pairing "Not linked" health error.

actual: |
  After login and opening the webspace:
    "A source couldn't sync / Nothing to show here yet. Your other sources are unaffected —
     check the source chips above, then retry."
    whatsapp: match against source "whatsapp": rpc error: code = Unavailable desc =
    whatsapp: Not linked — pair this device with WhatsApp to start syncing. Use this source's
    chip menu ("Re-link…") or run this plugin binary's -link flag.

errors: |
  rpc error: code = Unavailable desc = whatsapp: Not linked — pair this device with WhatsApp
  to start syncing. (the plugin's own health-gated Unavailable error, surfaced through Match)

reproduction: |
  Test 1 in .planning/phases/08-whatsapp-conversations-managed-risk/08-UAT.md — real device:
  pair/re-link a WhatsApp account via `make dev`, then open its webspace immediately after.

started: |
  Discovered during the UAT re-run on 2026-08-11, immediately after the G-08-3 gap-closure
  wave (plans 08-09, 08-10) merged. The presentation half of G-08-3 is CONFIRMED FIXED —
  the webspace rendered the per-source StreamSyncDegraded notice with correct copy. What is
  broken is upstream: after a real pairing completed, the running whatsapp instance still
  answers Match with its "Not linked" health error.

## Eliminated

- hypothesis: "The link subprocess writes the fresh session to a store the resumed instance does not read (different path / different DB file)"
  evidence: |
    Both sides derive the same directory from the SAME config value.
    web/src/lib/components/RelinkModal.svelte:48 passes `path = source?.path` straight from
    the running config into POST /api/config/whatsapp-link. The link subprocess is exec'd as
    `binPath -link-json -path <that value>` (whatsapplink.go:146) and expands it via
    main.go's expandHome, then opens filepath.Join(dir, "whatsmeow.db") through
    whatsmeowSessionDSN (link.go:86-87). The serve instance receives the same config value in
    WEBSPACES_SOURCE_CONFIG, expands it with the SAME expandHome, and opens the SAME
    filepath.Join(dir, "whatsmeow.db") through the SAME whatsmeowSessionDSN
    (connect.go:81-82). One config value, one expansion function, one join, one DSN builder.
  timestamp: 2026-08-11T01:05:00Z

- hypothesis: "Ordering: resume fires before the link subprocess has flushed/closed the store, so the relaunched instance opens a pre-session store or loses the flock race"
  evidence: |
    runLinkJSON emits the terminal `paired` event only AFTER runLinkCore has RETURNED
    (link.go:393-396), and runLinkCore's deferred cleanup runs before that return in LIFO
    order: client.Disconnect() -> container.Close() -> lock.Release() (link.go:84/91/123,141).
    storelock.go:60-67 shows Release() does a real syscall.Flock(LOCK_UN) + Close(). So the
    sqlite container is closed and the exclusive flock is released BEFORE the kernel can even
    observe the `paired` line — the poll handler's retire/kill/resume sequence
    (whatsapplink.go:721-735) cannot race it. Had this been the failure, the QR panel would
    have surfaced whatsapp_store_in_use, which the user did not report.
  timestamp: 2026-08-11T01:08:00Z

- hypothesis: "connect.go's transient-dial-failure branch produced the message (initial Connect() failed)"
  evidence: |
    That branch sets a non-empty detail — `initial connect failed, retrying: %v`
    (connect.go:124) — and currentMessage() appends any non-empty detail in parentheses
    (plugin.go:161-167). Empirically reproduced: that branch renders
    "…-link flag. (initial connect failed, retrying: dial tcp: i/o timeout)". The user's
    reported message has NO parenthetical, so p.detail was "". ELIMINATED.
  timestamp: 2026-08-11T01:22:00Z

- hypothesis: "resume()'s Reconcile failed, so the instance was never relaunched and the UI is showing the stale PRE-pairing error row"
  evidence: |
    Possible in principle (resume's error is only logged at Warn, whatsapplink.go:730-732),
    but it requires the relaunch to lose the store-lock race, which the second elimination
    above rules out. It also predicts the whatsapp source vanishing from the chip row
    entirely (a failed resume leaves the instance in neither host nor s.suspended —
    supervisor.go:387-397), which contradicts the reported UI. Note that this variant does
    not change the diagnosis materially: both it and the confirmed cause leave the same
    "Not linked" text as the latest sync_runs row, and the confirmed cause explains it
    without requiring an unobserved secondary failure.
  timestamp: 2026-08-11T01:30:00Z

## Evidence

- timestamp: 2026-08-11T00:58:00Z
  checked: "plugins/whatsapp/health.go — the healthState taxonomy"
  found: |
    Six states. healthStateNotLinked is `iota` == 0, i.e. the Go ZERO VALUE, and is
    explicitly documented as "this plugin's safe zero-value default (never silently reports
    linked)". healthStateLinked is the ONLY state whose Healthy() returns true (health.go:52).
    There is NO state meaning "linked, socket up, login handshake in progress".
  implication: |
    Any *SourcePlugin whose state has not been explicitly assigned reports "Not linked",
    regardless of whether a device is actually paired. The "safe default" is safe against
    false-healthy but produces a false-NEGATIVE that is user-visible and actionable-sounding.

- timestamp: 2026-08-11T00:59:00Z
  checked: "grep -rn healthStateLinked across plugins/whatsapp (non-test)"
  found: |
    Exactly ONE assignment: eventhandler.go:28, `case *events.Connected:
    p.setHealthState(healthStateLinked, "")`.
  implication: |
    The plugin can only ever become healthy by observing a whatsmeow *events.Connected.
    Nothing about reading the paired device row, and nothing about Connect() succeeding,
    makes it healthy.

- timestamp: 2026-08-11T01:00:00Z
  checked: "plugins/whatsapp/connect.go — startBackgroundClient's three exits"
  found: |
    (a) device.ID == nil            -> setHealthState(healthStateNotLinked, "")            [line 98]
    (b) Connect() returns an error  -> setHealthState(healthStateNotLinked, "initial connect
                                       failed, retrying: %v")                              [line 124]
    (c) device.ID != nil AND Connect() returns nil -> NO setHealthState CALL AT ALL; the
        function just `return nil`s.                                                       [line 128]
  implication: |
    Path (c) — the NORMAL, SUCCESSFUL path for a freshly paired device — leaves p.state at
    the zero value healthStateNotLinked with p.detail == "". This is the exact shape of the
    user's reported message. The doc comment at connect.go:113-121 shows the author was aware
    the state only flips on a later *events.Connected, but treated the window as invisible.

- timestamp: 2026-08-11T01:22:00Z
  checked: |
    EXPERIMENT (scratchpad copy of plugins/whatsapp, repo untouched):
    `p := &SourcePlugin{}` — exactly what path (c) leaves behind — then call Match and
    compare the gRPC status message byte-for-byte against the user's verbatim report.
  found: |
    PASS. code=Unavailable message="whatsapp: Not linked — pair this device with WhatsApp to
    start syncing. Use this source's chip menu (\"Re-link…\") or run this plugin binary's
    -link flag."  — identical, including the em dash and the curly quotes, with no
    parenthetical detail.
  implication: |
    DIRECT observation, not inference: the state the plugin is in immediately after a
    successful launch against a paired store produces precisely the reported error. Combined
    with the elimination of branch (b), only two producers remain — (a) "no device row" and
    (c) "device row, connected, awaiting login" — and (a) is ruled out by the path-identity
    elimination above plus phase 08's own recorded real-device spikes (08-01-SUMMARY.md),
    which prove serve mode does read the paired device and does capture history.

- timestamp: 2026-08-11T01:10:00Z
  checked: "plugins/whatsapp/main.go — startup order"
  found: |
    main() calls NewSourcePlugin(...) at line 135 and goplugin.Serve(...) at line 142.
    NewSourcePlugin -> startBackgroundClient -> client.Connect().
  implication: |
    The go-plugin handshake line is only written AFTER Connect() has returned. So the moment
    the kernel considers the plugin "launched and ready" is, by construction, the first
    instant of the not-yet-Connected window — not a moment before it and not a moment after.

- timestamp: 2026-08-11T01:12:00Z
  checked: |
    go.mau.fi/whatsmeow@v0.0.0-20260806224404-e277b766ab33 — client.go / connectionevents.go
  found: |
    unlockedConnect (client.go:546-581) returns immediately after fs.Connect(ctx) +
    cli.doHandshake(...) — the websocket dial and NOISE handshake only. *events.Connected is
    dispatched from exactly one place, connectionevents.go:204, inside handleConnectSuccess
    (line 158), which runs on receipt of the server's <success> stream node — a further
    network round trip and server-side authentication after Connect() has already returned.
  implication: |
    The window between "Connect() returned" and "*events.Connected" is a real WhatsApp-server
    round trip: hundreds of milliseconds on a normal connection, and unbounded on a slow one.

- timestamp: 2026-08-11T01:14:00Z
  checked: "kernel/syncer/scheduler.go + kernel/supervisor/supervisor.go's commitGeneration"
  found: |
    commitGeneration (supervisor.go:434-438) does newCoordinator -> s.cfg = cfg ->
    startScheduler(cfg). startScheduler spawns Scheduler.Run, whose runSource
    (scheduler.go:73-74) calls refreshAndLog IMMEDIATELY, before creating its ticker. There
    is no readiness probe anywhere between pluginhost launch and the first Match —
    Host.Reconcile (host.go:167-221) launches and commits, full stop.
  implication: |
    The kernel's first Match after any (re)launch is issued microseconds-to-milliseconds
    after the handshake completes. Against a window measured in hundreds of milliseconds,
    the kernel wins essentially 100% of the time. This is a DETERMINISTIC ordering defect
    (Bohrbug), not a flaky race.

- timestamp: 2026-08-11T01:16:00Z
  checked: "kernel/correlate/correlate.go:107 — the error-wrapping site"
  found: |
    `wrapped := fmt.Errorf("match against source %q: %w", src.Name(), err)` — reproduces the
    user's message prefix exactly: `match against source "whatsapp": rpc error: code =
    Unavailable desc = whatsapp: Not linked …`.
  implication: |
    The reported text is a SYNC-RUN error (persisted), not a live health probe — it came
    through Coordinator.Refresh -> Engine.SyncSource -> Match, i.e. through the scheduler's
    eager refresh, exactly as predicted.

- timestamp: 2026-08-11T01:18:00Z
  checked: "kernel/httpapi/stream.go:95-97 and kernel/httpapi/sources.go:88-113"
  found: |
    The stream banner's `sync` object is built from store.LatestSyncRunPerSource(ctx) — the
    PERSISTED latest sync_runs row per source. The source CHIP, by contrast, gets its
    Reachable flag from a LIVE prober.ProbeSources(ctx) but takes LastError from the same
    persisted row.
  implication: |
    Once the racing eager refresh writes an errored row, the banner shows it until a LATER
    run supersedes it. The chip's live Reachable can flip green (the plugin does connect a
    second later) while the banner still says "Not linked" — the same contradictory-UI
    signature as KB-002, on a different seam.

- timestamp: 2026-08-11T01:19:00Z
  checked: "kernel/config — sync interval default"
  found: "DefaultSyncInterval = \"15m\" (kernel/config/types.go:252)."
  implication: |
    After the racing first refresh fails, nothing retries for 15 minutes. That is what turns
    a sub-second lifecycle artifact into the "pinned" state the gap's truth statement names.
    The user's only escapes are the chip's manual refresh or waiting out the interval.

- timestamp: 2026-08-11T01:26:00Z
  checked: "kernel/supervisor/supervisor.go SuspendInstance doc comment + 08-09-SUMMARY.md"
  found: |
    08-09 (the G-08-3 fix) deliberately converted suspend AND resume into full generation
    changes: stopScheduler -> Host.Reconcile -> commitGeneration. Its own doc comment says
    "Each suspend and each resume restarts the scheduler generation, and Scheduler.Run fires
    every configured source's first refresh immediately by its own existing design — so a
    link session costs two eager full refreshes … This is accepted, not overlooked."
  implication: |
    This is the newly-introduced trigger, and it explains the timing of G-08-4's appearance
    precisely. Before 08-09 the resume path never rebuilt the coordinator, so no eager Match
    followed a re-link (that WAS G-08-3 — the source simply never synced again). 08-09
    correctly made resume relaunch-and-resync, which for the first time routed a Match into
    the plugin's login window. The same window is also hit at kernel boot (NewSupervisor,
    supervisor.go:106-118) and on every Apply — this is not unique to the re-link path.

- timestamp: 2026-08-11T01:32:00Z
  checked: "plugins/mock/plugin.go — the plugin every hermetic supervisor/httpapi test drives"
  found: |
    The mock plugin's Match (plugin.go:201-210) is unconditionally successful: it has no
    health state, no connection, and no asynchronous readiness of any kind. Its readiness
    window is exactly zero.
  implication: |
    GATE GAP. TestSuspendInstance_ResumedInstanceStillSyncs proves the coordinator holds a
    live handle after a resume, but it structurally CANNOT surface "the relaunched plugin is
    not ready yet" — the fixture is ready the instant it is launched. Every automated gate in
    this phase shares that blind spot, which is why 08-09/08-10 went green and the real
    device did not.

- timestamp: 2026-08-11T01:35:00Z
  checked: "plugins/whatsapp/pairwait.go vs plugins/whatsapp/connect.go"
  found: |
    The LINK flow already solves exactly this problem: pairLoginWaiter blocks on a real
    *events.Connected (with a 60s postPairLoginTimeout) before ever declaring success,
    precisely because "whatsmeow persists Store.ID at PairSuccess time … BEFORE the post-pair
    websocket reconnection and login handshake completes". The SERVE flow
    (startBackgroundClient) has no equivalent and no intermediate state.
  implication: |
    The correct primitive already exists in this package and is already proven against a real
    device. The defect is that serve mode does not use it, and that the health taxonomy has
    no state to express what it would be waiting for.

## Resolution

root_cause: |
  (AND-gated — two contributing causes must hold simultaneously; the third and fourth
  determine how long the user sees it.)

  1. PLUGIN (primary). plugins/whatsapp has no "linked, connecting" health state, and
     connect.go's successful path (device.ID != nil && client.Connect() == nil) sets no
     health state at all — leaving p.state at the Go zero value, which health.go defines as
     healthStateNotLinked with an empty detail. healthStateLinked is assigned in exactly one
     place, eventhandler.go's *events.Connected case. A correctly paired, actively connecting
     instance therefore reports, verbatim, "Not linked — pair this device with WhatsApp to
     start syncing" and Match returns codes.Unavailable for it.

  2. KERNEL (co-cause). There is no plugin-readiness gate between launching a subprocess and
     issuing its first Match. supervisor.commitGeneration calls startScheduler, and
     Scheduler.runSource refreshes immediately. Because plugins/whatsapp/main.go runs
     NewSourcePlugin BEFORE goplugin.Serve, the handshake that tells the kernel "ready"
     completes at the exact instant Connect() returns — the FIRST instant of the
     not-yet-Connected window. The kernel's Match arrives milliseconds later; the WhatsApp
     login round trip takes hundreds. The kernel wins deterministically.

  3. The errored sync_runs row this writes is what the stream banner renders
     (LatestSyncRunPerSource), and DefaultSyncInterval is 15m, so nothing supersedes it for
     15 minutes — turning a sub-second lifecycle artifact into a pinned, false, and
     actionable-sounding instruction to go and pair a device that is already paired.

  Trigger timing: 08-09 (the G-08-3 fix) correctly made suspend/resume a generation change,
  which for the first time fires an eager refresh immediately after the post-pairing
  relaunch — routing a Match into the plugin's login window on the very flow the UAT tests.
  The same window is also hit at kernel boot and on every config Apply.

fix: "NOT APPLIED — diagnose-only mode (goal: find_root_cause_only). See ROOT CAUSE FOUND return."
verification: "n/a — no fix applied in this session"
files_changed: []
