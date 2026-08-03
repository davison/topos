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
	// Username is the IMAP login username for a source using SRC-01's
	// Proton Mail plugin (Phase 3 deviation, Rule 2: paperless-ngx and
	// SilverBullet both authenticate with a bearer token alone; IMAP
	// authenticates with username+password, so a generic Source needs a
	// second identity field). Token is reused as the IMAP password for
	// this source type — there is no separate password field. Left empty
	// for a bearer-token source (paperless, SilverBullet), which never
	// reads this field.
	Username string `toml:"username,omitempty"`
	// WebmailBaseURL is the Proton webmail root for this account,
	// including the account index (e.g. "https://mail.proton.me/u/0") —
	// required by SRC-01's email plugin to build an ANCHORED deep link
	// into that account's All Mail view, pre-filled with a search for the
	// message's subject (03-RESEARCH.md Pitfall 5: no verified mapping
	// from an IMAP Message-Id/UID to Proton's internal webmail message id
	// exists, so the plugin never fetches this URL itself — it only ever
	// builds a link from it). All Mail, not the matched mailbox's own
	// view, is the target: Proton addresses custom labels and folders by
	// an internal id rather than by name, so a link built from a label's
	// name is not addressable and lands on the inbox instead. Left empty
	// for any source that has no equivalent webmail concept.
	WebmailBaseURL string `toml:"webmail_base_url,omitempty"`
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
	// Path is the filesystem path a local-path source reads from —
	// Signal Desktop's own config directory (default "~/.config/Signal")
	// for SRC-02's Signal plugin, the first source with no network
	// endpoint at all (04-01-PLAN.md; closes the relaxation STATE.md
	// logged as deferred from 02-04-SUMMARY.md). Left empty by every
	// network source (paperless, SilverBullet, Proton), which never read
	// it. A source declaring Path is validated differently: Validate
	// accepts it in place of BaseURL+Token rather than requiring both,
	// and kernel/pluginhost.launch adds it to WEBSPACES_SOURCE_CONFIG
	// under the "path" key so the plugin subprocess can locate its
	// source directory. Unlike CACert, a leading "~" here is expanded by
	// the plugin subprocess itself (plugins/signal/main.go), not by the
	// kernel — the kernel never needs to open this path itself, only
	// pass it through.
	Path string `toml:"path,omitempty"`
	// Agent declares this source's per-plugin agent grants (AGENT-01,
	// D-11): whether an automated caller through /agent/v1 may read this
	// source's items at all, and whether it may hand actions off through
	// this source's own interfaces (metadata only in this phase — see
	// AgentGrant). An absent [sources.<name>.agent] block decodes to the
	// Go zero value, which is default-deny for both grants — there is no
	// separate "enabled" key that could widen this; the absence of a
	// grant block IS the deny.
	Agent AgentGrant `toml:"agent"`
}

// AgentGrant is one source's per-plugin agent permission grant (AGENT-01,
// D-11). Read and Handoff are independent booleans: neither implies the
// other, and both default to false (deny) when the block or the key is
// absent, purely by Go's zero value — no special-case decoding needed, and
// deliberately no "default"/"enabled" key that could be set to widen this.
type AgentGrant struct {
	// Read grants an automated caller through /agent/v1 read access to
	// this source's items — the source appears in /agent/v1/sources, its
	// items appear in agent streams, and its items are readable through
	// the agent item routes. False (the zero value) means the source is
	// structurally absent from every agent-facing response, exactly as if
	// it did not exist (T-02-20 — no existence leak).
	Read bool `toml:"read"`
	// Handoff grants this source's action hand-off capability. Published
	// as metadata only in this phase (the "capabilities.handoff" field of
	// /agent/v1/sources) — no route in v1 acts on a Handoff grant; actual
	// agent-initiated actions are AGENT-11, deferred to v1.x.
	Handoff bool `toml:"handoff"`
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
