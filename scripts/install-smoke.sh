#!/usr/bin/env bash
# Hermetic gate for `make install` (INST-01/INST-03): builds a fixture
# release on local disk and installs from it with NO network and NO
# credentials, then asserts the far end of the path — the installed
# kernel at $PREFIX/bin/topos discovers and launches the installed
# plugin from $PREFIX/lib/topos/plugins with the STOCK relative
# `[plugins] dir = "plugins"` config value, no config edit.
#
# The fixture release is a directory tree shaped like the real GitHub
# release surface — download/<tag>/ holding the kernel, the plugin
# binary under its published flat basename, and a checksums.txt over the
# "topos" + "plugins/<name>" relative layout release.yml publishes —
# served to scripts/install.sh via its TOPOS_RELEASE_BASE_URL test seam
# as a file:// URL. That seam changes WHICH release is fetched, never
# which checks run: every checksum, allowlist, and placement rule runs
# identically here and against the real CDN.
#
# House style follows scripts/dev-guard-smoke.sh: required-tool
# preflight loop, mktemp -d work dir, cleanup trap, loud FAIL: messages
# naming the specific violation, ephemeral self-selected ports only
# (safe to run while a real kernel holds 127.0.0.1:7777).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

for tool in curl sha256sum mktemp python3 go; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "FAIL: required tool '$tool' not found on PATH" >&2
    exit 1
  fi
done

WORK="$(mktemp -d)"
KERNEL_PID=""

cleanup() {
  if [ -n "$KERNEL_PID" ] && kill -0 "$KERNEL_PID" 2>/dev/null; then
    kill "$KERNEL_PID" 2>/dev/null || true
    wait "$KERNEL_PID" 2>/dev/null || true
  fi
  rm -rf "$WORK"
}
trap cleanup EXIT

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

free_port() {
  python3 -c '
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
'
}

# ---------------------------------------------------------------------
# Fixture release: build the mock plugin, link a kernel whose manifest
# covers exactly that binary, and assemble the release-shaped tree.
# ---------------------------------------------------------------------
TAG="v0.0.0-install-smoke"

echo "==> building fixture release ($TAG)"
mkdir -p bin/plugins
go build -o bin/plugins/topos-plugin-mock ./plugins/mock

# Same generator invocation the Makefile's MANIFEST_GEN_* variables use,
# over exactly the one fixture binary; the -ldflags -X symbol path must
# match the Makefile's MANIFEST_LDFLAGS_VAR — a trusted-tier binary
# absent from the kernel's link-time manifest is refused at launch
# (kernel/pluginhost.VerifyTrustedBinary), which is precisely why the
# fixture kernel cannot be an unmanifested build.
MANIFEST="$(go run ./cmd/topos-manifest bin/plugins/topos-plugin-mock)"
CGO_ENABLED=0 go build \
  -ldflags "-X github.com/davison/topos/kernel/pluginhost.buildManifest=$MANIFEST" \
  -o "$WORK/topos" ./cmd/topos

# Assemble download/<tag>/: flat basenames, exactly like the published
# release assets, plus a checksums.txt over the RELATIVE layout
# ("topos", "plugins/topos-plugin-mock") that release.yml records.
RELEASE_DIR="$WORK/release/download/$TAG"
mkdir -p "$RELEASE_DIR"
cp "$WORK/topos" "$RELEASE_DIR/topos"
cp bin/plugins/topos-plugin-mock "$RELEASE_DIR/topos-plugin-mock"

CHK="$WORK/chk"
mkdir -p "$CHK/plugins"
cp "$WORK/topos" "$CHK/topos"
cp bin/plugins/topos-plugin-mock "$CHK/plugins/topos-plugin-mock"
(cd "$CHK" && sha256sum topos plugins/topos-plugin-mock) > "$RELEASE_DIR/checksums.txt"

# ---------------------------------------------------------------------
# Case: happy-path install into a temp PREFIX from the file:// fixture.
# ---------------------------------------------------------------------
echo "==> Case: install from fixture release"
PREFIX_DIR="$WORK/prefix"

PREFIX="$PREFIX_DIR" TOPOS_RELEASE_BASE_URL="file://$WORK/release" \
  ./scripts/install.sh "$TAG"

for path in "$PREFIX_DIR/bin/topos" "$PREFIX_DIR/lib/topos/plugins/topos-plugin-mock"; do
  if [ ! -f "$path" ]; then
    fail "expected installed file missing: $path"
  fi
  mode="$(stat -c '%a' "$path")"
  if [ "$mode" != "755" ]; then
    fail "expected mode 755 on $path, got $mode"
  fi
done

# The installer must never create $PREFIX/bin/plugins — its absence is
# what proves the kernel's resolution below reached the lib-tree
# sibling, not a directory the installer happened to make.
if [ -e "$PREFIX_DIR/bin/plugins" ]; then
  fail "\$PREFIX/bin/plugins exists after install — the installer must not create it"
fi

echo "==> Case PASS: install from fixture release"

# ---------------------------------------------------------------------
# Case: the installed kernel launches the installed plugin with the
# stock relative [plugins] dir — the INST-03 claim, end to end.
# ---------------------------------------------------------------------
echo "==> Case: installed kernel finds installed plugins (stock config)"
PORT="$(free_port)"

# Stock relative `dir = "plugins"` (kernel/config.DefaultPluginsDir) —
# deliberately NOT an absolute path: this case exists to prove the
# installed layout needs no config edit. index and all HOME/XDG state
# live inside the temp tree so the fixture kernel can never touch the
# operator's real config, index, or plugin set.
cat > "$WORK/config.toml" <<EOF
[server]
listen = "127.0.0.1:$PORT"

[index]
path = "$WORK/index/index.db"

[plugins]
dir = "plugins"

[sync]
interval = "1h"

[sources.mock]
plugin = "topos-plugin-mock"
base_url = "install-smoke-unused"
token = "install-smoke-unused"
EOF

mkdir -p "$WORK/home" "$WORK/xdg-config" "$WORK/xdg-data"
HOME="$WORK/home" XDG_CONFIG_HOME="$WORK/xdg-config" XDG_DATA_HOME="$WORK/xdg-data" \
  "$PREFIX_DIR/bin/topos" serve --config "$WORK/config.toml" \
  >"$WORK/kernel.log" 2>&1 &
KERNEL_PID=$!

# Poll GET /api/sources until the mock instance reports a landed first
# sync: present, no launch_failure, syncing false, non-empty
# last_status. A manifest-unverified or binary-not-found outcome
# surfaces as launch_failure (or the instance never landing) and fails
# this loop. 60s budget — the fixture kernel is a local binary with one
# in-process mock source, so the normal case lands in well under 5s.
DEADLINE=60
i=0
while :; do
  if ! kill -0 "$KERNEL_PID" 2>/dev/null; then
    echo "--- kernel log ---" >&2
    cat "$WORK/kernel.log" >&2 || true
    fail "installed kernel exited before the mock source landed"
  fi
  VERDICT="$(curl -fsS "http://127.0.0.1:$PORT/api/sources" 2>/dev/null | python3 -c '
import json, sys
try:
    body = json.load(sys.stdin)
except Exception:
    print("pending"); raise SystemExit
for s in body.get("sources") or []:
    if s.get("name") == "mock":
        if s.get("launch_failure"):
            print("launch_failure: " + s["launch_failure"]); break
        if s.get("syncing") is False and s.get("last_status"):
            print("landed"); break
        print("pending"); break
else:
    print("pending")
' || echo "pending")"
  case "$VERDICT" in
    landed) break ;;
    launch_failure:*)
      echo "--- kernel log ---" >&2
      cat "$WORK/kernel.log" >&2 || true
      fail "mock instance was refused at launch (${VERDICT}) — the installed kernel did not verify/find its installed plugin"
      ;;
  esac
  i=$((i + 1))
  if [ "$i" -ge "$DEADLINE" ]; then
    echo "--- kernel log ---" >&2
    cat "$WORK/kernel.log" >&2 || true
    fail "mock source did not report a landed sync within ${DEADLINE}s"
  fi
  sleep 1
done

kill "$KERNEL_PID" 2>/dev/null || true
wait "$KERNEL_PID" 2>/dev/null || true
KERNEL_PID=""

echo "==> Case PASS: installed kernel finds installed plugins (stock config)"

echo "==> install-smoke: all cases passed"
