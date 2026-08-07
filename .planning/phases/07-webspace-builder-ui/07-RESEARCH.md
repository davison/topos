# Phase 7: Webspace Builder UI - Research

**Researched:** 2026-08-07
**Domain:** Config-writing HTTP API (Go/chi/go-toml) + composition UI (SvelteKit/bits-ui/shadcn-svelte)
**Confidence:** MEDIUM — the config-shape and UI-shape decisions are already locked in 07-CONTEXT.md; what remained to research was **how the existing codebase's own architecture constrains the implementation** of those decisions. Several load-bearing findings below (hot-apply plumbing, secret round-tripping, Describe-before-persist sequencing) are original analysis of this repo's code, not textbook patterns — flagged accordingly.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Config persistence & clobber safety**
- **D-01:** UI saves perform a **canonical rewrite** of the whole config file — one machine-written form regenerated on every save. Hand-written comments are flattened the first time the UI saves; no comment-preserving round-trip machinery is built. — *Reversibility: reversible.*
- **D-02:** Canonical style is **minimal + header pointer**: clean TOML with just the values, preceded by a short generated header comment ("managed by the topos UI — hand-edits are honored via Reload; see config.example.toml for full documentation"). No per-key generated doc comments.
- **D-03:** Clobber guard is an **optimistic lock on file content hash**: the kernel records the config file's hash at load; a UI save first re-checks it, and if the file changed on disk since, the save is **rejected** ("config changed on disk — review and retry"), the kernel reloads the newer file, and the UI refreshes. No merge attempt, no silent loss.
- **D-04:** **Single rolling backup**: every UI save first writes the outgoing file to `config.toml.bak`, overwriting the previous backup. No timestamped set, no backup directory.
- **D-05 (hard requirement):** `${ENV_VAR}` secret references are written back **verbatim, never expanded** — the kernel must retain the raw pre-expansion form of the config for persistence (expansion via os.Expand happens at load only). Secret *values* never appear in the file, the API, or the UI. — *Reversibility: one-way in effect; treat as a tested invariant.*

**Apply / reload semantics**
- **D-06:** **Save = apply immediately**: one request validates, writes the file, and hot-swaps the kernel's running config. No restart, no separate apply step. — *Reversibility: costly.*
- **D-07:** Reconciliation after apply is **eager**: a new/connection-changed instance syncs immediately; a changed webspace/match block immediately re-syncs affected sources; a removed instance's plugin process shuts down and its index rows are removed right away.
- **D-08:** Hand-edits reach the running kernel via an **explicit Reload affordance** (UI button + API endpoint) — same validate-then-apply path. **No file watcher.** An invalid file on reload keeps the last-good config running and surfaces the error.
- **D-09:** Validation is **validate-on-save only**: the save endpoint runs the kernel's full existing load-time validation as a dry-run before writing; on failure nothing is written and the UI shows the kernel's error messages verbatim. No live per-field validation endpoint, no client-side reimplementation of the rules.

**Builder UX shape (locked: "yes, that matches — lock it in")**
- **D-10:** **Full in-header composition — no standalone settings section, standalone home page retired.** Title becomes a drop-down webspace switcher with a "+" to create. Root URL redirects to the first/last-visited webspace; a zero-webspaces empty state hosts first-run creation.
- **D-11:** A **"+" at the end of the source-chip row** adds a source to the current webspace. Picker offers **existing configured instances** and **"New <plugin type>…"** entries for discovered plugin binaries. Existing instance → modal asking only for match fields. New plugin type → **two-step modal: connection config (instance) then match config (webspace)**, driven by the plugin's declared match vocabulary (Describe RPC, Phase 5 D-05).
- **D-12:** Editing an existing source's config happens via a **chip menu/popover affordance — never plain chip click** (click stays filter-toggle per Phase 6 D-01). Editing instance-level connection fields must be visibly marked as affecting every webspace using that instance.
- **D-13:** A minimal **"Manage sources…" entry in the title drop-down** is the escape hatch for instance-level edit/delete, webspace deletion, and the Reload-config affordance. No other global settings surface.
- **D-14:** **UI-built webspaces always write an explicit `sources` allowlist** — participation is exactly what was added via "+". Hand-written webspaces without an allowlist keep Phase 5 D-03's all-instances-participate default. The builder never silently rewrites a hand-written webspace's participation model unless the user edits it in the UI.
- **D-15:** Secret fields in the modal ask for the **environment variable name** (persisted as `token = "${VAR}"`), with a **set/unset badge** reported by the kernel for that variable in its environment. Unset ⇒ save still succeeds with a warning. Values are never displayed or transmitted.

**Search-promotion filter semantics**
- **D-16:** A promoted filter acts at **query time as an FTS filter, and is also applied to the `/agent/v1` read surface** — the filtered view IS the webspace for every consumer. Index contents stay exactly as match config dictates; no sync-time narrowing. — *Reversibility: costly.*
- **D-17:** Filters persist as a **config key on the webspace block** (part of the webspace's definition), riding the whole Phase 7 machinery.
- **D-18:** Filters are a **stackable AND list** (e.g. `filter = ["boiler", "quote"]`): each promotion appends a term; all terms AND together; each is removable independently. Live searches AND with the active filter stack.
- **D-19:** UI surfacing: after a search, a **"Save as filter" affordance** appears by the search box; active permanent filters render as **labeled chips visually distinct from source chips**, each with an × to remove. Removing writes config immediately through the same save path.

### Claude's Discretion
- Mutating-API design (endpoint shapes, HTTP verbs, request/response envelope) — must follow `docs/api.md` conventions; loopback-only/no-auth posture unchanged this phase.
- How the kernel retains the raw pre-expansion config for canonical rewrite (re-parse on save vs. retained AST/document model), TOML serializer choice, and atomic-write mechanics (temp file + rename).
- Hot-apply internals: diffing old vs. new config, pluginhost instance lifecycle ordering, syncer re-registration, in-flight sync handling during apply.
- Modal/form layout details, picker presentation, delete-confirmation UX, first-run empty state design, and what "first/last webspace" the root redirect targets.
- FTS query semantics for stacked filter terms (phrase vs. token handling), and how filter chips interact visually with the existing source-chip filtering row.
- Where the "+"-picker learns about available-but-unconfigured plugin types (plugin binary discovery is existing kernel behavior).

### Deferred Ideas (OUT OF SCOPE)
- Comment-preserving TOML round-trip — declined in favor of canonical rewrite (D-01).
- File watcher / auto-reload of hand-edits — declined (D-08); explicit Reload only.
- WR-01 (Phase 6 review advisory: `highlightText` case-fold positional bug in `web/src/lib/format.ts`) — offered as a fold, declined; remains open for a later phase.
- Signal schema-version verify-and-accept tooling — explicitly declined for this phase.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| KERN-08 | Webspace and source-instance configuration is editable through the kernel API (non-secret fields only; secrets stay environment-only), while hand-editing the config file remains supported | See "The config write path" and "Hot-apply architecture" below — covers the dual-parse raw/expanded config model (D-05), the content-hash optimistic lock (D-03), the canonical writer (D-01/D-02), and the new mutable-config-holder plumbing needed for D-06's immediate hot-swap. |
| UI-12 | Webspace builder UI — pick plugin types, configure named instances, save the set as a webspace, and promote a live search into a permanent webspace filter refinable by further search | See "Frontend composition" and "Search-promotion filter" below — covers the header-drop-down/chip-picker component plan, the Describe-after-connection-config sequencing for the two-step modal (D-11), and the FTS filter-stack query builder (D-16/D-18) applied uniformly to stream/search/agent-stream. |
</phase_requirements>

## Summary

This phase adds the kernel's first mutating HTTP surface (config read/write/reload) and a composition UI built entirely inside the existing header (`WebspaceHeader.svelte`). The *what* (config shape, UX shape, filter semantics) is already locked in 07-CONTEXT.md — the research below is almost entirely about *how the current codebase's own wiring* forces or constrains the implementation, because this phase touches three load-bearing seams that have never been touched before:

1. **Secret round-tripping.** `config.Load` today expands `${VAR}` references in the raw file bytes *before* unmarshaling — the in-memory `*config.Config` a request handler sees already holds real secret values, never the `"${VAR}"` literal. D-05's "write `${VAR}` back verbatim" requirement cannot be satisfied from that struct. The clean fix (no AST, no retained document model) is to **unmarshal the raw, unexpanded file bytes into a second `*config.Config` at load time** and treat *that* struct — not the expanded runtime one — as the canonical form the UI edits and the writer serializes. Both are the same Go type; only which string went through `os.Expand` differs.

2. **Hot-apply plumbing does not exist yet.** Every current call site (`httpapi.Router`, `agent.go`, `correlate.Engine`, `syncer.Scheduler`) captures `*config.Config` as a **fixed pointer at kernel startup** — there is no indirection to swap through. `pluginhost.Host` only supports "launch everything" (`Discover`) and "kill everything" (`Shutdown`); there is no incremental reconcile. `syncer.Scheduler.Run` spawns one goroutine per source **once**, from a config snapshot, and loops until its `ctx` is cancelled — it has no way to notice a source was added, removed, or re-intervalled. D-06 ("save = apply immediately, no restart") requires new infrastructure: a swappable config holder, and a supervisor that can tear down and rebuild the plugin host + coordinator + scheduler goroutine set on every apply. For MVP scope, the simplest correct approach is a **full rebuild** (kill every plugin, relaunch every plugin, restart the scheduler) rather than a surgical diff — see Pitfall 1.

3. **The two-step "New plugin type" modal (D-11) has a sequencing dependency the context doesn't spell out.** `DescribeResponse.match_vocabulary` (the thing step 2's form is built from) is only knowable by actually launching the plugin subprocess and calling `Describe` — the proto carries no static "this plugin type declares these connection fields / this match vocabulary" schema the kernel can read without a live process. This is exactly why D-11 already orders the modal connection-config-then-match-config: **step 1's submitted connection fields are used to trial-launch the plugin, call Describe, and immediately kill it** (no config is written yet), and step 2 renders from that response. Verified for the one connection-optional plugin (Signal) that this trial-launch does not require *working* credentials, only *present* fields — see Pitfall 4.

**Primary recommendation:** Build the config write path as (a) a `config.Store` wrapping an `atomic.Pointer[config.Config]` pair — one raw/unexpanded, one expanded/runtime — threaded through every current `*config.Config` call site in place of the raw pointer; (b) a canonical TOML writer that serializes the raw struct via `go-toml/v2`'s `Marshal` (which already sorts map keys deterministically — verified against the vendored source); (c) a `pluginhost.DescribePluginType` helper that reuses the existing unexported `launch` machinery for the trial-launch-then-kill Describe call; (d) a new `index.Store.DeleteSourceItems` for D-07's removed-instance cleanup (no such method exists today); and (e) an FTS query builder shared by the stream, search, and agent-stream handlers so a webspace's persisted `filter` stack is honored identically everywhere, per D-16.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Config file read/write/hash-lock/backup | API / Backend (`kernel/config`) | — | Filesystem + secret handling must never leave the kernel process; the UI never sees raw file bytes or secret values. |
| Config validate-on-save (dry run) | API / Backend (`kernel/config.Validate` + `kernel/pluginhost.ValidateMatchConfig`) | — | D-09 mandates reusing the existing load-time validator verbatim — no second validation implementation. |
| Hot-apply (plugin lifecycle, syncer re-registration, scheduler restart) | API / Backend (`kernel/pluginhost`, `kernel/syncer`, `cmd/topos`) | — | Plugin subprocess supervision and background scheduling are kernel-only concerns; the browser has no visibility into subprocess state. |
| Plugin-type discovery ("+"-picker's "New <type>…" list) | API / Backend (`kernel/pluginhost`, directory listing) | Browser (renders the returned list) | Only the kernel process can see the `plugins/` directory on the desktop machine's filesystem. |
| Match-vocabulary discovery for a not-yet-configured instance | API / Backend (trial-launch + Describe RPC) | — | Only the kernel can launch a plugin subprocess and speak gRPC to it; this cannot be inferred client-side. |
| Webspace switcher, "+" chip picker, modals, filter chips | Browser / Client (SvelteKit SPA) | — | Pure presentation/composition state over the kernel's JSON API — no server-rendering need beyond what already exists. |
| FTS filter-stack application (stream/search/agent-stream) | API / Backend (`kernel/index`, `kernel/httpapi`) | — | The filtered view must be identical for the human UI and the agent surface (D-16) — computed once, server-side, never duplicated in the browser. |
| Env-var set/unset badge | API / Backend (`os.LookupEnv`, exposed via config response) | Browser (renders the badge) | The kernel process is the only place that can see its own environment; the UI never receives the value, only a boolean. |

## Standard Stack

### Core

No new external packages are required — this phase is built entirely on dependencies already vetted and present in the repo.

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|---------------|
| `github.com/pelletier/go-toml/v2` | v2.4.3 [VERIFIED: go.mod:21] | TOML decode (already used) **and** encode (new use this phase) | Already the project's TOML library (`kernel/config/config.go:11`); its `Marshal` sorts map keys deterministically ([VERIFIED: `/opt/go/pkg/mod/github.com/pelletier/go-toml/v2@v2.4.3/marshaler.go:1007-1032`], `slices.SortFunc(entries, func(a, b entry) int { return strings.Compare(a.key, b.key) })`) — exactly the determinism D-03's content-hash lock and D-01's canonical rewrite need, with zero extra code. Currently marked `// indirect` in go.mod ([VERIFIED: go.mod:21]) despite `go-chi/chi` and `hashicorp/go-plugin` *also* being marked indirect while directly imported elsewhere in the tree ([VERIFIED: go.mod:9,14] vs. direct imports in `kernel/httpapi/routes.go:15` and `kernel/pluginhost/host.go:20]) — this repo's `go.mod` indirect markers are already stale from a prior deliberately-skipped `go mod tidy` (STATE.md, Phase 05-04 note); promoting this one similarly is not a new pattern. |
| `bits-ui` | v2.18.1 [VERIFIED: web/node_modules/bits-ui/package.json] | Headless primitives backing shadcn-svelte's Dialog/Dropdown-menu/Select/Combobox | Already a direct dependency (`web/package.json`); confirmed its bundled component set already includes `dialog`, `dropdown-menu`, `select`, `combobox`, `alert-dialog` ([VERIFIED: `web/node_modules/bits-ui/dist/bits/` directory listing]) — adding the shadcn-svelte wrapper components this phase needs (modals for the two-step instance/match config, the webspace-switcher drop-down) requires **zero new npm dependencies**, only new hand-authored files under `web/src/lib/components/ui/`, following the same pattern the repo already used for `popover`/`tooltip`/etc. |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Go stdlib `crypto/sha256` | stdlib | Content-hash for D-03's optimistic lock | Hash the raw file bytes at load and at pre-save re-check; no external hashing library needed for a single-file, single-writer local tool. |
| Go stdlib `os.CreateTemp` + `os.Rename` | stdlib | Atomic write for the canonical rewrite + `.bak` (D-01/D-04) | Standard Go idiom for crash-safe file replacement: write to a temp file in the same directory, `fsync`, then `os.Rename` over the target — rename is atomic on the same filesystem, avoiding a torn/partial `config.toml` if the kernel is killed mid-write. |
| Go stdlib `sync/atomic` (`atomic.Pointer[T]`) | stdlib (Go 1.23+ per project's pinned toolchain) | Swappable config holder for D-06's hot-apply | No existing `atomic.*`/`sync.Mutex` usage exists anywhere in `kernel/` today ([VERIFIED: `grep -rn "atomic\.\|sync.RWMutex\|sync.Mutex" kernel --include="*.go"` returned zero non-test matches]) — this is genuinely new infrastructure for this phase, not an established local pattern to imitate. |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Dual-parse (raw + expanded) `*config.Config` pair | A retained TOML AST/document model (e.g. hand-rolled node tree preserving byte offsets) | Only worth it if comment-preserving round-trip were in scope — it explicitly isn't (D-01). The dual-parse approach reuses the exact same `config.Config` struct and `toml.Unmarshal` call twice with different input strings; an AST model would be new, unused-elsewhere machinery solving a problem (comment preservation) this phase doesn't have. |
| Full plugin-host rebuild on every apply (kill all, relaunch all) | A surgical add/remove/restart-changed diff against `pluginhost.Host.plugins` | The diff is the "correct" long-term answer and is explicitly reserved as Claude's Discretion ("diffing old vs. new config... in-flight sync handling during apply") — but `Host` has no such diff method today, and building one is materially more work than a rebuild for an MVP-mode phase. A full rebuild is simpler to get correct (no partial-failure edge cases to reason about) at the cost of a brief (sub-second, all-local-subprocess) reachability blip on every save for sources *not* touched by that save. |
| shadcn-svelte `Combobox`/`cmdk`-style search-filterable list for the "+" picker | Reusing the existing `Popover` + plain list pattern already used for the chip-overflow menu (`WebspaceHeader.svelte:200-231`) | With five plugin types and a handful of instances in a real deployment, a filterable combobox is unnecessary complexity; a Popover with a flat list (existing house pattern) is simpler and matches the user's own stated expectation of "a working composition flow first, not pixel-perfect modals." Revisit only if a deployment's instance count grows enough that scrolling a flat list becomes the actual bottleneck (mirrors UI-07's own precedent for the chip row itself). |

**Installation:**
```bash
# Go: promote the already-vendored go-toml/v2 from indirect to direct
cd /home/darren/projects/davison/topos
go get github.com/pelletier/go-toml/v2@v2.4.3

# Frontend: add the shadcn-svelte Dialog and Dropdown Menu source files.
# NOTE (Pitfall 6, below): this repo's components.json already drifted from
# the live shadcn-svelte CLI's registry scheme (STATE.md, Phase 06 note) —
# treat the CLI's output as a starting point to hand-adapt, not a drop-in,
# exactly as every other component under web/src/lib/components/ui/ was.
cd web && npx shadcn-svelte@latest add dialog dropdown-menu
```

**Version verification:** `go-toml/v2 v2.4.3` and `bits-ui 2.18.1` were confirmed present in this repo's own `go.mod` / `node_modules` — no registry lookup was needed since nothing new is being introduced.

## Package Legitimacy Audit

**No new external packages are introduced by this phase.** `go-toml/v2` is promoted from an existing indirect dependency (already fully vetted, already in `go.sum`) to a direct one; the shadcn-svelte `dialog`/`dropdown-menu` additions are source-file copies over the already-installed `bits-ui` primitives, adding no new `package.json` entries. The Package Legitimacy Gate protocol (registry/downloads/postinstall-script checks) does not apply — there is nothing new to typosquat-check.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|--------------|---------|-------------|
| `github.com/pelletier/go-toml/v2` | Go modules | Already vendored | N/A (Go modules, not download-counted) | github.com/pelletier/go-toml | OK | Approved — already a transitive dependency, promoted to direct only |
| `bits-ui` | npm | Already installed | N/A (already in package.json) | github.com/huntabyte/bits-ui | OK | Approved — no new install, existing dependency's bundled components used |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
Browser (SvelteKit SPA)
  │
  │  GET  /api/webspaces, /api/sources                 (existing, unchanged)
  │  GET  /api/config                                   (NEW — read raw+resolved config for the builder)
  │  POST /api/config/describe-plugin                   (NEW — trial-launch + Describe, D-11 step 1→2)
  │  POST /api/config/save                               (NEW — validate-dry-run → hash-check → write → hot-apply)
  │  POST /api/config/reload                             (NEW — D-08, hand-edit pickup)
  │  GET  /api/webspaces/{ws}/stream?...                 (existing route, now filter-stack aware — D-16)
  ▼
kernel/httpapi (chi router)
  │
  ├─▶ kernel/config.Store (NEW: atomic.Pointer[Config] pair — raw + expanded)
  │     ├─ Load(path)            → (rawCfg, expandedCfg, fileHash)
  │     ├─ Save(mutation, prevHash) → validate-dry-run(expandedCfg') → hash-recheck → atomic write + .bak → Swap()
  │     └─ Reload(path)          → same Load path, same Swap()
  │
  ├─▶ kernel/pluginhost.Host (EXTENDED: incremental relaunch on Swap, D-06/D-07)
  │     ├─ DescribePluginType(ctx, pluginsDir, binary, src) → trial-launch, Describe, Kill  [NEW]
  │     └─ Reconcile(ctx, newSources) → kill removed, launch added/changed                  [NEW]
  │
  ├─▶ kernel/syncer.Coordinator + Scheduler (RESTARTED on Swap: new Host.Plugins() snapshot,
  │     fresh scheduler goroutines cancelled/respawned against the new Config)
  │
  └─▶ kernel/index.Store
        ├─ DeleteSourceItems(ctx, source)              [NEW — D-07 removed-instance cleanup]
        └─ Stream/Search now take an optional filter-term list (D-16/D-18), AND-ed with any
           live search query — one shared query-builder function, called from StreamHandler,
           SearchHandler, and agentStreamHandler alike.

Plugin subprocesses (go-plugin/gRPC) — UNCHANGED this phase: no new RPCs, no contract version
bump. The trial-launch path above reuses the existing Discover/launch/Describe/Kill sequence
verbatim, just against one instance instead of the whole configured set.
```

### Recommended Project Structure

```
kernel/config/
├── types.go          # Webspace gains Filter []string `toml:"filter,omitempty"` (D-17)
├── config.go          # Load() extended to also parse the raw/unexpanded pass (D-05)
├── store.go            # NEW: config.Store — atomic.Pointer pair, Save/Reload/hash-lock (D-03/D-06)
└── writer.go            # NEW: canonical TOML serialization (D-01/D-02) — header + toml.Marshal(rawCfg)

kernel/pluginhost/
├── host.go             # Discover/launch/Shutdown unchanged; ADD Reconcile, DescribePluginType
└── discover_binaries.go # NEW: list plugins/ dir entries matching the topos-plugin-<type> convention

kernel/httpapi/
├── config.go            # NEW: GET/POST /api/config* handlers (the phase's mutating surface)
└── stream.go, search.go, agent.go  # extended with the shared filter-stack query builder (D-16)

kernel/index/
└── store.go             # ADD DeleteSourceItems (D-07); extend Search/StreamItems call sites for filters

web/src/lib/components/
├── ui/dialog/, ui/dropdown-menu/    # NEW shadcn-svelte wrapper components
├── WebspaceSwitcher.svelte           # NEW — title drop-down (D-10)
├── AddSourceModal.svelte             # NEW — the "+" chip picker + two-step modal (D-11)
├── EditSourceModal.svelte            # NEW — chip popover edit affordance (D-12)
├── ManageSourcesModal.svelte         # NEW — "Manage sources…" escape hatch (D-13)
├── FilterChip.svelte                 # NEW — permanent-filter chip, distinct styling from SourceChip (D-19)
└── WebspaceHeader.svelte             # extended: switcher + "+" + filter-chip row

web/src/routes/
├── +page.svelte         # becomes a redirect-only route (D-10) — first/last-visited webspace, or empty state
└── w/[webspace]/+page.svelte  # gains save-as-filter / filter-chip wiring, "+" picker wiring
```

### Pattern 1: Dual-parse raw + expanded config for secret-safe round-tripping (D-05)

**What:** Parse the config file's raw bytes into a `*config.Config` **twice** at load/reload time — once through the existing `os.Expand`-then-unmarshal path (the runtime config every handler that needs real connectivity uses), and once by unmarshaling the *unexpanded* raw bytes directly into a second `*config.Config` (the "canonical" struct the UI reads/edits and the writer serializes). Both instances are the same Go type; the only difference is whether `${VAR}` tokens were substituted before `toml.Unmarshal` ran.

**When to use:** Any time the API needs to expose or persist a field that might be a secret reference (`token`, and by the same reasoning any future secret-shaped field) without ever holding or leaking the real value.

**Example:**
```go
// Source: this repo's own kernel/config/config.go:21-48 (existing Load),
// extended per D-05's requirement. Verified against the current
// implementation, which already computes `expanded` from `raw` via
// os.Expand before unmarshaling (config.go:27-32) — the only change needed
// is unmarshaling `raw` itself into a second struct alongside `expanded`.
func Load(path string) (*Config, *Config, error) { // (expanded, raw, err)
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	expandedText, missing := expandEnv(string(raw))

	var expandedCfg Config
	if err := toml.Unmarshal([]byte(expandedText), &expandedCfg); err != nil {
		return nil, nil, fmt.Errorf("config: parse %s: %w", path, err)
	}
	applyDefaults(&expandedCfg)
	// ... expandIndexPathHome, expandSourceCACertPathsHome, Validate(missing) unchanged ...

	// rawCfg never sees os.Expand — every ${VAR} token in the file is still
	// a literal "${VAR}" string here. This is the struct the config-write
	// path edits and re-serializes; it is NEVER used to launch a plugin
	// subprocess or make a live connection (it would fail — base_url would
	// literally be "${PAPERLESS_URL}").
	var rawCfg Config
	if err := toml.Unmarshal(raw, &rawCfg); err != nil {
		return nil, nil, fmt.Errorf("config: parse (raw) %s: %w", path, err)
	}
	applyDefaults(&rawCfg) // defaults are structural, not secret-bearing — safe to apply to both

	return &expandedCfg, &rawCfg, nil
}
```

### Pattern 2: Canonical TOML writer with a generated header (D-01/D-02)

**What:** Serialize the raw/unexpanded `*config.Config` via `go-toml/v2`'s `Marshal`, prepend a short fixed header comment, and write atomically.

**Example:**
```go
// Source: original composition for this phase, built on verified
// go-toml/v2 Marshal behavior (deterministic key sort — see Standard
// Stack, above) and the standard Go atomic-write idiom.
const canonicalHeader = `# managed by the topos UI — hand-edits are honored via Reload
# see config.example.toml for full field documentation

`

func WriteCanonical(path string, rawCfg *Config) error {
	body, err := toml.Marshal(rawCfg)
	if err != nil {
		return fmt.Errorf("config: marshal canonical form: %w", err)
	}
	out := append([]byte(canonicalHeader), body...)

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".config-*.toml")
	if err != nil {
		return fmt.Errorf("config: create temp file: %w", err)
	}
	defer os.Remove(tmp.Name()) // no-op once Rename succeeds

	if _, err := tmp.Write(out); err != nil {
		tmp.Close()
		return fmt.Errorf("config: write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("config: fsync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("config: close temp file: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("config: rename into place: %w", err)
	}
	return nil
}
```

### Pattern 3: Trial-launch for Describe, before persisting anything (D-11 step 1 → step 2)

**What:** When the user submits step 1's connection config for a brand-new instance, launch the plugin subprocess exactly as `pluginhost.launch` already does for a configured source — but against the just-submitted (not-yet-persisted) `config.Source` — call `Describe`, capture `match_vocabulary`/`display_name`, then `Kill()` the subprocess immediately. Nothing is written to disk or applied to the running `Host` at this point; only step 2's save actually persists+applies.

**When to use:** Populating the match-vocabulary-driven form in D-11's step 2, for a plugin type with no existing configured instance.

**Verified precondition:** at least one real plugin in this repo (`plugins/signal/main.go`) only requires its connection field(s) to be **present**, not **working**, before it starts serving RPCs — `main()` calls `fatal()`/`os.Exit` only if `WEBSPACES_SOURCE_CONFIG` is unset or `path` is empty ([VERIFIED: `plugins/signal/main.go:37-49`]); it does not open the SQLCipher database or resolve the OS keyring until a later RPC call (`NewSourcePlugin(configDir)` at line 56 constructs the struct only). This means a trial-launch with placeholder-but-present connection fields is expected to succeed far enough to answer `Describe` even before the user has verified their real credentials — consistent with `Describe` being a static, connection-independent response in every plugin (`source_type`, `display_name`, `contract_version`, `match_vocabulary` are all compile-time constants per plugin, per `docs/plugin-contract.md:254-259`). This has not been verified against every plugin's `main.go` in this repo — flagged as an assumption to confirm for paperless/silverbullet/proton during planning (see Assumptions Log, A2).

**Example:**
```go
// Source: adapts the existing unexported launch() in
// kernel/pluginhost/host.go:134-220 (unchanged signature/behavior) — this
// is a NEW exported wrapper, not a modification to launch() itself.
func DescribePluginType(ctx context.Context, pluginsDir string, src config.Source, logger hclog.Logger) (DescribeInfo, error) {
	p, err := launch(ctx, pluginsDir, "__trial__", src, logger)
	if err != nil {
		return DescribeInfo{}, fmt.Errorf("pluginhost: trial-launch for describe: %w", err)
	}
	defer p.Kill()

	return DescribeInfo{
		SourceType:      p.SourceType(),
		DisplayName:     p.PluginDisplayName(),
		MatchVocabulary: p.MatchVocabulary(),
	}, nil
}
```

### Pattern 4: One shared FTS filter-stack query builder (D-16/D-18)

**What:** A webspace's persisted `filter` list and any live in-flight search query must combine into one FTS5 `MATCH` expression, applied identically by `StreamHandler`, `SearchHandler`, and `agentStreamHandler` — never three separate implementations that could drift.

**Verified starting point:** the existing `ftsQuery` helper ([VERIFIED: `kernel/index/store.go:375-390`], quoted verbatim: `fields := strings.Fields(raw)` ... `kept = append(kept, `"`+f+`"`)` ... `kept[len(kept)-1] += "*"` ... `return strings.Join(kept, " ")`) already phrase-quotes each whitespace-delimited term and AND-combines them (FTS5's implicit operator between bare MATCH terms is AND) with only the *final* term getting a prefix-match `*` suffix — this is exactly the "as-you-type" behavior appropriate for a live search box, but a **permanent filter term should not get prefix-matched** (a saved filter of `"boiler"` should mean the word "boiler", not silently expand `boilerplate` if the corpus ever has one). The query builder for this phase needs to distinguish "filter terms" (each phrase-quoted, no trailing `*`) from "the live search query" (phrase-quoted terms, trailing-`*` on the last one only, appended after the filter terms) — both AND together in the final MATCH string, per D-18.

**Example:**
```go
// Source: original composition combining ftsQuery's existing phrase-
// quoting convention (kernel/index/store.go:375-390, verified above) with
// D-18's stacked-AND-filter requirement. filterTerms come from
// Webspace.Filter (persisted); liveQuery is the optional in-flight search
// box value (empty for a plain stream read).
func buildMatchQuery(filterTerms []string, liveQuery string) string {
	var parts []string
	for _, term := range filterTerms {
		term = strings.ReplaceAll(term, `"`, "")
		if term == "" {
			continue
		}
		parts = append(parts, `"`+term+`"`) // no trailing * — exact phrase, not prefix
	}
	if live := ftsQuery(liveQuery); live != "" {
		parts = append(parts, live) // ftsQuery already phrase-quotes + prefix-matches its own last term
	}
	return strings.Join(parts, " ") // FTS5's implicit AND between space-separated terms
}
```

### Anti-Patterns to Avoid

- **Calling `toml.Marshal` on the *expanded* runtime `*config.Config`:** this is the D-05 violation waiting to happen — the expanded struct's `Token`/`BaseURL` fields hold real secret text after `os.Expand`. Every write path must serialize the *raw*, unexpanded struct (Pattern 1).
- **Reimplementing config validation for the save dry-run:** D-09 is explicit that the save endpoint reuses `config.Validate` + `pluginhost.ValidateMatchConfig` byte-for-byte, including their existing error message construction (`"config: webspace %q ..."` etc., already sorted-and-deterministic per `kernel/config/config.go`'s existing discipline) — the UI must not have its own copy of "must declare either base_url+token or path" or any other rule.
- **Treating the plugin-binary directory listing as the source of match vocabulary:** the `plugins/` directory only tells you *which plugin types exist* (binary filenames); it never tells you what fields a type's `Match` RPC accepts — that only comes from a live `Describe` call (Pattern 3). Don't try to hardcode a "known plugin types → known fields" table in the kernel or the UI; `docs/plugin-contract.md` is explicit that "the kernel holds no built-in table of known plugin types" ([VERIFIED: `proto/topos/v1/plugin.proto:34`], quoted: `// match_vocabulary is the field-name vocabulary this plugin's Match RPC // reads from MatchRequest.match_fields, declared by the plugin itself — // the kernel holds no built-in table of known plugin types (D-05).`).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|--------------|-----|
| Deterministic map-key ordering for the content-hash lock | A custom sorted-map-to-TOML serializer | `go-toml/v2`'s `Marshal` (already sorts map entries — verified) | Reinventing this risks a subtle non-determinism bug (Go's native map iteration is randomized) that would make D-03's hash lock spuriously reject saves that didn't actually change anything. |
| Atomic file replacement | A custom "write, then check size, then swap" scheme | `os.CreateTemp` (same dir) + `Sync` + `os.Rename` | `os.Rename` within the same filesystem is POSIX-atomic; this is the textbook Go idiom and needs no invention. |
| Modal/dialog/dropdown primitives (focus trap, escape-to-close, ARIA roles) | Hand-rolled `<div>` overlays with manual keydown listeners | shadcn-svelte's `Dialog`/`DropdownMenu` over the already-installed `bits-ui` | `bits-ui` already ships accessible, Svelte-5-native primitives for exactly these components (confirmed present in `node_modules`) — the repo's own `Popover`/`Tooltip` components already follow this pattern; a hand-rolled modal would regress the accessibility this project already has for its other overlays. |
| Trial-launching a plugin subprocess to answer Describe | A separate "lightweight describe-only" subprocess protocol | The existing `pluginhost.launch` → `Describe` → `Kill` sequence, wrapped (Pattern 3) | The go-plugin handshake + gRPC dial + `Describe` call is already exactly what's needed; a second protocol would duplicate `sdk.Handshake`/`sdk.PluginMap` wiring for no benefit. |

**Key insight:** almost everything genuinely new in this phase (deterministic serialization, atomic writes, accessible modals, subprocess handshake) already has a correct, already-integrated implementation sitting one layer below where the phase's own new code needs to call it — the risk here is duplicating existing machinery under a new name, not needing to invent anything.

## Common Pitfalls

### Pitfall 1: Hot-apply has no existing plumbing to build on — plan for new infrastructure, not a small patch
**What goes wrong:** Planning treats D-06 ("save = apply immediately") as "just call `pluginhost.Discover` again," missing that `*config.Config` is currently captured as a fixed pointer at `httpapi.Router(store, cfg, host, host, coord)` construction time ([VERIFIED: `cmd/topos/main.go:201`]) and closed over by every handler — there is no swap point.
**Why it happens:** Every other phase so far has only ever *read* config once at boot; this is the first phase where the config can change while the process is running.
**How to avoid:** Introduce a `config.Store` (atomic-pointer pair) and thread it through `Router`, `MountAgentRoutes`, `correlate.Engine`, and `syncer.Scheduler` in place of the raw `*config.Config` argument — every current call site that reads a field off `cfg` needs to instead call `store.Expanded()` (or equivalent) at request time, not receive a stale pointer once. `syncer.Scheduler.Run` in particular needs a **restart**, not just a config swap: it spawns fixed goroutines from a boot-time snapshot ([VERIFIED: `kernel/syncer/scheduler.go:31-58`], `for name := range s.Config.Sources { ... go func(...) { s.runSource(...) }(...) }`) and has no mechanism to add/remove a source's goroutine after `Run` starts — an apply must cancel the current scheduler's context and start a fresh `Scheduler.Run` against the new config and a freshly-built `Coordinator`.
**Warning signs:** A save that "succeeds" (200 response, file written) but the UI's next `/api/sources` poll still shows the old source set, or a newly-added instance never gets its "eager" first sync (D-07) because nothing told the scheduler it exists.

### Pitfall 2: Serializing the wrong `*config.Config` leaks or destroys secrets
**What goes wrong:** Calling `toml.Marshal` on the expanded runtime config either writes a real secret value into `config.toml` (a privacy breach D-05 calls "one-way in effect, cannot be un-shipped") or, if the writer instead blanks `Token`/`BaseURL` fields defensively, silently destroys the user's `${VAR}` reference on their next save.
**Why it happens:** There is currently only one `*config.Config` in the codebase's mental model; this phase introduces a second one (raw/unexpanded) for the first time, and it's easy to reach for "the config I already have in scope" in a handler.
**How to avoid:** Make the type system help: keep the raw and expanded configs as genuinely distinct values returned from `Load`/held in the `Store` (Pattern 1), and never pass the expanded one to the writer. Add a test that round-trips a config with a `${VAR}` token through save and asserts the on-disk bytes still contain the literal `${VAR}` string, never the environment's real value.
**Warning signs:** A saved `config.toml` that differs from what the user typed in a secret field — either a real value appearing, or the `${VAR}` reference disappearing.

### Pitfall 3: `Source` struct's TOML tags aren't minimal-output-safe for every source shape
**What goes wrong:** `config.Source.BaseURL`/`Token`/`APIVersion` have **no `omitempty` tag** ([VERIFIED: `kernel/config/types.go:41-44`], `BaseURL string `toml:"base_url"`` / `Token string `toml:"token"`` / `APIVersion string `toml:"api_version"``, no `omitempty` on any of the three) — a canonical rewrite of a local-path source like Signal (which legitimately has empty `BaseURL`/`Token`) will emit `base_url = ""` / `token = ""` / `api_version = ""` into the "clean, minimal" file D-02 promises, which reads as broken/confusing to a hand-editing user even though it's harmless to the parser.
**Why it happens:** These fields were written for the always-network-source case (Phase 1-3); the local-path source (Signal) shape was added later (Phase 4) without revisiting the omitempty tags, since nothing serialized the struct back to TOML before this phase.
**How to avoid:** Add `omitempty` to `BaseURL`, `Token`, and `APIVersion` (safe — a *required* network source's Validate check already fails loudly on an empty `base_url`/`token` regardless of whether the tag omits a zero value from serialization) before building the writer, or build a dedicated output-shaping step that only emits fields relevant to that source's shape.
**Warning signs:** A round-tripped Signal (or future local-path source) config block growing three blank/empty keys the user never typed.

### Pitfall 4: Describe-before-persist assumption is unverified for 3 of 4 non-Signal plugin types
**What goes wrong:** Pattern 3's trial-launch approach assumes every plugin type can answer `Describe` from just-submitted, not-yet-verified connection fields — verified for Signal (`path` need only be non-empty), but paperless/silverbullet/proton's `main.go` were not read this session. If any of them resolve/validate connectivity eagerly at process startup (before serving any RPC), a trial-launch with placeholder-but-unverified credentials would fail before ever reaching `Describe`, breaking D-11's two-step modal for that plugin type.
**Why it happens:** Signal was the only plugin whose `main.go` this research session actually opened; the pattern was generalized from one example.
**How to avoid:** During planning or Task 1, read `plugins/paperless/main.go`, `plugins/silverbullet/main.go`, and `plugins/proton/main.go`'s startup sequence to confirm each defers any live connectivity check past `Describe`. If one doesn't, the fallback is: trial-launch anyway and treat a launch failure as "cannot determine match vocabulary yet — save connection config first, edit again to add match fields" — a two-request flow instead of one.
**Warning signs:** The "New <plugin type>…" step-2 modal failing to populate for a specific plugin type with a connection/timeout error instead of a match-vocabulary form.

### Pitfall 5: No existing `index.Store` method removes a source instance's rows
**What goes wrong:** D-07 requires "a removed instance's plugin process shuts down and its index rows are removed right away" — but every existing `Store` method either *replaces* rows scoped to one `(webspace, source)` pair (`ReplaceWebspaceSourceItems`) or never deletes at all. There is no `DELETE FROM items WHERE source = ?`-shaped method today ([VERIFIED: `grep -n "^func (s \*Store)" kernel/index/store.go`] lists 14 methods, none named `Delete*` or matching this shape).
**Why it happens:** Every prior phase only ever added or replaced index rows; removing a *configured source itself* (as opposed to items that no longer match a webspace) has never happened before this phase.
**How to avoid:** Add `Store.DeleteSourceItems(ctx, source string) error` doing `DELETE FROM items WHERE source = ?` — `webspace_items.item_id` already has `ON DELETE CASCADE` back to `items(id)` ([VERIFIED: `kernel/index/schema.go:47-51`], `item_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE`), and the `items_ad` trigger already keeps `items_fts` in sync on any `items` delete ([VERIFIED: `kernel/index/schema.go:99-101`]) — so one statement correctly cleans `items`, cascades to `webspace_items` across every webspace the instance participated in, and keeps FTS consistent, with no new triggers needed. Also consider clearing that source's `sync_runs` history (no FK there, but leaving stale rows around means a removed-then-re-added instance under the same name would show phantom old history).
**Warning signs:** Deleting a source instance from the builder, but its items/chip/history lingering in the UI until a manual restart.

### Pitfall 6: `npx shadcn-svelte add` will not match this repo's already-drifted `components.json`
**What goes wrong:** Running the shadcn-svelte CLI to scaffold `Dialog`/`DropdownMenu` may pull the live registry's current `baseColor`/`style` preset scheme, which already doesn't match what `components.json` in this repo declares — this exact drift was already hit and worked around in a prior phase.
**Why it happens:** Documented precedent: "shadcn-svelte's live CLI/registry retired baseColor slate and style new-york in favor of an encoded theme-preset system; components.json still records the plan's contract values and every actual color is hand-authored in src/app.css from UI-SPEC hex tokens" ([VERIFIED: STATE.md's Decisions log, `[Roadmap]` section, this exact sentence]).
**How to avoid:** Use the CLI output only as a structural starting point (component API shape — `Dialog.Root`/`Dialog.Trigger`/`Dialog.Content` etc.) and hand-adapt colors/classes to the project's existing `app.css` tokens, exactly as every other file under `web/src/lib/components/ui/` already was.
**Warning signs:** A newly-added modal visually clashing with the rest of the app's dark theme (unstyled or wrong-palette borders/backgrounds) immediately after `npx shadcn-svelte add`.

## Code Examples

### Env-var set/unset badge (D-15)
```go
// Straightforward stdlib check — no library needed. Exposed as part of
// the config-read response (e.g. one boolean per referenced ${VAR} name
// across the raw config), never the value itself.
func envVarIsSet(name string) bool {
	_, ok := os.LookupEnv(name)
	return ok
}
```

### Content-hash optimistic lock (D-03)
```go
// Source: standard library composition — sha256 over the raw file bytes,
// recomputed at pre-save re-check time and compared to the hash recorded
// at the Store's last successful Load/Save/Reload.
func fileHash(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (s *Store) Save(ctx context.Context, mutate func(*config.Config) error) error {
	current, err := os.ReadFile(s.path)
	if err != nil {
		return fmt.Errorf("config: re-read before save: %w", err)
	}
	if fileHash(current) != s.lastKnownHash {
		return ErrConfigChangedOnDisk // D-03: reject, kernel reloads the newer file, UI refreshes
	}
	// ... apply mutate() to a copy of the raw struct, dry-run validate the
	// resulting expanded struct, write canonically, Swap() the atomic
	// pointers, update s.lastKnownHash to the new file's hash ...
	return nil
}
```

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|----------------|
| A1 | Full plugin-host rebuild (kill-all/relaunch-all) is an acceptable MVP-mode implementation of D-06/D-07's "eager reconcile," rather than a surgical add/remove/restart-changed diff | Architecture Patterns, Alternatives Considered | If the user's real deployment has 5+ sources and expects a one-field connection-detail edit on one source to leave every other source's subprocess and reachability undisturbed, a full rebuild introduces a visible (sub-second) blip on unrelated sources' chips on every save — annoying but not data-lossy. Confirm during planning whether this tradeoff is acceptable for MVP or whether the diff is worth building now. |
| A2 | Every plugin type's `main.go` defers live connectivity checks past process startup (verified only for Signal), so Pattern 3's trial-launch-before-persist sequencing works uniformly for the "New plugin type" two-step modal | Architecture Patterns, Pattern 3 / Pitfall 4 | If paperless/silverbullet/proton eagerly validate connectivity at startup, the two-step modal's step 2 (match config) can't populate from a not-yet-verified step 1 for those plugin types — needs a two-request fallback flow instead of one trial-launch. |
| A3 | The `plugins/mock` reference plugin should be excluded from the "+"-picker's "New <plugin type>…" list in the shipped UI (it's a test/reference fixture, not a real source a user would knowingly add) | Don't Hand-Roll / Open Questions | If included, a user could accidentally "add" the mock source in a real deployment; low impact (it produces fixed fake items) but confusing. Needs an explicit decision — see Open Questions. |
| A4 | `Webspace.Keywords` and `AgentGrant`'s lack of `omitempty` tags (unlike most other `Source`/`Webspace` fields) is acceptable canonical-rewrite noise (`keywords = []`, an always-present `[sources.<id>.agent]` block even when never configured) rather than something the writer must special-case away | Common Pitfalls, Pitfall 3 | If left as-is, a canonical rewrite of a match-blocks-only webspace (no keywords fallback) emits a spurious `keywords = []` line, and every source gains a previously-absent `[sources.<id>.agent]\nread = false\nhandoff = false` block on its first UI save — cosmetically different from hand-authored files but not functionally wrong (both parse identically). Low risk; flagged for the planner's judgment call, not a blocker. |

## Open Questions

1. **Should `plugins/mock` be offered in the "New <plugin type>…" picker?**
   - What we know: it's discovered by the exact same `topos-plugin-<type>` binary-listing convention as every real plugin, and `config.example.toml` itself treats it as a legitimate (if optional, commented-out) source a user can enable.
   - What's unclear: whether a real desktop deployment's `plugins/` directory even contains `topos-plugin-mock` in a released build, or whether it's a dev-only artifact of `make plugins`.
   - Recommendation: confirm whether `make build`'s distributed artifact includes the mock binary; if it does, either filter it out of the picker by a hardcoded exception or leave it in (it's harmless — fixed fake items, never a real data leak) and let the planner decide based on release packaging, not this phase's UI code.

2. **Does D-06's "no restart" extend to the scheduler's goroutines mid-flight during an apply?**
   - What we know: D-07 says a removed instance's plugin process "shuts down... right away" and a changed instance "immediately re-syncs" — implying in-flight sync handling during apply needs *some* answer.
   - What's unclear: whether an apply that removes a source currently mid-sync should let that sync finish and then tear down, or cancel it immediately. `syncer.Coordinator.syncOne` already detaches its `sync_runs` finalize write from the calling context specifically to survive a cancelled context cleanly ([VERIFIED: `kernel/syncer/coordinator.go:191-197`]) — this existing detach-on-finalize behavior is likely sufficient (a cancelled in-flight sync still records "error"/whatever partial outcome cleanly rather than leaving a stuck "running" row), but this hasn't been exercised against a live-apply-during-sync scenario.
   - Recommendation: this is explicitly named as Claude's Discretion in 07-CONTEXT.md ("in-flight sync handling during apply") — the planner should pick a concrete answer (recommend: cancel the old scheduler's context immediately on apply, let `syncOne`'s existing detached-finalize logic handle any sync that was mid-flight, same as kernel shutdown already does) rather than leave it implicit.

## Environment Availability

No new external tools, services, or runtimes are introduced by this phase — it is built entirely on the kernel's own filesystem (`config.toml`), the already-running plugin subprocess mechanism, and the already-installed frontend toolchain. Skipping the full dependency-availability table as not applicable.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|----------------|---------|--------------------|
| V1 Architecture | yes | No change to the loopback-only, no-auth posture (explicitly locked unchanged this phase per Claude's Discretion in 07-CONTEXT.md) — the new mutating routes live under the same `/api/*` boundary as every existing route, with the same "reachable only from this machine" trust model documented in `docs/api.md`. |
| V4 Access Control | yes | The config-write surface is scoped to configuration only — it must never expose a path to mutate source data (success criterion 4). Enforce this structurally: the new `kernel/httpapi/config.go` handlers must only ever call into `kernel/config`/`kernel/pluginhost` lifecycle methods, never a plugin's `Match`/`Fetch` RPCs with attacker-controlled parameters beyond what `Describe`/trial-launch already need. |
| V5 Input Validation | yes | Reuse `config.Validate` + `pluginhost.ValidateMatchConfig` verbatim as the save dry-run (D-09) — no second, UI-facing validation implementation to keep in sync or accidentally under-validate. |
| V6 Cryptography | n/a | No cryptographic operation is added this phase — secrets are handled as opaque `${VAR}` references, never encrypted/decrypted by the kernel itself. |
| V7 Error Handling / Logging | yes | `Scheduler.refreshAndLog` already establishes the house convention of logging only names/error strings, never config/secret content ([VERIFIED: `kernel/syncer/scheduler.go:81-91`], comment: "the source name and error string only, never the source's config, its token, or any item content") — the new config-save/reload logging must follow the identical discipline: log which webspace/instance changed, never a diff containing a `${VAR}`-resolved value (moot here since the kernel never resolves it) or a raw config blob. |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|-----------------------|
| Concurrent save race (two browser tabs, or the UI vs. a hand-edit-in-progress) corrupting `config.toml` | Tampering | D-03's content-hash optimistic lock, already specified — reject-and-reload rather than merge, no torn-write window because writes are atomic-rename (Pattern 2). |
| A malicious or buggy plugin binary placed in `plugins/` and picked up by the "+"-picker's directory listing | Spoofing / Elevation of Privilege | Out of scope for this phase's own threat model — plugin binary trust is an existing, unchanged assumption (the kernel already executes anything named `topos-plugin-*` in that directory, per `pluginhost.launch`); this phase does not change who can place a binary there or weaken any existing check. Flag explicitly in the plan as "not addressed here" rather than silently assuming it's covered. |
| Config write path used to smuggle a mutation into source data (via a crafted match block or instance name) | Elevation of Privilege | Structural: the write path only ever produces a `config.Config` value that flows through the *exact same* `config.Validate`/`pluginhost.ValidateMatchConfig`/`pluginhost.launch` path a hand-edited file already goes through — there is no new code path from "UI request" to "plugin RPC call" that bypasses existing validation. |
| Secret value leakage via the config-read API response | Information Disclosure | D-05 (hard requirement) — the config-read endpoint must serialize the raw (unexpanded) struct, never the expanded one, exactly like the writer (Pattern 1) — add a shared serializer/redactor used by both the GET response and the write path so there is only one place secrets could ever leak from. |

## Sources

### Primary (HIGH confidence — read directly this session)
- `kernel/config/types.go`, `kernel/config/config.go` — full config model + existing Load/Validate implementation
- `kernel/httpapi/routes.go`, `sources.go`, `webspaces.go`, `stream.go`, `search.go`, `agent.go` — existing HTTP surface conventions, envelope shape, cfg-pointer plumbing
- `kernel/pluginhost/host.go`, `matchconfig.go` — plugin discovery/launch/Describe/Kill lifecycle, post-launch match validation
- `kernel/syncer/coordinator.go`, `scheduler.go` — single-flight coordinator, fixed-at-boot scheduler goroutines
- `kernel/index/store.go`, `schema.go` — FTS5 search query construction, schema/triggers, full method inventory
- `cmd/topos/main.go` — boot sequence, where `*config.Config` is captured as a fixed pointer
- `docs/api.md`, `docs/plugin-contract.md`, `config.example.toml`, `proto/topos/v1/plugin.proto` — published contracts
- `web/src/lib/components/WebspaceHeader.svelte`, `SourceChip.svelte`, `SearchBox.svelte`, `SearchResults.svelte`, `api.ts`, `routes/+page.svelte`, `routes/+layout.svelte`, `routes/w/[webspace]/+page.svelte` — frontend integration points
- `web/package.json`, `web/components.json`, `web/node_modules/bits-ui/package.json` + bundled component listing — frontend dependency surface
- `/opt/go/pkg/mod/github.com/pelletier/go-toml/v2@v2.4.3/marshaler.go` — verified map-key-sort determinism in `Marshal`
- `plugins/signal/main.go` — verified startup connectivity-deferral for the trial-launch pattern
- `.planning/phases/07-webspace-builder-ui/07-CONTEXT.md`, `.planning/REQUIREMENTS.md`, `.planning/STATE.md`

### Secondary (MEDIUM confidence)
- WebSearch, shadcn-svelte official docs (`shadcn-svelte.com/docs/components/dialog`, `.../dropdown-menu`) — component API shape (`Dialog.Root`/`Trigger`/`Content`, child-snippet pattern) confirmed consistent with this repo's existing `Popover` usage.

### Tertiary (LOW confidence)
- None used as authoritative for any locked recommendation above; every claim tagged `[ASSUMED]` in-line is called out in the Assumptions Log.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new packages; every claim verified against files already in this repo.
- Architecture (hot-apply plumbing, dual-parse config, filter query builder): MEDIUM — grounded in direct reading of the current implementation, but the specific new-code shapes proposed (`config.Store`, `pluginhost.Reconcile`, `DescribePluginType`) are original design composition for this research, not verified against any existing prototype — the planner should treat these as strong starting points, not gospel.
- Pitfalls: HIGH for pitfalls 1, 2, 3, 5, 6 (each grounded in a specific verified file/line); MEDIUM for pitfall 4 (verified for one of four comparable plugins only — flagged as Assumption A2).

**Research date:** 2026-08-07
**Valid until:** ~30 days (stable, local codebase — the only external-facing risk is shadcn-svelte registry drift, already flagged as a known, recurring issue for this repo).
