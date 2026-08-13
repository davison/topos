package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/index"
	"github.com/davison/topos/kernel/item"
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
	return newTestRouterWithConfig(store, &config.Config{})
}

func newTestRouterWithConfig(store *index.Store, cfg *config.Config) http.Handler {
	r := chi.NewRouter()
	// Rule 3 (07-01-PLAN.md Task 1): StreamHandler now takes a
	// *config.Store rather than a bare *config.Config — every existing
	// test fixture here builds a Config by hand, so
	// config.NewStoreForTesting wraps it rather than every test being
	// rewritten to load a real TOML file.
	r.Get("/api/webspaces/{webspace}/stream", StreamHandler(store, config.NewStoreForTesting(cfg)))
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

// TestStreamHandler_ConfigKnownNeverSyncedReturns200EmptyArray (07-15-PLAN.md
// Task 1, G-07-1) proves a webspace named in the running config, with
// nothing seeded in the index at all, is servable the instant it is
// configured — the reported create-flow symptom. The pre-fix gate consults
// only Store.WebspaceExists (sync history), so this fails 404 until the
// config half of the disjunction lands.
func TestStreamHandler_ConfigKnownNeverSyncedReturns200EmptyArray(t *testing.T) {
	store := newTestStoreForHTTP(t)
	cfg := &config.Config{Webspaces: map[string]config.Webspace{
		"new-project": {}, // D-20 empty shell: the create flow's first PUT
	}}
	router := newTestRouterWithConfig(store, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/webspaces/new-project/stream", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for a config-known-never-synced webspace, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp streamResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal stream response: %v", err)
	}
	if resp.Items == nil {
		t.Error("expected items to be a non-nil empty array, not null")
	}
	if len(resp.Items) != 0 {
		t.Errorf("expected zero items, got %d", len(resp.Items))
	}
}

// TestStreamHandler_ZeroConfiguredSourcesReturns200EmptyArray (07-15-PLAN.md
// Task 1, G-07-1 corollary) proves the permanent-404 case: on an install
// with ZERO [sources.*] blocks at all, no sync ever runs, so a
// sync-history-derived gate 404s forever and Retry can never heal it. The
// config half removes the dependency on sync timing entirely.
func TestStreamHandler_ZeroConfiguredSourcesReturns200EmptyArray(t *testing.T) {
	store := newTestStoreForHTTP(t)
	cfg := &config.Config{
		Sources:   map[string]config.Source{},
		Webspaces: map[string]config.Webspace{"new-project": {}},
	}
	router := newTestRouterWithConfig(store, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/webspaces/new-project/stream", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on a zero-configured-sources install (the permanent-404 corollary), got %d: %s", rec.Code, rec.Body.String())
	}
	var resp streamResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal stream response: %v", err)
	}
	if resp.Items == nil {
		t.Error("expected items to be a non-nil empty array, not null")
	}
}

func TestStreamHandler_ReturnsItemsWithLinkAndProvenance(t *testing.T) {
	store := newTestStoreForHTTP(t)
	ctx := context.Background()

	it := item.Item{
		ID: "paperless:42", Source: "paperless", SourceType: "paperless", SourceID: "42",
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

// TestStreamHandler_TwoInstancesOfOnePluginTypeReportDistinctSourceAndDisplayName
// proves the stream envelope carries source (instance id) and
// source_display_name distinctly for two source instances sharing one
// plugin kind (D-08/D-09) — the exact shape two [sources.*] entries
// pointing at the same plugin binary produce.
func TestStreamHandler_TwoInstancesOfOnePluginTypeReportDistinctSourceAndDisplayName(t *testing.T) {
	store := newTestStoreForHTTP(t)
	ctx := context.Background()

	home := item.Item{
		ID: "home-email:1", Source: "home-email", SourceType: "proton", SourceID: "1",
		Title: "Home item", Fidelity: item.FidelityExact, DeepLink: "https://mail.proton.me/home/1",
		TimestampUnix: 200,
	}
	work := item.Item{
		ID: "work-email:1", Source: "work-email", SourceType: "proton", SourceID: "1",
		Title: "Work item", Fidelity: item.FidelityExact, DeepLink: "https://mail.proton.me/work/1",
		TimestampUnix: 100,
	}
	if err := store.ReplaceWebspaceSourceItems(ctx, "ws", "home-email", []item.Item{home}); err != nil {
		t.Fatalf("seed home-email: %v", err)
	}
	if err := store.ReplaceWebspaceSourceItems(ctx, "ws", "work-email", []item.Item{work}); err != nil {
		t.Fatalf("seed work-email: %v", err)
	}

	cfg := &config.Config{Sources: map[string]config.Source{
		"home-email": {Plugin: "topos-plugin-proton", DisplayName: "Home Email"},
		"work-email": {Plugin: "topos-plugin-proton", DisplayName: "Work Email"},
	}}
	router := newTestRouterWithConfig(store, cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/webspaces/ws/stream", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp streamResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}

	byID := map[string]streamItem{}
	for _, it := range resp.Items {
		byID[it.ID] = it
	}
	homeItem, ok := byID["home-email:1"]
	if !ok {
		t.Fatalf("expected item 'home-email:1' present, got: %+v", resp.Items)
	}
	if homeItem.Source != "home-email" || homeItem.SourceDisplayName != "Home Email" {
		t.Errorf("expected source=home-email display_name='Home Email', got source=%q display_name=%q", homeItem.Source, homeItem.SourceDisplayName)
	}
	if homeItem.SourceType != "proton" {
		t.Errorf("expected source_type 'proton' (shared plugin kind), got %q", homeItem.SourceType)
	}
	workItem, ok := byID["work-email:1"]
	if !ok {
		t.Fatalf("expected item 'work-email:1' present, got: %+v", resp.Items)
	}
	if workItem.Source != "work-email" || workItem.SourceDisplayName != "Work Email" {
		t.Errorf("expected source=work-email display_name='Work Email', got source=%q display_name=%q", workItem.Source, workItem.SourceDisplayName)
	}
}

// TestStreamHandler_NonParticipatingSourceFailureDoesNotEscalate proves a
// webspace whose Sources allowlist excludes a failing instance reports
// sync.status "ok" — that instance's failure must never leak into a
// webspace it does not feed (08-UAT.md G-08-3, kernel/httpapi/sources.go's
// filterRunsByParticipation).
func TestStreamHandler_NonParticipatingSourceFailureDoesNotEscalate(t *testing.T) {
	store := newTestStoreForHTTP(t)
	ctx := context.Background()
	if err := store.ReplaceWebspaceSourceItems(ctx, "house-move", "paperless", nil); err != nil {
		t.Fatalf("seed: %v", err)
	}

	id1, _ := store.StartSyncRun(ctx, "silverbullet")
	store.FinishSyncRun(ctx, id1, "error", "connection refused", 0)
	id2, _ := store.StartSyncRun(ctx, "paperless")
	store.FinishSyncRun(ctx, id2, "ok", "", 5)

	cfg := &config.Config{
		Sources: map[string]config.Source{
			"paperless":    {Plugin: "x", BaseURL: "http://x", Token: "t"},
			"silverbullet": {Plugin: "y", BaseURL: "http://y", Token: "t"},
		},
		Webspaces: map[string]config.Webspace{
			// Explicit allowlist excludes silverbullet — its failure must
			// never surface for this webspace.
			"house-move": {Keywords: []string{"x"}, Sources: []string{"paperless"}},
		},
	}
	router := newTestRouterWithConfig(store, cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/webspaces/house-move/stream", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var resp streamResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Sync.Status != "ok" {
		t.Errorf("expected sync.status 'ok' when the only failing source is excluded by the webspace's allowlist, got %q (error: %q)", resp.Sync.Status, resp.Sync.Error)
	}
}

// TestStreamHandler_ParticipatingSourceFailureStillEscalates proves the
// guarantee this scoping change must not break: a source that DOES
// participate in the webspace still escalates sync.status to "error",
// naming the failing instance id in sync.error.
func TestStreamHandler_ParticipatingSourceFailureStillEscalates(t *testing.T) {
	store := newTestStoreForHTTP(t)
	ctx := context.Background()
	if err := store.ReplaceWebspaceSourceItems(ctx, "house-move", "paperless", nil); err != nil {
		t.Fatalf("seed: %v", err)
	}

	id1, _ := store.StartSyncRun(ctx, "silverbullet")
	store.FinishSyncRun(ctx, id1, "error", "connection refused", 0)

	cfg := &config.Config{
		Sources: map[string]config.Source{
			"paperless":    {Plugin: "x", BaseURL: "http://x", Token: "t"},
			"silverbullet": {Plugin: "y", BaseURL: "http://y", Token: "t"},
		},
		Webspaces: map[string]config.Webspace{
			"house-move": {Keywords: []string{"x"}},
		},
	}
	router := newTestRouterWithConfig(store, cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/webspaces/house-move/stream", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var resp streamResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Sync.Status != "error" {
		t.Errorf("expected sync.status 'error' for a participating source's failure, got %q", resp.Sync.Status)
	}
	if !strings.Contains(resp.Sync.Error, "silverbullet") {
		t.Errorf("expected sync.error to name the failing instance 'silverbullet', got %q", resp.Sync.Error)
	}
}

// TestStreamHandler_IndexOnlyWebspaceReportsZeroValueSyncDespiteOtherFailure
// proves a webspace known only from surviving index rows (absent from
// config) reports the zero-value sync object even while another source's
// latest run has failed — it has no [webspaces.*] block, therefore no
// participants (correlate.ParticipatesIn is false for every source
// against the zero-value config.Webspace), therefore nothing to
// aggregate over.
func TestStreamHandler_IndexOnlyWebspaceReportsZeroValueSyncDespiteOtherFailure(t *testing.T) {
	store := newTestStoreForHTTP(t)
	ctx := context.Background()
	if err := store.ReplaceWebspaceSourceItems(ctx, "orphaned-webspace", "paperless", nil); err != nil {
		t.Fatalf("seed: %v", err)
	}

	id1, _ := store.StartSyncRun(ctx, "paperless")
	store.FinishSyncRun(ctx, id1, "error", "connection refused", 0)

	// paperless is still configured (a run row can outlive a webspace's
	// removal from config, per filterRunsByParticipation's own doc
	// comment) but "orphaned-webspace" itself has no [webspaces.*] block.
	cfg := &config.Config{
		Sources: map[string]config.Source{
			"paperless": {Plugin: "x", BaseURL: "http://x", Token: "t"},
		},
	}
	router := newTestRouterWithConfig(store, cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/webspaces/orphaned-webspace/stream", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for an index-only webspace, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp streamResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Sync.Status != "" || resp.Sync.FinishedUnix != 0 || resp.Sync.Error != "" {
		t.Errorf("expected the zero-value sync object for an index-only, config-unknown webspace, got: %+v", resp.Sync)
	}
}

// --- resolveStreamLinkURL (12-01-PLAN.md Task 2, D-06): the file://-scheme rewrite ---

// TestStreamLink_FileSchemeDeepLinkIsRewrittenToTheOpenRoute proves an
// item whose DeepLink carries the file:// scheme is served with link.url
// pointed at the kernel's loopback open route for that item's own id,
// never the raw file:// value.
func TestStreamLink_FileSchemeDeepLinkIsRewrittenToTheOpenRoute(t *testing.T) {
	it := item.Item{
		ID:       "docs-folder:invoice.pdf",
		Source:   "docs-folder",
		DeepLink: "file:///home/user/Documents/invoice.pdf",
		Fidelity: item.FidelityExact,
	}
	got := resolveStreamLinkURL(it)
	want := "/api/items/docs-folder:invoice.pdf/open"
	if got != want {
		t.Errorf("expected link.url %q, got %q", want, got)
	}
}

// TestStreamLink_NonFileSchemeDeepLinkIsEchoedUnchanged proves an item
// whose DeepLink is an ordinary https URL is served with that value
// verbatim — the rewrite is additive, never touching any other source's
// existing behavior.
func TestStreamLink_NonFileSchemeDeepLinkIsEchoedUnchanged(t *testing.T) {
	it := item.Item{
		ID:       "paperless:42",
		Source:   "paperless",
		DeepLink: "https://paperless.example/documents/42",
		Fidelity: item.FidelityExact,
	}
	got := resolveStreamLinkURL(it)
	want := "https://paperless.example/documents/42"
	if got != want {
		t.Errorf("expected link.url echoed unchanged as %q, got %q", want, got)
	}
}

// TestStreamHandler_FileSchemeItemServesRewrittenLinkEndToEnd proves the
// rewrite reaches the real HTTP response, not merely the unit-level helper.
func TestStreamHandler_FileSchemeItemServesRewrittenLinkEndToEnd(t *testing.T) {
	store := newTestStoreForHTTP(t)
	ctx := context.Background()
	items := []item.Item{{
		ID:       "docs-folder:invoice.pdf",
		Source:   "docs-folder",
		SourceID: "invoice.pdf",
		Title:    "invoice.pdf",
		Fidelity: item.FidelityExact,
		DeepLink: "file:///home/user/Documents/invoice.pdf",
	}}
	if err := store.ReplaceWebspaceSourceItems(ctx, "house-move", "docs-folder", items); err != nil {
		t.Fatalf("seed: %v", err)
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
		t.Fatalf("expected exactly 1 item, got %d", len(resp.Items))
	}
	got := resp.Items[0].Link.URL
	if got == "" || got[0] != '/' || !strings.HasSuffix(got, "/open") {
		t.Errorf("expected link.url to be the loopback open route, got %q", got)
	}
	if strings.HasPrefix(got, "file://") {
		t.Errorf("expected the raw file:// deep link to never be served, got %q", got)
	}
}
