---
phase: 03-email-in-the-webspace
plan: 02
subsystem: api
tags: [bluemonday, html-sanitization, go-imap, read-only-enforcement, ast-scan, svelte]

requires:
  - phase: 03-email-in-the-webspace
    provides: "plan 03-01's plugins/proton tracer (Describe/Match/Fetch/Health against Proton Mail Bridge, host-pinned TLS, plain-text-only Fetch, client.go's dial seam)"
affects: [03-04 (search UI can now assume email items render real HTML bodies), any future phase touching plugins/proton or DetailPane.svelte]

tech-stack:
  added: [github.com/microcosm-cc/bluemonday v1.0.27 (plugins/proton — already running at this exact version in plugins/silverbullet)]
  patterns:
    - "Email HTML sanitize-and-wrap pipeline (plugins/proton/body.go): bluemonday policy narrowly scoped (style attribute allowed only on a named element set, CSS property allowlist restricted to presentational declarations) plus themeStyle/WrapDocument copied verbatim from plugins/silverbullet/render.go — reuses the existing text/html rendition route and sandboxed iframe branch with zero kernel or DetailPane branch changes"
    - "IMAP read-only AST scan (plugins/proton/readonly_test.go): mirrors plugins/paperless/readonly_test.go's ast.Inspect/*ast.SelectorExpr mechanism, targeting IMAP-mutating client identifiers instead of non-GET net/http identifiers, with a negative-control fixture proving the scanner is non-vacuous"
    - "Wire-level transcript proof (plugins/proton/imap_transcript_test.go): a recording TCP relay tees the client-to-server byte stream in front of a real github.com/emersion/go-imap server+backend/memory instance, asserting EXAMINE/BODY.PEEK[ presence and mutating-command absence without needing a live Bridge"
    - "Client.dial seam (established in 03-01) used as the test substitution point for the transcript test — no production code path changed to make this testable"

key-files:
  created:
    - plugins/proton/render_test.go
    - plugins/proton/imap_transcript_test.go
    - plugins/proton/readonly_test.go
    - plugins/proton/outbound_hosts_test.go
    - plugins/proton/live_bridge_test.go
  modified:
    - plugins/proton/body.go
    - plugins/proton/plugin.go
    - plugins/proton/go.mod
    - plugins/proton/go.sum
    - web/src/lib/components/DetailPane.svelte
    - .gitignore

key-decisions:
  - "extractPart(raw, wantContentType) shared internal helper for both PlainTextPart and HTMLPart — one MIME-walk implementation, bounded identically by maxPartBytes/maxParts, parameterized by content type rather than two independently-maintained loops"
  - "emailSanitizePolicy allows the style attribute only on a named element set (p, span, div, td, th, h1-h6, li, a) with a presentational-only CSS property allowlist — deliberately narrower than bluemonday's own published HTML-email demo (which allows style Globally() and calls that unsafe in its own comment), per 03-RESEARCH.md Pitfall 3"
  - "disallowedIMAPIdents (readonly_test.go) covers both message-mutating (Store/UidStore/Expunge/Move/UidMove/Append/Copy/UidCopy) and mailbox-mutating (Create/Delete/Rename/Subscribe/Unsubscribe) client identifiers, broader than the plan's own worked example — Delete and Rename apply to mailboxes, not just messages, and are equally a PLUG-02 violation if ever called"
  - "imap_transcript_test.go seeds two DIFFERENT mailbox leaf names (Labels/AlphaTeam, Labels/BetaTeam) sharing one Message-Id, rather than two mailboxes with the identical leaf name, so the dedup assertion can verify both distinct labels survive on the single merged item — the plan's own wording ('both mailbox leaf names in its labels') requires two distinct leaf names to be meaningful"
  - "Live Bridge login was deliberately NOT attempted this session (Proof 4 implemented but not run) — 03-01-SUMMARY.md already documents a 'no such user' rejection followed by Bridge's own rate-limiting from two prior attempts; retrying now would extend that lockout without new information, so the live run is left for the user to execute once the Bridge account credential is corrected"

requirements-completed: [SRC-01]

coverage:
  - id: D1
    description: "HTML email bodies render inline in the detail pane's existing sandboxed iframe, sanitized by a narrowly-scoped bluemonday policy; plain-text-only emails render as text unchanged from 03-01"
    requirement: "SRC-01"
    verification:
      - kind: unit
        ref: "plugins/proton/render_test.go — TestRenderSanitizedEmail_* (script/onerror stripping, javascript: href stripping, color-preserved/position-dropped, style scoped to named elements, remote image preserved, ordinary HTML survives), TestWrapDocument_*"
        status: pass
      - kind: unit
        ref: "npm --prefix web run check (0 errors, 0 warnings) and npm --prefix web run test (44 tests passed)"
        status: pass
    human_judgment: true
    rationale: "The plan's own <verify> block requires a human-check against a running kernel/webUI with a real HTML email opened in the detail pane — no live Bridge email was available this session (see Known Stubs / WINDOWS.md), so the visual rendering claim rests on unit tests of the sanitizer/wrapper plus a code-level review of the Fetch dispatch, not an actual click-through."
  - id: D2
    description: "Sender (item.group_label) renders as its own truncated line in the detail pane header, between the title and the date/labels row, omitted entirely when empty"
    requirement: "SRC-01"
    verification:
      - kind: unit
        ref: "npm --prefix web run check — 0 errors, 0 warnings"
        status: pass
    human_judgment: false
  - id: D3
    description: "The never-mark-read guarantee is mechanically enforced: an AST scan fails the build on any IMAP-mutating identifier reference, with a negative control proving the scanner is non-vacuous"
    requirement: "SRC-01"
    verification:
      - kind: unit
        ref: "plugins/proton/readonly_test.go — TestPluginIssuesNoIMAPMutatingCommands (zero offenses in production files + negative-control fixture reports at least one offense)"
        status: pass
    human_judgment: false
  - id: D4
    description: "A wire-level transcript test proves EXAMINE/BODY.PEEK[ are the only mailbox-open/body-fetch commands issued, and no mutating command substring appears, across a full Describe/Match/Fetch/Health cycle against a local fake IMAP server — no live Bridge required"
    requirement: "SRC-01"
    verification:
      - kind: integration
        ref: "plugins/proton/imap_transcript_test.go — TestIMAPTranscript_ExamineAndPeekOnly"
        status: pass
    human_judgment: false
  - id: D5
    description: "The outbound-host allowlist permits the configured Bridge host (any case, any port), localhost, and loopback IP literals, and refuses a foreign hostname, a foreign non-loopback IP literal, and an empty host, every refusal via errors.Is(err, ErrForeignHost)"
    requirement: "SRC-01"
    verification:
      - kind: unit
        ref: "plugins/proton/outbound_hosts_test.go — TestAllowHost_PredicateTable"
        status: pass
    human_judgment: false
  - id: D6
    description: "An environment-gated live integration test reads a real message's \\Seen flag through a second, independent IMAP connection before and after a full plugin Match+Fetch cycle, asserting it is unchanged"
    requirement: "SRC-01"
    verification:
      - kind: integration
        ref: "plugins/proton/live_bridge_test.go — TestSeenFlagUnchanged_LiveBridge (skips cleanly, not fails, when WEBSPACES_PROTON_LIVE_IT is unset — confirmed this session)"
        status: unknown
    human_judgment: true
    rationale: "The live run itself was not executed this session — 03-01 already documented the Bridge account's LOGIN rejection ('no such user') and subsequent rate-limiting from two prior attempts. Retrying now would risk extending that lockout for no new information. The test is implemented exactly as specified and is ready to run once the user corrects the Bridge account credential; a human must run it and confirm the result."

duration: ~55min
completed: 2026-07-31
status: complete
---

# Phase 3 Plan 2: HTML Email Rendering and Mechanically-Enforced Read-Only Guarantees Summary

**Sanitized HTML email bodies now render inline through the detail pane's existing iframe (bluemonday policy narrowly scoped to presentational CSS, reusing plugins/silverbullet's WrapDocument verbatim), with the sender shown under the subject — and SRC-01's never-mark-read guarantee is now proven four independent ways: a build-failing AST scan with a negative control, a no-live-Bridge wire transcript test, a host-allowlist predicate table, and an environment-gated live \Seen-unchanged integration test.**

## Performance

- **Duration:** ~55 min
- **Completed:** 2026-07-31
- **Tasks:** 2
- **Files modified:** 6 modified/created in Task 1, 5 created/modified in Task 2 (11 total, some overlapping)

## Accomplishments

- `plugins/proton/body.go` now extracts both the `text/plain` and `text/html` inline parts (shared `extractPart` helper, identical `maxPartBytes`/`maxParts` bounds), and adds `emailSanitizePolicy` (bluemonday, style attribute scoped to a named element set with a presentational-only CSS property allowlist), `RenderSanitizedEmail`, and `themeStyle`/`WrapDocument` copied verbatim from `plugins/silverbullet/render.go`.
- `plugins/proton/plugin.go`'s `fetchFull` now returns a sanitized, wrapped `text/html` rendition when an HTML part exists, falling through to 03-01's plain-text-only behavior otherwise — no kernel change needed, since `text/html` was already on `kernel/httpapi/item.go`'s rendition MIME allowlist for SilverBullet.
- `web/src/lib/components/DetailPane.svelte` renders `item.group_label` as its own truncated, `title`-attributed line directly under the subject, before the date/labels row — omitted entirely when empty, so paperless/SilverBullet detail panes are visually unchanged.
- Four independent, committed proofs that the plugin never mutates the user's mailbox: `readonly_test.go`'s build-failing AST scan (with a negative-control fixture), `imap_transcript_test.go`'s wire-level transcript test against a local fake IMAP server (no live Bridge needed), `outbound_hosts_test.go`'s host-allowlist predicate table, and `live_bridge_test.go`'s environment-gated live `\Seen`-unchanged integration test.
- The transcript test also proves the Message-ID dedup/label-merge path end to end: a message present in two keyword-matching mailboxes (different leaf names) yields exactly one `Item` carrying both leaf names in its labels.

## Task Commits

1. **Task 1: HTML email bodies render inline, with the sender in the detail pane** — `7e79fee` (feat)
2. **Task 2: Prove read-only — wire transcript, AST scan, host allowlist, live \Seen check** — `1828684` (test)

**Plan metadata:** (this commit, pending)

## Files Created/Modified

- `plugins/proton/body.go` — `extractPart` shared MIME-walk helper (backs both `PlainTextPart` and new `HTMLPart`), `emailSanitizePolicy`/`newEmailSanitizePolicy`/`RenderSanitizedEmail`, `themeStyle`/`WrapDocument` (copied from `plugins/silverbullet/render.go`)
- `plugins/proton/plugin.go` — `fetchFull` extended to sanitize+wrap an extracted HTML part into a `text/html` rendition; gofmt whitespace fix (trailing blank line)
- `plugins/proton/render_test.go` — sanitizer/wrapper fixture tests: script/event-handler stripping, javascript: href stripping, presentational-vs-behavioural CSS property allow/deny, style-attribute element scoping, remote image preserved (CSP handles the tracking-pixel risk), ordinary HTML survives, theme injection, no re-sanitization of the wrapper's own `<style>`
- `plugins/proton/readonly_test.go` — `disallowedIMAPIdents` (message- and mailbox-mutating client identifiers), `scanFileForIMAPMutation`/`scanSourceForIMAPMutation`/`scanASTForIMAPMutation`, `TestPluginIssuesNoIMAPMutatingCommands` with an embedded negative-control fixture
- `plugins/proton/imap_transcript_test.go` — `recordingRelay` (mutex-guarded tee of the client-to-server TCP direction), `newTestIMAPServer` (real `go-imap` server + `backend/memory`, seeded with two labeled mailboxes sharing one message), `TestIMAPTranscript_ExamineAndPeekOnly`
- `plugins/proton/outbound_hosts_test.go` — `TestAllowHost_PredicateTable` against the proton client's `allowHost`
- `plugins/proton/live_bridge_test.go` — `TestSeenFlagUnchanged_LiveBridge`, gated on `WEBSPACES_PROTON_LIVE_IT=1` plus the three `PROTON_BRIDGE_*` env vars; helper functions for an independent second IMAP connection's before/after flag snapshots
- `plugins/proton/go.mod`/`go.sum` — added `github.com/microcosm-cc/bluemonday v1.0.27` as a direct dependency (already running at this exact version in `plugins/silverbullet`)
- `web/src/lib/components/DetailPane.svelte` — sender line inserted between the `h2` title and the date/labels row
- `.gitignore` — added `/plugins/proton/proton` (stray build artifact from a bare `go build ./...`, matching the existing entries for the sibling plugins)

## Decisions Made

- `extractPart(raw, wantContentType)` is a single shared internal helper backing both `PlainTextPart` and the new `HTMLPart`, rather than two independently-duplicated MIME-walk loops — same `maxPartBytes`/`maxParts` bounds enforced once, not twice.
- `emailSanitizePolicy`'s style-attribute allowlist is scoped to a named element set (`p, span, div, td, th, h1-h6, li, a`) with a presentational-only CSS property list (`color, background-color, font-weight, font-style, font-size, font-family, text-align, text-decoration, padding, margin, border, width, height`) — deliberately narrower than bluemonday's own published HTML-email demo, which allows `style` `Globally()` and says in its own comment that this is not safe (03-RESEARCH.md Pitfall 3).
- `disallowedIMAPIdents` in `readonly_test.go` covers 13 identifiers (message-mutating: `Store`, `UidStore`, `Expunge`, `Move`, `UidMove`, `Append`, `Copy`, `UidCopy`; mailbox-mutating: `Create`, `Delete`, `Rename`, `Subscribe`, `Unsubscribe`) — broader than the plan's own six-identifier worked example (`Store/Expunge/Move/Append/Delete/Copy`), since the UID variants and the mailbox-level operations are equally real IMAP mutations this plugin must never issue.
- `imap_transcript_test.go` seeds two mailboxes with *different* leaf names (`AlphaTeam`, `BetaTeam`) sharing one Message-ID, rather than two mailboxes with an identical leaf name — the plan's acceptance criterion ("both mailbox leaf names in its labels") is only a meaningful assertion when the two leaf names actually differ.
- The live Bridge test (Proof 4) was implemented exactly as specified but **not executed live** this session — see Deviations/Issues below.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added `/plugins/proton/proton` to `.gitignore`**
- **Found during:** Task 1, post-build check
- **Issue:** A bare `go build ./...` inside `plugins/proton` drops a stray top-level binary named after the directory (matching the pre-existing, already-documented pattern for `plugins/paperless/paperless`, `plugins/silverbullet/silverbullet`, `plugins/mock/mock`), which `git status` then reports as untracked.
- **Fix:** Added `/plugins/proton/proton` to `.gitignore`, matching the existing entries and their documented rationale (stray binaries from a bare build, not the `make build`-produced artifacts).
- **Files modified:** `.gitignore`
- **Verification:** `git status --short` no longer reports the binary as untracked.
- **Committed in:** `7e79fee`

**2. [Rule 1 - Bug] `github.com/microcosm-cc/bluemonday` initially resolved as an indirect dependency**
- **Found during:** Task 1, after `go get`
- **Issue:** `go get github.com/microcosm-cc/bluemonday@v1.0.27` placed the requirement in the `// indirect` block even though `body.go` imports it directly, because `go get` couldn't fully resolve the module graph in this workspace-mode module (the same pre-existing `sdk` no-remote limitation 03-01-SUMMARY.md documents for `go mod tidy`).
- **Fix:** Manually moved the `bluemonday` require line into the direct-dependency block in `go.mod`, matching how `go-imap`/`go-message` are already declared.
- **Files modified:** `plugins/proton/go.mod`
- **Verification:** `go build ./...` and `go test ./...` both pass; `go vet ./...` and `gofmt -l .` report no issues.
- **Committed in:** `7e79fee`

### Notable Deferred Item (not a code defect)

**Live Bridge test (Proof 4) implemented but not executed.** `WEBSPACES_PROTON_LIVE_IT=1 go test -run TestSeenFlagUnchanged_LiveBridge` was **not run this session**. 03-01-SUMMARY.md already documents that the real Bridge account's IMAP `LOGIN` was rejected twice ("no such user", then "too many login attempts" after Bridge's own rate-limiting kicked in) — a Bridge-account credential issue on the user's environment, not a code defect. Per this wave's explicit instruction not to retry live logins in a loop or hammer the Bridge, a third attempt was deliberately skipped. `TestSeenFlagUnchanged_LiveBridge` is fully implemented, compiles, and correctly **skips (not fails)** when its gating env vars are unset (confirmed this session) — it is ready to run once the user corrects the Bridge account username in Bridge → Settings. This is tracked as an open item in `.planning/WINDOWS.md` (three `unrun-verify` entries: this live test, and both plan `<verify>` human-check steps that require a live email in the running app).

---

**Total deviations:** 2 auto-fixed (1 blocking — gitignore; 1 bug — go.mod dependency placement); 1 deferred live-verification item (documented, not a code fix, tracked in WINDOWS.md).
**Impact on plan:** No scope creep. Every code-level acceptance criterion in both tasks passes automated verification. The one unmet item (the live `\Seen`-unchanged confirmation and the two human-check UI walkthroughs) is blocked by the same pre-existing, already-documented Bridge credential issue from 03-01, not by anything in this plan's code.

## Issues Encountered

- No live Bridge email was available this session to visually confirm HTML rendering in the running app (Task 1's human-check) or to confirm unread status in the real Proton client (Task 2's human-check) — both are recorded as `unrun-verify` entries in `.planning/WINDOWS.md` rather than silently skipped.
- Web frontend `node_modules` was not present in this worktree at session start; `npm install` was run once (110 packages) before `npm run check`/`npm run test` — a normal per-worktree setup step, not a plan deviation.

## User Setup Required

None new. Once the Bridge account's `PROTON_BRIDGE_USER` value is corrected (the outstanding action 03-01-SUMMARY.md already identified), the user can run:

```bash
WEBSPACES_PROTON_LIVE_IT=1 \
PROTON_BRIDGE_ADDR=<addr> PROTON_BRIDGE_USER=<user> PROTON_BRIDGE_PASS=<pass> \
go test -run TestSeenFlagUnchanged_LiveBridge -v ./plugins/proton/...
```

and separately do a live click-through (open a real HTML email in the running app's detail pane; check the same email's read status in Proton's own web/mobile client) to close out the two `unrun-verify` WINDOWS.md entries.

## Next Phase Readiness

- `plugins/proton` now has a real HTML rendering path and four independent, committed read-only proofs — the email source plugin's SRC-01 code-level surface is complete and build/test/vet/fmt-clean.
- Three `unrun-verify` items remain open in `.planning/WINDOWS.md`, all blocked on the same pre-existing Bridge-account credential issue, not on anything this plan changed.
- Plans 03-03 (FTS5 search backend) and 03-04 (search UI) are unaffected by this plan's scope and can proceed independently — no shared files were touched beyond `DetailPane.svelte`'s header block (sender line), which 03-04's search-results reuse of `StreamRow.svelte` does not touch.

## Self-Check: PASSED

- FOUND: plugins/proton/body.go, plugins/proton/plugin.go, plugins/proton/render_test.go, plugins/proton/readonly_test.go, plugins/proton/imap_transcript_test.go, plugins/proton/outbound_hosts_test.go, plugins/proton/live_bridge_test.go
- FOUND: web/src/lib/components/DetailPane.svelte contains `item.group_label` guarded by `{#if}`
- FOUND: .gitignore contains `/plugins/proton/proton`
- FOUND: commit 7e79fee in `git log --oneline`
- FOUND: commit 1828684 in `git log --oneline`

---
*Phase: 03-email-in-the-webspace*
*Completed: 2026-07-31*
