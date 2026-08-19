---
phase: 15-installed-instance-dev-isolation
reviewed: 2026-08-19T00:17:32Z
depth: standard
files_reviewed: 21
files_reviewed_list:
  - scripts/install.sh
  - scripts/install-smoke.sh
  - scripts/uninstall.sh
  - scripts/install-signal.sh
  - scripts/simultaneity-smoke.sh
  - scripts/smoke-lib.sh
  - scripts/dev-guard-smoke.sh
  - cmd/topos/main.go
  - cmd/topos/pluginsdir_test.go
  - cmd/topos-devguard/main.go
  - cmd/topos-devguard/devguard_test.go
  - Makefile
  - config.dev.example.toml
  - web/vite.config.ts
  - .github/workflows/release.yml
  - .github/workflows/nightly.yml
  - docs/install.md
  - docs/releasing.md
  - docs/testing.md
  - docs/plugins/signal.md
  - CONTRIBUTING.md
  - README.md
findings:
  critical: 0
  warning: 2
  info: 4
  total: 6
status: issues_found
---

# Phase 15: Code Review Report

**Reviewed:** 2026-08-19T00:17:32Z
**Depth:** standard
**Files Reviewed:** 21
**Status:** issues_found

## Summary

The install/uninstall/devguard/isolation machinery itself is careful and
well-defended: the checksums.txt path allowlist correctly rejects
traversal shapes (`plugins/../../../x` is caught before any file is
created, matching the smoke test), the uninstaller's removal set really
is closed (non-recursive `rmdir`, exact `topos-plugin-*` glob scoped to
direct children only, no path outside `$PREFIX` is ever named), the
installer never escalates privileges and stages+verifies before any
placement begins, and `topos-devguard`'s `containedIn` correctly decides
containment on cleaned path components rather than string prefixes (the
`topos` vs `topos-extra` sibling case is handled right). I did not find
a working traversal, injection, or privilege-escalation path in any of
the reviewed shell or Go code.

Two real defects are worth fixing before this ships: `docs/install.md`
actively contradicts the shipped `INST-02` "latest release" feature
(README.md, the Makefile, and `install.sh`'s own header all agree the
feature exists; `docs/install.md`'s "Installing a release" section
still says `VERSION` is required), and `cmd/topos-devguard`'s
containment check for a relative `[sources.*] path` uses the wrong base
directory in the one documented use case (`make dev
DEV_CONFIG=<path-outside-the-checkout>`) where it matters — the config
file's own directory, not the directory the kernel subprocess actually
runs from and resolves relative source paths against. The remaining
items are lower-severity robustness/documentation notes.

## Warnings

### WR-01: docs/install.md contradicts the shipped latest-release feature

**File:** `docs/install.md:18-26`
**Issue:** This section still reads:

```
```sh
make install VERSION=1.1.0
```

`VERSION` names the release tag, with or without the leading `v`
(`1.1.0` and `v1.1.0` are the same release). It is required — there is
no implicit "latest" yet.
```

This is factually wrong against the code in this same phase.
`scripts/install.sh`'s own header states "With no version argument, the
latest published STABLE release is resolved (INST-02)"; `Makefile`'s
`install` target comment lists `make install` (no VERSION) as the first
of "Two first-class forms"; and `README.md:57` correctly documents
`make install` as resolving "the latest published stable release". A
reader of `docs/install.md` alone — which is the document this phase's
own doc set points readers at for "the full treatment" — will believe
`VERSION` is mandatory and never discover the no-argument latest-release
path this phase built and gave its own smoke-test cases
(`install-smoke.sh`'s "latest-resolution validator" and "latest-resolution
end to end" cases).
**Fix:** Update the "Installing a release" section to document both
forms, matching `README.md`'s phrasing, e.g.:

```markdown
## Installing a release

```sh
make install                 # latest published stable release
make install VERSION=1.1.0   # a specific release, with or without the leading v
```

With no `VERSION`, `make install` resolves and installs the latest
published **stable** release (never a nightly or prerelease) by
following the releases/latest redirect and validating the URL it lands
on. `VERSION` names an exact release tag when you want to pin to one.
```

### WR-02: devguard's relative source-path containment check uses the wrong base directory

**File:** `cmd/topos-devguard/main.go:214-222` (candidate construction), compare `kernel/pluginhost/host.go:1020` (`exec.Command(binPath)` sets no `cmd.Dir`)
**Issue:** For every source, the guard computes the containment
candidate as `absolutize(src.Path, configDir)` — i.e. a relative
`path` value is resolved against the **config file's own directory**.
But the kernel launches every plugin subprocess with no `cmd.Dir`
override (`kernel/pluginhost/host.go:1020`), so a plugin subprocess
inherits the **kernel process's own working directory** — for `make
dev`, that is wherever `go run ./cmd/topos-devguard`/`go run
./cmd/topos serve` were actually invoked from (the Makefile always runs
them from the repo root), not the directory the `--config` file happens
to live in. `docs/plugins/signal.md`'s own note that "plugins/signal
resolves its own Path field's leading `~` itself, not the kernel"
already establishes that a source's `path` is resolved by the plugin,
not the kernel/guard — the guard's own `expandHome` doc comment says
the same for `~`. The guard silently applies the *config-relative*
base to a value that will actually be resolved *process-cwd-relative*
by the real subprocess.

This is invisible in the default flow because the documented dev
convention always writes an absolute, `@CHECKOUT@`-prefixed source
`path` (`config.dev.example.toml`'s "per-checkout plugin stores"
section) — so `configDir` and the real cwd never diverge in practice
*for the generated template*. But `docs/testing.md`'s own "The real
config and the dev config" section documents `make dev
DEV_CONFIG=<path>` as a supported way to point the dev loop at *any*
other config, including one that lives in a different directory from
the checkout root. If that config declares a **relative** source
`path` (nothing in the guard or the kernel rejects one), the guard's
containment verdict is computed against a base directory the real
subprocess will never actually use — which can produce either a false
clear (the guard says "OK" for a path that, once resolved by the
plugin against the real cwd, actually lands inside the protected
config/state root) or a spurious refusal, in the one code path whose
entire job is to catch exactly this class of mistake (ISOL-01).
**Fix:** Either (a) refuse (independent of containment) whenever a
source's `path` is relative — the guard cannot correctly evaluate it
without knowing the real launch cwd, so a relative source `path` should
be treated as "cannot verify, refuse by name" rather than silently
checked against an approximation; or (b) resolve relative source paths
against the guard's own working directory (`os.Getwd()`) rather than
`configDir`, since `make dev` always invokes both the guard and the
kernel from the same cwd. Document whichever choice is made next to the
`expandHome` comment that already flags this exact plugin-vs-kernel
resolution split.

## Info

### IN-01: devguard's containment check does not resolve symlinks

**File:** `cmd/topos-devguard/main.go:99-110` (`containedIn`)
**Issue:** `containedIn` compares `filepath.Clean`ed, absolutized
string paths only — it never calls `filepath.EvalSymlinks`. If any
path component between a declared candidate and the topos-owned roots
is a symlink (e.g. a dotfile-manager-stowed `~/.config` tree, or a bind
mount), a candidate that textually looks isolated can still resolve, at
open time, to a location inside the protected root — and the reverse
(a candidate that textually looks like it's inside a root but is
actually a symlink pointing elsewhere) is also possible. The doc
comment for `containedIn` only promises "never on string prefixes," not
symlink-aware resolution, so this isn't a broken promise, but it is a
real gap against the guard's stated purpose (preventing a dev run from
reaching the installed instance's real files).
**Fix:** Consider `filepath.EvalSymlinks` on both the candidate and the
root before comparing, falling back to the un-resolved path when
`EvalSymlinks` fails (e.g. path doesn't exist yet) — mirroring the
fail-open-to-"not a directory" discipline `main.go`'s own `isDir` already
uses elsewhere in this phase.

### IN-02: default release base URL is a duplicated magic string

**File:** `scripts/install.sh:62`, `:108`, `:117`
**Issue:** `"https://github.com/davison/topos/releases"` is written out
three separate times: once inside `resolve_latest_effective_url`'s
`curl` invocation, once again in `main`'s failure message for that same
call, and a third time as the `TOPOS_RELEASE_BASE_URL` default
assignment a few lines later. All three currently agree, but nothing
enforces that — a future edit to one (e.g. a repo rename) that misses
the other two would silently resolve against the wrong host in one of
the three call sites.
**Fix:** Assign `TOPOS_RELEASE_BASE_URL="${TOPOS_RELEASE_BASE_URL:-https://github.com/davison/topos/releases}"` once, at the top of `main`, before `resolve_latest_effective_url` is ever called, and have `resolve_latest_effective_url`/the failure message both read the now-set variable instead of re-embedding the default.

### IN-03: ephemeral-port helpers have an inherent bind/release/rebind race

**File:** `scripts/smoke-lib.sh:11-19` (`smoke_free_port`), reused by `scripts/dev-guard-smoke.sh:97-105`
**Issue:** `smoke_free_port` binds an OS-assigned port, reads it back,
then closes the socket and returns just the number — there is a window
between the `s.close()` and whatever later process binds that same
port number in which another process on the machine could grab it
first. This is a well-known TOCTOU limitation of "ask the OS for a free
port, then let it go" and is unlikely to bite on a single-user desktop
CI/dev box, but it is the one source of inherent (not code-bug) flake
risk in these hermetic gates, worth a one-line note since the header
comments otherwise document every other safety property explicitly.
**Fix:** No action required; consider a short comment acknowledging the
known TOCTOU window so a future flaky run isn't mistaken for a new
regression.

### IN-04: plugin-name allowlist regex permits a leading hyphen

**File:** `scripts/install.sh:173` (`^[a-z0-9-]+$`)
**Issue:** The manifest-line allowlist for `plugins/<name>` accepts any
run of lowercase letters, digits, and hyphens, including a name that is
entirely hyphens or starts with one (e.g. `plugins/-x`). No exploitable
path was found — every place `$rel`/`$name` is later used is always
concatenated onto an absolute directory prefix (`$STAGE/...`,
`$PLUGINS_DIR/...`) before being passed to `curl -o`, `cp`, `chmod`, or
`mv`, so the resulting argument never itself begins with `-`. Still,
this is one regex character class away from being airtight against a
name that could be misread as a flag if a future edit ever passes
`$name` (rather than the resolved absolute `$dest`) to a command.
**Fix:** Tighten to `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$` (or equivalent)
so a bare leading/trailing hyphen is rejected, matching the stricter
shape real plugin binary names in this repo already have
(`topos-plugin-<name>`).

---

_Reviewed: 2026-08-19T00:17:32Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
