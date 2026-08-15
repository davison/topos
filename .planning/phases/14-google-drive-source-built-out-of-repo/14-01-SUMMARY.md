---
phase: 14-google-drive-source-built-out-of-repo
plan: 01
subsystem: infra
tags: [cli-flags, makefile, dev-loop, config-resolution, go]

# Dependency graph
requires: []
provides:
  - "topos serve/sync --config flag + TOPOS_CONFIG env var, precedence flag > env > XDG"
  - "Per-checkout generated config.dev.toml (make dev-config), make dev wired to it via DEV_CONFIG"
  - "docs/testing.md split documentation, cross-referenced from config.example.toml"
affects: [14-02, 14-03, phase-14-uat]

# Actuals (#2632)
actuals:
  tokens: 4814
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Config-path resolution deliberately diverges from the executable-relative convention pluginsDir/externalPluginsDir use: --config/TOPOS_CONFIG resolve relative to process cwd, never os.Executable()'s directory, because `go run` builds into a throwaway temp dir"
    - "Makefile create-only generation (dev-config): a target that writes a per-checkout artifact only when absent, so operator edits and DEV_CONFIG overrides pointed at pre-existing files are never clobbered"

key-files:
  created:
    - cmd/topos/configpath_test.go
    - config.dev.example.toml
  modified:
    - cmd/topos/main.go
    - Makefile
    - .gitignore
    - config.example.toml
    - docs/testing.md

key-decisions:
  - "--config/TOPOS_CONFIG resolve verbatim against the process cwd, rejecting 14-PATTERNS.md's suggestion to reuse the executable-relative rule pluginsDir/externalPluginsDir apply — go run's own executable lives in a throwaway build dir that never holds a config file"
  - "dev-config is create-only: regeneration never overwrites an existing config.dev.toml, so hand edits and a DEV_CONFIG override pointed at the production config both survive every make dev"

patterns-established:
  - "Config-path resolver as its own testable function (resolveConfigPath), reachable from a table test with no config file, no index, no plugin subprocess"

requirements-completed: [SRC-05]

coverage:
  - id: D1
    description: "topos serve/sync accept --config <path> (and --config=<path>) and TOPOS_CONFIG, taking precedence over the unchanged XDG default chain"
    requirement: SRC-05
    verification:
      - kind: unit
        ref: "cmd/topos/configpath_test.go#TestResolveConfigPath_Precedence"
        status: pass
      - kind: manual_procedural
        ref: "go build -o /tmp/topos-cfgcheck ./cmd/topos && /tmp/topos-cfgcheck serve --config /nonexistent/dir/... (non-zero exit, names the given path)"
        status: pass
    human_judgment: false
  - id: D2
    description: "make dev generates and uses a per-checkout config.dev.toml (DEV_CONFIG), so a worktree kernel never touches the production config/index/plugins dir; make dev DEV_CONFIG=<path> restores the old behavior"
    requirement: SRC-05
    verification:
      - kind: integration
        ref: "make dev-config && grep -q \"$(pwd)/bin/plugins\" config.dev.toml && ! grep -q '@CHECKOUT@' config.dev.toml && git check-ignore -q config.dev.toml && make dev-check"
        status: pass
    human_judgment: false
  - id: D3
    description: "docs/testing.md documents the split (precedence, per-worktree consequence, DEV_CONFIG override, Playwright harness unaffected); config.example.toml cross-references config.dev.example.toml"
    verification:
      - kind: other
        ref: "make docs-check"
        status: pass
    human_judgment: false

duration: ~15min
completed: 2026-08-15
status: complete
---

# Phase 14 Plan 01: Real Config / Dev Config Split Summary

**`--config`/`TOPOS_CONFIG` flag+env on `topos serve`/`topos sync`, plus a per-checkout generated `config.dev.toml` that `make dev` now uses by default, so a worktree kernel never reads or writes the operator's production config.**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-08-15T23:29:00Z (approx)
- **Completed:** 2026-08-15T23:34:00Z
- **Tasks:** 3
- **Files modified:** 7 (2 new, 5 modified)

## Accomplishments
- `topos serve`/`topos sync` accept `--config <path>`/`--config=<path>`, honour `TOPOS_CONFIG`, and fall back to the unchanged XDG chain — precedence pinned by a 9-case Go table test (`cmd/topos/configpath_test.go`)
- `make dev` generates a per-checkout, gitignored `config.dev.toml` from a tracked `config.dev.example.toml` template (create-only — never overwrites an existing file) and passes it to the kernel via `--config`; `make dev DEV_CONFIG=<path>` is the documented escape hatch back to any other config, including the production one
- Verified live end to end: a kernel started with `--config` pointed at a temp config with a temp index path never touched the production `~/.local/share/topos/index.db` (mtime byte-identical before/after)
- `docs/testing.md` gained "The real config and the dev config", placed right after the four-gates section, naming the precedence chain, the Phase 13 UAT defect this closes, the `DEV_CONFIG` override, and why the Playwright harness needs none of this

## Task Commits

Each task was committed atomically:

1. **Task 1: End-to-end "run this kernel against that config file" — one flag, one path** - `eef30c4` (feat)
2. **Task 2: Per-checkout dev config — `make dev` stops reading the production config** - `9602262` (feat)
3. **Task 3: Document the split where the operator will look for it** - `bc91bc5` (docs)

_No TDD tasks in this plan — all three tasks were `type="tracer"`/`type="auto"`, each already includes tests where called for (Task 1's Go table test) rather than a separate RED/GREEN cycle._

## Files Created/Modified
- `cmd/topos/main.go` - `parseConfigFlag`, `resolveConfigPath`; `setup`/`runServe`/`runSync` now take an explicit `path`; usage string updated
- `cmd/topos/configpath_test.go` - table test pinning the 9-case precedence chain, hermetic via `t.Setenv`
- `Makefile` - `DEV_CONFIG` variable, `dev-config` target/prerequisite, `DEV_KERNEL_CMD` gains `--config $(DEV_CONFIG)`
- `config.dev.example.toml` - new tracked template (`[server]`/`[index]`/`[plugins]` only, `@CHECKOUT@` placeholder)
- `.gitignore` - `/config.dev.toml`
- `config.example.toml` - head-comment cross-reference to the dev-config template
- `docs/testing.md` - new "The real config and the dev config" section

## Decisions Made
- Rejected 14-PATTERNS.md's suggestion to resolve `--config`/`TOPOS_CONFIG` relative to `os.Executable()`'s directory (the convention `pluginsDir`/`externalPluginsDir` already apply to `cfg.Plugins.Dir`/`ExternalDir`). A `--config` value is typed on a command line by whoever runs the process; the dev loop starts the kernel via `go run`, whose own `os.Executable()` lives in a throwaway build directory that never holds a config file. Resolution stays verbatim against the process cwd, documented inline on `resolveConfigPath`.
- `dev-config` is strictly create-only: it never overwrites an existing `$(DEV_CONFIG)`, so an operator's hand-added `[sources.*]`/`[webspaces.*]` blocks survive every `make dev`, and pointing `DEV_CONFIG` at an already-existing file (e.g. the production config) is a safe no-op for the generator.

## Deviations from Plan

None - plan executed exactly as written. All three tasks' acceptance criteria were verified directly (see coverage block) rather than assumed.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- The precondition Phase 14's later plans (and their UAT) depend on is now live: any kernel built and run inside this checkout or any worktree of it resolves its own `config.dev.toml`, never the production config, index, or plugins directory.
- `make dev-check` and `make docs-check` both pass against the modified recipe/docs; `make test-portable` is unaffected (no other Go package changed behavior).
- The escape hatch (`make dev DEV_CONFIG=<path>`) is available for any later plan that legitimately needs to run the dev loop against a shared or production config.

---
*Phase: 14-google-drive-source-built-out-of-repo*
*Completed: 2026-08-15*
