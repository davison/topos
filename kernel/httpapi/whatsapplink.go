package httpapi

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/pluginhost"
)

// Suspender is satisfied structurally by *supervisor.Supervisor (matching
// how Applier/Fetcher/HealthProber/Refresher are already declared in this
// package), the seam WhatsAppLinkStartHandler uses to stop a named running
// instance for a link session's duration — the second, independent layer
// behind the plugin's own store-lock (T-08-07): the pluginhost-launched
// instance and the link-mode subprocess this file spawns must never both
// hold the same whatsmeow session store open at once.
type Suspender interface {
	SuspendInstance(ctx context.Context, name string) (func(context.Context) error, error)
}

// --- link subprocess spawning (the seam tests drive without a real binary) ---

// linkSpawnResult is what a linkSpawner returns: a channel of raw
// newline-delimited JSON lines read from the subprocess's stdout (closed
// once stdout is exhausted), a channel receiving the subprocess's final
// exit error exactly once (nil on a clean exit), and a kill func that
// terminates the subprocess immediately.
type linkSpawnResult struct {
	lines <-chan []byte
	done  <-chan error
	kill  func()
}

// linkSpawner starts the link subprocess for the plugin binary at binPath
// against the data directory dataPath. Production wiring
// (execLinkSpawner) execs binPath -link-json -path dataPath; tests inject
// a fake that never touches the filesystem or a real process.
type linkSpawner func(ctx context.Context, binPath, dataPath string) (linkSpawnResult, error)

// stderrLineLoggerMaxBuffer bounds how much unterminated stderr text
// stderrLineLogger retains between newlines (T-08-17): a subprocess
// emitting one endless line without a newline flushes and resets at this
// cap instead of growing kernel memory without limit.
const stderrLineLoggerMaxBuffer = 64 * 1024

// stderrLineLogger is a line-oriented io.Writer assigned to a spawned
// link subprocess's cmd.Stderr: it buffers whatever is written, emits
// each complete line through logger as it arrives, and retains any
// trailing partial line for the next Write. flush (called once, after
// cmd.Wait() returns) emits any still-retained partial line so a
// diagnostic written without a trailing newline is never silently
// dropped.
//
// It is deliberately NOT an *os.File: assigning a non-*os.File Writer to
// cmd.Stderr makes os/exec create the pipe itself and run its own copy
// goroutine, and cmd.Wait() waits for that copy to finish before
// returning — the documented-correct shape. The alternative
// (cmd.StderrPipe() read concurrently with Wait) is the same
// incorrect-usage pattern already flagged on the stdout side; there is no
// reason to add a second instance of it here.
type stderrLineLogger struct {
	mu     sync.Mutex
	logger hclog.Logger
	buf    []byte
}

func newStderrLineLogger(logger hclog.Logger) *stderrLineLogger {
	return &stderrLineLogger{logger: logger.Named("link-subprocess")}
}

func (w *stderrLineLogger) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buf = append(w.buf, p...)
	for {
		idx := bytes.IndexByte(w.buf, '\n')
		if idx < 0 {
			break
		}
		w.emitLocked(w.buf[:idx])
		w.buf = w.buf[idx+1:]
	}
	if len(w.buf) > stderrLineLoggerMaxBuffer {
		w.emitLocked(w.buf)
		w.buf = nil
	}
	return len(p), nil
}

// flush emits any still-retained partial line. Call once, after cmd.Wait()
// has returned. On the healthy path that means os/exec's stderr copy
// goroutine has already finished, so nothing can Write concurrently; on the
// linkSubprocessWaitDelay path Wait returns having force-closed the pipe
// without waiting for that goroutine, so a final Write can still be in
// flight — which is why w.mu guards this rather than the call-ordering
// alone.
func (w *stderrLineLogger) flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.buf) > 0 {
		w.emitLocked(w.buf)
		w.buf = nil
	}
}

// emitLocked logs line through a single Info call carrying the line as a
// field, under the named "link-subprocess" sublogger so plugin
// diagnostics interleave legibly with kernel logs rather than arriving as
// a separate, untagged stream. Whitespace-only lines are skipped. Must be
// called with w.mu held.
func (w *stderrLineLogger) emitLocked(line []byte) {
	trimmed := bytes.TrimSpace(line)
	if len(trimmed) == 0 {
		return
	}
	w.logger.Info("whatsapp link subprocess diagnostic", "line", string(trimmed))
}

// linkSubprocessWaitDelay bounds how long cmd.Wait may stay blocked after
// the link subprocess itself is gone. os/exec's Wait waits for the stderr
// copy goroutine below to reach EOF, and that EOF requires EVERY holder of
// the stderr pipe's write end to close it — including any process the
// subprocess forked, which inherits that descriptor and is NOT killed by
// exec.CommandContext (it signals one pid, and SysProcAttr is deliberately
// nil here so link subprocesses share the kernel's process group; see
// knowledge-base.md KB-004). Without this bound, killing a link session
// whose subprocess had forked left Wait — and therefore linkSession.consume's
// <-done — parked for the orphan's entire lifetime.
//
// One second is ~1000x the copy goroutine's observed drain time on a clean
// exit, so it never elapses on the healthy path (where it would wrongly turn
// a nil exit error into ErrWaitDelay), while still returning promptly enough
// that a cancelled session's terminal event is not held up.
const linkSubprocessWaitDelay = time.Second

// newExecLinkSpawner returns the production linkSpawner: runs the
// discovered plugin binary in machine-readable link mode
// (plugins/whatsapp/link.go's runLinkJSON, Task 2 of this plan) as a raw
// subprocess, entirely outside the go-plugin gRPC handshake — this is
// deliberately exec.CommandContext against the resolved DISCOVERED path,
// never a path built from request-supplied input (T-08-06; the caller
// here always passes a path already validated by
// pluginhost.DiscoverAllBinaries). logger is the kernel's own hclog
// logger, already available at routes.go's call site — every line the
// subprocess writes to stderr is captured through it rather than
// discarded (08-UAT.md's G-08-1: previously cmd.Stderr was never
// assigned, so os/exec routed the subprocess's stderr to /dev/null and
// every plugin diagnostic and whatsmeow WARN/ERROR from a failing run was
// destroyed).
func newExecLinkSpawner(logger hclog.Logger) linkSpawner {
	return func(ctx context.Context, binPath, dataPath string) (linkSpawnResult, error) {
		spawnCtx, cancel := context.WithCancel(ctx)

		cmd := exec.CommandContext(spawnCtx, binPath, "-link-json", "-path", dataPath)
		cmd.WaitDelay = linkSubprocessWaitDelay
		// A nil Env means the child inherits this process's own
		// environment — os/exec's documented behavior, and the correct
		// one here, not an oversight (a prior version of this comment
		// stated the opposite; see 08-UAT.md's G-08-1, where that false
		// claim was found). The plugin expands a leading ~ in its -path
		// value through os/user, which reads HOME, and a host that
		// relocates its TLS trust store through the standard certificate
		// environment variables (e.g. SSL_CERT_FILE, SSL_CERT_DIR) would
		// otherwise break the linked-device websocket if this process's
		// environment were not inherited.
		cmd.Env = nil

		stderrLogger := newStderrLineLogger(logger)
		cmd.Stderr = stderrLogger

		// Deliberately NOT cmd.StdoutPipe(): os/exec owns the pipe it returns
		// and closes it inside cmd.Wait() as soon as the child exits,
		// which makes reading it from the goroutine below concurrently with
		// the Wait goroutine the exact usage os/exec documents as incorrect —
		// "it is thus incorrect to call Wait before all reads from the pipe
		// have completed". When Wait won that race the scanner's first Read
		// returned os.ErrClosed, Scan reported no lines at all, and a
		// fast-exiting subprocess's events were destroyed wholesale (48% of
		// iterations under a starved scheduler; the CI flake this comment
		// exists because of). Handing cmd.Stdout an *os.File we own instead
		// hands the child that descriptor directly: os/exec creates no pipe,
		// starts no copy goroutine, and never closes our read end, so reads
		// and Wait are simply independent (KB-004's "hand it an *os.File").
		stdoutRead, stdoutWrite, err := os.Pipe()
		if err != nil {
			cancel()
			return linkSpawnResult{}, fmt.Errorf("whatsapp link: stdout pipe: %w", err)
		}
		cmd.Stdout = stdoutWrite

		if err := cmd.Start(); err != nil {
			cancel()
			stdoutRead.Close()
			stdoutWrite.Close()
			return linkSpawnResult{}, fmt.Errorf("whatsapp link: start subprocess: %w", err)
		}
		// The child holds its own descriptor for the write end now. This
		// parent-side copy must be closed or the scanner below would never
		// observe EOF, even after the subprocess exits.
		stdoutWrite.Close()

		lines := make(chan []byte)
		done := make(chan error, 1)
		scanDone := make(chan struct{})

		// A subprocess's forked children inherit the stdout write end too, so
		// after a kill the scanner can sit in a read that will never reach EOF
		// while the orphan lives. Closing the read end on cancellation
		// releases it, which is what makes lines close promptly on kill.
		// Cancellation is the ONLY trigger: closing this on any success path
		// (e.g. a tidy-looking cancel() after Wait returns) would truncate
		// output still buffered in the pipe and reinstate the bug above.
		go func() {
			select {
			case <-spawnCtx.Done():
				stdoutRead.Close()
			case <-scanDone:
			}
		}()

		go func() {
			defer close(lines)
			defer close(scanDone)
			defer stdoutRead.Close()
			scanner := bufio.NewScanner(stdoutRead)
			// A rendered PNG data URI can comfortably exceed bufio.Scanner's
			// 64 KiB default token limit — raise it well past any QR image
			// this phase's fixed error-correction level and module count
			// could plausibly produce.
			scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
			for scanner.Scan() {
				line := append([]byte(nil), scanner.Bytes()...)
				select {
				case lines <- line:
				case <-spawnCtx.Done():
					return
				}
			}
		}()

		go func() {
			waitErr := cmd.Wait()
			// cmd.Wait() has returned, so the stderr copy goroutine os/exec
			// started for this non-*os.File cmd.Stderr has finished — except
			// on the linkSubprocessWaitDelay path, where Wait force-closes
			// the pipe and returns without awaiting it. flush holds the same
			// mutex Write does, so it is safe either way.
			stderrLogger.flush()
			done <- waitErr
			close(done)
		}()

		return linkSpawnResult{lines: lines, done: done, kill: cancel}, nil
	}
}

// --- session state ---

// maxConcurrentLinkSessions caps the number of link sessions this kernel
// process holds open at once (T-08-10): a stuck or abandoned browser tab
// can accumulate at most this many subprocesses before new start requests
// are refused.
const maxConcurrentLinkSessions = 4

// linkSessionDeadline bounds how long a session may go unpolled before the
// reaper terminates it (T-08-10) — generous enough to cover the full
// pairing flow (whatsmeow's own QR-code validity window rotates roughly
// every 20-60s, and postPairLoginTimeout in plugins/whatsapp/link.go
// allows up to 60s for the post-pair handshake) plus normal polling
// jitter, without leaving an abandoned session's subprocess (and,
// transitively, its suspended instance) alive indefinitely.
const linkSessionDeadline = 5 * time.Minute

// linkSessionReapInterval is how often the background reaper scans for
// sessions past their deadline.
const linkSessionReapInterval = 15 * time.Second

// linkWireEvent mirrors plugins/whatsapp/link.go's linkEvent JSON shape —
// the kernel's own decoding target for each newline-delimited event line a
// link subprocess emits. Kept as an unexported, package-local copy rather
// than a shared type: the wire contract is JSON tags, not a compiled Go
// dependency between the plugin and kernel modules.
type linkWireEvent struct {
	Kind             string `json:"kind"`
	PNGDataURI       string `json:"png_data_uri,omitempty"`
	ExpiresInSeconds int    `json:"expires_in_seconds,omitempty"`
	Code             string `json:"code,omitempty"`
	Message          string `json:"message,omitempty"`
}

// linkSession holds one in-flight link attempt's subprocess handle, the
// latest event a background reader goroutine observed, and the resume
// closure SuspendInstance returned (a no-op for the D-02 Add-Source case,
// where no instance was named).
type linkSession struct {
	kill     func()
	resume   func(context.Context) error
	deadline time.Time

	mu       sync.Mutex
	latest   linkWireEvent
	hasEvent bool
	terminal bool
}

// snapshot returns a copy of the session's latest observed event and
// whether it is a terminal (paired/error/timeout) state.
func (s *linkSession) snapshot() (ev linkWireEvent, hasEvent, terminal bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.latest, s.hasEvent, s.terminal
}

func (s *linkSession) setEvent(ev linkWireEvent, terminal bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latest = ev
	s.hasEvent = true
	if terminal {
		s.terminal = true
	}
}

// isTerminalKind reports whether kind is one of the three terminal
// linkEvent kinds plugins/whatsapp/link.go ever emits.
func isTerminalKind(kind string) bool {
	switch kind {
	case "paired", "error", "timeout":
		return true
	}
	return false
}

// consume runs as a background goroutine for the session's whole
// lifetime, decoding each newline-delimited JSON line the spawned
// subprocess emits into the session's latest observed event (T-08-11: a
// line that fails to decode is itself treated as an error event, never
// trusted as a partial/silent success). If the subprocess's stdout closes
// without ever emitting a terminal event (a crash, a kill from outside
// this goroutine, or any other unexpected exit), the subprocess's own
// Wait() error becomes a synthesized error event so a poll can never hang
// forever on a session whose process is already gone.
func (s *linkSession) consume(lines <-chan []byte, done <-chan error) {
	for line := range lines {
		var ev linkWireEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			s.setEvent(linkWireEvent{
				Kind:    "error",
				Code:    linkErrorCodeFailed,
				Message: "whatsapp: received a malformed link event from the plugin subprocess",
			}, true)
			continue
		}
		s.setEvent(ev, isTerminalKind(ev.Kind))
	}

	if _, _, terminal := s.snapshot(); terminal {
		return
	}

	msg := "whatsapp: link subprocess exited unexpectedly before reporting an outcome"
	if err := <-done; err != nil {
		msg = fmt.Sprintf("%s: %v", msg, err)
	}
	s.setEvent(linkWireEvent{Kind: "error", Code: linkErrorCodeFailed, Message: msg}, true)
}

// --- session store ---

// linkSessionStore is the in-memory registry of every live link session,
// keyed by an unguessable session id. One store is constructed per kernel
// process (cmd/topos/main.go) and shared across every WhatsAppLink*
// handler and the background reaper.
type linkSessionStore struct {
	mu       sync.Mutex
	sessions map[string]*linkSession
	// reserved counts slots claimed by reserve (WR-01, 08-REVIEW.md) that
	// have not yet been converted into a live sessions entry by register,
	// or given back by release. len(sessions)+reserved is the true
	// in-flight count reserve/register enforce maxConcurrentLinkSessions
	// against — reserved exists so the cap is checked BEFORE a subprocess
	// is spawned, not only after, since sess (and therefore a map key for
	// it) cannot exist until the spawn that produces its kill func has
	// already succeeded.
	reserved int
	stop     chan struct{}
	stopOnce sync.Once
}

// newLinkSessionStore constructs a store and starts its background reaper
// goroutine, which runs until Shutdown is called.
func newLinkSessionStore() *linkSessionStore {
	s := &linkSessionStore{
		sessions: make(map[string]*linkSession),
		stop:     make(chan struct{}),
	}
	go s.reapLoop()
	return s
}

func (s *linkSessionStore) reapLoop() {
	ticker := time.NewTicker(linkSessionReapInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.reapExpired()
		case <-s.stop:
			return
		}
	}
}

// reapExpired terminates and retires every session past its deadline
// (T-08-10) — a code is never served past its validity by an abandoned
// session outliving it.
func (s *linkSessionStore) reapExpired() {
	now := time.Now()

	s.mu.Lock()
	var expired []*linkSession
	for id, sess := range s.sessions {
		if now.After(sess.deadline) {
			expired = append(expired, sess)
			delete(s.sessions, id)
		}
	}
	s.mu.Unlock()

	for _, sess := range expired {
		sess.kill()
		if sess.resume != nil {
			_ = sess.resume(context.Background())
		}
	}
}

var errTooManyLinkSessions = fmt.Errorf("whatsapp link: too many concurrent link sessions (limit %d)", maxConcurrentLinkSessions)

// reserve claims one of maxConcurrentLinkSessions slots, refusing beyond
// the cap (WR-01, 08-REVIEW.md). Callers MUST call this — and get a nil
// error back — BEFORE spawning a link subprocess, not after: spawning
// execs a process, opens two sqlite files, and takes an exclusive
// storelock.go flock, all real costs that N simultaneous start requests
// would otherwise all pay before only the first maxConcurrentLinkSessions
// survived a capacity check made after the fact. Every path that claims a
// slot via reserve and does NOT go on to call register must call release
// to give it back (WhatsAppLinkStartHandler's own failure branches).
func (s *linkSessionStore) reserve() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.sessions)+s.reserved >= maxConcurrentLinkSessions {
		return errTooManyLinkSessions
	}
	s.reserved++
	return nil
}

// release gives back a slot claimed by reserve that was never converted
// into a live session via register — e.g. because SuspendInstance or the
// subprocess spawn itself failed after the slot was already claimed.
// Safe to call at most once per successful reserve() that was not
// followed by register(); a spurious extra call is a no-op rather than an
// underflow, since s.reserved never drops below zero.
func (s *linkSessionStore) release() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.reserved > 0 {
		s.reserved--
	}
}

// register assigns a fresh unguessable id to sess and stores it, converting
// a slot previously claimed by reserve into a live session entry.
// register itself no longer checks or claims capacity (WR-01,
// 08-REVIEW.md moved that to reserve, called BEFORE the subprocess this
// sess wraps was ever spawned) — every caller is required to have already
// called reserve() successfully first. A call with no outstanding
// reservation (s.reserved == 0) still succeeds and simply does not
// decrement below zero — register's own contract has always been "add
// sess to the store," never "enforce the cap," and existing tests that
// construct a *linkSession by hand and register it directly (bypassing
// WhatsAppLinkStartHandler's reserve call entirely) must keep working
// unchanged.
func (s *linkSessionStore) register(sess *linkSession) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, err := newLinkSessionID()
	if err != nil {
		return "", fmt.Errorf("whatsapp link: generate session id: %w", err)
	}
	s.sessions[id] = sess
	if s.reserved > 0 {
		s.reserved--
	}
	return id, nil
}

func (s *linkSessionStore) get(id string) (*linkSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	return sess, ok
}

// retire removes id from the store, returning the session it held (if
// any) so the caller can kill/resume it exactly once. A second concurrent
// caller racing to retire the same id sees ok=false.
func (s *linkSessionStore) retire(id string) (*linkSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if ok {
		delete(s.sessions, id)
	}
	return sess, ok
}

// Shutdown terminates every live session's subprocess and stops the
// reaper — called once, from kernel shutdown (cmd/topos/main.go), so a
// Ctrl-C never orphans a linking process holding a store lock.
func (s *linkSessionStore) Shutdown() {
	s.stopOnce.Do(func() { close(s.stop) })

	s.mu.Lock()
	sessions := make([]*linkSession, 0, len(s.sessions))
	for id, sess := range s.sessions {
		sessions = append(sessions, sess)
		delete(s.sessions, id)
	}
	s.mu.Unlock()

	for _, sess := range sessions {
		sess.kill()
		if sess.resume != nil {
			_ = sess.resume(context.Background())
		}
	}
}

// newLinkSessionID returns a random, unguessable 32-hex-character session
// id — crypto/rand, never math/rand: a session id is the sole bearer
// token guarding poll/cancel access to a live link subprocess (and,
// transitively, the suspended instance behind it) for this session's
// deadline window.
func newLinkSessionID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// --- error-code mapping ---

// linkErrorCodeStoreInUse and linkErrorCodeFailed mirror the two error
// event codes plugins/whatsapp/link.go's runLinkJSON ever emits
// (link.go's linkErrorCodeStoreInUse/linkErrorCodeFailed) — kept as a
// package-local copy for the identical reason linkWireEvent is: the wire
// contract is a JSON string value, not a compiled dependency.
const (
	linkErrorCodeStoreInUse = "store_in_use"
	linkErrorCodeFailed     = "link_failed"
)

// mapPluginErrorCode maps a plugin-emitted error event's Code to this
// route's own API error code vocabulary: the store-in-use failure (the
// plugin's store-lock refusing a concurrent instance — the independent
// second layer behind SuspendInstance, T-08-07) surfaces distinctly as
// whatsapp_store_in_use; every other plugin failure surfaces as the
// generic link_failed, carrying the plugin's own message verbatim.
func mapPluginErrorCode(pluginCode string) string {
	if pluginCode == linkErrorCodeStoreInUse {
		return "whatsapp_store_in_use"
	}
	return "link_failed"
}

// --- request/response shapes ---

// whatsappLinkStartRequest is POST /api/config/whatsapp-link's body.
// Instance is optional: present for the D-03 Re-link flow (an already-
// configured instance to suspend for the session's duration), absent for
// the D-02 Add-Source flow (nothing configured yet to suspend).
type whatsappLinkStartRequest struct {
	Plugin   string `json:"plugin"`
	Path     string `json:"path"`
	Instance string `json:"instance,omitempty"`
}

// whatsappLinkResponse is the shared response shape for all three routes:
// the session id, the latest known state ("pending" before the first
// event arrives, then "qr"/"paired"/"error"/"timeout", or "cancelled" for
// DELETE's own response), the QR payload when State is "qr", and the
// mapped error code/message when State is "error".
type whatsappLinkResponse struct {
	SchemaVersion    int    `json:"schema_version"`
	Session          string `json:"session,omitempty"`
	State            string `json:"state"`
	PNGDataURI       string `json:"png_data_uri,omitempty"`
	ExpiresInSeconds int    `json:"expires_in_seconds,omitempty"`
	Code             string `json:"code,omitempty"`
	Message          string `json:"message,omitempty"`
}

// --- handlers ---

// WhatsAppLinkStartHandler serves POST /api/config/whatsapp-link (D-01):
// spawns the named plugin binary in machine-readable link mode as a raw
// subprocess, outside the go-plugin gRPC handshake — this is not a
// SourcePlugin RPC, so the locked four-RPC contract (docs/plugin-
// contract.md) is untouched by this route's existence.
//
// Order of operations is load-bearing (T-08-06): decode the request,
// validate plugin against pluginhost.DiscoverAllBinaries and refuse an
// unknown name with 404 plugin_binary_not_found BEFORE anything is
// executed — directory listing, never a caller-supplied value, is the
// authority over what may be launched — THEN reject an empty path, THEN
// suspend instance (when non-empty), THEN spawn the subprocess via the
// resolved discovered path (never a path built from the request body).
func WhatsAppLinkStartHandler(pluginsDir string, suspender Suspender, spawner linkSpawner, store *linkSessionStore, logger hclog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req whatsappLinkStartRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_request", "request body is not valid JSON: "+err.Error())
			return
		}

		available, err := pluginhost.DiscoverAllBinaries(pluginsDir)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		known := false
		for _, name := range available {
			if name == req.Plugin {
				known = true
				break
			}
		}
		if !known {
			WriteError(w, http.StatusNotFound, "plugin_binary_not_found", "plugin \""+req.Plugin+"\" is not a discovered plugin binary")
			return
		}

		if req.Path == "" {
			WriteError(w, http.StatusBadRequest, "invalid_request", "path must not be empty")
			return
		}

		// WR-01 (08-REVIEW.md): claim a session slot BEFORE suspending an
		// instance or spawning the link subprocess, both of which have a
		// real cost — SuspendInstance reconciles the host down for the
		// named instance; spawning execs a process, opens two sqlite
		// files, and takes an exclusive flock. Enforcing the cap here,
		// rather than only after a successful spawn, means N simultaneous
		// start requests beyond the cap are refused before any of that
		// cost is paid, matching this handler's own existing ordering
		// discipline for the plugin-binary allowlist check above
		// ("directory listing … is the authority over what may be
		// launched … BEFORE anything is executed").
		if err := store.reserve(); err != nil {
			WriteError(w, http.StatusTooManyRequests, "link_failed", err.Error())
			return
		}

		resume := func(context.Context) error { return nil }
		if req.Instance != "" {
			// context.Background(), not r.Context(): SuspendInstance now
			// performs a real generation change (08-09-PLAN.md Task 1,
			// closing 08-UAT.md G-08-3) — a stop-scheduler, a subprocess
			// Reconcile, and a coordinator rebuild — that must outlive this
			// request. A browser disconnect mid-call must never abort that
			// work and leave the instance absent from both the host and the
			// coordinator until the next config save.
			r2, err := suspender.SuspendInstance(context.Background(), req.Instance)
			if err != nil {
				store.release()
				WriteError(w, http.StatusInternalServerError, "internal_error", "suspend instance \""+req.Instance+"\": "+err.Error())
				return
			}
			resume = r2
		}

		// binPath is the resolved DISCOVERED path (pluginsDir joined with
		// the already-validated req.Plugin) — never anything derived
		// directly from the request body beyond the name already checked
		// against DiscoverAllBinaries above (T-08-06).
		binPath := filepath.Join(pluginsDir, req.Plugin)

		// Every call in this file that outlives, or can mutate state that
		// outlives, this request runs detached on context.Background() —
		// the subprocess spawn below, every suspend, and every resume
		// (08-09-PLAN.md Task 1, closing 08-UAT.md G-08-3): suspend and
		// resume each perform a real subprocess reconcile AND a generation
		// rebuild, so a browser disconnect mid-call would abort the
		// relaunch and leave the instance absent from both the host and the
		// coordinator until the next config save.
		spawnResult, err := spawner(context.Background(), binPath, req.Path)
		if err != nil {
			store.release()
			if resumeErr := resume(context.Background()); resumeErr != nil {
				logger.Warn("whatsapp link: resume after failed spawn", "instance", req.Instance, "error", resumeErr.Error())
			}
			WriteError(w, http.StatusBadGateway, "link_failed", "failed to start link subprocess: "+err.Error())
			return
		}

		sess := &linkSession{
			kill:     spawnResult.kill,
			resume:   resume,
			deadline: time.Now().Add(linkSessionDeadline),
		}
		go sess.consume(spawnResult.lines, spawnResult.done)

		// register converts the slot reserve() already claimed above into
		// this live session entry — it can now fail only on session-id
		// generation (crypto/rand), never on capacity, since that was
		// already enforced before the subprocess was ever spawned.
		id, err := store.register(sess)
		if err != nil {
			store.release()
			sess.kill()
			if resumeErr := resume(context.Background()); resumeErr != nil {
				logger.Warn("whatsapp link: resume after registration failure", "instance", req.Instance, "error", resumeErr.Error())
			}
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		WriteJSON(w, http.StatusOK, whatsappLinkResponse{
			SchemaVersion: schemaVersion,
			Session:       id,
			State:         "pending",
		})
	}
}

// WhatsAppLinkPollHandler serves GET /api/config/whatsapp-link/{session}:
// returns the latest event's state, the png_data_uri/expires_in_seconds
// when it is a qr event, and the mapped error code/message when it is an
// error event. On any terminal state (paired, error, timeout) the handler
// retires the session and runs its resume closure AFTER building this
// response but BEFORE writing it, so the caller sees the terminal outcome
// exactly once (a second poll for the same id then 404s) and the
// suspended instance comes back up as soon as this response is produced.
func WhatsAppLinkPollHandler(store *linkSessionStore, logger hclog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "session")
		sess, ok := store.get(id)
		if !ok {
			WriteError(w, http.StatusNotFound, "link_session_not_found", "no active link session for id \""+id+"\"")
			return
		}

		ev, hasEvent, terminal := sess.snapshot()

		resp := whatsappLinkResponse{SchemaVersion: schemaVersion, Session: id, State: "pending"}
		if hasEvent {
			resp.State = ev.Kind
			resp.PNGDataURI = ev.PNGDataURI
			resp.ExpiresInSeconds = ev.ExpiresInSeconds
			if ev.Kind == "error" {
				resp.Code = mapPluginErrorCode(ev.Code)
				resp.Message = ev.Message
			}
		}

		if terminal {
			if retired, ok := store.retire(id); ok {
				retired.kill()
				if retired.resume != nil {
					// context.Background(), not r.Context(): resume performs
					// a real subprocess Reconcile and a generation rebuild
					// (08-09-PLAN.md Task 1, closing 08-UAT.md G-08-3) —
					// this poll's own client disconnecting must never abort
					// relaunching the suspended instance.
					if err := retired.resume(context.Background()); err != nil {
						logger.Warn("whatsapp link: resume after terminal poll", "session", id, "error", err.Error())
					}
				}
			}
		}

		WriteJSON(w, http.StatusOK, resp)
	}
}

// WhatsAppLinkCancelHandler serves DELETE /api/config/whatsapp-link/{session}:
// kills the subprocess, runs the resume closure, and retires the session.
func WhatsAppLinkCancelHandler(store *linkSessionStore, logger hclog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "session")
		sess, ok := store.retire(id)
		if !ok {
			WriteError(w, http.StatusNotFound, "link_session_not_found", "no active link session for id \""+id+"\"")
			return
		}

		sess.kill()
		if sess.resume != nil {
			// context.Background(), not r.Context() — see the poll handler's
			// identical comment above.
			if err := sess.resume(context.Background()); err != nil {
				logger.Warn("whatsapp link: resume after cancel", "session", id, "error", err.Error())
			}
		}

		WriteJSON(w, http.StatusOK, whatsappLinkResponse{
			SchemaVersion: schemaVersion,
			Session:       id,
			State:         "cancelled",
		})
	}
}
