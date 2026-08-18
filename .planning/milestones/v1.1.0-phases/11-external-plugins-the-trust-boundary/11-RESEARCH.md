# Phase 11: External Plugins & the Trust Boundary - Research

**Researched:** 2026-08-12
**Domain:** Go kernel plugin-host trust model (directory-tier provenance, content-hash pinning, subprocess environment hygiene) + SvelteKit picker/chip UI for an "untrusted" affordance
**Confidence:** HIGH (this phase is almost entirely an extension of existing, already-read kernel/web code; very little net-new external research was needed)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Hash pinning & upgrades**
- **D-01:** The pinned content hash lives in **config TOML** (e.g. a `[plugins]`-level pins map), written by the existing hot-apply config path at add-time. No kernel-owned sidecar store — config stays the single source of truth, and Phase 13's per-item marks remain "the kernel's first user-owned data beyond config".
- **D-02:** Pins are **per external binary**, not per source instance. All instances of a plugin share one pin; a re-accept updates every instance at once (divergent per-instance pins can't legally exist — one binary on disk serves all instances).
- **D-03:** Hash mismatch at launch fails loudly by name into a **named "binary changed" health state** on the source chip; the per-chip menu (Phase 9 pattern) offers an explicit **"Trust updated binary"** re-pin action that shows the new hash and rewrites the pin via hot-apply config. No remove-and-re-add, no manual config editing required.
- **D-04:** Pinning applies to the **external tier only**. Trusted-dir binaries launch unpinned — their provenance is the directory itself (they're rebuilt constantly by `make build`/`make dev`; pinning them would false-alarm on every rebuild).

**Warning & badge UX**
- **D-05:** Adding a source from an untrusted plugin inserts an explicit **confirm interstitial**: what "untrusted" means (code topos didn't build; no sandbox — honest labeling), the binary name, the full hash being pinned, plus a **type-the-plugin-name box** before confirm. The friction's purpose is accidental-click protection, not ceremony — keep it light.
- **D-06:** Chip badge = a small **warning glyph overlaid on the plugin's identity icon** inside the 44px merged pill; the health tooltip spells out "untrusted external plugin". No pill widening, no chip-wide tinting.
- **D-07:** Picker: external plugins list **inline in the existing install-catalog section** (alphabetical, as today), each row carrying the warning glyph + an "untrusted" label. No separate picker section.
- **D-08:** The pinned hash is **user-visible**: short copyable form in the per-chip menu/tooltip; full hash at the add-time confirm and in the re-pin flow — so a user can verify against an author-published checksum.

**External dir & collisions**
- **D-09:** Default external dir resolves to the **platform data dir** via a small per-OS helper — `$XDG_DATA_HOME/topos/...` (`~/.local/share/topos/plugins-external`) on Linux, `~/Library/Application Support` on macOS, `%LOCALAPPDATA%` on Windows — overridable in config. Portable resolution without committing to Windows support (the project stays Linux-anchored). Survives release upgrades; drop-a-binary-in works with zero config.
- **D-10:** External binaries follow the **same `topos-plugin-` prefix convention**; discovery reuses the exact `DiscoverBinaries` semantics (regular files, one-level symlink follow, sorted).
- **D-11:** Name collision between tiers: **trusted shadows external**, with a loud, named log line (never silent). Phase 12's rehearsal handles this as a test-setup detail (renamed copy or a trusted dir without the fs plugin).

**Extras config shape**
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

### Deferred Ideas (OUT OF SCOPE)
- Plugin distribution/pull-by-URL (PLUG-10).
- Developer guide, certification/trust promotion (PLUG-11).
- Sandboxing — trust marking is honest labeling, documented as such; `go-plugin` remains a transport, not a sandbox.
- `2026-08-05-signal-schema-version-verify-and-accept-tooling.md` — reviewed at phase start, unrelated (Signal-plugin maintenance tooling).
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PLUG-06 | Kernel discovers and launches plugin binaries from a configured external plugins directory, distinct from the trusted in-repo plugins directory | "Two-Tier Discovery" pattern below extends `DiscoverAllBinaries`/`DiscoverBinaries` (`kernel/pluginhost/discover_binaries.go`) with a second directory + tier tagging; `PluginsConfig` gains an `ExternalDir` field with the D-09 per-OS default resolver |
| PLUG-07 | Kernel derives a plugin's trusted/untrusted status from provenance (directory tier, content hash pinned at add-time and re-verified at every launch) — never from anything the plugin declares about itself | "Pin Verification Gate" pattern below: a pre-exec SHA-256 check inserted into `launch()` (`kernel/pluginhost/host.go`), reusing the `crypto/sha256`+hex pattern already established in `kernel/config/store.go`'s `fileHash` |
| PLUG-08 | User sees an explicit warning when adding a source from an untrusted plugin, and a persistent untrusted badge wherever that source appears (picker, source chips) | `AddSourceModal.svelte` interstitial step + `SourceChip.svelte` badge overlay on `PluginIcon.svelte`, both existing components this phase extends in place; "Named Health State vs. Launch-Failure Visibility" pitfall below covers the surfacing gap |
| PLUG-09 | Plugin host passes arbitrary per-instance config keys through to plugins (generic extra-fields map), so external plugins can receive provider-specific settings without kernel changes | `Source.Extras map[string]string` addition to `kernel/config/types.go`, threaded into `launch()`'s `WEBSPACES_SOURCE_CONFIG` JSON marshal (`kernel/pluginhost/host.go:426-437`) and `docs/plugin-contract.md`'s Describe extras declaration (D-15) |
</phase_requirements>

## Summary

Phase 11 is almost entirely an in-place widening of code this project already has: `kernel/pluginhost/discover_binaries.go`'s directory-listing discovery, `kernel/config`'s hot-apply save/canonical-rewrite path, `kernel/pluginhost/host.go`'s `launch()` function, and the `SourceChip.svelte`/`AddSourceModal.svelte`/`PluginIcon.svelte` UI trio from Phases 7 and 9. No new third-party dependency is needed anywhere in this phase — content hashing reuses the exact `crypto/sha256` + hex pattern `kernel/config/store.go`'s `fileHash` already establishes for the config-file clobber guard, and `${VAR}` expansion for extras is free: `expandEnv` (`kernel/config/config.go:131`) operates on the raw TOML **text** before it's ever decoded into the `Config` struct, so a new `[sources.X.extras]` sub-table's string values get identical `${VAR}`/`$VAR` substitution with zero new code.

The one piece of this phase that is **not** a small extension is D-03's requirement that a hash mismatch surface as a *named, visible, per-source health state on the chip* rather than an opaque failure. Today, `pluginhost.Discover` and `pluginhost.Host.Reconcile` treat **any** single source's launch failure as fatal to the *whole* boot (`Discover`) or the *whole* config apply (`Reconcile` kills what it just launched and returns an error, aborting the save with `500 apply_failed`) — and a source whose plugin never successfully launches has **no entry at all** in `GET /api/sources` today, because `SourcesHandler` iterates only `Host.snapshot()` (the launched-plugin list). A tampered/re-pinned external binary must be refused *before exec* (that's the whole point of the pin), which means this phase must introduce a new "configured but not launched, and here's why" representation that today's plugin host has no concept of. This is flagged in detail in Common Pitfalls and is the single design decision most likely to reshape task sequencing — resolve it explicitly during planning, since CONTEXT.md's decisions describe the desired *UX* (D-03) without addressing this *architectural* gap.

**Primary recommendation:** Add tier-aware discovery (`DiscoverAllBinaries` widened to accept a trusted dir + an external dir, returning binaries tagged by tier with the D-11 shadow rule applied), a `[plugins]` pins map + `Source.Extras` on `config.Config`/`config.Source`, a pre-exec SHA-256 gate in `launch()` that is soft-fails (never hard-fails Reconcile/Discover) for the external tier only, and a small `Host`-level "launch failures" side-channel that `SourcesHandler` merges into `GET /api/sources` so a pin-mismatched source still renders a chip. Reuse every existing UI pattern (`PluginIcon.svelte`'s fallback chain, `SourceChip.svelte`'s dropdown menu, `AddSourceModal.svelte`'s two-step flow) rather than building new components.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Directory-tier discovery (trusted vs. external, shadow rule) | API / Backend (`kernel/pluginhost`) | — | Only the kernel process can see the desktop filesystem; this is pure Go, no UI involvement |
| Content-hash pinning (compute, store, verify, re-pin) | API / Backend (`kernel/pluginhost` + `kernel/config`) | Database/Storage (config TOML is the store, per D-01) | Trust derivation must live entirely server-side — a client-computed hash would be an attacker-controlled input |
| Env allowlist / subprocess launch hygiene | API / Backend (`kernel/pluginhost/host.go`) | — | `exec.Cmd.Env` construction is exclusively a kernel-process concern |
| Untrusted warning interstitial + badge | Browser / Client (SvelteKit, `AddSourceModal.svelte`/`SourceChip.svelte`) | API / Backend (must publish tier + pin fields for the client to render) | The badge/warning is pure presentation over kernel-published facts — the client must never *decide* trust, only display it |
| Extras form (declared + free-form) | Browser / Client (`AddSourceModal.svelte` add-form) | API / Backend (Describe extras declaration passthrough) | Rendering a labeled form from a plugin's declared schema is a client concern; the schema itself travels over the existing Describe RPC → HTTP JSON path |
| "Binary changed" / launch-failure visibility | API / Backend (new Host-level failure tracking, `GET /api/sources` merge) | Browser / Client (chip health-state rendering) | The kernel is the only place a launch failure is observed; the client only renders what the kernel reports — but the kernel currently has *no channel* to report a source that never launched (see Common Pitfalls) |
| Platform external-dir default resolution | API / Backend (`cmd/topos` or `kernel/config`, a small per-OS helper) | — | Filesystem path resolution is inherently a backend/OS concern; no existing precedent in this repo (see Don't Hand-Roll) |

## Standard Stack

### Core

No new third-party library is required for this phase. Every capability is built from packages already imported by the kernel:

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `crypto/sha256` + `encoding/hex` (Go stdlib) | Go 1.25.0 toolchain (`go.mod:3` [VERIFIED: go.mod:3] `go 1.25.0`) | Content-hash pinning (D-01/D-02/D-03) | Already the exact pattern `kernel/config/store.go:16-20` uses for the config-file clobber guard (`fileHash`) — reuse the identical shape (`sha256.Sum256` → `hex.EncodeToString`) for plugin-binary hashing so the codebase has one hashing convention, not two |
| `github.com/pelletier/go-toml/v2` | already a direct dependency ([VERIFIED: go.mod] listed, version not independently re-verified this session — no change needed, this phase adds fields to existing structs it already marshals) | Canonical TOML rewrite of the new `[plugins]` pins map and `[sources.X.extras]` sub-table | `WriteCanonical` (`kernel/config/writer.go:40-41`) already calls `toml.Marshal(rawCfg)` over the whole `*Config` struct — a new field on `Config`/`Source` round-trips automatically, no new marshal code needed |
| `github.com/hashicorp/go-plugin` | v1.8.0 [VERIFIED: go.mod:13, cross-checked against `go list -m -versions` which confirms v1.8.0 is the latest published tag] | Subprocess transport (unchanged) | Already in use; this phase does not touch the handshake/transport layer, only what environment/args `exec.Cmd` is built with before `goplugin.NewClient` is called |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| none new | — | — | This phase is additive to existing Go stdlib + already-vendored packages only |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Stdlib per-OS `if runtime.GOOS` dir helper (D-09) | `github.com/adrg/xdg` or `github.com/kirsle/configdir` (third-party XDG/platform-dir libraries) | A third-party library would give broader OS coverage (BSD, etc.) for free, but this project's own stated position is "portable resolution without committing to Windows support... the project stays Linux-anchored" — a ~15-line hand-rolled `switch runtime.GOOS` (Linux: `$XDG_DATA_HOME` else `~/.local/share`; Darwin: `~/Library/Application Support`; Windows: `%LOCALAPPDATA%`) is simpler to audit for a trust-boundary-relevant default path than pulling in a new dependency whose own correctness now matters for where "untrusted binaries get discovered" resolves to. See Don't Hand-Roll below for the counter-argument on why this one *is* worth hand-rolling. |
| Config-stored pins map (D-01, locked) | A kernel-owned sidecar SQLite/JSON store for pins | Rejected by the user explicitly (D-01) — config stays the single source of truth; a sidecar store would be the *second* kernel-owned data store after Phase 13's per-item marks, undermining "config is authoritative" |

**Installation:**
No `go get`/`npm install` needed — this phase adds fields and functions to existing modules already in `go.work`/`go.mod`/`package.json`.

**Version verification:** `go-plugin` confirmed current at v1.8.0 via `go list -m -versions github.com/hashicorp/go-plugin`, matching `go.mod:13`'s already-pinned version — no bump required. `go.mod:3` confirms the toolchain floor as `go 1.25.0`.

## Package Legitimacy Audit

No external packages are introduced by this phase — every capability is implemented with Go stdlib (`crypto/sha256`, `encoding/hex`, `path/filepath`, `runtime`, `os`) plus packages already present in `go.mod`/`go.work` (`github.com/pelletier/go-toml/v2`, `github.com/hashicorp/go-plugin`). The Package Legitimacy Gate protocol (registry check, postinstall-script scan) is not applicable — there is nothing new to check.

**Packages removed due to [SLOP] verdict:** none — no new packages proposed.
**Packages flagged as suspicious [SUS]:** none.

## Architecture Patterns

### System Architecture Diagram

```
                     ┌─────────────────────────────────────────────┐
                     │  Browser (SvelteKit SPA)                     │
                     │                                               │
                     │  AddSourceModal.svelte                       │
                     │   picker: pluginTypes[] (now tier-tagged) ───┼──► GET /api/config/plugin-types
                     │   [+] untrusted row → confirm interstitial   │      { name, tier: "trusted"|"external" }
                     │        (type-name box, full hash shown) ─────┼──► POST /api/config/describe-plugin
                     │                                               │      (trial-launch, unchanged RPC shape)
                     │  SourceChip.svelte                            │
                     │   badge glyph over PluginIcon (tier=external)│
                     │   "Trust updated binary" menu item ──────────┼──► PUT /api/config (re-pin, base_hash lock)
                     └───────────────────┬───────────────────────────┘
                                          │ HTTP JSON
                     ┌────────────────────▼──────────────────────────┐
                     │  kernel/httpapi                                │
                     │   PluginTypesHandler   — now tier-aware        │
                     │   DescribePluginHandler — unchanged trial path │
                     │   ConfigSaveHandler → config.Store.Save        │
                     │        (writes pins map + extras sub-table)    │
                     │   SourcesHandler → NEW merge: launched plugins │
                     │        UNION configured-but-launch-failed set  │
                     └───────────────────┬─────────────────────────────┘
                                          │
                     ┌────────────────────▼──────────────────────────┐
                     │  kernel/pluginhost                             │
                     │                                                 │
                     │  DiscoverAllBinaries(trustedDir) ─┐             │
                     │  DiscoverAllBinaries(externalDir) ┼─► shadow    │
                     │                                    │  rule      │
                     │                                    ▼  (D-11)    │
                     │  ResolveBinary(name) → (path, tier)             │
                     │                                                 │
                     │  launch(name, src):                            │
                     │   1. resolve binary + tier                      │
                     │   2. tier==external? verify SHA-256(file)       │
                     │      against pins[binary] — MISMATCH: do NOT    │
                     │      exec; record failure; return soft-fail     │
                     │   3. tier==trusted: skip pin check (D-04)       │
                     │   4. build allowlisted env (D-14): PATH, HOME,  │
                     │      locale vars + this instance's own expanded │
                     │      config fields (incl. extras) — NEVER       │
                     │      os.Environ()                                │
                     │   5. marshal WEBSPACES_SOURCE_CONFIG JSON        │
                     │      { base_url, token, ..., extras: {...} }    │
                     │   6. exec, handshake, Describe (unchanged)       │
                     └───────────────────┬─────────────────────────────┘
                                          │ subprocess (gRPC over stdio pipe)
                     ┌────────────────────▼──────────────────────────┐
                     │  Plugin binary (trusted dir OR external dir)   │
                     │   reads WEBSPACES_SOURCE_CONFIG.extras[...]    │
                     │   (declares expected keys via Describe, D-15)  │
                     └─────────────────────────────────────────────────┘
```

### Recommended Project Structure

No new top-level packages — this phase extends existing files in place:

```
kernel/
├── config/
│   ├── types.go        # Source.Extras map[string]string; PluginsConfig.{ExternalDir, Pins}
│   ├── config.go        # Validate(): extras key-shape checks, pin-map shape checks
│   └── writer.go        # unchanged — toml.Marshal already round-trips new struct fields
├── pluginhost/
│   ├── discover_binaries.go   # widen to accept two dirs; add Tier type + shadow-rule resolver
│   ├── host.go                 # launch(): pre-exec pin check, env allowlist, extras in WEBSPACES_SOURCE_CONFIG
│   └── binaryhash.go           # NEW: small file — HashBinary(path) (string, error) via sha256+hex
├── httpapi/
│   ├── config.go        # PluginTypesHandler: tier-tagged response; re-pin endpoint (or reuse PUT /api/config)
│   └── sources.go        # SourcesHandler: merge in launch-failed configured sources
web/src/lib/
├── components/
│   ├── AddSourceModal.svelte   # untrusted interstitial step; extras form (declared + free-form)
│   └── SourceChip.svelte       # badge overlay on PluginIcon; "Trust updated binary" menu item
├── plugin-fields.ts             # extras field-declaration types, tier label helpers
└── api.ts                       # SourceStatus/DescribePluginResponse: tier, pin_hash, extras fields
docs/
├── plugin-contract.md   # Describe extras declaration, WEBSPACES_SOURCE_CONFIG.extras shape, env-allowlist note
└── api.md                # tier/pin fields on GET /api/sources and /api/config/plugin-types
```

### Pattern 1: Two-Tier Discovery with a Shadow Rule

**What:** Discovery becomes a two-directory operation. `DiscoverAllBinaries` (unexported internals aside) is called once per directory (trusted, external); a binary name present in both resolves to the **trusted** copy (D-11), with a loud named log line.

**When to use:** Every place that currently calls `pluginhost.DiscoverBinaries`/`DiscoverAllBinaries` with a single `pluginsDir` — `PluginTypesHandler`, `DescribePluginHandler`, `launch()`'s `os.Stat(binPath)` resolution, and `cmd/topos`'s boot-time `pluginsDir(cfg)` helper.

**Example (illustrative, not copied from any existing file — this is new code the planner should shape; it composes the exact existing functions read this session):**
```go
// Tier is the provenance signal PLUG-07 requires the kernel to derive —
// never anything a plugin declares about itself.
type Tier string

const (
	TierTrusted  Tier = "trusted"
	TierExternal Tier = "external"
)

// ResolveBinary applies D-11's shadow rule: a name present in both
// directories resolves to the trusted copy, logged loudly (never silent).
func ResolveBinary(trustedDir, externalDir, name string, logger hclog.Logger) (path string, tier Tier, err error) {
	trustedPath := filepath.Join(trustedDir, name)
	if _, statErr := os.Stat(trustedPath); statErr == nil {
		if externalPath := filepath.Join(externalDir, name); fileExists(externalPath) {
			logger.Warn("plugin binary name collision: trusted shadows external", "name", name)
		}
		return trustedPath, TierTrusted, nil
	}
	externalPath := filepath.Join(externalDir, name)
	if _, statErr := os.Stat(externalPath); statErr == nil {
		return externalPath, TierExternal, nil
	}
	return "", "", fmt.Errorf("plugin binary %s not found in trusted or external dir", name)
}
```

### Pattern 2: Pre-Exec Pin Verification Gate

**What:** For an external-tier binary, compute `SHA-256(file bytes)` and compare against `cfg.Plugins.Pins[binaryName]` **before** `exec.Command` is ever constructed. A mismatch must prevent the subprocess from ever running — the entire value of pinning is refusing to execute tampered bytes, not merely reporting after the fact.

**When to use:** Inside `launch()` (`kernel/pluginhost/host.go`), immediately after `ResolveBinary` returns `TierExternal`, before the existing `os.Stat(binPath)`/`exec.Command(binPath)` lines.

**Example (composes the existing `fileHash` pattern from `kernel/config/store.go:16-20`, which this session confirmed reads):**
```go
// kernel/config/store.go:16-20 — the existing convention this phase mirrors:
//   func fileHash(raw []byte) string {
//       sum := sha256.Sum256(raw)
//       return hex.EncodeToString(sum[:])
//   }

// HashBinary computes the same hex-SHA-256 digest over a plugin binary's
// file bytes (not TOML text) — the value pinned in cfg.Plugins.Pins and
// re-verified at every launch (PLUG-07).
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

**Design implication (see Common Pitfalls below):** a pin mismatch must be a **soft failure** recorded per-instance, not a hard error that aborts `Discover`/`Reconcile` for every other configured source.

### Pattern 3: Subprocess Env Allowlist (D-14)

**What:** Replace the current `env := append(os.Environ(), "WEBSPACES_SOURCE_CONFIG="+...)` (verbatim at `kernel/pluginhost/host.go:449` [VERIFIED: kernel/pluginhost/host.go:449] — the exact line reads `env := append(os.Environ(), "WEBSPACES_SOURCE_CONFIG="+string(sourceConfig))`) with an explicit allowlist built from scratch, never `os.Environ()`.

**When to use:** Same call site, `launch()` in `kernel/pluginhost/host.go`.

**Precedent already in this repo** (cited in CONTEXT.md's canonical_refs and confirmed this session — `docs/testing.md:107-111` [VERIFIED: docs/testing.md:107-111], which reads: *"An explicit environment allowlist on every kernel spawn (`PATH`, `HOME`, `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_CACHE_HOME` — all pointed at the fixture's own temp directory, never a `process.env` spread)"* — the e2e harness already does exactly this shape for spawning the **kernel** itself; this phase applies the identical discipline one process-boundary down, kernel → plugin.

```go
func allowedEnv(instanceConfig map[string]string) []string {
	env := []string{}
	for _, name := range []string{"PATH", "HOME", "LANG", "LC_ALL", "TZ"} { // exact locale-class set is Claude's discretion
		if v, ok := os.LookupEnv(name); ok {
			env = append(env, name+"="+v)
		}
	}
	sourceConfig, _ := json.Marshal(instanceConfig) // already-expanded, this instance's own fields only
	env = append(env, "WEBSPACES_SOURCE_CONFIG="+string(sourceConfig))
	return env
}
```

### Pattern 4: Extras Threading Through the Wire (D-12/D-13/D-15)

**What:** `Source.Extras map[string]string` (new field on the struct read this session — `kernel/config/types.go:46-132` [VERIFIED: kernel/config/types.go:46-132], the `Source` struct — currently has no `Extras` field; every existing field there, e.g. `BaseURL string \`toml:"base_url,omitempty" json:"base_url,omitempty"\`` (line 52), follows the `omitempty` convention this new field should match: `Extras map[string]string \`toml:"extras,omitempty" json:"extras,omitempty"\``) is marshaled into the existing `WEBSPACES_SOURCE_CONFIG` JSON envelope built at `kernel/pluginhost/host.go:426-437` [VERIFIED: kernel/pluginhost/host.go:426-437] — that block currently reads:
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
Because this is a flat `map[string]string`, adding `"extras": src.Extras` requires switching the marshaled value from `map[string]string` to a small anonymous/named struct (a nested map can't live inside a `map[string]string`) — e.g.:
```go
sourceConfig, err := json.Marshal(struct {
    BaseURL        string            `json:"base_url"`
    Token          string            `json:"token"`
    APIVersion     string            `json:"api_version"`
    CACert         string            `json:"ca_cert"`
    Username       string            `json:"username"`
    WebmailBaseURL string            `json:"webmail_base_url"`
    Path           string            `json:"path"`
    Extras         map[string]string `json:"extras,omitempty"`
}{ /* ... */ })
```
This is a **wire-shape change to `WEBSPACES_SOURCE_CONFIG`** — additive (existing plugins reading top-level keys are unaffected; `omitempty` means a source with no extras emits no `extras` key at all), but it is the exact JSON shape `docs/plugin-contract.md`'s "Configuration" section documents today (lines 192-244) and that section needs republishing to describe the new `extras` key, per this phase's canonical_refs note.

### Anti-Patterns to Avoid
- **Trusting a plugin's own claim about its trust tier or identity:** the contract's existing discipline — *"a plugin's identity is never trusted from its filename or its config key"* (`docs/plugin-contract.md:172-173`) — extends unchanged to trust tier: tier is derived **only** from which directory the binary was resolved from plus the pin, never from anything in `DescribeResponse` (PLUG-07's own wording).
- **Computing or accepting a client-supplied hash:** the pin recorded at add-time must be computed **server-side** from the binary the kernel itself just resolved and trial-launched (`DescribePluginType`'s existing trial-launch path is the natural place to compute it) — never trust a hash value in the request body for `describe-plugin` or the save.
- **Widening `PluginTypesHandler`'s response shape without a `schema_version` bump or an additive-only change:** `plugin_types` is currently `[]string` (`kernel/httpapi/config.go:233` [VERIFIED: kernel/httpapi/config.go:233], `PluginTypes []string \`json:"plugin_types"\``). Changing every element from a bare string to `{name, tier}` is a **breaking** shape change for any existing consumer parsing `plugin_types[i]` as a string — per this repo's own documented convention (`kernel/httpapi/routes.go:23-25` [VERIFIED: kernel/httpapi/routes.go:23-25], *"schemaVersion is the envelope's schema_version field. Bump only for breaking JSON shape changes."* `const schemaVersion = 1`), this either needs a `schemaVersion` bump or must be done as an **additive** second field (e.g. keep `plugin_types []string` unchanged and add `plugin_type_tiers map[string]string` or a parallel `plugin_types_detail []{name,tier}` array) rather than mutating the existing array's element shape in place. This exact tradeoff is not decided anywhere in CONTEXT.md — flag it for the planner as a concrete task-level decision.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Content hashing for pin verification | A custom hash/digest scheme, or shelling out to `sha256sum` | `crypto/sha256` + `encoding/hex`, exactly mirroring `kernel/config/store.go`'s existing `fileHash` (verified this session, lines 16-20) | Stdlib, already the established in-repo convention for content-hash-based integrity, zero new dependency risk |
| Per-OS data directory resolution | A hand-rolled scheme that guesses wrong on one platform and is never caught because the project only tests on Linux | A small, explicitly-tested `switch runtime.GOOS` helper (this genuinely IS the right scope to hand-roll per D-09's own reasoning: "without committing to Windows support... the project stays Linux-anchored" — pulling in `github.com/adrg/xdg` or similar for two never-tested branches (macOS/Windows) adds a dependency whose correctness the project cannot verify, for platforms it doesn't ship on) | A ~15-line function with one Linux integration test and two untested-but-documented branches is more honest than a third-party dependency implying full cross-platform support this project doesn't provide |
| Env-var reference scanning for the "list every env var handed over" confirm-step requirement (D-14) | A new regex/parser for `${VAR}`/`$VAR` | `envVarPattern` (`kernel/httpapi/config.go:51` [VERIFIED: kernel/httpapi/config.go:51], `regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}\|\$([A-Za-z_][A-Za-z0-9_]*)`)`) plus `envVarsIn`/`collectEnvVarNames` (lines 58-103) already walk an arbitrary `*Config` reflectively and report every referenced `${VAR}` name — reuse this exact machinery scoped to one `Source`'s fields (including the new `Extras` map, which `collectEnvVarNames`'s `reflect.Map` case already handles generically) rather than writing a second scanner |
| "Fail loudly by name" error construction for pin mismatches / vocabulary mismatches | Ad-hoc error strings per call site | Follow the exact convention `pluginhost.ValidateMatchConfig`/`validateMatchConfig` (`kernel/pluginhost/matchconfig.go`) and `config.Validate`'s errors already establish: name the webspace/instance/binary/expected-vs-actual value in one `fmt.Errorf`, sorted iteration for deterministic ordering | This project has an established, tested house style for "fail loudly by name" — a new ad-hoc error shape for pin mismatches would be inconsistent with the rest of the codebase's error UX |

**Key insight:** This phase's entire job is threading a new provenance/trust concept through machinery that already exists for a *different* purpose (config hot-apply, discovery, health surfacing, per-chip menus) — the highest-value engineering here is *reuse*, not new abstractions. The one place genuinely new machinery is required (launch-failure visibility, see below) should be scoped as narrowly as possible.

## Common Pitfalls

### Pitfall 1: All-or-Nothing Launch Failure Breaks D-03's Per-Source Health State (the central architectural gap)

**What goes wrong:** `pluginhost.Discover` (`kernel/pluginhost/host.go:217-230`, read this session) calls `h.Shutdown()` and returns a hard error the **first** time any single source's `launch()` fails — this fails the entire kernel boot (`kernel/supervisor/supervisor.go:133`, `pluginhost.Discover(ctx, pluginsDir, cfg.Sources, logger)`, unguarded), not just the one misconfigured source. `Host.Reconcile` (lines 261-321) is only slightly softer: it kills whatever it *just* launched this round and returns an error, which propagates up through `Supervisor.Apply` (`kernel/supervisor/supervisor.go:672-679`) as `500 apply_failed` — **the entire config save is rejected**, not just the pin-mismatched source. Separately, `SourcesHandler` (`kernel/httpapi/sources.go:65-124`, read this session) builds its response **exclusively** from `prober.ProbeSources(ctx)`, which iterates `Host.snapshot()` — the launched-plugin list. A source whose plugin never successfully launched has **no entry in `GET /api/sources` at all** today.

**Why it happens:** The existing launch-failure model was designed for "your binary is missing/broken — fix your config" (a genuinely all-or-nothing failure class, reasonable for that case). D-03 asks for something categorically different: a *specific, expected, per-source, recoverable* condition (an external binary that got rebuilt/swapped after being pinned) that must (a) never execute the tampered bytes, but (b) still let every *other* configured source boot/apply normally, and (c) still render a visible, actionable chip for the affected source.

**How to avoid:** This needs new plumbing, scoped as narrowly as possible:
1. Insert the pin-verification gate (Pattern 2 above) as a check performed *before* `os.Stat`/`exec.Command` inside `launch()`, for `TierExternal` only.
2. On mismatch, `launch()` must NOT return the same hard error class `Discover`/`Reconcile` already treat as fatal-to-everything — introduce a distinguishable sentinel (e.g. `ErrPinMismatch`) that `Discover`/`Reconcile` catch specifically and handle as a **soft, per-instance** failure: skip adding that instance to `Host.plugins`, record it in a new small side-channel (e.g. `Host.launchFailures map[string]string` under the existing `h.mu`), and continue processing every other configured source normally.
3. Widen `SourcesHandler`'s merge (`sourceStatusesFrom`, `kernel/httpapi/sources.go:88-124`) to iterate the **configured source set** (not just launched plugins) and fall back to a synthesized `sourceStatus` entry (using `cfg.Sources[name].DisplayName`/instance id, since there's no `Plugin`/`SourceType` from a `Describe` call that never happened) for any name present in config but absent from `ProbeSources`'s result, carrying whatever distinguishing signal (see Pitfall 2) the UI needs to render "binary changed" specifically, as opposed to the existing `unknown`/never-synced tone.
4. Decide explicitly (this is not decided in CONTEXT.md) whether **other** launch failure classes (missing binary, crash before handshake) should be softened the same way, or should keep today's hard-fail-the-whole-apply behavior. The safest minimal-footprint choice: soften **only** the pin-mismatch path (new, narrow sentinel), leave every other failure class's existing hard-fail behavior untouched — this satisfies D-03 without silently changing the risk profile of an unrelated failure mode (a genuinely broken/missing binary arguably *should* still block the whole apply, so the operator notices immediately, exactly as it does today).

**Warning signs during planning:** if a plan's task list treats "add pin verification" as a single self-contained task inside `launch()` without also touching `Discover`, `Reconcile`, and `SourcesHandler`, the resulting "binary changed" state will be technically correct (the tampered binary genuinely never executes) but **invisible** — success criterion 3 ("fails loudly by name... instead of inheriting stale trust") will not actually render on any chip, silently failing UAT.

### Pitfall 2: No Existing Machine-Readable "Health State" Field to Key the UI Off Of

**What goes wrong:** WhatsApp's much-referenced "five/six named health states" (`STATE.md`, `PROJECT.md`, `docs/plugins/whatsapp.md:50`) are **not** a discrete enum surfaced anywhere in the JSON contract — they are distinguishable **`last_error` text strings** the plugin itself returns via `HealthResponse.last_error`, rendered through the *existing* `warning`/`destructive` `HealthTone` branches (`web/src/lib/format.ts:108-123`, confirmed this session — `healthTone` only ever returns one of `'success' | 'warning' | 'destructive' | 'unknown'`, derived from `last_status`/`reachable`, never from a state-name field). `SourceStatus` (`web/src/lib/api.ts:262-272`, read this session) has no `health_state`/`state_name` field at all — only `reachable: boolean`, `last_status: '' | 'running' | 'ok' | 'error'`, `last_error: string`.

**Why it happens:** WhatsApp's states all occur **after** a successful launch (the plugin is running and reports its own degraded condition via `Health` RPC) — a case the existing `reachable`/`last_error` shape already covers adequately with distinguishable text. A "binary changed" state is different in kind: it's a **pre-launch** kernel-side refusal, not a plugin self-report, and per D-03 the UI needs to *conditionally offer a specific action* ("Trust updated binary") — which means the frontend needs a **reliable, non-text-parsing signal**, not a string it would otherwise have to pattern-match.

**How to avoid:** Add an explicit machine-readable field (e.g. `sourceStatus.LaunchFailure string` with a fixed enum-like vocabulary such as `""`/`"pin_mismatch"`/`"binary_missing"`, or a boolean `PinMismatch bool` plus the existing `LastError` for the human-readable text) to the `GET /api/sources` response shape, rather than relying on the frontend to string-match `last_error` text to decide whether to show the "Trust updated binary" menu item. This is a genuinely new field on `sourceStatus`/`SourceStatus` — plan for it explicitly (and consider whether it's additive, avoiding a `schemaVersion` bump, since a new optional field with `omitempty` should be safe under this repo's existing convention).

### Pitfall 3: Extras Wire-Shape Change Breaks the Flat `map[string]string` Marshal

**What goes wrong:** `WEBSPACES_SOURCE_CONFIG` is currently built as `json.Marshal(map[string]string{...})` (`kernel/pluginhost/host.go:426-437`) — a flat map literal. A nested `extras` object cannot be expressed inside a `map[string]string` (Go's type system rejects a `map[string]string` value that is itself a map). Naively adding `"extras": someMap` to that literal is a compile error, not a runtime surprise — but it's easy to instead flatten extras keys into the same top-level map (losing the D-12 "dedicated sub-table, kernel-known fields stay typed" boundary) as an expedient fix under time pressure.

**Why it happens:** The existing code's flat-map shape was correct when every config field was a scalar string; extras breaks that invariant for the first time.

**How to avoid:** Switch the marshal target from an untyped `map[string]string` literal to a named/anonymous struct with an explicit `Extras map[string]string \`json:"extras,omitempty"\`` field (shown in Pattern 4 above) — this is additive to the wire contract (old plugins reading only top-level keys are unaffected) and preserves D-12's sub-table boundary end-to-end (TOML `[sources.X.extras]` → Go `Source.Extras` → nested `extras` JSON key → plugin-side `WEBSPACES_SOURCE_CONFIG.extras`).

### Pitfall 4: `DiscoverBinaries` vs `DiscoverAllBinaries`'s UI-Policy/Security-Authority Split Must Be Preserved Across Two Directories

**What goes wrong:** The existing split — `DiscoverBinaries` (UI-policy: what the "+" picker offers, excludes `mock`/`mockstrict`) vs. `DiscoverAllBinaries` (security-authority: what `DescribePluginHandler` accepts as a launchable binary name, per the documented rationale at `kernel/pluginhost/discover_binaries.go:76-93`, read this session) — exists specifically so an *already-configured* instance of an excluded/dev-only binary keeps working (Describe/edit/sync) even though it's never offered as a *new* pick. When widening discovery to two directories, it's easy to accidentally apply `ExcludedPluginBinaries` filtering to only one tier, or to forget that `DescribePluginHandler`'s existing 404 `plugin_binary_not_found` guard (`kernel/httpapi/config.go:286-299`, T-07-09's "directory listing, never a caller-supplied path, is the authority over what may be launched") must now check membership across **both** directories' `DiscoverAllBinaries` results, tier-aware.

**How to avoid:** Whatever `ResolveBinary`/two-tier discovery shape the planner lands on, explicitly re-verify (with a test, mirroring the existing `discover_binaries_test.go` style — e.g. `TestDiscoverBinaries_ExcludesMockBinary`, `TestDiscoverAllBinaries_IncludesMockBinary`, read this session as `func Test...` names) that: (a) the mock/mockstrict exclusion policy decision (explicitly left to Claude's discretion in CONTEXT.md — "Whether `ExcludedPluginBinaries` policy applies to the external tier") is deliberately applied or deliberately not, not accidentally dropped; (b) `DescribePluginHandler`'s security check still spans both tiers.

### Pitfall 5: Repo-Internal AST/Audit Scans Only Cover `plugins/` — the Proof Binary Must Not Accidentally Trigger or Evade Them Incorrectly

**What goes wrong:** `internal/audit` runs repo-wide Go-AST scans scoped to files under `plugins/` (confirmed this session — `internal/audit/outbound_hosts_test.go`'s `sanctionedEgressFiles` allowlist and `internal/audit/plugin_icons_test.go`'s `provenanceKeys`/`pluginIconOffenses` both walk `plugins/*` module directories). Success criterion 5's proof binary ("a real binary built outside the in-repo plugin set") is explicitly supposed to be built **out-of-repo** in spirit — if the planner's chosen approach (Claude's discretion: "likely a rebuilt/renamed mock-derived plugin") is built as a new directory *under* `plugins/` in this repository, it will be silently swept into these audits (module-pin floor scan, outbound-egress AST scan, icon-provenance scan) and must either satisfy them or be added to their allowlists — expanding audit scope for a binary that's conceptually meant to simulate a third party's own separate build.

**How to avoid:** Build the criterion-5 proof binary in a location genuinely outside `plugins/` (e.g. a scratch directory, or copy-and-rebuild `plugins/mock`'s source into a `.planning`/tooling-adjacent throwaway module, or into the e2e fixtures directory) so it is not picked up by `internal/audit`'s repo-walking scans at all — matching the spirit of "proving a third party can ship a working untrusted plugin" (this exact framing is echoed in SRC-06's later requirement for the real Google Drive plugin, Phase 14).

## Code Examples

### `fileHash`-style content hashing (the exact convention to mirror)
```go
// Source: kernel/config/store.go:16-20 (read this session)
func fileHash(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
```

### Today's plugin-subprocess env construction (the line D-14 replaces)
```go
// Source: kernel/pluginhost/host.go:448-453 (read this session)
cmd := exec.Command(binPath)
env := append(os.Environ(), "WEBSPACES_SOURCE_CONFIG="+string(sourceConfig))
if describeOnly {
	env = append(env, "WEBSPACES_DESCRIBE_ONLY=1")
}
cmd.Env = env
```

### `DiscoverAllBinaries`'s symlink-following regular-file check (reuse verbatim for the external dir)
```go
// Source: kernel/pluginhost/discover_binaries.go:141-155 (read this session)
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

### Kernel's own env-allowlist precedent (e2e harness spawning the kernel)
```
Source: docs/testing.md:107-111 (read this session)
"An explicit environment allowlist on every kernel spawn (PATH, HOME,
XDG_CONFIG_HOME, XDG_DATA_HOME, XDG_CACHE_HOME — all pointed at the
fixture's own temp directory, never a process.env spread)"
```

## State of the Art

| Old Approach | Current/Recommended Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Single plugins directory, `DiscoverBinaries(pluginsDir string)` | Two-tier `DiscoverAllBinaries` call per directory + a `ResolveBinary` shadow-rule resolver | This phase (PLUG-06) | Every call site currently taking one `pluginsDir string` (`PluginTypesHandler`, `DescribePluginHandler`, `launch()`, `cmd/topos.pluginsDir`) needs a second directory threaded through |
| `exec.Cmd.Env = append(os.Environ(), ...)` (full env inheritance) | Explicit allowlist (`PATH`/`HOME`/locale + instance's own expanded fields) | This phase (D-14) | Any plugin relying on an *inherited* env var it never declared in its own config (undocumented today, but theoretically possible under the old code) will break — acceptable, matches the trust-boundary goal |
| `map[string]string` flat `WEBSPACES_SOURCE_CONFIG` marshal | Struct with a nested `extras map[string]string` field | This phase (PLUG-09/D-12) | Additive wire change; `docs/plugin-contract.md`'s "Configuration" section needs republishing per canonical_refs |
| `plugin_types: string[]` picker response | Either additive parallel tier field, or a breaking shape change requiring a `schemaVersion` bump | This phase (PLUG-06/D-07) | Planner must pick one explicitly — see Anti-Patterns above |

**Deprecated/outdated:** Nothing in this phase deprecates prior phases' work — it is purely additive to the plugin-host/config/UI surfaces Phases 1-10 already built.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The exact locale-class env vars to allowlist beyond `PATH`/`HOME` (e.g. `LANG`, `LC_ALL`, `TZ`) — explicitly left to Claude's discretion in CONTEXT.md, and this research's Pattern 3 example is illustrative only | Architecture Patterns, Pattern 3 | Low — CONTEXT.md already delegates this choice; a too-narrow allowlist just means a plugin needing e.g. `TZ` fails loudly and the list is widened, never a silent security gap |
| A2 | The recommendation to build the criterion-5 proof binary genuinely outside `plugins/` (Pitfall 5) is this research's own inference from reading `internal/audit`'s scan scope, not a decision recorded anywhere in CONTEXT.md/ROADMAP.md | Common Pitfalls, Pitfall 5 | Medium — if the planner instead builds it under `plugins/` deliberately (e.g. to reuse `go.work`/Makefile plumbing) and consciously adds the required audit allowlist entries, that is a legitimate alternative; flagging it here so the choice is made deliberately rather than by accident |
| A3 | Whether other launch-failure classes (missing binary, crash before handshake) should be softened the same way as pin mismatches, or should keep the existing hard-fail-the-whole-apply behavior — this research recommends narrowing the fix to pin-mismatch only, but CONTEXT.md does not address this at all | Common Pitfalls, Pitfall 1 | High if unaddressed during planning — this is the single largest architectural decision this phase requires that isn't already locked by CONTEXT.md's decisions; getting it wrong either over-scopes the phase (softening every failure class) or under-delivers D-03 (pin mismatch still aborts the whole apply) |
| A4 | `PluginTypesHandler`'s response shape change (tier-tagging) should be additive rather than a breaking `schemaVersion` bump — this research presents both options without picking one | Architecture Patterns, Anti-Patterns | Low-Medium — either choice is workable; picking wrong just means an extra migration step for any external consumer of this specific endpoint, which today has none (it's SPA-internal) |

**If this table is empty:** N/A — see above.

## Open Questions

1. **How should a pin-mismatch launch failure be represented in `Host` and surfaced through `GET /api/sources`, without weakening the existing "partial apply must never look successful" (T-07-11) guarantee for genuine (non-pin) launch failures?**
   - What we know: `Discover`/`Reconcile`'s current all-or-nothing failure model, and D-03's requirement for a per-source visible state, are in direct tension (Pitfall 1).
   - What's unclear: whether the fix should be scoped narrowly (pin-mismatch only) or should more broadly introduce a "degraded/failed source" concept `Host` and `GET /api/sources` don't have today for any failure class.
   - Recommendation: scope narrowly (a new `ErrPinMismatch` sentinel, caught specifically by `Discover`/`Reconcile`, recorded in a small `Host`-level side-channel) — this satisfies D-03's literal wording ("fails loudly by name into a named health state") with the smallest surface-area change, and doesn't retroactively change the risk posture of an existing failure mode (a broken/missing binary) that the project has never described as needing a soft-fail treatment.

2. **Does `sourceStatus`/`SourceStatus` need a new machine-readable field (e.g. a launch-failure reason enum), or can the "Trust updated binary" menu item be gated on the existing `last_error` string via a fixed prefix/marker convention?**
   - What we know: WhatsApp's precedent is text-only (`last_error` string matching), never a discrete field (Pitfall 2).
   - What's unclear: whether reusing text-matching for a UI *action-gating* decision (not just tooltip copy) is acceptable given this repo's general aversion to fragile string-matching in the frontend, or whether it crosses a line WhatsApp's precedent didn't (WhatsApp's states never conditionally show/hide a menu item based on parsing `last_error` — they only change tooltip copy).
   - Recommendation: add an explicit boolean or enum field — cheap, unambiguous, and avoids coupling UI logic to a human-readable string that copy-editing could later break.

3. **Exact JSON key names for the new wire fields** (`tier` vs `trust`/`trusted`; `pin_hash` vs `pinned_hash`; `extras` — this last one is effectively locked by D-12/D-15's own wording) are not specified anywhere and should be decided once, consistently, across `GET /api/config/plugin-types`, `POST /api/config/describe-plugin`, `GET /api/sources`, and the `[sources.X.extras]`/`[plugins.pins]` TOML keys.
   - Recommendation: `tier: "trusted" | "external"` (matches D-09/D-11's own vocabulary), `pinned_hash` (TOML key, matching `[plugins]`-level pins map per D-01's own phrasing "a `[plugins]`-level pins map").

## Environment Availability

No new external service, CLI tool, or runtime dependency is introduced by this phase — it is Go kernel code + SvelteKit frontend code only, both already fully available in this repository's existing build (`make build`/`make dev`/`make e2e`, confirmed present via `Makefile` lines 135-157 read this session). The only "environment" consideration is that the default external plugins directory (D-09) may not exist on first run — `launch()`/discovery must treat a missing external directory identically to how `DiscoverAllBinaries` already treats a missing trusted directory today (`kernel/pluginhost/discover_binaries.go:112-118`, read this session: `if os.IsNotExist(err) { return []string{}, nil }` — "a legitimate empty state ... not an error").

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Building the kernel/pluginhost changes | ✓ | 1.25.0 (`go.mod:3`) | — |
| `github.com/hashicorp/go-plugin` | Subprocess transport (unchanged) | ✓ | v1.8.0 (current, `go.mod:13`) | — |
| Platform data directory (`~/.local/share/topos/plugins-external` on Linux) | Default external plugins dir (D-09) | Created on demand | — | `mkdir -p` at first discovery attempt, or documented manual creation; missing dir = empty external tier, never an error |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** the external plugins directory itself, per above.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V1 Architecture, Design and Threat Modeling | yes | This phase *is* a threat-model exercise made visible in the UI (honest labeling of an untrusted execution boundary) — `docs/plugin-contract.md`'s existing "What this document does not cover" / trust-boundary framing (lines 62-69, read this session: *"Installing a third-party plugin is therefore the same trust decision as installing the kernel binary itself"*) is the documented threat model this phase builds controls around |
| V5 Input Validation | yes | Extras free-form key/value editor (D-15) accepts arbitrary operator-typed keys — validate key names are non-empty/reasonable TOML identifiers before writing to `[sources.X.extras]`; validate the requested plugin binary name against `DiscoverAllBinaries`' own result set (existing T-07-09 discipline, unchanged, extended to two tiers) |
| V6 Cryptography | no (narrow) | SHA-256 here is an **integrity/identity** check (content-addressing), not an authentication or confidentiality control — no key management, no signing. Worth noting explicitly so the planner doesn't over-engineer this as a cryptographic-authentication feature: a pin proves "this is the same bytes I saw before," not "this bytes came from a trusted publisher" (there is no publisher signature anywhere in this design — D-08's "verify against an author-published checksum" is a manual, human-driven trust step, not something the kernel cryptographically verifies against a third party) |
| V10 Malicious Code | yes | This phase's entire purpose is drawing an honest line around code the project can't vouch for — no sandboxing is implemented (explicitly out of scope, matches `docs/plugin-contract.md`'s existing "no containment" framing) — the control here is disclosure (badge, warning, hash pin, env scrub), not prevention |
| V12 Files and Resources | yes | Directory-listing-only discovery (never a caller-supplied path) must be preserved across both tiers (Pitfall 4); the external dir's default resolution must not silently resolve to an attacker-writable shared location — `$XDG_DATA_HOME`/`~/Library/Application Support`/`%LOCALAPPDATA%` are all user-owned by OS convention, consistent with the existing `~/.config/topos`/`~/.local/share/topos` precedent this project already uses for config/index |
| V14 Configuration | yes | The pins map and extras sub-table both live in `config.toml`, inheriting all of that file's existing protections (D-05 secret-value-never-in-response, D-01 lossless-rewrite, D-03 optimistic-lock clobber guard) — no new config-security surface beyond what Phase 7 already built |

### Known Threat Patterns for This Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Binary swap after pinning (operator pins v1, an attacker or a bad update silently replaces the file with v2 before the next launch) | Tampering | Pre-exec SHA-256 re-verification at every launch (PLUG-07, Pattern 2) — refuse to exec, never silently relaunch with stale trust |
| Env-var exfiltration via a malicious plugin reading secrets it was never given (`os.Environ()` inheritance today) | Information Disclosure | D-14's allowlist — this is the single largest concrete threat this phase closes, explicitly user-raised in CONTEXT.md's `<specifics>` ("a malicious plugin could hoover up any number of values from well known env vars and phone home") |
| A malicious plugin's `Describe` response suggesting an extras default that references a secret env var (`${SOME_SECRET}`) to get the operator to type it into the add-form | Information Disclosure / Social Engineering | D-14: "plugin-suggested extras defaults are never auto-filled" — declared defaults from `Describe` are display-only, never pre-populated into the form field the operator would then save |
| Directory-traversal or symlink trickery in the external plugins directory (e.g. a symlink pointing outside the intended tree) | Tampering / Elevation of Privilege | Existing `isRegularFileFollowingSymlinks`'s one-level-only symlink-follow (`kernel/pluginhost/discover_binaries.go:141-155`) already bounds this for the trusted tier; extend identically to the external tier — no new traversal surface beyond what already exists |
| Accidental widening of `ExcludedPluginBinaries`/discovery security-authority split when adding a second directory (Pitfall 4) | Elevation of Privilege (an excluded dev-fixture binary becomes launchable via the external tier by omission) | Explicit tests mirroring existing `discover_binaries_test.go` coverage, across both tiers |

## Sources

### Primary (HIGH confidence — read directly this session)
- `kernel/pluginhost/discover_binaries.go` (full file) — discovery, prefix convention, exclusion policy, symlink-following
- `kernel/pluginhost/host.go` (full file) — `launch()`, env construction, `Discover`/`Reconcile`'s all-or-nothing failure model, `SourceHealth`/`ProbeSources`
- `kernel/pluginhost/matchconfig.go` (partial) — the "fail loudly by name" validation precedent
- `kernel/config/types.go` (full file) — `Config`/`Source`/`PluginsConfig`/`Webspace` struct shapes
- `kernel/config/config.go` (partial) — `LoadRaw`, `expandEnv`, `applyDefaults`, `Validate` structure
- `kernel/config/store.go` (partial) — `fileHash`, `Save`'s optimistic-lock/hot-swap sequence
- `kernel/config/writer.go` (full file) — `WriteCanonical`'s deterministic marshal + backup + atomic rename
- `kernel/httpapi/config.go` (full file) — `ConfigHandler`/`ConfigSaveHandler`/`PluginTypesHandler`/`DescribePluginHandler`, `envVarsIn`/`collectEnvVarNames`
- `kernel/httpapi/sources.go` (partial) — `SourcesHandler`/`sourceStatusesFrom`'s launched-plugin-only merge
- `kernel/supervisor/supervisor.go` (partial) — `NewSupervisor`'s `Discover` call, `Apply`'s `Reconcile` call and error propagation
- `cmd/topos/main.go` (partial) — `configPath`/`pluginsDir` resolution helpers, existing XDG_CONFIG_HOME precedent
- `docs/plugin-contract.md` (full file) — the published third-party contract this phase extends
- `docs/api.md` (partial, lines 540-599) — `GET`/`PUT /api/config` envelope semantics, `env_vars` boolean-only discipline
- `docs/testing.md` (lines 80-137) — e2e harness architecture, the kernel-spawn env-allowlist precedent
- `web/src/lib/components/SourceChip.svelte` (full file) — the 44px merged pill, per-chip dropdown menu pattern
- `web/src/lib/components/PluginIcon.svelte` (full file) — three-step fallback chain, badge-overlay integration point
- `web/src/lib/components/AddSourceModal.svelte` (partial, lines 1-120) — two-step picker/add flow structure
- `web/src/lib/format.ts` (partial) — `healthTone`, the four-value `HealthTone` type
- `web/src/lib/api.ts` (partial) — `SourceStatus`, `PluginTypesResponse`, `DescribePluginResponse` interfaces
- `web/e2e/fixtures/plugin-binaries.ts` (full file) — `linkPluginBinaries`, the closed-set e2e fixture symlinker
- `internal/audit/outbound_hosts_test.go` / `internal/audit/plugin_icons_test.go` (partial) — repo-scoped audit-scan boundaries
- `.planning/phases/11-external-plugins-the-trust-boundary/11-CONTEXT.md`, `.planning/REQUIREMENTS.md`, `.planning/STATE.md` (full files) — phase scope and locked decisions
- `go.mod` (partial) — Go 1.25.0 toolchain, `go-plugin` v1.8.0 pin
- `go list -m -versions github.com/hashicorp/go-plugin` (tool output) — confirms v1.8.0 is current

### Secondary (MEDIUM confidence)
- WebSearch, XDG Base Directory Specification confirmation (`specifications.freedesktop.org/basedir/latest/`, cross-referenced against `xdgbasedirectoryspecification.com` and the ArchWiki) — confirms `$XDG_DATA_HOME` defaults to `$HOME/.local/share`, matching D-09's own stated default exactly

### Tertiary (LOW confidence)
- None used untagged — every claim above is either read from this repository's own source this session or a well-established, cross-checked platform-directory convention.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; every pattern reuses code read directly this session
- Architecture: HIGH for the discovery/pin/env-allowlist/extras patterns (directly extending read code); MEDIUM for the exact shape of the launch-failure-visibility fix (Pitfall 1/Open Question 1) — the *problem* is verified with high confidence (read `Discover`/`Reconcile`/`SourcesHandler` directly), the *specific recommended fix* is this research's own design proposal, not something already decided in the codebase or CONTEXT.md
- Pitfalls: HIGH — every pitfall traces to a specific file/line read this session, not speculation

**Research date:** 2026-08-12
**Valid until:** 30 days (stable, in-repo-only domain; no fast-moving external dependency to go stale)
