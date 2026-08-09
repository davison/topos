// Package main implements topos-plugin-mockstrict's SourcePlugin: a
// fixed, in-memory three-item corpus reproducing plugins/mock's
// Describe/Match/Fetch/Health skeleton with two deliberate differences
// (07.1-02-PLAN.md D-05/D-06):
//
//  1. SourcePlugin carries a `path string` field (mock's struct is
//     empty) — the required-field mechanism this plugin exists to
//     exercise. The path is never opened; only main.go's pre-Serve
//     guard cares whether it's empty.
//  2. matchVocabulary is `["tags"]`, not mock's own single-value
//     vocabulary — a deliberately different field name so a rendered
//     match form makes the two plugin types visually distinguishable
//     (07.1-05's UAT item 10 race test has no other observable signal,
//     since EditSourceModal's dialog titles are fixed strings that never
//     name the instance).
//
// This file is NOT written as third-party-facing documentation the way
// plugins/mock/plugin.go is (PLUG-05's deliverable) — it exists purely
// as harness fixture infrastructure.
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
	sourceType      = "mockstrict"
	displayName     = "Mockstrict Test Source"
	contractVersion = "topos.v2"

	// sourceSystem stands in for a real base URL / connection string —
	// this plugin has no real instance, but Provenance's "source_system"
	// key is documented as required on every item, so this is a fixed
	// placeholder rather than an omitted key (mirrors plugins/mock's own
	// sourceSystem constant).
	sourceSystem = "mockstrict://in-memory"
)

// matchVocabulary is deliberately "tags", NOT the field name
// plugins/mock declares: the two fixture plugins must never declare the
// same vocabulary field name, so the SPA's match step renders a distinct
// input ("Tags" here, mock's own field name there) — the one observable
// signal 07.1-05's UAT item 10 race spec has to prove which plugin
// type's describe-response won a race, since neither type's dialog
// chrome names the instance.
var matchVocabulary = []string{"tags"}

// strictItems is this plugin's fixed, in-memory item set — three items
// with source ids s1/s2/s3 and titles unmistakably distinct from mock's
// four, so a spec asserting on one source can never accidentally satisfy
// itself with the other's items. Labels are drawn from "strict" and
// "fixture" (the two values Match resolves the "tags" vocabulary field
// against).
//
// DeepLink values use "http://localhost/..." literals, never a
// fictitious external hostname — internal/audit/outbound_hosts_test.go
// mechanically fails the build on any other absolute-URL host (mirrors
// plugins/mock/plugin.go's identical constraint and its comment on why).
var strictItems = []*toposv1.Item{
	{
		SourceId:      "s1",
		SourceType:    sourceType,
		Title:         "Mockstrict fixture: alpha record",
		Preview:       "First item in the mockstrict fixture corpus — labelled strict only.",
		TimestampUnix: 1706745600, // 2024-02-01T00:00:00Z
		Fidelity:      toposv1.LinkFidelity_LINK_FIDELITY_EXACT,
		DeepLink:      "http://localhost/mockstrict/items/s1",
		Labels:        []string{"strict"},
		Provenance:    provenanceFor("s1"),
	},
	{
		SourceId:      "s2",
		SourceType:    sourceType,
		Title:         "Mockstrict fixture: beta record",
		Preview:       "Second item in the mockstrict fixture corpus — labelled fixture only.",
		TimestampUnix: 1706832000, // 2024-02-02T00:00:00Z
		Fidelity:      toposv1.LinkFidelity_LINK_FIDELITY_EXACT,
		DeepLink:      "http://localhost/mockstrict/items/s2",
		Labels:        []string{"fixture"},
		Provenance:    provenanceFor("s2"),
	},
	{
		SourceId:      "s3",
		SourceType:    sourceType,
		Title:         "Mockstrict fixture: gamma record",
		Preview:       "Third item in the mockstrict fixture corpus — labelled both strict and fixture.",
		TimestampUnix: 1706918400, // 2024-02-03T00:00:00Z
		Fidelity:      toposv1.LinkFidelity_LINK_FIDELITY_EXACT,
		DeepLink:      "http://localhost/mockstrict/items/s3",
		Labels:        []string{"strict", "fixture"},
		Provenance:    provenanceFor("s3"),
	},
}

// strictFullText holds the CONTENT_VARIANT_FULL "text" body for each
// item, keyed by source_id — kept separate from strictItems for the same
// reason plugins/mock/plugin.go's mockFullText is: Item.preview is a
// bounded snippet the index stores, while Fetch's extracted text is the
// live, full-length content, fetched fresh on every call and never
// persisted (KERN-03).
var strictFullText = map[string]string{
	"s1": "Full extracted text for mockstrict fixture item s1 (alpha record).",
	"s2": "Full extracted text for mockstrict fixture item s2 (beta record).",
	"s3": "Full extracted text for mockstrict fixture item s3 (gamma record).",
}

// noRenditionReason is returned as FetchResponse.unavailable_reason for
// every variant this plugin serves — mirrors plugins/mock's identical
// constant and its documented "available=false with a populated reason
// is a normal outcome, not an error" contract.
const noRenditionReason = "the mockstrict source plugin has no rendition to offer for any item"

// provenanceFor builds the five plugin-populated provenance keys the
// contract documents (docs/plugin-contract.md "Provenance"). The sixth
// key, synced_at_unix, is filled in by the kernel's index layer at read
// time and must never be set here.
func provenanceFor(sourceID string) map[string]string {
	return map[string]string{
		"source_type":      sourceType,
		"source_system":    sourceSystem,
		"source_id":        sourceID,
		"plugin":           "topos-plugin-mockstrict",
		"contract_version": contractVersion,
	}
}

// SourcePlugin implements sdk.SourcePlugin over the fixed strictItems
// corpus. Unlike plugins/mock's empty struct, it carries a path field —
// the value main.go's pre-Serve fatal guard requires be non-empty. The
// value itself is never read by any RPC below; only its presence at
// launch time matters (this plugin reads a fixed in-memory corpus, never
// a real filesystem path).
type SourcePlugin struct {
	path string
}

// NewSourcePlugin builds a SourcePlugin. Unlike plugins/mock's
// NewSourcePlugin() (no arguments — the mock has no connection details
// to configure), this constructor takes the required path string main.go
// resolved from WEBSPACES_SOURCE_CONFIG.
func NewSourcePlugin(path string) *SourcePlugin {
	return &SourcePlugin{path: path}
}

// Describe is called once, immediately after the handshake, before any
// other RPC. Returns match_vocabulary ["tags"] — deliberately different
// from plugins/mock's own vocabulary (see the matchVocabulary var
// comment).
func (p *SourcePlugin) Describe(_ context.Context, _ *toposv1.DescribeRequest) (*toposv1.DescribeResponse, error) {
	return &toposv1.DescribeResponse{
		SourceType:      sourceType,
		DisplayName:     displayName,
		ContractVersion: contractVersion,
		MatchVocabulary: matchVocabulary,
	}, nil
}

// Match reads only the "tags" key from MatchRequest.match_fields (its
// one declared vocabulary field) and ignores any other key present in
// the map. Returns every item whose native categorization (its fixed
// Labels slice) matches ANY of the supplied values, exactly and
// case-insensitively — never substring/prefix. An empty or absent "tags"
// value list matches nothing.
func (p *SourcePlugin) Match(_ context.Context, req *toposv1.MatchRequest) (*toposv1.MatchResponse, error) {
	keywords := req.GetMatchFields()["tags"].GetValues()
	var items []*toposv1.Item
	for _, it := range strictItems {
		if labelsMatchAnyKeyword(it.GetLabels(), keywords) {
			items = append(items, it)
		}
	}
	return &toposv1.MatchResponse{Items: items}, nil
}

// labelsMatchAnyKeyword reports whether any of labels exactly,
// case-insensitively equals any of keywords — mirrors plugins/mock's
// identical helper.
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

// Fetch is called only at request time, never from the sync/Match path.
// This plugin never has a byte rendition to offer for any variant — a
// normal "available: false" outcome, not an error. An unknown source id
// maps to codes.NotFound; CONTENT_VARIANT_UNSPECIFIED maps to
// codes.InvalidArgument — mirrors plugins/mock's identical shape.
func (p *SourcePlugin) Fetch(_ context.Context, req *toposv1.FetchRequest) (*toposv1.FetchResponse, error) {
	sourceID := req.GetSourceId()
	it := findStrictItem(sourceID)
	if it == nil {
		return nil, status.Errorf(codes.NotFound, "mockstrict: item %q not found", sourceID)
	}

	switch req.GetVariant() {
	case toposv1.ContentVariant_CONTENT_VARIANT_FULL:
		return &toposv1.FetchResponse{
			Available:         false,
			UnavailableReason: noRenditionReason,
			Text:              strictFullText[sourceID],
			Provenance:        it.GetProvenance(),
		}, nil
	case toposv1.ContentVariant_CONTENT_VARIANT_PREVIEW, toposv1.ContentVariant_CONTENT_VARIANT_THUMBNAIL:
		return &toposv1.FetchResponse{
			Available:         false,
			UnavailableReason: noRenditionReason,
		}, nil
	default:
		// CONTENT_VARIANT_UNSPECIFIED is the zero value and is never a
		// valid request.
		return nil, status.Error(codes.InvalidArgument, "mockstrict: unspecified content variant")
	}
}

// findStrictItem returns the fixed item with the given source_id, or nil.
func findStrictItem(sourceID string) *toposv1.Item {
	for _, it := range strictItems {
		if it.GetSourceId() == sourceID {
			return it
		}
	}
	return nil
}

// Health is a lightweight reachability probe. This plugin has nothing to
// be unreachable from, so it always reports reachable with no error.
func (p *SourcePlugin) Health(_ context.Context, _ *toposv1.HealthRequest) (*toposv1.HealthResponse, error) {
	return &toposv1.HealthResponse{
		Reachable:    true,
		LastSyncUnix: time.Now().Unix(),
	}, nil
}
