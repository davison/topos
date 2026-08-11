// 09-01-PLAN.md Task 3's structural guard over PluginIcon.svelte's
// mandatory three-step fallback chain (09-UI-SPEC.md Fix 10) and its
// SourceChip.svelte call site's ordering — [dot][icon][name].
//
// House pattern (matches source-chip-pill.test.ts / chip-edit-menu.test.ts):
// comment-stripped source scanning, `extractBetween` scoping, a
// found-non-empty-source guard first, and one consequence-describing
// message per assertion. PluginIcon's contract is entirely structural
// (which branch renders which element, which attributes/classes land on
// the <img>) — no component-mount harness exists in this repo's vitest
// config (environment: 'node'), so every claim below is a source-scan
// assertion, same as every other component-structure guard in this
// directory.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));

const iconPath = join(here, 'PluginIcon.svelte');
const chipPath = join(here, 'SourceChip.svelte');

const rawIcon = readFileSync(iconPath, 'utf-8');
const rawChip = readFileSync(chipPath, 'utf-8');

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

const strippedIcon = stripComments(rawIcon);
const strippedChip = stripComments(rawChip);

describe('plugin-icon guard: found non-empty comment-stripped sources', () => {
	it('PluginIcon.svelte', () => {
		expect(strippedIcon.length).toBeGreaterThan(0);
	});
	it('SourceChip.svelte', () => {
		expect(strippedChip.length).toBeGreaterThan(0);
	});
});

const imgTag = extractBetween(strippedIcon, '<img', '/>');

describe('PluginIcon: an empty or unknown plugin renders Puzzle, not an <img>', () => {
	it('the component branches on a truthy plugin name before rendering the <img>', () => {
		expect(
			/\{#if\s+showImage\}/.test(strippedIcon) || /\{#if[^}]*plugin[^}]*\}/.test(strippedIcon),
			'expected an {#if} branch gating the <img> render on plugin being present — an empty/unknown plugin binary name must render Puzzle instead, never an <img> with an empty src'
		).toBe(true);
	});
	it('imports the Lucide Puzzle glyph as the fallback', () => {
		expect(
			strippedIcon.includes("@lucide/svelte/icons/puzzle"),
			'expected PluginIcon.svelte to import Puzzle from @lucide/svelte/icons/puzzle — the one fallback glyph every failure path must terminate in'
		).toBe(true);
	});
});

describe('PluginIcon: the <img> element carries the mandatory decorative/no-shift/no-color contract', () => {
	it('carries alt="" (decorative — the adjacent text label carries the name)', () => {
		expect(
			/alt=""/.test(imgTag),
			'expected the <img> to carry alt="" — the icon is decorative, the adjacent text label already carries the plugin name'
		).toBe(true);
	});
	it('carries object-contain so the icon never distorts inside its fixed box', () => {
		expect(
			/object-contain/.test(imgTag),
			'expected the <img> to carry object-contain'
		).toBe(true);
	});
	it('has an onerror handler that swaps to the Puzzle fallback', () => {
		expect(
			/onerror=/.test(imgTag),
			"expected the <img> to carry an onerror handler — this is the only path that can catch a kernel 404 or malformed bytes; without it those cases render a broken-image glyph instead of Puzzle"
		).toBe(true);
	});
	it('sources from the one kernel plugin-icon route, with no other api/plugins/ reference in the file', () => {
		const matches = strippedIcon.match(/api\/plugins\//g) ?? [];
		expect(
			matches.length,
			'expected exactly one reference to the kernel icon route — a second reference would mean a second, divergent rendering path'
		).toBe(1);
		expect(
			/src=\{`\/api\/plugins\/\$\{plugin\}\/icon`\}/.test(strippedIcon),
			'expected the <img> src to be built from the plugin prop against /api/plugins/{plugin}/icon'
		).toBe(true);
	});
	it('carries no text-colour utility class — an <img> cannot inherit currentColor, so baking one in here would be dead weight at best and misleading at worst', () => {
		expect(
			/text-(muted-foreground|foreground|primary|destructive|success|warning)/.test(imgTag),
			'found a text-colour utility class on the <img> — colour is baked into the served bytes (09-UI-SPEC.md "baked color, not currentColor"), so a text-* class here does nothing and should not exist'
		).toBe(false);
	});
});

const puzzleUsage = extractBetween(strippedIcon, '<Puzzle', '/>');

describe('PluginIcon: the Puzzle fallback carries text-muted-foreground', () => {
	it('Puzzle is rendered with text-muted-foreground — unlike the <img>, a live Lucide component still inherits CSS colour', () => {
		expect(
			/text-muted-foreground/.test(puzzleUsage),
			'expected the rendered <Puzzle> element to carry text-muted-foreground'
		).toBe(true);
	});
});

describe('SourceChip: renders PluginIcon between the health dot and the display name', () => {
	it('imports PluginIcon', () => {
		expect(
			strippedChip.includes("import PluginIcon from '$lib/components/PluginIcon.svelte';") ||
				/import PluginIcon from ['"]\$lib\/components\/PluginIcon\.svelte['"];/.test(
					strippedChip
				),
			'expected SourceChip.svelte to import PluginIcon'
		).toBe(true);
	});

	it('places <PluginIcon ...> after the health-dot span and before the display-name span', () => {
		const dotIndex = strippedChip.indexOf('rounded-full');
		const iconIndex = strippedChip.indexOf('<PluginIcon');
		const nameIndex = strippedChip.indexOf('source.display_name}</span');
		expect(dotIndex, 'expected to find the health dot span').toBeGreaterThanOrEqual(0);
		expect(iconIndex, 'expected to find a <PluginIcon usage').toBeGreaterThanOrEqual(0);
		expect(nameIndex, 'expected to find the display-name span').toBeGreaterThanOrEqual(0);
		expect(
			dotIndex < iconIndex,
			'expected <PluginIcon> to render after the health dot — chip order must read [dot][icon][name]'
		).toBe(true);
		expect(
			iconIndex < nameIndex,
			'expected <PluginIcon> to render before the display-name span — chip order must read [dot][icon][name]'
		).toBe(true);
	});

	it('passes plugin={source.plugin} and size-3.5', () => {
		const usage = extractBetween(strippedChip, '<PluginIcon', '/>');
		expect(
			/plugin=\{source\.plugin\}/.test(usage),
			'expected <PluginIcon> to be called with plugin={source.plugin}'
		).toBe(true);
		expect(/size-3\.5/.test(usage), 'expected the chip usage to pass the size-3.5 size class').toBe(
			true
		);
	});
});
