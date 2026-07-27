// Package config loads and validates the webspaces TOML config file.
package config

// Config is the root of ~/.config/webspaces/config.toml (or
// $XDG_CONFIG_HOME/webspaces/config.toml). One file holds kernel settings,
// source connections, and webspace definitions (D-04).
type Config struct {
	Server    ServerConfig        `toml:"server"`
	Index     IndexConfig         `toml:"index"`
	Plugins   PluginsConfig       `toml:"plugins"`
	Sources   map[string]Source   `toml:"sources"`
	Webspaces map[string]Webspace `toml:"webspaces"`
}

// ServerConfig configures the kernel's loopback HTTP listener.
type ServerConfig struct {
	Listen string `toml:"listen"` // default "127.0.0.1:7777"
}

// IndexConfig configures the local SQLite index file location.
type IndexConfig struct {
	Path string `toml:"path"` // default "~/.local/share/webspaces/index.db"
}

// PluginsConfig configures where plugin binaries are discovered.
type PluginsConfig struct {
	Dir string `toml:"dir"` // default "plugins" (resolved relative to the running executable)
}

// Source configures a single source plugin: which binary to launch and the
// connection details injected into its subprocess environment.
type Source struct {
	Plugin     string `toml:"plugin"`      // plugin binary name, e.g. "webspaces-plugin-paperless"
	BaseURL    string `toml:"base_url"`    // "${PAPERLESS_URL}" style env reference
	Token      string `toml:"token"`       // "${PAPERLESS_TOKEN}" style env reference — never a literal secret (D-04)
	APIVersion string `toml:"api_version"` // e.g. "10"
}

// Webspace declares a shared keyword list matched against every source's
// native categorization (D-02) — no per-source override.
type Webspace struct {
	Keywords []string `toml:"keywords"`
}

const (
	DefaultListen     = "127.0.0.1:7777"
	DefaultIndexPath  = "~/.local/share/webspaces/index.db"
	DefaultPluginsDir = "plugins"
)
