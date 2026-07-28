import { describe, it, expect } from 'vitest';
import {
	healthTone,
	formatRelativeTime,
	syncingSourceTypes,
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
		source_type: 'paperless',
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

describe('syncingSourceTypes', () => {
	it('returns exactly the source_types currently mid-sync', () => {
		const sources = [
			makeSource({ source_type: 'paperless', syncing: true }),
			makeSource({ source_type: 'silverbullet', syncing: false })
		];
		expect(syncingSourceTypes(sources)).toEqual(['paperless']);
	});

	it('returns an empty array when nothing is syncing', () => {
		expect(syncingSourceTypes([makeSource({ syncing: false })])).toEqual([]);
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
		makeSource({ source_type: 'paperless' }),
		makeSource({ source_type: 'silverbullet', name: 'silverbullet', display_name: 'SilverBullet' })
	];

	it('resolves null (no query param) to no filter', () => {
		expect(resolveSourceFilter(null, sources)).toBeNull();
	});

	it('resolves a configured source_type to itself', () => {
		expect(resolveSourceFilter('silverbullet', sources)).toBe('silverbullet');
	});

	it('degrades an unrecognised value to no filter rather than an empty/error state', () => {
		expect(resolveSourceFilter('not-a-real-source', sources)).toBeNull();
	});
});

describe('filterItemsBySource', () => {
	const items = [
		makeItem({ id: 'a', source_type: 'paperless' }),
		makeItem({ id: 'b', source_type: 'silverbullet' }),
		makeItem({ id: 'c', source_type: 'paperless' })
	];

	it('returns every item, unchanged in order, for null (All)', () => {
		expect(filterItemsBySource(items, null).map((i) => i.id)).toEqual(['a', 'b', 'c']);
	});

	it('narrows to exactly the matching source_type, preserving order', () => {
		expect(filterItemsBySource(items, 'paperless').map((i) => i.id)).toEqual(['a', 'c']);
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
			items: [makeItem({ source_type: 'paperless' })]
		});
		expect(streamVariant(response, 'silverbullet')).toBe('empty-filtered');
	});

	it('chooses populated when at least one item matches', () => {
		const response = makeResponse({
			sync: { status: 'ok', finished_unix: 1, error: '' },
			items: [makeItem({ source_type: 'paperless' })]
		});
		expect(streamVariant(response, null)).toBe('populated');
	});
});
