import { describe, it, expect } from 'vitest';
import { detailPaneState, staleSourceTypes, filterItemsBySource } from '$lib/format';
import type { ItemContent, SourceStatus, StreamItem } from '$lib/api';

function makeContent(overrides: Partial<ItemContent> = {}): ItemContent {
	return { available: true, unavailable_reason: '', text: 'body', rendition: null, ...overrides };
}

function makeSource(overrides: Partial<SourceStatus> = {}): SourceStatus {
	return {
		name: 'silverbullet',
		source_type: 'silverbullet',
		display_name: 'SilverBullet',
		reachable: true,
		syncing: false,
		last_status: 'ok',
		last_sync_unix: 1700000000,
		last_error: '',
		...overrides
	};
}

function makeItem(overrides: Partial<StreamItem> = {}): StreamItem {
	return {
		id: 'silverbullet:1',
		source_type: 'silverbullet',
		source_id: '1',
		title: 'Item',
		preview: '',
		timestamp_unix: 1700000000,
		secondary_timestamp_unix: 0,
		labels: [],
		group_id: '',
		group_label: '',
		link: { url: 'https://example.test/1', fidelity: 'exact' },
		provenance: {},
		...overrides
	};
}

describe('detailPaneState', () => {
	it('returns loaded when there are no failure signals', () => {
		expect(detailPaneState(makeContent({ available: true }), null, true)).toBe('loaded');
	});

	it('returns deleted when the kernel reports the content is unavailable', () => {
		expect(detailPaneState(makeContent({ available: false }), null, true)).toBe('deleted');
	});

	it('returns unreachable when the fetch failed and the source is not reachable', () => {
		expect(detailPaneState(null, 'source_unavailable', false)).toBe('unreachable');
	});

	it('returns the generic error state when the fetch failed and the source is reachable', () => {
		expect(detailPaneState(null, 'source_unavailable', true)).toBe('error');
	});

	it('prefers deleted over unreachable when both signals are present', () => {
		expect(detailPaneState(makeContent({ available: false }), 'source_unavailable', false)).toBe('deleted');
	});

	it('never returns two states or a falsy state for any combination', () => {
		const outcomes = new Set([
			detailPaneState(makeContent({ available: true }), null, true),
			detailPaneState(makeContent({ available: false }), null, true),
			detailPaneState(null, 'source_unavailable', false),
			detailPaneState(null, 'source_unavailable', true),
			detailPaneState(makeContent({ available: false }), 'source_unavailable', false)
		]);
		for (const outcome of outcomes) {
			expect(['loaded', 'deleted', 'unreachable', 'error']).toContain(outcome);
		}
	});
});

describe('staleSourceTypes', () => {
	it('derives exactly the unreachable source_types from a two-source response', () => {
		const sources = [
			makeSource({ source_type: 'paperless', reachable: true }),
			makeSource({ source_type: 'silverbullet', reachable: false })
		];
		expect(staleSourceTypes(sources)).toEqual(new Set(['silverbullet']));
	});

	it('returns an empty set when every source is reachable', () => {
		const sources = [makeSource({ source_type: 'paperless', reachable: true })];
		expect(staleSourceTypes(sources).size).toBe(0);
	});
});

describe('stale markers never reorder the stream', () => {
	it('preserves the response order when mapping rows to their stale flag', () => {
		const sources = [
			makeSource({ source_type: 'paperless', reachable: true }),
			makeSource({ source_type: 'silverbullet', reachable: false })
		];
		const stale = staleSourceTypes(sources);
		const items = [
			makeItem({ id: 'a', source_type: 'paperless' }),
			makeItem({ id: 'b', source_type: 'silverbullet' }),
			makeItem({ id: 'c', source_type: 'paperless' }),
			makeItem({ id: 'd', source_type: 'silverbullet' })
		];

		// Order must be untouched by the stale-flag mapping...
		const rows = items.map((item) => ({ id: item.id, stale: stale.has(item.source_type) }));
		expect(rows.map((r) => r.id)).toEqual(['a', 'b', 'c', 'd']);

		// ...and only the unreachable source's rows carry the marker.
		expect(rows).toEqual([
			{ id: 'a', stale: false },
			{ id: 'b', stale: true },
			{ id: 'c', stale: false },
			{ id: 'd', stale: true }
		]);
	});

	it('filtering to the stale source alone still preserves order and the stale flag', () => {
		const sources = [makeSource({ source_type: 'silverbullet', reachable: false })];
		const stale = staleSourceTypes(sources);
		const items = [
			makeItem({ id: 'b', source_type: 'silverbullet' }),
			makeItem({ id: 'd', source_type: 'silverbullet' })
		];
		const filtered = filterItemsBySource(items, 'silverbullet');
		expect(filtered.map((i) => i.id)).toEqual(['b', 'd']);
		expect(filtered.every((i) => stale.has(i.source_type))).toBe(true);
	});
});
