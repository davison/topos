// 08-04-PLAN.md Task 2's structural guard over RelinkModal.svelte (new)
// and the webspace route's Re-link… wiring (D-03): RelinkModal renders
// QRPanel and no connection/match form, handleChipEdit branches on the
// 'relink' kind before the describe path, the relinking-instance state is
// distinct from editInstance, and the route's RelinkModal render block is
// keyed on the relinking instance.
//
// House pattern (matches chip-edit-menu.test.ts / add-source.test.ts):
// comment-stripped source scanning, `extractBetween` scoping, a
// found-non-empty-source guard first, and one consequence-describing
// message per assertion.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const relinkModalPath = join(here, 'RelinkModal.svelte');
const routePath = join(here, '..', '..', 'routes', 'w', '[webspace]', '+page.svelte');

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

const relinkModalStripped = stripComments(readFileSync(relinkModalPath, 'utf-8'));
const routeStripped = stripComments(readFileSync(routePath, 'utf-8'));

describe('relink guard: found non-empty comment-stripped sources', () => {
	it('RelinkModal.svelte', () => {
		expect(relinkModalStripped.length).toBeGreaterThan(0);
	});
	it('the webspace route (+page.svelte)', () => {
		expect(routeStripped.length).toBeGreaterThan(0);
	});
});

describe('RelinkModal.svelte: renders QRPanel and no connection or match form', () => {
	it('renders <QRPanel', () => {
		expect(relinkModalStripped.includes('<QRPanel')).toBe(true);
	});

	it('renders no <ConnectionForm', () => {
		expect(
			relinkModalStripped.includes('<ConnectionForm'),
			'expected RelinkModal to carry no connection fields — it re-links an already-configured instance, it does not edit one'
		).toBe(false);
	});

	it('renders no <MatchFieldsForm', () => {
		expect(
			relinkModalStripped.includes('<MatchFieldsForm'),
			'expected RelinkModal to carry no match fields — re-linking changes nothing about match settings'
		).toBe(false);
	});

	it('carries no max-w-lg override on DialogContent — deliberately smaller than the Add-Source dialog', () => {
		const contentBlock = extractBetween(relinkModalStripped, '<DialogContent', '>');
		expect(
			contentBlock.includes('max-w-lg'),
			'expected DialogContent to carry no max-w-lg override — this dialog is sized to its own content, not the wider Add-Source Step 1 shape'
		).toBe(false);
	});
});

describe("the route's handleChipEdit branches on 'relink' before the describe path", () => {
	// 11-06-PLAN.md Task 1 widens handleChipEdit's kind parameter across two
	// lines (the union no longer fits one line) — the marker below matches
	// only up through the opening brace, which still uniquely locates this
	// function regardless of how its parameter list wraps.
	const handleChipEditBody = extractBetween(
		routeStripped,
		'async function handleChipEdit(',
		'\n\t}'
	);

	it("checks kind === 'relink' before any describePlugin( call", () => {
		const relinkIndex = handleChipEditBody.indexOf("kind === 'relink'");
		const describeIndex = handleChipEditBody.indexOf('describePlugin(');
		expect(relinkIndex, "expected handleChipEdit to check kind === 'relink'").toBeGreaterThanOrEqual(0);
		expect(describeIndex, 'expected handleChipEdit to still call describePlugin( for the match-edit path').toBeGreaterThanOrEqual(0);
		expect(
			relinkIndex,
			"expected the 'relink' branch to run strictly before the describePlugin( call — Re-link needs no describe round-trip and therefore no stale-response guard"
		).toBeLessThan(describeIndex);
	});

	it("the 'relink' branch assigns relinkInstance rather than editInstance", () => {
		const relinkBranch = extractBetween(handleChipEditBody, "kind === 'relink'", 'return;');
		expect(relinkBranch.includes('relinkInstance = name')).toBe(true);
		expect(
			relinkBranch.includes('editInstance = name'),
			"expected the 'relink' branch to never touch editInstance — the two sessions must never collide over the same shared state"
		).toBe(false);
	});
});

describe('relinkInstance is tracked in its own state, distinct from editInstance', () => {
	it('relinkInstance is declared as its own $state, not aliased to editInstance', () => {
		expect(
			routeStripped.includes('let relinkInstance = $state<string | null>(null);'),
			'expected relinkInstance to be its own independent $state<string | null> declaration'
		).toBe(true);
	});

	it('editInstance and relinkInstance are two distinct identifiers, each assigned independently', () => {
		expect(routeStripped.includes('editInstance = name;')).toBe(true);
		expect(routeStripped.includes('relinkInstance = name;')).toBe(true);
	});
});

describe("the route's RelinkModal render block is keyed on the relinking instance", () => {
	it('the {#if configResponse && relinkInstance} block wraps a {#key relinkInstance} around <RelinkModal', () => {
		const blockStart = routeStripped.indexOf('{#if configResponse && relinkInstance}');
		expect(blockStart, 'expected a {#if configResponse && relinkInstance} render guard').toBeGreaterThanOrEqual(0);
		const block = extractBetween(routeStripped, '{#if configResponse && relinkInstance}', '<RelinkModal');
		expect(
			block.includes('{#key relinkInstance}'),
			'expected the RelinkModal render block to be keyed on relinkInstance, matching the keyed EditSourceModal block\'s own discipline — a different chip always mounts a fresh session'
		).toBe(true);
	});

	it('RelinkModal is wired to handleRelinkClose/handleRelinked', () => {
		const renderBlock = extractBetween(routeStripped, '<RelinkModal', '/>');
		expect(renderBlock.includes('onclose={handleRelinkClose}')).toBe(true);
		expect(renderBlock.includes('onrelinked={handleRelinked}')).toBe(true);
	});
});
