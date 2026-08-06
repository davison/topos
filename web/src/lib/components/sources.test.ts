import { describe, it, expect } from 'vitest';
import {
	healthTone,
	formatRelativeTime,
	syncingSources,
	shouldShowSourceRows,
	resolveSourceFilter,
	filterItemsBySource,
	streamVariant
} from '$lib/format';
import type { SourceStatus, StreamItem, StreamResponse } from '$lib/api';

function makeSource(overrides: Partial<SourceStatus> = {}): SourceStatus {
	return {
		name: 'paperless',
		source_type: 'paperless',
		display_name: 'paperless-ngx',
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
		id: 'paperless:1',
		source: 'paperless',
		source_type: 'paperless',
		source_display_name: 'paperless-ngx',
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

function makeResponse(overrides: Partial<StreamResponse> = {}): StreamResponse {
	return {
		schema_version: 1,
		webspace: 'house-move',
		sync: { status: 'ok', finished_unix: 1700000000, error: '' },
		items: [],
		...overrides
	};
}

describe('healthTone (full matrix)', () => {
	it('reachable + ok -> success', () => {
		expect(healthTone(makeSource({ reachable: true, last_status: 'ok' }))).toBe('success');
	});

	it('reachable + error -> warning', () => {
		expect(healthTone(makeSource({ reachable: true, last_status: 'error' }))).toBe('warning');
	});

	it('unreachable -> destructive', () => {
		expect(healthTone(makeSource({ reachable: false, last_status: 'error' }))).toBe('destructive');
	});

	it('never synced ("") -> unknown, never success', () => {
		const tone = healthTone(makeSource({ last_status: '', last_sync_unix: 0 }));
		expect(tone).toBe('unknown');
		expect(tone).not.toBe('success');
	});
});

describe('formatRelativeTime (zero case)', () => {
	it('returns the empty string for a zero timestamp', () => {
		expect(formatRelativeTime(0)).toBe('');
	});
});

describe('syncingSources', () => {
	it('returns exactly the instance ids (name) currently mid-sync', () => {
		const sources = [
			makeSource({ name: 'paperless', syncing: true }),
			makeSource({ name: 'silverbullet', syncing: false })
		];
		expect(syncingSources(sources)).toEqual(['paperless']);
	});

	it('returns an empty array when nothing is syncing', () => {
		expect(syncingSources([makeSource({ syncing: false })])).toEqual([]);
	});

	// D-08: two instances of one plugin type must report independently —
	// syncing one instance must never mark its sibling instance as syncing
	// too, even though both share source_type.
	it('keys strictly on instance id, never on the shared source_type of two instances', () => {
		const sources = [
			makeSource({ name: 'home-email', source_type: 'proton', syncing: true }),
			makeSource({ name: 'work-email', source_type: 'proton', syncing: false })
		];
		expect(syncingSources(sources)).toEqual(['home-email']);
	});
});

describe('shouldShowSourceRows', () => {
	it('is false while sources are loading', () => {
		expect(shouldShowSourceRows('loading', [makeSource()])).toBe(false);
	});

	it('is false when the sources request failed', () => {
		expect(shouldShowSourceRows('error', [makeSource()])).toBe(false);
	});

	it('is false when ready but zero sources are configured', () => {
		expect(shouldShowSourceRows('ready', [])).toBe(false);
	});

	it('is true when ready with at least one source', () => {
		expect(shouldShowSourceRows('ready', [makeSource()])).toBe(true);
	});
});

describe('resolveSourceFilter', () => {
	const sources = [
		makeSource({ name: 'paperless' }),
		makeSource({ name: 'silverbullet', source_type: 'silverbullet', display_name: 'SilverBullet' })
	];

	it('resolves null (no query param) to no filter', () => {
		expect(resolveSourceFilter(null, sources)).toBeNull();
	});

	it('resolves a configured instance id to itself', () => {
		expect(resolveSourceFilter('silverbullet', sources)).toBe('silverbullet');
	});

	it('degrades an unrecognised value to no filter rather than an empty/error state', () => {
		expect(resolveSourceFilter('not-a-real-source', sources)).toBeNull();
	});

	// D-08: two instances of one plugin type must resolve independently —
	// a filter value naming one instance must never also match its sibling.
	it('resolves exactly one of two instances sharing a plugin kind', () => {
		const twoInstances = [
			makeSource({ name: 'home-email', source_type: 'proton' }),
			makeSource({ name: 'work-email', source_type: 'proton' })
		];
		expect(resolveSourceFilter('home-email', twoInstances)).toBe('home-email');
		expect(resolveSourceFilter('proton', twoInstances)).toBeNull();
	});
});

describe('filterItemsBySource', () => {
	const items = [
		makeItem({ id: 'a', source: 'paperless' }),
		makeItem({ id: 'b', source: 'silverbullet' }),
		makeItem({ id: 'c', source: 'paperless' })
	];

	it('returns every item, unchanged in order, for null (All)', () => {
		expect(filterItemsBySource(items, null).map((i) => i.id)).toEqual(['a', 'b', 'c']);
	});

	it('narrows to exactly the matching instance id, preserving order', () => {
		expect(filterItemsBySource(items, 'paperless').map((i) => i.id)).toEqual(['a', 'c']);
	});

	// D-08: two items sharing one source_type but differing in source
	// (instance id) must never both match one instance's filter value.
	it('narrows correctly when two items share a source_type but differ in source', () => {
		const twoInstanceItems = [
			makeItem({ id: 'home1', source: 'home-email', source_type: 'proton' }),
			makeItem({ id: 'work1', source: 'work-email', source_type: 'proton' })
		];
		expect(filterItemsBySource(twoInstanceItems, 'home-email').map((i) => i.id)).toEqual([
			'home1'
		]);
	});
});

describe('streamVariant', () => {
	it('chooses sync-failed when the aggregate status is error and items are empty, unfiltered', () => {
		const response = makeResponse({ sync: { status: 'error', finished_unix: 1, error: 'boom' }, items: [] });
		expect(streamVariant(response, null)).toBe('sync-failed');
	});

	it('chooses sync-failed even while a filter is active', () => {
		const response = makeResponse({ sync: { status: 'error', finished_unix: 1, error: 'boom' }, items: [] });
		expect(streamVariant(response, 'silverbullet')).toBe('sync-failed');
	});

	it('chooses sync-failed over empty even when the response items array happens to be empty for other reasons', () => {
		const response = makeResponse({ sync: { status: 'error', finished_unix: 1, error: 'boom' }, items: [] });
		expect(streamVariant(response, null)).not.toBe('empty');
	});

	it('chooses empty (unfiltered) when sync ok and zero items', () => {
		const response = makeResponse({ sync: { status: 'ok', finished_unix: 1, error: '' }, items: [] });
		expect(streamVariant(response, null)).toBe('empty');
	});

	it('chooses empty-filtered when sync ok, items exist, but none match the filter', () => {
		const response = makeResponse({
			sync: { status: 'ok', finished_unix: 1, error: '' },
			items: [makeItem({ source: 'paperless' })]
		});
		expect(streamVariant(response, 'silverbullet')).toBe('empty-filtered');
	});

	it('chooses populated when at least one item matches', () => {
		const response = makeResponse({
			sync: { status: 'ok', finished_unix: 1, error: '' },
			items: [makeItem({ source: 'paperless' })]
		});
		expect(streamVariant(response, null)).toBe('populated');
	});
});
