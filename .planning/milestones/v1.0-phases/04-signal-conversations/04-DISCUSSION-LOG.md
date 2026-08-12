# Phase 4: Signal Conversations - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-02
**Phase:** 4-Signal Conversations
**Areas discussed:** Stream granularity, Conversation matching

---

## Stream granularity

### What is one Signal item in the stream?

| Option | Description | Selected |
|--------|-------------|----------|
| Conversation-day digest | One row per conversation per day with activity; detail pane opens the thread | ✓ |
| Per message | Every Signal message its own stream row, like email; chat volume dominates the stream | |
| Per conversation | One row per matched conversation, resurfacing on latest activity | |

**User's choice:** Conversation-day digest (recommended option)

### What does a digest row show?

| Option | Description | Selected |
|--------|-------------|----------|
| Name + count + tail | Title: conversation name + message count; snippet: last 2–3 messages of the day with sender names | ✓ |
| Name + count + head | Snippet shows the first messages of the day instead | |
| Name + full-day text | Preview stores the whole day's messages; richest search but large previews | |

**User's choice:** Name + count + tail (recommended option)

### How much Signal text does the local index hold for search?

| Option | Description | Selected |
|--------|-------------|----------|
| Tail snippet only | Index only what the preview shows; minimal Signal plaintext in the index | ✓ |
| Full day, search-only | Whole day's text in FTS (not shown); a second full copy of Signal plaintext in the unencrypted index | |
| You decide | Leave to research/planning | |

**User's choice:** Tail snippet only (recommended option)
**Notes:** Deliberate privacy-over-search-reach trade, consistent with the stack's "don't duplicate chat plaintext" rule.

### Day boundary and digest timestamp

| Option | Description | Selected |
|--------|-------------|----------|
| Local midnight, last-msg time | Midnight-to-midnight local time; digest timestamped at its last message | ✓ |
| Local midnight, first-msg time | Same boundary, anchored where the day's conversation started | |
| Activity sessions | Split by quiet gaps (e.g. >4h) instead of calendar days | |

**User's choice:** Local midnight, last-msg time (recommended option)

---

## Conversation matching

### Which conversations are eligible?

| Option | Description | Selected |
|--------|-------------|----------|
| Groups + 1:1 chats | Keyword can match a group name or a 1:1 contact name | ✓ |
| Groups only | Only group names matched | |
| Groups + 1:1 + Note to Self | Also treat Note to Self as matchable | |

**User's choice:** Groups + 1:1 chats (recommended option)

### Which name(s) match for a 1:1 chat?

| Option | Description | Selected |
|--------|-------------|----------|
| Display name | The single name Signal Desktop displays (nickname → address-book → profile precedence) | |
| Any known name | Any of nickname, address-book, or profile name matching includes the chat | |
| You decide | Let research confirm the schema and pick | |

**User's choice:** Other (free text): "Display name and address-book name, but not the user's own profile name (for the reason outlined)"
**Notes:** The user's names for the contact (nickname + address book) both match; the contact's self-chosen profile name never does — a contact must not be able to name themselves into a webspace.

### What happens to digests when a matched conversation is renamed away?

| Option | Description | Selected |
|--------|-------------|----------|
| Mirror source truth | Digests disappear at next sync, same rule as every other source; re-adding the new name as a keyword restores history | ✓ |
| Sticky once matched | Past digests stay after a rename; index needs its own membership memory | |

**User's choice:** Mirror source truth (recommended option)

### How far back does backfill go?

| Option | Description | Selected |
|--------|-------------|----------|
| Full history | Every day with activity in Signal Desktop's DB gets a digest | ✓ |
| Window, configurable | Last N months by default, TOML-overridable | |

**User's choice:** Full history (recommended option)

---

## Claude's Discretion

- Thread view rendering in the detail pane (bubble vs transcript, context depth, sender grouping) — area offered but not selected for discussion
- Message richness: attachments, reactions, quotes, edits, disappearing/deleted messages — area offered but not selected
- Deep-link mechanics for "open in Signal" (fidelity fixed at conversation-only)
- Keyring-failure / Signal-not-installed UX, local-DB sync cadence, config shape for a no-base_url/no-token source

## Deferred Ideas

None — discussion stayed within phase scope.
