#!/usr/bin/env bash
# Shared helpers for the hermetic smoke gates (install-smoke.sh and
# simultaneity-smoke.sh). SOURCED, never executed — defines functions
# only. Extracted so the fixture-release builder and the free-port
# helper exist exactly once (15-05; the same one-place-only discipline
# the Makefile applies to its build definitions).

# smoke_free_port: prints an OS-assigned free TCP port on 127.0.0.1 —
# the same technique dev-guard-smoke.sh's free_port uses. Never a fixed
# port with a retry loop.
smoke_free_port() {
  python3 -c '
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
'
}

# smoke_build_fixture_release <workdir> <tag>: builds the mock plugin
# into bin/plugins/, links a kernel whose link-time manifest covers
# exactly that binary (a trusted-tier binary absent from the manifest is
# refused at launch — kernel/pluginhost.VerifyTrustedBinary), and
# assembles a release-shaped tree at <workdir>/release/download/<tag>/:
# flat basenames exactly like published release assets, plus a
# checksums.txt over the "topos" + "plugins/<name>" relative layout
# release.yml records. The built kernel is also left at <workdir>/topos
# for callers that boot it directly. Must be called from the repo root.
# The -ldflags -X symbol path must match the Makefile's
# MANIFEST_LDFLAGS_VAR.
smoke_build_fixture_release() {
  local workdir="$1" tag="$2"

  mkdir -p bin/plugins
  go build -o bin/plugins/topos-plugin-mock ./plugins/mock

  local manifest
  manifest="$(go run ./cmd/topos-manifest bin/plugins/topos-plugin-mock)"
  CGO_ENABLED=0 go build \
    -ldflags "-X github.com/davison/topos/kernel/pluginhost.buildManifest=$manifest" \
    -o "$workdir/topos" ./cmd/topos

  local release_dir="$workdir/release/download/$tag"
  mkdir -p "$release_dir"
  cp "$workdir/topos" "$release_dir/topos"
  cp bin/plugins/topos-plugin-mock "$release_dir/topos-plugin-mock"

  local chk="$workdir/chk"
  mkdir -p "$chk/plugins"
  cp "$workdir/topos" "$chk/topos"
  cp bin/plugins/topos-plugin-mock "$chk/plugins/topos-plugin-mock"
  (cd "$chk" && sha256sum topos plugins/topos-plugin-mock) > "$release_dir/checksums.txt"
}
