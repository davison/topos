# Project Retrospective

*A living document updated after each milestone. Lessons feed forward into future planning.*

## Milestone: v1.0 — MVP

**Shipped:** 2026-08-12
**Phases:** 12 (10 planned + 07.1, 09.1 inserted) | **Plans:** 92 | **Tasks:** 241

### What Was Built

- A Go kernel correlating five real personal-data sources (paperless-ngx, SilverBullet, Proton/IMAP, Signal, WhatsApp) into per-topic webspaces, each source a go-plugin gRPC subprocess behind a published, third-party-buildable contract
- A Svelte 5 SPA (stream + detail pane, search, source chips) that also builds and edits webspaces end to end — the kernel's first mutating API with hot apply — plus mobile layout and first-run bootstrap
- Regression armor and release engineering: hermetic Playwright e2e suite in CI, change-gated nightlies, tag-triggered release artifacts, docs for users/contributors/security/plugin operators

### What Worked

- **Vertical MVP slices against real sources from day one** — no mock-only phase; every phase ended user-visible, which surfaced integration truths (Bridge certs, Signal schema drift, WhatsApp session lifecycle) at the cheapest possible moment
- **Risk-ascending source order** — paperless → SilverBullet → IMAP → Signal → WhatsApp meant v1 was already useful before the most bannable source was attempted; WhatsApp failure modes degraded one plugin, not the milestone
- **Early contract decisions while the blast radius was small** — agent permissions in Phase 2 (two plugins), instance identity + typed matching in Phase 5 (before any external plugin exists)
- **Inserting the e2e harness (07.1) before Phase 8's churn** — the standing D-11 rule (UI phases extend the Playwright suite as definition of done) converted UAT findings into permanent specs; 4 pre-existing bugs were flushed out just by building the harness
- **Structural/AST guards over prose promises** — read-only scans, RPC allowlists, egress pinning, comment-stripped source scans; several verification gaps were caught precisely because a guard was mechanical rather than a remembered convention

### What Was Inefficient

- **Phase 7 took 16 plans, 11 of them gap closure across three UAT rounds** — the supervisor's generation semantics (Apply/commitGeneration/rollback ordering) were re-fixed in 07-09, 07-10, then again in 08-09 and 08-13; the lifecycle contract should have been designed and structurally pinned once, up front
- **Phase 8's wave-6 fix introduced the wave-7 regression (G-08-5)** — a blocking login wait added under the supervisor's read mutex froze every source; a lock-discipline guard earlier would have caught it at authoring time
- **17 debug sessions accumulated unclosed** even though every one's fix had shipped — closing them required a dedicated verification sweep (4 agents cross-checking diagnoses against code) at milestone close instead of a one-line status flip when each fix landed
- **Phase 6's chip polish took three UAT rounds** — the UI spec's prose was internally contradictory (three statements about one control's geometry); pixel-level acceptance examples in the spec would have converged in one round

### Patterns Established

- **Generation change as the only plugin-lifecycle mutation** — stopScheduler → Reconcile → commitGeneration, no bespoke suspend/resume paths (violating this is what caused G-08-3)
- **Kernel-owned trust boundaries** — sanitize/wrap/theme at the rendition boundary, MIME-allowlisted CSP-sandboxed icon serving; plugins produce content, never presentation
- **AND-gate root-cause discipline in debug sessions** — multi-cause bugs documented with each cause's individual sufficiency stated, preventing single-cause "fixes" that leave the symptom alive
- **Debug knowledge base of defect *classes*** (KB-001…004: shutdown-context cancels the finalizing write; any-row vs latest-row; latest-row aggregate as wrong oracle; defer behind a blocking call) — pattern recall decoupled from individual sessions
- **UAT items become Playwright specs, not remembered manual checks** (07.1 D-11, in `.claude/CLAUDE.md` and `docs/testing.md`)

### Key Lessons

1. **Close debug sessions the moment their fix ships.** A `status: diagnosed` file whose defect is already fixed is debt that compounds into an audit-blocking pile at milestone close.
2. **Concurrency/lifecycle seams need their contract enforced structurally on first design.** Four separate plans re-fixed supervisor generation/lock discipline; the eventual AST-guard approach should arrive with the first implementation, not the fourth fix.
3. **Real-device checkpoints are irreplaceable for linked-device sources.** Hermetic gates caught nothing about WhatsApp's pairing window (G-08-1/-3/-4/-5 were all real-device findings); budget the human re-test into every plan touching such lifecycles.
4. **UI specs need concrete geometry, not adjectives.** Every multi-round UAT convergence traced back to spec prose that could be satisfied by contradictory implementations.

### Cost Observations

- Model mix: not tracked for v1.0
- Sessions: not tracked (per-plan durations recorded in STATE.md ranged ~4min–3h)
- Notable: gap-closure plans (verification/UAT/review-driven) accounted for roughly a third of all plans (≈30 of 92) — the verification loop is where much of the milestone's wall-clock went, and also where most of its durable guarantees came from

---

## Cross-Milestone Trends

### Process Evolution

| Milestone | Phases | Plans | Key Change |
|-----------|--------|-------|------------|
| v1.0 | 12 | 92 | Established: vertical slices, risk-ascending ordering, D-11 e2e rule, structural guards |

### Cumulative Quality

| Milestone | E2E Specs | Go Test Posture | Notable |
|-----------|-----------|-----------------|---------|
| v1.0 | 42 tests / 16 spec files | race suite + AST guards + portable/cgo split | 31/31 requirements verified |
