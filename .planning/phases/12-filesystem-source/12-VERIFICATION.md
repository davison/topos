---
phase: 12-filesystem-source
verified: 2026-08-14T00:30:00Z
status: gaps_found
score: 4/6 must-have truths verified (2 blocked by unresolved Critical code-review findings)
behavior_unverified: 0
overrides_applied: 0
gaps:
  - truth: "Every filesystem item deep-links back to the document in the desktop's own file handler, or declares honestly that it can only raise (success criterion 3)"
    status: failed
    reason: "kernel/httpapi/fsopen.go's newXDGOpener runs exec.CommandContext(ctx, \"xdg-open\", path) with ctx = r.Context() — the HTTP request's own context, which net/http cancels the instant the handler returns. FilesystemOpenHandler starts the opener and returns within microseconds, so the xdg-open child is routinely SIGKILLed moments after launch, before it can hand off to (or become) the target desktop application. The kernel returns 200 {\"opened\": true} to the browser regardless — the API lies about success. This is CR-01 in 12-REVIEW.md (Critical, unresolved as of the latest commit acaad5b). No test catches it: fsopen_test.go stubs Opener entirely, and the e2e suite intentionally excludes real xdg-open assertions — this is a pure Go process-lifecycle bug, not desktop-environment variance."
    artifacts:
      - path: "kernel/httpapi/fsopen.go"
        issue: "newXDGOpener (lines 27-40) ties the child process's lifetime to the per-request context instead of context.Background()"
    missing:
      - "Decouple the xdg-open child's exec.CommandContext from r.Context() — use context.Background() (optionally with its own fixed timeout) so the child survives the HTTP response, per 12-REVIEW.md CR-01's suggested fix"
      - "A regression test that proves the opener's context is NOT the request's own context (e.g. asserting the child is unaffected by request-context cancellation), since the existing stubbed-Opener tests structurally cannot catch this class of bug"
  - truth: "MUST NOT index, serve, preview, or open any file outside the configured source root (12-01-PLAN.md hard prohibition, marked status: resolved in PLAN frontmatter)"
    status: failed
    reason: "Both re-validation points that actually touch bytes or exec a program — plugins/filesystem/item.go's resolvePath (called from fetch.go's fetchByKind before any file is opened) and kernel/httpapi/fsopen.go's inline containment check (before xdg-open is exec'd) — are purely lexical (filepath.Join + strings.HasPrefix) and never call filepath.EvalSymlinks, unlike walk.go's Match-time symlink check which correctly resolves and rejects. A file indexed as legitimate can later be swapped on disk for a symlink pointing outside the configured root (a realistic TOCTOU on a shared or network-writable mount); both the Fetch route (serves the symlink target's bytes to the browser) and the Open route (execs xdg-open against the escaped path) will follow it. This is CR-02 in 12-REVIEW.md (Critical, unresolved). It directly contradicts the guarantee docs/plugin-contract.md publishes to third-party plugin authors ('The kernel's own re-resolution on the open route re-validates the joined path stays inside the configured root before ever exec'ing anything') — verified still present verbatim at docs/plugin-contract.md:851-853, and not true of the shipped code."
    artifacts:
      - path: "plugins/filesystem/item.go"
        issue: "resolvePath (lines 70-77) has no filepath.EvalSymlinks call before its containment comparison"
      - path: "kernel/httpapi/fsopen.go"
        issue: "the inline containment check (lines 92-96) has no filepath.EvalSymlinks call before its containment comparison"
    missing:
      - "Resolve the joined path with filepath.EvalSymlinks (failing safe on resolution error) and re-check containment against the RESOLVED path, in both resolvePath and fsopen.go's inline check — mirroring walk.go's own discipline, per 12-REVIEW.md CR-02's suggested fix"
      - "A symlink-swap regression test at Fetch and Open time (not just Match/walk time) — the existing coverage (TestFetch_SourceIDEscapingTheRootIsRefusedBeforeAnyFileIsOpened, TestFilesystemOpen_PathEscapeAnswersInvalidPath) only exercises '..'-segment traversal, never a post-index symlink swap"
deferred: []
---

# Phase 12: Filesystem Source Verification Report

**Phase Goal:** The user can point topos at a folder — local or on a network mount — and see its documents in the right webspace.
**Verified:** 2026-08-14T00:30:00Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (mapped to the 5 roadmap success criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | User adds a folder as a source from the UI, recursion on/off, documents appear in the matching webspace stream with previews | ✓ VERIFIED | `web/src/lib/plugin-fields.ts` declares the `topos-plugin-filesystem` connection row (Display Name, Local Path, Include subfolders checkbox, Sync Interval Override); `web/src/lib/components/ConnectionForm.svelte:180` has the `field.kind === 'checkbox'` render branch writing to `SourceConfig.recursive`; `web/e2e/specs/12-filesystem-add-source.spec.ts` exists as the UI-driven proof. `connection-checkbox` and `plugin-fields` vitest suites pass (57/57). Preview pipeline (classify.go/render.go/fetch.go) present and unit-tested. |
| 2 | Files added/changed/removed are reflected on next sync, including on NFS/SMB mounts with no OS notification dependency | ✓ VERIFIED | `plugins/filesystem/walk.go` performs a full `filepath.WalkDir` re-walk every Match call, returns the complete current set (never partial), and relies on the kernel's existing full-replace persistence (`ReplaceWebspaceSourceItems`) for removal detection — no filesystem watcher anywhere in the design. Root-unreadable aborts with an error (never an empty set); per-entry permission errors skip and continue; context cancellation aborts. `plugins/filesystem` unit tests pass. |
| 3 | Every filesystem item deep-links back to the desktop's own file handler, or declares honestly it can only raise | ✗ FAILED | **CR-01 (unresolved):** `kernel/httpapi/fsopen.go`'s `newXDGOpener` runs `exec.CommandContext(ctx, "xdg-open", path)` with `ctx = r.Context()`. Go cancels a request's context essentially synchronously with the handler returning, and `FilesystemOpenHandler` returns immediately after starting the opener — so the child is routinely SIGKILLed before the desktop handler can take over, while the kernel reports success. See gaps below. |
| 4 | The plugin never writes to the source folder — enforced by committed guards | ✓ VERIFIED | `plugins/filesystem/readonly_test.go` is a committed Go-AST scan over every non-test file in the package, failing the build on any `os`-package write selector (`WriteFile`, `Remove`, `RemoveAll`, `Create`, `OpenFile`, `Rename`, `Mkdir`, `MkdirAll`, `Chmod`, `Chown`, `Truncate`, `Symlink`, `Link`) — mirrors the signal/paperless plugins' own precedent. `go test ./plugins/filesystem/... -count=1` passes. |
| 5 | The filesystem binary loads/syncs identically from the external plugins directory, showing the untrusted badge, before Google Drive work begins | ✓ VERIFIED | `web/e2e/specs/12-external-rehearsal.spec.ts` exists, uses `externalPluginBinaries` to link the real `topos-plugin-filesystem` binary into the external tier and assert tier `external` + untrusted badge — a real source plugin, not a fixture-only proof binary, per 12-05-PLAN.md's stated purpose. |
| — | (Cross-cutting) MUST NOT index/serve/preview/open any file outside the configured source root | ✗ FAILED | **CR-02 (unresolved):** both `plugins/filesystem/item.go`'s `resolvePath` (used by `Fetch`) and `kernel/httpapi/fsopen.go`'s inline check (used by `Open`) are lexical-only (`filepath.Join` + `strings.HasPrefix`), never calling `filepath.EvalSymlinks` — unlike `walk.go`'s Match-time check, which does. A post-index symlink swap escapes containment on both the byte-serving and exec-triggering paths. Contradicts the published contract guarantee in `docs/plugin-contract.md:851-853`. See gaps below. |

**Score:** 4/6 truths verified — 2 FAILED, both tracing to Critical findings from `12-REVIEW.md` that remain unresolved in the latest commit (`acaad5b`, "docs(12): add code review report" — no fix commit follows it).

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `plugins/filesystem/plugin.go` | SourcePlugin Describe/Match/Fetch/Health | ✓ VERIFIED | Present, substantive, `topos.v2` contract version confirmed |
| `plugins/filesystem/item.go` | relative-path source_id, folder labels, file:// deep link | ⚠️ PRESENT BUT UNSAFE | `resolvePath` present and wired into `fetch.go`, but lexical-only containment (CR-02) |
| `kernel/httpapi/fsopen.go` | loopback open route, path resolved server-side from index | ⚠️ PRESENT BUT UNSAFE / BROKEN | Route exists, wired, path resolution correct in provenance (index+config, never request) but (a) containment check is lexical-only (CR-02) and (b) the opened child process is killed before handoff (CR-01) |
| `plugins/filesystem/walk.go` | recursion-aware, symlink-safe, permission-tolerant walk | ✓ VERIFIED (with WR-01 caveat, see below) | Full re-walk, dot-file policy, symlinked-directory refusal, permission-tolerant, cap-enforced |
| `plugins/filesystem/readonly_test.go` | committed AST guard | ✓ VERIFIED | Present, substantive, passing |
| `web/src/lib/components/ui/checkbox/checkbox.svelte` | shadcn-svelte Checkbox primitive | ✓ VERIFIED | Present, wired into ConnectionForm |
| `docs/plugins/filesystem.md`, `docs/api.md`, `docs/plugin-contract.md` | operator + contract docs | ✓ VERIFIED | All updated; `docs/plugin-contract.md` documents the `file://` convention including the now-inaccurate "re-validates ... before ever exec'ing anything" guarantee (CR-02 makes this line inaccurate as shipped) |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `plugins/filesystem/item.go` | `kernel/httpapi/stream.go` | `file://` deep_link rewritten to loopback open route | ✓ WIRED | `resolveStreamLinkURL` keyed on scheme only, `TestStreamLink*` passing |
| `kernel/httpapi/routes.go` | `kernel/httpapi/fsopen.go` | `POST /api/items/{id}/open` registration | ✓ WIRED | Single registration on `/api`, confirmed by grep and passing route tests |
| `web/src/lib/components/OpenInSource.svelte` | `kernel/httpapi/fsopen.go` | fetch POST against same-origin `/api/` link | ✓ WIRED | `isLocalExecLink` branch present, vitest suite passing |
| `web/src/lib/plugin-fields.ts` | `web/src/lib/components/ConnectionForm.svelte` | field descriptor `kind` selects render branch | ✓ WIRED | `checkbox` kind branch present and tested |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Kernel open-route + stream-link unit tests | `go test ./kernel/httpapi/ -run 'TestFilesystemOpen\|TestStreamLink' -count=1` | ok | ✓ PASS (but does not cover CR-01/CR-02 — stubbed opener, no symlink-swap or context-cancellation case) |
| Filesystem plugin unit tests | `cd plugins/filesystem && CGO_ENABLED=0 go test ./... -count=1` | ok | ✓ PASS |
| internal/audit icon/provenance contract | `go test ./internal/audit/ -count=1` | ok | ✓ PASS |
| Web checkbox + plugin-fields unit tests | `npm --prefix web run test -- connection-checkbox plugin-fields` | 57/57 passed | ✓ PASS |
| Full repo build | `go build ./...` | clean | ✓ PASS |

No full e2e run was executed in this verification pass (would require booting a real kernel/browser harness); the e2e spec files were confirmed to exist and their assertions read against the described criteria, consistent with SUMMARY claims and the code review's own file-by-file read.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| SRC-04 | 12-01 through 12-05 (all declare `requirements: [SRC-04]`) | User can add a local/network filesystem folder as a source; documents appear with previews and deep links, synced via stat-diff polling | ⚠️ PARTIALLY SATISFIED | Add-source UI, scope/preview pipeline, and stat-diff (full-replace) sync are solid. The "deep links" clause is not actually satisfied: CR-01 breaks the open action in practice, and CR-02 means the read-only/containment guarantee the deep-link and preview machinery both depend on is violated. `REQUIREMENTS.md` already marks SRC-04 "Complete" (traceability table, line 78) — this mark is premature given the two unresolved Critical findings and should be reverted pending a gap-closure plan. |

No orphaned requirements found — SRC-04 is the only requirement mapped to Phase 12 and it is claimed by all five plans.

### Anti-Patterns Found

None. No debt markers (`TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`) found in any file touched by this phase. `go build ./...` is clean.

### Related, Non-Blocking Finding (carried forward from 12-REVIEW.md, not scored as a gap)

**WR-01 (Warning, unresolved):** `plugins/filesystem/walk.go`'s in-tree symlink check (line 140) resolves a symlinked file's target via `filepath.EvalSymlinks` but compares it against `cleanRoot` (`filepath.Clean(root)`, never itself resolved). If the *configured root itself* sits behind a symlink or bind mount (e.g. `~/Documents` → `~/dotfiles/Documents`), every legitimately in-tree symlinked file is silently dropped from the corpus (`skipped` isn't even incremented) — an under-inclusion bug, not a security defect, but a real regression risk for a common desktop pattern. No phase must-have explicitly covers symlinked-root behavior, so this does not change the overall status, but it is a known, documented, unresolved bug in shipped code and should be closed alongside CR-01/CR-02.

### Human Verification Required

None — both blocking findings (CR-01, CR-02) are demonstrable directly from the source (process-lifecycle logic and absence of `EvalSymlinks` calls), not runtime-only behavior requiring a human to exercise a live desktop environment.

### Gaps Summary

Phase 12 built a substantively complete filesystem source plugin: the UI flow, extension/preview classification, recursive stat-diff-style sync, read-only enforcement, and the external-plugin-tier rehearsal are all present, wired, and covered by passing tests — success criteria 1, 2, 4, and 5 hold up under inspection.

However, the phase's own code review (`12-REVIEW.md`, committed as the latest commit on this branch, with no fix commit following it) found two unresolved Critical defects that this verification independently confirmed by reading the current source:

1. **CR-01** — the kernel's `xdg-open` open action is wired to the HTTP request's own context, so the launched application is routinely killed within milliseconds of being started, while the kernel reports success. This breaks success criterion 3 in practice for the common case.
2. **CR-02** — the path-containment re-validation on both the Fetch and Open routes is lexical-only (no `filepath.EvalSymlinks`), so a post-index symlink swap can escape the configured source root and disclose or open files the user never consented to expose. This breaks the phase's own hard prohibition ("MUST NOT index, serve, preview, or open any file outside the configured source root") and contradicts the guarantee published in `docs/plugin-contract.md` for third-party plugin authors — a guarantee Phase 14's Google Drive plugin author would reasonably rely on for their own local-path handling.

Both findings are precisely in the two areas the phase's own threat model calls its sharpest surfaces, both have documented, concrete fixes already written out in `12-REVIEW.md`, and neither is covered by any existing test (by design/oversight, not by intentional scope exclusion). This phase should not be marked complete, and `REQUIREMENTS.md`'s existing "Complete" mark for SRC-04 should be reverted, until a closure plan lands both fixes (and, ideally, WR-01 alongside them) with regression tests that specifically exercise the request-context-cancellation and symlink-swap cases the current suite cannot catch.

---

_Verified: 2026-08-14T00:30:00Z_
_Verifier: Claude (gsd-verifier)_
