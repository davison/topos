// G-06-3b (Phase 6 UAT round 2): the merged source chip shipped as a
// polished-looking pill with three co-located geometry/interaction defects.
// (1) The chip's height was inherited from its tallest child (a 44px
// refresh button) rather than declared by the pill itself, so the chip
// drifted ~10px taller than the `h-11` overflow trigger sitting beside it in
// the same row, and the wrapper's own padding opened bands above and below
// the filter button that looked clickable and were not. A pill whose height
// is inherited rather than declared will always drift away from the row's
// other controls this way. (2) The refresh Button's class override set a
// size but no radius, so the shared button base's rectangular radius shaped
// the hover fill as a rounded square inside the rounded-full pill. (3) The
// refresh reveal was keyed on the wrapper's `:focus-within` state, which
// also matches the persistent focus a mouse click leaves on a button — a
// reveal keyed on any-focus rather than keyboard-focus will always pin
// itself visible after a mouse click, because a mouse click focuses a
// button.
//
// G-06-3 (the sibling incident) got a guard for the chip's *colour*
// (source-chip-selected.test.ts); this is the guard for its *geometry and
// interaction surface* — the other axis on which this component has now
// shipped a defect. The two files are kept separate: they fail for
// different reasons and should read as different incidents.
//
// House pattern (matches source-chip-selected.test.ts): comment-stripped
// source scanning (web/vite.config.ts's test block runs environment: 'node'
// with no component-mount harness, so there is no DOM to render against),
// `extractBetween` scoping, a found-non-empty-source guard first, and one
// consequence-describing message per assertion. All three assertion groups
// below scan comment-stripped source, so no prose in the component —
// including a comment quoting a class name — can satisfy or trip a
// structural check. Helpers are redefined here rather than imported, same
// as the sibling file.

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
// assertion to one element's opening tag or one element's block rather than
// the whole file, so a match elsewhere in the document can never satisfy or
// trip it by accident. Asserts both markers were found so a bad marker fails
// loudly instead of vacuously passing.
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

describe('source-chip-pill guard: found non-empty comment-stripped sources', () => {
	it('SourceChip.svelte', () => {
		expect(strippedChip.length).toBeGreaterThan(0);
	});
	it('WebspaceHeader.svelte', () => {
		expect(strippedHeader.length).toBeGreaterThan(0);
	});
});

// Four scoped slices: the chip wrapper's opening tag, the filter button
// element, the refresh Button element, and the overflow trigger's opening
// tag in the header. The filter button uses a lowercase `<button` marker
// (the refresh control is a capitalised component tag, so this matches only
// the filter button). The overflow trigger is located via the literal
// `more sources` text inside its aria-label — not `<button`, since
// WebspaceHeader.svelte has more than one `<button` element — through the
// following `>`; the popover's own "More sources" heading uses different
// capitalisation and cannot collide with this lowercase marker.
const wrapperTag = extractBetween(strippedChip, '<div', '>');
const filterButtonBlock = extractBetween(strippedChip, '<button', '</button>');
const refreshButtonBlock = extractBetween(strippedChip, '<Button', '</Button>');
const overflowTriggerTag = extractBetween(strippedHeader, 'more sources', '>');

describe('height parity: the chip and the overflow trigger declare the same height', () => {
	it('both the wrapper and the overflow trigger carry the h-11 height utility', () => {
		expect(
			/\bh-11\b/.test(wrapperTag),
			"expected the chip wrapper's opening tag to declare h-11 — these two controls sit side by side in one row; when the chip declares no height of its own it inherits its tallest child's and overshoots the trigger, which is what made the row read as ragged"
		).toBe(true);
		expect(
			/\bh-11\b/.test(overflowTriggerTag),
			'expected the overflow trigger to still declare h-11 — the height-parity check above is only meaningful if this literal has not drifted on the other side of the row'
		).toBe(true);
	});
});

describe('no dead bands on the pill', () => {
	it('the wrapper carries no vertical, all-sides, or left padding utility of its own', () => {
		expect(
			/\bpy-\d/.test(wrapperTag),
			"found a vertical padding utility on the chip wrapper — the filter button stretches to the wrapper's content box, so vertical padding on the wrapper is surface that looks like chip and does not toggle the filter (this IS G-06-3b's first defect)"
		).toBe(false);
		expect(
			/(^|\s)p-\d/.test(wrapperTag),
			"found an all-sides padding utility on the chip wrapper — the filter button stretches to the wrapper's content box, so padding on the wrapper is surface that looks like chip and does not toggle the filter (this IS G-06-3b's first defect)"
		).toBe(false);
		expect(
			/\bpl-\d/.test(wrapperTag),
			'found a left padding utility on the chip wrapper — that strip belongs on the filter button (which owns the hit area), not on the wrapper it sits inside, or it becomes a click-dead moat around the button rather than part of it'
		).toBe(false);
	});
});

describe('the filter button owns the full pill surface', () => {
	it('the filter button self-stretches and carries its own left padding', () => {
		expect(
			/\bself-stretch\b/.test(filterButtonBlock),
			"expected the filter button to carry self-stretch — without it the button shrink-wraps its ~20px text line and the rest of the pill is click-dead, reopening G-06-3b's first defect"
		).toBe(true);
		expect(
			/\bpl-\d/.test(filterButtonBlock),
			"expected the filter button to carry its own left padding — that padding must live on the button so the strip between the pill's edge and the health dot is part of the hit area, not a moat around it"
		).toBe(true);
	});
});

describe('circular refresh surface, and no 44px height driver', () => {
	it('the refresh Button carries a full-round radius and does not carry the 44px square-size utility', () => {
		expect(
			/\brounded-full\b/.test(refreshButtonBlock),
			"expected the refresh Button to carry rounded-full explicitly — the shared button base supplies a rectangular radius, so an override that sets a size without a radius paints a rounded square inside the oval (this IS G-06-3b's second defect)"
		).toBe(true);
		expect(
			/\bsize-11\b/.test(refreshButtonBlock),
			"found size-11 on the refresh Button — a 44px child cannot fit inside a 44px border-box pill, so reinstating that size silently re-inflates the chip and reopens G-06-3b's first defect"
		).toBe(false);
	});
});

describe('reveal scoping: hover and keyboard focus only, never a mouse-click pin', () => {
	it('the refresh Button reveals on hover and keyboard focus, and no longer on focus-within', () => {
		expect(
			refreshButtonBlock.includes('group-hover:opacity-100'),
			"expected the refresh Button to still reveal on group-hover — this is D-03's primary reveal path"
		).toBe(true);
		expect(
			refreshButtonBlock.includes('group-has-[:focus-visible]:opacity-100'),
			"expected the refresh Button to reveal on group-has-[:focus-visible] — this is D-03's keyboard clause, and dropping it strands keyboard users on an invisible control"
		).toBe(true);
		expect(
			refreshButtonBlock.includes('group-focus-within:opacity-100'),
			"found a focus-within-scoped reveal on the refresh Button — focus-within also matches the persistent focus a mouse click leaves on the button, pinning the icon visible until the user clicks elsewhere (this IS G-06-3b's third defect)"
		).toBe(false);
	});
});

describe('syncing spinner survives the reveal-scoping change', () => {
	it('the refresh Button still forces itself visible while the source is syncing', () => {
		expect(
			/source\.syncing\s*&&\s*'opacity-100'/.test(refreshButtonBlock),
			"expected the refresh Button's class expression to still force opacity-100 while source.syncing is true — the spinning icon is the sole in-place syncing indicator this phase (D-03), so a reveal-scoping change must never be able to hide it mid-sync"
		).toBe(true);
	});
});
