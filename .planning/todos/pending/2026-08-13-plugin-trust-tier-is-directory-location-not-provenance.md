---
created: 2026-08-13T18:36:29.949Z
title: Plugin trust tier is directory-location, not provenance
area: security
severity: major
files:
  - kernel/config/types.go:32-66
  - kernel/pluginhost/discover_binaries.go:169-247
  - kernel/pluginhost/discover_binaries.go:354-397
  - cmd/topos/main.go:85
---

## Problem

The original intent for the plugin trust boundary was **repo provenance**: plugins
built from the davison/topos repo are trusted; anything else is not. What Phase 11
shipped is a **filesystem-location proxy** for that intent — `kernel/config/types.go:38`
states it outright: "trust is derived purely from WHICH directory a binary resolved
from, never from anything the binary declares about itself."

- **Trusted tier** = anything in `plugins.dir` (default `plugins/` next to the
  executable, user-writable, config-overridable). No hash pin, no badge, no consent
  flow — deliberately, because `make build`/`make dev` rebuilds these binaries
  constantly and pinning would false-alarm every rebuild (types.go:57-59).
- **External tier** = anything in `external_dir` — SHA-256 pinned at add-time,
  re-verified at every launch, badged untrusted, consent-gated.

Three paths bypass the boundary (mistake or social engineering):

1. **Config edit**: pointing `plugins.dir` at any other directory (in `config.toml`
   or via the hot-apply `PUT /api/config`) silently promotes everything there to
   trusted — no pin, no badge, no confirm.
2. **File drop**: copying any third-party binary into the trusted dir makes it
   trusted instantly.
3. **Shadowing (D-11)**: a same-named binary dropped in the trusted dir shadows a
   pinned external plugin, and trusted-tier binaries launch unpinned — so the pin
   stops applying. Surfaced only as a kernel log line, never in the UI.

Severity framing: the tier system is a consent/provenance layer, not a privilege
boundary (external plugins are unsandboxed subprocesses with full user privileges),
but these paths bypass the consent flow — which is the boundary's entire value.

## Solution

Candidate direction: derive trusted-tier status from **build provenance**, not
location alone — e.g. embed a manifest of first-party plugin identities/hashes into
the kernel at build time (kernel and plugins are rebuilt together by the same
`make build`, so the dev-rebuild false-alarm argument against pinning the trusted
dir does not apply to a link-time-embedded hash set). Anything unverifiable demotes
to external-tier semantics (badge + pin + consent), regardless of directory.

Weaker alternative: make `plugins.dir` non-hot-applyable / confirm-gated so the
API path can't silently retarget the trusted tier — but that does not cover file
drops into the directory, so it is a partial fix at best.

Also: surface the D-11 trusted-shadows-external event in the UI (health state or
badge), not just a log line.

Routing note: cross-cutting security debt from Phase 11 — likely its own hardening
phase or backlog item. Touches Phase 12 only insofar as criterion 5 rehearses the
external path; Phase 12 planning does not depend on resolving this first.
