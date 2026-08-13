# topos-plugin-external-demo

This module is Phase 11's standing proof for ROADMAP success criterion 5:
"the [external-plugin] mechanism is proven before any out-of-repo source
work starts." It is a genuine, standalone Go module — module path
`example.com/acme/topos-plugin-external-demo`, deliberately **not** under
`github.com/davison/topos` — written entirely from the published,
third-party-facing contract:

- `docs/plugin-contract.md`
- `proto/topos/v1/plugin.proto`
- the `sdk` Go module (`github.com/davison/topos/sdk`)

exactly as a genuine third-party plugin author would build one, with no
access to any other file in this repository. Its shape mirrors
`plugins/mock` (the contract's own worked reference example) in spirit,
but its `Match` implementation is **observational, not illustrative**:
every item it returns reports back exactly what config and environment
the kernel actually handed this process, so this phase's extras-passthrough
claim (PLUG-09) and launch-environment-allowlist claim (D-14) are provable
by inspecting the synced corpus, not merely asserted by a test's own mock.

## Why this lives under `testdata/`

Two independent reasons, both load-bearing:

1. **The Go toolchain itself never compiles it as part of `./...`.**
   Directories named `testdata` are ignored by `go build`/`go test`
   pattern-matching everywhere in the module tree — this is a standing Go
   tool convention, not a project-specific rule — so `CGO_ENABLED=0 go
   build ./...` and `go test ./...` from the repository root never touch
   this module.
2. **This repository's own AST audits (`internal/audit`) deliberately
   skip any directory named `testdata`, anywhere it occurs
   (`internal/audit/outbound_hosts_test.go`'s `shouldSkipDir`).** A real
   third party's out-of-repo module is never scanned by this repo's
   egress/module-pin/icon audits either — living under `testdata/` keeps
   this proof binary's audit exposure identical to the real thing it
   stands in for, rather than accidentally joining the in-repo plugin
   set's own audited surface.

This module is a **stand-in for a third party's own separate build** — it
must never live under `plugins/`, must never be built by `make build`,
`make plugins`, or `make plugins-portable`, and must never be copied into
a real installation's trusted `[plugins] dir`. It exists purely to prove
the external-plugin mechanism end to end; it is never shipped.

## Building it

Its own dedicated Makefile target (`make external-demo`, defined at the
repository root) builds it, `CGO_ENABLED=0`, into its own output
directory:

```
make external-demo
# -> bin/plugins-external/topos-plugin-external-demo
```

`bin/plugins-external/` is a directory distinct from `bin/plugins/` (what
`make build`/`make dev` populate and a real `[plugins] dir` config value
can point at) precisely so this binary can never be picked up by a real
installation's trusted plugin directory by accident.

## Configuration

Like every contract-conformant plugin, it reads its connection details
from the `WEBSPACES_SOURCE_CONFIG` environment variable the kernel sets
before launching it:

```json
{ "path": "/any/non-empty/value", "extras": { "workspace_id": "acme-42" } }
```

- `path` is the one **required** kernel-known key (mirroring
  `plugins/mockstrict/main.go`'s and `plugins/signal/main.go`'s identical
  shape): the process fails startup loudly, by name, non-zero, when it is
  empty or `WEBSPACES_SOURCE_CONFIG` is unset entirely — never silently,
  never mid-`Match`. The path itself is never opened; any non-empty string
  satisfies the guard.
- `extras` carries this instance's own `[sources.<id>.extras]` table
  verbatim (D-12/D-13) — the provider-specific passthrough this whole
  module exists to prove reaches an out-of-repo process unmodified.

## What it reports back

`Describe` declares `source_type "external-demo"`, `contract_version
"topos.v2"`, `match_vocabulary ["labels"]`, and an `extras` declaration
exercising every field of `ExtrasField`: one required, non-secret key
(`workspace_id`) and one optional, secret key (`api_key`), each with a
label and a placeholder. It declares no icon — an omitted icon is a
supported, documented contract state.

`Match` returns, against a configured keyword matching its fixed
`external-demo-proof` label:

- one item per configured extras key (`extras/<key>`), so an operator
  (or a test) can see exactly which provider-specific keys reached this
  process;
- one item per environment variable actually **visible to this process**
  (`env/<NAME>`) — variable **names only**, never values, in both the
  item id and its title, so the synced corpus can never carry a secret
  even when a harness deliberately hands one to this process;
- one fixed anchor item, so a match always returns something even with
  no extras configured.

## Who consumes this

- `kernel/supervisor/externalproof_test.go`'s
  `TestExternalProof_OutOfRepoBinaryEndToEnd` is the standing, mechanical
  gate for ROADMAP success criterion 5: it builds this module fresh, links
  it into a fixture's external plugin directory with a computed content
  pin, boots a real supervisor against it, and asserts discovery, tier,
  pin enforcement, extras passthrough, and environment scrubbing — all
  from this plugin's own point of view (its synced item corpus), not from
  a test's own assumptions about what the kernel did.
- Later Phase 11 plans' browser harness (`web/e2e/`) link the built binary
  (`bin/plugins-external/topos-plugin-external-demo`) into a fixture's
  external directory the same way, for specs that need a real,
  out-of-repo-shaped plugin binary running in the browser suite.

## What this module is not

It is not a template to copy for a real out-of-repo plugin — build a real
plugin from `docs/plugin-contract.md` and `plugins/mock` instead, the
inputs the contract itself names as sufficient (`PLUG-05`). This module's
own `Match`/`Fetch` behavior is deliberately self-referential (it reports
on its own launch environment) precisely because its only job is proving
the mechanism, not modeling a real source's data shape.
