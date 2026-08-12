# Phase 7: Webspace Builder UI - Pattern Map

**Mapped:** 2026-08-07
**Files analyzed:** 16 (new/modified, backend + frontend)
**Analogs found:** 14 / 16 (2 have no close analog — genuinely new infrastructure)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `kernel/config/store.go` (NEW) | service/state-holder | event-driven (config swap) | `kernel/pluginhost/host.go` (`Host` struct + methods) — closest "owns a live mutable resource, exposes methods over it" shape in the repo | partial — no existing atomic-swap holder anywhere |
| `kernel/config/writer.go` (NEW) | utility (serializer) | transform | `kernel/config/config.go` (`Load`, `expandEnv`) — inverse operation, same package/file conventions | role-match (inverse direction) |
| `kernel/config/types.go` (MODIFIED — add `Webspace.Filter`) | model | CRUD | itself — extend existing `Webspace` struct in place | exact |
| `kernel/config/config.go` (MODIFIED — dual-parse Load) | model/loader | transform | itself — extend existing `Load`/`expandEnv` | exact |
| `kernel/httpapi/config.go` (NEW) | controller (HTTP handlers) | request-response (first mutating) | `kernel/httpapi/sources.go` (`SourceRefreshHandler`, POST-with-validation shape) | exact (closest POST handler with validate-then-act) |
| `kernel/httpapi/routes.go` (MODIFIED — mount new routes) | route | request-response | itself | exact |
| `kernel/httpapi/stream.go` / `search.go` / `agent.go` (MODIFIED — filter-stack query builder) | controller | request-response | `kernel/index/store.go` (`ftsQuery` at ~line 375-390) for the query-building half; `kernel/httpapi/webspaces.go` (`aggregateSyncStatus`) for the "one shared helper called from 3 handlers" shape | exact |
| `kernel/pluginhost/host.go` (MODIFIED — add `DescribePluginType`, `Reconcile`) | service | event-driven / RPC | itself — extend `launch`/`Discover`/`Shutdown` | exact |
| `kernel/pluginhost/discover_binaries.go` (NEW) | utility | file-I/O | `kernel/pluginhost/host.go`'s `launch` (binary path resolution via `os.Stat`) | role-match |
| `kernel/index/store.go` (MODIFIED — add `DeleteSourceItems`) | model/store (CRUD) | CRUD | `kernel/index/store.go`'s own `ReplaceWebspaceSourceItems` (~line 191) — nearly identical shape, one is delete-only | exact |
| `kernel/syncer/scheduler.go` (MODIFIED — restartable Run) | service | event-driven | itself — extend existing `Run`/goroutine-per-source loop | exact |
| `web/src/lib/components/ui/dialog/` (NEW) | component (primitive) | — | `web/src/lib/components/ui/popover/` — existing bits-ui wrapper pattern | exact |
| `web/src/lib/components/ui/dropdown-menu/` (NEW) | component (primitive) | — | `web/src/lib/components/ui/popover/` | exact |
| `web/src/lib/components/WebspaceSwitcher.svelte` (NEW) | component | request-response (loads webspace list) | `web/src/routes/+page.svelte` (existing webspace-list load pattern) + `web/src/lib/components/SourceChip.svelte`'s Popover-driven trigger pattern | role-match |
| `web/src/lib/components/AddSourceModal.svelte` / `EditSourceModal.svelte` / `ManageSourcesModal.svelte` (NEW) | component (form/modal) | request-response | `web/src/lib/components/WebspaceHeader.svelte`'s existing `Popover`/`PopoverContent` usage (source-chip overflow menu, lines 199-231) | role-match |
| `web/src/lib/components/FilterChip.svelte` (NEW) | component | — | `web/src/lib/components/SourceChip.svelte` — near-identical chip shape, distinct styling | exact |
| `web/src/lib/components/WebspaceHeader.svelte` (MODIFIED) | component | request-response | itself | exact |
| `web/src/routes/+page.svelte` (MODIFIED → redirect-only) | route | request-response | itself | exact |
| `web/src/lib/api.ts` (MODIFIED — new config/describe/save/reload client fns) | utility (HTTP client) | request-response | itself — `getJSON`/`postJSON` + existing typed fn-per-route pattern | exact |

## Pattern Assignments

### `kernel/config/config.go` — dual-parse `Load` (D-05)

**Analog:** itself, `kernel/config/config.go:21-48`

**Current pattern to extend** (lines 21-48):
```go
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}
	expanded, missing := expandEnv(string(raw))
	var cfg Config
	if err := toml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}
	applyDefaults(&cfg)
	if err := cfg.expandIndexPathHome(); err != nil { return nil, err }
	if err := cfg.expandSourceCACertPathsHome(); err != nil { return nil, err }
	if err := cfg.Validate(missing); err != nil { return nil, err }
	return &cfg, nil
}
```

**Change required:** unmarshal `raw` bytes a second time (unexpanded) into a second `Config` struct, apply only structural defaults (`applyDefaults`) to it — never `os.Expand`, never home-dir expansion (those touch fields that could be secret-shaped). Return `(expandedCfg, rawCfg, fileHash, error)`. See RESEARCH.md Pattern 1 for the exact target shape — this is the single most load-bearing change in the phase; follow the existing error-wrapping convention (`fmt.Errorf("config: ...: %w", ...)`) exactly.

**Validation-reuse pattern (D-09):** `Validate(missing []string) error` (lines 180-222) is the byte-for-byte dry-run the save endpoint must call — do not duplicate any of its rules (`validateWebspaces`, `validateMatchBlocks`, `validateSourcesAllowlist`, `validateFallbackCoverage`, `validateDisplayNameUniqueness`, all sorted-iteration for deterministic errors). Every error string is prefixed `"config: ..."` — the UI must surface these verbatim (D-09).

**Filter field (D-17):** add to `Webspace` struct in `kernel/config/types.go` (after `Match map[string]MatchBlock`, line 181):
```go
// Filter is the promoted-search permanent filter stack (D-16/D-17/D-18):
// each entry is an AND-ed FTS term, appended by "Save as filter" and
// removed independently. Empty means no permanent filter is active.
Filter []string `toml:"filter,omitempty"`
```
Follow the existing doc-comment density/style on `Keywords`/`Sources`/`Match` — every field in this file carries a multi-sentence rationale comment; match that convention.

---

### `kernel/config/writer.go` (NEW) — canonical TOML writer (D-01/D-02)

**Analog:** inverse of `Load`; no direct precedent for writing exists yet. Follow package conventions from `config.go`: same error-wrap prefix `"config: ..."`, same package `config`, same import grouping (stdlib first, then `github.com/pelletier/go-toml/v2`).

**Pattern (from RESEARCH.md Pattern 2, ready to use as-is):**
```go
const canonicalHeader = `# managed by the topos UI — hand-edits are honored via Reload
# see config.example.toml for full field documentation

`

func WriteCanonical(path string, rawCfg *Config) error {
	body, err := toml.Marshal(rawCfg)
	if err != nil {
		return fmt.Errorf("config: marshal canonical form: %w", err)
	}
	out := append([]byte(canonicalHeader), body...)

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".config-*.toml")
	if err != nil {
		return fmt.Errorf("config: create temp file: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(out); err != nil { tmp.Close(); return fmt.Errorf("config: write temp file: %w", err) }
	if err := tmp.Sync(); err != nil { tmp.Close(); return fmt.Errorf("config: fsync temp file: %w", err) }
	if err := tmp.Close(); err != nil { return fmt.Errorf("config: close temp file: %w", err) }
	if err := os.Rename(tmp.Name(), path); err != nil { return fmt.Errorf("config: rename into place: %w", err) }
	return nil
}
```

**IMPORTANT (Pitfall 3):** before this writer is safe to use, add `omitempty` to `Source.BaseURL`, `Source.Token`, `Source.APIVersion` in `kernel/config/types.go` (lines 42-44 — currently no `omitempty`), or a canonical rewrite of a local-path source (Signal) will emit spurious `base_url = ""` / `token = ""` / `api_version = ""`.

**Anti-pattern (D-05, hard requirement):** never call `toml.Marshal` on the expanded runtime `*Config` — only ever on the raw/unexpanded one from `Load`'s second parse.

---

### `kernel/config/store.go` (NEW) — swappable config holder (D-03/D-06)

**Analog:** `kernel/pluginhost/host.go`'s `Host` struct — closest existing "long-lived owner of a mutable process-wide resource with methods to read/replace it" shape, though `Host` itself is not swappable (`Discover`/`Shutdown` are whole-lifecycle, not incremental). No existing `atomic.Pointer`/`sync.Mutex` usage anywhere in `kernel/` — this is genuinely new infrastructure (confirmed via grep in RESEARCH.md).

**Content-hash lock pattern (D-03, from RESEARCH.md, ready to use):**
```go
func fileHash(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (s *Store) Save(ctx context.Context, mutate func(*config.Config) error) error {
	current, err := os.ReadFile(s.path)
	if err != nil {
		return fmt.Errorf("config: re-read before save: %w", err)
	}
	if fileHash(current) != s.lastKnownHash {
		return ErrConfigChangedOnDisk
	}
	// apply mutate() to a copy of the raw struct, dry-run validate the
	// resulting expanded struct (config.Validate), write canonically
	// (writer.WriteCanonical), Swap() the atomic pointers, update
	// s.lastKnownHash to the new file's hash
	return nil
}
```

**Error-sentinel convention to follow:** `kernel/pluginhost/host.go` declares package-level sentinel errors this way (lines 30-38):
```go
var ErrItemNotFound = errors.New("pluginhost: item not found")
var ErrSourceUnavailable = errors.New("pluginhost: source unavailable")
```
Mirror this exactly for `config.ErrConfigChangedOnDisk` (same `errors.New("config: ...")` shape, package-prefixed message).

---

### `kernel/httpapi/config.go` (NEW) — mutating config endpoints (D-06/D-08/D-09)

**Analog:** `kernel/httpapi/sources.go` — `SourceRefreshHandler` (lines 153-174) is the closest existing "validate the path param against known state, then call into a lower-layer coordinator, translate its error into the shared envelope" shape.

**Imports pattern** (`kernel/httpapi/sources.go:1-15`):
```go
package httpapi

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/kernel/index"
	"github.com/davison/topos/kernel/pluginhost"
	"github.com/davison/topos/kernel/syncer"
)
```

**Core request-response + error-envelope pattern** (`kernel/httpapi/sources.go:153-174`):
```go
func SourceRefreshHandler(cfg *config.Config, refresher Refresher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
Apply the identical shape for `POST /api/config/save`: validate-dry-run first (reuse `config.Validate` — no reimplementation, D-09), then hash-recheck (`config.ErrConfigChangedOnDisk` → 409-style envelope, distinct `code` e.g. `"config_changed_on_disk"`), then write+apply. Use `httpapi.WriteJSON`/`WriteError` (`kernel/httpapi/routes.go:94-108`) for every response — no ad-hoc `json.NewEncoder` calls anywhere else.

**Response envelope convention** (`kernel/httpapi/routes.go:82-99`):
```go
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type errorEnvelope struct {
	SchemaVersion int      `json:"schema_version"`
	Error         apiError `json:"error"`
}
func WriteJSON(w http.ResponseWriter, status int, v interface{}) { ... }
func WriteError(w http.ResponseWriter, status int, code, message string) { ... }
```
Every new response type in `config.go` must embed `SchemaVersion int `json:"schema_version"`` as its first field, matching `sourcesResponse`/`webspacesResponse`/`sourceRefreshResponse` exactly.

**Route mounting pattern** (`kernel/httpapi/routes.go:41-57`):
```go
r.Get("/api/sources", SourcesHandler(store, prober))
r.Post("/api/sources/{name}/refresh", SourceRefreshHandler(cfg, refresher))
r.Post("/api/sync", SyncRefreshHandler(refresher))
```
Add the new config routes in this same block, same verb-first style: `r.Get("/api/config", ...)`, `r.Post("/api/config/describe-plugin", ...)`, `r.Post("/api/config/save", ...)`, `r.Post("/api/config/reload", ...)`. Note `Router`'s signature will need a `*config.Store` param threaded through in place of (or alongside) the current fixed `*config.Config` — every existing call site (`WebspacesHandler(store, cfg)`, `StreamHandler(store, cfg)`, etc.) captures `cfg` as a fixed pointer today (`kernel/httpapi/routes.go:41-57`); this is Pitfall 1 from RESEARCH.md — plan the `Store`-threading change explicitly, don't patch around it.

---

### FTS filter-stack query builder (D-16/D-18) — `kernel/index/store.go` + 3 call sites

**Analog:** `kernel/index/store.go`'s existing `ftsQuery` helper (verified ~lines 375-390) and `kernel/httpapi/webspaces.go`'s `aggregateSyncStatus` (lines 195-251) for the "one shared helper, called identically from every consuming handler" shape.

**Existing `ftsQuery` behavior to build on (verified, phrase-quote + AND + trailing prefix-match):**
```go
fields := strings.Fields(raw)
// ... each field phrase-quoted: `"` + f + `"` ...
// ... final field also gets a trailing `*` for prefix/as-you-type match ...
return strings.Join(kept, " ") // FTS5 implicit AND between space-separated MATCH terms
```

**New shared builder (ready-to-use composition from RESEARCH.md):**
```go
func buildMatchQuery(filterTerms []string, liveQuery string) string {
	var parts []string
	for _, term := range filterTerms {
		term = strings.ReplaceAll(term, `"`, "")
		if term == "" {
			continue
		}
		parts = append(parts, `"`+term+`"`) // no trailing * — exact phrase, not prefix
	}
	if live := ftsQuery(liveQuery); live != "" {
		parts = append(parts, live)
	}
	return strings.Join(parts, " ")
}
```
Call this identically from `StreamHandler` (`kernel/httpapi/stream.go`), `SearchHandler` (`kernel/httpapi/search.go`), and the agent-surface stream handler (`kernel/httpapi/agent.go`) — never three separate implementations (D-16's "filtered view IS the webspace for every consumer").

---

### `kernel/index/store.go` — `DeleteSourceItems` (D-07, Pitfall 5)

**Analog:** `kernel/index/store.go`'s own `ReplaceWebspaceSourceItems` (~line 191) — nearest existing method touching rows scoped to `(webspace, source)` or `source` alone; this is a delete-only variant of that same scoping idea.

**Schema precedent already in place** (`kernel/index/schema.go:47-51`, `:99-101` — cascades and FTS trigger already correct, no new trigger needed):
```
item_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE   -- webspace_items
-- items_ad trigger already keeps items_fts in sync on any items delete
```
New method should follow the same signature/error-wrap style as `ReplaceWebspaceSourceItems` and `UpsertItems` (`ctx context.Context` first param, `%w`-wrapped SQL errors prefixed with the method's own concern):
```go
func (s *Store) DeleteSourceItems(ctx context.Context, source string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM items WHERE source = ?`, source)
	if err != nil {
		return fmt.Errorf("index: delete items for source %q: %w", source, err)
	}
	return nil
}
```
(Confirm the exact `s.db`/transaction-wrapping convention against the neighboring method bodies before finalizing — read `ReplaceWebspaceSourceItems`'s full body during planning, not just its signature.)

---

### `kernel/pluginhost/host.go` — `DescribePluginType` (D-11 step 1→2, Pattern 3)

**Analog:** itself — `launch` (lines 134-220) is reused verbatim/wrapped, not duplicated.

**Existing `launch` signature and Describe-call shape to wrap** (lines 134, 194-198):
```go
func launch(ctx context.Context, pluginsDir, name string, src config.Source, logger hclog.Logger) (*Plugin, error) {
	// ... binary stat, sourceConfig JSON marshal, exec.Command, goplugin.NewClient, Dispense ...
	desc, err := impl.Describe(ctx, &toposv1.DescribeRequest{})
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("describe plugin %s: %w", name, err)
	}
	// ...
}
```

**New wrapper (ready-to-use composition):**
```go
func DescribePluginType(ctx context.Context, pluginsDir string, src config.Source, logger hclog.Logger) (DescribeInfo, error) {
	p, err := launch(ctx, pluginsDir, "__trial__", src, logger)
	if err != nil {
		return DescribeInfo{}, fmt.Errorf("pluginhost: trial-launch for describe: %w", err)
	}
	defer p.Kill()
	return DescribeInfo{
		SourceType:      p.SourceType(),
		DisplayName:     p.PluginDisplayName(),
		MatchVocabulary: p.MatchVocabulary(),
	}, nil
}
```
**Verify before relying on this uniformly (Pitfall 4):** confirmed only for `plugins/signal/main.go` that startup defers connectivity checks past `Describe`. Read `plugins/paperless/main.go`, `plugins/silverbullet/main.go`, `plugins/proton/main.go`'s `main()` before assuming this trial-launch works for all four non-Signal types.

**Error-wrap/doc-comment convention throughout this file:** every exported method has a multi-line doc comment naming the design rationale (D-08, D-09, T-01-07 etc. cross-refs) — match this density for `DescribePluginType` and `Reconcile`.

---

### `web/src/lib/components/FilterChip.svelte` (NEW) — D-19

**Analog:** `web/src/lib/components/SourceChip.svelte` (full file, 127 lines) — near-identical shape (health-dot-less pill with label + trailing icon-button), distinct styling and an `×`-remove instead of a refresh icon.

**Structure to copy (imports + chip shell + trailing icon button, lines 1-11, 74-126):**
```svelte
<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { cn } from '$lib/utils.js';
	// FilterChip needs no Tooltip/health-tone import — it has no health
	// state; keep only what's structurally needed, don't copy tooltip
	// wiring that doesn't apply.
</script>

<div class={cn('group flex h-11 shrink-0 items-center rounded-full border border-border bg-card pr-1', /* distinct filter styling per D-19 */)}>
	<span class="truncate text-[14px] leading-[1.4] pl-2.5">{term}</span>
	<Button variant="ghost" size="icon" class="size-8 rounded-full" aria-label={`Remove filter ${term}`} onclick={handleRemoveClick}>
		<X class="size-4" />
	</Button>
</div>
```
Use `stopPropagation` on the remove click exactly as `SourceChip.svelte:68-71` does on its refresh click — the pattern of "sibling control inside a chip must not also trigger the chip's own click semantics" is the same concern here (chip has no click-toggle semantics itself, but the convention of isolating the icon-button's click handler is still correct to follow for consistency/future-proofing).

**Distinct styling requirement (D-19):** must be "visually distinct from source chips" — do not reuse `SourceChip`'s exact tone-dot/border treatment; pick a different border/background token (e.g. accent color) so filter chips read as a different affordance class at a glance.

---

### `web/src/lib/components/ui/dialog/`, `ui/dropdown-menu/` (NEW)

**Analog:** `web/src/lib/components/ui/popover/` (existing bits-ui wrapper — read its files directly during Task 1 before scaffolding; they establish the house style for prop-forwarding and `child({ props })` snippet usage already visible in `WebspaceHeader.svelte:200-217` and `SourceChip.svelte:82-108`).

**Snippet/child-prop forwarding convention already established** (`WebspaceHeader.svelte:201-217`, `SourceChip.svelte:82-108`):
```svelte
<PopoverTrigger>
	{#snippet child({ props })}
		<button {...props} type="button" aria-label="..." class="...">
			...
		</button>
	{/snippet}
</PopoverTrigger>
```
New `Dialog`/`DropdownMenu` wrapper components must expose the same `child({ props })` snippet pattern for their `Trigger` so callers (`AddSourceModal.svelte` etc.) can render a custom trigger button exactly like the existing overflow-chip trigger does — don't invent a different trigger-composition API.

**Pitfall 6 (styling drift):** running `npx shadcn-svelte add dialog dropdown-menu` will not match this repo's already-drifted `components.json` (documented precedent in STATE.md). Use CLI output only for component-API shape (`Dialog.Root`/`Trigger`/`Content`); hand-adapt every color/class to `app.css`'s existing dark-theme tokens, exactly as `popover`/`tooltip` were.

---

### `web/src/lib/components/WebspaceHeader.svelte` (MODIFIED) — switcher + "+" + filter row (D-10/D-11/D-19)

**Analog:** itself — extend in place; this is the single file nearly every UI decision in this phase lands in (per RESEARCH.md's Recommended Project Structure).

**Existing title render to replace with `WebspaceSwitcher`** (line 176-178):
```svelte
<h1 class="truncate text-[28px] leading-[1.2] font-semibold text-foreground" title={webspace}>
	{webspace}
</h1>
```
Becomes a `<WebspaceSwitcher {webspace} .../>` invocation — the switcher owns its own drop-down/list-webspaces/+  affordance internally (D-10), following `+page.svelte`'s existing `listWebspaces()` load pattern for its data source.

**Existing chip-row structure to extend with a trailing "+"** (lines 189-197, and the overflow popover at 199-231): the "+" per D-11 is a sibling appended after the last visible chip / before the overflow trigger, using the same `Button variant="ghost" size="icon"` shape already used elsewhere in this file (e.g. the `X`/clear-filters button at lines 243-246) — do not invent a new button-sizing convention.

**Filter-chip row (D-19):** add as a new row (or appended within the existing chip row, visually distinct per `FilterChip.svelte`'s distinct styling) — follow the exact `{#each ... as x (x.key)}` keyed-each convention already used for `visibleSources`/`hiddenSources` (lines 190, 221).

---

### `web/src/lib/api.ts` (MODIFIED) — new config/describe/save/reload client functions

**Analog:** itself — `getJSON`/`postJSON` (lines 131-171) plus the existing typed-function-per-route convention (`getSources`, `refreshSource`, `refreshAll`, lines 259-272).

**Pattern to copy exactly:**
```ts
/** GET /api/sources */
export function getSources(): Promise<SourcesResponse> {
	return getJSON<SourcesResponse>('/api/sources');
}

/** POST /api/sources/{name}/refresh */
export function refreshSource(name: string): Promise<SourceRefreshResponse> {
	return postJSON<SourceRefreshResponse>(`/api/sources/${encodeURIComponent(name)}/refresh`);
}
```
New functions (`getConfig`, `describePlugin`, `saveConfig`, `reloadConfig`) must follow this exact shape: one-line JSDoc naming the HTTP verb+path, `encodeURIComponent` on every path segment, return type is always the matching response-envelope interface. **`postJSON` currently takes no body** (`postJSON<T>(path: string)`, line 155) — the save/describe-plugin endpoints need a body-carrying POST; extend `postJSON` to accept an optional `body: unknown` param (`JSON.stringify` + `Content-Type: application/json` header) rather than writing a parallel `postJSONWithBody` — keep one POST helper, matching this file's existing "one shared error-envelope helper" discipline (`getJSON`/`postJSON` already duplicate the same error-handling block verbatim at lines 131-146 and 155-170 — note this existing duplication as a candidate to collapse into one shared error-handling function when touching this file, though not required by this phase).

**New response-envelope interfaces** must follow the existing style exactly: `schema_version: number` first field, JSDoc-free (interfaces here carry field-level comments only where non-obvious, e.g. `StreamItem.source_display_name` at lines 47-51) — match that comment density, not more.

---

### `web/src/routes/+page.svelte` (MODIFIED → redirect-only, D-10)

**Analog:** itself — the existing `onMount` + `listWebspaces()` load (lines 1-20) is retained but its render branch changes: instead of rendering a card list, it redirects (via SvelteKit's `goto`) to the first/last-visited webspace, or renders the empty-state creation flow when `webspaces.length === 0` (the existing empty-state branch at lines 41-44 is the closest precedent for "zero webspaces" copy/tone — match its plain-text, muted-foreground style, just change the CTA from "edit config.toml" to an inline "+ Create webspace" affordance per D-10).

## Shared Patterns

### JSON error envelope (all new HTTP handlers)
**Source:** `kernel/httpapi/routes.go:82-108`
**Apply to:** every handler in `kernel/httpapi/config.go`
```go
func WriteJSON(w http.ResponseWriter, status int, v interface{}) { ... }
func WriteError(w http.ResponseWriter, status int, code, message string) { ... }
```
Never hand-roll `json.NewEncoder(w).Encode(...)` directly in a new handler — always go through these two functions, matching every existing handler in the package.

### Sorted-iteration determinism (all new config validation/serialization code)
**Source:** `kernel/config/config.go` (`validateWebspaces`, `validateDisplayNameUniqueness`, etc. — every map iteration in this file sorts keys first via `sort.Strings`)
**Apply to:** any new code iterating `cfg.Sources`/`cfg.Webspaces`/`Webspace.Match` — never iterate a Go map directly when the output (error message order, JSON array order, TOML key order) is observable; `go-toml/v2`'s `Marshal` already sorts map keys on its own (verified), but any Go-side loop building a response or log line must sort explicitly, matching this file's existing discipline throughout.

### Frontend typed-fetch-per-route (all new api.ts functions)
**Source:** `web/src/lib/api.ts:131-272`
**Apply to:** every new config-related client function
One-line JSDoc + `getJSON<T>`/`postJSON<T>` + `encodeURIComponent` on path params — no direct `fetch()` calls anywhere outside this file.

### bits-ui primitive wrapper + `child({ props })` trigger snippet (all new modal/dropdown components)
**Source:** `web/src/lib/components/ui/popover/` + its usage in `WebspaceHeader.svelte:200-217`, `SourceChip.svelte:82-108`
**Apply to:** `ui/dialog/`, `ui/dropdown-menu/`, and every component that opens one (`AddSourceModal`, `EditSourceModal`, `ManageSourcesModal`, `WebspaceSwitcher`)

### Doc-comment density (all new Go exported symbols)
**Source:** every file in `kernel/config/`, `kernel/pluginhost/`, `kernel/httpapi/` — multi-sentence rationale comments citing decision IDs (D-XX) are the house style, not optional polish
**Apply to:** every new exported type/function in this phase's Go changes

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `kernel/config/store.go` (`atomic.Pointer[Config]` swap mechanism) | service/state-holder | event-driven | No `sync/atomic` or `sync.Mutex` usage exists anywhere in `kernel/` today (confirmed via grep in RESEARCH.md) — this is genuinely new infrastructure; use RESEARCH.md's Pattern 1/Code Examples as the design starting point instead of a codebase analog |
| `kernel/syncer/scheduler.go` restart-on-apply mechanism | service | event-driven | `Scheduler.Run` (lines 31-58) spawns fixed goroutines once from a boot-time snapshot with no add/remove capability today — the "cancel old scheduler's context, build a fresh one against the new config" mechanism has no existing precedent in this codebase; RESEARCH.md's Open Question 2 recommends the concrete approach (cancel immediately, rely on `syncOne`'s existing detached-finalize behavior, same as kernel shutdown) |

## Metadata

**Analog search scope:** `kernel/config/`, `kernel/httpapi/`, `kernel/pluginhost/`, `kernel/index/`, `kernel/syncer/`, `web/src/lib/components/`, `web/src/lib/components/ui/`, `web/src/routes/`, `web/src/lib/api.ts`
**Files scanned:** ~20 (all files named in CONTEXT.md's Integration Points + RESEARCH.md's Recommended Project Structure)
**Pattern extraction date:** 2026-08-07
