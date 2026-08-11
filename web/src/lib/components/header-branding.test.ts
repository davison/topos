// Quick task 260811-rqc: the header gains a static branding lockup (app
// icon + "topos" wordmark + tagline) in its top band, right-aligned beside
// the webspace-switcher title. Purely structural guard against
// COMMENT-STRIPPED WebspaceHeader.svelte source, mirroring
// source-chip-pill.test.ts's house pattern (vitest, environment: 'node',
// no component-mount harness — extractBetween-scoped regions, a
// found-non-empty-source guard first, one consequence-describing message
// per assertion).
//
// The load-bearing assertion is placement (see "placement" describe
// block below): the branding block must be a SIBLING of the measured chip
// row (`bind:this={rowEl}`), never a child of it — nesting it inside would
// silently shrink visibleChipCount's available-width input with no
// visible error (T-rqc-02).

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const headerPath = join(here, 'WebspaceHeader.svelte');
const rawHeader = readFileSync(headerPath, 'utf-8');

// Strips HTML comments, CSS/JS block comments and JS line comments, each
// replaced with a single space (never deleted outright) so two tokens
// separated only by a comment can never fuse into one identifier.
function stripComments(source: string): string {
	return source
		.replace(/<!--[\s\S]*?-->/g, ' ')
		.replace(/\/\*[\s\S]*?\*\//g, ' ')
		.replace(/\/\/.*$/gm, ' ');
}

const strippedHeader = stripComments(rawHeader);

// Slices the text from the first occurrence of `startMarker` through the
// following occurrence of `endMarker` (inclusive) — used to scope an
// assertion to one region rather than the whole file, so a match elsewhere
// in the document can never satisfy or trip it by accident. Asserts both
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

describe('header-branding guard: found non-empty comment-stripped source', () => {
	it('WebspaceHeader.svelte', () => {
		expect(strippedHeader.length).toBeGreaterThan(0);
	});
});

describe('the branding icon is the shipped app-icon asset, decorative', () => {
	it('an <img> sources /app-icon.png with an empty alt', () => {
		const imgTag = extractBetween(strippedHeader, '<img src="/app-icon.png"', '/>');
		expect(
			/alt=""/.test(imgTag),
			'expected the branding icon to carry an empty alt — decorative, the adjacent wordmark already names the product'
		).toBe(true);
	});
});

describe('the wordmark and tagline text render literally', () => {
	it('contains the literal wordmark "topos"', () => {
		expect(
			strippedHeader.includes('>topos<'),
			'expected the literal wordmark text "topos" to appear in a text node in the markup'
		).toBe(true);
	});
	it('contains the literal tagline "bringing all your topics to one place"', () => {
		expect(
			strippedHeader.includes('bringing all your topics to one place'),
			'expected the literal tagline text to appear in the markup'
		).toBe(true);
	});
});

describe('the branding lockup is muted, never the application default text colour', () => {
	it('the lockup region carries text-muted-foreground and never text-foreground', () => {
		const lockupRegion = extractBetween(
			strippedHeader,
			'<img src="/app-icon.png"',
			'bringing all your topics to one place'
		);
		expect(
			/\btext-muted-foreground\b/.test(lockupRegion),
			'expected the branding lockup to carry text-muted-foreground somewhere in its region — the wordmark/tagline must read as muted, not full-strength application text'
		).toBe(true);
		expect(
			/\btext-foreground\b/.test(lockupRegion),
			"found text-foreground inside the branding lockup region — the branding must never use the application's default text-colour utility, only the muted token"
		).toBe(false);
	});
});

describe('placement: the branding lockup is a SIBLING of the measured chip row, never nested inside it', () => {
	it('the /app-icon.png <img> appears BEFORE bind:this={rowEl} in source order', () => {
		const iconIndex = strippedHeader.indexOf('/app-icon.png');
		const rowElIndex = strippedHeader.indexOf('bind:this={rowEl}');
		expect(
			iconIndex,
			'expected to find /app-icon.png in the scanned source'
		).toBeGreaterThanOrEqual(0);
		expect(
			rowElIndex,
			'expected to find bind:this={rowEl} in the scanned source'
		).toBeGreaterThanOrEqual(0);
		expect(
			iconIndex,
			"the branding lockup must be emitted BEFORE — and therefore outside — the measured chip row; a lockup nested inside rowEl would silently shrink visibleChipCount's available width with no visible error"
		).toBeLessThan(rowElIndex);
	});
});

describe('type scale: wordmark smaller than the switcher title, tagline smaller than the wordmark', () => {
	it("the wordmark font-size class is strictly smaller than the switcher title's text-[28px]", () => {
		const wordmarkSpan = extractBetween(strippedHeader, 'text-muted-foreground', '>topos<');
		const match = wordmarkSpan.match(/text-\[(\d+)px\]/);
		expect(
			match,
			'expected the wordmark span to declare a text-[NNpx] arbitrary font-size class'
		).not.toBeNull();
		const wordmarkSize = Number(match![1]);
		expect(
			wordmarkSize,
			'expected the wordmark to be smaller than the switcher title (28px) so it never out-shouts the webspace name it sits beside'
		).toBeLessThan(28);
	});

	it("the tagline font-size class is strictly smaller than the wordmark's", () => {
		const wordmarkSpan = extractBetween(strippedHeader, 'text-muted-foreground', '>topos<');
		const wordmarkMatch = wordmarkSpan.match(/text-\[(\d+)px\]/);
		expect(wordmarkMatch).not.toBeNull();
		const wordmarkSize = Number(wordmarkMatch![1]);

		const taglineSpan = extractBetween(
			strippedHeader,
			'>topos<',
			'bringing all your topics to one place'
		);
		const taglineMatch = taglineSpan.match(/text-\[(\d+)px\]/);
		expect(
			taglineMatch,
			'expected the tagline span to declare a text-[NNpx] arbitrary font-size class'
		).not.toBeNull();
		const taglineSize = Number(taglineMatch![1]);

		expect(
			taglineSize,
			'expected the tagline to render smaller than the wordmark above it'
		).toBeLessThan(wordmarkSize);
	});
});
