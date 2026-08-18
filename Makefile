.PHONY: build test test-portable proto dev dev-config plugins plugins-portable signal test-signal dev-check e2e build-portable docs-check external-demo gdrive-external-rehearsal install install-check uninstall

# E2E_PROJECT selects which Playwright project `make e2e` installs/runs —
# "chromium" (the default, and the only engine CI gates on, D-14) or
# "firefox"/"webkit" for a manual cross-engine pass
# (`make e2e E2E_PROJECT=firefox`). E2E_PW_INSTALL_FLAGS is left empty by
# default so a local run never invokes the sudo-requiring system-dependency
# installer; CI overrides it to "--with-deps". E2E_ARGS is a passthrough
# for extra `playwright test` flags (e.g. --grep).
E2E_PROJECT ?= chromium
E2E_PW_INSTALL_FLAGS ?=
E2E_ARGS ?=

# MANIFEST_PLUGIN_BINARIES / MANIFEST_PLUGIN_BINARIES_PORTABLE /
# MANIFEST_E2E_BINARIES name, in exactly one place each, the plugin binary
# PATHS the matching recipe builds — the explicit list
# cmd/topos-manifest's own generator hashes into the kernel's link-time
# trust manifest (D-12). Never a bin/plugins/* glob: a stale binary left
# in bin/plugins/ from an earlier `make plugins` run must never silently
# enter a `make build-portable` manifest (RESEARCH Pitfall 6) — the
# recipe's own explicit binary list is the only authority over what gets
# hashed, mirroring the discipline plugins-portable's own comment already
# states for its six binary names.
MANIFEST_PLUGIN_BINARIES := bin/plugins/topos-plugin-paperless bin/plugins/topos-plugin-silverbullet bin/plugins/topos-plugin-proton bin/plugins/topos-plugin-mock bin/plugins/topos-plugin-whatsapp bin/plugins/topos-plugin-filesystem bin/plugins/topos-plugin-signal
MANIFEST_PLUGIN_BINARIES_PORTABLE := bin/plugins/topos-plugin-paperless bin/plugins/topos-plugin-silverbullet bin/plugins/topos-plugin-proton bin/plugins/topos-plugin-mock bin/plugins/topos-plugin-whatsapp bin/plugins/topos-plugin-filesystem
MANIFEST_E2E_BINARIES := bin/plugins/topos-plugin-mock bin/plugins/topos-plugin-mockstrict bin/plugins/topos-plugin-filesystem

# MANIFEST_GEN_PLUGINS / _PORTABLE / _E2E are the ONE-PLACE-ONLY generator
# invocation for each binary list above — build, build-portable, dev, and
# e2e all reference these instead of re-typing "go run
# ./cmd/topos-manifest ..." at each call site, so the invocation and its
# binary list can never drift apart independently.
MANIFEST_GEN_PLUGINS = go run ./cmd/topos-manifest $(MANIFEST_PLUGIN_BINARIES)
MANIFEST_GEN_PLUGINS_PORTABLE = go run ./cmd/topos-manifest $(MANIFEST_PLUGIN_BINARIES_PORTABLE)
MANIFEST_GEN_E2E = go run ./cmd/topos-manifest $(MANIFEST_E2E_BINARIES)

# MANIFEST_LDFLAGS_VAR is the ONE place the -ldflags -X path is written —
# every recipe below that links bin/topos passes
# "-X $(MANIFEST_LDFLAGS_VAR)=$$MANIFEST", so the Go symbol path can never
# drift out of sync between recipes.
MANIFEST_LDFLAGS_VAR := github.com/davison/topos/kernel/pluginhost.buildManifest

# DEV_HOST/DEV_PORT are the dev-loop kernel's bind address, used by the
# `dev` recipe's pre-flight port guard and readiness gate below. The
# default MUST match web/vite.config.ts's hardcoded proxy target and
# the [server] listen value in ~/.config/topos/config.toml — if one
# moves, the other must move. Override on the command line
# (`make dev DEV_PORT=9999`) if your config uses a non-default port;
# this is the supported escape valve, not a way to skip the guard.
DEV_HOST ?= 127.0.0.1
DEV_PORT ?= 7777

# DEV_CONFIG is the config file `make dev` passes to the kernel via
# --config (see DEV_KERNEL_CMD below). Two things a reader needs: (1)
# each checkout — including each git worktree of this repo — gets its
# OWN generated config.dev.toml, because the default is derived from
# CURDIR, which differs per checkout; a worktree kernel therefore never
# reads, launches from, or writes to the operator's production config,
# index, or plugins directory (the defect 2026-08-14's Phase 13 UAT hit,
# see docs/testing.md). (2) `make dev DEV_CONFIG=$$HOME/.config/topos/
# config.toml` is the supported way to run the dev loop against the
# production config instead — the escape hatch back, in one variable.
DEV_CONFIG ?= $(CURDIR)/config.dev.toml

# DEV_READY_TIMEOUT is how many seconds (one per readiness-gate
# iteration, below) the kernel is allowed to take to start listening
# before `dev` gives up and fails loud. Generous by default because
# `go run` may need to compile first.
DEV_READY_TIMEOUT ?= 60

# DEV_KERNEL_CMD/DEV_UI_CMD are the two child commands the `dev` recipe
# runs. They exist so scripts/dev-guard-smoke.sh can exercise this
# recipe hermetically with fake children instead of a real kernel and a
# real Vite server. Overriding them changes WHICH children run — it
# never disables any guard; the pre-flight port check and readiness
# gate run identically regardless of what these point at. The default
# reads $$TOPOS_DEV_MANIFEST — an environment variable the `dev` recipe
# body exports right before starting this command (never read here at
# Make-parse time, since the manifest doesn't exist yet at this point in
# the file) — at shell expansion time, so the dev-loop kernel links in a
# real manifest of the plugin binaries `plugins` (this target's own
# prerequisite) just built. An overridden DEV_KERNEL_CMD simply ignores
# the variable — dev-guard-smoke.sh's fake children ARE such an override,
# and remain completely unaffected by this. The trailing --config
# $(DEV_CONFIG) is what makes the real dev-loop kernel read this
# checkout's own generated dev config instead of the operator's
# production one (see DEV_CONFIG above) — it stays part of the `?=`
# default, so scripts/dev-guard-smoke.sh's fake-child overrides (which
# replace this whole variable) are unaffected by it.
DEV_KERNEL_CMD ?= go run -ldflags "-X $(MANIFEST_LDFLAGS_VAR)=$$TOPOS_DEV_MANIFEST" ./cmd/topos serve --config $(DEV_CONFIG)
# --host exposes the Vite dev server on all interfaces (including the
# tailscale one); vite.config.ts allowlists *.ts.net Host headers so the
# MagicDNS name works too. Raw-IP access (100.x.y.z:5173) needs no allowlist.
DEV_UI_CMD ?= npm --prefix web run dev -- --open --host

# plugins-portable is the cgo-free sibling of "plugins" — it builds
# exactly the six CGO_ENABLED=0 plugin binaries "plugins" builds
# (paperless, silverbullet, proton, mock, whatsapp, filesystem) and,
# unlike "plugins", does NOT chain to the cgo "signal" target. It exists
# for the same reason "test-portable" exists: a runner (or a developer's
# machine) without the system sqlcipher package installed can still
# produce a complete, real, operator-facing plugin set. The six names
# are written HERE ONLY — build-portable below reaches them by
# delegating to this target, never by naming them itself, so the two
# variants cannot drift apart.
plugins-portable:
	mkdir -p bin/plugins
	go build -o bin/plugins/topos-plugin-paperless ./plugins/paperless
	go build -o bin/plugins/topos-plugin-silverbullet ./plugins/silverbullet
	go build -o bin/plugins/topos-plugin-proton ./plugins/proton
	go build -o bin/plugins/topos-plugin-mock ./plugins/mock
	go build -o bin/plugins/topos-plugin-whatsapp ./plugins/whatsapp
	go build -o bin/plugins/topos-plugin-filesystem ./plugins/filesystem

# build-portable is the cgo-free sibling of "build" — it runs the same
# SPA-build step "build" runs, then delegates to plugins-portable
# (above) instead of plugins, so the resulting artifact set never
# requires a C toolchain or the system sqlcipher package, THEN links the
# kernel — plugins now build BEFORE the kernel (reversed from this
# target's pre-manifest order), because bin/topos's own build embeds a
# link-time manifest (D-12) of the plugin binaries plugins-portable just
# built; hashing binaries that don't exist yet is not possible, so the
# kernel build can no longer come first. This is the entry point CI's
# release/nightly workflows use.
build-portable:
	npm --prefix web ci
	npm --prefix web run build
	$(MAKE) plugins-portable
	MANIFEST="$$($(MANIFEST_GEN_PLUGINS_PORTABLE))" && \
		CGO_ENABLED=0 go build -ldflags "-X $(MANIFEST_LDFLAGS_VAR)=$$MANIFEST" -o bin/topos ./cmd/topos

# build produces the SvelteKit SPA (embedded via kernel/webui/embed.go),
# the plugin binaries — topos-plugin-paperless,
# topos-plugin-silverbullet, topos-plugin-proton,
# topos-plugin-mock (the reference plugin PLUG-05 validates
# docs/plugin-contract.md against), topos-plugin-whatsapp, topos-plugin-
# filesystem, and topos-plugin-signal (built by the "signal" target
# below — the first cgo-enabled plugin in this repo) — and finally the
# kernel binary itself, in that order. The kernel embed directive needs
# kernel/webui/build populated before `go build` runs to embed anything
# beyond the committed .gitkeep placeholder. The plugin set itself is
# built by the "plugins" target below so `dev` and `build` share one
# definition and can never drift apart. Plugins now build BEFORE the
# kernel (reversed from this target's pre-manifest order): bin/topos's
# own build embeds a link-time manifest (D-12) of the plugin binaries
# `plugins` just built — hashing binaries that don't exist yet is not
# possible, so the plugin build must complete first.
build:
	npm --prefix web ci
	npm --prefix web run build
	$(MAKE) plugins
	MANIFEST="$$($(MANIFEST_GEN_PLUGINS))" && \
		CGO_ENABLED=0 go build -ldflags "-X $(MANIFEST_LDFLAGS_VAR)=$$MANIFEST" -o bin/topos ./cmd/topos

# plugins builds the full plugin set — the five CGO_ENABLED=0 binaries
# (now including topos-plugin-whatsapp, pure Go — SRC-03) plus the cgo
# signal plugin via its own isolated target — and nothing else (no npm,
# no kernel build), so it is usable from the dev loop without touching
# kernel/webui/build. It exists so `make dev` can guarantee fresh plugin
# binaries before the kernel starts without paying for a full `build`.
# It deliberately includes the cgo signal plugin: a machine without
# system sqlcipher fails here rather than starting a kernel with an
# incomplete plugin set. `build` delegates to this target so the plugin
# set is defined once. topos-plugin-whatsapp is built BEFORE the signal
# target, not after — it is CGO-free and must not sit behind the one
# cgo-gated build in this target.
plugins:
	mkdir -p bin/plugins
	go build -o bin/plugins/topos-plugin-paperless ./plugins/paperless
	go build -o bin/plugins/topos-plugin-silverbullet ./plugins/silverbullet
	go build -o bin/plugins/topos-plugin-proton ./plugins/proton
	go build -o bin/plugins/topos-plugin-mock ./plugins/mock
	go build -o bin/plugins/topos-plugin-whatsapp ./plugins/whatsapp
	go build -o bin/plugins/topos-plugin-filesystem ./plugins/filesystem
	$(MAKE) signal

# signal builds the Signal plugin binary (SRC-02). Unlike every other
# plugin in this repo, this build is CGO_ENABLED=1 with the libsqlcipher
# build tag — the Task 1-authorised dynamic-link driver strategy
# (04-01-PLAN.md; see plugins/signal/go.mod's replace directive for the
# exact pinned fork/commit) that dynamically links the system's own
# SQLCipher library. This REQUIRES the system sqlcipher library/headers
# present at build time — install before running this target:
#   Arch:          sudo pacman -S sqlcipher
#   Debian/Ubuntu: sudo apt-get install libsqlcipher-dev
# The kernel's own build (above) stays CGO_ENABLED=0 — this is the only
# cgo-enabled build in the repo, isolated to this one target so a build
# of everything else never needs a C toolchain or this system package.
signal:
	CGO_ENABLED=1 go build -tags libsqlcipher -o bin/plugins/topos-plugin-signal ./plugins/signal

# external-demo builds Phase 11's out-of-repo proof plugin
# (testdata/external-plugin, ROADMAP success criterion 5) — a genuinely
# separate Go module (module path example.com/acme/topos-plugin-external-demo,
# outside github.com/davison/topos) written entirely from the published
# contract, standing in for a third party's own separate build. This
# binary is deliberately NOT part of "plugins", "plugins-portable" or
# "build" — it must never land in bin/plugins/, the directory a real
# installation's [plugins] dir can point at — so it gets its own output
# directory, bin/plugins-external/, that no real "[plugins] dir" or
# "external_dir" config value in this repository's own examples ever
# names. CGO_ENABLED=0 like every other plugin in this repo except signal.
external-demo:
	mkdir -p bin/plugins-external
	CGO_ENABLED=0 go build -o bin/plugins-external/topos-plugin-external-demo ./testdata/external-plugin

# test-portable runs every workspace module test EXCEPT the cgo Signal
# plugin (sdk, paperless, silverbullet, proton, mock, mockstrict, whatsapp,
# plus the root kernel module) — the credential-free, cgo-free half of
# `test`, and what CI runs (D-13): a runner without system sqlcipher can
# run everything else. This is the ONLY place that module list is written;
# `test` below delegates to this target rather than duplicating it, so the
# two definitions cannot drift apart (the same discipline the "plugins"
# target's own comment documents for its cgo split). `make test` on a
# desktop is unchanged — it still runs test-portable followed by
# test-signal.
test-portable:
	CGO_ENABLED=0 go build ./... && go test ./...
	cd sdk && CGO_ENABLED=0 go build ./... && go test ./...
	cd plugins/paperless && CGO_ENABLED=0 go build ./... && go test ./...
	cd plugins/silverbullet && CGO_ENABLED=0 go build ./... && go test ./...
	cd plugins/proton && CGO_ENABLED=0 go build ./... && go test ./...
	cd plugins/mock && CGO_ENABLED=0 go build ./... && go test ./...
	cd plugins/mockstrict && CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go test ./...
	cd plugins/whatsapp && CGO_ENABLED=0 go build ./... && go test ./...
	cd plugins/filesystem && CGO_ENABLED=0 go build ./... && go test ./...

# test runs the full test suite across all seven workspace modules (sdk,
# paperless, silverbullet, proton, mock, mockstrict, signal) plus the root
# kernel module: test-portable (above) followed by test-signal (below).
# Go workspaces scope "./..." to the module containing the working
# directory, so each module is tested explicitly rather than relying on a
# single "./..." from the repo root covering all of them.
test: test-portable
	$(MAKE) test-signal

# test-signal runs the Signal plugin module's own tests under the same
# cgo/libsqlcipher build tag as the signal target above — see that
# target's comment for the sqlcipher system-package prerequisite.
test-signal:
	cd plugins/signal && CGO_ENABLED=1 go build -tags libsqlcipher ./... && CGO_ENABLED=1 go test -tags libsqlcipher ./...

# proto regenerates the sdk/gen Go stubs from proto/topos/v1/plugin.proto.
# Prefers buf; falls back to protoc + protoc-gen-go + protoc-gen-go-grpc
# when buf is not installed (see 01-RESEARCH.md Environment Availability).
proto:
	@if command -v buf >/dev/null 2>&1; then \
		cd proto && buf generate; \
	else \
		protoc --go_out=sdk/gen --go_opt=paths=source_relative \
			--go-grpc_out=sdk/gen --go-grpc_opt=paths=source_relative \
			--proto_path=proto \
			proto/topos/v1/plugin.proto; \
	fi

# docs-check guards every relative markdown link/image target across the
# repository's maintained doc set (scripts/check-doc-links.sh) — needs no
# network access and no credentials, since it deliberately does not
# check external URLs. Fails loud, naming every broken target's file and
# line, on a link that no longer resolves.
docs-check:
	./scripts/check-doc-links.sh

# dev-check runs the hermetic behavioural guard for the `dev` recipe
# above (scripts/dev-guard-smoke.sh): squatter on the dev port, kernel
# dying during startup for another reason, and the unregressed happy
# path. Unlike `smoke`, this needs no network access, no live
# credentials, and no config file — it is safe to run at any time,
# including while a real kernel is up on 127.0.0.1:7777, since it only
# ever binds ephemeral ports it selects itself.
dev-check:
	./scripts/dev-guard-smoke.sh

# e2e builds a fresh SPA, builds ONLY the mock, mockstrict, and filesystem
# plugins (deliberately NOT the "plugins" target's full real-plugin set —
# this target does not depend on "plugins" at all, because that target
# chains to the cgo "signal" target, which needs the system sqlcipher
# library, and this harness is cgo-free by design, D-07), links those
# three binaries into the kernel's own manifest, embeds the freshly built
# SPA into that freshly built kernel binary, ensures the requested
# Playwright browser is installed, then runs the suite against the built
# artifact. The SPA build MUST precede the Go build: bin/topos go:embeds
# kernel/webui/build at compile time (see the "build" target's own
# comment above), so building the kernel first would embed whatever
# stale SPA happens to be on disk and the whole suite would then test
# yesterday's UI while reporting green. The three plugin binaries and the
# manifest step now ALSO precede the kernel build (reversed from this
# target's pre-manifest order) for the identical D-12 reason "build" and
# "build-portable" reorder above: bin/topos's own build embeds a
# link-time manifest of the plugin binaries this target just built, which
# is not possible before they exist. topos-plugin-mockstrict is built
# HERE and nowhere else: it exists only for this browser harness
# (07.1-02-PLAN.md D-06). Shipping it into the "plugins" target would
# still be wrong, but keeping it out of that target is NOT what protects
# the operator's own "+ New …" picker — this target writes both fixture
# binaries into bin/plugins/, the same directory `make build`/`make dev`
# populate and a real `[plugins] dir` can point at, so any developer who
# has run `make e2e` has them there too. The actual guarantee is
# kernel/pluginhost.ExcludedPluginBinaries (quick task 260811-r5d), which
# excludes both fixture binaries from the picker's catalog listing
# unconditionally, regardless of what happens to be sitting in
# bin/plugins/.
#
# This target also depends on external-demo (above): Phase 11's
# out-of-repo proof plugin, built into its own bin/plugins-external/
# directory (never bin/plugins/), for the browser specs plans 11-05/11-06
# add that link a real, out-of-repo-shaped plugin binary into a fixture's
# external plugin directory — see testdata/external-plugin/README.md.
e2e:
	npm --prefix web ci
	npm --prefix web run build
	mkdir -p bin/plugins
	go build -o bin/plugins/topos-plugin-mock ./plugins/mock
	go build -o bin/plugins/topos-plugin-mockstrict ./plugins/mockstrict
	go build -o bin/plugins/topos-plugin-filesystem ./plugins/filesystem
	$(MAKE) external-demo
	MANIFEST="$$($(MANIFEST_GEN_E2E))" && \
		CGO_ENABLED=0 go build -ldflags "-X $(MANIFEST_LDFLAGS_VAR)=$$MANIFEST" -o bin/topos ./cmd/topos
	cd web && npx playwright install $(E2E_PW_INSTALL_FLAGS) $(E2E_PROJECT)
	cd web && npx playwright test --project=$(E2E_PROJECT) $(E2E_ARGS)

# TOPOS_GDRIVE_BIN is the absolute path to the real, built
# topos-plugin-gdrive binary this target's one spec
# (web/e2e/specs/14-gdrive-external-rehearsal.spec.ts) drives on the
# untrusted external path. This binary is never built by this repository,
# at any commit (D-08's clean-room boundary) — it comes from the separate,
# out-of-repo `topos-plugin-gdrive` checkout (14-03-SUMMARY.md). Defaults
# to that checkout's own conventional location on this operator's machine;
# override on the command line for a different checkout
# (`make gdrive-external-rehearsal TOPOS_GDRIVE_BIN=/path/to/binary`).
TOPOS_GDRIVE_BIN ?= $(HOME)/projects/davison/topos-plugin-gdrive/topos-plugin-gdrive

# gdrive-external-rehearsal builds the identical fixture artifact set "e2e"
# builds (SPA, mock/mockstrict/filesystem, external-demo, the kernel binary
# under the e2e manifest), then runs ONLY
# web/e2e/specs/14-gdrive-external-rehearsal.spec.ts, with TOPOS_GDRIVE_BIN
# exported into the Playwright process's own environment so the spec can
# resolve the real, out-of-repo Drive binary. Deliberately NOT a
# prerequisite of "e2e" and deliberately never adds the Drive binary to
# bin/plugins/, MANIFEST_E2E_BINARIES, or any other manifest recipe above —
# this binary is never built by this repository and never enters a trusted
# directory (14-04-PLAN.md Task 1's own acceptance criterion). Run with
# TOPOS_GDRIVE_BIN unset (or pointed at a nonexistent path) and this spec's
# own top-level test.skip guard skips loudly, naming the variable, rather
# than silently passing — see the spec's own file header.
gdrive-external-rehearsal:
	npm --prefix web ci
	npm --prefix web run build
	mkdir -p bin/plugins
	go build -o bin/plugins/topos-plugin-mock ./plugins/mock
	go build -o bin/plugins/topos-plugin-mockstrict ./plugins/mockstrict
	go build -o bin/plugins/topos-plugin-filesystem ./plugins/filesystem
	$(MAKE) external-demo
	MANIFEST="$$($(MANIFEST_GEN_E2E))" && \
		CGO_ENABLED=0 go build -ldflags "-X $(MANIFEST_LDFLAGS_VAR)=$$MANIFEST" -o bin/topos ./cmd/topos
	cd web && npx playwright install $(E2E_PW_INSTALL_FLAGS) $(E2E_PROJECT)
	cd web && TOPOS_GDRIVE_BIN=$(TOPOS_GDRIVE_BIN) npx playwright test --project=$(E2E_PROJECT) e2e/specs/14-gdrive-external-rehearsal.spec.ts $(E2E_ARGS)

# PREFIX is the install root `make install` places a published release
# into: the kernel at $(PREFIX)/bin/topos, plugin binaries at
# $(PREFIX)/lib/topos/plugins/ (INST-01). /usr/local matches the FHS
# convention for locally-installed software; override per-invocation
# (`make install VERSION=1.1.0 PREFIX=$$HOME/.local`) for a no-sudo
# user-local install. scripts/install.sh reads this via its environment.
PREFIX ?= /usr/local

# install downloads a published release's artifacts, verifies every
# file's SHA-256 against that release's own checksums.txt, and places
# them under $(PREFIX) — see scripts/install.sh for the full sequence
# (preflight -> stage -> verify -> place; $(PREFIX) is untouched until
# everything has verified). Two first-class forms:
#   make install                 — resolves the LATEST published stable
#                                  release (INST-02) by following the
#                                  releases/latest redirect; the script
#                                  validates the landing URL's host,
#                                  repository path, and three-part
#                                  v<maj>.<min>.<patch> tag shape, so a
#                                  prerelease or the moving nightly tag
#                                  can never be auto-selected
#   make install VERSION=1.1.0   — installs exactly that tag (with or
#                                  without the leading v)
# Both follow the dev/e2e precedent of variables over goal-name hacks.
# Needs curl + sha256sum only — no Go toolchain, no credential, no
# token, no GitHub CLI (public releases download anonymously). An
# unwritable $(PREFIX) fails loud naming `sudo make install`; nothing
# here ever escalates itself.
install:
	PREFIX="$(PREFIX)" ./scripts/install.sh $(VERSION)

# uninstall removes the PREFIX ARTIFACTS ONLY (INST-05): the kernel at
# $(PREFIX)/bin/topos and the topos-plugin-* binaries directly inside
# $(PREFIX)/lib/topos/plugins, then removes those directories with a
# non-recursive rmdir only when they are left empty. The operator's
# config file, kernel index, and plugin stores are NEVER touched —
# scripts/uninstall.sh names no path outside the prefix and offers no
# data-removal flag at all; that absence is the guarantee. Idempotent:
# a second run is a clean no-op that exits 0. Safe while the installed
# kernel is running (removal is by unlink; the live process keeps its
# already-open files).
uninstall:
	PREFIX="$(PREFIX)" ./scripts/uninstall.sh

# install-check runs the hermetic behavioural guard for `make install`
# (scripts/install-smoke.sh): builds a fixture release on local disk,
# installs from it via install.sh's TOPOS_RELEASE_BASE_URL file:// test
# seam — no network, no credentials — and asserts the installed kernel
# at $PREFIX/bin/topos launches the installed plugin from
# $PREFIX/lib/topos/plugins with the stock relative [plugins] dir
# (INST-03). Like dev-check, it only ever binds ephemeral ports it
# selects itself, so it is safe to run while a real kernel is up.
install-check:
	./scripts/install-smoke.sh

# dev-config generates $(DEV_CONFIG) from the tracked config.dev.example.toml
# template, substituting the @CHECKOUT@ placeholder with $(CURDIR) — but
# ONLY when $(DEV_CONFIG) does not already exist. Regeneration must never
# clobber an existing file: an operator's hand edits to their dev config
# (added [sources.*]/[webspaces.*] blocks) survive every `make dev`
# unconditionally, and a DEV_CONFIG override pointed at an already-
# existing file (e.g. the production config, via `make dev
# DEV_CONFIG=$$HOME/.config/topos/config.toml`) is never touched by this
# target either. Prints a loud, multi-line notice on the generation path
# only — a silent no-op the rest of the time.
dev-config:
	@if [ ! -f "$(DEV_CONFIG)" ]; then \
		sed 's|@CHECKOUT@|$(CURDIR)|g' config.dev.example.toml > "$(DEV_CONFIG)"; \
		echo "make dev-config: wrote a fresh per-checkout dev config:" >&2; \
		echo "  $(DEV_CONFIG)" >&2; \
		echo "  it starts with no [sources.*] or [webspaces.*] tables — add your own" >&2; \
		echo "  dev-only source instances directly to that file, or run" >&2; \
		echo "  'make dev DEV_CONFIG=<path>' to point the dev loop at a different" >&2; \
		echo "  config entirely (e.g. your production config)." >&2; \
	fi

# dev runs the kernel and the SvelteKit dev server together. The kernel
# binary is never embedded here — Vite's dev server proxies /api to
# 127.0.0.1:7777 (see web/vite.config.ts), so edits to either side hot
# reload independently. Guarded against the two footguns a real
# debugging session hit on 2026-08-05: (1) a plugin source edit
# silently not taking effect (closed by the `plugins` prerequisite
# above) and (2) a stale kernel already holding the dev port, which
# used to leave Vite running and proxying to that stale kernel with no
# indication anything was wrong. The pre-flight port guard below runs
# BEFORE any child starts; the readiness gate runs BEFORE the UI dev
# server starts — both refusals happen loud and non-zero, and neither
# ever leaves the UI dev server running against a kernel this recipe
# did not itself just start. On the happy path, the EXIT trap's
# `kill 0` deliberately also terminates make itself once `wait`
# returns — kept as-is (this is existing, pre-guard behaviour) because
# a process-group kill is the only teardown that reliably reaps
# `go run`'s kernel child, the kernel's own plugin subprocesses, and
# npm's node child together.
dev: plugins dev-config
	@if ! command -v ss >/dev/null 2>&1; then \
		echo "make dev: 'ss' (iproute2) is required by the dev port guard — install iproute2" >&2; \
		exit 1; \
	fi; \
	HOLDER="$$(ss -H -tlnp "sport = :$(DEV_PORT)")"; \
	if [ -n "$$HOLDER" ]; then \
		echo "make dev: refusing to start — $(DEV_HOST):$(DEV_PORT) is already in use" >&2; \
		echo "$$HOLDER" >&2; \
		echo "make dev: stop that process, or re-run with DEV_PORT=<a different port>" >&2; \
		exit 1; \
	fi; \
	export TOPOS_DEV_MANIFEST="$$($(MANIFEST_GEN_PLUGINS))"; \
	trap 'kill 0' INT TERM; \
	$(DEV_KERNEL_CMD) & \
	KPID=$$!; \
	i=1; \
	while [ "$$i" -le $(DEV_READY_TIMEOUT) ]; do \
		if ! kill -0 $$KPID 2>/dev/null; then \
			echo "make dev: kernel exited during startup (its output is above) — the UI dev server was NOT started ($(DEV_HOST):$(DEV_PORT))" >&2; \
			exit 1; \
		fi; \
		if [ -n "$$(ss -H -tln "sport = :$(DEV_PORT)")" ]; then \
			break; \
		fi; \
		if [ "$$i" -eq $(DEV_READY_TIMEOUT) ]; then \
			echo "make dev: kernel never listened on $(DEV_HOST):$(DEV_PORT) within $(DEV_READY_TIMEOUT)s" >&2; \
			pkill -P $$KPID 2>/dev/null || true; \
			kill $$KPID 2>/dev/null || true; \
			exit 1; \
		fi; \
		sleep 1; \
		i=$$((i + 1)); \
	done; \
	trap 'kill 0' EXIT INT TERM; \
	$(DEV_UI_CMD) & \
	wait
