# Contributing to topos

This is the contributor-facing half of this repository's documentation:
repository layout, the dev loop, the testing gates, and how a change gets
built and released. If you're looking for what topos is and how to
install and run it, see [`README.md`](README.md) instead.

## Repository layout

```
cmd/                topos (serve, sync, plugin pull), topos-manifest, topos-provenance, topos-devguard
kernel/             config, index (SQLite), correlate (sync-time matching), syncer, httpapi, pluginhost, webui (embed)
internal/audit/     repo-wide egress, module-pin and icon audits
proto/topos/v1/     the published plugin contract (source of truth)
sdk/                the plugin-author-facing Go module (handshake, interfaces, generated stubs, sdk.ContractVersion)
plugins/            the two reference/fixture plugins only: mock and mockstrict
testdata/           the out-of-repo proof plugin (external-plugin) and other never-shipped fixtures
web/                the SvelteKit SPA and the Playwright e2e harness
scripts/            install/uninstall and every hermetic smoke gate (dev-guard, install, provenance, simultaneity, doc links, milestone sync)
docs/               plugin-contract.md, plugin-development.md, plugin-trust.md, api.md, install.md, testing.md, releasing.md, milestones/
```

This is a **Go workspace** (`go.work`) with five members: the root
kernel module, `sdk`, the two fixture plugins (`plugins/mock`,
`plugins/mockstrict`), and `testdata/external-plugin` — the out-of-repo
proof plugin, a workspace member only so its sdk import resolves
locally, never built by `./...` patterns or scanned by the audits (its
README explains why it lives under `testdata/`). Every functional source plugin lives in
[`topos-plugins`](https://github.com/davison/topos-plugins) — one Go
module per plugin under that repository's own workspace, with its own
CI and signed releases — and crosses the same published contract a
third-party plugin does. Nothing in this repository builds with cgo:
the one cgo plugin (Signal, linking the system SQLCipher) is built in
the sibling repository, and a C toolchain is never a prerequisite here.

## Development loop

```bash
make dev
```

Runs the kernel (`go run ./cmd/topos serve`) and the SvelteKit dev
server together; Vite proxies `/api` requests to `127.0.0.1:7778` —
the dev port; an installed instance keeps `7777`, so both can run side
by side — and edits to either side hot-reload independently. The kernel binary built
this way never embeds a built SPA — only `make build`'s production build
does that.

`make dev` rebuilds the mock plugin (`make plugins`) and adopts the
functional fleet from a sibling checkout: every `topos-plugin-*` binary
in `DEV_PLUGINS_DIR` (default `../topos-plugins/bin`, which topos-plugins'
`make build`/`make build-signal` fill) is copied beside the
mock and hashed into the dev kernel's link-time manifest at build time,
so the fleet runs at the trusted tier in the dev instance with no
consent-pin churn. Absent or empty, dev runs with the mock only and says
so. A plugin source edit takes effect by rebuilding it in the sibling
checkout and re-running `make dev`. This guarantee holds only for your
config's default, relative `[plugins] dir` — if your `config.toml`
overrides it to an **absolute** path (e.g. one pointing at a different
checkout), that adoption never touches the binaries the kernel actually
loads, and a plugin-side change (including a plugin's declared icon)
can go silently stale. See `config.example.toml`'s `[plugins] dir`
comment.

Before starting anything, `make dev` runs the isolation pre-flight
(`cmd/topos-devguard`): a dev config whose config file, index, plugin
directories, or any source store path resolves inside the installed
instance's own config or state roots is refused, naming the config
key, the resolved path, and the root it falls inside. The same check
refuses a stale dev config whose `[server] listen` port disagrees with
`DEV_PORT`, naming both values (delete the generated `config.dev.toml`
and the generator writes a fresh one). The only bypass is the explicit
`DEV_ISOLATION_BYPASS=1` variable, which prints a loud banner listing
every path it permits — see
[`docs/testing.md`](docs/testing.md)'s "The real config and the dev
config" for the full account.

It also refuses to start when the dev port `127.0.0.1:7778` is already
in use, naming the process already holding it and that process's PID,
and it will not start the Vite dev server against a kernel it did not
itself just start — this is the failure mode the guard exists to
prevent: a working-looking UI silently proxying to a stale kernel
running old code. A kernel that dies during startup for any other
reason (compile error, config error, bind failure) produces the same
loud, non-zero failure instead of a half-started stack.

`DEV_PORT`, `DEV_HOST`, and `DEV_READY_TIMEOUT` can be overridden on
the `make` command line (e.g. `make dev DEV_PORT=9999`) if your dev
config uses a non-default address — `DEV_PORT` must stay in step with
your dev config's `[server] listen` value (the pre-flight enforces
this) and with `web/vite.config.ts`'s hardcoded proxy target if you
change it.

## Testing

```bash
make test-portable       # go build + go test across the four buildable modules (root, sdk, mock, mockstrict; CGO_ENABLED=0) — what CI runs
make test                # alias of test-portable, kept for muscle memory
make e2e                 # build + hermetic Playwright suite (Chromium) — the pre-ship gate
make dev-check            # scripts/dev-guard-smoke.sh — behavioural guard for `make dev`
make docs-check           # scripts/check-doc-links.sh — every relative doc link resolves
make install-check        # scripts/install-smoke.sh — hermetic guard for `make install`/`uninstall`
make provenance-check     # scripts/provenance-smoke.sh — keygen → sign → verify → tamper → refuse
make isolation-check      # scripts/simultaneity-smoke.sh — dev + installed side by side (ISOL-03)
make dev-config           # generate config.dev.toml from config.dev.example.toml (never clobbers an existing one)
make proto                # regenerate sdk/gen from proto/topos/v1/plugin.proto (buf, or protoc + plugins)
```

`make test-portable` covers the four buildable modules — the root
kernel module, `sdk`, `plugins/mock`, `plugins/mockstrict`; the fifth
workspace member, `testdata/external-plugin`, is built only by
`make external-demo` and exercised by the e2e harness — and needs no
network access, no live credentials, and no C toolchain: every committed
test runs against fixtures and a temp SQLite file. `make e2e` builds the shipped SPA and kernel, then
drives a real Chromium against a hermetic kernel instance seeded from
mock-shaped plugin fixtures — also no network access, no live source
credentials, and no `.env` file. `make dev-check` proves `make dev`'s own
port-guard and readiness-gate behaviour hermetically. See
[`docs/testing.md`](docs/testing.md) for the full map: every gate, how to
run a single spec, the harness architecture, and how to write a new one.

**Any phase or change that touches the UI extends the Playwright e2e
suite as part of its definition of done; any UAT item a browser can
drive becomes a spec (`web/e2e/specs/`) rather than staying a manual
check** — a manual check that has to be remembered is not a gate.

## Building and releasing

```bash
make build              # SPA + the mock plugin + the kernel (embedding both)
make build-portable      # the same set plus bin/topos-provenance — the entry point CI's release/nightly workflows use
make plugins             # rebuild only the mock plugin binary
make external-demo       # the out-of-repo proof plugin, into its own bin/plugins-external/
make install             # place a published kernel release under PREFIX (docs/install.md)
make uninstall           # remove exactly the kernel artifacts it placed
```

Kernel releases ship the kernel and the provenance verifier; the plugin
fleet is released, installed and updated from
[`topos-plugins`](https://github.com/davison/topos-plugins). See
[`docs/releasing.md`](docs/releasing.md) for how a release is actually
cut and what the nightly build does.

## Where to look next

- **[`docs/plugin-development.md`](docs/plugin-development.md)** — writing
  a plugin out of tree, from an empty module to `topos plugin pull`.
- **[`docs/plugin-contract.md`](docs/plugin-contract.md)** — the
  published contract for writing a new source plugin: the interface you
  implement, how the kernel discovers and launches your binary, how
  config reaches it, and what every `Item` field means.
- **[`docs/api.md`](docs/api.md)** — the complete kernel HTTP JSON
  contract: every route, the stable-id scheme, the ordering guarantee,
  provenance keys, and the full error-code list.
- **[`docs/testing.md`](docs/testing.md)** — the full testing map.
- **[`docs/plugins/`](docs/plugins/)** — where the per-plugin operator
  docs went: each plugin's own README in `topos-plugins`.
- **[`docs/releasing.md`](docs/releasing.md)** — cutting a release,
  nightly builds, and GitHub milestone sync.
- **[`.planning/`](.planning/)** — this project's phase-by-phase
  requirements, design decisions, and roadmap.
