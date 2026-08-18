---
quick_id: 260818-ov4
status: complete
date: 2026-08-18
commits: [fbf58c4]
---

# Quick Task Summary: harden CI e2e step against apt mirror hangs

Two consecutive CI runs (both cancelled manually at 17m) hung inside
Playwright's `--with-deps` apt phase: the runner's default
`azure.archive.ubuntu.com` mirror failed repeatedly and the fallback fetch
stalled with no timeout anywhere to reap it. Same code path was green at
11:24 the same day; nothing in the intervening commits touched CI, web
deps, or apt — environmental, not content.

## Changes (.github/workflows/ci.yml)

- New named step "Harden apt against mirror stalls" (5-min cap) before
  `make e2e`: repoints apt to canonical `archive.ubuntu.com` in both
  sources formats, sets `Acquire::Retries 3` + 30s connection timeouts,
  pre-warms indices.
- `make e2e` capped at 10 minutes; job capped at 30 — a silent multi-hour
  hang is now structurally impossible; a sick mirror fails fast in a named
  step.

## Verification

Run 32162833220 on the hardened workflow: **success in 4m27s** — apt step
passed, full e2e suite passed (incl. the shutdown-reap fix's hardened
test). YAML validated locally before push.
