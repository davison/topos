# Releasing

The one page to read when cutting a release or crossing a milestone
boundary: how GitHub milestones stay in step with `.planning/`, how a
release is actually cut, what the nightly build does, and why no plugin
binary is among the published artifacts.

## Milestones

`.planning/` is the source of truth for milestone state. GitHub
milestones are a mirror, kept in step by an explicit step — never the
reverse. If a milestone's title, state, or description is edited directly
in the GitHub UI, that edit is silently overwritten the next time the
sync step below runs. GSD has no native mechanism for keeping a GitHub
milestone in step with a `.planning/` milestone, so this repository
carries its own: a committed, idempotent `gh api` wrapper.

At a milestone boundary, run `scripts/sync-milestones.sh` with the
milestone title from `.planning/STATE.md`'s frontmatter `milestone` key,
and the appropriate action — once when the milestone opens, once when it
closes:

```bash
# When a milestone opens (e.g. right after /gsd-new-milestone):
scripts/sync-milestones.sh v1.0 open

# When a milestone closes (e.g. right after /gsd-complete-milestone):
scripts/sync-milestones.sh v1.0 close
```

Two guarantees make this safe to trust:

- **Idempotent.** The script looks the milestone up by exact title,
  across all states, before deciding whether to create or patch it. Safe
  to re-run.
- **No delete path.** The script cannot delete a milestone — the
  capability is absent, not merely unused. `.planning/` never deletes a
  milestone either, only opens or closes one; a delete would orphan every
  issue assigned to it.

The real current state is the worked example: milestone `v1.0` already
exists on `davison/topos` as milestone number 1. Running
`scripts/sync-milestones.sh v1.0 open` against it reconciles that
existing milestone rather than creating a second, differently-numbered
`v1.0`.

## Cutting a release

The trigger is a human pushing a tag matching `v*.*.*` — nothing in CI
creates that tag. `.github/workflows/release.yml` watches for it and, on
a match, builds and publishes.

Sequence:

1. Confirm the portable gate is green (`make test-portable`, or check the
   latest `ci.yml` run on `main`).
2. Push a tag matching `v*.*.*`, e.g. `git tag v1.0.0 && git push origin
   v1.0.0`.
3. Watch the `Release` workflow run to completion (`gh run watch` or the
   Actions tab).
4. Verify the published assets on the resulting GitHub Release.

The release contains the kernel binary (`topos`), the provenance
verifier (`topos-provenance`) and a `checksums.txt` —
`sha256sum` output over every other published asset. A downloader
verifies their copy with `sha256sum -c checksums.txt` after downloading
everything into the same directory.

Plugin binaries are no longer published here: the source plugins live in
[`topos-plugins`](https://github.com/davison/topos-plugins), whose own
tag-triggered releases build, checksum, and ed25519-sign the fleet,
installed by that repository's own `make install`. See
[`docs/install.md`](install.md) for how an installed instance gets them.
Older kernel releases still carry whatever their tag published —
`make install <tag>` installs exactly that and reports what it installed;
an install consumes published artifacts, it does not build them.

The fixture plugin binary (`topos-plugin-mock`) is deliberately not
published: it is a contract-reference and test-harness fixture, not an
installable source, matching the kernel's own exclusion of it from the
operator's "+" source picker.

## Nightlies

`.github/workflows/nightly.yml` runs on a `0 3 * * *` cron schedule, plus
a `workflow_dispatch` manual-trigger escape hatch (load-bearing for
testing the gate below without waiting a day). A cron trigger fires on a
timer regardless of whether the repository actually changed, so cron
alone does not mean "something changed" — the workflow's change gate is
what makes that true.

The mechanism: a moving `nightly` git tag records the commit the current
nightly build was produced from. Every run's first job (`check-changes`)
compares `HEAD` against that tag; if they're equal, the `build` job is
skipped entirely — not short-circuited partway through, skipped as a
whole job — and nothing is published. Only when `HEAD` has moved past the
`nightly` tag does the build run, publish a fresh `--prerelease` GitHub
Release over the same asset set `release.yml` publishes, and force-move
the `nightly` tag to the new `HEAD`.

Practical consequence: a maintainer can see exactly what's in the current
nightly by diffing from the `nightly` tag (`git log nightly..main`, or
`git diff nightly..main`).

## The Signal plugin binary

**Decision (2026-08-12, Plan 10-01 Task 2 checkpoint):** the Signal
plugin binary is excluded from every published artifact, because it is a
cgo build dynamically linking the system's SQLCipher library — a binary
built on the CI runner's distro carries no promise of running on
another. That decision travelled with the plugin to
[`topos-plugins`](https://github.com/davison/topos-plugins), whose
releases carry the same exclusion and whose `plugins/signal/README.md`
documents the local build (`CGO_ENABLED=1 go build -tags libsqlcipher`)
and the per-distro `sqlcipher` package names.

## See also

- **[`CONTRIBUTING.md`](../CONTRIBUTING.md)** — the local dev loop and the
  test gates a change must pass before it's mergeable.
- **[`topos-plugins`](https://github.com/davison/topos-plugins)** — the
  source plugins themselves, their per-plugin READMEs, and their own
  signed releases.
- **[`SECURITY.md`](../SECURITY.md)** — how to report a vulnerability.
