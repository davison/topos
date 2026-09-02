package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/item"
	"github.com/davison/topos/kernel/pluginhost"
)

// The agent mirror's date handling (QA's M3 finding on #76): live params
// narrow within the saved range, cannot widen it, and a malformed value
// is 400 by name — the same three claims the human routes pin.
func TestAgentStream_LiveDateParamsHonouredAndValidated(t *testing.T) {
	store := newTestStoreForHTTP(t)
	ctx := context.Background()
	if err := store.ReplaceWebspaceSourceItems(ctx, "house", "mail", []item.Item{
		agentTestItem("mail", "1", 100),
		agentTestItem("mail", "2", 200),
		agentTestItem("mail", "3", 300),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	cfg := &config.Config{
		Sources:   map[string]config.Source{"mail": {Plugin: "x", BaseURL: "http://x", Token: "t", Agent: config.AgentGrant{Read: true}}},
		Webspaces: map[string]config.Webspace{"house": {Keywords: []string{"x"}}},
	}
	prober := &fakeProber{healths: []pluginhost.SourceHealth{{Name: "mail", SourceType: "mail", DisplayName: "Mail", Reachable: true}}}
	router := newAgentTestRouter(store, cfg, &fakeFetcher{}, prober)

	get := func(path string) (streamResponse, int) {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var resp streamResponse
		if rec.Code == http.StatusOK {
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
		}
		return resp, rec.Code
	}

	resp, code := get("/agent/v1/webspaces/house/stream")
	if code != http.StatusOK || len(resp.Items) != 3 {
		t.Fatalf("baseline: %d items, code %d", len(resp.Items), code)
	}
	_, code = get("/agent/v1/webspaces/house/stream?from=not-a-date")
	if code != http.StatusBadRequest {
		t.Fatalf("malformed from on the agent route: %d, want 400", code)
	}
	// Timestamps 100/200/300 are all far before any calendar date — a
	// modern from excludes everything (the narrowing is honoured).
	resp, code = get("/agent/v1/webspaces/house/stream?from=2026-01-01")
	if code != http.StatusOK || len(resp.Items) != 0 {
		t.Fatalf("live from must narrow the agent stream: %d items, code %d", len(resp.Items), code)
	}
	// A saved range cannot be widened by a live param.
	cfg.Webspaces["house"] = config.Webspace{Keywords: []string{"x"}, DateFrom: "2026-01-01"}
	router2 := newAgentTestRouter(store, cfg, &fakeFetcher{}, prober)
	req := httptest.NewRequest(http.MethodGet, "/agent/v1/webspaces/house/stream?from=1970-01-01", nil)
	rec := httptest.NewRecorder()
	router2.ServeHTTP(rec, req)
	var widened streamResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &widened)
	if rec.Code != http.StatusOK || len(widened.Items) != 0 {
		t.Fatalf("a live param widened past the saved range on the agent route: %d items", len(widened.Items))
	}
}
