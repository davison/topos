# Phase 6: UI — Scalable Source Surface - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-06
**Phase:** 6-ui-scalable-source-surface
**Areas discussed:** Combined source chip

---

## Area Selection

| Option | Description | Selected |
|--------|-------------|----------|
| Combined source chip | Chip interaction model: click behavior, filter mode, refresh, health detail | ✓ |
| Scaling to 10+ instances | Overflow, grouping, or collapse strategy | |
| Fidelity differentiation | Raise-window-only vs navigating link treatment | |
| Search highlight & date markers | Highlight style, match navigation, scrollbar markers | |

**User's choice:** Combined source chip only; remaining areas explicitly left to Claude's discretion.

---

## Combined Source Chip

### Q1 — What happens when you click a source chip?

| Option | Description | Selected |
|--------|-------------|----------|
| Click = filter toggle | Chip click filters the stream to that source; dot stays passive; refresh a small icon | ✓ |
| Click opens a popover | Panel with health detail, filter toggle, refresh — one tap more for everything | |
| Split zones on the chip | Name = filter, dot = health tooltip, trailing icon = refresh — three hit targets | |

**User's choice:** Click = filter toggle (recommended option).

### Q2 — Should source filtering allow multiple sources selected at once?

| Option | Description | Selected |
|--------|-------------|----------|
| Multi-select | Each chip toggles its source in/out; all-off = show everything; `?sources=` list in URL | ✓ |
| Single-select (keep D-09) | One source at a time, click again for all — today's semantics | |

**User's choice:** Multi-select (recommended option). Supersedes Phase 2 D-09 single-select.

### Q3 — Where does the per-source refresh control live on the chip?

| Option | Description | Selected |
|--------|-------------|----------|
| Hover/focus reveal | Icon appears on hover/keyboard focus (and as spinner while syncing); compact at rest | ✓ |
| Always-visible icon | Permanent smaller icon per chip — discoverable but noisy at 10+ | |
| In the health popover | Refresh inside a dot-triggered popover — cleanest chip, two steps to refresh | |

**User's choice:** Hover/focus reveal (recommended option).

### Q4 — How should health detail (last sync, last error) be surfaced?

| Option | Description | Selected |
|--------|-------------|----------|
| Hover tooltip | Keep today's behavior: display name, relative last-sync, last error on hover | ✓ |
| Click the dot for popover | Pinned popover with full detail; second hit target inside the chip | |
| Tooltip + error strip | Tooltip plus a dismissible error line under the header for erroring sources | |

**User's choice:** Hover tooltip (recommended option).

### Continuation check

**User's choice:** Done discussing — wrap up and write CONTEXT.md; undiscussed areas become Claude's discretion.

---

## Claude's Discretion

- Scaling strategy for 10+ instances (overflow / grouping / collapse)
- Fidelity differentiation treatment (UI-08)
- Search highlighting style, match navigation, and per-variant mechanism (UI-09)
- Stream scrollbar date-marker design and interactivity (UI-11)
- "Refresh all" placement, syncing indicator, selected-state chip styling

## Deferred Ideas

None — discussion stayed within phase scope.
