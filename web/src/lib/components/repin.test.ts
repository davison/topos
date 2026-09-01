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
				baseSource({
					launch_failure: 'pin_mismatch',
					last_status: '',
					reachable: false
				})
			)
		).toBe('destructive');
		expect(healthTone(baseSource({ launch_failure: 'pin_mismatch', last_status: 'error' }))).toBe(
			'destructive'
		);
	});

	it('still returns the four pre-existing values for a source with no pin mismatch', () => {
		expect(healthTone(baseSource({ launch_failure: '', last_status: '' }))).toBe('unknown');
		expect(healthTone(baseSource({ launch_failure: '', reachable: false }))).toBe('destructive');
		expect(healthTone(baseSource({ launch_failure: '', last_status: 'error' }))).toBe('warning');
		expect(healthTone(baseSource({ launch_failure: '', last_status: 'ok' }))).toBe('success');
	});

	it("healthTone's source places the launch_failure check as the FIRST branch (extends the chain, never a parallel tone system)", () => {
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
	it("derives isPinMismatch from source.launch_failure === 'pin_mismatch', never a last_error match (T-11-32)", () => {
		expect(
			strippedChip.includes(
				"let isPinMismatch = $derived(source.launch_failure === 'pin_mismatch')"
			)
		).toBe(true);
	});

	it("derives isExternal from source.tier === 'external'", () => {
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
		expect(
			guardIdx,
			'expected the item to be preceded by {#if isPinMismatch}'
		).toBeGreaterThanOrEqual(0);
		expect(
			guardCloseIdx,
			'expected the {#if isPinMismatch} guard to close with {/if}'
		).toBeGreaterThan(idx);
		// No `disabled=` attribute anywhere on this item — an absent item,
		// never a merely-disabled one (E4's own empty-state contract).
		const itemBlock = extractBetween(
			menuBlock.slice(guardIdx),
			'<DropdownMenuItem',
			'</DropdownMenuItem>'
		);
		expect(itemBlock.includes('disabled')).toBe(false);
	});

	it("is the literal first DropdownMenuItem in the menu, calling onedit(source.name, 'trust-update')", () => {
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
	it("the Refresh now item's disabled expression references both source.syncing and isPinMismatch", () => {
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

describe("SourceChip.svelte: onedit kind union widened with 'trust-update'", () => {
	it('the onedit prop type includes trust-update alongside the four pre-existing kinds (and, since M2-R4, the two key consents)', () => {
		const union =
			/kind:\s*\|?\s*'connection'\s*\|\s*'match'\s*\|\s*'relink'\s*\|\s*'remove'\s*\|\s*'trust-update'\s*\|\s*'trust-key'\s*\|\s*'untrust-key'/;
		expect(union.test(strippedChip)).toBe(true);
	});
});

// --- Task 2: TrustUpdateDialog.svelte and the route's 'trust-update' wiring ---

const dialogPath = join(here, 'TrustUpdateDialog.svelte');
const routePath = join(here, '..', '..', 'routes', 'w', '[webspace]', '+page.svelte');
const rawDialog = readFileSync(dialogPath, 'utf-8');
const rawRoute = readFileSync(routePath, 'utf-8');
const strippedDialog = stripComments(rawDialog);
const strippedRoute = stripComments(rawRoute);

describe('repin guard (Task 2): found non-empty comment-stripped sources', () => {
	it('TrustUpdateDialog.svelte', () => {
		expect(strippedDialog.length).toBeGreaterThan(0);
	});
	it('+page.svelte', () => {
		expect(strippedRoute.length).toBeGreaterThan(0);
	});
});

describe('TrustUpdateDialog.svelte: E4 Copywriting Contract', () => {
	it('renders the exact dialog title', () => {
		expect(strippedDialog.includes('Binary changed')).toBe(true);
	});

	it('renders the exact body copy, with {source.plugin} as the binary name', () => {
		const normalized = strippedDialog.replace(/\s+/g, ' ');
		expect(
			normalized.includes(
				"{source.plugin} no longer matches the hash topos pinned when you added it. This can mean the binary was rebuilt, or that something else replaced it — topos can't tell which. Only continue if you trust this change."
			)
		).toBe(true);
	});

	it('renders the hash block lines', () => {
		expect(strippedDialog.includes('Previously pinned:')).toBe(true);
		expect(strippedDialog.includes('Currently on disk:')).toBe(true);
	});

	it('renders Cancel and Trust updated binary buttons', () => {
		expect(strippedDialog.includes('Cancel')).toBe(true);
		expect(strippedDialog.includes('Trust updated binary')).toBe(true);
	});
});

describe('TrustUpdateDialog.svelte: hash block — short previous, full+break-all current, no-previous-pin branch', () => {
	const hashBlock = extractBetween(strippedDialog, 'Previously pinned:', 'Currently on disk:');

	it('the previously-pinned line branches on source.pinned_hash — a "not pinned" reading when absent (edge: empty)', () => {
		expect(
			strippedDialog.includes(
				"Previously pinned: {source.pinned_hash ? shortHash(source.pinned_hash) : 'not pinned'}"
			)
		).toBe(true);
	});

	it('the currently-on-disk line renders the FULL hash with break-all so it wraps rather than overflows', () => {
		const currentlyBlockIdx = strippedDialog.indexOf('Currently on disk:');
		const surrounding = strippedDialog.slice(
			Math.max(0, currentlyBlockIdx - 100),
			currentlyBlockIdx
		);
		expect(surrounding.includes('break-all')).toBe(true);
		expect(strippedDialog.includes('Currently on disk: {source.current_hash')).toBe(true);
	});

	it('never renders a shortened form on the currently-on-disk line', () => {
		const currentlyBlockIdx = strippedDialog.indexOf('Currently on disk:');
		const line = strippedDialog.slice(currentlyBlockIdx, currentlyBlockIdx + 120);
		expect(line.includes('shortHash')).toBe(false);
	});

	it('imports shortHash from $lib/format for the previously-pinned line only', () => {
		expect(strippedDialog.includes("import { shortHash } from '$lib/format';")).toBe(true);
		expect(hashBlock.length).toBeGreaterThan(0);
	});
});

describe('TrustUpdateDialog.svelte: confirm handler — setPluginPin + putConfig exactly once each, keyed on the binary (D-02)', () => {
	const confirmBody = extractBetween(
		strippedDialog,
		'async function confirmTrustUpdate() {',
		'\n\t}'
	);

	it('imports setPluginPin from $lib/config-edit', () => {
		expect(strippedDialog.includes("import { setPluginPin } from '$lib/config-edit';")).toBe(true);
	});

	it('calls setPluginPin exactly once, keyed on source.plugin (the binary name, never the instance id — D-02)', () => {
		const calls = confirmBody.match(/setPluginPin\(/g) ?? [];
		expect(calls.length).toBe(1);
		expect(confirmBody.includes('setPluginPin(config, source.plugin, source.current_hash')).toBe(
			true
		);
	});

	it('passes the kernel-published source.current_hash, never a value it computes itself', () => {
		expect(/sha256|SHA-256|createHash/i.test(confirmBody)).toBe(false);
	});

	it('calls putConfig exactly once — the pin write and the save are the SAME network round trip', () => {
		const calls = confirmBody.match(/putConfig\(/g) ?? [];
		expect(calls.length).toBe(1);
	});

	it('the confirm button disables while saving is true', () => {
		expect(strippedDialog.includes('disabled={saving}')).toBe(true);
	});

	it('renders the destructive Alert + CONFIG_CONFLICT_MESSAGE pattern on failure', () => {
		expect(strippedDialog.includes('Alert variant="destructive"')).toBe(true);
		expect(strippedDialog.includes('CONFIG_CONFLICT_MESSAGE')).toBe(true);
	});
});

describe("+page.svelte: the 'trust-update' kind is handled and mounts TrustUpdateDialog", () => {
	it('imports TrustUpdateDialog', () => {
		expect(
			strippedRoute.includes(
				"import TrustUpdateDialog from '$lib/components/TrustUpdateDialog.svelte';"
			)
		).toBe(true);
	});

	it("handleChipEdit branches on kind === 'trust-update' before the describe path, exactly like 'relink'", () => {
		const handleChipEditBody = extractBetween(
			strippedRoute,
			'async function handleChipEdit(',
			'\n\t}'
		);
		const relinkIdx = handleChipEditBody.indexOf("kind === 'relink'");
		const trustUpdateIdx = handleChipEditBody.indexOf("kind === 'trust-update'");
		const describeIdx = handleChipEditBody.indexOf('describePlugin(');
		expect(
			trustUpdateIdx,
			"expected handleChipEdit to check kind === 'trust-update'"
		).toBeGreaterThanOrEqual(0);
		expect(relinkIdx).toBeGreaterThanOrEqual(0);
		expect(
			trustUpdateIdx,
			"expected the 'trust-update' branch to run strictly before the describePlugin( call"
		).toBeLessThan(describeIdx);
		expect(
			trustUpdateIdx,
			"expected 'trust-update' to be checked after 'relink', mirroring source order"
		).toBeGreaterThan(relinkIdx);
	});

	it("sets trustUpdateInstance = name inside the 'trust-update' branch", () => {
		const handleChipEditBody = extractBetween(
			strippedRoute,
			'async function handleChipEdit(',
			'\n\t}'
		);
		const branchStart = handleChipEditBody.indexOf("kind === 'trust-update'");
		const branch = handleChipEditBody.slice(branchStart, branchStart + 200);
		expect(branch.includes('trustUpdateInstance = name')).toBe(true);
	});

	it('mounts TrustUpdateDialog gated on configResponse && trustUpdateInstance && trustUpdateSource', () => {
		expect(
			strippedRoute.includes('{#if configResponse && trustUpdateInstance && trustUpdateSource}')
		).toBe(true);
	});

	it('TrustUpdateDialog is keyed on trustUpdateInstance — a genuinely different instance always remounts fresh', () => {
		const mountIdx = strippedRoute.indexOf(
			'{#if configResponse && trustUpdateInstance && trustUpdateSource}'
		);
		const afterMount = strippedRoute.slice(mountIdx, mountIdx + 120);
		expect(afterMount.includes('{#key trustUpdateInstance}')).toBe(true);
	});

	it('passes source={trustUpdateSource}, config, baseHash, onclose, onsaved', () => {
		const dialogTag = extractBetween(strippedRoute, '<TrustUpdateDialog', '/>');
		expect(dialogTag.includes('source={trustUpdateSource}')).toBe(true);
		expect(dialogTag.includes('config={configResponse.config}')).toBe(true);
		expect(dialogTag.includes('baseHash={configResponse.hash}')).toBe(true);
		expect(dialogTag.includes('onclose={handleTrustUpdateClose}')).toBe(true);
		expect(dialogTag.includes('onsaved={handleTrustUpdateSaved}')).toBe(true);
	});

	it('handleTrustUpdateSaved clears trustUpdateInstance and refreshes config/sources/stream (D-07 eager reconcile)', () => {
		const fnBody = extractBetween(
			strippedRoute,
			'async function handleTrustUpdateSaved() {',
			'\n\t}'
		);
		expect(fnBody.includes('trustUpdateInstance = null')).toBe(true);
		expect(fnBody.includes('loadConfig(navGeneration)')).toBe(true);
		expect(fnBody.includes('loadSources()')).toBe(true);
		expect(fnBody.includes('load(navGeneration)')).toBe(true);
	});
});
