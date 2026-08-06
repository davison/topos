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

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/index"
	"github.com/davison/topos/kernel/item"
	"github.com/davison/topos/kernel/pluginhost"
	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

// fakeFetcher is a test double satisfying httpapi.Fetcher without
// launching a real plugin subprocess. gotSource records the last source
// argument Fetch was called with, so a test can assert callers pass the
// instance id (item.Item.Source), never the plugin kind (item.Item.SourceType).
type fakeFetcher struct {
	result pluginhost.FetchResult
	err    error

	gotSource string
}

func (f *fakeFetcher) Fetch(_ context.Context, source, _ string, _ toposv1.ContentVariant) (pluginhost.FetchResult, error) {
	f.gotSource = source
	return f.result, f.err
}

func newTestItemRouter(store *index.Store, fetcher Fetcher) http.Handler {
	return newTestItemRouterWithConfig(store, &config.Config{}, fetcher)
}

func newTestItemRouterWithConfig(store *index.Store, cfg *config.Config, fetcher Fetcher) http.Handler {
	r := chi.NewRouter()
	r.Get("/api/items/{id}", ItemHandler(store, cfg, fetcher))
	r.Get("/api/items/{id}/content", ItemContentHandler(store, fetcher))
	r.Get("/api/items/{id}/thumbnail", ItemThumbnailHandler(store, fetcher))
	return r
}

func seedTestItem(t *testing.T, store *index.Store, it item.Item) {
	t.Helper()
	source := it.Source
	if source == "" {
		source = it.SourceType
	}
	if err := store.ReplaceWebspaceSourceItems(context.Background(), "test-webspace", source, []item.Item{it}); err != nil {
		t.Fatalf("seed item: %v", err)
	}
}

func testItem() item.Item {
	return item.Item{
		ID: "paperless:42", Source: "paperless", SourceType: "paperless", SourceID: "42",
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

// TestItemHandler_FetchCalledWithInstanceIDNotPluginKind is the
// regression proof for D-08's Fetch rewire: seeds an item whose Source
// (instance id) and SourceType (plugin kind) deliberately differ, and
// asserts fetcher.Fetch is called with the instance id. Before this
// phase's identity split, ItemHandler called Fetch with it.SourceType —
// which resolves the wrong (or no) plugin once two instances of one
// plugin type exist, since pluginhost.Host.Fetch looks up its launched
// plugin by instance id, not by plugin kind.
func TestItemHandler_FetchCalledWithInstanceIDNotPluginKind(t *testing.T) {
	store := newTestStoreForHTTP(t)
	it := item.Item{
		ID: "home-email:1", Source: "home-email", SourceType: "proton", SourceID: "1",
		Title: "Home item", Fidelity: item.FidelityExact, DeepLink: "https://mail.proton.me/home/1",
	}
	seedTestItem(t, store, it)

	fetcher := &fakeFetcher{result: pluginhost.FetchResult{Available: true, Text: "x"}}
	router := newTestItemRouter(store, fetcher)

	req := httptest.NewRequest(http.MethodGet, "/api/items/home-email:1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if fetcher.gotSource != "home-email" {
		t.Errorf("expected Fetch called with the instance id 'home-email', got %q", fetcher.gotSource)
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

// TestItemContentHandler_TextHTMLRenditionServedWithSecurityHeaders proves
// a text/html rendition is now sanitized, wrapped and themed by the kernel
// (D-11) before being served: the plugin-supplied fragment must never
// reach the response byte-for-byte, and the fetched result must declare a
// recognised ContentShape or the kernel refuses to serve it.
func TestItemContentHandler_TextHTMLRenditionServedWithSecurityHeaders(t *testing.T) {
	store := newTestStoreForHTTP(t)
	seedTestItem(t, store, testItem())

	fragment := []byte("<h1>Decking</h1><p>unsanitized fragment content</p>")
	router := newTestItemRouter(store, &fakeFetcher{result: pluginhost.FetchResult{
		Available: true, MimeType: "text/html", SizeBytes: int64(len(fragment)),
		Body:         io.NopCloser(bytes.NewReader(fragment)),
		ContentShape: toposv1.ContentShape_CONTENT_SHAPE_MARKDOWN_HTML,
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
	// Regression test: default-src 'none' with no style-src directive
	// silently blocks the browser from applying ANY inline <style>
	// element — including the kernel's own composed rendition stylesheet
	// (kernel/httpapi/rendition.go), served correctly but never applied.
	// Caught by live UAT; see 02-01-SUMMARY.md.
	if !strings.Contains(rec.Header().Get("Content-Security-Policy"), "style-src 'unsafe-inline'") {
		t.Errorf("expected style-src 'unsafe-inline' in the CSP so an iframe document's own inline stylesheet is applied, got %q", rec.Header().Get("Content-Security-Policy"))
	}
	// Scripts must remain fully blocked regardless of the style-src
	// widening above — default-src 'none' with no script-src override
	// still denies script execution, and the sandbox directive (with no
	// allow-scripts token) is unchanged.
	if !strings.Contains(rec.Header().Get("Content-Security-Policy"), "default-src 'none'") {
		t.Errorf("expected default-src 'none' to remain in the CSP, got %q", rec.Header().Get("Content-Security-Policy"))
	}
	if strings.Contains(rec.Header().Get("Content-Security-Policy"), "script-src") {
		t.Errorf("expected no script-src override (scripts must stay blocked by default-src 'none'), got %q", rec.Header().Get("Content-Security-Policy"))
	}
	// The response is the SANITIZED, WRAPPED, THEMED document — never the
	// plugin's raw fragment byte-for-byte (D-11's whole point).
	if rec.Body.String() == string(fragment) {
		t.Error("expected the response to be the kernel-wrapped document, not the plugin's raw fragment")
	}
	if !strings.HasPrefix(rec.Body.String(), "<!doctype html>") {
		t.Errorf("expected the response to start with a doctype, got: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Decking") || !strings.Contains(rec.Body.String(), "unsanitized fragment content") {
		t.Errorf("expected the fragment's own visible text to survive sanitization, got: %s", rec.Body.String())
	}
}

// TestItemContentHandler_UnrecognisedContentShapeRefusedNoBody proves the
// kernel fails closed (T-05-16): a text/html rendition whose ContentShape
// is unspecified is refused with a distinct error code and no body — never
// served unsanitized.
func TestItemContentHandler_UnrecognisedContentShapeRefusedNoBody(t *testing.T) {
	store := newTestStoreForHTTP(t)
	seedTestItem(t, store, testItem())

	fragment := []byte("<p>hello</p>")
	router := newTestItemRouter(store, &fakeFetcher{result: pluginhost.FetchResult{
		Available: true, MimeType: "text/html", SizeBytes: int64(len(fragment)),
		Body: io.NopCloser(bytes.NewReader(fragment)),
		// ContentShape left at its zero value: CONTENT_SHAPE_UNSPECIFIED.
	}})

	req := httptest.NewRequest(http.MethodGet, "/api/items/paperless:42/content", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", rec.Code, rec.Body.String())
	}
	var envelope errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal error envelope: %v", err)
	}
	if envelope.Error.Code != "unsupported_content_shape" {
		t.Errorf("expected code unsupported_content_shape, got %q", envelope.Error.Code)
	}
	if strings.Contains(rec.Body.String(), "<p>hello</p>") {
		t.Error("expected no fragment bytes to leak into the error response")
	}
}

// TestItemContentHandler_PDFRenditionCSPUnaffectedByStyleSrcWidening proves
// the style-src 'unsafe-inline' widening (added for text/html renditions)
// is applied uniformly by the shared renditionHandler and does not change
// PDF rendition behavior: the same hardened header set (sandbox,
// default-src 'none', nosniff) still applies, and the widened style-src
// is present but inert for a PDF (which carries no inline stylesheet to
// apply) — a monotonic widening, not a behavior change, for this MIME
// type.
func TestItemContentHandler_PDFRenditionCSPUnaffectedByStyleSrcWidening(t *testing.T) {
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
	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "sandbox") || !strings.Contains(csp, "default-src 'none'") {
		t.Errorf("expected the PDF rendition's hardened CSP to be unchanged, got %q", csp)
	}
	if !strings.Contains(csp, "style-src 'unsafe-inline'") {
		t.Errorf("expected the shared CSP (including the new style-src) to apply to PDF renditions too, got %q", csp)
	}
	if rec.Body.String() != string(body) {
		t.Error("PDF rendition body must be served unchanged by the CSP widening")
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
