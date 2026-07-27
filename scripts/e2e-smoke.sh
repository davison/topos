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
CGO_ENABLED=0 go build -o bin/webspaces ./cmd/webspaces
go build -o bin/plugins/webspaces-plugin-paperless ./plugins/paperless

echo "==> Running webspaces sync"
./bin/webspaces sync

echo "==> Starting webspaces serve"
./bin/webspaces serve &
SERVER_PID=$!
trap 'kill "$SERVER_PID" 2>/dev/null || true; wait "$SERVER_PID" 2>/dev/null || true' EXIT

BASE="http://127.0.0.1:7777"

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
CODE="$(curl -sS -o /tmp/webspaces-404-body.json -w '%{http_code}' "$BASE/api/webspaces/does-not-exist/stream")"
if [ "$CODE" != "404" ]; then
  echo "FAIL: expected 404 for unknown webspace, got $CODE" >&2
  cat /tmp/webspaces-404-body.json >&2 || true
  exit 1
fi
jq -e '.error.code == "webspace_not_found"' /tmp/webspaces-404-body.json >/dev/null

echo "==> e2e smoke test passed"
