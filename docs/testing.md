# Testing topos

This document is the map of every automated gate in this repository: what
each one covers, when to run it, how to write a new browser spec, and the
standing rule that keeps the browser suite from decaying beside the UI it
guards.

None of the four gates below needs credentials, network access, or a
`.env` file. That was not always true — see "What changed", below.

## The four gates

### `make test` — every Go module, including the cgo Signal plugin

Builds and tests all seven Go workspace modules (the root kernel module,
`sdk`, `paperless`, `silverbullet`, `proton`, `mock`, `mockstrict`) plus
the Signal plugin module, which is `CGO_ENABLED=1` against the system
SQLCipher library. This is the full local gate a desktop developer runs —
it requires a C toolchain and the system `sqlcipher` package (see the
Makefile's `signal` target for the install commands), but no network
access, live credentials, or config file.

### `make test-portable` — every Go module except Signal

Everything `make test` runs, minus the Signal module — the credential-free,
**cgo-free** half. This is what CI runs: a GitHub Actions runner has no
reason to install the system SQLCipher library for a plugin the browser
harness never launches. `test-portable` and `test` share one module list
(defined once, in `test-portable`) so the two definitions cannot drift
apart — the same discipline the Makefile's `plugins`/`signal` split
already established for the build side.

### `make dev-check` — the hermetic guard for `make dev`

Runs `scripts/dev-guard-smoke.sh`, which exercises `make dev`'s port-guard
and readiness-gate behaviour (squatter on the dev port, kernel dying
during startup, the happy path) using ephemeral ports it selects itself.
Needs no network access, no live credentials, and no config file — it is
safe to run at any time, including while a real kernel is up on
`127.0.0.1:7777`.

### `make e2e` — the hermetic browser suite, the pre-ship gate

Builds a fresh SPA, embeds it in a freshly built kernel binary, builds
only the `mock` and `mockstrict` plugin binaries (never the real-source
plugins — see "Harness architecture", below), installs the requested
Playwright browser if needed, and runs the spec tree in
`web/e2e/specs/`. This is the pre-ship gate: it replaced the last
credential-requiring automated check in this repository (see "What
changed"). It needs no network access, no live source credentials, and
no `.env` file.

## Running a single spec, a single project, or a manual cross-engine pass

```bash
# a single spec file
cd web && npx playwright test --project=chromium e2e/specs/smoke-core-journey.spec.ts

# a single test by name
cd web && npx playwright test --project=chromium -g "search finds, highlights"

# via the Makefile wrapper (passes extra flags through E2E_ARGS)
make e2e E2E_ARGS="--grep 'search finds'"
```

Chromium is the only engine the automated gate runs (CI's `make e2e` and a
plain local `make e2e` both default `E2E_PROJECT` to `chromium`). Firefox
and WebKit are defined as Playwright projects too, and can be run manually
on demand:

```bash
make e2e E2E_PROJECT=firefox
make e2e E2E_PROJECT=webkit
```

Neither engine is ever part of the automated gate — this is a deliberate,
user-stated scope limit, not an oversight: cross-engine coverage is a
manual affordance for an occasional pass, not a per-PR requirement.

## Harness architecture

Each spec **file** boots its own fresh `bin/topos serve` process, on an
OS-assigned ephemeral port, driven from a generated temp `config.toml`,
against a temp plugins directory containing only the `mock` and (where a
spec needs it) `mockstrict` fixture plugin binaries. Every one of those
isolation properties exists to prevent a specific failure:

- **Fresh kernel per spec file** (not a single shared server for the whole
  run) — so one spec's state can never leak into another's, and a spec
  can freely mutate config (add/remove a source, edit match settings)
  without coordinating with every other file in the run.
- **OS-assigned ephemeral port** (not a fixed port with a retry loop) — so
  a full parallel run never collides with another worker's kernel, or
  with a developer's own `make dev` kernel already running on
  `127.0.0.1:7777`.
- **Generated temp `config.toml`, written before boot** (never driven
  through the UI to build preconditions) — state is seeded the same way a
  fixture seeds a database: directly, before the thing under test starts.
  A spec that needs specific state writes it into the fixture's
  `FixtureConfigSpec`, not into a sequence of UI clicks.
- **A closed-set temp plugins directory, symlinked from `bin/plugins`**
  (never a directory copy or glob of the real `bin/plugins`) — so the
  paperless/silverbullet/proton/signal plugins can never enter this
  harness, structurally, not just by convention. The harness genuinely
  cannot reach a real source, a real credential, or the operator's real
  `~/.config/topos/config.toml` / `~/.local/share/topos/index.db`.
- **An explicit environment allowlist on every kernel spawn** (`PATH`,
  `HOME`, `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_CACHE_HOME` — all
  pointed at the fixture's own temp directory, never a `process.env`
  spread) — so a stray `XDG_CONFIG_HOME` set in a developer's own shell
  can never redirect a spec's kernel at their real config.

The fixture API that implements all of this lives in `web/e2e/fixtures/`:

- `kernel.ts` — the worker-scoped `kernel` fixture (boot/teardown,
  `waitForFirstSync`), keyed on a per-file `configSpec` worker-option so
  each spec file's own config gets its own fresh kernel even when
  Playwright reuses one OS worker process across files.
- `config-builder.ts` — `buildConfig`/`writeConfig` (the single writer of
  `index.path`/`plugins.dir` — every fixture helper routes through this,
  so the absolute-path guard can never be bypassed), plus four ready-made
  spec shapes: `mockInstances(n)`, `webspacesWithKeywords(names,
  keywords)`, `emptyShellWebspace(name)`, `attachedWebspace(...)`.
- `plugin-binaries.ts` — `linkPluginBinaries`, the closed-set symlinker
  described above.
- `toml.ts` — `readConfigToml`, for parsed-TOML config-effect assertions.

## Writing a new spec

- **Seed state through the fixture, not the UI.** Build your `config.toml`
  via `FixtureConfigSpec` (or one of `config-builder.ts`'s ready-made
  shapes) and pass it to `test.use({ configSpec })` at the top of your
  spec file. Do not drive the UI through a sequence of setup clicks to
  reach the state your assertions actually care about — that couples an
  unrelated UI flow's stability to your spec's pass/fail.
- **Wait for the first sync before asserting on stream contents.** A
  kernel reporting "listening" and a kernel having completed its first
  sync are two different events — the scheduler fires each configured
  source's first refresh at boot, and racing that refresh is a real,
  observed flake source, not a theoretical one. Use `waitForFirstSync`
  from `kernel.ts`.
- **Assert file outcomes through parsed TOML, never text matching.** Read
  a written `config.toml` back with `readConfigToml` and assert on the
  parsed structure. A canonical-writer regression (reordered keys,
  changed quoting, a dropped table) must be caught by structural
  comparison — a raw substring match can survive exactly the kind of
  change you want a test to catch.
- **Register route handlers before the action that triggers them.**
  `page.route(...)` must be wired before the click/navigation that
  provokes the request it intercepts, or the real request goes out
  unmocked.
- **Prefer user-facing locators.** `getByRole`, `getByText`,
  `getByLabel` — not CSS selectors reaching into implementation
  structure. Do not add `data-testid`-style test attributes to shipped
  components purely to make a locator easier to write; if a locator is
  hard to write, that is often a signal the accessible name/role itself
  needs work, which is a real UI defect, not a test-authoring problem to
  route around.

## Zero-retries policy

`web/playwright.config.ts` sets `retries: 0`, locally and in CI. A flaky
test is treated as a bug and gets fixed, not silenced with a retry — the
per-file hermetic kernel already removes the usual sources of flake
(shared state, port collisions, live network variance), so a retry here
would only mask a real regression rather than compensate for a
legitimately unreliable environment. Every failure — local or CI — leaves
a trace and a screenshot in the artifact directory
(`web/e2e/.artifacts/`); CI additionally uploads that directory as a
build artifact on a failed run.

## The two mock-shaped plugins

- **`topos-plugin-mock`** (`plugins/mock/`) — the PLUG-05 reference
  plugin: a complete, working, config-free plugin built from nothing but
  the published contract (`docs/plugin-contract.md`, the proto, and the
  `sdk` module). It carries a small fixed corpus exercising every
  `LinkFidelity`/`ContentVariant` value the contract defines.
- **`topos-plugin-mockstrict`** — a second fixture plugin, built only for
  this browser harness, used by specs that need to prove the kernel's
  plugin-discovery and multi-plugin-type behaviour against more than one
  plugin type at once.

Neither ever ships in `make build` or `make plugins` — both are built
exclusively by `make e2e`'s own build step. Adding either to the real
plugin set would ship a fixture plugin into every real install's plugin
directory, where the kernel would discover and offer it in the
operator's own "+ New …" picker.

## What stays manual, and why

Two UAT items from Phase 7 remain manual, accepted risk — they are
non-deterministic timing windows around a `SIGKILL`, not something a
browser driver can reliably provoke:

1. **Killing the kernel between the `config.toml.bak` write and the
   atomic rename during a config save.** The window is a handful of
   syscalls wide; a browser-driven test has no way to land a signal
   inside it deterministically.
2. **Killing the kernel mid-way through cleanup after removing 2+ source
   instances** (between one instance's item deletion returning and its
   sync-run deletion starting). Same class of problem: the window this
   needs to land inside is a race between two database writes, not
   something observable or controllable from the browser.

Recording this here, in shipped documentation, is deliberate: the
standing rule below says future UI work extends this suite, and a reader
who only sees the suite's specs (never this file) could reasonably assume
it covers everything. It does not — these two remain an accepted,
explicitly-scoped gap, not a silent one.

## What changed

Before this phase, the pre-ship gate was `make smoke`
(`scripts/e2e-smoke.sh`): a real end-to-end run against the operator's own
live paperless-ngx instance, requiring `PAPERLESS_URL`/`PAPERLESS_TOKEN`
and network access. It has been retired — every assertion it made now has
a hermetic counterpart in `web/e2e/specs/smoke-served-spa.spec.ts` — and
`make e2e` has taken its place. Live-source verification against a real
paperless-ngx/SilverBullet/Proton instance is now a manual UAT activity,
not an automated gate. There was no CI at all before this phase; now
every push and pull request to `main` runs the full portable gate
automatically.

## Standing rule

**Any phase that touches the UI extends this suite as part of its
definition of done.** Any UAT item that a browser can drive becomes a
spec in `web/e2e/specs/` rather than staying a manual checklist entry.

This is the user's own stated reasoning, and it is the reason this
document exists at all: a manual check that has to be remembered is not
a gate. A gate runs whether or not anyone remembers it — on every push,
on every pull request, without anyone deciding to run it that day. Every
future phase's planner and executor should read this rule as a hard
requirement on scope, not an aspiration: if a phase adds or changes
UI behaviour, its definition of done includes the spec (or spec update)
that proves it, and any UAT item that turns out to be browser-automatable
should be ported into this suite rather than left to accumulate as a
manual re-check on every future release.
