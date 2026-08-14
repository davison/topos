package httpapi

import (
	"bytes"
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
