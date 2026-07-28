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
