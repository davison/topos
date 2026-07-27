// Package httpapi implements the kernel's loopback JSON HTTP API. The same
// JSON contract serves both the embedded SPA and any programmatic agent
// (AGENT-02) — there is no separate "agent API".
package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/darrendavison/webspaces/kernel/config"
	"github.com/darrendavison/webspaces/kernel/index"
)

// schemaVersion is the envelope's schema_version field. Bump only for
// breaking JSON shape changes.
const schemaVersion = 1

// Router builds the chi router serving /api/webspaces and
// /api/webspaces/{webspace}/stream. StreamHandler deliberately takes only
// *index.Store — httpapi cannot import kernel/pluginhost, so the stream
// route is structurally incapable of reaching a plugin (KERN-02 /
// Pitfall 1).
func Router(store *index.Store, cfg *config.Config) chi.Router {
	r := chi.NewRouter()
	r.Get("/api/webspaces", WebspacesHandler(store, cfg))
	r.Get("/api/webspaces/{webspace}/stream", StreamHandler(store))
	return r
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
