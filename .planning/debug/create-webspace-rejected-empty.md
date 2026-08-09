---
status: diagnosed
trigger: "G-07-3: Creating a new webspace from the UI's create-webspace modal can never succeed — the kernel rejects the write."
created: 2026-08-09T00:00:00Z
updated: 2026-08-09T00:10:00Z
---

## Current Focus

hypothesis: CONFIRMED — see Resolution
test: n/a (diagnosis-only session)
expecting: n/a
next_action: none — return ROOT CAUSE FOUND to caller

## Symptoms

expected: Create-webspace modal: submitting a name writes a new [webspaces.<name>] block through PUT /api/config and navigates to it without a kernel restart; a kernel rejection leaves the modal open with the typed name intact
actual: Creating any new webspace fails. The modal correctly stays open showing the kernel's verbatim message, but the rejection is structural and unavoidable
errors: config: webspace "uat" declares neither a keywords fallback nor any match block — declare `keywords = [...]`, a `[webspaces.uat.match.<instance>]` block, or both
reproduction: Test 3 in UAT — live kernel via make dev, click + New webspace, type any name, submit
started: Discovered during UAT 2026-08-09 (structural — present since 07-03 shipped, latent until first live-kernel UAT of this flow)

## Eliminated

(none — first hypothesis formed from direct code trace was confirmed; no red herrings encountered)

## Evidence

- timestamp: 2026-08-09T00:01:00Z
  checked: web/src/lib/components/CreateWebspaceModal.svelte (full file)
  found: handleSubmit calls addWebspace(config, trimmed) then putConfig({base_hash, config: nextConfig}) — exactly one PUT, per the plan's own contract. On ApiError it sets error to err.message verbatim and leaves `name` untouched. Behaves exactly as designed.
  implication: The modal/frontend write path is not the defect — it does precisely what 07-03-PLAN.md/07-UI-SPEC.md specify.

- timestamp: 2026-08-09T00:02:00Z
  checked: web/src/lib/config-edit.ts addWebspace()
  found: "next.webspaces[name] = { keywords: [], sources: [], match: {} };" — writes an empty shell with no keywords, no match, no sources allowlist.
  implication: Confirms the exact shape PUT to the kernel: all three of keywords/match/sources are empty.

- timestamp: 2026-08-09T00:03:00Z
  checked: web/src/lib/config-edit.test.ts describe('addWebspace') block
  found: 'adds an empty webspace entry with no sources allowlist yet (D-14)' asserts next.webspaces['new-project'] equals exactly {keywords:[], sources:[], match:{}} — a pure-function shape assertion, never round-tripped through a real kernel's config.Validate.
  implication: 07-03's own tests could never have caught this — they assert the JS object shape only, not kernel acceptance. 07-03-SUMMARY.md's own D2/D3 rationale entry states outright: "CreateWebspaceModal's actual PUT /api/config round trip against a live kernel (success, validation-failure Alert, hash-conflict Alert, disabled-while-saving) was not exercised live — same environment limitation as D2." This explains why the defect reached UAT undetected.

- timestamp: 2026-08-09T00:04:00Z
  checked: kernel/config/config.go validateWebspaces (line ~310-345)
  found: "if len(ws.Keywords) == 0 && len(ws.Match) == 0 { return fmt.Errorf(\"config: webspace %q declares neither a keywords fallback nor any match block...\") }" — this is an unconditional, blanket gate evaluated for EVERY webspace name in the document, with no exemption for a webspace whose `sources` allowlist is also empty/new.
  implication: Exact source of the verbatim error text reported in UAT. This check runs regardless of whether the webspace has any participating source instances at all.

- timestamp: 2026-08-09T00:05:00Z
  checked: kernel/config/types.go Webspace.Participates()
  found: "if len(w.Sources) == 0 { return true }" — an empty/absent `sources` allowlist means ALL currently configured source instances participate (Phase 5 D-03's deliberate default-all-participation encoding, reconfirmed by 07-05's D-14 interaction note: "[] IS the kernel's own all-instances-participate default encoding").
  implication: The `sources: []` that addWebspace() writes for a brand-new UI webspace is NOT interpreted by the kernel as "participates in nothing yet" — it is interpreted as "participates in everything already configured." This makes the freshly-created webspace look, to the validator, like a webspace with active participants and no coverage for any of them — even before validateWebspaces' line-323 blanket gate is reached, validateFallbackCoverage (D-06, ~line 416) would independently re-derive the same failure for any config that already has at least one [sources.*] block, which every real installation does.

- timestamp: 2026-08-09T00:06:00Z
  checked: kernel/httpapi/config.go ConfigSaveHandler + its doc comment
  found: "every rule up to and including the write (the clobber guard, the unknown-key guard, the validate-dry-run, the canonical write, the in-memory hot-swap) lives in config.Store.Save" — i.e. PUT /api/config runs full Config.Validate() (which calls validateWebspaces) synchronously on the WHOLE document on every single save, not just on the fields being touched.
  implication: There is no way to persist an intermediate "webspace shell, no source yet" state through this endpoint — the very first PUT that creates the webspace is validated against the same unconditional invariant a fully-populated webspace must satisfy.

- timestamp: 2026-08-09T00:07:00Z
  checked: 07-UI-SPEC.md line 60, 07-CONTEXT.md D-14, 07-03-SUMMARY.md line 136, 07-04-SUMMARY.md D-14 notes
  found: 07-UI-SPEC.md (07-03's own governing spec) states explicitly: "creating a webspace here always produces an empty webspace with **no** `sources` allowlist yet (D-14's explicit allowlist is written the first time a source is actually added)." The design is deliberately two-phase: Step A (this modal) creates an empty shell; Step B (a later, separate PUT via the "+" add-source picker, 07-04) is what first populates a match block (and, per D-14, the sources allowlist).
  implication: The two-phase creation flow is intentional product design (07-03/07-04), not an oversight in the modal. But no PLAN/SUMMARY/RESEARCH artifact for 07-03 or 07-04 revisits 05-03's validateWebspaces invariant to reconcile it with this new two-write flow — the two decisions (05-03 D-01 "keywords or match always mandatory" and 07-03/07-04 D-14 "empty shell created first, populated second") were made in different phases and are mutually exclusive at the moment Step A's PUT is submitted.

- timestamp: 2026-08-09T00:08:00Z
  checked: 05-03-PLAN.md/05-03-SUMMARY.md for any exemption clause
  found: No exemption for a webspace with zero configured sources or a not-yet-populated sources allowlist is mentioned anywhere in 05-03's task or acceptance criteria; validateWebspaces was designed against Phase 5's hand-written-config-only world, where every [webspaces.*] block in config.toml was written complete and saved in one shot by a human editing the file directly — a scenario where "declares neither keywords nor match" is unambiguously a mistake worth failing loudly on. Phase 7 introduced a UI that legitimately needs to persist a transient, valid-but-incomplete intermediate state between two separate writes, and that need was never fed back into config.Validate.
  implication: This is a cross-phase contract gap, not a local coding bug — RCA branching check below.

## Resolution

root_cause: |
  kernel/config/config.go's validateWebspaces (kernel/config/config.go:323, established by 05-03 D-01) unconditionally requires every `[webspaces.<name>]` block to declare either a non-empty `keywords` fallback or at least one `match` block — with no exemption for a webspace that has zero source instances allowlisted/participating yet. `config.Store.Save` (invoked synchronously by every `PUT /api/config`, kernel/httpapi/config.go's ConfigSaveHandler) runs this same full-document `Config.Validate()` on every save, including the very first save that creates a webspace.

  The Create-webspace modal (web/src/lib/components/CreateWebspaceModal.svelte, via config-edit.ts's addWebspace()) is deliberately designed (07-03/07-04 decision D-14, documented in 07-UI-SPEC.md line 60 and 07-CONTEXT.md) as a two-phase flow: Step A creates an empty webspace shell (`{keywords: [], sources: [], match: {}}`) via its own standalone PUT, and Step B — a separate, later PUT triggered by the "+" add-source picker — is what first writes a match block (and the `sources` allowlist, per D-14). This is intentional UX: create the container first, populate it by adding sources afterward.

  These two decisions are mutually exclusive at the moment Step A's PUT is submitted: 05-03's validator has no notion of "webspace legitimately empty so far, more to come in a later save" — it treats every webspace document as complete-and-final at every single save. Compounding this, `Webspace.Participates()` treats an empty/absent `sources` allowlist as "all currently configured instances participate" (Phase 5 D-03's deliberate default for hand-written configs), so the freshly-created webspace's `sources: []` is read by the kernel as "participates in every already-configured source," not "participates in nothing yet" — meaning even a narrower fix to just the line-323 blanket gate would still be caught by validateFallbackCoverage (kernel/config/config.go:416) for any installation that already has at least one `[sources.*]` block configured (true for essentially every real installation past initial setup).

  Root cause is a single cross-phase contract conflict (not a multi-condition AND-gate): 05-03's "every webspace must declare keywords-or-match, always, at every save" invariant was written before, and never revisited against, 07-03/07-04's "create an empty shell now, populate it with a second, later save" UI flow. The reported failure is 100% deterministic for any name — it is not data-, environment-, or timing-dependent (confirmed by reading the validator directly: it has no branch keyed on anything besides `len(Keywords)` and `len(Match)`).

  Why this reached UAT undetected: 07-03's own addWebspace() unit test (config-edit.test.ts, 'adds an empty webspace entry ... (D-14)') only asserts the returned JS object's shape — it never round-trips through a real kernel's config.Validate(). 07-03-SUMMARY.md's own D2/D3 rationale entry states this limitation outright: "CreateWebspaceModal's actual PUT /api/config round trip against a live kernel ... was not exercised live." No gate between 05-03 and 07-03/07-04 (test, typecheck, lint, review, or verify) actually submitted this write to a real config.Validate() call before this UAT session.

fix: (not applied — find_root_cause_only mode; diagnosis only)

verification: (not applicable — no fix applied in this session)

files_changed: []
