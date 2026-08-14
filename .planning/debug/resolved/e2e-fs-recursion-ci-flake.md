---
status: resolved
trigger: "CI-only flake in web/e2e/specs/12-filesystem-recursion.spec.ts: second test fails with ENOENT on unlinkSync of the module-scope corpus file at line 126, while line 124's stream assertion passed"
created: 2026-08-14T15:45:00Z
updated: 2026-08-14T17:20:00Z
---

## Current Focus
<!-- OVERWRITE on each update - reflects NOW -->

bug_class: Bohrbug-in-disguise — deterministic given a scheduling precondition (both tests of the file dispatched as separate jobs to the SAME worker), which local high-worker-count runs almost never produce. NOT a race: no timing window, only a scheduling arrangement.

hypothesis (sharpened, v2): `kernel` is a WORKER-scoped fixture (kernel.ts:345-365) but `test.afterAll` is a SUITE-scoped hook. Under fullyParallel each test becomes its own Playwright job. When one worker runs both jobs of this file consecutively, Playwright runs afterAll at the END OF EACH JOB — so rmSync(corpusDir) fires after test 1 — while the worker-scoped kernel survives into job 2 (same workerHash, worker not restarted) with its index still holding the nested item from job 1's healthy sync. Node's module cache means the module-scope mkdtempSync/writeFileSync at lines 49-53 does NOT re-run on job 2, so corpusDir points at the deleted tree. Result: waitForFirstSync returns instantly (last_status already set), line 124 passes off the STALE index, and unlinkSync at line 126 hits ENOENT. Locally, high worker count sends the two jobs to two different fresh worker processes, each with its own module scope AND its own kernel — so it never fires.

test: CONFIRMED — deterministic repro reproduced the exact CI signature (see Evidence 2026-08-14T16:35Z).
expecting: n/a — moving to fix.
next_action: apply the fix (shared corpus helper whose teardown scope matches its setup scope) across all 7 affected spec files, then mutation-check with the repro command.

```yaml
reasoning_checkpoint:
  hypothesis: >
    The temp corpus is CREATED by module-scope code, which Node executes exactly once per
    worker PROCESS, but DESTROYED by `test.afterAll`, which Playwright executes once per
    JOB. Under `fullyParallel: true`, createTestGroups splits a file's tests into
    ceil(tests/workers) groups, so at workers>=2 this file becomes two jobs; the dispatcher
    then preferentially hands job 2 back to the same worker (matching workerHash), where the
    module cache is warm and will never re-seed. Job 2 therefore runs against a corpus that
    job 1's afterAll deleted.
  confirming_evidence:
    - "Deterministic repro: --workers=2 over [12-filesystem-add-source, 12-filesystem-recursion] reproduces the identical ENOENT at line 126, same artifact directory name as the CI trace."
    - "Direct read of playwright@1.62.1 workerProcessEntry.js: afterAll fires whenever the next test in the SAME JOB does not share the suite — i.e. at the end of every job."
    - "Direct read of createTestGroups: parallelWithHooksGroupSize = ceil(tests/workers); workers==1 gives ONE group and the bug is then impossible."
    - "Direct read of Dispatcher._scheduleJob: a free worker whose hash already matches is preferred over starting a fresh one — co-location is the preferred path, not bad luck."
    - "kernel.ts:340-365 declares both configSpec and kernel `scope: 'worker'`, so the kernel outlives the suite hook that deletes the corpus."
    - "Repro test durations of 9ms/4ms prove no kernel boot occurred for test 2 — it reused job 1's kernel and its index."
  falsification_test: >
    If the mechanism were something other than afterAll-vs-module scope, then forcing the two
    tests into ONE job (workers=1, which makes ceil(2/1)=2 → a single group) would still fail.
    It does not: `--workers=1` on this file passes. Equally, if the fix is right, removing the
    afterAll teardown must make the SAME two-file --workers=2 command pass — and re-adding it
    must make it fail again.
  fix_rationale: >
    Make the teardown scope match the setup scope. Module-scope state is per-worker-process
    state, and the only Node primitive whose lifetime is the process is `process.on('exit')`.
    Route every spec's corpus creation through one shared helper that mkdtemps and registers
    its own process-lifetime cleanup, and delete the per-job `test.afterAll` teardown entirely.
    This addresses the root cause (scope mismatch) rather than the symptom (this one ENOENT):
    it does not touch what any test proves, does not reduce parallelism, and closes the three
    sibling files that currently pass VACUOUSLY under the same arrangement.
  blind_spots:
    - "Whether `process.on('exit')` actually fires in a Playwright worker on graceful shutdown — must be verified empirically by checking /tmp for leaked corpora after a full run, not assumed. If it does not fire, the fix trades a failure for a temp-dir leak."
    - "A SIGKILLed worker will leak its corpus — but so did the old code, so this is not a regression. Not tested."
    - "Not tested against shard splitting (--shard), which the repo does not use."
    - "The structural guard is textual, not AST-based; a spec could evade it with an aliased import. Accepted: it pins the pattern for the way it is actually written here."
  candidate_causes:
    - "code (test lifecycle): module-scope setup torn down by a per-job hook — THE DEFECT"
    - "config: fullyParallel + default workers '50%' governs whether the file splits into >= 2 jobs at all — precondition"
    - "environment: CI vCPU count sets the worker count, and hence both the split and the co-location odds — precondition"
    - "design (fixture scoping): worker-scoped kernel keeps serving a stale index, which is why the failure surfaces at line 126 instead of a clearer failure at line 124 — masking factor"
  and_gate: >
    YES — three conditions must hold simultaneously. (A) afterAll destroys state the module
    will not recreate; (B) the dispatcher returns job 2 to the same worker process, so the
    module cache is warm; (C) the worker-scoped kernel survives with a stale index, so the
    first assertion passes and the failure lands on the filesystem call. (A) alone with a fresh
    worker is harmless; (A)+(B) without (C) would have failed at line 124 with a much clearer
    message. Only (A) is ours — (B) and (C) are correct, deliberate behaviours (Playwright's
    dispatcher, and D-02's one-kernel-per-spec-file decision). The fix therefore targets (A)
    only, and must not "fix" (B) or (C).
```

## Symptoms

DATA_START
expected: `make e2e` green on CI; the second test in 12-filesystem-recursion.spec.ts deletes the nested corpus file, POSTs /api/sources/docs-recursive/refresh, and asserts the item leaves the stream.
actual: CI run fails: `Error: ENOENT: no such file or directory, unlink '/tmp/topos-e2e-fs-recursion-MaUXSz/receipts/nested-invoice.pdf'` at web/e2e/specs/12-filesystem-recursion.spec.ts:126 (`unlinkSync(nestedFilePath)`). Notably, line 124's assertion that the item was present in the stream PASSED — the kernel index still had the item while the file was already gone from disk. 119 other tests passed. Exit code 2 via Makefile:247.
errors: ENOENT on unlinkSync (see actual). Trace artifact recorded at e2e/.artifacts/test-results/12-filesystem-recursion-12-d7bb3--just-the-next-full-replace-chromium/trace.zip on the CI run (2026-08-14T15:25 UTC, ubuntu 2-core runner).
timeline: First observed on the push after quick task 260814-mkl (which only added an ambient type declaration — unrelated to runtime). The spec landed in phase 12. Not seen locally.
reproduction: Not reproducible with plain local `make e2e` (full suite passes locally). Suspected to need CI-like scheduling: fewer workers (2-core runner) with other spec files interleaving so a worker exits and re-enters this spec file.
constraints: retries: 0 is deliberate (D-15 — a flaky test is a bug and gets fixed; do not add retries). D-03: corpus state must be seeded before kernel boot. The deletion test must still prove removal via the kernel's full-replace re-sync (no watcher). fullyParallel: true is the configured mode. Fix should target the spec's state lifecycle, not weaken what it proves.
key_files: web/e2e/specs/12-filesystem-recursion.spec.ts (module-scope mkdtempSync corpus at line 49, test.afterAll rmSync at line 82-84), web/playwright.config.ts (fullyParallel: true, retries: 0), web/e2e/fixtures/kernel.ts (per-spec-file hermetic kernel fixture), web/e2e/fixtures/config-builder.ts
DATA_END

## Evidence
<!-- APPEND ONLY -->

- timestamp: 2026-08-14T16:05Z
  checked: KB scan of .planning/debug/knowledge-base.md (KB-001..KB-006) for a matching class
  found: no direct match. KB-003 and KB-005 are adjacent in *shape* only — both are "latent defect, invisible at a dev box's core count, surfaced by a starved CI runner, blamed on the most recent (unrelated) commit". KB-005's line "docs-only commits failing was proof the failure was diff-independent and latent" applies verbatim to this session's `260814-mkl` (ambient-type-declaration-only) timeline entry.
  implication: treat the timeline correlation with 260814-mkl as noise, not signal. Look for a core-count-sensitive scheduling precondition, not a regression in the diff.

- timestamp: 2026-08-14T16:10Z
  checked: web/e2e/fixtures/kernel.ts fixture scoping (lines 340-365)
  found: BOTH `configSpec` and `kernel` are declared `{ scope: 'worker' }`. The kernel process, its index DB, and its temp dir therefore live for the lifetime of the WORKER PROCESS, not the test and not the spec file. `test.afterAll` in the spec, by contrast, is a SUITE-scoped hook.
  implication: the spec's corpus lifetime (suite-scoped) and the kernel's lifetime (worker-scoped) are governed by two different, unsynchronised scopes. Any scheduling arrangement where the suite scope closes but the worker scope does not leaves a live kernel pointed at a deleted corpus.

- timestamp: 2026-08-14T16:15Z
  checked: playwright@1.62.1 `createTestGroups` in web/node_modules/playwright/lib/runner/index.js
  found: a test lands in the `parallelWithHooks` bucket when it is `insideParallel` AND its ancestor chain declares a beforeAll/afterAll AND no ancestor is serial/default mode. All three hold here (fullyParallel:true; module-level `test.afterAll` on the file suite; no `describe.configure`). That bucket is then split by `parallelWithHooksGroupSize = Math.ceil(tests.length / expectedParallelism)`, where `expectedParallelism` is the resolved `config.workers`.
  implication: with 2 tests in the file — workers==1 → ceil(2/1)=2 → ONE group (afterAll runs once, no bug); workers>=2 → ceil(2/2)=1 → TWO groups, one test each. Worker count does not merely change speed here; it changes whether afterAll fires between the two tests at all.

- timestamp: 2026-08-14T16:18Z
  checked: playwright@1.62.1 worker teardown, `web/node_modules/playwright/lib/worker/workerProcessEntry.js`
  found: `for (const suite of reversedSuites) { if (!nextSuites.has(suite) || testInfo._isFailure()) await this._runAfterAllHooksForSuite(suite, testInfo); }` — `nextSuites` is the suite chain of the next test IN THE SAME JOB. Worker-scoped fixtures are torn down only in the separate `_isStopped` "Worker Cleanup" branch.
  found: `resolveWorkers` default is `"50%"` → `Math.max(1, floor(cpus*0.5))`.
  implication: afterAll fires at the end of EVERY job. A one-test job therefore runs `rmSync(corpusDir)` immediately after test 1, while the worker-scoped kernel keeps running untouched. Local box = 12 cpus → 6 workers; CI ubuntu-latest → 2 (4 vCPU) or 1 (2 vCPU).

- timestamp: 2026-08-14T16:22Z
  checked: playwright@1.62.1 `Dispatcher._scheduleJob` / `_runJobInWorker`
  found: `let workerIndex = this._workerSlots.findIndex(w => !w.jobDispatcher && w.worker && w.worker.hash() === job.workerHash && ...)` — the dispatcher PREFERS a free worker whose hash already matches, and only falls back to an arbitrary free slot (restarting that worker) when none matches. Each spec file's `test.use({ configSpec })` gives the file its own workerHash.
  implication: after job 1 of this file finishes, job 2 (identical hash) is preferentially handed back to the SAME worker process — module cache intact, kernel still alive. This is not an unlucky edge case; it is the dispatcher's preferred path whenever that worker is the free one.

- timestamp: 2026-08-14T16:35Z
  checked: deterministic repro — `npx playwright test --project=chromium --workers=2 e2e/specs/12-filesystem-add-source.spec.ts e2e/specs/12-filesystem-recursion.spec.ts`
  found: REPRODUCED on the first attempt, exact CI signature. `Error: ENOENT: no such file or directory, unlink '/tmp/topos-e2e-fs-recursion-a0oBAe/receipts/nested-invoice.pdf'` at 12-filesystem-recursion.spec.ts:126, with test 1 GREEN and the line-124 "item present before deletion" assertion GREEN — and the artifact directory name matches the CI trace path in the symptoms character for character (`12-filesystem-recursion-12-d7bb3--just-the-next-full-replace-chromium`). Test durations 9ms / 4ms confirm no kernel boot happened for test 2: it reused job 1's worker-scoped kernel.
  implication: root cause confirmed. The bug needs no timing race at all — only the scheduling arrangement "both jobs of this file executed by the same worker process", which `--workers=2` plus a slower preceding single-test file produces every time.

- timestamp: 2026-08-14T16:40Z
  checked: deduction from the CI failure itself against `parallelWithHooksGroupSize = ceil(tests / workers)`
  found: with this file's 2 tests, workers==1 yields ceil(2/1)=2 → a single group → afterAll cannot fire between the tests → the bug is impossible. The CI run DID fail, so the CI runner resolved `workers >= 2`, i.e. >= 4 vCPU.
  implication: the symptoms' "ubuntu 2-core runner" note is incorrect (ubuntu-latest is 4 vCPU for public repos). This matters: it means the failure is not rare bad luck on a starved box but the dispatcher's *preferred* scheduling on a 2-worker runner, and will recur on most CI runs where the sibling slot is busy.

- timestamp: 2026-08-14T16:45Z
  checked: blast radius — every spec file using the module-scope-mkdtemp + afterAll-rmSync pattern (7 files) and their test counts
  found: 12-filesystem-recursion (2 tests), 12-zero-match-diagnostic (2), 12-filesystem-tracer (2), 12-external-rehearsal (3) all split into >= 2 jobs at workers>=2. 12-filesystem-root-label-match, 12-filesystem-add-source and 12-include-glob-metadata-preview have 1 test each and are structurally immune *today only*.
  implication: the recursion spec is the only one that FAILS, because it is the only one that touches the corpus from disk in a later job (`unlinkSync`). The other three read only the kernel's stream/UI, which the surviving worker-scoped kernel still serves from its stale index — so on a co-located run they PASS VACUOUSLY against a corpus that no longer exists. This is a defect class across the phase-12 filesystem spec family, not a single-file bug, and the silent instances are worse than the loud one.

- timestamp: 2026-08-14T16:55Z
  checked: falsification test — the same two files at `--workers=1`
  found: 3 passed. As predicted by `ceil(2/1)=2` producing a single group, afterAll cannot fire between the two tests and the failure is impossible. The only variable changed was the worker count.
  implication: the mechanism is confirmed by both directions — it fires exactly when the group splits and never when it does not. No other explanation survives.

- timestamp: 2026-08-14T17:00Z
  checked: /tmp for leaked corpora, by prefix (`ls -d /tmp/topos-e2e-*`)
  found: 72 leaked directories — 13 `topos-e2e-fs`, 13 `-fs-add-source`, 13 `-fs-recursion`, 10 `-fs-includeglob`, 8 `-fs-external`, 8 `-fs-root-label`, 7 `zero-match`. Roughly ONE per spec file per suite run, including the single-test files that are immune to the group-splitting bug.
  implication: `test.afterAll` is failing to clean up on a path unrelated to job splitting — a second symptom of the same scope mismatch, present on every run, including runs that go green.

- timestamp: 2026-08-14T17:03Z
  checked: `npx playwright test --list e2e/specs/12-filesystem-recursion.spec.ts` (collection only, runs no test), counting corpora before and after
  found: 13 → 14. The collection pass creates a corpus and never removes it.
  implication: Playwright loads every spec file in a LOADER process to build the test tree. Module-scope side effects therefore execute there too — but no test and no afterAll ever runs in that process, so its corpus is orphaned unconditionally. This is the same root cause (setup bound to module/process lifetime, teardown bound to a test hook) presenting as a permanent leak rather than a failure, and it confirms that the correct teardown scope is the PROCESS, not any Playwright hook. A `describe.configure({ mode: 'serial' })` fix would have closed the ENOENT and left this leak untouched.

## Eliminated
<!-- APPEND ONLY -->

- hypothesis: a worker leaves the spec file, runs another file, then re-enters it with a warm module cache (the original v1 theory in Current Focus)
  evidence: each spec file's own `test.use({ configSpec })` gives it a distinct workerHash, and `Dispatcher._runJobInWorker` stops and recreates the worker whenever `worker.hash() !== job.workerHash`. Leaving the file therefore kills the process, so re-entry always gets a fresh module scope. The real arrangement needs no other file to run in between — only two jobs of the SAME file, back to back, in one worker.
  timestamp: 2026-08-14T16:22Z

- hypothesis: this is a regression from quick task 260814-mkl
  evidence: that task added an ambient type declaration only, and the defect reproduces deterministically from the spec's own structure with no reference to it. Per KB-005's precedent, a failure that is diff-independent is latent, not introduced.
  timestamp: 2026-08-14T16:35Z

- hypothesis: a timing race between the kernel's re-sync and the test's read
  evidence: the failure is a synchronous `unlinkSync` ENOENT on the test's own corpus path — no kernel call is involved at line 126, and the repro is deterministic rather than probabilistic. Nothing about sync timing participates.
  timestamp: 2026-08-14T16:35Z

## Resolution

root_cause: |
  A setup/teardown SCOPE MISMATCH in the phase-12 filesystem specs. Each spec seeds its temp
  corpus in module scope — correct, and required by D-03 (state must exist before kernel boot) —
  but tore it down in `test.afterAll`. Module scope runs once per Node PROCESS; `test.afterAll`
  runs once per Playwright JOB. Those are not the same unit, and the gap produced two distinct
  failures from one cause:

  (1) THE CI FAILURE. With `fullyParallel: true`, `createTestGroups` splits a file's tests into
      groups of `ceil(tests / workers)`, so at workers>=2 a two-test file becomes two jobs.
      afterAll fires at the end of EVERY job, and `Dispatcher._scheduleJob` PREFERS handing the
      next job back to a free worker whose workerHash already matches — the same process, whose
      module cache is warm, so module scope never re-seeds. Job 2 therefore ran against a corpus
      job 1 had deleted. This required two further conditions to present as it did (AND-gate):
      the dispatcher actually co-locating the two jobs, and the worker-scoped `kernel` fixture
      surviving with a stale index — which is why line 124's "item is present" assertion PASSED
      off that index and the first thing to notice was an ENOENT on `unlinkSync` at line 126.
      Only the scope mismatch is a defect; the other two are correct, deliberate behaviours
      (Playwright's dispatcher, and D-02's one-kernel-per-spec-file decision), and were left alone.

  (2) A PERMANENT LEAK, present on every run including green ones. Playwright loads every spec
      file in a collection process to build the test tree. Module scope runs there too, but no
      test — and therefore no afterAll — ever does, so that process's corpus was orphaned
      unconditionally. 72 such directories had accumulated in /tmp, ~7 per suite run.

  Local runs never showed either: a 12-core box resolves to 6 workers, so the two tests land in
  separate worker PROCESSES that each re-run module scope and own their own corpus and kernel.
  A CI runner resolving to 2 workers makes co-location the dispatcher's preferred outcome.

  NOT the cause: quick task 260814-mkl (the change was an ambient type declaration; the defect
  reproduces from the spec's structure alone and predates it). NOT a race — there is no timing
  window, only a scheduling arrangement, which is why the repro is deterministic.

fix: |
  Made the teardown scope match the setup scope. New `web/e2e/fixtures/corpus.ts` exports
  `mkdtempCorpus(prefix)`, which mkdtemps under the OS temp dir and registers a single
  `process.on('exit')` handler that `rmSync`s every corpus it created — `process.on('exit')`
  being the only Node hook whose lifetime is the process, which is the lifetime module-scope
  state actually has. All seven phase-12 filesystem specs now call it, and every
  `test.afterAll(() => rmSync(corpusDir, ...))` block is deleted. Nothing about what any spec
  proves changed, no parallelism was given up, and D-02/D-03/D-15 are all untouched — in
  particular no retry was added and no test was made serial.

  Recurrence guard: `web/e2e/specs/spec-hygiene.spec.ts`, a structural spec that fails if any
  spec file calls `mkdtempSync` directly or removes a directory inside a `test.afterAll` body
  (extracted by balanced-paren scan, not a regex that a parenthesised hook body would defeat).
  It boots no kernel. A structural gate is used because no runtime assertion can catch this
  class reliably: the failing arrangement is scheduling-dependent, and the sibling specs fail
  SILENTLY rather than loudly. Rule also written into docs/testing.md's "Writing a new spec".

oracle_type: derived — the guard asserts a contract about the spec tree ("state whose setup is
  process-scoped must have process-scoped teardown"), covering both halves of it (the creation
  path and the teardown path), which are the two adjacent ways to reintroduce the defect. The
  end-to-end oracle for the fix itself is `specified`: the original ENOENT must not recur under
  the arrangement that produced it.

verification: |
  guardrail_verdict: accepted

  signal 1 — bug returns on revert: PASS. Restoring the pre-fix spec from HEAD and re-running
    the repro command reproduced `ENOENT ... /tmp/topos-e2e-fs-recursion-w0Bh6l/receipts/
    nested-invoice.pdf` at line 126. The fix is causally responsible, not coincidental.
  signal 2 — regression test fails against unfixed code: PASS, 2/2. Against that same mutant,
    both hygiene tests failed and named the offending file exactly ("These specs call
    mkdtempSync directly: 12-filesystem-recursion.spec.ts" / "These specs remove a directory
    inside test.afterAll: 12-filesystem-recursion.spec.ts"). The guard cannot pass vacuously.
  signal 3 — not a deletion-only diff: PASS. The removed hook WAS the defect (justified by the
    RCA above), and it is replaced by a real mechanism, not by nothing: +1 new fixture module,
    +1 new guard spec, +1 narrowed ambient declaration.
  signal 4 — adjacent gates: PASS. `npm run check:e2e` clean; `npm run test` 999/999;
    `npm run check` 0 errors (10 pre-existing warnings, untouched); `make docs-check` all links
    resolve; full Playwright suite 122 passed at `--workers=2` (the CI-like arrangement that
    failed) AND at the default local worker count.
  signal 5 — stability: PASS. 8 consecutive runs of the exact repro command, 0 failures.
  leak check: PASS. A full 122-test run at `--workers=2` added ZERO temp corpora (74 before,
    74 after), and the collection-pass probe that previously leaked one directory per invocation
    now leaks none. This also closes the fix's main stated blind spot — that `process.on('exit')`
    might not fire inside a Playwright worker. It fires, in worker and collection processes alike.

files_changed:
  - web/e2e/fixtures/corpus.ts (new — mkdtempCorpus, process-lifetime cleanup)
  - web/e2e/specs/spec-hygiene.spec.ts (new — recurrence guard)
  - web/e2e/e2e-builtins.d.ts (narrow ambient decl for process.on('exit'))
  - web/e2e/specs/12-filesystem-recursion.spec.ts
  - web/e2e/specs/12-filesystem-tracer.spec.ts
  - web/e2e/specs/12-filesystem-add-source.spec.ts
  - web/e2e/specs/12-filesystem-root-label-match.spec.ts
  - web/e2e/specs/12-include-glob-metadata-preview.spec.ts
  - web/e2e/specs/12-zero-match-diagnostic.spec.ts
  - web/e2e/specs/12-external-rehearsal.spec.ts
  - docs/testing.md (rule added to "Writing a new spec")
