---
phase: 05-source-instances-per-type-matching
plan: 05
subsystem: docs
tags: [docs, toml, config, plugin-contract, http-api, matching, rendition]

# Dependency graph
requires:
  - phase: 05-source-instances-per-type-matching
    plan: 01
    provides: "Source instance identity (item.Source, config-key-trusted) split from the Describe-learned plugin kind (item.SourceType); D-08/D-09/D-10"
  - phase: 05-source-instances-per-type-matching
    plan: 02
    provides: "Typed match fields on the wire: MatchRequest.match_fields (map<string, StringList>), DescribeResponse.match_vocabulary; sdk.Handshake.ProtocolVersion 2; contract_version topos.v2"
  - phase: 05-source-instances-per-type-matching
    plan: 03
    provides: "config.MatchBlock, Webspace{Keywords, Sources, Match}, kernel/correlate.matchFieldsFor's allowlist->block->fallback resolution, pluginhost.ValidateMatchConfig"
  - phase: 05-source-instances-per-type-matching
    plan: 04
    provides: "kernel/httpapi/rendition.go's kernel-owned sanitize/wrap/theme boundary; proto ContentShape enum; FetchResponse.content_shape"
provides:
  - "docs/plugin-contract.md republished for the shipped instance-identity, typed-matching, and kernel-owned-presentation contract (ProtocolVersion 2, match_vocabulary, match_fields/StringList, ContentShape)"
  - "docs/api.md completed with unsupported_content_shape and the kernel-owned rendition sanitize/wrap/theme description (replacing the stale per-plugin-sanitizes claim)"
  - "config.example.toml rewritten as the de facto documentation of the per-instance match shape: display_name on every source, a commented two-instance worked example, and one worked [webspaces.<ws>.match.<instance>] example per in-repo plugin type"
  - "The operator's live config migrated by hand to the new shape (.planning/phases/05-source-instances-per-type-matching/migrated-config.toml), installed onto ~/.config/topos/config.toml with a pre-phase05 backup, and verified live: builds, tests, syncs, and serves all four real sources with no validation error"
affects: [06-ui-scalable-source-surface, 07-webspace-builder-ui]

# Actuals (#2632)
actuals:
  tokens: 12500
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "config.example.toml as documentation-by-example: every new key (display_name, match, sources) ships with the same purpose/default/validation-rule comment convention as every pre-existing key, and the file's worked examples were verified by actually loading them through kernel/config.Load with fake env vars, not just eyeballed for prose accuracy"
    - "Migration-by-hand as delivery work, not tooling: migrated-config.toml carries inline provenance comments (what changed, why the fallback keywords line is kept even though presently unused) so the file documents its own migration rationale rather than relying on the SUMMARY alone"

key-files:
  created:
    - .planning/phases/05-source-instances-per-type-matching/migrated-config.toml
  modified:
    - docs/plugin-contract.md
    - docs/api.md
    - config.example.toml
    - README.md

key-decisions:
  - "docs/plugin-contract.md's Item field table corrected beyond the plan's explicit scope (Rule 1 - bug): the source_id row still claimed the kernel derives the stable id as \"{source_type}:{source_id}\", which 05-01 already made false (it's \"{source}:{source_id}\", the instance id) — left uncorrected this would have actively misled a third-party plugin author reading the contract"
  - "config.example.toml's two-instance worked example (home-email/work-email, both topos-plugin-proton) is committed COMMENTED OUT, matching the file's own established convention for the [sources.mock]/[webspaces.demo] illustrations — it demonstrates the shape without silently doubling a copy-pasting operator's live credential requirements"
  - "The operator's real webspaces (house-move, test) were migrated with an explicit [webspaces.<ws>.match.<instance>] block for EVERY currently-configured instance, each field set to the SAME keyword values the old shared list carried — this reproduces the D-01 fallback's fan-out byte-for-byte (proven live: item counts per source and per webspace before/after are consistent) rather than leaving instances on the fallback path unexercised in the operator's own real file; the original keywords line is kept as a documented safety net for any future fifth instance, per the plan's own \"keep the keywords line as the fallback where an instance has no explicit block\" instruction"
  - "Task 3's live verification ran against an ephemeral second kernel instance (XDG_CONFIG_HOME override, distinct port, same already-synced index) rather than the user's own separately-running make dev session on 127.0.0.1:7777 (started earlier, in an unrelated terminal) — avoids disrupting a live user session while still exercising the real migrated config end to end"

patterns-established: []

requirements-completed: [KERN-06, KERN-07]

coverage:
  - id: D1
    description: "docs/plugin-contract.md documents instance identity, the declared match vocabulary, the typed Match request, the content-shape Fetch response, and the bumped handshake, verifiable against the shipped proto/sdk/mock plugin"
    requirement: "KERN-07"
    verification:
      - kind: unit
        ref: "make test (full six-module suite, unchanged behavior) plus a live grep cross-check: every ContentShape/ContentVariant/LinkFidelity enum value named in docs/plugin-contract.md exists verbatim in proto/topos/v1/plugin.proto"
        status: pass
    human_judgment: false
  - id: D2
    description: "docs/api.md documents the unsupported_content_shape error code and the kernel-owned sanitize/wrap/theme boundary description, replacing the stale per-plugin-sanitizes claim"
    requirement: "KERN-07"
    verification:
      - kind: unit
        ref: "grep -c 'unsupported_content_shape' docs/api.md (2), grep -c '{source}:{source_id}' docs/api.md (2)"
        status: pass
    human_judgment: false
  - id: D3
    description: "config.example.toml is a complete, valid example of the new shape: display_name on every source, two instances of one plugin type, a per-instance match block, a webspace keywords fallback, and a participation allowlist"
    requirement: "KERN-07"
    verification:
      - kind: unit
        ref: "node -e substring check (display_name, sources =, [webspaces., .match., conversations, folders, pages, tags all present); go test ./kernel/config/ -count=1; the active portion of the file loaded and validated end to end via a standalone config.Load call with fake env vars"
        status: pass
    human_judgment: false
  - id: D4
    description: "The standing contract tests (RPC allowlist, both enum zero-value pins, the read-only AST scan, the module pins, the egress host pinning) all pass unchanged in intent"
    requirement: "KERN-06"
    verification:
      - kind: unit
        ref: "make test — full six-module + root suite, run three times across this plan's tasks, exits 0 every time"
        status: pass
    human_judgment: false
  - id: D5
    description: "The operator's live config is migrated to the new shape and the kernel starts, syncs, and serves against it, with every configured instance visible under its own resolved display name and no config/match-vocabulary validation error"
    requirement: "KERN-06"
    verification:
      - kind: integration
        ref: "./bin/topos sync (real paperless/proton/signal/silverbullet item counts, no error); GET /api/sources on an ephemeral live instance (all four instances reachable/ok with resolved display_name); GET /api/webspaces/{ws}/stream (both webspaces populated); GET /api/items/{id}/content 200 for one email/chat/markdown item each"
        status: pass
      human_judgment: true
      rationale: "Verified mechanically via curl against a live kernel instance (real data, real sync, real HTTP responses) rather than by a human opening the actual web UI in a browser — the plan's own Task 3 <human-check> asks a human to visually confirm distinct source chips and pixel-level rendition parity in DetailPane.svelte's iframe, which this session could not do. Logged to .planning/WINDOWS.md (unrun-verify, entry 4) for follow-up."

duration: ~35min
completed: 2026-08-06
status: complete
---

# Phase 5 Plan 5: Source Instances & Per-Type Matching Summary

**Republished `docs/plugin-contract.md`/`docs/api.md` and rewrote `config.example.toml` for the shipped instance-identity, typed-match, and kernel-owned-rendition contract, then hand-migrated the operator's live config onto the new shape and verified it live — build, test, sync, and serve all pass against it with all four real sources reporting resolved display names and no validation error.**

## Performance

- **Duration:** ~35 min
- **Completed:** 2026-08-06
- **Tasks:** 3
- **Files modified:** 4 modified, 1 created

## Accomplishments

- `docs/plugin-contract.md`: handshake table corrected to `ProtocolVersion 2` with an explanation of why it moved; "Discovery and launch" gained a "the kernel may launch the same plugin binary more than once" section documenting the config-key-as-instance-identity model (a plugin still only ever declares `source_type`); `Describe` documents `match_vocabulary` in full (kernel validation, empty-vocabulary consequence, the four in-repo examples as illustrations only); `Match` is rewritten end to end for `match_fields`/`StringList`, the three implementation rules (read only declared keys, exact/case-insensitive, empty list matches nothing), and a corrected worked example matching the shipped `plugins/mock/plugin.go`; `Fetch` documents `content_shape` and the kernel-owned sanitize/wrap/theme boundary (D-11), including the plugin-author-facing "must not emit a full document or its own stylesheet" rule; a new `## ContentShape` section mirrors the existing `LinkFidelity`/`ContentVariant` sections; the `Item` table's stale `"{source_type}:{source_id}"` id-scheme claim is corrected to `"{source}:{source_id}"` (Rule 1 bug fix, beyond the plan's literal scope)
- `docs/api.md`: added the `unsupported_content_shape` (502) error-code row; rewrote the `GET /api/items/{id}/content` sanitization paragraph from "the producing plugin sanitizes via goldmark/bluemonday" (false since 05-04) to the kernel-owned `sanitizeAndWrapRendition` pipeline, its fail-closed behavior on an unrecognised/unspecified shape, and why `style-src 'unsafe-inline'` is still safe under kernel-only stylesheet injection
- `README.md`: "Status and roadmap" moved Phase 5 from "coming" to "complete" with a summary of what shipped (named instances, typed per-instance matching, kernel-owned rendition); the "Configure" section now distinguishes the `keywords` fallback from a per-instance `match` block
- `config.example.toml`: every source block gained `display_name` with D-09-accurate purpose/validation comments; a commented two-instance worked example (`home-email`/`work-email`, both `topos-plugin-proton`) demonstrates D-08/D-10 identity and independent agent grants; the webspaces section is rewritten with the full new shape — the `keywords`-vs-`match`-block distinction (D-01/D-02), the `sources` participation allowlist (D-03), one worked `[webspaces.<ws>.match.<instance>]` example per in-repo plugin type (paperless `tags`, silverbullet `tags`+`pages`, proton `folders`, signal `conversations`), and all five load-time failure modes an operator can hit, listed explicitly
- `.planning/phases/05-source-instances-per-type-matching/migrated-config.toml`: the operator's real `~/.config/topos/config.toml` hand-migrated onto the new shape — every `${VAR}` reference, `ca_cert`, `path`, `webmail_base_url`, `api_version`, and `[sources.*.agent]` block preserved byte for byte; `display_name` added to all four sources (each plugin's own `Describe`-reported label — `paperless-ngx`, `SilverBullet`, `Proton Mail`, `Signal`); both real webspaces (`house-move`, `test`) gained an explicit per-instance match block for every configured instance, values identical to the prior shared keyword list, with the original `keywords` line kept as a documented fallback safety net
- Installed: `~/.config/topos/config.toml.pre-phase05.bak` (backup of the pre-migration file) and the migrated content copied onto `~/.config/topos/config.toml`
- Live verification: `make build && make test` pass; `./bin/topos sync` succeeds against all four real sources (paperless 37, proton 44, signal 110, silverbullet 16 items) with no config/match-vocabulary validation error; an ephemeral `./bin/topos serve` instance (separate port via `XDG_CONFIG_HOME`, same already-migrated index) confirmed `GET /api/sources` reports all four instances `reachable`/`ok` under their resolved display names, both webspaces (`house-move` 102 items, `test` 105 items) populate via `GET /api/webspaces/{ws}/stream`, and a live `text/html` rendition of each content shape (Proton email, Signal chat, SilverBullet markdown) serves `200` through the kernel-owned sanitize/wrap pipeline (confirmed via the response body's kernel-composed `<!doctype html>`/`<style>` wrapper)

## Task Commits

1. **Task 1: Republish the plugin contract and the HTTP API reference** — `70bce9b` (docs)
2. **Task 2: Rewrite config.example.toml as the new shape's documentation** — `50dee53` (docs)
3. **Task 3: Migrate the operator's live config and verify the running system** — `7146636` (docs)

_No plan-metadata commit yet — SUMMARY.md/STATE.md/ROADMAP.md/REQUIREMENTS.md land in the final metadata commit per the execution workflow's `<final_commit>` step._

## Files Created/Modified

- `docs/plugin-contract.md` — republished contract: handshake v2, multi-instance launch, `match_vocabulary`, `match_fields`/`StringList` Match rewrite, `content_shape`/`ContentShape`, corrected stable-id scheme
- `docs/api.md` — `unsupported_content_shape` error code, kernel-owned sanitize/wrap/theme description
- `README.md` — Phase 5 marked complete; keywords-vs-match-block distinguished in Configure
- `config.example.toml` — `display_name` everywhere, two-instance worked example, full webspaces-section rewrite
- `.planning/phases/05-source-instances-per-type-matching/migrated-config.toml` (new) — the operator's real config, hand-migrated

## Decisions Made

- Corrected `docs/plugin-contract.md`'s stale `"{source_type}:{source_id}"` id-scheme claim to `"{source}:{source_id}"` beyond the plan's literal file-touch instructions (Rule 1 — this was an active inaccuracy left over from before 05-01, not new drift this plan introduced).
- Kept the two-instance `home-email`/`work-email` worked example commented out in `config.example.toml`, consistent with the file's own `[sources.mock]`/`[webspaces.demo]` convention for illustrative, non-default content.
- Migrated the operator's real webspaces with an explicit per-instance match block for every configured instance (not left on the bare fallback), reproducing the fallback's exact fan-out with the same values — this exercises the new shape in the operator's own real file rather than leaving it implicit, while the retained `keywords` line still documents the fallback for any future instance.
- Verified the live migration against a second, ephemeral kernel instance on a separate port (via `XDG_CONFIG_HOME`) rather than restarting the user's own pre-existing `make dev` session on port 7777, to avoid disrupting unrelated live work in that session.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `docs/plugin-contract.md`'s Item table still claimed the pre-05-01 id scheme**
- **Found during:** Task 1, cross-checking the `Item` message table against `docs/api.md`'s already-updated "stable-ID scheme" section
- **Issue:** The `source_id` row read `"the kernel derives its own global id as \"{source_type}:{source_id}\""` — false since 05-01 moved identity to the source INSTANCE id, not the plugin kind
- **Fix:** Corrected both the `source_id` and `source_type` table rows to describe the instance-id-keyed scheme, cross-referencing "Discovery and launch" and `docs/api.md`
- **Files modified:** `docs/plugin-contract.md`
- **Verification:** Re-read the corrected table against `docs/api.md`'s "The stable-ID scheme" section for consistency
- **Committed in:** `70bce9b` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 — bug)
**Impact on plan:** A pre-existing documentation inaccuracy corrected while already editing the same file for its own declared scope; no scope creep beyond what the plan's own acceptance criteria ("every proto message, field and enum value named in `docs/plugin-contract.md` exists in the shipped `.proto`") already implied for contract accuracy.

## Issues Encountered

- The user's own `make dev` session (started earlier, in a separate terminal, PID unrelated to this session) was already bound to `127.0.0.1:7777` when Task 3 attempted its own `./bin/topos serve` verification run. Resolved by using `XDG_CONFIG_HOME` to point a second, ephemeral kernel instance at a copy of the migrated config on a different port, sharing the same already-synced index file — verified the real migrated config end to end without touching the user's own running session. That pre-existing session is still running against whatever config it loaded at its own earlier startup; it will pick up the newly-installed `~/.config/topos/config.toml` on its own next restart.

## User Setup Required

None for this plan's own deliverables. One informational note: the user's separately-running `make dev` session (port 7777, started before this plan ran) has not been restarted and is therefore still operating on the config it loaded at its own startup — it will need a normal restart (stop and re-run `make dev`) to pick up the newly-installed migrated config at `~/.config/topos/config.toml`. The pre-migration file is preserved at `~/.config/topos/config.toml.pre-phase05.bak`.

## Known Stubs

None — this plan touched only documentation, an example config file, and a hand-migrated live config; no code stubs were introduced.

## Next Phase Readiness

- ROADMAP success criterion 4 (contract publication) is satisfied: `docs/plugin-contract.md`, `proto/topos/v1/`, `config.example.toml`, and `plugins/mock` all reflect per-instance match config, and the standing contract tests (`make test`, three full runs across this plan's tasks) stay green throughout.
- ROADMAP success criterion 1 (two instances of one plugin type as separate sources) is demonstrated in `config.example.toml`'s worked example and structurally proven live via `GET /api/sources`'s four independently-named, independently-healthy source instances; the operator's own real deployment does not currently run two instances of one plugin type (no second Proton/paperless/etc. account configured), so this remains demonstrated rather than exercised against the operator's own real data.
- Phase 6 (UI: Scalable Source Surface) and Phase 7 (Webspace Builder UI) can build on the now-published `match_vocabulary`/`match_fields`/`content_shape` contract and the `display_name`-bearing config shape without further contract rework.
- Outstanding gap logged to `.planning/WINDOWS.md` (unrun-verify, entry 4): a human should open `make dev`, confirm the source filter/health chips show one entry per configured instance under its own display name, and visually confirm rendition parity (dark background, typography, scrollbar, chat bubble styling/no accent) for one item from each of an email, SilverBullet, and Signal source — the plan's own Task 3 `<human-check>`, substituted this session with equivalent API-level verification against a live ephemeral instance rather than an actual browser session.

---
*Phase: 05-source-instances-per-type-matching*
*Completed: 2026-08-06*

## Self-Check: PASSED

All key files verified present on disk (`docs/plugin-contract.md`, `docs/api.md`, `config.example.toml`, `README.md`, `.planning/phases/05-source-instances-per-type-matching/migrated-config.toml`, this SUMMARY); all three task commits (`70bce9b`, `50dee53`, `7146636`) verified present in `git log --oneline --all`.
