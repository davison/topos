# M4: Protocol housekeeping

Milestone issue [#91](https://github.com/davison/topos/issues/91); the
fourth milestone run under CodeCrew, opened 2026-09-05 after M3's
release ([v1.3.2](https://github.com/davison/topos/releases/tag/v1.3.2),
2026-09-02). This record is the milestone's last task
([#96](https://github.com/davison/topos/issues/96)) — and unlike
[M1's](1-plugin-repo-split-and-the-third-party-path.md) and
[M3's](3-daily-driver-polish.md), it does not land ahead of a tag:
this milestone touched no kernel, web, plugin or SDK code, and the
derivation the release gate runs yields no release. Its PR is also the
first in this repository to *add* a milestone's `ROADMAP.md` row rather
than flip one — the convention M4 itself adopted, and this document is
where it first applies. Every claim below links to the comment, PR or
commit that recorded it; where a choice clearly happened without a
record, this document says so rather than reconstructing a rationale.

## Goal and outcome

**Chartered:** the housekeeping that the gh-codecrew v1.1.0→v1.2.0
update owed this hub before the platform trio
([#2](https://github.com/davison/topos/issues/2),
[#3](https://github.com/davison/topos/issues/3),
[#4](https://github.com/davison/topos/issues/4)) opens as M5
([#91](https://github.com/davison/topos/issues/91)). Three
requirements, all docs/config/CI, no kernel behaviour: the role
contracts reconcile with the v1.2.0 embedded ones, the codex-harnessed
reviewer seat pins its model, and commit messages are linted on every
PR so the version derivation in [docs/releasing.md](../releasing.md)
rests on a gate rather than review vigilance. The charter named its own
release outcome in advance — "expected tag derivation at the gate:
docs/chore/ci only → no release" — and the gate bore that out.

**The upstream that occasioned it.**
[gh-codecrew v1.2.0](https://github.com/radiusred/gh-codecrew/releases/tag/v1.2.0)
published 2026-09-05. Two of its changelog entries are what M4 adopts.
*"The ROADMAP row belongs to the doc-synthesizer at both ends"*
([gh-codecrew #208](https://github.com/radiusred/gh-codecrew/issues/208),
[PR #216](https://github.com/radiusred/gh-codecrew/pull/216)):
`milestone new` creates the tracking issue and nothing else — the local
append to `ROADMAP.md` and the "rides in this milestone's first PR"
line are gone, because the row had no PR to ride in when a milestone's
tasks all lived in spokes, a shape hit three times in the field
upstream. *"The codex seats are pinned to gpt-5.5"*
([gh-codecrew #226](https://github.com/radiusred/gh-codecrew/issues/226),
[PR #228](https://github.com/radiusred/gh-codecrew/pull/228)): upstream
moved its own codex seats off `gpt-5.6-sol` for cost on 2026-09-04, and
the dispatch guidance became "pass the model, never inherit the harness
default" ([gh-codecrew #212](https://github.com/radiusred/gh-codecrew/issues/212)).

**Shipped:**

- The three drifted contracts reconcile, and the reviewer seat pins its
  model ([#92](https://github.com/davison/topos/issues/92),
  [PR #94](https://github.com/davison/topos/pull/94)):
  `roles/implementer.md`, `roles/doc-synthesizer.md` and
  `roles/coordinator.md` take the v1.2.0 embedded text verbatim;
  `roles/reviewer.md` and `roles/qa.md` bodies already matched and took
  only their headers, so all five scaffold headers read `v1.2.0`; every
  `roles/*.local.md` extension is untouched
  ([8e3da52](https://github.com/davison/topos/commit/8e3da52)).
  `.codecrew.yml`'s reviewer row becomes
  `{ identity: ~, harness: codex, model: gpt-5.5 }`, its comment naming
  both dispatch flags
  ([eced14a](https://github.com/davison/topos/commit/eced14a)). Review
  round one then found the retired flow still alive in one live doc, so
  [docs/releasing.md](../releasing.md)'s Milestones section now says
  opening creates the issue and nothing else and closing's record PR
  adds the Done row
  ([25076e2](https://github.com/davison/topos/commit/25076e2)).
- Commit messages are linted on every pull request
  ([#93](https://github.com/davison/topos/issues/93),
  [PR #95](https://github.com/davison/topos/pull/95)):
  [`commitlint.config.mjs`](../../commitlint.config.mjs) at the repo
  root inherits `@commitlint/config-conventional` — the org action's
  config copied in, not referenced — with `body-max-line-length` and
  `footer-max-line-length` off and the file recording why; a third
  `ci.yml` job, id `commitlint`, name `Lint commit messages`, runs on
  pull requests only, ungated by `changes`, checking out full history
  and linting `base.sha..head.sha` with the CLI and preset pinned at
  21.2.2
  ([19a3cc3](https://github.com/davison/topos/commit/19a3cc3),
  [b5b7abf](https://github.com/davison/topos/commit/b5b7abf));
  [CONTRIBUTING.md](../../CONTRIBUTING.md) names the check and the
  local command, and [docs/releasing.md](../releasing.md)'s Versioning
  section says the lint gates the form of a type before the review
  judges its truth
  ([319c34b](https://github.com/davison/topos/commit/319c34b)).
- Two QA-remedy tasks, both on M4-R3. The lint's `type-enum` narrows to
  the eight types the derivation defines and CONTRIBUTING's local
  command pins the versions CI runs
  ([#98](https://github.com/davison/topos/issues/98),
  [PR #99](https://github.com/davison/topos/pull/99),
  [6553d35](https://github.com/davison/topos/commit/6553d35),
  [784cb25](https://github.com/davison/topos/commit/784cb25)); then the
  releasing doc's lint sentence names commitlint's own ignore list, and
  the config comment records `defaultIgnores` as inherited-and-kept
  ([#100](https://github.com/davison/topos/issues/100),
  [PR #101](https://github.com/davison/topos/pull/101),
  [903fc8a](https://github.com/davison/topos/commit/903fc8a)).

Four tasks and four PRs before this record — nine commits,
`639896b..903fc8a`, every one of them `docs:`, `docs(roles):`,
`docs(releasing):`, `chore:` or `ci:` — all rebase-merged through
`gh codecrew task finish --operator-confirm` after a model review under
the reviewer contract. Two of the four tasks
([#98](https://github.com/davison/topos/issues/98),
[#100](https://github.com/davison/topos/issues/100)) are QA remedies,
and the second remedies a finding on the re-verification of the first:
where [M3](3-daily-driver-polish.md) was the first milestone whose QA
verdicts produced merged remedies before its record, M4 is the first
whose remedy produced a remedy of its own.

## Requirement outcomes

Drawn from the qa seat's verdicts on
[#91](https://github.com/davison/topos/issues/91) — a clean-context
session under `roles/qa.md` posting under the operator's own auth, the
solo-mode arrangement `.codecrew.yml` records
([#73](https://github.com/davison/topos/issues/73), M3-R5). It ran
`gh codecrew milestone evidence 4` first, as #91's Gates require:
"requirements counted: M4-R1, M4-R2, M4-R3 (3) — all 0 cited links
resolve across 4 issues", nothing in this milestone's record having
cited a link for the verb to reject. Both verdict rounds re-ran the
floors by their own hands on fresh full clones of merged `main` —
`make docs-check` (433 links across 20 files), the split-claims sweep
(flagged: 0), and `gh codecrew roles diff` for all five roles — and
both say that green there is the verdict's first sentence, not its
evidence.

| Requirement | Delivered by | Outcome |
|---|---|---|
| M4-R1 — the role contracts reconcile with gh-codecrew v1.2.0: three bodies adopted, five headers at v1.2.0, `roles diff` clean, the decision recorded | #92 / PR #94 | **satisfied** ([verdict](https://github.com/davison/topos/issues/91#issuecomment-5554238086)) — QA diffed each `roles/<role>.md` against `gh codecrew roles show <role> --latest` itself: for all five the only difference is the two-line scaffold header, and all five read `v1.2.0`; `git diff 639896b..HEAD -- 'roles/*.local.md'` is empty; `git log 639896b..HEAD -- ROADMAP.md` is empty and `ROADMAP.md` carried no M4 row, so #91 really was opened without one. Two non-blocking findings on #92 (below) |
| M4-R2 — the reviewer seat pins `model: gpt-5.5`, and dispatch passes it explicitly rather than inheriting the harness default | #92 / PR #94 | **satisfied** ([verdict](https://github.com/davison/topos/issues/91#issuecomment-5554238086)) — the config half verified directly (`.codecrew.yml` parses; `status`, `role reviewer` and `roles show reviewer` all still work); the practice half evidenced only by the review comments on #94 and #95, each stating the session was dispatched with `-m gpt-5.5`. QA states what it could not verify: "nothing outside those comments attests the model a codex session actually ran under — the claim is the operator's, posted verbatim, and there is no artifact to check it against" |
| M4-R3 — commit messages are linted on every PR: a `Lint commit messages` job runs commitlint against an in-repo config, the departures recorded with their reason, CONTRIBUTING names the check | #93 / PR #95; remedies #98 / PR #99 and #100 / PR #101 | **satisfied** on the [first verdict](https://github.com/davison/topos/issues/91#issuecomment-5554238086) — all four clauses verified by QA's own execution, the corrected 22-of-60 / 12-of-60 measurement reproduced exactly, the six M4 commits then on `main` passing 0 problems each — with two findings filed on #93 that the verdict did not treat as blocking; **satisfied** again on the [superseding verdict](https://github.com/davison/topos/issues/91#issuecomment-5554335027) after PR #99 closed both, re-run on a new clone at `784cb25`: `perf: x`, `style: x` and `revert: x` each refused with `✖ type must be one of [feat, fix, docs, chore, test, ci, build, refactor] [type-enum]`, all eight named types and the breaking marker still passing, and CONTRIBUTING's command resolving `@commitlint/cli@21.2.2` from a pristine clone with an empty npm cache |

The superseding verdict carries one residue of its own — commitlint's
`defaultIgnores` letting four untyped header shapes through a doc
sentence that said none could
([finding on #98](https://github.com/davison/topos/issues/98#issuecomment-5554332766)).
It became [#100](https://github.com/davison/topos/issues/100) and
merged as [PR #101](https://github.com/davison/topos/pull/101). **The
gap:** no third verdict re-verifies M4-R3 after that remedy — the
superseding verdict predates #100 by minutes, and the only independent
check on PR #101 is its review round, which re-ran the probes and
approved
([round 1](https://github.com/davison/topos/pull/101#issuecomment-5554363299)).
The requirement's own four clauses are untouched by #101, which changed
a doc sentence and a comment and no rule.

## The decisions that shaped it

**Adopt the three drifted contracts verbatim; pin only the reviewer
row.** The contracts are this project's fork, so each changed passage
was read against what this hub actually does before adopting, and none
conflicted: [#91](https://github.com/davison/topos/issues/91) had been
created with no ROADMAP row (the verb already behaved the new way),
`ROADMAP.md`'s Status column — "Done — record, release" rather than the
bare record-linking `Done` the upstream bullet spells — is a local shape the
doc-synthesizer's "Add the ROADMAP row" bullet fits, and the
coordinator's two new notes (collision renumbering, milestone-level
`checkpoint`) describe verbs this hub already runs. Local conventions
stay in the `.local.md` extensions, untouched.
**Rejected:** pinning a model on the `qa` row — it declares no harness,
so there is no default for a pin to override; and keeping the v1.1.0
"The ROADMAP row is yours" bullet as a local convention, because it
named a behaviour `milestone new` no longer has
([Decision on #92](https://github.com/davison/topos/issues/92#issuecomment-5554073841)).
The model itself is the operator's word, recorded as such: gpt-5.5,
2026-09-05, following upstream's own re-pin for cost the day before.

**Copy the org's commitlint config into the repo, and switch off the
two rules the house style trips.**
[`commitlint.config.mjs`](../../commitlint.config.mjs) `extends`
`@commitlint/config-conventional` with `body-max-line-length` and
`footer-max-line-length` at `[0]`. The house style writes commit bodies
as unwrapped paragraphs, which `git log` and GitHub render fine and
which nothing in the version derivation reads; the header limit, the
type list and the subject rules stay, because those are exactly what
the derivation reads. The lint applies only to a PR's own commits, so
history is untouched, and the operator can restore any rule by editing
one file.
**Rejected:** the org's composite action or `wagoid/commitlint-github-action`
directly — the operator asked for the config copied rather than
referenced, and `ci.yml`'s header documents that every step invokes an
official `actions/*` action only, which a pinned `npm install` keeps
true; also rejected, a root `package.json` for the two packages
([Decision on #93](https://github.com/davison/topos/issues/93#issuecomment-5554094193)).

**The measurement behind that trade-off was wrong, and the correction
is on the record beside it.** The Decision above was first written
against "50 of the last 60 commits fail config-conventional as-is".
Review round one on [PR #95](https://github.com/davison/topos/pull/95)
could not reproduce it, and the re-measurement per commit with the
pinned 21.2.2 CLI over `639896b~60..639896b` explains why: the "50" had
counted commitlint's `✖` marker lines — one per rule plus one summary
per failing commit — from an older CLI, not commits.

| config | failing commits | by rule |
|---|---|---|
| unmodified `config-conventional` | 22 of 60 | 16 `body-max-line-length`, 12 `header-max-length`, 6 both; `footer-max-line-length` 0; 4 warning-only `footer-leading-blank` |
| `commitlint.config.mjs` | 12 of 60 | all `header-max-length`, at 101–135 characters |

The decision stands on the corrected numbers — the body rule is the one
the house style trips, and the footer rule is off for the same reason,
a trailing paragraph parsing as a footer — and the config comment, the
`ci:` commit body and the PR body were all rewritten to carry them
([Decision (correction) on #93](https://github.com/davison/topos/issues/93#issuecomment-5554134455)).
QA reproduced both rows exactly, and the twelve failing header lengths
individually
([QA on #93](https://github.com/davison/topos/issues/93#issuecomment-5554228691)).

**When the gate and the derivation disagree, narrow the gate.** QA
found `perf:`, `style:` and `revert:` passing the lint while
[docs/releasing.md](../releasing.md)'s derivation and
`roles/implementer.local.md` define a consequence for none of them —
so the shipped sentence claimed a property the gate did not have. The
remedy narrowed `type-enum` to the eight types the derivation defines
rather than widening the derivation to say what the other three mean:
the derivation rule is the operator's, decided at the M2 release gate
([#64](https://github.com/davison/topos/issues/64)), and changing what
a type means to the version is a policy change, not a QA remedy. The
cost is stated in the config comment — a contributor reaching for
`perf:` is refused until the derivation says what it means, so the
refusal points at the right place.
**Rejected:** weakening the sentence alone, which would leave three
types landing with no defined consequence (QA's actual finding); and
adding the three to the derivation as no-bump — plausible for `style`,
wrong-by-default for `perf`, whose change can carry behaviour, and the
operator's call in the derivation's own section if wanted
([Decision on #98](https://github.com/davison/topos/issues/98#issuecomment-5554250894)).
The same Decision records what it declined to remedy: `main` carries no
`required_status_checks`, so the lint gates merges made through
`task finish` rather than GitHub's own button — noted, not fixed,
because every merge in this hub goes through the verb and a ruleset
entry is an operator setting, not a commit.

**The doc says what the gate does, and the gate keeps `git revert`'s
own header.** The last remedy chose the same way one level down: rather
than set `defaultIgnores: false`, which would refuse the header
`git revert` writes for itself while `revert:` is not a type the
derivation defines, the releasing doc's sentence names the exception and
the config comment records `defaultIgnores` as inherited and why it
stays. **The gap:** [#100](https://github.com/davison/topos/issues/100)
carries no `**Decision:**` comment — its reasoning lives in the issue
body's Goal and Plan and in
[PR #101](https://github.com/davison/topos/pull/101)'s description,
which say exactly this, but the protocol's recording point was not
used. Named here rather than filled; it is the same shape M3's record
named for the rename affordance, and the same feedback on protocol
discipline.

## The deviation, and its correction

**The lint's packages install at the workspace root, not the runner's
temp directory.** The plan had the job `npm install` into
`$RUNNER_TEMP`; the shipped job uses `--prefix "$GITHUB_WORKSPACE"`.
The reason is commitlint's own resolution order: `resolve-extends` looks
for a preset from the working directory first and the npm cache second,
never from the CLI's own install directory, so a temp-prefix install is
invisible to it. It had passed locally only because an earlier
`npx --package` run had left the preset in that machine's npm cache —
which is also, by design, why the documented `npx` form in CONTRIBUTING
works. A root `node_modules/` is already gitignored and `--no-save`
keeps `package.json` absent, so the workspace stays as the plan
described it
([Deviation on #93](https://github.com/davison/topos/issues/93#issuecomment-5554107514)).

**It failed twice, not once, and the second was a rewrite regression.**
The Deviation and
[PR #95](https://github.com/davison/topos/pull/95)'s body both say "the
job's first run … failed". QA read `ci.yml` at each of the four shas the
check ran on and found the fuller shape
([finding 3 on #93](https://github.com/davison/topos/issues/93#issuecomment-5554228691));
the check runs confirm it:

| sha | `Lint commit messages` | why |
|---|---|---|
| `168f88f` | [failure](https://github.com/davison/topos/actions/runs/33985871541/job/101359131984) | `MODULE_NOT_FOUND` for `@commitlint/config-conventional`, resolved from the workspace against a `$RUNNER_TEMP` install |
| `d4a26e0` | [success](https://github.com/davison/topos/actions/runs/33985986053/job/101359437823) | carried `--prefix "$GITHUB_WORKSPACE"` |
| `17adfbb` | [failure](https://github.com/davison/topos/actions/runs/33986194086/job/101360006111) | the round-one branch rewrite, which carried the corrected measurement into the `ci:` commit body, went back to `--prefix "$RUNNER_TEMP/commitlint"` and dropped the fix |
| `ee15c1e` | [success](https://github.com/davison/topos/actions/runs/33986209921/job/101360050582) | the workspace-root install re-applied |

The rewrite that fixed the record broke the fix the record described —
the cost of rewriting a branch for the sake of an honest commit body,
paid once and worth the record getting right.

## Why the record looks like this: the review loop

Four PRs, six rounds: [#94](https://github.com/davison/topos/pull/94)
two, [#95](https://github.com/davison/topos/pull/95) two,
[#99](https://github.com/davison/topos/pull/99) one,
[#101](https://github.com/davison/topos/pull/101) one. Every one of the
six is a clean-context codex session loaded with
`gh codecrew roles show reviewer` and dispatched with `-m gpt-5.5`,
whose text the operator posted verbatim on the PR and then confirmed as
both author and operator (pure solo tier, SPEC §6) — the convention M1
established. All six are dispatches under the pin, the first of them on
[PR #94](https://github.com/davison/topos/pull/94), the PR that
introduced it
([round 1](https://github.com/davison/topos/pull/94#issuecomment-5554090583):
"the first dispatch under the pin this PR introduces").
That the model was actually passed is attested by those comments and
nothing else; QA says so, and this record repeats it rather than
implying enforcement.

Both PRs that took two rounds were caught by a round-one finding that
was real, and neither was a failing test — the same pattern M1, M2 and
M3 record: a premise the author held that a de-correlated reader did
not.

**PR #94, round one: the adopted contract and the repo's own
instructions contradicted each other.** The PR replaced three role
contracts with text saying nothing writes the ROADMAP row before the
record PR — and left
[docs/releasing.md](../releasing.md):19 documenting the old
consequence, that `milestone new` writes the row locally and that edit
rides in the first PR. "Leaving the release/process doc with the old
command consequence makes the repo's operator instructions contradict
the adopted role contract"
([round 1](https://github.com/davison/topos/pull/94#issuecomment-5554090583),
REQUEST CHANGES). The Milestones section was rewritten on both ends —
Opening creates the issue and nothing else, Closing's record PR adds
the Done row
([reply](https://github.com/davison/topos/pull/94#issuecomment-5554092416))
— and
[round 2](https://github.com/davison/topos/pull/94#issuecomment-5554108990)
approved after re-running `roles diff` for all five roles and checking
the adopted bodies against `roles show --latest` itself. QA's later
sweep is what closes this thread: grepping every Markdown file outside
the frozen `.planning/` archive for "rides in", "milestone new",
"ROADMAP row" and "Flip the ROADMAP", `docs/releasing.md` was the
**only** live doc still describing the retired v1.1.0 flow; what
remains is the adopted contracts themselves and the M1/M2/M3 records
describing what happened at the time, which is history, not instruction
([M4-R1 verdict](https://github.com/davison/topos/issues/91#issuecomment-5554238086)).

**PR #95, round one: a recorded measurement that would not
reproduce.** M4-R3 requires the config's departures to be recorded
*with their reason*, so the supporting number is part of the
requirement, not decoration. The reviewer re-ran the intended range
with the pinned CLI, counted 22 failing commits where the config
comment claimed 50, and asked for the config comment, the task
Decision, the PR body "and the `ci:` commit body if history is being
kept clean" to be corrected
([round 1](https://github.com/davison/topos/pull/95#issuecomment-5554125602),
REQUEST CHANGES). All four were — the branch rewritten so the `ci:`
commit body carries the right numbers, at the cost recorded in the
deviation above
([reply](https://github.com/davison/topos/pull/95#issuecomment-5554135575))
—
and [round 2](https://github.com/davison/topos/pull/95#issuecomment-5554158592)
approved after reproducing the corrected table itself, down to the four
warning-only `footer-leading-blank` results.

**The two remedies passed first time**, each on a bounded change the
reviewer verified by its own probes: PR #99's narrowed type list
([round 1](https://github.com/davison/topos/pull/99#issuecomment-5554263068),
APPROVE, no findings) and PR #101's doc sentence, where the reviewer ran
the ignore-list shapes through the pinned CLI, checked that
`defaultIgnores: false` would indeed reject them, and confirmed `docs:`
was honest for a docs-and-comment-only change
([round 1](https://github.com/davison/topos/pull/101#issuecomment-5554363299),
APPROVE). Both are the shape M3's record already named: the tightly
bounded PRs are the ones that sail through.

## Why the record looks like this: the QA loop

The qa seat worked hairbrush-first across two sessions, fixing nothing
and filing everything, and its probes are most of what this record can
say about what the shipped work actually does.

**Session one** ([verdicts](https://github.com/davison/topos/issues/91#issuecomment-5554238086),
findings on [#92](https://github.com/davison/topos/issues/92#issuecomment-5554228605)
and [#93](https://github.com/davison/topos/issues/93#issuecomment-5554228691),
from a fresh clone at `b5b7abf`) filed six things and one out-of-scope
capture. Two became remedy work:

- `docs/releasing.md`'s new sentence over-claimed what the lint gates —
  `perf:`, `style:` and `revert:` all pass while the derivation defines
  nothing for them. Became [#98](https://github.com/davison/topos/issues/98).
- CONTRIBUTING's documented local command resolved `@commitlint/cli`
  **unpinned** while CI pins 21.2.2 — "the moment 22.x ships they
  diverge, and a contributor's local green will not mean CI green",
  precisely the failure mode that produced the withdrawn "50 of 60".
  Also [#98](https://github.com/davison/topos/issues/98).

Two more corrected this record rather than the code: the deviation
failed twice, not once (above), and the "`status` hides the drift"
shorthand #96's plan carried does not reproduce (below). Two are
upstream shape, not this milestone's to fix, and the operator said so
explicitly
([reply on #92](https://github.com/davison/topos/issues/92#issuecomment-5554242623)):

- **`gh codecrew roles diff` compares bodies only, not the scaffold
  header.** QA set `roles/qa.md`'s header to `codecrew v1.1.0` with the
  body untouched; `roles diff qa` still reported "matches the embedded
  v1.2.0 contract" and `status` reported no drift. M4-R1 asks for two
  things — the bodies adopted *and* the headers at v1.2.0 — and #91
  names `roles diff` as the gate for both. The shipped work is correct
  because QA read all five headers, not because the gate proved it; a
  future header bump could be skipped with every check staying green.
- **Nothing validates the `model` pin.** `modl: gpt-5.5` and
  `model: not-a-real-model` are both accepted silently by `role`,
  `roles show` and `status`, all exiting 0, and no verb in
  `gh codecrew help` surfaces the model at all; breaking the YAML
  outright makes `status` print a parse error and **still exit 0**. The
  pin is durable documentation for the dispatcher and nothing more —
  "a defensible design for a solo hub, but the record should say so
  rather than imply enforcement."

**Session two** ([superseding M4-R3 verdict](https://github.com/davison/topos/issues/91#issuecomment-5554335027),
finding on [#98](https://github.com/davison/topos/issues/98#issuecomment-5554332766))
re-verified from a **new** full clone at `784cb25` — no reuse of the
first clone or its `node_modules` — plus a second pristine clone with an
empty npm cache for the `npx` check, and closed both findings by hand.
Then it found the residue: commitlint's `defaultIgnores` skip
`Revert "…"`, `Merge …`, `fixup!` and `squash!` headers *before any rule
runs*, so `Revert "total garbage with no type at all"` exits 0 where a
bare `wip` correctly fails on `type-empty` — while the sentence PR #99
had just shipped said a commit "with no type or subject at all, never
reaches `main`". Two of those shapes are live here: `revert: x` is
refused while the header `git revert` writes for itself is not, and
`task finish` rebase-merges without autosquash, so a forgotten `fixup!`
lands untyped with the gate green. The finding was explicitly not
prescriptive about the fix, and named both options with the rule that
decides between them — "widening starts in the derivation", the rule
[#98's Decision](https://github.com/davison/topos/issues/98#issuecomment-5554250894)
had itself established. [#100](https://github.com/davison/topos/issues/100)
took the doc option.

**The capture QA filed against nothing in M4:**
[#97](https://github.com/davison/topos/issues/97) — the push-to-`main`
CI run for `b5b7abf`
([run 33986504938](https://github.com/davison/topos/actions/runs/33986504938))
failed on one e2e spec while 160 passed. Neither #94 nor #95 touches
`kernel/`, `web/`, `plugins/`, `sdk/` or `proto/`, and the same spec
passed 6/6 from a fresh clone of the same tree, so it reads as a
runner-side flake; a `gh run rerun --failed` with no tree change came
back green
([comment](https://github.com/davison/topos/issues/97#issuecomment-5554283809)).
A second occurrence the same day on
[PR #99](https://github.com/davison/topos/pull/99) — a *different*
spec, `uat-02-remove-source-items.spec.ts:38`, same shape — turned the
capture from one spec into a class: `toHaveCount`/visibility waits
pinned at Playwright's 15s expect timeout on cold runners
([comment](https://github.com/davison/topos/issues/97#issuecomment-5554284811)).
It is deliberately not a sub-issue of #91, so it does not gate
`milestone close 4`; it sits on the release gate, whose step 1 is
"confirm the portable gate is green … or check the latest `ci.yml` run
on `main`", which is why the next tag needs it decided.

**A process incident, recorded on the issue it happened to.** At 19:45Z
#97's body was overwritten and its task started by mistake — a wrong
issue-number lookup while planning #100. The body was restored from the
edit history to the qa seat's original text, the accidental branch
deleted and the assignment removed, with nothing else changed
([operator note](https://github.com/davison/topos/issues/97#issuecomment-5554343183)).
The incident is in the record because the recovery was: an issue body is
a protocol artifact, and its edit history is what made the restore
possible.

## What M4 changed about how this hub runs

- **The ROADMAP row is added Done by the record PR, and nothing writes
  it earlier.** M1's, M2's and M3's rows were all created `Open` — M3's
  by hand in a mid-milestone PR
  ([d58c24e](https://github.com/davison/topos/commit/d58c24e),
  [#73](https://github.com/davison/topos/issues/73)) — and flipped to
  Done by their record PRs
  ([d0ef57d](https://github.com/davison/topos/commit/d0ef57d),
  [dab702b](https://github.com/davison/topos/commit/dab702b),
  [9e44d23](https://github.com/davison/topos/commit/9e44d23)).
  [#91](https://github.com/davison/topos/issues/91) was opened with no
  row at all, and this document's PR adds M4's — the first row in this
  repository added rather than flipped. `ROADMAP.md` now lists finished
  milestones; `gh codecrew status` reports the open one.
- **The reviewer seat's model is written down.** A dispatch passes
  `-m gpt-5.5` and never inherits the codex default; the row's comment
  also names the sandbox network flag a seat that must reach GitHub
  takes. It binds the dispatcher, not the CLI — see QA's second finding
  above.
- **Every PR is commit-lint-gated.** `Lint commit messages` runs on
  every pull request, ungated by the `changes` filter that lets
  docs-only work skip the heavy gate, and reports `skipped` — not
  failed, not absent — on push-to-`main` runs, which have no PR range.
- **The derivation's eight types are now the lint's.** After
  [#98](https://github.com/davison/topos/issues/98), three lists say
  the same eight words: `commitlint.config.mjs`'s `type-enum`,
  [docs/releasing.md](../releasing.md)'s derivation bullets, and
  `roles/implementer.local.md` — with `roles/reviewer.local.md`'s
  no-consequence list matching the six. Widening any of them starts in
  the derivation, which is the operator's.
- **What the gate still does not reach**, stated so nobody mistakes it
  for coverage: the type's *truth* (only the review judges whether a
  `fix:` is really a `feat!:`); commitlint's own ignore list; and
  merges made through GitHub's button rather than `task finish`, since
  `main`'s ruleset carries no `required_status_checks`.

## The release: none

The gate ran the derivation in [docs/releasing.md](../releasing.md) over
the log since the last tag, and it yields nothing:

```
$ git log v1.3.2..main --pretty=%s
docs: the lint sentence names commitlint's own ignore list (#100)
docs: the lint sentence claims what the lint checks, and the local command is pinned (#98)
ci: the lint's type list is the derivation's eight types (#98)
ci: the lint's packages install at the workspace root, where extends resolves (#93)
docs: the contributing guide and releasing name the commit lint (#93)
ci: commit messages are linted on every pull request (#93)
docs(releasing): opening a milestone writes no ROADMAP row (#92)
chore: the reviewer seat pins gpt-5.5 (#92)
docs(roles): the contracts reconcile with gh-codecrew v1.2.0 (#92)
docs: the roadmap's M3 row links the release that shipped (#89)
```

No `feat:`, no `fix:`, no breaking marker: "a log holding only `docs:`,
`chore:`, `test:`, `ci:`, `build:` or `refactor:` commits bumps nothing
on its own — there is no new tag to cut until behaviour changes." The
charter predicted exactly this
([#91](https://github.com/davison/topos/issues/91)), and no UAT was
raised, because nothing here changes kernel or web behaviour. `v1.3.2`
remains the shipped release at this boundary, and the roadmap's M4 row
says so.

One line of #91's Gates did not survive contact with its own
milestone: "the M4 record's PR is the first to land under it [the
lint]". Written when M4 had two tasks, it was overtaken by the two QA
remedies — [PR #95](https://github.com/davison/topos/pull/95) landed
under the check it added, and
[PR #99](https://github.com/davison/topos/pull/99) and
[PR #101](https://github.com/davison/topos/pull/101) landed under it
after that. This record's PR is the fourth to be gated, not the first.
The gate's substance — the check running green on its own PR — held.

## Left behind, deliberately

- **The platform trio, now M5's charter**: in-app install-from-URL
  ([#2](https://github.com/davison/topos/issues/2)), a marketplace
  ([#3](https://github.com/davison/topos/issues/3)) and kernel
  OAuth/secrets services
  ([#4](https://github.com/davison/topos/issues/4)), with the
  certification capture
  ([#1](https://github.com/davison/topos/issues/1)) folded into #3's
  design — deferred by M3 and named again by
  [#91](https://github.com/davison/topos/issues/91) as what M4 was
  clearing the way for.
- **The e2e flake class**
  ([#97](https://github.com/davison/topos/issues/97)) — two different
  specs failing at the 15s expect timeout on cold CI runners, both
  green on re-run, characterised but not fixed. Open, not a sub-issue
  of any milestone, and sitting on the release gate: whoever cuts the
  next tag reads step 1 against a run that may be red for this reason.
- **Two upstream-shaped captures for
  [radiusred/gh-codecrew](https://github.com/radiusred/gh-codecrew),
  not yet filed there**: `roles diff` comparing contract bodies without
  the scaffold header, and the `model:` key being unvalidated and
  unsurfaced by every verb — both noted for the operator on
  [#92](https://github.com/davison/topos/issues/92#issuecomment-5554242623)
  as upstream's, not M4's. At the time of writing neither exists as an
  issue on that repository.
- **The upstream `status` drift bug**, restated precisely because
  #96's plan carried it in a shorthand QA disproved. The observation
  was made in the operator's session on 2026-09-05 **before
  [#91](https://github.com/davison/topos/issues/91) existed**: with M3
  closed and no open milestone, `gh codecrew status` printed only "no
  open milestones in davison/topos" while `roles diff` showed drift on
  three roles. QA's reproduction ran with M4 open, and there `status`
  *does* print three `contract drift:` lines (implementer 6,
  doc-synthesizer 10, coordinator 17 drift lines; reviewer and qa 0,
  confirming their bodies already matched). So the bug is narrower than
  "`status` hides the drift": `status` returns early on "no open
  milestones" before it reaches the drift report. The operator reports
  it confirmed upstream with a fix due in the next version; that report
  is the only record of the confirmation
  ([reply on #92](https://github.com/davison/topos/issues/92#issuecomment-5554242623)).
- **[#100](https://github.com/davison/topos/issues/100)'s missing
  Decision comment** and the fact that no QA verdict covers the state
  of the tree after [PR #101](https://github.com/davison/topos/pull/101)
  — both named above, both left as they are rather than reconstructed
  after the fact.
- **`docs/introduction.md` still does not exist.** The
  doc-synthesizer's v1.2.0 contract asks for it to be refreshed at every
  milestone boundary alongside the README; in this repository the
  landing-page obligation continues to fall entirely on
  [`README.md`](../../README.md), as it has since M1.
