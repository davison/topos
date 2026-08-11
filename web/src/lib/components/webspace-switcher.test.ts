// 07-03-PLAN.md Task 2's structural guard over WebspaceSwitcher.svelte:
// the trigger renders through the child({ props }) snippet composition API
// (matching every other trigger in this app), the current webspace item
// carries aria-current, exactly one font-semibold weight cue appears in
// the per-webspace item template (no font-medium anywhere in the file —
// 07-UI-SPEC.md Typography's strict two-weight budget), the menu contains
// exactly the three static items 09-UI-SPEC.md Fix 7 permits (New
// webspace, Reload config, Manage sources…) and no fourth action item —
// widened from D-13's original two-item rule (09-06-PLAN.md Task 1) — and
// the trigger keeps its truncate + title treatment (the same one the
// retired <h1> carried).
//
// House pattern (matches filter-chip.test.ts / overlay-primitives.test.ts):
// comment-stripped source scanning, a found-non-empty-source guard first,
// and one consequence-describing message per assertion.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const switcherPath = join(here, 'WebspaceSwitcher.svelte');

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

const rawSwitcher = readFileSync(switcherPath, 'utf-8');
const stripped = stripComments(rawSwitcher);

describe('webspace-switcher guard: found non-empty comment-stripped source', () => {
	it('WebspaceSwitcher.svelte', () => {
		expect(stripped.length).toBeGreaterThan(0);
	});
});

describe('trigger composition: renders through a child({ props }) snippet', () => {
	it('DropdownMenuTrigger wraps a #snippet child({ props }) block', () => {
		const triggerBlock = extractBetween(stripped, '<DropdownMenuTrigger>', '</DropdownMenuTrigger>');
		expect(
			/\{#snippet\s+child\(\s*\{\s*props\s*\}\s*\)\}/.test(triggerBlock),
			'expected DropdownMenuTrigger to compose via {#snippet child({ props })}, matching the house pattern every other trigger in this app uses'
		).toBe(true);
	});
});

describe('trigger: keeps the truncate + title treatment the retired <h1> carried', () => {
	it('the trigger element declares both truncate and a title binding', () => {
		const triggerBlock = extractBetween(stripped, '<DropdownMenuTrigger>', '</DropdownMenuTrigger>');
		expect(
			/\btruncate\b/.test(triggerBlock),
			'expected the switcher trigger to truncate a long webspace name to one line'
		).toBe(true);
		expect(
			/title=\{webspace\}/.test(triggerBlock),
			'expected the switcher trigger to carry title={webspace} so the full name is reachable on hover, matching the retired <h1>'
		).toBe(true);
	});
});

describe('current-webspace emphasis: aria-current plus exactly one weight cue', () => {
	const itemBlock = extractBetween(stripped, '{#each webspaces', '{/each}');

	it('the per-webspace item carries aria-current', () => {
		expect(
			/aria-current=/.test(itemBlock),
			'expected the per-webspace item to carry an aria-current binding marking the active webspace'
		).toBe(true);
	});

	it('exactly one font-semibold cue appears in the per-webspace item template', () => {
		const matches = itemBlock.match(/font-semibold/g) ?? [];
		expect(
			matches.length,
			'expected exactly one font-semibold occurrence inside the {#each webspaces} item block — the active webspace is the only emphasis point, no separate icon'
		).toBe(1);
	});

	it('font-medium never appears anywhere in the file (07-UI-SPEC.md two-weight budget)', () => {
		expect(
			/font-medium/.test(stripped),
			'found font-medium in WebspaceSwitcher.svelte — every weight decision in this phase uses only 400 or 600, never 500'
		).toBe(false);
	});
});

describe('menu contents: exactly the three 09-UI-SPEC.md Fix 7 static items, no fourth action item', () => {
	// 09-06-PLAN.md Task 1 supersedes 07-UI-SPEC.md's D-13 ("exactly two
	// static DropdownMenuItem entries outside the per-webspace loop") per
	// 09-UI-SPEC.md Fix 7's explicit "widened to three" contract note. This
	// is a deliberate contract change, updated in place here per Fix 7's
	// own instruction, not a regression to route around.

	it('contains "New webspace" (single "+" — the leading Plus icon alone, no doubled text)', () => {
		expect(stripped.includes('New webspace')).toBe(true);
		expect(stripped.includes('+ New webspace')).toBe(false);
	});

	it('contains "Reload config"', () => {
		expect(stripped.includes('Reload config')).toBe(true);
	});

	it('contains "Manage sources…"', () => {
		expect(stripped.includes('Manage sources…')).toBe(true);
	});

	it('renders exactly three static-copy DropdownMenuItem entries beyond the per-webspace loop', () => {
		// Every DropdownMenuItem usage in the file: one dynamic (inside the
		// {#each webspaces} block, asserted separately above) plus exactly
		// three static action items (create, reload, manage) — never a
		// fourth. Widened from two to three by 09-UI-SPEC.md Fix 7.
		const allItemOpenings = stripped.match(/<DropdownMenuItem\b/g) ?? [];
		const eachBlock = extractBetween(stripped, '{#each webspaces', '{/each}');
		const dynamicItemOpenings = eachBlock.match(/<DropdownMenuItem\b/g) ?? [];
		const staticItemCount = allItemOpenings.length - dynamicItemOpenings.length;
		expect(
			staticItemCount,
			'expected exactly three static DropdownMenuItem entries outside the per-webspace loop (New webspace, Reload config, Manage sources…) — a fourth would violate 09-UI-SPEC.md Fix 7\'s widened "no other global settings surface" rule'
		).toBe(3);
	});

	it('Reload config sits between New webspace and the separator before Manage sources…', () => {
		const createIndex = stripped.indexOf('New webspace');
		const reloadIndex = stripped.indexOf('Reload config');
		const manageIndex = stripped.indexOf('Manage sources…');
		expect(
			createIndex,
			'expected to find New webspace'
		).toBeGreaterThanOrEqual(0);
		expect(
			reloadIndex,
			'expected Reload config to appear after New webspace in source order'
		).toBeGreaterThan(createIndex);
		expect(
			manageIndex,
			'expected Manage sources… to appear after Reload config in source order'
		).toBeGreaterThan(reloadIndex);
	});

	it('Reload config carries a leading RotateCw icon', () => {
		expect(stripped.includes("from '@lucide/svelte/icons/rotate-cw'")).toBe(true);
		const reloadItemBlock = extractBetween(stripped, 'onSelect={onreload}', '</DropdownMenuItem>');
		expect(
			/<RotateCw\b/.test(reloadItemBlock),
			'expected the Reload config item to render a leading RotateCw icon'
		).toBe(true);
	});

	it('Reload config disables while a reload is in flight', () => {
		expect(stripped.includes('disabled={reloadBusy}')).toBe(true);
	});
});
