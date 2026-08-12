---
phase: 03-email-in-the-webspace
plan: 07
subsystem: security
tags: [go-modules, cve-remediation, dependency-audit, bluemonday, golang.org-x-net, static-analysis]

# Dependency graph
requires:
  - phase: 03-email-in-the-webspace
    provides: "plugins/proton's IMAP sync/render pipeline (03-01 through 03-06) — RenderSanitizedEmail's bluemonday-based sanitize-and-wrap path, the concrete surface this plan's CVE fix applies to"
provides:
  - "plugins/proton/go.mod declares golang.org/x/net at v0.56.0 (past the CVE-2024-45338 / GO-2024-3333 fix boundary of v0.33.0), with golang.org/x/text raised to the version that requires"
  - "A repo-wide, permanent audit test (internal/audit/module_pins_test.go) that fails any future `go test ./...` run if any of the six go.work modules declares a dependency below a documented security floor"
  - "A recorded, evidence-backed correction to 03-REVIEW.md CR-01 / 03-VERIFICATION.md's exploitability claim: the affected tokenizer has never been compiled into a shipped binary (workspace MVS already selected v0.56.0), but the declared manifest contract was still the correct thing to fix"
affects: [phase-04-signal, phase-05-whatsapp, any-future-dependency-bump]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Repo-wide go.mod audit via filesystem walk + line-based require-block parsing (parenthesised and standalone shapes), mirroring the existing AST-based outbound-egress scanner's repoRoot/skipDirs/shouldSkipDir idiom in internal/audit"
    - "Non-vacuity proven twice: a committed negative-control fixture (testdata/vulnerable_pin_go.mod.txt) plus a temporary, uncommitted floor-raise inversion against the live manifests, restored before commit"

key-files:
  created:
    - internal/audit/module_pins_test.go
    - internal/audit/testdata/vulnerable_pin_go.mod.txt
  modified:
    - plugins/proton/go.mod
    - plugins/proton/go.sum
    - plugins/proton/render_test.go
    - internal/audit/doc.go

key-decisions:
  - "Bumped golang.org/x/net to v0.56.0 (the version go.work's MVS already selects for plugins/proton) rather than the minimum-sufficient v0.33.0 or the other five modules' v0.53.0, because declaring v0.53.0 would have measurably *lowered* the workspace-selected version and altered the shipped plugin binary — the declared contract is repaired without disturbing the compiled dependency set"
  - "plugins/silverbullet and the root module were measured already declaring golang.org/x/net v0.53.0 (already past the fix boundary) and therefore correctly received no change — the gap's second `missing` item is discharged by measurement, not deferred"
  - "The repo-wide floor guard's version comparator (compareGoVersions) is a small local vMAJOR.MINOR.PATCH helper, not a dependency on an external semver package, to avoid adding a dependency to the root module in the very plan whose subject is dependency hygiene"
  - "npm --prefix web ci was run to restore this worktree's missing node_modules (lockfile-pinned, no new package resolved) so the frontend suite's acceptance criteria could be executed — an environment-setup step, not a new package install"

requirements-completed: [SRC-01]

coverage:
  - id: D1
    description: "plugins/proton/go.mod's declared golang.org/x/net requirement is raised past the CVE-2024-45338 / GO-2024-3333 fix boundary (v0.26.0 -> v0.56.0), with golang.org/x/text dragged to v0.38.0, via go get (not go mod tidy, which cannot run cleanly in this module)"
    requirement: "SRC-01"
    verification:
      - kind: unit
        ref: "cd plugins/proton && GOWORK=off go list -m golang.org/x/net (module-scoped resolution now v0.56.0)"
        status: pass
      - kind: unit
        ref: "cd plugins/proton && go list -m golang.org/x/net (workspace-selected resolution unchanged at v0.56.0)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The sanitizer's rendered output is unchanged for all eight pre-existing TestRenderSanitizedEmail_*/TestWrapDocument_* cases, and two new tests pin the nil/empty input boundary across the tokenizer swap"
    requirement: "SRC-01"
    verification:
      - kind: unit
        ref: "plugins/proton/render_test.go#TestRenderSanitizedEmail_EmptyAndNilInputYieldNoOutput"
        status: pass
      - kind: unit
        ref: "plugins/proton/render_test.go#TestWrapDocument_NilFragmentStillYieldsADocument"
        status: pass
      - kind: unit
        ref: "cd plugins/proton && go test -run 'TestRenderSanitizedEmail_|TestWrapDocument_' -count=1 -v ./... (10/10 PASS)"
        status: pass
    human_judgment: false
  - id: D3
    description: "A repo-wide audit test fails if any of the six workspace modules declares a dependency below a documented security floor, discovers at least six manifests, handles both go.mod require shapes, and is proven non-vacuous by both a committed fixture and a temporary live inversion"
    requirement: "SRC-01"
    verification:
      - kind: unit
        ref: "internal/audit/module_pins_test.go#TestNoModuleDeclaresAKnownVulnerablePin"
        status: pass
      - kind: unit
        ref: "internal/audit/module_pins_test.go#TestPinScanner_FixtureReportsTheBelowFloorDeclaration"
        status: pass
    human_judgment: false
  - id: D4
    description: "No new module path enters any manifest, no dependency is added to the root module, and the other five workspace modules plus go.work/go.work.sum are provably untouched"
    requirement: "SRC-01"
    verification:
      - kind: unit
        ref: "diff of go.mod require-path sets before/after (empty diff, exit 0)"
        status: pass
      - kind: unit
        ref: "git diff --exit-code go.work go.work.sum go.mod go.sum sdk/go.mod sdk/go.sum plugins/silverbullet/go.mod plugins/paperless/go.mod plugins/mock/go.mod"
        status: pass
    human_judgment: false

duration: 20min
completed: 2026-07-31
status: complete
---

# Phase 3 Plan 07: CVE-2024-45338 declared-pin bump + repo-wide dependency-floor guard Summary

**Bumped plugins/proton's declared `golang.org/x/net` past the CVE-2024-45338 HTML-tokenizer DoS fix boundary (v0.26.0 -> v0.56.0, matching what the go.work workspace already builds), and added a permanent repo-wide audit test that fails any future `go test ./...` if any of the six workspace modules regresses below a documented dependency-security floor.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-07-31
- **Tasks:** 2
- **Files modified:** 6 (2 created, 4 modified)

## Accomplishments

- `plugins/proton/go.mod`'s declared `golang.org/x/net` requirement moved from v0.26.0 (CVE-2024-45338 / GO-2024-3333 affected) to v0.56.0, with `golang.org/x/text` dragged from v0.16.0 to v0.38.0, via `go get` — never `go mod tidy`, which this module's own go.mod comment records as unable to run cleanly (the workspace-local `sdk` module has no published remote).
- Measured, not assumed: the module-scoped resolution (`GOWORK=off go list -m golang.org/x/net`) moved v0.26.0 -> v0.56.0 (the fix, closing the gap for any module-scoped consumer/scanner/SBOM/third-party plugin author). The workspace-selected resolution (`go list -m golang.org/x/net`) stayed v0.56.0 -> v0.56.0 (the shipped `make build` plugin binary is provably unchanged).
- Added `TestRenderSanitizedEmail_EmptyAndNilInputYieldNoOutput` and `TestWrapDocument_NilFragmentStillYieldsADocument` to `plugins/proton/render_test.go`, pinning nil/empty input-boundary behaviour across the tokenizer swap. All eight pre-existing sanitize/wrap tests plus these two new ones pass unchanged.
- Added `internal/audit/module_pins_test.go`: a permanent, repo-wide guard (`TestNoModuleDeclaresAKnownVulnerablePin`) that walks all six `go.work` modules' `go.mod` files and fails if any declares a version below a documented security floor (seeded with `golang.org/x/net` at v0.33.0). This is the standing, mechanical form of the gap's one-time "also check `plugins/silverbullet`" instruction.
- Proved the guard non-vacuous twice: a committed negative-control fixture (`testdata/vulnerable_pin_go.mod.txt`) that the scanner reports an offence against, and a temporary, uncommitted floor-raise inversion (v99.0.0) against the live manifests that made the repo-wide walk fail, naming all six real module files by their declared version — then restored.
- Added no dependency anywhere: the version comparator is a small local helper, not a new import.

## Task Commits

Each task was committed atomically:

1. **Task 1: the Proton module's declared HTML tokenizer is past the CVE fix boundary — with the shipped build and the rendered output both proven unchanged** - `9b01797` (fix)
2. **Task 2: no workspace module can declare a below-floor dependency again — a repo-wide guard with a negative control** - `33f3f7d` (test)

## Files Created/Modified

- `plugins/proton/go.mod` - `golang.org/x/net` v0.26.0 -> v0.56.0, `golang.org/x/text` v0.16.0 -> v0.38.0 (both indirect requirement version strings only; no module path added/removed/renamed)
- `plugins/proton/go.sum` - 3 checksum lines added for the new selections (obtained via `go get` from the module cache, never hand-written)
- `plugins/proton/render_test.go` - added `TestRenderSanitizedEmail_EmptyAndNilInputYieldNoOutput` and `TestWrapDocument_NilFragmentStillYieldsADocument`
- `internal/audit/module_pins_test.go` (new) - `minimumModuleVersions` floor table, `scanGoModForBelowFloorPins`, `compareGoVersions`/`parseGoVersion` helpers, `TestNoModuleDeclaresAKnownVulnerablePin`, `TestPinScanner_FixtureReportsTheBelowFloorDeclaration`
- `internal/audit/testdata/vulnerable_pin_go.mod.txt` (new) - negative-control fixture manifest with a below-floor `golang.org/x/net` requirement (standalone-`require` shape), a safe requirement, and a requirement absent from the floor table
- `internal/audit/doc.go` - extended the package-doc closing parenthetical to name the new declared-dependency-floor invariant alongside the existing outbound-host allowlist

## Evidence: exploitability correction (Step 1 / Step 5 measurements)

| Measurement | Before (Step 1) | After (Step 5) |
|---|---|---|
| Module-scoped (`GOWORK=off go list -m golang.org/x/net`) | `golang.org/x/net v0.26.0` | `golang.org/x/net v0.56.0` |
| Workspace-selected (`go list -m golang.org/x/net`) | `golang.org/x/net v0.56.0` | `golang.org/x/net v0.56.0` (unchanged) |

Verbatim `GOWORK=off go build ./...` failure at Step 1 (proof no module-scoped build of this plugin exists today, so the affected tokenizer was never compiled into a shipped artefact):

```
main.go:15:2: no required module provides package github.com/davison/webspaces/sdk; to add it:
	go get github.com/davison/webspaces/sdk
plugin.go:19:2: no required module provides package github.com/davison/webspaces/sdk/gen/webspaces/v1; to add it:
	go get github.com/davison/webspaces/sdk/gen/webspaces/v1
/opt/go/pkg/mod/github.com/emersion/go-imap@v1.2.1/utf7/utf7.go:7:2: missing go.sum entry for module providing package golang.org/x/text/encoding (imported by github.com/emersion/go-imap/utf7); to add:
	go get github.com/emersion/go-imap/utf7@v1.2.1
/opt/go/pkg/mod/github.com/emersion/go-imap@v1.2.1/utf7/decoder.go:8:2: missing go.sum entry for module providing package golang.org/x/text/transform (imported by github.com/emersion/go-imap/utf7); to add:
	go get github.com/emersion/go-imap/utf7@v1.2.1
/opt/go/pkg/mod/github.com/emersion/go-imap@v1.2.1/commands/authenticate.go:11:2: missing go.sum entry for module providing package github.com/emersion/go-sasl (imported by github.com/emersion/go-imap/client); to add:
	go get github.com/emersion/go-imap/client@v1.2.1
main.go:13:2: no required module provides package github.com/hashicorp/go-plugin; to add it:
	go get github.com/hashicorp/go-plugin
plugin.go:16:2: no required module provides package google.golang.org/grpc/codes; to add it:
	go get google.golang.org/grpc/codes
plugin.go:17:2: no required module provides package google.golang.org/grpc/status; to add it:
	go get google.golang.org/grpc/status
```

**Six-module declared-version audit table (Task 1 Step 2):**

| Module | Declared `golang.org/x/net` (before this plan) | Safe (>= v0.33.0)? | Action |
|---|---|---|---|
| `.` (root) | v0.53.0 | Yes | None — measured already-safe |
| `sdk` | v0.53.0 | Yes | None — measured already-safe |
| `plugins/mock` | v0.53.0 | Yes | None — measured already-safe |
| `plugins/paperless` | v0.53.0 | Yes | None — measured already-safe |
| `plugins/silverbullet` | v0.53.0 | Yes | None — measured already-safe (discharges the gap's "check silverbullet" instruction by measurement) |
| `plugins/proton` | v0.26.0 | **No** | Bumped to v0.56.0 (this plan) |

`plugins/silverbullet` and the root module were measured already-safe and therefore correctly received no change — this is the explicit decision (measurement, not deferral) that discharges the gap's second `missing` item.

**Target-version rationale:** v0.56.0 was chosen over the minimum-sufficient v0.33.0 and over the other five modules' v0.53.0 because it is the version `go.work`'s MVS already selects for `plugins/proton`'s build. Declaring v0.53.0 was measured at plan time to *lower* the workspace-selected version from v0.56.0 to v0.53.0 — still safe with respect to this CVE, but a change to the compiled dependency set that a pure security-hygiene bump has no business making. v0.56.0 repairs the declared contract while leaving the shipped binary provably byte-identical.

**On the retained stale `go.sum` lines:** the pre-existing `golang.org/x/net v0.26.0` and `golang.org/x/text v0.16.0` lines remain in `plugins/proton/go.sum` after this bump. `go get` appends hashes and never prunes; `go mod tidy` — the only command that prunes — cannot run cleanly against this module in isolation (its own go.mod comment records why: the workspace-local `sdk` module has no published remote). `go.sum` is an integrity ledger, not a version selector, so these retained lines are inert and not an incomplete fix.

## Evidence: pre-existing and new render tests (Task 1, gap `missing` item 3)

```
--- PASS: TestRenderSanitizedEmail_StripsScriptElement (0.00s)
--- PASS: TestRenderSanitizedEmail_StripsJavascriptSchemeHref (0.00s)
--- PASS: TestRenderSanitizedEmail_PreservesColorDropsPosition (0.00s)
--- PASS: TestRenderSanitizedEmail_StyleAttributeScopedToNamedElements (0.00s)
--- PASS: TestRenderSanitizedEmail_RemoteImagePreservedButHarmless (0.00s)
--- PASS: TestRenderSanitizedEmail_OrdinaryHTMLSurvives (0.00s)
--- PASS: TestWrapDocument_InjectsThemeStyleAndPreservesFragment (0.00s)
--- PASS: TestWrapDocument_StyleNeverReprocessedThroughSanitizer (0.00s)
--- PASS: TestRenderSanitizedEmail_EmptyAndNilInputYieldNoOutput (0.00s)
--- PASS: TestWrapDocument_NilFragmentStillYieldsADocument (0.00s)
PASS
ok  	github.com/davison/webspaces/plugins/proton	0.010s
```

Full `plugins/proton` suite (unaffected tests included): `TestSeenFlagUnchanged_LiveBridge` reported `--- SKIP` (environment-blocked, unchanged), all others PASS.

## Evidence: repo-wide guard non-vacuity (Task 2)

Negative-control fixture, verbatim offence output:

```
=== RUN   TestPinScanner_FixtureReportsTheBelowFloorDeclaration
    module_pins_test.go:207: negative control reported: [testdata/vulnerable_pin_go.mod.txt: golang.org/x/net declares v0.26.0, below the required floor v0.33.0 (CVE-2024-45338 / GO-2024-3333)]
--- PASS: TestPinScanner_FixtureReportsTheBelowFloorDeclaration (0.00s)
```

Temporary floor-raise inversion (`minimumModuleVersions["golang.org/x/net"].MinVersion` raised to `v99.0.0`, uncommitted, restored immediately after), verbatim output proving the repo-wide walk reaches every real manifest:

```
=== RUN   TestNoModuleDeclaresAKnownVulnerablePin
    module_pins_test.go:272: declared-dependency-floor violation(s) found — a workspace module declares a version below its documented security floor:
        ../../go.mod: golang.org/x/net declares v0.53.0, below the required floor v99.0.0 (TEMPORARY INVERSION — DO NOT COMMIT)
        ../../plugins/mock/go.mod: golang.org/x/net declares v0.53.0, below the required floor v99.0.0 (TEMPORARY INVERSION — DO NOT COMMIT)
        ../../plugins/paperless/go.mod: golang.org/x/net declares v0.53.0, below the required floor v99.0.0 (TEMPORARY INVERSION — DO NOT COMMIT)
        ../../plugins/proton/go.mod: golang.org/x/net declares v0.56.0, below the required floor v99.0.0 (TEMPORARY INVERSION — DO NOT COMMIT)
        ../../plugins/silverbullet/go.mod: golang.org/x/net declares v0.53.0, below the required floor v99.0.0 (TEMPORARY INVERSION — DO NOT COMMIT)
        ../../sdk/go.mod: golang.org/x/net declares v0.53.0, below the required floor v99.0.0 (TEMPORARY INVERSION — DO NOT COMMIT)
--- FAIL: TestNoModuleDeclaresAKnownVulnerablePin (0.00s)
```

After restoring the floor to v0.33.0, the full `internal/audit` suite returned to green (4/4 PASS): `TestPinScanner_FixtureReportsTheBelowFloorDeclaration`, `TestNoModuleDeclaresAKnownVulnerablePin`, `TestNoForeignEgressOutsideSanctionedClient`, `TestScanner_FixtureReportsBothOffenseKinds`.

## Full regression evidence (both tasks)

- Repo root: `CGO_ENABLED=0 go build ./...`, `CGO_ENABLED=0 go vet ./...`, `CGO_ENABLED=0 go test ./... -count=1` — all exit 0, all packages `ok` or `[no test files]`.
- Each of `sdk`, `plugins/paperless`, `plugins/silverbullet`, `plugins/proton`, `plugins/mock`: `go build ./... && go test ./... -count=1` — all `ok`.
- `npm --prefix web run test` — 4 test files, 72 tests, all passed.
- `npm --prefix web run check` (svelte-check) — `746 FILES 0 ERRORS 1 WARNINGS 1 FILES_WITH_PROBLEMS` (the single pre-existing `SearchBox.svelte` `state_referenced_locally` warning, out of scope, left alone).
- `git diff --exit-code web/` — clean; this plan touches no frontend file.

## Decisions Made

- Bumped `golang.org/x/net` to v0.56.0 rather than v0.33.0 or v0.53.0, matching the workspace-selected version exactly, so the declared-contract repair leaves the compiled dependency set byte-identical (measured, see rationale table above).
- `plugins/silverbullet` and the root module needed no change — measured already-safe at v0.53.0, discharging the gap's "check silverbullet" instruction by measurement rather than deferral.
- The repo-wide floor guard's version comparator is a small local helper (`compareGoVersions`/`parseGoVersion`), not a dependency on an external semver package — adding one in a dependency-hygiene plan would be a self-contradicting trade.
- Ran `npm --prefix web ci` to restore this worktree's missing `node_modules` (a fresh git worktree checkout does not carry it, and it is not itself tracked in git). This is restoring the exact lockfile-pinned dependency tree, not installing a new/candidate package, so it is not subject to the package-legitimacy checkpoint in the deviation rules — it was necessary to execute the plan's own frontend acceptance criteria (`npm --prefix web run test`, `npm --prefix web run check`).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Restored missing `web/node_modules` via `npm --prefix web ci`**
- **Found during:** Task 1 Step 7 (full regression set, frontend leg)
- **Issue:** This worktree's `web/` directory had no `node_modules` (a fresh worktree checkout does not carry git-ignored install output), so `npm --prefix web run test` and `npm --prefix web run check` failed with `command not found` before any plan-relevant code could be exercised.
- **Fix:** Ran `npm --prefix web ci`, which installs exactly the versions pinned in the existing `package-lock.json` — no new package name was resolved or added, and no manifest file changed.
- **Files modified:** None tracked (`web/node_modules` is gitignored; `git status --short web/` was empty before and after).
- **Verification:** `npm --prefix web run test` (72/72 passed) and `npm --prefix web run check` (0 ERRORS) both then ran to completion as the plan's acceptance criteria require.
- **Committed in:** n/a (no tracked files changed by this fix).

---

**Total deviations:** 1 auto-fixed (1 blocking, environment-setup only)
**Impact on plan:** No scope creep — this restored the ability to run the plan's own already-specified frontend verification commands; it introduced no new dependency, no new package, and no file change of any kind.

## Issues Encountered

None beyond the node_modules environment gap above.

## Still Outstanding — NOT Closed By This Plan (verbatim, per plan's own instruction)

The four `human_verification` items in `03-VERIFICATION.md` all require a live, currently-authenticating Proton Mail Bridge account. The Bridge account rejected LOGIN with "no such user" and then rate-limited (`03-01-SUMMARY.md`, "Notable Live-Environment Finding"), unchanged across three verification passes. That is a credential/environment correction, not a code defect, and is out of scope here. Truth 2 of the phase (`\Seen` unchanged) therefore remains PRESENT_BEHAVIOR_UNVERIFIED for the same reason; this plan issues no IMAP command and changes no IMAP call path, so `TestPluginIssuesNoIMAPMutatingCommands` and `TestIMAPTranscript_ExamineAndPeekOnly` remain the standing proof and both still pass (confirmed in this session's full regression run). These are already tracked as open items 1-3 in `.planning/WINDOWS.md` from prior phase work; this plan added no new entry because it introduces no new stub, skip, or unrun verification.

## Also Deliberately Not Closed (owners named, not silently dropped)

`03-REVIEW.md`'s WR-01…WR-05 and IN-01…IN-04, recorded non-blocking in `03-VERIFICATION.md` and "left to the project's own backlog process"; `.planning/REQUIREMENTS.md`'s status bookkeeping for SRC-01, owned by the re-verify/seal step per the 03-05 and 03-06 precedent; `COVERAGE.md`, unchanged because no external API surface changed.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `plugins/proton`'s declared dependency contract is repaired and the shipped build is provably unchanged; the repo-wide floor guard now prevents recurrence in any of the six workspace modules on every `go test` run.
- The four Proton Bridge credential-blocked `human_verification` items remain the sole open blocker for a complete phase-3 close-out; they require a working Bridge login, not further code changes.
- No blockers for Phase 4 (Signal) or Phase 5 (WhatsApp) planning introduced by this plan.

## Self-Check: PASSED

All created/modified files confirmed present on disk (`plugins/proton/go.mod`, `plugins/proton/render_test.go`, `internal/audit/module_pins_test.go`, `internal/audit/testdata/vulnerable_pin_go.mod.txt`, this SUMMARY.md); all three commit hashes (`9b01797`, `33f3f7d`, `46ef4b2`) confirmed present in `git log --oneline --all`.

---
*Phase: 03-email-in-the-webspace*
*Completed: 2026-07-31*
