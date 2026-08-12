---
phase: 10-docs-and-release-readiness
plan: 01
subsystem: infra
tags: [github-actions, ci-cd, makefile, release-engineering, sha256, gh-cli]

# Dependency graph
requires: []
provides:
  - "Makefile plugins-portable/build-portable targets — cgo-free build entry point, mirrors the existing test-portable/test split"
  - ".github/workflows/release.yml — tag-triggered (v*.*.*) GitHub Release publishing bin/topos + 4 portable plugin binaries + checksums.txt"
  - ".github/workflows/nightly.yml — change-gated nightly prerelease on a moving 'nightly' tag, cron + workflow_dispatch"
  - "Recorded decision: Signal plugin binary excluded from all published artifacts (option-b); make signal is the documented local build path"
affects: [10-04-docs-releasing, 10-05-readme-install-section]

# Actuals (#2632)
actuals:
  tokens: 2275
  tasks: 3
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Hash-and-publish combined in one workflow step so the shipped asset list is written exactly once (release.yml and nightly.yml both)"
    - "Change-gated nightly build via a moving git tag compared with git rev-parse, not the Actions API"

key-files:
  created:
    - .github/workflows/release.yml
    - .github/workflows/nightly.yml
  modified:
    - Makefile

key-decisions:
  - "Task 2 checkpoint: option-b selected — Signal plugin binary is excluded from every published artifact (release and nightly); make signal remains the supported local build path. User's stated reason: 'Publish only static CGO_ENABLED=0 binaries that run anywhere; workflows stay cgo-free, matching ci.yml's existing test-portable choice. Signal users follow the already-documented make signal local build against their own SQLCipher.'"

patterns-established:
  - "plugins-portable/build-portable: cgo-free Makefile targets, five real plugin names written in plugins-portable only, build-portable delegates rather than re-listing them"

requirements-completed: [SC-6]

coverage:
  - id: D1
    description: "make build-portable and make plugins-portable produce a complete, cgo-free artifact set (bin/topos + 5 plugin binaries, no signal binary)"
    requirement: "SC-6"
    verification:
      - kind: other
        ref: "make build-portable && test -x bin/topos && test -x bin/plugins/topos-plugin-whatsapp && test ! -e bin/plugins/topos-plugin-signal (run live this session)"
        status: pass
      - kind: other
        ref: "rm -rf bin && CGO_ENABLED=0 make plugins-portable — produced exactly topos-plugin-mock topos-plugin-paperless topos-plugin-proton topos-plugin-silverbullet topos-plugin-whatsapp"
        status: pass
    human_judgment: false
  - id: D2
    description: "A pushed v*.*.* tag produces a real GitHub Release with the kernel binary, four portable plugin binaries, and a valid checksums.txt"
    requirement: "SC-6"
    verification:
      - kind: other
        ref: "gh run watch 31587119112 --exit-status against tag v0.0.0-citest on davison/topos — conclusion success in 55s; gh release view v0.0.0-citest --json assets confirmed exactly checksums.txt,topos,topos-plugin-paperless,topos-plugin-proton,topos-plugin-silverbullet,topos-plugin-whatsapp; checksums.txt downloaded and confirmed 5 well-formed sha256sum lines; release+tag cleaned up after"
        status: pass
    human_judgment: false
  - id: D3
    description: "nightly.yml's change gate skips the build job entirely on a second dispatch at the same commit, and the Signal-exclusion decision is applied identically to both workflows"
    verification: []
    human_judgment: true
    rationale: "Live dispatch verification (two consecutive workflow_dispatch runs against nightly.yml, proving build-then-skip, plus the resulting nightly prerelease) requires the file to be pushed to origin/main first (Task 3's own <precondition>). The coordinator explicitly instructed no further pushes to origin/main after Task 1's real release run, so this commit (16d6bc2) is local-only on worktree-agent-aa4413156731f3f6e. All static/grep-based acceptance criteria (schedule+workflow_dispatch triggers, fetch-depth:0, needs/if gating expression, permissions scoping, action pins, no secrets beyond GITHUB_TOKEN, no go build literal, no apt-get, no topos-plugin-signal in either asset list, make signal named in both release-notes bodies) were verified and pass. Logged to .planning/WINDOWS.md (unrun-verify, entry #6) for a human/orchestrator to complete after this branch is integrated."

# Metrics
duration: ~25min active (plus a checkpoint pause awaiting the Task 2 decision)
completed: 2026-08-12
status: complete
---

# Phase 10 Plan 1: Release Engineering Foundation Summary

**Tag-triggered GitHub Release workflow and change-gated nightly build, both cgo-free (Signal excluded by explicit decision), proven against the real GitHub API with a live tag push and cleanup**

## Performance

- **Duration:** ~25 min active work (checkpoint pause for the Task 2 decision not counted)
- **Started:** 2026-08-12
- **Completed:** 2026-08-12
- **Tasks:** 3 (tracer, checkpoint:decision, auto)
- **Files modified:** 3 (Makefile, .github/workflows/release.yml, .github/workflows/nightly.yml)

## Accomplishments
- Added `plugins-portable`/`build-portable` Makefile targets — the cgo-free build entry point, mirroring the existing `test-portable`/`test` split; verified locally (`make build-portable` and an isolated `CGO_ENABLED=0 make plugins-portable` both produce the correct artifact set with no signal binary)
- Added `.github/workflows/release.yml` (tag-triggered `v*.*.*` → GitHub Release) and proved it end-to-end against the real `davison/topos` GitHub API: pushed tag `v0.0.0-citest`, watched the run to `success` (55s), confirmed the exact six-asset set plus a valid `checksums.txt`, then cleaned up
- Task 2 checkpoint decided: Signal plugin binary is excluded from every published artifact (option-b) — recorded below verbatim
- Added `.github/workflows/nightly.yml`: cron (`0 3 * * *`) + `workflow_dispatch`, `check-changes` job gating the `build` job via a moving `nightly` git tag comparison, and applied the Signal-exclusion decision identically to both workflows (both stay on `make build-portable`, no `apt-get`, both release-notes bodies name `make signal` as the local build path for Signal)

## Task Commits

Each task was committed atomically:

1. **Task 1: End-to-end release publish — one tag, real run, real assets** - `74239d2` (feat) — **pushed directly to `origin/main`** (see Deviations — this was required by the task's own live-verification `<verify>` steps)
2. **Task 2: Decide whether the Signal plugin binary ships as a published artifact** - checkpoint, no code commit (decision recorded below)
3. **Task 3: Change-gated nightly workflow, and the Signal decision applied to both** - `16d6bc2` (feat) — **local-only on `worktree-agent-aa4413156731f3f6e`, not pushed** (coordinator instruction — see Deviations)

**Plan metadata:** this commit (SUMMARY.md + WINDOWS.md) — see below

## Files Created/Modified
- `Makefile` - added `plugins-portable` (cgo-free 5-binary plugin set) and `build-portable` (SPA + kernel + delegates to plugins-portable); added both to `.PHONY`
- `.github/workflows/release.yml` - tag-triggered (`v*.*.*`) release workflow: `actions/checkout@v7`/`setup-go@v7`/`setup-node@v7` pins matching `ci.yml`, job-level `permissions: contents: write` only, `make build-portable`, combined hash-and-publish step (`sha256sum` → `checksums.txt`, `gh release create` with the Signal-exclusion release-notes line)
- `.github/workflows/nightly.yml` - cron + `workflow_dispatch` nightly workflow: `check-changes` job (fetch-depth 0, `rev-parse HEAD` vs `rev-parse nightly` comparison, `changed` output) gates the `build` job (`needs`/`if`), which mirrors `release.yml`'s build steps then force-moves the `nightly` tag, deletes any existing `nightly` release, and republishes with `--prerelease`

## Decisions Made

**Task 2 checkpoint decision — recorded verbatim for Task 3 and Plan 10-04:**

- **Selected:** `option-b` — Exclude the Signal binary; document `make signal` as the local build path.
- **User's stated reason:** "Publish only static CGO_ENABLED=0 binaries that run anywhere; workflows stay cgo-free, matching ci.yml's existing test-portable choice. Signal users follow the already-documented `make signal` local build against their own SQLCipher."
- **Applied to:** both `.github/workflows/release.yml` and `.github/workflows/nightly.yml` — both build via `make build-portable` (no `apt-get install libsqlcipher-dev`), neither asset list names `topos-plugin-signal`, and both release-notes bodies name `make signal` (with a pointer to `plugins/signal/README.md`) as the supported local build path.
- **Downstream:** Plan 10-04 (`docs/releasing.md`) and Plan 10-05 (README install section) should state this decision and its reason as-is, not re-litigate it.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Rebased the worktree branch onto a moved `origin/main` before Task 1's required live push**
- **Found during:** Task 1, immediately before the plan's `git push origin HEAD:main` verify step
- **Issue:** This worktree's base commit (`fa8ba89`) and `origin/main`'s actual tip at task start (`31e3b98`) were different commit objects (a prior rebase-merge artifact — different hashes/committer timestamps for otherwise-identical history), so a plain `git push origin HEAD:main` would have been rejected as non-fast-forward. Task 1's own `<verify>` block requires pushing straight to `origin/main` for a real GitHub Actions run — this is not optional per the plan's own text ("The verifications therefore push commits and a throwaway tag to origin, and clean up after themselves").
- **Fix:** Confirmed via `git cat-file -p` that `fa8ba89`'s tree (`5f1eafbb49ee...`) was byte-identical to `31e3b98`'s tree — i.e., zero real content divergence, only rewritten commit metadata. Ran `git rebase origin/main` (clean, zero conflicts, confirmed by the identical trees) then pushed as a genuine fast-forward: `31e3b98..74239d2`. This is non-destructive (no force-push, no history rewrite of anything already on `origin/main`) and consistent with CLAUDE.md's stated rebase-only linear-history preference.
- **Files modified:** none beyond Task 1's own `Makefile` / `.github/workflows/release.yml` — this was a branch-integration operation, not a code change.
- **Verification:** `git merge-base --is-ancestor origin/main HEAD` returned true after the rebase; the subsequent real Actions run against `74239d2` succeeded.
- **Committed in:** `74239d2` (the commit itself is unchanged by the rebase; only its parent/hash context moved)

**2. [Rule 3 - Blocking, coordinator-directed] Task 3's live-dispatch verification deferred**
- **Found during:** Task 3, after Task 1's real push to `origin/main`
- **Issue:** Task 3's own `<precondition>` requires `.github/workflows/nightly.yml` to be pushed to `origin/main` before its dispatch-based `<verify>` steps (two consecutive `workflow_dispatch` runs proving build-then-skip) can run at all.
- **Fix:** The coordinator explicitly instructed, after the Task 2 checkpoint resumed: "For Tasks 2-3, do NOT push to origin/main again — commit locally on your worktree branch only." Complied: Task 3's code (`nightly.yml`, plus the Signal-exclusion changes to `release.yml`) is fully implemented and committed (`16d6bc2`), and every acceptance criterion checkable without a live dispatch (schedule/workflow_dispatch triggers present, `fetch-depth: 0` + `outputs.changed`, `needs`/`if` gating expression, `permissions: contents: write` only with `write-all` count 0, only official `actions/*@v7` pins, no `secrets.` reference beyond `GITHUB_TOKEN`, `go build` count 0, `apt-get` count 0, `make build-portable` in both, no `topos-plugin-signal` in either asset list, `make signal` named in both release-notes bodies) was verified via grep/YAML-parse and passes. The live two-dispatch proof and the resulting `nightly` prerelease were NOT run.
- **Files modified:** none beyond Task 3's own files.
- **Verification:** Recorded as an open `unrun-verify` entry (#6) in `.planning/WINDOWS.md` for a human/orchestrator to close out once this worktree branch is integrated with `origin/main`.
- **Committed in:** `16d6bc2` (local-only, not pushed)

---

**Total deviations:** 2 auto-fixed (2 blocking, one coordinator-directed)
**Impact on plan:** No scope creep — both deviations are branch-integration/verification-sequencing issues, not code changes beyond what the plan specified. Task 1's real verification ran in full against the real GitHub API and passed. Task 3's static verification passed in full; its live-dispatch proof is the one open item, tracked in WINDOWS.md.

## Issues Encountered

- The npm/SvelteKit build step in `make build-portable` overwrites `kernel/webui/build/.gitkeep` with real build output as a side effect; restored the placeholder (`git checkout -- kernel/webui/build/.gitkeep`) before staging Task 1's commit so no unrelated file showed as deleted.

## User Setup Required

None - no external service configuration required. `gh auth status` was already authenticated with `repo`/`workflow` scopes for `davison/topos` (Task 1's `<precondition>`), so no setup step was needed this session.

## Next Phase Readiness

- Plan 10-04 (`docs/releasing.md`) and Plan 10-05 (README install section) can consume the Task 2 decision (option-b) verbatim from this SUMMARY without re-deciding it.
- **Open item before this worktree branch is integrated:** `.github/workflows/nightly.yml` and the Signal-exclusion edits to `.github/workflows/release.yml` are committed locally (`16d6bc2`) but not yet on `origin/main`. Once integrated, a human/orchestrator should dispatch `nightly.yml` twice at the same commit to complete Task 3's live change-gate proof (`gh workflow run nightly.yml --ref main`, twice, confirming the second run's job list omits `build`), and confirm the resulting `nightly` GitHub Release. This closes WINDOWS.md entry #6.
- Task 1's own commit (`74239d2`) is already live on `origin/main` — the release workflow itself is real and does not need re-verification, only Task 3's nightly addition does.

## Self-Check: PASSED

- FOUND: `Makefile`
- FOUND: `.github/workflows/release.yml`
- FOUND: `.github/workflows/nightly.yml`
- FOUND: `.planning/phases/10-docs-and-release-readiness/10-01-SUMMARY.md`
- FOUND commit: `74239d2` (Task 1, pushed to origin/main)
- FOUND commit: `16d6bc2` (Task 3, local-only)
- FOUND commit: `c702ea0` (plan metadata — SUMMARY.md + WINDOWS.md)

---
*Phase: 10-docs-and-release-readiness*
*Completed: 2026-08-12*
