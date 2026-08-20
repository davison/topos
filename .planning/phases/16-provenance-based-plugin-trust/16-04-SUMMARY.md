---
phase: 16-provenance-based-plugin-trust
plan: 04
subsystem: security
tags: [ed25519, plugin-trust, provenance, github-actions, release-pipeline, topos-plugins]

# Dependency graph
requires:
  - phase: 16-01
    provides: "Signed release-manifest trust arm end-to-end: format, producers (BuildProvenanceManifest/SignProvenanceManifest), verifier (VerifySignedProvenance/EvaluateTrust), embeddedProvenanceKeys seam, cmd/topos-provenance CLI (keygen/sign/verify)"
  - phase: 16-03
    provides: "TRUST-04 escalation suite closed (config edit, file drop, shadowing), proving the tier model this plan's real release is evaluated against is already hardened"
provides:
  - "The topos-plugins sibling repository (public, davison/topos-plugins): a tag-triggered ed25519 signing workflow, one trivial real plugin (cmd/topos-plugin-demo), and its first signed release (v0.0.1) — the seed Phase 17 fills"
  - "embeddedProvenanceKeys populated with the real topos-plugins-2026a public key, plus TestEmbeddedProvenanceKeys_WellFormed/_NamesToposPlugins2026a pinning it against a malformed or missing future edit"
  - "16-04-TRUST02-PROOF.md: recorded end-to-end evidence that a real, pipeline-produced release verifies as first-party trusted with no link-time build-manifest entry, including negative cases and fully offline verification"
affects: [16-05, 17]

# Actuals (#2632)
actuals:
  tokens: 9000
  tasks: 4
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Real-pipeline proof over hand-crafted test double: TRUST-02's proving artifact is a binary that came from a genuine tag-push -> GitHub Actions -> gh release create pipeline in its own repository, never a locally hand-signed stand-in presented as a release"
    - "Exact-commit CLI pinning as a stronger substitute for a not-yet-existing release tag: topos-plugins' release workflow pins cmd/topos-provenance to a full commit SHA rather than @latest, satisfying 'pinned, not floating' before any topos kernel release tag carries that CLI"
    - "GPG public-key encryption to an existing operator identity as the safe offline-backup mechanism for a one-way signing key, avoiding both an unencrypted backup and an executor-invented passphrase"

key-files:
  created:
    - .planning/phases/16-provenance-based-plugin-trust/16-04-TRUST02-PROOF.md
  modified:
    - kernel/pluginhost/provenance.go
    - kernel/pluginhost/provenance_test.go

key-decisions:
  - "Pinned the topos-plugins release workflow's cmd/topos-provenance invocation to the exact topos kernel commit SHA (e0290110026c5717d8fadebcb15f681314b9f2c1) rather than a release tag, because no topos kernel release tag yet includes cmd/topos-provenance (it was added earlier in this same phase, still mid-worktree). Required pushing this worktree's branch to origin so GitHub Actions could resolve the module via the Go module proxy. An exact commit SHA is a strictly narrower pin than a floating tag, satisfying the plan's 'pinned version rather than a floating one' acceptance criterion; should move to a real semver tag once the kernel next cuts a release including this CLI."
  - "Restructured the topos-plugins repository from a nested demo/ submodule to a single root-level module with the plugin package at cmd/topos-plugin-demo/, because a nested submodule made a bare 'go build ./...' from the repository root a no-op (no root go.mod) — failing the plan's own literal acceptance criterion. The chosen layout also avoids the alternate failure mode (a top-level directory literally named topos-plugin-demo colliding with go build's own default output filename of the same name)."
  - "Encrypted the key-backup decision's private-key copy via GPG public-key encryption to the operator's own existing GPG identity (darren@davisononline.org, ultimate trust) rather than a passphrase-protected symmetric encryption, because generating or inventing a passphrase on the operator's behalf was explicitly prohibited by the checkpoint resolution — this required no interactive prompt and the operator already controls the corresponding private key's passphrase."

patterns-established:
  - "Throwaway-prefix installed-instance proof: build a kernel with make build-portable + go build ./cmd/topos-provenance, place both plus downloaded release assets into a manually-assembled $PREFIX layout (bin/, lib/topos/plugins/, etc/config.toml) — the pattern any future retroactive install-time proof in this repo can reuse without needing scripts/install.sh to support unpublished local builds."

requirements-completed: [TRUST-02]

coverage:
  - id: D1
    description: "The topos-plugins repository exists as a public, minimal seed: one trivial real plugin (cmd/topos-plugin-demo) and a tag-triggered signing workflow, with the documented first-party trust boundary in its README"
    requirement: "TRUST-02"
    verification:
      - kind: other
        ref: "gh repo view davison/topos-plugins --json name,visibility (public, name topos-plugins)"
        status: pass
      - kind: other
        ref: "gh secret list --repo davison/topos-plugins (TOPOS_PROVENANCE_SIGNING_KEY present)"
        status: pass
      - kind: other
        ref: "cd topos-plugins && go build ./... (exit 0, produces ./topos-plugin-demo)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The kernel's embedded accepted-key set names the topos-plugins key (topos-plugins-2026a) by id, compiled in, pinned by a dedicated test asserting well-formedness and presence"
    requirement: "TRUST-02"
    verification:
      - kind: unit
        ref: "kernel/pluginhost/provenance_test.go#TestEmbeddedProvenanceKeys_WellFormed"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/provenance_test.go#TestEmbeddedProvenanceKeys_NamesToposPlugins2026a"
        status: pass
      - kind: unit
        ref: "go test ./kernel/pluginhost/... -count=1 (full package, unaffected)"
        status: pass
    human_judgment: false
  - id: D3
    description: "A real topos-plugins release (v0.0.1), signed by its own tag-triggered GitHub Actions workflow using the repository's own secret, verifies as first-party trusted on an installed topos instance built from this worktree, with NO entry in that kernel's link-time build manifest"
    requirement: "TRUST-02"
    verification:
      - kind: other
        ref: "gh run list --repo davison/topos-plugins --workflow release.yml --limit 1 --json conclusion (success)"
        status: pass
      - kind: other
        ref: "throwaway-prefix installed instance: topos-provenance verify --dir <plugins dir> exits 0, names topos-plugins-v0.0.1.provenance.json"
        status: pass
      - kind: other
        ref: "throwaway-prefix installed instance: GET /api/sources reports tier=trusted, no launch_failure — recorded in 16-04-TRUST02-PROOF.md"
        status: pass
    human_judgment: false
  - id: D4
    description: "The three negative cases behave as required: missing signature falls to the external/untrusted consent path (never silently trusted), a tampered binary refuses launch and names the binary, and verification succeeds with the kernel process unable to reach any external network (D-01, fully offline)"
    requirement: "TRUST-02"
    verification:
      - kind: other
        ref: "throwaway-prefix: .provenance.sig removed -> GET /api/sources reports tier=external, launch_failure=pin_mismatch"
        status: pass
      - kind: other
        ref: "throwaway-prefix: one byte appended to binary -> topos-provenance verify FAILs by name; GET /api/sources reports launch_failure=manifest_unverified"
        status: pass
      - kind: other
        ref: "unshare --net network-isolated namespace: verify + GET /api/sources both succeed with tier=trusted while curl to an external host returns http_code=000"
        status: pass
    human_judgment: false
  - id: D5
    description: "The private signing key exists only as the topos-plugins GitHub Actions secret plus one GPG-encrypted offline backup (the operator's key-backup decision) — never committed, never printed, never left on local disk in plaintext"
    verification:
      - kind: other
        ref: "grep of the full release.yml workflow run log for the private key material: no match; keygen temp directory deleted and asserted absent before this SUMMARY was written; GPG-encrypted backup round-tripped successfully before plaintext deletion"
        status: pass
    human_judgment: false
  - id: D6
    description: "The operator has confirmed the live-instance re-check (real installed kernel, not the throwaway-prefix proof) after merge/rebuild, as part of phase UAT"
    human_judgment: true
    rationale: "The plan's final checkpoint:human-verify task asks the operator to re-run steps 3-5 (add source, remove/restore signature) on their OWN live installed instance. The operator approved the checkpoint and the published one-way artifacts, but explicitly deferred the live-instance re-check to post-merge/rebuild as part of phase UAT, since the operator's currently-installed kernel predates this plan's embedded-key change. This is a genuinely deferred human step, not something a test can stand in for."

# Metrics
duration: 22min
completed: 2026-08-20
status: complete
---

# Phase 16 Plan 4: Provenance-Based Plugin Trust — topos-plugins Signing Pipeline Proof Summary

**Stood up the public `davison/topos-plugins` sibling repository with a tag-triggered ed25519 signing workflow, cut its first real signed release (v0.0.1), embedded that key in the kernel's accepted-key set, and proved end to end — including two negative cases and a fully network-isolated verification run — that an installed kernel trusts the released binary with no link-time build-manifest entry.**

## Performance

- **Duration:** 22 min (active execution; the plan's final checkpoint paused for operator approval between sessions)
- **Started:** 2026-08-20T02:37:00Z
- **Completed:** 2026-08-20T08:56:00Z (checkpoint approved and SUMMARY finalized after the pause)
- **Tasks:** 4 (checkpoint:decision resolved per operator's `key-backup` instruction; Task 1; Task 2; checkpoint:human-verify approved with a deferred live-instance item)
- **Files modified:** 3 in this repository (2 modified, 1 created) + 7 in the new `topos-plugins` sibling repository

## Accomplishments

- **`davison/topos-plugins`** (public): a genuine, minimal seed repository — `cmd/topos-plugin-demo/` (a real, contract-conformant source plugin built from `docs/plugin-contract.md`/`proto/topos/v1/plugin.proto`/the `sdk` module), a tag-triggered `.github/workflows/release.yml` that builds, signs (via the kernel repo's own `cmd/topos-provenance sign`, key passed only through the environment, never argv or a log), hashes, and publishes a GitHub Release, and a README documenting the trust boundary sentence 16-CONTEXT.md asks be preserved.
- **First real signed release, `v0.0.1`**: cut by pushing the tag, watched through a real GitHub Actions run (`success`), publishing all four required assets (binary, `.provenance.json`, `.provenance.sig`, `checksums.txt`) — independently re-verified against `checksums.txt` after download.
- **`kernel/pluginhost/provenance.go`**: `embeddedProvenanceKeys` now names the real `topos-plugins-2026a` key (previously an empty placeholder from 16-01), with an extended doc comment naming the key's custody and the D-03 rotation story. Two new tests (`TestEmbeddedProvenanceKeys_WellFormed`, `TestEmbeddedProvenanceKeys_NamesToposPlugins2026a`) pin the embedded set against a future malformed or missing edit.
- **End-to-end proof, recorded in `16-04-TRUST02-PROOF.md`**: a kernel built from this worktree (embedding the new key), installed into a throwaway prefix, verifies the downloaded `v0.0.1` release as `tier: trusted` via `GET /api/sources`, with the kernel's link-time build manifest structurally containing no entry for `topos-plugin-demo` (only the six in-repo portable plugins). Two negative cases proven: removing `.provenance.sig` falls to the external/pin-required path (never silently trusted); appending one byte to the binary makes both `topos-provenance verify` and a real launch refuse, naming the binary. Steps 1–2 were re-run inside a `unshare --net` network-isolated namespace, proving verification needs no network access at all (D-01).

## Task Commits

Each task was committed atomically:

1. **Checkpoint: confirm one-way topos-plugins artifacts** — resolved via the orchestrator's injected `key-backup` decision; no code commit (decision-only)
2. **Task 1: Stand up the topos-plugins repository with its signing workflow** — no commit in this repository (the sibling repo lives entirely outside this worktree, per plan instructions); external commits `3a8fecf` and `d2cc683` in `davison/topos-plugins`
3. **Task 2: Embed the key, cut the release, and verify the real artifact end to end** — `452c811` (feat)
4. **Checkpoint: verify the real signed release on your live installed instance** — approved by the operator, with the live-instance re-check explicitly deferred to post-merge phase UAT (see D6 above and "Deferred Verification" below)

**Plan metadata:** commit pending (this SUMMARY, parallel/worktree mode — STATE.md/ROADMAP.md are updated centrally by the orchestrator)

## Files Created/Modified

**This repository:**
- `kernel/pluginhost/provenance.go` — `embeddedProvenanceKeys` populated with the real `topos-plugins-2026a` public key; `mustDecodeProvenancePublicKey` helper
- `kernel/pluginhost/provenance_test.go` — `TestEmbeddedProvenanceKeys_WellFormed`, `TestEmbeddedProvenanceKeys_NamesToposPlugins2026a`
- `.planning/phases/16-provenance-based-plugin-trust/16-04-TRUST02-PROOF.md` — recorded end-to-end evidence

**`davison/topos-plugins` (new sibling repository, outside this worktree):**
- `go.mod`, `go.sum` — root module `github.com/davison/topos-plugins`
- `cmd/topos-plugin-demo/main.go`, `cmd/topos-plugin-demo/plugin.go` — the trivial real source plugin
- `.github/workflows/release.yml` — tag-triggered build/sign/publish workflow
- `README.md` — trust boundary documentation
- `.gitignore`

## Decisions Made

- Pinned `topos-plugins`' release workflow to the exact topos kernel commit SHA that introduced `cmd/topos-provenance`, rather than a release tag — see key-decisions above for the full reasoning; required pushing this worktree's branch to `origin` so the commit was resolvable via the Go module proxy from GitHub Actions.
- Restructured the sibling repository from a nested `demo/` submodule to a single root module with the plugin at `cmd/topos-plugin-demo/`, so a bare `go build ./...` from the repository root — the plan's own literal acceptance criterion — actually produces a file named `topos-plugin-demo` with no directory-name collision.
- Encrypted the `key-backup` decision's offline copy of the private signing key via GPG public-key encryption to the operator's own existing GPG identity, rather than inventing or prompting for a symmetric passphrase.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `cmd/topos-provenance` is not resolvable at any topos kernel release tag yet**
- **Found during:** Task 1, writing the release workflow's signing step
- **Issue:** The plan's action text specifies `go run github.com/davison/topos/cmd/topos-provenance@<pinned tag> sign`. No topos kernel release tag (`v1.2.0` is the latest) includes `cmd/topos-provenance` — it was added earlier in this same phase (16-01), still only present in unmerged worktree history.
- **Fix:** Pinned to the exact commit SHA (`e0290110026c5717d8fadebcb15f681314b9f2c1`) instead of a tag — a strictly narrower, non-floating pin — and pushed this worktree's branch to `origin/worktree-agent-aed5e1fb106997092` so the commit was fetchable by the Go module proxy from a real GitHub Actions run. Verified locally first (`go run github.com/davison/topos/cmd/topos-provenance@<sha>` resolves and runs) before wiring it into the workflow.
- **Files modified:** `topos-plugins/.github/workflows/release.yml` (external repo); `16-04-TRUST02-PROOF.md` documents the pin and the follow-up (move to a real tag once one exists)
- **Verification:** The real GitHub Actions run (`32325806543`) succeeded end to end using this pin; `grep -c 'topos-provenance'` on the workflow file returns 4, naming a fixed SHA rather than `@latest`.
- **Committed in:** external repo commit `3a8fecf`

**2. [Rule 1 - Bug] Nested `demo/` submodule made `go build ./...` from the repo root a no-op**
- **Found during:** Task 1, running the plan's own acceptance criterion (`cd ../topos-plugins && go build ./...`)
- **Issue:** The initial layout put the plugin in its own nested module (`demo/go.mod`). A bare `go build ./...` from the repository root failed with "directory prefix . does not contain main module" — the acceptance criterion could not pass as written.
- **Fix:** Restructured to a single root-level module (`go.mod` at the repository root) with the plugin package at `cmd/topos-plugin-demo/` — chosen specifically so `go build`'s default output-naming rule (last path component of the package directory) produces a file literally named `topos-plugin-demo` in the repository root with no collision against a same-named source directory (an intermediate attempt using a top-level `topos-plugin-demo/` directory directly under the repo root hit exactly that collision).
- **Files modified:** `topos-plugins/go.mod`, `topos-plugins/go.sum`, `topos-plugins/cmd/topos-plugin-demo/{main.go,plugin.go}`, `topos-plugins/.github/workflows/release.yml`, `topos-plugins/README.md`, `topos-plugins/.gitignore`
- **Verification:** `cd topos-plugins && go build ./...` exits 0 and produces `./topos-plugin-demo`; re-verified after the fix.
- **Committed in:** external repo commit `d2cc683`

**3. [Rule 3 - Blocking] The release workflow's own explanatory comment tripped its own `set -x` grep acceptance check**
- **Found during:** Task 1, running the plan's own acceptance criterion (`grep -c 'set -x'` must return 0)
- **Issue:** A doc comment stating "no `set -x` anywhere in this job" contained the literal substring being grepped for — the same class of self-tripping mechanical check 16-03 hit with a different substring.
- **Fix:** Reworded to "no shell trace flag is enabled anywhere in this job", preserving the same meaning without the literal substring.
- **Files modified:** `topos-plugins/.github/workflows/release.yml`
- **Verification:** `grep -c 'set -x' .github/workflows/release.yml` returns 0.
- **Committed in:** external repo commit `3a8fecf`

---

**Total deviations:** 3 auto-fixed (1 bug, 2 blocking)
**Impact on plan:** All three were necessary to satisfy the plan's own literal acceptance criteria or a genuine, discovered real-world constraint (no kernel release tag yet carries `cmd/topos-provenance`). No scope creep — no new behavior beyond what Task 1's action text and acceptance criteria specify; the pinning approach is more precise than the plan's own sketch, not looser.

## Issues Encountered

None beyond the three deviations documented above, each resolved within the task that surfaced it.

## Deferred Verification

The plan's final `checkpoint:human-verify` task was approved by the operator, who accepted the published one-way artifacts (`davison/topos-plugins`, public; key id `topos-plugins-2026a` with a GPG-encrypted offline backup at `~/topos-plugins-2026a.key.asc`; first tag `v0.0.1`) and the throwaway-prefix end-to-end proof recorded in `16-04-TRUST02-PROOF.md`. The operator explicitly deferred the checkpoint's steps 3–5 (adding a source on their OWN live installed instance, removing/restoring `.provenance.sig` there) to post-merge/rebuild, as part of phase UAT — their currently-installed kernel predates this plan's embedded-key change, so that specific re-check cannot pass until a kernel built from this worktree (or a later release) is actually installed. This is recorded here as coverage item D6 (`human_judgment: true`) so the verifier surfaces it rather than silently treating the checkpoint as fully closed.

## User Setup Required

**External service configuration was performed by this plan itself, not deferred to the operator:**
- Created the public GitHub repository `davison/topos-plugins` under the `davison` namespace (via the already-authenticated `gh` CLI).
- Generated the `topos-plugins-2026a` ed25519 signing keypair locally, uploaded the private key as the `TOPOS_PROVENANCE_SIGNING_KEY` GitHub Actions secret in that repository, and deleted the plaintext key file and its temp directory from local disk immediately after.
- Per the operator's `key-backup` decision, created a GPG-encrypted offline backup of the private key at `~/topos-plugins-2026a.key.asc`, encrypted to the operator's own existing GPG identity (`darren@davisononline.org`) — round-tripped (decrypt-and-compare) successfully before the plaintext was deleted. Decrypting it later requires only the operator's own already-set GPG passphrase; no new secret was invented on their behalf.

No further action is required from the operator for this plan's own scope. The one deferred item is the live-instance re-check described above (Deferred Verification), which naturally happens once this worktree is merged and a new kernel build/install is performed.

## Next Phase Readiness

- TRUST-02's proving artifact is a real signed release from a real, repeatable pipeline — not a hand-crafted test double — and the kernel's accepted-key set names it by id, embedded at compile time.
- `davison/topos-plugins` is the seed 16-05 (install-time verification via `topos-provenance verify`, D-09) and Phase 17 (the real repo split) build directly on, with a working tag-triggered signing pipeline and a documented trust boundary already proven end to end.
- The `TRUST-02` requirement ID is shared with 16-05-PLAN.md; per the shared-ID gate (#2388), it is NOT marked complete in `REQUIREMENTS.md` yet — it will flip to complete once 16-05 also finishes and produces its own SUMMARY.
- `go build ./...`, `go vet ./...`, `go test ./... -count=1`, and `make test` (including the cgo `test-signal` target) all pass locally with the newly embedded key in place.
- No blockers for 16-05. The one open item is the operator's deferred live-instance re-check (see "Deferred Verification" above), which is not a blocker for continuing execution — it is a post-merge UAT item.

---
*Phase: 16-provenance-based-plugin-trust*
*Completed: 2026-08-20*

## Self-Check: PASSED

- `kernel/pluginhost/provenance.go` — FOUND
- `kernel/pluginhost/provenance_test.go` — FOUND
- `.planning/phases/16-provenance-based-plugin-trust/16-04-TRUST02-PROOF.md` — FOUND
- Commit `452c811` (Task 2) — FOUND in `git log --oneline --all`
- `davison/topos-plugins` repository — FOUND (public, https://github.com/davison/topos-plugins)
- `davison/topos-plugins` release `v0.0.1` — FOUND, 4 assets, workflow run `32325806543` conclusion `success`
- All plan-level `<verification>` commands re-run and green: `go build ./...`, `go vet ./...`, `go test ./... -count=1`, `make test`
- Named tests `TestEmbeddedProvenanceKeys_WellFormed`, `TestEmbeddedProvenanceKeys_NamesToposPlugins2026a` in `kernel/pluginhost/provenance_test.go` — PASS
