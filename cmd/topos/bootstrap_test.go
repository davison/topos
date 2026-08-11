// bootstrap_test.go covers 09.1-BOOTSTRAP's bootstrapConfig gate: a
// genuinely missing config.toml is bootstrapped to a default; anything
// else (malformed file, permission denied, an already-broken existing
// file) is left completely untouched and reported as-is. Uses t.TempDir()
// throughout — never configPath()'s real resolution or the user's home.
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/config"
)

// TestBootstrapConfig_WritesOnTrulyMissingFileIntoNestedDir points path at
// a nested directory that does not exist yet — the nested segment is what
// proves the MkdirAll step, since a bare t.TempDir() path would pass even
// without it. loadErr comes from actually calling config.NewStore on the
// missing path (not a hand-made sentinel), matching what setup() itself
// would see.
func TestBootstrapConfig_WritesOnTrulyMissingFileIntoNestedDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "topos", "config.toml")

	_, loadErr := config.NewStore(path)
	if loadErr == nil {
		t.Fatalf("expected config.NewStore(%q) to fail on a missing file", path)
	}

	wrote, err := bootstrapConfig(path, loadErr, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("bootstrapConfig: %v", err)
	}
	if !wrote {
		t.Fatalf("expected bootstrapConfig to report wrote == true for a missing file")
	}

	if _, err := config.NewStore(path); err != nil {
		t.Fatalf("config.NewStore after bootstrap: %v", err)
	}
}

// TestBootstrapConfig_DirAndFileModes asserts the created parent
// directory's permission bits are 0700 (T-09.1-B2) and the written file's
// are 0600 (WriteCanonical's own os.CreateTemp contract).
func TestBootstrapConfig_DirAndFileModes(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "topos")
	path := filepath.Join(dir, "config.toml")

	_, loadErr := config.NewStore(path)

	wrote, err := bootstrapConfig(path, loadErr, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("bootstrapConfig: %v", err)
	}
	if !wrote {
		t.Fatalf("expected wrote == true")
	}

	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat config dir: %v", err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0o700 {
		t.Errorf("expected config dir mode 0700, got %#o", perm)
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat config file: %v", err)
	}
	if perm := fileInfo.Mode().Perm(); perm != 0o600 {
		t.Errorf("expected config file mode 0600, got %#o", perm)
	}
}

// TestBootstrapConfig_RefusesNonENOENTError is threat T-09.1-B1's core
// case: a loadErr that is not os.ErrNotExist (here, a permission error
// wrapped with %w to mirror LoadRaw's own wrapping shape) must never be
// treated as "must be missing" — bootstrapConfig must decline and create
// nothing.
func TestBootstrapConfig_RefusesNonENOENTError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	loadErr := fmt.Errorf("config: read %s: %w", path, fs.ErrPermission)

	wrote, err := bootstrapConfig(path, loadErr, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("bootstrapConfig: %v", err)
	}
	if wrote {
		t.Fatalf("expected wrote == false for a non-ENOENT loadErr")
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("expected no file to be created at %s, stat returned: %v", path, statErr)
	}
}

// TestBootstrapConfig_NeverOverwritesExistingFile is the test RESEARCH
// Pitfall 3 explicitly warns a positive-case-only suite misses: a
// deliberately malformed (present-but-broken) config file must be left
// byte-identical on disk, not replaced by a fresh default, when
// bootstrapConfig is called with the real error config.NewStore produces
// for it (a parse failure, not ENOENT).
func TestBootstrapConfig_NeverOverwritesExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	malformed := []byte("this is not valid toml [[[\n")
	if err := os.WriteFile(path, malformed, 0o600); err != nil {
		t.Fatalf("write malformed config: %v", err)
	}

	_, loadErr := config.NewStore(path)
	if loadErr == nil {
		t.Fatalf("expected config.NewStore to fail on malformed TOML")
	}
	if errors.Is(loadErr, os.ErrNotExist) {
		t.Fatalf("test setup bug: loadErr must not be os.ErrNotExist, got %v", loadErr)
	}

	wrote, err := bootstrapConfig(path, loadErr, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("bootstrapConfig: %v", err)
	}
	if wrote {
		t.Fatalf("expected wrote == false for an existing-but-broken file")
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config after bootstrapConfig: %v", err)
	}
	if string(after) != string(malformed) {
		t.Errorf("existing broken config was modified: before=%q after=%q", malformed, after)
	}
}

// TestBootstrapConfig_HermeticEndToEndViaConfigPath closes the loop between
// "the branch is correct in isolation" (the cases above, all hand-passed
// paths) and "a fresh machine actually boots": it exercises configPath()
// itself — the real XDG_CONFIG_HOME-resolution seam — rather than a
// hand-passed path, and runs the exact sequence setup() runs:
// config.NewStore, bootstrapConfig, config.NewStore again.
//
// Deliberately does NOT call setup() itself: setup also calls index.Open
// on the expanded index path, which would create a database outside this
// test's temp root. A later reader must not "simplify" this test into
// calling setup() directly — doing so would start writing to the
// developer's real data directory the moment DefaultConfig()'s
// Index.Path ("~/.local/share/topos/index.db") got expanded and opened.
func TestBootstrapConfig_HermeticEndToEndViaConfigPath(t *testing.T) {
	xdgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgHome)

	path := configPath()
	wantPrefix := filepath.Join(xdgHome, "topos")
	if filepath.Dir(path) != wantPrefix {
		t.Fatalf("configPath() = %q, expected it to resolve under %q", path, wantPrefix)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no file at %q before bootstrap, stat returned: %v", path, err)
	}

	_, loadErr := config.NewStore(path)
	if loadErr == nil {
		t.Fatalf("expected config.NewStore(%q) to fail before bootstrap", path)
	}

	wrote, err := bootstrapConfig(path, loadErr, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("bootstrapConfig: %v", err)
	}
	if !wrote {
		t.Fatalf("expected bootstrapConfig to report wrote == true")
	}

	cfgStore, err := config.NewStore(path)
	if err != nil {
		t.Fatalf("config.NewStore after bootstrap: %v", err)
	}
	expanded := cfgStore.Expanded()
	if len(expanded.Webspaces) != 0 {
		t.Errorf("expected zero webspaces on the freshly bootstrapped config, got %d", len(expanded.Webspaces))
	}
}
