<script lang="ts">
	// The single QR pairing panel component (D-01/D-02/D-03, 08-04-PLAN.md
	// Task 1) — reused, unforked, from exactly two entry points: inline in
	// the Add-Source Step 1 dialog (AddSourceModal.svelte) and inside the
	// small Re-link… dialog (RelinkModal.svelte, Task 2). Drives the
	// link-session HTTP surface (docs/api.md's "POST/GET/DELETE
	// /api/config/whatsapp-link") via api.ts's typed start/poll/cancel
	// clients — see 08-UI-SPEC.md's "Amendment — In-App QR Panel" for the
	// full component contract implemented here (sizing, copy, the five
	// states, the two entry-point success transitions).
	import { onMount, onDestroy } from 'svelte';
	import { Alert, AlertDescription } from '$lib/components/ui/alert/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import {
		startWhatsAppLink,
		pollWhatsAppLink,
		cancelWhatsAppLink,
		ApiError,
		type WhatsAppLinkSession
	} from '$lib/api';

	let {
		plugin,
		path,
		instance,
		onpaired,
		oncancelled
	}: {
		// plugin is the plugin binary name (e.g. "topos-plugin-whatsapp"),
		// forwarded verbatim into startWhatsAppLink's own request shape.
		plugin: string;
		// path is the WhatsApp instance's own data directory — the
		// just-typed Step 1 value in the Add-Source flow, or the
		// already-configured instance's own stored path in the Re-link
		// flow.
		path: string;
		// instance is present for the Re-link… entry point (an
		// already-configured instance name the kernel suspends for the
		// session's duration, docs/api.md's SuspendInstance note) and
		// absent for the Add-Source entry point, where nothing is
		// configured yet to suspend.
		instance?: string;
		// onpaired fires once the session reaches the paired terminal
		// state — the Add-Source caller advances to the match step; the
		// Re-link caller closes its dialog and shows a brief confirmation
		// in the chip's own area.
		onpaired: () => void;
		// oncancelled fires only on an explicit user cancel — the
		// Add-Source caller returns to the connect step with every typed
		// connection value intact (D-02); the Re-link caller closes its
		// dialog. Never fired on unmount (see retireSession below); that
		// path cancels the session but does not call this, since the
		// caller is already tearing this component down for its own
		// reason.
		oncancelled: () => void;
	} = $props();

	// The six states 08-UI-SPEC.md's Amendments define: loading (before
	// the first qr event), qr (the populated state), pairing (post-scan
	// progress, Amendment 2/G-08-1), error, expired (timeout) and success
	// (paired).
	type PanelPhase = 'loading' | 'qr' | 'pairing' | 'error' | 'expired' | 'success';

	let phase = $state<PanelPhase>('loading');
	let qrDataUri = $state('');
	let remainingSeconds = $state(0);
	let errorMessage = $state('');
	let pairingMessage = $state('');

	let sessionId: string | null = null;
	let pollTimer: ReturnType<typeof setTimeout> | null = null;
	let countdownTimer: ReturnType<typeof setInterval> | null = null;
	// retired guards every async continuation below (a poll response, a
	// start response) against writing $state or scheduling a further poll
	// once this component has moved past caring — either it unmounted, or
	// the session already reached a terminal state via a previous
	// response. Set back to false only when a fresh session begins
	// (beginSession, including a user-initiated Retry/Restart).
	let retired = false;

	// 08-UI-SPEC.md Amendment 2 progress copies (G-08-1) — module-level
	// literals so the markup and the structural guards reference one
	// literal each.
	const PAIRING_ACCEPTED_MESSAGE = 'Scan accepted — completing login…';
	const ALREADY_LINKED_MESSAGE = 'Already linked — confirming this session…';

	// POLL_INTERVAL_MS is the panel's own liveness clock, deliberately
	// independent of any code's validity window (08-UAT.md's G-08-1):
	// tying the poll cadence to expires_in_seconds left a terminal event
	// the kernel had already recorded undelivered to the browser for up
	// to a full 60-second first-code window.
	const POLL_INTERVAL_MS = 2000;

	function clearTimers() {
		if (pollTimer !== null) {
			clearTimeout(pollTimer);
			pollTimer = null;
		}
		clearCountdown();
	}

	function clearCountdown() {
		if (countdownTimer !== null) {
			clearInterval(countdownTimer);
			countdownTimer = null;
		}
	}

	function startCountdown(seconds: number) {
		clearCountdown();
		remainingSeconds = seconds;
		countdownTimer = setInterval(() => {
			remainingSeconds = Math.max(0, remainingSeconds - 1);
		}, 1000);
	}

	// schedulePoll takes no delay argument at all — that signature is the
	// structural foreclosure of ever re-tying cadence to a code's expiry
	// again (G-08-1).
	function schedulePoll() {
		if (retired) return;
		if (pollTimer !== null) clearTimeout(pollTimer);
		pollTimer = setTimeout(() => void poll(), POLL_INTERVAL_MS);
	}

	function applySession(session: WhatsAppLinkSession) {
		if (retired) return;
		switch (session.state) {
			case 'pending':
				phase = 'loading';
				schedulePoll();
				break;
			case 'qr': {
				// A new qr event (the code rotating) swaps the image and
				// resets the countdown in place — no flash back to
				// 'loading' between rotations (08-UI-SPEC.md's Populated
				// row). whatsmeow stops emitting codes at PairSuccess, so
				// the kernel's `latest` freezes on the already-scanned
				// code once the phone accepts it; restart the countdown
				// only when the incoming code actually differs from the
				// one already rendered — an unconditional restart is what
				// made a frozen (already-scanned) session look like a
				// live one still refreshing (G-08-1).
				phase = 'qr';
				const incomingDataUri = session.png_data_uri ?? '';
				if (incomingDataUri !== qrDataUri) {
					startCountdown(session.expires_in_seconds ?? 0);
				}
				qrDataUri = incomingDataUri;
				schedulePoll();
				break;
			}
			case 'pairing_accepted':
				// Non-terminal (docs/api.md): the phone accepted the scan;
				// the plugin is completing the post-pair login handshake.
				// Stop the frozen code's countdown — it must not keep
				// ticking behind the progress line — and keep polling.
				phase = 'pairing';
				pairingMessage = PAIRING_ACCEPTED_MESSAGE;
				clearCountdown();
				schedulePoll();
				break;
			case 'already_linked':
				// Non-terminal (docs/api.md): the store already held a
				// linked device when the session started; the plugin is
				// reconnecting to confirm that session is genuinely
				// usable. Keep polling.
				phase = 'pairing';
				pairingMessage = ALREADY_LINKED_MESSAGE;
				clearCountdown();
				schedulePoll();
				break;
			case 'paired':
				retired = true;
				clearTimers();
				phase = 'success';
				onpaired();
				break;
			case 'error':
				retired = true;
				clearTimers();
				phase = 'error';
				errorMessage = session.message || 'The link attempt failed.';
				break;
			case 'timeout':
				retired = true;
				clearTimers();
				phase = 'expired';
				break;
			default:
				// An unrecognised non-terminal state must never terminate
				// the liveness poll — that fallthrough (a bare break) is
				// what would have hung the panel had the producer shipped
				// first (G-08-1).
				schedulePoll();
				break;
		}
	}

	async function poll() {
		if (retired || !sessionId) return;
		try {
			const session = await pollWhatsAppLink(sessionId);
			applySession(session);
		} catch (err) {
			if (retired) return;
			retired = true;
			clearTimers();
			phase = 'error';
			errorMessage =
				err instanceof ApiError
					? err.message
					: 'Lost contact with the link session — check the browser console and try again.';
		}
	}

	async function beginSession() {
		retired = false;
		phase = 'loading';
		errorMessage = '';
		qrDataUri = '';
		// Reset per-session so a Retry after a failed already-linked
		// confirmation does not flash the previous run's progress line.
		pairingMessage = '';
		try {
			const session = await startWhatsAppLink({ plugin, path, instance });
			if (retired) {
				// The component was torn down while this start request was
				// still in flight (08-REVIEW.md CR-01) — the kernel has
				// already spawned a link subprocess, and on the Re-link
				// entry point already suspended the real source instance,
				// by the time it answers 200. A session id learned after
				// teardown is still a live resource this component owns,
				// per this file's own onDestroy invariant below. The
				// alternative is the kernel's five-minute reaper, and a few
				// repeated open/close cycles exhaust the four-slot
				// concurrency cap long before that. Do NOT assign
				// sessionId here: leaving it null keeps a later
				// retireSession from issuing a second cancel for this id,
				// and keeps poll()'s own early return in force so no poll
				// is ever issued for a session this panel has abandoned.
				void cancelWhatsAppLink(session.session).catch(() => {
					// best-effort — the session may already be retired
					// server-side.
				});
				return;
			}
			sessionId = session.session;
			applySession(session);
		} catch (err) {
			if (retired) return;
			retired = true;
			phase = 'error';
			errorMessage =
				err instanceof ApiError
					? err.message
					: 'Could not start the link session — check the browser console and try again.';
		}
	}

	// retireSession cancels the live session (best-effort — a session
	// already at a terminal state on the kernel side answers 404, which is
	// harmless here) and stops every timer, without itself firing either
	// caller callback — handleCancel/onDestroy below decide whether
	// oncancelled fires; a plain terminal-state transition (paired/error/
	// timeout) never calls this at all, since the kernel has already
	// retired that session itself (docs/api.md's "terminal states are
	// delivered exactly once").
	function retireSession() {
		retired = true;
		clearTimers();
		if (sessionId) {
			const id = sessionId;
			sessionId = null;
			void cancelWhatsAppLink(id).catch(() => {
				// best-effort — the session may already be retired
				// server-side.
			});
		}
	}

	function handleCancel() {
		retireSession();
		oncancelled();
	}

	function handleRetry() {
		retireSession();
		void beginSession();
	}

	onMount(() => {
		void beginSession();
	});

	// T-08-10's mitigation, second half: cancel on unmount too, not only
	// on an explicit Cancel click — navigating away (closing the
	// surrounding modal any other way) must never leave a subprocess
	// alive holding the WhatsApp store lock.
	onDestroy(() => {
		retireSession();
	});

	function formatCountdown(seconds: number): string {
		const clamped = Math.max(0, seconds);
		const m = Math.floor(clamped / 60);
		const s = clamped % 60;
		return `${m}:${String(s).padStart(2, '0')}`;
	}

	// 08-UI-SPEC.md Amendment 2's countdown floor copy (G-08-1): a code
	// that is not going to refresh must not claim it is about to.
	const countdownLine = $derived(
		remainingSeconds > 0 ? `Refreshes in ${formatCountdown(remainingSeconds)}` : 'Waiting for a new code…'
	);
</script>

<div class="flex flex-col items-center gap-2">
	<!-- topos-branded pairing surface (09-UI-SPEC.md Fix 10): a small,
	     decorative topos app icon above the QR/skeleton area so a user
	     scanning a code sees they are pairing a device with topos, not
	     WhatsApp/Meta. alt="" is deliberate — the copy directly below
	     already states what this is; this image carries no independent
	     accessible name. Rendered unconditionally across every phase
	     (additive branding, not a rework of the existing phase branches
	     below, which stay byte-identical). -->
	<img src="/app-icon.png" alt="" class="size-10 rounded-md" />
	{#if phase === 'loading'}
		<Skeleton class="size-48 rounded-md" />
	{:else if phase === 'qr'}
		<img
			src={qrDataUri}
			alt="WhatsApp pairing QR code — scan with your phone's WhatsApp app to link this device"
			class="size-48"
		/>
		<p class="text-[14px] leading-[1.4] text-foreground">Scan with your phone to link</p>
		<p class="text-[14px] leading-[1.4] text-muted-foreground">
			{countdownLine}
		</p>
	{:else if phase === 'pairing'}
		<Skeleton class="size-48 rounded-md" />
		<p class="text-[14px] leading-[1.4] text-foreground">{pairingMessage}</p>
	{:else if phase === 'error'}
		<Alert variant="destructive" class="w-full">
			<AlertDescription>{errorMessage}</AlertDescription>
		</Alert>
		<Button type="button" variant="outline" onclick={handleRetry}>Retry</Button>
	{:else if phase === 'expired'}
		<Alert variant="destructive" class="w-full">
			<AlertDescription>This code expired — start again to get a new one.</AlertDescription>
		</Alert>
		<Button type="button" variant="outline" onclick={handleRetry}>Restart</Button>
	{:else if phase === 'success'}
		<div class="flex items-center gap-1.5 text-[14px] leading-[1.4] text-foreground">
			<CircleCheck class="size-4 text-success" aria-hidden="true" />
			Linked successfully.
		</div>
	{/if}

	<!-- The pairing phase's absence here is load-bearing, not incidental
	     (08-UI-SPEC.md Amendment 2, G-08-1): cancelling inside the
	     post-pair window SIGKILLs a subprocess mid-login-handshake and
	     strands a pairing whatsmeow has already persisted to disk. -->
	{#if phase === 'loading' || phase === 'qr'}
		<Button type="button" variant="ghost" size="sm" onclick={handleCancel}>Cancel</Button>
	{/if}
</div>
