# Phase 8: WhatsApp Conversations (Managed Risk) - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-10
**Phase:** 8-whatsapp-conversations-managed-risk
**Areas discussed:** QR pairing & re-link UX, Matching scope (groups vs 1:1)

---

## Pre-discussion: todo cross-reference

| Option | Description | Selected |
|--------|-------------|----------|
| Leave it deferred (Recommended) | Keyword-noise match — Signal tooling has nothing to do with the WhatsApp plugin. Stays in the pending todos. | ✓ |
| Fold it in | Add the Signal schema verify-and-accept tooling to Phase 8's scope. | |

**User's choice:** Leave it deferred
**Notes:** "Signal schema-version verify-and-accept tooling" matched at 0.6 on keyword noise; Phase 7 also declined it (at 0.2).

---

## Area selection

Offered: QR pairing & re-link UX / Matching scope: groups vs 1:1 / History depth on first link / Message-store lifecycle.
**Selected:** QR pairing & re-link UX, Matching scope. The two unselected areas fell to Claude's discretion on research defaults.

---

## QR pairing & re-link UX

### Q1 — How should the initial WhatsApp QR pairing work?

| Option | Description | Selected |
|--------|-------------|----------|
| CLI linking step (Recommended) | One-time terminal command renders the QR as ASCII; zero new kernel/UI surface. Research and UI-SPEC default. | |
| In-app QR flow | New kernel HTTP endpoint spawns the plugin in link mode and streams the rotating QR into the Add-Source modal. Adds endpoint, subprocess lifecycle, QR component, mutual-exclusion guard; requires UI-SPEC amendment. | ✓ |
| Let the spike decide | Task 1 spike picks A or B at a checkpoint, A as default. | |

**User's choice:** In-app QR flow (selected the modal mockup preview showing QR + refresh countdown alongside Display name / Local path fields).
**Notes:** User chose the polished flow over the research's recommended lower-risk default, accepting the UI-SPEC amendment and extra kernel surface.

### Q2 — When does the QR panel appear during the Add-Source flow?

| Option | Description | Selected |
|--------|-------------|----------|
| Inline during add (Recommended) | After Step 1's trial-launch reports "not linked", the modal offers the QR panel right there — configure → scan → saved and syncing. | ✓ |
| Save first, link after | Save the unlinked instance; link later from the source chip. | |

**User's choice:** Inline during add.

### Q3 — After a de-link, ban, or session expiry, where does re-linking happen?

| Option | Description | Selected |
|--------|-------------|----------|
| Chip menu "Re-link…" (Recommended) | Phase 7 D-12 popover gains a "Re-link…" entry opening the same QR panel in a dialog; health tooltip points at it. | ✓ |
| Re-open the edit modal | QR panel inside the existing edit-config modal; less discoverable. | |

**User's choice:** Chip menu "Re-link…".

### Q4 — Should the standalone CLI link mode also exist as a fallback?

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, keep both (Recommended) | Terminal ASCII-QR path is cheap and gives a recovery route if the in-app flow breaks. | ✓ |
| In-app only | One linking path, less to build; no workaround if it breaks. | |

**User's choice:** Yes, keep both.

---

## Matching scope: groups vs 1:1

### Q1 — Groups only, or also 1:1 chats?

| Option | Description | Selected |
|--------|-------------|----------|
| Groups only | As SRC-03 is written; single `groups` match field. | |
| Groups + 1:1 chats | Mirror Signal's Phase 4 scope: second contacts match field, matching the user's own saved name (push name excluded, anti-injection rationale). Widens SRC-03's wording and the UI-SPEC match-field table. | ✓ |

**User's choice:** Groups + 1:1 chats — resolves research Assumption A5 (the groups-only reading was never user-confirmed).

### Q2 — Chats with unsaved contacts?

| Option | Description | Selected |
|--------|-------------|----------|
| Not matchable (Recommended) | No saved name → can't match any webspace. Mirrors Signal; keeps the anti-injection rule airtight. | ✓ |
| Match phone number too | E.164 keywords could match unsaved chats. | |

**User's choice:** Not matchable.

---

## Claude's Discretion

- History depth on first link (area offered, not selected): whatsmeow default backfill; observe at spike.
- Message-store lifecycle (area offered, not selected): keep-and-merge across re-links; never deleted on failure states; no purge affordance this phase.
- Link-endpoint transport and shape, QR encoder selection (pending legitimacy audit), match-field naming, deep-link mechanism (spike), whatsmeow commit pin, health `last_error` wording.

## Deferred Ideas

- E.164 phone-number matching for unsaved contacts (declined D-07 alternative).
- Message-store purge/reset affordance.
- Signal schema-version verify-and-accept tooling (reviewed todo, not folded — stays pending).
