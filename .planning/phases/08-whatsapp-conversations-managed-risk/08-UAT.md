---
status: complete
phase: 08-whatsapp-conversations-managed-risk
source: [08-01-SUMMARY.md, 08-02-SUMMARY.md, 08-03-SUMMARY.md, 08-04-SUMMARY.md]
started: 2026-08-10T00:00:00Z
updated: 2026-08-10T00:00:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Real-device WhatsApp link end-to-end
expected: A real WhatsApp account links via the QR flow, the linked session survives a kernel restart (reconnects with no second QR), a matching group's digest appears in a real webspace stream, and opening it renders the Phase 4 chat transcript.
context: Requires a real WhatsApp account with an active phone and a real kernel run — inherently outside the automated execution environment. Was verified live in three rounds on 2026-08-10 during execution (616 messages/134 chats backfilled, restart reconnected without a second QR, /api/sources reachable:true); confirm it still holds from your perspective.
coverage_id: 08-01/D2
result: issue
reported: "After linking using the QR code, phone shows successful link. WhatsApp modal in topos remains on screen with the refresh counter dwindling. No connection from the topos side after cancelling the dialog"
severity: major

### 2. Phase 7 modal state categories still hold
expected: The Add Source modal's inherited state categories (empty/loading/error/overflow/long-text) still behave as in Phase 7 — the new WhatsApp link branch sits alongside them without regressing any of them.
context: Declared pre-existing, unchanged Phase 7 behavior; no code in plan 08-04's modified files touches those branches, only the new WhatsApp-specific link branch. Human confirmation requested because no new test exercises them.
coverage_id: 08-04/D7
result: pass

### 3. WhatsApp plugin builds pure-Go and its suite passes
expected: plugins/whatsapp module builds pure-Go (CGO_ENABLED=0), joins go.work, and its own test suite passes
result: pass
source: automated
coverage_id: 08-01/D1

### 4. Health states surface distinct, honest last_error
expected: Five named health states (not-linked, linked, de-linked, banned, session-expired, plus stream-replaced) each surface a distinct, honest last_error that never implies data loss
result: pass
source: automated
coverage_id: 08-02/D1

### 5. Non-healthy Match returns Unavailable, never empty success
expected: Every non-healthy state makes Match return codes.Unavailable, never an empty success, even with zero keywords
result: pass
source: automated
coverage_id: 08-02/D2

### 6. Failure events never delete the message store
expected: No failure event (LoggedOut/TemporaryBan/ConnectFailure/StreamReplaced) deletes, truncates, or empties this plugin's own message store
result: pass
source: automated
coverage_id: 08-02/D3

### 7. 1:1 matching on saved contact name only
expected: 1:1 chats match on the user's own saved address-book contact name only; unsaved contacts and remote-supplied push names are never matchable (D-05/D-06/D-07)
result: pass
source: automated
coverage_id: 08-02/D4

### 8. Additive contact_name migration preserves rows
expected: A store created by Plan 08-01 (no contact_name column) opens successfully and preserves its existing rows after the additive migration
result: pass
source: automated
coverage_id: 08-02/D5

### 9. AST scans enforce read-only and no-egress boundaries
expected: Non-vacuous, negative-controlled AST scans enforce the read-only boundary (no send-capable whatsmeow Client selector) and the no-egress boundary (no self-constructed net/http client, no non-allowlisted host literal)
result: pass
source: automated
coverage_id: 08-02/D6

### 10. UI-SPEC amendment recorded with provenance
expected: 08-UI-SPEC.md carries the dated QR-panel/match-table amendment with supersession pointers and an intact Checker Sign-Off block
result: pass
source: automated
coverage_id: 08-03/D1

### 11. QR encoder audited before entering go.mod
expected: The QR-to-PNG encoder is audited via the Go-ecosystem legitimacy protocol; verdict recorded as a dated go.mod comment and a row in 08-RESEARCH.md's audit table; root go.mod unchanged
result: pass
source: automated
coverage_id: 08-03/D2

### 12. -link-json machine-readable link mode
expected: plugins/whatsapp gains a -link-json mode sharing one QR-channel core with ASCII -link; every linkEvent marshals to one newline-free JSON line; raw QR payload never appears in any emitted event or log; store-lock failure emits a distinct error code; -link and -link-json are mutually exclusive
result: pass
source: automated
coverage_id: 08-03/D3

### 13. Kernel link endpoints with allowlist-before-spawn
expected: POST/GET/DELETE /api/config/whatsapp-link: discovered-binary allowlist check runs before any subprocess spawns; poll/cancel/reap/suspend-resume semantics hold; store-in-use maps to a distinct code; SDK contract unchanged; docs/api.md updated
result: pass
source: automated
coverage_id: 08-03/D4

### 14. QRPanel covers all five states and cancels on unmount
expected: QRPanel.svelte covers loading/qr/error/expired/success with the exact instruction line and a floor-clamped countdown, and cancels the link session on both explicit Cancel and unmount
result: pass
source: automated
coverage_id: 08-04/D1

### 15. WhatsApp branch renders QRPanel inline in Step 1 dialog
expected: AddSourceModal's WhatsApp branch renders QRPanel inline inside the existing Step 1 dialog on trial-launch success, without the describe-failure flag or Save anyway
result: pass
source: automated
coverage_id: 08-04/D2

### 16. Declined QR still saves a configured, inert instance
expected: A declined QR opportunity (cancelled once) still reaches the match step and saves a fully configured, functionally-inert WhatsApp instance
result: pass
source: automated
coverage_id: 08-04/D3

### 17. Re-link menu entry gated on WhatsApp source type
expected: SourceChip offers a Re-link… menu entry only when source_type === 'whatsapp', opening RelinkModal (same QRPanel) in a small dialog
result: pass
source: automated
coverage_id: 08-04/D4

### 18. Add-Source picker WhatsApp row with pre-seeded path
expected: The Add-Source picker offers a WhatsApp row with Display Name and a required, pre-seeded Local Path matching config.example.toml's default
result: pass
source: automated
coverage_id: 08-04/D5

### 19. Hermetic Playwright coverage of the QR flow
expected: The Playwright suite covers the WhatsApp QR flow hermetically via route-layer interception — never reaching WhatsApp's servers, a real account, or a real credential (8/8 spec, 34/34 suite)
result: pass
source: automated
coverage_id: 08-04/D6

## Summary

total: 19
passed: 18
issues: 1
pending: 0
skipped: 0

## Gaps

- gap_id: G-08-1
  truth: "After a successful phone-side QR pairing, the topos QR panel transitions to the success state and the linked session is usable — the source connects from the topos side"
  status: failed
  reason: "User reported: After linking using the QR code, phone shows successful link. WhatsApp modal in topos remains on screen with the refresh counter dwindling. No connection from the topos side after cancelling the dialog"
  severity: major
  test: 1
  artifacts: []  # Filled by diagnosis
  missing: []    # Filled by diagnosis
