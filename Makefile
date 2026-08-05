.PHONY: build test proto smoke dev plugins signal test-signal

# build produces the SvelteKit SPA (embedded via kernel/webui/embed.go),
# the kernel binary, and the plugin binaries — webspaces-plugin-paperless,
# webspaces-plugin-silverbullet, webspaces-plugin-proton,
# webspaces-plugin-mock (the reference plugin PLUG-05 validates
# docs/plugin-contract.md against), and webspaces-plugin-signal (built by
# the "signal" target below — the first cgo-enabled plugin in this repo)
# — in that order. The kernel embed directive needs kernel/webui/build
# populated before `go build` runs to embed anything beyond the committed
# .gitkeep placeholder. The plugin set itself is built by the "plugins"
# target below so `dev` and `build` share one definition and can never
# drift apart.
build:
	npm --prefix web ci
	npm --prefix web run build
	CGO_ENABLED=0 go build -o bin/webspaces ./cmd/webspaces
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
	go build -o bin/plugins/webspaces-plugin-paperless ./plugins/paperless
	go build -o bin/plugins/webspaces-plugin-silverbullet ./plugins/silverbullet
	go build -o bin/plugins/webspaces-plugin-proton ./plugins/proton
	go build -o bin/plugins/webspaces-plugin-mock ./plugins/mock
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
	CGO_ENABLED=1 go build -tags libsqlcipher -o bin/plugins/webspaces-plugin-signal ./plugins/signal

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

# proto regenerates the sdk/gen Go stubs from proto/webspaces/v1/plugin.proto.
# Prefers buf; falls back to protoc + protoc-gen-go + protoc-gen-go-grpc
# when buf is not installed (see 01-RESEARCH.md Environment Availability).
proto:
	@if command -v buf >/dev/null 2>&1; then \
		cd proto && buf generate; \
	else \
		protoc --go_out=sdk/gen --go_opt=paths=source_relative \
			--go-grpc_out=sdk/gen --go-grpc_opt=paths=source_relative \
			--proto_path=proto \
			proto/webspaces/v1/plugin.proto; \
	fi

smoke: build
	./scripts/e2e-smoke.sh

# dev runs the kernel and the SvelteKit dev server together. The kernel
# binary is never embedded here — Vite's dev server proxies /api to
# 127.0.0.1:7777 (see web/vite.config.ts), so edits to either side hot
# reload independently.
dev: plugins
	@trap 'kill 0' EXIT INT TERM; \
	go run ./cmd/webspaces serve & \
	npm --prefix web run dev -- --open & \
	wait
