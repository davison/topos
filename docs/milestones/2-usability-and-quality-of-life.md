# M2: Usability and quality of life (v1.3.1)

Milestone issue [#40](https://github.com/davison/topos/issues/40); the
second milestone run under CodeCrew, opened 2026-09-01 at M1's close.
This record is the milestone's last task
([#68](https://github.com/davison/topos/issues/68)) — and unlike
[M1's record](1-plugin-repo-split-and-the-third-party-path.md), which
merged ahead of its tag, this one lands after the release: the tag pair
(kernel v1.3.1, fleet v0.3.1), the operator's live UAT and the QA
verdicts are already recorded on #40, in the gate sequence the release
section below retells. Every claim below links to the comment, PR or
issue that recorded it; where a choice clearly happened without a
record, this document says so rather than reconstructing a rationale.

## Goal and outcome

**Chartered:** what actual daily use of v1.3.0 surfaced, and what the
close of M1 cost ([#40](https://github.com/davison/topos/issues/40)): a
CI gate that spent four minutes on a prose change, a search that
reached titles and previews while the detail pane implied it reached
bodies, filters that could only say one thing to every source,
third-party plugins external for everyone forever, a Signal schema
ceiling that moved only by hand, and the housekeeping M1's close
surfaced. Every requirement a usability or quality-of-life improvement
for the operator of a single instance; the larger product surfaces —
in-app install ([#2](https://github.com/davison/topos/issues/2)), a
marketplace ([#3](https://github.com/davison/topos/issues/3)), kernel
OAuth/secrets ([#4](https://github.com/davison/topos/issues/4)) — wait
for a later milestone.

**Shipped:**

- CI is two jobs: a cheap `changes` job always reports — so the finish
  verb always has a fact to count — and the four-minute `test` job runs
  only when the diff touches code, with the classifier committed in the
  workflow, never read from a commit message
  ([#41](https://github.com/davison/topos/issues/41)). The proof, taken
  on the first docs-only PR after merge: `changes` pass, `test`
  skipping, the finish verb's CI gate satisfied, in seconds
  ([recorded on #41](https://github.com/davison/topos/issues/41#issuecomment-5495988134)).
- Search reaches bodies without a second copy: an additive, optional
  `Search` RPC in the plugin contract, designed and gated first
  ([#50](https://github.com/davison/topos/issues/50)), then the kernel's
  fail-closed fan-out with per-source budgets
  ([#53](https://github.com/davison/topos/issues/53)), all seven
  plugins searching their own stores within membership
  ([tp#25](https://github.com/davison/topos-plugins/issues/25)), and a
  result set that says where each hit matched (`matched_in`/`origin`/
  `indexed`), which sources answered, arriving progressively — with the
  detail pane's body highlight finally labelled as find-in-page
  ([#54](https://github.com/davison/topos/issues/54)). Nothing a search
  returns is written to the index.
- A webspace filter speaks per source:
  `[webspaces.<w>.filter_by_source]` beside the global stack, the source
  chip's *Filter this source…* popover, and `instance:term` in the
  search box as sugar; each instance's terms ride to its own source as
  `required_terms` ([#55](https://github.com/davison/topos/issues/55)).
- Trust gained a second word: the operator's. A developer's signing key,
  consented to once via `[[plugins.trusted_keys]]`, earns that
  developer's signed plugins an `operator_trusted` tier across releases
  instead of a per-binary pin — designed and gated first
  ([#49](https://github.com/davison/topos/issues/49)), then the kernel
  tier, offer and `topos plugin pull` behaviour
  ([#56](https://github.com/davison/topos/issues/56)) and the badge,
  consents and specs in the app
  ([#57](https://github.com/davison/topos/issues/57)).
- The Signal plugin's schema ceiling moves by one command:
  `make signal-schema-accept` runs the live read-set test against the
  operator's real database (read-only, as always), names the
  intervening upstream migrations, and rewrites `schemaguard.go` with
  its provenance bullet — printing the diff, never committing
  ([tp#23](https://github.com/davison/topos-plugins/issues/23)).
- The housekeeping: [`docs/releasing.md`](../releasing.md) describes the
  CodeCrew lifecycle, not GSD's
  ([#43](https://github.com/davison/topos/issues/43));
  `scripts/sync-milestones.sh` is gone
  ([#44](https://github.com/davison/topos/issues/44)); `install-signal`
  refuses to run as root and the docs say which steps need sudo
  ([tp#21](https://github.com/davison/topos-plugins/issues/21),
  [#47](https://github.com/davison/topos/issues/47)); the next tag is
  derived from the commit log, in both repositories
  ([#64](https://github.com/davison/topos/issues/64),
  [tp#27](https://github.com/davison/topos-plugins/issues/27)); and the
  version the docs name is the version that shipped
  ([#66](https://github.com/davison/topos/issues/66)).

Seventeen tasks, seventeen PRs before this record — thirteen on the
kernel repository and four on
[`topos-plugins`](https://github.com/davison/topos-plugins) — all
rebase-merged through `task finish` after a model review under the
reviewer contract in a headless Codex session, the solo-tier convention
M1 established and M2 kept (every PR's thread records it; e.g.
[PR #60](https://github.com/davison/topos/pull/60),
[tp PR #26](https://github.com/davison/topos-plugins/pull/26)).

## Requirement outcomes

Drawn from the qa seat's verdicts on #40
([the six-verdict comment](https://github.com/davison/topos/issues/40#issuecomment-5500629657);
R6's [superseding verdict](https://github.com/davison/topos/issues/40#issuecomment-5500694636)).
QA's sandbox could not run the merged-main Go/e2e floor (the mandated
`$HOME/.cache/go-build` is read-only there), and each verdict says so
and names what was independently exercised instead — the published
v1.3.1/v0.3.1 releases driven over the HTTP API, the workflow's own
check runs, the schema-accept dry-run — and the assumption that floor
leaves standing.

| Requirement | Delivered by | Outcome |
|---|---|---|
| M2-R1 — docs-only PRs satisfy the finish gate in seconds | #41; proof on PR #45 | satisfied |
| M2-R2 — search finds a term in bodies across every source, without a second copy | #50, #53, tp#25, #54 | satisfied |
| M2-R3 — per-source filter terms beside one general search | #50, #55 | satisfied |
| M2-R4 — operator-trusted developer keys, with consent and a badge | #49, #56, #57 | satisfied |
| M2-R5 — Signal's schema ceiling moves by verify-and-accept | tp#23 | satisfied |
| M2-R6 — post-M1 housekeeping | #43, #44, #47, tp#21, tp#27, #64, #66 | satisfied — on the second verdict; the first found maintained docs dating a shipped feature to a release that never existed ([finding on #64](https://github.com/davison/topos/issues/64#issuecomment-5500619803)), remedied by #66 |

## The decisions that shaped it

Both feature families put the design in its own docs-only task before
any code, each behind a `cc:needs-decision` gate resolved by the
operator "as recommended"
([#50's gate](https://github.com/davison/topos/issues/50#issuecomment-5496387816)
and [resolution](https://github.com/davison/topos/issues/50#issuecomment-5497003319);
[#49's gate](https://github.com/davison/topos/issues/49#issuecomment-5496353224)
and [resolution](https://github.com/davison/topos/issues/49#issuecomment-5497002771)).
The pattern itself was never recorded as a Decision — it is simply how
the milestone was cut into tasks — but it is why the review loop below
could correct both designs on paper, before the mistakes were code.
Each entry names the choice, its trade-off, and what was rejected, and
links the comment that recorded it.

**An additive, optional RPC — and membership is the source's, not the
kernel's.** `Search` is declared in `Describe`; a `topos.v2` plugin
without it is "no body search from this source," never an
incompatibility, and there is no `topos.v3`
([design note](https://github.com/davison/topos/issues/50#issuecomment-5496387484)).
The design's first draft had the kernel correlating returned hits with
"the same rules sync uses" — which do not exist as a local predicate:
sync asks `Match`, and an `Item` does not carry the native values that
decision needs. So the resolved `match_fields` ride in `SearchRequest`
and the source ANDs search with membership, returning only hits `Match`
would also return; the SDK exposes `Search` as an optional interface
found by type assertion, never a new method on `sdk.SourcePlugin`
([decision](https://github.com/davison/topos/issues/50#issuecomment-5497114215)).
Rejected: the kernel calling `Match` per search and intersecting ids —
a full membership scan of every source per search, and a second round
trip on the slow path.

**Fail closed, and the trust boundary stated as it is.** The kernel
fans out only to instances that participate with resolved membership
input, and `Search` refuses an empty or absent map — rejected: "empty
map means the whole source," which could disclose an operator's whole
mail or chat archive into an empty webspace. The boundary is recorded
without flattery: search trusts the source to AND with the supplied
fields exactly as sync trusts its `Match` result set; the kernel
guarantees only its own side. And the saved filter stack rides to the
source as `required_terms`, so a body hit cannot bypass a saved filter —
rejected: post-filtering source hits on their snippets, which drops
true body matches
([decisions](https://github.com/davison/topos/issues/50#issuecomment-5497168771)).

**Progressive is two requests, not a stream.** `?scope=index` answers
from FTS in milliseconds; `?scope=all` (the default) brings the fan-out
under a 5-second per-source budget and the `sources` status map —
rejected: server-sent events, a second transport for one screen. The
route discovers the fan-out by type-asserting its existing `Fetcher`
for a `Searcher`, an FTS row's `matched_in` is `title` only when every
term is in the title, and `indexed` is decided by `store.GetItem`;
nothing a search returns is written
([decisions on #53](https://github.com/davison/topos/issues/53#issuecomment-5498055560)).

**The key travels with the signature.** `.provenance.sig` gains
`public_key` (written by `topos-provenance sign`; the schema stays
`topos.provenance.sig.v1` — an added field older kernels ignore), so an
offer is only ever made for a key that demonstrably signed the
manifest; the fingerprint is the SHA-256 of the raw key bytes, trusting
stores the bytes, and a trusted key id arriving with different bytes is
treated as unknown and the offer says so. Rejected: a separate
`.pubkey` asset (a discovery rule, and a second fetch for pull's single
URL), and the key in the manifest (the manifest is the signed thing).
Also corrected on the record: an unknown-key manifest was already
*no evidence* at launch — it was `topos plugin pull` that aborted — so
the offer changes pull's behaviour and adds the consent; the launch
gate's tiers are preserved exactly
([decision](https://github.com/davison/topos/issues/49#issuecomment-5497132095)).

**A second word for trust, named for whose word it is.** The operator's
keys live in `[[plugins.trusted_keys]]` beside the pins — the config
file was already the operator's trust surface — each key remembering
which word it is; a distinct `operator_trusted` tier and badge say on
whose word a plugin runs; and D-12 is revised on the record: link-time-
only stays true of the *kernel author's* keys, while the operator's are
runtime configuration exactly as the pins are
([design note](https://github.com/davison/topos/issues/49#issuecomment-5496352913),
[trust doc](../plugin-trust.md)). Rejected at the checkpoint: folding
operator keys into `trusted` (dishonest about whose word), keeping the
unknown-key refusal (a self-signed third-party plugin stays
uninstallable by pull until trusted), and a separate keyring file (two
trust surfaces).

**The seven searches share one small kit.** `searchkit/` (term
splitting, required-term AND, the snippet window, limit/truncation, the
empty-membership refusal) lives in the repository's root module, and
each plugin module reaches it with a `replace` to `../..` — versioned
with the plugins that use it. Rejected: publishing it as a second
module; a plugin that ever leaves the repo copies the file
([decision](https://github.com/davison/topos-plugins/issues/25#issuecomment-5498522145)).

**The next tag is read off the commit log — and this release's tags
follow reality, not the pre-named numbers.** Operator-requested at the
release gate: the derivation now lives in
[`docs/releasing.md`](../releasing.md) ("Versioning" — breaking-marked
commits force a minor bump, additive `feat:` or any `fix:` a patch,
docs-class commits nothing, a major always a human decision), enforced
by the commit-type obligations added to
[`roles/implementer.local.md`](../../roles/implementer.local.md) and
[`roles/reviewer.local.md`](../../roles/reviewer.local.md)
([#64](https://github.com/davison/topos/issues/64),
[PR #65](https://github.com/davison/topos/pull/65)), with the fleet's
README pointing at the same rule
([tp#27](https://github.com/davison/topos-plugins/issues/27)). Applied
to the actual logs, the rule yielded v1.3.1 and v0.3.1 — not the
v1.4.0/v0.4.0 the roadmap and milestone had pre-named
([gate addendum](https://github.com/davison/topos/issues/40#issuecomment-5499780905)) —
and a commit-by-commit reality audit of every `feat:`/`fix:` since the
last tags confirmed nothing breaking on any published surface, naming
three additive-but-noticeable edges for honesty
([the audit](https://github.com/davison/topos/issues/40#issuecomment-5500439303)).
Rejected: grandfathering the pre-named pair for this one release
(option A); the operator resolved the fork as apply-immediately
([resolution](https://github.com/davison/topos/issues/40#issuecomment-5500446903)).

## Deviations, with their reasons

**The docs-only proof could not precede its own merge.** `ci.yml`
triggers on pull requests to `main` only, and any PR to main carrying
the two-job change diffs as code — the workflow file itself — while
widening the triggers for a proof would change the thing under test.
The proof was deferred to the first docs-only PR after merge and taken
there, closing the deviation out
([deviation](https://github.com/davison/topos/issues/41#issuecomment-5495517182),
[proof](https://github.com/davison/topos/issues/41#issuecomment-5495988134)).

**The picker could not add the shape the contract prescribes.** Driving
the add-source interstitial for the trust specs exposed a gap outside
the task's plan: an unknown plugin type got the generic connect form —
display name and sync interval only — so a third-party plugin whose
fatal-guard requires `path` (the external-demo fixture, and the
contract's own documented shape) could not be added from the picker at
all. The generic form gained the three kernel-known connection keys as
Advanced options. Rejected: weakening the fixture's fatal-guard, a
documented pattern other plugins mirror
([deviation](https://github.com/davison/topos/issues/57#issuecomment-5498229923)).

**Signal searches through the proven read path, not `messages_fts`.**
The plan named Signal Desktop's own FTS table; the implementation reads
bodies through the same read-only `readMessages` path `Match` and
`Fetch` use, because the read-only-by-construction guarantee is proven
for that path and pinned by the byte-identical test — touching the FTS
virtual table would widen the surface the guarantee covers for no
reachable gain at a webspace's scale. WhatsApp does the same over its
own store
([deviation](https://github.com/davison/topos-plugins/issues/25#issuecomment-5498522145)).

**No snippet from the server-indexed services — amended under review.**
Proton and Google Drive return hits without snippets: the server-side
index is what's searched, and fetching bodies to build snippets would
turn one `SEARCH` into N fetches. The deviation as first recorded also
covered paperless-ngx — but paperless returns document content with its
results, so its hits do carry a snippet, and the record was amended to
say so when review caught the docs contradicting the code
([deviation](https://github.com/davison/topos-plugins/issues/25#issuecomment-5498522145),
[amendment](https://github.com/davison/topos-plugins/issues/25#issuecomment-5498650572)).

**The reviewer's own contract file went missing mid-milestone.** The
round-2 reviewer on PR #42 reported that the composed reviewer contract
it was handed was zero bytes; the implementer confirmed it — truncated
in a full-`/tmp` incident — and recorded the regeneration on the PR
([exchange](https://github.com/davison/topos/pull/42#issuecomment-5495836251)).
That comment promises a Deviation on the two topos-plugins tasks
reviewed meanwhile; no such comment exists on
[tp#21](https://github.com/davison/topos-plugins/issues/21) or
[tp#23](https://github.com/davison/topos-plugins/issues/23) — an
unrecorded gap this record names rather than fills.

## Why the record looks like this: the review loop

Every implementation PR in this milestone took at least one
CHANGES REQUESTED round, and — twice — the review rewrote the design
before any code existed. On the search design
([PR #51](https://github.com/davison/topos/pull/51), three rounds), the
reviewer proved correlate-before-merge unimplementable as drafted (no
kernel predicate can repeat `Match`'s decision over an `Item`), which
produced the membership-in-`SearchRequest` decision; round two then
killed a fail-open "empty map means the whole source" rule, an
overstated kernel membership guarantee, and a protobuf type that did
not exist. On the trust design
([PR #52](https://github.com/davison/topos/pull/52), three rounds), the
"kernel refuses unknown keys outright" premise the captures had carried
was shown false against the code, and the reviewer's demand for a
recorded key transport became the key-travels-with-the-signature
decision.

In the implementations: the CI classifier's anchored regex group made
`docs/config.toml` code
([PR #42](https://github.com/davison/topos/pull/42), which also
weathered GitHub itself silently stalling the PR's check suites until a
close/reopen re-fired the event); the fan-out had a data race — the
capability-declining branch appended to the shared slice outside the
mutex — fixed with preallocated per-instance slots and a `-race` test
([PR #60](https://github.com/davison/topos/pull/60)); the serve path
never installed the operator's keys, a rejected apply left proposed
keys installed, and an operator key id could shadow the kernel author's
own ([PR #58](https://github.com/davison/topos/pull/58)); the app half
of trust had its interstitial paths untested until the review demanded
them, twice ([PR #59](https://github.com/davison/topos/pull/59));
clearing the search box left in-flight requests able to repopulate the
cleared results ([PR #61](https://github.com/davison/topos/pull/61));
the per-source chip's remove path lacked the stale-navigation guard its
siblings had ([PR #62](https://github.com/davison/topos/pull/62)); and
on the versioning doc the reviewer ran the documented derivation
command verbatim, twice, and it failed both times — first as
shell-redirection syntax, then in a clone with no tags — alongside an
unscoped-`fix!:` omission and a mixed-commit allowance that contradicted
the base contract's atomic commits
([PR #65](https://github.com/davison/topos/pull/65)).

On the fleet, the milestone's sharpest catch: a `match_fields` map
holding only a *foreign* plugin's keys turned the membership filter off
entirely and made the search global — the exact trust boundary the
design had just settled — remedied by making `RequireMembership` demand
a value under the plugin's own declared vocabulary, with a
foreign-only-key regression test in all seven plugins
([tp PR #26](https://github.com/davison/topos-plugins/pull/26)). The
schema-accept task's smoke never drove the script's `main`, so the
required refusal proofs did not exist until review made them acceptance
coverage rather than optional hardening
([tp PR #24](https://github.com/davison/topos-plugins/pull/24)).

Two findings came from the specs rather than the seats: the e2e suite
for the result set exposed that the Supervisor never forwarded
`SearchSources` — the route's type assertion failed silently and
`scope=all` degraded to index-only everywhere outside the unit tests'
fakes — fixed with a delegation and a compile-time interface pin
([PR #61](https://github.com/davison/topos/pull/61)); and driving the
interstitial exposed the generic-form gap the Deviations section
records. QA judged the tree and the releases: R1 through R5 passed on
the first verdict, and R6's hairbrush read the release record against
reality — maintained docs dating operator-trusted keys to a v1.4.0 that
never existed, while QA had itself proven the feature against the
published v1.3.1 binary
([finding on #64](https://github.com/davison/topos/issues/64#issuecomment-5500619803));
[#66](https://github.com/davison/topos/issues/66) reconciled the four
claims and the
[superseding verdict](https://github.com/davison/topos/issues/40#issuecomment-5500694636)
closed the requirement. As in M1, almost none of this was a test
failure; each finding was a premise the author held that a
de-correlated reader — or a browser driving the real thing — did not.

## Left behind, deliberately

- The larger plugin-ecosystem surfaces the charter deferred: in-app
  install-from-URL ([#2](https://github.com/davison/topos/issues/2)),
  a marketplace ([#3](https://github.com/davison/topos/issues/3)),
  kernel OAuth/secrets services
  ([#4](https://github.com/davison/topos/issues/4)). M2-R4 drew on the
  certification capture
  ([#1](https://github.com/davison/topos/issues/1)) but delivered
  trust-by-the-operator's-consent, not certification; #1 stays open.
- The per-source opt-in local body index — option C in
  [#39](https://github.com/davison/topos/issues/39) — stays deferred: a
  separate decision if live search proves too slow for a source whose
  content is already local
  ([design note](https://github.com/davison/topos/issues/50#issuecomment-5496387484)).
- The verifier pin: topos-plugins' `TOPOS_PROVENANCE_REF` stays at
  v1.3.0 for this release (the manifest schema is unchanged and the
  v1.3.0 verifier still accepts the fleet key); a small parity-bump
  task follows later if wanted
  ([gate resolution](https://github.com/davison/topos/issues/40#issuecomment-5500446903)).
- A stream-row rendering bug the operator reported mid-milestone —
  label pills forced onto a third line are clipped by the row's fixed
  height — captured as backlog, not fixed here
  ([#63](https://github.com/davison/topos/issues/63)).

## The release

M2's release preceded this record, and its story is the milestone's
final gate, told in five comments on #40. The gate was raised with all
six requirements done, recommending tags v1.4.0 and v0.4.0 — the
numbers the roadmap had pre-named
([gate](https://github.com/davison/topos/issues/40#issuecomment-5499193905)) —
and the operator first resolved exactly that:
["cut 1.4.0"](https://github.com/davison/topos/issues/40#issuecomment-5499657651).
Then the versioning rule the operator had requested at the same gate
([#64](https://github.com/davison/topos/issues/64)) landed mid-gate,
and applying it to the actual logs yielded v1.3.1 and v0.3.1 — no
breaking-marked commit since either tag
([addendum](https://github.com/davison/topos/issues/40#issuecomment-5499780905));
the reality audit confirmed the log told the truth
([addendum 2](https://github.com/davison/topos/issues/40#issuecomment-5500439303));
and the final resolution named the tags the releases now carry: kernel
**[v1.3.1](https://github.com/davison/topos/releases/tag/v1.3.1)** at
`2a85ef2`, fleet
**[v0.3.1](https://github.com/davison/topos-plugins/releases/tag/v0.3.1)**
at `96866d0`, both published 2026-09-01, with the operator's UAT a pass
— "search works well from the testing I did"
([resolution](https://github.com/davison/topos/issues/40#issuecomment-5500446903)).

Two thin spots in that sequence, named rather than smoothed: the UAT is
recorded as that one line, not itemized against the four-surface walk
the gate comment proposed; and the milestone issue's title still
pre-names v1.4.0 — everywhere else, the release column and every
maintained doc were reconciled to v1.3.1 by
[#66](https://github.com/davison/topos/issues/66) after QA refused R6
over exactly that discrepancy.

This record's PR flips the roadmap's M2 row to Done; `milestone
evidence 2` and `milestone close 2` follow on
[#40](https://github.com/davison/topos/issues/40) once it merges.
