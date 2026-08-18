#!/usr/bin/env bash
# Removes what `make install` placed under $PREFIX (INST-05) — and ONLY
# that. The removal set is closed and explicit, mirroring
# scripts/install.sh's placement set exactly:
#
#   - the kernel binary at $PREFIX/bin/topos
#   - each FILE directly inside $PREFIX/lib/topos/plugins whose name
#     starts with "topos-plugin-"
#
# After removal, the lib/topos/plugins and lib/topos directories are
# removed with a NON-RECURSIVE rmdir only — a non-empty directory
# survives, and this script reports what remains and why rather than
# forcing it. No recursive removal appears anywhere in this script, and
# it never descends into subdirectories (T-15-08/T-15-09).
#
# The operator's configuration, kernel index, and plugin stores live in
# home-relative locations this script never names: it takes no flag and
# offers no path — not even an opt-in one — that could touch them. The
# ABSENCE of that capability is the data-safety guarantee, not a
# default that could be overridden.
#
# Idempotent: an already-absent path is reported, not an error; a
# second run is a clean no-op that exits 0. Removal is by unlink, so
# running this while the installed kernel is live succeeds and leaves
# that process running on its already-open files until it exits of its
# own accord.

set -euo pipefail

PREFIX="${PREFIX:-/usr/local}"
KERNEL_PATH="$PREFIX/bin/topos"
PLUGINS_DIR="$PREFIX/lib/topos/plugins"
TOPOS_LIB_DIR="$PREFIX/lib/topos"

REMOVED=0

# The kernel binary.
if [ -f "$KERNEL_PATH" ]; then
  rm -f "$KERNEL_PATH"
  echo "uninstall: removed $KERNEL_PATH"
  REMOVED=$((REMOVED + 1))
else
  echo "uninstall: already absent: $KERNEL_PATH"
fi

# Installed plugin binaries: files DIRECTLY inside the plugins
# directory matching the installed binary name prefix — never a
# subdirectory, never a foreign file an operator placed there.
if [ -d "$PLUGINS_DIR" ]; then
  for f in "$PLUGINS_DIR"/topos-plugin-*; do
    [ -e "$f" ] || continue
    if [ ! -f "$f" ]; then
      echo "uninstall: left in place (not a regular file): $f"
      continue
    fi
    rm -f "$f"
    echo "uninstall: removed $f"
    REMOVED=$((REMOVED + 1))
  done
else
  echo "uninstall: already absent: $PLUGINS_DIR"
fi

# Directory cleanup: non-recursive rmdir ONLY, innermost first. A
# directory that still holds anything survives, with its contents
# reported by name.
for dir in "$PLUGINS_DIR" "$TOPOS_LIB_DIR"; do
  if [ -d "$dir" ]; then
    if rmdir "$dir" 2>/dev/null; then
      echo "uninstall: removed empty directory $dir"
    else
      echo "uninstall: left in place (not empty): $dir"
      ls -A "$dir" | while IFS= read -r entry; do
        echo "uninstall:   remaining: $dir/$entry"
      done
    fi
  fi
done

if [ "$REMOVED" -eq 0 ]; then
  echo "uninstall: nothing left to remove under $PREFIX"
else
  echo "uninstall: removed $REMOVED file(s) from $PREFIX"
fi
