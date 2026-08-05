// Package item holds the kernel's normalized Item type, mirroring
// topos.v1.Item, and the conversion helpers between the two.
package item

import (
	"fmt"

	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

// Fidelity mirrors toposv1.LinkFidelity as a lowercase-hyphenated
// string, matching the kernel HTTP JSON contract.
type Fidelity string

const (
	FidelityUnspecified      Fidelity = ""
	FidelityExact            Fidelity = "exact"
	FidelityAnchored         Fidelity = "anchored"
	FidelityConversationOnly Fidelity = "conversation-only"
)

// FidelityFromProto converts a toposv1.LinkFidelity to the kernel's
// string representation.
func FidelityFromProto(f toposv1.LinkFidelity) Fidelity {
	switch f {
	case toposv1.LinkFidelity_LINK_FIDELITY_EXACT:
		return FidelityExact
	case toposv1.LinkFidelity_LINK_FIDELITY_ANCHORED:
		return FidelityAnchored
	case toposv1.LinkFidelity_LINK_FIDELITY_CONVERSATION_ONLY:
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
	// SyncedAtUnix is the index's own record of when this row was last
	// written (the items.synced_at column). It is never populated by a
	// plugin's MatchResponse — FromProto leaves it zero — and is instead
	// filled in by the index layer (kernel/index/store.go) when an item is
	// read back, so the kernel HTTP layer can publish it as the
	// synced_at_unix provenance key (AGENT-02) without trusting a plugin
	// to report its own sync time.
	SyncedAtUnix int64
}

// ID derives the kernel-wide stable item ID from a source type and a
// plugin-local source ID: "{source_type}:{source_id}".
func ID(sourceType, sourceID string) string {
	return fmt.Sprintf("%s:%s", sourceType, sourceID)
}

// FromProto converts a toposv1.Item (as returned by a plugin's Match
// RPC) into the kernel's normalized Item type.
func FromProto(sourceType string, p *toposv1.Item) Item {
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
