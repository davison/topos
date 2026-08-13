// pin_test.go pins launch's pre-exec pin-verification gate (T-11-07,
// D-01/D-02/D-03/D-04): a matching pin launches, a mismatched or missing
// pin refuses BEFORE any subprocess is created, and the trusted tier never
// even looks at [plugins.pins].
package pluginhost

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/config"
)

// copyMockBinaryToFreshDir copies buildMockPluginDir's cached mock plugin
// binary (reconcile_test.go's sync.Once-built fixture, shared read-only by
// every other test in this package) into a brand-new, private temp
// directory — so a test that needs to MUTATE the bytes (the byte-rewrite
// mismatch case below) never corrupts the shared fixture every other test
// in this binary run also depends on.
func copyMockBinaryToFreshDir(t *testing.T) string {
	t.Helper()
	srcDir := buildMockPluginDir(t)
	src, err := os.Open(filepath.Join(srcDir, "topos-plugin-mock"))
	if err != nil {
		t.Fatalf("open shared mock plugin fixture: %v", err)
	}
	defer src.Close()

	dstDir := t.TempDir()
	dst, err := os.OpenFile(filepath.Join(dstDir, "topos-plugin-mock"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		t.Fatalf("create private mock plugin copy: %v", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		t.Fatalf("copy mock plugin fixture: %v", err)
	}
	return dstDir
}

// TestLaunch_Pin_MatchingHashLaunches proves the base case: an
// external-tier binary whose pin matches its on-disk SHA-256 launches
// exactly like an unpinned trusted-tier binary would.
func TestLaunch_Pin_MatchingHashLaunches(t *testing.T) {
	externalDir := buildMockPluginDir(t)
	binPath := filepath.Join(externalDir, "topos-plugin-mock")

	hash, err := HashBinary(binPath)
	if err != nil {
		t.Fatalf("HashBinary: %v", err)
	}

	dirs := Dirs{External: externalDir}
	raw := &config.Config{Plugins: config.PluginsConfig{Pins: map[string]string{"topos-plugin-mock": hash}}}
	src := config.Source{Plugin: "topos-plugin-mock"}

	p, err := launch(context.Background(), dirs, "demo", src, raw, hclog.NewNullLogger(), false)
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	defer p.Kill()

	if p.Tier() != TierExternal {
		t.Errorf("expected tier %q, got %q", TierExternal, p.Tier())
	}
}

// TestLaunch_Pin_AlteredBytesReturnsErrPinMismatch proves T-11-07's whole
// point: rewriting one byte of the pinned binary makes the NEXT launch fail
// with an error satisfying errors.Is(err, ErrPinMismatch), naming the
// instance and the binary, and — since exec.Command is never reached — the
// call returns near-instantly rather than waiting anywhere near go-plugin's
// own one-minute handshake StartTimeout, proving no child process was ever
// started.
func TestLaunch_Pin_AlteredBytesReturnsErrPinMismatch(t *testing.T) {
	externalDir := copyMockBinaryToFreshDir(t)
	binPath := filepath.Join(externalDir, "topos-plugin-mock")

	originalHash, err := HashBinary(binPath)
	if err != nil {
		t.Fatalf("HashBinary (pre-mutation): %v", err)
	}

	// Flip one byte at the very end of the file — anywhere works, since
	// SHA-256 avalanches on any single-bit change.
	f, err := os.OpenFile(binPath, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("open for mutation: %v", err)
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		t.Fatalf("stat: %v", err)
	}
	buf := make([]byte, 1)
	if _, err := f.ReadAt(buf, info.Size()-1); err != nil {
		f.Close()
		t.Fatalf("read last byte: %v", err)
	}
	buf[0] ^= 0xFF
	if _, err := f.WriteAt(buf, info.Size()-1); err != nil {
		f.Close()
		t.Fatalf("write mutated byte: %v", err)
	}
	f.Close()

	dirs := Dirs{External: externalDir}
	raw := &config.Config{Plugins: config.PluginsConfig{Pins: map[string]string{"topos-plugin-mock": originalHash}}}
	src := config.Source{Plugin: "topos-plugin-mock"}

	start := time.Now()
	p, err := launch(context.Background(), dirs, "demo", src, raw, hclog.NewNullLogger(), false)
	elapsed := time.Since(start)

	if err == nil {
		if p != nil {
			p.Kill()
		}
		t.Fatal("expected an error for a mutated binary, got nil")
	}
	if !errors.Is(err, ErrPinMismatch) {
		t.Fatalf("expected errors.Is(err, ErrPinMismatch), got: %v", err)
	}
	if p != nil {
		t.Fatalf("expected a nil *Plugin on pin mismatch (no subprocess should ever be created), got %+v", p)
	}
	if !strings.Contains(err.Error(), "demo") {
		t.Errorf("expected the error to name the instance %q, got: %v", "demo", err)
	}
	if !strings.Contains(err.Error(), "topos-plugin-mock") {
		t.Errorf("expected the error to name the binary %q, got: %v", "topos-plugin-mock", err)
	}
	// go-plugin's own client StartTimeout defaults to a full minute — a
	// pin-mismatch refusal that actually happens BEFORE exec.Command is
	// constructed returns in well under a second. This is a generous
	// margin (not a knife edge) that would fail if the refusal instead
	// waited on a real subprocess handshake.
	if elapsed >= 5*time.Second {
		t.Fatalf("expected the pin-mismatch refusal to return near-instantly (no subprocess ever started), took %v", elapsed)
	}
}

// TestLaunch_Pin_NoPinRecordedBehavesAsMismatch proves the "empty pin
// value" edge (must_haves: "An external-tier binary with NO pin recorded
// is treated as a pin mismatch against an empty pinned value"): an
// external-tier binary with no [plugins.pins] entry at all — not even an
// empty one — refuses exactly like a genuine mismatch, and the error's
// pinned-value text reads "not pinned" rather than a raw empty string.
func TestLaunch_Pin_NoPinRecordedBehavesAsMismatch(t *testing.T) {
	externalDir := buildMockPluginDir(t)

	dirs := Dirs{External: externalDir}
	raw := &config.Config{} // no [plugins.pins] table at all
	src := config.Source{Plugin: "topos-plugin-mock"}

	p, err := launch(context.Background(), dirs, "demo", src, raw, hclog.NewNullLogger(), false)
	if err == nil {
		if p != nil {
			p.Kill()
		}
		t.Fatal("expected an error for an unpinned external binary, got nil")
	}
	if !errors.Is(err, ErrPinMismatch) {
		t.Fatalf("expected errors.Is(err, ErrPinMismatch), got: %v", err)
	}
	if !strings.Contains(err.Error(), "not pinned") {
		t.Errorf("expected the error to read \"not pinned\" for an absent pin entry, got: %v", err)
	}
}

// TestLaunch_Pin_NilRawBehavesAsUnpinned proves a nil raw config (the
// describeOnly=false, no-raw-document edge — should not occur for a real
// boot but must degrade gracefully rather than panic) is handled exactly
// like an empty pins map: every external-tier launch is treated as
// unpinned.
func TestLaunch_Pin_NilRawBehavesAsUnpinned(t *testing.T) {
	externalDir := buildMockPluginDir(t)

	dirs := Dirs{External: externalDir}
	src := config.Source{Plugin: "topos-plugin-mock"}

	p, err := launch(context.Background(), dirs, "demo", src, nil, hclog.NewNullLogger(), false)
	if err == nil {
		if p != nil {
			p.Kill()
		}
		t.Fatal("expected an error for a nil raw config against an external-tier binary, got nil")
	}
	if !errors.Is(err, ErrPinMismatch) {
		t.Fatalf("expected errors.Is(err, ErrPinMismatch), got: %v", err)
	}
}

// TestLaunch_Pin_TrustedTierIgnoresPins proves D-04: a trusted-tier binary
// launches successfully even with a DELIBERATELY WRONG pin entry present —
// pins apply to the external tier only, and a trusted-dir binary (rebuilt
// constantly by `make build`/`make dev`) must never be pin-checked at all.
func TestLaunch_Pin_TrustedTierIgnoresPins(t *testing.T) {
	trustedDir := buildMockPluginDir(t)

	dirs := Dirs{Trusted: trustedDir}
	raw := &config.Config{Plugins: config.PluginsConfig{Pins: map[string]string{
		"topos-plugin-mock": strings.Repeat("0", 64),
	}}}
	src := config.Source{Plugin: "topos-plugin-mock"}

	p, err := launch(context.Background(), dirs, "demo", src, raw, hclog.NewNullLogger(), false)
	if err != nil {
		t.Fatalf("expected a trusted-tier launch to ignore a wrong pin entirely, got error: %v", err)
	}
	defer p.Kill()

	if p.Tier() != TierTrusted {
		t.Errorf("expected tier %q, got %q", TierTrusted, p.Tier())
	}
}
