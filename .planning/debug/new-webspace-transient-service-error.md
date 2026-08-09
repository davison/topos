---
status: diagnosed
trigger: "G-07-1: Immediately after creating a new webspace via the UI, the newly created /w/<name> page shows an error state instead of the empty stream; clicking Retry then correctly shows 'nothing here yet'."
created: 2026-08-09T16:00:00Z
updated: 2026-08-09T16:20:00Z
bug_class: Bohrbug (deterministic given the async-sync window; reproduces on every create while any configured source's eager resync is still in flight)
---

## Current Focus
<!-- OVERWRITE on each update - reflects NOW -->

hypothesis: CONFIRMED — see reasoning_checkpoint and Resolution below
next_action: return ROOT CAUSE FOUND to orchestrator (goal: find_root_cause_only; no fix applied)

reasoning_checkpoint:
  hypothesis: "GET /api/webspaces/<name>/stream 404s (webspace_not_found) for a just-created webspace because StreamHandler gates existence on the index's `webspaces` table — whose ONLY insert site is the sync-time ReplaceWebspaceSourceItems — while the eager resync that eventually performs that insert is dispatched as fire-and-forget goroutines AFTER PUT /api/config has already answered 200; the client's load() then misclassifies the typed 404 as 'The topos service didn't respond'."
  confirming_evidence:
    - "kernel/httpapi/stream.go:72-80 — StreamHandler calls store.WebspaceExists and 404s webspace_not_found when false; existence never consults the config, despite cfgStore being right there (line 68)"
    - "kernel/index/store.go:652-662 — WebspaceExists is `SELECT 1 FROM webspaces WHERE name = ?`; doc comment: 'reports whether name has completed at least one sync'"
    - "kernel/index/store.go:230-235 — the ONLY `INSERT INTO webspaces` in the kernel lives inside ReplaceWebspaceSourceItems (grep-verified single site)"
    - "kernel/correlate/correlate.go:88-103,170-188 — an empty shell hits matchFieldsFor rule 3 (no match block, no keywords → participates=false), and the non-participation branch STILL calls ReplaceWebspaceSourceItems(ctx, name, src, nil) — the clear path that inserts the webspaces row; this is what makes Retry heal"
    - "kernel/supervisor/supervisor.go:399-406 — when oldCfg.Webspaces != newCfg.Webspaces, Apply dispatches `go coord.Refresh(context.Background(), name)` per unchanged source (fire-and-forget) and returns; kernel/httpapi/config.go:185-191 — the 200 is written immediately after Apply returns, i.e. BEFORE any refresh goroutine has run SyncSource"
    - "web/src/routes/w/[webspace]/+page.svelte:306-318 — load()'s bare catch maps EVERY getStream failure (including ApiError code webspace_not_found from api.ts getJSON:140-156) to loadState='error'; StreamList.svelte:56-57 renders StreamError for that state; StreamError.svelte:20-22 is the verbatim reported copy 'Couldn't load this webspace / The topos service didn't respond — check that it's running, then retry.'"
    - "Sequencing rules out a pre-PUT fetch: CreateWebspaceModal.svelte:59-62 awaits putConfig before oncreated; handleWebspaceCreated (+page.svelte:243-253) awaits loadConfig then goto, so the failing GET is strictly after the PUT's 200"
    - "User-observed behavior matches the mechanism exactly: error immediately after navigation, Retry (same load() call, seconds later) returns the empty stream — the interval in which the async Refresh's SyncSource clear path inserted the row"
  falsification_test: "Would be disproven if (a) WebspaceExists consulted config (it doesn't — direct read), (b) a webspaces row were written synchronously during Save/Apply (impossible — single INSERT site, reachable only via SyncSource, dispatched via `go`), or (c) the first stream GET could precede the PUT completing (ruled out by awaited promise chain). Live falsification: curl GET /api/webspaces/<new>/stream immediately after a PUT adding an empty shell → expect 404 webspace_not_found envelope; retry after one sync cycle → 200 empty."
  fix_rationale: "n/a — diagnose-only mode; fix direction recorded in Resolution"
  blind_spots: "Mechanism established by exhaustive code reading, not a live kernel repro (no ephemeral-kernel run this session). Confidence high: the code path is fully determined (row can only exist post-SyncSource) and the observed heal-on-retry matches. Not measured: actual width of the async window per source type (irrelevant to the mechanism)."
  candidate_causes:
    - "code (kernel/API design): stream existence gate derives 'webspace exists' from sync history (index webspaces table) rather than the running config — a config-known, never-synced webspace 404s [CONFIRMED, primary]"
    - "code (client): load() collapses all typed ApiError codes into the fixed service-unreachable copy — same class as resolved G-07-4 [CONFIRMED, co-cause of the misleading copy]"
    - "environment/timing: fire-and-forget eager-resync goroutines in Supervisor.Apply open the window between PUT 200 and the row insert [CONFIRMED, the trigger mechanism of cause 1]"
    - "data: stale/corrupt index rows — ELIMINATED (fresh webspace has no rows by definition; heal-on-retry inconsistent with corruption)"
  and_gate: "YES — the reported symptom requires BOTH the 404 window (kernel existence semantics + async resync) AND the client misclassification (which turns an accurate 'webspace_not_found' into false 'service didn't respond' copy). Fixing only the client would still flash a transient error state (accurate copy); fixing only the kernel removes the symptom for this flow but leaves the misclassification latent for every other non-network ApiError."

## Symptoms
<!-- Written during gathering, then IMMUTABLE -->

expected: After "+ New webspace" -> name -> submit, the modal closes and the app navigates to /w/<name>, which renders the EMPTY stream ("nothing here yet") immediately — no error state.
actual: "When the space is initially created, an error appears in the newly created space; 'Couldn't load this webspace — The topos service didn't respond — check that it's running, then retry.' upon hitting retry it correctly shows 'nothing here yet'" (verbatim user report). The chip-add flow that follows works correctly.
errors: UI error copy "Couldn't load this webspace — The topos service didn't respond — check that it's running, then retry." No kernel-side errors reported.
reproduction: Test 1 in .planning/phases/07-webspace-builder-ui/07-UAT.md — `make dev`, webspace title drop-down -> "+ New webspace" -> type name -> submit; observe the newly navigated /w/<name> page before any retry.
started: Discovered during round-2 UAT (2026-08-09), first live run after gap-closure plan 07-11 (D-20 empty webspace shell) landed. The creation flow does PUT /api/config (create empty shell) then client-side navigation to /w/<name>.

## Symptoms
<!-- Written during gathering, then IMMUTABLE -->

expected: After "+ New webspace" -> name -> submit, the modal closes and the app navigates to /w/<name>, which renders the EMPTY stream ("nothing here yet") immediately — no error state.
actual: "When the space is initially created, an error appears in the newly created space; 'Couldn't load this webspace — The topos service didn't respond — check that it's running, then retry.' upon hitting retry it correctly shows 'nothing here yet'" (verbatim user report). The chip-add flow that follows works correctly.
errors: UI error copy "Couldn't load this webspace — The topos service didn't respond — check that it's running, then retry." No kernel-side errors reported.
reproduction: Test 1 in .planning/phases/07-webspace-builder-ui/07-UAT.md — `make dev`, webspace title drop-down -> "+ New webspace" -> type name -> submit; observe the newly navigated /w/<name> page before any retry.
started: Discovered during round-2 UAT (2026-08-09), first live run after gap-closure plan 07-11 (D-20 empty webspace shell) landed. The creation flow does PUT /api/config (create empty shell) then client-side navigation to /w/<name>.

## Eliminated
<!-- APPEND only - prevents re-investigating -->

- hypothesis: "Client-side TypeError (G-07-4's exact shape — e.g. iterating a null collection from the response) thrown inside load()'s catch"
  evidence: "load()'s try block (+page.svelte:306-318) only awaits getStream and assigns; response fields are null-guarded at every use site. The throw here is a genuine typed ApiError from a non-OK HTTP response, not a client TypeError. The misclassification half of G-07-4's pattern applies (bare catch → service-unreachable copy), but the throw source is different."
  timestamp: 2026-08-09T16:15:00Z

- hypothesis: "GET blocks on a lock held during Apply and a client/proxy timeout aborts it"
  evidence: "No AbortController/timeout anywhere in web/src/lib/api.ts (getJSON is a bare fetch); Apply completes before the PUT's 200 is written (config.go:185-191), so the navigation-triggered GET never races Apply's own lock. Also the error appears near-instantly per the user, not after a timeout-scale delay."
  timestamp: 2026-08-09T16:15:00Z

- hypothesis: "Stale boot-time config snapshot in the stream handler (07-01's accepted debt)"
  evidence: "StreamHandler reads cfg fresh from cfgStore.Expanded() per request (stream.go:68) — 07-02 closed that debt for this handler. And a stale snapshot would not heal on retry without a kernel restart; the user's retry healed in seconds."
  timestamp: 2026-08-09T16:15:00Z

- hypothesis: "Data problem — stale or corrupt index rows for the new webspace"
  evidence: "A freshly created webspace has no index rows by definition; the failure is the ABSENCE of a webspaces row, and heal-on-retry via the sync clear-path insert is inconsistent with corruption."
  timestamp: 2026-08-09T16:15:00Z

## Evidence
<!-- APPEND only - facts discovered -->

- timestamp: 2026-08-09T16:00:00Z
  checked: .planning/debug/knowledge-base.md (head) + 07-UAT.md gap history
  found: "G-07-4 root cause was a client-side TypeError (Object.keys on null webspaces) inside the same try/catch that catches fetch failures in +page.svelte onMount, rendering the generic service-didn't-respond copy while the kernel answered 200 OK. Fixed by 07-12 for the root page; the /w/[name] page may have the same misclassification shape."
  implication: The identical error copy ('The topos service didn't respond') appearing here strongly suggests the /w/[name] page shares the misclassification pattern — a non-network throw or a non-OK response being reported as service-unreachable.

- timestamp: 2026-08-09T16:05:00Z
  checked: web/src/lib/components/StreamError.svelte + StreamList.svelte + web/src/routes/w/[webspace]/+page.svelte
  found: "StreamError.svelte:20-22 carries the verbatim reported copy. StreamList renders it whenever state==='error'. load() (+page.svelte:306-318) sets loadState='error' in a bare catch around getStream — every failure class (network, 404, 500) collapses to the same copy. Retry (onretry) re-invokes the identical load(navGeneration)."
  implication: The failure is genuinely transient at the HTTP layer (same call fails then succeeds) — and any non-OK status is presented as 'service didn't respond'.

- timestamp: 2026-08-09T16:08:00Z
  checked: web/src/lib/api.ts getJSON/putJSON + CreateWebspaceModal.svelte + handleWebspaceCreated
  found: "getJSON throws ApiError(code,message) for ANY non-ok response — a 404 with the kernel's error envelope becomes ApiError('webspace_not_found', ...). No timeout/abort exists. The modal awaits putConfig before oncreated; handleWebspaceCreated awaits loadConfig then goto — so the stream GET is strictly ordered after the PUT's 200."
  implication: The transient failure must be a kernel-side non-OK response in a post-PUT window, not a request race against the PUT itself.

- timestamp: 2026-08-09T16:10:00Z
  checked: kernel/httpapi/stream.go StreamHandler
  found: "Line 72-80: known, err := store.WebspaceExists(ctx, name); if !known → 404 webspace_not_found ('webspace \"<name>\" is not configured or has not been synced'). Existence is decided by the INDEX store alone; the live config (cfgStore, already read on line 68 for filters/display names) is never consulted for existence."
  implication: A config-known but never-synced webspace 404s. The question becomes: what inserts the index row, and when relative to the PUT response?

- timestamp: 2026-08-09T16:12:00Z
  checked: kernel/index/store.go WebspaceExists (652-662) + repo-wide grep for INSERT INTO webspaces
  found: "WebspaceExists = SELECT 1 FROM webspaces WHERE name=? ('has completed at least one sync'). The ONLY insert site in the kernel is ReplaceWebspaceSourceItems (store.go:230-235), which upserts the row on any source's contribution — including a nil-items clear."
  implication: The row can only appear via a sync-cycle write; nothing in the config save/apply path writes it synchronously.

- timestamp: 2026-08-09T16:14:00Z
  checked: kernel/correlate/correlate.go SyncSource + matchFieldsFor
  found: "For an empty shell: Participates()=true (empty allowlist = all-participate, types.go:211-221), no match block, len(Keywords)==0 → rule 3 (D-20 safety) returns participates=false. SyncSource's non-participation branch STILL calls ReplaceWebspaceSourceItems(ctx, name, src.Name(), nil) — the de-allowlist clear path — which performs the webspaces-row upsert."
  implication: The FIRST sync of ANY source after the config gains the shell inserts the shell's webspaces row. This is exactly why Retry heals. Corollary: with ZERO configured sources, no sync ever runs and the 404 would be PERMANENT — retry would never fix it on a fresh install.

- timestamp: 2026-08-09T16:16:00Z
  checked: kernel/supervisor/supervisor.go Apply (337-409) + kernel/httpapi/config.go ConfigSaveHandler (153-193)
  found: "Apply: on Webspaces diff, dispatches `go coord.Refresh(context.Background(), name)` for every unchanged source instance — fire-and-forget goroutines — then returns nil. ConfigSaveHandler writes the 200 immediately after Apply returns. So at the instant the client receives 200, the eager resync (whose SyncSource clear path will insert the new webspace's row) has merely been SCHEDULED, not completed; a real source's Match RPC takes real time (IMAP/SQLCipher reads)."
  implication: Deterministic window: PUT 200 → navigate → GET stream → WebspaceExists=false → 404 → StreamError with service-unreachable copy. Seconds later the refresh goroutines finish → row exists → Retry returns 200 with empty items → 'nothing here yet'. Complete causal chain, every link directly observed in code.

## Resolution
<!-- OVERWRITE as understanding evolves -->

root_cause: "Two co-causes (AND): (1) Kernel — GET /api/webspaces/{name}/stream decides webspace existence solely from the index's `webspaces` table (index/store.go WebspaceExists: 'has completed at least one sync'), whose only insert site is the sync-time ReplaceWebspaceSourceItems; a just-created webspace is config-known but index-unknown until the eager resync that Supervisor.Apply dispatches as fire-and-forget goroutines (supervisor.go:399-406, `go coord.Refresh`) completes AFTER PUT /api/config has already returned 200 — so the create flow's immediate stream GET deterministically hits 404 webspace_not_found; with zero configured sources the 404 would be permanent, not transient. (2) Client — load() in web/src/routes/w/[webspace]/+page.svelte (306-318) collapses every getStream failure, including the typed ApiError('webspace_not_found'), into loadState='error', which renders StreamError.svelte's fixed 'The topos service didn't respond' copy — misreporting a definitive kernel 404 as an outage (same misclassification class as resolved G-07-4)."
fix: "(not applied — diagnose-only) Direction: (1) make StreamHandler's existence gate config-aware — it already reads cfg := cfgStore.Expanded(); treat a webspace present in cfg.Webspaces as known (200 + empty items) even before its first sync, keeping the 404 for names in neither config nor index; alternatively upsert the webspaces row synchronously in the save/apply path for newly configured webspaces. (2) In load(), branch on ApiError code: webspace_not_found → a distinct 'this webspace doesn't exist' state (or empty state), reserving the service-unreachable copy for genuine network failures — mirroring 07-12's fix shape on the root route."
verification: "n/a — no fix applied this session"
files_changed: []
