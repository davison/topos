package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/index"
	"github.com/davison/topos/kernel/pluginhost"
	"github.com/davison/topos/kernel/syncer"
)

// fakeProber is a test double satisfying httpapi.HealthProber without
// launching real plugin subprocesses.
type fakeProber struct {
	healths []pluginhost.SourceHealth
}

func (f *fakeProber) ProbeSources(context.Context) []pluginhost.SourceHealth {
	return f.healths
}

// fakeRefresher is a test double satisfying httpapi.Refresher.
type fakeRefresher struct {
	refreshResult syncer.RunResult
	refreshErr    error
	refreshAll    []syncer.RunResult
	lastRefreshed string
}

func (f *fakeRefresher) Refresh(_ context.Context, sourceName string) (syncer.RunResult, error) {
	f.lastRefreshed = sourceName
	return f.refreshResult, f.refreshErr
}

func (f *fakeRefresher) RefreshAll(context.Context) []syncer.RunResult {
	return f.refreshAll
}

func newTestSourcesRouter(store *index.Store, cfg *config.Config, prober HealthProber, refresher Refresher) http.Handler {
	r := chi.NewRouter()
	r.Get("/api/sources", SourcesHandler(store, prober))
	// Rule 3 (07-02-PLAN.md Task 2): SourceRefreshHandler now takes a
	// *config.Store — every call site in this file still builds a
	// *config.Config by hand, so wrap it here rather than touching each one.
	r.Post("/api/sources/{name}/refresh", SourceRefreshHandler(config.NewStoreForTesting(cfg), refresher))
	r.Post("/api/sync", SyncRefreshHandler(refresher))
	return r
}

// TestSourcesHandler_ReportsTierPerInstance is Phase 11's shape check
// (PLUG-06/07): GET /api/sources publishes the launch-time provenance
// Tier verbatim, per instance — one trusted, one external, in the same
// response — never merged or dropped. The trust badge (SourceChip's
// TrustBadge) renders off this field alone, so the wire contract must
// carry it byte-exact.
func TestSourcesHandler_ReportsTierPerInstance(t *testing.T) {
	store := newTestStoreForHTTP(t)
	prober := &fakeProber{healths: []pluginhost.SourceHealth{
		{Name: "paperless", SourceType: "paperless", DisplayName: "paperless-ngx", Reachable: true, Tier: pluginhost.TierTrusted},
		{Name: "example", SourceType: "example", DisplayName: "Example", Reachable: true, Tier: pluginhost.TierExternal},
	}}
	router := newTestSourcesRouter(store, &config.Config{}, prober, &fakeRefresher{})

	req := httptest.NewRequest(http.MethodGet, "/api/sources", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var resp sourcesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var paperless, example sourceStatus
	for _, s := range resp.Sources {
		switch s.Name {
		case "paperless":
			paperless = s
		case "example":
			example = s
		}
	}
	if paperless.Tier != "trusted" {
		t.Errorf("expected paperless tier %q, got %q", "trusted", paperless.Tier)
	}
	if example.Tier != "external" {
		t.Errorf("expected example tier %q, got %q", "external", example.Tier)
	}
}

func TestSourcesHandler_ReturnsBothSourcesSortedByName(t *testing.T) {
	store := newTestStoreForHTTP(t)
	prober := &fakeProber{healths: []pluginhost.SourceHealth{
		{Name: "silverbullet", SourceType: "silverbullet", DisplayName: "SilverBullet", Reachable: true},
		{Name: "paperless", SourceType: "paperless", DisplayName: "paperless-ngx", Reachable: true},
	}}
	router := newTestSourcesRouter(store, &config.Config{}, prober, &fakeRefresher{})

	req := httptest.NewRequest(http.MethodGet, "/api/sources", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp sourcesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(resp.Sources))
	}
	if resp.Sources[0].Name != "paperless" || resp.Sources[1].Name != "silverbullet" {
		t.Errorf("expected sources sorted by name, got: %+v", resp.Sources)
	}
	for _, s := range resp.Sources {
		if s.DisplayName == "" || s.SourceType == "" {
			t.Errorf("expected non-empty display_name/source_type, got: %+v", s)
		}
	}
}

// TestSourcesHandler_UnreachableSourceReportsLastErrorFromRecordedRun
// proves an unreachable source has reachable:false and a non-empty
// last_error, while a sibling source in the same response is unaffected —
// last_error itself comes from the kernel's own sync_runs history
// (A-PLUG-04), so this seeds a previously-recorded failed run for the
// unreachable source rather than relying on the live probe's own error
// text.
func TestSourcesHandler_UnreachableSourceReportsLastErrorFromRecordedRun(t *testing.T) {
	store := newTestStoreForHTTP(t)
	ctx := context.Background()
	runID, err := store.StartSyncRun(ctx, "silverbullet")
	if err != nil {
		t.Fatalf("StartSyncRun: %v", err)
	}
	if err := store.FinishSyncRun(ctx, runID, "error", "dial tcp: connection refused", 0); err != nil {
		t.Fatalf("FinishSyncRun: %v", err)
	}

	prober := &fakeProber{healths: []pluginhost.SourceHealth{
		{Name: "paperless", SourceType: "paperless", DisplayName: "paperless-ngx", Reachable: true},
		{Name: "silverbullet", SourceType: "silverbullet", DisplayName: "SilverBullet", Reachable: false, ProbeError: "dial tcp: connection refused"},
	}}
	router := newTestSourcesRouter(store, &config.Config{}, prober, &fakeRefresher{})

	req := httptest.NewRequest(http.MethodGet, "/api/sources", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var resp sourcesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var paperless, silverbullet sourceStatus
	for _, s := range resp.Sources {
		switch s.Name {
		case "paperless":
			paperless = s
		case "silverbullet":
			silverbullet = s
		}
	}
	if silverbullet.Reachable {
		t.Error("expected silverbullet.Reachable to be false")
	}
	if silverbullet.LastError == "" {
		t.Error("expected silverbullet.LastError to be non-empty")
	}
	if !paperless.Reachable {
		t.Error("expected the sibling source (paperless) to be unaffected: Reachable true")
	}
}

func TestSourcesHandler_NoRecordedRunReportsNeutralUnknownState(t *testing.T) {
	store := newTestStoreForHTTP(t)
	prober := &fakeProber{healths: []pluginhost.SourceHealth{
		{Name: "paperless", SourceType: "paperless", DisplayName: "paperless-ngx", Reachable: true},
	}}
	router := newTestSourcesRouter(store, &config.Config{}, prober, &fakeRefresher{})

	req := httptest.NewRequest(http.MethodGet, "/api/sources", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var resp sourcesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(resp.Sources))
	}
	got := resp.Sources[0]
	if got.LastStatus != "" || got.LastSyncUnix != 0 || got.LastError != "" {
		t.Errorf("expected the neutral unknown state (empty status, zero last_sync_unix, empty error) for a source with no recorded run, got: %+v", got)
	}
}

func TestSourcesHandler_RunningRowReportsSyncingTrue(t *testing.T) {
	store := newTestStoreForHTTP(t)
	ctx := context.Background()
	if _, err := store.StartSyncRun(ctx, "paperless"); err != nil {
		t.Fatalf("StartSyncRun: %v", err)
	}

	prober := &fakeProber{healths: []pluginhost.SourceHealth{
		{Name: "paperless", SourceType: "paperless", DisplayName: "paperless-ngx", Reachable: true},
	}}
	router := newTestSourcesRouter(store, &config.Config{}, prober, &fakeRefresher{})

	req := httptest.NewRequest(http.MethodGet, "/api/sources", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var resp sourcesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Sources) != 1 || !resp.Sources[0].Syncing {
		t.Errorf("expected syncing:true for a source with an unfinished run, got: %+v", resp.Sources)
	}
}

// TestSourcesHandler_ProbeFailureForOnePluginDoesNotFailWholeResponse
// proves a probe error for one plugin yields reachable:false for that
// source and a 200 overall — never a 500 for the whole response.
func TestSourcesHandler_ProbeFailureForOnePluginDoesNotFailWholeResponse(t *testing.T) {
	store := newTestStoreForHTTP(t)
	prober := &fakeProber{healths: []pluginhost.SourceHealth{
		{Name: "paperless", SourceType: "paperless", DisplayName: "paperless-ngx", Reachable: true},
		{Name: "silverbullet", SourceType: "silverbullet", DisplayName: "SilverBullet", Reachable: false, ProbeError: "context deadline exceeded"},
	}}
	router := newTestSourcesRouter(store, &config.Config{}, prober, &fakeRefresher{})

	req := httptest.NewRequest(http.MethodGet, "/api/sources", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 even with one plugin's probe failing, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSourceRefreshHandler_ValidNameReturns200WithRunStatus(t *testing.T) {
	store := newTestStoreForHTTP(t)
	cfg := &config.Config{Sources: map[string]config.Source{"paperless": {Plugin: "x", BaseURL: "http://x", Token: "t"}}}
	refresher := &fakeRefresher{refreshResult: syncer.RunResult{Source: "paperless", SourceType: "paperless", Status: "ok", ItemCount: 3}}
	router := newTestSourcesRouter(store, cfg, &fakeProber{}, refresher)

	req := httptest.NewRequest(http.MethodPost, "/api/sources/paperless/refresh", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if refresher.lastRefreshed != "paperless" {
		t.Errorf("expected the refresher to be called with 'paperless', got %q", refresher.lastRefreshed)
	}
	var resp sourceRefreshResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Source.Status != "ok" || resp.Source.ItemCount != 3 {
		t.Errorf("unexpected refresh response: %+v", resp.Source)
	}
}

// TestSourceRefreshHandler_UnconfiguredNameReturns404WithoutEnumerating
// proves an unconfigured {name} returns 404 source_not_found in the
// standard error envelope, and that the response body does not contain
// the name of any configured source (T-02-09).
func TestSourceRefreshHandler_UnconfiguredNameReturns404WithoutEnumerating(t *testing.T) {
	store := newTestStoreForHTTP(t)
	cfg := &config.Config{Sources: map[string]config.Source{
		"paperless":    {Plugin: "x", BaseURL: "http://x", Token: "t"},
		"silverbullet": {Plugin: "y", BaseURL: "http://y", Token: "t"},
	}}
	refresher := &fakeRefresher{}
	router := newTestSourcesRouter(store, cfg, &fakeProber{}, refresher)

	req := httptest.NewRequest(http.MethodPost, "/api/sources/nope/refresh", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assertErrorEnvelope(t, rec, http.StatusNotFound, "source_not_found")
	body := rec.Body.String()
	if strings.Contains(body, "paperless") || strings.Contains(body, "silverbullet") {
		t.Errorf("expected the 404 body to not enumerate configured source names, got: %s", body)
	}
	if refresher.lastRefreshed != "" {
		t.Error("expected the refresher to never be dispatched for an unconfigured name")
	}
}

func TestSyncRefreshHandler_ReturnsOneStatusPerSource(t *testing.T) {
	store := newTestStoreForHTTP(t)
	refresher := &fakeRefresher{refreshAll: []syncer.RunResult{
		{Source: "paperless", SourceType: "paperless", Status: "ok", ItemCount: 5},
		{Source: "silverbullet", SourceType: "silverbullet", Status: "error", Error: "boom"},
	}}
	router := newTestSourcesRouter(store, &config.Config{}, &fakeProber{}, refresher)

	req := httptest.NewRequest(http.MethodPost, "/api/sync", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp syncRefreshResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Sources) != 2 {
		t.Fatalf("expected 2 statuses, got %d", len(resp.Sources))
	}
}

// TestAggregateSyncStatus_ErrorTakesPrecedenceOverOK proves the stream
// envelope's sync.status is "error" when one of two sources' latest run
// errored and the other's succeeded later — a webspace whose only failing
// source returned nothing must never look merely empty.
func TestAggregateSyncStatus_ErrorTakesPrecedenceOverOK(t *testing.T) {
	runs := map[string]index.SyncRun{
		"paperless":    {Source: "paperless", Status: "ok", FinishedUnix: 200},
		"silverbullet": {Source: "silverbullet", Status: "error", Error: "connection refused", FinishedUnix: 100},
	}
	got := aggregateSyncStatus(runs)
	if got.Status != "error" {
		t.Errorf("expected aggregate status 'error', got %q", got.Status)
	}
	if got.FinishedUnix != 200 {
		t.Errorf("expected FinishedUnix to be the newest finished timestamp (200), got %d", got.FinishedUnix)
	}
	if got.Error == "" {
		t.Error("expected a non-empty aggregate error message")
	}
}

// TestFilterRunsByParticipation_FourBranches is the direct unit test of
// filterRunsByParticipation (08-10-PLAN.md Task 2, 08-UAT.md G-08-3):
// four source instances against one webspace, each exercising a distinct
// branch of the (config-membership AND correlate.ParticipatesIn) rule.
func TestFilterRunsByParticipation_FourBranches(t *testing.T) {
	runs := map[string]index.SyncRun{
		// removed-from-config: still has a run row, but no longer a key of
		// cfg.Sources — must be dropped regardless of the webspace's own
		// participation rules.
		"removed-instance": {Source: "removed-instance", Status: "error"},
		// excluded-by-allowlist: still configured, but the webspace's
		// explicit Sources allowlist does not name it — must be dropped.
		"excluded-instance": {Source: "excluded-instance", Status: "error"},
		// keywords-fallback: configured, no explicit Match block, kept in
		// by the webspace's non-empty Keywords fallback — must be kept.
		"keywords-instance": {Source: "keywords-instance", Status: "ok"},
		// explicit-match-block: configured, and named directly in the
		// webspace's Match map — must be kept regardless of Keywords.
		"explicit-match-instance": {Source: "explicit-match-instance", Status: "ok"},
	}

	cfg := &config.Config{
		Sources: map[string]config.Source{
			"excluded-instance":       {Plugin: "x", BaseURL: "http://x", Token: "t"},
			"keywords-instance":       {Plugin: "x", BaseURL: "http://x", Token: "t"},
			"explicit-match-instance": {Plugin: "x", BaseURL: "http://x", Token: "t"},
			// "removed-instance" is deliberately absent from cfg.Sources.
		},
		Webspaces: map[string]config.Webspace{
			"ws": {
				Keywords: []string{"demo"},
				Sources:  []string{"keywords-instance", "explicit-match-instance"},
				Match: map[string]config.MatchBlock{
					"explicit-match-instance": {"field": []string{"value"}},
				},
			},
		},
	}

	got := filterRunsByParticipation(runs, cfg, "ws")

	if _, ok := got["removed-instance"]; ok {
		t.Error("expected removed-instance (dropped from cfg.Sources) to be excluded")
	}
	if _, ok := got["excluded-instance"]; ok {
		t.Error("expected excluded-instance (not in the webspace's Sources allowlist) to be excluded")
	}
	if _, ok := got["keywords-instance"]; !ok {
		t.Error("expected keywords-instance (kept in via the Keywords fallback) to be included")
	}
	if _, ok := got["explicit-match-instance"]; !ok {
		t.Error("expected explicit-match-instance (named in an explicit Match block) to be included")
	}
	if len(got) != 2 {
		t.Errorf("expected exactly 2 participating sources, got %d: %+v", len(got), got)
	}
}

func TestAggregateSyncStatus_RunningTakesPrecedenceOverOK(t *testing.T) {
	runs := map[string]index.SyncRun{
		"paperless":    {Source: "paperless", Status: "ok", FinishedUnix: 200},
		"silverbullet": {Source: "silverbullet", Status: "running"},
	}
	got := aggregateSyncStatus(runs)
	if got.Status != "running" {
		t.Errorf("expected aggregate status 'running', got %q", got.Status)
	}
}

func TestAggregateSyncStatus_EmptyMapReturnsZeroValue(t *testing.T) {
	got := aggregateSyncStatus(map[string]index.SyncRun{})
	if got.Status != "" || got.FinishedUnix != 0 || got.Error != "" {
		t.Errorf("expected the zero value for no recorded runs, got: %+v", got)
	}
}

// TestStreamHandler_SyncStatusErrorWhenOneOfTwoSourcesFailed proves the
// stream envelope itself (not just aggregateSyncStatus in isolation)
// reports sync.status "error" when one of two sources' latest run
// errored and the other's succeeded later.
//
// Rebuilt on newTestRouterWithConfig with a real config (08-10-PLAN.md
// Task 2, G-08-3): this test originally seeded its webspace into the
// index only and built its router with an EMPTY config (newTestRouter),
// which under participation scoping resolves cfg.Webspaces["house-move"]
// to the zero value — no participants, the zero-value sync object, and
// this test failing for the WRONG reason (an unconfigured webspace, not
// the two-source precedence rule it was written to guard). Both
// instances are now explicitly configured and the webspace declares a
// Keywords fallback so both participate (correlate.ParticipatesIn) —
// the assertion below now holds for the reason it always meant to.
func TestStreamHandler_SyncStatusErrorWhenOneOfTwoSourcesFailed(t *testing.T) {
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
		t.Errorf("expected stream sync.status 'error' when one of two participating sources failed, got %q", resp.Sync.Status)
	}
}
