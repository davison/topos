---
phase: 13-per-item-curation-installable-app
plan: 04
subsystem: pwa
tags: [vite-plugin-pwa, sveltekit, workbox, service-worker, svelte-sonner, playwright, go-mime]

requires: []
provides:
  - "PWA installability: vite-plugin-pwa + @vite-pwa/sveltekit generateSW/autoUpdate, manifest.webmanifest + icon set generated from web/static/app-icon.png, kernel-served with the correct MIME type"
  - "A ServiceWorker that precaches the app shell only (never /api/*) and self-updates a long-lived installed window via periodic + focus/online/visibilitychange checks (web/src/lib/pwa-update.ts)"
  - "web/src/routes/+page.svelte's kernel-unreachable state now reuses StreamError.svelte (with a working Retry) instead of a bespoke, retry-less paragraph"
  - "README.md '### As an app (PWA)' — the UI-14 install/limitation/workarounds documentation"
affects: []

actuals:
  tokens: 68384
  tasks: 4
  commits: 6

tech-stack:
  added: ["vite-plugin-pwa", "@vite-pwa/sveltekit", "@vite-pwa/assets-generator"]
  patterns:
    - "SvelteKitPWA's top-level `base` option must be pinned explicitly ('/') in this repo — SvelteKit's client build environment resolves Vite's own `base` to a relative './' internally (vs '/' for the SSR environment), which silently produces a wrong-scoped ServiceWorker registration if left unset"
    - "A dependency-injected, testable wrapper (web/src/lib/pwa-update.ts) around browser-native ServiceWorker/timer APIs, following this project's existing 'no jsdom, DI over globals' unit-test convention (vitest environment: 'node')"
    - "Never destructure native browser methods (setInterval/addEventListener/etc.) off their host object without re-binding — detached native methods throw 'Illegal invocation' when called with the wrong `this` receiver"

key-files:
  created:
    - web/pwa-assets.config.ts
    - web/static/pwa-64x64.png
    - web/static/pwa-192x192.png
    - web/static/pwa-512x512.png
    - web/static/maskable-icon-512x512.png
    - web/static/apple-touch-icon-180x180.png
    - web/static/favicon.ico
    - cmd/topos/mime_test.go
    - web/e2e/specs/13-pwa-manifest-sw.spec.ts
    - web/src/lib/pwa-update.ts
    - web/src/lib/pwa-update.test.ts
  modified:
    - web/vite.config.ts
    - web/src/app.html
    - web/src/app.d.ts
    - web/package.json
    - web/package-lock.json
    - cmd/topos/main.go
    - web/src/lib/toast.ts
    - web/src/routes/+layout.svelte
    - web/src/routes/+page.svelte
    - web/src/routes/root-empty-state.test.ts
    - web/e2e/specs/uat-04-zero-webspace-vs-outage.spec.ts
    - web/src/lib/components/ui/overlay-primitives.test.ts
    - web/src/lib/components/ui/sonner/sonner.svelte
    - README.md
    - docs/testing.md

key-decisions:
  - "SvelteKitPWA(strategies:'generateSW', registerType:'autoUpdate', kit:{adapterFallback:'200.html', spa:true}, base:'/') — autoUpdate per PD-08 (a prompt-based flow can sit un-acted-on indefinitely in a long-lived installed-app session); base pinned explicitly after discovering SvelteKit's client-build Vite environment resolves base to './' (relative) in this repo, which produced a wrong-scoped SW registration when left unset"
  - "Kernel-unreachable state on '/' (the manifest's start_url) now reuses StreamError.svelte via an extracted loadConfig()/handleRetry() pair, rather than keeping a second, retry-less copy of the same copy — found via the Task 4 checkpoint's real-browser 'kernel not running' check (C), not by inspection"
  - "PWA update detection: registration.update() (never /api) fires periodically (20s) plus on window focus/online and document visibilitychange-to-visible (web/src/lib/pwa-update.ts) — closes the gap that a standalone installed window never navigates, so the browser's own automatic SW update check (navigation-triggered, or ~24h background) never ran"
  - "Toast surface, two rounds per iterative user feedback: round 1 kept the neutral --popover elevation-staircase surface (same token dropdown-menu-content.svelte already uses) with a strengthened border/shadow; round 2 replaced it with SourceChip's own hover-tooltip treatment (bg-foreground/text-background, a full light/dark inversion) after the user judged round 1 still too close to the page background — the tooltip is the one chip-adjacent floating surface that is NOT just another step of the same dark elevation staircase"

patterns-established:
  - "PWA update-check DI pattern (web/src/lib/pwa-update.ts): browser-native APIs (timers, window/document event targets) are always passed in as narrow structural interfaces, never referenced as bare globals inside testable logic — this project's vitest config has no jsdom, so this is what keeps such logic unit-testable"
  - "A visible-but-neutral surface for transient dark-theme UI (toasts, and future similar surfaces) should reuse SourceChip's own tooltip inversion (bg-foreground/text-background) rather than another step of the app's own --background/--card/--popover/--secondary elevation staircase, when the goal is genuine standout contrast rather than subtle layering"

requirements-completed: [UI-13, UI-14]

coverage:
  - id: D1
    description: "The production build emits an installable manifest, a full generated icon set (192/512/maskable), and a Workbox service worker whose precache/navigateFallback/denylist config declares no runtime-caching rules at all"
    requirement: UI-13
    verification:
      - kind: other
        ref: "npm --prefix web run build; manifest field/icon-array assertions (13-04-PLAN.md Task 1 <verify>, adapted for Node's require() not parsing .webmanifest — see Deviations)"
        status: pass
      - kind: other
        ref: "built sw.js source inspection: single precacheAndRoute() call + one NavigationRoute with denylist [/^\\/api\\//], zero other registerRoute calls"
        status: pass
    human_judgment: false
  - id: D2
    description: "The kernel serves manifest.webmanifest with Content-Type application/manifest+json regardless of the host OS mime database, and the served document carries a manifest <link>"
    requirement: UI-13
    verification:
      - kind: unit
        ref: "cmd/topos/mime_test.go#TestRegisterManifestMimeType_WebmanifestResolvesToManifestJSON"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/13-pwa-manifest-sw.spec.ts#GET /manifest.webmanifest returns 200 with Content-Type application/manifest+json"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/13-pwa-manifest-sw.spec.ts#the served document carries a manifest link element"
        status: pass
    human_judgment: false
  - id: D3
    description: "A desktop browser pointed at the kernel offers its own native install affordance (topos adds no install button of its own), and the installed window opens standalone against the local kernel"
    requirement: UI-13
    human_judgment: true
    verification:
      - kind: manual_procedural
        ref: "13-04-PLAN.md Task 4 checkpoint, step A"
        status: pass
    rationale: "Requires a real browser's own install UI and real OS window chrome — not producible by the hermetic Playwright harness. Verified live during the Task 4 checkpoint; not flagged as a problem in either re-verification round."
  - id: D4
    description: "After the kernel binary is upgraded and restarted, the installed app self-updates onto the new build within a few seconds with no manual action, and the update toast is clearly visible"
    requirement: UI-13
    human_judgment: true
    verification:
      - kind: e2e
        ref: "web/e2e/specs/13-pwa-manifest-sw.spec.ts#a window focus event triggers a ServiceWorker update check (checkpoint defect 1)"
        status: pass
      - kind: unit
        ref: "web/src/lib/pwa-update.test.ts (7 cases: periodic interval, override, focus/online/visibility triggers, visibility gate, cleanup)"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/13-pwa-manifest-sw.spec.ts#a toast renders with a genuinely high-contrast background against the page (checkpoint defect 2, round 2)"
        status: pass
      - kind: manual_procedural
        ref: "13-04-PLAN.md Task 4 checkpoint, step B — three rounds (initial fail: sources stayed unreachable after restart, traced to the operator's crossed-builds config, not this plan; auto-update-detection + toast-contrast fixes; final re-verification)"
        status: pass
    rationale: "Requires a real kernel binary rebuild and restart — not producible by the hermetic Playwright harness. The automated cases above prove the update-check wiring and toast contrast mechanically; the human checkpoint is what confirms the end-to-end 'self-updates within a few seconds, clearly visible' experience, and it is now approved."
  - id: D5
    description: "The service worker precaches the app shell only; after a real stream fetch, zero Cache Storage entries exist under /api/"
    verification:
      - kind: e2e
        ref: "web/e2e/specs/13-pwa-manifest-sw.spec.ts#after a page load and a stream fetch, no Cache Storage entry has a /api/ request URL"
        status: pass
    human_judgment: false
  - id: D6
    description: "With the kernel process not running, the installed app renders the existing StreamError copy (with a working Retry button) rather than a browser error page — on both / and /w/[webspace]"
    requirement: UI-13
    human_judgment: true
    verification:
      - kind: unit
        ref: "web/src/routes/root-empty-state.test.ts (structural guard: onMount(loadConfig), the error branch renders <StreamError onretry={handleRetry}>, handleRetry resets to loading and re-invokes loadConfig, StreamError.svelte itself still carries the exact copy and a real Retry button)"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/uat-04-zero-webspace-vs-outage.spec.ts#a failed config fetch renders the outage copy with a working Retry, and never the zero-webspace empty state"
        status: pass
      - kind: manual_procedural
        ref: "13-04-PLAN.md Task 4 checkpoint, step C"
        status: pass
    rationale: "The 'kernel process genuinely not running' case (vs. a live kernel returning an error) is not producible by the hermetic harness, which always drives a real running kernel. Verified live during the checkpoint; the / route's missing Retry button was found here and fixed."
  - id: D7
    description: "The mobile/LAN install limitation (secure-context requirement) is documented with cause and three user-provided HTTPS workarounds, cross-referenced against the kernel's own non-loopback listener warning"
    requirement: UI-14
    verification:
      - kind: other
        ref: "make docs-check (38 links, 19 files, all resolve); grep -n '### As an app (PWA)' README.md; grep -ci 'secure context' README.md >= 1"
        status: pass
      - kind: manual_procedural
        ref: "13-04-PLAN.md Task 4 checkpoint, step D"
        status: pass
    human_judgment: false

duration: ~3h45m (includes two checkpoint round-trips awaiting user re-verification)
completed: 2026-08-14
status: complete
---

# Phase 13 Plan 04: PWA Installability & the Never-Stale Guarantee Summary

**topos installs as a desktop PWA (vite-plugin-pwa/@vite-pwa/sveltekit, generateSW/autoUpdate, kernel-served manifest/icons), self-updates a long-lived installed window via periodic + focus/online-triggered `registration.update()` checks with a genuinely high-contrast update toast (SourceChip's own tooltip inversion), renders the existing `StreamError`/Retry state everywhere the kernel is unreachable (including the root route, not just `/w/[webspace]`), and documents the LAN/mobile secure-context limitation in the README.**

## Performance

- **Duration:** ~3h45m (including two checkpoint round-trips awaiting user re-verification)
- **Completed:** 2026-08-14
- **Tasks:** 4 (3 auto + 1 checkpoint, all complete/approved)
- **Files modified:** 21 (11 created, 15 modified across 6 commits)

## Accomplishments

- `vite-plugin-pwa` + `@vite-pwa/sveltekit` wired with `generateSW`/`autoUpdate`, a generated icon set (`@vite-pwa/assets-generator`, `minimal2023Preset`) from the existing `app-icon.png`, and a Workbox service worker whose precache/navigateFallback/denylist config declares zero runtime-caching rules — proven via built-`sw.js` source inspection and a real-browser Cache Storage check
- `cmd/topos/main.go` registers `.webmanifest` → `application/manifest+json` once at startup, closing the OS-mime-database gap browsers can be strict about for installability
- `web/src/lib/pwa-update.ts`'s `scheduleUpdateChecks` closes the "standalone window never navigates" gap: `registration.update()` (never `/api`) fires periodically and on focus/online/visibility, wired into the existing `onNeedReload`/toast machinery with no changes to that machinery itself
- `web/src/routes/+page.svelte`'s kernel-unreachable state (the manifest's `start_url`, so the first thing a freshly-launched installed app with the kernel down actually renders) now reuses `StreamError.svelte` — same copy, now with a working Retry — instead of a bespoke, retry-less paragraph
- The update/undo/failure toast surface (`sonner.svelte`) now reuses SourceChip's own hover-tooltip inversion (`bg-foreground`/`text-background`) for genuine, WCAG-AA-grade background contrast against the page, after a first attempt (a neutral `--popover` surface) was judged insufficient on re-verification
- README's new `### As an app (PWA)` section documents install, why the desktop path needs no setup, the LAN/mobile secure-context limitation cross-referenced against the kernel's own `isLoopback()` warning, and three user-provided HTTPS workarounds

## Task Commits

Each task was committed atomically (checkpoint fixes as separate, clearly-labeled follow-up commits):

1. **Task 1: PWA build tooling — manifest, icon set, and a service worker that never touches the API** - `35c4a31` (feat)
2. **Task 2: Kernel serving, the update toast, and a browser-driven spec** - `5f28145` (feat)
3. **Task 3: UI-14 — document the mobile/LAN install limitation and its workarounds** - `ae5c294` (docs)
4. **Task 4 checkpoint fix — root route Retry button (defect found in step C)** - `19d34cd` (fix)
5. **Task 4 checkpoint fix — self-updating window + toast contrast round 1 (defects found in step B)** - `bb2ea01` (fix)
6. **Task 4 checkpoint fix — toast contrast round 2, per final user feedback** - `03f4f34` (fix)

## Files Created/Modified

- `web/pwa-assets.config.ts`, `web/static/{pwa-64x64,pwa-192x192,pwa-512x512,maskable-icon-512x512,apple-touch-icon-180x180}.png`, `web/static/favicon.ico` — generated icon set
- `web/vite.config.ts` — `SvelteKitPWA` config (`generateSW`/`autoUpdate`, `kit.adapterFallback`/`kit.spa`, explicit `base:'/'`, `workbox` precache-only scope)
- `web/src/app.html` — manifest `<link>` + `theme-color` meta (SvelteKitPWA doesn't transform `app.html` the way it transforms a plain Vite `index.html`)
- `web/src/app.d.ts` — `/// <reference types="vite-plugin-pwa/client" />` for `virtual:pwa-register`'s ambient module
- `cmd/topos/main.go`, `cmd/topos/mime_test.go` — `.webmanifest` MIME registration
- `web/src/lib/toast.ts` — `pwaUpdatedToast()`
- `web/src/routes/+layout.svelte` — `virtual:pwa-register` wiring (the one import site), `onNeedReload`/sessionStorage-flag reload-then-explain pattern, `onRegisteredSW` → `scheduleUpdateChecks`
- `web/src/lib/pwa-update.ts`, `web/src/lib/pwa-update.test.ts` — the DI-testable periodic/event-triggered update-check module
- `web/src/routes/+page.svelte`, `web/src/routes/root-empty-state.test.ts` — root route reuses `StreamError`/`handleRetry` instead of a bespoke paragraph
- `web/e2e/specs/uat-04-zero-webspace-vs-outage.spec.ts` — updated outage-copy assertions to match `StreamError`'s actual DOM shape, extended to click Retry and confirm recovery
- `web/src/lib/components/ui/sonner/sonner.svelte` — toast surface (two rounds; final: inverted `bg-foreground`/`text-background`)
- `web/src/lib/components/ui/overlay-primitives.test.ts` — added the three new PWA devDependencies to the frozen-set regression guard
- `web/e2e/specs/13-pwa-manifest-sw.spec.ts` — manifest/SW/Cache-Storage proof (Task 2) extended with the update-check-trigger and toast-contrast cases (checkpoint fixes)
- `README.md`, `docs/testing.md` — UI-14 documentation

## Decisions Made

- `registerType: 'autoUpdate'` (PD-08) over a prompt-based flow, matching the plan's own stated reasoning: a prompt can sit un-acted-on indefinitely in a long-lived installed session.
- `SvelteKitPWA`'s top-level `base` must be pinned explicitly to `'/'` in this repo — left unset, it inherits Vite's own resolved `base`, which SvelteKit's CLIENT build environment resolves to `'./'` (relative) while the SSR environment resolves it to `'/'`; unset, this produced `new Workbox("./sw.js", {scope: "./"})`, registering with the wrong scope from any route below root.
- The root route's kernel-unreachable state now shares `StreamError.svelte` (and its `loadConfig()`/`handleRetry()` pair) with `/w/[webspace]`, rather than keeping a second, independently-drifting copy of the same failure copy.
- PWA update detection is DI'd through `web/src/lib/pwa-update.ts` rather than referencing `window`/`document`/timers as bare globals inside testable logic — this project's vitest config has no jsdom (`environment: 'node'`), so this is what keeps the scheduling logic unit-testable at all.
- Toast surface settled on SourceChip's own tooltip inversion (`bg-foreground`/`text-background`) rather than another step of the app's own dark elevation staircase (`--background`/`--card`/`--popover`/`--secondary`) — the only chip-adjacent floating surface offering genuine, not just incremental, contrast against this dark-mode-only theme.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Plan's own `<verify>` script for Task 1 doesn't run as written**
- **Found during:** Task 1
- **Issue:** `node -e "const m=require('./.../manifest.webmanifest')..."` throws `SyntaxError` — Node's `require()` only parses `.json` as JSON, not `.webmanifest`, and falls back to executing the file as JS.
- **Fix:** Verified the same assertions via `JSON.parse(fs.readFileSync(...))` instead.
- **Files modified:** none (verification-only)
- **Committed in:** `35c4a31`

**2. [Rule 3 - Blocking] Approved sharp's npm install script**
- **Found during:** Task 1
- **Issue:** npm's install-scripts allowlist blocked `sharp@0.33.5`'s install script (native binary download) — required by `@vite-pwa/assets-generator` for icon generation. `sharp` is an extremely well-known, widely-used image library (not a new/unverified package name), so this is an install-scripts permission gate, not a package-legitimacy question.
- **Fix:** `npm install-scripts approve sharp` (adds `allowScripts` to `web/package.json`).
- **Files modified:** `web/package.json`
- **Committed in:** `35c4a31`

**3. [Rule 1 - Bug] `@vite-pwa/sveltekit` registered the ServiceWorker at the wrong scope**
- **Found during:** Task 2, via the new e2e spec
- **Issue:** With `base` unset, the compiled register script was `new Workbox("./sw.js", {scope: "./"})` — relative to whatever route the browser is currently on, so from `/w/<webspace>` the SW never registered at all (404 → served the SPA fallback with the wrong script MIME type).
- **Fix:** Set `base: '/'` explicitly on `SvelteKitPWA`'s top-level options.
- **Files modified:** `web/vite.config.ts`
- **Committed in:** `5f28145`

**4. [Rule 1 - Bug, checkpoint C] Root route's kernel-unreachable state had no Retry button**
- **Found during:** Task 4 checkpoint, step C re-verification
- **Issue:** The manifest's `start_url` is `/`, so `+page.svelte` — not `/w/[webspace]` — is the first thing a freshly-launched installed app with the kernel down renders; its pre-existing (07-12-PLAN.md) error phase was a bespoke, copy-only paragraph with no retry affordance.
- **Fix:** Extracted `loadConfig()`/added `handleRetry()`; error branch now renders `<StreamError onretry={handleRetry} />`.
- **Files modified:** `web/src/routes/+page.svelte`, `web/src/routes/root-empty-state.test.ts`, `web/e2e/specs/uat-04-zero-webspace-vs-outage.spec.ts`, `web/src/lib/components/ui/overlay-primitives.test.ts` (unrelated pre-existing devDependencies-guard gap caught by the same `npm run test` pass)
- **Committed in:** `19d34cd`

**5. [Rule 1 - Bug, checkpoint B defect 1] No automatic update detection for a long-lived installed window**
- **Found during:** Task 4 checkpoint, step B re-verification
- **Issue:** A standalone installed window never navigates, so the browser's own automatic SW update check never ran; only a manual browser-menu "Reload" surfaced a new build.
- **Fix:** `web/src/lib/pwa-update.ts`'s `scheduleUpdateChecks` — periodic + focus/online/visibilitychange-triggered `registration.update()`.
- **Files modified:** `web/src/lib/pwa-update.ts`, `web/src/lib/pwa-update.test.ts`, `web/src/routes/+layout.svelte`, `web/e2e/specs/13-pwa-manifest-sw.spec.ts`
- **Committed in:** `bb2ea01`

**6. [Rule 1 - Bug, found via the new e2e test] "Illegal invocation" from detached native timer methods**
- **Found during:** the same checkpoint-fix round, while writing the e2e proof for #5
- **Issue:** `timers: { setInterval, clearInterval }` destructured off the global scope detaches these native methods from their required `window` receiver — calling them via `options.timers.setInterval(...)` throws `TypeError: Illegal invocation`. The failure was silently swallowed inside `registerSW`'s own promise chain (an unset `onRegisterError` optional-chains to a no-op), so nothing about the wiring appeared broken until the e2e spec actually asserted on it.
- **Fix:** Bound both to `window` explicitly (`window.setInterval.bind(window)`); added a `try/catch` + `console.error` around the `onRegisteredSW` body so a future regression here surfaces instead of vanishing silently.
- **Files modified:** `web/src/routes/+layout.svelte`
- **Committed in:** `bb2ea01`

**7. [Rule 1 - Bug, checkpoint B defect 2, two rounds] Update toast had insufficient contrast**
- **Found during:** Task 4 checkpoint, step B re-verification (round 1); user's final approval message (round 2)
- **Issue round 1:** `--normal-border` was the raw `--border` token, too close in lightness to the toast's own `--popover` background to read as a visible edge, and svelte-sonner's hardcoded box-shadow is black — invisible against an already near-black page.
- **Fix round 1:** Strengthened border via `color-mix(... --muted-foreground ...)`; overrode the hardcoded shadow via `toastOptions.style`.
- **Issue round 2:** User re-verification: "functionally approved... the background itself, not just the border, needs to contrast with the main scheme... similar to the popovers on the source chips."
- **Fix round 2:** Reused SourceChip's own hover-tooltip inversion (`bg-foreground`/`text-background`) instead of another step of the same dark elevation staircase — the round-1 surface (`--popover`) is the SAME token `dropdown-menu-content.svelte` already uses for the chip menus, so reusing it literally could not have increased background contrast; the tooltip is the one chip-adjacent surface that genuinely differs in luminance.
- **Files modified:** `web/src/lib/components/ui/sonner/sonner.svelte`, `web/e2e/specs/13-pwa-manifest-sw.spec.ts` (round 1: `bb2ea01`; round 2: replaced the "not equal" assertion with a WCAG-style relative-luminance contrast-ratio check, >= 4.5)
- **Committed in:** `bb2ea01` (round 1), `03f4f34` (round 2)

---

**Total deviations:** 7 auto-fixed (4 bugs found via the plan's own build/e2e process, 1 blocking npm install-scripts permission, 2 checkpoint-driven UX fixes with 3 constituent rounds)
**Impact on plan:** All fixes were either bugs in the plan's own verify script/tooling interaction, or defects the Task 4 checkpoint itself exists to catch (real-browser install/upgrade/kernel-down behavior a hermetic harness cannot produce). No scope creep — every fix stayed inside UI-13/UI-14's own success criteria.

## Issues Encountered

- **Checkpoint failure 1 (initial round): "all sources unreachable" after kernel restart — diagnosed as NOT originating in this plan.** The user's FIRST re-verification round reported that after a kernel rebuild+restart, every source showed a yellow (warning-tone) health chip that never recovered. Diagnosis (documented in the interim checkpoint report, not fixed here): `web/src/lib/format.ts`'s `healthTone()` only reaches the warning branch when `source.reachable === true` — a launch-time refusal (including wave-1's new build-manifest verification gate, already in this worktree's base) would render destructive/red via an earlier branch, not warning/yellow, ruling that out. Wave-1's manifest-gate commits (`38178f6`, `c083de9`) are also scoped entirely to `kernel/pluginhost`/`kernel/httpapi/sources.go` and test-only touches elsewhere — never `kernel/syncer` (scheduler/coordinator), the actual sync-execution path. The user subsequently root-caused it themselves to a crossed-builds `plugins.dir` config pointing outside this worktree (not a topos code defect) and captured a todo for config separation — no code change was needed in this plan.
- **`go:embed` / manifest-generation ordering during manual verification.** Every hand-triggered rebuild during debugging had to go through the full `make e2e` pipeline (SPA build → plugin build → `cmd/topos-manifest` hashing → kernel build with the `-ldflags -X` manifest) — a partial rebuild (e.g. `go build -o bin/topos` alone, without regenerating the manifest) silently produces a kernel with an EMPTY trust manifest, refusing every trusted-tier plugin at launch. Cost one debugging round-trip (a hung `waitForFirstSync` with no obvious cause) before tracing it back to this.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- UI-13 and UI-14 are both satisfied and marked complete in REQUIREMENTS.md.
- The installed-app self-update mechanism (`web/src/lib/pwa-update.ts`) and the `StreamError`/Retry reuse pattern are generic enough that any future route needing the same "kernel unreachable" treatment can reuse them directly.
- No blockers for later Phase 13 plans. The crossed-builds `plugins.dir` config issue surfaced during checkpoint B re-verification is captured as a todo (config separation) outside this plan's scope — not a blocker for anything in this phase, but worth routing to its own future fix.
- `go vet ./cmd/topos/...`, `go test ./cmd/topos/... -count=1`, `npm --prefix web run test` (1017/1017), `npm --prefix web run check`/`check:e2e` (0 errors), and `make e2e` (131/131) all pass clean.

---
*Phase: 13-per-item-curation-installable-app*
*Completed: 2026-08-14*
