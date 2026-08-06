// A term that matched only an item's title must produce a visible highlight
// on a surface the user can see, and every SPA match surface must resolve
// through one shared class so the treatment cannot drift apart again
// (G-06-1). The existing 188-test suite could never have caught this: it
// encodes the same narrow UI-09 scope 06-01-PLAN.md did, and the runner is
// `environment: 'node'` with no component-mount harness (web/vite.config.ts)
// — so structural proof off disk is the available instrument here, the same
// precedent recorded for 06-04 ("a comment-stripped source-scan guard, not a
// raw grep — a bare grep is what let a gap through once already").
//
// House pattern: scrollbar-theme.test.ts (stylesheet block extraction),
// pane-layout.test.ts (svelte class-attribute scanning) and
// resize-observer.test.ts (comment-stripped <script>/markup split). This
// file proves the WIRING across four files; highlight.test.ts already fully
// covers highlightText's own segmentation algorithm and is not repeated here.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));

const appCssPath = join(here, '..', '..', 'app.css');
const detailPanePath = join(here, 'DetailPane.svelte');
const streamRowPath = join(here, 'StreamRow.svelte');
const searchResultsPath = join(here, 'SearchResults.svelte');

const rawAppCss = readFileSync(appCssPath, 'utf-8');
const rawDetailPane = readFileSync(detailPanePath, 'utf-8');
const rawStreamRow = readFileSync(streamRowPath, 'utf-8');
const rawSearchResults = readFileSync(searchResultsPath, 'utf-8');

// Strips CSS/JS block comments, JS line comments and HTML comments, each
// replaced with a single space (never deleted outright) so two tokens
// separated only by a comment can never fuse into one identifier. No
// assertion below ever runs against a raw*, unstripped source — a prose
// comment describing the invariant (this very file's header, or the
// components' own doc comments) must never be able to satisfy or trip a
// structural assertion.
function stripComments(source: string): string {
	return source
		.replace(/<!--[\s\S]*?-->/g, ' ')
		.replace(/\/\*[\s\S]*?\*\//g, ' ')
		.replace(/\/\/.*$/gm, ' ');
}

const strippedAppCss = stripComments(rawAppCss);
const strippedDetailPane = stripComments(rawDetailPane);
const strippedStreamRow = stripComments(rawStreamRow);
const strippedSearchResults = stripComments(rawSearchResults);

// Matches a `#` followed by exactly six hex digits — a hardcoded literal
// color value, mirroring scrollbar-theme.test.ts's HEX_LITERAL guard.
const HEX_LITERAL = /#[0-9a-fA-F]{6}\b/;

// Slices the `@layer components { ... }` block out of the (already
// comment-stripped) stylesheet source, tracking brace depth so a nested
// rule body's own braces don't prematurely close the outer match.
function extractLayerComponentsBlock(source: string): string {
	const openerIndex = source.indexOf('@layer components');
	expect(
		openerIndex,
		'expected to find an "@layer components" block opener in app.css — this guard cannot scope its assertions without it'
	).toBeGreaterThanOrEqual(0);
	const braceOpen = source.indexOf('{', openerIndex);
	expect(braceOpen, 'expected an opening brace after "@layer components"').toBeGreaterThan(
		openerIndex
	);
	let depth = 0;
	for (let i = braceOpen; i < source.length; i++) {
		if (source[i] === '{') depth++;
		else if (source[i] === '}') {
			depth--;
			if (depth === 0) return source.slice(braceOpen + 1, i);
		}
	}
	throw new Error('unbalanced braces while extracting @layer components block');
}

// Slices a `<selector> { ... }` rule body out of an already comment-stripped
// stylesheet source, given the selector text (e.g. ".search-highlight").
function extractRuleBody(source: string, selector: string): string {
	const selectorIndex = source.indexOf(selector);
	expect(
		selectorIndex,
		`expected to find the "${selector}" selector in the stylesheet`
	).toBeGreaterThanOrEqual(0);
	const braceOpen = source.indexOf('{', selectorIndex);
	expect(braceOpen, `expected an opening brace after "${selector}"`).toBeGreaterThan(
		selectorIndex
	);
	const braceClose = source.indexOf('}', braceOpen);
	expect(braceClose, `expected a closing brace to end the "${selector}" rule`).toBeGreaterThan(
		braceOpen
	);
	return source.slice(braceOpen + 1, braceClose);
}

// Slices the text from the first occurrence of `startMarker` through the
// following occurrence of `endMarker` (inclusive) — used to scope an
// assertion to one rendered element (a title heading, a matched-segment
// each-block) rather than the whole file, so a match elsewhere in the
// document can never satisfy it by accident.
function extractBetween(source: string, startMarker: string, endMarker: string): string {
	const startIndex = source.indexOf(startMarker);
	expect(
		startIndex,
		`expected to find "${startMarker}" in the scanned source`
	).toBeGreaterThanOrEqual(0);
	const endIndex = source.indexOf(endMarker, startIndex);
	expect(
		endIndex,
		`expected to find "${endMarker}" after "${startMarker}"`
	).toBeGreaterThan(startIndex);
	return source.slice(startIndex, endIndex + endMarker.length);
}

describe('search-emphasis guard: found non-empty comment-stripped sources', () => {
	// Guards against a silent no-op: a wrong path resolution must fail
	// loudly here rather than making every later assertion vacuously pass
	// over an empty string.
	it('app.css', () => {
		expect(strippedAppCss.length).toBeGreaterThan(0);
	});
	it('DetailPane.svelte', () => {
		expect(strippedDetailPane.length).toBeGreaterThan(0);
	});
	it('StreamRow.svelte', () => {
		expect(strippedStreamRow.length).toBeGreaterThan(0);
	});
	it('SearchResults.svelte', () => {
		expect(strippedSearchResults.length).toBeGreaterThan(0);
	});
});

describe('one declaration, one place', () => {
	it('the .search-highlight class is declared exactly once in app.css', () => {
		const declarations = strippedAppCss.match(/\.search-highlight\s*\{/g) ?? [];
		expect(
			declarations.length,
			'expected exactly one .search-highlight rule declaration in app.css — a second declaration anywhere (including a component-scoped <style> block elsewhere) is exactly the shape that let the detail pane and the stream row drift onto two different match vocabularies'
		).toBe(1);
	});

	it('the declaration sits inside the @layer components block', () => {
		const componentsBlock = extractLayerComponentsBlock(strippedAppCss);
		expect(
			/\.search-highlight\s*\{/.test(componentsBlock),
			'expected .search-highlight to be declared inside @layer components, alongside .stream-row-surface — a rule declared outside a layer would win specificity battles unpredictably against layered utility classes'
		).toBe(true);
	});

	it('resolves through the existing --warning/--background theme tokens, never a literal hex value', () => {
		const ruleBody = extractRuleBody(strippedAppCss, '.search-highlight');
		expect(
			/var\(--warning\)/.test(ruleBody),
			'expected the background declaration to resolve through var(--warning)'
		).toBe(true);
		expect(
			/var\(--background\)/.test(ruleBody),
			'expected the foreground declaration to resolve through var(--background)'
		).toBe(true);
		expect(
			HEX_LITERAL.test(ruleBody),
			'expected no six-digit hex literal inside the .search-highlight rule body — a new literal color here would defeat the whole point of sharing one token-derived class'
		).toBe(false);
	});
});

describe('no component-scoped redeclaration', () => {
	it('DetailPane.svelte declares no .search-highlight rule of its own', () => {
		expect(
			/\.search-highlight\s*\{/.test(strippedDetailPane),
			'expected DetailPane.svelte to carry no <style> block declaring .search-highlight — a Svelte-scoped class cannot be shared with a sibling component, which is the mechanical reason the detail pane and the stream row drifted apart once already'
		).toBe(false);
	});

	it('StreamRow.svelte declares no .search-highlight rule of its own', () => {
		expect(
			/\.search-highlight\s*\{/.test(strippedStreamRow),
			'expected StreamRow.svelte to carry no <style> block declaring .search-highlight — the class must come from app.css alone'
		).toBe(false);
	});
});

describe('the title surfaces are wired', () => {
	it("DetailPane.svelte's title renders through highlightText and the shared class", () => {
		const headingBlock = extractBetween(strippedDetailPane, '<h2', '</h2>');
		expect(
			headingBlock.includes('highlightText(item.title'),
			'expected the detail-pane title heading to render its each-block over highlightText(item.title, ...) — a title-only FTS match must be visually explained here, not just in the body'
		).toBe(true);
		expect(
			headingBlock.includes('search-highlight'),
			"expected the detail-pane title's matched-segment branch to name the shared search-highlight class"
		).toBe(true);
	});

	it("StreamRow.svelte's title renders through highlightText and the shared class", () => {
		const titleBlock = extractBetween(
			strippedStreamRow,
			'text-[20px] leading-[1.2] font-semibold text-foreground',
			'</p>'
		);
		expect(
			titleBlock.includes('highlightText(item.title'),
			'expected the stream-row title to render its each-block over highlightText(item.title, ...) — a title-only match must be visually explained in the results row too, not only in the opened detail pane'
		).toBe(true);
		expect(
			titleBlock.includes('search-highlight'),
			"expected the stream-row title's matched-segment branch to name the shared search-highlight class"
		).toBe(true);
	});
});

describe('the snippet surface is unified', () => {
	it("StreamRow.svelte's parseSnippet matched segments use the shared class", () => {
		const snippetBlock = extractBetween(
			strippedStreamRow,
			'{#each parseSnippet(snippet)',
			'{/each}'
		);
		expect(
			snippetBlock.includes('search-highlight'),
			'expected the search-result snippet\'s matched-segment branch to name the shared search-highlight class — the same amber treatment the detail pane and the kernel rendition iframe already use'
		).toBe(true);
	});

	it('no font-weight utility is used as a match treatment anywhere in StreamRow.svelte', () => {
		// This is a negative assertion against the comment-stripped SOURCE,
		// never a shell grep — a shell grep would also match plan or comment
		// prose describing the retired rule, which is exactly what would
		// make this gate vacuous.
		expect(
			/match\s*\?\s*'font-semibold'/.test(strippedStreamRow),
			'expected no `segment.match ? \'font-semibold\' : ...` ternary anywhere in StreamRow.svelte — the retired Phase 3 weight-only match treatment (03-UI-SPEC.md) must be fully superseded by the shared amber class, not left dormant behind an unreachable branch'
		).toBe(false);
	});
});

describe('the query reaches the row', () => {
	it('SearchResults.svelte passes the live query down as the searchQuery prop', () => {
		expect(
			/<StreamRow[\s\S]*?searchQuery=\{query\}[\s\S]*?\/>/.test(strippedSearchResults),
			'expected SearchResults.svelte to pass searchQuery={query} to StreamRow — without this, a search-results row can never highlight its own title'
		).toBe(true);
	});

	it('StreamRow.svelte declares searchQuery with an empty-string default', () => {
		expect(
			/searchQuery\s*=\s*''/.test(strippedStreamRow),
			"expected StreamRow.svelte's props destructure to default searchQuery to '' — this is what keeps the unfiltered stream (which never passes the prop) byte-identical to before this plan"
		).toBe(true);
	});
});
