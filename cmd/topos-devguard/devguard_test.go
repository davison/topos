// devguard_test.go pins every behaviour of the dev isolation guard
// (ISOL-01/ISOL-02): a dev run must refuse to start when any writable
// path its config declares — the config file itself, the index, either
// plugin directory (including the omitted-external-dir default), or a
// source's own store path — resolves inside the topos config root or
// state root the installed instance owns. Every case builds its config
// and directory tree under t.TempDir() and points the config-home and
// data-home environment variables into that tree, so no case can ever
// touch the developer's real directories.
package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// guardEnv points every root-deriving environment variable into tmp and
// returns the derived config root, state root, and a checkout directory
// for legitimately-isolated paths.
func guardEnv(t *testing.T, tmp string) (configRoot, stateRoot, checkout string) {
	t.Helper()
	xdgConfig := filepath.Join(tmp, "xdg-config")
	xdgData := filepath.Join(tmp, "xdg-data")
	home := filepath.Join(tmp, "home")
	checkout = filepath.Join(tmp, "checkout")
	for _, d := range []string{xdgConfig, xdgData, home, checkout} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("MkdirAll(%q): %v", d, err)
		}
	}
	t.Setenv("XDG_CONFIG_HOME", xdgConfig)
	t.Setenv("XDG_DATA_HOME", xdgData)
	t.Setenv("HOME", home)
	return filepath.Join(xdgConfig, "topos"), filepath.Join(xdgData, "topos"), checkout
}

// writeConfig writes a config file at path (creating parents) and
// returns path.
func writeConfig(t *testing.T, path, content string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
	return path
}

// isolatedConfig returns a config document whose kernel paths all sit
// inside checkout — the shape a healthy generated config.dev.toml has.
func isolatedConfig(checkout string, extra string) string {
	return fmt.Sprintf(`[server]
listen = "127.0.0.1:7778"

[index]
path = "%s/bin/topos-dev-index.db"

[plugins]
dir = "%s/bin/plugins"
external_dir = "%s/bin/plugins-external-dev"
%s`, checkout, checkout, checkout, extra)
}

// runGuard runs the guard once and returns exit code, stdout, stderr.
func runGuard(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestDevguard(t *testing.T) {
	t.Run("fully isolated config with zero sources exits 0 with zero violations", func(t *testing.T) {
		tmp := t.TempDir()
		_, _, checkout := guardEnv(t, tmp)
		cfg := writeConfig(t, filepath.Join(checkout, "config.dev.toml"), isolatedConfig(checkout, ""))

		code, stdout, stderr := runGuard(t, "--config", cfg)
		if code != 0 {
			t.Fatalf("expected exit 0, got %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
		}
		if strings.Contains(stdout+stderr, "VIOLATION") {
			t.Errorf("expected zero violations, got:\n%s%s", stdout, stderr)
		}
	})

	t.Run("index path inside the state root is exactly one violation", func(t *testing.T) {
		tmp := t.TempDir()
		_, stateRoot, checkout := guardEnv(t, tmp)
		cfg := writeConfig(t, filepath.Join(checkout, "config.dev.toml"), fmt.Sprintf(`[server]
listen = "127.0.0.1:7778"

[index]
path = "%s/index.db"

[plugins]
dir = "%s/bin/plugins"
external_dir = "%s/bin/plugins-external-dev"
`, stateRoot, checkout, checkout))

		code, stdout, _ := runGuard(t, "--config", cfg)
		if code == 0 {
			t.Fatalf("expected a non-zero exit, got 0\n%s", stdout)
		}
		if got := strings.Count(stdout, "VIOLATION"); got != 1 {
			t.Errorf("expected exactly 1 violation, got %d:\n%s", got, stdout)
		}
		if !strings.Contains(stdout, "[index] path") {
			t.Errorf("violation does not name the [index] path key:\n%s", stdout)
		}
	})

	t.Run("omitted external_dir resolves to the state-root default and is a violation", func(t *testing.T) {
		tmp := t.TempDir()
		_, _, checkout := guardEnv(t, tmp)
		cfg := writeConfig(t, filepath.Join(checkout, "config.dev.toml"), fmt.Sprintf(`[server]
listen = "127.0.0.1:7778"

[index]
path = "%s/bin/topos-dev-index.db"

[plugins]
dir = "%s/bin/plugins"
`, checkout, checkout))

		code, stdout, _ := runGuard(t, "--config", cfg)
		if code == 0 {
			t.Fatalf("expected a non-zero exit for the omitted external_dir default, got 0\n%s", stdout)
		}
		if !strings.Contains(stdout, "[plugins] external_dir") {
			t.Errorf("violation does not name the [plugins] external_dir key:\n%s", stdout)
		}
		if !strings.Contains(stdout, "plugins-external") {
			t.Errorf("violation does not show the resolved platform default:\n%s", stdout)
		}
	})

	t.Run("source store path inside the state root is a violation naming that source", func(t *testing.T) {
		tmp := t.TempDir()
		_, stateRoot, checkout := guardEnv(t, tmp)
		cfg := writeConfig(t, filepath.Join(checkout, "config.dev.toml"), isolatedConfig(checkout, fmt.Sprintf(`
[sources.whatsapp]
plugin = "topos-plugin-whatsapp"
path = "%s/whatsapp-store"
`, stateRoot)))

		code, stdout, _ := runGuard(t, "--config", cfg)
		if code == 0 {
			t.Fatalf("expected a non-zero exit, got 0\n%s", stdout)
		}
		if !strings.Contains(stdout, "[sources.whatsapp] path") {
			t.Errorf("violation does not name the source's own key path:\n%s", stdout)
		}
	})

	t.Run("read-only source location outside the topos roots is clean", func(t *testing.T) {
		tmp := t.TempDir()
		_, _, checkout := guardEnv(t, tmp)
		signalDir := filepath.Join(tmp, "home", ".config", "Signal")
		if err := os.MkdirAll(signalDir, 0o755); err != nil {
			t.Fatal(err)
		}
		cfg := writeConfig(t, filepath.Join(checkout, "config.dev.toml"), isolatedConfig(checkout, fmt.Sprintf(`
[sources.signal]
plugin = "topos-plugin-signal"
path = "%s"
`, signalDir)))

		code, stdout, stderr := runGuard(t, "--config", cfg)
		if code != 0 {
			t.Fatalf("expected exit 0 for a read-only source outside the roots, got %d\n%s%s", code, stdout, stderr)
		}
		if strings.Contains(stdout+stderr, "VIOLATION") {
			t.Errorf("expected zero violations:\n%s%s", stdout, stderr)
		}
	})

	t.Run("path exactly equal to a root is a violation", func(t *testing.T) {
		tmp := t.TempDir()
		_, stateRoot, checkout := guardEnv(t, tmp)
		if err := os.MkdirAll(stateRoot, 0o755); err != nil {
			t.Fatal(err)
		}
		cfg := writeConfig(t, filepath.Join(checkout, "config.dev.toml"), fmt.Sprintf(`[server]
listen = "127.0.0.1:7778"

[index]
path = "%s/bin/topos-dev-index.db"

[plugins]
dir = "%s/bin/plugins"
external_dir = "%s"
`, checkout, checkout, stateRoot))

		code, stdout, _ := runGuard(t, "--config", cfg)
		if code == 0 {
			t.Fatalf("expected a non-zero exit for a path equal to the state root, got 0\n%s", stdout)
		}
		if !strings.Contains(stdout, "[plugins] external_dir") {
			t.Errorf("violation does not name the key:\n%s", stdout)
		}
	})

	t.Run("sibling directory sharing a leading string with a root is clean", func(t *testing.T) {
		tmp := t.TempDir()
		_, stateRoot, checkout := guardEnv(t, tmp)
		sibling := stateRoot + "-extra"
		for _, d := range []string{stateRoot, sibling} {
			if err := os.MkdirAll(d, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		cfg := writeConfig(t, filepath.Join(checkout, "config.dev.toml"), fmt.Sprintf(`[server]
listen = "127.0.0.1:7778"

[index]
path = "%s/index.db"

[plugins]
dir = "%s/bin/plugins"
external_dir = "%s/bin/plugins-external-dev"
`, sibling, checkout, checkout))

		code, stdout, stderr := runGuard(t, "--config", cfg)
		if code != 0 {
			t.Fatalf("expected exit 0 for a prefix-sharing sibling, got %d\n%s%s", code, stdout, stderr)
		}
		if strings.Contains(stdout+stderr, "VIOLATION") {
			t.Errorf("expected zero violations for the sibling directory:\n%s%s", stdout, stderr)
		}
	})

	t.Run("config file inside the config root is itself a violation", func(t *testing.T) {
		tmp := t.TempDir()
		configRoot, _, checkout := guardEnv(t, tmp)
		cfg := writeConfig(t, filepath.Join(configRoot, "config.toml"), isolatedConfig(checkout, ""))

		code, stdout, _ := runGuard(t, "--config", cfg)
		if code == 0 {
			t.Fatalf("expected a non-zero exit for the config file inside the config root, got 0\n%s", stdout)
		}
		if !strings.Contains(stdout, "config") || !strings.Contains(stdout, "config root") {
			t.Errorf("violation does not name the config path and root:\n%s", stdout)
		}
	})

	t.Run("multiple violations are all reported in one deterministic pass", func(t *testing.T) {
		tmp := t.TempDir()
		configRoot, stateRoot, checkout := guardEnv(t, tmp)
		_ = configRoot
		cfg := writeConfig(t, filepath.Join(checkout, "config.dev.toml"), fmt.Sprintf(`[server]
listen = "127.0.0.1:7778"

[index]
path = "%s/index.db"

[plugins]
dir = "%s/bin/plugins"

[sources.whatsapp]
plugin = "topos-plugin-whatsapp"
path = "%s/whatsapp-store"

[sources.proton]
plugin = "topos-plugin-proton"
path = "%s/proton-cache"
`, stateRoot, checkout, stateRoot, stateRoot))

		code1, stdout1, _ := runGuard(t, "--config", cfg)
		code2, stdout2, _ := runGuard(t, "--config", cfg)
		if code1 == 0 {
			t.Fatalf("expected a non-zero exit, got 0\n%s", stdout1)
		}
		if code1 != code2 || stdout1 != stdout2 {
			t.Errorf("two runs over one config disagreed:\nrun1 (%d):\n%s\nrun2 (%d):\n%s", code1, stdout1, code2, stdout2)
		}
		// One pass reports everything: index, omitted external_dir
		// default, and both source stores.
		for _, key := range []string{"[index] path", "[plugins] external_dir", "[sources.proton] path", "[sources.whatsapp] path"} {
			if !strings.Contains(stdout1, key) {
				t.Errorf("missing violation for %s in:\n%s", key, stdout1)
			}
		}
		// Sorted by config key path: [index] before [plugins] before
		// [sources.proton] before [sources.whatsapp].
		idx := func(s string) int { return strings.Index(stdout1, s) }
		if !(idx("[index] path") < idx("[plugins] external_dir") &&
			idx("[plugins] external_dir") < idx("[sources.proton] path") &&
			idx("[sources.proton] path") < idx("[sources.whatsapp] path")) {
			t.Errorf("violations are not sorted by config key path:\n%s", stdout1)
		}
	})

	t.Run("warn-only reports identical findings and exits 0", func(t *testing.T) {
		tmp := t.TempDir()
		_, stateRoot, checkout := guardEnv(t, tmp)
		doc := fmt.Sprintf(`[server]
listen = "127.0.0.1:7778"

[index]
path = "%s/index.db"

[plugins]
dir = "%s/bin/plugins"
external_dir = "%s/bin/plugins-external-dev"
`, stateRoot, checkout, checkout)
		cfg := writeConfig(t, filepath.Join(checkout, "config.dev.toml"), doc)

		refuseCode, refuseOut, _ := runGuard(t, "--config", cfg)
		warnCode, _, warnErr := runGuard(t, "--config", cfg, "--warn-only")
		if refuseCode == 0 {
			t.Fatalf("expected the refusing run to exit non-zero\n%s", refuseOut)
		}
		if warnCode != 0 {
			t.Fatalf("expected warn-only to exit 0, got %d\n%s", warnCode, warnErr)
		}
		if !strings.Contains(warnErr, "[index] path") {
			t.Errorf("warn-only did not report the same finding on stderr:\n%s", warnErr)
		}
		if !strings.Contains(warnErr, "BYPASS") {
			t.Errorf("warn-only did not print the bypass banner:\n%s", warnErr)
		}
	})

	t.Run("expected-port mismatch is a violation naming both ports", func(t *testing.T) {
		tmp := t.TempDir()
		_, _, checkout := guardEnv(t, tmp)
		cfg := writeConfig(t, filepath.Join(checkout, "config.dev.toml"), strings.Replace(
			isolatedConfig(checkout, ""), "127.0.0.1:7778", "127.0.0.1:7777", 1))

		code, stdout, _ := runGuard(t, "--config", cfg, "--expected-port", "7778")
		if code == 0 {
			t.Fatalf("expected a non-zero exit for a port mismatch, got 0\n%s", stdout)
		}
		if !strings.Contains(stdout, "7777") || !strings.Contains(stdout, "7778") {
			t.Errorf("port violation does not name both ports:\n%s", stdout)
		}
		if !strings.Contains(stdout, "[server] listen") {
			t.Errorf("port violation does not name the [server] listen key:\n%s", stdout)
		}
	})

	t.Run("expected-port match with isolated config exits 0", func(t *testing.T) {
		tmp := t.TempDir()
		_, _, checkout := guardEnv(t, tmp)
		cfg := writeConfig(t, filepath.Join(checkout, "config.dev.toml"), isolatedConfig(checkout, ""))

		code, stdout, stderr := runGuard(t, "--config", cfg, "--expected-port", "7778")
		if code != 0 {
			t.Fatalf("expected exit 0, got %d\n%s%s", code, stdout, stderr)
		}
	})
}
