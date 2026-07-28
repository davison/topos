package correlate

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/davison/webspaces/kernel/config"
	"github.com/davison/webspaces/kernel/index"
	"github.com/davison/webspaces/kernel/item"
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
				{SourceId: "1", Title: "Doc 1", Fidelity: webspacesv1.LinkFidelity_LINK_FIDELITY_EXACT, DeepLink: "http://paperless.lan/documents/1", TimestampUnix: 100},
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
			{SourceId: "1", Title: "Doc 1", Fidelity: webspacesv1.LinkFidelity_LINK_FIDELITY_EXACT, DeepLink: "http://paperless.lan/documents/1", TimestampUnix: 100},
			{SourceId: "2", Title: "Doc 2", Fidelity: webspacesv1.LinkFidelity_LINK_FIDELITY_EXACT, DeepLink: "http://paperless.lan/documents/2", TimestampUnix: 200},
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

// TestSyncAll_RejectsUnspecifiedFidelityAndEmptyDeepLink verifies PLUG-03:
// an item with an unspecified fidelity, or an empty deep link, is skipped
// at the correlation boundary and never reaches the index, while other
// valid items from the same source still persist normally, and the
// rejection is named (plugin + source id) in the recorded sync run.
func TestSyncAll_RejectsUnspecifiedFidelityAndEmptyDeepLink(t *testing.T) {
	store := newTestStore(t)

	src := &fakeSource{
		name: "paperless", sourceType: "paperless",
		matchFunc: func([]string) (*webspacesv1.MatchResponse, error) {
			return &webspacesv1.MatchResponse{Items: []*webspacesv1.Item{
				{SourceId: "good", Title: "Valid item", Fidelity: webspacesv1.LinkFidelity_LINK_FIDELITY_EXACT, DeepLink: "http://paperless.lan/documents/good", TimestampUnix: 100},
				{SourceId: "no-fidelity", Title: "Missing fidelity", Fidelity: webspacesv1.LinkFidelity_LINK_FIDELITY_UNSPECIFIED, DeepLink: "http://paperless.lan/documents/no-fidelity", TimestampUnix: 200},
				{SourceId: "no-link", Title: "Missing deep link", Fidelity: webspacesv1.LinkFidelity_LINK_FIDELITY_EXACT, DeepLink: "", TimestampUnix: 300},
			}}, nil
		},
	}
	cfg := &config.Config{Webspaces: map[string]config.Webspace{
		"ws": {Keywords: []string{"a"}},
	}}
	engine := &Engine{Store: store, Sources: []Source{src}, Config: cfg, NowFunc: fixedNow}

	results, err := engine.SyncAll(context.Background())
	if err != nil {
		t.Fatalf("SyncAll: %v", err)
	}
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

	run, ok, err := store.LatestSyncRun(context.Background())
	if err != nil || !ok {
		t.Fatalf("LatestSyncRun: ok=%v err=%v", ok, err)
	}
	if run.Status != "ok" {
		t.Errorf("expected sync run status ok (rejections are per-item, not fatal), got %q", run.Status)
	}
	if run.Error == "" {
		t.Error("expected the sync run to record the rejected items")
	}
	if !strings.Contains(run.Error, "paperless") || !strings.Contains(run.Error, "no-fidelity") || !strings.Contains(run.Error, "no-link") {
		t.Errorf("expected the sync run error to name the plugin and both rejected source ids, got: %q", run.Error)
	}
}

// TestSyncAll_PartialSourceFailure_HealthySourceItemsPersist is the
// load-bearing regression test for 02-01-PLAN.md's objective: with two
// configured sources, one of which fails Match, the healthy source's
// freshly matched items must persist for every webspace, and the failing
// source's previously persisted rows must be left completely unchanged —
// never rolled back, never discarded, just because a sibling source was
// unreachable this cycle (02-RESEARCH.md "Critical Architecture Finding").
func TestSyncAll_PartialSourceFailure_HealthySourceItemsPersist(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	healthy := &fakeSource{
		name: "paperless", sourceType: "paperless",
		matchFunc: func([]string) (*webspacesv1.MatchResponse, error) {
			return &webspacesv1.MatchResponse{Items: []*webspacesv1.Item{
				{SourceId: "1", Title: "Doc 1", Fidelity: webspacesv1.LinkFidelity_LINK_FIDELITY_EXACT, DeepLink: "http://paperless.lan/documents/1", TimestampUnix: 100},
			}}, nil
		},
	}
	flaky := &fakeSource{
		name: "silverbullet", sourceType: "silverbullet",
		matchFunc: func([]string) (*webspacesv1.MatchResponse, error) {
			return &webspacesv1.MatchResponse{Items: []*webspacesv1.Item{
				{SourceId: "notes/a", Title: "Note A", Fidelity: webspacesv1.LinkFidelity_LINK_FIDELITY_EXACT, DeepLink: "http://sb.lan/notes/a", TimestampUnix: 50},
			}}, nil
		},
	}
	cfg := &config.Config{Webspaces: map[string]config.Webspace{
		"house-move": {Keywords: []string{"house"}},
	}}

	// First cycle: both sources healthy — seed a baseline so the flaky
	// source has previously persisted rows to prove untouched in the
	// second cycle.
	engine := &Engine{Store: store, Sources: []Source{healthy, flaky}, Config: cfg, NowFunc: fixedNow}
	if _, err := engine.SyncAll(ctx); err != nil {
		t.Fatalf("baseline SyncAll: %v", err)
	}
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
		matchFunc: func([]string) (*webspacesv1.MatchResponse, error) {
			return &webspacesv1.MatchResponse{Items: []*webspacesv1.Item{
				{SourceId: "2", Title: "Doc 2", Fidelity: webspacesv1.LinkFidelity_LINK_FIDELITY_EXACT, DeepLink: "http://paperless.lan/documents/2", TimestampUnix: 200},
			}}, nil
		},
	}
	failingFlaky := &fakeSource{
		name: "silverbullet", sourceType: "silverbullet",
		matchFunc: func([]string) (*webspacesv1.MatchResponse, error) {
			return nil, errors.New("connection refused")
		},
	}
	engine2 := &Engine{Store: store, Sources: []Source{healthy2, failingFlaky}, Config: cfg, NowFunc: fixedNow}

	results, err := engine2.SyncAll(ctx)
	if err != nil {
		t.Fatalf("SyncAll should not return a top-level error on a source failure: %v", err)
	}

	var sawPaperlessOK, sawSilverbulletErr bool
	for _, r := range results {
		if r.SourceType == "paperless" && r.Webspace == "house-move" {
			sawPaperlessOK = true
			if r.Err != nil {
				t.Errorf("expected the healthy source's result to have no error, got: %v", r.Err)
			}
		}
		if r.SourceType == "silverbullet" && r.Webspace == "house-move" {
			sawSilverbulletErr = true
			if r.Err == nil {
				t.Error("expected the failing source's result to carry an error")
			}
		}
	}
	if !sawPaperlessOK || !sawSilverbulletErr {
		t.Fatalf("expected one result per (webspace, source), got: %+v", results)
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
// SyncSource itself (not just SyncAll) is source-major: calling it once
// for one source persists that source's contribution to every configured
// webspace, and does so via the source-scoped store method, never
// touching another webspace's or another source's rows.
func TestSyncSource_SourceMajorPersistsIndependentlyPerWebspace(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	src := &fakeSource{
		name: "silverbullet", sourceType: "silverbullet",
		matchFunc: func(keywords []string) (*webspacesv1.MatchResponse, error) {
			return &webspacesv1.MatchResponse{Items: []*webspacesv1.Item{
				{SourceId: "notes/a", Title: "Note A", Fidelity: webspacesv1.LinkFidelity_LINK_FIDELITY_EXACT, DeepLink: "http://sb.lan/notes/a", TimestampUnix: 100},
			}}, nil
		},
	}
	cfg := &config.Config{Webspaces: map[string]config.Webspace{
		"house-move": {Keywords: []string{"house"}},
		"garden":     {Keywords: []string{"garden"}},
	}}
	engine := &Engine{Store: store, Sources: []Source{src}, Config: cfg, NowFunc: fixedNow}

	results := engine.SyncSource(ctx, src)
	if len(results) != 2 {
		t.Fatalf("expected one result per configured webspace, got %d: %+v", len(results), results)
	}
	for _, r := range results {
		if r.SourceType != "silverbullet" {
			t.Errorf("expected every result's SourceType to be 'silverbullet', got %q", r.SourceType)
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
