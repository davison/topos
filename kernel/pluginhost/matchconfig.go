package pluginhost

import (
	"fmt"
	"sort"
	"strings"

	"github.com/davison/topos/kernel/config"
)

// ValidateMatchConfig implements D-05's second validation phase: the
// post-launch, pre-sync cross-check between a webspace's match
// configuration and each participating instance's launched plugin's own
// declared vocabulary (DescribeResponse.match_vocabulary). It cannot live
// in config.Validate, which is deliberately plugin-independent and runs
// before any subprocess exists (05-RESEARCH.md Pitfall 1) — it belongs
// beside the launched *Host, which is why it lives in this package instead.
//
// Webspaces, match block instances, and field names within a block are all
// iterated in sorted order, mirroring config.Validate's own discipline, so
// the reported error is deterministic run to run and never depends on Go's
// randomized map iteration order (KERN-07 ordering).
//
// Three failure shapes, each following config.Validate's
// "config: <subject> %q <problem>" error idiom:
//   - an explicit match block declares a field the instance's plugin did
//     not list in its Describe vocabulary — named with the offending
//     field, the plugin binary's display name, its source_type, and the
//     vocabulary it does declare;
//   - a match block, or a participating instance relying on the keywords
//     fallback, names an instance with no launched plugin — this should
//     not happen for a real config (pluginhost.Discover launches one
//     subprocess per configured source, or fails outright), but is
//     checked rather than assumed, so this function never passes
//     vacuously against a config/host pairing that don't actually agree;
//   - a participating instance relying on the keywords fallback has a
//     plugin that declared an empty vocabulary — there is no field for the
//     fallback to fan into (D-01 requires at least one).
func ValidateMatchConfig(cfg *config.Config, h *Host) error {
	byInstance := make(map[string]*Plugin, len(h.Plugins()))
	for _, p := range h.Plugins() {
		byInstance[p.Name()] = p
	}

	webspaceNames := make([]string, 0, len(cfg.Webspaces))
	for name := range cfg.Webspaces {
		webspaceNames = append(webspaceNames, name)
	}
	sort.Strings(webspaceNames)

	for _, wsName := range webspaceNames {
		ws := cfg.Webspaces[wsName]

		if err := validateMatchBlockVocabulary(wsName, ws, byInstance); err != nil {
			return err
		}
		if err := validateFallbackVocabulary(wsName, ws, cfg, byInstance); err != nil {
			return err
		}
	}

	return nil
}

// validateMatchBlockVocabulary checks every explicit
// [webspaces.<wsName>.match.<instance>] block against its instance's
// launched plugin vocabulary.
func validateMatchBlockVocabulary(wsName string, ws config.Webspace, byInstance map[string]*Plugin) error {
	instances := make([]string, 0, len(ws.Match))
	for instance := range ws.Match {
		instances = append(instances, instance)
	}
	sort.Strings(instances)

	for _, instance := range instances {
		p, ok := byInstance[instance]
		if !ok {
			return fmt.Errorf("config: webspace %q match block names source %q, which has no launched plugin", wsName, instance)
		}

		vocab := p.MatchVocabulary()
		vocabSet := make(map[string]bool, len(vocab))
		for _, v := range vocab {
			vocabSet[v] = true
		}

		block := ws.Match[instance]
		fields := make([]string, 0, len(block))
		for field := range block {
			fields = append(fields, field)
		}
		sort.Strings(fields)

		for _, field := range fields {
			if vocabSet[field] {
				continue
			}
			return fmt.Errorf(
				"config: webspace %q match block for source %q declares unknown match field %q — plugin %q (source_type %q) declares: [%s]",
				wsName, instance, field, p.PluginDisplayName(), p.SourceType(), joinVocabulary(vocab),
			)
		}
	}

	return nil
}

// validateFallbackVocabulary checks every participating instance that has
// no explicit match block in ws (and therefore relies on ws.Keywords, D-01)
// against its launched plugin's vocabulary: the plugin must exist and
// declare at least one field for the fallback to land in.
func validateFallbackVocabulary(wsName string, ws config.Webspace, cfg *config.Config, byInstance map[string]*Plugin) error {
	if len(ws.Keywords) == 0 {
		// config.Validate's validateFallbackCoverage already guarantees
		// every participating, block-less instance has a non-empty
		// Keywords fallback to rely on; an empty Keywords list here means
		// every participating instance has its own explicit block, already
		// checked above.
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
		if _, hasBlock := ws.Match[instance]; hasBlock {
			continue
		}

		p, ok := byInstance[instance]
		if !ok {
			return fmt.Errorf("config: webspace %q relies on the keywords fallback for source %q, which has no launched plugin", wsName, instance)
		}
		if len(p.MatchVocabulary()) == 0 {
			return fmt.Errorf(
				"config: webspace %q relies on the keywords fallback for source %q, but plugin %q (source_type %q) declares no match fields for the fallback to apply to",
				wsName, instance, p.PluginDisplayName(), p.SourceType(),
			)
		}
	}

	return nil
}

// joinVocabulary renders a plugin's declared vocabulary for an error
// message: comma-joined, or the literal "none" when the plugin declared
// zero fields.
func joinVocabulary(vocab []string) string {
	if len(vocab) == 0 {
		return "none"
	}
	return strings.Join(vocab, ", ")
}
