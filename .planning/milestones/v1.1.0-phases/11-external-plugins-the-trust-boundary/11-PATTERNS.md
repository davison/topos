# Phase 11: External Plugins & the Trust Boundary - Pattern Map

**Mapped:** 2026-08-13
**Files analyzed:** 13 (10 modified in place, 3 net-new small files)
**Analogs found:** 13 / 13 (this phase is entirely in-place extension of existing files; RESEARCH.md already identified exact call sites/line numbers, confirmed live against current source this session)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `kernel/pluginhost/discover_binaries.go` (widen: two-tier + `ResolveBinary`) | service (discovery) | file-I/O | itself (prior version) | exact — same file, extend in place |
| `kernel/pluginhost/binaryhash.go` (NEW) | utility | file-I/O | `kernel/config/store.go` `fileHash` (lines 16-22) | exact — identical sha256+hex shape, different input (file bytes vs TOML bytes) |
| `kernel/pluginhost/host.go` (`launch()`, `Discover`, `Reconcile`) | service (process lifecycle) | event-driven / request-response | itself (prior version) | exact — same file, extend in place |
| `kernel/config/types.go` (`PluginsConfig.{ExternalDir,Pins}`, `Source.Extras`) | model | CRUD | itself (prior version) — `Source` struct's existing `omitempty` field convention | exact |
| `kernel/config/config.go` (`Validate`: extras/pin shape checks; `expandEnv` reuse) | service (validation) | transform | itself — `Validate`/`expandEnv`/`collectEnvVarNames` already-existing functions | exact |
| `kernel/httpapi/config.go` (`PluginTypesHandler` tier tag; describe/save extras+pins passthrough) | controller | request-response | itself (prior version) | exact |
| `kernel/httpapi/sources.go` (`SourcesHandler`/`sourceStatusesFrom` merge configured-but-unlaunched) | controller | request-response | itself (prior version) | exact |
| `kernel/supervisor/supervisor.go` (`Apply`: soft-fail pin-mismatch path) | service (orchestration) | event-driven | itself (prior version) | exact |
| `cmd/topos/main.go` (external dir resolver, per-OS default) | config/utility | file-I/O | itself — existing `pluginsDir(cfg)`/`configPath` resolution helpers | role-match (new helper, same file/pattern) |
| `web/src/lib/components/AddSourceModal.svelte` (untrusted interstitial step + extras form) | component | request-response | itself (prior version) — existing two-step `step` state machine (`'existing' \| 'connect' \| 'link' \| 'match' \| ...`) | exact |
| `web/src/lib/components/SourceChip.svelte` (badge overlay + "Trust updated binary" menu item) | component | request-response | itself (prior version) — existing `isWhatsApp`/"Re-link…" conditional menu-item pattern (D-03, 08-04-PLAN.md) | exact |
| `web/src/lib/components/PluginIcon.svelte` (badge overlay integration point) | component | request-response | itself — three-step fallback chain, unchanged, badge composes around it | exact |
| `web/src/lib/api.ts` (`SourceStatus`, `PluginTypesResponse`, `DescribePluginResponse` new fields) | model (type defs) | transform | itself — existing interface blocks, additive fields | exact |
| `web/src/lib/plugin-fields.ts` (tier label helper, extras field-declaration types) | utility | transform | itself — existing `pluginTypeLabel`/`WHATSAPP_PLUGIN_BINARY`-style small helpers | exact |
| `docs/plugin-contract.md` / `docs/api.md` | docs | — | itself | exact |

## Pattern Assignments

### `kernel/pluginhost/binaryhash.go` (NEW utility, file-I/O)

**Analog:** `kernel/config/store.go` lines 16-22 (`fileHash`)

**Exact pattern to mirror** (verified this session, `kernel/config/store.go:16-22`):
```go
// fileHash hex-encodes the SHA-256 digest of raw — the same digest LoadRaw
// computes over a freshly-read file, so Store.Save's re-read-and-compare
// (D-03) is always comparing like with like.
func fileHash(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
```

**New code shape** (streams the file rather than holding it all in memory — binaries can be much larger than config.toml):
```go
package pluginhost

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// HashBinary hex-encodes the SHA-256 digest of the plugin binary at path —
// the value pinned in cfg.Plugins.Pins and re-verified at every launch
// (PLUG-07). Mirrors kernel/config/store.go's fileHash convention exactly,
// so the codebase has one hashing shape, not two.
func HashBinary(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
```

---

### `kernel/pluginhost/discover_binaries.go` (widen to two-tier discovery)

**Analog:** itself — `DiscoverAllBinaries`/`isRegularFileFollowingSymlinks` (full file read this session, 155 lines)

**Existing symlink-following check to reuse verbatim for the external dir** (lines 141-155):
```go
func isRegularFileFollowingSymlinks(dir string, e os.DirEntry) bool {
	if e.Type().IsRegular() {
		return true
	}
	if e.Type()&os.ModeSymlink == 0 {
		return false
	}
	info, err := os.Stat(filepath.Join(dir, e.Name()))
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}
```

**Missing-directory-is-legitimate-empty-state pattern** (lines 111-118, apply identically to the external dir):
```go
func DiscoverAllBinaries(pluginsDir string) ([]string, error) {
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	// ...
}
```

**Existing UI-policy vs security-authority split (preserve across both tiers, Pitfall 4):**
```go
// DiscoverBinaries: UI-POLICY view, excludes ExcludedPluginBinaries
// ("topos-plugin-mock", "topos-plugin-mockstrict") from the "+" picker.
// DiscoverAllBinaries: security-authority raw listing, used by
// DescribePluginHandler's launchable-binary-name check — must span BOTH
// tiers once widened, or an excluded dev fixture becomes launchable via
// the external tier by omission.
var ExcludedPluginBinaries = map[string]bool{
	"topos-plugin-mock":       true,
	"topos-plugin-mockstrict": true,
}
```

**New `Tier` type + `ResolveBinary` shadow-rule resolver** (RESEARCH.md Pattern 1, composes the two functions above, not copied from any existing file — new code):
```go
type Tier string

const (
	TierTrusted  Tier = "trusted"
	TierExternal Tier = "external"
)

// ResolveBinary applies D-11's shadow rule: a name present in both
// directories resolves to the trusted copy, logged loudly (never silent).
func ResolveBinary(trustedDir, externalDir, name string, logger hclog.Logger) (path string, tier Tier, err error) {
	// see RESEARCH.md Pattern 1 for full body
}
```

---

### `kernel/pluginhost/host.go` (`launch()` — pin gate + env allowlist + extras)

**Analog:** itself (prior version) — `launch()` at line 420, current env construction at lines 426-453

**Current flat-map marshal to replace** (verified this session, `kernel/pluginhost/host.go:426`):
```go
sourceConfig, err := json.Marshal(map[string]string{
    "base_url":         src.BaseURL,
    "token":            src.Token,
    "api_version":      src.APIVersion,
    "ca_cert":          src.CACert,
    "username":         src.Username,
    "webmail_base_url": src.WebmailBaseURL,
    "path": src.Path,
})
```

**Current env construction to replace** (verified this session, `kernel/pluginhost/host.go:449`):
```go
env := append(os.Environ(), "WEBSPACES_SOURCE_CONFIG="+string(sourceConfig))
```

**Replacement shapes:** see RESEARCH.md Pattern 3 (`allowedEnv`) and Pattern 4 (nested-struct marshal with `Extras map[string]string \`json:"extras,omitempty"\``). Do not flatten extras keys into the top-level map — Go's type system already rejects this at compile time (a `map[string]string` cannot hold a nested map), but the danger is silently falling back to something that breaks D-12's sub-table boundary.

**Pin-verification gate insertion point:** immediately after `ResolveBinary` returns `TierExternal`, before the existing `os.Stat(binPath)`/`exec.Command(binPath)` lines (RESEARCH.md Pattern 2). On mismatch, return a distinguishable sentinel (e.g. `ErrPinMismatch`), never the same error class `Discover`/`Reconcile` already treat as fatal-to-everything.

**Existing all-or-nothing failure model to soften narrowly (Pitfall 1):**
- `Discover` (`host.go:217`) — calls `h.Shutdown()` and hard-fails kernel boot on first launch failure.
- `Reconcile` (`host.go:261`) — kills what it just launched, hard-fails the whole `PUT /api/config` apply.
- Both must catch `ErrPinMismatch` specifically and treat it as soft/per-instance: skip that instance, record it in a new `Host.launchFailures map[string]string` side-channel under the existing `h.mu`, continue processing every other configured source normally. Every other failure class keeps today's hard-fail behavior unchanged (RESEARCH.md's explicit recommendation, A3/Open Question 1).

---

### `kernel/httpapi/sources.go` (`SourcesHandler`/`sourceStatusesFrom` — merge unlaunched sources)

**Analog:** itself (prior version) — `sourceStatusesFrom` at line 93, `SourcesHandler` at line 76

**Current behavior (verified this session):** builds its response exclusively from `prober.ProbeSources(ctx)`, which iterates `Host.snapshot()` (the launched-plugin list only) — a source that never launched has no entry today.

**Existing `sourceStatus` field shape to extend additively** (`web/src/lib/api.ts:262-272`, mirrored server-side in `kernel/httpapi/sources.go`):
```go
Reachable    bool   `json:"reachable"`
LastError    string `json:"last_error"`
// ... last_status, syncing, last_sync_unix, name, source_type, display_name, plugin
```

**New field to add** (Pitfall 2 — do not gate the "Trust updated binary" menu item on string-matching `last_error`): an explicit machine-readable field, e.g. `PinMismatch bool \`json:"pin_mismatch,omitempty"\`` (or a `launch_failure` enum string), additive with `omitempty` so no `schemaVersion` bump is needed per this repo's own convention (`kernel/httpapi/routes.go:23-25`, `const schemaVersion = 1`, "bump only for breaking JSON shape changes").

**Merge logic:** widen `sourceStatusesFrom` to iterate the configured source set (`cfg.Sources`), not just `ProbeSources`'s launched-plugin result — for any name present in config but absent from the probe result, synthesize a `sourceStatus` entry from `cfg.Sources[name].DisplayName`/instance id (no `Plugin`/`SourceType` available since `Describe` never ran) carrying the new pin-mismatch signal.

---

### `kernel/config/types.go` (`Source.Extras`, `PluginsConfig.{ExternalDir,Pins}`)

**Analog:** itself — the `Source` struct's own established `omitempty` convention (verified this session, full file read)

**Existing field convention to mirror exactly** (`kernel/config/types.go:52-53`):
```go
BaseURL    string `toml:"base_url,omitempty" json:"base_url,omitempty"`
Token      string `toml:"token,omitempty" json:"token,omitempty"`
```

**New fields to add, following the identical shape:**
```go
// Source struct addition:
Extras map[string]string `toml:"extras,omitempty" json:"extras,omitempty"`

// PluginsConfig addition (currently just Dir string, line 33-35):
type PluginsConfig struct {
	Dir         string            `toml:"dir" json:"dir"`
	ExternalDir string            `toml:"external_dir,omitempty" json:"external_dir,omitempty"`
	Pins        map[string]string `toml:"pins,omitempty" json:"pins,omitempty"`
}
```

**Load-bearing precedent for non-nil empty maps** (`kernel/config/types.go:271-280`, `DefaultConfig()`): `Sources`/`Webspaces` are deliberately non-nil empty maps, not nil — apply the same discipline if `Pins`/`Extras` need a default-construction path, since a nil map marshals to an omitted TOML key while an empty non-nil map marshals to an explicit table header.

---

### `web/src/lib/components/SourceChip.svelte` (badge overlay + re-pin menu item)

**Analog:** itself (prior version) — the existing `isWhatsApp`/"Re-link…" conditional menu-item pattern (D-03, 08-04-PLAN.md Task 2), verified this session at lines 1-140

**Exact conditional-menu-item precedent to mirror** (`SourceChip.svelte`, `isWhatsApp` derivation + doc comment):
```svelte
// isWhatsApp gates the Re-link… menu entry (D-03) — keyed on
// source_type, the Describe-reported plugin KIND GET /api/sources
// actually exposes, never on a plugin binary name this component has
// no other reason to know.
let isWhatsApp = $derived(source.source_type === WHATSAPP_SOURCE_TYPE);
```

**Apply identically for "Trust updated binary":** `let isPinMismatch = $derived(source.pin_mismatch === true);` (or equivalent new field name), rendering a fourth/fifth `DropdownMenuItem` only when true — same `onedit(name, 'connection' | 'match' | 'relink' | 'remove')`-style discriminated callback shape, extended with a new kind, e.g. `'trust-update'`.

**Health dot tone table to extend for the new state** (`SourceChip.svelte`):
```ts
const DOT_TONE_CLASS: Record<HealthTone, string> = {
	success: 'bg-success',
	warning: 'bg-warning',
	destructive: 'bg-destructive',
	unknown: 'bg-muted-foreground'
};
```
D-03's "binary changed" state should map to `warning` or `destructive` tone via `healthTone()` (`web/src/lib/format.ts`) — extend `healthTone`'s existing branch logic rather than inventing a parallel tone system.

**Badge glyph overlay integration point:** `PluginIcon.svelte` is rendered inline in the chip's `[dot][icon][name]` layout (per its own doc comment, "09-01-PLAN.md Task 3 adds the plugin's own identity icon... between the health dot and the display name"). D-06's warning glyph overlays this icon — wrap the existing `<PluginIcon plugin={source.plugin} size="size-3.5" />` call in a small relatively-positioned container with an absolutely-positioned warning icon in the corner, gated on `source.tier === 'external'` (new field, from `GET /api/sources`). No pill widening, no chip-wide tinting (D-06, locked).

**Tooltip pattern to extend** (`SourceChip.svelte` `tooltipText` `$derived.by`):
```svelte
let tooltipText = $derived.by(() => {
	if (source.syncing) return `${source.display_name} — syncing…`;
	const relative = formatRelativeTime(source.last_sync_unix);
	switch (tone) {
		case 'success': return `${source.display_name} — synced ${relative}`;
		case 'warning': return `${source.display_name} — last error ${relative}: ${source.last_error}`;
		case 'destructive': return `${source.display_name} — unreachable since ${relative}`;
		default: return `${source.display_name} — not yet synced`;
	}
});
```
Add an untrusted-badge branch (independent of the sync-state switch, D-06: "the health tooltip spells out 'untrusted external plugin'") — likely appended/prefixed to the existing text rather than replacing the switch, since sync state and trust tier are orthogonal facts about the same chip.

---

### `web/src/lib/components/AddSourceModal.svelte` (untrusted interstitial + extras form)

**Analog:** itself (prior version) — existing multi-step state machine, verified this session at lines 1-140

**Existing step-state machine to extend** (`AddSourceModal.svelte`):
```ts
let step = $state<'existing' | 'connect' | 'link' | 'match' | 'connect-saved' | null>(null);
```
Add a new step value, e.g. `'untrusted-confirm'`, inserted between plugin-type selection and `'connect'` when the selected `pluginType`'s tier is `external` — directly analogous to how `'link'` (WhatsApp QR) is inserted conditionally between `'connect'` and `'match'` today (per the file's own doc comment: "'link' ... is the WhatsApp-only branch between 'connect' and 'match' ... entered instead of 'match' when the selected plugin type is WhatsApp").

**Existing `pluginTypes: string[]` prop to widen (Anti-Pattern warning, RESEARCH.md):** `PluginTypesResponse`/`pluginTypes` is currently a bare string array (`kernel/httpapi/config.go:233`, `web/src/lib/api.ts`). Adding tier metadata must be additive (a parallel `plugin_type_tiers` map or a new `plugin_types_detail` array), not a breaking change to the existing array's element shape — this is an explicit planner decision flagged in RESEARCH.md's Anti-Patterns section, not yet resolved.

**Existing conditional Alert-based warning UI to reuse for the interstitial's "what untrusted means" copy:**
```svelte
import { Alert, AlertDescription } from '$lib/components/ui/alert/index.js';
```
Already imported and used elsewhere in this file for error/notice copy (e.g. `linkNotice`) — reuse the same `Alert`/`AlertDescription` component for D-05's interstitial explanation text, keeping visual consistency with the existing `connectError`/`savedAnywayMessage` notice patterns in this same file.

**Type-the-plugin-name confirm box:** no existing analog in this file (net-new interaction) — model it as a plain bound `<input>` compared against the plugin binary name, gating the confirm `Button`'s `disabled` state, following this file's existing `saving`/`describing` boolean-gated-button convention (e.g. `disabled={saving}` on existing footer buttons).

**Extras form (declared + free-form):** compose `MatchFieldsForm.svelte`/`ConnectionForm.svelte` (both already imported into this file) as the closest existing analog for "render labeled inputs from a declared schema" — `ConnectionForm.svelte` in particular already renders `SourceConfig` fields from `defaultConnectionValues`/`missingRequiredFields` (`$lib/plugin-fields`); the extras form's declared-keys half should follow the identical field-rendering shape, with a free-form key/value editor appended below for undeclared keys (D-15).

---

### `web/src/lib/api.ts` (type additions)

**Analog:** itself — existing interface blocks, additive-field convention throughout this file

**Existing `SourceStatus` interface to extend** (verified this session, lines 262-272):
```ts
export interface SourceStatus {
	name: string;
	source_type: string;
	display_name: string;
	plugin: string;
	reachable: boolean;
	syncing: boolean;
	last_status: '' | 'running' | 'ok' | 'error';
	last_sync_unix: number;
	last_error: string;
}
```
Add: `tier: 'trusted' | 'external'; pinned_hash?: string; pin_mismatch?: boolean;` — additive, optional where the field may be absent for a pre-Phase-11 kernel or a trusted-tier source.

**Existing `PluginTypesResponse` interface** (line 417) — widen additively (see Anti-Pattern note above), do not mutate `plugin_types: string[]`'s element shape in place.

---

## Shared Patterns

### Content hashing
**Source:** `kernel/config/store.go:16-22` (`fileHash`)
**Apply to:** `kernel/pluginhost/binaryhash.go` (new `HashBinary`) — identical `sha256.Sum256`/`hex.EncodeToString` shape, one hashing convention project-wide.

### "Fail loudly by name" error construction
**Source:** `kernel/pluginhost/matchconfig.go` (`ValidateMatchConfig`) and `kernel/config.Validate`'s existing errors
**Apply to:** pin-mismatch errors, D-11 shadow-collision log lines — name the webspace/instance/binary/expected-vs-actual value in one `fmt.Errorf`, sorted iteration for deterministic ordering. Never an ad-hoc error shape.

### `${VAR}` env-reference scanning (reuse, do not reimplement)
**Source:** `kernel/httpapi/config.go:51` (`envVarPattern` regexp) + `envVarsIn`/`collectEnvVarNames` (lines 58-103)
**Apply to:** D-14's "list every env var handed over" confirm-step requirement — this machinery already walks an arbitrary `*Config` reflectively including `reflect.Map` cases, so it generically covers the new `Source.Extras` map with zero new scanning code. Scope the existing walk to one `Source`'s fields for the interstitial's disclosure list.

### Missing-directory-is-legitimate-empty-state
**Source:** `kernel/pluginhost/discover_binaries.go:111-118` (`DiscoverAllBinaries`)
**Apply to:** the external plugins directory (D-09) — apply identically; a missing external dir on first run is not an error, is an empty tier.

### UI-policy vs security-authority discovery split
**Source:** `kernel/pluginhost/discover_binaries.go`'s `DiscoverBinaries` (UI-policy, `ExcludedPluginBinaries`) vs `DiscoverAllBinaries` (security-authority, no exclusion)
**Apply to:** every widened call site (`PluginTypesHandler`, `DescribePluginHandler`, `launch()`, `cmd/topos`'s `pluginsDir` helper) must preserve this split across BOTH tiers — see Pitfall 4 in RESEARCH.md.

### Additive-only JSON shape changes (avoid `schemaVersion` bump)
**Source:** `kernel/httpapi/routes.go:23-25` (`const schemaVersion = 1`, "bump only for breaking JSON shape changes")
**Apply to:** every new field on `SourceStatus`/`PluginTypesResponse`/`DescribePluginResponse` — use `omitempty`/optional TS fields, never mutate an existing array's element shape in place (the `plugin_types: string[]` anti-pattern).

### Conditional per-chip menu items keyed on a derived boolean
**Source:** `web/src/lib/components/SourceChip.svelte`'s `isWhatsApp`/"Re-link…" pattern (D-03, 08-04-PLAN.md Task 2)
**Apply to:** the new "Trust updated binary" menu item, gated on a new `isPinMismatch` derived value, same discriminated-callback shape (`onedit(name, kind)`).

## No Analog Found

None — every file this phase touches is an in-place extension of code read directly this session; no capability requires a pattern from outside this codebase.

## Metadata

**Analog search scope:** `kernel/pluginhost/`, `kernel/config/`, `kernel/httpapi/`, `kernel/supervisor/`, `cmd/topos/`, `web/src/lib/components/`, `web/src/lib/api.ts`, `web/src/lib/plugin-fields.ts` — all read directly this session or in RESEARCH.md's own primary-source pass.
**Files scanned:** 13 target files + 3 analog-confirmation reads (`discover_binaries.go`, `config/store.go`, `config/types.go`, `SourceChip.svelte`, `PluginIcon.svelte`, `AddSourceModal.svelte`, `api.ts`)
**Pattern extraction date:** 2026-08-13
