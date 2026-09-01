# Releasing

The one page to read when cutting a release or crossing a milestone
boundary: how a milestone opens and closes under CodeCrew, how a
release is actually cut, what the nightly build does, and why no plugin
binary is among the published artifacts.

## Milestones

A milestone is a GitHub issue labelled `cc:milestone` — [#6](https://github.com/davison/topos/issues/6)
was M1, [#40](https://github.com/davison/topos/issues/40) is M2 — and
its lifecycle is CodeCrew's, driven by the `gh codecrew` verbs (the
protocol is [gh-codecrew's SPEC](https://github.com/radiusred/gh-codecrew/blob/main/SPEC.md);
`AGENTS.md` at the repository root says how this project runs it).
GitHub's own milestone objects are not part of it.

- **Opening.** `gh codecrew milestone new --title … --goal … --requirement …`
  creates the issue with its numbered requirements (`M<n>-R<k>`) and
  writes the milestone's row into [`ROADMAP.md`](../ROADMAP.md) locally —
  that edit rides in the milestone's first PR, as the verb says.
- **Work.** Every change is a task issue (`task new --milestone N`), with
  its plan in the issue body, started (`task start`, which creates the
  linked branch) and finished (`task finish --operator-confirm`, which
  enforces the gates — closing PR, CI checks reporting green or skipped,
  a model review under the reviewer contract, operator confirmation —
  and rebase-merges). Decisions and deviations are recorded on the task
  as they happen; a question only the operator can answer is raised as a
  `cc:needs-decision` gate (`checkpoint`).
- **Closing.** `milestone evidence N` checks every link the milestone's
  record cites; the qa seat posts one verdict line per requirement on
  the milestone issue and files a remedy task for anything not
  satisfied; the milestone document lands under
  [`docs/milestones/`](milestones/) as the last task; the release tag is
  cut from the main that document merged into (below); the operator's
  live-instance UAT is raised as a checkpoint; then
  `gh codecrew milestone close N` closes the issue once its gates —
  tasks closed, requirements declared, QA verdicts present, document
  merged — all pass. The close verb sweeps every task's PR and takes
  several minutes; that is normal.

Before the migration to CodeCrew, milestones lived under `.planning/`
(the GSD era — see [the genesis record](milestones/0-genesis-the-gsd-era.md));
that directory is a frozen archive now. The GSD-era script that mirrored
its milestone state into GitHub milestone objects went with the premise
([#44](https://github.com/davison/topos/issues/44)); the milestone
objects it created remain as history.

## Cutting a release

The trigger is a human pushing a tag matching `v*.*.*` — nothing in CI
creates that tag. `.github/workflows/release.yml` watches for it and, on
a match, builds and publishes.

### Versioning: the next tag is derived from the commit log

The version number is never chosen by feel — it is read off the
conventional-commit log since the last tag
([#64](https://github.com/davison/topos/issues/64), operator-decided at
the M2 release gate on
[#40](https://github.com/davison/topos/issues/40)):

```bash
git log "$(git describe --tags --abbrev=0)"..main --pretty=%s
```

(`git describe --tags --abbrev=0` resolves the last tag; substitute a
specific tag to derive against a different baseline.)

- Any commit whose type carries the breaking marker — any `<type>!:` or
  `<type>(scope)!:`, so `feat!:` and an unscoped `fix!:` alike — forces
  a **minor** bump.
- Otherwise, any `feat:` (a purely additive feature) or any `fix:`
  yields a **patch** bump.
- A log holding only `docs:`, `chore:`, `test:`, `ci:`, `build:` or
  `refactor:` commits bumps nothing on its own — there is no new tag to
  cut until behaviour changes.
- A **major** bump is reserved for substantial rework with breaking
  changes, and is always a human decision — never derived mechanically.

"Breaking" means breaking for a consumer of a published surface: the
plugin contract (`proto/topos/v1/plugin.proto` and the SDK), the HTTP
API (`docs/api.md`), the config schema, the CLI's commands and flags,
or the install layout. The same derivation governs
[`topos-plugins`](https://github.com/davison/topos-plugins)' `v*.*.*`
tags against its own log.

Because the tag is checkable from the log alone, the log's discipline
is what the process stands on: commit types must truthfully classify
each change, and the reviewer verifies them against the diff —
`roles/implementer.local.md` and `roles/reviewer.local.md` carry those
obligations. A wrongly-typed commit discovered after merge is corrected
by a follow-up commit stating the true consequence in its own type; the
derivation then reads both.

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
releases carry the same exclusion and whose `plugins/signal/README.md` in topos-plugins
documents the local build (`CGO_ENABLED=1 go build -tags libsqlcipher`)
and the per-distro `sqlcipher` package names.

## See also

- **[`CONTRIBUTING.md`](../CONTRIBUTING.md)** — the local dev loop and the
  test gates a change must pass before it's mergeable.
- **[`topos-plugins`](https://github.com/davison/topos-plugins)** — the
  source plugins themselves, their per-plugin READMEs, and their own
  signed releases.
- **[`SECURITY.md`](../SECURITY.md)** — how to report a vulnerability.
