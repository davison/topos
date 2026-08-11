package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// fakeIcon is one fake plugin's stored icon bytes+mime.
type fakeIcon struct {
	bytes []byte
	mime  string
}

// fakePluginIconProvider is a test double satisfying httpapi.PluginIconProvider
// without launching a real plugin subprocess (mirrors fakeFetcher/fakeProber's
// own shape in this package). calledWith records every binary name passed to
// PluginIcon, so a test can assert the handler's path-traversal guard rejects
// a malicious {plugin} value BEFORE ever calling the provider.
type fakePluginIconProvider struct {
	icons      map[string]fakeIcon
	calledWith []string
}

func (f *fakePluginIconProvider) PluginIcon(binary string) ([]byte, string, bool) {
	f.calledWith = append(f.calledWith, binary)
	if f.icons == nil {
		return nil, "", false
	}
	v, ok := f.icons[binary]
	if !ok {
		return nil, "", false
	}
	return v.bytes, v.mime, true
}

func newTestPluginIconRouter(icons PluginIconProvider) http.Handler {
	r := chi.NewRouter()
	r.Get("/api/plugins/{plugin}/icon", PluginIconHandler(icons))
	return r
}

// TestPluginIconHandler_ServesIconWithHardenedHeaders proves the full
// success path: exact bytes, the declared Content-Type, an immutable
// Cache-Control, an ETag, and the two hardening headers (T-09-01/T-09-04)
// (09-01-PLAN.md Task 2).
func TestPluginIconHandler_ServesIconWithHardenedHeaders(t *testing.T) {
	want := []byte("<svg>mock</svg>")
	provider := &fakePluginIconProvider{icons: map[string]fakeIcon{
		"topos-plugin-mock": {bytes: want, mime: "image/svg+xml"},
	}}
	router := newTestPluginIconRouter(provider)

	req := httptest.NewRequest(http.MethodGet, "/api/plugins/topos-plugin-mock/icon", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != string(want) {
		t.Errorf("expected body %q, got %q", want, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "image/svg+xml" {
		t.Errorf("expected Content-Type image/svg+xml, got %q", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Errorf("expected immutable Cache-Control, got %q", got)
	}
	if got := rec.Header().Get("ETag"); got == "" {
		t.Error("expected a non-empty ETag")
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("expected X-Content-Type-Options nosniff, got %q", got)
	}
	if got := rec.Header().Get("Content-Disposition"); got != "inline" {
		t.Errorf("expected Content-Disposition inline, got %q", got)
	}
	if got := rec.Header().Get("Content-Security-Policy"); got == "" {
		t.Error("expected a non-empty Content-Security-Policy (T-09-01)")
	}
}

// TestPluginIconHandler_ConditionalRequestReturns304 proves an
// If-None-Match matching the served ETag gets a 304 with no body.
func TestPluginIconHandler_ConditionalRequestReturns304(t *testing.T) {
	provider := &fakePluginIconProvider{icons: map[string]fakeIcon{
		"topos-plugin-mock": {bytes: []byte("<svg>mock</svg>"), mime: "image/svg+xml"},
	}}
	router := newTestPluginIconRouter(provider)

	first := httptest.NewRequest(http.MethodGet, "/api/plugins/topos-plugin-mock/icon", nil)
	firstRec := httptest.NewRecorder()
	router.ServeHTTP(firstRec, first)
	etag := firstRec.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected the first response to carry an ETag")
	}

	second := httptest.NewRequest(http.MethodGet, "/api/plugins/topos-plugin-mock/icon", nil)
	second.Header.Set("If-None-Match", etag)
	secondRec := httptest.NewRecorder()
	router.ServeHTTP(secondRec, second)

	if secondRec.Code != http.StatusNotModified {
		t.Fatalf("expected 304, got %d", secondRec.Code)
	}
	if secondRec.Body.Len() != 0 {
		t.Errorf("expected an empty 304 body, got %d bytes", secondRec.Body.Len())
	}
}

// TestPluginIconHandler_UnknownPluginReturns404 proves a plugin binary the
// kernel has never successfully Described 404s in the package's standard
// error envelope — the routine, expected "undescribed" state
// PluginIcon.svelte's fallback chain always covers (09-UI-SPEC.md Fix 10).
func TestPluginIconHandler_UnknownPluginReturns404(t *testing.T) {
	router := newTestPluginIconRouter(&fakePluginIconProvider{})

	req := httptest.NewRequest(http.MethodGet, "/api/plugins/topos-plugin-does-not-exist/icon", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assertErrorEnvelope(t, rec, http.StatusNotFound, "icon_unavailable")
}

// TestPluginIconHandler_PathTraversalAttemptReturns404WithoutCallingProvider
// proves a {plugin} value containing a path separator or ".." 404s BEFORE
// the provider is ever consulted — the lookup is an exact-match over an
// in-memory map, so traversal must be structurally impossible, not merely
// filtered (T-09-04).
func TestPluginIconHandler_PathTraversalAttemptReturns404WithoutCallingProvider(t *testing.T) {
	cases := []string{
		"..%2F..%2Fetc%2Fpasswd",
		"foo..bar",
		"a%5Cb",
	}
	for _, plugin := range cases {
		t.Run(plugin, func(t *testing.T) {
			provider := &fakePluginIconProvider{icons: map[string]fakeIcon{
				"topos-plugin-mock": {bytes: []byte("x"), mime: "image/svg+xml"},
			}}
			router := newTestPluginIconRouter(provider)

			req := httptest.NewRequest(http.MethodGet, "/api/plugins/"+plugin+"/icon", nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Errorf("expected 404 for plugin=%q, got %d", plugin, rec.Code)
			}
			if len(provider.calledWith) != 0 {
				t.Errorf("expected the provider to never be called for plugin=%q, got calls: %v", plugin, provider.calledWith)
			}
		})
	}
}
