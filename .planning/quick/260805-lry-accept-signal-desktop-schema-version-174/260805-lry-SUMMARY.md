---
phase: quick-260805-lry
plan: 01
subsystem: signal-plugin
tags: [sqlite, sqlcipher, schema-guard, signal-desktop, read-only]

requires:
  - phase: 04-02
    provides: guardSchemaVersion, highestSupportedSchemaVersion (originally pinned to 1730), TestSchemaVersionCeiling negative control
provides:
  - "plugins/signal/schema_readset.go: committed declaration of every table/column the plugin's SQL depends on, including WHERE/ORDER-BY-only columns"
  - "plugins/signal/live_schema_test.go: opt-in TestLiveSchemaReadSet, verifies the read set + functional read path against the real live database"
  - "highestSupportedSchemaVersion raised 1730 -> 1740 with dual-pin provenance and a corrected guarantee statement"
  - "Precedent record (this SUMMARY) for the next schema bump to diff against"
affects: [signal-plugin, schema-verification-tooling-todo]

actuals:
  tokens: 2700
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Opt-in live-database verification test (WEBSPACES_SIGNAL_LIVE_SCHEMA=1) that skips loudly by default, mirroring the existing WEBSPACES_SIGNAL_LIVE_IT pattern in byte_identical_test.go"
    - "Declared read-set map (schema_readset.go) as a non-test file so future tooling can reuse it without extracting it from a test binary"

key-files:
  created:
    - plugins/signal/schema_readset.go
    - plugins/signal/live_schema_test.go
  modified:
    - plugins/signal/schemaguard.go

key-decisions:
  - "highestSupportedSchemaVersion raised to 1740, the exact integer TestLiveSchemaReadSet's live PRAGMA user_version read returned — not assumed, not quoted from the failure message"
  - "Doc comment corrected: the constant tracks the newest schema STATE verified, not the newest Signal Desktop release supported — confirmed by the installed package staying at 8.21.0-1 across both the 1730 and 1740 verifications"
  - "guardSchemaVersion's body and TestSchemaVersionCeiling left untouched; the fixture test is relative to the constant and re-proves the guard at 1740 automatically"
  - "Task 3 (the make dev recovery + live digest render check) is a blocking human-verify checkpoint, deliberately not executed by this run per explicit constraint"

patterns-established:
  - "Live database verification tests are opt-in via env var, chain readSignalConfig -> resolveKey -> openReadOnly directly (never openGuarded when the guard itself is under test), and log aggregate counts only"

requirements-completed: [SRC-02]

coverage:
  - id: D1
    description: "Read set (tables + columns, including WHERE/ORDER-BY-only columns) declared in a committed, non-test file"
    requirement: SRC-02
    verification:
      - kind: unit
        ref: "plugins/signal/live_schema_test.go#TestLiveSchemaReadSet (column-presence assertions)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Opt-in live test verifies the read set and functionally exercises readOwnAci/readConversations/readMessages (covering readAttachments/readReactions) against the real database"
    requirement: SRC-02
    verification:
      - kind: integration
        ref: "plugins/signal/live_schema_test.go#TestLiveSchemaReadSet (opt-in run, WEBSPACES_SIGNAL_LIVE_SCHEMA=1)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Default test run (no live Signal install assumed) still passes — live check skips loudly"
    verification:
      - kind: unit
        ref: "plugins/signal/live_schema_test.go#TestLiveSchemaReadSet (default run, no env var set)"
        status: pass
    human_judgment: false
  - id: D4
    description: "highestSupportedSchemaVersion raised to the observed live value (1740), with corrected provenance doc comment; guard behavior and negative control unchanged"
    requirement: SRC-02
    verification:
      - kind: unit
        ref: "plugins/signal/schema_version_fixture_test.go#TestSchemaVersionCeiling"
        status: pass
      - kind: integration
        ref: "cd plugins/signal && CGO_ENABLED=1 go build -tags libsqlcipher ./... && CGO_ENABLED=1 go test -tags libsqlcipher ./..."
        status: pass
    human_judgment: false
  - id: D5
    description: "Plugin binary rebuilt from the bumped source and the Signal source recovers to a green, correctly-rendering state in the running app"
    verification: []
    human_judgment: true
    rationale: "Requires restarting make dev (a still-running kernel holds the old binary), triggering a live sync in the UI, and visually confirming digest transcripts render correctly with senders/attachments/reactions intact — genuine UAT judgment a column-level check cannot make. This is Task 3, a blocking human-verify checkpoint left open by this run per explicit instruction."

duration: ~20min (Tasks 1-2; Task 3 resolved by user recovery check)
completed: 2026-08-05
status: complete
---

# Quick Task 260805-lry: Accept Signal Desktop Schema Version 1740 Summary

**Raised the Signal plugin's schema-version ceiling from 1730 to 1740 after proving, against the real live database, that every table/column the plugin's SQL depends on is intact and every one of the plugin's own read functions still returns rows — checkpoint for the live `make dev` recovery/render confirmation left open.**

## Performance

- **Duration:** ~20 min (automated Tasks 1-2 only)
- **Completed (automated portion):** 2026-08-05
- **Tasks:** 2 of 3 (Task 3 is a blocking human-verify checkpoint, not run)
- **Files modified:** 3 (2 created, 1 modified)

## Accomplishments

- Declared the plugin's exact SQL read set (`plugins/signal/schema_readset.go`), including three WHERE/ORDER-BY-only columns (`message_attachments.conversationId/editHistoryIndex/orderInMessage`, `reactions.conversationId/timestamp`) that a SELECT-only read set would have silently missed.
- Built an opt-in live verification test (`plugins/signal/live_schema_test.go`, `TestLiveSchemaReadSet`) that opens the real `~/.config/Signal/sql/db.sqlite` strictly read-only via `openReadOnly` (deliberately bypassing `openGuarded`/`guardSchemaVersion`), asserts every declared column present, captures each table's live CREATE statement, and functionally exercises `readOwnAci` → `readConversations` → `buildSenderNames` → `readMessages` (which internally covers `readAttachments`/`readReactions`) against real rows.
- Ran the live check: **clean result** — every read-set column present, every read function returned non-zero rows.
- Raised `highestSupportedSchemaVersion` from 1730 to 1740 (the exact observed value), rewrote its doc comment to carry provenance for both pins and to correct the guarantee it claims.
- Full `plugins/signal` module test suite green under `CGO_ENABLED=1 -tags libsqlcipher`; `bin/plugins/webspaces-plugin-signal` rebuilt via `make signal`.

## Task Commits

Each task was committed atomically:

1. **Task 1: prove the read set is intact on the real database at the new schema version** — `cc11a7e` (feat)
2. **Task 2: raise the ceiling to the verified version, with its provenance, and rebuild** — `9f000c3` (feat)

**Task 3 (checkpoint:human-verify, gate="blocking") was NOT executed** — per explicit run instruction, this executor stopped after rebuilding the plugin binary and did not run `make dev` or perform the live UI recovery check.

## Files Created/Modified

- `plugins/signal/schema_readset.go` — new. Declares `readSetColumns`, the committed table→column map every future schema check diffs against.
- `plugins/signal/live_schema_test.go` — new. `TestLiveSchemaReadSet`, opt-in via `WEBSPACES_SIGNAL_LIVE_SCHEMA=1`.
- `plugins/signal/schemaguard.go` — modified. `highestSupportedSchemaVersion` 1730 → 1740; doc comment rewritten with dual-pin provenance and corrected guarantee.

## Schema Acceptance Record

This is the precedent L-7 requires — the next bump should be a comparison against this record, not a fresh investigation.

**Observed `PRAGMA user_version`:** `1740`

**Installed Signal Desktop package at verification time:** `signal-desktop 8.21.0-1` (build date 2026-07-30, install date 2026-08-03) — **unchanged** from the version the original 1730 pin (2026-08-03) was verified against. No Signal Desktop upgrade occurred between the two verifications; the schema version advanced within a single installed app release. This confirms the anomaly the plan flagged: the ceiling tracks the newest schema state verified, not "the newest Signal Desktop release this plugin supports" — those are different guarantees, and the doc comment now states the corrected one.

**Per-table verdict (all PASS — every declared column present):**

| Table | Required columns | Verdict |
|-------|-------------------|---------|
| `conversations` | id, type, name, profileName, profileFamilyName, e164, serviceId, json | PASS — all present |
| `items` | id, json | PASS — all present |
| `messages` | id, conversationId, sent_at, type, sourceServiceId, body, isErased, json | PASS — all present |
| `message_attachments` | messageId, fileName, contentType, attachmentType, conversationId, editHistoryIndex, orderInMessage | PASS — all present |
| `reactions` | messageId, emoji, fromId, conversationId, timestamp | PASS — all present |

**Captured CREATE statements (verbatim, schema only — no row content):**

`conversations`:
```sql
CREATE TABLE conversations(
      id STRING PRIMARY KEY ASC,
      json TEXT,

      active_at INTEGER,
      type STRING,
      members TEXT,
      name TEXT,
      profileName TEXT
    , profileFamilyName TEXT, profileFullName TEXT, e164 TEXT, serviceId TEXT, groupId TEXT, profileLastFetchedAt INTEGER, expireTimerVersion INTEGER NOT NULL DEFAULT 1)
```

`items`:
```sql
CREATE TABLE items(
      id STRING PRIMARY KEY ASC,
      json TEXT
    )
```

`message_attachments` (STRICT table; note the schema now carries dozens of additional columns beyond the plugin's read set — all extras, no read-set column removed):
```sql
CREATE TABLE message_attachments (
        messageId TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
        editHistoryIndex INTEGER NOT NULL,
        attachmentType TEXT NOT NULL, -- 'long-message' | 'quote' | 'attachment' | 'preview' | 'contact' | 'sticker'
        orderInMessage INTEGER NOT NULL,
        conversationId TEXT NOT NULL,
        sentAt INTEGER NOT NULL,
        clientUuid TEXT,
        size INTEGER NOT NULL,
        contentType TEXT NOT NULL,
        path TEXT,
        plaintextHash TEXT,
        localKey TEXT,
        caption TEXT,
        fileName TEXT,
        blurHash TEXT,
        height INTEGER,
        width INTEGER,
        digest TEXT,
        key TEXT,
        downloadPath TEXT,
        version INTEGER,
        incrementalMac TEXT,
        incrementalMacChunkSize INTEGER,
        transitCdnKey TEXT,
        transitCdnNumber INTEGER,
        transitCdnUploadTimestamp INTEGER,
        backupCdnNumber INTEGER,
        thumbnailPath TEXT,
        thumbnailSize INTEGER,
        thumbnailContentType TEXT,
        thumbnailLocalKey TEXT,
        thumbnailVersion INTEGER,
        screenshotPath TEXT,
        screenshotSize INTEGER,
        screenshotContentType TEXT,
        screenshotLocalKey TEXT,
        screenshotVersion INTEGER,
        backupThumbnailPath TEXT,
        backupThumbnailSize INTEGER,
        backupThumbnailContentType TEXT,
        backupThumbnailLocalKey TEXT,
        backupThumbnailVersion INTEGER,
        storyTextAttachmentJson TEXT,
        localBackupPath TEXT,
        flags INTEGER,
        error INTEGER,
        wasTooBig INTEGER,
        isCorrupted INTEGER,
        copiedFromQuotedAttachment INTEGER,
        pending INTEGER,
        backfillError INTEGER, messageType TEXT, receivedAt INTEGER, receivedAtMs INTEGER, isViewOnce INTEGER, duration REAL,
        PRIMARY KEY (messageId, editHistoryIndex, attachmentType, orderInMessage)
      ) STRICT
```

`messages` (also carries many generated/virtual columns beyond the read set — none of the plugin's required columns were removed or renamed):
```sql
CREATE TABLE messages(
        rowid INTEGER PRIMARY KEY ASC,
        id STRING UNIQUE,
        json TEXT,
        readStatus INTEGER,
        expires_at INTEGER,
        sent_at INTEGER,
        schemaVersion INTEGER,
        conversationId STRING,
        received_at INTEGER,
        hasAttachments INTEGER,
        hasFileAttachments INTEGER,
        hasVisualMediaAttachments INTEGER,
        expireTimer INTEGER,
        expirationStartTimestamp INTEGER,
        type STRING,
        body TEXT,
        messageTimer INTEGER,
        messageTimerStart INTEGER,
        messageTimerExpiresAt INTEGER,
        isErased INTEGER,
        isViewOnce INTEGER,
        sourceServiceId TEXT, serverGuid STRING NULL, sourceDevice INTEGER, storyId STRING, isStory INTEGER
        GENERATED ALWAYS AS (type IS 'story'), isChangeCreatedByUs INTEGER NOT NULL DEFAULT 0, isTimerChangeFromSync INTEGER
        GENERATED ALWAYS AS (json_extract(json, '$.expirationTimerUpdate.fromSync') IS 1),
        seenStatus NUMBER default 0, storyDistributionListId STRING,
        expiresAt INT GENERATED ALWAYS AS (ifnull(expirationStartTimestamp + (expireTimer * 1000), 9007199254740991)),
        isUserInitiatedMessage INTEGER GENERATED ALWAYS AS (
          type IS NULL OR type NOT IN (
            'change-number-notification','contact-removed-notification','conversation-merge',
            'group-v1-migration','group-v2-change','keychange','message-history-unsynced',
            'profile-change','story','universal-timer-notification','verified-change'
          )
        ),
        mentionsMe INTEGER NOT NULL DEFAULT 0,
        isGroupLeaveEvent INTEGER GENERATED ALWAYS AS (...),
        isGroupLeaveEventFromOther INTEGER GENERATED ALWAYS AS (...),
        callId TEXT GENERATED ALWAYS AS (json_extract(json, '$.callId')),
        shouldAffectPreview INTEGER GENERATED ALWAYS AS (...),
        shouldAffectActivity INTEGER GENERATED ALWAYS AS (...),
        isAddressableMessage INTEGER GENERATED ALWAYS AS (type IS NULL OR type IN ('incoming','outgoing')),
        timestamp INTEGER, received_at_ms INTEGER, unidentifiedDeliveryReceived INTEGER, serverTimestamp INTEGER, source TEXT,
        isSearchable INT GENERATED ALWAYS AS (isViewOnce IS NOT 1 AND storyId IS NULL) VIRTUAL,
        searchableText TEXT GENERATED ALWAYS AS (CASE WHEN json->'poll' IS NOT NULL THEN json->'poll'->>'question' ELSE body END) VIRTUAL,
        hasUnreadPollVotes INTEGER NOT NULL DEFAULT 0,
        hasExpireTimer INTEGER NOT NULL GENERATED ALWAYS AS (COALESCE(expireTimer, 0) > 0) VIRTUAL,
        hasPreviews INTEGER NOT NULL GENERATED ALWAYS AS (IFNULL(json_array_length(json, '$.preview'), 0) > 0),
        hasContacts INTEGER NOT NULL GENERATED ALWAYS AS (IFNULL(json_array_length(json, '$.contact'), 0) > 0))
```
(Full unabbreviated CREATE statement is in this task's live test output; the abbreviation above collapses several structurally-identical GENERATED ALWAYS boolean expressions for readability — no column was omitted from the verdict table above, which was verified against the complete, unabbreviated text.)

`reactions`:
```sql
CREATE TABLE reactions(
        conversationId STRING,
        emoji STRING,
        fromId STRING,
        messageReceivedAt INTEGER,
        targetAuthorAci STRING,
        targetTimestamp INTEGER,
        unread INTEGER
      , messageId STRING, timestamp NUMBER)
```

**Functional read-path verdict:** `readOwnAci` returned a non-empty identifier; `readConversations` returned 210 conversations; a bounded 5-conversation probe through `buildSenderNames` → `readMessages` returned 279 message records, 29 with at least one attachment, 22 with at least one reaction — all read functions ran without error. No message body, contact name, phone number, service identifier, attachment filename, or key material was logged at any point (L-2).

## Decisions Made

- `highestSupportedSchemaVersion` raised to 1740 — the exact integer the live `PRAGMA user_version` read returned, not an assumed or quoted value.
- The doc comment's guarantee corrected: the constant tracks "the newest schema state this plugin has been verified against," not "the newest Signal Desktop release this plugin supports" — the installed package version staying at `8.21.0-1` across both verifications is direct evidence these are different things.
- `guardSchemaVersion`'s body, its error message, and `TestSchemaVersionCeiling`'s three relative test cases were left untouched, per L-4 and the plan's own key_link — the fixture test needed no edit and automatically re-proves the guard at 1740.
- The pending verify-and-accept tooling todo (`2026-08-05-signal-schema-version-verify-and-accept-tooling.md`) remains open. This plan's `schema_readset.go` declaration and `live_schema_test.go` opt-in test are the minimal foundation that todo can build on (a subcommand, auto-bumper, or committed schema-snapshot fixture were explicitly out of scope, L-8).

## Deviations from Plan

None — plan executed exactly as written for Tasks 1 and 2. Task 3 was deliberately not executed, per the explicit run instruction that this executor complete the automated tasks and rebuild, then stop at the checkpoint without attempting the `make dev` recovery check itself. This is not a deviation from the plan (the plan itself defines Task 3 as `autonomous: false`, `type="checkpoint:human-verify"`, `gate="blocking"`) — it is the plan's own designed stopping point.

## Issues Encountered

None. The live database opened cleanly on the first attempt (plaintext `key` field in `config.json`, no keyring/D-Bus resolution needed on this machine, consistent with the environment findings recorded in the plan). No column was missing, no table was gone, and no read function errored — the "dirty diff halts the task" path (L-5) was not exercised because the result was clean.

## User Setup Required

None — no external service configuration required. Task 3 requires the user to restart `make dev` and interact with the running web UI, which is a verification action, not a setup action.

## Task 3 Resolution (blocking human-verify — APPROVED 2026-08-05)

**Checkpoint resolved:** the user restarted `make dev`, confirmed the Signal source syncs green (no "unrecognised database schema version" error) and digests render correctly, and replied "approved". The Signal source is recovered. The original checkpoint steps are preserved below for reference:

1. Stop any running kernel, then run `make dev` fresh so the rebuilt `bin/plugins/webspaces-plugin-signal` (rebuilt in Task 2, confirmed newer than `schemaguard.go`) is the binary actually loaded.
2. Open the web UI, go to a webspace with the Signal source configured, trigger a refresh, and confirm the health indicator resolves to green — the "unrecognised database schema version" failure text should be gone.
3. Open at least one Signal item in the detail pane and confirm the transcript renders correctly: messages in order, sender names resolved, attachments/reactions present where expected.
4. Confirm the other sources are unaffected.

If the sync is green but a digest renders empty, has unresolved senders, or is missing attachments/reactions known to exist, that is a real schema change the column-level check could not detect — the ceiling should be reverted, not kept, and this should be reported rather than approved.

**Resume signal:** Type "approved" if the Signal source syncs green AND digests render correctly, or describe exactly what is wrong or missing.

---
*Phase: quick-260805-lry*
*Completed: 2026-08-05 — all 3 tasks (Task 3 recovery check approved by user)*
*Status: complete*

## Self-Check: PASSED

All created/modified files confirmed present on disk; both task commits (`cc11a7e`, `9f000c3`) confirmed in git history.
