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
	return item.Item{
		ID:                     item.ID("paperless", sourceID),
		SourceType:             "paperless",
		SourceID:               sourceID,
		Title:                  "Doc " + sourceID,
		Preview:                "preview text",
		TimestampUnix:          ts,
		SecondaryTimestampUnix: ts,
		Fidelity:               item.FidelityExact,
		DeepLink:               "http://paperless.lan:8000/documents/" + sourceID,
		Labels:                 []string{"House"},
		Provenance:             map[string]string{"source_type": "paperless"},
	}
}

func TestReplaceWebspaceItems_TwoWebspacesShareItemNoCollision(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	shared := sampleItem("1", 100)

	if err := s.ReplaceWebspaceItems(ctx, "webspace-a", []item.Item{shared}); err != nil {
		t.Fatalf("ReplaceWebspaceItems webspace-a: %v", err)
	}
	if err := s.ReplaceWebspaceItems(ctx, "webspace-b", []item.Item{shared}); err != nil {
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

	if err := s.ReplaceWebspaceItems(ctx, "ws", []item.Item{older, newer}); err != nil {
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

func TestReplaceWebspaceItems_Idempotent(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	items := []item.Item{sampleItem("1", 100), sampleItem("2", 200)}

	if err := s.ReplaceWebspaceItems(ctx, "ws", items); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	first, err := s.StreamItems(ctx, "ws")
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}

	if err := s.ReplaceWebspaceItems(ctx, "ws", items); err != nil {
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
