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
	ID string // "{source}:{source_id}"
	// Source is the source INSTANCE id — the [sources.<id>] config map key
	// this item was synced through. This is the kernel's identity key
	// everywhere it matters (item ids, sync_runs rows, agent grants, HTTP
	// responses — D-08): it is config-key-trusted, set only from the
	// operator's own config map, never from anything a plugin process can
	// assert.
	Source string
	// SourceType is the plugin KIND learned from the plugin's own Describe
	// RPC response (T-01-07) — descriptive provenance only, never an
	// identity key after this split. Two instances of one plugin binary
	// share the same SourceType but always have distinct Source values.
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

// ID derives the kernel-wide stable item ID from a source instance id and a
// plugin-local source ID: "{source}:{source_id}".
func ID(source, sourceID string) string {
	return fmt.Sprintf("%s:%s", source, sourceID)
}

// FromProto converts a toposv1.Item (as returned by a plugin's Match
// RPC) into the kernel's normalized Item type. source is the instance id
// (config-key-trusted, D-08); sourceType is the Describe-learned plugin
// kind (T-01-07) — the two are never merged.
func FromProto(source, sourceType string, p *toposv1.Item) Item {
	prov := make(map[string]string, len(p.GetProvenance()))
	for k, v := range p.GetProvenance() {
		prov[k] = v
	}
	labels := make([]string, len(p.GetLabels()))
	copy(labels, p.GetLabels())

	return Item{
		ID:                     ID(source, p.GetSourceId()),
		Source:                 source,
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
