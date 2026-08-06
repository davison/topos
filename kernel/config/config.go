package config

import (
	"fmt"
	"os"
	"os/user"
	"sort"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// Load reads, expands, decodes and validates the config file at path.
//
// Secrets are never templated: the raw file bytes are expanded with
// os.Expand (${VAR} / $VAR references) before TOML decoding, and the
// expanded result is held in memory only — never written back to disk.
// A referenced-but-unset environment variable expands to the empty string
// and is reported by name in the returned error, not silently accepted.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	expanded, missing := expandEnv(string(raw))

	var cfg Config
	if err := toml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}

	applyDefaults(&cfg)

	if err := cfg.expandIndexPathHome(); err != nil {
		return nil, err
	}
	if err := cfg.expandSourceCACertPathsHome(); err != nil {
		return nil, err
	}

	if err := cfg.Validate(missing); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// expandEnv expands ${VAR}/$VAR references in raw using the current
// environment, and returns the sorted, de-duplicated list of variable
// names that were referenced but not set (so callers can name them in
// validation errors instead of silently treating them as empty strings).
func expandEnv(raw string) (string, []string) {
	missingSet := map[string]struct{}{}
	expanded := os.Expand(raw, func(name string) string {
		v, ok := os.LookupEnv(name)
		if !ok {
			missingSet[name] = struct{}{}
		}
		return v
	})

	missing := make([]string, 0, len(missingSet))
	for name := range missingSet {
		missing = append(missing, name)
	}
	sort.Strings(missing)

	return expanded, missing
}

func applyDefaults(cfg *Config) {
	if cfg.Server.Listen == "" {
		cfg.Server.Listen = DefaultListen
	}
	if cfg.Index.Path == "" {
		cfg.Index.Path = DefaultIndexPath
	}
	if cfg.Plugins.Dir == "" {
		cfg.Plugins.Dir = DefaultPluginsDir
	}
	if cfg.Sync.Interval == "" {
		cfg.Sync.Interval = DefaultSyncInterval
	}
}

// SyncIntervalFor returns the sync interval to use for sourceName: that
// source's own [sources.<name>] sync_interval when non-empty, else the
// global [sync] interval. Both are parsed with time.ParseDuration —
// Validate has already rejected an unparseable or non-positive value at
// load time, so a caller of SyncIntervalFor after a successful Load never
// sees this error in practice, but the signature still returns one
// (rather than panicking) so a hand-built *Config in a test isn't a
// footgun.
func (cfg *Config) SyncIntervalFor(sourceName string) (time.Duration, error) {
	raw := cfg.Sync.Interval
	if raw == "" {
		raw = DefaultSyncInterval
	}
	if src, ok := cfg.Sources[sourceName]; ok && src.SyncInterval != "" {
		raw = src.SyncInterval
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("config: sync interval %q for source %q: %w", raw, sourceName, err)
	}
	return d, nil
}

// AgentReadGrantedNames returns the config names (the [sources.<name>] key)
// of every source with agent.read = true. This is the sole authorization
// decision point for AGENT-01: a source with an absent [sources.<name>.agent]
// block, an absent read key, or an explicit read = false all decode to the
// same Go zero value (false) and are simply never added to the returned set
// — no special-case code distinguishes the three cases, which is what makes
// them structurally identical to callers (T-02-19).
func (cfg *Config) AgentReadGrantedNames() map[string]bool {
	granted := map[string]bool{}
	for name, src := range cfg.Sources {
		if src.Agent.Read {
			granted[name] = true
		}
	}
	return granted
}

// DisplayNameFor returns the resolved display name for the source instance
// keyed instance: its configured [sources.<instance>] display_name when
// non-blank, else the instance id itself (D-09). The kernel never emits an
// empty display name — an instance absent from cfg.Sources entirely (which
// should not happen for a real index row, but a defensive default all the
// same) also resolves to instance, never to "".
func (cfg *Config) DisplayNameFor(instance string) string {
	if src, ok := cfg.Sources[instance]; ok && strings.TrimSpace(src.DisplayName) != "" {
		return src.DisplayName
	}
	return instance
}

// expandIndexPathHome expands a leading "~" in [index] path to the current
// user's home directory.
func (cfg *Config) expandIndexPathHome() error {
	if !strings.HasPrefix(cfg.Index.Path, "~") {
		return nil
	}
	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("config: resolve home directory for index path: %w", err)
	}
	cfg.Index.Path = strings.Replace(cfg.Index.Path, "~", u.HomeDir, 1)
	return nil
}

// expandSourceCACertPathsHome expands a leading "~" in any configured
// [sources.<name>] ca_cert path to the current user's home directory —
// same convention as expandIndexPathHome, extended to this new field
// (deviation beyond the plan's originally scoped Source fields, added
// live during 02-01-PLAN.md Task 1 for the CA-cert-pinning need).
func (cfg *Config) expandSourceCACertPathsHome() error {
	for name, src := range cfg.Sources {
		if !strings.HasPrefix(src.CACert, "~") {
			continue
		}
		u, err := user.Current()
		if err != nil {
			return fmt.Errorf("config: resolve home directory for source %q ca_cert: %w", name, err)
		}
		src.CACert = strings.Replace(src.CACert, "~", u.HomeDir, 1)
		cfg.Sources[name] = src
	}
	return nil
}

// Validate checks structural correctness of the decoded config. missing is
// the list of environment variable names referenced in the raw file but
// unset, as returned by expandEnv — used to name the culprit when a
// required field is empty because of an unset variable rather than a
// simply-omitted key.
func (cfg *Config) Validate(missing []string) error {
	if err := cfg.validateWebspaces(); err != nil {
		return err
	}

	for name, src := range cfg.Sources {
		// A local-path source (Signal, SRC-02, and any future local-
		// database source) resolves everything it needs to open its
		// source directly from Path, so base_url/token are not required
		// and this branch skips both checks below entirely — see
		// kernel/config/types.go's Path doc comment.
		if strings.TrimSpace(src.Path) == "" {
			if strings.TrimSpace(src.BaseURL) == "" && strings.TrimSpace(src.Token) == "" {
				// Neither shape is present at all: name both accepted
				// shapes so a misconfigured local-path source (e.g. a
				// typo'd "path" key) gets an actionable message, rather
				// than only naming the first missing field.
				return fmt.Errorf("config: source %q must declare either base_url and token, or path%s", name, missingSuffix(missing))
			}
			if strings.TrimSpace(src.BaseURL) == "" {
				return fmt.Errorf("config: source %q has empty base_url%s", name, missingSuffix(missing))
			}
			if strings.TrimSpace(src.Token) == "" {
				return fmt.Errorf("config: source %q has empty token%s", name, missingSuffix(missing))
			}
		}
		if src.SyncInterval != "" {
			if err := validatePositiveDuration(src.SyncInterval); err != nil {
				return fmt.Errorf("config: [sources.%s] sync_interval: %w", name, err)
			}
		}
	}

	if err := validatePositiveDuration(cfg.Sync.Interval); err != nil {
		return fmt.Errorf("config: [sync] interval: %w", err)
	}

	if err := cfg.validateDisplayNameUniqueness(); err != nil {
		return err
	}

	return nil
}

// validateWebspaces checks every [webspaces.<name>] block against the
// per-instance match shape (D-01/D-02/D-03), independent of any launched
// plugin — the plugin-vocabulary cross-check (D-05) is a separate,
// post-launch phase (pluginhost.ValidateMatchConfig), since config.Load
// runs before any plugin subprocess exists (05-RESEARCH.md Pitfall 1).
// Webspace names, match block instance names, and field names within a
// block are all iterated in sorted order so the first reported error is
// deterministic run to run and never depends on Go's randomized map
// iteration order (KERN-07 ordering).
func (cfg *Config) validateWebspaces() error {
	names := make([]string, 0, len(cfg.Webspaces))
	for name := range cfg.Webspaces {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("config: webspace has empty name")
		}
		ws := cfg.Webspaces[name]

		if len(ws.Keywords) == 0 && len(ws.Match) == 0 {
			return fmt.Errorf("config: webspace %q declares neither a keywords fallback nor any match block — declare `keywords = [...]`, a `[webspaces.%s.match.<instance>]` block, or both", name, name)
		}

		for _, kw := range ws.Keywords {
			if strings.TrimSpace(kw) == "" {
				return fmt.Errorf("config: webspace %q declares an empty or whitespace-only keyword", name)
			}
		}

		if err := cfg.validateMatchBlocks(name, ws); err != nil {
			return err
		}
		if err := cfg.validateSourcesAllowlist(name, ws); err != nil {
			return err
		}
		if err := cfg.validateFallbackCoverage(name, ws); err != nil {
			return err
		}
	}

	return nil
}

// validateMatchBlocks checks every [webspaces.<name>.match.<instance>]
// block: the instance must be a configured source (unknown instance is a
// typo signal), must not be excluded by the same webspace's sources
// allowlist (dead config — 05-RESEARCH.md Open Question 1, decided here as
// a load-time error), and the block itself must declare at least one
// field, no empty field name, and no field with zero or empty/
// whitespace-only values (all silently-matches-nothing shapes D-06
// forbids).
func (cfg *Config) validateMatchBlocks(webspaceName string, ws Webspace) error {
	instances := make([]string, 0, len(ws.Match))
	for instance := range ws.Match {
		instances = append(instances, instance)
	}
	sort.Strings(instances)

	for _, instance := range instances {
		if _, ok := cfg.Sources[instance]; !ok {
			return fmt.Errorf("config: webspace %q match block names unknown source instance %q", webspaceName, instance)
		}
		if !ws.Participates(instance) {
			return fmt.Errorf("config: webspace %q declares a match block for source %q, which is excluded by this webspace's sources allowlist — remove one or the other", webspaceName, instance)
		}

		block := ws.Match[instance]
		if len(block) == 0 {
			return fmt.Errorf("config: webspace %q match block for source %q declares zero fields", webspaceName, instance)
		}

		fields := make([]string, 0, len(block))
		for field := range block {
			fields = append(fields, field)
		}
		sort.Strings(fields)

		for _, field := range fields {
			if strings.TrimSpace(field) == "" {
				return fmt.Errorf("config: webspace %q match block for source %q declares an empty field name", webspaceName, instance)
			}
			values := block[field]
			if len(values) == 0 {
				return fmt.Errorf("config: webspace %q match block for source %q field %q declares zero values", webspaceName, instance, field)
			}
			for _, v := range values {
				if strings.TrimSpace(v) == "" {
					return fmt.Errorf("config: webspace %q match block for source %q field %q declares an empty or whitespace-only value", webspaceName, instance, field)
				}
			}
		}
	}

	return nil
}

// validateSourcesAllowlist checks every [webspaces.<name>] sources entry
// names a configured source instance.
func (cfg *Config) validateSourcesAllowlist(webspaceName string, ws Webspace) error {
	for _, instance := range ws.Sources {
		if _, ok := cfg.Sources[instance]; !ok {
			return fmt.Errorf("config: webspace %q sources allowlist names unknown source instance %q", webspaceName, instance)
		}
	}
	return nil
}

// validateFallbackCoverage checks D-06: a participating instance with no
// explicit match block, in a webspace whose keywords fallback is empty, has
// nothing to resolve its match input from at sync time — this must fail
// loudly at load time, naming both accepted shapes, rather than silently
// matching nothing.
func (cfg *Config) validateFallbackCoverage(webspaceName string, ws Webspace) error {
	if len(ws.Keywords) > 0 {
		return nil
	}

	instances := make([]string, 0, len(cfg.Sources))
	for instance := range cfg.Sources {
		instances = append(instances, instance)
	}
	sort.Strings(instances)

	for _, instance := range instances {
		if !ws.Participates(instance) {
			continue
		}
		if _, ok := ws.Match[instance]; ok {
			continue
		}
		return fmt.Errorf("config: webspace %q has no keywords fallback and no match block for participating source %q — declare `keywords = [...]`, a `[webspaces.%s.match.%s]` block, or exclude %q via `sources`", webspaceName, instance, webspaceName, instance, instance)
	}

	return nil
}

// validateDisplayNameUniqueness enforces D-09: two source instances must
// never resolve to the same display name. Comparison is case-insensitive
// via strings.EqualFold, with no Unicode normalization — the same exactness
// convention D-03 set for match-field comparison — while instance ids
// themselves are compared byte-exact as TOML map keys, which go-toml/v2
// already guarantees are unique within [sources] by construction. Sources
// are iterated in sorted key order so the reported colliding pair is
// deterministic run to run rather than dependent on Go's randomized map
// iteration order.
func (cfg *Config) validateDisplayNameUniqueness() error {
	names := make([]string, 0, len(cfg.Sources))
	for name := range cfg.Sources {
		names = append(names, name)
	}
	sort.Strings(names)

	for i, name := range names {
		display := cfg.DisplayNameFor(name)
		for _, other := range names[i+1:] {
			otherDisplay := cfg.DisplayNameFor(other)
			if strings.EqualFold(display, otherDisplay) {
				return fmt.Errorf("config: sources %q and %q both resolve display_name %q — display names must be unique", name, other, display)
			}
		}
	}
	return nil
}

// validatePositiveDuration parses raw as a Go duration and rejects a zero
// or negative result — a zero interval would spin the scheduler's ticker
// at its minimum resolution, and a negative one panics time.NewTicker.
func validatePositiveDuration(raw string) error {
	d, err := time.ParseDuration(raw)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", raw, err)
	}
	if d <= 0 {
		return fmt.Errorf("must be a positive duration, got %q", raw)
	}
	return nil
}

func missingSuffix(missing []string) string {
	if len(missing) == 0 {
		return ""
	}
	return fmt.Sprintf(" (missing environment variable(s): %s)", strings.Join(missing, ", "))
}
