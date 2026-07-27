# Stack Research

**Domain:** Local-first, plugin-extensible personal data-aggregation service with a web UI (kernel + source plugins + stream/detail-pane frontend)
**Researched:** 2026-07-27
**Confidence:** HIGH (kernel language, IMAP, local index) / MEDIUM (plugin RPC mechanism, Signal/WhatsApp specifics, SilverBullet HTTP API detail) / LOW (WhatsApp desktop-store approach — no official API, landscape shifts)

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Go | 1.23+ (current stable toolchain) | Kernel language: HTTP server, sync engine, plugin host, embeds web UI | The single hard constraint in this project is `whatsmeow` (go.mau.fi/whatsmeow), the only mature, actively-maintained WhatsApp linked-device library — and it's Go-only. Writing the kernel in a different language would force a second runtime just for WhatsApp. Go also has the best story for everything else here: native cgo access to link SQLCipher (needed for Signal), a mature IMAP client (`go-imap`), first-class concurrency for N long-lived source connections (IMAP IDLE, WhatsApp WebSocket) running concurrently, single static-binary deployment (`go:embed` the built web UI directly into the kernel binary — one executable to run on the desktop), and the standard-library `net/http` is sufficient for the API surface this project needs. **This is not a close call** — Go is the only language where the riskiest requirement (WhatsApp) is a solved problem. |
| SQLite (via `modernc.org/sqlite` v1.34+, pure Go) | bundled SQLite ~3.46+ | Local metadata/preview index (the "hybrid data model" store) | Single-file embedded DB matching the local-first, single-user, no-server-process deployment model. `modernc.org/sqlite` is a cgo-free transpile of the real SQLite C source, so the kernel binary stays a single static executable with no libsqlite3 system dependency and cross-compiles trivially — important because the *Signal plugin* already needs cgo (for SQLCipher) and you don't want that requirement to bleed into the kernel's own storage layer. It ships FTS5, JSON1, and RTree compiled in. |
| FTS5 (external-content virtual table) | bundled with SQLite | Full-text search over synced previews (email subject/body preview, chat message text, note/doc titles) across all sources | Standard SQLite full-text pattern: keep item metadata in a normal table, create an FTS5 table with `content='items', content_rowid='id'` plus triggers to keep it in sync on insert/update/delete. Gives cross-source keyword search "for free" without duplicating full content (full content stays in the source, per the hybrid model) and without adding an external search service (Elasticsearch/Meilisearch) that would be pure operational overhead for a single-user desktop tool. |
| `go-imap/v2` client — actually recommend **`go-imap` v1** (`github.com/emersion/go-imap` v1.x) for the client, not v2 | v1: latest tagged release (`v1.2.1` line, actively used); v2 is beta | IMAP client for the email plugin, against Proton Mail Bridge (and generically any IMAP server) | `emersion/go-imap` is the de facto standard Go IMAP library (used by Delta Chat, aerc, and dozens of other clients/bridges). v2 (IMAP4rev2, RFC 9051) is real but **still in beta as of this research** — the maintainers themselves say "not recommended for critical workloads" and the API isn't fully stable. v1 (IMAP4rev1) is what Proton Mail Bridge and virtually every IMAP server actually speak, is production-proven, and is what you should build against now. Revisit v2 once it hits GA — the sync engine's plugin-facing contract should be written so swapping the underlying IMAP client later is an internal detail, not a plugin-interface change. |
| `go.mau.fi/whatsmeow` | latest tagged (actively published, imported by 300+ projects incl. the Mautrix bridges) | WhatsApp linked-device plugin | The only viable option for reading a personal WhatsApp account's chat history without an official Business API subscription. It implements the multi-device web protocol client-side, is what powers `mautrix-whatsapp` (a bridge running 24/7 in production for thousands of Matrix users), and is far more stable across WhatsApp's protocol churn than the Node.js alternative (`Baileys` — see "What NOT to Use"). Reverse-engineered and against WhatsApp's ToS like any personal-use WhatsApp automation; accept that risk knowingly (already implicit in the project's chosen approach). |
| SQLCipher access via **`mutecomm/go-sqlcipher/v4`** (cgo) | v4.4.2 | Read (never write) Signal Desktop's local `db.sqlite` | Signal Desktop's message store is a SQLCipher-encrypted SQLite database. `mutecomm/go-sqlcipher/v4` is a self-contained `database/sql` driver that statically links SQLCipher + OpenSSL — no need to separately install `libsqlcipher` on the desktop. Pass the key via DSN, e.g. `sql.Open("sqlite3", path+"?_pragma_key=x'<hexkey>'&_pragma_cipher_page_size=4096")`. It has low commit velocity (SQLCipher's on-disk format is stable, so this is expected, not concerning) but is still the most self-contained option; if it stops building against a newer SQLCipher/OpenSSL, the fallback is `xeodou/go-sqlcipher` (same driver shape, more recent forks exist). **This is inherently cgo** — isolate it behind the Signal plugin process (see Plugin Architecture below) so a crash or build issue there can't take down the kernel or other plugins, and so users who don't use Signal never need a C toolchain to build the kernel. |
| Signal key retrieval: `libsecret`/D-Bus Secret Service call (no dedicated Go lib needed — thin wrapper over `github.com/godbus/dbus/v5` or shell out to `secret-tool`) | — | Unwrap the SQLCipher key that Signal Desktop stores | On current Signal Desktop (Electron `safeStorage`), the raw key is no longer plaintext in `config.json` on Linux — it's protected via the OS-native secret store. On Arch/GNOME that's GNOME Keyring via `libsecret`/freedesktop Secret Service API; on KDE it's KWallet. Fetch the wrapped blob from `~/.config/Signal/config.json` (`encryptedKey`/`safeStorageBackend` fields), then unwrap it via a D-Bus call to the Secret Service (`org.freedesktop.secrets`) using `godbus/dbus`, matching Electron's own `safeStorage` scheme (AES-GCM with a key stored under a well-known libsecret attribute). This is a genuine "deeper research" item — flag it for its own phase; DE-dependent behavior (GNOME vs KDE) needs to be validated against whatever the user actually runs. |
| Plugin transport: **`hashicorp/go-plugin`** (gRPC-over-subprocess) | v1.7.0 | Kernel⇄plugin RPC contract; each source plugin is a separately-built, separately-launched subprocess | See "Plugin Architecture" discussion below — this is the most consequential decision in the stack. |
| SvelteKit | Svelte 5 / SvelteKit 2.x, `@sveltejs/adapter-static` (SPA mode) | Web UI: stream + detail-pane app | See "Web UI Framework" discussion below. |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `go-chi/chi` | v5.x | HTTP router for the kernel's JSON API | Thin, stdlib-compatible (`net/http` handlers), no magic — appropriate for a single-service API with a handful of routes (webspace list, item stream, item detail, search, plugin health). Avoids pulling in a heavier framework (Gin/Echo) for no benefit at this scale. |
| `google.golang.org/grpc` + `google.golang.org/protobuf` | current (grpc-go v1.6x) | Wire format for `go-plugin`'s RPC layer | Required transitively by `go-plugin`'s gRPC plugin type (preferred over its legacy net/rpc type — gRPC gives you streaming, which the sync engine needs for large mailboxes/chat histories). |
| `hashicorp/go-hclog` | current | Structured logging surfaced from plugin subprocesses back to the kernel | Ships with `go-plugin`'s expected logging interface; plugin stdout/stderr gets parsed and re-emitted through this so kernel logs and plugin logs interleave sanely instead of being separate log streams to tail. |
| `mattn/go-sqlite3` (build tag `fts5`) | v1.14.x | **Only if** you need cgo SQLite specifically inside a plugin process (e.g. co-locating with SQLCipher linkage) | The kernel itself should use `modernc.org/sqlite` (pure Go, no cgo) for its own index. But if a plugin process already links cgo (Signal plugin does, for SQLCipher), and that plugin wants a local scratch SQLite file too, `mattn/go-sqlite3` is fine there since the cgo cost is already paid. Don't use it for the kernel's index — no reason to force cgo on the whole binary. |
| `godbus/dbus/v5` | v5.1.x | Talk to the freedesktop Secret Service (libsecret) to unwrap Signal's DB key | Signal-plugin-only dependency; see key retrieval note above. |
| `emersion/go-message` | latest | MIME parsing for fetched email bodies (multipart, attachments, encoding) | Pairs with `go-imap` — IMAP gives you raw RFC822 bytes, `go-message` parses them into usable parts for the preview/detail view. |
| `paperless-ngx` client: hand-rolled thin REST client over `net/http` (no dedicated Go SDK needed) | Paperless-ngx API v9/v10 | Fetch document metadata + tags for the paperless plugin | Paperless-ngx's REST API is small, stable, and well-documented (token auth via `/api/token/`, `Authorization: Token <token>` header, versioned via `Accept` header). There's no actively-maintained official Go client; a Python client (`pypaperless`) exists but isn't usable from Go. Given the API surface you need (documents by tag, metadata, thumbnail/preview URL) is ~4 endpoints, hand-rolling a typed client is less risk than adopting a third-party wrapper. |
| SilverBullet client: hand-rolled thin REST client over `net/http` | SilverBullet's `/.fs` HTTP API | List/read space files (markdown notes) and their frontmatter/tags for the SilverBullet plugin | SilverBullet exposes its space filesystem directly over HTTP: `GET /.fs` lists all files with metadata, `GET /.fs/<path>` and `PUT/DELETE /.fs/<path>` read/write individual files (you only need GET — view-only). Auth is a Bearer token (`SB_AUTH_TOKEN`) or session cookie. No client library exists or is needed — this is a handful of `net/http` calls plus your own markdown-frontmatter/tag parsing (e.g. `github.com/adrg/frontmatter` or a small hand-rolled parser, since SilverBullet's tag/attribute conventions are simple `key: value` YAML-ish frontmatter plus inline `#tag` scanning). |
| `sveltejs/svelte-virtual` (or hand-rolled windowing) | latest | Virtualize the cross-source stream list | The stream view is a chronological, potentially-long, cross-source feed — render only visible rows to keep scroll smooth as the local index grows across years of email/chat history. |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| `air` (cosmtrek/air or air-verse/air fork) | Live-reload for the Go kernel during development | Rebuilds/restarts the kernel binary on file change; pair with Vite's own dev server for the SvelteKit side (don't embed the built UI during dev — proxy to Vite instead, only `go:embed` the production build). |
| `sqlc` or plain `database/sql` + hand-written SQL | Typed query layer over the local index DB | Given the schema is small and stable (items, sources, webspaces), hand-written SQL with `database/sql` is defensible; `sqlc` is worth adopting once the schema stabilizes past MVP if you want compile-time-checked queries. |
| `golangci-lint` | Go linting | Standard for any nontrivial Go project; enable `errcheck` particularly given how many I/O boundaries (IMAP, D-Bus, gRPC, HTTP) this project touches. |
| `buf` | Protobuf tooling for the `go-plugin` gRPC service definitions | Generates the Go stubs for the kernel⇄plugin contract from a `.proto` file — this **is** the documented plugin contract mentioned in the project's requirements; check the `.proto` into the repo as the source of truth for third-party plugin authors. |

## Installation

```bash
# Kernel module init
go mod init github.com/<you>/webspaces

# Core
go get modernc.org/sqlite@latest
go get github.com/emersion/go-imap@v1
go get github.com/emersion/go-message@latest
go get go.mau.fi/whatsmeow@latest
go get github.com/mutecomm/go-sqlcipher/v4@latest   # Signal plugin module only (separate go.mod recommended)
go get github.com/hashicorp/go-plugin@latest
go get google.golang.org/grpc@latest
go get google.golang.org/protobuf@latest
go get github.com/hashicorp/go-hclog@latest
go get github.com/go-chi/chi/v5@latest
go get github.com/godbus/dbus/v5@latest          # Signal plugin module only

# Frontend
npx sv create webspaces-ui   # SvelteKit 2.x / Svelte 5 scaffold
cd webspaces-ui
npm install -D @sveltejs/adapter-static
npm install svelte-virtual   # or equivalent windowing lib
```

**Module layout note:** because the Signal plugin needs cgo (SQLCipher) and the kernel/other plugins should stay cgo-free, structure this as a Go workspace (`go.work`) with separate modules per plugin binary (`./kernel`, `./plugins/email`, `./plugins/signal`, `./plugins/whatsapp`, `./plugins/paperless`, `./plugins/silverbullet`), each built and shipped as its own executable that the kernel launches via `go-plugin`. Only the Signal plugin's build needs `CGO_ENABLED=1` and a C toolchain.

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|--------------------------|
| Go kernel | Node.js/TypeScript kernel + `Baileys` for WhatsApp | Only if you're willing to accept `Baileys`' materially worse stability (frequent breaking forks, higher susceptibility to WhatsApp protocol changes) in exchange for a single-language stack with the SvelteKit frontend. Given whatsmeow's proven track record (Mautrix bridges), this tradeoff isn't worth it here. |
| `hashicorp/go-plugin` (gRPC-over-subprocess) | Plain HTTP/JSON sidecar processes (kernel spawns each plugin binary, plugin exposes a small REST API on a Unix domain socket) | If third-party plugin authors writing in non-Go languages is a near-term priority over battle-tested process supervision. HTTP/JSON is a lower-ceremony contract to implement in any language (no protobuf toolchain required) but you'd be re-implementing handshake, health-checking, and process lifecycle management yourself — `go-plugin` already solved this (it's what Terraform/Vault/Nomad providers use). For a v1 with plugins you're writing yourself, `go-plugin`'s Go-native ergonomics (interface → generated stubs) outweigh the extra ceremony. Reconsider the HTTP/JSON option specifically at the point a real third-party plugin author shows up and finds the protobuf/gRPC contract to be the adoption blocker. |
| `hashicorp/go-plugin` | Extism/WASM (`extism/go-sdk` + `extism/go-pdk`) | **Not recommended for this project at all** — see "What NOT to Use." Only reconsider if a future plugin type is pure data transformation with no need for raw sockets, cgo, or long-lived stateful connections (i.e., not any of the current five sources). |
| `modernc.org/sqlite` (pure Go) for kernel index | `mattn/go-sqlite3` (cgo) for kernel index | If you later need a SQLite extension only available in mattn's cgo build (e.g., `sqlite-vec` for semantic/vector search, relevant once the "AI-inferred correlation" v2 layer is built) and are fine with the whole kernel binary requiring cgo. Cross this bridge in v2, not now. |
| SvelteKit (SPA mode) | htmx + Go `html/template` server-rendered UI | If you want to minimize frontend tooling entirely and are comfortable with more server-side state (selected item, active filters) round-tripped per request. Reasonable for a single-user tool, but the stream+detail-pane UX (independent scroll position, live-updating stream while a detail pane is open, client-side filter chips, virtualized long lists) is meaningfully easier to build with real client-side component state than with hypermedia swaps. If you find yourself fighting htmx for this specific pane-within-pane interaction model, that's the sign to switch. |
| SvelteKit | React/Next.js | Only if you have significantly more React experience than Svelte experience — there's no domain reason to prefer React here, and Next.js in particular brings server-rendering/routing machinery (App Router, RSC) that's pure overhead for a single-user tool that already has a real backend (the Go kernel) and just needs a build-time SPA bundle. |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|--------------|
| `Baileys` (Node.js WhatsApp library) | Frequent breaking forks and maintenance churn as WhatsApp changes its protocol; also forces a second language runtime alongside Go for no benefit. `whatsmeow` is the more stable, more battle-tested choice and is Go-native. | `go.mau.fi/whatsmeow` |
| Extism/WASM (`extism`) as the plugin mechanism | WASM sandboxing is the wrong tool here: two of your five plugins need capabilities WASM/WASI actively restricts or makes awkward — cgo linkage against SQLCipher/OpenSSL (Signal) and a long-lived raw WebSocket connection with persistent session state (WhatsApp). WASM's core value proposition (safely running untrusted third-party code) also doesn't apply yet: you're the only plugin author, running on your own machine, for view-only reads. Adopting WASM now buys sandboxing you don't need at the cost of fighting the sandbox for capabilities you do need. | `hashicorp/go-plugin` (subprocess + gRPC), each plugin a regular Go binary with full OS access |
| `go-imap/v2` | Still beta; maintainers explicitly say not production-ready; API still shifting. | `go-imap` v1 (IMAP4rev1) — production-proven, what Proton Mail Bridge and virtually all IMAP servers speak |
| `mattn/go-sqlite3` for the kernel's own local index | Forces `CGO_ENABLED=1` and a C toolchain onto the whole kernel binary just for the kernel's own metadata store, when nothing about that store needs cgo. Complicates cross-compilation and build reproducibility. | `modernc.org/sqlite` (pure Go, FTS5 included) for the kernel index; reserve cgo SQLite for the Signal plugin only, where it's unavoidable |
| Storing full email/chat/document content in the local index | Explicitly contradicts the project's hybrid data model (metadata+preview locally, full content fetched live from source on open) — and duplicates data you don't own (Signal/WhatsApp message plaintext, in particular, is sensitive and the whole point of the desktop-local-DB approach is to avoid a second copy sitting in a less-protected store). | Store enough to render the stream (subject/sender/snippet/timestamp/tags) and a stable per-source ID; fetch full content live through the plugin on item-open |
| An external search engine (Elasticsearch, Meilisearch, Typesense) for cross-source search | Pure operational overhead for a single-user desktop tool — a second long-running service to install, configure, and keep in sync, for a search workload SQLite FTS5 handles comfortably at personal-data scale. | SQLite FTS5 (external-content table) over the local index |
| Plaintext key extraction assuming Signal stores keys unencrypted in `config.json` | This was true in older Signal Desktop versions but is **not current behavior** — modern Signal Desktop wraps the key via Electron's `safeStorage`, backed by the OS secret store (libsecret/GNOME Keyring or KWallet on Linux). Code written against the old plaintext-key assumption will silently fail or read garbage on a current install. | Read `config.json`'s `encryptedKey`/`safeStorageBackend` fields and unwrap via a Secret Service (`org.freedesktop.secrets`) D-Bus call matching the DE's backend |

## Stack Patterns by Variant

**If the Signal or WhatsApp plugin needs to be disabled/omitted (e.g., building/running on a machine without those desktop apps installed):**
- Because each source plugin is its own subprocess/binary under `go-plugin`, the kernel should treat plugin discovery as "whatever plugin binaries are present in the plugins directory" rather than a compile-time list — this lets you build and ship a Signal-plugin-free kernel with zero cgo/C-toolchain requirement for users who don't need it, without touching kernel code.

**If/when the "AI-inferred correlation" v2 layer is built (explicitly out of scope for v1):**
- Use `sqlite-vec` (a loadable SQLite extension for vector search) alongside FTS5 for hybrid keyword+semantic search — but note this requires the cgo `mattn/go-sqlite3` build (or SQLite's loadable-extension mechanism) since `modernc.org/sqlite`'s pure-Go nature means it can't load native `.so` extensions. Budget for that migration cost at that point; don't pre-optimize for it in v1.

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|------------------|-------|
| `modernc.org/sqlite@v1.34+` | Go 1.21+ | Transpiled-C approach tracks upstream SQLite closely; pin a specific version and re-verify FTS5 trigger behavior on upgrade (virtual table internals occasionally shift between SQLite minor versions). |
| `hashicorp/go-plugin@v1.7.0` | `grpc-go` v1.6x, `google.golang.org/protobuf` v1.3x+ | Use the gRPC plugin type (not the legacy net/rpc type) for streaming support needed by large mailbox/chat-history syncs. |
| `mutecomm/go-sqlcipher/v4` | Requires `CGO_ENABLED=1` + gcc/clang; statically links its own OpenSSL/SQLCipher, so no system `libsqlcipher` needed | Build this plugin in its own Go module/workspace member so the cgo requirement doesn't propagate to `go build ./...` for the rest of the repo. |
| SvelteKit 2.x | Svelte 5, Vite 5/6, Node 20+ (build-time only — no Node runtime needed once `adapter-static` output is embedded in the Go binary) | Confirm `adapter-static`'s SPA fallback mode (`fallback: 'index.html'`) so client-side routing (stream → detail pane deep links) works when the Go kernel serves the embedded `index.html` for any unmatched UI route. |

## Sources

- pkg.go.dev / GitHub: `go.mau.fi/whatsmeow` (imported by 300+ projects, actively published) — HIGH
- pkg.go.dev / GitHub: `emersion/go-imap` (v1 stable, v2 beta status confirmed by maintainers) — HIGH
- GitHub: `hashicorp/go-plugin` (v1.7.0, used by Terraform/Vault/Nomad/Boundary/Waypoint/Packer) — HIGH
- GitHub/pkg.go.dev: `modernc.org/sqlite` (FTS5/RTree/JSON1/Session/GeoPoly confirmed compiled in) — HIGH
- GitHub: `mutecomm/go-sqlcipher/v4` (v4.4.2, DSN-based key pragma usage pattern) — MEDIUM (low commit velocity, but SQLCipher format stability makes this acceptable)
- GitHub issues (flathub/org.signal.Signal) + community blog (yingtongli.me, Aug 2025) on Signal Desktop's `safeStorage`/libsecret/KWallet key-wrapping behavior — MEDIUM (community-verified, no single official Signal doc covers Linux keyring specifics)
- SilverBullet official docs (`silverbullet.md/HTTP API`, `silverbullet.md/Architecture`) — HIGH for `/.fs` endpoint contract and Bearer-token auth
- Paperless-ngx official docs (`docs.paperless-ngx.com/api/`, `paperless-ngx/docs/api.md`) — HIGH for `/api/token/` auth and versioned API
- SQLite official docs (`sqlite.org/fts5.html`) — HIGH for external-content FTS5 pattern
- Community comparison articles (Baileys vs whatsmeow stability, htmx vs SvelteKit tradeoffs, 2025-2026) — MEDIUM (cross-checked across multiple independent sources, consistent conclusions)

---
*Stack research for: local-first personal cross-source data aggregation kernel + plugins + web UI*
*Researched: 2026-07-27*
