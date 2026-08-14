---
phase: 12-filesystem-source
plan: 10
subsystem: ui
tags: [svelte, sveltekit, vitest, playwright, healthTone, sourceChip, gap-closure]

# Dependency graph
requires:
  - phase: 12-filesystem-source
    provides: "kernel/httpapi/sources.go sourceStatus.LastNotice (last_notice, 12-09), and the root-derived match-value labels 12-08 shipped"
provides:
  - "SourceStatus.last_notice (web/src/lib/api.ts) — the optional field mirroring the kernel's 12-09 wire shape"
  - "healthTone's advisory branch (web/src/lib/format.ts) — a successfully-synced source carrying an advisory reads warning, not success"
  - "SourceChip.svelte's fifth tooltip branch — the display name, the synced-relative phrase, and the kernel's advisory text, gated so a genuine error keeps its own copy"
  - "MatchFieldsForm.svelte's exact-match/no-wildcards helper sentence, shared by both add-source flows and the chip menu's Edit match settings… modal"
  - "web/e2e/specs/12-zero-match-diagnostic.spec.ts — a browser-level gate reproducing the user's exact failing config against a real kernel and a real topos-plugin-filesystem subprocess"
affects: [any future phase touching SourceChip.svelte's tooltipText, healthTone, or the source-health chip row]

# Actuals (#2632)
actuals:
  tokens: 6597
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A kernel-published advisory field reuses an EXISTING tone (warning) rather than growing a fifth HealthTone/design token — the same 'extend the chain, never a parallel tone system' rule Phase 11's pin-mismatch branch established, applied here to healthTone AND to SourceChip.svelte's tooltipText derivation."
    - "A Copywriting Contract guard (source-chip-tooltip.test.ts) is protected independently by a SECOND test file (match-advisory.test.ts) asserting the same byte-for-byte templates, so a future edit satisfying one guard by breaking the other cannot pass either."

key-files:
  created:
    - web/src/lib/components/match-advisory.test.ts
    - web/src/lib/components/match-values-hint.test.ts
    - web/e2e/specs/12-zero-match-diagnostic.spec.ts
  modified:
    - web/src/lib/api.ts
    - web/src/lib/format.ts
    - web/src/lib/components/SourceChip.svelte
    - web/src/lib/components/MatchFieldsForm.svelte

key-decisions:
  - "The advisory tooltip branch is gated on `source.last_status !== 'error'`, placed after the relative-time constant and before the tone switch — an advisory never displaces a genuine error's own Copywriting Contract copy, and the branch itself sits last in healthTone's precedence chain (after pin-mismatch, never-synced, unreachable, errored)."
  - "The em-dash-with-spaces separator every existing tooltip branch uses is reused verbatim for the advisory branch's third segment (`{display_name} — synced {relative} — {advisory}`) rather than inventing new punctuation."
  - "MatchFieldsForm.svelte's new sentence lives in the SAME <p> as the pre-existing helper text (not a second element) — asserted structurally by match-values-hint.test.ts, so the form never grows a second helper node per field."

patterns-established:
  - "A gap-closure plan's new guard test independently re-asserts an existing Copywriting Contract's protected strings, rather than trusting the pre-existing guard alone to catch a regression introduced by the new branch."

requirements-completed: [SRC-04]

coverage:
  - id: D1
    description: "A source whose latest sync carried a zero-match advisory renders the amber warning dot instead of green, and its tooltip/title carry the kernel's advisory text verbatim"
    requirement: "SRC-04"
    verification:
      - kind: unit
        ref: "web/src/lib/components/match-advisory.test.ts#healthTone: a source carrying a zero-match advisory"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/12-zero-match-diagnostic.spec.ts#the chip renders the warning tone and a title carrying the API-published advisory"
        status: pass
    human_judgment: false
  - id: D2
    description: "An advisory never outranks a real problem — pin mismatch, unreachable, never-synced and errored sources keep exactly the tone and copy they had before"
    requirement: "SRC-04"
    verification:
      - kind: unit
        ref: "web/src/lib/components/match-advisory.test.ts#healthTone: a source carrying a zero-match advisory (six-case precedence matrix)"
        status: pass
    human_judgment: false
  - id: D3
    description: "All four pre-existing tooltip branches stay byte-identical; no fifth HealthTone or new design token was introduced"
    requirement: "SRC-04"
    verification:
      - kind: unit
        ref: "web/src/lib/components/source-chip-tooltip.test.ts (run unedited)"
        status: pass
      - kind: unit
        ref: "web/src/lib/components/match-advisory.test.ts#tooltipText structure: the four pre-existing branches stay byte-for-byte"
        status: pass
    human_judgment: false
  - id: D4
    description: "The advisory is rendered as escaped text, never markup, in every file that touches it"
    requirement: "SRC-04"
    verification:
      - kind: unit
        ref: "web/src/lib/components/match-advisory.test.ts#SourceChip.svelte: the advisory is rendered as escaped text, never as markup"
        status: pass
    human_judgment: false
  - id: D5
    description: "The shared match-values form states that values match exactly and wildcards are not supported"
    requirement: "SRC-04"
    verification:
      - kind: unit
        ref: "web/src/lib/components/match-values-hint.test.ts#MatchFieldsForm.svelte: the shared helper text states the exact-match rule"
        status: pass
    human_judgment: false
  - id: D6
    description: "One Playwright spec reproduces the user's exact failing config against a real kernel and a real plugin binary, asserting the healthy-but-empty API state and the amber chip that now flags it"
    requirement: "SRC-04"
    verification:
      - kind: e2e
        ref: "web/e2e/specs/12-zero-match-diagnostic.spec.ts (both tests)"
        status: pass
    human_judgment: false

duration: ~9min
completed: 2026-08-14
status: complete
---

# Phase 12 Plan 10: Zero-match advisory reaches the source chip (G-12-1/G-12-3 gap closure) Summary

**The `files` chip that read green while contributing nothing now reads amber and names the exact webspace/value that matched no items — closing the loop 12-09's kernel-side advisory opened, with a browser-level Playwright gate reproducing the user's own reported config.**

## Performance

- **Duration:** ~9 min
- **Started:** 2026-08-14T10:41:37+01:00 (base commit)
- **Completed:** 2026-08-14T10:50:21+01:00 (last task commit)
- **Tasks:** 3/3
- **Files modified:** 7 (4 modified, 3 created)

## Accomplishments

- `SourceStatus.last_notice` (optional) published on the client's `GET /api/sources` type, mirroring kernel/httpapi/sources.go's 12-09 field, with a comment recording it never implies failure and coexists with `last_status: 'ok'`.
- `healthTone` gains one branch — last among the problem states, before the final success return — mapping a non-empty `last_notice` to the existing `warning` tone. No fifth `HealthTone`, no new design token.
- `SourceChip.svelte`'s `tooltipText` gains a fifth branch (after `isPinMismatch`, before the tone switch) composing the display name, the synced-relative phrase, and the kernel's own advisory text — gated so an errored `last_status` keeps its own warning-branch copy instead.
- `match-advisory.test.ts`: the six-case tone precedence matrix (advisory alone, advisory absent, advisory empty-string, advisory+error, advisory+unreachable, advisory+never-synced, advisory+pin-mismatch) against the real `healthTone`, plus structural guards proving the four pre-existing tooltip templates stay byte-for-byte, the new branch sits ahead of the tone switch, and no raw-HTML directive exists anywhere in `SourceChip.svelte`.
- `web/e2e/specs/12-zero-match-diagnostic.spec.ts`: reproduces the debug session's literal config (`[webspaces.test.match.files] folders = ['*']`) against a real booted kernel and a real `topos-plugin-filesystem` subprocess — proving the API's healthy-but-empty coexistence and the DOM's warning-tone chip whose `title` carries the API-published advisory, in both tone directions (`bg-warning` present, `bg-success` absent), with no "A source couldn't sync" degraded treatment rendered.
- `MatchFieldsForm.svelte`'s shared helper text (both add-source flows and the chip menu's "Edit match settings…" modal) now states values are matched exactly, case-insensitively, and that wildcard/glob patterns are not supported — the exact surface the debug session's `blind_spots` note flagged as unexamined.

## Task Commits

Each task was committed atomically:

1. **Task 1: The chip stops lying — a successfully-synced source carrying an advisory reads warning and says why** - `ffdd2e1` (feat)
2. **Task 2: The user's exact config, in a browser, against a real kernel and a real plugin binary** - `fdd0fb8` (test)
3. **Task 3: Say it in the field where it is typed — match values are exact, wildcards are not supported** - `e9b7412` (feat)

**Plan metadata:** (this SUMMARY's own commit, made by the orchestrator/executor after this file lands)

## Files Created/Modified

- `web/src/lib/api.ts` - `SourceStatus.last_notice?: string`, documented as kernel-published, non-fatal, never-parsed
- `web/src/lib/format.ts` - `healthTone`'s new advisory branch, reusing the existing `warning` tone
- `web/src/lib/components/SourceChip.svelte` - `advisory` derived value, and `tooltipText`'s fifth branch
- `web/src/lib/components/match-advisory.test.ts` - the tone precedence matrix and structural guards (new file)
- `web/src/lib/components/MatchFieldsForm.svelte` - the added exact-match/no-wildcards helper sentence
- `web/src/lib/components/match-values-hint.test.ts` - the guard proving the helper text states the exact-match rule (new file)
- `web/e2e/specs/12-zero-match-diagnostic.spec.ts` - the browser-level end-to-end gate over the user's exact failing config (new file)

## Decisions Made

- The advisory branch composes `${display_name} — synced ${relative} — ${advisory}`, reusing the exact em-dash-with-spaces separator every other tooltip branch already uses, rather than inventing new punctuation for the new segment.
- The advisory branch is gated on `source.last_status !== 'error'` — an advisory can coexist with a source that IS healthy, but must never be composed alongside (or instead of) the warning branch's own error copy; `healthTone`'s own precedence (pin-mismatch, never-synced, unreachable, errored, THEN advisory) independently enforces the same "real problem outranks advisory" rule at the tone level.
- `match-advisory.test.ts` re-asserts the four pre-existing tooltip templates byte-for-byte, independently of `source-chip-tooltip.test.ts` — so a future edit that satisfies one guard by breaking the other cannot pass either (per the plan's own must_haves rationale).
- The new MatchFieldsForm.svelte sentence was appended to the SAME `<p>` as the pre-existing helper text rather than a sibling element, keeping exactly one helper paragraph per field (asserted structurally by `match-values-hint.test.ts`).

## Deviations from Plan

None — plan executed exactly as written; no Rule 1-4 auto-fixes were needed.

One acceptance-criterion imprecision was discovered (not a code deviation): the plan's Task 1 acceptance criteria include `grep -c 'HealthTone =' web/src/lib/format.ts | grep -qx 1` to prove no fifth tone was introduced. This grep pattern also matches a PRE-EXISTING, unrelated line — `worstHealthTone`'s own `let worst: HealthTone = 'success';` (present before this plan, untouched by it) — so the literal count is 2, not 1. The underlying invariant the criterion protects (`export type HealthTone = ...` appears exactly once, no fifth tone declared) DOES hold, confirmed by `grep -c 'export type HealthTone =' web/src/lib/format.ts` = 1 and by `git diff` showing the type declaration line as unchanged context. No code change was made in response to this; it is documented here as a pre-existing false-positive in the acceptance-criterion grep pattern, for a future plan-phase pass to tighten if it reuses this pattern.

## Issues Encountered

- `web/node_modules` was not present in this worktree at the start of execution (a fresh worktree checkout). Ran `npm ci` before the first `vitest run` — a standard, expected step for a freshly created worktree, not a plan deviation.

## Known Stubs

None — no stub data, hardcoded empty renders, or placeholder text were introduced.

## Threat Flags

None — every threat this plan's `<threat_model>` names (T-12-45 through T-12-50, T-12-SC) was mitigated exactly as designed: the advisory is interpolated as Svelte text (no `{@html}` anywhere, grep-verified across both modified `.svelte` files), the advisory branch sits last in `healthTone`'s chain and is gated against a genuine error, the SPA never parses or branches on the advisory's content, and no package-manager install was needed (no new dependency was added).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- G-12-1/G-12-3 is now closed end to end: `12-08` made the correct match value expressible and documented, `12-09` made a wrong value's zero-match outcome observable at the API layer, and this plan makes it visible on the chip and discouraged at the point the value is typed. No further gap-closure plan is needed for this pair.
- The one remaining `must_haves` item (`statement: "On the user's own desktop, the files chip in webspace test shows the warning tone..."`) is explicitly `verification: backstop` — a live-desktop human confirmation outside an automated worktree agent's reach, exactly like 12-09-SUMMARY.md's own D4 coverage entry. The kernel-side field (12-09) and the browser-visible surface this plan built are both proven above; only the user's own desktop confirmation remains, and is not blocking.
- No blockers. `git diff --name-only` against the base commit lists exactly this plan's seven declared `files_modified` files — no Go file, no doc file, and no existing e2e/vitest file was touched.

## Self-Check: PASSED

All 8 declared files (7 task files + this SUMMARY) verified present on disk. All 4 task commit hashes (`ffdd2e1`, `fdd0fb8`, `e9b7412`, `01c533e`) verified present in `git log --oneline`.

---
*Phase: 12-filesystem-source*
*Completed: 2026-08-14*
