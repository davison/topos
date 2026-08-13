package config

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/user"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// Load reads, expands, decodes and validates the config file at path,
// returning only the expanded runtime config — a thin wrapper over LoadRaw
// kept so every pre-existing caller (cmd/topos/main.go et al) compiles
// unchanged. Prefer LoadRaw directly wherever the raw pre-expansion form or
// the file hash is also needed (config.Store, the save dry-run path).
func Load(path string) (*Config, error) {
	expanded, _, _, _, err := LoadRaw(path)
	if err != nil {
		return nil, err
	}
	return expanded, nil
}

// LoadRaw reads the config file at path ONCE and produces both forms this
// phase needs (07-01-PLAN.md Task 1, D-05):
//
//   - expanded: the runtime config, exactly what Load has always returned —
//     os.Expand'd (${VAR}/$VAR resolved against the process environment),
//     home-dir-expanded, defaulted and validated. This is the form every
//     plugin/index/sync code path operates over; it may hold secret VALUES
//     in memory and must never be marshalled back to disk or serialized
//     over HTTP.
//   - raw: the same file, decoded WITHOUT os.Expand and WITHOUT home-dir
//     expansion — both of those touch fields that could be secret-shaped or
//     machine-specific, so raw retains ${VAR} references and "~"-prefixed
//     paths verbatim. This is the only form config.Store ever hands to
//     WriteCanonical or serializes as a GET /api/config response body — a
//     canonical rewrite of the expanded form would leak a resolved secret
//     value into the file (T-07-01/T-07-02, D-05's hard requirement).
//
// fileHash is the hex-encoded SHA-256 of the raw file bytes, used by
// config.Store's optimistic clobber lock (D-03). unknownKeys is
// UnknownKeys' strict-decode probe over the same raw bytes, used by the
// save path's lossless-rewrite guard (D-01's "flattens comments only", not
// data). raw applies only applyDefaults — never Validate, since a
// hand-edited file with ${VAR} references unresolved is not yet in a
// state Validate's field-presence checks can judge correctly; only
// expanded is ever validated.
func LoadRaw(path string) (expanded *Config, raw *Config, fileHash string, unknownKeys []string, err error) {
	rawBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, "", nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	expandedStr, missing := expandEnv(string(rawBytes))

	var expandedCfg Config
	if err := toml.Unmarshal([]byte(expandedStr), &expandedCfg); err != nil {
		return nil, nil, "", nil, fmt.Errorf("config: parse %s: %w", path, err)
	}
	applyDefaults(&expandedCfg)
	if err := expandedCfg.expandIndexPathHome(); err != nil {
		return nil, nil, "", nil, err
	}
	if err := expandedCfg.expandSourceCACertPathsHome(); err != nil {
		return nil, nil, "", nil, err
	}
	if err := expandedCfg.Validate(missing); err != nil {
		return nil, nil, "", nil, err
	}

	var rawCfg Config
	if err := toml.Unmarshal(rawBytes, &rawCfg); err != nil {
		return nil, nil, "", nil, fmt.Errorf("config: parse %s (raw): %w", path, err)
	}
	applyDefaults(&rawCfg)

	sum := sha256.Sum256(rawBytes)

	return &expandedCfg, &rawCfg, hex.EncodeToString(sum[:]), UnknownKeys(rawBytes), nil
}

// UnknownKeys strict-decodes raw TOML bytes into a throwaway Config and
// reports every key path the Config struct does not model, sorted for
// deterministic output. Used by config.Store.Save (D-01's lossless-rewrite
// prohibition): a canonical rewrite silently drops any key the struct
// doesn't know about, which is data loss the operator never consented to —
// this is the probe that catches it before a write happens.
//
// A decode error other than *toml.StrictMissingError (e.g. malformed TOML
// syntax) is not this function's concern — the normal (non-strict) decode
// elsewhere already reports that — so it returns nil rather than
// attempting to interpret an unrelated failure as "no unknown keys".
func UnknownKeys(raw []byte) []string {
	var probe Config
	dec := toml.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	err := dec.Decode(&probe)
	if err == nil {
		return nil
	}

	var strictErr *toml.StrictMissingError
	if !errors.As(err, &strictErr) {
		return nil
	}

	keySet := make(map[string]struct{}, len(strictErr.Errors))
	for _, de := range strictErr.Errors {
		keySet[strings.Join(de.Key(), ".")] = struct{}{}
	}
	keys := make([]string, 0, len(keySet))
	for k := range keySet {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
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

// applyDefaults guarantees two distinct things about cfg, called from
// LoadRaw on BOTH the expanded and raw forms (once each — the raw form is
// what Store.Raw() returns and GET/PUT /api/config and WriteCanonical all
// operate over): scalar defaults for the four settings below, and — since
// 07-12-PLAN.md Task 1 — non-nil collections for every field the config API
// exposes, so no consumer of either form ever observes a nil map or slice
// where the struct declares one.
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

	// Collection normalization (07-12-PLAN.md Task 1, closes 07-UAT.md
	// G-07-4's kernel-side half): a nil Go map or slice marshals as JSON
	// null via encoding/json, and the SPA iterates config.sources,
	// config.webspaces, and each webspace's keywords/sources/match
	// directly — Object.keys(null) throws, which is exactly the mechanism
	// that let a healthy, 200-OK kernel present as "the topos service
	// didn't respond" (.planning/debug/root-empty-state-service-error.md).
	// This mirrors kernel/httpapi/config.go's unknownKeysOrEmpty, which
	// already refuses to put a null collection in this same response body
	// — this block generalizes that same convention to every other
	// collection field the API exposes.
	//
	// Behaviour-neutral for Validate: every collection check in the
	// validation path (validateWebspaces, validateMatchBlocks,
	// validateSourcesAllowlist, validateFallbackCoverage,
	// validateDisplayNameUniqueness, Webspace.IsEmptyShell) is
	// len(...)-based, for which nil and empty are indistinguishable —
	// pinned mechanically (not just argued) by
	// TestApplyDefaults_NormalizesCollectionsWithoutChangingValidateVerdicts
	// in config_test.go, which includes 07-11's D-20 empty shell explicitly
	// since IsEmptyShell tests three collections at once and is the check
	// most sensitive to a nil-versus-empty distinction being introduced
	// anywhere.
	//
	// Invisible on disk: toml.Marshal already emits [] for a nil slice, so
	// WriteCanonical's written bytes are unchanged for every config valid
	// today — pinned by writer_test.go's unmodified byte-level golden and
	// fixed-point tests.
	//
	// The permanent-filter stack (`filter` in TOML/JSON) is deliberately
	// excluded: it is the one collection field carrying `omitempty`
	// (D-17/D-18), so a webspace with no permanent filter writes no
	// `filter` key at all — normalizing it to an empty slice would add a
	// meaningless key to every webspace block on the next canonical save.
	if cfg.Sources == nil {
		cfg.Sources = make(map[string]Source)
	}
	if cfg.Webspaces == nil {
		cfg.Webspaces = make(map[string]Webspace)
	}
	for name, ws := range cfg.Webspaces {
		// Webspace is a value type: mutating the range variable ws alone
		// would be discarded on loop exit, so any changed field must be
		// written back into cfg.Webspaces[name] explicitly.
		changed := false
		if ws.Keywords == nil {
			ws.Keywords = []string{}
			changed = true
		}
		if ws.Sources == nil {
			ws.Sources = []string{}
			changed = true
		}
		if ws.Match == nil {
			ws.Match = make(map[string]MatchBlock)
			changed = true
		}
		if changed {
			cfg.Webspaces[name] = ws
		}
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

	if err := cfg.validatePins(); err != nil {
		return err
	}

	return nil
}

// pinKeyPluginPrefix mirrors pluginhost.PluginBinaryPrefix
// (kernel/pluginhost/discover_binaries.go) verbatim, duplicated here rather
// than imported: config must never import pluginhost (config.Load runs
// before any plugin subprocess exists — pluginhost already imports config
// the other way, so importing it back here would be a cycle). Both
// constants must be kept in sync by hand if the naming convention ever
// changes.
const pinKeyPluginPrefix = "topos-plugin-"

// pinHashPattern matches exactly 64 lowercase hex characters — a SHA-256
// digest in the same lowercase hex.EncodeToString shape
// kernel/pluginhost.HashBinary produces (and kernel/config/store.go's
// fileHash already established project-wide). Uppercase hex or any other
// length is rejected: this repo has exactly one hashing convention, and a
// pin value that doesn't match it can never legitimately compare equal to
// what HashBinary computes at launch time.
var pinHashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// validatePins checks every [plugins.pins] entry (D-01/D-02, Phase 11):
// the key must look like a plugin binary name (the pluginBinaryPrefix
// convention, checked as a string shape only — config has no way to know
// which binaries actually exist on disk, and doesn't need to for this
// check), and the value must be exactly a 64-character lowercase hex
// SHA-256 digest. Keys are iterated in sorted order so a multi-error pins
// table reports the same offending key deterministically run to run,
// matching this file's own established discipline (validateWebspaces,
// validateDisplayNameUniqueness).
func (cfg *Config) validatePins() error {
	names := make([]string, 0, len(cfg.Plugins.Pins))
	for name := range cfg.Plugins.Pins {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if !strings.HasPrefix(name, pinKeyPluginPrefix) {
			return fmt.Errorf("config: [plugins.pins] key %q is not a plugin binary name (expected a name prefixed %q)", name, pinKeyPluginPrefix)
		}
		value := cfg.Plugins.Pins[name]
		if !pinHashPattern.MatchString(value) {
			return fmt.Errorf("config: [plugins.pins] value for %q is not a 64-character lowercase hex SHA-256 digest, got %q", name, value)
		}
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

		// D-20 (07-11-PLAN.md, closes 07-UAT.md G-07-3): an empty webspace
		// shell — no keywords, no match blocks, no sources allowlist — is
		// a legitimate, loadable config state meaning "a webspace that
		// exists and matches nothing yet." It is the exact document
		// web/src/lib/config-edit.ts's addWebspace() PUTs as the
		// create-webspace modal's first (of two) writes; 07-03/07-04's
		// D-14 flow populates match input in a LATER, separate save.
		//
		// Skipping the whole loop body here — rather than adding a branch
		// inside validateFallbackCoverage below — is deliberate:
		// validateFallbackCoverage would otherwise independently reject a
		// shell on any install carrying at least one [sources.*] block,
		// because an empty sources allowlist means every configured
		// instance participates (Phase 5 D-03, Webspace.Participates), so
		// a shell looks to that check like a webspace full of
		// participants with no coverage for any of them. Skipping the
		// loop body is the only placement that spares the shell without
		// weakening a single rule for any other webspace shape — a
		// webspace with a non-empty sources allowlist and no match input
		// is NOT a shell (IsEmptyShell requires all three collections
		// empty) and still fails below, exactly as it does today.
		if ws.IsEmptyShell() {
			continue
		}

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
