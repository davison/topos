# Phase 11: External Plugins & the Trust Boundary - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-12
**Phase:** 11-External Plugins & the Trust Boundary
**Areas discussed:** Hash pinning & upgrades, Warning & badge UX, External dir & collisions, Extras config shape

---

## Hash pinning & upgrades

| Option | Description | Selected |
|--------|-------------|----------|
| In config TOML | Pin in config.toml via hot-apply path; visible, survives backup/restore; keeps config the single source of truth | ✓ |
| Kernel-owned store | Index DB or sidecar file; cleaner config but a second persistence surface | |
| You decide | | |

**User's choice:** In config TOML

| Option | Description | Selected |
|--------|-------------|----------|
| Per binary | One pin per binary name; all instances share it | ✓ |
| Per instance | Each source entry carries its own hash; duplicated values | |
| You decide | | |

**User's choice:** Per binary

| Option | Description | Selected |
|--------|-------------|----------|
| UI re-pin action | Named "binary changed" health state + per-chip "Trust updated binary" action showing new hash | ✓ |
| Config edit only | Health state names it; user edits pinned hash by hand | |
| Remove and re-add | Re-triggers add-time warning + pin, destroys instance config | |

**User's choice:** UI re-pin action

| Option | Description | Selected |
|--------|-------------|----------|
| External only | Trusted-dir binaries launch unpinned; directory is their provenance | ✓ |
| Pin both tiers | Uniform mechanics but every dev rebuild is a mismatch | |
| You decide | | |

**User's choice:** External only

---

## Warning & badge UX

| Option | Description | Selected |
|--------|-------------|----------|
| Confirm step | Interstitial: what untrusted means, binary name + hash, confirm button | ✓ (modified) |
| Inline banner | Warning banner in the add form, no separate step | |
| Typed acknowledgment | Type the plugin name, rm-rf-style friction | |

**User's choice:** Confirm step, **plus** a type-the-plugin-name box — "just for accidental click protection", not ceremony.

| Option | Description | Selected |
|--------|-------------|----------|
| Glyph overlay | Warning glyph on the plugin identity icon in the 44px pill; tooltip spells it out | ✓ |
| Chip outline/tint | Whole-chip border/tint; louder but collides with health/selected states | |
| Suffix badge | Separate badge element; costs pill width | |

**User's choice:** Glyph overlay

| Option | Description | Selected |
|--------|-------------|----------|
| Inline badges | External plugins in the same catalog section, glyph + "untrusted" label per row | ✓ |
| Separate section | Third picker section for external/untrusted | |
| You decide | | |

**User's choice:** Inline badges

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, in menu/tooltip | Short copyable hash in per-chip menu/tooltip; full at add-confirm and re-pin | ✓ |
| Add-time only | Hash shown only during add/re-pin | |
| Not user-facing | Config-file detail only | |

**User's choice:** Yes, in menu/tooltip

---

## External dir & collisions

| Option | Description | Selected |
|--------|-------------|----------|
| Platform data dir | Per-OS helper: XDG_DATA_HOME (Linux) / Application Support (macOS) / %LOCALAPPDATA% (Windows); overridable | ✓ |
| No default — explicit only | Off until [plugins] external_dir set | |
| Exe-relative sibling | plugins-external next to trusted dir; clobbered by upgrades | |

**User's choice:** Platform data dir
**Notes:** User asked whether a Windows equivalent to XDG exists or the project is "already too far from windows compliance". Clarified: %LOCALAPPDATA% is the equivalent; Go stdlib lacks a UserDataDir so a ~10-line per-OS helper is needed; the project is Linux-anchored (D-Bus keyring, desktop chat DBs) so the stance is "portable resolution without committing to Windows support".

| Option | Description | Selected |
|--------|-------------|----------|
| Same prefix | Reuse the exact topos-plugin-* DiscoverBinaries convention | ✓ |
| Any executable | Any executable is a candidate; identity needs a new source of truth | |

**User's choice:** Same prefix

| Option | Description | Selected |
|--------|-------------|----------|
| Trusted shadows external | Trusted wins; external copy ignored with loud named log line | ✓ |
| Both offered, tier-qualified | Both in picker with tier field on instances | |
| Fail loudly at startup | Collision is a config error | |

**User's choice:** Trusted shadows external

---

## Extras config shape

| Option | Description | Selected |
|--------|-------------|----------|
| [sources.X.extras] table | Dedicated sub-table; typed top level keeps typo detection; rewrite round-trips one opaque map | ✓ |
| Free-form top-level keys | Unknown keys on [sources.X] pass through; typos silently become extras | |
| You decide | | |

**User's choice:** [sources.X.extras] table

| Option | Description | Selected |
|--------|-------------|----------|
| Strings only | map[string]string end to end; plugins parse from strings | ✓ |
| Full TOML values | Richer authoring; heavier wire contract | |

**User's choice:** Strings only

| Option | Description | Selected |
|--------|-------------|----------|
| Scrub + expand + disclose | Minimal allowlisted plugin env; uniform ${VAR} expansion; confirm step lists env values handed over; no auto-filled plugin defaults | ✓ |
| Scrub all, expand trusted only | Untrusted plugins get extras verbatim; breaks SRC-05's stated credential path | |
| Defer scrubbing | Expand now, leave os.Environ() inheritance for later | |

**User's choice:** Scrub + expand + disclose
**Notes:** Question was asked twice. First pass: user flagged the threat unprompted — "A malicious plugin could hoover up any number of values from well known env vars and phone home with them" — and corrected a stale premise ("the v1.0 wrapper script and .env file are long dead"). Verified in code that `kernel/pluginhost/host.go:449` passes full `os.Environ()` to every plugin regardless of how the kernel env is populated; re-asked with corrected premise and the scrub option was chosen. Env scrubbing is thereby explicitly in Phase 11 scope.

| Option | Description | Selected |
|--------|-------------|----------|
| Describe-declared + free-form | Contract's Describe declares expected keys for labeled inputs; free-form editor covers the rest | ✓ |
| Free-form editor only | Generic key/value table; setup UX degrades to "consult the README" | |
| Describe-declared only | Tightest UX; strands undeclared keys | |

**User's choice:** Describe-declared + free-form

---

## Claude's Discretion

- Hash algorithm and TOML pin-map format
- "Binary changed" health-state name and interstitial copy
- Whether mock/mockstrict exclusions apply to the external tier
- Criterion-5 proof binary choice
- Exact env allowlist contents; discovery timing

## Deferred Ideas

- None new. Reviewed todo left pending: Signal schema-version verify-and-accept tooling (Signal maintenance, unrelated to this phase).

## Todo Cross-Reference

- Matched (score 0.6): `2026-08-05-signal-schema-version-verify-and-accept-tooling.md` — user chose **Leave it pending**.
