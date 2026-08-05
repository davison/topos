// Package syncer implements the kernel's per-source sync coordinator
// (KERN-04): the single entry point every sync in the system — the
// scheduler, the manual-refresh HTTP routes, and the CLI — must go
// through. It owns the sync_runs two-phase write and the single-flight
// guarantee that one source never runs two syncs concurrently.
//
// Named "syncer", not "kernel/sync" as 02-PATTERNS.md's analog search
// suggested: this package already needs the standard library's own "sync"
// package (sync.WaitGroup in scheduler.go) alongside
// golang.org/x/sync/singleflight, and Phases 3-5 add exactly the kind of
// concurrent plugin code (IMAP IDLE, WhatsApp's WebSocket) that would also
// want the standard library's sync — a package literally named "sync"
// would force every one of those files to alias one import or the other.
// Recorded as a deviation in 02-02-SUMMARY.md.
package syncer

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/davison/topos/kernel/correlate"
	"github.com/davison/topos/kernel/index"
)

// ErrUnknownSource is returned by Refresh when sourceName is not a
// configured source (no config-key match among the sources this
// Coordinator was built with).
var ErrUnknownSource = errors.New("syncer: unknown source")

// finishRunTimeout bounds the detached sync_runs finalize write in
// syncOne. It is deliberately short: the write is a single indexed UPDATE
// against a local SQLite file, so anything approaching this budget means
// the DB is wedged, and blocking shutdown longer would not help.
const finishRunTimeout = 5 * time.Second

// RunResult is the caller-facing outcome of one Refresh call — a summary
// of a sync run that may have coalesced into an already in-flight one.
type RunResult struct {
	Source       string // config name (correlate.Source.Name())
	SourceType   string
	Status       string // "ok" | "error"
	ItemCount    int
	Error        string
	Coalesced    bool // true when this call joined an already-in-flight run rather than starting a fresh one
	StartedUnix  int64
	FinishedUnix int64
}

// Coordinator is the single source of truth for sync and health state
// that every plugin added in Phases 3-5 inherits by being a configured
// source (D-06). Refresh is single-flight per source name: a call
// arriving while that source is already syncing coalesces into the
// in-flight run and reports its outcome — it is never queued behind it
// and never runs a second concurrent sync for that source. There is
// deliberately no "already syncing, try later" rejection path: coalescing
// is the answer, so a refresh button in the UI never lies about whether a
// refresh happened.
type Coordinator struct {
	store   *index.Store
	engine  *correlate.Engine
	sources map[string]correlate.Source
	group   singleflight.Group
}

// NewCoordinator builds a Coordinator over sources, keyed by each source's
// own Name() (the config key under [sources.<name>] — see
// kernel/config.Source and kernel/pluginhost.Plugin.Name).
func NewCoordinator(store *index.Store, engine *correlate.Engine, sources []correlate.Source) *Coordinator {
	byName := make(map[string]correlate.Source, len(sources))
	for _, s := range sources {
		byName[s.Name()] = s
	}
	return &Coordinator{store: store, engine: engine, sources: byName}
}

// Refresh syncs the named source through the one coordinator entry point.
// A concurrent call for the same source coalesces into the same in-flight
// sync cycle rather than queuing a second one or being rejected; the
// returned RunResult.Coalesced reports which callers joined an in-flight
// run rather than triggering a fresh one. A source-level failure — a
// Match error, a persistence error, or an item rejected at the
// correlation boundary — is reported as data on the returned RunResult;
// it never surfaces as a non-nil error here. The only error Refresh
// itself returns is ErrUnknownSource, for a sourceName this Coordinator
// was not built with.
func (c *Coordinator) Refresh(ctx context.Context, sourceName string) (RunResult, error) {
	src, ok := c.sources[sourceName]
	if !ok {
		return RunResult{}, ErrUnknownSource
	}

	// c.group.Do's own error return is always nil here: syncOne never
	// returns a non-nil error to the closure — every sync-level failure is
	// captured as data on the RunResult instead, so a coalesced caller
	// never sees a stale or unrelated Go error from whichever goroutine
	// happened to run the shared closure.
	v, _, shared := c.group.Do(sourceName, func() (interface{}, error) {
		return c.syncOne(ctx, src), nil
	})

	result := v.(RunResult)
	result.Coalesced = shared
	return result, nil
}

// RefreshAll syncs every configured source, in a deterministic (sorted by
// config name) order, one Refresh call each. One source's failure never
// prevents another source's result from being collected.
func (c *Coordinator) RefreshAll(ctx context.Context) []RunResult {
	names := make([]string, 0, len(c.sources))
	for name := range c.sources {
		names = append(names, name)
	}
	sort.Strings(names)

	results := make([]RunResult, 0, len(names))
	for _, name := range names {
		// Refresh only returns a non-nil error for a name absent from
		// c.sources, which cannot happen here since names is derived from
		// c.sources itself.
		result, _ := c.Refresh(ctx, name)
		results = append(results, result)
	}
	return results
}

// syncOne runs one full sync cycle for src, recording the two-phase
// sync_runs write around the call to correlate.Engine.SyncSource — this
// is the only place in the kernel that calls StartSyncRun/FinishSyncRun,
// which is what makes the Coordinator the single source of truth for "is
// this source syncing right now" (D-06, A-PLUG-04).
func (c *Coordinator) syncOne(ctx context.Context, src correlate.Source) RunResult {
	sourceType := src.SourceType()
	started := time.Now().Unix()

	runID, err := c.store.StartSyncRun(ctx, sourceType)
	if err != nil {
		// A sync_runs write failure is itself a result the caller needs to
		// see, not a silently swallowed error.
		return RunResult{Source: src.Name(), SourceType: sourceType, Status: "error", Error: fmt.Sprintf("start sync run: %v", err), StartedUnix: started}
	}

	results, rejections := c.engine.SyncSource(ctx, src)

	totalItems := 0
	var firstErr error
	for _, r := range results {
		if r.Err != nil {
			if firstErr == nil {
				firstErr = r.Err
			}
			continue
		}
		totalItems += r.ItemCount
	}

	status := "ok"
	errMsg := ""
	switch {
	case firstErr != nil:
		status = "error"
		errMsg = firstErr.Error()
	case rejections != "":
		// The sync itself succeeded (other items from this source
		// persisted normally) but these specific items were rejected at
		// the correlation boundary — recorded, not silently dropped.
		errMsg = rejections
	}

	finished := time.Now().Unix()

	// Finalize on a context detached from ctx's cancellation. The run's
	// outcome has ALREADY happened by this point, so recording it must
	// never be skipped merely because the context that triggered the sync
	// was cancelled in the meantime. Passing ctx straight through here was
	// a latent orphan bug: when the scheduler's ctx is cancelled at
	// shutdown (runServe's cancel()) while a source is mid-sync — or when
	// the HTTP client behind a manual refresh disconnects — database/sql
	// rejects this UPDATE outright with "context canceled", leaving the
	// sync_runs row permanently at status "running". Because nothing
	// finalizes it afterwards, that source's syncing indicator stays
	// pinned on forever, across restarts. Proton was the source that hit
	// this in practice: as the slowest source (IMAP over the network to
	// Bridge) it has by far the widest window to be mid-sync at shutdown.
	// The timeout keeps a detached write from hanging shutdown.
	finishCtx, cancelFinish := context.WithTimeout(context.WithoutCancel(ctx), finishRunTimeout)
	defer cancelFinish()

	if err := c.store.FinishSyncRun(finishCtx, runID, status, errMsg, totalItems); err != nil {
		// A failure to record the finished run is worse than a normal
		// sync error — surface it in preference to the sync's own outcome.
		return RunResult{Source: src.Name(), SourceType: sourceType, Status: "error", Error: fmt.Sprintf("finish sync run: %v", err), StartedUnix: started, FinishedUnix: finished}
	}

	return RunResult{
		Source:       src.Name(),
		SourceType:   sourceType,
		Status:       status,
		ItemCount:    totalItems,
		Error:        errMsg,
		StartedUnix:  started,
		FinishedUnix: finished,
	}
}
