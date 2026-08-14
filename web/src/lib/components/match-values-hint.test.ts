// 12-10-PLAN.md Task 3's guard over MatchFieldsForm.svelte's helper text
// (G-12-1/G-12-3 gap closure): the debug session's own blind_spots note —
// "Did not verify the UI match editor's placeholder/help text — if it
// hints at globs it compounds the trap" — flagged this exact surface as
// unexamined. This form is the ONE place every plugin's match values are
// typed into (both add-source flows and the chip menu's "Edit match
// settings…" modal, 07-04-PLAN.md D-11), so its helper text is where the
// mistake that produced this gap (a glob-shaped value the pipeline treats
// as a literal) is discouraged at the moment it is made.
//
// House pattern (matches extras-form.test.ts / connection-checkbox.test.ts
// — no component-mount harness exists in this repo's vitest config,
// environment: 'node'): comment-stripped source scanning, a found-non-
// empty-source guard first, and one consequence-describing message per
// assertion.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const formPath = join(here, 'MatchFieldsForm.svelte');
const rawForm = readFileSync(formPath, 'utf-8');

function stripComments(source: string): string {
	return source
		.replace(/<!--[\s\S]*?-->/g, ' ')
		.replace(/\/\*[\s\S]*?\*\//g, ' ')
		.replace(/\/\/.*$/gm, ' ');
}

const strippedForm = stripComments(rawForm);

describe('match-values-hint guard: found non-empty comment-stripped source', () => {
	it('MatchFieldsForm.svelte', () => {
		expect(strippedForm.length).toBeGreaterThan(0);
	});
});

describe('MatchFieldsForm.svelte: the shared helper text states the exact-match rule', () => {
	it('preserves the pre-existing sentence verbatim', () => {
		expect(
			strippedForm.includes('Comma-separated. Matches if any value is present.'),
			'expected the pre-existing helper sentence to remain byte-identical'
		).toBe(true);
	});

	it('states that values are matched exactly', () => {
		expect(/exactly/i.test(strippedForm)).toBe(true);
	});

	it('states that wildcard/glob patterns are not supported', () => {
		expect(/wildcard/i.test(strippedForm)).toBe(true);
	});

	it('renders exactly one helper <p> per field — no second element grown alongside it', () => {
		const matches = strippedForm.match(/<p class="text-\[14px\] leading-\[1\.4\] text-muted-foreground">/g) ?? [];
		expect(
			matches.length,
			'expected exactly one muted helper paragraph per field, not a second element'
		).toBe(1);
	});

	it('the new sentence lives in the SAME paragraph as the pre-existing one, not a sibling node', () => {
		const pBlock = strippedForm.slice(
			strippedForm.indexOf('<p class="text-[14px] leading-[1.4] text-muted-foreground">'),
			strippedForm.indexOf('</p>', strippedForm.indexOf('<p class="text-[14px] leading-[1.4] text-muted-foreground">'))
		);
		expect(pBlock.includes('Comma-separated. Matches if any value is present.')).toBe(true);
		expect(/exactly/i.test(pBlock)).toBe(true);
		expect(/wildcard/i.test(pBlock)).toBe(true);
	});

	it('contains no raw-HTML output directive', () => {
		expect(strippedForm.includes('{@html')).toBe(false);
	});

	it('the label for and input id wiring is unchanged — every existing locator still resolves', () => {
		expect(strippedForm.includes('for={`match-field-${field}`}')).toBe(true);
		expect(strippedForm.includes('id={`match-field-${field}`}')).toBe(true);
	});
});
