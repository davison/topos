// live_config_test.go proves the promote decision 07-01-PLAN.md locked at
// its Task 1 tracer checkpoint holds mechanically over the REAL Router
// (07-02-PLAN.md Task 2): a config-derived value saved through
// Store.Save is visible on the very next request to the SAME
// already-constructed router instance, on every route that reads config —
// confirmed individually against each of the three handlers this task
// fixed (WebspacesHandler, ItemHandler, SourceRefreshHandler), since each
// was a distinct boot-time snapshot before this task.
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
)

// liveConfigTestRouter builds the REAL Router (not individual handlers)
// over a real temp-file *config.Store, so Store.Save's on-disk swap is
// exactly the production path every assertion below observes through. It
// also returns the *index.Store the router was built over, so a test can
// seed items through the identical store the router reads.
func liveConfigTestRouter(t *testing.T, contents string) (http.Handler, *config.Store, *index.Store) {
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
	router, _ := Router(store, cfgStore, &fakeFetcher{}, &fakeProber{}, &fakeRefresher{}, &fakeApplier{}, &fakeSuspender{}, "testdata-unused-plugins-dir", hclog.NewNullLogger())
	return router, cfgStore, store
}

func saveConfig(t *testing.T, cfgStore *config.Store, next *config.Config) {
	t.Helper()
	if err := cfgStore.Save(next, cfgStore.Hash()); err != nil {
		t.Fatalf("Save: %v", err)
	}
}

// TestLiveConfig_WebspacesHandler_KeywordsChangeVisibleWithoutRestart
// proves WebspacesHandler's own fix: the keywords a webspace reports come
// from cfgStore, resolved fresh per request, not a value captured once
// when Router was constructed.
func TestLiveConfig_WebspacesHandler_KeywordsChangeVisibleWithoutRestart(t *testing.T) {
	router, cfgStore, _ := liveConfigTestRouter(t, `
[webspaces.house-move]
keywords = ["before"]
`)

	before := getWebspaces(t, router)
	if got := before.Webspaces[0].Keywords; len(got) != 1 || got[0] != "before" {
		t.Fatalf("expected initial keywords [\"before\"], got %v", got)
	}

	saveConfig(t, cfgStore, &config.Config{
		Webspaces: map[string]config.Webspace{"house-move": {Keywords: []string{"after"}}},
	})

	after := getWebspaces(t, router)
	if got := after.Webspaces[0].Keywords; len(got) != 1 || got[0] != "after" {
		t.Fatalf("expected keywords to reflect the save on the SAME router instance with no restart, got %v", got)
	}
}

func getWebspaces(t *testing.T, router http.Handler) webspacesResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/webspaces", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/webspaces: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp webspacesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return resp
}

// TestLiveConfig_ItemHandler_DisplayNameChangeVisibleWithoutRestart proves
// ItemHandler's own fix: source_display_name is resolved via cfgStore
// fresh per request — a display_name edit saved through Store.Save is
// visible on the very next GET /api/items/{id} against the SAME router.
func TestLiveConfig_ItemHandler_DisplayNameChangeVisibleWithoutRestart(t *testing.T) {
	router, cfgStore, store := liveConfigTestRouter(t, `
[sources.paperless]
plugin = "topos-plugin-paperless"
base_url = "http://paperless.lan:8000"
token = "unused"
display_name = "before"

[webspaces.house-move]
keywords = ["house-move"]
`)

	if err := store.ReplaceWebspaceSourceItems(context.Background(), "house-move", "paperless", []item.Item{
		{
			ID: "paperless:1", Source: "paperless", SourceType: "paperless", SourceID: "1",
			Title: "Doc 1", Fidelity: item.FidelityExact, DeepLink: "http://paperless.lan:8000/documents/1",
		},
	}); err != nil {
		t.Fatalf("seed item: %v", err)
	}

	before := getItem(t, router, "paperless:1")
	if before.Item.SourceDisplayName != "before" {
		t.Fatalf("expected initial source_display_name %q, got %q", "before", before.Item.SourceDisplayName)
	}

	saveConfig(t, cfgStore, &config.Config{
		Sources: map[string]config.Source{
			"paperless": {Plugin: "topos-plugin-paperless", BaseURL: "http://paperless.lan:8000", Token: "unused", DisplayName: "after"},
		},
		Webspaces: map[string]config.Webspace{"house-move": {Keywords: []string{"house-move"}}},
	})

	after := getItem(t, router, "paperless:1")
	if after.Item.SourceDisplayName != "after" {
		t.Fatalf("expected source_display_name to reflect the save on the SAME router instance with no restart, got %q", after.Item.SourceDisplayName)
	}
}

func getItem(t *testing.T, router http.Handler, id string) itemDetailResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/items/"+id, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/items/%s: expected 200, got %d: %s", id, rec.Code, rec.Body.String())
	}
	var resp itemDetailResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return resp
}

// TestLiveConfig_SourceRefreshHandler_NewlyAddedSourceIsImmediatelyRefreshable
// proves SourceRefreshHandler's own fix: a source added by a save is
// found — not 404'd as unknown — on the very next refresh request against
// the SAME router instance, since {name} is now checked against cfgStore
// fresh per request rather than a set frozen at Router construction.
func TestLiveConfig_SourceRefreshHandler_NewlyAddedSourceIsImmediatelyRefreshable(t *testing.T) {
	router, cfgStore, _ := liveConfigTestRouter(t, `
[webspaces.house-move]
keywords = ["house-move"]
`)

	req := httptest.NewRequest(http.MethodPost, "/api/sources/paperless/refresh", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assertErrorEnvelope(t, rec, http.StatusNotFound, "source_not_found")

	saveConfig(t, cfgStore, &config.Config{
		Sources: map[string]config.Source{
			"paperless": {Plugin: "topos-plugin-paperless", BaseURL: "http://paperless.lan:8000", Token: "unused"},
		},
		Webspaces: map[string]config.Webspace{"house-move": {Keywords: []string{"house-move"}}},
	})

	req2 := httptest.NewRequest(http.MethodPost, "/api/sources/paperless/refresh", nil)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	// fakeRefresher's zero-value Refresh returns a zero syncer.RunResult
	// and a nil error — the point under test is that the route no longer
	// 404s the newly added source, not what the fake refresher itself
	// returns.
	if rec2.Code == http.StatusNotFound {
		t.Fatalf("expected the newly added source to be found on the SAME router instance with no restart, got 404: %s", rec2.Body.String())
	}
}
