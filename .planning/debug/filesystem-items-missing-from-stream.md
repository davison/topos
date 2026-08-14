---
status: diagnosed
trigger: "UAT phase 12 (G-12-1/G-12-3): the filesystem plugin is not showing files that exist within the configured directory / plugin shows no files in the stream"
created: 2026-08-14T09:00:00Z
updated: 2026-08-14T09:20:00Z
---

## Current Focus
<!-- OVERWRITE on each update - reflects NOW -->

hypothesis: CONFIRMED — the webspace match block `[webspaces.test.match.files] folders = ['*']` is compared as a LITERAL string (exact, case-insensitive) against the only label the plugin emits for top-level files ("Lucid", the root's base name); '*' never matches, so every synced item is filtered out at the Match RPC and zero rows persist.
test: complete
expecting: n/a
next_action: return diagnosis (goal: find_root_cause_only)
known_pattern_candidate: none (KB-001..006 are lifecycle/exec classes, no match)
bug_class: Bohrbug (deterministic in the user's environment; environment/config-specific, invisible to the hermetic harness whose fixtures always use real label values)

reasoning_checkpoint:
  hypothesis: "The stream is empty because the webspace's explicit match block value '*' (and previously '**', per config.toml.bak) is treated as a literal label under the exact-case-insensitive-match discipline (plugins/filesystem/plugin.go labelMatchesAny -> strings.EqualFold), while every top-level file under /home/darren/Documents/Lucid carries exactly one label, 'Lucid' (item.go folderLabels). No component in the pipeline interprets wildcards, so the plugin's Match RPC filters out all ~13 in-allowlist items and correlate persists zero rows for (test, files)."
  confirming_evidence:
    - "User's real config (~/.config/topos/config.toml): [sources.files] path=/home/darren/Documents/Lucid, no recursive key; [webspaces.test] keywords=[], sources allowlist includes 'files'; [webspaces.test.match.files] folders = ['*']"
    - "config.toml.bak diff: line 104 was folders = ['**'] before the last save — the user tried doublestar glob syntax first, then '*': unambiguous wildcard intent"
    - "Live kernel GET /api/sources: files source reachable:true, last_status:ok, last_sync fresh — sync itself succeeds"
    - "Live kernel GET /api/webspaces/test/stream: 97 items, ALL source 'docs' (paperless); zero from 'files'"
    - "ls /home/darren/Documents/Lucid: 15 top-level files, ~13 in the default allowlist (pdf/pptx/xlsx/png) — the walk has plenty to return"
    - "plugins/filesystem/plugin.go Match(): when match_fields['folders'] present, keeps only items where labelMatchesAny(labels, values); labelMatchesAny uses strings.EqualFold — exact, no glob"
    - "plugins/filesystem/item.go folderLabels(): top-level file's ONLY label is filepath.Base(root) = 'Lucid'"
    - "kernel/correlate/correlate.go matchFieldsFor(): an explicit ws.Match block is passed to the plugin VERBATIM — no kernel-side wildcard handling either (grep for glob/wildcard in kernel/correlate + kernel/config: none)"
  falsification_test: "Change the match block to folders = ['Lucid'] and reload config: if items still don't appear, the hypothesis is wrong. (Not executed — find_root_cause_only forbids touching the user's live config; the e2e suite already proves the positive case: specs that use a real label value (12-01/D5, 12-03/D3, 12-04/D5) all see stream items.)"
  fix_rationale: "n/a — diagnose-only; fix direction handed to gap-closure plan"
  blind_spots: "Did not observe whether the '*' was typed in the UI's match editor or hand-edited (config header says 'managed by the topos UI'; both paths accept it silently). Did not verify the UI match editor's placeholder/help text — if it hints at globs it compounds the trap. test-ext also lands zero items (labels=['untrust']) — possibly the same mismatch class for the demo plugin, unexamined."
  candidate_causes:
    - "config: match value '*' equals no emitted label (CONFIRMED — sufficient alone)"
    - "code: plugin match/walk logic broken (ELIMINATED — e2e with real label values passes; code reads correct)"
    - "environment: stale plugin binary / wrong plugins dir (ELIMINATED — config points at this checkout's bin/plugins, all binaries rebuilt 2026-08-14 09:34, source healthy)"
    - "data: directory empty at top level or files outside allowlist (ELIMINATED — 15 top-level files, ~13 in allowlist)"
    - "environment: mount/permission failure (ELIMINATED — Health reachable:true, last_status ok)"
  and_gate: "no — the single config condition fully produces the symptom given a healthy sync. recursive=false is NOT contributing (the files sit at top level). But note a latent design gap discovered en route: with recursive=true, nested files carry only per-segment labels (folderLabels never includes the root base name for nested files), so NO single folders value can match all files of a recursive source — 'everything from this instance' is inexpressible in the current match vocabulary."

## Symptoms
<!-- Written during gathering, then IMMUTABLE -->

expected: With a real filesystem source configured on the user's desktop, files inside the configured directory appear as items in the matching webspace stream (PDF inline preview, office honest no-preview, deep-link open action).
actual: "the filesystem plugin is not showing files that exist within the configured directory" / "plugin shows no files in the stream" (UAT tests 1 and 3; gaps G-12-1, G-12-3 — same observable failure)
errors: None reported
reproduction: User's real desktop kernel with their real config — NOT the hermetic e2e harness. Full Go suite and make e2e (115 specs, incl. real topos-plugin-filesystem binary boots) pass.
started: Discovered 2026-08-14 during UAT immediately after phase 12 completed (7/7 plans, verification 7/7)

## Eliminated
<!-- APPEND only - prevents re-investigating -->

- hypothesis: Stale/missing plugin binary (bin/plugins pre-phase-12, or external-dir copy with pin mismatch)
  evidence: "config [plugins] dir points at THIS checkout's /home/darren/projects/davison/topos/bin/plugins (absolute); topos-plugin-filesystem there rebuilt 2026-08-14 09:34 (after all phase-12 commits); source runs tier 'trusted' (no pin applies); GET /api/sources shows reachable:true, last_status:ok. The stray plugins/filesystem/filesystem binary (02:16) is an inert go-build leftover — nothing resolves plugins from package directories."
  timestamp: 2026-08-14T09:12:00Z

- hypothesis: Sync never runs or fails / health degraded (unreachable root, permission)
  evidence: "GET /api/sources: files reachable:true, syncing:false, last_status:'ok', last_sync_unix fresh (same tick as every other source). Stream response's own sync block: status ok. Directory is readable (ls succeeds)."
  timestamp: 2026-08-14T09:14:00Z

- hypothesis: Participation allowlist excludes the filesystem instance
  evidence: "[webspaces.test] sources = ['docs', 'test-ext', 'files'] — 'files' is explicitly allowlisted."
  timestamp: 2026-08-14T09:14:00Z

- hypothesis: recursive=false hides the files (all content nested in subfolders)
  evidence: "find -maxdepth 1 -type f = 15; find -type f = 15 — every file is at the top level; recursion setting is irrelevant here."
  timestamp: 2026-08-14T09:14:00Z

- hypothesis: include/exclude glob interaction drops everything
  evidence: "No [sources.files.extras] block exists in the real config — scope is the default extension allowlist, which covers ~13 of the 15 files (pdf/pptx/xlsx/png; only .MOV and .diff fall outside)."
  timestamp: 2026-08-14T09:14:00Z

- hypothesis: Plugin core logic bug (walk/labels/match broken)
  evidence: "Full Go suite + 115 e2e specs pass, including specs booting a real kernel with the real topos-plugin-filesystem binary that see stream items — all of them configure match/keyword values that equal real emitted labels. Code read confirms Match/labelMatchesAny/folderLabels behave exactly as documented."
  timestamp: 2026-08-14T09:16:00Z

## Evidence
<!-- APPEND only - facts discovered -->

- timestamp: 2026-08-14T09:00:00Z
  checked: .planning/debug/knowledge-base.md (Phase 0)
  found: No entry matches "source syncs but stream empty"; KB classes are two-phase-write/query-semantics/signal-teardown/pipe-ownership.
  implication: No known-pattern shortcut; fresh hypothesis formation.

- timestamp: 2026-08-14T09:00:00Z
  checked: docs/plugins/filesystem.md (operator contract)
  found: Match vocabulary is one field, 'folders'. A top-level file carries exactly ONE label — the configured root's base name. Matching exact, case-insensitive, no substring. NOTE — the doc never states that glob patterns are unsupported in match values, while the SAME page documents include_glob/exclude_glob extras that DO take doublestar globs.
  implication: The one plugin whose extras accept '**' globs invites the assumption its match values do too.

- timestamp: 2026-08-14T09:08:00Z
  checked: ~/.config/topos/config.toml (the user's real config, mode 600, saved 09:35 today by the UI)
  found: "[sources.files] plugin=topos-plugin-filesystem, path=/home/darren/Documents/Lucid, no recursive key. [webspaces.test] keywords=[], sources=['docs','test-ext','files']. [webspaces.test.match.files] folders = ['*']."
  implication: Explicit match block replaces keywords for 'files'; everything hinges on '*' matching an emitted label.

- timestamp: 2026-08-14T09:08:00Z
  checked: diff config.toml.bak vs config.toml
  found: "Single change: folders = ['**'] -> folders = ['*']. The user iterated on wildcard syntax."
  implication: Clear operator intent — 'match all files' — expressed in glob syntax the match pipeline does not have.

- timestamp: 2026-08-14T09:10:00Z
  checked: /home/darren/Documents/Lucid contents
  found: 15 files, all at top level; ~13 within the default allowlist (PDFs, .pptx, .xlsx, .png; .MOV and .diff outside).
  implication: The walk returns a healthy candidate set; the loss happens after walking.

- timestamp: 2026-08-14T09:12:00Z
  checked: bin/, bin/plugins/ mtimes; stray plugins/filesystem/filesystem; ~/.local/share/topos/
  found: kernel + all plugin binaries rebuilt today 09:34; config's plugins dir is this checkout's bin/plugins (absolute); stray binary is an inert build leftover; index.db WAL active (kernel live).
  implication: No staleness; the running system is current code.

- timestamp: 2026-08-14T09:14:00Z
  checked: Live GET /api/sources and GET /api/webspaces/test/stream
  found: files source reachable:true/last_status:ok; stream has 97 items, ALL from 'docs' (paperless, labels ['cars']); zero from 'files' (and zero from 'test-ext').
  implication: Sync healthy + zero filesystem items = filtering, not fetching, is the failure site.

- timestamp: 2026-08-14T09:16:00Z
  checked: plugins/filesystem/plugin.go (Match, labelMatchesAny), plugins/filesystem/item.go (folderLabels)
  found: Match keeps an item only if labelMatchesAny(labels, folders values); comparison is strings.EqualFold — exact, case-insensitive, no glob. folderLabels: top-level file's only label = filepath.Base(root) = 'Lucid'; nested files get per-segment labels + cumulative relative dir path, NEVER the root base name.
  implication: "'*' vs 'Lucid' never matches -> zero items. Also: for recursive sources no single folders value covers all files (root base name absent from nested files' labels) — 'match everything' is structurally inexpressible."

- timestamp: 2026-08-14T09:18:00Z
  checked: kernel/correlate/correlate.go (matchFieldsFor), grep for wildcard/glob in kernel/correlate + kernel/config + pluginhost
  found: An explicit ws.Match block is passed to the plugin VERBATIM (host.go:282 forwards it as MatchRequest.MatchFields). No wildcard handling anywhere kernel-side. Config validation (ValidateMatchConfig) checks field NAMES against the plugin vocabulary only — values are never validated or warned about.
  implication: A match value that can never match any label loads silently, syncs 'ok', and produces an empty stream with zero diagnostics at any layer — the silent-failure shape that turned a config mistake into a UAT-blocking mystery.

## Resolution
<!-- OVERWRITE as understanding evolves -->

root_cause: "The webspace's explicit match block for the filesystem instance — [webspaces.test.match.files] folders = ['*'] (previously ['**']) — is compared as a literal string under the pipeline's exact/case-insensitive match discipline. The plugin emits exactly one label for each top-level file: the configured root's base name ('Lucid'). No layer (kernel correlate, pluginhost, plugin Match) interprets wildcards, so labelMatchesAny(['Lucid'], ['*']) is false for every item, the Match RPC returns zero items, and zero rows persist for (test, files) despite a fully healthy sync. Compounding product gaps: (a) match values are never validated against anything — a can-never-match value loads and syncs silently 'ok'; (b) the same plugin's include_glob/exclude_glob extras DO accept doublestar globs, inviting exactly this confusion; (c) 'match everything from this instance' is inexpressible for a recursive source, since nested files' labels never include the root base name."
fix: ""
verification: ""
files_changed: []
oracle_type: n/a (diagnose-only)
