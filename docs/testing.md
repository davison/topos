# Testing topos

This document is the map of every automated gate in this repository: what
each one covers, when to run it, how to write a new browser spec, and the
standing rule that keeps the browser suite from decaying beside the UI it
guards.

None of the eight gates below needs credentials or a `.env` file, and
none needs the network — with one explicit exception: `make
install-check` carries a single live latest-release case that skips
loudly, naming itself, when the network is unreachable. That was not
always true — see "What changed", below.

## The eight gates

### `make test` — every Go module, including the cgo Signal plugin

Builds and tests every Go workspace module (the root kernel module,
`sdk`, `mock`, `mockstrict`) — credential-free, `CGO_ENABLED=0`, no
network access, no config file. Since the functional plugins (including
the cgo Signal build) moved to
[`topos-plugins`](https://github.com/davison/topos-plugins), `test` is
an alias for `test-portable` below.

### `make test-portable` — the same gate, under CI's name

The identical module list — the name survives the plugin split because
CI (D-13) and muscle memory both call it. The module list is defined
once, in `test-portable`, so the two names cannot drift apart.

### `make dev-check` — the hermetic guard for `make dev`

Runs `scripts/dev-guard-smoke.sh`, which exercises `make dev`'s guard
behaviour across six cases: squatter on the dev port, kernel dying
during startup, the happy path, the isolation refusal (an unisolated
dev config is refused by name before any child starts), the
`DEV_ISOLATION_BYPASS` escape hatch (proceeds, with the loud banner),
and a stale dev config's port mismatch failing in seconds rather than
as a readiness-gate timeout. Uses ephemeral ports it selects itself.
Needs no network access, no live credentials, and no config file — it
is safe to run at any time, including while a real kernel is up on the
production port `127.0.0.1:7777`.

### `make e2e` — the hermetic browser suite, the pre-ship gate

Builds a fresh SPA, embeds it in a freshly built kernel binary, builds
only the `mock` and `mockstrict` plugin binaries (never the real-source
plugins — see "Harness architecture", below), installs the requested
Playwright browser if needed, and runs the spec tree in
`web/e2e/specs/`. This is the pre-ship gate: it replaced the last
credential-requiring automated check in this repository (see "What
changed"). It needs no network access, no live source credentials, and
no `.env` file.

### `make docs-check` — every relative doc link resolves

Runs `scripts/check-doc-links.sh` over the maintained documentation set
(repo-root markdown, `docs/`, `web/`, `plugins/`): every relative link
and image target must resolve to a real file. Deliberately never checks
external URLs — a network-dependent gate is a flaky gate. No network,
no credentials, no config file.

### `make install-check` — the hermetic guard for `make install`

Runs `scripts/install-smoke.sh`: builds a fixture release on local
disk, installs from it through the real `scripts/install.sh` (via its
`TOPOS_RELEASE_BASE_URL` `file://` test seam — the seam changes WHICH
release is fetched, never which checks run), and asserts the installed
kernel launches the installed plugin from `$PREFIX/lib/topos/plugins`
with the stock relative `[plugins] dir` (INST-01/INST-03). Its cases
also pin every refusal and repair behaviour: corrupted asset,
traversal-shaped manifest, unwritable prefix, idempotent re-run, live
replacement over a running kernel, the install-time provenance step
(16-05-PLAN.md Task 1, [`docs/plugin-trust.md`](plugin-trust.md)) — a
provenance-free release installs unchanged, a validly-signed release
verifies and installs, and a binary altered after signing aborts naming
it with the target prefix byte-identical before/after — the uninstall
data-safety cycle (a seeded home/XDG tree byte-identical across
install+uninstall; the kernel binaries — `topos` and a seeded
`topos-provenance` — go, while the plugin fleet, a hand-placed foreign
file, and their directory survive byte-identical for topos-plugins'
own `make uninstall`, with the hand-off reported), uninstall under a
live kernel, the toolchain tripwire (the base install completes with
failing compiler shims first on `PATH`), and the latest-release URL
validator.
Hermetic and offline except one live latest-release resolution case,
which skips loudly by name when the network is unreachable. Safe beside
a running kernel — every port is ephemeral and self-selected.

### `make isolation-check` — the dev/installed simultaneity gate

Runs `scripts/simultaneity-smoke.sh`, the committed ISOL-03 proof: an
installed-shaped instance (installed through the real install script,
resolving config through home/XDG) and a checkout-shaped dev instance
(whose config must first pass the real `cmd/topos-devguard`) run side
by side, each answering only its own webspace set; the installed
tree's recursive digest manifest is byte-identical across a dev
instance's sync and index write; its file set is unchanged during
concurrent operation; and the production listen default and `DEV_PORT`
default are asserted different by reading both from source — without
binding either port. No network, no credentials, no config file
outside its own temp tree; a real-port safety baseline is re-asserted
after every case, so it is safe to run while the operator's own
installed instance is serving.

### `make provenance-check` — the hermetic signed-manifest round trip

Runs `scripts/provenance-smoke.sh` (16-01-PLAN.md Task 3): a full
keygen → sign → verify → tamper → refuse round trip for the
ed25519-signed release-manifest trust arm (Phase 16, D-01/D-05/D-08),
including the link-time `provenanceKeysExtra` key-injection seam
(D-12) a dev/e2e build uses to trust its own locally signed fixtures.
Needs no network and no config file; safe to run at any time,
including while a real kernel is up, since it only ever writes inside
its own temp directory.

## The real config and the dev config

The kernel resolves its config file from three sources, in precedence
order: the `--config <path>` flag on `topos serve`/`topos sync`; then the
`TOPOS_CONFIG` environment variable; then the XDG default
(`$XDG_CONFIG_HOME/topos/config.toml`, falling back to
`~/.config/topos/config.toml`).

`make dev` defaults to a generated, per-checkout `config.dev.toml`
(`DEV_CONFIG` in the Makefile, expanded from the tracked
`config.dev.example.toml` template) — so the dev loop never reads or
writes the operator's production config, index, or plugins directory.

The two instances deliberately bind different ports: the dev loop
serves on `127.0.0.1:7778` (`DEV_PORT`, matched by the template's
`[server] listen` and Vite's `/api` proxy target) while the installed
instance keeps `127.0.0.1:7777` (the kernel's compiled-in default), so
both can run at the same time (ISOL-02; `make isolation-check` pins
this contract). And since 15-04 the isolation is mechanical, not
conventional: before starting anything, `make dev` runs
`cmd/topos-devguard` against its config and refuses — naming the
config key, the resolved path, and the root it falls inside — when any
writable path the config declares (the config file itself, the index,
either plugin directory including the omitted-`external_dir` default,
or any source's own store path) resolves inside the topos config or
state roots the installed instance owns (ISOL-01). The same invocation
refuses a stale dev config whose listen port disagrees with
`DEV_PORT`, naming both values, before the readiness gate could mask
the drift as a timeout.

Because the generated path is derived from the checkout root (`CURDIR`),
each git worktree of this repo gets its OWN `config.dev.toml`. This is
the concrete defect that split closes: a Phase 13 UAT run (2026-08-14)
had a worktree kernel resolve the *main checkout's* plugin binaries
through the shared production config, and the build-manifest gate
correctly refused all six trusted plugins as unverified. `make dev` in a
worktree now runs entirely inside that worktree.

`make dev DEV_CONFIG=<path>` runs the dev loop against any other
config — but a config that reaches the installed instance's own
locations now also needs the guard's explicit escape hatch,
`DEV_ISOLATION_BYPASS` (any non-empty value switches the guard to
warn-only, printing a loud banner that lists every path it is
permitting). Running the dev loop against the operator's own
production config therefore takes both, deliberately:
`make dev DEV_CONFIG=$HOME/.config/topos/config.toml
DEV_ISOLATION_BYPASS=1` — the first says which config, the second says
the operator accepts that the dev run will read and write the
installed instance's own config and state. There is no partial or
per-key bypass.

### The per-checkout plugin-store convention

A source's own store path (its `path` key — a WhatsApp linked-device
session store, most consequentially) is the one writable location the
dev template cannot pre-fill, because stores are declared per source.
The convention, documented and demonstrated in
`config.dev.example.toml`: every source store in a dev config sits
under the checkout, in a per-checkout directory
(`@CHECKOUT@/bin/plugin-state/...`), so each checkout — and each git
worktree — gets its own. Copying a source block from the production
config without changing its path is the specific mistake the guard
refuses: it would point the dev run's store (or a fresh device link)
at the very files the installed instance is using.

The hermetic Playwright harness is unaffected by any of this: it already
isolates each spec via its own `XDG_CONFIG_HOME`/`XDG_DATA_HOME`
(see "Harness architecture", below), so nothing in `web/e2e/` needs to
pass `--config` or set `TOPOS_CONFIG`.

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
  (never a directory copy or glob of the real `bin/plugins`) — so no
  real source plugin (the fleet lives in
  [`topos-plugins`](https://github.com/davison/topos-plugins), and
  `make dev` may have adopted its binaries into `bin/plugins`) can
  enter this harness, structurally, not just by convention. The harness genuinely
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
- **Create a temp corpus with `mkdtempCorpus`, never `mkdtempSync`, and
  never tear one down in `test.afterAll`.** Module-scope code runs once
  per Node *process*; `test.afterAll` runs once per Playwright *job*, and
  `fullyParallel` splits a file into `ceil(tests / workers)` jobs. Pairing
  the two means a later job runs against a corpus an earlier job already
  deleted — and because the `kernel` fixture is worker-scoped, its index
  keeps serving the vanished items, so the spec either fails on an
  unrelated-looking `ENOENT` or, worse, passes green while proving
  nothing. `mkdtempCorpus` (`web/e2e/fixtures/corpus.ts`) ties cleanup to
  process exit, which is the lifetime the corpus actually has.
  `web/e2e/specs/spec-hygiene.spec.ts` enforces both halves of this rule
  mechanically, because a run on a developer machine will not reveal the
  mistake.

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
exclusively by `make e2e`'s own build step. `make e2e` DOES write both
binaries into `bin/plugins/` — the same directory `make build`/`make dev`
populate and a real `[plugins] dir` config value can point at — so a
developer who has run the harness has both fixture binaries sitting in
the same directory a real install reads. The kernel therefore refuses to
offer either as an installable plugin type via
`kernel/pluginhost.ExcludedPluginBinaries`, which filters the "+"
picker's catalog listing only (`GET /api/config/plugin-types`); an
already-configured instance of either is unaffected, since
`DescribePluginHandler` reads the unfiltered `DiscoverAllBinaries`. See
`kernel/pluginhost/discover_binaries_test.go`'s `ExcludesMockBinary`/
`ExcludesMockstrictBinary` pair and `web/e2e/specs/
mockstrict-discovery.spec.ts` for the specs that gate this.

## `topos-plugin-external-demo` — the out-of-repo proof plugin

Phase 11 (`testdata/external-plugin/`, ROADMAP success criterion 5) adds a
third fixture binary, built and consumed very differently from the two
mock-shaped plugins above: it is a genuinely **separate Go module**
(module path `example.com/acme/topos-plugin-external-demo`, deliberately
outside `github.com/davison/topos`), written from nothing but the
published contract, standing in for a third party's own out-of-repo
build — the standing, mechanical gate for the claim that a binary this
project did not build as part of its plugin set can be discovered, marked
untrusted, launched under a content-hash pin, and synced end to end.

- **Built only by its own target**, `make external-demo`, into its own
  output directory, `bin/plugins-external/topos-plugin-external-demo` —
  never `bin/plugins/`, and never by `make build` or `make plugins`. `make e2e` depends on this target so the
  browser harness (plans 11-05/11-06) can link the built binary into a
  fixture's external plugin directory alongside the mock-shaped plugins'
  trusted-directory symlinks.
- **Its synced corpus reports back exactly what it was launched
  with** — this is the mechanism by which Phase 11's extras-passthrough
  (`PLUG-09`) and launch-environment-allowlist (D-14) claims are proven by
  observation rather than asserted by a test's own mock: it returns one
  item per configured `[sources.<id>.extras]` key, and one item per
  environment variable actually visible to the launched subprocess
  (variable **names only**, never values, so the synced corpus can never
  carry a secret).
- **Gated by `kernel/supervisor/externalproof_test.go`**'s
  `TestExternalProof_OutOfRepoBinaryEndToEnd` — a real supervisor boot
  against the built binary, asserting discovery, tier, pin enforcement
  (including refusal of a tampered copy by name, alongside a still-healthy
  control source), extras passthrough, and environment scrubbing, all from
  the plugin's own point of view. See `testdata/external-plugin/README.md`
  for the full shape and the reasons it lives under `testdata/`.

### `WEBSPACES_MOCK_READY_AFTER_MS` — the mock's launch-readiness fixture

`topos-plugin-mock` also carries an opt-in launch-readiness window
(`plugins/mock/readiness.go`), closing 08-UAT.md gap G-08-4's fixture gap:
until this existed, the mock's `Match`/`Health` were unconditionally
ready, so no fixture in this repo could express "the relaunched plugin is
not ready yet" — the exact shape a real plugin subprocess can present in
the window between completing the go-plugin handshake and being able to
actually serve a sync.

- Set `WEBSPACES_MOCK_READY_AFTER_MS=<milliseconds>` on the **kernel**
  process AND reference `${WEBSPACES_MOCK_READY_AFTER_MS}` somewhere in
  that instance's own source config (an `extras` value is the
  convention) — `kernel/pluginhost/host.go`'s `allowedEnv` builds every
  plugin subprocess's environment from a strict allowlist (D-14), never
  `os.Environ()` wholesale, and the `${VAR}` reference is what admits
  the variable for exactly that instance
  (`kernel/supervisor/readiness_test.go`'s own extras block is the
  worked example).
- Off by default (absent, empty, or `"0"`): the mock behaves
  byte-identically to a build with no readiness window at all. This means
  it is invisible to `make e2e` (which never sets the variable) and to
  any real installation.
- While the window has not elapsed, `Match` returns a `codes.Unavailable`
  error and `Health` reports `Reachable: false`; `Describe` is
  deliberately never gated by it, since `kernel/pluginhost.launch` calls
  `Describe` immediately after the handshake and treats any error there
  as a launch failure.
- Exercised by `kernel/supervisor/readiness_test.go`'s
  `TestBoot_FirstRefreshSurvivesAPluginLaunchReadinessWindow`, which
  drives a real mock subprocess through a 700ms window and proves the
  scheduler's bounded first-refresh retry (`kernel/syncer/scheduler.go`)
  survives it.

### `WEBSPACES_MOCK_CONTRACT_VERSION` — the mock's contract-declaration fixture

`topos-plugin-mock` carries a third opt-in fixture
(`plugins/mock/contractfixture.go`, M1-R6/DIST-03, davison/topos#17):
any non-empty value is declared VERBATIM as the plugin's
`Describe.contract_version`, so the kernel's contract-generation launch
gate can be driven end to end — a real plugin, declaring a real
unsupported generation, refused for real with both generations named.

- Same delivery as the two fixtures above: set it on the kernel process
  and reference `${WEBSPACES_MOCK_CONTRACT_VERSION}` in the instance's
  own source config.
- Off by default (absent or empty): the mock declares
  `sdk.ContractVersion`, byte-identical to a build with no fixture.
- Exercised by `kernel/pluginhost/incompat_test.go` (the Go gate, both
  Discover and Reconcile) and `web/e2e/specs/17-contract-incompatible.spec.ts`
  (the chip surface).

### `WEBSPACES_MOCK_LAUNCH_DELAY_MS` — the mock's launch-delay fixture

`topos-plugin-mock` also carries a second, sibling opt-in fixture
(`plugins/mock/readiness.go`), closing 08-VERIFICATION.md gap G-08-5:
a delay applied BEFORE the go-plugin handshake, distinct from
`WEBSPACES_MOCK_READY_AFTER_MS` above, which delays `Match`/`Health`
readiness AFTER the handshake has already completed. It models a plugin
that is slow to come up at all — the shape the whatsapp plugin's
serve-mode login wait can present, and the shape any plugin can present,
since go-plugin's own client `StartTimeout` default is a full minute.

- Set `WEBSPACES_MOCK_LAUNCH_DELAY_MS=<milliseconds>` on the **kernel**
  process, with the same per-instance `${VAR}` config reference
  `WEBSPACES_MOCK_READY_AFTER_MS` above needs — the allowlist admits it
  the same way.
- Off by default (absent, empty, or `"0"`): `make e2e` and every real
  installation are unaffected.
- A malformed value fails the subprocess's startup loudly, matching every
  other plugin's fail-loud-by-name startup discipline.
- Exercised by `kernel/supervisor/launchlatency_test.go`'s
  `TestResume_SlowRelaunchDoesNotFreezeOtherSources`, which proves phase
  success criterion 4's "every other source is unaffected" clause: while
  a suspended source instance's resume closure is relaunching a plugin
  subprocess held to a 4-second delay by this variable, every other
  configured source's health-probe and manual-refresh routes still answer
  promptly — a slow plugin relaunch no longer freezes every other
  source's routes kernel-wide.

### `WEBSPACES_MOCK_RENDITION` — the mock's media-previewer fixture

`topos-plugin-mock` also carries a third, sibling opt-in fixture
(`plugins/mock/renditionfixture.go`), making `DetailPane.svelte`'s
`bodyVariant === 'media'` branch (09-UI-SPEC.md Fix 9) reachable by this
hermetic browser harness at all: the mock never has a byte rendition to
offer for any item (`noRenditionReason`), so nothing in this repo's closed
mock-shaped plugin set could otherwise route an item through the media
branch, and Fix 9's aspect-locked box/conditional-float geometry could
never be proven end to end in a real browser.

- Set `WEBSPACES_MOCK_RENDITION=<any non-empty, non-"0" value>` on the
  **kernel** process — inherited by the subprocess the same way
  `WEBSPACES_MOCK_READY_AFTER_MS`/`WEBSPACES_MOCK_LAUNCH_DELAY_MS` are,
  through `pluginhost.launch`'s `os.Environ()` construction.
- Off by default (absent, empty, or `"0"`): the mock's `Fetch` behaviour
  is byte-identical to a build with no fixture at all, so `make e2e`'s
  other specs and every real installation are unaffected.
- While on, only the ONE designated fixture item (`"1"`, which also
  carries non-empty extracted text) gains a rendition: its `FULL` Fetch
  response additionally carries a rendition descriptor (`mime_type
  image/png`), and its `PREVIEW`/`THUMBNAIL` responses return a small
  embedded PNG's bytes instead of `available: false`. Every other mock
  item stays on the no-rendition path in the same run, unlike the two
  fixtures above (which apply kernel-wide once set), a narrower "gates a
  case" fixture (a simple boolean, no numeric parsing/malformed-value
  path) — Task 3 confirms `image/png` is already on
  `kernel/httpapi/item.go`'s `allowedRenditionTypes` allowlist rather
  than widening it.
- Exercised by `web/e2e/specs/09-search-clear-and-previewer.spec.ts`,
  which enables this fixture for its own kernel to prove the previewer's
  geometry (bounded 3:4 aspect ratio) and that extracted text flows
  beside the float, both against a real browser.

## The filesystem and gdrive spec families — moved out at the split

Phases 12–14 drove two real source plugins through this harness: the
filesystem plugin (six specs proving its semantics end to end, plus the
external-tier rehearsal that reused its binary) and the clean-room
Google Drive plugin (one spec behind the `TOPOS_GDRIVE_BIN` seam, with
its own `make gdrive-external-rehearsal` target). At the plugin split
([davison/topos#13](https://github.com/davison/topos/issues/13)) their
subjects moved to
[`topos-plugins`](https://github.com/davison/topos-plugins) — the specs
and the rehearsal target went with them, and browser-level coverage for
the moved fleet is tracked there
([topos-plugins#6](https://github.com/davison/topos-plugins/issues/6)).
The dropped specs' last in-repo versions live at tag `v1.2.0` under
`web/e2e/specs/`. The mock-backed specs those families sat beside
(tooltip precedence, the 13-* curation set) remain, and the trust-tier
specs below never depended on a functional plugin.

## `web/e2e/specs/13-multi-select-bulk-exclude.spec.ts` / `13-excluded-view.spec.ts` / `13-undo-across-webspace-switch.spec.ts` — per-item curation, real end to end

Phase 13's KERN-09/KERN-10 user-facing half (13-03-PLAN.md), driven
entirely from the real UI against a real booted kernel and the
`topos-plugin-mock` reference plugin's fixed four-item corpus (see
`13-exclude-tracer.spec.ts`, the single-item tracer this plan's bulk/view
specs build on).

- **`13-multi-select-bulk-exclude.spec.ts`** — proves the desktop
  multi-select round trip in one continuous walk: ctrl-click and
  shift-click compose a contiguous selection (replacing, never unioning,
  per 13-UI-SPEC.md E1); a plain click on an unrelated row opens the
  detail pane and leaves an existing selection untouched — pinning D-01's
  rule that only Esc and the action bar's own Clear button may empty it;
  Esc empties it; and a bulk Exclude removes both rows with the
  contract-exact `Excluded 2 items` toast, with Undo restoring them.
- **`13-excluded-view.spec.ts`** — proves the excluded-items view toggle
  is fully absent (not disabled) at zero exclusions (D-06), appears
  reading `Excluded (1)` once an item is excluded via the detail pane,
  reuses the stream surface wholesale when flipped (D-05), and
  auto-flips back to the normal stream — with the toggle disappearing
  again — the instant its one item is un-excluded via the bulk action
  bar's Include button (E4's "no sustained empty-excluded-view state").
- **`13-undo-across-webspace-switch.spec.ts`** (13-07-PLAN.md gap
  closure; extended by 13-08-PLAN.md for G-13-1) — pins 13-REVIEW.md
  WR-01 / 13-VERIFICATION.md's recorded gap: the undo toast's reversal
  must target the webspace the toast was created in, not whichever
  webspace is current when Undo is clicked, across all three write paths
  (single-item exclude, bulk exclude, detail-pane include). A real
  WebspaceSwitcher navigation interrupts the toast's 5000ms window before
  Undo fires. Asserts on two layers, for two different reasons: webspace
  A's reversal is read directly from the kernel, because the SPA
  genuinely issues no refetch for a webspace that is no longer current —
  there is no rendered A-side signal to assert on until the user
  navigates back. Webspace B's stream, by contrast, is asserted on the
  RENDERED page (rows or empty-state copy, plus zero stream skeletons) as
  a direct consequence of `load()`'s entry guard in
  `web/src/routes/w/[webspace]/+page.svelte`: a call whose generation is
  already stale at entry is a true no-op — no state write, no request —
  so B's stream can never be driven into a stranded loading state by A's
  reversal. A fourth test drives 13-UAT.md's exact reported reproduction
  (exclude in a populated webspace, switch to a genuinely EMPTY second
  webspace, Undo) and pins G-13-1: pre-fix, `load()` wrote `loadState =
  'loading'` before checking staleness, and its post-await guards could
  never undo that write for a call that was already stale at entry — so
  the navigated-to webspace was stranded with four permanent skeleton
  rows until a reload or "Refresh all". The three same-webspace-pair
  tests were extended with the identical rendered-stream assertions on B,
  since the data-layer-only version of this spec passed while the
  view-layer defect shipped — the two-layer strategy is what closes that
  gap for good. Each test owns its own webspace pair so the absolute
  `excluded_count` assertions stay hermetic under a shared worker-scoped
  kernel.

## `web/e2e/specs/13-manifest-unverified.spec.ts` / `13-shadowed-advisory.spec.ts` / `16-file-drop-external-tier.spec.ts` / `16-signed-provenance-tier.spec.ts` — the trust-tier states, real binaries

Phase 13's own trust-tier hardening (13-06-PLAN.md, D-12/D-13/D-14) and
Phase 16's provenance-based rewrite (16-03-PLAN.md, D-11) together
prove every bypass path the folded todo
(`2026-08-13-plugin-trust-tier-is-directory-location-not-provenance.md`)
documented is closed by real kernel code and a real browser, not only
by `kernel/pluginhost`'s own unit tests (`manifestgate_test.go`,
`escalation_test.go` — see below). See
[`docs/plugin-trust.md`](plugin-trust.md) for the trust model these
specs exercise. None of the four specs builds a new plugin binary — each
is driven entirely by binaries the existing `make e2e` recipe already
produces.

- **`13-manifest-unverified.spec.ts`** (D-05/D-12/D-13, repointed by
  16-03-PLAN.md Task 2 for D-11) — links a TAMPERED copy of the real
  `topos-plugin-mock` binary (one byte appended) into the hermetic
  kernel's TRUSTED directory under the name `topos-plugin-mock` — a
  name `MANIFEST_E2E_BINARIES` (Makefile) DOES cover, so the kernel's
  link-time build manifest positively vouches for it with a digest
  that no longer matches what's on disk: the tampered-provenance case,
  distinct from a genuinely evidence-free file drop (below), since
  D-11 made location alone confer nothing. Proves the destructive
  chip, the contract-exact `binary not in the trusted build manifest`
  tooltip, an unreachable source, no re-pin menu action, and — pinning
  D-13's log-AND-UI requirement — that the refusal is also named in
  the kernel's own captured log output, not only rendered in the
  browser.
- **`16-file-drop-external-tier.spec.ts`** (D-11, TRUST-04's file-drop
  path) — links the real out-of-repo `topos-plugin-external-demo`
  binary into the hermetic kernel's TRUSTED directory under its own
  name — a name `MANIFEST_E2E_BINARIES` never covers, and carrying no
  signed release manifest either — with NO pin recorded for it. Proves
  the dropped binary resolves to the SAME untrusted/consent-required
  state (destructive chip, trust badge, a "Trust updated binary…"
  remedy, no synced items) as any other never-pinned external-tier
  source, never a silent trusted launch just because it sits in the
  trusted directory, while a healthy `topos-plugin-mock` control chip
  alongside it is unaffected — and that the refusal names the dropped
  binary in the kernel's own log.
- **`16-signed-provenance-tier.spec.ts`** (16-05-PLAN.md Task 2, D-01/
  D-05/D-07/D-12) — the positive half of success criterion 1, in the
  EXTERNAL directory: links the real `topos-plugin-mockstrict` binary
  under two RENAMED destinations (neither a `MANIFEST_E2E_BINARIES`
  name, so the link-time arm cannot be what grants trust), signs each
  independently through the real `topos-provenance` CLI via the
  link-time `provenanceKeysExtra` key seam `make e2e` injects
  (Makefile), and records no pin for either. Proves a validly signed
  external-directory binary renders healthy and trusted with no badge
  and no re-pin remedy, its items sync, and its sibling — signed then
  had its OWN `.provenance.sig` deleted before boot — resolves the
  untrusted path instead: badge, no launch, no synced items. Both cases
  share one kernel boot, since Playwright forbids varying a
  worker-scoped fixture option across `test.describe` blocks in one
  file.
- **`13-shadowed-advisory.spec.ts`** (D-14) — links
  `bin/plugins/topos-plugin-mock` into BOTH the hermetic trusted and
  external directories under the identical name, with a real pin
  recorded for the external copy. The trusted copy wins the launch (the
  shadow rule is unchanged) and verifies against the manifest, so the
  instance launches and syncs normally; the spec proves its chip renders
  the WARNING tone — explicitly asserting it is NOT the destructive tone
  — with the contract-exact shadowing tooltip.

**`kernel/pluginhost/escalation_test.go`** is the Go-level home of
TRUST-04's own escalation coverage — three named tests (config edit,
file drop, name shadowing) that drive the real `EvaluateTrust`/
`resolveBinaryDetailed` gate directly, plus a skipped-by-default
falsifiability proof demonstrated live (a maintainer-reproducible
edit-and-revert against `discover_binaries.go`, documented in the
file's own header). Read it first when the question is "where is the
proof that trust cannot be escalated."

## `web/e2e/specs/14-chip-health-narrow-viewport.spec.ts` — chip health reachable at mobile width

Phase 14's gap closure (14-06-PLAN.md, G-14-2): converts the UAT item
that failed — "the popover is not shown at all unless the viewport is
wide enough to accommodate it" — from a manual re-check into a gate.
Six cases at 375×812 (plus one desktop-parity case at 1280×900) prove
that a long-named source chip stays in the header row at mobile width,
that hovering it opens the styled health popover fully inside the
viewport with its sentence wrapped rather than clipped, that `Refresh
all` and the add-source `+` survive alongside the forced chip, that
exactly ONE chip is forced (the Phase 6 overflow design still owns
everything past the first), and that 14-02's option-b
accessible-description surface is untouched.

The one thing a future reader most needs to know: the popover machinery
was never the defect — `.planning/debug/popover-hidden-narrow-viewport.md`
proved it wraps, flips and shifts correctly down to 375px whenever a
trigger exists, cross-engine. The diagnosed failure was TRIGGER
disappearance (`visibleChipCount` flooring at zero and relegating every
chip into the "+N" pill), and the fix is a budget-gated minimum-chip
floor (`MIN_INLINE_CHIP_PX`, WebspaceHeader.svelte). A regression here
will therefore present as a missing trigger — case 1's chip locator
failing — not a missing popover. The spec pins its own copy of
`MIN_INLINE_CHIP_PX`; change it and the component's constant together.

## `web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts` — the WhatsApp QR pairing flow

Covers the in-app QR pairing surface (08-04-PLAN.md, D-01/D-02/D-03) end
to end in the browser, hermetically: both entry points (inline in the
Add-Source Step 1 dialog, and the source chip's Re-link… entry), all five
panel states, QR rotation, session cancellation releasing the subprocess,
and the "not linked is not a trial-launch failure" rule (the E5
evidence). Because `topos-plugin-whatsapp` is, like the real
paperless/silverbullet/proton/signal plugins, structurally excluded from
this harness's closed plugin set (see "The two mock-shaped plugins",
above), this spec never exercises a real WhatsApp plugin binary — it
intercepts plugin discovery, the trial-launch describe call, and the
three link-session routes at the route layer, and seeds a real
`topos-plugin-mock` instance for the parts of the flow (an actual saved
webspace/source) that don't need WhatsApp specifically. The spec file's
own header comment records the full reasoning, including the one
deliberate exception (case 8's own final save is written directly to
`config.toml` rather than round-tripped through a real, un-launchable
plugin subprocess — real kernel behaviour against a binary this harness
never builds, not a gap in the shipped SPA code).

## What stays manual, and why

Five items remain manual, accepted risk — non-deterministic timing
windows a browser driver cannot reliably provoke, requiring hardware or a
real browser install this harness cannot have, or requiring a real Google
account this harness must never be given:

1. **Killing the kernel between the `config.toml.bak` write and the
   atomic rename during a config save.** The window is a handful of
   syscalls wide; a browser-driven test has no way to land a signal
   inside it deterministically.
2. **Killing the kernel mid-way through cleanup after removing 2+ source
   instances** (between one instance's item deletion returning and its
   sync-run deletion starting). Same class of problem: the window this
   needs to land inside is a race between two database writes, not
   something observable or controllable from the browser.
3. **A real pairing against a real WhatsApp account.** This needs an
   actual phone running WhatsApp to scan a genuine, live QR code — no
   hermetic harness can fabricate that hardware step. Covered instead by
   Plan 08-01 Task 3's hands-on spike (a real, one-time manual pairing
   run, recorded in `08-01-SUMMARY.md`), not by an automated gate.
4. **The actual desktop PWA install, the standalone window launch, and
   the post-upgrade freshness check (13-04-PLAN.md Task 4).** These need
   a real browser's own install affordance and a real kernel binary
   upgrade+restart — neither is something Playwright's hermetic fixture
   process can fabricate. The browser-drivable half (manifest
   Content-Type, the manifest `<link>`, ServiceWorker registration and
   scope, and zero `/api/` entries in Cache Storage) is covered by
   `13-pwa-manifest-sw.spec.ts`.
5. **The three Google Drive criteria that need a real Google account**
   (Phase 14, SRC-05): authorizing once and syncing again across a kernel
   restart with no re-authorization; documents in the configured folder —
   including Workspace-native Docs, Sheets and Slides via export — reaching
   the stream with previews and correct deep links to the Drive web UI;
   and the second and subsequent syncs pulling only changed items from
   Google rather than a full folder re-listing. No hermetic harness can
   fabricate a real Google Drive account, a real OAuth consent flow, or
   observe Google's own incremental change feed. Covered instead by a
   recorded, filled-in run against a real Drive, per
   `.planning/phases/14-google-drive-source-built-out-of-repo/
   14-LIVE-UAT.md` — mirroring how item 3's real-device WhatsApp pairing
   spike is recorded rather than automated.

Recording this here, in shipped documentation, is deliberate: the
standing rule below says future UI work extends this suite, and a reader
who only sees the suite's specs (never this file) could reasonably assume
it covers everything. It does not — these five remain an accepted,
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

**2026-08-11**: closing 08-UAT.md gap G-08-4, the scheduler's bounded
first-refresh retry (`kernel/syncer/scheduler.go`) and the mock plugin's
opt-in launch-readiness fixture (`WEBSPACES_MOCK_READY_AFTER_MS`, above)
landed together — a plugin subprocess that completes its handshake before
it can actually serve `Match` no longer pins a source on an errored sync
run for the default 15-minute sync interval, and this failure class now
has a hermetic gate over a real plugin subprocess
(`kernel/supervisor/readiness_test.go`).

**2026-08-11**: closing 08-VERIFICATION.md gap G-08-5, the supervisor's
reader path (`Supervisor.Host()`/`Coordinator()`) was split off the
mutation lock (`s.mu`) onto its own reader-only lock (`genMu`), and
`pluginhost.Host` became internally synchronised (a new `Host.mu`
guarding its launched-plugin set, with `Reconcile` performing its
subprocess launches with no lock held) — closing both the structural
freeze a slow plugin relaunch used to impose on every other source's
routes, and a latent, pre-existing data race between a reader and a
concurrent `Reconcile`. This failure class now has a hermetic gate over a
real plugin subprocess (`kernel/supervisor/launchlatency_test.go`,
`TestResume_SlowRelaunchDoesNotFreezeOtherSources`), using the mock
plugin's new `WEBSPACES_MOCK_LAUNCH_DELAY_MS` fixture (above).

**2026-08-13**: Phase 11 adds `topos-plugin-external-demo`
(`testdata/external-plugin/`, ROADMAP success criterion 5) — a genuinely
separate Go module, built by its own `make external-demo` target into its
own `bin/plugins-external/` directory, standing in for a third party's own
out-of-repo plugin build. `make e2e` now depends on this target alongside
the mock/mockstrict builds. See "`topos-plugin-external-demo` — the
out-of-repo proof plugin", above, and
`kernel/supervisor/externalproof_test.go`'s
`TestExternalProof_OutOfRepoBinaryEndToEnd` for the standing gate this
lands.

**2026-08-14**: Phase 12 admits a sixth real-source plugin,
`topos-plugin-filesystem`, directly into this harness's `bin/plugins/` —
the first real (non-mock-shaped) source plugin this suite builds and
drives, both from the trusted directory (`12-filesystem-tracer.spec.ts`,
`12-filesystem-recursion.spec.ts`, `12-filesystem-add-source.spec.ts`)
and, for criterion 5's rehearsal, from the external one
(`12-external-rehearsal.spec.ts`). (Those specs left with their plugin at the 2026-08-31 split — see the
entry below.)

**2026-08-31**: the plugin split
([davison/topos#13](https://github.com/davison/topos/issues/13)) moves
every functional plugin to
[`topos-plugins`](https://github.com/davison/topos-plugins): the
filesystem/gdrive spec families and `make gdrive-external-rehearsal`
leave with their subjects, `make test` becomes `test-portable`'s alias,
and the harness builds only the mock-shaped fixtures — which it never
launched a real source through anyway.

**2026-08-19**: Phase 15 (the installed instance) adds the last three
gates: `make docs-check` (previously an undocumented target) joins the
map, and `make install-check` / `make isolation-check` land alongside
the install surface itself. The dev loop moved off the production port
to `127.0.0.1:7778` in the same phase — the installed instance owns
`7777`, and dev isolation became a mechanical pre-flight
(`cmd/topos-devguard`) rather than a convention: a dev config that
would read or write the installed instance's config or state refuses
to start, and the only bypass is the loud, explicit
`DEV_ISOLATION_BYPASS`.

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
