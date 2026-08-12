# Phase 5: Source Instances & Per-Type Matching - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-06
**Phase:** 5-Source Instances & Per-Type Matching
**Areas discussed:** Todo folding, Config shape, Matching semantics, Migration & compat, Instance lifecycle, Rendition-theming scope

---

## Todo Folding

| Option | Description | Selected |
|--------|-------------|----------|
| Rendition theming | Move theme/sanitization from plugin themeStyles to the kernel content boundary — rides the same contract break | ✓ |
| Signal schema tooling | Verify-and-accept tooling for future user_version bumps — standalone | |

**User's choice:** Fold rendition theming only; Signal tooling stays a pending todo.

---

## Config Shape

### Where matching config lives (with TOML previews)

| Option | Description | Selected |
|--------|-------------|----------|
| Webspace-centric + fallback | Per-instance match blocks under the webspace, optional webspace-level default keywords | ✓ |
| Webspace-centric, explicit only | Every participating instance must have an explicit typed match block | |
| Source-centric | Each source instance declares which webspaces it feeds | |

### Override semantics

| Option | Description | Selected |
|--------|-------------|----------|
| Replace | Explicit match block fully replaces the fallback for that instance | ✓ |
| Extend | Block terms add on top of fallback keywords | |

### Instance participation

| Option | Description | Selected |
|--------|-------------|----------|
| All + optional allowlist | Every instance participates by default; optional `sources = [...]` restricts | ✓ |
| Match block = membership | Only explicitly-configured instances participate | |
| All, no restriction | Every instance always participates | |

---

## Matching Semantics

### Value semantics

| Option | Description | Selected |
|--------|-------------|----------|
| Exact-only | Exact case-insensitive native names within typed fields (D-03 determinism carried) | ✓ |
| Exact + hierarchy | Plus IMAP subfolder expansion | |
| Plugin-defined syntax | Each plugin defines its own matcher semantics | |

### Vocabulary ownership

| Option | Description | Selected |
|--------|-------------|----------|
| Plugin declares it | Describe response declares match fields; kernel validates, unknown field fails loudly | ✓ |
| Kernel hardcodes it | Built-in table of known plugins' fields | |

---

## Migration & Compat

### Config shape compatibility

| Option | Description | Selected |
|--------|-------------|----------|
| Clean break | Kernel rejects old shape with clear error; user's config hand-migrated in-phase | ✓ |
| Auto-migrate once | Kernel rewrites old config with backup | |
| Support both | Old shape works indefinitely | |

### Index data

| Option | Description | Selected |
|--------|-------------|----------|
| Drop & re-sync | Index is a cache; rebuild from sources (one full backfill) | ✓ |
| Migrate rows in place | SQL migration of source identity | |

---

## Instance Lifecycle

### Identity

| Option | Description | Selected |
|--------|-------------|----------|
| Key = identity | TOML map key is the durable instance id; key rename = new instance (re-sync) | ✓ |
| Separate stable id field | Explicit id field, key freely renameable | |

### display_name rules

| Option | Description | Selected |
|--------|-------------|----------|
| Optional, unique, loud | Defaults to key; duplicates rejected at config load | ✓ |
| Optional, duplicates allowed | UI disambiguates | |
| Required, unique | Always explicit | |

---

## Rendition-Theming Scope (folded todo)

| Option | Description | Selected |
|--------|-------------|----------|
| Theming + sanitization | Kernel re-sanitizes per content-shape policy AND wraps/themes — one contract break, trust boundary done properly | ✓ |
| Theming only | Kernel wraps/themes; plugins keep sanitizing (second contract change later) | |

---

## Claude's Discretion

- Proto mechanics of the contract change (message evolution, contract version bump, RPC-allowlist updates)
- Exact TOML field/key names beyond the previewed shapes; validation error wording
- Content-shape taxonomy and per-shape sanitizer policy details
- Kernel rendition stylesheet ↔ app.css token consistency mechanism
- Work ordering within the phase; live-config migration sequencing

## Deferred Ideas

- Richer per-type matchers (hierarchy, globs, plugin syntax) — additive later
- Config auto-migration / writing — Phase 7
- Scalable source-surface UI — Phase 6
- Signal schema verify-and-accept tooling — pending todo, reviewed but not folded
