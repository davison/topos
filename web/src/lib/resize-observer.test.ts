// Two describe blocks guard one invariant from two directions. The helper
// block proves observeResize's attachment/teardown semantics in isolation
// (a fake observer, no DOM, no jsdom — matches web/vite.config.ts's
// environment: 'node' test runner). The header block is a comment-stripped
// source-scan guard proving WebspaceHeader.svelte actually wires the
// helper to a ref-driven $effect covering all four measured elements.
// Neither subsumes the other: a helper that behaves correctly but is
// never wired, and a component that appears wired but calls a broken
// helper, are both real defects — this is exactly the gap
// 06-VERIFICATION.md recorded (the observer was constructed in a one-shot
// mount hook that fired before any of its four targets existed, so every
// guarded `observe` call was skipped and nothing ever re-attached).

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { observeResize, type ResizeObserverLike, type CreateResizeObserver } from './resize-observer';

// --- Helper behaviour ---------------------------------------------------

class FakeResizeObserver implements ResizeObserverLike {
	observed: Element[] = [];
	disconnectCount = 0;
	observe(target: Element): void {
		this.observed.push(target);
	}
	disconnect(): void {
		this.disconnectCount++;
	}
}

// `state` is a single mutable object (not individually destructured
// getters) so callers can destructure `{ factory, instances, state }`
// once, up front, and still observe live values afterward — destructuring
// a getter's return value instead would freeze it at call time, before
// observeResize ever runs.
function makeFactory() {
	const instances: FakeResizeObserver[] = [];
	const state: { callCount: number; callback: (() => void) | undefined } = {
		callCount: 0,
		callback: undefined
	};
	const factory: CreateResizeObserver = (onResize) => {
		state.callCount++;
		state.callback = onResize;
		const instance = new FakeResizeObserver();
		instances.push(instance);
		return instance;
	};
	return { factory, instances, state };
}

// Nothing in the helper dereferences a target — it only calls .observe()
// with it — so a plain distinct object cast to Element is sufficient and
// keeps this suite DOM-free.
function makeElement(): Element {
	return {} as Element;
}

describe('observeResize', () => {
	it('observes every bound target exactly once and constructs a single observer', () => {
		const a = makeElement();
		const b = makeElement();
		const c = makeElement();
		const d = makeElement();
		const { factory, instances, state } = makeFactory();

		observeResize([a, b, c, d], () => {}, factory);

		expect(state.callCount, 'expected the factory to be called exactly once').toBe(1);
		expect(
			instances[0].observed,
			'expected all four bound targets to be observed, in the order given'
		).toEqual([a, b, c, d]);
	});

	it('observes only the bound targets and ignores the unbound ones', () => {
		const a = makeElement();
		const c = makeElement();
		const { factory, instances } = makeFactory();

		expect(() => observeResize([a, undefined, c, null], () => {}, factory)).not.toThrow();

		expect(
			instances[0].observed,
			'expected only the two bound targets to be observed, with the unbound ones skipped and nothing thrown'
		).toEqual([a, c]);
	});

	it('constructs no observer at all when every target is still unbound', () => {
		const { factory, state } = makeFactory();

		const teardown = observeResize([undefined, null, undefined, null], () => {}, factory);

		expect(
			state.callCount,
			'expected no observer to be constructed when every target is still unbound — this is the state on first mount, while the sources request is still in flight, and constructing an observer with nothing to watch would be the dead wiring this plan removes'
		).toBe(0);
		expect(
			() => teardown(),
			'expected the returned teardown to be safely callable even though nothing was ever observed'
		).not.toThrow();
	});

	it('routes the observer callback through to the supplied resize handler', () => {
		let resizeCount = 0;
		const { factory, state } = makeFactory();

		observeResize(
			[makeElement()],
			() => {
				resizeCount++;
			},
			factory
		);

		expect(
			state.callback,
			'expected the factory to have captured the callback it was handed'
		).toBeDefined();
		state.callback!();
		expect(
			resizeCount,
			'expected invoking the factory-captured callback to call the supplied resize handler — the path a real layout change travels'
		).toBe(1);
	});

	it('disconnects exactly once when the teardown runs, and a second call is inert', () => {
		const { factory, instances } = makeFactory();
		const teardown = observeResize([makeElement()], () => {}, factory);

		teardown();
		teardown();

		expect(
			instances[0].disconnectCount,
			'expected disconnect to be called exactly once even though the teardown ran twice — a ref-driven effect can legitimately tear down more than once, and a double disconnect must be inert rather than a second call into a released observer'
		).toBe(1);
	});

	it('re-observes a fresh set of elements when called again after teardown', () => {
		const first = makeElement();
		const second = makeElement();
		const { factory, instances } = makeFactory();

		const teardownFirst = observeResize([first], () => {}, factory);
		teardownFirst();
		observeResize([second], () => {}, factory);

		expect(
			instances.length,
			'expected a second, distinct observer to be constructed on re-attachment'
		).toBe(2);
		expect(
			instances[1].observed,
			'expected the second attachment to observe the new element set — the re-attachment a ref-driven effect depends on every time the refs rebind'
		).toEqual([second]);
	});
});

// --- Header source-scan guard --------------------------------------------
//
// scrollbar-theme.test.ts and pane-layout.test.ts establish the house
// pattern this follows: read the component off disk, strip comments so
// prose can never satisfy or trip an assertion, then assert exclusively
// against the stripped constant. A bare grep over raw source is what let
// 06-VERIFICATION.md's gap through once already — the dead mount-hook
// wiring satisfied a grep for 'ResizeObserver' in the component while its
// .observe() calls never fired.

const componentPath = join(
	dirname(fileURLToPath(import.meta.url)),
	'components',
	'WebspaceHeader.svelte'
);
const rawComponentSource = readFileSync(componentPath, 'utf-8');

// Block comments replaced first (so a line-comment marker inside a block
// comment can't truncate the block-comment strip), then line comments.
// Both are replaced with a single space, never deleted outright, so two
// tokens separated only by a comment can never fuse into one identifier.
function stripComments(source: string): string {
	return source.replace(/\/\*[\s\S]*?\*\//g, ' ').replace(/\/\/.*$/gm, ' ');
}

// Extracts the text from `source[openIndex]` (which must be '(') through
// its matching close paren, tracking only paren depth — nested braces
// (object literals, block bodies) don't affect paren balance, so this is
// correct for the arrow-function and call-argument shapes this guard
// inspects.
function extractBalancedParens(source: string, openIndex: number): string {
	let depth = 0;
	for (let i = openIndex; i < source.length; i++) {
		if (source[i] === '(') depth++;
		else if (source[i] === ')') {
			depth--;
			if (depth === 0) return source.slice(openIndex, i + 1);
		}
	}
	throw new Error(`unbalanced parentheses starting at index ${openIndex}`);
}

const scriptMatch = rawComponentSource.match(/<script[^>]*>([\s\S]*?)<\/script>/);
const rawScript = scriptMatch ? scriptMatch[1] : '';
const strippedScript = stripComments(rawScript);
const markup = scriptMatch
	? rawComponentSource.slice(scriptMatch.index! + scriptMatch[0].length)
	: '';

describe('WebspaceHeader source-scan guard', () => {
	it('found a non-empty comment-stripped <script> block that still contains a known measurement identifier', () => {
		// Guards against a silent no-op: a wrong path or a broken extraction
		// must fail loudly here, not make every later assertion in this
		// block vacuously pass over an empty string.
		expect(
			strippedScript.length,
			'expected to find a non-empty <script> block in WebspaceHeader.svelte'
		).toBeGreaterThan(0);
		expect(
			strippedScript.includes('availableWidth'),
			'expected the comment-stripped script to still reference the measurement state it declares'
		).toBe(true);
	});

	it("attaches the observer from inside an $effect whose teardown is the attachment call's return value", () => {
		const effectIndex = strippedScript.indexOf('$effect(');
		expect(
			effectIndex,
			'expected to find a $effect(...) call attaching the resize observer'
		).toBeGreaterThanOrEqual(0);
		const parenStart = effectIndex + '$effect'.length;
		const effectCallText = extractBalancedParens(strippedScript, parenStart);

		expect(
			effectCallText.includes('observeResize('),
			'expected the $effect call to invoke observeResize'
		).toBe(true);

		// Accept exactly two shapes: an arrow with an expression body that
		// returns the call directly, or an arrow with a block body that
		// explicitly `return`s the call. A block body that calls the helper
		// and returns nothing must fail this — that shape leaks an observer
		// on every re-run.
		const isExpressionBody = /\(\s*\)\s*=>\s*observeResize\(/.test(effectCallText);
		const isReturnedFromBlock = /\{[\s\S]*?return\s+observeResize\(/.test(effectCallText);
		expect(
			isExpressionBody || isReturnedFromBlock,
			'expected the $effect body to either return observeResize(...) directly as an expression body, or explicitly `return observeResize(...)` from a block body — a block body that calls the helper without returning it discards the teardown and leaks an observer on every re-run'
		).toBe(true);

		// Pins IN-01 closed: the attachment call must name all four measured
		// elements, including the overflow-trigger measurement clone.
		const callIndex = effectCallText.indexOf('observeResize(');
		const callParenStart = callIndex + 'observeResize'.length;
		const callArgsText = extractBalancedParens(effectCallText, callParenStart);
		for (const name of ['rowEl', 'measureEl', 'trailingEl', 'overflowTriggerMeasureEl']) {
			expect(
				new RegExp(`\\b${name}\\b`).test(callArgsText),
				`expected the observeResize(...) call to name "${name}" among its measured elements`
			).toBe(true);
		}
	});

	it('no longer references the one-shot lifecycle mount hook', () => {
		expect(
			strippedScript.includes('onMount'),
			'expected the onMount import and its lifecycle block to be fully removed — attachment is now ref-driven via $effect, not a one-shot mount hook'
		).toBe(false);
	});

	it('constructs no observer of its own', () => {
		// This necessarily carries the construction expression as a string
		// literal in this test file — that is unavoidable and correct, and
		// is precisely why the task's tree-wide uniqueness gate
		// (`grep -rl 'new ResizeObserver' --exclude='*.test.ts' src/`)
		// excludes *.test.ts. The two checks are complementary: the shell
		// gate proves no *other* production file constructs an observer,
		// this proves the header's own construction is really gone rather
		// than merely moved into a comment.
		expect(
			strippedScript.includes('new ResizeObserver'),
			"expected construction to live solely in the extracted helper's default factory, not inline in the component"
		).toBe(false);
	});

	it('still binds all four measured elements in the markup', () => {
		for (const name of ['rowEl', 'measureEl', 'trailingEl', 'overflowTriggerMeasureEl']) {
			expect(
				markup.includes(`bind:this={${name}}`),
				`expected the markup to still bind ${name} via bind:this, so the identifiers passed to the helper correspond to elements that really exist`
			).toBe(true);
		}
	});
});
