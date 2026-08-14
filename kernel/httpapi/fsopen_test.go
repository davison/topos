package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/item"
)

// recordingOpener is a test Opener stub that records the path (and, for
// CR-01's regression proof, the context) it was handed — or none, if never
// called — instead of execing xdg-open — the seam FilesystemOpenHandler's
// tests exercise every refusal branch through, so the gate does not depend
// on xdg-open being installed on the runner (12-01-PLAN.md's own
// <verification> requirement).
type recordingOpener struct {
	calledWith string
	calledCtx  context.Context
	called     bool
	err        error
}

func (o *recordingOpener) open(ctx context.Context, path string) error {
	o.called = true
	o.calledWith = path
	o.calledCtx = ctx
	return o.err
}

func newFsopenTestRouter(t *testing.T, cfg *config.Config, opener *recordingOpener) (http.Handler, *fsopenSeed) {
	t.Helper()
	store := newTestStoreForHTTP(t)
	r := chi.NewRouter()
	r.Post("/api/items/{id}/open", FilesystemOpenHandler(store, config.NewStoreForTesting(cfg), opener.open, hclog.NewNullLogger()))
	return r, &fsopenSeed{store: store}
}

// fsopenSeed is a thin helper wrapping *index.Store so each test can seed
// exactly the item(s) it needs without repeating ReplaceWebspaceSourceItems
// boilerplate.
type fsopenSeed struct {
	store interface {
		ReplaceWebspaceSourceItems(ctx context.Context, webspaceName, source string, items []item.Item) error
	}
}

func (s *fsopenSeed) put(t *testing.T, it item.Item) {
	t.Helper()
	if err := s.store.ReplaceWebspaceSourceItems(context.Background(), "fixture-webspace", it.Source, []item.Item{it}); err != nil {
		t.Fatalf("seed item: %v", err)
	}
}

func doOpen(router http.Handler, id string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/items/"+id+"/open", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) errorEnvelope {
	t.Helper()
	var envelope errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal error envelope: %v (body: %s)", err, rec.Body.String())
	}
	return envelope
}

// TestFilesystemOpen_HappyPathOpensTheJoinedAbsolutePath proves the opener
// receives the absolute join of the source's configured root with the
// item's own source_id, and the response is 200 with opened: true.
func TestFilesystemOpen_HappyPathOpensTheJoinedAbsolutePath(t *testing.T) {
	root := t.TempDir()
	// Fail-closed symlink resolution (CR-02, 12-06-PLAN.md Task 1) means the
	// containment check now reaches filepath.EvalSymlinks, which requires a
	// real fixture file rather than a merely lexical join — fixture
	// correction, not assertion loosening.
	if err := os.WriteFile(filepath.Join(root, "invoice.pdf"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	cfg := &config.Config{Sources: map[string]config.Source{
		"docs-folder": {Plugin: "topos-plugin-filesystem", Path: root},
	}}
	opener := &recordingOpener{}
	router, seed := newFsopenTestRouter(t, cfg, opener)
	seed.put(t, item.Item{
		ID: "docs-folder:invoice.pdf", Source: "docs-folder", SourceID: "invoice.pdf",
		Fidelity: item.FidelityExact, DeepLink: "file://" + filepath.Join(root, "invoice.pdf"),
	})

	rec := doOpen(router, "docs-folder:invoice.pdf")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !opener.called {
		t.Fatal("expected the opener to be called")
	}
	want := filepath.Join(root, "invoice.pdf")
	if opener.calledWith != want {
		t.Errorf("expected opener called with %q, got %q", want, opener.calledWith)
	}
	var resp fsopenResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Opened {
		t.Error("expected opened: true in the response")
	}
}

// TestFilesystemOpen_TildeInConfiguredPathIsExpandedBeforeTheJoin proves a
// leading "~" in the source's configured path is expanded before joining
// with the item's source_id.
func TestFilesystemOpen_TildeInConfiguredPathIsExpandedBeforeTheJoin(t *testing.T) {
	home, err := userHomeDirForTest(t)
	if err != nil {
		t.Skipf("could not resolve current user home dir: %v", err)
	}
	// Fail-closed symlink resolution (CR-02, 12-06-PLAN.md Task 1) means the
	// fixture directory must genuinely exist under the current user's home
	// rather than being a hardcoded, never-created path — fixture
	// correction, not assertion loosening. This test also incidentally
	// proves a home directory behind a symlink no longer breaks containment
	// (the WR-01 class).
	fixtureDir, err := os.MkdirTemp(home, "topos-fsopen-fixture-")
	if err != nil {
		t.Skipf("could not create fixture dir under home: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(fixtureDir) })
	if err := os.WriteFile(filepath.Join(fixtureDir, "invoice.pdf"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	cfg := &config.Config{Sources: map[string]config.Source{
		"docs-folder": {Plugin: "topos-plugin-filesystem", Path: "~/" + filepath.Base(fixtureDir)},
	}}
	opener := &recordingOpener{}
	router, seed := newFsopenTestRouter(t, cfg, opener)
	seed.put(t, item.Item{
		ID: "docs-folder:invoice.pdf", Source: "docs-folder", SourceID: "invoice.pdf",
		Fidelity: item.FidelityExact, DeepLink: "file:///anything",
	})

	rec := doOpen(router, "docs-folder:invoice.pdf")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	want := filepath.Join(fixtureDir, "invoice.pdf")
	if opener.calledWith != want {
		t.Errorf("expected the tilde expanded before the join: expected %q, got %q", want, opener.calledWith)
	}
}

// TestFilesystemOpen_PathEscapeAnswersInvalidPath proves a source_id
// containing ".." segments that escape the configured root produces
// invalid_path and the opener is never called.
func TestFilesystemOpen_PathEscapeAnswersInvalidPath(t *testing.T) {
	root := t.TempDir()
	cfg := &config.Config{Sources: map[string]config.Source{
		"docs-folder": {Plugin: "topos-plugin-filesystem", Path: root},
	}}
	opener := &recordingOpener{}
	router, seed := newFsopenTestRouter(t, cfg, opener)
	seed.put(t, item.Item{
		ID: "docs-folder:escape", Source: "docs-folder", SourceID: "../../etc/passwd",
		Fidelity: item.FidelityExact, DeepLink: "file:///anything",
	})

	rec := doOpen(router, "docs-folder:escape")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if envelope := decodeError(t, rec); envelope.Error.Code != "invalid_path" {
		t.Errorf("expected code invalid_path, got %q", envelope.Error.Code)
	}
	if opener.called {
		t.Error("expected the opener to never be called for a path-escaping source_id")
	}
}

// TestFilesystemOpen_UnknownItemAnswersItemNotFound proves an unknown item
// id produces item_not_found and the opener is never called.
func TestFilesystemOpen_UnknownItemAnswersItemNotFound(t *testing.T) {
	cfg := &config.Config{}
	opener := &recordingOpener{}
	router, _ := newFsopenTestRouter(t, cfg, opener)

	rec := doOpen(router, "does-not-exist:anything")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
	if envelope := decodeError(t, rec); envelope.Error.Code != "item_not_found" {
		t.Errorf("expected code item_not_found, got %q", envelope.Error.Code)
	}
	if opener.called {
		t.Error("expected the opener to never be called for an unknown item id")
	}
}

// TestFilesystemOpen_NonFileSchemeDeepLinkAnswersItemNotFound proves an
// item whose deep_link does not carry the file:// scheme produces
// item_not_found and the opener is never called — this is what keeps the
// route from being reachable for a non-filesystem item.
func TestFilesystemOpen_NonFileSchemeDeepLinkAnswersItemNotFound(t *testing.T) {
	root := t.TempDir()
	cfg := &config.Config{Sources: map[string]config.Source{
		"paperless": {Plugin: "x", BaseURL: "http://x", Token: "t", Path: root},
	}}
	opener := &recordingOpener{}
	router, seed := newFsopenTestRouter(t, cfg, opener)
	seed.put(t, item.Item{
		ID: "paperless:42", Source: "paperless", SourceID: "42",
		Fidelity: item.FidelityExact, DeepLink: "https://paperless.example/documents/42",
	})

	rec := doOpen(router, "paperless:42")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
	if envelope := decodeError(t, rec); envelope.Error.Code != "item_not_found" {
		t.Errorf("expected code item_not_found, got %q", envelope.Error.Code)
	}
	if opener.called {
		t.Error("expected the opener to never be called for a non-file:// deep link")
	}
}

// TestFilesystemOpen_OpenerErrorAnswersOpenFailed proves an opener that
// returns an error produces open_failed carrying the opener's own message.
func TestFilesystemOpen_OpenerErrorAnswersOpenFailed(t *testing.T) {
	root := t.TempDir()
	// Fail-closed symlink resolution (CR-02, 12-06-PLAN.md Task 1) means the
	// containment check now reaches filepath.EvalSymlinks, which requires a
	// real fixture file rather than a merely lexical join — fixture
	// correction, not assertion loosening.
	if err := os.WriteFile(filepath.Join(root, "invoice.pdf"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	cfg := &config.Config{Sources: map[string]config.Source{
		"docs-folder": {Plugin: "topos-plugin-filesystem", Path: root},
	}}
	opener := &recordingOpener{err: errors.New("xdg-open: no such file or handler")}
	router, seed := newFsopenTestRouter(t, cfg, opener)
	seed.put(t, item.Item{
		ID: "docs-folder:invoice.pdf", Source: "docs-folder", SourceID: "invoice.pdf",
		Fidelity: item.FidelityExact, DeepLink: "file:///anything",
	})

	rec := doOpen(router, "docs-folder:invoice.pdf")

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", rec.Code, rec.Body.String())
	}
	envelope := decodeError(t, rec)
	if envelope.Error.Code != "open_failed" {
		t.Errorf("expected code open_failed, got %q", envelope.Error.Code)
	}
	if envelope.Error.Message != opener.err.Error() {
		t.Errorf("expected the opener's own message %q, got %q", opener.err.Error(), envelope.Error.Message)
	}
}

// TestFilesystemOpen_SymlinkSwappedAfterIndexingAnswersInvalidPathAndNeverOpens
// proves CR-02 is closed at the exec site: a file indexed as legitimate and
// then swapped on disk for a symlink pointing outside the configured root
// is refused before opener is ever called.
func TestFilesystemOpen_SymlinkSwappedAfterIndexingAnswersInvalidPathAndNeverOpens(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on windows")
	}
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.pdf"), []byte("top secret"), 0o644); err != nil {
		t.Fatalf("write outside fixture: %v", err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.pdf"), filepath.Join(root, "invoice.pdf")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	cfg := &config.Config{Sources: map[string]config.Source{
		"docs-folder": {Plugin: "topos-plugin-filesystem", Path: root},
	}}
	opener := &recordingOpener{}
	router, seed := newFsopenTestRouter(t, cfg, opener)
	seed.put(t, item.Item{
		ID: "docs-folder:invoice.pdf", Source: "docs-folder", SourceID: "invoice.pdf",
		Fidelity: item.FidelityExact, DeepLink: "file://" + filepath.Join(root, "invoice.pdf"),
	})

	rec := doOpen(router, "docs-folder:invoice.pdf")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if envelope := decodeError(t, rec); envelope.Error.Code != "invalid_path" {
		t.Errorf("expected code invalid_path, got %q", envelope.Error.Code)
	}
	if opener.called {
		t.Error("expected the opener to never be called for a post-index symlink swap outside the root")
	}
}

// TestFilesystemOpen_VanishedFileAnswersItemNotFoundAndNeverOpens proves an
// indexed item whose file was deleted from an otherwise valid root answers
// 404 item_not_found rather than a false 200 opened: true, and the opener
// is never called.
func TestFilesystemOpen_VanishedFileAnswersItemNotFoundAndNeverOpens(t *testing.T) {
	root := t.TempDir()
	cfg := &config.Config{Sources: map[string]config.Source{
		"docs-folder": {Plugin: "topos-plugin-filesystem", Path: root},
	}}
	opener := &recordingOpener{}
	router, seed := newFsopenTestRouter(t, cfg, opener)
	seed.put(t, item.Item{
		ID: "docs-folder:invoice.pdf", Source: "docs-folder", SourceID: "invoice.pdf",
		Fidelity: item.FidelityExact, DeepLink: "file://" + filepath.Join(root, "invoice.pdf"),
	})
	// No fixture file is written — the item is indexed but its file has
	// since vanished from an otherwise valid root.

	rec := doOpen(router, "docs-folder:invoice.pdf")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
	if envelope := decodeError(t, rec); envelope.Error.Code != "item_not_found" {
		t.Errorf("expected code item_not_found, got %q", envelope.Error.Code)
	}
	if opener.called {
		t.Error("expected the opener to never be called for a vanished file")
	}
}

// userHomeDirForTest resolves the current user's home directory the same
// way expandHomeAndAbs does, for building the expected-path assertion in
// TestFilesystemOpen_TildeInConfiguredPathIsExpandedBeforeTheJoin.
func userHomeDirForTest(t *testing.T) (string, error) {
	t.Helper()
	expanded, err := expandHomeAndAbs("~")
	if err != nil {
		return "", err
	}
	return expanded, nil
}

// TestFilesystemOpen_OpenerContextIsDetachedFromTheRequestContext proves
// CR-01 is closed at the handler seam: the context handed to opener is NOT
// the HTTP request's own — cancelling the request's context after
// ServeHTTP has returned leaves the opener's captured context uncancelled.
// Under the pre-fix code, the captured context WAS the request context, so
// cancelling it would fail this assertion — that is the regression this
// test exists to catch.
func TestFilesystemOpen_OpenerContextIsDetachedFromTheRequestContext(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "invoice.pdf"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	cfg := &config.Config{Sources: map[string]config.Source{
		"docs-folder": {Plugin: "topos-plugin-filesystem", Path: root},
	}}
	opener := &recordingOpener{}
	router, seed := newFsopenTestRouter(t, cfg, opener)
	seed.put(t, item.Item{
		ID: "docs-folder:invoice.pdf", Source: "docs-folder", SourceID: "invoice.pdf",
		Fidelity: item.FidelityExact, DeepLink: "file://" + filepath.Join(root, "invoice.pdf"),
	})

	reqCtx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodPost, "/api/items/docs-folder:invoice.pdf/open", nil).WithContext(reqCtx)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !opener.called {
		t.Fatal("expected the opener to be called")
	}
	if opener.calledCtx == nil {
		t.Fatal("expected the opener to have been handed a non-nil context")
	}

	// Cancel the REQUEST's own context, after ServeHTTP has already
	// returned — this is exactly what net/http does the instant a handler
	// returns.
	cancel()

	if err := opener.calledCtx.Err(); err != nil {
		t.Errorf("expected the opener's captured context to report no error after the request's own context was cancelled, got %v", err)
	}
	select {
	case <-opener.calledCtx.Done():
		t.Error("expected the opener's captured context to not be done after the request's own context was cancelled")
	default:
	}
}

// TestNewXDGOpener_ChildIsNotBoundToACallerContext structurally proves two
// properties of newXDGOpener via go/ast (never file text, so a comment or
// a string can neither trip nor defeat this check): the returned function
// literal's context parameter is the blank identifier, making it
// structurally impossible for this implementation to pass it to anything;
// and no selector named CommandContext appears anywhere in that function's
// body. The same test also asserts the child is still constructed from the
// fixed "xdg-open" string literal, so T-12-01's no-configurable-binary
// property is asserted by the same test that asserts the CR-01 fix.
// Scoped to fsopen.go alone (never the whole package), so this test file's
// own mention of the forbidden constructor cannot make the guard
// self-invalidating. A behavioural test cannot reach this: the production
// opener execs a fixed binary that is not guaranteed present on a CI
// runner, and making it substitutable would trade away T-12-01.
func TestNewXDGOpener_ChildIsNotBoundToACallerContext(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fsopen.go", nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse fsopen.go: %v", err)
	}

	var fn *ast.FuncDecl
	for _, decl := range file.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok && fd.Name.Name == "newXDGOpener" {
			fn = fd
			break
		}
	}
	if fn == nil {
		t.Fatal("expected to find a newXDGOpener function declaration in fsopen.go")
	}

	var lit *ast.FuncLit
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if fl, ok := n.(*ast.FuncLit); ok && lit == nil {
			lit = fl
		}
		return true
	})
	if lit == nil {
		t.Fatal("expected newXDGOpener to return a function literal")
	}

	if lit.Type.Params == nil || len(lit.Type.Params.List) == 0 {
		t.Fatal("expected the returned function literal to declare parameters")
	}
	ctxParam := lit.Type.Params.List[0]
	if len(ctxParam.Names) != 1 || ctxParam.Names[0].Name != "_" {
		t.Errorf("expected the context parameter to be the blank identifier, got %v", ctxParam.Names)
	}

	var foundCommandContext bool
	var foundFixedBinary bool
	ast.Inspect(lit.Body, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == "CommandContext" {
			foundCommandContext = true
		}
		if bl, ok := n.(*ast.BasicLit); ok && bl.Kind == token.STRING && bl.Value == `"xdg-open"` {
			foundFixedBinary = true
		}
		return true
	})
	if foundCommandContext {
		t.Error("expected no CommandContext selector anywhere in newXDGOpener's returned closure")
	}
	if !foundFixedBinary {
		t.Error("expected the fixed \"xdg-open\" binary literal to still be constructed in place")
	}
}
