package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/item"
	"github.com/davison/topos/kernel/pluginhost"
)

// fakeSearchingFetcher is a Fetcher that also fans out — what the
// kernel's Host is to the route.
type fakeSearchingFetcher struct {
	fakeFetcher
	outcomes []pluginhost.SourceSearchOutcome
	gotWS    config.Webspace
	gotQuery string
	gotReq   []string
	calls    int
}

func (f *fakeSearchingFetcher) SearchSources(_ context.Context, ws config.Webspace, query string, required []string) []pluginhost.SourceSearchOutcome {
	f.calls++
	f.gotWS, f.gotQuery, f.gotReq = ws, query, required
	return f.outcomes
}

func searchRouter(t *testing.T, cfg *config.Config, f Fetcher) (chi.Router, func(id string) bool) {
	t.Helper()
	store := newTestStoreForHTTP(t)
	r := chi.NewRouter()
	r.Get("/api/webspaces/{webspace}/search", SearchHandler(store, config.NewStoreForTesting(cfg), f))
	return r, func(id string) bool { _, ok, _ := store.GetItem(context.Background(), id); return ok }
}

func searchCfg() *config.Config {
	return &config.Config{
		Sources:   map[string]config.Source{"mail": {Plugin: "topos-plugin-x"}},
		Webspaces: map[string]config.Webspace{"house": {Keywords: []string{"boiler"}, Filter: []string{"invoice"}}},
	}
}

func get(t *testing.T, r http.Handler, path string) (searchResponse, int) {
	t.Helper()
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	var resp searchResponse
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v: %s", err, rec.Body.String())
		}
	}
	return resp, rec.Code
}

// TestSearchHandler_FanOutMergesSourceHitsAndReportsSources (M2-R2,
// davison/topos#50): scope=all asks the sources with the webspace's saved
// filter as required terms, merges by stable id, marks where each hit
// matched and whether it is indexed, and reports every instance's
// outcome; scope=index never asks.
func TestSearchHandler_FanOutMergesSourceHitsAndReportsSources(t *testing.T) {
	hit := item.Item{ID: "mail:m-1", Source: "mail", SourceType: "proton", SourceID: "m-1", Title: "Boiler quote", Preview: "…", TimestampUnix: 100}
	f := &fakeSearchingFetcher{outcomes: []pluginhost.SourceSearchOutcome{
		{Instance: "mail", DisplayName: "Mail", Status: pluginhost.SearchStatusOK, Hits: []pluginhost.SearchHit{{Item: hit, Snippet: "…the boiler…", MatchedIn: "body"}}, Note: "bodies", ElapsedMS: 12},
		{Instance: "chat", DisplayName: "Chat", Status: pluginhost.SearchStatusUnsupported},
		{Instance: "docs", DisplayName: "Docs", Status: pluginhost.SearchStatusTimeout, ElapsedMS: 5000},
	}}
	r, _ := searchRouter(t, searchCfg(), f)

	resp, code := get(t, r, "/api/webspaces/house/search?q=boiler")
	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if resp.Scope != "all" || f.calls != 1 {
		t.Fatalf("default scope must fan out once: scope=%q calls=%d", resp.Scope, f.calls)
	}
	if f.gotQuery != "boiler" || len(f.gotReq) != 1 || f.gotReq[0] != "invoice" {
		t.Errorf("the source must receive the live query and the saved filter as required terms: %q %v", f.gotQuery, f.gotReq)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected the one source hit, got %+v", resp.Results)
	}
	got := resp.Results[0]
	if got.ID != "mail:m-1" || got.Origin != "source" || got.Indexed || len(got.MatchedIn) != 1 || got.MatchedIn[0] != "body" || got.Snippet != "…the boiler…" {
		t.Errorf("source hit rendered wrongly: %+v", got)
	}
	for name, want := range map[string]string{"mail": "ok", "chat": "unsupported", "docs": "timeout"} {
		if resp.Sources[name].Status != want {
			t.Errorf("sources[%s] = %+v, want status %s", name, resp.Sources[name], want)
		}
	}
	if resp.Sources["mail"].Hits != 1 || resp.Sources["mail"].Note != "bodies" || resp.Sources["docs"].ElapsedMS != 5000 {
		t.Errorf("per-source facts lost: %+v", resp.Sources)
	}

	resp, _ = get(t, r, "/api/webspaces/house/search?q=boiler&scope=index")
	if f.calls != 1 || resp.Scope != "index" || resp.Sources != nil || len(resp.Results) != 0 {
		t.Errorf("scope=index must not fan out and reports no sources: calls=%d %+v", f.calls, resp)
	}
	if _, code := get(t, r, "/api/webspaces/house/search?q=boiler&scope=everything"); code != http.StatusBadRequest {
		t.Errorf("an unknown scope is a 400, got %d", code)
	}
}

// TestSearchHandler_WithoutASearcherIsIndexOnly: a Fetcher that does not
// fan out (every existing test fake) leaves the route exactly as it was.
func TestSearchHandler_WithoutASearcherIsIndexOnly(t *testing.T) {
	r, _ := searchRouter(t, searchCfg(), &fakeFetcher{})
	resp, code := get(t, r, "/api/webspaces/house/search?q=boiler")
	if code != http.StatusOK || resp.Sources != nil || resp.Scope != "all" {
		t.Errorf("expected an index-only answer with no sources map: %d %+v", code, resp)
	}
}

// TestSearchHandler_DateRangeFiltersSourceHits (M3-R1, #76; spec 21's
// catch): a source hit outside the effective date range is dropped at the
// merge — the fan-out cannot smuggle an out-of-range item past the range
// the index rows were already filtered by.
func TestSearchHandler_DateRangeFiltersSourceHits(t *testing.T) {
	inside := item.Item{ID: "mail:in", Source: "mail", SourceType: "proton", SourceID: "in", Title: "inside", TimestampUnix: 1704153600}
	outside := item.Item{ID: "mail:out", Source: "mail", SourceType: "proton", SourceID: "out", Title: "outside", TimestampUnix: 1704412800}
	f := &fakeSearchingFetcher{outcomes: []pluginhost.SourceSearchOutcome{{
		Instance: "mail", DisplayName: "Mail", Status: pluginhost.SearchStatusOK,
		Hits: []pluginhost.SearchHit{
			{Item: inside, MatchedIn: "body"},
			{Item: outside, MatchedIn: "body"},
		},
	}}}
	r, _ := searchRouter(t, searchCfg(), f)

	resp, code := get(t, r, "/api/webspaces/house/search?q=boiler&scope=all&from=2024-01-01&to=2024-01-03")
	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if len(resp.Results) != 1 || resp.Results[0].ID != "mail:in" {
		t.Fatalf("want only the in-range source hit, got %+v", resp.Results)
	}

	// A malformed param is refused by name, not silently ignored.
	_, code = get(t, r, "/api/webspaces/house/search?q=boiler&from=soon")
	if code != http.StatusBadRequest {
		t.Fatalf("malformed from: status %d, want 400", code)
	}
}
