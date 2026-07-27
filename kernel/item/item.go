// Package item holds the kernel's normalized Item type, mirroring
// webspaces.v1.Item, and the conversion helpers between the two.
package item

import (
	"fmt"

	webspacesv1 "github.com/davison/webspaces/sdk/gen/webspaces/v1"
)

// Fidelity mirrors webspacesv1.LinkFidelity as a lowercase-hyphenated
// string, matching the kernel HTTP JSON contract.
type Fidelity string

const (
	FidelityUnspecified      Fidelity = ""
	FidelityExact            Fidelity = "exact"
	FidelityAnchored         Fidelity = "anchored"
	FidelityConversationOnly Fidelity = "conversation-only"
)

// FidelityFromProto converts a webspacesv1.LinkFidelity to the kernel's
// string representation.
func FidelityFromProto(f webspacesv1.LinkFidelity) Fidelity {
	switch f {
	case webspacesv1.LinkFidelity_LINK_FIDELITY_EXACT:
		return FidelityExact
	case webspacesv1.LinkFidelity_LINK_FIDELITY_ANCHORED:
		return FidelityAnchored
	case webspacesv1.LinkFidelity_LINK_FIDELITY_CONVERSATION_ONLY:
		return FidelityConversationOnly
	default:
		return FidelityUnspecified
	}
}

// Item is the kernel's normalized representation of a single indexed item,
// sourced from a plugin's MatchResponse and persisted into the local index.
type Item struct {
	ID                     string // "{source_type}:{source_id}"
	SourceType             string
	SourceID               string
	Title                  string
	Preview                string
	TimestampUnix          int64
	SecondaryTimestampUnix int64
	Fidelity               Fidelity
	DeepLink               string
	Labels                 []string
	Provenance             map[string]string
	GroupID                string
	GroupLabel             string
	HasThumbnail           bool
}

// ID derives the kernel-wide stable item ID from a source type and a
// plugin-local source ID: "{source_type}:{source_id}".
func ID(sourceType, sourceID string) string {
	return fmt.Sprintf("%s:%s", sourceType, sourceID)
}

// FromProto converts a webspacesv1.Item (as returned by a plugin's Match
// RPC) into the kernel's normalized Item type.
func FromProto(sourceType string, p *webspacesv1.Item) Item {
	prov := make(map[string]string, len(p.GetProvenance()))
	for k, v := range p.GetProvenance() {
		prov[k] = v
	}
	labels := make([]string, len(p.GetLabels()))
	copy(labels, p.GetLabels())

	return Item{
		ID:                     ID(sourceType, p.GetSourceId()),
		SourceType:             sourceType,
		SourceID:               p.GetSourceId(),
		Title:                  p.GetTitle(),
		Preview:                p.GetPreview(),
		TimestampUnix:          p.GetTimestampUnix(),
		SecondaryTimestampUnix: p.GetSecondaryTimestampUnix(),
		Fidelity:               FidelityFromProto(p.GetFidelity()),
		DeepLink:               p.GetDeepLink(),
		Labels:                 labels,
		Provenance:             prov,
		GroupID:                p.GetGroupId(),
		GroupLabel:             p.GetGroupLabel(),
		HasThumbnail:           p.GetHasThumbnail(),
	}
}
