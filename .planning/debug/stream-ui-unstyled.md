---
status: diagnosed
trigger: "UAT tests 2-4 (G-01-2): stream rows render completely unstyled in built SPA; DetailPane stacks below stream instead of beside it; Tailwind utility-class styling appears absent while hand-authored CSS works"
created: 2026-07-28T10:30:00Z
updated: 2026-07-28T10:50:00Z
---

## Current Focus
<!-- OVERWRITE on each update - reflects NOW -->

hypothesis: "CONFIRMED: web/src/routes/+layout.svelte never imports src/app.css, so app.css (the SOLE entry point for Tailwind v4 AND all hand-authored tokens/component classes) never enters the Vite module graph — the production build emits ZERO CSS"
test: "complete — build tree inspection, source grep, live-server curl, git archaeology"
expecting: "n/a — diagnosis complete"
next_action: "Return ROOT CAUSE FOUND to orchestrator (goal: find_root_cause_only — no fix applied)"

bug_class: Bohrbug (deterministic — 100% reproducible, same result every build)

reasoning_checkpoint:
  hypothesis: "Missing `import '../app.css';` in web/src/routes/+layout.svelte causes the entire stylesheet (Tailwind utilities + design tokens + hand-authored component classes) to be tree-shaken out of the build, because app.css is referenced by no module in the Vite build graph"
  confirming_evidence:
    - "kernel/webui/build/ contains ZERO .css files; _app/immutable/ has no assets/ dir at all"
    - "200.html (and the live server's response at http://127.0.0.1:7777/w/house-move) contains 0 <link rel=stylesheet> tags — only JS modulepreloads"
    - "grep for app.css across web/src/ finds ONLY comments (StreamRow/StreamList doc comments) — no import statement anywhere"
    - "git -S 'app.css' over web/src/routes/ across all history: no commit ever contained the import; the original 01-01 scaffold commit (da15f94) already lacked it"
  falsification_test: "If the hypothesis were wrong, the build output would contain a CSS asset (missing only utility rules), or app.css would be imported somewhere. Neither is true — no CSS asset exists and no import exists."
  fix_rationale: "Adding the single standard SvelteKit import (`import '../app.css';`) to the root layout puts app.css into the module graph; @tailwindcss/vite then processes `@import 'tailwindcss'`, scans source files, and emits the full stylesheet as a linked asset"
  blind_spots: "Not verified by an actual rebuild (diagnose-only mode forbids code changes). Residual risk is low: vite.config.ts already registers tailwindcss() before sveltekit(), and app.css itself is well-formed Tailwind v4 syntax. After the fix, Tailwind v4 auto source detection should cover web/src/**; if any utility is still missing post-fix, check @source coverage — but that is a secondary concern, not this root cause."
  candidate_causes:
    - "code: root layout missing the app.css import (CONFIRMED)"
    - "config: Tailwind v4 source scanning (@source/content detection) not covering web/src/lib/components/*.svelte (ELIMINATED — build has no CSS at all, not merely missing utilities)"
    - "config/build: make build or go:embed picking up stale/wrong output dir (ELIMINATED — build tree freshly written, served HTML identical to build output, adapter paths correct)"
    - "environment: browser dark-theme UA defaults (NOT a cause of the defect, but the cause of the MISLEADING 'hand-authored CSS works' observation)"
  and_gate: "no — the single missing import explains 100% of the observed behavior; no second contributing condition is required. The 'partial styling' of the DetailPane is fully accounted for by browser UA defaults on semantic HTML in a dark-themed browser, not by any CSS actually shipping."

## Symptoms
<!-- Written during gathering, then IMMUTABLE -->

expected: "In built SPA served by Go binary at http://127.0.0.1:7777/w/house-move — stream rows are compact cards on dark (#020617) theme: small consistent portrait thumbnail, one-line ellipsised title, date + tag pills, two-line clamped snippet; detail pane opens BESIDE the stream (two-pane layout) with independent scroll regions"
actual: "Stream rows render completely unstyled — very large preview images centered, unformatted title/date/tags/abstract text, no source links visible, whole-page scrolling. DetailPane renders STACKED BELOW the 35 stream rows; its styling largely works (dark background, heading, working 'Open in paperless-ngx | exact' link, small thumbnail with own scroll region, preview too small). Pattern: hand-authored CSS applies; Tailwind utility-class styling absent."
errors: "None reported; no console errors mentioned"
reproduction: "make build; set -a; . ./.env; set +a; ./bin/webspaces serve; open http://127.0.0.1:7777/w/house-move. Server may already be running on 7777 — curl its assets without restarting."
started: "Discovered during UAT, first ever browser walkthrough of the built (embedded) SPA. Executors verified npm run build/check pass but only eyeballed API JSON. Frontend scaffolded in plan 01-01 (SvelteKit + Tailwind v4 + shadcn-svelte, hand-authored components.json and app.css tokens due to shadcn CLI preset change), detail pane in 01-02, stream row components in 01-03."

## Eliminated
<!-- APPEND only - prevents re-investigating -->

- hypothesis: "Tailwind v4 source scanning (@source / content detection) not covering web/src/lib/components/*.svelte, so the built CSS contains tokens but lacks utility rules"
  evidence: "The build contains NO CSS file whatsoever — not a CSS file missing some rules. If this were a scanning problem, a CSS asset with the token/base layers would still be emitted and linked. Zero CSS assets + zero stylesheet links proves the entry CSS never entered the build graph at all."
  timestamp: 2026-07-28T10:42:00Z

- hypothesis: "app.css @import 'tailwindcss' setup only emits the hand-authored token CSS into the build (partial emission)"
  evidence: "Same as above — nothing from app.css is in the build, including the hand-authored .stream-row-surface/.stream-row-meta classes and the #020617 background token. The DetailPane's 'working' dark styling is browser UA dark-theme defaults on semantic HTML (h2 → bold heading, a → blue link, iframe → 300×150 box with internal PDF-viewer scrollbars), not authored CSS."
  timestamp: 2026-07-28T10:44:00Z

- hypothesis: "make build / go:embed wiring picks up a stale or wrong output directory"
  evidence: "Makefile: npm --prefix web run build → adapter-static writes ../kernel/webui/build (svelte.config.js pages/assets both point there) → go build embeds it. Build tree timestamps are fresh (Jul 28 02:21), and the live server at 127.0.0.1:7777 serves HTML identical in structure to kernel/webui/build/200.html (both with 0 stylesheet links). The embed pipeline faithfully ships what Vite produced; Vite just produced no CSS."
  timestamp: 2026-07-28T10:46:00Z

- hypothesis: "Structural markup bug causes the DetailPane to stack below the list"
  evidence: "web/src/routes/w/[webspace]/+page.svelte markup is correct: <main class='flex h-full min-h-0 gap-8'> with stream div (flex-1 overflow-y-auto) and conditional detail div (w-[480px] border-l). Side-by-side layout depends entirely on the .flex utility; with no CSS, divs fall back to block layout and stack vertically. Markup is fine — CSS absence explains stacking."
  timestamp: 2026-07-28T10:47:00Z

## Evidence
<!-- APPEND only - facts discovered -->

- timestamp: 2026-07-28T10:35:00Z
  checked: kernel/webui/build/ (the embedded production build output)
  found: "ZERO .css files anywhere in the build tree. _app/immutable/ has nodes/, chunks/, entry/ — no assets/ dir. 200.html <head> contains only modulepreload links for JS; NO <link rel=stylesheet> at all."
  implication: "Not just Tailwind utilities missing — the ENTIRE app.css (utilities + hand-authored tokens) is absent from the build. app.css never enters the Vite module graph."

- timestamp: 2026-07-28T10:35:00Z
  checked: web/vite.config.ts, web/src/app.css, web/svelte.config.js, web/package.json
  found: "vite.config.ts has tailwindcss() plugin correctly before sveltekit(). app.css correctly does @import 'tailwindcss' + @theme tokens + hand-authored .stream-row-surface/.stream-row-meta component classes. adapter-static outputs to ../kernel/webui/build with 200.html fallback. tailwindcss ^4.3.3 + @tailwindcss/vite ^4.3.3 both present."
  implication: "Tailwind/Vite/adapter config is all correct. The missing link is whatever should import app.css."

- timestamp: 2026-07-28T10:38:00Z
  checked: "grep -rn 'app.css' web/src/ + full read of web/src/routes/+layout.svelte and +layout.js"
  found: "The ONLY occurrences of 'app.css' in web/src/ are doc comments in StreamRow.svelte and StreamList.svelte. +layout.svelte imports favicon, $app/state, WebspaceHeader — but NOT '../app.css'. No other module imports it either."
  implication: "SMOKING GUN: the standard SvelteKit scaffold line `import '../app.css';` is missing from the root layout. app.css is an orphan file — Vite emits only what the module graph references, so no CSS is built."

- timestamp: 2026-07-28T10:41:00Z
  checked: .planning/phases/01-first-webspace-end-to-end/01-uat-test4-evidence.png (screenshot) vs component sources
  found: "Stream area: grey blocks with centered large images and centered text — exactly UA default <button> rendering (StreamRow's root element is a <button>; UA buttons have ButtonFace background and centered content). Detail block: near-black canvas (browser dark 'website appearance' default for pages with html class=dark and no author CSS), bold large title (<h2> UA default), blue boxed 'Open in paperless-ngx' link (<a> UA default + focus rectangle), tiny 300×150 preview with its own scrollbars (<iframe> UA default size, PDF viewer scrolls internally)."
  implication: "The 'hand-authored CSS works on DetailPane' interpretation in the UAT report is FALSE — no authored CSS applies anywhere. DetailPane merely LOOKS partially styled because it uses semantic HTML whose browser defaults coincidentally resemble the intended dark design. Also: 'no source links in stream rows' is correct-by-design (OpenInSource renders only in DetailPane), not part of this bug."

- timestamp: 2026-07-28T10:45:00Z
  checked: "curl http://127.0.0.1:7777/w/house-move against the running server"
  found: "Served HTML is the SPA fallback (200.html), structurally identical to kernel/webui/build/200.html: 0 stylesheet links, JS modulepreloads only."
  implication: "The running binary embeds and serves exactly the CSS-less build — no serving/embed discrepancy. Root cause is upstream at Vite build time."

- timestamp: 2026-07-28T10:48:00Z
  checked: "git log --follow -- web/src/routes/+layout.svelte; git log --all -S 'app.css' -- web/src/routes/; git show da15f94:web/src/routes/+layout.svelte"
  found: "Only two commits ever touched the root layout (da15f94 plan 01-01 scaffold, 11424a4 plan 01-03 header wrapper). Pickaxe search proves NO commit in history ever contained an app.css import under routes/. The original 01-01 scaffold version already lacked the import (it only imported favicon)."
  implication: "The import was never dropped in a refactor — it was never there. The 01-01 executor hand-authored the layout (plausibly related to the recorded shadcn-CLI-preset-change workaround, where app.css itself was also hand-authored) and omitted the one line the stock `npx sv create` scaffold normally provides. Bug present since the walking skeleton; invisible because `npm run build`/`svelte-check` treat an orphan CSS file as perfectly valid, executors only inspected API JSON, and `make dev` (which would render equally unstyled) was never eyeballed either."

## Resolution
<!-- OVERWRITE as understanding evolves -->

root_cause: "web/src/routes/+layout.svelte is missing the standard SvelteKit root-layout stylesheet import (`import '../app.css';`). src/app.css is the single entry point for ALL styling — Tailwind v4 (`@import 'tailwindcss'` processed by @tailwindcss/vite), the UI-SPEC design tokens, the base layer (body background/foreground/font), and the hand-authored .stream-row-surface/.stream-row-meta classes. Because no module in the app imports it, Vite excludes it from the build graph entirely: the production build emits zero CSS assets and the HTML links no stylesheet. Every reported symptom follows: stream rows render as UA-default <button> elements (grey, centered, natural-size images), the two-pane flex layout collapses to block stacking (DetailPane below the list), scroll containment vanishes (whole-page scrolling), and the DetailPane only APPEARS partially styled because its semantic HTML (h2/a/iframe) picks up browser dark-theme UA defaults that mimic the intended design. Present since the 01-01 scaffold commit (da15f94) — the import never existed in any committed revision."
fix: ""
verification: ""
files_changed: []
