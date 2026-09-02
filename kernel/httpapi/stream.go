package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/index"
	"github.com/davison/topos/kernel/item"
)

// syncStatus mirrors the shared "sync" object in the JSON contract.
type syncStatus struct {
	Status       string `json:"status"`
	FinishedUnix int64  `json:"finished_unix"`
	Error        string `json:"error"`
}

type link struct {
	URL      string `json:"url"`
	Fidelity string `json:"fidelity"`
}

type streamItem struct {
	ID string `json:"id"`
	// Source is the source INSTANCE id ([sources.<id>] config key) — the
	// identity key everywhere it matters (D-08). SourceType is retained
	// unchanged alongside it as the descriptive plugin kind (never an
	// identity key after this split).
	Source                 string            `json:"source"`
	SourceType             string            `json:"source_type"`
	SourceDisplayName      string            `json:"source_display_name"`
	SourceID               string            `json:"source_id"`
	Title                  string            `json:"title"`
	Preview                string            `json:"preview"`
	TimestampUnix          int64             `json:"timestamp_unix"`
	SecondaryTimestampUnix int64             `json:"secondary_timestamp_unix"`
	Labels                 []string          `json:"labels"`
	GroupID                string            `json:"group_id"`
	GroupLabel             string            `json:"group_label"`
	Link                   link              `json:"link"`
	ThumbnailURL           string            `json:"thumbnail_url,omitempty"`
	Provenance             map[string]string `json:"provenance"`
}

type streamResponse struct {
	SchemaVersion int          `json:"schema_version"`
	Webspace      string       `json:"webspace"`
	Sync          syncStatus   `json:"sync"`
	Items         []streamItem `json:"items"`
	// ExcludedCount is the webspace's LIVE total of items carrying an
	// excluded mark (13-02-PLAN.md Task 1, 13-UI-SPEC.md E4) — populated
	// on EVERY stream request, in both views, so the excluded-view toggle
	// never needs a second round trip to learn its own count.
	ExcludedCount int `json:"excluded_count"`
}

// StreamHandler serves GET /api/webspaces/{webspace}/stream — a free
// function taking only the index store and the live config store, so this
// handler is structurally unable to reach a plugin (KERN-02 / Pitfall 1):
// cfg is inert configuration data, resolved here only to label each item's
// source_display_name (D-09) and to read the webspace's saved permanent
// filter (D-16), never to reach a plugin process. It never calls Match; it
// only reads the already-correlated index. cfg is read fresh from
// cfgStore as the first statement of the returned closure (not captured
// once at Router construction, unlike WebspacesHandler/ItemHandler/
// SourceRefreshHandler) — a filter saved through PUT /api/config must be
// visible to the very next stream request with no kernel restart (D-06).
func StreamHandler(store *index.Store, cfgStore *config.Store) http.HandlerFunc {
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

		// view is parsed AFTER the webspaceIsKnown gate (13-02-PLAN.md
		// Task 1): an unknown webspace must still answer 404
		// webspace_not_found regardless of the view value, never a 400 for
		// a bad view masking the real "this webspace doesn't exist"
		// answer.
		view, ok := parseStreamView(r.URL.Query().Get("view"))
		if !ok {
			WriteError(w, http.StatusBadRequest, "invalid_request", "view must be \"included\" or \"excluded\"")
			return
		}

		dateFrom, dateTo, derr := effectiveDateRange(r, cfg.Webspaces[name])
		if derr != nil {
			WriteError(w, http.StatusBadRequest, "invalid_request", derr.Error())
			return
		}
		items, err := store.StreamItems(ctx, name, cfg.Webspaces[name].Filter, cfg.Webspaces[name].FilterBySource, dateFrom, dateTo, view)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		// ExcludedCount is read fresh on EVERY request, in both views
		// (13-02-PLAN.md Task 1) — the excluded-view toggle's own live
		// count (13-UI-SPEC.md E4), never derived from len(items) so a
		// client viewing the included bucket still learns the excluded
		// count without a second request.
		excludedCount, err := store.CountItemMarks(ctx, name, index.MarkKindExcluded)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		resp := streamResponse{
			SchemaVersion: schemaVersion,
			Webspace:      name,
			Items:         make([]streamItem, len(items)),
			ExcludedCount: excludedCount,
		}

		if runs, err := store.LatestSyncRunPerSource(ctx); err == nil {
			resp.Sync = aggregateSyncStatus(filterRunsByParticipation(runs, cfg, name))
		}

		for i, it := range items {
			resp.Items[i] = toStreamItemFor(it, cfg.DisplayNameFor)
		}

		WriteJSON(w, http.StatusOK, resp)
	}
}

// parseStreamView resolves the stream route's ?view= query value to an
// index.MarkView (13-02-PLAN.md Task 1): an absent or empty value is
// treated as "included" (unchanged pre-Phase-13 behavior — every existing
// caller that never sends ?view= at all sees byte-identical output), and
// only "included"/"excluded" are accepted otherwise. ok=false signals any
// other value — the caller rejects with 400 invalid_request naming both
// allowed values, never silently coercing an unrecognized value to either
// bucket.
func parseStreamView(raw string) (view index.MarkView, ok bool) {
	switch raw {
	case "":
		return index.ViewIncluded, true
	case string(index.ViewIncluded):
		return index.ViewIncluded, true
	case string(index.ViewExcluded):
		return index.ViewExcluded, true
	default:
		return "", false
	}
}

// webspaceIsKnown is the single existence gate every surface that answers
// "does this webspace exist" asks through (07-15-PLAN.md, closes
// 07-UAT.md G-07-1.missing[2]'s audit of search.go and agent.go): stream,
// search and the agent stream mirror all call this and nothing else.
//
// It is a deliberate disjunction of two halves, config first:
//
//   - The config half answers true the instant name is a key of cfg's
//     Webspaces map. The create flow's first `PUT /api/config` returns
//     200 before any sync has ever run (D-06/D-14/D-20) — a webspace is
//     therefore servable from the moment it is configured, with no
//     dependency on whether or when the eager resync `Supervisor.Apply`
//     dispatches as a detached goroutine has actually completed. This is
//     the half that closes G-07-1: the pre-fix gate asked the index alone
//     (sync history), which made the create-flow's immediate stream GET
//     land in a real, and on a zero-configured-sources install
//     PERMANENT, 404 window.
//   - The index half falls through to store.WebspaceExists only when the
//     config half answers false, so a webspace whose `[webspaces.*]`
//     block was removed from the file while its previously-synced index
//     rows survive still answers true — TestStreamHandler_
//     KnownEmptyWebspaceReturns200EmptyArray depends on exactly this
//     half and must keep passing.
//
// The gate is additive by construction: every request that answered true
// before this change still does (the index half is untouched), and the
// config half only ever turns a prior 404 into a 200, never the reverse.
func webspaceIsKnown(ctx context.Context, store *index.Store, cfg *config.Config, name string) (bool, error) {
	if _, ok := cfg.Webspaces[name]; ok {
		return true, nil
	}
	return store.WebspaceExists(ctx, name)
}

// writeWebspaceNotFound writes the one webspace_not_found 404 envelope,
// so its message literal exists once rather than duplicated across
// stream.go, search.go and agent.go. After webspaceIsKnown, a configured
// webspace is always servable regardless of sync history, so a 404 here
// means exactly one thing: name is not in the running configuration (and
// has no surviving index rows either) — the message no longer claims it
// might merely be unsynced.
func writeWebspaceNotFound(w http.ResponseWriter, name string) {
	WriteError(w, http.StatusNotFound, "webspace_not_found", "webspace \""+name+"\" is not configured")
}

// toStreamItem converts it into its streamItem representation with
// source_display_name resolved to the identity function (it.Source
// itself) — the same fallback D-09 already declares as an instance's
// default display name when no config override exists. This is the
// signature search.go's toSearchResult calls (out of this plan's file
// scope, per 05-01-PLAN.md's files_modified list), so a search result's
// source_display_name reports the instance id rather than any configured
// override; every other caller in this package uses toStreamItemFor
// directly, resolving against the real *config.Config.
func toStreamItem(it item.Item) streamItem {
	return toStreamItemFor(it, func(source string) string { return source })
}

// toStreamItemFor is toStreamItem's config-aware sibling: resolveDisplayName
// is applied to it.Source to populate source_display_name — callers pass
// cfg.DisplayNameFor (or an equivalent test fake) so the resolved name
// reflects the operator's own [sources.<id>] display_name (D-09).
func toStreamItemFor(it item.Item, resolveDisplayName func(string) string) streamItem {
	labels := it.Labels
	if labels == nil {
		labels = []string{}
	}
	// prov is a fresh copy, never the plugin-supplied map itself: the
	// kernel owns synced_at_unix (it.SyncedAtUnix, populated by the index
	// layer at read time — never by a plugin) and always sets it here,
	// overriding anything a plugin might have supplied under the same
	// key. Copying also keeps toStreamItem free of any shared-map
	// mutation surprises across repeated calls with the same item.Item.
	prov := make(map[string]string, len(it.Provenance)+1)
	for k, v := range it.Provenance {
		prov[k] = v
	}
	prov["synced_at_unix"] = strconv.FormatInt(it.SyncedAtUnix, 10)
	thumb := ""
	if it.HasThumbnail {
		thumb = "/api/items/" + it.ID + "/thumbnail"
	}
	return streamItem{
		ID:                     it.ID,
		Source:                 it.Source,
		SourceType:             it.SourceType,
		SourceDisplayName:      resolveDisplayName(it.Source),
		SourceID:               it.SourceID,
		Title:                  it.Title,
		Preview:                it.Preview,
		TimestampUnix:          it.TimestampUnix,
		SecondaryTimestampUnix: it.SecondaryTimestampUnix,
		Labels:                 labels,
		GroupID:                it.GroupID,
		GroupLabel:             it.GroupLabel,
		Link:                   link{URL: resolveStreamLinkURL(it), Fidelity: string(it.Fidelity)},
		ThumbnailURL:           thumb,
		Provenance:             prov,
	}
}

// resolveStreamLinkURL resolves the served link.url for it (12-01-PLAN.md
// Task 2, D-06): an item whose DeepLink carries the file:// scheme is
// served with the kernel's loopback open route for its own id
// (FilesystemOpenHandler, fsopen.go) instead of the raw file:// value —
// built with the same string-concatenation convention
// "/api/items/" + it.ID + "/content" already uses. Every other item is
// served with its deep link echoed unchanged. Keyed on the URL SCHEME
// alone, never it.SourceType — so a future third-party local-path plugin
// is covered for free, matching the contract's "no built-in table of known
// plugin types" discipline. This one helper covers the item-detail route
// too, since ItemHandler serializes through toStreamItemFor.
func resolveStreamLinkURL(it item.Item) string {
	if strings.HasPrefix(it.DeepLink, "file://") {
		return "/api/items/" + it.ID + "/open"
	}
	return it.DeepLink
}
