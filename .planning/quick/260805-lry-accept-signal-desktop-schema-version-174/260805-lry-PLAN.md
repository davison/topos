---
phase: quick-260805-lry
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - plugins/signal/schema_readset.go
  - plugins/signal/live_schema_test.go
  - plugins/signal/schemaguard.go
autonomous: false
requirements: [SRC-02]
user_setup: []

estimate:
  tokens: 45000
  raw_tokens: 45000
  tasks: 3
  confidence: low

must_haves:
  truths:
    - "The pinned ceiling is raised ONLY after the plugin's exact read set was re-read off the real, live db.sqlite at the new schema version — every table and every column the shipped SQL names, including the columns that appear only in WHERE and ORDER BY clauses and never in a SELECT list. (SRC-02)"
    - "The verification exercises the plugin's OWN row-reading functions against the live database, not merely a column-presence check. A column that survived a rename in name but changed meaning, type or encoding is caught because readConversations/readOwnAci/readAttachments/readReactions/readMessages actually run and actually return rows. (L-3)"
    - "The live verification opens the database strictly read-only and therefore cannot itself be the thing that changed it. It reuses the plugin's own already-proven openReadOnly (mode=ro) rather than opening a second, differently-configured connection. (L-1)"
    - "The verification deliberately bypasses guardSchemaVersion. The guard is precisely what is refusing to run at the new version, so a verification routed through openGuarded would be unable to gather the evidence needed to raise it. (L-4)"
    - "A dirty result blocks the bump. If any read-set column is absent, or any of the plugin's own read functions errors, the ceiling is not touched and the diff is surfaced for a decision instead. (L-5)"
    - "The default test run — no live Signal install, no live database, any machine — still passes. The live check is opt-in via an environment variable and skips loudly rather than failing when the database is absent. (L-6)"
    - "The negative control survives the bump: a fixture one above the new ceiling still fails loudly naming both the version found and the ceiling, and a fixture at the new ceiling still passes. (L-4)"
    - "The evidence is written down. The observed user_version, the captured CREATE statements for exactly the read-set tables, the installed Signal Desktop version, and the pass/fail of each read-set assertion are recorded in the SUMMARY so the NEXT bump is a comparison rather than a fresh investigation. (L-7)"
    - statement: "Whether the raised ceiling is the HIGHEST schema version this Signal Desktop release will ever migrate to cannot be established from inside this repository. The ceiling tracks the version observed on disk, not the maximum migration the app ships — the two are not the same number, and this task's own evidence shows they can diverge (see the anomaly register). Only a future failing sync can distinguish them."
      verification: backstop
  prohibitions:
    - statement: "MUST NOT bump the ceiling on a dirty or partial verification result. A missing column, a renamed column, or an error from any of the plugin's own read functions HALTS the task — the loud failure is the correct behavior and staying broken is strictly better than importing against a schema nobody checked."
      status: unverified
      verification: test
    - statement: "MUST NOT bump the ceiling to a number that was assumed, quoted from the error message, or carried over from this plan's prose. The value written into the constant is the integer the live PRAGMA read returned in Task 1, whatever it turned out to be."
      status: unverified
      verification: test
    - statement: "MUST NOT open the live database in any mode other than read-only, and MUST NOT issue VACUUM, PRAGMA wal_checkpoint, PRAGMA journal_mode, PRAGMA optimize, or any other statement that can write to the file, its WAL, or its shm. Signal Desktop is a live concurrent writer."
      status: unverified
      verification: test
    - statement: "MUST NOT log, print, write to a test artifact, or paste into the SUMMARY any message body, quoted text, attachment filename, contact display name, phone number, service identifier, or the SQLCipher key. Counts, column names, table names and CREATE statements only."
      status: unverified
      verification: test
    - statement: "MUST NOT weaken, soften, or make conditional guardSchemaVersion's fail-loud behavior. This task raises one integer; it does not turn the guard into a warning, a log line, or an auto-accepting check."
      status: unverified
      verification: test
    - statement: "MUST NOT build the durable verify-and-accept tooling the pending todo describes. The declared read set plus the opt-in live test is the minimal foundation; a subcommand, a committed schema-snapshot fixture, an auto-bumper or a diff formatter are all out of scope and the todo stays open."
      status: unverified
      verification: manual
  artifacts:
    - path: "plugins/signal/schema_readset.go"
      provides: "The single declared, committed statement of exactly which tables and columns this plugin's SQL depends on — the expectation any future schema check diffs against. Non-test file specifically so a future tooling pass can reuse it without lifting it out of a _test.go."
      contains: "readSetColumns"
    - path: "plugins/signal/live_schema_test.go"
      provides: "The opt-in live verification: skips by default, and when enabled resolves the key, opens the real database read-only WITHOUT the version guard, reads PRAGMA user_version, asserts every declared read-set column is present via PRAGMA table_info, captures each read-set table's CREATE statement, and then functionally exercises the plugin's own read functions against live rows."
      contains: "TestLiveSchemaReadSet"
    - path: "plugins/signal/schemaguard.go"
      provides: "The raised highestSupportedSchemaVersion constant plus a doc comment recording WHEN, against WHICH Signal Desktop version, and by WHICH mechanism this raise was verified — extending, never replacing, the provenance the 1730 pin already carries."
      contains: "highestSupportedSchemaVersion"
  key_links:
    - "live_schema_test.go -> readSignalConfig + resolveKey + openReadOnly, deliberately NOT openGuarded. openGuarded calls guardSchemaVersion, which is the thing currently failing at the new version — routing the verification through it would make the evidence-gathering impossible by construction. This is the single most important implementation detail in the plan and the one most likely to be got wrong."
    - "readSetColumns MUST carry the WHERE-clause and ORDER-BY columns, not just the SELECT lists. message_attachments.conversationId / editHistoryIndex / orderInMessage and reactions.conversationId / timestamp never appear in a SELECT list but the queries break without them. A read set built by reading only the SELECT lines is silently incomplete."
    - "schema_version_fixture_test.go's three cases are written RELATIVE to highestSupportedSchemaVersion (+1 / +0 / -1), never against a hardcoded 1730. Bumping the constant therefore requires NO edit to that file and it re-proves the guard at the new value automatically. Editing it would be a signal something was misunderstood."
    - "The ceiling constant is read by exactly one function (guardSchemaVersion) and pinned by exactly one test file. Changing the integer is a one-line edit; the doc comment above it is the only other thing that must move with it."
---

<objective>
Accept Signal Desktop's new database schema version — after proving, against the real database,
that accepting it is safe.

The Signal plugin is currently refusing to sync with `unrecognised database schema version 1740
(this plugin was built against up to 1730)`. That refusal is correct and designed (Phase 4
success criterion 5, ROADMAP): the plugin reads a database it does not own, whose owner migrates
it without warning, and the guard exists so a schema change surfaces as a loud stop rather than
as silently wrong digests.

Raising the ceiling is therefore not a fix — it is an acceptance decision, and it is only
legitimate with evidence. This plan gathers that evidence first and bumps second, and it makes
the evidence-gathering repeatable so the next Signal update is a comparison instead of an
investigation.

Purpose: restore the Signal source to a working state without ever converting a designed
fail-loud guard into a rubber stamp.

Output: a committed declaration of exactly which tables and columns this plugin depends on; an
opt-in test that verifies that read set against the real, live database read-only; a raised
ceiling carrying its own verification provenance; a rebuilt plugin binary; and a recorded
schema-acceptance precedent in the SUMMARY.
</objective>

<execution_context>
@$HOME/.claude/gsd-core/workflows/execute-plan.md
@$HOME/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/STATE.md

@plugins/signal/schemaguard.go
@plugins/signal/schema_version_fixture_test.go
@.planning/todos/pending/2026-08-05-signal-schema-version-verify-and-accept-tooling.md
</context>

<locked_constraints>

From the Phase 4 hard rules and the task brief. NOT open for reconsideration during execution.
Task actions cite the ones they implement.

| ID | Constraint |
|----|------------|
| L-1 | The live database is opened strictly read-only (`mode=ro`). No VACUUM, no checkpoint, no journal-mode change, nothing that can write to the file, its WAL or its shm. Signal Desktop is a live concurrent writer. |
| L-2 | Read-only extends to privacy: the verification reads schema and counts. No message body, contact name, phone number, service id, attachment filename or key material is logged, printed, or recorded anywhere. |
| L-3 | The evidence must come from the REAL database at the new version. Not a fixture, not the previous introspection, not inference from a Signal Desktop changelog, not an assumption that a minor version bump is cosmetic. |
| L-4 | `guardSchemaVersion`'s fail-loud behavior is untouched. This task changes one integer and the comment above it. The guard does not become a warning and never auto-accepts. |
| L-5 | A dirty diff STOPS the task. No bump, no rebuild — surface the difference and stop for a decision. |
| L-6 | The repository's default test run must still pass on a machine with no Signal install. The live check is opt-in and skips, never fails, when the database is absent. |
| L-7 | The result is written down as a precedent: observed version, captured schema, installed Signal Desktop version, and per-table verdict, in the SUMMARY. |
| L-8 | Minimal foundation only for the pending tooling todo. The declared read set and the opt-in test are in scope; a subcommand, an auto-bumper, or a committed schema snapshot are not. The todo stays open. |

</locked_constraints>

<environment_findings>

Three things were established by direct probing during planning. They correct assumptions in the
task brief and materially change what this plan can automate — **do not re-derive them from the
brief, and do not route around them.**

**1. The SQLite floor is NOT a problem in this environment.** The brief warned that
`go test ./plugins/signal/...` partially fails here because the system libsqlcipher embeds SQLite
3.39.4, below the plugin's 3.51.3 floor. That is stale. Measured on this machine during planning:

- `pkg-config --modversion sqlcipher` → `3.51.3`
- `/usr/bin/sqlcipher --version` → `3.51.3 ... (SQLCipher 4.14.0 community)`
- `cd plugins/signal && CGO_ENABLED=1 go test -tags libsqlcipher ./...` → **`ok ... 2.983s`**, whole suite green

The full `make test-signal` gate therefore CAN and MUST run in this task. There is no need for a
reduced test subset, a purpose-built external helper binary, or a human-run introspection step.
If a test does fail during execution, it is a real failure caused by this task — treat it as
such, do not attribute it to the environment.

**2. Key resolution needs no keyring and no D-Bus.** `~/.config/Signal/config.json` on this
machine carries a plaintext `key` field and has no `encryptedKey` and no `safeStorageBackend`
(this install was never safeStorage-migrated — STATE.md already records this). `resolveKey`'s
`hasKey && !hasEncrypted` branch returns that value directly. The live verification is therefore
fully automatable in-process; it does not need a Secret Service session and must not be designed
around one.

**3. The plugin's read set lives entirely in `plugins/signal/plugin.go`.** A repo-wide search for
SQL across the plugin's non-test sources returns only `plugin.go`'s five queries plus `dsn.go`'s
`SELECT sqlite_version()`. `match.go` and `digest.go` operate purely on in-memory structs and
issue no SQL of their own — the brief's suggestion that they might is not borne out. The
enumeration in the next block is complete.

</environment_findings>

<read_set>

The exact tables and columns the shipped SQL depends on, enumerated from `plugins/signal/plugin.go`
during planning. **Columns that appear only in a WHERE or ORDER BY clause are part of the read set
and are marked** — a read set built from the SELECT lists alone is silently incomplete and would
let a breaking change through.

| Table | Columns in SELECT | Columns only in WHERE / ORDER BY | JSON blob fields consumed |
|-------|-------------------|----------------------------------|---------------------------|
| `conversations` | `id`, `type`, `name`, `profileName`, `profileFamilyName`, `e164`, `serviceId`, `json` | — (`type` filters `IN ('private','group')`, already in SELECT) | `systemGivenName`, `systemFamilyName`, `nicknameGivenName`, `nicknameFamilyName` |
| `items` | `id`, `json` | — (`id = 'uuid_id'`, already in SELECT) | `value` |
| `messages` | `id`, `conversationId`, `sent_at`, `type`, `sourceServiceId`, `body`, `isErased`, `json` | — (`type`, `sent_at` filter and order, both already in SELECT) | `deletedForEveryone`, `editHistory`, `quote.text` |
| `message_attachments` | `messageId`, `fileName`, `contentType`, `attachmentType` | **`conversationId`**, **`editHistoryIndex`**, **`orderInMessage`** | — |
| `reactions` | `messageId`, `emoji`, `fromId` | **`conversationId`**, **`timestamp`** | — |

Value-level dependencies the SQL also encodes, worth confirming still yield rows but NOT worth a
separate assertion each: `conversations.type IN ('private','group')`, `messages.type IN
('incoming','outgoing')`, `message_attachments.editHistoryIndex = -1`, and the
`realAttachmentTypes` value set (`attachment`, `sticker`, `contact`) — see `plugin.go`'s own
doc comment for why `preview`, `quote` and `long-message` are excluded.

</read_set>

<anomaly_register>

One thing does not add up, and Task 1 must record what it observes rather than explain it away.

The ceiling was pinned to 1730 on 2026-08-03 against Signal Desktop 8.21.0 (Arch package
`signal-desktop 8.21.0-1`). Probed during planning today, the installed package is **still
`8.21.0-1`, installed 2026-08-03, built 2026-07-30** — no Signal Desktop upgrade has happened on
this machine since the pin. Yet the database now reports a higher schema version.

The most likely reading is that the schema advanced *within* a single app version — Signal Desktop
migrates its database on launch, and the version observed on 2026-08-03 was the state at that
moment, not the maximum migration that release ships. If so, the ceiling has never tracked "the
newest Signal Desktop this plugin supports"; it tracks "the newest schema state this plugin has
looked at". Those are different guarantees and the doc comment currently implies the stronger one.

Task 1 records the observed facts (installed package version, observed `user_version`). Task 2
corrects the doc comment to describe the weaker, true guarantee. **Neither task speculates in the
source about the cause** — the observation is the deliverable, and it is exactly the precedent
that makes the next bump cheap.

</anomaly_register>

<tasks>

<task type="tracer">
  <name>Task 1: prove the read set is intact on the real database at the new schema version</name>

  <files>plugins/signal/schema_readset.go, plugins/signal/live_schema_test.go</files>

  <read_first>
    - `plugins/signal/plugin.go` lines 355-460 — `conversationFields`, `readConversations`,
      `readOwnAci`. Confirm every column named in this plan's read-set table against the actual
      SELECT text before declaring it.
    - `plugins/signal/plugin.go` lines 471-605 — `realAttachmentTypes`, `readAttachments`,
      `readReactions`, including their WHERE and ORDER BY clauses.
    - `plugins/signal/plugin.go` lines 606-700 — `readMessages`, and note it takes an explicit
      `conversationIDs` slice, which is what makes a bounded live read possible.
    - `plugins/signal/plugin.go` lines 83-130 — `openGuarded`. Read it specifically to see why
      this task must NOT use it.
    - `plugins/signal/dsn.go` lines 1-90 — `openReadOnly` and `buildReadOnlyDSN`. Reuse, never
      reimplement, and never construct a second DSN.
    - `plugins/signal/keyresolve.go` lines 54-90 — `readSignalConfig` and `resolveKey`.
    - `plugins/signal/byte_identical_test.go` — `buildFixtureDatabase` and `fixtureKeyHex`, for
      the package's existing conventions around opening a database inside a test.
  </read_first>

  <action>
    Create `plugins/signal/schema_readset.go` (package `main`, non-test) declaring one
    package-level map from table name to the required column names, populated EXACTLY from this
    plan's read-set table including the WHERE-only and ORDER-BY-only columns. Give it a doc
    comment stating that it is the committed expectation a schema check diffs against, that it
    must be updated in the same commit as any change to the plugin's SQL, and that columns
    appearing only in filter or ordering clauses belong in it. Non-test file deliberately, per
    L-8, so a later tooling pass can reuse it without extracting it from a test binary.

    Create `plugins/signal/live_schema_test.go` (package `main`) with one test function that:

    Skips immediately, with an explanatory skip message naming the environment variable, unless
    an opt-in environment variable is set. This satisfies L-6: the default run on any machine
    stays green. Also skip if the database file is absent.

    Resolves the key and opens the database by calling `readSignalConfig` on the real config
    path, then `resolveKey`, then `openReadOnly` on the real database path — chained directly,
    NOT through `openGuarded`. This is the load-bearing detail: `openGuarded` calls
    `guardSchemaVersion`, which is exactly what is refusing to run, so a verification routed
    through it cannot gather the evidence needed to raise the ceiling (L-4, and the first
    key_link). Resolve both paths from the user's home directory rather than hardcoding an
    absolute literal.

    Reads `PRAGMA user_version` off that connection and logs the integer it returns. Do NOT
    assert it equals any particular value and do NOT hardcode an expected version here — the
    observed integer is an output of this task, consumed by Task 2 (L-3, and the second
    prohibition). Log it in a form that is easy to copy into the SUMMARY.

    For each table in the read-set map, in a stable sorted order: run `PRAGMA table_info` for
    that table, collect the returned column names into a set, and report a distinct failure
    naming the table and the column for every required column that is absent. Report a failure
    naming the table if `table_info` returns nothing at all, since that means the table itself is
    gone. Then read that table's own `sql` text from `sqlite_master` and log it verbatim — this
    is the captured CREATE statement the SUMMARY records as the precedent (L-7).

    Then functionally exercise the plugin's own read path against live rows, which is what makes
    this a tracer rather than a column-presence check. Call `readOwnAci` and require it to return
    without error and to yield a non-empty identifier — this is the whole `items` / `uuid_id` /
    `value` JSON path proven end to end. Call `readConversations` with that identifier and
    require no error and a non-zero row count. Take a SMALL bounded slice of the returned
    conversation ids — no more than a handful — and pass it to `buildSenderNames` and then to
    `readMessages`, requiring no error. `readMessages` internally calls `readAttachments` and
    `readReactions` for those same conversations, so this one bounded call proves all three
    dedicated tables and the message JSON blob parse in a single step. Bounding the slice matters:
    `readMessages` applies no time window by design (D-08, full history), and the real database is
    hundreds of megabytes.

    Log only aggregate counts from that functional step — how many conversations were returned,
    how many message records the bounded read produced, and how many of those carried at least
    one attachment and at least one reaction. Log nothing derived from the CONTENT of any row:
    no bodies, no names, no identifiers, no filenames, no key material (L-2). Prefer a
    conversation slice chosen deterministically (for instance the first few in the returned
    order) over anything that would require inspecting conversation content to select.

    Run the test with the opt-in variable set and read the output carefully. If every required
    column is present and every read function succeeded, this task passes and Task 2 may proceed.
    If ANY column is missing, any table has vanished, or any read function errors, STOP: do not
    proceed to Task 2, do not touch the ceiling constant, and surface the specific table, column
    and error for a decision (L-5).
  </action>

  <verify>
    <automated>cd plugins/signal &amp;&amp; CGO_ENABLED=1 go test -tags libsqlcipher -run TestLiveSchemaReadSet -v ./... 2>&amp;1 | grep -q -- '--- SKIP'</automated>
    <automated>cd plugins/signal &amp;&amp; WEBSPACES_SIGNAL_LIVE_SCHEMA=1 CGO_ENABLED=1 go test -tags libsqlcipher -run TestLiveSchemaReadSet -v ./...</automated>
  </verify>

  <done>
    The first command proves the live check skips by default, so the repository's normal test run
    is unaffected on any machine (L-6). The second command passes against the real database, and
    its verbose output contains: the observed `PRAGMA user_version` integer, one captured CREATE
    statement per read-set table, and non-zero counts from the functional read. Every column in
    the declared read set was found present. No message content of any kind appears in the output.
    The observed version integer is carried forward to Task 2.
  </done>
</task>

<task type="auto">
  <name>Task 2: raise the ceiling to the verified version, with its provenance, and rebuild</name>

  <precondition>Task 1's live verification passed: every declared read-set column was present on the real database and every one of the plugin's own read functions ran without error. If any column was missing or any read errored, this task does not run — halt and surface the diff (L-5).</precondition>

  <files>plugins/signal/schemaguard.go</files>

  <read_first>
    - `plugins/signal/schemaguard.go` — the whole file, especially the existing doc comment's
      provenance paragraph, which is extended rather than discarded.
    - `plugins/signal/schema_version_fixture_test.go` — the whole file. Read it to confirm its
      three cases are expressed relative to the constant (`+1`, exact, `-1`) and therefore need
      NO edit. Editing this file is a signal something was misunderstood.
  </read_first>

  <action>
    Change `highestSupportedSchemaVersion` to the integer Task 1's live `PRAGMA user_version` read
    actually returned. Use the observed value, not a value quoted from the failure message and not
    one carried over from this plan's prose (second prohibition). This is a one-line change to the
    constant.

    Rewrite the constant's doc comment so it carries provenance for BOTH pins rather than
    replacing the first with the second. It should record: the newly observed version, the date it
    was verified, the installed Signal Desktop package version at verification time (capture this
    from the system package manager during the task — planning observed it as unchanged from the
    original pin, and if it is still unchanged that is itself the notable fact), and the mechanism
    of verification, naming the read-set declaration and the opt-in live test by file so the next
    person re-runs the same check instead of inventing one.

    Correct the guarantee the comment claims, per the anomaly register. The current wording ties
    re-verification to "a newer Signal Desktop release", which this task's own evidence
    contradicts: the schema advanced with no package upgrade on this machine. Reword so the
    comment describes what the constant actually tracks — the newest schema state this plugin has
    been verified against — and states that a schema version can advance within a single Signal
    Desktop release, so the trigger for re-verification is the guard firing, not an app upgrade.
    Record the observation; do not speculate in the source about its cause.

    Keep the existing instruction that raising the constant is a deliberate act performed only
    after re-running the introspection, and keep it never-bumped-speculatively. Do not touch
    `guardSchemaVersion`'s body — its fail-loud behavior, its message, and the fact that it names
    both the found version and the ceiling are all unchanged (L-4).

    Do not edit `schema_version_fixture_test.go`. Its cases are relative to the constant, so they
    automatically re-prove the guard at the new value.

    Then run the gates and rebuild. Per the environment findings, the full Signal module test
    suite passes in this environment and is the correct gate — do not substitute a reduced subset.
    Rebuild the plugin binary with the repository's own Makefile target rather than a hand-written
    `go build` invocation, so the cgo and build-tag flags stay in one place.
  </action>

  <verify>
    <automated>cd plugins/signal &amp;&amp; CGO_ENABLED=1 go build -tags libsqlcipher ./... &amp;&amp; CGO_ENABLED=1 go test -tags libsqlcipher ./...</automated>
    <automated>cd plugins/signal &amp;&amp; WEBSPACES_SIGNAL_LIVE_SCHEMA=1 CGO_ENABLED=1 go test -tags libsqlcipher -run 'TestLiveSchemaReadSet|TestSchemaVersionCeiling' -v ./...</automated>
    <automated>make signal &amp;&amp; test bin/plugins/webspaces-plugin-signal -nt plugins/signal/schemaguard.go</automated>
    <automated>test -z "$(git diff --name-only -- plugins/signal/schema_version_fixture_test.go)"</automated>
  </verify>

  <done>
    `highestSupportedSchemaVersion` equals the integer Task 1 read off the live database. The whole
    Signal module builds and its full test suite is green, including `TestSchemaVersionCeiling`,
    which re-proves at the new value that one above the ceiling fails loudly naming both numbers
    and that the ceiling itself passes. The live read-set check still passes. The fixture test file
    is unmodified. `bin/plugins/webspaces-plugin-signal` is rebuilt and newer than the source it
    was built from. The doc comment records the new version, the verification date, the installed
    Signal Desktop version, the verification mechanism by filename, and the corrected statement of
    what the constant tracks.
  </done>
</task>

<task type="checkpoint:human-verify" gate="blocking">
  <name>Task 3: the Signal source syncs green again and its digests render correctly</name>
  <what-built>
    The Signal plugin's pinned schema-version ceiling was raised to the version now on disk, after
    an opt-in live test confirmed — against the real database, opened read-only — that every table
    and column the plugin's SQL depends on is still present, and that the plugin's own read
    functions still return rows. The plugin binary at `bin/plugins/webspaces-plugin-signal` has
    been rebuilt with the new ceiling.

    The Signal source has been failing every sync since Signal Desktop migrated the database, so
    this is also the recovery check: it should go from failing to green.
  </what-built>
  <how-to-verify>
    1. Stop any running kernel, then start the stack fresh so the rebuilt plugin binary is the one
       actually loaded — a still-running kernel holds the OLD binary and will keep failing
       identically, which would look like the fix did not work:

       `make dev`

    2. Open the web UI and go to a webspace that has the Signal source configured.

    3. Trigger a refresh of the Signal source and confirm its health indicator resolves to green /
       ok, and that the failure text naming an unrecognised database schema version is gone.

    4. Confirm Signal digests actually RENDER in the stream — not just that the sync reports
       success. Open at least one Signal item in the detail pane and confirm the transcript
       renders: messages in order, sender names resolved, and — if the conversation has any —
       attachments and reactions present. This is the part that would catch a column that survived
       under the same name but changed meaning, which no schema check can see.

    5. Confirm the other sources are unaffected.

    If the sync is green but a digest renders empty, renders with unknown senders throughout, or
    is missing attachments or reactions you know are there, say so rather than approving — that is
    a real schema change the column-level check could not detect, and it means the ceiling should
    be reverted rather than kept.
  </how-to-verify>
  <resume-signal>Type "approved" if the Signal source syncs green AND digests render correctly, or describe exactly what is wrong or missing</resume-signal>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Signal Desktop's database → this plugin | A third-party application's private, encrypted, actively-written store crosses into this project. Its schema and contents are entirely outside this project's control and change without notice. |
| SQLCipher key material → process memory | The raw database key is resolved from `config.json` and held in-process for the duration of a read. |
| Live personal message content → test output / SUMMARY / git | A verification step that reads real conversations can trivially leak their content into a log, a test artifact, or a committed document. |

## STRIDE Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation Plan |
|-----------|----------|-----------|----------|-------------|-----------------|
| T-lry-01 | Tampering | Live `db.sqlite` opened by the verification test | critical | mitigate | The test reuses `openReadOnly` (`mode=ro`) unchanged rather than building its own connection, and issues only `PRAGMA user_version`, `PRAGMA table_info`, a `sqlite_master` read, and the plugin's existing read functions. No VACUUM, checkpoint, journal-mode or optimize statement (L-1). Signal Desktop may be running concurrently. |
| T-lry-02 | Information disclosure | Test output, SUMMARY, git history | high | mitigate | The functional step logs aggregate counts only. No body, contact name, phone number, service id or attachment filename is logged or recorded. Captured schema is CREATE statements and column names, which contain no user data (L-2). |
| T-lry-03 | Information disclosure | SQLCipher raw key in DSN / error paths | high | mitigate | The key is obtained via the existing `resolveKey` and passed straight to the existing `openReadOnly`; it is never logged, never interpolated into a test failure message, and no new code path formats it. `readSignalConfig`'s own contract already forbids logging config contents. |
| T-lry-04 | Spoofing (of safety evidence) | The ceiling raise itself | high | mitigate | The bump is gated on a live read of the real database, not on the failure message, the app changelog, or an assumption that a schema bump is cosmetic. The plan's own second prohibition forbids writing an assumed integer, and Task 2 carries an explicit precondition on Task 1's result (L-3, L-5). |
| T-lry-05 | Elevation of privilege (of a broken schema into accepted state) | `guardSchemaVersion` | high | mitigate | The guard's body is untouched; only the integer moves. `TestSchemaVersionCeiling` is written relative to the constant and re-proves the fail-loud behavior at the new value on every run, so the raise cannot silently defang the guard (L-4). |
| T-lry-06 | Denial of service | Bounded live read of a ~200MB database | low | accept | `readMessages` applies no time window by design, so the read is bounded by passing only a handful of conversation ids. Worst case is a slow opt-in test on the developer's own machine. |
| T-lry-SC | Tampering | npm/pip/cargo installs | n/a | accept | No package-manager install occurs in this task. No dependency is added, removed or upgraded; `plugins/signal/go.mod` is not modified. |
</threat_model>

<verification>
- The live read-set test skips by default and passes when opted in against the real database.
- Every table and column in the declared read set is present at the new schema version.
- `readOwnAci`, `readConversations` and a bounded `readMessages` (which internally covers
  `readAttachments` and `readReactions`) all run against live rows without error and return
  non-zero results.
- The whole Signal module builds and its full test suite is green under
  `CGO_ENABLED=1 -tags libsqlcipher`.
- `TestSchemaVersionCeiling` still proves the guard fails loudly one above the new ceiling and
  passes at it, without that file having been edited.
- `bin/plugins/webspaces-plugin-signal` is rebuilt from the bumped source.
- The user confirms live that the Signal source syncs green and digests render with senders,
  attachments and reactions intact.
</verification>

<success_criteria>
- The ceiling constant equals the version observed on the real database, not an assumed one.
- The evidence that justified the raise is recorded in the SUMMARY: observed `user_version`, the
  captured CREATE statement per read-set table, the installed Signal Desktop version, and a
  per-table verdict — enough that the next bump is a diff against this record rather than a fresh
  investigation.
- The doc comment above the constant states what it actually tracks (the newest verified schema
  state) and that a schema version can advance without a Signal Desktop upgrade.
- The fail-loud guard is behaviorally unchanged.
- The repository's default test run is unaffected on a machine with no Signal install.
- The pending verify-and-accept tooling todo remains open, with the declared read set and the
  opt-in test noted in the SUMMARY as the foundation it can build on.
</success_criteria>

<output>
Create `.planning/quick/260805-lry-accept-signal-desktop-schema-version-174/260805-lry-SUMMARY.md` when done.

The SUMMARY must include a "Schema acceptance record" section carrying the observed
`user_version`, the installed Signal Desktop version, the per-table verdict, and the captured
CREATE statements — this is the precedent L-7 requires and the artifact that makes the next bump
cheap.
</output>
