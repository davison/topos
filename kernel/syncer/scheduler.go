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
//
// A Scheduler value is a single generation: Run has no add/remove-source
// capability once started (07-RESEARCH.md Pitfall 1). kernel/supervisor's
// Apply (07-02-PLAN.md Task 1, D-06/D-07's hot-apply) is the only caller
// that needs a source set to change at runtime — it does so by cancelling
// the current generation's context, waiting for its Run call to return,
// and starting a brand new *Scheduler against the newly swapped config,
// rather than this type growing any in-place mutation of its own. Every
// fresh generation's Run fires each configured source's first refresh
// immediately, by the existing design below — which is now retried on
// failure (see FirstRefreshRetryDelays) so a plugin subprocess that needed
// a moment after launch to become ready is not pinned behind an errored
// sync_runs row for the whole sync interval (08-UAT.md gap G-08-4).
type Scheduler struct {
	Coordinator *Coordinator
	Config      *config.Config
	Logger      hclog.Logger

	// FirstRefreshRetryDelays overrides the backoff schedule applied to a
	// generation's immediate first refresh when it reports an error status
	// (see firstRefresh). nil (the zero value — what supervisor.startScheduler
	// produces, since it is left untouched by this field's introduction)
	// means "use defaultFirstRefreshRetryDelays"; an explicitly non-nil but
	// empty slice means "no retries at all — behave exactly as before this
	// field existed". Tests set this to a short, compressed schedule so the
	// suite does not need to sleep out the production delays.
	FirstRefreshRetryDelays []time.Duration
}

// defaultFirstRefreshRetryDelays is the production backoff schedule for a
// generation's first refresh (see firstRefresh). As of plan 08-14 (closing
// 08-REVIEW.md WR-01 and 08-VERIFICATION.md G-08-5), no plugin in this repo
// blocks its handshake on a live login — the WhatsApp plugin's own bounded
// login wait was moved off the launch path entirely, onto a background
// goroutine that runs concurrently with goplugin.Serve — so this schedule
// now has to cover the WHOLE window between a plugin subprocess completing
// its handshake and being able to answer Match successfully, not merely the
// remainder left over after a launch-time absorber. This is a superseding
// retry, not a readiness probe: the kernel has no RPC to ask a plugin "are
// you ready yet", and inventing one would be a plugin contract change,
// which this plan deliberately does not make. The cost bound is unchanged:
// at most two extra Match calls per source per generation. The schedule's
// numeric values still suffice for that wider window: a WhatsApp login
// round trip is measured in hundreds of milliseconds, so the first retry
// landing two seconds after the immediate refresh already clears it with a
// wide margin, and a genuinely broken source still ends on an errored row a
// few seconds later exactly as before.
var defaultFirstRefreshRetryDelays = []time.Duration{2 * time.Second, 5 * time.Second}

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

// runSource is one source's scheduler goroutine: refresh immediately (with
// bounded retry on failure, via firstRefresh — deliberately first-run-only,
// see firstRefresh's doc comment), then on every tick, until ctx is done.
func (s *Scheduler) runSource(ctx context.Context, name string, interval time.Duration, logger hclog.Logger) {
	s.firstRefresh(ctx, name, logger)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Ticker refreshes are never retried: a ticker refresh already
			// has the next tick as its own retry, and this is the same
			// Coordinator.Refresh every other caller (the manual-refresh
			// HTTP route, the CLI) goes through — those callers must keep
			// their existing one-call-one-answer behaviour, so a refresh
			// button never silently takes several extra seconds.
			s.refreshAndLog(ctx, name, logger)
		}
	}
}

// firstRefresh performs a generation's immediate first refresh for a
// source, retrying it on a bounded, context-cancellable backoff schedule
// when it reports an error status. This is deliberately scoped to a
// generation's first refresh only (see runSource and Coordinator.Refresh's
// other callers' doc comments) — closing 08-UAT.md gap G-08-4's root
// causes 2 and 3: there is no readiness gate between launching a plugin
// subprocess and issuing its first Match, and the errored sync_runs row
// that first Match failure records is what LatestSyncRunPerSource and the
// stream banner render until the next scheduled refresh, which defaults to
// 15 minutes away. Retrying here means a successful retry writes a LATER
// sync_runs row, which LatestSyncRunPerSource's MAX(id)-per-source
// selection then picks as the latest — superseding the earlier errored row
// within seconds instead of the sync interval. A source that is
// genuinely broken still exhausts every delay and ends on an errored row,
// unchanged from today's behaviour, a few seconds later.
func (s *Scheduler) firstRefresh(ctx context.Context, name string, logger hclog.Logger) {
	delays := s.FirstRefreshRetryDelays
	if delays == nil {
		delays = defaultFirstRefreshRetryDelays
	}

	if ok := s.refreshAndLog(ctx, name, logger); ok {
		return
	}

	for _, delay := range delays {
		logger.Info("retrying first refresh after error", "source", name, "delay", delay.String())

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}

		if ok := s.refreshAndLog(ctx, name, logger); ok {
			return
		}
	}
}

// refreshAndLog calls Coordinator.Refresh and logs the outcome — the
// source name and error string only, never the source's config, its
// token, or any item content (T-02-10). Reports true when the run
// finished without an error status (false for both a dispatch error and a
// result.Status == "error"), so firstRefresh can decide whether to retry.
func (s *Scheduler) refreshAndLog(ctx context.Context, name string, logger hclog.Logger) bool {
	result, err := s.Coordinator.Refresh(ctx, name)
	if err != nil {
		logger.Error("scheduled sync failed to dispatch", "source", name, "error", err.Error())
		return false
	}
	if result.Status == "error" {
		logger.Error("scheduled sync run failed", "source", name, "error", result.Error)
		return false
	}
	logger.Info("scheduled sync run completed", "source", name, "item_count", result.ItemCount, "coalesced", result.Coalesced)
	return true
}
