// Package correlate (kernel/correlate) implements sync-time correlation
// (KERN-02): for each configured webspace, it calls every plugin's Match
// RPC once and persists the resulting items into the local index. This is
// the ONLY package in the repository permitted to call a plugin's Match
// RPC — kernel/httpapi's stream handler reads exclusively from
// kernel/index. Its only non-test callers are cmd/topos/main.go
// (source-list construction) and kernel/syncer.Coordinator, which is the
// only caller of SyncSource — see kernel/syncer/coordinator.go's package
// doc for why.
package correlate

import (
	"context"
	"fmt"
	"strings"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/index"
	"github.com/davison/topos/kernel/item"
	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

// Source is the minimal plugin surface correlate depends on. Deliberately
// decoupled from kernel/pluginhost (which implements it structurally via
// *pluginhost.Plugin) so this package can be unit tested with fakes
// without launching real plugin subprocesses.
type Source interface {
	Name() string
	SourceType() string
	// MatchVocabulary returns the field-name vocabulary this instance's
	// plugin declared in its Describe response — the set of keys
	// matchFieldsFor may populate when resolving a webspace's match input
	// for this instance.
	MatchVocabulary() []string
	Match(ctx context.Context, fields map[string][]string) (*toposv1.MatchResponse, error)
}

// Engine runs sync cycles against a set of sources and a config. Sources
// is retained for callers that build an Engine alongside a
// kernel/syncer.Coordinator (the coordinator takes its own source list
// separately) but is no longer read by anything in this package — the
// coordinator is what iterates sources now (see SyncSource's doc comment).
type Engine struct {
	Store   *index.Store
	Sources []Source
	Config  *config.Config
}

// WebspaceResult summarizes one (webspace, source) sync outcome. The sync
// identity is promoted from "webspace" to "(webspace, source)"
// (02-01-PLAN.md's objective, generalized to source INSTANCE identity by
// D-08): with two-or-more sources, "any source fails" and "the only source
// fails" are no longer the same event, so a result is now reported per
// source per webspace rather than once per webspace. Source is the
// instance id ([sources.<id>] config key) — two instances of one plugin
// type report entirely independent results.
type WebspaceResult struct {
	Webspace  string
	Source    string
	ItemCount int
	Err       error
}

// SyncSource runs one source's Match RPC against every configured
// webspace's keyword list — once per webspace — and persists that source's
// contribution to each webspace independently via
// Store.ReplaceWebspaceSourceItems, regardless of whether any other
// configured source succeeds or fails in the same sync cycle. A Match
// error for this source is recorded on that source's result for every
// webspace and skips only this source's persistence for that webspace;
// it never discards, delays, or rolls back another source's already-
// persisted rows (the partial-source-failure bug fixed by this method —
// see 02-RESEARCH.md "Critical Architecture Finding").
//
// SyncSource never writes to sync_runs itself (02-02-PLAN.md Task 1): it
// returns its per-webspace results plus rejections, the aggregated
// "plugin %q source_id %q: %v" message for every item skipped at the
// correlation boundary across every webspace this call touched, joined
// with "; ". The caller — kernel/syncer.Coordinator, the only caller of
// this method now that SyncAll is gone — owns the two-phase
// StartSyncRun/FinishSyncRun write around this call and decides the
// run's overall status from these return values, so the coordinator
// remains the single source of truth for sync history (D-06).
func (e *Engine) SyncSource(ctx context.Context, src Source) (results []WebspaceResult, rejections string) {
	results = make([]WebspaceResult, 0, len(e.Config.Webspaces))
	var rejected []string

	for name, ws := range e.Config.Webspaces {
		fields, participates := matchFieldsFor(ws, src)
		if !participates {
			// D-03: this instance is excluded from this webspace by a
			// non-empty sources allowlist. Never call Match — instead
			// clear this instance's previously persisted rows for this
			// webspace so a de-allowlisted instance leaves no orphaned
			// rows behind (ROADMAP success criterion 3).
			if err := e.Store.ReplaceWebspaceSourceItems(ctx, name, src.Name(), nil); err != nil {
				wrapped := fmt.Errorf("clear webspace %q source %q: %w", name, src.Name(), err)
				results = append(results, WebspaceResult{Webspace: name, Source: src.Name(), Err: wrapped})
				continue
			}
			results = append(results, WebspaceResult{Webspace: name, Source: src.Name(), ItemCount: 0})
			continue
		}

		resp, err := src.Match(ctx, fields)
		if err != nil {
			wrapped := fmt.Errorf("match against source %q: %w", src.Name(), err)
			results = append(results, WebspaceResult{Webspace: name, Source: src.Name(), Err: wrapped})
			continue
		}

		var items []item.Item
		for _, protoItem := range resp.GetItems() {
			it := item.FromProto(src.Name(), src.SourceType(), protoItem)
			// PLUG-03: an item with an unspecified fidelity or an empty
			// deep link must never reach the index. Skip just this item
			// (not the whole sync) and name the plugin and source id so
			// the sync run records it.
			if rejErr := validateCorrelatedItem(it); rejErr != nil {
				rejected = append(rejected,
					fmt.Sprintf("plugin %q source_id %q: %v", src.Name(), it.SourceID, rejErr))
				continue
			}
			items = append(items, it)
		}

		if err := e.Store.ReplaceWebspaceSourceItems(ctx, name, src.Name(), items); err != nil {
			wrapped := fmt.Errorf("persist webspace %q source %q: %w", name, src.Name(), err)
			results = append(results, WebspaceResult{Webspace: name, Source: src.Name(), Err: wrapped})
			continue
		}

		results = append(results, WebspaceResult{Webspace: name, Source: src.Name(), ItemCount: len(items)})
	}

	return results, strings.Join(rejected, "; ")
}

// matchFieldsFor resolves one source instance's Match input for one
// webspace, implementing the full D-01/D-02/D-03 resolution in order:
//
//  1. Allowlist (D-03): if ws.Sources is non-empty and does not name src,
//     src does not participate in ws at all — the second return value is
//     false and the caller must not call Match for this (webspace, source)
//     pair.
//  2. Explicit block (D-02): if ws.Match names src, that block is returned
//     verbatim — it replaces the Keywords fallback outright for this
//     instance; the two are never combined.
//  3. Fallback (D-01): otherwise, ws.Keywords is fanned into every field of
//     src's declared vocabulary (src.MatchVocabulary()) — a webspace
//     declaring only `keywords` therefore reproduces the pre-Phase-5
//     shared-keyword-list behaviour byte for byte.
//
// Each call returns a map scoped to exactly this one instance's own
// resolved fields — never the webspace's whole match configuration — so a
// Match RPC to one plugin process never discloses another instance's match
// configuration (T-05-07).
func matchFieldsFor(ws config.Webspace, src Source) (fields map[string][]string, participates bool) {
	if !ws.Participates(src.Name()) {
		return nil, false
	}

	if block, ok := ws.Match[src.Name()]; ok {
		return map[string][]string(block), true
	}

	fields = make(map[string][]string, len(src.MatchVocabulary()))
	for _, field := range src.MatchVocabulary() {
		fields[field] = ws.Keywords
	}
	return fields, true
}

// validateCorrelatedItem enforces PLUG-03 at the sync boundary: no item
// with an unspecified fidelity or an empty deep link may reach the index.
func validateCorrelatedItem(it item.Item) error {
	if it.Fidelity == item.FidelityUnspecified {
		return fmt.Errorf("unspecified link fidelity")
	}
	if strings.TrimSpace(it.DeepLink) == "" {
		return fmt.Errorf("empty deep link")
	}
	return nil
}
