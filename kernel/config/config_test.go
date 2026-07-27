package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempConfig(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

func TestLoad_ValidConfigWithEnvExpansion(t *testing.T) {
	t.Setenv("TEST_PAPERLESS_URL", "http://paperless.lan:8000")
	t.Setenv("TEST_PAPERLESS_TOKEN", "secret-token")

	path := writeTempConfig(t, `
[server]
listen = "127.0.0.1:7777"

[index]
path = "/tmp/webspaces-index.db"

[sources.paperless]
plugin = "webspaces-plugin-paperless"
base_url = "${TEST_PAPERLESS_URL}"
token = "${TEST_PAPERLESS_TOKEN}"
api_version = "10"

[webspaces.house-move]
keywords = ["house-move", "House"]
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Sources["paperless"].BaseURL != "http://paperless.lan:8000" {
		t.Errorf("base_url not expanded, got %q", cfg.Sources["paperless"].BaseURL)
	}
	if cfg.Sources["paperless"].Token != "secret-token" {
		t.Errorf("token not expanded, got %q", cfg.Sources["paperless"].Token)
	}
	if len(cfg.Webspaces["house-move"].Keywords) != 2 {
		t.Errorf("expected 2 keywords, got %d", len(cfg.Webspaces["house-move"].Keywords))
	}
}

func TestLoad_Defaults(t *testing.T) {
	path := writeTempConfig(t, `
[webspaces.empty-space]
keywords = ["x"]
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Listen != DefaultListen {
		t.Errorf("expected default listen %q, got %q", DefaultListen, cfg.Server.Listen)
	}
	if cfg.Plugins.Dir != DefaultPluginsDir {
		t.Errorf("expected default plugins dir %q, got %q", DefaultPluginsDir, cfg.Plugins.Dir)
	}
}

func TestLoad_ZeroWebspacesStartsFine(t *testing.T) {
	path := writeTempConfig(t, `
[server]
listen = "127.0.0.1:7777"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load with zero webspaces should not error: %v", err)
	}
	if len(cfg.Webspaces) != 0 {
		t.Errorf("expected zero webspaces, got %d", len(cfg.Webspaces))
	}
}

func TestLoad_ZeroKeywordsFails(t *testing.T) {
	path := writeTempConfig(t, `
[webspaces.house-move]
keywords = []
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for zero keywords, got nil")
	}
	if !strings.Contains(err.Error(), "house-move") {
		t.Errorf("expected error to name the webspace, got: %v", err)
	}
}

func TestLoad_WhitespaceOnlyKeywordFails(t *testing.T) {
	path := writeTempConfig(t, `
[webspaces.house-move]
keywords = ["house-move", "   "]
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for whitespace-only keyword, got nil")
	}
}

func TestLoad_MissingEnvVarNamedInError(t *testing.T) {
	os.Unsetenv("TEST_UNSET_PAPERLESS_TOKEN")
	t.Setenv("TEST_UNSET_PAPERLESS_URL", "http://paperless.lan:8000")

	path := writeTempConfig(t, `
[sources.paperless]
plugin = "webspaces-plugin-paperless"
base_url = "${TEST_UNSET_PAPERLESS_URL}"
token = "${TEST_UNSET_PAPERLESS_TOKEN}"
api_version = "10"
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing env var, got nil")
	}
	if !strings.Contains(err.Error(), "TEST_UNSET_PAPERLESS_TOKEN") {
		t.Errorf("expected error to name TEST_UNSET_PAPERLESS_TOKEN, got: %v", err)
	}
}

func TestLoad_ExpandsHomeInIndexPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir available in this environment")
	}
	path := writeTempConfig(t, `
[index]
path = "~/.local/share/webspaces/index.db"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := home + "/.local/share/webspaces/index.db"
	if cfg.Index.Path != want {
		t.Errorf("expected index path %q, got %q", want, cfg.Index.Path)
	}
}
