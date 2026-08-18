# Phase 13: Per-Item Curation & Installable App - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-14
**Phase:** 13-per-item-curation-installable-app
**Areas discussed:** Todo folding, Exclude interaction, Excluded-items view, Orphaned exclusions, Trust-tier hardening (folded), TODO.md reconciliation

---

## Todo Folding

| Option | Description | Selected |
|--------|-------------|----------|
| Fold: trust-tier hardening | Phase 11 debt — consent flow bypassable via config edit/file drop/D-11 shadowing | ✓ |
| Fold: Signal schema tooling | Verify-and-accept tooling for new Signal Desktop schema versions | |
| Fold neither | Both stay pending | |

**User's choice:** Fold the trust-tier hardening todo into Phase 13; Signal tooling stays pending.

---

## Exclude Interaction

### Where the action lives

| Option | Description | Selected |
|--------|-------------|----------|
| Both row + detail pane | Hover-revealed row action + detail pane action | |
| Detail pane only | Exclusion requires opening the item first | |
| Stream row only | Curation in the stream; pane view-only | |

**User's choice:** Free-text — stream multi-select (shift-click/ctrl-click standards) treated with "include all"/"exclude all" bulk actions, plus a single-item exclude button in the detail pane.

### Selection mechanics

| Option | Description | Selected |
|--------|-------------|----------|
| Ctrl/shift only + action bar | Plain click still opens pane; floating action bar while selection non-empty; Esc/Clear empties | ✓ |
| Explicit select mode | Header toggle enters checkbox mode | |
| Row checkboxes always | Hover-revealed checkbox on every row | |

### Exclude feel

| Option | Description | Selected |
|--------|-------------|----------|
| Instant + undo toast | Immediate removal, toast with Undo; no confirms | ✓ |
| Confirm bulk only | Confirm at 2+ selected | |
| Always confirm | Every exclusion confirms | |

### Reach

| Option | Description | Selected |
|--------|-------------|----------|
| All surfaces | Stream, FTS search, date markers, counts | ✓ |
| Stream only | Search still surfaces excluded items | |

### Digest granularity

| Option | Description | Selected |
|--------|-------------|----------|
| Per-digest is correct | Digest row is an item like any other; conversation-wide is match-rules territory | ✓ |
| Offer conversation-wide too | Second option keyed on conversation/group id | |

---

## Excluded-Items View

### Placement

| Option | Description | Selected |
|--------|-------------|----------|
| Stream filter toggle | Stream itself flips to the excluded bucket; reuses StreamRow/pane/multi-select | ✓ |
| Modal | ManageSourcesModal-style dialog | |
| Own route | /w/{name}/excluded page | |

### Toggle visibility

| Option | Description | Selected |
|--------|-------------|----------|
| Only when count > 0 | Zero-clutter default; count is the discovery cue | ✓ |
| Always visible | Discoverable before first use | |

### Ordering

| Option | Description | Selected |
|--------|-------------|----------|
| Chronological, like stream | Same ordering + date markers | ✓ |
| By exclusion time | Most-recently-excluded first | |

### Un-exclude mechanics

| Option | Description | Selected |
|--------|-------------|----------|
| Mirror of exclude | Multi-select + "Include" action bar + pane button, instant + undo | ✓ |
| Per-row button only | Visible restore button, no multi-select | |

---

## Orphaned Exclusions

### Mark fate when item vanishes from source

| Option | Description | Selected |
|--------|-------------|----------|
| Silent auto-prune | Mark swept when a healthy sync omits the item; reappearing item returns unexcluded | ✓ |
| Keep mark, hide row | Mark persists to protect against reappearance | |
| Show as gone | Orphaned marks listed flagged until dismissed | |

### Edge acceptance

| Option | Description | Selected |
|--------|-------------|----------|
| Accept both | Rename loses exclusion (Phase 12 identity model); prune only on healthy sync, never on failed sync or index rebuild | ✓ |
| Discuss further | | |

---

## Trust-Tier Hardening (Folded Todo)

### Remediation direction

| Option | Description | Selected |
|--------|-------------|----------|
| Build-provenance manifest | Link-time-embedded first-party hash set; unverifiable trusted-dir binaries lose trusted treatment | ✓ |
| Confirm-gate plugins.dir | Non-hot-applyable dir changes; partial fix | |
| Both, layered | Manifest + confirm gate | |

### Demotion loudness (superseded by TODO.md reconciliation below)

| Option | Description | Selected |
|--------|-------------|----------|
| Named health state + badge | "Unverified binary" state + untrusted badge + tooltip | ✓ (superseded) |
| Badge only | Quieter | |

### D-11 shadowing surfacing

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — surface it | Health state/advisory instead of log line | ✓ |
| Log line stays | Manifest demotion defangs the attack | |

### Locally-built Signal plugin

| Option | Description | Selected |
|--------|-------------|----------|
| Demote + pin, honestly | External-tier consent + pin + badge | ✓ (with note) |
| User-declared trusted pin | Config escape hatch | |
| Rethink manifest scope | | |

**User's choice:** Free-text — "demote and pin, but document clearly why in the plugin doc".

---

## TODO.md Reconciliation (user-raised at the done gate)

User pointed at `TODO.md` §"Plugin Trust System" before locking the approach.

### Hash mismatch behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Refuse to load, loudly | TODO.md's position: binary never launches; named error state in log AND UI; consent/re-accept is the only path | ✓ |
| Demote to untrusted | Runs with external-tier semantics | |
| Split by tier | Trusted refuses, external keeps soft-fail | |

**Notes:** Supersedes the earlier "demote to untrusted" answer — unverified code never executes.

### Directory layout

| Option | Description | Selected |
|--------|-------------|----------|
| Keep two dirs this phase | Manifest is authority; dirs stay as shipped conveniences; collapse at PLUG-10 | ✓ |
| Collapse to one now | TODO.md end-state; config migration churn | |

### Distribution scope

| Option | Description | Selected |
|--------|-------------|----------|
| Defer all to backlog | publish-plugin GHA, pull-by-URL, PR promotion → Phase 999.1 (PLUG-10/11) with TODO.md notes folded in | ✓ |
| Pull publish-plugin GHA in | Fork-side workflow + docs only | |
| Pull all of it in | Full vision this phase (~doubles the phase) | |

---

## Claude's Discretion

- PWA mechanics end to end (tooling, manifest, icons, SW update strategy, kernel serving, kernel-down window state) — user left the PWA update-experience area undiscussed deliberately
- Marks storage mechanics (separate file vs rebuild-exempt table, schema, endpoint shapes)
- Undo-toast duration, keyboard shortcuts, action-bar copy
- Manifest generation mechanics and refuse-to-load state naming/copy
- Excluded-count toggle placement within the webspace chrome

## Deferred Ideas

- Plugin distribution (publish-plugin GHA, pull-by-repo-URL install, PR-merge trust promotion, single managed plugin dir) → backlog Phase 999.1 (PLUG-10/11), absorbing TODO.md §Plugin Trust System
- Conversation-wide exclusion for chat digests → match-rules/config territory if ever wanted
- Plugin picker search/filter box → vNext (already in TODO.md)
