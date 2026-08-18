---
phase: 15-installed-instance-dev-isolation
plan: 01
subsystem: infra
tags: [install, fhs, sha256, makefile, bash, plugin-resolution]

requires:
  - phase: 10-release
    provides: release.yml's checksums.txt asset manifest and flat-basename release asset layout
  - phase: 12-external-loading
    provides: link-time build manifest (VerifyTrustedBinary) the fixture kernel must embed
provides:
  - "scripts/install.sh: stage -> verify -> place release installer with checksums.txt-derived allowlisted asset manifest and atomic same-directory rename placement"
  - "make install VERSION=<tag> [PREFIX=...] and make install-check targets"
  - "resolvePluginsDir (cmd/topos): pure 3-branch plugin-dir resolution — absolute verbatim, checkout bin/plugins, installed lib/topos/plugins sibling gated on exe dir named bin"
  - "scripts/install-smoke.sh: hermetic offline gate — fixture release, installed-kernel-launches-installed-plugin proof, and five hardening cases"
  - "docs/install.md operator page (extensible; later plans append)"
affects: [15-02, 15-03, 15-04, 15-05]

actuals:
  tokens: 12000
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "TOPOS_RELEASE_BASE_URL as a test seam changing WHICH release is fetched, never which checks run (DEV_KERNEL_CMD precedent)"
    - "checksums.txt as the single asset manifest — derived, never a second hardcoded list"
    - "temp-name-in-destination-dir + mv -f rename for atomic, live-process-safe placement"

key-files:
  created:
    - scripts/install.sh
    - scripts/install-smoke.sh
    - cmd/topos/pluginsdir_test.go
    - docs/install.md
  modified:
    - Makefile
    - cmd/topos/main.go

key-decisions:
  - "Installed-layout probe is gated on the executable's directory being literally named `bin` and only ever applies to a relative [plugins] dir — existence probes, not a compiled-in prefix, let one published binary serve both checkout and installed layouts"
  - "Version passed as a make variable (make install VERSION=1.1.0), following the dev/e2e variable precedent over goal-name hacks"
  - "Writability of both destinations probed before any download work, so a doomed install wastes nothing and never needs privilege escalation"

patterns-established:
  - "install-smoke case style: named case sections, run_install captures rc+output, assert_prefix_untouched re-checks the defined failure state"

requirements-completed: [INST-01, INST-03]

coverage:
  - id: D1
    description: "make install places a release's kernel at <dir>/bin/topos and plugins at <dir>/lib/topos/plugins, SHA-256-verified against that release's own checksums.txt first"
    requirement: INST-01
    verification:
      - kind: integration
        ref: "make install-check (install-smoke.sh: 'install from fixture release' case)"
        status: pass
  - id: D2
    description: "Installed kernel discovers plugins from $PREFIX/lib/topos/plugins with the stock relative [plugins] dir — no config edit; checkout build unaffected; absolute dir verbatim"
    requirement: INST-03
    verification:
      - kind: unit
        ref: "cmd/topos/pluginsdir_test.go#TestResolvePluginsDir (7 subtests, mutation-verified)"
        status: pass
      - kind: integration
        ref: "make install-check ('installed kernel finds installed plugins' case asserts GET /api/sources shows mock launched+synced, $PREFIX/bin/plugins absent)"
        status: pass
  - id: D3
    description: "Checksum mismatch, traversal-shaped manifest, and unwritable prefix each refuse loudly by name leaving $PREFIX file-free; re-run is byte-identical; install over a running kernel succeeds via rename"
    requirement: INST-01
    verification:
      - kind: integration
        ref: "make install-check (corrupted asset / traversal-shaped manifest / unwritable prefix / idempotent re-run / live replacement cases)"
        status: pass

duration: 17min
completed: 2026-08-18
---

# Phase 15 Plan 01: Installed layout end-to-end Summary

`make install` downloads a release, verifies every byte against the release's own checksums.txt before placement, installs the FHS `$PREFIX/bin` + `$PREFIX/lib/topos/plugins` layout via atomic renames, and the installed kernel's new `resolvePluginsDir` probe finds those plugins with the stock relative config — proven end to end by a hermetic offline fixture-release gate (`make install-check`).

- Duration: 17min (2026-08-18T22:22:33Z → 2026-08-18T22:40:32Z)
- Tasks: 3 (tracer, TDD tests, hardening)
- Files: 6 (4 created, 2 modified)

## Accomplishments

- `scripts/install.sh`: preflight (tools + destination writability) → stage (mktemp, curl -fsSL) → verify (`sha256sum -c` over the release's own checksums.txt) → place (temp-name copy + `mv -f` rename in the destination directory). Asset list derived from checksums.txt's second column with a strict allowlist (`topos` | `plugins/[a-z0-9-]+`); anything else rejected by name.
- `resolvePluginsDir` split out of `pluginsDir` as a pure function; installed-layout sibling probe (`<parent-of-bin>/lib/topos/<dir>`) applies only when the exe dir is named `bin`, the relative join doesn't exist, and the sibling does. Doc comment records why (INST-01 layout, INST-03 no-config-edit).
- `cmd/topos/pluginsdir_test.go`: 7 real-directory subtests, mutation-verified (deleting the installed-layout branch fails the suite).
- `scripts/install-smoke.sh`: fixture release built locally (mock plugin + manifested kernel), installed via `file://` seam, then `$PREFIX/bin/topos serve` asserted over HTTP to launch and sync the installed mock plugin — plus corrupted-asset, traversal-manifest, unwritable-prefix, idempotent-re-run and live-replacement cases.
- Makefile: `PREFIX ?= /usr/local`, `install`, `install-check` (both in .PHONY, house comment style).
- `docs/install.md`: prerequisites, usage, PREFIX, exact paths written, verification semantics, failed-install and re-run sections.

## Task Commits

1. Task 1 (tracer): `cb79c2e` feat(15-01) install a verified release into PREFIX and resolve installed-layout plugins
2. Task 2 (TDD): `1019633` test(15-01) pin every resolvePluginsDir branch with real-directory table tests
3. Task 3 (hardening): `db8865c` feat(15-01) pin install refusal, idempotency and live-replacement behaviour hermetically

## Deviations from Plan

None - plan executed exactly as written.

## Verification Results

- `make install-check` — all 7 cases pass (happy install, installed-kernel launch, corrupted asset, traversal manifest, unwritable prefix, idempotent re-run, live replacement)
- `go test ./cmd/topos/ -run TestResolvePluginsDir -v` — 7 subtests pass; mutation check observed failing then restored
- `make test-portable` — all 17 module test suites pass
- `./scripts/check-doc-links.sh` — passes with docs/install.md present

## Issues Encountered

None.

## Self-Check: PASSED

- All key-files exist on disk
- 3 task commits present (`git log --grep=15-01`)
- All acceptance criteria re-run and passing at close-out

## Next Phase Readiness

Ready for 15-02 (latest-release resolution, uninstall, release.yml/nightly alignment). docs/install.md and install-smoke.sh were written to be appended to, not rewritten.
