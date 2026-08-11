package config

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDefaultConfig_RoundTripsThroughLoadRaw proves DefaultConfig() survives
// the kernel's own real load-and-validate path (LoadRaw, which includes
// Validate) rather than a shape that only happens to marshal — the
// assertion that matters most for 09.1-BOOTSTRAP, since a default that
// fails validation would strand every fresh install at boot.
func TestDefaultConfig_RoundTripsThroughLoadRaw(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	if err := WriteCanonical(path, DefaultConfig()); err != nil {
		t.Fatalf("WriteCanonical(DefaultConfig()): %v", err)
	}

	expanded, _, _, _, err := LoadRaw(path)
	if err != nil {
		t.Fatalf("LoadRaw of the written default: %v", err)
	}
	if len(expanded.Webspaces) != 0 {
		t.Errorf("expected zero webspaces, got %d", len(expanded.Webspaces))
	}
	if len(expanded.Sources) != 0 {
		t.Errorf("expected zero sources, got %d", len(expanded.Sources))
	}
}

// TestDefaultConfig_WrittenBytesStartWithCanonicalHeader guards against a
// future refactor that bypasses WriteCanonical for the bootstrap write path.
func TestDefaultConfig_WrittenBytesStartWithCanonicalHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	if err := WriteCanonical(path, DefaultConfig()); err != nil {
		t.Fatalf("WriteCanonical(DefaultConfig()): %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read written config: %v", err)
	}
	if !strings.HasPrefix(string(raw), CanonicalHeader) {
		t.Errorf("written config does not start with CanonicalHeader, got:\n%s", raw)
	}
}

// TestDefaultConfig_WritesExplicitEmptyTables proves R1's marshal-shape
// guarantee: non-nil empty Sources/Webspaces maps produce explicit table
// headers in the written TOML, not omitted keys — the written file
// documents its own shape to a user who opens it.
func TestDefaultConfig_WritesExplicitEmptyTables(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	if err := WriteCanonical(path, DefaultConfig()); err != nil {
		t.Fatalf("WriteCanonical(DefaultConfig()): %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read written config: %v", err)
	}
	body := string(raw)
	if !strings.Contains(body, "[sources]") {
		t.Errorf("written config missing explicit [sources] table header, got:\n%s", body)
	}
	if !strings.Contains(body, "[webspaces]") {
		t.Errorf("written config missing explicit [webspaces] table header, got:\n%s", body)
	}
}

// TestDefaultConfig_ListenIsLoopback is threat T-09.1-B4: the first-run
// default must never ship a machine listening beyond loopback. Asserts the
// SAME predicate main.isLoopback applies (host-only, via net.SplitHostPort)
// rather than duplicating main's own rule as a separate literal string
// comparison.
func TestDefaultConfig_ListenIsLoopback(t *testing.T) {
	listen := DefaultConfig().Server.Listen
	host, _, err := net.SplitHostPort(listen)
	if err != nil {
		t.Fatalf("split host:port from %q: %v", listen, err)
	}
	if host != "127.0.0.1" {
		t.Errorf("expected DefaultConfig().Server.Listen host to be 127.0.0.1, got %q (full: %q)", host, listen)
	}
}

// TestDefaultConfig_WrittenFileModeIs0600 is threat T-09.1-B2: the written
// config file must never be readable beyond the owning user, since it can
// later hold ${VAR} references and per-source connection details.
func TestDefaultConfig_WrittenFileModeIs0600(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	if err := WriteCanonical(path, DefaultConfig()); err != nil {
		t.Fatalf("WriteCanonical(DefaultConfig()): %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat written config: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("expected written config file mode 0600, got %#o", perm)
	}
}
