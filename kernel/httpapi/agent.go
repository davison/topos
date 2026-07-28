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
// granted set via grantedSourceTypes, then do the identical index
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

	"github.com/davison/webspaces/kernel/config"
	"github.com/davison/webspaces/kernel/index"
	webspacesv1 "github.com/davison/webspaces/sdk/gen/webspaces/v1"
)

// grantedSourceTypes returns the set of Describe-learned source_types
// whose config source has agent.read = true, intersected with byName
// (every launched plugin's config name -> source_type mapping, from
// HealthProber.SourceTypesByName). A configured-but-unlaunched source has
// no entry in byName and is therefore never granted, even if its config
// carries agent.read = true — there is nothing to serve on its behalf.
func grantedSourceTypes(cfg *config.Config, byName map[string]string) map[string]bool {
	granted := map[string]bool{}
	for name := range cfg.AgentReadGrantedNames() {
		if st, ok := byName[name]; ok {
			granted[st] = true
		}
	}
	return granted
}

// filterRunsByGrant restricts a LatestSyncRunPerSource-shaped map (keyed
// by source_type) to the granted set, so aggregateSyncStatus computes its
// error/running/ok precedence over granted sources only (per-handler
// requirement: the webspace and stream listings' sync status must never
// be influenced by an ungranted source's failure or success).
func filterRunsByGrant(runs map[string]index.SyncRun, granted map[string]bool) map[string]index.SyncRun {
	out := make(map[string]index.SyncRun, len(runs))
	for st, run := range runs {
		if granted[st] {
			out[st] = run
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
func agentSourcesHandler(store *index.Store, cfg *config.Config, prober HealthProber) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		granted := grantedSourceTypes(cfg, prober.SourceTypesByName())

		statuses, err := sourceStatusesFrom(ctx, store, prober)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		out := make([]agentSourceStatus, 0, len(statuses))
		for _, s := range statuses {
			if !granted[s.SourceType] {
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
// the items whose source_type is granted — GET /agent/v1/webspaces has no
// dedicated per-source-filtered index query, so this reuses StreamItems
// (the same read StreamHandler/agentStreamHandler use) rather than adding
// a new Store method outside this plan's scope.
func agentGrantedItemCount(ctx context.Context, store *index.Store, webspaceName string, granted map[string]bool) (int, error) {
	items, err := store.StreamItems(ctx, webspaceName)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, it := range items {
		if granted[it.SourceType] {
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
func agentWebspacesHandler(store *index.Store, cfg *config.Config, prober HealthProber) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		granted := grantedSourceTypes(cfg, prober.SourceTypesByName())

		runs, err := store.LatestSyncRunPerSource(ctx)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		lastSync := aggregateSyncStatus(filterRunsByGrant(runs, granted))

		names := make([]string, 0, len(cfg.Webspaces))
		for name := range cfg.Webspaces {
			names = append(names, name)
		}
		sort.Strings(names)

		resp := webspacesResponse{SchemaVersion: schemaVersion, Webspaces: make([]webspaceSummary, 0, len(names))}
		for _, name := range names {
			count, err := agentGrantedItemCount(ctx, store, name, granted)
			if err != nil {
				WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
				return
			}
			resp.Webspaces = append(resp.Webspaces, webspaceSummary{
				Name:      name,
				Keywords:  cfg.Webspaces[name].Keywords,
				ItemCount: count,
				LastSync:  lastSync,
			})
		}

		WriteJSON(w, http.StatusOK, resp)
	}
}

// agentStreamHandler serves GET /agent/v1/webspaces/{webspace}/stream: the
// identical streamResponse shape /api/webspaces/{webspace}/stream reports,
// with items filtered to the granted source set and sync status
// aggregated over that same restricted set. An unknown webspace still
// returns 404 webspace_not_found (the webspace's existence is not a grant
// question); a known webspace with zero granted items returns 200 with an
// empty items array (AGENT-01/empty), never 404. Filtering preserves
// StreamItems' total chronological order — ungranted rows are dropped, the
// remaining rows are never reordered (AGENT-01/ordering).
func agentStreamHandler(store *index.Store, cfg *config.Config, prober HealthProber) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "webspace")
		ctx := r.Context()

		known, err := store.WebspaceExists(ctx, name)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		if !known {
			WriteError(w, http.StatusNotFound, "webspace_not_found", "webspace \""+name+"\" is not configured or has not been synced")
			return
		}

		granted := grantedSourceTypes(cfg, prober.SourceTypesByName())

		items, err := store.StreamItems(ctx, name)
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
			if !granted[it.SourceType] {
				continue
			}
			resp.Items = append(resp.Items, toStreamItem(it))
		}

		if runs, err := store.LatestSyncRunPerSource(ctx); err == nil {
			resp.Sync = aggregateSyncStatus(filterRunsByGrant(runs, granted))
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
func agentItemHandler(store *index.Store, cfg *config.Config, prober HealthProber, fetcher Fetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := itemIDParam(r)
		ctx := r.Context()

		it, ok, err := store.GetItem(ctx, id)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		granted := grantedSourceTypes(cfg, prober.SourceTypesByName())
		if !ok || !granted[it.SourceType] {
			agentItemNotFound(w, id)
			return
		}

		result, err := fetcher.Fetch(ctx, it.SourceType, it.SourceID, webspacesv1.ContentVariant_CONTENT_VARIANT_FULL)
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
			Item:          toStreamItem(it),
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
func agentRenditionHandler(store *index.Store, cfg *config.Config, prober HealthProber, fetcher Fetcher, variant webspacesv1.ContentVariant) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := itemIDParam(r)
		ctx := r.Context()

		it, ok, err := store.GetItem(ctx, id)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		granted := grantedSourceTypes(cfg, prober.SourceTypesByName())
		if !ok || !granted[it.SourceType] {
			agentItemNotFound(w, id)
			return
		}

		result, err := fetcher.Fetch(ctx, it.SourceType, it.SourceID, variant)
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

		h := w.Header()
		h.Set("Content-Type", result.MimeType)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Content-Disposition", "inline")
		h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; object-src 'none'; sandbox")
		h.Set("Cache-Control", "private, no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, result.Body)
	}
}

// MountAgentRoutes mounts the /agent/v1 namespace on r using the same
// store/config/Fetcher/HealthProber the human-facing /api/* routes use
// (Router, in routes.go) — there is no second store, no second sync
// pipeline, only a grant-filtered read path over the same data.
func MountAgentRoutes(r chi.Router, store *index.Store, cfg *config.Config, fetcher Fetcher, prober HealthProber) {
	r.Get("/agent/v1/sources", agentSourcesHandler(store, cfg, prober))
	r.Get("/agent/v1/webspaces", agentWebspacesHandler(store, cfg, prober))
	r.Get("/agent/v1/webspaces/{webspace}/stream", agentStreamHandler(store, cfg, prober))
	r.Get("/agent/v1/items/{id}", agentItemHandler(store, cfg, prober, fetcher))
	r.Get("/agent/v1/items/{id}/content", agentRenditionHandler(store, cfg, prober, fetcher, webspacesv1.ContentVariant_CONTENT_VARIANT_PREVIEW))
	r.Get("/agent/v1/items/{id}/thumbnail", agentRenditionHandler(store, cfg, prober, fetcher, webspacesv1.ContentVariant_CONTENT_VARIANT_THUMBNAIL))
}
