package index

import (
	"context"
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

	itemsA, err := s.StreamItems(ctx, "ws-a", nil)
	if err != nil {
		t.Fatalf("StreamItems(ws-a): %v", err)
	}
	if len(itemsA) != 1 || itemsA[0].ID != newer.ID {
		t.Fatalf("expected ws-a to carry exactly [%s] after excluding %s, got %v", newer.ID, older.ID, idsOf(itemsA))
	}

	itemsB, err := s.StreamItems(ctx, "ws-b", nil)
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

	items, err := s.StreamItems(ctx, "ws", nil)
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

	resultsA, err := s.Search(ctx, "ws-a", "invoice", nil)
	if err != nil {
		t.Fatalf("Search(ws-a): %v", err)
	}
	if len(resultsA) != 0 {
		t.Errorf("expected Search(ws-a) to omit the excluded item, got %d result(s)", len(resultsA))
	}

	resultsB, err := s.Search(ctx, "ws-b", "invoice", nil)
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
