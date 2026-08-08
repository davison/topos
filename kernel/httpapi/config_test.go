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
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/davison/topos/kernel/config"
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
	r := chi.NewRouter()
	r.Get("/api/config", ConfigHandler(cfgStore))
	r.Put("/api/config", ConfigSaveHandler(cfgStore, &fakeApplier{}))
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
			// Neither a keywords fallback nor any match block — fails
			// Validate's first structural check.
			"broken": {},
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
// asserts the ONLY non-GET routes Router registers are PUT /api/config
// plus the two pre-existing refresh routes. A future PR adding any other
// mutating route fails this test outright, forcing that addition to be a
// deliberate, reviewed decision rather than a silent expansion of the
// kernel's one mutating surface.
func TestRoutesGuard_NonGetRoutesScopedToConfig(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "routes.go", nil, 0)
	if err != nil {
		t.Fatalf("parse routes.go: %v", err)
	}

	want := map[allowedNonGetRoute]bool{
		{"Put", "/api/config"}:                  true,
		{"Post", "/api/sources/{name}/refresh"}: true,
		{"Post", "/api/sync"}:                   true,
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
// source and asserts it imports no plugin-facing package and references
// no Match/Fetch call — the config write path must never grow a route
// into a plugin RPC beyond the existing read-only Describe path, which
// lives entirely outside this file.
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
		if strings.Contains(path, "pluginhost") || strings.Contains(path, "/sdk/") {
			t.Errorf("config.go imports %q — the config handlers must never reach a plugin (success criterion 4); the one existing exception (read-only Describe) lives outside this file", path)
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
