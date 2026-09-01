package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/index"
	"github.com/davison/topos/kernel/item"
)

func newSearchTestRouter(store *index.Store) http.Handler {
	return newSearchTestRouterWithConfig(store, &config.Config{})
}

// newSearchTestRouterWithConfig mirrors stream_test.go's
// newTestRouterWithConfig: SearchHandler now takes the same *config.Store
// shape StreamHandler does (Task 2), so a filter-bearing test wraps a
// hand-built *config.Config through config.NewStoreForTesting rather than
// loading a real TOML file.
func newSearchTestRouterWithConfig(store *index.Store, cfg *config.Config) http.Handler {
	r := chi.NewRouter()
	r.Get("/api/webspaces/{webspace}/search", SearchHandler(store, config.NewStoreForTesting(cfg), &fakeFetcher{}))
	return r
}

func seedSearchableItem(t *testing.T, store *index.Store, webspace, sourceID, title, preview string) item.Item {
	t.Helper()
	it := item.Item{
		ID: item.ID("paperless", sourceID), SourceType: "paperless", SourceID: sourceID,
		Title: title, Preview: preview,
		TimestampUnix: 100, SecondaryTimestampUnix: 100,
		Fidelity: item.FidelityExact, DeepLink: "http://paperless.lan:8000/documents/" + sourceID,
		Labels: []string{"House"}, Provenance: map[string]string{"source_type": "paperless"},
	}
	if err := store.ReplaceWebspaceSourceItems(context.Background(), webspace, "paperless", []item.Item{it}); err != nil {
		t.Fatalf("ReplaceWebspaceSourceItems: %v", err)
	}
	return it
}

func TestSearchHandler_UnknownWebspace404(t *testing.T) {
	store := newTestStoreForHTTP(t)
	router := newSearchTestRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/api/webspaces/does-not-exist/search?q=x", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
	var envelope errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal error envelope: %v", err)
	}
	if envelope.Error.Code != "webspace_not_found" {
		t.Errorf("expected code webspace_not_found, got %q", envelope.Error.Code)
	}
}

// TestSearchHandler_ConfigKnownNeverSyncedNoQReturns200EmptyResults
// (07-15-PLAN.md Task 1, G-07-1.missing[2]) proves SearchHandler asks the
// same config-aware existence gate StreamHandler does — a config-known,
// never-synced webspace answers 200, not 404, with no `q` supplied.
func TestSearchHandler_ConfigKnownNeverSyncedNoQReturns200EmptyResults(t *testing.T) {
	store := newTestStoreForHTTP(t)
	cfg := &config.Config{Webspaces: map[string]config.Webspace{"new-project": {}}}
	router := newSearchTestRouterWithConfig(store, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/webspaces/new-project/search", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for a config-known-never-synced webspace, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp searchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Results == nil {
		t.Error("expected results to be a non-nil empty array, not null")
	}
}

// TestSearchHandler_ConfigKnownNeverSyncedWithQReturns200EmptyResults is the
// mirror case with a non-empty `q` — the config half is consulted before
// any index query, so this never round-trips into a WebspaceExists lookup.
func TestSearchHandler_ConfigKnownNeverSyncedWithQReturns200EmptyResults(t *testing.T) {
	store := newTestStoreForHTTP(t)
	cfg := &config.Config{Webspaces: map[string]config.Webspace{"new-project": {}}}
	router := newSearchTestRouterWithConfig(store, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/webspaces/new-project/search?q=boiler", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for a config-known-never-synced webspace with a query, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp searchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Results) != 0 {
		t.Errorf("expected zero results, got %d", len(resp.Results))
	}
}

func TestSearchHandler_AbsentQReturns200EmptyResults(t *testing.T) {
	store := newTestStoreForHTTP(t)
	seedSearchableItem(t, store, "ws", "1", "Boiler invoice", "annual boiler service")
	router := newSearchTestRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/api/webspaces/ws/search", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp searchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Results == nil {
		t.Error("expected results to be an empty array, not null")
	}
	if len(resp.Results) != 0 {
		t.Errorf("expected zero results, got %d", len(resp.Results))
	}
	if resp.Query != "" {
		t.Errorf("expected empty query field, got %q", resp.Query)
	}
}

func TestSearchHandler_EmptyQReturns200EmptyResults(t *testing.T) {
	store := newTestStoreForHTTP(t)
	seedSearchableItem(t, store, "ws", "1", "Boiler invoice", "annual boiler service")
	router := newSearchTestRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/api/webspaces/ws/search?q=", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp searchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Results) != 0 {
		t.Errorf("expected zero results, got %d", len(resp.Results))
	}
}

func TestSearchHandler_WhitespaceQReturns200EmptyResults(t *testing.T) {
	store := newTestStoreForHTTP(t)
	seedSearchableItem(t, store, "ws", "1", "Boiler invoice", "annual boiler service")
	router := newSearchTestRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/api/webspaces/ws/search?q=%20%20%20", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp searchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Results) != 0 {
		t.Errorf("expected zero results, got %d", len(resp.Results))
	}
}

func TestSearchHandler_SingleQuoteQReturns200NotServerError(t *testing.T) {
	store := newTestStoreForHTTP(t)
	seedSearchableItem(t, store, "ws", "1", "Boiler invoice", "annual boiler service")
	router := newSearchTestRouter(store)

	req := httptest.NewRequest(http.MethodGet, `/api/webspaces/ws/search?q=%22`, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for a syntactically-hostile query, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp searchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Results) != 0 {
		t.Errorf("expected zero results for a bare double-quote query, got %d", len(resp.Results))
	}
}

func TestSearchHandler_ZeroMatches(t *testing.T) {
	store := newTestStoreForHTTP(t)
	seedSearchableItem(t, store, "ws", "1", "Boiler invoice", "annual boiler service")
	router := newSearchTestRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/api/webspaces/ws/search?q=nonexistentword", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp searchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Results) != 0 {
		t.Errorf("expected zero results, got %d", len(resp.Results))
	}
}

// TestSearchHandler_ReadsWebspaceFilterFromLiveConfig (07-01-PLAN.md Task
// 2, D-16/D-18) proves SearchHandler reads the webspace's saved filter
// from the live config store and AND-combines it with the live query: an
// item matching only the live query, with no saved filter term, is absent.
func TestSearchHandler_ReadsWebspaceFilterFromLiveConfig(t *testing.T) {
	store := newTestStoreForHTTP(t)
	both := seedSearchableItem(t, store, "ws", "1", "Boiler quote", "a boiler repair quote")
	seedSearchableItem(t, store, "ws", "2", "Garden fence quote", "replacing the back fence")

	cfg := &config.Config{
		Webspaces: map[string]config.Webspace{
			"ws": {Filter: []string{"boiler"}},
		},
	}
	router := newSearchTestRouterWithConfig(store, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/webspaces/ws/search?q=quote", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp searchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Results) != 1 || resp.Results[0].ID != both.ID {
		t.Fatalf("expected exactly the item matching both the saved filter and the live query, got %+v", resp.Results)
	}
}

func TestSearchHandler_OneMatchFlattenedShapeWithSnippet(t *testing.T) {
	store := newTestStoreForHTTP(t)
	it := seedSearchableItem(t, store, "ws", "1", "Boiler invoice", "annual boiler service")
	router := newSearchTestRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/api/webspaces/ws/search?q=boiler", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp searchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	got := resp.Results[0]
	if got.ID != it.ID {
		t.Errorf("expected id %q, got %q", it.ID, got.ID)
	}
	if got.Link.URL != it.DeepLink {
		t.Errorf("expected flattened stream-item fields present (link url), got %+v", got)
	}
	if got.Snippet == "" {
		t.Error("expected a non-empty snippet field")
	}
	if resp.Query != "boiler" {
		t.Errorf("expected query field to echo the request's q, got %q", resp.Query)
	}

	// Confirm the JSON is genuinely flattened (streamItem fields at the
	// top level of the result object, not nested under an "item" key).
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	var results []map[string]json.RawMessage
	if err := json.Unmarshal(raw["results"], &results); err != nil {
		t.Fatalf("unmarshal raw results: %v", err)
	}
	if _, ok := results[0]["id"]; !ok {
		t.Error("expected top-level 'id' key in the flattened result object")
	}
	if _, ok := results[0]["snippet"]; !ok {
		t.Error("expected top-level 'snippet' key in the flattened result object")
	}
}
