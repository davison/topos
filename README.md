# topos

topos pulls together related information from your disparate personal
data silos — email, chats, document storage, notes, wikis — into per-topic
"webspaces," so you can open one view and see everything related to a
topic instead of hunting across apps. It runs entirely on your own
desktop, is read-only against every source it touches, and never sends
your data anywhere else.

## Status and roadmap

Phases 1–7 of an eight-phase v1 are complete: four sources ship today —
paperless-ngx, SilverBullet, Proton Mail (via Bridge/IMAP, never marking
mail read), and Signal Desktop (read strictly read-only from its local
database) — interleaved in one chronological stream per webspace, with
source filtering, per-source health and manual refresh, full-text search,
a live detail pane, and a default-deny agent permission model. As of
Phase 5, a plugin can be configured more than once under distinct display
names ("Home email" / "Work email"), each instance's matching config is
typed to that plugin's own declared field vocabulary (folders, tags,
conversation names, pages) rather than one shared keyword list, and every
text/html rendition is sanitized, wrapped and themed by the kernel at one
content-serving boundary instead of by each plugin individually. Phase 6
gave the header a scalable source surface — one merged chip per instance,
deep-link fidelity cues, search-term highlighting, scrollbar date markers.
Phase 7 added a **webspace builder UI**: create/edit a webspace and its
source instances entirely from the browser (no more hand-editing TOML to
get started), promote a live search into a webspace's permanent filter,
and pull a hand-edited `config.toml` into the running kernel on demand via
a Reload config affordance — the config write path this introduced is the
kernel's first mutating HTTP surface, scoped strictly to configuration; see
`docs/plugin-contract.md`, `docs/api.md` and `config.example.toml` for the
shipped shape. What's coming next:

- **Phase 8**: WhatsApp, as a linked-device client with its own
  persistent message store — deliberately last, as the highest-risk
  source: everything above stays useful if it has to degrade.

## Prerequisites

- **Go 1.23+** (developed against 1.25).
- **Node 20+** (for building the SvelteKit web UI).
- The proto codegen toolchain (**`buf`**, or `protoc` +
  `protoc-gen-go` + `protoc-gen-go-grpc`) — only needed if you're
  regenerating `sdk/gen` from `proto/topos/v1/plugin.proto`; not
  needed for a normal build.

## Configure

topos needs two things from you: your paperless-ngx credentials in the
environment, and a config file describing your webspaces.

1. Set your source connection details as environment variables (put these
   in a `.env` file at the repo root for local development — it's
   gitignored — or export them in your shell):

   ```bash
   export PAPERLESS_URL="https://paperless.example.lan:8000"
   export PAPERLESS_TOKEN="<your paperless-ngx API token>"
   export SB_URL="https://silverbullet.example.lan:3000"
   export SB_AUTH_TOKEN="<your SilverBullet auth token>"
   ```

   This is the same `.env` file that `./scripts/run-with-env.sh` sources —
   put all four keys in one place and the wrapper picks them up.

2. Copy the example config and edit it:

   ```bash
   mkdir -p ~/.config/topos
   cp config.example.toml ~/.config/topos/config.toml
   ```

   `config.example.toml` is a fully-commented reference — every key
   documents its purpose, default, and validation rule. At minimum,
   define at least one `[webspaces.<name>]` block with a `keywords` list
   matching your own paperless-ngx tag names (matching is exact and
   case-insensitive — see the comments in the example file for the exact
   rule and a worked counterexample). `keywords` is a webspace-level
   fallback applied across every source instance's own declared match
   fields; a per-instance `[webspaces.<name>.match.<instance>]` block
   (also documented in `config.example.toml`) replaces that fallback for
   one instance when you need typed, per-source matching instead — e.g.
   distinct `folders` for two configured email instances.

## Build and run

```bash
make build              # builds the SPA, the kernel binary, and all five plugin binaries
./bin/topos sync    # runs one sync cycle against your configured sources
./bin/topos serve   # starts the kernel's HTTP API + embedded web UI
```

`make plugins` rebuilds only the plugin binaries (paperless, silverbullet,
proton, mock, and the cgo-enabled signal plugin) — useful when you only
changed plugin source and don't want to pay for a full SPA + kernel
rebuild.

If your credentials live in a repo-root `.env` (see Configure above), use the
wrapper instead of exporting them by hand:

```bash
./scripts/run-with-env.sh serve   # same, with .env exported into the process
```

`scripts/run-with-env.sh` sources the gitignored repo-root `.env`, exports
its keys, and `exec`s `bin/topos` with whatever arguments follow — so it
works from a fresh shell with nothing pre-exported, and any `topos`
subcommand works the same way (`./scripts/run-with-env.sh sync` as well as
`serve`). It prints no credential value on any code path.

Then open `http://127.0.0.1:7777/w/<your-webspace-name>` in a browser.

The server binds `127.0.0.1` (loopback) only by default, and there is no
authentication on its HTTP API in v1 — that's the whole security boundary
for now. Binding it to a LAN interface is a deliberate decision this
project has not made; the server logs a warning at startup if it detects a
non-loopback bind, but does not refuse to start.

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
