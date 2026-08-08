// 07-04-PLAN.md Task 3's structural guard over SourceChip.svelte's new
// edit-menu control (D-12): the aria-label prefix and size-8 sizing, the
// stopPropagation-before-callback click handler (the D-12 vs Phase 6 D-01
// collision — a click must never also toggle the chip's filter) with the
// filter button's own handler left untouched, the exact three-item-plus-
// separator menu contents (no instance-deletion item — D-12/D-13's "exactly
// one place in the app deletes an instance" rule), the remove item's
// destructive tint, and WebspaceHeader.svelte's measurement clones still
// wiring a no-op onedit.
//
// House pattern (matches source-chip-pill.test.ts / source-chip-selected.test.ts):
// comment-stripped source scanning, `extractBetween` scoping, a
// found-non-empty-source guard first, and one consequence-describing
// message per assertion.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));

const chipPath = join(here, 'SourceChip.svelte');
const headerPath = join(here, 'WebspaceHeader.svelte');

const rawChip = readFileSync(chipPath, 'utf-8');
const rawHeader = readFileSync(headerPath, 'utf-8');

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
	const endIndex = source.indexOf(endMarker, startIndex);
	expect(endIndex, `expected to find "${endMarker}" after "${startMarker}"`).toBeGreaterThan(
		startIndex
	);
	return source.slice(startIndex, endIndex + endMarker.length);
}

const strippedChip = stripComments(rawChip);
const strippedHeader = stripComments(rawHeader);

describe('chip-edit-menu guard: found non-empty comment-stripped sources', () => {
	it('SourceChip.svelte', () => {
		expect(strippedChip.length).toBeGreaterThan(0);
	});
	it('WebspaceHeader.svelte', () => {
		expect(strippedHeader.length).toBeGreaterThan(0);
	});
});

// The edit control's own Button element, inside the DropdownMenuTrigger's
// child snippet — located via its unique aria-label expression, then
// walked backward to that same element's own `<Button` opening (not
// `indexOf('<Button')`, which would find the earlier refresh Button
// instead).
const ariaLabelMarker = 'aria-label={`Edit ${source.display_name}`}';
const ariaLabelIndex = strippedChip.indexOf(ariaLabelMarker);
expect(ariaLabelIndex, `expected to find "${ariaLabelMarker}" in the scanned source`).toBeGreaterThanOrEqual(
	0
);
const editButtonStart = strippedChip.lastIndexOf('<Button', ariaLabelIndex);
const editButtonEnd = strippedChip.indexOf('</Button>', ariaLabelIndex);
const editButtonBlock = strippedChip.slice(editButtonStart, editButtonEnd + '</Button>'.length);
const editMenuBlock = extractBetween(strippedChip, '<DropdownMenuContent>', '</DropdownMenuContent>');
const filterButtonBlock = extractBetween(strippedChip, '<button', '</button>');
const editClickHandlerBody = extractBetween(
	strippedChip,
	'function handleEditClick(event: MouseEvent, triggerOnClick?: (e: MouseEvent) => void) {',
	'\n\t}'
);

describe('edit control: aria-label prefix and size-8 sizing', () => {
	it('the aria-label carries an "Edit " prefix', () => {
		expect(strippedChip.includes('aria-label={`Edit ${source.display_name}`}')).toBe(true);
	});

	it('the trigger button carries size-8 sizing', () => {
		expect(
			/\bsize-8\b/.test(editButtonBlock),
			'expected the edit-menu trigger to carry size-8 sizing, matching the refresh control\'s own sub-44px exception'
		).toBe(true);
	});
});

describe('click handling: stopPropagation before the callback, filter button untouched', () => {
	it('handleEditClick calls stopPropagation before invoking the trigger callback', () => {
		const stopIndex = editClickHandlerBody.indexOf('event.stopPropagation()');
		const callbackIndex = editClickHandlerBody.indexOf('triggerOnClick?.(event)');
		expect(stopIndex, 'expected handleEditClick to call event.stopPropagation()').toBeGreaterThanOrEqual(
			0
		);
		expect(
			callbackIndex,
			'expected handleEditClick to invoke the forwarded trigger callback'
		).toBeGreaterThanOrEqual(0);
		expect(
			stopIndex < callbackIndex,
			'expected stopPropagation() to run BEFORE the forwarded callback — this is the D-12 vs Phase 6 D-01 collision, and getting the order wrong would let an edit-menu open also toggle the chip\'s filter'
		).toBe(true);
	});

	it('the filter button\'s own click handler is untouched (still just onfilter(source.name))', () => {
		expect(
			filterButtonBlock.includes('onclick={() => onfilter(source.name)}'),
			'expected the filter button\'s click handler to remain exactly onclick={() => onfilter(source.name)} — a plain chip click must still only toggle the filter'
		).toBe(true);
	});
});

describe('menu contents: exactly three action items plus one separator, no delete-instance item', () => {
	it('contains exactly one DropdownMenuSeparator', () => {
		const separators = editMenuBlock.match(/<DropdownMenuSeparator/g) ?? [];
		expect(separators.length).toBe(1);
	});

	// Finds the index of the `>` that closes an opening JSX tag starting at
	// `start`, tracking `{...}` brace depth so a `>` inside an attribute
	// expression (e.g. an arrow function's own `=>`) is correctly skipped
	// — a naive `[^>]*`/first-newline heuristic breaks on this file's
	// multi-line attribute formatting (the "Remove…" item wraps its
	// `class`/`onSelect` attributes across two lines).
	function findOpenTagEnd(source: string, start: number): number {
		let depth = 0;
		for (let i = start; i < source.length; i += 1) {
			const ch = source[i];
			if (ch === '{') depth += 1;
			else if (ch === '}') depth -= 1;
			else if (ch === '>' && depth === 0) return i;
		}
		return -1;
	}

	it('the menu item label set equals exactly the three expected labels', () => {
		const items = [...editMenuBlock.matchAll(/<DropdownMenuItem[\s\S]*?<\/DropdownMenuItem>/g)].map(
			(m) => m[0]
		);
		const labels = items.map((raw) => {
			const tagEnd = findOpenTagEnd(raw, 0);
			expect(tagEnd, 'expected to find the opening tag\'s closing ">"').toBeGreaterThanOrEqual(0);
			const body = raw.slice(tagEnd + 1, raw.lastIndexOf('</DropdownMenuItem>'));
			return body
				.replace(/<[^>]+>/g, '')
				.replace(/\s+/g, ' ')
				.trim();
		});
		expect(
			labels,
			'expected the menu item label set to equal exactly [Edit connection…, Edit match settings…, Remove from this webspace] — proven by set equality, not by grepping for the absent word "Delete"'
		).toEqual(['Edit connection…', 'Edit match settings…', 'Remove from this webspace']);
	});
});

describe('remove item: destructive tint', () => {
	it('the "Remove from this webspace" item carries a destructive class', () => {
		const removeItemBlock = extractBetween(
			editMenuBlock,
			'Remove from this webspace',
			'</DropdownMenuItem>'
		);
		const precedingOpenTag = editMenuBlock.slice(
			editMenuBlock.lastIndexOf('<DropdownMenuItem', editMenuBlock.indexOf('Remove from this webspace')),
			editMenuBlock.indexOf('Remove from this webspace')
		);
		expect(
			/text-destructive/.test(precedingOpenTag) || /text-destructive/.test(removeItemBlock),
			'expected the "Remove from this webspace" item to carry a text-destructive tint class'
		).toBe(true);
	});
});

describe('measurement clones keep no-op handlers', () => {
	it('every SourceChip usage in WebspaceHeader.svelte with an inline onfilter no-op also carries an inline onedit no-op', () => {
		const cloneBlocks = [...strippedHeader.matchAll(/<SourceChip[\s\S]*?\/>/g)].filter((m) =>
			m[0].includes('onfilter={() => {}}')
		);
		expect(
			cloneBlocks.length,
			'expected to find at least one measurement-clone SourceChip usage (onfilter={() => {}})'
		).toBeGreaterThan(0);
		for (const clone of cloneBlocks) {
			expect(
				clone[0].includes('onedit={() => {}}'),
				'expected every measurement-clone SourceChip usage to also carry onedit={() => {}} — a measurement clone must never be able to dispatch a real edit'
			).toBe(true);
		}
	});

	it('every real (non-clone) SourceChip usage passes the shared {onedit} binding', () => {
		const realBlocks = [...strippedHeader.matchAll(/<SourceChip[\s\S]*?\/>/g)].filter(
			(m) => !m[0].includes('onfilter={() => {}}')
		);
		expect(realBlocks.length, 'expected to find real (row + overflow) SourceChip usages').toBeGreaterThan(
			0
		);
		for (const real of realBlocks) {
			expect(
				real[0].includes('{onedit}'),
				'expected every real SourceChip usage to pass the shared {onedit} binding, so a chip in the overflow popover behaves identically to one in the row'
			).toBe(true);
		}
	});
});
