import { describe, it, expect } from 'vitest';
import { formatItemDate, formatFidelity } from './format';

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
