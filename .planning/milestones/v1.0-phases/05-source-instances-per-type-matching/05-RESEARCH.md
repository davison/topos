# Phase 5: Source Instances & Per-Type Matching - Research

**Researched:** 2026-08-06
**Domain:** Kernel identity model rewire (config → index → sync → HTTP/agent API → UI) + gRPC contract evolution + kernel-owned content sanitization/theming
**Confidence:** HIGH (in-repo architecture — every claim below is grounded in this session's `Read` of the actual source, not external unknowns)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Config shape (webspace-centric + fallback)**
- **D-01:** Matching config lives **webspace-centric**: a webspace declares per-instance match blocks (`[webspaces.house-move.match.home-email]` with typed fields like `folders = [...]`), plus an optional webspace-level `keywords = [...]` **fallback** applied (as native-name matching) to any participating instance without an explicit block. — **Reversibility:** costly — the shape is baked into the config loader, plugin contract match request, docs, the user's hand-maintained file, and Phase 7's builder UI edits this exact structure.
- **D-02:** An explicit match block **replaces** the fallback for that instance (no hidden union) — what's in the block is exactly what matches.
- **D-03:** **All configured instances participate in a webspace by default** (via fallback or match block); a webspace may set an optional `sources = ["work-email", "paperless"]` allowlist to restrict participation (e.g. a work webspace excluding Home email).

**Matching semantics**
- **D-04:** Match values are **exact, case-insensitive native names only** — Phase 1 D-03's determinism guarantee carries into the typed fields unchanged. No globs, no hierarchy expansion, no plugin-specific syntax in this phase; richer per-type matchers can be added later without another contract break. — **Reversibility:** reversible — widening semantics later is additive.
- **D-05:** **Each plugin declares its match-field vocabulary** (e.g. `folders`, `tags`, `conversations`, `pages`) in its Describe response; the kernel validates config against the declared vocabulary and an unknown field **fails loudly at startup, naming the field and the plugin**. Ecosystem-ready: external plugins bring their own vocabulary, no kernel table to update. — **Reversibility:** costly — becomes part of the published contract third parties implement.

**Migration & compatibility**
- **D-06:** **Clean break** on config: the kernel rejects an old-shape config with a clear error pointing at the documented new shape. No auto-migration code, no dual-shape support. The user's real `~/.config/topos/config.toml` is hand-migrated as part of this phase's delivery. — **Reversibility:** one-way — the old shape's code path is deleted; restoring it would be a rewrite, and the published contract moves on.
- **D-07:** **Index is dropped and fully re-synced** on upgrade rather than migrated in place — the index is a cache by design (hybrid model); source identity on rows re-derives from the new instance identity at first sync. One full backfill accepted (Proton re-scan is the slow one, minutes).

**Instance identity & lifecycle**
- **D-08:** **The TOML map key is the instance's durable identity** (`[sources.home-email]` → instance id `home-email`) used in the index, API, and match blocks. Renaming the key = a new instance: old rows vanish at next sync, new ones backfill — index mirrors config truth, same philosophy as Phase 4 D-07 renames. `display_name` changes are purely cosmetic.
- **D-09:** `display_name` is **optional (defaults to the instance key) and must be unique across instances**; duplicates are rejected loudly at config load. Two chips both labeled "Email" would be unusable.
- **D-10:** Agent grants (`[sources.<instance>.agent]`, default-deny, Phase 2 D-11) are **per instance** — falls out of the map shape naturally; each instance of the same plugin type carries its own independent grants.

**Rendition presentation (folded todo)**
- **D-11:** Plugins stop emitting themed full HTML documents. They return content plus content-shape metadata; the kernel's content-serving boundary (`kernel/httpapi/item.go`, which already owns the rendition CSP) **sanitizes per content-shape policy profile (email style-allowlist, chat no-styles, markdown), wraps in a single kernel-owned document skeleton, and themes from one kernel-owned stylesheet** derived from the same tokens as `web/src/app.css`. The Phase 3 rule "the producing plugin decides readability" (text vs HTML choice) stays plugin-side — only presentation and the trust boundary move. — **Reversibility:** costly — this is a Fetch response-shape change in the published contract; it is also a prerequisite for the ecosystem milestone (untrusted third-party plugins must not own sanitization).

### Claude's Discretion
- Proto mechanics of the contract change (how MatchRequest/DescribeResponse/FetchResponse evolve, contract version bump strategy, RPC-allowlist updates).
- Exact TOML field/key names beyond the shapes shown in D-01 previews; validation error wording.
- Content-shape taxonomy details and per-shape sanitizer policy specifics (guided by the existing three plugin sanitizer policies).
- How the kernel-owned rendition stylesheet stays in sync with `web/src/app.css` tokens (shared source, generation, or disciplined duplication in ONE place).
- Order of work within the phase (identity rewiring vs matching config vs rendition move) and migration sequencing for the user's live config.

### Deferred Ideas (OUT OF SCOPE)
- Richer per-type matchers (IMAP folder-hierarchy expansion, globs, plugin-specific syntax) — deliberately excluded by D-04; additive later.
- Config auto-migration / config-writing code — Phase 7 (Webspace Builder UI) owns config writing.
- Scalable source-surface UI for many instances — Phase 6.
- **Reviewed, not folded:** Signal schema-version verify-and-accept tooling (`.planning/todos/pending/2026-08-05-signal-schema-version-verify-and-accept-tooling.md`) — reviewed (matched at 0.6); deferred as standalone tooling that doesn't need the contract-break window.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| KERN-06 | Sources are named instances — the same plugin type can be configured multiple times, each under a user-chosen display name (e.g. "Home email" / "Work email") used throughout the UI and API | Architecture Patterns' "Recommended Task Ordering" (steps 2-5) and Pitfall 3 enumerate every kernel touchpoint (`kernel/pluginhost`, `kernel/index`, `kernel/syncer`, `kernel/correlate`, `kernel/httpapi/agent.go`, `kernel/httpapi/stream.go`, `kernel/httpapi/item.go`, `kernel/httpapi/sources.go`) where identity must move from Describe-learned `source_type` to the config map key (D-08), plus the existing multi-launch loop confirmation (Code Examples) that no change is needed in `pluginhost.Discover` itself. |
| KERN-07 | Matching config is declared per source instance and typed to its plugin (IMAP folders/labels, document tags, chat conversation/group names, wiki tags/pages), replacing the single shared per-webspace keyword list | Standard Stack's go-toml/v2 dynamic-key capability, Pitfall 1 (vocabulary-validation sequencing against plugin launch), Pitfall 2 (proto shape tradeoff for typed match fields), and Code Examples' exact-match-rule preservation across all four real plugins' existing `Match` implementations. |
</phase_requirements>

## Summary

Phase 5 is not primarily a "new feature" phase — it is a **rewire of the kernel's identity model**. Today, `source_type` (the plugin's own Describe-learned identity, e.g. `"proton"`) is used simultaneously as (a) the plugin's type/kind and (b) the sole identity key for every stateful and API-facing concept in the kernel: the composite item id (`"{source_type}:{source_id}"`), `items.source_type` and `sync_runs.source_type` index columns, the coordinator's `StartSyncRun`/`FinishSyncRun`/`SyncingSourceTypes`/`LatestSyncRunPerSource` calls, the agent-grant filter (`grantedSourceTypes` keyed by `source_type`, not by config name), the stream/item JSON's `source_type` field, and the frontend's per-source filter/health chips. Because `source_type` is shared by every instance of the same plugin binary, none of these will distinguish "Home email" from "Work email" today — two instances of `topos-plugin-proton` would silently conflate their items, sync history, and agent grants under one `source_type: "proton"` key. This conflation, not the TOML shape, is the real engineering core of KERN-06.

The fix (per CONTEXT.md D-08) is to promote the **config map key** (`[sources.<name>]`'s `<name>`, already called `Name()` on `pluginhost.Plugin` and already used as the singleflight/coordinator key and the `/api/sources` `name` field) to be the identity used everywhere `source_type` is used as a key today. `source_type` itself is demoted to a purely descriptive field (which plugin binary this instance runs) — still needed for Describe-reported match-vocabulary lookups, contract-version checks, and possibly UI iconography, but no longer an identity. This has a specific, subtle interaction with the existing "never trust a plugin's identity from its filename or config key" rule (`kernel/pluginhost/host.go`, T-01-07): that rule stays true for *plugin type* (still learned exclusively from `Describe`), but instance *identity* becomes explicitly config-key-trusted (D-08) — these are now two different questions the kernel asks, and the plan must not conflate them.

The second engineering thread is the typed, per-instance match config (KERN-07): replacing `Webspace.Keywords []string` (global) with a per-instance match block whose field *names* are plugin-declared (`folders`, `tags`, `conversations`, `pages` per the four current plugin types) and whose *validation* happens kernel-side against a vocabulary the plugin publishes in `DescribeResponse`. `go-toml/v2` supports this today via `map[string]interface{}`/`UnmarshalTOML`-driven dynamic-key decoding — no new dependency is needed.

The third thread, folded in because it also breaks `FetchResponse`, is moving rendition sanitization/theming from three duplicated per-plugin `themeStyle` implementations into `kernel/httpapi/item.go`.

None of the three threads requires a new external package. This is a pure refactor across `proto/topos/v1/plugin.proto`, `kernel/config`, `kernel/pluginhost`, `kernel/correlate`, `kernel/syncer`, `kernel/index`, `kernel/httpapi`, four plugin binaries, `docs/plugin-contract.md`, `docs/api.md`, `config.example.toml`, and four standing tests (`sdk/contract_test.go`'s RPC allowlist, `internal/audit/outbound_hosts_test.go`'s egress/AST scan, `kernel/httpapi/contract_test.go`'s AGENT-02 id/provenance shape pin, and the render-test suites in the three plugins whose `themeStyle` moves).

**Primary recommendation:** Treat this phase as an **identity-key rename sweep** (source_type-as-identity → instance-id-as-identity) that touches ~11 Go files plus 3 docs plus proto, done in a deliberate order (proto/contract first, then config, then index/sync plumbing, then plugin match-field vocabulary, then the kernel-owned rendition pipeline last since it's independently testable) — not as a set of unrelated small features.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Instance identity (config key → index/API identity) | Backend (kernel/config, kernel/pluginhost) | Database/Storage (kernel/index schema) | The TOML map key is the source of truth (D-08); the index is a derived cache that re-keys on it. |
| Declared match vocabulary (per plugin type) | Backend (proto DescribeResponse + kernel/config validation) | Plugin process (each plugin's Describe impl) | Plugin declares its own vocabulary; kernel enforces it at config-load time — no kernel-side table of "known plugin types" (ecosystem-ready per D-05). |
| Typed match config decode/validate | Backend (kernel/config) | — | TOML dynamic-key decode + vocabulary check is a config-load-time concern, never touches the plugin process. |
| Match execution (folders/tags/conversations/pages → items) | Plugin process (each plugin's `Match` RPC) | — | Unchanged responsibility — only the *input shape* (typed fields, not one keyword list) changes; the plugin still owns resolving its own native categorization. |
| Sync identity / coalescing / history | Backend (kernel/syncer, kernel/index) | — | `sync_runs`/singleflight already key on something instance-shaped in most places (coordinator uses `Name()`) but `sync_runs.source_type`/`SyncingSourceTypes` do not — this phase closes that gap. |
| Agent grant scoping (per instance) | Backend (kernel/config, kernel/httpapi/agent.go) | — | Falls out of the config map shape naturally (D-10) once grant filtering keys on instance id instead of `source_type`. |
| Rendition sanitize/theme | Backend (kernel/httpapi/item.go) | Plugin process (still decides content-shape/text-vs-HTML per D-11's carryover of the Phase 3 rule) | Trust boundary moves into the kernel ahead of the external-plugin ecosystem milestone; plugin still owns *which* representation is canonical. |
| Source display in UI (chips, filters) | Browser/Client (web/src) | Backend (HTTP API's `name`/`display_name` fields) | Minimal touch this phase — API already carries `name`; UI mostly needs to read display_name/instance id instead of `source_type` for filter/label purposes. Full redesign is Phase 6. |

## Package Legitimacy Audit

Not applicable — this phase adds **zero new external dependencies**. It edits existing Go packages (`kernel/*`, `plugins/*`, `sdk`), the existing `proto/topos/v1/plugin.proto`, and existing TOML config parsing via the already-vendored `github.com/pelletier/go-toml/v2` (`v2.4.3`, confirmed in `go.mod` this session — `[VERIFIED: go.mod:18]` — `github.com/pelletier/go-toml/v2 v2.4.3 // indirect`). No `npm view`/`pip`/`cargo` check is applicable; no package-legitimacy gate is triggered.

## Standard Stack

This phase does not introduce a new stack — it rewires the existing one. No additions to Core/Supporting tables are needed. The one library-capability question researched below is whether the existing TOML library supports what D-05 requires.

### Confirmed capability: dynamic per-plugin match vocabulary via go-toml/v2

`github.com/pelletier/go-toml/v2` (already a project dependency, `v2.4.3`) supports decoding a table with unknown/dynamic keys into `map[string]interface{}`, and separately supports a `DynamicConfig`-style pattern where a type's `UnmarshalTOML` method reads a discriminator field first (here: which plugin type owns this match block) and then decodes the rest against that type's own expected shape `[CITED: github.com/pelletier/go-toml README.md, v2 branch]`. This is exactly the shape D-05 needs: `[webspaces.house-move.match.home-email]` has field names (`folders`, `tags`, `conversations`, `pages`, …) that only the receiving plugin's Describe-declared vocabulary can validate — the kernel config loader does not need a compile-time struct per plugin type. Two viable implementation strategies, either acceptable:
- **Late-bound validation**: decode every `[webspaces.*.match.*]` block into `map[string][]string` (TOML dynamic key decode), and validate each key against the *launched* plugin's Describe-reported vocabulary as a *post-launch* config validation pass (this requires reordering `Load`/`Validate` relative to `pluginhost.Discover`, since the vocabulary is only known after Describe — see Common Pitfalls).
- **Manual `UnmarshalTOML`**: skip generic-map decode and implement `UnmarshalTOML` directly, deferring vocabulary-checking the same way.

Either way, this is a config-loader/pluginhost sequencing decision the plan must make explicit (see Pitfall 1).

**Installation:** none — no new dependency.

**Version verification:** `github.com/pelletier/go-toml/v2 v2.4.3` confirmed present in `go.mod` this session `[VERIFIED: go.mod:18]`.

## Architecture Patterns

### System Architecture Diagram

```
                 config.toml (source of truth for instance identity — D-08)
                       │
                       ▼
        ┌─────────────────────────────┐
        │ kernel/config.Load/Validate  │  1. decode [sources.<id>] map
        │  - instance id = map key     │  2. decode [webspaces.<ws>.match.<id>]
        │  - display_name uniqueness   │     dynamic-key blocks (per D-05 vocab)
        │  - unknown match field ->    │  3. cross-validate against Describe
        │    loud startup error        │     vocabulary (needs plugin launch —
        └───────────┬──────────────────┘     see Pitfall 1 ordering)
                     │ launches N subprocesses (same binary, different <id>)
                     ▼
        ┌─────────────────────────────┐
        │ kernel/pluginhost.Discover   │  Describe() returns:
        │  - Plugin.instanceID = <id>  │    - source_type (plugin kind, NOT identity)
        │    (config key, D-08)        │    - match_vocabulary []string (NEW,
        │  - Plugin.sourceType (kind)  │      D-05 — e.g. ["folders"] for IMAP)
        │    from Describe (unchanged  │    - display_name, contract_version
        │    T-01-07 rule)             │
        └───────────┬──────────────────┘
                     │ Match(typed fields for THIS instance's block)
                     ▼
        ┌─────────────────────────────┐
        │ kernel/correlate.SyncSource  │  per (webspace, INSTANCE) — not
        │  - resolves fallback vs      │  (webspace, source_type) — iterates
        │    explicit block (D-01/D-02)│  configured instances, applies D-03
        │  - applies sources allowlist │  participation default + allowlist
        │    (D-03)                    │
        └───────────┬──────────────────┘
                     │ item.FromProto(instanceID, protoItem)  <- was sourceType
                     ▼
        ┌─────────────────────────────┐
        │ kernel/index (SQLite)        │  items.source_type column now HOLDS
        │  - id = "{instance}:{srcid}" │  the instance id (D-07: schema drop +
        │  - sync_runs keyed by        │  re-sync, no in-place migration)
        │    instance, not plugin type │
        └───────────┬──────────────────┘
                     │ StreamItems / GetItem / LatestSyncRunPerSource
                     ▼
        ┌─────────────────────────────┐
        │ kernel/httpapi                │  /api/sources, /api/items, /api/*/
        │  - source identity in every   │    stream all report the INSTANCE id
        │    JSON response = instance   │    (KERN-06's "never the bare plugin
        │  - agent.go grant filter now  │    type" requirement)
        │    keys on instance, not      │  - item.go: rendition wrap/sanitize/
        │    source_type (D-10 falls    │    theme moves here (D-11)
        │    out of map shape)          │
        └───────────┬──────────────────┘
                     │
                     ▼
              web/src (SPA) — reads instance id/display_name for
              filter chips, health chips (Phase 6 does the real
              redesign; this phase keeps it functionally correct)
```

### Recommended Task Ordering (not file structure — this is a rewire, not new files)

1. **Proto + SDK first.** `proto/topos/v1/plugin.proto`: replace `MatchRequest{repeated string keywords}` with a typed match-fields message (e.g. `map<string, StringList> match_fields` or per-plugin oneof — see Pitfall 2 for the tradeoff), add a `repeated string match_vocabulary` (or similar) to `DescribeResponse`. Regenerate `sdk/gen/topos/v1`. Update `sdk/contract_test.go`'s allowlist only if RPC *names* change (they don't — only message shapes change, so the RPC allowlist test should stay green untouched, which is worth confirming early as a cheap sanity check).
2. **kernel/config**: `Source` gains no new required field for identity (the map key already *is* the id) but the config loader needs (a) `display_name` uniqueness validation (D-09), (b) `Webspace` restructured to carry `Match map[string]MatchBlock` + optional `Keywords []string` fallback + optional `Sources []string` allowlist (D-01/D-03), (c) vocabulary cross-validation against launched plugins (Pitfall 1).
3. **kernel/pluginhost**: `Plugin.name` (already the config key) becomes the thing every downstream caller uses for identity; `Plugin.SourceType()` stays but its *meaning* narrows to "plugin kind, for vocabulary/contract-version purposes only." Multiple instances of the same plugin type now legitimately launch the same binary path N times (`launch()` already takes `name` distinctly per config entry — the loop in `Discover` already iterates `sources map[string]config.Source`, so multi-instance-same-binary launch is **already supported by the existing loop shape**; nothing structural needs to change there — confirmed `[VERIFIED: kernel/pluginhost/host.go:93-106]`, `for name, src := range sources { p, err := launch(ctx, pluginsDir, name, src, logger) ... }`).
4. **kernel/correlate + kernel/syncer + kernel/index**: rename the identity carried through `SyncSource`, `ReplaceWebspaceSourceItems`, `StartSyncRun`/`FinishSyncRun`, `LatestSyncRunPerSource`, `SyncingSourceTypes` from `source_type` semantics to instance-id semantics. Concretely, `WebspaceResult.SourceType`, `SyncRun.SourceType`, the `sync_runs.source_type` column, and `items.source_type` column all become **instance-id-valued** — consider renaming the Go field/column to `source` or `instance` in this phase to stop the two concepts (plugin kind vs. identity) from continuing to share one misleading name (see Pitfall 3 — this is a judgment call the plan should make explicitly, not silently keep the old field name with new semantics).
5. **kernel/httpapi**: `grantedSourceTypes`/`SourceTypesByName` (currently `map[configName]sourceType`) become identity-keyed the same way — actually simpler post-rewire, since `it.SourceType` (index) already *is* the instance id by step 4, so `grantedSourceTypes` can key directly on instance id without the extra `byName` indirection it needs today (today it exists specifically to translate config name → source_type; once index rows carry instance id directly, `AgentReadGrantedNames()`'s config-name keys already match).
6. **Plugins**: each plugin's `Match` signature changes from `(keywords []string)` to typed fields; each plugin's `Describe` adds its vocabulary. **Signal's `conversations` field must preserve D-05/D-06 exactly** (group name only; 1:1 matches only nickname/system-contact name, never profile name, never Note-to-Self) — this logic in `plugins/signal/match.go` doesn't change, only its *input* (a `conversations []string` field instead of the shared `keywords`).
7. **Rendition kernel-side move (D-11)**: last, since it's the most independently testable slice — move `themeStyle`/`WrapDocument` out of `plugins/proton/body.go`, `plugins/signal/render.go`, `plugins/silverbullet/render.go` into `kernel/httpapi/item.go`, with content-shape-scoped bluemonday policies. Relocate (not lose) the scrollbar assertions from 260805-j98.
8. **Docs + config.example.toml + mock plugin**: rewrite last, once the shape is locked, so they document what shipped rather than what was planned.
9. **Migration**: hand-migrate the user's real `~/.config/topos/config.toml` (D-06 — no auto-migration code) and confirm the kernel rejects the old shape loudly (D-06).

### Pattern: Instance identity vs. plugin kind — two different trust sources, both legitimate

**What:** `Describe`'s `source_type` remains the kernel's only trusted source of *what plugin binary this is* (T-01-07 unchanged — a plugin still cannot claim to be a type it isn't). The config map key becomes the kernel's only trusted source of *which configured instance this is* (D-08, new this phase). These answer different questions and must not be merged into one field with one meaning.
**When to use:** Anywhere the kernel currently reads `SourceType()`/`source_type` to mean "identity" (index rows, sync_runs, agent grants, HTTP `source_type` JSON field, singleflight/coordinator keys) — swap to instance id. Anywhere it currently reads it to mean "plugin kind" (vocabulary validation, contract_version compatibility checks, possibly future UI icon-by-kind) — keep `source_type`.
**Example:** `[VERIFIED: kernel/pluginhost/host.go:41-63]` — `Plugin.name` (`"name string // config key under [sources.<name>]"`) and `Plugin.sourceType` (`"sourceType string // learned via Describe, not trusted from the filename"`) already exist as two separate fields on the same struct today; the rewire is about which one downstream callers key on, not about inventing a new field.

### Pattern: Sources JSON envelope already has a `name` field to build on

**What:** `GET /api/sources` already returns a `sourceStatus{Name, SourceType, DisplayName, ...}` struct where `Name` is the config key `[VERIFIED: kernel/httpapi/sources.go:42-51]`. This is already instance-shaped. The gap is that `/api/items`, `/api/webspaces/{ws}/stream`, and the agent-namespace mirrors only expose `source_type` on each *item* `[VERIFIED: kernel/httpapi/stream.go:27]` (`SourceType string \`json:"source_type"\`` on `streamItem`), not the instance id — so item-level source identity in the stream/detail views is the gap KERN-06's success criterion #3 targets, not the sources list.
**When to use:** When updating `docs/api.md`'s stream/item JSON examples and `web/src/lib/api.ts`'s `StreamItem` interface — the field is likely renamed or a new field added (`source` or `instance`) alongside/instead of `source_type` on every item.

### Anti-Patterns to Avoid

- **Silently redefining `source_type`'s meaning without a name change:** the field/column named `source_type` currently means "plugin kind." If the plan keeps that literal name but changes its *stored value* to be the instance id, every future contributor (and this project's own STATE.md history, which already narrates `source_type` as the plugin kind in a dozen places) will misread it. Either rename the field (recommended) or add an explicit, prominent doc comment at every touch point stating the new semantics.
- **Trusting the plugin's own filename or launch order for the vocabulary check instead of Describe:** D-05 requires the *declared* vocabulary from `Describe`, not a kernel-side table keyed by `source_type` string literals (`"proton" → ["folders"]`) — that would defeat the stated ecosystem-readiness goal (an external plugin brings its own vocabulary; the kernel must not need a table entry for it).
- **Migrating the index in place:** D-07 explicitly says drop and re-sync. Don't write a migration script for `items`/`sync_runs` — that's wasted engineering effort against a locked decision.
- **Reusing `FetchResponse.data` for both raw plugin fragment and kernel-wrapped document during a transition period:** D-11 is a clean contract break, not a dual-mode fallback — plugins stop emitting `<!doctype>`/`<head>`/theme CSS in the same PR that makes the kernel start wrapping. A halfway state (kernel wraps AND plugin still emits a full doc) would double-wrap and is worth flagging as a sequencing risk in the plan (contract version bump + all four in-repo plugins updated atomically).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Dynamic-key TOML decode for per-plugin match vocabulary | A custom TOML sub-parser or reflection-based table walker | `go-toml/v2`'s existing `map[string]interface{}` / `UnmarshalTOML` dynamic-key support | Already a dependency, already handles this exact "table shape depends on a sibling key's value" pattern (`[CITED: go-toml README, DynamicConfig example]`). |
| HTML sanitization policy per content shape (email/chat/markdown) | A new sanitizer from scratch | Reuse the three plugins' existing bluemonday policies as the kernel's three content-shape profiles | `plugins/proton/body.go`, `plugins/signal/render.go`, `plugins/silverbullet/render.go` already encode exactly the right per-shape allowlists (email narrow-style-allowlist, chat no-styles, markdown) — this is a relocation, not a redesign (per CONTEXT.md's own framing). |
| Proto message versioning / compatibility signaling | An ad hoc version string comparison scheme | `DescribeResponse.contract_version` (already exists) + `ProtocolVersion`/`MagicCookieValue` handshake (already exists, `sdk.Handshake`) | This machinery is already in place and already documented in `docs/plugin-contract.md`'s "Handshake and the plugin-map key" section — the phase should extend it (bump `contract_version` string, decide if `ProtocolVersion` needs a bump — see Pitfall 5), not invent a parallel versioning concept. |

**Key insight:** Every mechanism this phase needs (dynamic config decode, per-shape sanitization policy, contract versioning) already has a working, in-repo precedent from a prior phase. The engineering risk here is disciplined rewiring and sequencing, not missing tooling.

## Runtime State Inventory

> This phase is a rename/refactor of the kernel's own identity model plus a config-shape break — the Runtime State Inventory categories below apply.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | Local SQLite index (`~/.local/share/topos/index.db` — default path `[VERIFIED: kernel/config/types.go:142]` `DefaultIndexPath = "~/.local/share/topos/index.db"`): `items` and `sync_runs` tables key on `source_type`, which currently holds the plugin kind (e.g. `"proton"`), not an instance id. | **Data migration is explicitly NOT required** — D-07 locks "index is dropped and fully re-synced on upgrade." Confirm the plan includes an explicit index-file deletion/rebuild step (or a schema-version bump that triggers `Store.Open` to detect and rebuild) rather than leaving old rows to silently coexist with new-shape rows under the same `source_type` column name. |
| Live service config | None found beyond this repo's own config file — this phase touches no external service (paperless-ngx, SilverBullet, Proton Bridge, Signal Desktop) configuration; those systems' own tag/folder/conversation names are read live via `Match`, never cached outside the index covered above. | None. |
| OS-registered state | None — no daemon registration, no scheduled task, no systemd unit is affected by an identity rename. | None. |
| Secrets/env vars | `WEBSPACES_SOURCE_CONFIG` env var passed per-launched-subprocess (`[VERIFIED: kernel/pluginhost/host.go:114-131]`) is keyed by nothing external — it's built fresh per `launch()` call from that instance's own `config.Source` struct, so multiple instances of the same plugin type already get independently-correct env (different `base_url`/`token` per instance) with **zero code change needed** here; confirmed the existing loop already passes each instance's own `src` separately. | None — already correct for multi-instance. |
| Build artifacts | None — plugin binaries are named by `[sources.<name>].plugin` (a shared binary filename like `topos-plugin-proton`), not by instance; two instances of the same plugin type already correctly resolve to the same on-disk binary path launched twice `[VERIFIED: kernel/pluginhost/host.go:108-112]` (`launch(ctx, pluginsDir, name, src, logger)` uses `src.Plugin` for `binPath`, `name` only for the config/log identity). | None. |
| User's real config file | `~/.config/topos/config.toml` (the user's own, not in this repo) has the OLD shape (`[webspaces.<name>] keywords = [...]`) per D-06's explicit callout: "The user's real `~/.config/topos/config.toml` is hand-migrated as part of this phase's delivery." | **Manual migration task required** — this is delivery work, not optional. The plan must include a task to hand-edit the user's actual config file to the new per-instance match-block shape, and verify the kernel starts and syncs against it (this is effectively the phase's own UAT/live checkpoint). |

**Nothing found in category "Live service config" or "OS-registered state"** — verified by reading `kernel/pluginhost/host.go`'s full launch/env logic and confirming no other process-registration touchpoint exists in the kernel for source plugins.

## Common Pitfalls

### Pitfall 1: Vocabulary validation needs a plugin launch — config load and plugin discovery are not currently ordered for that

**What goes wrong:** D-05 requires the kernel to validate a configured match block's field names (`folders`, `tags`, etc.) against the vocabulary the *specific plugin* declares in `Describe` — but `Describe` is only callable after `pluginhost.Discover` has launched that plugin subprocess, which today happens *after* `config.Load`/`Validate` succeeds (`cmd/topos/main.go`'s construction order: load config → validate → discover plugins). A naive implementation might try to validate vocabulary inside `config.Validate`, before any plugin has been launched, and fail because there's nothing to validate against yet.
**Why it happens:** The current `config.Validate` is a pure, plugin-independent function (`missing []string` is its only external input) — it was designed to fail fast before spending the cost of launching subprocesses. D-05's requirement inherently couples config validation to plugin identity, which the current phase boundary doesn't have.
**How to avoid:** Explicitly design a **two-phase validate**: `config.Validate` keeps its current plugin-independent checks (webspace keyword/display_name-shape/duration checks) plus adds *structural* checks that don't need a plugin (e.g. "display_name is unique," "an explicit match block replaces the fallback," "an unknown `sources` allowlist name doesn't exist" — this one might also need to wait). A second pass, after `pluginhost.Discover` returns each plugin's `Describe`-reported vocabulary, does the per-field vocabulary check and fails startup loudly (per D-05: "naming the field and the plugin") before any sync begins. The plan should name this second pass explicitly (e.g. `pluginhost.ValidateMatchVocabulary` or similar) rather than leaving it implicit.
**Warning signs during planning:** If a plan task tries to do vocabulary validation entirely inside `kernel/config` without any dependency on `pluginhost`, it will either be unable to validate at all, or will end up hard-coding a `source_type → vocabulary` table in `kernel/config` — which is exactly the kernel-side table D-05 says NOT to build.

### Pitfall 2: Typed `MatchRequest` proto shape — a single flat `map<string, repeated string>` vs. per-type message

**What goes wrong:** Two plausible proto shapes for the new `MatchRequest` have different tradeoffs the plan must pick between explicitly (CONTEXT.md leaves "Proto mechanics of the contract change" to Claude's discretion, so this is a live decision, not settled):
  - **(a) `map<string, StringList> match_fields`** (a generic string-list map, field names not fixed in the proto) — maximally ecosystem-flexible (an external plugin's field names never require a proto change), mirrors the TOML shape 1:1, but loses proto-level type safety/documentation of what fields exist for the four in-repo plugins.
  - **(b) A message per plugin type or a single message with one repeated-string field per known plugin (`folders`, `tags`, `conversations`, `pages`)** — self-documenting in the .proto, but requires a proto field addition for every future plugin type, working against the "ecosystem-ready, no kernel table to update" framing of D-05.
**Why it happens:** The proto is the published, versioned contract (`docs/plugin-contract.md`'s whole premise) — whichever shape is picked here is what third-party plugin authors read cold, so it's worth getting deliberately, not accidentally.
**How to avoid:** Given D-05's explicit ecosystem framing ("external plugins bring their own vocabulary, no kernel table to update"), option (a) — a generic string-list map — is the shape consistent with the rest of this phase's stated philosophy; each plugin interprets only the keys it declared in its own `Describe` vocabulary and should treat any other key as absent/ignorable (not an error at the plugin level — validation already happened kernel-side per Pitfall 1). Document this explicitly as the chosen tradeoff rather than leaving it implicit in generated code.
**Warning signs:** A plan that adds one proto field per plugin type (`repeated string folders = 2; repeated string tags = 3; repeated string conversations = 4;` …) is quietly reintroducing the "kernel needs to know every plugin type" coupling D-05 says to avoid, and will need a proto change for every future source plugin — including Phase 8's WhatsApp.

### Pitfall 3: `source_type` field/column rename is a sweep across 8+ files — an incomplete sweep silently reintroduces the bug KERN-06 exists to fix

**What goes wrong:** Confirmed via this session's reads, `source_type`/`SourceType` currently appears as an identity-bearing symbol in (non-test) code in: `kernel/syncer/coordinator.go`, `kernel/index/schema.go`, `kernel/index/store.go`, `kernel/correlate/correlate.go`, `kernel/httpapi/agent.go`, `kernel/httpapi/item.go`, `kernel/httpapi/sources.go`, `kernel/httpapi/stream.go`, `kernel/pluginhost/host.go`, `kernel/item/item.go` `[VERIFIED: grep -rln "source_type\|SourceType" kernel --include=*.go, excluding _test.go, 10 files]`. If even one of these is missed (e.g. `agent.go`'s `grantedSourceTypes` keeps filtering on plugin kind while `item.go`'s composite id switches to instance id), the result is a partially-migrated kernel where two instances of the same plugin type get correctly separate index rows but incorrectly shared/leaked agent grants (a security-relevant regression, not just a display bug) — or vice versa.
**Why it happens:** This is exactly the kind of change that "compiles fine" at every intermediate state (Go's type system doesn't distinguish "a string that means plugin kind" from "a string that means instance id") so a partial sweep produces no compiler error, only a subtly wrong runtime behavior that only manifests once a user actually configures two instances of one plugin type.
**How to avoid:** Plan this as one deliberate, single-PR-scoped sweep (or a small number of tightly sequenced plans within the phase) with an explicit file checklist (the 10 files above, plus the 4 test files that assert against the old shape: `kernel/syncer/coordinator_test.go`, `kernel/syncer/scheduler_test.go`, `kernel/index/store_test.go`, `kernel/correlate/correlate_test.go`, `kernel/httpapi/*_test.go` (5 files), plus `kernel/httpapi/contract_test.go`'s `idPattern`/`requiredProvenanceKeys`). Consider a mechanical grep-based checklist as a plan verification step: "after this task, `grep -rn 'SourceType()' kernel/ --include=*.go` returns only calls whose result is used for vocabulary/contract-version purposes, never as a map/index key."
**Warning signs:** Any code review turning up `sourceType` and `instanceID`/`name` both flowing into the same function signature (e.g. `Fetch(ctx, sourceType, sourceID string, ...)` in `pluginhost.Host.Fetch` and `httpapi.Fetcher` — `[VERIFIED: kernel/pluginhost/host.go:270]`, `[VERIFIED: kernel/httpapi/item.go:26]`) without a clear doc comment on which one it actually is post-rewire is a signal the sweep is incomplete. Note `pluginhost.Host.bySourceType` (`kernel/pluginhost/host.go:307-314`) resolves a plugin by **plugin kind**, which is a real, ongoing need (Fetch must reach the one live process for that instance) — this specific lookup likely needs to become "resolve by instance id" too, since `Fetch`'s `sourceType` argument is what the index row's (now instance-id-valued) column supplies.

### Pitfall 4: The `sync_runs` table's `GROUP BY source_type` queries assume one row-family per key — multiplying instances multiplies rows, which is correct but changes cardinality assumptions

**What goes wrong:** `LatestSyncRunPerSource` and `SyncingSourceTypes` both use `WHERE id IN (SELECT MAX(id) FROM sync_runs GROUP BY source_type)` `[VERIFIED: kernel/index/store.go:587, 628]`. Once `source_type` holds instance ids, this SQL is *already correct as written* — no query logic change needed, since `GROUP BY` on the new instance-id-valued column does exactly the right per-instance grouping. The pitfall is assuming a query change is needed when it isn't (wasted plan effort) — OR, if the column is renamed (Pitfall 3's recommendation), forgetting to update these two literal SQL strings' column references, which would be a silent compile-time-invisible break (raw SQL strings, not struct field references).
**Why it happens:** Raw SQL string literals don't get renamed by `gofmt`/an IDE rename refactor — a Go-level field rename (`SyncRun.SourceType` → `SyncRun.Source`) does not touch `store.go`'s embedded SQL text unless done by hand.
**How to avoid:** If the column/field is renamed, grep the literal SQL strings in `kernel/index/schema.go` and `kernel/index/store.go` separately from the Go struct field rename — these are two different rename operations that happen to share a name today.

### Pitfall 5: Handshake `ProtocolVersion` vs. `contract_version` — don't conflate an additive change with a breaking one

**What goes wrong:** `docs/plugin-contract.md` is explicit that `ProtocolVersion` (the `go-plugin` handshake constant, currently `1`) is bumped only for a breaking *wire-protocol* change, while `DescribeResponse.contract_version` (currently `"topos.v1"`) is the additive-compatibility signal `[VERIFIED: docs/plugin-contract.md:132-136, 231-235]`. This phase changes `MatchRequest`'s shape and `FetchResponse`'s meaning (D-11) — genuinely breaking changes to what a plugin must send/receive, not additive ones. A plan that only bumps `contract_version`'s string value (treating this as forward-compatible) while leaving `ProtocolVersion` at `1` would misrepresent the compatibility signal the contract's own docs define — an old-contract plugin binary would still handshake successfully (same magic cookie/protocol version) but then fail confusingly at the first `Match` call with a shape mismatch, rather than failing fast at handshake.
**How to avoid:** The plan should explicitly decide whether `ProtocolVersion` bumps to `2` (clean failure at handshake for a stale plugin binary) given this is explicitly framed by CONTEXT.md as a breaking, "no dual-shape support" change (D-06's philosophy extends naturally to the plugin binaries themselves, not just the TOML config) — this is a "Claude's Discretion" item per CONTEXT.md ("contract version bump strategy") but the pitfall is *not deciding it deliberately* and leaving a mismatched signal.

## Code Examples

Verified patterns from this repo's own source (no external contract to fetch — this is an in-repo rewire):

### Existing multi-launch loop already supports N instances of one binary — no structural change needed here
```go
// Source: kernel/pluginhost/host.go:93-106 (VERIFIED, read this session)
func Discover(ctx context.Context, pluginsDir string, sources map[string]config.Source, logger hclog.Logger) (*Host, error) {
	h := &Host{}

	for name, src := range sources {
		p, err := launch(ctx, pluginsDir, name, src, logger)
		if err != nil {
			h.Shutdown()
			return nil, fmt.Errorf("pluginhost: launch source %q: %w", name, err)
		}
		h.plugins = append(h.plugins, p)
	}

	return h, nil
}
```
Two config entries `[sources.home-email]` and `[sources.work-email]`, both with `plugin = "topos-plugin-proton"`, already launch two independent subprocesses today with independently-correct `WEBSPACES_SOURCE_CONFIG` env — confirmed by reading the loop and `launch()`'s env-marshal call (`kernel/pluginhost/host.go:114-131`). The gap this phase closes is entirely downstream of this loop (index/sync/API identity), not in launch/discovery itself.

### Existing per-plugin exact-match pattern to preserve when moving to typed fields
```go
// Source: plugins/mock/plugin.go:200-213 (VERIFIED, read this session) — the
// exact-case-insensitive rule (Phase 1 D-04's carryover, unchanged by D-04
// of this phase) every plugin's typed-field matcher must keep:
func labelsMatchAnyKeyword(labels, keywords []string) bool {
	for _, label := range labels {
		for _, kw := range keywords {
			if strings.EqualFold(label, kw) {
				return true
			}
		}
	}
	return false
}
```
Every one of the four real plugins (`plugins/proton/plugin.go:278-285` `matchesAnyKeyword`, `plugins/signal/match.go:49-62` `matchesAnyKeyword`, `plugins/silverbullet` via `MatchesKeyword`, `plugins/paperless` via `ResolveTagIDs`) already implements this same exact/case-insensitive rule independently — when each plugin's `Match` moves from one shared `keywords []string` to its own typed field(s), this comparison function itself does not need to change, only what it's called with.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| One shared `Webspace.Keywords []string` matched against every plugin's native categorization (Phase 1 D-02/D-03) | Per-instance typed match blocks (`folders`, `tags`, `conversations`, `pages`) with an optional webspace-level keyword fallback | This phase (2026-08 restructure) | Every plugin's `Match` RPC signature changes; every webspace config block's shape changes; deliberately, not additively (D-06 clean break). |
| `source_type` (Describe-learned plugin kind) doubles as the kernel-wide identity key everywhere | Config map key (instance id) is the identity; `source_type` narrows to "plugin kind" only | This phase | Touches ~10 kernel Go files, 3 docs, `config.example.toml`, and every plugin's `Describe`; the single largest engineering surface of this phase. |
| Plugins own rendition presentation (`themeStyle` + `WrapDocument`, duplicated in 3 plugins) | Kernel (`kernel/httpapi/item.go`) owns wrap/sanitize/theme; plugins return content + shape metadata | This phase (folded todo, D-11) | `FetchResponse` shape changes (plugins stop emitting full HTML documents); kernel gains per-content-shape sanitizer policies. |

**Deprecated/outdated:**
- The single shared per-webspace keyword list (Phase 1 D-02/D-03): explicitly and deliberately reversed this phase per CONTEXT.md's own framing — this is not a mistake being corrected silently, it's a documented, accepted-cost reversal.
- Per-plugin hardcoded `themeStyle` Go constants (`plugins/proton/body.go:196`, `plugins/silverbullet/render.go:58`, and the signal equivalent): superseded by the kernel-owned stylesheet; the todo file `.planning/todos/pending/2026-08-05-centralize-rendition-theming-in-kernel.md` should move to completed when this phase ships (per CONTEXT.md).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `go-toml/v2`'s dynamic-key decode (`map[string]interface{}` / `UnmarshalTOML`) is the right mechanism for D-05's per-plugin match vocabulary, rather than requiring a new TOML library or a hand-rolled two-pass parse | Standard Stack | Low — this is a well-established go-toml/v2 capability per its own README `[CITED, not independently exercised against this repo's exact nested shape this session]`; worst case, the plan spikes a small decode prototype early in Task 1 to confirm the exact nesting (`[webspaces.<ws>.match.<instance>]`) decodes cleanly before committing to the approach. |
| A2 | A generic `map<string, StringList>` proto shape (Pitfall 2, option a) is the right choice over a per-plugin-type message, based on inferring intent from D-05's stated ecosystem-readiness philosophy | Pitfall 2 | Medium — this is explicitly left to "Claude's Discretion" in CONTEXT.md, so it is a real open design decision, not a verified fact; the plan or a discuss-phase follow-up should confirm this reasoning with the user before locking the proto shape, since it is flagged D-05-adjacent as "costly to reverse" (published contract). |
| A3 | `ProtocolVersion` should bump from `1` to `2` given the breaking nature of this contract change | Pitfall 5 | Low-Medium — reversible in the sense that a version bump is cheap to add later, but *not* bumping it when the shape is genuinely breaking risks a confusing runtime failure mode (stale plugin binary handshakes successfully, fails at first Match) rather than a clean fail-fast; recommend the plan decide this explicitly rather than defaulting to "leave it at 1." |

## Open Questions

1. **Exact TOML nesting for `sources` allowlist interaction with fallback (D-01/D-02/D-03 combined)**
   - What we know: an explicit match block replaces the fallback for that instance (D-02); all configured instances participate by default unless a `sources = [...]` allowlist restricts (D-03); the fallback is "native-name matching" using the webspace-level `keywords` list (D-01).
   - What's unclear: whether an instance can be *allowlisted out* (via `sources`) while still having its own explicit match block defined in the same webspace (dead config), and whether the config loader should warn/error on that specific combination, or silently ignore the unused block.
   - Recommendation: the plan should decide (or defer to Claude's Discretion, since it's not explicitly locked in CONTEXT.md) whether this is a load-time error (loud, consistent with D-05's "fail loudly" philosophy elsewhere in this phase) or silently permitted (an instance's match block staged for future use without yet enabling participation).

2. **Whether `source_type` (or its replacement column name) needs to stay in the item's JSON envelope at all, or whether the new instance identity fully replaces it in the public API**
   - What we know: `docs/api.md`'s provenance-keys table documents `source_type` as one of six required provenance keys, and `kernel/httpapi/contract_test.go` mechanically pins that key's presence (AGENT-02) `[VERIFIED: kernel/httpapi/contract_test.go:27-29]`.
   - What's unclear: whether Phase 5 keeps `source_type` in provenance (still meaning plugin kind, informational) AND adds a new top-level instance-id field to `streamItem`/`Item`, or renames/repurposes the existing field.
   - Recommendation: keep `source_type` (plugin kind) in `provenance` as-is (agent consumers may reasonably want to know "this came from an IMAP-shaped source" regardless of which account), and add a distinct instance-identifying field at the top level of the item/stream response (e.g. `source` or `source_instance`) — this keeps the AGENT-02 contract test's *existing* assertions valid (minimizing test churn) while adding what KERN-06 requires. The plan should confirm this against the discuss-phase record or ask the user if ambiguous, since it affects the published HTTP contract.

## Security Domain

### Applicable ASVS Categories (ASVS Level 1, per `.planning/config.json`)

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | No auth added or changed this phase — API remains unauthenticated, loopback-only (unchanged project posture per `docs/api.md`'s "Loopback-only default, no auth"). |
| V3 Session Management | No | No session concept in this project. |
| V4 Access Control | **Yes** | The per-instance agent grant (D-10, AGENT-01 carryover) is an access-control mechanism — `kernel/httpapi/agent.go`'s `grantedSourceTypes`/`filterRunsByGrant` must be re-verified to key correctly on instance id post-rewire (Pitfall 3's security-relevant case: an incomplete sweep could leak one instance's items under another instance's grant, or vice versa, if both share the same underlying plugin kind). Standard control: continue the existing default-deny, explicit-grant-only pattern already in place — no new control needed, but existing tests (`kernel/httpapi/agent_test.go`) must be extended to cover the two-instances-same-type case specifically, since today's tests can't exercise it (the shape didn't exist before this phase). |
| V5 Input Validation | **Yes** | The new per-plugin match vocabulary validation (D-05) is itself an input-validation control: config-supplied match field names are validated against a plugin-declared allowlist, failing loudly and by name on an unknown field — this is exactly the ASVS V5 "positive/allowlist validation" pattern, already the right shape; no library needed (see Standard Stack). |
| V6 Cryptography | No | No cryptographic material touched by this phase (Signal's SQLCipher key handling, Phase 4, is untouched). |

### Known Threat Patterns for this stack (this phase's scope)

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Cross-instance data/grant leakage from an incomplete `source_type`→instance-id rewire (Pitfall 3) | Information Disclosure / Elevation of Privilege (an agent reading an ungranted instance's items via a shared plugin-kind key) | Complete, mechanically-checked sweep (grep-based verification step per Pitfall 3) plus a new test case: two configured instances of the same plugin type, one granted `agent.read`, one not — assert the ungranted instance's items never appear in `/agent/v1/*` responses. This is the single most important new test this phase should add, beyond what CONTEXT.md's existing "standing contract tests" cover. |
| Kernel-side sanitization gap during the D-11 rendition move (a moment where neither the old plugin-side sanitizer nor the new kernel-side one is fully wired) | Tampering (XSS via an under-sanitized rendition served from the kernel's own origin) | Land the kernel-side sanitizer and its content-shape policy profiles *before* removing any plugin's existing bluemonday call — i.e. the plan should not have an intermediate state where a plugin has stopped sanitizing but the kernel hasn't started; sequence each plugin's cutover as its own atomic task (plugin stops emitting a full doc + kernel gains that content-shape's policy, in the same task). |
| Unknown match field silently accepted instead of failing loudly (D-05 regression) | Tampering / Repudiation (a typo'd match field silently matches nothing, and the operator has no signal their config is wrong) | Config load fails hard, naming the field and the plugin (D-05's own requirement) — this is already the intended design, called out here only to confirm it's an explicit test case (`kernel/config/config_test.go` gets a new negative test), not an incidental side effect of validation code. |

## Sources

### Primary (HIGH confidence — in-repo, read this session)
- `kernel/config/types.go`, `kernel/config/config.go` — current `Source`/`Webspace` shape, `Validate` logic, env-var expansion.
- `proto/topos/v1/plugin.proto` — current wire contract (`DescribeResponse`, `MatchRequest`/`MatchResponse`, `Item`, `FetchRequest`/`FetchResponse`, `LinkFidelity`, `ContentVariant`).
- `kernel/pluginhost/host.go` — plugin launch/discovery, `Plugin` struct fields, `Fetch`/`ProbeSources`/`bySourceType`.
- `kernel/index/schema.go`, `kernel/index/store.go` — full SQLite schema and every query keyed on `source_type`.
- `kernel/correlate/correlate.go`, `kernel/syncer/coordinator.go` — sync identity plumbing (`SyncSource`, `Refresh`, `syncOne`, singleflight keying).
- `kernel/httpapi/item.go`, `kernel/httpapi/sources.go`, `kernel/httpapi/agent.go`, `kernel/httpapi/stream.go`, `kernel/httpapi/contract_test.go` — HTTP/agent API surface, grant filtering, the AGENT-02 standing contract test.
- `kernel/item/item.go` — the kernel's normalized `Item` type and `FromProto`/`ID` composite-key construction.
- `plugins/paperless/plugin.go`, `plugins/proton/plugin.go`, `plugins/silverbullet/plugin.go`, `plugins/signal/match.go`, `plugins/mock/plugin.go` — every real plugin's current `Match`/matching-rule implementation.
- `plugins/proton/body.go`, `plugins/signal/render.go`, `plugins/silverbullet/render.go` — the three `themeStyle`/`WrapDocument` implementations D-11 centralizes.
- `docs/plugin-contract.md`, `docs/api.md`, `config.example.toml` — the three published-contract docs this phase rewrites.
- `sdk/contract_test.go`, `internal/audit/outbound_hosts_test.go` — the RPC allowlist and egress/AST standing tests referenced in CONTEXT.md.
- `.planning/todos/pending/2026-08-05-centralize-rendition-theming-in-kernel.md` — the folded rendition-theming todo's full problem statement.
- `go.mod`, `go.work` — dependency/module layout confirmation (`go-toml/v2 v2.4.3`, five plugin modules plus `sdk`).
- `web/src/lib/api.ts`, `web/src/lib/format.ts`, `web/src/lib/components/SourceFilterChips.svelte` — current frontend `source_type` usage.

### Secondary (MEDIUM confidence)
- `github.com/pelletier/go-toml` README (v2 branch, GitHub) — dynamic-key/`map[string]interface{}` decode and `DynamicConfig`/`UnmarshalTOML` pattern, via WebSearch this session, cross-referenced against the pinned version in `go.mod` `[CITED: github.com/pelletier/go-toml/blob/v2/README.md]`.

### Tertiary (LOW confidence)
- None — this phase's domain is entirely in-repo; no unverified external claims were needed.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependency; existing `go-toml/v2` capability confirmed present via `go.mod` and cross-checked against official README.
- Architecture: HIGH — every touchpoint enumerated above was located via direct `Read`/`grep` of the actual source this session, not inferred from documentation alone.
- Pitfalls: HIGH — each pitfall is grounded in a specific, cited line range of existing code, not speculative.

**Research date:** 2026-08-06
**Valid until:** No external time-decay risk (in-repo architecture research); re-verify only if the codebase changes materially before planning begins (e.g. another phase lands first and touches these same files).
