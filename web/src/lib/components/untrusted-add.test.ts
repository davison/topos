// 11-05-PLAN.md Task 1's structural guard over AddSourceModal.svelte's
// tier-aware picker rows and the untrusted-source confirm step (E1/E2/E3):
// the 'untrusted-confirm' step value, the picker's TrustBadge + "Untrusted"
// label usage in both groups, the confirm step's exact E1 copy and
// disabled-until-exact-match binding, the Save-anyway exclusion for
// external-tier plugin types, and setPluginPin's single call site inside
// submitMatch.
//
// House pattern (matches add-source.test.ts / webspace-switcher.test.ts):
// comment-stripped source scanning, a found-non-empty-source guard first,
// and one consequence-describing message per assertion.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const modalPath = join(here, 'AddSourceModal.svelte');
const configEditPath = join(here, '..', 'config-edit.ts');
const pluginFieldsPath = join(here, '..', 'plugin-fields.ts');

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

const raw = readFileSync(modalPath, 'utf-8');
const stripped = stripComments(raw);
const configEditStripped = stripComments(readFileSync(configEditPath, 'utf-8'));
const pluginFieldsStripped = stripComments(readFileSync(pluginFieldsPath, 'utf-8'));

describe('untrusted-add guard: found non-empty comment-stripped sources', () => {
	it('AddSourceModal.svelte', () => {
		expect(stripped.length).toBeGreaterThan(0);
	});
	it('config-edit.ts', () => {
		expect(configEditStripped.length).toBeGreaterThan(0);
	});
	it('plugin-fields.ts', () => {
		expect(pluginFieldsStripped.length).toBeGreaterThan(0);
	});
});

describe('plugin-fields.ts: isExternalTier and UNTRUSTED_LABEL exports', () => {
	it('exports isExternalTier', () => {
		expect(pluginFieldsStripped.includes('export function isExternalTier(')).toBe(true);
	});
	it('exports UNTRUSTED_LABEL set to "Untrusted"', () => {
		expect(
			/export const UNTRUSTED_LABEL = 'Untrusted'/.test(pluginFieldsStripped),
			'expected UNTRUSTED_LABEL to carry the exact E3 picker label copy'
		).toBe(true);
	});
});

describe('config-edit.ts: setPluginPin export', () => {
	it('exports setPluginPin(cfg, pluginBinary, hash)', () => {
		expect(
			/export function setPluginPin\(\s*cfg: KernelConfig,\s*pluginBinary: string,\s*hash: string\s*\): KernelConfig/.test(
				configEditStripped
			),
			'expected setPluginPin to accept (cfg, pluginBinary, hash) and return a KernelConfig'
		).toBe(true);
	});

	it('never computes or hashes anything itself — only ever assigns the passed-in hash verbatim', () => {
		const fnBody = extractBetween(
			configEditStripped,
			'export function setPluginPin(',
			'\n}'
		);
		expect(
			fnBody.includes('[pluginBinary]: hash'),
			'expected setPluginPin to write the hash argument verbatim under the pluginBinary key'
		).toBe(true);
		expect(
			/sha256|SHA-256|createHash/i.test(fnBody),
			'expected setPluginPin to contain no hashing logic of its own (T-11-25)'
		).toBe(false);
	});
});

describe("AddSourceModal.svelte: 'untrusted-confirm' step value", () => {
	it('is declared in the step union', () => {
		expect(
			stripped.includes("'untrusted-confirm'"),
			"expected the step state union to declare 'untrusted-confirm'"
		).toBe(true);
	});

	it('is included in the two-step Dialog\'s open binding', () => {
		const dialogOpenExpr = extractBetween(stripped, '<Dialog', 'onOpenChange={handleConnectOpenChange}');
		expect(
			dialogOpenExpr.includes("step === 'untrusted-confirm'"),
			"expected the two-step modal's open binding to include step === 'untrusted-confirm'"
		).toBe(true);
	});

	it('is rendered as its own branch inside the two-step Dialog', () => {
		expect(stripped.includes("{:else if step === 'untrusted-confirm'}")).toBe(true);
	});
});

describe('picker: TrustBadge usage in both groups (E2), scale="picker"', () => {
	it('imports TrustBadge', () => {
		expect(stripped.includes("import TrustBadge from './TrustBadge.svelte';")).toBe(true);
	});

	it('renders TrustBadge with scale="picker" at least twice (Group 1 rows and Group 2 tiles)', () => {
		const matches = stripped.match(/<TrustBadge\s+tier=\{[^}]+\}\s+scale="picker">/g) ?? [];
		expect(
			matches.length,
			'expected at least two <TrustBadge ... scale="picker"> usages — one for Group 1 instance rows, one for Group 2 catalog tiles'
		).toBeGreaterThanOrEqual(2);
	});

	it('derives Group 1 rows\' tier from isExternalTier(pluginTypeTiers, source.plugin)', () => {
		expect(stripped.includes('isExternalTier(pluginTypeTiers, source.plugin)')).toBe(true);
	});

	it('derives Group 2 tiles\' tier from isExternalTier(pluginTypeTiers, plugin)', () => {
		expect(stripped.includes('isExternalTier(pluginTypeTiers, plugin)')).toBe(true);
	});
});

describe('picker: "Untrusted" label (E3) in both groups', () => {
	it('renders UNTRUSTED_LABEL conditionally on the untrusted flag, not unconditionally', () => {
		const matches = stripped.match(/\{#if untrusted\}[\s\S]*?\{UNTRUSTED_LABEL\}[\s\S]*?\{\/if\}/g) ?? [];
		expect(
			matches.length,
			'expected UNTRUSTED_LABEL to render inside its own {#if untrusted} guard at least twice (Group 1 and Group 2)'
		).toBeGreaterThanOrEqual(2);
	});

	it('the picker still renders exactly two group headers — no third, untrusted-only section (D-07)', () => {
		const addHeaderMatches = stripped.match(/Add to this webspace/g) ?? [];
		const installHeaderMatches = stripped.match(/Install a new source/g) ?? [];
		expect(addHeaderMatches.length, 'expected the Group 1 header to appear exactly once').toBe(1);
		expect(
			installHeaderMatches.length,
			'expected the Group 2 header to appear exactly once'
		).toBe(1);
		expect(
			/untrusted plugins|external plugins/i.test(stripped),
			'expected no separate "untrusted"/"external" section header — external plugins list inline in the existing catalog section (D-07)'
		).toBe(false);
	});

	it('Group 2\'s tile widens to a justify-between row so the label right-aligns', () => {
		expect(stripped.includes('flex items-center justify-between gap-1.5')).toBe(true);
		expect(stripped.includes('ml-auto')).toBe(true);
	});
});

describe('E1 — untrusted-confirm step: exact Copywriting Contract strings', () => {
	const confirmBranch = extractBetween(
		stripped,
		"{:else if step === 'untrusted-confirm'}",
		"{:else if step === 'match'}"
	);

	it('renders the exact dialog title', () => {
		expect(confirmBranch.includes('Add an untrusted source')).toBe(true);
	});

	it('renders the exact explanation copy', () => {
		// Markup line-wraps the paragraph across several template lines —
		// normalize runs of whitespace to a single space before matching,
		// same discipline a reader applies when the source is re-wrapped by
		// a formatter.
		const normalized = confirmBranch.replace(/\s+/g, ' ');
		expect(
			normalized.includes(
				"lives outside topos's own plugin directory — this is code topos didn't build and can't vouch for. It runs with the same access as any other plugin process; topos does not sandbox it."
			)
		).toBe(true);
	});

	it('renders the binary and hash info-block lines', () => {
		expect(confirmBranch.includes('Binary: {selectedPluginType}')).toBe(true);
		expect(confirmBranch.includes('Pinned hash (SHA-256): {pendingBinaryHash}')).toBe(true);
	});

	it('the hash line carries font-mono and break-all so it wraps rather than overflows', () => {
		const hashLineIdx = confirmBranch.indexOf('Pinned hash (SHA-256)');
		expect(hashLineIdx).toBeGreaterThanOrEqual(0);
		const surrounding = confirmBranch.slice(Math.max(0, hashLineIdx - 200), hashLineIdx);
		expect(/font-mono/.test(surrounding)).toBe(true);
		expect(/break-all/.test(surrounding)).toBe(true);
	});

	it('renders both env-disclosure copy branches (zero vs. one-or-many referenced vars)', () => {
		expect(
			confirmBranch.includes(
				'topos will hand this plugin only the standard PATH/HOME/locale environment'
			),
			'expected the zero-referenced-vars branch'
		).toBe(true);
		expect(
			confirmBranch.includes(
				'topos will also hand this plugin the standard PATH/HOME/locale environment'
			),
			'expected the one-or-many-referenced-vars branch'
		).toBe(true);
		expect(
			confirmBranch.includes('{#if pendingEnvVarNames.length === 0}'),
			'expected the two branches to be gated on pendingEnvVarNames.length'
		).toBe(true);
	});

	it('renders the exact type-to-confirm label', () => {
		expect(confirmBranch.includes('Type {selectedPluginType} to confirm')).toBe(true);
	});

	it('renders Cancel and Add untrusted source buttons', () => {
		expect(confirmBranch.includes('Cancel')).toBe(true);
		expect(confirmBranch.includes('Add untrusted source')).toBe(true);
	});
});

describe('E1 — confirm step: disabled-until-exact-match binding', () => {
	const confirmBranch = extractBetween(
		stripped,
		"{:else if step === 'untrusted-confirm'}",
		"{:else if step === 'match'}"
	);

	it('the primary action\'s disabled expression compares confirmTyped to selectedPluginType for exact equality', () => {
		expect(
			confirmBranch.includes('disabled={confirmTyped !== selectedPluginType}'),
			'expected the primary Add untrusted source button to disable unless confirmTyped exactly equals selectedPluginType'
		).toBe(true);
	});

	it('confirmUntrusted() itself re-checks the same equality before advancing to match', () => {
		const fnBody = extractBetween(stripped, 'function confirmUntrusted() {', '\n\t}');
		expect(fnBody.includes('confirmTyped !== selectedPluginType')).toBe(true);
		expect(fnBody.includes("step = 'match'")).toBe(true);
	});

	it('cancelUntrustedConfirm returns to connect and clears only confirmTyped, leaving connectionValues untouched', () => {
		const fnBody = extractBetween(stripped, 'function cancelUntrustedConfirm() {', '\n\t}');
		expect(fnBody.includes("step = 'connect'")).toBe(true);
		expect(fnBody.includes("confirmTyped = ''")).toBe(true);
		expect(
			fnBody.includes('connectionValues ='),
			'expected cancelUntrustedConfirm to leave connectionValues untouched — cancelling must preserve every typed connection value'
		).toBe(false);
	});
});

describe('handleConnectNext: routes an external-tier plugin type to untrusted-confirm', () => {
	const handleConnectNextBody = extractBetween(
		stripped,
		'async function handleConnectNext(event: SubmitEvent) {',
		'\n\t}'
	);

	it('captures resp.binary_hash and resp.env_var_names into pendingBinaryHash/pendingEnvVarNames', () => {
		expect(handleConnectNextBody.includes('pendingBinaryHash = resp.binary_hash')).toBe(true);
		expect(handleConnectNextBody.includes('pendingEnvVarNames = resp.env_var_names')).toBe(true);
	});

	it("routes to step = 'untrusted-confirm' when isExternalTier(pluginTypeTiers, selectedPluginType)", () => {
		expect(
			handleConnectNextBody.includes('isExternalTier(pluginTypeTiers, selectedPluginType)')
		).toBe(true);
		expect(handleConnectNextBody.includes("step = 'untrusted-confirm'")).toBe(true);
	});

	it('the WhatsApp link branch is checked first, so it is unaffected by the external-tier branch', () => {
		const whatsappIdx = handleConnectNextBody.indexOf(
			'selectedPluginType === WHATSAPP_PLUGIN_BINARY'
		);
		const externalIdx = handleConnectNextBody.indexOf(
			'isExternalTier(pluginTypeTiers, selectedPluginType)'
		);
		expect(whatsappIdx).toBeGreaterThanOrEqual(0);
		expect(externalIdx).toBeGreaterThan(whatsappIdx);
	});
});

describe('Save anyway is excluded for an external-tier plugin type (T-11-27)', () => {
	it('the Save anyway control\'s guard also checks isExternalTier(pluginTypeTiers, selectedPluginType)', () => {
		const connectDialogBlock = extractBetween(
			stripped,
			"open={step === 'connect' ||",
			'</Dialog>'
		);
		const connectBranch = extractBetween(
			connectDialogBlock,
			"{#if step === 'connect'}",
			"{:else if step === 'link'}"
		);
		expect(
			connectBranch.includes(
				"{#if describeFailed && !(selectedPluginType && isExternalTier(pluginTypeTiers, selectedPluginType))}"
			),
			'expected the Save anyway control to be wrapped in a condition excluding external-tier plugin types'
		).toBe(true);
	});
});

describe('submitMatch: setPluginPin is called only here, riding the single putConfig call', () => {
	it('imports setPluginPin from config-edit', () => {
		expect(
			stripped.includes('setPluginPin') && stripped.includes("from '$lib/config-edit';")
		).toBe(true);
	});

	it('setPluginPin appears exactly once in the whole file, inside submitMatch', () => {
		const allCalls = stripped.match(/setPluginPin\(/g) ?? [];
		expect(allCalls.length, 'expected exactly one setPluginPin( call site').toBe(1);

		const submitMatchBody = extractBetween(
			stripped,
			'async function submitMatch(event: SubmitEvent) {',
			'\n\t}'
		);
		expect(submitMatchBody.includes('setPluginPin(')).toBe(true);
	});

	it('submitMatch still calls putConfig exactly once — the pin rides the existing save, no second round trip', () => {
		const submitMatchBody = extractBetween(
			stripped,
			'async function submitMatch(event: SubmitEvent) {',
			'\n\t}'
		);
		const putConfigCalls = submitMatchBody.match(/putConfig\(/g) ?? [];
		expect(putConfigCalls.length).toBe(1);
	});

	it('setPluginPin is called with pendingBinaryHash, never a locally-computed value', () => {
		const submitMatchBody = extractBetween(
			stripped,
			'async function submitMatch(event: SubmitEvent) {',
			'\n\t}'
		);
		expect(submitMatchBody.includes('setPluginPin(withMatch, selectedPluginType, pendingBinaryHash)')).toBe(
			true
		);
	});
});
