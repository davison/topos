#!/usr/bin/env bash
# Standalone assertion over one built app stylesheet. Guards two gaps:
#   G-01-2: the SPA shipping zero real CSS (empty file / no design tokens)
#   G-02-1: a named --spacing-<key> theme entry shadowing Tailwind v4's
#           default container scale, collapsing named max-w-* utilities
#           (health-chip tooltip, index page column, empty/error copy) to
#           a few pixels wide.
#
# Usage: scripts/assert-stylesheet.sh [CSS_FILE]
#   With an argument, assert against that file.
#   With no argument, resolve the built app stylesheet by globbing
#   kernel/webui/build/_app/immutable/assets/*.css and picking the first
#   match containing the #020617 design token.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

CSS_FILE="${1:-}"

if [ -z "$CSS_FILE" ]; then
  shopt -s nullglob
  CANDIDATES=("$REPO_ROOT"/kernel/webui/build/_app/immutable/assets/*.css)
  shopt -u nullglob
  if [ "${#CANDIDATES[@]}" -eq 0 ]; then
    echo "FAIL: no built stylesheet found under kernel/webui/build/_app/immutable/assets/*.css — run 'npm --prefix web run build' or 'make build' first" >&2
    exit 1
  fi
  for candidate in "${CANDIDATES[@]}"; do
    if grep -qF '#020617' "$candidate" 2>/dev/null; then
      CSS_FILE="$candidate"
      break
    fi
  done
  if [ -z "$CSS_FILE" ]; then
    echo "FAIL: no candidate stylesheet under kernel/webui/build/_app/immutable/assets/*.css contains the #020617 design token — the SPA has not been built (or was built from stale/placeholder CSS)" >&2
    exit 1
  fi
fi

# 1. File exists and is non-empty.
if [ ! -s "$CSS_FILE" ]; then
  echo "FAIL: stylesheet '$CSS_FILE' does not exist or is empty" >&2
  exit 1
fi

# 2. Contains the #020617 design token (G-01-2 guard — carried over from
#    e2e-smoke.sh's prior inline check; must not be weakened).
if ! grep -qF '#020617' "$CSS_FILE"; then
  echo "FAIL: stylesheet '$CSS_FILE' does not contain the app's #020617 design token — it may be an empty or placeholder CSS file (G-01-2)" >&2
  exit 1
fi

# 3. Named container widths resolve correctly. For each key actually used
#    in web/src, the file must satisfy at least one of two forms: the
#    --container-<key> custom property set to the expected rem value, or
#    the .max-w-<key> rule carrying that rem value inline.
check_container_width() {
  local key="$1" rem="$2"
  if grep -qE -- "--container-${key}: *${rem}rem" "$CSS_FILE"; then
    return 0
  fi
  if grep -qE -- "\.max-w-${key}\{max-width: *${rem}rem\}" "$CSS_FILE"; then
    return 0
  fi
  echo "FAIL: stylesheet '$CSS_FILE' does not resolve max-w-${key} to ${rem}rem via --container-${key} or an inlined rule — the container scale was not consulted at all (G-02-1)" >&2
  return 1
}

check_container_width "xs" "20"
check_container_width "md" "28"
check_container_width "3xl" "48"

# 4. No named max-width utility resolves to a raw pixel value. Match rules
#    whose selector is .max-w- followed by one of the container-scale key
#    shapes (xs, sm, md, lg, xl, or a digit followed by xl) and whose
#    declared max-width is an integer count of pixels. Tailwind's own
#    max-w-px utility (key "px") is legitimately one pixel and must not
#    trip this check — its key shape is deliberately excluded below.
COLLAPSED_RULE="$(grep -oE '\.max-w-(xs|sm|md|lg|xl|[0-9]+xl)\{max-width:[0-9]+px\}' "$CSS_FILE" || true)"
if [ -n "$COLLAPSED_RULE" ]; then
  echo "FAIL: stylesheet '$CSS_FILE' resolves a named max-width utility to a raw pixel value: $COLLAPSED_RULE — this is gap G-02-1's exact mechanism (a theme entry shadowing the container scale)" >&2
  exit 1
fi

echo "OK: stylesheet '$CSS_FILE' passes G-01-2 and G-02-1 guards"
