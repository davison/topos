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
