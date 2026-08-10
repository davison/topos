// 08-04-PLAN.md Task 1's structural guard over QRPanel.svelte, extended by
// 08-05-PLAN.md Task 2 (G-08-1): all six states (loading/qr/pairing/error/
// expired/success) render a distinct branch, the exact instruction line the
// UI-SPEC amendment locks, and both the explicit-cancel and unmount paths
// invoke cancelWhatsAppLink (T-08-10's mitigation, pinned structurally since
// this component's real network behaviour needs a mounted-component harness
// this house pattern doesn't use).
//
// House pattern (matches add-source.test.ts / chip-edit-menu.test.ts):
// comment-stripped source scanning, `extractBetween` scoping, a
// found-non-empty-source guard first, and one consequence-describing
// message per assertion.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const panelPath = join(here, 'QRPanel.svelte');

function stripComments(source: string): string {
	return source
		.replace(/<!--[\s\S]*?-->/g, ' ')
		.replace(/\/\*[\s\S]*?\*\//g, ' ')
		.replace(/\/\/.*$/gm, ' ');
}

function extractBetween(source: string, startMarker: string, endMarker: string): string {
	const startIndex = source.indexOf(startMarker);
	expect(
		startIndex,
		`expected to find "${startMarker}" in the scanned source`
	).toBeGreaterThanOrEqual(0);
	const endIndex = source.indexOf(endMarker, startIndex);
	expect(endIndex, `expected to find "${endMarker}" after "${startMarker}"`).toBeGreaterThan(
		startIndex
	);
	return source.slice(startIndex, endIndex + endMarker.length);
}

const raw = readFileSync(panelPath, 'utf-8');
const stripped = stripComments(raw);

describe('qr-panel guard: found non-empty comment-stripped source', () => {
	it('QRPanel.svelte', () => {
		expect(stripped.length).toBeGreaterThan(0);
	});
});

describe('all six states render a distinct branch', () => {
	// Scanned directly against the whole comment-stripped file rather than
	// a bounded "outer <div>...</div>" slice — several branches (the
	// success branch's confirmation line, in particular) render their own
	// nested <div>, which would make a naive first-</div> extraction stop
	// at the WRONG closing tag. Anchoring on the unique phase-branch
	// markers below is unambiguous either way.

	it("loading renders a Skeleton (before the first qr event arrives)", () => {
		expect(
			/\{#if phase === 'loading'\}[\s\S]*?<Skeleton/.test(stripped),
			'expected the loading branch to render a Skeleton in place of the QR image'
		).toBe(true);
	});

	it('qr (populated) renders the QR image, the instruction line and a countdown line', () => {
		const qrBranch = extractBetween(stripped, "{:else if phase === 'qr'}", "{:else if phase === 'pairing'}");
		expect(qrBranch.includes('<img'), 'expected the qr branch to render the pairing image').toBe(true);
		expect(
			qrBranch.includes('Scan with your phone to link'),
			'expected the qr branch to render the exact frozen instruction line'
		).toBe(true);
		expect(
			qrBranch.includes('{countdownLine}'),
			'expected the qr branch to render the derived countdownLine value (Refreshes in M:SS, or the waiting-for-a-new-code fallback at zero)'
		).toBe(true);
	});

	it('error renders the kernel error message through the destructive Alert, with a Retry control', () => {
		const errorBranch = extractBetween(stripped, "{:else if phase === 'error'}", "{:else if phase === 'expired'}");
		expect(errorBranch.includes('variant="destructive"'), 'expected the error branch to use the destructive Alert variant').toBe(true);
		expect(errorBranch.includes('{errorMessage}'), 'expected the error branch to render the kernel\'s own error message verbatim').toBe(true);
		expect(errorBranch.includes('onclick={handleRetry}'), 'expected the error branch\'s Retry button to restart the link session').toBe(true);
	});

	it('expired renders the fixed expiry copy through the same Alert treatment, with a Restart control, and keeps the surrounding modal open (no close call in this branch)', () => {
		const expiredBranch = extractBetween(stripped, "{:else if phase === 'expired'}", "{:else if phase === 'success'}");
		expect(expiredBranch.includes('variant="destructive"'), 'expected the expired branch to use the same destructive Alert treatment as the error branch').toBe(true);
		expect(
			expiredBranch.includes('This code expired — start again to get a new one.'),
			'expected the expired branch to render the exact frozen expiry copy'
		).toBe(true);
		expect(expiredBranch.includes('onclick={handleRetry}'), 'expected the expired branch\'s Restart button to restart the link session').toBe(true);
		expect(
			expiredBranch.includes('oncancelled') || expiredBranch.includes('onclose'),
			'expected the expired branch to carry no close/cancel call of its own — the surrounding modal must stay open'
		).toBe(false);
	});

	it('success renders a confirmation line (never inline invokes onpaired — onpaired is called from applySession, not a click handler)', () => {
		const successBranch = extractBetween(
			stripped,
			"{:else if phase === 'success'}",
			"{#if phase === 'loading' || phase === 'qr'}"
		);
		expect(successBranch.includes('Linked successfully.'), 'expected the success branch to render a confirmation line').toBe(true);
		expect(
			successBranch.includes('onpaired()'),
			'expected the success branch\'s markup to contain no direct onpaired() call — it is invoked from applySession when the paired state first arrives, not from a template click handler'
		).toBe(false);
	});
});

describe('cancel and unmount both invoke cancelWhatsAppLink (T-08-10)', () => {
	const retireSessionBody = extractBetween(stripped, 'function retireSession() {', '\n\t}');

	it('retireSession calls cancelWhatsAppLink(', () => {
		expect(
			retireSessionBody.includes('cancelWhatsAppLink('),
			'expected retireSession — the one function both the cancel and unmount paths call — to invoke cancelWhatsAppLink('
		).toBe(true);
	});

	it('handleCancel (the explicit Cancel button path) calls retireSession(', () => {
		const handleCancelBody = extractBetween(stripped, 'function handleCancel() {', '\n\t}');
		expect(handleCancelBody.includes('retireSession()')).toBe(true);
	});

	it('onDestroy (the unmount path) calls retireSession(', () => {
		const onDestroyBody = extractBetween(stripped, 'onDestroy(() => {', '\n\t});');
		expect(onDestroyBody.includes('retireSession()')).toBe(true);
	});

	it('onDestroy never calls oncancelled — only an explicit user Cancel does', () => {
		const onDestroyBody = extractBetween(stripped, 'onDestroy(() => {', '\n\t});');
		expect(
			onDestroyBody.includes('oncancelled()'),
			'expected onDestroy to cancel the session without firing oncancelled — the caller is already tearing this component down for its own reason'
		).toBe(false);
	});
});

describe('the liveness poll runs on its own fixed cadence, decoupled from any code\'s validity window (G-08-1)', () => {
	it('schedulePoll takes no delay argument at all', () => {
		expect(
			/function schedulePoll\(\)\s*\{/.test(stripped),
			'expected schedulePoll to be declared with an empty parameter list — a cadence tied to a per-response value is not expressible through this signature'
		).toBe(true);
	});

	it('schedulePoll schedules against the fixed POLL_INTERVAL_MS constant', () => {
		const schedulePollBody = extractBetween(stripped, 'function schedulePoll() {', '\n\t}');
		expect(
			schedulePollBody.includes('POLL_INTERVAL_MS'),
			'expected schedulePoll to schedule its timer against the fixed POLL_INTERVAL_MS constant, not any per-response value'
		).toBe(true);
	});

	it("applySession's qr case calls schedulePoll() with no arguments", () => {
		const qrCaseBody = extractBetween(stripped, "case 'qr': {", "\n\t\t\tcase 'pairing_accepted':");
		expect(
			/schedulePoll\(\)/.test(qrCaseBody),
			'expected the qr case to call schedulePoll() with no arguments'
		).toBe(true);
	});

	it("applySession's default case calls schedulePoll() — an unrecognised non-terminal state must never hang the liveness poll", () => {
		const defaultCaseBody = extractBetween(stripped, 'default:', '\n\t\t}\n\t}');
		expect(
			/schedulePoll\(\)/.test(defaultCaseBody),
			'expected the default case to call schedulePoll() so an unrecognised non-terminal state keeps the liveness poll alive rather than hanging the panel'
		).toBe(true);
	});
});

describe("the qr case restarts the countdown only when the incoming code differs from the one already rendered (G-08-1)", () => {
	it('the qr case guards startCountdown behind a comparison against the currently-rendered qrDataUri', () => {
		const qrCaseBody = extractBetween(stripped, "case 'qr': {", "\n\t\t\tcase 'pairing_accepted':");
		expect(
			/if\s*\([^)]*!==\s*qrDataUri[^)]*\)/.test(qrCaseBody) || /if\s*\(qrDataUri\s*!==/.test(qrCaseBody),
			'expected the qr case to call startCountdown only inside a guard comparing the incoming png_data_uri against the already-rendered qrDataUri — an unconditional restart makes a frozen (already-scanned) code look like it is still refreshing'
		).toBe(true);
	});
});

describe('a post-pair progress phase sits between qr and the terminal states (G-08-1)', () => {
	it("applySession has a 'pairing_accepted' case that sets the pairing phase and calls schedulePoll()", () => {
		const caseBody = extractBetween(stripped, "case 'pairing_accepted':", "\n\t\t\tcase 'already_linked':");
		expect(caseBody.includes("phase = 'pairing'"), 'expected the pairing_accepted case to set phase to \'pairing\'').toBe(true);
		expect(/schedulePoll\(\)/.test(caseBody), 'expected the pairing_accepted case to keep polling by calling schedulePoll()').toBe(true);
		expect(caseBody.includes('clearCountdown()'), 'expected the pairing_accepted case to stop the frozen code\'s countdown via clearCountdown()').toBe(true);
	});

	it("applySession has an 'already_linked' case that sets the pairing phase and calls schedulePoll()", () => {
		const caseBody = extractBetween(stripped, "case 'already_linked':", "\n\t\t\tcase 'paired':");
		expect(caseBody.includes("phase = 'pairing'"), 'expected the already_linked case to set phase to \'pairing\'').toBe(true);
		expect(/schedulePoll\(\)/.test(caseBody), 'expected the already_linked case to keep polling by calling schedulePoll()').toBe(true);
		expect(caseBody.includes('clearCountdown()'), 'expected the already_linked case to stop the frozen code\'s countdown via clearCountdown()').toBe(true);
	});

	it("a {:else if phase === 'pairing'} branch renders a Skeleton and the progress line, and both progress copies exist in the file", () => {
		const pairingBranch = extractBetween(
			stripped,
			"{:else if phase === 'pairing'}",
			"{:else if phase === 'error'}"
		);
		expect(pairingBranch.includes('<Skeleton'), 'expected the pairing branch to render a Skeleton, the same generic loading treatment the loading branch already uses').toBe(true);
		expect(pairingBranch.includes('pairingMessage'), 'expected the pairing branch to render the pairingMessage progress line').toBe(true);
		expect(
			stripped.includes('Scan accepted — completing login…'),
			'expected the scan-accepted progress copy to be present in the file, as a fixed literal'
		).toBe(true);
		expect(
			stripped.includes('Already linked — confirming this session…'),
			'expected the already-linked progress copy to be present in the file, as a fixed literal'
		).toBe(true);
	});

	it('the Cancel button guard lists loading and qr but not pairing — no cancel affordance during the post-pair window', () => {
		const cancelGuard = extractBetween(
			stripped,
			"{#if phase === 'loading' || phase === 'qr'}",
			'{/if}'
		);
		expect(cancelGuard.includes('handleCancel'), 'expected the located guard to be the Cancel button block').toBe(true);
		expect(
			cancelGuard.includes("phase === 'pairing'"),
			"expected the Cancel button's {#if} to NOT list the pairing phase — cancelling mid-login-handshake SIGKILLs a subprocess and strands an already-persisted pairing"
		).toBe(false);
	});
});

describe('the countdown line falls back to a waiting copy once it reaches zero without a replacement code (G-08-1)', () => {
	it('a derived countdown line carries both the "Refreshes in" shape and the "Waiting for a new code…" fallback', () => {
		expect(
			stripped.includes('Refreshes in'),
			'expected the countdown line to still render "Refreshes in {…}" while remainingSeconds is above zero'
		).toBe(true);
		expect(
			stripped.includes('Waiting for a new code…'),
			'expected the countdown line to fall back to "Waiting for a new code…" once the countdown reaches zero without a replacement code — a code that is not going to refresh must not claim it is about to'
		).toBe(true);
	});
});

describe('the three terminal cases still set retired = true and call clearTimers() (unchanged, pinned against this refactor)', () => {
	for (const [label, caseMarker, nextMarker] of [
		['paired', "case 'paired':", "\n\t\t\tcase 'error':"],
		['error', "case 'error':", "\n\t\t\tcase 'timeout':"],
		['timeout', "case 'timeout':", '\n\t\t\tdefault:']
	] as const) {
		it(`${label} sets retired = true and calls clearTimers()`, () => {
			const caseBody = extractBetween(stripped, caseMarker, nextMarker);
			expect(caseBody.includes('retired = true'), `expected the ${label} case to set retired = true`).toBe(true);
			expect(caseBody.includes('clearTimers()'), `expected the ${label} case to call clearTimers()`).toBe(true);
		});
	}
});
