package pluginhost

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/go-hclog"
)

// PluginBinaryPrefix is the naming convention every plugin binary follows
// (D-11): DiscoverBinaries lists only files carrying this prefix, so the
// "+" chip picker's "New <plugin type>…" list is derived entirely from
// what is actually present on disk, never from a built-in table of known
// plugin types — the same "the kernel holds no built-in table of known
// plugin types" discipline docs/plugin-contract.md already states for the
// match vocabulary itself (D-05), extended here to plugin-type discovery.
const PluginBinaryPrefix = "topos-plugin-"

// ExcludedPluginBinaries names a binary DiscoverBinaries finds on disk but
// never offers through the "+" chip picker (07-RESEARCH.md Open Question
// 1, decided here): "topos-plugin-mock" is a developer/reference fixture
// — the plugin PLUG-05's fresh-context proof and the contract's own
// worked example build against — not a source an operator configuring a
// real deployment ever knowingly adds. Offering it in the picker would
// let a real config accidentally enable a fixed set of fake demo items
// with no corresponding real data anywhere. It is still discoverable by
// DiscoverBinaries' own return value filtering, not hidden from the
// directory listing itself — only excluded from what the UI surfaces.
//
// "topos-plugin-mockstrict" (quick task 260811-r5d) is excluded for the
// identical reason and joined the map after it reached a live operator's
// picker: plugins/mockstrict exists purely as browser-harness fixture
// infrastructure introduced by 07.1-02 and is never a real source. The
// exposure path was that `make e2e` builds topos-plugin-mockstrict into
// bin/plugins/ — the same directory `make build`/`make dev` populate and
// a real `[plugins] dir` config value can point at — so any developer who
// has ever run the harness has the binary sitting in their real plugin
// directory, and DiscoverBinaries offered it exactly like a real plugin
// type. As with mock, this is a catalog-listing policy only:
// DiscoverAllBinaries below is deliberately untouched, so an
// already-configured topos-plugin-mockstrict instance still describes,
// syncs, renders and remains editable.
var ExcludedPluginBinaries = map[string]bool{
	"topos-plugin-mock":       true,
	"topos-plugin-mockstrict": true,
}

// DiscoverBinaries lists pluginsDir for regular files whose name carries
// PluginBinaryPrefix and is not named in ExcludedPluginBinaries, returned
// sorted so the "+" chip picker's plugin-type order is stable run to run.
// A missing pluginsDir is a legitimate state — an operator who has not
// installed any plugin binaries yet — and returns an empty (never nil)
// slice with a nil error, never a failure.
//
// This is a UI-POLICY view (what may be OFFERED as a brand-new instance
// type) built on top of DiscoverAllBinaries' security-relevant raw
// listing — see that function's doc comment for why
// kernel/httpapi/config.go's DescribePluginHandler deliberately does NOT
// call this one. Symlinked plugin binaries (the e2e harness's fixture
// shape) are handled by DiscoverAllBinaries — see its doc comment.
func DiscoverBinaries(pluginsDir string) ([]string, error) {
	all, err := DiscoverAllBinaries(pluginsDir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(all))
	for _, name := range all {
		if ExcludedPluginBinaries[name] {
			continue
		}
		names = append(names, name)
	}
	return names, nil
}

// DiscoverAllBinaries lists pluginsDir exactly like DiscoverBinaries but
// WITHOUT applying ExcludedPluginBinaries — the raw, complete set of real
// on-disk plugin binaries, sorted. kernel/httpapi/config.go's
// DescribePluginHandler uses THIS function (not DiscoverBinaries) for its
// T-07-09 security check ("a request naming an arbitrary binary/path is
// refused... directory listing, never a caller-supplied path, is the
// authority over what may be launched"): ExcludedPluginBinaries is a
// UI-POLICY concern (never OFFER "topos-plugin-mock" as a brand-new
// instance type in the "+" picker) that must not also gate describing an
// instance that is ALREADY legitimately configured. Before this split,
// both concerns shared DiscoverBinaries' filtered result, which made
// POST /api/config/describe-plugin 404 for an already-configured
// topos-plugin-mock instance — breaking the "+" picker's one-step
// existing-instance add flow for every mock-typed instance (discovered
// live: 07.1-04-PLAN.md's re-add-a-removed-source spec, which the 07.1
// harness's own D-07 decision requires to use mock instances throughout).
// A missing pluginsDir behaves identically to DiscoverBinaries: a
// legitimate empty state, not an error.
//
// A directory entry is followed through one level of symlink before its
// regular-file-ness is judged (os.Stat, not os.Lstat/DirEntry.Type()):
// os.ReadDir's own DirEntry.Type() reports a SYMLINK's mode bits, which
// fs.FileMode.IsRegular() always reports false for — a plain
// `if !e.Type().IsRegular() { continue }` (this function's shape before
// this fix) silently discarded every entry in ANY symlinked plugins
// directory, discovering nothing at all. This is exactly how the 07.1
// browser-e2e-harness's own fixture populates a hermetic kernel's plugins
// dir (web/e2e/fixtures/plugin-binaries.ts's linkPluginBinaries
// deliberately symlinks rather than copies, per 07.1-01-SUMMARY.md's
// key-decisions) — so this bug made EVERY plugin type invisible to both
// PluginTypesHandler and DescribePluginHandler inside the hermetic e2e
// harness specifically, not only the mock-exclusion regression above.
// Live production installs (Makefile's `plugins`/`e2e` targets `go build`
// real files, never symlinks) never hit this path, which is why it went
// unnoticed until the harness's own symlink-based fixture exercised it.
func DiscoverAllBinaries(pluginsDir string) ([]string, error) {
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, PluginBinaryPrefix) {
			continue
		}
		if !isRegularFileFollowingSymlinks(pluginsDir, e) {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// isRegularFileFollowingSymlinks reports whether e — a DirEntry from
// pluginsDir — is, or resolves through exactly one os.Stat call to, a
// regular file. os.Stat (unlike os.Lstat/DirEntry.Type()) follows
// symlinks, so a plugin binary symlinked into pluginsDir (the e2e
// harness's own fixture shape) is correctly recognised without this
// function needing to special-case the symlink bit itself.
func isRegularFileFollowingSymlinks(dir string, e os.DirEntry) bool {
	if e.Type().IsRegular() {
		return true
	}
	if e.Type()&os.ModeSymlink == 0 {
		return false
	}
	info, err := os.Stat(filepath.Join(dir, e.Name()))
	if err != nil {
		// A dangling symlink is not a usable plugin binary — skip it
		// rather than failing the whole listing.
		return false
	}
	return info.Mode().IsRegular()
}

// Tier names which of the two configured plugin directories a binary
// resolved from (Phase 11, PLUG-06/PLUG-07). Trust is derived EXCLUSIVELY
// from this provenance fact — never from anything a plugin declares about
// itself via Describe, its filename, or a config key — so Tier is set
// once, at ResolveBinary time, and never overwritten from RPC data
// afterward (T-11-01).
type Tier string

const (
	// TierTrusted names a binary that resolved from Dirs.Trusted — built
	// from the davison/topos repo itself (`make build`/`make dev`).
	TierTrusted Tier = "trusted"
	// TierExternal names a binary that resolved from Dirs.External — code
	// topos did not build and cannot vouch for. Untrusted, not sandboxed:
	// an external-tier plugin runs with the same OS-level access as any
	// other plugin process (11-CONTEXT.md's explicit out-of-scope note on
	// sandboxing).
	TierExternal Tier = "external"
)

// Dirs is the two configured plugin directories every discovery and
// launch call site addresses a plugin binary through, replacing the
// single pluginsDir string every pre-Phase-11 call site used. Either
// field may be empty or name a directory that does not yet exist on
// disk — both are legitimate empty-tier states, never an error (mirrors
// DiscoverAllBinaries' own missing-directory contract, extended to two
// directories).
type Dirs struct {
	// Trusted is the existing single plugin directory (PluginsConfig.Dir)
	// every plugin binary resolved from before Phase 11 — code built from
	// the davison/topos repo.
	Trusted string
	// External is the second, untrusted directory Phase 11 introduces
	// (PluginsConfig.ExternalDir, D-09) — code topos did not build.
	External string
}

// TieredBinary is one binary name paired with the tier it resolved to —
// DiscoverTiered and DiscoverAllTiered's element type.
type TieredBinary struct {
	Name string
	Tier Tier
}

// DiscoverAllTiered is the two-tier SECURITY-AUTHORITY listing (the
// direct analogue of DiscoverAllBinaries, widened to two directories):
// every binary discoverable in EITHER configured directory, tagged with
// the tier it resolved to, sorted by name and de-duplicated. Callers that
// need to know "is this a legitimately launchable binary, and if so
// which tier" — kernel/httpapi/config.go's DescribePluginHandler
// membership check chief among them (T-11-02) — use this function, never
// DiscoverTiered, for the identical reason DiscoverAllBinaries exists
// alongside DiscoverBinaries: a UI-policy exclusion must never also gate
// what may legitimately be described or launched.
//
// D-11's shadow rule: a binary name present in BOTH directories resolves
// to TierTrusted and appears exactly once in the result — the trusted
// copy always wins a name collision, never silently (see ResolveBinary,
// below, for the launch-time half of this rule, which additionally logs
// the collision by name). A directory that is empty, missing, or whose
// Dirs field is the empty string contributes nothing and never errors —
// two absent directories return an empty (never nil) slice with a nil
// error, the same "operator hasn't installed anything yet" legitimacy
// DiscoverAllBinaries already grants a single missing directory.
func DiscoverAllTiered(dirs Dirs) ([]TieredBinary, error) {
	tierOf := make(map[string]Tier)

	if dirs.Trusted != "" {
		names, err := DiscoverAllBinaries(dirs.Trusted)
		if err != nil {
			return nil, fmt.Errorf("pluginhost: discover trusted plugins: %w", err)
		}
		for _, name := range names {
			tierOf[name] = TierTrusted
		}
	}

	if dirs.External != "" {
		names, err := DiscoverAllBinaries(dirs.External)
		if err != nil {
			return nil, fmt.Errorf("pluginhost: discover external plugins: %w", err)
		}
		for _, name := range names {
			if _, alreadyTrusted := tierOf[name]; alreadyTrusted {
				// D-11: trusted shadows external — the name stays mapped
				// to TierTrusted, set above; only the sorted-set MEMBERSHIP
				// changes here (it already does, since the map key already
				// exists), never the tier value. The loud, named log line
				// this rule also requires lives in ResolveBinary, the
				// launch-time call site, which has a logger to write to —
				// this discovery-only function deliberately has none, so it
				// stays a pure, side-effect-free listing.
				continue
			}
			tierOf[name] = TierExternal
		}
	}

	names := make([]string, 0, len(tierOf))
	for name := range tierOf {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]TieredBinary, 0, len(names))
	for _, name := range names {
		out = append(out, TieredBinary{Name: name, Tier: tierOf[name]})
	}
	return out, nil
}

// DiscoverTiered is the two-tier UI-POLICY catalog (the analogue of
// DiscoverBinaries): DiscoverAllTiered's result minus every name in
// ExcludedPluginBinaries. The exclusion is applied UNIFORMLY across both
// tiers — a fixture binary (topos-plugin-mock/-mockstrict) sitting in
// EITHER directory is excluded identically, because the exclusion exists
// to keep dev fixtures out of the "+" picker regardless of which
// directory a developer's copy happens to sit in (11-CONTEXT.md leaves
// this to planner discretion; decided here). As with DiscoverBinaries,
// this is a policy view over what may be OFFERED as a brand-new instance
// — it must never also gate what may be described or launched, which is
// exactly why DescribePluginHandler and ResolveBinary below use
// DiscoverAllTiered instead.
func DiscoverTiered(dirs Dirs) ([]TieredBinary, error) {
	all, err := DiscoverAllTiered(dirs)
	if err != nil {
		return nil, err
	}
	out := make([]TieredBinary, 0, len(all))
	for _, tb := range all {
		if ExcludedPluginBinaries[tb.Name] {
			continue
		}
		out = append(out, tb)
	}
	return out, nil
}

// validatePluginBinaryName rejects any name shape that could escape a
// configured plugin directory or resolve to something other than the
// directory member it claims to be (CR-01, 11-REVIEW.md; T-11-35/T-11-36).
// Its hand-kept twin is kernel/config/config.go's validateSourcePlugins —
// config must never import pluginhost (pluginhost already imports config,
// so the reverse would be an import cycle), so the same four rules are
// duplicated by hand there, exactly like the existing pinKeyPluginPrefix
// precedent. Both must be changed together.
//
// The four rules, in order:
//  1. an empty or whitespace-only name is rejected — filepath.Join(dir, "")
//     equals dir itself, which would otherwise resolve the plugins
//     directory as if it were a binary.
//  2. a name containing a '/' or a '\' is rejected — both separators
//     explicitly, so a value authored on Windows is rejected identically
//     on Linux, where filepath.Base does not recognise a backslash.
//  3. "." or ".." is rejected — with separators already barred above,
//     these are the only remaining ".."-segment forms a name can take.
//  4. a name that does not equal filepath.Base(name) is rejected — a
//     belt-and-braces restatement of rules 2/3 in the standard library's
//     own terms, so the guard tracks filepath semantics rather than only
//     this function's hand-enumerated cases.
//
// Deliberately NOT enforced here: the "topos-plugin-" prefix. Prefix
// filtering is DiscoverAllBinaries' catalog policy (D-10), not a
// resolution-shape rule — several existing callers resolve short names,
// and duplicating the prefix policy here would both break them and split
// ownership of a rule that already has a home.
func validatePluginBinaryName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("pluginhost: plugin binary name is empty")
	}
	if strings.ContainsRune(name, '/') || strings.ContainsRune(name, '\\') ||
		name == "." || name == ".." || name != filepath.Base(name) {
		return fmt.Errorf("pluginhost: invalid plugin binary name %q — must be a bare filename with no path separators or \"..\" segments", name)
	}
	return nil
}

// ResolveBinary is the one launch-time authority for turning a plugin
// binary NAME into a filesystem PATH plus the Tier it resolved to
// (T-11-01): every launch in this package — Discover, Reconcile, and
// DescribePluginType's trial launch — goes through this function via
// launch(), so tier is set from provenance at exactly one point and never
// re-derived or overwritten from anything the plugin process itself
// later reports.
//
// Confinement contract (CR-01, 11-REVIEW.md; T-11-35): name is validated
// by validatePluginBinaryName as this function's FIRST statement, before
// either configured directory is consulted at all — only a bare filename
// is resolvable, so a resolved path always lies directly inside one of
// the two configured directories. kernel/config/config.go's
// validateSourcePlugins is this contract's hand-kept twin at the
// config-validation end of the same chain (see that function's doc
// comment for why the rule is duplicated rather than imported).
//
// Resolution order implements D-11's shadow rule: dirs.Trusted is
// consulted first. When name exists there AS A REGULAR FILE (T-11-36:
// os.Stat, which follows symlinks — the e2e harness's symlinked plugin
// fixtures depend on this — but never a directory or other non-regular
// entry), ResolveBinary returns that path with TierTrusted immediately —
// but first checks whether name ALSO exists in dirs.External, and if so
// emits a named hclog.Warn line carrying the colliding binary name (a
// shadow must never be silent). When name does not resolve to a regular
// file in dirs.Trusted, dirs.External is checked next; a hit there
// returns TierExternal. Neither directory holding name returns an error
// naming the binary and both directories searched — an empty Dirs field
// is treated as "nothing to check there", not a separate failure mode,
// mirroring DiscoverAllBinaries' own missing-directory-is-empty-state
// contract.
func ResolveBinary(dirs Dirs, name string, logger hclog.Logger) (path string, tier Tier, err error) {
	if err := validatePluginBinaryName(name); err != nil {
		return "", "", err
	}

	if dirs.Trusted != "" {
		trustedPath := filepath.Join(dirs.Trusted, name)
		if info, statErr := os.Stat(trustedPath); statErr == nil && info.Mode().IsRegular() {
			if dirs.External != "" {
				externalPath := filepath.Join(dirs.External, name)
				if extInfo, extStatErr := os.Stat(externalPath); extStatErr == nil && extInfo.Mode().IsRegular() {
					if logger == nil {
						logger = hclog.NewNullLogger()
					}
					logger.Warn("plugin binary name shadowed: trusted copy wins, external copy ignored (D-11)",
						"binary", name)
				}
			}
			return trustedPath, TierTrusted, nil
		}
	}

	if dirs.External != "" {
		externalPath := filepath.Join(dirs.External, name)
		if info, statErr := os.Stat(externalPath); statErr == nil && info.Mode().IsRegular() {
			return externalPath, TierExternal, nil
		}
	}

	return "", "", fmt.Errorf("plugin binary %q not found in trusted directory %q or external directory %q", name, dirs.Trusted, dirs.External)
}
