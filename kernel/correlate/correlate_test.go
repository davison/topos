package correlate

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/davison/webspaces/kernel/config"
	"github.com/davison/webspaces/kernel/index"
	webspacesv1 "github.com/davison/webspaces/sdk/gen/webspaces/v1"
)

// fakeSource is a test double satisfying correlate.Source without launching
// a real plugin subprocess.
type fakeSource struct {
	name       string
	sourceType string
	matchFunc  func(keywords []string) (*webspacesv1.MatchResponse, error)
	calls      [][]string
}

func (f *fakeSource) Name() string       { return f.name }
func (f *fakeSource) SourceType() string { return f.sourceType }
func (f *fakeSource) Match(_ context.Context, keywords []string) (*webspacesv1.MatchResponse, error) {
	f.calls = append(f.calls, keywords)
	return f.matchFunc(keywords)
}

func newTestStore(t *testing.T) *index.Store {
	t.Helper()
	s, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func fixedNow() time.Time { return time.Unix(1780000000, 0) }

func TestSyncAll_PersistsMatchedItems(t *testing.T) {
	store := newTestStore(t)

	src := &fakeSource{
		name: "paperless", sourceType: "paperless",
		matchFunc: func(keywords []string) (*webspacesv1.MatchResponse, error) {
			return &webspacesv1.MatchResponse{Items: []*webspacesv1.Item{
				{SourceId: "1", Title: "Doc 1", Fidelity: webspacesv1.LinkFidelity_LINK_FIDELITY_EXACT, TimestampUnix: 100},
			}}, nil
		},
	}

	cfg := &config.Config{Webspaces: map[string]config.Webspace{
		"house-move": {Keywords: []string{"house-move", "House"}},
	}}

	engine := &Engine{Store: store, Sources: []Source{src}, Config: cfg, NowFunc: fixedNow}

	results, err := engine.SyncAll(context.Background())
	if err != nil {
		t.Fatalf("SyncAll: %v", err)
	}
	if len(results) != 1 || results[0].ItemCount != 1 {
		t.Fatalf("unexpected results: %+v", results)
	}

	items, err := store.StreamItems(context.Background(), "house-move")
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}
	if len(items) != 1 || items[0].ID != "paperless:1" {
		t.Fatalf("unexpected persisted items: %+v", items)
	}

	run, ok, err := store.LatestSyncRun(context.Background())
	if err != nil || !ok {
		t.Fatalf("LatestSyncRun: ok=%v err=%v", ok, err)
	}
	if run.Status != "ok" || run.ItemCount != 1 {
		t.Errorf("unexpected sync run: %+v", run)
	}
}

func TestSyncAll_KeywordOrderDoesNotAffectResult(t *testing.T) {
	store := newTestStore(t)

	matchFunc := func(keywords []string) (*webspacesv1.MatchResponse, error) {
		return &webspacesv1.MatchResponse{Items: []*webspacesv1.Item{
			{SourceId: "1", Title: "Doc 1", Fidelity: webspacesv1.LinkFidelity_LINK_FIDELITY_EXACT, TimestampUnix: 100},
			{SourceId: "2", Title: "Doc 2", Fidelity: webspacesv1.LinkFidelity_LINK_FIDELITY_EXACT, TimestampUnix: 200},
		}}, nil
	}

	src := &fakeSource{name: "paperless", sourceType: "paperless", matchFunc: matchFunc}
	cfg := &config.Config{Webspaces: map[string]config.Webspace{
		"ws": {Keywords: []string{"a", "b"}},
	}}
	engine := &Engine{Store: store, Sources: []Source{src}, Config: cfg, NowFunc: fixedNow}
	if _, err := engine.SyncAll(context.Background()); err != nil {
		t.Fatalf("SyncAll (order 1): %v", err)
	}
	first, err := store.StreamItems(context.Background(), "ws")
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}

	cfg2 := &config.Config{Webspaces: map[string]config.Webspace{
		"ws": {Keywords: []string{"b", "a"}},
	}}
	src2 := &fakeSource{name: "paperless", sourceType: "paperless", matchFunc: matchFunc}
	engine2 := &Engine{Store: store, Sources: []Source{src2}, Config: cfg2, NowFunc: fixedNow}
	if _, err := engine2.SyncAll(context.Background()); err != nil {
		t.Fatalf("SyncAll (order 2): %v", err)
	}
	second, err := store.StreamItems(context.Background(), "ws")
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}

	if len(first) != len(second) {
		t.Fatalf("item count differs between keyword orderings: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].ID != second[i].ID {
			t.Errorf("stream order differs at index %d: %s vs %s", i, first[i].ID, second[i].ID)
		}
	}
}

func TestSyncAll_MatchErrorRecordsFailedSyncRun(t *testing.T) {
	store := newTestStore(t)

	src := &fakeSource{
		name: "paperless", sourceType: "paperless",
		matchFunc: func([]string) (*webspacesv1.MatchResponse, error) {
			return nil, errors.New("connection refused")
		},
	}
	cfg := &config.Config{Webspaces: map[string]config.Webspace{
		"ws": {Keywords: []string{"a"}},
	}}
	engine := &Engine{Store: store, Sources: []Source{src}, Config: cfg, NowFunc: fixedNow}

	results, err := engine.SyncAll(context.Background())
	if err != nil {
		t.Fatalf("SyncAll should not return a top-level error on a source failure: %v", err)
	}
	if len(results) != 1 || results[0].Err == nil {
		t.Fatalf("expected a webspace-level error result, got: %+v", results)
	}

	run, ok, err := store.LatestSyncRun(context.Background())
	if err != nil || !ok {
		t.Fatalf("LatestSyncRun: ok=%v err=%v", ok, err)
	}
	if run.Status != "error" || run.Error == "" {
		t.Errorf("expected recorded error status, got: %+v", run)
	}
}
