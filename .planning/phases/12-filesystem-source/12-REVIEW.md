---
phase: 12-filesystem-source
reviewed: 2026-08-14T00:00:00Z
depth: standard
files_reviewed: 56
files_reviewed_list:
  - config.example.toml
  - docs/api.md
  - docs/plugin-contract.md
  - docs/plugins/filesystem.md
  - docs/plugins/README.md
  - docs/testing.md
  - go.work
  - kernel/config/config_test.go
  - kernel/config/store_test.go
  - kernel/config/types.go
  - kernel/httpapi/config_test.go
  - kernel/httpapi/contract_test.go
  - kernel/httpapi/fsopen.go
  - kernel/httpapi/fsopen_test.go
  - kernel/httpapi/item.go
  - kernel/httpapi/item_test.go
  - kernel/httpapi/routes.go
  - kernel/httpapi/stream.go
  - kernel/httpapi/stream_test.go
  - kernel/pluginhost/env_test.go
  - kernel/pluginhost/extras_test.go
  - kernel/pluginhost/host.go
  - Makefile
  - plugins/filesystem/assets/icon.svg
  - plugins/filesystem/classify.go
  - plugins/filesystem/classify_test.go
  - plugins/filesystem/fetch.go
  - plugins/filesystem/fetch_test.go
  - plugins/filesystem/go.mod
  - plugins/filesystem/go.sum
  - plugins/filesystem/health_test.go
  - plugins/filesystem/item.go
  - plugins/filesystem/item_test.go
  - plugins/filesystem/main.go
  - plugins/filesystem/plugin.go
  - plugins/filesystem/readonly_test.go
  - plugins/filesystem/render.go
  - plugins/filesystem/render_test.go
  - plugins/filesystem/scope.go
  - plugins/filesystem/scope_test.go
  - plugins/filesystem/walk.go
  - plugins/filesystem/walk_test.go
  - README.md
  - web/e2e/e2e-builtins.d.ts
  - web/e2e/fixtures/config-builder.ts
  - web/e2e/specs/12-external-rehearsal.spec.ts
  - web/e2e/specs/12-filesystem-add-source.spec.ts
  - web/e2e/specs/12-filesystem-recursion.spec.ts
  - web/e2e/specs/12-filesystem-tracer.spec.ts
  - web/src/lib/api.ts
  - web/src/lib/components/connection-checkbox.test.ts
  - web/src/lib/components/ConnectionForm.svelte
  - web/src/lib/components/open-in-source-local-exec.test.ts
  - web/src/lib/components/OpenInSource.svelte
  - web/src/lib/components/save-state.test.ts
  - web/src/lib/components/ui/checkbox/checkbox.svelte
  - web/src/lib/components/ui/checkbox/index.ts
  - web/src/lib/plugin-fields.test.ts
  - web/src/lib/plugin-fields.ts
findings:
  critical: 2
  warning: 1
  info: 0
  total: 3
status: issues_found
---

# Phase 12: Code Review Report

**Reviewed:** 2026-08-14T00:00:00Z
**Depth:** standard
**Files Reviewed:** 56
**Status:** issues_found

## Summary

Phase 12 adds the filesystem source plugin (read-only folder indexing with
recursion, glob-based scope, per-preview-kind `Fetch`), the kernel-side
`file://` deep-link rewrite to a loopback `POST /api/items/{id}/open`
route that execs `xdg-open`, a checkbox connection-form field kind, and an
external-plugin-tier rehearsal for the new plugin. The code is unusually
well-documented and the test suite is extensive (AST guards for
read-only-ness and route scoping, negative controls on the read-only
scanner, symlink/`..`-escape unit tests, an external-tier rehearsal spec).
Path-traversal-via-`..`-segments is correctly closed on both the kernel
and plugin sides, and the read-only guarantee is mechanically enforced.

Two BLOCKER-level defects were found, both in the two areas the phase's
own threat model explicitly calls the sharpest surfaces (the loopback
`xdg-open` exec route, and path containment against the configured
root):

1. The kernel's `xdg-open` invocation is wired to the HTTP request's own
   `context.Context`, which Go cancels as soon as the handler returns —
   this races (and, for many file associations, reliably loses against)
   the just-launched application, silently killing it moments after
   "opening" it. No existing test can catch this: the unit tests stub the
   `Opener` entirely, and the e2e suite explicitly, and reasonably,
   declines to assert on real desktop-handler behavior — but this bug is
   a kernel-side process-lifecycle defect, not a desktop-environment
   variance the e2e suite was right to exclude.
2. The path-containment re-validation that runs immediately before a byte
   is read (`plugins/filesystem/fetch.go`, via `item.go`'s `resolvePath`)
   and immediately before `xdg-open` is exec'd (`kernel/httpapi/fsopen.go`)
   is purely lexical (`filepath.Join` + `strings.HasPrefix`) and never
   calls `filepath.EvalSymlinks` — unlike the Match-time walk, which does.
   A file indexed as legitimate can later be swapped for a symlink
   pointing outside the configured root (a realistic TOCTOU on a shared
   or network-writable mount) and both routes will follow it, contradicting
   the documented guarantee that the open route "re-validates the joined
   path stays inside the configured root before ever exec'ing anything."

One WARNING (a related, non-security correctness bug in the same
symlink-containment code, opposite direction: legitimate in-tree symlinks
under a symlinked root are silently dropped from the corpus).

## Critical Issues

### CR-01: `xdg-open` subprocess is killed by HTTP request-context cancellation

**File:** `kernel/httpapi/fsopen.go:27-40` (`newXDGOpener`), invoked from `FilesystemOpenHandler` at `kernel/httpapi/fsopen.go:98`

**Issue:** `newXDGOpener` builds the child process with
`exec.CommandContext(ctx, "xdg-open", path)`, where `ctx` is the
*HTTP request's own* `context.Context` (`FilesystemOpenHandler` sets
`ctx := r.Context()`). Per the documented `net/http.Request.Context()`
contract, a per-request context is canceled "when the ServeHTTP method
returns" — and Go's `net/http` server implementation cancels it
essentially synchronously with the handler function returning (it calls
`cancelCtx()` immediately after `ServeHTTP` completes for that request).
`FilesystemOpenHandler` calls `opener(ctx, full)` (which only `Start()`s
the process and returns) and then immediately calls `WriteJSON` and
returns — so the request context is canceled within microseconds of the
child process being started.

`exec.CommandContext` installs a watchdog goroutine (started inside
`Cmd.Start()`) that races `ctx.Done()` against the process's own exit
(observed only once `Cmd.Wait()` — called here in a background goroutine
— actually returns, i.e. once the OS process has genuinely exited). If
`ctx.Done()` fires first, the default `Cmd.Cancel` behavior sends the
process a `Kill` (SIGKILL). Since the handler's background goroutine only
calls `cmd.Wait()` to *reap* the process (it does not, and structurally
cannot, make the process exit any faster), the almost-immediate context
cancellation will race — and very often lose — against xdg-open actually
handing off to (and, for handlers that `exec()` in place of forking,
*becoming*) the target application. The practical effect: the desktop
application launches and is then SIGKILLed a few milliseconds later,
after the kernel has already returned `200 {"opened": true}` to the
browser. This is not a rare race for any file-association handler that
doesn't self-daemonize away from being a descendant of the `xdg-open`
invocation (a very common shape).

The code's own comment ("T-12-05: the child is reaped by a background
wait") shows the *intent* was correctly "don't block the HTTP response on
the child's exit" — but tying the child's `exec.CommandContext` context to
the request achieves the opposite: it makes the child's survival
contingent on the request's own (very short) lifetime.

This is untested: `kernel/httpapi/fsopen_test.go` stubs `Opener` entirely
(by design, per its own doc comment), and `docs/testing.md` explicitly
scopes real `xdg-open` behavior out of the e2e suite ("Whether `xdg-open`
genuinely hands a file to the desktop's own handler is a live,
machine-dependent fact a hermetic browser harness cannot assert on") —
reasonably, for desktop-environment variance, but this bug reproduces
regardless of desktop environment; it is a pure Go process-lifecycle
defect in the kernel's own code.

**Fix:** Decouple the child process's lifetime from the request context —
use `context.Background()` (optionally with its own generous, fixed
timeout unrelated to the request) for `exec.CommandContext`, never
`r.Context()`:

```go
func newXDGOpener(logger hclog.Logger) Opener {
	return func(_ context.Context, path string) error {
		// Deliberately NOT the caller's context: an HTTP request context is
		// canceled the instant the handler returns, which would otherwise
		// race (and often lose against) xdg-open actually handing off to
		// the target application. The subprocess's lifetime must outlive
		// this one request.
		cmd := exec.Command("xdg-open", path)
		if err := cmd.Start(); err != nil {
			return err
		}
		go func() {
			if err := cmd.Wait(); err != nil {
				logger.Warn("filesystem open: xdg-open exited with error", "path", path, "error", err.Error())
			}
		}()
		return nil
	}
}
```

---

### CR-02: Path-containment re-validation is lexical-only — TOCTOU symlink escape at Fetch/Open time

**Files:**
`kernel/httpapi/fsopen.go:90-96` (the kernel's own re-resolution before exec'ing `xdg-open`),
`plugins/filesystem/item.go:70-77` (`resolvePath`, called from `plugins/filesystem/fetch.go:71`'s `fetchByKind` before any file is opened)

**Issue:** Both the kernel-side open-route containment check and the
plugin-side `Fetch`-time containment check are purely **lexical**:

```go
// kernel/httpapi/fsopen.go
full := filepath.Join(root, filepath.FromSlash(it.SourceID))
if full != root && !strings.HasPrefix(full, root+string(filepath.Separator)) { ... }
```
```go
// plugins/filesystem/item.go
full := filepath.Join(cleanRoot, filepath.FromSlash(sourceID))
if full != cleanRoot && !strings.HasPrefix(full, cleanRoot+string(filepath.Separator)) { ... }
```

Neither calls `filepath.EvalSymlinks` — unlike `plugins/filesystem/walk.go`'s
Match-time symlink check (`walk.go:135-144`), which correctly resolves an
in-tree symlink and refuses to include it as an item if the resolved
target falls outside the configured root (T-12-12). Because the two
places that actually **touch bytes or exec a program** — `fetchByKind`
(reads the file and serves its bytes back to the requester) and
`FilesystemOpenHandler` (execs `xdg-open` against the resolved path) — only
re-check the *string shape* of the joined path, an attacker with write
access to the watched folder (a realistic threat on a shared network
mount, or any multi-writer directory) can:

1. Place (or wait for the operator to place) a legitimate file that gets
   indexed normally at sync time.
2. Replace that file on disk with a symlink pointing outside the
   configured root (e.g. at `/etc/passwd`, `~/.ssh/id_rsa`, or any file
   the kernel/plugin process can read) — a classic TOCTOU window between
   one sync and the next `Fetch`/`Open` call.
3. Trigger `GET /api/items/{id}` (or click "Open in …" in the UI) for
   that item's still-valid, unchanged `source_id`.

The lexical `filepath.Join`+`HasPrefix` check passes unchanged (the
`source_id` string never contained a `..` segment), so:
- `plugins/filesystem/fetch.go`'s `fetchBytesRendition`/
  `fetchMarkdownRendition`/`fetchPlainTextRendition` will `os.ReadFile`/
  `os.Open` the **symlink target** and serve those bytes back to the
  browser — a disclosure of any file readable by the kernel/plugin
  process, from outside the folder the operator consented to expose.
- `kernel/httpapi/fsopen.go`'s `FilesystemOpenHandler` will `exec.Start`
  `xdg-open` against the same escaped path.

This directly contradicts the guarantee `docs/plugin-contract.md`
publishes for third-party plugin authors: *"The kernel's own
re-resolution on the open route re-validates the joined path stays
inside the configured root before ever exec'ing anything"* — that
re-validation exists, but does not actually resolve symlinks, so the
guarantee is honored only at Match/walk time, not at either of the two
places where it is load-bearing. No existing test covers this: both
`TestFetch_SourceIDEscapingTheRootIsRefusedBeforeAnyFileIsOpened`
(`plugins/filesystem/fetch_test.go`) and
`TestFilesystemOpen_PathEscapeAnswersInvalidPath`
(`kernel/httpapi/fsopen_test.go`) only exercise `..`-segment traversal,
never a symlink swap — the symlink coverage that does exist
(`plugins/filesystem/walk_test.go`'s `TestWalk_SymlinkToFileOutsideRootIsExcluded`)
only proves the *indexing*-time guard, not the *serving*-time one.

**Fix:** Resolve the joined path with `filepath.EvalSymlinks` (falling
back to the lexical path when resolution fails, e.g. for the item-open
route's happy path where the target may legitimately not exist yet is
never expected here, but should still fail safe) before the containment
comparison, in both `resolvePath` (`plugins/filesystem/item.go`) and
`fsopen.go`'s inline check — mirroring `walk.go`'s own discipline:

```go
func resolvePath(root, sourceID string) (string, error) {
	cleanRoot := filepath.Clean(root)
	full := filepath.Join(cleanRoot, filepath.FromSlash(sourceID))
	if full != cleanRoot && !strings.HasPrefix(full, cleanRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("resolved path escapes source root")
	}
	// Re-resolve any symlink in the final path component (or an ancestor)
	// and re-check containment against the RESOLVED path — the lexical
	// check above cannot see that a file legitimately indexed earlier has
	// since been swapped for a symlink pointing outside root.
	real, err := filepath.EvalSymlinks(full)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	if real != cleanRoot && !strings.HasPrefix(real, cleanRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("resolved path escapes source root")
	}
	return full, nil
}
```

Apply the identical resolve-then-recheck step in `kernel/httpapi/fsopen.go`
before calling `opener`.

## Warnings

### WR-01: In-tree symlink containment check compares against the unresolved root

**File:** `plugins/filesystem/walk.go:77` (`cleanRoot`), used at `walk.go:140`

**Issue:** The in-tree symlink check (`walk.go:122-145`) correctly calls
`filepath.EvalSymlinks(path)` to resolve a symlinked file's real target,
but compares the result against `cleanRoot`, which is only
`filepath.Clean(root)` — never itself resolved through `EvalSymlinks`. If
the *configured root* sits behind a symlink or bind mount (not unusual:
a dotfile-manager symlink such as `~/Documents` → `~/dotfiles/Documents`,
or a container/flatpak bind mount), then `EvalSymlinks` on any
legitimately in-tree symlinked file resolves through the root's own
symlink chain as well, producing a real path that will never share
`cleanRoot`'s literal prefix even though the file is genuinely inside the
folder the operator configured. The check at `walk.go:140` then
(incorrectly) treats every such file as "resolves outside root" and
silently drops it from the corpus (`return nil`, no log, `skipped` not
even incremented since this isn't a stat/permission error) — an
under-inclusion bug an operator would observe only as "my symlinked
documents never appear in the stream," with nothing pointing at the
cause.

This is the inverse of CR-02 (over-restriction instead of
under-restriction), so it's not a security defect, but it is a real
functional regression risk for a common desktop pattern (symlinked
document directories), and it degrades trust in the "resolves outside
root" refusal — an operator hitting this will reasonably suspect a
security false-positive rather than the actual cause.

**Fix:** Resolve `root` itself once via `filepath.EvalSymlinks` (falling
back to `cleanRoot` if resolution fails, e.g. because `root` doesn't
exist) and compare `real` against that resolved value, not the merely
lexically-cleaned one.

---

_Reviewed: 2026-08-14T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
