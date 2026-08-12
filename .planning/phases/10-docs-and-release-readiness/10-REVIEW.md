---
phase: 10-docs-and-release-readiness
reviewed: 2026-08-12T00:00:00Z
depth: standard
files_reviewed: 18
files_reviewed_list:
  - .github/workflows/ci.yml
  - .github/workflows/nightly.yml
  - .github/workflows/release.yml
  - CONTRIBUTING.md
  - Makefile
  - README.md
  - SECURITY.md
  - docs/plugins/README.md
  - docs/plugins/_template.md
  - docs/plugins/paperless.md
  - docs/plugins/proton.md
  - docs/plugins/signal.md
  - docs/plugins/silverbullet.md
  - docs/plugins/whatsapp.md
  - docs/releasing.md
  - docs/ss/README.md
  - scripts/check-doc-links.sh
  - scripts/sync-milestones.sh
  - web/README.md
findings:
  critical: 1
  warning: 3
  info: 3
  total: 7
status: issues_found
---

# Phase 10: Code Review Report

**Reviewed:** 2026-08-12T00:00:00Z
**Depth:** standard
**Files Reviewed:** 18
**Status:** issues_found

## Summary

This phase is almost entirely documentation and release-engineering
config (GitHub Actions workflows, `CONTRIBUTING.md`/`README.md`
restructuring, per-plugin operator docs, and two new guard scripts). Most
of the surface is internally consistent — I cross-checked every
plugin doc's match-vocabulary, referenced test name, and TOML env-var
claim against the actual plugin source and `config.example.toml`, ran
`scripts/check-doc-links.sh` directly, and validated all three workflow
YAML files parse and use option combinations (`gh release create --notes
--generate-notes`) that are actually valid together.

One finding is a genuine blocker: `README.md`'s "Configure" step tells a
new user to `export SB_URL=...` for the SilverBullet source, but nothing
in the codebase reads `SB_URL` — the actual expected variable is
`SILVERBULLET_URL` (used by both `config.example.toml` and
`docs/plugins/silverbullet.md`). A user following the README exactly, as
written for a first-time setup, gets a config-load failure that names a
variable they never saw. The remaining findings are quality/consistency
issues: a broken implicit cross-reference in `web/README.md` that the new
`check-doc-links.sh` guard cannot catch because it isn't real markdown
link syntax, duplicated release-asset lists across two workflow files
with no single source of truth, and a pre-existing `Makefile` shell-quoting
issue where `CGO_ENABLED=0` silently stops applying to the `go test` half
of most `test-portable` recipe lines.

## Critical Issues

### CR-01: README.md's SilverBullet env var name does not match what the config actually expands

**File:** `README.md:110`
**Issue:** The "Configure" section instructs the reader to set:
```bash
export SB_URL="https://silverbullet.example.lan:3000"
```
but no code anywhere in this repository reads `SB_URL`. Both
`config.example.toml:174-182` and `docs/plugins/silverbullet.md:16`
expand `base_url = "${SILVERBULLET_URL}"` — a different variable name.
`SB_URL` does not appear anywhere outside this one README line
(verified via repo-wide grep). Following the README's own instructions
verbatim leaves `SILVERBULLET_URL` unset, which — per
`config.example.toml`'s own documented validation rule — "fails config
load, naming the missing variable," producing an error that names a
variable the user was never told to set. This is exactly the "getting
started" path a new operator follows first, per the README's own framing
("Configure" step 1).
**Fix:**
```diff
-   export SB_URL="https://silverbullet.example.lan:3000"
+   export SILVERBULLET_URL="https://silverbullet.example.lan:3000"
```

## Warnings

### WR-01: web/README.md points at CONTRIBUTING.md and docs/testing.md as if they live in web/, but they're at the repo root

**File:** `web/README.md:8-10`
**Issue:**
```markdown
- **`CONTRIBUTING.md`** — the dev loop, build, and test commands for
  working on this package.
- **`docs/testing.md`** — the testing map covering this package's own
  `svelte-check`, unit, and Playwright gates.
```
Neither `web/CONTRIBUTING.md` nor `web/docs/testing.md` exists — the
real files are two directories up, at the repo root (`CONTRIBUTING.md`
and `docs/testing.md`). Because these are rendered as bold+code text
rather than markdown link syntax (`[text](target)`), `scripts/check-doc-links.sh`
never inspects them (it only matches `[...](...)`/`![...](...)`), so this
gap was not caught by the very guard this phase introduced. A reader
opening this file from inside `web/` (e.g. GitHub's directory browser, or
a local editor) has no way to tell these are relative to the repo root,
not to `web/`.
**Fix:** Either make them real relative links, or state the path explicitly:
```diff
-- **`CONTRIBUTING.md`** — the dev loop, build, and test commands for
+- [`../CONTRIBUTING.md`](../CONTRIBUTING.md) — the dev loop, build, and test commands for
   working on this package.
-- **`docs/testing.md`** — the testing map covering this package's own
+- [`../docs/testing.md`](../docs/testing.md) — the testing map covering this package's own
   `svelte-check`, unit, and Playwright gates.
```

### WR-02: nightly.yml and release.yml duplicate the published-asset list verbatim, with no shared source of truth

**File:** `.github/workflows/nightly.yml:77`, `.github/workflows/release.yml:49`
**Issue:** Both workflows independently hard-code the identical string:
```bash
ASSETS="topos plugins/topos-plugin-paperless plugins/topos-plugin-silverbullet plugins/topos-plugin-proton plugins/topos-plugin-whatsapp"
```
This is the exact kind of duplication the Makefile itself explicitly
guards against elsewhere in this same phase — e.g. `plugins-portable`'s
own comment: "The five names are written HERE ONLY... so the two
variants cannot drift apart." Here, the same five-binary list (four
plugins + kernel) is written independently in two separate CI files. If
a future phase adds a sixth published plugin, it is easy to update one
workflow and forget the other, silently shipping an incomplete nightly
or release with no error — `gh release create` will happily publish
whatever `$PUBLISH` resolves to.
**Fix:** Derive `ASSETS` from what `plugins-portable` actually built
(e.g. `ASSETS="topos $(cd bin/plugins && ls topos-plugin-* | grep -v mock | sed 's#^#plugins/#')"`),
or extract the list into a single small script/file both workflows source,
so there is exactly one place that names the published set.

### WR-03: `make test-portable`'s `CGO_ENABLED=0` only applies to the `go build` half of each line, not the paired `go test`

**File:** `Makefile:135-142`
**Issue:** Every `test-portable` recipe line except `mockstrict` is shaped:
```makefile
cd sdk && CGO_ENABLED=0 go build ./... && go test ./...
```
In POSIX shell, an inline `VAR=value` prefix scopes only to the single
command it immediately precedes — here, `go build` — not to any command
chained after it with `&&`. `go test ./...` therefore runs with whatever
`CGO_ENABLED` value is ambient in the environment (unset by default),
which on a machine with a C compiler present (true of most dev machines
and `ubuntu-latest` CI runners) resolves to `CGO_ENABLED=1` by Go's own
default-detection. This contradicts the documented, load-bearing
invariant repeated across this same phase's own docs — CONTRIBUTING.md
("the `-portable` variant additionally needs no C toolchain"),
docs/releasing.md, and the Makefile's own header comments for
`test-portable` ("cgo-free half of `test`"). Confirmed with a minimal
repro: `bash -c 'CGO_ENABLED=0 true && echo ${CGO_ENABLED:-<unset>}'`
prints `<unset>`. Notably, the `mockstrict` line is the one exception
that gets this right (`CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go
test ./...`), which shows the omission on the other six lines is an
inconsistency rather than an intentional choice. This predates this
phase's diff (the `test-portable` target itself wasn't touched by
phase 10), but it sits inside a file this phase's docs (CONTRIBUTING.md,
docs/releasing.md) now actively point readers at as the authoritative
description of the portable/cgo-free guarantee, so the docs are making a
promise this Makefile target doesn't actually keep for its test phase.
**Fix:**
```makefile
test-portable:
	CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go test ./...
	cd sdk && CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go test ./...
	cd plugins/paperless && CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go test ./...
	cd plugins/silverbullet && CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go test ./...
	cd plugins/proton && CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go test ./...
	cd plugins/mock && CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go test ./...
	cd plugins/mockstrict && CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go test ./...
	cd plugins/whatsapp && CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go test ./...
```

## Info

### IN-01: check-doc-links.sh only recognizes inline `[text](target)`/`![alt](target)` syntax

**File:** `scripts/check-doc-links.sh:129`
**Issue:** The link scan is a single grep for inline markdown link/image
syntax. It silently skips reference-style links (`[text]: target`) and,
as WR-01 shows, anything not written as an actual markdown link at all
(bold+code pseudo-references). None of the currently-tracked docs use
reference-style links, so this isn't a live false-negative today, but the
gap is worth a one-line comment so a future doc author doesn't assume the
guard is checking more than it is.
**Fix:** Add a note in the script's header comment calling out that only
inline link/image syntax is checked, or extend the grep to also catch
`^\[[^]]+\]:\s*(\S+)` reference-definition lines.

### IN-02: check-doc-links.sh's target regex truncates at the first `)` inside the link target

**File:** `scripts/check-doc-links.sh:129`
**Issue:** `grep -noE '!?\[[^]]*\]\([^)]+\)'` uses `[^)]+` for the link
target, so a target containing a literal `)` (e.g. a path like
`notes(v2).md`) would be mis-captured — the regex would stop at the
first `)`, producing a malformed target and either a false "broken link"
or, worse, silently checking the wrong (truncated) path. No file in this
repo currently has such a path, so this is latent, not active.
**Fix:** Not urgent given current repo content; if ever hit, switch to a
non-greedy match anchored on the outer bracket/paren pair, or document
the limitation inline.

### IN-03: sync-milestones.sh interpolates `$TITLE` unescaped into a jq filter string

**File:** `scripts/sync-milestones.sh:83`
**Issue:**
```bash
existing_number=$(gh api "repos/${REPO}/milestones?state=all" \
  --jq ".[] | select(.title==\"${TITLE}\") | .number" | head -1)
```
`$TITLE` is spliced directly into the jq program text. A title containing
a double quote or backslash would break the jq filter (syntax error or,
in principle, alter the filter's meaning) rather than being treated as a
literal string. This is not exploitable as a security issue — the script
is invoked manually by a maintainer with a title they typed themselves,
not from any untrusted/remote input — but it's a correctness footgun the
moment a milestone title contains a quote (e.g. `v1.0 "beta"`).
**Fix:** Pass the title as a `--arg` instead of interpolating it into the
program text:
```bash
existing_number=$(gh api "repos/${REPO}/milestones?state=all" \
  --jq --arg t "${TITLE}" '.[] | select(.title==$t) | .number' | head -1)
```

---

_Reviewed: 2026-08-12T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
