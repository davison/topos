---
status: testing
phase: 10-docs-and-release-readiness
source: [10-VERIFICATION.md]
started: 2026-08-12T11:35:00Z
updated: 2026-08-12T11:35:00Z
---

## Current Test

number: 1
name: Nightly workflow change-gate live proof
expected: |
  After pushing main to origin (local main is ~24 commits ahead), dispatch
  .github/workflows/nightly.yml twice via `gh workflow run nightly.yml` (or wait
  for two scheduled runs). First dispatch: sees new commits since the last
  `nightly` tag, builds, and moves the `nightly` tag with the five portable
  assets (no topos-plugin-signal). Second dispatch with no new commits: the
  change gate short-circuits — the run completes quickly without rebuilding or
  republishing assets. Closes WINDOWS.md unrun-verify entry #6.
awaiting: user response

## Tests

### 1. Nightly workflow change-gate live proof
expected: Push main to origin, dispatch nightly.yml twice: first run builds and moves the `nightly` tag with the five portable assets; second run's change gate short-circuits without rebuilding. WINDOWS.md unrun-verify entry #6 is then closed.
result: [pending]

## Summary

total: 1
passed: 0
issues: 0
pending: 1
skipped: 0
blocked: 0

## Gaps
