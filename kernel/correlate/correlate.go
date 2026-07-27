// Package correlate implements sync-time correlation (KERN-02): for each
// configured webspace, it calls every plugin's Match RPC once and persists
// the resulting items into the local index. This is the ONLY package in
// the repository permitted to call a plugin's Match RPC — kernel/httpapi's
// stream handler reads exclusively from kernel/index.
package correlate

import (
	"context"
	"fmt"
	"strings"
	"time"

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

// Engine runs sync cycles against a set of sources and a config.
type Engine struct {
	Store   *index.Store
	Sources []Source
	Config  *config.Config
	NowFunc func() time.Time // overridable for tests; defaults to time.Now
}

func (e *Engine) now() time.Time {
	if e.NowFunc != nil {
		return e.NowFunc()
	}
	return time.Now()
}

// WebspaceResult summarizes one webspace's sync outcome.
type WebspaceResult struct {
	Webspace  string
	ItemCount int
	Err       error
}

// SyncAll runs one full sync cycle: for every configured webspace, its full
// keyword list is passed to every source's Match RPC exactly once;
// keyword-list order never affects the persisted set or stream order (the
// plugin and the index both treat the keyword list as an unordered set of
// match criteria). Matched items are persisted through the index in a
// single transaction per webspace. One sync_runs row is recorded per
// source, summarizing the whole cycle.
func (e *Engine) SyncAll(ctx context.Context) ([]WebspaceResult, error) {
	started := e.now().Unix()

	sourceItemCounts := map[string]int{}
	sourceErrors := map[string]error{}
	rejectedItems := map[string][]string{}

	results := make([]WebspaceResult, 0, len(e.Config.Webspaces))

	for name, ws := range e.Config.Webspaces {
		var items []item.Item
		var firstErr error

		for _, src := range e.Sources {
			resp, err := src.Match(ctx, ws.Keywords)
			if err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("match against source %q: %w", src.Name(), err)
				}
				sourceErrors[src.SourceType()] = err
				continue
			}
			for _, protoItem := range resp.GetItems() {
				it := item.FromProto(src.SourceType(), protoItem)
				// PLUG-03: an item with an unspecified fidelity or an
				// empty deep link must never reach the index. Skip just
				// this item (not the whole sync) and name the plugin and
				// source id so the sync run records it.
				if rejErr := validateCorrelatedItem(it); rejErr != nil {
					rejectedItems[src.SourceType()] = append(rejectedItems[src.SourceType()],
						fmt.Sprintf("plugin %q source_id %q: %v", src.Name(), it.SourceID, rejErr))
					continue
				}
				items = append(items, it)
				sourceItemCounts[src.SourceType()]++
			}
		}

		if firstErr != nil {
			results = append(results, WebspaceResult{Webspace: name, Err: firstErr})
			continue
		}

		if err := e.Store.ReplaceWebspaceItems(ctx, name, items); err != nil {
			results = append(results, WebspaceResult{Webspace: name, Err: fmt.Errorf("persist webspace %q: %w", name, err)})
			continue
		}

		results = append(results, WebspaceResult{Webspace: name, ItemCount: len(items)})
	}

	finished := e.now().Unix()
	for _, src := range e.Sources {
		run := index.SyncRun{
			SourceType:   src.SourceType(),
			StartedUnix:  started,
			FinishedUnix: finished,
			Status:       "ok",
			ItemCount:    sourceItemCounts[src.SourceType()],
		}
		if err, failed := sourceErrors[src.SourceType()]; failed {
			run.Status = "error"
			run.Error = err.Error()
		} else if msgs := rejectedItems[src.SourceType()]; len(msgs) > 0 {
			// The sync itself succeeded (other items from this source
			// persisted normally) but these specific items were rejected
			// at the correlation boundary — recorded, not silently
			// dropped.
			run.Error = strings.Join(msgs, "; ")
		}
		if err := e.Store.RecordSyncRun(ctx, run); err != nil {
			return results, fmt.Errorf("correlate: record sync run for %s: %w", src.SourceType(), err)
		}
	}

	return results, nil
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
