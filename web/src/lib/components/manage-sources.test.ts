// 07-05-PLAN.md Task 1's structural guard over ManageSourcesModal.svelte,
// updated by 09-06-PLAN.md Task 2 (09-UI-SPEC.md Fix 7 relocates
// Reload config out of this modal entirely): both section headings, the
// truncate+title precedent on instance rows, both AlertDialogs' exact
// UI-SPEC copy with no type-to-confirm text input, both lists'
// height-cap+scroll pair, the single-entry-point rule for instance
// deletion (D-12/D-13, the destructive delete action lives only here,
// never also in SourceChip's own edit menu), and — new this plan — the
// modal's control-label SET asserted by equality against exactly what its
// instance/webspace rows still offer, so a lingering reload control fails
// as a set mismatch rather than needing a grep for an absent word.
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

describe('instance rows: leading PluginIcon at size-4 (09-06-PLAN.md Task 3, 09-UI-SPEC.md Fix 10)', () => {
	const rowBlock = extractBetween(stripped, '{#each instanceIds as id (id)}', '{/each}');

	it('ManageSourcesModal imports PluginIcon', () => {
		expect(stripped.includes("from '$lib/components/PluginIcon.svelte'")).toBe(true);
	});

	it('the instance row renders a PluginIcon keyed on source.plugin', () => {
		expect(
			/<PluginIcon\s+plugin=\{source\.plugin\}/.test(rowBlock),
			'expected the instance row to render <PluginIcon plugin={source.plugin} .../>'
		).toBe(true);
	});

	it('the icon renders at size-4 (16px), matching the Pencil/Trash2 sizing already in this row', () => {
		expect(
			/<PluginIcon\s+plugin=\{source\.plugin\}\s+size="size-4/.test(rowBlock),
			'expected the PluginIcon to carry a size-4 class, matching the row\'s existing Pencil/Trash2 sizing'
		).toBe(true);
	});

	it('the icon carries shrink-0 so it survives a long truncating display name', () => {
		expect(
			/<PluginIcon\s+plugin=\{source\.plugin\}\s+size="size-4\s+shrink-0"/.test(rowBlock),
			'expected the PluginIcon\'s size class to include shrink-0'
		).toBe(true);
	});

	it('the row\'s existing two-line display-name/plugin-name layout is otherwise unchanged', () => {
		expect(rowBlock.includes('{displayNameFor(id)}')).toBe(true);
		expect(rowBlock.includes('{source.plugin}')).toBe(true);
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

describe('reload control relocation (09-06-PLAN.md Task 2, 09-UI-SPEC.md Fix 7): no second entry point', () => {
	// The modal's own rendered control-label set is asserted by SET
	// EQUALITY against exactly what its instance/webspace rows still
	// offer — a lingering reload control fails as a set mismatch rather
	// than needing a grep for an absent word (the acceptance criterion
	// this describe block exists to satisfy).
	function extractButtonLabels(source: string): Set<string> {
		// Attribute values in this file's <Button> opening tags include
		// arrow-function onclick handlers (e.g. `onclick={() => (id = x)}`)
		// whose `=>` contains a literal '>' — a naive `[^>]*` scan for the
		// opening tag's end would stop there instead of at the tag's real
		// closing '>'. Neutralising every arrow first (a value never
		// otherwise present in this file's markup) removes that ambiguity
		// before the tag-matching regex runs.
		const neutralised = source.replace(/=>/g, '~~');
		const labels = new Set<string>();
		const buttonRegex = /<Button\b[^>]*>([\s\S]*?)<\/Button>/g;
		let match: RegExpExecArray | null;
		while ((match = buttonRegex.exec(neutralised)) !== null) {
			const inner = match[1]
				.replace(/<[^>]+>/g, ' ')
				.replace(/\s+/g, ' ')
				.trim();
			if (inner) labels.add(inner);
		}
		return labels;
	}

	it("the Dialog region's own control-label set equals exactly {Delete, Edit, Rename} — the instance/webspace rows' own labels, nothing else", () => {
		const dialogBlock = extractBetween(
			stripped,
			'<Dialog {open} onOpenChange={handleOpenChange}>',
			'</Dialog>'
		);
		const labels = [...extractButtonLabels(dialogBlock)].sort();
		expect(
			labels,
			'expected the modal\'s Dialog-region control-label set to equal exactly {Delete, Edit} — a lingering Reload config label would surface here as an unexpected third member'
		).toEqual(['Delete', 'Edit', 'Rename']);
	});

	it('the file no longer imports the config-reload API client or the RotateCw icon', () => {
		expect(stripped.includes('reloadConfig')).toBe(false);
		expect(stripped.includes("from '@lucide/svelte/icons/rotate-cw'")).toBe(false);
		expect(stripped.includes('RotateCw')).toBe(false);
	});

	it('never renders RefreshCw (the per-source refresh icon stays SourceChip\'s own, unaffected by this relocation)', () => {
		expect(stripped.includes('RefreshCw')).toBe(false);
	});

	it('SourceChip.svelte still uses RefreshCw, unaffected by this relocation', () => {
		expect(strippedChip.includes("from '@lucide/svelte/icons/refresh-cw'")).toBe(true);
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
