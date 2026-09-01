<!--
roles/reviewer.local.md — this project's extension to the reviewer contract.
Loaded after roles/reviewer.md, append-only, never a replacement: an extension
that contradicts its contract is a review finding. gh codecrew roles show reviewer
prints the composition. Comments only, it adds nothing.

What belongs here, with worked examples (house style, repo conventions,
a platform's wake syntax, ids and tooling):
https://github.com/radiusred/gh-codecrew/blob/main/docs/extensions.md
Protocol: https://github.com/radiusred/gh-codecrew/blob/main/SPEC.md (section 7)
-->

## Verify commit types against the diff (#64)

Release tags are derived from the commit log alone
(`docs/releasing.md`, "Versioning"), so type honesty is part of every
review, not a style preference. Check each commit on the branch:

- A behaviour change under `docs:`/`chore:`/`test:`/`ci:`/`build:`/
  `refactor:` is a finding — it would silently escape the version.
- A change that breaks a consumer of a published surface (plugin
  contract/SDK, HTTP API, config schema, CLI, install layout) without
  the `!` marker is a finding — it would under-bump.
- An additive change carrying `!`, or a `feat:` that only fixes, is a
  finding in the other direction.

Name the commit and the correct type in the verdict comment; the fix is
a reworded commit before merge, or a follow-up commit stating the true
consequence when history is already shared.
