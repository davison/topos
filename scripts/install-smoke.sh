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
#
# Beyond the happy path, five named hardening cases pin every refusal
# and repair behaviour: corrupted asset (checksum mismatch aborts,
# $PREFIX stays empty), traversal-shaped manifest (a checksums.txt path
# escaping the staging tree is rejected by name, nothing created),
# unwritable prefix (fails loud naming the directory and the sudo
# re-run, never escalates, target stays read-only and empty),
# idempotent re-run (two installs of one tag leave byte-identical
# prefix trees), and live replacement (re-installing over a RUNNING
# kernel succeeds and lands the new file — the case that would fail if
# placement ever wrote directly over the destination instead of
# renaming into it).

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

# run_install_expect_fail <prefix> <base-url>: runs install.sh expecting
# a non-zero exit; captures combined output into $INSTALL_OUT and the
# exit status into $INSTALL_RC. A zero exit is the caller's assertion to
# make — this helper only ever records.
run_install() {
  INSTALL_RC=0
  INSTALL_OUT="$(PREFIX="$1" TOPOS_RELEASE_BASE_URL="$2" ./scripts/install.sh "$TAG" 2>&1)" \
    || INSTALL_RC=$?
}

# assert_prefix_untouched <prefix>: asserts no kernel binary and no
# plugin entries were placed. The writability preflight may legitimately
# have created the (empty) directories themselves — a failed install's
# defined state is "no FILES placed", not "no directories exist".
assert_prefix_untouched() {
  local prefix="$1"
  if [ -e "$prefix/bin/topos" ]; then
    fail "a failed install left $prefix/bin/topos behind"
  fi
  if [ -d "$prefix/lib/topos/plugins" ] && [ -n "$(ls -A "$prefix/lib/topos/plugins")" ]; then
    fail "a failed install left entries in $prefix/lib/topos/plugins"
  fi
}

# ---------------------------------------------------------------------
# Case: corrupted asset — a byte appended to one published file makes
# its recorded digest stale; the install must abort naming that asset
# and place nothing.
# ---------------------------------------------------------------------
echo "==> Case: corrupted asset"
CORRUPT_BASE="$WORK/release-corrupt"
mkdir -p "$CORRUPT_BASE/download"
cp -r "$WORK/release/download/$TAG" "$CORRUPT_BASE/download/$TAG"
printf 'x' >> "$CORRUPT_BASE/download/$TAG/topos-plugin-mock"

run_install "$WORK/prefix-corrupt" "file://$CORRUPT_BASE"
if [ "$INSTALL_RC" -eq 0 ]; then
  fail "corrupted asset: install exited 0, expected a checksum refusal"
fi
if ! printf '%s' "$INSTALL_OUT" | grep -q "topos-plugin-mock"; then
  fail "corrupted asset: refusal did not name the failing asset
$INSTALL_OUT"
fi
assert_prefix_untouched "$WORK/prefix-corrupt"
echo "==> Case PASS: corrupted asset"

# ---------------------------------------------------------------------
# Case: traversal-shaped manifest — a checksums.txt line whose path
# escapes the staging directory must be rejected by name before any
# file is created (T-15-02).
# ---------------------------------------------------------------------
echo "==> Case: traversal-shaped manifest"
TRAVERSAL_BASE="$WORK/release-traversal"
mkdir -p "$TRAVERSAL_BASE/download"
cp -r "$WORK/release/download/$TAG" "$TRAVERSAL_BASE/download/$TAG"
BOGUS_HASH="$(printf 'traversal' | sha256sum | cut -d' ' -f1)"
printf '%s  plugins/../../../traversal-escape\n' "$BOGUS_HASH" \
  >> "$TRAVERSAL_BASE/download/$TAG/checksums.txt"

run_install "$WORK/prefix-traversal" "file://$TRAVERSAL_BASE"
if [ "$INSTALL_RC" -eq 0 ]; then
  fail "traversal-shaped manifest: install exited 0, expected a rejection"
fi
if ! printf '%s' "$INSTALL_OUT" | grep -q "traversal-escape"; then
  fail "traversal-shaped manifest: rejection did not name the offending line
$INSTALL_OUT"
fi
if [ -e "$WORK/traversal-escape" ] || [ -e "$WORK/release-traversal/traversal-escape" ]; then
  fail "traversal-shaped manifest: a file escaped the staging tree"
fi
assert_prefix_untouched "$WORK/prefix-traversal"
echo "==> Case PASS: traversal-shaped manifest"

# ---------------------------------------------------------------------
# Case: unwritable prefix — must fail loud naming the directory and the
# sudo re-run form, must never escalate itself, and must leave the
# read-only target exactly as it found it (an installer that had
# escalated would have written into it).
# ---------------------------------------------------------------------
echo "==> Case: unwritable prefix"
RO_PREFIX="$WORK/prefix-readonly"
mkdir -p "$RO_PREFIX"
chmod 555 "$RO_PREFIX"

run_install "$RO_PREFIX" "file://$WORK/release"
RO_RC="$INSTALL_RC"
RO_OUT="$INSTALL_OUT"
chmod_restored=false
if [ "$RO_RC" -eq 0 ]; then
  chmod 755 "$RO_PREFIX"; chmod_restored=true
  fail "unwritable prefix: install exited 0, expected a refusal"
fi
if ! printf '%s' "$RO_OUT" | grep -q "$RO_PREFIX"; then
  chmod 755 "$RO_PREFIX"; chmod_restored=true
  fail "unwritable prefix: refusal did not name the unwritable directory
$RO_OUT"
fi
if ! printf '%s' "$RO_OUT" | grep -q "sudo make install"; then
  chmod 755 "$RO_PREFIX"; chmod_restored=true
  fail "unwritable prefix: refusal did not name the sudo make install re-run
$RO_OUT"
fi
MODE_AFTER="$(stat -c '%a' "$RO_PREFIX")"
if [ "$MODE_AFTER" != "555" ]; then
  fail "unwritable prefix: target directory mode changed to $MODE_AFTER — the installer must not alter it"
fi
if [ -n "$(ls -A "$RO_PREFIX")" ]; then
  fail "unwritable prefix: target directory is no longer empty — an installer that escalated would have written into it"
fi
[ "$chmod_restored" = true ] || chmod 755 "$RO_PREFIX"
echo "==> Case PASS: unwritable prefix"

# ---------------------------------------------------------------------
# Case: idempotent re-run — installing the same tag twice into one
# prefix succeeds both times and leaves byte-identical trees (the
# supported repair path: re-verify, re-place, no bookkeeping).
# ---------------------------------------------------------------------
echo "==> Case: idempotent re-run"
IDEM_PREFIX="$WORK/prefix-idem"

# manifest_of_prefix <dir>: recursive path+SHA-256 manifest, sorted for
# stable comparison. xargs -r so an empty tree yields an empty manifest
# rather than sha256sum reading stdin.
manifest_of_prefix() {
  (cd "$1" && find . -type f | LC_ALL=C sort | xargs -r sha256sum)
}

run_install "$IDEM_PREFIX" "file://$WORK/release"
if [ "$INSTALL_RC" -ne 0 ]; then
  fail "idempotent re-run: first install failed (rc=$INSTALL_RC)
$INSTALL_OUT"
fi
manifest_of_prefix "$IDEM_PREFIX" > "$WORK/idem-manifest-1"

run_install "$IDEM_PREFIX" "file://$WORK/release"
if [ "$INSTALL_RC" -ne 0 ]; then
  fail "idempotent re-run: second install failed (rc=$INSTALL_RC)
$INSTALL_OUT"
fi
manifest_of_prefix "$IDEM_PREFIX" > "$WORK/idem-manifest-2"

if ! cmp -s "$WORK/idem-manifest-1" "$WORK/idem-manifest-2"; then
  diff "$WORK/idem-manifest-1" "$WORK/idem-manifest-2" >&2 || true
  fail "idempotent re-run: the two installs left different bytes in $IDEM_PREFIX"
fi
echo "==> Case PASS: idempotent re-run"

# ---------------------------------------------------------------------
# Case: live replacement — re-running the install while the installed
# kernel is RUNNING must succeed and land the new file. This is the
# case that fails if placement ever writes directly over the
# destination path (text-file-busy / torn write) instead of renaming
# into it (T-15-04).
# ---------------------------------------------------------------------
echo "==> Case: live replacement"
LIVE_PORT="$(free_port)"
cat > "$WORK/config-live.toml" <<EOF
[server]
listen = "127.0.0.1:$LIVE_PORT"

[index]
path = "$WORK/index-live/index.db"

[plugins]
dir = "plugins"

[sync]
interval = "1h"

[sources.mock]
plugin = "topos-plugin-mock"
base_url = "install-smoke-unused"
token = "install-smoke-unused"
EOF

HOME="$WORK/home" XDG_CONFIG_HOME="$WORK/xdg-config" XDG_DATA_HOME="$WORK/xdg-data" \
  "$PREFIX_DIR/bin/topos" serve --config "$WORK/config-live.toml" \
  >"$WORK/kernel-live.log" 2>&1 &
KERNEL_PID=$!

i=0
until curl -fsS "http://127.0.0.1:$LIVE_PORT/api/sources" >/dev/null 2>&1; do
  if ! kill -0 "$KERNEL_PID" 2>/dev/null; then
    echo "--- kernel log ---" >&2
    cat "$WORK/kernel-live.log" >&2 || true
    fail "live replacement: kernel exited before listening"
  fi
  i=$((i + 1))
  if [ "$i" -ge 30 ]; then
    fail "live replacement: kernel never listened on 127.0.0.1:$LIVE_PORT"
  fi
  sleep 1
done

INODE_BEFORE="$(stat -c '%i' "$PREFIX_DIR/bin/topos")"
run_install "$PREFIX_DIR" "file://$WORK/release"
if [ "$INSTALL_RC" -ne 0 ]; then
  fail "live replacement: install over a running kernel failed (rc=$INSTALL_RC)
$INSTALL_OUT"
fi
INODE_AFTER="$(stat -c '%i' "$PREFIX_DIR/bin/topos")"
if [ "$INODE_BEFORE" = "$INODE_AFTER" ]; then
  fail "live replacement: on-disk kernel is not a new file — placement must rename a fresh file into place, never write over the destination"
fi
if ! kill -0 "$KERNEL_PID" 2>/dev/null; then
  fail "live replacement: the running kernel died during the re-install"
fi

kill "$KERNEL_PID" 2>/dev/null || true
wait "$KERNEL_PID" 2>/dev/null || true
KERNEL_PID=""
echo "==> Case PASS: live replacement"

# ---------------------------------------------------------------------
# Case: uninstall data-safety cycle — the real INST-05 gate. A seeded
# home/XDG tree (config, index, a plugin store) must be byte-identical
# across a full install+uninstall cycle, and the prefix must be clean
# afterwards.
# ---------------------------------------------------------------------
echo "==> Case: uninstall data-safety cycle"
SEED_HOME="$WORK/seed-home"
SEED_CONFIG="$WORK/seed-xdg-config"
SEED_DATA="$WORK/seed-xdg-data"
mkdir -p "$SEED_HOME" "$SEED_CONFIG/topos" "$SEED_DATA/topos/whatsapp-store"
echo '[server]' > "$SEED_CONFIG/topos/config.toml"
printf 'operator index bytes' > "$SEED_DATA/topos/index.db"
printf 'linked-device session' > "$SEED_DATA/topos/whatsapp-store/session.db"

seed_manifest() {
  { manifest_of_prefix "$SEED_HOME"; manifest_of_prefix "$SEED_CONFIG"; manifest_of_prefix "$SEED_DATA"; }
}

seed_manifest > "$WORK/seed-before"

CYCLE_PREFIX="$WORK/prefix-cycle"
HOME="$SEED_HOME" XDG_CONFIG_HOME="$SEED_CONFIG" XDG_DATA_HOME="$SEED_DATA" \
  PREFIX="$CYCLE_PREFIX" TOPOS_RELEASE_BASE_URL="file://$WORK/release" \
  ./scripts/install.sh "$TAG" >/dev/null

HOME="$SEED_HOME" XDG_CONFIG_HOME="$SEED_CONFIG" XDG_DATA_HOME="$SEED_DATA" \
  PREFIX="$CYCLE_PREFIX" ./scripts/uninstall.sh >"$WORK/uninstall-cycle.out"

seed_manifest > "$WORK/seed-after"
if ! cmp -s "$WORK/seed-before" "$WORK/seed-after"; then
  echo "FAIL: the install+uninstall cycle changed the operator's seeded tree:" >&2
  diff "$WORK/seed-before" "$WORK/seed-after" >&2 || true
  exit 1
fi
if [ -e "$CYCLE_PREFIX/bin/topos" ]; then
  fail "uninstall left the kernel binary at $CYCLE_PREFIX/bin/topos"
fi
if [ -e "$CYCLE_PREFIX/lib/topos/plugins" ]; then
  fail "uninstall left the plugins directory at $CYCLE_PREFIX/lib/topos/plugins"
fi
echo "==> Case PASS: uninstall data-safety cycle"

# ---------------------------------------------------------------------
# Case: idempotent uninstall — a second run exits 0, reports nothing
# left to remove, and changes nothing on disk.
# ---------------------------------------------------------------------
echo "==> Case: idempotent uninstall"
manifest_of_prefix "$CYCLE_PREFIX" > "$WORK/uninstall-manifest-1"
UN2_RC=0
UN2_OUT="$(PREFIX="$CYCLE_PREFIX" ./scripts/uninstall.sh 2>&1)" || UN2_RC=$?
if [ "$UN2_RC" -ne 0 ]; then
  fail "second uninstall exited $UN2_RC, expected 0
$UN2_OUT"
fi
if ! printf '%s' "$UN2_OUT" | grep -q "nothing left to remove"; then
  fail "second uninstall did not report that nothing was left to remove
$UN2_OUT"
fi
manifest_of_prefix "$CYCLE_PREFIX" > "$WORK/uninstall-manifest-2"
if ! cmp -s "$WORK/uninstall-manifest-1" "$WORK/uninstall-manifest-2"; then
  fail "second uninstall changed bytes in $CYCLE_PREFIX"
fi
echo "==> Case PASS: idempotent uninstall"

# ---------------------------------------------------------------------
# Case: foreign file in the plugins directory — a hand-placed file
# survives, its directory survives, and the uninstall says so.
# ---------------------------------------------------------------------
echo "==> Case: uninstall leaves a foreign file"
FOREIGN_PREFIX="$WORK/prefix-foreign"
PREFIX="$FOREIGN_PREFIX" TOPOS_RELEASE_BASE_URL="file://$WORK/release" \
  ./scripts/install.sh "$TAG" >/dev/null
printf 'operator notes — not ours to delete' > "$FOREIGN_PREFIX/lib/topos/plugins/my-notes.txt"
FOREIGN_DIGEST_BEFORE="$(sha256sum "$FOREIGN_PREFIX/lib/topos/plugins/my-notes.txt" | cut -d' ' -f1)"

FOREIGN_RC=0
FOREIGN_OUT="$(PREFIX="$FOREIGN_PREFIX" ./scripts/uninstall.sh 2>&1)" || FOREIGN_RC=$?
if [ "$FOREIGN_RC" -ne 0 ]; then
  fail "uninstall with a foreign file exited $FOREIGN_RC, expected 0
$FOREIGN_OUT"
fi
if [ ! -f "$FOREIGN_PREFIX/lib/topos/plugins/my-notes.txt" ]; then
  fail "uninstall removed the operator's hand-placed file — the removal set must be closed"
fi
FOREIGN_DIGEST_AFTER="$(sha256sum "$FOREIGN_PREFIX/lib/topos/plugins/my-notes.txt" | cut -d' ' -f1)"
if [ "$FOREIGN_DIGEST_BEFORE" != "$FOREIGN_DIGEST_AFTER" ]; then
  fail "the hand-placed file's digest changed across uninstall"
fi
if [ ! -d "$FOREIGN_PREFIX/lib/topos/plugins" ]; then
  fail "uninstall removed a non-empty plugins directory — directory cleanup must be non-recursive rmdir only"
fi
if ! printf '%s' "$FOREIGN_OUT" | grep -q "left in place (not empty)"; then
  fail "uninstall did not report the surviving directory and its contents
$FOREIGN_OUT"
fi
echo "==> Case PASS: uninstall leaves a foreign file"

# ---------------------------------------------------------------------
# Case: uninstall under a live kernel — removal is by unlink, so the
# running process keeps serving on its already-open files.
# ---------------------------------------------------------------------
echo "==> Case: uninstall under a live kernel"
UNLIVE_PORT="$(free_port)"
cat > "$WORK/config-unlive.toml" <<EOF
[server]
listen = "127.0.0.1:$UNLIVE_PORT"

[index]
path = "$WORK/index-unlive/index.db"

[plugins]
dir = "plugins"

[sync]
interval = "1h"

[sources.mock]
plugin = "topos-plugin-mock"
base_url = "install-smoke-unused"
token = "install-smoke-unused"
EOF

HOME="$WORK/home" XDG_CONFIG_HOME="$WORK/xdg-config" XDG_DATA_HOME="$WORK/xdg-data" \
  "$PREFIX_DIR/bin/topos" serve --config "$WORK/config-unlive.toml" \
  >"$WORK/kernel-unlive.log" 2>&1 &
KERNEL_PID=$!

i=0
until curl -fsS "http://127.0.0.1:$UNLIVE_PORT/api/sources" >/dev/null 2>&1; do
  if ! kill -0 "$KERNEL_PID" 2>/dev/null; then
    echo "--- kernel log ---" >&2
    cat "$WORK/kernel-unlive.log" >&2 || true
    fail "uninstall under a live kernel: kernel exited before listening"
  fi
  i=$((i + 1))
  if [ "$i" -ge 30 ]; then
    fail "uninstall under a live kernel: kernel never listened on 127.0.0.1:$UNLIVE_PORT"
  fi
  sleep 1
done

UNLIVE_RC=0
UNLIVE_OUT="$(PREFIX="$PREFIX_DIR" ./scripts/uninstall.sh 2>&1)" || UNLIVE_RC=$?
if [ "$UNLIVE_RC" -ne 0 ]; then
  fail "uninstall under a live kernel exited $UNLIVE_RC, expected 0
$UNLIVE_OUT"
fi
if ! curl -fsS "http://127.0.0.1:$UNLIVE_PORT/api/sources" >/dev/null 2>&1; then
  fail "the running kernel stopped answering after the uninstall — unlink-based removal must leave the live process untouched"
fi
if ! kill -0 "$KERNEL_PID" 2>/dev/null; then
  fail "the running kernel died during the uninstall"
fi

kill "$KERNEL_PID" 2>/dev/null || true
wait "$KERNEL_PID" 2>/dev/null || true
KERNEL_PID=""
echo "==> Case PASS: uninstall under a live kernel"

# ---------------------------------------------------------------------
# Case: toolchain tripwire — the base install must be download-and-copy
# ONLY (INST-04's other half). Failing shims for the Go tool, the
# conventional C compiler names, and the web build's package manager go
# FIRST on PATH; if the install ever reaches for any of them, the shim
# records a marker and fails. This proves the no-toolchain property as
# a behaviour, not as a promise about the script's text (T-15-14).
# `make` itself is deliberately excluded — the recipe is invoked
# through it.
# ---------------------------------------------------------------------
echo "==> Case: toolchain tripwire"
SHIM_DIR="$WORK/toolchain-shims"
TRIPWIRE_MARKER="$WORK/toolchain-tripped"
mkdir -p "$SHIM_DIR"
for tool in go cc gcc clang npm node; do
  cat > "$SHIM_DIR/$tool" <<SHIM
#!/bin/sh
echo "TRIPWIRE: \$0 invoked with: \$*" >> "$TRIPWIRE_MARKER"
exit 1
SHIM
  chmod +x "$SHIM_DIR/$tool"
done

TRIP_RC=0
TRIP_OUT="$(PATH="$SHIM_DIR:$PATH" PREFIX="$WORK/prefix-tripwire" \
  TOPOS_RELEASE_BASE_URL="file://$WORK/release" \
  ./scripts/install.sh "$TAG" 2>&1)" || TRIP_RC=$?
if [ "$TRIP_RC" -ne 0 ]; then
  fail "toolchain tripwire: the base install failed with hostile toolchain shims on PATH (rc=$TRIP_RC) — it must not need any of them
$TRIP_OUT"
fi
if [ -f "$TRIPWIRE_MARKER" ]; then
  echo "FAIL: toolchain tripwire: the base install invoked a toolchain:" >&2
  cat "$TRIPWIRE_MARKER" >&2
  exit 1
fi
if [ ! -f "$WORK/prefix-tripwire/bin/topos" ]; then
  fail "toolchain tripwire: install exited 0 but placed no kernel"
fi
echo "==> Case PASS: toolchain tripwire"

# ---------------------------------------------------------------------
# Case: Signal removal — uninstall-signal reaches exactly one path. A
# planted unrelated file and the directory itself both survive, and a
# second run is a clean, visible no-op (T-15-15).
# ---------------------------------------------------------------------
echo "==> Case: Signal removal"
SIG_EXT="$WORK/signal-external"
mkdir -p "$SIG_EXT"
printf 'not a signal binary — not ours to delete' > "$SIG_EXT/other-external-plugin"
printf 'fake signal binary bytes' > "$SIG_EXT/topos-plugin-signal"
SIG_MARKER_DIGEST_BEFORE="$(sha256sum "$SIG_EXT/other-external-plugin" | cut -d' ' -f1)"

SIGRM_RC=0
SIGRM_OUT="$(TOPOS_EXTERNAL_PLUGINS_DIR="$SIG_EXT" ./scripts/install-signal.sh --uninstall 2>&1)" || SIGRM_RC=$?
if [ "$SIGRM_RC" -ne 0 ]; then
  fail "Signal removal exited $SIGRM_RC, expected 0
$SIGRM_OUT"
fi
if [ -e "$SIG_EXT/topos-plugin-signal" ]; then
  fail "Signal removal left the Signal binary in place"
fi
if [ ! -f "$SIG_EXT/other-external-plugin" ]; then
  fail "Signal removal deleted an unrelated file — the removal set must be exactly one path"
fi
SIG_MARKER_DIGEST_AFTER="$(sha256sum "$SIG_EXT/other-external-plugin" | cut -d' ' -f1)"
if [ "$SIG_MARKER_DIGEST_BEFORE" != "$SIG_MARKER_DIGEST_AFTER" ]; then
  fail "Signal removal changed an unrelated file's bytes"
fi
if [ ! -d "$SIG_EXT" ]; then
  fail "Signal removal removed the external directory itself"
fi

SIGRM2_RC=0
SIGRM2_OUT="$(TOPOS_EXTERNAL_PLUGINS_DIR="$SIG_EXT" ./scripts/install-signal.sh --uninstall 2>&1)" || SIGRM2_RC=$?
if [ "$SIGRM2_RC" -ne 0 ]; then
  fail "second Signal removal exited $SIGRM2_RC, expected 0
$SIGRM2_OUT"
fi
if ! printf '%s' "$SIGRM2_OUT" | grep -q "already absent"; then
  fail "second Signal removal did not report an already-absent outcome
$SIGRM2_OUT"
fi
echo "==> Case PASS: Signal removal"

# ---------------------------------------------------------------------
# Case: latest-resolution validator (offline) — source the script (the
# source-guard seam: defines functions, runs nothing) and drive
# validate_latest_url with a table of effective URLs. Exactly one shape
# is accepted; every refusal names a distinct reason (T-15-06/T-15-07).
# ---------------------------------------------------------------------
echo "==> Case: latest-resolution validator (offline)"

# The source-guard itself: sourcing must produce no output and start no
# download — anything on stdout/stderr here is a guard regression.
SOURCE_OUT="$( (source ./scripts/install.sh) 2>&1 )"
if [ -n "$SOURCE_OUT" ]; then
  fail "sourcing scripts/install.sh produced output — the source-guard is broken:
$SOURCE_OUT"
fi

# validate_case <expect: accept|refuse> <input-url> <required-fragment>
# Runs the validator in a sourced subshell; on accept, the emitted tag
# must equal <required-fragment>; on refuse, the exit must be non-zero
# and the message must contain <required-fragment>.
validate_case() {
  local expect="$1" input="$2" required="$3"
  local rc=0 out
  out="$( (source ./scripts/install.sh && validate_latest_url "$input") 2>&1 )" || rc=$?
  if [ "$expect" = "accept" ]; then
    if [ "$rc" -ne 0 ]; then
      fail "validator refused an acceptable URL '$input': $out"
    fi
    if [ "$out" != "$required" ]; then
      fail "validator accepted '$input' but extracted tag '$out', expected '$required'"
    fi
  else
    if [ "$rc" -eq 0 ]; then
      fail "validator accepted '$input', expected a refusal (${required})"
    fi
    if ! printf '%s' "$out" | grep -q "$required"; then
      fail "validator refused '$input' but without the expected named reason '$required':
$out"
    fi
  fi
}

# The one acceptable shape: this repository's own three-part release-tag
# URL, with the tag extracted as the trailing path segment.
validate_case accept "https://github.com/davison/topos/releases/tag/v1.2.3" "v1.2.3"
# Nightly-shaped tag: refused by the three-part shape guard.
validate_case refuse "https://github.com/davison/topos/releases/tag/nightly" "not a three-part stable"
# Prerelease-suffixed tag: refused by the same shape guard's anchoring.
validate_case refuse "https://github.com/davison/topos/releases/tag/v1.2.3-rc.1" "not a three-part stable"
# Off-host URL: refused by the scheme/host guard.
validate_case refuse "https://evil.example.com/davison/topos/releases/tag/v1.2.3" "scheme/host is not https://github.com"
# Plain-http URL: also the scheme/host guard.
validate_case refuse "http://github.com/davison/topos/releases/tag/v1.2.3" "scheme/host is not https://github.com"
# Different repository's release path: refused by the repo-path guard.
validate_case refuse "https://github.com/eve/malice/releases/tag/v1.2.3" "not this repository's release-tag path"
# Empty input: refused by name.
validate_case refuse "" "empty effective URL"

echo "==> Case PASS: latest-resolution validator (offline)"

# ---------------------------------------------------------------------
# Case: latest-resolution end to end (network) — runs the real resolver
# against the real release host. When the network is unreachable this
# case SKIPS LOUDLY, naming itself and the reason (the gdrive rehearsal
# spec's skip-loudly discipline) — it must never pass silently.
# ---------------------------------------------------------------------
echo "==> Case: latest-resolution end to end (network)"
if ! curl -fsSI --max-time 10 "https://github.com" >/dev/null 2>&1; then
  echo "==> Case SKIP: latest-resolution end to end (network) — https://github.com is unreachable from here; the resolver was NOT exercised against the live endpoint"
else
  LATEST_RC=0
  LATEST_TAG="$( (source ./scripts/install.sh \
    && validate_latest_url "$(resolve_latest_effective_url)") 2>&1 )" || LATEST_RC=$?
  if [ "$LATEST_RC" -ne 0 ]; then
    fail "latest-resolution end to end: resolver/validator failed against the live endpoint:
$LATEST_TAG"
  fi
  if ! printf '%s' "$LATEST_TAG" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
    fail "latest-resolution end to end: resolved tag '$LATEST_TAG' is not three-part stable semver"
  fi
  echo "    resolved latest stable release: $LATEST_TAG"
  echo "==> Case PASS: latest-resolution end to end (network)"
fi

echo "==> install-smoke: all cases passed"
