// House pattern (matches stream-row-plugin-icon.test.ts / search-emphasis
// .test.ts): comment-stripped source scanning — this runner is
// `environment: 'node'` (web/vite.config.ts), with no jsdom/
// @testing-library/svelte harness, and 13-03-PLAN.md's own Task 1 action
// explicitly says not to add one for this plan alone. Structural proof off
// disk is therefore the available instrument for asserting the pure label
// composition (13-UI-SPEC.md E2's Copywriting Contract: `{N} selected`,
// `Exclude`/`Include`, `Clear` — all fixed-shape strings whose only
// variable is the digit count of N) and the count-gated render/absence
// rule.
import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const componentPath = join(here, 'SelectionActionBar.svelte');
const raw = readFileSync(componentPath, 'utf-8');

function stripComments(source: string): string {
	return source
		.replace(/<!--[\s\S]*?-->/g, ' ')
		.replace(/\/\*[\s\S]*?\*\//g, ' ')
		.replace(/\/\/.*$/gm, ' ');
}

const stripped = stripComments(raw);

describe('SelectionActionBar.svelte guard: found non-empty comment-stripped source', () => {
	it('SelectionActionBar.svelte', () => {
		expect(stripped.length).toBeGreaterThan(0);
	});
});

describe('SelectionActionBar.svelte declares the props 13-03-PLAN.md Task 1 specifies', () => {
	it('destructures count, primaryLabel, busy, onprimary, onclear from $props', () => {
		expect(
			/let\s*\{\s*count,\s*primaryLabel,\s*busy,\s*onprimary,\s*onclear\s*\}/.test(stripped),
			'expected the $props() destructure to name exactly count/primaryLabel/busy/onprimary/onclear, in that order'
		).toBe(true);
	});
});

describe('the count label is the one fixed-shape template — correct for a count of 1 and a count of 5 alike, since it never special-cases pluralization', () => {
	it('renders the exact contract string "{count} selected" with no other text interpolated into that span', () => {
		expect(
			/\{count\}\s*selected</.test(stripped),
			'expected the template to render `{count} selected` verbatim — the Copywriting Contract\'s count label has no plural form, so the same template is correct whether count is 1, 5, or any other digit count'
		).toBe(true);
	});

	it('never special-cases singular vs. plural for the count label (no itemNoun-style helper, unlike toast.ts)', () => {
		expect(
			/count\s*===\s*1/.test(stripped),
			'expected no singular/plural branch on count — "{N} selected" reads correctly for every N with no pluralization helper'
		).toBe(false);
	});
});

describe('the bar renders nothing at count 0 — appearing/disappearing is the whole selection-mode signal (D-01)', () => {
	it('the entire markup is gated behind {#if count > 0}', () => {
		expect(
			/\{#if\s+count\s*>\s*0\s*\}/.test(stripped),
			'expected the whole bar to be wrapped in {#if count > 0} — a zero count must render nothing, not a disabled/greyed variant'
		).toBe(true);
	});
});

describe('primaryLabel drives the primary button verbatim — Exclude in the normal stream, Include in the excluded view', () => {
	it('the primary Button renders {primaryLabel}, never a hardcoded "Exclude" or "Include" string', () => {
		expect(
			/<Button\s+size="sm"\s+disabled=\{busy\}\s+onclick=\{onprimary\}>\{primaryLabel\}<\/Button>/.test(
				stripped
			),
			'expected the primary action button to render {primaryLabel} literally, so the caller\'s own view state (normal stream vs. excluded bucket) is the only thing that ever changes its copy'
		).toBe(true);
	});
});

describe('button variants — Exclude/Include is never destructive-styled (13-UI-SPEC.md Color rationale)', () => {
	it('the primary button carries no variant="destructive" attribute', () => {
		expect(
			/<Button[^>]*variant="destructive"/.test(stripped),
			'expected no Button anywhere in this component to carry variant="destructive" — Exclude/Include is instant and trivially reversible (undo toast + the excluded view), so it must never read as more dangerous than it is'
		).toBe(false);
	});

	it('the Clear button carries variant="ghost"', () => {
		expect(
			/<Button\s+variant="ghost"\s+size="sm"\s+disabled=\{busy\}\s+onclick=\{onclear\}>Clear<\/Button>/.test(
				stripped
			),
			'expected the secondary Clear button to use variant="ghost", matching the plan\'s own composition spec'
		).toBe(true);
	});
});

describe('both buttons disable for the duration of an in-flight write (prevents a double-fire double-submit)', () => {
	it('the primary button carries disabled={busy}', () => {
		expect(/<Button\s+size="sm"\s+disabled=\{busy\}/.test(stripped)).toBe(true);
	});

	it('the Clear button carries disabled={busy}', () => {
		expect(/variant="ghost"\s+size="sm"\s+disabled=\{busy\}/.test(stripped)).toBe(true);
	});
});
