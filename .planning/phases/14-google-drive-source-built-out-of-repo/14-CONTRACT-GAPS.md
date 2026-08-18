# Phase 14 Contract Gaps — Imported and Triaged

**Imported:** 2026-08-18, from `/home/darren/projects/davison/topos-plugin-gdrive/CONTRACT-GAPS.md`
(that repository's own gap log, produced by its clean-room build against the
published contract — see `14-03-SUMMARY.md`/`14-04-SUMMARY.md` for how it got
there). This is this repository's record of the D-07 exercise's output: the
imported log unedited, anything the live UAT run surfaced that never reached
the plugin repository's own log, and a triage table dispositioning every
entry by id.

No entry below is reworded, merged, or dropped from what the plugin
repository actually recorded — including entries that make the published
contract look incomplete, per this plan's own prohibition.

---

## Part 1 — Imported gap log (verbatim from the plugin repository)

> Reproduced unedited from `topos-plugin-gdrive/CONTRACT-GAPS.md`'s own
> "Entry format" preamble, for context: each entry carries an ID, the date
> the question was actually encountered, the question the four published
> inputs could not answer, what those inputs come closest to saying, the
> resolution the plugin repository actually took, and its own proposed
> disposition (`documentation-fixable` or `contract-change`) — this phase's
> own Part 3 triage table below is the authoritative disposition, not the
> plugin repository's proposal, though in every case the two agree.

### GAP-01

- **Date:** 2026-08-15
- **Question:** Where does a plugin keep its own private state that must
  survive process restarts — specifically, the OAuth refresh token this
  plugin must retain between the standalone `auth` command and every
  later `serve`-mode launch?
- **What the inputs say:** Nothing. The published contract documents how a
  plugin receives *connection details* (`WEBSPACES_SOURCE_CONFIG`, the
  `extras` map) and describes the launch environment a plugin subprocess
  receives, but it is silent on where a plugin is expected to persist its
  own private, plugin-owned data between one process launch and the next.
- **Resolution taken:** A plugin-owned file, mode `0600`, under the
  operator's XDG data directory (e.g.
  `~/.local/share/topos-plugin-gdrive/token.json` on Linux) — per this
  repository's own PRD.md, which locks this as a topos-side decision this
  repository does not revisit. This choice works headlessly, with no
  D-Bus or OS-keyring dependency, consistent with the reduced environment
  a launched plugin subprocess receives.
- **Proposed disposition:** `documentation-fixable` — the published
  contract's "Configuration" or a new "Plugin-private state" section could
  simply say plugins should not assume a home for their own runtime state
  and should choose one deliberately (e.g. XDG data directory), without
  requiring any wire-level change.

### GAP-02

- **Date:** 2026-08-15
- **Question:** Where does a plugin keep its own working cache — in this
  plugin's case, the folder-membership and delta-sync bookkeeping state
  it must maintain between one sync and the next, since the source
  system's own incremental-change feed is not scoped to a single folder
  and this plugin must resolve folder membership itself against locally
  maintained state?
- **What the inputs say:** The same silence GAP-01 already identifies
  extends here — the contract's "Configuration" section models connection
  details (`base_url`, `token`, `path`, `extras`) but says nothing about a
  plugin's own working cache, which is a second and, in this plugin's
  case, larger category of plugin-private state than a single credential
  file.
- **Resolution taken:** Maintained alongside the token file, under the
  same plugin-owned XDG data directory this repository uses for GAP-01's
  resolution — a second file (or set of files) in that same directory,
  read and written exclusively by this plugin, with no expectation the
  topos host ever inspects or manages it.
- **Proposed disposition:** `documentation-fixable` — the same prose
  clarification proposed for GAP-01 naturally covers this second case too:
  a plugin choosing and documenting its own home for whatever private
  runtime state it needs, credential or otherwise.

### GAP-03

- **Date:** 2026-08-16
- **Question:** What `source_type` and `display_name` values should this
  plugin's `Describe` report?
- **What the inputs say:** `contract/plugin.proto:17-19` requires both
  fields and shows `"paperless"` / `"paperless-ngx"` as an illustrative
  comment only; `contract/plugin-contract.md` lists `"folders"`/`"tags"`/
  `"conversations"`/`"pages"` as illustrative match vocabularies and
  describes `source_type` as descriptive provenance retained on items,
  never used to key identity (the `[sources.<id>]` config map key does
  that); `contract/mock/plugin.go:26-27` picks its own two values with no
  stated rule; `PRD.md` locks the module path, the three extras entries,
  and the match vocabulary but is silent on these two strings.
- **Resolution taken:** `source_type = "gdrive"` and `display_name =
  "Google Drive"` — lowercase, no hyphen, matching the shape of every
  illustrative value the contract itself shows, with the human-readable
  name matching the product name an operator would recognize in the
  host's add-source form. Record explicitly that this value is
  user-visible and must stay stable across all five phases once chosen,
  since it is retained as provenance on every item the plugin later
  emits.
- **Proposed disposition:** `documentation-fixable` — the contract could
  state a naming convention (or state that the plugin author chooses
  freely) in the `DescribeResponse` field documentation without any
  wire-level change.

### GAP-04

- **Date:** 2026-08-16
- **Question:** How does a plugin whose three declared `ExtrasField`
  entries are all `required: true` reconcile the contract's "fail startup
  loudly when a required key is empty" discipline with the requirement
  that `Describe` succeed against a host trial launch where the operator
  has configured nothing at all?
- **What the inputs say:** `contract/plugin-contract.md:488-501` states
  the kernel trial-launches the binary and calls `Describe` against fields
  the operator has typed but not yet saved, and that a well-behaved
  `Describe` is idempotent and side-effect-free regardless of call site;
  `contract/plugin-contract.md:436-442` and `contract/mock/main.go:22-29`
  show the mock deliberately not failing on missing config while noting
  that every real plugin does fail loudly; `contract/plugin.proto:83-88`
  says `required` is advisory only and the kernel never rejects a saved
  config missing a required extras key. Nothing in the four inputs
  reconciles these for a plugin whose extras are all required, and
  `contract/mock/` cannot serve as a worked example because it declares
  zero extras and therefore never faces the tension.
- **Resolution taken:** Phase 1 performs no configuration read and no
  required-key validation anywhere — not in `main`, not in `Describe`.
  `Describe` is therefore unconditionally successful and side-effect-free
  regardless of configured state, satisfying Phase 1 success criterion 3.
  Fail-loud-on-missing-required-key enforcement is deferred to the point a
  real Drive call is actually attempted (`Match`/`Health`, Phase 2 and
  later), where a missing credential is a genuine failure rather than an
  expected state during an add-source form fill. Record explicitly that
  this defers work: a later phase must add that validation and must test
  it against a real host's trial launch rather than assuming the marker's
  behavior.
- **Proposed disposition:** `documentation-fixable` — the contract's
  Configuration section could state which RPCs the fail-loud discipline
  applies to, and confirm that a trial launch must never be failed for
  absent configuration.

### GAP-05

- **Date:** 2026-08-16
- **Question:** Does the host's own launch mechanism tolerate a plugin
  binary that dispatches on its first argument — running its
  authorization flow and exiting on `auth`, and falling through to serve
  mode on anything else, including zero arguments?
- **What the inputs say:** `contract/plugin-contract.md` documents
  discovery and launch and states the kernel launches a discovered
  binary, but says nothing at all about arguments — it neither guarantees
  zero arguments nor forbids a plugin from inspecting them;
  `contract/mock/main.go` never touches `os.Args`, so the reference
  implementation offers no worked example; `contract/plugin.proto` is a
  wire contract with no launch semantics; `PRD.md:246-254` states the
  dual-mode shape as design guidance and flags it `[Unproven]` precisely
  because none of the four inputs answer it.
- **Resolution taken:** Confirmed empirically against a real, locally
  installed `topos` host (operator-supplied binary, per `01-CONTEXT.md`
  D-01) — **yes, the host tolerates this shape.** With a pinned,
  external-tier install of this plugin's binary configured as
  `[sources.gdrive] plugin`, the host launched the binary with zero
  arguments (its ordinary launch path — nothing in the config or CLI
  instructed it to run `auth`) and it reached serve mode cleanly:
  `GET /api/sources` reported a correct `source_type: "gdrive"` (proving
  a successful post-handshake `Describe` call) and the host's own
  scheduled-sync attempts surfaced this plugin's own stub `Match` error
  verbatim (`"topos-plugin-gdrive: not yet implemented (Phase 1
  skeleton)"`), proving `Match` was also successfully dispatched to the
  same subprocess. No hang, crash, or argument-shape complaint was
  observed anywhere in the host's logs across multiple sync-retry cycles.
  Full verbatim record: `docs/smoke-test-phase1.md`.
- **Proposed disposition:** `documentation-fixable` — the contract's
  "Discovery and launch" section could state explicitly that a plugin
  subprocess is always launched with zero arguments (or, if that is not
  guaranteed, that plugin authors must not assume any particular `argv`
  shape), closing this `[Unproven]` question for every future plugin
  author without requiring each one to smoke-test it independently as
  this repository just did.

### GAP-06

- **Date:** 2026-08-16
- **Question:** How does a plugin whose entire declared configuration
  lives in `extras` (this plugin's three PRD-locked fields: `client_id`,
  `client_secret`, `folder_id`) satisfy the kernel's own config-load
  requirement that every `[sources.<id>]` entry declare at least one of
  `base_url`+`token` or `path`?
- **What the inputs say:** `contract/plugin-contract.md:417-422` states
  plainly: "a source declaring none of `base_url`, `token`, or `path`
  still fails config load — every source must declare at least one
  recognized connection-detail shape." This is explicit, not silent — but
  `PRD.md`'s own "Declared Configuration" section (`PRD.md:114-144`)
  locks exactly three `ExtrasField` entries and states "no row is added
  anywhere... if this plugin's `Describe` implementation is correct, the
  host needs zero code written specifically for it," without ever
  mentioning a required top-level `base_url`/`token`/`path` key alongside
  those three extras. Neither `contract/plugin.proto` nor
  `contract/mock/` (which declares zero extras and therefore never faces
  this tension) offers a worked example of an extras-only source
  satisfying this requirement.
- **Resolution taken:** Empirically confirmed via the Phase 1 host smoke
  test: hand-authoring `[sources.gdrive]` with only `plugin` and `extras`
  (no `base_url`/`token`/`path`) makes the host refuse to boot at all —
  `topos: config: source "gdrive" must declare either base_url and
  token, or path`, exit 1, before any plugin subprocess is even launched.
  Worked around for that walkthrough only (never in the committed
  `testdata/topos-smoke-config.toml`) by adding a functionally inert
  `path = "unused-in-phase-1"` stub to a throwaway config copy — Phase
  1's plugin never reads `WEBSPACES_SOURCE_CONFIG` at all (GAP-04), so
  this has zero behavioral effect this phase. This is recorded as an
  open item rather than a permanent resolution: a later phase (or a PRD
  correction on the topos side) should decide whether this plugin
  declares a real `path`/`base_url`+`token` value with actual meaning, or
  whether the kernel's own validation should exempt fully-`extras`
  sources. This repository's own `testdata/topos-smoke-config.toml` is
  deliberately left in its PRD-faithful, three-extras-only shape pending
  that decision.
- **Proposed disposition:** `contract-change` (or a PRD-side correction)
  — closing this gap requires either the kernel's config-load validation
  to exempt sources with a fully-populated `extras` table (a wire/kernel-
  level change on the topos side) or `PRD.md`'s own Declared
  Configuration section to specify a fourth top-level key this plugin
  must also declare (a correction to the hand-off document, made on the
  topos side per `CLAUDE.md`'s rule that `PRD.md` corrections happen
  there, not here).
- **Observed outcome (2026-08-17, Phase 3 live UAT):** the refusal still
  stands on the current host build — launching `topos serve` against the
  committed three-extras-only smoke config fails config load with the
  same shape message, now suffixed with the unresolved reference:
  `source "gdrive" must declare either base_url and token, or path
  (missing environment variable(s): GDRIVE_SMOKE_FOLDER_ID)`. Two new
  facts observed, neither documented in the four inputs: (1) the kernel
  reports unresolved `${...}` environment references in `extras` at
  config load, before any plugin subprocess launches; (2) the operator
  expectation this gap records — that a Describe-driven plugin should be
  configurable entirely from the host UI with no hand-authored
  `[sources.<id>]` block — was independently voiced by this repository's
  operator when hitting the refusal, reinforcing that the eventual fix
  belongs on the topos side (UI/kernel), not in a fourth declared key
  invented here.

### GAP-07

- **Date:** 2026-08-16
- **Question:** How does a plugin subprocess launched by the host obtain
  the OAuth `client_id`/`client_secret` it needs to refresh an access
  token, given that the launch environment is reduced to a fixed
  allowlist plus the values behind `${VAR}` references the instance's own
  config declares?
- **What the inputs say:** `PRD.md`'s Locked Decisions bind
  `GDRIVE_CLIENT_ID`/`GDRIVE_CLIENT_SECRET` to the standalone `auth`
  subcommand only and say nothing about serve mode; the contract's
  launch-environment section forecloses the raw-shell path structurally;
  the SDK ships no config-parsing helper; `contract/mock/` declares zero
  extras and so never faces the question.
- **Resolution taken:** Serve mode reads `client_id` and `client_secret`
  out of the `extras` object inside `WEBSPACES_SOURCE_CONFIG` (the kernel
  having already expanded the operator's `${GDRIVE_CLIENT_ID}` /
  `${GDRIVE_CLIENT_SECRET}` references), with a documented secondary
  fallback to the same two raw environment variables for direct local
  invocation; implemented in plan 02-02. Note explicitly that this brings
  forward part of the `WEBSPACES_SOURCE_CONFIG` parsing GAP-04 deferred to
  "Phase 2+", and that GAP-04's own resolution is preserved — no
  configuration read happens in `main` or in `Describe`, and startup is
  never failed for absent configuration.
- **Proposed disposition:** `documentation-fixable`.
- **observed outcome (2026-08-16, plan 02-03):** the extras-resolution
  code itself (`sourceConfig.clientCredentials`, extras-first with a
  raw-env fallback) is proven offline end to end (`sourceconfig_test.go`,
  9 tests, all passing). The specific claim this bullet was asked to
  record — that the extras path was exercised end to end against a real
  subprocess launched with only the contract's nine allowlisted
  variables plus a `WEBSPACES_SOURCE_CONFIG` payload — was **not**
  exercised this phase or the previous one: `integration_test.go`'s
  `TestServeMode_MintsAnAccessTokenFromThePersistedTokenWithNoBrowser`
  is coded for exactly that scenario and is opt-in via
  `GDRIVE_LIVE_TOKEN_TEST=1`, but its own precondition check (real
  `GDRIVE_CLIENT_ID`/`GDRIVE_CLIENT_SECRET` exported) has failed loudly,
  never proceeded to actually launch the subprocess, in both the plan
  02-02 and plan 02-03 executor sessions — neither session's shell held
  real Google OAuth credentials. What was confirmed instead: the test
  skips cleanly with the flag unset, and fails loudly (not silently) with
  the flag set and credentials absent. The live subprocess run — proving
  the extras path actually resolves a real credential and mints a real
  access token inside a subprocess denied every other route to one — is
  still outstanding and requires the operator to run `make verify-token`
  with their own exported credentials.
- **observed outcome (2026-08-17, phase-2 UAT):** the live leg described
  above is no longer outstanding. The operator ran `make verify-token`
  with their own exported credentials and reported it passing (recorded
  in `.planning/phases/02-authorization/02-UAT.md`, test 1, result:
  pass). The 2026-08-16 bullet above is retained unedited per this
  file's append-only discipline; this bullet supersedes its "still
  outstanding" clause.

### GAP-08

- **Date:** 2026-08-16
- **Question:** A plugin that resolves its private state directory per
  the XDG Base Directory specification resolves `$XDG_DATA_HOME/...` in
  the operator's own shell but, in a host-launched subprocess, resolves
  `$HOME/.local/share/...` instead — because `HOME` is allowlisted and
  `XDG_DATA_HOME` is not. Which path is a plugin supposed to use, given
  GAP-01 already established the contract says nothing about
  plugin-private state at all?
- **What the inputs say:** The launch-environment allowlist enumerates
  exactly `PATH`, `HOME`, `LANG`, `LC_ALL`, `LC_CTYPE`, `TZ`, `TMPDIR`,
  `XDG_RUNTIME_DIR`, `DBUS_SESSION_BUS_ADDRESS` — `XDG_DATA_HOME` is not
  among them, and the document does not connect that omission to GAP-01's
  unaddressed plugin-private-state question.
- **Resolution taken:** One shared resolver used by both runtime
  contexts, preferring `XDG_DATA_HOME` when non-empty and falling back to
  `$HOME/.local/share`; because the two contexts can therefore disagree
  when an operator sets a non-default `XDG_DATA_HOME`, the `auth`
  subcommand additionally recomputes the path a host-launched subprocess
  would resolve and, when it differs, prints a loud warning naming the
  variable, both absolute paths, and the remedy — after a successful
  write, never instead of one, and never containing a token or secret
  value.
- **Proposed disposition:** `documentation-fixable` — adding
  `XDG_DATA_HOME` to the allowlist, or stating that plugin-private state
  must be resolved from `HOME` alone, closes it with no wire-level
  change.
- **observed outcome (2026-08-16, plan 02-03):** not conclusively
  observed. Per `.planning/phases/02-authorization/02-01-SUMMARY.md`
  Task 3, the operator's live-run report was the single summary-form
  confirmation "everything above passed," not the six verbatim per-item
  observations the checkpoint asked for; item (e) of that summary's own
  table records the divergence-warning text specifically as "not
  independently confirmed as absent or present" and states that "exact
  warning text, if any appeared, was not transcribed." The operator's
  actual `XDG_DATA_HOME` state during that run (set to a non-default
  value, or unset) was likewise not recorded. This gap's resolution
  itself — the shared resolver plus the loud divergence warning
  (`warnIfDataDirIsInvisibleToServeMode`) — is unit-tested offline
  (`auth_test.go`'s `TestWarnIfDataDirIsInvisibleToServeMode_*` pair,
  covering both the warns-on-divergence and silent-when-equal branches),
  but whether the warning actually fired during the one real live run
  this repository has performed is not on the record.

### GAP-09

- **Date:** 2026-08-17
- **Question:** What per-item literal set satisfies `PRD.md`'s three
  simultaneous Match-value constraints for the `folders` field ("resolved
  folder-path segments relative to the configured root folder", "exact
  literals — never globs, never prefix or substring matching", and
  "everything synced by this instance is expressed by matching against the
  root folder's own name")?
- **What the inputs say:** `PRD.md:150-158` states all three constraints
  but never states the per-item value-set algorithm that satisfies them
  simultaneously — it gives two illustrative standalone strings (`Reports`,
  `Reports/2026`) for one worked example, never that example's *complete*
  value list. `contract/plugin-contract.md`'s Match section states the
  comparison rule (exact, case-insensitive, D-04) but is a general
  mechanism description, not specific to this plugin's own folder-path
  semantics. `contract/mock/plugin.go` declares a flat, pre-populated
  `Labels` slice per fixed item with no path-resolution concept at all, so
  it offers no worked example either.
- **Resolution taken:** A planning-time decision checkpoint (03-01-PLAN.md
  Task 1) presented three candidate algorithms and selected **Option A —
  the cumulative ancestor-chain value set**: each item exposes the
  configured root's own name, plus every cumulative relative path prefix
  from the root down to (not including) the item's own name. Worked
  example: configured root folder named `Team Docs`, item at
  `Team Docs/Reports/2026/q1.pdf` — the item exposes exactly
  `["Team Docs", "Reports", "Reports/2026"]`. This is the only one of the
  three candidates that satisfies all three PRD constraints
  simultaneously: the root's own name matches everything synced by this
  instance; `Reports` matches the whole `Reports` subtree at any depth;
  `Reports/2026` narrows precisely to that one subtree; every comparison
  stays an exact literal, never a glob, prefix, or substring match. It is
  also the only option under which both of PRD's own illustrative values
  (`Reports` AND `Reports/2026`) are simultaneously usable against the
  same nested item. The accepted cost: a deeply nested item exposes N+1
  values in `Item.labels`, and two different subtrees that share a folder
  name at the same relative depth expose the same literal (both
  `Reports`), so a configured value can match across sibling branches —
  judged an acceptable, disclosed tradeoff against the alternative of an
  operator having to enumerate every depth by hand (the rejected Option B)
  or losing all path-scoping (the rejected Option C, where `Reports/2026`
  would never match anything at all). Implemented in `match.go`'s
  `ancestorChainValues`.
- **Proposed disposition:** `documentation-fixable` — `PRD.md`'s own Match
  Vocabulary section could show one fully worked multi-level example (an
  item's *complete* value list, not two standalone illustrative strings)
  to remove this ambiguity for any future reader.

### GAP-10

- **Date:** 2026-08-17
- **Question:** Are Drive folder objects (`mimeType ==
  "application/vnd.google-apps.folder"`) ever emitted as `Item`s from
  `Match`, or are they structural-only, never surfaced to the host?
- **What the inputs say:** Neither `contract/plugin-contract.md` nor
  `PRD.md` explicitly excludes folder objects from the Match item set. The
  `Item` message (contract/plugin-contract.md's "The `Item` message"
  section) has no folder-specific semantics — no content field, no
  preview concept distinct from a document's — and `PRD.md`'s Drive API
  Surface table frames `files.list` as "the baseline folder walk —
  establishing and maintaining the set of files under the configured
  root", implying folders are the walk's own traversal mechanism, not
  members of the set it establishes.
- **Resolution taken:** Folders are structural only and are never emitted
  as `Item`s. `match.go`'s `matchItems` explicitly skips every tree node
  whose `MimeType` is `folderMimeType` before building an `Item` — folder
  nodes remain in the persisted tree (the ancestor-chain membership walk
  needs them) but never reach the returned `MatchResponse`.
- **Proposed disposition:** `documentation-fixable` — `PRD.md`'s Drive API
  Surface table could add one sentence clarifying that folder objects are
  structural only, never surfaced as Items, closing this for any future
  folder/tree-shaped source plugin.

### GAP-11

- **Date:** 2026-08-17
- **Question:** At what point relative to the first `files.list` walk
  should a plugin capture `changes.getStartPageToken` — before the walk
  begins, at some point during it, or only after it completes?
- **What the inputs say:** Neither `contract/plugin-contract.md` nor
  `PRD.md` specifies the ordering. `PRD.md`'s Drive API Surface table
  states `changes.getStartPageToken` "establishes the starting point for
  the incremental delta feed" but says nothing about timing relative to
  the initial full walk.
- **Resolution taken:** Captured immediately BEFORE the first
  `files.list` walk begins (`syncengine.go`'s `ensureSynced` calls
  `startPageToken` before `walkFolder`), so a file added to the folder
  DURING a slow first walk is redelivered by the very next `changes.list`
  poll rather than silently falling into the gap between "the walk
  observed the tree as of some moment" and "the token started tracking
  changes as of some later moment." The accepted cost: at-most-once
  redundant reprocessing of a change already reflected in the walked tree
  — harmless, because the tree is keyed by Drive file id and a redundant
  update is idempotent.
- **Proposed disposition:** `documentation-fixable` — a one-sentence note
  in a future contract revision's guidance for incrementally-synced
  plugins (capture the delta-feed starting token before, never after, the
  initial full establishment of state) would remove this ambiguity for
  every folder/tree-shaped source plugin, not just this one.

### GAP-12

- **Date:** 2026-08-17
- **Question:** What value should a plugin with no `base_url` of any kind
  report for the `source_system` provenance key?
- **What the inputs say:** `contract/plugin-contract.md`'s Provenance
  subsection shows `"source_system": p.baseURL` and describes it as "the
  source instance this came from", implicitly assuming every plugin has a
  `baseURL` field. `contract/mock/plugin.go` substitutes the literal
  `mock://in-memory` with no stated rule for what a plugin without a real
  base URL should use instead. This plugin has no base URL of any kind —
  it targets a single Drive folder by id, not a network endpoint.
- **Resolution taken:** The configured folder's own canonical Drive URL,
  `https://drive.google.com/drive/folders/<folder_id>`, computed fresh at
  `Match` time from the persisted `syncState.RootID` (which is always the
  configured `folder_id`) — since the configured folder IS this plugin's
  source instance, and the value is both stable across a given instance's
  lifetime and non-secret (a folder id is already visible in `Describe`'s
  own placeholder text). Implemented as `plugin.go`'s `sourceSystem`
  function rather than a package-level constant, since — unlike
  `sourceType`/`displayName` — the value is not knowable at compile time;
  it depends on the operator's own configured folder id.
- **Proposed disposition:** `documentation-fixable` — `contract/plugin-contract.md`'s
  Provenance subsection could name the fallback for a plugin with no
  `base_url`-shaped connection detail (e.g. "the most specific stable
  identifier your plugin's own configuration provides"), closing this for
  any future local-path or single-target source plugin.

### GAP-13

- **Date:** 2026-08-17
- **Question:** When a plugin stops returning a previously-returned item
  from `Match` because that item has left the source's configured scope,
  does the kernel remove the item from its index, or does the
  previously-indexed row — with the `deep_link` the plugin supplied —
  persist indefinitely? Equivalently: is each `Match` response the
  authoritative full current set the kernel reconciles its index against,
  or is it additive?
- **What the inputs say:** `contract/plugin-contract.md`'s `Match` section
  gives three matching rules (read only declared keys; exact and
  case-insensitive; an empty value list matches nothing) and states
  `Match` is called only at sync time, never at request time — but says
  nothing about what the kernel does with items a previous sync returned
  and the current sync does not. The `Item` table's fidelity row and the
  `LINK_FIDELITY_UNSPECIFIED` row describe per-item admission only ("this
  specific item is skipped and logged; the rest of that sync's valid items
  still persist") — "persist" is used about the current sync's valid
  items, never about reconciliation or removal of previously persisted
  rows. `contract/plugin.proto`'s `MatchResponse` is a bare
  `repeated Item items` with no removal, tombstone, or generation field.
  `contract/mock/`'s fixed in-memory item set never shrinks, so the
  reference plugin exercises no removal path at all.
- **Resolution taken:** Implement the stricter reading: `Match` returns
  the authoritative full current set on every call (which SYNC-04 already
  requires of this plugin) and an out-of-scope item is simply absent from
  every subsequent response. This matters acutely here because this
  plugin's entire folder-membership filter — the access-control boundary
  plan 03-04 exists to repair — is expressed solely by ceasing to return
  an item; if the kernel's index is additive, excluding an out-of-scope
  document revokes nothing already indexed, and the `deep_link` the host
  persisted stays openable. This plugin has no contract-provided mechanism
  to request removal of an already-indexed row and does not invent one. If
  the kernel turns out to be additive, the residual staleness is a
  kernel-side concern this entry hands back to the topos project rather
  than something this plugin can close from its own side.
- **Proposed disposition:** `documentation-fixable` — one sentence in the
  contract's `Match` section stating whether the kernel reconciles its
  index against each sync's full response would close it. If the honest
  answer turns out to require a wire-level removal signal (the kernel is
  additive and revocation needs a tombstone or generation marker in
  `MatchResponse`), the disposition becomes `contract-change`.
- **Observed outcome (2026-08-18, plan 14-05 gap triage, this repository):**
  resolved by reading the kernel's own index code, not by guessing:
  `kernel/index/store.go`'s `ReplaceWebspaceSourceItems` — the one call
  every sync makes per (webspace, source instance) pair
  (`kernel/correlate/correlate.go:157`) — upserts the items a `Match` call
  returned and then, in the SAME transaction, deletes every prior
  `webspace_items` row for that exact (webspace, source) pair before
  reinserting rows only for the items just returned (`store.go:199-234`,
  its own doc comment: "replaces ONLY the webspace_items rows for
  (webspaceName, source)... a sync that fails or is interrupted leaves the
  pre-sync item set... intact, never a partially-written set"). The
  kernel's reconciliation is confirmed **full-replace, never additive**:
  this plugin's own stricter-reading resolution above was correct, and no
  wire-level removal signal is needed. Disposition is `documentation-fixable`
  on the strength of this reading, not `contract-change`.

### GAP-14

- **Date:** 2026-08-17
- **Question:** How long should the bounded preview `Match` returns in
  `Item.preview` be — in bytes fetched from Drive, and in the final
  returned string length?
- **What the inputs say:** Neither `contract/plugin-contract.md` nor
  `PRD.md` names an exact number. The contract's `Item.preview` field doc
  says only that it is "a bounded snippet (hundreds of characters, not the
  full document/message)" — a qualitative order of magnitude, not a value a
  plugin can implement directly. `contract/mock/plugin.go` never builds a
  preview from live bytes (its fixed items carry no preview-length
  precedent), and `contract/plugin.proto` is a wire contract with no
  length constant of its own.
- **Resolution taken:** A 500-rune bound (`previewRuneLimit`), built by
  truncating an 8192-byte raw `files.get` `Range` fetch window
  (`previewRangeBytes`) — both plugin-local constants declared in
  `preview.go`. 500 runes sits squarely inside "hundreds of characters,
  not the full document"; 8192 raw bytes is comfortably enough to contain
  500 runes of ordinary text after any multi-byte UTF-8 expansion, with
  headroom to spare, while staying a small, cheap fetch.
- **Proposed disposition:** `documentation-fixable` — the contract's
  `Item.preview` field doc could name an exact target length (or a
  suggested range) instead of only a qualitative description, removing
  this repo-local guess for every future plugin author who must otherwise
  pick their own number.

### GAP-15

- **Date:** 2026-08-17
- **Question:** CONT-01 requires "a bounded preview fetched via
  `files.get`" for regular (non-Workspace) files — but which Drive MIME
  types is a plugin expected to actually attempt that fetch for? A binary
  file (a PDF, an image, an office-binary format) has no obvious
  "preview text" `files.get`'s raw bytes alone can produce.
- **What the inputs say:** Nothing beyond CONT-01's own wording. Neither
  `contract/plugin-contract.md` nor `PRD.md` names a MIME allowlist or
  otherwise scopes which regular files get a preview attempt versus which
  ones don't. `PRD.md`'s own "Recommended Stack" section states plainly
  that no third dependency is recommended for this plugin, which rules out
  bundling a PDF/office-binary/OCR text-extraction library as the way to
  produce a preview for a non-text file.
- **Resolution taken:** A text-shaped MIME allowlist: a MIME type carrying
  the `text/` prefix, or exactly `application/json`, gets a live
  `files.get`-derived preview; every other regular-file MIME type gets an
  empty `Item.preview` and this plugin makes zero Drive calls for it. This
  is the only choice consistent with "no third dependency recommended" —
  extracting readable text from a binary format without a dedicated
  library is not something this plugin attempts.
- **Proposed disposition:** `documentation-fixable` — the contract's
  `Item.preview` field doc, or a new guidance section for content-fetching
  source plugins, could name (or explicitly leave open, with a suggested
  default) which MIME types a plugin is expected to attempt a preview for,
  closing this scope question for every future plugin author who reaches
  the same "no bundled text-extraction library" constraint.

### GAP-16

- **Date:** 2026-08-17
- **Question:** CONT-02 requires a Workspace document (Doc, Sheet, or
  Slide) be previewable "via `files.export` to a concrete format" — but
  which concrete MIME type should each of the three types export to?
- **What the inputs say:** Nothing beyond the qualitative phrase "a
  concrete format." Neither `contract/plugin-contract.md` nor `PRD.md`
  names an export MIME type for any Workspace type. `contract/mock/`
  never builds a preview from live bytes, so it offers no worked example
  either.
- **Resolution taken:** `text/plain` for
  `application/vnd.google-apps.document`, `text/csv` for
  `application/vnd.google-apps.spreadsheet`, and `text/plain` for
  `application/vnd.google-apps.presentation` — extractable text, chosen so
  a Match-time preview and a Fetch-time content result are the same
  substance and so no `text/html` rendition is ever produced (which would
  drag in the contract's `content_shape` requirement this plugin has no
  shape to fill). Implemented as the `workspaceExportMIME` table in
  `workspaceexport.go`, consulted identically by both `Match`'s
  `exportPreview` and `Fetch`'s `fetchWorkspaceDoc`.
- **Proposed disposition:** `documentation-fixable` — the contract's
  `Item.preview`/`FetchResponse` field docs could name a suggested export
  MIME type per Workspace type (or explicitly state the choice is left to
  the plugin author), closing this ambiguity for every future Workspace-
  aware source plugin.

### GAP-17

- **Date:** 2026-08-17
- **Question:** CONT-02 names exactly three Workspace types (Docs, Sheets,
  Slides) as in scope — but neither the contract nor `PRD.md` states what a
  plugin should do with every OTHER `application/vnd.google-apps.*` MIME
  type Drive can return (Drawings, Forms, Sites, Jamboard, Fusion Tables,
  My Maps, Apps Script, shortcuts, and others), several of which Google's
  own API can technically export.
- **What the inputs say:** `PRD.md`'s own success criterion 2 scopes
  CONT-02 explicitly to "Docs/Sheets/Slides" and says nothing about any
  other Workspace type. `contract/plugin-contract.md` and
  `contract/plugin.proto` are silent on Workspace MIME type enumeration
  entirely — a source plugin's own domain, not the wire contract's.
  `contract/mock/` declares no Workspace-native fixture at all.
- **Resolution taken:** Exactly the three PRD-named types get a
  `workspaceExportMIME` table entry; every other
  `application/vnd.google-apps.*` MIME type — including `drawing`
  (technically exportable by Drive but outside PRD's stated CONT-02 scope)
  and `shortcut` (which this plugin deliberately never resolves to its
  target) — is a declined format, decided by table lookup alone, before
  any Drive call is ever issued for that file. `Match`'s `exportPreview`
  degrades such a file to an empty preview; `Fetch`'s `fetchWorkspaceDoc`
  returns `available: false` with the `unavailableDeclinedFormat` reason
  (GAP-18).
- **Proposed disposition:** `documentation-fixable` — the contract could
  state (or explicitly leave open) which Workspace MIME types a
  content-fetching source plugin is expected to support versus decline,
  closing this scope question for every future Workspace-aware plugin
  author.

### GAP-18

- **Date:** 2026-08-17
- **Question:** CONT-03 requires a document over the export ceiling, or in
  a declined format, to return "a named reason" — but neither the four
  inputs nor `PRD.md` names verbatim text for either reason, nor states
  whether `FetchResponse.unavailable_reason` is free text at all or is
  expected to come from a shared vocabulary the host itself renders.
- **What the inputs say:** `contract/plugin-contract.md`'s Fetch section
  (lines 783-798, read this session) states `available = false` with a
  populated `unavailable_reason` is "a normal, expected outcome... not an
  error," giving one illustrative example ("a document type your source
  can't render a preview for") but no verbatim string. `PRD.md`'s own
  locked decisions pin four EXACT verbatim sentences for Phase 5's `Health`
  states but say nothing about `Fetch`'s `unavailable_reason` field at all
  — unlike those four sentences, no permitted input treats this field as
  carrying contract-mandated text. `contract/mock/plugin.go` never returns
  `available: false` for any fixture item, offering no worked example.
- **Resolution taken:** Two stable, distinguishable, plugin-local free-text
  constants — `unavailableExportCeiling` and `unavailableDeclinedFormat`,
  declared in `workspaceexport.go` — deliberately worded so neither can be
  confused with any of the four Phase 5 verbatim health sentences or the
  four Phase 2 status constants (`plugin.go:64-69`). A guard test
  (`workspaceexport_test.go`) pins both constants non-empty, distinct from
  each other, and distinct from all eight of those other strings.
- **Proposed disposition:** `documentation-fixable` — the contract's
  `FetchResponse.unavailable_reason` field doc could state whether the
  value is free text chosen by the plugin author or is expected to draw
  from a shared, host-rendered vocabulary, closing this ambiguity for every
  future content-fetching source plugin that has more than one distinct
  unavailable cause to report.

### GAP-19

- **Date:** 2026-08-18
- **Question:** `HealthResponse.last_sync_unix` is declared by the wire
  contract (`contract/plugin-contract.md:803`), but no permitted input
  states whether a plugin is expected to populate it, from which clock, or
  what the host renders when it is zero.
- **What the inputs say:** `contract/plugin.proto`'s `HealthResponse`
  message declares the field (`int64 last_sync_unix = 2`) with no doc
  comment of its own. `contract/plugin-contract.md`'s `Health` section
  documents `reachable` and `last_error` in prose but never mentions
  `last_sync_unix` at all. `PRD.md`'s own Health States table names only
  `reachable` and `last_error`; it is silent on this third field entirely.
  `contract/mock/plugin.go` does not implement a real `Health` RPC with a
  worked example to copy from.
- **Resolution taken:** Report the persisted sync-state file's own
  modification time in Unix seconds when that file exists, and `0` when it
  does not — no schema change to `syncstate.json`, no additional Drive
  call, one filesystem `stat` on the already-resolved `syncStatePath`.
  Implemented as `(*SourcePlugin).lastSyncUnix()`.
- **Proposed disposition:** `documentation-fixable` — the contract's
  `HealthResponse.last_sync_unix` field doc could state which clock/event a
  plugin should report (last successful sync completion is the most
  natural reading) and what a zero value is understood to mean, closing
  this for every future plugin author who reaches this field.

### GAP-20

- **Date:** 2026-08-18
- **Question:** PRD.md's Open Question 2 (RPC-06): how finely do Google's
  own token-refresh error responses let a client distinguish "revoked by
  the user" from "expired from inactivity" from "the OAuth application
  itself was un-published or deleted"?
- **What the inputs say:** Nothing — this is a question about a third
  party's (Google's) API behavior, not something any of the four permitted
  clean-room inputs could ever answer; they document this plugin's own
  contract surface, not Google's OAuth2 token endpoint's response shape.
  `PRD.md:182-187` names this open question explicitly and requires it be
  resolved empirically, against a real revoked or expired token, before
  treating the answer as final.
- **Resolution taken:** Pre-spike best guess (05-RESEARCH.md Assumption
  A1, `[CITED]` from a third-party engineering blog, cross-checked against
  this repository's own existing `secrets_test.go:249` fixture): Google's
  token endpoint returns the same generic `invalid_grant` /
  `"Token has been expired or revoked."` body regardless of which of the
  three sub-causes actually occurred. This is a starting hypothesis, not a
  settled finding — the live observation an operator obtains by running
  `make spike-auth-failure` against a real, deliberately revoked token
  (plan 05-01 Task 3's harness, `authfailurespike_test.go`) will be
  APPENDED as a dated bullet below by plan 05-03, never substituted for
  this text. Regardless of what that live run observes, this plugin's own
  design collapses all three sub-causes into the single
  `stateExpiredRevoked` health state and its one PRD-mandated sentence —
  `PRD.md:175-180` specifies exactly four sentence-bearing rows and
  `PRD.md:167-170` forbids inventing a fifth, so even full indistinguishably
  changes nothing about what this plugin reports.
- **Proposed disposition:** `documentation-fixable` — this is fundamentally
  a question about Google's own API, not the topos contract; there is
  nothing for the topos project's own documentation to fix. Recorded here
  only because `CLAUDE.md`'s gap-logging obligation requires every
  question the four permitted inputs cannot answer to be logged, not only
  the ones a contract correction could actually close.
- **Observed outcome (2026-08-18, plan 05-03 live spike):** the operator
  revoked this plugin's access at
  <https://myaccount.google.com/permissions> and ran
  `make spike-auth-failure` against a real Google account
  (`docs/rpc06-spike.md`'s Protocol, Step 2's primary path). The observed
  refresh-error response: HTTP status **400**, `RetrieveError.ErrorCode` =
  **`invalid_grant`**, `RetrieveError.ErrorDescription` = **`Token has
  been expired or revoked.`**, no error URI populated, and a **91-byte**
  raw response body. This is byte-for-byte the pre-spike best guess
  (Assumption A1) above, for the one sub-cause actually exercised
  (user-revoked access). The Protocol's optional Step 6 — a second,
  independently-caused observation against a naturally
  Testing-status-expired token, rather than a user-revoked one — was not
  performed, so this observation confirms the pre-spike guess for the
  revoked-access sub-cause specifically and does not, on its own,
  establish whether Google's response actually varies across the three
  named sub-causes (revoked / expired-from-inactivity / app-deleted).
  This plugin's own design already collapses all three sub-causes into
  the single `stateExpiredRevoked` health state and its one PRD-mandated
  sentence regardless of what a second observation would have shown, so
  the practical resolution recorded above is unchanged either way. Full
  verbatim record: `docs/rpc06-spike.md`.

---

## Part 2 — Additions from the live UAT run (never in the plugin repository's log)

**None.** `14-LIVE-UAT.md`'s own "Anything to carry into the gap triage
(14-05)" line (its Criterion 4 results-table footnote) reads: "None
reported. The operator's report was a blanket 'everything passes' with no
additional detail or gap-log reference supplied." The operator's live run
against a real Google account surfaced nothing beyond what the plugin
repository's own clean-room sessions already logged as GAP-01 through
GAP-20 above. This is stated explicitly rather than left implicit, per
this plan's own instruction for the no-additions case.

---

## Part 3 — Triage table

One row per gap id. Every `documentation-fixable` row names the section of
`docs/plugin-contract.md` its answer landed in (Task 2, this plan). Every
`contract-change` row names the backlog item it was filed as (Task 3, this
plan).

| ID | Disposition | Landing place |
|----|-------------|----------------|
| GAP-01 | documentation-fixable | `docs/plugin-contract.md` — new "Plugin-private state" section |
| GAP-02 | documentation-fixable | `docs/plugin-contract.md` — new "Plugin-private state" section |
| GAP-03 | documentation-fixable | `docs/plugin-contract.md` — "Describe" (`DescribeResponse` field docs) |
| GAP-04 | documentation-fixable | `docs/plugin-contract.md` — "Configuration: `WEBSPACES_SOURCE_CONFIG`" |
| GAP-05 | documentation-fixable | `docs/plugin-contract.md` — "Discovery and launch" |
| GAP-06 | contract-change | ROADMAP.md Backlog — Phase 999.1: Plugin distribution, dev guide, certification (item cites GAP-06) |
| GAP-07 | documentation-fixable | `docs/plugin-contract.md` — "Configuration: `WEBSPACES_SOURCE_CONFIG`" (extras subsection) |
| GAP-08 | documentation-fixable | `docs/plugin-contract.md` — new "Plugin-private state" section |
| GAP-09 | documentation-fixable | `docs/plugin-contract.md` — "Match" |
| GAP-10 | documentation-fixable | `docs/plugin-contract.md` — "Match" |
| GAP-11 | documentation-fixable | `docs/plugin-contract.md` — "Match" |
| GAP-12 | documentation-fixable | `docs/plugin-contract.md` — "Provenance" |
| GAP-13 | documentation-fixable | `docs/plugin-contract.md` — "Match" (resolved by reading `kernel/index/store.go`'s `ReplaceWebspaceSourceItems` — confirmed full-replace, not additive; see this gap's 2026-08-18 observed-outcome bullet above) |
| GAP-14 | documentation-fixable | `docs/plugin-contract.md` — "The `Item` message" (`preview` field) |
| GAP-15 | documentation-fixable | `docs/plugin-contract.md` — "Fetch" |
| GAP-16 | documentation-fixable | `docs/plugin-contract.md` — "Fetch" |
| GAP-17 | documentation-fixable | `docs/plugin-contract.md` — "Fetch" |
| GAP-18 | documentation-fixable | `docs/plugin-contract.md` — "Fetch" |
| GAP-19 | documentation-fixable | `docs/plugin-contract.md` — "Health" |
| GAP-20 | documentation-fixable | `docs/plugin-contract.md` — "What this document does not cover" |

**Reconciliation:** 20 imported entries (GAP-01 through GAP-20) + 0 UAT
additions = 20 rows in the triage table above. 19 documentation-fixable, 1
contract-change (GAP-06), 0 not-a-gap. 20 = 20. Nothing dropped.

---

*Phase: 14-google-drive-source-built-out-of-repo*
*Imported and triaged: 2026-08-18*
