#!/usr/bin/env bash
#
# scripts/sync-milestones.sh — reconciles ONE GitHub repository milestone
# with .planning/'s milestone state (success criterion 5, 10-04-PLAN.md).
#
# Direction of authority is one-way and must stay that way: .planning/ is
# the source of truth for milestone state; GitHub milestones are a mirror
# kept in step by this script, never the reverse. A milestone edited
# directly in the GitHub UI will be silently overwritten the next time
# this script runs — that is intentional, not a bug.
#
# When to run it: manually, at milestone boundaries — paired with
# /gsd-complete-milestone (close the outgoing milestone) and
# /gsd-new-milestone (open the incoming one). Deliberately NOT run on a
# schedule and deliberately NOT wired as a GitHub Action reacting to
# .planning/ commits: milestone boundaries are infrequent, human-triggered
# events, not continuous state that needs a standing sync job.
#
# Arguments (both required, positional):
#   $1  milestone title    — e.g. "v1.0", read from .planning/STATE.md's
#                             frontmatter `milestone` key by the operator
#                             before invoking this script
#   $2  action              — exactly "open" or "close"
#
# Environment:
#   TOPOS_REPO  overrides the target repository, default "davison/topos".
#               The resolved repository and the intended operation are
#               always echoed before any mutation is attempted, so a
#               mis-targeted run is visible in the output rather than
#               discovered afterwards.
#
# Safety guarantees:
#   - Idempotent: safe to re-run. The milestone is looked up by exact
#     title across all states BEFORE deciding whether to create or patch
#     it — this lookup-before-write is the idempotency mechanism and is
#     not optional. An unconditional create against an already-existing
#     title returns a duplicate-title validation error from the GitHub
#     API.
#   - No delete path: this script cannot delete a milestone. The
#     capability is absent, not merely unused — .planning/ never deletes
#     a milestone either (only opens or closes one), and a delete would
#     orphan every issue assigned to it.
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

usage() {
  echo "Usage: $0 <milestone-title> <open|close>" >&2
  echo "  received: title='${TITLE:-}' action='${ACTION:-}'" >&2
}

TITLE="${1:-}"
ACTION="${2:-}"

if [ -z "$TITLE" ]; then
  echo "FAIL: missing milestone title" >&2
  usage
  exit 1
fi

case "$ACTION" in
  open|close) ;;
  *)
    echo "FAIL: action must be exactly 'open' or 'close', received: '${ACTION}'" >&2
    usage
    exit 1
    ;;
esac

REPO="${TOPOS_REPO:-davison/topos}"

# The GitHub milestones API accepts state values "open"/"closed", not the
# action vocabulary "open"/"close" this script takes as $2 — map it here.
# (The two coincide for "open", which is why only the close path ever 422s.)
case "$ACTION" in
  close) STATE="closed" ;;
  *)     STATE="$ACTION" ;;
esac

echo "Repository: ${REPO}"
echo "Operation: set milestone '${TITLE}' to state '${STATE}'"

# Lookup-before-write: the idempotency mechanism. Looks across ALL
# milestone states (open and closed) so re-running against an
# already-closed milestone still finds it rather than creating a
# duplicate.
existing_number=$(gh api "repos/${REPO}/milestones?state=all" \
  --jq ".[] | select(.title==\"${TITLE}\") | .number" | head -1)

if [ -z "$existing_number" ]; then
  echo "No milestone titled '${TITLE}' found — creating it with state '${ACTION}'..."
  number=$(gh api --method POST "repos/${REPO}/milestones" \
    -f title="${TITLE}" -f state="${STATE}" --jq '.number')
  echo "Created milestone '${TITLE}' as #${number}, state=${STATE}"
else
  echo "Milestone '${TITLE}' already exists as #${existing_number} — reconciling state to '${STATE}'..."
  gh api --method PATCH "repos/${REPO}/milestones/${existing_number}" \
    -f state="${STATE}" >/dev/null
  echo "Reconciled milestone '${TITLE}' (#${existing_number}), state=${STATE}"
fi
