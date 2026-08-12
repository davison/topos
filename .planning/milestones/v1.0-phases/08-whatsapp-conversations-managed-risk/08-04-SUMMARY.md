---
phase: 08-whatsapp-conversations-managed-risk
plan: 04
subsystem: ui
tags: [svelte5, whatsapp, qr-pairing, playwright, add-source-modal, source-chip]

requires:
  - phase: 08-whatsapp-conversations-managed-risk (Plan 08-02)
    provides: "The five-state healthState taxonomy and widened two-field (groups/contacts) match vocabulary the Match settings step renders against"
  - phase: 08-whatsapp-conversations-managed-risk (Plan 08-03)
    provides: "POST/GET/DELETE /api/config/whatsapp-link's exact request/response shapes (docs/api.md), and 08-UI-SPEC.md's Amendment section (QR sizing, copy, five-state coverage, two entry points) this plan builds directly against"
provides:
  - "web/src/lib/components/QRPanel.svelte: the single QR pairing panel — polls on a session-validity-derived, floor-clamped cadence, covers loading/qr/error/expired/success, cancels on both explicit Cancel and unmount (T-08-10)"
  - "AddSourceModal.svelte's new 'link' step (D-01/D-02): a WhatsApp trial-launch success renders QRPanel inline inside the same Step 1 dialog; a declined QR opportunity (linkOffered) still reaches Step 2 and saves"
  - "SourceChip.svelte's Re-link… menu entry (D-03), gated on source_type === 'whatsapp', and RelinkModal.svelte — the same QRPanel in a small dialog opened from the chip menu"
  - "web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts: eight hermetic Playwright tests covering both entry points, all five states, rotation, cancellation, the Re-link gate, and the not-linked-is-not-a-failure rule"
affects: []

actuals:
  tokens: 21655
  tasks: 3
  commits: 4

tech-stack:
  added: []
  patterns:
    - "linkOffered flag (AddSourceModal.svelte): the WhatsApp QR opportunity is offered once per modal session — a first trial-launch success routes to 'link', a later one (reached after the panel's own cancelled callback returns to 'connect') routes straight to 'match' instead of re-showing the panel, so cancelling remains a real decision to move on rather than a gate blocking Step 2"
    - "retireSession as the one cancel-session call site (QRPanel.svelte): both the explicit Cancel button and onDestroy (unmount) route through it, so T-08-10's mitigation (never leave an orphaned link subprocess) is provably a single code path, not two independently-maintained ones"
    - "Route-intercepted plugin discovery for an e2e-excluded plugin type: since topos-plugin-whatsapp is structurally excluded from the hermetic harness's closed plugin set (docs/testing.md), the new spec intercepts GET /api/config/plugin-types, POST /api/config/describe-plugin, and the three whatsapp-link routes at the Playwright route layer rather than exercising a real plugin — the same technique 07.1's specs already use for kernel-shaped fakes, extended to a whole undiscoverable plugin type"

key-files:
  created:
    - web/src/lib/components/QRPanel.svelte
    - web/src/lib/components/qr-panel.test.ts
    - web/src/lib/components/RelinkModal.svelte
    - web/src/lib/components/relink.test.ts
    - web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts
  modified:
    - web/src/lib/api.ts
    - web/src/lib/plugin-fields.ts
    - web/src/lib/plugin-fields.test.ts
    - web/src/lib/components/AddSourceModal.svelte
    - web/src/lib/components/add-source.test.ts
    - web/src/lib/components/SourceChip.svelte
    - web/src/lib/components/WebspaceHeader.svelte
    - web/src/lib/components/chip-edit-menu.test.ts
    - web/src/routes/w/[webspace]/+page.svelte
    - docs/testing.md

key-decisions:
  - "Describe carries no linked-status field by design (confirmed against kernel/pluginhost/host.go's DescribeInfo and kernel/httpapi/config.go's describePluginResponse — both fixed at source_type/plugin_display_name/match_vocabulary). So Step 1's WhatsApp branch cannot literally 'check whether the instance reports itself linked' as the plan's prose puts it; the implemented signal is simply 'trial-launch succeeded for the WhatsApp plugin type' — the panel is then offered unconditionally, and an already-linked device's own link session resolves straight to 'paired' with no QR ever shown (link.go's runLinkCore reconnect-and-confirm branch), which reads as the same 'opportunity, not a gate' UX the plan describes."
  - "linkOffered flag added beyond the plan's literal action text (see tech-stack pattern above) — the plan's own Task 3 acceptance criteria require that cancelling the QR panel still let the flow reach match settings and save, which a strict first-cancel-returns-to-connect-and-nothing-else design cannot satisfy (a second 'Next' click would just re-show the same panel forever)."
  - "QRPanel renders ConnectionForm (still visible, editable) above itself in the 'link' step, per 08-UI-SPEC.md's literal 'renders inline below the connection fields' wording — not a swap-out of Step 1's own fields."
  - "e2e spec's case 8 final save writes config.toml directly (the fixture's own smol-toml writer) rather than round-tripping through a real PUT /api/config — kernel/pluginhost/host.go's launch() os.Stat()s the plugin binary before ever calling Reconcile, so a genuine save of a topos-plugin-whatsapp source would apply-fail (500) in this harness, which structurally excludes that binary by design. Verified live: the real PUT does write config.toml successfully (Store.Save has no plugin-existence check) before Apply's Reconcile fails — this is real, documented kernel behaviour (T-07-11) against an artifact of the harness, not a defect in the shipped SPA."

requirements-completed: [SRC-03]

coverage:
  - id: D1
    description: "QRPanel.svelte covers all five states (loading/qr/error/expired/success) with the exact instruction line and a session-validity-driven, floor-clamped countdown, and cancels the link session on both explicit Cancel and unmount"
    requirement: "SRC-03"
    verification:
      - kind: unit
        ref: "web/src/lib/components/qr-panel.test.ts (11 tests)"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts, cases 1/2/4/5/6"
        status: pass
    human_judgment: false
  - id: D2
    description: "AddSourceModal.svelte's WhatsApp branch renders QRPanel inline inside the existing Step 1 dialog (never a new Dialog) on trial-launch success, without setting the describe-failure flag or revealing Save anyway"
    requirement: "SRC-03"
    verification:
      - kind: unit
        ref: "web/src/lib/components/add-source.test.ts, 'WhatsApp not-linked branch (D-01/D-02)' describe block"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts, case 1 and case 8"
        status: pass
    human_judgment: false
  - id: D3
    description: "A declined QR opportunity (cancelled once) still reaches the match step and saves a fully configured, functionally-inert WhatsApp instance — the E5 backstop evidence"
    requirement: "SRC-03"
    verification:
      - kind: e2e
        ref: "web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts, case 8 (config outcome asserted via readConfigToml)"
        status: pass
    human_judgment: false
  - id: D4
    description: "SourceChip.svelte offers a Re-link… menu entry gated on source_type === 'whatsapp', opening RelinkModal (the same QRPanel) in a small dialog; a non-WhatsApp chip never shows the entry"
    requirement: "SRC-03"
    verification:
      - kind: unit
        ref: "web/src/lib/components/chip-edit-menu.test.ts, 'Re-link… entry: guarded on the WhatsApp source type' describe block"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts, case 7"
        status: pass
    human_judgment: false
  - id: D5
    description: "The Add-Source picker offers a WhatsApp row with Display Name and a required, pre-seeded Local Path matching config.example.toml's [sources.whatsapp] default"
    requirement: "SRC-03"
    verification:
      - kind: unit
        ref: "web/src/lib/plugin-fields.test.ts, WhatsApp-row assertions"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts, openWhatsAppConnectStep's pre-fill assertion"
        status: pass
    human_judgment: false
  - id: D6
    description: "The Playwright suite covers the WhatsApp QR flow hermetically — the link endpoint, plugin discovery, and the trial launch are intercepted at the route layer, so the suite never reaches WhatsApp's servers, a real account, or a real credential"
    requirement: "SRC-03"
    verification:
      - kind: e2e
        ref: "web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts (8/8 passing), make e2e (34/34 passing)"
        status: pass
    human_judgment: false
  - id: D7
    description: "UI-SPEC E5's explicit state categories (empty/loading/error/overflow/long-text) continue to hold — inherited Phase 7 two-step-modal machinery, unmodified by this plan"
    verification: []
    human_judgment: true
    rationale: "Not exercised by a new test this plan added — these categories are explicitly declared as pre-existing, unchanged Phase 7 behavior in both the plan's own must_haves and 08-UI-SPEC.md's UI Considerations table; no code in this plan's files_modified list touches the empty/loading/error/overflow/long-text branches themselves, only the new WhatsApp-specific 'link' branch alongside them."

duration: ~45min
completed: 2026-08-10
status: complete
---

# Phase 8 Plan 4: WhatsApp In-App Pairing (UI Half) Summary

**QRPanel.svelte — one QR pairing component reused from two entry points (inline in the Add-Source Step 1 dialog, and the source chip's Re-link… menu), plus eight hermetic Playwright tests proving the whole flow, closing the phase's user-visible promise: "link WhatsApp by scanning a QR code."**

## Performance

- **Duration:** ~45 min
- **Started:** 2026-08-10 (worktree agent-a7d9334ff8b89b42e)
- **Completed:** 2026-08-10
- **Tasks:** 3 of 3 complete
- **Files modified:** 15 (5 created, 10 modified)

## Accomplishments

- `QRPanel.svelte` (new): the single QR pairing panel — polls on a cadence derived from the session's own reported validity (floor-clamped to prevent a request storm), covers all five states the UI-SPEC amendment defines (loading/qr/error/expired/success), and cancels the session on both an explicit Cancel click and component unmount (T-08-10's DoS mitigation), through one shared `retireSession` call site
- `api.ts`: typed `startWhatsAppLink`/`pollWhatsAppLink`/`cancelWhatsAppLink` clients matching `docs/api.md`'s field names verbatim, plus a new `deleteJSON` helper mirroring the existing get/post/put envelope handling for the kernel's one DELETE route
- `plugin-fields.ts`: the `topos-plugin-whatsapp` connection-field row (Display Name + a required, pre-seeded Local Path matching `config.example.toml`'s default + the standard Sync Interval Override) and its "WhatsApp" label
- `AddSourceModal.svelte`: a new `link` step between `connect` and `match` — a WhatsApp trial-launch success renders `QRPanel` inline inside the SAME Step 1 dialog, beside the already-entered connection fields, never setting the describe-failure flag or revealing Save anyway. A `linkOffered` flag (a deviation from the plan's literal text, documented below) makes a declined QR opportunity still reach Step 2 and save
- `SourceChip.svelte`/`WebspaceHeader.svelte`: a "Re-link…" menu entry, rendered only for a `source_type === 'whatsapp'` instance, routed through the existing `onedit` callback and stopPropagation discipline
- `RelinkModal.svelte` (new): a small Dialog (no `max-w-lg` override) wrapping `QRPanel` — no connection or match fields, just the panel
- `+page.svelte`: tracks the relinking instance in its own `relinkInstance` state (deliberately distinct from `editInstance`/`editMode`), branches `handleChipEdit` on `'relink'` before the describe path, and renders `RelinkModal` keyed on the relinking instance
- `web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts` (new): eight hermetic Playwright tests — populated QR, rotation, scan success, expiry, error, cancel-releases-session (asserted on the intercepted DELETE request, not merely the UI closing), the Re-link entry-point gate, and the E5 not-linked-is-not-a-failure evidence (config outcome asserted via `readConfigToml`)
- All automated verification passes: `cd web && npm run check` (0 errors), `cd web && npm run test` (642 tests), `make e2e` (34/34, including the new spec), `make test-portable` (all 8 Go modules)

## Task Commits

1. **Task 1: The QR panel, and WhatsApp inline in the Add-Source flow (D-01/D-02)**
   - `80cb143` (feat) — `api.ts` clients, `plugin-fields.ts` row, `QRPanel.svelte`, `AddSourceModal.svelte`'s `link` step, and the corresponding test files
2. **Task 2: Re-link… from the source chip's menu (D-03)**
   - `9c50c4f` (feat) — `SourceChip.svelte`/`WebspaceHeader.svelte` widened `onedit`, `RelinkModal.svelte`, `+page.svelte`'s `relinkInstance` wiring, and the corresponding test files
3. **(fix, found while designing Task 3's hermetic coverage — see Deviations)**
   - `fdfd50e` (fix) — `linkOffered` flag: a declined QR opportunity now still reaches match settings and saves
4. **Task 3: Hermetic Playwright coverage of the QR flow (Phase 07.1 standing rule)**
   - `8bd073a` (test) — `uat-08-whatsapp-qr-link.spec.ts`, `docs/testing.md`

**Plan metadata:** this commit (docs: finalize SUMMARY — plan complete)

## Files Created/Modified

- `web/src/lib/components/QRPanel.svelte` (new) — the single QR pairing panel, both entry points' shared component
- `web/src/lib/components/qr-panel.test.ts` (new) — structural coverage of all five states and the cancel/unmount-both-cancel invariant
- `web/src/lib/components/RelinkModal.svelte` (new) — the chip menu's Re-link… entry point, a small Dialog wrapping QRPanel
- `web/src/lib/components/relink.test.ts` (new) — structural coverage of RelinkModal and the route's relink wiring
- `web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts` (new) — eight hermetic Playwright tests over the whole QR flow
- `web/src/lib/api.ts` — `WhatsAppLinkState`/`WhatsAppLinkSession`/`StartWhatsAppLinkRequest`, `startWhatsAppLink`/`pollWhatsAppLink`/`cancelWhatsAppLink`, `deleteJSON`
- `web/src/lib/plugin-fields.ts` — `topos-plugin-whatsapp` row, its "WhatsApp" label, `WHATSAPP_PLUGIN_BINARY`/`WHATSAPP_SOURCE_TYPE` exported constants
- `web/src/lib/plugin-fields.test.ts` — WhatsApp-row assertions (required path, seeded default, label, derived-required-flags table truth)
- `web/src/lib/components/AddSourceModal.svelte` — the `link` step, `handleLinkPaired`/`handleLinkCancelled`, `linkOffered`
- `web/src/lib/components/add-source.test.ts` — WhatsApp not-linked branch assertions, the widened Dialog `open` binding marker
- `web/src/lib/components/SourceChip.svelte` — widened `onedit` kind union, the Re-link… `DropdownMenuItem`, `isWhatsApp` derivation
- `web/src/lib/components/WebspaceHeader.svelte` — widened `onedit` prop type
- `web/src/lib/components/chip-edit-menu.test.ts` — updated to the four-entry-plus-separator menu shape, new Re-link… gate assertions
- `web/src/routes/w/[webspace]/+page.svelte` — `relinkInstance` state, `handleChipEdit`'s `'relink'` branch, `RelinkModal` render block
- `docs/testing.md` — the new spec's own section, and the third "what stays manual" item (a real WhatsApp account pairing)

## Decisions Made

See `key-decisions` in frontmatter above. Summary: Describe carries no linked-status field by design, so the WhatsApp branch offers the QR panel unconditionally on trial-launch success rather than literally checking a "linked" flag that doesn't exist on the wire; `linkOffered` was added to satisfy Task 3's own acceptance criteria (cancelling must still reach Step 2); `QRPanel` renders inline below the still-visible `ConnectionForm`, matching the UI-SPEC's literal wording; and the e2e spec's one real-write case (case 8) writes `config.toml` directly rather than round-tripping through a real, un-launchable plugin subprocess, since the harness structurally excludes `topos-plugin-whatsapp` by design.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Added `linkOffered` so a declined QR opportunity still reaches match settings and saves**
- **Found during:** Designing Task 3's own acceptance criteria (case 8: "cancelling out still reaches the match step and saves")
- **Issue:** Task 1's literal action text ("branch on whether the described instance reports itself linked... move to the link step") gave no way back to Step 2 once a user cancelled the QR panel — a second "Next" click just re-showed the same panel, contradicting both Task 3's own acceptance criteria and the UI-SPEC amendment's framing that Step 1 and Step 2 can both succeed while the device stays unpaired
- **Fix:** Added a `linkOffered` flag: the first trial-launch success still routes to `'link'` (the QR opportunity, offered once); once the panel has been shown and cancelled, a later success routes straight to `'match'`
- **Files modified:** `web/src/lib/components/AddSourceModal.svelte`, `web/src/lib/components/add-source.test.ts`
- **Committed in:** `fdfd50e`

**2. [Rule 4-adjacent — a genuinely underspecified contract, resolved by reading the source] Describe carries no linked-status field**
- **Found during:** Task 1, reading `kernel/pluginhost/host.go`'s `DescribeInfo` and `kernel/httpapi/config.go`'s `describePluginResponse` to implement the plan's "branch on whether the described instance reports itself linked" instruction
- **Issue:** No such field exists on the wire — `Describe`'s response is fixed at `source_type`/`plugin_display_name`/`match_vocabulary`, confirmed by reading the actual kernel source rather than assuming the plan's prose implied an undocumented field
- **Resolution:** Implemented the only signal that genuinely exists — trial-launch success for the WhatsApp plugin type — and verified (via `plugins/whatsapp/link.go`'s `runLinkCore`) that an already-linked device's own link session resolves straight to `paired` with no QR ever shown, which produces the same "opportunity, not obstacle" UX the plan describes without inventing an API field
- **Files modified:** none beyond the already-planned `AddSourceModal.svelte`/`QRPanel.svelte` work
- **Committed in:** `80cb143` (no separate commit; documented here for the audit trail)

---

**Total deviations:** 2 (1 auto-fixed bug found via the plan's own acceptance criteria, 1 documented contract-reading resolution)
**Impact on plan:** Both necessary for the plan's own must_haves/acceptance criteria to hold. No scope creep — every change was a direct, mechanical consequence of implementing the plan's own stated intent against the real, already-shipped kernel contract.

## Issues Encountered

- `kernel/pluginhost/host.go`'s `launch()` `os.Stat()`s a plugin binary before `Host.Reconcile` will ever launch it, so a real `PUT /api/config` save of a `topos-plugin-whatsapp` source in this hermetic harness (which structurally excludes that binary, matching every other real-source plugin) would apply-fail with `500 apply_failed` even though `config.Store.Save` itself succeeds and writes the file. Verified this live before designing around it. Resolved by having case 8's own final save write `config.toml` directly via the fixture's own smol-toml writer — proving the SPA sends the correct structural document (the real defect surface that case guards) without requiring a real, launchable WhatsApp binary in the harness. Documented in the spec's own header comment and `docs/testing.md`.
- No Playwright browser was pre-installed in this environment; `make e2e` downloaded Chromium (BEWARE: unsupported-OS fallback build warning, harmless) before running — first run took longer than subsequent runs as a result.

## User Setup Required

None. This plan adds no new external service, credential, or environment variable — it is pure SPA code plus a hermetic browser spec.

## Next Phase Readiness

- **Ready to ship.** This is the last plan in Phase 8 — the phase's user-visible promise ("link WhatsApp by scanning a QR code") is now true end to end: a user picks "New WhatsApp…", fills in a path, scans the QR right there in Step 1, and lands on match settings; a broken session is recoverable later from the chip's Re-link… entry.
- **No open blockers.** All three tasks' acceptance criteria pass; `cd web && npm run check` is 0 errors; `cd web && npm run test` is 642/642; `make e2e` is 34/34 (including the new spec, no pre-existing spec regressed); `make test-portable` is green across all 8 Go modules.
- **Known gap, not a defect:** a real pairing against a real WhatsApp account (requiring an actual phone) stays a manual activity, covered by Plan 08-01 Task 3's hands-on spike rather than an automated gate — recorded explicitly in `docs/testing.md`'s "What stays manual, and why" section, per the standing rule that a manual check has to be remembered to be honored, and an unrecorded one usually isn't.

## Self-Check: PASSED

- All 5 declared new files (`QRPanel.svelte`, `qr-panel.test.ts`, `RelinkModal.svelte`, `relink.test.ts`, `uat-08-whatsapp-qr-link.spec.ts`) confirmed present on disk.
- All 4 commit hashes (`80cb143`, `9c50c4f`, `fdfd50e`, `8bd073a`) confirmed present in `git log`.
- `cd web && npm run check` (0 errors), `cd web && npm run test` (642/642), `make e2e` (34/34), and `make test-portable` (all 8 Go modules) re-verified passing before this SUMMARY was written.

---
*Phase: 08-whatsapp-conversations-managed-risk*
*Completed: 2026-08-10*
