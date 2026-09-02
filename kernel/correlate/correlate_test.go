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
// a real plugin subprocess. Its matchFunc receives the "keywords" field of
// the resolved match_fields map (fakeSource declares a single-field
// vocabulary, "keywords", mirroring the single-field vocabularies the
// first-party plugins — now in topos-plugins — declare, and letting
// existing tests keep passing a flat
// []string without change).
type fakeSource struct {
	name       string
	sourceType string
	vocabulary []string // defaults to []string{"keywords"} when unset (see MatchVocabulary)
	matchFunc  func(keywords []string) (*toposv1.MatchResponse, error)
	calls      [][]string
	// receivedFields records every full match_fields map this source's
	// Match was called with — used by tests asserting explicit-block or
	// multi-field fallback resolution, where a flat []string of "keywords"
	// alone isn't enough to inspect the call.
	receivedFields []map[string][]string
}

func (f *fakeSource) Name() string       { return f.name }
func (f *fakeSource) SourceType() string { return f.sourceType }
func (f *fakeSource) MatchVocabulary() []string {
	if f.vocabulary != nil {
		return f.vocabulary
	}
	return []string{"keywords"}
}
func (f *fakeSource) Match(_ context.Context, fields map[string][]string) (*toposv1.MatchResponse, error) {
	f.receivedFields = append(f.receivedFields, fields)
	keywords := fields["keywords"]
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

	items, err := store.StreamItems(context.Background(), "house-move", nil, nil, 0, 0, index.ViewIncluded)
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
	first, err := store.StreamItems(context.Background(), "ws", nil, nil, 0, 0, index.ViewIncluded)
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
	second, err := store.StreamItems(context.Background(), "ws", nil, nil, 0, 0, index.ViewIncluded)
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

// TestSyncSource_MatchFailureNeverPrunesAMark proves the STRUCTURAL half
// of D-10 (13-02-PLAN.md Task 2): a Match failure for a (webspace, source)
// pair never reaches Store.ReplaceWebspaceSourceItems at all — SyncSource's
// error branch appends a WebspaceResult and continues, skipping the
// persistence call entirely for that pair — so the orphan prune sweep is
// structurally unreachable on a failed sync, not merely skipped by a
// runtime check that could later be bypassed.
func TestSyncSource_MatchFailureNeverPrunesAMark(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Seed and exclude an item via a prior HEALTHY sync of this same
	// (webspace, source) pair.
	it := item.Item{
		ID: item.ID("paperless", "1"), Source: "paperless", SourceType: "paperless", SourceID: "1",
		Title: "Doc 1", Fidelity: item.FidelityExact, DeepLink: "http://paperless.lan/documents/1", TimestampUnix: 100,
	}
	if err := store.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{it}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := store.SetItemMarks(ctx, "ws", index.MarkKindExcluded, []string{it.ID}); err != nil {
		t.Fatalf("SetItemMarks: %v", err)
	}

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

	results, _ := engine.SyncSource(ctx, src)
	if len(results) != 1 || results[0].Err == nil {
		t.Fatalf("expected a webspace-level error result for the failed Match, got: %+v", results)
	}

	count, err := store.CountItemMarks(ctx, "ws", index.MarkKindExcluded)
	if err != nil {
		t.Fatalf("CountItemMarks: %v", err)
	}
	if count != 1 {
		t.Fatalf("a failed sync ate a mark: expected the mark to survive a Match failure untouched, got count=%d", count)
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

	items, err := store.StreamItems(context.Background(), "ws", nil, nil, 0, 0, index.ViewIncluded)
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
	baseline, err := store.StreamItems(ctx, "house-move", nil, nil, 0, 0, index.ViewIncluded)
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

	items, err := store.StreamItems(ctx, "house-move", nil, nil, 0, 0, index.ViewIncluded)
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
		items, err := store.StreamItems(ctx, ws, nil, nil, 0, 0, index.ViewIncluded)
		if err != nil {
			t.Fatalf("StreamItems(%s): %v", ws, err)
		}
		if len(items) != 1 || items[0].ID != "silverbullet:notes/a" {
			t.Fatalf("expected silverbullet's item persisted independently to webspace %q, got: %+v", ws, items)
		}
	}
}

// TestMatchFieldsFor_ExplicitBlockReplacesFallback proves D-02: an instance
// with an explicit ws.Match block receives that block verbatim, never a
// union with ws.Keywords.
func TestMatchFieldsFor_ExplicitBlockReplacesFallback(t *testing.T) {
	src := &fakeSource{name: "home-email", vocabulary: []string{"folders"}}
	ws := config.Webspace{
		Keywords: []string{"house"},
		Match: map[string]config.MatchBlock{
			"home-email": {"folders": {"Home"}},
		},
	}

	fields, participates, _ := matchFieldsFor(ws, src)
	if !participates {
		t.Fatal("expected instance with an explicit block to participate")
	}
	if len(fields) != 1 {
		t.Fatalf("expected exactly the explicit block's one field, got %+v", fields)
	}
	if got := fields["folders"]; len(got) != 1 || got[0] != "Home" {
		t.Errorf("expected fields[\"folders\"] == [\"Home\"] (the explicit block, not the keywords fallback), got %+v", got)
	}
}

// TestMatchFieldsFor_FallbackFansAcrossTwoFieldVocabulary proves D-01: an
// instance with no explicit block receives ws.Keywords fanned into every
// field of its declared (here two-field) vocabulary.
func TestMatchFieldsFor_FallbackFansAcrossTwoFieldVocabulary(t *testing.T) {
	src := &fakeSource{name: "wiki", vocabulary: []string{"tags", "pages"}}
	ws := config.Webspace{Keywords: []string{"house"}}

	fields, participates, _ := matchFieldsFor(ws, src)
	if !participates {
		t.Fatal("expected instance relying on the fallback to participate")
	}
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields (one per declared vocabulary entry), got %+v", fields)
	}
	for _, field := range []string{"tags", "pages"} {
		got := fields[field]
		if len(got) != 1 || got[0] != "house" {
			t.Errorf("expected fields[%q] == [\"house\"] (the fanned-out fallback), got %+v", field, got)
		}
	}
}

// TestMatchFieldsFor_DeallowlistedInstanceDoesNotParticipate proves D-03: a
// webspace's non-empty sources allowlist excludes any instance not named in
// it, regardless of whether that instance has an explicit block or would
// otherwise use the fallback.
func TestMatchFieldsFor_DeallowlistedInstanceDoesNotParticipate(t *testing.T) {
	src := &fakeSource{name: "personal-signal", vocabulary: []string{"conversations"}}
	ws := config.Webspace{Keywords: []string{"house"}, Sources: []string{"work-email"}}

	fields, participates, _ := matchFieldsFor(ws, src)
	if participates {
		t.Fatalf("expected de-allowlisted instance to not participate, got fields %+v", fields)
	}
	if fields != nil {
		t.Errorf("expected a nil fields map for a non-participating instance, got %+v", fields)
	}
}

// TestMatchFieldsFor_NoBlockAndNoKeywordsDoesNotParticipate proves D-20's
// safety rule (07-11-PLAN.md): an instance with neither an explicit match
// block nor a non-empty keywords fallback does not participate at all —
// Match is never called with a field map whose every value list would
// otherwise be empty. Before D-20 this state was unreachable
// (validateFallbackCoverage guaranteed it away); against pre-D-20 code
// matchFieldsFor returns participates == true with fields fanned from an
// empty Keywords slice (every value list empty) — exactly the shape a
// plugin could read as "no constraint" and answer with its entire corpus.
func TestMatchFieldsFor_NoBlockAndNoKeywordsDoesNotParticipate(t *testing.T) {
	src := &fakeSource{name: "home-email", vocabulary: []string{"folders"}}
	ws := config.Webspace{}

	fields, participates, _ := matchFieldsFor(ws, src)
	if participates {
		t.Fatalf("expected an instance with no block and no keywords fallback to not participate, got fields %+v", fields)
	}
	if fields != nil {
		t.Errorf("expected a nil fields map for a non-participating instance, got %+v", fields)
	}
}

// TestParticipatesIn_ResolutionShapes is the table 07-16-PLAN.md Task 1
// requires: every shape named in the plan's <behavior> block, each named by
// the user-visible consequence it stands for ("which chips a webspace would
// show and which items would land in it") — plus the agreement assertion
// below tying ParticipatesIn to matchFieldsFor's own second return value, so
// the two definitions can never diverge without a test failing.
func TestParticipatesIn_ResolutionShapes(t *testing.T) {
	cases := []struct {
		name     string
		ws       config.Webspace
		instance string
		want     bool
	}{
		{
			name: "a non-empty allowlist naming the instance, with an explicit match block: the chip shows and its items land in the webspace",
			ws: config.Webspace{
				Sources: []string{"home-email"},
				Match:   map[string]config.MatchBlock{"home-email": {"folders": {"Home"}}},
			},
			instance: "home-email",
			want:     true,
		},
		{
			name: "a non-empty allowlist NOT naming the instance: no chip, no items, whatever the keywords or match map say",
			ws: config.Webspace{
				Sources:  []string{"work-email"},
				Keywords: []string{"house"},
				Match:    map[string]config.MatchBlock{"home-email": {"folders": {"Home"}}},
			},
			instance: "home-email",
			want:     false,
		},
		{
			name:     "an empty allowlist with a non-empty keywords fallback: the chip shows (Phase 5 D-03's default)",
			ws:       config.Webspace{Keywords: []string{"house"}},
			instance: "home-email",
			want:     true,
		},
		{
			name: "an empty allowlist, no keywords, and an explicit match block for this instance: the chip shows",
			ws: config.Webspace{
				Match: map[string]config.MatchBlock{"home-email": {"folders": {"Home"}}},
			},
			instance: "home-email",
			want:     true,
		},
		{
			name:     "an empty allowlist, no keywords, and no block for this instance: no chip — 07-11's D-20 rule",
			ws:       config.Webspace{},
			instance: "home-email",
			want:     false,
		},
		{
			name:     "a D-20 empty shell: no instance participates",
			ws:       config.Webspace{},
			instance: "any-instance",
			want:     false,
		},
		{
			name:     "a webspace whose collections are nil rather than empty: no panic, and the no-block-no-keywords answer is unchanged",
			ws:       config.Webspace{Keywords: nil, Sources: nil, Match: nil},
			instance: "home-email",
			want:     false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ParticipatesIn(tc.ws, tc.instance); got != tc.want {
				t.Errorf("ParticipatesIn(%+v, %q) = %v, want %v", tc.ws, tc.instance, got, tc.want)
			}

			// Agreement: the predicate's answer must equal matchFieldsFor's own
			// second return value for the identical (webspace, instance) pair,
			// against a source whose vocabulary is non-empty — the two
			// definitions can never diverge without this failing.
			src := &fakeSource{name: tc.instance, vocabulary: []string{"folders"}}
			_, participates, _ := matchFieldsFor(tc.ws, src)
			if participates != tc.want {
				t.Errorf("matchFieldsFor(%+v, %q) participates = %v, want %v (must agree with ParticipatesIn)", tc.ws, tc.instance, participates, tc.want)
			}
		})
	}
}

// TestSyncSource_DeallowlistedInstanceRowsCleared proves the ROADMAP
// success-criterion-3 guarantee end to end: when a webspace's sources
// allowlist stops including a previously-participating instance, that
// instance's Match is never called and its previously persisted rows for
// that webspace are cleared at the next sync, leaving no orphaned rows.
func TestSyncSource_DeallowlistedInstanceRowsCleared(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	seed := &fakeSource{
		name: "personal-signal", sourceType: "signal",
		matchFunc: func([]string) (*toposv1.MatchResponse, error) {
			return &toposv1.MatchResponse{Items: []*toposv1.Item{
				{SourceId: "1", Title: "Chat", Fidelity: toposv1.LinkFidelity_LINK_FIDELITY_EXACT, DeepLink: "signal://x", TimestampUnix: 100},
			}}, nil
		},
	}
	cfg := &config.Config{Webspaces: map[string]config.Webspace{
		"work": {Keywords: []string{"work"}},
	}}
	engine := &Engine{Store: store, Config: cfg}

	// First cycle: the instance participates (no allowlist yet) and its
	// item persists.
	if _, rejections := engine.SyncSource(ctx, seed); rejections != "" {
		t.Fatalf("seed SyncSource: unexpected rejections %q", rejections)
	}
	baseline, err := store.StreamItems(ctx, "work", nil, nil, 0, 0, index.ViewIncluded)
	if err != nil {
		t.Fatalf("StreamItems (baseline): %v", err)
	}
	if len(baseline) != 1 {
		t.Fatalf("expected 1 baseline item, got %d: %+v", len(baseline), baseline)
	}

	// Second cycle: the webspace's sources allowlist now excludes this
	// instance. Its Match must never be called, and its previously
	// persisted rows must be cleared.
	cfg2 := &config.Config{Webspaces: map[string]config.Webspace{
		"work": {Keywords: []string{"work"}, Sources: []string{"work-email"}},
	}}
	engine2 := &Engine{Store: store, Config: cfg2}
	guarded := &fakeSource{
		name: "personal-signal", sourceType: "signal",
		matchFunc: func([]string) (*toposv1.MatchResponse, error) {
			t.Fatal("Match must not be called for an instance excluded by the sources allowlist")
			return nil, nil
		},
	}

	results, rejections := engine2.SyncSource(ctx, guarded)
	if rejections != "" {
		t.Fatalf("unexpected rejections: %q", rejections)
	}
	if len(results) != 1 || results[0].Err != nil || results[0].ItemCount != 0 {
		t.Fatalf("expected a zero-count, error-free result for the de-allowlisted instance, got: %+v", results)
	}

	items, err := store.StreamItems(ctx, "work", nil, nil, 0, 0, index.ViewIncluded)
	if err != nil {
		t.Fatalf("StreamItems (after de-allowlisting): %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected the de-allowlisted instance's rows to be cleared, got: %+v", items)
	}
}

// --- 12-09-PLAN.md Task 2: the zero-match notice's boundaries ---

// TestSyncSource_ExplicitMatchBlockThatMatchedNothingRecordsANotice proves
// the G-12-1/G-12-3 gap closure fires for an explicit ws.Match block that
// matched zero items across an otherwise-healthy sync, and that the rule
// is plugin-agnostic: a filesystem-shaped instance (a "path" field, the
// reported failure's own shape) and a differently-shaped instance (a
// "labels" field, mirroring the debug session's second dead instance
// "test-ext") both get the same treatment.
func TestSyncSource_ExplicitMatchBlockThatMatchedNothingRecordsANotice(t *testing.T) {
	t.Run("filesystem-shaped: a path field matching a doublestar pattern", func(t *testing.T) {
		store := newTestStore(t)
		src := &fakeSource{
			name: "files", sourceType: "filesystem", vocabulary: []string{"path"},
			matchFunc: func([]string) (*toposv1.MatchResponse, error) {
				return &toposv1.MatchResponse{}, nil
			},
		}
		cfg := &config.Config{Webspaces: map[string]config.Webspace{
			"test": {Match: map[string]config.MatchBlock{
				"files": {"path": {"/nonexistent/**"}},
			}},
		}}
		engine := &Engine{Store: store, Config: cfg}

		results, rejections := engine.SyncSource(context.Background(), src)
		if rejections != "" {
			t.Fatalf("unexpected rejections: %q", rejections)
		}
		if len(results) != 1 {
			t.Fatalf("expected exactly 1 result, got %d: %+v", len(results), results)
		}
		r := results[0]
		if r.ItemCount != 0 || r.Err != nil {
			t.Fatalf("expected ItemCount 0 and no error, got: %+v", r)
		}
		if !strings.Contains(r.Notice, "test") || !strings.Contains(r.Notice, "path") || !strings.Contains(r.Notice, "/nonexistent/**") {
			t.Errorf("expected the notice to name the webspace, field and value, got %q", r.Notice)
		}
	})

	t.Run("plugin-agnostic: a labels field matching nothing (mirrors the debug session's test-ext instance)", func(t *testing.T) {
		store := newTestStore(t)
		src := &fakeSource{
			name: "test-ext", sourceType: "example", vocabulary: []string{"labels"},
			matchFunc: func([]string) (*toposv1.MatchResponse, error) {
				return &toposv1.MatchResponse{}, nil
			},
		}
		cfg := &config.Config{Webspaces: map[string]config.Webspace{
			"test": {Match: map[string]config.MatchBlock{
				"test-ext": {"labels": {"untrust"}},
			}},
		}}
		engine := &Engine{Store: store, Config: cfg}

		results, rejections := engine.SyncSource(context.Background(), src)
		if rejections != "" {
			t.Fatalf("unexpected rejections: %q", rejections)
		}
		if len(results) != 1 {
			t.Fatalf("expected exactly 1 result, got %d: %+v", len(results), results)
		}
		r := results[0]
		if r.ItemCount != 0 || r.Err != nil {
			t.Fatalf("expected ItemCount 0 and no error, got: %+v", r)
		}
		if !strings.Contains(r.Notice, "test") || !strings.Contains(r.Notice, "labels") || !strings.Contains(r.Notice, "untrust") {
			t.Errorf("expected the notice to name the webspace, field and value even for a non-filesystem instance, got %q", r.Notice)
		}
	})
}

// TestSyncSource_ExplicitMatchBlockWithItemsRecordsNoNotice proves an
// explicit match block that DID match something never gets a notice.
func TestSyncSource_ExplicitMatchBlockWithItemsRecordsNoNotice(t *testing.T) {
	store := newTestStore(t)
	src := &fakeSource{
		name: "files", sourceType: "filesystem", vocabulary: []string{"path"},
		matchFunc: func([]string) (*toposv1.MatchResponse, error) {
			return &toposv1.MatchResponse{Items: []*toposv1.Item{
				{SourceId: "1", Title: "Doc", Fidelity: toposv1.LinkFidelity_LINK_FIDELITY_EXACT, DeepLink: "file:///x", TimestampUnix: 100},
			}}, nil
		},
	}
	cfg := &config.Config{Webspaces: map[string]config.Webspace{
		"test": {Match: map[string]config.MatchBlock{
			"files": {"path": {"/home/**"}},
		}},
	}}
	engine := &Engine{Store: store, Config: cfg}

	results, _ := engine.SyncSource(context.Background(), src)
	if len(results) != 1 || results[0].Notice != "" {
		t.Fatalf("expected no notice when the explicit block matched something, got: %+v", results)
	}
}

// TestSyncSource_KeywordsFallbackThatMatchedNothingRecordsNoNotice proves
// the fallback path — fanned across every source and legitimately
// matching nothing for most of them — never produces a notice, unlike the
// explicit-block path.
func TestSyncSource_KeywordsFallbackThatMatchedNothingRecordsNoNotice(t *testing.T) {
	store := newTestStore(t)
	src := &fakeSource{
		name: "paperless", sourceType: "paperless",
		matchFunc: func([]string) (*toposv1.MatchResponse, error) {
			return &toposv1.MatchResponse{}, nil
		},
	}
	cfg := &config.Config{Webspaces: map[string]config.Webspace{
		"ws": {Keywords: []string{"nonexistent-keyword"}},
	}}
	engine := &Engine{Store: store, Config: cfg}

	results, _ := engine.SyncSource(context.Background(), src)
	if len(results) != 1 || results[0].Notice != "" {
		t.Fatalf("expected no notice for a keywords-fallback zero match, got: %+v", results)
	}
}

// TestSyncSource_EveryItemRejectedReportsRejectionsNotAZeroMatchNotice
// proves PLUG-03 rejections and the zero-match notice never conflate: the
// plugin DID return something (tested via resp.GetItems(), before
// validation), so emptiness here is a rejection, never a match-config
// problem.
func TestSyncSource_EveryItemRejectedReportsRejectionsNotAZeroMatchNotice(t *testing.T) {
	store := newTestStore(t)
	src := &fakeSource{
		name: "files", sourceType: "filesystem", vocabulary: []string{"path"},
		matchFunc: func([]string) (*toposv1.MatchResponse, error) {
			return &toposv1.MatchResponse{Items: []*toposv1.Item{
				{SourceId: "bad", Title: "Missing link", Fidelity: toposv1.LinkFidelity_LINK_FIDELITY_EXACT, DeepLink: "", TimestampUnix: 100},
			}}, nil
		},
	}
	cfg := &config.Config{Webspaces: map[string]config.Webspace{
		"test": {Match: map[string]config.MatchBlock{
			"files": {"path": {"/home/**"}},
		}},
	}}
	engine := &Engine{Store: store, Config: cfg}

	results, rejections := engine.SyncSource(context.Background(), src)
	if rejections == "" || !strings.Contains(rejections, "bad") {
		t.Fatalf("expected the rejections string to name the rejected item, got: %q", rejections)
	}
	if len(results) != 1 || results[0].ItemCount != 0 {
		t.Fatalf("expected ItemCount 0, got: %+v", results)
	}
	if results[0].Notice != "" {
		t.Errorf("expected NO notice when the plugin returned an item (even if every item was rejected), got %q", results[0].Notice)
	}
}

// TestZeroMatchNotice_GlobShapedValueAlsoStatesTheExactMatchRule proves a
// glob-metacharacter-bearing value additionally states the exact-match
// rule in the same sentence, while a plain value gets the report-values
// clause instead.
func TestZeroMatchNotice_GlobShapedValueAlsoStatesTheExactMatchRule(t *testing.T) {
	globCases := []string{"*", "**", "?", "[abc]"}
	for _, v := range globCases {
		t.Run(v, func(t *testing.T) {
			got := zeroMatchNotice("test", map[string][]string{"path": {v}})
			if !strings.Contains(got, "never as glob patterns") {
				t.Errorf("expected the glob-metacharacter clause for value %q, got: %q", v, got)
			}
		})
	}

	got := zeroMatchNotice("test", map[string][]string{"path": {"plain-value"}})
	if strings.Contains(got, "never as glob patterns") {
		t.Errorf("expected NO glob clause for a plain value, got: %q", got)
	}
	if !strings.Contains(got, "compared exactly against the values this source reports") {
		t.Errorf("expected the report-values clause for a plain value, got: %q", got)
	}
}

// TestZeroMatchNotice_RendersEveryFieldSortedWithQuotedValues proves the
// rendered field order is deterministic (sorted by field name regardless
// of map iteration order) and every value is quoted, and that repeated
// calls on identical input are byte-identical.
func TestZeroMatchNotice_RendersEveryFieldSortedWithQuotedValues(t *testing.T) {
	fields := map[string][]string{
		"zebra": {"z-value"},
		"alpha": {"a-value"},
	}

	got := zeroMatchNotice("ws", fields)

	alphaIdx := strings.Index(got, "alpha")
	zebraIdx := strings.Index(got, "zebra")
	if alphaIdx == -1 || zebraIdx == -1 || alphaIdx > zebraIdx {
		t.Errorf("expected fields rendered in sorted order (alpha before zebra), got: %q", got)
	}
	if !strings.Contains(got, `"a-value"`) || !strings.Contains(got, `"z-value"`) {
		t.Errorf("expected every value quoted, got: %q", got)
	}

	again := zeroMatchNotice("ws", fields)
	if got != again {
		t.Errorf("expected zeroMatchNotice to be deterministic across repeated calls, got %q then %q", got, again)
	}
}
