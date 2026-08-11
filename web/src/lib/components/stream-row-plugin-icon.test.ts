// Checkpoint-directed scope addition (09-02-PLAN.md Task 4 checkpoint
// feedback, Item 2): plugin identity icons must appear on each stream/
// search-result row, not only on the source chip, so a user can tell at a
// glance which source an item came from in a mixed pane. Reuses
// PluginIcon.svelte's existing kernel-served fallback chain unchanged —
// same endpoint, same Puzzle fallback, same fixed-size no-layout-shift box
// — as additive metadata alongside the row's existing Thumbnail, which
// this file also guards as unchanged (the user explicitly asked to keep
// the PDF/document thumbnails exactly as they are).
//
// House pattern (matches qr-panel.test.ts / search-emphasis.test.ts):
// comment-stripped source scanning, `extractBetween` scoping, a
// found-non-empty-source guard first, and one consequence-describing
// message per assertion. No component-mount harness is used — this
// runner is `environment: 'node'` (web/vite.config.ts) — so structural
// proof off disk is the available instrument, the same precedent
// search-emphasis.test.ts and qr-panel.test.ts already establish.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));

const streamRowPath = join(here, 'StreamRow.svelte');
const streamListPath = join(here, 'StreamList.svelte');
const searchResultsPath = join(here, 'SearchResults.svelte');

const rawStreamRow = readFileSync(streamRowPath, 'utf-8');
const rawStreamList = readFileSync(streamListPath, 'utf-8');
const rawSearchResults = readFileSync(searchResultsPath, 'utf-8');

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

const strippedStreamRow = stripComments(rawStreamRow);
const strippedStreamList = stripComments(rawStreamList);
const strippedSearchResults = stripComments(rawSearchResults);

describe('stream-row plugin icon guard: found non-empty comment-stripped sources', () => {
	it('StreamRow.svelte', () => {
		expect(strippedStreamRow.length).toBeGreaterThan(0);
	});
	it('StreamList.svelte', () => {
		expect(strippedStreamList.length).toBeGreaterThan(0);
	});
	it('SearchResults.svelte', () => {
		expect(strippedSearchResults.length).toBeGreaterThan(0);
	});
});

describe('StreamRow.svelte reuses PluginIcon, never a second icon path', () => {
	it("imports PluginIcon from './PluginIcon.svelte' — the same component SourceChip.svelte already uses", () => {
		expect(
			/import\s+PluginIcon\s+from\s+'\.\/PluginIcon\.svelte'/.test(strippedStreamRow),
			'expected StreamRow.svelte to import the existing PluginIcon component rather than inventing a second icon-rendering path'
		).toBe(true);
	});

	it("declares a 'plugin' prop defaulting to '' — PluginIcon's own empty-plugin branch already resolves to the Puzzle fallback", () => {
		expect(
			/plugin\s*=\s*''/.test(strippedStreamRow),
			"expected StreamRow.svelte's props destructure to default plugin to '' — this is what keeps a caller that hasn't threaded the prop (or a historic item whose source instance was removed from config) falling back to Puzzle rather than an empty box"
		).toBe(true);
	});

	it('renders <PluginIcon {plugin} ...> inside the metadata strip (.stream-row-meta), sized size-3.5 like the chip', () => {
		const metaBlock = extractBetween(strippedStreamRow, 'stream-row-meta', '</div>');
		expect(
			/<PluginIcon\s+\{plugin\}\s+size="size-3\.5"/.test(metaBlock),
			'expected the metadata strip to render <PluginIcon {plugin} size="size-3.5" /> — the same size-3.5 the source chip already uses for this component'
		).toBe(true);
	});

	it('the plugin icon renders with a decorative role — no accessible-name query is required to find it', () => {
		// PluginIcon.svelte itself owns alt="" on the real <img> and the
		// Puzzle fallback's aria-hidden-by-default Lucide rendering; this
		// row only wraps it in a plain <span title=...> for hover
		// discoverability, never a second alt/aria-label that could
		// conflict with PluginIcon's own contract.
		const metaBlock = extractBetween(strippedStreamRow, 'stream-row-meta', '</div>');
		const iconWrapper = extractBetween(metaBlock, '<span class="shrink-0"', '</span>');
		expect(
			iconWrapper.includes('<PluginIcon'),
			'expected the shrink-0 wrapper span immediately preceding the sender/date line to contain the PluginIcon element'
		).toBe(true);
	});
});

describe('Thumbnail is unchanged — the source icon is additive metadata, never a replacement (checkpoint constraint)', () => {
	it('StreamRow.svelte still renders <Thumbnail {item} /> verbatim', () => {
		expect(
			strippedStreamRow.includes('<Thumbnail {item} />'),
			"expected StreamRow.svelte's leading Thumbnail element to be byte-identical to before this checkpoint follow-up — the user explicitly asked to keep PDF/document thumbnails exactly as they are; the source icon is new metadata in the row body, not a change to the thumbnail"
		).toBe(true);
	});

	it('the PluginIcon element sits inside the metadata strip, not inside the Thumbnail component call', () => {
		const thumbnailToMeta = extractBetween(
			strippedStreamRow,
			'<Thumbnail {item} />',
			'stream-row-meta'
		);
		expect(
			thumbnailToMeta.includes('<PluginIcon'),
			'expected no PluginIcon reference between the Thumbnail element and the metadata strip opening — the icon is additive inside stream-row-meta, structurally separate from the thumbnail'
		).toBe(false);
	});
});

describe('the plugin binary name reaches the row from the already-fetched sources list (no new kernel field needed)', () => {
	it('StreamList.svelte passes plugin={sourcesByInstance.get(item.source)?.plugin ?? \'\'} to StreamRow', () => {
		expect(
			/<StreamRow[\s\S]*?plugin=\{sourcesByInstance\.get\(item\.source\)\?\.plugin\s*\?\?\s*''\}[\s\S]*?\/>/.test(
				strippedStreamList
			),
			"expected StreamList.svelte to resolve each row's plugin binary name from sourcesByInstance (GET /api/sources, which already carries SourceStatus.plugin since 09-01) — the identical map sourceDisplayName already resolves from, so no kernel/proto change is needed"
		).toBe(true);
	});

	it('SearchResults.svelte passes plugin={sourcesByInstance.get(result.source)?.plugin ?? \'\'} to StreamRow', () => {
		expect(
			/<StreamRow[\s\S]*?plugin=\{sourcesByInstance\.get\(result\.source\)\?\.plugin\s*\?\?\s*''\}[\s\S]*?\/>/.test(
				strippedSearchResults
			),
			'expected SearchResults.svelte to resolve each result row\'s plugin binary name the same way StreamList.svelte does, so a mixed search-results pane is also scannable by source icon — search deliberately spans every source in the webspace'
		).toBe(true);
	});
});
