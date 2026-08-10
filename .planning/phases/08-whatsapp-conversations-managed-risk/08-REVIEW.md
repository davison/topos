---
phase: 08-whatsapp-conversations-managed-risk
reviewed: 2026-08-10T00:00:00Z
depth: standard
files_reviewed: 5
files_reviewed_list:
  - web/src/lib/components/QRPanel.svelte
  - web/src/lib/components/qr-panel.test.ts
  - web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts
  - web/src/lib/components/AddSourceModal.svelte
  - web/src/lib/components/add-source.test.ts
findings:
  critical: 0
  warning: 1
  info: 1
  total: 2
status: issues_found
---

# Phase 08: Code Review Report

**Reviewed:** 2026-08-10T00:00:00Z
**Depth:** standard
**Files Reviewed:** 5
**Status:** issues_found

## Summary

**Scope note:** this review targets the plan `08-08` gap-closure diff
(commits `aa2fba1..d71b408`, base `6e86968`), which closed three findings
from the prior review of this phase: CR-01 (unmount-during-in-flight-start
race in `QRPanel.svelte`), WR-01 (stale declined-link notice in
`AddSourceModal.svelte`), and IN-01 (stale e2e comment referencing a
deleted `POLL_FLOOR_MS` mechanism).

All three closures were traced line-by-line against their new regression
coverage:

- **CR-01** — `beginSession`'s post-`await` `if (retired)` branch now
  issues a best-effort `cancelWhatsAppLink(session.session)` using the
  server-returned id (never the still-null module-level `sessionId`),
  leaves `sessionId` unassigned so a later `retireSession()` cannot
  double-cancel and `poll()`'s own `!sessionId` guard stays in force, and
  swallows rejection via `.catch()`. Traced every interleaving (unmount
  before start resolves, explicit Cancel-click before start resolves, the
  normal non-retired path) — each produces exactly one `DELETE` call for
  the abandoned session and zero polls, matching e2e case 12's
  assertions (`deleteCalls === 1`, `deletedSessionIds === ['sess-inflight']`,
  `pollCalls === 0`). This closure is correct.
- **WR-01** — `handleConnectNext` now clears `linkNotice` unconditionally
  as its first statement after the `!selectedPluginType || describing`
  guard, strictly before `missingRequiredFields(`, so a stale
  declined-link notice can never survive into a fresh trial-launch
  attempt's failure branch (or its missing-field branch). Verified against
  e2e case 13 and the new structural guard in `add-source.test.ts`. This
  closure is correct.
- **IN-01** — the case 2 setup comment no longer mentions
  `POLL_FLOOR_MS`, and the deleted-mechanism string does not appear
  anywhere in the shipped `.svelte`/`.ts` sources. However, see IN-02
  below: a second, wording-only echo of the same deleted mechanism
  survives in a comment the gap-closure plan's automated check (a literal
  `grep -c 'POLL_FLOOR_MS'`) could not detect because it uses a synonym
  rather than the literal identifier.

One residual issue was found while specifically tracing the
"exactly-one-cancel" invariant the CR-01 fix's own doc comments assert:
that invariant is not actually held end-to-end, because the adjacent
(unmodified) terminal-state branches of `applySession` never clear
`sessionId`. See WR-02 below.

## Warnings

### WR-02: `sessionId` survives every terminal-state transition, so a later unmount, Retry, or Restart re-issues a redundant cancel — contradicting the invariant the CR-01 fix's own comments assert

**File:** `web/src/lib/components/QRPanel.svelte:175-191` (the `paired`/`error`/`timeout` cases of `applySession`) and `web/src/lib/components/QRPanel.svelte:271-282` (`retireSession`)

**Issue:** The CR-01 fix's own doc comment on `retireSession` states the
invariant this file is meant to hold: "a plain terminal-state transition
(paired/error/timeout) never calls this at all, since the kernel has
already retired that session itself" (lines 267-270), and the new
in-flight-branch comment explicitly frames "leaving it null keeps a later
`retireSession` from issuing a second cancel for this id" (lines 239-243)
as a deliberate design goal — i.e., the codebase's stated intent is
*exactly one* cancel per session, ever.

That invariant does not actually hold. None of the three terminal cases
in `applySession` (`paired` at 175-180, `error` at 181-186, `timeout` at
187-191) clears `sessionId` back to `null` — only `retired` is set and
timers are cleared. `onDestroy` unconditionally calls `retireSession()`
on unmount regardless of what phase the panel is in:

```js
onDestroy(() => {
    retireSession();
});
```

Concretely:

- A successful pairing (`paired`) transitions `AddSourceModal`'s `step`
  from `'link'` to `'match'` (`handleLinkPaired`), which unmounts
  `QRPanel`. `onDestroy` → `retireSession()` finds `sessionId` still set
  from the last `qr` event and issues a second `DELETE` for a session the
  kernel already retired (server-side) at the moment the terminal `paired`
  poll response was built (`kernel/httpapi/whatsapplink.go`'s
  `WhatsAppLinkPollHandler` calls `store.retire(id)` before writing the
  terminal response). That second `DELETE` 404s
  (`link_session_not_found`) — currently harmless, but it is a real,
  observable violation of the "exactly one cancel" property the new code's
  own comments claim, and it happens on the *success* path, the one this
  phase cares most about getting right.
- Clicking **Retry** from the `error` phase, or **Restart** from the
  `expired` phase, calls `handleRetry` → `retireSession()` → finds the
  old (already-terminal) `sessionId` still set → fires a redundant cancel
  for it before starting the new session.

No existing test (unit or e2e) catches this: e2e case 3 (`paired`
success) never asserts `deleteCalls` stays at 0, and the qr-panel
structural guard only checks that the three terminal cases set
`retired = true` and call `clearTimers()` — it does not check `sessionId`.

This is not currently a data-loss or security risk (the kernel's
`WhatsAppLinkCancelHandler` finds no session and 404s harmlessly, and the
already-completed pairing is unaffected), so it is not a BLOCKER. It is a
genuine contract-consistency defect, though: it produces unnecessary
network calls and server-side "session not found" log noise on every
successful link, every Retry, and every Restart, and it means the
invariant this same diff introduces comments asserting is not actually
enforced by the code adjacent to it.

**Fix:** Clear `sessionId` in all three terminal branches of
`applySession`, matching what `retireSession` already does when it
cancels a session itself:

```js
case 'paired':
    retired = true;
    clearTimers();
    sessionId = null;
    phase = 'success';
    onpaired();
    break;
case 'error':
    retired = true;
    clearTimers();
    sessionId = null;
    phase = 'error';
    errorMessage = session.message || 'The link attempt failed.';
    break;
case 'timeout':
    retired = true;
    clearTimers();
    sessionId = null;
    phase = 'expired';
    break;
```

Consider also extending the qr-panel structural guard's "terminal cases"
loop to assert `sessionId = null` alongside the existing `retired = true`
/ `clearTimers()` checks, and adding an e2e assertion (e.g. to case 3)
that `deleteCalls` stays `0` after reaching the `paired` terminal state
and unmounting — otherwise this exact regression can reappear silently.

## Info

### IN-02: A second, non-literal echo of the deleted `POLL_FLOOR_MS` mechanism survives in the case 1 setup comment, undetected by the gap-closure's own automated check

**File:** `web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts:319-324`

**Issue:** IN-01's fix corrected the case 2 setup comment (which
literally named `POLL_FLOOR_MS`) and the gap-closure plan's self-check
gated on `grep -c 'POLL_FLOOR_MS' ... = 0`. That check passes — the
literal identifier is gone from the file. But the case 1 setup comment,
a few lines earlier, still describes the same now-deleted
clamped-to-a-floor mechanism using different words, which the literal
grep cannot catch:

```js
// start answers 'qr' directly — the panel's own 'pending' ->
// first-poll round trip (a real, floored delay) is exercised by
// case 2 below, deliberately; every other case starts already in
// its target state so its assertions are not incidentally timing-
// dependent on that floor.
```

"a real, floored delay" and "that floor" both describe the old
`POLL_FLOOR_MS`-clamped-cadence design that G-08-1 replaced with the
fixed, unconditional `POLL_INTERVAL_MS`. As currently worded this
comment misleads a future reader into thinking a clamping/floor
computation still exists and that case 1's lack of assertions on it is
deliberate — case 1 doesn't poll at all (it answers `'qr'` directly from
`start`), so the "that floor" reference is doubly confusing since it
also isn't case 1's own concern.

**Fix:** Reword to match the corrected case 2 comment's terminology
(fixed cadence, not a floor):

```js
// start answers 'qr' directly — the panel's own 'pending' -> first-
// poll round trip (a real delay, on QRPanel's fixed POLL_INTERVAL_MS
// cadence) is exercised by case 2 below, deliberately; every other
// case starts already in its target state so its assertions are not
// incidentally timing-dependent on that cadence.
```

---

_Reviewed: 2026-08-10T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
