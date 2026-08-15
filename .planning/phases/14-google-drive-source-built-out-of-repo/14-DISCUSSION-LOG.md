# Phase 14: Google Drive Source, Built Out-of-Repo - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-15
**Phase:** 14-Google Drive Source, Built Out-of-Repo
**Areas discussed:** Todo folding, OAuth flow & token home, Out-of-repo dev model

---

## Todo Folding (cross-reference step)

| Option | Description | Selected |
|--------|-------------|----------|
| Real vs dev config split | Kernel `--config` flag / `TOPOS_CONFIG` + Makefile dev convention; Phase 14 UAT runs dev kernels against real config with live OAuth creds | ✓ |
| Tooltip suppression | SourceChip title/alt removal; small web polish ride-along | ✓ |
| Signal schema tooling | Signal plugin maintenance tooling, unrelated (4th keyword-noise match) | |

**User's choice:** Fold config split + tooltip suppression.
**Notes:** The trust-tier todo (2026-08-13) was excluded from the options as stale — delivered by Phase 13 D-12..D-16; archived at this commit.

---

## Gray Area Selection

Offered: OAuth flow & token home / Out-of-repo dev model / Drive scope & matching / Previews & export shapes.
**User's choice:** Discuss the first two; Drive scope & matching and Previews & export shapes left to Claude's discretion (grounded in Phase 12 precedents).

---

## OAuth flow & token home

**Q1: How should the one-time Google authorization happen?**

| Option | Description | Selected |
|--------|-------------|----------|
| CLI auth command (Recommended) | `topos-plugin-gdrive auth` run once in a terminal; browser + loopback; named health state points to it | ✓ |
| Auth URL via health surface | Plugin health carries live auth URL; conflicts with "UI text is kernel-composed, never plugin text" (deliberate contract gap) | |
| Kernel-mediated flow | Kernel proxies OAuth; kernel changes undermine the no-kernel-change proof | |

**Q2: Where should the plugin persist the OAuth refresh token?**

| Option | Description | Selected |
|--------|-------------|----------|
| Keyring (Secret Service) | go-keyring via D-Bus; scrubbed env may not pass DBUS_SESSION_BUS_ADDRESS | |
| Plugin-owned file (Recommended) | Token JSON 0600 under ~/.local/share/topos-plugin-gdrive/; plugin-state home undefined by contract → documented gap | ✓ |
| Try keyring, fall back to file | Two persistence paths to test | |

**Q3: How to handle the OAuth client's publishing status (testing = 7-day refresh-token expiry)?**

| Option | Description | Selected |
|--------|-------------|----------|
| Docs mandate production status (Recommended) | Setup guide walks through publishing the personal app to production; tokens live until revoked | ✓ |
| Accept testing status | Weekly re-auth chore surfaced as named health state | |
| Document both paths | Production recommended, testing as known limitation | |

**Q4: Where does the standalone auth command get the OAuth client ID/secret?**

| Option | Description | Selected |
|--------|-------------|----------|
| Same env vars, your shell (Recommended) | CLI reads the exact env var names the source config references | ✓ |
| CLI flags | Secret in shell history; names drift | |
| Env vars with flag override | More surface to document/test | |

---

## Out-of-repo dev model

**Q1: Where does the plugin live?**

| Option | Description | Selected |
|--------|-------------|----------|
| Public GitHub repo (Recommended) | github.com/davison/topos-plugin-gdrive, sibling checkout; strongest SRC-06 proof; seeds PLUG-10 | ✓ |
| Private GitHub repo | Proof unverifiable externally; can flip later | |
| Local-only repo | Weakest form of the proof | |

**Q2: How strictly is "developed against the published contract alone" enforced?**

| Option | Description | Selected |
|--------|-------------|----------|
| Separate sessions + rule file (Recommended) | Plugin repo CLAUDE.md restricts refs to the four published inputs; mandatory gap logging | ✓ |
| Import-boundary only | Module boundary alone; kernel knowledge leaks silently | |
| Isolated agent, restricted access | Strongest simulation, most orchestration overhead | |

**Q3: How are contract gaps captured and acted on?**

| Option | Description | Selected |
|--------|-------------|----------|
| Repo log + in-phase doc fixes (Recommended) | CONTRACT-GAPS.md in plugin repo; doc-only fixes republished in-phase; proto changes → PLUG-11 backlog | ✓ |
| Repo log, defer all fixes | Ships a contract known to be misleading | |
| GitHub issues on topos | Scatters the deliverable across the tracker | |

**Q4: How does GSD planning span the two repositories?**

| Option | Description | Selected |
|--------|-------------|----------|
| Plugin repo gets own GSD project (Recommended) | Bootstrapped from a contract-only PRD; topos Phase 14 plans cover kernel side, install/UAT, gap triage | ✓ |
| All planning in topos | topos-side planner would leak kernel internals into the clean-room | |
| Plugin repo unplanned | No execution guarantees on the proof half of the phase | |

---

## Claude's Discretion

- Drive scope & matching (folder targeting, recursion, Shared Drives, item allowlist, match vocabulary)
- Previews & export shapes (per-Workspace-type export formats, bytes+MIME reuse, quota honesty)
- Incremental sync mechanics (changes.list flow, cadence, first-sync strategy)
- Auth CLI ergonomics + named auth health states
- Plugin repo bootstrap mechanics (PRD, rule file, gap-log format, release recipe)
- Folded-todo implementation details (both todos carry their own solution sketches)

## Deferred Ideas

- Contract/proto wire-level changes from the gap log → PLUG-11
- Pull-by-URL install of the Drive plugin → PLUG-10 (backlog Phase 999.1)
- OneDrive source (SRC-07) — future, "same shape as Google Drive"
