---
phase: 01-first-webspace-end-to-end
plan: 06
subsystem: security
tags: [go, net-http, ast-audit, go-plugin, paperless]

requires:
  - phase: 01-first-webspace-end-to-end (plan 01)
    provides: "The paperless plugin's Client (plugins/paperless/client.go) and its outbound-transmission prohibition (must_haves.prohibitions #3)"
provides:
  - "Host-pinned outbound policy on the paperless client (ErrForeignHost sentinel, allowHost predicate, wired into both CheckRedirect and DialContext)"
  - "A committed runtime test suite proving the allowlist holds under cross-host redirect, same-host redirect, redirect-cap, and foreign-pagination-next scenarios"
  - "A repo-wide, non-vacuous AST audit (internal/audit) that fails the build on any foreign URL literal or outbound HTTP construction outside the one sanctioned egress point"
  - "A traceable verified_by record on 01-01-PLAN.md's third prohibition"
affects: [phase-3-email, phase-4-signal, phase-5-whatsapp]

tech-stack:
  added: []
  patterns:
    - "Outbound host allowlist enforced at both hook points an HTTP client can originate a connection from (CheckRedirect for the common source-controlled case, DialContext as the backstop) rather than trusting request-building code alone"
    - "Repo-wide AST audit as a test-only package (internal/audit) using filepath.WalkDir + go/parser, mirroring plugins/paperless/readonly_test.go's mechanism, to cross all three go.work modules in one pass"
    - "Negative-control fixture (testdata/*.go.txt) proving a static-analysis test is non-vacuous, not merely that today's tree happens to be clean"

key-files:
  created:
    - plugins/paperless/outbound_hosts_test.go
    - internal/audit/doc.go
    - internal/audit/outbound_hosts_test.go
    - internal/audit/testdata/foreign_host_violation.go.txt
  modified:
    - plugins/paperless/client.go
    - .planning/phases/01-first-webspace-end-to-end/01-01-PLAN.md

key-decisions:
  - "The host predicate excludes port from its comparison — the configured host is the user's own paperless-ngx instance, and a reverse proxy in front of it may legitimately move between ports on that same host; the prohibition is about foreign hosts, not foreign ports"
  - "Same-host redirect test uses a distinct trailing-slash path (not the literal same path plus one more slash) because Go's own net/url reference resolution collapses repeated slashes as part of RFC 3986 dot-segment removal — a literal double-slash target is normalized away by Go before the client's guard ever sees it, making that exact byte sequence untestable"
  - "Foreign-pagination-next test asserts existing splitNextURL behavior (already re-pins path+query, dropping any absolute host) as a committed guarantee rather than changing that code, since it was already correct"

requirements-completed: [PLUG-02, SRC-04]

coverage:
  - id: D1
    description: "plugins/paperless/client.go refuses foreign hosts at both CheckRedirect (before a redirect is followed) and DialContext (before any connection opens), with ErrForeignHost as the sentinel error"
    requirement: "PLUG-02"
    verification:
      - kind: unit
        ref: "plugins/paperless/outbound_hosts_test.go#TestAllowHost_PredicateTable"
        status: pass
      - kind: unit
        ref: "plugins/paperless/outbound_hosts_test.go#TestDocument_CrossHostRedirect_Refused"
        status: pass
      - kind: unit
        ref: "plugins/paperless/outbound_hosts_test.go#TestDocument_SameHostRedirect_StillFollowed"
        status: pass
      - kind: unit
        ref: "plugins/paperless/outbound_hosts_test.go#TestDocument_RedirectCap_StopsLooping"
        status: pass
    human_judgment: false
  - id: D2
    description: "Foreign pagination next URLs are re-pinned to the configured base host rather than followed"
    requirement: "SRC-04"
    verification:
      - kind: unit
        ref: "plugins/paperless/outbound_hosts_test.go#TestListDocuments_ForeignNextURL_RepinnedToBaseHost"
        status: pass
    human_judgment: false
  - id: D3
    description: "Repo-wide AST audit fails the build on outbound HTTP construction outside plugins/paperless/client.go or any foreign absolute URL literal in shipped code, proven non-vacuous by a negative-control fixture"
    verification:
      - kind: unit
        ref: "internal/audit/outbound_hosts_test.go#TestNoForeignEgressOutsideSanctionedClient"
        status: pass
      - kind: unit
        ref: "internal/audit/outbound_hosts_test.go#TestScanner_FixtureReportsBothOffenseKinds"
        status: pass
    human_judgment: false
  - id: D4
    description: "01-01-PLAN.md's outbound-transmission prohibition names the tests that enforce it (verified_by), closing gap G-01-6"
    verification:
      - kind: other
        ref: "grep -q verified_by AND grep -q outbound_hosts_test.go on .planning/phases/01-first-webspace-end-to-end/01-01-PLAN.md"
        status: pass
    human_judgment: false

duration: ~15min
completed: 2026-07-28
status: complete
---

# Phase 01 Plan 06: Host-pin paperless client outbound egress Summary

**Host-pinned paperless client (ErrForeignHost sentinel at CheckRedirect + DialContext) plus a repo-wide AST audit that fails the build on any foreign URL literal or unsanctioned outbound HTTP construction**

## Performance

- **Duration:** ~15 min
- **Completed:** 2026-07-28T12:13:47Z
- **Tasks:** 3
- **Files modified:** 6 (4 created, 2 modified)

## Accomplishments
- `plugins/paperless/client.go` now refuses to dial or follow a redirect to any host that is neither the configured `base_url` hostname (any port) nor a loopback address / `localhost`, via a shared `allowHost` predicate wired into both `http.Transport.DialContext` and `http.Client.CheckRedirect`
- `plugins/paperless/outbound_hosts_test.go` proves this holds: a predicate table (permit/refuse cases), a cross-host redirect refused via `errors.Is(err, ErrForeignHost)`, a same-host redirect still succeeding, the re-implemented 10-hop redirect cap, and foreign pagination `next` URLs re-pinned to the base host — plus every pre-existing test in the module untouched and passing
- `internal/audit` (new root-module package) walks the entire repository via `go/parser`/`go/ast` and fails the build on any foreign absolute URL literal or outbound HTTP construction (`http.Get`, `http.NewRequest`, `http.Client{}`, etc.) anywhere except `plugins/paperless/client.go`, proven non-vacuous by a `testdata` negative-control fixture containing both offense kinds
- `01-01-PLAN.md`'s third prohibition (the outbound-transmission MUST NOT) now carries a `verified_by` list naming both new test files, closing gap G-01-6 and making the phase's `verification: test` claim traceable

## Task Commits

Each task was committed atomically:

1. **Task 1: Host-pin the paperless client's outbound policy and prove it** - `337a72f` (feat, tdd: test written first and confirmed failing via `go vet` before implementation)
2. **Task 2: Repo-wide AST audit — no foreign hosts, one sanctioned egress point** - `7082eb3` (feat)
3. **Task 3: Make the prohibition's verification claim traceable in 01-01-PLAN.md** - `567526a` (docs)

**Plan metadata:** (this commit, following SUMMARY.md creation)

## Files Created/Modified
- `plugins/paperless/client.go` - added `ErrForeignHost`, `Client.baseHost`, `Client.allowHost`, and wired the predicate into `DialContext`/`CheckRedirect`
- `plugins/paperless/outbound_hosts_test.go` - new: predicate table, cross-host redirect refusal, same-host redirect success, redirect-cap, foreign-next re-pinning
- `internal/audit/doc.go` - new: package clause + rationale for a test-only package directory
- `internal/audit/outbound_hosts_test.go` - new: repo-wide AST scanner and its own tests
- `internal/audit/testdata/foreign_host_violation.go.txt` - new: negative-control fixture (one URL-literal offense, one construction offense)
- `.planning/phases/01-first-webspace-end-to-end/01-01-PLAN.md` - amended only the third prohibition entry with `verified_by` and a closure note

## Decisions Made
- Port excluded from the host-equality comparison (reverse proxies legitimately move ports on the same host); documented in-code so a future reader doesn't "tighten" it into a false failure
- Same-host redirect test targets a distinct trailing-slash path rather than literally "the same path plus one more slash", because Go's `net/url` reference resolution (`resolvePath`) collapses repeated slashes during redirect-following before this client's guard would ever see them — confirmed empirically (the double-slash version looped through the redirect cap and failed) and documented in the test's own comment
- Left `splitNextURL`'s pre-existing re-pinning logic unchanged; the foreign-next test asserts it as a now-committed guarantee rather than treating it as new code

## Deviations from Plan

None - plan executed exactly as written. The one implementation-detail discovery (double-slash redirect targets get collapsed by Go's own URL resolution) was resolved within Task 1's own TDD loop before any commit, using a same-host-different-path redirect target that exercises the identical guard logic; documented above and in the test file itself rather than treated as a deviation from the plan's intent.

## Issues Encountered
None beyond the redirect-path normalization detail noted above, resolved during the RED/GREEN cycle prior to committing.

## User Setup Required
None - no external service configuration required. Every test in this plan is hermetic (httptest servers on loopback, AST parsing from disk); no `PAPERLESS_URL`/`PAPERLESS_TOKEN` needed.

## Next Phase Readiness
- Gap G-01-6 is closed; Phase 01's outbound-transmission prohibition is now enforced by construction and asserted by committed, non-vacuous tests
- The `internal/audit` pattern (test-only package, `filepath.WalkDir` + `go/parser`, negative-control fixture) is reusable for future repo-wide invariants as later phases add more source plugins (email/Signal/WhatsApp) that will each need their own sanctioned egress point added to `internal/audit/outbound_hosts_test.go`'s allowlist
- No blockers

---
*Phase: 01-first-webspace-end-to-end*
*Completed: 2026-07-28*

## Self-Check: PASSED

All created/modified files verified present on disk; all three task commit hashes (337a72f, 7082eb3, 567526a) verified present in git log.
