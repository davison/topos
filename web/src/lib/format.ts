// Deterministic UTC date formatting for date-only source timestamps, plus
// the pure (component-free, unit-testable) derivation helpers backing
// Phase 2's health chips, source filter and staleness states — kept here
// alongside formatItemDate/formatFidelity so every branch a template picks
// between is assertable without mounting a component.
//
// paperless-ngx's `created` field is a date-only value stored as midnight
// UTC. If we format it in the browser's local timezone, a user west of UTC
// (any negative offset) would see the previous day for documents dated on
// the boundary — the calendar day must follow the source, not the viewer's
// clock, so `timeZone: 'UTC'` is pinned unconditionally here rather than
// left to the locale's default.

import { SNIPPET_OPEN, SNIPPET_CLOSE } from './api';
import type { ItemContent, SourceStatus, StreamItem, StreamResponse } from './api';

/** Formats a unix timestamp (seconds) as a UTC calendar day, e.g. "1 Jan 2024". */
export function formatItemDate(timestampUnix: number): string {
	if (!timestampUnix) return '';
	return new Intl.DateTimeFormat(undefined, {
		day: 'numeric',
		month: 'short',
		year: 'numeric',
		timeZone: 'UTC'
	}).format(new Date(timestampUnix * 1000));
}

// Fixed display labels for the three-value LinkFidelity enum (PLUG-03).
// The label is never derived from source content — always one of these
// three fixed strings, or the raw enum value as a defensive fallback.
const FIDELITY_LABELS: Record<string, string> = {
	exact: 'exact',
	anchored: 'anchored',
	'conversation-only': 'conversation-only'
};

/** Maps a LinkFidelity enum string to its fixed display label. */
export function formatFidelity(fidelity: string): string {
	return FIDELITY_LABELS[fidelity] ?? fidelity;
}

// --- Deep-link fidelity affordance (UI-08) ---

export interface FidelityAffordance {
	windowOnly: boolean;
	label: string;
	title: string;
}

/**
 * Maps a `Link.fidelity` value plus the item's resolved source display name
 * to the open-affordance's icon class, verb and title (04-UAT follow-up,
 * UI-08). This is a deliberate TWO-class split — `windowOnly` is true only
 * for the literal string `conversation-only`; every other value, including
 * an unrecognised one, produces the navigating treatment — layered
 * alongside `formatFidelity` above, which stays a three-value, power-user
 * badge. Defaulting an unfamiliar value to the navigating treatment is
 * deliberate: an unfamiliar enum value degrades to the commoner, less
 * alarming affordance rather than throwing or rendering blank, the same
 * defensive posture `formatFidelity`'s own raw-value fallback already
 * takes for the badge. Both copy strings are frozen from the UI-SPEC's
 * Copywriting Contract — do not paraphrase them.
 */
export function fidelityAffordance(fidelity: string, displayName: string): FidelityAffordance {
	if (fidelity === 'conversation-only') {
		return {
			windowOnly: true,
			label: `Show in ${displayName}`,
			title: `Raise ${displayName} — opens the app/conversation, not this exact message`
		};
	}
	return {
		windowOnly: false,
		label: `Open in ${displayName}`,
		title: `Open in ${displayName}`
	};
}

// --- Health chips (D-08) ---

const RELATIVE_UNITS: [Intl.RelativeTimeFormatUnit, number][] = [
	['year', 60 * 60 * 24 * 365],
	['month', 60 * 60 * 24 * 30],
	['week', 60 * 60 * 24 * 7],
	['day', 60 * 60 * 24],
	['hour', 60 * 60],
	['minute', 60],
	['second', 1]
];

/**
 * Human relative-time string for a unix-seconds timestamp, e.g. "5 minutes
 * ago". Returns the empty string for a zero/missing timestamp — a source
 * that has never synced must never render as "56 years ago".
 */
export function formatRelativeTime(unixSeconds: number): string {
	if (!unixSeconds) return '';
	const deltaSeconds = unixSeconds - Math.floor(Date.now() / 1000);
	const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' });
	for (const [unit, secondsInUnit] of RELATIVE_UNITS) {
		if (Math.abs(deltaSeconds) >= secondsInUnit || unit === 'second') {
			return rtf.format(Math.round(deltaSeconds / secondsInUnit), unit);
		}
	}
	return '';
}

export type HealthTone = 'success' | 'warning' | 'destructive' | 'unknown';

/**
 * Maps a source's live-reachability + recorded-sync-history fields to the
 * health-chip dot tone. `last_status: ''` (never synced) is checked first
 * and always renders neutral/unknown — per docs/api.md, this is the
 * "render as a neutral indicator, never as a green 'ok'" rule — before
 * live reachability is even considered, since a never-synced source has
 * no sync history to be "unreachable since" either.
 *
 * 12-10-PLAN.md (G-12-1/G-12-3 gap closure) adds ONE more branch, LAST
 * among the problem states, immediately before the final success return: a
 * source carrying a non-empty `last_notice` — today, a webspace's explicit
 * match block that matched none of this source's items — reads `warning`
 * even though its sync genuinely succeeded. It sits last because a pin
 * mismatch, a never-synced source, an unreachable source and a failed sync
 * are each a bigger fact than an advisory about an otherwise-successful
 * run: a real problem must never be displaced by "also, here's a heads
 * up." It reuses the existing `warning` tone rather than introducing a
 * fifth `HealthTone` — the errored-status branch above already returns
 * `warning` for exactly this reason: in this system amber already means
 * "synced, but attend to this," which is precisely the advisory's
 * meaning, so no new design token is warranted.
 *
 * 13-06-PLAN.md (D-12/D-13/D-14, PLUG-07) adds two more inputs to this
 * SAME chain, never a parallel gate:
 *  - `launch_failure === 'manifest_unverified'` is checked immediately
 *    beside the existing pin-mismatch check, for the identical reason: a
 *    trusted-directory binary refused at launch by the build-provenance
 *    manifest never got a chance to sync either, and must not fall into
 *    the neutral 'unknown' branch below.
 *  - `launch_advisory === 'shadowed'` occupies the SAME position the
 *    `last_notice` branch above does — last among the problem states,
 *    immediately before the success return — for the identical reason: a
 *    same-named trusted-directory binary shadowing a pinned external
 *    plugin is an ambiguity advisory, not a failure, and must never
 *    displace a real problem (pin mismatch, manifest-unverified,
 *    never-synced, unreachable, errored). Reuses the existing `warning`
 *    tone, same rationale as the advisory branch above.
 */
export function healthTone(source: SourceStatus): HealthTone {
	// Phase 11 (11-UI-SPEC.md E4, D-03): a source refused launch on a pin
	// mismatch is destructive REGARDLESS of its sync history — checked
	// first, ahead of the never-synced/unreachable/last-status branches,
	// since a pin-mismatched instance never got a chance to sync at all
	// and would otherwise wrongly fall into the neutral 'unknown' branch
	// below. Extends the existing branch chain; never a parallel tone
	// system.
	if (source.launch_failure === 'pin_mismatch') return 'destructive';
	// 13-06-PLAN.md (D-12/D-13): identical reasoning to the pin-mismatch
	// check immediately above — a trusted-directory binary that failed
	// build-manifest verification never launched, and never sees a chance
	// to sync.
	if (source.launch_failure === 'manifest_unverified') return 'destructive';
	// M1-R6/DIST-03 (davison/topos#17): the three incompatibility/launch
	// classes join the SAME position for the identical reason yet again —
	// an instance refused or failed at launch never got a chance to sync
	// and must not fall into the neutral 'unknown' branch below. Same
	// chain, never a parallel gate.
	if (source.launch_failure === 'handshake_incompatible') return 'destructive';
	if (source.launch_failure === 'contract_incompatible') return 'destructive';
	if (source.launch_failure === 'launch_failed') return 'destructive';
	if (source.last_status === '') return 'unknown';
	if (!source.reachable) return 'destructive';
	if (source.last_status === 'error') return 'warning';
	if ((source.last_notice ?? '') !== '') return 'warning';
	// 13-06-PLAN.md (D-14): same position as the last_notice branch above
	// — last among the problem states, immediately before success — so a
	// real problem (pin mismatch, manifest-unverified, never-synced,
	// unreachable, errored, or a last_notice advisory) always outranks
	// this one.
	if (source.launch_advisory === 'shadowed') return 'warning';
	return 'success';
}

/**
 * 12-11-PLAN.md (CR-01 gap closure): answers exactly one question — "would
 * this source be plain healthy if it carried no advisory at all?" —
 * `healthTone`'s own advisory branch (above) is only ever reached after
 * every earlier problem branch (pin mismatch, never-synced, unreachable,
 * errored) has already declined, so re-asking `healthTone` with the notice
 * removed reuses that precedence chain instead of re-deriving a narrower
 * copy of it. Any surface that needs to know whether the advisory is the
 * most important thing to say about this source (e.g. SourceChip.svelte's
 * tooltip gate) calls this rather than inventing its own condition — that
 * is what stops the tooltip's copy and the dot's tone from drifting apart,
 * which is exactly how CR-01 shipped: the old gate checked `last_status`
 * directly and never consulted `reachable`, so an unreachable source with a
 * stale successful status and a leftover notice read as a benign advisory
 * while its own dot was red.
 *
 * The trap: in the advisory state `healthTone` itself returns `warning`,
 * never `success` — a source carrying a non-empty `last_notice` can never
 * satisfy `healthTone(source) === 'success'`. Gating a consumer on the
 * chip's own already-computed `tone` value being `'success'` would
 * therefore make an advisory-gated branch unreachable dead code.
 *
 * 13-06-PLAN.md (D-14) widens this to a SECOND input, `launch_advisory`:
 * the early return now answers `false` only when BOTH `last_notice` and
 * `launch_advisory` are empty, and BOTH are cleared together when
 * re-asking `healthTone` — clearing only one would leave the other's
 * branch permanently reachable-but-never-`success`, the identical
 * dead-code trap this function's own doc comment already warns about, now
 * applied to the second advisory source.
 */
export function isAdvisoryOnly(source: SourceStatus): boolean {
	const hasNotice = (source.last_notice ?? '') !== '';
	const hasShadowed = source.launch_advisory === 'shadowed';
	if (!hasNotice && !hasShadowed) return false;
	return healthTone({ ...source, last_notice: '', launch_advisory: '' }) === 'success';
}

/**
 * Formats a hex hash (SHA-256 or otherwise) into its fixed, display-only
 * short form: the first 12 characters plus an ellipsis (11-UI-SPEC.md E5's
 * "short to look at, complete when copied/compared" rule, reused by E4's
 * dialog). The empty string is returned unchanged — a caller with no pin to
 * show renders its own "not pinned"-style copy rather than this helper
 * guessing at it.
 */
export function shortHash(hash: string): string {
	if (!hash) return '';
	return `${hash.slice(0, 12)}…`;
}

/** Instance ids (SourceStatus.name) of every source currently mid-sync. */
export function syncingSources(sources: SourceStatus[]): string[] {
	return sources.filter((s) => s.syncing).map((s) => s.name);
}

/**
 * Whether the health-chip row and the source-filter row should render at
 * all — both are governed by this single rule (02-UI-SPEC.md UI
 * Considerations, E1/E5 empty+loading+error rows): absent until the
 * sources response resolves successfully with at least one configured
 * source, so a non-critical header surface failing (or a sourceless
 * webspace) never half-renders header chrome.
 */
export function shouldShowSourceRows(
	sourcesState: 'loading' | 'error' | 'ready',
	sources: SourceStatus[]
): boolean {
	return sourcesState === 'ready' && sources.length > 0;
}

// --- Source filter (D-02 multi-select, superseding Phase 2's D-09 single-select) ---

/**
 * Resolves a requested source-instance filter (typically the URL's
 * `sources` query parameter, a comma-joined list of source INSTANCE ids —
 * D-08) against the configured sources. Splits on commas, trims each
 * member, drops empties, and keeps only members naming a configured
 * instance by `name`. Degrades PER MEMBER (06-RESEARCH.md Pitfall 4, the
 * multi-value form of T-02-17's single-value degrade rule): a value
 * naming one configured and one since-removed instance yields a
 * one-member set holding only the configured name — never an empty set
 * and never an error just because one member is stale. A null, empty or
 * whitespace-only input yields an empty set, meaning "no filter, show
 * everything." Load-time uniqueness (D-09) guarantees every configured
 * instance's `name` is distinct, so membership testing is unambiguous
 * even with two instances sharing one plugin kind.
 */
export function resolveSourceFilters(requested: string | null, sources: SourceStatus[]): Set<string> {
	const result = new Set<string>();
	if (!requested) return result;
	const known = new Set(sources.map((s) => s.name));
	for (const raw of requested.split(',')) {
		const member = raw.trim();
		if (member === '') continue;
		if (known.has(member)) result.add(member);
	}
	return result;
}

/**
 * Returns a new set with `name` added when it is absent from `current`
 * and removed when it is present — `current` itself is never mutated.
 */
export function toggleSourceFilter(current: Set<string>, name: string): Set<string> {
	const next = new Set(current);
	if (next.has(name)) {
		next.delete(name);
	} else {
		next.add(name);
	}
	return next;
}

/**
 * Serializes a selection set back into the `sources` query value: members
 * joined by commas in the caller's iteration order. An empty set
 * serializes to the empty string (the caller deletes the query key
 * entirely for that case, per D-02's "all-off = show everything" rule).
 */
export function serializeSourceFilters(selected: Set<string>): string {
	return [...selected].join(',');
}

/** Narrows a stream's items to the selected source INSTANCE ids; an empty set returns every item, unchanged in order. */
export function filterItemsBySource(items: StreamItem[], selected: Set<string>): StreamItem[] {
	if (selected.size === 0) return items;
	return items.filter((item) => selected.has(item.source));
}

export type StreamVariant = 'sync-failed' | 'empty' | 'empty-filtered' | 'populated';

/**
 * The single decision StreamList.svelte renders from (once loaded): a
 * failed sync with zero items always wins, filtered or not — computed
 * from the response's own aggregate sync status and its *unfiltered*
 * item count, never the filtered subset, so a filter can never mask a
 * sync failure (T-02-16, PLAN.md prohibitions). Only once that's ruled
 * out does an empty filtered view get its own distinct copy from an
 * empty unfiltered one — keyed on whether the selection set is empty,
 * rather than whether a single value is null.
 */
export function streamVariant(response: StreamResponse, selected: Set<string>): StreamVariant {
	if (response.sync.status === 'error' && response.items.length === 0) return 'sync-failed';
	if (filterItemsBySource(response.items, selected).length === 0) {
		return selected.size === 0 ? 'empty' : 'empty-filtered';
	}
	return 'populated';
}

/**
 * Reduces a set of sources to the single most alarming health tone —
 * fixed precedence destructive, then warning, then unknown, then success
 * — the existing per-source `healthTone` map reduced over a set, not a
 * new tone vocabulary. Returns the neutral `unknown` tone for an empty
 * input, matching `healthTone`'s own neutral default.
 */
export function worstHealthTone(sources: SourceStatus[]): HealthTone {
	const TONE_RANK: Record<HealthTone, number> = {
		destructive: 0,
		warning: 1,
		unknown: 2,
		success: 3
	};
	if (sources.length === 0) return 'unknown';
	// Seed with `success` — the least-alarming rank — so any real source's
	// tone always wins the first comparison; the empty-input case is
	// handled separately above rather than by this seed, since `success`
	// would otherwise be a wrong answer for zero sources.
	let worst: HealthTone = 'success';
	for (const source of sources) {
		const tone = healthTone(source);
		if (TONE_RANK[tone] < TONE_RANK[worst]) worst = tone;
	}
	return worst;
}

/**
 * Returns how many chips fit inline given their natural widths (in
 * order), the row's available width, the width reserved for the trailing
 * controls (Clear filters / Refresh all — never pushed into overflow),
 * the overflow trigger's own width, and the row's real inter-item gap
 * (the `gap-2` / 8px spacing the flex row applies between every adjacent
 * child — CR-01).
 *
 * `reservedWidth` is subtracted from `availableWidth` first. Every visible
 * chip charges its own width plus the gap that precedes it (no gap before
 * the first chip), plus one trailing gap between the last visible item and
 * the reserved trailing group — `N` chips therefore consume `N-1` between-
 * chip gaps and 1 trailing gap, never zero, since the trailing group is
 * always rendered. If every chip's combined width (plus those gaps) fits
 * that budget, every chip is visible and no overflow trigger is needed —
 * the trigger's width is never charged in that case, since nothing is
 * deferred to it. Only once the full set does NOT fit is the budget
 * further reduced by `overflowTriggerWidth` plus two more gaps (one
 * between the last visible chip and the trigger, one between the trigger
 * and the trailing group), and chips are accumulated in order against
 * that reduced budget, each still charging its own preceding gap.
 *
 * Accumulation uses a strict `>` comparison, so a chip that would exceed
 * the budget by any fraction of a pixel is excluded rather than rendered
 * half-clipped at the row's trailing edge — this is the answer to the
 * sub-pixel measurement question. The function is pure and
 * side-effect-free: a repeated pass over unchanged inputs yields an
 * identical count, and a budget that goes negative (reserved and trigger
 * widths alone exceed the available width) naturally floors the result
 * at zero rather than a negative count.
 *
 * `minInlineChipWidth` (14-06, G-14-2): when the accumulation above lands
 * on ZERO chips — the narrow-viewport case where even the first chip's
 * natural width overruns the reduced budget — the count is floored at one
 * chip, provided the row genuinely has a chip to show and the overflow
 * budget can seat at least `minInlineChipWidth` of it. The default of 0
 * disables the floor entirely, preserving the previous contract exactly —
 * which is why the call-site-composition note above and the existing
 * tests hold unmodified. The gate depends on how many chips remain: with
 * two or more candidates a forced chip coexists with the overflow trigger
 * (something is always left to relegate), so the trigger's width is
 * charged via the OVERFLOW budget; with exactly ONE candidate, forcing it
 * leaves nothing to relegate — no trigger ever renders — so the true
 * ceiling is the plain budget minus the one trailing gap (WR-01,
 * 14-REVIEW-GAPS.md). Exactly one chip is forced — never more — so the
 * Phase 6 UI-07 overflow design still owns everything past the first
 * chip. The caller is responsible for making that forced chip shrinkable:
 * this arithmetic promises only that space for a minimum-width chip
 * exists, not that the chip's natural width fits.
 */
export function visibleChipCount(
	chipWidths: number[],
	availableWidth: number,
	reservedWidth: number,
	overflowTriggerWidth: number,
	gapWidth: number,
	minInlineChipWidth: number = 0
): number {
	const budget = availableWidth - reservedWidth;
	const gapsAmong = (n: number) => (n > 0 ? (n - 1) * gapWidth : 0);
	// +1 trailing gap: the space between the last visible item (chip or
	// trigger) and the reserved trailing group.
	const total =
		chipWidths.reduce((sum, width) => sum + width, 0) +
		gapsAmong(chipWidths.length) +
		(chipWidths.length > 0 ? gapWidth : 0);
	if (total <= budget) return chipWidths.length;

	// One more gap for the trigger itself, plus the gap before it.
	const overflowBudget = budget - overflowTriggerWidth - gapWidth * 2;
	let used = 0;
	let count = 0;
	for (const width of chipWidths) {
		const candidateGap = count > 0 ? gapWidth : 0;
		if (used + candidateGap + width > overflowBudget) break;
		used += candidateGap + width;
		count += 1;
	}
	// The G-14-2 floor: only ever raises a zero, only when a chip exists,
	// and only when the applicable budget can seat a minimum-width chip. A
	// zero minInlineChipWidth disables it (the pre-14-06 contract).
	if (count === 0 && minInlineChipWidth > 0 && chipWidths.length > 0) {
		// Forcing the ONLY candidate chip leaves nothing to relegate, so no
		// overflow trigger will ever render for it — the true ceiling is
		// the plain budget minus one trailing gap, not the multi-chip
		// overflow-budget formula (which needlessly reserves the trigger's
		// width plus an extra gap that will never be spent). WR-01.
		const forcedBudget = chipWidths.length === 1 ? budget - gapWidth : overflowBudget;
		if (forcedBudget >= minInlineChipWidth) return 1;
	}
	return count;
}

// --- Staleness (D-10) ---

/**
 * Source INSTANCE ids (SourceStatus.name) considered stale — currently
 * unreachable, per the live sources response. Keyed by instance, never by
 * plugin kind (D-08): a healthy instance's rows are never marked stale
 * merely because a sibling instance of the same plugin type is
 * unreachable.
 */
export function staleSources(sources: SourceStatus[]): Set<string> {
	return new Set(sources.filter((s) => !s.reachable).map((s) => s.name));
}

export type DetailPaneState = 'loaded' | 'deleted' | 'unreachable' | 'error';

/**
 * The one place that decides which of the detail pane's four mutually
 * exclusive states to show — exactly one, never two, never none (D-10):
 *  - `deleted`: the kernel reports the content itself is unavailable
 *    (`content.available === false`), checked first so it takes
 *    precedence over a simultaneous fetch-error signal.
 *  - `unreachable` / `error`: the live fetch failed; which of the two
 *    depends only on whether this item's source is currently reachable,
 *    not on the specific error code.
 *  - `loaded`: neither signal is present.
 */
export function detailPaneState(
	content: ItemContent | null,
	fetchErrorCode: string | null,
	sourceReachable: boolean
): DetailPaneState {
	if (content !== null && content.available === false) return 'deleted';
	if (fetchErrorCode !== null) return sourceReachable ? 'error' : 'unreachable';
	return 'loaded';
}

export type DetailBodyVariant = 'html' | 'media' | 'text' | 'empty';

/**
 * The single decision the detail pane's body region renders from (once
 * `detailPaneState` above has resolved to `loaded`) — evaluated in this
 * order, mirroring `detailPaneState`'s own documented precedence:
 *  1. a `null` content is `empty`;
 *  2. an unavailable content (`available === false`) is `empty`
 *     regardless of its other fields — the pane routes that case to its
 *     own `deleted` state before ever asking for a body variant, but this
 *     function stays total anyway;
 *  3. a `text/html` rendition is `html` — checked BEFORE text precisely
 *     because a source may legitimately return both simultaneously (the
 *     SilverBullet shape: a rendered-markdown iframe alongside raw
 *     markdown `text`), and the rendered document is that source's
 *     chosen representation, so it must win;
 *  4. an `application/pdf` rendition, or a mime type beginning with
 *     `image/`, is `media` — also checked alongside non-empty text (the
 *     paperless shape: a fixed-height preview box plus text below,
 *     unchanged by this decision);
 *  5. text that is non-empty after trimming is `text` — the branch a
 *     Proton email carrying a plain-text alternative lands on, because
 *     its plugin (03-09-PLAN.md Task 1) declines to emit a rendition at
 *     all when the message has readable plain text, so this decision
 *     needs no source identity to reach the right outcome;
 *  6. everything else — no rendition and no (or whitespace-only) text, or
 *     a rendition of an unrecognised mime type with no text either — is
 *     `empty`, so an unknown rendition type never blanks a pane that has
 *     readable text (it falls through to rule 5 first).
 */
export function detailBodyVariant(content: ItemContent | null): DetailBodyVariant {
	if (content === null) return 'empty';
	if (!content.available) return 'empty';

	const mimeType = content.rendition?.mime_type ?? '';
	if (mimeType === 'text/html') return 'html';
	if (mimeType === 'application/pdf' || mimeType.startsWith('image/')) return 'media';

	return content.text.trim() !== '' ? 'text' : 'empty';
}

// --- Search (KERN-05 browser half, 03-04) ---

export interface SnippetSegment {
	text: string;
	match: boolean;
}

/**
 * Splits a kernel search-result `snippet` string (03-03-SUMMARY.md's
 * SnippetOpen/SnippetClose STX/ETX delimiter contract) into an ordered
 * list of plain-text/matched segments a template renders as one span
 * each. Never emits a segment carrying a delimiter character, and never
 * throws — a malformed or unpaired delimiter run degrades the whole
 * snippet to a single plain-text (non-matched) segment with every
 * delimiter character stripped, rather than propagating a parse error
 * into the render path (T-03-22).
 */
export function parseSnippet(snippet: string): SnippetSegment[] {
	if (snippet === '') return [];

	const positions: { index: number; open: boolean }[] = [];
	let i = 0;
	while (i < snippet.length) {
		if (snippet.startsWith(SNIPPET_OPEN, i)) {
			positions.push({ index: i, open: true });
			i += SNIPPET_OPEN.length;
		} else if (snippet.startsWith(SNIPPET_CLOSE, i)) {
			positions.push({ index: i, open: false });
			i += SNIPPET_CLOSE.length;
		} else {
			i += 1;
		}
	}

	// A well-formed snippet alternates open, close, open, close, ...
	// starting with an open. Any other shape (an unpaired trailing open,
	// two opens in a row, a stray leading close) is malformed — degrade
	// the entire snippet to one unmatched segment with every delimiter
	// character stripped, rather than guessing at intent.
	const wellFormed =
		positions.length % 2 === 0 && positions.every((p, idx) => p.open === (idx % 2 === 0));

	if (!wellFormed) {
		const plain = snippet.split(SNIPPET_OPEN).join('').split(SNIPPET_CLOSE).join('');
		return plain.length > 0 ? [{ text: plain, match: false }] : [];
	}

	const segments: SnippetSegment[] = [];
	let cursor = 0;
	let match = false;
	for (const p of positions) {
		const text = snippet.slice(cursor, p.index);
		if (text.length > 0) segments.push({ text, match });
		cursor = p.index + (p.open ? SNIPPET_OPEN.length : SNIPPET_CLOSE.length);
		match = p.open;
	}
	const trailing = snippet.slice(cursor);
	if (trailing.length > 0) segments.push({ text: trailing, match: false });

	return segments;
}

export type SearchVariant = 'idle' | 'loading' | 'error' | 'empty' | 'populated';

/**
 * The single decision SearchResults.svelte renders from. An empty or
 * whitespace-only query is always idle regardless of request state — the
 * precedence check that lets clearing the box always return the
 * unfiltered stream, even if a stale in-flight request resolves after the
 * clear. Otherwise the variant follows the request lifecycle: loading
 * while in flight, error on a failed request, and once ready, empty or
 * populated depending on whether any rows came back.
 */
export function searchVariant(
	query: string,
	state: 'idle' | 'loading' | 'error' | 'ready',
	resultCount: number
): SearchVariant {
	if (query.trim() === '') return 'idle';
	if (state === 'loading') return 'loading';
	if (state === 'error') return 'error';
	if (state === 'ready') return resultCount === 0 ? 'empty' : 'populated';
	return 'idle';
}

// The four locked search strings (03-UI-SPEC.md Copywriting Contract),
// frozen so this is their one definition — SearchBox.svelte and
// SearchResults.svelte render from this object (and noMatchesHeading
// below) rather than from inline literals, so the copy can never drift
// between the two components or between a component and its test.
export const searchCopy = Object.freeze({
	placeholder: 'Search this webspace',
	clearLabel: 'Clear search',
	emptyBody: 'Try a different word, or clear the search to see the full stream.',
	errorInline: 'Search is unavailable right now — try again in a moment.'
});

/**
 * The no-matches heading, with the raw query interpolated verbatim —
 * including any character that would be markup if this were ever
 * rendered as HTML. Callers must render this through Svelte's default
 * text binding (never `{@html}`); this function itself does no escaping
 * because it produces plain text content, not markup (T-03-21).
 */
export function noMatchesHeading(query: string): string {
	return `No matches for "${query}"`;
}

// --- Where a result matched, and how each source answered (M2-R2, #54) ---

import type { MatchedIn, SourceSearchStatus, SourceSearchState } from './api';

// matchedInLabel: the one wording per matched_in value — a closed
// vocabulary (docs/api.md), so an unknown value renders as itself rather
// than nothing.
export function matchedInLabel(where: MatchedIn | string): string {
	switch (where) {
		case 'title':
			return 'Title';
		case 'preview':
			return 'Preview';
		case 'body':
			return 'Body';
		case 'labels':
			return 'Labels';
		case 'attachment':
			return 'Attachment';
		default:
			return where;
	}
}

// matchedInSummary joins the labels for a row's tag: "Title · Body".
export function matchedInSummary(where: ReadonlyArray<MatchedIn | string>): string {
	return Array.from(new Set(where.map(matchedInLabel))).join(' · ');
}

// The status row's per-source sentence: the closed vocabulary
// ok | unsupported | timeout | error, with the count where there is one.
export function sourceSearchSummary(status: SourceSearchStatus): string {
	const n = status.hits;
	switch (status.status) {
		case 'ok':
			return `${n} ${n === 1 ? 'hit' : 'hits'}${status.truncated ? ' (more than shown)' : ''}`;
		case 'unsupported':
			return 'no content search';
		case 'timeout':
			return 'timed out';
		case 'error':
			return 'failed';
		default:
			return status.status;
	}
}

export function sourceSearchTone(state: SourceSearchState): 'ok' | 'muted' | 'warning' {
	switch (state) {
		case 'ok':
			return 'ok';
		case 'unsupported':
			return 'muted';
		default:
			return 'warning';
	}
}

// The per-source elapsed time, rendered for EVERY outcome (the plan's
// status row is "status, counts, elapsed" — a timeout's five seconds is
// as informative as a fast answer's milliseconds).
export function sourceSearchElapsed(ms: number): string {
	if (ms >= 1000) {
		const s = ms / 1000;
		return `${Number.isInteger(s) ? s : s.toFixed(1)}s`;
	}
	return `${ms}ms`;
}

// --- Stream-row label overflow (M3-R3, #63/#75) ---
//
// The desktop meta strip no longer wraps into the row's fixed-height clip
// zone: it is one non-wrapping line, and the tag pills clamp to a
// character budget with the remainder declared as a +N pill. A character
// budget is deliberately width-agnostic and makes no promise about any
// particular pane width: it bounds HOW MANY pills render, and the
// pills' own CSS (the one shrinkable region in the strip) ellipsizes
// what the width cannot fit, while the +N declaration and the search
// badges stay shrink-0 and always survive (PR #78 round 1). Pure and
// unit-testable, never a ResizeObserver dance.
export const LABEL_BUDGET_CHARS = 36;

export function clampLabels(
	labels: ReadonlyArray<string>,
	budget: number = LABEL_BUDGET_CHARS
): { visible: string[]; hidden: string[] } {
	const visible: string[] = [];
	const hidden: string[] = [];
	let used = 0;
	for (const label of labels) {
		// Each pill costs its text plus padding/gap (~4 chars' worth).
		const cost = label.length + 4;
		if (visible.length > 0 && used + cost > budget) {
			hidden.push(label);
			continue;
		}
		// The first label always renders (truncated by CSS if enormous) —
		// a row with one long label shows that label, never only "+1".
		visible.push(label);
		used += cost;
	}
	return { visible, hidden };
// dateRangeChipLabel renders the saved date narrowing's chip (M3-R1,
// #70): both sides, or the open-ended forms, in the reader's locale.
export function dateRangeChipLabel(from?: string, to?: string): string {
	const fmt = (d: string) =>
		new Intl.DateTimeFormat(undefined, { day: 'numeric', month: 'short', year: 'numeric' }).format(
			new Date(d + 'T00:00:00')
		);
	if (from && to) return `${fmt(from)} – ${fmt(to)}`;
	if (from) return `from ${fmt(from)}`;
	if (to) return `until ${fmt(to)}`;
	return '';
}

// intersectDateRanges resolves what a promotion PERSISTS (M3-R1, PR #79
// round 1): the saved range intersected with the live preview — a live
// side narrows or stands aside, and an absent live side keeps the saved
// side, so promotion can never widen the permanent filter. ISO calendar
// dates compare lexically, so string comparison is date comparison.
export function intersectDateRanges(
	saved: { from?: string; to?: string },
	live: { from?: string; to?: string }
): { from: string; to: string } {
	const froms = [saved.from, live.from].filter((v): v is string => Boolean(v));
	const tos = [saved.to, live.to].filter((v): v is string => Boolean(v));
	return {
		from: froms.length ? froms.reduce((a, b) => (a > b ? a : b)) : '',
		to: tos.length ? tos.reduce((a, b) => (a < b ? a : b)) : ''
	};
}

export const searchSourcesCopy = Object.freeze({
	pending: 'Searching sources…',
	failed: 'The sources could not be searched — showing what the index found.',
	unsynced: 'Not yet synced',
	findInPage: 'Find in page'
});

// --- Search-term highlighting (UI-09) ---
//
// This section is the client half of UI-09's shared term-derivation rule.
// The kernel half lives in kernel/httpapi/rendition.go's own
// highlightTerms (Task 1 of this phase) and is DELIBERATELY duplicated
// here — the two implementations must stay in step, term-for-term and
// tie-break-for-tie-break, so what the client highlights in a plain-text
// or media body never disagrees with what the kernel highlights inside a
// sandboxed iframe for the same search query.

// HIGHLIGHT_TERM_MAX_LENGTH bounds an individual term's maximum length
// (WR-02) — mirrors kernel/httpapi/rendition.go's highlightTermMaxRunes.
// Alongside the count cap and the <2-character drop below, this is the
// third bounded-work control the docstring promises for threat T-06-03;
// without it a caller could supply a single, arbitrarily long
// whitespace-free "word" that survives the split() as one term.
const HIGHLIGHT_TERM_MAX_LENGTH = 64;

/**
 * Derives the bounded, literal term set a search query highlights:
 * trims, splits on whitespace, lowercases, de-duplicates, drops any term
 * shorter than 2 characters or longer than HIGHLIGHT_TERM_MAX_LENGTH, and
 * caps the result at the first 8 terms. Returns an empty array for an
 * empty or all-dropped query. Identical rule to
 * kernel/httpapi/rendition.go's highlightTerms — see that function's doc
 * comment for why the two must never diverge.
 */
export function highlightTerms(query: string): string[] {
	const fields = query.split(/\s+/).filter((f) => f.length > 0);
	const seen = new Set<string>();
	const terms: string[] = [];
	for (const raw of fields) {
		const term = raw.toLowerCase();
		// Count Unicode code points, not UTF-16 code units (`term.length`),
		// to match the kernel's `utf8.RuneCountInString` (WR-01) — a plain
		// `.length` check counts each astral-plane character (emoji, many
		// CJK Extension B/C characters, etc.) as 2 UTF-16 units, which would
		// silently diverge from the kernel's rune-counted <2/>64 bounds for
		// any query containing them. `[...term]` iterates by code point.
		const runeCount = [...term].length;
		if (runeCount < 2 || runeCount > HIGHLIGHT_TERM_MAX_LENGTH) continue;
		if (seen.has(term)) continue;
		seen.add(term);
		terms.push(term);
		if (terms.length === 8) break;
	}
	return terms;
}

/**
 * Splits text into an ordered SnippetSegment[] — the same shape
 * parseSnippet returns, so StreamRow.svelte's existing
 * `{#each parseSnippet(snippet) as segment, i (i)}` render loop is
 * reusable verbatim against this function's output too. Case-insensitive
 * literal index scanning only: the query is never turned into a RegExp,
 * so a query carrying regex metacharacters is matched literally and can
 * never throw or over-match. At each scan position, terms are tried
 * longest-first (matching the kernel's own tie-break exactly) so a
 * longer term always wins over a shorter overlapping one — no nested or
 * duplicated segments are ever produced.
 *
 * An empty query, an empty text, or a zero-match scan degrades to a
 * single unmatched segment holding the whole text (or an empty array for
 * empty text) — never an error. Follows noMatchesHeading's discipline
 * above: this function returns data, never a markup string, so Svelte's
 * default text binding does the escaping; it is never rendered via
 * `{@html}`.
 *
 * The round-trip invariant this function guarantees: concatenating every
 * returned segment's `text` reproduces `text` exactly — no character is
 * ever lost or duplicated.
 */
export function highlightText(text: string, query: string): SnippetSegment[] {
	if (text === '') return [];

	const terms = highlightTerms(query);
	if (terms.length === 0) return [{ text, match: false }];

	// Longest-first so a longer term always wins over a shorter
	// overlapping one at the same scan position.
	const sorted = [...terms].sort((a, b) => b.length - a.length);
	const lower = text.toLowerCase();

	const segments: SnippetSegment[] = [];
	let cursor = 0;
	let i = 0;
	while (i < text.length) {
		let matchLen = 0;
		for (const term of sorted) {
			if (lower.startsWith(term, i)) {
				matchLen = term.length;
				break;
			}
		}
		if (matchLen === 0) {
			i += 1;
			continue;
		}
		if (i > cursor) segments.push({ text: text.slice(cursor, i), match: false });
		segments.push({ text: text.slice(i, i + matchLen), match: true });
		i += matchLen;
		cursor = i;
	}
	if (cursor < text.length) segments.push({ text: text.slice(cursor), match: false });
	if (segments.length === 0) segments.push({ text, match: false });

	return segments;
}

// --- Stream date markers (UI-11) ---

export interface DateMarker {
	itemId: string;
	timestampUnix: number;
	topPx: number;
	/**
	 * True for a marker whose grouping key one granularity coarser than the
	 * rendering granularity differs from the previous KEPT marker's — month
	 * for day/week candidates, year for month candidates. The first marker
	 * is always major. This is the grouping vocabulary the overlay uses to
	 * distinguish a month (or year) boundary from a plain date tick (UI-11
	 * gap closure G-06-6) — see `candidateMarkers`.
	 */
	major: boolean;
}

export type MarkerGranularity = 'day' | 'week' | 'month';

/**
 * The minimum vertical spacing, in pixels, `dateMarkers` guarantees
 * between any two returned markers. Never violated: adaptive thinning
 * (day -> week -> month) exists specifically to hold this floor, and the
 * degenerate case where even month periods are too dense enforces it by
 * construction (see `dateMarkers`'s own doc comment).
 */
const MARKER_SPACING_FLOOR_PX = 24;

/**
 * Half the tick hit-area's 16px height (06-UI-SPEC.md's documented
 * desktop-only spacing exception) — the inset `markerLaneTop` applies at
 * both ends of the track so a tick centred on its returned offset always
 * lies entirely inside the pane (UI-11 gap closure G-06-6, defect 4: a
 * marker at `topPx = 0` was previously always half-rendered above the
 * pane's top edge).
 */
export const MARKER_LANE_INSET_PX = 8;

/**
 * Whether the stream pane's scroll region actually has a scrollbar to
 * annotate — true only when the content is strictly taller than the
 * track; equal heights mean no scrollbar exists, and a non-positive track
 * is degenerate. This is the missing guard behind defect 5 (UI-11 gap
 * closure G-06-6): a multi-date stream shorter than its pane has nothing
 * to navigate between via a scrollbar, so the ruler must render nothing.
 * Deliberately kept a separate predicate rather than a third parameter on
 * `dateMarkers` — every existing `dateMarkers` call site and assertion
 * stays intact, and the gate is legible at the one place that renders.
 */
export function streamScrolls(contentHeightPx: number, trackHeightPx: number): boolean {
	if (trackHeightPx <= 0) return false;
	return contentHeightPx > trackHeightPx;
}

/**
 * Remaps a marker's raw, index-proportional track position into an inset
 * lane: 0 maps to `MARKER_LANE_INSET_PX`, `trackHeightPx` maps to
 * `trackHeightPx - MARKER_LANE_INSET_PX`, and the mapping is monotonic and
 * order-preserving in between — an affine remap of the range
 * `[0, trackHeightPx]` onto `[inset, trackHeightPx - inset]`. This closes
 * defect 4 without touching `dateMarkers`'s own position formula:
 * `dateMarkers` still returns true index-proportional positions, and edge
 * safety is applied here, at render time. The remap compresses spacing by
 * a factor of `(trackHeight - 2*inset) / trackHeight` — a fraction of a
 * pixel at realistic pane heights — so the 24px floor holds in spirit.
 *
 * A non-positive track height returns the inset rather than dividing by
 * zero. A track shorter than twice the inset is a further degenerate case:
 * the usable range would go negative and the mapping would invert (a
 * larger `topPx` producing a SMALLER result) — clamped to zero-width
 * instead, so every position in a too-short track resolves to the same
 * inset offset rather than an inverted one.
 */
export function markerLaneTop(topPx: number, trackHeightPx: number): number {
	if (trackHeightPx <= 0) return MARKER_LANE_INSET_PX;
	const usable = Math.max(0, trackHeightPx - 2 * MARKER_LANE_INSET_PX);
	const fraction = topPx / trackHeightPx;
	return MARKER_LANE_INSET_PX + fraction * usable;
}

/**
 * UTC calendar-day key. Unix timestamps are seconds since the epoch in
 * UTC, so floor-dividing by the number of seconds in a day yields the same
 * day boundary `formatItemDate`'s `timeZone: 'UTC'` formatter would show —
 * no `Date` construction needed, and no risk of the two ever disagreeing
 * about which day an item belongs to.
 */
function dayKey(timestampUnix: number): number {
	return Math.floor(timestampUnix / 86400);
}

/**
 * ISO-8601 week key (`{iso-week-year}-W{week}`), computed in UTC. Follows
 * the standard "nearest Thursday" ISO week algorithm: shifting each date to
 * the Thursday of its own week makes the week-year and week-number
 * unambiguous even across a Dec/Jan boundary (a Thursday-anchored week
 * never crosses into a different ISO year than the days around it).
 */
function isoWeekKey(timestampUnix: number): string {
	const source = new Date(timestampUnix * 1000);
	const d = new Date(
		Date.UTC(source.getUTCFullYear(), source.getUTCMonth(), source.getUTCDate())
	);
	const dayNum = (d.getUTCDay() + 6) % 7; // Monday=0 .. Sunday=6
	d.setUTCDate(d.getUTCDate() - dayNum + 3); // shift to this week's Thursday

	const firstThursday = new Date(Date.UTC(d.getUTCFullYear(), 0, 4));
	const firstDayNum = (firstThursday.getUTCDay() + 6) % 7;
	firstThursday.setUTCDate(firstThursday.getUTCDate() - firstDayNum + 3);

	const weekNum =
		1 + Math.round((d.getTime() - firstThursday.getTime()) / (7 * 24 * 60 * 60 * 1000));
	return `${d.getUTCFullYear()}-W${weekNum}`;
}

/** UTC calendar-month key (`{year}-{month}`), month `0`-indexed. */
function monthKey(timestampUnix: number): string {
	const d = new Date(timestampUnix * 1000);
	return `${d.getUTCFullYear()}-${d.getUTCMonth()}`;
}

/**
 * UTC calendar-year key. The "coarser key" for month-granularity
 * candidates, alongside `monthKey` above (the coarser key for day/week
 * candidates) — see `candidateMarkers`'s `majorKeyOf` parameter.
 */
function yearKey(timestampUnix: number): number {
	return new Date(timestampUnix * 1000).getUTCFullYear();
}

/**
 * Builds one candidate marker per new period key, walking `items` in the
 * order supplied — never re-sorted, since the marker's whole job is to
 * describe the stream's actual order. Position is index-proportional
 * (`(index / items.length) * trackHeightPx`), valid because the stream is
 * a non-virtualised list of fixed-height rows.
 *
 * `majorKeyOf` computes the "coarser key" — one granularity above `keyOf`
 * (month for day/week candidates, year for month candidates) — and a
 * marker is `major` when its coarser key differs from the previously
 * pushed marker's, or when it is the first marker pushed. Since every
 * marker this function pushes is provisionally "kept" at this stage (only
 * `enforceSpacingFloor`'s degenerate backstop may thin further, and that
 * function carries the flag through untouched per its own doc comment),
 * "the previous pushed marker" and "the previous KEPT marker" coincide
 * here.
 */
function candidateMarkers(
	items: StreamItem[],
	trackHeightPx: number,
	keyOf: (timestampUnix: number) => string | number,
	majorKeyOf: (timestampUnix: number) => string | number
): DateMarker[] {
	const markers: DateMarker[] = [];
	let lastKey: string | number | null = null;
	let lastMajorKey: string | number | null = null;
	items.forEach((item, index) => {
		const key = keyOf(item.timestamp_unix);
		if (key !== lastKey) {
			const majorKey = majorKeyOf(item.timestamp_unix);
			markers.push({
				itemId: item.id,
				timestampUnix: item.timestamp_unix,
				topPx: (index / items.length) * trackHeightPx,
				major: lastMajorKey === null || majorKey !== lastMajorKey
			});
			lastKey = key;
			lastMajorKey = majorKey;
		}
	});
	return markers;
}

/** True when every adjacent pair of a (position-sorted-by-construction) candidate list is at least the spacing floor apart. */
function satisfiesSpacingFloor(markers: DateMarker[]): boolean {
	for (let i = 1; i < markers.length; i += 1) {
		if (markers[i].topPx - markers[i - 1].topPx < MARKER_SPACING_FLOOR_PX) return false;
	}
	return true;
}

/**
 * The degenerate-case backstop: the first candidate is always kept, and
 * every subsequent candidate is kept only if it lands at least the
 * spacing floor away from the last KEPT marker (not the last candidate) —
 * so the floor holds by construction rather than being silently
 * abandoned when even month periods are too dense.
 */
function enforceSpacingFloor(markers: DateMarker[]): DateMarker[] {
	if (markers.length === 0) return markers;
	const kept: DateMarker[] = [markers[0]];
	for (let i = 1; i < markers.length; i += 1) {
		const candidate = markers[i];
		const last = kept[kept.length - 1];
		if (candidate.topPx - last.topPx >= MARKER_SPACING_FLOOR_PX) {
			kept.push(candidate);
		}
	}
	return kept;
}

/**
 * Derives the stream pane's date-tick overlay markers (UI-11) from the
 * already-in-memory, already-visible stream items — no new fetch. Pure and
 * side-effect-free; the same input always yields the same output.
 *
 * Rules:
 * - Fewer than two items, a non-positive track height, or every item
 *   sharing one UTC calendar date all return an empty array — a
 *   single-date stream has nothing to navigate between, so it renders no
 *   markers at all rather than one useless tick.
 * - Granularity is adaptive: day candidates are computed first and used
 *   if every adjacent pair is at least `MARKER_SPACING_FLOOR_PX` apart;
 *   otherwise week candidates are tried, then month candidates. The first
 *   granularity that satisfies the floor wins.
 * - If even month periods violate the floor, month-period candidates are
 *   thinned by `enforceSpacingFloor` (dropping any candidate landing
 *   within the floor of the last KEPT marker), so the floor holds by
 *   construction in the degenerate case rather than being silently
 *   abandoned.
 * - The first candidate at every granularity is always kept; the floor is
 *   only ever evaluated against the previously kept marker.
 */
export function dateMarkers(items: StreamItem[], trackHeightPx: number): DateMarker[] {
	if (items.length < 2 || trackHeightPx <= 0) return [];

	const firstDay = dayKey(items[0].timestamp_unix);
	if (items.every((item) => dayKey(item.timestamp_unix) === firstDay)) return [];

	const dayMarkers = candidateMarkers(items, trackHeightPx, dayKey, monthKey);
	if (satisfiesSpacingFloor(dayMarkers)) return dayMarkers;

	const weekMarkers = candidateMarkers(items, trackHeightPx, isoWeekKey, monthKey);
	if (satisfiesSpacingFloor(weekMarkers)) return weekMarkers;

	const monthMarkers = candidateMarkers(items, trackHeightPx, monthKey, yearKey);
	if (satisfiesSpacingFloor(monthMarkers)) return monthMarkers;

	return enforceSpacingFloor(monthMarkers);
}
