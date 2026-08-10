// 08-04-PLAN.md Task 1's structural guard over QRPanel.svelte: all five
// states (loading/qr/error/expired/success) render a distinct branch, the
// exact instruction line the UI-SPEC amendment locks, and both the
// explicit-cancel and unmount paths invoke cancelWhatsAppLink (T-08-10's
// mitigation, pinned structurally since this component's real network
// behaviour needs a mounted-component harness this house pattern doesn't
// use).
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

describe('all five states render a distinct branch', () => {
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
		const qrBranch = extractBetween(stripped, "{:else if phase === 'qr'}", "{:else if phase === 'error'}");
		expect(qrBranch.includes('<img'), 'expected the qr branch to render the pairing image').toBe(true);
		expect(
			qrBranch.includes('Scan with your phone to link'),
			'expected the qr branch to render the exact frozen instruction line'
		).toBe(true);
		expect(
			/Refreshes in \{formatCountdown\(remainingSeconds\)\}/.test(qrBranch),
			'expected the qr branch to render a "Refreshes in M:SS" countdown line driven by remainingSeconds'
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
		const successBranch = extractBetween(stripped, "{:else if phase === 'success'}", '{/if}\n\n\t{#if phase');
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

describe('the poll cadence is clamped to a floor (T-08-10, request-storm guard)', () => {
	it('schedulePoll clamps its delay against POLL_FLOOR_MS', () => {
		const schedulePollBody = extractBetween(stripped, 'function schedulePoll(delayMs: number) {', '\n\t}');
		expect(
			/Math\.max\(delayMs,\s*POLL_FLOOR_MS\)/.test(schedulePollBody),
			'expected schedulePoll to clamp its delay to at least POLL_FLOOR_MS, so a short or malformed expires_in_seconds cannot produce a request storm'
		).toBe(true);
	});
});
