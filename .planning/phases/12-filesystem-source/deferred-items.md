# Deferred items — Phase 12 (filesystem-source)

Out-of-scope discoveries logged during plan execution, per the executor's
scope-boundary rule (fix only what the current task's own files touch;
log the rest here rather than silently expanding scope).

## 12-05 Task 2: filesystem plugin binary missing from the published release/nightly asset list

**Found during:** 12-05 Task 2 (documentation republish), while verifying
README.md's "Prebuilt" section against the actual CI release pipeline.

**Issue:** `.github/workflows/release.yml` and `.github/workflows/
nightly.yml` both hardcode their published `ASSETS` list:

```
ASSETS="topos plugins/topos-plugin-paperless plugins/topos-plugin-silverbullet plugins/topos-plugin-proton plugins/topos-plugin-whatsapp"
```

`topos-plugin-filesystem` is not in this list, even though `make
plugins-portable` (the target `make build-portable` — the entry point
both workflows use — delegates to) already builds it alongside the other
five `CGO_ENABLED=0` binaries (12-01-SUMMARY.md's Makefile change).
Unlike Signal, the filesystem plugin needs no cgo and no distro-specific
system library, so there is no correctness reason to exclude it from
publication — it was simply never added to either workflow's asset list
across 12-01 through 12-04.

**Effect:** an operator downloading the latest prebuilt release currently
gets five binaries (kernel + four non-Signal plugins) and must build
`topos-plugin-filesystem` from source themselves, even though it builds
identically to the four that ARE published. README.md's "Prebuilt"
section (updated by 12-05 Task 2) states this explicitly rather than
claiming a publication that doesn't yet exist.

**Why not fixed here:** `.github/workflows/release.yml` and `nightly.yml`
are not in 12-05's declared `<files>` list (docs/plugins/filesystem.md,
docs/plugins/README.md, docs/plugin-contract.md, docs/api.md,
docs/testing.md, config.example.toml, README.md) and are release
engineering, not documentation — outside the scope-boundary rule's
"directly caused by the current task's changes" test. The gap predates
this task (introduced somewhere across 12-01–12-04, none of which touch
either workflow file either).

**Suggested fix:** add `plugins/topos-plugin-filesystem` to both
`ASSETS` lines in `.github/workflows/release.yml` and `.github/
workflows/nightly.yml`, then remove the caveat this task added to
README.md's "Prebuilt" section.
