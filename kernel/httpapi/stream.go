package httpapi

import (
	"net/http"
	"strconv"

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

		known, err := store.WebspaceExists(ctx, name)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		if !known {
			WriteError(w, http.StatusNotFound, "webspace_not_found", "webspace \""+name+"\" is not configured or has not been synced")
			return
		}

		items, err := store.StreamItems(ctx, name, cfg.Webspaces[name].Filter)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		resp := streamResponse{
			SchemaVersion: schemaVersion,
			Webspace:      name,
			Items:         make([]streamItem, len(items)),
		}

		if runs, err := store.LatestSyncRunPerSource(ctx); err == nil {
			resp.Sync = aggregateSyncStatus(runs)
		}

		for i, it := range items {
			resp.Items[i] = toStreamItemFor(it, cfg.DisplayNameFor)
		}

		WriteJSON(w, http.StatusOK, resp)
	}
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
		Link:                   link{URL: it.DeepLink, Fidelity: string(it.Fidelity)},
		ThumbnailURL:           thumb,
		Provenance:             prov,
	}
}
