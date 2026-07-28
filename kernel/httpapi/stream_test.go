package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/davison/webspaces/kernel/index"
	"github.com/davison/webspaces/kernel/item"
)

func newTestStoreForHTTP(t *testing.T) *index.Store {
	t.Helper()
	s, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func newTestRouter(store *index.Store) http.Handler {
	r := chi.NewRouter()
	r.Get("/api/webspaces/{webspace}/stream", StreamHandler(store))
	return r
}

func TestStreamHandler_UnknownWebspace404(t *testing.T) {
	store := newTestStoreForHTTP(t)
	router := newTestRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/api/webspaces/does-not-exist/stream", nil)
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
	if envelope.SchemaVersion != schemaVersion {
		t.Errorf("expected schema_version %d, got %d", schemaVersion, envelope.SchemaVersion)
	}
}

func TestStreamHandler_KnownEmptyWebspaceReturns200EmptyArray(t *testing.T) {
	store := newTestStoreForHTTP(t)
	ctx := context.Background()
	if err := store.ReplaceWebspaceSourceItems(ctx, "house-move", "paperless", nil); err != nil {
		t.Fatalf("ReplaceWebspaceItems: %v", err)
	}

	router := newTestRouter(store)
	req := httptest.NewRequest(http.MethodGet, "/api/webspaces/house-move/stream", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp streamResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal stream response: %v", err)
	}
	if resp.Items == nil {
		t.Error("expected items to be an empty array, not null")
	}
	if len(resp.Items) != 0 {
		t.Errorf("expected zero items, got %d", len(resp.Items))
	}
}

func TestStreamHandler_ReturnsItemsWithLinkAndProvenance(t *testing.T) {
	store := newTestStoreForHTTP(t)
	ctx := context.Background()

	it := item.Item{
		ID: "paperless:42", SourceType: "paperless", SourceID: "42",
		Title: "Completion statement", Preview: "some text",
		TimestampUnix: 1773532800, SecondaryTimestampUnix: 1773561234,
		Fidelity: item.FidelityExact, DeepLink: "http://paperless.lan:8000/documents/42",
		Labels: []string{"House"}, Provenance: map[string]string{"source_type": "paperless"},
	}
	if err := store.ReplaceWebspaceSourceItems(ctx, "house-move", "paperless", []item.Item{it}); err != nil {
		t.Fatalf("ReplaceWebspaceItems: %v", err)
	}

	router := newTestRouter(store)
	req := httptest.NewRequest(http.MethodGet, "/api/webspaces/house-move/stream", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp streamResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}
	got := resp.Items[0]
	if got.Link.Fidelity != "exact" {
		t.Errorf("expected fidelity 'exact', got %q", got.Link.Fidelity)
	}
	if got.Link.URL != it.DeepLink {
		t.Errorf("expected deep link %q, got %q", it.DeepLink, got.Link.URL)
	}
	if len(got.Preview) > 500 {
		t.Errorf("preview exceeds 500 runes: %d", len(got.Preview))
	}
}
