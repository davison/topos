# Phase 14: Google Drive Source, Built Out-of-Repo - Pattern Map

**Mapped:** 2026-08-15
**Files analyzed:** 8 (this repo — the plugin repo itself is out of scope for this map, D-08)
**Analogs found:** 7 / 8

## Scope Note

Phase 14's *actual new feature code* (OAuth, Drive API, sync-state cache) lives
entirely in the separate `topos-plugin-gdrive` repository per D-08 and must
never be planned from this repo's patterns (clean-room discipline, D-06). This
map covers only what topos-side plans touch: the two folded todos, the e2e
rehearsal spec, the contract-gap republish, and the plugin-repo PRD hand-off
document. Do not read this file as license to sketch plugin-repo Go code from
kernel analogs — the PRD cites only the four published inputs.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|-----------------|---------------|
| `cmd/topos/main.go` (config-path resolution edit) | config | request-response (CLI arg/env resolution) | same file, `configPath()`/`pluginsDir()` (existing functions being extended) | exact (self-analog) |
| `Makefile` (dev-config target/convention) | config | batch | same file (existing `make dev`/`make build` targets) | exact (self-analog) |
| `config.example.toml` (dev-config example, if added) | config | — | same file (existing documented example blocks, e.g. `[sources.filesystem.extras]`) | exact (self-analog) |
| `docs/testing.md` (doc split note) | config/docs | — | same file (existing "what stays manual" sections) | exact (self-analog) |
| `web/src/lib/components/SourceChip.svelte` (tooltip suppression edit) | component | request-response (DOM render) | same file (existing `title={...}` usages, lines 310, 327-339, 410) | exact (self-analog) |
| `web/src/lib/components/source-chip-tooltip.test.ts` (extended assertions) | test | transform (structural regex assertions over source text) | same file (existing structural + behavioral test pattern) | exact (self-analog) |
| `web/e2e/specs/14-*-external-rehearsal.spec.ts` (new) | test | request-response (Playwright e2e) | `web/e2e/specs/12-external-rehearsal.spec.ts` | exact |
| `docs/plugin-contract.md` (gap-triage republish edits) | config/docs | — | same file (existing documented sections, e.g. "Fetch", "Logging", "Describe") | exact (self-analog) |
| plugin-repo `PRD.md` (hand-off doc, written into the sibling checkout) | docs | — | no in-repo analog — first PRD this project has authored for an out-of-repo GSD project | no analog |

## Pattern Assignments

### `cmd/topos/main.go` (config, request-response) — folded todo: config-path split

**Analog:** same file, existing `configPath()` and `pluginsDir()` functions

**Current resolution pattern** (`cmd/topos/main.go:73-95`, verified read this session):
```go
func configPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "topos", "config.toml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.toml"
	}
	return filepath.Join(home, ".config", "topos", "config.toml")
}

// pluginsDir resolves cfg.Plugins.Dir relative to the running executable
// when it is not already absolute.
func pluginsDir(cfg *config.Config) (string, error) {
	if filepath.IsAbs(cfg.Plugins.Dir) {
		return cfg.Plugins.Dir, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	return filepath.Join(filepath.Dir(exe), cfg.Plugins.Dir), nil
}
```

**Argv dispatch precedent to extend** (`cmd/topos/main.go:29-45`, the shape a
`--config`/flag-parsing addition slots into):
```go
func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "serve":
		if err := runServe(); err != nil {
			fatal(err)
		}
	case "sync":
		if err := runSync(); err != nil {
			fatal(err)
		}
	default:
		usage()
		os.Exit(2)
	}
}
```

**Relative-path resolution convention to mirror for a dev config's
`plugins.dir`** — `externalPluginsDir()` (`cmd/topos/main.go:148-160`) already
applies "absolute value used verbatim; relative value resolved against
`os.Executable()`'s directory" identically to `pluginsDir()`; a `--config
<path>`/`TOPOS_CONFIG` flag should follow this same absolute-vs-relative
convention rather than inventing a third resolution rule.

**Where the new flag/env check must be inserted:** before the existing
`configPath()` call sites — `runSync()` at line ~299 (`pdir, err :=
pluginsDir(cfg)` follows a config-load call above it) and `runServe()` at
line ~354. Read the surrounding ~20 lines of each function when implementing
to find the exact `cfg, err := config.Load(configPath())`-shaped call to
gate.

---

### `Makefile` / `config.example.toml` / `docs/testing.md` (config, folded todo continued)

**Analog:** same files — no external analog needed; this is additive
convention documentation, not a new pattern to import. Read the existing
`make dev`/`make build` targets in `Makefile` and the existing example
`[sources.*.extras]` blocks in `config.example.toml` (e.g. the filesystem
block already documented) before adding a `config.dev.toml` convention, to
match existing TOML comment style and target naming.

---

### `web/src/lib/components/SourceChip.svelte` (component, request-response) — folded todo: tooltip suppression

**Analog:** same file — the five existing `title={...}` usages are the
exact touch points enumerated in the todo file
(`.planning/todos/pending/2026-08-14-suppress-native-tooltips-under-chip-popovers.md`):

```
310:    title={tooltipText}
327-329: // button's title={tooltipText} is the touch degrade for ...
339:    title={source.display_name}>{source.display_name}</span>
410:    <span class="truncate font-mono" title={source.pinned_hash}
```

**Pattern to apply:** remove `title=` where a richer app popover already
covers the same content (line 310's `tooltipText`, line 339's
`display_name`), or convert to `aria-label` where the element carries no
richer popover of its own (line 410's `pinned_hash` span) — per the todo's
own solution sketch ("Remove the `title`/`alt` attributes... or move their
text to `aria-label`").

---

### `web/src/lib/components/source-chip-tooltip.test.ts` (test, transform)

**Analog:** same file — existing structural+behavioral test pattern (house
pattern shared with `chip-edit-menu.test.ts` / `plugin-icon.test.ts`, per
this file's own header comment)

**Existing structural-assertion approach** (verified read this session,
lines 1-40): the test reads `SourceChip.svelte`'s raw source via
`readFileSync`, strips comments, and asserts exact template-literal text is
present/absent via regex — no component-mount harness exists in this repo's
vitest config (`environment: 'node'`). Extend this file with a new
structural assertion of the form "no `title=` attribute survives on the
chip's popover-bearing elements" (a negative-presence regex assertion,
mirroring the existing positive-presence assertions already in the file),
per the todo's own acceptance criterion ("Add/extend a component test
asserting the popover-bearing elements carry no `title` attribute").

---

### `web/e2e/specs/14-*-external-rehearsal.spec.ts` (test, request-response)

**Analog:** `web/e2e/specs/12-external-rehearsal.spec.ts` (12-05-PLAN.md
Task 1) — the closest possible match: an external-tier rehearsal spec
against a real, full-featured source plugin binary, not a bare
Describe/Match fixture stub.

**Structural pattern to copy** (verified read this session, header comment
+ opening ~50 lines):
```typescript
// 12-05-PLAN.md Task 1's rehearsal shape to imitate — deliberately omits
// the plugin binary from the trusted `pluginBinaries` list, naming it ONLY
// in `externalPluginBinaries` (Phase 11 D-11: a binary present in BOTH
// directories resolves as trusted, which would prove nothing here).
import { writeFileSync } from 'node:fs';
import { basename, join } from 'node:path';

import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import { mkdtempCorpus } from '../fixtures/corpus';
import type { FixtureConfigSpec } from '../fixtures/config-builder';
```

**What to change for the Drive plugin spec:** swap `topos-plugin-filesystem`
/ `FILESYSTEM_BINARY` for `topos-plugin-gdrive`; this spec cannot seed a
corpus directory or drive real Google API calls hermetically (no CI-safe
Google credentials, per RESEARCH.md "Environment Availability") — scope it
to what IS hermetically provable: discovery from the external directory,
untrusted badge, pin verification, and the named auth-state health surface
(missing-auth) using a stub/mock build mode if the plugin repo exposes one,
OR defer this spec's full authoring until the real binary exists per
RESEARCH.md's sequencing note ("sequence the e2e task after the plugin
repo's first build"). Do not invent hermetic Drive API mocking inside this
repo — that infrastructure, if any, belongs in the plugin repo.

**Supporting fixtures to reuse unmodified:**
- `web/e2e/fixtures/kernel.ts` — `test`, `expect`, `waitForFirstSync`
- `web/e2e/fixtures/config-builder.ts` — `FixtureConfigSpec`
- `web/e2e/fixtures/plugin-binaries.ts` — the `externalPluginBinaries` vs
  `pluginBinaries` distinction (Phase 11 D-09/D-11) that makes the
  external-tier proof meaningful

**Also relevant — named auth-state health pattern this spec should assert
against** (`plugins/whatsapp/health.go`, verified read this session):
```go
type healthState int

const (
	healthStateConnecting healthState = iota
	healthStateNotLinked
	healthStateLinked
	healthStateDelinked
	healthStateBanned
	healthStateExpired
	healthStateStreamReplaced
)

func (s healthState) Healthy() bool { return s == healthStateLinked }

var healthMessages = map[healthState]string{
	healthStateLinked: "",
	healthStateNotLinked: "Not linked — pair this device with WhatsApp to start syncing. ...",
	// ...
}
```
This is the exact shape D-01's "named health state" (missing/expired/revoked
auth) should surface through — a plugin-internal enum + `Health.LastError`
free text + `Match` → `codes.Unavailable`, never a new kernel-level
closed-vocabulary field. It lives in the plugin repo (out of scope to copy
verbatim there per clean-room discipline — cite it in the PRD as "mirror
this shape," not as code to paste), but the topos-side e2e spec and any
UAT checklist should assert the kernel surfaces `reachable: false` +
`last_error` verbatim for whatever the plugin reports, exactly as it does
today for WhatsApp's states.

---

### `docs/plugin-contract.md` (docs, config) — D-07 gap-triage republish

**Analog:** same file — existing documented sections ("Fetch", "Logging",
"Describe", the four-RPC shape). No external analog; this is an editorial
update to an existing published document, not new code. When
`CONTRACT-GAPS.md` entries from the plugin repo are triaged as
"documentation-fixable," locate the relevant existing section (e.g. a gap
about "where does plugin-private state live" slots into a new subsection
near "Logging" or a new top-level "Plugin-private state" section) and edit
in place — do not restructure sections unrelated to the specific gap.

---

### Plugin-repo `PRD.md` (docs, hand-off) — D-08

**No in-repo analog.** This is the first PRD authored by this project for a
sibling out-of-repo GSD project. It must cite **only**:
- `docs/plugin-contract.md`
- `proto/topos/v1/plugin.proto`
- `sdk/` module
- `plugins/mock/`
- this phase's locked decisions (D-01..D-08, Claude's Discretion items)

Do not inject kernel internals (e.g. `kernel/pluginhost` implementation
details, `ReplaceWebspaceSourceItems` internals) into the PRD prose beyond
what the published contract already documents — RESEARCH.md's own findings
(drive-wide `changes.list`, Match's full-replace semantics, `Describe`
must-not-require-credentials) are appropriate to cite in the PRD as *design
guidance*, since RESEARCH.md itself is a topos-side artifact grounding the
hand-off, not a kernel-internals leak.

## Shared Patterns

### Config-path resolution convention (absolute verbatim / relative resolved against executable dir)
**Source:** `cmd/topos/main.go:148-160` (`externalPluginsDir`), mirrored by
`pluginsDir` (`cmd/topos/main.go:86-95`)
**Apply to:** the new `--config`/`TOPOS_CONFIG` resolution in `configPath()`
and any Makefile dev-config convention — reuse the identical
absolute-vs-relative-to-executable rule rather than a third variant.

### Structural test-over-source-text (no component-mount harness)
**Source:** `web/src/lib/components/source-chip-tooltip.test.ts:1-40`
**Apply to:** any new/extended `SourceChip.svelte` assertions — this repo's
vitest config has no DOM-mount environment; string/regex assertions against
`readFileSync`'d component source are the house pattern for Svelte
component tests here.

### External-tier rehearsal spec shape (trusted-dir omission proves the tier)
**Source:** `web/e2e/specs/12-external-rehearsal.spec.ts` (full file, header
comment lines 1-17 read this session)
**Apply to:** the new Drive external-rehearsal spec — the load-bearing
technique is naming the binary ONLY in `externalPluginBinaries`, never in
`pluginBinaries`, per Phase 11 D-11.

### Named health-state taxonomy (plugin-internal enum, never a new kernel field)
**Source:** `plugins/whatsapp/health.go:5-98`
**Apply to:** citation only, in the plugin-repo PRD — describes the shape
the Drive plugin's auth-state health surfacing should mirror; not code to
copy into this repo.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| plugin-repo `PRD.md` | docs | — | First out-of-repo PRD hand-off this project has produced; no prior PRD-authoring-for-a-sibling-project artifact exists in this repo to pattern-match against. Author from the four published inputs + this phase's locked decisions directly, per D-08. |
| plugin-repo `CLAUDE.md` (D-06 rule file) | config | — | Also new — not this repo's `.claude/skills` or root `CLAUDE.md` conventions (those are for THIS project); the plugin repo's rule file is a distinct, narrower "allowed inputs" allowlist document that the topos-side planner should describe requirements for but not draft verbatim, since drafting it is itself plugin-repo-side work per D-08's boundary. |

## Metadata

**Analog search scope:** `cmd/topos/`, `web/e2e/specs/`, `web/e2e/fixtures/`,
`web/src/lib/components/`, `plugins/whatsapp/`, `docs/`,
`.planning/todos/pending/`
**Files scanned:** ~15 (targeted reads; no full-repo sweep needed — CONTEXT.md
and RESEARCH.md already named almost every touch point explicitly)
**Pattern extraction date:** 2026-08-15
