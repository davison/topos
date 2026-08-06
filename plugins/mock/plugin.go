// Package main implements the reference "mock" source plugin (PLUG-05):
// a SourcePlugin built entirely from the published contract
// (docs/plugin-contract.md, proto/topos/v1/plugin.proto, and the sdk
// module) with no network dependency and no real source system behind
// it. This file is deliberately written to be read as documentation —
// each RPC method's comment states what the contract requires of it, not
// merely what this particular implementation happens to do — because
// this file is half of PLUG-05's deliverable: a stranger should be able
// to read it end to end, alongside the contract document, and understand
// everything a real plugin author needs to know.
package main

import (
	"context"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

const (
	sourceType      = "mock"
	displayName     = "Mock Source"
	contractVersion = "topos.v2"

	// sourceSystem stands in for a real base URL / connection string — the
	// mock has no real instance, but Provenance's "source_system" key is
	// documented as required on every item, so this is a fixed
	// placeholder rather than an omitted key.
	sourceSystem = "mock://in-memory"
)

// matchVocabulary is the field-name vocabulary this plugin declares in its
// Describe response and reads from MatchRequest.match_fields. "labels" is
// the mock's own native categorization — each fixed item's Labels slice —
// exactly the role a paperless-ngx tag name or an IMAP folder name plays
// for a real plugin (see the worked example in docs/plugin-contract.md).
var matchVocabulary = []string{"labels"}

// mockItems is the plugin's fixed, in-memory item set — no network call,
// no filesystem read, no configuration. Four items with deliberately
// varied shapes so this file exercises more of the Item message than a
// single trivial row would:
//
//   - "1" has no group (a standalone document, like paperless-ngx).
//   - "2" and "3" share group "team-standup" (a chat/thread-shaped
//     source), demonstrating group_id/group_label and, for "3",
//     LINK_FIDELITY_CONVERSATION_ONLY — a deep link that opens the
//     surrounding thread, not the exact message, which is the normal
//     shape for a chat source with no per-message deep-link scheme.
//   - "4" has has_thumbnail = true with LINK_FIDELITY_ANCHORED, to show
//     a case where the deep link opens the right context (a folder view)
//     without landing exactly on the object.
//
// Labels are each item's native categorization — what a real plugin's
// Match implementation would resolve keywords against (see Match, below).
//
// DeepLink values below deliberately use "http://localhost/..." rather
// than a fictitious external hostname: this repository's own build
// mechanically fails on any non-test, non-loopback absolute URL literal
// in shipped Go source (a no-foreign-egress guarantee — see
// internal/audit's egress scan, outside the four inputs this plugin was
// built from). A real plugin never hits this because its deep_link is
// built at runtime from the operator's own configured base_url — e.g.
// fmt.Sprintf("%s/documents/%s", p.baseURL, sourceID) — never a
// hardcoded literal. The mock has no real per-instance base_url to build
// from, so its deep links are fixed strings, and loopback is the only
// literal host such a scan structurally accepts.
var mockItems = []*toposv1.Item{
	{
		SourceId:      "1",
		SourceType:    sourceType,
		Title:         "Welcome to the mock source",
		Preview:       "This item exists purely to demonstrate a standalone document with no group/thread concept.",
		TimestampUnix: 1704067200, // 2024-01-01T00:00:00Z
		Fidelity:      toposv1.LinkFidelity_LINK_FIDELITY_EXACT,
		DeepLink:      "http://localhost/mock/items/1",
		Labels:        []string{"demo"},
		Provenance:    provenanceFor("1"),
	},
	{
		SourceId:               "2",
		SourceType:             sourceType,
		Title:                  "Standup: Tuesday",
		Preview:                "First message in a fixed mock thread — demonstrates group_id/group_label (a chat/conversation-shaped source).",
		TimestampUnix:          1704153600, // 2024-01-02T00:00:00Z
		SecondaryTimestampUnix: 1704153610,
		Fidelity:               toposv1.LinkFidelity_LINK_FIDELITY_EXACT,
		DeepLink:               "http://localhost/mock/threads/team-standup#msg-2",
		Labels:                 []string{"demo", "meeting"},
		GroupId:                "team-standup",
		GroupLabel:             "Team Standup",
		Provenance:             provenanceFor("2"),
	},
	{
		SourceId:               "3",
		SourceType:             sourceType,
		Title:                  "Standup: Wednesday",
		Preview:                "Second message in the same fixed mock thread as item 2 — demonstrates LINK_FIDELITY_CONVERSATION_ONLY, the normal fidelity for a chat source with no per-message deep-link scheme.",
		TimestampUnix:          1704240000, // 2024-01-03T00:00:00Z
		SecondaryTimestampUnix: 1704240010,
		Fidelity:               toposv1.LinkFidelity_LINK_FIDELITY_CONVERSATION_ONLY,
		DeepLink:               "http://localhost/mock/threads/team-standup",
		Labels:                 []string{"demo", "meeting"},
		GroupId:                "team-standup",
		GroupLabel:             "Team Standup",
		Provenance:             provenanceFor("3"),
	},
	{
		SourceId:      "4",
		SourceType:    sourceType,
		Title:         "Shopping list",
		Preview:       "Demonstrates LINK_FIDELITY_ANCHORED (the deep link opens the right context — a folder view — but not necessarily scrolled to this exact object) and has_thumbnail.",
		TimestampUnix: 1704326400, // 2024-01-04T00:00:00Z
		Fidelity:      toposv1.LinkFidelity_LINK_FIDELITY_ANCHORED,
		DeepLink:      "http://localhost/mock/folders/lists",
		Labels:        []string{"demo", "errands"},
		HasThumbnail:  true,
		Provenance:    provenanceFor("4"),
	},
}

// mockFullText holds the CONTENT_VARIANT_FULL "text" body for each item,
// keyed by source_id — kept separate from mockItems because Item.preview
// is a bounded snippet the index stores (per the hybrid data model) while
// Fetch's extracted text is the live, full-length content, fetched fresh
// on every call and never persisted (KERN-03).
var mockFullText = map[string]string{
	"1": "This is the mock source plugin's full extracted text for item 1. A real plugin's Fetch implementation would call out to its source system here (an HTTP client, a local database read, ...); the mock has nothing to call and simply returns this fixed string.",
	"2": "Full text for item 2: the first standup message in the mock thread.",
	"3": "Full text for item 3: the second standup message in the mock thread.",
	"4": "Full text for item 4: a shopping list, mocked.",
}

// noRenditionReason is returned as FetchResponse.unavailable_reason for
// every variant this plugin serves — the mock never has a byte rendition
// to offer (no PDF, no image, no rendered HTML), which is itself a
// normal, documented outcome (see the contract's Fetch section: available
// = false with a populated reason is expected, not an error).
const noRenditionReason = "the mock source plugin has no rendition to offer for any item"

// provenanceFor builds the five plugin-populated provenance keys the
// contract documents (docs/plugin-contract.md "Provenance"). The sixth
// key, synced_at_unix, is filled in by the kernel's index layer at read
// time and must never be set here — a plugin doesn't know when the
// kernel will next read its own persisted row.
func provenanceFor(sourceID string) map[string]string {
	return map[string]string{
		"source_type":      sourceType,
		"source_system":    sourceSystem,
		"source_id":        sourceID,
		"plugin":           "topos-plugin-mock",
		"contract_version": contractVersion,
	}
}

// SourcePlugin implements sdk.SourcePlugin with a fixed, in-memory item
// set. No fields: unlike a real plugin (which holds an HTTP client, a
// base URL, a database handle, ...), the mock has no per-instance state
// at all — every launched copy behaves identically.
type SourcePlugin struct{}

// NewSourcePlugin builds a SourcePlugin. Unlike every real plugin's
// constructor (plugins/paperless.NewSourcePlugin,
// plugins/silverbullet.NewSourcePlugin), this one takes no arguments —
// the mock has no connection details to configure.
func NewSourcePlugin() *SourcePlugin {
	return &SourcePlugin{}
}

// Describe is called once, immediately after the handshake, before any
// other RPC (contract: "RPC semantics: Describe"). It returns the
// plugin's identity: source_type is the kernel's only trusted source of
// this plugin's identity (never the config key or the binary filename),
// display_name is for UI/logs, and contract_version is the
// additive-compatibility signal.
func (p *SourcePlugin) Describe(_ context.Context, _ *toposv1.DescribeRequest) (*toposv1.DescribeResponse, error) {
	return &toposv1.DescribeResponse{
		SourceType:      sourceType,
		DisplayName:     displayName,
		ContractVersion: contractVersion,
		MatchVocabulary: matchVocabulary,
	}, nil
}

// Match is called only at sync time, never at request time (contract:
// "RPC semantics: Match"). The kernel resolves one webspace's match input
// down to this instance's own declared vocabulary and sends it as
// MatchRequest.match_fields; this mock reads only the "labels" key (its one
// declared field) and ignores any other key present in the map. The plugin
// returns every item whose native categorization (its fixed Labels slice)
// matches ANY of the supplied values, exactly and case-insensitively —
// never a substring or prefix match. An empty or absent "labels" value list
// matches nothing (see the worked example in docs/plugin-contract.md).
func (p *SourcePlugin) Match(_ context.Context, req *toposv1.MatchRequest) (*toposv1.MatchResponse, error) {
	keywords := req.GetMatchFields()["labels"].GetValues()
	var items []*toposv1.Item
	for _, it := range mockItems {
		if labelsMatchAnyKeyword(it.GetLabels(), keywords) {
			items = append(items, it)
		}
	}
	return &toposv1.MatchResponse{Items: items}, nil
}

// labelsMatchAnyKeyword reports whether any of labels exactly,
// case-insensitively equals any of keywords. "house" matches a label
// literally "House" but never "Household" — no substring/prefix matching,
// per the contract's Match section.
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

// Fetch is called only at request time — when a user or agent opens a
// specific item — never from the sync/Match path (contract: "RPC
// semantics: Fetch"). It is a single unary RPC: the full response is
// returned in one message, not a stream (decision D-Task1, option-a).
// This mock never has a byte rendition to offer for any variant, which is
// itself a normal "available: false" outcome (see noRenditionReason,
// above) — not an error. A source id that doesn't exist in mockItems maps
// to a gRPC codes.NotFound error, exactly as the contract requires for a
// source object that no longer exists.
func (p *SourcePlugin) Fetch(_ context.Context, req *toposv1.FetchRequest) (*toposv1.FetchResponse, error) {
	sourceID := req.GetSourceId()
	it := findMockItem(sourceID)
	if it == nil {
		return nil, status.Errorf(codes.NotFound, "mock: item %q not found", sourceID)
	}

	switch req.GetVariant() {
	case toposv1.ContentVariant_CONTENT_VARIANT_FULL:
		return &toposv1.FetchResponse{
			Available:         false,
			UnavailableReason: noRenditionReason,
			Text:              mockFullText[sourceID],
			Provenance:        it.GetProvenance(),
		}, nil
	case toposv1.ContentVariant_CONTENT_VARIANT_PREVIEW, toposv1.ContentVariant_CONTENT_VARIANT_THUMBNAIL:
		// No text on these variants — matches the contract's "PREVIEW:
		// just the inline-preview rendition, no text" / "THUMBNAIL: just
		// a small thumbnail rendition, no text".
		return &toposv1.FetchResponse{
			Available:         false,
			UnavailableReason: noRenditionReason,
		}, nil
	default:
		// CONTENT_VARIANT_UNSPECIFIED is the zero value and is never a
		// valid request (contract: "ContentVariant").
		return nil, status.Error(codes.InvalidArgument, "mock: unspecified content variant")
	}
}

// findMockItem returns the fixed item with the given source_id, or nil.
func findMockItem(sourceID string) *toposv1.Item {
	for _, it := range mockItems {
		if it.GetSourceId() == sourceID {
			return it
		}
	}
	return nil
}

// Health is a lightweight reachability probe (contract: "RPC semantics:
// Health"). The mock has nothing to be unreachable from, so it always
// reports reachable with no error.
func (p *SourcePlugin) Health(_ context.Context, _ *toposv1.HealthRequest) (*toposv1.HealthResponse, error) {
	return &toposv1.HealthResponse{
		Reachable:    true,
		LastSyncUnix: time.Now().Unix(),
	}, nil
}
