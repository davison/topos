---
status: diagnosed
trigger: "After linking using the QR code, phone shows successful link. WhatsApp modal in topos remains on screen with the refresh counter dwindling. No connection from the topos side after cancelling the dialog"
created: 2026-08-10T00:00:00Z
updated: 2026-08-10T00:00:00Z
---

## Current Focus

bug_class: Bohrbug (deterministic — reproduces on every real-device link; the
  "intermittent" appearance is only the variable post-pair login latency
  interacting with a fixed poll cadence)

reasoning_checkpoint:
  hypothesis: >
    The `paired` terminal event IS produced by the plugin and IS recorded by the
    kernel session, but it cannot reach the browser for up to `expires_in_seconds`
    (60s on whatsmeow's FIRST QR code) because QRPanel derives its poll interval
    from the QR code's own validity window. During that window the panel re-renders
    the same stale QR and restarts its countdown, and no wire state exists to say
    "scan accepted, completing login" — so the panel is visually indistinguishable
    from a dead session. The user cancels inside that window; cancel SIGKILLs the
    subprocess and the Add-Source flow persists no instance, so nothing connects.
  confirming_evidence:
    - "QRPanel.svelte:128 — `schedulePoll(seconds * 1000)` inside `case 'qr'`: the poll interval IS the QR validity window."
    - "whatsmeow qrchan.go:88-90 — first code (`len(codes) == 6`) carries `timeout = 60 * time.Second`; subsequent codes 20s."
    - "plugins/whatsapp/link.go:28,35 — `paired` is emitted only after `postPairLoginTimeout` (up to 60s) AND an unconditional `postPairGraceWindow` 5s sleep."
    - "link.go:331-341 — `pairingAccepted()` writes to STDERR only; `loggedIn()` is a deliberate no-op. There is no `pairing_accepted` member of `linkEventKind` (qr/paired/error/timeout only)."
    - "kernel/httpapi/whatsapplink.go:59-108 — `execLinkSpawner` never sets `cmd.Stderr`, so os/exec routes the subprocess's stderr to /dev/null. Every plugin diagnostic is destroyed."
    - "whatsmeow qrchan.go:185-190 — PairSuccess calls `close(qrc.stopQRs)` and closes the output channel: after a scan NO further qr events are ever produced, so the kernel's `latest` freezes on the last (already-scanned, already-invalid) code."
    - "QRPanel.svelte:98-104 — `startCountdown` re-runs on EVERY poll response, so an unchanged stale qr event resets the countdown to its full value; `Math.max(0, ...)` floors it at 0:00 and `phase` stays 'qr' forever (only a `timeout` wire event yields 'expired', and that can never arrive post-pair)."
  falsification_test: >
    Instrument the kernel session: if a real-device run shows the session reaching
    terminal `paired` promptly while the browser's next GET is still tens of seconds
    away, the hypothesis holds. It is falsified if the session never reaches `paired`
    (i.e. the plugin genuinely never emits it).
  fix_rationale: >
    Decouple liveness polling from QR validity (poll on a fixed short cadence, e.g.
    the existing POLL_FLOOR_MS), and add an intermediate `pairing_accepted` wire state
    driven by the plugin's already-existing `pairingAccepted()` hook so the panel can
    show "Scan accepted — completing login…" during the plugin's own 5-65s post-pair
    window. Both are needed (see and_gate).
  blind_spots: >
    Could not run a real-device link (needs a live WhatsApp account). The exact
    number of seconds the user waited before cancelling is unknown, so the precise
    point inside the blind window is inferred, not observed. The kernel discards the
    subprocess's stderr, so no log from the failing run exists to confirm directly.
  candidate_causes:
    - "code (web UI): poll interval tied to expires_in_seconds — QRPanel.svelte:128"
    - "code (plugin/protocol): no intermediate progress state in linkEventKind; pairingAccepted() goes to stderr — link.go:223-228, 331-341"
    - "code (kernel): subprocess stderr discarded (cmd.Stderr never set) — whatsapplink.go:59-108"
    - "process (test gate): e2e success case scripts `paired` into the START response, never exercising qr -> paired"
    - "environment/data: RULED OUT — QR encoded correctly, phone paired, whatsmeow persisted the device row"
  and_gate: >
    YES — this needs more than one condition simultaneously. Shortening the poll
    cadence alone still leaves a >=5s silent window with a stale QR on screen.
    Adding a progress state alone still leaves that state undelivered for up to 60s.
    The panel only looks permanently dead when a long blind poll window AND the
    absence of any progress signal coincide.

## Symptoms

expected: A real WhatsApp account links via the in-app QR flow — after the phone scans and accepts the pairing, the QRPanel in the Add Source / Re-link modal transitions to its success state, and the WhatsApp source subsequently connects (serve-mode instance resumes and reports healthy/linked).
actual: After linking using the QR code, phone shows successful link. WhatsApp modal in topos remains on screen with the refresh counter dwindling. No connection from the topos side after cancelling the dialog.
errors: None reported
reproduction: UAT Test 1, phase 08. Add Source -> WhatsApp -> QR panel -> scan with phone -> phone reports linked -> topos panel keeps showing QR with the expiry countdown still ticking. After Cancel, the source does not connect.
started: Discovered during Phase 8 UAT (2026-08-10). Terminal ASCII `-link` flow verified working live during plan 08-01. Failure is in the NEW in-app flow (kernel link-session endpoints, plugin `-link-json` mode, QRPanel polling) which was only ever tested hermetically with route-layer mocks.

## Eliminated

- hypothesis: "The kernel loses the terminal `paired` line because execLinkSpawner calls cmd.Wait() concurrently with reading the stdout pipe (os/exec documents this as incorrect usage)."
  evidence: >
    Built a fake plugin binary emitting 3 large qr lines (3 KB payloads) then
    `{"kind":"paired"}` then exit 0, and drove the REAL execLinkSpawner +
    linkSession.consume 200 times. RESULT: 0/200 runs lost the terminal paired
    event. The Wait/read race is real per the os/exec contract but does not
    manifest for this write-then-exit pattern.
  timestamp: phase-3

- hypothesis: "The plugin never emits `paired` because the post-pair path is broken."
  evidence: >
    runLinkCore is SHARED byte-for-byte between `-link` (ASCII) and `-link-json`;
    only the emitter differs. The ASCII mode was verified working live on the same
    day, same whatsmeow pin (v0.0.0-20260806224404-e277b766ab33), and reported
    "Linked successfully." Confirmed in whatsmeow pair.go:211-215 that
    `cli.Store.Save(ctx)` persists the device BEFORE PairSuccess is dispatched, and
    connectionevents.go:204 dispatches `events.Connected` after login — the exact
    signal pairwait.go waits on. The core path is sound.
  timestamp: phase-3

- hypothesis: "A malformed/interleaved log line on stdout is decoded as a bogus event."
  evidence: >
    connect.go:38-40 — pluginLogger writes exclusively to os.Stderr, never stdout.
    Had it hit stdout, consume() would have synthesized a terminal `error` event and
    the panel would show the error state, which is not the reported symptom.
  timestamp: phase-1

- hypothesis: "Store-lock contention (a running serve-mode instance) blocked the link subprocess."
  evidence: >
    acquireStoreLock failure surfaces as a terminal `error` event with code
    `store_in_use`, mapped by the kernel to `whatsapp_store_in_use` and rendered by
    the panel's error branch. The user saw a QR and a countdown, i.e. the subprocess
    got past the lock and was streaming qr events.
  timestamp: phase-1

## Evidence

- timestamp: phase-0
  checked: .planning/debug/knowledge-base.md
  found: No keyword match on QR/link-session/pairing. KB-001/KB-002 both concern the sync_runs table.
  implication: Novel defect class for this codebase; no prior pattern to reuse.

- timestamp: phase-1
  checked: web/src/lib/components/QRPanel.svelte applySession()
  found: "case 'qr' calls startCountdown(seconds) then schedulePoll(seconds * 1000) — the liveness poll interval IS the QR code's validity window."
  implication: "The terminal `paired` state can sit undelivered in the kernel for a full validity window before the browser asks for it."

- timestamp: phase-1
  checked: whatsmeow qrchan.go emitQRs (module cache, pinned version)
  found: "timeout := 20 * time.Second; if len(codes) == 6 { timeout = 60 * time.Second } — the FIRST code, the one a user actually scans, carries a 60-second window."
  implication: "The blind window on the first (most-likely-scanned) code is 60 seconds, not 20."

- timestamp: phase-1
  checked: whatsmeow qrchan.go handleEvent, PairSuccess branch
  found: "close(qrc.stopQRs); qrc.close(); output <- QRChannelSuccess; close(qrc.output) — the QR emitter is stopped and the channel closed at PairSuccess."
  implication: "After a successful scan NO further qr events exist. The kernel session's `latest` freezes on the already-consumed code, so every subsequent poll returns a byte-identical stale qr event."

- timestamp: phase-1
  checked: QRPanel.svelte startCountdown / phase transitions
  found: "startCountdown re-runs unconditionally on every poll response; remainingSeconds is floored at 0 via Math.max(0, ...); `phase` only leaves 'qr' on a paired/error/timeout wire state."
  implication: "A frozen stale qr event makes the countdown either sit at 0:00 or restart at its full value on each poll — exactly the reported 'refresh counter dwindling' with an image that never actually refreshes. The panel can never self-diagnose as expired, because whatsmeow will never send `timeout` after a successful pair."

- timestamp: phase-1
  checked: plugins/whatsapp/link.go constants and jsonLinkEmitter
  found: "postPairLoginTimeout = 60s, postPairGraceWindow = 5s (unconditional time.Sleep). jsonLinkEmitter.pairingAccepted() writes only to stderr; loggedIn() is an explicit no-op. linkEventKind = {qr, paired, error, timeout} — no progress kind."
  implication: "Even a perfect poller sees >=5s of silence after the phone reports success, and up to 65s in the worst case, with zero wire signal that anything is happening."

- timestamp: phase-1
  checked: kernel/httpapi/whatsapplink.go execLinkSpawner
  found: "cmd.StdoutPipe() is wired, but cmd.Stderr is never assigned — os/exec therefore connects the child's stderr to os.DevNull. Separately, the comment above `cmd.Env = nil` claims 'an explicit, minimal environment', but in os/exec a nil Env means INHERIT the parent's environment."
  implication: "Every plugin diagnostic (including 'pairing accepted — completing login…' and all whatsmeow WARN/ERROR output) is destroyed. This is why the UAT reports 'errors: None' — there was nowhere for an error to appear. The Env comment is a separate, non-causal correctness bug in the same function."

- timestamp: phase-3
  checked: EXPERIMENT — real execLinkSpawner + linkSession.consume, 200 runs against a fake binary emitting qr lines then `paired` then exit 0
  found: "RESULT: 0/200 runs lost the terminal paired event."
  implication: "Kernel-side delivery of the terminal event is sound. The defect is not in the kernel plumbing; it is in when the browser asks and what the wire vocabulary can express."

- timestamp: phase-3
  checked: kernel/httpapi/whatsapplink_test.go
  found: "Seven tests, all injecting a fake linkSpawner. `execLinkSpawner` — the production spawner — is referenced only from routes.go:106 and is never executed by any test."
  implication: "The kernel/plugin process boundary (argv, env, stderr, pipe lifecycle) has no automated coverage at all."

- timestamp: phase-3
  checked: web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts test 3 ("Scan success")
  found: "const pairedResponse = {... state: 'paired'}; scriptLinkSession(page, { start: pairedResponse, polls: [pairedResponse] }) — the START response itself answers `paired`, so the panel reaches success without ever polling."
  implication: "The qr -> paired transition — the ONLY sequence a real device produces — is never exercised. An in-code comment makes this explicit: 'every other case starts already in its target state so its assertions are not incidentally timing-dependent on that floor.' The suite was deliberately written to avoid the very latency that broke UAT."

- timestamp: phase-3
  checked: web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts test 2 ("Rotation")
  found: "The one test that does exercise the poll loop scripts expires_in_seconds: 1, which the panel floors to POLL_FLOOR_MS (2s)."
  implication: "The only poll-loop coverage uses a value ~60x smaller than the real first-code window, so the cadence defect is structurally invisible to the suite."

- timestamp: phase-3
  checked: docs/api.md, POST/GET/DELETE /api/config/whatsapp-link section
  found: "'the browser ... polls GET .../whatsapp-link/{session} on its own cadence (driven by the rotating QR code's own expires_in_seconds) until the session reaches a terminal state'."
  implication: "The defect is specified, not merely implemented — the documented contract prescribes the broken cadence. A fix must amend docs/api.md too, or the next implementer reintroduces it."

- timestamp: phase-3
  checked: AddSourceModal.svelte handleLinkCancelled / resetFlowState + whatsmeow pair.go Store.Save ordering
  found: "handleLinkCancelled sets step = 'connect'; closing the dialog runs resetFlowState(), persisting nothing. whatsmeow writes cli.Store.ID and calls cli.Store.Save(ctx) BEFORE dispatching PairSuccess."
  implication: "Second half of the symptom explained: the pairing genuinely succeeded and the device row is on disk in whatsmeow.db, but no [sources.*] instance was ever written to config.toml, so the kernel launches no WhatsApp instance and nothing connects. A real, completed pairing is stranded with no UI affordance that it happened."

- timestamp: phase-3
  checked: kernel/httpapi/whatsapplink.go WhatsAppLinkCancelHandler -> sess.kill -> exec.CommandContext cancel
  found: "CommandContext's default Cancel is Process.Kill() — SIGKILL."
  implication: "Cancelling during the post-pair window SIGKILLs the subprocess mid login handshake — the same 'premature disconnect' class pairwait.go's own doc comment says Plan 08-01 fixed, reintroduced here through the cancel path rather than the code path."

## Resolution

root_cause: >
  Three contributing conditions that only produce this failure together (AND-gate confirmed):
  (1) QRPanel.svelte:128 sets its liveness poll interval to the QR code's own validity
  window (`schedulePoll(seconds * 1000)`), which whatsmeow reports as 60s for the first
  code — so the terminal `paired` event, though recorded by the kernel promptly and
  correctly, is not fetched by the browser for up to 60 seconds after the phone reports
  success; (2) the link wire vocabulary (`linkEventKind` = qr|paired|error|timeout) has no
  intermediate progress state, and the plugin's existing `pairingAccepted()` hook writes
  only to stderr, which `execLinkSpawner` discards by never assigning `cmd.Stderr` — so
  during the plugin's own 5-65s post-pair login window (`postPairLoginTimeout` +
  `postPairGraceWindow`) neither the UI nor the operator log has any signal at all;
  (3) whatsmeow stops the QR emitter at PairSuccess, freezing the kernel's `latest` on the
  already-scanned code, while `applySession` restarts `startCountdown` on every poll and
  `phase` can only leave 'qr' via a terminal wire state — so the panel renders a stale QR
  with a countdown that either sits at 0:00 or resets to full and dwindles again,
  indistinguishable from a dead session. The user cancels inside that blind window; cancel
  SIGKILLs the subprocess and the Add-Source flow persists no `[sources.*]` instance, so
  the (genuinely successful, on-disk) pairing produces no connection.
fix: "" # diagnose-only mode (goal: find_root_cause_only)
verification: "" # diagnose-only mode
files_changed: []
