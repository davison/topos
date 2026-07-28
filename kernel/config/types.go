// Package config loads and validates the webspaces TOML config file.
package config

// Config is the root of ~/.config/webspaces/config.toml (or
// $XDG_CONFIG_HOME/webspaces/config.toml). One file holds kernel settings,
// source connections, and webspace definitions (D-04).
type Config struct {
	Server    ServerConfig        `toml:"server"`
	Index     IndexConfig         `toml:"index"`
	Plugins   PluginsConfig       `toml:"plugins"`
	Sync      SyncConfig          `toml:"sync"`
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

// SyncConfig configures the background scheduler's global sync interval
// (KERN-04, D-05). A source can override this with its own
// Source.SyncInterval.
type SyncConfig struct {
	Interval string `toml:"interval"` // Go duration string; default DefaultSyncInterval ("15m") if empty
}

// Source configures a single source plugin: which binary to launch and the
// connection details injected into its subprocess environment.
type Source struct {
	Plugin     string `toml:"plugin"`      // plugin binary name, e.g. "webspaces-plugin-paperless"
	BaseURL    string `toml:"base_url"`    // "${PAPERLESS_URL}" style env reference
	Token      string `toml:"token"`       // "${PAPERLESS_TOKEN}" style env reference — never a literal secret (D-04)
	APIVersion string `toml:"api_version"` // e.g. "10"
	// CACert is an optional filesystem path to a PEM-encoded CA
	// certificate a source plugin's HTTP client should trust in addition
	// to (by replacing, for that plugin's client only) the system trust
	// store. Deviation beyond the plan's originally scoped Source fields
	// (Rule 2): discovered live against the user's real SilverBullet
	// instance, which serves HTTPS behind a self-signed certificate the
	// system trust store does not contain — a plugin's Go HTTP client
	// otherwise cannot connect at all. Left empty for a source (like
	// paperless-ngx) whose instance uses a CA already in the system trust
	// store; the field itself is generic (not silverbullet-specific) since
	// any future LAN source could hit the same self-signed-cert situation.
	CACert string `toml:"ca_cert,omitempty"`
	// SyncInterval optionally overrides [sync] interval for this source
	// alone (D-05) — e.g. a heavy source can be slowed without affecting
	// every other configured source. A Go duration string; empty means
	// "use the global [sync] interval".
	SyncInterval string `toml:"sync_interval,omitempty"`
}

// Webspace declares a shared keyword list matched against every source's
// native categorization (D-02) — no per-source override.
type Webspace struct {
	Keywords []string `toml:"keywords"`
}

const (
	DefaultListen       = "127.0.0.1:7777"
	DefaultIndexPath    = "~/.local/share/webspaces/index.db"
	DefaultPluginsDir   = "plugins"
	DefaultSyncInterval = "15m"
)
