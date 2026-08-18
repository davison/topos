# Phase 12: Filesystem Source - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-13
**Phase:** 12-filesystem-source
**Areas discussed:** Stable item identity, Document scope & previews, Folder→webspace matching, Deep-link behavior

---

## Stable item identity

| Option | Description | Selected |
|--------|-------------|----------|
| Relative path | ID = path relative to source folder; deterministic on all mounts; rename = remove+add | ✓ |
| Inode/device number | Survives local renames but shaky on NFS/SMB; copy/restore churns inodes | |
| Path + content-hash rename detection | Preserves identity across moves at the cost of full reads and a corner-case matrix | |

**User's choice:** Relative path (recommended option)
**Notes:** Chosen with awareness that Phase 13 exclusions and index rows key on this ID — changing later forces an index rebuild.

---

## Document scope & previews

| Option | Description | Selected |
|--------|-------------|----------|
| Doc allowlist + extras override | Doc-ish default (pdf/office/md/txt/images); per-instance extras globs widen/narrow | ✓ |
| Every regular file | No filtering — noisy on messy folders | |
| Fixed allowlist, no override | No configuration escape hatch | |

**User's choice:** Doc allowlist + extras override (recommended option)

Preview follow-up — first framing was rejected by the user with a correction: inline PDF rendering already works for paperless-ngx via Fetch bytes+MIME and the existing media previewer, so it is reuse, not new work.

| Option | Description | Selected |
|--------|-------------|----------|
| Reuse pipeline | PDF+images inline via existing previewer; markdown rendered; text as text; office = metadata + deep link | ✓ |
| Also convert office docs | Server-side office→PDF/HTML conversion (LibreOffice headless) | |

**User's choice:** Reuse pipeline (recommended, corrected framing)
**Notes:** User explicitly flagged the paperless PDF precedent; office conversion recorded as a deferred idea.

---

## Folder→webspace matching

| Option | Description | Selected |
|--------|-------------|----------|
| Folder paths | Plugin declares subfolder paths/names as match field; keywords fallback matches folder names | ✓ |
| Folders + filename tokens | Adds filename-word matching; more reach, more accidental matches | |
| Whole instance per webspace | No sub-matching; multiple topics need multiple instances | |

**User's choice:** Folder paths (recommended option)

---

## Deep-link behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Kernel-mediated open | Loopback-only endpoint runs xdg-open on the item's real path, constrained to indexed items | ✓ |
| file:// link, declared raise-only | Zero new surface, browser blocks it; weakest experience | |
| Open containing folder instead | Reveal in file manager rather than open the file | |

**User's choice:** Kernel-mediated open (recommended option)
**Notes:** Investigated `ExecLinkSpawner` during discussion — it is the WhatsApp QR link spawner, so this endpoint is new (small) machinery following that exec-surface care, not reuse. "Open containing folder" left to Claude's discretion as a possible secondary affordance.

---

## Todo cross-reference

- "Signal schema-version verify-and-accept tooling" (score 0.6, keyword match) — **not folded**; unrelated Signal tooling, stays pending.

## Deferred ideas

- Server-side office-document conversion for inline docx/xlsx previews.
