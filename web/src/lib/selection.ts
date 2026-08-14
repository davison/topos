// The bulk-selection axis for the stream (13-UI-SPEC.md E1, KERN-09/
// KERN-10's user-facing half) — pure, Svelte-free helpers, following
// web/src/lib/format.ts's own pure-helper + colocated-vitest convention
// (selection.test.ts, alongside this file). Deliberately orthogonal to
// StreamRow.svelte's pre-existing `selected`/`onselect` axis ("which item
// is open in the detail pane") — this module knows nothing about the
// detail pane, only about which item ids are currently bulk-selected, so
// the two axes can render additively without either one reading the
// other's state.

/**
 * Returns a new Set with `id` added when it is absent from `current` and
 * removed when it is present — `current` itself is never mutated.
 */
export function toggleSelection(current: Set<string>, id: string): Set<string> {
	const next = new Set(current);
	if (next.has(id)) {
		next.delete(id);
	} else {
		next.add(id);
	}
	return next;
}

/**
 * Returns every id from `anchorId` to `targetId` in `orderedIds`,
 * inclusive, in either direction — REPLACING (never unioning) whatever
 * selection came before, per 13-UI-SPEC.md E1's "matches the simplest
 * native file-manager convention, not an additive union." There is
 * deliberately no `current` selection parameter: the absence of one is
 * what makes "replace" true by construction rather than by a caller
 * remembering to discard the old set.
 *
 * Falls back to a single-id selection of `targetId` when `anchorId` is
 * `null` or is not present in `orderedIds` (e.g. the anchor has since
 * scrolled out of — or was removed from — the currently-rendered list).
 * `anchorId === targetId` returns exactly that one id. `orderedIds` is
 * never mutated.
 */
export function selectRange(
	orderedIds: string[],
	anchorId: string | null,
	targetId: string
): Set<string> {
	if (anchorId === null) return new Set([targetId]);

	const anchorIndex = orderedIds.indexOf(anchorId);
	if (anchorIndex === -1) return new Set([targetId]);

	const targetIndex = orderedIds.indexOf(targetId);
	if (targetIndex === -1) return new Set([targetId]);

	const [start, end] =
		anchorIndex <= targetIndex ? [anchorIndex, targetIndex] : [targetIndex, anchorIndex];
	return new Set(orderedIds.slice(start, end + 1));
}

/** Returns a new, empty selection Set. */
export function clearSelection(): Set<string> {
	return new Set();
}
