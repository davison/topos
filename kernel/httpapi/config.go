package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"reflect"
	"regexp"

	"github.com/davison/topos/kernel/config"
)

// Applier is the minimal apply-after-save surface ConfigSaveHandler and
// ConfigReloadHandler depend on — *supervisor.Supervisor satisfies this
// structurally. Kept as an interface, rather than an import of
// kernel/supervisor, so kernel/httpapi never imports the supervisor
// package: the config write path must never grow a dependency on plugin
// process lifecycle beyond the read-only trial-launch it already needs
// for DescribePluginType (success criterion 4, T-07-11).
type Applier interface {
	// Apply reconciles the running kernel against the config.Store's
	// current swapped state (D-06/D-07). See kernel/supervisor.Supervisor.Apply.
	Apply(ctx context.Context) error
}

// configResponse is the shared shape both GET /api/config and a successful
// PUT /api/config return — one envelope, no separate "save result" shape
// (D-06: save = apply immediately, so the post-save state IS the next
// GET's state).
//
// Config always serializes Store.Raw() — the unexpanded, pre-os.Expand
// form — NEVER Store.Expanded(). This is D-05's hard requirement at the
// HTTP boundary: a resolved secret VALUE must never reach the browser, so
// this handler is structurally incapable of returning one (T-07-01).
type configResponse struct {
	SchemaVersion int             `json:"schema_version"`
	Hash          string          `json:"hash"`
	Config        *config.Config  `json:"config"`
	EnvVars       map[string]bool `json:"env_vars"`
	UnknownKeys   []string        `json:"unknown_keys"`
}

// envVarPattern matches both ${NAME} and bare $NAME env-var reference
// shapes — the same two forms os.Expand itself recognises (config.go's
// expandEnv delegates to os.Expand directly).
var envVarPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}|\$([A-Za-z_][A-Za-z0-9_]*)`)

// envVarsIn scans every string field reachable from cfg (structs, maps,
// slices, pointers) for ${VAR}/$VAR references and reports, per referenced
// name, whether that variable is currently set in the kernel process's own
// environment (os.LookupEnv) — a boolean only, never the value (D-05: the
// response boundary carries references and booleans, never secret values).
func envVarsIn(cfg *config.Config) map[string]bool {
	names := map[string]struct{}{}
	collectEnvVarNames(reflect.ValueOf(cfg), names)

	out := make(map[string]bool, len(names))
	for name := range names {
		_, ok := os.LookupEnv(name)
		out[name] = ok
	}
	return out
}

func collectEnvVarNames(v reflect.Value, names map[string]struct{}) {
	if !v.IsValid() {
		return
	}
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return
		}
		collectEnvVarNames(v.Elem(), names)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			collectEnvVarNames(v.Field(i), names)
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			collectEnvVarNames(v.MapIndex(key), names)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			collectEnvVarNames(v.Index(i), names)
		}
	case reflect.String:
		for _, m := range envVarPattern.FindAllStringSubmatch(v.String(), -1) {
			name := m[1]
			if name == "" {
				name = m[2]
			}
			if name != "" {
				names[name] = struct{}{}
			}
		}
	}
}

// unknownKeysOrEmpty normalises UnknownKeys' nil-when-none result to an
// empty JSON array, matching keywordsOrEmpty's convention elsewhere in this
// package — API consumers never need to special-case a null field.
func unknownKeysOrEmpty(keys []string) []string {
	if keys == nil {
		return []string{}
	}
	return keys
}

func toConfigResponse(cfgStore *config.Store) configResponse {
	raw := cfgStore.Raw()
	return configResponse{
		SchemaVersion: schemaVersion,
		Hash:          cfgStore.Hash(),
		Config:        raw,
		EnvVars:       envVarsIn(raw),
		UnknownKeys:   unknownKeysOrEmpty(cfgStore.UnknownKeys()),
	}
}

// ConfigHandler serves GET /api/config: the current raw (pre-expansion)
// config document, its content hash (the base_hash a subsequent PUT must
// echo back, D-03), and the set-or-not status of every ${VAR} reference it
// contains. This is a plain read of cfgStore's in-memory state — it never
// re-reads the file from disk (Store.Save/Reload own that).
func ConfigHandler(cfgStore *config.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, toConfigResponse(cfgStore))
	}
}

// configSaveRequest is PUT /api/config's request body: the base_hash the
// client last read (D-03's clobber-guard proof) and the edited raw config
// document to persist in its place.
type configSaveRequest struct {
	BaseHash string         `json:"base_hash"`
	Config   *config.Config `json:"config"`
}

// ConfigSaveHandler serves PUT /api/config — the kernel's first mutating
// HTTP surface (success criterion 4). It does very little config logic
// itself: every rule up to and including the write (the clobber guard,
// the unknown-key guard, the validate-dry-run, the canonical write, the
// in-memory hot-swap) lives in config.Store.Save; this handler decodes
// the request, calls Save, calls applier.Apply to reconcile the running
// kernel against the just-swapped config (D-06), and maps both calls'
// error classes onto the shared HTTP error envelope.
func ConfigSaveHandler(cfgStore *config.Store, applier Applier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req configSaveRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_request", "request body is not valid JSON: "+err.Error())
			return
		}
		if req.Config == nil {
			WriteError(w, http.StatusBadRequest, "invalid_request", "request body is missing \"config\"")
			return
		}

		if err := cfgStore.Save(req.Config, req.BaseHash); err != nil {
			switch {
			case errors.Is(err, config.ErrConfigChangedOnDisk):
				WriteError(w, http.StatusConflict, "config_changed_on_disk", "config changed on disk — review and retry")
			case errors.Is(err, config.ErrConfigHasUnknownKeys):
				WriteError(w, http.StatusConflict, "config_has_unknown_keys", err.Error())
			default:
				WriteError(w, http.StatusUnprocessableEntity, "config_invalid", err.Error())
			}
			return
		}

		// T-07-11 / D-06: the file is already written and cfgStore's
		// in-memory pointers already swapped by this point — an apply
		// failure here means the RUNNING kernel (plugin host, coordinator,
		// scheduler) has not fully caught up with that swap. This is never
		// a silent 200: the response says so explicitly, and names
		// POST /api/config/reload (kernel/httpapi's Task 2 addition) as the
		// recovery path, since a reload re-runs this exact validate-then-
		// apply sequence against the file that IS now on disk.
		if err := applier.Apply(r.Context()); err != nil {
			WriteError(w, http.StatusInternalServerError, "apply_failed",
				"config.toml was written and is now the kernel's config-of-record, but the running kernel could not fully apply it — the runtime configuration may be out of sync with the file until a successful POST /api/config/reload: "+err.Error())
			return
		}

		WriteJSON(w, http.StatusOK, toConfigResponse(cfgStore))
	}
}
