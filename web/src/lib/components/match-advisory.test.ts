// 12-10-PLAN.md Task 1's guard over the G-12-1/G-12-3 gap closure: the
// zero-match advisory (kernel/httpapi/sources.go's last_notice, 12-09) now
// reaches SourceChip.svelte as a fifth tooltip branch and healthTone's own
// tone precedence chain. This file protects two things that must both hold
// at once and cannot be satisfied at each other's expense:
//
//  1. Behavioural — the SIX-case precedence matrix against the real
//     `healthTone`, including every "advisory coexists with a bigger
//     problem" combination (pin mismatch, unreachable, never-synced,
//     errored), proving the advisory never outranks a real problem.
//  2. Structural — the four PRE-EXISTING tooltip template literals stay
//     byte-for-byte (asserted here independently of
//     source-chip-tooltip.test.ts, so a future edit that satisfies one
//     guard by breaking the other cannot pass either), the new branch
//     exists ahead of the tone switch and is gated so a genuine error's
//     own copy is never displaced, no raw-HTML output directive exists
//     anywhere in the file, and the new branch does not repeat the
//     "${relative} ago" doubled-word defect the Fix 3 guard already
//     protects the other branches against.
//
// House pattern (matches repin.test.ts / source-chip-tooltip.test.ts):
// comment-stripped source scanning, a found-non-empty-source guard first,
// a local `makeSource` fixture modelled on format.test.ts's, and one
// consequence-describing message per assertion.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { healthTone } from '../format';
import type { SourceStatus } from '../api';

const here = dirname(fileURLToPath(import.meta.url));
const chipPath = join(here, 'SourceChip.svelte');
const rawChip = readFileSync(chipPath, 'utf-8');

function stripComments(source: string): string {
	return source
		.replace(/<!--[\s\S]*?-->/g, ' ')
		.replace(/\/\*[\s\S]*?\*\//g, ' ')
		.replace(/\/\/.*$/gm, ' ');
}

const strippedChip = stripComments(rawChip);

describe('match-advisory guard: found non-empty comment-stripped source', () => {
	it('SourceChip.svelte', () => {
		expect(strippedChip.length).toBeGreaterThan(0);
	});
});

// --- Behavioural: tone precedence against the real healthTone ---

function makeSource(overrides: Partial<SourceStatus> = {}): SourceStatus {
	return {
		name: 'files',
		source_type: 'filesystem',
		display_name: 'Household Docs',
		plugin: 'topos-plugin-filesystem',
		tier: 'trusted',
		reachable: true,
		syncing: false,
		last_status: 'ok',
		last_sync_unix: 1700000000,
		last_error: '',
		...overrides
	};
}

const NOTICE = "webspace 'test': match value 'files.folders=*' matched no items";

describe('healthTone: a source carrying a zero-match advisory', () => {
	it('reachable + last_status ok + non-empty last_notice -> warning (the defect this plan closes)', () => {
		expect(healthTone(makeSource({ last_notice: NOTICE }))).toBe('warning');
	});

	it('reachable + last_status ok + last_notice ABSENT (undefined) -> success, never throws', () => {
		const source = makeSource();
		delete (source as Partial<SourceStatus>).last_notice;
		expect(healthTone(source)).toBe('success');
	});

	it('reachable + last_status ok + last_notice EMPTY STRING -> success', () => {
		expect(healthTone(makeSource({ last_notice: '' }))).toBe('success');
	});

	it('reachable + last_status error + non-empty last_notice -> warning (an advisory never displaces the warning tone the error already gets)', () => {
		expect(healthTone(makeSource({ last_status: 'error', last_notice: NOTICE }))).toBe('warning');
	});

	it('reachable: false + non-empty last_notice -> destructive (a real unreachable problem outranks the advisory)', () => {
		expect(healthTone(makeSource({ reachable: false, last_notice: NOTICE }))).toBe('destructive');
	});

	it('last_status "" (never synced) + non-empty last_notice -> unknown (a source with no sync history outranks the advisory)', () => {
		expect(healthTone(makeSource({ last_status: '', last_sync_unix: 0, last_notice: NOTICE }))).toBe(
			'unknown'
		);
	});

	it('launch_failure pin_mismatch + non-empty last_notice -> destructive (a pin mismatch outranks the advisory)', () => {
		expect(
			healthTone(makeSource({ launch_failure: 'pin_mismatch', last_notice: NOTICE }))
		).toBe('destructive');
	});
});

// --- Structural: the four pre-existing tooltip branches are untouched ---

describe('tooltipText structure: the four pre-existing branches stay byte-for-byte (Fix 3 Copywriting Contract, asserted independently of source-chip-tooltip.test.ts)', () => {
	it('success branch is byte-identical', () => {
		expect(
			strippedChip.includes('return `${source.display_name} — synced ${relative}`;'),
			'expected the success branch to remain exactly "{display_name} — synced {relative}"'
		).toBe(true);
	});

	it('warning branch is byte-identical', () => {
		expect(
			strippedChip.includes(
				'return `${source.display_name} — last error ${relative}: ${source.last_error}`;'
			),
			'expected the warning branch to remain exactly "{display_name} — last error {relative}: {last_error}"'
		).toBe(true);
	});

	it('destructive branch is byte-identical', () => {
		expect(
			strippedChip.includes('return `${source.display_name} — unreachable since ${relative}`;'),
			'expected the destructive branch to remain byte-identical'
		).toBe(true);
	});

	it('unknown/default branch is byte-identical', () => {
		expect(
			strippedChip.includes('return `${source.display_name} — not yet synced`;'),
			'expected the unknown/default branch to remain byte-identical'
		).toBe(true);
	});
});

describe('tooltipText structure: the new advisory branch', () => {
	it('a branch referencing last_notice (via the advisory derived value) exists strictly before the tone switch', () => {
		const advisoryIdx = strippedChip.indexOf("advisory !== '' && source.last_status !== 'error'");
		const switchIdx = strippedChip.indexOf('switch (tone)');
		expect(advisoryIdx, 'expected an advisory-gated branch in tooltipText').toBeGreaterThanOrEqual(
			0
		);
		expect(switchIdx, 'expected the pre-existing tone switch to still exist').toBeGreaterThanOrEqual(
			0
		);
		expect(
			advisoryIdx,
			'expected the advisory branch to run BEFORE the tone switch, mirroring isPinMismatch\'s placement'
		).toBeLessThan(switchIdx);
	});

	it("the advisory derived value is keyed on source.last_notice, never on a last_error string match (T-11-32 discipline)", () => {
		expect(strippedChip.includes('let advisory = $derived((source.last_notice ?? \'\').trim());')).toBe(
			true
		);
	});

	it('the advisory branch is gated so an errored last_status keeps its own warning-branch copy — an advisory never displaces a real error', () => {
		const guardIdx = strippedChip.indexOf("if (advisory !== '' && source.last_status !== 'error')");
		expect(guardIdx, 'expected the advisory branch guard to exclude last_status === "error"').toBeGreaterThanOrEqual(
			0
		);
	});

	it('the advisory branch composes display name, synced-relative phrase, and advisory text with the same em-dash separator every other branch uses', () => {
		expect(
			strippedChip.includes(
				'return `${source.display_name} — synced ${relative} — ${advisory}`;'
			)
		).toBe(true);
	});

	it('does not place the word "ago" after ${relative} anywhere in the file (the general defect class Fix 3 guards against, re-checked for the new branch)', () => {
		expect(
			/\$\{relative\}\s*ago/.test(strippedChip),
			'found "${relative} ago" in the source — formatRelativeTime already returns a complete phrase'
		).toBe(false);
	});
});

describe('SourceChip.svelte: the advisory is rendered as escaped text, never as markup', () => {
	it('contains no raw-HTML output directive anywhere in this file', () => {
		expect(
			strippedChip.includes('{@html'),
			'expected no {@html} directive — the advisory carries operator-authored configuration values and must be interpolated as text only'
		).toBe(false);
	});
});
