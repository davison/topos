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
  critical: 1
  warning: 2
  info: 1
  total: 4
status: issues_found
---

# Phase 12: Code Review Report

**Reviewed:** 2026-08-14T00:00:00Z
**Depth:** standard
**Files Reviewed:** 56
**Status:** issues_found

## Summary

This report supersedes the prior 12-REVIEW.md, which predates the phase's
gap-closure plan (12-06-PLAN.md). That plan's commits fixed the prior
report's two blockers — I re-verified both fixes hold rather than
re-reporting them:

- **CR-01 (xdg-open killed by request-context cancellation):** fixed.
  `newXDGOpener` (`kernel/httpapi/fsopen.go:39-52`) now builds the child
  with plain `exec.Command("xdg-open", path)` and `cmd.Start()`, never
  `exec.CommandContext`. `FilesystemOpenHandler` hands the opener
  `context.WithoutCancel(ctx)` (`fsopen.go:167`), and an AST test
  (`TestNewXDGOpener_ChildIsNotBoundToACallerContext`, `fsopen_test.go`)
  mechanically pins that `newXDGOpener`'s closure takes a blank-identifier
  context parameter and its body contains no `CommandContext` selector.
- **CR-02 (lexical-only containment check, TOCTOU symlink escape):**
  fixed. Both `kernel/httpapi/fsopen.go` (`FilesystemOpenHandler`,
  lines 142-155) and `plugins/filesystem/item.go` (`resolvePath`,
  lines 99-116) now call `filepath.EvalSymlinks` on the joined path and
  on the configured root, and compare the *resolved* pair before ever
  reading a byte or exec'ing `xdg-open` — matching `walk.go`'s own
  discipline. Both fail closed (an unresolvable path — including a
  vanished target — is refused, mapped to `item_not_found` for
  `fs.ErrNotExist`, `invalid_path` otherwise). Regression tests exist on
  both sides (`TestFilesystemOpen_SymlinkSwappedAfterIndexingAnswersInvalidPathAndNeverOpens`,
  `TestFetch_SymlinkSwappedAfterIndexingIsRefusedBeforeAnyBytesAreServed`).
- **WR-01 (symlinked-root containment compared against the unresolved
  root):** fixed. `walk.go` now resolves the configured root once via a
  shared `resolveRoot` helper before the walk begins and before the
  in-tree symlink comparison, with matching regression coverage
  (`TestWalk_InTreeSymlinkUnderASymlinkedRootIsStillIncluded`,
  `TestResolvePath_SymlinkedRootStillResolvesAnInRootFile`).

Tracing the new plugin's own logic end to end (Match → walk/scope →
Fetch) surfaced one new BLOCKER: `Fetch`'s per-preview-kind dispatch
re-derives a file's classification through a code path that ignores the
same `extras`-driven `include_glob` scope `Match` used to decide the file
belongs in the index at all — so a class of item that `Match`
legitimately returns, and that the UI legitimately lists in the stream,
404s when opened. Two WARNINGs and one INFO item round out the findings
below.

## Critical Issues

### CR-01: `Fetch` re-derives classification without the scope/extras that admitted the item, breaking `include_glob`-only files

**File:** `plugins/filesystem/fetch.go:81-93` (`fetchByKind`)

**Issue:** `Match` (`plugins/filesystem/plugin.go:126-154`) delegates
inclusion/classification to `walk.go`'s `walk`, which in turn calls
`scope.includes` (`scope.go:58-87`) — the function that implements the
documented precedence rule (`docs/plugins/filesystem.md`'s "Resolution
order"): exclude first, then **include-if-declared, which REPLACES the
default extension allowlist entirely**. When `include_glob` matches a
file whose extension is *not* in the built-in `extensionTable`,
`scope.includes` deliberately returns `included=true` with
`classification{kind: previewKindMetadataOnly}` — proven directly by
`scope_test.go`'s `TestScope_UnknownExtensionIncludedByGlobIsMetadataOnly`
and `walk_test.go`'s `TestScope_IncludeGlobWidensPastTheDefaultAllowlist`.
Such a file is a completely legitimate `Match` result: it appears in the
index and in the UI's stream exactly like any other item.

`fetchByKind`, however, never builds a `*scope` from `p.extras` at all —
it calls the bare package-level `classify(sourceID)` (`classify.go:94-98`)
directly:

```go
c, ok := classify(sourceID)
if !ok {
    return nil, status.Errorf(codes.NotFound, "filesystem: item %q not found", sourceID)
}
```

`classify` only ever consults the fixed `extensionTable` — it has no
knowledge of `include_glob`/`exclude_glob` and cannot reproduce
`scope.includes`'s "unknown extension admitted by glob → metadata-only"
branch. For any item that only exists in the index *because*
`include_glob` widened scope past an unrecognized extension (e.g. an
operator configuring `include_glob = "**/*.epub"` to pull in an ebook
collection, or any extension outside the fixed table), `classify` returns
`ok=false`, and `fetchByKind` answers `codes.NotFound` — mapped by the
kernel to `404 item_not_found` on both `GET /api/items/{id}` and
`GET /api/items/{id}/content` (`kernel/httpapi/item.go`'s
`writeFetchError`).

The user-visible effect: the item renders correctly in the stream (Match
never consulted `classify` either — it has no gate at all), but opening it
produces a false "not found" error for a file that is present on disk and
was legitimately synced — instead of the documented, correct outcome
(`available: false`, `unavailable_reason: "preview not supported for this
file type; open in source"`, exactly like a `.docx` file gets today).
This directly contradicts `docs/plugins/filesystem.md`'s own documented
behavior: *"A file matching an include pattern is included even if its
extension is outside the default allowlist below (classified
metadata-only if the extension is unrecognized, never guessing a MIME
type)"* — the classification happens at `Match`/index time, but is lost
and re-derived incorrectly at `Fetch` time.

No existing test catches this: `fetch_test.go`'s fixtures only cover
extensions already in `extensionTable` (`.pdf`, `.png`, `.md`, `.txt`,
`.docx`, `.svg`) plus a genuinely-missing file; none exercises `Fetch`
against a `NewSourcePlugin(root, extras, ...)` instance with a
non-default `include_glob` for an unrecognized extension — the exact gap
`scope_test.go`'s `TestScope_UnknownExtensionIncludedByGlobIsMetadataOnly`
proves exists at the `Match` layer but that no test carries through to
`Fetch`.

**Fix:** Build a `*scope` from `p.extras` in `fetchByKind` and classify
through it — the same path `Match`/`walk` already use — instead of
calling the bare `classify` helper directly:

```go
func (p *SourcePlugin) fetchByKind(sourceID string) (*toposv1.FetchResponse, error) {
	full, err := resolvePath(p.root, sourceID)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, status.Errorf(codes.NotFound, "filesystem: item %q not found", sourceID)
		}
		return nil, status.Errorf(codes.InvalidArgument, "filesystem: %v", err)
	}

	sc := newScope(p.extras)
	c, included, err := sc.includes(sourceID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "filesystem: %v", err)
	}
	if !included {
		return nil, status.Errorf(codes.NotFound, "filesystem: item %q not found", sourceID)
	}

	switch c.kind {
	// ...unchanged
	}
}
```

(`included=false` still correctly maps to `NotFound` for a `source_id`
that is on disk but genuinely outside this instance's configured scope —
e.g. scope narrowed since the item was indexed — matching today's
behavior for that case.)

## Warnings

### WR-01: Filesystem plugin's `Item.Provenance` omits the documented `source_system` key

**File:** `plugins/filesystem/plugin.go:190-195` (`toItem`)

**Issue:** `docs/plugin-contract.md`'s "Provenance" section documents six
provenance keys the kernel's HTTP API publishes on every item
(`docs/api.md`), five of which "a plugin is responsible for populating":
`source_type`, `source_system`, `source_id`, `plugin`, `contract_version`
(`synced_at_unix` is filled in by the kernel's index layer). Every other
in-repo plugin follows this: `plugins/paperless/plugin.go:129-135`,
`plugins/silverbullet/plugin.go:237-243`, and
`plugins/signal/plugin.go:259-265` all set `"source_system"` (to their
own `base_url`/`configDir`). `plugins/filesystem/plugin.go`'s `toItem`
does not:

```go
Provenance: map[string]string{
    "source_type":      sourceType,
    "source_id":        sourceID,
    "plugin":           "topos-plugin-filesystem",
    "contract_version": contractVersion,
},
```

`kernel/httpapi/stream.go`'s `toStreamItemFor` copies `it.Provenance`
verbatim (adding only `synced_at_unix`) — it never synthesizes
`source_system` on a plugin's behalf
(`grep -rn "source_system" kernel/` finds it referenced only inside
`contract_test.go`'s own hand-built fixture). The result: every
filesystem-sourced item's `GET /api/items/{id}` (and
`GET /api/webspaces/{ws}/stream`) response is missing the
`provenance.source_system` field entirely, silently breaking the
documented "every item carries these keys" contract and the AGENT-02
"republish provenance verbatim" guarantee for this one source type. The
frontend's `SourceConfig`/item type already marks the field optional
(`web/src/lib/api.ts:31`), so nothing currently crashes on its absence —
but this is a genuine, silent contract regression with no test coverage
anywhere in this phase (no filesystem-specific provenance assertion
exists in `plugin_test.go`/`item_test.go`).

**Fix:** Add the missing key, using the configured root (the "source
instance" address a filesystem source's items came from, mirroring
`p.baseURL`/`p.configDir` in the sibling plugins):

```go
Provenance: map[string]string{
    "source_type":      sourceType,
    "source_system":    p.root,
    "source_id":        sourceID,
    "plugin":            "topos-plugin-filesystem",
    "contract_version": contractVersion,
},
```

### WR-02: Symlink-containment revalidation is discarded before the actual read/exec — narrow TOCTOU window remains

**Files:** `plugins/filesystem/item.go:99-116` (`resolvePath`), consumed
by `plugins/filesystem/fetch.go`'s `fetchBytesRendition`/
`fetchMarkdownRendition`/`fetchPlainTextRendition`/`statForFetch`;
`kernel/httpapi/fsopen.go:127-170` (`FilesystemOpenHandler`)

**Issue:** Both `resolvePath` and `FilesystemOpenHandler` correctly
resolve the joined path with `filepath.EvalSymlinks` and compare the
*resolved* value against the *resolved* root for containment (closing the
prior CR-02) — but both then discard that resolved value and perform the
actual I/O (`os.Stat`/`os.ReadFile`/`os.Open` in `fetch.go`, or
`opener(ctx, full)` → `exec.Command("xdg-open", full)` in `fsopen.go`)
against the original **lexical** `full` path, not the resolved one:

```go
// item.go resolvePath — validates `resolved`, returns the lexical `full`
resolved, err := filepath.EvalSymlinks(full)
// ...containment check against resolved...
return full, nil   // <- the unresolved path is what callers actually read
```

Because the path handed to `os.ReadFile`/`xdg-open` is a plain filesystem
path (not a file descriptor opened at validation time, and not
re-validated a second time immediately before the syscall), a symlink
swapped into place in the interval between `EvalSymlinks`'s validation and
the subsequent read/exec — a narrow but real window, on a filesystem an
operator has already indicated may be a shared/network mount with other
writers (`docs/plugins/filesystem.md`'s own NFS/SMB framing) — is followed
transparently rather than caught. This is the same class of gap CR-02
closed for the *indexed-then-swapped-before-next-request* case, just
narrowed to a race within a single request rather than across two.

Given the project's stated threat model (`docs/plugin-contract.md`: "only
run plugin binaries you built yourself or whose source you trust"; the
kernel already trusts arbitrary local file reads by design) and that
`xdg-open` fundamentally consumes a path rather than a descriptor, this is
a lower-priority hardening item rather than a fresh exploitable path this
phase introduced — but it does mean the CR-02 fix's own stated guarantee
("fails closed... before ever exec'ing anything") is not quite complete:
the window is narrowed from "any time after indexing" to "microseconds
around the syscall," not eliminated.

**Fix:** Where practical, read via the already-resolved path/descriptor
rather than re-walking symlinks a second time implicitly on the read call
(e.g. open the file once via `os.Open` immediately after
`EvalSymlinks` succeeds and reuse that handle for both the size check and
the read, rather than re-resolving on every subsequent `os.Stat`/
`os.ReadFile`/`os.Open` call against the bare path string). For
`xdg-open`, which has no descriptor-based invocation, documenting this as
an accepted residual risk (rather than implying the guarantee is
airtight) is a reasonable alternative to a code change.

## Info

### IN-01: `plugins/filesystem/main.go`'s package doc comment is stale

**File:** `plugins/filesystem/main.go:7-12`

**Issue:** The package doc comment still describes the 12-01/12-02
tracer-era state: *"this bootstrap now decodes and forwards those extras,
but still does not expand recursion — that remains a later plan's
work."* Recursion has since been implemented and shipped (12-03-PLAN.md,
`walk.go`'s `recursive` parameter, `sourceConfig.Recursive` decoded and
passed to `NewSourcePlugin` a few lines below this very comment). The
comment now contradicts the code directly beneath it.

**Fix:** Update or remove the stale sentence; a brief note that recursion
is fully wired (matching `docs/plugins/filesystem.md`'s current
documentation) would keep the file's own history accurate for a future
reader.

---

_Reviewed: 2026-08-14T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
