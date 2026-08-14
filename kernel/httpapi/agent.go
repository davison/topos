// agent.go implements the /agent/v1 route namespace (AGENT-01, D-12): a
// default-deny, per-source-grant-filtered mirror of the human-facing
// /api/* routes, for an automated caller with no authentication of its
// own (T-02-22 — the grant model is an authorization layer on top of the
// same unauthenticated loopback boundary, never an authentication
// mechanism).
//
// Deviation from 02-PATTERNS.md (recorded in 02-04-SUMMARY.md): the
// pattern map sketched this as its own kernel/httpapi/agent subpackage.
// It lives in package httpapi instead — a subpackage would need
// WriteJSON, WriteError, toStreamItem, syncStatus, streamItem,
// itemDetailResponse, itemContent, rendition, allowedRenditionTypes,
// writeFetchError and the Fetcher interface from its parent, while the
// parent package (routes.go) mounts it, which is an import cycle no
// subpackage split can avoid without duplicating all of the above.
//
// Every handler here follows the same shape: resolve the request-scoped
// granted set via grantedSources, then do the identical index
// read/plugin call the /api/* sibling handler does, filtering by that
// granted set as the one added step (T-02-19). Every not-found branch for
// an ungranted item reuses the exact "item_not_found" code/status/message
// construction kernel/httpapi/item.go uses for a genuinely nonexistent id
// — never a distinct code — so the namespace cannot be used to enumerate
// which sources exist but are withheld (T-02-20).
package httpapi

import (
	"context"
	"io"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/index"
	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

// grantedSources returns the set of source INSTANCE ids whose config
// source has agent.read = true (D-08, D-10) — directly from
// cfg.AgentReadGrantedNames(), with no name-to-source_type indirection.
// Index rows already carry the instance id (item.Item.Source), so a grant
// is checked by matching that instance id against this set, never by
// translating through the plugin kind first — two configured instances of
// one plugin type therefore never share a grant (T-05-01).
func grantedSources(cfg *config.Config) map[string]bool {
	return cfg.AgentReadGrantedNames()
}

// filterRunsByGrant restricts a LatestSyncRunPerSource-shaped map (keyed
// by source INSTANCE id) to the granted set, so aggregateSyncStatus
// computes its error/running/ok precedence over granted sources only
// (per-handler requirement: the webspace and stream listings' sync status
// must never be influenced by an ungranted source's failure or success).
func filterRunsByGrant(runs map[string]index.SyncRun, granted map[string]bool) map[string]index.SyncRun {
	out := make(map[string]index.SyncRun, len(runs))
	for source, run := range runs {
		if granted[source] {
			out[source] = run
		}
	}
	return out
}

// agentCapabilities publishes both grants on a GET /agent/v1/sources
// entry. Handoff is metadata only — no route in this phase consumes it
// (T-02-21); actual agent-initiated actions are AGENT-11, v1.x.
type agentCapabilities struct {
	Read    bool `json:"read"`
	Handoff bool `json:"handoff"`
}

// agentSourceStatus is one entry of GET /agent/v1/sources: the same
// sourceStatus shape /api/sources reports, plus the requesting agent's own
// capabilities for that source.
type agentSourceStatus struct {
	sourceStatus
	Capabilities agentCapabilities `json:"capabilities"`
}

type agentSourcesResponse struct {
	SchemaVersion int                 `json:"schema_version"`
	Sources       []agentSourceStatus `json:"sources"`
}

// agentSourcesHandler serves GET /agent/v1/sources: the same per-source
// health/sync merge sourceStatusesFrom builds for /api/sources, filtered
// to the granted set and wrapped with each source's own capabilities. A
// zero-grant config returns 200 with an empty array, never an error
// (AGENT-01/empty).
func agentSourcesHandler(store *index.Store, cfgStore *config.Store, prober HealthProber) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := cfgStore.Expanded()
		ctx := r.Context()
		granted := grantedSources(cfg)

		statuses, err := sourceStatusesFrom(ctx, store, prober)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		out := make([]agentSourceStatus, 0, len(statuses))
		for _, s := range statuses {
			if !granted[s.Name] {
				continue
			}
			out = append(out, agentSourceStatus{
				sourceStatus: s,
				Capabilities: agentCapabilities{
					Read:    true,
					Handoff: cfg.Sources[s.Name].Agent.Handoff,
				},
			})
		}

		WriteJSON(w, http.StatusOK, agentSourcesResponse{SchemaVersion: schemaVersion, Sources: out})
	}
}

// agentGrantedItemCount reads webspaceName's full item set and counts only
// the items whose source INSTANCE is granted — GET /agent/v1/webspaces has
// no dedicated per-source-filtered index query, so this reuses StreamItems
// (the same read StreamHandler/agentStreamHandler use) rather than adding
// a new Store method outside this plan's scope.
func agentGrantedItemCount(ctx context.Context, store *index.Store, webspaceName string, granted map[string]bool, filterTerms []string) (int, error) {
	// index.ViewIncluded, explicit: the agent mirror has no excluded view
	// (13-02-PLAN.md Task 1) — an agent grant can never surface the
	// excluded bucket.
	items, err := store.StreamItems(ctx, webspaceName, filterTerms, index.ViewIncluded)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, it := range items {
		if granted[it.Source] {
			count++
		}
	}
	return count, nil
}

// agentWebspacesHandler serves GET /agent/v1/webspaces: the identical
// webspacesResponse shape /api/webspaces reports, with each webspace's
// item_count and last_sync computed over the granted source set only
// (structural filtering, not a cosmetic count adjustment — a webspace
// whose only items belong to an ungranted source reports item_count 0,
// exactly as if that source had never synced anything).
//
// last_sync is computed PER WEBSPACE (08-UAT.md G-08-3, mirroring
// WebspacesHandler's own move), composing filterRunsByParticipation with
// the existing filterRunsByGrant rather than replacing it — grant
// filtering stays outermost in meaning: an ungranted source must remain
// invisible through this namespace regardless of whether it participates
// in the webspace being asked about.
func agentWebspacesHandler(store *index.Store, cfgStore *config.Store, prober HealthProber) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := cfgStore.Expanded()
		ctx := r.Context()
		granted := grantedSources(cfg)

		runs, err := store.LatestSyncRunPerSource(ctx)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		names := make([]string, 0, len(cfg.Webspaces))
		for name := range cfg.Webspaces {
			names = append(names, name)
		}
		sort.Strings(names)

		resp := webspacesResponse{SchemaVersion: schemaVersion, Webspaces: make([]webspaceSummary, 0, len(names))}
		for _, name := range names {
			count, err := agentGrantedItemCount(ctx, store, name, granted, cfg.Webspaces[name].Filter)
			if err != nil {
				WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
				return
			}
			resp.Webspaces = append(resp.Webspaces, webspaceSummary{
				Name:      name,
				Keywords:  keywordsOrEmpty(cfg.Webspaces[name].Keywords),
				ItemCount: count,
				LastSync:  aggregateSyncStatus(filterRunsByGrant(filterRunsByParticipation(runs, cfg, name), granted)),
			})
		}

		WriteJSON(w, http.StatusOK, resp)
	}
}

// agentStreamHandler serves GET /agent/v1/webspaces/{webspace}/stream: the
// identical streamResponse shape /api/webspaces/{webspace}/stream reports,
// with items filtered to the granted source set AND to the webspace's
// saved permanent filter (D-16: the filtered view IS the webspace for
// every consumer, human and agent alike), and sync status aggregated over
// the webspace's participating sources (08-UAT.md G-08-3), further
// restricted to the granted set — the same composition
// agentWebspacesHandler applies. An unknown webspace still returns 404
// webspace_not_found (the webspace's existence is not a grant question);
// existence itself is now answered by webspaceIsKnown (07-15-PLAN.md) — a
// name in the running config OR with surviving index rows — so a
// config-known webspace is servable through this mirror too, before its
// first sync. A known webspace with zero granted items returns 200 with an
// empty items array (AGENT-01/empty), never 404. Filtering preserves StreamItems'
// total chronological order — ungranted rows are dropped, the remaining
// rows are never reordered (AGENT-01/ordering). cfg is read fresh from
// cfgStore as the first statement of the returned closure — the identical
// live-read treatment StreamHandler gets, so a filter saved through
// PUT /api/config narrows this surface with no kernel restart either.
func agentStreamHandler(store *index.Store, cfgStore *config.Store, prober HealthProber) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := cfgStore.Expanded()
		name := chi.URLParam(r, "webspace")
		ctx := r.Context()

		known, err := webspaceIsKnown(ctx, store, cfg, name)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		if !known {
			writeWebspaceNotFound(w, name)
			return
		}

		granted := grantedSources(cfg)

		// index.ViewIncluded, explicit: the agent mirror has no excluded
		// view (13-02-PLAN.md Task 1) — an agent grant can never surface
		// the excluded bucket.
		items, err := store.StreamItems(ctx, name, cfg.Webspaces[name].Filter, index.ViewIncluded)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		resp := streamResponse{
			SchemaVersion: schemaVersion,
			Webspace:      name,
			Items:         []streamItem{},
		}
		for _, it := range items {
			if !granted[it.Source] {
				continue
			}
			resp.Items = append(resp.Items, toStreamItemFor(it, cfg.DisplayNameFor))
		}

		if runs, err := store.LatestSyncRunPerSource(ctx); err == nil {
			resp.Sync = aggregateSyncStatus(filterRunsByGrant(filterRunsByParticipation(runs, cfg, name), granted))
		}

		WriteJSON(w, http.StatusOK, resp)
	}
}

// agentItemNotFound writes the exact item_not_found envelope
// kernel/httpapi/item.go's ItemHandler writes for a genuinely nonexistent
// id — byte-for-byte the same code, status and message construction —
// used here for both "id not in the index" and "id exists but its source
// is ungranted" so the two are indistinguishable to the caller (T-02-20).
func agentItemNotFound(w http.ResponseWriter, id string) {
	WriteError(w, http.StatusNotFound, "item_not_found", "item \""+id+"\" was not found in the index")
}

// agentItemHandler serves GET /agent/v1/items/{id}: the identical
// itemDetailResponse shape /api/items/{id} reports for a granted item: an
// ungranted item is reported as not-found, never with a distinct code.
func agentItemHandler(store *index.Store, cfgStore *config.Store, prober HealthProber, fetcher Fetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := cfgStore.Expanded()
		id := itemIDParam(r)
		ctx := r.Context()

		it, ok, err := store.GetItem(ctx, id)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		granted := grantedSources(cfg)
		if !ok || !granted[it.Source] {
			agentItemNotFound(w, id)
			return
		}

		result, err := fetcher.Fetch(ctx, it.Source, it.SourceID, toposv1.ContentVariant_CONTENT_VARIANT_FULL)
		if err != nil {
			writeFetchError(w, id, err)
			return
		}

		content := itemContent{
			Available:         result.Available,
			UnavailableReason: result.UnavailableReason,
			Text:              result.Text,
		}
		if result.Available && result.MimeType != "" {
			content.Rendition = &rendition{
				MimeType:  result.MimeType,
				SizeBytes: result.SizeBytes,
				URL:       "/agent/v1/items/" + id + "/content",
			}
		}

		WriteJSON(w, http.StatusOK, itemDetailResponse{
			SchemaVersion: schemaVersion,
			Item:          toStreamItemFor(it, cfg.DisplayNameFor),
			Content:       content,
		})
	}
}

// agentRenditionHandler is the agent-namespace sibling of
// kernel/httpapi/item.go's renditionHandler — kept as its own function
// rather than an exported/shared helper because item.go is not a file
// this plan modifies (02-04-PLAN.md's files_modified list). Every
// behavior below (allowlist check, hardened header set) is copied
// verbatim from renditionHandler; the only added step is the grant check
// before any plugin call is made.
func agentRenditionHandler(store *index.Store, cfgStore *config.Store, prober HealthProber, fetcher Fetcher, variant toposv1.ContentVariant) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := cfgStore.Expanded()
		id := itemIDParam(r)
		ctx := r.Context()

		it, ok, err := store.GetItem(ctx, id)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		granted := grantedSources(cfg)
		if !ok || !granted[it.Source] {
			agentItemNotFound(w, id)
			return
		}

		result, err := fetcher.Fetch(ctx, it.Source, it.SourceID, variant)
		if err != nil {
			writeFetchError(w, id, err)
			return
		}

		if !result.Available || result.Body == nil {
			WriteError(w, http.StatusNotFound, "content_unavailable", "no rendition is available for item \""+id+"\"")
			return
		}
		defer result.Body.Close()

		if !allowedRenditionTypes[result.MimeType] {
			WriteError(w, http.StatusUnsupportedMediaType, "unsupported_rendition_type",
				"rendition MIME type \""+result.MimeType+"\" is not on the allowlist")
			return
		}

		// D-11: identical branch to renditionHandler's sibling in item.go
		// — see that function's doc comment for why sanitizing/wrapping
		// happens here, once, kernel-side.
		//
		// UI-09 / 06-RESEARCH.md Open Question 2: the agent surface has no
		// search UI (AGENT-10/11 are v1.x-deferred), so this call site
		// passes nil terms deliberately — every /agent/v1 rendition stays
		// byte-identical to its pre-UI-09 output. This is a scope
		// boundary, not an oversight: sanitizeAndWrapRendition's shared
		// signature means the compiler forces this call site to be
		// updated, but the choice of "always unhighlighted" is the
		// explicit decision recorded here.
		var body []byte
		if result.MimeType == "text/html" {
			fragment, err := io.ReadAll(result.Body)
			if err != nil {
				WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
				return
			}
			wrapped, err := sanitizeAndWrapRendition(result.ContentShape, fragment, nil)
			if err != nil {
				WriteError(w, http.StatusBadGateway, "unsupported_content_shape",
					"item \""+id+"\": "+err.Error())
				return
			}
			body = wrapped
		}

		h := w.Header()
		h.Set("Content-Type", result.MimeType)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Content-Disposition", "inline")
		h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; object-src 'none'; sandbox")
		h.Set("Cache-Control", "private, no-store")
		w.WriteHeader(http.StatusOK)
		if result.MimeType == "text/html" {
			_, _ = w.Write(body)
			return
		}
		_, _ = io.Copy(w, result.Body)
	}
}

// MountAgentRoutes mounts the /agent/v1 namespace on r using the same
// store/cfgStore/Fetcher/HealthProber the human-facing /api/* routes use
// (Router, in routes.go) — there is no second store, no second sync
// pipeline, only a grant-filtered read path over the same data. Every
// handler registered below — agentSourcesHandler, agentWebspacesHandler,
// agentStreamHandler, agentItemHandler, agentRenditionHandler — resolves
// the running config from cfgStore as the first statement of its own
// request closure, matching StreamHandler/WebspacesHandler/ItemHandler/
// SourceRefreshHandler (07-02-PLAN.md Task 2's fix, extended to this
// namespace here). This function itself holds no config value at all: a
// handler registered here later has nothing stale in scope to be
// accidentally handed.
//
// The agent namespace in particular cannot tolerate a router-construction-
// time resolution the way a merely-cosmetic surface might: a revoked
// agent.read grant that stays in force until the process restarts is a
// live authorization-bypass window on AGENT-01's default-deny model, and
// D-06 promises the operator that a config save applies immediately — a
// promise this namespace broke until 07-REVIEW.md's CR-01 was closed here.
func MountAgentRoutes(r chi.Router, store *index.Store, cfgStore *config.Store, fetcher Fetcher, prober HealthProber) {
	r.Get("/agent/v1/sources", agentSourcesHandler(store, cfgStore, prober))
	r.Get("/agent/v1/webspaces", agentWebspacesHandler(store, cfgStore, prober))
	r.Get("/agent/v1/webspaces/{webspace}/stream", agentStreamHandler(store, cfgStore, prober))
	r.Get("/agent/v1/items/{id}", agentItemHandler(store, cfgStore, prober, fetcher))
	r.Get("/agent/v1/items/{id}/content", agentRenditionHandler(store, cfgStore, prober, fetcher, toposv1.ContentVariant_CONTENT_VARIANT_PREVIEW))
	r.Get("/agent/v1/items/{id}/thumbnail", agentRenditionHandler(store, cfgStore, prober, fetcher, toposv1.ContentVariant_CONTENT_VARIANT_THUMBNAIL))
}
