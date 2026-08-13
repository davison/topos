// 11-06-PLAN.md's structural guard over the binary-changed chip state, the
// re-pin menu action + confirmation dialog, and the pinned-hash footer
// (11-UI-SPEC.md E4/E5): Task 1 covers format.ts's healthTone/shortHash and
// SourceChip.svelte's isPinMismatch/isExternal derivations, the trust-update
// menu item, the Refresh now disable, and the pinned-hash footer. Task 2
// extends this same file with TrustUpdateDialog.svelte and the route's
// 'trust-update' wiring.
//
// House pattern (matches chip-edit-menu.test.ts / untrusted-add.test.ts):
// comment-stripped source scanning, `extractBetween` scoping, a
// found-non-empty-source guard first, and one consequence-describing
// message per assertion.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { healthTone, shortHash } from '../format';
import type { SourceStatus } from '../api';

const here = dirname(fileURLToPath(import.meta.url));
const chipPath = join(here, 'SourceChip.svelte');
const formatPath = join(here, '..', 'format.ts');

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

const rawChip = readFileSync(chipPath, 'utf-8');
const rawFormat = readFileSync(formatPath, 'utf-8');
const strippedChip = stripComments(rawChip);
const strippedFormat = stripComments(rawFormat);

describe('repin guard: found non-empty comment-stripped sources', () => {
	it('SourceChip.svelte', () => {
		expect(strippedChip.length).toBeGreaterThan(0);
	});
	it('format.ts', () => {
		expect(strippedFormat.length).toBeGreaterThan(0);
	});
});

// --- format.ts: healthTone's leading pin-mismatch branch, shortHash ---

function baseSource(overrides: Partial<SourceStatus> = {}): SourceStatus {
	return {
		name: 'demo',
		source_type: 'external-demo',
		display_name: 'Demo',
		plugin: 'topos-plugin-external-demo',
		tier: 'external',
		reachable: true,
		syncing: false,
		last_status: 'ok',
		last_sync_unix: 1700000000,
		last_error: '',
		...overrides
	};
}

describe('format.ts: healthTone branches on launch_failure before its existing tone logic', () => {
	it('returns destructive for a pin-mismatched source regardless of sync history (unreachable, never-synced, or healthy-looking)', () => {
		expect(healthTone(baseSource({ launch_failure: 'pin_mismatch', reachable: true }))).toBe(
			'destructive'
		);
		expect(
			healthTone(
				baseSource({ launch_failure: 'pin_mismatch', last_status: '', reachable: false })
			)
		).toBe('destructive');
		expect(
			healthTone(baseSource({ launch_failure: 'pin_mismatch', last_status: 'error' }))
		).toBe('destructive');
	});

	it('still returns the four pre-existing values for a source with no pin mismatch', () => {
		expect(healthTone(baseSource({ launch_failure: '', last_status: '' }))).toBe('unknown');
		expect(healthTone(baseSource({ launch_failure: '', reachable: false }))).toBe('destructive');
		expect(healthTone(baseSource({ launch_failure: '', last_status: 'error' }))).toBe('warning');
		expect(healthTone(baseSource({ launch_failure: '', last_status: 'ok' }))).toBe('success');
	});

	it('healthTone\'s source places the launch_failure check as the FIRST branch (extends the chain, never a parallel tone system)', () => {
		const fnBody = extractBetween(
			strippedFormat,
			'export function healthTone(source: SourceStatus): HealthTone {',
			'\n}'
		);
		const lines = fnBody
			.split('\n')
			.map((l) => l.trim())
			.filter((l) => l.length > 0);
		// lines[0] is the function signature itself (extractBetween includes
		// the start marker) — the first STATEMENT inside the function body
		// is lines[1].
		expect(
			lines[1].includes("launch_failure === 'pin_mismatch'"),
			'expected the launch_failure === pin_mismatch check to be the first statement inside healthTone'
		).toBe(true);
	});
});

describe('format.ts: shortHash — fixed 12-char-plus-ellipsis short form', () => {
	it('formats a 64-hex-char hash to its first 12 characters plus an ellipsis', () => {
		const full = 'a1b2c3d4e5f6' + '0'.repeat(52);
		expect(shortHash(full)).toBe('a1b2c3d4e5f6…');
	});

	it('returns the empty string unchanged for an empty hash — never a bare ellipsis', () => {
		expect(shortHash('')).toBe('');
	});
});

// --- SourceChip.svelte: isPinMismatch/isExternal, trust-update item, ---
// --- Refresh now disable, pinned-hash footer ---

describe('SourceChip.svelte: isPinMismatch/isExternal keyed on kernel-published fields', () => {
	it('derives isPinMismatch from source.launch_failure === \'pin_mismatch\', never a last_error match (T-11-32)', () => {
		expect(
			strippedChip.includes("let isPinMismatch = $derived(source.launch_failure === 'pin_mismatch')")
		).toBe(true);
	});

	it('derives isExternal from source.tier === \'external\'', () => {
		expect(strippedChip.includes("let isExternal = $derived(source.tier === 'external')")).toBe(
			true
		);
	});

	it('the {#if isPinMismatch} guard around the trust-update item references no last_error text (T-11-32)', () => {
		const guardIdx = strippedChip.indexOf('{#if isPinMismatch}');
		expect(guardIdx).toBeGreaterThanOrEqual(0);
		const guardBlock = strippedChip.slice(guardIdx, strippedChip.indexOf('{/if}', guardIdx));
		expect(guardBlock.includes('last_error')).toBe(false);
	});
});

describe('SourceChip.svelte: Trust updated binary… menu item (E4)', () => {
	const menuBlock = extractBetween(strippedChip, '<DropdownMenuContent>', '</DropdownMenuContent>');

	it('imports ShieldCheck from @lucide/svelte/icons/shield-check', () => {
		expect(
			strippedChip.includes("import ShieldCheck from '@lucide/svelte/icons/shield-check';")
		).toBe(true);
	});

	it('is wrapped in {#if isPinMismatch} — absent (not disabled) when the signal is unset', () => {
		const idx = menuBlock.indexOf('Trust updated binary…');
		expect(idx, 'expected to find the "Trust updated binary…" menu item').toBeGreaterThanOrEqual(0);
		const guardIdx = menuBlock.lastIndexOf('{#if isPinMismatch}', idx);
		const guardCloseIdx = menuBlock.indexOf('{/if}', idx);
		expect(guardIdx, 'expected the item to be preceded by {#if isPinMismatch}').toBeGreaterThanOrEqual(
			0
		);
		expect(guardCloseIdx, 'expected the {#if isPinMismatch} guard to close with {/if}').toBeGreaterThan(
			idx
		);
		// No `disabled=` attribute anywhere on this item — an absent item,
		// never a merely-disabled one (E4's own empty-state contract).
		const itemBlock = extractBetween(menuBlock.slice(guardIdx), '<DropdownMenuItem', '</DropdownMenuItem>');
		expect(itemBlock.includes('disabled')).toBe(false);
	});

	it('is the literal first DropdownMenuItem in the menu, calling onedit(source.name, \'trust-update\')', () => {
		const firstItemIndex = menuBlock.indexOf('<DropdownMenuItem');
		const firstItemBlock = menuBlock.slice(
			firstItemIndex,
			menuBlock.indexOf('</DropdownMenuItem>', firstItemIndex)
		);
		expect(firstItemBlock.includes('Trust updated binary')).toBe(true);
		expect(firstItemBlock.includes("onedit(source.name, 'trust-update')")).toBe(true);
	});

	it('uses the ShieldCheck icon, distinct from RefreshCw (sync) and any reload icon already in this menu', () => {
		const idx = menuBlock.indexOf('Trust updated binary…');
		const preceding = menuBlock.slice(Math.max(0, idx - 200), idx);
		expect(preceding.includes('<ShieldCheck')).toBe(true);
	});
});

describe('SourceChip.svelte: Refresh now disabled while isPinMismatch (E4)', () => {
	it('the Refresh now item\'s disabled expression references both source.syncing and isPinMismatch', () => {
		expect(strippedChip.includes('disabled={source.syncing || isPinMismatch}')).toBe(true);
		// The disabled expression precedes the item's own "Refresh now" text
		// in source order, confirming it is this item's own attribute.
		const disabledIdx = strippedChip.indexOf('disabled={source.syncing || isPinMismatch}');
		const refreshTextIdx = strippedChip.indexOf('Refresh now');
		expect(refreshTextIdx).toBeGreaterThan(disabledIdx);
	});
});

describe('SourceChip.svelte: pinned-hash footer (E5)', () => {
	const menuBlock = extractBetween(strippedChip, '<DropdownMenuContent>', '</DropdownMenuContent>');

	it('imports Copy and Check from @lucide/svelte', () => {
		expect(strippedChip.includes("import Copy from '@lucide/svelte/icons/copy';")).toBe(true);
		expect(strippedChip.includes("import Check from '@lucide/svelte/icons/check';")).toBe(true);
	});

	it('is gated on {#if isExternal && source.pinned_hash} — a static row, not a DropdownMenuItem', () => {
		expect(strippedChip.includes('{#if isExternal && source.pinned_hash}')).toBe(true);
		const guardIdx = strippedChip.indexOf('{#if isExternal && source.pinned_hash}');
		const footerBlock = strippedChip.slice(guardIdx, strippedChip.indexOf('{/if}', guardIdx));
		expect(
			footerBlock.includes('<DropdownMenuItem'),
			'expected the pinned-hash footer to render a plain div, never a selectable DropdownMenuItem'
		).toBe(false);
	});

	it('carries aria-label="Copy pinned hash" on the copy button', () => {
		expect(menuBlock.includes('aria-label="Copy pinned hash"')).toBe(true);
	});

	it('the displayed text uses shortHash(source.pinned_hash) while the title attribute carries the full value', () => {
		const guardIdx = strippedChip.indexOf('{#if isExternal && source.pinned_hash}');
		const footerBlock = strippedChip.slice(guardIdx, strippedChip.indexOf('{/if}', guardIdx));
		expect(
			footerBlock.includes('title={source.pinned_hash}'),
			'expected the title attribute to carry the full, unshortened hash'
		).toBe(true);
		expect(
			footerBlock.includes('Pinned: {shortHash(source.pinned_hash)}'),
			'expected the displayed text to use the shortened hash form'
		).toBe(true);
	});

	it('imports shortHash from $lib/format', () => {
		expect(/import\s*\{[^}]*shortHash[^}]*\}\s*from\s*'\$lib\/format';/.test(strippedChip)).toBe(
			true
		);
	});
});

describe('SourceChip.svelte: onedit kind union widened with \'trust-update\'', () => {
	it('the onedit prop type includes trust-update alongside the four pre-existing kinds', () => {
		expect(
			strippedChip.includes(
				"kind: 'connection' | 'match' | 'relink' | 'remove' | 'trust-update'"
			)
		).toBe(true);
	});
});
