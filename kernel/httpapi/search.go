package httpapi

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/davison/topos/kernel/index"
)

// searchResult embeds every field a stream item carries (via streamItem)
// and adds the highlighted snippet, so the SPA can render a search result
// with the same row component it renders a stream item with.
type searchResult struct {
	streamItem
	Snippet string `json:"snippet"`
}

type searchResponse struct {
	SchemaVersion int            `json:"schema_version"`
	Webspace      string         `json:"webspace"`
	Query         string         `json:"query"`
	Results       []searchResult `json:"results"`
}

// SearchHandler serves GET /api/webspaces/{webspace}/search?q= — a free
// function taking only the index store, exactly like StreamHandler, so
// this handler is structurally unable to reach a plugin (KERN-02). It
// reads only the already-correlated local index and never triggers a live
// source call.
func SearchHandler(store *index.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "webspace")
		ctx := r.Context()

		known, err := store.WebspaceExists(ctx, name)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		if !known {
			WriteError(w, http.StatusNotFound, "webspace_not_found", "webspace \""+name+"\" is not configured or has not been synced")
			return
		}

		q := r.URL.Query().Get("q")
		if strings.TrimSpace(q) == "" {
			WriteJSON(w, http.StatusOK, searchResponse{
				SchemaVersion: schemaVersion,
				Webspace:      name,
				Query:         "",
				Results:       []searchResult{},
			})
			return
		}

		results, err := store.Search(ctx, name, q)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		resp := searchResponse{
			SchemaVersion: schemaVersion,
			Webspace:      name,
			Query:         q,
			Results:       make([]searchResult, len(results)),
		}
		for i, res := range results {
			resp.Results[i] = toSearchResult(res)
		}

		WriteJSON(w, http.StatusOK, resp)
	}
}

func toSearchResult(res index.SearchResult) searchResult {
	return searchResult{
		streamItem: toStreamItem(res.Item),
		Snippet:    res.Snippet,
	}
}
