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
	items, err := reopened.StreamItems(ctx, "ws", nil)
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
	itemsAfterReinsert, err := reopened.StreamItems(ctx, "ws", nil)
	if err != nil {
		t.Fatalf("StreamItems after re-insert: %v", err)
	}
	if len(itemsAfterReinsert) != 0 {
		t.Fatalf("the rebuild ate the mark: expected the surviving mark to still filter %s after it re-entered the index, got %v", it.ID, idsOf(itemsAfterReinsert))
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

	items, err := s.StreamItems(ctx, "ws", nil)
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected the mark written before indexing to still filter %s out once matched, got %v", it.ID, idsOf(items))
	}
}
