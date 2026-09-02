# M3: Daily-driver polish

Milestone issue [#72](https://github.com/davison/topos/issues/72); the
third milestone run under CodeCrew, opened 2026-09-01 at M2's close.
This record is the milestone's last task
([#81](https://github.com/davison/topos/issues/81)) — and like
[M1's record](1-plugin-repo-split-and-the-third-party-path.md), it
lands ahead of its tag: the QA verdicts are already on #72, but the
release gate — the operator's live UAT and the tag the commit log
derives ([docs/releasing.md](../releasing.md)) — follows this record's
merge. Every claim below links to the comment, PR or commit that
recorded it; where a choice clearly happened without a record, this
document says so rather than reconstructing a rationale.

## Goal and outcome

**Chartered:** what daily driving surfaced since v1.3.1
([#72](https://github.com/davison/topos/issues/72)): a webspace could
say *what* but not *when* — a holiday webspace drowns in past holidays
([#70](https://github.com/davison/topos/issues/70)); a webspace's name
was frozen at creation
([#71](https://github.com/davison/topos/issues/71)); the stream row
silently clipped its label pills (the leftover
[the M2 record](2-usability-and-quality-of-life.md) captured as
backlog, [#63](https://github.com/davison/topos/issues/63)); and two
pieces of hygiene — a plugin test suite that became a live OAuth flow
on the operator's own exported credentials
([tp#5](https://github.com/davison/topos-plugins/issues/5)) and the
fleet's verifier pin one release behind (the parity bump the M2 record
left open). Five requirements, all small-to-medium, all independently
landable; the larger platform surfaces
([#2](https://github.com/davison/topos/issues/2),
[#3](https://github.com/davison/topos/issues/3),
[#4](https://github.com/davison/topos/issues/4), with
[#1](https://github.com/davison/topos/issues/1) folded into #3's
future design) wait for the next milestone.

**Shipped:**

- A webspace narrows by date range
  ([#76](https://github.com/davison/topos/issues/76),
  [PR #79](https://github.com/davison/topos/pull/79)): two date
  pickers beside the search bar preview an unsaved range live
  (`?from`/`?to`, calendar dates resolved day-inclusive in
  kernel-local time); a set range promotes exactly as a search term
  does — the same *Save as filter*, `date_from`/`date_to` persisted
  beside `filter`/`filter_by_source`, one labelled removable date
  chip — and removal widens instantly, query-time-only like `filter`
  itself. The search fan-out's live source hits honour the range at
  the merge, live params can only narrow the saved range, and — after
  QA's finding — the agent mirror validates and honours live params
  exactly as its `/api` siblings
  ([#83](https://github.com/davison/topos/issues/83),
  [PR #84](https://github.com/davison/topos/pull/84)).
- A webspace's name can change after creation
  ([#77](https://github.com/davison/topos/issues/77),
  [PR #80](https://github.com/davison/topos/pull/80)): a Rename
  control beside each webspace's Delete in Manage Sources; one config
  write carries the map key rename alone, and the kernel migrates
  `webspace_items`, `item_marks` and the sync record in one
  transaction — items and exclusion marks survive with no resync, the
  old URL 404s, and the renamed space's own open tab lands on the new
  URL. A collision is refused client-side with the error surfaced.
- The stream row declares its overflow instead of clipping it
  ([#75](https://github.com/davison/topos/issues/75),
  [PR #78](https://github.com/davison/topos/pull/78)): the desktop
  meta strip is one non-wrapping line with a measured priority order
  (icon → matched-in/unsynced badges → date → pills); the pills clamp
  to a visible set plus a `+N` badge whose title names every hidden
  label; a visible pill gives way by truncation, never a half-cut
  crop; and — after QA's finding — the date collapses entirely before
  the declaration ever does, so the `+N` survives the narrowest
  pane-open band the layout produces
  ([#82](https://github.com/davison/topos/issues/82),
  [PR #85](https://github.com/davison/topos/pull/85)). The fixed row
  height (D-08) and the compact branch are untouched.
- Plugin test suites are hermetic on any machine
  ([tp#29](https://github.com/davison/topos-plugins/issues/29),
  [tp PR #31](https://github.com/davison/topos-plugins/pull/31)):
  gdrive's spawned-binary tests build the child environment by
  copying the parent's minus every `GDRIVE_*` entry — case-insensitively,
  for Windows — so the fail-loud dispatch/auth paths can never wander
  into a live OAuth flow on the operator's machine, with a regression
  test that exports fakes (and a lowercase alias) and proves the
  child sees none of them.
- The housekeeping: `TOPOS_PROVENANCE_REF` moves to the v1.3.1 tag so
  the fleet's release verifier is built at the kernel it actually
  pairs with ([tp#30](https://github.com/davison/topos-plugins/issues/30),
  [tp PR #32](https://github.com/davison/topos-plugins/pull/32)), and
  `.codecrew.yml` records that the qa seat's `~` routing is the
  solo-mode design — dispatched clean-context sessions under the
  operator's auth, the close verb's unrouted-seat note informational —
  in the PR that also carried M3's roadmap row
  ([#73](https://github.com/davison/topos/issues/73),
  [PR #74](https://github.com/davison/topos/pull/74)).

Eight tasks, eight PRs before this record — six on the kernel
repository and two on
[`topos-plugins`](https://github.com/davison/topos-plugins) — all
rebase-merged through `task finish` after a model review under the
reviewer contract in a headless Codex session, the solo-tier
convention M1 established and every PR's thread records (e.g.
[PR #78](https://github.com/davison/topos/pull/78#issuecomment-5503841842),
[tp PR #31](https://github.com/davison/topos-plugins/pull/31#issuecomment-5502785812)).
Two of the eight ([#82](https://github.com/davison/topos/issues/82),
[#83](https://github.com/davison/topos/issues/83)) are QA-remedy
tasks — the first milestone whose QA verdicts produced merged remedies
before its record, in the loop the requirement table retells.

## Requirement outcomes

Drawn from the qa seat's verdicts on #72
([the five-verdict comment](https://github.com/davison/topos/issues/72#issuecomment-5504128711);
[M3-R3's superseding verdict](https://github.com/davison/topos/issues/72#issuecomment-5504353106)).
Unlike M2's sandbox-blocked floor, this QA ran every floor by its own
hands on fresh clones of merged main — topos `make test` (14
packages) and `make e2e` (159 passed) green, topos-plugins
`make test` green with `GDRIVE_*` unset — with a kernel built from
main standing in for an installed release, the M1 precedent the
verdict names (no release carries this milestone yet), and
`gh codecrew milestone evidence 3` resolving every cited link.

| Requirement | Delivered by | Outcome |
|---|---|---|
| M3-R1 — a webspace narrows by date range, promotable like a term, inherited by the agent mirror | #76 / PR #79; remedy #83 / PR #84 | satisfied — one non-blocking finding: the agent stream silently ignored live `?from`/`?to` (malformed included) where [docs/api.md](../api.md) promised 400 by name ([finding on #76](https://github.com/davison/topos/issues/76#issuecomment-5504115466)), remedied by [PR #84](https://github.com/davison/topos/pull/84) |
| M3-R2 — a rename carries config key, index rows and URL, orphaning nothing | #77 / PR #80 | satisfied — QA's kernel-level gauntlet (the dead-namesake rename, rename-and-edit's bounded fallback, a collision-shaped write) held beyond the e2e suite's assumptions |
| M3-R3 — the stream row never silently clips its metadata | #75 / PR #78; remedy #82 / PR #85 | not satisfied on the first verdict — at pane-open viewports 768–~840px the pills *and* the `+N` marker clipped away silently, between spec 20's pinned widths ([finding on #75](https://github.com/davison/topos/issues/75#issuecomment-5504122595)); satisfied on the [superseding verdict](https://github.com/davison/topos/issues/72#issuecomment-5504353106) after PR #85 — a 21-width pane-open sweep from 768 to 1920, clean everywhere |
| M3-R4 — plugin test suites are hermetic on any machine | tp#29 / tp PR #31 | satisfied — including QA's own sweep for the shape the recorded grep could not catch (a child inheriting the environment implicitly via a nil `cmd.Env`): no other plugin test spawns a child at all |
| M3-R5 — post-release housekeeping: the verifier pin and the qa routing note | #73 / PR #74; tp#30 / tp PR #32 | satisfied — the pin proven from the built artifact (`go version -m` names v1.3.1), not the file alone |

## The decisions that shaped it

**The rename preserves history — and the heuristic's limit is stated,
not hidden.** `Supervisor.Apply` detects a rename (exactly one
webspace key vanished while exactly one appeared with an identical
body) and migrates the index rows, so items, exclusion marks and
saved filters all survive; rejected: clear-and-resync, because
exclusion marks are operator judgments a rename must not silently
reset. The trade-off is a detection heuristic that a simultaneous
rename-and-edit defeats — that write falls back to clear-and-resync
for the renamed key, stated in the code, and the rename dialog never
produces it
([Decision on #77](https://github.com/davison/topos/issues/77#issuecomment-5502894160)).
One letter of the Decision did not survive review: it placed the
migration "before the reconcile," and round 1 of the review proved
that slot wrong — a failed `Reconcile` would strand the restored old
runtime without its rows — so the shipped migration runs after a
successful `Reconcile`, a correction the approving review notes
against the older wording
([round 1](https://github.com/davison/topos/pull/80#issuecomment-5503662599),
[round 4](https://github.com/davison/topos/pull/80#issuecomment-5503913933)).

**The affordance is the capture's fallback, decided before code.**
Rename lives beside Delete in Manage Sources because in-place header
editing collides with the switcher dropdown — recorded in the task's
own goal, from [#71](https://github.com/davison/topos/issues/71)'s
capture, rather than as a Decision comment
([#77](https://github.com/davison/topos/issues/77),
[PR #80](https://github.com/davison/topos/pull/80)).

**The strip earned a priority order by measurement.** `clampLabels`
is a character budget, width-agnostic by design (the first label
always renders); the strip's survival order — icon, then the
matched-in/unsynced badges, then the date, then the pills — came from
a layout dump at the failing width (the ~100px strip's date alone
consumed 81px), not from intuition
([round-1 reply on PR #78](https://github.com/davison/topos/pull/78#issuecomment-5503604309)).
The recorded fallback — CSS-only truncation of the whole pill group,
the compact branch's behaviour promoted to desktop
([#75's plan](https://github.com/davison/topos/issues/75)) — stayed
in reserve, unneeded. The order reached its logical end in
[PR #85](https://github.com/davison/topos/pull/85): the date's shrink
floor is retired entirely, so under terminal pressure it ellipsises
away before the declaration ever moves — exactly where the recorded
priority puts it
([#82's plan](https://github.com/davison/topos/issues/82)).

**Dates, not datetimes; query-time, not sync-time; narrow-only,
live.** `date_from`/`date_to` are calendar dates resolved
day-inclusive in kernel-local time (DST-correct via next-local-midnight);
the range applies with `filter`'s query-time-only semantics — removal
widens instantly, nothing resyncs; live `?from`/`?to` can only narrow
the saved range, and promotion persists the saved∩live intersection,
never the raw live values
([PR #79](https://github.com/davison/topos/pull/79),
[round-1 finding](https://github.com/davison/topos/pull/79#issuecomment-5503642922),
[round-2 verification](https://github.com/davison/topos/pull/79#issuecomment-5503790690)).
A range saved together with terms or instance tokens lands in the
same single `putConfig`. That mixed-save shape is recorded in the PR
and its review, but the plan's promised Decision comment for it was
never posted on [#76](https://github.com/davison/topos/issues/76) — a
small recording gap this document names rather than fills.

**One offender, proven by sweep.** The plan's sibling sweep found
exactly one test file in any plugin spawning a child with
`os.Environ()` inherited — gdrive's own — and nothing that
legitimately requires wholesale inheritance
([Decision on tp#29](https://github.com/davison/topos-plugins/issues/29#issuecomment-5502696848)).
The scrub became case-insensitive under review, because Windows'
environment lookup is and a lowercase `gdrive_client_id` would have
survived the filter
([tp PR #31, round 1](https://github.com/davison/topos-plugins/pull/31#issuecomment-5502712666));
QA's verdict then swept for the shape the Decision's grep could not
see — implicit inheritance via a nil `cmd.Env` — and found no other
plugin test spawns a child at all
([M3-R4 verdict](https://github.com/davison/topos/issues/72#issuecomment-5504128711)).

**The unrouted qa seat is design, on the record.** A comment above
`qa:` in `.codecrew.yml` records that `~` is the solo-mode
arrangement — verdicts come from dispatched clean-context sessions
under the operator's auth, `milestone close` counts them as the
operator holding the seat, and the close verb's unrouted-seat note is
informational — mirroring the reviewer line's own comment, checked
against SPEC §5 by both the reviewing seat and QA
([#73](https://github.com/davison/topos/issues/73),
[PR #74](https://github.com/davison/topos/pull/74#issuecomment-5502705832),
[M3-R5 verdict](https://github.com/davison/topos/issues/72#issuecomment-5504128711)).

## Deviations, with their reasons

**The rebase unions damaged what they carried — recorded only by
their repairs.** The two feature branches landed in sequence onto a
main each had diverged from ([PR #80](https://github.com/davison/topos/pull/80)
even warned its reviewer it pre-dated #79's `StreamItems` change and
would "rebase mechanically"). The rebases did not stay mechanical:
resolving them left files syntactically broken — two dropped closers
in `format.test.ts` and `clampLabels`' closing brace in `format.ts`
on #76's branch
([b354f5f](https://github.com/davison/topos/commit/b354f5f),
[663f5c5](https://github.com/davison/topos/commit/663f5c5)), and
spliced helpers, interleaved describes and the un-widened
`StreamItems` arity across `config-edit.ts`, `config-edit.test.ts`
and `renamewebspace_test.go` on #77's
([8470677](https://github.com/davison/topos/commit/8470677)). All
five files were reconstructed in those three commits — titled
"rebase repairs" so the log says why they exist — before their PRs
merged. **The gap:** no Decision or Deviation comment on any task
records the incidents, what the union resolution did, or which gates
were rerun over the repairs; the commit titles are the only record
(undocumented deviation, inferred from the repair commits
themselves), and the doubled operator confirmation on
[PR #79](https://github.com/davison/topos/pull/79#issuecomment-5503933842)
is the visible trace of a finish run twice.

**Post-approval fixes rode to main without a review round.** The
union repairs above, and one more: CI's locale disagreed with the
suite's about a formatted date's ordering, and the chip-label
assertion stopped assuming one
([beead20](https://github.com/davison/topos/commit/beead20)). All
landed on #76's branch after the reviewer's round-2 approve; the
record's trace is the commits and the second operator confirmation,
nothing issue-side. Named here for the same reason as the unions: the
milestone's memory should not need archaeology to explain them.

## Why the record looks like this: the review loop

The three feature PRs took three, two and four CHANGES REQUESTED-inclusive
rounds; both QA remedies took two; the three that sailed through on
their first round were the tightly bounded ones — the docs-only
[PR #74](https://github.com/davison/topos/pull/74#issuecomment-5502705832),
the one-line pin
[tp PR #32](https://github.com/davison/topos-plugins/pull/32#issuecomment-5502712222),
and the one-route remedy
[PR #84](https://github.com/davison/topos/pull/84#issuecomment-5504192270).
As in M1 and M2, almost none of it was a failing test; each finding
was a premise the author held that a de-correlated reader — or a
browser driving the real thing — did not.

**PR #78, three rounds: the strip is measured, and Tailwind's
emission order is learned twice.**
[Round 1](https://github.com/davison/topos/pull/78#issuecomment-5503537661):
every badge was `shrink-0` inside an `overflow-hidden` strip, so at
the selected-pane widths the badges — the `+N` included — could still
clip silently; the spec's 1100px viewport bounded only the bottom
edge; and the helper tests never pinned the counting rule's
boundary. The answer replaced assumption with measurement — a layout
dump at the failing width produced the strip's priority order — and
recorded the first emission-order lesson: the pill region's shrink
override went inline *because Tailwind's emission order, not class
order, decides conflicts* against Badge's base `shrink-0`
([reply](https://github.com/davison/topos/pull/78#issuecomment-5503604309)).
[Round 2](https://github.com/davison/topos/pull/78#issuecomment-5503763041)
proved the lesson had not been fully applied to its own fix: the
visible pills themselves still carried the base `shrink-0` (the
utility class lost the same emission-order fight), so a
helper-classified "visible" pill could crop half-cut at the region's
edge — the exact silent-clipping class M3-R3 forbids — while the spec
bounded only the marker. The
[round-2 reply](https://github.com/davison/topos/pull/78#issuecomment-5503778287)
put the override inline on every pill ("the utility class provably
loses the emission-order fight … which round 1's history demonstrates
twice over") and the spec grew to bound every visible pill's box;
[round 3 approved](https://github.com/davison/topos/pull/78#issuecomment-5503841842).
By [#82's plan](https://github.com/davison/topos/issues/82) the rule
was "thrice-proven" and applied from the start.

**PR #79, two rounds: promotion must not widen, and the harness
leaked state.** Round 1's sharpest catch: promotion wrote the raw
live range as the new saved filter, so a one-sided live bound could
*widen* the permanent filter — remove a saved `date_to`, or replace a
saved bound with a wider one; the same round caught the empty-query
search path returning before validation (200 on a malformed param
where [docs/api.md](../api.md) promises 400 by name) and named the
spec's missing proofs
([round 1](https://github.com/davison/topos/pull/79#issuecomment-5503642922)).
The fix persists `intersectDateRanges(saved, live)` on both save
paths and validates before the fast path — and growing spec 21
surfaced a real harness trap: a sibling spec's promoted range leaked
through the worker-scoped kernel, made sticky by the new intersection
rule, so the spec resets deterministically via the config API and the
agent test owns a dedicated webspace
([reply](https://github.com/davison/topos/pull/79#issuecomment-5503730998)).
[Round 2 approved](https://github.com/davison/topos/pull/79#issuecomment-5503790690)
after its own boundary pass — DST-day correctness, the
zero-timestamp source hit, inverted saved ranges refused.

**PR #80, four rounds: the rename's failure story, hardened one
layer at a time.**
[Round 1](https://github.com/davison/topos/pull/80#issuecomment-5503662599):
three findings, all in the failure plane — a rename onto a *deleted*
webspace's name could merge that dead namesake's retained rows and
exclusion marks into the renamed space (config validation only sees
the current config); the migration ran before a `Reconcile` that can
fail, stranding the restored old runtime without its rows; and a
migration failure was logged-and-continued, reporting success over a
dropped mark-preservation guarantee.
[Round 2](https://github.com/davison/topos/pull/80#issuecomment-5503817006):
the corrected severity had introduced an early return that skipped
the cleanup/purge region and `commitGeneration` — stranding
removed-source rows and breaking the file's one-generation
invariant — so the error now travels through the repair region and
leads the joined apply failure
([reply](https://github.com/davison/topos/pull/80#issuecomment-5503848406)).
[Round 3](https://github.com/davison/topos/pull/80#issuecomment-5503877130):
the new failure-path test could not fail its own regression — the
fixture gave the repair region no observable work — fixed by removing
a source instance in the same failing write, whose
attempted-and-failed error must appear joined
([reply](https://github.com/davison/topos/pull/80#issuecomment-5503893903)).
[Round 4 approved](https://github.com/davison/topos/pull/80#issuecomment-5503913933),
noting one non-blocking honesty item this record keeps: spec 22's
collision test proves the surfaced refusal, not the "nothing
written" its title claims.

**The R3 loop: QA finds what the pins bracket.** Spec 20 was green
at its pinned widths (1100 unselected, 900 pane-open) while the
forbidden state lived between them: at pane-open viewports
768–~840px the pane clamp bottoms out at 240px, the row falls to
~216px, and pills *and* marker clipped away silently — QA's width
sweep found it, screenshot-confirmed, mechanism and reproduction
filed on the task
([finding](https://github.com/davison/topos/issues/75#issuecomment-5504122595)).
The remedy measured first (the ~73px strip cannot hold a 2rem date
floor and the 34px marker), retired the date's floor per the
recorded priority, and pinned QA's exact band in spec 20
([PR #85](https://github.com/davison/topos/pull/85)). Review round 1
caught the new test bounding the marker on only three edges — the
top was missing — and
[round 2 approved](https://github.com/davison/topos/pull/85#issuecomment-5504324680)
the four-edge bound; the
[superseding verdict](https://github.com/davison/topos/issues/72#issuecomment-5504353106)
then swept 21 pane-open widths from 768 to 1920, clean everywhere.
It is M2's e2e-catches-what-fakes-hide pattern moved one level up:
the suite catches what unit fakes hide, and QA's sweep catches what
the suite's pinned widths bracket.

**On the fleet:**
[tp PR #31's round 1](https://github.com/davison/topos-plugins/pull/31#issuecomment-5502712666)
caught the scrub being case-sensitive where Windows' environment
lookup is not — a lowercase `gdrive_client_id` inherited from the
parent would have survived the filter and still satisfied the
production `os.Getenv` — fixed with a normalized comparison and a
lowercase-alias regression;
[round 2](https://github.com/davison/topos-plugins/pull/31#issuecomment-5502785812)
independently re-ran the sweep grep before approving.

**And the qa seat itself** worked hairbrush-first, fixing nothing
and filing everything
([verdicts](https://github.com/davison/topos/issues/72#issuecomment-5504128711)):
R1's kernel probes went past the suite (day-inclusive boundaries
exact at the edge, four live widen attempts all refused, `scope=all`
source hits obeying the range at the merge, malformed and inverted
saved values refused by name); R2's gauntlet drove raw
`PUT /api/config` through the dead-namesake and rename-and-edit
edges the UI never produces; R5 proved the verifier pin from the
built artifact rather than the file. The blocking finding became
[#82](https://github.com/davison/topos/issues/82), the non-blocking
one [#83](https://github.com/davison/topos/issues/83), and both
remedies merged before this record.

## Left behind, deliberately

- The platform trio the charter deferred — in-app install-from-URL
  ([#2](https://github.com/davison/topos/issues/2)), a marketplace
  ([#3](https://github.com/davison/topos/issues/3)), kernel
  OAuth/secrets services
  ([#4](https://github.com/davison/topos/issues/4)) — with the
  certification capture
  ([#1](https://github.com/davison/topos/issues/1)) folded into #3's
  future design ([#72](https://github.com/davison/topos/issues/72)).
- An inverted *live* range (`from` > `to`) returns 200-empty — QA
  called it defensible but undocumented and filed nothing
  ([M3-R1 verdict](https://github.com/davison/topos/issues/72#issuecomment-5504128711));
  it remains undocumented at this boundary.
- Spec 22's collision test title promises "nothing written" that its
  assertions do not independently prove; the pre-request return is
  established by code inspection, accepted as non-blocking on the
  record ([round 4](https://github.com/davison/topos/pull/80#issuecomment-5503913933)).
- `docs/introduction.md` still does not exist — the landing-page
  obligation at this boundary falls entirely on
  [`README.md`](../../README.md), as it has every milestone so far.

## The release

M3's record lands ahead of its tag, the M1 shape rather than the M2
one. Once this document merges, the gate sequence runs on
[#72](https://github.com/davison/topos/issues/72): the operator's
live-instance UAT, the tag derived from the commit log since v1.3.1
per [docs/releasing.md](../releasing.md) — this record does not
pre-name the number; [M2's record](2-usability-and-quality-of-life.md)
says what pre-naming cost — then `milestone close 3`. The roadmap's
M3 row flips to Done with this record now and gains its release link
when the tag exists.
