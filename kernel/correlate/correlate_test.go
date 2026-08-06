package correlate

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/index"
	"github.com/davison/topos/kernel/item"
	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

// fakeSource is a test double satisfying correlate.Source without launching
// a real plugin subprocess.
type fakeSource struct {
	name       string
	sourceType string
	matchFunc  func(keywords []string) (*toposv1.MatchResponse, error)
	calls      [][]string
}

func (f *fakeSource) Name() string       { return f.name }
func (f *fakeSource) SourceType() string { return f.sourceType }
func (f *fakeSource) Match(_ context.Context, keywords []string) (*toposv1.MatchResponse, error) {
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

func TestSyncSource_PersistsMatchedItems(t *testing.T) {
	store := newTestStore(t)

	src := &fakeSource{
		name: "paperless", sourceType: "paperless",
		matchFunc: func(keywords []string) (*toposv1.MatchResponse, error) {
			return &toposv1.MatchResponse{Items: []*toposv1.Item{
				{SourceId: "1", Title: "Doc 1", Fidelity: toposv1.LinkFidelity_LINK_FIDELITY_EXACT, DeepLink: "http://paperless.lan/documents/1", TimestampUnix: 100},
			}}, nil
		},
	}

	cfg := &config.Config{Webspaces: map[string]config.Webspace{
		"house-move": {Keywords: []string{"house-move", "House"}},
	}}

	engine := &Engine{Store: store, Config: cfg}

	results, rejections := engine.SyncSource(context.Background(), src)
	if rejections != "" {
		t.Fatalf("expected no rejections, got %q", rejections)
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
}

func TestSyncSource_KeywordOrderDoesNotAffectResult(t *testing.T) {
	store := newTestStore(t)

	matchFunc := func(keywords []string) (*toposv1.MatchResponse, error) {
		return &toposv1.MatchResponse{Items: []*toposv1.Item{
			{SourceId: "1", Title: "Doc 1", Fidelity: toposv1.LinkFidelity_LINK_FIDELITY_EXACT, DeepLink: "http://paperless.lan/documents/1", TimestampUnix: 100},
			{SourceId: "2", Title: "Doc 2", Fidelity: toposv1.LinkFidelity_LINK_FIDELITY_EXACT, DeepLink: "http://paperless.lan/documents/2", TimestampUnix: 200},
		}}, nil
	}

	src := &fakeSource{name: "paperless", sourceType: "paperless", matchFunc: matchFunc}
	cfg := &config.Config{Webspaces: map[string]config.Webspace{
		"ws": {Keywords: []string{"a", "b"}},
	}}
	engine := &Engine{Store: store, Config: cfg}
	if _, rejections := engine.SyncSource(context.Background(), src); rejections != "" {
		t.Fatalf("SyncSource (order 1): unexpected rejections %q", rejections)
	}
	first, err := store.StreamItems(context.Background(), "ws")
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}

	cfg2 := &config.Config{Webspaces: map[string]config.Webspace{
		"ws": {Keywords: []string{"b", "a"}},
	}}
	src2 := &fakeSource{name: "paperless", sourceType: "paperless", matchFunc: matchFunc}
	engine2 := &Engine{Store: store, Config: cfg2}
	if _, rejections := engine2.SyncSource(context.Background(), src2); rejections != "" {
		t.Fatalf("SyncSource (order 2): unexpected rejections %q", rejections)
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

func TestSyncSource_MatchErrorReturnsWebspaceResultErr(t *testing.T) {
	store := newTestStore(t)

	src := &fakeSource{
		name: "paperless", sourceType: "paperless",
		matchFunc: func([]string) (*toposv1.MatchResponse, error) {
			return nil, errors.New("connection refused")
		},
	}
	cfg := &config.Config{Webspaces: map[string]config.Webspace{
		"ws": {Keywords: []string{"a"}},
	}}
	engine := &Engine{Store: store, Config: cfg}

	results, _ := engine.SyncSource(context.Background(), src)
	if len(results) != 1 || results[0].Err == nil {
		t.Fatalf("expected a webspace-level error result, got: %+v", results)
	}
}

// TestSyncSource_RejectsUnspecifiedFidelityAndEmptyDeepLink verifies
// PLUG-03: an item with an unspecified fidelity, or an empty deep link, is
// skipped at the correlation boundary and never reaches the index, while
// other valid items from the same source still persist normally, and the
// rejection is named (plugin + source id) in the returned rejections
// string — the caller (kernel/syncer.Coordinator) is what records this on
// the sync_runs row.
func TestSyncSource_RejectsUnspecifiedFidelityAndEmptyDeepLink(t *testing.T) {
	store := newTestStore(t)

	src := &fakeSource{
		name: "paperless", sourceType: "paperless",
		matchFunc: func([]string) (*toposv1.MatchResponse, error) {
			return &toposv1.MatchResponse{Items: []*toposv1.Item{
				{SourceId: "good", Title: "Valid item", Fidelity: toposv1.LinkFidelity_LINK_FIDELITY_EXACT, DeepLink: "http://paperless.lan/documents/good", TimestampUnix: 100},
				{SourceId: "no-fidelity", Title: "Missing fidelity", Fidelity: toposv1.LinkFidelity_LINK_FIDELITY_UNSPECIFIED, DeepLink: "http://paperless.lan/documents/no-fidelity", TimestampUnix: 200},
				{SourceId: "no-link", Title: "Missing deep link", Fidelity: toposv1.LinkFidelity_LINK_FIDELITY_EXACT, DeepLink: "", TimestampUnix: 300},
			}}, nil
		},
	}
	cfg := &config.Config{Webspaces: map[string]config.Webspace{
		"ws": {Keywords: []string{"a"}},
	}}
	engine := &Engine{Store: store, Config: cfg}

	results, rejections := engine.SyncSource(context.Background(), src)
	if len(results) != 1 || results[0].Err != nil || results[0].ItemCount != 1 {
		t.Fatalf("expected the sync to succeed with exactly 1 persisted item, got: %+v", results)
	}

	items, err := store.StreamItems(context.Background(), "ws")
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}
	if len(items) != 1 || items[0].ID != "paperless:good" {
		t.Fatalf("expected only the valid item to be persisted, got: %+v", items)
	}

	if rejections == "" {
		t.Error("expected the rejections string to record the rejected items")
	}
	if !strings.Contains(rejections, "paperless") || !strings.Contains(rejections, "no-fidelity") || !strings.Contains(rejections, "no-link") {
		t.Errorf("expected the rejections string to name the plugin and both rejected source ids, got: %q", rejections)
	}
}

// TestSyncSource_PartialSourceFailure_HealthySourceItemsPersist is the
// load-bearing regression test for 02-01-PLAN.md's objective, adapted for
// 02-02-PLAN.md's per-source-call shape (kernel/syncer.Coordinator now
// calls SyncSource once per source independently, never via a joint
// whole-run loop): with two sources, each synced via its own SyncSource
// call, one of which fails Match, the healthy source's freshly matched
// items must persist for every webspace, and the failing source's
// previously persisted rows must be left completely unchanged — never
// rolled back, never discarded, just because a sibling source was
// unreachable this cycle (02-RESEARCH.md "Critical Architecture Finding").
func TestSyncSource_PartialSourceFailure_HealthySourceItemsPersist(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	healthy := &fakeSource{
		name: "paperless", sourceType: "paperless",
		matchFunc: func([]string) (*toposv1.MatchResponse, error) {
			return &toposv1.MatchResponse{Items: []*toposv1.Item{
				{SourceId: "1", Title: "Doc 1", Fidelity: toposv1.LinkFidelity_LINK_FIDELITY_EXACT, DeepLink: "http://paperless.lan/documents/1", TimestampUnix: 100},
			}}, nil
		},
	}
	flaky := &fakeSource{
		name: "silverbullet", sourceType: "silverbullet",
		matchFunc: func([]string) (*toposv1.MatchResponse, error) {
			return &toposv1.MatchResponse{Items: []*toposv1.Item{
				{SourceId: "notes/a", Title: "Note A", Fidelity: toposv1.LinkFidelity_LINK_FIDELITY_EXACT, DeepLink: "http://sb.lan/notes/a", TimestampUnix: 50},
			}}, nil
		},
	}
	cfg := &config.Config{Webspaces: map[string]config.Webspace{
		"house-move": {Keywords: []string{"house"}},
	}}

	// First cycle: both sources healthy, each synced independently — seed
	// a baseline so the flaky source has previously persisted rows to
	// prove untouched in the second cycle.
	engine := &Engine{Store: store, Config: cfg}
	engine.SyncSource(ctx, healthy)
	engine.SyncSource(ctx, flaky)
	baseline, err := store.StreamItems(ctx, "house-move")
	if err != nil {
		t.Fatalf("StreamItems (baseline): %v", err)
	}
	if len(baseline) != 2 {
		t.Fatalf("expected 2 baseline items, got %d: %+v", len(baseline), baseline)
	}

	// Second cycle: silverbullet now fails; paperless returns a NEW item
	// (proving it's not just "the old rows happened to still be there").
	healthy2 := &fakeSource{
		name: "paperless", sourceType: "paperless",
		matchFunc: func([]string) (*toposv1.MatchResponse, error) {
			return &toposv1.MatchResponse{Items: []*toposv1.Item{
				{SourceId: "2", Title: "Doc 2", Fidelity: toposv1.LinkFidelity_LINK_FIDELITY_EXACT, DeepLink: "http://paperless.lan/documents/2", TimestampUnix: 200},
			}}, nil
		},
	}
	failingFlaky := &fakeSource{
		name: "silverbullet", sourceType: "silverbullet",
		matchFunc: func([]string) (*toposv1.MatchResponse, error) {
			return nil, errors.New("connection refused")
		},
	}

	paperlessResults, _ := engine.SyncSource(ctx, healthy2)
	silverbulletResults, _ := engine.SyncSource(ctx, failingFlaky)

	var sawPaperlessOK, sawSilverbulletErr bool
	for _, r := range paperlessResults {
		if r.Source == "paperless" && r.Webspace == "house-move" {
			sawPaperlessOK = true
			if r.Err != nil {
				t.Errorf("expected the healthy source's result to have no error, got: %v", r.Err)
			}
		}
	}
	for _, r := range silverbulletResults {
		if r.Source == "silverbullet" && r.Webspace == "house-move" {
			sawSilverbulletErr = true
			if r.Err == nil {
				t.Error("expected the failing source's result to carry an error")
			}
		}
	}
	if !sawPaperlessOK || !sawSilverbulletErr {
		t.Fatalf("expected one result per (webspace, source), got paperless=%+v silverbullet=%+v", paperlessResults, silverbulletResults)
	}

	items, err := store.StreamItems(ctx, "house-move")
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}
	ids := map[string]bool{}
	for _, it := range items {
		ids[it.ID] = true
	}

	// The healthy source's NEW item replaced its old one (source-scoped
	// replace, not additive).
	if ids["paperless:1"] {
		t.Error("expected paperless's OLD item to be gone after a fresh successful sync replaced it")
	}
	if !ids["paperless:2"] {
		t.Error("expected paperless's NEW item to be present")
	}
	// The failing source's previously persisted row must be untouched —
	// this is the actual regression this test guards against.
	if !ids["silverbullet:notes/a"] {
		t.Errorf("expected silverbullet's previously persisted item to survive a sync cycle where silverbullet's Match failed; got items: %v", idsOf(items))
	}
	if len(items) != 2 {
		t.Errorf("expected exactly 2 items (1 fresh paperless + 1 untouched silverbullet), got %d: %+v", len(items), items)
	}
}

func idsOf(items []item.Item) []string {
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	return ids
}

// TestSyncSource_SourceMajorPersistsIndependentlyPerWebspace proves
// SyncSource is source-major: calling it once for one source persists
// that source's contribution to every configured webspace, and does so
// via the source-scoped store method, never touching another webspace's
// or another source's rows.
func TestSyncSource_SourceMajorPersistsIndependentlyPerWebspace(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	src := &fakeSource{
		name: "silverbullet", sourceType: "silverbullet",
		matchFunc: func(keywords []string) (*toposv1.MatchResponse, error) {
			return &toposv1.MatchResponse{Items: []*toposv1.Item{
				{SourceId: "notes/a", Title: "Note A", Fidelity: toposv1.LinkFidelity_LINK_FIDELITY_EXACT, DeepLink: "http://sb.lan/notes/a", TimestampUnix: 100},
			}}, nil
		},
	}
	cfg := &config.Config{Webspaces: map[string]config.Webspace{
		"house-move": {Keywords: []string{"house"}},
		"garden":     {Keywords: []string{"garden"}},
	}}
	engine := &Engine{Store: store, Config: cfg}

	results, rejections := engine.SyncSource(ctx, src)
	if rejections != "" {
		t.Fatalf("unexpected rejections: %q", rejections)
	}
	if len(results) != 2 {
		t.Fatalf("expected one result per configured webspace, got %d: %+v", len(results), results)
	}
	for _, r := range results {
		if r.Source != "silverbullet" {
			t.Errorf("expected every result's Source to be 'silverbullet', got %q", r.Source)
		}
		if r.Err != nil {
			t.Errorf("unexpected error in result %+v", r)
		}
		if r.ItemCount != 1 {
			t.Errorf("expected 1 item persisted for webspace %q, got %d", r.Webspace, r.ItemCount)
		}
	}

	for _, ws := range []string{"house-move", "garden"} {
		items, err := store.StreamItems(ctx, ws)
		if err != nil {
			t.Fatalf("StreamItems(%s): %v", ws, err)
		}
		if len(items) != 1 || items[0].ID != "silverbullet:notes/a" {
			t.Fatalf("expected silverbullet's item persisted independently to webspace %q, got: %+v", ws, items)
		}
	}
}
