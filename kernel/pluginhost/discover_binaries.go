package pluginhost

import (
	"os"
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
var ExcludedPluginBinaries = map[string]bool{
	"topos-plugin-mock": true,
}

// DiscoverBinaries lists pluginsDir for regular files whose name carries
// PluginBinaryPrefix and is not named in ExcludedPluginBinaries, returned
// sorted so the "+" chip picker's plugin-type order is stable run to run.
// A missing pluginsDir is a legitimate state — an operator who has not
// installed any plugin binaries yet — and returns an empty (never nil)
// slice with a nil error, never a failure.
func DiscoverBinaries(pluginsDir string) ([]string, error) {
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, PluginBinaryPrefix) {
			continue
		}
		if ExcludedPluginBinaries[name] {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}
