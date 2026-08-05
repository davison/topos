#!/usr/bin/env bash
# Signal read-only smoke test (SRC-02, 04-01-PLAN.md): builds the kernel
# and the (cgo) Signal plugin, runs a full sync against the developer's
# REAL, live Signal Desktop database, and proves the two things this
# phase's success criteria most directly require:
#
#   1. ~/.config/Signal/sql/db.sqlite is byte-identical before and after
#      the sync — including while Signal Desktop is running (success
#      criterion 3). Only db.sqlite itself is hashed, never its -wal/-shm
#      sidecars: SQLite's own WAL-reader protocol writes reader
#      bookkeeping into -shm even for a genuinely read-only connection
#      (04-RESEARCH.md Pitfall 2) — hashing those would produce a false
#      failure.
#   2. The synced stream contains at least one item whose source_type is
#      "signal".
#
# This script deliberately never touches your real
# ~/.config/topos/config.toml: it builds its own throwaway config
# under a temp XDG_CONFIG_HOME, with [sources.signal] pointing at your
# REAL (read-only) Signal Desktop directory, so it can prove the
# pipeline end to end without risking any change to your real
# deployment's config, webspaces, or index.
#
# SIGNAL_CONFIG_DIR overrides Signal Desktop's config directory (default
# "$HOME/.config/Signal").
#
# SIGNAL_SMOKE_KEYWORD selects which real Signal conversation or group
# name this script's throwaway webspace matches — it must exactly,
# case-insensitively equal a real conversation/group name in YOUR OWN
# Signal Desktop history (Phase 1 D-03's exact-match rule applies here
# too — mirrors scripts/e2e-smoke.sh's own precedent of a hardcoded
# "house-move"/"house and home" pairing tied to this specific
# deployment's real paperless-ngx data). Defaults to a group name
# confirmed present in this project's own developer's Signal history
# during this task's schema-introspection step; override it if you run
# this script against a different Signal account.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

SIGNAL_CONFIG_DIR="${SIGNAL_CONFIG_DIR:-$HOME/.config/Signal}"
SIGNAL_DB="$SIGNAL_CONFIG_DIR/sql/db.sqlite"
SIGNAL_SMOKE_KEYWORD="${SIGNAL_SMOKE_KEYWORD:-Davisons}"

if [ ! -f "$SIGNAL_DB" ]; then
  echo "FAIL: $SIGNAL_DB not found — is Signal Desktop installed and linked on this machine? Set SIGNAL_CONFIG_DIR to override." >&2
  exit 1
fi

echo "==> Building binaries"
mkdir -p bin/plugins
CGO_ENABLED=0 go build -o bin/topos ./cmd/topos
CGO_ENABLED=1 go build -tags libsqlcipher -o bin/plugins/topos-plugin-signal ./plugins/signal

echo "==> Hashing $SIGNAL_DB before sync"
HASH_BEFORE="$(sha256sum "$SIGNAL_DB" | awk '{print $1}')"

TMP_XDG="$(mktemp -d /tmp/topos-signal-smoke.XXXXXX)"
cleanup() {
  kill "${SERVER_PID:-}" 2>/dev/null || true
  wait "${SERVER_PID:-}" 2>/dev/null || true
  rm -rf "$TMP_XDG"
}
trap cleanup EXIT

mkdir -p "$TMP_XDG/topos"
cat > "$TMP_XDG/topos/config.toml" <<EOF
[index]
path = "$TMP_XDG/index.db"

[plugins]
dir = "$REPO_ROOT/bin/plugins"

[sources.signal]
plugin = "topos-plugin-signal"
path = "$SIGNAL_CONFIG_DIR"

[sources.signal.agent]
read = false
handoff = false

[webspaces.signal-smoke]
keywords = ["$SIGNAL_SMOKE_KEYWORD"]
EOF

export XDG_CONFIG_HOME="$TMP_XDG"

echo "==> Running topos sync (XDG_CONFIG_HOME=$TMP_XDG, isolated from your real config)"
./bin/topos sync

echo "==> Re-hashing $SIGNAL_DB after sync"
HASH_AFTER="$(sha256sum "$SIGNAL_DB" | awk '{print $1}')"
if [ "$HASH_BEFORE" != "$HASH_AFTER" ]; then
  echo "FAIL: $SIGNAL_DB changed during sync (before=$HASH_BEFORE after=$HASH_AFTER) — success criterion 3 violated" >&2
  exit 1
fi
echo "    db.sqlite hash unchanged: $HASH_AFTER"

BASE="http://127.0.0.1:7777"

echo "==> Checking 127.0.0.1:7777 is free"
if curl -fsS -o /dev/null --max-time 2 "$BASE/api/webspaces"; then
  echo "FAIL: something is already listening on 127.0.0.1:7777 — stop it before running this smoke test (it would let this script's checks pass vacuously against a stale build)" >&2
  exit 1
fi

echo "==> Starting topos serve"
./bin/topos serve &
SERVER_PID=$!

echo "==> Waiting for the server to accept connections"
for i in $(seq 1 30); do
  if curl -fsS -o /dev/null "$BASE/api/webspaces"; then
    break
  fi
  if [ "$i" -eq 30 ]; then
    echo "FAIL: server did not become ready in time" >&2
    exit 1
  fi
  sleep 1
done

echo "==> GET /api/webspaces/signal-smoke/stream"
STREAM_JSON="$(curl -fsS "$BASE/api/webspaces/signal-smoke/stream")"

echo "$STREAM_JSON" | jq -e '[.items[] | select(.source_type == "signal")] | length > 0' >/dev/null || {
  echo "FAIL: signal-smoke stream has zero items with source_type == \"signal\" (keyword \"$SIGNAL_SMOKE_KEYWORD\" may not match any real conversation/group name on this machine — override SIGNAL_SMOKE_KEYWORD)" >&2
  echo "$STREAM_JSON" >&2
  exit 1
}

# Contact-form deep-link shape guard (04-04-PLAN.md gap closure G-04-1):
# the unit tests in deeplink_test.go passed continuously while the
# emitted payload was being rejected by the real consumer (Signal
# Desktop), because they asserted the builder against itself — the
# expected value was re-derived by calling the same encoder under test,
# so a bug in the encoder was invisible to the assertion. A shape check
# taken from a real stream response, over the actual bytes a browser
# would hand to the OS scheme handler, is the layer that would have
# caught it. Deliberately no decoding or normalisation here: the
# assertion is about the raw bytes that reach Signal Desktop's route.
CONTACT_FORM_PREFIX='sgnl://signal.me/#p/'
CONTACT_LINKS="$(echo "$STREAM_JSON" | jq -r --arg prefix "$CONTACT_FORM_PREFIX" \
  '[.items[] | select(.source_type == "signal") | .link.url | select(startswith($prefix))] | .[]')"

CONTACT_LINK_COUNT=0
BAD_LINKS=""
if [ -n "$CONTACT_LINKS" ]; then
  while IFS= read -r link; do
    [ -z "$link" ] && continue
    CONTACT_LINK_COUNT=$((CONTACT_LINK_COUNT + 1))
    fragment="${link#"$CONTACT_FORM_PREFIX"}"
    first_char="${fragment:0:1}"
    if [ "$first_char" != "+" ]; then
      BAD_LINKS="$BAD_LINKS
  $link"
    fi
  done <<< "$CONTACT_LINKS"
fi

if [ -n "$BAD_LINKS" ]; then
  echo "FAIL: contact-form deep link(s) missing a literal '+' immediately after '#p/' — Signal Desktop's own validator rejects these:$BAD_LINKS" >&2
  exit 1
fi

if [ "$CONTACT_LINK_COUNT" -eq 0 ]; then
  echo "    note: zero contact-form links checked this run — SIGNAL_SMOKE_KEYWORD=\"$SIGNAL_SMOKE_KEYWORD\" matched no 1:1 conversation with a known E.164. This check was vacuous for this run; point SIGNAL_SMOKE_KEYWORD at a 1:1 conversation to exercise it. The deterministic non-vacuity guarantee for the literal-plus rule lives in plugins/signal/deeplink_test.go's unit matrix, not here."
else
  echo "    checked $CONTACT_LINK_COUNT contact-form link(s): all carry a literal '+' immediately after '#p/'"
fi

echo "==> signal-readonly-smoke test passed"
