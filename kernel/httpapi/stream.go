package httpapi

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/davison/webspaces/kernel/index"
	"github.com/davison/webspaces/kernel/item"
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
	ID                     string            `json:"id"`
	SourceType             string            `json:"source_type"`
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
// function taking only the index store, so this handler is structurally
// unable to reach a plugin (KERN-02 / Pitfall 1). It never calls Match; it
// only reads the already-correlated index.
func StreamHandler(store *index.Store) http.HandlerFunc {
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

		items, err := store.StreamItems(ctx, name)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		resp := streamResponse{
			SchemaVersion: schemaVersion,
			Webspace:      name,
			Items:         make([]streamItem, len(items)),
		}

		if run, ok, err := store.LatestSyncRun(ctx); err == nil && ok {
			resp.Sync = syncStatus{Status: run.Status, FinishedUnix: run.FinishedUnix, Error: run.Error}
		}

		for i, it := range items {
			resp.Items[i] = toStreamItem(it)
		}

		WriteJSON(w, http.StatusOK, resp)
	}
}

func toStreamItem(it item.Item) streamItem {
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
		SourceType:             it.SourceType,
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
