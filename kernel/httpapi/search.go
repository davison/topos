package httpapi

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/index"
	"github.com/davison/topos/kernel/item"
	"github.com/davison/topos/kernel/pluginhost"
)

// Searcher is the fan-out the search route uses when its Fetcher also
// provides one (the kernel's pluginhost.Host does; test fakes need not).
// Without it the route is the index-only search it always was.
type Searcher interface {
	SearchSources(ctx context.Context, ws config.Webspace, query string, required []string) []pluginhost.SourceSearchOutcome
}

// searchResult is one row of GET /api/webspaces/{w}/search: a stream row
// plus, since M2-R2 (davison/topos#50), where it matched and where it
// came from.
type searchResult struct {
	streamItem
	Snippet string `json:"snippet"`
	// MatchedIn: "title"/"preview" from the index; "body"/"labels"/
	// "attachment" from a source; an item found by both lists both.
	MatchedIn []string `json:"matched_in"`
	// Origin: "index", "source" or "both".
	Origin string `json:"origin"`
	// Indexed is false for a source hit whose item is not (yet) in the
	// local index — rendered from the plugin's own Item fields.
	Indexed bool `json:"indexed"`
}

// sourceSearchStatus is the per-instance outcome of the fan-out — the
// CLOSED vocabulary ok | unsupported | timeout | error.
type sourceSearchStatus struct {
	Status    string `json:"status"`
	Hits      int    `json:"hits"`
	Truncated bool   `json:"truncated,omitempty"`
	Note      string `json:"note,omitempty"`
	Error     string `json:"error,omitempty"`
	ElapsedMS int64  `json:"elapsed_ms"`
}

type searchResponse struct {
	SchemaVersion int            `json:"schema_version"`
	Webspace      string         `json:"webspace"`
	Query         string         `json:"query"`
	Scope         string         `json:"scope"`
	Results       []searchResult `json:"results"`
	// Sources is absent for scope=index; for scope=all it names every
	// participating instance and how it answered.
	Sources map[string]sourceSearchStatus `json:"sources,omitempty"`
}

// SearchHandler serves GET /api/webspaces/{webspace}/search?q=…&scope=index|all.
//
// scope=index (the fast path, milliseconds): FTS over titles and previews,
// the search this route always was. scope=all (the default): that, plus
// the fan-out to every participating source that searches its own content
// — each under pluginhost.SearchBudget — merged by stable id. The UI asks
// index first, then all, so a slow source never delays the fast answer
// (the "progressive" delivery decided at #50, as two requests). Nothing a
// search returns is written to the index.
func SearchHandler(store *index.Store, cfgStore *config.Store, fetcher Fetcher) http.HandlerFunc {
	searcher, _ := fetcher.(Searcher)
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := cfgStore.Expanded()
		name := chi.URLParam(r, "webspace")
		ctx := r.Context()
		known, err := webspaceIsKnown(ctx, store, cfg, name)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		if !known {
			writeWebspaceNotFound(w, name)
			return
		}
		scope := r.URL.Query().Get("scope")
		if scope == "" {
			scope = "all"
		}
		if scope != "index" && scope != "all" {
			WriteError(w, http.StatusBadRequest, "invalid_request", "scope must be \"index\" or \"all\"")
			return
		}
		q := r.URL.Query().Get("q")
		if strings.TrimSpace(q) == "" {
			WriteJSON(w, http.StatusOK, searchResponse{SchemaVersion: schemaVersion, Webspace: name, Query: "", Scope: scope, Results: []searchResult{}})
			return
		}
		ws := cfg.Webspaces[name]
		results, err := store.Search(ctx, name, q, ws.Filter, ws.FilterBySource)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		terms := queryTerms(q)
		merged := make([]searchResult, 0, len(results))
		byID := make(map[string]int, len(results))
		for _, res := range results {
			row := searchResult{streamItem: toStreamItem(res.Item), Snippet: res.Snippet, MatchedIn: []string{indexMatchedIn(res.Item, terms)}, Origin: "index", Indexed: true}
			byID[res.Item.ID] = len(merged)
			merged = append(merged, row)
		}
		resp := searchResponse{SchemaVersion: schemaVersion, Webspace: name, Query: q, Scope: scope}
		if scope == "all" && searcher != nil {
			outcomes := searcher.SearchSources(ctx, ws, q, ws.Filter)
			resp.Sources = make(map[string]sourceSearchStatus, len(outcomes))
			for _, o := range outcomes {
				resp.Sources[o.Instance] = sourceSearchStatus{Status: o.Status, Hits: len(o.Hits), Truncated: o.Truncated, Note: o.Note, Error: o.Error, ElapsedMS: o.ElapsedMS}
				for _, hit := range o.Hits {
					if i, seen := byID[hit.Item.ID]; seen {
						merged[i].Origin = "both"
						if hit.MatchedIn != "" && !contains(merged[i].MatchedIn, hit.MatchedIn) {
							merged[i].MatchedIn = append(merged[i].MatchedIn, hit.MatchedIn)
						}
						if merged[i].Snippet == "" {
							merged[i].Snippet = hit.Snippet
						}
						continue
					}
					_, indexed, lookErr := store.GetItem(ctx, hit.Item.ID)
					if lookErr != nil {
						indexed = false
					}
					row := searchResult{streamItem: toStreamItem(hit.Item), Snippet: hit.Snippet, MatchedIn: []string{hit.MatchedIn}, Origin: "source", Indexed: indexed}
					if hit.MatchedIn == "" {
						row.MatchedIn = []string{}
					}
					byID[hit.Item.ID] = len(merged)
					merged = append(merged, row)
				}
			}
		}
		// Stable order: the index's ranked rows first, then source hits by
		// timestamp desc then id — never by which source answered first.
		sort.SliceStable(merged[len(results):], func(i, j int) bool {
			a, b := merged[len(results)+i], merged[len(results)+j]
			if a.TimestampUnix != b.TimestampUnix {
				return a.TimestampUnix > b.TimestampUnix
			}
			return a.ID < b.ID
		})
		resp.Results = merged
		WriteJSON(w, http.StatusOK, resp)
	}
}

func queryTerms(q string) []string {
	var out []string
	for _, f := range strings.Fields(strings.ToLower(q)) {
		if len(f) >= 2 {
			out = append(out, f)
		}
	}
	return out
}

// indexMatchedIn says which indexed field an FTS row matched — the title
// when every term appears there, otherwise the preview (FTS5 indexes only
// those two columns).
func indexMatchedIn(it item.Item, terms []string) string {
	t := strings.ToLower(it.Title)
	for _, term := range terms {
		if !strings.Contains(t, term) {
			return "preview"
		}
	}
	if len(terms) == 0 {
		return "preview"
	}
	return "title"
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}
