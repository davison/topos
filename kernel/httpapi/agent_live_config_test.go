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
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
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

// --- TestAgentGuard_EveryHandlerResolvesConfigPerRequest support (AST) ---
//
// This mirrors contract_test.go's own AST-scanning precedent
// (nonGetRoutesInFile / TestContract_MutatingRoutesAreConfigScoped):
// token.NewFileSet + parser.ParseFile + ast.Inspect over the plain source
// filename, run from the package directory go test already sets as its
// working directory.

// astParseAgentGo parses kernel/httpapi/agent.go's current source.
func astParseAgentGo(t *testing.T) *ast.File {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "agent.go", nil, 0)
	if err != nil {
		t.Fatalf("parse agent.go: %v", err)
	}
	return f
}

// funcDeclsEndingIn returns every top-level (non-method) function
// declaration in f whose name ends in suffix.
func funcDeclsEndingIn(f *ast.File, suffix string) []*ast.FuncDecl {
	var out []*ast.FuncDecl
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Recv != nil {
			continue
		}
		if len(fd.Name.Name) >= len(suffix) && fd.Name.Name[len(fd.Name.Name)-len(suffix):] == suffix {
			out = append(out, fd)
		}
	}
	return out
}

// funcDeclNamed returns the top-level (non-method) function declaration in
// f named name, or nil if none exists.
func funcDeclNamed(f *ast.File, name string) *ast.FuncDecl {
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Recv != nil {
			continue
		}
		if fd.Name.Name == name {
			return fd
		}
	}
	return nil
}

// paramCount returns fd's total parameter count, expanding grouped
// parameter names (e.g. `a, b string`) to their individual count.
func paramCount(fd *ast.FuncDecl) int {
	if fd.Type.Params == nil {
		return 0
	}
	count := 0
	for _, field := range fd.Type.Params.List {
		if len(field.Names) == 0 {
			count++
			continue
		}
		count += len(field.Names)
	}
	return count
}

// hasParamOfType reports whether fd declares any parameter of type
// *pkg.typeName (e.g. *config.Config).
func hasParamOfType(fd *ast.FuncDecl, pkg, typeName string) bool {
	if fd.Type.Params == nil {
		return false
	}
	for _, field := range fd.Type.Params.List {
		if isStarSelector(field.Type, pkg, typeName) {
			return true
		}
	}
	return false
}

// isStarSelector reports whether expr is exactly *pkg.typeName.
func isStarSelector(expr ast.Expr, pkg, typeName string) bool {
	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == pkg && sel.Sel.Name == typeName
}

// isExpandedCall reports whether expr is a call whose selector name is
// Expanded — the store's config-resolution method (config.Store.Expanded).
func isExpandedCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	return sel.Sel.Name == "Expanded"
}

// countExpandedCalls walks node and counts every call expression whose
// selector name is Expanded.
func countExpandedCalls(node ast.Node) int {
	count := 0
	ast.Inspect(node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Expanded" {
			count++
		}
		return true
	})
	return count
}

// returnedFuncLit returns the func literal fd's body returns (the
// http.HandlerFunc closure every agent handler constructor builds), or
// nil if fd's body returns no func literal.
func returnedFuncLit(fd *ast.FuncDecl) *ast.FuncLit {
	var lit *ast.FuncLit
	for _, stmt := range fd.Body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if !ok {
			continue
		}
		for _, res := range ret.Results {
			if fl, ok := res.(*ast.FuncLit); ok {
				lit = fl
			}
		}
	}
	return lit
}

func sortedBoolKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedFuncDeclKeys(m map[string]*ast.FuncDecl) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TestAgentGuard_EveryHandlerResolvesConfigPerRequest pins the invariant
// Task 1 established as a structural, AST-enforced guarantee: a future
// handler added to agent.go with the pre-fix shape (a resolved
// *config.Config parameter, or a config resolved inside MountAgentRoutes)
// cannot silently reintroduce CR-01's authorization-bypass window. A
// behavioural test alone would not catch a NEW handler added with the old
// shape — that is exactly how this defect survived 07-02's /api/* gap
// closure, so this guard is structural rather than behavioural.
func TestAgentGuard_EveryHandlerResolvesConfigPerRequest(t *testing.T) {
	f := astParseAgentGo(t)

	// Assertion 1: the handler set is enumerated, not sampled. A new
	// handler is not forbidden, but whoever adds one must extend this
	// guard to cover it.
	wantHandlers := map[string]bool{
		"agentSourcesHandler":   true,
		"agentWebspacesHandler": true,
		"agentStreamHandler":    true,
		"agentItemHandler":      true,
		"agentRenditionHandler": true,
	}
	gotHandlers := make(map[string]*ast.FuncDecl)
	for _, fd := range funcDeclsEndingIn(f, "Handler") {
		gotHandlers[fd.Name.Name] = fd
	}
	if len(gotHandlers) != len(wantHandlers) {
		t.Fatalf("expected exactly the handler set %v declared in agent.go, got %d: %v — a new handler is not forbidden, but whoever adds one must extend this guard to cover it",
			sortedBoolKeys(wantHandlers), len(gotHandlers), sortedFuncDeclKeys(gotHandlers))
	}
	for name := range wantHandlers {
		if _, ok := gotHandlers[name]; !ok {
			t.Errorf("expected handler %q to be declared in agent.go — a new handler is not forbidden, but whoever adds or renames one must extend this guard to cover it", name)
		}
	}
	for name := range gotHandlers {
		if !wantHandlers[name] {
			t.Errorf("unexpected handler %q declared in agent.go and not covered by this guard — a new handler is not forbidden, but whoever adds one must extend this guard to cover it", name)
		}
	}

	// Assertion 2: no handler receives a resolved config. A handler
	// holding a config resolved before the request began (CR-01) serves a
	// grant set that a revocation cannot reach until the process
	// restarts.
	for name, fd := range gotHandlers {
		if hasParamOfType(fd, "config", "Config") {
			t.Errorf("%s takes a parameter of type *config.Config (CR-01): a handler holding a config resolved before the request began serves a grant set that a revocation cannot reach until the process restarts", name)
		}
	}

	// Assertion 3: grantedSources keeps the resolved-config parameter —
	// the store is threaded as far as the handler closure and no further,
	// so the "resolve once, at the top of the request" discipline stays
	// visible at every call site rather than hiding inside a helper.
	grantedSourcesDecl := funcDeclNamed(f, "grantedSources")
	if grantedSourcesDecl == nil {
		t.Fatal("expected grantedSources to be declared in agent.go")
	}
	if n := paramCount(grantedSourcesDecl); n != 1 {
		t.Errorf("expected grantedSources to declare exactly 1 parameter, got %d", n)
	} else if !hasParamOfType(grantedSourcesDecl, "config", "Config") {
		t.Error("expected grantedSources' one parameter to be of type *config.Config — the store is threaded as far as the handler closure and no further, so the 'resolve once, at the top of the request' discipline stays visible at every call site rather than hiding inside a helper")
	}

	// Assertion 4: nothing is resolved at mount time. A config resolved in
	// MountAgentRoutes is a snapshot every handler registered below it
	// would silently inherit — precisely the defect this guard pins.
	mountDecl := funcDeclNamed(f, "MountAgentRoutes")
	if mountDecl == nil {
		t.Fatal("expected MountAgentRoutes to be declared in agent.go")
	}
	if n := countExpandedCalls(mountDecl.Body); n != 0 {
		t.Errorf("expected MountAgentRoutes' body to contain zero Expanded() calls, got %d — a config resolved in this function is a snapshot every handler registered below it would silently inherit, which is precisely the defect this guard pins", n)
	}

	// Assertion 5: resolved once, first, per request. Two resolutions in
	// one request body would let a save landing between them produce a
	// response mixing a pre-save granted set with post-save display names
	// or webspace filters.
	for name, fd := range gotHandlers {
		lit := returnedFuncLit(fd)
		if lit == nil {
			t.Errorf("%s: expected a returned http.HandlerFunc closure (func literal)", name)
			continue
		}
		if len(lit.Body.List) == 0 {
			t.Errorf("%s: expected the returned closure to have at least one statement", name)
			continue
		}
		first := lit.Body.List[0]
		assign, ok := first.(*ast.AssignStmt)
		if !ok || len(assign.Rhs) != 1 || !isExpandedCall(assign.Rhs[0]) {
			t.Errorf("%s: expected the closure's first statement to assign from a single call to the store's config-resolution method (Expanded) — a resolution anywhere but first, or a second resolution later in the body, would let a save landing mid-request produce a response mixing a pre-save granted set with post-save display names or webspace filters", name)
		}
		if n := countExpandedCalls(lit); n != 1 {
			t.Errorf("%s: expected exactly one Expanded() call in the closure body, got %d — a second resolution reintroduces CR-01's defect at a smaller time scale", name, n)
		}
	}
}
