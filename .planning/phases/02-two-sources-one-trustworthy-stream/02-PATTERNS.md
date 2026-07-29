# Phase 2: Two Sources, One Trustworthy Stream - Pattern Map

**Mapped:** 2026-07-28
**Files analyzed:** 24
**Analogs found:** 22 / 24

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `plugins/silverbullet/main.go` | config/entrypoint | request-response | `plugins/paperless/main.go` | exact |
| `plugins/silverbullet/client.go` | service (HTTP client) | request-response | `plugins/paperless/client.go` | exact |
| `plugins/silverbullet/plugin.go` | controller (gRPC adapter) | request-response | `plugins/paperless/plugin.go` | exact |
| `plugins/silverbullet/frontmatter.go` | utility (transform) | transform | none (new); `adrg/frontmatter` per RESEARCH.md | no analog |
| `plugins/silverbullet/render.go` | utility (transform) | transform | none (new); goldmark+bluemonday per RESEARCH.md | no analog |
| `plugins/silverbullet/client_test.go` | test | request-response | `plugins/paperless/client_test.go` | exact |
| `plugins/silverbullet/outbound_hosts_test.go` | test | request-response | `plugins/paperless/outbound_hosts_test.go` | exact |
| `plugins/silverbullet/readonly_test.go` | test | request-response | `plugins/paperless/readonly_test.go` | exact |
| `plugins/mock/main.go` | config/entrypoint | request-response | `plugins/paperless/main.go` (structure only, trivial impl) | role-match |
| `plugins/mock/plugin.go` | controller (gRPC adapter) | request-response | `plugins/paperless/plugin.go` (structure only, no real client) | role-match |
| `kernel/sync/coordinator.go` | service (coordinator) | event-driven | `kernel/correlate/correlate.go` (SyncAll orchestration) | role-match |
| `kernel/sync/scheduler.go` | service (scheduler) | event-driven | `cmd/webspaces/main.go` `runServe`'s startup-sync goroutine | role-match |
| `kernel/correlate/correlate.go` (MODIFIED) | service (orchestration) | CRUD/event-driven | itself (Phase 1) — restructure source-major | exact (self) |
| `kernel/index/store.go` (MODIFIED: `ReplaceWebspaceSourceItems`, `StartSyncRun`/`FinishSyncRun`) | model/store (CRUD) | CRUD | `Store.ReplaceWebspaceItems` / `Store.RecordSyncRun` (same file) | exact (self) |
| `kernel/index/schema.go` (MODIFIED if needed) | model (schema) | CRUD | itself (Phase 1 `sync_runs` table) | exact (self) |
| `kernel/httpapi/sources.go` | controller (HTTP route) | request-response | `kernel/httpapi/stream.go` (`StreamHandler`) | exact |
| `kernel/httpapi/agent/*.go` | controller (HTTP route, grant-filtered) | request-response | `kernel/httpapi/routes.go` + `kernel/httpapi/stream.go` (mirrored, with grant filter) | role-match |
| `kernel/httpapi/routes.go` (MODIFIED — mount health/refresh/agent routes) | controller (router) | request-response | itself (Phase 1 `Router`) | exact (self) |
| `kernel/httpapi/item.go` (MODIFIED — MIME allowlist +`text/html`) | controller (HTTP route) | request-response | itself (Phase 1 allowlist code) | exact (self) |
| `kernel/config/types.go` (MODIFIED — sync interval, agent grants, SilverBullet source fields) | config (types) | CRUD | itself (Phase 1 `Source`/`Webspace` structs) | exact (self) |
| `kernel/config/config.go` (MODIFIED — validate new fields) | config (loader/validator) | CRUD | itself (Phase 1 `Validate`) | exact (self) |
| `kernel/pluginhost/host.go` (MODIFIED — no new file, Health already wired) | service (subprocess host) | request-response | itself (Phase 1 `Health` method already exists) | exact (self) |
| `cmd/webspaces/main.go` (MODIFIED — route through coordinator) | config/entrypoint | event-driven | itself (Phase 1 `runServe`/`runSync`) | exact (self) |
| `web/src/lib/components/WebspaceHeader.svelte` (MODIFIED — health chips + filter chips) | component | request-response | itself (Phase 1, currently just a title) | exact (self) |
| `web/src/lib/components/SourceHealthChip.svelte` (NEW) | component | request-response | `web/src/lib/components/StreamRow.svelte` badge usage + `ui/badge`, `ui/tooltip` | role-match |
| `web/src/lib/components/SourceFilterChips.svelte` (NEW) | component | request-response | `WebspaceHeader.svelte` (props/URL pattern) + `ui/badge` | role-match |
| `web/src/lib/components/StreamList.svelte` (MODIFIED — per-item stale badge) | component | request-response | itself (Phase 1 sync-failure/empty/populated branches) | exact (self) |
| `web/src/lib/components/DetailPane.svelte` (MODIFIED — stale/unavailable alert branch, text/html iframe branch, parameterized copy) | component | request-response | itself (Phase 1 two-stage render + PDF iframe branch) | exact (self) |
| `web/src/lib/api.ts` (MODIFIED — `getSources`, `refreshSource`, `refreshAll` typed calls) | utility (API client) | request-response | itself (Phase 1 `getJSON`/`listWebspaces`/`getStream` pattern) | exact (self) |
| `web/src/routes/w/[webspace]/+page.svelte` (MODIFIED — URL `?source=` filter state) | route/page | request-response | itself (Phase 1 page, per RESEARCH.md Code Example) | exact (self) |
| `config.example.toml` (MODIFIED) | config | — | itself | exact (self) |
| `docs/api.md` (MODIFIED) | docs | — | itself | exact (self) |
| `docs/plugin-contract.md` (MODIFIED) | docs | — | itself | exact (self) |

## Pattern Assignments

### `plugins/silverbullet/plugin.go` (controller, request-response)

**Analog:** `plugins/paperless/plugin.go`

**Imports pattern** (lines 1-16):
```go
package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	webspacesv1 "github.com/davison/webspaces/sdk/gen/webspaces/v1"
)
```
For SilverBullet, add `bytes`, `github.com/yuin/goldmark`, `github.com/microcosm-cc/bluemonday`, `github.com/adrg/frontmatter` imports; drop `strconv` (source_id is a path string, not numeric).

**Describe pattern** (lines 42-48) — copy verbatim shape, swap `sourceType`/`displayName` constants:
```go
func (p *SourcePlugin) Describe(_ context.Context, _ *webspacesv1.DescribeRequest) (*webspacesv1.DescribeResponse, error) {
	return &webspacesv1.DescribeResponse{
		SourceType:      sourceType,
		DisplayName:     displayName,
		ContractVersion: contractVersion,
	}, nil
}
```

**Match pattern** (lines 50-75) — same shape (list → filter/resolve → build items), but SilverBullet's "resolve" step is per-page frontmatter/name matching (D-03) rather than a tag-ID API call (see Pitfall 2 in RESEARCH.md — fetch every candidate page body, bounded concurrency).

**Item construction / deep link / fidelity** (lines 77-106) — copy the `Provenance` map shape and `Fidelity: webspacesv1.LinkFidelity_LINK_FIDELITY_EXACT` (D-01 requires exact fidelity); deep link built as `fmt.Sprintf("%s/%s", baseURL, sourceID)` (matches D-01 exactly, no `/documents/` segment).

**Fetch / two-field response pattern** (lines 132-181, and RESEARCH.md's `fetchFull` example at lines 320-350) — mirrors `fetchFull`'s not-found/unavailable error mapping:
```go
func (p *SourcePlugin) fetchFull(ctx context.Context, path string) (*webspacesv1.FetchResponse, error) {
	raw, err := p.client.ReadFile(ctx, path)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "silverbullet: page %q not found", path)
		}
		return nil, status.Errorf(codes.Unavailable, "silverbullet: read %q: %v", path, err)
	}
	// ... goldmark render + bluemonday sanitize, return MimeType: "text/html"
}
```

**Health pattern** (lines 203-215) — copy verbatim shape, swap the underlying reachability probe (a lightweight `ListFiles` call instead of `AllTags`):
```go
func (p *SourcePlugin) Health(ctx context.Context, _ *webspacesv1.HealthRequest) (*webspacesv1.HealthResponse, error) {
	_, err := p.client.ListFiles(ctx)
	if err != nil {
		return &webspacesv1.HealthResponse{Reachable: false, LastError: err.Error()}, nil
	}
	return &webspacesv1.HealthResponse{Reachable: true, LastSyncUnix: time.Now().Unix()}, nil
}
```

---

### `plugins/silverbullet/client.go` (service, request-response)

**Analog:** `plugins/paperless/client.go`

**Host-pinned transport construction** (lines 51-120) — copy `Client` struct shape (`baseURL`, `baseHost`, `token`, `http *http.Client`), `NewClient`'s `DialContext`/`CheckRedirect` wiring verbatim; only the `apiVersion` field and its two Accept-header uses are dropped (SilverBullet has no API-version header).

**`allowHost` predicate** (lines 122-151) — copy verbatim, unchanged (host-pinning logic is source-agnostic):
```go
func (c *Client) allowHost(hostport string) error {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	host = strings.ToLower(strings.Trim(host, "[]"))
	if host != "" && c.baseHost != "" && host == c.baseHost {
		return nil
	}
	if host == "localhost" {
		return nil
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return nil
	}
	return fmt.Errorf("%w: %q", ErrForeignHost, host)
}
```

**Auth header pattern** (lines 302, 348, 409) — same `Authorization` header shape, swap scheme:
```go
req.Header.Set("Authorization", "Bearer "+c.token) // SilverBullet's SB_AUTH_TOKEN, vs paperless's "Token "+c.token
```

**`getJSON` shared GET+decode helper** (lines 399-428) — copy verbatim structure for `ListFiles` (`GET /.fs`) and add a raw-bytes variant (`ReadFile`, `GET /.fs/{path}`) since page bodies are markdown text, not JSON — model the raw variant on `rendition()` (lines 343-368) instead, since that already handles a non-JSON byte-body GET with 404-as-non-error semantics... except SilverBullet's `ReadFile` 404 IS an error (`ErrNotFound`), unlike paperless's renditions — follow `Document`'s not-found mapping (lines 295-323) for `ReadFile` instead.

**Pagination note:** SilverBullet's `/.fs` has no pagination/`next` field (RESEARCH.md A1) — omit `splitNextURL`/`documentsPage`-style loop entirely; `ListFiles` returns the full flat array in one call.

---

### `plugins/silverbullet/outbound_hosts_test.go` / `readonly_test.go` (test)

**Analogs:** `plugins/paperless/outbound_hosts_test.go`, `plugins/paperless/readonly_test.go` — copy both files near-verbatim. `readonly_test.go`'s `pluginsRoot = ".."` AST walk already covers every file under `plugins/` including the new `plugins/silverbullet/` and `plugins/mock/` directories automatically — no changes needed to that file itself, only add the SilverBullet-specific `outbound_hosts_test.go` (swap `paperless.lan`-style fixture host for the SilverBullet base URL) per the host-pinning table pattern at lines 19-54.

---

### `plugins/mock/plugin.go` (controller, request-response — PLUG-05 artifact)

**Analog:** `plugins/paperless/plugin.go`'s method signatures (`Describe`/`Match`/`Fetch`/`Health`), but the body of each method returns a fixed, hand-written `[]*webspacesv1.Item` / deterministic `HealthResponse` with no HTTP client at all (RESEARCH.md "PLUG-05 validation via isolated mock plugin"). Do not give this file read access to `plugins/paperless/` or `plugins/silverbullet/` during the actual validation exercise — the exercise's fresh-context builder should reconstruct the shape from `proto/webspaces/v1/plugin.proto` + `docs/plugin-contract.md` + the `sdk` module alone.

---

### `kernel/sync/coordinator.go` (service, event-driven)

**Analog:** `kernel/correlate/correlate.go`'s `Engine.SyncAll` — restructure per the Critical Architecture Finding in RESEARCH.md (source-major loop, `ReplaceWebspaceSourceItems`).

**Coordinator wraps the corrected per-source sync in `singleflight.Group.Do`:**
```go
// kernel/sync/coordinator.go
type Coordinator struct {
	group   singleflight.Group
	Store   *index.Store
	Sources []correlate.Source
	Config  *config.Config
}

func (c *Coordinator) Refresh(ctx context.Context, sourceName string) (RunResult, error) {
	v, err, _ := c.group.Do(sourceName, func() (interface{}, error) {
		return c.syncOne(ctx, sourceName)
	})
	// ...
}
```
Both the ticker (`scheduler.go`) and the manual-refresh HTTP handler (`sources.go`) MUST call this same `Coordinator.Refresh` entry point — see Security Domain's "Sync-race double-run" threat pattern in RESEARCH.md.

**Two-phase sync_runs write** (extends `RecordSyncRun`, `kernel/index/store.go` lines 314-324):
```go
func (s *Store) StartSyncRun(ctx context.Context, sourceType string) (runID int64, err error) {
	res, err := s.db.ExecContext(ctx, `
INSERT INTO sync_runs (source_type, started_unix, status) VALUES (?, unixepoch(), 'running')
`, sourceType)
	// ...
}

func (s *Store) FinishSyncRun(ctx context.Context, runID int64, status, errMsg string, itemCount int) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE sync_runs SET finished_unix = unixepoch(), status = ?, error = ?, item_count = ? WHERE id = ?
`, status, errMsg, itemCount, runID)
	return err
}
```

---

### `kernel/index/store.go` — `ReplaceWebspaceSourceItems` (model, CRUD)

**Analog:** `Store.ReplaceWebspaceItems` (same file, lines 74-118) — copy the transaction shape (`upsertItemsTx` then delete-then-reinsert then commit), but scope the delete to `(webspace_name, source_type)` per RESEARCH.md's Critical Architecture Finding:
```go
func (s *Store) ReplaceWebspaceSourceItems(ctx context.Context, webspaceName, sourceType string, items []item.Item) error {
	tx, err := s.db.BeginTx(ctx, nil)
	// ... upsertItemsTx(ctx, tx, items) — unchanged
	if _, err := tx.ExecContext(ctx, `
DELETE FROM webspace_items
WHERE webspace_name = ?
  AND item_id IN (SELECT id FROM items WHERE source_type = ?)
`, webspaceName, sourceType); err != nil { /* ... */ }
	// ... insert new rows, mark webspaces.synced_unix (ON CONFLICT DO UPDATE, unchanged), commit
}
```
`webspaces.synced_unix` upsert (lines 107-112) — copy verbatim; RESEARCH.md notes it must fire on ANY source's first successful contribution, which this per-source method already satisfies naturally (each source's sync calls it independently).

---

### `kernel/httpapi/sources.go` (controller, request-response)

**Analog:** `kernel/httpapi/stream.go`'s `StreamHandler` (handler shape, envelope construction) + `kernel/httpapi/routes.go`'s error/JSON helpers.

**Envelope + handler shape** (mirrors `stream.go` lines 41-89):
```go
type sourceStatus struct {
	Name         string `json:"name"`
	SourceType   string `json:"source_type"`
	DisplayName  string `json:"display_name"`
	Reachable    bool   `json:"reachable"`
	Syncing      bool   `json:"syncing"`
	LastSyncUnix int64  `json:"last_sync_unix"`
	LastError    string `json:"last_error"`
}

type sourcesResponse struct {
	SchemaVersion int            `json:"schema_version"`
	Sources       []sourceStatus `json:"sources"`
}

func SourcesHandler(store *index.Store, host *pluginhost.Host) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// live Health() call per plugin (pluginhost.Plugin.Health, host.go
		// lines 66-68) merged with store-side sync_runs history —
		// see RESEARCH.md "Health merge" example response shape.
	}
}
```

**Refresh route validation pattern** — mirrors `StreamHandler`'s `WebspaceExists`-then-404 shape (lines 57-65): validate `{name}` against the configured source-name set before dispatch, 404 in the same shape as any other not-found route (Security Domain V5).

**Shared JSON/error helpers** (`kernel/httpapi/routes.go` lines 61-87) — reuse `WriteJSON`/`WriteError` verbatim, no new envelope shape needed.

---

### `kernel/httpapi/agent/*.go` (controller, request-response — grant-filtered mirror)

**Analog:** `kernel/httpapi/routes.go`'s `Router` (mount-point pattern) + `kernel/httpapi/stream.go`'s `StreamHandler`/`toStreamItem` — the agent namespace mirrors these exactly, with one added filtering step per D-12: before calling `store.StreamItems`/`store.GetItem`, check the source's `agent.read` grant from `cfg.Sources[name].Agent.Read`; an ungranted source's items are silently absent (no distinct error), and an ungranted direct item/content request returns the identical `item_not_found` shape `ItemHandler` already returns for a genuinely nonexistent item (`kernel/httpapi/item.go` — read for the not-found error shape before implementing this).

**Existence-leak avoidance** (Security Domain table, RESEARCH.md) — copy `WriteError(w, http.StatusNotFound, "item_not_found", ...)` verbatim for both the "genuinely doesn't exist" and "exists but ungranted" cases — never a distinct code for the latter.

---

### `kernel/config/types.go` (config, CRUD)

**Analog:** itself — extend the existing `Source` struct (lines 30-37) and `Config` struct (lines 6-13) in place:
```go
type Source struct {
	Plugin     string      `toml:"plugin"`
	BaseURL    string      `toml:"base_url"`
	Token      string      `toml:"token"`
	APIVersion string      `toml:"api_version"`
	SyncInterval string    `toml:"sync_interval,omitempty"` // per-source override of [sync].interval (D-05)
	Agent      AgentGrant  `toml:"agent"`                    // D-11
}

type AgentGrant struct {
	Read    bool `toml:"read"`
	Handoff bool `toml:"handoff"`
}

type SyncConfig struct {
	Interval string `toml:"interval"` // default "15m" (D-05)
}
```
Follow the existing `DefaultListen`/`DefaultIndexPath` const-default pattern (types.go lines 45-49) for a new `DefaultSyncInterval = "15m"`.

**Validation pattern** — extend `Validate` (`config.go` lines 100-125) with the same per-field empty/missing-env-var check style already used for `BaseURL`/`Token` (lines 116-122); absent `[sources.*.agent]` block or absent key defaults to `false` structurally (Go zero-value `bool` — no special-case code needed) per D-11.

---

## Shared Patterns

### Read-only host-pinned HTTP client
**Source:** `plugins/paperless/client.go` lines 51-151 (full `NewClient` + `allowHost`)
**Apply to:** `plugins/silverbullet/client.go` — copy the `DialContext`/`CheckRedirect`/`allowHost` triad verbatim; only the auth-header scheme and JSON-vs-raw-body decode differ.

### Plugin gRPC adapter shape (Describe/Match/Fetch/Health)
**Source:** `plugins/paperless/plugin.go` (whole file)
**Apply to:** `plugins/silverbullet/plugin.go`, `plugins/mock/plugin.go` — same four-method shape, same `status.Errorf(codes.X, "sourcetype: ...")` error-wrapping convention throughout.

### JSON envelope + error shape
**Source:** `kernel/httpapi/routes.go` lines 61-87 (`WriteJSON`, `WriteError`, `errorEnvelope`)
**Apply to:** all new kernel HTTP handlers (`sources.go`, `agent/*.go`) — reuse verbatim, never hand-roll a new envelope shape.

### Read-then-persist source-scoped transaction
**Source:** `kernel/index/store.go` lines 79-118 (`ReplaceWebspaceItems`)
**Apply to:** `ReplaceWebspaceSourceItems` — same upsert-then-delete-then-reinsert-then-commit transaction shape, source-scoped delete instead of whole-webspace delete.

### Two-stage detail pane render (metadata sync, content async)
**Source:** `web/src/lib/components/DetailPane.svelte` lines 51-104
**Apply to:** the same file's modification — add the `content_unavailable`/stale-source alert branch and the `text/html` iframe branch alongside the existing `application/pdf`/`image/*` branches (line 84-93), following the exact `{:else if content.available && content.rendition?.mime_type === '...'}` chain already established. Never use `{@html}` (RESEARCH.md's explicit anti-pattern) — the sanitized markdown must render inside the same `<iframe src={contentUrl(item.id)}>` pattern already used for PDFs (line 85).

### Sync-failure-before-empty branch ordering
**Source:** `web/src/lib/components/StreamList.svelte` lines 22-31 (`syncFailed` computed strictly before `isEmpty`)
**Apply to:** any new per-item stale-badge logic added to `StreamList.svelte`/`StreamRow.svelte` — extend, never reorder or bypass, this branch check.

### URL-query-param-as-state (client-side filter)
**Source:** RESEARCH.md's own worked example (Architecture Patterns "Client-side source filter"), modeled on how `web/src/routes/w/[webspace]/+page.svelte` already reads route params
**Apply to:** `SourceFilterChips.svelte` + the `+page.svelte` modification — `page.url.searchParams` read/write via `goto(url, { replaceState: true, keepFocus: true, noScroll: true })`.

### Typed API client function shape
**Source:** `web/src/lib/api.ts` lines 99-140 (`getJSON<T>`, `listWebspaces`, `getStream`)
**Apply to:** new `getSources()`, `refreshSource(name)`, `refreshAll()` — same `getJSON<T>(path)` wrapper (add a `postJSON` for the two refresh calls, mirroring `getJSON`'s error-envelope handling exactly).

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `plugins/silverbullet/frontmatter.go` | utility (transform) | transform | No existing frontmatter/tag-extraction code in the repo; build directly from `adrg/frontmatter`'s documented API (`frontmatter.Parse(io.Reader, &struct) (rest []byte, err error)`) per RESEARCH.md Standard Stack — plus a small hand-rolled inline `#tag` regex scan, which has no prior in-repo analog either. |
| `plugins/silverbullet/render.go` | utility (transform) | transform | No existing markdown-rendering code in the repo; build directly from `goldmark.Convert` + `bluemonday.UGCPolicy().SanitizeBytes` per RESEARCH.md's worked `fetchFull` example (Code Examples section, lines 320-350 of RESEARCH.md). |
| `kernel/sync/scheduler.go` | service (scheduler) | event-driven | Closest precedent is the one-shot goroutine in `cmd/webspaces/main.go`'s `runServe` (lines 162-166) — a single `go func(){ SyncAll() }()`, not a repeating ticker. No existing `time.Ticker` loop in the codebase; build per RESEARCH.md's Standard Stack recommendation (plain `time.Ticker`, one per configured source, calling `Coordinator.Refresh(name)`). |

## Metadata

**Analog search scope:** `plugins/paperless/`, `kernel/correlate/`, `kernel/index/`, `kernel/httpapi/`, `kernel/pluginhost/`, `kernel/config/`, `cmd/webspaces/`, `web/src/lib/components/`, `web/src/lib/api.ts`, `web/src/routes/`
**Files scanned:** ~35 (all Go files in `plugins/`, `kernel/`, `cmd/`; all Svelte/TS files in `web/src/lib` and `web/src/routes`)
**Pattern extraction date:** 2026-07-28
