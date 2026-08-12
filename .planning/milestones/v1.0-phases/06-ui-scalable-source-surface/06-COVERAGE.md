---
phase: 06-ui-scalable-source-surface
status: decided
---

# API Coverage Matrix — Phase 6

Phase 6 (UI: scalable source surface) is a frontend-only phase. It integrates no
external API. The `api-coverage.verify-pre` gate's detection (`verb: integration,
noun: api`) is a heuristic false positive triggered by phase-artifact prose
referring to the kernel's own internal HTTP API, not a third-party API surface.

## Coverage

| Capability | Decision | Reason |
|------------|----------|--------|
| External API integration | OPT-OUT | Phase 6 touches only the SvelteKit web UI and the kernel's existing internal JSON API (already covered by prior phases). No new external API surface is introduced. |
