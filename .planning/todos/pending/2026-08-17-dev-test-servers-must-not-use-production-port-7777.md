---
created: 2026-08-17T11:40:00.000Z
title: Dev/test servers must not use the production port 7777
area: kernel
severity: minor
resolves_phase: 15
files:
  - Makefile
  - config.dev.example.toml
  - config.example.toml
  - kernel/config/types.go
---

## Problem

Development and test servers currently use the same port (7777) as an
installed/in-use production topos server. When the real server is already
listening, a dev kernel either fails to bind or — the sharper edge — dev/e2e
tooling pointed at `localhost:7777` silently talks to the production instance
instead of the checkout under test.

This is the same isolation principle Plan 14-01 just enforced for config paths
(a dev kernel must never read, launch from, or write to the production config
or index): the listen port is the remaining shared surface between a dev
checkout and the installed server. Port 7777 is baked into the Makefile, both
example configs, and the kernel's config default (`kernel/config/types.go`);
the e2e harness and any dev scripts inherit it.

## Solution

Amend the dev/test port so it cannot clash with the installed server:

- Pick a distinct dev/test default (e.g. 7778, or an ephemeral port for e2e)
  and set it in `config.dev.example.toml` and wherever `make dev` / the e2e
  harness resolve their target URL.
- Leave 7777 as the production default in `config.example.toml` and
  `kernel/config/types.go`.
- Follows on naturally from 14-01's real-config/dev-config split — the dev
  config file is the right home for the override, so this may be a small
  change to the dev example config plus the Makefile/e2e base URL.
