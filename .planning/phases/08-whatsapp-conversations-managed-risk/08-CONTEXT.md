# Phase 8: WhatsApp Conversations (Managed Risk) - Context

**Gathered:** 2026-08-10
**Status:** Ready for planning

<domain>
## Phase Boundary

The WhatsApp plugin (SRC-03): a `plugins/whatsapp` linked-device client built on `go.mau.fi/whatsmeow` (pure Go, no cgo) that pairs via QR code, holds a live connection for its subprocess lifetime, persists every captured message into its **own** local message store (WhatsApp is not a durable source), and surfaces matched chats in the stream as conversation-day digests using the Phase 4 chat rendering verbatim. De-link, ban, or session expiry surfaces as an explicit named plugin-health error while previously captured messages remain browsable and every other source is unaffected — a de-linked plugin returns a gRPC error from `Match`, never an empty success (the kernel wipes a source's rows on empty success; see `08-RESEARCH.md` Pitfall 1).

Per ROADMAP.md the hands-on spike (Task 1) remains mandatory — linking stability, first-link backfill volume, de-link/ban event taxonomy, deep-link mechanism — but the QR-pairing UX architecture question the research left open is now **decided** (D-01 below): the spike validates the in-app flow's mechanics rather than choosing between options.

Phase 07.1's standing rule applies: all UI work here extends the Playwright e2e suite (`web/e2e/specs/`) as part of definition of done.

</domain>

<decisions>
## Implementation Decisions

### QR pairing & linking UX
- **D-01:** **In-app QR flow (research's Option B), not the CLI-only default.** A new kernel HTTP surface runs the plugin binary in a dedicated link mode as a raw subprocess (outside the `go-plugin` gRPC handshake — this is not a `SourcePlugin` RPC, so the locked four-RPC contract is untouched) and relays the rotating QR code to the browser for display. Two hard consequences the user accepts: (a) `08-UI-SPEC.md` must be **amended** with the QR panel's component contract (QR image sizing, rotation/expiry countdown copy, scan-success transition) before the wave implementing it is planned — the approved UI-SPEC explicitly does not design this surface; (b) the QR-to-image encoder package (e.g. `skip2/go-qrcode`) is **unaudited** — run the Go-ecosystem package-legitimacy check on whichever encoder is selected before adding it. — **Reversibility:** costly — a new kernel endpoint, a second subprocess-lifecycle path distinct from `pluginhost`, and a frontend QR component; falling back to CLI-only linking would abandon that surface (though D-04 keeps the CLI path alive as insurance).
- **D-02:** **The QR panel appears inline during Add-Source.** When Step 1's trial-launch reports "not linked", the modal offers the QR panel right there — configure → scan → saved and syncing in one continuous flow. The UI-SPEC's "saved but not yet linked" state (E5 partial) remains valid for users who cancel out mid-link; it is no longer the *only* path.
- **D-03:** **Re-linking lives in the source chip's menu as "Re-link…"** (extends the Phase 7 D-12 edit popover), opening the same QR component in a small dialog. The not-linked/de-linked/expired health-tooltip copy should point the user at it. One QR component, two entry points (Add-Source flow + chip menu).
- **D-04:** **The standalone CLI link mode ships too** (same plugin binary, a link flag/subcommand rendering ASCII QR via `qrterminal`) as a fallback/recovery path if the in-app flow breaks — managed-risk insurance; the in-app flow is the primary UX.
- **Hard requirement carried from research:** the link-mode subprocess and the regular `pluginhost`-launched instance must never both hold the same whatsmeow `sqlstore` open at once — mutual exclusion is part of D-01's design, mechanism at Claude's discretion.

### Matching scope
- **D-05:** **Groups AND 1:1 chats both match** — the user explicitly widened SRC-03's literal "matches on group names" wording (research Assumption A5 flagged this reading as unconfirmed; now resolved). The match vocabulary gains a second field for 1:1 contacts alongside `groups`. The UI-SPEC's match-field table (single `Groups` row) is superseded on this point — fold into the same D-01 amendment. SRC-03's requirement text should be treated as extended by this decision.
- **D-06:** **1:1 chats match against the user's own saved contact name only** (the phone's address-book name as synced to the linked device) — **never the contact's self-chosen push/profile name**. Same anti-injection rationale as Signal's Phase 4 D-06, carried over verbatim: a contact must not be able to pull themselves into a webspace by renaming themselves.
- **D-07:** **Chats with unsaved contacts are not matchable at all.** No phone-number matching rule. Saving the contact on the phone is the way to make them matchable.

### Claude's Discretion
- **History depth on first link** (area offered, not selected for discussion): accept whatsmeow's default backfill; observe the real volume during the spike per ROADMAP's note. Restate Signal's D-08 as "digest everything the local message store has captured" — the plugin does not control the backfill window's origin.
- **Message-store lifecycle** (area offered, not selected): research defaults stand — the plugin's own message DB is kept across de-link/re-link and never deleted on any failure state (success criterion 4); separate DB file from whatsmeow's `sqlstore`; default path under the instance's configured `path`. No purge/reset affordance required this phase.
- Link-endpoint transport (SSE vs short-poll), endpoint shape per `docs/api.md` conventions, and the QR-relay subprocess protocol.
- QR encoder package selection (subject to the D-01 legitimacy audit) and match-vocabulary field naming for the 1:1 field.
- Deep-link mechanism for "Open in WhatsApp" — spike verifies the URI scheme hands-on; `LINK_FIDELITY_CONVERSATION_ONLY` is fixed regardless.
- Exact whatsmeow commit to pin (untagged upstream — dated pseudo-version pin comment per `plugins/signal/go.mod` precedent; SUS-flagged in research, needs the plan's explicit acknowledgment).
- Health `last_error` wording per cause — UI-SPEC's templates are a non-binding starting point; the one binding rule is the copy must never imply captured data was lost, and (per D-03) should point at the Re-link affordance where re-linking is the fix.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope & requirements
- `.planning/ROADMAP.md` — Phase 8 goal, 4 success criteria, notes (mandatory spike list; managed-risk framing; e2e standing rule).
- `.planning/REQUIREMENTS.md` — SRC-03 (single requirement; D-05 above widens its matching scope to include 1:1 chats).
- `.planning/PROJECT.md` — constraints (read-only, privacy, all-data-local) and Key Decisions table.

### Phase 8 artifacts (both exist already — read before planning)
- `.planning/phases/08-whatsapp-conversations-managed-risk/08-RESEARCH.md` — the architecture this phase follows: long-lived in-process whatsmeow client, own message store, Match-error-vs-empty-success (Pitfall 1 governs criterion 4), whatsmeow pin strategy, spike scope. Valid until ~2026-08-24.
- `.planning/phases/08-whatsapp-conversations-managed-risk/08-UI-SPEC.md` — approved UI contract (chat-transcript reuse, health taxonomy, Add-Source row, copy). **Requires amendment before the QR-panel wave is planned:** the in-app QR component contract (D-01) and the widened match-field table (D-05).

### Prior locked decisions this phase builds on
- `.planning/phases/04-signal-conversations/04-CONTEXT.md` — D-01–D-04 digest shape (reused verbatim), D-06 contact-name anti-injection rule (carried into this phase's D-06), D-08 backfill framing (restated per research).
- `.planning/phases/05-source-instances-per-type-matching/05-CONTEXT.md` — per-instance typed match contract the plugin's vocabulary must implement; instance identity rules.
- `.planning/phases/07-webspace-builder-ui/07-CONTEXT.md` — D-11 (Add-Source two-step modal this phase's QR panel extends), D-12 (chip menu the "Re-link…" entry joins), D-06/D-07 (hot-apply/reconcile the new instance rides).

### Published contracts (extend carefully)
- `docs/plugin-contract.md` — the locked four-RPC, read-only contract; the link mode must stay outside it (D-01).
- `docs/api.md` — HTTP envelope conventions for the new link endpoint.
- `docs/testing.md` — e2e harness map; standing rule that UI work lands with specs in `web/e2e/specs/`.
- `proto/topos/v1/` — match-vocabulary wire shape (`match_fields`); no proto change expected.
- `config.example.toml` — gains the WhatsApp source-block example (local-path source, Signal precedent).

### Technology stack (locked)
- `.claude/CLAUDE.md` — whatsmeow as the only viable WhatsApp route; kernel stays cgo-free (this plugin is pure Go, unlike Signal).

</canonical_refs>

<code_context>
## Existing Code Insights

(Research performed a same-day, code-verified scout; findings adopted directly.)

### Reusable Assets
- `plugins/signal/` — `digest.go` (conversation-day digest assembly), `render.go` (transcript markup within the locked class-token vocabulary), `plugin.go` (`openGuarded` → `codes.Unavailable` pattern this plugin's not-linked/de-linked states must mirror).
- `kernel/httpapi/rendition.go` — `CONTENT_SHAPE_CHAT_TRANSCRIPT` sanitize/wrap/theme policy, reused with zero new CSS.
- Phase 7 builder UI — `AddSourceModal` two-step flow (QR panel slots into Step 1's not-linked outcome), `SourceChip.svelte` menu (gains "Re-link…").
- `web/e2e/` — hermetic Playwright harness; `mockstrict` plugin shows the e2e-only plugin pattern if a mock link-mode is useful for specs.

### Established Patterns
- Plugin subprocesses are launched once and live across all RPCs (`kernel/pluginhost/host.go`) — the persistent whatsmeow connection fits the existing lifecycle with no contract change.
- `kernel/correlate/correlate.go` (~105–110): `Match` error → previous rows preserved; empty success → rows wiped. The entire criterion-4 guarantee hangs on the plugin using the error channel correctly.
- Read-only enforced by test per plugin — this plugin's variant is behavioral: an AST scan asserting no send-capable whatsmeow `Client` method is ever referenced outside tests.
- Local-path source config (`Source.Path`, Signal precedent) — no base_url/token; session keys live only in whatsmeow's own store, never in config/env.

### Integration Points
- `kernel/httpapi` — the new link endpoint (first kernel surface that spawns a plugin binary outside `pluginhost`); mutual exclusion with the running instance is a design requirement.
- `plugins/whatsapp/` — new pure-Go `go.work` member; fits existing `make build`/`test-portable` (no cgo targets needed).
- `web/src/lib/components/` — QR panel component (new, UI-SPEC amendment required), AddSourceModal Step 1 branch, SourceChip menu entry.

</code_context>

<specifics>
## Specific Ideas

- User chose the polished in-app pairing flow over the research's lower-risk CLI default, with eyes open to the added kernel/frontend surface — but kept the CLI path as a fallback ("managed-risk insurance"). Build one QR component used from both entry points (Add-Source inline + chip-menu Re-link dialog).
- The selected mockup: QR panel inside the Connect WhatsApp modal alongside the Display name / Local path fields, with a "Scan with your phone to link" instruction and a refresh countdown ("Refreshes in 0:18").

</specifics>

<deferred>
## Deferred Ideas

- Phone-number (E.164) matching for chats with unsaved contacts — explicitly declined (D-07); revisit only if real usage surfaces a need (e.g. tradespeople deliberately kept out of the address book).
- Message-store purge/reset affordance — not required this phase; the store is keep-forever by design.

### Reviewed Todos (not folded)
- **Signal schema-version verify-and-accept tooling** (`.planning/todos/pending/2026-08-05-signal-schema-version-verify-and-accept-tooling.md`) — matched at 0.6 on keyword noise (signal/desktop/user); Signal-specific maintenance tooling, unrelated to WhatsApp. Declined here (also declined in Phase 7); stays pending.

</deferred>

---

*Phase: 8-WhatsApp Conversations (Managed Risk)*
*Context gathered: 2026-08-10*
