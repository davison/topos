# Contributing to topos

This is the contributor-facing half of this repository's documentation:
repository layout, the dev loop, the testing gates, and how a change gets
built and released. If you're looking for what topos is and how to
install and run it, see [`README.md`](README.md) instead.

## Repository layout

```
cmd/topos/          kernel binary entrypoint (serve, sync)
kernel/             config, index (SQLite), correlate (sync-time matching), syncer, httpapi, pluginhost, webui (embed)
internal/audit/     repo-wide dependency-floor audit test
proto/topos/v1/     the published plugin contract (source of truth)
sdk/                the plugin-author-facing Go module (handshake, interfaces, generated stubs)
plugins/            source plugins: paperless, silverbullet, proton, signal (cgo), whatsapp, mock and mockstrict (fixtures)
web/                the SvelteKit SPA
scripts/            guard scripts (dev-guard, signal-readonly-smoke, built-stylesheet assertions, milestone sync)
docs/               published contracts: plugin-contract.md, api.md, testing.md, releasing.md, plugins/
```

This is a **Go workspace** (`go.work`) with nine modules: the root kernel
module, `sdk`, and one module per plugin (paperless, silverbullet, proton,
signal, whatsapp, mock, mockstrict). That's deliberate, not incidental —
the Signal plugin needs `cgo` (it dynamically links the system
SQLCipher), and keeping every plugin in its own module means that
requirement stays scoped to the one plugin that needs it. Building the
kernel (or any other plugin) never requires a C toolchain. Don't collapse
these into one module later without re-checking that isolation still
holds.

## Development loop

```bash
make dev
```

Runs the kernel (`go run ./cmd/topos serve`) and the SvelteKit dev
server together; Vite proxies `/api` requests to `127.0.0.1:7777`, so
edits to either side hot-reload independently. The kernel binary built
this way never embeds a built SPA — only `make build`'s production build
does that.

`make dev` rebuilds every plugin binary (including the cgo-enabled
signal plugin, via `make plugins`) before starting the kernel, so a
plugin source edit always takes effect — this is why `make dev` needs
system sqlcipher, the same prerequisite `make signal`/`make build`
already have. This guarantee holds only for your config's default,
relative `[plugins] dir` — if your `config.toml` overrides it to an
**absolute** path (e.g. one pointing at a different checkout), that
rebuild never touches the binaries the kernel actually loads, and a
plugin-side code change (including a plugin's declared icon) can go
silently stale. See `config.example.toml`'s `[plugins] dir` comment.

It also refuses to start when `127.0.0.1:7777` is already in use,
naming the process already holding it and that process's PID, and it
will not start the Vite dev server against a kernel it did not itself
just start — this is the failure mode the guard exists to prevent: a
working-looking UI silently proxying to a stale kernel running old
code. A kernel that dies during startup for any other reason (compile
error, config error, bind failure) produces the same loud, non-zero
failure instead of a half-started stack.

`DEV_PORT`, `DEV_HOST`, and `DEV_READY_TIMEOUT` can be overridden on
the `make` command line (e.g. `make dev DEV_PORT=9999`) if your
`[server] listen` config uses a non-default address — `DEV_PORT` must
stay in step with `web/vite.config.ts`'s hardcoded proxy target if you
change it.

## Testing

```bash
make test               # go build + go test across all workspace modules, including cgo Signal
make test-portable       # same, minus the cgo Signal plugin — what CI runs
make e2e                 # build + hermetic Playwright suite (Chromium) — the pre-ship gate
make dev-check            # scripts/dev-guard-smoke.sh — behavioural guard for `make dev`
```

`make test`/`make test-portable` need no network access or live
credentials — every committed test runs against fixtures and a temp
SQLite file; the `-portable` variant additionally needs no C toolchain
or system sqlcipher. `make e2e` builds the shipped SPA and kernel, then
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
make build              # SPA + kernel + all plugin binaries, including cgo Signal
make build-portable      # same, minus Signal — the cgo-free entry point CI's release/nightly workflows use
make plugins             # rebuild only the plugin binaries
make plugins-portable    # rebuild only the cgo-free plugin binaries
make signal               # build only the Signal plugin (needs system sqlcipher)
```

See [`docs/releasing.md`](docs/releasing.md) for how a release is
actually cut, what the nightly build does, and why the Signal plugin
binary is never among the published artifacts.

## Where to look next

- **[`docs/plugin-contract.md`](docs/plugin-contract.md)** — the
  published contract for writing a new source plugin: the interface you
  implement, how the kernel discovers and launches your binary, how
  config reaches it, and what every `Item` field means.
- **[`docs/api.md`](docs/api.md)** — the complete kernel HTTP JSON
  contract: every route, the stable-id scheme, the ordering guarantee,
  provenance keys, and the full error-code list.
- **[`docs/testing.md`](docs/testing.md)** — the full testing map.
- **[`docs/plugins/`](docs/plugins/)** — per-plugin operator docs for
  each shipped source.
- **[`docs/releasing.md`](docs/releasing.md)** — cutting a release,
  nightly builds, and GitHub milestone sync.
- **[`.planning/`](.planning/)** — this project's phase-by-phase
  requirements, design decisions, and roadmap.
