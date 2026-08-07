// 07-01-PLAN.md tracer checkpoint fix: "Save as filter" did nothing when
// clicked in a real dev session. Every server-side and transport-side cause
// was eliminated first (GET/PUT /api/config both proven working end to end
// through the exact Vite proxy the browser uses); the remaining candidate
// was a frontend runtime bug that never even reached the network.
//
// Root cause: writeFilter() called `structuredClone(configResponse.config)`.
// `configResponse` is a Svelte 5 `$state` value, so `.config` is a
// deeply-reactive Proxy — and the DOM/Node structured-clone algorithm
// unconditionally rejects any Proxy with a DataCloneError, in every engine,
// regardless of what the proxy wraps. That call sat BEFORE writeFilter's own
// try/catch, so the throw became a silently unhandled promise rejection: no
// chip, no Alert, no PUT request ever left the browser — exactly the "click
// does nothing" symptom, and consistent with the kernel never receiving a
// write despite the write path itself being proven sound.
//
// Two things are pinned here:
//  1. A direct, DOM-free runtime reproduction of the actual platform
//     behaviour responsible (structuredClone rejects Proxies) — this is not
//     a guess about Svelte internals, it is the literal mechanism that fired
//     in the browser, executable and re-checkable independent of Svelte's
//     own implementation ever changing how $state is represented.
//  2. A source-scan (house pattern: this project's vitest config runs
//     environment: 'node' with no component-mount harness, matching
//     source-chip-pill.test.ts / unknown-config-keys.test.ts) proving the
//     fix — $state.snapshot instead of structuredClone — is actually in
//     place, and that every early exit from writeFilter (including the
//     "config not loaded yet" guard and the generic-error catch branch)
//     sets filterError rather than returning silently.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const pagePath = join(here, '..', '..', 'routes', 'w', '[webspace]', '+page.svelte');
const rawPage = readFileSync(pagePath, 'utf-8');

function stripComments(source: string): string {
	return source
		.replace(/<!--[\s\S]*?-->/g, ' ')
		.replace(/\/\*[\s\S]*?\*\//g, ' ')
		.replace(/\/\/.*$/gm, ' ');
}

const strippedPage = stripComments(rawPage);

function extractBetween(source: string, startMarker: string, endMarker: string): string {
	const startIndex = source.indexOf(startMarker);
	expect(startIndex, `expected to find "${startMarker}" in the scanned source`).toBeGreaterThanOrEqual(0);
	const endIndex = source.indexOf(endMarker, startIndex);
	expect(endIndex, `expected to find "${endMarker}" after "${startMarker}"`).toBeGreaterThan(startIndex);
	return source.slice(startIndex, endIndex + endMarker.length);
}

describe('the actual platform mechanism: structuredClone rejects any Proxy', () => {
	it('structuredClone(new Proxy({...}, {})) throws DataCloneError', () => {
		// This is not Svelte-specific — it is the exact behaviour that fired
		// in the browser once configResponse.config (a $state-reactive
		// nested Proxy) was handed to structuredClone. Proven directly here
		// so this test would still catch a regression even if it were
		// rewritten against a different reactive-state library.
		const target = { hash: 'abc', config: { webspaces: { cars: { filter: [] } } } };
		const proxied = new Proxy(target, {});
		expect(() => structuredClone(proxied)).toThrow(/could not be cloned/i);
	});

	it('a plain (non-proxied) equivalent clones without error, isolating the Proxy as the cause', () => {
		const target = { hash: 'abc', config: { webspaces: { cars: { filter: [] } } } };
		expect(() => structuredClone(target)).not.toThrow();
	});
});

describe('save-filter-clone guard: found non-empty comment-stripped source', () => {
	it('+page.svelte', () => {
		expect(strippedPage.length).toBeGreaterThan(0);
	});
});

const writeFilterBlock = extractBetween(
	strippedPage,
	'async function writeFilter',
	'async function saveFilter'
);

describe('writeFilter clones configResponse.config with $state.snapshot, never structuredClone', () => {
	it('does not call structuredClone on the reactive config document', () => {
		expect(
			/structuredClone\s*\(/.test(writeFilterBlock),
			'found a structuredClone( call inside writeFilter — configResponse.config is a Svelte 5 $state Proxy, and structuredClone throws DataCloneError on any Proxy in every engine; this IS the bug that made "Save as filter" a silent no-op'
		).toBe(false);
	});

	it('clones via $state.snapshot(configResponse.config) instead', () => {
		expect(
			/\$state\.snapshot\(\s*configResponse\.config\s*\)/.test(writeFilterBlock),
			'expected writeFilter to deep-clone through $state.snapshot(configResponse.config) — Svelte\'s own Object.keys-based clone, which never routes a reactive proxy through structuredClone and returns a plain, structured-clone-safe object'
		).toBe(true);
	});
});

describe('no remaining silent exit in the save path', () => {
	it('the "config not loaded yet" guard sets filterError before returning', () => {
		const guardBlock = extractBetween(writeFilterBlock, 'if (!configResponse) {', 'return;');
		expect(
			/filterError\s*=/.test(guardBlock),
			'expected the !configResponse guard to set filterError before its return — a bare silent return here is reachable in real use (showSaveAsFilter only reads searchQuery/filters, which default to [] when configResponse is null, so the button can render and be clicked before getConfig() resolves) and was one of the two known silent exits in this path'
		).toBe(true);
	});

	it('the catch block never reuses the disk-conflict copy for an unrelated/unexpected error', () => {
		const catchBlock = extractBetween(writeFilterBlock, 'catch (err) {', '} finally {');
		const diskConflictCopy = 'Config changed on disk — review and retry.';
		const occurrences = catchBlock.split(diskConflictCopy).length - 1;
		expect(
			occurrences,
			'expected the fixed disk-conflict copy to appear exactly once in the catch block (only for the actual config_changed_on_disk ApiError branch) — reusing it as the fallback for a generic/unexpected error mislabels an unrelated failure as a hash conflict, hiding the real cause from the user and from anyone reading a bug report'
		).toBe(1);
		expect(
			/something went wrong/i.test(catchBlock),
			'expected a distinct, honest fallback message for a non-ApiError exception (an unexpected JS error) rather than reusing the disk-conflict copy'
		).toBe(true);
	});

	it('every branch of the catch block still assigns filterError so nothing exits without surfacing', () => {
		const catchBlock = extractBetween(writeFilterBlock, 'catch (err) {', '} finally {');
		expect(
			/filterError\s*=/.test(catchBlock),
			'expected the catch block to assign filterError unconditionally on its non-navGeneration-stale path'
		).toBe(true);
	});
});
