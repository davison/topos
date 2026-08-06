# Phase 6: UI — Scalable Source Surface - Context

**Gathered:** 2026-08-06
**Status:** Ready for planning

<domain>
## Phase Boundary

The webspace header presents many source instances without duplication — health and filtering combined into one affordance per source (UI-07) — and the accumulated UI polish lands: deep-link fidelity differentiation for raise-window-only links (UI-08, closes the 04-UAT follow-up), search-term highlighting in the detail pane's rendered content across content variants (UI-09), and themed scrollbars with date markers on the stream scrollbar (UI-11; the app-wide thin themed scrollbars already landed via quick task 260805-j98 — the stream scrollbar's date markers are the remaining novel piece).

All in-repo UI work against the Phase 5 instance-identity model. Hard constraints from ROADMAP.md notes: highlighting inside sanitized HTML and chat transcripts must happen **after** sanitization without weakening it — the sanitizer contract (now kernel-owned per Phase 5 D-11) is untouchable. Fidelity differentiation surfaces the `LINK_FIDELITY_*` already declared per item (PLUG-03) — UI surfacing only, no contract change. No config-writing UI (Phase 7), no new sources (Phase 8).

</domain>

<decisions>
## Implementation Decisions

### Combined source chip (UI-07)
- **D-01:** **Clicking the chip body toggles the source filter** — click filters the stream to that source, click again removes it from the filter. The health dot stays a passive indicator; refresh is a separate small control on the chip. One chip per instance replaces both of today's rows (SourceHealthChip row + SourceFilterChips row).
- **D-02:** **Filtering is multi-select**: each chip toggles its source in/out of the visible set; **all-off = show everything** (no dedicated "All" chip needed, though planning may keep one if it aids clarity). Supersedes Phase 2 D-09's single-select semantics. URL persistence carries forward as a list (e.g. `?sources=home-email,work-email`) so reloads and deep links preserve the selection. — **Reversibility:** reversible — narrowing back to single-select is a UI/URL change only.
- **D-03:** **Per-source refresh is hover/focus-revealed**: a small refresh icon appears on the chip on hover or keyboard focus (and remains visible as the spinner while syncing). Chips stay compact at rest; refresh stays on the chip itself per the success criterion. Touch fallback is planner discretion (focus reveal covers most cases).
- **D-04:** **Health detail stays in a hover tooltip** — hovering the chip shows display name, relative last-sync time, and last error text, carrying forward today's tooltip copy contract (02-UI-SPEC.md rows). No pinned popover, no error strip; the colored dot remains the at-a-glance signal.

### Claude's Discretion
The user explicitly left the remaining gray areas to research/planning:
- **Scaling to 10+ instances** (success criterion 1's overflow/grouping/collapse requirement): the strategy — overflow menu, grouping by plugin type, collapse threshold, wrap behavior — is open. Whatever is chosen must keep every instance reachable and avoid unbounded chip rows.
- **Fidelity differentiation (UI-08):** how raise-window-only links (Signal, `conversation-only`) are visually distinguished from navigating links — wording, icon, treatment of the current fidelity `Badge` in `OpenInSource.svelte`. Requirement is only that the user can tell the difference before clicking.
- **Search highlighting (UI-09):** highlight styling, whether match navigation (prev/next/jump-to-first) exists, and the mechanism per content variant — including how highlighting reaches the kernel-served iframe renditions without weakening the sanitizer (post-sanitization injection kernel-side vs client-side approaches are both open, sanitizer contract untouchable either way).
- **Stream scrollbar date markers (UI-11):** marker appearance, granularity (adaptive vs fixed), interactivity (drag tooltip vs clickable jump), and whether it's a custom scrollbar component or an overlay on the native one.
- "Refresh all" placement in the redesigned header, syncing-indicator treatment on the new chip, selected-state (filtered) chip styling.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope & requirements
- `.planning/ROADMAP.md` — Phase 6 goal, 4 success criteria, notes (fidelity is UI surfacing of PLUG-03's `LINK_FIDELITY_*`, sanitizer untouchable, scrollbar date markers the only novel piece).
- `.planning/REQUIREMENTS.md` — UI-07, UI-08, UI-09, UI-11 (this phase); UI-10 is deferred v1.x, don't build it.
- `.planning/PROJECT.md` — constraints and Key Decisions table.

### Prior locked decisions this phase builds on or supersedes
- `.planning/phases/05-source-instances-per-type-matching/05-CONTEXT.md` — instance identity (D-08 map key, D-09 unique display names), kernel-owned rendition sanitize/wrap/theme (D-11) which defines where sanitized HTML now comes from; the header redesign is against this final source-identity model.
- `.planning/phases/02-two-sources-one-trustworthy-stream/02-CONTEXT.md` — D-08 (health chip semantics/tooltip contract, kernel-side health merge), D-09 (filter-in-URL persistence; its single-select rule is superseded by this phase's D-02), D-10 (staleness semantics unchanged).
- `.planning/phases/04-signal-conversations/04-CONTEXT.md` — the conversation-only "open in Signal" affordance whose UAT follow-up UI-08 closes.
- `.planning/phases/02-two-sources-one-trustworthy-stream/` `02-UI-SPEC.md` (if present in phase dir) — copywriting contract rows the chip tooltip copy carries forward.

### Published contracts (consume, don't change)
- `docs/api.md` — sources/health/search endpoints and envelope; any new query params (multi-source filter) must follow it.
- `proto/topos/v1/` — `LINK_FIDELITY_*` enum consumed by UI-08; no contract change in this phase.

### Technology stack (locked)
- `.claude/CLAUDE.md` — SvelteKit 2/Svelte 5 SPA, shadcn-derived components, virtualized stream list guidance.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `web/src/lib/components/SourceHealthChip.svelte` — dot-tone mapping (`healthTone`), tooltip copy, syncing spinner: the merged chip inherits all of this; the component itself is rebuilt/absorbed.
- `web/src/lib/components/SourceFilterChips.svelte` — the filter row being absorbed into the combined chip; its `onfilter` wiring in `WebspaceHeader.svelte` is the integration point.
- `web/src/lib/components/WebspaceHeader.svelte` — owns both rows today plus "Refresh all" and `shouldShowSourceRows` gating (a non-critical sources failure never blanks the stream — preserve).
- `web/src/lib/components/OpenInSource.svelte` — the UI-08 change site: `fidelityLabel` map and secondary `Badge`; `displayName` parameterization already instance-correct.
- `web/src/lib/components/DetailPane.svelte` — `detailBodyVariant` branches (text / media / iframe rendition); UI-09 highlighting must handle each branch, iframe renditions come from the kernel content route.
- `web/src/app.css` — scrollbar tokens (`--scrollbar-thumb` etc., 260805-j98) already app-wide; UI-11's remaining work is the stream date-marker affordance, not the base theming.
- `web/src/lib/format.ts` — `healthTone`, `formatRelativeTime`, `shouldShowSourceRows`, fidelity helpers; extend rather than fork.

### Established Patterns
- Filter state persists in the URL query (Phase 2 D-09) — multi-select must keep reload/deep-link fidelity.
- Sync-failure branch renders before the empty branch; a failed sync never looks like an empty webspace — the merged chip row must not regress the header's error/loading gating.
- Kernel-served sanitized renditions in a sandboxed iframe with a rendition CSP (`kernel/httpapi/item.go`, Phase 5 D-11) — any kernel-side highlight injection lands at that boundary, after sanitization.
- Component tests colocated (`sources.test.ts`, `staleness.test.ts`, `detail-body.test.ts`) — chip merge and highlighting need equivalent coverage; built-stylesheet recurrence guard exists from 02-06.

### Integration Points
- `web/src/routes/w/` page — owns `selectedSource` state and the `?source=` query param; becomes a set (`?sources=`).
- `GET /api/webspaces/{ws}` stream endpoint — if filtering is server-side, it needs a multi-source query param; if client-side, no API change (planner decides).
- `kernel/httpapi/item.go` content route — only touched if highlighting is injected kernel-side (query param on the rendition URL); sanitizer and CSP unchanged either way.

</code_context>

<specifics>
## Specific Ideas

- The motivating example for scale is the Phase 5 instance model: multiple accounts of the same plugin type ("Home email" / "Work email") each get their own chip — the header must stay usable as that multiplies.
- No other specific references — user locked the chip interaction model and delegated the rest to standard approaches.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 6-UI — Scalable Source Surface*
*Context gathered: 2026-08-06*
