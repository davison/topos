.PHONY: build test test-portable proto dev plugins plugins-portable signal test-signal dev-check e2e build-portable docs-check

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

# DEV_HOST/DEV_PORT are the dev-loop kernel's bind address, used by the
# `dev` recipe's pre-flight port guard and readiness gate below. The
# default MUST match web/vite.config.ts's hardcoded proxy target and
# the [server] listen value in ~/.config/topos/config.toml — if one
# moves, the other must move. Override on the command line
# (`make dev DEV_PORT=9999`) if your config uses a non-default port;
# this is the supported escape valve, not a way to skip the guard.
DEV_HOST ?= 127.0.0.1
DEV_PORT ?= 7777

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
# gate run identically regardless of what these point at.
DEV_KERNEL_CMD ?= go run ./cmd/topos serve
# --host exposes the Vite dev server on all interfaces (including the
# tailscale one); vite.config.ts allowlists *.ts.net Host headers so the
# MagicDNS name works too. Raw-IP access (100.x.y.z:5173) needs no allowlist.
DEV_UI_CMD ?= npm --prefix web run dev -- --open --host

# plugins-portable is the cgo-free sibling of "plugins" — it builds
# exactly the five CGO_ENABLED=0 plugin binaries "plugins" builds
# (paperless, silverbullet, proton, mock, whatsapp) and, unlike
# "plugins", does NOT chain to the cgo "signal" target. It exists for
# the same reason "test-portable" exists: a runner (or a developer's
# machine) without the system sqlcipher package installed can still
# produce a complete, real, operator-facing plugin set. The five names
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

# build-portable is the cgo-free sibling of "build" — it runs the same
# SPA-build and kernel-build steps "build" runs, then delegates to
# plugins-portable (above) instead of plugins, so the resulting
# artifact set never requires a C toolchain or the system sqlcipher
# package. This is the entry point CI's release/nightly workflows use.
build-portable:
	npm --prefix web ci
	npm --prefix web run build
	CGO_ENABLED=0 go build -o bin/topos ./cmd/topos
	$(MAKE) plugins-portable

# build produces the SvelteKit SPA (embedded via kernel/webui/embed.go),
# the kernel binary, and the plugin binaries — topos-plugin-paperless,
# topos-plugin-silverbullet, topos-plugin-proton,
# topos-plugin-mock (the reference plugin PLUG-05 validates
# docs/plugin-contract.md against), and topos-plugin-signal (built by
# the "signal" target below — the first cgo-enabled plugin in this repo)
# — in that order. The kernel embed directive needs kernel/webui/build
# populated before `go build` runs to embed anything beyond the committed
# .gitkeep placeholder. The plugin set itself is built by the "plugins"
# target below so `dev` and `build` share one definition and can never
# drift apart.
build:
	npm --prefix web ci
	npm --prefix web run build
	CGO_ENABLED=0 go build -o bin/topos ./cmd/topos
	$(MAKE) plugins

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

# e2e builds a fresh SPA, embeds it into a freshly built kernel binary,
# builds ONLY the mock and mockstrict plugins (deliberately NOT the
# "plugins" target's full real-plugin set — this target does not depend
# on "plugins" at all, because that target chains to the cgo "signal"
# target, which needs the system sqlcipher library, and this harness is
# cgo-free by design, D-07), ensures the requested Playwright browser is
# installed, then runs the suite against the built artifact. The SPA
# build MUST precede the Go build: bin/topos go:embeds kernel/webui/build
# at compile time (see the "build" target's own comment above), so
# building the kernel first would embed whatever stale SPA happens to be
# on disk and the whole suite would then test yesterday's UI while
# reporting green. topos-plugin-mockstrict is built HERE and nowhere
# else: it exists only for this browser harness (07.1-02-PLAN.md D-06).
# Shipping it into the "plugins" target would still be wrong, but keeping
# it out of that target is NOT what protects the operator's own "+ New …"
# picker — this target writes both fixture binaries into bin/plugins/,
# the same directory `make build`/`make dev` populate and a real
# `[plugins] dir` can point at, so any developer who has run `make e2e`
# has them there too. The actual guarantee is
# kernel/pluginhost.ExcludedPluginBinaries (quick task 260811-r5d),
# which excludes both fixture binaries from the picker's catalog listing
# unconditionally, regardless of what happens to be sitting in
# bin/plugins/.
e2e:
	npm --prefix web ci
	npm --prefix web run build
	CGO_ENABLED=0 go build -o bin/topos ./cmd/topos
	mkdir -p bin/plugins
	go build -o bin/plugins/topos-plugin-mock ./plugins/mock
	go build -o bin/plugins/topos-plugin-mockstrict ./plugins/mockstrict
	cd web && npx playwright install $(E2E_PW_INSTALL_FLAGS) $(E2E_PROJECT)
	cd web && npx playwright test --project=$(E2E_PROJECT) $(E2E_ARGS)

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
dev: plugins
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
