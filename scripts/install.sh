#!/usr/bin/env bash
# Installs a published topos release into $PREFIX (INST-01): the kernel
# at $PREFIX/bin/topos and every published plugin binary at
# $PREFIX/lib/topos/plugins/, each SHA-256-verified against that
# release's own checksums.txt BEFORE anything is written to $PREFIX.
#
# Usage: install.sh <version>       (with or without a leading "v")
#
# Environment:
#   PREFIX                  install root      (default /usr/local)
#   TOPOS_RELEASE_BASE_URL  release base URL  (default the GitHub
#                           releases root). This is a test seam exactly
#                           like the Makefile's DEV_KERNEL_CMD: it
#                           changes WHICH release is fetched (e.g. a
#                           file:// fixture tree in
#                           scripts/install-smoke.sh), never which
#                           checks run.
#
# Sequence: preflight -> stage -> verify -> place. Placement does not
# begin until every asset has verified, so an abort at any earlier point
# leaves $PREFIX byte-unchanged. Each placed file is copied to a
# temporary name inside its own destination directory and completed with
# an atomic mv -f rename — never written directly over the destination —
# so a re-run over a running kernel/plugin process replaces the file
# without a text-file-busy failure and an interrupted run leaves every
# destination either wholly old bytes or wholly new bytes.
#
# This script never escalates privileges: an unwritable $PREFIX fails
# loud, naming the directory and the `sudo make install` re-run for the
# operator to perform deliberately.

set -euo pipefail

fail() {
  echo "install: FAIL: $*" >&2
  exit 1
}

VERSION_ARG="${1:-}"
if [ -z "$VERSION_ARG" ]; then
  fail "a release version is required — usage: install.sh <version> (e.g. 1.1.0 or v1.1.0)"
fi
# Normalise to a single tag form: accept "1.1.0" or "v1.1.0", use "v1.1.0".
TAG="v${VERSION_ARG#v}"

PREFIX="${PREFIX:-/usr/local}"
TOPOS_RELEASE_BASE_URL="${TOPOS_RELEASE_BASE_URL:-https://github.com/davison/topos/releases}"

BIN_DIR="$PREFIX/bin"
PLUGINS_DIR="$PREFIX/lib/topos/plugins"

# --- preflight --------------------------------------------------------
for tool in curl sha256sum mktemp; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    fail "required tool '$tool' not found on PATH"
  fi
done

# Probe destination writability BEFORE any download work is thrown away.
# mkdir -p is a no-op on an existing directory and never escalates; a
# failure here (or a directory we cannot write) names the directory and
# the escalated re-run form — this script itself must never run sudo.
for dir in "$BIN_DIR" "$PLUGINS_DIR"; do
  if ! mkdir -p "$dir" 2>/dev/null || [ ! -w "$dir" ]; then
    fail "cannot write to $dir — re-run as: sudo make install (this script never escalates privileges itself)"
  fi
done

# --- stage ------------------------------------------------------------
STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT

DOWNLOAD_BASE="$TOPOS_RELEASE_BASE_URL/download/$TAG"

# -f makes an HTTP error a non-zero curl exit, not a saved error page.
if ! curl -fsSL "$DOWNLOAD_BASE/checksums.txt" -o "$STAGE/checksums.txt"; then
  fail "could not fetch checksums.txt for release $TAG from $DOWNLOAD_BASE"
fi

# Derive the asset list from checksums.txt's second column — that file
# IS the manifest (it is what release.yml's own one-place-only ASSETS
# list produced); a second hardcoded list here is exactly the drift that
# comment guards against. Every derived relative path must match an
# allowlist shape: exactly "topos", or "plugins/" followed by a plugin
# binary name of lowercase letters, digits and hyphens. Anything
# absolute, anything with a parent-directory segment, anything else at
# all is rejected by name — the manifest's paths are untrusted text that
# becomes local write paths (T-15-02).
ASSETS=()
while IFS= read -r line; do
  [ -n "$line" ] || continue
  # sha256sum output: "<64 hex>  <path>" (two-space separator; a binary
  # -mode marker would make it " *<path>", which the allowlist rejects).
  rel="${line#*  }"
  if [ "$rel" = "$line" ] || [ -z "$rel" ]; then
    fail "checksums.txt line is not a sha256sum entry: $line"
  fi
  case "$rel" in
    topos)
      ;;
    plugins/*)
      name="${rel#plugins/}"
      if ! printf '%s' "$name" | grep -Eq '^[a-z0-9-]+$'; then
        fail "checksums.txt names a disallowed path (rejected): $line"
      fi
      ;;
    *)
      fail "checksums.txt names a disallowed path (rejected): $line"
      ;;
  esac
  ASSETS+=("$rel")
done < "$STAGE/checksums.txt"

if [ "${#ASSETS[@]}" -eq 0 ]; then
  fail "checksums.txt for $TAG lists no assets"
fi

# Download each asset under its published flat basename into the staging
# directory at its full checksums.txt-relative path — release assets are
# uploaded as flat basenames while checksums.txt records the
# "plugins/<name>" relative layout, so the staging tree must reproduce
# that layout for `sha256sum -c` to run unmodified.
mkdir -p "$STAGE/plugins"
for rel in "${ASSETS[@]}"; do
  base="${rel##*/}"
  if ! curl -fsSL "$DOWNLOAD_BASE/$base" -o "$STAGE/$rel"; then
    fail "could not download asset '$base' for release $TAG from $DOWNLOAD_BASE — nothing was written to $PREFIX"
  fi
done

# --- verify -----------------------------------------------------------
# Every asset must verify before ANY placement begins — an abort here
# leaves $PREFIX byte-unchanged.
if ! (cd "$STAGE" && sha256sum -c checksums.txt >"$STAGE/verify.out" 2>&1); then
  failed="$(grep -v ': OK$' "$STAGE/verify.out" 2>/dev/null || true)"
  fail "SHA-256 verification failed for release $TAG — nothing was written to $PREFIX
${failed}"
fi

# --- place ------------------------------------------------------------
# Copy to a temporary name INSIDE the destination directory, set mode,
# then complete with an atomic same-directory rename. Never write
# directly over the destination path: the rename is what makes a re-run
# over a live kernel/plugin process safe (T-15-04) and what makes an
# interrupted run leave whole-old or whole-new bytes, never a torn file.
WRITTEN=()
for rel in "${ASSETS[@]}"; do
  case "$rel" in
    topos) dest="$BIN_DIR/topos" ;;
    plugins/*) dest="$PLUGINS_DIR/${rel#plugins/}" ;;
  esac
  destdir="$(dirname "$dest")"
  tmp="$(mktemp "$destdir/.topos-install.XXXXXX")" \
    || fail "cannot create a temporary file in $destdir — re-run as: sudo make install (this script never escalates privileges itself)"
  cp "$STAGE/$rel" "$tmp"
  chmod 0755 "$tmp"
  mv -f "$tmp" "$dest"
  WRITTEN+=("$dest")
done

# --- report -----------------------------------------------------------
echo "install: release $TAG installed into $PREFIX"
for path in "${WRITTEN[@]}"; do
  echo "install:   wrote $path"
done
