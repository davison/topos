// 07-01-PLAN.md Task 2's structural guard over the D-19 filter-chip
// distinctness rule and the E9/E10 loading/empty contracts (07-UI-SPEC.md):
// FilterChip must read as visibly different from SourceChip at a glance
// (rounded-md vs. rounded-full — proven as a DIFFERENCE between the two
// files, never a hardcoded literal, so the guard fails if either file's
// shape drifts toward the other rather than only if one specific class
// disappears), the filter row must be absent (not empty-styled) at zero
// active filters, the remove control must carry a discoverable aria-label,
// and both the save affordance and the remove control must be disableable
// while a write is in flight.
//
// House pattern (matches source-chip-pill.test.ts and
// search-emphasis.test.ts): comment-stripped source scanning
// (web/vite.config.ts's test block runs environment: 'node' with no
// component-mount harness), `extractBetween` scoping so a match elsewhere
// in the document can never satisfy or trip an assertion by accident, a
// found-non-empty-source guard first, and one consequence-describing
// message per assertion.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));

const filterChipPath = join(here, 'FilterChip.svelte');
const sourceChipPath = join(here, 'SourceChip.svelte');
const headerPath = join(here, 'WebspaceHeader.svelte');

const rawFilterChip = readFileSync(filterChipPath, 'utf-8');
const rawSourceChip = readFileSync(sourceChipPath, 'utf-8');
const rawHeader = readFileSync(headerPath, 'utf-8');

// Strips HTML comments, CSS/JS block comments and JS line comments, each
// replaced with a single space (never deleted outright) so two tokens
// separated only by a comment can never fuse into one identifier — no
// assertion below can be satisfied or tripped by prose (including this very
// file's own header, or the components' doc comments).
function stripComments(source: string): string {
	return source
		.replace(/<!--[\s\S]*?-->/g, ' ')
		.replace(/\/\*[\s\S]*?\*\//g, ' ')
		.replace(/\/\/.*$/gm, ' ');
}

const strippedFilterChip = stripComments(rawFilterChip);
const strippedSourceChip = stripComments(rawSourceChip);
const strippedHeader = stripComments(rawHeader);

// Slices the text from the first occurrence of `startMarker` through the
// following occurrence of `endMarker` (inclusive), asserting both markers
// were found so a bad marker fails loudly instead of vacuously passing.
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

describe('filter-chip guard: found non-empty comment-stripped sources', () => {
	it('FilterChip.svelte', () => {
		expect(strippedFilterChip.length).toBeGreaterThan(0);
	});
	it('SourceChip.svelte', () => {
		expect(strippedSourceChip.length).toBeGreaterThan(0);
	});
	it('WebspaceHeader.svelte', () => {
		expect(strippedHeader.length).toBeGreaterThan(0);
	});
});

// FilterChip's wrapper is the first <div ...> element; SourceChip's wrapper
// is likewise its first <div ...> element — both components open with
// exactly one top-level wrapper div before any nested control.
const filterChipWrapperTag = extractBetween(strippedFilterChip, '<div', '>');
const sourceChipWrapperTag = extractBetween(strippedSourceChip, '<div', '>');

describe('D-19 chip-shape distinctness: FilterChip and SourceChip disagree on radius', () => {
	it('FilterChip carries rounded-md and SourceChip does not', () => {
		expect(
			/\brounded-md\b/.test(filterChipWrapperTag),
			'expected the FilterChip wrapper to declare rounded-md — this is the one geometric difference that makes a filter chip legible as "a different kind of thing" from a source chip at a glance'
		).toBe(true);
		expect(
			/\brounded-md\b/.test(sourceChipWrapperTag),
			"found rounded-md on the SourceChip wrapper — if both chips converge on the same radius, the D-19 distinctness rule is gone even though FilterChip's own literal is unchanged"
		).toBe(false);
	});
	it('SourceChip carries rounded-full and FilterChip does not', () => {
		expect(
			/\brounded-full\b/.test(sourceChipWrapperTag),
			'expected the SourceChip wrapper to still declare rounded-full — this guard is only meaningful if this literal has not drifted on the other side of the comparison'
		).toBe(true);
		expect(
			/\brounded-full\b/.test(filterChipWrapperTag),
			'found rounded-full on the FilterChip wrapper — a filter chip sharing SourceChip\'s exact radius would be indistinguishable from it at a glance, reopening D-19'
		).toBe(false);
	});
});

describe('remove control aria-label', () => {
	it('FilterChip carries a "Remove filter" aria-label', () => {
		expect(
			strippedFilterChip.includes('Remove filter'),
			'expected the remove control to carry a "Remove filter" aria-label, so its purpose is discoverable independent of the visual x icon'
		).toBe(true);
	});
});

// The filter row block in WebspaceHeader.svelte: located via the FilterChip
// usage itself rather than a bare `{#if`, since the header has more than
// one `{#if` block — the marker anchors on the one guarding FilterChip.
const filterRowBlock = extractBetween(strippedHeader, '{#if filters.length', '{/if}');

describe('E9 empty: the filter row is absent (not empty-styled) at zero filters', () => {
	it('WebspaceHeader gates the filter row on filters.length', () => {
		expect(
			/\{#if filters\.length\s*>\s*0\s*\|\|\s*Object\.keys\(filterBySource\)\.length\s*>\s*0\}/.test(strippedHeader),
			'expected the filter row to be gated behind the filters.length / filterBySource check — with zero active filters the row must be absent entirely, not rendered empty-styled'
		).toBe(true);
	});
	it('the gated block actually renders a FilterChip', () => {
		expect(
			filterRowBlock.includes('<FilterChip'),
			'expected the filters.length-gated block to render FilterChip — otherwise the gate is guarding the wrong content'
		).toBe(true);
	});
});

describe('E9/E10 loading: both the save affordance and the remove control are disableable', () => {
	it('the "Save as filter" button receives a disabled binding', () => {
		const saveButtonBlock = extractBetween(strippedHeader, 'Save as filter', '</Button>');
		expect(
			/disabled=\{filterBusy\}/.test(strippedHeader) || /disabled=\{filterBusy\}/.test(saveButtonBlock),
			'expected the "Save as filter" button to carry disabled={filterBusy} — it must disable while a write is in flight (E10 loading)'
		).toBe(true);
	});
	it('FilterChip forwards a disabled prop to its remove control', () => {
		const removeButtonBlock = extractBetween(strippedFilterChip, '<Button', '</Button>');
		expect(
			/\{disabled\}/.test(removeButtonBlock) || /disabled=\{disabled\}/.test(removeButtonBlock),
			"expected the remove control to receive FilterChip's own disabled prop — it must disable while a write is in flight (E9 loading)"
		).toBe(true);
	});
});
