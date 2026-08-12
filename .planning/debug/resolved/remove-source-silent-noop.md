---
status: resolved
trigger: "G-07-6: Removing a source from a webspace via the chip menu's 'Remove from this webspace' is a silent no-op. Reported during UAT test 6."
created: 2026-08-09T00:00:00Z
updated: 2026-08-09T01:00:00Z
---

## Current Focus

reasoning_checkpoint:
  hypothesis: "TWO independent, each-individually-sufficient code defects combine to make 'Remove from this webspace' a visible no-op: (A) config-edit.ts's removeSourceFromWebspace naively .filter()s the sources allowlist array, which is a no-op whenever the result would be/already is empty — and kernel/config/types.go's Webspace.Participates() treats an empty sources array as 'ALL instances participate', not 'none' — so the removed instance keeps participating via the all-participate default; (B) WebspaceHeader.svelte's chip row renders the raw, kernel-wide GET /api/sources list completely unfiltered by webspace participation (no client-side Participates()-equivalent filter, unlike AddSourceModal.svelte's own correctly-implemented participatingSet) — so even a config write that DID correctly narrow participation would never make the chip visually disappear."
  confirming_evidence:
    - "config-edit.ts:156-172 removeSourceFromWebspace: `sources: existing.sources.filter((s) => s !== instance)` — for webspace 'cars' (no explicit allowlist, existing.sources === []), filtering an empty array yields [] again; the write is syntactically valid but semantically inert for participation."
    - "kernel/config/types.go:211-221 Webspace.Participates: `if len(w.Sources) == 0 { return true }` — confirms empty sources array is the kernel's own encoding of all-participate, not none-participate."
    - "config-edit.ts:128-145 addSourceToWebspace already implements the correct mirror-image pattern for ADDING (seeds `sources = Object.keys(cfg.sources)` when starting empty, before appending) — proving the team already knows this exact semantics gap exists, but the symmetric seed was never applied to removeSourceFromWebspace."
    - "config-edit.test.ts:231-251 removeSourceFromWebspace's ONLY test fixture ('house-move') starts with an explicit, non-empty sources: ['paperless'] allowlist — the empty/all-participate starting case (the user's actual 'cars' webspace shape) is never exercised by any test, so this gap was never caught by CI."
    - "Same test fixture also reveals a second manifestation of the identical mechanism: removing the SOLE remaining named entry ('paperless' from sources: ['paperless']) produces sources: [] — which Participates() also reads as all-participate, so even a webspace WITH an explicit single-entry allowlist doesn't actually lose the removed instance as a participant, though the test only asserts the JS array shape, not kernel participation semantics."
    - "WebspaceHeader.svelte's visible chip row (line 322-332, `{#each visibleSources as source}`) and its `visibleSources = $derived(sources.slice(0, visibleCount))` (line 275) both consume the `sources: SourceStatus[]` prop directly with zero reference to `config.webspaces[webspace]` or any Participates-equivalent predicate anywhere in the file (confirmed by full-file read and targeted grep for 'config.webspaces', 'ws.sources', 'participat' — zero hits)."
    - "+page.svelte's loadSources() (line 426-444) sets `sources = res.sources` directly from `getSources()` -> `GET /api/sources`, with no filtering step before passing `{sources}` into WebspaceHeader at line 578."
    - "kernel/httpapi/sources.go:58-79 SourcesHandler/sourceStatusesFrom's own doc comment: 'one entry per source INSTANCE... sorted by name' — explicitly kernel-wide/unfiltered; there is no webspace-scoped sources endpoint in kernel/httpapi/routes.go (confirmed: only GET /api/sources exists, no /api/webspaces/{ws}/sources)."
    - "AddSourceModal.svelte:72-80 DOES implement the correct client-side mirror of Participates() ('participatingSet') for its own 'instances not yet in this webspace' picker list — proving the concept exists and is correctly implemented exactly once in this codebase, but was never reused/applied to the main chip row in WebspaceHeader.svelte."
    - "07-04-PLAN.md's own Task 3 acceptance criteria (line 258) require: 'Remove from this webspace writes immediately... the chip disappears' — an explicit, authoritative contract this defect violates."
    - "07-04-SUMMARY.md line 191 admits: 'No live make dev session was available... the live browser<->kernel round trip is not [tested]' for exactly this flow — explaining why this gap was never caught before now."
    - "Ruled out kernel-side silent rejection / swallowed error (hint options b/c): +page.svelte's handleRemoveSource (line 204-224) sets filterError=null immediately after a successful putConfig (line 211) and only sets a rendered destructive Alert (WebspaceHeader.svelte line 480-482) on a caught ApiError; the user explicitly reported no error surfaced anywhere in the UI, which is only possible if putConfig succeeded (200 OK) — directly confirming the write round-trips successfully (matching the observed 'brief spinner on all chips') and is a genuine semantic no-op, not a failed/rejected write."
  falsification_test: "If removeSourceFromWebspace were fixed alone (seed sources from Object.keys(cfg.sources) before filtering, mirroring addSourceToWebspace) but WebspaceHeader.svelte's chip row were left unfiltered, the chip would still visually persist after removal — proving B is independently necessary. Conversely, if only WebspaceHeader.svelte gained a Participates()-equivalent filter but removeSourceFromWebspace's array-filter bug remained, the (still-empty) sources array would still resolve to 'all participate' under the corrected filter's own Participates() logic, so the chip would still show — proving A is independently necessary. Both must be fixed together; this was verified by tracing each function/component's logic in isolation, not by live-toggling code (diagnosis-only constraint)."
  fix_rationale: "(Not applied — find_root_cause_only mode.) The fix shape for A mirrors the already-established addSourceToWebspace pattern (materialize the current full participant set — existing.sources if non-empty, else Object.keys(cfg.sources) — THEN filter out the removed instance, never filtering an as-is array that may already/become empty). The fix shape for B is adding a participatingSet-equivalent filter (reusable, ideally extracted so AddSourceModal and WebspaceHeader share one implementation rather than diverging again) applied to the `sources` prop before slicing into visibleSources/hiddenSources."
  blind_spots: "Have not observed this live against the running kernel (diagnosis-only constraint — no PUT to the live kernel or ephemeral instance was performed); the AND-gate conclusion (both A and B are real, independently-sufficient defects) rests on static code reading across config-edit.ts, WebspaceHeader.svelte, +page.svelte, kernel/config/types.go, kernel/httpapi/sources.go and kernel/httpapi/routes.go plus the phase's own PLAN/SUMMARY docs, not on toggling the code and observing the DOM. Have not confirmed whether the user's live 'topos' config currently has any OTHER webspace or any source instance NOT participating in 'cars' — if their whole config.sources set happens to equal exactly what 'cars' already all-participates in, defect B would currently be behaviorally invisible even after A is fixed (both bugs still real and both required for the documented contract, but B's user-visible impact for this specific webspace may be masked until a second, differently-scoped webspace or a truly-excluded instance exists)."
  candidate_causes:
    - "code: removeSourceFromWebspace's array .filter() no-op on an empty/soon-to-be-empty sources allowlist (config-edit.ts) — CONFIRMED, category: code"
    - "code: WebspaceHeader.svelte's chip row has no webspace-participation filter at all, unlike AddSourceModal's participatingSet (WebspaceHeader.svelte / +page.svelte) — CONFIRMED, category: code (a second, distinct code site — different file, different mechanism, not a chained consequence of A)"
    - "config: considered and ELIMINATED — kernel-side rejection of the resulting document (e.g., config.Validate rejecting sources:[] or the narrowed match block) was ruled out because a rejection would surface via the caught ApiError -> destructive Alert path (+page.svelte:213-219, WebspaceHeader.svelte:480-482), and the user explicitly reported no error surfaced anywhere — inconsistent with a rejected/failed write."
    - "data: considered and ELIMINATED as an independent cause — the user's specific webspace shape (no explicit sources allowlist) is the DATA precondition that triggers defect A, but it is a triggering condition for a code defect, not a separate root cause in its own right (any webspace whose allowlist becomes empty after removal hits the same code defect, regardless of which specific webspace/data it is)."
  and_gate: "Effectively yes, though not the textbook 'both conditions must co-occur to produce the symptom' shape — here EITHER defect (A or B) is independently sufficient on its own to reproduce the exact observed symptom (chip does not disappear, no error). But BOTH must be fixed for the documented contract (07-04-PLAN.md's 'the chip disappears' acceptance criterion) to actually hold — fixing only one leaves the user-visible bug completely unchanged via the other mechanism. Recorded as a two-element root_cause set per the anti-single-cause-bias discipline, with this nuance made explicit so a future fix pass does not stop after addressing only the first one found."
next_action: "goal is find_root_cause_only — investigation complete, returning diagnosis. No fix applied (diagnosis-only constraint)."

## Symptoms

expected: Chip menu 'Remove from this webspace' removes the source from the webspace: the write round-trips through PUT /api/config and the chip disappears without reload
actual: All chips briefly show the syncing spinner, then the webspace is unmodified — the chip is still there, no error is surfaced anywhere
errors: None visible in the UI
reproduction: Test 6 in UAT — live kernel via make dev, chip menu → Remove from this webspace, confirm
started: Discovered during UAT 2026-08-09 (test 6 in 07-UAT.md)

## Eliminated

## Evidence

- timestamp: 2026-08-09T00:00:00Z
  checked: STATE.md decisions log, [Phase 07-05] entry
  found: "removeSourceInstance writes sources=[] (never omits the key) on a delete that empties a webspace's allowlist — Webspace.Participates treats empty identically to absent, so [] IS the kernel's own all-instances-participate default encoding"
  implication: This decision was scoped to the case where delete EMPTIES the allowlist (i.e. removing the last remaining explicit source). It does not obviously describe what happens when a webspace has NO allowlist to begin with and one of several all-participating instances is removed — that requires materializing a new allowlist of the remaining instances, which is a different code path. Need to read the actual implementation to see if that case is handled at all.

- timestamp: 2026-08-09T00:00:00Z
  checked: grep for removeSourceInstance usage
  found: defined/used in web/src/lib/config-edit.ts, web/src/lib/config-edit.test.ts, web/src/lib/components/ManageSourcesModal.svelte
  implication: ManageSourcesModal.svelte is the "Manage sources..." modal (UAT test 9, confirmed passing) — NOT the chip menu. The chip ⋮ menu's "Remove from this webspace" (UAT test 6, failing) may call a DIFFERENT function than removeSourceInstance, or the same function from a different call site. Need to find the chip-menu-specific call site.

- timestamp: 2026-08-09T00:15:00Z
  checked: web/src/lib/config-edit.ts full file, grep for "Remove from this webspace" and "removeSourceFromWebspace"
  found: "removeSourceFromWebspace (lines 156-172) is the actual chip-menu write path (confirmed by its own doc comment and by web/src/routes/w/[webspace]/+page.svelte:208 `removeSourceFromWebspace(configResponse.config, webspace, name)`). Implementation: `sources: existing.sources.filter((s) => s !== instance)`."
  implication: For a webspace with existing.sources === [] (no explicit allowlist, all-participate default — exactly 'cars'), filtering an empty array yields [] again. The write is a pure no-op on the sources field; only the match block entry is actually removed.

- timestamp: 2026-08-09T00:20:00Z
  checked: kernel/config/types.go Webspace.Participates (lines 208-221)
  found: "func (w Webspace) Participates(instance string) bool { if len(w.Sources) == 0 { return true } ... }"
  implication: Confirms empty sources array IS the kernel's all-participate encoding. Since removeSourceFromWebspace leaves sources=[] unchanged for 'cars', the removed instance ('docs') still Participates()==true after the write — it just lost its explicit match block and falls back to matchFieldsFor's keywords fallback (kernel/correlate/correlate.go:158-172, D-01) instead of being excluded.

- timestamp: 2026-08-09T00:25:00Z
  checked: config-edit.ts addSourceToWebspace (lines 128-145), the sibling ADD function
  found: "addSourceToWebspace already contains the correct symmetric handling: `if (sources.length === 0) { sources = Object.keys(cfg.sources); }` before appending — seeding the full participant set when starting from the all-participate default."
  implication: The correct pattern for materializing an explicit allowlist from the all-participate default is already established in this exact file for the ADD direction, but was never mirrored into the REMOVE direction (removeSourceFromWebspace). This is strong evidence the omission is a straightforward implementation gap, not a deliberate design choice.

- timestamp: 2026-08-09T00:30:00Z
  checked: web/src/lib/config-edit.test.ts describe('removeSourceFromWebspace') block (lines 231-251)
  found: "The only fixture used ('house-move') starts with an explicit non-empty allowlist `sources: ['paperless']`. No test exercises a webspace with an empty/absent allowlist (the all-participate default)."
  implication: Confirms this exact edge case was never covered by the test suite — explains why it shipped without being caught. Also: the house-move fixture removing 'paperless' (the sole named entry) produces sources:[], which the test asserts but which Participates() ALSO reads as all-participate — meaning even THIS passing test's scenario doesn't kernel-side actually revoke participation; the test only checks the JS array shape.

- timestamp: 2026-08-09T00:40:00Z
  checked: web/src/lib/components/WebspaceHeader.svelte (full file), web/src/routes/w/[webspace]/+page.svelte loadSources() and the WebspaceHeader invocation (line 573-599), kernel/httpapi/sources.go, kernel/httpapi/routes.go
  found: "The chip row (`visibleSources`/`hiddenSources`, derived directly from the `sources: SourceStatus[]` prop) has no filter by webspace participation anywhere. `sources` state in +page.svelte is set verbatim from `getSources()` -> GET /api/sources, which kernel/httpapi/sources.go's own doc comment confirms returns 'one entry per source INSTANCE' kernel-wide (unfiltered by webspace) — there is no webspace-scoped sources endpoint in routes.go."
  implication: Even a correctly-written config change that DID properly exclude 'docs' from webspace 'cars' participation would not make its chip disappear from the header, because the header was never wired to consult webspace participation for the chip row at all — a second, independent defect.

- timestamp: 2026-08-09T00:45:00Z
  checked: web/src/lib/components/AddSourceModal.svelte lines 72-84
  found: "participatingSet correctly mirrors kernel Participates() semantics client-side: `const sources = ws?.sources ?? []; return new Set(sources.length > 0 ? sources : Object.keys(config.sources));` — used only to compute the add-picker's 'available instances' list."
  implication: The correct filtering concept exists and is correctly implemented exactly once in this codebase, but was never reused for WebspaceHeader's main chip row — confirms defect B is a real, addressable gap (the pattern to copy already exists) rather than a fundamentally different intended architecture.

- timestamp: 2026-08-09T00:50:00Z
  checked: .planning/phases/07-webspace-builder-ui/07-04-PLAN.md (Task 3 acceptance criteria, line 258) and 07-04-SUMMARY.md (line 191)
  found: "PLAN acceptance criteria explicitly requires: 'Remove from this webspace writes immediately with no confirmation dialog, the chip disappears, and the instance's [sources.<id>] block remains in config.toml.' SUMMARY explicitly admits: 'No live make dev session was available in this execution environment to perform the plan's own <verification> section's live-kernel checks... the live browser<->kernel round trip is not [tested].'"
  implication: Confirms 'the chip disappears' is the authoritative, documented contract (not my own inference), and confirms why defect B was never caught — the only verification performed was unit-level (pure config-edit.ts functions, structural component guards), never a live end-to-end check of the chip actually disappearing.

- timestamp: 2026-08-09T00:55:00Z
  checked: web/src/routes/w/[webspace]/+page.svelte handleRemoveSource (lines 204-224) and WebspaceHeader.svelte filterError rendering (lines 480-482)
  found: "filterError is set to null immediately after a successful putConfig (line 211); it is only ever set to a user-visible message inside the catch block (ApiError handling), which renders as a destructive Alert. The user explicitly reported no error surfaced anywhere in the UI."
  implication: Rules out 'kernel rejects the write' and 'UI write fails and error is swallowed' (hint options b/c) — a rejection or thrown error would have produced a visible Alert. The absence of any visible error confirms the PUT succeeded (200 OK) and the observed 'brief spinner on all chips' is the normal post-save reload (loadSources()/load()), consistent with option (a): the write is a genuine semantic no-op, not a failure.

## Resolution

root_cause: "TWO independent, each-individually-sufficient defects (see reasoning_checkpoint above for full evidence): (A) web/src/lib/config-edit.ts's removeSourceFromWebspace (lines 156-172) filters the webspace's `sources` allowlist array without first materializing it from the all-participate default (Object.keys(cfg.sources)) when it starts empty — unlike its sibling addSourceToWebspace, which already does this correctly for the add direction. Per kernel/config/types.go's Webspace.Participates (lines 211-221), an empty `sources` array means ALL instances participate, so removing the last/only reference to an instance from an already-empty or soon-to-be-empty allowlist leaves that instance still participating — kernel-side, nothing was actually removed. (B) web/src/lib/components/WebspaceHeader.svelte's main chip row renders the `sources` prop (raw, kernel-wide GET /api/sources output) with no client-side filter by webspace participation at all — unlike AddSourceModal.svelte's own correctly-implemented participatingSet — so the chip row can never reflect a webspace's actual participation set, even when the underlying config is correct. Both must be fixed for the documented 'the chip disappears' behavior (07-04-PLAN.md Task 3 acceptance criteria) to hold; fixing only one leaves the symptom unchanged via the other."
fix: (not applied — find_root_cause_only mode)
verification: (not applicable — diagnosis only; no PUT issued against the live kernel or any config file per the DIAGNOSIS ONLY constraint)
files_changed: []
