# Phase 2: Two Sources, One Trustworthy Stream - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-28
**Phase:** 2-Two Sources, One Trustworthy Stream
**Areas discussed:** None interactively — user delegated all areas to Claude

---

## Area Selection

Four gray areas were presented for discussion:

| Option | Description | Selected |
|--------|-------------|----------|
| SilverBullet in the stream | Page→item mapping, ordering timestamp, preview snippet, rendered vs raw markdown, tag vs page-name matching | — |
| Refresh & sync cadence | Manual refresh placement, background interval, in-flight feedback | — |
| Health, filter & staleness UI | Health display location, filter behavior, stale/unavailable/deleted presentation | — |
| Agent permission config | TOML grant shape, meaning of "agent-facing API" | — |

**User's choice:** "Move on to the next stage, happy to accept agent decisions at this step" — no areas discussed interactively.
**Notes:** All four areas were resolved by Claude's recommended defaults and recorded as locked decisions D-01 through D-12 in 02-CONTEXT.md.

---

## Claude's Discretion

The entire phase discussion was delegated. Key judgment calls made without user input:

- **Last-modified ordering for SilverBullet pages** — creation time isn't reliably exposed by `/.fs`.
- **Single-flight coordinator** (coalesce, never queue) — simplest semantics that satisfy "no stacked concurrent syncs" and the cleanest baseline for Phases 3–5.
- **Separate `/agent/v1` namespace for AGENT-01** — chosen over header-based caller identification (unauthenticated, spoofable, and inverts default-deny) and over filtering the shared `/api/*` (would hide sources from the human UI). Requires revising the "no separate agent API" line in `docs/api.md`, which described AGENT-02 only.
- **Deleted-at-source items** leave the stream at the next successful sync but show an explicit unavailable state until then — the index mirrors source truth rather than accumulating tombstones.

## Deferred Ideas

None.
