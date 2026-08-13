// 11-05-PLAN.md Task 2's structural guard over ConnectionForm.svelte's E6
// extras section (declared fields + free-form editor) and plugin-fields.ts's
// extrasToRows/rowsToExtras/extrasKeyError helpers: the "Provider-specific
// settings"/"Additional fields" labels, the declared-field placeholder-vs-
// value binding (D-14), the free-form row shape/placeholders/remove
// control, the "Add field" button, and both AddSourceModal/EditSourceModal
// passing extrasFields into the same unforked ConnectionForm.
//
// House pattern (matches add-source.test.ts / untrusted-add.test.ts):
// comment-stripped source scanning, a found-non-empty-source guard first,
// and one consequence-describing message per assertion, plus direct unit
// tests of the pure plugin-fields.ts helpers.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { extrasToRows, rowsToExtras, extrasKeyError } from '$lib/plugin-fields';
import type { ExtrasFieldDecl } from '$lib/api';

const here = dirname(fileURLToPath(import.meta.url));
const connectionFormPath = join(here, 'ConnectionForm.svelte');
const addModalPath = join(here, 'AddSourceModal.svelte');
const editModalPath = join(here, 'EditSourceModal.svelte');

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

const connectionFormStripped = stripComments(readFileSync(connectionFormPath, 'utf-8'));
const addModalStripped = stripComments(readFileSync(addModalPath, 'utf-8'));
const editModalStripped = stripComments(readFileSync(editModalPath, 'utf-8'));

describe('extras-form guard: found non-empty comment-stripped sources', () => {
	it('ConnectionForm.svelte', () => {
		expect(connectionFormStripped.length).toBeGreaterThan(0);
	});
	it('AddSourceModal.svelte', () => {
		expect(addModalStripped.length).toBeGreaterThan(0);
	});
	it('EditSourceModal.svelte', () => {
		expect(editModalStripped.length).toBeGreaterThan(0);
	});
});

// --- plugin-fields.ts: pure helper unit tests ---

describe('extrasToRows', () => {
	it('excludes declared keys, keeping only undeclared saved keys as rows', () => {
		const declared: ExtrasFieldDecl[] = [
			{ key: 'folder_id', label: 'Folder ID', required: false, secret: false, placeholder: '' }
		];
		const rows = extrasToRows({ folder_id: 'abc', workspace_id: 'acme-42' }, declared);
		expect(rows).toEqual([{ key: 'workspace_id', value: 'acme-42' }]);
	});

	it('returns rows in stable alphabetical order', () => {
		const rows = extrasToRows({ zeta: '1', alpha: '2' }, []);
		expect(rows.map((r) => r.key)).toEqual(['alpha', 'zeta']);
	});

	it('returns [] for undefined extras', () => {
		expect(extrasToRows(undefined, [])).toEqual([]);
	});

	it('a saved key no longer declared still surfaces as a row (D-15: never invisible)', () => {
		const rows = extrasToRows({ retired_key: 'still-here' }, []);
		expect(rows).toEqual([{ key: 'retired_key', value: 'still-here' }]);
	});
});

describe('rowsToExtras', () => {
	it('composes declared (non-blank) values and free-form rows into one map', () => {
		const result = rowsToExtras({ folder_id: 'abc', blank_declared: '' }, [
			{ key: 'workspace_id', value: 'acme-42' }
		]);
		expect(result).toEqual({ folder_id: 'abc', workspace_id: 'acme-42' });
	});

	it('omits a blank (non-required) declared value entirely', () => {
		const result = rowsToExtras({ folder_id: '' }, []);
		expect(result).toEqual({});
	});

	it('trims a free-form row key before writing it', () => {
		const result = rowsToExtras({}, [{ key: '  spaced_key  ', value: 'v' }]);
		expect(result).toEqual({ spaced_key: 'v' });
	});

	it('drops a free-form row with an empty/whitespace-only key', () => {
		const result = rowsToExtras({}, [{ key: '   ', value: 'v' }]);
		expect(result).toEqual({});
	});
});

describe('extrasKeyError', () => {
	const declared: ExtrasFieldDecl[] = [
		{ key: 'folder_id', label: 'Folder ID', required: false, secret: false, placeholder: '' }
	];

	it('returns null for zero rows', () => {
		expect(extrasKeyError(declared, [])).toBeNull();
	});

	it('returns null for distinct, non-empty, non-declared-colliding keys', () => {
		expect(
			extrasKeyError(declared, [
				{ key: 'workspace_id', value: 'acme-42' },
				{ key: 'region', value: 'eu' }
			])
		).toBeNull();
	});

	it('returns the fixed copy for an empty key', () => {
		expect(extrasKeyError(declared, [{ key: '', value: 'v' }])).toBe(
			'Every extra field needs a unique key.'
		);
	});

	it('returns the fixed copy for a whitespace-only key', () => {
		expect(extrasKeyError(declared, [{ key: '   ', value: 'v' }])).toBe(
			'Every extra field needs a unique key.'
		);
	});

	it('returns the fixed copy for two free-form rows sharing a key', () => {
		expect(
			extrasKeyError(declared, [
				{ key: 'dup', value: 'a' },
				{ key: 'dup', value: 'b' }
			])
		).toBe('Every extra field needs a unique key.');
	});

	it('returns the fixed copy for a free-form row colliding with a declared key', () => {
		expect(extrasKeyError(declared, [{ key: 'folder_id', value: 'x' }])).toBe(
			'Every extra field needs a unique key.'
		);
	});
});

// --- ConnectionForm.svelte: E6 markup structure ---

describe('ConnectionForm.svelte: E6 extras section markup', () => {
	it('renders the "Provider-specific settings" label', () => {
		expect(connectionFormStripped.includes('Provider-specific settings')).toBe(true);
	});

	it('renders the "Additional fields" sub-label', () => {
		expect(connectionFormStripped.includes('Additional fields')).toBe(true);
	});

	it('"Provider-specific settings" is gated on extrasFields.length, "Additional fields" is not', () => {
		const providerIdx = connectionFormStripped.indexOf('Provider-specific settings');
		const before = connectionFormStripped.slice(0, providerIdx);
		const guardIdx = before.lastIndexOf('{#if extrasFields.length > 0}');
		expect(
			guardIdx,
			'expected "Provider-specific settings" to be preceded by an {#if extrasFields.length > 0} guard'
		).toBeGreaterThanOrEqual(0);

		const additionalIdx = connectionFormStripped.indexOf('Additional fields');
		const providerBlockEnd = connectionFormStripped.indexOf('{/if}', providerIdx);
		expect(
			additionalIdx,
			'expected "Additional fields" to render AFTER the gated declared-fields block closes, unconditionally'
		).toBeGreaterThan(providerBlockEnd);
	});

	it('renders free-form row inputs with the exact Key/Value placeholders', () => {
		expect(connectionFormStripped.includes('placeholder="Key"')).toBe(true);
		expect(connectionFormStripped.includes('placeholder="Value"')).toBe(true);
	});

	it('renders a remove control with aria-label="Remove field" and Trash2 at size-3.5', () => {
		expect(connectionFormStripped.includes('aria-label="Remove field"')).toBe(true);
		expect(connectionFormStripped.includes('<Trash2 class="size-3.5" aria-hidden="true" />')).toBe(
			true
		);
	});

	it('renders an "Add field" button', () => {
		expect(connectionFormStripped.includes('Add field')).toBe(true);
	});

	it('exports extrasFields and a bindable extrasRows prop', () => {
		expect(connectionFormStripped.includes('extrasFields = []')).toBe(true);
		expect(connectionFormStripped.includes('extrasRows = $bindable([])')).toBe(true);
	});
});

describe('ConnectionForm.svelte: declared field placeholder-vs-value binding (D-14)', () => {
	const extrasBlock = extractBetween(
		connectionFormStripped,
		'{#if extrasFields.length > 0}',
		'Additional fields'
	);

	it("the declared field's Input binds placeholder to field.placeholder", () => {
		expect(/placeholder=\{field\.placeholder\}/.test(extrasBlock)).toBe(true);
	});

	it("the declared field's Input value binds to local declaredExtrasValues state, never to field.placeholder or a 'default'", () => {
		expect(
			/value=\{declaredExtrasValues\[field\.key\] \?\? ''\}/.test(extrasBlock),
			'expected the declared field Input to read its value from declaredExtrasValues, not from the declaration itself'
		).toBe(true);
		expect(
			/value=\{field\.(placeholder|default)/.test(extrasBlock),
			'expected NO value binding anywhere in the extras block to read a declaration field directly (D-14: a plugin-suggested default is display-only)'
		).toBe(false);
	});

	it('a declared secret-ish field renders through SecretField, unwrapping/wrapping exactly like a secret connection field', () => {
		expect(extrasBlock.includes('<SecretField')).toBe(true);
		expect(extrasBlock.includes('setDeclaredExtra(field.key, wrapVar(name))')).toBe(true);
		expect(extrasBlock.includes("unwrapVar(declaredExtrasValues[field.key] ?? '')")).toBe(true);
	});

	it('commitExtras composes via rowsToExtras — declaredExtrasValues and extrasRows, never a raw values.extras mutation', () => {
		const fnBody = extractBetween(connectionFormStripped, 'function commitExtras() {', '\n\t}');
		expect(fnBody.includes('rowsToExtras(declaredExtrasValues, extrasRows)')).toBe(true);
	});
});

describe('AddSourceModal.svelte and EditSourceModal.svelte: both pass extrasFields into the same unforked ConnectionForm', () => {
	it('AddSourceModal renders <ConnectionForm ... extrasFields={declaredExtras} bind:extrasRows', () => {
		const matches =
			addModalStripped.match(/<ConnectionForm[\s\S]*?extrasFields=\{declaredExtras\}[\s\S]*?bind:extrasRows/g) ??
			[];
		expect(
			matches.length,
			'expected every ConnectionForm usage in AddSourceModal.svelte to pass extrasFields={declaredExtras} and bind:extrasRows'
		).toBeGreaterThanOrEqual(1);
	});

	it('EditSourceModal renders <ConnectionForm ... extrasFields={extrasFields}', () => {
		expect(editModalStripped.includes('<ConnectionForm')).toBe(true);
		const connectionFormBlock = extractBetween(editModalStripped, '<ConnectionForm', '/>');
		expect(connectionFormBlock.includes('{extrasFields}')).toBe(true);
	});

	it('EditSourceModal declares an extrasFields prop of type ExtrasFieldDecl[]', () => {
		expect(editModalStripped.includes('extrasFields: ExtrasFieldDecl[]')).toBe(true);
	});
});
