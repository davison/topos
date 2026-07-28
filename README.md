# Webspaces

Webspaces pulls together related information from your disparate personal
data silos — email, chats, document storage, notes, wikis — into per-topic
"webspaces," so you can open one view and see everything related to a
topic instead of hunting across apps. It runs entirely on your own
desktop, is read-only against every source it touches, and never sends
your data anywhere else.

## What this does not do yet

This is Phase 1 of a five-phase build. Right now: one source
(paperless-ngx), no source filtering, no search, no agent permission
model, no health UI. What's coming, in order:

- **Phase 2**: a second, structurally different source (SilverBullet)
  proves the plugin contract is source-agnostic, plus source
  filtering, per-plugin health, manual refresh, and a default-deny agent
  permission model.
- **Phase 3**: email (Proton/IMAP), plus full-text search across a
  webspace.
- **Phase 4**: Signal.
- **Phase 5**: WhatsApp.

## Prerequisites

- **Go 1.23+** (developed against 1.25).
- **Node 20+** (for building the SvelteKit web UI).
- The proto codegen toolchain (**`buf`**, or `protoc` +
  `protoc-gen-go` + `protoc-gen-go-grpc`) — only needed if you're
  regenerating `sdk/gen` from `proto/webspaces/v1/plugin.proto`; not
  needed for a normal build.

## Repository layout

```
cmd/webspaces/        kernel binary entrypoint (serve, sync)
kernel/                config, index (SQLite), correlate (sync-time matching), httpapi, pluginhost, webui (embed)
proto/webspaces/v1/    the published plugin contract (source of truth)
sdk/                    the plugin-author-facing Go module (handshake, interfaces, generated stubs)
plugins/paperless/     the reference source plugin
web/                    the SvelteKit SPA
docs/                   published contracts: plugin-contract.md, api.md
```

This is a **Go workspace** (`go.work`) with three separate modules: the
root kernel module, `sdk`, and `plugins/paperless`. That's deliberate, not
incidental — a future Signal plugin needs `cgo` (to link SQLCipher), and
keeping every plugin in its own module means that requirement stays
scoped to the one plugin that needs it. Building the kernel (or any other
plugin) never requires a C toolchain. Don't collapse these into one
module later without re-checking that isolation still holds.

## Configure

Webspaces needs two things from you: your paperless-ngx credentials in the
environment, and a config file describing your webspaces.

1. Set your paperless-ngx connection details as environment variables
   (put these in a `.env` file at the repo root for local development —
   it's gitignored — or export them in your shell):

   ```bash
   export PAPERLESS_URL="https://paperless.example.lan:8000"
   export PAPERLESS_TOKEN="<your paperless-ngx API token>"
   ```

2. Copy the example config and edit it:

   ```bash
   mkdir -p ~/.config/webspaces
   cp config.example.toml ~/.config/webspaces/config.toml
   ```

   `config.example.toml` is a fully-commented reference — every key
   documents its purpose, default, and validation rule. At minimum,
   define at least one `[webspaces.<name>]` block with a `keywords` list
   matching your own paperless-ngx tag names (matching is exact and
   case-insensitive — see the comments in the example file for the exact
   rule and a worked counterexample).

## Build and run

```bash
make build              # builds the SPA, the kernel binary, and the paperless plugin binary
./bin/webspaces sync    # runs one sync cycle against your configured sources
./bin/webspaces serve   # starts the kernel's HTTP API + embedded web UI
```

Then open `http://127.0.0.1:7777/w/<your-webspace-name>` in a browser.

The server binds `127.0.0.1` (loopback) only by default, and there is no
authentication on its HTTP API in v1 — that's the whole security boundary
for now. Binding it to a LAN interface is a deliberate decision this
project has not made; the server logs a warning at startup if it detects a
non-loopback bind, but does not refuse to start.

## Development loop

```bash
make dev
```

Runs the kernel (`go run ./cmd/webspaces serve`) and the SvelteKit dev
server together; Vite proxies `/api` requests to `127.0.0.1:7777`, so
edits to either side hot-reload independently. The kernel binary built
this way never embeds a built SPA — only `make build`'s production build
does that.

## Testing

```bash
make test               # go build + go test across all three workspace modules
make smoke              # make build, then scripts/e2e-smoke.sh
```

`make test` needs no network access or live credentials — every committed
test (including the PLUG-02/AGENT-02 contract tests in `sdk/`,
`plugins/paperless/`, and `kernel/httpapi/`) runs against fixtures and a
temp SQLite file. `make smoke` is different: it's a real end-to-end run
against your actual configured paperless-ngx instance (it needs
`PAPERLESS_URL`/`PAPERLESS_TOKEN` set, and a config file with at least one
webspace already matching real documents), so it needs network access to
your paperless-ngx instance and a live account.

## Where to look next

- **`docs/plugin-contract.md`** — the published contract for writing a new
  source plugin: the interface you implement, how the kernel discovers
  and launches your binary, how config reaches it, and what every `Item`
  field means.
- **`docs/api.md`** — the complete kernel HTTP JSON contract: every route,
  the stable-id scheme, the ordering guarantee, provenance keys, and the
  full error-code list. This is the same JSON the web UI consumes — there
  is no separate agent API.
- **`.planning/`** — this project's phase-by-phase requirements, design
  decisions, and roadmap.
