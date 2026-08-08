// 07-03-PLAN.md Task 1's structural guard over the three new bits-ui
// wrappers (dialog, dropdown-menu, alert-dialog): each barrel exports
// exactly the named component set the rest of the phase composes from,
// both triggers keep the exact pass-through shape that lets a caller's own
// `child({ props })` snippet (WebspaceHeader.svelte lines 199-231,
// SourceChip.svelte lines 80-111) reach the underlying bits-ui primitive
// unmolested, no file in the three directories resolves a colour through a
// raw hex literal instead of an app.css token, and web/package.json's own
// dependency set is unchanged — these three primitives come from the
// already-installed bits-ui, not a new npm dependency (T-07-15/T-07-SC).
//
// House pattern (matches filter-chip.test.ts / source-chip-pill.test.ts):
// comment-stripped source scanning (web/vite.config.ts's test block runs
// environment: 'node' with no component-mount harness), a found-non-empty-
// source guard first, and one consequence-describing message per
// assertion.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const dialogDir = join(here, 'dialog');
const dropdownDir = join(here, 'dropdown-menu');
const alertDialogDir = join(here, 'alert-dialog');
const packageJsonPath = join(here, '..', '..', '..', '..', 'package.json');

// Strips HTML comments, CSS/JS block comments and JS line comments, each
// replaced with a single space (never deleted outright) so two tokens
// separated only by a comment can never fuse into one identifier.
function stripComments(source: string): string {
	return source
		.replace(/<!--[\s\S]*?-->/g, ' ')
		.replace(/\/\*[\s\S]*?\*\//g, ' ')
		.replace(/\/\/.*$/gm, ' ');
}

function readStripped(path: string): string {
	return stripComments(readFileSync(path, 'utf-8'));
}

const dialogIndex = readStripped(join(dialogDir, 'index.ts'));
const dropdownIndex = readStripped(join(dropdownDir, 'index.ts'));
const alertDialogIndex = readStripped(join(alertDialogDir, 'index.ts'));

const dialogTrigger = readStripped(join(dialogDir, 'dialog-trigger.svelte'));
const dropdownTrigger = readStripped(join(dropdownDir, 'dropdown-menu-trigger.svelte'));
const dropdownContent = readStripped(join(dropdownDir, 'dropdown-menu-content.svelte'));

describe('overlay-primitives guard: found non-empty comment-stripped sources', () => {
	for (const [label, source] of [
		['dialog/index.ts', dialogIndex],
		['dropdown-menu/index.ts', dropdownIndex],
		['alert-dialog/index.ts', alertDialogIndex],
		['dialog/dialog-trigger.svelte', dialogTrigger],
		['dropdown-menu/dropdown-menu-trigger.svelte', dropdownTrigger]
	] as const) {
		it(label, () => {
			expect(source.length).toBeGreaterThan(0);
		});
	}
});

describe('barrel exports: each directory exports exactly the named component set', () => {
	it('dialog/index.ts exports the dialog set', () => {
		for (const name of [
			'Dialog',
			'DialogTrigger',
			'DialogContent',
			'DialogHeader',
			'DialogTitle',
			'DialogFooter'
		]) {
			expect(
				new RegExp(`\\b${name}\\b`).test(dialogIndex),
				`expected dialog/index.ts to export ${name}`
			).toBe(true);
		}
	});

	it('dropdown-menu/index.ts exports the dropdown-menu set', () => {
		for (const name of [
			'DropdownMenu',
			'DropdownMenuTrigger',
			'DropdownMenuContent',
			'DropdownMenuItem',
			'DropdownMenuSeparator'
		]) {
			expect(
				new RegExp(`\\b${name}\\b`).test(dropdownIndex),
				`expected dropdown-menu/index.ts to export ${name}`
			).toBe(true);
		}
	});

	it('alert-dialog/index.ts exports the alert-dialog set', () => {
		for (const name of [
			'AlertDialog',
			'AlertDialogContent',
			'AlertDialogTitle',
			'AlertDialogDescription',
			'AlertDialogAction',
			'AlertDialogCancel'
		]) {
			expect(
				new RegExp(`\\b${name}\\b`).test(alertDialogIndex),
				`expected alert-dialog/index.ts to export ${name}`
			).toBe(true);
		}
	});
});

// The `child({ props })` snippet composition API is delivered by bits-ui's
// own Trigger primitives internally (each one destructures its own `child`
// snippet and renders it in place of its default <button> when present,
// confirmed by reading node_modules/bits-ui's dialog-trigger.svelte and
// menu-trigger.svelte source directly) — the wrapper's only job is to stay
// out of the way: destructure nothing but `ref`, and spread every other
// prop (including a caller-supplied `child` snippet) straight onto the
// primitive, exactly as popover-trigger.svelte (this repo's own established
// house pattern) already does. A wrapper that destructured `child`/
// `children` itself, or narrowed the props type away from the primitive's
// own `TriggerProps`, would silently swallow that composition capability.
function assertPassthroughTriggerShape(source: string, label: string): void {
	expect(
		/ref\s*=\s*\$bindable\(null\)/.test(source),
		`expected ${label} to destructure a bindable ref, matching popover-trigger.svelte's shape`
	).toBe(true);
	expect(
		/\.\.\.restProps/.test(source),
		`expected ${label} to spread ...restProps onto the underlying primitive, so a caller-supplied child snippet passes through unmolested`
	).toBe(true);
	expect(
		/\.Trigger[\s\S]*?\{\.\.\.restProps\}/.test(source),
		`expected ${label} to spread ...restProps directly on the rendered primitive Trigger element`
	).toBe(true);
	// The destructuring statement itself (`let { ... }: X.TriggerProps =
	// $props();`) must never separately name `child` or `children` — doing
	// so would consume the snippet before it reaches the primitive.
	const destructure = source.match(/let\s*\{[\s\S]*?\}\s*:\s*\S+\.TriggerProps/)?.[0] ?? '';
	expect(
		destructure.length,
		`expected ${label} to destructure its props from a TriggerProps type (matching the primitive's own composition-preserving type), not a narrower custom interface`
	).toBeGreaterThan(0);
	expect(
		/\bchild\b/.test(destructure),
		`expected ${label} to NOT destructure "child" itself — that would swallow the child({ props }) snippet before it reaches the primitive`
	).toBe(false);
}

describe('child({ props }) trigger composition: both new triggers match popover-trigger.svelte', () => {
	it('DialogTrigger keeps the pass-through shape', () => {
		assertPassthroughTriggerShape(dialogTrigger, 'dialog-trigger.svelte');
	});
	it('DropdownMenuTrigger keeps the pass-through shape', () => {
		assertPassthroughTriggerShape(dropdownTrigger, 'dropdown-menu-trigger.svelte');
	});
});

describe('dropdown-menu overflow backstop: content is height-capped and scrolls', () => {
	it('DropdownMenuContent carries a max-height plus overflow-y class pair', () => {
		expect(
			/\bmax-h-\S+/.test(dropdownContent),
			'expected dropdown-menu-content.svelte to declare a max-h-* class, so a long menu (many webspaces, many unconfigured plugin types) cannot grow past the viewport'
		).toBe(true);
		expect(
			/\boverflow-y-auto\b/.test(dropdownContent),
			'expected dropdown-menu-content.svelte to declare overflow-y-auto alongside its max-height, so the capped content scrolls rather than clipping'
		).toBe(true);
	});
});

describe('registry safety: no raw hex colour anywhere in the three new directories', () => {
	const files = [
		['dialog/dialog.svelte', join(dialogDir, 'dialog.svelte')],
		['dialog/dialog-trigger.svelte', join(dialogDir, 'dialog-trigger.svelte')],
		['dialog/dialog-content.svelte', join(dialogDir, 'dialog-content.svelte')],
		['dialog/dialog-header.svelte', join(dialogDir, 'dialog-header.svelte')],
		['dialog/dialog-title.svelte', join(dialogDir, 'dialog-title.svelte')],
		['dialog/dialog-footer.svelte', join(dialogDir, 'dialog-footer.svelte')],
		['dialog/index.ts', join(dialogDir, 'index.ts')],
		['dropdown-menu/dropdown-menu.svelte', join(dropdownDir, 'dropdown-menu.svelte')],
		['dropdown-menu/dropdown-menu-trigger.svelte', join(dropdownDir, 'dropdown-menu-trigger.svelte')],
		['dropdown-menu/dropdown-menu-content.svelte', join(dropdownDir, 'dropdown-menu-content.svelte')],
		['dropdown-menu/dropdown-menu-item.svelte', join(dropdownDir, 'dropdown-menu-item.svelte')],
		[
			'dropdown-menu/dropdown-menu-separator.svelte',
			join(dropdownDir, 'dropdown-menu-separator.svelte')
		],
		['dropdown-menu/index.ts', join(dropdownDir, 'index.ts')],
		['alert-dialog/alert-dialog.svelte', join(alertDialogDir, 'alert-dialog.svelte')],
		[
			'alert-dialog/alert-dialog-content.svelte',
			join(alertDialogDir, 'alert-dialog-content.svelte')
		],
		['alert-dialog/alert-dialog-title.svelte', join(alertDialogDir, 'alert-dialog-title.svelte')],
		[
			'alert-dialog/alert-dialog-description.svelte',
			join(alertDialogDir, 'alert-dialog-description.svelte')
		],
		[
			'alert-dialog/alert-dialog-action.svelte',
			join(alertDialogDir, 'alert-dialog-action.svelte')
		],
		[
			'alert-dialog/alert-dialog-cancel.svelte',
			join(alertDialogDir, 'alert-dialog-cancel.svelte')
		],
		['alert-dialog/index.ts', join(alertDialogDir, 'index.ts')]
	] as const;

	for (const [label, path] of files) {
		it(`${label} resolves every colour through an app.css token`, () => {
			const source = readStripped(path);
			expect(
				/#[0-9a-fA-F]{3,8}\b/.test(source),
				`found a raw hex colour literal in ${label} — every colour in this phase must resolve through an existing app.css token (bg-card, border-border, text-muted-foreground, etc.), never a hardcoded hex value`
			).toBe(false);
		});
	}
});

describe('T-07-SC: web/package.json gained no new dependency for these three primitives', () => {
	it('dependency and devDependency key sets are unchanged from pre-task content', () => {
		const pkg = JSON.parse(readFileSync(packageJsonPath, 'utf-8')) as {
			dependencies: Record<string, string>;
			devDependencies: Record<string, string>;
		};

		const expectedDependencies = [
			'@fontsource-variable/inter',
			'@lucide/svelte',
			'bits-ui',
			'clsx',
			'tailwind-merge',
			'tailwind-variants'
		].sort();
		const expectedDevDependencies = [
			'@internationalized/date',
			'@sveltejs/adapter-static',
			'@sveltejs/kit',
			'@sveltejs/vite-plugin-svelte',
			'@tailwindcss/vite',
			'svelte',
			'svelte-check',
			'tailwindcss',
			'typescript',
			'vite',
			'vitest'
		].sort();

		expect(
			Object.keys(pkg.dependencies).sort(),
			'expected web/package.json "dependencies" to be byte-for-byte the pre-task set — the three overlay primitives come from the already-installed bits-ui, not a new package'
		).toEqual(expectedDependencies);
		expect(
			Object.keys(pkg.devDependencies).sort(),
			'expected web/package.json "devDependencies" to be byte-for-byte the pre-task set'
		).toEqual(expectedDevDependencies);
	});
});
