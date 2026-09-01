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

topos is past its v1 milestones and shipped v1.3.1, the usability
milestone. Seven sources ship today — paperless-ngx, SilverBullet,
Proton Mail (via Bridge, never marking mail read), Signal Desktop (read
strictly read-only from its local database, with a schema ceiling that
advances by a verify-and-accept command run against your own database,
never by hand), WhatsApp (as a linked device, with its own persistent
message store), Google Drive, and a local/network filesystem folder —
interleaved in one chronological stream per webspace. They live in
their own repository,
[`topos-plugins`](https://github.com/davison/topos-plugins), and cross
the same published contract a third-party plugin does: signed releases
the kernel verifies, one-command install and update independent of the
kernel, a mismatch refused by name rather than silently, a
[developer guide](docs/plugin-development.md) from an empty module to
`topos plugin pull` — and a developer whose signing key you choose to
trust, through a consent step that says what a key admits, earns a
trusted-by-you tier across their releases instead of a per-binary pin.
You get source filtering (including per-source filter terms, set from
each source's chip or as `instance:term` in the search box), per-source
health with manual refresh, full-text search that reaches item bodies
across every source in a webspace — each source searching its own
store, with a result set that says where each hit matched and which
sources answered — a live detail pane, webspace creation and source
configuration entirely from the browser (no hand-editing TOML required
to get started), a default-deny agent permission model, and a
responsive layout that adapts down to mobile widths. Development
quality-of-life moved with it: docs-only changes clear CI in seconds,
and release version numbers are derived from the commit log rather
than chosen by feel. The record of each milestone lives under
[`docs/milestones/`](docs/milestones/).

## Install

There are two ways to get a running `topos`: download a prebuilt release,
or build it from source.

### Prebuilt (recommended)

From a checkout, one command installs a verified release:

- `make install` — resolves and installs the latest published stable
  release (never a nightly or prerelease)
- `make install VERSION=<tag>` — installs exactly that release
- `make install PREFIX=$HOME/.local ...` — installs somewhere your user
  can write (the default `PREFIX` is `/usr/local`, which usually needs
  `sudo make install`)
- `make uninstall` — removes the kernel's own artifacts (`topos` and
  `topos-provenance` under `$PREFIX/bin`); the plugin fleet is
  [`topos-plugins`](https://github.com/davison/topos-plugins)' own
  `make uninstall` to remove, and your config, index, and plugin
  stores are never touched

Every downloaded file is SHA-256-verified against the release's own
`checksums.txt` before anything is placed. See
[`docs/install.md`](docs/install.md) for the full treatment. A single
plugin installs straight from a URL with
`topos plugin pull <url>` — into the trust tier its provenance earns,
never a tier a flag chose (`docs/install.md`, "Installing a single
plugin from a URL").

Prefer to place files yourself? Download the assets from the
[latest release](https://github.com/davison/topos/releases/latest) —
`topos` (the kernel), `topos-provenance` (the provenance verifier, on
releases that ship it), and `checksums.txt` — and verify what you
downloaded before running any of it:

```bash
sha256sum -c checksums.txt
```

Place both binaries somewhere on your `PATH` (or run `topos` from
wherever you put it).

**Plugin binaries come from
[`topos-plugins`](https://github.com/davison/topos-plugins)** — the
source plugins live there now, and its signed releases carry the fleet;
kernel releases here ship the kernel alone. Install and update the
fleet with that repository's own `make install` (see
[`docs/install.md`](docs/install.md#plugins-on-an-installed-instance);
for Signal, the fleet's one cgo build, topos-plugins' `make build-signal`
and the external-tier consent flow). Older kernel releases still carry
whatever their tag published.

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
mock reference plugin under `bin/plugins/`; `make build-portable` is the
same set plus `bin/topos-provenance`, the entry point the release
workflows use. Nothing here needs cgo. The functional plugins are built
in a [`topos-plugins`](https://github.com/davison/topos-plugins)
checkout (topos-plugins' `make build` and `make build-signal`, the
latter for the one cgo plugin), and `make dev` adopts that checkout's `bin/` through
`DEV_PLUGINS_DIR` — see [`CONTRIBUTING.md`](CONTRIBUTING.md).

### As an app (PWA)

Once the kernel is running, topos installs like a desktop app.

Open the loopback address (`http://127.0.0.1:7777` by default) in a
Chromium-based browser or Edge, and use the browser's own install
affordance — the install icon in the address bar, or "Install topos" in
the browser's app menu. topos adds no install button of its own; the
browser owns that affordance entirely.

This works with no extra setup because `http://localhost`/`http://127.0.0.1`
on any port is a browser-recognized secure context, which is what
ServiceWorker registration and install eligibility both require — and the
kernel's default listener is already loopback.

**The limitation:** opening topos from another device on your LAN — a
phone, a tablet, another desktop — over a plain-HTTP LAN address such as
`http://192.168.1.20:7777` is *not* a secure context, so the browser will
not register a ServiceWorker and will not offer to install the app. The
page itself still loads and works over LAN; only installability is
unavailable there. This is a browser rule, not a topos bug — it's the
same secure-context requirement the kernel's own non-loopback listener
warning is the network-layer counterpart of (see "Run", below: binding
beyond loopback is a deliberate exposure decision the kernel warns about
at startup).

If you want to install topos from another device anyway, that means
putting real TLS in front of the kernel yourself — none of these are
something topos ships, all of them are things you run:

- **A reverse proxy you already trust**, terminating TLS with a
  certificate your devices already trust, forwarding to the kernel.
- **A mesh VPN that issues real certificates for its own hostnames**
  (this repo's own dev tooling already allowlists such hostnames — see
  `allowedHosts` in `web/vite.config.ts`).
- **A locally-generated certificate authority**, added to one trusted
  device's own trust store, for a single-device setup.

Whichever you pick, the kernel has to be reachable from wherever TLS
terminates, which means widening `[server] listen` beyond loopback — and
the kernel logs a warning when you do, deliberately, because that
widening is a real exposure decision (see "Run", below).

No built-in HTTPS listen mode is planned for this release; that's tracked
separately, with no date attached.

## Configure

topos needs two things from you: a config file describing your sources
and webspaces, and — for sources that need them — connection details in
the environment.

1. Each source's connection details (a URL, a token, a Bridge login…)
   are set as environment variables and referenced from the config as
   `${VAR}` — never written literally. Which variables a source needs,
   its prerequisites, and its gotchas are in that plugin's own README in
   [`topos-plugins`](https://github.com/davison/topos-plugins) (the
   plugin table there links each one); sources that read a local store
   need none at all. Put them in a `.env` file at the repo root for
   local development — it's gitignored — and `source` it, or `export`
   them in your shell directly.

2. Copy the example config and edit it:

   ```bash
   mkdir -p ~/.config/topos
   cp config.example.toml ~/.config/topos/config.toml
   ```

   `config.example.toml` documents every key the kernel reads — purpose,
   default, validation rule — with the mock source live so the copy
   syncs as-is; each plugin's own `[sources.<name>]` block is in its
   README. At minimum, define at least one `[webspaces.<name>]` block
   with a `keywords` list matching values your sources actually carry
   (matching is exact and case-insensitive — see the comments in the
   example file for the exact rule). `keywords` is a webspace-level
   fallback applied across every source instance's own declared match
   fields; a per-instance `[webspaces.<name>.match.<instance>]` block
   (also documented in `config.example.toml`) replaces that fallback for
   one instance when you need typed, per-source matching instead — e.g.
   two instances of one plugin type matched on different values. If no
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
- **[`docs/plugin-development.md`](docs/plugin-development.md)** — the
  guide for writing a plugin out of tree: your own module, a kernel to
  run it under, shipping a release, and installing it with
  `topos plugin pull`.
- **[`docs/plugin-contract.md`](docs/plugin-contract.md)** — the published
  contract for writing a new source plugin: the interface you implement,
  how the kernel discovers and launches your binary, how config reaches
  it, and what every `Item` field means.
- **[`docs/api.md`](docs/api.md)** — the complete kernel HTTP JSON
  contract: every route, the stable-id scheme, the ordering guarantee,
  provenance keys, and the full error-code list. This is the same JSON
  the web UI consumes — there is no separate agent API.
- **[`docs/plugins/`](docs/plugins/)** — where the per-plugin operator
  docs went: each plugin's own README in
  [`topos-plugins`](https://github.com/davison/topos-plugins).
- **[`SECURITY.md`](SECURITY.md)** — how to report a vulnerability.
- **[`docs/releasing.md`](docs/releasing.md)** — how a release is cut,
  what the nightly build does, and how a milestone opens and closes
  under CodeCrew.
- **[`ROADMAP.md`](ROADMAP.md)** and **[`docs/milestones/`](docs/milestones/)**
  — the live roadmap and the record of each milestone.
- **[`.planning/`](.planning/)** — the frozen archive of the GSD era:
  its requirements, phase plans and the debug records that code
  comments still cite.

## Credits

Built with [Claude](https://www.anthropic.com/claude), by Anthropic, using
the [openGSD](https://github.com/open-gsd/gsd-core) planning and
execution workflow.
