package index

import (
	"context"
	"testing"

	"github.com/davison/topos/kernel/item"
)

// The per-instance filter map at the index (M2-R3, #55): terms narrow one
// instance's rows and leave every other instance's untouched, AND-ed on
// top of the global filter and the live query, in both the stream and the
// search reads.
func TestStreamAndSearch_FilterBySourceNarrowsOneInstanceOnly(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	mk := func(source, sourceID, title string, ts int64) item.Item {
		it := sampleItemForSource(source, sourceID, ts)
		it.Source = source
		it.Title = title
		return it
	}
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "mail-01", []item.Item{
		mk("mail-01", "1", "boiler quote arrived", 300),
		mk("mail-01", "2", "boiler survey booked", 200),
	}); err != nil {
		t.Fatalf("seed mail-01: %v", err)
	}
	if err := s.ReplaceWebspaceSourceItems(ctx, "ws", "docs-01", []item.Item{
		mk("docs-01", "3", "boiler manual scanned", 100),
	}); err != nil {
		t.Fatalf("seed docs-01: %v", err)
	}

	bySource := map[string][]string{"mail-01": {"quote"}}

	// Stream: mail-01 narrows to its quote item; docs-01 is untouched.
	items, err := s.StreamItems(ctx, "ws", nil, bySource, ViewIncluded)
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}
	if len(items) != 2 || items[0].Title != "boiler quote arrived" || items[1].Title != "boiler manual scanned" {
		t.Fatalf("stream narrowing: %+v", items)
	}

	// A global filter still ANDs with the per-instance terms.
	items, err = s.StreamItems(ctx, "ws", []string{"manual"}, bySource, ViewIncluded)
	if err != nil {
		t.Fatalf("StreamItems global+per: %v", err)
	}
	if len(items) != 1 || items[0].Source != "docs-01" {
		t.Fatalf("global AND per-instance: %+v", items)
	}

	// Search: the live query refines within both.
	results, err := s.Search(ctx, "ws", "boiler", nil, bySource)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("search narrowing: %+v", results)
	}
	for _, r := range results {
		if r.Item.Source == "mail-01" && r.Item.Title != "boiler quote arrived" {
			t.Fatalf("mail-01 must only surface its quote item: %+v", r.Item)
		}
	}

	// An FTS-hostile per-instance term degrades to a no-op for that
	// instance, exactly as BuildMatchQuery degrades the global filter.
	items, err = s.StreamItems(ctx, "ws", nil, map[string][]string{"mail-01": {`""`}}, ViewIncluded)
	if err != nil {
		t.Fatalf("StreamItems hostile term: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("hostile term must degrade to no narrowing: %d items", len(items))
	}
}
