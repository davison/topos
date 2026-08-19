---
phase: 15-installed-instance-dev-isolation
verified: 2026-08-19T01:30:00Z
status: gaps_found
score: 7/8 requirement-level truths verified
behavior_unverified: 0
overrides_applied: 0
gaps:
  - truth: "ISOL-01 — a dev run refuses to start when any writable path its config declares (including a source's own store path) resolves inside the topos config or state root"
    status: failed
    reason: >
      Reproduced directly against the current binary (not just the PLAN's own test
      matrix): cmd/topos-devguard resolves a RELATIVE `[sources.*] path` against the
      config file's own directory (`configDir`), but the real kernel launches every
      plugin subprocess with no `cmd.Dir` override, so the plugin actually resolves a
      relative `path` against the kernel process's own working directory at launch —
      NOT the config file's directory. When a dev config lives outside the checkout
      (the documented, supported `make dev DEV_CONFIG=<path>` override) and declares a
      relative source `path`, and the guard/kernel's real launch cwd happens to sit
      inside the topos-owned state root, the guard reports "OK" (exit 0) for a source
      store the real subprocess would actually open inside the protected state root —
      a false clear in the exact defect class ISOL-01 exists to catch. Confirmed live:
      `go build ./cmd/topos-devguard` then invoking the binary from a cwd inside a
      simulated XDG state root, against a config in a different directory declaring
      `path = "relative-source-store"`, printed `devguard: OK` and exited 0, even
      though a real subprocess launched from that same cwd would open
      `<state-root>/relative-source-store`. This is the same defect the phase's own
      code review flagged (15-REVIEW.md WR-02, filed 2026-08-19T00:17:32Z) and it
      remains unfixed — no commit after `0482cc6` (the review commit) touches
      `cmd/topos-devguard/main.go`.
      Scope: this does NOT affect the default/documented dev flow — the generated
      `config.dev.toml` (from `config.dev.example.toml`'s `@CHECKOUT@`-prefixed
      convention) always uses absolute source paths, so `make dev` with the generated
      config is not exposed. It requires an operator to combine `DEV_CONFIG=<path
      outside the checkout>` with a hand-written relative `[sources.*] path` — a
      documented but non-default combination (`docs/testing.md`, "The real config and
      the dev config").
    artifacts:
      - path: cmd/topos-devguard/main.go
        issue: >
          Candidate construction (`candidates = append(candidates, candidate{...,
          absolutize(src.Path, configDir)})`, ~line 214-222) uses `configDir` as the
          base for a relative source `path`, but `kernel/pluginhost.host.go`'s
          `exec.Command(binPath)` sets no `cmd.Dir`, so the real subprocess resolves a
          relative source `path` against the kernel process's own cwd, not the config
          file's directory. No test in devguard_test.go exercises a relative
          `[sources.*] path` from a config directory that differs from the process cwd.
    missing:
      - >
        Either: (a) refuse outright (independent of containment) whenever a source's
        `path` is relative, since the guard cannot correctly evaluate it without
        knowing the real launch cwd — "cannot verify, refuse by name" rather than
        silently checking an approximation; or (b) resolve relative source paths
        against the guard's own `os.Getwd()` (matching the fact that `make dev`
        always invokes both the guard and the kernel from the same cwd), with a
        comment next to `expandHome` recording the choice — mirroring the plugin-vs-
        kernel resolution split already documented there.
      - A devguard_test.go case building a relative `[sources.*] path` under a config
        directory that differs from the process's cwd, asserting a violation.
  - truth: "docs/install.md accurately documents the shipped `make install` (no-argument, latest-release) path (supports INST-02/INST-03 discoverability)"
    status: failed
    reason: >
      `docs/install.md`'s "Installing a release" section (lines 18-26) still reads
      `make install VERSION=1.1.0` as the only form and states "`VERSION` ... is
      required — there is no implicit 'latest' yet." This directly contradicts the
      shipped, tested, and verified INST-02 feature: `scripts/install.sh`'s own header
      states latest-resolution exists, the Makefile's `install` target comment lists
      the no-argument form as first-class, `README.md` correctly documents `make
      install` (no VERSION) as resolving the latest stable release, and
      `install-smoke.sh` has dedicated "latest-resolution validator (offline)" and
      "latest-resolution end to end (network)" cases that pass (confirmed live:
      resolved v1.1.0 from the real github.com/davison/topos redirect during this
      verification run). This is the same defect the phase's own code review flagged
      (15-REVIEW.md WR-01) and it remains unfixed — no commit after the review commit
      touches `docs/install.md`'s "Installing a release" section. `docs/install.md` is
      the document every other doc in this phase points readers at "for the full
      treatment" (README.md, docs/plugins/signal.md), so an operator who only reads it
      will believe `VERSION` is mandatory and never discover the no-argument path this
      phase built.
    artifacts:
      - path: docs/install.md
        issue: >
          Lines 18-26 ("Installing a release" section) state VERSION is required with
          no implicit latest, contradicting the shipped and tested no-argument
          latest-release resolution.
    missing:
      - >
        Update the "Installing a release" section to document both forms (`make
        install` for latest, `make install VERSION=<tag>` for a pinned release),
        matching README.md's phrasing — the fix is purely textual, no code change
        required.
---

# Phase 15: Installed Instance & Dev Isolation Verification Report

**Phase Goal:** The operator runs topos daily from installed release artifacts while
developing the next milestone from the checkout — the two instances can never clash
on port, config, or state. Phase 15 delivers `make install` / `install-signal` /
`uninstall` from published release artifacts, plus full dev-side port, config, and
state isolation from the checkout.
**Verified:** 2026-08-19
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (requirement-level, cross-referenced to ROADMAP.md success criteria)

| # | Truth (Requirement) | Status | Evidence |
|---|---|---|---|
| 1 | `make install [VERSION]` places kernel at `$PREFIX/bin`, plugins at `$PREFIX/lib/topos/plugins`, SHA-256-verified against the release's own `checksums.txt` first; download-and-copy only, no toolchain (INST-01) | ✓ VERIFIED | `make install-check` (re-run live, this verification): 15/15 cases pass — fixture install, checksum verification, corrupted-asset refusal, traversal-shaped-manifest refusal, unwritable-prefix refusal, idempotent re-run, live replacement, toolchain tripwire (go/cc/gcc/clang/npm/node shims first on PATH, install still exits 0). Both `release.yml` and `nightly.yml` publish `topos-plugin-filesystem` alongside the other 4 operator-facing plugins (grep-confirmed). |
| 2 | `make install` with no version resolves and installs the **latest published stable** release; never a prerelease/nightly; host- and path-validated (INST-02) | ✓ VERIFIED (code) / ⚠️ doc gap | `install-smoke.sh`'s "latest-resolution validator (offline)" and "latest-resolution end to end (network)" cases pass — re-run live during this verification, resolved `v1.1.0` from the real `github.com/davison/topos/releases/latest` redirect. **But** `docs/install.md`'s own "Installing a release" section still says this doesn't exist (see Gap 2 below) — the feature is real and tested, the primary operator doc for it is wrong. |
| 3 | `topos` from PATH serves on 7777, uses home/XDG config and state unchanged, discovers plugins from the installed plugins dir with no config edit; an operator's live checkout instance migrates without touching config/index (INST-03) | ✓ VERIFIED | `resolvePluginsDir`: 7/7 subtests pass (re-run live) including installed-layout-sibling-gated-on-`bin`-dirname, checkout-layout-wins, absolute-verbatim, adjacency, determinism. `install-smoke.sh`'s "installed kernel finds installed plugins (stock config)" case boots `$PREFIX/bin/topos serve` and asserts `GET /api/sources` shows the mock plugin launched via `$PREFIX/lib/topos/plugins`, `$PREFIX/bin/plugins` absent. `docs/install.md`'s migration runbook (`## Migrating from a checkout build to an installed instance`) is complete and accurate: preservation rationale, the absolute-`[plugins] dir` edge case with both remedies, verification checklist, back-out via `make uninstall`. |
| 4 | `make install-signal` builds the cgo Signal plugin through the repo's single `signal` definition and places it in the installed instance's **external** plugin dir (never trusted — refused by build-manifest verification), prints destination + one-time consent steps; base `make install` needs no toolchain (INST-04) | ✓ VERIFIED | Re-run live during this verification: `TOPOS_EXTERNAL_PLUGINS_DIR=<tmp> make install-signal` built the real cgo plugin (system SQLCipher present), placed a 0755 `topos-plugin-signal`, and printed the resolved destination plus the untrusted-add consent-and-pin steps matching `docs/plugins/signal.md` verbatim. Pointing the override at a trusted-dir-shaped path (`.../prefix/lib/topos/plugins`) refused with exit 1 naming `manifest_unverified`, confirmed live. `install-signal.sh --uninstall` removed only the one binary, leaving an unrelated marker file and the directory untouched, confirmed live. `install-smoke.sh`'s toolchain-tripwire case (base install completes with failing go/cc/gcc/clang/npm/node shims first on PATH) passed live. |
| 5 | `make uninstall` removes exactly `$PREFIX/bin/topos` and `topos-plugin-*` under `$PREFIX/lib/topos/plugins`, non-recursive dir cleanup only, config/index/plugin stores byte-identical, idempotent, live-kernel-safe (INST-05) | ✓ VERIFIED | `scripts/uninstall.sh`: zero recursive `rm` (`grep -cE 'rm[[:space:]]+-[a-zA-Z]*[rR]'` = 0), zero XDG/home references. `install-smoke.sh`'s data-safety cycle, idempotent-uninstall, foreign-file, and live-kernel cases all passed on this verification's live `make install-check` run. |
| 6 | A dev run refuses to start when any writable path its config declares — config file, index, either plugin dir (including the omitted-`external_dir` default), or any source's own store path — resolves inside the topos config or state root, in one deterministic pass; dev port moves off 7777 (ISOL-01, ISOL-02) | ✗ **GAP** (see Gap 1) | `go test ./cmd/topos-devguard/ -v`: 13/13 subtests pass (re-run live). `make dev-check`: 6/6 cases pass live (isolation refusal, escape hatch, stale-port fast-fail). Port move confirmed: `DEV_PORT ?= 7778` in Makefile, `listen = "127.0.0.1:7778"` in `config.dev.example.toml`, Vite proxy target `7778`; production `7777` unchanged in `config.example.toml`/`kernel/config/types.go`. **However**, independently reproduced during this verification: the guard's containment check for a relative `[sources.*] path` uses the config file's own directory as the resolution base, while the real kernel launches every plugin subprocess with no `cmd.Dir` override (resolves against the kernel process's actual cwd) — a false clear for the documented `DEV_CONFIG=<path-outside-checkout>` + relative-source-path combination. See Gap 1 for full reproduction. This is the same defect the phase's own code review flagged (WR-02) and it was never fixed. |
| 7 | The installed instance and a dev instance run simultaneously, neither able to see or disturb the other's data, pinned by a committed gate (ISOL-03) | ✓ VERIFIED | `make isolation-check`: 3/3 cases pass, re-run live — static port-contract assertion (production 7777 ≠ dev 7778, read from source, no binding), byte-unchanged installed tree across a driven dev sync+mark-write (digest manifest comparison), concurrent-and-independent (each instance answers only its own seeded webspace, installed file-set unchanged during concurrent dev activity). Real-port safety baseline captured and re-asserted after every case. The gate's dev-shaped instance uses the standard generated-config shape (absolute paths), so it is not itself exposed to Gap 1's relative-path edge case — the gate proves the primary flow, not the edge case. |
| 8 | Requirements traceability: all 8 requirement IDs (INST-01..05, ISOL-01..03) map to Phase 15 plans and are accounted for | ✓ VERIFIED | Every ID appears in exactly one plan's `requirements:` frontmatter (15-01: INST-01/03; 15-02: INST-01/02/05; 15-03: INST-04; 15-04: ISOL-01/02; 15-05: INST-03/ISOL-03); REQUIREMENTS.md's traceability table lists all 8 mapped to Phase 15 with no unmapped rows. No orphaned requirements. |

**Score:** 7/8 truths verified (1 gap: row 6 / ISOL-01's general guarantee, in a documented but non-default configuration). Row 2 also carries an unfixed documentation-accuracy defect (Gap 2) that does not itself invalidate the functional truth but misleads an operator reading the canonical doc.

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `scripts/install.sh` | stage→verify→place installer | ✓ VERIFIED | Exists, mode 0755, `bash -n` clean, exercised live |
| `scripts/install-smoke.sh` | hermetic install gate | ✓ VERIFIED | Exists, mode 0755, 15 cases pass live |
| `scripts/uninstall.sh` | closed removal set | ✓ VERIFIED | Exists, mode 0755, zero recursive-rm greps |
| `scripts/install-signal.sh` | cgo Signal build+place, `--uninstall` mode | ✓ VERIFIED | Exists, mode 0755, exercised live (build, place, refuse-trusted-dir, uninstall) |
| `scripts/simultaneity-smoke.sh` | ISOL-03 committed gate | ✓ VERIFIED | Exists, mode 0755, 3 cases pass live |
| `scripts/smoke-lib.sh` | shared fixture-release + free-port helpers | ✓ VERIFIED | Exists, mode 0755, sourced by both smokes |
| `cmd/topos/pluginsdir_test.go` | `resolvePluginsDir` branch coverage | ✓ VERIFIED | 7 subtests pass live |
| `cmd/topos-devguard/main.go` + `devguard_test.go` | isolation guard | ⚠️ HOLLOW (partial) | Exists, wired, 13/13 shipped tests pass — but the relative-source-path resolution base is provably wrong for the documented `DEV_CONFIG` + relative-path combination (Gap 1); no test covers that combination |
| `docs/install.md` | operator install/uninstall/signal/migration doc | ⚠️ Present, one section stale | Exists, all required headings present, migration runbook and Signal section accurate and cross-linked; "Installing a release" section contradicts the shipped INST-02 feature (Gap 2) |
| `docs/testing.md` | names all gates | ✓ VERIFIED | "The seven gates" section; heading count matches subsection count (7); `install-check`/`isolation-check` subsections present and accurate |
| Makefile targets: `install`, `install-check`, `install-signal`, `uninstall`, `uninstall-signal`, `isolation-check` | registered, single-definition | ✓ VERIFIED | All present exactly once, all in `.PHONY` |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `resolvePluginsDir` (cmd/topos/main.go) | `$PREFIX/bin`+`$PREFIX/lib/topos/plugins` layout (scripts/install.sh) | existence probe gated on exe dir named `bin` | ✓ WIRED | `install-smoke.sh` "installed kernel finds installed plugins" case proves this end to end live |
| `scripts/install.sh` asset list | release's own `checksums.txt` | derived, never hardcoded | ✓ WIRED | Confirmed via source read: `sha256sum`'s 2nd column drives the manifest |
| `release.yml`/`nightly.yml` `ASSETS` | published kernel's link-time build manifest | `MANIFEST_PLUGIN_BINARIES_PORTABLE` | ✓ WIRED | `topos-plugin-filesystem` present in both; portable build's manifest already covers it (per 15-02 SUMMARY, confirmed before the workflow edit) |
| `cmd/topos-devguard` | `kernel/config`'s real loader | `config.NewStore` | ✓ WIRED | `grep -c 'config.NewStore' cmd/topos-devguard/main.go` = 1 — single parser, confirmed |
| `cmd/topos-devguard`'s root derivation | `cmd/topos`'s `configPath`/`defaultExternalPluginsDir` | mirrored env-var + fallback logic | ⚠️ PARTIAL | Config/state root derivation matches; **source-path candidate base does not match the kernel's actual subprocess-launch resolution** (Gap 1) |
| `DEV_PORT` (Makefile) | `[server] listen` (config.dev.example.toml) | `[server] listen` (config.dev.example.toml) | `web/vite.config.ts` proxy target | ✓ WIRED | All three read `7778`; production `7777` unchanged in `config.example.toml`/`kernel/config/types.go` |
| `scripts/simultaneity-smoke.sh` | `scripts/install.sh` + `cmd/topos-devguard` | drives the real scripts, no reimplementation | ✓ WIRED | Confirmed by reading the script: builds a fixture release via the real installer, validates the dev-shaped config through the real guard binary |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| INST-01 | 15-01, 15-02 | Install kernel+plugins to `$PREFIX`, checksum-verified | ✓ SATISFIED | Row 1 above |
| INST-02 | 15-02 | `make install` with no version = latest release | ✓ SATISFIED (code); doc gap noted | Row 2 above |
| INST-03 | 15-01, 15-05 | `topos` from PATH, home/XDG, plugin discovery, migration | ✓ SATISFIED | Row 3 above |
| INST-04 | 15-03 | `make install-signal` builds+places to external tier | ✓ SATISFIED | Row 4 above |
| INST-05 | 15-02 | `make uninstall` — exact removal, data untouched | ✓ SATISFIED | Row 5 above |
| ISOL-01 | 15-04 | Dev isolation refusal for all writable paths | ✗ **BLOCKED** (partial) | Row 6 above / Gap 1 |
| ISOL-02 | 15-04 | Non-7777 dev port | ✓ SATISFIED | Row 6 above (port-move sub-claim) |
| ISOL-03 | 15-05 | Simultaneous run, no clash | ✓ SATISFIED | Row 7 above |

No orphaned requirements — REQUIREMENTS.md's 8-row traceability table exactly matches the union of all 5 plans' `requirements:` frontmatter.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `cmd/topos-devguard/main.go` | ~214-222 | Wrong resolution base for a relative `[sources.*] path` candidate (Gap 1) | 🛑 Blocker (for the ISOL-01 general guarantee, narrow scope) | False clear (guard exits 0) for a source store that the real subprocess would resolve inside the protected state root, when `DEV_CONFIG` names a config outside the checkout with a relative source path |
| `docs/install.md` | 18-26 | Stale "VERSION is required" text contradicting the shipped no-argument latest-release path (Gap 2) | ⚠️ Warning | Misleads an operator reading the canonical install doc; does not affect the underlying feature, which works and is tested |
| `cmd/topos-devguard/main.go` | 99-110 (`containedIn`) | No symlink resolution (`filepath.EvalSymlinks`) before comparing cleaned/absolutized paths (code review IN-01) | ℹ️ Info | A stowed dotfile/bind-mount could produce a false clear or false refusal; the doc comment only promises "not string prefixes," so not a broken promise, but a real residual gap |
| `scripts/install.sh` | 62, 108, 117 | `TOPOS_RELEASE_BASE_URL` default string duplicated 3x (code review IN-02) | ℹ️ Info | Drift risk on a future repo-rename edit; all 3 currently agree |
| `scripts/smoke-lib.sh` | 11-19 (`smoke_free_port`) | TOCTOU window between ephemeral-port allocation and re-bind (code review IN-03) | ℹ️ Info | Inherent to "ask-then-release" port allocation; unlikely to bite on single-user dev boxes |
| `scripts/install.sh` | 173 | Plugin-name allowlist regex permits a leading/trailing hyphen (code review IN-04) | ℹ️ Info | No exploitable path found (values always concatenated onto an absolute prefix before use) |

No debt markers (`TBD`/`FIXME`/`XXX`) found in any file this phase touched. The `mktemp ... XXXXXX` template strings in `install.sh`/`install-signal.sh` are `mktemp` placeholder syntax, not TODO markers.

### Behavioral Spot-Checks (re-run live during this verification)

| Behavior | Command | Result | Status |
|---|---|---|---|
| `resolvePluginsDir` branch coverage | `go test ./cmd/topos/ -run TestResolvePluginsDir -v` | 7/7 subtests pass | ✓ PASS |
| `topos-devguard` behavior coverage | `go test ./cmd/topos-devguard/ -v` | 13/13 subtests pass | ✓ PASS |
| Full install gate | `make install-check` | 15/15 cases pass | ✓ PASS |
| Dev-guard gate | `make dev-check` | 6/6 cases pass | ✓ PASS |
| Simultaneity gate | `make isolation-check` | 3/3 cases pass | ✓ PASS |
| Doc links | `make docs-check` | 46 links across 21 files, all resolve | ✓ PASS |
| Workspace build+test | `make test-portable` | All modules green | ✓ PASS |
| `make install-signal` (live, real cgo build, system SQLCipher present) | `TOPOS_EXTERNAL_PLUGINS_DIR=<tmp> make install-signal` | 0755 `topos-plugin-signal` placed, correct guidance printed | ✓ PASS |
| `install-signal.sh` trusted-dir refusal | override pointed at `.../prefix/lib/topos/plugins` | exit 1, names `manifest_unverified` | ✓ PASS |
| `install-signal.sh --uninstall` | plant marker + binary, remove | binary gone, marker + dir untouched | ✓ PASS |
| Devguard relative-source-path false clear | binary invoked from a cwd inside a simulated state root, config elsewhere with relative source path | `devguard: OK`, exit 0 (expected: violation) | ✗ **FAIL** — Gap 1 |

### Human Verification Required

Two items are recorded in the plan SUMMARYs as deliberately deferred to end-of-phase human judgment (`human_judgment: true`); they remain open regardless of the gaps above and should still be exercised before milestone close:

#### 1. Live Signal install + consent flow (15-03)

**Test:** With the installed instance replaced by `make install`, run `make install-signal`, restart the installed kernel, and add the Signal source through the app's untrusted-add consent flow.
**Expected:** The source syncs and the chip shows the untrusted badge rather than a launch failure.
**Why human:** Requires the operator's real Signal Desktop database and a real UI consent interaction — not reproducible hermetically.

#### 2. Live migration + simultaneity UAT (15-05, the roadmap's stated end-to-end proof)

**Test:** Follow the roadmap's 5-step UAT: note the running checkout instance's webspace/mark/health state → `make install` → start `topos` from PATH → confirm identical state on 7777 → run `make dev` alongside and confirm 7778 comes up with neither instance showing the other's data → `make uninstall` and confirm config/index/marks survive.
**Expected:** All 5 steps succeed with no data loss and no visible clash.
**Why human:** Requires the operator's real installed instance, real personal data (webspaces, marks, WhatsApp/Signal links), and real concurrent process observation — not reproducible hermetically. Note: this UAT flow itself uses the standard generated dev config (absolute paths), so it is not expected to trigger Gap 1.

### Gaps Summary

Both gaps are the same two defects the phase's own code review (`15-REVIEW.md`, committed as `0482cc6`, the last commit in the phase) already identified as WR-01 and WR-02 — neither was fixed afterward. This verification independently reproduced both against the current codebase rather than taking the review's word for it:

1. **Gap 1 (ISOL-01, narrow but real):** `cmd/topos-devguard` computes a relative `[sources.*] path`'s containment candidate against the config file's directory, but the kernel launches plugin subprocesses with no `cmd.Dir` override, so the real resolution base is the kernel process's own cwd. Independently reproduced live: the guard reports no violation for a source path that would, once resolved by the real subprocess, land inside the protected state root — when `DEV_CONFIG` points at a config outside the checkout and that config uses a relative source path. The documented, generated `config.dev.toml` flow (`make dev` with no `DEV_CONFIG` override) is not exposed, because the template always emits absolute `@CHECKOUT@`-prefixed paths — so the primary supported flow is genuinely safe. The gap is real for the secondary, documented `DEV_CONFIG=<path>` override combined with a relative source path, which is exactly the class of mistake ISOL-01 exists to catch and currently does not, in that combination.
2. **Gap 2 (documentation accuracy, INST-02/INST-03):** `docs/install.md`'s "Installing a release" section still asserts `VERSION` is required and "there is no implicit 'latest' yet," directly contradicting the shipped, tested, and (during this verification) live-confirmed no-argument latest-release resolution. This is a text-only fix with no code risk.

Recommendation: both are small, well-scoped fixes (a base-directory correction plus a test case for Gap 1; a doc-text edit for Gap 2) — appropriate for a short closure plan before milestone completion, given this is the milestone's single phase.

---

_Verified: 2026-08-19_
_Verifier: Claude (gsd-verifier)_
