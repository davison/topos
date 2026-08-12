// Source-scan guard over the two-pane layout in
// web/src/routes/w/[webspace]/+page.svelte. No component-mount harness
// exists (web/vite.config.ts's test block: environment: 'node'), and
// jsdom would not help even if it did — jsdom performs no layout, so
// flexbox widths are never computed there. This test reads the route's
// source text off disk instead, following the same pattern as
// web/src/lib/components/date-format.test.ts.
//
// The invariant being guarded is now THREE-BANDED (09.1-01-PLAN.md D-02,
// widened by Task 3 per RESEARCH Pitfall 2 — the guard is updated as a
// behaviour contract, not merely hoped to still pass):
//   - below 768px: the stream pane stays invisible-but-full-width (its
//     scroll box survives, D-01) while the detail pane becomes a
//     full-viewport takeover (D-01/D-04);
//   - 768-1024px: the stream pane holds a proportional clamped width and
//     the detail pane sits beside it, sibling-flex, exactly as before;
//   - 1024px and up: byte-identical to the pre-phase layout — the stream
//     pane holds a fixed width and the detail pane absorbs the rest.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const routePath = join(
	dirname(fileURLToPath(import.meta.url)),
	'..',
	'routes',
	'w',
	'[webspace]',
	'+page.svelte'
);
const routeSource = readFileSync(routePath, 'utf-8');

// A fixed pixel-width utility, e.g. `w-[480px]` — an arbitrary-value
// Tailwind width bracket containing digits followed by `px`.
const FIXED_PX_WIDTH = /w-\[\d+px\]/;

// Collects every `class="..."` attribute value in the route source and
// returns the single one containing `marker`. Asserting "exactly one
// match" (not "at least one") is what scopes every subsequent assertion
// to that one pane wrapper's class string, rather than the whole file —
// this is what makes the guard immune to any prose an author later
// writes in a Svelte comment elsewhere in the file (e.g. a comment that
// happens to mention `border-l` in passing would fail this helper loudly
// instead of silently making a later assertion pass against comment
// text).
function findUniqueClassAttr(marker: string): string {
	const classAttrRegex = /class="([^"]*)"/g;
	const matches: string[] = [];
	let match: RegExpExecArray | null;
	while ((match = classAttrRegex.exec(routeSource)) !== null) {
		if (match[1].includes(marker)) {
			matches.push(match[1]);
		}
	}
	expect(
		matches.length,
		`expected exactly one class attribute containing "${marker}" in ${routePath}, found ${matches.length} — the pane-wrapper markers must stay unique to their own wrapper for this guard to remain meaningful`
	).toBe(1);
	return matches[0];
}

describe('pane-layout source-scan guard', () => {
	it('found a non-empty route source to scan', () => {
		// Guards against a silent no-op: a wrong path resolution must fail
		// loudly here rather than making every assertion below vacuously
		// pass over an empty string.
		expect(routeSource.length).toBeGreaterThan(0);
	});

	it('has exactly one stream-wrapper class attribute and one detail-wrapper class attribute', () => {
		// overflow-x-hidden and border-l are each unique to their own pane
		// wrapper in this file today; findUniqueClassAttr's exactly-one
		// assertion is what enforces that this stays true.
		expect(() => findUniqueClassAttr('overflow-x-hidden')).not.toThrow();
		expect(() => findUniqueClassAttr('border-l')).not.toThrow();
	});

	it('detail pane (reading surface) is not pinned to a fixed width and does not refuse to shrink', () => {
		const detailClass = findUniqueClassAttr('border-l');
		expect(
			FIXED_PX_WIDTH.test(detailClass),
			'the detail pane must not carry a fixed pixel width — it is the reading surface and must absorb viewport width changes, not the list of truncated titles'
		).toBe(false);
		expect(
			detailClass.includes('shrink-0'),
			'the detail pane must not carry shrink-0 — a fixed no-shrink utility on the reading surface would fight the flex-1 growth utility it needs'
		).toBe(false);
	});

	it('detail pane (reading surface) grows to absorb viewport width changes', () => {
		const detailClass = findUniqueClassAttr('border-l');
		expect(
			detailClass.includes('flex-1'),
			'the detail pane must carry flex-1 so it absorbs viewport width changes — extra width belongs to the content being read, not to the list of truncated titles'
		).toBe(true);
		expect(
			detailClass.includes('min-w-0'),
			'the detail pane must carry min-w-0 — as a growing flex child, wide intrinsic content (a PDF embed, a long unbroken token) would otherwise set a floor on the pane width and push the layout'
		).toBe(true);
	});

	it('stream pane (list) holds a fixed width and does not shrink when an item is open', () => {
		const streamClass = findUniqueClassAttr('overflow-x-hidden');
		expect(
			FIXED_PX_WIDTH.test(streamClass),
			'the stream pane must carry a fixed pixel width — widening or narrowing the browser must not change the width of the list of truncated one-line titles'
		).toBe(true);
		expect(
			streamClass.includes('shrink-0'),
			'the stream pane must carry shrink-0 alongside its fixed width so it never gets compressed by its flex sibling'
		).toBe(true);
	});
});

describe('pane-layout source-scan guard: three-band mobile takeover contract (D-02, 09.1-01-PLAN.md Task 3)', () => {
	it('stream pane carries the 1024px-and-up fixed-width band, byte-identical to the pre-phase layout', () => {
		const streamClass = findUniqueClassAttr('overflow-x-hidden');
		expect(
			streamClass.includes('lg:w-[480px]'),
			'the stream pane must carry lg:w-[480px] — the 1024px-and-above band is today\'s exact fixed width, and a future edit that drops the lg: band must fail here'
		).toBe(true);
	});

	it('stream pane carries a proportional (never fixed) 768-1024px mid-band', () => {
		const streamClass = findUniqueClassAttr('overflow-x-hidden');
		expect(
			streamClass.includes('md:w-[clamp('),
			'the stream pane must carry a md:w-[clamp(...)] mid-band — the 768-1024px band is proportional, never the fixed 480px'
		).toBe(true);
	});

	it('stream pane keeps its scroll box alive below 768px via invisible, never hidden', () => {
		const streamClass = findUniqueClassAttr('overflow-x-hidden');
		expect(
			streamClass.includes('max-md:invisible'),
			'the stream pane must carry max-md:invisible below 768px so its DOM node — and therefore its scroll offset (D-01) — survives while the takeover conceals it'
		).toBe(true);
		expect(
			streamClass.includes('max-md:hidden'),
			'the stream pane must NOT carry max-md:hidden — a display toggle destroys the scroll box D-01 requires preserved (planner_resolutions R1)'
		).toBe(false);
	});

	it('detail pane overlays the full viewport below 768px and reverts to sibling-flex above it', () => {
		const detailClass = findUniqueClassAttr('border-l');
		expect(
			detailClass.includes('fixed inset-0'),
			'the detail pane must carry fixed inset-0 — below 768px it is a full-viewport takeover (D-01/D-04), not a sibling flex pane'
		).toBe(true);
		expect(
			detailClass.includes('md:static'),
			'the detail pane must carry md:static — at 768px and above it reverts to sibling-flex behaviour identical to the pre-phase layout'
		).toBe(true);
	});
});
