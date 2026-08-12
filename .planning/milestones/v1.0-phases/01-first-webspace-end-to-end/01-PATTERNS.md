# Phase 1: First Webspace, End to End - Pattern Map

**Mapped:** 2026-07-27
**Files analyzed:** 17 (new files, greenfield repo)
**Analogs found:** 0 / 17 (no existing source code in repo — see note below)

## Note: Greenfield Repository

This repository currently contains only `.planning/` and `.claude/` — no `kernel/`, `plugins/`, `web/`, `proto/`, or `cmd/` directories exist yet. There are **no in-repo analogs** for any file in this phase. Per the task instructions, every pattern assignment below is anchored to the concrete, source-verified code excerpts already captured in `01-RESEARCH.md` (which quotes live upstream code from `hashicorp/go-plugin`, `svelte.dev` docs, and hand-designed schemas/protos built specifically for this project). No analog file paths are invented.

Because this phase establishes the patterns every later phase inherits, treat this PATTERNS.md as authoritative for Phase 2+ as well — future phase pattern-mappers should point to the actual files this phase creates (e.g. `kernel/pluginhost/`, `plugins/paperless/`) as real analogs once they exist.

## File Classification

| New File | Role | Data Flow | Pattern Source (RESEARCH.md) | Match Quality |
|----------|------|-----------|-------------------------------|---------------|
| `go.work` | config | — | Recommended Project Structure | no analog (research-only) |
| `cmd/webspaces/main.go` | controller (entrypoint) | request-response | Pattern 4 (go:embed + SPA handler) + Pattern 1 (host-side plugin client) | no analog (research-only) |
| `kernel/config/config.go` (TOML loader) | config | file-I/O | `os.ExpandEnv` + `go-toml/v2` pattern (Standard Stack / Supporting) | no analog (research-only) |
| `kernel/config/types.go` (webspace/keyword types) | model | — | D-01–D-04 config shape (CONTEXT.md) | no analog (research-only) |
| `kernel/index/schema.go` (SQLite schema) | model / migration | CRUD | Pattern 3 (`items` / `webspace_items` SQL) | no analog (research-only) |
| `kernel/index/store.go` (index read/write layer) | service | CRUD | Pattern 3 (stream query as plain SQL read) | no analog (research-only) |
| `kernel/correlate/correlate.go` (sync-time correlation engine) | service | event-driven / batch | Pattern 3 (correlation-at-sync) + Pitfall 1 | no analog (research-only) |
| `kernel/pluginhost/host.go` (go-plugin client wiring) | service | request-response | Pattern 1 (host-side entry point) | no analog (research-only) |
| `kernel/httpapi/routes.go` (go-chi routes) | controller | request-response | Architecture Diagram (HTTP API layer) + Pitfall 1 | no analog (research-only) |
| `kernel/httpapi/stream.go` (`GET /api/webspaces/{keyword}/stream`) | controller | CRUD (read) | Pattern 3 (index-only read) | no analog (research-only) |
| `kernel/httpapi/item.go` (`GET /api/items/{id}`) | controller | request-response (proxies live fetch) | Architecture Diagram trace (item-open flow) | no analog (research-only) |
| `kernel/item/item.go` (normalized Item schema, Go types mirroring `.proto`) | model | transform | Pattern 2 (`Item` / `LinkFidelity` message) | no analog (research-only) |
| `proto/webspaces/v1/plugin.proto` | config (contract) | request-response | Pattern 2 (full `.proto` sketch) | no analog (research-only) |
| `plugins/paperless/main.go` (plugin.Serve entrypoint) | controller (plugin) | request-response | Pattern 1 (plugin-side entry point) | no analog (research-only) |
| `plugins/paperless/client.go` (hand-rolled REST client) | service | request-response | Code Examples: tag resolution + document fetch + deep link + preview/download/thumb | no analog (research-only) |
| `web/svelte.config.js` | config | — | Pattern 4 (`adapter-static`, `fallback: '200.html'`) | no analog (research-only) |
| `web/src/routes/+layout.js` + stream/detail SPA components | component | request-response | Pattern 4 (`ssr = false`, `prerender = false`) + Architecture Diagram (SPA calls) | no analog (research-only) |

## Pattern Assignments

### `proto/webspaces/v1/plugin.proto` (config/contract, request-response)

**Source:** RESEARCH.md Pattern 2 (hand-designed for this project, `[ASSUMED]` shape — refine during planning against a second hypothetical source per phase notes)

```protobuf
syntax = "proto3";
package webspaces.v1;

service SourcePlugin {
  rpc Match(MatchRequest) returns (MatchResponse);
  rpc Fetch(FetchRequest) returns (FetchResponse);
  rpc Health(HealthRequest) returns (HealthResponse);
}

message MatchRequest {
  repeated string keywords = 1; // exact, case-insensitive (D-03)
}

message Item {
  string source_id      = 1; // stable, plugin-local ID (AGENT-02)
  string source_type    = 2; // "paperless" | "chat" | ...
  string title          = 3;
  string preview         = 4; // short text for the stream row
  int64  timestamp_unix  = 5; // drives chronological order
  LinkFidelity fidelity  = 6; // PLUG-03
  string deep_link       = 7; // exact URL back to the source
  map<string, string> provenance = 8; // machine-readable provenance (AGENT-02)
}

enum LinkFidelity {
  LINK_FIDELITY_UNSPECIFIED = 0;
  EXACT = 1;
  ANCHORED = 2;
  CONVERSATION_ONLY = 3;
}
```

**Constraint (PLUG-02):** No mutating RPC may ever be added to this service — read-only by construction, enforced at contract-design time, not runtime.

---

### `kernel/pluginhost/host.go` and `plugins/paperless/main.go` (go-plugin subprocess wiring)

**Source:** RESEARCH.md Pattern 1, quoted verbatim from `hashicorp/go-plugin` `main` branch (`examples/grpc/shared/interface.go`, `examples/grpc/plugin-go-grpc/main.go`, `examples/grpc/main.go`, verified live 2026-07-27)

**Handshake + GRPCPlugin shape:**
```go
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "BASIC_PLUGIN",
	MagicCookieValue: "hello",
}

type KVGRPCPlugin struct {
	plugin.Plugin
	Impl KV
}

func (p *KVGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterKVServer(s, &GRPCServer{Impl: p.Impl})
	return nil
}

func (p *KVGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{client: proto.NewKVClient(c)}, nil
}
```

**Plugin-side entrypoint (`plugins/paperless/main.go`):**
```go
plugin.Serve(&plugin.ServeConfig{
	HandshakeConfig: shared.Handshake,
	Plugins: map[string]plugin.Plugin{
		"kv": &shared.KVGRPCPlugin{Impl: &KV{}},
	},
	GRPCServer: plugin.DefaultGRPCServer,
})
```

**Host-side entrypoint (`kernel/pluginhost/host.go`):**
```go
client := plugin.NewClient(&plugin.ClientConfig{
	HandshakeConfig: shared.Handshake,
	Plugins:         shared.PluginMap,
	Cmd:             exec.Command("sh", "-c", os.Getenv("KV_PLUGIN")),
	AllowedProtocols: []plugin.Protocol{
		plugin.ProtocolNetRPC, plugin.ProtocolGRPC},
})
defer client.Kill()
rpcClient, _ := client.Client()
raw, _ := rpcClient.Dispense("kv_grpc")
```

**Adaptation notes for this project:**
- Restrict `AllowedProtocols` to `[]plugin.Protocol{plugin.ProtocolGRPC}` only (no net/rpc) — gRPC streaming matters for later large-volume sources.
- `Cmd` should point at a discovered plugin binary path from a plugins directory (default: `plugins/` next to the kernel binary, overridable via `plugins_dir` config key), not an env var.
- Wire `hashicorp/go-hclog` from the start so plugin subprocess stdout/stderr is captured structurally, not left as raw interleaved output (Supporting Libraries table, RESEARCH.md).
- Security: scrub `Authorization` header values before any log line reaches `go-hclog` (Security Domain, Known Threat Patterns table).

---

### `kernel/index/schema.go` and `kernel/correlate/correlate.go` (sync-time correlation, KERN-02)

**Source:** RESEARCH.md Pattern 3, hand-designed schema for this project (standard materialized-correlation pattern)

```sql
CREATE TABLE items (
  id            TEXT PRIMARY KEY,     -- "{source_type}:{source_id}"
  source_type   TEXT NOT NULL,
  title         TEXT NOT NULL,
  preview       TEXT NOT NULL,
  timestamp_unix INTEGER NOT NULL,
  fidelity      TEXT NOT NULL,
  deep_link     TEXT NOT NULL,
  provenance_json TEXT NOT NULL,
  synced_at     INTEGER NOT NULL
);

CREATE TABLE webspace_items (
  webspace_keyword TEXT NOT NULL,
  item_id          TEXT NOT NULL REFERENCES items(id),
  PRIMARY KEY (webspace_keyword, item_id)
);

SELECT items.* FROM items
JOIN webspace_items ON webspace_items.item_id = items.id
WHERE webspace_items.webspace_keyword = ?
ORDER BY items.timestamp_unix ASC;
```

**Core rule (must be enforced in code structure, not just convention):** `kernel/correlate/correlate.go` is the ONLY code path allowed to call a plugin's `Match()` RPC, and it runs only at sync time (startup + manual trigger for Phase 1, per Claude's Discretion). `kernel/httpapi/stream.go` must be structured so it physically cannot import `pluginhost` — it only imports `kernel/index` and does a plain SQL read (see Pitfall 1 in RESEARCH.md). Keep the schema `rowid`-compatible (don't declare `items` as `WITHOUT ROWID`) so an external-content FTS5 table can be bolted on in Phase 3 without a migration.

---

### `plugins/paperless/client.go` (hand-rolled paperless-ngx REST client, SRC-04)

**Source:** RESEARCH.md "Code Examples" section, verified against `paperless-ngx` `src/documents/filters.py` and `src-ui/src/app/app-routing.module.ts` (main branch, fetched 2026-07-27)

**Two-step tag resolution + document fetch (exact, case-insensitive keyword matching — D-03):**
```
# Step 1 — resolve each webspace keyword to a tag ID (case-insensitive exact match)
GET /api/tags/?name__iexact=house-move
Authorization: Token <token>
Accept: application/json; version=10

# Step 2 — fetch documents matching ANY resolved tag ID (confirmed OR / __in semantics)
GET /api/documents/?tags__id__in=12,47
Authorization: Token <token>
Accept: application/json; version=10
```

**Exact deep link (PLUG-03 "exact" fidelity):**
```
https://<paperless-host>/documents/{id}
```

**Live content fetch on item-open (KERN-03 — called only from `Fetch()`, never `Match()`):**
```
GET /api/documents/{id}/preview/     # inline preview
GET /api/documents/{id}/download/    # original file
GET /api/documents/{id}/thumb/       # thumbnail
# all accept an optional ?version={version_id} query param
```

**Critical pitfall to avoid (Pitfall 2, RESEARCH.md):** Do NOT use `tags__name__icontains` — it is the most "discoverable" filter but causes false-positive substring matches (`house` matching `Household`), violating D-03. Always resolve via `iexact` (or fetch `/api/tags/` once and filter client-side with `strings.EqualFold`), then join via `tags__id__in`.

**Chronological ordering (Pitfall 4, RESEARCH.md — Claude's Discretion, A2):** Sort primarily by `created` (date-only as of API v9+, do NOT use deprecated `created_date`), secondarily by `added` (full datetime) for stable same-day ordering and a millisecond-level `timestamp_unix`.

**Preview storage (Open Question 1, RESEARCH.md):** Truncate the document's OCR'd `content` field to a fixed character budget (280–500 chars) at sync time and store as `items.preview` — keeps KERN-03's "metadata + preview only" boundary honest.

---

### `cmd/webspaces/main.go` + static serving (SvelteKit embed)

**Source:** RESEARCH.md Pattern 4, `svelte.dev/docs/kit/single-page-apps` (fetched 2026-07-27) + hand-designed Go serving code

**`web/svelte.config.js`:**
```javascript
import adapter from '@sveltejs/adapter-static';

export default {
  kit: {
    adapter: adapter({
      fallback: '200.html' // NOT 'index.html' — conflicts with prerendering output
    })
  }
};
```

**`web/src/routes/+layout.js`:**
```javascript
export const ssr = false;
export const prerender = false;
```

**`cmd/webspaces/main.go` embed + SPA fallback handler:**
```go
//go:embed all:web/build
var webAssets embed.FS

func spaHandler(assets fs.FS) http.HandlerFunc {
	fileServer := http.FileServer(http.FS(assets))
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := fs.Stat(assets, strings.TrimPrefix(r.URL.Path, "/")); err != nil {
			r.URL.Path = "/200.html" // SPA fallback for unmatched client routes
		}
		fileServer.ServeHTTP(w, r)
	}
}
```

**Critical pitfall (Pitfall 3, RESEARCH.md):** Never name the fallback `index.html` — must be `200.html` (or another non-`index.html` name), both in `svelte.config.js` and the Go handler's fallback path.

---

### `kernel/config/config.go` (TOML loader, KERN-01, D-01–D-04)

**Source:** RESEARCH.md Supporting Libraries table (`os.ExpandEnv` + `pelletier/go-toml/v2`)

**Pattern:** Read the raw TOML file as a string, run `os.ExpandEnv()` over it for `${VAR}` secret interpolation, THEN unmarshal with `go-toml/v2`. Do not reach for Viper or a templating engine.

```go
raw, err := os.ReadFile(configPath)
expanded := os.ExpandEnv(string(raw))
var cfg Config
err = toml.Unmarshal([]byte(expanded), &cfg)
```

**Constraint (D-04):** Secrets (e.g. paperless API token) must never be written to `config.toml` directly — only `${VAR}` references belong in the file; validate this isn't violated (e.g. no bare-looking-like-a-token strings) is out of scope for Phase 1, but never store resolved tokens back to disk.

**Config location:** `~/.config/webspaces/config.toml` (single file — kernel settings, source connections, webspace definitions all in one place, per D-04).

---

## Shared Patterns

### Read-only enforcement (PLUG-02)
**Source:** RESEARCH.md Security Domain / Known Threat Patterns table
**Apply to:** `.proto` contract design and all plugin implementations.
No mutating RPC (`Update`/`Delete`/`Create`) exists anywhere in `plugin.proto`. This is a design constraint enforced by contract shape, not a runtime check — every future plugin (Phase 2+) inherits this.

### Correlation-at-sync boundary (KERN-02)
**Source:** RESEARCH.md Pitfall 1
**Apply to:** `kernel/httpapi/stream.go`, `kernel/correlate/correlate.go`, `kernel/pluginhost/host.go`
Structure packages so the HTTP stream handler is physically incapable of importing `pluginhost` — enforce via package boundaries, not code review alone.

### cgo isolation (multi-module workspace)
**Source:** RESEARCH.md Pitfall 5 + Recommended Project Structure
**Apply to:** `go.work`, `kernel/` module, `plugins/paperless/` module
Even though the paperless plugin needs no cgo, set up `go.work` with kernel and plugin as separate modules from the first commit, so Phase 4's Signal plugin (which needs cgo for SQLCipher) adds a module rather than forcing a repo restructure.

### Localhost-only binding (security default)
**Source:** RESEARCH.md Security Domain / Known Threat Patterns table
**Apply to:** `kernel/httpapi/routes.go`, `cmd/webspaces/main.go`
Bind the `go-chi` HTTP server to `127.0.0.1` by default for Phase 1 (single local user, no remote clients yet, no auth). Document this as the default; flag any future LAN exposure as its own security review.

### Deep-link fidelity as three-value enum, not boolean (PLUG-03, UI-04)
**Source:** RESEARCH.md Don't Hand-Roll table
**Apply to:** `kernel/item/item.go`, `proto/webspaces/v1/plugin.proto`, SPA "open in source" component
Use `LinkFidelity` (`EXACT` / `ANCHORED` / `CONVERSATION_ONLY`), not an ad hoc boolean — the UI needs the distinction, and later sources (chat) can't always deep-link to one message.

## No Analog Found

All 17 files listed above have no in-repo analog (greenfield repository, no source code exists prior to this phase). Every pattern assignment is anchored to RESEARCH.md's source-verified code excerpts and hand-designed schemas instead. The planner should treat RESEARCH.md's Architecture Patterns section (Patterns 1–4), Code Examples section, and Common Pitfalls section as the canonical pattern source for this phase, cross-referenced via this file's per-file table above.

## Metadata

**Analog search scope:** entire repository (`find . -type f`, excluding `.git`, `.planning`, `.claude`) — confirmed empty of source code.
**Files scanned:** 0 source files (repository contains only planning/config directories at time of this phase)
**Pattern extraction date:** 2026-07-27
</content>
</invoke>
