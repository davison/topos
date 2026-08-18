---
phase: 14-google-drive-source-built-out-of-repo
plan: 05
subsystem: docs
tags: [contract-gaps, plugin-contract, gap-triage, google-drive, backlog]

# Dependency graph
requires:
  - phase: 14-04
    provides: "The proven, out-of-repo topos-plugin-gdrive plugin and 14-LIVE-UAT.md's closing line confirming no additional gap-log entries surfaced live"
provides:
  - "14-CONTRACT-GAPS.md — the plugin repository's 20-entry gap log imported unedited into this repository, plus a triage table dispositioning every entry by id"
  - "docs/plugin-contract.md — 19 documentation-fixable gaps republished in place, including a new 'Plugin-private state' section the contract never had before"
  - "A backlog item under ROADMAP.md's Phase 999.1 (plugin distribution/dev guide) filing the one contract-change gap (GAP-06), and a PLUG-11 evidence note in REQUIREMENTS.md"
affects: [phase-14-uat, 999.1, 999.2]

# Actuals (#2632)
actuals:
  tokens: 17100
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Gap-triage-with-code-verification: when a gap's proposed disposition is uncertain (GAP-13's additive-vs-full-replace question), resolve it by reading the actual kernel source (kernel/index/store.go's ReplaceWebspaceSourceItems) rather than guessing or leaving the plugin repository's tentative proposal unexamined — the resulting documentation-fixable answer is backed by a real code citation, not restated uncertainty"

key-files:
  created:
    - .planning/phases/14-google-drive-source-built-out-of-repo/14-CONTRACT-GAPS.md
  modified:
    - docs/plugin-contract.md
    - .planning/ROADMAP.md
    - .planning/REQUIREMENTS.md

key-decisions:
  - "GAP-13 (does the kernel treat each Match response as additive or as the authoritative full current set?) was left ambiguous by the plugin repository's own proposal ('documentation-fixable... if the honest answer requires a wire-level removal signal, the disposition becomes contract-change'). Resolved by reading kernel/index/store.go's ReplaceWebspaceSourceItems directly: it upserts items and then, in the same transaction, deletes every prior webspace_items row for that exact (webspace, source) pair before reinserting rows only for the items just returned — confirmed full-replace, never additive. Dispositioned documentation-fixable on that evidence, avoiding a false contract-change filing."
  - "GAP-08's two disposition options (add XDG_DATA_HOME to the launch-environment allowlist — a contract/kernel change — or state that plugin-private state must resolve from HOME alone — a documentation-only fix) were both legitimate per the plugin repository's own note. Chose the documentation-only path: the new 'Plugin-private state' section tells plugin authors to resolve from HOME alone, closing the gap with zero wire-level change, consistent with this plan's general preference for the doc-only fix when one genuinely closes the gap."
  - "GAP-20 (a question about Google's own OAuth API error-response granularity) does not cleanly fit any of the three disposition categories as written — the four published inputs never answered it because it isn't a question about the topos contract at all. Dispositioned documentation-fixable, landing in 'What this document does not cover': the contract now states explicitly that third-party source-API behavior questions are out of scope for it, which is itself the honest, useful answer for a future plugin author who reaches the same question."
  - "The triage table's 'landing place' column was written with its final, accurate value directly in Task 1 rather than as a placeholder later updated by Task 2 — both tasks ran in the same continuous session, so there was no intermediate state where the table named a target that Task 2's actual edit didn't match. No factual conflict results: every row's recorded landing place is exactly where Task 2's edit landed."

patterns-established: []

requirements-completed: [SRC-06]

coverage:
  - id: D1
    description: "14-CONTRACT-GAPS.md imports all 20 gap-log entries from the plugin repository unedited, records that 14-LIVE-UAT.md's live run surfaced nothing additional, and triages every entry to exactly one disposition (19 documentation-fixable, 1 contract-change) with a landing place named for each and a reconciliation line proving nothing was dropped"
    requirement: SRC-06
    verification:
      - kind: other
        ref: "test -s 14-CONTRACT-GAPS.md && grep -q GAP-01 && grep -q GAP-02 && make docs-check — PASS; id-list diff between the plugin repo's CONTRACT-GAPS.md and the imported section — empty (no loss); 20 imported + 0 UAT additions = 20 triage rows, arithmetically verified in the reconciliation line"
        status: pass
    human_judgment: false
  - id: D2
    description: "docs/plugin-contract.md answers all 19 documentation-fixable gaps in place, including a new 'Plugin-private state' section covering where a plugin keeps its own state, what the scrubbed launch environment does/doesn't guarantee, and that the host never reads/migrates/removes it — every edit additive, none touching a section the triage table didn't name"
    requirement: SRC-06
    verification:
      - kind: other
        ref: "make docs-check (0 exit) && grep -qi 'private state' docs/plugin-contract.md && make test-portable (0 exit, all packages including sdk/mock/paperless/silverbullet/proton/mockstrict/whatsapp/filesystem) — all PASS; git diff docs/plugin-contract.md reviewed hunk by hunk — every hunk purely additive, landing exactly in the 12 sections the triage table named (Configuration, Discovery and launch, Describe, Match, Fetch, Health, The Item message, Provenance, What this document does not cover, plus the new Plugin-private state section)"
        status: pass
    human_judgment: false
  - id: D3
    description: "GAP-06 (the sole contract-change row) filed as a new backlog item under ROADMAP.md's Phase 999.1, citing the gap id; REQUIREMENTS.md's PLUG-11 statement carries a one-line evidence note citing all 20 gap ids and pointing at the full triage — both diffs scoped to only the intended section"
    requirement: SRC-06
    verification:
      - kind: other
        ref: "git diff .planning/ROADMAP.md shows one added line inside Phase 999.1's Plans list only, Phase 14's own entry and every other phase entry untouched; git diff .planning/REQUIREMENTS.md shows one added line under PLUG-11 only, no requirement checkbox or mapping-table row touched; gsd-tools query roadmap.get-phase 14 still returns found:true with all four success criteria intact"
        status: pass
    human_judgment: false

# Metrics
duration: ~55min
completed: 2026-08-18
status: complete
---

# Phase 14 Plan 05: Contract-Gap Triage Summary

**The plugin repository's 20-entry `CONTRACT-GAPS.md` is imported unedited and triaged (19 documentation-fixable, 1 contract-change); the 19 are republished into `docs/plugin-contract.md` — including a new "Plugin-private state" section the published contract never had before — and the one contract-change gap (GAP-06) is filed as a backlog item under Phase 999.1, closing the loop D-07 exists to produce.**

## Performance

- **Duration:** ~55 min
- **Tasks:** 3 of 3 complete
- **Files modified:** 4 (1 created, 3 modified)

## Accomplishments

- **Task 1 — Imported and triaged the gap log.** `14-CONTRACT-GAPS.md` reproduces all 20 entries (GAP-01 through GAP-20) from `topos-plugin-gdrive/CONTRACT-GAPS.md` verbatim — every question, what the four published inputs said, the resolution the plugin repository actually took, and its own proposed disposition — with an id-list diff against the source file confirming zero loss. `14-LIVE-UAT.md`'s closing line ("None reported") is recorded explicitly as Part 2's answer rather than left implicit. Part 3's triage table dispositions every id exactly once: 19 documentation-fixable (each naming the `docs/plugin-contract.md` section its answer would land in) and 1 contract-change (GAP-06, filed against the Phase 999.1 backlog). GAP-13's disposition was resolved by reading `kernel/index/store.go`'s `ReplaceWebspaceSourceItems` directly rather than accepting the plugin repository's own hedge — confirmed full-replace reconciliation, not additive, closing what could otherwise have become a second contract-change filing. The closing reconciliation line (20 imported + 0 UAT additions = 20 triage rows) is arithmetically correct.
- **Task 2 — Republished the 19 documentation-fixable gaps.** `docs/plugin-contract.md` gained a new "Plugin-private state" section (GAP-01/GAP-02/GAP-08: where a plugin keeps state that must survive restarts, what the deliberately-scrubbed launch environment does and doesn't guarantee about that location — resolve from `HOME` alone, never assume `XDG_DATA_HOME` reaches the subprocess — and that the host never reads, migrates, backs up, or removes it) plus in-place edits to 9 existing sections: Describe (source_type/display_name naming convention), Configuration (fail-loud discipline scope excludes `Describe`/trial launches; credential-shaped extras values), Discovery and launch (zero-argument launch guarantee), Match (full-replace reconciliation; a worked hierarchical match-value example; structural/folder nodes never emitted as Items; delta-token capture-before-walk guidance), Fetch (MIME/export-format scope is the plugin author's own choice; `unavailable_reason` is free text, not a shared vocabulary), Health (`last_sync_unix` semantics), The Item message (a concrete preview-length anchor), Provenance (the `source_system` fallback for a plugin with no `base_url`), and "What this document does not cover" (third-party source-API behavior is explicitly out of scope). Every hunk in `git diff docs/plugin-contract.md` is additive — verified line by line — landing only in sections the triage table named. `make docs-check` and `make test-portable` (all 12 Go modules/packages, including `sdk`, `mock`, `paperless`, `silverbullet`, `proton`, `mockstrict`, `whatsapp`, `filesystem`) both pass.
- **Task 3 — Filed GAP-06 against the backlog.** A new item under `ROADMAP.md`'s Phase 999.1 ("Plugin distribution, dev guide, certification") states the gap as a change to make — exempt a fully-`extras` source from the `base_url`/`token`/`path` config-load requirement, or define a fourth top-level key such a source declares instead — citing GAP-06 and the triage file. `REQUIREMENTS.md`'s `PLUG-11` future requirement gained a one-line evidence note citing all 20 gap ids and pointing a future reader at the full triage. Both diffs are scoped exactly as required: only the Phase 999.1 Plans list in ROADMAP.md (Phase 14's own entry, every other phase entry, and the progress table untouched), and only the line under PLUG-11 in REQUIREMENTS.md (no requirement checkbox or traceability-table row touched). `gsd-tools query roadmap.get-phase 14` still returns `found: true` with all four success criteria intact.

## Task Commits

Each completed task was committed atomically:

1. **Task 1: Import the gap log and disposition every entry** - `0ffddd1` (docs)
2. **Task 2: Republish the documentation-fixable gaps into the published contract** - `050e888` (docs)
3. **Task 3: File the contract-change gaps against the backlog** - `2b3a43e` (docs)

**Plan metadata:** this SUMMARY.md and the STATE.md/ROADMAP.md progress-tracking updates are the orchestrator's own follow-on commit after this worktree merges (worktree mode) — see "Next Phase Readiness" below.

## Files Created/Modified

- `.planning/phases/14-google-drive-source-built-out-of-repo/14-CONTRACT-GAPS.md` (new) — the imported gap log, UAT-addition check, and triage table described above.
- `docs/plugin-contract.md` (modified) — 194 lines added across 12 sections, described above.
- `.planning/ROADMAP.md` (modified) — one new backlog item under Phase 999.1.
- `.planning/REQUIREMENTS.md` (modified) — one new evidence line under PLUG-11.

## Decisions Made

1. **GAP-13's additive-vs-full-replace question was resolved by reading kernel source, not by accepting the plugin repository's own hedge.** Its proposed disposition was conditional ("documentation-fixable... if the honest answer requires a wire-level removal signal, the disposition becomes contract-change"). `kernel/index/store.go`'s `ReplaceWebspaceSourceItems` doc comment and implementation confirm the kernel deletes every prior `webspace_items` row for the exact (webspace, source) pair before reinserting rows for the items a `Match` call just returned, inside one transaction — full-replace, never additive. This closed the gap as documentation-fixable on real evidence rather than leaving an unresolved contract-change filed defensively.
2. **GAP-08 chose the documentation-only path over the allowlist-change path.** The plugin repository's own gap entry offered two ways to close it: add `XDG_DATA_HOME` to the launch environment's allowlist (a kernel/contract wire change), or state that plugin-private state must resolve from `HOME` alone (documentation-only). The new "Plugin-private state" section takes the second path, closing the gap with zero wire-level change.
3. **GAP-20 doesn't cleanly fit the three-way disposition vocabulary, so it was placed where it does the most good.** It's a question about Google's own OAuth API, which none of the four published inputs could ever answer — not because the answer was hard to find (the `not-a-gap` category), but because the question is out of the topos contract's scope entirely. Dispositioned documentation-fixable, landing in "What this document does not cover," which states that scope boundary explicitly — the honest, useful answer for a future plugin author who reaches the same question about their own source system.
4. **The triage table's landing-place values were written accurate from Task 1, not as placeholders Task 2 later updated.** Since both tasks ran in one continuous session, there was no need for an intermediate "target section named" state distinct from the final "where the answer landed" state — the plan's two-step wording (Task 1 names a target, Task 2 records the landing) collapsed into one accurate value with no factual conflict.

## Deviations from Plan

None - plan executed as written, with the two judgment calls (GAP-13's kernel-code verification, GAP-20's not-quite-fitting disposition) both resolved within the plan's own triage vocabulary and acceptance criteria rather than requiring a deviation.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All three tasks complete and verified: `make docs-check` and `make test-portable` both pass; the gap id list in `14-CONTRACT-GAPS.md` is confirmed a superset (in fact identical set) of the plugin repository's own 20 ids; every gap id has exactly one disposition with a landing place named; `gsd-tools query roadmap.get-phase 14` still returns the phase with its four success criteria intact.
- This is Phase 14's final plan (Wave 4 of 4). Once this worktree merges, the orchestrator should mark Phase 14 complete in `STATE.md`/`ROADMAP.md`'s progress table and requirement `SRC-06` complete in `REQUIREMENTS.md` (this plan's own `requirements:` frontmatter names `SRC-06`; `SRC-05` was already completed by 14-04's live UAT run) — deliberately not done by this execution per worktree-mode convention (STATE.md untouched; ROADMAP.md/REQUIREMENTS.md edits here are scoped strictly to the backlog/PLUG-11 content this plan's own declared scope covers, not phase-progress tracking).
- Phase 999.1 (plugin distribution, dev guide, certification) now carries one concrete backlog item (GAP-06) ready for a future `/gsd-review-backlog` promotion, alongside its existing TBD placeholder for the rest of that phase's scope.
- Phase 999.2 (move functional plugins to a `topos-plugins` sibling repo) references "the Phase 14 contract-gap triage (14-05)" in its own open-question note — that reference now resolves to a real, completed artifact (`14-CONTRACT-GAPS.md`) rather than a future one.

---
*Phase: 14-google-drive-source-built-out-of-repo*
*Completed: 2026-08-18*

## Self-Check: PASSED

`.planning/phases/14-google-drive-source-built-out-of-repo/14-CONTRACT-GAPS.md` confirmed present on disk in this worktree. `docs/plugin-contract.md`, `.planning/ROADMAP.md`, `.planning/REQUIREMENTS.md` confirmed modified on disk. Commits `0ffddd1`, `050e888`, `2b3a43e` confirmed present via `git log --oneline -5` in this worktree.
