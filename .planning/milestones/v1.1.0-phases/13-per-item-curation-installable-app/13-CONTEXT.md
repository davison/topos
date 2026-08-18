# Phase 13: Per-Item Curation & Installable App - Context

**Gathered:** 2026-08-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Three independent capability strands:

1. **Per-item exclusion (KERN-09/KERN-10)** — the final tier of the filter hierarchy and the kernel's first user-owned data beyond config. A user excludes individual items from a webspace; exclusions survive re-sync, kernel restart, and index rebuild, and always outrank automatic match rules. An excluded-items view shows exactly what was removed and lets the user un-exclude it. Include means un-exclude only — pulling never-matched items in via a Browse RPC is explicitly out of scope (KERN-11, future).
2. **PWA installability (UI-13/UI-14)** — topos installs from the browser on desktop (manifest, ServiceWorker, icons) and launches as a standalone app window against the local kernel; after a kernel upgrade the installed app never serves a stale UI. The mobile/LAN secure-context limitation is documented with recommended user-provided HTTPS workarounds — no kernel HTTPS mode (UI-15, future), no offline API caching.
3. **Trust-tier hardening (folded todo, Phase 11 debt)** — trusted-tier status derives from build provenance (link-time-embedded manifest of first-party plugin hashes), not directory location alone, closing the config-edit / file-drop / D-11-shadowing bypass paths of the Phase 11 consent boundary.

Requirements: KERN-09, KERN-10, UI-13, UI-14 + the folded 2026-08-13 trust-tier todo. Depends on nothing (Phases 11–12 shipped the machinery the hardening strand touches).

</domain>

<decisions>
## Implementation Decisions

### Exclude interaction
- **D-01:** The stream gets desktop-standard **multi-select** — ctrl-click toggles, shift-click extends a range — while plain click keeps opening the detail pane. A **floating action bar** appears while the selection is non-empty ("N selected — Exclude · Clear"); Esc or Clear empties it. The detail pane additionally carries a single-item exclude button for the currently open item.
- **D-02:** Exclusion (single or bulk) is **instant with an undo toast** — no confirm dialogs ever; the excluded-items view is the durable reversal path.
- **D-03:** Exclusion removes the item from **every webspace surface**: stream, in-webspace FTS search results, date-marker ruler, and any counts. One consistent rule — exclusion is the final filter tier applied wherever webspace items are read.
- **D-04:** Chat conversation-day digests exclude **per digest row** (that conversation, that day). "This conversation never belongs here" is match-rules territory, not per-item curation — no conversation-wide exclusion mechanism.

### Excluded-items view
- **D-05:** The view is a **stream filter toggle**: a chip/toggle in the webspace chrome (e.g. "Excluded (7)") flips the stream itself to show the excluded bucket, reusing StreamRow, the detail pane, and the multi-select machinery. No modal, no separate route.
- **D-06:** The toggle appears **only when the webspace's excluded count > 0** (count shown). A webspace with no exclusions looks exactly like today.
- **D-07:** The excluded view orders **chronologically like the stream** (same date markers) — it is literally the same stream showing the other bucket.
- **D-08:** Un-exclude is the **exact mirror of exclude**: multi-select + action bar offering "Include", single-item include button in the detail pane, instant with undo toast.

### Orphaned exclusions
- **D-09:** **Silent auto-prune**: when a **healthy** sync no longer reports an excluded item, the mark is swept with the index row. The excluded view only ever shows items that would otherwise be in the stream. A reappearing item (e.g. restored file) re-enters the stream unexcluded.
- **D-10:** Prune fires **only on a healthy sync that omits the item** — a failed sync (unreachable mount, plugin error) must never count as "item gone", and an **index rebuild must never prune marks** (rebuild wipes item rows before re-sync completes; marks must survive per KERN-09). This is the never-silently-empty precedent applied to marks.
- **D-11:** Rename/move losing an exclusion (new stable ID under Phase 12's remove+add identity model) is **accepted** as the honest consequence — no rename-tracking for marks.

### Trust-tier hardening (folded todo)
- **D-12:** Trusted-tier status derives from a **build-provenance manifest**: a set of first-party plugin identities/hashes embedded into the kernel at link time (kernel and plugins are rebuilt together by `make build`/`make dev`, so dev rebuilds stay false-alarm-free). A trusted-dir binary that doesn't verify against the manifest gets **no trusted-tier treatment regardless of directory** — closing the config-edit, file-drop, and shadowing bypass paths. — **Reversibility:** costly — the manifest becomes the trust authority the UI, docs, and release process describe; retiring it means re-deriving trust from directory location again and re-opening the three bypass paths.
- **D-13:** **Hash-verification failure refuses to load, loudly** (supersedes the demote-and-run idea discussed en route; matches the operator's TODO.md intent): the binary does not launch; the affected source chip shows a named error state (log AND UI) explaining why; the explicit consent + pin flow (external-tier add / re-accept) is the only path to running it. Unverified code never executes. This harmonizes with Phase 11's shipped external-tier behavior, where a pin mismatch already fails the launch per-instance with a named state and a two-click re-pin.
- **D-14:** The **D-11 shadowing event** (same-named trusted-dir binary shadowing a pinned external plugin) is surfaced in the **UI** (health state / advisory on affected sources), not just a kernel log line.
- **D-15:** The **locally-built Signal plugin** (release artifacts exclude it; `make signal` builds it locally, so a release kernel's manifest can never contain its hash) is handled **honestly via consent + pin**: it refuses to load until the user explicitly accepts it through the external-tier consent flow, then runs with a pinned hash and the untrusted badge. **The plugin documentation must clearly explain why** (cgo local build ≠ release manifest). Source-builders running `make build` (kernel + plugins together) are unaffected.
- **D-16:** The **two plugin directories stay** this phase. The manifest is the trust *authority*; directories remain the shipped install conveniences. Collapsing to a single directory waits for the distribution work (PLUG-10), where a managed plugin directory becomes the natural shape.

### Claude's Discretion
- **PWA mechanics end to end**: tooling choice (vite-plugin-pwa / @vite-pwa/sveltekit vs hand-rolled SW), manifest contents, icon set derivation from the existing `web/static/app-icon.png`, ServiceWorker update strategy satisfying the never-stale criterion (the operator deliberately left the update UX — silent reload vs prompt — to Claude), kernel MIME-type/serving adjustments, and what the installed window shows when the kernel isn't running.
- Marks storage mechanics: where the marks live (separate SQLite file vs rebuild-exempt table in the index DB), schema, and the API shape for exclude/include endpoints — constrained only by D-09/D-10 survival semantics and "not config TOML" (Phase 11 D-01 framing: marks are the kernel's first user-owned data beyond config).
- Undo-toast duration/stacking, selection keyboard shortcuts beyond ctrl/shift-click, action-bar copy.
- Manifest generation mechanics (how `make build` embeds the hash set, hash algorithm, manifest format) and the exact name/copy of the refuse-to-load error state.
- Whether the excluded-count toggle lives with the filter chips or elsewhere in the webspace chrome.

### Folded Todos
- **"Plugin trust tier is directory-location, not provenance"** (`.planning/todos/pending/2026-08-13-plugin-trust-tier-is-directory-location-not-provenance.md`, security, major) — Phase 11 shipped a filesystem-location proxy for repo-provenance trust; config edits, file drops, and D-11 shadowing bypass the consent flow. Folded into this phase as the trust-tier hardening strand (D-12..D-16), adopting the todo's own candidate direction (link-time-embedded manifest, unverifiable binaries lose trusted treatment).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Operator's own design notes
- `TODO.md` — §"Plugin Trust System": the operator's written vision for hash-recorded-in-config, load-time verification, refuse-to-load-with-UI-error, and PR-merge trust promotion. D-13 follows its mismatch stance; its distribution parts (publish-plugin GHA, pull-by-URL) are deferred to PLUG-10/11.
- `.planning/todos/pending/2026-08-13-plugin-trust-tier-is-directory-location-not-provenance.md` — the folded todo: three bypass paths, severity framing (consent layer, not privilege boundary), candidate manifest direction, affected files (`kernel/config/types.go:32-66`, `kernel/pluginhost/discover_binaries.go`, `cmd/topos/main.go:85`).

### Plugin trust machinery being hardened
- `.planning/phases/11-external-plugins-the-trust-boundary/11-CONTEXT.md` — Phase 11's locked decisions this phase extends: pin-in-config (D-01/D-02), "binary changed" soft-fail + re-pin (D-03), trusted-tier-unpinned rationale (D-04) that D-12 supersedes with the manifest, tier collision (D-11), env hygiene (D-14).
- `docs/plugin-contract.md` — published contract external authors build against; trust tiers and launch behavior documented here must be republished coherently after the manifest lands (including the Signal local-build explanation, D-15).
- `docs/api.md` — HTTP API surface; new marks endpoints and any trust-state additions land here.

### Curation substrate
- `kernel/index/schema.go` — the index schema and its **rebuild-drops-every-table** discipline (schemaVersion mismatch → drop + re-derive); marks must live outside this blast radius (D-10). FTS5 external-content table + triggers that D-03's search exclusion must respect.
- `.planning/phases/12-filesystem-source/12-CONTEXT.md` — Phase 12's identity decisions (D-01 relative-path IDs, D-02 rename = remove+add) that D-11 accepts the consequences of.
- `.planning/research/SUMMARY.md` — v1.1.0 research: marks pitfalls (sync-replace wipes marks, orphan handling), PWA tooling recommendations and secure-context findings.

### Requirements & conventions
- `.planning/REQUIREMENTS.md` — KERN-09/KERN-10/UI-13/UI-14 definitions; Out of Scope table (no offline caching, no sandboxing); KERN-11/UI-15 future markers.
- `docs/testing.md` — testing map; UI work extends the Playwright e2e suite as definition of done (project convention).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `web/src/lib/components/StreamList.svelte` / `StreamRow.svelte` / `StreamDateMarkers.svelte` — the stream surface that gains multi-select, the action bar, and the excluded-bucket toggle; the excluded view reuses these wholesale (D-05/D-07).
- `web/src/lib/components/DetailPane.svelte` — gains the single-item exclude/include button (D-01/D-08).
- `kernel/index/store.go` — `ReplaceWebspaceSourceItems` (the sync-time write path that must never touch marks) and `rebuildOnSchemaChange` (the rebuild path marks must survive).
- `kernel/httpapi/` — route patterns for the new marks endpoints; `stream.go`'s webspace read path is where mark filtering joins in (D-03 applies it to stream + search alike).
- Phase 11 pin/consent machinery (`kernel/pluginhost/`, re-pin flow, untrusted interstitial) — D-13/D-15's refuse-to-load and consent+pin path reuse this rather than inventing a parallel flow.
- Named health-state vocabulary + `isAdvisoryOnly` precedence chain (`web/src/lib/format.ts`, Phase 12 CR-01 fix pattern) — the "unverified binary" and shadowing states join it; any new health-adjacent surface must consult the same precedence chain.
- `web/static/app-icon.png` + `kernel/webui/embed.go` (go:embed of the adapter-static build, `200.html` fallback) — the PWA manifest/SW/icons ride the existing embed; SW scope and update flow must work when served by the kernel's catch-all handler.
- `web/e2e/` hermetic Playwright harness — new specs for multi-select/exclude/undo, excluded view, refuse-to-load states, and (where drivable) PWA manifest/SW presence.

### Established Patterns
- Fail loudly by name — refuse-to-load states, shadowing advisories, and zero-match notices all name their cause; nothing silent (D-13/D-14).
- Never silently empty a stream — the prune-only-on-healthy-sync rule (D-10) is this pattern applied to marks.
- Kernel-composed UI text for advisories (Phase 12: `last_notice` is never plugin text).
- Config stays the single source of truth for *configuration*; marks are deliberately the first kernel-owned user data **outside** config (Phase 11 D-01 framing).
- Schema-version-gated index rebuild (D-07 v1.0) — marks storage must be exempt from its drop list or live in a separate file.

### Integration Points
- Stream read path (`kernel/httpapi/stream.go` + search handler) — mark-aware filtering with an `excluded` view parameter.
- Sync completion path (`kernel/syncer/`) — the healthy-sync orphan-prune hook (D-09/D-10).
- `kernel/pluginhost/discover_binaries.go` + launch gate — manifest verification joins the existing pin verification site.
- `make build` / `Makefile` — link-time manifest embedding (plugins hashed, hash set injected into the kernel build).
- Kernel HTTP serving of the embedded UI — manifest.webmanifest/SW routes with correct MIME types; SW-driven update check against the running kernel's build identity.

</code_context>

<specifics>
## Specific Ideas

- The operator asked for **shift-click/ctrl-click standards** explicitly — desktop-native selection semantics, not checkboxes or a select mode; selections are "treated with include all / exclude all" via the bulk actions.
- TODO.md §Plugin Trust System is the operator's own written design; D-13's refuse-to-load stance comes directly from it ("it doesn't load it and outputs an error to the log AND the UI saying why"). The link-time manifest was recognized as the offline implementation of its "does the plugin exist in the davison/topos repo" check — it resolves the noted forge-unreachable-at-load-time concern by requiring no network.
- The Signal local-build demotion must be **documented clearly in the plugin doc** — the operator conditioned acceptance on the docs explaining why.

</specifics>

<deferred>
## Deferred Ideas

- **Plugin distribution** (TODO.md §Plugin Trust System): fork-based development with a `publish-plugin` GHA building binary + hash artifact, pull-by-repo-URL install with hash-verified fetch, single managed plugin directory, and PR-merge trust promotion — defers to backlog Phase 999.1 (PLUG-10/11). The backlog entry should absorb TODO.md's notes; Phase 13's manifest + refuse-to-load verification is the foundation it builds on.
- **Single plugin directory** (collapse of the Phase 11 two-tier layout) — revisit alongside PLUG-10 distribution (D-16).
- **Conversation-wide exclusion** for chat digests — match-rules/config territory if ever wanted; explicitly not a mark type (D-04).
- **Plugin picker search/filter box** (TODO.md vNext) — noted, stays vNext.

### Reviewed Todos (not folded)
- "Signal schema-version verify-and-accept tooling" (2026-08-05) — keyword-noise match; Signal plugin maintenance tooling, unrelated to curation, PWA, or trust hardening. Stays pending for a phase that touches the Signal plugin.

</deferred>

---

*Phase: 13-per-item-curation-installable-app*
*Context gathered: 2026-08-14*
