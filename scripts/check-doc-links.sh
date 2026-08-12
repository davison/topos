#!/usr/bin/env bash
# Repo-wide relative-markdown-link resolution guard.
#
# What it checks: every markdown link target and inline image target in
# this repository's maintained documentation set — the repo-root
# markdown files, everything under docs/, web/, and plugins/ — resolves
# to a file or directory that actually exists. Documentation link rot is
# invisible until a reader hits it, and this phase (10) built a web of
# cross-references between root docs, per-plugin pages, and the release
# process page; a broken one of those should fail a push, not wait to be
# noticed by a reader.
#
# What it deliberately does NOT check: external HTTP(S) URLs. Verifying
# those needs network access, and a network-dependent gate is a flaky
# gate — this script only resolves paths inside the repository itself.
#
# The one deliberate exception: a path under docs/ss/ that matches the
# indexed screenshot naming convention (docs/ss/README.md) is expected
# NOT to exist until the operator drops a real screenshot in — the
# README embeds docs/ss/1.png through docs/ss/4.png before any of those
# files are committed, by design. A missing indexed screenshot is
# therefore accepted; any OTHER missing target under docs/ss/ still
# fails, same as anywhere else. Don't mistake this exception for a bug —
# it exists because docs/ss/README.md documents exactly this state.
#
# Deliberately excluded from the scan: .planning/ (phase-by-phase
# planning artifacts reference files across phase boundaries and archive
# states — not a maintained public doc surface, and scanning it would
# make this gate noisy rather than honest) and node_modules (vendored,
# not ours to fix).
#
# Usage:
#   scripts/check-doc-links.sh
#   make docs-check

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

# Enumerate the tracked markdown files this repository actually
# maintains: repo-root *.md, plus everything under docs/, web/, and
# plugins/ — explicitly excluding .planning/ and node_modules (belt and
# braces: the find below never descends into either directory tree, on
# top of never targeting .planning/ or plugins/*/node_modules at all).
mapfile -t DOC_FILES < <(
  {
    find "$REPO_ROOT" -maxdepth 1 -name '*.md' -type f
    find "$REPO_ROOT/docs" -name '*.md' -type f 2>/dev/null
    find "$REPO_ROOT/web" -name '*.md' -type f -not -path '*/node_modules/*' 2>/dev/null
    find "$REPO_ROOT/plugins" -name '*.md' -type f -not -path '*/node_modules/*' 2>/dev/null
  } | sort -u
)

FAILURES=0
LINKS_CHECKED=0
FILES_CHECKED=0

# is_screenshot_exception PATH — true if PATH is under docs/ss/ and its
# basename matches the indexed screenshot convention (digits + .png),
# per docs/ss/README.md.
is_screenshot_exception() {
  local rel="$1"
  case "$rel" in
    docs/ss/*.png)
      local base
      base="$(basename "$rel")"
      [[ "$base" =~ ^[0-9]+\.png$ ]]
      ;;
    *)
      return 1
      ;;
  esac
}

for doc in "${DOC_FILES[@]}"; do
  FILES_CHECKED=$((FILES_CHECKED + 1))
  doc_dir="$(dirname "$doc")"
  rel_doc="${doc#"$REPO_ROOT"/}"

  # Extract markdown link/image targets: [text](target) and
  # ![alt](target). grep -n gives us the line number for reporting;
  # the sed strips everything up to the first "](" and the trailing ")".
  while IFS=: read -r line_no raw_target; do
    [ -z "$raw_target" ] && continue
    LINKS_CHECKED=$((LINKS_CHECKED + 1))

    target="$raw_target"

    # Skip external URLs and mail links — not this script's job (see
    # header comment).
    case "$target" in
      http://*|https://*|mailto:*)
        continue
        ;;
    esac

    # Skip pure-fragment anchors (e.g. "#section") — same-document,
    # nothing to resolve on disk.
    case "$target" in
      '#'*)
        continue
        ;;
    esac

    # Strip a trailing fragment (e.g. "path.md#section" -> "path.md").
    target="${target%%#*}"
    [ -z "$target" ] && continue

    # Resolve relative to the containing file's own directory.
    case "$target" in
      /*)
        resolved="$target"
        ;;
      *)
        resolved="$doc_dir/$target"
        ;;
    esac

    if [ ! -e "$resolved" ]; then
      rel_target="${resolved#"$REPO_ROOT"/}"
      if is_screenshot_exception "$rel_target"; then
        continue
      fi
      echo "BROKEN LINK: $rel_doc:$line_no: $raw_target"
      FAILURES=$((FAILURES + 1))
    fi
  done < <(grep -noE '!?\[[^]]*\]\([^)]+\)' "$doc" | sed -E 's/^([0-9]+):.*\(([^)]+)\)$/\1:\2/')
done

if [ "$FAILURES" -gt 0 ]; then
  echo "check-doc-links: $FAILURES broken link(s) found across $FILES_CHECKED file(s)" >&2
  exit 1
fi

echo "check-doc-links: checked $LINKS_CHECKED link(s) across $FILES_CHECKED file(s) — all resolve"
