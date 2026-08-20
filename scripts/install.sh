#!/usr/bin/env bash
# Installs a published topos release into $PREFIX (INST-01): the kernel
# at $PREFIX/bin/topos and every published plugin binary at
# $PREFIX/lib/topos/plugins/, each SHA-256-verified against that
# release's own checksums.txt BEFORE anything is written to $PREFIX.
#
# Usage: install.sh [version]       (with or without a leading "v")
#
# With no version argument, the latest published STABLE release is
# resolved (INST-02) by following the release root's `latest` redirect
# and validating the effective URL it lands on: scheme and host must be
# exactly https://github.com, the path must be this repository's own
# release-tag path, and the trailing tag must be three-part
# v<major>.<minor>.<patch> semver — which structurally excludes the
# moving `nightly` prerelease tag and any prerelease-suffixed tag, as a
# second guard the script enforces itself rather than trusting the
# endpoint's prerelease-exclusion semantics (T-15-06/T-15-07). No
# credential, token, or GitHub CLI is involved.
#
# This file is source-guarded: sourcing it defines its functions and
# runs nothing — the seam scripts/install-smoke.sh uses to drive the
# validator offline. Only direct execution runs the install flow.
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

# resolve_latest_effective_url: issues a redirect-following HEAD request
# against the release root's `latest` path and prints the final
# effective URL. Deliberately decides NOTHING about whether that answer
# is acceptable — validate_latest_url below is the sole authority on
# that, so the network step and the safety check stay independently
# testable.
resolve_latest_effective_url() {
  curl -fsSLI -o /dev/null -w '%{url_effective}' \
    "${TOPOS_RELEASE_BASE_URL:-https://github.com/davison/topos/releases}/latest"
}

# validate_latest_url <effective-url>: validates the URL the `latest`
# redirect landed on and prints the release tag it names. Refuses by
# name (T-15-06/T-15-07) when: the input is empty; the scheme/host are
# not exactly https://github.com; the path is not this repository's own
# release-tag path; or the trailing segment is not a bare three-part
# v<digits>.<digits>.<digits> tag. The tag-shape check is a second,
# independent prerelease guard: the `latest` endpoint already excludes
# prereleases by definition, and this makes that exclusion something
# this script enforces rather than trusts — the moving `nightly` tag
# and any `-rc`/`-beta` suffixed tag both fail the shape.
validate_latest_url() {
  local url="${1:-}"
  if [ -z "$url" ]; then
    fail "latest-release resolution returned an empty effective URL"
  fi
  case "$url" in
    https://github.com/*) ;;
    *)
      fail "latest-release URL refused: scheme/host is not https://github.com — got: $url"
      ;;
  esac
  local path="${url#https://github.com}"
  case "$path" in
    /davison/topos/releases/tag/*) ;;
    *)
      fail "latest-release URL refused: not this repository's release-tag path (/davison/topos/releases/tag/...) — got: $url"
      ;;
  esac
  local tag="${path#/davison/topos/releases/tag/}"
  if ! printf '%s' "$tag" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
    fail "latest-release URL refused: tag '$tag' is not a three-part stable v<major>.<minor>.<patch> release (prerelease/nightly tags are never auto-selected)"
  fi
  printf '%s' "$tag"
}

main() {

VERSION_ARG="${1:-}"
if [ -z "$VERSION_ARG" ]; then
  # No version given: resolve the latest published stable release
  # (INST-02) — resolve, then validate, then continue into the exact
  # tag-based flow an explicit version takes.
  EFFECTIVE_URL="$(resolve_latest_effective_url)" \
    || fail "could not resolve the latest release from ${TOPOS_RELEASE_BASE_URL:-https://github.com/davison/topos/releases}/latest — pass an explicit version (make install VERSION=<tag>) if the network is unavailable"
  TAG="$(validate_latest_url "$EFFECTIVE_URL")"
  echo "install: resolved latest stable release: $TAG"
else
  # Normalise to a single tag form: accept "1.1.0" or "v1.1.0", use "v1.1.0".
  TAG="v${VERSION_ARG#v}"
fi

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
# allowlist shape: exactly "topos", exactly "topos-provenance" (16-05-
# PLAN.md Task 1: a release MAY optionally publish the provenance
# verifier CLI itself as an asset — see the provenance-verification step
# below), "plugins/" followed by a plugin binary name of lowercase
# letters, digits and hyphens, or "plugins/" followed by a provenance
# file shape (D-05/D-07: *.provenance.json / *.provenance.sig, whose
# basenames combine a release repo and tag and so may contain dots the
# plain-binary-name shape below forbids — validated instead as "no path
# separators, no leading dot" so a provenance file cannot itself smuggle
# a traversal path, T-15-02's own discipline, extended). Anything
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
    topos-provenance)
      ;;
    plugins/*)
      name="${rel#plugins/}"
      case "$name" in
        *.provenance.json | *.provenance.sig)
          case "$name" in
            */* | .*)
              fail "checksums.txt names a disallowed path (rejected): $line"
              ;;
          esac
          ;;
        *)
          if ! printf '%s' "$name" | grep -Eq '^[a-z0-9-]+$'; then
            fail "checksums.txt names a disallowed path (rejected): $line"
          fi
          ;;
      esac
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

# A staged topos-provenance CLI needs its execute bit set BEFORE it can be
# invoked by the provenance-verification step below (curl -o never sets
# it) — placement's own chmod 0755 happens later and is a separate copy.
if [ -f "$STAGE/topos-provenance" ]; then
  chmod +x "$STAGE/topos-provenance"
fi

# --- verify -----------------------------------------------------------
# Every asset must verify before ANY placement begins — an abort here
# leaves $PREFIX byte-unchanged.
if ! (cd "$STAGE" && sha256sum -c checksums.txt >"$STAGE/verify.out" 2>&1); then
  failed="$(grep -v ': OK$' "$STAGE/verify.out" 2>/dev/null || true)"
  fail "SHA-256 verification failed for release $TAG — nothing was written to $PREFIX
${failed}"
fi

# --- verify: provenance (D-09's install-time arm, 16-05-PLAN.md Task 1) -
# SHA-256 (above) proves transport integrity only — "these are the bytes
# checksums.txt named" — never publisher authenticity. A staged payload
# that ALSO carries at least one *.provenance.json runs the second,
# independent check here, still before any placement: the same
# `topos-provenance verify` the kernel's own launch gate calls
# (kernel/pluginhost.VerifySignedProvenance) is invoked over the staged
# plugins directory, and a failure aborts naming the binary — nothing has
# been placed yet either way. A payload with NO provenance files at all
# (every release this repository has published to date, and any future
# one that simply doesn't opt in) makes this a documented no-op: the
# install proceeds exactly as it always has (TRUST-03).
#
# Verifier resolution order, tried in sequence — a payload that ships
# evidence must have that evidence checked, so failing to resolve ANY
# verifier is a loud abort here, never a silent skip.
#
# 16-REVIEW.md WR-01: provenance verification exists to catch an
# attacker who can tamper with release artifacts AND regenerate
# checksums.txt to match (i.e. someone who controls the release
# publishing pipeline but does not hold the ed25519 private signing
# key). Under exactly that threat model, such an attacker can ALSO
# publish a topos-provenance binary in the same tampered release that
# unconditionally reports success. The staged payload's own copy is
# therefore the LEAST trustworthy of the three candidates — it ships
# from the same pipeline as the bytes it would be verifying — so it is
# now tried LAST, only when no more-trustworthy verifier exists at all:
#   1. one already installed at $BIN_DIR/topos-provenance (a prior
#      install's own copy — untouched by whatever produced THIS
#      release's payload);
#   2. one already on PATH (operator-controlled, resolved independently
#      of this release's own staged bytes);
#   3. a topos-provenance binary present in the STAGED payload itself
#      (a release may legitimately publish one alongside its plugins,
#      and on a machine with no prior install and nothing on PATH this
#      is the only option — see docs/install.md's "Provenance
#      verification" section for the bootstrap-trust caveat this
#      still leaves: a fully-compromised release payload supplying its
#      own verifier is a residual risk this ordering narrows, not
#      eliminates).
PROVENANCE_FILES=()
if [ -d "$STAGE/plugins" ]; then
  for f in "$STAGE"/plugins/*.provenance.json; do
    [ -e "$f" ] || continue
    PROVENANCE_FILES+=("$f")
  done
fi

if [ "${#PROVENANCE_FILES[@]}" -gt 0 ]; then
  PROVENANCE_VERIFIER=""
  if [ -x "$BIN_DIR/topos-provenance" ]; then
    PROVENANCE_VERIFIER="$BIN_DIR/topos-provenance"
  elif command -v topos-provenance >/dev/null 2>&1; then
    PROVENANCE_VERIFIER="$(command -v topos-provenance)"
  elif [ -x "$STAGE/topos-provenance" ]; then
    PROVENANCE_VERIFIER="$STAGE/topos-provenance"
  fi

  if [ -z "$PROVENANCE_VERIFIER" ]; then
    fail "release $TAG carries ${#PROVENANCE_FILES[@]} signed provenance manifest(s) but no topos-provenance verifier could be resolved (checked the staged payload, $BIN_DIR/topos-provenance, and PATH) — a payload that ships evidence must have that evidence checked; nothing was written to $PREFIX"
  fi

  if ! "$PROVENANCE_VERIFIER" verify --dir "$STAGE/plugins" >"$STAGE/provenance-verify.out" 2>&1; then
    fail "provenance verification failed for release $TAG — nothing was written to $PREFIX
$(cat "$STAGE/provenance-verify.out")"
  fi
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
    topos-provenance) dest="$BIN_DIR/topos-provenance" ;;
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

}

# Source-guard: executed directly, run the install flow; sourced (the
# install-smoke test seam), define functions only and do nothing else.
if [ "${BASH_SOURCE[0]}" = "$0" ]; then
  main "$@"
fi
