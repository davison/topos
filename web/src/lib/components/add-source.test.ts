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

describe('two-step modal: step indicator reads both "1. Connect" and "2. Match"', () => {
	const connectDialogBlock = extractBetween(
		stripped,
		"open={step === 'connect' || step === 'match' || step === 'connect-saved'}",
		'</Dialog>'
	);

	it('renders "1. Connect"', () => {
		expect(connectDialogBlock.includes('1. Connect')).toBe(true);
	});

	it('renders "2. Match"', () => {
		expect(connectDialogBlock.includes('2. Match')).toBe(true);
	});
});

describe('Step 1 failure branch: exact copy plus a Save anyway action', () => {
	it('renders the exact "Couldn\'t verify this connection." copy', () => {
		expect(
			stripped.includes("Couldn't verify this connection."),
			'expected the Step 1 failure branch to render the exact frozen copy'
		).toBe(true);
	});

	it('renders a "Save anyway" action', () => {
		expect(stripped.includes('Save anyway')).toBe(true);
	});

	it('the Save anyway action is gated on the describe-failed flag', () => {
		expect(
			/\{#if describeFailed\}/.test(stripped),
			'expected the Save anyway button to be gated on a describe-failed flag, so it never appears before Step 1 has actually failed'
		).toBe(true);
	});
});

describe('Step 2 submit path: exactly one putConfig call', () => {
	it('submitMatch calls putConfig exactly once', () => {
		const fnBody = extractBetween(
			stripped,
			'async function submitMatch(event: SubmitEvent) {',
			'\n\t}'
		);
		const calls = fnBody.match(/putConfig\(/g) ?? [];
		expect(
			calls.length,
			'expected submitMatch to call putConfig exactly once — two sequential saves could leave a configured-but-unparticipating instance if the second failed'
		).toBe(1);
	});
});

describe('saveAnyway: CR-01 regression — resolveNewInstanceId guards every write', () => {
	const fnBody = extractBetween(stripped, 'async function saveAnyway() {', '\n\t}');

	it('calls resolveNewInstanceId( before upsertSourceInstance(, with a return between them (CR-01)', () => {
		const guardIndex = fnBody.indexOf('resolveNewInstanceId(');
		const writeIndex = fnBody.indexOf('upsertSourceInstance(');
		expect(
			guardIndex,
			'expected saveAnyway to call resolveNewInstanceId( — 07-REVIEW.md CR-01 found this guard missing here'
		).toBeGreaterThanOrEqual(0);
		expect(
			writeIndex,
			'expected saveAnyway to call upsertSourceInstance( at all'
		).toBeGreaterThanOrEqual(0);
		expect(
			guardIndex,
			'expected the resolveNewInstanceId( guard to run strictly before upsertSourceInstance( — otherwise CR-01\'s unguarded overwrite is still reachable'
		).toBeLessThan(writeIndex);
		const between = fnBody.slice(guardIndex, writeIndex);
		expect(
			/\breturn\b/.test(between),
			'expected a return between resolveNewInstanceId( and upsertSourceInstance( so a not-ok result cannot fall through to the write (CR-01)'
		).toBe(true);
	});
});
