package pluginhost

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
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
