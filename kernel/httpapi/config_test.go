// config_test.go pins the HTTP-layer half of 07-01-PLAN.md Task 3's
// invariants: the secret round trip (D-05) as seen through the actual
// GET /api/config response body, the clobber guard (D-03) as a real 409
// through the router, the marshal-write-reload-write idempotency fixed
// point at the API surface, validation reuse (D-09) — the message is the
// live validator's own string, never a hardcoded sentence — and the
// config-scoped-mutation guard (success criterion 4): an AST scan proving
// routes.go registers no non-GET route beyond the config routes and the
// two pre-existing refresh routes, and that config.go itself references
// no plugin call.
package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/pluginhost"
)

// newTestConfigStoreFromFile writes contents to a real temp config.toml
// and wraps it in a *config.Store via config.NewStore — unlike
// config.NewStoreForTesting (used elsewhere in this package for handlers
// that never call Save/Reload), ConfigSaveHandler's Save re-reads the file
// from disk, so these tests need a genuine backing path.
func newTestConfigStoreFromFile(t *testing.T, contents string) *config.Store {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	s, err := config.NewStore(path)
	if err != nil {
		t.Fatalf("config.NewStore: %v", err)
	}
	return s
}

func newConfigTestRouter(cfgStore *config.Store) http.Handler {
	return newConfigTestRouterWithApplier(cfgStore, &fakeApplier{})
}

func newConfigTestRouterWithApplier(cfgStore *config.Store, applier Applier) http.Handler {
	r := chi.NewRouter()
	r.Get("/api/config", ConfigHandler(cfgStore))
	r.Put("/api/config", ConfigSaveHandler(cfgStore, applier))
	r.Post("/api/config/reload", ConfigReloadHandler(cfgStore, applier))
	return r
}

func doConfigRequest(t *testing.T, router http.Handler, method, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, "/api/config", nil)
	} else {
		req = httptest.NewRequest(method, "/api/config", strings.NewReader(body))
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// TestConfigHandler_NeverReturnsResolvedSecretValue is the HTTP-layer half
// of T-07-01: GET /api/config's response body must carry the literal
// ${VAR} reference an operator authored, never the resolved secret value
// the kernel process holds in memory.
func TestConfigHandler_NeverReturnsResolvedSecretValue(t *testing.T) {
	t.Setenv("TEST_HTTP_TOKEN_SENTINEL", "sentinel-secret-value")

	cfgStore := newTestConfigStoreFromFile(t, `
[sources.paperless]
plugin = "topos-plugin-paperless"
base_url = "http://paperless.lan:8000"
token = "${TEST_HTTP_TOKEN_SENTINEL}"

[webspaces.house-move]
keywords = ["house-move"]
`)
	router := newConfigTestRouter(cfgStore)

	rec := doConfigRequest(t, router, http.MethodGet, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if strings.Contains(body, "sentinel-secret-value") {
		t.Fatalf("GET /api/config response body contains the RESOLVED secret value — D-05 violated: %s", body)
	}
	if !strings.Contains(body, `${TEST_HTTP_TOKEN_SENTINEL}`) {
		t.Fatalf("expected the literal ${VAR} reference in the response body, got: %s", body)
	}

	var resp configResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.EnvVars["TEST_HTTP_TOKEN_SENTINEL"] != true {
		t.Errorf("expected env_vars to report TEST_HTTP_TOKEN_SENTINEL as set (true, never the value itself), got %v", resp.EnvVars)
	}
}

// TestConfigHandler_ZeroWebspacesSerializesEmptyObjectsNotNull is the
// wire-level proof for 07-12-PLAN.md Task 1 / 07-UAT.md G-07-4's KERN-08
// empty edge: a config with zero [sources.*] and zero [webspaces.*]
// blocks must answer GET /api/config with EMPTY JSON objects for both
// collections, never null. Asserts both on the raw body text (a lenient
// decoder happily turns JSON null into a nil Go map that reads as length
// zero — an assertion that only checked length would pass against the
// unfixed defect) and on the decoded shape (a non-nil map of length zero),
// so neither half of the assertion can be satisfied by accident. Against
// the pre-Task-1 applyDefaults, this fails: the response body contains
// "sources":null and "webspaces":null (see 07-12-SUMMARY.md for the
// recorded RED output), and Object.keys(null) is exactly what threw in the
// SPA's root route (.planning/debug/root-empty-state-service-error.md).
func TestConfigHandler_ZeroWebspacesSerializesEmptyObjectsNotNull(t *testing.T) {
	cfgStore := newTestConfigStoreFromFile(t, `
[server]
listen = "127.0.0.1:7777"
`)
	router := newConfigTestRouter(cfgStore)

	rec := doConfigRequest(t, router, http.MethodGet, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if strings.Contains(body, `"webspaces":null`) {
		t.Fatalf("response body serializes webspaces as null (07-UAT.md G-07-4) — the SPA's root route iterates this field directly via Object.keys, which throws on null: %s", body)
	}
	if strings.Contains(body, `"sources":null`) {
		t.Fatalf("response body serializes sources as null: %s", body)
	}

	var decoded map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	cfgField, ok := decoded["config"].(map[string]any)
	if !ok {
		t.Fatalf("expected the config field to decode as an object, got %T", decoded["config"])
	}
	webspaces, ok := cfgField["webspaces"].(map[string]any)
	if !ok {
		t.Fatalf("expected config.webspaces to decode as a non-nil object (empty map), got %T: %v — a decoder silently turning null into a nil map would mask the defect this test guards against", cfgField["webspaces"], cfgField["webspaces"])
	}
	if len(webspaces) != 0 {
		t.Errorf("expected zero webspaces, got %d: %v", len(webspaces), webspaces)
	}
	sources, ok := cfgField["sources"].(map[string]any)
	if !ok {
		t.Fatalf("expected config.sources to decode as a non-nil object (empty map), got %T", cfgField["sources"])
	}
	if len(sources) != 0 {
		t.Errorf("expected zero sources, got %d: %v", len(sources), sources)
	}
}

// TestConfigHandler_WebspaceOmittingCollectionsSerializesEmptyNotNull is
// 07-12-PLAN.md Task 1's UI-12 null edge: a webspace declaring only
// `keywords` (omitting `sources` and `match`) must answer with an empty
// array for `sources` and an empty object for `match`, never null — every
// client reader of these fields (the chip row, the add-source picker,
// every config-edit.ts helper) reads them directly. Same raw-body-plus-
// decoded-shape double assertion as the sibling test above.
func TestConfigHandler_WebspaceOmittingCollectionsSerializesEmptyNotNull(t *testing.T) {
	cfgStore := newTestConfigStoreFromFile(t, `
[webspaces.house-move]
keywords = ["house-move"]
`)
	router := newConfigTestRouter(cfgStore)

	rec := doConfigRequest(t, router, http.MethodGet, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if strings.Contains(body, `"sources":null`) {
		t.Fatalf("expected the webspace's sources collection to serialize as an empty array, not null: %s", body)
	}
	if strings.Contains(body, `"match":null`) {
		t.Fatalf("expected the webspace's match collection to serialize as an empty object, not null: %s", body)
	}

	var decoded map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	cfgField, ok := decoded["config"].(map[string]any)
	if !ok {
		t.Fatalf("expected the config field to decode as an object, got %T", decoded["config"])
	}
	webspaces, ok := cfgField["webspaces"].(map[string]any)
	if !ok {
		t.Fatalf("expected config.webspaces to decode as an object, got %T", cfgField["webspaces"])
	}
	ws, ok := webspaces["house-move"].(map[string]any)
	if !ok {
		t.Fatalf("expected webspaces.house-move to decode as an object, got %T", webspaces["house-move"])
	}
	sources, ok := ws["sources"].([]any)
	if !ok {
		t.Fatalf("expected webspaces.house-move.sources to decode as a non-nil array (empty), got %T: %v", ws["sources"], ws["sources"])
	}
	if len(sources) != 0 {
		t.Errorf("expected zero sources for this webspace, got %d: %v", len(sources), sources)
	}
	match, ok := ws["match"].(map[string]any)
	if !ok {
		t.Fatalf("expected webspaces.house-move.match to decode as a non-nil object (empty), got %T: %v", ws["match"], ws["match"])
	}
	if len(match) != 0 {
		t.Errorf("expected zero match blocks for this webspace, got %d: %v", len(match), match)
	}
}

// TestConfigSaveHandler_StaleHashReturns409AndFileUnchanged is the
// HTTP-layer half of D-03: a PUT carrying a base_hash that no longer
// matches the on-disk file's current hash is rejected with 409
// config_changed_on_disk, and the out-of-band edit's content survives
// untouched.
func TestConfigSaveHandler_StaleHashReturns409AndFileUnchanged(t *testing.T) {
	cfgStore := newTestConfigStoreFromFile(t, `
[webspaces.house-move]
keywords = ["house-move"]
`)
	router := newConfigTestRouter(cfgStore)
	staleHash := cfgStore.Hash()

	outOfBand := []byte(`
[webspaces.house-move]
keywords = ["changed-out-of-band"]
`)
	if err := os.WriteFile(cfgStore.Path(), outOfBand, 0o600); err != nil {
		t.Fatalf("simulate out-of-band edit: %v", err)
	}

	putBody := `{"base_hash":"` + staleHash + `","config":{"webspaces":{"house-move":{"keywords":["house-move"],"filter":["boiler"]}}}}`
	rec := doConfigRequest(t, router, http.MethodPut, putBody)

	assertErrorEnvelope(t, rec, http.StatusConflict, "config_changed_on_disk")

	got, err := os.ReadFile(cfgStore.Path())
	if err != nil {
		t.Fatalf("read config after rejected save: %v", err)
	}
	if !bytes.Equal(got, outOfBand) {
		t.Fatalf("expected the rejected save to leave no trace — file must be exactly the out-of-band edit's content:\ngot=%s\nwant=%s", got, outOfBand)
	}
}

// TestConfigSaveHandler_EchoedGETLeavesFileByteIdentical proves the
// PUT-echoes-GET idempotency fixed point at the API surface: starting
// from an already-canonical file, a GET followed immediately by a PUT of
// that exact document must leave config.toml byte-identical to what a
// second such round trip would also produce — a save-of-a-save is a true
// no-op, not a slow drift.
func TestConfigSaveHandler_EchoedGETLeavesFileByteIdentical(t *testing.T) {
	cfgStore := newTestConfigStoreFromFile(t, `
[webspaces.house-move]
keywords = ["house-move"]
filter = ["boiler"]
`)
	router := newConfigTestRouter(cfgStore)

	echoGETThroughPUT := func() []byte {
		t.Helper()
		getRec := doConfigRequest(t, router, http.MethodGet, "")
		if getRec.Code != http.StatusOK {
			t.Fatalf("GET expected 200, got %d: %s", getRec.Code, getRec.Body.String())
		}
		var getResp configResponse
		if err := json.Unmarshal(getRec.Body.Bytes(), &getResp); err != nil {
			t.Fatalf("unmarshal GET response: %v", err)
		}

		putBody, err := json.Marshal(configSaveRequest{BaseHash: getResp.Hash, Config: getResp.Config})
		if err != nil {
			t.Fatalf("marshal PUT body: %v", err)
		}
		putRec := doConfigRequest(t, router, http.MethodPut, string(putBody))
		if putRec.Code != http.StatusOK {
			t.Fatalf("PUT expected 200, got %d: %s", putRec.Code, putRec.Body.String())
		}

		onDisk, err := os.ReadFile(cfgStore.Path())
		if err != nil {
			t.Fatalf("read config after PUT: %v", err)
		}
		return onDisk
	}

	// First round trip canonicalizes the hand-authored fixture; the SECOND
	// round trip is the one under test — comparing the two proves the
	// GET->PUT cycle is a genuine no-op once the file is already canonical,
	// not merely "PUT doesn't crash".
	afterFirst := echoGETThroughPUT()
	afterSecond := echoGETThroughPUT()

	if !bytes.Equal(afterFirst, afterSecond) {
		t.Fatalf("expected config.toml to be byte-identical after a second GET->PUT echo round trip:\nfirst=%s\nsecond=%s", afterFirst, afterSecond)
	}
}

// TestConfigSaveHandler_ApplyingSameFilterTermTwiceLeavesOneEntry proves
// that saving an identical filter value through two successive PUTs never
// accumulates a duplicate — the second PUT's base_hash is the first PUT's
// own response hash, exactly the sequence a client performing two
// "apply this same filter" actions in a row would produce.
func TestConfigSaveHandler_ApplyingSameFilterTermTwiceLeavesOneEntry(t *testing.T) {
	cfgStore := newTestConfigStoreFromFile(t, `
[webspaces.house-move]
keywords = ["house-move"]
`)
	router := newConfigTestRouter(cfgStore)

	applyFilter := func(baseHash string) configResponse {
		t.Helper()
		reqBody, err := json.Marshal(configSaveRequest{
			BaseHash: baseHash,
			Config: &config.Config{
				Webspaces: map[string]config.Webspace{
					"house-move": {Keywords: []string{"house-move"}, Filter: []string{"boiler"}},
				},
			},
		})
		if err != nil {
			t.Fatalf("marshal PUT body: %v", err)
		}
		rec := doConfigRequest(t, router, http.MethodPut, string(reqBody))
		if rec.Code != http.StatusOK {
			t.Fatalf("PUT expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp configResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal PUT response: %v", err)
		}
		return resp
	}

	first := applyFilter(cfgStore.Hash())
	second := applyFilter(first.Hash)

	filter := second.Config.Webspaces["house-move"].Filter
	if len(filter) != 1 || filter[0] != "boiler" {
		t.Fatalf("expected exactly one filter entry after applying the same term twice, got %v", filter)
	}
}

// TestConfigSaveHandler_InvalidConfigReturns422WithValidatorMessageVerbatim
// is the HTTP-layer half of D-09: a config that fails
// (*config.Config).Validate must be rejected 422 config_invalid, with the
// response's error message being EXACTLY the validator's own string — the
// test asserts against a live call to Validate, never a hardcoded
// sentence, so the assertion cannot drift from the validator it pins.
func TestConfigSaveHandler_InvalidConfigReturns422WithValidatorMessageVerbatim(t *testing.T) {
	cfgStore := newTestConfigStoreFromFile(t, `
[webspaces.house-move]
keywords = ["house-move"]
`)
	router := newConfigTestRouter(cfgStore)

	invalid := &config.Config{
		Webspaces: map[string]config.Webspace{
			// A non-empty sources allowlist disqualifies this from D-20's
			// empty-webspace-shell exemption (07-11-PLAN.md;
			// Webspace.IsEmptyShell requires ALL of keywords/sources/match
			// to be empty) — neither a keywords fallback nor any match
			// block — fails Validate's first structural check. A bare
			// `{}` (no allowlist either) would now be a legitimate shell
			// and load cleanly, which is no longer a usable "invalid"
			// fixture for this test.
			"broken": {Sources: []string{"placeholder"}},
		},
	}
	wantErr := invalid.Validate(nil)
	if wantErr == nil {
		t.Fatal("test fixture itself must be invalid — Validate(nil) unexpectedly returned nil")
	}

	reqBody, err := json.Marshal(configSaveRequest{BaseHash: cfgStore.Hash(), Config: invalid})
	if err != nil {
		t.Fatalf("marshal PUT body: %v", err)
	}
	rec := doConfigRequest(t, router, http.MethodPut, string(reqBody))

	assertErrorEnvelope(t, rec, http.StatusUnprocessableEntity, "config_invalid")

	var envelope errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal error envelope: %v", err)
	}
	if envelope.Error.Message != wantErr.Error() {
		t.Errorf("expected the validator's own message verbatim:\ngot=%q\nwant=%q", envelope.Error.Message, wantErr.Error())
	}
}

// TestConfigSaveHandler_UnknownKeysReturns409 is the HTTP-layer half of
// the lossless-rewrite prohibition: a config.toml carrying a key the
// Config struct does not model must refuse the save with 409
// config_has_unknown_keys through the real router, matching
// kernel/config/store_test.go's equivalent Store-level assertion.
func TestConfigSaveHandler_UnknownKeysReturns409(t *testing.T) {
	cfgStore := newTestConfigStoreFromFile(t, `
[webspaces.house-move]
keywords = ["house-move"]

[a_stray_table]
some_key = "some_value"
`)
	router := newConfigTestRouter(cfgStore)

	putBody := `{"base_hash":"` + cfgStore.Hash() + `","config":{"webspaces":{"house-move":{"keywords":["house-move"],"filter":["boiler"]}}}}`
	rec := doConfigRequest(t, router, http.MethodPut, putBody)

	assertErrorEnvelope(t, rec, http.StatusConflict, "config_has_unknown_keys")

	var envelope errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal error envelope: %v", err)
	}
	if !strings.Contains(envelope.Error.Message, "a_stray_table") {
		t.Errorf("expected the error message to name the unrecognised key, got: %q", envelope.Error.Message)
	}
}

// TestConfigSaveHandler_MalformedBodyReturns400InvalidRequest pins the
// last remaining error-code row: a body that isn't valid JSON, or is
// missing "config", is rejected 400 invalid_request before Save is ever
// called.
func TestConfigSaveHandler_MalformedBodyReturns400InvalidRequest(t *testing.T) {
	cfgStore := newTestConfigStoreFromFile(t, `
[webspaces.house-move]
keywords = ["house-move"]
`)
	router := newConfigTestRouter(cfgStore)

	for name, body := range map[string]string{
		"not JSON at all":      "{not json",
		"missing config field": `{"base_hash":"abc"}`,
	} {
		t.Run(name, func(t *testing.T) {
			rec := doConfigRequest(t, router, http.MethodPut, body)
			assertErrorEnvelope(t, rec, http.StatusBadRequest, "invalid_request")
		})
	}
}

// allowedNonGetRoute is one (method, path) pair TestRoutesGuard_* permits
// routes.go to register outside GET.
type allowedNonGetRoute struct{ method, path string }

// TestRoutesGuard_NonGetRoutesScopedToConfig is the AST half of success
// criterion 4 (T-07-05): parses kernel/httpapi/routes.go's source and
// asserts the ONLY non-GET routes Router registers are PUT /api/config,
// the two pre-existing refresh routes, and (08-03-PLAN.md Task 3) the two
// mutating whatsapp-link routes — a raw-subprocess surface outside the
// go-plugin gRPC handshake, not a SourcePlugin RPC, and still scoped to
// configuration/linking exactly like every other route on this allowlist.
// A future PR adding any other mutating route fails this test outright,
// forcing that addition to be a deliberate, reviewed decision rather than
// a silent expansion of the kernel's mutating surface.
func TestRoutesGuard_NonGetRoutesScopedToConfig(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "routes.go", nil, 0)
	if err != nil {
		t.Fatalf("parse routes.go: %v", err)
	}

	want := map[allowedNonGetRoute]bool{
		{"Put", "/api/config"}:                            true,
		{"Post", "/api/config/reload"}:                    true,
		{"Post", "/api/config/describe-plugin"}:           true,
		{"Post", "/api/sources/{name}/refresh"}:           true,
		{"Post", "/api/sync"}:                             true,
		{"Post", "/api/config/whatsapp-link"}:             true,
		{"Delete", "/api/config/whatsapp-link/{session}"}: true,
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
		if !ok || recv.Name != "r" {
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

	if len(found) != len(want) {
		t.Fatalf("expected exactly %d non-GET route(s) registered in routes.go, got %d: %+v", len(want), len(found), found)
	}
	for _, rt := range found {
		if !want[rt] {
			t.Errorf("unexpected non-GET route registered: %s %s — adding a new mutating route must be a deliberate, reviewed decision (success criterion 4); update this guard's allowlist if intentional", rt.method, rt.path)
		}
	}
}

// TestConfigHandlersGuard_NoPluginCallOtherThanDescribe is the AST second
// half of success criterion 4 / T-07-05: parses kernel/httpapi/config.go's
// source and asserts it references no Match/Fetch call anywhere — the
// config write path must never grow a route into a plugin RPC beyond the
// existing read-only Describe path (07-02-PLAN.md Task 3's
// DescribePluginType, defined entirely in kernel/pluginhost/host.go, not
// this file).
//
// Unlike 07-01, this guard no longer forbids config.go from importing
// kernel/pluginhost outright — Task 3 makes that import necessary
// (DiscoverBinaries/DescribePluginType for the "+" chip picker). The
// import itself is no longer the boundary this test polices; what
// config.go's OWN source calls is. A direct import of the generated
// protobuf/sdk package remains forbidden: config.go has no legitimate
// reason to construct an RPC request/response type itself.
func TestConfigHandlersGuard_NoPluginCallOtherThanDescribe(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "config.go", nil, parser.ImportsOnly|parser.ParseComments)
	if err != nil {
		t.Fatalf("parse config.go imports: %v", err)
	}
	for _, imp := range f.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		if strings.Contains(path, "/sdk/") {
			t.Errorf("config.go imports %q — the config handlers must never construct a plugin RPC type directly (success criterion 4)", path)
		}
	}

	full, err := parser.ParseFile(fset, "config.go", nil, 0)
	if err != nil {
		t.Fatalf("parse config.go: %v", err)
	}
	forbidden := map[string]bool{"Fetch": true, "Match": true}
	ast.Inspect(full, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if forbidden[sel.Sel.Name] {
			t.Errorf("config.go references %q — the config write path must never reach a plugin's Match or Fetch RPC (success criterion 4, T-07-05)", sel.Sel.Name)
		}
		return true
	})
}

// TestDescribePluginTypeGuard_ReachesNoRPCBeyondDescribe is the AST proof
// pinning T-07-10/PLUG-02: kernel/pluginhost.DescribePluginType's own
// function body (not launch(), which it wraps) must reference no
// Match/Fetch selector — the trial-launch path must never become a
// general plugin-invocation surface for request-supplied input. Parses
// kernel/pluginhost/host.go directly (a sibling package directory), per
// 07-02-PLAN.md Task 3's own declared test location for this assertion.
func TestDescribePluginTypeGuard_ReachesNoRPCBeyondDescribe(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "../pluginhost/host.go", nil, 0)
	if err != nil {
		t.Fatalf("parse kernel/pluginhost/host.go: %v", err)
	}

	var fn *ast.FuncDecl
	ast.Inspect(f, func(n ast.Node) bool {
		d, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}
		if d.Name.Name == "DescribePluginType" {
			fn = d
			return false
		}
		return true
	})
	if fn == nil {
		t.Fatal("DescribePluginType not found in kernel/pluginhost/host.go")
	}

	forbidden := map[string]bool{"Fetch": true, "Match": true}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if forbidden[sel.Sel.Name] {
			t.Errorf("DescribePluginType references %q — the trial-launch path must reach no RPC beyond the Describe call it inherits from launch() (PLUG-02, T-07-10)", sel.Sel.Name)
		}
		return true
	})
}

// doPathRequest issues a request against an arbitrary path (unlike
// doConfigRequest, hardcoded to "/api/config") — needed for
// POST /api/config/reload, mounted alongside it on the same test router.
func doPathRequest(t *testing.T, router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// TestConfigReloadHandler_ValidHandEditAppliesAndReturnsNewDocument is the
// HTTP-layer half of D-08: a hand-edited, valid config.toml reaches the
// running kernel through POST /api/config/reload — the response carries
// the newly reloaded document and hash, and the applier is invoked so the
// running kernel catches up too.
func TestConfigReloadHandler_ValidHandEditAppliesAndReturnsNewDocument(t *testing.T) {
	cfgStore := newTestConfigStoreFromFile(t, `
[webspaces.house-move]
keywords = ["house-move"]
`)
	applier := &fakeApplier{}
	router := newConfigTestRouterWithApplier(cfgStore, applier)

	handEdited := []byte(`
[webspaces.house-move]
keywords = ["hand-edited"]
`)
	if err := os.WriteFile(cfgStore.Path(), handEdited, 0o600); err != nil {
		t.Fatalf("simulate hand-edit: %v", err)
	}

	rec := doPathRequest(t, router, http.MethodPost, "/api/config/reload", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp configResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := resp.Config.Webspaces["house-move"].Keywords; len(got) != 1 || got[0] != "hand-edited" {
		t.Errorf("expected the reloaded document to reflect the hand-edit, got %v", got)
	}
	if resp.Hash != cfgStore.Hash() {
		t.Errorf("expected the response hash to match the reloaded Store's own hash")
	}
	if !applier.called {
		t.Error("expected ConfigReloadHandler to call Applier.Apply on a successful reload")
	}
}

// TestConfigReloadHandler_InvalidFileReturns422AndKeepsLastGoodConfig is
// the HTTP-layer half of D-08's "the kernel never dies on a bad reload":
// a syntactically valid but semantically invalid hand-edit is rejected
// 422 config_invalid with the loader's own message, and a subsequent
// GET /api/config still returns the PREVIOUS good document and hash,
// completely unchanged.
func TestConfigReloadHandler_InvalidFileReturns422AndKeepsLastGoodConfig(t *testing.T) {
	cfgStore := newTestConfigStoreFromFile(t, `
[webspaces.house-move]
keywords = ["house-move"]
`)
	router := newConfigTestRouter(cfgStore)

	beforeHash := cfgStore.Hash()
	beforeGET := doConfigRequest(t, router, http.MethodGet, "")
	var before configResponse
	if err := json.Unmarshal(beforeGET.Body.Bytes(), &before); err != nil {
		t.Fatalf("unmarshal GET before reload: %v", err)
	}

	// Syntactically valid TOML, semantically invalid config: a webspace
	// declaring a sources allowlist but neither a keywords fallback nor
	// any match block. A bare `keywords = []` with no allowlist would now
	// be a legitimate D-20 empty webspace shell (07-11-PLAN.md) and load
	// cleanly, so the allowlist is what keeps this fixture invalid.
	invalid := []byte(`
[webspaces.house-move]
keywords = []
sources = ["placeholder"]
`)
	if err := os.WriteFile(cfgStore.Path(), invalid, 0o600); err != nil {
		t.Fatalf("simulate invalid hand-edit: %v", err)
	}

	rec := doPathRequest(t, router, http.MethodPost, "/api/config/reload", "")
	assertErrorEnvelope(t, rec, http.StatusUnprocessableEntity, "config_invalid")

	if cfgStore.Hash() != beforeHash {
		t.Errorf("expected the Store's hash to be untouched by a failed reload, got %q want %q", cfgStore.Hash(), beforeHash)
	}

	afterGET := doConfigRequest(t, router, http.MethodGet, "")
	if afterGET.Code != http.StatusOK {
		t.Fatalf("expected GET /api/config to still succeed after a failed reload, got %d", afterGET.Code)
	}
	var after configResponse
	if err := json.Unmarshal(afterGET.Body.Bytes(), &after); err != nil {
		t.Fatalf("unmarshal GET after failed reload: %v", err)
	}
	if after.Hash != before.Hash {
		t.Errorf("expected GET /api/config to still report the previous good hash after a failed reload, got %q want %q", after.Hash, before.Hash)
	}
	gotKeywords := after.Config.Webspaces["house-move"].Keywords
	if len(gotKeywords) != 1 || gotKeywords[0] != "house-move" {
		t.Errorf("expected GET /api/config to still report the previous good document after a failed reload, got keywords %v", gotKeywords)
	}
}

// TestConfigReloadHandler_ApplyFailureReturns500ApplyFailed proves the
// reload route surfaces an apply failure the same way the save route
// does — never a silent 200 when the file reloaded fine but the running
// kernel could not fully catch up.
func TestConfigReloadHandler_ApplyFailureReturns500ApplyFailed(t *testing.T) {
	cfgStore := newTestConfigStoreFromFile(t, `
[webspaces.house-move]
keywords = ["house-move"]
`)
	router := newConfigTestRouterWithApplier(cfgStore, &fakeApplier{err: errApplyBoom})

	rec := doPathRequest(t, router, http.MethodPost, "/api/config/reload", "")
	assertErrorEnvelope(t, rec, http.StatusInternalServerError, "apply_failed")
}

// TestRoutesGuard_NoLocalConfigSnapshot is the AST proof for
// 07-02-PLAN.md Task 2's own acceptance criterion: routes.go must declare
// no local *config.Config value at all — after this task, every mounted
// handler reads cfgStore fresh per request, so a future handler cannot
// reintroduce a captured boot-time snapshot (the identifier "cfg", which
// every prior snapshot in this file was named) without this test failing.
func TestRoutesGuard_NoLocalConfigSnapshot(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "routes.go", nil, 0)
	if err != nil {
		t.Fatalf("parse routes.go: %v", err)
	}
	ast.Inspect(f, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		if id.Name == "cfg" {
			t.Errorf("routes.go declares or references a local %q identifier — every mounted handler must read live config via cfgStore instead of a boot-time snapshot (07-02-PLAN.md Task 2)", id.Name)
		}
		return true
	})
}

// newPluginTypesTestRouter mounts only the two plugin-discovery routes —
// PluginTypesHandler and DescribePluginHandler are pure functions of
// dirs, needing no config.Store or index.Store at all. pluginsDir is
// wrapped as the TRUSTED directory only — every pre-existing caller of
// this helper predates the external tier and asserts trusted-directory
// behavior exclusively.
func newPluginTypesTestRouter(pluginsDir string) http.Handler {
	dirs := pluginhost.Dirs{Trusted: pluginsDir}
	r := chi.NewRouter()
	r.Get("/api/config/plugin-types", PluginTypesHandler(dirs))
	r.Post("/api/config/describe-plugin", DescribePluginHandler(dirs, hclog.NewNullLogger()))
	return r
}

// TestPluginTypesHandler_ReturnsSortedMockFreeList proves GET
// /api/config/plugin-types surfaces exactly the discovered, non-excluded
// binaries, sorted — the same guarantee kernel/pluginhost.DiscoverBinaries
// itself pins, exercised here through the real HTTP route. Both fixture
// binaries excluded from the picker catalog (mock and mockstrict, the
// browser-harness fixture plugin — quick task 260811-r5d) are written
// alongside real plugin types to pin the guarantee at the HTTP boundary
// the SPA actually consumes, not only at the package boundary.
func TestPluginTypesHandler_ReturnsSortedMockFreeList(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"topos-plugin-silverbullet", "topos-plugin-mock", "topos-plugin-mockstrict", "topos-plugin-paperless"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o755); err != nil {
			t.Fatalf("write fixture %s: %v", name, err)
		}
	}
	router := newPluginTypesTestRouter(dir)

	req := httptest.NewRequest(http.MethodGet, "/api/config/plugin-types", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp pluginTypesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := []string{"topos-plugin-paperless", "topos-plugin-silverbullet"}
	if len(resp.PluginTypes) != len(want) {
		t.Fatalf("expected %v, got %v", want, resp.PluginTypes)
	}
	for i := range want {
		if resp.PluginTypes[i] != want[i] {
			t.Fatalf("expected sorted, mock-free %v, got %v", want, resp.PluginTypes)
		}
	}
}

// TestPluginTypesHandler_ReturnsPluginTypeTiersAlongsidePluginTypes is
// Phase 11's own shape check (PLUG-06/07): GET /api/config/plugin-types
// returns BOTH the pre-existing plugin_types array of strings AND the new
// additive plugin_type_tiers object — a tier lookup table spanning every
// discovered binary in both directories, including a fixture name
// (topos-plugin-mock) excluded from plugin_types itself, proving
// PluginTypeTiers is sourced from DiscoverAllTiered (the security-
// authority listing), never the UI-policy-filtered DiscoverTiered.
func TestPluginTypesHandler_ReturnsPluginTypeTiersAlongsidePluginTypes(t *testing.T) {
	trustedDir := t.TempDir()
	for _, name := range []string{"topos-plugin-paperless", "topos-plugin-mock"} {
		if err := os.WriteFile(filepath.Join(trustedDir, name), []byte("x"), 0o755); err != nil {
			t.Fatalf("write trusted fixture %s: %v", name, err)
		}
	}
	externalDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(externalDir, "topos-plugin-example"), []byte("x"), 0o755); err != nil {
		t.Fatalf("write external fixture: %v", err)
	}

	dirs := pluginhost.Dirs{Trusted: trustedDir, External: externalDir}
	r := chi.NewRouter()
	r.Get("/api/config/plugin-types", PluginTypesHandler(dirs))
	router := r

	req := httptest.NewRequest(http.MethodGet, "/api/config/plugin-types", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp pluginTypesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	wantTypes := []string{"topos-plugin-example", "topos-plugin-paperless"}
	if len(resp.PluginTypes) != len(wantTypes) {
		t.Fatalf("expected plugin_types %v, got %v", wantTypes, resp.PluginTypes)
	}
	for i := range wantTypes {
		if resp.PluginTypes[i] != wantTypes[i] {
			t.Fatalf("expected plugin_types %v, got %v", wantTypes, resp.PluginTypes)
		}
	}

	wantTiers := map[string]string{
		"topos-plugin-paperless": "trusted",
		"topos-plugin-mock":      "trusted",
		"topos-plugin-example":   "external",
	}
	if len(resp.PluginTypeTiers) != len(wantTiers) {
		t.Fatalf("expected plugin_type_tiers %v, got %v", wantTiers, resp.PluginTypeTiers)
	}
	for name, wantTier := range wantTiers {
		if got := resp.PluginTypeTiers[name]; got != wantTier {
			t.Errorf("expected plugin_type_tiers[%q] = %q, got %q", name, wantTier, got)
		}
	}
}

// TestDescribePluginHandler_UnknownBinaryReturns404WithoutExecuting proves
// T-07-09: a request naming a binary DiscoverBinaries does not return is
// refused 404 plugin_binary_not_found — directory listing, never the
// caller-supplied name, is the authority over what may be launched.
func TestDescribePluginHandler_UnknownBinaryReturns404WithoutExecuting(t *testing.T) {
	dir := t.TempDir() // empty — nothing discoverable at all
	router := newPluginTypesTestRouter(dir)

	body := `{"plugin":"topos-plugin-does-not-exist","source":{}}`
	req := httptest.NewRequest(http.MethodPost, "/api/config/describe-plugin", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assertErrorEnvelope(t, rec, http.StatusNotFound, "plugin_binary_not_found")
}

// buildPaperlessPluginDir builds the repo's real paperless plugin binary
// fresh, once per test binary run, into a shared temp directory — the
// acceptance criterion "against the repo's own built plugins" (07-02-
// PLAN.md Task 3) requires a genuine, non-Signal, non-mock plugin type,
// and paperless is the simplest of the three remaining real types (no
// username/webmail fields silverbullet/proton also need beyond
// base_url/token).
var (
	paperlessPluginDirOnce sync.Once
	paperlessPluginDir     string
	paperlessPluginDirErr  error
)

func buildPaperlessPluginDir(t *testing.T) string {
	t.Helper()
	paperlessPluginDirOnce.Do(func() {
		out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/davison/topos").Output()
		if err != nil {
			paperlessPluginDirErr = fmt.Errorf("resolve module root: %w", err)
			return
		}
		root := strings.TrimSpace(string(out))

		dir, err := os.MkdirTemp("", "topos-httpapi-describe-test-*")
		if err != nil {
			paperlessPluginDirErr = err
			return
		}

		bin := filepath.Join(dir, "topos-plugin-paperless")
		cmd := exec.Command("go", "build", "-o", bin, "./plugins/paperless")
		cmd.Dir = root
		if buildOut, err := cmd.CombinedOutput(); err != nil {
			paperlessPluginDirErr = fmt.Errorf("build paperless plugin: %w\n%s", err, buildOut)
			return
		}

		paperlessPluginDir = dir
	})
	if paperlessPluginDirErr != nil {
		t.Fatalf("build paperless plugin fixture: %v", paperlessPluginDirErr)
	}
	return paperlessPluginDir
}

// TestDescribePluginHandler_RealPaperlessBinary_ReturnsMatchVocabulary is
// the acceptance criterion's own "against the repo's own built plugins"
// proof: paperless's Describe-declared match_vocabulary (["tags"]) comes
// back from present-but-unverified connection fields — DescribePluginType
// never opens a live connection to base_url before Describe answers
// (plugins/paperless/main.go only checks field presence) — and, since
// this handler has no config.Store dependency at all, there is
// structurally no [sources.*] block it could ever write.
func TestDescribePluginHandler_RealPaperlessBinary_ReturnsMatchVocabulary(t *testing.T) {
	dir := buildPaperlessPluginDir(t)
	router := newPluginTypesTestRouter(dir)

	body := `{"plugin":"topos-plugin-paperless","source":{"base_url":"http://paperless.example.test","token":"unverified"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/config/describe-plugin", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp describePluginResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.SourceType != "paperless" {
		t.Errorf("expected source_type %q, got %q", "paperless", resp.SourceType)
	}
	if len(resp.MatchVocabulary) != 1 || resp.MatchVocabulary[0] != "tags" {
		t.Errorf("expected match_vocabulary [\"tags\"], got %v", resp.MatchVocabulary)
	}
}

// buildMockPluginDir mirrors buildPaperlessPluginDir above, built fresh
// once per test binary run into a shared temp directory.
var (
	mockPluginDirOnce sync.Once
	mockPluginDir     string
	mockPluginDirErr  error
)

func buildMockPluginDir(t *testing.T) string {
	t.Helper()
	mockPluginDirOnce.Do(func() {
		out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/davison/topos").Output()
		if err != nil {
			mockPluginDirErr = fmt.Errorf("resolve module root: %w", err)
			return
		}
		root := strings.TrimSpace(string(out))

		dir, err := os.MkdirTemp("", "topos-httpapi-describe-mock-test-*")
		if err != nil {
			mockPluginDirErr = err
			return
		}

		bin := filepath.Join(dir, "topos-plugin-mock")
		cmd := exec.Command("go", "build", "-o", bin, "./plugins/mock")
		cmd.Dir = root
		if buildOut, err := cmd.CombinedOutput(); err != nil {
			mockPluginDirErr = fmt.Errorf("build mock plugin: %w\n%s", err, buildOut)
			return
		}

		mockPluginDir = dir
	})
	if mockPluginDirErr != nil {
		t.Fatalf("build mock plugin fixture: %v", mockPluginDirErr)
	}
	return mockPluginDir
}

// TestDescribePluginHandler_ExistingMockInstance_ReturnsMatchVocabulary is
// the regression test for the bug 07.1-04-PLAN.md discovered live: the
// "+" picker's one-step existing-instance add flow calls this exact route
// with an ALREADY-CONFIGURED instance's own plugin field — before the
// DiscoverAllBinaries split, "topos-plugin-mock" 404'd here even though it
// was genuinely present on disk and already running as a configured
// instance, because this handler shared DiscoverBinaries' UI-policy
// exclusion with the "+ New <plugin type>…" picker's OFFERED-types list
// (PluginTypesHandler). A mock instance must remain excluded from THAT
// list (TestPluginTypesHandler_ReturnsSortedMockFreeList, above) while
// still being describable here, since describing it is not "offering it
// as new" — it is resolving the vocabulary of a source that already
// legitimately exists in config.
func TestDescribePluginHandler_ExistingMockInstance_ReturnsMatchVocabulary(t *testing.T) {
	dir := buildMockPluginDir(t)
	router := newPluginTypesTestRouter(dir)

	// Confirm the setup precondition this regression depends on: mock is
	// still excluded from the OFFERED-types list even though it is about
	// to be described successfully below.
	typesReq := httptest.NewRequest(http.MethodGet, "/api/config/plugin-types", nil)
	typesRec := httptest.NewRecorder()
	router.ServeHTTP(typesRec, typesReq)
	var typesResp pluginTypesResponse
	if err := json.Unmarshal(typesRec.Body.Bytes(), &typesResp); err != nil {
		t.Fatalf("unmarshal plugin-types: %v", err)
	}
	for _, name := range typesResp.PluginTypes {
		if name == "topos-plugin-mock" {
			t.Fatalf("expected topos-plugin-mock to stay excluded from /api/config/plugin-types, got %v", typesResp.PluginTypes)
		}
	}

	body := `{"plugin":"topos-plugin-mock","source":{}}`
	req := httptest.NewRequest(http.MethodPost, "/api/config/describe-plugin", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (mock must remain describable even though it is excluded from the offered-types list), got %d: %s", rec.Code, rec.Body.String())
	}
	var resp describePluginResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.SourceType != "mock" {
		t.Errorf("expected source_type %q, got %q", "mock", resp.SourceType)
	}
	if len(resp.MatchVocabulary) != 1 || resp.MatchVocabulary[0] != "labels" {
		t.Errorf("expected match_vocabulary [\"labels\"], got %v", resp.MatchVocabulary)
	}
}
