package index

import (
	"context"
	"testing"

	"github.com/davison/topos/kernel/item"
)

// RenameWebspace (M3-R2, #77): items, marks and the sync record all
// follow the new name in one transaction; the old name is empty after.
func TestRenameWebspace_CarriesItemsMarksAndSyncRecord(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	if err := s.ReplaceWebspaceSourceItems(ctx, "old", "paperless", []item.Item{
		sampleItem("1", 100), sampleItem("2", 200),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := s.SetItemMarks(ctx, "old", "excluded", []string{sampleItem("1", 100).ID}); err != nil {
		t.Fatalf("mark: %v", err)
	}

	if err := s.RenameWebspace(ctx, "old", "new"); err != nil {
		t.Fatalf("RenameWebspace: %v", err)
	}

	items, err := s.StreamItems(ctx, "new", nil, nil, 0, 0, ViewIncluded)
	if err != nil {
		t.Fatalf("StreamItems(new): %v", err)
	}
	if len(items) != 1 { // one excluded — the mark survived under the new name
		t.Fatalf("new name: got %d included items, want 1 (the exclusion must survive)", len(items))
	}
	excluded, _ := s.StreamItems(ctx, "new", nil, nil, 0, 0, ViewExcluded)
	if len(excluded) != 1 {
		t.Fatalf("the excluded mark did not follow the rename: %d", len(excluded))
	}
	old, _ := s.StreamItems(ctx, "old", nil, nil, 0, 0, ViewIncluded)
	if len(old) != 0 {
		t.Fatalf("the old name still holds %d items", len(old))
	}
	exists, err := s.WebspaceExists(ctx, "new")
	if err != nil || !exists {
		t.Fatalf("the sync record did not follow: %v %v", exists, err)
	}
}

// A deleted namesake's stale rows under the destination are cleared, not
// merged (PR #80 review round 1): the renamed space arrives with exactly
// its own items and marks.
func TestRenameWebspace_ClearsADeletedNamesakesRows(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	// The destination name once existed and was deleted — its rows remain.
	if err := s.ReplaceWebspaceSourceItems(ctx, "new", "paperless", []item.Item{
		sampleItem("9", 900),
	}); err != nil {
		t.Fatalf("seed dead namesake: %v", err)
	}
	if _, err := s.SetItemMarks(ctx, "new", "excluded", []string{sampleItem("9", 900).ID}); err != nil {
		t.Fatalf("mark dead namesake: %v", err)
	}
	if err := s.ReplaceWebspaceSourceItems(ctx, "old", "paperless", []item.Item{
		sampleItem("1", 100),
	}); err != nil {
		t.Fatalf("seed source: %v", err)
	}

	if err := s.RenameWebspace(ctx, "old", "new"); err != nil {
		t.Fatalf("RenameWebspace: %v", err)
	}
	items, err := s.StreamItems(ctx, "new", nil, nil, 0, 0, ViewIncluded)
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}
	if len(items) != 1 || items[0].SourceID != "1" {
		t.Fatalf("destination must hold exactly the renamed space's own items: %+v", items)
	}
	excluded, _ := s.StreamItems(ctx, "new", nil, nil, 0, 0, ViewExcluded)
	if len(excluded) != 0 {
		t.Fatalf("a dead namesake's exclusion leaked into the renamed space: %d", len(excluded))
	}
}
