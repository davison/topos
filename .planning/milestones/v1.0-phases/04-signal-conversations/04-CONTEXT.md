# Phase 4: Signal Conversations - Context

**Gathered:** 2026-08-02
**Status:** Ready for planning

<domain>
## Phase Boundary

The Signal plugin (SRC-02): messages from Signal conversations and groups whose names match the webspace keyword appear in the stream as conversation-day digests, with a detail pane showing the surrounding thread and an "open in Signal" affordance at conversation-only fidelity. Signal Desktop's local SQLCipher database is opened strictly read-only (`mode=ro`) and must be byte-identical after a full sync — including while Signal Desktop is running. The decryption key is obtained from whichever keyring backend the user's install actually uses (detected at runtime, never assumed), and an unrecognised DB schema version fails loudly, naming the version found. This phase introduces the chat-thread renderer that Phase 5 (WhatsApp) reuses — extend the existing UI contract, don't define a new one. No WhatsApp, no agent chat, no write paths.

**Mandatory spike before planning** (per ROADMAP.md notes): Signal Desktop DB schema, `safeStorage` keyring extraction tested hands-on against the user's actual Arch/DE setup, schema-version detection, SQLCipher/SQLite version stability (pin SQLite ≥ 3.51.3; never VACUUM or checkpoint).

</domain>

<decisions>
## Implementation Decisions

### Stream granularity
- **D-01:** One Signal stream item = a **conversation-day digest** — one row per conversation per day with activity ("House Move — 23 messages"). Never one row per message (chat volume would drown other sources) and never a single per-conversation row that resurfaces. — **Reversibility:** costly — item identity (`source_id` shape), digest assembly in the plugin, and the FTS preview contract all bake in the day-digest unit; Phase 5 (WhatsApp) reuses the same shape.
- **D-02:** Digest row content: title = conversation name + message count for the day; snippet = the **last 2–3 messages** of the day (the tail), each prefixed with sender name.
- **D-03:** FTS/search indexes **only the tail snippet** shown in the preview — no hidden full-day text in the index. Deliberate privacy trade: minimal Signal plaintext lands in the unencrypted index DB (consistent with the stack's "don't duplicate chat plaintext" rule); finding an old message means opening the thread, not keyword search. — **Reversibility:** reversible — widening what's indexed later is a re-sync, not a migration.
- **D-04:** A "day" is **midnight-to-midnight in the user's local timezone**. The digest's stream timestamp is its **last message of that day**, so it sits where the conversation left off. Today's digest updates in place (count, snippet, timestamp) as new messages sync.

### Conversation matching
- **D-05:** Eligible conversations: **groups and 1:1 chats**. Note to Self is excluded. Matching itself stays exact, case-insensitive against the shared keyword list (Phase 1 D-02/D-03 — locked; a group named differently is handled by adding its exact name as a keyword).
- **D-06:** For 1:1 chats, the keyword matches the **user's own names for the contact** — the nickname set in Signal and the address-book/system contact name — and **never the contact's self-chosen profile name**. Rationale (user-stated): a contact must not be able to pull themselves into a webspace by renaming their own profile.
- **D-07:** Renames mirror source truth: if a conversation's name changes so it no longer matches, its digests disappear at the next sync — identical to every other source (Phase 2 D-10, index mirrors source). Re-adding the new name as a keyword restores the full history. No sticky-membership memory.
- **D-08:** **Full history backfill**: every day with activity in Signal Desktop's DB gets a digest for matched conversations — no time window, consistent with how documents, notes, and email backfill.

### Claude's Discretion
The user left these areas to research/planning:
- **Thread view rendering** in the detail pane (the renderer Phase 5 reuses): bubble vs transcript layout, how much surrounding context a digest opens into, sender grouping, day headers.
- **Message richness**: how attachments, reactions, quotes, edits, and disappearing/deleted messages render in digests and the thread view.
- **Deep-link mechanics** for "open in Signal" (conversation-only fidelity is fixed; the exact `sgnl://` or launch mechanism is discretion).
- Keyring-failure / Signal-not-installed UX (existing health-chip + fail-loudly patterns apply), sync cadence for a local DB file, and the config shape for a source with no `base_url`/`token` — see the known `kernel/config.Validate` relaxation below.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope & requirements
- `.planning/ROADMAP.md` — Phase 4 goal, 5 success criteria, and notes (mandatory spike list; chat renderer reused by Phase 5; SQLite ≥ 3.51.3 pin; never VACUUM/checkpoint).
- `.planning/REQUIREMENTS.md` — SRC-02 (the phase's single requirement, full text).
- `.planning/PROJECT.md` — constraints (read-only, privacy, all-data-local) and Key Decisions table.
- `.planning/phases/01-first-webspace-end-to-end/01-CONTEXT.md` — locked config/keyword decisions (TOML, shared keyword list, exact case-insensitive match, env-var secrets).
- `.planning/phases/02-two-sources-one-trustworthy-stream/02-CONTEXT.md` — coordinator single-flight semantics, health/staleness UI rules, agent-grant config shape every new source must declare.

### Published contracts (extend, don't break)
- `docs/plugin-contract.md` — third-party plugin contract; the Signal plugin is another conforming plugin.
- `docs/api.md` — HTTP envelope, error codes, content/rendition routes.
- `proto/webspaces/v1/` — Item already carries `group_id` ("chat thread / mail conversation") and `LINK_FIDELITY_CONVERSATION_ONLY`; any new RPC fails the build until allowlisted.
- `config.example.toml` — extend with the Signal source block (local-path source, no base_url/token).

### Technology stack (locked)
- `.claude/CLAUDE.md` — Signal-specific stack: `mutecomm/go-sqlcipher/v4` (cgo, isolated in the plugin's own module so the kernel stays cgo-free), key unwrap via Electron `safeStorage` → freedesktop Secret Service D-Bus (`godbus/dbus/v5`), `encryptedKey`/`safeStorageBackend` fields in `~/.config/Signal/config.json`; "What NOT to Use" includes the outdated plaintext-`config.json` key assumption.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `plugins/proton/` — the freshest full plugin exemplar (client, plugin.go, readonly/outbound-host/live tests, credentials handling); the Signal plugin mirrors its structure but with a local file instead of a network endpoint.
- `web/src/lib/components/` — StreamRow/StreamList/DetailPane/SourceHealthChip etc.; DetailPane branches on content shape (`detailBodyVariant`: html/media/text), so the thread view is a new content shape rendered source-agnostically, per the Phase 3 "producing plugin decides readability" decision.
- `kernel/syncer` single-flight coordinator + `sync_runs` health history — Signal registers like any other source; no new sync machinery.
- Proto `Item.group_id` — designed for exactly this; digests set it to the conversation id.

### Established Patterns
- Read-only enforcement by test (AST read-only tests, RPC allowlist) — the Signal plugin needs the strongest version yet: `mode=ro` DSN plus a byte-identical-after-sync guarantee (success criterion 3) worth a dedicated test.
- Per-item rejection (not whole-batch) on contract violations at sync time.
- Sync-failure branch renders before empty branch; a failed sync never looks like an empty webspace.
- Egress: paperless/SilverBullet/proton pin outbound hosts — Signal is the inverse (a local-only plugin should have NO outbound network at all; its egress test can assert zero hosts).

### Integration Points
- `kernel/config/config.go` `Validate` — unconditionally requires `base_url`+`token` per source; the logged Phase 02-04 deferred item lands here: relax for local-path sources like Signal.
- `kernel/pluginhost` discovery — Signal plugin binary joins `bin/plugins/`; its cgo build stays in its own Go module (go.work member) so `go build ./...` on the kernel remains cgo-free.
- `kernel/index` — digests are ordinary Items; day-digest identity means `source_id` must encode (conversation, local-date) stably across re-syncs.

</code_context>

<specifics>
## Specific Ideas

- User's stated rationale for D-06: match on "display name and address-book name, but not the contact's own profile name" — so a contact renaming their profile can never inject themselves into a webspace. Preserve this exact behavior in matching code and docs.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 4-Signal Conversations*
*Context gathered: 2026-08-02*
