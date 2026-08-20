package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/correlate"
	"github.com/davison/topos/kernel/index"
	"github.com/davison/topos/kernel/pluginhost"
	"github.com/davison/topos/kernel/syncer"
	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

// fakeProber is a test double satisfying httpapi.HealthProber without
// launching real plugin subprocesses.
type fakeProber struct {
	healths  []pluginhost.SourceHealth
	failures []pluginhost.LaunchFailure
}

func (f *fakeProber) ProbeSources(context.Context) []pluginhost.SourceHealth {
	return f.healths
}

// LaunchFailures satisfies httpapi.HealthProber's Phase 11 widening
// (11-03-PLAN.md Task 1) — defaults to nil (no launch failures) for every
// existing test that constructs a bare &fakeProber{healths: ...} literal.
func (f *fakeProber) LaunchFailures() []pluginhost.LaunchFailure {
	return f.failures
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

// TestSourcesHandler_MergesLaunchFailureIntoSourcesResponse is Phase 11's
// central shape check (PLUG-07/08, 11-RESEARCH.md Pitfall 1/2): a configured
// source that never launched at all (a pin mismatch) still produces a real,
// named entry in GET /api/sources — not an entry silently absent — carrying
// the closed-vocabulary launch_failure field and both hashes.
func TestSourcesHandler_MergesLaunchFailureIntoSourcesResponse(t *testing.T) {
	store := newTestStoreForHTTP(t)
	prober := &fakeProber{
		healths: []pluginhost.SourceHealth{
			{Name: "paperless", SourceType: "paperless", DisplayName: "paperless-ngx", Reachable: true, Tier: pluginhost.TierTrusted},
		},
		failures: []pluginhost.LaunchFailure{
			{
				Instance:    "swapped",
				Plugin:      "topos-plugin-example",
				DisplayName: "Swapped Source",
				Tier:        pluginhost.TierExternal,
				Reason:      pluginhost.LaunchFailurePinMismatch,
				PinnedHash:  "aaaa",
				CurrentHash: "bbbb",
				Message:     `pluginhost: instance "swapped" binary "topos-plugin-example" hash mismatch: pinned=aaaa current=bbbb`,
			},
		},
	}
	router := newTestSourcesRouter(store, &config.Config{}, prober, &fakeRefresher{})

	req := httptest.NewRequest(http.MethodGet, "/api/sources", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var resp sourcesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Sources) != 2 {
		t.Fatalf("expected 2 entries (1 probed + 1 launch-failed), got %d: %+v", len(resp.Sources), resp.Sources)
	}
	// Sorted by name: "paperless" < "swapped".
	if resp.Sources[0].Name != "paperless" || resp.Sources[1].Name != "swapped" {
		t.Fatalf("expected entries sorted by name, got: %+v", resp.Sources)
	}
	swapped := resp.Sources[1]
	if swapped.DisplayName != "Swapped Source" {
		t.Errorf("expected display_name %q, got %q", "Swapped Source", swapped.DisplayName)
	}
	if swapped.Plugin != "topos-plugin-example" {
		t.Errorf("expected plugin %q, got %q", "topos-plugin-example", swapped.Plugin)
	}
	if swapped.Tier != "external" {
		t.Errorf("expected tier %q, got %q", "external", swapped.Tier)
	}
	if swapped.LaunchFailure != "pin_mismatch" {
		t.Errorf("expected launch_failure %q, got %q", "pin_mismatch", swapped.LaunchFailure)
	}
	if swapped.PinnedHash != "aaaa" {
		t.Errorf("expected pinned_hash %q, got %q", "aaaa", swapped.PinnedHash)
	}
	if swapped.CurrentHash != "bbbb" {
		t.Errorf("expected current_hash %q, got %q", "bbbb", swapped.CurrentHash)
	}
	if swapped.Reachable {
		t.Error("expected reachable:false for a launch-failed source")
	}
	if swapped.SourceType != "" {
		t.Errorf("expected empty source_type (Describe never ran), got %q", swapped.SourceType)
	}
	if swapped.LastError == "" {
		t.Error("expected last_error to carry the kernel's own named message")
	}
	// The healthy sibling is unaffected.
	paperless := resp.Sources[0]
	if paperless.LaunchFailure != "" {
		t.Errorf("expected the sibling healthy source to have no launch_failure, got %q", paperless.LaunchFailure)
	}
}

// TestSourcesHandler_LaunchFailedEntryCarriesNoLastNotice pins WR-01
// (12-11-PLAN.md gap closure): docs/api.md's last_notice bullet already
// publishes that this field is empty for an instance that never launched
// — this test records a COMPLETED sync run carrying a non-empty notice for
// an instance, then proves the launch-failure merge deliberately omits
// that notice from the response while still copying the run's
// last_status/last_sync_unix (proving the merge genuinely saw the
// recorded run, so an empty last_notice here is a real contract rather
// than a vacuous pass over a run the merge never looked at).
func TestSourcesHandler_LaunchFailedEntryCarriesNoLastNotice(t *testing.T) {
	store := newTestStoreForHTTP(t)
	ctx := context.Background()

	const recordedNotice = "webspace 'test': match value 'swapped.folders=*' matched no items"
	runID, err := store.StartSyncRun(ctx, "swapped")
	if err != nil {
		t.Fatalf("StartSyncRun: %v", err)
	}
	if err := store.FinishSyncRunWithNotice(ctx, runID, "ok", "", recordedNotice, 3); err != nil {
		t.Fatalf("FinishSyncRunWithNotice: %v", err)
	}

	prober := &fakeProber{
		failures: []pluginhost.LaunchFailure{
			{
				Instance:    "swapped",
				Plugin:      "topos-plugin-example",
				DisplayName: "Swapped Source",
				Tier:        pluginhost.TierExternal,
				Reason:      pluginhost.LaunchFailurePinMismatch,
				PinnedHash:  "aaaa",
				CurrentHash: "bbbb",
				Message:     `pluginhost: instance "swapped" binary "topos-plugin-example" hash mismatch: pinned=aaaa current=bbbb`,
			},
		},
	}
	router := newTestSourcesRouter(store, &config.Config{}, prober, &fakeRefresher{})

	req := httptest.NewRequest(http.MethodGet, "/api/sources", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/sources: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Assert against the raw decoded JSON body first — proving the wire
	// tag itself, not only the Go field — matching
	// TestSources_ZeroMatchNoticeTravelsFromCorrelateToTheSourcesAPI's own
	// technique.
	var raw map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	sources, _ := raw["sources"].([]any)
	var rawEntry map[string]any
	for _, s := range sources {
		m, _ := s.(map[string]any)
		if m["name"] == "swapped" {
			rawEntry = m
			break
		}
	}
	if rawEntry == nil {
		t.Fatalf("expected a 'swapped' entry in raw sources, got: %+v", raw["sources"])
	}
	if rawEntry["last_notice"] != "" {
		t.Errorf("expected last_notice exactly empty for a launch-failed entry (WR-01, docs/api.md's published contract), got %v", rawEntry["last_notice"])
	}

	var resp sourcesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal typed: %v", err)
	}
	var swapped *sourceStatus
	for i := range resp.Sources {
		if resp.Sources[i].Name == "swapped" {
			swapped = &resp.Sources[i]
			break
		}
	}
	if swapped == nil {
		t.Fatalf("expected a 'swapped' entry in the typed response, got: %+v", resp.Sources)
	}
	if swapped.LastNotice != "" {
		t.Errorf("expected LastNotice exactly empty for a launch-failed entry, got %q", swapped.LastNotice)
	}
	// last_status/last_sync_unix DO still come from the recorded run —
	// proving this test genuinely recorded a run the merge saw, so the
	// empty LastNotice assertion above means something.
	if swapped.LastStatus != "ok" {
		t.Errorf("expected LastStatus %q from the recorded run, got %q", "ok", swapped.LastStatus)
	}
	if swapped.LastSyncUnix == 0 {
		t.Error("expected a non-zero LastSyncUnix from the recorded run")
	}
	if swapped.LaunchFailure != "pin_mismatch" {
		t.Errorf("expected launch_failure %q, got %q", "pin_mismatch", swapped.LaunchFailure)
	}
}

// TestSourcesHandler_ProbeResultWinsOverLaunchFailureForSameInstance proves
// an instance present in BOTH the probe result and the launch-failure set
// yields exactly one entry — the probe result, since it describes what is
// actually running right now.
func TestSourcesHandler_ProbeResultWinsOverLaunchFailureForSameInstance(t *testing.T) {
	store := newTestStoreForHTTP(t)
	prober := &fakeProber{
		healths: []pluginhost.SourceHealth{
			{Name: "example", SourceType: "example", DisplayName: "Example", Reachable: true, Tier: pluginhost.TierExternal},
		},
		failures: []pluginhost.LaunchFailure{
			{Instance: "example", Plugin: "topos-plugin-example", DisplayName: "Stale Failure", Tier: pluginhost.TierExternal, Reason: pluginhost.LaunchFailurePinMismatch},
		},
	}
	router := newTestSourcesRouter(store, &config.Config{}, prober, &fakeRefresher{})

	req := httptest.NewRequest(http.MethodGet, "/api/sources", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var resp sourcesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Sources) != 1 {
		t.Fatalf("expected exactly 1 entry for a name present in both sets, got %d: %+v", len(resp.Sources), resp.Sources)
	}
	got := resp.Sources[0]
	if got.LaunchFailure != "" {
		t.Errorf("expected the probe result to win (no launch_failure), got %q", got.LaunchFailure)
	}
	if !got.Reachable {
		t.Error("expected the probe result's reachable:true to win")
	}
}

// TestSourcesHandler_HealthyExternalSourceReportsPinnedHash proves the
// pinned_hash field is populated for a HEALTHY external-tier source, not
// only a pin-mismatched one — the chip menu's pinned-hash footer needs it
// either way.
func TestSourcesHandler_HealthyExternalSourceReportsPinnedHash(t *testing.T) {
	store := newTestStoreForHTTP(t)
	prober := &fakeProber{healths: []pluginhost.SourceHealth{
		{Name: "example", SourceType: "example", DisplayName: "Example", Reachable: true, Tier: pluginhost.TierExternal, PinnedHash: "cccc"},
	}}
	router := newTestSourcesRouter(store, &config.Config{}, prober, &fakeRefresher{})

	req := httptest.NewRequest(http.MethodGet, "/api/sources", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var resp sourcesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Sources) != 1 || resp.Sources[0].PinnedHash != "cccc" {
		t.Errorf("expected pinned_hash %q on the healthy external source, got: %+v", "cccc", resp.Sources)
	}
}

// TestAgentSourcesHandler_LaunchFailedSourceRespectsGrant proves the merge
// in sourceStatusesFrom stays subject to agent.go's grant filter: a
// launch-failed source with no read grant is structurally absent from
// GET /agent/v1/sources, and present (with capabilities.read=true) once
// granted.
func TestAgentSourcesHandler_LaunchFailedSourceRespectsGrant(t *testing.T) {
	store := newTestStoreForHTTP(t)
	failure := pluginhost.LaunchFailure{
		Instance: "swapped", Plugin: "topos-plugin-example", DisplayName: "Swapped Source",
		Tier: pluginhost.TierExternal, Reason: pluginhost.LaunchFailurePinMismatch,
	}

	ungranted := &config.Config{Sources: map[string]config.Source{
		"swapped": {Plugin: "topos-plugin-example", BaseURL: "http://x", Token: "t"},
	}}
	prober := &fakeProber{failures: []pluginhost.LaunchFailure{failure}}
	router := newAgentTestRouter(store, ungranted, &fakeFetcher{}, prober)

	req := httptest.NewRequest(http.MethodGet, "/agent/v1/sources", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var resp agentSourcesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Sources) != 0 {
		t.Errorf("expected an ungranted launch-failed source to be absent, got: %+v", resp.Sources)
	}

	granted := &config.Config{Sources: map[string]config.Source{
		"swapped": {Plugin: "topos-plugin-example", BaseURL: "http://x", Token: "t", Agent: config.AgentGrant{Read: true}},
	}}
	router2 := newAgentTestRouter(store, granted, &fakeFetcher{}, prober)
	req2 := httptest.NewRequest(http.MethodGet, "/agent/v1/sources", nil)
	rec2 := httptest.NewRecorder()
	router2.ServeHTTP(rec2, req2)
	var resp2 agentSourcesResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp2.Sources) != 1 {
		t.Fatalf("expected a granted launch-failed source to be present, got: %+v", resp2.Sources)
	}
	if resp2.Sources[0].LaunchFailure != "pin_mismatch" {
		t.Errorf("expected launch_failure %q, got %q", "pin_mismatch", resp2.Sources[0].LaunchFailure)
	}
	if !resp2.Sources[0].Capabilities.Read {
		t.Error("expected capabilities.read true for a granted source")
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

// zeroItemMatchSource is the minimal correlate.Source fake
// TestSources_ZeroMatchNoticeTravelsFromCorrelateToTheSourcesAPI needs
// locally: a plugin whose Match answers with no items and no error — "the
// plugin filtered everything out," exactly what the filesystem plugin did
// in the reported failure (.planning/debug/filesystem-items-missing-from-stream.md).
type zeroItemMatchSource struct {
	name       string
	sourceType string
	vocabulary []string
}

func (z *zeroItemMatchSource) Name() string              { return z.name }
func (z *zeroItemMatchSource) SourceType() string        { return z.sourceType }
func (z *zeroItemMatchSource) MatchVocabulary() []string { return z.vocabulary }
func (z *zeroItemMatchSource) Match(context.Context, map[string][]string) (*toposv1.MatchResponse, error) {
	return &toposv1.MatchResponse{}, nil
}

// TestSources_ZeroMatchNoticeTravelsFromCorrelateToTheSourcesAPI is the
// end-to-end proof for the SECOND `missing:` item of G-12-1/G-12-3
// (12-09-PLAN.md): a webspace's explicit match block that matched zero
// items across a healthy sync is named, by webspace and offending value,
// all the way to GET /api/sources's last_notice — while last_status stays
// "ok" and last_error stays empty, so the advisory is never mistaken for
// a failure. Wires a real index.Store, a real correlate.Engine, and a
// real syncer.Coordinator behind Router (modelled on agent_test.go's
// newAgentTestRouter and coordinator_test.go's real-stack tests), with no
// fake between them except the source plugin itself.
func TestSources_ZeroMatchNoticeTravelsFromCorrelateToTheSourcesAPI(t *testing.T) {
	store := newTestStoreForHTTP(t)

	cfg := &config.Config{
		Sources: map[string]config.Source{
			"files": {Plugin: "topos-plugin-filesystem"},
		},
		Webspaces: map[string]config.Webspace{
			"test": {
				Match: map[string]config.MatchBlock{
					"files": {"path": {"/nonexistent/**"}},
				},
			},
		},
	}

	src := &zeroItemMatchSource{name: "files", sourceType: "filesystem", vocabulary: []string{"path"}}
	engine := &correlate.Engine{Store: store, Config: cfg}
	coord := syncer.NewCoordinator(store, engine, []correlate.Source{src})

	prober := &fakeProber{healths: []pluginhost.SourceHealth{
		{Name: "files", SourceType: "filesystem", DisplayName: "Files", Reachable: true, Tier: pluginhost.TierTrusted},
	}}

	router, _ := Router(store, config.NewStoreForTesting(cfg), &fakeFetcher{}, prober, coord, &fakeApplier{}, &fakeSuspender{}, &fakePluginIconProvider{}, pluginhost.Dirs{Trusted: "testdata-unused-plugins-dir"}, hclog.NewNullLogger())

	refreshReq := httptest.NewRequest(http.MethodPost, "/api/sources/files/refresh", nil)
	refreshRec := httptest.NewRecorder()
	router.ServeHTTP(refreshRec, refreshReq)
	if refreshRec.Code != http.StatusOK {
		t.Fatalf("POST /api/sources/files/refresh: expected 200, got %d: %s", refreshRec.Code, refreshRec.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sources", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/sources: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Decode from the raw JSON body (map[string]any), not only the Go
	// struct, so the wire tag "last_notice" itself is proven — not merely
	// that the Go field exists.
	var raw map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if schemaVersion, _ := raw["schema_version"].(float64); schemaVersion != 1 {
		t.Errorf("expected schema_version 1 (last_notice is additive), got %v", raw["schema_version"])
	}
	sources, _ := raw["sources"].([]any)
	var entry map[string]any
	for _, s := range sources {
		m, _ := s.(map[string]any)
		if m["name"] == "files" {
			entry = m
			break
		}
	}
	if entry == nil {
		t.Fatalf("expected a 'files' entry in raw sources, got: %+v", raw["sources"])
	}
	if entry["last_status"] != "ok" {
		t.Errorf("expected last_status exactly \"ok\", got %v", entry["last_status"])
	}
	if entry["last_error"] != "" {
		t.Errorf("expected last_error exactly empty, got %v", entry["last_error"])
	}
	rawNotice, _ := entry["last_notice"].(string)
	if rawNotice == "" {
		t.Fatal("expected a non-empty last_notice")
	}
	if !strings.Contains(rawNotice, "test") || !strings.Contains(rawNotice, "path") || !strings.Contains(rawNotice, "/nonexistent/**") {
		t.Errorf("expected last_notice to name the webspace, field and value, got %q", rawNotice)
	}

	// Also assert through the typed struct, proving both the wire tag and
	// the Go field agree.
	var resp sourcesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal typed: %v", err)
	}
	var typedEntry *sourceStatus
	for i := range resp.Sources {
		if resp.Sources[i].Name == "files" {
			typedEntry = &resp.Sources[i]
			break
		}
	}
	if typedEntry == nil {
		t.Fatal("expected a 'files' entry in the typed response")
	}
	if typedEntry.LastStatus != "ok" || typedEntry.LastError != "" || typedEntry.LastNotice == "" {
		t.Errorf("expected LastStatus ok, LastError empty, LastNotice non-empty, got: %+v", typedEntry)
	}
}

// TestSourcesHandler_ShadowedSourceCarriesLaunchAdvisory is 13-05-PLAN.md
// Task 3's own shape check (D-14/PD-05): GET /api/sources carries
// launch_advisory equal to the shadowed value on a launched, shadowed
// source, and that its launch_failure is empty — a shadowed instance DID
// launch.
func TestSourcesHandler_ShadowedSourceCarriesLaunchAdvisory(t *testing.T) {
	store := newTestStoreForHTTP(t)
	prober := &fakeProber{healths: []pluginhost.SourceHealth{
		{Name: "paperless", SourceType: "paperless", DisplayName: "paperless-ngx", Reachable: true, Tier: pluginhost.TierTrusted, LaunchAdvisory: pluginhost.LaunchAdvisoryShadowed},
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
		t.Fatalf("expected exactly one entry, got %d: %+v", len(resp.Sources), resp.Sources)
	}
	entry := resp.Sources[0]
	if entry.LaunchAdvisory != pluginhost.LaunchAdvisoryShadowed {
		t.Errorf("expected launch_advisory %q, got %q", pluginhost.LaunchAdvisoryShadowed, entry.LaunchAdvisory)
	}
	if entry.LaunchFailure != "" {
		t.Errorf("expected an empty launch_failure on a launched, shadowed source, got %q", entry.LaunchFailure)
	}
	if !entry.Reachable {
		t.Error("expected the shadowed source to be reachable — it DID launch")
	}

	rawBody := rec.Body.String()
	if !strings.Contains(rawBody, `"launch_advisory":"shadowed"`) {
		t.Errorf("expected the raw response body to carry \"launch_advisory\":\"shadowed\", got: %s", rawBody)
	}
}

// TestSourcesHandler_ManifestUnverifiedEntryCarriesNoLaunchAdvisory is
// TestSourcesHandler_ShadowedSourceCarriesLaunchAdvisory's negative
// sibling: a manifest-refused source's entry carries launch_failure equal
// to manifest_unverified and an empty launch_advisory — the two fields
// are never both populated on the same entry.
func TestSourcesHandler_ManifestUnverifiedEntryCarriesNoLaunchAdvisory(t *testing.T) {
	store := newTestStoreForHTTP(t)
	prober := &fakeProber{
		failures: []pluginhost.LaunchFailure{
			{
				Instance:    "dropped",
				Plugin:      "topos-plugin-dropped",
				DisplayName: "Dropped binary",
				Tier:        pluginhost.TierTrusted,
				Reason:      pluginhost.LaunchFailureManifestUnverified,
				CurrentHash: "cccc",
				Message:     `pluginhost: instance "dropped" binary "topos-plugin-dropped" is not verified by the kernel's build manifest (current=cccc)`,
			},
		},
	}
	router := newTestSourcesRouter(store, &config.Config{}, prober, &fakeRefresher{})

	req := httptest.NewRequest(http.MethodGet, "/api/sources", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var resp sourcesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Sources) != 1 {
		t.Fatalf("expected exactly one entry, got %d: %+v", len(resp.Sources), resp.Sources)
	}
	entry := resp.Sources[0]
	if entry.LaunchFailure != pluginhost.LaunchFailureManifestUnverified {
		t.Errorf("expected launch_failure %q, got %q", pluginhost.LaunchFailureManifestUnverified, entry.LaunchFailure)
	}
	if entry.LaunchAdvisory != "" {
		t.Errorf("expected an empty launch_advisory on a manifest-refused entry, got %q", entry.LaunchAdvisory)
	}
	if entry.CurrentHash != "cccc" {
		t.Errorf("expected current_hash %q, got %q", "cccc", entry.CurrentHash)
	}
	if entry.Reachable {
		t.Error("expected reachable:false for a manifest-refused entry")
	}

	rawBody := rec.Body.String()
	if strings.Contains(rawBody, "launch_advisory") {
		t.Errorf("expected omitempty to drop launch_advisory entirely from the raw body, got: %s", rawBody)
	}
}

// copyMockBinaryToFreshDirForFallbackPathTest mirrors
// kernel/pluginhost/pin_test.go's copyMockBinaryToFreshDir: buildMockPluginDir
// (config_test.go) caches ONE shared binary/directory across this whole
// package's test run — a private copy is required here because this test
// mutates the binary's bytes (the re-pin scenario below) and must never
// corrupt the fixture every other test in this file also depends on.
func copyMockBinaryToFreshDirForFallbackPathTest(t *testing.T) string {
	t.Helper()
	srcDir := buildMockPluginDir(t)
	data, err := os.ReadFile(filepath.Join(srcDir, "topos-plugin-mock"))
	if err != nil {
		t.Fatalf("read shared mock plugin fixture: %v", err)
	}
	dstDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dstDir, "topos-plugin-mock"), data, 0o755); err != nil {
		t.Fatalf("write private mock plugin copy: %v", err)
	}
	return dstDir
}

// writeTamperedCopyForFallbackPathTest reads srcPath's bytes, flips the
// final byte in MEMORY, and writes the result into a brand-new file at
// dstDir/topos-plugin-mock — never mutating srcPath itself. This
// deliberately avoids in-place mutation of a binary that may have just
// been exec'd (go-plugin's Kill() does not guarantee the kernel has fully
// released the process image before returning; a same-inode O_RDWR open
// shortly after can race a lingering ETXTBSY on Linux) by always producing
// tampered bytes as a NEW file the OS has never mapped as a running
// process's text segment.
func writeTamperedCopyForFallbackPathTest(t *testing.T, srcPath string) string {
	t.Helper()
	data, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("read %s: %v", srcPath, err)
	}
	tampered := append([]byte{}, data...)
	tampered[len(tampered)-1] ^= 0xFF

	dstDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dstDir, "topos-plugin-mock"), tampered, 0o755); err != nil {
		t.Fatalf("write tampered copy: %v", err)
	}
	return dstDir
}

// TestSources_UnsignedExternalBinaryConsentAndPinPathUnchanged is the
// explicit TRUST-03 regression net this phase owes (16-02-PLAN.md Task 3,
// D-11): exercises the REAL pluginhost.Discover/launch machinery (never
// fakeProber) against a genuinely unsigned external-tier binary, end to
// end through GET /api/sources — proving the unsigned consent-and-pin path
// is byte-for-byte unchanged by the provenance-derived tier rewrite. Four
// sub-steps: (1) no pin recorded -> tier "external", launch_failure
// "pin_mismatch" (never "manifest_unverified"); (2) pinned and matching ->
// launches; (3) the SAME pin against DIFFERENT bytes (simulating "the
// binary changed since it was pinned") -> fails again, still
// "pin_mismatch"; (4) a re-pin write (a fresh pin matching the changed
// bytes' NEW hash) restores launchability — the fallback path's full
// consent-and-pin lifecycle, unchanged by this phase.
func TestSources_UnsignedExternalBinaryConsentAndPinPathUnchanged(t *testing.T) {
	dir := copyMockBinaryToFreshDirForFallbackPathTest(t)
	binPath := filepath.Join(dir, "topos-plugin-mock")
	originalHash, err := pluginhost.HashBinary(binPath)
	if err != nil {
		t.Fatalf("HashBinary: %v", err)
	}

	store := newTestStoreForHTTP(t)
	sources := map[string]config.Source{"demo": {Plugin: "topos-plugin-mock"}}

	// 1. No pin recorded at all: refuses with pin_mismatch, never
	// manifest_unverified — the tier is external from the start (D-11: no
	// provenance evidence anywhere for this binary), so the manifest gate
	// is never even consulted.
	h1, err := pluginhost.Discover(context.Background(), pluginhost.Dirs{External: dir}, &config.Config{Sources: sources}, sources, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("Discover (unpinned): %v", err)
	}
	router1 := newTestSourcesRouter(store, &config.Config{}, h1, &fakeRefresher{})
	rec1 := httptest.NewRecorder()
	router1.ServeHTTP(rec1, httptest.NewRequest(http.MethodGet, "/api/sources", nil))
	var resp1 sourcesResponse
	if err := json.Unmarshal(rec1.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("unmarshal (unpinned): %v", err)
	}
	if len(resp1.Sources) != 1 {
		t.Fatalf("expected exactly one source entry, got %+v", resp1.Sources)
	}
	if resp1.Sources[0].Tier != "external" {
		t.Errorf("expected tier %q, got %q", "external", resp1.Sources[0].Tier)
	}
	if resp1.Sources[0].LaunchFailure != pluginhost.LaunchFailurePinMismatch {
		t.Errorf("expected launch_failure %q for an unpinned external binary, got %q", pluginhost.LaunchFailurePinMismatch, resp1.Sources[0].LaunchFailure)
	}
	h1.Shutdown()

	// 2. Pinned and matching: launches successfully, zero failures.
	pinnedRaw := &config.Config{Sources: sources, Plugins: config.PluginsConfig{Pins: map[string]string{"topos-plugin-mock": originalHash}}}
	h2, err := pluginhost.Discover(context.Background(), pluginhost.Dirs{External: dir}, pinnedRaw, sources, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("Discover (pinned): %v", err)
	}
	if len(h2.Plugins()) != 1 {
		t.Fatalf("expected the pinned instance to launch, got failures: %+v", h2.LaunchFailures())
	}
	if len(h2.LaunchFailures()) != 0 {
		t.Fatalf("expected zero launch failures for a matching pin, got %+v", h2.LaunchFailures())
	}
	h2.Shutdown()

	// 3. "The binary changed since it was pinned": a fresh directory with
	// DIFFERENT bytes under the SAME pin — the same_pin, different_bytes
	// shape a real post-pin tamper produces, without touching the
	// already-executed original file (see
	// writeTamperedCopyForFallbackPathTest's own doc comment for why).
	tamperedDir := writeTamperedCopyForFallbackPathTest(t, binPath)
	tamperedPath := filepath.Join(tamperedDir, "topos-plugin-mock")
	h3, err := pluginhost.Discover(context.Background(), pluginhost.Dirs{External: tamperedDir}, pinnedRaw, sources, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("Discover (tampered): %v", err)
	}
	if len(h3.Plugins()) != 0 {
		t.Fatalf("expected the tampered instance to fail to launch, got %+v", h3.Plugins())
	}
	failures := h3.LaunchFailures()
	if len(failures) != 1 || failures[0].Reason != pluginhost.LaunchFailurePinMismatch {
		t.Fatalf("expected exactly one pin_mismatch failure after tampering, got %+v", failures)
	}
	h3.Shutdown()

	// 4. A re-pin write (a fresh pin matching the changed bytes' NEW hash)
	// makes the changed binary launchable again.
	newHash, err := pluginhost.HashBinary(tamperedPath)
	if err != nil {
		t.Fatalf("HashBinary (tampered): %v", err)
	}
	rePinnedRaw := &config.Config{Sources: sources, Plugins: config.PluginsConfig{Pins: map[string]string{"topos-plugin-mock": newHash}}}
	h4, err := pluginhost.Discover(context.Background(), pluginhost.Dirs{External: tamperedDir}, rePinnedRaw, sources, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("Discover (re-pinned): %v", err)
	}
	defer h4.Shutdown()
	if len(h4.Plugins()) != 1 {
		t.Fatalf("expected the re-pinned instance to launch again, got failures: %+v", h4.LaunchFailures())
	}
	if len(h4.LaunchFailures()) != 0 {
		t.Fatalf("expected zero launch failures after re-pinning, got %+v", h4.LaunchFailures())
	}
}
