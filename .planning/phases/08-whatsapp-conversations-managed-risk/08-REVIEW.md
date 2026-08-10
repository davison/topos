---
phase: 08-whatsapp-conversations-managed-risk
reviewed: 2026-08-10T00:00:00Z
depth: standard
files_reviewed: 13
files_reviewed_list:
  - docs/api.md
  - kernel/httpapi/routes.go
  - kernel/httpapi/whatsapplink_exec_test.go
  - kernel/httpapi/whatsapplink.go
  - kernel/httpapi/whatsapplink_test.go
  - plugins/whatsapp/link.go
  - plugins/whatsapp/link_test.go
  - web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts
  - web/src/lib/api.ts
  - web/src/lib/components/AddSourceModal.svelte
  - web/src/lib/components/add-source.test.ts
  - web/src/lib/components/QRPanel.svelte
  - web/src/lib/components/qr-panel.test.ts
findings:
  critical: 1
  warning: 1
  info: 1
  total: 3
status: issues_found
---

# Phase 08: Code Review Report

**Reviewed:** 2026-08-10T00:00:00Z
**Depth:** standard
**Files Reviewed:** 13
**Status:** issues_found

## Summary

**Scope note:** this review REPLACES the prior `08-REVIEW.md`, which covered
the whole phase (44 files). This pass is scoped strictly to the 13 files
touched by gap-closure plans `08-05`/`08-06`/`08-07` for gap `G-08-1` (fixed
QR poll cadence, non-terminal `pairing_accepted`/`already_linked` link states
threaded through plugin→kernel→web, kernel capture of link-subprocess
stderr, the declined-link notice, and the new e2e regression specs). The
prior review's CR-01/WR-01/WR-02/IN-01 findings live in files outside this
diff's scope (`storelock.go`, `supervisor.go`, `+page.svelte`,
`RelinkModal.svelte`, etc.) and were not re-audited here except where this
diff directly touches the same mechanism (WR-01 there — the reserve-before-spawn
cap fix — is confirmed still correctly applied in the current
`kernel/httpapi/whatsapplink.go`, see below).

The kernel-side session lifecycle (`kernel/httpapi/whatsapplink.go`) was
re-audited in full given how much of `G-08-1` lived there — the
reserve/register/release slot bookkeeping, the background reaper, the
exactly-once terminal-retirement contract, and the new `stderrLineLogger`
are all correct and well covered by `whatsapplink_test.go` /
`whatsapplink_exec_test.go`. The plugin-side shared link core
(`plugins/whatsapp/link.go`) correctly emits the two new non-terminal
progress kinds without ever leaking the raw QR payload or a device
identifier onto stdout, matching `docs/api.md`'s own contract, and is well
covered by `link_test.go`.

The one real defect found is client-side: `QRPanel.svelte`'s unmount-time
session cancellation has a race that can silently orphan a live link session
(and, for the Re-link flow, leave a real source instance suspended) for up
to five minutes when the panel is torn down while its initial `POST
/api/config/whatsapp-link` call is still in flight — directly contradicting
this same file's own "must never leave a subprocess alive holding the
WhatsApp store lock" invariant. A second, lower-severity issue is a stale UI
notice that can co-render with a later connection-failure alert. A third is
a stale comment in the new e2e spec referencing a poll-cadence mechanism
that no longer exists in the shipped code.

## Critical Issues

### CR-01: QRPanel unmount during the in-flight start request never cancels the created link session

**File:** `web/src/lib/components/QRPanel.svelte:219-262`
**Issue:**

`retireSession()` (called from both `onDestroy` and the explicit Cancel
button) only issues `cancelWhatsAppLink` when `sessionId` is already set:

```js
function retireSession() {
	retired = true;
	clearTimers();
	if (sessionId) {
		const id = sessionId;
		sessionId = null;
		void cancelWhatsAppLink(id).catch(() => {});
	}
}
```

`sessionId` is only assigned after `startWhatsAppLink` resolves, inside
`beginSession`:

```js
async function beginSession() {
	retired = false;
	...
	try {
		const session = await startWhatsAppLink({ plugin, path, instance });
		if (retired) return;              // <-- discards the session with no cancel
		sessionId = session.session;
		applySession(session);
	} catch (err) { ... }
}
```

If the component unmounts (dialog closed via Escape, backdrop click, or the
surrounding modal being torn down for any other reason) while the initial
`POST /api/config/whatsapp-link` is still in flight, `onDestroy` fires
`retireSession()` while `sessionId` is still `null` — so no cancel request is
ever sent. When the `startWhatsAppLink` promise then resolves, `if (retired)
return;` discards the response without ever recording `sessionId`, so the
now-unreachable session can never be cancelled by this component either.

The kernel has already spawned a real subprocess for that session by the
time it answers `200` (and, for the Re-link entry point, has already called
`SuspendInstance` on the real source instance —
`kernel/httpapi/whatsapplink.go`'s `WhatsAppLinkStartHandler` suspends and
spawns before it ever returns a session id). That subprocess — and the
suspended instance behind it — is now orphaned client-side. It is only
recovered by the kernel's own background reaper after
`linkSessionDeadline` (5 minutes, `kernel/httpapi/whatsapplink.go:216-223`),
or sooner if a fifth concurrent start request hits
`maxConcurrentLinkSessions` (4) and is rejected with `429`.

This is squarely the failure mode `onDestroy`'s own comment says must never
happen:

> "T-08-10's mitigation, second half: cancel on unmount too... navigating
> away... must never leave a subprocess alive holding the WhatsApp store
> lock." (`QRPanel.svelte:278-284`)

The race window is not theoretical: `POST /api/config/whatsapp-link` does
real work before responding (directory-listing check, `SuspendInstance`,
`exec.Start`, two SQLite opens, an exclusive flock) — plenty of time for a
user to press Escape or click away immediately after opening the Add-Source
/ Re-link dialog, which is a completely ordinary interaction, not an
adversarial one. Repeating that a few times in quick succession (e.g.
opening and immediately closing the dialog while deciding whether to link)
can also exhaust `maxConcurrentLinkSessions`, making the link feature return
`429` for up to 5 minutes for an unrelated, well-behaved future attempt —
undermining the very cap this phase's kernel-side `reserve()` mechanism
(WR-01 from the prior `08-REVIEW.md`, confirmed still correctly applied at
`kernel/httpapi/whatsapplink.go:614-617`) exists to protect.

**Fix:** cancel the session the moment it is known, even if the component
has already been marked `retired` by the time the start response arrives:

```js
try {
	const session = await startWhatsAppLink({ plugin, path, instance });
	if (retired) {
		// The component was torn down while the start request was still
		// in flight — the kernel has already spawned a subprocess (and,
		// for Re-link, already suspended the real instance) for this
		// session id. Cancel it now rather than leaving it to the
		// kernel's 5-minute reaper.
		void cancelWhatsAppLink(session.session).catch(() => {});
		return;
	}
	sessionId = session.session;
	applySession(session);
} catch (err) { ... }
```

## Warnings

### WR-01: Stale declined-link notice can co-render with a later connection-failure alert

**File:** `web/src/lib/components/AddSourceModal.svelte:248-325,555-572`
**Issue:** `handleLinkCancelled` sets `linkNotice` to the neutral "Not linked
yet…" copy when the user cancels out of the QR panel, and by design never
touches `describeFailed`/`connectError` (correctly — declining to link is not
a connection failure). However, `linkNotice` is also never cleared by
`handleConnectNext`. If the user, after declining the link opportunity once,
edits the connect-step fields and clicks "Next" again and this second
`describePlugin` trial launch genuinely fails (network hiccup, transient
plugin error, etc.), `handleConnectNext`'s catch branch sets `describeFailed =
true` and `connectError = "Couldn't verify this connection. …"` but leaves
the earlier `linkNotice` untouched. Both are rendered unconditionally
whenever set:

```svelte
{#if connectError}
	<Alert variant="destructive" class="mt-4">
		<AlertDescription>{connectError}</AlertDescription>
	</Alert>
{/if}

{#if linkNotice}
	<p class="mt-4 text-[14px] leading-[1.4] text-muted-foreground">{linkNotice}</p>
{/if}
```

The user would see a destructive "Couldn't verify this connection…" alert and
the muted "Not linked yet — you can save this source now and link later…"
notice at the same time — the second message implies a working, saveable
connection while the first says the connection just failed, which is
confusing and self-contradictory.

**Fix:** clear `linkNotice` at the top of `handleConnectNext` (mirroring how
`selectPluginType` already resets it), so a fresh trial-launch attempt starts
without a stale prior-outcome message:

```js
async function handleConnectNext(event: SubmitEvent) {
	event.preventDefault();
	if (!selectedPluginType || describing) return;
	linkNotice = '';
	...
```

## Info

### IN-01: Stale comment references a `POLL_FLOOR_MS` mechanism that no longer exists

**File:** `web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts:298-302`
**Issue:** Case 2's setup comment reads:

```ts
// start answers the FIRST qr response directly (floored
// expires_in_seconds so the panel's own poll cadence — clamped to
// POLL_FLOOR_MS — fires promptly); the one scripted poll answers
// the SECOND, different qr response.
```

`POLL_FLOOR_MS` does not exist anywhere in the shipped source
(`QRPanel.svelte` only defines a fixed `POLL_INTERVAL_MS = 2000`, with no
clamp/floor logic tied to `expires_in_seconds` — that per-response-tied
cadence is exactly what `G-08-1`'s fix removed). This comment appears to be
left over from an earlier design iteration and now describes a mechanism
that was deliberately deleted by this same gap-closure. It doesn't affect
test correctness, but it will actively mislead a future reader trying to
understand the panel's poll cadence from this spec.

**Fix:** update the comment to reference `POLL_INTERVAL_MS` (as the header
comment and case 9's comment already correctly do), e.g.:

```ts
// start answers the FIRST qr response directly; the one scripted poll
// answers the SECOND, different qr response, delivered on QRPanel's own
// fixed POLL_INTERVAL_MS cadence (not tied to expires_in_seconds — G-08-1).
```

---

_Reviewed: 2026-08-10T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
