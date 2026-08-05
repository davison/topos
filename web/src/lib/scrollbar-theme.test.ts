// Source-scan guard over the scrollbar styling in web/src/app.css. No
// component-mount harness exists (web/vite.config.ts's test block:
// environment: 'node'), and jsdom would not help even if it did — jsdom
// performs no layout and paints no scrollbars. This test reads app.css off
// disk instead, following the same readFileSync/dirname/join/fileURLToPath
// pattern as web/src/lib/pane-layout.test.ts.
//
// The invariants being guarded:
//   - a thin, palette-derived scrollbar is declared once (root inheritance
//     is the whole delivery mechanism — see the comment on the `:root`
//     rule in app.css itself);
//   - every scrollbar color resolves through var(--...) from an existing
//     palette token, never a new literal color (G-01-2/G-02-1 sibling
//     discipline: no new hardcoded value slips into the design system);
//   - no --scrollbar-* custom property is declared inside `@theme inline`,
//     which would shadow a Tailwind v4 theme namespace — the exact
//     mechanism of the recorded G-02-1 regression.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const stylesheetPath = join(dirname(fileURLToPath(import.meta.url)), '..', 'app.css');
const rawSource = readFileSync(stylesheetPath, 'utf-8');

// app.css is heavily commented by house style (see e.g. the standing G-02-1
// warning above @theme inline, which itself mentions "--spacing-<key>" and
// "max-w-*" in prose). An assertion made against the raw file text could be
// satisfied — or tripped — by prose that has no effect on any rendered
// pixel, so every assertion below runs against this comment-stripped
// constant instead, never against rawSource directly. Comments are
// replaced with a single space (not deleted outright) so two declarations
// separated only by a comment can never fuse into one token.
const strippedSource = rawSource.replace(/\/\*[\s\S]*?\*\//g, ' ');

// Slices the `@theme inline { ... }` block out of the (already
// comment-stripped) source. The block carries no nested braces today
// (only property declarations and calc()/var() calls, which use
// parentheses, not braces), so "first opening brace after the opener,
// first closing brace after that" is a correct, simple match. Asserts the
// opener was actually found so a future rename of that block fails loudly
// here instead of silently returning an empty string that would make the
// negative gate below vacuous.
function extractThemeBlock(source: string): string {
	const openerIndex = source.indexOf('@theme inline');
	expect(
		openerIndex,
		'expected to find an "@theme inline" block opener in app.css — this guard cannot scope its negative gate without it'
	).toBeGreaterThanOrEqual(0);
	const braceOpen = source.indexOf('{', openerIndex);
	expect(braceOpen, 'expected an opening brace after "@theme inline"').toBeGreaterThan(openerIndex);
	const braceClose = source.indexOf('}', braceOpen);
	expect(braceClose, 'expected a closing brace to end the "@theme inline" block').toBeGreaterThan(braceOpen);
	return source.slice(braceOpen + 1, braceClose);
}

// Matches a `#` followed by exactly six hex digits — a hardcoded literal
// color value, the thing this whole guard exists to forbid in derived
// scrollbar tokens.
const HEX_LITERAL = /#[0-9a-fA-F]{6}\b/;

describe('scrollbar-theme source-scan guard', () => {
	it('found a non-empty comment-stripped app.css source to scan', () => {
		// Guards against a silent no-op: a wrong path resolution must fail
		// loudly here rather than making every later assertion vacuously
		// pass over an empty string.
		expect(strippedSource.length).toBeGreaterThan(0);
	});

	it('declares a thin scrollbar width and a scrollbar color pair', () => {
		expect(
			/scrollbar-width:\s*thin/.test(strippedSource),
			'expected a `scrollbar-width: thin` declaration somewhere in app.css'
		).toBe(true);
		expect(
			/scrollbar-color:\s*[^;]+;/.test(strippedSource),
			'expected a `scrollbar-color: <thumb> <track>;` declaration somewhere in app.css'
		).toBe(true);
	});

	it("the scrollbar-color declaration's value is composed exclusively of var(--...) references", () => {
		const match = strippedSource.match(/scrollbar-color:\s*([^;]+);/);
		expect(match, 'expected to find a scrollbar-color declaration to inspect').not.toBeNull();
		const value = match![1].trim();
		const tokens = value.split(/\s+/).filter(Boolean);
		expect(
			tokens.length,
			`expected the scrollbar-color value to carry exactly two whitespace-separated tokens (thumb, track); got "${value}"`
		).toBe(2);
		for (const token of tokens) {
			expect(
				/^var\(--[\w-]+\)$/.test(token),
				`expected scrollbar-color token "${token}" to be a bare var(--...) reference — no hex literal or bare color keyword may appear directly in this declaration, only references to tokens already defined elsewhere`
			).toBe(true);
		}
		expect(
			HEX_LITERAL.test(value),
			'the scrollbar-color value must contain no six-digit hex literal'
		).toBe(false);
	});

	it('each color-mix-derived scrollbar custom property references the app palette and carries no hex literal', () => {
		const mixDeclarations = strippedSource.match(/--scrollbar-[\w-]+:\s*color-mix\([^;]+;/g) ?? [];
		expect(
			mixDeclarations.length,
			'expected at least one --scrollbar-* custom property derived via color-mix() from an existing palette token'
		).toBeGreaterThan(0);
		for (const decl of mixDeclarations) {
			expect(
				/var\(--[\w-]+\)/.test(decl),
				`expected color-mix-derived declaration "${decl.trim()}" to reference at least one var(--...) palette token, so the scrollbar palette can never drift from the app palette`
			).toBe(true);
			expect(
				HEX_LITERAL.test(decl),
				`expected color-mix-derived declaration "${decl.trim()}" to contain no six-digit hex literal`
			).toBe(false);
		}
	});

	it('declares no --scrollbar-* custom property inside the @theme inline block', () => {
		const themeBlock = extractThemeBlock(strippedSource);
		expect(
			/--scrollbar/.test(themeBlock),
			'a --scrollbar-* custom property must never be declared inside @theme inline — a key declared inside that block joins a Tailwind v4 theme namespace, which is the exact mechanism of the recorded G-02-1 regression (named spacing keys shadowing the container scale and collapsing every max-w-* utility)'
		).toBe(false);
	});

	it('declares the WebKit pseudo-element fallback for pre-121 Chromium', () => {
		expect(
			strippedSource.includes('::-webkit-scrollbar'),
			'expected a ::-webkit-scrollbar rule (the WebKit fallback base selector)'
		).toBe(true);
		expect(
			strippedSource.includes('::-webkit-scrollbar-thumb'),
			'expected a ::-webkit-scrollbar-thumb rule'
		).toBe(true);
		expect(
			strippedSource.includes('::-webkit-scrollbar-track'),
			'expected a ::-webkit-scrollbar-track rule'
		).toBe(true);
	});
});
