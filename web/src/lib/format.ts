// Deterministic UTC date formatting for date-only source timestamps.
//
// paperless-ngx's `created` field is a date-only value stored as midnight
// UTC. If we format it in the browser's local timezone, a user west of UTC
// (any negative offset) would see the previous day for documents dated on
// the boundary — the calendar day must follow the source, not the viewer's
// clock, so `timeZone: 'UTC'` is pinned unconditionally here rather than
// left to the locale's default.

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
