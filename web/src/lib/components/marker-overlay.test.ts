// Structural + computed-contrast guard for the stream date-marker overlay
// (UI-11 gap closure G-06-6). 06-03-SUMMARY.md recorded the overlay's
// visual check as human judgment and it was never exercised — an
// arithmetically-obvious 2px-inset-versus-~11px-scrollbar error survived
// to the user as a result (.planning/debug/date-marker-tick-styling.md).
// The two properties that failed there — rest contrast and lane geometry
// — are both computable off disk without a component-mount harness (no
// DOM exists under this project's `environment: 'node'` vitest runner),
// so they become assertions here rather than judgement calls, following
// the same readFileSync/comment-stripping discipline as
// scrollbar-theme.test.ts (app.css) and pane-layout.test.ts (a .svelte
// route).

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const appCssPath = join(here, '..', '..', 'app.css');
const overlayPath = join(here, 'StreamDateMarkers.svelte');
const routePath = join(here, '..', '..', 'routes', 'w', '[webspace]', '+page.svelte');

const rawAppCss = readFileSync(appCssPath, 'utf-8');
const rawOverlay = readFileSync(overlayPath, 'utf-8');
const rawRoute = readFileSync(routePath, 'utf-8');

// Comments stripped (not deleted, so two declarations separated only by a
// comment can never fuse into one token) so no assertion below can be
// satisfied — or tripped — by prose rather than shipped markup/CSS. CSS
// block comments and Svelte/HTML comments are stripped independently,
// since a .svelte file carries both kinds.
function stripCssComments(source: string): string {
	return source.replace(/\/\*[\s\S]*?\*\//g, ' ');
}
function stripHtmlComments(source: string): string {
	return source.replace(/<!--[\s\S]*?-->/g, ' ');
}

const appCss = stripCssComments(rawAppCss);
const overlaySource = stripHtmlComments(stripCssComments(rawOverlay));
const routeSource = stripHtmlComments(stripCssComments(rawRoute));

describe('marker-overlay guard — found non-empty sources to scan', () => {
	// Guards against a silent no-op: a wrong path resolution must fail
	// loudly here rather than making every assertion below vacuously pass
	// over an empty string.
	it('app.css', () => {
		expect(appCss.length).toBeGreaterThan(0);
	});
	it('StreamDateMarkers.svelte', () => {
		expect(overlaySource.length).toBeGreaterThan(0);
	});
	it('+page.svelte', () => {
		expect(routeSource.length).toBeGreaterThan(0);
	});
});

// --- WCAG 2.x sRGB relative luminance + contrast ratio, small and pure,
// no dependency (mirrors the standard formula: contrast = (L1+0.05)/(L2+0.05)
// with L1 the lighter of the two relative luminances). ---

function srgbToLinear(channel: number): number {
	const c = channel / 255;
	return c <= 0.04045 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
}

function relativeLuminance([r, g, b]: [number, number, number]): number {
	const [rl, gl, bl] = [r, g, b].map(srgbToLinear);
	return 0.2126 * rl + 0.7152 * gl + 0.0722 * bl;
}

function contrastRatio(a: [number, number, number], b: [number, number, number]): number {
	const la = relativeLuminance(a);
	const lb = relativeLuminance(b);
	const [hi, lo] = la > lb ? [la, lb] : [lb, la];
	return (hi + 0.05) / (lo + 0.05);
}

function hexToRgb(hex: string): [number, number, number] {
	const n = parseInt(hex.replace('#', ''), 16);
	return [(n >> 16) & 255, (n >> 8) & 255, n & 255];
}

/**
 * Alpha-composites `fg` at `pct` (0-1) over `bg` — mirrors what
 * `color-mix(in srgb, fg pct%, transparent)` renders as once painted atop
 * `bg`, since the transparent component contributes nothing but lets `bg`
 * show through proportionally.
 */
function compositeOver(
	fg: [number, number, number],
	bg: [number, number, number],
	pct: number
): [number, number, number] {
	return [fg[0] * pct + bg[0] * (1 - pct), fg[1] * pct + bg[1] * (1 - pct), fg[2] * pct + bg[2] * (1 - pct)];
}

/** Parses a `--name: <value>;` custom-property declaration's raw value out of the comment-stripped app.css. */
function readCssVar(name: string): string {
	const match = appCss.match(new RegExp(`--${name}:\\s*([^;]+);`));
	expect(match, `expected to find a --${name} declaration in app.css`).not.toBeNull();
	return match![1].trim();
}

/** Parses the `color-mix(in srgb, var(--muted-foreground) N%, transparent)` percentage out of a raw declaration value. */
function parseMutedForegroundMixPercent(rawValue: string, label: string): number {
	const match = rawValue.match(
		/color-mix\(in srgb,\s*var\(--muted-foreground\)\s*(\d+)%,\s*transparent\)/
	);
	expect(
		match,
		`expected ${label} to be declared as color-mix(in srgb, var(--muted-foreground) N%, transparent) — got "${rawValue}". A future rewrite to some other form must fail loudly here rather than silently skipping the contrast check.`
	).not.toBeNull();
	return Number(match![1]) / 100;
}

// A `#` followed by exactly six hex digits — a hardcoded literal color
// value, the thing the marker tokens must never carry (they must resolve
// exclusively through the derived color-mix() of an existing palette
// token, per the derived-token discipline scrollbar-theme.test.ts already
// guards for the scrollbar tokens).
const HEX_LITERAL = /#[0-9a-fA-F]{6}\b/;

describe('marker-overlay guard — computed rest-tone contrast (WCAG 1.4.11, >= 3:1)', () => {
	const mutedForeground = hexToRgb(readCssVar('muted-foreground'));
	const background = hexToRgb(readCssVar('background'));
	const card = hexToRgb(readCssVar('card'));

	// The retired token this guard exists to never regress back to
	// (.planning/debug/date-marker-tick-styling.md): --scrollbar-thumb at
	// 35% alpha computed to 1.86:1 against --background, below the 3:1
	// floor and the mechanical cause of G-06-6. Included in every failure
	// message below so a future regression report explains itself without
	// re-deriving the number.
	const retiredThumbContrast = contrastRatio(
		compositeOver(mutedForeground, background, 0.35),
		background
	);

	for (const name of ['stream-marker', 'stream-marker-strong'] as const) {
		it(`--${name} is declared as a color-mix() of --muted-foreground`, () => {
			const raw = readCssVar(name);
			parseMutedForegroundMixPercent(raw, `--${name}`);
		});

		it(`--${name} carries no six-digit hex literal`, () => {
			const raw = readCssVar(name);
			expect(
				HEX_LITERAL.test(raw),
				`--${name} must contain no six-digit hex literal, got "${raw}"`
			).toBe(false);
		});

		it(`--${name} clears 3:1 against both --background and --card`, () => {
			const raw = readCssVar(name);
			const pct = parseMutedForegroundMixPercent(raw, `--${name}`);
			const overBackground = contrastRatio(compositeOver(mutedForeground, background, pct), background);
			const overCard = contrastRatio(compositeOver(mutedForeground, card, pct), card);
			expect(
				overBackground,
				`--${name} computes to ${overBackground.toFixed(2)}:1 against --background — below the 3:1 non-text floor (the retired --scrollbar-thumb token computed to ${retiredThumbContrast.toFixed(2)}:1, the mechanical cause of G-06-6)`
			).toBeGreaterThanOrEqual(3.0);
			expect(
				overCard,
				`--${name} computes to ${overCard.toFixed(2)}:1 against --card — below the 3:1 non-text floor (the retired --scrollbar-thumb token computed to ${retiredThumbContrast.toFixed(2)}:1, the mechanical cause of G-06-6)`
			).toBeGreaterThanOrEqual(3.0);
		});
	}
});

describe('marker-overlay guard — lane geometry clears the native scrollbar', () => {
	it('the ::-webkit-scrollbar fallback rule declares a numeric width in app.css', () => {
		const block = appCss.match(/::-webkit-scrollbar\s*\{([^}]*)\}/);
		expect(block, 'expected a ::-webkit-scrollbar { ... } rule in app.css').not.toBeNull();
		expect(/width:\s*\d+px/.test(block![1]), 'expected a numeric width: Npx inside ::-webkit-scrollbar').toBe(
			true
		);
	});

	it("the overlay container's right offset is at least the declared scrollbar width", () => {
		const scrollbarBlock = appCss.match(/::-webkit-scrollbar\s*\{([^}]*)\}/)![1];
		const scrollbarWidthPx = Number(scrollbarBlock.match(/width:\s*(\d+)px/)![1]);

		const rightOffsetMatches = [...overlaySource.matchAll(/right-\[(\d+)px\]/g)];
		expect(
			rightOffsetMatches.length,
			'expected exactly one right-[Npx] arbitrary-value offset in StreamDateMarkers.svelte (the overlay container) — found ' +
				rightOffsetMatches.length
		).toBe(1);
		const containerRightPx = Number(rightOffsetMatches[0][1]);

		expect(
			containerRightPx,
			`the overlay container's right offset (${containerRightPx}px) must be at least app.css's own declared scrollbar width (${scrollbarWidthPx}px), derived from app.css rather than a second hardcoded copy of the number — otherwise the lane sits on top of the native scrollbar again (G-06-6's original defect)`
		).toBeGreaterThanOrEqual(scrollbarWidthPx);
	});
});

describe('marker-overlay guard — form and token wiring', () => {
	it('tick classes resolve through the dedicated marker tokens, never the scrollbar thumb tokens', () => {
		expect(overlaySource.includes('--stream-marker')).toBe(true);
		expect(
			overlaySource.includes('--scrollbar-thumb'),
			'the tick must no longer reference --scrollbar-thumb/--scrollbar-thumb-hover — reusing the scrollbar thumb tokens is what produced G-06-6 (a 1.86:1 rest contrast)'
		).toBe(false);
	});

	it('a rail element is rendered', () => {
		expect(
			overlaySource.includes('bg-border'),
			'expected a rail element painted in the --border token to group the ticks into a ruler'
		).toBe(true);
	});

	it('the tick markup branches on the major flag', () => {
		expect(overlaySource.includes('marker.major')).toBe(true);
	});

	it('cursor-pointer is present on the tick', () => {
		expect(overlaySource.includes('cursor-pointer')).toBe(true);
	});

	it('a focus-visible ring utility is present', () => {
		expect(overlaySource.includes('focus-visible:ring')).toBe(true);
	});

	it('markerLaneTop is used for vertical placement', () => {
		expect(overlaySource.includes('markerLaneTop(')).toBe(true);
	});

	it('streamScrolls gates the derived marker list', () => {
		expect(overlaySource.includes('streamScrolls(')).toBe(true);
	});

	it('CSS.escape still wraps the item id in the click handler', () => {
		expect(overlaySource.includes('CSS.escape(itemId)')).toBe(true);
	});
});

describe('marker-overlay guard — the stream scroll region reserves the gutter', () => {
	it('the stream scroll region in +page.svelte carries a right padding', () => {
		expect(
			/\bpr-(\d+|\[\d+px\])\b/.test(routeSource),
			'expected a pr-<n> (or pr-[Npx]) right-padding utility on the stream scroll region — reclaiming this gutter would put the ruler lane back on top of the row list, reintroducing the two-tone banding defect this gap closure exists to fix'
		).toBe(true);
	});
});
