// Package supervisor owns the apply seam a config.Store swap has nowhere
// else to hand its result to (07-02-PLAN.md Task 1, D-06/D-07). Every
// current call site of pluginhost/syncer captures its plugin host and
// coordinator once, at kernel boot, from a fixed config snapshot — this
// package generalises that boot sequence into a long-lived value whose
// Apply method repeats the relevant part of it after every successful
// config.Store.Save/Reload, so the running kernel and the on-disk config
// never disagree for longer than one HTTP request.
package supervisor

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"sync"

	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/correlate"
	"github.com/davison/topos/kernel/index"
	"github.com/davison/topos/kernel/pluginhost"
	"github.com/davison/topos/kernel/syncer"
	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

// Supervisor owns the lifetime of the plugin host, the sync coordinator,
// and the background scheduler goroutine set — the three subsystems a
// config.Store swap must reconfigure in place. NewSupervisor performs the
// kernel's boot sequence once; Apply repeats the relevant part of it
// after every successful config.Store.Save/Reload.
type Supervisor struct {
	idx        *index.Store
	cfgStore   *config.Store
	pluginsDir string
	logger     hclog.Logger

	// baseCtx is the long-lived context every scheduler generation derives
	// its own cancellable context from (never a per-request ctx passed
	// into Apply — a request's context is cancelled once its HTTP handler
	// returns, which would kill a scheduler generation started from it
	// almost immediately).
	baseCtx context.Context

	mu     sync.Mutex // serializes Apply calls — never two applies in flight together
	host   *pluginhost.Host
	coord  *syncer.Coordinator
	cfg    *config.Config // the config.Config the currently running host/coord/scheduler set was built from
	cancel context.CancelFunc
	done   chan struct{} // closed when the CURRENT scheduler generation's Run has fully returned
}

// NewSupervisor performs the kernel's boot sequence — discover and launch
// every configured plugin, validate the launched set's match
// configuration against the webspace config (D-05's second phase), build
// the sync coordinator, and start the background scheduler — and returns
// a *Supervisor holding the result. Callers hold the returned value for
// the kernel's lifetime and call Shutdown() when done; Apply is the seam
// every subsequent config.Store.Save/Reload must call so the running
// kernel catches up with the swapped config (D-06).
func NewSupervisor(ctx context.Context, idx *index.Store, cfgStore *config.Store, pluginsDir string, logger hclog.Logger) (*Supervisor, error) {
	if logger == nil {
		logger = hclog.NewNullLogger()
	}

	s := &Supervisor{
		idx:        idx,
		cfgStore:   cfgStore,
		pluginsDir: pluginsDir,
		logger:     logger,
		baseCtx:    ctx,
	}

	cfg := cfgStore.Expanded()

	host, err := pluginhost.Discover(ctx, pluginsDir, cfg.Sources, logger)
	if err != nil {
		return nil, err
	}
	if err := pluginhost.ValidateMatchConfig(cfg, host); err != nil {
		host.Shutdown()
		return nil, err
	}

	s.host = host
	s.coord = newCoordinator(idx, cfg, host)
	s.cfg = cfg
	s.startScheduler(cfg)

	return s, nil
}

// newCoordinator builds the correlate.Engine + syncer.Coordinator pair
// over host's currently launched plugins — the same construction
// cmd/topos/main.go's now-removed newCoordinator helper performed at
// boot, generalised here as the one place this pairing is ever built.
func newCoordinator(idx *index.Store, cfg *config.Config, host *pluginhost.Host) *syncer.Coordinator {
	engine := &correlate.Engine{Store: idx, Config: cfg}
	plugins := host.Plugins()
	sources := make([]correlate.Source, len(plugins))
	for i, p := range plugins {
		sources[i] = p
	}
	return syncer.NewCoordinator(idx, engine, sources)
}

// startScheduler launches a fresh background scheduler goroutine set
// against cfg and s.coord, deriving its own cancellable context from
// s.baseCtx (never from a per-request ctx) so a later Apply or Shutdown
// can stop exactly this generation's goroutines independently of
// whatever HTTP request triggered it. Caller must hold s.mu.
func (s *Supervisor) startScheduler(cfg *config.Config) {
	ctx, cancel := context.WithCancel(s.baseCtx)
	done := make(chan struct{})
	s.cancel = cancel
	s.done = done

	sched := &syncer.Scheduler{Coordinator: s.coord, Config: cfg, Logger: s.logger}
	go func() {
		sched.Run(ctx)
		close(done)
	}()
}

// stopScheduler cancels the current scheduler generation's context and
// blocks until its Run call has fully returned, so nothing from the OLD
// generation can still be calling into the plugin host or coordinator
// while a caller (Apply, Shutdown) goes on to mutate either. Caller must
// hold s.mu.
func (s *Supervisor) stopScheduler() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.done != nil {
		<-s.done
	}
}

// Host returns the currently launched plugin host.
func (s *Supervisor) Host() *pluginhost.Host {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.host
}

// Coordinator returns the current sync coordinator.
func (s *Supervisor) Coordinator() *syncer.Coordinator {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.coord
}

// Fetch satisfies kernel/httpapi.Fetcher by delegating to the currently
// launched plugin host, resolved fresh via Host() on every call — never a
// pointer captured once at Router-construction time. This matters because
// Reconcile mutates the SAME *pluginhost.Host in place (so a stale
// *Host reference would still work for Fetch), but the sibling
// Refresh/RefreshAll methods below do NOT have that luxury (Apply
// replaces the *syncer.Coordinator outright — see its own doc comment) —
// Fetch is written the identical way for consistency, and so a future
// change to how Reconcile commits its result can never silently
// reintroduce staleness here.
func (s *Supervisor) Fetch(ctx context.Context, source, sourceID string, variant toposv1.ContentVariant) (pluginhost.FetchResult, error) {
	return s.Host().Fetch(ctx, source, sourceID, variant)
}

// ProbeSources satisfies kernel/httpapi.HealthProber, delegating to the
// current plugin host.
func (s *Supervisor) ProbeSources(ctx context.Context) []pluginhost.SourceHealth {
	return s.Host().ProbeSources(ctx)
}

// Refresh satisfies kernel/httpapi.Refresher by delegating to the CURRENT
// coordinator, resolved fresh via Coordinator() on every call. This is
// load-bearing, not defensive style: Apply replaces s.coord with a brand
// new *syncer.Coordinator on every successful reconcile (a coordinator has
// no in-place "update sources" seam), so a caller holding a *Coordinator
// snapshot taken once at kernel-start (e.g. passed directly to
// httpapi.Router instead of the Supervisor itself) would keep dispatching
// manual refreshes against an increasingly stale source set — a source
// added by a later apply would be permanently unreachable through
// POST /api/sources/{name}/refresh and POST /api/sync. Every caller of a
// sync must go through the supervisor itself, never a Coordinator pointer
// captured once (cmd/topos/main.go passes sup, not sup.Coordinator(), into
// Router accordingly).
func (s *Supervisor) Refresh(ctx context.Context, sourceName string) (syncer.RunResult, error) {
	return s.Coordinator().Refresh(ctx, sourceName)
}

// RefreshAll satisfies kernel/httpapi.Refresher, delegating to the current
// coordinator — see Refresh's doc comment for why this must never be a
// coordinator snapshot captured once.
func (s *Supervisor) RefreshAll(ctx context.Context) []syncer.RunResult {
	return s.Coordinator().RefreshAll(ctx)
}

// Shutdown cancels the current scheduler generation and kills every
// launched plugin subprocess. Not safe to call Apply after Shutdown.
func (s *Supervisor) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopScheduler()
	if s.host != nil {
		s.host.Shutdown()
	}
}

// Apply is the seam every successful config.Store.Save/Reload must call
// (D-06): it reconciles the launched plugin set against the just-swapped
// config, removes index rows for any instance no longer configured
// (D-07), rebuilds the coordinator, and restarts the scheduler — which
// fires every currently configured source's first refresh immediately by
// its own existing design (Scheduler.Run), satisfying D-07's eager
// reconcile for new and connection-changed instances with no second
// mechanism. All of this runs under s.mu, so two overlapping applies (a
// save landing while a reload from another tab is also mid-flight) never
// interleave.
//
// In-flight sync handling (07-RESEARCH.md Open Question 2, decided here):
// the OLD scheduler generation's context is cancelled and its Run call is
// awaited BEFORE Host.Reconcile runs — Reconcile mutates the launched
// plugin set in place, and a scheduler goroutine from the old generation
// still calling Coordinator.Refresh against that set while Reconcile
// tears it down would race it. A sync that was mid-flight when the
// cancellation lands is handled by Coordinator.syncOne's own existing
// detached sync_runs finalize (kernel/syncer/coordinator.go) — identical
// to how kernel shutdown already handles a mid-flight sync — so no
// sync_runs row is ever left stranded at status "running" by an apply.
//
// On any error, Apply changes nothing further and returns: the caller
// (ConfigSaveHandler/ConfigReloadHandler) decides how to surface it. The
// config file and the swapped config.Store state are already valid by
// construction — Store.Save/Reload validate before swapping — so an
// error here is a runtime reconciliation problem (e.g. a plugin binary
// failed to relaunch), never a config-validity problem. A failed apply
// restarts the scheduler against the previously running (unchanged) host
// and coordinator pairing, so periodic sync does not stall indefinitely
// while the operator retries the save or reloads the file.
func (s *Supervisor) Apply(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	newCfg := s.cfgStore.Expanded()
	oldCfg := s.cfg

	s.stopScheduler()

	if err := s.host.Reconcile(ctx, newCfg.Sources, s.logger); err != nil {
		s.startScheduler(oldCfg)
		return fmt.Errorf("supervisor: apply: %w", err)
	}

	if err := pluginhost.ValidateMatchConfig(newCfg, s.host); err != nil {
		s.startScheduler(oldCfg)
		return fmt.Errorf("supervisor: apply: %w", err)
	}

	// D-07: an instance present before this apply and absent now has its
	// index rows and sync history removed right away, across every
	// webspace it participated in — a re-added instance under the same id
	// must never inherit phantom history (T-07-13).
	for _, name := range removedInstances(oldCfg, newCfg) {
		if err := s.idx.DeleteSourceItems(ctx, name); err != nil {
			s.startScheduler(newCfg)
			s.coord = newCoordinator(s.idx, newCfg, s.host)
			s.cfg = newCfg
			return fmt.Errorf("supervisor: apply: delete items for removed source %q: %w", name, err)
		}
		if err := s.idx.DeleteSyncRuns(ctx, name); err != nil {
			s.startScheduler(newCfg)
			s.coord = newCoordinator(s.idx, newCfg, s.host)
			s.cfg = newCfg
			return fmt.Errorf("supervisor: apply: delete sync history for removed source %q: %w", name, err)
		}
	}

	s.coord = newCoordinator(s.idx, newCfg, s.host)
	s.startScheduler(newCfg)

	// A match-only edit (a webspace or match block changed, but an
	// instance's own connection config did not) has no relaunch of its own
	// to trigger an eager resync from. Dispatch one explicitly for every
	// instance whose config.Source is byte-identical to before, so it is
	// not left waiting for its own next scheduled tick. This is safe to
	// call even though the scheduler generation just started may also be
	// about to refresh the same instance — Coordinator.Refresh's
	// single-flight guarantee coalesces the two into one sync, never two.
	if !reflect.DeepEqual(oldCfg.Webspaces, newCfg.Webspaces) {
		coord := s.coord
		for name, src := range newCfg.Sources {
			if oldSrc, ok := oldCfg.Sources[name]; ok && reflect.DeepEqual(oldSrc, src) {
				go coord.Refresh(context.Background(), name)
			}
		}
	}

	s.cfg = newCfg
	return nil
}

// removedInstances returns the names present in oldCfg.Sources and absent
// from newCfg.Sources, sorted for deterministic iteration order.
func removedInstances(oldCfg, newCfg *config.Config) []string {
	var out []string
	for name := range oldCfg.Sources {
		if _, ok := newCfg.Sources[name]; !ok {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}
