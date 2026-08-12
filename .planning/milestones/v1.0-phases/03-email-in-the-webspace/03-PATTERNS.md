# Phase 3: Email in the Webspace - Pattern Map

**Mapped:** 2026-07-29
**Files analyzed:** 15 (10 backend, 5 frontend)
**Analogs found:** 15 / 15

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|---------------|
| `plugins/proton/go.mod` | config | — | `plugins/silverbullet/go.mod` | exact |
| `plugins/proton/main.go` | controller (subprocess entrypoint) | request-response (gRPC serve) | `plugins/silverbullet/main.go` | exact |
| `plugins/proton/plugin.go` | controller (SourcePlugin adapter) | CRUD (Match/Fetch/Health) | `plugins/silverbullet/plugin.go` | exact |
| `plugins/proton/client.go` | service (IMAP connection/dialer) | request-response (protocol client) | `plugins/silverbullet/client.go` | role-match (HTTP client → IMAP client, same host-pinning shape) |
| `plugins/proton/mailbox_scan.go` | service (LIST/EXAMINE/FETCH + dedup) | batch/CRUD | `plugins/silverbullet/plugin.go`'s `Match()` (candidate scan + worker pool) | role-match |
| `plugins/proton/body.go` | transform (MIME parse + sanitize) | transform | `plugins/silverbullet/render.go` | exact (same bluemonday + WrapDocument shape) |
| `plugins/proton/outbound_hosts_test.go` | test | — | `plugins/silverbullet/outbound_hosts_test.go` | exact |
| `plugins/proton/readonly_test.go` | test (AST scan) | — | `plugins/paperless/readonly_test.go` | role-match (new pattern: IMAP idents instead of net/http idents) |
| `kernel/config/types.go` (add `Source.Username`) | config (struct field) | — | `kernel/config/types.go`'s existing `CACert` field | exact (same file, same precedent-add pattern) |
| `kernel/pluginhost/host.go` (`launch()`) | service (subprocess launcher) | request-response | same file's existing `sourceConfig` JSON map | exact |
| `kernel/index/schema.go` (add `items_fts` + triggers) | migration | CRUD | same file's existing table/index DDL | exact |
| `kernel/index/store.go` (add `Search()`) | service (index reads) | CRUD (query) | same file's `StreamItems()` | exact |
| `kernel/httpapi/search.go` | controller (route handler) | request-response | `kernel/httpapi/stream.go` | exact |
| `kernel/httpapi/routes.go` (register search route) | route | — | same file's existing `r.Get(...)` registrations | exact |
| `web/src/lib/api.ts` (add `SearchResponse`/`searchWebspace`) | service (typed API client) | request-response | same file's `getStream`/`StreamItem` | exact |
| `web/src/lib/components/StreamRow.svelte` (render `group_label`) | component | CRUD (render) | same file (self-modification) | exact |
| `web/src/lib/components/DetailPane.svelte` (render `group_label`) | component | CRUD (render) | same file (self-modification) | exact |
| `web/src/lib/components/SearchBox.svelte` | component | request-response (debounced query) | `web/src/lib/components/SourceFilterChips.svelte` (closest existing header-tier control) | role-match |

## Pattern Assignments

### `plugins/proton/main.go` (controller, request-response)

**Analog:** `plugins/silverbullet/main.go` (full file, 67 lines — read in full, see above)

**Imports pattern:**
```go
import (
	"encoding/json"
	"fmt"
	"os"

	goplugin "github.com/hashicorp/go-plugin"

	"github.com/davison/webspaces/sdk"
)
```

**Core pattern — env-config decode + goplugin.Serve:**
```go
type sourceConfig struct {
	BaseURL  string `json:"base_url"`
	Username string `json:"username"` // NEW field for IMAP auth — see Pitfall 1
	Token    string `json:"token"`    // reused as the IMAP password
	CACert   string `json:"ca_cert"`
}

func main() {
	raw := os.Getenv("WEBSPACES_SOURCE_CONFIG")
	if raw == "" {
		fatal(fmt.Errorf("WEBSPACES_SOURCE_CONFIG is not set"))
	}
	var cfg sourceConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		fatal(fmt.Errorf("parse WEBSPACES_SOURCE_CONFIG: %w", err))
	}
	// ... required-field checks (BaseURL, Username, Token) ...
	impl := NewSourcePlugin(cfg.BaseURL, cfg.Username, cfg.Token, cfg.CACert)
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: sdk.Handshake,
		Plugins: map[string]goplugin.Plugin{
			"source": &sdk.SourcePluginGRPCPlugin{Impl: impl},
		},
		GRPCServer: sdk.GRPCServer,
	})
}
```

**Error handling pattern:** `fatal()` writes to stderr and `os.Exit(1)` — copy verbatim.

---

### `plugins/proton/plugin.go` (controller, CRUD)

**Analog:** `plugins/silverbullet/plugin.go` (full file, 272 lines — read in full, see above)

**Imports pattern (lines 1-15):**
```go
import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	webspacesv1 "github.com/davison/webspaces/sdk/gen/webspaces/v1"
)
```

**Describe pattern (lines 50-56):** copy verbatim shape — `sourceType`, `displayName` ("Proton Mail" per UI-SPEC), `contractVersion` constants.

**Core Match pattern (lines 74-150):** the errgroup-bounded worker-pool scan is the closest analog to `mailbox_scan.go`'s per-mailbox `EXAMINE` + `FETCH` loop, but the *dedup* shape (Pattern 2 in RESEARCH.md, `byMessageID` map merging `labels`) replaces SilverBullet's per-page independent-match loop — this is the one place email's data flow diverges from the analog and must follow RESEARCH.md's Pattern 2 code example instead.

**Error discrimination pattern (lines 100-126):** the `errors.Is(err, ErrNotFound)` → skip vs. everything-else → fail-the-whole-Match distinction is directly reusable: apply the same discrimination to IMAP-specific errors (e.g. a message deleted between `LIST` and `FETCH`).

**Fetch dispatch pattern (lines 195-204):** copy verbatim — switch on `req.GetVariant()`, `FULL`/`PREVIEW` share one path, `THUMBNAIL` always unavailable with a fixed reason string (email has no thumbnail rendition either).

**Health pattern (lines 259-271):** same shape, but see RESEARCH.md's `Health()` code example for the IMAP-specific timeout-and-login version (own dialer + explicit `client.Timeout`, since no kernel-side timeout wraps this RPC).

**Error handling pattern (throughout):** every plugin-boundary error is wrapped with `status.Errorf(codes.Unavailable, "proton: <op>: %v", err)` or `codes.NotFound` — copy this convention exactly (mirrors `codes.Unavailable`/`codes.NotFound`/`codes.InvalidArgument` usage seen in the analog).

---

### `plugins/proton/client.go` (service, request-response)

**Analog:** `plugins/silverbullet/client.go` (full file, 296 lines — read in full, see above)

**Host-pinning / TLS pattern (lines 83-159 of the analog):** the `NewClient` constructor's `tls.Config{}` + optional `ca_cert` PEM load + host-pinned dialer shape is the direct template — RESEARCH.md's own "Host-pinned imap.Dialer" code example (`pinnedDialer` struct) is this exact pattern re-expressed for `go-imap`'s `Dialer` interface instead of `http.Transport.DialContext`. Copy the `allowHost`-equivalent predicate (lines 171-188) verbatim in shape: strip port, lowercase, allow configured host + `localhost` + loopback IPs, else return a wrapped sentinel error (`ErrForeignHost`-equivalent, e.g. `ErrForeignHost` in the proton package).

**Sentinel error pattern (lines 26-34):**
```go
var ErrNotFound = errors.New("silverbullet: not found")
var ErrForeignHost = errors.New("silverbullet: foreign host refused")
```
Rename prefix to `proton:` in the new file, same shape.

**Never-log-the-secret discipline (comment at line 212):** `// Never log the request object or its headers — the Authorization header carries the bearer token.` — apply the identical discipline to the IMAP password in the new client (never log `Login()`'s error in a way that could echo credentials).

---

### `plugins/proton/body.go` (transform)

**Analog:** `plugins/silverbullet/render.go` (full file, 106 lines — read in full, see above)

**Sanitize + wrap pattern (lines 21-105):**
```go
var sanitizePolicy = bluemonday.UGCPolicy() // adapt per Pitfall 3: narrower AllowAttrs("style") scope, not Globally()

func RenderSanitized(htmlPart []byte) ([]byte, error) {
	return sanitizePolicy.SanitizeBytes(htmlPart), nil
}

const themeStyle = `...` // copy verbatim — same dark-theme CSS constants (#0f172a, #f1f5f9, #60a5fa, etc.)

func WrapDocument(sanitizedFragment []byte) []byte {
	// identical doctype+head+style+body wrap — copy verbatim
}
```
This is a near-exact copy: reuse `themeStyle` and `WrapDocument` byte-for-byte (same iframe/CSP rendering path per RESEARCH.md's Architectural Responsibility Map), and only replace `goldmark`'s markdown→HTML conversion step with the MIME-part extraction shown in RESEARCH.md's "Fetch() body extraction" code example (`mail.CreateReader`/`NextPart()` loop). Do NOT copy `goldmark.New()` — email bodies are already HTML.

---

### `plugins/proton/outbound_hosts_test.go` (test)

**Analog:** `plugins/silverbullet/outbound_hosts_test.go` (full file, 141 lines — read in full, see above)

**Core pattern (lines 18-53):** `TestAllowHost_PredicateTable` with a `permit`/`refuse` string-slice table, asserting `errors.Is(err, ErrForeignHost)` on every refusal — copy this test shape verbatim against the new IMAP dialer's `allowHost`-equivalent method. The redirect-following tests (lines 55-141) don't apply to IMAP (no HTTP redirects) — omit those, keep only the predicate-table test.

---

### `plugins/proton/readonly_test.go` (test, NEW pattern)

**Analog:** `plugins/paperless/readonly_test.go` (full file, 146 lines — read in full, see above)

**AST-scan pattern (lines 44-145):** `TestPluginsIssueOnlyGetRequests` walks the Go AST of every `.go` file under `plugins/` and flags disallowed `net/http` identifiers. RESEARCH.md's "Don't Hand-Roll" table explicitly calls for a **new**, IMAP-specific sibling test using the identical AST-walk mechanism (`ast.Inspect` + `*ast.SelectorExpr` check) but with a different disallowed-identifier map:
```go
var disallowedIMAPIdents = map[string]bool{
	"Store": true, "Expunge": true, "Move": true,
	"Append": true, "Delete": true, "Copy": true,
}
```
Mirror `scanFileForNonGET`'s structure (`ast.Inspect` walking `*ast.SelectorExpr`, checking `expr.X.(*ast.Ident).Name == "c"` or the imap client variable/package alias) — this is a new test file, not a copy, but the walking mechanism and failure-message format (`fmt.Sprintf("%s: ...", fset.Position(...))`) should be copied verbatim.

---

### `kernel/config/types.go` (config, add `Source.Username`)

**Analog:** same file — the existing `CACert` field (lines 45-56) is the precedent for adding a new optional/required `Source` field.

**Pattern to copy:**
```go
// Username is the IMAP login username for a source using SRC-01's email
// plugin — added beyond Phase 1/2's original Source fields (deviation,
// Rule 2: SilverBullet/paperless use bearer-token auth, IMAP needs
// username+password). Token is reused as the IMAP password.
Username string `toml:"username,omitempty"`
```
Add the doc-comment explaining *why* this field exists, matching `CACert`'s own comment style (explains the concrete deployment reason a generic field was needed).

---

### `kernel/pluginhost/host.go` (`launch()`, service)

**Analog:** same file, lines 108-125 (the existing `sourceConfig` JSON map) — READ THIS EXACT RANGE, already in context via Bash grep above; re-read only if line numbers shift.

**Pattern to copy (CRITICAL — Pitfall 1 in RESEARCH.md):**
```go
sourceConfig, err := json.Marshal(map[string]string{
	"base_url":    src.BaseURL,
	"username":    src.Username, // NEW — must be added here or the plugin never receives it
	"token":       src.Token,
	"api_version": src.APIVersion,
	"ca_cert":     src.CACert,
})
```
This map is hand-maintained (no reflection-based passthrough) — the new field must be added to `config/types.go`, this map, AND the plugin's own `sourceConfig` struct (`main.go`), all three, or the value is silently dropped.

---

### `kernel/index/schema.go` (migration, additive)

**Analog:** same file — the existing `CREATE TABLE IF NOT EXISTS` / `CREATE INDEX IF NOT EXISTS` statements (full file, 65 lines, read above) are the template for additive, idempotent DDL.

**Pattern to copy — use RESEARCH.md's verified FTS5 block verbatim** (already tested against this repo's pinned `modernc.org/sqlite v1.54.0`):
```sql
CREATE VIRTUAL TABLE IF NOT EXISTS items_fts USING fts5(
  title, preview, content='items', content_rowid='rowid'
);

CREATE TRIGGER IF NOT EXISTS items_ai AFTER INSERT ON items BEGIN
  INSERT INTO items_fts(rowid, title, preview) VALUES (new.rowid, new.title, new.preview);
END;

CREATE TRIGGER IF NOT EXISTS items_ad AFTER DELETE ON items BEGIN
  INSERT INTO items_fts(items_fts, rowid, title, preview) VALUES('delete', old.rowid, old.title, old.preview);
END;

CREATE TRIGGER IF NOT EXISTS items_au AFTER UPDATE ON items BEGIN
  INSERT INTO items_fts(items_fts, rowid, title, preview) VALUES('delete', old.rowid, old.title, old.preview);
  INSERT INTO items_fts(rowid, title, preview) VALUES (new.rowid, new.title, new.preview);
END;
```
Append this to the `const schema = \`...\`` string, after the existing `idx_items_chrono` index — same file, same string-literal convention, no new migration mechanism needed.

---

### `kernel/index/store.go` (add `Search()`)

**Analog:** same file, `StreamItems()` (lines 193-242, read above).

**Pattern to copy:** identical method shape — `func (s *Store) Search(ctx context.Context, webspaceName, query string) ([]SearchResult, error)`, same `QueryContext` → `rows.Scan` → `json.Unmarshal` labels/provenance loop, same `fmt.Errorf("index: ...: %w", err)` wrapping convention. Use RESEARCH.md's verified query (webspace-scoped join through `webspace_items`, `ORDER BY rank ASC` — bm25 is more-negative-is-better, do NOT use DESC):
```go
const searchQuery = `
SELECT items.id, items.source_type, items.title, items.preview,
       items.timestamp_unix, items.deep_link, items.fidelity,
       snippet(items_fts, -1, '‹', '›', '…', 12) AS snip,
       bm25(items_fts) AS rank
FROM items_fts
JOIN items ON items.rowid = items_fts.rowid
JOIN webspace_items ON webspace_items.item_id = items.id
WHERE webspace_items.webspace_name = ?
  AND items_fts MATCH ?
ORDER BY rank ASC
LIMIT 50
`
```
**FTS5 syntax caveat (must handle):** wrap the raw query in a helper that either quotes it as an FTS5 phrase or catches a `MATCH` syntax error and returns an empty result set — do not hand raw user text to `MATCH` unescaped (RESEARCH.md Pattern 3).

---

### `kernel/httpapi/search.go` (controller, request-response)

**Analog:** `kernel/httpapi/stream.go` (full file, 126 lines, read above).

**Imports pattern (lines 1-11):**
```go
import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/davison/webspaces/kernel/index"
)
```

**Handler pattern (lines 52-89 of `stream.go`):** copy the shape exactly — `chi.URLParam(r, "webspace")`, `store.WebspaceExists` 404-check, then the source-specific read (`store.Search` instead of `store.StreamItems`), then `WriteJSON`/`WriteError` from `routes.go`. Read the `q` query param via `r.URL.Query().Get("q")`; an empty/missing `q` should return an empty results array (200), not an error — mirrors `stream.go`'s "never reaches a plugin" structural guarantee (`Search(store *index.Store)` — index-only signature, same as `StreamHandler(store *index.Store)`).

**Response shape convention (lines 41-46, 91-126 of `stream.go`):** define a `searchResponse struct { SchemaVersion int; Webspace string; Query string; Results []searchResult }` and a `toSearchResult(row) searchResult` mapping function, same pattern as `toStreamItem`.

---

### `kernel/httpapi/routes.go` (register search route)

**Analog:** same file, line 43 (`r.Get("/api/webspaces/{webspace}/stream", StreamHandler(store))`).

**Pattern to copy:**
```go
r.Get("/api/webspaces/{webspace}/search", SearchHandler(store))
```
Add directly below the existing stream route registration (line 43), same `Router()` function, same `store`-only signature convention.

---

### `web/src/lib/api.ts` (add search types + function)

**Analog:** same file — `StreamResponse`/`getStream` (lines 55-60, 146-149, read above).

**Pattern to copy:**
```typescript
export interface SearchResult {
	id: string;
	source_type: string;
	title: string;
	snippet: string;
	timestamp_unix: number;
	link: Link;
}

export interface SearchResponse {
	schema_version: number;
	webspace: string;
	query: string;
	results: SearchResult[];
}

/** GET /api/webspaces/{webspace}/search?q= */
export function searchWebspace(webspace: string, query: string): Promise<SearchResponse> {
	return getJSON<SearchResponse>(
		`/api/webspaces/${encodeURIComponent(webspace)}/search?q=${encodeURIComponent(query)}`
	);
}
```
Uses the existing `getJSON<T>` helper (lines 99-115) — same `ApiError` envelope handling, no new fetch/error-handling code needed.

---

### `web/src/lib/components/StreamRow.svelte` (render `group_label`)

**Analog:** same file (self-modification) — the existing metadata strip (lines 62-91, read above).

**Pattern to copy:** add a `shrink-0` span for `item.group_label` as the FIRST item inside `.stream-row-meta`, before the date span (per UI-SPEC's E3 resolution: "sender... before the date"), using the same Label-role typography (14px/400/1.4) and plain `text-muted-foreground`/`text-foreground` color (never accent, per UI-SPEC Color section). Omit entirely when `group_label` is empty (no "Unknown sender" placeholder) — same `{#if}` guard style already used for `item.preview` (lines 99-103).
```svelte
{#if item.group_label}
	<span class="shrink-0">{item.group_label}</span>
{/if}
<span class="shrink-0">{formatItemDate(item.timestamp_unix)}</span>
```

---

### `web/src/lib/components/DetailPane.svelte` (render `group_label`)

**Analog:** same file (self-modification) — the header block (lines 1-60+, read above; the title/date rendering is just below what was read — grep `formatDate` usage in the template for exact insertion point).

**Pattern to copy:** insert `item.group_label` as its own line directly under the title, before the date+labels row (per UI-SPEC), truncating to one line with ellipsis and a `title` attribute for the full value on hover (same treatment UI-SPEC specifies as matching the webspace title's own truncation).

---

### `web/src/lib/components/SearchBox.svelte` (NEW component)

**Analog:** `web/src/lib/components/SourceFilterChips.svelte` — closest existing header-tier interactive control (not fully read in this pass since UI-SPEC already fully specifies SearchBox's behavior; read this file before implementation for the exact header-slot integration pattern and `$props()`/`$state()` conventions this codebase uses, matching `StreamRow.svelte`'s own `let { ... } = $props()` style shown above).

**Core pattern:** new shadcn `input` block (`npx shadcn-svelte@latest add input`, per UI-SPEC Registry Safety) + `@lucide/svelte/icons/search` and `.../x` icons + a ~300ms debounce calling `searchWebspace()` from `api.ts`. Follow `StreamRow.svelte`'s `$props()`/`$state()` idiom (Svelte 5 runes) for `query`, `onquery` callback prop shape. Copywriting strings (placeholder, clear-button label, empty/error states) are locked verbatim in `03-UI-SPEC.md`'s Copywriting Contract table — do not paraphrase.

## Shared Patterns

### Outbound host allowlist (plugin egress restriction)
**Source:** `plugins/silverbullet/client.go` lines 161-188 (`allowHost`) + `plugins/silverbullet/outbound_hosts_test.go` lines 18-53 (predicate-table test)
**Apply to:** `plugins/proton/client.go`'s dialer and its test — same permit/refuse shape (configured host + localhost + loopback IPs; everything else refused via a wrapped sentinel error), adapted from `http.Transport.DialContext` to `go-imap`'s `Dialer` interface (see RESEARCH.md's `pinnedDialer` code example for the adapted shape).

### PLUG-02 read-only mechanical enforcement
**Source:** `plugins/paperless/readonly_test.go` (full AST-scan pattern)
**Apply to:** `plugins/proton/readonly_test.go` — new sibling test targeting IMAP mutating identifiers (`Store`/`Expunge`/`Move`/`Append`/`Delete`/`Copy`) instead of `net/http` verbs, using the identical `ast.Inspect` walking mechanism.

### Config field plumbing (three-place rule)
**Source:** `kernel/config/types.go` (`CACert` field) + `kernel/pluginhost/host.go`'s `launch()` `sourceConfig` map + `plugins/silverbullet/main.go`'s `sourceConfig` struct
**Apply to:** any new `config.Source` field this phase needs (`Username`) — must be added in all three places or the value is silently dropped at the plugin boundary (RESEARCH.md Pitfall 1).

### HTML rendition sanitize-and-wrap (bluemonday + CSP iframe)
**Source:** `plugins/silverbullet/render.go` (full file — `sanitizePolicy`, `themeStyle`, `WrapDocument`) + `kernel/httpapi/item.go`'s `renditionHandler` CSP (referenced by RESEARCH.md, not re-read this pass — grep confirms `default-src 'none'; style-src 'unsafe-inline'; object-src 'none'; sandbox`)
**Apply to:** `plugins/proton/body.go` — reuse `themeStyle`/`WrapDocument` verbatim; sanitize policy needs a narrower `AllowAttrs("style")` scope than bluemonday's own email demo (RESEARCH.md Pitfall 3) since email HTML (unlike SilverBullet's own-authored markdown) is untrusted third-party content.

### Index-only handler signature (structural plugin-isolation guarantee)
**Source:** `kernel/httpapi/stream.go`'s `StreamHandler(store *index.Store) http.HandlerFunc` + `kernel/httpapi/routes.go`'s `Router()` composition
**Apply to:** `kernel/httpapi/search.go`'s `SearchHandler(store *index.Store) http.HandlerFunc` — same "structurally unable to reach a plugin" guarantee (KERN-02), same `WebspaceExists` 404-check-first pattern, same `WriteJSON`/`WriteError` envelope helpers from `routes.go`.

### Sentinel error + status-code mapping (plugin RPC boundary)
**Source:** `plugins/silverbullet/client.go`'s `ErrNotFound`/`ErrForeignHost` + `plugins/silverbullet/plugin.go`'s `errors.Is(err, ErrNotFound)` → `codes.NotFound` mapping (lines 122-126, 226-229)
**Apply to:** `plugins/proton/client.go` and `plugins/proton/plugin.go` — same sentinel-error discrimination pattern for "safe to skip" vs "fail the whole Match/Fetch" errors, plus a new sentinel for messages missing `Message-Id` (Pitfall 6).

## No Analog Found

None — every file in this phase has at least a role-match analog somewhere in the existing SilverBullet/paperless plugins or the existing kernel/httpapi/index/web layers. The one genuinely novel element (FTS5 search) has no *code* analog in this repo but is fully specified by RESEARCH.md's own verified query/schema, which was tested end-to-end against this repo's pinned SQLite dependency in the research session — treat RESEARCH.md's Pattern 3 as authoritative in place of a codebase analog for `items_fts`/`Search()`.

## Metadata

**Analog search scope:** `plugins/silverbullet/`, `plugins/paperless/`, `kernel/config/`, `kernel/pluginhost/`, `kernel/index/`, `kernel/httpapi/`, `web/src/lib/`
**Files scanned:** ~20 (full reads: `silverbullet/{client,plugin,main,render}.go`, `silverbullet/outbound_hosts_test.go`, `paperless/readonly_test.go`, `kernel/config/types.go`, `kernel/index/schema.go`, `kernel/httpapi/{stream,routes}.go`, `web/src/lib/{api.ts,components/StreamRow.svelte,components/DetailPane.svelte}`; targeted reads/greps: `kernel/pluginhost/host.go` launch(), `kernel/index/store.go` StreamItems/GetItem)
**Pattern extraction date:** 2026-07-29
