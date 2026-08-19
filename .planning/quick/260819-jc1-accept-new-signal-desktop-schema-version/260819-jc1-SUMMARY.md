---
phase: quick-260819-jc1
plan: 01
subsystem: signal-plugin
tags: [sqlite, sqlcipher, schema-guard, signal-desktop, read-only, trust-tier]

requires:
  - phase: quick-260805-lry
    provides: schema_readset.go (declared read set), live_schema_test.go (opt-in TestLiveSchemaReadSet), highestSupportedSchemaVersion pinned to 1740 with dual-pin provenance
provides:
  - "highestSupportedSchemaVersion raised 1740 -> 1760 with a third dated provenance entry, extending (not replacing) the existing dual-pin history"
  - "Refreshed Schema Acceptance Record (this SUMMARY) for the next bump to diff against"
  - "Operator's INSTALLED instance recovered: Signal source re-pinned, syncs green, digests render correctly"
affects: [signal-plugin, schema-verification-tooling-todo, installed-instance-deployment]

actuals:
  tokens: 409
  tasks: 3
  commits: 1

tech-stack:
  added: []
  patterns:
    - "Re-ran precedent's unmodified verification tooling (schema_readset.go declared read set + live_schema_test.go opt-in live test) rather than building anything new — this is the pattern the pending verify-and-accept tooling todo should eventually automate"

key-files:
  created: []
  modified:
    - plugins/signal/schemaguard.go

key-decisions:
  - "highestSupportedSchemaVersion raised to 1760 — the exact integer the live PRAGMA user_version read returned, not assumed or quoted from the refusal message"
  - "Third provenance entry appended without disturbing the 1730/1740 entries or the corrected guarantee statement (constant tracks newest schema STATE verified, not newest Signal Desktop release supported)"
  - "This verification crossed a real Signal Desktop package boundary (8.21.0-1 -> 8.22.0-1), unlike the 1730->1740 advance which stayed on the same package — recorded as an observation in the doc comment, not speculated on as a cause"
  - "Rebuilt binary placed into the installed instance via make install-signal (external tier only); operator's config.toml pin was deliberately left untouched by this run — the pin mismatch and re-pin is the human consent gate, resolved in Task 3"
  - "guardSchemaVersion's body, schema_version_fixture_test.go, and both verification tooling files (schema_readset.go, live_schema_test.go) were left unmodified, per L-4/L-8"

patterns-established: []

requirements-completed: [SRC-02]

coverage:
  - id: D1
    description: "highestSupportedSchemaVersion raised to the live-verified 1760, with a third dated provenance entry; guard behavior and negative control (TestSchemaVersionCeiling) unchanged"
    requirement: SRC-02
    verification:
      - kind: unit
        ref: "plugins/signal/schema_version_fixture_test.go#TestSchemaVersionCeiling"
        status: pass
      - kind: integration
        ref: "make test-signal"
        status: pass
      - kind: integration
        ref: "plugins/signal/live_schema_test.go#TestLiveSchemaReadSet (opt-in, WEBSPACES_SIGNAL_LIVE_SCHEMA=1)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Operator's INSTALLED instance (not dev) re-pins the changed external binary, syncs the Signal source green, and Signal digests render correctly (senders resolved, transcript intact, attachments/reactions present)"
    verification:
      - kind: manual_procedural
        ref: "Task 3 checkpoint:human-verify (gate=blocking) — operator restarted the installed kernel, used the chip's re-pin consent flow, confirmed sync green and digests rendering"
        status: pass
    human_judgment: true
    rationale: "Requires restarting the installed kernel, exercising the app's untrusted-add consent-and-pin UI flow, and visually confirming digest transcripts render correctly with senders/attachments/reactions intact — genuine UAT judgment a column-level check cannot make."

duration: ~66min (2min automated Tasks 1-2; remainder awaiting Task 3 operator verification)
completed: 2026-08-19
status: complete
---

# Quick Task 260819-jc1: Accept New Signal Desktop Schema Version (1760) Summary

**Raised the Signal plugin's schema-version ceiling from 1740 to 1760 after re-proving, against the real live database, that every table/column the plugin's SQL depends on is intact — then recovered the operator's INSTALLED instance through the external-tier re-pin consent flow, confirmed live with digests rendering correctly.**

## Performance

- **Duration:** ~66 min total (~2 min automated Tasks 1-2; remainder was the Task 3 blocking checkpoint awaiting operator verification)
- **Started:** 2026-08-19T13:01:34Z
- **Completed:** 2026-08-19T14:07:50Z
- **Tasks:** 3 of 3 (Task 1 verification-only, no files changed; Task 2 code change + rebuild + place; Task 3 human-verify checkpoint, approved)
- **Files modified:** 1 (`plugins/signal/schemaguard.go`)

## Accomplishments

- Re-ran the precedent's unmodified live verification tooling (`schema_readset.go` + `live_schema_test.go`'s `TestLiveSchemaReadSet`) against the real, live `~/.config/Signal/sql/db.sqlite`, opened strictly read-only, deliberately bypassing `openGuarded`/`guardSchemaVersion`.
- Confirmed a clean result: observed `PRAGMA user_version = 1760`, every declared read-set column present across all five tables, and every one of the plugin's own read functions returned rows (210 conversations; a bounded 5-conversation probe returned 279 messages, 29 with attachments, 22 with reactions).
- Raised `highestSupportedSchemaVersion` from 1740 to 1760 (the exact observed value), appending a third dated provenance entry without disturbing the two already there or the corrected guarantee statement.
- Full `plugins/signal` module suite green under `make test-signal`, including `TestSchemaVersionCeiling` re-proving the fail-loud guard at 1760.
- Rebuilt `bin/plugins/topos-plugin-signal` via `make signal` and placed it into the installed instance's external plugin tier via `make install-signal` — now byte-identical (`d6cf7168...`) between the dev tree and `~/.local/share/topos/plugins-external/topos-plugin-signal`.
- Operator restarted the installed kernel, used the Signal source chip's pin-mismatch re-pin affordance to re-accept the changed binary, confirmed the sync resolves green, and confirmed Signal digests render correctly (senders resolved, transcript intact). **Task 3 checkpoint approved.**

## Task Commits

Each code-changing task was committed atomically:

1. **Task 1: re-run the live read-set verification and discover the schema version on disk** — no commit (writes zero repo files, per plan; verification-only, output carried into Task 2)
2. **Task 2: raise the ceiling to the verified version, rebuild, and place into the external tier** — `3acc5fa` (feat)
3. **Task 3: the installed instance re-pins, syncs green, and renders Signal digests correctly** — `checkpoint:human-verify`, gate=`blocking`; no commit (verification-only) — **APPROVED** by operator

## Files Created/Modified

- `plugins/signal/schemaguard.go` — modified. `highestSupportedSchemaVersion` 1740 → 1760; doc comment gained a third dated provenance entry naming the observed version, date, quick-task id, installed Signal Desktop package version, and the verification mechanism by filename.

## Schema Acceptance Record

This is the refreshed precedent L-7 requires — the next bump should diff against this record, not start a fresh investigation. Compare directly against `260805-lry-SUMMARY.md`'s 1740 record, reproduced inline below where it changed.

**Observed `PRAGMA user_version`:** `1760` (previous ceiling: `1740`)

**Installed Signal Desktop package at verification time:** `signal-desktop 8.22.0-1` (build date 2026-08-06, install date 2026-08-12) — **changed** from `8.21.0-1`, which both the 1730 and 1740 verifications ran against. This is the first schema-acceptance run in this project's history to cross a real Signal Desktop release boundary, rather than observing the schema advance within a single installed package. The doc comment's corrected guarantee (the ceiling tracks the newest schema *state* verified, not the newest Signal Desktop *release* supported) is unaffected by this — it is one more data point, not a refutation, since the two properties were already shown to move independently by the 1730→1740 case.

**Per-table verdict (all PASS — every declared column present):**

| Table | Required columns | Verdict |
|-------|-------------------|---------|
| `conversations` | id, type, name, profileName, profileFamilyName, e164, serviceId, json | PASS — all present, no structural change vs. 1740 |
| `items` | id, json | PASS — all present, no structural change vs. 1740 |
| `messages` | id, conversationId, sent_at, type, sourceServiceId, body, isErased, json | PASS — all present, no read-set column added/removed/renamed vs. 1740 |
| `message_attachments` | messageId, fileName, contentType, attachmentType, conversationId, editHistoryIndex, orderInMessage | PASS — all present, no column change vs. 1740 |
| `reactions` | messageId, emoji, fromId, conversationId, timestamp | PASS — all present, no structural change vs. 1740 |

**Diff notes versus the 1740 record (structural changes only — no read-set column was added, removed, or renamed anywhere):**

- `conversations`, `items`, `reactions`: byte-for-byte identical `CREATE` statements to the 1740 record.
- `message_attachments`: identical column list and types; gained two explanatory SQL comments above `editHistoryIndex` describing the `-1` sentinel value used for "root message, not in edit history" — cosmetic, no schema impact.
- `messages`: identical column list and order to the 1740 record. Three `GENERATED ALWAYS` expression bodies changed internally (none are in the plugin's read set, so none affect this plugin regardless): `shouldAffectPreview` and `shouldAffectActivity` gained an additional exclusion for `type IS 'message-request-response-event'` combined with specific `messageRequestResponseEvent` values; `isGroupLeaveEvent`'s expression is now fully spelled out (`type IS 'group-v2-change' AND json_array_length(...) IS 1 AND ...`) where the 1740 record had abbreviated it with "...". No new column, no removed column, no renamed column.

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

`message_attachments` (STRICT table; unchanged column set vs. 1740, two new explanatory comments only):
```sql
CREATE TABLE message_attachments (
        messageId TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
        -- For editHistoryIndex to be part of the primary key, it cannot be NULL in strict tables.
        -- For that reason, we use a value of -1 to indicate that it is the root message (not in editHistory)
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

`messages` (also carries many generated/virtual columns beyond the read set — none of the plugin's required columns were removed or renamed; three GENERATED ALWAYS expression bodies changed internally, none in the read set):
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
        GENERATED ALWAYS AS (
          json_extract(json, '$.expirationTimerUpdate.fromSync') IS 1
        ), seenStatus NUMBER default 0, storyDistributionListId STRING, expiresAt INT
        GENERATED ALWAYS
        AS (ifnull(
          expirationStartTimestamp + (expireTimer * 1000),
          9007199254740991
        )), isUserInitiatedMessage INTEGER
        GENERATED ALWAYS AS (
          type IS NULL
          OR
          type NOT IN (
            'change-number-notification',
            'contact-removed-notification',
            'conversation-merge',
            'group-v1-migration',
            'group-v2-change',
            'keychange',
            'message-history-unsynced',
            'profile-change',
            'story',
            'universal-timer-notification',
            'verified-change'
          )
        ), mentionsMe INTEGER NOT NULL DEFAULT 0, isGroupLeaveEvent INTEGER
        GENERATED ALWAYS AS (
          type IS 'group-v2-change' AND
          json_array_length(json_extract(json, '$.groupV2Change.details')) IS 1 AND
          json_extract(json, '$.groupV2Change.details[0].type') IS 'member-remove' AND
          json_extract(json, '$.groupV2Change.from') IS NOT NULL AND
          json_extract(json, '$.groupV2Change.from') IS json_extract(json, '$.groupV2Change.details[0].aci')
        ), isGroupLeaveEventFromOther INTEGER
        GENERATED ALWAYS AS (
          isGroupLeaveEvent IS 1
          AND
          isChangeCreatedByUs IS 0
        ), callId TEXT
        GENERATED ALWAYS AS (
          json_extract(json, '$.callId')
        ), shouldAffectPreview INTEGER
        GENERATED ALWAYS AS (
      type IS NULL
      OR
      type NOT IN (
        'change-number-notification',
        'contact-removed-notification',
        'conversation-merge',
        'group-v1-migration',
        'keychange',
        'message-history-unsynced',
        'profile-change',
        'story',
        'universal-timer-notification',
        'verified-change'
      )
      AND NOT (
        type IS 'message-request-response-event'
        AND json_extract(json, '$.messageRequestResponseEvent') IN ('ACCEPT', 'BLOCK', 'UNBLOCK')
      )
    ), shouldAffectActivity INTEGER
        GENERATED ALWAYS AS (
      type IS NULL
      OR
      type NOT IN (
        'change-number-notification',
        'contact-removed-notification',
        'conversation-merge',
        'group-v1-migration',
        'keychange',
        'message-history-unsynced',
        'profile-change',
        'story',
        'universal-timer-notification',
        'verified-change'
      )
      AND NOT (
        type IS 'message-request-response-event'
        AND json_extract(json, '$.messageRequestResponseEvent') IN ('ACCEPT', 'BLOCK', 'UNBLOCK')
      )
    ), isAddressableMessage INTEGER
        GENERATED ALWAYS AS (
          type IS NULL
          OR
          type IN (
            'incoming',
            'outgoing'
          )
        ), timestamp INTEGER, received_at_ms INTEGER, unidentifiedDeliveryReceived INTEGER, serverTimestamp INTEGER, source TEXT, isSearchable INT
      GENERATED ALWAYS AS (isViewOnce IS NOT 1 AND storyId IS NULL) VIRTUAL, searchableText TEXT GENERATED ALWAYS AS (
      CASE
        WHEN json->'poll' IS NOT NULL THEN json->'poll'->>'question'
        ELSE body
      END
      ) VIRTUAL, hasUnreadPollVotes INTEGER NOT NULL DEFAULT 0, hasExpireTimer INTEGER NOT NULL
    GENERATED ALWAYS AS (COALESCE(expireTimer, 0) > 0) VIRTUAL, hasPreviews INTEGER NOT NULL
      GENERATED ALWAYS AS (
        IFNULL(json_array_length(json, '$.preview'), 0) > 0
      ), hasContacts INTEGER NOT NULL
      GENERATED ALWAYS AS (
        IFNULL(json_array_length(json, '$.contact'), 0) > 0
      ))
```

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

## Deployment: Installed-Instance Recovery

**Topology (probed during planning, unchanged at execution):**
- `/usr/local/lib/topos/plugins/` (the TRUSTED tier) deliberately carries no Signal binary — Signal is not in the released kernel's build manifest.
- The installed kernel actually runs `~/.local/share/topos/plugins-external/topos-plugin-signal` (the EXTERNAL tier).
- `~/.config/topos/config.toml`'s `[plugins.pins] topos-plugin-signal` holds a SHA-256 content pin, enforced at launch.

**What this run did:**
1. `make signal` rebuilt `bin/plugins/topos-plugin-signal` from the bumped source (SHA-256 `d6cf7168...`).
2. `make install-signal` placed it atomically into `/home/darren/.local/share/topos/plugins-external/topos-plugin-signal` (resolved via `XDG_DATA_HOME`, the kernel default). Confirmed byte-identical to the dev-tree binary after placement.
3. This deliberately changed the pinned bytes: `~/.config/topos/config.toml`'s recorded pin (`8b1703aa...`) no longer matched the newly placed binary (`d6cf7168...`) — confirmed read-only, not edited by this run. Editing the pin from a script would bypass the deliberate human consent gate (Phase 11's open security todo already warns against exactly this).

**`install-signal.sh`'s printed one-time steps, verbatim:**
```
install-signal: external plugin directory: /home/darren/.local/share/topos/plugins-external (kernel default (XDG_DATA_HOME))
install-signal: placed /home/darren/.local/share/topos/plugins-external/topos-plugin-signal

install-signal: one-time steps (docs/plugins/signal.md 'The fix, step by step'):
install-signal:   This binary is untrusted by construction — it was not built beside
install-signal:   the kernel that will run it, so the kernel's link-time build manifest
install-signal:   cannot vouch for it.
install-signal:   1. Restart (or start) your installed kernel.
install-signal:   2. Add the Signal source through the app's untrusted-add consent flow —
install-signal:      the same explicit consent-and-pin path any external binary goes through.
install-signal:   3. It then runs pinned and badged untrusted.
install-signal:   Re-running 'make install-signal' later produces new bytes: the changed
install-signal:   binary must be re-accepted through the chip's re-pin flow.
install-signal:   If your config's [plugins] external_dir names a different directory,
install-signal:   re-run with TOPOS_EXTERNAL_PLUGINS_DIR=<that directory>.
```

**Task 3 human verification outcome — APPROVED:** The operator restarted the installed kernel, opened the installed instance's web UI (production port, not the `make dev` port — `config.dev.toml` has no Signal source and cannot demonstrate this recovery), saw the Signal source chip surface a named pin mismatch, used the chip's re-pin affordance to re-accept the changed binary through the app's untrusted-add consent-and-pin flow, triggered a refresh, and confirmed: the health indicator resolved to green (the "unrecognised database schema version" failure text was gone), Signal digests render in the stream with senders resolved and the transcript intact, and other sources were unaffected. Operator's verbatim response: *"approved — the Signal source re-pinned via the chip's consent flow, synced green, and digests render correctly on the installed instance (senders resolved, transcript intact)."*

## Decisions Made

- `highestSupportedSchemaVersion` raised to 1760 — the exact integer the live `PRAGMA user_version` read returned in Task 1, not assumed or quoted from the refusal message.
- Third provenance entry appended in the same shape as the existing two, explicitly recording that this verification crossed a real Signal Desktop package boundary (`8.21.0-1` → `8.22.0-1`) — the first time that has happened in this project's schema-acceptance history — while preserving the existing corrected guarantee statement (the constant tracks newest schema *state* verified, not newest release *supported*) rather than treating the release-boundary crossing as evidence against it.
- `guardSchemaVersion`'s body, its error message, `TestSchemaVersionCeiling`'s three relative test cases, and both verification tooling files (`schema_readset.go`, `live_schema_test.go`) were left untouched — per L-4 and L-8, and the plan's own key_links.
- The rebuilt binary was placed only via `make install-signal` into the EXTERNAL tier; the operator's `config.toml` pin was left untouched by this run so the re-pin remained a deliberate, human-driven consent step (L-9), resolved in Task 3.
- The pending verify-and-accept tooling todo (`signal-schema-version-verify-and-accept-tooling`) remains open — this run reused the precedent's tooling exactly as-is and surfaced no new friction that would change the shape of that future tooling.

## Deviations from Plan

None — plan executed exactly as written for all three tasks. Task 1 confirmed its precondition (observed version strictly greater than the compiled ceiling) before Task 2 proceeded; the live verification result was clean on the first attempt, so the "dirty diff halts the task" path (L-5) was not exercised. Task 3's blocking checkpoint was resolved by the operator's live verification and explicit "approved" response, as designed.

## Issues Encountered

None. The live database opened cleanly on the first attempt (plaintext `key` field in `config.json`, consistent with the environment findings recorded in the plan). No column was missing, no table was gone, and no read function errored.

## User Setup Required

None beyond Task 3 itself, which was the designed human-verification step (restart the installed kernel, exercise the re-pin consent flow, confirm live rendering) — not an external service configuration step. That step is complete and approved.

## Next Phase Readiness

- The Signal source is fully recovered on the operator's installed instance: green sync, correctly-rendering digests, other sources unaffected.
- The refreshed Schema Acceptance Record above (1760, package `8.22.0-1`) is now the precedent the next bump should diff against.
- The pending verify-and-accept tooling todo remains open, unchanged in scope — this run's clean, uneventful re-run of the existing tooling is itself evidence that the precedent's minimal-tooling approach (declared read set + opt-in live test, no subcommand/auto-bumper/snapshot fixture) continues to be sufficient for a routine bump, including one that crosses a Signal Desktop release boundary.

---
*Phase: quick-260819-jc1*
*Completed: 2026-08-19*
