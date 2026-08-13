// 11-01-PLAN.md Task 2's guard over TrustBadge.svelte's markup contract
// (11-UI-SPEC.md E2) and its SourceChip.svelte call site: the badge
// renders nothing at all for a trusted-tier source, the exact
// CircleAlert/backdrop shape at both declared scales, and the tooltip's
// untrusted clause — all independent of (never replacing) the chip's
// four pre-existing sync-state branches.
//
// House pattern (matches plugin-icon.test.ts / source-chip-pill.test.ts):
// comment-stripped source scanning (this repo's vitest config runs
// environment: 'node' with no component-mount harness), `extractBetween`
// scoping, a found-non-empty-source guard first, and one
// consequence-describing message per assertion.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));

const badgePath = join(here, 'TrustBadge.svelte');
const chipPath = join(here, 'SourceChip.svelte');

const rawBadge = readFileSync(badgePath, 'utf-8');
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

const strippedBadge = stripComments(rawBadge);
const strippedChip = stripComments(rawChip);

describe('trust-badge guard: found non-empty comment-stripped sources', () => {
	it('TrustBadge.svelte', () => {
		expect(strippedBadge.length).toBeGreaterThan(0);
	});
	it('SourceChip.svelte', () => {
		expect(strippedChip.length).toBeGreaterThan(0);
	});
});

describe('TrustBadge: renders nothing at all when tier is not external', () => {
	it('gates the whole backdrop/glyph block on tier === "external"', () => {
		expect(
			/\{#if\s+tier\s*===\s*'external'\}/.test(strippedBadge),
			'expected an {#if tier === \'external\'} branch gating the badge — a trusted-tier source must render nothing extra at all, not merely a hidden or empty element'
		).toBe(true);
	});
});

describe('TrustBadge: the badge markup carries the full E2 contract', () => {
	it('imports CircleAlert from @lucide/svelte/icons/circle-alert — the same glyph SecretField.svelte already established', () => {
		expect(
			strippedBadge.includes("@lucide/svelte/icons/circle-alert"),
			'expected TrustBadge.svelte to import CircleAlert from @lucide/svelte/icons/circle-alert'
		).toBe(true);
	});

	it('positions the backdrop absolutely at -bottom-1 -right-1', () => {
		expect(
			/-bottom-1/.test(strippedBadge) && /-right-1/.test(strippedBadge),
			'expected the badge backdrop to carry -bottom-1 -right-1 (overlapping the wrapped icon\'s bottom-right corner)'
		).toBe(true);
	});

	it('the backdrop carries rounded-full and aria-hidden="true"', () => {
		const backdropTag = extractBetween(strippedBadge, '<span\n\t\t\tclass=', 'aria-hidden="true"');
		expect(
			/rounded-full/.test(backdropTag),
			'expected the badge backdrop to carry rounded-full'
		).toBe(true);
		expect(
			strippedBadge.includes('aria-hidden="true"'),
			'expected the badge backdrop to carry aria-hidden="true" — it is redundant with the adjacent text label and the chip tooltip, and must never be the only channel conveying "untrusted" to an assistive-tech user'
		).toBe(true);
	});

	it('the outer wrapper carries no size/color of its own beyond relative positioning (relative inline-flex shrink-0)', () => {
		expect(
			strippedBadge.includes('relative inline-flex shrink-0'),
			'expected the badge\'s outer wrapper to carry "relative inline-flex shrink-0" — the positioning anchor for the absolutely-positioned backdrop'
		).toBe(true);
	});
});

describe('TrustBadge: declares two scales with distinct backdrop/glyph sizes and surfaces', () => {
	it('chip scale: size-3.5 backdrop, bg-card surface, size-2.5 glyph', () => {
		expect(
			/'chip'\s*\?\s*'size-3\.5'/.test(strippedBadge),
			'expected the chip-scale backdrop size to be size-3.5'
		).toBe(true);
		expect(
			/'chip'\s*\?\s*'bg-card'/.test(strippedBadge),
			'expected the chip-scale backdrop surface to be bg-card — matching the chip\'s own bg-card surface'
		).toBe(true);
		expect(
			/'chip'\s*\?\s*'size-2\.5'/.test(strippedBadge),
			'expected the chip-scale glyph size to be size-2.5'
		).toBe(true);
	});

	it('picker scale: size-4 backdrop, bg-popover surface, size-3 glyph', () => {
		expect(
			/:\s*'size-4'/.test(strippedBadge),
			'expected the picker-scale backdrop size to be size-4'
		).toBe(true);
		expect(
			/:\s*'bg-popover'/.test(strippedBadge),
			'expected the picker-scale backdrop surface to be bg-popover — matching PopoverContent\'s own surface token'
		).toBe(true);
		expect(
			/:\s*'size-3'/.test(strippedBadge),
			'expected the picker-scale glyph size to be size-3'
		).toBe(true);
	});

	it('the glyph carries text-warning, never text-primary — the trust signal is never the accent colour', () => {
		expect(
			strippedBadge.includes('text-warning'),
			'expected the CircleAlert glyph to carry text-warning'
		).toBe(true);
		expect(
			/CircleAlert[^}]*text-primary\b/.test(strippedBadge),
			'found text-primary anywhere near the CircleAlert glyph — accent stays reserved for affirmative/navigational actions, never the trust-warning signal'
		).toBe(false);
	});
});

describe('SourceChip: wraps PluginIcon in TrustBadge without changing the pill', () => {
	it('imports TrustBadge', () => {
		expect(
			/import TrustBadge from ['"]\$lib\/components\/TrustBadge\.svelte['"];/.test(strippedChip),
			'expected SourceChip.svelte to import TrustBadge'
		).toBe(true);
	});

	it('passes tier={source.tier} and scale="chip" to TrustBadge, wrapping PluginIcon', () => {
		const badgeUsage = extractBetween(strippedChip, '<TrustBadge', '</TrustBadge>');
		expect(
			/tier=\{source\.tier\}/.test(badgeUsage),
			'expected <TrustBadge> to be called with tier={source.tier}'
		).toBe(true);
		expect(
			/scale="chip"/.test(badgeUsage),
			'expected the chip call site to pass scale="chip"'
		).toBe(true);
		expect(
			badgeUsage.includes('<PluginIcon'),
			'expected <PluginIcon> to be nested inside the <TrustBadge> usage'
		).toBe(true);
	});

	it('the outer pill still carries h-11 and gains no tier-conditional class', () => {
		const wrapperTag = extractBetween(strippedChip, '<div', '>');
		expect(
			/\bh-11\b/.test(wrapperTag),
			'expected the chip wrapper to still declare h-11 — D-06 forbids widening the pill for the trust badge'
		).toBe(true);
		expect(
			/source\.tier/.test(wrapperTag),
			'found a source.tier reference on the pill wrapper\'s own opening tag — the trust badge must never change the pill\'s own border/background/height (D-06); tier only ever gates the icon-corner overlay'
		).toBe(false);
	});

	it('the wrapper still carries border-border and bg-card, unconditionally', () => {
		const wrapperTag = extractBetween(strippedChip, '<div', '>');
		expect(
			/border-border/.test(wrapperTag) && /bg-card/.test(wrapperTag),
			'expected the pill wrapper to still carry border-border and bg-card — byte-identical to a trusted-tier chip'
		).toBe(true);
	});
});

describe('SourceChip: tooltip text gains an untrusted clause for external-tier sources', () => {
	it('appends " — untrusted external plugin" for tier === "external", independent of the sync-state branches', () => {
		expect(
			strippedChip.includes('— untrusted external plugin'),
			'expected the tooltip derivation to append " — untrusted external plugin" for an external-tier source'
		).toBe(true);
		expect(
			/source\.tier\s*===\s*'external'/.test(strippedChip),
			'expected the tooltip derivation to check source.tier === \'external\''
		).toBe(true);
	});

	it('the four base sync-state template strings remain byte-identical (D-06: a trusted chip is unchanged)', () => {
		expect(
			strippedChip.includes('return `${source.display_name} — synced ${relative}`;'),
			'expected the success branch to remain byte-identical'
		).toBe(true);
		expect(
			strippedChip.includes(
				'return `${source.display_name} — last error ${relative}: ${source.last_error}`;'
			),
			'expected the warning branch to remain byte-identical'
		).toBe(true);
		expect(
			strippedChip.includes('return `${source.display_name} — unreachable since ${relative}`;'),
			'expected the destructive branch to remain byte-identical'
		).toBe(true);
		expect(
			strippedChip.includes('return `${source.display_name} — not yet synced`;'),
			'expected the unknown/default branch to remain byte-identical'
		).toBe(true);
	});
});
