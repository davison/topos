package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/pluginhost"
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

// envVarsIn reports, per ${VAR}/$VAR reference reachable from cfg (via
// config.EnvRefNames — Phase 11 Task 2's single shared scanner, moved to
// kernel/config so kernel/pluginhost's plugin-launch env allowlist can use
// the identical implementation), whether that variable is currently set in
// the kernel process's own environment (os.LookupEnv) — a boolean only,
// never the value (D-05: the response boundary carries references and
// booleans, never secret values).
func envVarsIn(cfg *config.Config) map[string]bool {
	names := config.EnvRefNames(cfg)
	out := make(map[string]bool, len(names))
	for _, name := range names {
		_, ok := os.LookupEnv(name)
		out[name] = ok
	}
	return out
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

// ConfigReloadHandler serves POST /api/config/reload (D-08): re-reads
// config.toml from disk through the identical validate-then-apply path a
// UI save uses — the only way a hand-edited file reaches the running
// kernel, since there is deliberately no file watcher (07-CONTEXT.md).
//
// cfgStore.Reload's own documented contract (kernel/config/store.go)
// loads into locals and swaps only on full success — a failed reload
// therefore leaves the previous raw/expanded pointers and hash completely
// untouched, so this handler's 422 response is never followed by a
// half-applied in-memory state: the running kernel keeps serving exactly
// the last-good configuration it was already serving, and never exits.
func ConfigReloadHandler(cfgStore *config.Store, applier Applier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := cfgStore.Reload(); err != nil {
			WriteError(w, http.StatusUnprocessableEntity, "config_invalid", err.Error())
			return
		}

		// Same apply-after-mutate discipline as ConfigSaveHandler: the file
		// is already reloaded and cfgStore's pointers already swapped by
		// this point, so an apply failure here is a runtime reconciliation
		// problem, never a config-validity one — never a silent 200.
		if err := applier.Apply(r.Context()); err != nil {
			WriteError(w, http.StatusInternalServerError, "apply_failed",
				"config.toml was reloaded and is now the kernel's config-of-record, but the running kernel could not fully apply it — retry POST /api/config/reload: "+err.Error())
			return
		}

		WriteJSON(w, http.StatusOK, toConfigResponse(cfgStore))
	}
}

// pluginTypesResponse is GET /api/config/plugin-types's response: the
// plugin binaries actually present on disk (pluginhost.DiscoverTiered),
// sorted, excluding the mock reference fixture — never a built-in table
// of known plugin types.
//
// PluginTypeTiers is an ADDITIVE sibling field (Phase 11, PLUG-06/07): a
// tier lookup table spanning EVERY discovered binary in BOTH tiers, keyed
// by binary name — sourced from pluginhost.DiscoverAllTiered, not
// DiscoverTiered, so it deliberately INCLUDES the excluded fixture names
// (topos-plugin-mock/-mockstrict). This is intentional, not an
// inconsistency with PluginTypes above: PluginTypeTiers is a lookup table
// for names a caller ALREADY HOLDS (e.g. resolving the tier of an
// already-configured instance's binary, which may legitimately be an
// excluded fixture — see DescribePluginHandler's own doc comment for the
// identical DiscoverBinaries/DiscoverAllBinaries split this mirrors),
// never a second catalog to browse. No schema_version bump accompanies
// this addition: PluginTypes' own element shape (a bare string) is
// unchanged, and an API consumer that does not know about
// PluginTypeTiers simply never reads it — the same additive-field
// discipline every other Phase 11 wire addition in this repo follows.
type pluginTypesResponse struct {
	SchemaVersion   int               `json:"schema_version"`
	PluginTypes     []string          `json:"plugin_types"`
	PluginTypeTiers map[string]string `json:"plugin_type_tiers"`
}

// PluginTypesHandler serves GET /api/config/plugin-types — the kernel
// half of D-11's "+" chip picker's "New <plugin type>…" list. Only the
// kernel process can see the plugins/ directories on the desktop
// machine's filesystem; the browser has no other way to learn which
// plugin types are installed, in which tier.
func PluginTypesHandler(dirs pluginhost.Dirs) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		catalog, err := pluginhost.DiscoverTiered(dirs)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		names := make([]string, len(catalog))
		for i, tb := range catalog {
			names[i] = tb.Name
		}

		all, err := pluginhost.DiscoverAllTiered(dirs)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		tiers := make(map[string]string, len(all))
		for _, tb := range all {
			tiers[tb.Name] = string(tb.Tier)
		}

		WriteJSON(w, http.StatusOK, pluginTypesResponse{SchemaVersion: schemaVersion, PluginTypes: names, PluginTypeTiers: tiers})
	}
}

// describePluginRequest is POST /api/config/describe-plugin's request
// body: the plugin binary name (must be a member of
// pluginhost.DiscoverAllTiered's own result set — see
// DescribePluginHandler) and the connection fields the operator has typed
// into step 1 of the "New <plugin type>…" modal, OR an already-configured
// instance's own stored connection fields (the "+" picker's one-step
// existing-instance add flow reuses this same request shape — see
// AddSourceModal.svelte's own doc comment), not yet persisted anywhere by
// this call either way.
type describePluginRequest struct {
	Plugin string        `json:"plugin"`
	Source config.Source `json:"source"`
}

// describePluginResponse is POST /api/config/describe-plugin's response:
// the three Describe-derived facts step 2 of the modal needs to build its
// match-vocabulary-driven form. No connection field from the request is
// ever echoed back here (T-07-10) — only source_type, the plugin KIND's
// own display name, and its declared match vocabulary.
type describePluginResponse struct {
	SchemaVersion     int      `json:"schema_version"`
	SourceType        string   `json:"source_type"`
	PluginDisplayName string   `json:"plugin_display_name"`
	MatchVocabulary   []string `json:"match_vocabulary"`
}

// DescribePluginHandler serves POST /api/config/describe-plugin (D-11
// step 1 -> step 2, and the "+" picker's one-step existing-instance add
// flow): trial-launches the named plugin binary against the just-submitted
// (or already-configured) connection fields, calls its Describe RPC, and
// kills the subprocess before this handler returns — writing nothing to
// disk and registering nothing on the running kernel's plugin host (see
// pluginhost.DescribePluginType's own doc comment).
//
// The requested plugin MUST be a member of pluginhost.DiscoverAllTiered's
// own result set (spanning both directories), checked BEFORE anything is
// executed (T-07-09, extended to two tiers T-11-02): a request naming an
// arbitrary binary/path is refused 404 plugin_binary_not_found and never
// reaches exec.Command — directory listing across BOTH configured
// directories, never a caller-supplied path, is the authority over what
// may be launched. Deliberately DiscoverAllTiered, not DiscoverTiered:
// the latter's ExcludedPluginBinaries filtering (kernel/pluginhost/
// discover_binaries.go) is a UI-policy concern scoped to what the "+ New
// <plugin type>…" picker OFFERS (PluginTypesHandler, below) — it must not
// also block describing an instance whose plugin type is ALREADY
// legitimately configured, which is exactly what the one-step
// existing-instance flow does for every already-configured instance,
// including a topos-plugin-mock one (see DiscoverAllBinaries' own doc
// comment for the regression this fixes, which DiscoverAllTiered
// inherits unchanged).
func DescribePluginHandler(dirs pluginhost.Dirs, logger hclog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req describePluginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_request", "request body is not valid JSON: "+err.Error())
			return
		}

		available, err := pluginhost.DiscoverAllTiered(dirs)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		known := false
		for _, tb := range available {
			if tb.Name == req.Plugin {
				known = true
				break
			}
		}
		if !known {
			WriteError(w, http.StatusNotFound, "plugin_binary_not_found", "plugin \""+req.Plugin+"\" is not a discovered plugin binary")
			return
		}

		// The submitted source's Plugin field is authoritative from the
		// validated req.Plugin value, never trusted verbatim from the
		// request body's nested source object — the two could otherwise
		// disagree.
		req.Source.Plugin = req.Plugin

		info, err := pluginhost.DescribePluginType(r.Context(), dirs, req.Source, logger)
		if err != nil {
			WriteError(w, http.StatusBadGateway, "plugin_describe_failed", err.Error())
			return
		}

		WriteJSON(w, http.StatusOK, describePluginResponse{
			SchemaVersion:     schemaVersion,
			SourceType:        info.SourceType,
			PluginDisplayName: info.PluginDisplayName,
			MatchVocabulary:   info.MatchVocabulary,
		})
	}
}
