.PHONY: build test proto smoke dev

# build produces the kernel binary and the paperless plugin binary.
# The SvelteKit SPA build step (npm --prefix web ...) is added by plan
# 01-01 Task 3 once web/ exists.
build:
	CGO_ENABLED=0 go build -o bin/webspaces ./cmd/webspaces
	go build -o bin/plugins/webspaces-plugin-paperless ./plugins/paperless

# test runs the test suite across all three workspace modules. Go
# workspaces scope "./..." to the module containing the working directory,
# so each module is tested explicitly rather than relying on a single
# "./..." from the repo root covering all of them.
test:
	CGO_ENABLED=0 go build ./... && go test ./...
	cd sdk && CGO_ENABLED=0 go build ./... && go test ./...
	cd plugins/paperless && CGO_ENABLED=0 go build ./... && go test ./...

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

dev:
	@echo "Run in two terminals:"
	@echo "  1) go run ./cmd/webspaces serve"
	@echo "  2) npm --prefix web run dev"
