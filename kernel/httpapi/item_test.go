package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/davison/webspaces/kernel/index"
	"github.com/davison/webspaces/kernel/item"
	"github.com/davison/webspaces/kernel/pluginhost"
	webspacesv1 "github.com/davison/webspaces/sdk/gen/webspaces/v1"
)

// fakeFetcher is a test double satisfying httpapi.Fetcher without
// launching a real plugin subprocess.
type fakeFetcher struct {
	result pluginhost.FetchResult
	err    error
}

func (f *fakeFetcher) Fetch(_ context.Context, _, _ string, _ webspacesv1.ContentVariant) (pluginhost.FetchResult, error) {
	return f.result, f.err
}

func newTestItemRouter(store *index.Store, fetcher Fetcher) http.Handler {
	r := chi.NewRouter()
	r.Get("/api/items/{id}", ItemHandler(store, fetcher))
	r.Get("/api/items/{id}/content", ItemContentHandler(store, fetcher))
	r.Get("/api/items/{id}/thumbnail", ItemThumbnailHandler(store, fetcher))
	return r
}

func seedTestItem(t *testing.T, store *index.Store, it item.Item) {
	t.Helper()
	if err := store.ReplaceWebspaceSourceItems(context.Background(), "test-webspace", "paperless", []item.Item{it}); err != nil {
		t.Fatalf("seed item: %v", err)
	}
}

func testItem() item.Item {
	return item.Item{
		ID: "paperless:42", SourceType: "paperless", SourceID: "42",
		Title: "Completion statement", Fidelity: item.FidelityExact,
		DeepLink: "http://paperless.lan:8000/documents/42",
	}
}

func TestItemHandler_UnknownID404(t *testing.T) {
	store := newTestStoreForHTTP(t)
	router := newTestItemRouter(store, &fakeFetcher{})

	req := httptest.NewRequest(http.MethodGet, "/api/items/paperless:99999999", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
	var envelope errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal error envelope: %v", err)
	}
	if envelope.Error.Code != "item_not_found" {
		t.Errorf("expected code item_not_found, got %q", envelope.Error.Code)
	}
}

func TestItemHandler_SourceUnavailable502(t *testing.T) {
	store := newTestStoreForHTTP(t)
	seedTestItem(t, store, testItem())

	router := newTestItemRouter(store, &fakeFetcher{
		err: fmt.Errorf("%w: connection refused", pluginhost.ErrSourceUnavailable),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/items/paperless:42", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", rec.Code, rec.Body.String())
	}
	var envelope errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal error envelope: %v", err)
	}
	if envelope.Error.Code != "source_unavailable" {
		t.Errorf("expected code source_unavailable, got %q", envelope.Error.Code)
	}
}

func TestItemHandler_ReturnsLiveContentAndFidelity(t *testing.T) {
	store := newTestStoreForHTTP(t)
	seedTestItem(t, store, testItem())

	router := newTestItemRouter(store, &fakeFetcher{result: pluginhost.FetchResult{
		Available: true, Text: "full extracted text", MimeType: "application/pdf", SizeBytes: 1024,
	}})

	req := httptest.NewRequest(http.MethodGet, "/api/items/paperless:42", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp itemDetailResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Content.Text != "full extracted text" {
		t.Errorf("unexpected content.text: %q", resp.Content.Text)
	}
	if !resp.Content.Available {
		t.Error("expected content.available = true")
	}
	if resp.Content.Rendition == nil || resp.Content.Rendition.MimeType != "application/pdf" {
		t.Errorf("unexpected rendition: %+v", resp.Content.Rendition)
	}
	if resp.Item.Link.Fidelity != "exact" {
		t.Errorf("expected item.link.fidelity 'exact', got %q", resp.Item.Link.Fidelity)
	}
}

func TestItemHandler_UnavailableContentStillReturns200(t *testing.T) {
	store := newTestStoreForHTTP(t)
	seedTestItem(t, store, testItem())

	router := newTestItemRouter(store, &fakeFetcher{result: pluginhost.FetchResult{
		Available: false, UnavailableReason: "no previewable rendition", Text: "some extracted text",
	}})

	req := httptest.NewRequest(http.MethodGet, "/api/items/paperless:42", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 even when content is unavailable, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp itemDetailResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Content.Available {
		t.Error("expected content.available = false")
	}
	if resp.Content.Rendition != nil {
		t.Error("expected a nil rendition when content is unavailable")
	}
}

func TestItemHandler_PercentEncodedIDMatchesRaw(t *testing.T) {
	store := newTestStoreForHTTP(t)
	seedTestItem(t, store, testItem())

	router := newTestItemRouter(store, &fakeFetcher{result: pluginhost.FetchResult{Available: true, Text: "x"}})

	req1 := httptest.NewRequest(http.MethodGet, "/api/items/paperless:42", nil)
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)

	req2 := httptest.NewRequest(http.MethodGet, "/api/items/paperless%3A42", nil)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	if rec1.Code != http.StatusOK || rec2.Code != http.StatusOK {
		t.Fatalf("expected both to return 200: raw=%d encoded=%d", rec1.Code, rec2.Code)
	}
	if rec1.Body.String() != rec2.Body.String() {
		t.Errorf("expected identical response bodies for raw and percent-encoded id:\nraw=%s\nencoded=%s",
			rec1.Body.String(), rec2.Body.String())
	}
}

func TestItemContentHandler_SecurityHeadersOnAllowedMIME(t *testing.T) {
	store := newTestStoreForHTTP(t)
	seedTestItem(t, store, testItem())

	body := []byte("%PDF-1.4 fake rendition bytes")
	router := newTestItemRouter(store, &fakeFetcher{result: pluginhost.FetchResult{
		Available: true, MimeType: "application/pdf", SizeBytes: int64(len(body)),
		Body: io.NopCloser(bytes.NewReader(body)),
	}})

	req := httptest.NewRequest(http.MethodGet, "/api/items/paperless:42/content", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "application/pdf" {
		t.Errorf("unexpected Content-Type: %q", rec.Header().Get("Content-Type"))
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing X-Content-Type-Options: nosniff")
	}
	if !strings.Contains(rec.Header().Get("Content-Security-Policy"), "sandbox") {
		t.Errorf("expected a sandboxing CSP, got %q", rec.Header().Get("Content-Security-Policy"))
	}
	if rec.Header().Get("Content-Disposition") != "inline" {
		t.Errorf("unexpected Content-Disposition: %q", rec.Header().Get("Content-Disposition"))
	}
	if rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Errorf("unexpected Cache-Control: %q", rec.Header().Get("Cache-Control"))
	}
	if rec.Body.String() != string(body) {
		t.Error("response body does not match the fetched rendition bytes")
	}
}

func TestItemContentHandler_TextHTMLRenditionServedWithSecurityHeaders(t *testing.T) {
	store := newTestStoreForHTTP(t)
	seedTestItem(t, store, testItem())

	body := []byte("<h1>Decking</h1><p>sanitized page content</p>")
	router := newTestItemRouter(store, &fakeFetcher{result: pluginhost.FetchResult{
		Available: true, MimeType: "text/html", SizeBytes: int64(len(body)),
		Body: io.NopCloser(bytes.NewReader(body)),
	}})

	req := httptest.NewRequest(http.MethodGet, "/api/items/paperless:42/content", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "text/html" {
		t.Errorf("unexpected Content-Type: %q", rec.Header().Get("Content-Type"))
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing X-Content-Type-Options: nosniff")
	}
	if rec.Header().Get("Content-Disposition") != "inline" {
		t.Errorf("unexpected Content-Disposition: %q", rec.Header().Get("Content-Disposition"))
	}
	if !strings.Contains(rec.Header().Get("Content-Security-Policy"), "sandbox") {
		t.Errorf("expected a sandboxing CSP, got %q", rec.Header().Get("Content-Security-Policy"))
	}
	if rec.Body.String() != string(body) {
		t.Error("response body does not match the fetched rendition bytes")
	}
}

func TestItemContentHandler_DisallowedMIME415(t *testing.T) {
	store := newTestStoreForHTTP(t)
	seedTestItem(t, store, testItem())

	router := newTestItemRouter(store, &fakeFetcher{result: pluginhost.FetchResult{
		Available: true, MimeType: "application/x-not-allowlisted",
		Body: io.NopCloser(strings.NewReader("whatever")),
	}})

	req := httptest.NewRequest(http.MethodGet, "/api/items/paperless:42/content", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d: %s", rec.Code, rec.Body.String())
	}
	var envelope errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal error envelope: %v", err)
	}
	if envelope.Error.Code != "unsupported_rendition_type" {
		t.Errorf("expected code unsupported_rendition_type, got %q", envelope.Error.Code)
	}
}

func TestItemContentHandler_UnavailableRendition404(t *testing.T) {
	store := newTestStoreForHTTP(t)
	seedTestItem(t, store, testItem())

	router := newTestItemRouter(store, &fakeFetcher{result: pluginhost.FetchResult{Available: false}})

	req := httptest.NewRequest(http.MethodGet, "/api/items/paperless:42/content", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}
