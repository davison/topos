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
	"errors"
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
	idx      *index.Store
	cfgStore *config.Store
	dirs     pluginhost.Dirs
	logger   hclog.Logger

	// baseCtx is the long-lived context every scheduler generation derives
	// its own cancellable context from (never a per-request ctx passed
	// into Apply — a request's context is cancelled once its HTTP handler
	// returns, which would kill a scheduler generation started from it
	// almost immediately).
	baseCtx context.Context

	// mu is the MUTATION lock only (08-13-PLAN.md Task 1(D), closing
	// G-08-5): it serializes Apply, SuspendInstance, its resume closure and
	// Shutdown — never two mutations in flight together — but the reader
	// path (Host/Coordinator, and therefore Fetch/ProbeSources/Refresh/
	// RefreshAll) deliberately does NOT sit behind it. A resume closure
	// runs synchronously on the WhatsApp link poll/cancel HTTP request path
	// (kernel/httpapi/whatsapplink.go) and holds this lock across a real
	// subprocess launch (Host.Reconcile); before this fix that used to
	// freeze every other source's item-fetch, health-probe and
	// manual-refresh routes for the duration — phase success criterion 4's
	// "every other source is unaffected", violated (08-VERIFICATION.md
	// G-08-5). See genMu below for what readers take instead.
	mu    sync.Mutex
	host  *pluginhost.Host
	coord *syncer.Coordinator
	cfg   *config.Config // the config.Config the currently running host/coord/scheduler set was built from

	// genMu guards ONLY s.host and s.coord (08-13-PLAN.md Task 1(D)):
	// Host()/Coordinator() take genMu.RLock() and never touch s.mu at all,
	// so a reader never waits behind a mutation's Reconcile call — no
	// matter how long a plugin subprocess takes to launch. Every write to
	// s.host or s.coord (NewSupervisor's two assignments, and
	// commitGeneration's s.coord assignment) takes genMu.Lock() for the
	// assignment alone, never across a Reconcile or a stopScheduler. Every
	// writer of s.host/s.coord already holds s.mu too (mutations are still
	// fully serialized by it), so a read of either field from INSIDE a
	// mutation path (Apply, SuspendInstance, its resume closure, Shutdown)
	// needs no genMu read lock to be correct, and must not take one where
	// genMu.Lock() is already held in the same call path.
	genMu sync.RWMutex

	cancel context.CancelFunc
	done   chan struct{} // closed when the CURRENT scheduler generation's Run has fully returned

	// genCtx and genWG are generation-scoped (08-09-PLAN.md Task 2, closing
	// 08-UAT.md G-08-3's untracked-eager-resync sibling): replaced wholesale
	// by startScheduler, drained by stopScheduler — the SAME "caller must
	// hold s.mu" convention cancel/done already state. Every background sync
	// this package dispatches OUTSIDE Scheduler.Run's own goroutine set
	// (Apply's eager resync, below) must derive its context from genCtx and
	// register on genWG before its own goroutine starts, so stopScheduler's
	// wait bounds ALL work belonging to the generation it is tearing down —
	// not only the scheduler's own tick goroutines (which Scheduler.Run's
	// internal sync.WaitGroup and the done channel above already cover) —
	// so a suspend or resume (SuspendInstance, above — now itself a
	// generation change and on the WhatsApp link-start HTTP request path)
	// can never block on a dispatched sync it has no way to cancel.
	genCtx context.Context
	genWG  *sync.WaitGroup

	// suspended holds one entry per instance name SuspendInstance has
	// currently stopped (WR-02, 08-REVIEW.md): the *pluginhost.Plugin
	// value it was launched as immediately before being killed, kept
	// around purely for its cached Describe-learned fields
	// (SourceType/PluginDisplayName/MatchVocabulary — plain struct reads,
	// no live RPC) so Apply can still validate match config against it
	// and can skip trying to relaunch it while it remains suspended.
	// Entries are added by SuspendInstance and removed by the resume
	// closure it returns; both run under s.mu, so this map is never read
	// or written concurrently with itself.
	suspended map[string]*pluginhost.Plugin
}

// NewSupervisor performs the kernel's boot sequence — discover and launch
// every configured plugin, validate the launched set's match
// configuration against the webspace config (D-05's second phase), build
// the sync coordinator, and start the background scheduler — and returns
// a *Supervisor holding the result. Callers hold the returned value for
// the kernel's lifetime and call Shutdown() when done; Apply is the seam
// every subsequent config.Store.Save/Reload must call so the running
// kernel catches up with the swapped config (D-06).
func NewSupervisor(ctx context.Context, idx *index.Store, cfgStore *config.Store, dirs pluginhost.Dirs, logger hclog.Logger) (*Supervisor, error) {
	if logger == nil {
		logger = hclog.NewNullLogger()
	}

	s := &Supervisor{
		idx:      idx,
		cfgStore: cfgStore,
		dirs:     dirs,
		logger:   logger,
		baseCtx:  ctx,
	}

	cfg := cfgStore.Expanded()

	host, err := pluginhost.Discover(ctx, dirs, cfgStore.Raw(), cfg.Sources, logger)
	if err != nil {
		return nil, err
	}
	if err := pluginhost.ValidateMatchConfig(cfg, host); err != nil {
		host.Shutdown()
		return nil, err
	}

	s.genMu.Lock()
	s.host = host
	s.coord = newCoordinator(idx, cfg, host)
	s.genMu.Unlock()
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
	s.genCtx = ctx
	s.genWG = &sync.WaitGroup{}

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
	// s.cancel has already fired by this point (above), so every tracked
	// goroutine's context is already done — this wait is bounded by how
	// fast an in-flight Match RPC aborts on cancellation, never unbounded
	// (08-09-PLAN.md Task 2, closing 08-UAT.md G-08-3's eager-resync
	// sibling: before this fix, Apply's eager-resync goroutines were
	// dispatched on a detached context.Background() and untracked here,
	// so this wait could not bound them at all).
	if s.genWG != nil {
		s.genWG.Wait()
	}
}

// Host returns the currently launched plugin host. Takes genMu.RLock()
// only — never s.mu (08-13-PLAN.md Task 1(D), G-08-5) — so this never
// waits behind an in-flight mutation's Host.Reconcile call, no matter how
// long a plugin subprocess takes to launch.
func (s *Supervisor) Host() *pluginhost.Host {
	s.genMu.RLock()
	defer s.genMu.RUnlock()
	return s.host
}

// Coordinator returns the current sync coordinator. Takes genMu.RLock()
// only — never s.mu — for the identical reason Host() does, above.
func (s *Supervisor) Coordinator() *syncer.Coordinator {
	s.genMu.RLock()
	defer s.genMu.RUnlock()
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

// LaunchFailures delegates to the CURRENT plugin host's LaunchFailures,
// resolved fresh via Host() on every call — never a captured host pointer,
// the same "never a snapshot taken once" discipline ProbeSources above
// already follows (Reconcile mutates the SAME *pluginhost.Host in place, so
// this stays correct across a config apply with no extra care needed).
func (s *Supervisor) LaunchFailures() []pluginhost.LaunchFailure {
	return s.Host().LaunchFailures()
}

// PluginIcon satisfies kernel/httpapi.PluginIconProvider (09-01-PLAN.md
// Task 2), delegating to the current plugin host resolved fresh via
// Host() — the same "never a pointer captured once" discipline Fetch and
// ProbeSources above already follow, for the identical reason: Reconcile
// mutates the SAME *pluginhost.Host in place, so this stays correct across
// a config apply with no extra care needed here.
func (s *Supervisor) PluginIcon(binary string) ([]byte, string, bool) {
	return s.Host().PluginIcon(binary)
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

// SuspendInstance satisfies kernel/httpapi.Suspender (08-03-PLAN.md Task 3,
// D-01's hard requirement): stops exactly one named running instance for
// the duration the caller holds the returned resume closure, so a
// link-mode subprocess (kernel/httpapi/whatsapplink.go) and the
// pluginhost-launched instance it is about to re-pair can never both hold
// the same WhatsApp session store open at once — the second, independent
// layer behind the plugin's own store-lock (whatsapp_store_in_use).
//
// When name is absent from the currently launched Host — the D-02
// Add-Source flow's own case, where the instance being linked has not been
// saved to config yet at all — this is a deliberate no-op: it returns a
// resume closure that does nothing and a nil error, so the HTTP handler
// needs no special-casing between "suspend a running instance" and
// "nothing to suspend" at its own call site.
//
// When name IS present, SuspendInstance reconciles the Host against the
// current source map with exactly that one name removed — reusing
// Host.Reconcile exactly as Apply does, so the same launch/kill discipline
// (T-07-11: a partial apply never looks successful) governs this seam too
// — which stops the named instance's subprocess and releases its store
// lock, and returns a resume closure that reconciles the instance back in
// by reading s.cfg.Sources FRESH at resume time (not a value captured at
// suspend time): if a config.Store.Save/Reload lands while the caller
// holds the resume closure, resume still reflects whatever the running
// kernel's config-of-record is by the time it is actually called, rather
// than resurrecting a since-edited or since-removed instance definition.
//
// A SUSPEND OR RESUME IS A GENERATION CHANGE (08-UAT.md G-08-3; corrected
// here from this method's own prior claim that no scheduler generation is
// ever stopped/restarted): a real-device WhatsApp re-link session
// (D-03) that suspended-then-resumed the "whatsapp" instance left every
// sync of that instance — scheduled tick, manual refresh, eager resync —
// calling Match through the go-plugin client Host.Reconcile had already
// Kill()ed, because syncer.Coordinator captures its *pluginhost.Plugin
// handles once at construction (commitGeneration's own doc comment: "a
// coordinator has no in-place update seam") and neither SuspendInstance
// nor its resume closure ever rebuilt one. Only a config save
// (Apply -> commitGeneration) or a kernel restart healed it, and a re-link
// never saves config, so a *successful* pairing left that source's sync
// broken indefinitely — pinned as its latest sync_runs row. Both branches
// below now go through stopScheduler -> Host.Reconcile -> commitGeneration
// in that order, the identical sequence Apply already uses and
// commitGeneration's own doc comment already requires: s.host, s.coord,
// s.cfg and the running scheduler generation must always reflect ONE AND
// THE SAME generation. Three consequences a future reader needs:
//
//   - While the instance is suspended the coordinator has no entry for it,
//     so Coordinator.Refresh answers ErrUnknownSource before a sync_runs
//     row is ever started (see coordinator.go's own doc comment). A
//     scheduled tick during the suspension window therefore logs a
//     dispatch failure and records nothing — the intended answer: a
//     lifecycle artifact must never be pinned to a source's health surface
//     as a failed sync.
//   - Each suspend and each resume restarts the scheduler generation, and
//     Scheduler.Run fires every configured source's first refresh
//     immediately by its own existing design — so a link session costs two
//     eager full refreshes, the identical consequence any config save
//     already has, coalesced by the coordinator's single-flight guarantee.
//     This is accepted, not overlooked.
//   - stopScheduler blocks until the old generation's Run has fully
//     returned, and that now happens on the link-start HTTP request path
//     (kernel/httpapi/whatsapplink.go). It is bounded by how fast a
//     cancelled context aborts an in-flight Match RPC — never unbounded —
//     which is why every background sync this package dispatches is also
//     bound to its own generation's context and wait group (see the
//     genCtx/genWG fields below and Apply's eager-resync dispatch): the one
//     class of dispatched sync that used to carry an uncancellable context
//     is exactly the class that would otherwise make this wait unbounded.
//
// Still true, unchanged by this fix: this touches exactly the named
// instance's launched-set membership — no index row is ever touched by
// SuspendInstance or its resume closure, and no OTHER launched plugin is
// affected (Reconcile's own launch/kill discipline, T-07-11, still governs
// this seam). It still takes the SAME s.mu Apply takes (and, like Apply,
// calls Host.Reconcile only while holding it), so a suspension and a
// config apply can never interleave — one always fully completes before
// the other's own Reconcile call begins. The WR-02 s.suspended bookkeeping
// is also unchanged in shape: an instance is recorded as suspended for
// exactly the window between SuspendInstance returning and its resume
// closure actually running (see both failure branches below, which leave
// this window exactly where a Reconcile failure logically puts it).
func (s *Supervisor) SuspendInstance(ctx context.Context, name string) (func(context.Context) error, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var found *pluginhost.Plugin
	for _, p := range s.host.Plugins() {
		if p.Name() == name {
			found = p
			break
		}
	}
	if found == nil {
		return func(context.Context) error { return nil }, nil
	}

	withoutName := make(map[string]config.Source, len(s.cfg.Sources))
	for n, src := range s.cfg.Sources {
		if n == name {
			continue
		}
		withoutName[n] = src
	}

	// Stop the current generation BEFORE Reconcile tears down the plugin
	// set it holds handles to — the identical reason Apply stops the
	// scheduler before its own Reconcile call, and it also closes the
	// mid-flight-kill window (T-08-adjacency) that a suspension landing
	// beside an in-flight sync would otherwise open.
	s.stopScheduler()

	if err := s.host.Reconcile(ctx, s.cfgStore.Raw(), withoutName, s.logger); err != nil {
		// Mirror Apply's own pre-Reconcile failure branch exactly, and for
		// the identical reason: Reconcile's own T-07-11 guarantee is that a
		// launch failure leaves the previously running set fully intact, so
		// the OLD generation is the consistent one here and must be put
		// back — a suspend that fails must never leave the kernel with no
		// scheduler running at all.
		s.startScheduler(s.cfg)
		return nil, fmt.Errorf("supervisor: suspend instance %q: %w", name, err)
	}

	// Record name as suspended (WR-02, 08-REVIEW.md) — found is the
	// *pluginhost.Plugin the just-committed Reconcile call killed;
	// keeping it around lets a concurrent Apply still validate this
	// instance's match config (ValidateMatchConfigWithSuspended) and skip
	// trying to relaunch it (Apply's own reconcileSources filtering)
	// while it remains suspended, rather than losing the race for its
	// store lock against the live link-mode subprocess this suspension
	// exists to make room for.
	if s.suspended == nil {
		s.suspended = make(map[string]*pluginhost.Plugin)
	}
	s.suspended[name] = found

	// s.cfg, not a new config: config-of-record is deliberately unchanged
	// by a suspension (the instance is still configured, just not
	// launched) — what changed is the launched set, which is precisely
	// what commitGeneration rebuilds the coordinator over.
	s.commitGeneration(s.cfg)

	resume := func(ctx context.Context) error {
		s.mu.Lock()
		defer s.mu.Unlock()
		// Deleted before Reconcile, exactly as before this fix: a resume
		// whose Reconcile fails below must leave the instance neither
		// launched nor suspended, because the link session is genuinely
		// over and a later Apply must be free to relaunch it.
		delete(s.suspended, name)

		s.stopScheduler()

		if err := s.host.Reconcile(ctx, s.cfgStore.Raw(), s.cfg.Sources, s.logger); err != nil {
			// Same mirror of Apply's pre-Reconcile branch as the suspend
			// path above: the old (still-suspended-minus-name) generation
			// is what Reconcile left intact, so it is what gets restarted.
			s.startScheduler(s.cfg)
			return fmt.Errorf("supervisor: resume instance %q: %w", name, err)
		}

		s.commitGeneration(s.cfg)
		return nil
	}
	return resume, nil
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

// commitGeneration is reached from exactly one place in Apply — the single
// shared commit site the whole post-Reconcile region funnels through,
// whether that region ends in success or in either kind of failure it can
// return (07-10-PLAN.md; originally 07-09-PLAN.md, closing
// 07-VERIFICATION.md gaps[0], when Apply still had four separate call
// sites). NewSupervisor's own inline boot sequence deliberately does not
// use it: at boot s.host, s.coord and s.cfg are being set for the first
// time, not adopted as a new generation over a running one, so there is no
// "single site every branch funnels through" property to preserve there.
// commitGeneration performs exactly three steps in exactly this order:
// install the coordinator built over the host's CURRENT plugin set, record
// cfg as the generation s.cfg reflects, then start the scheduler against
// cfg. The order is load-bearing: startScheduler reads s.coord at call
// time into the syncer.Scheduler value it constructs, so a coordinator
// installed AFTER startScheduler runs would be invisible to the goroutine
// already launched — the same class of defect as gaps[0] itself, on a
// different seam. Caller must hold s.mu (the same convention startScheduler
// and stopScheduler already state). The read of s.host and the write of
// s.coord below need no genMu.RLock/Lock ceremony beyond the write itself
// (08-13-PLAN.md Task 1(D)): the caller already holds s.mu, and every
// writer of s.host also holds s.mu, so s.host cannot change underneath
// this read; the s.coord assignment still takes genMu.Lock(), for the
// assignment alone, so a concurrent reader (Coordinator(), on the
// lock-free path) never observes a torn value.
func (s *Supervisor) commitGeneration(cfg *config.Config) {
	newCoord := newCoordinator(s.idx, cfg, s.host)
	s.genMu.Lock()
	s.coord = newCoord
	s.genMu.Unlock()
	s.cfg = cfg
	s.startScheduler(cfg)
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
// Suspension-aware (WR-02, 08-REVIEW.md): any instance name currently
// present in s.suspended (SuspendInstance, above — an active WhatsApp
// link/re-link session in flight) is excluded from what Reconcile is
// asked to launch here, and validated via
// pluginhost.ValidateMatchConfigWithSuspended instead of
// ValidateMatchConfig, so an unrelated config save landing during that
// window neither tries to relaunch the suspended instance (which would
// lose the store-lock race against the live link subprocess and fail
// this ENTIRE save) nor spuriously rejects a webspace the suspended
// instance participates in as "has no launched plugin". The suspended
// instance's own resume closure reconciles it back in once the session
// ends, reading s.cfg.Sources fresh at that point — including whatever
// this Apply call itself just committed.
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
// This paragraph's contract is unchanged by 07-09 and remains pinned by
// TestApply_MidFlightSyncLeavesNoStrandedRunningRow.
//
// Apply has exactly two error regimes, divided by whether s.host.Reconcile
// has committed (07-VERIFICATION.md gaps[0] — corrected here; this
// comment previously claimed a failed apply ALWAYS restarts the scheduler
// against an unchanged host, which was only ever true for the first of
// these two regimes, and 07-VERIFICATION.md named this doc comment as
// contradicting the code beneath it):
//
//   - Before Reconcile commits (Reconcile itself returns a non-nil
//     error): the previously running plugin set is genuinely untouched —
//     Reconcile's own documented guarantee (kernel/pluginhost/host.go,
//     T-07-11) is that every new/changed launch is attempted before
//     anything currently running is torn down, so a launch failure
//     leaves the prior set fully intact. Apply keeps the OLD generation:
//     it restarts the scheduler against oldCfg, and the running kernel is
//     simply unchanged.
//
//   - After Reconcile commits (every failure beneath it: the D-07
//     removed-instance index cleanup, and the match-vocabulary check):
//     there is no undo. Reconcile has already mutated the launched
//     plugin set in place — the replaced instance's old subprocess is
//     killed and dead by the time Reconcile returns nil — and the config
//     store already swapped to newCfg before Apply was ever called, so
//     cfgStore.Expanded() is already the config-of-record, agreeing with
//     the file on disk and with every other per-request
//     cfgStore.Expanded() read elsewhere in the kernel. The only
//     self-consistent state left available is the NEW generation, so
//     the whole post-Reconcile region adopts newCfg through the single
//     commitGeneration call and still returns its error unchanged.
//     Re-running Reconcile against oldCfg.Sources as a rollback is
//     deliberately rejected: a rollback that performs real subprocess
//     launches can itself fail, leaving the kernel strictly worse off
//     than the defect it was meant to fix, with no third recourse.
//
//     The D-07 removed-instance index cleanup (cleanupRemovedInstances)
//     is part of this regime and therefore runs on EVERY path through
//     it, not only the success path (07-VERIFICATION.md 2026-08-08
//     gaps[0]; 07-REVIEW.md's post-07-09 CR-01): once Reconcile has
//     returned nil the removed instances are already gone from the host,
//     so their index rows are the only trace left, and gating their
//     deletion on a check that runs later — such as the match-vocabulary
//     check immediately below — strands them permanently, because this
//     same call also advances s.cfg past the removal, after which
//     removedInstances can never again compute them as removed and no
//     retry path exists (T-07-13). A per-instance cleanup failure is
//     collected rather than returned early, so one instance's failure
//     cannot abandon the rest of the batch. Every collected cleanup
//     failure is joined via errors.Join with the match-vocabulary error
//     into the single error Apply returns — vocabulary error first
//     (D-09: the operator-actionable message must reach the UI verbatim
//     and lead) — so a rejection and a cleanup fault are both reported
//     rather than one silently masking the other.
//
//     Immediately after the cleanup, purgeDeparticipatedWebspaceRows
//     (07-16-PLAN.md, closes 07-UAT.md G-07-7) runs for the identical
//     reason and at the identical point in this region: it clears the
//     webspace_items rows for every (webspace, instance) pair still
//     configured on both sides whose participation just flipped from true
//     to false — the finer-grained sibling of the whole-instance D-07
//     cleanup above it, catching a webspace merely NARROWED to exclude a
//     still-configured instance rather than an instance removed outright.
//     It is deliberately synchronous, unlike the eager resync dispatched
//     near the end of this function: it is a pure local index write with
//     no plugin RPC, so — unlike a resync, which performs a real Match RPC
//     per webspace against a plugin subprocess reading a mailbox or an
//     encrypted database — it can run on the request path without making
//     one unreachable source's latency felt by every config save
//     (T-07-62's prohibition). The pre-Reconcile failure branch above
//     purges nothing, for the same reason it skips the D-07 cleanup: the
//     old generation is genuinely intact and still running against the
//     old config, so there is nothing stale to clear yet. Its own error is
//     joined last, after the cleanup error, leaving the vocabulary error
//     leading.
//
// Whichever branch Apply exits by, s.host, s.coord, s.cfg and the running
// scheduler generation always reflect ONE AND THE SAME config generation
// as each other — never a new host paired with a stale coordinator, which
// is the shape gaps[0] took before this fix.
//
// Adopting the new generation on a post-Reconcile failure is state
// repair, not success: it does NOT convert a rejected save into an
// apparent one. Apply still returns a non-nil error on every one of these
// branches, so ConfigSaveHandler/ConfigReloadHandler still answer 500
// apply_failed and the operator is still told the runtime may be out of
// sync with the file until a successful reload
// (kernel/httpapi/config.go).
//
// D-08's "an invalid file on reload keeps the last-good config running"
// guarantee is enforced upstream of Apply entirely, in
// config.Store.Reload, which validates before it swaps — a structurally
// invalid file never reaches Apply at all. A match-vocabulary rejection
// is a different class: only a live post-launch cross-check against a
// launched plugin's declared vocabulary can detect it
// (pluginhost.ValidateMatchConfig's own doc comment), so by the time
// Apply can reject it, Reconcile has already run and the store has
// already swapped — there is no last-known-good config left at the store
// level to fall back to either.
//
// See 07-09-PLAN.md for the fix that closed 07-VERIFICATION.md gaps[0].
// See 07-10-PLAN.md for the fix that closed 07-VERIFICATION.md's
// 2026-08-08 gaps[0] (07-REVIEW.md's post-07-09 CR-01) — the D-07 cleanup
// had regressed to running only on the success path.
func (s *Supervisor) Apply(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	newCfg := s.cfgStore.Expanded()
	oldCfg := s.cfg
	// The operator's accepted keys follow the config they live in
	// (davison/topos#49): installed here so a saved key takes effect at
	// the launches this apply performs, and at the listing endpoints that
	// evaluate trust without launching.
	pluginhost.SetOperatorProvenanceKeys(pluginhost.OperatorProvenanceKeysFromConfig(newCfg.Plugins.TrustedKeys))

	s.stopScheduler()

	// WR-02 (08-REVIEW.md): exclude every currently suspended instance
	// name from what Reconcile is asked to launch. A suspended instance
	// (SuspendInstance, above) is already absent from s.host — its
	// subprocess was deliberately killed to make room for an in-flight
	// WhatsApp link/re-link session's own subprocess, which holds the
	// same data directory's exclusive store lock for as long as the
	// session lasts. Without this exclusion, an unrelated Apply landing
	// during that window would see the suspended instance still present
	// in newCfg.Sources (SuspendInstance never touches config-of-record,
	// only the launched set) and Reconcile would try to relaunch it,
	// losing the store-lock race against the live link subprocess and
	// failing this call's Reconcile outright — rejecting the ENTIRE
	// otherwise-valid save with 500 apply_failed for a reason that has
	// nothing to do with what the save actually changed. A suspended
	// instance's *pluginhost.Plugin value is kept (s.suspended) purely so
	// ValidateMatchConfigWithSuspended below can still validate its match
	// config without it being currently launched; SuspendInstance's own
	// resume closure is what relaunches it once the session ends, reading
	// s.cfg.Sources (which by then reflects whatever this Apply just
	// committed) fresh at that point.
	var reconcileSources map[string]config.Source
	if len(s.suspended) == 0 {
		reconcileSources = newCfg.Sources
	} else {
		reconcileSources = make(map[string]config.Source, len(newCfg.Sources))
		for name, src := range newCfg.Sources {
			if _, isSuspended := s.suspended[name]; isSuspended {
				continue
			}
			reconcileSources[name] = src
		}
	}

	if err := s.host.Reconcile(ctx, s.cfgStore.Raw(), reconcileSources, s.logger); err != nil {
		// Pre-Reconcile failure: Reconcile's own T-07-11 guarantee means the
		// previously running plugin set is genuinely untouched, so the OLD
		// generation is the consistent one here — this is the mirror image
		// of every branch below and must stay asymmetric with them. The
		// operator's keys follow the generation: the proposed set was
		// installed above for the launches this apply would have made,
		// so the OLD config's set is reinstalled here — trust must never
		// outlive the config it came from (davison/topos#49).
		pluginhost.SetOperatorProvenanceKeys(pluginhost.OperatorProvenanceKeysFromConfig(oldCfg.Plugins.TrustedKeys))
		s.startScheduler(oldCfg)
		return fmt.Errorf("supervisor: apply: %w", err)
	}

	// Reconcile has committed (gaps[0]'s corrected region): the D-07 cleanup
	// runs unconditionally here, textually and temporally BEFORE the
	// match-vocabulary check, so the check's outcome can only ADD to the
	// returned error and can never gate, shorten or skip the cleanup. By
	// this point the removed instances are already gone from the host
	// regardless of what the vocabulary check decides; gating the cleanup
	// on a later check would strand their rows permanently, because the
	// same call also advances s.cfg past the removal and destroys the diff
	// (removedInstances(oldCfg, newCfg)) that would ever detect them as
	// removed again. See 07-10-PLAN.md, closing 07-VERIFICATION.md gaps[0]
	// and 07-REVIEW.md's post-07-09 CR-01.
	cleanupErr := s.cleanupRemovedInstances(ctx, oldCfg, newCfg)

	// The synchronous purge (07-16-PLAN.md, closes 07-UAT.md G-07-7): for
	// every (webspace, instance) pair still present in both configs whose
	// participation flipped from true to false, clear that pair's
	// webspace_items rows now, before this Apply call answers its caller.
	// This is deliberately synchronous where the eager resync dispatched
	// near the end of this function is deliberately NOT (T-07-62's
	// prohibition): the purge is a pure local index write with no plugin
	// RPC, so it can run on the request path without coupling a config
	// save's latency to a plugin's reachability, whereas the resync below
	// performs a real Match RPC per webspace against a plugin subprocess.
	// Runs here, after cleanupRemovedInstances and before the
	// vocabulary check, for the identical reason the cleanup does: this
	// region runs unconditionally on every path through it, and by this
	// point Reconcile has already committed, so there is no later point
	// where skipping this would still be safe to retry.
	purgeErr := s.purgeDeparticipatedWebspaceRows(ctx, oldCfg, newCfg)

	// ValidateMatchConfigWithSuspended (WR-02, 08-REVIEW.md), not
	// ValidateMatchConfig: a suspended instance is temporarily absent from
	// s.host by design (see the reconcileSources comment above), but is
	// still fully configured — validating against s.host alone would
	// reject every webspace it participates in as "has no launched
	// plugin" for a reason unrelated to this save. suspendedPlugins
	// carries each suspended instance's already-cached Describe-learned
	// vocabulary (no live RPC) so validation still runs against it exactly
	// as if it were launched.
	suspendedPlugins := make([]*pluginhost.Plugin, 0, len(s.suspended))
	for _, p := range s.suspended {
		suspendedPlugins = append(suspendedPlugins, p)
	}
	validateErr := pluginhost.ValidateMatchConfigWithSuspended(newCfg, s.host, suspendedPlugins)

	// One shared commit site for the whole post-Reconcile region (07-09's
	// invariant, strengthened): Reconcile has already committed its
	// mutation in place and provides no undo, and the config store already
	// swapped to newCfg before Apply was ever called — the only
	// self-consistent state available is the new generation, whether this
	// region ends in success or in either kind of failure above.
	// Re-running Reconcile against oldCfg.Sources as a rollback is rejected
	// outright: a rollback that performs real subprocess launches can
	// itself fail, leaving the kernel strictly worse off than this defect
	// with no third recourse.
	s.commitGeneration(newCfg)

	// The vocabulary error leads (D-09: the operator-actionable message
	// must reach the UI verbatim and first), the cleanup error follows, and
	// the purge error follows that — errors.Join drops nils, so each
	// single-fault case still produces byte-identical text to before this
	// restructuring, and a genuine multi-fault case reports all of them.
	if err := errors.Join(validateErr, cleanupErr, purgeErr); err != nil {
		return fmt.Errorf("supervisor: apply: %w", err)
	}

	// A match-only edit (a webspace or match block changed, but an
	// instance's own connection config did not) has no relaunch of its own
	// to trigger an eager resync from. Dispatch one explicitly for every
	// instance whose config.Source is byte-identical to before, so it is
	// not left waiting for its own next scheduled tick. This is safe to
	// call even though the scheduler generation just started may also be
	// about to refresh the same instance — Coordinator.Refresh's
	// single-flight guarantee coalesces the two into one sync, never two.
	if !reflect.DeepEqual(oldCfg.Webspaces, newCfg.Webspaces) {
		// Read coord/genCtx/genWG together, right here, next to each other:
		// commitGeneration has already run by this point and installed a
		// NEW generation, so all three must be re-read now rather than
		// captured before the call — they are exactly the generation these
		// dispatched goroutines belong to (08-09-PLAN.md Task 2, closing
		// 08-UAT.md G-08-3's eager-resync sibling: before this fix these
		// goroutines ran on a detached context.Background(), untracked by
		// any generation, so a later stopScheduler — now reachable from the
		// WhatsApp link-start HTTP request path via SuspendInstance — could
		// never bound how long it waited on one).
		coord := s.coord
		genCtx := s.genCtx
		genWG := s.genWG
		for name, src := range newCfg.Sources {
			if _, isSuspended := s.suspended[name]; isSuspended {
				// Still suspended (WR-02): not currently launched, so
				// coord has no entry for it — dispatching would only
				// produce a discarded ErrUnknownSource from the
				// fire-and-forget goroutine below. Its own resume closure
				// (SuspendInstance) is what relaunches and syncs it once
				// the in-flight link session ends.
				continue
			}
			if oldSrc, ok := oldCfg.Sources[name]; ok && reflect.DeepEqual(oldSrc, src) {
				// A resync cancelled by the NEXT generation change is not
				// work lost: every fresh generation's Scheduler.Run fires
				// each configured source's first refresh immediately
				// anyway, so the cancellation costs at most a duplicate
				// that single-flight would have coalesced.
				genWG.Add(1)
				go func(name string) {
					defer genWG.Done()
					coord.Refresh(genCtx, name)
				}(name)
			}
		}
	}

	return nil
}

// purgeDeparticipatedWebspaceRows clears the webspace_items join rows for
// every (webspace, instance) pair whose participation flipped from true to
// false between oldCfg and newCfg — the D-07 answer at a finer identity
// than cleanupRemovedInstances' whole-instance removal (07-16-PLAN.md,
// closes 07-UAT.md G-07-7): a webspace narrowed to exclude a
// still-configured instance has that instance's items purged from ITS
// stream rows synchronously, before Apply returns, rather than waiting for
// a later scheduled sync to notice.
//
// Scope is deliberately the intersection of both configs, on both axes:
//
//   - Webspace names present in BOTH oldCfg.Webspaces and newCfg.Webspaces.
//     A webspace absent from newCfg is excluded: ReplaceWebspaceSourceItems
//     upserts the webspaces table row as part of the same transaction, so
//     clearing a deleted webspace's rows through it would leave the index
//     asserting that a deleted webspace exists — strictly worse than the
//     stale rows it removed. Orphaned rows for a deleted webspace are a
//     pre-existing, deliberately out-of-scope condition (07-16-PLAN.md
//     planning choice 8).
//   - Instance ids present in BOTH oldCfg.Sources and newCfg.Sources. An
//     instance absent from newCfg.Sources is excluded: cleanupRemovedInstances
//     (called earlier in this same post-Reconcile region — see Apply,
//     below) already deletes that instance's items rows, and the existing
//     ON DELETE CASCADE on webspace_items.item_id already clears it from
//     every webspace it participated in — purging it a second time here
//     adds a second failure mode for no behavioural gain.
//
// Participation is decided by asking correlate.ParticipatesIn against the
// OLD webspace and the NEW webspace for each pair, in that order — the
// same predicate the sync path (correlate.matchFieldsFor) applies, so the
// purge can never clear a pair a sync would keep, or leave one a sync
// would clear (T-07-65). Only a true-to-false flip clears rows; every
// other transition (false-to-false, false-to-true, true-to-true) is left
// alone — a false-to-true flip needs no clear at all (there is nothing
// stale to remove), and the next sync (this Apply's own eager resync
// dispatch below, or the instance's regular scheduled tick) populates it
// going forward.
//
// Iteration is over webspace names, then instance ids, both sorted —
// mirroring removedInstances' own convention — so a multi-fault error
// report is deterministic run to run. A per-pair clear failure is
// collected with the webspace and instance named, never returned early, so
// one failing pair cannot abandon the rest of the batch (mirroring
// cleanupRemovedInstances exactly); all collected failures are joined into
// the single error returned once the whole batch has been attempted.
//
// Issues no plugin RPC: ReplaceWebspaceSourceItems(ctx, ws, instance, nil)
// is a pure local index write, which is why this can run synchronously on
// the request path where an eager resync (a real Match RPC per webspace
// against a plugin subprocess) deliberately cannot (T-07-62's prohibition).
func (s *Supervisor) purgeDeparticipatedWebspaceRows(ctx context.Context, oldCfg, newCfg *config.Config) error {
	var webspaceNames []string
	for name := range oldCfg.Webspaces {
		if _, ok := newCfg.Webspaces[name]; ok {
			webspaceNames = append(webspaceNames, name)
		}
	}
	sort.Strings(webspaceNames)

	var instanceNames []string
	for name := range oldCfg.Sources {
		if _, ok := newCfg.Sources[name]; ok {
			instanceNames = append(instanceNames, name)
		}
	}
	sort.Strings(instanceNames)

	var failures []error
	for _, wsName := range webspaceNames {
		oldWS := oldCfg.Webspaces[wsName]
		newWS := newCfg.Webspaces[wsName]
		for _, instance := range instanceNames {
			wasParticipating := correlate.ParticipatesIn(oldWS, instance)
			nowParticipating := correlate.ParticipatesIn(newWS, instance)
			if wasParticipating && !nowParticipating {
				if err := s.idx.ReplaceWebspaceSourceItems(ctx, wsName, instance, nil); err != nil {
					failures = append(failures, fmt.Errorf("clear webspace %q source %q: %w", wsName, instance, err))
				}
			}
		}
	}
	return errors.Join(failures...)
}

// cleanupRemovedInstances performs the D-07 index cleanup for every
// instance named by removedInstances(oldCfg, newCfg) (already sorted, so
// its reporting order is deterministic run to run) — deleting that
// instance's items and sync_runs rows so a re-added instance under the
// same [sources.<id>] key can never inherit phantom history (T-07-13).
//
// Must only ever be called AFTER s.host.Reconcile has returned nil: before
// that point the instance's subprocess is still alive and the pre-Reconcile
// failure branch can still keep the old generation, so deleting rows there
// would destroy a still-configured, still-running source's data on a save
// that was never applied. The cleanup's correctness depends entirely on
// Reconcile having already committed.
//
// A per-instance DeleteSourceItems failure is NOT followed by that same
// instance's DeleteSyncRuns — that instance is already stranded (T-07-13
// needs both deletes), so attempting the second buys nothing and muddies
// the report. Every name is still attempted: a failure never returns early
// and never abandons any later-sorted instance in the batch. All collected
// failures are joined into the single error returned once the whole batch
// has been attempted — errors.Join returns nil when every element is nil
// or the slice is empty, so the zero-removed-instances case returns nil
// with no special-casing.
//
// The wrapping verbs and message text below are deliberately byte-identical
// to what Apply produced inline before this extraction, so an
// operator-visible message is unchanged by this restructuring. The caller
// supplies the "supervisor: apply: " prefix.
func (s *Supervisor) cleanupRemovedInstances(ctx context.Context, oldCfg, newCfg *config.Config) error {
	var failures []error
	for _, name := range removedInstances(oldCfg, newCfg) {
		if err := s.idx.DeleteSourceItems(ctx, name); err != nil {
			failures = append(failures, fmt.Errorf("delete items for removed source %q: %w", name, err))
			continue
		}
		if err := s.idx.DeleteSyncRuns(ctx, name); err != nil {
			failures = append(failures, fmt.Errorf("delete sync history for removed source %q: %w", name, err))
		}
	}
	return errors.Join(failures...)
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
