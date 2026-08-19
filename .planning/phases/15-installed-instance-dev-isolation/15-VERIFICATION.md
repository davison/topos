---
phase: 15-installed-instance-dev-isolation
verified: 2026-08-19T02:05:00Z
status: human_needed
score: 8/8 requirement-level truths verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 7/8
  gaps_closed:
    - "ISOL-01 — devguard's relative [sources.*] path resolution base (Gap 1)"
    - "docs/install.md 'Installing a release' section contradicting the shipped no-argument latest-release path (Gap 2)"
  gaps_remaining: []
  regressions: []
human_verification:
  - test: "With the installed instance replaced by `make install`, run `make install-signal`, restart the installed kernel, and add the Signal source through the app's untrusted-add consent flow."
    expected: "The source syncs and the chip shows the untrusted badge rather than a launch failure."
    why_human: "Requires the operator's real Signal Desktop database and a real UI consent interaction — not reproducible hermetically."
  - test: "Follow the roadmap's 5-step UAT: note the running checkout instance's webspace/mark/health state → `make install` → start `topos` from PATH → confirm identical state on 7777 → run `make dev` alongside and confirm 7778 comes up with neither instance showing the other's data → `make uninstall` and confirm config/index/marks survive."
    expected: "All 5 steps succeed with no data loss and no visible clash."
    why_human: "Requires the operator's real installed instance, real personal data (webspaces, marks, WhatsApp/Signal links), and real concurrent process observation — not reproducible hermetically."
---

# Phase 15: Installed Instance & Dev Isolation Verification Report

**Phase Goal:** The operator runs topos daily from installed release artifacts while
developing the next milestone from the checkout — the two instances can never clash
on port, config, or state. Phase 15 delivers `make install` / `install-signal` /
`uninstall` from published release artifacts, plus full dev-side port, config, and
state isolation from the checkout.
**Verified:** 2026-08-19 (re-verification after gap-closure commit `2ff7f52`)
**Status:** human_needed
**Re-verification:** Yes — after gap closure

## Re-verification Summary

The initial pass (this same date, commit `0482cc6`) found 2 gaps: `cmd/topos-devguard`'s
false clear for a relative `[sources.*] path` resolved against the wrong base directory
(ISOL-01), and `docs/install.md` contradicting the shipped no-argument latest-release
feature (INST-02 discoverability). Commit `2ff7f52` ("fix(15): close verification gaps")
fixes both. Both fixes were independently re-verified against the rebuilt binary and
current doc text, not accepted on the commit message's word:

1. **Gap 1 fix, confirmed:** `cmd/topos-devguard/main.go` now resolves a relative
   `[sources.*] path` against `os.Getwd()` instead of `configDir`, with a doc comment
   recording why (the kernel launches plugin subprocesses with no `cmd.Dir` override, so
   cwd — identical for the guard and the kernel under `make dev` — is the real resolution
   base). Re-running my exact original repro (guard invoked from a cwd inside a simulated
   XDG state root, against a config in a different directory declaring a relative source
   `path`) now printed:
   ```
   devguard: VIOLATION: [sources.mock] path -> /tmp/.../xdgdata/topos/relative-source-store (inside topos state root /tmp/.../xdgdata/topos)
   devguard: 1 violation(s) — refusing to let a dev run reach the installed instance's config or state
   exit=1
   ```
   — the false clear is gone. Two new subtests (`relative source path resolves against
   cwd inside the state root and is a violation` / `... a checkout cwd and is clean`,
   using `t.Chdir`) pin both sides. `go test ./cmd/topos-devguard/ -v` re-run live: all
   subtests pass, including the two new ones. `make dev-check` re-run live: 6/6 cases
   still pass (no regression from the resolution-base change).
2. **Gap 2 fix, confirmed:** `docs/install.md`'s "Installing a release" section now shows
   both forms (`make install` for latest, `make install VERSION=<tag>` for a pin),
   documents the host/path/tag-shape validation and the printed-tag/no-credentials
   behavior, and no longer contains the "VERSION is required — no implicit latest"
   claim. Matches `README.md`'s phrasing. `make docs-check` re-run live: 46 links across
   21 files, all resolve.

No regressions found in either passed-item quick-check (`make dev-check`, `make
docs-check`) or the fixed items' own test suites.

## Goal Achievement

### Observable Truths (requirement-level, cross-referenced to ROADMAP.md success criteria)

| # | Truth (Requirement) | Status | Evidence |
|---|---|---|---|
| 1 | `make install [VERSION]` places kernel at `$PREFIX/bin`, plugins at `$PREFIX/lib/topos/plugins`, SHA-256-verified against the release's own `checksums.txt` first; download-and-copy only, no toolchain (INST-01) | ✓ VERIFIED | `make install-check`: 15/15 cases pass (initial pass, unaffected by the gap-closure commit — no install.sh changes) |
| 2 | `make install` with no version resolves and installs the **latest published stable** release; never a prerelease/nightly; host- and path-validated; accurately documented (INST-02) | ✓ VERIFIED | Code: `install-smoke.sh`'s latest-resolution cases pass, live-resolved `v1.1.0` from the real GitHub redirect (initial pass). Docs: `docs/install.md`'s "Installing a release" section now documents both forms accurately (Gap 2 fix, re-verified this pass) |
| 3 | `topos` from PATH serves on 7777, uses home/XDG config and state unchanged, discovers plugins from the installed plugins dir with no config edit; an operator's live checkout instance migrates without touching config/index (INST-03) | ✓ VERIFIED | `resolvePluginsDir`: 7/7 subtests pass. `install-smoke.sh`'s installed-kernel-finds-installed-plugins case passes. Migration runbook in `docs/install.md` complete and accurate (initial pass, unaffected by gap-closure commit) |
| 4 | `make install-signal` builds the cgo Signal plugin through the repo's single `signal` definition and places it in the installed instance's **external** plugin dir (never trusted), prints destination + one-time consent steps; base `make install` needs no toolchain (INST-04) | ✓ VERIFIED | Live cgo build, placement, trusted-dir refusal, and `--uninstall` all re-confirmed working in the initial pass; unaffected by the gap-closure commit |
| 5 | `make uninstall` removes exactly `$PREFIX/bin/topos` and `topos-plugin-*` under `$PREFIX/lib/topos/plugins`, non-recursive dir cleanup only, config/index/plugin stores byte-identical, idempotent, live-kernel-safe (INST-05) | ✓ VERIFIED | `scripts/uninstall.sh` zero-recursive-rm / zero-XDG-reference greps; `install-smoke.sh`'s data-safety, idempotent, foreign-file, live-kernel cases all pass (initial pass, unaffected) |
| 6 | A dev run refuses to start when any writable path its config declares — config file, index, either plugin dir (including the omitted-`external_dir` default), or any source's own store path — resolves inside the topos config or state root, in one deterministic pass; dev port moves off 7777 (ISOL-01, ISOL-02) | ✓ VERIFIED | **Gap 1 fixed and re-verified this pass:** `cmd/topos-devguard` now resolves a relative `[sources.*] path` against its own `os.Getwd()`, matching the kernel's real subprocess-launch resolution (no `cmd.Dir` override — cwd is shared between guard and kernel under `make dev`). My exact original false-clear reproduction now refuses correctly (see Re-verification Summary). `go test ./cmd/topos-devguard/ -v` passes including the 2 new subtests pinning both the violation and clean sides. `make dev-check` re-run live: 6/6 cases pass, no regression. Port move (7778 dev / 7777 production) unaffected, still confirmed across Makefile/config.dev.example.toml/vite.config.ts |
| 7 | The installed instance and a dev instance run simultaneously, neither able to see or disturb the other's data, pinned by a committed gate (ISOL-03) | ✓ VERIFIED | `make isolation-check`: 3/3 cases pass (initial pass, unaffected by the gap-closure commit — the gate's dev-shaped instance already used absolute paths, so it was never exposed to Gap 1 in the first place) |
| 8 | Requirements traceability: all 8 requirement IDs (INST-01..05, ISOL-01..03) map to Phase 15 plans and are accounted for | ✓ VERIFIED | Every ID appears in exactly one plan's `requirements:` frontmatter; REQUIREMENTS.md's traceability table lists all 8 mapped to Phase 15 with no unmapped rows |

**Score:** 8/8 truths verified. Both gaps from the initial pass are closed and independently re-confirmed against the rebuilt binary / current doc text.

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `scripts/install.sh` | stage→verify→place installer | ✓ VERIFIED | Unaffected by gap-closure commit |
| `scripts/install-smoke.sh` | hermetic install gate | ✓ VERIFIED | 15 cases pass (initial pass) |
| `scripts/uninstall.sh` | closed removal set | ✓ VERIFIED | Unaffected by gap-closure commit |
| `scripts/install-signal.sh` | cgo Signal build+place, `--uninstall` mode | ✓ VERIFIED | Unaffected by gap-closure commit |
| `scripts/simultaneity-smoke.sh` | ISOL-03 committed gate | ✓ VERIFIED | 3 cases pass (initial pass) |
| `scripts/smoke-lib.sh` | shared fixture-release + free-port helpers | ✓ VERIFIED | Unaffected by gap-closure commit |
| `cmd/topos/pluginsdir_test.go` | `resolvePluginsDir` branch coverage | ✓ VERIFIED | 7 subtests pass (initial pass) |
| `cmd/topos-devguard/main.go` + `devguard_test.go` | isolation guard | ✓ VERIFIED | Resolution-base fix confirmed correct; 2 new subtests pin both sides; all subtests pass live this pass |
| `docs/install.md` | operator install/uninstall/signal/migration doc | ✓ VERIFIED | "Installing a release" section now accurate; migration runbook and Signal section unaffected and still accurate |
| `docs/testing.md` | names all gates | ✓ VERIFIED | Unaffected by gap-closure commit |
| Makefile targets: `install`, `install-check`, `install-signal`, `uninstall`, `uninstall-signal`, `isolation-check` | registered, single-definition | ✓ VERIFIED | Unaffected by gap-closure commit |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `resolvePluginsDir` (cmd/topos/main.go) | `$PREFIX/bin`+`$PREFIX/lib/topos/plugins` layout | existence probe gated on exe dir named `bin` | ✓ WIRED | Unaffected |
| `scripts/install.sh` asset list | release's own `checksums.txt` | derived, never hardcoded | ✓ WIRED | Unaffected |
| `release.yml`/`nightly.yml` `ASSETS` | published kernel's link-time build manifest | `MANIFEST_PLUGIN_BINARIES_PORTABLE` | ✓ WIRED | Unaffected |
| `cmd/topos-devguard` | `kernel/config`'s real loader | `config.NewStore` | ✓ WIRED | Unaffected |
| `cmd/topos-devguard`'s root + candidate derivation | `cmd/topos`'s real path resolution (kernel + plugin subprocess launch) | mirrored env-var/fallback logic for roots; `os.Getwd()` for relative source paths (post-fix) | ✓ WIRED | **Now correct:** the guard's relative-source-path base matches the kernel's actual subprocess-launch resolution (no `cmd.Dir` override) |
| `DEV_PORT` (Makefile) | `[server] listen` / Vite proxy target | shared literal `7778` | ✓ WIRED | Unaffected |
| `scripts/simultaneity-smoke.sh` | `scripts/install.sh` + `cmd/topos-devguard` | drives the real scripts, no reimplementation | ✓ WIRED | Unaffected |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| INST-01 | 15-01, 15-02 | Install kernel+plugins to `$PREFIX`, checksum-verified | ✓ SATISFIED | Row 1 |
| INST-02 | 15-02 | `make install` with no version = latest release, documented | ✓ SATISFIED | Row 2 |
| INST-03 | 15-01, 15-05 | `topos` from PATH, home/XDG, plugin discovery, migration | ✓ SATISFIED | Row 3 |
| INST-04 | 15-03 | `make install-signal` builds+places to external tier | ✓ SATISFIED | Row 4 |
| INST-05 | 15-02 | `make uninstall` — exact removal, data untouched | ✓ SATISFIED | Row 5 |
| ISOL-01 | 15-04 | Dev isolation refusal for all writable paths | ✓ SATISFIED | Row 6 (Gap 1 closed and re-verified) |
| ISOL-02 | 15-04 | Non-7777 dev port | ✓ SATISFIED | Row 6 |
| ISOL-03 | 15-05 | Simultaneous run, no clash | ✓ SATISFIED | Row 7 |

No orphaned requirements.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `cmd/topos-devguard/main.go` | 99-110 (`containedIn`) | No symlink resolution (`filepath.EvalSymlinks`) before comparing cleaned/absolutized paths (code review IN-01, still open) | ℹ️ Info | A stowed dotfile/bind-mount could produce a false clear or false refusal; not addressed by the gap-closure commit, not required for this verification's must-haves |
| `scripts/install.sh` | 62, 108, 117 | `TOPOS_RELEASE_BASE_URL` default string duplicated 3x (code review IN-02, still open) | ℹ️ Info | Drift risk on a future repo-rename edit; all 3 currently agree |
| `scripts/smoke-lib.sh` | 11-19 (`smoke_free_port`) | TOCTOU window between ephemeral-port allocation and re-bind (code review IN-03, still open) | ℹ️ Info | Inherent to "ask-then-release" port allocation |
| `scripts/install.sh` | 173 | Plugin-name allowlist regex permits a leading/trailing hyphen (code review IN-04, still open) | ℹ️ Info | No exploitable path found |

The two prior BLOCKER/WARNING findings (Gap 1 / WR-02, Gap 2 / WR-01) are resolved as of
commit `2ff7f52` and are no longer listed here. The four remaining info-level findings
from the phase's own code review were never blocking and are unaffected by this
verification's scope — noted for completeness, not gating.

No debt markers (`TBD`/`FIXME`/`XXX`) found in any file this phase touched.

### Behavioral Spot-Checks (re-run live this pass)

| Behavior | Command | Result | Status |
|---|---|---|---|
| `topos-devguard` full suite, including the 2 new gap-1 subtests | `go test ./cmd/topos-devguard/ -v` | All subtests pass, including `relative_source_path_resolves_against_cwd_inside_the_state_root_and_is_a_violation` and `relative_source_path_resolves_against_a_checkout_cwd_and_is_clean` | ✓ PASS |
| Devguard gap-1 false-clear reproduction, against the rebuilt binary | Guard invoked from a cwd inside a simulated state root, config elsewhere with a relative source path | `devguard: VIOLATION: [sources.mock] path -> .../state-root/relative-source-store (inside topos state root ...)`, exit 1 | ✓ PASS (previously FAIL, now fixed) |
| Dev-guard regression check | `make dev-check` | 6/6 cases pass | ✓ PASS |
| Doc-links regression check | `make docs-check` | 46 links across 21 files, all resolve | ✓ PASS |
| Docs content check | `docs/install.md` "Installing a release" section | Documents both `make install` (latest) and `make install VERSION=<tag>` forms, matches README.md | ✓ PASS |
| Workspace build | `CGO_ENABLED=0 go build ./...` | Clean | ✓ PASS |

### Human Verification Required

Two items remain, deferred by the phase's own plans (`human_judgment: true` in the
15-03 and 15-05 SUMMARYs) and by the roadmap's own stated UAT. Neither is affected by
the two closed gaps — both were already flagged in the initial pass and remain open
regardless of the fix commit:

#### 1. Live Signal install + consent flow (15-03)

**Test:** With the installed instance replaced by `make install`, run `make install-signal`, restart the installed kernel, and add the Signal source through the app's untrusted-add consent flow.
**Expected:** The source syncs and the chip shows the untrusted badge rather than a launch failure.
**Why human:** Requires the operator's real Signal Desktop database and a real UI consent interaction — not reproducible hermetically.

#### 2. Live migration + simultaneity UAT (15-05, the roadmap's stated end-to-end proof)

**Test:** Follow the roadmap's 5-step UAT: note the running checkout instance's webspace/mark/health state → `make install` → start `topos` from PATH → confirm identical state on 7777 → run `make dev` alongside and confirm 7778 comes up with neither instance showing the other's data → `make uninstall` and confirm config/index/marks survive.
**Expected:** All 5 steps succeed with no data loss and no visible clash.
**Why human:** Requires the operator's real installed instance, real personal data (webspaces, marks, WhatsApp/Signal links), and real concurrent process observation — not reproducible hermetically. This UAT flow uses the standard generated dev config (absolute paths), so it was never exposed to Gap 1 even before the fix.

### Gaps Summary

No gaps remain. Both defects found in the initial verification pass — `cmd/topos-devguard`'s
relative-source-path false clear (ISOL-01) and `docs/install.md`'s stale "VERSION
required" text (INST-02 discoverability) — are fixed in commit `2ff7f52` and independently
re-confirmed against the rebuilt binary and current doc text during this re-verification,
not accepted on the fix commit's own description. No regressions were introduced (`make
dev-check` and `make docs-check` both re-run green).

The phase's automated surface (all 8 requirement-level truths, all required artifacts, all
key links, all committed gates) is fully verified. What remains is the two live, human-only
UAT items the phase's own plans deliberately deferred — these are not new findings, they
were flagged in the initial pass and are unaffected by the gap-closure commit. Status is
`human_needed` on that basis: the automated work is complete and correct, and the
milestone's stated end-to-end proof still needs the operator to run it on their own
machine and real data before this phase (and the single-phase v1.2.0 milestone it
constitutes) can be marked fully done.

---

_Verified: 2026-08-19 (re-verification)_
_Verifier: Claude (gsd-verifier)_
