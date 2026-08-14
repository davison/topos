---
created: 2026-08-14T21:55:00.000Z
title: Separate the real (production) config from dev-work configs
area: kernel
severity: major
files:
  - cmd/topos/main.go
  - config.example.toml
  - Makefile
  - docs/testing.md
---

## Problem

`cmd/topos` resolves exactly one config location: `$XDG_CONFIG_HOME/topos/config.toml` (falling back to `~/.config/topos/config.toml`). As the user starts running topos "for real", that single shared path means every dev-built kernel — including ones built inside GSD executor worktrees — reads and acts on the *production* config: its `plugins.dir` (absolute path to the main checkout's `bin/plugins`), its index path, its source credentials.

This bit concretely during Phase 13 UAT (13-04, 2026-08-14): a kernel built in an agent worktree read the real config, discovered the *main checkout's* plugin binaries, and the D-12/D-13 build-manifest gate correctly refused all six trusted plugins ("plugin launch refused: trusted binary not verified by the build manifest") because those binaries came from a different `make build` than the kernel that verified them. All source chips went yellow and sync dispatch spammed `syncer: unknown source`. The gate behaved as designed; the config sharing is the actual defect. Raised by the user at the 13-04 checkpoint: "I need to be able to separate my real config (as I start to use this properly) from any dev work configs. This issue here is a symptom of that problem."

## Solution

Give dev/test runs a first-class way to run against a non-production config, e.g.:

- A `--config <path>` flag (and/or `TOPOS_CONFIG` env var) on `topos serve`, taking precedence over the XDG default — the standard escape hatch, and enough on its own to unblock worktree UAT.
- A dev convention wired into the Makefile (`make dev` / worktree runs point at a repo-local `config.dev.toml` or the worktree's own generated config, with `plugins.dir` left relative so it resolves next to the built executable — the relative-resolution logic in `pluginsDir()` already exists and does the right thing).
- Document the split (real config vs dev config) in README/docs/testing.md so future UAT instructions don't ask the operator to hand-edit their production config and restore it afterwards.

Note: `plugins.dir` in the user's real config is currently an *absolute* path into the repo checkout (`/home/darren/projects/davison/topos/bin/plugins`) — once a real install location exists (installed binary + plugins dir outside the repo), the production config should point there instead, which makes the repo checkout purely a dev tree and removes the remaining coupling.
