// 09-05-PLAN.md Task 1's guard over SourceChip.svelte's tooltipText
// derivation (09-UI-SPEC.md Fix 3): formatRelativeTime (Intl.RelativeTimeFormat,
// numeric: 'auto') already returns a complete phrase — "5 minutes ago" for a
// numeric delta, but also "yesterday", "last week" and "now" for its
// special-cased deltas. The success/warning branches must use that phrase
// VERBATIM, with no appended word — appending " ago" was wrong in every
// case, not just the commonly-hit numeric-delta one.
//
// No component-mount harness exists in this repo's vitest config
// (environment: 'node'), so this test combines two layers, per the house
// pattern (chip-edit-menu.test.ts / plugin-icon.test.ts) extended with a
// behavioral layer the plan's own <action> calls for:
//  1. Structural — the exact template-literal text for each branch is
//     extracted from the real SourceChip.svelte source and asserted
//     byte-for-byte, so a future edit that reintroduces a literal suffix (in
//     either branch) fails here even if no test ever "runs" the component.
//  2. Behavioral — the SAME literal template (proven structurally present
//     above) is evaluated against formatRelativeTime's REAL output for a
//     numeric delta and every Intl.RelativeTimeFormat numeric-auto
//     special-cased delta ("yesterday", "last week", "now"), asserting the
//     resulting string is the exact sentence a user sees, and that no word
//     repeats back-to-back anywhere in it — the general form of the
//     "ago ago" / "yesterday ago" defect class.

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { formatRelativeTime } from '$lib/format';

const here = dirname(fileURLToPath(import.meta.url));
const chipPath = join(here, 'SourceChip.svelte');
const rawChip = readFileSync(chipPath, 'utf-8');

function stripComments(source: string): string {
	return source
		.replace(/<!--[\s\S]*?-->/g, ' ')
		.replace(/\/\*[\s\S]*?\*\//g, ' ')
		.replace(/\/\/.*$/gm, ' ');
}

const strippedChip = stripComments(rawChip);

describe('source-chip-tooltip guard: found non-empty comment-stripped source', () => {
	it('SourceChip.svelte', () => {
		expect(strippedChip.length).toBeGreaterThan(0);
	});
});

describe('tooltipText structure: exact template per branch (Fix 3 Copywriting Contract)', () => {
	it('success branch uses `${relative}` verbatim — no appended word', () => {
		expect(
			strippedChip.includes('return `${source.display_name} — synced ${relative}`;'),
			'expected the success branch to read exactly "{display_name} — synced {relative}" with nothing appended after ${relative}'
		).toBe(true);
	});

	it('warning branch uses `${relative}` verbatim — no appended word before the colon', () => {
		expect(
			strippedChip.includes(
				'return `${source.display_name} — last error ${relative}: ${source.last_error}`;'
			),
			'expected the warning branch to read exactly "{display_name} — last error {relative}: {last_error}" with nothing appended after ${relative}'
		).toBe(true);
	});

	it('destructive branch is untouched — it never had the defect', () => {
		expect(
			strippedChip.includes('return `${source.display_name} — unreachable since ${relative}`;'),
			'expected the destructive branch to remain byte-identical'
		).toBe(true);
	});

	it('unknown/default branch is untouched — it never had the defect', () => {
		expect(
			strippedChip.includes('return `${source.display_name} — not yet synced`;'),
			'expected the unknown/default branch to remain byte-identical'
		).toBe(true);
	});

	it('no branch appends a literal " ago" after ${relative}', () => {
		expect(
			/\$\{relative\}\s*ago/.test(strippedChip),
			'found "${relative} ago" (or "${relative}" followed by "ago") in the source — formatRelativeTime already returns a complete phrase, so this is always a doubled word'
		).toBe(false);
	});
});

// 13-06-PLAN.md (D-12/D-13/D-14): the two new named chip states' contract
// sentences (13-UI-SPEC.md Copywriting Contract), asserted byte-for-byte
// against the real component source — the behavioural precedence coverage
// (which branch is chosen for which combination of inputs) lives in
// match-advisory.test.ts; this file's own job is the exact wording.
// 14-02-PLAN.md Task 2 (option-b, see Task 1's decision record): the
// native-tooltip `title` attributes that used to duplicate/cover the app's
// own styled Tooltip popover are gone from both popover-bearing elements —
// the outer chip filter button and the inner truncated display-name span.
// The dropdown-footer pinned-hash span is untouched (it sits inside
// already-open dropdown content, not behind a hover popover; 14-UI-SPEC.md
// G1 point 3 rules it out of scope). The replacement is a visually-hidden
// `sr-only` sibling span carrying the exact `tooltipText` expression, wired
// to the button via `aria-describedby` — preserving the button's accessible
// NAME (the display name, via its own text content) while keeping the
// health sentence reachable as an accessible DESCRIPTION, the role the
// removed `title` played.
describe('native-tooltip suppression (14-02-PLAN.md, option-b): no title on popover-bearing elements', () => {
	it('the outer chip filter button carries no title attribute', () => {
		expect(
			strippedChip.includes('title={tooltipText}'),
			'expected title={tooltipText} to be removed from the outer chip button'
		).toBe(false);
	});

	it('the inner truncated display-name span carries no title attribute', () => {
		expect(
			strippedChip.includes('title={source.display_name}'),
			'expected title={source.display_name} to be removed from the inner display-name span'
		).toBe(false);
	});

	it('the dropdown-footer pinned-hash span still carries its own title attribute (untouched, out of scope)', () => {
		const matches = strippedChip.match(/title=\{source\.pinned_hash\}/g) ?? [];
		expect(
			matches.length,
			'expected exactly one title={source.pinned_hash} to remain, on the dropdown-footer span'
		).toBe(1);
	});

	it('the outer chip button is wired to the replacement description via aria-describedby={chipDescId}', () => {
		expect(
			strippedChip.includes('aria-describedby={chipDescId}'),
			'expected the outer chip button to carry aria-describedby={chipDescId}'
		).toBe(true);
	});

	it('a visually-hidden sr-only span renders the same tooltipText expression as the replacement description', () => {
		expect(
			/id=\{chipDescId\}\s+class="sr-only">\{tooltipText\}/.test(strippedChip),
			'expected a `sr-only` span with id={chipDescId} rendering {tooltipText} verbatim'
		).toBe(true);
	});

	it('no aria-label carries the tooltipText expression on the outer button (option-a was not taken)', () => {
		expect(
			strippedChip.includes('aria-label={tooltipText}'),
			'expected NO aria-label={tooltipText} — option-b keeps the sentence as a description, not the accessible name'
		).toBe(false);
	});
});

describe('tooltipText structure: the two new 13-06-PLAN.md branches (D-12/D-13/D-14)', () => {
	it('manifest-unverified branch reads the contract-exact sentence', () => {
		expect(
			strippedChip.includes(
				'return `${source.display_name} — binary not in the trusted build manifest`;'
			),
			'expected the manifest-unverified branch to read exactly "{display_name} — binary not in the trusted build manifest"'
		).toBe(true);
	});

	it('shadowed branch reads the contract-exact sentence', () => {
		expect(
			strippedChip.includes(
				'return `${source.display_name} — a same-named trusted-directory binary is shadowing this pinned plugin`;'
			),
			'expected the shadowed branch to read exactly "{display_name} — a same-named trusted-directory binary is shadowing this pinned plugin"'
		).toBe(true);
	});
});

// Mirrors the exact branch templates proven present above — this function
// is intentionally the same shape as SourceChip.svelte's own tooltipText
// switch, so evaluating it against formatRelativeTime's REAL output proves
// what a user actually sees, not merely that a substring is absent.
function successTooltip(displayName: string, relative: string): string {
	return `${displayName} — synced ${relative}`;
}

function warningTooltip(displayName: string, relative: string, lastError: string): string {
	return `${displayName} — last error ${relative}: ${lastError}`;
}

/** True if any whitespace-delimited word repeats immediately after itself (case-insensitive). */
function hasAdjacentDuplicateWord(text: string): boolean {
	const words = text.split(/\s+/).filter((w) => w.length > 0);
	for (let i = 1; i < words.length; i += 1) {
		if (words[i].toLowerCase() === words[i - 1].toLowerCase()) return true;
	}
	return false;
}

const DISPLAY_NAME = 'paperless-ngx';
const nowUnix = () => Math.floor(Date.now() / 1000);

describe('tooltipText behavior: every Intl.RelativeTimeFormat numeric-auto shape, against real formatRelativeTime output', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2024-06-15T12:00:00Z'));
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	it('numeric delta (5 minutes ago) reads as a complete sentence with the number preserved', () => {
		const relative = formatRelativeTime(nowUnix() - 5 * 60);
		expect(relative).toMatch(/^5 minutes ago$/);
		const text = successTooltip(DISPLAY_NAME, relative);
		expect(text).toBe('paperless-ngx — synced 5 minutes ago');
		expect(hasAdjacentDuplicateWord(text)).toBe(false);
	});

	it('"yesterday" (special-cased) reads as a complete sentence with no appended "ago"', () => {
		const relative = formatRelativeTime(nowUnix() - 24 * 60 * 60);
		expect(relative).toBe('yesterday');
		const text = successTooltip(DISPLAY_NAME, relative);
		expect(text).toBe('paperless-ngx — synced yesterday');
		expect(text).not.toContain('yesterday ago');
		expect(hasAdjacentDuplicateWord(text)).toBe(false);
	});

	it('"last week" (special-cased) reads as a complete sentence with no appended "ago"', () => {
		const relative = formatRelativeTime(nowUnix() - 7 * 24 * 60 * 60);
		expect(relative).toBe('last week');
		const text = successTooltip(DISPLAY_NAME, relative);
		expect(text).toBe('paperless-ngx — synced last week');
		expect(text).not.toContain('week ago');
		expect(hasAdjacentDuplicateWord(text)).toBe(false);
	});

	it('"now" (special-cased, zero delta) reads as a complete sentence with no appended "ago"', () => {
		const relative = formatRelativeTime(nowUnix());
		expect(relative).toBe('now');
		const text = successTooltip(DISPLAY_NAME, relative);
		expect(text).toBe('paperless-ngx — synced now');
		expect(text).not.toContain('now ago');
		expect(hasAdjacentDuplicateWord(text)).toBe(false);
	});

	it('the warning branch carries the same fix — a numeric delta reads correctly before the colon', () => {
		const relative = formatRelativeTime(nowUnix() - 3 * 60 * 60);
		expect(relative).toMatch(/^3 hours ago$/);
		const text = warningTooltip(DISPLAY_NAME, relative, 'connection refused');
		expect(text).toBe('paperless-ngx — last error 3 hours ago: connection refused');
		expect(hasAdjacentDuplicateWord(text)).toBe(false);
	});

	it('the warning branch carries the fix for a special-cased delta too ("yesterday")', () => {
		const relative = formatRelativeTime(nowUnix() - 24 * 60 * 60);
		expect(relative).toBe('yesterday');
		const text = warningTooltip(DISPLAY_NAME, relative, 'timeout');
		expect(text).toBe('paperless-ngx — last error yesterday: timeout');
		expect(text).not.toContain('yesterday ago');
		expect(hasAdjacentDuplicateWord(text)).toBe(false);
	});
});
