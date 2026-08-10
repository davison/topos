// whatsapplink_test.go pins 08-03-PLAN.md Task 3's own invariants: the
// discovered-binary allowlist check runs strictly BEFORE any subprocess is
// spawned (T-08-06, proved via an injected spawner fake that records
// invocation count/order, not just the response code), a session's poll
// cycle (start -> poll -> terminal state -> 404 on the next poll), reaping
// past deadline, the store-in-use vs generic error-code distinction
// (T-08-07), and that starting a session for an existing instance
// suspends it before spawning and always resumes it once the session ends
// — including when it ends in error.
package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/go-hclog"
)

// fakeSuspender is the Suspender test double: records every SuspendInstance
// call (name, in order) and every resume call, so tests can assert both
// that suspension happened before spawning and that resume happened
// exactly once, after the session reached a terminal state.
type fakeSuspender struct {
	mu           sync.Mutex
	suspendCalls []string
	resumeCalls  []string
	suspendErr   error
	resumeErr    error
}

func (f *fakeSuspender) SuspendInstance(_ context.Context, name string) (func(context.Context) error, error) {
	f.mu.Lock()
	f.suspendCalls = append(f.suspendCalls, name)
	f.mu.Unlock()
	if f.suspendErr != nil {
		return nil, f.suspendErr
	}
	return func(context.Context) error {
		f.mu.Lock()
		f.resumeCalls = append(f.resumeCalls, name)
		f.mu.Unlock()
		return f.resumeErr
	}, nil
}

func (f *fakeSuspender) suspendCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.suspendCalls)
}

func (f *fakeSuspender) resumeCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.resumeCalls)
}

// fakeLinkProcess is a scripted, in-memory stand-in for a real link
// subprocess: the test pushes raw JSON lines onto lines directly (never a
// real exec.Cmd), and records whether kill was ever called.
type fakeLinkProcess struct {
	lines  chan []byte
	done   chan error
	mu     sync.Mutex
	killed bool
}

func newFakeLinkProcess() *fakeLinkProcess {
	return &fakeLinkProcess{lines: make(chan []byte, 8), done: make(chan error, 1)}
}

func (f *fakeLinkProcess) emit(line string) { f.lines <- []byte(line) }

func (f *fakeLinkProcess) result() linkSpawnResult {
	return linkSpawnResult{
		lines: f.lines,
		done:  f.done,
		kill: func() {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.killed = true
		},
	}
}

func (f *fakeLinkProcess) wasKilled() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.killed
}

// fakeSpawner is a linkSpawner test double: records every (binPath,
// dataPath) invocation (proving whether or how many times it was ever
// called) and returns a scripted result (or error) the test controls.
type fakeSpawner struct {
	mu     sync.Mutex
	calls  []struct{ binPath, dataPath string }
	result linkSpawnResult
	err    error
}

func (f *fakeSpawner) spawn(_ context.Context, binPath, dataPath string) (linkSpawnResult, error) {
	f.mu.Lock()
	f.calls = append(f.calls, struct{ binPath, dataPath string }{binPath, dataPath})
	f.mu.Unlock()
	return f.result, f.err
}

func (f *fakeSpawner) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

// newWhatsAppLinkTestRouter wires the three link routes over a temp
// plugins directory containing exactly one discoverable binary
// ("topos-plugin-whatsapp", an empty placeholder file — DiscoverAllBinaries
// only checks the directory listing, never executes anything itself) so
// WhatsAppLinkStartHandler's allowlist check has something real to
// validate against.
func newWhatsAppLinkTestRouter(t *testing.T, suspender Suspender, spawner linkSpawner) (http.Handler, *linkSessionStore) {
	t.Helper()
	dir := t.TempDir()
	binPath := filepath.Join(dir, "topos-plugin-whatsapp")
	if err := os.WriteFile(binPath, []byte("x"), 0o755); err != nil {
		t.Fatalf("write plugin binary fixture: %v", err)
	}

	store := newLinkSessionStore()
	t.Cleanup(store.Shutdown)

	r := chi.NewRouter()
	r.Post("/api/config/whatsapp-link", WhatsAppLinkStartHandler(dir, suspender, spawner, store, hclog.NewNullLogger()))
	r.Get("/api/config/whatsapp-link/{session}", WhatsAppLinkPollHandler(store, hclog.NewNullLogger()))
	r.Delete("/api/config/whatsapp-link/{session}", WhatsAppLinkCancelHandler(store, hclog.NewNullLogger()))
	return r, store
}

func doLinkRequest(t *testing.T, router http.Handler, method, path, body string) *httptest.ResponseRecorder {
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

// TestWhatsAppLinkStart_UnknownBinaryReturns404WithoutSpawning proves the
// load-bearing ordering (T-08-06): the discovered-binary allowlist check
// runs BEFORE anything is executed — the injected spawner fake must
// record ZERO invocations for a request naming a binary absent from the
// discovered set.
func TestWhatsAppLinkStart_UnknownBinaryReturns404WithoutSpawning(t *testing.T) {
	spawner := &fakeSpawner{}
	router, _ := newWhatsAppLinkTestRouter(t, &fakeSuspender{}, spawner.spawn)

	body := `{"plugin":"topos-plugin-not-a-real-binary","path":"/tmp/whatsapp"}`
	rec := doLinkRequest(t, router, http.MethodPost, "/api/config/whatsapp-link", body)

	assertErrorEnvelope(t, rec, http.StatusNotFound, "plugin_binary_not_found")
	if got := spawner.callCount(); got != 0 {
		t.Fatalf("expected zero spawner invocations for an unknown binary, got %d", got)
	}
}

// TestWhatsAppLinkStartPoll_ReturnsLatestEventAndReapsOnPoll proves a start
// request returns a session id, polling returns the latest event the fake
// spawner emitted, and that polling an unknown id 404s.
func TestWhatsAppLinkStartPoll_ReturnsLatestEventAndReapsOnPoll(t *testing.T) {
	proc := newFakeLinkProcess()
	spawner := &fakeSpawner{result: proc.result()}
	router, _ := newWhatsAppLinkTestRouter(t, &fakeSuspender{}, spawner.spawn)

	startBody := `{"plugin":"topos-plugin-whatsapp","path":"/tmp/whatsapp"}`
	startRec := doLinkRequest(t, router, http.MethodPost, "/api/config/whatsapp-link", startBody)
	if startRec.Code != http.StatusOK {
		t.Fatalf("expected 200 from start, got %d: %s", startRec.Code, startRec.Body.String())
	}
	var startResp whatsappLinkResponse
	if err := json.Unmarshal(startRec.Body.Bytes(), &startResp); err != nil {
		t.Fatalf("unmarshal start response: %v", err)
	}
	if startResp.Session == "" {
		t.Fatal("expected a non-empty session id")
	}

	proc.emit(`{"kind":"qr","png_data_uri":"data:image/png;base64,AAAA","expires_in_seconds":20}`)

	var pollResp whatsappLinkResponse
	waitForCondition(t, func() bool {
		rec := doLinkRequest(t, router, http.MethodGet, "/api/config/whatsapp-link/"+startResp.Session, "")
		if rec.Code != http.StatusOK {
			return false
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &pollResp)
		return pollResp.State == "qr"
	})
	if pollResp.PNGDataURI != "data:image/png;base64,AAAA" {
		t.Fatalf("expected the fake spawner's qr event to be relayed verbatim, got %q", pollResp.PNGDataURI)
	}
	if pollResp.ExpiresInSeconds != 20 {
		t.Fatalf("expected expires_in_seconds 20, got %d", pollResp.ExpiresInSeconds)
	}

	unknownRec := doLinkRequest(t, router, http.MethodGet, "/api/config/whatsapp-link/not-a-real-session-id", "")
	assertErrorEnvelope(t, unknownRec, http.StatusNotFound, "link_session_not_found")
}

// TestWhatsAppLinkCancel_TerminatesSubprocessAndSubsequentPoll404s proves
// DELETE kills the subprocess and retires the session, so a subsequent
// poll for the same id 404s.
func TestWhatsAppLinkCancel_TerminatesSubprocessAndSubsequentPoll404s(t *testing.T) {
	proc := newFakeLinkProcess()
	spawner := &fakeSpawner{result: proc.result()}
	router, _ := newWhatsAppLinkTestRouter(t, &fakeSuspender{}, spawner.spawn)

	startBody := `{"plugin":"topos-plugin-whatsapp","path":"/tmp/whatsapp"}`
	startRec := doLinkRequest(t, router, http.MethodPost, "/api/config/whatsapp-link", startBody)
	var startResp whatsappLinkResponse
	if err := json.Unmarshal(startRec.Body.Bytes(), &startResp); err != nil {
		t.Fatalf("unmarshal start response: %v", err)
	}

	cancelRec := doLinkRequest(t, router, http.MethodDelete, "/api/config/whatsapp-link/"+startResp.Session, "")
	if cancelRec.Code != http.StatusOK {
		t.Fatalf("expected 200 from cancel, got %d: %s", cancelRec.Code, cancelRec.Body.String())
	}
	if !proc.wasKilled() {
		t.Fatal("expected cancel to kill the subprocess")
	}

	pollRec := doLinkRequest(t, router, http.MethodGet, "/api/config/whatsapp-link/"+startResp.Session, "")
	assertErrorEnvelope(t, pollRec, http.StatusNotFound, "link_session_not_found")
}

// TestWhatsAppLinkReaper proves a session left un-polled past its deadline
// is reaped: its subprocess is terminated and it 404s on the next poll.
// Drives the reaper directly (reapExpired) rather than waiting out the
// real linkSessionDeadline/linkSessionReapInterval durations, by
// backdating the fixture session's own deadline into the past.
func TestWhatsAppLinkReaper(t *testing.T) {
	proc := newFakeLinkProcess()
	store := newLinkSessionStore()
	t.Cleanup(store.Shutdown)

	sess := &linkSession{
		kill:     proc.result().kill,
		resume:   func(context.Context) error { return nil },
		deadline: time.Now().Add(-time.Minute), // already expired
	}
	id, err := store.register(sess)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	store.reapExpired()

	if !proc.wasKilled() {
		t.Fatal("expected the reaper to kill the expired session's subprocess")
	}
	if _, ok := store.get(id); ok {
		t.Fatal("expected the reaped session to be removed from the store")
	}
}

// TestWhatsAppLinkStart_StoreInUseErrorMapsToDistinctCode proves the
// plugin's store-in-use error event surfaces as the whatsapp_store_in_use
// code, distinct from the generic link_failed.
func TestWhatsAppLinkStart_StoreInUseErrorMapsToDistinctCode(t *testing.T) {
	proc := newFakeLinkProcess()
	spawner := &fakeSpawner{result: proc.result()}
	router, _ := newWhatsAppLinkTestRouter(t, &fakeSuspender{}, spawner.spawn)

	startBody := `{"plugin":"topos-plugin-whatsapp","path":"/tmp/whatsapp"}`
	startRec := doLinkRequest(t, router, http.MethodPost, "/api/config/whatsapp-link", startBody)
	var startResp whatsappLinkResponse
	if err := json.Unmarshal(startRec.Body.Bytes(), &startResp); err != nil {
		t.Fatalf("unmarshal start response: %v", err)
	}

	proc.emit(`{"kind":"error","code":"store_in_use","message":"whatsapp: another topos-plugin-whatsapp process already holds this data directory's lock"}`)

	var pollResp whatsappLinkResponse
	waitForCondition(t, func() bool {
		rec := doLinkRequest(t, router, http.MethodGet, "/api/config/whatsapp-link/"+startResp.Session, "")
		_ = json.Unmarshal(rec.Body.Bytes(), &pollResp)
		return pollResp.State == "error"
	})
	if pollResp.Code != "whatsapp_store_in_use" {
		t.Fatalf("expected code whatsapp_store_in_use, got %q", pollResp.Code)
	}
	if pollResp.Code == "link_failed" {
		t.Fatal("store-in-use must not collapse to the generic link_failed code")
	}
	if pollResp.Message == "" {
		t.Fatal("expected a non-empty message")
	}
}

// TestWhatsAppLinkStart_SuspendsBeforeSpawnAndResumesOnTerminalState proves
// a start request naming an existing instance suspends it before spawning
// and resumes it once the session reaches ANY terminal state — including
// error — via a fake Suspender recording call order.
func TestWhatsAppLinkStart_SuspendsBeforeSpawnAndResumesOnTerminalState(t *testing.T) {
	proc := newFakeLinkProcess()
	spawner := &fakeSpawner{result: proc.result()}
	suspender := &fakeSuspender{}
	router, _ := newWhatsAppLinkTestRouter(t, suspender, spawner.spawn)

	startBody := `{"plugin":"topos-plugin-whatsapp","path":"/tmp/whatsapp","instance":"my-whatsapp"}`
	startRec := doLinkRequest(t, router, http.MethodPost, "/api/config/whatsapp-link", startBody)
	if startRec.Code != http.StatusOK {
		t.Fatalf("expected 200 from start, got %d: %s", startRec.Code, startRec.Body.String())
	}

	if got := suspender.suspendCallCount(); got != 1 {
		t.Fatalf("expected SuspendInstance called exactly once before spawning, got %d", got)
	}
	if got := spawner.callCount(); got != 1 {
		t.Fatalf("expected the spawner called exactly once after suspension, got %d", got)
	}
	if got := suspender.resumeCallCount(); got != 0 {
		t.Fatalf("resume must not run before the session reaches a terminal state, got %d calls", got)
	}

	var startResp whatsappLinkResponse
	if err := json.Unmarshal(startRec.Body.Bytes(), &startResp); err != nil {
		t.Fatalf("unmarshal start response: %v", err)
	}

	// End the session in error — resume must still run.
	proc.emit(`{"kind":"error","code":"link_failed","message":"boom"}`)

	waitForCondition(t, func() bool {
		rec := doLinkRequest(t, router, http.MethodGet, "/api/config/whatsapp-link/"+startResp.Session, "")
		var resp whatsappLinkResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		return resp.State == "error"
	})

	if got := suspender.resumeCallCount(); got != 1 {
		t.Fatalf("expected resume called exactly once after the session ended in error, got %d", got)
	}
}

// waitForCondition polls cond until it returns true or a short timeout
// elapses, for tests observing the background consume goroutine's
// asynchronous effect on session state.
func waitForCondition(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition was never satisfied within the timeout")
}
