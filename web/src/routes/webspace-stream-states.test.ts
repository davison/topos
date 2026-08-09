// 07-15-PLAN.md Task 2's structural guard, closing 07-UAT.md G-07-1's client
// half (.planning/debug/new-webspace-transient-service-error.md): a typed
// `ApiError('webspace_not_found')` answer from a healthy kernel must render
// a distinct not-configured state, never the fixed service-unreachable copy
// that state 07-12 already fixed on the root route's own catch. This guard
// pins:
//   1. web/src/routes/w/[webspace]/+page.svelte's load() catch classifies
//      on the caught error's code, producing 'not-found' for
//      webspace_not_found and 'error' for everything else, and the
//      generation (stale-navigation) check still runs first, unmoved.
//   2. web/src/lib/components/StreamList.svelte renders the new
//      StreamMissing component for the not-found state and still renders
//      StreamError for the error state.
//   3. web/src/lib/components/StreamMissing.svelte carries the approved
//      heading and body copy and offers no Retry control (planning choice
//      5 — a webspace_not_found answer is definitive, not transient).
//
// House pattern (matches root-empty-state.test.ts / filter-chip.test.ts /
// resize-observer.test.ts): comment-stripped source scanning (no
// component-mount harness in this project's vitest config), a
// found-non-empty-source guard first, extractBalancedBraces scoping so a
// match elsewhere in a file can never satisfy or trip an assertion by
// accident, and one consequence-describing message per assertion. Placed
// flat in web/src/routes/ (not beside the route file) because the route
// directory is literally named `w/[webspace]`, and a glob-special
// character in a test path is a class of runner-discovery surprise removed
// for zero cost (planning choice 6).

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const pagePath = join(here, 'w', '[webspace]', '+page.svelte');
const streamListPath = join(here, '..', 'lib', 'components', 'StreamList.svelte');
const streamMissingPath = join(here, '..', 'lib', 'components', 'StreamMissing.svelte');

const rawPage = readFileSync(pagePath, 'utf-8');
const rawStreamList = readFileSync(streamListPath, 'utf-8');
const rawStreamMissing = readFileSync(streamMissingPath, 'utf-8');

// Block comments replaced first (so a line-comment marker inside a block
// comment can't truncate the block-comment strip), then HTML and line
// comments — each replaced with a single space, never deleted outright, so
// two tokens separated only by a comment can never fuse into one
// identifier and no assertion below can be satisfied or tripped by this
// file's own prose.
function stripComments(source: string): string {
	return source
		.replace(/<!--[\s\S]*?-->/g, ' ')
		.replace(/\/\*[\s\S]*?\*\//g, ' ')
		.replace(/\/\/.*$/gm, ' ');
}

const strippedPage = stripComments(rawPage);
const strippedStreamList = stripComments(rawStreamList);
const strippedStreamMissing = stripComments(rawStreamMissing);

// Extracts the text from `source[openIndex]` (which must be '{') through
// its matching close brace, tracking only brace depth.
function extractBalancedBraces(source: string, openIndex: number): string {
	let depth = 0;
	for (let i = openIndex; i < source.length; i++) {
		if (source[i] === '{') depth++;
		else if (source[i] === '}') {
			depth--;
			if (depth === 0) return source.slice(openIndex, i + 1);
		}
	}
	throw new Error(`unbalanced braces starting at index ${openIndex}`);
}

describe('webspace-stream-states guard: found non-empty comment-stripped sources', () => {
	it('+page.svelte', () => {
		expect(strippedPage.length).toBeGreaterThan(0);
	});
	it('StreamList.svelte', () => {
		expect(strippedStreamList.length).toBeGreaterThan(0);
	});
	it('StreamMissing.svelte', () => {
		expect(strippedStreamMissing.length).toBeGreaterThan(0);
	});
});

// Scope to load()'s own catch block, not the file's only catch clause in
// general — this file also has code before/after load(); anchoring on
// 'async function load(' first keeps the scope honest even if a second
// catch is ever added elsewhere in the file.
const loadFnIndex = strippedPage.indexOf('async function load(');
const catchKeywordIndex = strippedPage.indexOf('catch', loadFnIndex);
const catchBraceOpenIndex = strippedPage.indexOf('{', catchKeywordIndex);

describe('found the structural markers this guard scopes against', () => {
	it('load(), then a catch, then that catch block’s opening brace', () => {
		expect(loadFnIndex, 'expected to find async function load( in +page.svelte').toBeGreaterThanOrEqual(
			0
		);
		expect(catchKeywordIndex, 'expected to find a catch clause after load(').toBeGreaterThan(
			loadFnIndex
		);
		expect(
			catchBraceOpenIndex,
			"expected to find the catch block's opening brace"
		).toBeGreaterThan(catchKeywordIndex);
	});
});

const catchBlock = extractBalancedBraces(strippedPage, catchBraceOpenIndex);

describe("load()'s catch classifies a typed not-found answer apart from every other failure", () => {
	it('the catch block still begins with the generation (stale-navigation) check', () => {
		const genCheckIndex = catchBlock.indexOf('if (gen !== navGeneration) return;');
		expect(
			genCheckIndex,
			'expected the generation check to be present in the catch block, unmoved by this change — reopening the stale-navigation race is exactly what 07-15-PLAN.md prohibits'
		).toBeGreaterThanOrEqual(0);
		// "begins with" here means it precedes the classification below, not
		// necessarily byte offset 0 of the block (which is the brace itself).
		const classificationIndex = catchBlock.indexOf("loadState =");
		expect(
			classificationIndex,
			'expected the classification assignment to exist in the catch block'
		).toBeGreaterThan(-1);
		expect(
			genCheckIndex,
			'expected the generation check to run BEFORE the classification assignment'
		).toBeLessThan(classificationIndex);
	});

	it('classifies on the caught error’s code, producing both the not-found and the error state', () => {
		expect(
			catchBlock.includes('webspace_not_found'),
			'expected the catch block to branch on the webspace_not_found code'
		).toBe(true);
		expect(
			catchBlock.includes("'not-found'"),
			'expected the catch block to be able to produce the not-found state'
		).toBe(true);
		expect(
			catchBlock.includes("'error'"),
			'expected the catch block to still be able to produce the error state for every other failure'
		).toBe(true);
		expect(
			catchBlock.includes('ApiError'),
			'expected the classification to check the caught value is an ApiError before reading its code'
		).toBe(true);
	});
});

describe('StreamList.svelte renders the not-found branch as well as the error branch', () => {
	it('imports and renders StreamMissing for the not-found state', () => {
		const importOccurrences = strippedStreamList.match(/StreamMissing/g) ?? [];
		expect(
			importOccurrences.length,
			'expected at least the import and the branch to reference StreamMissing'
		).toBeGreaterThanOrEqual(2);
		expect(
			/state === 'not-found'[\s\S]{0,40}<StreamMissing/.test(strippedStreamList),
			'expected a not-found branch that renders <StreamMissing'
		).toBe(true);
	});

	it('still renders StreamError for the error state', () => {
		expect(
			/state === 'error'[\s\S]{0,40}<StreamError/.test(strippedStreamList),
			'expected the existing error branch rendering <StreamError to be unchanged'
		).toBe(true);
	});
});

describe('StreamMissing.svelte carries the approved copy and offers no Retry', () => {
	it('contains the approved heading', () => {
		expect(
			strippedStreamMissing.includes("That webspace isn't configured"),
			'expected the approved not-configured heading copy'
		).toBe(true);
	});

	it('contains the approved body, interpolating the webspace name', () => {
		// Collapse whitespace runs (the source wraps this paragraph across
		// lines) so the assertion matches the copy regardless of how the
		// markup happens to be line-wrapped.
		const normalizedBody = strippedStreamMissing.replace(/\s+/g, ' ');
		expect(
			normalizedBody.includes(
				"isn't in your config — it may have been renamed or removed. Pick one from the switcher above."
			),
			'expected the approved not-configured body copy'
		).toBe(true);
		expect(
			strippedStreamMissing.includes('{webspace}'),
			'expected the webspace name to be interpolated into the body'
		).toBe(true);
	});

	it('contains no Retry control — the answer is definitive and cannot be changed by retrying', () => {
		expect(
			strippedStreamMissing.includes('Retry'),
			'expected no Retry button on the not-configured state (planning choice 5)'
		).toBe(false);
		expect(
			strippedStreamMissing.includes('onretry'),
			'expected no onretry prop/handler wired into the not-configured state'
		).toBe(false);
	});
});
