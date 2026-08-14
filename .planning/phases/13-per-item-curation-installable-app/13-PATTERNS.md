# Phase 13: Per-Item Curation & Installable App - Pattern Map

**Mapped:** 2026-08-14
**Files analyzed:** 19
**Analogs found:** 17 / 19

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `kernel/index/schema.go` (edit — add `item_marks` table) | model/migration | CRUD | same file, `webspace_items`/`sync_runs` table blocks + `rebuildOnSchemaChange` drop list | exact (edit-in-place) |
| `kernel/index/store.go` (edit — mark read/write + prune-sweep) | model/service | CRUD | same file, `ReplaceWebspaceSourceItems` / scoped `DELETE FROM webspace_items` | exact (edit-in-place) |
| `kernel/httpapi/marks.go` (new) | controller/route | CRUD (request-response) | `kernel/httpapi/sources.go` (`SourceRefreshHandler`, `SyncRefreshHandler`) | role-match |
| `kernel/httpapi/stream.go` (edit — `?view=excluded`, mark filter) | controller/route | request-response | same file (`StreamHandler`) | exact (edit-in-place) |
| `kernel/httpapi/search.go` (edit — mark filter) | controller/route | request-response | same file (`SearchHandler`) | exact (edit-in-place) |
| `kernel/httpapi/agent.go` (verify/edit — mark filter on agent mirror) | controller/route | request-response | same file; `kernel/httpapi/stream.go` as cross-reference | exact (edit-in-place, pending open-question verification) |
| `kernel/httpapi/routes.go` (edit — register marks routes) | route/config | request-response | same file's existing `r.Post("/api/sources/{name}/refresh", ...)` block | exact (edit-in-place) |
| `kernel/pluginhost/manifest.go` (new) | service/utility | CRUD (in-memory lookup) | `kernel/pluginhost/binaryhash.go` (`HashBinary`) | role-match |
| `kernel/pluginhost/manifest_generated.go` (new, generated, git-ignored) | config/data | batch (build-time generated) | none — new generated-source pattern | no analog |
| `kernel/pluginhost/discover_binaries.go` (edit — `ResolveBinary` trusted-tier branch calls verify) | service | request-response | same file (existing trusted/external resolution logic) | exact (edit-in-place) |
| `kernel/pluginhost/host.go` (edit — new `LaunchFailureManifestUnverified` reason) | model/constant | event-driven | same file (`LaunchFailurePinMismatch`, `LaunchFailure` struct) | exact (edit-in-place) |
| `cmd/topos/gen-manifest/main.go` (new, build tool) | utility/config | batch | `kernel/pluginhost/binaryhash.go` (`HashBinary`); Makefile `plugins`/`plugins-portable` targets | role-match |
| `Makefile` (edit — reorder build, add `generate-manifest` step) | config | batch | same file (`plugins`, `build`, `build-portable`, `dev` targets) | exact (edit-in-place) |
| `web/src/lib/components/StreamRow.svelte` (edit — multi-select props/handlers) | component | event-driven | same file (existing `onclick={onselect}` button + `selected` prop) | exact (edit-in-place) |
| `web/src/lib/components/SelectionActionBar.svelte` (new) | component | event-driven | `web/src/lib/components/FilterChip.svelte` (chip/button composition, `Button size="sm"`) | role-match |
| `web/src/lib/components/DetailPane.svelte` (edit — exclude/include button) | component | request-response | same file (`OpenInSource.svelte` button usage, `Retry` `Button variant="outline" size="sm"`) | exact (edit-in-place) |
| `web/src/lib/components/WebspaceHeader.svelte` (edit — excluded-count toggle) | component | request-response | same file (existing trailing button group incl. "Refresh all") | exact (edit-in-place) |
| `web/src/lib/components/TrustBadge.svelte` (edit — `manifest_unverified`/`shadowed` states) | component | event-driven | same file (existing `pin_mismatch` destructive-tone state) | exact (edit-in-place) |
| `web/src/lib/api.ts` (edit — marks endpoints, `view` param, toast helpers) | service/utility | request-response | existing `getItem`, `contentUrl` fetch wrappers | exact (edit-in-place) |
| `web/src/routes/+layout.svelte` (edit — mount Sonner toaster) | provider | event-driven | existing root layout structure | exact (edit-in-place) |
| `web/vite.config.ts` (edit — `VitePWA`/`SvelteKitPWA` plugin) | config | batch | existing Vite plugin registration | exact (edit-in-place) |
| `web/e2e/specs/13-*.spec.ts` (new, several) | test | event-driven | `web/e2e/specs/11-binary-changed-repin.spec.ts`, `web/e2e/specs/12-filesystem-add-source.spec.ts` | exact |

## Pattern Assignments

### `kernel/index/schema.go` (model/migration, CRUD)

**Analog:** same file — existing table definitions + `rebuildOnSchemaChange`'s explicit drop list.

**Core pattern** (from RESEARCH.md, verified this session against `kernel/index/store.go:119-127`):
```go
// Add to the unconditional `schema` string (run on every Open(), NOT
// gated by schemaVersion — see Pitfall 1, no version bump):
CREATE TABLE IF NOT EXISTS item_marks (
    webspace_name TEXT NOT NULL,
    item_id       TEXT NOT NULL,   -- NO FK to items(id) — deliberate, see below
    kind          TEXT NOT NULL,   -- e.g. "excluded"
    created_unix  INTEGER NOT NULL,
    PRIMARY KEY (webspace_name, item_id)
);
```

**Rebuild-exemption pattern** — `item_marks` must NOT appear in this list (verbatim, `kernel/index/store.go:119-127`):
```go
if itemsTableExists != 0 {
    for _, stmt := range []string{
        `DROP TABLE IF EXISTS items_fts`,
        `DROP TRIGGER IF EXISTS items_ai`,
        `DROP TRIGGER IF EXISTS items_ad`,
        `DROP TRIGGER IF EXISTS items_au`,
        `DROP TABLE IF EXISTS webspace_items`,
        `DROP TABLE IF EXISTS webspaces`,
        `DROP TABLE IF EXISTS sync_runs`,
        `DROP TABLE IF EXISTS items`,
        // item_marks intentionally absent (D-10)
    } { /* ... */ }
}
```

**No-FK rationale:** `webspace_items.item_id` DOES carry `REFERENCES items(id) ON DELETE CASCADE` (`kernel/index/schema.go:49`). `item_marks` must not, or `DeleteSourceItems`'s existing `DELETE FROM items` would silently cascade-delete marks outside the explicit D-09/D-10 healthy-sync-only sweep.

---

### `kernel/index/store.go` (model/service, CRUD)

**Analog:** same file — `ReplaceWebspaceSourceItems`'s existing scoped-delete pattern.

**Core pattern to mirror** (verbatim, `kernel/index/store.go:208-213`):
```go
if _, err := tx.ExecContext(ctx, `
DELETE FROM webspace_items
WHERE webspace_name = ?
  AND item_id IN (SELECT id FROM items WHERE source = ?)
`, webspaceName, source); err != nil {
    return fmt.Errorf("index: clear webspace_items for %s/%s: %w", webspaceName, source, err)
}
```

**New prune-sweep** (same shape, added inside the same transaction, same function):
```go
// DELETE FROM item_marks
// WHERE webspace_name = ?
//   AND item_id IN (SELECT id FROM items WHERE source = ?)
//   AND item_id NOT IN (<ids just re-inserted into webspace_items>)
```
Note the open decision (RESEARCH.md Open Question 2 / Assumption A4): the de-allowlist branch (`items=nil`) also reaches this function — decide explicitly at plan time whether the unconditional sweep should also prune marks there.

**New mark read/write methods** — follow the existing method style in this file (`context.Context` first param, `tx`/`db` receiver, parameterized `?` placeholders throughout — no string concatenation, per the Security Domain V5/Known Threat Patterns table in RESEARCH.md).

**Mark-aware filter join point** — add to both `StreamItems` and `Search` (`kernel/index/store.go:333-437`, both funnel through `BuildMatchQuery`):
```sql
-- included view (default):
WHERE item_id NOT IN (SELECT item_id FROM item_marks WHERE webspace_name = ? AND kind = 'excluded')
-- excluded view (?view=excluded):
WHERE item_id IN (SELECT item_id FROM item_marks WHERE webspace_name = ? AND kind = 'excluded')
```
This must be a second, independent WHERE clause composed with the existing filter-stack clause, never replacing it.

---

### `kernel/httpapi/marks.go` (new controller, request-response)

**Analog:** `kernel/httpapi/sources.go` — `SourceRefreshHandler` (single-target POST) and `SyncRefreshHandler` (bulk POST), both illustrating this codebase's handler shape.

**Imports pattern** (from `kernel/httpapi/sources.go:1-16`):
```go
package httpapi

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/correlate"
	"github.com/davison/topos/kernel/index"
	"github.com/davison/topos/kernel/pluginhost"
	"github.com/davison/topos/kernel/syncer"
)
```

**Core handler pattern** (mirror `SourceRefreshHandler`, `kernel/httpapi/sources.go:276-305` and `SyncRefreshHandler:315-324`):
```go
func SourceRefreshHandler(cfgStore *config.Store, refresher Refresher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := cfgStore.Expanded()
		name := chi.URLParam(r, "name")
		if _, ok := cfg.Sources[name]; !ok {
			WriteError(w, http.StatusNotFound, "source_not_found", "source \""+name+"\" was not found")
			return
		}
		result, err := refresher.Refresh(r.Context(), name)
		if err != nil {
			WriteError(w, http.StatusNotFound, "source_not_found", "source \""+name+"\" was not found")
			return
		}
		WriteJSON(w, http.StatusOK, sourceRefreshResponse{SchemaVersion: schemaVersion, Source: toRunStatus(result)})
	}
}
```
Adapt this shape for: `POST /api/webspaces/{webspace}/items/{id}/exclude`, `.../include`, and bulk `POST /api/webspaces/{webspace}/marks` (array of ids + kind action) — 404 via `WriteError` + `WriteJSON` response envelope carrying `schema_version`, matching every other handler in this package. Validate `webspace`/`item id` inputs the same way `webspaceIsKnown` and `chi.URLParam` extraction already do in `stream.go` — never trust unchecked client-supplied ids in a raw SQL string (parameterize).

**Error handling pattern:** `WriteError(w, statusCode, "error_code", "human message")` — the sole error-response helper used across `httpapi` (seen in `sources.go`, `stream.go`).

---

### `kernel/httpapi/stream.go` / `search.go` (edit, request-response)

**Analog:** same files.

**Imports** (`kernel/httpapi/stream.go:1-14`):
```go
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
```

**Core pattern** (`kernel/httpapi/stream.go:68-88`, `StreamHandler`): read `cfg` fresh from `cfgStore` at the top of the closure (never captured once at Router construction — matches D-06 "no kernel restart" precedent this phase's mark-view toggle should follow too), resolve `webspace` via `chi.URLParam`, verify `webspaceIsKnown`, call `store.StreamItems(ctx, name, cfg.Webspaces[name].Filter)`. Add a `view` query param (`r.URL.Query().Get("view")`) read the same way `strconv`/`strings` helpers are already used elsewhere in this file, threaded into `StreamItems`/`Search` as an additional argument selecting the mark-filter branch (included vs excluded).

**Open verification item (RESEARCH.md Assumption A2 / Open Question 1):** confirm whether `kernel/httpapi/agent.go`'s stream mirror calls `StreamItems` directly (inherits the filter automatically) or has an independent query path (needs the identical filter added explicitly) — must be resolved during planning, not assumed.

---

### `kernel/pluginhost/manifest.go` + `manifest_generated.go` (new service, CRUD/lookup)

**Analog:** `kernel/pluginhost/binaryhash.go` (`HashBinary`, reused verbatim — do not add a second hashing helper).

**Existing hashing helper to reuse** (verbatim, `kernel/pluginhost/binaryhash.go`):
```go
func HashBinary(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("pluginhost: hash binary %s: %w", path, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("pluginhost: hash binary %s: %w", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
```

**Generated manifest shape** — a plain `map[string]string` const/var (binary name → hex digest), NOT `go:embed` (see RESEARCH.md "Alternatives Considered" — no embed.FS plumbing needed for a single Go-code call site).

---

### `kernel/pluginhost/discover_binaries.go` (edit, request-response launch gate)

**Analog:** same file — existing `ResolveBinary` trusted/external resolution logic (verified this session, lines ~368-398):
```go
func ResolveBinary(dirs Dirs, name string, logger hclog.Logger) (path string, tier Tier, err error) {
	if err := validatePluginBinaryName(name); err != nil {
		return "", "", err
	}
	if dirs.Trusted != "" {
		trustedPath := filepath.Join(dirs.Trusted, name)
		if info, statErr := os.Stat(trustedPath); statErr == nil && info.Mode().IsRegular() {
			if dirs.External != "" {
				externalPath := filepath.Join(dirs.External, name)
				if extInfo, extStatErr := os.Stat(externalPath); extStatErr == nil && extInfo.Mode().IsRegular() {
					if logger == nil {
						logger = hclog.NewNullLogger()
					}
					logger.Warn("plugin binary name shadowed: trusted copy wins, external copy ignored (D-11)",
						"binary", name)
				}
			}
			return trustedPath, TierTrusted, nil
		}
	}
	if dirs.External != "" {
		externalPath := filepath.Join(dirs.External, name)
		if info, statErr := os.Stat(externalPath); statErr == nil && info.Mode().IsRegular() {
			return externalPath, TierExternal, nil
		}
	}
	return "", "", fmt.Errorf("plugin binary %q not found in trusted directory %q or external directory %q", name, dirs.Trusted, dirs.External)
}
```
The `TierTrusted` return branch is where D-12/D-13 insert manifest verification: after resolving `trustedPath`, re-hash via `HashBinary` and compare against the generated manifest map; a mismatch or absent entry must route to a new `LaunchFailure` (see host.go below) instead of returning `TierTrusted` — fail closed, never demote-and-run.

---

### `kernel/pluginhost/host.go` (edit, model/constant)

**Analog:** same file — existing named-failure vocabulary (verbatim):
```go
const LaunchFailurePinMismatch = "pin_mismatch"

type LaunchFailure struct {
	Instance    string
	Plugin      string
	DisplayName string
	Tier        Tier
	Reason      string
	PinnedHash  string
	CurrentHash string
	Message     string
}
```
Add `const LaunchFailureManifestUnverified = "manifest_unverified"` beside it — extend the CLOSED-VOCABULARY reason set the SPA gates remedial UI on (`kernel/httpapi/sources.go`'s own doc comment calls this out explicitly), never a parallel ad-hoc mechanism.

---

### `Makefile` (edit, config/batch)

**Analog:** same file — existing `plugins`, `build`, `build-portable`, `dev` targets.

**Pitfall to fix as prerequisite (verified this session):** `build`/`build-portable` currently compile the kernel BEFORE plugins exist. Must reorder to: `plugins`/`plugins-portable` → `generate-manifest` (new step, hashes each `bin/plugins/topos-plugin-*` via the same `HashBinary` algorithm, writes `manifest_generated.go`) → `go build -o bin/topos`. `dev` already runs plugins-first (`dev: plugins`) and only needs the manifest step inserted before the kernel build. Beware Pitfall 6: drive the manifest generator off the exact explicit binary list each Makefile target already enumerates (not a blind `bin/plugins/*` glob) so a stale `topos-plugin-signal`/`topos-plugin-mockstrict` left from a prior build never leaks into the current manifest.

---

### `web/src/lib/components/StreamRow.svelte` (edit, component, event-driven)

**Analog:** same file — existing button/click structure (verbatim, current file):
```svelte
<button
	type="button"
	onclick={onselect}
	aria-pressed={selected}
	data-item-id={item.id}
	class={cn(
		'stream-row-surface flex w-full items-start gap-4 overflow-hidden rounded-lg border border-border bg-card p-4 text-left transition-colors hover:bg-card/80 max-md:h-[60px] max-md:gap-0 max-md:p-1',
		'focus-visible:ring-ring focus-visible:ring-offset-background focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none',
		selected && 'border-l-primary border-l-2'
	)}
>
```
Add `bulkSelected`, `bulkModeActive` props and an `onbulktoggle: (id, mode) => void` callback; branch the click handler: plain click → unchanged `onselect`; ctrl/cmd-click → `onbulktoggle(item.id, 'toggle')`, no `onselect`; shift-click → `onbulktoggle(item.id, 'range')`, no `onselect`. Existing `selected`/`onselect` props stay orthogonal (E1 in 13-UI-SPEC.md) — bulk-selected renders via `bg-secondary/60` fill + a leading checkbox slot, never the `border-l-primary` accent already reserved for "currently open."

---

### `web/src/lib/components/SelectionActionBar.svelte` (new component, event-driven)

**Analog:** `web/src/lib/components/FilterChip.svelte` — closest existing component combining a `Button size="sm"` action row with chip-style surface/typography (Label role, 14px/400/1.4) that 13-UI-SPEC.md E2 explicitly cites as the typography match. Follow the existing shadcn-svelte `Button`/`Separator` import pattern already used throughout `web/src/lib/components/ui/*` wrappers.

---

### `web/src/lib/components/DetailPane.svelte` (edit, component, request-response)

**Analog:** same file — existing `Button` usage (verbatim):
```svelte
import { Button } from '$lib/components/ui/button/index.js';
...
<Button variant="outline" size="sm" onclick={() => loadContent(item.id)}>Retry</Button>
```
Add the exclude/include button using the identical `Button` import + `variant`/`size` convention (accent/primary styling per 13-UI-SPEC.md's Color rationale — NOT `variant="destructive"`), calling the new marks API (`web/src/lib/api.ts`) and firing the shared undo-toast helper on success.

---

### `web/src/lib/components/TrustBadge.svelte` (edit, component, event-driven)

**Analog:** same file — existing `pin_mismatch` destructive-tone state is the direct precedent for the new `manifest_unverified` state (same destructive `--destructive` token) and `shadowed` state (new `--warning` token usage, mirroring Phase 12's zero-match warning-tone precedent per 13-UI-SPEC.md Color table).

---

### `web/e2e/specs/13-*.spec.ts` (new tests)

**Analog:** `web/e2e/specs/11-binary-changed-repin.spec.ts` (named-failure-state spec pattern — directly relevant to the manifest_unverified/shadowed specs) and `web/e2e/specs/12-filesystem-add-source.spec.ts` (source-lifecycle spec pattern). Follow the existing `NN-description.spec.ts` naming convention; this phase's UI-heavy criteria (multi-select+undo, excluded view, refuse-to-load states, PWA manifest/SW presence where browser-drivable) must land as specs, not manual-only checks (project convention, `docs/testing.md`).

---

## Shared Patterns

### Error handling / JSON envelope
**Source:** `kernel/httpapi/sources.go`, `kernel/httpapi/stream.go` — `WriteError(w, statusCode, "error_code", "message")` and `WriteJSON(w, statusCode, responseStruct{SchemaVersion: schemaVersion, ...})`.
**Apply to:** `kernel/httpapi/marks.go` and any edits to `stream.go`/`search.go`/`agent.go`.

### Parameterized SQL, no string concatenation
**Source:** every example in `kernel/index/store.go` uses `?` placeholders exclusively (e.g. `ReplaceWebspaceSourceItems`'s scoped deletes).
**Apply to:** all new `item_marks` read/write SQL — client-supplied item/webspace ids must never be concatenated into a query string (Security Domain V5 in RESEARCH.md).

### Named, closed-vocabulary failure/health states
**Source:** `kernel/pluginhost/host.go`'s `LaunchFailurePinMismatch`; `kernel/httpapi/sources.go`'s doc comment calling this a "CLOSED-VOCABULARY reason."
**Apply to:** `LaunchFailureManifestUnverified`, the D-14 shadowing advisory, and `TrustBadge.svelte`'s new states — extend the existing enum, never invent a parallel ad-hoc string.

### Fresh-config-per-request (no stale capture)
**Source:** `kernel/httpapi/stream.go`'s `StreamHandler` reads `cfgStore.Expanded()` as the first statement of the closure so a config change is visible on the very next request.
**Apply to:** any new marks/manifest-aware handler that needs live config or trust state.

### Reuse existing hashing convention
**Source:** `kernel/pluginhost/binaryhash.go`'s `HashBinary` — documented "one-hashing-convention discipline."
**Apply to:** manifest generation tool and `manifest.go`'s verify helper — do not add a second SHA-256 implementation.

### shadcn-svelte Button/component import convention
**Source:** `web/src/lib/components/DetailPane.svelte` (`import { Button } from '$lib/components/ui/button/index.js'`), consistent across every existing component.
**Apply to:** `SelectionActionBar.svelte`, `DetailPane.svelte` edits, `WebspaceHeader.svelte` toggle, toast layer wiring (`sonner` block installed the same way every `ui/*` wrapper was — official registry, no vetting gate per 13-UI-SPEC.md).

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `kernel/pluginhost/manifest_generated.go` | config/data | batch | First generated-Go-source build artifact in this codebase — no prior generated `.go` file exists to pattern-match; follow the plain `map[string]string` shape RESEARCH.md specifies instead |
| `web/vite.config.ts` PWA plugin wiring / `web/svelte.config.js` fallback-revisioning interaction | config | batch | First PWA tooling addition in this repo; no existing analog. Use RESEARCH.md's cited `vite-plugin-pwa`/`@vite-pwa/sveltekit` docs and Pitfall 3 (200.html fallback revisioning) directly — verify end-to-end during implementation rather than assuming a generic example applies |

## Metadata

**Analog search scope:** `kernel/index/`, `kernel/httpapi/`, `kernel/pluginhost/`, `web/src/lib/components/`, `web/e2e/specs/`, `Makefile`
**Files scanned:** ~30 (targeted reads/greps; full-file reads limited to `stream.go`, `StreamRow.svelte`, partial reads of `sources.go`, `DetailPane.svelte`, `discover_binaries.go`)
**Pattern extraction date:** 2026-08-14
**Note:** 13-RESEARCH.md already contains extensive verified code excerpts (schema.go rebuild-drop list, store.go scoped-delete, binaryhash.go, host.go LaunchFailure) — this document cross-references rather than re-reading those already-quoted ranges, per the no-re-read discipline.

---
*Phase: 13-per-item-curation-installable-app*
*Patterns mapped: 2026-08-14*
