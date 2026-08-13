package httpapi

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/correlate"
	"github.com/davison/topos/kernel/index"
	"github.com/davison/topos/kernel/pluginhost"
	"github.com/davison/topos/kernel/syncer"
)

// HealthProber is the minimal live-reachability-probe surface
// SourcesHandler depends on. *pluginhost.Host satisfies this
// structurally. Kept as an interface (the same pattern item.go's Fetcher
// already establishes) so sources_test.go can exercise every merge branch
// with a fake, without launching real plugin subprocesses.
//
// The name-to-plugin-kind lookup method this interface previously
// required is gone (05-01-PLAN.md Task 2, D-08): kernel/httpapi/agent.go's
// grant filtering now keys directly on cfg.AgentReadGrantedNames() /
// item.Source (the instance id), so that indirection — and its
// pluginhost.Host-side implementation — is deleted alongside its last
// caller.
type HealthProber interface {
	ProbeSources(ctx context.Context) []pluginhost.SourceHealth
	// LaunchFailures returns every instance CURRENTLY refused at launch by
	// a soft, per-instance failure class (Phase 11: pin mismatch only,
	// pluginhost.LaunchFailurePinMismatch) — sourceStatusesFrom merges this
	// set into GET /api/sources so a configured source that never launched
	// still produces a real, named entry (11-RESEARCH.md Pitfall 1)
	// instead of vanishing from the response entirely. *pluginhost.Host
	// and *supervisor.Supervisor both satisfy this structurally.
	LaunchFailures() []pluginhost.LaunchFailure
}

// Refresher is the minimal sync-dispatch surface SourceRefreshHandler and
// SyncRefreshHandler depend on. *syncer.Coordinator satisfies this
// structurally.
type Refresher interface {
	Refresh(ctx context.Context, sourceName string) (syncer.RunResult, error)
	RefreshAll(ctx context.Context) []syncer.RunResult
}

// sourceStatus mirrors one entry of GET /api/sources's "sources" array.
type sourceStatus struct {
	Name        string `json:"name"`
	SourceType  string `json:"source_type"`
	DisplayName string `json:"display_name"`
	// Plugin is the launched instance's plugin BINARY name (e.g.
	// "topos-plugin-mock"), added 09-01-PLAN.md Task 3 so the SPA can
	// address GET /api/plugins/{plugin}/icon directly off this row —
	// never source_type (the plugin KIND) and never the instance id.
	Plugin string `json:"plugin"`
	// Tier is this instance's launch-time provenance ("trusted" or
	// "external", Phase 11 PLUG-06/07) — derived exclusively from which
	// configured directory the launched binary resolved from
	// (pluginhost.ResolveBinary), never from anything the plugin itself
	// declares (T-11-01). The SourceChip trust badge (11-UI-SPEC.md E2)
	// renders off this field alone.
	Tier string `json:"tier"`
	// PinnedHash is the SHA-256 this instance's external-tier binary is
	// pinned to in [plugins.pins] — populated for a healthy external-tier
	// entry (from the launch-time pin match, pluginhost.SourceHealth.
	// PinnedHash) as well as a pin-mismatched entry (from the
	// pluginhost.LaunchFailure record), so the chip menu's pinned-hash
	// footer (11-UI-SPEC.md E5) has this fact whether or not the source
	// is currently reachable. Always empty for a trusted-tier entry
	// (D-04: never pinned).
	PinnedHash string `json:"pinned_hash,omitempty"`
	// CurrentHash is the on-disk SHA-256 of a pin-mismatched instance's
	// binary — the value the operator would be re-pinning to if they
	// confirm the "Trust updated binary" action (11-UI-SPEC.md E4). Empty
	// except on a pin-mismatch entry.
	CurrentHash string `json:"current_hash,omitempty"`
	// LaunchFailure is the CLOSED-VOCABULARY reason this instance never
	// launched at all — today only "pin_mismatch"
	// (pluginhost.LaunchFailurePinMismatch) — or empty when the instance
	// did launch (whether or not it is currently reachable — see
	// Reachable/LastError for that). The SPA gates the re-pin remedial
	// action on THIS field alone, never on a last_error string match
	// (T-11-18): a copy edit to last_error can never enable or disable the
	// action.
	LaunchFailure string `json:"launch_failure,omitempty"`
	Reachable     bool   `json:"reachable"`
	Syncing       bool   `json:"syncing"`
	LastStatus    string `json:"last_status"`
	LastSyncUnix  int64  `json:"last_sync_unix"`
	LastError     string `json:"last_error"`
}

type sourcesResponse struct {
	SchemaVersion int            `json:"schema_version"`
	Sources       []sourceStatus `json:"sources"`
}

// SourcesHandler serves GET /api/sources: a kernel-side merge of every
// launched plugin's live reachability (ProbeSources) with the kernel's
// own sync_runs history (LatestSyncRunPerSource, SyncingSources) — D-08.
// One entry per source INSTANCE (never merged by plugin kind), sorted by
// instance id for a deterministic response order run to run. A plugin's
// own self-reported last-sync time and last error are
// deliberately never read here: last_status, last_sync_unix and
// last_error come exclusively from the kernel's own recorded sync_runs
// rows, so a plugin cannot report a rosier history than actually happened
// and turn its own chip green (A-PLUG-04). One plugin's probe failing
// never fails the whole response — it only makes that source's own
// reachable:false, never a 500.
func SourcesHandler(store *index.Store, prober HealthProber) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		statuses, err := sourceStatusesFrom(r.Context(), store, prober)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		WriteJSON(w, http.StatusOK, sourcesResponse{SchemaVersion: schemaVersion, Sources: statuses})
	}
}

// sourceStatusesFrom builds one sourceStatus per CONFIGURED source instance
// — one entry per instance whether it is actually launched (a probe result
// exists) or was refused at launch by a soft failure class (a LaunchFailure
// record exists, 11-RESEARCH.md Pitfall 1) — sorted by name. The shared
// merge SourcesHandler (unfiltered) and kernel/httpapi/agent.go's agent
// sources route (grant-filtered on top of this) both build on this.
// Factored out here (02-04-PLAN.md Task 1) so the agent route reuses the
// identical merge logic rather than reimplementing it against a restricted
// source set.
//
// The merge (11-03-PLAN.md Task 1): probe-derived entries are built first;
// then one synthesized entry is appended per LaunchFailure whose instance id
// is NOT already present among them. An instance can appear in both sets
// only in a narrow race (Reconcile launched it successfully after this
// failure was recorded) — the probe result always wins, since it describes
// what is actually running, so no instance ever produces two entries.
func sourceStatusesFrom(ctx context.Context, store *index.Store, prober HealthProber) ([]sourceStatus, error) {
	healths := prober.ProbeSources(ctx)
	failures := prober.LaunchFailures()

	runs, err := store.LatestSyncRunPerSource(ctx)
	if err != nil {
		return nil, err
	}
	syncing, err := store.SyncingSources(ctx)
	if err != nil {
		return nil, err
	}

	statuses := make([]sourceStatus, 0, len(healths)+len(failures))
	seen := make(map[string]bool, len(healths))
	for _, h := range healths {
		run := runs[h.Name] // zero value SyncRun ("" / 0) when no run has ever been recorded — the neutral "unknown" state
		statuses = append(statuses, sourceStatus{
			Name:         h.Name,
			SourceType:   h.SourceType,
			DisplayName:  h.DisplayName,
			Plugin:       h.Plugin,
			Tier:         string(h.Tier),
			PinnedHash:   h.PinnedHash,
			Reachable:    h.Reachable,
			Syncing:      syncing[h.Name],
			LastStatus:   run.Status,
			LastSyncUnix: run.FinishedUnix,
			LastError:    run.Error,
		})
		seen[h.Name] = true
	}

	for _, f := range failures {
		if seen[f.Instance] {
			// A probe result already exists for this instance — it is
			// actually running (a Reconcile launched it successfully after
			// this failure was recorded, or the prober's two calls raced
			// each other) — the probe result wins, since it describes what
			// is genuinely running right now. Never emit a second entry
			// for the same instance id.
			continue
		}
		run := runs[f.Instance]
		statuses = append(statuses, sourceStatus{
			Name: f.Instance,
			// SourceType is deliberately left empty: a launch-failed
			// instance's Describe RPC never ran (there is no live
			// subprocess to call it on), so the kernel never learned a
			// source_type for it — and T-01-07 forbids trusting the
			// binary's filename for that fact instead.
			DisplayName:   f.DisplayName,
			Plugin:        f.Plugin,
			Tier:          string(f.Tier),
			PinnedHash:    f.PinnedHash,
			CurrentHash:   f.CurrentHash,
			LaunchFailure: f.Reason,
			Reachable:     false,
			Syncing:       syncing[f.Instance],
			LastStatus:    run.Status,
			LastSyncUnix:  run.FinishedUnix,
			// LastError carries the kernel's own named failure message
			// (instance/binary/pinned-vs-current hash), never the
			// recorded sync_runs row's error — a source that never
			// launched has no sync history of its own to report.
			LastError: f.Message,
		})
	}

	sort.Slice(statuses, func(i, j int) bool { return statuses[i].Name < statuses[j].Name })

	return statuses, nil
}

// runStatus is the JSON shape of one source's outcome from a manual
// refresh — POST /api/sources/{name}/refresh and POST /api/sync.
type runStatus struct {
	Name         string `json:"name"`
	SourceType   string `json:"source_type"`
	Status       string `json:"status"`
	ItemCount    int    `json:"item_count"`
	Error        string `json:"error"`
	Coalesced    bool   `json:"coalesced"`
	FinishedUnix int64  `json:"finished_unix"`
}

func toRunStatus(r syncer.RunResult) runStatus {
	return runStatus{
		Name:         r.Source,
		SourceType:   r.SourceType,
		Status:       r.Status,
		ItemCount:    r.ItemCount,
		Error:        r.Error,
		Coalesced:    r.Coalesced,
		FinishedUnix: r.FinishedUnix,
	}
}

type sourceRefreshResponse struct {
	SchemaVersion int       `json:"schema_version"`
	Source        runStatus `json:"source"`
}

// SourceRefreshHandler serves POST /api/sources/{name}/refresh. {name} is
// validated against the configured source set BEFORE dispatch — an
// unconfigured name returns 404 source_not_found in the identical
// envelope shape every other not-found route uses, and the message names
// only the requested value, never the configured set (T-02-09): this
// route must not be usable to enumerate which source names exist.
// cfgStore is resolved fresh as the first statement of the returned
// closure (07-02-PLAN.md Task 2) so a source added by a save is
// refreshable through this route immediately, with no kernel restart —
// previously a source added after Router construction could never be
// found here, since {name} was checked against a config snapshot frozen
// at boot.
func SourceRefreshHandler(cfgStore *config.Store, refresher Refresher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := cfgStore.Expanded()
		name := chi.URLParam(r, "name")
		if _, ok := cfg.Sources[name]; !ok {
			WriteError(w, http.StatusNotFound, "source_not_found", "source \""+name+"\" was not found")
			return
		}

		result, err := refresher.Refresh(r.Context(), name)
		if err != nil {
			// syncer.ErrUnknownSource is now reachable here despite the
			// validation above (08-09-PLAN.md, landing in this same wave):
			// a source instance suspended for an in-flight WhatsApp link
			// session stays present in cfg.Sources but is deliberately
			// absent from the Coordinator, so a refresh request racing that
			// suspension window hits this branch legitimately, not only a
			// caller whose Refresher implementation disagrees with cfg. The
			// existing 404 source_not_found envelope is deliberately reused
			// for it — no new error code, no contract change — since from
			// this route's caller the two cases ("never existed" and
			// "temporarily unavailable") both mean "not refreshable right
			// now."
			WriteError(w, http.StatusNotFound, "source_not_found", "source \""+name+"\" was not found")
			return
		}

		WriteJSON(w, http.StatusOK, sourceRefreshResponse{SchemaVersion: schemaVersion, Source: toRunStatus(result)})
	}
}

type syncRefreshResponse struct {
	SchemaVersion int         `json:"schema_version"`
	Sources       []runStatus `json:"sources"`
}

// SyncRefreshHandler serves POST /api/sync: refreshes every configured
// source through the same coordinator entry point the scheduler and the
// per-source refresh route use, and returns one status per source.
func SyncRefreshHandler(refresher Refresher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		results := refresher.RefreshAll(r.Context())
		statuses := make([]runStatus, len(results))
		for i, res := range results {
			statuses[i] = toRunStatus(res)
		}
		WriteJSON(w, http.StatusOK, syncRefreshResponse{SchemaVersion: schemaVersion, Sources: statuses})
	}
}

// aggregateSyncStatus merges every configured source's latest sync_runs
// row (keyed by source INSTANCE id, as LatestSyncRunPerSource returns —
// D-08) into the single shared "sync" object used by the stream and
// webspace-list envelopes. Precedence: "error" if any source's latest run
// errored; else "running" if any source is still mid-sync; else "ok" if at
// least one run is recorded; else the zero value (nothing has ever
// synced). This is what stops a webspace whose only failing source
// returned nothing from ever rendering as merely empty — an aggregate that
// silently dropped a failing source with zero items would look identical
// to a healthy, genuinely-empty webspace. errParts names each failing
// entry by its instance id, never by the shared plugin kind two instances
// might have in common.
func aggregateSyncStatus(runs map[string]index.SyncRun) syncStatus {
	if len(runs) == 0 {
		return syncStatus{}
	}

	sources := make([]string, 0, len(runs))
	for source := range runs {
		sources = append(sources, source)
	}
	sort.Strings(sources)

	var hasError, hasRunning, hasOK bool
	var newestFinished int64
	var errParts []string

	for _, source := range sources {
		run := runs[source]
		switch run.Status {
		case "error":
			hasError = true
			if run.Error != "" {
				errParts = append(errParts, source+": "+run.Error)
			}
		case "running":
			hasRunning = true
		case "ok":
			hasOK = true
		}
		if run.FinishedUnix > newestFinished {
			newestFinished = run.FinishedUnix
		}
	}

	status := ""
	switch {
	case hasError:
		status = "error"
	case hasRunning:
		status = "running"
	case hasOK:
		status = "ok"
	}

	return syncStatus{Status: status, FinishedUnix: newestFinished, Error: strings.Join(errParts, "; ")}
}

// filterRunsByParticipation restricts a LatestSyncRunPerSource-shaped map
// (keyed by source INSTANCE id) to the subset that actually feeds
// webspace, mirroring agent.go's filterRunsByGrant shape but scoping by
// participation rather than by agent grant. A run's source qualifies only
// when BOTH:
//
//  1. It is still a key of cfg.Sources — a run row can outlive its
//     instance's removal from config, and a removed instance obviously
//     cannot participate in anything.
//  2. correlate.ParticipatesIn(cfg.Webspaces[webspace], source) is true —
//     the exact predicate the sync path itself applies via
//     correlate.matchFieldsFor, covering both the allowlist gate (D-03)
//     and the has-match-input rule (D-20). Scoping with anything else
//     would let this aggregate report a status for a (webspace, source)
//     pair no sync would ever run for, or drop one it would.
//
// A webspace known only from surviving index rows (07-15-PLAN.md), with
// no `[webspaces.*]` block left in config, resolves cfg.Webspaces[webspace]
// to the zero value — ParticipatesIn is false for every source against
// it, so this correctly returns an empty map (and therefore the zero-value
// sync object): an unconfigured webspace cannot sync at all, so reporting
// no participants is honest, not a bug.
//
// This exists because a webspace's reported sync status must describe the
// sources that actually feed it (08-UAT.md G-08-3): the client treats a
// failing status with zero items as a statement about THIS webspace, and
// before this scoping any configured source's failure — participating or
// not — leaked into every webspace's aggregate.
func filterRunsByParticipation(runs map[string]index.SyncRun, cfg *config.Config, webspace string) map[string]index.SyncRun {
	ws := cfg.Webspaces[webspace]
	out := make(map[string]index.SyncRun, len(runs))
	for source, run := range runs {
		if _, ok := cfg.Sources[source]; !ok {
			continue
		}
		if !correlate.ParticipatesIn(ws, source) {
			continue
		}
		out[source] = run
	}
	return out
}
