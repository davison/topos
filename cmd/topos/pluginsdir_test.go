// pluginsdir_test.go pins resolvePluginsDir's plugin-directory
// resolution branches (INST-03): these cases are what keep an installed
// instance ($PREFIX/bin/topos, plugins at $PREFIX/lib/topos/plugins)
// able to find its plugins with the stock relative `[plugins] dir`
// value and NO config edit — while a checkout build (bin/topos beside
// bin/plugins/) and an operator's absolute path keep resolving exactly
// as they always did. Every case builds its own real directory tree
// under t.TempDir(): the resolution is an existence probe, so a case
// that only compared strings would pass against a broken
// implementation.
package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/davison/topos/kernel/config"
)

// TestResolvePluginsDir covers every resolution branch, plus the
// adjacency and repeat-call determinism cases.
func TestResolvePluginsDir(t *testing.T) {
	// stockCfg returns a config carrying the real stock relative value
	// (kernel/config.DefaultPluginsDir), never a re-typed literal.
	stockCfg := func() *config.Config {
		return &config.Config{Plugins: config.PluginsConfig{Dir: config.DefaultPluginsDir}}
	}

	mkdirs := func(t *testing.T, paths ...string) {
		t.Helper()
		for _, p := range paths {
			if err := os.MkdirAll(p, 0o755); err != nil {
				t.Fatalf("MkdirAll(%q): %v", p, err)
			}
		}
	}

	cases := []struct {
		name string
		// setup builds the case's directory tree under root and returns
		// (cfg, exeDir, want). want is always an exact expected path.
		setup func(t *testing.T, root string) (cfg *config.Config, exeDir, want string)
	}{
		{
			// Branch 1: an absolute configured dir is returned verbatim,
			// regardless of what exists on disk and regardless of the
			// executable directory's name — nothing in this tree exists.
			name: "absolute configured dir returned verbatim",
			setup: func(t *testing.T, root string) (*config.Config, string, string) {
				abs := filepath.Join(root, "opt", "topos-plugins-custom")
				cfg := &config.Config{Plugins: config.PluginsConfig{Dir: abs}}
				exeDir := filepath.Join(root, "bin")
				mkdirs(t, exeDir)
				return cfg, exeDir, abs
			},
		},
		{
			// Branch 2: checkout layout — the executable directory has a
			// plugins subdirectory, so that path is returned even though
			// the installed-layout sibling ALSO exists (the checkout
			// probe wins; the sibling is never consulted).
			name: "checkout layout wins over coexisting installed sibling",
			setup: func(t *testing.T, root string) (*config.Config, string, string) {
				exeDir := filepath.Join(root, "bin")
				checkout := filepath.Join(exeDir, "plugins")
				sibling := filepath.Join(root, "lib", "topos", "plugins")
				mkdirs(t, checkout, sibling)
				return stockCfg(), exeDir, checkout
			},
		},
		{
			// Branch 3: installed layout — exe dir named "bin" with no
			// plugins subdirectory, parent holds lib/topos/plugins.
			name: "installed layout resolves the lib/topos sibling",
			setup: func(t *testing.T, root string) (*config.Config, string, string) {
				exeDir := filepath.Join(root, "bin")
				sibling := filepath.Join(root, "lib", "topos", "plugins")
				mkdirs(t, exeDir, sibling)
				return stockCfg(), exeDir, sibling
			},
		},
		{
			// Fallback: neither candidate exists — the executable-relative
			// join is returned so a "no plugins found" message names the
			// primary, documented location.
			name: "neither candidate exists falls back to exe-relative join",
			setup: func(t *testing.T, root string) (*config.Config, string, string) {
				exeDir := filepath.Join(root, "bin")
				mkdirs(t, exeDir)
				return stockCfg(), exeDir, filepath.Join(exeDir, config.DefaultPluginsDir)
			},
		},
		{
			// Gate: the installed-layout probe only applies when the
			// executable's own directory is named "bin" (the FHS shape
			// make install writes) — any other name returns the
			// exe-relative join even when a lib/topos/plugins sibling
			// exists.
			name: "non-bin exe dir never consults the installed sibling",
			setup: func(t *testing.T, root string) (*config.Config, string, string) {
				exeDir := filepath.Join(root, "build")
				sibling := filepath.Join(root, "lib", "topos", "plugins")
				mkdirs(t, exeDir, sibling)
				return stockCfg(), exeDir, filepath.Join(exeDir, config.DefaultPluginsDir)
			},
		},
		{
			// Adjacency: a sibling tree whose name merely shares a prefix
			// with the installed layout (lib/topos-extra/plugins) is never
			// selected — only the exact lib/topos/plugins shape counts.
			name: "prefix-sharing sibling directory is never selected",
			setup: func(t *testing.T, root string) (*config.Config, string, string) {
				exeDir := filepath.Join(root, "bin")
				near := filepath.Join(root, "lib", "topos-extra", "plugins")
				mkdirs(t, exeDir, near)
				return stockCfg(), exeDir, filepath.Join(exeDir, config.DefaultPluginsDir)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, exeDir, want := tc.setup(t, t.TempDir())
			got := resolvePluginsDir(cfg, exeDir)
			if got != want {
				t.Errorf("resolvePluginsDir(%q, %q) = %q, want %q",
					cfg.Plugins.Dir, exeDir, got, want)
			}
		})
	}

	// Repeat-call determinism: two calls against one identical tree must
	// return the same path — catches a future change that introduced a
	// filesystem-order dependency into the probes.
	t.Run("repeat call against identical tree is deterministic", func(t *testing.T) {
		root := t.TempDir()
		exeDir := filepath.Join(root, "bin")
		sibling := filepath.Join(root, "lib", "topos", "plugins")
		mkdirs(t, exeDir, sibling)

		first := resolvePluginsDir(stockCfg(), exeDir)
		second := resolvePluginsDir(stockCfg(), exeDir)
		if first != second {
			t.Errorf("two calls over an identical tree disagreed: %q then %q", first, second)
		}
		if want := filepath.Join(root, "lib", "topos", "plugins"); first != want {
			t.Errorf("expected the installed sibling %q, got %q", want, first)
		}
	})
}
