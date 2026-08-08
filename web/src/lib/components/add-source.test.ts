// 07-04-PLAN.md Task 1's structural guard over AddSourceModal.svelte: the
// "+" trigger's aria-label and dashed-border styling, the picker's
// nothing-left-to-add copy, the popover content's height cap, and the
// one-step (existing-instance) modal's shape — MatchFieldsForm alone, no
// connection fields, a disabled-bindable submit control. Task 2 extends
// this same file (and this same test file) with the two-step new-instance
// flow; every one-step-modal assertion below is scoped via extractBetween
// to the EXISTING-instance Dialog block specifically (located by its
// `open={step === 'existing'}` binding, unique in the file), so these
// assertions keep holding unchanged once that second flow lands alongside
// it.
//
// House pattern (matches webspace-switcher.test.ts / source-chip-pill.test.ts):
// comment-stripped source scanning, a found-non-empty-source guard first,
// and one consequence-describing message per assertion.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const modalPath = join(here, 'AddSourceModal.svelte');

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

describe('add-source guard: found non-empty comment-stripped source', () => {
	it('AddSourceModal.svelte', () => {
		expect(stripped.length).toBeGreaterThan(0);
	});
});

describe('trigger: aria-label and dashed-border styling', () => {
	const triggerBlock = extractBetween(stripped, '<PopoverTrigger>', '</PopoverTrigger>');

	it('carries aria-label="Add source"', () => {
		expect(
			triggerBlock.includes('aria-label="Add source"'),
			'expected the "+" trigger to carry aria-label="Add source"'
		).toBe(true);
	});

	it('carries a dashed-border class', () => {
		expect(
			/border-dashed/.test(triggerBlock),
			'expected the "+" trigger to carry a dashed-border class, the deliberate "this reads as add-new" visual cue'
		).toBe(true);
	});
});

describe('picker: nothing-left-to-add copy', () => {
	it('renders the exact copy', () => {
		expect(
			stripped.includes('All available sources are already in this webspace.'),
			'expected the picker\'s empty branch to render the exact frozen copy'
		).toBe(true);
	});
});

describe('picker content: height-capped and scrollable', () => {
	it('the PopoverContent usage carries both a max-height and an overflow-y-auto class', () => {
		const popoverBlock = extractBetween(stripped, '<PopoverContent', '</PopoverContent>');
		expect(
			/max-h-\d+/.test(popoverBlock),
			'expected the picker popover to carry a max-height class so a long instance list cannot outgrow the viewport'
		).toBe(true);
		expect(
			/overflow-y-auto/.test(popoverBlock),
			'expected the picker popover to carry overflow-y-auto alongside its max-height class'
		).toBe(true);
	});
});

describe('one-step modal: MatchFieldsForm alone, no connection fields', () => {
	const existingDialogBlock = extractBetween(stripped, "open={step === 'existing'}", '</Dialog>');

	it('renders MatchFieldsForm', () => {
		expect(
			existingDialogBlock.includes('<MatchFieldsForm'),
			'expected the existing-instance modal to render MatchFieldsForm'
		).toBe(true);
	});

	it('renders no ConnectionForm (no connection fields — the instance already has them)', () => {
		expect(
			existingDialogBlock.includes('<ConnectionForm'),
			'expected the existing-instance modal to render no connection fields, since the instance already has them'
		).toBe(false);
	});

	it('the submit control carries a disabled binding', () => {
		expect(
			/type="submit"[^>]*disabled=\{/.test(existingDialogBlock),
			'expected the existing-instance modal\'s submit control to bind a disabled state (the shared save-in-flight pattern)'
		).toBe(true);
	});
});
