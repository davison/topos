package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/index"
)

// marksRequest is the request body POST /api/webspaces/{webspace}/marks
// decodes. kind is a closed vocabulary (today, only "excluded" —
// index.MarkKindExcluded); action is "add" (exclude) or "remove"
// (include, the exact mirror).
type marksRequest struct {
	Kind    string   `json:"kind"`
	Action  string   `json:"action"`
	ItemIDs []string `json:"item_ids"`
}

// marksResponse mirrors every other envelope in this package: a fixed
// schema_version plus the write's own outcome. excluded_count is the
// webspace's LIVE total after this write — the same count the
// excluded-items view toggle renders (13-UI-SPEC.md E4) — read fresh via
// CountItemMarks rather than derived from changed, so a client never has
// to track the running total itself.
type marksResponse struct {
	SchemaVersion int    `json:"schema_version"`
	Webspace      string `json:"webspace"`
	Kind          string `json:"kind"`
	Action        string `json:"action"`
	Changed       int    `json:"changed"`
	ExcludedCount int    `json:"excluded_count"`
}

// MarksHandler serves POST /api/webspaces/{webspace}/marks (KERN-09/
// KERN-10) — the single write path for per-item exclude/include, shaped
// exactly like StreamHandler: cfg is resolved fresh as the first
// statement of the returned closure (D-06 — no kernel restart needed for
// a config change to be visible here), {webspace} is validated through
// the same webspaceIsKnown/writeWebspaceNotFound pair stream.go and
// search.go already use, so all three surfaces agree on what "this
// webspace exists" means. Every id ever reaches the store as a bound `?`
// parameter (SetItemMarks/ClearItemMarks) — never concatenated into SQL
// (T-13-01).
//
// This task's validation is intentionally the minimal set 13-01-PLAN.md
// Task 1's <behavior> names: an empty/absent item_ids is rejected; a
// bad kind/action, an over-cap id list, and blank-id trimming are Task
// 2's hardening pass (13-01-PLAN.md), landing in this same file without
// changing this route's path or response shape.
func MarksHandler(store *index.Store, cfgStore *config.Store) http.HandlerFunc {
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

		var req marksRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_request", "request body must be valid JSON")
			return
		}

		if req.Kind != index.MarkKindExcluded {
			WriteError(w, http.StatusBadRequest, "invalid_request", "kind must be \"excluded\"")
			return
		}
		if req.Action != "add" && req.Action != "remove" {
			WriteError(w, http.StatusBadRequest, "invalid_request", "action must be \"add\" or \"remove\"")
			return
		}
		if len(req.ItemIDs) == 0 {
			WriteError(w, http.StatusBadRequest, "invalid_request", "item_ids must not be empty")
			return
		}

		var changed int
		if req.Action == "add" {
			changed, err = store.SetItemMarks(ctx, name, req.Kind, req.ItemIDs)
		} else {
			changed, err = store.ClearItemMarks(ctx, name, req.Kind, req.ItemIDs)
		}
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		excludedCount, err := store.CountItemMarks(ctx, name, index.MarkKindExcluded)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		WriteJSON(w, http.StatusOK, marksResponse{
			SchemaVersion: schemaVersion,
			Webspace:      name,
			Kind:          req.Kind,
			Action:        req.Action,
			Changed:       changed,
			ExcludedCount: excludedCount,
		})
	}
}
