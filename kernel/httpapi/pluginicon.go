package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// PluginIconProvider is the minimal icon-lookup surface PluginIconHandler
// depends on. *pluginhost.Host (via *supervisor.Supervisor's delegation,
// see kernel/supervisor/supervisor.go's PluginIcon method) satisfies this
// structurally — the same interface-next-to-its-handler pattern
// HealthProber (sources.go) and Fetcher (item.go) already establish.
type PluginIconProvider interface {
	PluginIcon(binary string) (iconBytes []byte, iconMIME string, ok bool)
}

// PluginIconHandler serves GET /api/plugins/{plugin}/icon: a plugin
// BINARY's (never an instance's) own declared identity icon, captured once
// at its launch-time Describe call and cached in-memory
// (09-01-PLAN.md Task 2, 09-UI-SPEC.md Fix 10). Keyed by binary name
// (source.plugin, e.g. "topos-plugin-mock") rather than source_type or
// instance id: two instances of one binary have byte-identical icons, and
// source_type — like the icon itself — is only learned once a plugin has
// actually been launched and Described.
//
// This route mutates nothing and issues no new RPC — Describe already ran
// at launch — so PLUG-02's read-only guarantee is untouched.
//
// A 404 here (the named binary has never successfully Described — no
// configured instance of it has ever launched, or every launch attempt so
// far failed before reaching Describe) is an expected, ROUTINE state,
// never a client-visible error: PluginIcon.svelte's mandatory fallback
// chain always covers it by rendering the Puzzle glyph instead. Do not
// treat a 404 from this route as exceptional.
func PluginIconHandler(icons PluginIconProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		plugin := chi.URLParam(r, "plugin")

		// {plugin} is untrusted URL input reaching an in-memory lookup
		// key (T-09-04). Reject anything containing a path separator or
		// a ".." segment BEFORE calling the provider — the lookup itself
		// is an exact-match map read, never a filesystem access, so this
		// guard makes traversal structurally impossible rather than
		// merely filtered.
		if plugin == "" || strings.ContainsAny(plugin, "/\\") || strings.Contains(plugin, "..") {
			WriteError(w, http.StatusNotFound, "icon_unavailable", "no icon is available for plugin \""+plugin+"\"")
			return
		}

		iconBytes, iconMIME, ok := icons.PluginIcon(plugin)
		if !ok {
			WriteError(w, http.StatusNotFound, "icon_unavailable", "no icon is available for plugin \""+plugin+"\"")
			return
		}

		sum := sha256.Sum256(iconBytes)
		etag := `"` + hex.EncodeToString(sum[:]) + `"`

		if inm := r.Header.Get("If-None-Match"); inm != "" && inm == etag {
			w.Header().Set("ETag", etag)
			w.WriteHeader(http.StatusNotModified)
			return
		}

		h := w.Header()
		h.Set("Content-Type", iconMIME)
		// Icon bytes are static for a given binary build — a rebuilt
		// plugin binary is a new process, not a live-reloadable asset —
		// so a long-lived immutable cache is safe (T-09-06 accepts the
		// narrow staleness window a plugin rebuild could otherwise cause;
		// ETag revalidation and a hard reload both resolve it).
		h.Set("Cache-Control", "public, max-age=31536000, immutable")
		h.Set("ETag", etag)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Content-Disposition", "inline")
		// T-09-01: a plugin-supplied SVG may carry <script> or event-
		// handler payloads. PluginIcon.svelte's <img> render path
		// neutralises them structurally, but this route is directly
		// navigable in the kernel's own origin — this CSP is the second,
		// independent mitigation: a directly-navigated SVG executes no
		// script and cannot reach the kernel API from that document. Not
		// redundant with the <img> path's safety; both layers are
		// required and neither may be dropped.
		h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(iconBytes)
	}
}
