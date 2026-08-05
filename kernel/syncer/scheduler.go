package syncer

import (
	"context"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/config"
)

// Scheduler runs one background goroutine per configured source, each
// ticking at that source's own resolved sync interval (global default or
// per-source override — Config.SyncIntervalFor) and calling
// Coordinator.Refresh — never the correlation engine directly, so every
// scheduled sync still goes through the coordinator's single-flight
// guarantee (D-05, D-06).
type Scheduler struct {
	Coordinator *Coordinator
	Config      *config.Config
	Logger      hclog.Logger
}

// Run starts one goroutine per configured source and blocks until ctx is
// done, then waits for every goroutine to exit before returning — a
// caller can rely on Run returning as a clean shutdown signal. Each
// source's first refresh fires immediately (retaining Phase 1's
// startup-sync behavior as that source's first scheduled run, per D-05),
// then repeats on a time.Ticker at its own resolved interval.
func (s *Scheduler) Run(ctx context.Context) {
	logger := s.Logger
	if logger == nil {
		logger = hclog.NewNullLogger()
	}

	var wg sync.WaitGroup
	for name := range s.Config.Sources {
		interval, err := s.Config.SyncIntervalFor(name)
		if err != nil {
			// Config.Validate already rejects an unparseable/non-positive
			// interval at load time, so this should be unreachable in
			// practice for a Config that came through config.Load — but a
			// hand-built *config.Config (e.g. in a test) could still hit
			// it, and skipping this source's goroutine is safer than
			// panicking time.NewTicker on a zero duration.
			logger.Error("skipping scheduler goroutine: could not resolve sync interval", "source", name, "error", err)
			continue
		}

		wg.Add(1)
		go func(name string, interval time.Duration) {
			defer wg.Done()
			s.runSource(ctx, name, interval, logger)
		}(name, interval)
	}
	wg.Wait()
}

// runSource is one source's scheduler goroutine: refresh immediately,
// then on every tick, until ctx is done.
func (s *Scheduler) runSource(ctx context.Context, name string, interval time.Duration, logger hclog.Logger) {
	s.refreshAndLog(ctx, name, logger)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.refreshAndLog(ctx, name, logger)
		}
	}
}

// refreshAndLog calls Coordinator.Refresh and logs the outcome — the
// source name and error string only, never the source's config, its
// token, or any item content (T-02-10).
func (s *Scheduler) refreshAndLog(ctx context.Context, name string, logger hclog.Logger) {
	result, err := s.Coordinator.Refresh(ctx, name)
	if err != nil {
		logger.Error("scheduled sync failed to dispatch", "source", name, "error", err.Error())
		return
	}
	if result.Status == "error" {
		logger.Error("scheduled sync run failed", "source", name, "error", result.Error)
		return
	}
	logger.Info("scheduled sync run completed", "source", name, "item_count", result.ItemCount, "coalesced", result.Coalesced)
}
