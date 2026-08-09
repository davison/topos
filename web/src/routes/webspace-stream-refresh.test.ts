// 07-16-PLAN.md Task 3's structural guard, closing 07-UAT.md G-07-7's
// residual case (.planning/debug/removed-source-items-linger-in-stream.md):
// with the synchronous purge in Supervisor.Apply (Task 2) clearing a
// de-participated pair before a config save answers, the one gap left is an
// EAGER RESYNC that FAILS at the time of the save — its rows are only
// cleared by a LATER scheduled sync, and until now nothing on screen ever
// refetched the stream when that background sync finished. This guard pins:
//
//   1. web/src/routes/w/[webspace]/+page.svelte's ensurePolling's stop
//      branch (the tick where syncing falls to false) ALSO calls the
//      stream load, passing the quiet flag, so a completed sync updates
//      the stream and not only the chips — and never blanks an
//      already-rendered stream into a loading skeleton.
//   2. The tick captures navGeneration into a local BEFORE its first
//      await, so a webspace navigation mid-poll cannot be clobbered by a
//      stale refetch.
//   3. load() guards its loading-state assignment on the quiet flag
//      rather than assigning it unconditionally, and its catch — still
//      beginning with the generation (stale-navigation) check, still
//      classifying the typed not-found error exactly as 07-15 left it —
//      leaves the screen untouched on a quiet failure.
//   4. handleRemoveSource performs exactly one stream refetch, with no
//      delay, timeout or retry newly wrapped around it.
//
// House pattern (matches webspace-stream-states.test.ts / root-empty-state
// .test.ts / filter-chip.test.ts / resize-observer.test.ts): comment-
// stripped source scanning (no component-mount harness in this project's
// vitest config), a found-non-empty-source guard first, extractBalancedBraces
// scoping so a match elsewhere in the file can never satisfy or trip an
// assertion by accident, and one consequence-describing message per
// assertion.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const pagePath = join(here, 'w', '[webspace]', '+page.svelte');

const rawPage = readFileSync(pagePath, 'utf-8');

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

describe('webspace-stream-refresh guard: found a non-empty comment-stripped source', () => {
	it('+page.svelte', () => {
		expect(strippedPage.length).toBeGreaterThan(0);
	});
});

// Scope to ensurePolling's own function body — not the whole file's only
// setInterval, in general, so a later addition elsewhere can never satisfy
// this guard by accident.
const ensurePollingIndex = strippedPage.indexOf('function ensurePolling(');
const ensurePollingBraceOpenIndex = strippedPage.indexOf('{', ensurePollingIndex);

// Scope to load()'s own function body.
const loadFnIndex = strippedPage.indexOf('async function load(');
const loadBraceOpenIndex = strippedPage.indexOf('{', strippedPage.indexOf(')', loadFnIndex));

// Scope to handleRemoveSource's own function body.
const removeFnIndex = strippedPage.indexOf('async function handleRemoveSource(');
const removeBraceOpenIndex = strippedPage.indexOf('{', strippedPage.indexOf(')', removeFnIndex));

describe('found the structural markers this guard scopes against', () => {
	it('ensurePolling(, then its opening brace', () => {
		expect(
			ensurePollingIndex,
			'expected to find function ensurePolling( in +page.svelte'
		).toBeGreaterThanOrEqual(0);
		expect(
			ensurePollingBraceOpenIndex,
			"expected to find ensurePolling's opening brace"
		).toBeGreaterThan(ensurePollingIndex);
	});

	it('async function load(, then its opening brace', () => {
		expect(loadFnIndex, 'expected to find async function load( in +page.svelte').toBeGreaterThanOrEqual(
			0
		);
		expect(loadBraceOpenIndex, "expected to find load()'s opening brace").toBeGreaterThan(
			loadFnIndex
		);
	});

	it('async function handleRemoveSource(, then its opening brace', () => {
		expect(
			removeFnIndex,
			'expected to find async function handleRemoveSource( in +page.svelte'
		).toBeGreaterThanOrEqual(0);
		expect(
			removeBraceOpenIndex,
			"expected to find handleRemoveSource's opening brace"
		).toBeGreaterThan(removeFnIndex);
	});
});

const ensurePollingBody = extractBalancedBraces(strippedPage, ensurePollingBraceOpenIndex);
const loadBody = extractBalancedBraces(strippedPage, loadBraceOpenIndex);
const removeBody = extractBalancedBraces(strippedPage, removeBraceOpenIndex);

describe("ensurePolling's stop branch refetches the stream, quietly, on the captured generation", () => {
	it('calls load( inside the interval callback, passing the quiet flag', () => {
		expect(
			/load\(\s*gen\s*,\s*\{\s*quiet:\s*true\s*\}\s*\)/.test(ensurePollingBody),
			'expected the stop branch to call load(gen, { quiet: true }) so a completed sync updates the stream and not only the chips, without blanking an already-rendered stream into a skeleton'
		).toBe(true);
	});

	it('the load( call sits after clearInterval, inside the "no longer syncing" branch', () => {
		const clearIntervalIndex = ensurePollingBody.indexOf('clearInterval(pollHandle)');
		const loadCallIndex = ensurePollingBody.indexOf('load(gen');
		expect(
			clearIntervalIndex,
			'expected to find the clearInterval(pollHandle) call this guard anchors against'
		).toBeGreaterThanOrEqual(0);
		expect(
			loadCallIndex,
			'expected the quiet stream refetch to be present in the stop branch'
		).toBeGreaterThan(-1);
		expect(
			loadCallIndex,
			'expected the quiet stream refetch to run AFTER the interval is cleared — the poll must not keep re-ticking once the stream refetch is in flight'
		).toBeGreaterThan(clearIntervalIndex);
	});

	it('captures the generation into a local BEFORE the tick’s first await, matching every other call path in this file', () => {
		const genCaptureIndex = ensurePollingBody.indexOf('const gen = navGeneration;');
		const firstAwaitIndex = ensurePollingBody.indexOf('await ');
		expect(
			genCaptureIndex,
			'expected ensurePolling to capture navGeneration into a local "gen" before awaiting anything, so a webspace navigation mid-tick cannot clobber a stale refetch onto the new page'
		).toBeGreaterThanOrEqual(0);
		expect(
			firstAwaitIndex,
			'expected at least one await inside the poll tick'
		).toBeGreaterThan(-1);
		expect(
			genCaptureIndex,
			'expected the generation capture to precede the tick’s first await'
		).toBeLessThan(firstAwaitIndex);
	});
});

describe("load()'s loading-state assignment is guarded on the quiet flag", () => {
	it('does not assign loadState unconditionally as the first statement', () => {
		expect(
			/^\s*loadState\s*=\s*'loading'/.test(loadBody.slice(1).trimStart()),
			"expected load()'s body to no longer open with an unconditional loadState = 'loading' assignment — a quiet caller must not trigger the loading skeleton"
		).toBe(false);
	});

	it('assigns the loading state only when NOT quiet', () => {
		expect(
			/if\s*\(\s*!\s*quiet\s*\)\s*loadState\s*=\s*'loading'/.test(loadBody),
			"expected load() to assign loadState = 'loading' only inside an if (!quiet) guard"
		).toBe(true);
	});

	it('accepts an options parameter carrying the quiet flag, defaulting to false', () => {
		expect(
			strippedPage.slice(loadFnIndex, loadBraceOpenIndex).includes('quiet'),
			"expected load()'s parameter list to carry the quiet option"
		).toBe(true);
		expect(
			loadBody.includes('options?.quiet ?? false') || loadBody.includes('options?.quiet??false'),
			'expected quiet to default to false so every existing (non-quiet) call site is unaffected'
		).toBe(true);
	});
});

describe("load()'s catch survives this change exactly as 07-15 left it, plus a quiet early-return", () => {
	const catchKeywordIndex = loadBody.indexOf('catch');
	const catchBraceOpenIndex = loadBody.indexOf('{', catchKeywordIndex);
	const catchBlock = extractBalancedBraces(loadBody, catchBraceOpenIndex);

	it('the catch block still begins with the generation (stale-navigation) check', () => {
		const genCheckIndex = catchBlock.indexOf('if (gen !== navGeneration) return;');
		expect(
			genCheckIndex,
			'expected the generation check to still be present, unmoved, at the top of the catch block'
		).toBeGreaterThanOrEqual(0);
	});

	it('returns early on a quiet failure, before touching response or loadState', () => {
		const genCheckIndex = catchBlock.indexOf('if (gen !== navGeneration) return;');
		const quietReturnIndex = catchBlock.indexOf('if (quiet) return;');
		const responseNullIndex = catchBlock.indexOf('response = null;');
		expect(
			quietReturnIndex,
			'expected a quiet early-return in the catch block, so a failed background refresh leaves the screen exactly as it was'
		).toBeGreaterThan(-1);
		expect(
			quietReturnIndex,
			'expected the quiet return to come after the generation check'
		).toBeGreaterThan(genCheckIndex);
		expect(
			quietReturnIndex,
			'expected the quiet return to come before response is ever reset to null'
		).toBeLessThan(responseNullIndex);
	});

	it('still classifies a typed not-found answer apart from every other failure', () => {
		expect(
			catchBlock.includes('webspace_not_found'),
			'expected the catch block to still branch on the webspace_not_found code (07-15-PLAN.md)'
		).toBe(true);
		expect(catchBlock.includes("'not-found'"), 'expected the not-found state to still be reachable').toBe(
			true
		);
		expect(catchBlock.includes("'error'"), 'expected the error state to still be reachable').toBe(true);
	});
});

describe('handleRemoveSource performs exactly one stream refetch, unchanged by this task', () => {
	it('calls load( exactly once', () => {
		const occurrences = removeBody.match(/\bload\(/g) ?? [];
		expect(
			occurrences.length,
			'expected handleRemoveSource to call load( exactly once — a second call, a delay, or a retry loop would mean this task touched the one thing its own prohibition forbids changing'
		).toBe(1);
	});

	it('introduces no setTimeout, setInterval, delay or retry wrapper', () => {
		for (const forbidden of ['setTimeout', 'setInterval', 'retry', 'delay(']) {
			expect(
				removeBody.includes(forbidden),
				`expected handleRemoveSource to contain no ${forbidden} — its single immediate refetch must stay exactly as it was`
			).toBe(false);
		}
	});
});

describe('no new timer was introduced anywhere in the file — the only interval is the pre-existing poll', () => {
	it('setInterval appears exactly once', () => {
		const occurrences = strippedPage.match(/\bsetInterval\(/g) ?? [];
		expect(
			occurrences.length,
			'expected exactly one setInterval in the whole file (the pre-existing sync-status poll) — a second interval would mean this task added a new timer instead of reusing the existing poll'
		).toBe(1);
	});
});
