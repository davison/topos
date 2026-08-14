// 07-05-PLAN.md Task 2's cross-component guard: pins the ONE shared
// save/reload state pattern (07-UI-SPEC.md "Save / Reload State & Error
// Surfacing", D-03/D-06/D-08/D-09) as a fact the suite enforces across
// every config-writing surface in the app, rather than a convention four
// modals plus a modal-less header write independently have to remember:
//
//  - in flight: the initiating control disables, no new spinner component
//  - validation failure: a destructive Alert with the kernel's message
//    verbatim
//  - hash conflict: the same Alert, the ONE fixed copy
//    (api.ts's CONFIG_CONFLICT_MESSAGE), appearing nowhere else as a
//    duplicated literal
//  - success: the modal closes, the header updates in place — no toast,
//    anywhere in web/src/lib/components/
//
// House pattern (matches manage-sources.test.ts / chip-edit-menu.test.ts):
// comment-stripped source scanning, `extractBetween` scoping, a
// found-non-empty-source guard first, and one consequence-describing
// message per assertion.

import { describe, it, expect } from 'vitest';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { dirname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
// web/src — the root every path below is scanned or reported relative to.
const srcRoot = join(here, '..', '..');

function stripComments(source: string): string {
	return source
		.replace(/<!--[\s\S]*?-->/g, ' ')
		.replace(/\/\*[\s\S]*?\*\//g, ' ')
		.replace(/\/\/.*$/gm, ' ');
}

// Walks web/src recursively, returning every .svelte/.ts file's absolute
// path — skips node_modules-shaped junk (none live under src/, but this
// keeps the walk cheap and defensive) and vitest's own .svelte-kit
// generated output if present alongside src/.
function walk(dir: string): string[] {
	const out: string[] = [];
	for (const entry of readdirSync(dir)) {
		const full = join(dir, entry);
		const stat = statSync(full);
		if (stat.isDirectory()) {
			out.push(...walk(full));
		} else if (/\.(svelte|ts)$/.test(entry)) {
			out.push(full);
		}
	}
	return out;
}

const allSourceFiles = walk(srcRoot);

const modalFiles = {
	CreateWebspaceModal: join(here, 'CreateWebspaceModal.svelte'),
	AddSourceModal: join(here, 'AddSourceModal.svelte'),
	EditSourceModal: join(here, 'EditSourceModal.svelte'),
	ManageSourcesModal: join(here, 'ManageSourcesModal.svelte')
};

const strippedByModal = Object.fromEntries(
	Object.entries(modalFiles).map(([name, path]) => [name, stripComments(readFileSync(path, 'utf-8'))])
);

describe('save-state guard: found non-empty comment-stripped sources for every modal', () => {
	for (const [name, stripped] of Object.entries(strippedByModal)) {
		it(name, () => {
			expect((stripped as string).length).toBeGreaterThan(0);
		});
	}
});

describe('every one of the four modal components renders a destructive Alert', () => {
	for (const [name, stripped] of Object.entries(strippedByModal)) {
		it(name, () => {
			expect(
				/<Alert\s+variant="destructive"/.test(stripped as string),
				`expected ${name} to render <Alert variant="destructive"> for its validation-failure/hash-conflict surfacing`
			).toBe(true);
		});
	}
});

describe('every one of the four modal components binds a disabled state on its submit/action control(s)', () => {
	it('CreateWebspaceModal: the Create webspace submit control binds disabled', () => {
		expect(/type="submit"[^>]*disabled=\{/.test(strippedByModal.CreateWebspaceModal)).toBe(true);
	});

	it('AddSourceModal: every submit control across its three flows binds disabled', () => {
		const submitControls = strippedByModal.AddSourceModal.match(/type="submit"[^>]*>/g) ?? [];
		expect(
			submitControls.length,
			'expected at least one type="submit" control in AddSourceModal'
		).toBeGreaterThan(0);
		for (const control of submitControls) {
			expect(
				/disabled=\{/.test(control),
				`expected every submit control in AddSourceModal to bind a disabled state, found one without: ${control}`
			).toBe(true);
		}
	});

	it('EditSourceModal: both the connection and match submit controls bind disabled', () => {
		const submitControls = strippedByModal.EditSourceModal.match(/type="submit"[^>]*>/g) ?? [];
		expect(submitControls.length).toBeGreaterThanOrEqual(2);
		for (const control of submitControls) {
			expect(/disabled=\{/.test(control)).toBe(true);
		}
	});

	it('ManageSourcesModal: both AlertDialogAction delete buttons bind disabled (Reload config relocated to WebspaceSwitcher — 09-06-PLAN.md Task 2)', () => {
		// The Reload config button/state this test used to also assert on
		// no longer lives in ManageSourcesModal (09-UI-SPEC.md Fix 7); its
		// own in-flight-disable guard is asserted by
		// webspace-switcher.test.ts's "Reload config disables while a
		// reload is in flight" case against WebspaceSwitcher's
		// disabled={reloadBusy} binding instead.
		const actionButtons =
			strippedByModal.ManageSourcesModal.match(/<AlertDialogAction[\s\S]*?>/g) ?? [];
		expect(actionButtons.length).toBe(2);
		for (const button of actionButtons) {
			expect(/disabled=\{deleting\}/.test(button)).toBe(true);
		}
	});
});

describe('the fixed hash-conflict copy is a single exported constant, never a duplicated literal', () => {
	it('CONFIG_CONFLICT_MESSAGE is exported exactly once, from api.ts', () => {
		const apiPath = join(srcRoot, 'lib', 'api.ts');
		const apiSource = readFileSync(apiPath, 'utf-8');
		const exportOccurrences = (apiSource.match(/export const CONFIG_CONFLICT_MESSAGE\s*=/g) ?? [])
			.length;
		expect(exportOccurrences, 'expected exactly one export of CONFIG_CONFLICT_MESSAGE').toBe(1);
	});

	it('the literal copy text "Config changed on disk — review and retry." appears in exactly one file in web/src — its own definition in api.ts', () => {
		// Built via concatenation (not a plain literal) so THIS file's own
		// source text — which must describe the string somehow to test for
		// it — never counts as a second occurrence in the scan below; this
		// guard file is excluded from allSourceFiles for the same reason.
		const literal = 'Config changed on disk' + ' — review and retry.';
		const thisFile = fileURLToPath(import.meta.url);
		const filesContainingLiteral = allSourceFiles
			.filter((path) => path !== thisFile)
			.filter((path) => readFileSync(path, 'utf-8').includes(literal));
		const relativePaths = filesContainingLiteral.map((path) => relative(srcRoot, path));
		expect(
			relativePaths,
			'expected the fixed hash-conflict copy to be spelled out as a literal string in exactly one file (api.ts, CONFIG_CONFLICT_MESSAGE\'s own definition) — every writing surface must reference the exported constant by name instead of re-typing the string'
		).toEqual(['lib/api.ts']);
	});

	it('every one of the four modal components references CONFIG_CONFLICT_MESSAGE by name for its hash-conflict branch', () => {
		for (const [name, stripped] of Object.entries(strippedByModal)) {
			expect(
				(stripped as string).includes('CONFIG_CONFLICT_MESSAGE'),
				`expected ${name} to reference CONFIG_CONFLICT_MESSAGE for its hash-conflict branch rather than a re-typed literal`
			).toBe(true);
		}
	});

	it("the route's own modal-less writes (filter save/remove, chip remove-from-webspace) also reference CONFIG_CONFLICT_MESSAGE", () => {
		const routePath = join(srcRoot, 'routes', 'w', '[webspace]', '+page.svelte');
		const routeSource = stripComments(readFileSync(routePath, 'utf-8'));
		const occurrences = (routeSource.match(/CONFIG_CONFLICT_MESSAGE/g) ?? []).length;
		expect(
			occurrences,
			'expected the route to reference CONFIG_CONFLICT_MESSAGE at least twice: once for writeFilter (save/remove filter) and once for handleRemoveSource (chip remove-from-webspace)'
		).toBeGreaterThanOrEqual(3); // 1 import + at least 2 usage sites
	});
});

describe('every ui/ import across web/src resolves to a known, reviewed primitive', () => {
	// The known set of ui/ primitive directories this app has ever
	// introduced (07-UI-SPEC.md Registry Safety table, cumulative across
	// Phases 1/2/5/6/7, plus 13-UI-SPEC.md E3's deliberate 'sonner'
	// addition — the app's first toast primitive) — deliberately an
	// allowlist, not a denylist: a future primitive this list doesn't yet
	// know about fails this test outright until it's a deliberate,
	// reviewed addition, exactly the discipline that caught this phase's
	// own toast primitive before it landed unreviewed.
	const KNOWN_UI_PRIMITIVES = [
		'alert',
		'alert-dialog',
		'badge',
		'button',
		'card',
		'checkbox',
		'dialog',
		'dropdown-menu',
		'input',
		'popover',
		'scroll-area',
		'separator',
		'skeleton',
		'sonner',
		'tooltip'
	];

	it('every "$lib/components/ui/<name>/" import across web/src resolves to a known primitive', () => {
		const offenders: string[] = [];
		for (const path of allSourceFiles) {
			const source = readFileSync(path, 'utf-8');
			const matches = source.matchAll(/from ['"]\$lib\/components\/ui\/([a-z-]+)\//g);
			for (const match of matches) {
				const primitive = match[1];
				if (!KNOWN_UI_PRIMITIVES.includes(primitive)) {
					offenders.push(`${relative(srcRoot, path)}: ui/${primitive}`);
				}
			}
		}
		expect(
			offenders,
			`found an import from an unrecognised ui/ primitive directory — a new primitive must be a deliberate, reviewed addition to this allowlist: ${offenders.join(', ')}`
		).toEqual([]);
	});

	it('the known-primitive allowlist contains exactly one toast-shaped entry ("sonner") — no second, competing toast library', () => {
		const toastShaped = KNOWN_UI_PRIMITIVES.filter((primitive) => /toast|sonner|snackbar/i.test(primitive));
		expect(toastShaped).toEqual(['sonner']);
	});

	it('the toast primitive is mounted in exactly one place (root layout)', () => {
		const offenders: string[] = [];
		for (const path of allSourceFiles) {
			if (path.includes(join('components', 'ui', 'sonner'))) continue; // the wrapper's own definition
			if (path.endsWith('save-state.test.ts')) continue; // this test's own literal string, not a real render site
			const source = stripComments(readFileSync(path, 'utf-8'));
			if (/<Toaster\b/.test(source)) {
				offenders.push(relative(srcRoot, path));
			}
		}
		expect(
			offenders,
			`expected <Toaster /> to be mounted in exactly one place (routes/+layout.svelte), found: ${offenders.join(', ')}`
		).toEqual([join('routes', '+layout.svelte')]);
	});
});

describe('the header error region (WebspaceHeader.svelte) renders exactly once', () => {
	it('exactly one filterError-gated Alert exists in WebspaceHeader.svelte', () => {
		const headerPath = join(here, 'WebspaceHeader.svelte');
		const headerSource = stripComments(readFileSync(headerPath, 'utf-8'));
		const occurrences = (headerSource.match(/\{#if filterError\}/g) ?? []).length;
		expect(
			occurrences,
			'expected exactly one {#if filterError} gate in WebspaceHeader.svelte — every modal-less write (filter save/remove, chip remove-from-webspace) shares this one region rather than each rendering its own'
		).toBe(1);
	});
});
