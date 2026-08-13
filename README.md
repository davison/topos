# <img alt="logo" src="./web/static/app-icon.png" style="width:50px"/> topos

topos pulls together related information from your disparate personal
data silos — email, chats, document storage, notes, wikis — into per-topic
"webspaces," so you can open one view and see everything related to a
topic instead of hunting across apps. It runs entirely on your own
desktop, is read-only against every source it touches, and never sends
your data anywhere else.

## Screenshots

![The webspace stream with the detail pane open](docs/ss/1.png)
_A webspace's chronological cross-source stream, with the detail pane
open on a selected item._
<!-- awaiting docs/ss/1.png — see docs/ss/README.md -->

![The source chip row with per-source health](docs/ss/2.png)
_One merged chip per configured source instance: health, filter, and
manual refresh in a single affordance._
<!-- awaiting docs/ss/2.png — see docs/ss/README.md -->

![Cross-source search results](docs/ss/3.png)
_Full-text search across every source in a webspace, with matched terms
highlighted._
<!-- awaiting docs/ss/3.png — see docs/ss/README.md -->

![The webspace builder UI](docs/ss/4.png)
_Creating a webspace and configuring its source instances entirely from
the browser._
<!-- awaiting docs/ss/4.png — see docs/ss/README.md -->

## Status

topos has completed 11 of its v1 phases. Six sources ship today —
paperless-ngx, SilverBullet, Proton Mail (via Bridge, never marking mail
read), Signal Desktop (read strictly read-only from its local database),
WhatsApp (as a linked device, with its own persistent message store), and
a local/network filesystem folder (docs in a directory, optionally its
subfolders) — interleaved in one chronological stream per webspace. You get source
filtering, per-source health with manual refresh, full-text search
across a webspace, a live detail pane, webspace creation and source
configuration entirely from the browser (no hand-editing TOML required
to get started), a default-deny agent permission model, and a responsive
layout that adapts down to mobile widths. This final phase (10) is
finishing the docs and release engineering that make the project easy to
pick up from the outside.

## Install

There are two ways to get a running `topos`: download a prebuilt release,
or build it from source.

### Prebuilt (recommended)

Download the binaries from the
[latest release](https://github.com/davison/topos/releases/latest):
`topos` (the kernel), `topos-plugin-paperless`,
`topos-plugin-silverbullet`, `topos-plugin-proton`,
`topos-plugin-whatsapp`, and `checksums.txt`. Verify what you downloaded
before running any of it:

```bash
sha256sum -c checksums.txt
```

Place `topos` somewhere on your `PATH` (or run it from wherever you put
it) and the four plugin binaries in a `plugins/` directory next to it —
this is the default `[plugins] dir` `config.example.toml` documents.

**The Signal plugin binary is not published** — it's this repository's
only binary that links against cgo (the system SQLCipher library), so a
build from one CI runner's distro carries no guarantee of running on
yours. If you use Signal, build the plugin yourself with `make signal`
against your own system's SQLCipher; see
[`docs/plugins/signal.md`](docs/plugins/signal.md) for the prerequisites.

**The filesystem plugin binary is not yet published either** — it needs
no cgo and builds cleanly under `make build-portable`, but the release
workflow's published-asset list has not been widened to include it yet.
Build it from source with `make build` or `make build-portable` (below)
in the meantime; see [`docs/plugins/filesystem.md`](docs/plugins/filesystem.md).

### From source

Prerequisites:

- **Go 1.23+** (developed against 1.25).
- **Node 20+** (for building the SvelteKit web UI).
- The proto codegen toolchain (**`buf`**, or `protoc` +
  `protoc-gen-go` + `protoc-gen-go-grpc`) — only needed if you're
  regenerating `sdk/gen` from `proto/topos/v1/plugin.proto`; not
  needed for a normal build.

```bash
make build
```

Produces the SPA (embedded into the kernel binary), `bin/topos`, and the
plugin binaries under `bin/plugins/` — including the cgo-enabled Signal
plugin, since a from-source build already has (or will need) a C
toolchain. Use `make build-portable` instead if you want the same output
minus Signal, with no cgo requirement at all.

## Configure

topos needs two things from you: your source connection details in the
environment, and a config file describing your webspaces.

1. Set the connection details for whichever sources you're using as
   environment variables (put these in a `.env` file at the repo root for
   local development — it's gitignored — and `source` it, or `export`
   them in your shell directly):

   ```bash
   export PAPERLESS_URL="https://paperless.example.lan:8000"
   export PAPERLESS_TOKEN="<your paperless-ngx API token>"
   export SILVERBULLET_URL="https://silverbullet.example.lan:3000"
   export SB_AUTH_TOKEN="<your SilverBullet auth token>"
   export PROTON_BRIDGE_ADDR="<lan-address:port of your Bridge forwarder>"
   export PROTON_BRIDGE_USER="<Bridge-generated username>"
   export PROTON_BRIDGE_PASS="<Bridge-generated password>"
   export PROTON_WEBMAIL_BASE="https://mail.proton.me/u/0"
   ```

   Signal and WhatsApp need no environment variables at all — Signal's
   SQLCipher key is resolved at runtime from your OS keyring, and
   WhatsApp links as its own device via a one-time QR scan. See
   [`docs/plugins/`](docs/plugins/) for every source's own prerequisites
   and gotchas.

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
   distinct `folders` for two configured email instances. If no
   `config.toml` exists on first run, topos bootstraps a default one for
   you and prompts you to create your first webspace.

## Run

```bash
./bin/topos serve
```

Then open `http://127.0.0.1:7777/w/<your-webspace-name>` in a browser.

The server binds `127.0.0.1` (loopback) only by default, and there is
no authentication on its HTTP API in v1 — that's the whole security
boundary for now. Binding it to a LAN interface is a deliberate decision
this project has not made; the server logs a warning at startup if it
detects a non-loopback bind, but does not refuse to start.

## Where to look next

- **[`CONTRIBUTING.md`](CONTRIBUTING.md)** — repository layout, the dev
  loop, and the testing gates, for anyone working on topos itself.
- **[`docs/plugin-contract.md`](docs/plugin-contract.md)** — the published
  contract for writing a new source plugin: the interface you implement,
  how the kernel discovers and launches your binary, how config reaches
  it, and what every `Item` field means.
- **[`docs/api.md`](docs/api.md)** — the complete kernel HTTP JSON
  contract: every route, the stable-id scheme, the ordering guarantee,
  provenance keys, and the full error-code list. This is the same JSON
  the web UI consumes — there is no separate agent API.
- **[`docs/plugins/`](docs/plugins/)** — per-plugin operator docs: install
  requirements, configuration, and gotchas for each of the six source
  plugins.
- **[`SECURITY.md`](SECURITY.md)** — how to report a vulnerability.
- **[`docs/releasing.md`](docs/releasing.md)** — how a release is cut,
  what the nightly build does, and GitHub milestone sync.
- **[`.planning/`](.planning/)** — this project's phase-by-phase
  requirements, design decisions, and roadmap.

## Credits

Built with [Claude](https://www.anthropic.com/claude), by Anthropic, using
the [openGSD](https://github.com/open-gsd/gsd-core) planning and
execution workflow.
