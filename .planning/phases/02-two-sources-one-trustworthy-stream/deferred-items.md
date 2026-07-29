# Deferred items — Phase 2

Out-of-scope discoveries logged during plan execution, per the executor's
scope-boundary rule (fix only what the current task's changes directly
caused; log everything else here rather than fixing it inline).

## 02-04: `kernel/config.Validate` requires `base_url`/`token` for every source

**Found during:** 02-04-PLAN.md Task 2, while verifying `plugins/mock` end
to end against the real kernel via a scratch config.

**Issue:** `kernel/config/config.go`'s `Validate` unconditionally requires
every `[sources.<name>]` entry to have a non-empty `base_url` and `token`,
regardless of whether the plugin behind that source actually needs them.
`plugins/mock`'s own `main.go` does not require
`WEBSPACES_SOURCE_CONFIG` to be set at all — by design, since the mock has
no connection details — but a real deployment enabling
`[sources.mock]` still needs two placeholder values (`base_url = "unused"`,
`token = "unused"`) to satisfy this kernel-level check, which has nothing
to do with the mock plugin's own requirements. `config.example.toml`'s
commented-out `[sources.mock]` block documents this workaround inline.

This will become a real (not just cosmetic) problem for a genuinely
configless local-database source — Signal or WhatsApp, per
`PROJECT.md`'s MVP source list — which will need to configure zero
connection fields, not two placeholder ones.

**Why deferred, not fixed here:** `kernel/config/config.go` is not in
02-04-PLAN.md Task 2's `files_modified` list, and relaxing this
unconditional check is a design decision (how does kernel-level
validation know which fields a given plugin needs without launching it
first?) rather than a one-line fix — out of scope for a plan whose
`files_modified` list was locked before this discovery. Pre-existing
behavior (predates this phase), not something 02-04's own changes caused.

**Suggested resolution:** revisit when the Signal or WhatsApp plugin
(Phase 4/5) is planned — likely either (a) drop the kernel-level
`base_url`/`token` requirement entirely and rely on each plugin's own
documented "fail startup loudly on a missing required key" contract
(`docs/plugin-contract.md`), or (b) make the requirement conditional on
whether `base_url`/`token` keys are present in the source's TOML block at
all (absent entirely = skip the check; present-but-empty = still an
error).

**Status:** open.

## 02-06: `e2e-smoke.sh` paperless deep-link check fails on `link.url` prefix

**Found during:** 02-06-PLAN.md Task 2, while running the full smoke test
to verify the stylesheet stage against the live server. (This entry was
claimed in `02-06-SUMMARY.md` but not actually written at the time; added
during phase re-verification so the record matches the SUMMARY.)

**Issue:** After the stylesheet stage passed, `scripts/e2e-smoke.sh`
failed at a pre-existing, unrelated check: a paperless item's `link.url`
did not start with the expected `$PAPERLESS_URL/documents/` prefix. This
concerns paperless-ngx deep-link construction (or a
`PAPERLESS_URL`/configured-source-URL mismatch in the execution
environment), not anything in 02-06's CSS/theme scope.

**Why deferred, not fixed here:** outside 02-06's declared
`files_modified` (`web/src/app.css`, `scripts/assert-stylesheet.sh`,
`scripts/e2e-smoke.sh`'s stylesheet stage only) and pre-existing — the
same check predates the gap-closure plan. 02-VERIFICATION.md's truth #1
separately verified the deep-link construction logic itself; the failure
is environment-dependent.

**Suggested resolution:** reproduce with the configured
`[sources.paperless] base_url` and the `PAPERLESS_URL` env var printed
side by side; either align the smoke test's expectation with the
configured base URL or normalize the URL the paperless plugin emits.

**Status:** open.
