package index

import (
	"context"
	"testing"

	"github.com/davison/topos/kernel/item"
)

// The date narrowing at the index (M3-R1, #70): bounds are inclusive,
// either side may be open, and zero/zero is byte-identical to no clause.
func TestStreamAndSearch_DateRange(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	mk := func(id string, ts int64) item.Item {
		it := sampleItem(id, ts)
		it.Title = "boiler day " + id
		return it
	}
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{
		mk("1", 100), mk("2", 200), mk("3", 300),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	items, err := s.StreamItems(ctx, "ws", nil, nil, 150, 250, ViewIncluded)
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}
	if len(items) != 1 || items[0].SourceID != "2" {
		t.Fatalf("bounded range: %+v", items)
	}

	items, _ = s.StreamItems(ctx, "ws", nil, nil, 200, 0, ViewIncluded)
	if len(items) != 2 {
		t.Fatalf("open-ended from (inclusive): got %d", len(items))
	}
	items, _ = s.StreamItems(ctx, "ws", nil, nil, 0, 200, ViewIncluded)
	if len(items) != 2 {
		t.Fatalf("open-ended to (inclusive): got %d", len(items))
	}
	items, _ = s.StreamItems(ctx, "ws", nil, nil, 0, 0, ViewIncluded)
	if len(items) != 3 {
		t.Fatalf("zero/zero must not narrow: got %d", len(items))
	}

	results, err := s.Search(ctx, "ws", "boiler", nil, nil, 150, 250)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 || results[0].Item.SourceID != "2" {
		t.Fatalf("search range: %+v", results)
	}
}
