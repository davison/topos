#!/usr/bin/env bash
# Runs the built ./bin/webspaces binary with the repo-root .env sourced and
# EXPORTED into this process's environment, then execs the binary with
# whatever arguments follow. This exists because kernel/config.Load expands
# ${PAPERLESS_URL}-style references in config.toml from the process
# environment at load time, and that environment is inherited by every
# go-plugin subprocess the kernel spawns — a plain `. .env` (without `set -a`)
# would only assign the shell's own variables, not export them, and the
# kernel would see empty strings.
#
# Usage:
#   ./scripts/run-with-env.sh serve
#   ./scripts/run-with-env.sh sync
#
# Overridable for testing (not a general-purpose configuration surface):
#   WEBSPACES_ENV_FILE  path to the env file to source (default: $REPO_ROOT/.env)
#   WEBSPACES_BIN       path to the binary to exec (default: $REPO_ROOT/bin/webspaces)
#
# Secret hygiene: this script never echoes the value of any variable it
# sources, and every diagnostic it emits names a variable or a path, never a
# value. The env file is *sourced*, i.e. executed as bash — acceptable at
# exactly the trust level scripts/e2e-smoke.sh already assumes for the same
# file.
set -euo pipefail
# Insurance against an inherited exported SHELLOPTS enabling xtrace, which
# would print sourced variable values to stderr.
set +x

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

ENV_FILE="${WEBSPACES_ENV_FILE:-$REPO_ROOT/.env}"
BIN_PATH="${WEBSPACES_BIN:-$REPO_ROOT/bin/webspaces}"

if [ "$#" -eq 0 ]; then
  echo "usage: scripts/run-with-env.sh <webspaces-args...>" >&2
  echo "  example: scripts/run-with-env.sh serve" >&2
  exit 2
fi

if [ -f "$ENV_FILE" ]; then
  set -a
  # shellcheck disable=SC1090
  . "$ENV_FILE"
  set +a
else
  echo "note: no env file found at $ENV_FILE — continuing with the current environment" >&2
fi

# Deliberately no re-validation of which keys are present: kernel/config.Load
# already fails with an error naming the specific missing variable, and
# duplicating that policy here would drift and could wrongly reject a config
# that only uses some of the four keys.

if [ ! -x "$BIN_PATH" ]; then
  echo "FAIL: $BIN_PATH not found or not executable — run 'make build' first" >&2
  exit 1
fi

exec "$BIN_PATH" "$@"
