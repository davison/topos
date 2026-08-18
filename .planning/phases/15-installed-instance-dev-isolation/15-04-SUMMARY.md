---
phase: 15-installed-instance-dev-isolation
plan: 04
subsystem: infra
tags: [dev-isolation, devguard, ports, makefile, vite, go]

requires:
  - phase: 14-webapp
    provides: "14-01's per-checkout config.dev.toml split (the config-path-as-instance-identity foundation)"
  - phase: 15-installed-instance-dev-isolation
    provides: "15-01's resolvePluginsDir conventions the guard mirrors; the installed instance the guard protects"
provides:
  - "cmd/topos-devguard (ISOL-01): pre-flight refusal when any declared writable path — config file, index, either plugin dir (omitted external_dir default included), any source store — resolves inside the topos config/state roots"
  - "Dev loop on 7778 end to end (ISOL-02): Makefile DEV_PORT, config.dev.example.toml listen, Vite /api proxy — installed instance keeps 7777"
  - "Stale-dev-config port-consistency refusal by name, before any child process"
  - "DEV_ISOLATION_BYPASS: the single, loud, total escape hatch (warn-only banner naming every permitted path)"
  - "config.dev.example.toml per-checkout plugin-store convention (@CHECKOUT@/bin/plugin-state)"
affects: [15-05]

actuals:
  tokens: 16000
  tasks: 3
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Guard loads config through kernel/config.NewStore — one parser, one document, never a second TOML reader"
    - "Component-wise containment (filepath.Rel), never string prefixes — one function, every call site"
    - "Escape hatches are total and loud: no per-key suppression, banner lists everything permitted"

key-files:
  created:
    - cmd/topos-devguard/main.go
    - cmd/topos-devguard/devguard_test.go
  modified:
    - Makefile
    - config.dev.example.toml
    - web/vite.config.ts
    - scripts/dev-guard-smoke.sh

key-decisions:
  - "Omitted [plugins] external_dir is a violation, not a pass — the kernel's omitted-key default resolves inside the state root, so the template now documents that the key must stay present"
  - "Relative plugin dirs resolve against the config file's directory in the guard (a dev kernel's go-run exe dir is a throwaway build dir that can never be a topos root); absolute values verbatim"
  - "DEV_PORT consistency is enforced, not advisory: overriding DEV_PORT now requires a matching listen value — the refusal names both values and the delete-and-regenerate fix"
  - "Production-config dev runs require BOTH DEV_CONFIG and DEV_ISOLATION_BYPASS — which config, and explicit acceptance of what that means"

patterns-established:
  - "dev-guard-smoke per-case generated dev configs: every case owns an isolated, port-matched config under $WORK with XDG roots pointed into $WORK"

requirements-completed: [ISOL-01, ISOL-02]

coverage:
  - id: D1
    description: "A dev run refuses to start when any declared writable path resolves inside the topos config or state root, reported in one deterministic pass"
    requirement: ISOL-01
    verification:
      - kind: unit
        ref: "cmd/topos-devguard/devguard_test.go#TestDevguard (13 subtests: containment, adjacency, omitted-default, determinism, warn-only, ports)"
        status: pass
      - kind: integration
        ref: "make dev-check (Case 4: recipe refuses naming [index] path, no child started; Case 5: bypass banner + progression)"
        status: pass
  - id: D2
    description: "Dev loop binds 7778 while the installed instance keeps 7777; a stale 7777 dev config fails by name in seconds, not as a readiness timeout"
    requirement: ISOL-02
    verification:
      - kind: integration
        ref: "make dev-check (Case 6: port mismatch names both ports, elapsed < 30s of a 60s readiness timeout); grep gates on the three 7778 sites and unchanged production 7777 counts"
        status: pass
      - kind: automated_ui
        ref: "npm --prefix web run build (Vite config change valid)"
        status: pass

duration: 35min
completed: 2026-08-19
---

# Phase 15 Plan 04: Dev isolation as a mechanical gate Summary

`cmd/topos-devguard` — loading the dev config through the kernel's own parser and deriving the kernel's own protected roots — now refuses any dev run whose config file, index, plugin directories, or source stores would touch what the installed instance owns, in one deterministic pass; the dev loop moved to 7778 across its three naming sites with a fast, named refusal for stale 7777 configs; and the only bypass is a single variable that banners every path it permits.

- Duration: ~35min (2026-08-18T23:39Z → 2026-08-18T23:46Z execution tail; TDD RED→GREEN cycle included)
- Tasks: 3 (TDD guard, port move, wiring + smoke)
- Files: 6 (2 created, 4 modified)

## Accomplishments

- TDD: 13-subtest RED suite first (`773d347`), then the guard implementation GREEN on first run (`16fa22e`). Containment is one component-wise function (equal-to-root refuses, `topos-extra` siblings pass); findings sort by config key path for byte-identical repeat runs; `--warn-only` prints identical findings behind the bypass banner on stderr and exits 0; `--expected-port` catches the stale-port case.
- Port move: `DEV_PORT ?= 7778`, template listen 7778, Vite proxy 7778; production 7777 counts verified unchanged in `config.example.toml` and `kernel/config/types.go`; dev-config notice announces the port; the recipe's guard invocation runs before the existing `ss` port guard.
- Wiring: one guard invocation gates isolation + port; `DEV_ISOLATION_BYPASS` documented as the deliberate companion to `DEV_CONFIG`; template gains the `@CHECKOUT@/bin/plugin-state` store convention, the separate-linked-device WhatsApp example, and the external_dir-must-stay note.
- dev-guard-smoke: 3 → 6 cases; existing cases now generate per-case isolated port-matched configs (the guard runs in every case, proving the test seam survived).

## Task Commits

1. Task 1 RED: `773d347` test(15-04) add failing devguard behaviour suite
2. Task 1 GREEN: `16fa22e` feat(15-04) implement topos-devguard — one-pass deterministic isolation refusal
3. Task 2: `e6cb5f1` feat(15-04) move the dev loop to 7778 and catch a stale dev config by name
4. Task 3: `ed5a5d8` feat(15-04) wire the isolation guard into make dev with a loud escape hatch and template store convention

## Deviations from Plan

- **[Rule 2 - Missing critical] dev-guard-smoke case adaptation pulled forward from Task 3 into Task 2** — Found during: Task 2. Issue: inserting the guard into the `dev` recipe made the three existing smoke cases fail (their ephemeral DEV_PORT never matched the checkout's config.dev.toml), but Task 2's own verify requires `make dev-check` green. Fix: the per-case generated-config adaptation (a Task 3 concern) was implemented in Task 2's commit; Task 3 then added only the three new cases. Files: scripts/dev-guard-smoke.sh. Verification: `make dev-check` green at both commits. Commit: `e6cb5f1`.

**Total deviations:** 1 auto-fixed (task-boundary reordering only — no scope change). **Impact:** none on plan outcomes; every acceptance criterion of both tasks passes.

## Verification Results

- `go test ./cmd/topos-devguard/ -v` — 13/13 subtests pass; mutation-grade RED observed first (all 13 failing against the stub)
- `make dev-check` — all six cases pass
- `make test-portable` — green across all modules
- `npm --prefix web run build` — green
- Generated-template check: `sed` a scratch checkout template → `topos-devguard --config` exits 0

## Issues Encountered

- **Operator note:** this checkout's own `config.dev.toml` (gitignored, generated pre-move) still names 7777 — the next `make dev` will refuse by name; delete it and the generator writes a fresh 7778 one. This is the designed stale-config behaviour, deliberately not auto-fixed by the executor (the file is operator-owned).

## Self-Check: PASSED

- All key-files exist; all acceptance-criteria greps re-run green
- 4 task commits present (`git log --grep=15-04`)

## Next Phase Readiness

Ready for 15-05 (simultaneity smoke: dev + installed running side by side). CONTRIBUTING.md and docs/testing.md still describe the old 7777 dev address — 15-05's doc task owns that update (flagged in this plan's read_first as deliberately not edited here).
