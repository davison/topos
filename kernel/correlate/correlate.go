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
// One sync_runs row is recorded for this source, summarizing the whole
// per-source cycle across every webspace.
func (e *Engine) SyncSource(ctx context.Context, src Source) []WebspaceResult {
	started := e.now().Unix()

	results := make([]WebspaceResult, 0, len(e.Config.Webspaces))

	totalItemCount := 0
	var rejections []string
	var runErr error

	for name, ws := range e.Config.Webspaces {
		resp, err := src.Match(ctx, ws.Keywords)
		if err != nil {
			wrapped := fmt.Errorf("match against source %q: %w", src.Name(), err)
			results = append(results, WebspaceResult{Webspace: name, SourceType: src.SourceType(), Err: wrapped})
			if runErr == nil {
				runErr = err
			}
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
				rejections = append(rejections,
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

		totalItemCount += len(items)
		results = append(results, WebspaceResult{Webspace: name, SourceType: src.SourceType(), ItemCount: len(items)})
	}

	finished := e.now().Unix()
	run := index.SyncRun{
		SourceType:   src.SourceType(),
		StartedUnix:  started,
		FinishedUnix: finished,
		Status:       "ok",
		ItemCount:    totalItemCount,
	}
	if runErr != nil {
		run.Status = "error"
		run.Error = runErr.Error()
	} else if len(rejections) > 0 {
		// The sync itself succeeded (other items from this source
		// persisted normally) but these specific items were rejected at
		// the correlation boundary — recorded, not silently dropped.
		run.Error = strings.Join(rejections, "; ")
	}
	if err := e.Store.RecordSyncRun(ctx, run); err != nil {
		// A sync_runs write failure is itself a result the caller needs
		// to see: append it as a webspace-less result carrying just the
		// source type and the error, rather than silently swallowing it
		// (which the previous SyncAll-only shape effectively did on a
		// full-cycle basis).
		results = append(results, WebspaceResult{SourceType: src.SourceType(), Err: fmt.Errorf("correlate: record sync run for %s: %w", src.SourceType(), err)})
	}

	return results
}

// SyncAll runs one full sync cycle: every configured source is synced via
// SyncSource in turn, and the results are concatenated. A returned error
// (or a per-webspace Err) from one source never short-circuits, delays, or
// discards another source's persistence — this is the source-major
// restructuring 02-01-PLAN.md's objective describes: the sync identity
// this loop operates on is "(webspace, source_type)", not "webspace".
func (e *Engine) SyncAll(ctx context.Context) ([]WebspaceResult, error) {
	results := make([]WebspaceResult, 0, len(e.Config.Webspaces)*len(e.Sources))
	for _, src := range e.Sources {
		results = append(results, e.SyncSource(ctx, src)...)
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
