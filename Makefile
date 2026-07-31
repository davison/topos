.PHONY: build test proto smoke dev

# build produces the SvelteKit SPA (embedded via kernel/webui/embed.go),
# the kernel binary, and the plugin binaries — webspaces-plugin-paperless,
# webspaces-plugin-silverbullet, webspaces-plugin-proton, and
# webspaces-plugin-mock (the reference plugin PLUG-05 validates
# docs/plugin-contract.md against) — in that order. The kernel embed
# directive needs kernel/webui/build populated before `go build` runs to
# embed anything beyond the committed .gitkeep placeholder.
build:
	npm --prefix web ci
	npm --prefix web run build
	CGO_ENABLED=0 go build -o bin/webspaces ./cmd/webspaces
	go build -o bin/plugins/webspaces-plugin-paperless ./plugins/paperless
	go build -o bin/plugins/webspaces-plugin-silverbullet ./plugins/silverbullet
	go build -o bin/plugins/webspaces-plugin-proton ./plugins/proton
	go build -o bin/plugins/webspaces-plugin-mock ./plugins/mock

# test runs the test suite across all five workspace modules (sdk,
# paperless, silverbullet, proton, mock — the webspaces-plugin-mock
# module built above) plus the root kernel module. Go workspaces scope
# "./..." to the module containing the working directory, so each module
# is tested explicitly rather than relying on a single "./..." from the
# repo root covering all of them.
test:
	CGO_ENABLED=0 go build ./... && go test ./...
	cd sdk && CGO_ENABLED=0 go build ./... && go test ./...
	cd plugins/paperless && CGO_ENABLED=0 go build ./... && go test ./...
	cd plugins/silverbullet && CGO_ENABLED=0 go build ./... && go test ./...
	cd plugins/proton && CGO_ENABLED=0 go build ./... && go test ./...
	cd plugins/mock && CGO_ENABLED=0 go build ./... && go test ./...

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
dev:
	@trap 'kill 0' EXIT INT TERM; \
	go run ./cmd/webspaces serve & \
	npm --prefix web run dev -- --open & \
	wait
