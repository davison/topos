// 07-05-PLAN.md Task 1's structural guard over ManageSourcesModal.svelte:
// both section headings, the truncate+title precedent on instance rows,
// both AlertDialogs' exact UI-SPEC copy with no type-to-confirm text
// input, the RotateCw/RefreshCw icon distinction against SourceChip.svelte,
// both lists' height-cap+scroll pair, and the single-entry-point rule for
// instance deletion (D-12/D-13): the destructive delete action lives only
// here, never also in SourceChip's own edit menu.
//
// House pattern (matches chip-edit-menu.test.ts / add-source.test.ts):
// comment-stripped source scanning, `extractBetween` scoping, a
// found-non-empty-source guard first, and one consequence-describing
// message per assertion.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const modalPath = join(here, 'ManageSourcesModal.svelte');
const chipPath = join(here, 'SourceChip.svelte');

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

const rawModal = readFileSync(modalPath, 'utf-8');
const rawChip = readFileSync(chipPath, 'utf-8');
const stripped = stripComments(rawModal);
const strippedChip = stripComments(rawChip);

describe('manage-sources guard: found non-empty comment-stripped sources', () => {
	it('ManageSourcesModal.svelte', () => {
		expect(stripped.length).toBeGreaterThan(0);
	});
	it('SourceChip.svelte', () => {
		expect(strippedChip.length).toBeGreaterThan(0);
	});
});

describe('section headings: both "Source instances" and "Webspaces"', () => {
	it('renders "Source instances"', () => {
		expect(stripped.includes('Source instances')).toBe(true);
	});

	it('renders "Webspaces"', () => {
		expect(stripped.includes('>Webspaces<')).toBe(true);
	});
});

describe('instance rows: truncate plus a title binding (SourceChip precedent)', () => {
	it('the instance-row name element carries both truncate and a title binding', () => {
		const rowBlock = extractBetween(stripped, '{#each instanceIds as id (id)}', '{/each}');
		expect(
			/\btruncate\b/.test(rowBlock),
			'expected the instance-row display name to truncate a long value to one line'
		).toBe(true);
		expect(
			/title=\{displayNameFor\(id\)\}/.test(rowBlock),
			'expected the instance-row display name to carry title={displayNameFor(id)}, matching SourceChip.svelte\'s own truncate+title precedent'
		).toBe(true);
	});
});

describe('lists: both carry a max-height plus overflow class pair', () => {
	it('the Source instances list carries max-h-* and overflow-y-auto', () => {
		const instanceListBlock = extractBetween(
			stripped,
			'<h3 class="text-[14px] leading-[1.4] font-semibold text-foreground">Source instances</h3>',
			'</div>\n\t\t</div>'
		);
		expect(/max-h-\d+/.test(instanceListBlock)).toBe(true);
		expect(/overflow-y-auto/.test(instanceListBlock)).toBe(true);
	});

	it('the Webspaces list carries max-h-* and overflow-y-auto', () => {
		const webspaceListBlock = extractBetween(
			stripped,
			'>Webspaces<',
			'</div>\n\t\t</div>'
		);
		expect(/max-h-\d+/.test(webspaceListBlock)).toBe(true);
		expect(/overflow-y-auto/.test(webspaceListBlock)).toBe(true);
	});
});

describe('instance deletion AlertDialog: exact UI-SPEC copy, no type-to-confirm input', () => {
	const instanceDialogBlock = extractBetween(
		stripped,
		'open={instanceDeleteTarget !== null}',
		'</AlertDialog>'
	);

	it('the title reads exactly "Delete {display_name}?"', () => {
		expect(
			instanceDialogBlock.includes('Delete {instanceDeleteTarget ? displayNameFor(instanceDeleteTarget) : \'\'}?')
		).toBe(true);
	});

	it('the body names both the cross-webspace removal and the indexed-item deletion, verbatim', () => {
		expect(instanceDialogBlock.includes('from every webspace and deletes its indexed items')).toBe(
			true
		);
		expect(instanceDialogBlock.includes("This can't be undone.")).toBe(true);
	});

	it('contains no text input (no type-to-confirm field)', () => {
		expect(/<input\b/i.test(instanceDialogBlock)).toBe(false);
	});
});

describe('webspace deletion AlertDialog: exact UI-SPEC copy, no type-to-confirm input', () => {
	const webspaceDialogBlock = extractBetween(
		stripped,
		'open={webspaceDeleteTarget !== null}',
		'</AlertDialog>'
	);

	it('the title reads exactly "Delete {webspace}?"', () => {
		expect(webspaceDialogBlock.includes('Delete {webspaceDeleteTarget}?')).toBe(true);
	});

	it('the body names both the filter removal and the unaffected instances/webspaces, verbatim', () => {
		expect(
			webspaceDialogBlock.includes('This removes the webspace and its filters.')
		).toBe(true);
		expect(
			webspaceDialogBlock.includes('Source instances and other webspaces are')
		).toBe(true);
		expect(webspaceDialogBlock.includes('unaffected.')).toBe(true);
	});

	it('contains no text input (no type-to-confirm field)', () => {
		expect(/<input\b/i.test(webspaceDialogBlock)).toBe(false);
	});
});

describe('reload icon: RotateCw here, distinct from SourceChip\'s own RefreshCw', () => {
	it('ManageSourcesModal imports and renders RotateCw', () => {
		expect(stripped.includes("from '@lucide/svelte/icons/rotate-cw'")).toBe(true);
		expect(/<RotateCw\b/.test(stripped)).toBe(true);
	});

	it('ManageSourcesModal never renders RefreshCw (the per-source refresh icon stays SourceChip\'s own)', () => {
		expect(stripped.includes('RefreshCw')).toBe(false);
	});

	it('SourceChip.svelte uses RefreshCw, not RotateCw — the two "reload" concepts are visually distinct icons across the two files', () => {
		expect(strippedChip.includes("from '@lucide/svelte/icons/refresh-cw'")).toBe(true);
		expect(strippedChip.includes('RotateCw')).toBe(false);
	});
});

describe('single entry point for instance deletion (D-12/D-13): the label set here never appears in SourceChip\'s own edit menu', () => {
	it('this file offers a "Delete" action for a source instance', () => {
		const instanceListBlock = extractBetween(stripped, '{#each instanceIds as id (id)}', '{/each}');
		expect(instanceListBlock.includes('>\n\t\t\t\t\t\t\t\tDelete\n\t\t\t\t\t\t\t</Button>') || instanceListBlock.includes('Delete')).toBe(
			true
		);
	});

	it('SourceChip.svelte\'s own edit-menu label set contains no "Delete" item — full instance deletion lives only in ManageSourcesModal', () => {
		const editMenuBlock = extractBetween(
			strippedChip,
			'<DropdownMenuContent>',
			'</DropdownMenuContent>'
		);
		expect(
			/>\s*Delete\s*</.test(editMenuBlock) || editMenuBlock.includes('Delete instance'),
			'expected SourceChip\'s own edit menu to contain no standalone "Delete" item — two competing entry points for the same irreversible action would violate D-12/D-13'
		).toBe(false);
	});
});
