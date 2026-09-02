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

	items, err := s.StreamItems(ctx, "new", nil, nil, ViewIncluded)
	if err != nil {
		t.Fatalf("StreamItems(new): %v", err)
	}
	if len(items) != 1 { // one excluded — the mark survived under the new name
		t.Fatalf("new name: got %d included items, want 1 (the exclusion must survive)", len(items))
	}
	excluded, _ := s.StreamItems(ctx, "new", nil, nil, ViewExcluded)
	if len(excluded) != 1 {
		t.Fatalf("the excluded mark did not follow the rename: %d", len(excluded))
	}
	old, _ := s.StreamItems(ctx, "old", nil, nil, ViewIncluded)
	if len(old) != 0 {
		t.Fatalf("the old name still holds %d items", len(old))
	}
	exists, err := s.WebspaceExists(ctx, "new")
	if err != nil || !exists {
		t.Fatalf("the sync record did not follow: %v %v", exists, err)
	}
}
