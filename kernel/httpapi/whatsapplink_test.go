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
// call (name, in order), the context each call and its resume closure were
// handed, and every resume call, so tests can assert both that suspension
// happened before spawning and that resume happened exactly once, after
// the session reached a terminal state — and (Task 3's Guard 2,
// 08-09-PLAN.md) that neither call was ever handed the request's own
// context.
type fakeSuspender struct {
	mu           sync.Mutex
	suspendCalls []string
	resumeCalls  []string
	suspendCtxs  []context.Context
	resumeCtxs   []context.Context
	suspendErr   error
	resumeErr    error
}

func (f *fakeSuspender) SuspendInstance(ctx context.Context, name string) (func(context.Context) error, error) {
	f.mu.Lock()
	f.suspendCalls = append(f.suspendCalls, name)
	f.suspendCtxs = append(f.suspendCtxs, ctx)
	f.mu.Unlock()
	if f.suspendErr != nil {
		return nil, f.suspendErr
	}
	return func(resumeCtx context.Context) error {
		f.mu.Lock()
		f.resumeCalls = append(f.resumeCalls, name)
		f.resumeCtxs = append(f.resumeCtxs, resumeCtx)
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

// lastSuspendCtx returns the context handed to the most recent
// SuspendInstance call, or nil if it was never called.
func (f *fakeSuspender) lastSuspendCtx() context.Context {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.suspendCtxs) == 0 {
		return nil
	}
	return f.suspendCtxs[len(f.suspendCtxs)-1]
}

// lastResumeCtx returns the context handed to the most recent resume
// closure invocation, or nil if resume was never called.
func (f *fakeSuspender) lastResumeCtx() context.Context {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.resumeCtxs) == 0 {
		return nil
	}
	return f.resumeCtxs[len(f.resumeCtxs)-1]
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

// TestWhatsAppLinkStart_CapEnforcedBeforeSpawn is WR-01's (08-REVIEW.md)
// regression test: with the session cap already reached, a further start
// request must be refused 429 WITHOUT ever invoking the spawner — before
// this fix, WhatsAppLinkStartHandler called the spawner first and only
// rejected afterward via linkSessionStore.register's own (now-removed)
// after-the-fact capacity check, so a request beyond the cap still paid
// the full cost of a process spawn (exec, two sqlite opens, an exclusive
// flock acquire/release) before being refused.
func TestWhatsAppLinkStart_CapEnforcedBeforeSpawn(t *testing.T) {
	spawner := &fakeSpawner{result: newFakeLinkProcess().result()}
	router, store := newWhatsAppLinkTestRouter(t, &fakeSuspender{}, spawner.spawn)

	// Fill every session slot directly via register — never through the
	// HTTP route, so the spawner fake's call count below reflects ONLY
	// the one request this test actually issues.
	for i := 0; i < maxConcurrentLinkSessions; i++ {
		sess := &linkSession{
			kill:     func() {},
			resume:   func(context.Context) error { return nil },
			deadline: time.Now().Add(time.Hour),
		}
		if _, err := store.register(sess); err != nil {
			t.Fatalf("register slot %d: %v", i, err)
		}
	}

	body := `{"plugin":"topos-plugin-whatsapp","path":"/tmp/whatsapp"}`
	rec := doLinkRequest(t, router, http.MethodPost, "/api/config/whatsapp-link", body)

	assertErrorEnvelope(t, rec, http.StatusTooManyRequests, "link_failed")

	if got := spawner.callCount(); got != 0 {
		t.Fatalf("expected zero spawner invocations once the session cap is already reached, got %d — the cap must be enforced BEFORE spawning, not after", got)
	}
}

// TestWhatsAppLink_ProgressStatesAreNonTerminal is the kernel-side half
// of a three-way agreement with plugins/whatsapp/link.go (Task 1, which
// emits the pairing_accepted/already_linked wire kinds this test drives
// as raw JSON strings) and docs/api.md (which now documents both as
// non-terminal). Table-driven over both progress kinds: observing either
// leaves the session live and pollable (200, not 404), leaves the
// suspended instance suspended (resume not yet called), and carries no
// diagnostic fields in the poll response body — proving "subprocess
// diagnostics never reach a response body" holds for these two kinds too.
// The session is then driven to the existing `paired` terminal state to
// prove the exactly-once terminal contract (T-08-10/T-08-07) is still
// intact behind the two new non-terminal states.
func TestWhatsAppLink_ProgressStatesAreNonTerminal(t *testing.T) {
	for _, kind := range []string{"pairing_accepted", "already_linked"} {
		t.Run(kind, func(t *testing.T) {
			proc := newFakeLinkProcess()
			spawner := &fakeSpawner{result: proc.result()}
			suspender := &fakeSuspender{}
			router, _ := newWhatsAppLinkTestRouter(t, suspender, spawner.spawn)

			startBody := `{"plugin":"topos-plugin-whatsapp","path":"/tmp/whatsapp","instance":"my-whatsapp"}`
			startRec := doLinkRequest(t, router, http.MethodPost, "/api/config/whatsapp-link", startBody)
			var startResp whatsappLinkResponse
			if err := json.Unmarshal(startRec.Body.Bytes(), &startResp); err != nil {
				t.Fatalf("unmarshal start response: %v", err)
			}

			proc.emit(`{"kind":"` + kind + `"}`)

			var rawBody []byte
			waitForCondition(t, func() bool {
				rec := doLinkRequest(t, router, http.MethodGet, "/api/config/whatsapp-link/"+startResp.Session, "")
				if rec.Code != http.StatusOK {
					return false
				}
				var resp whatsappLinkResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					return false
				}
				if resp.State != kind {
					return false
				}
				rawBody = rec.Body.Bytes()
				return true
			})

			// Key-set assertion: exactly schema_version, session, state —
			// no code, no message, no png_data_uri. Decoding into a map
			// (not the typed whatsappLinkResponse) so a stray field the
			// typed struct doesn't declare can't hide from this guard.
			var decoded map[string]any
			if err := json.Unmarshal(rawBody, &decoded); err != nil {
				t.Fatalf("unmarshal raw poll response: %v", err)
			}
			wantKeys := map[string]bool{"schema_version": true, "session": true, "state": true}
			for key := range decoded {
				if !wantKeys[key] {
					t.Fatalf("poll response for a %q progress state carried an unexpected field %q: %v", kind, key, decoded)
				}
			}

			// Observing a progress state must not retire the session: a
			// second poll for the same id still returns 200, not 404.
			secondRec := doLinkRequest(t, router, http.MethodGet, "/api/config/whatsapp-link/"+startResp.Session, "")
			if secondRec.Code != http.StatusOK {
				t.Fatalf("expected a second poll after a %q progress state to still return 200 (session not retired), got %d: %s", kind, secondRec.Code, secondRec.Body.String())
			}

			// The suspended instance must not have been resumed while the
			// link session is still live and holding the store — a
			// resumed instance racing the still-live link subprocess is
			// exactly the two-processes-contending-for-one-store
			// condition SuspendInstance and the plugin's own flock exist
			// to prevent (T-08-07).
			if got := suspender.resumeCallCount(); got != 0 {
				t.Fatalf("expected resume NOT called while a %q progress state is the latest observed event (would mean two processes contending for the same whatsmeow store, T-08-07), got %d calls", kind, got)
			}

			// Now drive the session to the paired terminal state and
			// prove the pre-existing exactly-once terminal contract still
			// holds behind the new non-terminal states.
			proc.emit(`{"kind":"paired"}`)

			waitForCondition(t, func() bool {
				rec := doLinkRequest(t, router, http.MethodGet, "/api/config/whatsapp-link/"+startResp.Session, "")
				var resp whatsappLinkResponse
				_ = json.Unmarshal(rec.Body.Bytes(), &resp)
				return resp.State == "paired"
			})

			if got := suspender.resumeCallCount(); got != 1 {
				t.Fatalf("expected resume called exactly once after the session ended in paired, got %d", got)
			}

			notFoundRec := doLinkRequest(t, router, http.MethodGet, "/api/config/whatsapp-link/"+startResp.Session, "")
			assertErrorEnvelope(t, notFoundRec, http.StatusNotFound, "link_session_not_found")
		})
	}
}

// TestIsTerminalKind_ProgressKindsAreNonTerminal is a guard against a
// future edit that "completes" linkEventKind's coverage by adding the two
// progress kinds to isTerminalKind's terminal set — doing so would
// reproduce this plan's own gap (G-08-1) in a new shape: a progress state
// retiring the session and resuming a suspended instance while the link
// subprocess is still live.
func TestIsTerminalKind_ProgressKindsAreNonTerminal(t *testing.T) {
	nonTerminal := []string{"pairing_accepted", "already_linked"}
	for _, kind := range nonTerminal {
		if isTerminalKind(kind) {
			t.Fatalf("expected isTerminalKind(%q) == false, got true — a progress state must never be treated as terminal", kind)
		}
	}

	terminal := []string{"paired", "error", "timeout"}
	for _, kind := range terminal {
		if !isTerminalKind(kind) {
			t.Fatalf("expected isTerminalKind(%q) == true, got false", kind)
		}
	}
}

// doLinkRequestWithCancelledContext is doLinkRequest, but the request is
// served with a context derived from its own and cancelled BEFORE
// ServeHTTP runs — simulating a browser that has already disconnected by
// the time this request reaches the handler.
func doLinkRequestWithCancelledContext(t *testing.T, router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	ctx, cancel := context.WithCancel(req.Context())
	cancel()
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// TestWhatsAppLink_SuspendAndResumeRunOnDetachedContexts is Task 3's Guard
// 2 (08-09-PLAN.md): the start, poll, and cancel handlers must hand the
// supervisor a context this request cannot cancel — context.Background(),
// never r.Context() — for every suspend/resume call, because each
// performs a real subprocess Reconcile and a generation rebuild
// (08-09-PLAN.md Task 1) that must outlive a browser that disappears
// mid-request. This is deliberately behavioural rather than a source
// scan: each sub-test serves a request whose OWN context is already
// cancelled before ServeHTTP runs, and asserts the context the fake
// suspender/resume closure actually received carries no error — a
// handler that forwarded the request's own context would hand over an
// already-cancelled one and the assertion would fail.
func TestWhatsAppLink_SuspendAndResumeRunOnDetachedContexts(t *testing.T) {
	t.Run("start hands the suspender a detached context", func(t *testing.T) {
		suspender := &fakeSuspender{}
		spawner := &fakeSpawner{result: newFakeLinkProcess().result()}
		router, _ := newWhatsAppLinkTestRouter(t, suspender, spawner.spawn)

		startBody := `{"plugin":"topos-plugin-whatsapp","path":"/tmp/whatsapp","instance":"my-whatsapp"}`
		rec := doLinkRequestWithCancelledContext(t, router, http.MethodPost, "/api/config/whatsapp-link", startBody)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 from start, got %d: %s", rec.Code, rec.Body.String())
		}

		ctx := suspender.lastSuspendCtx()
		if ctx == nil {
			t.Fatal("expected SuspendInstance to have been called")
		}
		if err := ctx.Err(); err != nil {
			t.Fatalf("expected the context handed to SuspendInstance to be detached from the request (no error), got: %v — a handler forwarding r.Context() would hand over an already-cancelled context here", err)
		}
	})

	t.Run("poll reaching a terminal state hands resume a detached context", func(t *testing.T) {
		proc := newFakeLinkProcess()
		spawner := &fakeSpawner{result: proc.result()}
		suspender := &fakeSuspender{}
		router, _ := newWhatsAppLinkTestRouter(t, suspender, spawner.spawn)

		startBody := `{"plugin":"topos-plugin-whatsapp","path":"/tmp/whatsapp","instance":"my-whatsapp"}`
		startRec := doLinkRequest(t, router, http.MethodPost, "/api/config/whatsapp-link", startBody)
		var startResp whatsappLinkResponse
		if err := json.Unmarshal(startRec.Body.Bytes(), &startResp); err != nil {
			t.Fatalf("unmarshal start response: %v", err)
		}

		proc.emit(`{"kind":"paired"}`)

		waitForCondition(t, func() bool {
			rec := doLinkRequestWithCancelledContext(t, router, http.MethodGet, "/api/config/whatsapp-link/"+startResp.Session, "")
			if rec.Code != http.StatusOK {
				return false
			}
			var resp whatsappLinkResponse
			_ = json.Unmarshal(rec.Body.Bytes(), &resp)
			return resp.State == "paired"
		})

		ctx := suspender.lastResumeCtx()
		if ctx == nil {
			t.Fatal("expected resume to have been called after the session reached the paired terminal state")
		}
		if err := ctx.Err(); err != nil {
			t.Fatalf("expected the context handed to resume from the poll handler to be detached from the request (no error), got: %v — a handler forwarding r.Context() would hand over an already-cancelled context here", err)
		}
	})

	t.Run("cancel hands resume a detached context", func(t *testing.T) {
		proc := newFakeLinkProcess()
		spawner := &fakeSpawner{result: proc.result()}
		suspender := &fakeSuspender{}
		router, _ := newWhatsAppLinkTestRouter(t, suspender, spawner.spawn)

		startBody := `{"plugin":"topos-plugin-whatsapp","path":"/tmp/whatsapp","instance":"my-whatsapp"}`
		startRec := doLinkRequest(t, router, http.MethodPost, "/api/config/whatsapp-link", startBody)
		var startResp whatsappLinkResponse
		if err := json.Unmarshal(startRec.Body.Bytes(), &startResp); err != nil {
			t.Fatalf("unmarshal start response: %v", err)
		}

		cancelRec := doLinkRequestWithCancelledContext(t, router, http.MethodDelete, "/api/config/whatsapp-link/"+startResp.Session, "")
		if cancelRec.Code != http.StatusOK {
			t.Fatalf("expected 200 from cancel, got %d: %s", cancelRec.Code, cancelRec.Body.String())
		}

		ctx := suspender.lastResumeCtx()
		if ctx == nil {
			t.Fatal("expected resume to have been called after cancel")
		}
		if err := ctx.Err(); err != nil {
			t.Fatalf("expected the context handed to resume from the cancel handler to be detached from the request (no error), got: %v — a handler forwarding r.Context() would hand over an already-cancelled context here", err)
		}
	})
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
