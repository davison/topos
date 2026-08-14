package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/index"
	"github.com/davison/topos/kernel/item"
)

func newTestMarksRouter(store *index.Store, cfg *config.Config) http.Handler {
	r := chi.NewRouter()
	r.Post("/api/webspaces/{webspace}/marks", MarksHandler(store, config.NewStoreForTesting(cfg)))
	return r
}

func postMarks(t *testing.T, router http.Handler, webspace string, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/webspaces/"+webspace+"/marks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// TestMarksHandler_AddOneIDReturns200Changed1 is Task 1's handler
// behavior: POST with action=add and one id returns 200 and changed:1.
func TestMarksHandler_AddOneIDReturns200Changed1(t *testing.T) {
	store := newTestStoreForHTTP(t)
	ctx := context.Background()
	it := sampleItemForHTTPTest("1", 100)
	if err := store.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", []item.Item{it}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	cfg := &config.Config{Webspaces: map[string]config.Webspace{"ws": {}}}
	router := newTestMarksRouter(store, cfg)

	rec := postMarks(t, router, "ws", `{"kind":"excluded","action":"add","item_ids":["`+it.ID+`"]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp marksResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Changed != 1 {
		t.Errorf("expected changed=1, got %d", resp.Changed)
	}
	if resp.ExcludedCount != 1 {
		t.Errorf("expected excluded_count=1, got %d", resp.ExcludedCount)
	}
	if resp.SchemaVersion != schemaVersion {
		t.Errorf("expected schema_version %d, got %d", schemaVersion, resp.SchemaVersion)
	}
}

// TestMarksHandler_UnknownWebspace404 mirrors StreamHandler's own
// webspace_not_found envelope shape and status.
func TestMarksHandler_UnknownWebspace404(t *testing.T) {
	store := newTestStoreForHTTP(t)
	router := newTestMarksRouter(store, &config.Config{})

	rec := postMarks(t, router, "does-not-exist", `{"kind":"excluded","action":"add","item_ids":["paperless:1"]}`)
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

// TestMarksHandler_EmptyItemIDsReturns400InvalidRequest asserts an empty
// item_ids array is rejected with 400 invalid_request, never a silent 200
// no-op (edge: KERN-09 empty).
func TestMarksHandler_EmptyItemIDsReturns400InvalidRequest(t *testing.T) {
	store := newTestStoreForHTTP(t)
	cfg := &config.Config{Webspaces: map[string]config.Webspace{"ws": {}}}
	router := newTestMarksRouter(store, cfg)

	rec := postMarks(t, router, "ws", `{"kind":"excluded","action":"add","item_ids":[]}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	var envelope errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal error envelope: %v", err)
	}
	if envelope.Error.Code != "invalid_request" {
		t.Errorf("expected code invalid_request, got %q", envelope.Error.Code)
	}
}

// TestMarksHandler_AbsentItemIDsReturns400InvalidRequest is the same
// edge case as above with the field omitted entirely rather than an
// explicit empty array.
func TestMarksHandler_AbsentItemIDsReturns400InvalidRequest(t *testing.T) {
	store := newTestStoreForHTTP(t)
	cfg := &config.Config{Webspaces: map[string]config.Webspace{"ws": {}}}
	router := newTestMarksRouter(store, cfg)

	rec := postMarks(t, router, "ws", `{"kind":"excluded","action":"add"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	var envelope errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal error envelope: %v", err)
	}
	if envelope.Error.Code != "invalid_request" {
		t.Errorf("expected code invalid_request, got %q", envelope.Error.Code)
	}
}

// TestMarksHandler_RemoveUnmarkedIDReturns200Changed0 proves un-excluding
// an item that carries no mark is a no-op, not an error (KERN-10
// adjacency, Task 2's behavior).
func TestMarksHandler_RemoveUnmarkedIDReturns200Changed0(t *testing.T) {
	store := newTestStoreForHTTP(t)
	cfg := &config.Config{Webspaces: map[string]config.Webspace{"ws": {}}}
	router := newTestMarksRouter(store, cfg)

	rec := postMarks(t, router, "ws", `{"kind":"excluded","action":"remove","item_ids":["paperless:never-marked"]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp marksResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Changed != 0 {
		t.Errorf("expected changed=0 removing a mark that was never set, got %d", resp.Changed)
	}
}

// TestMarksHandler_BadKindReturns400InvalidRequest asserts a kind other
// than "excluded" is rejected 400 invalid_request.
func TestMarksHandler_BadKindReturns400InvalidRequest(t *testing.T) {
	store := newTestStoreForHTTP(t)
	cfg := &config.Config{Webspaces: map[string]config.Webspace{"ws": {}}}
	router := newTestMarksRouter(store, cfg)

	rec := postMarks(t, router, "ws", `{"kind":"pinned","action":"add","item_ids":["paperless:1"]}`)
	assertInvalidRequest(t, rec)
}

// TestMarksHandler_BadActionReturns400InvalidRequest asserts an action
// other than "add"/"remove" is rejected 400 invalid_request.
func TestMarksHandler_BadActionReturns400InvalidRequest(t *testing.T) {
	store := newTestStoreForHTTP(t)
	cfg := &config.Config{Webspaces: map[string]config.Webspace{"ws": {}}}
	router := newTestMarksRouter(store, cfg)

	rec := postMarks(t, router, "ws", `{"kind":"excluded","action":"toggle","item_ids":["paperless:1"]}`)
	assertInvalidRequest(t, rec)
}

// TestMarksHandler_BlankIDReturns400InvalidRequest asserts an id that is
// empty or whitespace-only is rejected 400 invalid_request.
func TestMarksHandler_BlankIDReturns400InvalidRequest(t *testing.T) {
	store := newTestStoreForHTTP(t)
	cfg := &config.Config{Webspaces: map[string]config.Webspace{"ws": {}}}
	router := newTestMarksRouter(store, cfg)

	rec := postMarks(t, router, "ws", `{"kind":"excluded","action":"add","item_ids":["paperless:1","   "]}`)
	assertInvalidRequest(t, rec)
}

// TestMarksHandler_OverCapReturns400InvalidRequest asserts a request
// carrying more than 1000 ids is rejected 400 invalid_request, and that
// exactly 1000 ids succeeds.
func TestMarksHandler_OverCapReturns400InvalidRequest(t *testing.T) {
	store := newTestStoreForHTTP(t)
	ctx := context.Background()
	cfg := &config.Config{Webspaces: map[string]config.Webspace{"ws": {}}}
	router := newTestMarksRouter(store, cfg)

	overCap := marksRequestBody(1001)
	rec := postMarks(t, router, "ws", overCap)
	assertInvalidRequest(t, rec)

	// Seed 1000 items so the at-cap request has something real to mark —
	// SetItemMarks accepts unindexed ids too, but seeding keeps this
	// assertion end to end.
	items := make([]item.Item, 1000)
	ids := make([]string, 1000)
	for i := range items {
		it := sampleItemForHTTPTest(itemSuffix(i), int64(i))
		items[i] = it
		ids[i] = it.ID
	}
	if err := store.ReplaceWebspaceSourceItems(ctx, "ws", "paperless", items); err != nil {
		t.Fatalf("seed 1000 items: %v", err)
	}

	atCap := `{"kind":"excluded","action":"add","item_ids":["` + strings.Join(ids, `","`) + `"]}`
	rec = postMarks(t, router, "ws", atCap)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for exactly 1000 ids, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp marksResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Changed != 1000 {
		t.Errorf("expected changed=1000 for an at-cap request, got %d", resp.Changed)
	}
}

func itemSuffix(i int) string {
	return fmt.Sprintf("item-%04d", i)
}

// marksRequestBody builds a valid-shaped marks request body carrying n
// synthetic (never-seeded) item ids — used only to exercise the over-cap
// rejection path, which must fire before the store is ever reached.
func marksRequestBody(n int) string {
	ids := make([]string, n)
	for i := range ids {
		ids[i] = "paperless:" + itemSuffix(i)
	}
	return `{"kind":"excluded","action":"add","item_ids":["` + strings.Join(ids, `","`) + `"]}`
}

func assertInvalidRequest(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	var envelope errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal error envelope: %v", err)
	}
	if envelope.Error.Code != "invalid_request" {
		t.Errorf("expected code invalid_request, got %q", envelope.Error.Code)
	}
}

func sampleItemForHTTPTest(sourceID string, ts int64) item.Item {
	return item.Item{
		ID:                     item.ID("paperless", sourceID),
		Source:                 "paperless",
		SourceType:             "paperless",
		SourceID:               sourceID,
		Title:                  "Doc " + sourceID,
		Preview:                "preview text",
		TimestampUnix:          ts,
		SecondaryTimestampUnix: ts,
		Fidelity:               item.FidelityExact,
		DeepLink:               "http://paperless.lan:8000/documents/" + sourceID,
		Labels:                 []string{"House"},
		Provenance:             map[string]string{"source_type": "paperless"},
	}
}
