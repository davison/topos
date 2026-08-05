// Package httpapi implements the kernel's loopback JSON HTTP API: /api/*,
// the human/UI-facing surface the embedded SPA consumes (AGENT-02), and
// /agent/v1/*, a default-deny, per-source-grant-filtered mirror of it for
// an automated caller (AGENT-01, D-12; see agent.go). Both share one JSON
// envelope and one schema_version counter — there is no second versioning
// scheme, only a narrower view over the same data for the agent surface.
package httpapi

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/index"
	"github.com/davison/topos/kernel/webui"
)

// schemaVersion is the envelope's schema_version field. Bump only for
// breaking JSON shape changes.
const schemaVersion = 1

// Router builds the chi router serving /api/webspaces,
// /api/webspaces/{webspace}/stream, /api/items/{id} (+ /content,
// /thumbnail), /api/sources, the manual-refresh routes, and the mirrored
// /agent/v1/* namespace (MountAgentRoutes, agent.go).
// StreamHandler deliberately takes only *index.Store — httpapi's
// sync-time read path cannot import kernel/pluginhost, so the stream
// route is structurally incapable of reaching a plugin (KERN-02 /
// Pitfall 1). The /api/items/* routes are the one deliberate exception:
// fetcher is the request-time, item-open plugin call path (KERN-03).
// prober and refresher are the other two: prober drives GET
// /api/sources's live reachability probe, and refresher is the same
// kernel/syncer.Coordinator the scheduler and the CLI use, so every
// caller of a sync reaches the identical single-flight entry point
// (D-06).
func Router(store *index.Store, cfg *config.Config, fetcher Fetcher, prober HealthProber, refresher Refresher) chi.Router {
	r := chi.NewRouter()
	r.Get("/api/webspaces", WebspacesHandler(store, cfg))
	r.Get("/api/webspaces/{webspace}/stream", StreamHandler(store))
	r.Get("/api/webspaces/{webspace}/search", SearchHandler(store))
	r.Get("/api/items/{id}", ItemHandler(store, fetcher))
	r.Get("/api/items/{id}/content", ItemContentHandler(store, fetcher))
	r.Get("/api/items/{id}/thumbnail", ItemThumbnailHandler(store, fetcher))
	r.Get("/api/sources", SourcesHandler(store, prober))
	r.Post("/api/sources/{name}/refresh", SourceRefreshHandler(cfg, refresher))
	r.Post("/api/sync", SyncRefreshHandler(refresher))
	// MountAgentRoutes adds the /agent/v1 namespace (AGENT-01, D-12): a
	// default-deny, grant-filtered mirror of the routes above, over the
	// same store/config/fetcher/prober. Every /api/* route above is
	// unaffected by any grant configuration — grants gate the agent
	// surface only.
	MountAgentRoutes(r, store, cfg, fetcher, prober)
	// NotFound only fires for requests that matched none of the routes
	// above, so this never shadows /api/*: any request under /api/ that
	// falls through here is genuinely unmatched, and every UI route
	// (/, /w/house-move, a browser reload on a deep link, ...) is served
	// by the embedded SPA and its 200.html fallback.
	r.NotFound(spaHandler(webui.FS()))
	return r
}

// spaHandler serves the embedded SvelteKit build, rewriting any path with
// no matching embedded file to /200.html — the adapter-static SPA fallback
// page, which bootstraps Svelte's own client-side router. Never named
// index.html: that collides with adapter-static's prerendered-output
// handling (01-RESEARCH.md Pitfall 3).
func spaHandler(assets fs.FS) http.HandlerFunc {
	fileServer := http.FileServer(http.FS(assets))
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := fs.Stat(assets, strings.TrimPrefix(r.URL.Path, "/")); err != nil {
			r.URL.Path = "/200.html"
		}
		fileServer.ServeHTTP(w, r)
	}
}

// apiError is the shared error envelope shape:
// {"schema_version":1,"error":{"code":"<snake_case>","message":"<human text>"}}
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorEnvelope struct {
	SchemaVersion int      `json:"schema_version"`
	Error         apiError `json:"error"`
}

// WriteJSON writes v as a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError writes the shared error envelope with the given status, code
// and message.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, errorEnvelope{
		SchemaVersion: schemaVersion,
		Error:         apiError{Code: code, Message: message},
	})
}
