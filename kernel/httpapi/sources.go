package httpapi

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/davison/topos/kernel/config"
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
// SourceTypesByName extends this interface (02-04-PLAN.md Task 1):
// kernel/httpapi/agent.go's grant filtering needs the launched-plugin
// name-to-source_type mapping on every /agent/v1 request, and must reach
// it through the same no-RPC, already-cached path SourcesHandler's own
// merge relies on — never a live probe just to resolve a name.
type HealthProber interface {
	ProbeSources(ctx context.Context) []pluginhost.SourceHealth
	SourceTypesByName() map[string]string
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
	Name         string `json:"name"`
	SourceType   string `json:"source_type"`
	DisplayName  string `json:"display_name"`
	Reachable    bool   `json:"reachable"`
	Syncing      bool   `json:"syncing"`
	LastStatus   string `json:"last_status"`
	LastSyncUnix int64  `json:"last_sync_unix"`
	LastError    string `json:"last_error"`
}

type sourcesResponse struct {
	SchemaVersion int            `json:"schema_version"`
	Sources       []sourceStatus `json:"sources"`
}

// SourcesHandler serves GET /api/sources: a kernel-side merge of every
// launched plugin's live reachability (ProbeSources) with the kernel's
// own sync_runs history (LatestSyncRunPerSource, SyncingSourceTypes) —
// D-08. A plugin's own self-reported last-sync time and last error are
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

// sourceStatusesFrom builds one sourceStatus per launched plugin, sorted by
// name — the shared merge SourcesHandler (unfiltered) and
// kernel/httpapi/agent.go's agent sources route (grant-filtered on top of
// this) both build on. Factored out here (02-04-PLAN.md Task 1) so the
// agent route reuses the identical merge logic rather than reimplementing
// it against a restricted source set.
func sourceStatusesFrom(ctx context.Context, store *index.Store, prober HealthProber) ([]sourceStatus, error) {
	healths := prober.ProbeSources(ctx)

	runs, err := store.LatestSyncRunPerSource(ctx)
	if err != nil {
		return nil, err
	}
	syncing, err := store.SyncingSourceTypes(ctx)
	if err != nil {
		return nil, err
	}

	statuses := make([]sourceStatus, len(healths))
	for i, h := range healths {
		run := runs[h.SourceType] // zero value SyncRun ("" / 0) when no run has ever been recorded — the neutral "unknown" state
		statuses[i] = sourceStatus{
			Name:         h.Name,
			SourceType:   h.SourceType,
			DisplayName:  h.DisplayName,
			Reachable:    h.Reachable,
			Syncing:      syncing[h.SourceType],
			LastStatus:   run.Status,
			LastSyncUnix: run.FinishedUnix,
			LastError:    run.Error,
		}
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
func SourceRefreshHandler(cfg *config.Config, refresher Refresher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if _, ok := cfg.Sources[name]; !ok {
			WriteError(w, http.StatusNotFound, "source_not_found", "source \""+name+"\" was not found")
			return
		}

		result, err := refresher.Refresh(r.Context(), name)
		if err != nil {
			// The only error Refresher.Refresh returns is
			// syncer.ErrUnknownSource, which cannot happen here since name
			// was just validated against cfg.Sources — but if a caller's
			// Refresher implementation somehow disagrees with cfg, treat
			// it the same way rather than leaking a 500.
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
// row (keyed by source_type, as LatestSyncRunPerSource returns) into the
// single shared "sync" object used by the stream and webspace-list
// envelopes. Precedence: "error" if any source's latest run errored; else
// "running" if any source is still mid-sync; else "ok" if at least one
// run is recorded; else the zero value (nothing has ever synced). This is
// what stops a webspace whose only failing source returned nothing from
// ever rendering as merely empty — an aggregate that silently dropped a
// failing source with zero items would look identical to a healthy,
// genuinely-empty webspace.
func aggregateSyncStatus(runs map[string]index.SyncRun) syncStatus {
	if len(runs) == 0 {
		return syncStatus{}
	}

	sourceTypes := make([]string, 0, len(runs))
	for st := range runs {
		sourceTypes = append(sourceTypes, st)
	}
	sort.Strings(sourceTypes)

	var hasError, hasRunning, hasOK bool
	var newestFinished int64
	var errParts []string

	for _, st := range sourceTypes {
		run := runs[st]
		switch run.Status {
		case "error":
			hasError = true
			if run.Error != "" {
				errParts = append(errParts, st+": "+run.Error)
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
