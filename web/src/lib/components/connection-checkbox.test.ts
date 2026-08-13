// 12-04-PLAN.md Task 2 (12-UI-SPEC.md F1): ConnectionForm.svelte's third
// primary-fields render branch, alongside the existing secret and
// plain-text branches — the checkbox field kind's own markup/behavior
// contract (checked/unchecked coercion, the boolean-widened setField, the
// helper-text paragraph, and the whole-row clickable target).
//
// House pattern (matches secret-field.test.ts / extras-form.test.ts): this
// repo has no Svelte component mount harness (web/vite.config.ts's vitest
// environment is 'node' — no jsdom, no @testing-library/svelte). Component
// logic is verified by comment-stripped source scanning instead of DOM
// mounting/interaction.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const componentPath = join(here, 'ConnectionForm.svelte');

function stripComments(source: string): string {
	return source
		.replace(/<!--[\s\S]*?-->/g, ' ')
		.replace(/\/\*[\s\S]*?\*\//g, ' ')
		.replace(/\/\/.*$/gm, ' ');
}

function extractBetween(source: string, startMarker: string, endMarker: string): string {
	const startIndex = source.indexOf(startMarker);
	expect(startIndex, `expected to find "${startMarker}" in the scanned source`).toBeGreaterThanOrEqual(
		0
	);
	const endIndex = source.indexOf(endMarker, startIndex);
	expect(endIndex, `expected to find "${endMarker}" after "${startMarker}"`).toBeGreaterThan(
		startIndex
	);
	return source.slice(startIndex, endIndex + endMarker.length);
}

const raw = readFileSync(componentPath, 'utf-8');
const stripped = stripComments(raw);

describe('connection-checkbox guard: found a non-empty comment-stripped source', () => {
	it('ConnectionForm.svelte', () => {
		expect(stripped.length).toBeGreaterThan(0);
	});
});

// --- unchecked/checked coercion: a checkbox field with no or a non-boolean stored value renders unchecked ---

describe('boolFieldValue: coerces an unset or non-boolean stored value to false, never throws, never indeterminate', () => {
	const fnBody = extractBetween(stripped, 'function boolFieldValue(field: ConnectionField): boolean {', '\n\t}');

	it('reads values[field.key] and returns it only when it is already a boolean', () => {
		expect(fnBody.includes("typeof raw === 'boolean' ? raw : false")).toBe(true);
	});

	it('never references an "indeterminate" branch', () => {
		expect(/indeterminate/i.test(fnBody)).toBe(false);
	});
});

// --- setField widened to accept a boolean as well as a string ---

describe('setField: widened to accept string | boolean, still spreads the full values object', () => {
	const fnBody = extractBetween(stripped, 'function setField(', '\n\t}');

	it('accepts value: string | boolean', () => {
		expect(fnBody.includes('value: string | boolean')).toBe(true);
	});

	it('spreads ...values before overwriting the single key — every other key stays untouched', () => {
		expect(fnBody.includes('onchange({ ...values, [key]: value })')).toBe(true);
	});
});

// --- the checkbox render branch: markup shape, per 12-UI-SPEC.md F1 ---

describe("the checkbox branch: {:else if field.kind === 'checkbox'}", () => {
	const branchStart = stripped.indexOf("{:else if field.kind === 'checkbox'}");

	it('exists as a third branch in the primary-fields loop', () => {
		expect(branchStart).toBeGreaterThanOrEqual(0);
	});

	const branchBlock = extractBetween(
		stripped,
		"{:else if field.kind === 'checkbox'}",
		'{/each}'
	);

	it('wraps the Checkbox and label text in a single <label for=...> — the whole row is the clickable target, not only the control box', () => {
		expect(branchBlock.includes('for={`conn-${field.key}`}')).toBe(true);
		const labelOpenIdx = branchBlock.indexOf('<label');
		const checkboxIdx = branchBlock.indexOf('<Checkbox');
		const labelCloseIdx = branchBlock.indexOf('</label>');
		expect(labelOpenIdx).toBeGreaterThanOrEqual(0);
		expect(checkboxIdx).toBeGreaterThan(labelOpenIdx);
		expect(labelCloseIdx).toBeGreaterThan(checkboxIdx);
	});

	it('the wrapping label carries min-h-11 (the 44px touch-target floor) alongside the existing 14px label typography', () => {
		expect(
			branchBlock.includes('class="flex min-h-11 items-center gap-2 text-[14px] leading-[1.4] text-foreground"')
		).toBe(true);
	});

	it('binds Checkbox.checked to boolFieldValue(field) and onCheckedChange to setField(field.key, ...)', () => {
		expect(branchBlock.includes('checked={boolFieldValue(field)}')).toBe(true);
		expect(branchBlock.includes('onCheckedChange={(v) => setField(field.key, v)}')).toBe(true);
	});

	it('renders field.label inside the wrapping label', () => {
		expect(/<label[\s\S]*?\{field\.label\}[\s\S]*?<\/label>/.test(branchBlock)).toBe(true);
	});

	it('a field declaring helper text renders it in a muted paragraph gated on field.helperText; a field without it renders no helper paragraph', () => {
		expect(branchBlock.includes('{#if field.helperText}')).toBe(true);
		expect(
			branchBlock.includes('<p class="text-[14px] leading-[1.4] text-muted-foreground">{field.helperText}</p>')
		).toBe(true);
	});
});

// --- Checkbox primitive is imported ---

describe('ConnectionForm.svelte imports the shared Checkbox primitive', () => {
	it("imports Checkbox from '$lib/components/ui/checkbox/index.js'", () => {
		expect(stripped.includes("import { Checkbox } from '$lib/components/ui/checkbox/index.js';")).toBe(
			true
		);
	});
});
