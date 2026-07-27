package httpapi

import (
	"net/http"
	"sort"

	"github.com/darrendavison/webspaces/kernel/config"
	"github.com/darrendavison/webspaces/kernel/index"
)

type webspaceSummary struct {
	Name      string     `json:"name"`
	Keywords  []string   `json:"keywords"`
	ItemCount int        `json:"item_count"`
	LastSync  syncStatus `json:"last_sync"`
}

type webspacesResponse struct {
	SchemaVersion int               `json:"schema_version"`
	Webspaces     []webspaceSummary `json:"webspaces"`
}

// WebspacesHandler serves GET /api/webspaces: the configured webspace list
// (name + keywords, from config) enriched with the current item count and
// last sync status (from the index).
func WebspacesHandler(store *index.Store, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		counts, err := store.Webspaces(ctx)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		var lastSync syncStatus
		if run, ok, err := store.LatestSyncRun(ctx); err == nil && ok {
			lastSync = syncStatus{Status: run.Status, FinishedUnix: run.FinishedUnix, Error: run.Error}
		}

		names := make([]string, 0, len(cfg.Webspaces))
		for name := range cfg.Webspaces {
			names = append(names, name)
		}
		sort.Strings(names)

		resp := webspacesResponse{SchemaVersion: schemaVersion, Webspaces: make([]webspaceSummary, 0, len(names))}
		for _, name := range names {
			resp.Webspaces = append(resp.Webspaces, webspaceSummary{
				Name:      name,
				Keywords:  cfg.Webspaces[name].Keywords,
				ItemCount: counts[name],
				LastSync:  lastSync,
			})
		}

		WriteJSON(w, http.StatusOK, resp)
	}
}
