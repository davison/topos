import { describe, it, expect } from 'vitest';
import {
	formatItemDate,
	formatFidelity,
	formatRelativeTime,
	healthTone,
	parseSnippet,
	searchVariant,
	searchCopy,
	noMatchesHeading
} from './format';
import { SNIPPET_OPEN, SNIPPET_CLOSE } from './api';
import type { SourceStatus } from './api';

function makeSource(overrides: Partial<SourceStatus> = {}): SourceStatus {
	return {
		name: 'paperless',
		source_type: 'paperless',
		display_name: 'paperless-ngx',
		plugin: 'topos-plugin-paperless',
		reachable: true,
		syncing: false,
		last_status: 'ok',
		last_sync_unix: 1700000000,
		last_error: '',
		...overrides
	};
}

describe('formatItemDate', () => {
	it('formats a UTC timestamp using its UTC calendar day', () => {
		const result = formatItemDate(1704067200); // 2024-01-01T00:00:00Z
		expect(result).toContain('2024');
		expect(result).toMatch(/Jan/i);
		expect(result).not.toContain('2023');
	});

	it('pins the calendar day to UTC even for a timestamp a negative-offset zone would render as the previous day', () => {
		const midnightUtc = Date.UTC(2024, 0, 1, 0, 0, 0) / 1000; // 2024-01-01T00:00:00Z

		// Sanity-check the premise this test guards against: in a
		// negative-offset zone (America/Los_Angeles, UTC-8) this exact
		// instant is still the previous calendar day.
		const laFormatted = new Intl.DateTimeFormat('en-US', {
			day: 'numeric',
			month: 'short',
			year: 'numeric',
			timeZone: 'America/Los_Angeles'
		}).format(new Date(midnightUtc * 1000));
		expect(laFormatted).toContain('2023');

		// formatItemDate must not fall into that trap — it pins to UTC
		// unconditionally, regardless of the viewer's local timezone.
		const result = formatItemDate(midnightUtc);
		expect(result).toContain('2024');
		expect(result).not.toContain('2023');
	});

	it('returns an empty string for a falsy timestamp', () => {
		expect(formatItemDate(0)).toBe('');
	});
});

describe('formatFidelity', () => {
	it('maps each known fidelity value to its fixed display label', () => {
		expect(formatFidelity('exact')).toBe('exact');
		expect(formatFidelity('anchored')).toBe('anchored');
		expect(formatFidelity('conversation-only')).toBe('conversation-only');
	});

	it('falls back to the raw value for an unrecognized fidelity', () => {
		expect(formatFidelity('mystery')).toBe('mystery');
	});
});

describe('formatRelativeTime', () => {
	it('returns a relative string carrying a minute unit for a timestamp 90 seconds ago', () => {
		const nowUnix = Math.floor(Date.now() / 1000);
		const result = formatRelativeTime(nowUnix - 90);
		expect(result).toMatch(/minute/i);
	});

	it('returns the empty string for a zero timestamp rather than a 1970 date', () => {
		expect(formatRelativeTime(0)).toBe('');
	});
});

describe('healthTone', () => {
	it('maps reachable + last_status ok to the success tone', () => {
		expect(healthTone(makeSource({ reachable: true, last_status: 'ok' }))).toBe('success');
	});

	it('maps reachable + last_status error to the warning tone', () => {
		expect(healthTone(makeSource({ reachable: true, last_status: 'error' }))).toBe('warning');
	});

	it('maps unreachable to the destructive tone', () => {
		expect(healthTone(makeSource({ reachable: false, last_status: 'error' }))).toBe('destructive');
	});

	it('maps last_status "" (never synced) to the unknown tone, never success', () => {
		const tone = healthTone(makeSource({ last_status: '', last_sync_unix: 0, reachable: true }));
		expect(tone).toBe('unknown');
		expect(tone).not.toBe('success');
	});

	it('never renders the unknown state as success even when paired with reachable:false', () => {
		const tone = healthTone(makeSource({ last_status: '', last_sync_unix: 0, reachable: false }));
		expect(tone).not.toBe('success');
	});
});

describe('parseSnippet', () => {
	it('returns a single unmatched segment for a string with no delimiters', () => {
		expect(parseSnippet('plain text, no match')).toEqual([
			{ text: 'plain text, no match', match: false }
		]);
	});

	it('returns three alternating segments for one delimited region', () => {
		const snippet = `lead ${SNIPPET_OPEN}MATCH${SNIPPET_CLOSE} trail`;
		expect(parseSnippet(snippet)).toEqual([
			{ text: 'lead ', match: false },
			{ text: 'MATCH', match: true },
			{ text: ' trail', match: false }
		]);
	});

	it('returns five alternating segments for two delimited regions', () => {
		const snippet = `a${SNIPPET_OPEN}ONE${SNIPPET_CLOSE}b${SNIPPET_OPEN}TWO${SNIPPET_CLOSE}c`;
		expect(parseSnippet(snippet)).toEqual([
			{ text: 'a', match: false },
			{ text: 'ONE', match: true },
			{ text: 'b', match: false },
			{ text: 'TWO', match: true },
			{ text: 'c', match: false }
		]);
	});

	it('returns an empty array for an empty string', () => {
		expect(parseSnippet('')).toEqual([]);
	});

	it('never emits a segment carrying a delimiter character', () => {
		const snippet = `a${SNIPPET_OPEN}b${SNIPPET_OPEN}c${SNIPPET_CLOSE}d`; // malformed: two opens
		const segments = parseSnippet(snippet);
		for (const segment of segments) {
			expect(segment.text).not.toContain(SNIPPET_OPEN);
			expect(segment.text).not.toContain(SNIPPET_CLOSE);
		}
	});

	it('handles a snippet that begins with a matched region (no leading unmatched segment)', () => {
		const snippet = `${SNIPPET_OPEN}MATCH${SNIPPET_CLOSE} trail`;
		expect(parseSnippet(snippet)).toEqual([
			{ text: 'MATCH', match: true },
			{ text: ' trail', match: false }
		]);
	});

	it('handles a snippet that ends with a matched region (no trailing unmatched segment)', () => {
		const snippet = `lead ${SNIPPET_OPEN}MATCH${SNIPPET_CLOSE}`;
		expect(parseSnippet(snippet)).toEqual([
			{ text: 'lead ', match: false },
			{ text: 'MATCH', match: true }
		]);
	});

	it('handles a snippet that is entirely a matched region', () => {
		const snippet = `${SNIPPET_OPEN}MATCH${SNIPPET_CLOSE}`;
		expect(parseSnippet(snippet)).toEqual([{ text: 'MATCH', match: true }]);
	});

	it('handles an elision-adjacent match without splitting or leaking the ellipsis character', () => {
		const snippet = `…lead ${SNIPPET_OPEN}MATCH${SNIPPET_CLOSE} trail…`;
		expect(parseSnippet(snippet)).toEqual([
			{ text: '…lead ', match: false },
			{ text: 'MATCH', match: true },
			{ text: ' trail…', match: false }
		]);
	});

	it('never splits a multi-byte character adjacent to a delimiter', () => {
		// A surrogate-pair emoji directly touching both delimiters — if the
		// parser cut mid-character, one of these segments would carry a
		// lone unpaired surrogate instead of the intact emoji.
		const emoji = '😀';
		const snippet = `${emoji}${SNIPPET_OPEN}MATCH${SNIPPET_CLOSE}${emoji}`;
		expect(parseSnippet(snippet)).toEqual([
			{ text: emoji, match: false },
			{ text: 'MATCH', match: true },
			{ text: emoji, match: false }
		]);
	});
});

describe('searchVariant', () => {
	it('returns idle for an empty query regardless of state', () => {
		expect(searchVariant('', 'ready', 5)).toBe('idle');
		expect(searchVariant('', 'loading', 0)).toBe('idle');
		expect(searchVariant('', 'error', 0)).toBe('idle');
	});

	it('returns idle for a whitespace-only query regardless of state', () => {
		expect(searchVariant('   ', 'ready', 5)).toBe('idle');
	});

	it('returns loading while a request for a non-empty query is in flight', () => {
		expect(searchVariant('boiler', 'loading', 0)).toBe('loading');
	});

	it('returns error when the last request for the current non-empty query failed', () => {
		expect(searchVariant('boiler', 'error', 0)).toBe('error');
	});

	it('returns empty for a non-empty query whose completed request returned zero results', () => {
		expect(searchVariant('boiler', 'ready', 0)).toBe('empty');
	});

	it('returns populated for a non-empty query whose completed request returned at least one result', () => {
		expect(searchVariant('boiler', 'ready', 1)).toBe('populated');
	});
});

describe('searchVariant state matrix (one assertion per row)', () => {
	it('an empty query is idle whatever the request state, so clearing the box always returns the stream', () => {
		expect(searchVariant('', 'idle', 0)).toBe('idle');
		expect(searchVariant('', 'loading', 0)).toBe('idle');
		expect(searchVariant('', 'error', 0)).toBe('idle');
		expect(searchVariant('', 'ready', 5)).toBe('idle');
	});

	it('a whitespace-only query is idle whatever the request state', () => {
		expect(searchVariant('   ', 'loading', 0)).toBe('idle');
		expect(searchVariant('   ', 'ready', 5)).toBe('idle');
	});

	it('a non-empty query in flight is loading — the variant that drives the skeleton rows, never a disabled input', () => {
		expect(searchVariant('boiler', 'loading', 0)).toBe('loading');
	});

	it('a failed request for the current query is error', () => {
		expect(searchVariant('boiler', 'error', 0)).toBe('error');
	});

	it('a completed request with zero results is empty, distinct from idle and from error', () => {
		const empty = searchVariant('boiler', 'ready', 0);
		expect(empty).toBe('empty');
		expect(empty).not.toBe('idle');
		expect(empty).not.toBe('error');
	});

	it('a completed request with exactly one result is populated, and fifty results is still populated with no additional state', () => {
		expect(searchVariant('boiler', 'ready', 1)).toBe('populated');
		expect(searchVariant('boiler', 'ready', 50)).toBe('populated');
	});
});

describe('searchCopy', () => {
	it('matches the UI-SPEC Copywriting Contract character for character', () => {
		expect(searchCopy.placeholder).toBe('Search this webspace');
		expect(searchCopy.clearLabel).toBe('Clear search');
		expect(searchCopy.emptyBody).toBe(
			'Try a different word, or clear the search to see the full stream.'
		);
		// The em dash (U+2014) between "now" and "try again" is load-bearing —
		// a plain hyphen would silently diverge from the locked copy.
		expect(searchCopy.errorInline).toBe(
			'Search is unavailable right now — try again in a moment.'
		);
	});
});

describe('noMatchesHeading', () => {
	it('interpolates the query verbatim between straight double quotes', () => {
		expect(noMatchesHeading('boiler')).toBe('No matches for "boiler"');
	});

	it('interpolates the query verbatim even when it carries markup-like characters', () => {
		expect(noMatchesHeading('<script>')).toBe('No matches for "<script>"');
		expect(noMatchesHeading('a "quoted" term')).toBe('No matches for "a "quoted" term"');
	});
});
