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
	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/index"
	"github.com/davison/topos/kernel/webui"
)

// schemaVersion is the envelope's schema_version field. Bump only for
// breaking JSON shape changes.
const schemaVersion = 1

// Router builds the chi router serving /api/webspaces,
// /api/webspaces/{webspace}/stream, /api/items/{id} (+ /content,
// /thumbnail), /api/sources, /api/config (07-01-PLAN.md, the kernel's
// first mutating surface), the manual-refresh routes, and the mirrored
// /agent/v1/* namespace (MountAgentRoutes, agent.go).
//
// Router takes cfgStore *config.Store rather than a boot-time
// *config.Config (assumption-delta decision, 07-01-PLAN.md: the running
// configuration is the primary noun now, a live hash-identified resource
// with raw and expanded forms — not a value resolved once at startup).
// Every handler below resolves cfgStore.Expanded() fresh as the first
// statement of its own returned closure (07-02-PLAN.md Task 2 closed the
// last three handlers' boot-time snapshot gap 07-01 deliberately left
// open — there is no longer a local cfg value captured once here), so a
// filter, display-name edit, or newly added source saved through
// PUT /api/config or POST /api/config/reload is visible on the very next
// request to ANY route with no kernel restart (D-06/D-08/D-16). cfg is
// inert configuration data everywhere it's read, never a plugin handle,
// so none of this weakens KERN-02 / Pitfall 1: the stream/webspace/item
// routes remain structurally incapable of reaching a plugin (they never
// import kernel/pluginhost). The /api/items/* routes are the one
// deliberate exception: fetcher is the request-time, item-open plugin
// call path (KERN-03). prober and refresher are the other two: prober
// drives GET /api/sources's live reachability probe, and refresher is
// the same kernel/syncer.Coordinator the scheduler and the CLI use, so
// every caller of a sync reaches the identical single-flight entry point
// (D-06). applier is the apply-after-save/reload seam (07-02-PLAN.md
// Task 1; kernel/supervisor.Supervisor satisfies it structurally)
// ConfigSaveHandler and ConfigReloadHandler call after a successful
// config.Store mutation, so a save or reload reconfigures the running
// kernel in the same request rather than only the file (D-06/D-08).
// pluginsDir and logger (07-02-PLAN.md Task 3) feed PluginTypesHandler
// and DescribePluginHandler — the kernel-side half of the "+" chip
// picker's plugin-type discovery and trial-launch-then-Describe
// sequencing (D-11 step 1 -> step 2).
func Router(store *index.Store, cfgStore *config.Store, fetcher Fetcher, prober HealthProber, refresher Refresher, applier Applier, pluginsDir string, logger hclog.Logger) chi.Router {
	r := chi.NewRouter()
	r.Get("/api/webspaces", WebspacesHandler(store, cfgStore))
	r.Get("/api/webspaces/{webspace}/stream", StreamHandler(store, cfgStore))
	r.Get("/api/webspaces/{webspace}/search", SearchHandler(store, cfgStore))
	r.Get("/api/items/{id}", ItemHandler(store, cfgStore, fetcher))
	r.Get("/api/items/{id}/content", ItemContentHandler(store, fetcher))
	r.Get("/api/items/{id}/thumbnail", ItemThumbnailHandler(store, fetcher))
	r.Get("/api/sources", SourcesHandler(store, prober))
	r.Post("/api/sources/{name}/refresh", SourceRefreshHandler(cfgStore, refresher))
	r.Post("/api/sync", SyncRefreshHandler(refresher))
	// The kernel's first mutating HTTP surface (success criterion 4),
	// scoped strictly to configuration — see kernel/httpapi/config.go.
	r.Get("/api/config", ConfigHandler(cfgStore))
	r.Put("/api/config", ConfigSaveHandler(cfgStore, applier))
	// POST /api/config/reload (D-08): re-reads config.toml through the
	// same validate-then-apply path a save uses — the only way a
	// hand-edited file reaches the running kernel; there is deliberately
	// no file watcher.
	r.Post("/api/config/reload", ConfigReloadHandler(cfgStore, applier))
	// GET /api/config/plugin-types and POST /api/config/describe-plugin
	// (D-11 step 1 -> step 2): the kernel-side half of the "+" chip
	// picker's plugin-type discovery and trial-launch-then-Describe
	// sequencing — see kernel/httpapi/config.go.
	r.Get("/api/config/plugin-types", PluginTypesHandler(pluginsDir))
	r.Post("/api/config/describe-plugin", DescribePluginHandler(pluginsDir, logger))
	// MountAgentRoutes adds the /agent/v1 namespace (AGENT-01, D-12): a
	// default-deny, grant-filtered mirror of the routes above, over the
	// same store/cfgStore/fetcher/prober. Every /api/* route above is
	// unaffected by any grant configuration — grants gate the agent
	// surface only.
	MountAgentRoutes(r, store, cfgStore, fetcher, prober)
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
