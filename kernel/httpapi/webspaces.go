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
// last sync status (from the index). cfgStore is resolved fresh as the
// first statement of the returned closure (07-02-PLAN.md Task 2 — closes
// the boot-time snapshot gap 07-01-PLAN.md deliberately left open here),
// so a webspace added or edited through PUT /api/config or
// POST /api/config/reload is visible on the very next request with no
// kernel restart.
//
// last_sync is computed PER WEBSPACE (08-UAT.md G-08-3), scoped to each
// entry's own participating sources via filterRunsByParticipation — the
// runs map itself is still read once, before the loop, exactly as before;
// only the aggregation moved inside it, so each webspace's own status
// never leaks a non-participating source's failure or success.
func WebspacesHandler(store *index.Store, cfgStore *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := cfgStore.Expanded()
		ctx := r.Context()

		counts, err := store.Webspaces(ctx)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		runs, err := store.LatestSyncRunPerSource(ctx)
		if err != nil {
			runs = nil
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
				LastSync:  aggregateSyncStatus(filterRunsByParticipation(runs, cfg, name)),
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
