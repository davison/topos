// 12-01-PLAN.md Task 2 (12-UI-SPEC.md F2): the local-exec branch OpenInSource
// gains for a same-origin /api/-shaped link.url — the kernel-mediated
// xdg-open route the filesystem source's file://-scheme deep_link rewrites
// to at serve time (kernel/httpapi/stream.go's resolveStreamLinkURL).
//
// House pattern (matches repin.test.ts / chip-edit-menu.test.ts): this repo
// has no Svelte component mount harness (web/vite.config.ts's vitest
// environment is 'node' — no jsdom, no @testing-library/svelte). Component
// logic is verified by comment-stripped source scanning instead of DOM
// mounting/interaction, exactly like every other *.test.ts file in this
// directory that asserts on a .svelte file's own markup/script shape.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const componentPath = join(here, 'OpenInSource.svelte');

function stripComments(source: string): string {
	return source
		.replace(/<!--[\s\S]*?-->/g, ' ')
		.replace(/\/\*[\s\S]*?\*\//g, ' ')
		.replace(/\/\/.*$/gm, ' ');
}

function extractBetween(source: string, startMarker: string, endMarker: string): string {
	const startIndex = source.indexOf(startMarker);
	expect(startIndex, `expected to find "${startMarker}" in the scanned source`).toBeGreaterThanOrEqual(
		0
	);
	const endIndex = source.indexOf(endMarker, startIndex);
	expect(endIndex, `expected to find "${endMarker}" after "${startMarker}"`).toBeGreaterThan(
		startIndex
	);
	return source.slice(startIndex, endIndex + endMarker.length);
}

const raw = readFileSync(componentPath, 'utf-8');
const stripped = stripComments(raw);

describe('open-in-source-local-exec guard: found a non-empty comment-stripped source', () => {
	it('OpenInSource.svelte', () => {
		expect(stripped.length).toBeGreaterThan(0);
	});
});

// --- branch selection: isLocalExecLink keyed on URL shape alone ---

describe('isLocalExecLink: keyed on a same-origin /api/ path, never a plugin type', () => {
	it("derives isLocalExecLink from link.url.startsWith('/api/')", () => {
		expect(stripped.includes("let isLocalExecLink = $derived(link.url.startsWith('/api/'))")).toBe(
			true
		);
	});

	it('never branches on source_type or a plugin identifier anywhere in the component', () => {
		expect(/\bsource_type\b/.test(stripped)).toBe(false);
		expect(/\bSourceType\b/.test(stripped)).toBe(false);
		expect(/\bplugin\b/i.test(stripped)).toBe(false);
	});
});

// --- both presentations gain the identical conditional href/onclick branch ---

describe('both presentations (full and iconOnly) branch identically on isLocalExecLink', () => {
	it('the iconOnly Button omits href and wires onclick when isLocalExecLink', () => {
		const iconOnlyBlock = extractBetween(stripped, '{#if iconOnly}', '{:else}');
		expect(iconOnlyBlock.includes('href={isLocalExecLink ? undefined : link.url}')).toBe(true);
		expect(iconOnlyBlock.includes('onclick={isLocalExecLink ? openLocalExecLink : undefined}')).toBe(
			true
		);
	});

	it('the full-presentation Button omits href and wires onclick when isLocalExecLink', () => {
		const fullBlock = stripped.slice(stripped.indexOf('{:else}'));
		expect(fullBlock.includes('href={isLocalExecLink ? undefined : link.url}')).toBe(true);
		expect(fullBlock.includes('onclick={isLocalExecLink ? openLocalExecLink : undefined}')).toBe(
			true
		);
	});

	it('a non-local-exec link keeps target="_blank" rel="noopener noreferrer" for both presentations', () => {
		expect(stripped.includes("target={isLocalExecLink ? undefined : '_blank'}")).toBe(true);
		expect(stripped.includes("rel={isLocalExecLink ? undefined : 'noopener noreferrer'}")).toBe(
			true
		);
	});

	it('the icon selection (windowOnly -> AppWindow vs ArrowUpRight) is unchanged by the branch', () => {
		const occurrences = stripped.match(/affordance\.windowOnly/g) ?? [];
		// Once per presentation (iconOnly, full) — the derivation itself is
		// untouched by F2; only the click mechanism and title/label sourcing
		// change.
		expect(occurrences.length).toBe(2);
	});
});

// --- the single POST on click ---

describe('openLocalExecLink: exactly one POST to link.url', () => {
	const fnBody = extractBetween(
		stripped,
		'async function openLocalExecLink() {',
		'\n\t}'
	);

	it('issues exactly one fetch call, POST against link.url', () => {
		const calls = fnBody.match(/fetch\(/g) ?? [];
		expect(calls.length).toBe(1);
		expect(fnBody.includes("fetch(link.url, { method: 'POST' })")).toBe(true);
	});

	it('changes no visible state on a successful (res.ok) response', () => {
		expect(fnBody.includes('if (res.ok) return;')).toBe(true);
	});
});

// --- failure swap: rejected fetch and non-ok response both trigger it ---

describe('failure handling: a rejected fetch and a non-ok response both produce the failure swap', () => {
	const fnBody = extractBetween(stripped, 'async function openLocalExecLink() {', '\n\t}');

	it('a rejected fetch (catch block) calls showOpenFailure with the fixed fallback', () => {
		const catchBlock = fnBody.slice(fnBody.lastIndexOf('catch'));
		expect(catchBlock.includes('showOpenFailure(OPEN_FAILURE_FALLBACK_DETAIL)')).toBe(true);
	});

	it('a non-ok response calls showOpenFailure with a detail variable, not a bare literal', () => {
		expect(fnBody.includes('showOpenFailure(detail);')).toBe(true);
	});
});

describe('showOpenFailure: sets openFailure and schedules a revert', () => {
	const fnBody = extractBetween(stripped, 'function showOpenFailure(detail: string) {', '\n\t}');

	it('assigns openFailure = detail', () => {
		expect(fnBody.includes('openFailure = detail;')).toBe(true);
	});

	it('schedules setTimeout(..., FAILURE_REVERT_MS) that reverts openFailure to null', () => {
		expect(fnBody.includes('setTimeout(() => {')).toBe(true);
		expect(fnBody.includes('openFailure = null;')).toBe(true);
		expect(fnBody.includes('}, FAILURE_REVERT_MS);')).toBe(true);
	});

	it('clears any prior pending revert timer before scheduling a new one', () => {
		expect(fnBody.includes('clearTimeout(revertTimer);')).toBe(true);
	});
});

// --- the fixed 2500ms revert window, and the two locked copy strings ---

describe('Copywriting Contract: fixed strings and the 2500ms revert window', () => {
	it('FAILURE_REVERT_MS is 2500', () => {
		expect(stripped.includes('const FAILURE_REVERT_MS = 2500;')).toBe(true);
	});

	it("OPEN_FAILURE_LABEL is exactly \"Couldn't open\"", () => {
		expect(stripped.includes('const OPEN_FAILURE_LABEL = "Couldn\'t open";')).toBe(true);
	});

	it('OPEN_FAILURE_FALLBACK_DETAIL is the fixed fallback copy', () => {
		expect(
			stripped.includes(
				'const OPEN_FAILURE_FALLBACK_DETAIL = "Couldn\'t open — file may have moved or been removed.";'
			)
		).toBe(true);
	});
});

// --- detail-vs-fallback title/aria-label sourcing ---

describe('title/aria-label: the detail-vs-fallback copy reaches both presentations', () => {
	it('the full-presentation title falls back to affordance.title via openFailure ?? affordance.title', () => {
		expect(stripped.includes('title={openFailure ?? affordance.title}')).toBe(true);
	});

	it('the iconOnly presentation carries the same title AND aria-label sourcing (no visible label to swap)', () => {
		const iconOnlyBlock = extractBetween(stripped, '{#if iconOnly}', '{:else}');
		expect(iconOnlyBlock.includes('title={openFailure ?? affordance.title}')).toBe(true);
		expect(iconOnlyBlock.includes('aria-label={openFailure ?? affordance.label}')).toBe(true);
	});

	it('the full-presentation visible label swaps to OPEN_FAILURE_LABEL and text-destructive while openFailure is set', () => {
		expect(stripped.includes("class=\"truncate {openFailure ? 'text-destructive' : ''}\"")).toBe(
			true
		);
		expect(stripped.includes('{openFailure ? OPEN_FAILURE_LABEL : affordance.label}')).toBe(true);
	});
});

// --- no disabled state, ever, during the in-flight window ---

describe('never disabled: no disabled attribute anywhere in the component', () => {
	it('the word "disabled" never appears', () => {
		expect(stripped.includes('disabled')).toBe(false);
	});
});

// --- zero-one-many: the trigger condition is evaluated per link ---

describe('zero-one-many: isLocalExecLink is a per-instance $derived, not module/global state', () => {
	it('is declared with $derived, so every mounted instance evaluates its own link.url independently', () => {
		expect(stripped.includes('let isLocalExecLink = $derived(')).toBe(true);
	});
});
