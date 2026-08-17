---
created: 2026-08-17T11:26:20.726Z
title: Abstract OAuth connectivity and secrets management into the kernel for all plugins
area: kernel
severity: major
files:
  - docs/plugin-contract.md
  - kernel/pluginhost/
---

## Problem

The Phase 14 clean-room build of `topos-plugin-gdrive` (sibling repo,
`~/projects/davison/topos-plugin-gdrive`) had to create substantial in-plugin
machinery to manage OAuth connectivity and secrets/token storage — none of it
Google-specific in shape. Most future source plugins for cloud services
(calendar, cloud notes, other document stores) will need the same: an OAuth
authorization flow, token refresh, and safe local persistence of credentials.

Leaving this per-plugin means every third-party plugin author rebuilds it,
inconsistently and less safely — and secrets handling is exactly where
inconsistency hurts most on a privacy-first, local-only product. The clean-room
exercise already flagged the underlying contract gaps before any plugin code was
written: GAP-01 (token storage) and GAP-02 (sync-state cache) in the plugin
repo's `CONTRACT-GAPS.md`.

The user is confident enough, based on the gdrive build, to treat this as a
**requirement** (kernel-provided OAuth + secrets services, exposed through the
published plugin contract so plugins can leverage them) — not just an idea.

## Solution

Promote to a formal requirement and roadmap item rather than fixing ad hoc:

- Plan 14-05 already triages the plugin repo's `CONTRACT-GAPS.md` back into the
  published contract (`docs/plugin-contract.md`) — the wire-level surface for
  token storage/secrets should be decided there or filed by it.
- The kernel-side implementation (OAuth flow helper, token refresh, secret
  persistence — plausibly via the same OS secret-store integration used for the
  Signal key) is larger than a gap-triage note and likely warrants its own
  phase/requirement in the next milestone, with the gdrive plugin as the first
  consumer to migrate.
- When drafting the requirement, mine the gdrive plugin's actual OAuth/secrets
  code for the abstraction boundary — it is the working reference implementation.
