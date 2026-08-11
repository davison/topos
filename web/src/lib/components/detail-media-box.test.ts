// 09-UI-SPEC.md Fix 9: DetailPane.svelte's media branch previewer was
// pinned to a fixed h-72 (288px) height while its width ran free (w-full,
// unbounded by the flex-1 detail pane), so a portrait document rendered
// very wide and very short — and the extracted text, being a separate
// stacked flex child below it, could never occupy the horizontal space
// beside the preview.
//
// The fix: a bounded, aspect-locked box (w-full max-w-sm aspect-[3/4]),
// applied identically to both content types the box renders (the PDF
// iframe and the img fallback), that floats left only when there is
// extracted text to flow beside it — and stays unfloated/centred when
// there is not, since there is nothing to wrap around.
//
// House pattern (matches search-box-clear.test.ts, source-chip-pill.test.ts):
// comment-stripped source scanning, `extractBetween` scoping, a
// found-non-empty-source guard first, one consequence-describing message
// per assertion.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));

const detailPanePath = join(here, 'DetailPane.svelte');
const rawDetailPane = readFileSync(detailPanePath, 'utf-8');

// Strips HTML comments, CSS/JS block comments and JS line comments, each
// replaced with a single space (never deleted outright) so two tokens
// separated only by a comment can never fuse into one identifier.
function stripComments(source: string): string {
	return source
		.replace(/<!--[\s\S]*?-->/g, ' ')
		.replace(/\/\*[\s\S]*?\*\//g, ' ')
		.replace(/\/\/.*$/gm, ' ');
}

const stripped = stripComments(rawDetailPane);

// Slices the text from the first occurrence of `startMarker` through the
// following occurrence of `endMarker` (inclusive) — used to scope an
// assertion to one branch's block rather than the whole file. Asserts both
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

describe('detail-media-box guard: found non-empty comment-stripped source', () => {
	it('DetailPane.svelte', () => {
		expect(stripped.length).toBeGreaterThan(0);
	});
});

// Scope to the whole media branch, from its {:else if bodyVariant ===
// 'media'} through the following {:else if bodyVariant === 'text'} — this
// bounds every assertion below to only the media branch, never the
// unrelated unreachable/deleted branch's own separate thumbnail box or the
// loading skeleton, both of which legitimately keep h-72/w-full sizing of
// their own.
const mediaBranch = extractBetween(
	stripped,
	"{:else if bodyVariant === 'media'}",
	"{:else if bodyVariant === 'text'}"
);

describe('bounded, aspect-locked sizing', () => {
	it('the media box is sized w-full max-w-sm aspect-[3/4]', () => {
		expect(
			/\bw-full\b/.test(mediaBranch),
			'expected the media box to carry w-full — width-bounded by max-w-sm, scaling down on a narrower pane'
		).toBe(true);
		expect(
			/\bmax-w-sm\b/.test(mediaBranch),
			'expected the media box to carry max-w-sm — the 384px width bound the aspect ratio derives height from'
		).toBe(true);
		expect(
			/aspect-\[3\/4\]/.test(mediaBranch),
			'expected the media box to carry aspect-[3/4] — height must be derived from the ratio, never set independently'
		).toBe(true);
	});

	it('no fixed height (h-72) remains on the media box itself', () => {
		expect(
			/\bh-72\b/.test(mediaBranch),
			'found h-72 still on the media branch — the fixed-height box must be fully replaced by the aspect-locked sizing, not layered alongside it'
		).toBe(false);
	});

	it('no separate max-height utility is introduced', () => {
		expect(
			/max-h-/.test(mediaBranch),
			'found a max-h- utility on the media branch — the aspect ratio and max-width already cap the box at 512px tall; no extra viewport-relative cap should be added'
		).toBe(false);
	});
});

describe('same sizing applies to both content types', () => {
	it('the PDF iframe branch and the img fallback branch are both inside one aspect-locked container', () => {
		expect(
			mediaBranch.includes('application/pdf'),
			'expected the media branch to still distinguish the PDF iframe from the img fallback'
		).toBe(true);
		// Both <iframe> and <img> must appear strictly after the single
		// aspect-[3/4] declaration that opens their shared container — proof
		// that one box treatment wraps both, not two separate boxes.
		const aspectIndex = mediaBranch.search(/aspect-\[3\/4\]/);
		const iframeIndex = mediaBranch.indexOf('<iframe');
		const imgIndex = mediaBranch.indexOf('<img');
		expect(aspectIndex).toBeGreaterThanOrEqual(0);
		expect(
			iframeIndex,
			'expected the <iframe> element to sit inside the aspect-locked container'
		).toBeGreaterThan(aspectIndex);
		expect(
			imgIndex,
			'expected the <img> fallback to sit inside the same aspect-locked container as the iframe'
		).toBeGreaterThan(aspectIndex);
	});
});

describe('conditional float: only when content.text is present', () => {
	it('float-left appears exactly once, and only inside a content?.text conditional', () => {
		const floatMatches = mediaBranch.match(/\bfloat-left\b/g) ?? [];
		expect(
			floatMatches.length,
			'expected exactly one float-left occurrence in the media branch — floated only under one condition, not duplicated across branches'
		).toBe(1);

		const floatIndex = mediaBranch.search(/\bfloat-left\b/);
		const precedingText = mediaBranch.slice(0, floatIndex);
		expect(
			/content\?\.text/.test(precedingText),
			'expected float-left to be preceded by a content?.text check in the same expression — the box must only float when there is text to flow beside it'
		).toBe(true);
	});

	it('the float carries trailing margin so text does not hug the box edge', () => {
		expect(
			/\bmr-6\b/.test(mediaBranch),
			'expected the floated box to carry mr-6 — right margin so flowing text keeps clear space from the float edge'
		).toBe(true);
		expect(
			/\bmb-4\b/.test(mediaBranch),
			'expected the floated box to carry mb-4 — bottom margin so text below the float keeps clear space'
		).toBe(true);
	});

	it('the box is not floated when content.text is absent — it centres instead', () => {
		expect(
			/\bmx-auto\b/.test(mediaBranch),
			'expected a centring utility (mx-auto) for the no-text case — the box renders centred/block, not floated, since there is nothing to wrap around'
		).toBe(true);
	});
});

describe('extracted text renders as a flowing block, not the shared flex snippet', () => {
	it('the media branch does not invoke the loadedTextBlock snippet', () => {
		expect(
			mediaBranch.includes('{@render loadedTextBlock()}'),
			'expected the media branch to render its text inline as a plain flowing block, not via {@render loadedTextBlock()} — that snippet is flex-sized for the text-only branch and cannot sit beside a float correctly'
		).toBe(false);
	});

	it('the media branch text block keeps the same typography classes as loadedTextBlock', () => {
		expect(mediaBranch.includes('text-[16px]')).toBe(true);
		expect(mediaBranch.includes('leading-[1.5]')).toBe(true);
		expect(mediaBranch.includes('whitespace-pre-wrap')).toBe(true);
		expect(mediaBranch.includes('text-foreground')).toBe(true);
	});

	it('the media branch still renders content via highlightText, matching search-highlight spans', () => {
		expect(
			mediaBranch.includes('highlightText(content?.text'),
			'expected the media branch to still run extracted text through highlightText for UI-09 search-term highlighting'
		).toBe(true);
		expect(mediaBranch.includes('search-highlight')).toBe(true);
	});
});

describe('the unreachable/deleted branch keeps its own separate thumbnail box unchanged', () => {
	it('the unreachable/deleted branch still carries its own h-72 thumbnail sizing', () => {
		const unreachableBranch = extractBetween(
			stripped,
			"paneState === 'unreachable' || paneState === 'deleted'",
			"{:else if paneState === 'error'}"
		);
		expect(
			/\bh-72\b/.test(unreachableBranch),
			"expected the unreachable/deleted branch's own thumbnail box to keep its existing h-72 sizing — this task only changes the media branch"
		).toBe(true);
	});
});
