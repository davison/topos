# Phase 5: Source Instances & Per-Type Matching - Pattern Map

**Mapped:** 2026-08-06
**Files analyzed:** 18 (this phase is a rewire of existing files — there are no wholly new files except possibly a small validation helper)
**Analogs found:** 18 / 18 (every file's analog is itself — this phase edits established code in place; "analog" below means the pattern within the SAME file to preserve/extend, since no sibling equivalent exists elsewhere in the repo)

## Framing note for the planner

Unlike a typical phase, Phase 5 creates almost no new files — it rewires ~11 kernel Go files, 5 plugin files, proto, 3 docs, and `config.example.toml` (per RESEARCH.md's Task Ordering and Pitfall 3's file inventory). There is no "new controller, copy from old controller" shape here. Instead, each file below is its own analog: the existing code in that file IS the pattern to extend/rename, and RESEARCH.md's "Recommended Task Ordering" (proto → config → pluginhost → correlate/syncer/index → httpapi → plugins → rendition move → docs → migration) is the sequencing backbone plan files should follow. Where a genuinely new construct is needed (e.g. a vocabulary-validation pass, a typed match-block decoder, a kernel-side sanitize/wrap module), the closest sibling pattern in the SAME package is cited as the analog.

## File Classification

| File | Role | Data Flow | Analog (same-file pattern to extend, or nearest sibling) | Match Quality |
|------|------|-----------|-----------------------------------------------------------|---------------|
| `proto/topos/v1/plugin.proto` | contract/schema | request-response | itself — `MatchRequest`/`DescribeResponse` messages (lines 16-24) | exact (edit in place) |
| `kernel/config/types.go` | config/model | CRUD (load/validate) | itself — `Source`, `Webspace`, `AgentGrant` structs | exact |
| `kernel/config/config.go` | service (config loader/validator) | request-response (load-time) | itself — `Validate`, `AgentReadGrantedNames`, `expandSourceCACertPathsHome` | exact |
| `kernel/pluginhost/host.go` | service (process supervisor) | event-driven (subprocess lifecycle) + request-response (RPC) | itself — `Discover`, `Plugin`, `SourceTypesByName`, `bySourceType`, `Fetch` | exact |
| `kernel/correlate/correlate.go` | service (sync orchestration) | batch/CRUD | itself — `SyncSource`, `WebspaceResult`, `Source` interface | exact |
| `kernel/syncer/coordinator.go` | service (coordinator) | request-response + single-flight | itself — `Coordinator`, `syncOne`, `RunResult` | exact |
| `kernel/index/schema.go` | migration/model (SQLite schema) | CRUD | itself — `items`, `sync_runs` table DDL, FTS5 triggers | exact |
| `kernel/index/store.go` | service (data access) | CRUD | `LatestSyncRunPerSource`/`SyncingSourceTypes` raw SQL (lines ~587, 628 per RESEARCH.md) | exact |
| `kernel/httpapi/item.go` | controller (HTTP handler) + rendition boundary | request-response + file-I/O (stream bytes) | itself — `ItemHandler`, `renditionHandler`, `Fetcher` interface — also the new home for D-11's wrap/sanitize/theme pipeline | exact |
| `kernel/httpapi/sources.go` | controller | request-response | (not read in full this session — referenced via RESEARCH.md `sourceStatus{Name, SourceType, DisplayName}`) | role-match |
| `kernel/httpapi/agent.go` | controller (grant-filtered mirror) | request-response | itself — `grantedSourceTypes`, `filterRunsByGrant`, `agentItemHandler` | exact |
| `kernel/httpapi/stream.go` | controller | request-response | referenced via RESEARCH.md `streamItem.SourceType` — not read in full | role-match |
| `kernel/item/item.go` | model (domain type) | transform | `item.FromProto(sourceType, protoItem)` — signature changes to accept instance id | exact |
| `plugins/proton/plugin.go`, `plugins/paperless/plugin.go`, `plugins/silverbullet/plugin.go` | plugin (Match/Describe RPC impl) | request-response | `plugins/proton/plugin.go:278-285 matchesAnyKeyword`, `plugins/signal/match.go:49-62 matchesAnyKeyword` | exact |
| `plugins/signal/match.go` | plugin (Match RPC impl, typed) | request-response | itself — `matchesConversation`, `candidateNames`, `eligibleConversations` (D-06 1:1 rule to preserve) | exact |
| `plugins/proton/body.go` | plugin (sanitize+theme, being removed) | transform | itself — `RenderSanitizedEmail`, `themeStyle`, `WrapDocument` (lines 165-303) — content/policy to relocate into kernel | exact (source for kernel's new module) |
| `plugins/silverbullet/render.go` | plugin (sanitize+theme, being removed) | transform | itself — `RenderSanitized`, `themeStyle`, `WrapDocument` (full file, 132 lines) — content/policy to relocate | exact (source for kernel's new module) |
| `plugins/signal/render.go` | plugin (sanitize+theme, being removed) | transform | not fully read this session; same shape as proton/silverbullet per RESEARCH.md ("chat no-styles" policy) | role-match |
| `docs/plugin-contract.md`, `docs/api.md`, `config.example.toml` | docs/config | — | themselves — rewrite last per Task Ordering step 8 | exact |
| `sdk/contract_test.go` | test (RPC allowlist) | — | itself — extend allowlist only if RPC *names* change (they don't; message shapes do) | exact |
| `kernel/httpapi/contract_test.go` | test (AGENT-02 shape pin) | — | itself — `idPattern`/`requiredProvenanceKeys` per RESEARCH.md Pitfall 3 | exact |

## Pattern Assignments

### `kernel/config/types.go` + `kernel/config/config.go` (config loader, CRUD)

**Current shape to extend:**
```go
// kernel/config/types.go:7-14
type Config struct {
	Server    ServerConfig        `toml:"server"`
	Index     IndexConfig         `toml:"index"`
	Plugins   PluginsConfig       `toml:"plugins"`
	Sync      SyncConfig          `toml:"sync"`
	Sources   map[string]Source   `toml:"sources"`
	Webspaces map[string]Webspace `toml:"webspaces"`
}

// kernel/config/types.go:134-138 — TO BE REPLACED per D-01/D-02/D-03
type Webspace struct {
	Keywords []string `toml:"keywords"`
}
```
`Webspace` becomes (per D-01/D-02/D-03): `Keywords []string` (fallback, optional), `Sources []string` (allowlist, optional), `Match map[string]MatchBlock` (per-instance typed blocks) where `MatchBlock` is likely `map[string]interface{}` or `map[string][]string` per RESEARCH.md's "Standard Stack" section (go-toml/v2 dynamic-key decode — no new dependency).

**display_name uniqueness pattern to follow (D-09)** — model on the existing loud-validation style in `Validate`:
```go
// kernel/config/config.go:167-180 — this IS the validation idiom to copy
func (cfg *Config) Validate(missing []string) error {
	for name, ws := range cfg.Webspaces {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("config: webspace has empty name")
		}
		if len(ws.Keywords) == 0 {
			return fmt.Errorf("config: webspace %q declares zero keywords", name)
		}
		for _, kw := range ws.Keywords {
			if strings.TrimSpace(kw) == "" {
				return fmt.Errorf("config: webspace %q declares an empty or whitespace-only keyword", name)
			}
		}
	}
	...
```
Every new validation rule (display_name uniqueness D-09, unknown match field D-05, explicit-block-replaces-fallback D-02) should follow this exact shape: `fmt.Errorf("config: <subject> %q <problem>", name, ...)`, one rule per loop, fail on first violation, name both the field and the source/webspace.

**Grant-lookup pattern to mirror for instance-keyed grants (D-10 — already instance-shaped, no change needed to this specific function, but its *shape* is the template for any similar per-instance query):**
```go
// kernel/config/config.go:118-126
func (cfg *Config) AgentReadGrantedNames() map[string]bool {
	granted := map[string]bool{}
	for name, src := range cfg.Sources {
		if src.Agent.Read {
			granted[name] = true
		}
	}
	return granted
}
```

**Env-var/home-expansion iteration pattern** (for any new per-source or per-match-block field needing similar treatment):
```go
// kernel/config/config.go:147-160
func (cfg *Config) expandSourceCACertPathsHome() error {
	for name, src := range cfg.Sources {
		if !strings.HasPrefix(src.CACert, "~") {
			continue
		}
		u, err := user.Current()
		if err != nil {
			return fmt.Errorf("config: resolve home directory for source %q ca_cert: %w", name, err)
		}
		src.CACert = strings.Replace(src.CACert, "~", u.HomeDir, 1)
		cfg.Sources[name] = src
	}
	return nil
}
```

**Two-phase validate needed (Pitfall 1):** `config.Load`/`Validate` stays plugin-independent (structural checks only); a NEW second-pass function (e.g. `pluginhost.ValidateMatchVocabulary(cfg, host)`) runs after `Discover` returns each plugin's Describe-reported vocabulary, called from `cmd/topos/main.go` between `pluginhost.Discover` and the first sync. Model its error style on `Validate`'s `fmt.Errorf("config: ...")` convention but scope it under `pluginhost` since it needs `*Host`.

---

### `proto/topos/v1/plugin.proto` (contract/schema)

**Fields being replaced:**
```protobuf
// lines 16-24 — DescribeResponse gains match_vocabulary; MatchRequest is replaced
message DescribeResponse {
  string source_type      = 1;  // "paperless"
  string display_name     = 2;  // "paperless-ngx"
  string contract_version = 3;  // "topos.v1"
}

message MatchRequest  { repeated string keywords = 1; }  // exact, case-insensitive (D-03)
message MatchResponse { repeated Item   items    = 1; }
```
Per RESEARCH.md Pitfall 2 (Assumption A2), the recommended shape is a generic `map<string, StringList> match_fields` on `MatchRequest` (NOT one field per plugin type — that reintroduces the kernel-side-table anti-pattern D-05 forbids) and `repeated string match_vocabulary = 4;` on `DescribeResponse`. `Item.source_type` (line 35) also needs a documentation-comment update (still plugin kind, not identity, per the two-trust-sources pattern below) — do not rename this proto field, since `FromProto`'s Go-side caller is what re-keys it to instance id (see `kernel/item/item.go` below).

**FetchResponse (D-11) simplification** — drop nothing structurally (still `available`, `mime_type`, `text`, `data`, `provenance`), but add content-shape metadata (e.g. `string content_shape = 8;` — "email"|"chat"|"markdown") so the kernel's new sanitize/wrap module (see item.go below) knows which policy profile to apply; `data` becomes an unwrapped, unthemed HTML *fragment* for html-shaped content instead of a full themed document.

---

### `kernel/pluginhost/host.go` (process supervisor — instance identity source)

**The "two trust sources" pattern (already exists, just needs its usage swept — this is the single most load-bearing existing pattern in the phase):**
```go
// kernel/pluginhost/host.go:41-63 — already separates the two concepts
type Plugin struct {
	name        string // config key under [sources.<name>]  <- becomes THE identity (D-08)
	sourceType  string // learned via Describe, not trusted from the filename  <- stays "plugin kind" only
	displayName string // learned via Describe
	...
}
func (p *Plugin) Name() string { return p.name }
func (p *Plugin) SourceType() string { return p.sourceType }
```
Every downstream caller that today keys on `SourceType()` for identity purposes (item ids, sync_runs, agent grants, HTTP JSON) must switch to `Name()`/instance id; every caller using `SourceType()` for vocabulary/contract-version/kind purposes keeps it. Do not merge these fields.

**Discover loop — confirmed no structural change needed for multi-instance launch:**
```go
// kernel/pluginhost/host.go:93-106 (VERIFIED — copy verbatim, no changes needed here)
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

**`bySourceType` must become `byInstanceID` (or similar) — this is the resolution lookup `Fetch` depends on and is currently keyed on plugin kind, which breaks with 2 instances of one plugin type:**
```go
// kernel/pluginhost/host.go:270-314 — Fetch(ctx, sourceType, sourceID, variant) calls bySourceType;
// rename the parameter and the lookup to key on instance id (p.name), not p.sourceType
func (h *Host) bySourceType(sourceType string) *Plugin {
	for _, p := range h.plugins {
		if p.sourceType == sourceType {
			return p
		}
	}
	return nil
}
```

**`SourceTypesByName` is the map `agent.go`'s grant filter depends on — after the rewire this becomes unnecessary indirection (RESEARCH.md's step 5 insight) since index rows will already carry instance id directly:**
```go
// kernel/pluginhost/host.go:194-200 — likely deleted or repurposed once
// grantedSourceTypes (agent.go) can key on instance id directly
func (h *Host) SourceTypesByName() map[string]string {
	out := make(map[string]string, len(h.plugins))
	for _, p := range h.plugins {
		out[p.name] = p.sourceType
	}
	return out
}
```

---

### `kernel/correlate/correlate.go` (sync orchestration, batch)

**`Source` interface and `SyncSource` — both need identity-field changes plus the D-01/D-02/D-03 fallback/allowlist/explicit-block resolution logic:**
```go
// kernel/correlate/correlate.go:27-31 — Match signature changes from
// keywords []string to typed match fields (map[string][]string, per Pitfall 2)
type Source interface {
	Name() string
	SourceType() string
	Match(ctx context.Context, keywords []string) (*toposv1.MatchResponse, error)
}

// kernel/correlate/correlate.go:77-114 — SyncSource's per-webspace loop is
// the exact place D-01/D-02/D-03 resolution logic lands: for each webspace,
// resolve src's match block (explicit block replaces fallback per D-02),
// or fall back to ws.Keywords as native-name matching, or skip entirely if
// a sources allowlist (D-03) excludes this instance.
func (e *Engine) SyncSource(ctx context.Context, src Source) (results []WebspaceResult, rejections string) {
	for name, ws := range e.Config.Webspaces {
		resp, err := src.Match(ctx, ws.Keywords) // <- becomes typed match-field resolution
		...
		it := item.FromProto(src.SourceType(), protoItem) // <- becomes src.Name() (instance id)
		...
		e.Store.ReplaceWebspaceSourceItems(ctx, name, src.SourceType(), items) // <- becomes src.Name()
	}
}
```
`WebspaceResult.SourceType` (line 52) should rename to `WebspaceResult.Source` (instance id) per RESEARCH.md's explicit recommendation to rename, not silently redefine.

---

### `kernel/syncer/coordinator.go` (single-flight coordinator)

**`syncOne` and `RunResult` — coordinator already keys singleflight on `Name()` (correct), but `RunResult.SourceType`/`sync_runs` writes key on `SourceType()` (needs to become instance id):**
```go
// kernel/syncer/coordinator.go:70-79 — already correct pattern for instance keying
func NewCoordinator(store *index.Store, engine *correlate.Engine, sources []correlate.Source) *Coordinator {
	byName := make(map[string]correlate.Source, len(sources))
	for _, s := range sources {
		byName[s.Name()] = s
	}
	return &Coordinator{store: store, engine: engine, sources: byName}
}

// kernel/syncer/coordinator.go:137-141 — sourceType here becomes instance id (src.Name())
func (c *Coordinator) syncOne(ctx context.Context, src correlate.Source) RunResult {
	sourceType := src.SourceType() // <- change to src.Name()
	started := time.Now().Unix()
	runID, err := c.store.StartSyncRun(ctx, sourceType)
	...
```
`RunResult.SourceType` (line 46) — same rename-not-silently-redefine call as `WebspaceResult`.

---

### `kernel/index/schema.go` + `kernel/index/store.go` (SQLite schema/queries, D-07 drop-and-resync)

**Schema comment already documents the composite id shape that must be preserved with new semantics:**
```sql
-- kernel/index/schema.go:18-34
CREATE TABLE IF NOT EXISTS items (
  id                       TEXT PRIMARY KEY,   -- "{source_type}:{source_id}" -> becomes "{instance}:{source_id}"
  source_type              TEXT NOT NULL,      -- holds instance id after rewire (or rename column)
  source_id                TEXT NOT NULL,
  ...
);
CREATE TABLE IF NOT EXISTS sync_runs (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  source_type   TEXT NOT NULL,                 -- holds instance id after rewire
  ...
);
```
No `ALTER TABLE`/migration needed (D-07 — drop and re-sync); a schema-version bump or explicit index-file deletion step is the only "migration" required — do not write a data migration script (RESEARCH.md's explicit anti-pattern warning).

**Raw SQL string literals that must be hand-updated if the column is renamed (Pitfall 4 — these do NOT get touched by a Go-level rename):**
```go
// kernel/index/store.go:587, 628 (not read verbatim this session, but
// RESEARCH.md cites these lines directly) — grep for
// "GROUP BY source_type" in store.go and update the literal SQL text
// separately from any Go struct field rename.
```

---

### `kernel/httpapi/item.go` (controller + D-11's new rendition-boundary home)

**Existing Fetch/rendition pattern — the CSP header-setting idiom to preserve verbatim (already the hardened boundary D-11 builds on):**
```go
// kernel/httpapi/item.go:147-213 renditionHandler — copy this exact structure
// for the new wrap/sanitize/theme pipeline: resolve item -> fetch -> check
// MIME allowlist -> set hardened headers -> stream body. The NEW step D-11
// inserts is BETWEEN fetch and stream: sanitize(content, contentShape) then
// wrap(sanitized) using a content-shape-keyed policy table built from the
// three plugin sanitizer policies below.
h := w.Header()
h.Set("Content-Type", result.MimeType)
h.Set("X-Content-Type-Options", "nosniff")
h.Set("Content-Disposition", "inline")
h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; object-src 'none'; sandbox")
h.Set("Cache-Control", "private, no-store")
```
**Fetcher interface signature change:** `Fetch(ctx, sourceType, sourceID string, variant)` (line 26) — `sourceType` param becomes instance id; same change propagates to `pluginhost.Host.Fetch`.

---

### `kernel/httpapi/agent.go` (grant-filtered mirror controller)

**Grant-filter pattern — becomes simpler post-rewire per RESEARCH.md step 5 (no `byName` indirection needed once index rows carry instance id directly):**
```go
// kernel/httpapi/agent.go:40-54 — current two-hop indirection
func grantedSourceTypes(cfg *config.Config, byName map[string]string) map[string]bool {
	granted := map[string]bool{}
	for name := range cfg.AgentReadGrantedNames() {
		if st, ok := byName[name]; ok {
			granted[st] = true
		}
	}
	return granted
}
```
Post-rewire this collapses to `return cfg.AgentReadGrantedNames()` directly (config name IS the identity items now carry), eliminating the `byName`/`prober.SourceTypesByName()` dependency throughout this file. Every `it.SourceType`/`s.SourceType` comparison against `granted[...]` (lines 110, 138, 226, 263, 313) keeps its exact shape — only what's stored in that field changes semantics.

**Security-critical test to add (per RESEARCH.md's Security Domain table) — model on this file's existing not-found-indistinguishability pattern:**
```go
// kernel/httpapi/agent.go:240-247 — this exact pattern (never a distinct
// code for "exists but ungranted") extends to: two instances of the same
// plugin type, one granted, one not — assert the ungranted instance's
// items never leak through any /agent/v1/* response.
func agentItemNotFound(w http.ResponseWriter, id string) {
	WriteError(w, http.StatusNotFound, "item_not_found", "item \""+id+"\" was not found in the index")
}
```

---

### `plugins/signal/match.go` (typed match-field plugin implementation to preserve)

**D-06 1:1 matching rule — this logic is explicitly locked and must survive the `keywords []string` -> typed `conversations []string` field rename unchanged:**
```go
// plugins/signal/match.go:88-110 — candidateNames: preserve exactly.
// Only the caller's input source changes (typed "conversations" field
// instead of shared "keywords"); this function's body does not change.
func candidateNames(c conversation) []string {
	if c.IsNoteToSelf {
		return nil
	}
	switch c.Type {
	case "group":
		if c.Name == "" {
			return nil
		}
		return []string{c.Name}
	case "private":
		var out []string
		if nickname := joinName(c.NicknameGivenName, c.NicknameFamilyName); nickname != "" {
			out = append(out, nickname)
		}
		if system := joinName(c.SystemGivenName, c.SystemFamilyName); system != "" {
			out = append(out, system)
		}
		return out
	default:
		return nil
	}
}
```

**Exact-match comparison — every plugin's typed matcher keeps this unchanged (only the field name of its input changes):**
```go
// plugins/signal/match.go:49-62 — identical pattern in
// plugins/proton/plugin.go:278-285 (matchesAnyKeyword)
func matchesAnyKeyword(name string, keywords []string) bool {
	if name == "" {
		return false
	}
	for _, kw := range keywords {
		if strings.EqualFold(name, kw) {
			return true
		}
	}
	return false
}
```

---

### D-11 Rendition centralization: `plugins/proton/body.go` + `plugins/silverbullet/render.go` (source for kernel's new sanitize/wrap module)

**These two files are near-identical in shape (confirmed this session — `WrapDocument` is copied verbatim between them per body.go's own comment "copied verbatim in shape from plugins/silverbullet/render.go"). This IS the pattern the kernel's new `kernel/httpapi` sanitize/wrap module should be built from — literally relocate, don't redesign:**

```go
// plugins/silverbullet/render.go:1-38 — the two-layer sanitize pattern
// (goldmark defaults + bluemonday.UGCPolicy) to become the "markdown"
// content-shape profile in the kernel's new policy table
var mdConverter = goldmark.New()
var sanitizePolicy = bluemonday.UGCPolicy()

func RenderSanitized(markdown []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := mdConverter.Convert(markdown, &buf); err != nil {
		return nil, err
	}
	return sanitizePolicy.SanitizeBytes(buf.Bytes()), nil
}
```

```go
// plugins/proton/body.go:165-179 — the "email" content-shape profile
// (narrow style-allowlist) to become a second entry in the kernel's policy table
var emailSanitizePolicy = newEmailSanitizePolicy()
func newEmailSanitizePolicy() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("style").OnElements(styledElements...)
	p.AllowStyles(
		"color", "background-color", "font-weight", "font-style", "font-size",
		"font-family", "text-align", "text-decoration", "padding", "margin",
		"border", "width", "height",
	).OnElements(styledElements...)
	return p
}
```

**`WrapDocument` — identical between both files; becomes ONE function in the kernel, called with a per-content-shape stylesheet variant (or one shared stylesheet, per CONTEXT.md's discretion note on staying in sync with `web/src/app.css`):**
```go
// plugins/silverbullet/render.go:115-132 (byte-identical shape in
// plugins/proton/body.go:288-303) — this IS the kernel-side wrap function,
// relocate verbatim, parameterize themeStyle if per-shape variants are needed
func WrapDocument(sanitizedFragment []byte) []byte {
	var buf bytes.Buffer
	buf.WriteString("<!doctype html>\n<html><head><meta charset=\"utf-8\"><style>")
	buf.WriteString(themeStyle)
	buf.WriteString("</style></head><body>")
	buf.Write(sanitizedFragment)
	buf.WriteString("</body></html>")
	return buf.Bytes()
}
```

**Scrollbar CSS block (260805-j98) — MUST be relocated intact, not dropped, into the kernel-owned stylesheet (both files carry byte-identical copies, lines 197-224 of body.go / lines 59-86 of render.go) — their corresponding render tests (`TestRenderSanitizedEmail_...`, scrollbar assertions) must move to the kernel's new test file, not be deleted.**

**`img { display: none !important; }` (proton, line 255) vs `img { max-width: 100%; }` (silverbullet, line 110) — this is a genuine per-content-shape divergence (email hides images entirely for tracking-pixel defense; markdown allows them) — the kernel's policy table must preserve this per-shape distinction, not collapse to one shared rule.**

---

## Shared Patterns

### Identity rename sweep (the phase's central mechanical pattern)
**Source:** `kernel/pluginhost/host.go:41-63` (the `name`/`sourceType` two-field split already exists)
**Apply to:** Every file in the 10-file sweep RESEARCH.md Pitfall 3 enumerates: `kernel/syncer/coordinator.go`, `kernel/index/schema.go`, `kernel/index/store.go`, `kernel/correlate/correlate.go`, `kernel/httpapi/agent.go`, `kernel/httpapi/item.go`, `kernel/httpapi/sources.go`, `kernel/httpapi/stream.go`, `kernel/pluginhost/host.go`, `kernel/item/item.go`. Everywhere `SourceType()`/`source_type` means "which config entry" → swap to `Name()`/instance id. Everywhere it means "which plugin binary/kind" → keep as-is. Never let one function signature carry both meanings under one parameter name without a doc comment (RESEARCH.md's explicit warning sign).

### Loud config validation error style
**Source:** `kernel/config/config.go:167-215` (`Validate`)
**Apply to:** All new D-05/D-09/D-02 validation rules — `fmt.Errorf("config: <subject> %q <problem>%s", name, ..., missingSuffix(missing))`, fail on first violation encountered, name the offending field/plugin explicitly per D-05's "naming the field and the plugin" requirement.

### Grant-filter-then-reuse-sibling-handler pattern
**Source:** `kernel/httpapi/agent.go` (entire file, especially `agentItemHandler`/`agentRenditionHandler` mirroring `item.go`'s `ItemHandler`/`renditionHandler`)
**Apply to:** No new agent routes needed this phase, but any handler touched by the identity rewire must re-verify its grant check still uses `agentItemNotFound`'s exact code/status/message (never a distinct "ungranted" code) — this is a security invariant (T-02-20), not a style preference.

### Two-layer sanitize (parse/convert, then bluemonday) for the kernel's new rendition boundary
**Source:** `plugins/silverbullet/render.go:10-38`, `plugins/proton/body.go:150-187`
**Apply to:** `kernel/httpapi/item.go`'s new sanitize/wrap module (D-11) — build a `map[string]*bluemonday.Policy` keyed by content-shape ("email", "chat", "markdown"), each policy body copied from the corresponding plugin file, sanitize BEFORE wrap, wrap AFTER sanitize (never re-run wrapped output back through the sanitizer).

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `pluginhost.ValidateMatchVocabulary` (new function, exact name at Claude's discretion) | service (config/plugin cross-validation) | request-response (post-launch, pre-sync) | No existing second-pass, post-launch validation function exists in the codebase — `config.Validate` is the only validation precedent and is explicitly plugin-independent by design (Pitfall 1). Plan should design this as new code following `config.Validate`'s error-message conventions but living in (or called from) `pluginhost` or `cmd/topos/main.go`, since it needs `*Host`. |
| Kernel-owned unified content-shape policy table / sanitize-wrap module (new file, likely `kernel/httpapi/rendition.go` or similar) | service (sanitize/theme boundary) | transform | No kernel-side sanitizer exists today — D-11 is a genuine relocation-and-merge of three plugin-side implementations into one new kernel file; use the "Shared Patterns" two-layer sanitize entry above as its concrete source material. |

## Metadata

**Analog search scope:** `kernel/config`, `kernel/pluginhost`, `kernel/correlate`, `kernel/syncer`, `kernel/index`, `kernel/httpapi`, `kernel/item`, `plugins/{proton,signal,silverbullet,paperless}`, `proto/topos/v1`, `sdk`
**Files scanned:** 18 read in full or targeted excerpt this session; `kernel/httpapi/sources.go`, `kernel/httpapi/stream.go`, `plugins/signal/render.go`, `plugins/paperless/plugin.go` referenced via RESEARCH.md's verified line citations but not independently re-read (avoiding duplicate reads of content RESEARCH.md already captured verbatim with line numbers).
**Pattern extraction date:** 2026-08-06
