package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The per-instance filter map (M2-R3, #55): valid shape loads and
// round-trips; the three silently-broken shapes fail load by name.
func TestLoad_FilterBySource(t *testing.T) {
	write := func(t *testing.T, body string) string {
		t.Helper()
		dir := t.TempDir()
		path := filepath.Join(dir, "config.toml")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	base := `
[sources.mock-01]
plugin = "topos-plugin-mock"
path = "/tmp"

[webspaces.house]
keywords = ["demo"]
`
	t.Run("valid entry loads and is preserved", func(t *testing.T) {
		cfg, err := Load(write(t, base+`
[webspaces.house.filter_by_source]
mock-01 = ["boiler", "quote"]
`))
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		got := cfg.Webspaces["house"].FilterBySource["mock-01"]
		if len(got) != 2 || got[0] != "boiler" || got[1] != "quote" {
			t.Fatalf("FilterBySource round-trip: %v", got)
		}
	})
	for name, body := range map[string]string{
		"unknown instance": base + "\n[webspaces.house.filter_by_source]\nghost = [\"boiler\"]\n",
		"zero terms":       base + "\n[webspaces.house.filter_by_source]\nmock-01 = []\n",
		"whitespace term":  base + "\n[webspaces.house.filter_by_source]\nmock-01 = [\" \"]\n",
	} {
		t.Run(name+" fails load", func(t *testing.T) {
			if _, err := Load(write(t, body)); err == nil || !strings.Contains(err.Error(), "filter_by_source") {
				t.Fatalf("want a filter_by_source load error, got: %v", err)
			}
		})
	}
	t.Run("de-allowlisted instance fails load", func(t *testing.T) {
		_, err := Load(write(t, `
[sources.mock-01]
plugin = "topos-plugin-mock"
path = "/tmp"

[sources.mock-02]
plugin = "topos-plugin-mock"
path = "/tmp"

[webspaces.house]
keywords = ["demo"]
sources = ["mock-02"]

[webspaces.house.filter_by_source]
mock-01 = ["boiler"]
`))
		if err == nil || !strings.Contains(err.Error(), "excluded by this webspace's sources allowlist") {
			t.Fatalf("want the dead-config error, got: %v", err)
		}
	})
}
