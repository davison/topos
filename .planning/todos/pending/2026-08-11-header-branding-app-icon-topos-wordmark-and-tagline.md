---
created: 2026-08-11T18:37:52.739Z
title: "Header branding: app icon, topos wordmark, and tagline"
area: ui
severity: cosmetic
files:
  - web/src/lib/components/WebspaceHeader.svelte
  - web/static/app-icon.png
---

## Problem

The app has no visible branding inside the UI itself — the app icon (added in
Phase 9 as favicon `/app-icon.png`) appears only in the browser tab. The user
wants the topos identity present in the header.

## Solution

Requested layout (user's own words, 2026-08-11):

- App icon displayed in the **top right** of the header.
- Next to it, the word **"topos"** in large-ish font.
- Underneath "topos", the tagline **"bringing all your topics to one place"**
  in smaller text.
- Text colour **muted** compared with the application text colour (i.e. use the
  existing muted-foreground token, not the default foreground).

Implementation notes:

- Header component is `web/src/lib/components/WebspaceHeader.svelte`; the icon
  asset already ships as `web/static/app-icon.png` (embedded into the kernel
  binary via the SvelteKit build).
- Watch the chip row's overflow-measurement logic (`visibleChipCount`) — the
  header's available width feeds it, so a new fixed-width branding block on the
  right must be accounted for rather than silently shrinking chip space.
- Per the standing 07.1 D-11 rule, extend the Playwright e2e suite with a spec
  asserting the branding block renders (icon resolves, wordmark + tagline text,
  muted colour token).

Intended as a quick task (`/gsd-quick`), queued behind quick task 260811-r5d
(mockstrict picker exclusion).
