// Commit-message lint, run by the "Lint commit messages" job in
// .github/workflows/ci.yml on every pull request (davison/topos#93, M4-R3).
//
// Release tags are derived from the commit log alone (docs/releasing.md,
// "Versioning"), so a commit's type is release metadata, not style: this
// config is the gate that keeps every type one the derivation can read.
//
// Departures from @commitlint/config-conventional, each deliberate:
//
// - body-max-line-length and footer-max-line-length are off. The house
//   style writes commit bodies as unwrapped paragraphs, one per idea, and
//   both `git log` and GitHub render them fine. Measured with the pinned
//   CLI (21.2.2) over the 60 commits then on main (main~60..main at
//   639896b): 22 failed the unmodified preset — 16 on
//   body-max-line-length, 12 on header-max-length, 6 on both; the
//   footer rule fired on none, and is off for the same reason as the
//   body rule (a trailing paragraph parses as a footer). Nothing the
//   derivation reads lives in a body line's length.
//
// Everything else is inherited as-is — the type list (build, chore, ci,
// docs, feat, fix, perf, refactor, revert, style, test), the empty-type
// and empty-subject refusals, the lower-case subject, the 100-character
// header limit. With this config the same 60 commits fail 12 times, all
// on header-max-length (101–135 characters): the lint applies only to a
// pull request's own commits, so history is untouched and a long header
// is trimmed before merge.
export default {
  extends: ['@commitlint/config-conventional'],
  rules: {
    'body-max-line-length': [0],
    'footer-max-line-length': [0],
  },
};
