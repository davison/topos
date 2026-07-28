// Package correlate (kernel/correlate) implements sync-time correlation
// (KERN-02): for each configured webspace, it calls every plugin's Match
// RPC once and persists the resulting items into the local index. This is
// the ONLY package in the repository permitted to call a plugin's Match
// RPC — kernel/httpapi's stream handler reads exclusively from
// kernel/index. Its only non-test callers are cmd/webspaces/main.go
// (source-list construction) and kernel/syncer.Coordinator, which is the
// only caller of SyncSource — see kernel/syncer/coordinator.go's package
// doc for why.
package correlate

import (
	"context"
	"fmt"
	"strings"

	"github.com/davison/webspaces/kernel/config"
	"github.com/davison/webspaces/kernel/index"
	"github.com/davison/webspaces/kernel/item"
	webspacesv1 "github.com/davison/webspaces/sdk/gen/webspaces/v1"
)

// Source is the minimal plugin surface correlate depends on. Deliberately
// decoupled from kernel/pluginhost (which implements it structurally via
// *pluginhost.Plugin) so this package can be unit tested with fakes
// without launching real plugin subprocesses.
type Source interface {
	Name() string
	SourceType() string
	Match(ctx context.Context, keywords []string) (*webspacesv1.MatchResponse, error)
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
// identity is promoted from "webspace" to "(webspace, source_type)"
// (02-01-PLAN.md's objective): with two-or-more sources, "any source
// fails" and "the only source fails" are no longer the same event, so a
// result is now reported per source per webspace rather than once per
// webspace.
type WebspaceResult struct {
	Webspace   string
	SourceType string
	ItemCount  int
	Err        error
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
		resp, err := src.Match(ctx, ws.Keywords)
		if err != nil {
			wrapped := fmt.Errorf("match against source %q: %w", src.Name(), err)
			results = append(results, WebspaceResult{Webspace: name, SourceType: src.SourceType(), Err: wrapped})
			continue
		}

		var items []item.Item
		for _, protoItem := range resp.GetItems() {
			it := item.FromProto(src.SourceType(), protoItem)
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

		if err := e.Store.ReplaceWebspaceSourceItems(ctx, name, src.SourceType(), items); err != nil {
			wrapped := fmt.Errorf("persist webspace %q source %q: %w", name, src.SourceType(), err)
			results = append(results, WebspaceResult{Webspace: name, SourceType: src.SourceType(), Err: wrapped})
			continue
		}

		results = append(results, WebspaceResult{Webspace: name, SourceType: src.SourceType(), ItemCount: len(items)})
	}

	return results, strings.Join(rejected, "; ")
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
