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

/** Config names of every source currently mid-sync. */
export function syncingSourceTypes(sources: SourceStatus[]): string[] {
	return sources.filter((s) => s.syncing).map((s) => s.source_type);
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

// --- Source filter (D-09 / A-UI-02) ---

/**
 * Resolves a requested source-type filter (typically the URL's `source`
 * query parameter) against the configured sources. A value that names no
 * configured source — an unrecognised or stale bookmark — degrades to
 * `null` (no filter, i.e. "All") rather than an empty list or an error
 * (T-02-17).
 */
export function resolveSourceFilter(requested: string | null, sources: SourceStatus[]): string | null {
	if (!requested) return null;
	return sources.some((s) => s.source_type === requested) ? requested : null;
}

/** Narrows a stream's items to one source_type; `null` returns every item, unchanged in order. */
export function filterItemsBySource(items: StreamItem[], sourceType: string | null): StreamItem[] {
	if (sourceType === null) return items;
	return items.filter((item) => item.source_type === sourceType);
}

export type StreamVariant = 'sync-failed' | 'empty' | 'empty-filtered' | 'populated';

/**
 * The single decision StreamList.svelte renders from (once loaded): a
 * failed sync with zero items always wins, filtered or not — computed
 * from the response's own aggregate sync status and its *unfiltered*
 * item count, never the filtered subset, so a filter can never mask a
 * sync failure (T-02-16, PLAN.md prohibitions). Only once that's ruled
 * out does an empty filtered view get its own distinct copy from an
 * empty unfiltered one.
 */
export function streamVariant(response: StreamResponse, selectedSource: string | null): StreamVariant {
	if (response.sync.status === 'error' && response.items.length === 0) return 'sync-failed';
	if (filterItemsBySource(response.items, selectedSource).length === 0) {
		return selectedSource === null ? 'empty' : 'empty-filtered';
	}
	return 'populated';
}

// --- Staleness (D-10) ---

/** source_types considered stale — currently unreachable, per the live sources response. */
export function staleSourceTypes(sources: SourceStatus[]): Set<string> {
	return new Set(sources.filter((s) => !s.reachable).map((s) => s.source_type));
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
