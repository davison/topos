import { describe, it, expect } from 'vitest';
import {
	healthTone,
	formatRelativeTime,
	syncingSources,
	shouldShowSourceRows,
	resolveSourceFilters,
	toggleSourceFilter,
	serializeSourceFilters,
	filterItemsBySource,
	streamVariant,
	worstHealthTone,
	visibleChipCount
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

describe('resolveSourceFilters', () => {
	const sources = [
		makeSource({ name: 'paperless' }),
		makeSource({ name: 'silverbullet', source_type: 'silverbullet', display_name: 'SilverBullet' })
	];

	it('resolves null (no query param) to the empty set (no filter)', () => {
		expect(resolveSourceFilters(null, sources)).toEqual(new Set());
	});

	it('resolves an empty string to the empty set', () => {
		expect(resolveSourceFilters('', sources)).toEqual(new Set());
	});

	it('resolves a whitespace-only value to the empty set', () => {
		expect(resolveSourceFilters('   ', sources)).toEqual(new Set());
	});

	it('resolves a single configured instance id to a one-member set', () => {
		expect(resolveSourceFilters('silverbullet', sources)).toEqual(new Set(['silverbullet']));
	});

	it('resolves a comma-joined list of configured instance ids to their set', () => {
		expect(resolveSourceFilters('paperless,silverbullet', sources)).toEqual(
			new Set(['paperless', 'silverbullet'])
		);
	});

	it('degrades an unrecognised value to the empty set rather than an error state', () => {
		expect(resolveSourceFilters('not-a-real-source', sources)).toEqual(new Set());
	});

	// 06-RESEARCH.md Pitfall 4 / T-02-17's multi-value form: a partially
	// stale bookmark narrows to the still-valid member(s), never an error
	// and never the empty set just because one member is stale.
	it('degrades a partially-stale value per member, keeping the configured member', () => {
		expect(resolveSourceFilters('paperless,not-a-real-source', sources)).toEqual(
			new Set(['paperless'])
		);
	});

	it('tolerates surrounding whitespace and extra commas around members', () => {
		expect(resolveSourceFilters(' paperless , ,silverbullet ,', sources)).toEqual(
			new Set(['paperless', 'silverbullet'])
		);
	});

	// D-08: two instances of one plugin type must resolve independently —
	// a filter value naming one instance must never also match its sibling.
	it('resolves exactly one of two instances sharing a plugin kind', () => {
		const twoInstances = [
			makeSource({ name: 'home-email', source_type: 'proton' }),
			makeSource({ name: 'work-email', source_type: 'proton' })
		];
		expect(resolveSourceFilters('home-email', twoInstances)).toEqual(new Set(['home-email']));
		expect(resolveSourceFilters('proton', twoInstances)).toEqual(new Set());
	});
});

describe('toggleSourceFilter', () => {
	it('adds a name absent from the set', () => {
		const current = new Set(['paperless']);
		expect(toggleSourceFilter(current, 'silverbullet')).toEqual(
			new Set(['paperless', 'silverbullet'])
		);
	});

	it('removes a name present in the set', () => {
		const current = new Set(['paperless', 'silverbullet']);
		expect(toggleSourceFilter(current, 'paperless')).toEqual(new Set(['silverbullet']));
	});

	it('toggling twice returns a set equal to the original', () => {
		const original = new Set(['paperless']);
		const twice = toggleSourceFilter(toggleSourceFilter(original, 'silverbullet'), 'silverbullet');
		expect(twice).toEqual(original);
	});

	it('never mutates the input set', () => {
		const original = new Set(['paperless']);
		toggleSourceFilter(original, 'silverbullet');
		expect(original).toEqual(new Set(['paperless']));
	});
});

describe('serializeSourceFilters', () => {
	it('joins members with commas', () => {
		expect(serializeSourceFilters(new Set(['paperless', 'silverbullet']))).toBe(
			'paperless,silverbullet'
		);
	});

	it('serializes an empty set to the empty string', () => {
		expect(serializeSourceFilters(new Set())).toBe('');
	});

	it('round-trips through resolveSourceFilters to an equal set', () => {
		const sources = [makeSource({ name: 'paperless' }), makeSource({ name: 'silverbullet' })];
		const original = new Set(['paperless', 'silverbullet']);
		const resolved = resolveSourceFilters(serializeSourceFilters(original), sources);
		expect(resolved).toEqual(original);
	});
});

describe('filterItemsBySource', () => {
	const items = [
		makeItem({ id: 'a', source: 'paperless' }),
		makeItem({ id: 'b', source: 'silverbullet' }),
		makeItem({ id: 'c', source: 'paperless' })
	];

	it('returns every item, unchanged in order, for the empty set (All)', () => {
		expect(filterItemsBySource(items, new Set()).map((i) => i.id)).toEqual(['a', 'b', 'c']);
	});

	it('narrows to exactly the matching instance id, preserving order', () => {
		expect(filterItemsBySource(items, new Set(['paperless'])).map((i) => i.id)).toEqual([
			'a',
			'c'
		]);
	});

	it('narrows to exactly the matching members of a two-member set, preserving order', () => {
		expect(
			filterItemsBySource(items, new Set(['paperless', 'silverbullet'])).map((i) => i.id)
		).toEqual(['a', 'b', 'c']);
	});

	// D-08: two items sharing one source_type but differing in source
	// (instance id) must never both match one instance's filter value.
	it('narrows correctly when two items share a source_type but differ in source', () => {
		const twoInstanceItems = [
			makeItem({ id: 'home1', source: 'home-email', source_type: 'proton' }),
			makeItem({ id: 'work1', source: 'work-email', source_type: 'proton' })
		];
		expect(
			filterItemsBySource(twoInstanceItems, new Set(['home-email'])).map((i) => i.id)
		).toEqual(['home1']);
	});
});

describe('streamVariant', () => {
	it('chooses sync-failed when the aggregate status is error and items are empty, unfiltered', () => {
		const response = makeResponse({ sync: { status: 'error', finished_unix: 1, error: 'boom' }, items: [] });
		expect(streamVariant(response, new Set())).toBe('sync-failed');
	});

	it('chooses sync-failed even while a filter is active', () => {
		const response = makeResponse({ sync: { status: 'error', finished_unix: 1, error: 'boom' }, items: [] });
		expect(streamVariant(response, new Set(['silverbullet']))).toBe('sync-failed');
	});

	it('chooses sync-failed over empty even when the response items array happens to be empty for other reasons', () => {
		const response = makeResponse({ sync: { status: 'error', finished_unix: 1, error: 'boom' }, items: [] });
		expect(streamVariant(response, new Set())).not.toBe('empty');
	});

	it('chooses empty (unfiltered) when sync ok and zero items', () => {
		const response = makeResponse({ sync: { status: 'ok', finished_unix: 1, error: '' }, items: [] });
		expect(streamVariant(response, new Set())).toBe('empty');
	});

	it('chooses empty-filtered when sync ok, items exist, but none match the filter', () => {
		const response = makeResponse({
			sync: { status: 'ok', finished_unix: 1, error: '' },
			items: [makeItem({ source: 'paperless' })]
		});
		expect(streamVariant(response, new Set(['silverbullet']))).toBe('empty-filtered');
	});

	it('chooses populated when at least one item matches an empty (unfiltered) selection', () => {
		const response = makeResponse({
			sync: { status: 'ok', finished_unix: 1, error: '' },
			items: [makeItem({ source: 'paperless' })]
		});
		expect(streamVariant(response, new Set())).toBe('populated');
	});

	it('chooses populated when at least one item matches a populated selection', () => {
		const response = makeResponse({
			sync: { status: 'ok', finished_unix: 1, error: '' },
			items: [makeItem({ source: 'paperless' })]
		});
		expect(streamVariant(response, new Set(['paperless']))).toBe('populated');
	});
});

describe('worstHealthTone', () => {
	it('maps a single success source to success', () => {
		expect(worstHealthTone([makeSource({ reachable: true, last_status: 'ok' })])).toBe('success');
	});

	it('maps a single warning source to warning', () => {
		expect(worstHealthTone([makeSource({ reachable: true, last_status: 'error' })])).toBe(
			'warning'
		);
	});

	it('maps a single destructive (unreachable) source to destructive', () => {
		expect(worstHealthTone([makeSource({ reachable: false, last_status: 'error' })])).toBe(
			'destructive'
		);
	});

	it('maps a single never-synced source to unknown', () => {
		expect(worstHealthTone([makeSource({ last_status: '', last_sync_unix: 0 })])).toBe('unknown');
	});

	it('resolves the empty set to the neutral unknown tone', () => {
		expect(worstHealthTone([])).toBe('unknown');
	});

	it('a destructive source among mixed tones always wins, regardless of position', () => {
		const mixed = [
			makeSource({ name: 'a', reachable: true, last_status: 'ok' }),
			makeSource({ name: 'b', reachable: false, last_status: 'error' }),
			makeSource({ name: 'c', reachable: true, last_status: 'error' })
		];
		expect(worstHealthTone(mixed)).toBe('destructive');
	});

	it('warning wins over unknown and success when no destructive source is present', () => {
		const mixed = [
			makeSource({ name: 'a', reachable: true, last_status: 'ok' }),
			makeSource({ name: 'b', last_status: '', last_sync_unix: 0 }),
			makeSource({ name: 'c', reachable: true, last_status: 'error' })
		];
		expect(worstHealthTone(mixed)).toBe('warning');
	});

	it('unknown wins over success when neither destructive nor warning is present', () => {
		const mixed = [
			makeSource({ name: 'a', reachable: true, last_status: 'ok' }),
			makeSource({ name: 'b', last_status: '', last_sync_unix: 0 })
		];
		expect(worstHealthTone(mixed)).toBe('unknown');
	});
});

describe('visibleChipCount', () => {
	it('returns the full count with no overflow when every chip fits exactly', () => {
		expect(visibleChipCount([10, 10, 10], 30, 0, 8)).toBe(3);
	});

	it('returns the full count with no overflow when every chip fits with room to spare', () => {
		expect(visibleChipCount([10, 10, 10], 38, 0, 8)).toBe(3);
	});

	// One chip more than the exact-fit case above (same availableWidth,
	// reservedWidth and overflowTriggerWidth) hides exactly one chip once
	// the trigger's own width is charged against the budget.
	it('one chip more than fits produces a hidden count of one', () => {
		expect(visibleChipCount([10, 10, 10, 10], 38, 0, 8)).toBe(3);
	});

	it('a reserved trailing width reduces the inline count', () => {
		const withoutReserve = visibleChipCount([10, 10, 10], 30, 0, 8);
		const withReserve = visibleChipCount([10, 10, 10], 30, 10, 8);
		expect(withReserve).toBeLessThan(withoutReserve);
	});

	it('a zero available width produces a zero inline count, never a negative one', () => {
		expect(visibleChipCount([10, 10, 10], 0, 0, 8)).toBe(0);
	});

	it('reserved and trigger widths alone exceeding the available width still floor at zero', () => {
		expect(visibleChipCount([10, 10, 10], 5, 10, 8)).toBe(0);
	});

	it('an empty chip-widths array returns zero regardless of available width', () => {
		expect(visibleChipCount([], 500, 0, 8)).toBe(0);
	});

	it('is deterministic — repeating the same call over unchanged inputs yields the same count', () => {
		const widths = [10, 10, 10, 10, 10];
		const first = visibleChipCount(widths, 38, 4, 8);
		const second = visibleChipCount(widths, 38, 4, 8);
		expect(second).toBe(first);
	});
});
