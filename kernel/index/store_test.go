package index

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/davison/webspaces/kernel/item"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "index.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func sampleItem(sourceID string, ts int64) item.Item {
	return sampleItemForSource("paperless", sourceID, ts)
}

func sampleItemForSource(sourceType, sourceID string, ts int64) item.Item {
	return item.Item{
		ID:                     item.ID(sourceType, sourceID),
		SourceType:             sourceType,
		SourceID:               sourceID,
		Title:                  "Doc " + sourceID,
		Preview:                "preview text",
		TimestampUnix:          ts,
		SecondaryTimestampUnix: ts,
		Fidelity:               item.FidelityExact,
		DeepLink:               "http://" + sourceType + ".lan:8000/documents/" + sourceID,
		Labels:                 []string{"House"},
		Provenance:             map[string]string{"source_type": sourceType},
	}
}

// TestReplaceWebspaceSourceItems_OtherSourceRowsUntouched is the load-
// bearing proof for the sync identity promotion from "webspace" to
// "(webspace, source_type)" (02-01-PLAN.md's objective): seeding rows for
// two distinct source types in one webspace, then replacing only one
// source's rows, must leave the other source's webspace_items rows (same
// item ids) completely untouched — not merely re-inserted with the same
// content, but never deleted even transiently within the call.
func TestReplaceWebspaceSourceItems_OtherSourceRowsUntouched(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	paperlessItem := sampleItemForSource("paperless", "1", 100)
	silverbulletItem := sampleItemForSource("silverbullet", "notes/a", 200)

	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{paperlessItem}); err != nil {
		t.Fatalf("seed paperless: %v", err)
	}
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "silverbullet", []item.Item{silverbulletItem}); err != nil {
		t.Fatalf("seed silverbullet: %v", err)
	}

	// Replace only silverbullet's contribution, with a different item this
	// time — paperless's row must survive unchanged.
	newSilverbulletItem := sampleItemForSource("silverbullet", "notes/b", 300)
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "silverbullet", []item.Item{newSilverbulletItem}); err != nil {
		t.Fatalf("replace silverbullet: %v", err)
	}

	items, err := s.StreamItems(ctx, "ws")
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}
	ids := map[string]bool{}
	for _, it := range items {
		ids[it.ID] = true
	}
	if !ids[paperlessItem.ID] {
		t.Errorf("expected paperless item %q to still be present after a silverbullet-only replace, got items: %v", paperlessItem.ID, idsOf(items))
	}
	if ids[silverbulletItem.ID] {
		t.Errorf("expected the OLD silverbullet item %q to be gone after replace, got items: %v", silverbulletItem.ID, idsOf(items))
	}
	if !ids[newSilverbulletItem.ID] {
		t.Errorf("expected the NEW silverbullet item %q to be present, got items: %v", newSilverbulletItem.ID, idsOf(items))
	}
	if len(items) != 2 {
		t.Errorf("expected exactly 2 items (1 paperless + 1 silverbullet), got %d: %v", len(items), idsOf(items))
	}
}

// TestReplaceWebspaceSourceItems_EmptyItemsStillMarksWebspaceKnown proves a
// source that matched zero items for a webspace still registers that
// webspace as known (WebspaceExists true) — required so a webspace with a
// working source and a not-yet-configured/still-erroring sibling source
// does not incorrectly 404 as "never synced" (02-RESEARCH.md Critical
// Architecture Finding).
func TestReplaceWebspaceSourceItems_EmptyItemsStillMarksWebspaceKnown(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "silverbullet", nil); err != nil {
		t.Fatalf("ReplaceWebspaceSourceItems with nil items: %v", err)
	}

	exists, err := s.WebspaceExists(ctx, "ws")
	if err != nil {
		t.Fatalf("WebspaceExists: %v", err)
	}
	if !exists {
		t.Error("expected webspace 'ws' to be known after a zero-item sync from one source")
	}
}

func TestReplaceWebspaceSourceItems_TwoWebspacesShareItemNoCollision(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	shared := sampleItem("1", 100)

	if err := s.ReplaceWebspaceSourceItems(ctx, "webspace-a", "paperless", []item.Item{shared}); err != nil {
		t.Fatalf("ReplaceWebspaceItems webspace-a: %v", err)
	}
	if err := s.ReplaceWebspaceSourceItems(ctx, "webspace-b", "paperless", []item.Item{shared}); err != nil {
		t.Fatalf("ReplaceWebspaceItems webspace-b: %v", err)
	}

	itemsA, err := s.StreamItems(ctx, "webspace-a")
	if err != nil {
		t.Fatalf("StreamItems webspace-a: %v", err)
	}
	itemsB, err := s.StreamItems(ctx, "webspace-b")
	if err != nil {
		t.Fatalf("StreamItems webspace-b: %v", err)
	}
	if len(itemsA) != 1 || len(itemsB) != 1 {
		t.Fatalf("expected 1 item in each webspace, got %d / %d", len(itemsA), len(itemsB))
	}

	var itemCount int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM items`).Scan(&itemCount); err != nil {
		t.Fatalf("count items: %v", err)
	}
	if itemCount != 1 {
		t.Errorf("expected exactly one items row (shared, not duplicated), got %d", itemCount)
	}

	var joinCount int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM webspace_items`).Scan(&joinCount); err != nil {
		t.Fatalf("count webspace_items: %v", err)
	}
	if joinCount != 2 {
		t.Errorf("expected two distinct webspace_items rows, got %d", joinCount)
	}
}

func TestStreamItems_TotalOrderingWithTieBreak(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	// Two items share the same primary timestamp; secondary timestamp and
	// then id must produce a stable, deterministic order.
	older := sampleItem("2", 200)
	older.SecondaryTimestampUnix = 50

	newer := sampleItem("1", 200)
	newer.SecondaryTimestampUnix = 999

	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{older, newer}); err != nil {
		t.Fatalf("ReplaceWebspaceItems: %v", err)
	}

	items, err := s.StreamItems(ctx, "ws")
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].ID != newer.ID {
		t.Errorf("expected higher secondary_timestamp_unix first, got order %v then %v", items[0].ID, items[1].ID)
	}
}

func TestStreamItems_UnknownWebspaceReturnsEmpty(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	items, err := s.StreamItems(ctx, "does-not-exist")
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected zero items, got %d", len(items))
	}
}

func TestReplaceWebspaceSourceItems_Idempotent(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	items := []item.Item{sampleItem("1", 100), sampleItem("2", 200)}

	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", items); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	first, err := s.StreamItems(ctx, "ws")
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}

	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", items); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	second, err := s.StreamItems(ctx, "ws")
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}

	if len(first) != len(second) {
		t.Fatalf("item count changed across idempotent syncs: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].ID != second[i].ID {
			t.Errorf("order changed at index %d: %s vs %s", i, first[i].ID, second[i].ID)
		}
	}
}

// TestStreamItems_OrdersByPrimaryTimestampDescendingAcrossInsertOrder pins
// the primary sort key of the total order StreamHandler depends on
// (01-04-PLAN.md Task 1): items are inserted out of chronological order,
// so SQLite's natural row order would return low, high, mid — only the
// ORDER BY items.timestamp_unix DESC clause produces high, mid, low.
func TestStreamItems_OrdersByPrimaryTimestampDescendingAcrossInsertOrder(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	low := sampleItem("low", 100)
	high := sampleItem("high", 300)
	mid := sampleItem("mid", 200)

	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{low, high, mid}); err != nil {
		t.Fatalf("ReplaceWebspaceItems: %v", err)
	}

	items, err := s.StreamItems(ctx, "ws")
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}
	wantOrder := []string{high.ID, mid.ID, low.ID}
	if got := idsOf(items); !equalIDOrder(got, wantOrder) {
		t.Errorf("expected order %v, got %v", wantOrder, got)
	}
}

// TestStreamItems_TiesOnBothTimestampsBreakByIDAscending pins the final
// tie-break key (id ASC) of the total order: every item shares both the
// primary and secondary timestamp, so only the id-ASC clause can produce a
// deterministic result. Inserted in descending id order, so dropping the
// final ORDER BY clause would return the reverse of the expected order.
func TestStreamItems_TiesOnBothTimestampsBreakByIDAscending(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	c := sampleItem("charlie", 500)
	c.SecondaryTimestampUnix = 500
	b := sampleItem("bravo", 500)
	b.SecondaryTimestampUnix = 500
	a := sampleItem("alpha", 500)
	a.SecondaryTimestampUnix = 500

	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{c, b, a}); err != nil {
		t.Fatalf("ReplaceWebspaceItems: %v", err)
	}

	items, err := s.StreamItems(ctx, "ws")
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}
	// "paperless:alpha" < "paperless:bravo" < "paperless:charlie" lexically.
	wantOrder := []string{a.ID, b.ID, c.ID}
	if got := idsOf(items); !equalIDOrder(got, wantOrder) {
		t.Errorf("expected order %v, got %v", wantOrder, got)
	}
}

func idsOf(items []item.Item) []string {
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	return ids
}

func equalIDOrder(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestRecordSyncRunAndLatest(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	if err := s.RecordSyncRun(ctx, SyncRun{
		SourceType: "paperless", StartedUnix: 1, FinishedUnix: 2, Status: "ok", ItemCount: 5,
	}); err != nil {
		t.Fatalf("RecordSyncRun: %v", err)
	}

	run, ok, err := s.LatestSyncRun(ctx)
	if err != nil {
		t.Fatalf("LatestSyncRun: %v", err)
	}
	if !ok {
		t.Fatal("expected a sync run to be present")
	}
	if run.Status != "ok" || run.ItemCount != 5 {
		t.Errorf("unexpected sync run: %+v", run)
	}
}
