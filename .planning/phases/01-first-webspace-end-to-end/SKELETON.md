# Walking Skeleton — Webspaces

**Phase:** 1
**Generated:** 2026-07-27

## Capability Proven End-to-End

A user defines a `house-move` webspace in `~/.config/webspaces/config.toml`, runs `webspaces serve`, opens `http://127.0.0.1:7777/` in a browser, clicks the webspace, and sees their *real* paperless-ngx documents tagged `house-move` or `House` as one chronological stream — each row carrying a working deep link back to `{paperless}/documents/{id}`.

Every architectural seam the whole project depends on is exercised on that one path: TOML config → SQLite index → gRPC subprocess plugin → paperless-ngx REST over LAN → correlation persisted at sync time → kernel HTTP JSON API → embedded SvelteKit SPA.

## Architectural Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Kernel language | Go 1.23+ (local toolchain go1.26.5), pure-Go, `CGO_ENABLED=0` | Locked in `.claude/CLAUDE.md`; only language where Phase 5's WhatsApp linked-device requirement is a solved problem (`whatsmeow`). Kernel stays cgo-free so Phase 4's SQLCipher requirement never leaks into it. |
| Repo shape | Go **workspace** (`go.work`) with three modules from the first commit: root (kernel + `cmd/`), `sdk/`, `plugins/paperless/` | RESEARCH Pitfall 5. Phase 4's Signal plugin needs cgo; establishing the multi-module boundary now means Phase 4 *adds a module* instead of restructuring the repo. |
| Plugin transport | `hashicorp/go-plugin` v1.8.0, gRPC only (`AllowedProtocols` restricted to `ProtocolGRPC`) | Battle-tested subprocess supervision + handshake + crash isolation. gRPC (not net/rpc) because Phases 3–5 need server streaming for large mailbox/chat volumes. |
| Published contract | `proto/webspaces/v1/plugin.proto`, generated stubs vendored into the **`sdk/` module** that both kernel and every plugin import | The `.proto` + `sdk/` *is* the third-party plugin contract (PLUG-05, Phase 2). A third party imports one module, not the kernel. |
| Read-only enforcement | Contract declares only `Describe` / `Match` / `Fetch` / `Health`. No mutating RPC exists at all — enforced by a contract test, not by convention | PLUG-02 is read-only *by construction*. `Out of Scope` in REQUIREMENTS.md excludes source writes permanently. |
| Index / data layer | `modernc.org/sqlite` v1.54.0 (pure Go, FTS5 compiled in), single file at `~/.local/share/webspaces/index.db`, WAL mode, `items` kept rowid-compatible | Cgo-free single static binary. Schema is FTS5-external-content-ready so Phase 3's KERN-05 bolts on without a migration. |
| Correlation model | Resolved at **sync time** into a `webspace_items` join table. `kernel/httpapi/stream.go` takes only an `index.Store` and never references the plugin host | KERN-02, RESEARCH Pitfall 1. The package boundary makes the violation structurally impossible, not merely discouraged. |
| Webspace identity | A webspace has a **name** (`[webspaces.house-move]`) and a **list of keywords** (`["house-move", "House"]`). All keywords matched in all plugins; exact, case-insensitive; no per-source override | D-02 / D-03. RESEARCH's sketch keyed the join table on `webspace_keyword`; the correct model per the locked decision is `webspace_items(webspace_name, item_id)`. |
| Config | One TOML file at `~/.config/webspaces/config.toml`; `os.ExpandEnv` over the raw bytes, then `pelletier/go-toml/v2` decode. Secrets only as `${VAR}` references | D-01 / D-04. Stdlib expansion, no Viper, no templating engine. |
| Sync trigger (Phase 1 only) | Startup sync in a background goroutine + a one-shot `webspaces sync` CLI subcommand. **No** scheduler, **no** HTTP-triggered refresh | Claude's Discretion. The per-plugin coordinator and manual refresh are KERN-04 / Phase 2 — deliberately not poached. |
| Chronological order | `timestamp_unix` DESC (from paperless `created`, a date-only field as of API v9+), tie-broken by `secondary_timestamp_unix` DESC (from `added`, full datetime), then `id` ASC | RESEARCH Pitfall 4 / assumption A2. The `Item` message carries both timestamps so chat/email in later phases get a total, stable interleave. |
| Local "preview" | Whitespace-collapsed first 500 runes of paperless `content` (OCR text), stored in `items.preview`. Full text and file bytes are **never** persisted | KERN-03 hybrid boundary + the privacy prohibition. |
| Web UI | SvelteKit 2 / Svelte 5 SPA, `adapter-static` with `fallback: '200.html'` (never `index.html`), built into `kernel/webui/build/`, served via `go:embed all:build` behind one catch-all chi route | RESEARCH Pattern 4 + Pitfall 3. One static binary; `vite dev` proxies `/api` to the kernel during development. |
| Design system | shadcn-svelte (new-york / slate / CSS variables), Tailwind v4 Vite plugin, `bits-ui`, `lucide-svelte`, Inter variable. Dark-mode-only via `class="dark"` on `<html>`, no theme toggle | `01-UI-SPEC.md`, approved 6/6 dimensions 2026-07-27. |
| HTTP surface | `go-chi/chi/v5`, bound to **`127.0.0.1:7777`** by default. Same JSON routes serve the SPA and any agent — no separate "agent API" | AGENT-02. Loopback default is the security posture for a single-user local tool with no auth; LAN exposure is its own future security review. |
| Module path | `github.com/darrendavison/webspaces` (+ `/sdk`, `/plugins/paperless`) | No git remote is configured yet. Rated `costly`, not `one-way`: changeable via `go mod edit -module` + an import rewrite at any point before Phase 2 publishes the SDK to a third party. |

## Stack Touched in Phase 1

- [x] Project scaffold — `go.work` + 3 Go modules, `buf`-generated protobuf stubs, SvelteKit + Tailwind v4 + shadcn-svelte, `Makefile`, `golangci-lint`
- [x] Routing — kernel: `GET /api/webspaces`, `GET /api/webspaces/{webspace}/stream`, `GET /api/items/{id}`, `GET /api/items/{id}/content`, `GET /api/items/{id}/thumbnail`, SPA catch-all. Client: `/` and `/w/[webspace]`
- [x] Database — real write (sync upserts `items` + `webspace_items` + `sync_runs` in one transaction) AND real read (the stream query, index-only)
- [x] UI — a real interaction wired to the API: the webspace list at `/` is fetched from `GET /api/webspaces`; clicking one navigates to `/w/{name}` which fetches and renders the real stream; clicking a row opens the detail pane against `GET /api/items/{id}`
- [x] Deployment — documented local full-stack run: `make build && ./bin/webspaces serve` (single binary, embedded UI); `make dev` for the split Vite/kernel loop

## Out of Scope (Deferred to Later Slices)

Explicitly **not** in the skeleton — later phases must not treat these omissions as bugs:

- Any second source (SilverBullet, IMAP, Signal, WhatsApp) — Phases 2–5
- Source filter chips (UI-02) and per-source health / sync-status UI (UI-05, UI-06) — Phase 2
- Sync scheduler, per-plugin coordinator, refresh de-duplication, HTTP/manual refresh (KERN-04) — Phase 2
- Full-text search over the index (KERN-05, FTS5 tables and triggers) — Phase 3. The schema is *prepared* for it (`items` stays rowid-compatible) but no FTS5 object is created
- Agent permission model, per-plugin default-deny grants (AGENT-01) — Phase 2
- Third-party plugin authoring guide beyond the published `.proto` + generated SDK (PLUG-05 validation) — Phase 2
- Any write, edit, compose, or reply capability against any source — out of scope permanently
- Stream list virtualization — deferred until real volume demands it (per UI-SPEC zero-one-many row)
- Multi-user, remote access, auth, TLS on the kernel's own listener — the tool is single-user and loopback-bound

## Subsequent Slice Plan

Each later phase adds one vertical slice on top of this skeleton without altering its architectural decisions:

- **Phase 2:** a second, structurally different source (SilverBullet) joins the same stream, proving the contract is source-agnostic; plus filter, health, refresh, staleness and default-deny agent permissions
- **Phase 3:** IMAP/Proton email in the stream (never marked read), plus FTS5 search within a webspace
- **Phase 4:** Signal Desktop conversations, read strictly read-only with runtime keyring-backend detection
- **Phase 5:** WhatsApp via a `whatsmeow` linked device with its own persistent store, degrading gracefully on de-link or ban
