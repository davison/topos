// 07-04-PLAN.md Task 2's structural guard over SecretField.svelte (D-15,
// T-07-19): the field is a plain text input (never a password input, never
// autofill-eligible), both badge branches render their exact frozen copy
// gated on a non-blank name, and no prop or local in the component is
// named for a secret VALUE rather than a variable NAME.
//
// House pattern (matches source-chip-pill.test.ts / webspace-switcher.test.ts):
// comment-stripped source scanning, a found-non-empty-source guard first,
// and one consequence-describing message per assertion.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const fieldPath = join(here, 'SecretField.svelte');

function stripComments(source: string): string {
	return source
		.replace(/<!--[\s\S]*?-->/g, ' ')
		.replace(/\/\*[\s\S]*?\*\//g, ' ')
		.replace(/\/\/.*$/gm, ' ');
}

const raw = readFileSync(fieldPath, 'utf-8');
const stripped = stripComments(raw);

describe('secret-field guard: found non-empty comment-stripped source', () => {
	it('SecretField.svelte', () => {
		expect(stripped.length).toBeGreaterThan(0);
	});
});

describe('the field is a plain text input, never a password input', () => {
	it('declares type="text" on its Input usage', () => {
		expect(
			stripped.includes('type="text"'),
			'expected SecretField to render its Input with an explicit type="text"'
		).toBe(true);
	});

	it('never declares type="password" anywhere in the file', () => {
		expect(
			stripped.includes('type="password"'),
			'found type="password" in SecretField.svelte — this field must never render a password input (D-15/T-07-19)'
		).toBe(false);
	});
});

describe('no autofill-enabling attribute', () => {
	it('declares autocomplete="off"', () => {
		expect(
			stripped.includes('autocomplete="off"'),
			'expected SecretField to declare autocomplete="off" — a browser autofilling a stored credential into this field would be exactly the leak D-15 forbids'
		).toBe(true);
	});
});

describe('both badge branches render their exact copy, gated on a non-blank name', () => {
	it('renders the exact "Set" copy', () => {
		expect(stripped.includes('>Set<'), 'expected the set-badge branch to render the exact text "Set"').toBe(
			true
		);
	});

	it('renders the exact "Not set…" copy', () => {
		expect(
			stripped.includes('Not set — add it to .env and restart before this source can connect.'),
			'expected the unset-badge branch to render the exact frozen copy'
		).toBe(true);
	});

	it('gates the badge region on a non-blank trimmed name', () => {
		expect(
			/\{#if trimmed !== ''\}/.test(stripped),
			"expected the badge region to be gated on the trimmed value being non-blank — the badge must not render while the input is blank"
		).toBe(true);
	});
});

describe('no prop or local named for a secret VALUE rather than a variable NAME', () => {
	it('contains no identifier suggesting a secret value is held anywhere in this file', () => {
		expect(
			/\b(secretValue|tokenValue|password|secret_value)\b/i.test(stripped),
			'found an identifier suggesting this component holds a secret VALUE — its only content must ever be the variable NAME'
		).toBe(false);
	});
});
