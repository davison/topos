#!/usr/bin/env bash
# Removes the KERNEL'S OWN artifacts from $PREFIX — and ONLY those. The
# removal set is closed and explicit:
#
#   - the kernel binary at $PREFIX/bin/topos
#   - the provenance verifier at $PREFIX/bin/topos-provenance (shipped
#     by kernel releases from v1.3.0; reported as already absent under
#     an older install)
#
# The plugin fleet and its provenance pairs under
# $PREFIX/lib/topos/plugins are deliberately NOT touched (M1-R5,
# davison/topos#15): that directory is owned by topos-plugins' own
# `make install`/`make uninstall`, whichever installer placed its
# contents — including fleets placed by pre-split kernel releases
# (v1.1.0/v1.2.0), which that repository's uninstall removes all the
# same. This script prints an informational note when it leaves a
# non-empty fleet behind, and otherwise names nothing under lib/topos
# except the empty-directory tidy-up below.
#
# After removal, the lib/topos/plugins and lib/topos directories are
# removed with a NON-RECURSIVE rmdir only when they are left empty — a
# non-empty directory survives untouched. No recursive removal appears
# anywhere in this script, and it never descends into subdirectories.
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
PLUGINS_DIR="$PREFIX/lib/topos/plugins"
TOPOS_LIB_DIR="$PREFIX/lib/topos"

REMOVED=0

for f in "$PREFIX/bin/topos" "$PREFIX/bin/topos-provenance"; do
  if [ -f "$f" ]; then
    rm -f "$f"
    echo "uninstall: removed $f"
    REMOVED=$((REMOVED + 1))
  else
    echo "uninstall: already absent: $f"
  fi
done

if [ -d "$PLUGINS_DIR" ] && [ -n "$(ls -A "$PLUGINS_DIR")" ]; then
  echo "uninstall: left in place: $PLUGINS_DIR — the plugin fleet is topos-plugins' own make uninstall to remove"
fi

# Empty-directory tidy-up: non-recursive rmdir ONLY, innermost first —
# removes what the install's writability probe created when nothing
# lives there; anything occupied survives (reported above).
for dir in "$PLUGINS_DIR" "$TOPOS_LIB_DIR"; do
  if [ -d "$dir" ] && rmdir "$dir" 2>/dev/null; then
    echo "uninstall: removed empty directory $dir"
  fi
done

if [ "$REMOVED" -eq 0 ]; then
  echo "uninstall: nothing left to remove under $PREFIX"
else
  echo "uninstall: removed $REMOVED file(s) from $PREFIX"
fi
