import { describe, it, expect } from 'vitest';
import { toggleSelection, selectRange, clearSelection } from './selection';

describe('toggleSelection', () => {
	it('adds an absent id', () => {
		const current = new Set(['a']);
		const next = toggleSelection(current, 'b');
		expect(next).toEqual(new Set(['a', 'b']));
	});

	it('removes a present id', () => {
		const current = new Set(['a', 'b']);
		const next = toggleSelection(current, 'a');
		expect(next).toEqual(new Set(['b']));
	});

	it('never mutates the input Set', () => {
		const current = new Set(['a']);
		toggleSelection(current, 'b');
		expect(current).toEqual(new Set(['a']));
	});
});

describe('selectRange', () => {
	const orderedIds = ['a', 'b', 'c', 'd', 'e'];

	it('returns every id from the anchor to the target, inclusive, ascending', () => {
		expect(selectRange(orderedIds, 'b', 'd')).toEqual(new Set(['b', 'c', 'd']));
	});

	it('returns every id from the anchor to the target, inclusive, descending (either direction)', () => {
		expect(selectRange(orderedIds, 'd', 'b')).toEqual(new Set(['b', 'c', 'd']));
	});

	it('REPLACES rather than unions — the function takes no prior-selection argument, so a caller cannot accidentally union', () => {
		// A caller re-deriving the selection from scratch on every call (as
		// +page.svelte's handleBulkToggle does) can never end up with a
		// selection wider than the freshly computed range, because there is
		// no input path for a previous selection to survive through.
		const first = selectRange(orderedIds, 'a', 'b');
		const second = selectRange(orderedIds, 'd', 'e');
		expect(second).toEqual(new Set(['d', 'e']));
		expect(second).not.toEqual(new Set([...first, ...second]));
	});

	it('falls back to a single-id selection of the target when the anchor is not in the current list', () => {
		expect(selectRange(orderedIds, 'not-present', 'c')).toEqual(new Set(['c']));
	});

	it('falls back to a single-id selection of the target when the anchor is null', () => {
		expect(selectRange(orderedIds, null, 'c')).toEqual(new Set(['c']));
	});

	it('returns exactly one id when anchor equals target', () => {
		expect(selectRange(orderedIds, 'c', 'c')).toEqual(new Set(['c']));
	});

	it('never mutates the input array', () => {
		const copy = [...orderedIds];
		selectRange(orderedIds, 'a', 'c');
		expect(orderedIds).toEqual(copy);
	});
});

describe('clearSelection', () => {
	it('returns an empty Set', () => {
		expect(clearSelection()).toEqual(new Set());
	});

	it('returns a fresh Set on every call, not a shared reference', () => {
		expect(clearSelection()).not.toBe(clearSelection());
	});
});
