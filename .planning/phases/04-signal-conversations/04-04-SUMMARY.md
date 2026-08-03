---
phase: 04-signal-conversations
plan: 04
subsystem: signal-plugin
tags: [signal, deep-link, sgnl, e164, regexp, security-register]

# Dependency graph
requires:
  - phase: 04-signal-conversations
    provides: "Signal plugin, conversation digests, and the deep-link builder (04-01/04-02/04-03) whose contact-form output this plan corrects"
provides:
  - "conversationDeepLink() emitting a literal '+' for valid E.164 numbers instead of a percent-encoded '%2B' that Signal Desktop's shipped validator rejects"
  - "Allowlist (validate-and-refuse) discipline replacing the prior escape-the-value approach for URI construction from source-derived data"
  - "An end-to-end shape guard in scripts/signal-readonly-smoke.sh asserting real stream responses carry only accepted link shapes"
  - "A corrected 04-SECURITY.md register (T-04-14 superseded, T-04-17/18/19 added) describing the mitigation that actually ships"
affects: ["05-whatsapp (if a similar sgnl-style deep-link scheme is ever built, allowlist-not-escape is the established pattern)"]

# Actuals (#2632)
actuals:
  tokens: 4880
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Allowlist validate-and-refuse for URI construction from source-derived, semantically-constrained values (E.164), replacing escape-the-value — an allowlist cannot be defeated by a producer/consumer encoder mismatch, which is exactly how the escaped form failed"
    - "Regex pattern mirrored byte-for-byte from a traced third-party consumer's own shipped validator, with provenance recorded in a doc comment, rather than an independently-invented pattern"

key-files:
  created: []
  modified:
    - plugins/signal/deeplink.go
    - plugins/signal/deeplink_test.go
    - scripts/signal-readonly-smoke.sh
    - .planning/phases/04-signal-conversations/04-SECURITY.md

key-decisions:
  - "conversationDeepLink refuses (falls back to bare 'sgnl://') rather than escapes any e164 value that doesn't match ^\\+[1-9][0-9]{1,14}$ — mirrors Signal Desktop's own shipped validator exactly, traced statically from the installed app.asar bundle"
  - "T-04-14 kept in the security register (not deleted) and marked superseded, pointing at T-04-17, so the register records the supersession rather than losing history of a mitigation that no longer exists"
  - "Re-synced the developer's live index from this worktree (built all four required plugin binaries locally, sourced the main repo's gitignored .env via WEBSPACES_ENV_FILE) rather than deferring the whole task's automated verification to the orchestrator — only the final server restart before human-check is deferred"

requirements-completed: [SRC-02]

coverage:
  - id: D1
    description: "conversationDeepLink emits the E.164 verbatim (literal '+') for a valid private conversation number, and falls back to the bare 'sgnl://' form for any value that doesn't match Signal's own E.164 shape"
    requirement: "SRC-02"
    verification:
      - kind: unit
        ref: "plugins/signal/deeplink_test.go#TestDeepLink_PlusSignEmittedLiterally"
        status: pass
      - kind: unit
        ref: "plugins/signal/deeplink_test.go#TestDeepLink_E164BoundaryMatrix"
        status: pass
      - kind: unit
        ref: "plugins/signal/deeplink_test.go#TestDeepLink_NonE164FallsBackToBareForm"
        status: pass
    human_judgment: false
  - id: D2
    description: "The developer's live index (~/.local/share/webspaces/index.db) holds no Signal contact-form link in the rejected shape after a rebuild + re-sync"
    requirement: "SRC-02"
    verification:
      - kind: integration
        ref: "sqlite3 query: items where source_type='signal' and deep_link glob 'sgnl://signal.me/#p/+*' (count>0 -> PASS)"
        status: pass
      - kind: integration
        ref: "sqlite3 query: items where source_type='signal' and deep_link glob 'sgnl://signal.me/#p/*' and not glob '.../+*' (count=0 -> PASS)"
        status: pass
    human_judgment: false
  - id: D3
    description: "A real stream response (not a fixture) carries only contact-form links with a literal '+' immediately after '#p/'"
    requirement: "SRC-02"
    verification:
      - kind: integration
        ref: "scripts/signal-readonly-smoke.sh (run with SIGNAL_SMOKE_KEYWORD=Dad against a real 1:1 conversation: 105/105 links checked, all passed)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Clicking 'Open in Signal' on a 1:1 Signal digest raises Signal Desktop and lands on that contact's conversation, with no 'sgnl:// link didn't make sense' error modal — this is the only check that closes G-04-1"
    verification: []
    human_judgment: true
    rationale: "Requires a human eye on the developer's own running Signal Desktop window-raise and navigation; this executor confirmed the link's byte shape is correct (literal '+', no percent sign) via the live sync and a real stream response, but cannot itself observe Signal Desktop's UI. Deferred per this plan's own live_environment_note and Task 1's <human-check>."
    coverage_note: "Automated evidence de-risks this to near-certainty (root cause traced through the exact installed Signal Desktop binary in .planning/debug/sgnl-link-didnt-make-sense.md, live index carries the corrected shape, a real stream response was independently verified). The remaining gap is purely the visual window-raise confirmation."

duration: ~15min
completed: 2026-08-04
status: complete
---

# Phase 04 Plan 04: Signal contact-form deep-link fix (G-04-1 gap closure) Summary

**Replaced `plugins/signal/deeplink.go`'s escape-the-value approach with allowlist validate-and-refuse, closing the root cause of G-04-1: Signal Desktop's shipped validator requires a literal `+` in the contact-form fragment, and the previous `url.QueryEscape`-based encoder mangled it to `%2B`, which Signal's link pipeline never percent-decodes before validating.**

## Performance

- **Duration:** ~15 min
- **Tasks:** 3/3 completed
- **Files modified:** 4
- **Commits:** 3 task commits + this metadata commit

## Accomplishments

- `conversationDeepLink` now emits `sgnl://signal.me/#p/+15551234567` verbatim for a valid E.164 — no percent-encoding, no transformation of any kind — and falls back to the bare `sgnl://` scheme for anything that doesn't match `^\+[1-9][0-9]{1,14}$` (mirrored byte-for-byte from Signal Desktop's own shipped validator, traced statically from the installed `app.asar` bundle)
- Deleted `encodePhoneFragment` and its `net/url` import entirely; the compiler enforces the import is gone
- Rewrote `plugins/signal/deeplink_test.go`: `TestDeepLink_PlusSignEmittedLiterally` and `TestDeepLink_NonE164FallsBackToBareForm` replace the two tests that enshrined the old percent-encoding behaviour; `TestDeepLink_PrivateWithE164UsesContactForm` now compares against a hard-coded literal instead of re-deriving the expectation from the code under test; `TestDeepLink_E164BoundaryMatrix` pins 14 named boundary/refusal/acceptance cases (one-digit, leading-zero, 15/16-digit length boundary, no-leading-plus, six URI metacharacters, leading/trailing whitespace padding, two-digit acceptance boundary)
- Added an end-to-end contact-form shape guard to `scripts/signal-readonly-smoke.sh`, asserting a real stream response's link URLs — not a fixture, not the builder re-asserted against itself — carry a literal `+`; verified non-vacuous locally (105/105 real contact-form links checked against a real 1:1 conversation keyword)
- Rebuilt the plugin, stopped the running server that was still serving the old binary, and re-synced the developer's live index (`~/.local/share/webspaces/index.db`, 111 Signal rows) from this worktree; both index shape queries flipped from FAIL (pre-fix) to PASS
- Corrected `.planning/phases/04-signal-conversations/04-SECURITY.md`: `T-04-14` marked superseded (kept, not deleted) rather than continuing to claim an escaping mitigation this plan removed; added `T-04-17` (the allowlist control that ships), `T-04-18` (phone-number-in-URI exposure, accepted, local-only), `T-04-19` (no dependency installed by this plan)
- `make test` and `make build` both exit 0 across the whole repo (kernel, sdk, every plugin including the cgo Signal module, and the SvelteKit frontend build)

## Task Commits

Each task was committed atomically:

1. **Task 1: Literal '+' survives from the builder to the live index and into Signal Desktop** - `eebfa2d` (fix)
2. **Task 2: Widen the validator's test matrix and add the end-to-end shape guard** - `3a92206` (test)
3. **Task 3: Correct the security register and sweep the whole build** - `487f33b` (docs)

_Note: worktree-isolated execution — STATE.md/ROADMAP.md updates are deferred to the orchestrator per this plan's parallel-execution contract; no separate plan-metadata commit is made from this worktree._

## Files Created/Modified

- `plugins/signal/deeplink.go` - Replaced `encodePhoneFragment`/escape approach with `e164Pattern`/`isValidE164` allowlist validate-and-refuse; rewrote doc comments to correct the closed-A4 assumption and record the traced root cause
- `plugins/signal/deeplink_test.go` - Replaced the two tests that enshrined percent-encoding; hardcoded the contact-form expectation; added a 14-case named boundary matrix
- `scripts/signal-readonly-smoke.sh` - Added the contact-form literal-'+' shape guard over a real stream response, with a documented rationale for why this check exists above the unit-test layer
- `.planning/phases/04-signal-conversations/04-SECURITY.md` - T-04-14 superseded; T-04-17/18/19 added; audit trail table gets a 2026-08-04 row

## Decisions Made

- **conversationDeepLink refuses rather than escapes.** Any e164 value that doesn't match `^\+[1-9][0-9]{1,14}$` gets the bare `sgnl://` fallback instead of an escaped or transformed contact-form link. This mirrors Signal Desktop's own shipped validator exactly (traced statically from the installed `app.asar` bundle in `.planning/debug/sgnl-link-didnt-make-sense.md`), and an allowlist cannot be defeated by a producer/consumer encoder mismatch — precisely how the escape-based version failed (we percent-encoded `+`; Signal never percent-decodes the fragment before validating).
- **T-04-14 kept, not deleted, in the security register.** Marked "superseded" with a pointer to T-04-17, so the register's history of a since-removed mitigation stays visible rather than silently disappearing.
- **Re-synced the live index from this worktree.** Built all four required plugin binaries (`paperless`, `silverbullet`, `proton`, `signal`) locally in the worktree and sourced the main repo's gitignored `.env` via `WEBSPACES_ENV_FILE` (rather than copying the secrets file into the worktree) so the sync ran against the real, absolute-path config and index. Only the final `webspaces serve` restart before the human-check is deferred to the orchestrator, per this plan's `live_environment_note` — the orchestrator will restart the server from the merged main checkout before the human clicks "Open in Signal".

## Deviations from Plan

None — plan executed exactly as written. One environmental note: the plan's Task 1 assumed a running `webspaces serve` might exist; one was in fact found running (from the main repo's checkout, not this worktree) and was stopped per the plan's own instruction before the rebuild-and-resync sequence, as expected.

## Issues Encountered

- `run-with-env.sh`'s `plugins.dir = "plugins"` config value resolves relative to the invoking binary's working directory, and this worktree's `bin/plugins/` only had the freshly-built `webspaces-plugin-signal` binary (from `make signal`) until the other three source plugins (`paperless`, `silverbullet`, `proton`) were also built locally — the first sync attempt failed with a "plugin binary not found" error for `paperless`. Resolved by building all four required plugin binaries in the worktree before re-running sync; no code change, environment-setup only.
- This worktree does not carry the main repo's gitignored `.env` (git worktrees only check out tracked files). Resolved by pointing `run-with-env.sh` at the main repo's `.env` via its documented `WEBSPACES_ENV_FILE` override, rather than copying secrets into the worktree.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- G-04-1's automated evidence is fully in place: root cause traced through the exact installed Signal Desktop binary, unit boundary matrix passing, live index carrying the corrected shape (111 rows re-synced), and an independently-verified real stream response (105/105 contact-form links checked against a real 1:1 conversation).
- **Outstanding: the human-check in Task 1** — clicking "Open in Signal" on a live 1:1 Signal digest and visually confirming Signal Desktop raises and navigates to that contact's conversation with no error modal. This is the only check that can close G-04-1 per the plan's own `<verification>` section; it could not be performed by this executor (per `live_environment_note`, worktree-isolated execution cannot drive the developer's desktop UI). The orchestrator should rebuild from the merged main checkout, restart `webspaces serve`, and prompt the developer to perform this check before marking G-04-1 resolved.
- No blockers for future phases. The allowlist validate-and-refuse pattern established here is a reusable precedent if Phase 5 (WhatsApp) ever needs a similarly source-derived value embedded in a URI.

---
*Phase: 04-signal-conversations*
*Completed: 2026-08-04*
