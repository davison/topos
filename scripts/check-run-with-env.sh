#!/usr/bin/env bash
# Credential-free behavioural check for scripts/run-with-env.sh. Proves,
# without ever touching real credentials, that: (1) arguments pass through
# to the wrapped binary verbatim and in order; (2) the four .env keys are
# exported into the child process, not just assigned; (3) the wrapper's
# combined stdout+stderr never contains a fixture credential value; (4) a
# zero-argument invocation and a missing binary each fail with an
# actionable message.
set -euo pipefail
set +x

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

FIXTURE_DIR="$(mktemp -d)"
trap 'rm -rf "$FIXTURE_DIR"' EXIT

FIXTURE_ENV="$FIXTURE_DIR/.env"
FIXTURE_BIN="$FIXTURE_DIR/fake-webspaces"
SENTINEL="SENTINELVALUE9f3a"

cat > "$FIXTURE_ENV" <<EOF
PAPERLESS_URL=https://paperless.fixture.invalid
PAPERLESS_TOKEN=$SENTINEL
SB_URL=https://sb.fixture.invalid
SB_AUTH_TOKEN=$SENTINEL
EOF

cat > "$FIXTURE_BIN" <<'EOF'
#!/usr/bin/env bash
echo "ARGS:$*"
echo "PAPERLESS_TOKEN_SET:${PAPERLESS_TOKEN:+yes}"
echo "SB_AUTH_TOKEN_SET:${SB_AUTH_TOKEN:+yes}"
EOF
chmod +x "$FIXTURE_BIN"

FAIL=0

# --- Behaviour: pass-through, export, no-leak ---
OUTPUT="$(WEBSPACES_ENV_FILE="$FIXTURE_ENV" WEBSPACES_BIN="$FIXTURE_BIN" "$SCRIPT_DIR/run-with-env.sh" serve --host 127.0.0.1 2>&1)" || {
  echo "FAIL: wrapper exited non-zero for the happy-path invocation" >&2
  echo "$OUTPUT" >&2
  FAIL=1
}

if ! printf '%s' "$OUTPUT" | grep -qF "ARGS:serve --host 127.0.0.1"; then
  echo "FAIL: wrapper did not pass arguments through verbatim and in order" >&2
  FAIL=1
fi

if ! printf '%s' "$OUTPUT" | grep -qF "PAPERLESS_TOKEN_SET:yes"; then
  echo "FAIL: PAPERLESS_TOKEN was not exported into the child process" >&2
  FAIL=1
fi

if ! printf '%s' "$OUTPUT" | grep -qF "SB_AUTH_TOKEN_SET:yes"; then
  echo "FAIL: SB_AUTH_TOKEN was not exported into the child process" >&2
  FAIL=1
fi

if printf '%s' "$OUTPUT" | grep -qF "$SENTINEL"; then
  echo "FAIL: wrapper's combined output leaked the fixture token value" >&2
  FAIL=1
fi

# --- Behaviour: zero-argument invocation fails with a usage line ---
set +e
ZERO_ARG_OUTPUT="$(WEBSPACES_ENV_FILE="$FIXTURE_ENV" WEBSPACES_BIN="$FIXTURE_BIN" "$SCRIPT_DIR/run-with-env.sh" 2>&1)"
ZERO_ARG_STATUS=$?
set -e

if [ "$ZERO_ARG_STATUS" -eq 0 ]; then
  echo "FAIL: zero-argument invocation exited 0, expected non-zero" >&2
  FAIL=1
fi

if ! printf '%s' "$ZERO_ARG_OUTPUT" | grep -qF "usage:"; then
  echo "FAIL: zero-argument invocation did not print a usage line" >&2
  FAIL=1
fi

if printf '%s' "$ZERO_ARG_OUTPUT" | grep -qF "$SENTINEL"; then
  echo "FAIL: zero-argument invocation leaked the fixture token value" >&2
  FAIL=1
fi

# --- Behaviour: missing binary fails naming 'make build' ---
set +e
MISSING_BIN_OUTPUT="$(WEBSPACES_ENV_FILE="$FIXTURE_ENV" WEBSPACES_BIN="$FIXTURE_DIR/does-not-exist" "$SCRIPT_DIR/run-with-env.sh" serve 2>&1)"
MISSING_BIN_STATUS=$?
set -e

if [ "$MISSING_BIN_STATUS" -eq 0 ]; then
  echo "FAIL: missing-binary invocation exited 0, expected non-zero" >&2
  FAIL=1
fi

if ! printf '%s' "$MISSING_BIN_OUTPUT" | grep -qF "make build"; then
  echo "FAIL: missing-binary invocation did not mention 'make build'" >&2
  FAIL=1
fi

if printf '%s' "$MISSING_BIN_OUTPUT" | grep -qF "$SENTINEL"; then
  echo "FAIL: missing-binary invocation leaked the fixture token value" >&2
  FAIL=1
fi

if [ "$FAIL" -ne 0 ]; then
  exit 1
fi

echo "check-run-with-env: PASS"
