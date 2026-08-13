package pluginhost

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/config"
)

// writeFakeFatalPlugin writes a small shell-script "plugin binary" into
// t.TempDir(), under a name matching DiscoverBinaries' own
// topos-plugin-<name> convention, that writes message to stderr (nothing
// to stdout) and exits non-zero — exactly the shape of a real plugin's
// fatal helper (see plugins/signal/main.go's fatal: one line to stderr,
// then os.Exit(1)) firing BEFORE goplugin.Serve ever starts the
// handshake.
//
// A script rather than a compiled Go binary is deliberate: launch() only
// os.Stats and execs the path at binPath — it has no idea, and no way to
// tell, whether what it just ran was Go, a script, or anything else. The
// behaviour under test is entirely "writes to stderr, exits before
// writing a handshake line to stdout", which a shell script makes
// unmistakable and instant to build, with no go build step and no
// compiled-fixture cache to manage.
func writeFakeFatalPlugin(t *testing.T, message string) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "topos-plugin-fake-fatal")
	script := fmt.Sprintf("#!/bin/sh\necho %q 1>&2\nexit 1\n", message)
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake plugin script: %v", err)
	}
	return dir
}

// writeFakeSilentPlugin is writeFakeFatalPlugin's silent sibling: exits
// non-zero writing nothing to stderr or stdout at all — the "genuinely
// silent failure" edge KERN-08's empty-input truth names.
func writeFakeSilentPlugin(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "topos-plugin-fake-silent")
	script := "#!/bin/sh\nexit 1\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake silent plugin script: %v", err)
	}
	return dir
}

// TestDescribePluginType_PreHandshakeFatalSurfacesPluginStderr is RED
// against the unmodified host.go (before Stderr is wired into
// goplugin.ClientConfig): the returned error carries only go-plugin's own
// generic handshake-diagnostic text, and the fake plugin's distinctive
// message appears nowhere in it — the exact shape 07-UAT.md G-07-5
// reported ("four stated causes, all false"). GREEN once the capture is
// wired: the same distinctive message appears in the returned error,
// alongside the existing "connect to plugin subprocess:" wrap prefix.
func TestDescribePluginType_PreHandshakeFatalSurfacesPluginStderr(t *testing.T) {
	const message = "topos-plugin-fake-fatal: config.Path is required, got empty string"
	dir := writeFakeFatalPlugin(t, message)

	_, err := DescribePluginType(context.Background(), Dirs{Trusted: dir}, config.Source{Plugin: "topos-plugin-fake-fatal"}, hclog.NewNullLogger())
	if err == nil {
		t.Fatal("expected an error for a plugin that fatals before the handshake")
	}

	if !strings.HasPrefix(err.Error(), "pluginhost: trial-launch for describe:") {
		t.Errorf("expected the existing trial-launch wrap prefix to be preserved, got: %v", err)
	}
	if !strings.Contains(err.Error(), "connect to plugin subprocess:") {
		t.Errorf("expected the existing connect-failure wrap to be preserved, got: %v", err)
	}
	if !strings.Contains(err.Error(), message) {
		t.Errorf("expected the plugin's own stderr line to appear in the error, got: %v", err)
	}
}

// TestDescribePluginType_SilentPreHandshakeExitLeavesErrorUnchanged proves
// KERN-08's empty edge: a plugin that exits pre-handshake writing NOTHING
// to stderr must not have its error text made noisier or more confusing —
// no empty parenthetical, no dangling separator.
func TestDescribePluginType_SilentPreHandshakeExitLeavesErrorUnchanged(t *testing.T) {
	dir := writeFakeSilentPlugin(t)

	_, err := DescribePluginType(context.Background(), Dirs{Trusted: dir}, config.Source{Plugin: "topos-plugin-fake-silent"}, hclog.NewNullLogger())
	if err == nil {
		t.Fatal("expected an error for a plugin that exits non-zero before the handshake")
	}

	if strings.Contains(err.Error(), "(plugin stderr:") {
		t.Errorf("expected no plugin-stderr parenthetical when nothing was captured, got: %v", err)
	}
	if strings.HasSuffix(strings.TrimSpace(err.Error()), "(") || strings.Contains(err.Error(), ": )") {
		t.Errorf("expected no dangling separator when nothing was captured, got: %v", err)
	}
}

// TestStderrTail_BoundedRetainsMostRecentOutput writes well past the cap
// and asserts the retained length never exceeds it, and that what
// survives is the most recently written bytes (front-discard), not the
// earliest.
func TestStderrTail_BoundedRetainsMostRecentOutput(t *testing.T) {
	tail := &stderrTail{}

	// Write far more than stderrTailCap, one recognisable line at a time,
	// so the last line written is unambiguous.
	var lastWritten string
	total := 0
	for i := 0; total < stderrTailCap*3; i++ {
		line := fmt.Sprintf("line-%06d filler-filler-filler-filler-filler\n", i)
		n, err := tail.Write([]byte(line))
		if err != nil {
			t.Fatalf("Write: %v", err)
		}
		if n != len(line) {
			t.Fatalf("expected Write to report the full input length %d, got %d", len(line), n)
		}
		total += n
		lastWritten = strings.TrimSpace(line)
	}

	tail.mu.Lock()
	retained := len(tail.buf)
	tail.mu.Unlock()
	if retained > stderrTailCap {
		t.Errorf("expected retained length to never exceed stderrTailCap (%d), got %d", stderrTailCap, retained)
	}

	if got := tail.lastLine(); got != lastWritten {
		t.Errorf("expected lastLine() to return the most recently written line %q, got %q", lastWritten, got)
	}
}

// TestStderrTail_LastLineTrimsTrailingWhitespaceAndNewlines proves the
// last-line rule directly: trailing newlines/whitespace are stripped, and
// a buffer whose final bytes are bare newlines still yields the last line
// that actually has content.
func TestStderrTail_LastLineTrimsTrailingWhitespaceAndNewlines(t *testing.T) {
	tail := &stderrTail{}
	if _, err := tail.Write([]byte("first line\nsecond line  \n\n\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got, want := tail.lastLine(), "second line"; got != want {
		t.Errorf("expected lastLine() %q, got %q", want, got)
	}
}

// TestStderrTail_EmptyBufferYieldsEmptyString proves the empty-input edge:
// a tail nothing was ever written to returns "", never a nil-dereference
// or an error.
func TestStderrTail_EmptyBufferYieldsEmptyString(t *testing.T) {
	tail := &stderrTail{}
	if got := tail.lastLine(); got != "" {
		t.Errorf("expected an empty string from a never-written-to tail, got %q", got)
	}
}

// TestStderrTail_ConcurrentWritersThenRead is the concurrency proof
// KERN-08's concurrency edge and T-07-45 require: multiple goroutines
// writing concurrently, synchronized to have all completed before the
// single read — mirroring the real shape (go-plugin's logStderr goroutine
// writes; launch reads only after client.Kill() has returned, which
// waits on that goroutine). Clean only under `go test -race`.
func TestStderrTail_ConcurrentWritersThenRead(t *testing.T) {
	tail := &stderrTail{}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_, _ = tail.Write([]byte(fmt.Sprintf("writer-%02d-%03d\n", i, j)))
			}
		}(i)
	}
	wg.Wait()

	// The read happens only after every writer has returned — matching
	// launch's own "read only after client.Kill() returns" discipline.
	// This is not itself a race-detector coverage guarantee on its own;
	// the race detector proves the concurrent Write calls above and this
	// read are all synchronized correctly by the mutex, not by this
	// wg.Wait() alone.
	_ = tail.lastLine()
}
