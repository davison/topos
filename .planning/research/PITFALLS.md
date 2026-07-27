# Pitfalls Research

**Domain:** Local-first personal cross-source data aggregation (kernel + plugin architecture; email/chat/document/wiki correlation)
**Researched:** 2026-07-27
**Confidence:** MEDIUM-HIGH (source access patterns for Signal/Proton/SQLite are HIGH confidence, drawn from official repos and documented issues; WhatsApp ban mechanics and rates are MEDIUM/LOW confidence, drawn from secondary community sources with no official statistics)

## Critical Pitfalls

### Pitfall 1: Assuming the official WhatsApp Desktop app has a readable local message history

**What goes wrong:**
The plan (per PROJECT.md) says the WhatsApp plugin will "read WhatsApp desktop/linked-device local store on the same machine." The official WhatsApp Desktop/Web app is a *thin mirror* of the phone — it caches chat previews and recently viewed media locally but does not maintain a durable, decryptable full-history SQLite store the way the Android app's `msgstore.db` does. Building a plugin that tries to parse the official desktop app's local files will hit a wall: there's no persistent multi-year archive sitting on disk to read.

**Why it happens:**
The mental model "Signal Desktop has a local DB, so WhatsApp Desktop must too" is reasonable but wrong — WhatsApp's official desktop client and WhatsApp Web are companion/mirror clients by design, not full local replicas.

**How to avoid:**
Treat the WhatsApp plugin as an *active linked-device client*, not a passive file reader. The realistic architecture (validated by existing prior art like `wacli`, which is built on `whatsmeow`) is: your plugin itself links as a WhatsApp companion device via the multi-device protocol, receives the live event stream, and persists messages into its *own* local SQLite store as they arrive — the plugin becomes the durable store, not a reader of one. This has real consequences for the plugin contract (long-running connection/session lifecycle, not just a periodic file scan) and for what "history" means (see Pitfall 8).

**Warning signs:**
- Design docs or code describe the WhatsApp plugin as scanning a directory of existing WhatsApp Desktop files
- No mention of a persistent linked-device *session* the plugin itself owns and reconnects

**Phase to address:**
WhatsApp plugin phase — must be scoped from the start as "run a whatsmeow-based linked device and build a local mirror," not "parse existing files."

---

### Pitfall 2: WhatsApp account ban / device de-link from using an unofficial client

**What goes wrong:**
Any library speaking the WhatsApp multi-device protocol without Meta's official Business API (whatsmeow, Baileys, and everything built on them, including `wacli`) is, by WhatsApp's Terms of Service, an unauthorized client. Community reports (secondary sources, not official Meta statistics) describe device-linking being blocked for accounts that trip detection, and in the worst case account suspension. Because a webspace links as a *companion device* consuming one of WhatsApp's 4 extra device slots, a ban or forced de-link doesn't just break the plugin — it can disrupt the user's real phone-based WhatsApp use too.

**Why it happens:**
There is no officially sanctioned personal-use read API for WhatsApp; every non-Business-API integration is reverse-engineered against a protocol Meta can and does change without notice, and detection heuristics are opaque and can flag benign long-lived unofficial sessions.

**How to avoid:**
- Treat this as an accepted, disclosed risk to the user — not a solved problem. Document it explicitly as a tradeoff in PROJECT.md/Context (already partially captured).
- Use a well-maintained library (whatsmeow, actively updated as of mid-2026) rather than an abandoned fork — protocol breakage is routine and unmaintained libraries stop working within months.
- Isolate the WhatsApp plugin so a ban/de-link degrades gracefully (feed goes stale, rest of the system unaffected) rather than crashing the kernel.
- Avoid behavior that increases detection risk: no bulk backfill scraping in tight loops, no automation that resembles bot/spam patterns, keep the session long-lived and stable rather than repeatedly relinking.

**Warning signs:**
- Repeated forced logouts/relink prompts
- WhatsApp requiring phone-side re-verification unexpectedly
- Primary phone inactive >14 days (this alone logs out *all* linked devices, unrelated to detection — a real operational gotcha, not just a ban risk)

**Phase to address:**
WhatsApp plugin phase (build), plus a resilience/reconnection story in the kernel's plugin lifecycle (a plugin can go from healthy to permanently broken without warning, and the UI needs to reflect that rather than silently going stale).

---

### Pitfall 3: Signal Desktop key-access method changes across OS keyring backends and Signal versions

**What goes wrong:**
Since 2024, Signal Desktop protects its SQLCipher key via Electron's `safeStorage` API. On Linux this defers to the freedesktop.org Secrets API — GNOME Keyring on GNOME, KWallet on KDE. The key in `config.json` is no longer plaintext (as it was on older Signal versions and is still sometimes described in outdated guides) — it's wrapped by whichever keyring backend was active when Signal last started. If the user's desktop environment changes, or the keyring daemon isn't running/unlocked, Signal Desktop itself can fail to start with "OS encryption keyring backend has changed" errors — and a third-party plugin trying to extract the key will fail the same way, silently, if it assumes a single fixed extraction method.

**Why it happens:**
Most "how to decrypt Signal Desktop" writeups online predate the `safeStorage` migration and describe the old plaintext-key approach; building against that stale information breaks on any current Signal Desktop install on Linux.

**How to avoid:**
- Detect which keyring backend is active (read the `safeStorageBackend` field in `config.json`) before attempting extraction, and fail with a clear, specific error rather than a generic decrypt failure.
- Require the appropriate keyring daemon (gnome-keyring or kwalletd, matching the desktop environment) be unlocked/running as a documented precondition for the Signal plugin, since the deployment target is the user's own Arch Linux desktop.
- Do not hardcode assumptions from forensics blog posts written against Windows/older macOS plaintext-key behavior.

**Warning signs:**
- Plugin works in initial dev testing but breaks after a Signal Desktop auto-update
- Plugin works on GNOME but fails on KDE (or vice versa) if the dev machine and eventual real machine differ

**Phase to address:**
Signal plugin phase — key-extraction code must branch on backend type, and the phase's acceptance criteria should include "works against the actual OS/DE combination the user runs," not just a dev sandbox.

---

### Pitfall 4: Signal Desktop database schema evolves and silently breaks external parsers

**What goes wrong:**
Signal Desktop's SQLite schema has changed substantially across its lifetime (recipient-ID model changes, sms/mms table merges, column drops, trigger rewrites) and continues to change with app updates. A plugin hardcoded against today's schema will silently misparse or crash against tomorrow's Signal Desktop release — and unlike a web API, there's no version negotiation; the plugin discovers breakage only when Signal auto-updates.

**Why it happens:**
Signal Desktop is not a documented integration surface; its schema is an internal implementation detail with no stability guarantee, changed at Signal's discretion.

**How to avoid:**
- Read and check the database's own internal schema/user_version before parsing, and fail loudly (not silently return wrong data) on an unrecognized version.
- Keep the Signal parsing logic isolated behind a narrow internal interface so schema-version branches are additive, not a rewrite of the whole plugin.
- Pin against a known-working Signal Desktop version range in documentation, and treat "Signal auto-updated" as a routine maintenance event, not an exceptional bug.

**Warning signs:**
- Plugin returns empty/garbled results with no error after a Signal Desktop update
- Message counts or timestamps look subtly wrong rather than obviously erroring

**Phase to address:**
Signal plugin phase, with schema-version detection built in from day one rather than added reactively after the first breakage.

---

### Pitfall 5: Unsafe concurrent access to Signal's live, open SQLite database

**What goes wrong:**
Signal Desktop keeps its database open (typically in WAL mode) while running. Simple reading from a second connection while Signal is open is generally safe — WAL allows concurrent readers without blocking the writer — but any code path that triggers a write or a checkpoint from the reading side (e.g., a library that opens the file in default read-write mode and issues even an incidental write, or calls a checkpoint API) risks corruption. There is also a documented class of rare WAL-reset corruption bugs in SQLite itself (present in versions 3.7.0 through 3.51.2, fixed in 3.51.3, March 2026) that a project should not be shipping against unknowingly.

**Why it happens:**
Developers assume "open a SQLite file" is inherently safe regardless of connection mode, and don't realize their SQLite driver/version might default to read-write or attempt an automatic checkpoint.

**How to avoid:**
- Open Signal's (and WhatsApp's, if applicable) database explicitly in read-only mode (e.g., SQLite's `mode=ro` URI, or the driver's read-only flag) — never rely on "we just don't call write methods."
- Pin/verify the SQLite library version in use is ≥3.51.3 to avoid the known WAL-reset bug class.
- Never attempt to `VACUUM`, checkpoint, or otherwise touch write-adjacent operations against a live source database.
- Consider copying the DB file to a temp location before reading if the read-only guarantee can't be verified end-to-end (with the caveat that live-in-use files can be mid-write when copied — snapshot with appropriate care, e.g., via a filesystem-level exclusive lock check or the SQLite backup API rather than a raw `cp`).

**Warning signs:**
- Signal Desktop reports database errors or corruption shortly after the plugin runs
- Read-only mode isn't explicitly configured in code — it's just assumed

**Phase to address:**
Kernel/plugin-contract phase — read-only access should be a *contract-level guarantee* enforced once (e.g., a shared "open source DB read-only" helper all DB-backed plugins use), not left to each plugin author's discipline. Also re-verified in the Signal plugin phase specifically.

---

### Pitfall 6: Proton Mail Bridge's localhost-only design fights the LAN deployment plan

**What goes wrong:**
Proton Mail Bridge is architected to bind only to `127.0.0.1` and issues a self-signed certificate scoped to that assumption — it is explicitly *not* designed to be reachable over a LAN. PROJECT.md's plan (bridge on a separate home server, reached over LAN from the desktop) requires fighting this design: rebinding the listener to the LAN interface or tunnelling, and then trusting/pinning a self-signed cert across a network hop it wasn't built for. Naively rebinding to `0.0.0.0` exposes full IMAP/SMTP access to the Proton account to anything on the LAN, gated only by the Bridge-generated password — a materially larger attack surface than Bridge's authors intended.

**Why it happens:**
Bridge's self-signed cert and localhost bind are a deliberate security boundary (Proton assumes Bridge and mail client share a machine), not an oversight; treating it as "just another IMAP server to point at" skips that context.

**How to avoid:**
- Prefer tunnelling over raw LAN exposure: SSH port-forward or a WireGuard/VPN hop from desktop to the home server, so Bridge still only ever sees `127.0.0.1` traffic locally, with the tunnel providing the LAN traversal and its own auth.
- If rebinding directly is chosen instead, pin the Bridge's self-signed certificate fingerprint in the plugin's IMAP client config (don't blindly accept-any-self-signed-cert), and restrict LAN reachability to the specific desktop via firewall rules — not open to the whole LAN.
- Document the cert-pinning/tunnel requirement as a hard dependency in the email plugin phase, since it's infrastructure work outside the plugin code itself (bridge/server-side firewall changes).

**Warning signs:**
- IMAP client configured to accept-all-certificates ("disable TLS verification") — a red flag with no upstream trust anchor to fall back on
- Bridge process listening on `0.0.0.0` with no additional network-layer restriction

**Phase to address:**
Email/IMAP plugin phase — the connection-setup step should explicitly require either a tunnel or a pinned-cert LAN config, and reject "insecure/no-verify" as an acceptable default.

---

### Pitfall 7: Proton's label model produces duplicate messages across IMAP folders

**What goes wrong:**
Proton Mail's labels (non-exclusive, a message can have several) are exposed through Bridge as IMAP folders. A message with two labels appears as two distinct messages (in Inbox and in each label-folder) from a naive IMAP client's point of view. A correlation engine that treats "message present in folder X" as "one item" will double- (or triple-) count the same email across multiple webspaces/matches unless it de-duplicates.

**Why it happens:**
IMAP's folder model assumes one message lives in one place; Proton's label model doesn't map cleanly onto that, and Bridge papers over the mismatch by literally duplicating the message across folder views rather than exposing a true many-to-many label mechanism over IMAP.

**How to avoid:**
- De-duplicate by a stable identity (Message-ID header, or IMAP `X-GM`-style extension if available, or a hash of envelope+headers) before treating an item as a unique correlation match, regardless of how many folders/labels it appears under.
- When multiple labels on one message both match different configured webspace keywords, surface it as one item associated with multiple webspaces — not duplicate stream entries.

**Warning signs:**
- Same email subject/timestamp appearing twice in a single webspace's stream
- Item counts in a webspace roughly double the expected count for label-heavy accounts

**Phase to address:**
Email/IMAP plugin phase — dedup logic belongs in the plugin's ingestion path, before metadata reaches the shared index.

---

### Pitfall 8: Read-only guarantee silently broken by naive IMAP fetch calls

**What goes wrong:**
The project's core constraint is "plugins must never mutate source data." A common, easy-to-miss violation: fetching a message body with a plain `FETCH BODY[...]` IMAP command marks the message `\Seen` as a side effect defined by the IMAP protocol itself — even though no code path explicitly "wrote" anything. This silently changes read/unread state in the user's real mailbox (and in Proton Mail's own UI) just from previewing an email in a webspace.

**Why it happens:**
Most IMAP client libraries default to the "mark as seen" fetch behavior because that matches typical mail-client UX; using `BODY.PEEK[...]` instead requires deliberate opt-in that's easy to miss when using a library's convenience wrappers.

**How to avoid:**
- Audit every IMAP fetch call to confirm it uses `BODY.PEEK[]` (or the library's equivalent non-mutating accessor), never `BODY[]`.
- Add an integration test that fetches an unread message through the plugin and asserts the source mailbox's `\Seen` flag is unchanged afterward.
- Extend the same read-only audit mindset to any other flag-mutating IMAP verb (e.g., avoid `STORE` entirely; the plugin should have no code path capable of issuing it).

**Warning signs:**
- Emails in Proton Mail's own web UI show as read after only being viewed through a webspace
- No automated test exists that specifically checks flag-preservation post-fetch

**Phase to address:**
Email/IMAP plugin phase, with the read-only contract test added as a required verification step (see also the read-only-guarantee item in the "Looks Done But Isn't" checklist below).

---

### Pitfall 9: Hybrid index staleness — the index and the live source disagree

**What goes wrong:**
The chosen architecture (metadata/preview synced to a local index, full content fetched live on open) creates an inherent consistency gap: an item can be deleted, moved, relabeled, or renamed at the source after the index last synced. The "open in source" deep link then either 404s, opens the wrong item, or the live fetch silently fails — and if the UI doesn't handle this, the user loses trust in the whole system the first time a stream item turns out to be a ghost.

**Why it happens:**
Hybrid/local-index-plus-live-fetch designs are chosen precisely for speed and low duplication, but that tradeoff necessarily means the index is *not* the source of truth and can lag reality; teams often build the "happy path" (index matches source) and treat the mismatch case as an edge case rather than a first-class state.

**How to avoid:**
- Design the item detail view to explicitly handle and surface "no longer available at source" / "moved" states rather than assuming live fetch always succeeds.
- Include a lightweight incremental re-sync per source (watermark/cursor-based, not full rescan) so staleness windows stay small rather than accumulating.
- Decide per-source what "moved" detection looks like (e.g., IMAP UIDVALIDITY changes, paperless-ngx document ID no longer resolving, SilverBullet page renamed) and handle each explicitly rather than a single generic "fetch failed" catch-all.

**Warning signs:**
- Deep links that 404 with no explanit UI treatment
- No re-sync cadence defined, or re-sync implemented as "delete and rebuild the whole index"

**Phase to address:**
Hybrid data model phase (index design) and Web UI phase (staleness states in the detail pane) both need to address this — it can't be fixed in the UI alone if the index doesn't track enough to detect staleness.

---

### Pitfall 10: Plugin API designed around email/IMAP concepts, then broken by later sources

**What goes wrong:**
If the first plugin built is the IMAP/email one, it's tempting to bake IMAP-shaped concepts (folders, UIDs, flags) directly into the shared plugin contract. When the Signal, WhatsApp, paperless-ngx, and SilverBullet plugins are added, none of those map cleanly onto "folders and UIDs" — forcing either awkward shoehorning or a breaking rework of the contract exactly when the project's stated goal is "documented enough for third-party plugins."

**Why it happens:**
Building against one concrete source first and only generalizing afterward is natural, but the contract that results is often accidentally over-fit to that first source's data model.

**How to avoid:**
- Design the plugin contract's core abstractions (item, timestamp, native-category/tag, preview, deep-link) source-agnostically from the start, informed by sketching all five MVP sources' data shapes before finalizing the interface — not just the first one implemented.
- Keep source-specific concepts (IMAP UID, Signal conversation ID, WhatsApp JID, paperless-ngx document ID, SilverBullet page path) inside each plugin, exposed to the kernel only as an opaque per-source identifier plus the common fields.
- Treat the plugin API as versioned from v1 (even if only one version exists) so a deliberate deprecation path exists once a second consumer (a third-party plugin) appears.

**Warning signs:**
- Kernel/shared code has IMAP-specific types (e.g., `folder`, `uid`) leaking into interfaces meant to be source-agnostic
- Adding the second (Signal) plugin requires touching the shared contract far more than the plugin implementation itself

**Phase to address:**
Kernel + plugin architecture phase — sketch the contract against at least two structurally different sources (e.g., IMAP and Signal, or IMAP and paperless-ngx) before writing the first plugin, even though only the email plugin will be fully implemented that phase.

---

### Pitfall 11: Scope creep toward replicating source-app functionality

**What goes wrong:**
"Out of Scope" explicitly excludes write/edit and replicating source-app features (composing email, replying to chats, editing notes). In practice, this boundary erodes gradually: a "quick reply" button feels natural once you're staring at a chat thread in the UI; "mark as read from here" feels convenient; "add a tag" feels like a small addition. Each one individually seems reasonable, but together they turn Webspaces from a safe, view-only aggregator into a second, partial, less-capable client for every source — which is both a huge maintenance burden (mirroring N source apps' write semantics) and a direct threat to the read-only safety guarantees this document's other pitfalls depend on.

**Why it happens:**
Once a rich preview UI exists, every "why can't I just also..." request looks like a small increment rather than a scope violation, because the UI is right there and the source's write API often technically exists.

**How to avoid:**
- Treat "view-only, deep-link out for anything else" as a hard architectural invariant enforced at the plugin-contract level (plugins literally have no write methods available to call), not just a product-scope decision that can be quietly overridden by a future feature request.
- Any request that starts with "just let me quickly do X from within the webspace" should be redirected to "open in source" by design, and treated as validation that the deep-link UX needs to be fast/frictionless — not a signal to add write capability.
- Re-review this boundary explicitly at each milestone transition (the project's own process already does a "Requirements/Out of Scope" review — make read-only-ness itself one of the audited items, not just individual features).

**Warning signs:**
- Any plugin interface gains a method whose purpose is anything other than "read and return data"
- UI design discussions include verbs like "reply," "edit," "compose," "archive," or "delete" attached to a webspace item

**Phase to address:**
Kernel + plugin architecture phase (enforce no-write capability structurally) and every subsequent phase as an ongoing review item, not a one-time gate.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|-----------------|------------------|
| Store full email/chat bodies in the local index instead of preview + live fetch | Faster to build, no live-fetch error handling needed | Defeats the whole point of the hybrid model — duplicates sensitive content locally, drifts from source, larger attack surface for "data at rest" | Never for the MVP hybrid model; acceptable only as a throwaway prototype before the real architecture is built |
| Poll paperless-ngx/SilverBullet on a fixed interval instead of event/webhook-driven sync | Much simpler than wiring webhooks | Wasted cycles, coarser staleness window | Acceptable indefinitely at single-user/personal scale — this is a reasonable permanent choice, not just an MVP shortcut |
| Skip a schema-migration mechanism for the local index DB | Faster initial development | Any index schema change (e.g., adding a new source or field) requires a manual rebuild or breaks existing installs | Acceptable pre-v1 only; must exist before a second person besides the author runs this |
| Copy Signal Desktop's DB file to a temp location instead of true read-only live access | Sidesteps needing to prove live read-only safety | Copy timing races with an in-use file can grab a torn/mid-write snapshot; adds disk I/O and staleness | Acceptable as an interim step only if paired with a verified consistent-snapshot method (e.g., SQLite backup API), never a raw `cp` while Signal is running |
| Hardcode a single Proton Bridge / SilverBullet / paperless-ngx instance URL rather than configurable endpoints | One less config surface for a single-user MVP | Blocks any future second-instance or multi-environment use, and makes local dev/testing against a staging instance harder | Acceptable for v1 since this is explicitly a personal, single-user tool — revisit only if that constraint changes |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|-----------------|-------------------|
| Proton Mail Bridge | Assuming standard IMAP `MOVE`/label semantics apply, or exposing Bridge on the open LAN with cert verification disabled | Treat Bridge's IMAP as read/browse-only from this project's side; use `BODY.PEEK[]`; reach it via tunnel or a pinned self-signed cert with restricted firewall rules, never "disable TLS verification" |
| Signal Desktop DB | Assuming the key is plaintext in `config.json` (true only pre-2024 / non-Linux edge cases) | Detect the active `safeStorageBackend` and extract via the matching OS keyring (GNOME Keyring / KWallet), open the DB read-only, and version-check the schema before parsing |
| WhatsApp | Assuming an existing local file store can just be read (see Pitfall 1) | Run an active whatsmeow-based linked-device session that persists its own event stream into your own store; expect limited initial backfill, not full history |
| paperless-ngx REST API | Fetching all documents in one unpaginated call, or hitting `/api/selection_data/` for large result sets | Paginate through `/api/documents/`, filter server-side by tag/correspondent where possible, and load the tag list once to resolve tag IDs before any tag-based queries |
| SilverBullet | Assuming the sync/plugin layer guarantees no conflicts across concurrent writers | SilverBullet creates a "conflicting copy" file rather than merging on concurrent edits — since Webspaces is read-only this mostly doesn't apply to *your* writes, but be aware reads may transiently see a conflict-copy artifact if the user edits from two clients at once |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|-----------------|
| Full re-scan of Signal/WhatsApp DB or IMAP mailbox on every sync tick | Sync time grows linearly with total historical volume, not with new items | Use an incremental cursor per source (SQLite `rowid`/timestamp watermark, IMAP `UIDNEXT`/`MODSEQ`, paperless-ngx `created` filter) | Noticeable once a source has a few thousand items; becomes a real problem past tens of thousands |
| Eagerly fetching full body/attachments to build every stream preview | Sync/ingest slows down and network-fetches balloon even for items the user never opens | Generate lightweight previews at ingest time from headers/snippets only; fetch full content live, lazily, only when an item is opened | Breaks down once document/attachment-heavy sources (paperless-ngx especially) accumulate — full-content prefetch at ingest scales with total corpus size, not usage |
| Full-text scanning every source for keyword correlation instead of using native tags/labels/folders | Correlation step becomes slow and produces false positives from incidental keyword mentions | Use each source's own native categorization (IMAP folder/label, chat group name, paperless-ngx tag, SilverBullet tag/page) as the match target, per the project's own deterministic v1 design — don't quietly upgrade to full-text search as a "quick win" | Any dataset large enough that full-text scanning becomes the ingest bottleneck; also breaks the "no false positives" guarantee the deterministic model promises |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Storing Signal/WhatsApp session keys or derived secrets unencrypted in Webspaces' own config/index | Replicates the exact plaintext-key criticism leveled at pre-2024 Signal Desktop — any local-file-reading malware or other user on the machine gets full access | Rely on the OS keyring (same mechanism Signal itself now uses) or an equivalent encrypted-at-rest store for anything sensitive Webspaces itself persists |
| Binding Proton Bridge to `0.0.0.0` for LAN reachability with only the Bridge password as a gate | Opens full IMAP/SMTP access to the Proton account to the whole LAN segment | Prefer an authenticated tunnel (SSH/WireGuard) over raw rebind; if rebinding, restrict via firewall to the specific desktop IP and pin the cert fingerprint |
| Giving plugin code the same filesystem access level as the source apps themselves (e.g., full read-write handle to Signal's data directory) | A plugin bug could corrupt the live Signal/WhatsApp store even without intent to write | Open source databases via explicit read-only file handles/URIs at the point of access, and keep this enforced by the kernel's shared "open source DB" helper rather than plugin-author discipline |
| Treating "read-only by convention" as sufficient instead of structurally enforced | A future contributor's plugin could add a write path without anyone noticing until damage is done | Make the plugin interface itself incapable of exposing write methods (no `save`/`update`/`delete` in the contract at all) |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-------------------|
| Silent staleness — deep link 404s or opens the wrong item with no explanation | User loses trust in the whole aggregation; looks broken/buggy | Explicit "no longer available at source" / "may have moved" state in the detail pane, distinct from a generic error |
| Duplicate stream entries from Proton's multi-label model | Feed looks cluttered/buggy, undermines confidence in correlation quality | De-duplicate by stable message identity before display (see Pitfall 7) |
| No visibility into why an item wasn't included in a webspace | Since v1 correlation is deterministic keyword-to-native-category matching (not full-text/AI), users may wonder why an obviously-related item is missing | Some lightweight transparency (e.g., "N items in source X don't have a matching label/tag") builds trust in the deliberate deterministic-only design, rather than looking like a bug |

## "Looks Done But Isn't" Checklist

- [ ] **Signal plugin:** Often verified only against one Signal Desktop version/keyring backend on the dev machine — verify against the actual production machine's OS/DE combo, and add explicit version/backend detection with a clear failure message rather than a silent parse failure.
- [ ] **WhatsApp plugin:** Often claims "connected" after a successful link, without verifying how much history was actually backfilled — verify by checking the oldest available message timestamp immediately post-link, and surface that limit to the user rather than implying full history is present.
- [ ] **Read-only guarantee (all DB-backed plugins):** Often "verified" by code review ("we don't call write methods") rather than by an actual test — verify with an automated test that attempts to detect any mutation (checksum/mtime of the source file, or an assertion the DB connection itself is read-only-mode).
- [ ] **IMAP plugin:** Often verified by "email correctly displayed in the webspace," without checking source-side side effects — verify the source mailbox's `\Seen`/flag state is unchanged after a full sync-and-preview cycle.
- [ ] **Plugin contract "stability for third parties":** Often verified only by the plugins the author themselves wrote — verify by building (even a throwaway) second/independent plugin purely against the frozen documented interface, not against internal knowledge of the kernel's implementation.
- [ ] **Hybrid index staleness handling:** Often only tested on the happy path (index and source agree) — verify by manually deleting/renaming/relabeling a source item after indexing and confirming the UI degrades gracefully rather than erroring or showing wrong content.

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|----------------|------------------|
| Signal DB unreadable after a Signal Desktop upgrade or keyring backend change | MEDIUM | Detect the specific failure (schema-version mismatch vs. keyring-backend mismatch) and surface it distinctly; maintain a small table of known schema versions/keyring formats and add a compatibility branch rather than a full rewrite |
| WhatsApp linked device logged out or banned | MEDIUM | Re-link as a new device; because the plugin's own local store (built from the live event stream) is independent of the link status, prior history already ingested is not lost — only the gap since disconnection needs backfilling (and backfill may itself be limited, see Pitfall 1/8) |
| Local hybrid index corrupted or out of sync | LOW | Since the index is explicitly a derived cache and not the source of truth, the correct recovery is a full rebuild from live sources — this is a designed-in safety net of the hybrid model, not an emergency procedure |
| Proton Bridge certificate rotated or connection refused after a Bridge update | LOW | Re-pin the new certificate fingerprint and restart the email plugin's connection; treat Bridge updates as a routine, expected maintenance event |
| IMAP `\Seen` flags found to have been mutated by a fetch bug | LOW-MEDIUM | Flags are typically recoverable manually (mark back as unread) since email `\Seen` state, unlike Signal/WhatsApp message content, isn't destructively lost — but treat any occurrence as a release-blocking regression given the read-only guarantee, not a minor bug |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|-------------------|----------------|
| WhatsApp official-app-store misconception (P1) | WhatsApp plugin phase | Design doc explicitly describes an active whatsmeow-based linked-device session with its own persisted store, not a file reader |
| WhatsApp ban/de-link risk (P2) | WhatsApp plugin phase | Plugin degrades gracefully on disconnect/ban; risk explicitly documented for the user |
| Signal keyring backend churn (P3) | Signal plugin phase | Plugin detects `safeStorageBackend` and branches extraction accordingly; tested against the actual target machine's DE |
| Signal schema churn (P4) | Signal plugin phase | Plugin checks DB schema/user_version before parsing; fails loudly and specifically on unknown versions |
| Unsafe concurrent DB access (P5) | Kernel/plugin-contract phase (shared helper) + Signal plugin phase | Automated test confirms source DB opened read-only; no corruption after concurrent access with the live source app running |
| Proton Bridge LAN exposure (P6) | Email/IMAP plugin phase | Connection setup requires tunnel or pinned-cert LAN config; "accept any cert" rejected as a valid configuration |
| Proton label duplication (P7) | Email/IMAP plugin phase | Test with a multi-labeled message confirms single stream entry, not duplicates |
| IMAP Seen-flag mutation (P8) | Email/IMAP plugin phase | Automated test: fetch unread message, confirm `\Seen` unchanged on source after |
| Hybrid index staleness (P9) | Hybrid data model phase + Web UI phase | UI shows explicit "unavailable/moved" state for a manually-deleted/renamed source item |
| Plugin API over-fit to email (P10) | Kernel + plugin architecture phase | Contract sketched against ≥2 structurally different sources before first plugin is implemented |
| Scope creep toward write features (P11) | Kernel + plugin architecture phase, reviewed every milestone | Plugin interface has zero write-capable methods; milestone review explicitly re-confirms this |

## Sources

- [Signal-Desktop keyring backend migration issues #753/#754 — flathub/org.signal.Signal](https://github.com/flathub/org.signal.Signal/issues/753) (HIGH — official app-packaging issue tracker)
- [Migrating Signal Desktop keyring backend — Inane Observations](https://yingtongli.me/blog/2025/08/13/signal-secrets.html) (MEDIUM — independent technical writeup, corroborates GitHub issues)
- [Signal Desktop database schema history — DeepWiki bepaald/signalbackup-tools](https://deepwiki.com/bepaald/signalbackup-tools) (MEDIUM-HIGH — derived from the actively-maintained signalbackup-tools project that tracks schema changes)
- [Signal-Desktop GitHub issues on database errors after upgrades (#6597, #6639, #6970, #7029)](https://github.com/signalapp/Signal-Desktop/issues/7029) (HIGH — official repo)
- [SQLite: How To Corrupt An SQLite Database File](https://www.sqlite.org/howtocorrupt.html) (HIGH — official SQLite documentation)
- [SQLite Write-Ahead Logging documentation](https://www.sqlite.org/wal.html) (HIGH — official)
- [SQLite Forum: WAL-reset corruption bug discussion](https://sqlite.org/forum/info/d33843ff0dfdf9fd) (HIGH — official SQLite forum)
- [Proton: Labels in Bridge](https://proton.me/support/labels-in-bridge) (HIGH — official Proton documentation)
- [Proton Mail labels causing visual email duplication — eM Client forum](https://forum.emclient.com/t/proton-mail-labels-causing-visual-email-duplication/95637) (MEDIUM — corroborating third-party client bug reports)
- [Fixing ProtonMail Bridge SSL errors with Let's Encrypt](https://lder.dev/posts/Fixing-ProtonMail-Bridge-SSL-errors-with-Lets-Encrypt/) (MEDIUM — independent writeup documenting the LAN/cert workaround pattern)
- [Setting up ProtonMail Bridge on LAN server — vimoire](https://vimoire.com/blog/2025/setting_up_protonmail_bridge_on_lan_server) (MEDIUM)
- [IMAP BODY.PEEK vs BODY discussion — pythontutorials.net](https://www.pythontutorials.net/blog/fetch-an-email-with-imaplib-but-do-not-mark-it-as-seen/) (MEDIUM — practical, consistent with IMAP RFC behavior)
- [whatsmeow package documentation — pkg.go.dev](https://pkg.go.dev/go.mau.fi/whatsmeow) (HIGH — official library docs)
- [wacli — WhatsApp CLI built on whatsmeow](https://github.com/openclaw/wacli) (MEDIUM-HIGH — actively maintained prior-art project demonstrating the exact linked-device-plus-local-store architecture)
- [WhatsApp Help Center: About message history on linked devices](https://faq.whatsapp.com/653480766448040) (HIGH — official WhatsApp documentation)
- [Where WhatsApp Data is Stored on PC](https://blog.usro.net/2024/10/where-whatsapp-data-is-stored-on-pc-a-guide-to-locating-your-chats-media-and-backups/) (MEDIUM — corroborates official-app cache-only behavior)
- [WhatsApp Unofficial API Ban Risk — Wapisimo Blog](https://wapisimo.dev/blog/en/whatsapp-unofficial-api-ban-risk) (LOW-MEDIUM — vendor blog with anecdotal ban-rate claims, no official Meta data)
- [Paperless-ngx REST API documentation](https://docs.paperless-ngx.com/api/) (HIGH — official docs)
- [Paperless-ngx discussion: selection_data pagination performance #4054](https://github.com/paperless-ngx/paperless-ngx/discussions/4054) (HIGH — official repo discussion)
- [SilverBullet: Sync documentation](https://silverbullet.md/Sync) (HIGH — official docs, describes conflicting-copy behavior)

---
*Pitfalls research for: local-first personal cross-source data aggregation (Webspaces)*
*Researched: 2026-07-27*
