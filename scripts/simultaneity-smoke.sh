#!/usr/bin/env bash
# The committed ISOL-03 gate: an installed-shaped instance and a
# checkout-shaped dev instance run at the same time, each answering its
# own webspace set on its own port, with neither able to see — or
# write — the other's data. This guarantee decays silently if left as
# a remembered manual check, so it lives here as `make isolation-check`.
#
# Three named cases:
#
#   1. Port contract, statically — the kernel's compiled-in production
#      listen default and the dev recipe's DEV_PORT default are read
#      from source and must differ (ISOL-02's contract, asserted
#      without binding either port).
#   2. Byte-unchanged — with the installed-shaped instance stopped, a
#      dev instance syncs and performs a real write of its own; the
#      installed tree's recursive digest manifest must be byte-identical
#      before and after (any difference could only be the dev run's).
#   3. Concurrent and independent — both instances serve at once; each
#      lists only its own webspace, and the installed tree's file SET is
#      unchanged across a driven dev sync+write (a stray write appears
#      as a new path even where legitimate activity changes bytes).
#
# Safety discipline follows scripts/dev-guard-smoke.sh: everything this
# script starts gets HOME/XDG_CONFIG_HOME/XDG_DATA_HOME pointed into
# its own mktemp work tree; every listener is an OS-assigned ephemeral
# port from smoke_free_port; and the real production port's baseline
# holder is captured at start and re-asserted after every case — a
# regression that bound the operator's real port fails as a named
# safety violation, never as a quietly green run. Safe to run while the
# operator's own installed instance is serving. No network, no
# credentials, no config file outside the work tree.
#
# The proof drives the REAL machinery, never a reimplementation: the
# fixture release installs through scripts/install.sh, and the
# dev-shaped config must pass the real cmd/topos-devguard before it is
# used — a dev fixture that could not survive the guard would prove
# nothing.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

for tool in curl sha256sum mktemp python3 go ss; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "FAIL: required tool '$tool' not found on PATH" >&2
    exit 1
  fi
done

# shellcheck source=scripts/smoke-lib.sh
source "$SCRIPT_DIR/smoke-lib.sh"

WORK="$(mktemp -d)"
INSTALLED_PID=""
DEV_PID=""

cleanup() {
  for pid in "$INSTALLED_PID" "$DEV_PID"; do
    if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
      wait "$pid" 2>/dev/null || true
    fi
  done
  rm -rf "$WORK"
}
trap cleanup EXIT

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

# Real-port safety baseline (dev-guard-smoke.sh's discipline): the pid
# (if any) already listening on the real production port at start.
# Re-asserted after every case; a DIFFERENT pid there later is a
# critical safety violation — this script must never cause anything to
# bind the operator's real port.
REAL_PORT_BASELINE_PID="$(ss -H -tlnp 'sport = :7777' 2>/dev/null | grep -o 'pid=[0-9]*' | head -1 | cut -d= -f2 || true)"

assert_no_real_port_leak() {
  local holder pid
  holder="$(ss -H -tlnp 'sport = :7777' 2>/dev/null || true)"
  pid="$(printf '%s' "$holder" | grep -o 'pid=[0-9]*' | head -1 | cut -d= -f2 || true)"
  if [ -n "$pid" ] && [ "$pid" != "$REAL_PORT_BASELINE_PID" ]; then
    echo "FAIL: SAFETY VIOLATION — something is now listening on the real production port 7777 that was not there at script start" >&2
    echo "$holder" >&2
    kill -9 "$pid" 2>/dev/null || true
    return 1
  fi
  return 0
}

# Every child process environment: all roots inside $WORK — never the
# caller's real home, config, or data directories.
HOME_DIR="$WORK/home"
XDG_C="$WORK/xdg-config"
XDG_D="$WORK/xdg-data"
mkdir -p "$HOME_DIR" "$XDG_C/topos" "$XDG_D/topos"

# with_env: for FOREGROUND helper invocations only (install.sh, the
# guard). Kernel processes are backgrounded with direct env-assignment
# prefixes instead — backgrounding a shell FUNCTION forks a subshell
# whose $! is the subshell, not the kernel, so a later kill would
# orphan the kernel still holding its port.
with_env() {
  HOME="$HOME_DIR" XDG_CONFIG_HOME="$XDG_C" XDG_DATA_HOME="$XDG_D" "$@"
}

# wait_for_source <port> <instance-name> <logfile> <pid>: polls
# GET /api/sources until <instance-name> reports a landed first sync.
wait_for_source() {
  local port="$1" name="$2" logfile="$3" pid="$4" i=0
  while :; do
    if ! kill -0 "$pid" 2>/dev/null; then
      echo "--- kernel log ($logfile) ---" >&2
      cat "$logfile" >&2 || true
      fail "kernel on port $port exited before source '$name' landed"
    fi
    local verdict
    verdict="$(curl -fsS "http://127.0.0.1:$port/api/sources" 2>/dev/null | python3 -c '
import json, sys
name = sys.argv[1]
try:
    body = json.load(sys.stdin)
except Exception:
    print("pending"); raise SystemExit
for s in body.get("sources") or []:
    if s.get("name") == name:
        if s.get("launch_failure"):
            print("launch_failure: " + s["launch_failure"]); break
        if s.get("syncing") is False and s.get("last_status"):
            print("landed"); break
        print("pending"); break
else:
    print("pending")
' "$name" || echo "pending")"
    case "$verdict" in
      landed) return 0 ;;
      launch_failure:*)
        echo "--- kernel log ($logfile) ---" >&2
        cat "$logfile" >&2 || true
        fail "source '$name' refused at launch (${verdict})"
        ;;
    esac
    i=$((i + 1))
    if [ "$i" -ge 60 ]; then
      echo "--- kernel log ($logfile) ---" >&2
      cat "$logfile" >&2 || true
      fail "source '$name' did not land within 60s"
    fi
    sleep 1
  done
}

# webspace_names <port>: prints the sorted webspace names an instance
# answers with.
webspace_names() {
  curl -fsS "http://127.0.0.1:$1/api/webspaces" | python3 -c '
import json, sys
body = json.load(sys.stdin)
names = sorted(w.get("name", "") for w in body.get("webspaces") or [])
print("\n".join(names))
'
}

# manifest_tree: recursive path+digest manifest over the installed
# instance's whole config-and-state tree (home + XDG config + XDG
# data), sorted for stable comparison.
manifest_tree() {
  (cd "$WORK" && find home xdg-config xdg-data -type f | LC_ALL=C sort | xargs -r sha256sum)
}

# fileset_tree: the sorted file SET (paths only) of the same tree.
fileset_tree() {
  (cd "$WORK" && find home xdg-config xdg-data -type f | LC_ALL=C sort)
}

# drive_dev <port>: a landed sync plus a real write of the dev
# instance's own — an item mark through the marks API (SetItemMarks is
# a genuine index write; the id needs no live item to land as a mark).
drive_dev() {
  local port="$1"
  curl -fsS -X POST "http://127.0.0.1:$port/api/sources/mock-dev/refresh" >/dev/null
  wait_for_source "$port" "mock-dev" "$WORK/dev-kernel.log" "$DEV_PID"
  local out
  out="$(curl -fsS -X POST "http://127.0.0.1:$port/api/webspaces/dev-ws/marks" \
    -H 'Content-Type: application/json' \
    -d '{"kind":"excluded","action":"add","item_ids":["isolation-smoke-item"]}')"
  if ! printf '%s' "$out" | grep -q '"changed"'; then
    fail "the dev instance's mark write did not land: $out"
  fi
}

# ---------------------------------------------------------------------
# Fixture: install the installed-shaped instance through the REAL
# install path, then build the dev-shaped checkout tree.
# ---------------------------------------------------------------------
TAG="v0.0.0-isolation-smoke"
echo "==> building fixture release ($TAG)"
smoke_build_fixture_release "$WORK" "$TAG"

PREFIX_DIR="$WORK/prefix"
with_env env PREFIX="$PREFIX_DIR" TOPOS_RELEASE_BASE_URL="file://$WORK/release" \
  ./scripts/install.sh "$TAG" >/dev/null

# The installed-shaped instance's config lives at the temp XDG config
# location and is resolved THROUGH XDG (no --config flag below), its
# index and a seeded source store at the temp XDG data location, and
# its plugins through the stock relative dir — genuinely the installed
# instance's resolution paths, all inside $WORK.
INSTALLED_PORT="$(smoke_free_port)"
mkdir -p "$XDG_D/topos/whatsapp-store"
printf 'linked-device session bytes' > "$XDG_D/topos/whatsapp-store/session.db"
cat > "$XDG_C/topos/config.toml" <<EOF
[server]
listen = "127.0.0.1:$INSTALLED_PORT"

[index]
path = "$XDG_D/topos/index.db"

[plugins]
dir = "plugins"
external_dir = "$XDG_D/topos/plugins-external"

[sync]
interval = "1h"

[sources.mock-installed]
plugin = "topos-plugin-mock"
base_url = "isolation-smoke-unused"
token = "isolation-smoke-unused"

[webspaces.installed-ws]
keywords = ["standup"]
EOF

# First boot: populate the installed instance's index, then stop it so
# case 2 manifests a settled tree.
HOME="$HOME_DIR" XDG_CONFIG_HOME="$XDG_C" XDG_DATA_HOME="$XDG_D" \
  "$PREFIX_DIR/bin/topos" serve >"$WORK/installed-kernel.log" 2>&1 &
INSTALLED_PID=$!
wait_for_source "$INSTALLED_PORT" "mock-installed" "$WORK/installed-kernel.log" "$INSTALLED_PID"
kill "$INSTALLED_PID" 2>/dev/null || true
wait "$INSTALLED_PID" 2>/dev/null || true
INSTALLED_PID=""

# The dev-shaped instance: a checkout-shaped tree, every path its own.
DEVCO="$WORK/devco"
mkdir -p "$DEVCO/bin/plugins"
cp "$WORK/topos" "$DEVCO/bin/topos"
cp bin/plugins/topos-plugin-mock "$DEVCO/bin/plugins/topos-plugin-mock"
DEV_PORT_SMOKE="$(smoke_free_port)"
cat > "$DEVCO/config.dev.toml" <<EOF
[server]
listen = "127.0.0.1:$DEV_PORT_SMOKE"

[index]
path = "$DEVCO/topos-dev-index.db"

[plugins]
dir = "$DEVCO/bin/plugins"
external_dir = "$DEVCO/bin/plugins-external-dev"

[sync]
interval = "1h"

[sources.mock-dev]
plugin = "topos-plugin-mock"
base_url = "isolation-smoke-unused"
token = "isolation-smoke-unused"

[webspaces.dev-ws]
keywords = ["standup"]
EOF

# The dev fixture must pass the REAL guard — with the guard's roots
# derived from the same $WORK-rooted environment every kernel below
# runs under.
if ! with_env go run ./cmd/topos-devguard --config "$DEVCO/config.dev.toml" >"$WORK/devguard.out" 2>&1; then
  cat "$WORK/devguard.out" >&2
  fail "the dev-shaped fixture config did not pass cmd/topos-devguard — the fixture proves nothing unless the real guard accepts it"
fi

# ---------------------------------------------------------------------
# Case 1: port contract, statically.
# ---------------------------------------------------------------------
echo "==> Case: port contract (static)"
PROD_PORT="$(sed -n 's/.*DefaultListen[[:space:]]*=[[:space:]]*"127\.0\.0\.1:\([0-9]*\)".*/\1/p' kernel/config/types.go)"
DEV_PORT_DEFAULT="$(sed -n 's/^DEV_PORT ?= \([0-9]*\)$/\1/p' Makefile)"
if [ -z "$PROD_PORT" ] || [ -z "$DEV_PORT_DEFAULT" ]; then
  fail "could not read the production listen default ('$PROD_PORT') or the Makefile DEV_PORT default ('$DEV_PORT_DEFAULT') from source"
fi
if [ "$PROD_PORT" = "$DEV_PORT_DEFAULT" ]; then
  fail "the kernel's production listen default and the Makefile's DEV_PORT default are BOTH $PROD_PORT — the ISOL-02 port contract regressed"
fi
assert_no_real_port_leak || exit 1
echo "==> Case PASS: port contract (static) — production $PROD_PORT, dev $DEV_PORT_DEFAULT"

# ---------------------------------------------------------------------
# Case 2: byte-unchanged — the installed tree across a dev run.
# ---------------------------------------------------------------------
echo "==> Case: byte-unchanged installed tree across a dev run"
manifest_tree > "$WORK/installed-manifest-before"

HOME="$HOME_DIR" XDG_CONFIG_HOME="$XDG_C" XDG_DATA_HOME="$XDG_D" \
  "$DEVCO/bin/topos" serve --config "$DEVCO/config.dev.toml" >"$WORK/dev-kernel.log" 2>&1 &
DEV_PID=$!
wait_for_source "$DEV_PORT_SMOKE" "mock-dev" "$WORK/dev-kernel.log" "$DEV_PID"
drive_dev "$DEV_PORT_SMOKE"
kill "$DEV_PID" 2>/dev/null || true
wait "$DEV_PID" 2>/dev/null || true
DEV_PID=""

manifest_tree > "$WORK/installed-manifest-after"
if ! cmp -s "$WORK/installed-manifest-before" "$WORK/installed-manifest-after"; then
  echo "FAIL: a dev run changed the installed instance's config-and-state tree:" >&2
  diff "$WORK/installed-manifest-before" "$WORK/installed-manifest-after" >&2 || true
  exit 1
fi
assert_no_real_port_leak || exit 1
echo "==> Case PASS: byte-unchanged installed tree across a dev run"

# ---------------------------------------------------------------------
# Case 3: concurrent and independent.
# ---------------------------------------------------------------------
echo "==> Case: concurrent and independent"
HOME="$HOME_DIR" XDG_CONFIG_HOME="$XDG_C" XDG_DATA_HOME="$XDG_D" \
  "$PREFIX_DIR/bin/topos" serve >"$WORK/installed-kernel.log" 2>&1 &
INSTALLED_PID=$!
HOME="$HOME_DIR" XDG_CONFIG_HOME="$XDG_C" XDG_DATA_HOME="$XDG_D" \
  "$DEVCO/bin/topos" serve --config "$DEVCO/config.dev.toml" >"$WORK/dev-kernel.log" 2>&1 &
DEV_PID=$!
wait_for_source "$INSTALLED_PORT" "mock-installed" "$WORK/installed-kernel.log" "$INSTALLED_PID"
wait_for_source "$DEV_PORT_SMOKE" "mock-dev" "$WORK/dev-kernel.log" "$DEV_PID"

INSTALLED_WS="$(webspace_names "$INSTALLED_PORT")"
DEV_WS="$(webspace_names "$DEV_PORT_SMOKE")"
if [ "$INSTALLED_WS" != "installed-ws" ]; then
  fail "the installed instance's webspace list is not exactly its own: got '$INSTALLED_WS'"
fi
if [ "$DEV_WS" != "dev-ws" ]; then
  fail "the dev instance's webspace list is not exactly its own: got '$DEV_WS'"
fi

fileset_tree > "$WORK/installed-fileset-before"
drive_dev "$DEV_PORT_SMOKE"
fileset_tree > "$WORK/installed-fileset-after"
if ! cmp -s "$WORK/installed-fileset-before" "$WORK/installed-fileset-after"; then
  echo "FAIL: the installed tree's file set changed while the dev instance was driven:" >&2
  diff "$WORK/installed-fileset-before" "$WORK/installed-fileset-after" >&2 || true
  exit 1
fi

for pid in "$INSTALLED_PID" "$DEV_PID"; do
  kill "$pid" 2>/dev/null || true
  wait "$pid" 2>/dev/null || true
done
INSTALLED_PID=""
DEV_PID=""
assert_no_real_port_leak || exit 1
echo "==> Case PASS: concurrent and independent"

echo "==> simultaneity-smoke: all cases passed"
