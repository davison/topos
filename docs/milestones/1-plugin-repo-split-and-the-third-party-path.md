# M1: Plugin repo split and the third-party path (v1.3.0)

Milestone issue [#6](https://github.com/davison/topos/issues/6); the
first milestone run under CodeCrew, opened 2026-08-31 at the migration
this repository's [genesis record](0-genesis-the-gsd-era.md) describes.
This record is the milestone's last task: it merges ahead of the tag,
the operator's live UAT and the close verb, in the sequence #6 records
as each happens. Every claim below links to the comment, PR or
issue that recorded it; where a choice clearly happened without a
record, this document says so rather than reconstructing a rationale.

## Goal and outcome

**Chartered:** the remainder of the GSD-era v1.3.0 milestone — every
functional plugin moves out of the kernel repository into
[`topos-plugins`](https://github.com/davison/topos-plugins) with its
own CI and releases, backed by the provenance trust phase 16 shipped;
independent distribution, a one-command pull-by-URL install, and a
plugin developer guide make the kernel↔plugin boundary the operator's
own sources cross the same one a third-party author crosses
([#6](https://github.com/davison/topos/issues/6)).

**Shipped:** two repositories with one boundary between them.

- The kernel repository holds the kernel, the sdk, the contract, and two
  mock plugins — nothing per-plugin
  ([#13](https://github.com/davison/topos/issues/13)). Its releases
  ship the kernel and the provenance verifier
  ([#15](https://github.com/davison/topos/issues/15)).
- `topos-plugins` holds all seven functional plugins as per-plugin Go
  modules under one workspace
  ([tp#1](https://github.com/davison/topos-plugins/issues/1),
  [tp#3](https://github.com/davison/topos-plugins/issues/3)), with CI,
  a tag-triggered signing release (v0.1.0, v0.2.0, v0.3.0), and its
  own `make install`/`uninstall` for the fleet
  ([tp#8](https://github.com/davison/topos-plugins/issues/8)) plus
  `make install-signal` for the one cgo plugin
  ([tp#12](https://github.com/davison/topos-plugins/issues/12)).
- The kernel refuses incompatibility by name — a handshake or contract
  generation mismatch is a per-source launch failure on the chip, never
  a dead boot ([#17](https://github.com/davison/topos/issues/17)).
- `topos plugin pull <url>` takes one plugin from a URL to the tier its
  provenance earns ([#19](https://github.com/davison/topos/issues/19)),
  and [`docs/plugin-development.md`](../plugin-development.md) walks a
  stranger from an empty module to that command, validated clean-room
  ([#21](https://github.com/davison/topos/issues/21)).

Thirteen tasks, thirteen PRs, all rebase-merged through `task finish`
after a model review under the reviewer contract in a headless Codex
session — the solo-tier convention this milestone established and kept
(every PR's review thread records it; e.g.
[PR #14](https://github.com/davison/topos/pull/14),
[PR #9](https://github.com/davison/topos-plugins/pull/9)).

## Requirement outcomes

<!-- QA verdicts: the qa seat's verdict comments on #6 -->

| Requirement | Delivered by | Outcome |
|---|---|---|
| M1-R1 — all seven plugins build and run from topos-plugins | tp#1, tp#3 | satisfied |
| M1-R2 — the kernel retains only the mocks and no per-plugin knowledge | #13; then #24, tp#14, #26, #28, #30, #32, #34 after QA | satisfied — on the sixth verdict, after five not-satisfied passes each remedied; the last under a resolved gate on #6 |
| M1-R3 — the side-by-side dev loop | #13 | satisfied |
| M1-R4 — topos-plugins' own CI/release pipeline | tp#1, tp#3, tp#8, tp#10 | satisfied |
| M1-R5 — install and update plugins independently of the kernel, and vice versa | tp#8, #15 | satisfied |
| M1-R6 — incompatibility surfaces loudly by name | #17, tp#10 | satisfied |
| M1-R7 — Signal's local cgo build path survives the move | tp#12 | satisfied |
| M1-R8 — one command from URL to tier; failed verification places nothing | #19 | satisfied |
| M1-R9 — a developer guide from contract and mocks to an installable out-of-repo plugin | #21 | satisfied |

Three earlier tasks carried no requirement of their own but unblocked
the rest: the genesis record ([#7](https://github.com/davison/topos/issues/7)),
and two CI repairs adopted at the migration
([#9](https://github.com/davison/topos/issues/9),
[#11](https://github.com/davison/topos/issues/11)).

## The decisions that shaped it

Each entry names the choice, its trade-off, and what was rejected, and
links the comment that recorded it.

**The split itself: tracer first, clean copies, per-plugin modules.**
The filesystem plugin moved alone first to prove the pattern
(no cgo, no external service), as a clean copy from
`davison/topos@d9a37b1` with a moved-from note rather than grafted
history, each plugin its own Go module under one `go.work`
([tp#1's charter](https://github.com/davison/topos-plugins/issues/1)).
The other six followed the proven pattern; gdrive folded in from its
own repository as a Go module only, its clean-room scaffolding left
behind and that repository archived
([tp#3](https://github.com/davison/topos-plugins/issues/3)). The
kernel-side removal was chartered "repoint-or-drop per the requirement's
letter": generic UI specs repointed at the mocks, plugin-specific specs
dropped with their subjects, and the dev loop fed by a sibling
directory hashed into the dev manifest at build time — trust stays a
build-time input ([#13's charter](https://github.com/davison/topos/issues/13)).
*Recorded in task goals rather than Decision comments — the migration's
first days predate that discipline settling.*

**The fleet installer converges; the demo leaves the release; the
verifier ships but is never placed.** The release stopped shipping the
demo plugin whose job ended when the real fleet arrived
([decision](https://github.com/davison/topos-plugins/issues/8#issuecomment-5480392192);
rejected: placing it, or an installer-side exclusion list). The release
ships `topos-provenance` built at a pinned kernel commit, and the
installer consults a staged copy only last — after an installed kernel's
copy and `PATH` — and never places it, so one payload can never seed
the tier later installs prefer
([decision](https://github.com/davison/topos-plugins/issues/8#issuecomment-5480392412)).
v0.3.0 was tagged after merge as the end-to-end proof, on tp#3's
precedent ([decision](https://github.com/davison/topos-plugins/issues/8#issuecomment-5480392652)).

**The kernel's uninstall keeps to its own artifacts.** After the split,
`$PREFIX/lib/topos/plugins` is topos-plugins' to fill and empty; a
kernel uninstall that deleted a fleet it did not ship would break the
independence R5 names. Rejected: the pre-split removal set (GSD
INST-05's letter), and a flag to opt into fleet removal — two owners for
one directory ([decision](https://github.com/davison/topos/issues/15#issuecomment-5481457308)).

**Incompatibility is judged at runtime, and an apply with only
per-instance failures commits.** The R6 gate reads the plugin's own
`Describe.contract_version`, not the provenance manifest's recorded
field — because every shipped manifest recorded `topos.v1` while the
fleet speaks `topos.v2`, a mislabel invisible while nothing consumed the
field; gating on it would have refused a compatible fleet on day one.
And a config apply whose only casualties are per-instance launch
failures now commits with them surfaced, instead of a whole-save
failure with no named culprit — version skew is an everyday state once
plugins update independently
([decisions](https://github.com/davison/topos/issues/17#issuecomment-5481760182)).
The mislabel itself was corrected at the source for the next release
([tp#10](https://github.com/davison/topos-plugins/issues/10)).

**Pull-by-URL has one discovery convention and no override.** The
release's own `checksums.txt` beside the binary names its evidence; no
forge API, no `--provenance` flag (an attacker-chosen provenance URL is
an attacker-chosen trust input), and no flag can name a tier
([decisions](https://github.com/davison/topos/issues/19#issuecomment-5483704507)).
A present-but-contradicting `checksums.txt` is a failed verification,
never a fall-through to the external tier.

**The guide links; it never restates. Its validator is a stranger.**
The developer guide is its own page beside the contract, which stays
the single semantic reference; the clean-room builder is a fresh Codex
session briefed with the guide file alone; and the third-party tier is
stated truthfully as external-via-consent-and-pin until a certification
path exists ([decisions](https://github.com/davison/topos/issues/21#issuecomment-5485387918)).

**"No per-plugin knowledge" means the operator's configuration
reference too.** QA's hairbrush found `CONTRIBUTING.md` still describing
the pre-split workspace; the remedy's reviewer then asked whether the
requirement's letter also covered the fleet's configuration reference
in `config.example.toml` and README. That was a requirement's-meaning
question, raised as the milestone's first `cc:needs-decision` gate and
resolved by the operator as Option A: each plugin's README in
topos-plugins carries its own fully-commented block (the capture that
had tracked importing the operator pages became a task,
[tp#14](https://github.com/davison/topos-plugins/issues/14)), and the
kernel's example config keeps only the kernel's keys, the mock, and
pointers ([gate](https://github.com/davison/topos/issues/24),
[#24](https://github.com/davison/topos/issues/24)). Rejected: the
narrower build/layout reading, which would have left two owners for one
operator document.

**And it has a floor.** Five QA verdicts on M1-R2 each found residue one
stratum below the last remedy — the operator's documents, then the
workflows' and Makefile's own headers, then the contract and a source
comment at the phrase level, then ownership claims only a reading for
meaning could catch, then a single wrong count. The structural facts
were never in dispute and QA re-confirmed them every pass: no functional
plugin source or binary in the kernel tree, the mocks the only plugin
modules, a mock-only build manifest. The milestone's second gate asked
what the requirement means after that many passes, and the operator
resolved it as A: remedy the one enumerated line, and the next verdict
is final whichever way it goes, further residue being M2 follow-up
rather than an M1 blocker ([gate](https://github.com/davison/topos/issues/6),
[#34](https://github.com/davison/topos/issues/34)). The sixth verdict was
satisfied. Rejected: scoping the requirement by decision (B) — the
residue was bounded, so the requirement as written was reachable — and
remedying without bound (C).

## Deviations, with their reasons

**A CI repair swept in a second one.** Fixing the e2e typecheck
surfaced a `go:embed` break caused by a `git commit -am` that had swept
`kernel/webui/build/.gitkeep` off main; restored in the same PR, with
the lesson recorded — commit by pathspec in this tree
([deviation](https://github.com/davison/topos/issues/9#issuecomment-5472675191)),
and then healed at the choke point ([#11](https://github.com/davison/topos/issues/11)).

**The verifier pin moved.** The plugins release had built its verifier
at the commit that introduced the CLI — which predates the commit that
embedded the signing key, so a verifier built there rejected every real
release. Caught only by rehearsing against the real v0.2.0; the pin
moved to current kernel main and the pin file now states the
key-set precondition ([deviation](https://github.com/davison/topos-plugins/issues/8#issuecomment-5480483510)).

**Updates converge rather than coexist.** The plan's leave-inert stance
would have left a retired plugin trusted by a stale manifest; after the
reviewer's finding the installer retires what an older release placed
and the new one drops — and, after a second finding, only when the
verifier proves the on-disk bytes are the ones our own older manifest
vouched for ([deviation](https://github.com/davison/topos-plugins/issues/8#issuecomment-5481319407);
[PR #9](https://github.com/davison/topos-plugins/pull/9) rounds 1–2).

**The no-evidence rule was widened.** The first wording of R8's
Decision would have aborted every unsigned third-party release — the
exact audience R9 serves. Amended under review: no provenance pair is
the no-evidence state whether `checksums.txt` is absent or clean
([amended decision](https://github.com/davison/topos/issues/19#issuecomment-5483868237)).

## Why the record looks like this: the review loop

The reviewer seat earned its keep in every task where it found
something, and the findings are the milestone's real design history:
the removal task took four rounds of stale-claim sweeps at path, file
and phrase level ([PR #14](https://github.com/davison/topos/pull/14));
the fleet installer's first version did not converge and its second
deleted foreign bytes by name ([PR #9](https://github.com/davison/topos-plugins/pull/9));
the pull command's byte-identical proof ignored directory existence, its
placement created directories behind a refusal, and its redirect check
read the origin instead of the previous hop
([PR #20](https://github.com/davison/topos/pull/20));
the guide had waved off one clean-room gap as "environment" and restated
what it should have linked ([PR #22](https://github.com/davison/topos/pull/22));
and at the close, the qa seat's hairbrush found the per-plugin knowledge
the removal task's four review rounds had never looked for in
`CONTRIBUTING.md` ([#24](https://github.com/davison/topos/issues/24),
remedied under the first gate) — and then kept finding it, one stratum
down each pass, for five verdicts: the workflows' own headers and the
Makefile's own comments, still describing a kernel release that carried
the fleet ([#28](https://github.com/davison/topos/issues/28)); the
contract and a Go source comment at the phrase level
([#30](https://github.com/davison/topos/issues/30)), whose remedy turned
the sweep into a committed instrument, `scripts/split-claims-sweep.py`,
hardened over six reviewer rounds of adversarial fixtures until its
exemption was clause-local and its own limit stated — two of those
rounds caught the implementer reporting a fix that had not actually
been pushed, and the record keeps them; the ownership claims only a
reading for meaning could catch, exactly where that instrument said it
could not see ([#32](https://github.com/davison/topos/issues/32)); and
one wrong count ([#34](https://github.com/davison/topos/issues/34)),
remedied under the second gate. The reviewer judges the diff; QA judges
the tree; and QA read for meaning where the instrument read for
vocabulary — each split is what caught the next thing.
None of those was a test failure; each was a premise the author held
that a de-correlated reader did not.

*Undocumented in any Decision comment, inferred from the PR threads:*
the convention that the Codex reviewer posts its verdict as a PR
comment — GitHub refuses self-approval under the operator's own auth,
so `task finish --operator-confirm` is the merge — was settled in the
first task's PR and followed thereafter.

## Left behind, deliberately

- Captures carried over from the GSD backlog: a certification path out
  of the untrusted tier ([#1](https://github.com/davison/topos/issues/1)),
  in-app install-from-URL ([#2](https://github.com/davison/topos/issues/2)),
  a plugin marketplace ([#3](https://github.com/davison/topos/issues/3)),
  kernel OAuth/secrets services ([#4](https://github.com/davison/topos/issues/4)),
  Signal schema tooling ([#5](https://github.com/davison/topos/issues/5)).
- On topos-plugins: gdrive's env-sensitive tests
  ([tp#5](https://github.com/davison/topos-plugins/issues/5)), browser
  coverage for the moved plugins ([tp#6](https://github.com/davison/topos-plugins/issues/6)),
  the per-plugin operator pages ([tp#7](https://github.com/davison/topos-plugins/issues/7)).
- The archived `topos-plugin-gdrive` repository stays archived, not
  deleted: it holds the clean-room scaffolding the fold-in left behind,
  the milestone's one real case study of a third-party build.
- The operator's live-instance UAT — installing v1.3.0 and the fleet on
  the real machine, the consent-and-pin flow, real syncing — is the one
  gate no script here stands in for; it is raised as the milestone's
  human checkpoint.

## The release

v1.3.0 is cut after this record merges, from the main it lands on:
the first kernel release to carry the provenance verifier, `topos plugin
pull`, and the R6 gate. The fleet it pairs with is topos-plugins v0.3.0,
whose verifier pin moves to the tag once it exists. The tag, the
operator's live UAT and the close verb are recorded on #6 as they
happen; the roadmap row moves to Done with the M2-opening edit that
adds the next row.
