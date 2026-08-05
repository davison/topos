# Phase 5: Source Instances & Per-Type Matching - Context

**Gathered:** 2026-08-06
**Status:** Ready for planning

<domain>
## Phase Boundary

Sources become named instances and matching becomes typed. The same plugin type can be configured multiple times under user-chosen display names ("Home email" / "Work email"), and source identity throughout the kernel — index rows, sync runs, agent grants, HTTP API, UI display — becomes the named instance, never the bare plugin type (KERN-06). The single shared per-webspace keyword list is replaced by per-instance matching config typed to each plugin (IMAP folders/labels, document tags, chat conversation/group names, wiki tags/pages), with all five existing sources migrated (KERN-07). The contract change is published across `docs/plugin-contract.md`, `proto/topos/v1/`, `config.example.toml`, and the mock plugin, with all standing contract tests (read-only AST, RPC allowlist, egress pinning) still passing.

**Folded into this phase (rides the same contract break):** rendition presentation moves from plugins to the kernel — plugins return content, the kernel sanitizes (per content-shape policy), wraps, and themes the iframe documents it serves. This removes the per-plugin hardcoded `themeStyle` duplication (see Folded Todos below).

This phase deliberately reverses Phase 1 D-02/D-03's shared-keyword-list shape (flagged costly-to-reverse at the time); the 2026-08-05 restructure accepted that cost now, while all five plugins are in-repo, as the foundation for the plugin-ecosystem direction (backlog 999.1). No UI redesign (Phase 6), no config-writing UI (Phase 7), no WhatsApp (Phase 8).

</domain>

<decisions>
## Implementation Decisions

### Config shape (webspace-centric + fallback)
- **D-01:** Matching config lives **webspace-centric**: a webspace declares per-instance match blocks (`[webspaces.house-move.match.home-email]` with typed fields like `folders = [...]`), plus an optional webspace-level `keywords = [...]` **fallback** applied (as native-name matching) to any participating instance without an explicit block. — **Reversibility:** costly — the shape is baked into the config loader, plugin contract match request, docs, the user's hand-maintained file, and Phase 7's builder UI edits this exact structure.
- **D-02:** An explicit match block **replaces** the fallback for that instance (no hidden union) — what's in the block is exactly what matches.
- **D-03:** **All configured instances participate in a webspace by default** (via fallback or match block); a webspace may set an optional `sources = ["work-email", "paperless"]` allowlist to restrict participation (e.g. a work webspace excluding Home email).

### Matching semantics
- **D-04:** Match values are **exact, case-insensitive native names only** — Phase 1 D-03's determinism guarantee carries into the typed fields unchanged. No globs, no hierarchy expansion, no plugin-specific syntax in this phase; richer per-type matchers can be added later without another contract break. — **Reversibility:** reversible — widening semantics later is additive.
- **D-05:** **Each plugin declares its match-field vocabulary** (e.g. `folders`, `tags`, `conversations`, `pages`) in its Describe response; the kernel validates config against the declared vocabulary and an unknown field **fails loudly at startup, naming the field and the plugin**. Ecosystem-ready: external plugins bring their own vocabulary, no kernel table to update. — **Reversibility:** costly — becomes part of the published contract third parties implement.

### Migration & compatibility
- **D-06:** **Clean break** on config: the kernel rejects an old-shape config with a clear error pointing at the documented new shape. No auto-migration code, no dual-shape support. The user's real `~/.config/topos/config.toml` is hand-migrated as part of this phase's delivery. — **Reversibility:** one-way — the old shape's code path is deleted; restoring it would be a rewrite, and the published contract moves on.
- **D-07:** **Index is dropped and fully re-synced** on upgrade rather than migrated in place — the index is a cache by design (hybrid model); source identity on rows re-derives from the new instance identity at first sync. One full backfill accepted (Proton re-scan is the slow one, minutes).

### Instance identity & lifecycle
- **D-08:** **The TOML map key is the instance's durable identity** (`[sources.home-email]` → instance id `home-email`) used in the index, API, and match blocks. Renaming the key = a new instance: old rows vanish at next sync, new ones backfill — index mirrors config truth, same philosophy as Phase 4 D-07 renames. `display_name` changes are purely cosmetic.
- **D-09:** `display_name` is **optional (defaults to the instance key) and must be unique across instances**; duplicates are rejected loudly at config load. Two chips both labeled "Email" would be unusable.
- **D-10:** Agent grants (`[sources.<instance>.agent]`, default-deny, Phase 2 D-11) are **per instance** — falls out of the map shape naturally; each instance of the same plugin type carries its own independent grants.

### Rendition presentation (folded todo)
- **D-11:** Plugins stop emitting themed full HTML documents. They return content plus content-shape metadata; the kernel's content-serving boundary (`kernel/httpapi/item.go`, which already owns the rendition CSP) **sanitizes per content-shape policy profile (email style-allowlist, chat no-styles, markdown), wraps in a single kernel-owned document skeleton, and themes from one kernel-owned stylesheet** derived from the same tokens as `web/src/app.css`. The Phase 3 rule "the producing plugin decides readability" (text vs HTML choice) stays plugin-side — only presentation and the trust boundary move. — **Reversibility:** costly — this is a Fetch response-shape change in the published contract; it is also a prerequisite for the ecosystem milestone (untrusted third-party plugins must not own sanitization).

### Claude's Discretion
- Proto mechanics of the contract change (how MatchRequest/DescribeResponse/FetchResponse evolve, contract version bump strategy, RPC-allowlist updates).
- Exact TOML field/key names beyond the shapes shown in D-01 previews; validation error wording.
- Content-shape taxonomy details and per-shape sanitizer policy specifics (guided by the existing three plugin sanitizer policies).
- How the kernel-owned rendition stylesheet stays in sync with `web/src/app.css` tokens (shared source, generation, or disciplined duplication in ONE place).
- Order of work within the phase (identity rewiring vs matching config vs rendition move) and migration sequencing for the user's live config.

### Folded Todos
- **Centralize rendition theming (and sanitization) in the kernel content boundary** (`.planning/todos/pending/2026-08-05-centralize-rendition-theming-in-kernel.md`) — hardcoded per-plugin `themeStyle` constants don't scale to external plugins; a theme change during the 260805-j98 scrollbar task required editing three plugins. Folded because this phase already breaks and republishes the plugin contract — one break instead of two. User chose theming **and** sanitization kernel-side (D-11). The todo file should be moved to completed when this phase ships.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope & requirements
- `.planning/ROADMAP.md` — Phase 5 goal, 4 success criteria, notes (reverses Phase 1 D-02/D-03 deliberately; foundation for backlog 999.1 plugin ecosystem).
- `.planning/REQUIREMENTS.md` — KERN-06, KERN-07 (this phase), KERN-08/UI-12 (Phase 7 — do not build, but don't preclude), and the KERN-07 note that KERN-01 remains satisfied via migration.
- `.planning/PROJECT.md` — constraints (read-only, privacy, extensibility) and Key Decisions table.
- `.planning/todos/pending/2026-08-05-centralize-rendition-theming-in-kernel.md` — the folded rendition work: full problem statement and solution sketch (D-11 locks its direction).

### Prior locked decisions this phase builds on or supersedes
- `.planning/phases/01-first-webspace-end-to-end/01-CONTEXT.md` — D-01 (TOML), D-04 (one config file, env-var secrets) still stand; D-02/D-03's *shared list* shape is superseded by this phase (exactness semantics survive per D-04 above).
- `.planning/phases/02-two-sources-one-trustworthy-stream/02-CONTEXT.md` — coordinator single-flight semantics, health/staleness rules, agent-grant shape (now per instance, D-10), `/agent/v1` namespace behavior.
- `.planning/phases/04-signal-conversations/04-CONTEXT.md` — day-digest shape and D-06 (1:1 matching against the user's own names for a contact) — the Signal plugin's `conversations` field must preserve this behavior.

### Published contracts (this phase breaks and republishes them)
- `docs/plugin-contract.md` — the third-party-facing contract; must be rewritten for instance identity, declared match vocabulary, and the new Fetch content shape.
- `docs/api.md` — HTTP envelope; source identity in responses changes to instance; rendition serving behavior changes (kernel wraps/themes).
- `proto/topos/v1/` — `MatchRequest{keywords}` and `DescribeResponse` are the primary change sites; RPC-allowlist test gates any addition.
- `config.example.toml` — full rewrite to the new shape; it is the de facto config documentation.

### Code the folded rendition work moves or replaces
- `kernel/httpapi/item.go` — the content route + rendition CSP; the new wrap/sanitize/theme home.
- `plugins/proton/body.go`, `plugins/signal/render.go`, `plugins/silverbullet/render.go` — the three `themeStyle`/sanitizer implementations being centralized (their per-shape policies inform the kernel's policy profiles); each has render tests including the scrollbar assertions from 260805-j98 that must be relocated, not lost.
- `web/src/app.css` — token source the kernel-owned rendition stylesheet must stay consistent with (incl. the scrollbar rules).

### Technology stack (locked)
- `.claude/CLAUDE.md` — stack decisions; the kernel stays cgo-free (signal plugin isolation), `buf`-managed proto as the contract source of truth.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `kernel/config/types.go` — `Sources map[string]Source` already keys sources by name; the map key becomes the instance id (D-08). `Webspace` type and validation live here too.
- `proto/topos/v1/plugin.proto` — `DescribeResponse` (source_type, display_name, contract_version) is the natural carrier for the declared match vocabulary (D-05); `MatchRequest{repeated string keywords}` is the field being replaced.
- `kernel/httpapi/item.go` — already owns the rendition Content-Security-Policy; the wrap/sanitize/theme pipeline lands beside it.
- The three plugin sanitizer policies (bluemonday-based) are working per-shape references for the kernel's policy profiles: email narrow-style-allowlist (proton), chat no-styles (signal), markdown (silverbullet).

### Established Patterns
- Contract enforcement by test: RPC allowlist (`sdk/contract_test.go`), read-only AST tests, egress pinning — every contract change must keep these green and extend them.
- Loud config validation at load (this session's `display_name` uniqueness and unknown-match-field rules follow the same style as existing base_url/token validation).
- Sync identity is `(webspace, source_type)` since Phase 02-01 — this tuple's second element becomes the instance id; `ReplaceWebspaceSourceItems`/`SyncSource`, `sync_runs.source_type`, and `SyncingSourceTypes`/`LatestSyncRunPerSource` (just fixed in f01df96 — don't regress the orphan-reconciliation behavior) all rewire.
- `sourceDisplayName()` helper in the web UI (Phase 02-01) — display names now come from config/API instead of a hardcoded map.

### Integration Points
- `kernel/pluginhost` — instances of the same plugin type mean launching the same binary N times with different per-instance env/config; process-per-instance vs process-per-type is a research/planning question.
- `kernel/index` schema — `source_type` columns across items/sync_runs become instance ids; D-07's drop-and-re-sync makes this a schema change without data migration.
- `kernel/syncer` coordinator — single-flight keying moves from plugin type to instance.
- `web/src/lib/components/` — SourceHealthChip/SourceFilterChips/DetailPane consume instance display names via the API; minimal UI change here (the real source-surface redesign is Phase 6).

</code_context>

<specifics>
## Specific Ideas

- The user's motivating example for instances: two future GMail accounts shown as "Home email" and "Work email" in the UI (2026-08-04 restructure discussion).
- The user's stated driver for per-type matching: "The mechanism of having a single search list applied to all plugins was a poor choice. We will need tag/search config specific to plugin types."
- The user's stated driver for the rendition fold: "forcing a matching style into the plugin code isn't scaleable and will be a major issue when plugins all become external with their own lifecycles and developers" (260805-j98 approval).

</specifics>

<deferred>
## Deferred Ideas

- Richer per-type matchers (IMAP folder-hierarchy expansion, globs, plugin-specific syntax) — deliberately excluded by D-04; additive later.
- Config auto-migration / config-writing code — Phase 7 (Webspace Builder UI) owns config writing.
- Scalable source-surface UI for many instances — Phase 6.

### Reviewed Todos (not folded)
- **Signal schema-version verify-and-accept tooling** (`.planning/todos/pending/2026-08-05-signal-schema-version-verify-and-accept-tooling.md`) — reviewed (matched at 0.6); deferred: standalone tooling that doesn't need the contract-break window. `plugins/signal/schema_readset.go` (260805-lry) is its seed.

</deferred>

---

*Phase: 5-Source Instances & Per-Type Matching*
*Context gathered: 2026-08-06*
