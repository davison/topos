// Package config loads and validates the topos TOML config file.
package config

// Config is the root of ~/.config/topos/config.toml (or
// $XDG_CONFIG_HOME/topos/config.toml). One file holds kernel settings,
// source connections, and webspace definitions (D-04).
//
// json tags mirror every toml tag exactly (07-01-PLAN.md Task 1): the raw
// (unexpanded) form of this struct is what GET/PUT /api/config serializes
// over HTTP, so the config document has one stable wire shape shared by
// TOML-on-disk and JSON-over-the-wire — a field renamed in one tag and not
// the other would silently desync the UI from the file.
type Config struct {
	Server    ServerConfig        `toml:"server" json:"server"`
	Index     IndexConfig         `toml:"index" json:"index"`
	Plugins   PluginsConfig       `toml:"plugins" json:"plugins"`
	Sync      SyncConfig          `toml:"sync" json:"sync"`
	Sources   map[string]Source   `toml:"sources" json:"sources"`
	Webspaces map[string]Webspace `toml:"webspaces" json:"webspaces"`
}

// ServerConfig configures the kernel's loopback HTTP listener.
type ServerConfig struct {
	Listen string `toml:"listen" json:"listen"` // default "127.0.0.1:7777"
}

// IndexConfig configures the local SQLite index file location.
type IndexConfig struct {
	Path string `toml:"path" json:"path"` // default "~/.local/share/topos/index.db"
}

// PluginsConfig configures where plugin binaries are discovered.
type PluginsConfig struct {
	Dir string `toml:"dir" json:"dir"` // default "plugins" (resolved relative to the running executable)
}

// SyncConfig configures the background scheduler's global sync interval
// (KERN-04, D-05). A source can override this with its own
// Source.SyncInterval.
type SyncConfig struct {
	Interval string `toml:"interval" json:"interval"` // Go duration string; default DefaultSyncInterval ("15m") if empty
}

// Source configures a single source plugin: which binary to launch and the
// connection details injected into its subprocess environment.
type Source struct {
	Plugin string `toml:"plugin" json:"plugin"` // plugin binary name, e.g. "topos-plugin-paperless"
	// BaseURL and Token carry omitempty on both tags (07-01-PLAN.md Task 1,
	// RESEARCH.md Pitfall 3): without it, a canonical rewrite of a
	// local-path source (e.g. Signal, which uses Path instead) would emit
	// spurious base_url = ""/token = "" keys the operator never typed.
	BaseURL    string `toml:"base_url,omitempty" json:"base_url,omitempty"`       // "${PAPERLESS_URL}" style env reference
	Token      string `toml:"token,omitempty" json:"token,omitempty"`             // "${PAPERLESS_TOKEN}" style env reference — never a literal secret (D-04)
	APIVersion string `toml:"api_version,omitempty" json:"api_version,omitempty"` // e.g. "10"
	// Username is the IMAP login username for a source using SRC-01's
	// Proton Mail plugin (Phase 3 deviation, Rule 2: paperless-ngx and
	// SilverBullet both authenticate with a bearer token alone; IMAP
	// authenticates with username+password, so a generic Source needs a
	// second identity field). Token is reused as the IMAP password for
	// this source type — there is no separate password field. Left empty
	// for a bearer-token source (paperless, SilverBullet), which never
	// reads this field.
	Username string `toml:"username,omitempty" json:"username,omitempty"`
	// WebmailBaseURL is the Proton webmail root for this account,
	// including the account index (e.g. "https://mail.proton.me/u/0") —
	// required by SRC-01's email plugin to build an ANCHORED deep link
	// into that account's All Mail view, narrowed by a search for the
	// message's subject, sender and date (03-RESEARCH.md Pitfall 5: no
	// verified mapping from an IMAP Message-Id/UID to Proton's internal
	// webmail message id exists, so the plugin never fetches this URL
	// itself — it only ever builds a link from it). The extra sender/date
	// criteria narrow a generic subject's result list without upgrading
	// the link to point AT the message — it still lands the reader in a
	// short filtered list containing the message, adjacent to it rather
	// than on it. All Mail, not the matched mailbox's own view, is the
	// target: Proton addresses custom labels and folders by an internal
	// id rather than by name, so a link built from a label's name is not
	// addressable and lands on the inbox instead. Left empty for any
	// source that has no equivalent webmail concept.
	WebmailBaseURL string `toml:"webmail_base_url,omitempty" json:"webmail_base_url,omitempty"`
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
	CACert string `toml:"ca_cert,omitempty" json:"ca_cert,omitempty"`
	// SyncInterval optionally overrides [sync] interval for this source
	// alone (D-05) — e.g. a heavy source can be slowed without affecting
	// every other configured source. A Go duration string; empty means
	// "use the global [sync] interval".
	SyncInterval string `toml:"sync_interval,omitempty" json:"sync_interval,omitempty"`
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
	Path string `toml:"path,omitempty" json:"path,omitempty"`
	// Agent declares this source's per-plugin agent grants (AGENT-01,
	// D-11): whether an automated caller through /agent/v1 may read this
	// source's items at all, and whether it may hand actions off through
	// this source's own interfaces (metadata only in this phase — see
	// AgentGrant). An absent [sources.<name>.agent] block decodes to the
	// Go zero value, which is default-deny for both grants — there is no
	// separate "enabled" key that could widen this; the absence of a
	// grant block IS the deny.
	Agent AgentGrant `toml:"agent" json:"agent"`
	// DisplayName is this source instance's operator-authored label,
	// shown by the UI and published on every HTTP response that names a
	// source (D-09). Optional: when empty, the instance's display name
	// resolves to the instance id itself (the [sources.<id>] map key) via
	// Config.DisplayNameFor — the kernel never emits an empty display
	// name. Purely cosmetic (D-08): editing this value never changes
	// which instance an item, sync run, or agent grant belongs to — only
	// renaming the [sources.<id>] map key itself does that, because the
	// map key, not this field, is the instance's identity.
	DisplayName string `toml:"display_name,omitempty" json:"display_name,omitempty"`
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
	Read bool `toml:"read" json:"read"`
	// Handoff grants this source's action hand-off capability. Published
	// as metadata only in this phase (the "capabilities.handoff" field of
	// /agent/v1/sources) — no route in v1 acts on a Handoff grant; actual
	// agent-initiated actions are AGENT-11, deferred to v1.x.
	Handoff bool `toml:"handoff" json:"handoff"`
}

// MatchBlock is one source instance's explicit, typed match configuration
// within a webspace: field name (from that instance's plugin's declared
// vocabulary, DescribeResponse.match_vocabulary) to exact, case-insensitive
// values (D-04). A field name the plugin did not declare fails startup
// loudly via pluginhost.ValidateMatchConfig (D-05), never silently matching
// nothing.
type MatchBlock map[string][]string

// Webspace declares how each configured source instance's items land in
// this webspace (D-01/D-02/D-03), replacing the single shared per-webspace
// keyword list (05-01/05-02-PLAN.md's foundation; this phase's own KERN-07
// config half).
type Webspace struct {
	// Keywords is the optional fallback applied to every field of a
	// participating instance's declared vocabulary when that instance has
	// no explicit Match block (D-01). An instance WITH an explicit block
	// never sees this list — the block replaces the fallback outright for
	// that instance alone; the two are never combined (D-02). A webspace
	// must declare a non-empty Keywords list, at least one Match block, or
	// both — declaring neither fails config load (D-06).
	Keywords []string `toml:"keywords" json:"keywords"`
	// Sources is the optional participation allowlist (D-03): when
	// non-empty, only the named source instances participate in this
	// webspace — every other configured instance is skipped at sync time,
	// and its previously persisted rows for this webspace are cleared so
	// no orphaned rows survive the config change. Empty (the zero value)
	// means every configured instance participates by default. See
	// Participates.
	Sources []string `toml:"sources" json:"sources"`
	// Match is keyed by source instance id (the [sources.<id>] config
	// map key, D-08) — NOT by plugin type, so two instances of the same
	// plugin can carry independent blocks. An explicit block for an
	// instance replaces the Keywords fallback outright for that instance
	// (D-02); a Match entry naming an instance that is also excluded by a
	// non-empty Sources allowlist is dead config and fails load, a typo
	// signal rather than a staging feature (05-RESEARCH.md Open Question
	// 1, decided here).
	Match map[string]MatchBlock `toml:"match" json:"match"`
	// Filter is the promoted-search permanent filter stack (D-16/D-17/
	// D-18): each entry is an AND-ed FTS term, appended by "Save as
	// filter" and removed independently — a stackable, order-preserving
	// list rather than a set, since the persisted array order is also the
	// order the UI renders chips in (UI-12 ordering edge). It narrows
	// GET /api/webspaces/{ws}/stream, GET /api/webspaces/{ws}/search and
	// GET /agent/v1/webspaces/{ws}/stream identically — the filtered view
	// IS the webspace for every consumer, human and agent alike (D-16).
	// Index contents are never narrowed at sync time; this is a
	// query-time-only FTS filter, so removing a term instantly widens the
	// stream back out without any resync. Empty (the zero value) means no
	// permanent filter is active, and a webspace with no filter key
	// streams byte-identically to its pre-Phase-7 output.
	Filter []string `toml:"filter,omitempty" json:"filter,omitempty"`
}

// Participates reports whether source instance participates in webspace w:
// true when Sources is empty (every configured instance participates by
// default), or when instance is explicitly named in Sources (D-03).
func (w Webspace) Participates(instance string) bool {
	if len(w.Sources) == 0 {
		return true
	}
	for _, s := range w.Sources {
		if s == instance {
			return true
		}
	}
	return false
}

// IsEmptyShell reports whether w is D-20's "empty webspace shell"
// (07-11-PLAN.md, closes 07-UAT.md G-07-3): a webspace declaring none of
// Keywords, Sources or Match. This is the exact document
// web/src/lib/config-edit.ts's addWebspace() PUTs as the create-webspace
// modal's first write — 07-03/07-04's D-14 two-write flow deliberately
// creates an empty shell now and populates match input (and, per D-14,
// the sources allowlist) in a LATER, separate save. A shell is a
// legitimate, loadable config state meaning "a webspace that exists and
// matches nothing yet."
//
// All three conditions are required — a webspace naming instances in
// Sources while declaring no match input is NOT a shell. That shape is
// the operator-typo signal "allowlisted a source and then told it
// nothing to match" that config.Validate's validateWebspaces/
// validateFallbackCoverage exist to reject loudly; widening this
// predicate to two conditions would silently accept it instead.
//
// Filter is deliberately not considered: a permanent filter narrows a
// stream at query time (D-16/D-17/D-18) and cannot itself make a
// webspace match anything, so a webspace carrying only a filter and
// nothing else is still a shell for matching purposes.
func (w Webspace) IsEmptyShell() bool {
	return len(w.Keywords) == 0 && len(w.Sources) == 0 && len(w.Match) == 0
}

const (
	DefaultListen       = "127.0.0.1:7777"
	DefaultIndexPath    = "~/.local/share/topos/index.db"
	DefaultPluginsDir   = "plugins"
	DefaultSyncInterval = "15m"
)

// DefaultConfig returns the raw (never expanded) config written on first
// run, when cmd/topos's setup() finds no config.toml at all (09.1-BOOTSTRAP,
// planner_resolutions R1/R3). Its four scalar fields are set from the four
// Default* constants directly above; Sources and Webspaces are deliberately
// non-nil EMPTY maps rather than nil or a seeded example — a seeded example
// webspace would render a populated stream for a webspace the user never
// asked for, defeating the "friendly prompt to create your first webspace"
// framing 09.1-UI-SPEC.md requires. Non-nil is load-bearing, not stylistic:
// a nil map marshals to an omitted TOML key via go-toml/v2, while an empty
// non-nil map marshals to an explicit `[sources]`/`[webspaces]` table
// header, so the written file documents its own shape to whoever opens it.
//
// This value must only ever be passed to WriteCanonical in this exact raw
// form — never an expanded/secret-resolved config — per Phase 7's D-05
// secret discipline (WriteCanonical's own doc comment repeats this
// constraint at the write site).
func DefaultConfig() *Config {
	return &Config{
		Server:    ServerConfig{Listen: DefaultListen},
		Index:     IndexConfig{Path: DefaultIndexPath},
		Plugins:   PluginsConfig{Dir: DefaultPluginsDir},
		Sync:      SyncConfig{Interval: DefaultSyncInterval},
		Sources:   map[string]Source{},
		Webspaces: map[string]Webspace{},
	}
}
