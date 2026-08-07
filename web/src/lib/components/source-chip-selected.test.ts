// G-06-3 (Phase 6 UAT): a selected source chip shipped visually
// indistinguishable from an unselected one. The markup, the ARIA and the
// behaviour were all correct and the class string even matched what
// 06-UI-SPEC.md's prose spelled at the time — the defect existed only in
// rendered contrast, which no prior behavioural test could have caught. In
// this palette the token literally named "accent" (`--accent`) is a
// deliberately NEUTRAL dark slate, byte-identical to `--border`; the
// palette's real accent blue lives in the `--primary`/`--ring` family
// (see the standing comment in app.css above the `--accent` declaration).
// A selected-state treatment built from `bg-accent`/`ring-accent` is
// invisible by construction. This guard scans SourceChip.svelte's source
// off disk (no component-mount harness exists — web/vite.config.ts's test
// block is `environment: 'node'`) to assert the selected treatment resolves
// through the primary token family, that every child re-tones, and that the
// D-01/D-02/D-03/D-04 behaviour this styling fix must not disturb stays
// intact.
//
// House pattern: pane-layout.test.ts (svelte class-attribute scanning),
// scrollbar-theme.test.ts (comment-stripped stylesheet scanning) and
// search-emphasis.test.ts (extractBetween scoping + stripComments). No
// assertion below runs against raw, unstripped source — a doc comment
// mentioning "bg-accent" in passing (this very file's header, for instance,
// if it were the scanned file) must never be able to satisfy or trip a
// structural assertion.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));

const chipPath = join(here, 'SourceChip.svelte');
const headerPath = join(here, 'WebspaceHeader.svelte');

const rawChip = readFileSync(chipPath, 'utf-8');
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

const strippedChip = stripComments(rawChip);
const strippedHeader = stripComments(rawHeader);

// Slices the text from the first occurrence of `startMarker` through the
// following occurrence of `endMarker` (inclusive) — used to scope an
// assertion to one element's class expression or one function body rather
// than the whole file, so a match elsewhere in the document (e.g. the
// neutral-token check needing to fail specifically inside the wrapper's own
// expression) can never satisfy or trip it by accident.
function extractBetween(source: string, startMarker: string, endMarker: string): string {
	const startIndex = source.indexOf(startMarker);
	expect(
		startIndex,
		`expected to find "${startMarker}" in the scanned source`
	).toBeGreaterThanOrEqual(0);
	const endIndex = source.indexOf(endMarker, startIndex);
	expect(
		endIndex,
		`expected to find "${endMarker}" after "${startMarker}"`
	).toBeGreaterThan(startIndex);
	return source.slice(startIndex, endIndex + endMarker.length);
}

// The wrapper div's class expression, the display-name span's class
// expression, the health-dot span's class expression and the refresh
// Button's element (attrs + children), each scoped to just that fragment.
const wrapperClassExpr = extractBetween(strippedChip, "'group flex shrink-0", ')}');
const displayNameClassExpr = extractBetween(strippedChip, "'truncate text-[14px]", ')}');
const healthDotClassExpr = extractBetween(strippedChip, "'size-2 shrink-0", ')}');
const refreshButtonBlock = extractBetween(strippedChip, '<Button', '</Button>');
const refreshHandlerBody = extractBetween(
	strippedChip,
	'function handleRefreshClick(event: MouseEvent) {',
	'}'
);

describe('source-chip-selected guard: found non-empty comment-stripped sources', () => {
	// Guards against a silent no-op: a wrong path resolution must fail
	// loudly here rather than making every later assertion vacuously pass
	// over an empty string.
	it('SourceChip.svelte', () => {
		expect(strippedChip.length).toBeGreaterThan(0);
	});
	it('WebspaceHeader.svelte', () => {
		expect(strippedHeader.length).toBeGreaterThan(0);
	});
});

describe('the selected treatment resolves through the real accent blue', () => {
	it("the wrapper's selected-keyed class branch names the primary token family", () => {
		expect(
			/selected\s*&&\s*'[^']*\bprimary\b[^']*'/.test(wrapperClassExpr),
			'expected the wrapper div\'s cn() call to carry a `selected && \'...primary...\'` branch — the selected chip must fill through --primary, the token this palette actually uses for its accent blue'
		).toBe(true);
	});

	it('the wrapper never reaches for the neutral --accent token for its selected treatment', () => {
		expect(
			/\bbg-accent\b/.test(wrapperClassExpr),
			'found `bg-accent` in the wrapper class expression — --accent is a neutral slate in this palette, byte-identical to --border, so a selected treatment built from it is invisible by construction (this IS the G-06-3 regression)'
		).toBe(false);
		expect(
			/\bring-accent\b/.test(wrapperClassExpr),
			'found `ring-accent` in the wrapper class expression — --accent is a neutral slate in this palette, byte-identical to --border, so a selected treatment built from it is invisible by construction (this IS the G-06-3 regression)'
		).toBe(false);
	});
});

describe('every child re-tones when selected, not only the wrapper', () => {
	// The original defect was precisely that only the wrapper responded to
	// `selected` — the label, dot and refresh icon stayed on their
	// unselected tones, illegible against any wrapper fill a future fix
	// might add. Each check below is scoped to its own element's class
	// expression, so a `selected` reference elsewhere in the file cannot
	// satisfy it by accident.
	it('the display-name span class expression references `selected`', () => {
		expect(
			displayNameClassExpr.includes('selected'),
			'expected the display-name span to re-tone conditionally on `selected` — an unselected-tone label on a selected fill is illegible'
		).toBe(true);
	});

	it('the health-dot span class expression references `selected`', () => {
		expect(
			healthDotClassExpr.includes('selected'),
			'expected the health-dot span to reference `selected` (e.g. an added ring) — the dot must stay a distinguishable pip against a filled background, not blend into it'
		).toBe(true);
	});

	it("the refresh button's class expression references `selected`", () => {
		expect(
			refreshButtonBlock.includes('selected'),
			'expected the refresh Button to re-tone conditionally on `selected` — its icon and hover surface must read on the fill, not on the neutral ghost-variant default'
		).toBe(true);
	});
});

describe('D-01/D-02/D-03/D-04 behaviour is untouched by the styling fix', () => {
	it('aria-pressed is still bound to `selected` (assistive-technology channel for the filter state)', () => {
		expect(
			/aria-pressed=\{selected\}/.test(strippedChip),
			'expected aria-pressed={selected} on the chip body button — this is the only channel telling assistive technology whether a source is filtered, independent of any visual fill'
		).toBe(true);
	});

	it('the refresh handler still stops propagation before triggering refresh', () => {
		expect(
			refreshHandlerBody.includes('event.stopPropagation()'),
			'expected handleRefreshClick to call event.stopPropagation() — without it, a refresh click would also toggle the chip filter (D-01/D-03 are siblings, not nested controls)'
		).toBe(true);
		expect(
			refreshHandlerBody.includes('onrefresh(source.name)'),
			'expected handleRefreshClick to still call onrefresh(source.name)'
		).toBe(true);
	});

	it('the refresh control still reveals on group-hover and focus-within (D-03)', () => {
		expect(
			refreshButtonBlock.includes('group-hover:opacity-100'),
			'expected the refresh Button to still reveal on group-hover — a styling fix for the selected fill must not regress the hover-reveal behaviour'
		).toBe(true);
		expect(
			refreshButtonBlock.includes('group-focus-within:opacity-100'),
			'expected the refresh Button to still reveal on group-focus-within — keyboard users must still be able to tab to a visible refresh control'
		).toBe(true);
	});

	it('the refresh control still forces itself visible while syncing (D-03)', () => {
		expect(
			/source\.syncing\s*&&\s*'opacity-100'/.test(refreshButtonBlock),
			"expected the refresh Button's class expression to still force opacity-100 while source.syncing is true — the spinning icon is the sole in-place syncing indicator this phase and must never be hidden mid-sync"
		).toBe(true);
	});
});

describe("the chip row's overflow clipping (load-bearing for 06-04) is untouched", () => {
	it('WebspaceHeader.svelte still carries overflow-hidden on the visible chip row', () => {
		expect(
			/overflow-hidden/.test(strippedHeader),
			"expected WebspaceHeader.svelte's chip row to still carry overflow-hidden — removing it to \"fix\" ring clipping would silently break the 06-04 overflow measurement, which depends on the row clipping its own children (this plan's Task 1 deliberately chose a fill instead, which needs no clip change)"
		).toBe(true);
	});
});
