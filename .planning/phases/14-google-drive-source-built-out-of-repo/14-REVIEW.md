---
phase: 14-google-drive-source-built-out-of-repo
reviewed: 2026-08-18T00:00:00Z
depth: standard
files_reviewed: 19
files_reviewed_list:
  - cmd/topos/configpath_test.go
  - cmd/topos/main.go
  - config.dev.example.toml
  - config.example.toml
  - docs/plugin-contract.md
  - docs/testing.md
  - .gitignore
  - Makefile
  - web/e2e/specs/09-1-header-touch.spec.ts
  - web/e2e/specs/11-binary-changed-repin.spec.ts
  - web/e2e/specs/11-external-tier-badge.spec.ts
  - web/e2e/specs/12-external-rehearsal.spec.ts
  - web/e2e/specs/12-tooltip-precedence.spec.ts
  - web/e2e/specs/12-zero-match-diagnostic.spec.ts
  - web/e2e/specs/13-manifest-unverified.spec.ts
  - web/e2e/specs/13-shadowed-advisory.spec.ts
  - web/e2e/specs/14-gdrive-external-rehearsal.spec.ts
  - web/src/lib/components/SourceChip.svelte
  - web/src/lib/components/source-chip-pill.test.ts
  - web/src/lib/components/source-chip-tooltip.test.ts
findings:
  critical: 0
  warning: 2
  info: 4
  total: 6
status: issues_found
---

# Phase 14: Code Review Report

**Reviewed:** 2026-08-18T00:00:00Z
**Depth:** standard
**Files Reviewed:** 19
**Status:** issues_found

## Summary

This phase does three things: (1) threads an explicit `--config`/`TOPOS_CONFIG`
config-path precedence chain through `cmd/topos/main.go`, adding a
per-checkout dev config (`config.dev.example.toml`, `make dev-config`) so
worktree kernels stop colliding on the operator's production config; (2)
replaces `SourceChip.svelte`'s native `title` tooltips with an
`aria-describedby`-wired `sr-only` description span, repointing ~10 Playwright
specs from `toHaveAttribute('title', …)` onto `toHaveAccessibleDescription(…)`;
and (3) adds a hermetic, credential-free Playwright rehearsal
(`14-gdrive-external-rehearsal.spec.ts`) plus a `make gdrive-external-rehearsal`
target for a genuinely out-of-repo Google Drive plugin binary, and republishes
`docs/plugin-contract.md` with several new prose sections.

The config-path precedence logic (`resolveConfigPath`, `parseConfigFlag`) is
correct and well covered by `configpath_test.go`'s nine precedence cases. The
`SourceChip.svelte` diff is small, focused, and internally consistent with
every spec/unit-test file it touches — I checked the rest of `web/e2e/specs/`
for any stray `title`-attribute assertion against a source chip that this
phase's file list might have missed, and found none; the only other repo hit
for `toHaveAttribute('title', …)` in the e2e suite is an unrelated locator
row in `09-picker-groups.spec.ts`. No BLOCKER-class defects (crashes,
security gaps, data loss, or broken tests) were found in the reviewed diff.

Two WARNING-level quality issues and four INFO-level nits are recorded below:
a genuine (documented-as-deliberate, but still real) accessibility regression
for touch-only users, and a Makefile duplication that breaks this same file's
own "define the binary/build list in exactly one place" discipline.

## Warnings

### WR-01: Touch-only users below 768px lose all access to chip health detail, with no interim mitigation

**File:** `web/src/lib/components/SourceChip.svelte:314-320` (also
`web/src/lib/components/source-chip-pill.test.ts:206-229`,
`web/e2e/specs/09-1-header-touch.spec.ts:134-165`)
**Issue:** The native `title` attribute that served as the long-press touch
degrade for chip health detail (09.1-04-PLAN.md R2) is removed and replaced
solely with `aria-describedby` → an `sr-only` span. That is reachable by
assistive technology (screen readers) and by mouse/keyboard hover-focus (via
the still-present `Tooltip`), but a touch user below the 768px breakpoint who
is *not* running a screen reader now has **no path at all** to the health
sentence — no hover, no keyboard focus, and (as of this change) no
long-press title either. The code comments and both updated test files
correctly document this as a known, deliberate deviation from UI-SPEC §G1,
recorded in `14-02-SUMMARY.md`/`WINDOWS.md`, so this isn't an oversight —
but it is still a real, shipped accessibility regression for sighted
touch-only users (e.g. a phone/tablet operator), and the phase ships no
interim mitigation (e.g. a tap-to-reveal popover, or falling back to `title`
specifically below the 768px breakpoint while keeping `aria-describedby` for
AT users).
**Fix:** At minimum, confirm the WINDOWS.md follow-up item has a phase
assigned rather than sitting as an open-ended ledger entry; better, restore
reachability with a media-query-scoped affordance — e.g. render the
`sr-only` span's content in a tap-triggered popover below `768px`, or keep a
conditional `title` only under `max-md:` conditions so desktop/AT users get
the new description model while touch-only phone users keep a working
degrade.

### WR-02: `gdrive-external-rehearsal` duplicates `e2e`'s fixture-build steps instead of sharing them, breaking this Makefile's own "define once" discipline

**File:** `Makefile:309-357`
**Issue:** `gdrive-external-rehearsal` re-types the entire fixture-build
sequence `e2e` already defines — `mkdir -p bin/plugins`, the three
`go build -o bin/plugins/topos-plugin-{mock,mockstrict,filesystem}` lines,
`$(MAKE) external-demo`, the `$(MANIFEST_GEN_E2E)` + `CGO_ENABLED=0 go build
-ldflags … -o bin/topos ./cmd/topos` manifest/kernel build, and the
Playwright browser-install line — verbatim, in a second recipe. This is the
exact drift risk this same file explicitly calls out and defends against
elsewhere for near-identical lists: `MANIFEST_PLUGIN_BINARIES` /
`MANIFEST_PLUGIN_BINARIES_PORTABLE` / `MANIFEST_E2E_BINARIES` are each
"named, in exactly one place... [so] the two variants cannot drift apart",
and `test-portable`'s own comment states "This is the ONLY place that
module list is written; `test` below delegates to this target rather than
duplicating it". `gdrive-external-rehearsal` does not follow that pattern:
a future edit to `e2e`'s fixture-build steps (e.g. adding a fourth plugin
binary, or changing the manifest invocation) has no build-time signal
forcing the second copy to be updated, so `gdrive-external-rehearsal` can
silently start testing the Drive binary against a stale or incomplete
fixture kernel while `e2e` itself is green.
**Fix:** Factor the shared steps into their own prerequisite target (e.g.
`e2e-fixture-kernel: `, containing everything through the `bin/topos` build)
and have both `e2e` and `gdrive-external-rehearsal` depend on it, mirroring
how `build`/`build-portable` already delegate to `plugins`/`plugins-portable`
rather than re-typing the plugin build list.

## Info

### IN-01: `TOPOS_GDRIVE_BIN`'s default bakes in one developer's personal directory layout

**File:** `Makefile:331`
**Issue:** `TOPOS_GDRIVE_BIN ?= $(HOME)/projects/davison/topos-plugin-gdrive/topos-plugin-gdrive`
assumes every future contributor keeps a sibling `topos-plugin-gdrive`
checkout under `~/projects/davison/` — a path shape specific to the current
maintainer's own machine layout, committed into a shared build file. It's
fully overridable (`TOPOS_GDRIVE_BIN=...` on the command line or as an env
var) and the target/spec both skip loudly rather than silently misbehaving
when the path doesn't resolve, so this is a hygiene nit, not a functional
defect.
**Fix:** Consider defaulting to unset (forcing an explicit, self-documenting
override on every machine) or moving the personal-path convenience into a
local, gitignored `Makefile.local`-style include, if one exists in this
project's conventions.

### IN-02: New CLI flag-parsing path (`parseConfigFlag`) has no direct test coverage

**File:** `cmd/topos/main.go:67-76`, `cmd/topos/configpath_test.go`
**Issue:** `configpath_test.go` thoroughly covers `resolveConfigPath`'s
precedence chain (9 cases), but the new `parseConfigFlag` wrapper — which
handles the actual `os.Args` parsing, both `--config value` and
`--config=value` forms, an unrecognised-flag error path, and a
missing-value error path, each ending in `usage()` + `os.Exit(2)` — has no
test at all. `os.Exit` calls make this awkward to unit-test directly without
refactoring (e.g. extracting an exit-code return instead of calling
`os.Exit` inline), which may be why it was skipped, but the happy-path
`--config value` / `--config=value` parity claimed in the doc comment
("Both … forms are accepted") is currently unverified by any test.
**Fix:** At minimum, add a table test asserting `parseConfigFlag("serve",
[]string{"--config", "x"})` and `parseConfigFlag("serve", []string{"--config=x"})`
both return `"x"`, without needing to touch the `os.Exit` error paths.

### IN-03: `make dev-config`'s `sed` delimiter and unquoted `$(CURDIR)` interpolation are not defensive against unusual checkout paths

**File:** `Makefile:369-378`, `Makefile:90`
**Issue:** `sed 's|@CHECKOUT@|$(CURDIR)|g'` uses `|` as the substitution
delimiter while interpolating `$(CURDIR)` as the replacement text; a
checkout path that itself contains a literal `|` would break the
substitution. Separately, `DEV_KERNEL_CMD`'s `--config $(DEV_CONFIG)` is
unquoted in the shell command line it expands into, so a checkout path
containing whitespace (e.g. `~/My Projects/topos`) would be split into
multiple arguments and fail to resolve. Both are low-likelihood on a typical
Linux dev machine, but this Makefile otherwise invests heavily in
worktree/path safety (the whole point of this phase's `DEV_CONFIG` work), so
the gap is worth noting.
**Fix:** Quote `"$(DEV_CONFIG)"` in `DEV_KERNEL_CMD`, and/or pick a `sed`
delimiter unlikely to appear in a filesystem path (already true for `|` in
practice, so this is the lower-priority half of the two).

### IN-04: A CLI parse error on `--config` prints two separate usage blocks

**File:** `cmd/topos/main.go:67-76`
**Issue:** On a parse error (e.g. `topos serve --config` with no value, or
an unrecognised flag), Go's `flag` package first prints its own
auto-generated `Usage of serve:` block (via `fs.SetOutput(os.Stderr)` and
the FlagSet's default usage) naming just the `-config` flag, and then this
function calls the package's own `usage()`, printing a second, differently
formatted usage block. The doc comment states this is intentional ("the
operator sees both the specific parse error and the general usage line"),
but the result is two stacked, redundantly-formatted usage outputs on one
error, which reads as noisy rather than helpful.
**Fix:** Consider setting `fs.Usage = func() {}` (suppressing the FlagSet's
own auto usage block) so only this package's own `usage()` prints, while
still relying on `flag`'s own `"flag provided but not defined"` /
`"flag needs an argument"` error line (printed separately, before `Parse`
returns) to name the specific problem.

---

_Reviewed: 2026-08-18T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
