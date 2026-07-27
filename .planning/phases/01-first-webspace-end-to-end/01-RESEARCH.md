# Phase 1: First Webspace, End to End - Research

**Researched:** 2026-07-27
**Domain:** Go kernel + gRPC-subprocess plugin host + SQLite hybrid index + SvelteKit SPA, proven end-to-end against the paperless-ngx REST API
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Config format is **TOML**. — Reversibility: costly — every downstream artifact (config loader, docs, the user's hand-maintained file, examples in plugin contract docs) assumes the format; switching later means a migration of the user's config and all documentation.
- **D-02:** Each webspace defines a **list of keywords** (e.g. `["house-move", "House"]`). **All keywords are matched in all plugins** against each source's native categorization (paperless tags, IMAP folders/labels, chat group names, SilverBullet tags/pages). There is **no per-source override syntax** — a silo that names things differently is handled by adding its variant to the shared keyword list. — Reversibility: costly — the keyword-list shape is baked into the plugin contract's match request and every plugin's matching code; Phases 2–5 plugins all build on it.
- **D-03:** Keyword matching is **exact, case-insensitive**: `house` matches tag `House` but not `Household`. No substring or prefix matching — variants are listed explicitly. Deterministic, no false positives (consistent with the v1 config-mapped correlation decision in PROJECT.md).
- **D-04:** **One config file** at `~/.config/webspaces/config.toml` containing kernel settings, source connections, and webspace definitions. **Secrets are not stored in the file** — tokens (e.g. paperless-ngx API token) come from environment variables, with `${VAR}` interpolation supported in the config file.

### Claude's Discretion

- **Stream & detail presentation** — what a document item shows in the stream (title/snippet/tags/thumbnail), which timestamp drives chronological order, and what the detail pane renders for a paperless document (extracted text vs embedded PDF preview).
- **Sync trigger for Phase 1** — the full scheduler/coordinator is Phase 2 (KERN-04); Phase 1 needs a minimal trigger (startup sync, simple interval, or manual). Also what is stored as the local "preview" for a document.
- **Running the service** — command shape (e.g. single `webspaces serve`), plugins directory location, listen port, dev vs prod serving of the SPA.

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| KERN-01 | User can define webspaces in config: each has a keyword mapped to the native categorization of each source | TOML config shape (D-01–D-04) + `pelletier/go-toml/v2` decode pattern + `os.ExpandEnv` interpolation, documented below |
| KERN-02 | Correlation is resolved at sync time and persisted in the local index (never computed at query time) | "Correlation-at-sync" pattern + SQLite items/webspace_items schema in Architecture Patterns |
| KERN-03 | Hybrid store: metadata + preview indexed locally; full content fetched live from source on open | Hybrid store pattern + paperless-ngx live-fetch endpoints (`/preview/`, `/thumb/`, `/download/`) documented below |
| PLUG-01 | Plugins run as isolated subprocesses under a documented, versioned gRPC contract | `hashicorp/go-plugin` v1.8.0 verified handshake/GRPCPlugin/Serve pattern, `buf` codegen setup, both with source-verified code |
| PLUG-02 | Contract is read-only by construction — no source-mutating methods exist | `.proto` design guidance (two-source sketch: documents + chat) in Architecture Patterns |
| PLUG-03 | Every item declares deep-link fidelity: exact / anchored / conversation-only | paperless-ngx exact deep link path `/documents/{id}` verified from Angular routing source |
| SRC-04 | paperless-ngx plugin via REST API; matches on tags; exact deep links to documents | Verified `/api/documents/`, `/api/tags/`, tag-ID `__in` OR-matching, token auth header, API version pinning |
| UI-01 | Web UI shows a webspace as a chronological cross-source stream with a detail pane | SvelteKit SPA shell pattern (adapter-static `fallback`) + Recommended Project Structure |
| UI-03 | Items render inline previews appropriate to type (document) | Stream/detail presentation discretion notes + paperless preview endpoint |
| UI-04 | Every item has an "open in source" affordance matching its declared link fidelity | Same as PLUG-03/SRC-04 |
| AGENT-02 | Item schema and kernel HTTP API designed for programmatic/agent consumption | Item schema design notes (stable IDs, structured content, provenance) in Architecture Patterns |
</phase_requirements>

## Summary

Phase 1 is a walking skeleton: every kernel-level concept the whole project depends on (Item schema, index store, webspace config loader, sync-time correlation, plugin host, HTTP API, SPA shell) must land in this phase, but all of it proves itself against exactly one real, low-risk source — paperless-ngx. Nothing here is exotic engineering; the stack recommended in `.claude/CLAUDE.md` (Go + `hashicorp/go-plugin` gRPC subprocess + `modernc.org/sqlite` + `go-chi/chi` + SvelteKit `adapter-static`) is proven, current, and directly verified against upstream sources during this research pass (go-plugin v1.8.0, modernc.org/sqlite v1.54.0, go-chi/chi v5.3.1, grpc-go v1.82.1, protobuf v1.36.11 — all confirmed live against the Go module proxy on 2026-07-27; SvelteKit/adapter-static/Vite confirmed live against the npm registry the same day).

The one piece of real domain knowledge this phase needs that wasn't already nailed down by the stack research is **exactly how paperless-ngx exposes tags and documents**, since two of the five success criteria (SRC-04, PLUG-03) hinge on it. This was fetched directly from the paperless-ngx source (`src/documents/filters.py` and `src-ui/src/app/app-routing.module.ts` on `main`, 2026-07-27) rather than inferred: tag name matching only supports `istartswith/iendswith/icontains/iexact` (no `__in` for names), but tag **ID** matching supports `tags__id__in=<id1>,<id2>,...` with confirmed OR semantics (`__id__in`, Django ORM). The correct plugin implementation is therefore a two-step call: resolve each webspace keyword to a tag ID via `/api/tags/?name__iexact=<keyword>` (or one paginated fetch of all tags, filtered client-side), then fetch documents in one shot via `/api/documents/?tags__id__in=<id1>,<id2>`. The frontend's exact deep-link path is `/documents/{id}` (confirmed from the Angular routing module), which is what PLUG-03/UI-04 need for the "exact" fidelity tier.

**Primary recommendation:** Build the kernel as a single Go module (pure-Go, no cgo) with a `plugins/` subdirectory holding one more Go module (`plugins/paperless`, also no cgo needed for this plugin) that speaks the shared `.proto` contract over `hashicorp/go-plugin`'s gRPC transport; sketch the `.proto` against both a `Document` item and a hypothetical `ChatMessage` item now (per the phase notes) even though only the paperless-ngx plugin ships; persist correlation results as rows at sync time in a `webspace_items` join table (never a query-time JOIN against live tag data); serve the SvelteKit SPA in `adapter-static` SPA mode (`fallback: '200.html'`) embedded via `go:embed` behind `go-chi/chi`.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Webspace config definition & keyword list (KERN-01) | API/Backend (kernel config loader) | — | Config is parsed once at kernel startup; no client or plugin involvement |
| Sync-time correlation & persistence (KERN-02) | API/Backend (kernel sync/correlation engine) | Plugin (source of native categorization data) | Kernel owns the join between webspace keywords and per-source items; plugin only supplies raw matched items per a `MatchRequest` |
| Hybrid store: metadata/preview local, full content live (KERN-03) | Database/Storage (kernel SQLite index) | API/Backend (kernel HTTP API), Plugin (live content fetch) | Kernel owns the persisted preview; the plugin subprocess is the only thing allowed to touch the live paperless-ngx content, brokered through the kernel on item-open |
| Plugin subprocess isolation (PLUG-01, PLUG-02) | API/Backend (plugin host process supervision) | — | `hashicorp/go-plugin` client lives in the kernel process; each plugin is a separate OS process the kernel launches and owns the lifecycle of |
| Deep-link fidelity declaration (PLUG-03) | Plugin (per-source URL knowledge) | Browser/Client (renders the affordance) | Only the plugin knows the source's own URL scheme; the kernel/UI just relay the declared fidelity + URL |
| paperless-ngx REST integration (SRC-04) | Plugin (paperless plugin subprocess) | — | REST calls to paperless-ngx happen only inside the plugin process, never the kernel, keeping the kernel source-agnostic |
| Chronological cross-source stream + detail pane (UI-01, UI-03) | Browser/Client (SvelteKit SPA) | API/Backend (kernel HTTP API serves the stream JSON) | Stream renders from the kernel's already-correlated local index; the SPA does no correlation of its own |
| "Open in source" affordance (UI-04) | Browser/Client | Plugin (declares fidelity + link) | UI is a passive renderer of what the item's provenance data says |
| Agent-consumable HTTP API (AGENT-02) | API/Backend (kernel HTTP API) | — | Same JSON contract the SPA consumes must also be fetchable directly — no separate "agent API" |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | 1.23+ (toolchain on this machine: go1.26.5) `[VERIFIED: local toolchain]` | Kernel language | Locked in `.claude/CLAUDE.md`; only language with a mature WhatsApp linked-device library (needed by Phase 5, not this phase, but the module/workspace layout is decided now) |
| `modernc.org/sqlite` | v1.54.0, published 2026-07-15 `[VERIFIED: Go module proxy]` | Local metadata/preview index (hybrid store) with FTS5 compiled in | Pure-Go, cgo-free — keeps the kernel binary a single static executable; FTS5 needed later (Phase 3) but the schema should be FTS5-friendly from day one |
| `github.com/hashicorp/go-plugin` | v1.8.0, published 2026-04-29 `[VERIFIED: Go module proxy + source examples]` | Kernel⇄plugin gRPC-over-subprocess transport (PLUG-01) | Battle-tested (Terraform/Vault/Nomad providers); confirmed still actively maintained under IBM copyright (HashiCorp/IBM) as of the v1.8.0 tag |
| `google.golang.org/grpc` | v1.82.1, published 2026-07-15 `[VERIFIED: Go module proxy]` | Wire transport required by go-plugin's gRPC plugin type | Required transitively; gives streaming (useful for large mailbox/chat sync in later phases) |
| `google.golang.org/protobuf` | v1.36.11, published 2025-12-12 `[VERIFIED: Go module proxy]` | Protobuf runtime for the generated `.proto` stubs | Required transitively by grpc-go and go-plugin's gRPC codegen |
| `github.com/go-chi/chi/v5` | v5.3.1, published 2026-07-05 `[VERIFIED: Go module proxy]` | HTTP router for the kernel's JSON API (stream, item detail, webspace list) | Thin, `net/http`-compatible, no magic — right-sized for a handful of routes; also the substrate for AGENT-02's HTTP API |
| `github.com/pelletier/go-toml/v2` | v2.4.3, published 2026-07-05 `[VERIFIED: Go module proxy]` | TOML config parsing (D-01 locked format) | TOML 1.0 spec compliant, actively maintained, idiomatic struct-tag decode API; faster and more current than BurntSushi/toml for a v2-line project |
| SvelteKit | `@sveltejs/kit` 2.70.1 `[VERIFIED: npm registry]`, `svelte` 5.56.8 `[VERIFIED: npm registry]` | Web UI: cross-source stream + detail pane SPA | Locked in `.claude/CLAUDE.md`; SPA mode via adapter-static is the right shape for independent-scroll stream+detail UX |
| `@sveltejs/adapter-static` | 3.0.10 `[VERIFIED: npm registry]` | Builds the SvelteKit app as a static SPA bundle for `go:embed` | Only adapter that produces a plain static bundle with an SPA fallback page, matching the "single static Go binary" deployment model |
| `vite` | 8.1.5 `[VERIFIED: npm registry]` | Build tool underlying SvelteKit | Ships as SvelteKit's build dependency; no separate choice to make |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/hashicorp/go-hclog` | v1.6.3 `[VERIFIED: Go module proxy]` | Structured logging bridged from the plugin subprocess's stdout/stderr into the kernel's log stream | Wire this in from the start — without it, plugin panics/logs are invisible or interleave badly with kernel logs |
| `buf` CLI | not found on this machine (`command -v buf` empty); Go module `github.com/bufbuild/buf` latest tag v1.72.0 `[VERIFIED: Go module proxy]` | Protobuf codegen tool for the `.proto` contract (`buf generate`) | Install via `go install github.com/bufbuild/buf/cmd/buf@v1.72.0` or the Homebrew/binary release; `protoc` 35.1 is present on this dev machine as a fallback if `buf` is unavailable, but `buf` is what the upstream go-plugin gRPC example itself uses (`examples/grpc/buf.gen.yaml`, `buf.yaml`) and is the recommended path per `.claude/CLAUDE.md` |
| `os.ExpandEnv` (Go stdlib) | Go 1.23+ stdlib | `${VAR}` interpolation in the TOML config file (D-04) | Read the raw TOML file as a string, run `os.ExpandEnv()` over it, **then** unmarshal with `go-toml/v2` — this is the standard, dependency-free pattern; do not reach for Viper or a templating engine for this |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `pelletier/go-toml/v2` | `BurntSushi/toml` (v1.6.0, also verified live) | BurntSushi is older/more battle-tested but slower and has a less ergonomic struct-tag decode API for nested config; go-toml/v2 is the better default for a new project |
| `os.ExpandEnv` for secret interpolation | `knadh/koanf` (config-with-providers library, v2.3.5 confirmed live) | koanf adds real value once multiple config sources (flags, env, file) need merging — overkill for Phase 1's single TOML file + env-var secrets; revisit if config sourcing gets more complex |
| `buf` for `.proto` codegen | Plain `protoc` + `protoc-gen-go` + `protoc-gen-go-grpc` | `protoc` 35.1 is already present on this dev machine and works, but `buf` is what upstream go-plugin's own gRPC example ships (`buf.gen.yaml`/`buf.yaml`) and gives lint/breaking-change checks for free — worth the one-time install for a contract that's meant to be published for third parties (PLUG-05, Phase 2) |

**Installation:**
```bash
# Kernel module (repo root or kernel/ subdir)
go get modernc.org/sqlite@v1.54.0
go get github.com/hashicorp/go-plugin@v1.8.0
go get github.com/go-chi/chi/v5@v5.3.1
go get github.com/pelletier/go-toml/v2@v2.4.3
go get github.com/hashicorp/go-hclog@v1.6.3
go get google.golang.org/grpc@v1.82.1
go get google.golang.org/protobuf@v1.36.11

# protobuf toolchain (one-time, not a project dependency)
go install github.com/bufbuild/buf/cmd/buf@v1.72.0

# Frontend (SvelteKit SPA)
npx sv create web   # or `npm create svelte@latest web`
cd web && npm install -D @sveltejs/adapter-static@3.0.10
```

## Package Legitimacy Audit

> The seam commands `gsd-tools query package-legitimacy check` / `classify-confidence` / `research-plan` are not present in the installed gsd-tools build (v1.1.0) used in this session — they returned "Unknown command". This audit was therefore performed manually per the protocol's fallback expectations: registry existence, publish recency, source-repo linkage, and (for npm) `postinstall` script inspection, checked directly against the Go module proxy and npm registry rather than relying on training-data package names.

| Package | Registry | Age / Last Publish | Source Repo | Postinstall | Verdict | Disposition |
|---------|----------|---------------------|--------------|--------------|---------|-------------|
| `modernc.org/sqlite` | Go proxy | v1.54.0, 2026-07-15 | gitlab.com/cznic/sqlite | n/a (Go) | OK | Approved |
| `github.com/hashicorp/go-plugin` | Go proxy | v1.8.0, 2026-04-29 | github.com/hashicorp/go-plugin | n/a (Go) | OK | Approved |
| `google.golang.org/grpc` | Go proxy | v1.82.1, 2026-07-15 | github.com/grpc/grpc-go | n/a (Go) | OK | Approved |
| `google.golang.org/protobuf` | Go proxy | v1.36.11, 2025-12-12 | go.googlesource.com/protobuf | n/a (Go) | OK | Approved |
| `github.com/go-chi/chi/v5` | Go proxy | v5.3.1, 2026-07-05 | github.com/go-chi/chi | n/a (Go) | OK | Approved |
| `github.com/pelletier/go-toml/v2` | Go proxy | v2.4.3, 2026-07-05 | github.com/pelletier/go-toml | n/a (Go) | OK | Approved |
| `github.com/hashicorp/go-hclog` | Go proxy | v1.6.3 (latest tag) | github.com/hashicorp/go-hclog | n/a (Go) | OK | Approved |
| `github.com/bufbuild/buf` | Go proxy | v1.72.0 (latest tag) | github.com/bufbuild/buf | n/a (Go) | OK | Approved |
| `svelte` | npm | 5.56.8 | github.com/sveltejs/svelte | none found | OK | Approved |
| `@sveltejs/kit` | npm | 2.70.1 | github.com/sveltejs/kit | none found | OK | Approved |
| `@sveltejs/adapter-static` | npm | 3.0.10 | github.com/sveltejs/kit (monorepo) | none found | OK | Approved |
| `vite` | npm | 8.1.5 | github.com/vitejs/vite | none found | OK | Approved |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

All package names above were sourced from `.claude/CLAUDE.md` (itself a prior HIGH-confidence research pass) and cross-verified live against the Go module proxy / npm registry in this session — they carry `[VERIFIED]` status, not `[ASSUMED]`, because both the name provenance (official prior research, not a fresh WebSearch guess) and the registry check are satisfied.

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────┐
│ Browser (SvelteKit SPA,     │
│ served via adapter-static   │
│ SPA fallback + go:embed)    │
└───────────┬──────────────────┘
            │ 1. GET /api/webspaces/house-move/stream
            │ 2. GET /api/items/{id}            (detail pane open)
            ▼
┌─────────────────────────────────────────────────────────────┐
│ Kernel (Go process, go-chi router)                          │
│                                                               │
│  HTTP API layer  ──────────────────────────────────────────┐ │
│   - GET /api/webspaces/{keyword}/stream  (reads index only) │ │
│   - GET /api/items/{id}                  (may proxy live    │ │
│                                            fetch via plugin) │ │
│   - same routes serve AGENT-02's agent consumers            │ │
│  ────────────────────────────────────────────────────────────┘ │
│            │ reads                    │ on item-open,        │
│            ▼                          │ live-fetch request   │
│  ┌─────────────────────┐              ▼                      │
│  │ SQLite index         │   ┌───────────────────────────┐    │
│  │ (modernc.org/sqlite)  │   │ Plugin Host (go-plugin      │    │
│  │ - items table          │   │ client.NewClient, one per   │    │
│  │ - webspace_items table │   │ registered plugin binary)   │    │
│  │   (correlation,         │   └──────────┬───────────────┘    │
│  │    persisted at sync)   │              │ gRPC over stdio/   │
│  └─────────▲────────────────┘              │ Unix socket        │
│            │ write at sync time            ▼                    │
│  ┌─────────┴────────────────┐   ┌──────────────────────────┐   │
│  │ Sync/Correlation Engine    │──▶│ paperless-ngx plugin       │   │
│  │ - loads webspace config     │  │ (separate OS subprocess)   │   │
│  │ - for each webspace keyword  │ │ - Match(keywords) -> items  │   │
│  │   list, calls plugin.Match() │ │ - Fetch(id) -> full content │   │
│  │ - persists matched items     │ └──────────────┬─────────────┘   │
│  └──────────────────────────────┘                │ HTTPS REST      │
│            ▲ config load (TOML,                    ▼                 │
│            │ ${VAR} env interpolation)   ┌───────────────────────┐   │
│  ┌─────────┴────────────────┐            │ paperless-ngx server   │   │
│  │ ~/.config/webspaces/       │           │ (LAN, home server)     │   │
│  │ config.toml                 │           │ /api/tags/             │   │
│  └──────────────────────────────┘          │ /api/documents/         │   │
│                                              │ /documents/{id} (UI)    │   │
└──────────────────────────────────────────────────────────────────────┘
```

Trace for the primary use case: user opens the "house-move" webspace → SPA calls `GET /api/webspaces/house-move/stream` → kernel reads the already-correlated `webspace_items` + `items` join from SQLite (no live paperless call) → SPA renders the chronological stream → user clicks an item → SPA calls `GET /api/items/{id}` → kernel's plugin host asks the paperless plugin subprocess for live content over gRPC → plugin calls paperless-ngx's `/api/documents/{id}/preview/` (or similar) over HTTPS → kernel returns structured content + the `/documents/{id}` deep link + fidelity tier to the SPA.

### Recommended Project Structure

```
webspaces/
├── go.work                      # Go workspace: ties kernel + plugin modules together
├── cmd/
│   └── webspaces/               # kernel main package: `webspaces serve`
│       └── main.go
├── kernel/                      # kernel module (no cgo)
│   ├── config/                  # TOML loader, ${VAR} expansion, webspace/keyword types
│   ├── index/                   # modernc.org/sqlite access layer, schema, migrations
│   ├── correlate/                # sync-time correlation engine (KERN-02)
│   ├── pluginhost/               # go-plugin client wiring, health, lifecycle
│   ├── httpapi/                  # go-chi routes (stream, item detail, webspace list)
│   └── item/                     # normalized Item schema (shared Go types mirroring .proto)
├── proto/
│   └── webspaces/v1/
│       ├── plugin.proto          # published contract: Match, Fetch — read-only by construction
│       ├── buf.yaml
│       └── buf.gen.yaml
├── plugins/
│   └── paperless/                # separate Go module (own go.mod) — one binary per plugin
│       ├── go.mod
│       ├── main.go                # plugin.Serve(...)
│       └── client.go              # hand-rolled paperless-ngx REST client
├── web/                          # SvelteKit SPA (adapter-static, fallback: '200.html')
│   ├── src/
│   └── svelte.config.js
└── docs/
    └── plugin-contract.md         # third-party-facing docs for the .proto (PLUG-05, Phase 2)
```

**Why a Go workspace (`go.work`) with a separate module per plugin, starting now:** the paperless plugin doesn't need cgo, but the Signal plugin (Phase 4) does (SQLCipher), and per `.claude/CLAUDE.md` that cgo requirement must never leak into the kernel's own `go build`. Establishing the multi-module workspace shape in Phase 1 — even though only one plugin module exists yet — means Phase 4 adds a module, not a repo restructuring.

### Pattern 1: go-plugin gRPC subprocess (PLUG-01)

**What:** Kernel launches each plugin as a subprocess and communicates over a gRPC connection multiplexed through go-plugin's handshake protocol.
**When to use:** Every source plugin, including the paperless-ngx plugin this phase.
**Example (verified from `hashicorp/go-plugin` `main` branch, `examples/grpc/shared/`, fetched 2026-07-27):**

```go
// Source: github.com/hashicorp/go-plugin examples/grpc/shared/interface.go (verified live)
// Handshake is a common handshake that is shared by plugin and host.
var Handshake = plugin.HandshakeConfig{
	// This isn't required when using VersionedPlugins
	ProtocolVersion:  1,
	MagicCookieKey:   "BASIC_PLUGIN",
	MagicCookieValue: "hello",
}

// This is the implementation of plugin.GRPCPlugin so we can serve/consume this.
type KVGRPCPlugin struct {
	plugin.Plugin
	Impl KV // concrete Go implementation, host-side only
}

func (p *KVGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterKVServer(s, &GRPCServer{Impl: p.Impl})
	return nil
}

func (p *KVGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{client: proto.NewKVClient(c)}, nil
}
```

Plugin-side entry point:
```go
// Source: github.com/hashicorp/go-plugin examples/grpc/plugin-go-grpc/main.go (verified live)
plugin.Serve(&plugin.ServeConfig{
	HandshakeConfig: shared.Handshake,
	Plugins: map[string]plugin.Plugin{
		"kv": &shared.KVGRPCPlugin{Impl: &KV{}},
	},
	GRPCServer: plugin.DefaultGRPCServer,
})
```

Host-side entry point:
```go
// Source: github.com/hashicorp/go-plugin examples/grpc/main.go (verified live)
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

For webspaces, `AllowedProtocols` should be restricted to `[]plugin.Protocol{plugin.ProtocolGRPC}` only (no net/rpc) since gRPC streaming matters for later large-volume sources, and `Cmd` should point at the discovered plugin binary path from a plugins directory rather than an env var (per the "Claude's Discretion" note on running the service).

### Pattern 2: `.proto` contract sketched against two structurally different sources (PLUG-02, PLUG-05)

**What:** Even though only the paperless-ngx plugin ships this phase, the `.proto` must model a `Document`-shaped item and a `ChatMessage`-shaped item so Phase 2+ additions don't force a breaking contract change.
**When to use:** Writing `proto/webspaces/v1/plugin.proto` before any Go code.
**Example (hand-designed for this project — not upstream, `[ASSUMED]` shape, refine during planning):**

```protobuf
// Source: designed for this project; loosely modeled on go-plugin's KV example service shape
syntax = "proto3";
package webspaces.v1;

service SourcePlugin {
  // Match returns items whose native categorization matches any of the
  // given keywords. Read-only: no method in this contract ever mutates
  // the source.
  rpc Match(MatchRequest) returns (MatchResponse);

  // Fetch returns full, live content for a single item, keyed by the
  // plugin-local stable ID. Called on item-open, never at stream-render time.
  rpc Fetch(FetchRequest) returns (FetchResponse);

  // Health reports subprocess reachability and last sync/error (PLUG-04, Phase 2).
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
  EXACT = 1;             // e.g. paperless /documents/{id}
  ANCHORED = 2;           // e.g. IMAP message within a folder view
  CONVERSATION_ONLY = 3;  // e.g. a chat app that can't deep-link to one message
}
```

### Pattern 3: Correlation resolved at sync time, never at query time (KERN-02)

**What:** The sync engine calls each plugin's `Match(keywords)` once per webspace per sync cycle and writes the resulting item IDs into a `webspace_items` table. The HTTP stream handler only ever does a plain SQL read of that table — it never calls a plugin or recomputes matching.
**When to use:** Every sync cycle (Phase 1's minimal trigger — startup + manual — through Phase 2's full scheduler).
**Example:**

```sql
-- Source: designed for this project (standard materialized-correlation pattern)
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
-- Stream query is a plain, index-only read:
SELECT items.* FROM items
JOIN webspace_items ON webspace_items.item_id = items.id
WHERE webspace_items.webspace_keyword = ?
ORDER BY items.timestamp_unix ASC;
```

This schema is already FTS5-friendly (Phase 3, KERN-05) without a migration: an external-content FTS5 virtual table can be added over `items(title, preview)` later using `content='items', content_rowid=rowid` — no need to build that this phase, just don't design `items` in a way that blocks it (e.g. don't use a non-integer/rowid-incompatible primary key strategy without also keeping a stable rowid — SQLite's default `rowid` table behavior already satisfies this as long as `id TEXT PRIMARY KEY` isn't declared `WITHOUT ROWID`).

### Pattern 4: SvelteKit SPA embedded via `go:embed`

**What:** Build the SvelteKit app as a static SPA bundle and serve it (plus its fallback page) from the Go binary.
**When to use:** Kernel HTTP server, non-`/api/*` routes.
**Example:**

```javascript
// Source: svelte.dev/docs/kit/single-page-apps (fetched 2026-07-27) — svelte.config.js
import adapter from '@sveltejs/adapter-static';

export default {
  kit: {
    adapter: adapter({
      fallback: '200.html' // NOT 'index.html' — conflicts with prerendering output
    })
  }
};
```
```javascript
// src/routes/+layout.js
export const ssr = false;
export const prerender = false;
```

```go
// Source: designed for this project — standard go:embed + SPA-fallback serving pattern
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

### Anti-Patterns to Avoid

- **Live-fetching paperless tags/documents on every stream request:** violates KERN-02 explicitly; only `Fetch()` (item-open) may call the plugin at request time — `Match()` only ever runs at sync time.
- **Using `index.html` as the SvelteKit SPA fallback name:** current adapter-static guidance explicitly avoids this because it collides with prerendered output; use `200.html` (or another non-`index.html` name).
- **Matching keywords with substring/`icontains`:** D-03 requires exact, case-insensitive matching; paperless-ngx's own `icontains` filter is tempting but wrong here — use `iexact` (or exact match after fetching all tags client-side).
- **Storing the paperless API token in `config.toml`:** D-04 requires it come from an environment variable; only the `${VAR}` reference belongs in the file.
- **Giving the kernel binary a cgo dependency for the index:** `modernc.org/sqlite` is chosen specifically to keep the kernel pure-Go; don't reach for `mattn/go-sqlite3` here even though a later plugin (Signal) needs cgo elsewhere.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|--------------|-----|
| Subprocess handshake, health-checking, process supervision for plugins | A custom Unix-socket/JSON-RPC protocol | `hashicorp/go-plugin` | Already solves clean shutdown, crash isolation, protocol negotiation, and (via `VersionedPlugins`) contract versioning — reinventing this is exactly the kind of complexity the phase notes warn is "established" |
| TOML env-var secret interpolation | A custom templating/token-replace engine | `os.ExpandEnv` over the raw file, then `go-toml/v2` decode | Stdlib already does `${VAR}`/`$VAR` expansion; anything fancier (defaults, nested lookups) is unneeded for Phase 1's scope |
| SPA static file serving + fallback routing | Hand-rolled routing logic duplicating SvelteKit's client router | `adapter-static`'s `fallback` file + a single catch-all `go-chi` route | The SPA fallback file already contains the logic to bootstrap Svelte's own router; the Go side only needs to serve it for unmatched paths |
| Full-text index sync (deferred to Phase 3, but don't design against it) | Hand-written insert/update/delete trigger logic to keep a denormalized search index in sync | SQLite FTS5 external-content table + auto-generated triggers | Standard, well-documented SQLite pattern; don't invent a custom cache-invalidation scheme when FTS5 external-content tables solve exactly this |
| Deep-link fidelity semantics per source | An ad hoc "canOpen" boolean per item | The three-value `LinkFidelity` enum (exact/anchored/conversation-only) already decided at the requirements level (PLUG-03) | A boolean loses the distinction the UI needs to make (UI-04) between "opens exactly this item" vs "opens the containing thread" |

**Key insight:** every piece of infrastructure this phase needs (plugin RPC, config parsing, static file serving, index storage) has a mature, already-chosen library in `.claude/CLAUDE.md`. The only genuinely custom code this phase should write is: the correlation engine's SQL, the `.proto` contract's message shapes, and the paperless-ngx REST client (which paperless-ngx's own docs confirm has no maintained Go SDK, so a thin hand-rolled client is correct, not a shortcut).

## Common Pitfalls

### Pitfall 1: Recomputing correlation in the HTTP handler
**What goes wrong:** It's tempting, when wiring up the stream endpoint, to just call the plugin's `Match()` live and skip the index for "freshness." This silently violates KERN-02.
**Why it happens:** During early development the sync trigger is minimal (startup/manual per Claude's Discretion), so live-calling feels like it "just works" and is less code.
**How to avoid:** Structure the HTTP handler so it is physically incapable of importing `pluginhost` — only `index` — for the stream route. Reserve plugin calls for the `Fetch` (item-open) and sync-engine code paths.
**Warning signs:** Stream response times scale with plugin/network latency instead of SQLite query time; stream still updates without a sync having run.

### Pitfall 2: `tags__name__icontains`-style matching instead of exact, case-insensitive
**What goes wrong:** paperless-ngx's filter API makes `icontains` the most "discoverable" tag filter in casual testing; using it would make keyword `house` match tag `Household` (violates D-03).
**Why it happens:** `icontains` "just works" in ad hoc curl testing without surfacing the false-positive risk until real data is loaded.
**How to avoid:** Resolve each keyword to a tag ID via `tags__name__iexact=<keyword>` (or fetch `/api/tags/` once and filter client-side with `strings.EqualFold`), then use `tags__id__in=<id1>,<id2>` for the actual document fetch (confirmed OR semantics via Django's `__in` lookup in `documents/filters.py`).
**Warning signs:** Documents tagged `Household` (or any tag containing the keyword as a substring) show up in a webspace whose keyword list didn't include that variant.

### Pitfall 3: SvelteKit fallback named `index.html`
**What goes wrong:** Naming the adapter-static fallback `index.html` (the intuitive default) breaks in combination with any prerendered routes, per current SvelteKit docs.
**Why it happens:** Older SvelteKit tutorials/community examples predate this guidance.
**How to avoid:** Use `fallback: '200.html'` (or another non-`index.html` name) in `svelte.config.js`, and make the Go static handler serve that specific file for unmatched paths.
**Warning signs:** Client-side routes 404 or serve stale content only in production builds, not in `vite dev`.

### Pitfall 4: `created` field precision loss for chronological ordering
**What goes wrong:** As of paperless-ngx API v9+, the document `created` field is a **date**, not a datetime (the deprecated `created_date` field is being removed). Sorting purely by `created` for a same-day batch of documents gives no stable, meaningful order, and later phases that interleave chat/email (which have full timestamps) will have documents clump at midnight.
**Why it happens:** This is a real, recent API behavior change in paperless-ngx (not the old assumption); code/docs written against the pre-v9 datetime `created` field will misbehave.
**How to avoid:** Use `created` as the primary sort key (it reflects the document's real-world date, which is what a "topic timeline" should show), but use `added` (a full datetime — when the doc was ingested) as a secondary/tie-break sort key and to give the UI a stable millisecond-level `timestamp_unix` even on same-day documents. Document this choice explicitly since it's a Claude's-Discretion area.
**Warning signs:** Multiple documents added on the same day render in an order that changes between syncs, or that doesn't match file content order.

### Pitfall 5: cgo leaking into the kernel module
**What goes wrong:** If the paperless plugin (or any future dependency) is developed inside the same Go module as the kernel, and something transitively pulls in a cgo package, `go build ./...` for the whole repo now requires a C toolchain — defeating the "kernel binary has zero cgo requirement for users who don't need Signal" goal from `.claude/CLAUDE.md`.
**Why it happens:** Easiest to start everything in one `go.mod` during early development; splitting later means import-path surgery.
**How to avoid:** Set up the `go.work` multi-module layout (Recommended Project Structure, above) from the first commit — kernel module, plugin module(s) — even with only one plugin this phase.
**Warning signs:** `CGO_ENABLED=0 go build ./...` starts failing once a plugin dependency changes, or cross-compilation for another OS/arch stops working.

## Code Examples

Verified patterns from official sources (see Architecture Patterns above for full context):

### paperless-ngx: resolve keywords to tag IDs, then fetch documents (OR-matched)
```
# Source: verified against paperless-ngx src/documents/filters.py (main branch, fetched 2026-07-27)
# Step 1 — resolve each webspace keyword to a tag ID (case-insensitive exact match)
GET /api/tags/?name__iexact=house-move
Authorization: Token <token>
Accept: application/json; version=10

# Step 2 — fetch documents matching ANY resolved tag ID (confirmed OR / __in semantics)
GET /api/documents/?tags__id__in=12,47
Authorization: Token <token>
Accept: application/json; version=10
```

### paperless-ngx: exact deep link (PLUG-03 "exact" fidelity)
```
# Source: verified against paperless-ngx src-ui/src/app/app-routing.module.ts (main branch)
# Frontend route pattern: 'documents/:id' (and 'documents/:id/:section')
https://<paperless-host>/documents/{id}
```

### paperless-ngx: live content fetch on item-open (KERN-03)
```
# Source: verified against paperless-ngx docs/api.md (main branch)
GET /api/documents/{id}/preview/     # inline preview
GET /api/documents/{id}/download/    # original file
GET /api/documents/{id}/thumb/       # thumbnail
# all accept an optional ?version={version_id} query param
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| paperless-ngx `created` as a full datetime | `created` is a **date only**; `created_date` is deprecated | paperless-ngx API v9+ (confirmed current default is v10 in docs fetched 2026-07-27) | Chronological ordering logic must use a secondary datetime field (`added`) for stable, precise ordering — see Pitfall 4 |
| SvelteKit SPA fallback named `index.html` | Named `200.html` (or platform-specific), explicitly *not* `index.html` | Current SvelteKit docs (`svelte.dev/docs/kit/single-page-apps`, fetched 2026-07-27) | Following an older tutorial here silently breaks prerendering interplay |
| go-plugin under HashiCorp | go-plugin's copyright header now reads "IBM Corp. 2016, 2026" (SPDX MPL-2.0 unchanged) | Recent releases (v1.8.0, 2026-04-29) | Confirms active maintenance continued post-HashiCorp/IBM changes; no code-level impact, just a maintenance-continuity signal worth knowing |

**Deprecated/outdated:**
- paperless-ngx `created_date` field: deprecated, being removed in a future API version — don't build against it even though some older third-party clients/tutorials reference it.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The `.proto` message shapes in Pattern 2 (`Item`, `LinkFidelity`, `MatchRequest`/`FetchRequest`) are a workable first sketch, not verified against any existing webspaces code (none exists — greenfield) | Architecture Patterns, Pattern 2 | If the planner locks this exact shape without revisiting it against a second hypothetical source (chat) as the phase notes require, a real structural mismatch (e.g. chat needs a "thread_id" grouping concept documents don't) could force a breaking contract change in Phase 2 |
| A2 | Recommending `created` (primary) + `added` (secondary) as the chronological sort key pair for paperless documents | Common Pitfalls, Pitfall 4 | This is a Claude's-Discretion area per CONTEXT.md, so it's explicitly meant to be a recommendation, not a locked fact — but if the user has a different mental model of "chronological" (e.g. always by `added`/ingestion time) the stream order will look wrong to them |
| A3 | `pelletier/go-toml/v2` recommended over `BurntSushi/toml` for the config loader | Standard Stack / Alternatives Considered | Low risk — both are legitimate, verified libraries; a switch later is a config-loader-only change, not a downstream contract change |
| A4 | The `.proto`-level `Health` RPC is sketched now even though PLUG-04 (health reporting) is a Phase 2 requirement | Architecture Patterns, Pattern 2 | Minor — if the planner decides not to reserve the RPC number now, Phase 2 just adds a new RPC; not a breaking risk either way |

## Open Questions

1. **What exactly should the "preview" text be for a paperless document?**
   - What we know: paperless-ngx returns full extracted `content` (OCR'd text) in the document detail response; the Phase 1 notes leave "what is stored as the local preview" to Claude's discretion.
   - What's unclear: whether to truncate `content` to N characters, or use some other summary (e.g. first line / title only) — this affects both index storage size and stream UI design (UI-03).
   - Recommendation: Truncate `content` to a fixed character budget (e.g. 280–500 chars) at sync time and store that truncated string as `items.preview`; this keeps KERN-03's "metadata + preview only" boundary honest and gives the stream something meaningful to render without fetching full content.

2. **Where does the plugins directory live, and how are plugin binaries discovered?**
   - What we know: `.claude/CLAUDE.md`'s "Stack Patterns by Variant" says plugin discovery should be "whatever plugin binaries are present in the plugins directory," not a compile-time list.
   - What's unclear: default path (`~/.config/webspaces/plugins/`? next to the kernel binary? both, in order?) — this is explicitly a "Claude's Discretion" item ("Running the service").
   - Recommendation: Default to a `plugins/` directory next to the kernel binary (simplest for a single-binary desktop tool), overridable via a `plugins_dir` key in `config.toml`; document this as a planning decision rather than researching further — it's a design choice, not a technical unknown.

## Environment Availability

| Dependency | Required By | Available (this dev sandbox) | Version | Fallback |
|------------|------------|-------|---------|----------|
| Go toolchain | Kernel build | ✓ | go1.26.5 | — |
| Node.js / npm | SvelteKit build | ✓ | node v26.5.0 / npm 12.0.1 | — |
| `protoc` | `.proto` codegen | ✓ | libprotoc 35.1 | — |
| `buf` CLI | `.proto` codegen (recommended path) | ✗ | — | Use `protoc` + `protoc-gen-go` + `protoc-gen-go-grpc` directly, or `go install github.com/bufbuild/buf/cmd/buf@v1.72.0` before planning starts |
| `protoc-gen-go` / `protoc-gen-go-grpc` | `.proto` codegen (protoc path) | ✗ | — | `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest` and `go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest` |
| paperless-ngx instance (LAN, home server) | SRC-04 end-to-end verification | Not reachable from this research sandbox (`curl` to a placeholder host returned no response, as expected — this sandbox is not the user's desktop/LAN) | — | Must be verified on the user's actual desktop machine during planning/execution, not assumable from this research session |

**Missing dependencies with no fallback:**
- None — every missing tool above has a documented install path or CLI fallback.

**Missing dependencies with fallback:**
- `buf` — install before/at start of execution, or use the already-present `protoc` + Go plugin binaries.
- Live paperless-ngx reachability — this must be confirmed by the user/planner against the real home-server instance; this research session could not verify network reachability since it doesn't run on the target desktop/LAN.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | No (this phase has no end-user auth — single local user, no login flow) | — |
| V3 Session Management | No | — |
| V4 Access Control | Partial — plugin contract must be read-only by construction (PLUG-02) | Enforce at the `.proto` level: no mutating RPC defined at all, not just "unused" — the contract itself has no `Update`/`Delete`/`Create` method for source data |
| V5 Input Validation | Yes — webspace keywords from `config.toml`, and paperless API responses, both cross a trust boundary into the kernel | Validate keyword strings (non-empty, reasonable length) at config-load time; treat all paperless-ngx API response fields as untrusted input when building `Item` structs (don't trust `content-type`/size assumptions blindly) |
| V6 Cryptography | Partial — the paperless-ngx API token is a bearer credential | Never write the token to `config.toml` (D-04 already requires this); load only from env var; ensure it's never logged (including in `go-hclog` plugin log passthrough) |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| Plugin subprocess given more capability than the contract allows (e.g. a plugin author adds a mutating method later) | Elevation of Privilege | `.proto` contract has no mutating RPC defined; enforce this at code-review/contract-design time (PLUG-02 is a design constraint, not a runtime check) |
| paperless-ngx API token leaked via logs (plugin stdout/stderr piped through `go-hclog`) | Information Disclosure | Scrub `Authorization` header values before any log line touches `go-hclog`; never log full HTTP request objects verbatim |
| Kernel HTTP API (AGENT-02) exposed on a LAN-reachable interface without auth | Spoofing / Information Disclosure | Bind the kernel's `go-chi` HTTP server to `127.0.0.1` by default for Phase 1 (single local user, no remote clients yet); document this as the default and flag any future LAN-exposure decision as its own security review |
| Self-signed / untrusted TLS from paperless-ngx over LAN | Tampering (MITM) | Not applicable this phase if paperless-ngx is plain HTTP over LAN as configured; if the home server actually terminates TLS with a self-signed cert, the Go `net/http` client must be configured with an explicit trusted cert pool, not blanket `InsecureSkipVerify` |

## Sources

### Primary (HIGH confidence — verified live in this session, 2026-07-27)
- Go module proxy: `github.com/hashicorp/go-plugin@v1.8.0` (2026-04-29), `modernc.org/sqlite@v1.54.0` (2026-07-15), `github.com/go-chi/chi/v5@v5.3.1` (2026-07-05), `google.golang.org/grpc@v1.82.1`, `google.golang.org/protobuf@v1.36.11`, `github.com/pelletier/go-toml/v2@v2.4.3`, `github.com/hashicorp/go-hclog@v1.6.3`, `github.com/bufbuild/buf@v1.72.0`
- npm registry: `svelte@5.56.8`, `@sveltejs/kit@2.70.1`, `@sveltejs/adapter-static@3.0.10`, `vite@8.1.5` — all checked for `postinstall` scripts (none found) and source-repo linkage
- `raw.githubusercontent.com/hashicorp/go-plugin/main/examples/grpc/shared/interface.go`, `.../shared/grpc.go`, `.../plugin-go-grpc/main.go`, `.../main.go`, `.../buf.gen.yaml`, `.../buf.yaml` — fetched directly, code quoted verbatim
- `raw.githubusercontent.com/paperless-ngx/paperless-ngx/main/src/documents/filters.py` — fetched directly; confirms `TagFilterSet` (`name`: `iexact`/`icontains`/etc., no `__in`) and `ObjectFilter`/`tags__id__in` OR semantics
- `raw.githubusercontent.com/paperless-ngx/paperless-ngx/main/src-ui/src/app/app-routing.module.ts` — fetched directly; confirms `'documents/:id'` and `'documents/:id/:section'` frontend routes

### Secondary (MEDIUM confidence)
- `svelte.dev/docs/kit/single-page-apps` (WebFetch, 2026-07-27) — adapter-static SPA fallback (`200.html`) configuration
- `raw.githubusercontent.com/paperless-ngx/paperless-ngx/main/docs/api.md` (WebFetch, 2026-07-27) — token auth header format, document detail/preview/download/thumb endpoints, pagination shape, API versioning via `Accept` header, `created` field date-only note

### Tertiary (LOW confidence)
- General WebSearch results on paperless-ngx frontend architecture / DeepWiki summaries — used only to locate the primary sources above, not cited as fact on their own
- WebSearch on Go TOML env-var interpolation patterns — corroborates the `os.ExpandEnv` approach but is community-blog-sourced, not official Go documentation (the underlying `os.ExpandEnv` stdlib function itself is authoritative; the "use it before TOML unmarshal" pattern is community convention)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — every package version was checked live against its authoritative registry (Go module proxy / npm) in this session, not recalled from training data
- Architecture: HIGH — the go-plugin gRPC pattern and paperless-ngx API/routing details are quoted from live-fetched upstream source files, not inferred
- Pitfalls: MEDIUM-HIGH — most are grounded in verified source behavior (paperless `created` field change, SvelteKit fallback naming, go-plugin OR semantics); the cgo-isolation and correlation-timing pitfalls are architectural reasoning consistent with `.claude/CLAUDE.md`, not independently re-verified this session

**Research date:** 2026-07-27
**Valid until:** 30 days for the Go/npm package versions (fast-moving ecosystem, re-check before a long-delayed execution); the paperless-ngx API contract details (filters, routes) are more stable but should be re-confirmed against the user's actual paperless-ngx version during planning since "current default is v10" per the fetched docs could differ from the home server's installed version.
