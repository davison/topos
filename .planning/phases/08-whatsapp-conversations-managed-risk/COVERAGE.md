# Phase 8 — API Coverage Matrix

**External integration:** `go.mau.fi/whatsmeow` — the WhatsApp multi-device linked-device client SDK.

The deterministic detector returned `detected: false` when run against the ROADMAP phase section alone (that section names no API/SDK term outside a code fence). This matrix is produced anyway because the phase genuinely integrates an external SDK, and because the plan bodies this phase produces will trip the detector at seal time. Full coverage is the default; every row below starts as `INTEGRATE` and every `OPT-OUT` carries a reason.

Capability surface enumerated from `go.mau.fi/whatsmeow`'s `Client` API and its `types/events` package at the pseudo-version Plan 08-01 Task 1 pins.

| capability | decision | reason |
|---|---|---|
| `pair-qr` — `Client.GetQRChannel`, QR-code device pairing | INTEGRATE | |
| `session-persist` — `store/sqlstore.Container`, `GetFirstDevice`, auto-save on pair | INTEGRATE | |
| `connect` — `Client.Connect`, `Disconnect`, `IsConnected`, connection lifecycle | INTEGRATE | |
| `events-message` — `AddEventHandler` for `events.Message` | INTEGRATE | |
| `events-history-sync` — `events.HistorySync` backfill ingestion | INTEGRATE | |
| `events-session` — `events.LoggedOut`, `events.StreamReplaced`, `events.Connected`, temporary-ban events | INTEGRATE | |
| `groups-read` — `GetJoinedGroups`, `GetGroupInfo`, group-metadata events | INTEGRATE | |
| `contacts-read` — the device store's contact cache (saved/address-book names) | INTEGRATE | |
| `pair-phone` — `Client.PairPhone`, pairing by one-time code instead of QR | OPT-OUT | not needed — D-01 and D-04 lock QR pairing as the only flow, in-app and via CLI |
| `media-download` — `Client.Download` / `DownloadAny` for attachment bytes | OPT-OUT | not needed yet — the hybrid data model keeps content in the source; the Phase 4 transcript renders attachment placeholders and stores no media |
| `send-message` — `SendMessage`, `SendReaction`, `MarkRead`, `SendChatPresence`, `RevokeMessage`, `EditMessage` | OPT-OUT | explicitly out of scope — `docs/plugin-contract.md`'s read-only-by-construction rule; enforced by Plan 08-02 Task 3's AST scan |
| `presence` — `SendPresence`, `SubscribePresence`, availability broadcast | OPT-OUT | explicitly out of scope — broadcasting presence changes what the user's contacts observe, which prohibition P1 in Plan 08-02 forbids |
| `groups-mutate` — `CreateGroup`, `JoinGroupWithLink`, `LeaveGroup`, `UpdateGroupParticipants`, `SetGroupName`/`Topic` | OPT-OUT | explicitly out of scope — read-only plugin; mutating the user's WhatsApp groups is never a source-plugin action |
| `newsletters` — `NewsletterSubscribe`/`Unsubscribe`/`GetNewsletterInfo` (WhatsApp Channels) | OPT-OUT | not needed — SRC-03 as widened by D-05 scopes matching to groups and 1:1 chats; channels are neither |
| `calls` — call-offer/terminate events | OPT-OUT | not needed — a call has no message content and no place in the conversation-day digest model |
| `app-state` — app-state sync for mute / pin / archive / starred | OPT-OUT | not needed yet — no topos surface consumes a chat's mute, pin or archive state |
| `privacy-settings` — `GetPrivacySettings`, `SetPrivacySettings` | OPT-OUT | explicitly out of scope — reading is unnecessary and writing would mutate the user's WhatsApp account settings, forbidden by the read-only contract |
| `logout` — `Client.Logout`, actively unpairing this device | OPT-OUT | not needed — de-linking is always initiated from the user's phone; the plugin only observes the resulting event. No purge/reset affordance is required this phase (CONTEXT.md Deferred Ideas) |
| `push-name` — the remote party's self-chosen profile/push name | OPT-OUT | explicitly out of scope — D-06's anti-injection rule: a contact must not be able to pull themselves into a webspace by renaming themselves, so this field is never read as a match candidate |
| `phone-number-match` — matching a chat by its E.164 number | OPT-OUT | explicitly out of scope — D-07 declines phone-number matching outright; saving the contact on the phone is the way to make a chat matchable |
