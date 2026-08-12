# Phase 7: Webspace Builder UI - Context

**Gathered:** 2026-08-07
**Status:** Ready for planning

<domain>
## Phase Boundary

Configure sources and webspaces from the UI instead of hand-editing TOML (KERN-08, UI-12): pick plugin types from a list, configure named instances (non-secret fields only — secrets stay environment-only), save the configured set as a webspace, and promote a live search into the webspace's permanent filter, refinable by further search. The kernel gains its **first mutating API surface**, scoped strictly to configuration — the plugin contract's read-only guarantee over source data is untouched. Hand-editing the config file remains fully supported: the UI write path and the file agree on one persisted form, and a hand-edited file is never clobbered blindly.

Builds directly on Phase 5's locked config shape (`[sources.<id>]` instance blocks keyed by identity, webspace-centric `[webspaces.<name>.match.<instance>]` typed match blocks, `keywords` fallback, `sources` allowlist) and Phase 6's header (source chips, search box). No new sources (WhatsApp is Phase 8), no agent chat, no write paths into any source.

</domain>

<decisions>
## Implementation Decisions

### Config persistence & clobber safety
- **D-01:** UI saves perform a **canonical rewrite** of the whole config file — one machine-written form regenerated on every save. Hand-written comments are flattened the first time the UI saves; no comment-preserving round-trip machinery is built. — **Reversibility:** reversible — a later comment-preserving writer could replace the serializer without changing the API or file location.
- **D-02:** Canonical style is **minimal + header pointer**: clean TOML with just the values, preceded by a short generated header comment ("managed by the topos UI — hand-edits are honored via Reload; see config.example.toml for full documentation"). No per-key generated doc comments.
- **D-03:** Clobber guard is an **optimistic lock on file content hash**: the kernel records the config file's hash at load; a UI save first re-checks it, and if the file changed on disk since, the save is **rejected** ("config changed on disk — review and retry"), the kernel reloads the newer file, and the UI refreshes. No merge attempt, no silent loss.
- **D-04:** **Single rolling backup**: every UI save first writes the outgoing file to `config.toml.bak`, overwriting the previous backup. No timestamped set, no backup directory.
- **D-05 (hard requirement):** `${ENV_VAR}` secret references are written back **verbatim, never expanded** — the kernel must retain the raw pre-expansion form of the config for persistence (expansion via os.Expand happens at load only). Secret *values* never appear in the file, the API, or the UI. — **Reversibility:** one-way in effect — writing an expanded secret into the file even once is a privacy breach that cannot be un-shipped; treat as a tested invariant.

### Apply / reload semantics
- **D-06:** **Save = apply immediately**: one request validates, writes the file, and hot-swaps the kernel's running config. No restart, no separate apply step. — **Reversibility:** costly — hot-apply requires in-process reconfiguration of pluginhost (launch/shutdown instances) and syncer registration; backing down to restart-required would abandon that machinery.
- **D-07:** Reconciliation after apply is **eager**: a new or connection-changed instance gets its first sync immediately (mirroring startup behavior); a changed webspace/match block immediately re-syncs the affected sources; a removed instance's plugin process shuts down and its index rows are removed right away (index mirrors config — Phase 5 D-07/D-08 precedent). The stream reflects an edit within one sync, not one interval.
- **D-08:** Hand-edits reach the running kernel via an **explicit Reload affordance** (UI button + API endpoint) that re-reads the file through the same validate-then-apply path. **No file watcher.** An invalid file on reload keeps the last-good config running and surfaces the error; the kernel never dies on a bad reload.
- **D-09:** Validation is **validate-on-save only**: the save endpoint runs the kernel's full existing load-time validation as a dry-run before writing; on failure nothing is written and the UI shows the kernel's error messages (which already name field + plugin). No live per-field validation endpoint, no client-side reimplementation of the rules — one validation code path.

### Builder UX shape (locked by user: "yes, that matches — lock it in")
- **D-10:** **Full in-header composition — no standalone settings section, and the standalone home page is retired.** The webspace title in the header becomes a **drop-down webspace switcher** listing all webspaces, with a "+" beside it to create a new (empty) webspace. Root URL redirects to the first/last-visited webspace; a zero-webspaces empty state hosts first-run creation. — **Reversibility:** reversible — routes/components; refinement explicitly expected after it's seen working.
- **D-11:** A **"+" at the end of the source-chip row** adds a source to the current webspace. The picker offers **existing configured instances** (e.g. "Home email") and **"New <plugin type>…"** entries for discovered plugin binaries. Picking an existing instance opens a modal asking **only for match fields**; picking a plugin type opens a **two-step modal: connection config (instance) then match config (webspace)**. Match-field forms are driven by the plugin's declared match vocabulary (Describe RPC, Phase 5 D-05).
- **D-12:** Editing an existing source's config happens via a **chip menu/popover affordance — never plain chip click** (click stays filter-toggle per Phase 6 D-01). Editing instance-level connection fields must be visibly marked as affecting every webspace using that instance.
- **D-13:** A minimal **"Manage sources…" entry in the title drop-down** is the escape hatch for instance-level edit/delete, webspace deletion, and the Reload-config affordance (D-08). No other global settings surface.
- **D-14:** **UI-built webspaces always write an explicit `sources` allowlist** — participation is exactly what was added via "+". Hand-written webspaces without an allowlist keep Phase 5 D-03's all-instances-participate default. The two coexist; the builder never silently rewrites a hand-written webspace's participation model unless the user edits it in the UI.
- **D-15:** Secret fields in the modal ask for the **environment variable name** (persisted as `token = "${VAR}"`), with a **set/unset badge** reported by the kernel for that variable in its environment. Unset ⇒ save still succeeds with a warning ("add it to .env and restart before this source can connect") — the health chip then reports the truth as usual. Values are never displayed or transmitted.

### Search-promotion filter semantics
- **D-16:** A promoted filter acts at **query time as an FTS filter, and is also applied to the `/agent/v1` read surface** — the filtered view IS the webspace for every consumer (human UI and agent API). Index contents stay exactly as match config dictates; no sync-time narrowing. — **Reversibility:** costly — agent-surface behavior becomes observable API contract; retracting it changes what agents see.
- **D-17:** Filters persist as a **config key on the webspace block** (part of the webspace's definition), riding the whole Phase 7 machinery: builder writes it, hand-editing works, optimistic lock guards it, Reload applies it — one persisted form (success criterion 3).
- **D-18:** Filters are a **stackable AND list** (e.g. `filter = ["boiler", "quote"]`): each promotion appends a term; all terms AND together; each is removable independently. Live searches AND with the active filter stack ("further searches refine within that filter").
- **D-19:** UI surfacing: after a search, a **"Save as filter" affordance** appears by the search box; active permanent filters render as **labeled chips visually distinct from source chips**, each with an × to remove. Removing writes config immediately through the same save path.

### Gap closure — 2026-08-09 (07-UAT.md `G-07-3`, `.planning/debug/create-webspace-rejected-empty.md`)
- **D-20:** An **"empty webspace shell"** is a `[webspaces.<name>]` block declaring **all three** of `keywords`, `sources`, and `match` empty at once — no keywords fallback, no allowlist, no match blocks. `config.Validate` **accepts** this shape (`Webspace.IsEmptyShell`, checked before any other per-webspace rule) and it **correlates nothing**: `kernel/correlate.matchFieldsFor` mirrors the same three-condition test and reports non-participation for every source instance rather than fanning an empty `keywords` list across a plugin's vocabulary (which would read to the plugin as "no constraint" and risk flooding the index with its entire corpus). This is the exact document `web/src/lib/config-edit.ts`'s `addWebspace()` PUTs as the create-webspace modal's first (of two) writes — 05-03 D-01's "keywords or match always required" invariant and 07-03/07-04's D-14 two-write create flow were mutually exclusive at the instant that first PUT was submitted, and this decision reconciles them.
  - **Option (a)** — a participation-aware exemption in validation for a genuinely source-less shell — was **chosen**, over:
  - **(c) single-write create-and-seed-first-source**, rejected because it is unimplementable on the flow that most needs it: a first-run install has zero `[sources.*]` blocks, so a create modal requiring a first source has nothing to offer.
  - **(b) the modal's single PUT writes a document that already satisfies the invariant** (e.g. seeding `keywords = ["<webspace name>"]`), rejected as dishonest and unremovable: it would make a brand-new webspace silently all-participate against a guessed term the UI has no editor to remove.
  - `validateFallbackCoverage` (05-03 D-06's guard against a participating instance with nothing to match) and `Webspace.Participates` (Phase 5 D-03's empty-allowlist-means-all-participate default) are **both unchanged** — the shell exemption short-circuits BEFORE either is reached, so strictness for every hand-edited config is provably preserved: a webspace naming an instance under `sources` while declaring neither keywords nor match is NOT a shell and still fails load with the existing message, and a webspace covering only some of its participating instances still fails naming the uncovered one. — **Reversibility:** costly — once operators have persisted shell webspaces through the UI, reverting the exemption means a kernel that will not load a file it wrote itself.

### Claude's Discretion
- Mutating-API design (endpoint shapes, HTTP verbs, request/response envelope) — must follow `docs/api.md` conventions; loopback-only/no-auth posture is the existing v1 stance and unchanged this phase.
- How the kernel retains the raw pre-expansion config for canonical rewrite (re-parse on save vs. retained AST/document model), TOML serializer choice, and atomic-write mechanics (temp file + rename).
- Hot-apply internals: diffing old vs. new config, pluginhost instance lifecycle ordering, syncer re-registration, in-flight sync handling during apply.
- Modal/form layout details, picker presentation, delete-confirmation UX, first-run empty state design, and what "first/last webspace" the root redirect targets.
- FTS query semantics for stacked filter terms (phrase vs. token handling), and how filter chips interact visually with the existing source-chip filtering row.
- Where the "+"-picker learns about available-but-unconfigured plugin types (plugin binary discovery is existing kernel behavior).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope & requirements
- `.planning/ROADMAP.md` — Phase 7 goal, 4 success criteria, notes (TOML round-tripping vs UI-owned layer flagged pre-planning — resolved by D-01; search-promotion called the heart of the phase).
- `.planning/REQUIREMENTS.md` — KERN-08, UI-12 (this phase); SRC-03 is Phase 8, AGENT-10..12/UI-10 are v1.x — do not build.
- `.planning/PROJECT.md` — constraints (read-only sources, privacy, secrets never in files/commits) and Key Decisions table.

### Prior locked decisions this phase builds on
- `.planning/phases/05-source-instances-per-type-matching/05-CONTEXT.md` — the config shape the builder edits: D-01 (webspace-centric match blocks + keywords fallback), D-02 (block replaces fallback), D-03 (default-all participation + optional `sources` allowlist — interacts with this phase's D-14), D-05 (plugin-declared match vocabulary drives the modal forms), D-08 (map key = instance identity; renaming = new instance), D-09 (display_name uniqueness), D-10 (per-instance agent grants).
- `.planning/phases/06-ui-scalable-source-surface/06-CONTEXT.md` — chip interaction model (D-01 click = filter — this phase's D-12 must not collide), multi-select filtering + URL persistence (D-02), hover-revealed refresh (D-03), tooltip health (D-04).
- `.planning/phases/01-first-webspace-end-to-end/01-CONTEXT.md` — D-01 (TOML), D-04 (one config file, env-var secrets) — both still stand and constrain the write path.

### Published contracts (extend carefully; first mutating surface)
- `docs/api.md` — HTTP envelope and conventions the new config endpoints must follow; agent surface (`/agent/v1`) behavior changes per D-16.
- `docs/plugin-contract.md` — NOT changed this phase; the read-only guarantee over source data is explicitly untouched (success criterion 4).
- `config.example.toml` — the de facto config documentation the canonical writer's header points at (D-02); gains the webspace `filter` key (D-17/D-18) and any allowlist notes from D-14.
- `proto/topos/v1/` — Describe's declared match vocabulary consumed by the modal forms; no proto change expected.

### Technology stack (locked)
- `.claude/CLAUDE.md` — SvelteKit 2/Svelte 5 SPA, Go kernel, chi router, kernel stays cgo-free.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `kernel/config/types.go` + `config.go` — the full config model and the loud load-time validation (D-09 reuses it as the save dry-run); `Sources map[string]Source`, `Webspace`, match-block validation all live here.
- `kernel/httpapi/routes.go` — route mounting for the new config endpoints; `sources.go`/`webspaces.go` show the envelope conventions.
- `web/src/lib/components/WebspaceHeader.svelte` — the site of nearly all new UI: title → switcher drop-down (D-10), chip-row "+" (D-11), filter chips row (D-19).
- `web/src/lib/components/SourceChip.svelte` — the chip whose menu/popover gains the edit affordance (D-12); its click/hover/tooltip semantics are locked from Phase 6.
- `web/src/lib/components/SearchBox.svelte` / `SearchResults.svelte` — the "Save as filter" affordance site (D-19).
- `web/src/lib/components/ui/` — shadcn-derived primitives for modals/drop-downs/forms.
- `kernel/pluginhost` — plugin binary discovery (feeds the "New <plugin type>…" picker) and instance process lifecycle (hot-apply's launch/shutdown, D-06/D-07).
- `kernel/syncer` — single-flight coordinator; eager reconcile (D-07) triggers through it, same as manual refresh.
- `kernel/index` — FTS5 search machinery (KERN-05) that the permanent filter (D-16) reuses at stream-read time; `ReplaceWebspaceSourceItems` handles removed-instance row cleanup.

### Established Patterns
- Loud fail-at-load validation naming field + plugin — the save dry-run must surface these messages verbatim to the UI (D-09).
- Index mirrors config (Phase 5 D-07/D-08): renames = new instance, removed = rows gone — hot-apply reconcile (D-07) follows the same philosophy.
- Filter state persists in the URL query (Phase 2 D-09 / Phase 6 D-02) — the permanent filter is config-persisted instead (D-17); transient search stays URL/state-side. Keep the two visually and semantically distinct (D-19).
- Contract enforcement by test (RPC allowlist, read-only AST, egress pinning) — all must stay green; success criterion 4 wants an explicit guard that the mutating surface is config-scoped only.
- Sync-failure branch renders before empty branch — new builder states (saving, save-rejected, reloading) must not regress header gating.

### Integration Points
- `kernel/httpapi` — new mutating endpoints (config read/write/reload, env-var set/unset check, dry-run validation as part of save) — the first non-GET surface; `agent.go` gains the filter application (D-16).
- `kernel/config` — gains the canonical TOML writer (D-01/D-02), content-hash tracking (D-03), `.bak` write (D-04), raw pre-expansion retention (D-05), and the webspace `filter` field (D-17).
- Kernel startup vs. hot-apply — the apply path (D-06) refactors what today happens only at boot: config load → pluginhost launch → syncer registration.
- `web/src/routes/` — root route becomes a redirect (D-10); `/w/[webspace]` page absorbs builder state; the `+page.svelte` header wiring extends.

</code_context>

<specifics>
## Specific Ideas

- User's builder vision, quoted and locked: webspace title becomes a drop-down of webspaces with a "+" to create ("home page effectively redundant"); on any webspace, a "+" next to the configured source chips "selects a plugin from available sources and then a modal form can appear to accept name, tags, or whatever the plugin needs."
- User expectation on polish: "Once we see it working, it can be refined a little, but I doubt any big changes would be needed" — plan for a working composition flow first, not pixel-perfect modals.
- Backup preference stated as "keep a single .bak file only" — deliberately not a rotated set.

</specifics>

<deferred>
## Deferred Ideas

- Comment-preserving TOML round-trip — declined in favor of canonical rewrite (D-01); could return as a serializer swap if hand-authored comments matter again.
- File watcher / auto-reload of hand-edits — declined (D-08); explicit Reload only.
- WR-01 (Phase 6 review advisory: `highlightText` case-fold positional bug in `web/src/lib/format.ts`) — offered as a fold, declined; remains an open advisory for `/gsd-code-review 6 --fix` or a later phase.

### Reviewed Todos (not folded)
- **Signal schema-version verify-and-accept tooling** (`.planning/todos/pending/2026-08-05-signal-schema-version-verify-and-accept-tooling.md`) — matched this phase only at 0.2 (keyword noise); explicitly declined, stays deferred.

</deferred>

---

*Phase: 7-Webspace Builder UI*
*Context gathered: 2026-08-07*
