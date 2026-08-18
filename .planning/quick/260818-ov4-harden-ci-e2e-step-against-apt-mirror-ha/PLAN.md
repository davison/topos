---
quick_id: 260818-ov4
type: quick
autonomous: true
files_modified:
  - .github/workflows/ci.yml
---

<objective>
Harden CI against apt-mirror stalls. Two consecutive runs (32157487136 and
its predecessor, 2026-08-18 afternoon) hung >15 minutes inside Playwright's
`--with-deps` apt phase: the runner's default azure.archive.ubuntu.com
mirror failed repeatedly (Ign: spam), the fallback fetch stalled, and no
timeout existed anywhere to reap the hang. Everything before `make e2e` —
including the shutdown-reap fix's own test — was green in both runs; a CI
run on the same code path was green at 11:24 the same day, so the stall is
environmental (runner/mirror), not commit content.
</objective>

<tasks>
<task type="auto">
  <name>Task 1: mirror swap + apt bounds + step/job timeouts</name>
  <files>.github/workflows/ci.yml</files>
  <action>
    Add a "Harden apt against mirror stalls" step before make e2e
    (repoint azure mirror to archive.ubuntu.com in both sources formats,
    set Acquire retries/timeouts, pre-warm indices; timeout-minutes 5).
    Bound make e2e at timeout-minutes 10 and the job at 30. Comments name
    the incident and the quick task.
  </action>
  <verify>
    <automated>Push; observe the CI run either passes or fails fast in a named step — no silent hang past its bound.</automated>
  </verify>
</task>
</tasks>
