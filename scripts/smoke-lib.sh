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

# smoke_build_fixture_release_with_provenance <workdir> <tag>: sibling of
# smoke_build_fixture_release (above) whose release ALSO carries a signed
# release manifest for topos-plugin-mock plus a topos-provenance CLI
# relinked to trust the throwaway key that signed it (16-05-PLAN.md
# Task 1, D-09's install-time arm) — install.sh's provenance-verification
# step resolves and uses exactly this staged CLI, since no real release's
# actual topos-plugins-2026a private key is available to sign fixtures
# with outside the topos-plugins repository's own CI (16-04-SUMMARY.md).
# Mirrors scripts/provenance-smoke.sh's own keygen -> relink -> sign
# sequence, so this helper is ALSO an executable proof that the link-time
# provenanceKeysExtra seam (D-12) works, not merely an in-process
# override. Writes the signed manifest's basename to
# <workdir>/provenance-manifest-name so a caller that needs to tamper the
# release afterward (and must regenerate checksums.txt over the tampered
# bytes) can find the manifest/signature files by name without
# re-deriving cmd/topos-provenance's own naming convention.
smoke_build_fixture_release_with_provenance() {
  local workdir="$1" tag="$2"

  mkdir -p bin/plugins
  go build -o bin/plugins/topos-plugin-mock ./plugins/mock

  # The kernel build carries the SAME link-time manifest the base fixture
  # (smoke_build_fixture_release) uses — install-smoke.sh's own INST-03
  # launch check already proves that pre-existing arm; this helper exists
  # to prove the NEW install-time provenance-verification step, not to
  # re-derive kernel launch trust.
  local manifest
  manifest="$(go run ./cmd/topos-manifest bin/plugins/topos-plugin-mock)"
  CGO_ENABLED=0 go build \
    -ldflags "-X github.com/davison/topos/kernel/pluginhost.buildManifest=$manifest" \
    -o "$workdir/topos" ./cmd/topos

  go build -o "$workdir/topos-provenance-unsigned" ./cmd/topos-provenance

  mkdir -p "$workdir/keys"
  local key_spec
  key_spec="$("$workdir/topos-provenance-unsigned" keygen --key-id install-smoke --out-dir "$workdir/keys")"

  CGO_ENABLED=0 go build \
    -ldflags "-X github.com/davison/topos/kernel/pluginhost.provenanceKeysExtra=$key_spec" \
    -o "$workdir/topos-provenance" ./cmd/topos-provenance

  local sign_dir="$workdir/signed"
  mkdir -p "$sign_dir"
  "$workdir/topos-provenance" sign \
    --key-id install-smoke \
    --repo davison/topos-install-smoke \
    --tag "$tag" \
    --version 0.0.0 \
    --key-file "$workdir/keys/install-smoke.key" \
    --out-dir "$sign_dir" \
    bin/plugins/topos-plugin-mock \
    >"$workdir/sign.out"

  local manifest_path sig_path manifest_name sig_name
  manifest_path="$(find "$sign_dir" -maxdepth 1 -name '*.provenance.json')"
  sig_path="$(find "$sign_dir" -maxdepth 1 -name '*.provenance.sig')"
  manifest_name="$(basename "$manifest_path")"
  sig_name="$(basename "$sig_path")"
  printf '%s\n' "$manifest_name" > "$workdir/provenance-manifest-name"

  local release_dir="$workdir/release/download/$tag"
  mkdir -p "$release_dir"
  cp "$workdir/topos" "$release_dir/topos"
  cp "$workdir/topos-provenance" "$release_dir/topos-provenance"
  cp bin/plugins/topos-plugin-mock "$release_dir/topos-plugin-mock"
  cp "$manifest_path" "$release_dir/$manifest_name"
  cp "$sig_path" "$release_dir/$sig_name"

  local chk="$workdir/chk"
  mkdir -p "$chk/plugins"
  cp "$workdir/topos" "$chk/topos"
  cp "$workdir/topos-provenance" "$chk/topos-provenance"
  cp bin/plugins/topos-plugin-mock "$chk/plugins/topos-plugin-mock"
  cp "$manifest_path" "$chk/plugins/$manifest_name"
  cp "$sig_path" "$chk/plugins/$sig_name"
  (cd "$chk" && sha256sum topos topos-provenance "plugins/topos-plugin-mock" "plugins/$manifest_name" "plugins/$sig_name") \
    > "$release_dir/checksums.txt"
}
