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
	"sort"
	"strconv"
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
	// Notice is a non-fatal, human-readable advisory about this (webspace,
	// source) pair — never an error and never a substitute for one. Set
	// only when this webspace's explicit ws.Match block (never the
	// keywords fallback) matched zero items across an otherwise-successful
	// Match call (the G-12-1/G-12-3 gap closure): a value that can never
	// match any label used to load, sync "ok", and produce zero rows with
	// no diagnostic anywhere. Composed exclusively from configuration by
	// zeroMatchNotice — never from anything the plugin returned — so a
	// plugin can no more fabricate a notice than it can fabricate its own
	// sync history (A-PLUG-04).
	Notice string
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
		fields, participates, explicit := matchFieldsFor(ws, src)
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

		// The zero-match notice (G-12-1/G-12-3): fires only when the
		// fields came from an EXPLICIT ws.Match block (never the keywords
		// fallback, which is fanned across every source and legitimately
		// matches nothing for most of them) and the plugin returned zero
		// items. Tested against resp.GetItems() — the count the plugin
		// actually returned — deliberately BEFORE the PLUG-03 rejection
		// loop below: emptiness caused by every returned item being
		// rejected at the correlation boundary must keep reporting as
		// rejections, never be reattributed to the operator's match
		// config. The non-empty fields guard stops a structurally
		// impossible empty explicit block from producing a contentless
		// advisory.
		var notice string
		if explicit && len(fields) > 0 && len(resp.GetItems()) == 0 {
			notice = zeroMatchNotice(name, fields)
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

		results = append(results, WebspaceResult{Webspace: name, Source: src.Name(), ItemCount: len(items), Notice: notice})
	}

	return results, strings.Join(rejected, "; ")
}

// ParticipatesIn is the single kernel-side definition of whether source
// instance participates in webspace ws — both the sync path
// (matchFieldsFor, below) and the config-apply path
// (kernel/supervisor.Supervisor's purgeDeparticipatedWebspaceRows,
// 07-16-PLAN.md) ask it, so the two can never disagree about which
// (webspace, source) pairs are live.
//
// The conjunction is exactly what matchFieldsFor already applied inline
// before this extraction:
//
//  1. Phase 5 D-03's allowlist gate (Webspace.Participates): a non-empty
//     ws.Sources not naming instance excludes it outright, regardless of
//     keywords or match input.
//  2. 07-11's D-20 has-match-input rule (see matchFieldsFor's own rule 3
//     for the safety reasoning): an instance with neither an explicit
//     ws.Match block nor a non-empty ws.Keywords fallback has no match
//     input at all and does not participate.
//
// ParticipatesIn deliberately does NOT resolve match fields — it reports
// only whether instance participates, never what it would be matched
// against — so a caller with no plugin handle to ask MatchVocabulary of
// (the supervisor, which has instance ids and a config but no launched
// plugin vocabulary) can still ask it. matchFieldsFor, which DOES have a
// plugin handle, resolves the actual field map once this predicate has
// already answered true.
//
// web/src/lib/participation.ts's participatingInstances/participatesIn is
// this function's client-side mirror — any change to the rule here must be
// reflected there in the same commit.
func ParticipatesIn(ws config.Webspace, instance string) bool {
	if !ws.Participates(instance) {
		return false
	}
	if _, ok := ws.Match[instance]; ok {
		return true
	}
	return len(ws.Keywords) > 0
}

// matchFieldsFor resolves one source instance's Match input for one
// webspace, implementing the full D-01/D-02/D-03/D-20 resolution in order:
//
//  1. Allowlist (D-03) and has-match-input (D-20): if src does not
//     participate in ws at all — ParticipatesIn's definition, covering
//     both the allowlist gate and the has-match-input rule (rule 3 below)
//     — the second return value is false and the caller must not call
//     Match for this (webspace, source) pair.
//  2. Explicit block (D-02): if ws.Match names src, that block is returned
//     verbatim — it replaces the Keywords fallback outright for this
//     instance; the two are never combined.
//  3. No match input at all (D-20, 07-11-PLAN.md): this is ParticipatesIn's
//     own has-match-input half — a SAFETY rule, not a tidiness one.
//     Fanning an empty Keywords slice across src.MatchVocabulary() would
//     hand the plugin a field map whose every value list is empty; a
//     plugin that reads that shape as "no constraint" would answer with
//     its entire corpus, writing the operator's whole mail or chat archive
//     into a webspace they created and left empty. This state was
//     unreachable before D-20 — config.Validate's validateFallbackCoverage
//     guaranteed every participating, block-less instance had a non-empty
//     Keywords fallback — and is reachable now that Webspace.IsEmptyShell
//     makes an empty webspace shell a valid config state.
//  4. Fallback (D-01): otherwise, ws.Keywords is fanned into every field of
//     src's declared vocabulary (src.MatchVocabulary()) — a webspace
//     declaring only `keywords` therefore reproduces the pre-Phase-5
//     shared-keyword-list behaviour byte for byte. Reaching this branch
//     implies a non-empty Keywords list, because ParticipatesIn already
//     established that fact as a condition of returning true when no
//     explicit block exists.
//
// Each call returns a map scoped to exactly this one instance's own
// resolved fields — never the webspace's whole match configuration — so a
// Match RPC to one plugin process never discloses another instance's match
// configuration (T-05-07).
//
// The third result, explicit, reports whether fields came from rule 2 (an
// explicit ws.Match block) rather than rule 4 (the keywords fallback) —
// added 12-09-PLAN.md so SyncSource can tell the two apart when deciding
// whether a zero-item Match answer deserves a zeroMatchNotice: the
// fallback is fanned across every source and legitimately matches nothing
// for most of them, so only the explicit-block branch may ever produce
// one.
// MatchFieldsFor is matchFieldsFor exported for the search fan-out
// (M2-R2, davison/topos#50): the kernel asks a source to search only with
// the same resolved membership input sync gives Match — no input, no call.
func MatchFieldsFor(ws config.Webspace, src Source) (fields map[string][]string, participates, explicit bool) {
	return matchFieldsFor(ws, src)
}

func matchFieldsFor(ws config.Webspace, src Source) (fields map[string][]string, participates, explicit bool) {
	if !ParticipatesIn(ws, src.Name()) {
		return nil, false, false
	}

	if block, ok := ws.Match[src.Name()]; ok {
		return map[string][]string(block), true, true
	}

	fields = make(map[string][]string, len(src.MatchVocabulary()))
	for _, field := range src.MatchVocabulary() {
		fields[field] = ws.Keywords
	}
	return fields, true, false
}

// zeroMatchNotice composes the G-12-1/G-12-3 zero-match advisory from its
// two arguments and nothing else — this is the A-PLUG-04 property in
// miniature, and it is why this function takes a field map rather than
// the whole webspace or the Match response: no MatchResponse value, item
// title, source_id, label or plugin-provided string can ever reach it.
// The text is read by a human (a source chip tooltip, docs/api.md's
// last_notice) and is never parsed by a client.
//
// Fields render in sorted name order and values in their configured
// order, so the returned string is byte-identical across repeated calls
// on the same input — a caller that joins several of these (see
// kernel/syncer.joinNotices) needs that determinism to avoid a
// perpetually-changing tooltip. The closing clause states the matching
// rule: when any value contains a glob metacharacter (the reported
// failure — .planning/debug/filesystem-items-missing-from-stream.md — was
// precisely a doublestar pattern typed into an exact-match field) the
// notice says so explicitly, so the operator learns the rule as well as
// the fact that it matched nothing.
func zeroMatchNotice(webspace string, fields map[string][]string) string {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)

	hasGlob := false
	parts := make([]string, 0, len(names))
	for _, name := range names {
		values := fields[name]
		quoted := make([]string, len(values))
		for i, v := range values {
			quoted[i] = strconv.Quote(v)
			if strings.ContainsAny(v, "*?[") {
				hasGlob = true
			}
		}
		parts = append(parts, name+"="+strings.Join(quoted, ", "))
	}

	rule := "match values are compared exactly against the values this source reports"
	if hasGlob {
		rule = "match values are compared exactly and never as glob patterns"
	}

	return fmt.Sprintf("webspace %q: match block matched 0 items (%s) — %s",
		webspace, strings.Join(parts, "; "), rule)
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
