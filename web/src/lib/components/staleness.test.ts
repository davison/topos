import { describe, it, expect } from 'vitest';
import { detailPaneState, staleSources, filterItemsBySource } from '$lib/format';
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
		source: 'silverbullet',
		source_type: 'silverbullet',
		source_display_name: 'SilverBullet',
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

describe('staleSources', () => {
	it('derives exactly the unreachable instance ids from a two-source response', () => {
		const sources = [
			makeSource({ name: 'paperless', reachable: true }),
			makeSource({ name: 'silverbullet', reachable: false })
		];
		expect(staleSources(sources)).toEqual(new Set(['silverbullet']));
	});

	it('returns an empty set when every source is reachable', () => {
		const sources = [makeSource({ name: 'paperless', reachable: true })];
		expect(staleSources(sources).size).toBe(0);
	});

	// D-08: staleness is computed per instance. Two sources sharing one
	// source_type but differing in name (e.g. two Proton accounts), one
	// reachable and one not, must mark ONLY the unreachable instance's
	// rows stale — a healthy sibling instance of the same plugin type
	// must never be swept in just because it shares a plugin kind.
	it('marks only the unreachable instance stale when two instances share one source_type', () => {
		const sources = [
			makeSource({ name: 'home-email', source_type: 'proton', reachable: true }),
			makeSource({ name: 'work-email', source_type: 'proton', reachable: false })
		];
		const stale = staleSources(sources);
		expect(stale).toEqual(new Set(['work-email']));
		expect(stale.has('home-email')).toBe(false);

		const items = [
			makeItem({ id: 'home1', source: 'home-email', source_type: 'proton' }),
			makeItem({ id: 'work1', source: 'work-email', source_type: 'proton' })
		];
		expect(items.map((item) => ({ id: item.id, stale: stale.has(item.source) }))).toEqual([
			{ id: 'home1', stale: false },
			{ id: 'work1', stale: true }
		]);
	});
});

describe('stale markers never reorder the stream', () => {
	it('preserves the response order when mapping rows to their stale flag', () => {
		const sources = [
			makeSource({ name: 'paperless', reachable: true }),
			makeSource({ name: 'silverbullet', reachable: false })
		];
		const stale = staleSources(sources);
		const items = [
			makeItem({ id: 'a', source: 'paperless' }),
			makeItem({ id: 'b', source: 'silverbullet' }),
			makeItem({ id: 'c', source: 'paperless' }),
			makeItem({ id: 'd', source: 'silverbullet' })
		];

		// Order must be untouched by the stale-flag mapping...
		const rows = items.map((item) => ({ id: item.id, stale: stale.has(item.source) }));
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
		const sources = [makeSource({ name: 'silverbullet', reachable: false })];
		const stale = staleSources(sources);
		const items = [
			makeItem({ id: 'b', source: 'silverbullet' }),
			makeItem({ id: 'd', source: 'silverbullet' })
		];
		const filtered = filterItemsBySource(items, new Set(['silverbullet']));
		expect(filtered.map((i) => i.id)).toEqual(['b', 'd']);
		expect(filtered.every((i) => stale.has(i.source))).toBe(true);
	});
});
