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
 */
export function healthTone(source: SourceStatus): HealthTone {
	if (source.last_status === '') return 'unknown';
	if (!source.reachable) return 'destructive';
	if (source.last_status === 'error') return 'warning';
	return 'success';
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
 * and the overflow trigger's own width.
 *
 * `reservedWidth` is subtracted from `availableWidth` first. If every
 * chip's combined width fits that budget, every chip is visible and no
 * overflow trigger is needed — the trigger's width is never charged in
 * that case, since nothing is deferred to it. Only once the full set
 * does NOT fit is the budget further reduced by `overflowTriggerWidth`
 * (the trigger itself now occupies row space) and chips are accumulated
 * in order against that reduced budget.
 *
 * Accumulation uses a strict `>` comparison, so a chip that would exceed
 * the budget by any fraction of a pixel is excluded rather than rendered
 * half-clipped at the row's trailing edge — this is the answer to the
 * sub-pixel measurement question. The function is pure and
 * side-effect-free: a repeated pass over unchanged inputs yields an
 * identical count, and a budget that goes negative (reserved and trigger
 * widths alone exceed the available width) naturally floors the result
 * at zero rather than a negative count.
 */
export function visibleChipCount(
	chipWidths: number[],
	availableWidth: number,
	reservedWidth: number,
	overflowTriggerWidth: number
): number {
	const budget = availableWidth - reservedWidth;
	const total = chipWidths.reduce((sum, width) => sum + width, 0);
	if (total <= budget) return chipWidths.length;

	const overflowBudget = budget - overflowTriggerWidth;
	let used = 0;
	let count = 0;
	for (const width of chipWidths) {
		if (used + width > overflowBudget) break;
		used += width;
		count += 1;
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

// --- Search-term highlighting (UI-09) ---
//
// This section is the client half of UI-09's shared term-derivation rule.
// The kernel half lives in kernel/httpapi/rendition.go's own
// highlightTerms (Task 1 of this phase) and is DELIBERATELY duplicated
// here — the two implementations must stay in step, term-for-term and
// tie-break-for-tie-break, so what the client highlights in a plain-text
// or media body never disagrees with what the kernel highlights inside a
// sandboxed iframe for the same search query.

/**
 * Derives the bounded, literal term set a search query highlights:
 * trims, splits on whitespace, lowercases, de-duplicates, drops any term
 * shorter than 2 characters, and caps the result at the first 8 terms.
 * Returns an empty array for an empty or all-dropped query. Identical
 * rule to kernel/httpapi/rendition.go's highlightTerms — see that
 * function's doc comment for why the two must never diverge.
 */
export function highlightTerms(query: string): string[] {
	const fields = query.split(/\s+/).filter((f) => f.length > 0);
	const seen = new Set<string>();
	const terms: string[] = [];
	for (const raw of fields) {
		const term = raw.toLowerCase();
		if (term.length < 2) continue;
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
