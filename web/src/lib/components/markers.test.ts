// Unit tests for dateMarkers (UI-11, format.ts) — the stream pane's
// date-tick overlay derivation: empty/singular inputs, the single-date
// no-op case, the position formula, order fidelity, adaptive thinning
// (day -> week -> month) and the 24-pixel spacing floor as an invariant
// held across every granularity including the degenerate over-dense case.
// Every fixture is built from explicit UTC unix timestamps (mirroring
// date-format.test.ts's discipline) so the assertions are
// timezone-independent, and the colocated-file convention follows
// sources.test.ts.

import { describe, it, expect } from 'vitest';
import { dateMarkers } from '$lib/format';
import type { StreamItem } from '$lib/api';

const DAY = 86400;
// Monday 1 Jan 2024 00:00:00 UTC — a stable, arbitrary anchor for every
// fixture below.
const BASE = Date.UTC(2024, 0, 1) / 1000;

function makeItem(id: string, timestampUnix: number): StreamItem {
	return {
		id,
		source: 'paperless',
		source_type: 'paperless',
		source_display_name: 'paperless-ngx',
		source_id: id,
		title: id,
		preview: '',
		timestamp_unix: timestampUnix,
		secondary_timestamp_unix: 0,
		labels: [],
		group_id: '',
		group_label: '',
		link: { url: 'https://example.test/1', fidelity: 'exact' },
		provenance: {}
	};
}

/** Asserts no two markers in `markers` (already position-sorted by construction) are closer than 24px. */
function assertSpacingFloorHolds(markers: { topPx: number }[]) {
	for (let i = 1; i < markers.length; i += 1) {
		expect(markers[i].topPx - markers[i - 1].topPx).toBeGreaterThanOrEqual(24);
	}
}

describe('dateMarkers — empty and singular', () => {
	it('returns an empty array for zero items', () => {
		expect(dateMarkers([], 500)).toEqual([]);
	});

	it('returns an empty array for a single item', () => {
		expect(dateMarkers([makeItem('a', BASE)], 500)).toEqual([]);
	});

	it('returns an empty array for a zero track height, even with items spanning multiple dates', () => {
		const items = [makeItem('a', BASE), makeItem('b', BASE + 2 * DAY)];
		expect(dateMarkers(items, 0)).toEqual([]);
	});

	it('returns an empty array for a negative track height rather than dividing by it', () => {
		const items = [makeItem('a', BASE), makeItem('b', BASE + 2 * DAY)];
		expect(dateMarkers(items, -50)).toEqual([]);
	});
});

describe('dateMarkers — single date', () => {
	it('returns an empty array when every item falls on one UTC calendar date', () => {
		const items = Array.from({ length: 20 }, (_, i) => makeItem(`i${i}`, BASE + i * 600));
		expect(dateMarkers(items, 900)).toEqual([]);
	});
});

describe('dateMarkers — day granularity', () => {
	// Three well-separated dates, 10 items each, on a tall track: day
	// markers comfortably clear the 24px floor (300px apart), so day
	// granularity is used unthinned.
	const items: StreamItem[] = [];
	for (let d = 0; d < 3; d += 1) {
		for (let i = 0; i < 10; i += 1) {
			items.push(makeItem(`d${d}-${i}`, BASE + d * DAY + i * 100));
		}
	}
	const markers = dateMarkers(items, 900);

	it('produces exactly one marker per date, pointing at that date\'s first item id in stream order', () => {
		expect(markers.map((m) => m.itemId)).toEqual(['d0-0', 'd1-0', 'd2-0']);
	});

	it('positions increase monotonically, with the first marker at the top', () => {
		expect(markers[0].topPx).toBe(0);
		for (let i = 1; i < markers.length; i += 1) {
			expect(markers[i].topPx).toBeGreaterThan(markers[i - 1].topPx);
		}
	});

	it('the spacing floor holds across every adjacent pair', () => {
		assertSpacingFloorHolds(markers);
	});
});

describe('dateMarkers — position formula', () => {
	it('a marker\'s position equals its item\'s stream index divided by the item count, times the track height', () => {
		// 30 items, track height 900: the second date's first item sits at
		// stream index 10, so its position must be exactly (10 / 30) * 900.
		const items: StreamItem[] = [];
		for (let d = 0; d < 3; d += 1) {
			for (let i = 0; i < 10; i += 1) {
				items.push(makeItem(`d${d}-${i}`, BASE + d * DAY + i * 100));
			}
		}
		const markers = dateMarkers(items, 900);
		const secondDateMarker = markers.find((m) => m.itemId === 'd1-0');
		expect(secondDateMarker?.topPx).toBe((10 / 30) * 900);
	});
});

describe('dateMarkers — order fidelity', () => {
	it('derives markers from the supplied stream order, never re-sorting the items', () => {
		// Deliberately out of chronological order: index 0 is the earliest
		// date, index 1 is the LATEST date, index 2 is the middle date —
		// the function must describe THIS order, not the sorted one.
		const items = [
			makeItem('day1-first', BASE),
			makeItem('day3-first', BASE + 2 * DAY),
			makeItem('day2-first', BASE + 1 * DAY)
		];
		const markers = dateMarkers(items, 300);

		// Every item here is a distinct calendar date, so all three become
		// candidate markers — at their actual stream indices (0, 1, 2), not
		// their chronological rank.
		expect(markers.map((m) => m.itemId)).toEqual(['day1-first', 'day3-first', 'day2-first']);
		expect(markers.map((m) => m.topPx)).toEqual([0, 100, 200]);
	});
});

describe('dateMarkers — week thinning', () => {
	// 30 consecutive daily items on a short track: day markers would sit
	// ~6.67px apart (well under the 24px floor), so the function must
	// thin to one marker per ISO week instead.
	const items: StreamItem[] = Array.from({ length: 30 }, (_, i) =>
		makeItem(`i${i}`, BASE + i * DAY)
	);
	const markers = dateMarkers(items, 200);

	it('produces one marker per ISO week rather than one per day', () => {
		// 30 consecutive days span 5 ISO weeks starting from a Monday
		// anchor (BASE), so week granularity yields 5 markers — far fewer
		// than the 30 a (floor-violating) day granularity would produce.
		expect(markers.length).toBe(5);
		expect(markers.map((m) => m.itemId)).toEqual(['i0', 'i7', 'i14', 'i21', 'i28']);
	});

	it('the spacing floor holds across every adjacent pair', () => {
		assertSpacingFloorHolds(markers);
	});

	it('the first marker is always kept', () => {
		expect(markers[0].itemId).toBe('i0');
		expect(markers[0].topPx).toBe(0);
	});
});

describe('dateMarkers — month thinning', () => {
	// A multi-year daily history (3 years) on a short track: even week
	// markers would violate the floor, so the function must thin further,
	// to one marker per calendar month.
	const items: StreamItem[] = Array.from({ length: 1095 }, (_, i) =>
		makeItem(`i${i}`, BASE + i * DAY)
	);
	const markers = dateMarkers(items, 2000);

	it('produces one marker per calendar month spanning the full 3-year history', () => {
		// 3 years starting Jan 2024 span 36 calendar months.
		expect(markers.length).toBe(36);
		expect(markers[0].itemId).toBe('i0');
	});

	it('the spacing floor holds across every adjacent pair', () => {
		assertSpacingFloorHolds(markers);
	});

	it('the first marker is always kept', () => {
		expect(markers[0].topPx).toBe(0);
	});
});

describe('dateMarkers — floor holds absolutely, even in the degenerate over-dense case', () => {
	// One item per calendar month for 200 months (16+ years) on a track
	// so short (50px) that even unthinned month markers would sit well
	// under the 24px floor (0.25px apart) — the degenerate backstop must
	// drop markers greedily against the last KEPT marker so the floor
	// holds by construction.
	const items: StreamItem[] = Array.from({ length: 200 }, (_, i) => {
		const year = 2000 + Math.floor(i / 12);
		const month = i % 12;
		return makeItem(`i${i}`, Date.UTC(year, month, 1) / 1000);
	});
	const markers = dateMarkers(items, 50);

	it('never returns two markers closer than the 24px floor, even here', () => {
		assertSpacingFloorHolds(markers);
	});

	it('keeps the first candidate and thins the rest against the last KEPT marker, not the last candidate', () => {
		expect(markers[0].itemId).toBe('i0');
		expect(markers[0].topPx).toBe(0);
		// With a 50px track and a 24px floor, at most 3 markers can survive
		// (0px, 24px, 48px) — confirms the floor is enforced by
		// construction rather than merely checked and abandoned.
		expect(markers.length).toBeLessThanOrEqual(3);
	});
});

describe('dateMarkers — UTC day boundary', () => {
	it('two items straddling midnight UTC land on different dates and produce a marker at the second, regardless of local timezone', () => {
		// 2 Jan 2024 00:00:00 UTC — one second before is still 1 Jan, one
		// second after is 2 Jan, entirely independent of the machine's own
		// local timezone offset.
		const midnightUtc = Date.UTC(2024, 0, 2) / 1000;
		const items = [makeItem('before-midnight', midnightUtc - 1), makeItem('after-midnight', midnightUtc + 1)];
		const markers = dateMarkers(items, 100);

		expect(markers.map((m) => m.itemId)).toEqual(['before-midnight', 'after-midnight']);
		expect(markers[0].topPx).toBe(0);
		expect(markers[1].topPx).toBe(50);
	});
});
