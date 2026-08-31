.PHONY: build test test-portable proto dev dev-config plugins dev-check e2e build-portable docs-check external-demo install install-check uninstall isolation-check provenance-check

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

# MANIFEST_PLUGIN_BINARIES / MANIFEST_E2E_BINARIES name, in exactly one
# place each, the plugin binary PATHS the matching recipe builds — the
# explicit list
# cmd/topos-manifest's own generator hashes into the kernel's link-time
# trust manifest (D-12). Never a bin/plugins/* glob: a stale binary left
# in bin/plugins/ from an earlier `make plugins` run must never silently
# enter a `make build-portable` manifest (RESEARCH Pitfall 6) — the
# recipe's own explicit binary list is the only authority over what gets
# hashed — the discipline the removed plugins-portable target's own
# comment stated for its six binary names before the split.
MANIFEST_PLUGIN_BINARIES := bin/plugins/topos-plugin-mock
MANIFEST_E2E_BINARIES := bin/plugins/topos-plugin-mock bin/plugins/topos-plugin-mockstrict

# MANIFEST_GEN_PLUGINS / MANIFEST_GEN_E2E are the ONE-PLACE-ONLY generator
# invocation for each binary list above — build, build-portable, dev, and
# e2e all reference these instead of re-typing "go run
# ./cmd/topos-manifest ..." at each call site, so the invocation and its
# binary list can never drift apart independently.
MANIFEST_GEN_PLUGINS = go run ./cmd/topos-manifest $(MANIFEST_PLUGIN_BINARIES)
MANIFEST_GEN_E2E = go run ./cmd/topos-manifest $(MANIFEST_E2E_BINARIES)

# MANIFEST_LDFLAGS_VAR is the ONE place the -ldflags -X path is written —
# every recipe below that links bin/topos passes
# "-X $(MANIFEST_LDFLAGS_VAR)=$$MANIFEST", so the Go symbol path can never
# drift out of sync between recipes.
MANIFEST_LDFLAGS_VAR := github.com/davison/topos/kernel/pluginhost.buildManifest

# PROVENANCE_LDFLAGS_VAR is the ONE place the e2e recipe's
# link-time provenanceKeysExtra -ldflags -X path is written (16-05-PLAN.md
# Task 2, D-12): every recipe below that injects an extra accepted
# provenance-verification key passes
# "-X $(PROVENANCE_LDFLAGS_VAR)=$$PROVENANCE_KEY_SPEC", mirroring
# MANIFEST_LDFLAGS_VAR's own one-place-only discipline exactly, so the Go
# symbol path can never drift out of sync between recipes. This is the
# D-12 link-time seam — the ONLY way a kernel binary learns an extra
# accepted provenance-verification key beyond embeddedProvenanceKeys
# (kernel/pluginhost/provenance.go); production build/build-portable/dev
# recipes never set it. It is exactly what keeps "a dev kernel trusts its
# own local builds" possible for Phase 17 (D-12) without ever adding a
# runtime-readable trust input — the same closed discipline
# MANIFEST_LDFLAGS_VAR's own comment describes for the link-time build
# manifest.
PROVENANCE_LDFLAGS_VAR := github.com/davison/topos/kernel/pluginhost.provenanceKeysExtra

# E2E_PROVENANCE_KEY_ID is the fixed key id `make e2e`/
# generates an ephemeral signing keypair
# under, written HERE ONLY — the e2e fixture harness
# (web/e2e/fixtures/plugin-binaries.ts's signProvenanceFixture) reads the
# matching private key file back out of bin/ under this same id, so this
# variable and that TypeScript literal must never drift apart
# independently.
E2E_PROVENANCE_KEY_ID := e2e-fixture

# DEV_HOST/DEV_PORT are the dev-loop kernel's bind address, used by the
# `dev` recipe's pre-flight guards and readiness gate below. 7778 is
# the DEV port; the INSTALLED instance owns 7777 (kernel/config's
# DefaultListen) — they must differ so a dev loop and the installed
# kernel can run side by side without contending for one port
# (ISOL-02). The default MUST match web/vite.config.ts's hardcoded
# proxy target and the generated config.dev.toml's [server] listen
# value — if one moves, the others must move, and the topos-devguard
# pre-flight below fails loud (naming both values) when the config and
# this variable disagree, instead of letting the readiness gate mask
# the drift as a timeout. Override on the command line
# (`make dev DEV_PORT=9999`) only together with a matching listen value
# in your dev config; the consistency check is deliberate, not a hoop.
DEV_HOST ?= 127.0.0.1
DEV_PORT ?= 7778

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

# build-portable is the CI-facing sibling of "build" — since the
# functional plugins moved to davison/topos-plugins, both targets build
# the same mock-only set and this one remains as the stable entry point
# CI's release/nightly workflows call. It runs the SPA build, delegates
# to plugins (below), THEN links the
# kernel — plugins now build BEFORE the kernel (reversed from this
# target's pre-manifest order), because bin/topos's own build embeds a
# link-time manifest (D-12) of the plugin binary "plugins" just
# built; hashing binaries that don't exist yet is not possible, so the
# kernel build can no longer come first. This is the entry point CI's
# release/nightly workflows use.
build-portable:
	npm --prefix web ci
	npm --prefix web run build
	$(MAKE) plugins
	CGO_ENABLED=0 go build -o bin/topos-provenance ./cmd/topos-provenance
	MANIFEST="$$($(MANIFEST_GEN_PLUGINS))" && \
		CGO_ENABLED=0 go build -ldflags "-X $(MANIFEST_LDFLAGS_VAR)=$$MANIFEST" -o bin/topos ./cmd/topos

# build produces the SvelteKit SPA (embedded via kernel/webui/embed.go),
# the mock plugin binary (the reference plugin PLUG-05 validates
# docs/plugin-contract.md against — the functional plugins live in
# davison/topos-plugins now), and finally the kernel binary itself, in
# that order. The kernel embed directive needs kernel/webui/build
# populated before `go build` runs; plugins build BEFORE the kernel
# because bin/topos embeds a link-time manifest (D-12) of the binaries
# `plugins` just built.
build:
	npm --prefix web ci
	npm --prefix web run build
	$(MAKE) plugins
	MANIFEST="$$($(MANIFEST_GEN_PLUGINS))" && \
		CGO_ENABLED=0 go build -ldflags "-X $(MANIFEST_LDFLAGS_VAR)=$$MANIFEST" -o bin/topos ./cmd/topos

# plugins builds the kernel repo's own plugin set — just the mock
# reference plugin now (the six functional plugins live in
# davison/topos-plugins; mockstrict is the e2e target's own fixture and
# is built there). No npm, no kernel build, so `make dev` can guarantee
# a fresh binary without paying for a full `build`.
plugins:
	mkdir -p bin/plugins
	go build -o bin/plugins/topos-plugin-mock ./plugins/mock

# external-demo builds Phase 11's out-of-repo proof plugin
# (testdata/external-plugin, ROADMAP success criterion 5) — a genuinely
# separate Go module (module path example.com/acme/topos-plugin-external-demo,
# outside github.com/davison/topos) written entirely from the published
# contract, standing in for a third party's own separate build. This
# binary is deliberately NOT part of "plugins" or "build" — it must
# never land in bin/plugins/, the directory a real
# installation's [plugins] dir can point at — so it gets its own output
# directory, bin/plugins-external/, that no real "[plugins] dir" or
# "external_dir" config value in this repository's own examples ever
# names. CGO_ENABLED=0 like every plugin left in this repo.
external-demo:
	mkdir -p bin/plugins-external
	CGO_ENABLED=0 go build -o bin/plugins-external/topos-plugin-external-demo ./testdata/external-plugin

# test-portable builds and tests the four buildable workspace modules
# (the root kernel module, sdk, mock, mockstrict) — each module
# explicitly, since Go workspaces scope "./..." to the module containing
# the working directory. The fifth workspace member,
# testdata/external-plugin, is built only by external-demo (below) and
# exercised by the e2e harness. The name survives the plugin split (CI and muscle memory
# call it); with the cgo signal plugin gone to davison/topos-plugins,
# `test` below is the same thing.
test-portable:
	CGO_ENABLED=0 go build ./... && go test ./...
	cd sdk && CGO_ENABLED=0 go build ./... && go test ./...
	cd plugins/mock && CGO_ENABLED=0 go build ./... && go test ./...
	cd plugins/mockstrict && CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go test ./...

# test is test-portable's alias since the cgo signal plugin moved to
# davison/topos-plugins — kept so `make test` keeps working everywhere
# it is written down.
test: test-portable

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

# provenance-check runs the hermetic round-trip gate for the signed
# release-manifest trust arm (16-01-PLAN.md Task 3,
# scripts/provenance-smoke.sh): keygen -> sign -> verify -> tamper ->
# refuse, including the link-time provenanceKeysExtra key-injection
# seam (D-12) a dev/e2e build uses to trust its own locally signed
# fixtures. Needs no network and no config; safe to run while a real
# kernel is up, since it only ever writes inside its own temp
# directory.
provenance-check:
	./scripts/provenance-smoke.sh

# e2e builds a fresh SPA, builds the mock and mockstrict fixture
# plugins (the functional plugins live in davison/topos-plugins and the
# hermetic harness never launches one, D-07), links those
# two binaries into the kernel's own manifest, embeds the freshly built
# SPA into that freshly built kernel binary, ensures the requested
# Playwright browser is installed, then runs the suite against the built
# artifact. The SPA build MUST precede the Go build: bin/topos go:embeds
# kernel/webui/build at compile time (see the "build" target's own
# comment above), so building the kernel first would embed whatever
# stale SPA happens to be on disk and the whole suite would then test
# yesterday's UI while reporting green. The two plugin binaries and the
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
#
# 16-05-PLAN.md Task 2 (D-12): also builds bin/topos-provenance and
# generates an ephemeral e2e-only signing keypair into bin/ (gitignored —
# bin/ is wholesale-ignored), then injects the new PUBLIC key's spec into
# the kernel build via a SECOND -X, alongside the existing manifest one,
# through PROVENANCE_LDFLAGS_VAR. This is the ONLY way this e2e kernel
# learns to trust binaries signed by that throwaway key — the e2e fixture
# harness (web/e2e/fixtures/plugin-binaries.ts's signProvenanceFixture)
# signs against the matching PRIVATE key file this same step writes,
# proving location-independent, provenance-driven trust (16-signed-
# provenance-tier.spec.ts) without ever adding a runtime-readable trust
# input — production build/build-portable/dev recipes never set this.
e2e:
	npm --prefix web ci
	npm --prefix web run build
	mkdir -p bin/plugins
	go build -o bin/plugins/topos-plugin-mock ./plugins/mock
	go build -o bin/plugins/topos-plugin-mockstrict ./plugins/mockstrict
	$(MAKE) external-demo
	CGO_ENABLED=0 go build -o bin/topos-provenance ./cmd/topos-provenance
	PROVENANCE_KEY_SPEC="$$(./bin/topos-provenance keygen --key-id $(E2E_PROVENANCE_KEY_ID) --out-dir bin)" && \
	MANIFEST="$$($(MANIFEST_GEN_E2E))" && \
		CGO_ENABLED=0 go build -ldflags "-X $(MANIFEST_LDFLAGS_VAR)=$$MANIFEST -X $(PROVENANCE_LDFLAGS_VAR)=$$PROVENANCE_KEY_SPEC" -o bin/topos ./cmd/topos
	cd web && npx playwright install $(E2E_PW_INSTALL_FLAGS) $(E2E_PROJECT)
	cd web && npx playwright test --project=$(E2E_PROJECT) $(E2E_ARGS)

# PREFIX is the install root `make install` places a published kernel
# release into: the kernel at $(PREFIX)/bin/topos and, from v1.3.0, the
# provenance verifier at $(PREFIX)/bin/topos-provenance (INST-01); the
# plugin fleet at $(PREFIX)/lib/topos/plugins/ is placed by
# davison/topos-plugins' own make install into the same prefix (older
# kernel tags that shipped plugins still install them byte-for-byte —
# install.sh keeps that allowlist). /usr/local matches the FHS
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

# uninstall removes the KERNEL'S OWN prefix artifacts only:
# $(PREFIX)/bin/topos and $(PREFIX)/bin/topos-provenance. The plugin
# fleet under $(PREFIX)/lib/topos/plugins is deliberately left alone —
# it is topos-plugins' own make uninstall that removes it, whichever
# installer placed it (M1-R5's independence, davison/topos#15); only
# empty directories are tidied, by non-recursive rmdir. The operator's
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
# at $PREFIX/bin/topos launches a plugin from $PREFIX/lib/topos/plugins
# with the stock relative [plugins] dir (INST-03) — the fixture release
# ships one, the pre-split release shape older kernel tags still have,
# which is also why install.sh keeps accepting plugins/* assets. Like dev-check, it only ever binds ephemeral ports it
# selects itself, so it is safe to run while a real kernel is up.
install-check:
	./scripts/install-smoke.sh

# isolation-check runs the committed ISOL-03 gate
# (scripts/simultaneity-smoke.sh): an installed-shaped instance and a
# checkout-shaped dev instance run side by side, each answering only
# its own webspaces, with the installed tree proven byte-identical
# across a dev run and its file set unchanged during concurrent
# operation — plus the static assertion that the production listen
# default and DEV_PORT differ. Hermetic and offline: no network, no
# credentials, no config file outside its own temp tree, every listener
# an ephemeral port it picks itself — safe to run while the operator's
# own installed instance is serving on the production port (a real-port
# safety baseline is re-asserted after every case).
isolation-check:
	./scripts/simultaneity-smoke.sh

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
		echo "  it listens on the DEV port 7778 — the installed instance keeps 7777" >&2; \
		echo "  it starts with no [sources.*] or [webspaces.*] tables — add your own" >&2; \
		echo "  dev-only source instances directly to that file, or run" >&2; \
		echo "  'make dev DEV_CONFIG=<path>' to point the dev loop at a different" >&2; \
		echo "  config entirely (e.g. your production config)." >&2; \
	fi

# DEV_PLUGINS_DIR is where `make dev` looks for a locally built sibling
# plugin fleet (davison/topos-plugins' bin/, by convention beside this
# checkout). Binaries named topos-plugin-* found there are copied into
# bin/plugins/ and hashed into the dev kernel's link-time manifest at
# build time — the same trusted-tier mechanism a release build uses,
# fed by the developer's explicit dir. This is a deliberate exception
# to the explicit-list discipline the manifest variables document
# (RESEARCH Pitfall 6): pointing the variable at a directory IS the
# explicit act, and only `dev` — never build/build-portable/e2e —
# reads it. Absent or empty, dev runs with the mock plugin only and
# says so.
DEV_PLUGINS_DIR ?= ../topos-plugins/bin

# dev runs the kernel and the SvelteKit dev server together. The kernel
# binary is never embedded here — Vite's dev server proxies /api to
# 127.0.0.1:7778 (see web/vite.config.ts; the installed instance keeps
# 7777), so edits to either side hot reload independently. Guarded against the two footguns a real
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
# DEV_ISOLATION_BYPASS is the isolation guard's explicit escape hatch:
# empty by default; ANY non-empty value switches the topos-devguard
# pre-flight below to warn-only, which prints a loud multi-line banner
# to stderr listing EVERY path it is permitting and then proceeds. It
# is the deliberate companion to the DEV_CONFIG override above:
# running the dev loop against the operator's production config now
# requires BOTH (`make dev DEV_CONFIG=$$HOME/.config/topos/config.toml
# DEV_ISOLATION_BYPASS=1`) — DEV_CONFIG says which config, and this
# variable says the operator accepts that the dev run will read and
# write the installed instance's own config and state. There is no
# partial or per-key form: the guard either refuses or announces
# everything it lets through. Like every other guard in this recipe,
# the check runs regardless of what DEV_KERNEL_CMD/DEV_UI_CMD are
# overridden to — overriding the children changes WHICH children run,
# never whether a guard runs.
DEV_ISOLATION_BYPASS ?=

dev: plugins dev-config
	go run ./cmd/topos-devguard --config "$(DEV_CONFIG)" --expected-port $(DEV_PORT) $(if $(DEV_ISOLATION_BYPASS),--warn-only)
	@if ls $(DEV_PLUGINS_DIR)/topos-plugin-* >/dev/null 2>&1; then \
		cp -f $(DEV_PLUGINS_DIR)/topos-plugin-* bin/plugins/; \
		echo "make dev: adopted the sibling fleet from $(DEV_PLUGINS_DIR) into the dev manifest:" >&2; \
		ls $(DEV_PLUGINS_DIR) | sed 's/^/  /' >&2; \
	else \
		echo "make dev: no sibling plugins at $(DEV_PLUGINS_DIR) — dev runs with the mock plugin only" >&2; \
		echo "  (build the fleet in a topos-plugins checkout, or set DEV_PLUGINS_DIR=<dir>)" >&2; \
	fi
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
	export TOPOS_DEV_MANIFEST="$$(go run ./cmd/topos-manifest $(MANIFEST_PLUGIN_BINARIES) $$(ls $(DEV_PLUGINS_DIR)/topos-plugin-* 2>/dev/null | xargs -rn1 basename | sed 's|^|bin/plugins/|' | tr '\n' ' '))"; \
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
