// Source-scan guard over 09-UI-SPEC.md Fix 8: floating menus (dropdown /
// popover) must render on their own surface, visibly distinct from any pane
// behind them, while keeping their own border frame legible. No
// component-mount harness exists (web/vite.config.ts's test block:
// environment: 'node'), so this test reads app.css and the two menu
// components off disk instead, following scrollbar-theme.test.ts's
// readFileSync/dirname/join/fileURLToPath + comment-stripping pattern.
//
// The invariants being guarded (mirrors <behavior> in 09-03-PLAN.md Task 2):
//   - app.css declares --popover with a value distinct from --card at every
//     declaration site (the token is declared twice, :root and .dark);
//   - app.css's --popover value sits strictly between --card and --border in
//     lightness (relative luminance), the actual invariant Fix 8 needs — not
//     a pin to today's literal #172033, so this test survives a future
//     palette re-tune;
//   - dropdown-menu-content.svelte's and popover-content.svelte's surface
//     class is bg-popover, never bg-card;
//   - dialog-content.svelte is the explicit Fix 8 carve-out and must keep
//     bg-card — it already renders behind its own full-screen scrim;
//   - border-border is unchanged in both menu components.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));

const appCssPath = join(here, '..', 'app.css');
const dropdownMenuContentPath = join(
	here,
	'components',
	'ui',
	'dropdown-menu',
	'dropdown-menu-content.svelte'
);
const popoverContentPath = join(here, 'components', 'ui', 'popover', 'popover-content.svelte');
const dialogContentPath = join(here, 'components', 'ui', 'dialog', 'dialog-content.svelte');

const rawAppCss = readFileSync(appCssPath, 'utf-8');
const dropdownMenuContentSource = readFileSync(dropdownMenuContentPath, 'utf-8');
const popoverContentSource = readFileSync(popoverContentPath, 'utf-8');
const dialogContentSource = readFileSync(dialogContentPath, 'utf-8');

// app.css is heavily commented by house style — see scrollbar-theme.test.ts's
// identical rationale. Comments are replaced with a single space (not
// deleted outright) so two declarations separated only by a comment can
// never fuse into one token.
const strippedAppCss = rawAppCss.replace(/\/\*[\s\S]*?\*\//g, ' ');

/** Extracts every `--<name>: #rrggbb;` declaration's hex value, in source order. */
function extractHexDeclarations(source: string, name: string): string[] {
	const pattern = new RegExp(`--${name}:\\s*(#[0-9a-fA-F]{6})\\s*;`, 'g');
	const values: string[] = [];
	let match: RegExpExecArray | null;
	while ((match = pattern.exec(source)) !== null) {
		values.push(match[1].toLowerCase());
	}
	return values;
}

// WCAG relative luminance (sRGB -> linear -> weighted sum). Used only to
// compare lightness ordering between three known-neutral-ish dark tokens —
// not a contrast-ratio computation against text, just an ordering proof.
function relativeLuminance(hex: string): number {
	const r = parseInt(hex.slice(1, 3), 16) / 255;
	const g = parseInt(hex.slice(3, 5), 16) / 255;
	const b = parseInt(hex.slice(5, 7), 16) / 255;
	const linearize = (c: number) => (c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4));
	const [lr, lg, lb] = [linearize(r), linearize(g), linearize(b)];
	return 0.2126 * lr + 0.7152 * lg + 0.0722 * lb;
}

describe('popover-surface source-scan guard', () => {
	it('found non-empty sources to scan', () => {
		expect(strippedAppCss.length).toBeGreaterThan(0);
		expect(dropdownMenuContentSource.length).toBeGreaterThan(0);
		expect(popoverContentSource.length).toBeGreaterThan(0);
		expect(dialogContentSource.length).toBeGreaterThan(0);
	});

	it('declares --popover at every declaration site with a value distinct from --card', () => {
		const cardValues = extractHexDeclarations(strippedAppCss, 'card');
		const popoverValues = extractHexDeclarations(strippedAppCss, 'popover');

		expect(
			cardValues.length,
			'expected at least two --card declaration sites (:root and .dark)'
		).toBeGreaterThanOrEqual(2);
		expect(
			popoverValues.length,
			'expected --popover to be declared at the same number of sites as --card, so the two never disagree by cascade order'
		).toBe(cardValues.length);

		for (let i = 0; i < popoverValues.length; i++) {
			expect(
				popoverValues[i],
				`expected --popover declaration site #${i + 1} (${popoverValues[i]}) to differ from --card's own value (${cardValues[i]}) — a byte-identical value is the exact regression this token exists to fix`
			).not.toBe(cardValues[i]);
		}
	});

	it('--popover sits strictly between --card and --border in relative luminance at every site', () => {
		const cardValues = extractHexDeclarations(strippedAppCss, 'card');
		const popoverValues = extractHexDeclarations(strippedAppCss, 'popover');
		const borderValues = extractHexDeclarations(strippedAppCss, 'border');

		expect(borderValues.length).toBe(cardValues.length);
		expect(popoverValues.length).toBe(cardValues.length);

		for (let i = 0; i < cardValues.length; i++) {
			const cardLum = relativeLuminance(cardValues[i]);
			const popoverLum = relativeLuminance(popoverValues[i]);
			const borderLum = relativeLuminance(borderValues[i]);

			expect(
				popoverLum,
				`expected --popover (${popoverValues[i]}, luminance ${popoverLum}) to be lighter than --card (${cardValues[i]}, luminance ${cardLum}) at site #${i + 1} — a menu must read lighter than the pane behind it`
			).toBeGreaterThan(cardLum);
			expect(
				popoverLum,
				`expected --popover (${popoverValues[i]}, luminance ${popoverLum}) to be darker than --border (${borderValues[i]}, luminance ${borderLum}) at site #${i + 1} — otherwise the menu's own border frame would vanish into its own fill`
			).toBeLessThan(borderLum);
		}
	});

	it('dropdown-menu-content.svelte uses bg-popover, not bg-card, and keeps border-border', () => {
		expect(
			/\bbg-popover\b/.test(dropdownMenuContentSource),
			'expected dropdown-menu-content.svelte to declare bg-popover on its surface'
		).toBe(true);
		expect(
			/\bbg-card\b/.test(dropdownMenuContentSource),
			'expected dropdown-menu-content.svelte to no longer declare bg-card — that is the pane-collision defect Fix 8 exists to remove'
		).toBe(false);
		expect(
			/\bborder-border\b/.test(dropdownMenuContentSource),
			'expected dropdown-menu-content.svelte to keep border-border unchanged'
		).toBe(true);
	});

	it('popover-content.svelte uses bg-popover, not bg-card, and keeps border-border', () => {
		expect(
			/\bbg-popover\b/.test(popoverContentSource),
			'expected popover-content.svelte to declare bg-popover on its surface'
		).toBe(true);
		expect(
			/\bbg-card\b/.test(popoverContentSource),
			'expected popover-content.svelte to no longer declare bg-card — that is the pane-collision defect Fix 8 exists to remove'
		).toBe(false);
		expect(
			/\bborder-border\b/.test(popoverContentSource),
			'expected popover-content.svelte to keep border-border unchanged'
		).toBe(true);
	});

	it('dialog-content.svelte keeps bg-card — the explicit Fix 8 carve-out', () => {
		expect(
			/\bbg-card\b/.test(dialogContentSource),
			'expected dialog-content.svelte to still use bg-card: it already renders behind its own full-screen bg-background/80 scrim, which separates it from whatever pane sits behind it, so it is deliberately NOT changed by Fix 8'
		).toBe(true);
		expect(
			/\bbg-popover\b/.test(dialogContentSource),
			'dialog-content.svelte must not have picked up bg-popover — that carve-out is explicit, not accidental'
		).toBe(false);
	});
});
