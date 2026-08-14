package syncer

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/correlate"
	"github.com/davison/topos/kernel/index"
	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

// fakeSource is a test double satisfying correlate.Source without launching
// a real plugin subprocess — mirrors kernel/correlate/correlate_test.go's
// fixture shape, extended with an optional block channel so tests can
// prove single-flight coalescing by holding Match open across two
// concurrent Refresh calls.
type fakeSource struct {
	name       string
	sourceType string
	matchFunc  func() (*toposv1.MatchResponse, error)
	block      chan struct{} // if non-nil, Match blocks on receive before calling matchFunc

	mu    sync.Mutex
	calls int
}

func (f *fakeSource) Name() string              { return f.name }
func (f *fakeSource) SourceType() string        { return f.sourceType }
func (f *fakeSource) MatchVocabulary() []string { return []string{"keywords"} }

func (f *fakeSource) Match(_ context.Context, _ map[string][]string) (*toposv1.MatchResponse, error) {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	if f.block != nil {
		<-f.block
	}
	return f.matchFunc()
}

func (f *fakeSource) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
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

func testConfig() *config.Config {
	return &config.Config{Webspaces: map[string]config.Webspace{
		"house-move": {Keywords: []string{"house"}},
	}}
}

// TestTwoInstancesOfOnePluginType_StayDistinct is the invariant test wired
// into this phase's assumption-delta decision (05-01-PLAN.md): it goes red
// the instant a future change reintroduces the singular "source_type as
// identity" assumption anywhere on the sync/index/grant path. Two fake
// correlate.Source values share one SourceType() ("proton") but have
// distinct Name()s ("home-email", "work-email") — exactly the shape two
// [sources.*] entries pointing at the same plugin binary produce. Asserts:
// (a) two independent sync_runs series (LatestSyncRunPerSource has one
// entry per instance, never merged into one "proton" entry); (b)
// non-overlapping item id namespaces ("home-email:1" vs "work-email:1",
// not collapsed to "proton:1" for both); (c) a Refresh of one instance does
// not coalesce with, or block on, the other instance's single-flight key.
func TestTwoInstancesOfOnePluginType_StayDistinct(t *testing.T) {
	store := newTestStore(t)

	home := &fakeSource{
		name: "home-email", sourceType: "proton",
		matchFunc: func() (*toposv1.MatchResponse, error) {
			return okMatchResponse("1", "https://mail.proton.me/home/1")
		},
	}
	work := &fakeSource{
		name: "work-email", sourceType: "proton",
		matchFunc: func() (*toposv1.MatchResponse, error) {
			return okMatchResponse("1", "https://mail.proton.me/work/1")
		},
	}
	engine := &correlate.Engine{Store: store, Config: testConfig()}
	coord := NewCoordinator(store, engine, []correlate.Source{home, work})

	homeResult, err := coord.Refresh(context.Background(), "home-email")
	if err != nil {
		t.Fatalf("Refresh(home-email): %v", err)
	}
	workResult, err := coord.Refresh(context.Background(), "work-email")
	if err != nil {
		t.Fatalf("Refresh(work-email): %v", err)
	}
	if homeResult.Coalesced || workResult.Coalesced {
		t.Errorf("expected neither instance's Refresh to coalesce with the other's single-flight key, got home=%+v work=%+v", homeResult, workResult)
	}
	if homeResult.Source != "home-email" || workResult.Source != "work-email" {
		t.Errorf("expected each RunResult.Source to carry its own instance id, got home=%q work=%q", homeResult.Source, workResult.Source)
	}
	if homeResult.SourceType != "proton" || workResult.SourceType != "proton" {
		t.Errorf("expected both RunResult.SourceType to report the shared plugin kind 'proton', got home=%q work=%q", homeResult.SourceType, workResult.SourceType)
	}

	// (a) Two independent sync_runs series: LatestSyncRunPerSource must key
	// on instance id, never merge two instances of one plugin type into a
	// single "proton" entry.
	runs, err := store.LatestSyncRunPerSource(context.Background())
	if err != nil {
		t.Fatalf("LatestSyncRunPerSource: %v", err)
	}
	if _, ok := runs["home-email"]; !ok {
		t.Errorf("expected a sync_runs entry keyed 'home-email', got: %+v", runs)
	}
	if _, ok := runs["work-email"]; !ok {
		t.Errorf("expected a sync_runs entry keyed 'work-email', got: %+v", runs)
	}
	if _, ok := runs["proton"]; ok {
		t.Errorf("expected NO sync_runs entry keyed by the shared plugin kind 'proton' — instances must never merge, got: %+v", runs)
	}

	// (b) Non-overlapping item id namespaces: both instances matched
	// source_id "1", so a source_type-keyed id scheme would collide on
	// "proton:1" for both. Instance-keyed ids must not collide.
	items, err := store.StreamItems(context.Background(), "house-move", nil, index.ViewIncluded)
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}
	ids := map[string]bool{}
	for _, it := range items {
		ids[it.ID] = true
	}
	if !ids["home-email:1"] {
		t.Errorf("expected item id 'home-email:1' present, got items: %v", ids)
	}
	if !ids["work-email:1"] {
		t.Errorf("expected item id 'work-email:1' present, got items: %v", ids)
	}
	if len(items) != 2 {
		t.Errorf("expected exactly 2 distinct items (one per instance, never merged), got %d: %v", len(items), ids)
	}

	// (c) A single-flight key collision would manifest as the SECOND
	// Refresh call's underlying Match never actually running (coalesced
	// into the first). Both fakeSource call counts must be exactly 1.
	if home.callCount() != 1 {
		t.Errorf("expected home-email's Match called exactly once, got %d", home.callCount())
	}
	if work.callCount() != 1 {
		t.Errorf("expected work-email's Match called exactly once, got %d", work.callCount())
	}
}

func okMatchResponse(sourceID, deepLink string) (*toposv1.MatchResponse, error) {
	return &toposv1.MatchResponse{Items: []*toposv1.Item{
		{SourceId: sourceID, Title: "Doc", Fidelity: toposv1.LinkFidelity_LINK_FIDELITY_EXACT, DeepLink: deepLink, TimestampUnix: 100},
	}}, nil
}

// TestRefresh_ConcurrentCallsCoalesceIntoOneSyncCycle is the single-flight
// proof (D-06): two concurrent Refresh calls for the same source, with
// Match blocked open until both have entered it, drive exactly one
// underlying Match invocation, and at least one caller's RunResult reports
// it coalesced. This test MUST fail if singleflight is removed.
func TestRefresh_ConcurrentCallsCoalesceIntoOneSyncCycle(t *testing.T) {
	store := newTestStore(t)
	block := make(chan struct{})
	src := &fakeSource{
		name: "paperless", sourceType: "paperless", block: block,
		matchFunc: func() (*toposv1.MatchResponse, error) {
			return okMatchResponse("1", "http://paperless.lan/documents/1")
		},
	}
	engine := &correlate.Engine{Store: store, Config: testConfig()}
	coord := NewCoordinator(store, engine, []correlate.Source{src})

	var wg sync.WaitGroup
	results := make([]RunResult, 2)
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = coord.Refresh(context.Background(), "paperless")
		}(i)
	}

	// Give both goroutines time to block inside Match before releasing
	// them, so the assertion below genuinely proves a single-flight
	// coalesce rather than a fast sequential race that happened not to
	// overlap.
	time.Sleep(50 * time.Millisecond)
	close(block)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("Refresh[%d]: unexpected error: %v", i, err)
		}
	}
	if got := src.callCount(); got != 1 {
		t.Fatalf("expected exactly 1 underlying Match call for a single-flight coalesce, got %d", got)
	}
	if !results[0].Coalesced && !results[1].Coalesced {
		t.Error("expected at least one caller's RunResult to report Coalesced true")
	}
	if results[0].Status != "ok" || results[1].Status != "ok" {
		t.Errorf("expected both results ok, got %+v / %+v", results[0], results[1])
	}
}

func TestRefresh_MatchErrorReturnsErrorStatusNotGoError(t *testing.T) {
	store := newTestStore(t)
	src := &fakeSource{
		name: "paperless", sourceType: "paperless",
		matchFunc: func() (*toposv1.MatchResponse, error) {
			return nil, errors.New("connection refused")
		},
	}
	engine := &correlate.Engine{Store: store, Config: testConfig()}
	coord := NewCoordinator(store, engine, []correlate.Source{src})

	result, err := coord.Refresh(context.Background(), "paperless")
	if err != nil {
		t.Fatalf("Refresh should not return a Go error for a source-level failure: %v", err)
	}
	if result.Status != "error" || result.Error == "" {
		t.Errorf("expected error status with a non-empty message, got: %+v", result)
	}
}

func TestRefresh_UnknownSourceReturnsErrUnknownSource(t *testing.T) {
	store := newTestStore(t)
	engine := &correlate.Engine{Store: store, Config: testConfig()}
	coord := NewCoordinator(store, engine, nil)

	_, err := coord.Refresh(context.Background(), "nope")
	if !errors.Is(err, ErrUnknownSource) {
		t.Fatalf("expected ErrUnknownSource, got %v", err)
	}
}

func TestRefreshAll_OneSourceErrorDoesNotPreventOthers(t *testing.T) {
	store := newTestStore(t)
	ok := &fakeSource{
		name: "paperless", sourceType: "paperless",
		matchFunc: func() (*toposv1.MatchResponse, error) {
			return okMatchResponse("1", "http://paperless.lan/documents/1")
		},
	}
	bad := &fakeSource{
		name: "silverbullet", sourceType: "silverbullet",
		matchFunc: func() (*toposv1.MatchResponse, error) {
			return nil, errors.New("connection refused")
		},
	}
	engine := &correlate.Engine{Store: store, Config: testConfig()}
	coord := NewCoordinator(store, engine, []correlate.Source{ok, bad})

	results := coord.RefreshAll(context.Background())
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d: %+v", len(results), results)
	}
	var sawOK, sawErr bool
	for _, r := range results {
		if r.Source == "paperless" && r.Status == "ok" {
			sawOK = true
		}
		if r.Source == "silverbullet" && r.Status == "error" {
			sawErr = true
		}
	}
	if !sawOK || !sawErr {
		t.Fatalf("expected one ok and one error result, got: %+v", results)
	}
}

// TestRefresh_RejectedItemMessageReachesFinishedRunError proves a
// rejected item's message is never silently dropped: it must appear both
// on the returned RunResult and on the sync_runs row FinishSyncRun wrote.
func TestRefresh_RejectedItemMessageReachesFinishedRunError(t *testing.T) {
	store := newTestStore(t)
	src := &fakeSource{
		name: "paperless", sourceType: "paperless",
		matchFunc: func() (*toposv1.MatchResponse, error) {
			return &toposv1.MatchResponse{Items: []*toposv1.Item{
				{SourceId: "good", Title: "Valid", Fidelity: toposv1.LinkFidelity_LINK_FIDELITY_EXACT, DeepLink: "http://paperless.lan/documents/good", TimestampUnix: 100},
				{SourceId: "bad", Title: "Missing link", Fidelity: toposv1.LinkFidelity_LINK_FIDELITY_EXACT, DeepLink: "", TimestampUnix: 200},
			}}, nil
		},
	}
	engine := &correlate.Engine{Store: store, Config: testConfig()}
	coord := NewCoordinator(store, engine, []correlate.Source{src})

	result, err := coord.Refresh(context.Background(), "paperless")
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("expected status ok (a rejection is per-item, not fatal), got %q", result.Status)
	}
	if !strings.Contains(result.Error, "bad") {
		t.Errorf("expected the rejected item's source id in the result error, got: %q", result.Error)
	}

	runs, err := store.LatestSyncRunPerSource(context.Background())
	if err != nil {
		t.Fatalf("LatestSyncRunPerSource: %v", err)
	}
	run, ok := runs["paperless"]
	if !ok {
		t.Fatal("expected a recorded run for paperless")
	}
	if !strings.Contains(run.Error, "bad") {
		t.Errorf("expected the finished run's error column to contain the rejected item's source id, got: %q", run.Error)
	}
}

// TestRefresh_RepeatedRefreshDoesNotDuplicateItems proves two successive
// refreshes of an unchanged source leave the indexed item count for that
// source unchanged.
func TestRefresh_RepeatedRefreshDoesNotDuplicateItems(t *testing.T) {
	store := newTestStore(t)
	src := &fakeSource{
		name: "paperless", sourceType: "paperless",
		matchFunc: func() (*toposv1.MatchResponse, error) {
			return okMatchResponse("1", "http://paperless.lan/documents/1")
		},
	}
	engine := &correlate.Engine{Store: store, Config: testConfig()}
	coord := NewCoordinator(store, engine, []correlate.Source{src})

	if _, err := coord.Refresh(context.Background(), "paperless"); err != nil {
		t.Fatalf("first Refresh: %v", err)
	}
	if _, err := coord.Refresh(context.Background(), "paperless"); err != nil {
		t.Fatalf("second Refresh: %v", err)
	}

	items, err := store.StreamItems(context.Background(), "house-move", nil, index.ViewIncluded)
	if err != nil {
		t.Fatalf("StreamItems: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected item count unchanged at 1 after a second unchanged refresh, got %d", len(items))
	}
}

// TestRefresh_CancelledContextStillFinalisesSyncRun is the regression proof
// for the orphaned-run bug behind the permanently-stuck "Syncing..."
// indicator. syncOne used to finalise the sync_runs row using the same
// cancellable ctx the sync itself ran under, so when the scheduler's ctx
// was cancelled mid-sync at shutdown, database/sql rejected the finalising
// UPDATE with "context canceled" and the row stayed at status "running"
// forever — nothing else in the system ever finalises it.
//
// The source's Match cancels the context to simulate shutdown landing
// mid-sync. Afterwards the run must be recorded as finished (with any
// outcome), and the source must not still report as syncing. This test
// MUST fail if the finalise write is put back on the cancellable ctx.
func TestRefresh_CancelledContextStillFinalisesSyncRun(t *testing.T) {
	store := newTestStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	src := &fakeSource{
		name: "proton", sourceType: "proton",
		matchFunc: func() (*toposv1.MatchResponse, error) {
			cancel() // shutdown arrives while this source is mid-sync
			return okMatchResponse("1", "https://mail.proton.me/1")
		},
	}
	engine := &correlate.Engine{Store: store, Config: testConfig()}
	coord := NewCoordinator(store, engine, []correlate.Source{src})

	if _, err := coord.Refresh(ctx, "proton"); err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	// Query with a fresh context, exactly as a later HTTP request would.
	fresh := context.Background()

	syncing, err := store.SyncingSources(fresh)
	if err != nil {
		t.Fatalf("SyncingSources: %v", err)
	}
	if syncing["proton"] {
		t.Error("expected proton to not be syncing after a cancelled sync: the run was left orphaned at status \"running\"")
	}

	runs, err := store.LatestSyncRunPerSource(fresh)
	if err != nil {
		t.Fatalf("LatestSyncRunPerSource: %v", err)
	}
	run := runs["proton"]
	if run.Status == "running" || run.FinishedUnix == 0 {
		t.Errorf("expected the interrupted run to be finalised with an outcome and a finished time, got: %+v", run)
	}
}

// --- 12-09-PLAN.md Task 2: aggregation and isolation ---

// TestSyncOne_ZeroMatchNoticeLeavesStatusOKAndErrorEmpty proves that,
// through a real store + engine + coordinator, a zero-matching explicit
// match block leaves Status exactly "ok" and Error exactly empty while
// recording a non-empty Notice — both on the returned RunResult and on
// the persisted sync_runs row.
func TestSyncOne_ZeroMatchNoticeLeavesStatusOKAndErrorEmpty(t *testing.T) {
	store := newTestStore(t)
	src := &fakeSource{
		name: "files", sourceType: "filesystem",
		matchFunc: func() (*toposv1.MatchResponse, error) {
			return &toposv1.MatchResponse{}, nil
		},
	}
	cfg := &config.Config{Webspaces: map[string]config.Webspace{
		"test": {Match: map[string]config.MatchBlock{
			"files": {"keywords": {"nonexistent"}},
		}},
	}}
	engine := &correlate.Engine{Store: store, Config: cfg}
	coord := NewCoordinator(store, engine, []correlate.Source{src})

	result, err := coord.Refresh(context.Background(), "files")
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("expected Status exactly \"ok\", got %q", result.Status)
	}
	if result.Error != "" {
		t.Errorf("expected Error exactly empty, got %q", result.Error)
	}
	if result.Notice == "" {
		t.Error("expected a non-empty Notice")
	}

	runs, err := store.LatestSyncRunPerSource(context.Background())
	if err != nil {
		t.Fatalf("LatestSyncRunPerSource: %v", err)
	}
	run, ok := runs["files"]
	if !ok {
		t.Fatal("expected a recorded run for files")
	}
	if run.Status != "ok" || run.Error != "" || run.Notice == "" {
		t.Errorf("expected the persisted run's status ok, error empty, notice non-empty, got: %+v", run)
	}
}

// boomOnKeywordSource is a fakeSource-shaped test double that fails Match
// for exactly the webspace whose resolved "keywords" field is "boom" and
// answers zero items for every other webspace — used by
// TestSyncOne_NoticeNeverMasksASyncError to give one source two
// different, deterministic per-webspace outcomes in the same sync cycle
// (fakeSource's shared matchFunc ignores the fields argument, so it can't
// vary by webspace on its own).
type boomOnKeywordSource struct {
	name       string
	sourceType string
}

func (s *boomOnKeywordSource) Name() string              { return s.name }
func (s *boomOnKeywordSource) SourceType() string        { return s.sourceType }
func (s *boomOnKeywordSource) MatchVocabulary() []string { return []string{"keywords"} }
func (s *boomOnKeywordSource) Match(_ context.Context, fields map[string][]string) (*toposv1.MatchResponse, error) {
	kw := fields["keywords"]
	if len(kw) > 0 && kw[0] == "boom" {
		return nil, errors.New("connection refused")
	}
	return &toposv1.MatchResponse{}, nil
}

// TestSyncOne_NoticeNeverMasksASyncError proves a notice and a genuine
// sync error coexist without either being dropped: a config with two
// webspaces, one whose Match genuinely errors and one whose explicit
// match block matches nothing, must report Status "error" with the
// failure named in Error AND a non-empty Notice — an advisory neither
// replaces an error nor is swallowed by one.
func TestSyncOne_NoticeNeverMasksASyncError(t *testing.T) {
	store := newTestStore(t)
	src := &boomOnKeywordSource{name: "files", sourceType: "filesystem"}
	cfg := &config.Config{Webspaces: map[string]config.Webspace{
		"broken": {Match: map[string]config.MatchBlock{
			"files": {"keywords": {"boom"}},
		}},
		"empty": {Match: map[string]config.MatchBlock{
			"files": {"keywords": {"nonexistent"}},
		}},
	}}
	engine := &correlate.Engine{Store: store, Config: cfg}
	coord := NewCoordinator(store, engine, []correlate.Source{src})

	result, err := coord.Refresh(context.Background(), "files")
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if result.Status != "error" {
		t.Errorf("expected Status \"error\" (a genuine Match failure), got %q", result.Status)
	}
	if !strings.Contains(result.Error, "connection refused") {
		t.Errorf("expected Error to name the genuine failure, got %q", result.Error)
	}
	if result.Notice == "" {
		t.Error("expected the zero-match webspace's Notice to still be recorded alongside the error")
	}

	runs, err := store.LatestSyncRunPerSource(context.Background())
	if err != nil {
		t.Fatalf("LatestSyncRunPerSource: %v", err)
	}
	run, ok := runs["files"]
	if !ok {
		t.Fatal("expected a recorded run for files")
	}
	if run.Status != "error" || !strings.Contains(run.Error, "connection refused") || run.Notice == "" {
		t.Errorf("expected the persisted run to carry BOTH the error and the notice, got: %+v", run)
	}
}

// alwaysZeroMatchSource answers every Match call with zero items and no
// error, regardless of the fields it receives — used by
// TestSyncOne_NoticesFromSeveralWebspacesJoinSortedAndBounded to build a
// fixture with many zero-matching explicit-block webspaces.
type alwaysZeroMatchSource struct {
	name       string
	sourceType string
}

func (s *alwaysZeroMatchSource) Name() string              { return s.name }
func (s *alwaysZeroMatchSource) SourceType() string        { return s.sourceType }
func (s *alwaysZeroMatchSource) MatchVocabulary() []string { return []string{"keywords"} }
func (s *alwaysZeroMatchSource) Match(context.Context, map[string][]string) (*toposv1.MatchResponse, error) {
	return &toposv1.MatchResponse{}, nil
}

// TestSyncOne_NoticesFromSeveralWebspacesJoinSortedAndBounded proves
// joinNotices' determinism and bound end to end: enough zero-matching
// webspaces to exceed maxJoinedNotices produces a persisted notice
// listing the first entries in sorted webspace order, naming the number
// suppressed, and byte-identical across repeated runs of the same
// fixture.
func TestSyncOne_NoticesFromSeveralWebspacesJoinSortedAndBounded(t *testing.T) {
	buildWebspaces := func() map[string]config.Webspace {
		out := map[string]config.Webspace{}
		for _, name := range []string{"ws-g", "ws-f", "ws-e", "ws-d", "ws-c", "ws-b", "ws-a"} {
			out[name] = config.Webspace{Match: map[string]config.MatchBlock{
				"files": {"keywords": {"nonexistent-" + name}},
			}}
		}
		return out
	}

	src := &alwaysZeroMatchSource{name: "files", sourceType: "filesystem"}

	store1 := newTestStore(t)
	engine1 := &correlate.Engine{Store: store1, Config: &config.Config{Webspaces: buildWebspaces()}}
	coord1 := NewCoordinator(store1, engine1, []correlate.Source{src})
	result1, err := coord1.Refresh(context.Background(), "files")
	if err != nil {
		t.Fatalf("Refresh (fixture 1): %v", err)
	}

	store2 := newTestStore(t)
	engine2 := &correlate.Engine{Store: store2, Config: &config.Config{Webspaces: buildWebspaces()}}
	coord2 := NewCoordinator(store2, engine2, []correlate.Source{src})
	result2, err := coord2.Refresh(context.Background(), "files")
	if err != nil {
		t.Fatalf("Refresh (fixture 2): %v", err)
	}

	if result1.Notice != result2.Notice {
		t.Errorf("expected byte-identical notices across repeated runs of the same fixture, got %q vs %q", result1.Notice, result2.Notice)
	}
	if !strings.Contains(result1.Notice, "ws-a") {
		t.Errorf("expected the first sorted webspace name present, got %q", result1.Notice)
	}
	if strings.Contains(result1.Notice, "ws-g") || strings.Contains(result1.Notice, "ws-f") {
		t.Errorf("expected the 6th/7th sorted webspace names suppressed (bounded at maxJoinedNotices=5), got %q", result1.Notice)
	}
	if !strings.Contains(result1.Notice, "2 more") {
		t.Errorf("expected the suppressed count named, got %q", result1.Notice)
	}

	runs, err := store1.LatestSyncRunPerSource(context.Background())
	if err != nil {
		t.Fatalf("LatestSyncRunPerSource: %v", err)
	}
	if runs["files"].Notice != result1.Notice {
		t.Errorf("expected the persisted notice to match the returned RunResult's notice, got %q vs %q", runs["files"].Notice, result1.Notice)
	}
}
