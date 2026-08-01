---
phase: 03-email-in-the-webspace
plan: 10
subsystem: source-plugin
tags: [go, proton, imap, deep-link, url-encoding, gap-closure]

# Dependency graph
requires:
  - phase: 03-email-in-the-webspace
    provides: 03-01 through 03-09's Proton plugin (toItem, matched, HasRenderableText, Snippet precedent)
provides:
  - webmailSearchDeepLink/encodeKeywordFragment (plugins/proton/deeplink.go), a percent-encoded, rune-capped All Mail search deep link replacing the unaddressable label-name path
  - Corrected operator-facing descriptions of webmail_base_url in config.example.toml and kernel/config/types.go
affects: [03-UAT, 03-VERIFICATION]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Deep link construction isolated in its own file (deeplink.go) with an exact-match table test, mirroring the package's existing Snippet rune-cap precedent"
    - "net/url QueryEscape + '+' -> '%20' substitution for a fragment keyword that must decode identically under both form-style and straight percent-decoders"

key-files:
  created:
    - plugins/proton/deeplink.go
    - plugins/proton/deeplink_test.go
  modified:
    - plugins/proton/plugin.go
    - config.example.toml
    - kernel/config/types.go

key-decisions:
  - "webmailSearchDeepLink trims the subject once, tests HasRenderableText on the trimmed value (reusing body.go's existing predicate rather than re-testing for emptiness), and uses that same trimmed value for capping/encoding"
  - "Rune cap applied via rune-slice conversion identical to Snippet's own technique, never byte slicing, so a multi-byte subject longer than the cap always decodes back to valid UTF-8"
  - "pathEscapeSegment and its sole net/url import in plugin.go removed outright rather than left dead beside the replacement"

requirements-completed: [SRC-01]

coverage:
  - id: D1
    description: "An email item's deep link is a webmail search over the account's All Mail view for the message's subject, not an unaddressable label-name path"
    requirement: SRC-01
    verification:
      - kind: unit
        ref: "plugins/proton/deeplink_test.go#TestWebmailSearchDeepLink_Table"
        status: pass
      - kind: unit
        ref: "plugins/proton/deeplink_test.go#TestToItem_DeepLinkIsAWebmailSearchNotALabelPath"
        status: pass
    human_judgment: false
  - id: D2
    description: "A message with no subject yields the All Mail view with no search fragment; a hostile subject cannot restructure the URL's host, path or parameter structure"
    requirement: SRC-01
    verification:
      - kind: unit
        ref: "plugins/proton/deeplink_test.go#TestWebmailSearchDeepLink_Table (absent/empty/whitespace-only and hostile-punctuation rows)"
        status: pass
    human_judgment: false
  - id: D3
    description: "The search keyword is bounded by rune count and always decodes back to valid UTF-8 for a multi-byte subject longer than the cap"
    requirement: SRC-01
    verification:
      - kind: unit
        ref: "plugins/proton/deeplink_test.go#TestWebmailSearchDeepLink_OverCapMultiByteSubjectStaysValidUTF8"
        status: pass
    human_judgment: false
  - id: D4
    description: "Fidelity stays LINK_FIDELITY_ANCHORED, asserted rather than assumed"
    requirement: SRC-01
    verification:
      - kind: unit
        ref: "plugins/proton/deeplink_test.go#TestToItem_FidelityRemainsAnchored"
        status: pass
    human_judgment: false
  - id: D5
    description: "No absolute URL literal exists in the plugin's shipped source, and the repo-wide egress scan stays green over the new file"
    requirement: SRC-01
    verification:
      - kind: unit
        ref: "internal/audit/outbound_hosts_test.go#TestNoForeignEgressOutsideSanctionedClient"
        status: pass
    human_judgment: false
  - id: D6
    description: "Both operator-facing descriptions of webmail_base_url (config.example.toml, kernel/config/types.go) describe the All Mail search link the plugin now builds and record why a label-name path was replaced"
    requirement: SRC-01
    verification:
      - kind: other
        ref: "grep -q 'All Mail'/'internal id' config.example.toml kernel/config/types.go — see acceptance-criteria greps in this SUMMARY"
        status: pass
    human_judgment: false
  - id: D7
    description: "Proton webmail actually honours the produced hash-based All Mail search form live, against the user's real account"
    verification: []
    human_judgment: true
    rationale: "The plan's own must_haves.truths item 8 declares this a backstop truth — no Proton API contract is available inside this repository to confirm the URL form is honoured. Confirmable only by a live re-test of 03-UAT.md's Test 1 after a Proton source refresh, per the plan's verification section item 6."

# Metrics
duration: ~15min
completed: 2026-08-01
status: complete
---

# Phase 03 Plan 10: Proton email deep link becomes an All Mail search Summary

**Replaced the unaddressable label-name Proton webmail deep link with a percent-encoded, rune-capped All Mail search-by-subject link (`webmailSearchDeepLink`), closing UAT gap G-03-3 while keeping fidelity honestly declared ANCHORED.**

## Performance

- **Duration:** ~15 min
- **Completed:** 2026-08-01
- **Tasks:** 2/2
- **Files modified:** 5 (2 created, 3 modified)

## Accomplishments

- New `plugins/proton/deeplink.go` owns deep-link construction end to end: a fixed `all-mail` system-view path segment (the only name-addressable target available without a verified per-label id mapping), a 500-rune keyword cap mirroring `Snippet`'s existing precedent, and `encodeKeywordFragment`/`webmailSearchDeepLink` built only from `net/url`, `strings` and `unicode/utf8` (no new dependency).
- `toItem` now builds `DeepLink` from the envelope's own `Subject` via the new constructor; the old `firstLabel`-based path construction and its `pathEscapeSegment` helper (plus its sole `net/url` import) are removed outright.
- Table-driven test (`TestWebmailSearchDeepLink_Table`) proves ordinary, absent/empty/whitespace-only, hostile-punctuation, and trailing-separator-base cases by exact string equality; a dedicated test proves a multi-byte subject longer than the cap always decodes back to valid UTF-8 at exactly the cap's rune count.
- Two `toItem`-level tests prove the built item's `DeepLink` equals the constructor's own output (and never contains the matched label's leaf name) and that `Fidelity` is still `LINK_FIDELITY_ANCHORED`.
- Both operator-facing descriptions of `webmail_base_url` (`config.example.toml`, `kernel/config/types.go`) now describe the All Mail search link the plugin actually builds, and explain why a label-name path was never addressable — the same class of stale documentation that cost this phase four diagnosis rounds in G-03-1.

## Task Commits

Each task was committed atomically:

1. **Task 1: the email's deep link becomes a webmail search that lands next to the message — still honestly anchored** - `74ad381` (feat, tracer/TDD: RED observed via compile failure, then GREEN)
2. **Task 2: the two operator-facing definitions of webmail_base_url stop describing a link that no longer exists** - `44afe2a` (docs)

_Note: Task 1's RED phase was proven via a compile-time failure (`go test` reporting `undefined: webmailSearchDeepLink` etc. across all four new tests), not a runtime test failure — the constructor genuinely did not exist yet, which is the strongest possible RED signal for a not-yet-created function._

## Files Created/Modified

- `plugins/proton/deeplink.go` - New file: `webmailAllMailSegment`, `deepLinkKeywordRuneCap`, `encodeKeywordFragment`, `webmailSearchDeepLink`
- `plugins/proton/deeplink_test.go` - New file: table test, over-cap UTF-8 test, two `toItem`-level assertions
- `plugins/proton/plugin.go` - `toItem`'s deep-link construction rewritten to call `webmailSearchDeepLink`; `pathEscapeSegment` and its `net/url` import removed; doc comment corrected
- `config.example.toml` - `webmail_base_url` comment block corrected to describe the All Mail search link and why a label-name path was replaced
- `kernel/config/types.go` - `WebmailBaseURL` doc comment corrected identically; field name, tag, type and ordering unchanged

## Decisions Made

- `webmailSearchDeepLink` trims the subject once, tests `HasRenderableText` on that trimmed value, and reuses it for both the emptiness check and the capping/encoding step — one definition of "is there anything here", per `body.go`'s existing package convention.
- The rune cap is applied by converting to a rune slice and slicing, identically to `Snippet`'s own technique, never by slicing bytes — the reason the over-cap test can assert the decoded keyword is always valid UTF-8.
- `pathEscapeSegment` and its sole `net/url` import in `plugin.go` were deleted outright (not left dead beside the replacement); the compiler's unused-import refusal is the proof removal was complete.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None. The link is fully wired: `toItem` calls the real constructor with the plugin's configured `webmailBaseURL` and the envelope's actual subject; nothing renders a hardcoded or placeholder value.

## Live Verification Pending (backstop truth)

Task 1's `<verify><human-check>` step — restarting the kernel, refreshing the Proton source, and clicking "Open in Proton Mail" in the detail pane against the user's real Proton account — was not performed by this executor. This is not a deviation: the plan's own `must_haves.truths` explicitly marks this as a `backstop`-verified truth ("Whether Proton webmail honours a hash-based keyword search on the All Mail view... cannot be established from inside this repository"), and the plan's `<verification>` section (item 6) designates re-running `03-UAT.md`'s Test 1 after a source refresh as the sole confirmation mechanism. All automated verification (build, vet, full test suites across every module, the repo-wide egress scan) passed. Per the plan's own degradation-safety framing: if Proton does not honour this URL form, it redirects to the inbox — exactly today's pre-fix behavior, never worse.

## Next Phase Readiness

- G-03-3 is closed on the code side; the phase's UAT document should be refreshed with a live re-test of "Open in Proton Mail" against the real account to confirm the All Mail search form is honoured, per the plan's own backstop-truth framing.
- No blockers for merging this wave.

---
*Phase: 03-email-in-the-webspace*
*Completed: 2026-08-01*

## Self-Check: PASSED

- FOUND: plugins/proton/deeplink.go
- FOUND: plugins/proton/deeplink_test.go
- FOUND: plugins/proton/plugin.go
- FOUND: config.example.toml
- FOUND: kernel/config/types.go
- FOUND: .planning/phases/03-email-in-the-webspace/03-10-SUMMARY.md
- FOUND commit: 74ad381 (feat(03-10): replace label-name email deep link with an All Mail search link)
- FOUND commit: 44afe2a (docs(03-10): correct webmail_base_url descriptions to describe the All Mail search link)
- FOUND commit: 9744fcc (docs(03-10): complete deep-link All Mail search plan)
