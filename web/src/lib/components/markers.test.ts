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
import { dateMarkers, streamScrolls, markerLaneTop, MARKER_LANE_INSET_PX } from '$lib/format';
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

// --- streamScrolls (UI-11 gap closure G-06-6) ---
//
// The missing guard behind defect 5: a multi-date stream shorter than its
// pane has no scrollbar, so the ruler has nothing to annotate and must
// render nothing.

describe('streamScrolls', () => {
	it('is true only when content is strictly taller than the track', () => {
		expect(streamScrolls(1000, 500)).toBe(true);
	});

	it('is false when content height equals track height — no scrollbar exists at equality', () => {
		expect(streamScrolls(500, 500)).toBe(false);
	});

	it('is false when content is shorter than the track', () => {
		expect(streamScrolls(300, 500)).toBe(false);
	});

	it('is false for a zero track height', () => {
		expect(streamScrolls(1000, 0)).toBe(false);
	});

	it('is false for a negative track height', () => {
		expect(streamScrolls(1000, -50)).toBe(false);
	});
});

// --- markerLaneTop (UI-11 gap closure G-06-6) ---
//
// Closes defect 4 (a tick permanently half-rendered above the pane's top
// edge) without touching dateMarkers' own index-proportional position
// formula: the edge safety is applied at render time by remapping the raw
// position into an inset lane.

describe('markerLaneTop', () => {
	it('maps 0 to the inset', () => {
		expect(markerLaneTop(0, 900)).toBe(MARKER_LANE_INSET_PX);
	});

	it('maps trackHeightPx to trackHeightPx - inset', () => {
		expect(markerLaneTop(900, 900)).toBe(900 - MARKER_LANE_INSET_PX);
	});

	it('is monotonic and order-preserving in between', () => {
		const a = markerLaneTop(200, 900);
		const b = markerLaneTop(500, 900);
		const c = markerLaneTop(800, 900);
		expect(a).toBeLessThan(b);
		expect(b).toBeLessThan(c);
	});

	it('returns the inset for a non-positive track height rather than dividing by zero', () => {
		expect(markerLaneTop(50, 0)).toBe(MARKER_LANE_INSET_PX);
		expect(markerLaneTop(50, -100)).toBe(MARKER_LANE_INSET_PX);
	});

	it('clamps rather than inverts when the track is shorter than twice the inset', () => {
		const shortTrack = MARKER_LANE_INSET_PX; // half of 2*inset
		const top = markerLaneTop(0, shortTrack);
		const bottom = markerLaneTop(shortTrack, shortTrack);
		// Clamped, not inverted: the position at the track's end must never
		// resolve above the position at the track's start.
		expect(bottom).toBeGreaterThanOrEqual(top);
		expect(top).toBe(MARKER_LANE_INSET_PX);
		expect(bottom).toBe(MARKER_LANE_INSET_PX);
	});
});

// --- major/minor hierarchy (UI-11 gap closure G-06-6) ---
//
// The grouping vocabulary the flat dash set lacked: a month boundary (day
// and week granularity) or year boundary (month granularity) now reads
// differently from a plain date tick.

describe('dateMarkers — major flag', () => {
	it('is true for the first marker, and true only when the UTC month changes, at day granularity', () => {
		// Four well-separated day-groups spanning a month boundary: Jan 30,
		// Jan 31, Feb 1, Feb 2 2024 — 10 items each, on a tall track, so day
		// granularity is used unthinned (400px apart, well clear of the
		// 24px floor).
		const dayOffsets = [29, 30, 31, 32];
		const items: StreamItem[] = [];
		dayOffsets.forEach((offset, d) => {
			for (let i = 0; i < 10; i += 1) {
				items.push(makeItem(`d${d}-${i}`, BASE + offset * DAY + i * 100));
			}
		});
		const markers = dateMarkers(items, 1600);

		expect(markers.map((m) => m.itemId)).toEqual(['d0-0', 'd1-0', 'd2-0', 'd3-0']);
		expect(markers.map((m) => m.major)).toEqual([true, false, true, false]);
	});

	it('is true only when the UTC month changes, at week granularity', () => {
		// 60 consecutive daily items starting a few days before Jan's end,
		// thinned to one marker per ISO week (per the existing week-thinning
		// rule), spanning two month boundaries.
		const startOffset = 25; // 26 Jan 2024
		const items: StreamItem[] = Array.from({ length: 60 }, (_, i) =>
			makeItem(`i${i}`, BASE + (startOffset + i) * DAY)
		);
		const markers = dateMarkers(items, 300);

		// Independently derived from each kept marker's own UTC calendar
		// month compared against the previous kept marker's month — this is
		// the rule under test, restated without calling it.
		let lastMonth: number | null = null;
		const expectedMajors = markers.map((m) => {
			const month = new Date(m.timestampUnix * 1000).getUTCMonth();
			const major = lastMonth === null || month !== lastMonth;
			lastMonth = month;
			return major;
		});
		expect(markers.map((m) => m.major)).toEqual(expectedMajors);
		expect(markers[0].major).toBe(true);
		// Sanity: this fixture must actually cross a month boundary, or the
		// assertion above would pass vacuously with every flag false but
		// the first.
		expect(expectedMajors.filter(Boolean).length).toBeGreaterThan(1);
	});

	it('is true only when the UTC year changes, at month granularity', () => {
		// Reuses the month-thinning fixture (3-year daily history, 36
		// monthly markers spanning 2024/2025/2026).
		const items: StreamItem[] = Array.from({ length: 1095 }, (_, i) =>
			makeItem(`i${i}`, BASE + i * DAY)
		);
		const markers = dateMarkers(items, 2000);

		let lastYear: number | null = null;
		const expectedMajors = markers.map((m) => {
			const year = new Date(m.timestampUnix * 1000).getUTCFullYear();
			const major = lastYear === null || year !== lastYear;
			lastYear = year;
			return major;
		});
		expect(markers.map((m) => m.major)).toEqual(expectedMajors);
		expect(expectedMajors.filter(Boolean).length).toBe(3);
	});

	it('carries the major flag through the degenerate spacing-floor thinning untouched', () => {
		// The degenerate over-dense fixture (one marker per month for 200
		// months, thinned to at most 3 kept markers) — every surviving
		// marker still carries a boolean major flag, and the first is
		// always major.
		const items: StreamItem[] = Array.from({ length: 200 }, (_, i) => {
			const year = 2000 + Math.floor(i / 12);
			const month = i % 12;
			return makeItem(`i${i}`, Date.UTC(year, month, 1) / 1000);
		});
		const markers = dateMarkers(items, 50);

		expect(markers[0].major).toBe(true);
		for (const marker of markers) {
			expect(typeof marker.major).toBe('boolean');
		}
	});
});
