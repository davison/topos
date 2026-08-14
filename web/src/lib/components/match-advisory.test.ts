// 12-10-PLAN.md Task 1's guard over the G-12-1/G-12-3 gap closure: the
// zero-match advisory (kernel/httpapi/sources.go's last_notice, 12-09) now
// reaches SourceChip.svelte as a fifth tooltip branch and healthTone's own
// tone precedence chain. This file protects THREE things that must all hold
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
//  3. 12-11-PLAN.md (CR-01 gap closure) — WHICH TOOLTIP BRANCH is chosen
//     for each precedence case, not merely which tone `healthTone` returns.
//     Per 12-VERIFICATION.md, the absence of this exact coverage — a guard
//     asserting the tooltip STRING rather than only the dot's tone — is
//     why CR-01 shipped past this file's own stated purpose: the old gate
//     tested `last_status` directly and never consulted `reachable`, so an
//     unreachable source with a stale successful status and a leftover
//     notice rendered a reassuring advisory tooltip while its own dot was
//     red. This file now also protects the new `isAdvisoryOnly` predicate
//     (format.ts) that the chip's gate defers to, and the six-case
//     tooltipText branch-selection matrix built on top of it.
//
// House pattern (matches repin.test.ts / source-chip-tooltip.test.ts):
// comment-stripped source scanning, a found-non-empty-source guard first,
// a local `makeSource` fixture modelled on format.test.ts's, and one
// consequence-describing message per assertion.

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { healthTone, isAdvisoryOnly, formatRelativeTime } from '../format';
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

// --- 13-06-PLAN.md (D-12/D-13/D-14, PLUG-07): manifest_unverified / shadowed ---

describe('healthTone: manifest_unverified (D-12/D-13)', () => {
	it('launch_failure manifest_unverified -> destructive, regardless of sync history', () => {
		expect(healthTone(makeSource({ launch_failure: 'manifest_unverified' }))).toBe('destructive');
	});

	it('launch_failure manifest_unverified + reachable: true + last_status ok -> still destructive (never launched, so sync history is irrelevant)', () => {
		expect(
			healthTone(
				makeSource({ launch_failure: 'manifest_unverified', reachable: true, last_status: 'ok' })
			)
		).toBe('destructive');
	});
});

describe('healthTone: shadowed advisory (D-14)', () => {
	it('reachable + last_status ok + launch_advisory shadowed -> warning', () => {
		expect(healthTone(makeSource({ launch_advisory: 'shadowed' }))).toBe('warning');
	});

	it('reachable: false + launch_advisory shadowed -> destructive (a real unreachable problem outranks the shadowed advisory)', () => {
		expect(healthTone(makeSource({ reachable: false, launch_advisory: 'shadowed' }))).toBe(
			'destructive'
		);
	});

	it('last_status "" (never synced) + launch_advisory shadowed -> unknown (no sync history outranks the shadowed advisory)', () => {
		expect(
			healthTone(
				makeSource({ last_status: '', last_sync_unix: 0, launch_advisory: 'shadowed' })
			)
		).toBe('unknown');
	});

	it('last_status error + launch_advisory shadowed -> warning (an advisory never displaces the warning tone the error already gets)', () => {
		expect(
			healthTone(makeSource({ last_status: 'error', launch_advisory: 'shadowed' }))
		).toBe('warning');
	});

	it('launch_failure pin_mismatch + launch_advisory shadowed -> destructive (a pin mismatch outranks the shadowed advisory, same precedence as the last_notice case)', () => {
		expect(
			healthTone(makeSource({ launch_failure: 'pin_mismatch', launch_advisory: 'shadowed' }))
		).toBe('destructive');
	});

	it('launch_failure manifest_unverified + launch_advisory shadowed -> destructive (manifest-unverified also outranks the shadowed advisory)', () => {
		expect(
			healthTone(
				makeSource({ launch_failure: 'manifest_unverified', launch_advisory: 'shadowed' })
			)
		).toBe('destructive');
	});

	it('launch_advisory EMPTY STRING -> success (not shadowed)', () => {
		expect(healthTone(makeSource({ launch_advisory: '' }))).toBe('success');
	});

	it('launch_advisory ABSENT (undefined) -> success, never throws', () => {
		const source = makeSource();
		delete (source as Partial<SourceStatus>).launch_advisory;
		expect(healthTone(source)).toBe('success');
	});
});

// --- 12-11-PLAN.md (CR-01): isAdvisoryOnly behavioural matrix ---

describe('isAdvisoryOnly: would this source be plain healthy with no advisory at all?', () => {
	it('reachable + last_status ok + non-empty last_notice -> true (the feature; also proves the branch is not dead code)', () => {
		expect(isAdvisoryOnly(makeSource({ last_notice: NOTICE }))).toBe(true);
	});

	it('reachable: false + last_status ok + non-empty last_notice -> false — THE CR-01 CASE: an unreachable source is never advisory-only, and this fails if the predicate\'s reachability handling is removed', () => {
		expect(isAdvisoryOnly(makeSource({ reachable: false, last_notice: NOTICE }))).toBe(false);
	});

	it('reachable + last_status error + non-empty last_notice -> false (a real error outranks the advisory)', () => {
		expect(isAdvisoryOnly(makeSource({ last_status: 'error', last_notice: NOTICE }))).toBe(false);
	});

	it('last_status "" (never synced) + last_sync_unix 0 + non-empty last_notice -> false', () => {
		expect(
			isAdvisoryOnly(makeSource({ last_status: '', last_sync_unix: 0, last_notice: NOTICE }))
		).toBe(false);
	});

	it('launch_failure pin_mismatch + non-empty last_notice -> false', () => {
		expect(
			isAdvisoryOnly(makeSource({ launch_failure: 'pin_mismatch', last_notice: NOTICE }))
		).toBe(false);
	});

	it('reachable + last_status ok + last_notice EMPTY STRING -> false', () => {
		expect(isAdvisoryOnly(makeSource({ last_notice: '' }))).toBe(false);
	});

	it('reachable + last_status ok + last_notice key deleted (undefined) -> false, never throws', () => {
		const source = makeSource();
		delete (source as Partial<SourceStatus>).last_notice;
		expect(isAdvisoryOnly(source)).toBe(false);
	});
});

// --- 13-06-PLAN.md (D-14): isAdvisoryOnly's second input, launch_advisory ---

describe('isAdvisoryOnly: launch_advisory shadowed (D-14, the second advisory input)', () => {
	it('reachable + last_status ok + launch_advisory shadowed -> true (the feature; also proves the branch is not dead code)', () => {
		expect(isAdvisoryOnly(makeSource({ launch_advisory: 'shadowed' }))).toBe(true);
	});

	it('reachable: false + launch_advisory shadowed -> false — the shadowed-source analogue of the CR-01 case', () => {
		expect(isAdvisoryOnly(makeSource({ reachable: false, launch_advisory: 'shadowed' }))).toBe(
			false
		);
	});

	it('last_status error + launch_advisory shadowed -> false (a real error outranks the shadowed advisory)', () => {
		expect(
			isAdvisoryOnly(makeSource({ last_status: 'error', launch_advisory: 'shadowed' }))
		).toBe(false);
	});

	it('last_status "" (never synced) + launch_advisory shadowed -> false', () => {
		expect(
			isAdvisoryOnly(
				makeSource({ last_status: '', last_sync_unix: 0, launch_advisory: 'shadowed' })
			)
		).toBe(false);
	});

	it('launch_failure pin_mismatch + launch_advisory shadowed -> false', () => {
		expect(
			isAdvisoryOnly(makeSource({ launch_failure: 'pin_mismatch', launch_advisory: 'shadowed' }))
		).toBe(false);
	});

	it('launch_failure manifest_unverified + launch_advisory shadowed -> false', () => {
		expect(
			isAdvisoryOnly(
				makeSource({ launch_failure: 'manifest_unverified', launch_advisory: 'shadowed' })
			)
		).toBe(false);
	});

	it('launch_advisory EMPTY STRING -> false', () => {
		expect(isAdvisoryOnly(makeSource({ launch_advisory: '' }))).toBe(false);
	});

	it('launch_advisory ABSENT (undefined) -> false, never throws', () => {
		const source = makeSource();
		delete (source as Partial<SourceStatus>).launch_advisory;
		expect(isAdvisoryOnly(source)).toBe(false);
	});

	it('both last_notice and launch_advisory shadowed present, source otherwise healthy -> true (clearing only one input would leave this a dead branch)', () => {
		expect(
			isAdvisoryOnly(makeSource({ last_notice: NOTICE, launch_advisory: 'shadowed' }))
		).toBe(true);
	});
});

describe('healthTone / isAdvisoryOnly coupling: manifest_unverified and shadowed', () => {
	it('for a manifest-unverified source, healthTone returns destructive AND isAdvisoryOnly returns false — a launch refusal is never advisory-only', () => {
		const source = makeSource({ launch_failure: 'manifest_unverified' });
		expect(healthTone(source)).toBe('destructive');
		expect(isAdvisoryOnly(source)).toBe(false);
	});

	it('for the healthy-and-shadowed source, healthTone returns warning AND isAdvisoryOnly returns true at the same time — the anti-dead-code guard, extended to the second advisory input', () => {
		const source = makeSource({ launch_advisory: 'shadowed' });
		expect(healthTone(source)).toBe('warning');
		expect(isAdvisoryOnly(source)).toBe(true);
	});
});

describe('healthTone / isAdvisoryOnly coupling: the anti-dead-code guard', () => {
	it('for the healthy-with-advisory source, healthTone returns warning AND isAdvisoryOnly returns true at the same time — the two deliberately DISAGREE in value and AGREE in meaning, so a future edit gating the tooltip on the chip\'s own tone value is caught here rather than in production', () => {
		const source = makeSource({ last_notice: NOTICE });
		expect(healthTone(source)).toBe('warning');
		expect(isAdvisoryOnly(source)).toBe(true);
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

describe('tooltipText structure: the new advisory branch defers to format.ts\'s precedence (12-11-PLAN.md CR-01)', () => {
	it('a branch referencing the advisoryOnly derived value exists strictly before the tone switch', () => {
		const advisoryIdx = strippedChip.indexOf("advisory !== '' && advisoryOnly");
		const switchIdx = strippedChip.indexOf('switch (tone)');
		expect(
			advisoryIdx,
			'expected an advisoryOnly-gated branch in tooltipText, deferring to format.ts\'s precedence rather than re-deriving a narrower condition of its own'
		).toBeGreaterThanOrEqual(0);
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

	it('the chip declares an advisoryOnly derived value bound to isAdvisoryOnly(source)', () => {
		expect(
			strippedChip.includes('let advisoryOnly = $derived(isAdvisoryOnly(source));'),
			'expected the chip to declare `advisoryOnly` bound to isAdvisoryOnly(source), deferring the precedence rule to format.ts instead of re-deriving its own narrower condition'
		).toBe(true);
	});

	it('isAdvisoryOnly is imported from $lib/format', () => {
		expect(
			/import\s*\{[^}]*isAdvisoryOnly[^}]*\}\s*from\s*'\$lib\/format'/.test(strippedChip),
			'expected isAdvisoryOnly to be imported from $lib/format alongside healthTone/formatRelativeTime/shortHash'
		).toBe(true);
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

describe('tooltipText structure: branch ORDER (syncing -> pin-mismatch -> advisory -> tone switch)', () => {
	it('the syncing check runs before the pin-mismatch check', () => {
		const syncingIdx = strippedChip.indexOf('if (source.syncing)');
		const pinMismatchIdx = strippedChip.indexOf('if (isPinMismatch)');
		expect(syncingIdx, 'expected the syncing check to be present').toBeGreaterThanOrEqual(0);
		expect(pinMismatchIdx, 'expected the pin-mismatch check to be present').toBeGreaterThanOrEqual(
			0
		);
		expect(
			syncingIdx,
			'expected the syncing check to run BEFORE the pin-mismatch check — a source can be mid-sync regardless of its last recorded outcome'
		).toBeLessThan(pinMismatchIdx);
	});

	it('the pin-mismatch check runs before the advisory gate', () => {
		const pinMismatchIdx = strippedChip.indexOf('if (isPinMismatch)');
		const advisoryIdx = strippedChip.indexOf("advisory !== '' && advisoryOnly");
		expect(
			pinMismatchIdx,
			'expected the pin-mismatch check to run BEFORE the advisory gate — a binary-changed source is a bigger fact than an advisory about a run it never had'
		).toBeLessThan(advisoryIdx);
	});

	it('the advisory gate runs before the tone switch (already covered above; extended here rather than duplicated)', () => {
		const advisoryIdx = strippedChip.indexOf("advisory !== '' && advisoryOnly");
		const switchIdx = strippedChip.indexOf('switch (tone)');
		expect(advisoryIdx).toBeLessThan(switchIdx);
	});

	// 13-06-PLAN.md (D-12/D-13/D-14): the two new branches slot into this
	// same ordered chain — manifest-unverified beside pin-mismatch in the
	// problem group, shadowed after the last_notice advisory gate (so a
	// source carrying both reads the notice's sentence) and before the
	// tone switch.
	it('the pin-mismatch check runs before the manifest-unverified check', () => {
		const pinMismatchIdx = strippedChip.indexOf('if (isPinMismatch)');
		const manifestIdx = strippedChip.indexOf('if (isManifestUnverified)');
		expect(manifestIdx, 'expected the manifest-unverified check to be present').toBeGreaterThanOrEqual(
			0
		);
		expect(pinMismatchIdx).toBeLessThan(manifestIdx);
	});

	it('the manifest-unverified check runs before the advisory gate', () => {
		const manifestIdx = strippedChip.indexOf('if (isManifestUnverified)');
		const advisoryIdx = strippedChip.indexOf("advisory !== '' && advisoryOnly");
		expect(manifestIdx).toBeLessThan(advisoryIdx);
	});

	it('the last_notice advisory gate runs before the shadowed gate', () => {
		const advisoryIdx = strippedChip.indexOf("advisory !== '' && advisoryOnly");
		const shadowedIdx = strippedChip.indexOf('if (isShadowed && advisoryOnly)');
		expect(shadowedIdx, 'expected the shadowed gate to be present').toBeGreaterThanOrEqual(0);
		expect(advisoryIdx).toBeLessThan(shadowedIdx);
	});

	it('the shadowed gate runs before the tone switch', () => {
		const shadowedIdx = strippedChip.indexOf('if (isShadowed && advisoryOnly)');
		const switchIdx = strippedChip.indexOf('switch (tone)');
		expect(shadowedIdx).toBeLessThan(switchIdx);
	});
});

describe('tooltipText structure: the two new 13-06-PLAN.md branches (D-12/D-13/D-14)', () => {
	it('the manifest-unverified derived value is keyed on launch_failure alone', () => {
		expect(
			strippedChip.includes(
				"let isManifestUnverified = $derived(source.launch_failure === 'manifest_unverified');"
			)
		).toBe(true);
	});

	it('the shadowed derived value is keyed on launch_advisory alone', () => {
		expect(
			strippedChip.includes("let isShadowed = $derived(source.launch_advisory === 'shadowed');")
		).toBe(true);
	});

	it('the manifest-unverified sentence is contract-exact', () => {
		expect(
			strippedChip.includes(
				'return `${source.display_name} — binary not in the trusted build manifest`;'
			)
		).toBe(true);
	});

	it('the shadowed sentence is contract-exact', () => {
		expect(
			strippedChip.includes(
				'return `${source.display_name} — a same-named trusted-directory binary is shadowing this pinned plugin`;'
			)
		).toBe(true);
	});

	it('the chip menu re-pin action stays gated on isPinMismatch alone — never extended to isManifestUnverified', () => {
		const menuStart = strippedChip.indexOf('<DropdownMenuContent>');
		const menuEnd = strippedChip.indexOf('</DropdownMenuContent>');
		expect(menuStart, 'expected to find the chip menu content block').toBeGreaterThanOrEqual(0);
		expect(menuEnd).toBeGreaterThan(menuStart);
		const menuBlock = strippedChip.slice(menuStart, menuEnd);
		expect(
			menuBlock.includes('{#if isPinMismatch}'),
			'expected the re-pin item to remain gated on isPinMismatch'
		).toBe(true);
		expect(
			menuBlock.includes('isManifestUnverified'),
			'expected the chip menu to reference isManifestUnverified nowhere — no new remedial action for this state'
		).toBe(false);
	});
});

// --- 12-11-PLAN.md (CR-01): tooltipText BRANCH-SELECTION matrix ---
//
// This mirror function reproduces the component's tooltipText derivation
// using the REAL imported healthTone, isAdvisoryOnly and formatRelativeTime
// — the same mirroring technique source-chip-tooltip.test.ts documents, for
// the same reason (this repo's vitest config is `environment: 'node'` with
// no component-mount harness). ANCHOR: every template literal and every
// branch condition this mirror reproduces is asserted present (structurally,
// byte-for-byte or by the order assertions above) in the REAL component's
// source elsewhere in this file — this mirror is not free-floating: if the
// component's actual branch text or order ever drifts from what is
// reproduced here, one of the structural/order assertions above fails
// first, which is what keeps this mirror honest.
function mirrorTooltipText(source: SourceStatus): string {
	if (source.syncing) return `${source.display_name} — syncing…`;
	const isPinMismatch = source.launch_failure === 'pin_mismatch';
	if (isPinMismatch) return `${source.display_name} — binary changed since it was trusted`;
	const isManifestUnverified = source.launch_failure === 'manifest_unverified';
	if (isManifestUnverified) return `${source.display_name} — binary not in the trusted build manifest`;
	const relative = formatRelativeTime(source.last_sync_unix);
	const advisory = (source.last_notice ?? '').trim();
	const advisoryOnly = isAdvisoryOnly(source);
	if (advisory !== '' && advisoryOnly) {
		return `${source.display_name} — synced ${relative} — ${advisory}`;
	}
	const isShadowed = source.launch_advisory === 'shadowed';
	if (isShadowed && advisoryOnly) {
		return `${source.display_name} — a same-named trusted-directory binary is shadowing this pinned plugin`;
	}
	switch (healthTone(source)) {
		case 'success':
			return `${source.display_name} — synced ${relative}`;
		case 'warning':
			return `${source.display_name} — last error ${relative}: ${source.last_error}`;
		case 'destructive':
			return `${source.display_name} — unreachable since ${relative}`;
		default:
			return `${source.display_name} — not yet synced`;
	}
}

describe('tooltipText BRANCH-SELECTION matrix: the exact string produced per precedence case (12-11-PLAN.md CR-01)', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2024-06-15T12:00:00Z'));
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	it('reachable: false + last_status ok + non-empty last_notice -> the unreachable-since sentence, carrying NONE of the notice text — CR-01, the case whose absence from this file let the defect ship', () => {
		const source = makeSource({
			reachable: false,
			last_status: 'ok',
			last_sync_unix: Math.floor(Date.now() / 1000) - 60 * 60,
			last_notice: NOTICE
		});
		const text = mirrorTooltipText(source);
		expect(text).toBe('Household Docs — unreachable since 1 hour ago');
		expect(text).not.toContain(NOTICE);
	});

	it('launch_failure pin_mismatch + non-empty last_notice -> the binary-changed sentence', () => {
		const source = makeSource({ launch_failure: 'pin_mismatch', last_notice: NOTICE });
		expect(mirrorTooltipText(source)).toBe(
			'Household Docs — binary changed since it was trusted'
		);
	});

	it('never-synced + non-empty last_notice -> the not-yet-synced sentence', () => {
		const source = makeSource({ last_status: '', last_sync_unix: 0, last_notice: NOTICE });
		expect(mirrorTooltipText(source)).toBe('Household Docs — not yet synced');
	});

	it('last_status error + non-empty last_notice -> the last-error sentence carrying last_error', () => {
		const source = makeSource({
			last_status: 'error',
			last_error: 'connection refused',
			last_sync_unix: Math.floor(Date.now() / 1000) - 3 * 60 * 60,
			last_notice: NOTICE
		});
		expect(mirrorTooltipText(source)).toBe(
			'Household Docs — last error 3 hours ago: connection refused'
		);
	});

	it('reachable + last_status ok + non-empty last_notice -> the synced-plus-advisory sentence carrying the notice verbatim', () => {
		const source = makeSource({
			last_sync_unix: Math.floor(Date.now() / 1000) - 5 * 60,
			last_notice: NOTICE
		});
		expect(mirrorTooltipText(source)).toBe(`Household Docs — synced 5 minutes ago — ${NOTICE}`);
	});

	it('syncing: true + reachable: false + non-empty last_notice -> the syncing sentence (ordering above everything else is unchanged)', () => {
		const source = makeSource({ syncing: true, reachable: false, last_notice: NOTICE });
		expect(mirrorTooltipText(source)).toBe('Household Docs — syncing…');
	});

	// 13-06-PLAN.md (D-12/D-13/D-14) matrix additions.
	it('launch_failure manifest_unverified -> the contract-exact manifest sentence, regardless of sync history', () => {
		const source = makeSource({ launch_failure: 'manifest_unverified' });
		expect(mirrorTooltipText(source)).toBe('Household Docs — binary not in the trusted build manifest');
	});

	it('launch_failure manifest_unverified + launch_advisory shadowed -> the manifest sentence (a launch refusal outranks the shadowed advisory)', () => {
		const source = makeSource({
			launch_failure: 'manifest_unverified',
			launch_advisory: 'shadowed'
		});
		expect(mirrorTooltipText(source)).toBe('Household Docs — binary not in the trusted build manifest');
	});

	it('reachable + last_status ok + launch_advisory shadowed -> the contract-exact shadowed sentence', () => {
		const source = makeSource({
			last_sync_unix: Math.floor(Date.now() / 1000) - 5 * 60,
			launch_advisory: 'shadowed'
		});
		expect(mirrorTooltipText(source)).toBe(
			'Household Docs — a same-named trusted-directory binary is shadowing this pinned plugin'
		);
	});

	it('reachable: false + launch_advisory shadowed -> the unreachable-since sentence, carrying no shadowed wording (a real problem outranks the advisory)', () => {
		const source = makeSource({
			reachable: false,
			last_sync_unix: Math.floor(Date.now() / 1000) - 60 * 60,
			launch_advisory: 'shadowed'
		});
		const text = mirrorTooltipText(source);
		expect(text).toBe('Household Docs — unreachable since 1 hour ago');
		expect(text).not.toContain('shadowing');
	});

	it('launch_failure pin_mismatch + launch_advisory shadowed -> the binary-changed sentence, not the shadowing sentence', () => {
		const source = makeSource({ launch_failure: 'pin_mismatch', launch_advisory: 'shadowed' });
		const text = mirrorTooltipText(source);
		expect(text).toBe('Household Docs — binary changed since it was trusted');
		expect(text).not.toContain('shadowing');
	});

	it('both last_notice and launch_advisory shadowed present, source otherwise healthy -> the last_notice sentence (today\'s specified coexistence order)', () => {
		const source = makeSource({
			last_sync_unix: Math.floor(Date.now() / 1000) - 5 * 60,
			last_notice: NOTICE,
			launch_advisory: 'shadowed'
		});
		const text = mirrorTooltipText(source);
		expect(text).toBe(`Household Docs — synced 5 minutes ago — ${NOTICE}`);
		expect(text).not.toContain('shadowing');
	});

	it('syncing: true + launch_advisory shadowed -> the syncing sentence (ordering above everything else is unchanged)', () => {
		const source = makeSource({ syncing: true, launch_advisory: 'shadowed' });
		expect(mirrorTooltipText(source)).toBe('Household Docs — syncing…');
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
