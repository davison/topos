// 09-UI-SPEC.md Fix 2: SearchBox.svelte's clear button jumped on mousedown.
// Root cause: the clear Button was centred with `top-1/2 -translate-y-1/2`,
// which writes the same CSS custom property (--tw-translate-y) that
// ui/button/button.svelte's shared press affordance
// (`active:not-aria-[haspopup]:translate-y-px`) also writes. On mousedown the
// press value replaced the centring value outright, snapping the button by
// roughly half its own height.
//
// The fix decouples the two: the clear Button is now wrapped in an
// `absolute inset-y-0 right-0 flex items-center` track that centres its
// child with flex alignment on a fixed-height box, writing no
// vertical-translate CSS property of its own. The Button's own class no
// longer carries any positioning transform, so the shared press affordance
// has nothing to collide with.
//
// House pattern (matches source-chip-pill.test.ts): comment-stripped
// source scanning (web/vite.config.ts's test block runs environment: 'node'
// with no component-mount harness, so there is no DOM to render against),
// `extractBetween` scoping, a found-non-empty-source guard first, and one
// consequence-describing message per assertion. Helpers are redefined here
// rather than imported, same as every sibling source-scan test file.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));

const searchBoxPath = join(here, 'SearchBox.svelte');
const buttonPath = join(here, 'ui/button/button.svelte');

const rawSearchBox = readFileSync(searchBoxPath, 'utf-8');
const rawButton = readFileSync(buttonPath, 'utf-8');

// Strips HTML comments, CSS/JS block comments and JS line comments, each
// replaced with a single space (never deleted outright) so two tokens
// separated only by a comment can never fuse into one identifier.
function stripComments(source: string): string {
	return source
		.replace(/<!--[\s\S]*?-->/g, ' ')
		.replace(/\/\*[\s\S]*?\*\//g, ' ')
		.replace(/\/\/.*$/gm, ' ');
}

const strippedSearchBox = stripComments(rawSearchBox);
const strippedButton = stripComments(rawButton);

// Slices the text from the first occurrence of `startMarker` through the
// following occurrence of `endMarker` (inclusive) — used to scope an
// assertion to one element's block rather than the whole file. Asserts both
// markers were found so a bad marker fails loudly instead of vacuously
// passing.
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

describe('search-box-clear guard: found non-empty comment-stripped sources', () => {
	it('SearchBox.svelte', () => {
		expect(strippedSearchBox.length).toBeGreaterThan(0);
	});
	it('ui/button/button.svelte', () => {
		expect(strippedButton.length).toBeGreaterThan(0);
	});
});

// The clear control's wrapper div and the Button element nested inside it,
// plus the Search icon's own (unrelated, untouched) centring markup.
const clearWrapperBlock = extractBetween(strippedSearchBox, '<div class="absolute inset-y-0', '</div>');
const clearButtonBlock = extractBetween(clearWrapperBlock, '<Button', '</Button>');
const searchIconTag = extractBetween(strippedSearchBox, '<Search', '/>');

describe('clear control centring: no vertical-translate CSS property written', () => {
	it('the wrapper stretches the full input height and flex-centres its child', () => {
		expect(
			/\binset-y-0\b/.test(clearWrapperBlock),
			'expected the clear control wrapper to carry inset-y-0 — a fixed-height track the button centres on, with no vertical-translate of its own'
		).toBe(true);
		expect(
			/\bitems-center\b/.test(clearWrapperBlock),
			'expected the clear control wrapper to carry items-center — flex centring is the non-transformed technique that replaces top-1/2 -translate-y-1/2'
		).toBe(true);
		expect(
			/\bflex\b/.test(clearWrapperBlock),
			'expected the clear control wrapper to declare flex — items-center has no effect without a flex (or grid) container'
		).toBe(true);
	});

	it('the Button itself carries no translate-y utility', () => {
		expect(
			/-translate-y-1\/2/.test(clearButtonBlock),
			'found -translate-y-1/2 still on the clear Button — this is the exact CSS custom property the shared press affordance collides with; the fix requires it be gone from the Button entirely'
		).toBe(false);
		expect(
			/\btop-1\/2\b/.test(clearButtonBlock),
			'found top-1/2 still on the clear Button — the transform-based centring idiom must be fully replaced by the wrapper’s flex centring, not layered alongside it'
		).toBe(false);
	});

	it('the Button no longer declares its own absolute position', () => {
		expect(
			/\babsolute\b/.test(clearButtonBlock),
			'found absolute still on the clear Button itself — positioning now belongs to the wrapper div, not the Button, so the Button should render as a normal flex child'
		).toBe(false);
	});
});

describe('everything else about the clear control is unchanged', () => {
	it('still only renders when the input has a value, via the existing #if guard', () => {
		expect(
			strippedSearchBox.includes('{#if inputValue}'),
			'expected the clear control to still be gated behind {#if inputValue} — it must not render for an empty query'
		).toBe(true);
	});

	it('still calls the existing immediate (non-debounced) clear handler', () => {
		expect(
			/onclick=\{handleClear\}/.test(clearButtonBlock),
			'expected the clear Button to still call handleClear directly on click — clearing must stay immediate, not debounced'
		).toBe(true);
	});

	it('still carries the accessible label from searchCopy.clearLabel', () => {
		expect(
			/aria-label=\{searchCopy\.clearLabel\}/.test(clearButtonBlock),
			'expected the clear Button to still source its aria-label from searchCopy.clearLabel'
		).toBe(true);
	});

	it('still uses the ghost variant and icon size', () => {
		expect(/variant="ghost"/.test(clearButtonBlock)).toBe(true);
		expect(/size="icon"/.test(clearButtonBlock)).toBe(true);
	});

	it("the leading Search icon's own centring is unchanged — it is not a Button", () => {
		expect(
			/-translate-y-1\/2/.test(searchIconTag),
			'expected the leading Search icon to keep its own transform-based centring — it is a plain icon, not a Button, and was never affected by the press-affordance collision'
		).toBe(true);
		expect(/\btop-1\/2\b/.test(searchIconTag)).toBe(true);
	});
});

describe('app-wide press affordance guard: button.svelte is untouched', () => {
	it('the shared Button base class still declares its press affordance', () => {
		expect(
			strippedButton.includes("active:not-aria-[haspopup]:translate-y-px"),
			'expected ui/button/button.svelte to still declare its app-wide press affordance — this fix must decouple the clear control from the shared affordance, not remove or special-case the affordance itself, so every other button in the app keeps behaving identically'
		).toBe(true);
	});
});
