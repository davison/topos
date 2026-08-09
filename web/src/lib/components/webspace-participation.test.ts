// 07-14-PLAN.md Task 3's structural guard over WebspaceHeader.svelte and
// AddSourceModal.svelte, closing 07-UAT.md `G-07-6`. G-07-6 had TWO
// independent, each-individually-sufficient defects — a config write that
// silently no-op'd for an all-participate webspace (config-edit.ts's
// removeSourceFromWebspace, Task 2), and this file's target: the header's
// chip row consumed the kernel-wide `sources` prop verbatim, with no
// webspace-participation filter anywhere, so even a fully correct config
// write would never make the removed chip disappear. Both had to be fixed
// together for the documented "the chip disappears" behaviour to hold.
//
// The row-visibility split asserted below is deliberate, not an oversight:
// `shouldShowSourceRows` answers "does this installation have any
// configured source instances at all" (unfiltered `sources`), while the
// chip CONTENT answers "which of them belong to this webspace" (the
// filtered `participatingSources`). Filtering the gate's own input would
// hide the "+" add-source trigger for any webspace with zero
// participants — every freshly created one (07-11) and any webspace whose
// last source was just removed (this plan's Task 2) — replacing one dead
// end with another.
//
// House pattern (matches add-source.test.ts / webspace-switcher.test.ts /
// source-chip-pill.test.ts): comment-stripped source scanning, a
// found-non-empty-source guard first, and one consequence-describing
// message per assertion.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const headerPath = join(here, 'WebspaceHeader.svelte');
const modalPath = join(here, 'AddSourceModal.svelte');

function stripComments(source: string): string {
	return source
		.replace(/<!--[\s\S]*?-->/g, ' ')
		.replace(/\/\*[\s\S]*?\*\//g, ' ')
		.replace(/\/\/.*$/gm, ' ');
}

function extractBetween(source: string, startMarker: string, endMarker: string): string {
	const startIndex = source.indexOf(startMarker);
	expect(
		startIndex,
		`expected to find "${startMarker}" in the scanned source`
	).toBeGreaterThanOrEqual(0);
	const endIndex = source.indexOf(endMarker, startIndex + startMarker.length);
	expect(endIndex, `expected to find "${endMarker}" after "${startMarker}"`).toBeGreaterThan(
		startIndex
	);
	return source.slice(startIndex, endIndex);
}

function countOccurrences(source: string, needle: string): number {
	return source.split(needle).length - 1;
}

const headerRaw = readFileSync(headerPath, 'utf-8');
const headerStripped = stripComments(headerRaw);
const modalRaw = readFileSync(modalPath, 'utf-8');
const modalStripped = stripComments(modalRaw);

describe('webspace-participation guard: found non-empty comment-stripped source', () => {
	it('WebspaceHeader.svelte', () => {
		expect(headerStripped.length).toBeGreaterThan(0);
	});

	it('AddSourceModal.svelte', () => {
		expect(modalStripped.length).toBeGreaterThan(0);
	});
});

describe('WebspaceHeader.svelte: chip content is filtered through the shared participation predicate', () => {
	it('imports participatesIn from the shared participation module', () => {
		expect(
			headerStripped.includes("participatesIn") &&
				headerStripped.includes("from '$lib/participation'"),
			'the header must import the shared predicate rather than deriving its own filter — this is what keeps it from disagreeing with the picker and with config-edit.ts'
		).toBe(true);
	});

	it('derives a filtered list from the sources prop using participatesIn', () => {
		expect(
			headerStripped.includes('participatingSources = $derived(') &&
				headerStripped.includes('sources.filter(') &&
				headerStripped.includes('participatesIn(config, webspace, source.name)'),
			'the chip row must derive its content from a filtered list — without this, every configured instance renders as a chip regardless of which webspace is open'
		).toBe(true);
	});

	it('the visible slice is derived from the filtered list, not the raw sources prop', () => {
		expect(
			headerStripped.includes('visibleSources = $derived(participatingSources.slice(0, visibleCount))'),
			'a visible slice built from the raw sources prop would show a chip for every configured instance, including ones this webspace does not participate in'
		).toBe(true);
	});

	it('the hidden slice is derived from the filtered list, not the raw sources prop', () => {
		expect(
			headerStripped.includes('hiddenSources = $derived(participatingSources.slice(visibleCount))'),
			'a hidden (overflow-popover) slice built from the raw sources prop would offer a non-participating instance from the "+N more" popover'
		).toBe(true);
	});

	it('no chip-content derivation slices the raw sources prop directly', () => {
		// Every `sources.slice(` in the file must be prefixed by
		// `participating` (i.e. be `participatingSources.slice(`) — a bare
		// `sources.slice(` would mean some derivation fell back to the
		// unfiltered kernel-wide list.
		const bareSourceSlices = (headerStripped.match(/(?<!participating)sources\.slice\(/g) ?? [])
			.length;
		expect(
			bareSourceSlices,
			'found a chip-content derivation slicing the raw, unfiltered sources prop directly — the visible and hidden slices must both read participatingSources'
		).toBe(0);
	});

	it('the off-screen measurement clone loop iterates the filtered list, so measured widths correspond to the rendered chips', () => {
		const measureBlock = extractBetween(headerStripped, 'bind:this={measureEl}', '</div>');
		expect(
			measureBlock.includes('{#each participatingSources as source'),
			'a measurement clone loop iterating the raw sources prop would measure widths for chips that are never actually rendered, desynchronising the overflow math'
		).toBe(true);
	});

	it("the overflow-trigger measurement button's count label reads the filtered list", () => {
		const triggerMeasureBlock = extractBetween(
			headerStripped,
			'bind:this={overflowTriggerMeasureEl}',
			'</button>'
		);
		expect(
			triggerMeasureBlock.includes('+{participatingSources.length}'),
			'the hidden overflow-trigger clone must report the filtered count — a raw sources.length here would report a stale "+N" once a webspace filters some instances out'
		).toBe(true);
	});

	it('shouldShowSourceRows is still called with the raw, unfiltered sources prop (the deliberate split)', () => {
		expect(
			headerStripped.includes('shouldShowSourceRows(sourcesState, sources)'),
			'row VISIBILITY answers "does this installation have any configured source instances at all" — a different question from "which of them belong to this webspace". Filtering this gate\'s input would hide the "+" add-source trigger for every webspace with zero participants (every freshly created one, and any webspace whose last source was just removed), replacing one dead end with another.'
		).toBe(true);
	});

	it('the "+" AddSourceModal mount stays inside the showSourceRows block and is reached when the filtered list is empty', () => {
		const rowBlock = extractBetween(headerStripped, '{#if showSourceRows}', '<SearchBox');
		expect(
			rowBlock.includes('<AddSourceModal'),
			'the add-source trigger must still be inside the row block that showSourceRows (not participatingSources) gates'
		).toBe(true);
		// The AddSourceModal mount is gated only on `config` (it needs a
		// resolved config to compute the picker list) — never on
		// participatingSources or its length, which would hide the "+"
		// exactly when it is most needed: a webspace with no participants.
		expect(
			rowBlock.includes('{#if config}') && !rowBlock.includes('participatingSources.length > 0'),
			'the "+" trigger must be reachable regardless of how many sources currently participate, including zero'
		).toBe(true);
	});
});

describe('AddSourceModal.svelte: no longer computes its own participant set', () => {
	it('does not declare a local participant-set derivation', () => {
		expect(
			countOccurrences(modalStripped, 'participatingSet'),
			'AddSourceModal must not keep its own copy of the participation predicate — a second implementation is exactly the divergence mechanism that produced G-07-6\'s second half'
		).toBe(0);
	});

	it('imports and calls the shared participatingInstances helper instead', () => {
		expect(
			modalStripped.includes("participatingInstances") &&
				modalStripped.includes("from '$lib/participation'") &&
				modalStripped.includes('participatingInstances(config, webspace)'),
			'the picker\'s "not yet in this webspace" list must be computed via the same shared predicate the header\'s chip row uses'
		).toBe(true);
	});
});

describe('exactly one client-side implementation of participation', () => {
	it('participatesIn is called exactly once in WebspaceHeader.svelte — no second, competing computation', () => {
		expect(
			countOccurrences(headerStripped, 'participatesIn('),
			'a second call site (or a hand-rolled equivalent) would let the header\'s filter drift from the shared predicate over time'
		).toBe(1);
	});

	it('participatingInstances is called exactly once in AddSourceModal.svelte', () => {
		expect(
			countOccurrences(modalStripped, 'participatingInstances('),
			'a second call site would risk the picker and its own internal logic disagreeing with each other'
		).toBe(1);
	});
});
