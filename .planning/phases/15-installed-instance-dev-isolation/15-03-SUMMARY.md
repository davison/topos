---
phase: 15-installed-instance-dev-isolation
plan: 03
subsystem: infra
tags: [signal, cgo, external-tier, trust-model, install, sqlcipher]

requires:
  - phase: 15-installed-instance-dev-isolation
    provides: "15-01/15-02's install surface, install-smoke harness, uninstall's closed-removal discipline"
  - phase: 11-external-loading
    provides: external plugin tier, untrusted-add consent-and-pin flow, per-launch SHA-256 pin verification
provides:
  - "make install-signal (INST-04): builds the cgo Signal plugin through the single existing `signal` definition and places it atomically in the installed instance's external plugin directory"
  - "make uninstall-signal: removes exactly that one binary, nothing else, idempotent"
  - "Behavioural proof the base `make install` is download-and-copy only (toolchain tripwire smoke case)"
  - "docs/install.md Signal section + docs/plugins/signal.md step-by-step naming make install-signal, cross-linked"
affects: [15-04, 15-05]

actuals:
  tokens: 7000
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Build-as-Makefile-prerequisite: install-signal chains the existing `signal` target so cgo build flags stay one-place-only; the script only resolves, places, and prints"
    - "Toolchain tripwire: failing shims first on PATH prove a no-toolchain property behaviourally, not textually"

key-files:
  created:
    - scripts/install-signal.sh
  modified:
    - Makefile
    - scripts/install-smoke.sh
    - docs/install.md
    - docs/plugins/signal.md

key-decisions:
  - "Destination is the external plugin tier, never $PREFIX/lib/topos/plugins — a locally built binary is absent from the released kernel's link-time manifest and would be refused (manifest_unverified); the installer refuses a trusted-dir-shaped destination outright rather than letting the kernel refuse later"
  - "Destination default reproduces the kernel's own defaultExternalPluginsDir Linux branch exactly; TOPOS_EXTERNAL_PLUGINS_DIR is the single override"
  - "uninstall-signal lives behind a --uninstall mode flag in install-signal.sh so the destination resolution is never duplicated"

patterns-established: []

requirements-completed: [INST-04]

coverage:
  - id: D1
    description: "make install-signal builds locally via the single signal build definition and places the binary where the installed kernel can launch it (external tier), printing the resolved path and one-time consent steps"
    requirement: INST-04
    verification:
      - kind: integration
        ref: "live run: TOPOS_EXTERNAL_PLUGINS_DIR override placed a 0755 topos-plugin-signal; make -n install-signal shows the `signal` definition then the script; trusted-dir destination refused rc=1 naming manifest_unverified"
        status: pass
  - id: D2
    description: "Base make install completes with no compiler or language toolchain available"
    requirement: INST-04
    verification:
      - kind: integration
        ref: "make install-check ('toolchain tripwire' case: go/cc/gcc/clang/npm/node failing shims first on PATH; install exits 0, marker never created)"
        status: pass
  - id: D3
    description: "make uninstall-signal removes the Signal binary and nothing else; make uninstall leaves it alone (outside $PREFIX)"
    verification:
      - kind: integration
        ref: "make install-check ('Signal removal' case: planted unrelated file digest-identical, directory survives, second run exits 0 already-absent)"
        status: pass
  - id: D4
    description: "On the operator's machine, make install-signal + one-time untrusted-add consent results in the installed instance syncing Signal with the untrusted badge"
    requirement: INST-04
    human_judgment: true
    rationale: "Live end-of-phase check on the operator's real installed instance and real Signal Desktop database — plan's own <human-check> defers this to end-of-phase UAT"

duration: 8min
completed: 2026-08-19
---

# Phase 15 Plan 03: Signal for the installed instance Summary

`make install-signal` builds the cgo Signal plugin through the repository's single `signal` build definition and places it atomically in the installed instance's external plugin directory — the trust model's supported path for a locally built binary — while the toolchain-tripwire smoke case proves the base `make install` never touches a compiler, and `make uninstall-signal` gives the binary a closed, idempotent removal path.

- Duration: ~8min (2026-08-18T23:27Z → 2026-08-18T23:35Z)
- Tasks: 3
- Files: 5 (1 created, 4 modified)

## Accomplishments

- `scripts/install-signal.sh`: destination resolution reproducing `defaultExternalPluginsDir`'s Linux branch (XDG_DATA_HOME set-and-non-empty → `$XDG_DATA_HOME/topos/plugins-external`, else `~/.local/share/topos/plugins-external`), `TOPOS_EXTERNAL_PLUGINS_DIR` override, trusted-dir-shape refusal naming the manifest_unverified consequence, atomic temp-name+`mv -f` placement (live-kernel safe), and printed guidance matching docs/plugins/signal.md's recorded flow (untrusted by construction → consent-and-pin → rebuild means re-accept → external_dir override hint).
- `make install-signal: signal` — the build is a prerequisite on the one existing cgo build definition; `make -n` shows the real definition being reached. `make uninstall-signal` → `--uninstall` mode removing exactly one path.
- Toolchain tripwire case: `go`, `cc`, `gcc`, `clang`, `npm`, `node` failing shims first on PATH; fixture install exits 0 with the marker file never created and the kernel placed.
- Signal-removal case: unrelated planted file and the directory itself survive digest-identical; second run exits 0 reporting already-absent.
- Docs: `docs/install.md` "Signal on an installed instance" section (constraint stated as deliberate trust-model consequence); `docs/plugins/signal.md` step-by-step leads with `make install-signal`, checkout longhand preserved, `manifest_unverified` explanation untouched; cross-linked both directions.

## Task Commits

1. Task 1 (install-signal): `7fd6ab1` feat(15-03) make install-signal — build the cgo plugin locally, place it in the external tier
2. Task 2 (tripwire + removal): `f60105c` feat(15-03) prove the base install toolchain-free and give Signal a closed removal path
3. Task 3 (docs): `8da0af0` docs(15-03) one consistent account of Signal on an installed instance

## Deviations from Plan

None - plan executed exactly as written.

## Verification Results

- `make install-check` — all 15 cases pass (toolchain tripwire and Signal removal included)
- Live placement run: 0755 binary in override dir; trusted-dir refusal rc=1 with named reason
- `make docs-check` — passes; all cross-links resolve
- Negative greps: no recursive rm in install-signal.sh; `install-signal`/`uninstall-signal` each exactly once in Makefile + .PHONY

## Issues Encountered

None.

## Self-Check: PASSED

- All key-files exist on disk; scripts mode 0755
- 3 task commits present (`git log --grep=15-03`)
- All acceptance criteria re-run and passing at close-out

## Next Phase Readiness

Ready for 15-04 (dev isolation as a mechanical gate — topos-devguard). The plan's `<human-check>` (live `make install-signal` + consent flow on the operator's machine) is deferred to end-of-phase UAT and recorded as human_judgment coverage.
