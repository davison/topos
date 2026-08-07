// contract_test.go pins the agent-facing JSON envelope (AGENT-02) by
// exercising the real router (Router, not individual handlers) over a
// seeded temp-file index. See 01-04-PLAN.md Task 1.
package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

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

	router := Router(store, testConfig(), &fakeFetcher{}, &fakeProber{}, &fakeRefresher{})
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

	router := Router(store, testConfig(), &fakeFetcher{}, &fakeProber{}, &fakeRefresher{})
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
	router := Router(store, testConfig(), &fakeFetcher{}, &fakeProber{}, &fakeRefresher{})

	req := httptest.NewRequest(http.MethodGet, "/api/webspaces/does-not-exist/stream", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assertErrorEnvelope(t, rec, http.StatusNotFound, "webspace_not_found")
}

func TestContract_UnknownItem404(t *testing.T) {
	store := newTestStoreForHTTP(t)
	router := Router(store, testConfig(), &fakeFetcher{}, &fakeProber{}, &fakeRefresher{})

	req := httptest.NewRequest(http.MethodGet, "/api/items/paperless:does-not-exist", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assertErrorEnvelope(t, rec, http.StatusNotFound, "item_not_found")
}

func TestContract_FetchFailureReturns502SourceUnavailable(t *testing.T) {
	store := newTestStoreForHTTP(t)
	seedTestItem(t, store, testItem())

	router := Router(store, testConfig(), &fakeFetcher{
		err: fmt.Errorf("%w: connection refused", pluginhost.ErrSourceUnavailable),
	}, &fakeProber{}, &fakeRefresher{})

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

	router := Router(store, testConfig(), &fakeFetcher{}, &fakeProber{}, &fakeRefresher{})

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
