// whatsapplink_exec_test.go drives newExecLinkSpawner (08-06-PLAN.md
// Task 2) against a real, executable POSIX-shell fixture written into
// t.TempDir() — no plugin binary, no network, no WhatsApp. This is the
// first automated coverage of the production spawner: every other test in
// this package (whatsapplink_test.go) injects a fakeSpawner that never
// execs a real process, which structurally cannot exercise argv
// construction, stderr capture, environment inheritance, or process kill.
//
// Skipped on Windows: the fixtures below are /bin/sh scripts, and this
// project targets a Linux desktop (topos/CLAUDE.md).
package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
)

// skipOnWindows skips the calling test on Windows, where no /bin/sh
// exists for the shell-script fixtures below.
func skipOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("requires /bin/sh — this project targets a Linux desktop")
	}
}

// writeShellFixture writes an executable /bin/sh script into a fresh
// t.TempDir() and returns its path.
func writeShellFixture(t *testing.T, script string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fixture.sh")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fixture script: %v", err)
	}
	return path
}

// syncBuffer is a mutex-guarded bytes.Buffer for use as an hclog sink —
// guarded because a caller could in principle read the buffer while the
// spawned subprocess's stderr-copy goroutine is still writing to it,
// even though every test below only reads after result.done has fired
// (which happens-after cmd.Wait(), which happens-after that copy
// goroutine finishes).
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// newTestHCLogSink returns an hclog.Logger writing to a syncBuffer this
// test can assert against, and the buffer itself.
func newTestHCLogSink() (hclog.Logger, *syncBuffer) {
	buf := &syncBuffer{}
	logger := hclog.New(&hclog.LoggerOptions{Output: buf, Level: hclog.Trace})
	return logger, buf
}

// drainLines consumes result.lines to completion (the channel closes once
// the subprocess's stdout is exhausted), returning every line observed in
// order.
func drainLines(lines <-chan []byte) [][]byte {
	var got [][]byte
	for line := range lines {
		got = append(got, line)
	}
	return got
}

// drainLinesWithin is drainLines with a deadline, so a spawner that never
// closes lines fails the calling test with a clear message instead of
// hanging until the whole package times out.
func drainLinesWithin(t *testing.T, lines <-chan []byte, d time.Duration) [][]byte {
	t.Helper()
	drained := make(chan [][]byte, 1)
	go func() { drained <- drainLines(lines) }()
	select {
	case got := <-drained:
		return got
	case <-time.After(d):
		t.Fatalf("timed out after %v waiting for the lines channel to close", d)
		return nil
	}
}

// asStrings renders collected lines for comparison and failure messages.
func asStrings(lines [][]byte) []string {
	var out []string
	for _, l := range lines {
		out = append(out, string(l))
	}
	return out
}

// TestExecLinkSpawner_StreamsLinesInOrderAndExitsClean proves a fixture
// emitting three newline-delimited JSON lines then exiting 0 yields those
// three lines, in order, on result.lines, the channel then closes, and
// done yields a nil error.
func TestExecLinkSpawner_StreamsLinesInOrderAndExitsClean(t *testing.T) {
	skipOnWindows(t)

	script := "#!/bin/sh\n" +
		"echo '{\"kind\":\"qr\"}'\n" +
		"echo '{\"kind\":\"pairing_accepted\"}'\n" +
		"echo '{\"kind\":\"paired\"}'\n" +
		"exit 0\n"
	binPath := writeShellFixture(t, script)

	logger, _ := newTestHCLogSink()
	spawner := newExecLinkSpawner(logger)
	result, err := spawner(context.Background(), binPath, t.TempDir())
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}

	gotLines := drainLines(result.lines)
	want := []string{`{"kind":"qr"}`, `{"kind":"pairing_accepted"}`, `{"kind":"paired"}`}
	var got []string
	for _, l := range gotLines {
		got = append(got, string(l))
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got lines %v, want %v", got, want)
	}

	select {
	case doneErr := <-result.done:
		if doneErr != nil {
			t.Fatalf("expected nil done error, got %v", doneErr)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for done")
	}
}

// TestExecLinkSpawner_PassesLinkModeFlagAndDataPath proves the same
// fixture, made to record its own arguments to a file, sees exactly the
// link-mode flag and the path flag carrying the dataPath the spawner was
// called with.
func TestExecLinkSpawner_PassesLinkModeFlagAndDataPath(t *testing.T) {
	skipOnWindows(t)

	dir := t.TempDir()
	argFile := filepath.Join(dir, "args.txt")
	script := fmt.Sprintf("#!/bin/sh\necho \"$@\" > %q\nexit 0\n", argFile)
	binPath := writeShellFixture(t, script)

	logger, _ := newTestHCLogSink()
	spawner := newExecLinkSpawner(logger)
	dataPath := "/fake/whatsapp/data/path"
	result, err := spawner(context.Background(), binPath, dataPath)
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	drainLines(result.lines)
	<-result.done

	gotBytes, err := os.ReadFile(argFile)
	if err != nil {
		t.Fatalf("read recorded args file: %v", err)
	}
	got := strings.TrimSpace(string(gotBytes))
	want := fmt.Sprintf("-link-json -path %s", dataPath)
	if got != want {
		t.Fatalf("expected recorded args %q, got %q", want, got)
	}
}

// TestExecLinkSpawner_CapturesStderrIntoLogger proves a diagnostic the
// fixture writes to stderr appears in the hclog sink the spawner was
// constructed with.
func TestExecLinkSpawner_CapturesStderrIntoLogger(t *testing.T) {
	skipOnWindows(t)

	const diagnostic = "distinctive-stderr-diagnostic-from-fixture"
	script := "#!/bin/sh\n" +
		"echo '" + diagnostic + "' >&2\n" +
		"exit 0\n"
	binPath := writeShellFixture(t, script)

	logger, sink := newTestHCLogSink()
	spawner := newExecLinkSpawner(logger)
	result, err := spawner(context.Background(), binPath, t.TempDir())
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	drainLines(result.lines)
	<-result.done

	if !strings.Contains(sink.String(), diagnostic) {
		t.Fatalf("expected stderr diagnostic %q in logger sink, got %q", diagnostic, sink.String())
	}
}

// TestExecLinkSpawner_FlushesTrailingPartialStderrLine proves a fixture
// whose final stderr write has no trailing newline still has that text in
// the sink once done has yielded.
func TestExecLinkSpawner_FlushesTrailingPartialStderrLine(t *testing.T) {
	skipOnWindows(t)

	const partial = "no-trailing-newline-diagnostic"
	script := "#!/bin/sh\n" +
		"printf '" + partial + "' >&2\n" +
		"exit 0\n"
	binPath := writeShellFixture(t, script)

	logger, sink := newTestHCLogSink()
	spawner := newExecLinkSpawner(logger)
	result, err := spawner(context.Background(), binPath, t.TempDir())
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	drainLines(result.lines)
	<-result.done

	if !strings.Contains(sink.String(), partial) {
		t.Fatalf("expected trailing partial stderr line %q flushed into logger sink, got %q", partial, sink.String())
	}
}

// TestExecLinkSpawner_InheritsEnvironment proves a variable set with
// t.Setenv before spawning is visible to the child — the fixture echoes
// it back inside an event line, which is the assertion that makes the
// corrected environment comment non-vacuous.
func TestExecLinkSpawner_InheritsEnvironment(t *testing.T) {
	skipOnWindows(t)

	t.Setenv("TOPOS_EXEC_LINK_SPAWNER_TEST_VAR", "the-inherited-value")

	script := "#!/bin/sh\n" +
		"echo \"{\\\"kind\\\":\\\"error\\\",\\\"message\\\":\\\"$TOPOS_EXEC_LINK_SPAWNER_TEST_VAR\\\"}\"\n" +
		"exit 0\n"
	binPath := writeShellFixture(t, script)

	logger, _ := newTestHCLogSink()
	spawner := newExecLinkSpawner(logger)
	result, err := spawner(context.Background(), binPath, t.TempDir())
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}

	lines := drainLines(result.lines)
	<-result.done
	if len(lines) != 1 {
		t.Fatalf("expected exactly one event line, got %d: %v", len(lines), lines)
	}

	var ev struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(lines[0], &ev); err != nil {
		t.Fatalf("unmarshal event line: %v", err)
	}
	if ev.Message != "the-inherited-value" {
		t.Fatalf("expected inherited env var value echoed back, got %q", ev.Message)
	}
}

// TestExecLinkSpawner_KillTerminatesLongRunningSubprocess proves a
// fixture that sleeps far longer than the test is terminated by the
// returned kill func — done yields a non-nil error and lines closes, both
// within a couple of seconds.
func TestExecLinkSpawner_KillTerminatesLongRunningSubprocess(t *testing.T) {
	skipOnWindows(t)

	script := "#!/bin/sh\nsleep 30\nexit 0\n"
	binPath := writeShellFixture(t, script)

	logger, _ := newTestHCLogSink()
	spawner := newExecLinkSpawner(logger)
	result, err := spawner(context.Background(), binPath, t.TempDir())
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}

	result.kill()

	select {
	case doneErr := <-result.done:
		if doneErr == nil {
			t.Fatal("expected a non-nil done error after kill")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("expected done to yield within a few seconds of kill")
	}

	linesClosed := make(chan struct{})
	go func() {
		drainLines(result.lines) // returns once the channel closes
		close(linesClosed)
	}()
	select {
	case <-linesClosed:
	case <-time.After(3 * time.Second):
		t.Fatal("expected lines channel to close after kill")
	}
}

// TestExecLinkSpawner_DeliversBufferedOutputWhenDrainedAfterSubprocessExit
// is the deterministic regression guard for the CI flake in
// .planning/debug/resolved/ci-flaky-test-portable.md (root cause 1): the
// spawner used to read cmd.StdoutPipe() concurrently with cmd.Wait(), and
// Wait closes that pipe the moment the child exits, so whenever Wait won
// the race the scanner's first read failed with os.ErrClosed and every
// event line was destroyed. In production that silently replaced a
// fast-failing plugin's actionable error event with the generic "exited
// unexpectedly before reporting an outcome".
//
// The sibling tests above can only catch that by losing a scheduler race
// (~48% per iteration under a starved scheduler, near-never on an idle
// developer machine — hence months of intermittent CI failures). This test
// removes the race from the oracle instead: it waits for done, which proves
// the subprocess has exited AND Wait has fully returned, and only then
// reads a single line. Output buffered in a pipe stays readable after the
// writer exits, so a spawner that owns its read end still yields all three
// lines here; one that lets Wait close that pipe can physically yield no
// more than the single line an unbuffered channel send holds in flight.
func TestExecLinkSpawner_DeliversBufferedOutputWhenDrainedAfterSubprocessExit(t *testing.T) {
	skipOnWindows(t)

	script := "#!/bin/sh\n" +
		"echo '{\"kind\":\"qr\"}'\n" +
		"echo '{\"kind\":\"pairing_accepted\"}'\n" +
		"echo '{\"kind\":\"paired\"}'\n" +
		"exit 0\n"
	binPath := writeShellFixture(t, script)

	logger, _ := newTestHCLogSink()
	spawner := newExecLinkSpawner(logger)
	result, err := spawner(context.Background(), binPath, t.TempDir())
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}

	// Deliberately BEFORE reading any line: this is the whole point of the
	// test. Once done yields, cmd.Wait has returned, so any pipe Wait owns
	// is already closed.
	select {
	case doneErr := <-result.done:
		if doneErr != nil {
			t.Fatalf("expected nil done error, got %v", doneErr)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for done")
	}

	got := asStrings(drainLinesWithin(t, result.lines, 10*time.Second))
	want := []string{`{"kind":"qr"}`, `{"kind":"pairing_accepted"}`, `{"kind":"paired"}`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("subprocess output was lost after it exited: got lines %v, want %v", got, want)
	}
}

// TestExecLinkSpawner_KillYieldsDoneWhenSubprocessLeftAForkedChild is the
// deterministic regression guard for root cause 2 of the same session.
// exec.CommandContext's cancel signals exactly one pid, and SysProcAttr is
// deliberately nil here (KB-004: link subprocesses share the kernel's
// process group), so anything the subprocess forked survives the kill —
// still holding the stderr pipe's write end it inherited. Because
// cmd.Stderr is a non-*os.File, cmd.Wait waits for os/exec's stderr copy
// goroutine to reach EOF, and that EOF cannot arrive while the orphan
// lives: Wait, done, and therefore linkSession.consume's terminal error
// event were all parked for the orphan's full lifetime.
//
// TestExecLinkSpawner_KillTerminatesLongRunningSubprocess above hits this
// only when the shell wins the race to fork before the kill lands, which is
// exactly why it failed on loaded CI runners and passed locally. This test
// makes the orphan a precondition rather than an accident: the fixture
// backgrounds a child and only then emits its readiness line, so by the
// time kill is called the orphan provably exists.
func TestExecLinkSpawner_KillYieldsDoneWhenSubprocessLeftAForkedChild(t *testing.T) {
	skipOnWindows(t)

	dir := t.TempDir()
	pidFile := filepath.Join(dir, "forked-child.pid")
	// The backgrounded sleep outlives its parent by far more than this
	// test's bounds, so a spawner that waits for it fails outright rather
	// than passing slowly. The shell then parks in `wait`, a builtin — it
	// forks no second process, so this fixture leaves exactly one reapable
	// orphan (the pid recorded above) rather than an untracked one too.
	script := fmt.Sprintf("#!/bin/sh\n"+
		"sleep 30 &\n"+
		"echo $! > %q\n"+
		"echo '{\"kind\":\"qr\"}'\n"+
		"wait\n", pidFile)
	binPath := writeShellFixture(t, script)

	// The forked child outlives the subprocess by design, so reap it here
	// rather than leaving a stray process behind on every run.
	t.Cleanup(func() {
		raw, err := os.ReadFile(pidFile)
		if err != nil {
			return
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
		if err != nil {
			return
		}
		if proc, err := os.FindProcess(pid); err == nil {
			_ = proc.Kill()
		}
	})

	logger, _ := newTestHCLogSink()
	spawner := newExecLinkSpawner(logger)
	result, err := spawner(context.Background(), binPath, t.TempDir())
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}

	// Readiness: this line is emitted only after the fork, so receiving it
	// proves the orphan exists before the kill below. Without it this test
	// would race the shell exactly as the older kill test does.
	select {
	case line, ok := <-result.lines:
		if !ok {
			t.Fatal("lines closed before the readiness event")
		}
		if string(line) != `{"kind":"qr"}` {
			t.Fatalf("expected the readiness event, got %q", string(line))
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for the readiness event that proves the subprocess forked a child")
	}

	result.kill()

	select {
	case doneErr := <-result.done:
		if doneErr == nil {
			t.Fatal("expected a non-nil done error after kill")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("done did not yield within 3s of kill: cmd.Wait is blocked on a pipe the subprocess's forked child still holds open")
	}

	if lines := drainLinesWithin(t, result.lines, 3*time.Second); len(lines) != 0 {
		t.Fatalf("expected no further lines after kill, got %v", asStrings(lines))
	}
}
