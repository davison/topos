---
created: 2026-08-05T14:55:00.000Z
title: Centralize rendition theming (and sanitization) in the kernel content boundary
area: api
severity: major
files:
  - kernel/httpapi/item.go
  - plugins/proton/body.go
  - plugins/signal/render.go
  - plugins/silverbullet/render.go
  - docs/plugin-contract.md
---

## Problem

Every plugin that serves iframe-rendered rendition HTML (proton, signal, silverbullet) carries its own self-contained `themeStyle` constant with hardcoded resolved colors — duplicated per plugin because the iframe document boundary can't read the SPA's CSS custom properties. The 260805-j98 scrollbar task made the cost visible: a theme change (thin scrollbars) required editing three plugins' Go source and rebuilding three binaries. This does not scale to the plugin-ecosystem direction (backlog 999.1): external plugin authors would each hand-maintain a copy of the app theme, drift is guaranteed, and a theme change would require N third-party releases. Raised by the user at the 260805-j98 approval: "forcing a matching style into the plugin code isn't scalable and will be a major issue when plugins all become external."

## Solution

Move presentation ownership to the kernel's content-serving boundary (`kernel/httpapi/item.go`, which already owns the rendition CSP):

- Plugins return sanitized content (fragment or body) plus content-shape metadata; they stop emitting `<!doctype>`/`<head>`/theme CSS. The Phase 3 "producing plugin decides readability" rule (text vs HTML choice) stays plugin-side — only presentation moves.
- The kernel wraps the fragment in a single kernel-owned document skeleton and injects one kernel-owned theme stylesheet (source of truth shared with or derived from `web/src/app.css` tokens) before serving under the existing CSP.
- When plugins become third-party (999.1), kernel-side sanitization becomes mandatory anyway — the trust boundary moves into the kernel, so re-sanitizing there (per content-shape policy profiles: email-style-allowlist, chat-no-styles, markdown) is a prerequisite for the ecosystem, not an optional cleanup.

This is a plugin-contract change (Fetch response shape) — schedule it with Phase 5's contract republication or the ecosystem milestone's contract stabilization, not as a standalone quick task. Interim state (theme rules duplicated in three in-repo themeStyle constants, pinned by render tests) is acceptable until then.
