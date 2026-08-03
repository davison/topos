package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestSyncIntervalFor_NoSyncBlockDefaultsToFifteenMinutes(t *testing.T) {
	path := writeTempConfig(t, `
[webspaces.house-move]
keywords = ["house"]
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	d, err := cfg.SyncIntervalFor("anything")
	if err != nil {
		t.Fatalf("SyncIntervalFor: %v", err)
	}
	if d != 15*time.Minute {
		t.Errorf("expected 15m default, got %v", d)
	}
}

func TestSyncIntervalFor_GlobalOverrideAppliesToEverySource(t *testing.T) {
	t.Setenv("TEST_SYNC_URL", "http://x.lan")
	t.Setenv("TEST_SYNC_TOKEN", "tok")
	path := writeTempConfig(t, `
[sync]
interval = "5m"

[sources.paperless]
plugin = "webspaces-plugin-paperless"
base_url = "${TEST_SYNC_URL}"
token = "${TEST_SYNC_TOKEN}"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	d, err := cfg.SyncIntervalFor("paperless")
	if err != nil {
		t.Fatalf("SyncIntervalFor: %v", err)
	}
	if d != 5*time.Minute {
		t.Errorf("expected 5m global interval, got %v", d)
	}
}

func TestSyncIntervalFor_PerSourceOverrideAppliesOnlyToThatSource(t *testing.T) {
	t.Setenv("TEST_SYNC_URL", "http://x.lan")
	t.Setenv("TEST_SYNC_TOKEN", "tok")
	t.Setenv("TEST_SYNC_URL2", "http://y.lan")
	t.Setenv("TEST_SYNC_TOKEN2", "tok2")
	path := writeTempConfig(t, `
[sync]
interval = "5m"

[sources.paperless]
plugin = "webspaces-plugin-paperless"
base_url = "${TEST_SYNC_URL}"
token = "${TEST_SYNC_TOKEN}"

[sources.silverbullet]
plugin = "webspaces-plugin-silverbullet"
base_url = "${TEST_SYNC_URL2}"
token = "${TEST_SYNC_TOKEN2}"
sync_interval = "1m"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	sb, err := cfg.SyncIntervalFor("silverbullet")
	if err != nil {
		t.Fatalf("SyncIntervalFor(silverbullet): %v", err)
	}
	if sb != time.Minute {
		t.Errorf("expected 1m override for silverbullet, got %v", sb)
	}
	pl, err := cfg.SyncIntervalFor("paperless")
	if err != nil {
		t.Fatalf("SyncIntervalFor(paperless): %v", err)
	}
	if pl != 5*time.Minute {
		t.Errorf("expected 5m global interval for paperless (no override), got %v", pl)
	}
}

func TestLoad_UnparseableIntervalFails(t *testing.T) {
	path := writeTempConfig(t, `
[sync]
interval = "soon"

[webspaces.house-move]
keywords = ["house"]
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for unparseable interval, got nil")
	}
	if !strings.Contains(err.Error(), "interval") {
		t.Errorf("expected error to name 'interval', got: %v", err)
	}
}

func TestLoad_ZeroIntervalFails(t *testing.T) {
	path := writeTempConfig(t, `
[sync]
interval = "0s"

[webspaces.house-move]
keywords = ["house"]
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for zero interval, got nil")
	}
}

func TestLoad_NegativeIntervalFails(t *testing.T) {
	path := writeTempConfig(t, `
[sync]
interval = "-1m"

[webspaces.house-move]
keywords = ["house"]
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for negative interval, got nil")
	}
}

func TestLoad_UnparseablePerSourceIntervalFailsNamingTheKey(t *testing.T) {
	t.Setenv("TEST_SYNC_URL3", "http://x.lan")
	t.Setenv("TEST_SYNC_TOKEN3", "tok")
	path := writeTempConfig(t, `
[sources.paperless]
plugin = "webspaces-plugin-paperless"
base_url = "${TEST_SYNC_URL3}"
token = "${TEST_SYNC_TOKEN3}"
sync_interval = "soon"
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for unparseable sync_interval, got nil")
	}
	if !strings.Contains(err.Error(), "sync_interval") {
		t.Errorf("expected error to name 'sync_interval', got: %v", err)
	}
}

// TestAgentReadGrantedNames_AbsentEmptyAndExplicitFalseAreAllDenied proves
// the three ways a source can end up unread-granted (no [agent] block at
// all, an empty [agent] block, and an explicit read = false) all produce
// the identical result: absent from AgentReadGrantedNames. This is the
// three-way equivalence T-02-19 depends on.
func TestAgentReadGrantedNames_AbsentEmptyAndExplicitFalseAreAllDenied(t *testing.T) {
	t.Setenv("TEST_AGENT_URL", "http://x.lan")
	t.Setenv("TEST_AGENT_TOKEN", "tok")
	path := writeTempConfig(t, `
[sources.no-block]
plugin = "x"
base_url = "${TEST_AGENT_URL}"
token = "${TEST_AGENT_TOKEN}"

[sources.empty-block]
plugin = "x"
base_url = "${TEST_AGENT_URL}"
token = "${TEST_AGENT_TOKEN}"
[sources.empty-block.agent]

[sources.explicit-false]
plugin = "x"
base_url = "${TEST_AGENT_URL}"
token = "${TEST_AGENT_TOKEN}"
[sources.explicit-false.agent]
read = false
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	granted := cfg.AgentReadGrantedNames()
	if len(granted) != 0 {
		t.Errorf("expected zero read-granted sources across all three deny-equivalent shapes, got %v", granted)
	}
}

// TestAgentReadGrantedNames_HandoffWithoutReadIsNotReadGranted proves the
// two grants are independent: a source with handoff = true and no read key
// (or read = false) is still absent from AgentReadGrantedNames, while its
// own Handoff field is still readable off the Config for capability
// publishing.
func TestAgentReadGrantedNames_HandoffWithoutReadIsNotReadGranted(t *testing.T) {
	t.Setenv("TEST_AGENT_URL2", "http://x.lan")
	t.Setenv("TEST_AGENT_TOKEN2", "tok")
	path := writeTempConfig(t, `
[sources.handoff-only]
plugin = "x"
base_url = "${TEST_AGENT_URL2}"
token = "${TEST_AGENT_TOKEN2}"
[sources.handoff-only.agent]
handoff = true
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if granted := cfg.AgentReadGrantedNames(); len(granted) != 0 {
		t.Errorf("expected handoff=true/read=false to not be read-granted, got %v", granted)
	}
	if !cfg.Sources["handoff-only"].Agent.Handoff {
		t.Error("expected Agent.Handoff to be true even though Read is false")
	}
}

// TestAgentReadGrantedNames_ExplicitReadTrueIsGranted is the positive case:
// a source that explicitly sets read = true is present in the granted set.
func TestAgentReadGrantedNames_ExplicitReadTrueIsGranted(t *testing.T) {
	t.Setenv("TEST_AGENT_URL3", "http://x.lan")
	t.Setenv("TEST_AGENT_TOKEN3", "tok")
	path := writeTempConfig(t, `
[sources.granted]
plugin = "x"
base_url = "${TEST_AGENT_URL3}"
token = "${TEST_AGENT_TOKEN3}"
[sources.granted.agent]
read = true
handoff = true
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	granted := cfg.AgentReadGrantedNames()
	if !granted["granted"] {
		t.Errorf("expected 'granted' source to be read-granted, got %v", granted)
	}
}

// TestLoad_PathOnlySourceValidatesCleanly proves a source declaring only
// plugin and path (no base_url, no token) validates without error —
// SRC-02's Signal plugin is the first source of this shape (04-01-PLAN.md
// Task 2).
func TestLoad_PathOnlySourceValidatesCleanly(t *testing.T) {
	path := writeTempConfig(t, `
[sources.signal]
plugin = "webspaces-plugin-signal"
path = "~/.config/Signal"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Sources["signal"].Path != "~/.config/Signal" {
		t.Errorf("expected path to be preserved, got %q", cfg.Sources["signal"].Path)
	}
}

// TestLoad_SourceWithNeitherPathNorBaseURLTokenFailsNamingBothShapes
// proves a source declaring none of path/base_url/token fails config
// load with an error naming BOTH accepted shapes, not just the first
// missing field.
func TestLoad_SourceWithNeitherPathNorBaseURLTokenFailsNamingBothShapes(t *testing.T) {
	path := writeTempConfig(t, `
[sources.broken]
plugin = "webspaces-plugin-broken"
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for a source declaring none of path/base_url/token, got nil")
	}
	if !strings.Contains(err.Error(), "base_url") || !strings.Contains(err.Error(), "path") {
		t.Errorf("expected error to name both accepted shapes (base_url and path), got: %v", err)
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
