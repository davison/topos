package index

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/davison/topos/kernel/item"
)

// TestSetItemMarks_IdempotentInsert proves SetItemMarks on a fresh
// webspace inserts one row and reports changed=1, and that calling it
// again with the SAME (webspace, item, kind) reports changed=0 and leaves
// exactly one row — never a duplicate row and never a constraint error
// (KERN-09 adjacency: excluding an already-excluded item is idempotent).
func TestSetItemMarks_IdempotentInsert(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	it := sampleItem("1", 100)
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{it}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	changed, err := s.SetItemMarks(ctx, "ws", MarkKindExcluded, []string{it.ID})
	if err != nil {
		t.Fatalf("SetItemMarks (first): %v", err)
	}
	if changed != 1 {
		t.Errorf("expected changed=1 on first mark, got %d", changed)
	}

	count, err := s.CountItemMarks(ctx, "ws", MarkKindExcluded)
	if err != nil {
		t.Fatalf("CountItemMarks: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 item_marks row after first mark, got %d", count)
	}

	changed, err = s.SetItemMarks(ctx, "ws", MarkKindExcluded, []string{it.ID})
	if err != nil {
		t.Fatalf("SetItemMarks (second): %v", err)
	}
	if changed != 0 {
		t.Errorf("expected changed=0 on re-marking the same item, got %d", changed)
	}

	count, err = s.CountItemMarks(ctx, "ws", MarkKindExcluded)
	if err != nil {
		t.Fatalf("CountItemMarks after re-mark: %v", err)
	}
	if count != 1 {
		t.Errorf("expected exactly 1 item_marks row after re-marking the same item, got %d", count)
	}
}

// TestClearItemMarks_UnmarkedItemIsNoOp proves clearing a mark that was
// never set returns changed=0, never an error — un-excluding an item that
// carries no mark is not a failure case (KERN-10 adjacency).
func TestClearItemMarks_UnmarkedItemIsNoOp(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	changed, err := s.ClearItemMarks(ctx, "ws", MarkKindExcluded, []string{"paperless:does-not-exist"})
	if err != nil {
		t.Fatalf("ClearItemMarks: %v", err)
	}
	if changed != 0 {
		t.Errorf("expected changed=0 clearing a mark that was never set, got %d", changed)
	}
}

// TestStreamItems_OmitsExcludedItemForItsOwnWebspaceOnly proves the
// shared mark filter (markFilterClause): an item marked "excluded" for
// webspace A disappears from A's stream but keeps appearing in webspace
// B's stream, and every surviving item keeps the pre-existing
// timestamp_unix DESC, secondary_timestamp_unix DESC, id ASC ordering
// (D-03/D-04 adjacency, "mark filtering never reorders the stream").
func TestStreamItems_OmitsExcludedItemForItsOwnWebspaceOnly(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	older := sampleItem("1", 100)
	newer := sampleItem("2", 200)
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws-a", "paperless", []item.Item{older, newer}); err != nil {
		t.Fatalf("seed ws-a: %v", err)
	}
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws-b", "paperless", []item.Item{older, newer}); err != nil {
		t.Fatalf("seed ws-b: %v", err)
	}

	if _, err := s.SetItemMarks(ctx, "ws-a", MarkKindExcluded, []string{older.ID}); err != nil {
		t.Fatalf("SetItemMarks: %v", err)
	}

	itemsA, err := s.StreamItems(ctx, "ws-a", nil, nil, 0, 0, ViewIncluded)
	if err != nil {
		t.Fatalf("StreamItems(ws-a): %v", err)
	}
	if len(itemsA) != 1 || itemsA[0].ID != newer.ID {
		t.Fatalf("expected ws-a to carry exactly [%s] after excluding %s, got %v", newer.ID, older.ID, idsOf(itemsA))
	}

	itemsB, err := s.StreamItems(ctx, "ws-b", nil, nil, 0, 0, ViewIncluded)
	if err != nil {
		t.Fatalf("StreamItems(ws-b): %v", err)
	}
	if len(itemsB) != 2 {
		t.Fatalf("expected ws-b to still carry both items (mark scoped to ws-a only), got %v", idsOf(itemsB))
	}
	// Ordering unaffected by the mark filter: newest first, unchanged.
	if itemsB[0].ID != newer.ID || itemsB[1].ID != older.ID {
		t.Errorf("expected ws-b ordering [%s, %s] (newest first), got %v", newer.ID, older.ID, idsOf(itemsB))
	}
}

// TestStreamItems_ExcludedItemOrderingPreservedAmongSurvivors proves
// excluding a middle item never disturbs the chronological order of the
// items that remain.
func TestStreamItems_ExcludedItemOrderingPreservedAmongSurvivors(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	first := sampleItem("1", 300)  // newest
	middle := sampleItem("2", 200) // excluded
	last := sampleItem("3", 100)   // oldest
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{first, middle, last}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if _, err := s.SetItemMarks(ctx, "ws", MarkKindExcluded, []string{middle.ID}); err != nil {
		t.Fatalf("SetItemMarks: %v", err)
	}

	items, err := s.StreamItems(ctx, "ws", nil, nil, 0, 0, ViewIncluded)
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}
	if len(items) != 2 || items[0].ID != first.ID || items[1].ID != last.ID {
		t.Fatalf("expected surviving order [%s, %s], got %v", first.ID, last.ID, idsOf(items))
	}
}

// TestSearch_OmitsExcludedItemForItsOwnWebspaceOnly proves Search
// composes the same markFilterClause StreamItems does: a marked item is
// absent from its own webspace's search results and still present in a
// different webspace's.
func TestSearch_OmitsExcludedItemForItsOwnWebspaceOnly(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	it := sampleItem("1", 100)
	it.Title = "invoice from acme"
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws-a", "paperless", []item.Item{it}); err != nil {
		t.Fatalf("seed ws-a: %v", err)
	}
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws-b", "paperless", []item.Item{it}); err != nil {
		t.Fatalf("seed ws-b: %v", err)
	}

	if _, err := s.SetItemMarks(ctx, "ws-a", MarkKindExcluded, []string{it.ID}); err != nil {
		t.Fatalf("SetItemMarks: %v", err)
	}

	resultsA, err := s.Search(ctx, "ws-a", "invoice", nil, nil, 0, 0)
	if err != nil {
		t.Fatalf("Search(ws-a): %v", err)
	}
	if len(resultsA) != 0 {
		t.Errorf("expected Search(ws-a) to omit the excluded item, got %d result(s)", len(resultsA))
	}

	resultsB, err := s.Search(ctx, "ws-b", "invoice", nil, nil, 0, 0)
	if err != nil {
		t.Fatalf("Search(ws-b): %v", err)
	}
	if len(resultsB) != 1 || resultsB[0].Item.ID != it.ID {
		t.Fatalf("expected Search(ws-b) to still surface the item (mark scoped to ws-a only), got %d result(s)", len(resultsB))
	}
}

// TestDeleteSourceItems_MarkSurvives is the load-bearing no-cascade proof
// (KERN-09, D-10): item_marks carries NO FOREIGN KEY to items(id), so
// DeleteSourceItems' DELETE FROM items must never cascade-delete a mark.
// This fails if item_marks.item_id is ever given a REFERENCES clause.
func TestDeleteSourceItems_MarkSurvives(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	it := sampleItem("1", 100)
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{it}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := s.SetItemMarks(ctx, "ws", MarkKindExcluded, []string{it.ID}); err != nil {
		t.Fatalf("SetItemMarks: %v", err)
	}

	if err := s.DeleteSourceItems(ctx, "paperless"); err != nil {
		t.Fatalf("DeleteSourceItems: %v", err)
	}

	if _, ok, err := s.GetItem(ctx, it.ID); err != nil {
		t.Fatalf("GetItem after delete: %v", err)
	} else if ok {
		t.Fatalf("expected item %q to be gone from items after DeleteSourceItems", it.ID)
	}

	count, err := s.CountItemMarks(ctx, "ws", MarkKindExcluded)
	if err != nil {
		t.Fatalf("CountItemMarks after DeleteSourceItems: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected item_marks row to survive DeleteSourceItems with no FK cascade, got count=%d", count)
	}
}

// TestItemMarks_SurviveIndexRebuild pins the KERN-09 survival guarantee
// by a test rather than a comment (13-01-PLAN.md Task 2): a mark written
// under schemaVersion N is still present after an Open that triggers
// rebuildOnSchemaChange, while items/webspace_items rows for the same
// item are gone — and the surviving mark still filters the item out once
// it re-enters the index via a later ReplaceWebspaceSourceItems call.
// Mirrors TestOpen_IndexAtThePreviousSchemaVersionIsRebuiltAndAcceptsANotice's
// own "force PRAGMA user_version, close, reopen" mechanism. This test
// fails (see its own failure messages) if item_marks is ever added to
// rebuildOnSchemaChange's drop list — verified by temporarily adding it,
// observing the failure, and reverting.
func TestItemMarks_SurviveIndexRebuild(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.db")

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	ctx := context.Background()
	it := sampleItem("1", 100)
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{it}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := s.SetItemMarks(ctx, "ws", MarkKindExcluded, []string{it.ID}); err != nil {
		t.Fatalf("SetItemMarks: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Force the on-disk schema version to a value other than the current
	// schemaVersion, simulating an operator's existing index.db from
	// before a schema change — the same mechanism
	// TestOpen_IndexAtThePreviousSchemaVersionIsRebuiltAndAcceptsANotice
	// (store_test.go) already uses.
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open raw db: %v", err)
	}
	if _, err := raw.Exec(`PRAGMA user_version = 999`); err != nil {
		t.Fatalf("force user_version = 999: %v", err)
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("close raw db: %v", err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen at a stale schema version: %v", err)
	}
	t.Cleanup(func() { reopened.Close() })

	// (a) items/webspace_items rows are gone — the ordinary rebuild
	// behavior, unaffected by this phase's addition.
	items, err := reopened.StreamItems(ctx, "ws", nil, nil, 0, 0, ViewIncluded)
	if err != nil {
		t.Fatalf("StreamItems after rebuild: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected the schema rebuild to drop pre-existing item rows (D-07), got: %v", idsOf(items))
	}

	// (b) the item_marks row survives — the rebuild ate the mark if this
	// count is 0.
	count, err := reopened.CountItemMarks(ctx, "ws", MarkKindExcluded)
	if err != nil {
		t.Fatalf("CountItemMarks after rebuild: %v", err)
	}
	if count != 1 {
		t.Fatalf("the rebuild ate the mark: expected the item_marks row to survive a schema-version-triggered rebuild, got count=%d", count)
	}

	// The surviving mark still filters the item out once it re-enters the
	// index via an ordinary sync — a match rule that later pulls the item
	// back in cannot resurrect it.
	if err := reopened.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{it}); err != nil {
		t.Fatalf("re-insert after rebuild: %v", err)
	}
	itemsAfterReinsert, err := reopened.StreamItems(ctx, "ws", nil, nil, 0, 0, ViewIncluded)
	if err != nil {
		t.Fatalf("StreamItems after re-insert: %v", err)
	}
	if len(itemsAfterReinsert) != 0 {
		t.Fatalf("the rebuild ate the mark: expected the surviving mark to still filter %s after it re-entered the index, got %v", it.ID, idsOf(itemsAfterReinsert))
	}
}

// TestStreamItems_ViewExcludedReturnsExactlyMarkedItems proves the
// ViewExcluded branch (13-02-PLAN.md Task 1): it returns exactly the
// items carrying an excluded mark for the webspace, in the same
// timestamp_unix DESC ordering ViewIncluded uses, and never an unmarked
// survivor.
func TestStreamItems_ViewExcludedReturnsExactlyMarkedItems(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	newest := sampleItem("1", 300)
	excluded := sampleItem("2", 200)
	oldest := sampleItem("3", 100)
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{newest, excluded, oldest}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := s.SetItemMarks(ctx, "ws", MarkKindExcluded, []string{excluded.ID}); err != nil {
		t.Fatalf("SetItemMarks: %v", err)
	}

	got, err := s.StreamItems(ctx, "ws", nil, nil, 0, 0, ViewExcluded)
	if err != nil {
		t.Fatalf("StreamItems(ViewExcluded): %v", err)
	}
	if len(got) != 1 || got[0].ID != excluded.ID {
		t.Fatalf("expected ViewExcluded to carry exactly [%s], got %v", excluded.ID, idsOf(got))
	}
}

// TestStreamItems_IncludedAndExcludedViewsAreComplements proves the two
// views partition a webspace's full item set with no overlap and no gap:
// len(included) + len(excluded) equals the unfiltered item count.
func TestStreamItems_IncludedAndExcludedViewsAreComplements(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	a := sampleItem("1", 400)
	b := sampleItem("2", 300)
	c := sampleItem("3", 200)
	d := sampleItem("4", 100)
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{a, b, c, d}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := s.SetItemMarks(ctx, "ws", MarkKindExcluded, []string{b.ID, d.ID}); err != nil {
		t.Fatalf("SetItemMarks: %v", err)
	}

	included, err := s.StreamItems(ctx, "ws", nil, nil, 0, 0, ViewIncluded)
	if err != nil {
		t.Fatalf("StreamItems(ViewIncluded): %v", err)
	}
	excluded, err := s.StreamItems(ctx, "ws", nil, nil, 0, 0, ViewExcluded)
	if err != nil {
		t.Fatalf("StreamItems(ViewExcluded): %v", err)
	}

	if len(included)+len(excluded) != 4 {
		t.Fatalf("expected len(included)+len(excluded) == 4 (the unfiltered item count), got %d + %d", len(included), len(excluded))
	}
	seen := map[string]bool{}
	for _, it := range append(append([]item.Item{}, included...), excluded...) {
		if seen[it.ID] {
			t.Errorf("item %s appeared in both views — the partition must have no overlap", it.ID)
		}
		seen[it.ID] = true
	}
	for _, id := range []string{a.ID, b.ID, c.ID, d.ID} {
		if !seen[id] {
			t.Errorf("item %s missing from both views — the partition must have no gap", id)
		}
	}
}

// TestStreamItems_ZeroMarksExcludedViewReturnsEmpty proves a webspace with
// zero marks returns the full stream for ViewIncluded and an empty slice
// (not an error) for ViewExcluded.
func TestStreamItems_ZeroMarksExcludedViewReturnsEmpty(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	it := sampleItem("1", 100)
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{it}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	included, err := s.StreamItems(ctx, "ws", nil, nil, 0, 0, ViewIncluded)
	if err != nil {
		t.Fatalf("StreamItems(ViewIncluded): %v", err)
	}
	if len(included) != 1 {
		t.Fatalf("expected ViewIncluded to carry the full stream with zero marks, got %v", idsOf(included))
	}

	excluded, err := s.StreamItems(ctx, "ws", nil, nil, 0, 0, ViewExcluded)
	if err != nil {
		t.Fatalf("StreamItems(ViewExcluded): %v", err)
	}
	if len(excluded) != 0 {
		t.Fatalf("expected ViewExcluded to be empty with zero marks, got %v", idsOf(excluded))
	}
}

// --- Orphan prune sweep (13-02-PLAN.md Task 2, D-09/D-10) ---

// TestReplaceWebspaceSourceItems_ResyncedExcludedItemKeepsItsMark proves a
// sync that re-reports a previously-excluded item leaves its mark intact —
// the sweep only removes marks for items a healthy sync OMITS, never ones
// it still reports.
func TestReplaceWebspaceSourceItems_ResyncedExcludedItemKeepsItsMark(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	it := sampleItem("1", 100)
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{it}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := s.SetItemMarks(ctx, "ws", MarkKindExcluded, []string{it.ID}); err != nil {
		t.Fatalf("SetItemMarks: %v", err)
	}

	// Re-sync reporting the SAME item again.
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{it}); err != nil {
		t.Fatalf("re-sync: %v", err)
	}

	count, err := s.CountItemMarks(ctx, "ws", MarkKindExcluded)
	if err != nil {
		t.Fatalf("CountItemMarks: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected the mark to survive a re-sync that still reports the item, got count=%d", count)
	}
}

// TestReplaceWebspaceSourceItems_OmittedExcludedItemIsPruned proves a
// healthy sync that omits a previously-excluded item of the SAME source
// removes that mark — the core D-09 sweep behavior.
func TestReplaceWebspaceSourceItems_OmittedExcludedItemIsPruned(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	kept := sampleItem("1", 200)
	vanished := sampleItem("2", 100)
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{kept, vanished}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := s.SetItemMarks(ctx, "ws", MarkKindExcluded, []string{vanished.ID}); err != nil {
		t.Fatalf("SetItemMarks: %v", err)
	}

	// Healthy re-sync that no longer reports "vanished".
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{kept}); err != nil {
		t.Fatalf("re-sync omitting vanished item: %v", err)
	}

	count, err := s.CountItemMarks(ctx, "ws", MarkKindExcluded)
	if err != nil {
		t.Fatalf("CountItemMarks: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected the mark on the omitted item to be swept, got count=%d", count)
	}
}

// TestReplaceWebspaceSourceItems_SweepNeverTouchesSiblingSourceMarks proves
// a sync of source A never removes a mark on an item belonging to source B
// in the same webspace, even when B's item is never reported by A's sync.
func TestReplaceWebspaceSourceItems_SweepNeverTouchesSiblingSourceMarks(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	aItem := sampleItemForSource("paperless", "1", 200)
	bItem := sampleItemForSource("silverbullet", "1", 100)
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{aItem}); err != nil {
		t.Fatalf("seed paperless: %v", err)
	}
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "silverbullet", []item.Item{bItem}); err != nil {
		t.Fatalf("seed silverbullet: %v", err)
	}
	if _, err := s.SetItemMarks(ctx, "ws", MarkKindExcluded, []string{bItem.ID}); err != nil {
		t.Fatalf("SetItemMarks: %v", err)
	}

	// A healthy paperless sync that omits everything (paperless never
	// reported bItem in the first place — it belongs to silverbullet).
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", nil); err != nil {
		t.Fatalf("paperless sync: %v", err)
	}

	count, err := s.CountItemMarks(ctx, "ws", MarkKindExcluded)
	if err != nil {
		t.Fatalf("CountItemMarks: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected silverbullet's mark to survive a paperless-only sync, got count=%d", count)
	}
}

// TestReplaceWebspaceSourceItems_SweepNeverTouchesOtherWebspaceMarks proves
// a sync of webspace X never removes a mark in webspace Y, even for the
// SAME item id and the SAME source.
func TestReplaceWebspaceSourceItems_SweepNeverTouchesOtherWebspaceMarks(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	shared := sampleItem("1", 100)
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws-x", "paperless", []item.Item{shared}); err != nil {
		t.Fatalf("seed ws-x: %v", err)
	}
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws-y", "paperless", []item.Item{shared}); err != nil {
		t.Fatalf("seed ws-y: %v", err)
	}
	if _, err := s.SetItemMarks(ctx, "ws-y", MarkKindExcluded, []string{shared.ID}); err != nil {
		t.Fatalf("SetItemMarks(ws-y): %v", err)
	}

	// A healthy ws-x sync that omits the shared item entirely.
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws-x", "paperless", nil); err != nil {
		t.Fatalf("ws-x sync: %v", err)
	}

	count, err := s.CountItemMarks(ctx, "ws-y", MarkKindExcluded)
	if err != nil {
		t.Fatalf("CountItemMarks(ws-y): %v", err)
	}
	if count != 1 {
		t.Fatalf("expected ws-y's mark on the same item id to survive a ws-x sync, got count=%d", count)
	}
}

// TestReplaceWebspaceSourceItems_DeallowlistClearsThatSourcesMarks proves a
// de-allowlist call (items=nil) removes that source's marks for that
// webspace (PD-02) — verified against store.go's own PD-02-named comment
// at the sweep's empty-kept-set branch.
func TestReplaceWebspaceSourceItems_DeallowlistClearsThatSourcesMarks(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	it := sampleItem("1", 100)
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{it}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := s.SetItemMarks(ctx, "ws", MarkKindExcluded, []string{it.ID}); err != nil {
		t.Fatalf("SetItemMarks: %v", err)
	}

	// De-allowlist: items=nil.
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", nil); err != nil {
		t.Fatalf("de-allowlist sync: %v", err)
	}

	count, err := s.CountItemMarks(ctx, "ws", MarkKindExcluded)
	if err != nil {
		t.Fatalf("CountItemMarks: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected the de-allowlisted source's mark to be swept, got count=%d", count)
	}
}

// TestReplaceWebspaceSourceItems_ReappearingItemIsUnexcluded proves that
// after a prune, the same item arriving again on a LATER sync appears in
// the normal stream, unexcluded — its mark is gone, not dormant (D-09).
func TestReplaceWebspaceSourceItems_ReappearingItemIsUnexcluded(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	it := sampleItem("1", 100)
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{it}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := s.SetItemMarks(ctx, "ws", MarkKindExcluded, []string{it.ID}); err != nil {
		t.Fatalf("SetItemMarks: %v", err)
	}

	// Healthy sync that omits the item — prunes the mark.
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", nil); err != nil {
		t.Fatalf("sync omitting item: %v", err)
	}

	// The item reappears on a LATER sync.
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{it}); err != nil {
		t.Fatalf("sync reintroducing item: %v", err)
	}

	items, err := s.StreamItems(ctx, "ws", nil, nil, 0, 0, ViewIncluded)
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}
	if len(items) != 1 || items[0].ID != it.ID {
		t.Fatalf("expected the reappeared item %q back in the stream, unexcluded, got %v", it.ID, idsOf(items))
	}
}

// TestReplaceWebspaceSourceItems_InterruptedSyncLeavesItemsAndMarksUnchanged
// proves the whole method — upsert, webspace_items replace, the mark
// sweep, and the webspaces upsert — is one atomic transaction: a write
// interrupted by a concurrent writer holding the database's write lock
// fails cleanly and leaves the item set AND the mark set exactly as they
// were beforehand, never a partially-applied sweep.
func TestReplaceWebspaceSourceItems_InterruptedSyncLeavesItemsAndMarksUnchanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	ctx := context.Background()

	kept := sampleItem("1", 200)
	vanished := sampleItem("2", 100)
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{kept, vanished}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := s.SetItemMarks(ctx, "ws", MarkKindExcluded, []string{vanished.ID}); err != nil {
		t.Fatalf("SetItemMarks: %v", err)
	}

	before, err := s.StreamItems(ctx, "ws", nil, nil, 0, 0, ViewIncluded)
	if err != nil {
		t.Fatalf("StreamItems (before): %v", err)
	}
	beforeCount, err := s.CountItemMarks(ctx, "ws", MarkKindExcluded)
	if err != nil {
		t.Fatalf("CountItemMarks (before): %v", err)
	}

	// A second, independent connection to the SAME file holds the
	// database's write lock for the duration of the attempted sync below —
	// SetMaxOpenConns(1) forces every statement through one held
	// connection, so the two raw Execs below run on it without an
	// intervening implicit commit.
	blocker, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open blocker connection: %v", err)
	}
	defer blocker.Close()
	blocker.SetMaxOpenConns(1)
	if _, err := blocker.Exec(`BEGIN IMMEDIATE`); err != nil {
		t.Fatalf("blocker BEGIN IMMEDIATE: %v", err)
	}

	// This sync — which would otherwise omit "vanished" and prune its
	// mark — cannot acquire the write lock and must fail cleanly.
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{kept}); err == nil {
		t.Fatal("expected ReplaceWebspaceSourceItems to fail while another writer holds the database lock")
	}

	if _, err := blocker.Exec(`ROLLBACK`); err != nil {
		t.Fatalf("blocker ROLLBACK: %v", err)
	}

	after, err := s.StreamItems(ctx, "ws", nil, nil, 0, 0, ViewIncluded)
	if err != nil {
		t.Fatalf("StreamItems (after): %v", err)
	}
	if len(after) != len(before) {
		t.Fatalf("expected the interrupted sync to leave items unchanged: before=%v after=%v", idsOf(before), idsOf(after))
	}
	afterCount, err := s.CountItemMarks(ctx, "ws", MarkKindExcluded)
	if err != nil {
		t.Fatalf("CountItemMarks (after): %v", err)
	}
	if afterCount != beforeCount {
		t.Fatalf("expected the interrupted sync to leave marks unchanged: before=%d after=%d", beforeCount, afterCount)
	}
}

// TestSetItemMarks_MarkForUnindexedItemOutranksLaterMatch proves what
// "always outranks whatever the automatic match rules say" means
// concretely (KERN-09): a mark written for an item id that is not yet
// indexed is accepted and stored, and becomes effective the moment that
// id later appears via a normal sync/match — a match rule that pulls the
// item in cannot resurrect it.
func TestSetItemMarks_MarkForUnindexedItemOutranksLaterMatch(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	it := sampleItem("1", 100)

	// The mark is written BEFORE the item has ever been synced/indexed
	// for this webspace — item_marks carries no FK to items(id), so this
	// must succeed.
	changed, err := s.SetItemMarks(ctx, "ws", MarkKindExcluded, []string{it.ID})
	if err != nil {
		t.Fatalf("SetItemMarks for an unindexed item: %v", err)
	}
	if changed != 1 {
		t.Errorf("expected changed=1 marking an unindexed item, got %d", changed)
	}

	// The item now arrives via an ordinary sync/match.
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{it}); err != nil {
		t.Fatalf("ReplaceWebspaceSourceItems: %v", err)
	}

	items, err := s.StreamItems(ctx, "ws", nil, nil, 0, 0, ViewIncluded)
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected the mark written before indexing to still filter %s out once matched, got %v", it.ID, idsOf(items))
	}
}
