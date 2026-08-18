# Phase 11: External Plugins & the Trust Boundary - Context

**Gathered:** 2026-08-12
**Status:** Ready for planning

<domain>
## Phase Boundary

The kernel gains a second, external plugins directory. Binaries found there are discovered, configured, launched, and synced exactly like in-repo plugins, but are marked **untrusted** — trust derived purely from provenance (directory tier + a content hash pinned at add-time and re-verified at every launch), never from anything the plugin declares. Adding an untrusted source shows an explicit warning; an untrusted badge persists in the picker and on source chips. The plugin host also gains a generic per-instance extras map so external plugins receive provider-specific config keys without kernel changes, and the plugin launch environment is scrubbed to a minimal allowlist as part of drawing the trust boundary. (PLUG-06, PLUG-07, PLUG-08, PLUG-09)

Out of scope (already deferred): plugin distribution/pull-by-URL, developer guide, certification/trust promotion, sandboxing (trust marking is honest labeling, documented as such).

</domain>

<decisions>
## Implementation Decisions

### Hash pinning & upgrades
- **D-01:** The pinned content hash lives in **config TOML** (e.g. a `[plugins]`-level pins map), written by the existing hot-apply config path at add-time. No kernel-owned sidecar store — config stays the single source of truth, and Phase 13's per-item marks remain "the kernel's first user-owned data beyond config".
- **D-02:** Pins are **per external binary**, not per source instance. All instances of a plugin share one pin; a re-accept updates every instance at once (divergent per-instance pins can't legally exist — one binary on disk serves all instances).
- **D-03:** Hash mismatch at launch fails loudly by name into a **named "binary changed" health state** on the source chip; the per-chip menu (Phase 9 pattern) offers an explicit **"Trust updated binary"** re-pin action that shows the new hash and rewrites the pin via hot-apply config. No remove-and-re-add, no manual config editing required.
- **D-04:** Pinning applies to the **external tier only**. Trusted-dir binaries launch unpinned — their provenance is the directory itself (they're rebuilt constantly by `make build`/`make dev`; pinning them would false-alarm on every rebuild).

### Warning & badge UX
- **D-05:** Adding a source from an untrusted plugin inserts an explicit **confirm interstitial**: what "untrusted" means (code topos didn't build; no sandbox — honest labeling), the binary name, the full hash being pinned, plus a **type-the-plugin-name box** before confirm. The friction's purpose is accidental-click protection, not ceremony — keep it light.
- **D-06:** Chip badge = a small **warning glyph overlaid on the plugin's identity icon** inside the 44px merged pill; the health tooltip spells out "untrusted external plugin". No pill widening, no chip-wide tinting.
- **D-07:** Picker: external plugins list **inline in the existing install-catalog section** (alphabetical, as today), each row carrying the warning glyph + an "untrusted" label. No separate picker section.
- **D-08:** The pinned hash is **user-visible**: short copyable form in the per-chip menu/tooltip; full hash at the add-time confirm and in the re-pin flow — so a user can verify against an author-published checksum.

### External dir & collisions
- **D-09:** Default external dir resolves to the **platform data dir** via a small per-OS helper — `$XDG_DATA_HOME/topos/...` (`~/.local/share/topos/plugins-external`) on Linux, `~/Library/Application Support` on macOS, `%LOCALAPPDATA%` on Windows — overridable in config. Portable resolution without committing to Windows support (the project stays Linux-anchored). Survives release upgrades; drop-a-binary-in works with zero config.
- **D-10:** External binaries follow the **same `topos-plugin-` prefix convention**; discovery reuses the exact `DiscoverBinaries` semantics (regular files, one-level symlink follow, sorted).
- **D-11:** Name collision between tiers: **trusted shadows external**, with a loud, named log line (never silent). Phase 12's rehearsal handles this as a test-setup detail (renamed copy or a trusted dir without the fs plugin).

### Extras config shape
- **D-12:** Extras are a dedicated **`[sources.X.extras]` sub-table** of arbitrary keys. Kernel-known fields stay strictly typed at the top level (typos still caught); the canonical TOML rewrite round-trips extras as one opaque map.
- **D-13:** Extras values are **strings only** — `map[string]string` end to end (TOML → wire → plugin). Plugins parse what they need from strings, same as env vars. — **Reversibility:** costly — the wire shape lands in the published `topos.v1` plugin contract third parties build against; moving to typed values later means a contract addition and migration, not an internal edit.
- **D-14:** **Env hygiene at the trust boundary** (explicitly in scope for this phase): plugin subprocesses stop inheriting the kernel's full `os.Environ()` (today: `kernel/pluginhost/host.go:449`) and get a **minimal allowlisted environment** (PATH/HOME/locale-class vars + that instance's own expanded config values). `${VAR}` refs in extras expand uniformly like built-in fields — so only values the user explicitly typed into that instance's config ever reach the plugin. The untrusted confirm step **lists every env var whose value is being handed over**; plugin-suggested extras defaults are never auto-filled (blocks a malicious Describe response from suggesting `${SOME_SECRET}`). Canonical rewrite preserves literal `${VAR}` forms, never expanded values. This keeps SRC-05's bring-your-own-credentials flow working for Phase 14.
- **D-15:** The contract's **Describe response gains an optional declaration of expected extras keys** (key, label, required, secret-ish hint) so the add form renders labeled inputs; a **free-form key/value editor** underneath covers undeclared keys (older kernel + newer plugin still workable). Declared defaults are display-only per D-14.

### Claude's Discretion
- Hash algorithm choice (SHA-256 is the obvious default) and pin map key/format in TOML.
- Exact name and copy of the "binary changed" health state and the warning interstitial text.
- Whether `ExcludedPluginBinaries` (mock/mockstrict) policy applies to the external tier.
- The proof binary for success criterion 5 (a real out-of-repo build discovered, marked untrusted, synced end to end) — likely a rebuilt/renamed mock-derived plugin; planner's call.
- Precise env allowlist contents beyond "PATH/HOME/locale + instance config values".
- Discovery timing (per-request directory listing, matching today's behavior, vs cached) — follow existing kernel patterns.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Plugin contract & API
- `docs/plugin-contract.md` — the published `topos.v1` contract external authors build against; Phase 11 extends it (Describe extras declaration, launch-environment behavior) and must republish coherently. Documents `WEBSPACES_SOURCE_CONFIG` env-var config delivery (§ around line 196).
- `docs/api.md` — HTTP API surface incl. describe-plugin / config endpoints the picker and add-form flows extend; notes the existing "boolean only, never the value" env-presence discipline (D-05 lineage, ~line 566).
- `proto/topos/` — proto source of truth for the contract; Describe extras declaration lands here (via `buf`).

### Requirements & roadmap
- `.planning/REQUIREMENTS.md` — PLUG-06..09 definitions and the Out of Scope table (no sandboxing, honest-labeling framing).
- `.planning/ROADMAP.md` — Phase 11 goal + 5 success criteria (incl. criterion 5's real-external-binary proof); Phase 12/14 dependencies on this phase.

### Testing conventions
- `docs/testing.md` — testing map and gates; **note the e2e harness already spawns kernels with an explicit environment allowlist (~line 107)** — direct precedent for D-14's plugin-env scrubbing. UI work in this phase must extend the Playwright suite as definition of done (project convention in `.claude/CLAUDE.md`).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `kernel/pluginhost/discover_binaries.go` — `DiscoverBinaries`/`DiscoverAllBinaries` (prefix convention, symlink-following, UI-policy vs security-authority split, `ExcludedPluginBinaries`). The external tier is a second directory fed through the same machinery plus tier tagging.
- `kernel/config/types.go` — `PluginsConfig{Dir}` (default `plugins`, exe-relative) gains the external dir + pins map; `Source` struct gains the `extras` table.
- Hot-apply config path (`kernel/httpapi/config.go`, `PUT /api/config` with content-hash optimistic lock + canonical TOML rewrite) — carries pin writes (D-01) and re-pin (D-03); **the canonical rewrite must round-trip the new extras table and pins map**.
- Phase 9 per-chip menu — integration point for the re-pin action and hash display.
- Named health-state vocabulary (WhatsApp's five states are the precedent) — "binary changed" joins it.
- Plugin identity icons (Describe-carried, CSP-sandboxed route) — the badge glyph overlays these in chips and picker rows.

### Established Patterns
- "The kernel holds no built-in table of known plugin types" (D-05 discipline) — external discovery must stay directory-listing-driven; trust must derive from tier + pin, never from binary self-declaration.
- Fail loudly by name — mismatches and shadowed collisions surface with names, never silently.
- Secrets are environment-only `${VAR}` references in config (D-04 lineage); extras follow the same rule.
- Every plugin binary change touches `internal/audit` four-key provenance checks — new/renamed plugin binaries (criterion 5 proof binary, Phase 12 rehearsal copies) interact with this.

### Integration Points
- `kernel/pluginhost/host.go:449` — the `append(os.Environ(), ...)` launch-env construction D-14 replaces with an allowlist.
- Picker catalog (`plugin-types` API + two-section picker component) — external plugins join the catalog with tier metadata.
- `web/e2e/specs/` — new specs: untrusted add flow (interstitial + typed name), badge presence, binary-changed state + re-pin, extras form round-trip.

</code_context>

<specifics>
## Specific Ideas

- The typed-plugin-name box at the untrusted confirm is for **accidental-click protection, not rm-rf ceremony** — user's explicit framing; keep the interstitial light.
- User raised the env-hoovering threat themselves ("a malicious plugin could hoover up any number of values from well known env vars and phone home") — D-14's scrub+disclose design is the direct response; treat it as a first-class deliverable of the trust boundary, not a nice-to-have.
- Windows: user asked about XDG equivalence; agreed position is portable per-OS resolution (small helper) **without** committing to Windows support — the project stays Linux-anchored (D-Bus keyring, desktop chat DBs).

</specifics>

<deferred>
## Deferred Ideas

### Reviewed Todos (not folded)
- `2026-08-05-signal-schema-version-verify-and-accept-tooling.md` — reviewed at phase start; matched only on generic keywords. It's Signal-plugin maintenance tooling, unrelated to the trust boundary. Stays pending for a phase that touches the Signal plugin.

No new scope-creep ideas surfaced — discussion stayed within phase scope.

</deferred>

---

*Phase: 11-External Plugins & the Trust Boundary*
*Context gathered: 2026-08-12*
