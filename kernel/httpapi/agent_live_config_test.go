// agent_live_config_test.go is the agent-surface sibling of
// live_config_test.go. It exists because 07-REVIEW.md CR-01 found that the
// live-config fix 07-02-PLAN.md Task 2 performed for /api/* — resolving
// the running config fresh, per request, from cfgStore — was never
// extended to /agent/v1/*: MountAgentRoutes resolved cfgStore.Expanded()
// exactly once at router-construction time and closed that single
// snapshot into four of the five agent handlers, so a grant revoked
// through the same Store.Save path PUT /api/config and
// POST /api/config/reload use stayed in force on the agent surface until
// the kernel process restarted — a live authorization-bypass window on
// AGENT-01's default-deny model, and a direct contradiction of D-06 (save
// = apply immediately). Every assertion below runs against the SAME
// already-constructed router across a save, never a re-built one — that
// is the entire property under test.
package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/index"
	"github.com/davison/topos/kernel/item"
	"github.com/davison/topos/kernel/pluginhost"
)

// agentLiveConfigRouter builds the REAL Router (not individual handlers)
// over a real temp-file *config.Store, mirroring liveConfigTestRouter
// (live_config_test.go) exactly — a t.TempDir() config file, a real
// config.NewStore over that path, newTestStoreForHTTP for the index, and
// Router with fakeRefresher/fakeApplier/the unused-plugins-dir
// string/a null logger — except it also takes a HealthProber and a
// Fetcher, rather than liveConfigTestRouter's hardcoded zero-value
// &fakeProber{}. It needs to exist separately rather than reusing
// liveConfigTestRouter because /agent/v1/sources is built from
// sourceStatusesFrom, whose entire output comes from the prober: with no
// healths reported, a granted and an ungranted config would both produce
// an empty sources array, and a revocation test run against
// liveConfigTestRouter would pass for the wrong reason — there would
// never have been anything to revoke FROM.
func agentLiveConfigRouter(t *testing.T, contents string, prober HealthProber, fetcher Fetcher) (http.Handler, *config.Store, *index.Store) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	cfgStore, err := config.NewStore(path)
	if err != nil {
		t.Fatalf("config.NewStore: %v", err)
	}
	store := newTestStoreForHTTP(t)
	router := Router(store, cfgStore, fetcher, prober, &fakeRefresher{}, &fakeApplier{}, "testdata-unused-plugins-dir", hclog.NewNullLogger())
	return router, cfgStore, store
}

func getAgentSources(t *testing.T, router http.Handler) agentSourcesResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/agent/v1/sources", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /agent/v1/sources: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp agentSourcesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return resp
}

func getAgentWebspaces(t *testing.T, router http.Handler) webspacesResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/agent/v1/webspaces", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /agent/v1/webspaces: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp webspacesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return resp
}

// agentItemCountFor returns -1 if name is not present in resp, so a caller
// comparing against an expected count never confuses "absent" with "zero".
func agentItemCountFor(resp webspacesResponse, name string) int {
	for _, ws := range resp.Webspaces {
		if ws.Name == name {
			return ws.ItemCount
		}
	}
	return -1
}

// assertAgentItemRouteReflectsRevocation asserts rec is the exact
// item_not_found envelope a genuinely nonexistent id gets — never a
// distinct "forbidden" code, so the agent surface cannot be used to
// enumerate revoked-but-existing sources (T-02-20) — and states the
// authorization consequence in its failure message: this is a security
// assertion, not a cosmetic one. A revoked source's items must be
// unreadable through route on the SAME router instance with no restart,
// immediately, not eventually.
func assertAgentItemRouteReflectsRevocation(t *testing.T, rec *httptest.ResponseRecorder, route string) {
	t.Helper()
	if rec.Code != http.StatusNotFound {
		t.Fatalf("%s: expected 404 item_not_found reflecting the revocation on the SAME router instance with no restart (a revoked source's items must become unreadable immediately, not after a restart) — got %d: %s", route, rec.Code, rec.Body.String())
	}
	var envelope errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("%s: unmarshal error envelope: %v", route, err)
	}
	if envelope.Error.Code != "item_not_found" {
		t.Errorf("%s: expected code item_not_found — the identical envelope a nonexistent id gets, never a distinct code that would let the agent surface enumerate which sources exist but are withheld — reflecting the revocation on the SAME router instance with no restart, got %q", route, envelope.Error.Code)
	}
}

// TestAgentLiveConfig_RevokedReadGrantTakesEffectWithoutRestart proves
// 07-REVIEW.md CR-01's fix end to end: a source's agent.read grant
// revoked through Store.Save (the identical write path PUT /api/config
// and POST /api/config/reload use) disappears from every /agent/v1 route
// on the very next request against the SAME already-constructed router,
// with the kernel process never restarted.
func TestAgentLiveConfig_RevokedReadGrantTakesEffectWithoutRestart(t *testing.T) {
	prober := &fakeProber{healths: []pluginhost.SourceHealth{
		{Name: "paperless", SourceType: "paperless", DisplayName: "Paperless", Reachable: true},
	}}
	fetcher := &fakeFetcher{result: pluginhost.FetchResult{Available: true, Text: "extracted text", MimeType: "application/pdf", SizeBytes: 3}}

	router, cfgStore, store := agentLiveConfigRouter(t, `
[sources.paperless]
plugin = "topos-plugin-paperless"
base_url = "http://paperless.lan:8000"
token = "unused"
display_name = "Paperless"

[sources.paperless.agent]
read = true
handoff = true

[webspaces.house-move]
keywords = ["house-move"]
`, prober, fetcher)

	if err := store.ReplaceWebspaceSourceItems(context.Background(), "house-move", "paperless", []item.Item{
		{
			ID: "paperless:1", Source: "paperless", SourceType: "paperless", SourceID: "1",
			Title: "Doc 1", Fidelity: item.FidelityExact, DeepLink: "http://paperless.lan:8000/documents/1",
		},
	}); err != nil {
		t.Fatalf("seed item: %v", err)
	}

	// Before the save: granted end to end.
	sourcesBefore := getAgentSources(t, router)
	if len(sourcesBefore.Sources) != 1 || sourcesBefore.Sources[0].Name != "paperless" || !sourcesBefore.Sources[0].Capabilities.Read {
		t.Fatalf("expected exactly the granted 'paperless' source with read capability true before the save, got: %+v", sourcesBefore.Sources)
	}

	itemReq := httptest.NewRequest(http.MethodGet, "/agent/v1/items/paperless:1", nil)
	itemRec := httptest.NewRecorder()
	router.ServeHTTP(itemRec, itemReq)
	if itemRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for the granted item before the save, got %d: %s", itemRec.Code, itemRec.Body.String())
	}

	webspacesBefore := getAgentWebspaces(t, router)
	if got := agentItemCountFor(webspacesBefore, "house-move"); got != 1 {
		t.Fatalf("expected item_count 1 for house-move before the save, got %d", got)
	}

	// Revoke read alone — handoff stays true deliberately, proving read is
	// what gates this surface and matching the three-way-equivalence
	// property agent_test.go already covers.
	saveConfig(t, cfgStore, &config.Config{
		Sources: map[string]config.Source{
			"paperless": {
				Plugin: "topos-plugin-paperless", BaseURL: "http://paperless.lan:8000", Token: "unused",
				DisplayName: "Paperless", Agent: config.AgentGrant{Read: false, Handoff: true},
			},
		},
		Webspaces: map[string]config.Webspace{"house-move": {Keywords: []string{"house-move"}}},
	})

	// After the save, against the SAME router with nothing re-constructed:
	// revocation is in force on the very next request, no restart.
	sourcesAfter := getAgentSources(t, router)
	if len(sourcesAfter.Sources) != 0 {
		t.Errorf("expected zero sources to reflect the save on the SAME router instance with no restart, got: %+v", sourcesAfter.Sources)
	}

	itemReq2 := httptest.NewRequest(http.MethodGet, "/agent/v1/items/paperless:1", nil)
	itemRec2 := httptest.NewRecorder()
	router.ServeHTTP(itemRec2, itemReq2)
	assertAgentItemRouteReflectsRevocation(t, itemRec2, "GET /agent/v1/items/paperless:1")

	contentReq := httptest.NewRequest(http.MethodGet, "/agent/v1/items/paperless:1/content", nil)
	contentRec := httptest.NewRecorder()
	router.ServeHTTP(contentRec, contentReq)
	assertAgentItemRouteReflectsRevocation(t, contentRec, "GET /agent/v1/items/paperless:1/content")

	thumbnailReq := httptest.NewRequest(http.MethodGet, "/agent/v1/items/paperless:1/thumbnail", nil)
	thumbnailRec := httptest.NewRecorder()
	router.ServeHTTP(thumbnailRec, thumbnailReq)
	assertAgentItemRouteReflectsRevocation(t, thumbnailRec, "GET /agent/v1/items/paperless:1/thumbnail")

	webspacesAfter := getAgentWebspaces(t, router)
	if got := agentItemCountFor(webspacesAfter, "house-move"); got != 0 {
		t.Errorf("expected item_count 0 for house-move to reflect the save on the SAME router instance with no restart, got %d", got)
	}
}

// TestAgentLiveConfig_NewlyGrantedSourceIsVisibleWithoutRestart proves the
// inverse direction: a source newly granted agent.read = true through a
// save is present in /agent/v1/sources, counted in /agent/v1/webspaces,
// and readable through /agent/v1/items/{id} on the very next request
// against the SAME router, with no restart (D-06 in both directions).
func TestAgentLiveConfig_NewlyGrantedSourceIsVisibleWithoutRestart(t *testing.T) {
	prober := &fakeProber{healths: []pluginhost.SourceHealth{
		{Name: "paperless", SourceType: "paperless", DisplayName: "Paperless", Reachable: true},
	}}
	fetcher := &fakeFetcher{result: pluginhost.FetchResult{Available: true, Text: "extracted text", MimeType: "application/pdf", SizeBytes: 3}}

	router, cfgStore, store := agentLiveConfigRouter(t, `
[sources.paperless]
plugin = "topos-plugin-paperless"
base_url = "http://paperless.lan:8000"
token = "unused"
display_name = "Paperless"

[webspaces.house-move]
keywords = ["house-move"]
`, prober, fetcher)

	if err := store.ReplaceWebspaceSourceItems(context.Background(), "house-move", "paperless", []item.Item{
		{
			ID: "paperless:1", Source: "paperless", SourceType: "paperless", SourceID: "1",
			Title: "Doc 1", Fidelity: item.FidelityExact, DeepLink: "http://paperless.lan:8000/documents/1",
		},
	}); err != nil {
		t.Fatalf("seed item: %v", err)
	}

	// Before the save: no agent sub-table at all, absent everywhere.
	sourcesBefore := getAgentSources(t, router)
	if len(sourcesBefore.Sources) != 0 {
		t.Fatalf("expected zero sources before any grant is saved, got: %+v", sourcesBefore.Sources)
	}

	itemReq := httptest.NewRequest(http.MethodGet, "/agent/v1/items/paperless:1", nil)
	itemRec := httptest.NewRecorder()
	router.ServeHTTP(itemRec, itemReq)
	assertAgentItemRouteReflectsRevocation(t, itemRec, "GET /agent/v1/items/paperless:1 (pre-grant)")

	// Grant read = true.
	saveConfig(t, cfgStore, &config.Config{
		Sources: map[string]config.Source{
			"paperless": {
				Plugin: "topos-plugin-paperless", BaseURL: "http://paperless.lan:8000", Token: "unused",
				DisplayName: "Paperless", Agent: config.AgentGrant{Read: true},
			},
		},
		Webspaces: map[string]config.Webspace{"house-move": {Keywords: []string{"house-move"}}},
	})

	// After the save, against the SAME router: the new grant is in force
	// on the very next request, no restart.
	sourcesAfter := getAgentSources(t, router)
	if len(sourcesAfter.Sources) != 1 || sourcesAfter.Sources[0].Name != "paperless" || !sourcesAfter.Sources[0].Capabilities.Read {
		t.Errorf("expected the newly granted 'paperless' source with read capability true to reflect the save on the SAME router instance with no restart, got: %+v", sourcesAfter.Sources)
	}

	webspacesAfter := getAgentWebspaces(t, router)
	if got := agentItemCountFor(webspacesAfter, "house-move"); got != 1 {
		t.Errorf("expected item_count 1 for house-move to reflect the save on the SAME router instance with no restart, got %d", got)
	}

	itemReq2 := httptest.NewRequest(http.MethodGet, "/agent/v1/items/paperless:1", nil)
	itemRec2 := httptest.NewRecorder()
	router.ServeHTTP(itemRec2, itemReq2)
	if itemRec2.Code != http.StatusOK {
		t.Fatalf("expected 200 for the newly granted item to reflect the save on the SAME router instance with no restart, got %d: %s", itemRec2.Code, itemRec2.Body.String())
	}
}
