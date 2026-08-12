# Phase 7: Webspace Builder UI - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-07
**Phase:** 7-Webspace Builder UI
**Areas discussed:** Config persistence & clobber safety, Apply/reload semantics, Builder UX shape, Search-promotion filter semantics

---

## Config persistence & clobber safety

### Persistence strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Surgical round-trip (Recommended) | Kernel edits only changed keys, preserving comments/ordering/unknown keys; Go has no mature comment-preserving TOML editor — main technical risk | |
| Canonical rewrite | UI save regenerates the whole file in one canonical machine-written form; hand comments flattened on first save | ✓ |
| Two-file split | Hand file never touched; UI writes a separate merged-at-load file | |

**User's choice:** Canonical rewrite

### Clobber guard

| Option | Description | Selected |
|--------|-------------|----------|
| Optimistic lock: refuse + reload (Recommended) | Hash recorded at load; save re-checks, rejects on drift, kernel reloads, UI refreshes | ✓ |
| Backup then overwrite | UI always wins; previous file to timestamped backup | |
| Watch + auto-reload | Kernel watches file, reloads on change; stale saves rare but hash check still needed | |

**User's choice:** Optimistic lock: refuse + reload

### Backups

| Option | Description | Selected |
|--------|-------------|----------|
| Backup on every UI save (Recommended) | Rotated last-N backups directory | |
| One-time backup only | Only the first rewrite of a hand-authored file backed up | |
| No backups | User keeps config under their own version control | |

**User's choice:** Free text — "keep a single .bak file only" (every UI save overwrites `config.toml.bak` with the outgoing file)

### Canonical file style

| Option | Description | Selected |
|--------|-------------|----------|
| Minimal + header pointer (Recommended) | Values only + generated header pointing at config.example.toml docs | ✓ |
| Self-documenting | Generated doc comments above every key | |
| Bare values only | Pure TOML, zero comments | |

**User's choice:** Minimal + header pointer

**Notes:** Hard requirement recorded (not asked): `${ENV_VAR}` secret references written back verbatim, never expanded.

---

## Apply/reload semantics

### Apply model

| Option | Description | Selected |
|--------|-------------|----------|
| Save = apply immediately (Recommended) | Validate + write + hot-swap running config in one request | ✓ |
| Save, then explicit Reload | File write and apply are separate steps | |
| Save applies, restart for structural | Instance add/remove requires restart | |

**User's choice:** Save = apply immediately

### Reconciliation eagerness

| Option | Description | Selected |
|--------|-------------|----------|
| Eager: sync what changed (Recommended) | New/changed instances sync immediately; removed instances shut down, rows removed | ✓ |
| Lazy: wait for the schedule | Nothing syncs until next interval/manual refresh | |
| Eager for new, lazy for edits | Middle ground, inconsistent to explain | |

**User's choice:** Eager: sync what changed

### Hand-edit path

| Option | Description | Selected |
|--------|-------------|----------|
| Watch + auto-reload (Recommended) | Kernel watches config.toml, hot-applies valid changes on write | |
| Explicit reload affordance | 'Reload config' button/API re-reads the file on demand | ✓ |
| Restart only | Hand-edits apply at next restart | |

**User's choice:** Explicit reload affordance

### Validation surfacing

| Option | Description | Selected |
|--------|-------------|----------|
| Validate-on-save only (Recommended) | Save runs full load-time validation as dry-run; one code path | ✓ |
| Live field validation + save check | Extra dry-run endpoint marking bad fields while editing | |
| UI-side rules mirrored client-side | Client reimplements validation; drift risk | |

**User's choice:** Validate-on-save only

---

## Builder UX shape

### Placement

| Option | Description | Selected |
|--------|-------------|----------|
| Settings area + per-webspace edit (Recommended) | Settings section for instances + Edit affordance per webspace | ✓ (superseded below) |
| Single settings area only | All config in one section | |
| Everything inline per-webspace | No global settings area | |

**User's choice:** Initially "Settings area + per-webspace edit"; superseded by the freeform in-header composition model below.

### Flow shape (iterated freeform)

Options initially presented: Two sections inline create (Recommended) / Guided wizard / Strictly separate sections. User answered freeform twice, converging on an in-header composition model; Claude synthesized and the user locked it:

- Webspace title → drop-down switcher of all webspaces + "+" to create; standalone home page retired (root redirects; empty state covers first-run)
- "+" at end of source-chip row adds a source: existing instance → match-fields-only modal; "New <plugin type>…" → two-step modal (connection then match)
- Edit existing source via chip menu/popover — never plain chip click (click stays filter)
- "Manage sources…" in the title drop-down as escape hatch: instance edit/delete, webspace delete, Reload config
- UI-built webspaces write an explicit `sources` allowlist; hand-written webspaces keep D-03 default-all

**User's confirmation:** "yes, that matches - lock it in. Once we see it working, it can be refined a little, but I doubt any big changes would be needed"

### Secrets in the modal

| Option | Description | Selected |
|--------|-------------|----------|
| Env-var name + set/unset badge (Recommended) | Form takes variable NAME; kernel reports set/unset; unset saves with warning | ✓ |
| Name only, no resolution check | No feedback; typos invisible until sync fails | |
| Block save until set | Refuses save when variable unset | |

**User's choice:** Env-var name + set/unset badge

---

## Search-promotion filter semantics

### Filter layer

| Option | Description | Selected |
|--------|-------------|----------|
| Query-time view filter (Recommended) | FTS query applied when reading the stream only | |
| Sync-time narrowing | Filter becomes part of correlation; re-sync per change | |
| Hybrid: view filter, agent-visible | Query-time, but also applied to /agent/v1 reads | ✓ |

**User's choice:** Hybrid: view filter, agent-visible

### Storage

| Option | Description | Selected |
|--------|-------------|----------|
| Config key on the webspace (Recommended) | Part of the webspace definition; rides builder/hand-edit/lock/reload machinery | ✓ |
| Index DB as UI state | Second persistence store invisible to hand-editing | |

**User's choice:** Config key on the webspace

### Stacking

| Option | Description | Selected |
|--------|-------------|----------|
| Stackable AND list (Recommended) | Each promotion appends; terms AND; independently removable | ✓ |
| Single filter, replace on promote | One string, replaced with confirm | |
| Single filter, promote extends | One string rewritten as AND of old + new | |

**User's choice:** Stackable AND list

### Filter UX

| Option | Description | Selected |
|--------|-------------|----------|
| Filter chips row + save affordance (Recommended) | 'Save as filter' by search box; filters as distinct removable chips | ✓ |
| Badge + manage in editor | Compact badge; edits only in webspace editor | |
| Merge into search box | Locked tokens inside the search input | |

**User's choice:** Filter chips row + save affordance

---

## Claude's Discretion

- Mutating-API endpoint design (following docs/api.md conventions; loopback/no-auth posture unchanged)
- Raw pre-expansion config retention mechanism, TOML serializer choice, atomic-write mechanics
- Hot-apply internals (config diffing, pluginhost lifecycle ordering, in-flight sync handling)
- Modal/form layout, picker presentation, delete confirmations, first-run empty state, root-redirect target
- FTS semantics for stacked filter terms; filter-chip vs source-chip visual interaction
- Plugin-type discovery feed for the "+" picker

## Deferred Ideas

- Comment-preserving TOML round-trip (declined for canonical rewrite)
- File watcher / auto-reload of hand-edits (declined for explicit Reload)
- WR-01 highlight case-fold fix — fold offered, declined; stays an open Phase 6 advisory
- Signal schema-version verify-and-accept tooling todo — reviewed (0.2 match), declined, stays deferred
