package syncer

import (
	"context"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/correlate"
	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

// countingSource is a correlate.Source that records how many times its
// Match RPC was invoked — the seam scheduler_test.go drives the real
// Coordinator/Scheduler pair against (Scheduler.Coordinator is a concrete
// *Coordinator per the locked interface, so the fake lives one layer
// lower, at the correlate.Source the Coordinator wraps).
type countingSource struct {
	name       string
	sourceType string

	mu    sync.Mutex
	calls int
}

func (c *countingSource) Name() string              { return c.name }
func (c *countingSource) SourceType() string        { return c.sourceType }
func (c *countingSource) MatchVocabulary() []string { return []string{"keywords"} }
func (c *countingSource) Match(context.Context, map[string][]string) (*toposv1.MatchResponse, error) {
	c.mu.Lock()
	c.calls++
	c.mu.Unlock()
	return &toposv1.MatchResponse{}, nil
}

func (c *countingSource) callCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

// flakySource is a correlate.Source that fails its first N Match calls
// with a gRPC-shaped codes.Unavailable error (mirroring the shape a real
// not-yet-ready plugin subprocess answers with — see G-08-4's kernel-side
// diagnosis) and succeeds on every call after that, counting calls under
// its own mutex in the same shape countingSource already uses.
type flakySource struct {
	name       string
	sourceType string
	failFirstN int

	mu    sync.Mutex
	calls int
}

func (f *flakySource) Name() string              { return f.name }
func (f *flakySource) SourceType() string        { return f.sourceType }
func (f *flakySource) MatchVocabulary() []string { return []string{"keywords"} }
func (f *flakySource) Match(context.Context, map[string][]string) (*toposv1.MatchResponse, error) {
	f.mu.Lock()
	f.calls++
	n := f.calls
	f.mu.Unlock()
	if n <= f.failFirstN {
		return nil, status.Error(codes.Unavailable, "flakySource: not ready yet")
	}
	return &toposv1.MatchResponse{}, nil
}

func (f *flakySource) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func TestScheduler_Run_ImmediateFirstRunThenTicks(t *testing.T) {
	store := newTestStore(t)
	src := &countingSource{name: "paperless", sourceType: "paperless"}
	cfg := &config.Config{
		Sync:      config.SyncConfig{Interval: "50ms"},
		Sources:   map[string]config.Source{"paperless": {Plugin: "x", BaseURL: "http://x", Token: "t"}},
		Webspaces: map[string]config.Webspace{"house-move": {Keywords: []string{"house"}}},
	}
	engine := &correlate.Engine{Store: store, Config: cfg}
	coord := NewCoordinator(store, engine, []correlate.Source{src})
	sched := &Scheduler{Coordinator: coord, Config: cfg}

	ctx, cancel := context.WithTimeout(context.Background(), 220*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		sched.Run(ctx)
		close(done)
	}()

	// Almost immediately (well under one 50ms interval), the source should
	// already have been refreshed at least once — the immediate first run.
	time.Sleep(20 * time.Millisecond)
	if got := src.callCount(); got < 1 {
		t.Fatalf("expected at least 1 immediate refresh before one interval elapsed, got %d", got)
	}

	<-done
	// "at least N refreshes" only — never an exact count, to avoid flaking
	// on a loaded machine.
	if got := src.callCount(); got < 2 {
		t.Errorf("expected at least 2 refreshes over ~220ms at a 50ms interval, got %d", got)
	}
}

func TestScheduler_Run_ReturnsOnContextCancel(t *testing.T) {
	store := newTestStore(t)
	src := &countingSource{name: "paperless", sourceType: "paperless"}
	cfg := &config.Config{
		Sync:      config.SyncConfig{Interval: "1h"},
		Sources:   map[string]config.Source{"paperless": {Plugin: "x", BaseURL: "http://x", Token: "t"}},
		Webspaces: map[string]config.Webspace{},
	}
	engine := &correlate.Engine{Store: store, Config: cfg}
	coord := NewCoordinator(store, engine, []correlate.Source{src})
	sched := &Scheduler{Coordinator: coord, Config: cfg}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		sched.Run(ctx)
		close(done)
	}()

	// Let the immediate first run happen, then cancel — Run must return
	// promptly rather than waiting out the 1h interval.
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Scheduler.Run did not return within 2s of context cancellation")
	}
}

func TestScheduler_Run_TwoSourcesTickIndependently(t *testing.T) {
	store := newTestStore(t)
	fast := &countingSource{name: "fast", sourceType: "fast"}
	slow := &countingSource{name: "slow", sourceType: "slow"}
	cfg := &config.Config{
		Sync: config.SyncConfig{Interval: "1h"}, // global default, overridden per source below
		Sources: map[string]config.Source{
			"fast": {Plugin: "x", BaseURL: "http://x", Token: "t", SyncInterval: "20ms"},
			"slow": {Plugin: "x", BaseURL: "http://x", Token: "t", SyncInterval: "500ms"},
		},
		// At least one webspace is required for Match to be invoked at
		// all — correlate.Engine.SyncSource calls Match once per
		// configured webspace, so a zero-webspace config would make this
		// test's call counts trivially zero for both sources regardless
		// of ticking behavior.
		Webspaces: map[string]config.Webspace{"house-move": {Keywords: []string{"house"}}},
	}
	engine := &correlate.Engine{Store: store, Config: cfg}
	coord := NewCoordinator(store, engine, []correlate.Source{fast, slow})
	sched := &Scheduler{Coordinator: coord, Config: cfg}

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		sched.Run(ctx)
		close(done)
	}()
	<-done

	fastCalls, slowCalls := fast.callCount(), slow.callCount()
	if fastCalls <= slowCalls {
		t.Errorf("expected the 20ms-interval source to refresh strictly more often than the 500ms-interval source over the same span, got fast=%d slow=%d", fastCalls, slowCalls)
	}
}

// TestScheduler_FirstRefreshRetriesUntilTheSourceIsReady is the RED half of
// G-08-4's kernel-side fix: a source whose generation-first Match call
// fails must not be pinned on that errored sync_runs row for the whole
// sync interval — the scheduler must retry, and a later successful retry
// must supersede the earlier error as LatestSyncRunPerSource's answer.
func TestScheduler_FirstRefreshRetriesUntilTheSourceIsReady(t *testing.T) {
	store := newTestStore(t)
	src := &flakySource{name: "mock-01", sourceType: "mock", failFirstN: 1}
	cfg := &config.Config{
		Sync:      config.SyncConfig{Interval: "1h"}, // no ticker may contribute a second call
		Sources:   map[string]config.Source{"mock-01": {Plugin: "x", BaseURL: "http://x", Token: "t"}},
		Webspaces: map[string]config.Webspace{"house-move": {Keywords: []string{"house"}}},
	}
	engine := &correlate.Engine{Store: store, Config: cfg}
	coord := NewCoordinator(store, engine, []correlate.Source{src})
	sched := &Scheduler{
		Coordinator:             coord,
		Config:                  cfg,
		FirstRefreshRetryDelays: []time.Duration{20 * time.Millisecond, 20 * time.Millisecond},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		sched.Run(ctx)
		close(done)
	}()
	<-done

	// Vacuity guard: without a retry this would never exceed 1.
	if got := src.callCount(); got < 2 {
		t.Fatalf("expected the source to be Matched at least twice (initial failure + retry), got %d", got)
	}

	runs, err := store.LatestSyncRunPerSource(context.Background())
	if err != nil {
		t.Fatalf("LatestSyncRunPerSource: %v", err)
	}
	run, ok := runs["mock-01"]
	if !ok {
		t.Fatalf("expected a sync_runs entry for mock-01, found none")
	}
	if run.Status != "ok" {
		t.Errorf("expected the LATEST sync run to be status \"ok\" (the retry superseding the initial error), got %q (error=%q)", run.Status, run.Error)
	}
}

// TestScheduler_FirstRefreshGivesUpAndLeavesTheErrorRecorded proves the
// retry is bounded: a source that never recovers still ends on an errored
// row after exactly 1+len(delays) attempts — the decision G-08-4's
// missing[2] asks for, made explicit: a genuinely broken source still
// looks exactly as it does today, a few seconds later.
func TestScheduler_FirstRefreshGivesUpAndLeavesTheErrorRecorded(t *testing.T) {
	store := newTestStore(t)
	src := &flakySource{name: "mock-01", sourceType: "mock", failFirstN: 1000}
	cfg := &config.Config{
		Sync:      config.SyncConfig{Interval: "1h"},
		Sources:   map[string]config.Source{"mock-01": {Plugin: "x", BaseURL: "http://x", Token: "t"}},
		Webspaces: map[string]config.Webspace{"house-move": {Keywords: []string{"house"}}},
	}
	engine := &correlate.Engine{Store: store, Config: cfg}
	coord := NewCoordinator(store, engine, []correlate.Source{src})
	delays := []time.Duration{20 * time.Millisecond, 20 * time.Millisecond}
	sched := &Scheduler{
		Coordinator:             coord,
		Config:                  cfg,
		FirstRefreshRetryDelays: delays,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		sched.Run(ctx)
		close(done)
	}()
	<-done

	wantCalls := 1 + len(delays)
	if got := src.callCount(); got != wantCalls {
		t.Errorf("expected exactly %d calls (1 initial + %d retries, bounded), got %d", wantCalls, len(delays), got)
	}

	runs, err := store.LatestSyncRunPerSource(context.Background())
	if err != nil {
		t.Fatalf("LatestSyncRunPerSource: %v", err)
	}
	run, ok := runs["mock-01"]
	if !ok {
		t.Fatalf("expected a sync_runs entry for mock-01, found none")
	}
	if run.Status != "error" {
		t.Errorf("expected the latest sync run to be status \"error\" for a permanently broken source, got %q", run.Status)
	}
	if run.Error == "" {
		t.Errorf("expected a non-empty error on the latest sync run")
	}
}

// TestScheduler_FirstRefreshRetryStopsOnContextCancel proves the retry
// loop is context-cancellable: a permanently failing source with a
// multi-second delay schedule must not delay scheduler shutdown.
func TestScheduler_FirstRefreshRetryStopsOnContextCancel(t *testing.T) {
	store := newTestStore(t)
	src := &flakySource{name: "mock-01", sourceType: "mock", failFirstN: 1000}
	cfg := &config.Config{
		Sync:      config.SyncConfig{Interval: "1h"},
		Sources:   map[string]config.Source{"mock-01": {Plugin: "x", BaseURL: "http://x", Token: "t"}},
		Webspaces: map[string]config.Webspace{},
	}
	engine := &correlate.Engine{Store: store, Config: cfg}
	coord := NewCoordinator(store, engine, []correlate.Source{src})
	sched := &Scheduler{
		Coordinator: coord,
		Config:      cfg,
		FirstRefreshRetryDelays: []time.Duration{
			10 * time.Second, 10 * time.Second,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		sched.Run(ctx)
		close(done)
	}()

	// Let the immediate (failing) first attempt happen, then cancel mid-backoff.
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Scheduler.Run did not return within 2s of context cancellation during first-refresh retry backoff")
	}
}
