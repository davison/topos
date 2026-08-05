#!/usr/bin/env bash
# End-to-end smoke test: builds both binaries, runs a real sync against the
# user's live paperless-ngx instance, starts the server, and asserts the
# house-move webspace stream is populated with correctly-shaped items.
#
# Credentials (PAPERLESS_URL, PAPERLESS_TOKEN) are expected either already
# in the environment, or in a .env file at the repo root (KEY=value lines) —
# this script sources ./.env itself so it works from a fresh shell.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

if [ -f "$REPO_ROOT/.env" ]; then
  set -a
  # shellcheck disable=SC1091
  . "$REPO_ROOT/.env"
  set +a
fi

: "${PAPERLESS_URL:?PAPERLESS_URL must be set (env or .env)}"
: "${PAPERLESS_TOKEN:?PAPERLESS_TOKEN must be set (env or .env)}"

echo "==> Building binaries"
mkdir -p bin/plugins
CGO_ENABLED=0 go build -o bin/topos ./cmd/topos
go build -o bin/plugins/topos-plugin-paperless ./plugins/paperless

echo "==> Running topos sync"
./bin/topos sync

BASE="http://127.0.0.1:7777"

echo "==> Checking 127.0.0.1:7777 is free"
if curl -fsS -o /dev/null --max-time 2 "$BASE/api/webspaces"; then
  echo "FAIL: something is already listening on 127.0.0.1:7777 — stop it before running this smoke test (it would let this script's checks pass vacuously against a stale build)" >&2
  exit 1
fi

echo "==> Starting topos serve"
./bin/topos serve &
SERVER_PID=$!
trap 'kill "$SERVER_PID" 2>/dev/null || true; wait "$SERVER_PID" 2>/dev/null || true' EXIT

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

echo "==> Checking the served SPA links a real stylesheet"
SPA_HTML="$(curl -fsS "$BASE/")"
CSS_HREF="$(printf '%s' "$SPA_HTML" | grep -o 'rel="stylesheet"[^>]*href="[^"]*\.css"\|href="[^"]*\.css"[^>]*rel="stylesheet"' | grep -o 'href="[^"]*\.css"' | head -1 | cut -d'"' -f2)"
if [ -z "$CSS_HREF" ]; then
  echo "FAIL: served SPA HTML links no stylesheet — the production build may have shipped zero CSS (see gap G-01-2)" >&2
  exit 1
fi
# adapter-static emits relative asset hrefs (./_app/...); normalise to an
# absolute path before prefixing $BASE.
CSS_PATH="${CSS_HREF#.}"
case "$CSS_PATH" in
  /*) ;;
  *) CSS_PATH="/$CSS_PATH" ;;
esac
CSS_TMP="$(mktemp /tmp/topos-smoke-css.XXXXXX)"
trap 'kill "$SERVER_PID" 2>/dev/null || true; wait "$SERVER_PID" 2>/dev/null || true; rm -f "$CSS_TMP"' EXIT
curl -fsS "$BASE$CSS_PATH" -o "$CSS_TMP" || {
  echo "FAIL: stylesheet linked at $CSS_HREF did not fetch successfully" >&2
  exit 1
}
"$SCRIPT_DIR/assert-stylesheet.sh" "$CSS_TMP"

echo "==> GET /api/webspaces"
curl -fsS "$BASE/api/webspaces" | jq -e '.webspaces | length >= 0' >/dev/null

echo "==> GET /api/webspaces/house-move/stream"
STREAM_JSON="$(curl -fsS "$BASE/api/webspaces/house-move/stream")"

echo "$STREAM_JSON" | jq -e '.items | length > 0' >/dev/null || {
  echo "FAIL: house-move stream has zero items" >&2
  echo "$STREAM_JSON" >&2
  exit 1
}

echo "$STREAM_JSON" | jq -e '[.items[] | select(.link.fidelity != "exact")] | length == 0' >/dev/null || {
  echo "FAIL: not every item has link.fidelity == \"exact\"" >&2
  exit 1
}

echo "$STREAM_JSON" | jq -e --arg prefix "$PAPERLESS_URL/documents/" \
  '[.items[] | select(.link.url | startswith($prefix) | not)] | length == 0' >/dev/null || {
  echo "FAIL: not every item's link.url starts with \$PAPERLESS_URL/documents/" >&2
  exit 1
}

echo "==> GET /api/webspaces/does-not-exist/stream returns 404"
# Deliberately no -f here: curl -f suppresses writing the response body to
# -o on an HTTP error status, and this check needs the 404 error envelope
# body, not just the status code.
CODE="$(curl -sS -o /tmp/topos-404-body.json -w '%{http_code}' "$BASE/api/webspaces/does-not-exist/stream")"
if [ "$CODE" != "404" ]; then
  echo "FAIL: expected 404 for unknown webspace, got $CODE" >&2
  cat /tmp/topos-404-body.json >&2 || true
  exit 1
fi
jq -e '.error.code == "webspace_not_found"' /tmp/topos-404-body.json >/dev/null

echo "==> e2e smoke test passed"
