// 07-04-PLAN.md Task 3's structural guard over SourceChip.svelte's edit-menu
// control (D-12): the trigger's aria-label and size-8 sizing, the
// stopPropagation-before-callback click handler (the D-12 vs Phase 6 D-01
// collision — a click must never also toggle the chip's filter) with the
// filter button's own handler left untouched, the menu's exact contents
// (no instance-deletion item — D-12/D-13's "exactly one place in the app
// deletes an instance" rule), the remove item's destructive tint, and
// WebspaceHeader.svelte's measurement clones still wiring a no-op onedit.
//
// 08-04-PLAN.md Task 2 (D-03) extends this file: the menu's "Re-link…"
// entry is guarded on the WhatsApp source_type — this is a static
// source-scan guard (the DropdownMenuItem markup always exists in the
// component's source regardless of the runtime {#if} branch it's wrapped
// in), so the guard assertion below checks the {#if isWhatsApp} wrapper
// itself, not a rendered absence.
//
// 09-05-PLAN.md Task 2 (09-UI-SPEC.md Fix 5) updates three assertions that
// encoded the pre-fix state, in place, per the 07-11 precedent for updating
// a superseded rule rather than treating the update as a regression: the
// trigger's aria-label broadens from the edit-only "Edit {name}" phrasing to
// "{name} actions" (it is no longer edit-only); the separator count becomes
// two (Refresh now gained its own leading separator); and the menu-item
// label set gains "Refresh now" as its first member, since refresh folded
// into this menu from a standalone button. New behaviour this same task
// adds — the Refresh now item's syncing-disabled guard and spin class — has
// no prior assertion to update and is added fresh below.
//
// 11-06-PLAN.md Task 1 (11-UI-SPEC.md E4/E5) supersedes three more of these
// counts, in place, per the same precedent: a conditional "Trust updated
// binary…" item now precedes Refresh now in source order (menu-item label
// set widens to six, six-first-is-trust-update), a conditional pinned-hash
// footer (E5) adds a third static DropdownMenuSeparator and a second
// aria-label to the file. These are static SOURCE scans — every count below
// reflects markup as it exists in the file regardless of the {#if} runtime
// guards wrapping the new regions, exactly like the pre-existing Re-link…
// guard test already does.
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

// The overflow trigger's own Button element, inside the DropdownMenuTrigger's
// child snippet — located via its unique aria-label expression, then
// walked backward to that same element's own `<Button` opening. Since Fix 5
// removed the standalone refresh Button, this trigger is now the chip's
// ONLY Button element.
const ariaLabelMarker = 'aria-label={`${source.display_name} actions`}';
const ariaLabelIndex = strippedChip.indexOf(ariaLabelMarker);
expect(
	ariaLabelIndex,
	`expected to find "${ariaLabelMarker}" in the scanned source`
).toBeGreaterThanOrEqual(0);
const editButtonStart = strippedChip.lastIndexOf('<Button', ariaLabelIndex);
const editButtonEnd = strippedChip.indexOf('</Button>', ariaLabelIndex);
const editButtonBlock = strippedChip.slice(editButtonStart, editButtonEnd + '</Button>'.length);
const editMenuBlock = extractBetween(
	strippedChip,
	'<DropdownMenuContent>',
	'</DropdownMenuContent>'
);
const filterButtonBlock = extractBetween(strippedChip, '<button', '</button>');
const editClickHandlerBody = extractBetween(
	strippedChip,
	'function handleEditClick(event: MouseEvent, triggerOnClick?: (e: MouseEvent) => void) {',
	'\n\t}'
);

describe('overflow trigger: aria-label reads "{name} actions" (Fix 5, no longer edit-only), size-8 sizing', () => {
	it('the aria-label reads "{display_name} actions"', () => {
		expect(strippedChip.includes('aria-label={`${source.display_name} actions`}')).toBe(true);
	});

	it('the trigger button carries size-8 sizing', () => {
		expect(
			/\bsize-8\b/.test(editButtonBlock),
			"expected the overflow trigger to carry size-8 sizing — the chip's only sub-44px control now that Fix 5 removed the standalone refresh button"
		).toBe(true);
	});

	it("exactly two aria-label attributes exist in the file — the overflow trigger and the E5 pinned-hash footer's copy button (11-06-PLAN.md Task 1)", () => {
		const matches = strippedChip.match(/aria-label=/g) ?? [];
		expect(matches.length).toBe(2);
	});
});

describe('click handling: stopPropagation before the callback, filter button untouched', () => {
	it('handleEditClick calls stopPropagation before invoking the trigger callback', () => {
		const stopIndex = editClickHandlerBody.indexOf('event.stopPropagation()');
		const callbackIndex = editClickHandlerBody.indexOf('triggerOnClick?.(event)');
		expect(
			stopIndex,
			'expected handleEditClick to call event.stopPropagation()'
		).toBeGreaterThanOrEqual(0);
		expect(
			callbackIndex,
			'expected handleEditClick to invoke the forwarded trigger callback'
		).toBeGreaterThanOrEqual(0);
		expect(
			stopIndex < callbackIndex,
			"expected stopPropagation() to run BEFORE the forwarded callback — this is the D-12 vs Phase 6 D-01 collision, and getting the order wrong would let an edit-menu open also toggle the chip's filter"
		).toBe(true);
	});

	it("the filter button's own click handler is untouched (still just onfilter(source.name))", () => {
		expect(
			filterButtonBlock.includes('onclick={() => onfilter(source.name)}'),
			"expected the filter button's click handler to remain exactly onclick={() => onfilter(source.name)} — a plain chip click must still only toggle the filter"
		).toBe(true);
	});
});

describe('menu contents: Refresh now first, three separators, no delete-instance item', () => {
	it("contains exactly four DropdownMenuSeparators (Fix 5's two, the E5 pinned-hash footer's own — 11-06-PLAN.md Task 1 — and the operator-trusted key footer's own, M2-R4)", () => {
		const separators = editMenuBlock.match(/<DropdownMenuSeparator/g) ?? [];
		expect(separators.length).toBe(4);
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

	it('the menu item label set equals exactly the nine expected labels, Trust updated binary… first, the two key consents next (11-06-PLAN.md Task 1, E4; M2-R4)', () => {
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
			'expected the menu item label set to equal exactly [Trust updated binary…, Trust signing key…, Stop trusting key…, Refresh now, Edit connection…, Edit match settings…, Re-link…, Remove from this webspace], the trust remedies first — proven by an ordered array, not merely membership'
		).toEqual([
			'Trust updated binary…',
			'Trust signing key…',
			'Stop trusting key…',
			'Refresh now',
			'Edit connection…',
			'Edit match settings…',
			'Filter this source…',
			'Re-link…',
			'Remove from this webspace'
		]);
	});
});

// New behaviour Fix 5 adds — the removed standalone refresh button never had
// a syncing guard; this menu item does.
//
// 11-06-PLAN.md Task 1 inserts a new, conditional "Trust updated binary…"
// DropdownMenuItem AHEAD of Refresh now in source order — refreshItemBlock
// below is therefore located by the "Refresh now" text itself and walked
// backward to ITS OWN opening tag, rather than assuming Refresh now is the
// menu's literal first <DropdownMenuItem> (that assumption is now false;
// see the dedicated ordering test below).
describe('Refresh now: disabled and spinning while syncing (Fix 5, new guard)', () => {
	const refreshTextIdx = editMenuBlock.indexOf('Refresh now');
	const refreshItemStart = editMenuBlock.lastIndexOf('<DropdownMenuItem', refreshTextIdx);
	const refreshItemEnd = editMenuBlock.indexOf('</DropdownMenuItem>', refreshTextIdx);
	const refreshItemBlock = editMenuBlock.slice(
		refreshItemStart,
		refreshItemEnd + '</DropdownMenuItem>'.length
	);

	it('is the first DropdownMenuItem in the menu OUTSIDE the conditional Trust updated binary… item (11-06-PLAN.md Task 1, E4)', () => {
		const firstItemIndex = editMenuBlock.indexOf('<DropdownMenuItem');
		const refreshTextIndex = editMenuBlock.indexOf('Refresh now');
		expect(firstItemIndex, 'expected to find a DropdownMenuItem').toBeGreaterThanOrEqual(0);
		expect(refreshTextIndex, 'expected to find "Refresh now"').toBeGreaterThanOrEqual(0);
		// The static markup's literal first <DropdownMenuItem> is now the
		// conditional Trust updated binary… item — present in source order
		// regardless of the {#if isPinMismatch} runtime guard around it
		// (E4: "the item appears first in the menu when the mismatch
		// signal is set"). Refresh now is the first item OUTSIDE that
		// guard, i.e. the second <DropdownMenuItem> in source order.
		const firstItemBlock = editMenuBlock.slice(
			firstItemIndex,
			editMenuBlock.indexOf('</DropdownMenuItem>', firstItemIndex)
		);
		expect(
			firstItemBlock.includes('Trust updated binary'),
			"expected the menu's literal first DropdownMenuItem in source order to be the conditional Trust updated binary… item"
		).toBe(true);
		expect(
			refreshTextIndex,
			'expected Refresh now to appear after the Trust updated binary… item'
		).toBeGreaterThan(firstItemIndex);
	});

	it('is disabled while source.syncing OR isPinMismatch (11-06-PLAN.md Task 1 widens this guard — E4: there is nothing running to refresh)', () => {
		expect(
			refreshItemBlock.includes('disabled={source.syncing || isPinMismatch}'),
			'expected the Refresh now item to be disabled={source.syncing || isPinMismatch}'
		).toBe(true);
	});

	it('calls the existing onrefresh(source.name) — no prop-shape change', () => {
		expect(
			refreshItemBlock.includes('onSelect={() => onrefresh(source.name)}'),
			"expected Refresh now to call the component's existing onrefresh(source.name), the same call the removed standalone button made"
		).toBe(true);
		const matches = strippedChip.match(/onrefresh\(source\.name\)/g) ?? [];
		expect(
			matches.length,
			'expected exactly one onrefresh(source.name) call site — the refresh call survived the move, unduplicated'
		).toBe(1);
	});

	it('its RefreshCw icon carries animate-spin while source.syncing', () => {
		expect(
			/<RefreshCw[^>]*animate-spin[^>]*>/.test(refreshItemBlock) ||
				/<RefreshCw[\s\S]*?source\.syncing[\s\S]*?animate-spin/.test(refreshItemBlock),
			"expected the Refresh now item's RefreshCw icon to carry an animate-spin class while source.syncing"
		).toBe(true);
		const matches = strippedChip.match(/animate-spin/g) ?? [];
		expect(matches.length, 'expected exactly one animate-spin reference in the file').toBe(1);
	});
});

// 08-04-PLAN.md Task 2 (D-03): the Re-link… entry's render is guarded on
// the WhatsApp source type — no other plugin type has anything to
// re-link, and an inert menu entry is worse than an absent one.
describe('Re-link… entry: guarded on the WhatsApp source type', () => {
	it('the Re-link… DropdownMenuItem is wrapped in an {#if isWhatsApp} block', () => {
		const relinkIndex = editMenuBlock.indexOf('Re-link…');
		expect(relinkIndex, 'expected to find a "Re-link…" menu item').toBeGreaterThanOrEqual(0);
		const guardIndex = editMenuBlock.lastIndexOf('{#if isWhatsApp}', relinkIndex);
		const guardCloseIndex = editMenuBlock.indexOf('{/if}', relinkIndex);
		expect(
			guardIndex,
			'expected the Re-link… item to be preceded by an {#if isWhatsApp} guard'
		).toBeGreaterThanOrEqual(0);
		expect(
			guardCloseIndex,
			'expected the {#if isWhatsApp} guard around Re-link… to close with {/if}'
		).toBeGreaterThan(relinkIndex);
	});

	it('isWhatsApp is derived from source.source_type, not the plugin binary name', () => {
		expect(
			strippedChip.includes('source.source_type === WHATSAPP_SOURCE_TYPE'),
			'expected isWhatsApp to key off source.source_type — the field GET /api/sources actually exposes, not a plugin binary name this component has no other reason to know'
		).toBe(true);
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
			editMenuBlock.lastIndexOf(
				'<DropdownMenuItem',
				editMenuBlock.indexOf('Remove from this webspace')
			),
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
		expect(
			realBlocks.length,
			'expected to find real (row + overflow) SourceChip usages'
		).toBeGreaterThan(0);
		for (const real of realBlocks) {
			expect(
				real[0].includes('{onedit}'),
				'expected every real SourceChip usage to pass the shared {onedit} binding, so a chip in the overflow popover behaves identically to one in the row'
			).toBe(true);
		}
	});
});
