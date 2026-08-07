package index

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davison/topos/kernel/item"
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
		Source:                 sourceType,
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

	items, err := s.StreamItems(ctx, "ws", nil)
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

	itemsA, err := s.StreamItems(ctx, "webspace-a", nil)
	if err != nil {
		t.Fatalf("StreamItems webspace-a: %v", err)
	}
	itemsB, err := s.StreamItems(ctx, "webspace-b", nil)
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

	items, err := s.StreamItems(ctx, "ws", nil)
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

	items, err := s.StreamItems(ctx, "does-not-exist", nil)
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
	first, err := s.StreamItems(ctx, "ws", nil)
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}

	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", items); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	second, err := s.StreamItems(ctx, "ws", nil)
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

	items, err := s.StreamItems(ctx, "ws", nil)
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

	items, err := s.StreamItems(ctx, "ws", nil)
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

// TestStartAndFinishSyncRun is the load-bearing proof for the two-phase
// sync_runs write (02-02-PLAN.md Task 1): StartSyncRun inserts exactly one
// running row, SyncingSources reports the source as syncing while that
// row is unfinished, and FinishSyncRun updates THAT row (never inserting a
// second) so the total row count for the source stays at 1.
func TestStartAndFinishSyncRun(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	id, err := s.StartSyncRun(ctx, "paperless")
	if err != nil {
		t.Fatalf("StartSyncRun: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected a positive run id, got %d", id)
	}

	syncing, err := s.SyncingSources(ctx)
	if err != nil {
		t.Fatalf("SyncingSources: %v", err)
	}
	if !syncing["paperless"] {
		t.Error("expected paperless to be syncing between StartSyncRun and FinishSyncRun")
	}

	if err := s.FinishSyncRun(ctx, id, "ok", "", 12); err != nil {
		t.Fatalf("FinishSyncRun: %v", err)
	}

	syncing, err = s.SyncingSources(ctx)
	if err != nil {
		t.Fatalf("SyncingSources: %v", err)
	}
	if syncing["paperless"] {
		t.Error("expected paperless to no longer be syncing after FinishSyncRun")
	}

	runs, err := s.LatestSyncRunPerSource(ctx)
	if err != nil {
		t.Fatalf("LatestSyncRunPerSource: %v", err)
	}
	run, ok := runs["paperless"]
	if !ok {
		t.Fatal("expected a recorded run for paperless")
	}
	if run.Status != "ok" || run.ItemCount != 12 || run.FinishedUnix == 0 {
		t.Errorf("unexpected finished run: %+v", run)
	}

	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sync_runs WHERE source = ?`, "paperless").Scan(&count); err != nil {
		t.Fatalf("count sync_runs: %v", err)
	}
	if count != 1 {
		t.Errorf("expected exactly 1 sync_runs row (finish updates, never inserts), got %d", count)
	}
}

// TestLatestSyncRunPerSource_TwoSourcesReturnsBothNewest proves
// LatestSyncRunPerSource returns one entry per source_type, each the
// newest for its own source.
func TestLatestSyncRunPerSource_TwoSourcesReturnsBothNewest(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	id1, err := s.StartSyncRun(ctx, "paperless")
	if err != nil {
		t.Fatalf("StartSyncRun(paperless): %v", err)
	}
	if err := s.FinishSyncRun(ctx, id1, "ok", "", 3); err != nil {
		t.Fatalf("FinishSyncRun(paperless): %v", err)
	}
	id2, err := s.StartSyncRun(ctx, "silverbullet")
	if err != nil {
		t.Fatalf("StartSyncRun(silverbullet): %v", err)
	}
	if err := s.FinishSyncRun(ctx, id2, "error", "boom", 0); err != nil {
		t.Fatalf("FinishSyncRun(silverbullet): %v", err)
	}

	runs, err := s.LatestSyncRunPerSource(ctx)
	if err != nil {
		t.Fatalf("LatestSyncRunPerSource: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("expected 2 entries, got %d: %+v", len(runs), runs)
	}
	if runs["paperless"].Status != "ok" || runs["silverbullet"].Status != "error" {
		t.Errorf("unexpected runs: %+v", runs)
	}
}

// TestSyncingSources_UnrelatedSourceUnaffected proves a running row
// for one source does not mark a different, never-started source as
// syncing.
func TestSyncingSources_UnrelatedSourceUnaffected(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	if _, err := s.StartSyncRun(ctx, "paperless"); err != nil {
		t.Fatalf("StartSyncRun: %v", err)
	}

	syncing, err := s.SyncingSources(ctx)
	if err != nil {
		t.Fatalf("SyncingSources: %v", err)
	}
	if syncing["silverbullet"] {
		t.Error("expected silverbullet, which never started a run, to not be syncing")
	}
}

// startOrphanedRun inserts a sync_runs row for sourceType and abandons it
// unfinalised — reproducing a run stranded at status "running" by a kernel
// that died or was cancelled before FinishSyncRun could record its outcome.
func startOrphanedRun(t *testing.T, s *Store, sourceType string) {
	t.Helper()
	if _, err := s.StartSyncRun(context.Background(), sourceType); err != nil {
		t.Fatalf("StartSyncRun(%s): %v", sourceType, err)
	}
}

// TestSyncingSources_OrphanedRunDoesNotOutvoteLaterCompletedRun is the
// regression proof for the permanently-stuck "Syncing..." indicator: a
// source with one stranded "running" row followed by a COMPLETED run is not
// syncing. The query previously matched any running row at all, so a single
// orphan outvoted every later successful run and pinned the UI spinner on
// forever, across restarts. This test MUST fail if SyncingSources drops
// its latest-row-per-source restriction.
func TestSyncingSources_OrphanedRunDoesNotOutvoteLaterCompletedRun(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	startOrphanedRun(t, s, "proton")

	// One later run that starts AND finishes normally, exactly as every
	// scheduled sync after the orphan did.
	id, err := s.StartSyncRun(ctx, "proton")
	if err != nil {
		t.Fatalf("StartSyncRun: %v", err)
	}
	if err := s.FinishSyncRun(ctx, id, "ok", "", 44); err != nil {
		t.Fatalf("FinishSyncRun: %v", err)
	}

	syncing, err := s.SyncingSources(ctx)
	if err != nil {
		t.Fatalf("SyncingSources: %v", err)
	}
	if syncing["proton"] {
		t.Error("expected proton to not be syncing: its latest run completed, and an older orphaned row must not outvote it")
	}
}

// TestSyncingSources_OrphanBoundaries pins the neighbours either side of
// the fixed defect, so the latest-row restriction cannot be "fixed" into
// simply never reporting a source as syncing.
func TestSyncingSources_OrphanBoundaries(t *testing.T) {
	t.Run("orphan as the only row still reports syncing", func(t *testing.T) {
		s := openTestStore(t)
		startOrphanedRun(t, s, "proton")

		syncing, err := s.SyncingSources(context.Background())
		if err != nil {
			t.Fatalf("SyncingSources: %v", err)
		}
		if !syncing["proton"] {
			t.Error("expected proton to be syncing: its only (latest) run is unfinished")
		}
	})

	t.Run("a genuinely in-flight run after an orphan still reports syncing", func(t *testing.T) {
		s := openTestStore(t)
		ctx := context.Background()
		startOrphanedRun(t, s, "proton")

		id, err := s.StartSyncRun(ctx, "proton")
		if err != nil {
			t.Fatalf("StartSyncRun: %v", err)
		}
		syncing, err := s.SyncingSources(ctx)
		if err != nil {
			t.Fatalf("SyncingSources: %v", err)
		}
		if !syncing["proton"] {
			t.Error("expected proton to be syncing: its latest run is genuinely in flight")
		}

		if err := s.FinishSyncRun(ctx, id, "ok", "", 1); err != nil {
			t.Fatalf("FinishSyncRun: %v", err)
		}
		syncing, err = s.SyncingSources(ctx)
		if err != nil {
			t.Fatalf("SyncingSources: %v", err)
		}
		if syncing["proton"] {
			t.Error("expected proton to stop syncing once its latest run finished")
		}
	})

	t.Run("one source's orphan does not mark another source syncing", func(t *testing.T) {
		s := openTestStore(t)
		ctx := context.Background()
		startOrphanedRun(t, s, "proton")

		id, err := s.StartSyncRun(ctx, "paperless")
		if err != nil {
			t.Fatalf("StartSyncRun: %v", err)
		}
		if err := s.FinishSyncRun(ctx, id, "ok", "", 37); err != nil {
			t.Fatalf("FinishSyncRun: %v", err)
		}

		syncing, err := s.SyncingSources(ctx)
		if err != nil {
			t.Fatalf("SyncingSources: %v", err)
		}
		if syncing["paperless"] {
			t.Error("expected paperless to not be syncing: proton's orphan must not leak across sources")
		}
	})
}

// TestReconcileInterruptedSyncRuns proves the startup repair finalises
// stranded "running" rows — the only path in the system that ever heals an
// orphan left by a previous kernel session — without touching rows that
// already recorded an outcome.
func TestReconcileInterruptedSyncRuns(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	startOrphanedRun(t, s, "proton")
	startOrphanedRun(t, s, "signal")

	done, err := s.StartSyncRun(ctx, "paperless")
	if err != nil {
		t.Fatalf("StartSyncRun: %v", err)
	}
	if err := s.FinishSyncRun(ctx, done, "ok", "", 37); err != nil {
		t.Fatalf("FinishSyncRun: %v", err)
	}

	n, err := s.ReconcileInterruptedSyncRuns(ctx)
	if err != nil {
		t.Fatalf("ReconcileInterruptedSyncRuns: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 stranded rows repaired, got %d", n)
	}

	syncing, err := s.SyncingSources(ctx)
	if err != nil {
		t.Fatalf("SyncingSources: %v", err)
	}
	if len(syncing) != 0 {
		t.Errorf("expected no source to be syncing after reconciliation, got %v", syncing)
	}

	runs, err := s.LatestSyncRunPerSource(ctx)
	if err != nil {
		t.Fatalf("LatestSyncRunPerSource: %v", err)
	}
	for _, st := range []string{"proton", "signal"} {
		run := runs[st]
		if run.Status != "error" {
			t.Errorf("%s: expected repaired status \"error\", got %q", st, run.Status)
		}
		if run.Error == "" {
			t.Errorf("%s: expected a non-empty interrupted message on the repaired row", st)
		}
		if run.FinishedUnix == 0 {
			t.Errorf("%s: expected the repaired row to record a finished time", st)
		}
	}
	if runs["paperless"].Status != "ok" || runs["paperless"].Error != "" {
		t.Errorf("reconciliation must not touch an already-finalised row, got: %+v", runs["paperless"])
	}

	// Idempotent: a second run with nothing stranded repairs nothing.
	n, err = s.ReconcileInterruptedSyncRuns(ctx)
	if err != nil {
		t.Fatalf("ReconcileInterruptedSyncRuns (second call): %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 rows repaired on a second call, got %d", n)
	}
}

// searchableItem builds an item whose title/preview are distinguishable
// search targets, unlike sampleItem's fixed "Doc {id}"/"preview text".
func searchableItem(sourceID string, ts int64, title, preview string) item.Item {
	it := sampleItem(sourceID, ts)
	it.Title = title
	it.Preview = preview
	return it
}

// TestSearch_MatchesOnlyTheItemContainingTheWord is the core positive case:
// seeding two items into a webspace and searching for a word appearing
// only in the first returns exactly that item.
func TestSearch_MatchesOnlyTheItemContainingTheWord(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	target := searchableItem("1", 100, "Boiler service invoice", "annual boiler service")
	other := searchableItem("2", 200, "Garden fence quote", "replacing the back fence")

	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{target, other}); err != nil {
		t.Fatalf("ReplaceWebspaceSourceItems: %v", err)
	}

	results, err := s.Search(ctx, "ws", "boiler", nil)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected exactly 1 result, got %d: %+v", len(results), results)
	}
	if results[0].Item.ID != target.ID {
		t.Errorf("expected match %q, got %q", target.ID, results[0].Item.ID)
	}
}

// TestSearch_NoMatchReturnsEmptySliceNilError proves a query matching no
// item returns an empty (non-nil) slice and a nil error.
func TestSearch_NoMatchReturnsEmptySliceNilError(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	it := searchableItem("1", 100, "Boiler service invoice", "annual boiler service")
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{it}); err != nil {
		t.Fatalf("ReplaceWebspaceSourceItems: %v", err)
	}

	results, err := s.Search(ctx, "ws", "nonexistentword", nil)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if results == nil {
		t.Error("expected a non-nil empty slice, got nil")
	}
	if len(results) != 0 {
		t.Errorf("expected zero results, got %d", len(results))
	}
}

// TestSearch_ScopedToOneWebspaceOnly is the load-bearing proof for
// T-03-15: an item associated only with webspace B must never be returned
// by a search of webspace A, even when it is the only item matching the
// query.
func TestSearch_ScopedToOneWebspaceOnly(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	onlyInB := searchableItem("1", 100, "Boiler service invoice", "annual boiler service")
	if err := s.ReplaceWebspaceSourceItems(ctx, "webspace-b", "paperless", []item.Item{onlyInB}); err != nil {
		t.Fatalf("seed webspace-b: %v", err)
	}
	// webspace-a must be a known webspace (so Search isn't trivially empty
	// because of a 404-shaped "never synced" case), just with no matching
	// item.
	if err := s.ReplaceWebspaceSourceItems(ctx, "webspace-a", "paperless", nil); err != nil {
		t.Fatalf("seed webspace-a: %v", err)
	}

	results, err := s.Search(ctx, "webspace-a", "boiler", nil)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected an item indexed only in webspace-b to be absent from a webspace-a search, got %+v", results)
	}

	// Sanity: the same query against webspace-b (where the item actually
	// lives) does return it.
	resultsB, err := s.Search(ctx, "webspace-b", "boiler", nil)
	if err != nil {
		t.Fatalf("Search webspace-b: %v", err)
	}
	if len(resultsB) != 1 || resultsB[0].Item.ID != onlyInB.ID {
		t.Fatalf("expected webspace-b search to find %q, got %+v", onlyInB.ID, resultsB)
	}
}

// TestSearch_TitleMatchRanksAboveviewMatch proves ordering: an item whose
// title matches the query ranks above an item where only the preview
// matches.
func TestSearch_TitleMatchRanksAbovePreviewMatch(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	titleMatch := searchableItem("1", 100, "Boiler quote", "a document about heating")
	previewMatch := searchableItem("2", 200, "Heating document", "quote from the boiler engineer")

	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{titleMatch, previewMatch}); err != nil {
		t.Fatalf("ReplaceWebspaceSourceItems: %v", err)
	}

	results, err := s.Search(ctx, "ws", "boiler", nil)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d: %+v", len(results), results)
	}
	if results[0].Item.ID != titleMatch.ID {
		t.Errorf("expected the title match to rank first, got order %v", []string{results[0].Item.ID, results[1].Item.ID})
	}
}

// TestSearch_SnippetContainsDelimiters proves the snippet for a matching
// result is wrapped in the configured open/close delimiters.
func TestSearch_SnippetContainsDelimiters(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	it := searchableItem("1", 100, "Boiler service invoice", "annual boiler service")
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{it}); err != nil {
		t.Fatalf("ReplaceWebspaceSourceItems: %v", err)
	}

	results, err := s.Search(ctx, "ws", "boiler", nil)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !strings.Contains(results[0].Snippet, SnippetOpen) || !strings.Contains(results[0].Snippet, SnippetClose) {
		t.Errorf("expected snippet to contain both delimiters, got %q", results[0].Snippet)
	}
}

// TestSearch_EmptyOrWhitespaceQueryReturnsEmptyNoQuery proves ftsQuery's
// degradation path is reached from Search itself, not just unit-tested in
// isolation.
func TestSearch_EmptyOrWhitespaceQueryReturnsEmptyNoQuery(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	it := searchableItem("1", 100, "Boiler service invoice", "annual boiler service")
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{it}); err != nil {
		t.Fatalf("ReplaceWebspaceSourceItems: %v", err)
	}

	for _, q := range []string{"", "   ", `"`} {
		results, err := s.Search(ctx, "ws", q, nil)
		if err != nil {
			t.Fatalf("Search(%q): %v", q, err)
		}
		if len(results) != 0 {
			t.Errorf("Search(%q): expected zero results, got %d", q, len(results))
		}
	}
}

// TestSearch_FilterTermsANDCombineWithLiveQuery (07-01-PLAN.md Task 2,
// D-18) proves a live search AND-combines with the saved filter stack
// rather than escaping it: with filter = ["boiler"] and a live query of
// "quote", an item matching only "quote" must be absent from the result
// set — the saved filter narrows the live query, not the other way round.
func TestSearch_FilterTermsANDCombineWithLiveQuery(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	both := searchableItem("1", 100, "Boiler quote", "a boiler repair quote")
	queryOnly := searchableItem("2", 200, "Garden fence quote", "replacing the back fence")
	filterOnly := searchableItem("3", 300, "Boiler service invoice", "annual boiler service")

	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{both, queryOnly, filterOnly}); err != nil {
		t.Fatalf("ReplaceWebspaceSourceItems: %v", err)
	}

	results, err := s.Search(ctx, "ws", "quote", []string{"boiler"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected exactly 1 result (matching both the filter and the live query), got %d: %+v", len(results), results)
	}
	if results[0].Item.ID != both.ID {
		t.Errorf("expected the item matching both %q and %q, got %q", "boiler", "quote", results[0].Item.ID)
	}
}

// TestSearch_FilterOnlyNoLiveQueryStillQueries proves Store.Search's
// empty-result short circuit now depends on BOTH inputs (07-01-PLAN.md
// Task 2): a call with an empty live query but a non-empty filter stack
// must still query and rank by relevance, not return early as if there
// were no query at all.
func TestSearch_FilterOnlyNoLiveQueryStillQueries(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	it := searchableItem("1", 100, "Boiler service invoice", "annual boiler service")
	other := searchableItem("2", 200, "Garden fence quote", "replacing the back fence")
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{it, other}); err != nil {
		t.Fatalf("ReplaceWebspaceSourceItems: %v", err)
	}

	results, err := s.Search(ctx, "ws", "", []string{"boiler"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 || results[0].Item.ID != it.ID {
		t.Fatalf("expected the filter-only call to return exactly the matching item, got %+v", results)
	}
}

// TestBuildMatchQuery_FilterTermsCarryNoTrailingStar proves a saved filter
// term means the word, not a prefix (07-01-PLAN.md Task 1's own contract,
// re-pinned here against Task 2's AND-combination change): an item
// containing only "boilerplate" and not the exact word "boiler" must not
// match a filter of ["boiler"].
func TestBuildMatchQuery_FilterTermsCarryNoTrailingStar(t *testing.T) {
	match := BuildMatchQuery([]string{"boiler"}, "")
	if strings.Contains(match, "boiler*") || strings.HasSuffix(match, "*") {
		t.Fatalf("expected no trailing * on a filter-only match expression, got %q", match)
	}
}

func TestStreamItems_FilterTermCarriesNoTrailingStar(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	exact := searchableItem("1", 100, "Boiler service invoice", "annual boiler service")
	prefixOnly := searchableItem("2", 200, "Boilerplate legal text", "standard boilerplate wording")
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{exact, prefixOnly}); err != nil {
		t.Fatalf("ReplaceWebspaceSourceItems: %v", err)
	}

	items, err := s.StreamItems(ctx, "ws", []string{"boiler"})
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}
	if len(items) != 1 || items[0].ID != exact.ID {
		t.Fatalf("expected only the exact-word match, got %+v", items)
	}
}

// TestStreamItems_EmptyFilterReturnsIdenticalRowsAndOrder (07-01-PLAN.md
// Task 3, D-16/D-18) pins the byte-identical-to-pre-Phase-7 contract at
// the row level: both a nil and an explicitly-empty filter slice must
// return every row in the same newest-first order StreamItems has always
// used, with no items_fts join involved at all.
func TestStreamItems_EmptyFilterReturnsIdenticalRowsAndOrder(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	older := sampleItem("1", 100)
	newer := sampleItem("2", 200)
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{older, newer}); err != nil {
		t.Fatalf("ReplaceWebspaceSourceItems: %v", err)
	}

	withNilFilter, err := s.StreamItems(ctx, "ws", nil)
	if err != nil {
		t.Fatalf("StreamItems(nil): %v", err)
	}
	withEmptyFilter, err := s.StreamItems(ctx, "ws", []string{})
	if err != nil {
		t.Fatalf("StreamItems([]string{}): %v", err)
	}

	if len(withNilFilter) != 2 || len(withEmptyFilter) != 2 {
		t.Fatalf("expected 2 items for both a nil and an empty filter, got nil=%d empty=%d", len(withNilFilter), len(withEmptyFilter))
	}
	if withNilFilter[0].ID != newer.ID || withNilFilter[1].ID != older.ID {
		t.Fatalf("expected newest-first order with a nil filter, got %v", []string{withNilFilter[0].ID, withNilFilter[1].ID})
	}
	if withEmptyFilter[0].ID != newer.ID || withEmptyFilter[1].ID != older.ID {
		t.Fatalf("expected newest-first order with an empty filter, got %v", []string{withEmptyFilter[0].ID, withEmptyFilter[1].ID})
	}
}

// TestStreamItems_SingleTermFilterNarrowsToMatchingRowsChronological
// (Task 3) proves a single-term filter narrows to only the matching rows,
// preserving the same newest-first chronological order among the matches
// — never bm25 rank, which is Search's ordering, not StreamItems'.
func TestStreamItems_SingleTermFilterNarrowsToMatchingRowsChronological(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	matchOlder := searchableItem("1", 100, "Boiler service invoice", "annual boiler service")
	matchNewer := searchableItem("2", 200, "Boiler quote", "a boiler repair quote")
	nonMatch := searchableItem("3", 300, "Garden fence quote", "replacing the back fence")

	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{matchOlder, matchNewer, nonMatch}); err != nil {
		t.Fatalf("ReplaceWebspaceSourceItems: %v", err)
	}

	items, err := s.StreamItems(ctx, "ws", []string{"boiler"})
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected exactly 2 matching items, got %d: %+v", len(items), items)
	}
	if items[0].ID != matchNewer.ID || items[1].ID != matchOlder.ID {
		t.Fatalf("expected the two matches in newest-first chronological order, got %v", []string{items[0].ID, items[1].ID})
	}
}

// TestStreamItems_TwoTermFilterRequiresBoth (Task 3, D-18) proves a
// two-term filter stack is AND-ed, not OR-ed: an item matching only one
// of the two terms must be absent.
func TestStreamItems_TwoTermFilterRequiresBoth(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	both := searchableItem("1", 100, "Boiler quote", "a boiler repair quote")
	boilerOnly := searchableItem("2", 200, "Boiler service invoice", "annual boiler service")
	quoteOnly := searchableItem("3", 300, "Garden fence quote", "replacing the back fence")

	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{both, boilerOnly, quoteOnly}); err != nil {
		t.Fatalf("ReplaceWebspaceSourceItems: %v", err)
	}

	items, err := s.StreamItems(ctx, "ws", []string{"boiler", "quote"})
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}
	if len(items) != 1 || items[0].ID != both.ID {
		t.Fatalf("expected exactly the item matching both filter terms, got %+v", items)
	}
}

// TestStreamItems_HostileFilterTermNeverErrors (Task 3) mirrors Search's
// own fts5-error degradation (03-RESEARCH.md Pattern 3): StreamItems must
// never surface an error because of what's in a filter term.
// BuildMatchQuery's phrase-quoting makes a genuine FTS5 MATCH syntax error
// unreachable in principle (every surviving term is quote-stripped then
// wrapped in a phrase), so this exercises the class of hostile-shaped
// input that protection exists for, rather than asserting one specific
// string is proven to hit the "fts5" error-string branch.
func TestStreamItems_HostileFilterTermNeverErrors(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	it := sampleItem("1", 100)
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{it}); err != nil {
		t.Fatalf("ReplaceWebspaceSourceItems: %v", err)
	}

	for _, term := range []string{`"`, `*`, `(`, `)`, `NEAR`, `AND OR NOT`} {
		if _, err := s.StreamItems(ctx, "ws", []string{term}); err != nil {
			t.Fatalf("StreamItems([]string{%q}): expected a nil error regardless of what the filter term contains, got: %v", term, err)
		}
	}
}

// TestBackfill_ReopeningAPreexistingIndexFindsItsItems simulates opening a
// pre-Phase-3 index file: an index already holding items, whose items_fts
// table and triggers are dropped directly through a raw connection (as if
// this index predated the FTS5 schema addition), must have those items
// searchable again after being reopened through Open — proving the
// first-creation backfill, not just the sync triggers, keeps items_fts
// populated.
func TestBackfill_ReopeningAPreexistingIndexFindsItsItems(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.db")

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	it := searchableItem("1", 100, "Boiler service invoice", "annual boiler service")
	if err := s.ReplaceWebspaceSourceItems(context.Background(), "ws", "paperless", []item.Item{it}); err != nil {
		t.Fatalf("ReplaceWebspaceSourceItems: %v", err)
	}

	// Simulate a pre-Phase-3 index file by dropping items_fts and its
	// triggers directly.
	for _, stmt := range []string{
		`DROP TRIGGER IF EXISTS items_ai`,
		`DROP TRIGGER IF EXISTS items_ad`,
		`DROP TRIGGER IF EXISTS items_au`,
		`DROP TABLE IF EXISTS items_fts`,
	} {
		if _, err := s.db.Exec(stmt); err != nil {
			t.Fatalf("simulate pre-Phase-3 index (%s): %v", stmt, err)
		}
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() { reopened.Close() })

	results, err := reopened.Search(context.Background(), "ws", "boiler", nil)
	if err != nil {
		t.Fatalf("Search after reopen: %v", err)
	}
	if len(results) != 1 || results[0].Item.ID != it.ID {
		t.Fatalf("expected the pre-existing item %q to be found after backfill, got %+v", it.ID, results)
	}
}
