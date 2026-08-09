// 07-12-PLAN.md Task 2's structural guard over web/src/routes/+page.svelte,
// closing 07-UAT.md G-07-4's client-side half
// (.planning/debug/root-empty-state-service-error.md): with zero
// [webspaces.*] blocks configured, the root route must render "No
// webspaces yet" with a working Create webspace CTA — never the
// kernel-unreachable copy, which was factually wrong (the kernel answered
// 200 OK the whole time) and left the user with no affordance at all.
//
// Two independent defects composed to produce that symptom, and this guard
// pins the fix for both:
//   1. web/src/routes/+page.svelte read `res.config.webspaces` unguarded —
//      Object.keys(null) threw when the kernel's own response serialized a
//      null collection (07-12 Task 1 fixes the kernel side; this file's
//      defensive `?? {}` read is defence in depth against an unmatched
//      older kernel binary).
//   2. That throw landed inside the SAME catch block used to detect an
//      actually-unreachable kernel, so ANY exception raised while
//      processing an already-successful response rendered the
//      "the topos service didn't respond" copy — a general mechanism for
//      misattributing a downstream bug as a kernel outage, not a one-off.
//      A failure of any assertion below means that mechanism has come back:
//      a future bug in the redirect-decision logic would once again lie to
//      the user about a healthy kernel being down.
//
// House pattern (matches filter-chip.test.ts / resize-observer.test.ts):
// comment-stripped source scanning (no component-mount harness in this
// project's vitest config), a found-non-empty-source guard first,
// `extractBetween`/balanced-brace scoping so a match elsewhere in the file
// can never satisfy or trip an assertion by accident, and one
// consequence-describing message per assertion.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const pagePath = join(here, '+page.svelte');
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

const stripped = stripComments(rawPage);

// Extracts the text from `source[openIndex]` (which must be '{') through
// its matching close brace, tracking only brace depth — correct for the
// try/catch block this guard scopes, whose body contains no nested braces
// today, and robust to one growing a nested block later.
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

describe('root-empty-state guard: found a non-empty comment-stripped source', () => {
	it('+page.svelte', () => {
		expect(stripped.length).toBeGreaterThan(0);
	});
});

const onMountIndex = stripped.indexOf('onMount(async');
const catchKeywordIndex = stripped.indexOf('catch', onMountIndex);
const catchBraceOpenIndex = stripped.indexOf('{', catchKeywordIndex);

describe('found the structural markers this guard scopes against', () => {
	it('onMount, then a single catch, then that catch block’s opening brace', () => {
		expect(onMountIndex, 'expected to find onMount(async in +page.svelte').toBeGreaterThanOrEqual(
			0
		);
		expect(catchKeywordIndex, 'expected to find a catch clause after onMount').toBeGreaterThan(
			onMountIndex
		);
		expect(
			catchBraceOpenIndex,
			"expected to find the catch block's opening brace"
		).toBeGreaterThan(catchKeywordIndex);
	});
});

const catchBlock = extractBalancedBraces(stripped, catchBraceOpenIndex);
const onMountThroughFirstCatch = stripped.slice(onMountIndex, catchBraceOpenIndex + catchBlock.length);

describe('the fetch is isolated in its own catch, decision logic sits outside it', () => {
	it('the region from onMount through the end of the first catch block mentions the config request', () => {
		expect(
			onMountThroughFirstCatch.includes('getConfig'),
			'expected the region spanning onMount through the first catch block to contain the getConfig() request — that region is supposed to be exactly the isolated fetch'
		).toBe(true);
	});

	it('that same region does NOT contain the redirect-target resolution', () => {
		expect(
			onMountThroughFirstCatch.includes('resolveRedirectTarget'),
			'expected resolveRedirectTarget to sit OUTSIDE the fetch’s catch — if it is reachable from inside the catch, a throw in the redirect decision could again be misattributed as a kernel-unreachable failure'
		).toBe(false);
	});

	it('that same region does NOT contain the empty-phase assignment', () => {
		expect(
			onMountThroughFirstCatch.includes("phase = 'empty'"),
			"expected the empty-phase assignment to sit OUTSIDE the fetch's catch — folding it back inside would reintroduce the exact mechanism G-07-4 closes"
		).toBe(false);
	});

	it('the catch block itself assigns the error phase and returns immediately', () => {
		expect(
			catchBlock.includes("phase = 'error'"),
			'expected the isolated catch to assign the error phase'
		).toBe(true);
		expect(
			catchBlock.includes('return'),
			'expected the isolated catch to return immediately, so nothing below it in onMount ever runs after a fetch failure'
		).toBe(true);
	});
});

describe('each phase is assigned exactly once, in the right place', () => {
	it('the error phase is assigned exactly once, and it is inside the first catch block', () => {
		const occurrences = stripped.match(/phase\s*=\s*'error'/g) ?? [];
		expect(
			occurrences.length,
			'expected exactly one error-phase assignment in the whole file — a second one outside the catch would be a second, unguarded path to the service-unreachable copy'
		).toBe(1);
	});

	it('the empty phase is assigned exactly once, and it is outside any catch', () => {
		const occurrences = stripped.match(/phase\s*=\s*'empty'/g) ?? [];
		expect(
			occurrences.length,
			'expected exactly one empty-phase assignment in the whole file'
		).toBe(1);
		expect(
			catchBlock.includes("phase = 'empty'"),
			'expected the empty-phase assignment NOT to live inside the isolated catch block'
		).toBe(false);
	});

	it('this file declares exactly one catch clause', () => {
		const occurrences = stripped.match(/\bcatch\b/g) ?? [];
		expect(
			occurrences.length,
			'expected exactly one catch clause in +page.svelte, confirming there is no second, unscoped catch elsewhere in the file that this guard has not accounted for'
		).toBe(1);
	});
});

describe('a defensive read guards the webspaces collection before its keys are taken', () => {
	it('reads res.config.webspaces with a `?? {}` fallback before calling Object.keys', () => {
		expect(
			/Object\.keys\(\s*res\.config\.webspaces\s*\?\?\s*\{\}\s*\)/.test(stripped),
			'expected a defensive `res.config.webspaces ?? {}` read before Object.keys — a direct unguarded read is what threw and produced this exact defect (G-07-4)'
		).toBe(true);
	});
});

describe('the empty-state branch still renders the CTA the user found missing', () => {
	it('renders a "Create webspace" control', () => {
		expect(
			stripped.includes('Create webspace'),
			'expected the empty-state branch to still render its Create webspace control'
		).toBe(true);
	});

	it('still mounts CreateWebspaceModal', () => {
		expect(
			stripped.includes('<CreateWebspaceModal'),
			'expected the empty-state branch to still mount CreateWebspaceModal — the CTA is meaningless without it'
		).toBe(true);
	});
});

describe('the service-unreachable copy still appears verbatim in the error branch', () => {
	it('carries the exact "service didn’t respond" wording, unchanged', () => {
		expect(
			stripped.includes("Couldn't load this webspace"),
			'expected the error branch’s existing heading copy to be unchanged'
		).toBe(true);
		expect(
			stripped.includes("the topos service didn't respond"),
			'expected the error branch’s existing body copy to be unchanged — this plan changes which condition reaches this state, never what it says'
		).toBe(true);
	});
});
