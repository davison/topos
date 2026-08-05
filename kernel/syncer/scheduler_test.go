package syncer

import (
	"context"
	"sync"
	"testing"
	"time"

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

func (c *countingSource) Name() string       { return c.name }
func (c *countingSource) SourceType() string { return c.sourceType }
func (c *countingSource) Match(context.Context, []string) (*toposv1.MatchResponse, error) {
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
