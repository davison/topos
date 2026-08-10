// contract_test.go pins the agent-facing JSON envelope (AGENT-02) by
// exercising the real router (Router, not individual handlers) over a
// seeded temp-file index. See 01-04-PLAN.md Task 1.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/item"
	"github.com/davison/topos/kernel/pluginhost"
)

// idPattern pins the stable id shape {source_type}:{source_id} (AGENT-02).
var idPattern = regexp.MustCompile(`^[a-z0-9_-]+:.+$`)

// requiredProvenanceKeys are the six keys the published HTTP contract
// (docs/api.md) and 01-04-PLAN.md's <interfaces> block document on every
// item's provenance object.
var requiredProvenanceKeys = []string{
	"source_type", "source_system", "source_id", "plugin", "contract_version", "synced_at_unix",
}

var validFidelities = map[string]bool{"exact": true, "anchored": true, "conversation-only": true}

// testConfig returns a *config.Store wrapping a hand-built, empty Config —
// Rule 3 (07-01-PLAN.md Task 1): Router now takes a *config.Store rather
// than a bare *config.Config, so this fixture wraps via
// config.NewStoreForTesting rather than every call site below being
// rewritten to load a real TOML file.
func testConfig() *config.Store {
	return config.NewStoreForTesting(&config.Config{Webspaces: map[string]config.Webspace{}})
}

// fakeApplier is the shared test double for the Applier interface Router
// now requires (07-02-PLAN.md Task 1) — every test below that doesn't
// itself exercise apply behavior wires a no-op success. called records
// whether Apply was ever invoked, for tests proving a handler calls it at
// all (07-02-PLAN.md Task 2's reload route).
type fakeApplier struct {
	err    error
	called bool
}

func (f *fakeApplier) Apply(ctx context.Context) error {
	f.called = true
	return f.err
}

// errApplyBoom is a fixed sentinel error for tests proving a handler maps
// an Applier failure onto 500 apply_failed.
var errApplyBoom = errors.New("apply: boom")

func contractFixtureProvenance(sourceID string) map[string]string {
	return map[string]string{
		"source_type":      "paperless",
		"source_system":    "http://paperless.lan:8000",
		"source_id":        sourceID,
		"plugin":           "topos-plugin-paperless",
		"contract_version": "topos.v1",
	}
}

// contractFixtureItems returns three items: two sharing a primary
// timestamp but differing in secondary_timestamp_unix (exercises the
// tie-break), and one with an empty preview (exercises the "preview may be
// empty" case).
func contractFixtureItems() []item.Item {
	return []item.Item{
		{
			ID: "paperless:1", Source: "paperless", SourceType: "paperless", SourceID: "1",
			Title: "Same-day, added earlier", Preview: "a short preview",
			TimestampUnix: 1000, SecondaryTimestampUnix: 100,
			Fidelity: item.FidelityExact, DeepLink: "http://paperless.lan:8000/documents/1",
			Provenance: contractFixtureProvenance("1"),
		},
		{
			ID: "paperless:2", Source: "paperless", SourceType: "paperless", SourceID: "2",
			Title: "Same-day, added later", Preview: "",
			TimestampUnix: 1000, SecondaryTimestampUnix: 900,
			Fidelity: item.FidelityAnchored, DeepLink: "http://paperless.lan:8000/documents/2",
			Provenance: contractFixtureProvenance("2"),
		},
		{
			ID: "paperless:3", Source: "paperless", SourceType: "paperless", SourceID: "3",
			Title: "Newest", Preview: "another preview",
			TimestampUnix: 2000, SecondaryTimestampUnix: 1,
			Fidelity: item.FidelityConversationOnly, DeepLink: "http://paperless.lan:8000/documents/3",
			Provenance: contractFixtureProvenance("3"),
		},
	}
}

func TestContract_StreamEnvelope_IDsLinkAndProvenance(t *testing.T) {
	store := newTestStoreForHTTP(t)
	ctx := context.Background()
	if err := store.ReplaceWebspaceSourceItems(ctx, "house-move", "paperless", contractFixtureItems()); err != nil {
		t.Fatalf("seed: %v", err)
	}

	router, _ := Router(store, testConfig(), &fakeFetcher{}, &fakeProber{}, &fakeRefresher{}, &fakeApplier{}, &fakeSuspender{}, "testdata-unused-plugins-dir", hclog.NewNullLogger())
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
	if resp.SchemaVersion != schemaVersion {
		t.Errorf("expected schema_version %d, got %d", schemaVersion, resp.SchemaVersion)
	}
	if len(resp.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(resp.Items))
	}

	for _, it := range resp.Items {
		if !idPattern.MatchString(it.ID) {
			t.Errorf("item id %q does not match ^[a-z0-9_-]+:.+$", it.ID)
		}
		if it.ID != it.Source+":"+it.SourceID {
			t.Errorf("item id %q is not exactly source+\":\"+source_id (%q:%q)", it.ID, it.Source, it.SourceID)
		}
		if it.Source == "" {
			t.Errorf("item %q has an empty source (instance id)", it.ID)
		}
		if it.SourceDisplayName == "" {
			t.Errorf("item %q has an empty source_display_name", it.ID)
		}
		if !validFidelities[it.Link.Fidelity] {
			t.Errorf("item %q has unexpected link.fidelity %q", it.ID, it.Link.Fidelity)
		}
		if it.Link.URL == "" {
			t.Errorf("item %q has an empty link.url", it.ID)
		}
		for _, key := range requiredProvenanceKeys {
			if _, ok := it.Provenance[key]; !ok {
				t.Errorf("item %q provenance missing key %q; got %v", it.ID, key, it.Provenance)
			}
		}
	}
}

func TestContract_EmptyWebspaceReturns200EmptyArrayNotNull(t *testing.T) {
	store := newTestStoreForHTTP(t)
	ctx := context.Background()
	if err := store.ReplaceWebspaceSourceItems(ctx, "empty-space", "paperless", nil); err != nil {
		t.Fatalf("seed: %v", err)
	}

	router, _ := Router(store, testConfig(), &fakeFetcher{}, &fakeProber{}, &fakeRefresher{}, &fakeApplier{}, &fakeSuspender{}, "testdata-unused-plugins-dir", hclog.NewNullLogger())
	req := httptest.NewRequest(http.MethodGet, "/api/webspaces/empty-space/stream", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for a known-but-empty webspace, got %d: %s", rec.Code, rec.Body.String())
	}
	if regexp.MustCompile(`"items"\s*:\s*null`).MatchString(rec.Body.String()) {
		t.Fatalf("expected \"items\" to serialize as [], got null: %s", rec.Body.String())
	}

	var resp streamResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.SchemaVersion != schemaVersion {
		t.Errorf("expected schema_version %d on the empty-webspace response, got %d", schemaVersion, resp.SchemaVersion)
	}
	if resp.Items == nil {
		t.Error("expected a non-nil (empty) items slice")
	}
	if len(resp.Items) != 0 {
		t.Errorf("expected zero items, got %d", len(resp.Items))
	}
}

func TestContract_UnknownWebspace404(t *testing.T) {
	store := newTestStoreForHTTP(t)
	router, _ := Router(store, testConfig(), &fakeFetcher{}, &fakeProber{}, &fakeRefresher{}, &fakeApplier{}, &fakeSuspender{}, "testdata-unused-plugins-dir", hclog.NewNullLogger())

	req := httptest.NewRequest(http.MethodGet, "/api/webspaces/does-not-exist/stream", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assertErrorEnvelope(t, rec, http.StatusNotFound, "webspace_not_found")
}

func TestContract_UnknownItem404(t *testing.T) {
	store := newTestStoreForHTTP(t)
	router, _ := Router(store, testConfig(), &fakeFetcher{}, &fakeProber{}, &fakeRefresher{}, &fakeApplier{}, &fakeSuspender{}, "testdata-unused-plugins-dir", hclog.NewNullLogger())

	req := httptest.NewRequest(http.MethodGet, "/api/items/paperless:does-not-exist", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assertErrorEnvelope(t, rec, http.StatusNotFound, "item_not_found")
}

func TestContract_FetchFailureReturns502SourceUnavailable(t *testing.T) {
	store := newTestStoreForHTTP(t)
	seedTestItem(t, store, testItem())

	router, _ := Router(store, testConfig(), &fakeFetcher{
		err: fmt.Errorf("%w: connection refused", pluginhost.ErrSourceUnavailable),
	}, &fakeProber{}, &fakeRefresher{}, &fakeApplier{}, &fakeSuspender{}, "testdata-unused-plugins-dir", hclog.NewNullLogger())

	req := httptest.NewRequest(http.MethodGet, "/api/items/paperless:42", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assertErrorEnvelope(t, rec, http.StatusBadGateway, "source_unavailable")
}

// assertErrorEnvelope asserts the response matches the shared error
// envelope shape exactly: schema_version present, and an "error" object
// with exactly the two documented fields (code, message) — no more, no
// less.
func assertErrorEnvelope(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("expected status %d, got %d: %s", wantStatus, rec.Code, rec.Body.String())
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("unmarshal error body: %v", err)
	}
	if _, ok := raw["schema_version"]; !ok {
		t.Error("error response missing schema_version")
	}
	errRaw, ok := raw["error"]
	if !ok {
		t.Fatal("error response missing \"error\" object")
	}

	var errFields map[string]json.RawMessage
	if err := json.Unmarshal(errRaw, &errFields); err != nil {
		t.Fatalf("unmarshal error object: %v", err)
	}
	if len(errFields) != 2 {
		t.Errorf("expected error object to have exactly 2 fields (code, message), got %d: %v", len(errFields), errFields)
	}
	if _, ok := errFields["code"]; !ok {
		t.Error("error object missing \"code\"")
	}
	if _, ok := errFields["message"]; !ok {
		t.Error("error object missing \"message\"")
	}

	var envelope errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal error envelope: %v", err)
	}
	if envelope.Error.Code != wantCode {
		t.Errorf("expected error code %q, got %q", wantCode, envelope.Error.Code)
	}
}

// TestContract_StreamCalledTwiceIsByteIdentical pins both the ordering
// guarantee and the JSON key order: with no intervening sync, two
// successive calls to the same stream endpoint must produce byte-identical
// bodies.
func TestContract_StreamCalledTwiceIsByteIdentical(t *testing.T) {
	store := newTestStoreForHTTP(t)
	ctx := context.Background()
	if err := store.ReplaceWebspaceSourceItems(ctx, "house-move", "paperless", contractFixtureItems()); err != nil {
		t.Fatalf("seed: %v", err)
	}

	router, _ := Router(store, testConfig(), &fakeFetcher{}, &fakeProber{}, &fakeRefresher{}, &fakeApplier{}, &fakeSuspender{}, "testdata-unused-plugins-dir", hclog.NewNullLogger())

	req1 := httptest.NewRequest(http.MethodGet, "/api/webspaces/house-move/stream", nil)
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)

	req2 := httptest.NewRequest(http.MethodGet, "/api/webspaces/house-move/stream", nil)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	if rec1.Code != http.StatusOK || rec2.Code != http.StatusOK {
		t.Fatalf("expected both calls to return 200: first=%d second=%d", rec1.Code, rec2.Code)
	}
	if rec1.Body.String() != rec2.Body.String() {
		t.Errorf("expected byte-identical stream responses with no intervening sync:\nfirst=%s\nsecond=%s",
			rec1.Body.String(), rec2.Body.String())
	}
}

// nonGetRoutesInFile parses the named Go source file and returns every
// (method, path) pair registered via a call of the shape
// `<receiver>.<Method>("<path>", ...)` where method is anything other than
// Get/NotFound — the same AST-walk shape config_test.go's own
// TestRoutesGuard_NonGetRoutesScopedToConfig already established for
// routes.go, factored out here so TestContract_MutatingRoutesAreConfigScoped
// (below) can apply the identical scan to BOTH routes.go (the /api/*
// mutating-surface allowlist) and agent.go (the /agent/v1 GET-only
// guarantee) without duplicating the walk logic twice.
func nonGetRoutesInFile(t *testing.T, filename, receiverName string) []allowedNonGetRoute {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", filename, err)
	}

	var found []allowedNonGetRoute
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		recv, ok := sel.X.(*ast.Ident)
		if !ok || recv.Name != receiverName {
			return true
		}
		method := sel.Sel.Name
		if method == "Get" || method == "NotFound" {
			return true
		}
		if len(call.Args) == 0 {
			return true
		}
		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		path, err := strconv.Unquote(lit.Value)
		if err != nil {
			return true
		}
		found = append(found, allowedNonGetRoute{method: method, path: path})
		return true
	})
	return found
}

// TestContract_MutatingRoutesAreConfigScoped is the mechanical half of
// ROADMAP success criterion 4 (T-07-05), read directly against the real
// Router this file's other tests exercise: the complete set of non-GET
// routes Router registers must equal an explicit, named list — the config
// save, the config reload, the plugin-describe route, and the two
// pre-existing manual-refresh routes — so both an added AND a removed
// mutating route fail this test outright, forcing a deliberate decision
// and a threat-model row rather than a quiet append. This duplicates
// config_test.go's own TestRoutesGuard_NonGetRoutesScopedToConfig scan
// over routes.go by design (07-05-PLAN.md Task 3 names this exact test,
// in this exact file) rather than replacing it — that guard stays exactly
// as it is; this one ADDS the second assertion no route under /agent/v1
// is non-GET, closing the gap a routes.go-only scan can't see (every
// /agent/v1 route is registered inside agent.go's own MountAgentRoutes,
// not routes.go).
func TestContract_MutatingRoutesAreConfigScoped(t *testing.T) {
	want := map[allowedNonGetRoute]bool{
		{method: "Put", path: "/api/config"}:                  true,
		{method: "Post", path: "/api/config/reload"}:          true,
		{method: "Post", path: "/api/config/describe-plugin"}: true,
		{method: "Post", path: "/api/sources/{name}/refresh"}: true,
		{method: "Post", path: "/api/sync"}:                   true,
		// 08-03-PLAN.md Task 3 (D-01): the whatsapp-link session surface —
		// a raw-subprocess spawn outside the go-plugin gRPC handshake, not
		// a SourcePlugin RPC, still scoped to configuration/linking like
		// every other route on this allowlist. GET /api/config/whatsapp-
		// link/{session} (poll) is excluded here by construction: this
		// scan only ever records non-GET methods.
		{method: "Post", path: "/api/config/whatsapp-link"}:             true,
		{method: "Delete", path: "/api/config/whatsapp-link/{session}"}: true,
	}

	found := nonGetRoutesInFile(t, "routes.go", "r")
	if len(found) != len(want) {
		t.Fatalf("expected exactly %d non-GET route(s) registered in routes.go, got %d: %+v", len(want), len(found), found)
	}
	for _, rt := range found {
		if !want[rt] {
			t.Errorf("unexpected non-GET route registered in routes.go: %s %s — a new mutating route must be a deliberate, reviewed decision (success criterion 4) with its own threat-model row, never a silent append", rt.method, rt.path)
		}
	}

	// AGENT-11 (agent-initiated actions) is explicitly out of scope for v1
	// (docs/api.md's "What is not here yet") — this asserts that by
	// construction, not by convention: /agent/v1 registers no non-GET
	// route at all today, and a future PR adding one fails this test
	// outright rather than silently widening the agent surface into a
	// write path.
	agentFound := nonGetRoutesInFile(t, "agent.go", "r")
	if len(agentFound) != 0 {
		t.Errorf("expected zero non-GET routes registered in agent.go (the /agent/v1 namespace is GET-only in v1 — AGENT-11 is a v1.x concern), got %d: %+v", len(agentFound), agentFound)
	}
}
