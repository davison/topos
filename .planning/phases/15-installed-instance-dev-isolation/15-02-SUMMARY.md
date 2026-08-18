---
phase: 15-installed-instance-dev-isolation
plan: 02
subsystem: infra
tags: [install, uninstall, github-releases, latest-tag, ci, assets]

requires:
  - phase: 15-installed-instance-dev-isolation
    provides: "15-01's install.sh (stage/verify/place), install-smoke fixture-release harness, make install target"
provides:
  - "Latest stable release resolution for a bare `make install` (INST-02): redirect-following resolver + effective-URL validator (https://github.com host, this repo's release-tag path, bare three-part semver — nightly/prerelease structurally excluded)"
  - "scripts/uninstall.sh + make uninstall (INST-05): closed removal set, non-recursive rmdir only, idempotent, live-kernel safe, no data-removal capability at all"
  - "topos-plugin-filesystem in release.yml and nightly.yml ASSETS (INST-01): published releases now carry the full five-plugin operator set"
  - "install.sh source-guard: sourcing defines functions only — the offline test seam"
affects: [15-03, 15-04, 15-05]

actuals:
  tokens: 9000
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "resolver/validator split: the network step reports, a pure function decides — refusals named per guard, testable offline via the source-guard seam"
    - "skip-loudly network case discipline (gdrive-rehearsal precedent) applied to the live latest-resolution check"

key-files:
  created:
    - scripts/uninstall.sh
  modified:
    - scripts/install.sh
    - scripts/install-smoke.sh
    - Makefile
    - .github/workflows/release.yml
    - .github/workflows/nightly.yml
    - docs/releasing.md
    - README.md

key-decisions:
  - "Latest resolution follows the releases/latest redirect (no token, no rate-limited API, no JSON parser) and validates the landing URL itself — the three-part tag-shape check is the script's own prerelease guard, enforced rather than trusted"
  - "uninstall offers no opt-in data-removal path at all; the absence of the capability is the guarantee"
  - "Older releases (<= v1.1.0) without the filesystem plugin install exactly what they published — a missing plugin in an old tag is correct behaviour, not an error"

patterns-established:
  - "validate_case table-driven shell assertion helper: expect accept/refuse + required fragment, driven against sourced functions"

requirements-completed: [INST-01, INST-02, INST-05]

coverage:
  - id: D1
    description: "make install with no version resolves and installs the latest published stable release; prerelease/nightly/off-host/off-repo URLs each refused by name"
    requirement: INST-02
    verification:
      - kind: integration
        ref: "make install-check ('latest-resolution validator (offline)' table: 1 accept, 6 named refusals; 'latest-resolution end to end (network)' resolved v1.1.0 live)"
        status: pass
  - id: D2
    description: "make uninstall removes exactly the installed artifacts, idempotently, under a live kernel, with the operator's config/index/plugin stores provably byte-identical"
    requirement: INST-05
    verification:
      - kind: integration
        ref: "make install-check ('uninstall data-safety cycle' seeded-tree digest comparison; 'idempotent uninstall'; 'uninstall leaves a foreign file'; 'uninstall under a live kernel')"
        status: pass
  - id: D3
    description: "Published releases carry topos-plugin-filesystem alongside the kernel and the four other operator-facing plugins"
    requirement: INST-01
    verification:
      - kind: other
        ref: "grep topos-plugin-filesystem in both workflows' ASSETS; YAML parse; MANIFEST_PLUGIN_BINARIES_PORTABLE already includes it (link-time manifest covers the published binary)"
        status: pass

duration: 40min
completed: 2026-08-19
---

# Phase 15 Plan 02: Complete the install surface Summary

`make install` with no argument now resolves the latest published stable release through a host/path/tag-shape-validated redirect (live-verified against github.com, resolving v1.1.0); `make uninstall` removes exactly what install placed with operator data proven byte-identical across a full cycle; and both release workflows now publish `topos-plugin-filesystem`, completing the operator-facing source set.

- Duration: ~40min (2026-08-18T22:47Z → 2026-08-18T23:27Z)
- Tasks: 3
- Files: 8 (1 created, 7 modified)

## Accomplishments

- `install.sh`: source-guard seam + `resolve_latest_effective_url` (curl -fsSLI, reports only) + `validate_latest_url` (refuses by name: empty URL, non-`https://github.com` scheme/host, non-`/davison/topos/releases/tag/` path, non-three-part-semver tag). Resolved tag printed before download.
- `make install` and `make install VERSION=<tag>` both first-class; comment documents no credential/token/GitHub CLI involvement.
- `scripts/uninstall.sh`: removes `$PREFIX/bin/topos` and `topos-plugin-*` files directly in `$PREFIX/lib/topos/plugins`; non-recursive `rmdir` of the two owned directories only; per-path removed/already-absent reporting; "nothing left to remove" on a second run; zero recursive removals and zero home/XDG references (negative greps pinned as acceptance criteria).
- `install-smoke.sh` grew from 7 to 13 cases: 4 uninstall cases (seeded home/XDG tree digest-identical across install+uninstall, idempotent second run, foreign file + directory survive, kernel keeps serving after uninstall) and 2 latest-resolution cases (offline validator table; live end-to-end with loud skip when offline).
- `ASSETS` widened with `plugins/topos-plugin-filesystem` in both `release.yml` and `nightly.yml` — the single-variable design meant one line per workflow; mock stays excluded with its comment intact. Confirmed before editing that `plugins-portable` builds the binary and `MANIFEST_PLUGIN_BINARIES_PORTABLE` covers it in the published kernel's link-time manifest.
- `docs/releasing.md` names the five published plugins and the older-release behaviour; README gains the `make install`/`make uninstall` quick-start linking `docs/install.md`, the five-binary manual path, and drops "not yet published".

## Task Commits

1. Task 1 (latest resolution): `5c8ae7b` feat(15-02) resolve the latest published stable release when no version is given
2. Task 2 (uninstall): `ef0ae08` feat(15-02) make uninstall — closed removal set with provable operator-data safety
3. Task 3 (publish filesystem): `4a3a181` feat(15-02) publish topos-plugin-filesystem so installs get the full source set

## Deviations from Plan

None - plan executed exactly as written.

## Verification Results

- `make install-check` — all 13 cases pass; the network case resolved v1.1.0 from the live endpoint
- `bash -n` on install.sh, install-smoke.sh, uninstall.sh — clean; uninstall.sh mode 0755
- `./scripts/check-doc-links.sh` — passes
- Both workflow files parse as valid YAML
- All negative greps green: no recursive rm in uninstall.sh, no XDG/config references, mock absent from ASSETS

## Issues Encountered

None.

## Self-Check: PASSED

- scripts/uninstall.sh exists, mode 0755; all modified files on disk
- 3 task commits present (`git log --grep=15-02`)
- All acceptance criteria re-run and passing at close-out

## Next Phase Readiness

Ready for 15-03 (Signal support for the installed instance without weakening the trust model). Note for 15-03: the installed kernel's manifest covers only published binaries — a locally built `topos-plugin-signal` needs its own trust path, which is exactly that plan's subject.
