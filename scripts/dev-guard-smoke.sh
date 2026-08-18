#!/usr/bin/env bash
# Hermetic behavioural guard for the `make dev` recipe (quick task
# 260805-o5d, hardening it against two footguns that burned a real
# debugging session: silent plugin staleness and a silent port-race
# loss that leaves Vite proxying to a stale kernel).
#
# Proves three cases against the REAL `dev` recipe, using the
# Makefile's DEV_PORT/DEV_HOST/DEV_READY_TIMEOUT/DEV_KERNEL_CMD/
# DEV_UI_CMD test-seam variables so no real network port, real kernel,
# or real Vite dev server is required for cases 2 and 3, and case 1's
# squatter is a port this script itself selects:
#
#   1. squatter on the dev port -> `make dev` exits non-zero, names the
#      host:port and the holder's pid, leaves no process behind, and
#      the plugin binaries are still rebuilt (the `plugins` prerequisite
#      runs before the recipe body regardless of the guard's verdict)
#   2. kernel dies for any other reason -> `make dev` exits non-zero
#      without ever starting the UI dev server
#   3. healthy kernel -> `make dev` still reaches the UI launch and
#      tears both processes down cleanly afterwards
#
# Safe to run alongside a real kernel already listening on
# 127.0.0.1:7777: every port this script binds (the squatter in case 1,
# the fake kernel in case 3) is an ephemeral port it selects itself via
# free_port() below — it never binds, probe-connects, or requires
# 127.0.0.1:7777 to be free.
#
# This script triggers the `plugins` Makefile prerequisite (case 1, and
# the regression path generally), so it needs the system sqlcipher
# library the `signal` target requires — the same prerequisite
# `make build`/`make dev` already carry.
#
# The error-path cases (1, 2) are asserted via the recorded exit status
# of `make dev`, captured by a background runner script. The happy path
# (case 3) is asserted via a marker file instead, because on that path
# the recipe's own `trap 'kill 0' EXIT INT TERM` intentionally also
# kills `make` itself before any exit status can ever be recorded — see
# the Makefile's own comment above the `dev` target for why that's kept.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

for tool in python3 setsid ss ps; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "FAIL: required tool '$tool' not found on PATH" >&2
    exit 1
  fi
done

WORK="$(mktemp -d)"
SQUATTER_PID=""
CURRENT_PGID=""

# REAL_PORT_BASELINE_PID: the pid (if any) already listening on the real
# production dev port (127.0.0.1:7777, hardcoded — deliberately NOT
# $(DEV_PORT), since the whole point of this check is to catch the
# recipe under test binding the real port regardless of what override
# it was given or ignored) at script start. assert_no_real_port_leak,
# called after every case, treats any DIFFERENT pid later bound there as
# a critical safety violation — this script must never cause something
# to bind the developer's real dev port — and kills it by the pid `ss`
# itself reports (explicit pid, not name/pattern) before failing loudly.
REAL_PORT_BASELINE_PID="$(ss -H -tlnp 'sport = :7777' 2>/dev/null | grep -o 'pid=[0-9]*' | head -1 | cut -d= -f2 || true)"

assert_no_real_port_leak() {
  local holder pid
  holder="$(ss -H -tlnp 'sport = :7777' 2>/dev/null || true)"
  pid="$(printf '%s' "$holder" | grep -o 'pid=[0-9]*' | head -1 | cut -d= -f2 || true)"
  if [ -n "$pid" ] && [ "$pid" != "$REAL_PORT_BASELINE_PID" ]; then
    echo "FAIL: SAFETY VIOLATION — something is now listening on the real 127.0.0.1:7777 that was not there at script start" >&2
    echo "$holder" >&2
    kill -9 "$pid" 2>/dev/null || true
    return 1
  fi
  return 0
}

cleanup() {
  if [ -n "$SQUATTER_PID" ] && kill -0 "$SQUATTER_PID" 2>/dev/null; then
    kill "$SQUATTER_PID" 2>/dev/null || true
    wait "$SQUATTER_PID" 2>/dev/null || true
  fi
  if [ -n "$CURRENT_PGID" ]; then
    kill -- "-$CURRENT_PGID" 2>/dev/null || true
  fi
  rm -rf "$WORK"
}
trap cleanup EXIT

# mtime reference for the case-1 plugin-rebuild assertion — touched
# once, up front, before anything deletes or rebuilds the mock plugin.
touch "$WORK/marker"

free_port() {
  python3 -c '
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
'
}

# hold.sh <port> <duration>: binds 127.0.0.1:<port> and holds it for
# <duration> seconds. Used both as the case-1 squatter and as the
# case-3 fake kernel (a fake kernel just needs to bind the dev port so
# the readiness gate's `ss` check can see it listening).
cat > "$WORK/hold.sh" <<'HOLD'
#!/usr/bin/env bash
exec python3 -c '
import socket, sys, time
port = int(sys.argv[1])
duration = int(sys.argv[2])
s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
s.bind(("127.0.0.1", port))
s.listen(1)
time.sleep(duration)
' "$1" "$2"
HOLD
chmod +x "$WORK/hold.sh"

# ui.sh: the case-3 fake UI dev server — touches a marker proving the
# recipe reached the UI launch, then exits shortly after.
cat > "$WORK/ui.sh" <<UI
#!/usr/bin/env bash
touch "$WORK/ui-started"
sleep 1
UI
chmod +x "$WORK/ui.sh"

# run.sh <work-dir> <make args...>: POSIX sh. Writes its own \$\$ to
# <work-dir>/pgid — after setsid this PID is the new process-group ID,
# so the group can be killed deterministically regardless of whether
# setsid forked. Runs `make` with the given arguments, redirecting both
# streams to <work-dir>/log, then records make's exit status to
# <work-dir>/status — a line that is only ever reached on the error
# paths (see header). Two environment variables are set for every
# invocation, on every case, including the pre-guard RED run where the
# real dev recipe body still ignores every override below:
#   - BROWSER=none suppresses Vite's --open browser launch.
#   - XDG_CONFIG_HOME points at a directory this script controls that
#     never contains a topos/config.toml. If the real
#     `go run ./cmd/topos serve` command is ever actually invoked
#     (only possible pre-guard, since every guarded case replaces it
#     with a fake), config.Load fails immediately — before any plugin
#     subprocess launches, before any real sync starts, before any bind
#     attempt — so a RED run against the real recipe can never touch a
#     real Signal/Proton/SilverBullet source or strand a sync_runs row
#     in the user's real index database.
cat > "$WORK/run.sh" <<RUN
#!/bin/sh
WORK_DIR="\$1"
shift
echo "\$\$" > "\$WORK_DIR/pgid"
BROWSER=none XDG_CONFIG_HOME="$WORK/no-such-config" XDG_DATA_HOME="$WORK/no-such-data" make "\$@" >"\$WORK_DIR/log" 2>&1
echo "\$?" > "\$WORK_DIR/status"
RUN
chmod +x "$WORK/run.sh"

# gen_dev_config <port> <path>: writes an isolated per-case dev config
# whose every path sits under $WORK — the topos-devguard pre-flight the
# `dev` recipe now runs (15-04) must see a config that is both isolated
# and port-consistent before any case can reach the guard it is
# actually exercising. XDG_DATA_HOME above points the guard's state
# root into $WORK too, so these configs are judged against roots this
# script owns, never the developer's real ones.
gen_dev_config() {
  local port="$1" path="$2"
  cat > "$path" <<CFG
[server]
listen = "127.0.0.1:$port"

[index]
path = "$WORK/dev-smoke-index.db"

[plugins]
dir = "$WORK/dev-smoke-plugins"
external_dir = "$WORK/dev-smoke-plugins-external"
CFG
}

# run_case <budget-seconds> <make args...>: launches run.sh in a fresh
# session/process group via setsid, waits for it to record its pgid,
# then polls once a second up to <budget-seconds> for EITHER a status
# file to appear (error paths) OR the process group to become empty
# (happy path, where kill 0 tears everything down including make
# itself before status can be written). Returns 0 the moment either is
# observed. On budget exhaustion it group-kills, reports the hang, and
# returns 1 — this is the assertion that catches today's unguarded
# recipe blocking in `wait` forever.
run_case() {
  local budget="$1"
  shift
  rm -f "$WORK/pgid" "$WORK/status" "$WORK/log"
  setsid sh "$WORK/run.sh" "$WORK" "$@" &
  local waited=0
  while [ ! -f "$WORK/pgid" ]; do
    sleep 0.1
    waited=$((waited + 1))
    if [ "$waited" -gt 100 ]; then
      echo "FAIL: run.sh never recorded its process group (pgid file never appeared)" >&2
      return 1
    fi
  done
  PGID="$(cat "$WORK/pgid")"
  CURRENT_PGID="$PGID"
  local i=0
  while [ "$i" -lt "$budget" ]; do
    if [ -f "$WORK/status" ]; then
      return 0
    fi
    if [ -z "$(ps -o pid= -g "$PGID" 2>/dev/null)" ]; then
      return 0
    fi
    sleep 1
    i=$((i + 1))
  done
  echo "FAIL: make dev neither exited nor was torn down within ${budget}s — this is the hang this guard exists to catch" >&2
  kill -- "-$PGID" 2>/dev/null || true
  return 1
}

# assert_drained: allows up to 15s for the last case's process group to
# drain, then asserts it is empty. Prints the surviving processes and
# group-kills before failing if it is not.
assert_drained() {
  local pgid="$1"
  local i=0
  while [ "$i" -lt 15 ]; do
    if [ -z "$(ps -o pid= -g "$pgid" 2>/dev/null)" ]; then
      return 0
    fi
    sleep 1
    i=$((i + 1))
  done
  echo "FAIL: process group $pgid did not drain within 15s of the case finishing" >&2
  ps -g "$pgid" >&2 || true
  kill -- "-$pgid" 2>/dev/null || true
  return 1
}

# ---------------------------------------------------------------------
# Case 1: squatter holds the dev port — the reported footgun.
# ---------------------------------------------------------------------
echo "==> Case 1: squatter on the dev port"
P1="$(free_port)"
"$WORK/hold.sh" "$P1" 300 &
SQUATTER_PID=$!
sleep 0.5
rm -f bin/plugins/topos-plugin-mock

gen_dev_config "$P1" "$WORK/devcfg-case1.toml"
run_case 300 dev "DEV_PORT=$P1" "DEV_CONFIG=$WORK/devcfg-case1.toml"
RC1=$?
# Runs unconditionally, on both the pass AND fail branches of run_case:
# the RED failure path (recipe hangs, run_case times out) is exactly the
# branch where the real-port leak this checks for was observed to occur.
assert_no_real_port_leak || exit 1
if [ "$RC1" -ne 0 ]; then
  echo "FAIL: case 1 (squatter) failed" >&2
  exit 1
fi

STATUS1="$(cat "$WORK/status" 2>/dev/null || echo "")"
if [ -z "$STATUS1" ] || [ "$STATUS1" = "0" ]; then
  echo "FAIL: case 1 expected a non-zero recorded exit status, got '${STATUS1:-<none>}'" >&2
  cat "$WORK/log" >&2 || true
  exit 1
fi
if ! grep -q "127.0.0.1:$P1" "$WORK/log"; then
  echo "FAIL: case 1 output did not name 127.0.0.1:$P1" >&2
  cat "$WORK/log" >&2
  exit 1
fi
if ! grep -q "pid=" "$WORK/log"; then
  echo "FAIL: case 1 output did not contain a pid= token naming the squatting process" >&2
  cat "$WORK/log" >&2
  exit 1
fi
assert_drained "$PGID"
CURRENT_PGID=""

kill "$SQUATTER_PID" 2>/dev/null || true
wait "$SQUATTER_PID" 2>/dev/null || true
SQUATTER_PID=""

if [ ! -x bin/plugins/topos-plugin-mock ]; then
  echo "FAIL: case 1 expected bin/plugins/topos-plugin-mock to have been rebuilt by the plugins prerequisite" >&2
  exit 1
fi
if [ "bin/plugins/topos-plugin-mock" -ot "$WORK/marker" ]; then
  echo "FAIL: case 1 mock plugin binary is not newer than the run's start marker — it was not rebuilt" >&2
  exit 1
fi
echo "==> Case 1 PASS"

# ---------------------------------------------------------------------
# Case 2: the kernel dies during startup for any other reason.
# ---------------------------------------------------------------------
echo "==> Case 2: kernel dies for another reason"
P2="$(free_port)"

gen_dev_config "$P2" "$WORK/devcfg-case2.toml"
run_case 60 dev "DEV_PORT=$P2" "DEV_CONFIG=$WORK/devcfg-case2.toml" "DEV_READY_TIMEOUT=3" "DEV_KERNEL_CMD=false"
RC2=$?
assert_no_real_port_leak || exit 1
if [ "$RC2" -ne 0 ]; then
  echo "FAIL: case 2 (kernel dies) failed" >&2
  exit 1
fi

STATUS2="$(cat "$WORK/status" 2>/dev/null || echo "")"
if [ -z "$STATUS2" ] || [ "$STATUS2" = "0" ]; then
  echo "FAIL: case 2 expected a non-zero recorded exit status, got '${STATUS2:-<none>}'" >&2
  cat "$WORK/log" >&2 || true
  exit 1
fi
if ! grep -q "127.0.0.1:$P2" "$WORK/log"; then
  echo "FAIL: case 2 output did not name 127.0.0.1:$P2" >&2
  cat "$WORK/log" >&2
  exit 1
fi
assert_drained "$PGID"
CURRENT_PGID=""
echo "==> Case 2 PASS"

# ---------------------------------------------------------------------
# Case 3: the working path is unregressed.
# ---------------------------------------------------------------------
echo "==> Case 3: happy path still reaches the UI launch"
P3="$(free_port)"
rm -f "$WORK/ui-started"

gen_dev_config "$P3" "$WORK/devcfg-case3.toml"
run_case 60 dev "DEV_PORT=$P3" "DEV_CONFIG=$WORK/devcfg-case3.toml" "DEV_READY_TIMEOUT=15" "DEV_KERNEL_CMD=sh $WORK/hold.sh $P3 4" "DEV_UI_CMD=sh $WORK/ui.sh"
RC3=$?
assert_no_real_port_leak || exit 1
if [ "$RC3" -ne 0 ]; then
  echo "FAIL: case 3 (happy path) failed" >&2
  exit 1
fi

if [ ! -f "$WORK/ui-started" ]; then
  echo "FAIL: case 3 expected the UI launch marker to appear — the readiness gate never opened" >&2
  cat "$WORK/log" >&2 || true
  exit 1
fi
assert_drained "$PGID"
CURRENT_PGID=""
echo "==> Case 3 PASS"

echo "==> dev-guard-smoke: all three cases passed"
