<!--
roles/implementer.local.md — this project's extension to the implementer contract.
Loaded after roles/implementer.md, append-only, never a replacement: an extension
that contradicts its contract is a review finding. gh codecrew roles show implementer
prints the composition. Comments only, it adds nothing.

What belongs here, with worked examples (house style, repo conventions,
a platform's wake syntax, ids and tooling):
https://github.com/radiusred/gh-codecrew/blob/main/docs/extensions.md
Protocol: https://github.com/radiusred/gh-codecrew/blob/main/SPEC.md (section 7)
-->

## Commit types decide the version (#64)

Release tags in this project are DERIVED from the commit log alone —
`docs/releasing.md`, "Versioning" — so every commit subject's
conventional-commit type is load-bearing, not decoration:

- `feat:` — new behaviour that changes no existing consumer. Bumps the
  patch version.
- `fix:` — corrected behaviour. Bumps the patch version.
- `feat!:` / any `<type>(scope)!:` — breaking for a consumer of a
  published surface (plugin contract/SDK, HTTP API, config schema, CLI,
  install layout). Forces a minor bump. Never omit the bang on a
  breaking change; never add it to an additive one.
- `docs:`, `chore:`, `test:`, `ci:`, `build:`, `refactor:` — no version
  consequence, so they must contain NO behaviour change. A doc edit
  that also changes code is not `docs:`.
- A commit mixing kinds is split, or typed by its most
  version-significant part.

A wrongly-typed commit is a defect in its own right: it silently
mis-derives the next tag.
