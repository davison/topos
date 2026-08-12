---
phase: 10-docs-and-release-readiness
verified: 2026-08-12T12:35:00Z
status: human_needed
score: 8/9 must-haves verified
behavior_unverified: 1
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 7/9
  gaps_closed:
    - "README.md's Configure section named the wrong SilverBullet env var (SB_URL); fixed in commit b9946a6 ('fix(10): correct SilverBullet env var name in README quickstart') to SILVERBULLET_URL, matching config.example.toml's `base_url = \"${SILVERBULLET_URL}\"` and docs/plugins/silverbullet.md. SB_AUTH_TOKEN on the following line was checked and left unchanged (already correct). Re-verified independently this pass: repo-wide grep confirms zero remaining SB_URL references, and `make docs-check` still passes (35 links / 19 files)."
  gaps_remaining: []
  regressions: []
behavior_unverified_items:
  - truth: "The nightly workflow builds and publishes when HEAD differs from the last nightly, and skips the build entirely when it does not (SC-6 / 10-01 must-have)"
    test: "Push the local branch (currently ahead of origin/main, including the nightly.yml commit 16d6bc2) to origin/main, then run `gh workflow run nightly.yml --ref main` twice in a row at the same commit."
    expected: "First dispatch: run concludes success and the `build` job runs with conclusion success. Second dispatch at the same HEAD: run concludes success and the `build` job does not appear in the run's job list at all (skipped by the `needs`/`if` gate, not merely short-circuited)."
    why_human: "This is a state-transition/gate invariant that only a live GitHub Actions dispatch can prove; static grep/YAML inspection (all of which passed) cannot observe whether the `if: needs.check-changes.outputs.changed == 'true'` gate actually skips the job at runtime. The commit carrying nightly.yml has not been confirmed pushed to origin/main, so no live dispatch is possible from a local verification session. Already logged as an open `unrun-verify` item (#6) in .planning/WINDOWS.md."
human_verification:
  - test: "Push origin/main to include commit 16d6bc2 (and everything since), then dispatch nightly.yml twice at the same commit."
    expected: "Build-then-skip proven live, and the resulting `nightly` GitHub Release exists with isPrerelease: true and the expected asset set."
    why_human: "Requires a push to the real remote and two live GitHub Actions runs — cannot be exercised from a local verification pass. This closes WINDOWS.md entry #6."
---

# Phase 10: Docs and Release Readiness Verification Report

**Phase Goal:** A new user landing on the repo can understand, install, and configure topos from the docs alone — README speaks to users, contributor material lives in CONTRIBUTING.md, security reporting has a home, and every plugin has a consistent one-page doc — and the repo is release-ready: GitHub milestones mirror .planning/ milestones, and CI publishes nightly builds and release artifacts
**Verified:** 2026-08-12T12:35:00Z (re-verification after gap fix)
**Status:** human_needed
**Re-verification:** Yes — after gap closure (commit b9946a6)

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A new visitor reading only README.md can install, configure, and run topos | ✓ VERIFIED | Previously FAILED: `README.md:110` instructed `export SB_URL=...`, a variable nothing in the codebase reads. Fixed in commit `b9946a6` ("fix(10): correct SilverBullet env var name in README quickstart") to `export SILVERBULLET_URL=...`, matching `config.example.toml`'s `base_url = "${SILVERBULLET_URL}"` and `docs/plugins/silverbullet.md`. Re-verified independently this pass: `git show b9946a6` confirms the one-line diff; repo-wide grep for `SB_URL` across `*.go`/`*.toml`/`*.md` now returns zero matches outside `.planning/` and this report; the adjacent `SB_AUTH_TOKEN` line was checked against `config.example.toml:192` (`token = "${SB_AUTH_TOKEN}"`) and is correct, left unchanged. `make docs-check` still passes (35 links / 19 files). |
| 2 | Contributor material (repo layout, dev loop, testing gates, build/release targets) lives in CONTRIBUTING.md, reached from the README | ✓ VERIFIED | `CONTRIBUTING.md` (124 lines) contains all sections; `DEV_PORT`/`DEV_HOST`/`DEV_READY_TIMEOUT`/`make dev-check` etc. confirmed absent from `README.md`; README links `CONTRIBUTING.md`. `make test-portable` passes. |
| 3 | SECURITY.md exists in GitHub's recommended format, with a live private-reporting channel | ✓ VERIFIED | `SECURITY.md` has the three literal headings, links `security/advisories/new`, states the loopback/no-auth boundary and 7-day ack target. `gh api repos/davison/topos/private-vulnerability-reporting --jq '.enabled'` → `true` (confirmed live this session). |
| 4 | docs/plugins/ contains a consistent one-page doc per real plugin, derived from _template.md | ✓ VERIFIED | 7 files present (`_template.md`, `README.md`, 5 plugin pages). Every plugin page's `Match vocabulary:` line matches the plugin's own `matchVocabulary` source declaration exactly (paperless=tags, silverbullet=tags+pages, proton=folders, signal=conversations, whatsapp=groups+contacts — all confirmed by grep against `plugins/*/plugin.go`). |
| 5 | The Svelte scaffold README under web/ is replaced with something useful | ✓ VERIFIED (minor wiring caveat) | `web/README.md` (11 lines) contains no scaffold instructions; states the SPA is embedded at build time. Points at `CONTRIBUTING.md` and `docs/testing.md` using bold+code text rather than real relative links (`../CONTRIBUTING.md`), so a reader browsing from inside `web/` on GitHub can't click through — flagged as WR-01 in 10-REVIEW.md, not a broken *truth* (scaffold content is genuinely gone) but a real usability gap. See Anti-Patterns. |
| 6 | GitHub repo milestones stay in sync with .planning/ milestones via a committed, idempotent script | ✓ VERIFIED | `scripts/sync-milestones.sh` exists, executable, has no DELETE path, looks up by title before create/patch. Live check this session: `gh api repos/davison/topos/milestones?state=all` shows exactly one `v1.0` milestone, number 1, open — matches `.planning/STATE.md`'s `milestone: v1.0`. `docs/releasing.md` documents the process. |
| 7 | A tag push produces a GitHub Release with the kernel binary, operator-facing plugin binaries, and a checksums file | ✓ VERIFIED | `.github/workflows/release.yml` exists with the correct trigger, `permissions: contents: write` only, official-actions-only, no non-GITHUB_TOKEN secret. Proven with a real run per 10-01-SUMMARY.md (tag `v0.0.0-citest`, run succeeded in 55s, exact 6-asset set confirmed, then cleaned up). `make build-portable` re-run live this session: exit 0, produces `bin/topos` + 5 plugin binaries, no `topos-plugin-signal`. |
| 8 | The nightly workflow builds and publishes only when code has changed since the last nightly, and skips otherwise | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | `.github/workflows/nightly.yml` exists with the correct triggers, `check-changes`/`build` job split, `needs`/`if` gate, `fetch-depth: 0`, no secrets beyond `GITHUB_TOKEN`. All static/grep acceptance criteria pass. The live two-dispatch build-then-skip proof was NOT run this session. Logged as WINDOWS.md entry #6, an explicit, legitimate deferral. |
| 9 | Every relative markdown link in every committed doc resolves, and a broken future link fails CI | ✓ VERIFIED | `make docs-check` run live this session: "checked 35 link(s) across 19 file(s) — all resolve." `.github/workflows/ci.yml` runs `make docs-check` as the first step of the `test` job (confirmed at line 54), introduces no new `uses:` action (action count unchanged). Per 10-05-SUMMARY.md the guard was proven to genuinely fail on an injected broken link. |

**Score:** 8/9 truths verified (1 present, behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `.github/workflows/release.yml` | Tag-triggered release build/publish | ✓ VERIFIED | Exists, correct trigger/permissions/actions, proven with a real run. |
| `.github/workflows/nightly.yml` | Change-gated nightly build | ✓ VERIFIED (present+wired) | Exists, statically correct; live change-gate behavior unverified (see truth #8). |
| `Makefile` (`build-portable`/`plugins-portable`) | cgo-free build entry point | ✓ VERIFIED | Re-ran live: produces `bin/topos` + 5 plugin binaries (paperless, silverbullet, proton, mock, whatsapp), no signal binary. |
| `docs/plugins/_template.md` | Canonical section shape | ✓ VERIFIED | All four headings + Match vocabulary line present. |
| `docs/plugins/{signal,paperless,silverbullet,proton,whatsapp}.md` | One operator page per real plugin | ✓ VERIFIED | All present, all under 120 lines, match vocab confirmed against source. |
| `SECURITY.md` | GitHub-format vulnerability policy | ✓ VERIFIED | Present, correct shape, live-enabled reporting confirmed. |
| `web/README.md` | Non-scaffold pointer README | ✓ VERIFIED (see WR-01 caveat) | No scaffold content; cross-references use non-clickable bold+code text instead of real relative links. |
| `docs/ss/README.md` | Screenshot convention | ✓ VERIFIED | Documents 4 indices; directory holds only the README, no committed images. |
| `scripts/sync-milestones.sh` | Idempotent milestone sync | ✓ VERIFIED | Live-reconciled against real `v1.0` milestone (#1), no delete path. |
| `docs/releasing.md` | Maintainer release process page | ✓ VERIFIED | Covers milestones/release/nightly/Signal decision; all referenced paths resolve. |
| `CONTRIBUTING.md` | Contributor-facing content split | ✓ VERIFIED | All required content present; `README.md` cleaned of moved content. |
| `README.md` | New-user-facing README | ✓ VERIFIED | Configure section's SilverBullet env var corrected to `SILVERBULLET_URL` in commit `b9946a6`; re-verified against `config.example.toml` and `docs/plugins/silverbullet.md`. |
| `scripts/check-doc-links.sh` | Repo-wide link guard | ✓ VERIFIED | `make docs-check` passes live (35 links / 19 files); wired into `ci.yml`'s `test` job as the first step. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `.github/workflows/release.yml` | `Makefile` | `make build-portable` | ✓ WIRED | Confirmed in workflow source. |
| `.github/workflows/nightly.yml` | moving `nightly` tag | `git rev-parse` compare | ✓ WIRED (statically) | Gate logic present; live skip behavior unverified. |
| `docs/plugins/*.md` | `config.example.toml` | pointer, not duplication | ✓ WIRED | Every page under 120 lines, points at the reference block. |
| `docs/plugins/README.md` | `docs/plugins/_template.md` | index names template | ✓ WIRED | Confirmed by `make docs-check` (all 35 links resolve) and manual read. |
| `README.md` | `CONTRIBUTING.md`, `SECURITY.md`, `docs/plugins/`, `docs/releasing.md` | cross-links | ✓ WIRED | All present and resolve. |
| `web/README.md` | `CONTRIBUTING.md`, `docs/testing.md` | implied cross-reference | ⚠️ PARTIAL | Not real markdown links — bold+code text with no path prefix; a reader in `web/`'s directory view cannot click through or infer the repo-root-relative path. Not caught by `check-doc-links.sh` because it only scans `[text](target)`/`![alt](target)` syntax (documented limitation, IN-01 in 10-REVIEW.md). |
| `.github/workflows/ci.yml` | `scripts/check-doc-links.sh` | `make docs-check` step | ✓ WIRED | Confirmed at line 54, first step of `test` job. |
| `README.md` | Configure env vars | `${SILVERBULLET_URL}` expansion in config.example.toml | ✓ WIRED | Fixed in commit `b9946a6`; README now names `SILVERBULLET_URL`, matching `config.example.toml` and `docs/plugins/silverbullet.md`. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Portable build produces correct artifact set, no signal binary | `rm -rf bin && make build-portable` | exit 0; `bin/topos` + 5 plugin binaries under `bin/plugins/`, no `topos-plugin-signal` | ✓ PASS |
| Doc-link guard passes against current repo state | `make docs-check` | "checked 35 link(s) across 19 file(s) — all resolve" | ✓ PASS |
| GitHub milestone state matches `.planning/STATE.md` | `gh api repos/davison/topos/milestones?state=all` | exactly one `v1.0` milestone, number 1, open | ✓ PASS |
| Private vulnerability reporting live-enabled | `gh api repos/davison/topos/private-vulnerability-reporting --jq '.enabled'` | `true` | ✓ PASS |
| README no longer references `SB_URL` anywhere in the repo | `grep -rn "SB_URL" . --include="*.go" --include="*.toml" --include="*.md"` | zero matches outside `.planning/`/this report | ✓ PASS |
| Nightly workflow change-gate skip behavior | `gh workflow run nightly.yml --ref main` (x2) | not run — live dispatch proof still deferred | ? SKIP (see human verification) |

### Probe Execution

No conventional `scripts/*/tests/probe-*.sh` files exist for this phase; the phase's own live-API verifications (release run, milestone sync, private vulnerability reporting) were already executed and recorded by the executor in 10-01-SUMMARY.md/10-03-SUMMARY.md/10-04-SUMMARY.md, and spot-re-confirmed live in this verification pass where the remote state is still checkable (milestones, private vuln reporting, `make build-portable`, `make docs-check`).

### Requirements Coverage

Phase 10 declares no REQ-IDs (`REQUIREMENTS.md` maps no requirement to Phase 10 — confirmed by scanning its traceability table). Plans instead declare SC-* success criteria; all six roadmap success criteria are addressed by at least one plan:

| Success Criterion | Source Plan(s) | Status | Evidence |
|---|---|---|---|
| SC-1 (README for new users) | 10-05 | ✓ SATISFIED | README exists, well-structured; the SilverBullet env-var bug (CR-01) is fixed (commit `b9946a6`) and re-verified. |
| SC-2 (SECURITY.md) | 10-03 | ✓ SATISFIED | Verified live. |
| SC-3 (docs/plugins/) | 10-02 | ✓ SATISFIED | Verified, match vocab confirmed against source. |
| SC-4 (web/ scaffold README replaced) | 10-03 | ✓ SATISFIED (minor caveat) | Scaffold gone; cross-links imprecise (WR-01). |
| SC-5 (GitHub milestones mirror .planning/) | 10-04 | ✓ SATISFIED | Verified live against real milestone. |
| SC-6 (publishing workflow: nightly + release) | 10-01 | ⚠️ PARTIAL | Release half proven live; nightly change-gate proof deferred, legitimately, pending a push to origin/main (WINDOWS.md #6). |

No orphaned requirements — Phase 10 has none mapped in `REQUIREMENTS.md`, consistent with its stated scope ("docs backlog... none").

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `web/README.md` | 8-10 | Cross-references to `CONTRIBUTING.md`/`docs/testing.md` rendered as bold+code text, not real relative links, and don't indicate they live two directories up | ⚠️ Warning | A reader opening this file from inside `web/` (GitHub's directory browser, a local editor) cannot tell these paths are repo-root-relative. Not caught by `check-doc-links.sh`, which only scans real markdown link/image syntax — documented as IN-01 in 10-REVIEW.md. |
| `.github/workflows/nightly.yml` / `release.yml` | asset list | Identical `ASSETS=` string hard-coded independently in both files, no single source of truth | ⚠️ Warning | If a future phase adds a sixth published plugin, one workflow could be updated and the other forgotten with no error — flagged as WR-02 in 10-REVIEW.md. Not introduced by carelessness (the plan explicitly wrote the same list into both files), but is a real duplication risk going forward. |
| `Makefile` (`test-portable` target, lines ~135-142) | n/a | `CGO_ENABLED=0` prefix only scopes to the `go build` half of each `... && go test ...` line, not the paired `go test` | ℹ️ Info | Pre-existing (not touched by Phase 10's diff), but Phase 10's own new docs (`CONTRIBUTING.md`, `docs/releasing.md`) now actively cite `test-portable` as "needs no C toolchain," a promise the target doesn't fully keep for its test half on a machine with a C compiler present. Flagged as WR-03 in 10-REVIEW.md. Out of this phase's scope to fix (target predates Phase 10), noted for awareness only. |

No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers found in any file this phase created or modified. The previously-flagged blocker (`README.md`'s `SB_URL` bug, CR-01) is resolved as of commit `b9946a6` and no longer appears in this table.

**Working-tree note (not a code gap, resolved during initial verification pass):** at the start of the initial verification, the working tree had an uncommitted deletion of `kernel/webui/build/.gitkeep` — a known side effect of running `make build-portable`/`make test-portable` during phase execution. Restored via `git checkout -- kernel/webui/build/.gitkeep`; no source was affected. Working tree is clean again as of this re-verification (aside from this report file).

### Human Verification Required

### 1. Nightly workflow live change-gate proof (WINDOWS.md entry #6)

**Test:** Push the current local `main` to `origin/main` (including nightly.yml's commit), then run `gh workflow run nightly.yml --ref main` twice at the same commit.
**Expected:** First dispatch: run succeeds and the `build` job runs to `success`. Second dispatch at the same HEAD: run succeeds and the `build` job is entirely absent from the run's job list (not merely skipped-with-conclusion — absent). A GitHub Release on tag `nightly` exists afterward with `isPrerelease: true` and the expected asset set (`topos` + 4 portable plugin binaries + `checksums.txt`).
**Why human:** Requires pushing to the real remote and driving two live GitHub Actions runs — outside what a local verification pass can exercise. This is the same deferral the executor itself recorded (10-01-SUMMARY.md, WINDOWS.md entry #6); it is a legitimate, already-disclosed gap rather than a silently-missed one.

### Gaps Summary

No blocking gaps remain. The one previously-identified blocker — `README.md`'s Configure section naming `SB_URL` instead of `SILVERBULLET_URL` (CR-01 in 10-REVIEW.md) — is fixed in commit `b9946a6` ("fix(10): correct SilverBullet env var name in README quickstart") and independently re-verified this pass: the diff is a single-line correction, the adjacent `SB_AUTH_TOKEN` line was checked and confirmed already correct, a repo-wide grep confirms zero remaining `SB_URL` references, and `make docs-check` still passes (35 links / 19 files).

One item remains open and routes to human verification rather than blocking the phase: the nightly workflow's live two-dispatch change-gate proof (build-then-skip) has not been run against the real GitHub API. This is a disclosed, reasoned deferral — already logged as WINDOWS.md entry #6 by the executor — not an oversight, and is a state-transition invariant that only a live dispatch against `origin/main` can prove.

---

*Verified: 2026-08-12T12:35:00Z*
*Verifier: Claude (gsd-verifier)*
