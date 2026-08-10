// 08-10-PLAN.md Task 3's structural guard over StreamList.svelte's
// sync-failed branch (closes 08-UAT.md G-08-3): pins that the branch
// renders StreamSyncDegraded, not StreamError — reverting this one
// component swap silently reintroduces the "The topos service didn't
// respond" outage copy in front of a user whose kernel answered 200 and
// whose only real problem is one source's failed sync.
//
// House pattern (matches qr-panel.test.ts / add-source.test.ts /
// chip-edit-menu.test.ts): comment-stripped source scanning, an
// extractBetween helper that scopes a match so it can never be satisfied
// or tripped by an unrelated part of the file, a found-non-empty-source
// guard first, and one consequence-describing message per assertion —
// 06-04-SUMMARY.md's own record of why a bare grep is insufficient here:
// a bare grep is what let a prior verification gap through.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const streamListPath = join(here, 'StreamList.svelte');

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

const raw = readFileSync(streamListPath, 'utf-8');
const stripped = stripComments(raw);

describe('stream-degraded guard: found non-empty comment-stripped source', () => {
	it('StreamList.svelte', () => {
		expect(stripped.length).toBeGreaterThan(0);
	});
});

describe('the sync-failed branch renders StreamSyncDegraded, not StreamError (08-UAT.md G-08-3)', () => {
	const syncFailedBranch = extractBetween(
		stripped,
		"{:else if variant === 'sync-failed'",
		"{:else if variant === 'empty-filtered'}"
	);

	it('renders StreamSyncDegraded', () => {
		expect(
			syncFailedBranch.includes('<StreamSyncDegraded'),
			'expected the sync-failed branch to render StreamSyncDegraded — reverting this puts the false "The topos service didn\'t respond" outage copy back in front of a user whose kernel answered 200 and whose only real problem is one source\'s failed sync'
		).toBe(true);
	});

	it("passes the response's recorded sync error through", () => {
		expect(
			syncFailedBranch.includes('syncError={response.sync.error}'),
			'expected the sync-failed branch to forward response.sync.error, so the user is still told which source failed and why — dropping this leaves the degraded state uninformative'
		).toBe(true);
	});
});

describe('StreamError renders in exactly one branch — the fetch-failure state, never the sync-failed one', () => {
	it('StreamError appears exactly once in the whole file', () => {
		const occurrences = stripped.match(/<StreamError/g) ?? [];
		expect(
			occurrences.length,
			'expected exactly one <StreamError render site — a second render site (e.g. the sync-failed branch rendering it again) is how 08-UAT.md G-08-3 existed in the first place: one fixed-copy component doing double duty for two distinct causes'
		).toBe(1);
	});

	it("the one StreamError render is keyed on state === 'error' (the fetch-failure state)", () => {
		expect(
			/state === 'error'[\s\S]{0,40}<StreamError/.test(stripped),
			"expected the single <StreamError render site to be gated on state === 'error' — a genuine fetch failure — never on the sync-failed variant"
		).toBe(true);
	});
});

describe('the sync-failed branch still precedes both empty branches in source order (T-02-16)', () => {
	it('sync-failed comes before empty-filtered and empty, compared by index — presence alone cannot distinguish correct precedence from the masking defect T-02-16 exists to prevent', () => {
		const syncFailedIndex = stripped.indexOf("variant === 'sync-failed'");
		const emptyFilteredIndex = stripped.indexOf("variant === 'empty-filtered'");
		const emptyIndex = stripped.indexOf("variant === 'empty'");

		expect(syncFailedIndex, 'expected to find the sync-failed branch').toBeGreaterThanOrEqual(0);
		expect(
			emptyFilteredIndex,
			'expected to find the empty-filtered branch'
		).toBeGreaterThanOrEqual(0);
		expect(emptyIndex, 'expected to find the empty branch').toBeGreaterThanOrEqual(0);

		expect(
			syncFailedIndex,
			'expected the sync-failed branch to precede empty-filtered in source order — a filtered view must never mask a sync failure'
		).toBeLessThan(emptyFilteredIndex);
		expect(
			syncFailedIndex,
			'expected the sync-failed branch to precede empty in source order — an unfiltered empty view must never mask a sync failure'
		).toBeLessThan(emptyIndex);
	});
});
