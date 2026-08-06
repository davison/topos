package httpapi

import (
	"net/http"
	"sort"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/index"
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
		if runs, err := store.LatestSyncRunPerSource(ctx); err == nil {
			lastSync = aggregateSyncStatus(runs)
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
				Keywords:  keywordsOrEmpty(cfg.Webspaces[name].Keywords),
				ItemCount: counts[name],
				LastSync:  lastSync,
			})
		}

		WriteJSON(w, http.StatusOK, resp)
	}
}

// keywordsOrEmpty normalises a webspace's Keywords fallback list for JSON
// serialisation: nil (a webspace that relies entirely on explicit match
// blocks, per D-01/D-02, legitimately has no keywords fallback) becomes an
// empty array rather than encoding/json's default `null`, so API consumers
// never need to special-case a null keywords field.
func keywordsOrEmpty(keywords []string) []string {
	if keywords == nil {
		return []string{}
	}
	return keywords
}
