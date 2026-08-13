// Package main implements topos-plugin-external-demo (Phase 11, ROADMAP
// success criterion 5): a standalone, out-of-repo-shaped source plugin
// built entirely from the published contract (docs/plugin-contract.md),
// the wire contract (proto/topos/v1/plugin.proto), and the sdk module —
// exactly as a genuine third-party plugin author would build it, with no
// access to any other file in this repository.
//
// This module exists ONLY to prove the external-plugin mechanism end to
// end (kernel/supervisor/externalproof_test.go's
// TestExternalProof_OutOfRepoBinaryEndToEnd): it is never shipped, never
// built by `make build`/`make plugins`/`make plugins-portable`, and must
// never be copied into a real installation's trusted plugin directory —
// see README.md.
//
// Its Match implementation is deliberately observational rather than
// illustrative: every item it returns reports back exactly what config
// and environment the kernel actually handed this process, so the
// extras-passthrough (PLUG-09) and launch-environment allowlist (D-14)
// claims this phase makes are provable by inspecting the synced corpus,
// not merely asserted by a test's own mock.
package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

const (
	sourceType      = "external-demo"
	displayName     = "External Demo (untrusted)"
	contractVersion = "topos.v2"

	// sourceSystem stands in for a real base URL / connection string,
	// exactly like plugins/mock's own sourceSystem placeholder — this
	// proof plugin has no real remote instance, but Provenance's
	// "source_system" key is documented as required on every item.
	sourceSystem = "external-demo://out-of-repo"

	// pluginBinaryName is this module's own build-target name (Makefile's
	// external-demo target), used only in Provenance's "plugin" key.
	pluginBinaryName = "topos-plugin-external-demo"

	// anchorSourceID is the one item Match always returns regardless of
	// configured extras or visible environment, so a match against this
	// plugin's declared label never comes back empty.
	anchorSourceID = "anchor"

	// itemLabel is the one fixed native-categorization label every item
	// this plugin returns carries — the supervisor proof test's webspace
	// declares a keyword matching this exactly, per the contract's
	// exact-case-insensitive Match rule (docs/plugin-contract.md).
	itemLabel = "external-demo-proof"
)

// matchVocabulary is the field-name vocabulary this plugin declares in its
// Describe response and reads from MatchRequest.match_fields — mirrors
// plugins/mock/plugin.go's single-field "labels" shape exactly.
var matchVocabulary = []string{"labels"}

// itemLabels is the fixed Labels slice every item below carries.
var itemLabels = []string{itemLabel}

// SourcePlugin implements sdk.SourcePlugin. Unlike plugins/mock (a fixed,
// in-memory item set with no configuration at all), this plugin's item set
// is DERIVED from what the kernel actually launched it with — its own
// per-instance extras map, captured once at construction from the decoded
// WEBSPACES_SOURCE_CONFIG envelope (main.go) — so every item this plugin
// returns is direct, observable evidence of what config and environment
// this process was actually handed, never an assertion this repo's own
// tests would otherwise have to take on faith.
type SourcePlugin struct {
	extras map[string]string
}

// NewSourcePlugin builds a SourcePlugin carrying this instance's own
// decoded extras map (may be nil/empty — a legitimate "no extras
// configured" state, contributing zero extras items to Match's output).
func NewSourcePlugin(extras map[string]string) *SourcePlugin {
	return &SourcePlugin{extras: extras}
}

// Describe is called once, immediately after the handshake, before any
// other RPC (contract: "RPC semantics: Describe"). It returns this
// plugin's identity plus its extras declaration (D-15, PLUG-09) —
// exercising every field of proto/topos/v1/plugin.proto's ExtrasField
// message: one required, non-secret key (workspace_id) and one optional,
// secret key (api_key), each with a label and a placeholder. No icon is
// declared — an omitted icon is a supported, documented state
// (docs/plugin-contract.md "Describe"), and this proof plugin has no
// identity mark of its own to ship.
func (p *SourcePlugin) Describe(_ context.Context, _ *toposv1.DescribeRequest) (*toposv1.DescribeResponse, error) {
	return &toposv1.DescribeResponse{
		SourceType:      sourceType,
		DisplayName:     displayName,
		ContractVersion: contractVersion,
		MatchVocabulary: matchVocabulary,
		Extras: []*toposv1.ExtrasField{
			{
				Key:         "workspace_id",
				Label:       "Workspace ID",
				Required:    true,
				Secret:      false,
				Placeholder: "acme-42",
			},
			{
				Key:         "api_key",
				Label:       "API Key",
				Required:    false,
				Secret:      true,
				Placeholder: "sk-...",
			},
		},
	}, nil
}

// Match is called only at sync time, never at request time (contract: "RPC
// semantics: Match"). It reads only this plugin's one declared field,
// "labels", exactly like plugins/mock's Match — see
// labelsMatchAnyKeyword, below, the identical exact-case-insensitive rule.
// When the configured keyword matches, it returns:
//
//   - one item per configured extras key (id "extras/<key>"), proving
//     arbitrary provider-specific keys reached this out-of-repo process
//     with no kernel code change (PLUG-09);
//   - one item per environment variable actually visible to THIS process
//     (id "env/<NAME>") — NAMES only, in both the item id and its title,
//     never values, so the synced corpus can never carry a secret even
//     when a test harness deliberately hands one to this process (D-14's
//     launch-environment allowlist is provable by what does NOT appear
//     here just as much as by what does);
//   - one fixed anchor item, so a match against this plugin's declared
//     label always returns something even when no extras are configured.
func (p *SourcePlugin) Match(_ context.Context, req *toposv1.MatchRequest) (*toposv1.MatchResponse, error) {
	keywords := req.GetMatchFields()["labels"].GetValues()
	if !labelsMatchAnyKeyword(itemLabels, keywords) {
		return &toposv1.MatchResponse{}, nil
	}

	var items []*toposv1.Item
	var ts int64 = 1704067200 // 2024-01-01T00:00:00Z, fixed — this proof plugin has no real event time

	extrasKeys := make([]string, 0, len(p.extras))
	for k := range p.extras {
		extrasKeys = append(extrasKeys, k)
	}
	sort.Strings(extrasKeys)
	for _, key := range extrasKeys {
		value := p.extras[key]
		id := "extras/" + key
		items = append(items, &toposv1.Item{
			SourceId:      id,
			SourceType:    sourceType,
			Title:         fmt.Sprintf("extras %s=%s", key, value),
			TimestampUnix: ts,
			Fidelity:      toposv1.LinkFidelity_LINK_FIDELITY_EXACT,
			DeepLink:      "http://localhost/external-demo/" + id,
			Labels:        itemLabels,
			Provenance:    provenanceFor(id),
		})
		ts++
	}

	envNames := make([]string, 0)
	for _, kv := range os.Environ() {
		name, _, ok := strings.Cut(kv, "=")
		if !ok || name == "" {
			continue
		}
		envNames = append(envNames, name)
	}
	sort.Strings(envNames)
	for _, name := range envNames {
		id := "env/" + name
		items = append(items, &toposv1.Item{
			SourceId:      id,
			SourceType:    sourceType,
			Title:         "env " + name, // NAME only, never the value (D-14)
			TimestampUnix: ts,
			Fidelity:      toposv1.LinkFidelity_LINK_FIDELITY_EXACT,
			DeepLink:      "http://localhost/external-demo/" + id,
			Labels:        itemLabels,
			Provenance:    provenanceFor(id),
		})
		ts++
	}

	items = append(items, &toposv1.Item{
		SourceId:      anchorSourceID,
		SourceType:    sourceType,
		Title:         "external-demo anchor",
		TimestampUnix: ts,
		Fidelity:      toposv1.LinkFidelity_LINK_FIDELITY_EXACT,
		DeepLink:      "http://localhost/external-demo/" + anchorSourceID,
		Labels:        itemLabels,
		Provenance:    provenanceFor(anchorSourceID),
	})

	return &toposv1.MatchResponse{Items: items}, nil
}

// labelsMatchAnyKeyword mirrors plugins/mock/plugin.go's identical helper:
// exact, case-insensitive comparison only — never substring or prefix
// (contract: "Match" rule 2).
func labelsMatchAnyKeyword(labels, keywords []string) bool {
	for _, label := range labels {
		for _, kw := range keywords {
			if strings.EqualFold(label, kw) {
				return true
			}
		}
	}
	return false
}

// provenanceFor builds the five plugin-populated provenance keys the
// contract documents (docs/plugin-contract.md "Provenance") — the sixth,
// synced_at_unix, is filled in by the kernel's index layer at read time
// and must never be set here.
func provenanceFor(sourceID string) map[string]string {
	return map[string]string{
		"source_type":      sourceType,
		"source_system":    sourceSystem,
		"source_id":        sourceID,
		"plugin":           pluginBinaryName,
		"contract_version": contractVersion,
	}
}

// Fetch is called only at request time, never from the sync/Match path
// (contract: "RPC semantics: Fetch"). It returns a small text rendition of
// the requested item's title — the exact string Match already built for
// that id, recomputed here from the same inputs (extras/environment)
// rather than cached, so a stale Fetch after a config change never serves
// yesterday's value. codes.NotFound for an id this plugin does not
// recognise.
func (p *SourcePlugin) Fetch(_ context.Context, req *toposv1.FetchRequest) (*toposv1.FetchResponse, error) {
	sourceID := req.GetSourceId()

	switch {
	case sourceID == anchorSourceID:
		return &toposv1.FetchResponse{Available: true, Text: "external-demo anchor"}, nil

	case strings.HasPrefix(sourceID, "extras/"):
		key := strings.TrimPrefix(sourceID, "extras/")
		value, ok := p.extras[key]
		if !ok {
			return nil, status.Errorf(codes.NotFound, "external-demo: item %q not found", sourceID)
		}
		return &toposv1.FetchResponse{Available: true, Text: fmt.Sprintf("extras %s=%s", key, value)}, nil

	case strings.HasPrefix(sourceID, "env/"):
		name := strings.TrimPrefix(sourceID, "env/")
		if _, ok := os.LookupEnv(name); !ok {
			return nil, status.Errorf(codes.NotFound, "external-demo: item %q not found", sourceID)
		}
		return &toposv1.FetchResponse{Available: true, Text: "env " + name}, nil // NAME only, never the value

	default:
		return nil, status.Errorf(codes.NotFound, "external-demo: item %q not found", sourceID)
	}
}

// Health is a lightweight reachability probe (contract: "RPC semantics:
// Health"). This proof plugin has nothing external to be unreachable
// from, so it always reports reachable with no error — mirrors
// plugins/mock/plugin.go's own always-true shape.
func (p *SourcePlugin) Health(_ context.Context, _ *toposv1.HealthRequest) (*toposv1.HealthResponse, error) {
	return &toposv1.HealthResponse{Reachable: true}, nil
}
