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

	// The five states 08-UI-SPEC.md's Amendment defines: loading (before
	// the first qr event), qr (the populated state), error, expired
	// (timeout) and success (paired).
	type PanelPhase = 'loading' | 'qr' | 'error' | 'expired' | 'success';

	let phase = $state<PanelPhase>('loading');
	let qrDataUri = $state('');
	let remainingSeconds = $state(0);
	let errorMessage = $state('');

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

	// POLL_FLOOR_MS is the clamp the plan's own action text requires: the
	// poll cadence is derived from the session's own reported validity,
	// but never faster than this floor, so a short (or malformed)
	// expires_in_seconds can never produce a request storm.
	const POLL_FLOOR_MS = 2000;

	function clearTimers() {
		if (pollTimer !== null) {
			clearTimeout(pollTimer);
			pollTimer = null;
		}
		if (countdownTimer !== null) {
			clearInterval(countdownTimer);
			countdownTimer = null;
		}
	}

	function startCountdown(seconds: number) {
		remainingSeconds = seconds;
		if (countdownTimer !== null) clearInterval(countdownTimer);
		countdownTimer = setInterval(() => {
			remainingSeconds = Math.max(0, remainingSeconds - 1);
		}, 1000);
	}

	function schedulePoll(delayMs: number) {
		if (retired) return;
		if (pollTimer !== null) clearTimeout(pollTimer);
		pollTimer = setTimeout(() => void poll(), Math.max(delayMs, POLL_FLOOR_MS));
	}

	function applySession(session: WhatsAppLinkSession) {
		if (retired) return;
		switch (session.state) {
			case 'pending':
				phase = 'loading';
				schedulePoll(POLL_FLOOR_MS);
				break;
			case 'qr': {
				// A new qr event (the code rotating) swaps the image and
				// resets the countdown in place — no flash back to
				// 'loading' between rotations (08-UI-SPEC.md's Populated
				// row).
				phase = 'qr';
				qrDataUri = session.png_data_uri ?? '';
				const seconds = session.expires_in_seconds ?? 0;
				startCountdown(seconds);
				schedulePoll(seconds * 1000);
				break;
			}
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
		try {
			const session = await startWhatsAppLink({ plugin, path, instance });
			if (retired) return;
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
</script>

<div class="flex flex-col items-center gap-2">
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
			Refreshes in {formatCountdown(remainingSeconds)}
		</p>
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

	{#if phase === 'loading' || phase === 'qr'}
		<Button type="button" variant="ghost" size="sm" onclick={handleCancel}>Cancel</Button>
	{/if}
</div>
