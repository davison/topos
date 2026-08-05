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

func (f *fakeSource) Name() string       { return f.name }
func (f *fakeSource) SourceType() string { return f.sourceType }

func (f *fakeSource) Match(_ context.Context, _ []string) (*toposv1.MatchResponse, error) {
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

	items, err := store.StreamItems(context.Background(), "house-move")
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

	syncing, err := store.SyncingSourceTypes(fresh)
	if err != nil {
		t.Fatalf("SyncingSourceTypes: %v", err)
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
