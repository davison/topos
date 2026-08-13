// externaldir_test.go pins Phase 11's per-OS external plugin directory
// resolution (D-09, PLUG-06): defaultExternalPluginsDir's Linux
// XDG_DATA_HOME-set and -unset branches, and externalPluginsDir's
// config-override precedence over the computed default.
package main

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/davison/topos/kernel/config"
)

// TestDefaultExternalPluginsDir_LinuxXDGDataHomeSet proves the primary
// Linux branch: XDG_DATA_HOME set resolves to
// "$XDG_DATA_HOME/topos/plugins-external".
func TestDefaultExternalPluginsDir_LinuxXDGDataHomeSet(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-specific XDG_DATA_HOME branch")
	}
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg-data-home-test")

	got, err := defaultExternalPluginsDir()
	if err != nil {
		t.Fatalf("defaultExternalPluginsDir: %v", err)
	}
	want := filepath.Join("/tmp/xdg-data-home-test", "topos", "plugins-external")
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
	if !hasSuffixPath(got, filepath.Join("topos", "plugins-external")) {
		t.Errorf("expected the path to end in topos/plugins-external, got %q", got)
	}
}

// TestDefaultExternalPluginsDir_LinuxXDGDataHomeUnset proves the Linux
// fallback branch: XDG_DATA_HOME unset (or empty) resolves to
// "~/.local/share/topos/plugins-external".
func TestDefaultExternalPluginsDir_LinuxXDGDataHomeUnset(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-specific XDG_DATA_HOME fallback branch")
	}
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "/home/xdg-fallback-test-user")

	got, err := defaultExternalPluginsDir()
	if err != nil {
		t.Fatalf("defaultExternalPluginsDir: %v", err)
	}
	want := filepath.Join("/home/xdg-fallback-test-user", ".local", "share", "topos", "plugins-external")
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
	if !hasSuffixPath(got, filepath.Join("topos", "plugins-external")) {
		t.Errorf("expected the path to end in topos/plugins-external, got %q", got)
	}
}

func hasSuffixPath(path, suffix string) bool {
	return len(path) >= len(suffix) && path[len(path)-len(suffix):] == suffix
}

// TestExternalPluginsDir_EmptyConfigFallsBackToDefault proves an omitted
// [plugins] external_dir resolves via defaultExternalPluginsDir(), not
// an error and not a bare empty string.
func TestExternalPluginsDir_EmptyConfigFallsBackToDefault(t *testing.T) {
	want, err := defaultExternalPluginsDir()
	if err != nil {
		t.Fatalf("defaultExternalPluginsDir: %v", err)
	}

	got, err := externalPluginsDir(&config.Config{Plugins: config.PluginsConfig{Dir: "plugins"}})
	if err != nil {
		t.Fatalf("externalPluginsDir: %v", err)
	}
	if got != want {
		t.Errorf("expected the default %q, got %q", want, got)
	}
}

// TestExternalPluginsDir_AbsoluteConfigValueUsedVerbatim proves an
// explicitly configured absolute external_dir overrides the computed
// default entirely, used byte-exact.
func TestExternalPluginsDir_AbsoluteConfigValueUsedVerbatim(t *testing.T) {
	const abs = "/opt/topos/plugins-external-custom"
	got, err := externalPluginsDir(&config.Config{Plugins: config.PluginsConfig{ExternalDir: abs}})
	if err != nil {
		t.Fatalf("externalPluginsDir: %v", err)
	}
	if got != abs {
		t.Errorf("expected the absolute config value verbatim %q, got %q", abs, got)
	}
}

// TestExternalPluginsDir_RelativeConfigValueResolvesAgainstExecutable
// proves a relative external_dir resolves relative to the running
// executable's directory, exactly like pluginsDir already does for
// [plugins] dir — never relative to the current working directory.
func TestExternalPluginsDir_RelativeConfigValueResolvesAgainstExecutable(t *testing.T) {
	got, err := externalPluginsDir(&config.Config{Plugins: config.PluginsConfig{ExternalDir: "plugins-external"}})
	if err != nil {
		t.Fatalf("externalPluginsDir: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("expected a relative config value to resolve to an absolute path, got %q", got)
	}
	if filepath.Base(got) != "plugins-external" {
		t.Errorf("expected the resolved path to end in the configured relative value, got %q", got)
	}
}
