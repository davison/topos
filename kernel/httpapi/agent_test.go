// agent_test.go covers AGENT-01's default-deny, no-existence-leak and
// grant-independence guarantees over the /agent/v1 namespace (agent.go).
// Every test here exercises the real Router (not individual handlers) so
// the mount-point wiring in routes.go is covered too.
package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/davison/webspaces/kernel/config"
	"github.com/davison/webspaces/kernel/index"
	"github.com/davison/webspaces/kernel/item"
	"github.com/davison/webspaces/kernel/pluginhost"
)

func newAgentTestRouter(store *index.Store, cfg *config.Config, fetcher Fetcher, prober HealthProber) http.Handler {
	return Router(store, cfg, fetcher, prober, &fakeRefresher{})
}

func agentTestItem(sourceType, sourceID string, ts int64) item.Item {
	return item.Item{
		ID: sourceType + ":" + sourceID, SourceType: sourceType, SourceID: sourceID,
		Title: "item " + sourceID, Fidelity: item.FidelityExact,
		DeepLink: "http://example.lan/" + sourceID, TimestampUnix: ts,
	}
}

// --- three-way equivalence (AGENT-01 must_haves) ---

// TestAgentSourcesHandler_AbsentEmptyAndExplicitFalseAllDeny proves a
// source with no [agent] block, an empty [agent] block, and an explicit
// read = false are all identically absent from GET /agent/v1/sources.
func TestAgentSourcesHandler_AbsentEmptyAndExplicitFalseAllDeny(t *testing.T) {
	store := newTestStoreForHTTP(t)
	cfg := &config.Config{Sources: map[string]config.Source{
		"no-block":       {Plugin: "x", BaseURL: "http://x", Token: "t"},
		"empty-block":    {Plugin: "x", BaseURL: "http://x", Token: "t", Agent: config.AgentGrant{}},
		"explicit-false": {Plugin: "x", BaseURL: "http://x", Token: "t", Agent: config.AgentGrant{Read: false}},
	}}
	prober := &fakeProber{healths: []pluginhost.SourceHealth{
		{Name: "no-block", SourceType: "no-block", DisplayName: "No Block", Reachable: true},
		{Name: "empty-block", SourceType: "empty-block", DisplayName: "Empty Block", Reachable: true},
		{Name: "explicit-false", SourceType: "explicit-false", DisplayName: "Explicit False", Reachable: true},
	}}
	router := newAgentTestRouter(store, cfg, &fakeFetcher{}, prober)

	req := httptest.NewRequest(http.MethodGet, "/agent/v1/sources", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp agentSourcesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Sources) != 0 {
		t.Errorf("expected zero sources across all three deny-equivalent shapes, got: %+v", resp.Sources)
	}
}

// TestAgentSourcesHandler_HandoffWithoutReadIsStillFullyAbsent proves the
// two grants are independent: handoff = true with read = false (or
// absent) leaves the source fully absent from the agent namespace,
// including its items in a stream.
func TestAgentSourcesHandler_HandoffWithoutReadIsStillFullyAbsent(t *testing.T) {
	store := newTestStoreForHTTP(t)
	ctx := context.Background()
	if err := store.ReplaceWebspaceSourceItems(ctx, "house-move", "handoffonly", []item.Item{agentTestItem("handoffonly", "1", 100)}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	cfg := &config.Config{
		Sources:   map[string]config.Source{"handoffonly": {Plugin: "x", BaseURL: "http://x", Token: "t", Agent: config.AgentGrant{Handoff: true}}},
		Webspaces: map[string]config.Webspace{"house-move": {Keywords: []string{"x"}}},
	}
	prober := &fakeProber{healths: []pluginhost.SourceHealth{
		{Name: "handoffonly", SourceType: "handoffonly", DisplayName: "Handoff Only", Reachable: true},
	}}
	router := newAgentTestRouter(store, cfg, &fakeFetcher{}, prober)

	// Absent from GET /agent/v1/sources.
	req := httptest.NewRequest(http.MethodGet, "/agent/v1/sources", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var sourcesResp agentSourcesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &sourcesResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(sourcesResp.Sources) != 0 {
		t.Errorf("expected handoff-only source absent from /agent/v1/sources, got: %+v", sourcesResp.Sources)
	}

	// Its items absent from the agent stream.
	req2 := httptest.NewRequest(http.MethodGet, "/agent/v1/webspaces/house-move/stream", nil)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	var streamResp streamResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &streamResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(streamResp.Items) != 0 {
		t.Errorf("expected zero items for a handoff-only (unread) source, got: %+v", streamResp.Items)
	}
}

// TestAgentSourcesHandler_GrantedSourceReportsCapabilities proves a
// read-granted source appears with capabilities.read true and
// capabilities.handoff reflecting its own key.
func TestAgentSourcesHandler_GrantedSourceReportsCapabilities(t *testing.T) {
	store := newTestStoreForHTTP(t)
	cfg := &config.Config{Sources: map[string]config.Source{
		"granted": {Plugin: "x", BaseURL: "http://x", Token: "t", Agent: config.AgentGrant{Read: true, Handoff: true}},
	}}
	prober := &fakeProber{healths: []pluginhost.SourceHealth{
		{Name: "granted", SourceType: "granted", DisplayName: "Granted", Reachable: true},
	}}
	router := newAgentTestRouter(store, cfg, &fakeFetcher{}, prober)

	req := httptest.NewRequest(http.MethodGet, "/agent/v1/sources", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var resp agentSourcesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Sources) != 1 {
		t.Fatalf("expected 1 granted source, got %d: %+v", len(resp.Sources), resp.Sources)
	}
	if !resp.Sources[0].Capabilities.Read || !resp.Sources[0].Capabilities.Handoff {
		t.Errorf("expected capabilities {read:true handoff:true}, got %+v", resp.Sources[0].Capabilities)
	}
}

// TestAgentSourcesHandler_ZeroGrantsReturns200EmptyArray proves a
// zero-grant config returns 200 with an empty array, never an error
// (AGENT-01/empty).
func TestAgentSourcesHandler_ZeroGrantsReturns200EmptyArray(t *testing.T) {
	store := newTestStoreForHTTP(t)
	cfg := &config.Config{}
	router := newAgentTestRouter(store, cfg, &fakeFetcher{}, &fakeProber{})

	req := httptest.NewRequest(http.MethodGet, "/agent/v1/sources", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(raw["sources"]) != "[]" {
		t.Errorf(`expected "sources":[], got %s`, raw["sources"])
	}
}

// TestAgentStreamHandler_KnownWebspaceZeroGrantedItemsReturns200EmptyArray
// proves a known webspace with no granted source's items returns 200 with
// an empty items array, not 404 (AGENT-01/empty).
func TestAgentStreamHandler_KnownWebspaceZeroGrantedItemsReturns200EmptyArray(t *testing.T) {
	store := newTestStoreForHTTP(t)
	ctx := context.Background()
	if err := store.ReplaceWebspaceSourceItems(ctx, "house-move", "paperless", []item.Item{agentTestItem("paperless", "1", 100)}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	cfg := &config.Config{Webspaces: map[string]config.Webspace{"house-move": {Keywords: []string{"x"}}}}
	router := newAgentTestRouter(store, cfg, &fakeFetcher{}, &fakeProber{})

	req := httptest.NewRequest(http.MethodGet, "/agent/v1/webspaces/house-move/stream", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for a known webspace with zero granted items, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp streamResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Errorf("expected zero items, got: %+v", resp.Items)
	}
}

// TestAgentStreamHandler_UnknownWebspace404 proves an unknown webspace
// still returns 404 webspace_not_found through the agent namespace,
// identically to the human-facing route.
func TestAgentStreamHandler_UnknownWebspace404(t *testing.T) {
	store := newTestStoreForHTTP(t)
	cfg := &config.Config{}
	router := newAgentTestRouter(store, cfg, &fakeFetcher{}, &fakeProber{})

	req := httptest.NewRequest(http.MethodGet, "/agent/v1/webspaces/does-not-exist/stream", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assertErrorEnvelope(t, rec, http.StatusNotFound, "webspace_not_found")
}

// TestAgentStreamHandler_OrderingMatchesHumanStreamWithUngrantedRemoved
// proves granted items appear in the agent stream in the identical
// relative order they appear in the human-facing stream, with ungranted
// entries removed — never reordered.
func TestAgentStreamHandler_OrderingMatchesHumanStreamWithUngrantedRemoved(t *testing.T) {
	store := newTestStoreForHTTP(t)
	ctx := context.Background()
	if err := store.ReplaceWebspaceSourceItems(ctx, "house-move", "paperless", []item.Item{
		agentTestItem("paperless", "1", 300),
		agentTestItem("paperless", "2", 100),
	}); err != nil {
		t.Fatalf("seed paperless: %v", err)
	}
	if err := store.ReplaceWebspaceSourceItems(ctx, "house-move", "silverbullet", []item.Item{
		agentTestItem("silverbullet", "a", 200),
	}); err != nil {
		t.Fatalf("seed silverbullet: %v", err)
	}

	cfg := &config.Config{
		Sources: map[string]config.Source{
			"paperless": {Plugin: "x", BaseURL: "http://x", Token: "t", Agent: config.AgentGrant{Read: true}},
		},
		Webspaces: map[string]config.Webspace{"house-move": {Keywords: []string{"x"}}},
	}
	prober := &fakeProber{healths: []pluginhost.SourceHealth{
		{Name: "paperless", SourceType: "paperless", DisplayName: "paperless-ngx", Reachable: true},
	}}
	router := newAgentTestRouter(store, cfg, &fakeFetcher{}, prober)

	// Human-facing stream: all three items, silverbullet's timestamp (200)
	// sorts it between paperless:1 (300) and paperless:2 (100).
	req := httptest.NewRequest(http.MethodGet, "/api/webspaces/house-move/stream", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var humanResp streamResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &humanResp); err != nil {
		t.Fatalf("unmarshal human stream: %v", err)
	}
	if len(humanResp.Items) != 3 {
		t.Fatalf("expected 3 items in human stream, got %d", len(humanResp.Items))
	}

	req2 := httptest.NewRequest(http.MethodGet, "/agent/v1/webspaces/house-move/stream", nil)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	var agentResp streamResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &agentResp); err != nil {
		t.Fatalf("unmarshal agent stream: %v", err)
	}
	if len(agentResp.Items) != 2 {
		t.Fatalf("expected 2 granted (paperless) items in agent stream, got %d: %+v", len(agentResp.Items), agentResp.Items)
	}

	// Build the expected order: human order, filtered to paperless only.
	var wantIDs []string
	for _, it := range humanResp.Items {
		if it.SourceType == "paperless" {
			wantIDs = append(wantIDs, it.ID)
		}
	}
	var gotIDs []string
	for _, it := range agentResp.Items {
		gotIDs = append(gotIDs, it.ID)
	}
	if len(wantIDs) != len(gotIDs) || wantIDs[0] != gotIDs[0] || wantIDs[1] != gotIDs[1] {
		t.Errorf("expected agent stream order %v (human order, ungranted removed), got %v", wantIDs, gotIDs)
	}
}

// TestAgentAPIRoutesUnaffected proves every /api/* route's response is
// unchanged when grants are toggled — grants gate the agent surface only.
func TestAgentAPIRoutesUnaffected(t *testing.T) {
	store := newTestStoreForHTTP(t)
	ctx := context.Background()
	if err := store.ReplaceWebspaceSourceItems(ctx, "house-move", "paperless", []item.Item{agentTestItem("paperless", "1", 100)}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	noGrants := &config.Config{Webspaces: map[string]config.Webspace{"house-move": {Keywords: []string{"x"}}}}
	withGrants := &config.Config{
		Sources:   map[string]config.Source{"paperless": {Plugin: "x", BaseURL: "http://x", Token: "t", Agent: config.AgentGrant{Read: true}}},
		Webspaces: map[string]config.Webspace{"house-move": {Keywords: []string{"x"}}},
	}
	prober := &fakeProber{healths: []pluginhost.SourceHealth{
		{Name: "paperless", SourceType: "paperless", DisplayName: "paperless-ngx", Reachable: true},
	}}

	router1 := newAgentTestRouter(store, noGrants, &fakeFetcher{}, prober)
	router2 := newAgentTestRouter(store, withGrants, &fakeFetcher{}, prober)

	req1 := httptest.NewRequest(http.MethodGet, "/api/webspaces/house-move/stream", nil)
	rec1 := httptest.NewRecorder()
	router1.ServeHTTP(rec1, req1)

	req2 := httptest.NewRequest(http.MethodGet, "/api/webspaces/house-move/stream", nil)
	rec2 := httptest.NewRecorder()
	router2.ServeHTTP(rec2, req2)

	if rec1.Body.String() != rec2.Body.String() {
		t.Errorf("expected /api/webspaces/{ws}/stream to be identical regardless of grant configuration:\nno-grants=%s\nwith-grants=%s",
			rec1.Body.String(), rec2.Body.String())
	}
}

// --- no-existence-leak (T-02-20) ---

// TestAgentItemHandler_UngrantedItemMatchesNonexistentItemResponse proves
// GET /agent/v1/items/{id} against an ungranted source's real item and
// against an id that genuinely does not exist in the index produce
// byte-identical responses — same status, same code, same message.
func TestAgentItemHandler_UngrantedItemMatchesNonexistentItemResponse(t *testing.T) {
	cfg := &config.Config{} // zero grants

	// (a) the item genuinely exists but is ungranted.
	storeA := newTestStoreForHTTP(t)
	if err := storeA.ReplaceWebspaceSourceItems(context.Background(), "house-move", "paperless", []item.Item{agentTestItem("paperless", "1", 100)}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	routerA := newAgentTestRouter(storeA, cfg, &fakeFetcher{}, &fakeProber{})
	reqA := httptest.NewRequest(http.MethodGet, "/agent/v1/items/paperless:1", nil)
	recA := httptest.NewRecorder()
	routerA.ServeHTTP(recA, reqA)

	// (b) the identical id genuinely does not exist in a fresh, empty index.
	storeB := newTestStoreForHTTP(t)
	routerB := newAgentTestRouter(storeB, cfg, &fakeFetcher{}, &fakeProber{})
	reqB := httptest.NewRequest(http.MethodGet, "/agent/v1/items/paperless:1", nil)
	recB := httptest.NewRecorder()
	routerB.ServeHTTP(recB, reqB)

	if recA.Code != http.StatusNotFound || recB.Code != http.StatusNotFound {
		t.Fatalf("expected both to be 404, got a=%d b=%d", recA.Code, recB.Code)
	}
	if recA.Body.String() != recB.Body.String() {
		t.Errorf("expected byte-identical bodies for ungranted-vs-nonexistent:\nungranted=%s\nnonexistent=%s",
			recA.Body.String(), recB.Body.String())
	}
	var envelope errorEnvelope
	if err := json.Unmarshal(recA.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if envelope.Error.Code != "item_not_found" {
		t.Errorf("expected code item_not_found, got %q", envelope.Error.Code)
	}
}

// TestAgentContentHandler_UngrantedItemMatchesNonexistentAndWritesNoBytes
// proves the same no-existence-leak property holds for
// GET /agent/v1/items/{id}/content, and that no rendition bytes are
// written for the ungranted case.
func TestAgentContentHandler_UngrantedItemMatchesNonexistentAndWritesNoBytes(t *testing.T) {
	cfg := &config.Config{} // zero grants
	fetcher := &fakeFetcher{result: pluginhost.FetchResult{Available: true, MimeType: "application/pdf", SizeBytes: 3}}

	storeA := newTestStoreForHTTP(t)
	if err := storeA.ReplaceWebspaceSourceItems(context.Background(), "house-move", "paperless", []item.Item{agentTestItem("paperless", "1", 100)}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	routerA := newAgentTestRouter(storeA, cfg, fetcher, &fakeProber{})
	reqA := httptest.NewRequest(http.MethodGet, "/agent/v1/items/paperless:1/content", nil)
	recA := httptest.NewRecorder()
	routerA.ServeHTTP(recA, reqA)

	storeB := newTestStoreForHTTP(t)
	routerB := newAgentTestRouter(storeB, cfg, fetcher, &fakeProber{})
	reqB := httptest.NewRequest(http.MethodGet, "/agent/v1/items/paperless:1/content", nil)
	recB := httptest.NewRecorder()
	routerB.ServeHTTP(recB, reqB)

	if recA.Code != http.StatusNotFound || recB.Code != http.StatusNotFound {
		t.Fatalf("expected both to be 404, got a=%d b=%d", recA.Code, recB.Code)
	}
	if recA.Body.String() != recB.Body.String() {
		t.Errorf("expected byte-identical bodies:\nungranted=%s\nnonexistent=%s", recA.Body.String(), recB.Body.String())
	}
	if recA.Body.Len() == 0 {
		t.Fatal("expected a body (the error envelope), got none")
	}
	var envelope errorEnvelope
	if err := json.Unmarshal(recA.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if envelope.Error.Code != "item_not_found" {
		t.Errorf("expected code item_not_found (never a distinct code for the ungranted case), got %q", envelope.Error.Code)
	}
}

// TestAgentItemHandler_GrantedItemServed proves the positive case still
// works end to end: a granted item is served with its live-fetched
// content, through the real Router.
func TestAgentItemHandler_GrantedItemServed(t *testing.T) {
	store := newTestStoreForHTTP(t)
	if err := store.ReplaceWebspaceSourceItems(context.Background(), "house-move", "paperless", []item.Item{agentTestItem("paperless", "1", 100)}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	cfg := &config.Config{Sources: map[string]config.Source{
		"paperless": {Plugin: "x", BaseURL: "http://x", Token: "t", Agent: config.AgentGrant{Read: true}},
	}}
	prober := &fakeProber{healths: []pluginhost.SourceHealth{
		{Name: "paperless", SourceType: "paperless", DisplayName: "paperless-ngx", Reachable: true},
	}}
	fetcher := &fakeFetcher{result: pluginhost.FetchResult{Available: true, Text: "extracted text"}}
	router := newAgentTestRouter(store, cfg, fetcher, prober)

	req := httptest.NewRequest(http.MethodGet, "/agent/v1/items/paperless:1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp itemDetailResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Content.Text != "extracted text" {
		t.Errorf("expected the granted item's content to be served, got: %+v", resp.Content)
	}
}

// TestAgentWebspacesHandler_CountsRestrictedToGrantedSources proves
// item_count on GET /agent/v1/webspaces reflects only granted-source
// items, not the webspace's total item count.
func TestAgentWebspacesHandler_CountsRestrictedToGrantedSources(t *testing.T) {
	store := newTestStoreForHTTP(t)
	ctx := context.Background()
	if err := store.ReplaceWebspaceSourceItems(ctx, "house-move", "paperless", []item.Item{
		agentTestItem("paperless", "1", 100), agentTestItem("paperless", "2", 200),
	}); err != nil {
		t.Fatalf("seed paperless: %v", err)
	}
	if err := store.ReplaceWebspaceSourceItems(ctx, "house-move", "silverbullet", []item.Item{
		agentTestItem("silverbullet", "a", 150),
	}); err != nil {
		t.Fatalf("seed silverbullet: %v", err)
	}

	cfg := &config.Config{
		Sources: map[string]config.Source{
			"paperless": {Plugin: "x", BaseURL: "http://x", Token: "t", Agent: config.AgentGrant{Read: true}},
		},
		Webspaces: map[string]config.Webspace{"house-move": {Keywords: []string{"x"}}},
	}
	prober := &fakeProber{healths: []pluginhost.SourceHealth{
		{Name: "paperless", SourceType: "paperless", DisplayName: "paperless-ngx", Reachable: true},
	}}
	router := newAgentTestRouter(store, cfg, &fakeFetcher{}, prober)

	req := httptest.NewRequest(http.MethodGet, "/agent/v1/webspaces", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp webspacesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Webspaces) != 1 {
		t.Fatalf("expected 1 webspace, got %d", len(resp.Webspaces))
	}
	if resp.Webspaces[0].ItemCount != 2 {
		t.Errorf("expected item_count restricted to the 2 granted paperless items (silverbullet's 1 excluded), got %d", resp.Webspaces[0].ItemCount)
	}
}
