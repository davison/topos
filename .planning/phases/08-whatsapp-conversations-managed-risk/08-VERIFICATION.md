---
phase: 08-whatsapp-conversations-managed-risk
verified: 2026-08-10T18:05:00Z
status: passed
score: 5/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 4/5
  gaps_closed:
    - "A running (linked) WhatsApp instance's match configuration can be viewed and edited through the existing 'Edit match settings…' chip menu entry, and an already-configured WhatsApp instance can be added to a second webspace via the '+' picker's existing-instance flow"
  gaps_remaining: []
  regressions: []
---

# Phase 8: WhatsApp Conversations (Managed Risk) Verification Report

**Phase Goal:** User's WhatsApp groups for a topic appear in the webspace stream via a linked-device session, and everything else keeps working when that session breaks
**Verified:** 2026-08-10T18:05:00Z
**Status:** passed
**Re-verification:** Yes — after gap closure (CR-01 fix, commit `7cb51d1`)

## Goal Achievement

### Observable Truths

Roadmap Success Criteria (`.planning/ROADMAP.md`, Phase 8) plus the phase-level must-have that was previously failing:

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | User can link WhatsApp as a device by scanning a QR code, and the session survives service restarts without re-linking | ✓ VERIFIED (regression check — unchanged since prior verification) | No code under this truth's path (`link.go`, `connect.go`, `pairwait.go`) was touched by any of the three fix commits (`git diff --stat` confirms the 9-file diff is limited to `main.go`, `describeonly.go` (new), `host.go`, `describe_whatsapp_test.go` (new), `whatsapplink.go`, `whatsapplink_test.go`, `matchconfig.go`, `supervisor.go`, `suspend_test.go`). `plugins/whatsapp`'s own test suite re-run live: `ok github.com/davison/topos/plugins/whatsapp 0.053s`. |
| 2 | Messages from WhatsApp groups whose names match the webspace's matching config appear in the stream alongside every other source, using the Phase-4 chat rendering | ✓ VERIFIED (regression check — unchanged) | `plugin.go`, `render.go`, `digest.go` untouched by the fix commits. Same plugin test suite pass confirms `Match`/`toItem`/digest logic unaffected. |
| 3 | The plugin persists its own message store, so conversations captured while it was running stay browsable regardless of what the WhatsApp desktop app retains | ✓ VERIFIED (regression check — unchanged) | `messagestore.go` untouched by the fix commits; idempotent-append and chat-isolation unit tests still pass in the full suite run. |
| 4 | De-link, ban, or session expiry surfaces as an explicit plugin-health error in the UI while previously captured messages remain browsable and every other source is unaffected | ✓ VERIFIED (regression check — unchanged) | `health.go` untouched by the fix commits; `TestDelinkPreservesStore` and health-state tests still pass in the full suite run. WR-02 (noted as a narrow, unrelated warning in the prior verification, not part of this criterion) is now fixed as well (see truth 5's evidence) — strictly an improvement, not a regression risk to this truth. |
| 5 (phase-derived, from Plan 08-02's own must_have) | A running WhatsApp instance's match configuration can be viewed/edited through the existing "Edit match settings…" UI, and an existing instance can be added to a second webspace — both reused, unmodified Phase 7 machinery this phase's plans explicitly rely on | ✓ VERIFIED (gap closed) | CR-01 fix (commit `7cb51d1`) verified end to end. See "CR-01 Fix Verification" below for the full chain of evidence: code read of the actual fix, plus the regression test built against the real `topos-plugin-whatsapp` binary re-run live and passing. |

**Score:** 5/5 truths verified (was 4/5 — the one previously-failing truth is now closed)

### CR-01 Fix Verification (the gap being closed)

**Claim:** `POST /api/config/describe-plugin`'s trial-launch of `plugins/whatsapp` no longer collides with an already-running instance's exclusive store lock (`storelock.go`), because the trial-launch now runs in a describe-only mode that never opens the local store or acquires that lock.

**Code-level confirmation, read directly (not from SUMMARY/REVIEW-FIX prose):**

1. `kernel/pluginhost/host.go`'s `launch()` gained a `describeOnly bool` parameter. `Discover()` and `Reconcile()` — the kernel's real boot-time and hot-apply launch paths — both pass `false` (unchanged behavior). `DescribePluginType()` — the function `kernel/httpapi/config.go:331`'s `DescribePluginHandler` calls, i.e. the actual handler behind `POST /api/config/describe-plugin` — passes `true` unconditionally at its single call site. When `true`, `launch()` appends `WEBSPACES_DESCRIBE_ONLY=1` to the spawned subprocess's environment alongside the existing `WEBSPACES_SOURCE_CONFIG`.
2. `plugins/whatsapp/main.go` checks `os.Getenv("WEBSPACES_DESCRIBE_ONLY") == "1"` **before** requiring `WEBSPACES_SOURCE_CONFIG` at all. When set, `impl = describeOnlyPlugin{}` — the `NewSourcePlugin` → `startBackgroundClient` → `acquireStoreLock` chain (the exact chain the original gap named) is never reached. When unset, behavior is byte-identical to before the fix.
3. `plugins/whatsapp/describeonly.go`'s `describeOnlyPlugin.Describe` returns the same fixed package-level constants (`sourceType`, `displayName`, `contractVersion`, `matchVocabulary`) `SourcePlugin.Describe` always returned — no live connection or store access required. `Match`/`Fetch`/`Health` explicitly return `codes.Unimplemented` rather than silently no-op, and `kernel/httpapi/config_test.go`'s pre-existing `TestDescribePluginTypeGuard_ReachesNoRPCBeyondDescribe` AST guard (re-run live, passes) proves the describe-plugin handler's contract never calls them anyway.
4. Blast-radius check: `grep -rn "WEBSPACES_DESCRIBE_ONLY" plugins/ kernel/ --include=*.go` shows the variable is set in exactly one place (`host.go`) and read in exactly one place (`plugins/whatsapp/main.go`) — no other plugin type (Signal, Proton, paperless, SilverBullet, mock) reads it, confirming the fix is additive and doesn't change trial-launch behavior for any other source type.

**Regression test — built and run live in this verification session (not taken on faith from 08-REVIEW-FIX.md):**

```
$ CGO_ENABLED=0 go test ./kernel/pluginhost/... -run TestDescribePluginType_WhatsApp_SucceedsWhileARealInstanceHoldsTheStoreLock -v
=== RUN   TestDescribePluginType_WhatsApp_SucceedsWhileARealInstanceHoldsTheStoreLock
--- PASS: TestDescribePluginType_WhatsApp_SucceedsWhileARealInstanceHoldsTheStoreLock (0.95s)
PASS
```

This test builds the real `topos-plugin-whatsapp` binary (not the mock reference plugin), launches a genuine running instance via `Discover` (holding `storelock.go`'s exclusive flock for the data directory, exactly as a real linked source does), then — while that instance is still alive — calls `DescribePluginType` against the same binary and data directory (the exact call shape both "Edit match settings…" and the "+" picker's add-existing-instance flow make). It asserts `DescribePluginType` succeeds, returns the correct `source_type`/`plugin_display_name`/`match_vocabulary`, and that the real running instance is left completely untouched (`h.Plugins()` still reports exactly the one instance, unchanged). This is the specific scenario the original gap said always failed with `pluginhost: trial-launch for describe: …` (a handshake failure before `Describe` was ever reached) — it now passes.

**Verdict:** The fix closes the gap as claimed. The trial-launch path used by `POST /api/config/describe-plugin` (both UI entry points — chip "Edit match settings…" via `+page.svelte`'s `handleChipEdit`, and `AddSourceModal.svelte`'s `selectExisting`) now goes through `DescribePluginType` → `launch(describeOnly=true)` → `describeOnlyPlugin`, which cannot contend for the store lock because it never attempts to acquire it.

### Other Two Fixes (WR-01, WR-02) — Spot-Checked

These were Warning-grade findings in 08-REVIEW.md, not part of the original blocking gap, but their regression tests were re-run live as part of this re-verification since they landed in the same fix pass and touch adjacent WhatsApp-linking code paths:

```
$ CGO_ENABLED=0 go test ./kernel/httpapi/... -run TestWhatsAppLinkStart_CapEnforcedBeforeSpawn -v
--- PASS: TestWhatsAppLinkStart_CapEnforcedBeforeSpawn (0.00s)

$ CGO_ENABLED=0 go test ./kernel/supervisor/... -run TestApply_UnrelatedSaveSucceedsWhileAnInstanceIsSuspended -v
--- PASS: TestApply_UnrelatedSaveSucceedsWhileAnInstanceIsSuspended (0.50s)
```

Both pass. Neither was a blocking must-have in the prior verification, but their fix is a net improvement with no observed regression risk.

### Required Artifacts

Unchanged from the prior verification except the three fixed files. New/modified artifacts from the fix pass:

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `plugins/whatsapp/main.go` | Describe-only mode short-circuits before `NewSourcePlugin`/`acquireStoreLock` | ✓ VERIFIED | Code read confirms the `WEBSPACES_DESCRIBE_ONLY` check precedes the `WEBSPACES_SOURCE_CONFIG` requirement and the `NewSourcePlugin` call |
| `plugins/whatsapp/describeonly.go` (new) | Minimal `describeOnlyPlugin` answering only `Describe` from fixed constants | ✓ VERIFIED | Exists, substantive (not a stub — real constants, explicit `Unimplemented` refusals for the other 3 RPCs, not silent no-ops) |
| `kernel/pluginhost/host.go` | `launch()` gains `describeOnly` param; `DescribePluginType` passes `true`, `Discover`/`Reconcile` pass `false` | ✓ VERIFIED | Diff read directly; all three call sites confirmed |
| `kernel/pluginhost/describe_whatsapp_test.go` (new) | Regression test against a real built binary and a real running instance | ✓ VERIFIED | Read in full; re-run live, passes |
| `kernel/httpapi/whatsapplink.go` | `reserve`/`release` claim a session slot before spawn | ✓ VERIFIED | Regression test re-run live, passes |
| `kernel/supervisor/supervisor.go` | `suspended` map excludes suspended names from `Apply`'s relaunch/validation | ✓ VERIFIED | Regression test re-run live, passes |

All artifacts from the prior verification's table (messagestore.go, connect.go, link.go, storelock.go, plugin.go, health.go, digest.go, render.go, readonly_test.go, outbound_hosts_test.go, QRPanel.svelte, RelinkModal.svelte, uat-08-whatsapp-qr-link.spec.ts) are untouched by this fix pass (confirmed via `git diff --stat` showing exactly 9 files changed, none of these among them) and remain ✓ VERIFIED by regression (full `make test-portable` re-run, all pass).

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `web/src/routes/w/[webspace]/+page.svelte` (`handleChipEdit`, `'match'` kind) | `describePlugin` → `plugins/whatsapp` trial-launch | Edit match settings for an existing WhatsApp source | ✓ WIRED | Previously NOT_WIRED for a running instance (Gap 1). Now closed: `DescribePluginHandler` → `DescribePluginType(describeOnly=true)` → `launch()` sets `WEBSPACES_DESCRIBE_ONLY=1` → `describeOnlyPlugin` answers without touching the store lock. Proven live by `TestDescribePluginType_WhatsApp_SucceedsWhileARealInstanceHoldsTheStoreLock`, which exercises exactly this call shape against a real running instance. |
| `web/src/lib/components/AddSourceModal.svelte` (`selectExisting`) | `describePlugin` → `plugins/whatsapp` trial-launch | Add an already-configured WhatsApp instance to a second webspace | ✓ WIRED | Same fix, same call path (`DescribePluginHandler` is the single handler behind both UI flows) — closed by the same code change and proven by the same regression test. |

All other key links from the prior verification are unchanged (no touched files in their chain) and remain ✓ WIRED.

### Data-Flow Trace (Level 4)

Unchanged from prior verification — no data-flow-relevant code (message store reads, digest assembly, transcript rendering) was touched by the fix pass. `Describe`'s response (`match_vocabulary`) now flows from `describeOnlyPlugin`'s fixed constants when describe-only, and from `SourcePlugin.Describe`'s identical fixed constants otherwise — same static, real (non-mocked) values either way, confirmed by the regression test's explicit assertion on `info.MatchVocabulary`.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| CR-01 regression test (built against the real `topos-plugin-whatsapp` binary) | `CGO_ENABLED=0 go test ./kernel/pluginhost/... -run TestDescribePluginType_WhatsApp_SucceedsWhileARealInstanceHoldsTheStoreLock -v` | PASS (0.95s) | ✓ PASS — re-run live in this session, not taken from 08-REVIEW-FIX.md's report |
| AST guard: describe-only launch never reaches Match/Fetch/Health | `CGO_ENABLED=0 go test ./kernel/httpapi/... -run TestDescribePluginTypeGuard_ReachesNoRPCBeyondDescribe -v` | PASS | ✓ PASS |
| WR-01 regression test | `CGO_ENABLED=0 go test ./kernel/httpapi/... -run TestWhatsAppLinkStart_CapEnforcedBeforeSpawn -v` | PASS | ✓ PASS |
| WR-02 regression test | `CGO_ENABLED=0 go test ./kernel/supervisor/... -run TestApply_UnrelatedSaveSucceedsWhileAnInstanceIsSuspended -v` | PASS | ✓ PASS |
| whatsapp plugin's own full test suite (regression check on the 4 untouched must-haves) | `cd plugins/whatsapp && CGO_ENABLED=0 go test ./...` | PASS (0.053s) | ✓ PASS |
| Full workspace build + test-portable (all Go modules) | `CGO_ENABLED=0 go build ./...` then `make test-portable` | Both succeed across all Go modules including `plugins/whatsapp` | ✓ PASS |
| Blast-radius check: no other plugin reads the new env var | `grep -rn "WEBSPACES_DESCRIBE_ONLY" plugins/ kernel/ --include=*.go` | Exactly one setter (`host.go`), one reader (`plugins/whatsapp/main.go`) | ✓ PASS |
| Debt-marker scan on all 9 files touched by the fix pass | `grep -n -E "TBD\|FIXME\|XXX\|TODO\|HACK\|PLACEHOLDER"` on each | No matches in any file | ✓ PASS |
| Web unit suite (no web files touched by fix pass — confirmed by `git log -1 -- web/` predating all 3 fix commits) | Not re-run (no code under test changed); 642/642 previously confirmed and unaffected | N/A — regression not possible | ✓ PASS (by scope exclusion) |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes declared by this phase's plans or ROADMAP success criteria. Step 7c: SKIPPED (no declared probes) — unchanged from prior verification.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| SRC-03 | 08-01, 08-02, 08-03, 08-04 | WhatsApp plugin runs as a whatsmeow linked device with its own persistent message store; degrades gracefully on de-link/ban; matches on group names | ✓ SATISFIED | All 5 observable truths verified, including the previously-failing UI editing path for an already-linked instance (CR-01, now closed). No remaining caveat. `.planning/REQUIREMENTS.md` line 30 still shows `[ ]` Pending — this is expected, since REQUIREMENTS.md is updated by the orchestrator after verification passes, not by this verification step itself. |

No orphaned requirements — SRC-03 is the only ID `.planning/ROADMAP.md` maps to Phase 8, and all four plans declare `requirements: [SRC-03]`.

### Anti-Patterns Found

No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers found in any of the 9 files touched by the fix pass (`plugins/whatsapp/main.go`, `plugins/whatsapp/describeonly.go`, `kernel/pluginhost/host.go`, `kernel/pluginhost/describe_whatsapp_test.go`, `kernel/httpapi/whatsapplink.go`, `kernel/httpapi/whatsapplink_test.go`, `kernel/pluginhost/matchconfig.go`, `kernel/supervisor/supervisor.go`, `kernel/supervisor/suspend_test.go`). No blocker-grade debt markers. (Prior verification's scan of the original phase files also found none — unchanged.)

| File | Line | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| (none found) | — | — | — | — |

### Human Verification Required

None. CR-01's fix is a deterministic, code-level change proven by a regression test that was built and re-run live against the real plugin binary in this verification session — not a judgment call. The 4 previously-verified truths were confirmed unaffected by scope exclusion (no touched files in their code paths) plus a live re-run of their existing test coverage. No further human testing is needed.

### Gaps Summary

None. The single gap from the prior verification (CR-01: `describePlugin`'s trial-launch colliding with a running WhatsApp instance's exclusive store lock, breaking "Edit match settings…" and add-existing-instance) is closed. The fix was verified directly against the codebase in this session: the code path was read end to end (`DescribePluginHandler` → `DescribePluginType(describeOnly=true)` → `launch()` → `WEBSPACES_DESCRIBE_ONLY=1` → `describeOnlyPlugin`), and its regression test — which builds the real `topos-plugin-whatsapp` binary and launches a genuine running instance holding the store lock — was re-run live and passes. The two Warning-grade fixes (WR-01, WR-02) from the same fix pass were also spot-checked live and pass. No regressions were found in the 4 previously-verified truths, whose supporting code was entirely untouched by this fix pass (confirmed via `git diff --stat` scoped to the 3 fix commits).

Phase 8's goal — WhatsApp groups appear in the stream via a linked-device session, and everything else keeps working when that session breaks, including the UI machinery for editing an already-linked instance's match configuration — is now fully achieved and verified.

---

_Verified: 2026-08-10T18:05:00Z_
_Verifier: Claude (gsd-verifier)_
