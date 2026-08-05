.PHONY: build test proto smoke dev plugins signal test-signal dev-check

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
DEV_UI_CMD ?= npm --prefix web run dev -- --open

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

# plugins builds the full plugin set — the four CGO_ENABLED=0 binaries
# plus the cgo signal plugin via its own isolated target — and nothing
# else (no npm, no kernel build), so it is usable from the dev loop
# without touching kernel/webui/build. It exists so `make dev` can
# guarantee fresh plugin binaries before the kernel starts without
# paying for a full `build`. It deliberately includes the cgo signal
# plugin: a machine without system sqlcipher fails here rather than
# starting a kernel with an incomplete plugin set. `build` delegates to
# this target so the plugin set is defined once.
plugins:
	mkdir -p bin/plugins
	go build -o bin/plugins/topos-plugin-paperless ./plugins/paperless
	go build -o bin/plugins/topos-plugin-silverbullet ./plugins/silverbullet
	go build -o bin/plugins/topos-plugin-proton ./plugins/proton
	go build -o bin/plugins/topos-plugin-mock ./plugins/mock
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

# test runs the test suite across all six workspace modules (sdk,
# paperless, silverbullet, proton, mock, signal) plus the root kernel
# module. Go workspaces scope "./..." to the module containing the
# working directory, so each module is tested explicitly rather than
# relying on a single "./..." from the repo root covering all of them.
test:
	CGO_ENABLED=0 go build ./... && go test ./...
	cd sdk && CGO_ENABLED=0 go build ./... && go test ./...
	cd plugins/paperless && CGO_ENABLED=0 go build ./... && go test ./...
	cd plugins/silverbullet && CGO_ENABLED=0 go build ./... && go test ./...
	cd plugins/proton && CGO_ENABLED=0 go build ./... && go test ./...
	cd plugins/mock && CGO_ENABLED=0 go build ./... && go test ./...
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

smoke: build
	./scripts/e2e-smoke.sh

# dev-check runs the hermetic behavioural guard for the `dev` recipe
# above (scripts/dev-guard-smoke.sh): squatter on the dev port, kernel
# dying during startup for another reason, and the unregressed happy
# path. Unlike `smoke`, this needs no network access, no live
# credentials, and no config file — it is safe to run at any time,
# including while a real kernel is up on 127.0.0.1:7777, since it only
# ever binds ephemeral ports it selects itself.
dev-check:
	./scripts/dev-guard-smoke.sh

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
