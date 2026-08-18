#!/usr/bin/env bash
# Places (or removes, with --uninstall) the locally built Signal plugin
# binary for an INSTALLED topos instance (INST-04).
#
# Usage: install-signal.sh [--uninstall]
#
# The build itself is NOT here: `make install-signal` reaches the
# repository's single `signal` build definition as a prerequisite (the
# one place the cgo/libsqlcipher build flags live — duplicating them
# here is exactly the drift the Makefile's one-place-only comments
# guard against). This script only resolves the destination, places
# atomically, and prints the one-time steps.
#
# Destination: the installed instance's EXTERNAL plugin directory —
# deliberately NOT $PREFIX/lib/topos/plugins. A locally built binary is
# not in the released kernel's link-time build manifest, and a
# trusted-directory binary that fails manifest verification is refused
# before any subprocess is created (kernel/pluginhost.VerifyTrustedBinary,
# surfacing as launch_failure "manifest_unverified"). That gate is the
# trust system working correctly; the supported path — recorded in
# docs/plugins/signal.md — is the external directory plus the app's
# one-time untrusted-add consent-and-pin flow, which this script
# automates the placement half of and prints the rest of.
#
# The default destination reproduces the kernel's own Linux
# external-directory default (defaultExternalPluginsDir in
# cmd/topos/main.go): $XDG_DATA_HOME/topos/plugins-external when
# XDG_DATA_HOME is set and non-empty, else
# $HOME/.local/share/topos/plugins-external. An operator whose config
# names a different [plugins] external_dir points the installer at it
# via TOPOS_EXTERNAL_PLUGINS_DIR.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

BINARY_NAME="topos-plugin-signal"
BUILT_BINARY="$REPO_ROOT/bin/plugins/$BINARY_NAME"

fail() {
  echo "install-signal: FAIL: $*" >&2
  exit 1
}

# --- destination resolution ------------------------------------------
if [ -n "${TOPOS_EXTERNAL_PLUGINS_DIR:-}" ]; then
  DEST_DIR="$TOPOS_EXTERNAL_PLUGINS_DIR"
  DEST_SOURCE="TOPOS_EXTERNAL_PLUGINS_DIR override"
elif [ -n "${XDG_DATA_HOME:-}" ]; then
  DEST_DIR="$XDG_DATA_HOME/topos/plugins-external"
  DEST_SOURCE="kernel default (XDG_DATA_HOME)"
else
  DEST_DIR="$HOME/.local/share/topos/plugins-external"
  DEST_SOURCE="kernel default (~/.local/share)"
fi

# Footgun refusal: a trusted plugins directory under a prefix is never a
# valid destination for a locally built binary — the released kernel's
# build manifest cannot contain its hash, so the kernel would refuse it
# at launch (manifest_unverified) before any subprocess is created.
case "$DEST_DIR" in
  */lib/topos/plugins | */lib/topos/plugins/)
    fail "refusing destination '$DEST_DIR': that is an installed instance's TRUSTED plugins directory, and a locally built binary placed there is refused by the kernel's link-time build-manifest verification (launch_failure: manifest_unverified). The supported destination is the EXTERNAL plugin directory ([plugins] external_dir) — the kernel's default is \$XDG_DATA_HOME/topos/plugins-external (or ~/.local/share/topos/plugins-external)."
    ;;
esac

DEST="$DEST_DIR/$BINARY_NAME"

# --- uninstall mode ---------------------------------------------------
if [ "${1:-}" = "--uninstall" ]; then
  echo "install-signal: external plugin directory: $DEST_DIR ($DEST_SOURCE)"
  if [ -f "$DEST" ]; then
    rm -f "$DEST"
    echo "install-signal: removed $DEST"
  else
    echo "install-signal: already absent: $DEST — nothing to remove"
  fi
  # The directory itself and every other binary in it are untouched:
  # this removal path reaches exactly one file, ever.
  exit 0
fi

# --- install mode -----------------------------------------------------
if [ ! -f "$BUILT_BINARY" ]; then
  fail "built binary not found at $BUILT_BINARY — run this via 'make install-signal' (which builds it through the repository's own 'signal' target first; requires the system sqlcipher package, see docs/plugins/signal.md)"
fi

echo "install-signal: external plugin directory: $DEST_DIR ($DEST_SOURCE)"

mkdir -p "$DEST_DIR" || fail "cannot create $DEST_DIR"

# Atomic placement, same technique as scripts/install.sh: temporary name
# inside the destination directory, mode, then rename — safe to re-run
# while the installed kernel is serving (the live subprocess keeps its
# already-open file; the new bytes surface as a pin mismatch to
# re-accept).
tmp="$(mktemp "$DEST_DIR/.topos-install-signal.XXXXXX")" \
  || fail "cannot create a temporary file in $DEST_DIR"
cp "$BUILT_BINARY" "$tmp"
chmod 0755 "$tmp"
mv -f "$tmp" "$DEST"

echo "install-signal: placed $DEST"
echo ""
echo "install-signal: one-time steps (docs/plugins/signal.md 'The fix, step by step'):"
echo "install-signal:   This binary is untrusted by construction — it was not built beside"
echo "install-signal:   the kernel that will run it, so the kernel's link-time build manifest"
echo "install-signal:   cannot vouch for it."
echo "install-signal:   1. Restart (or start) your installed kernel."
echo "install-signal:   2. Add the Signal source through the app's untrusted-add consent flow —"
echo "install-signal:      the same explicit consent-and-pin path any external binary goes through."
echo "install-signal:   3. It then runs pinned and badged untrusted."
echo "install-signal:   Re-running 'make install-signal' later produces new bytes: the changed"
echo "install-signal:   binary must be re-accepted through the chip's re-pin flow."
echo "install-signal:   If your config's [plugins] external_dir names a different directory,"
echo "install-signal:   re-run with TOPOS_EXTERNAL_PLUGINS_DIR=<that directory>."
