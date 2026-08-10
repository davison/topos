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
