---
phase: quick-260814-mkl
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - web/e2e/e2e-builtins.d.ts
autonomous: true
requirements: [QUICK-260814-mkl]
user_setup: []

estimate:
  tokens: 14000
  raw_tokens: 14000
  tasks: 1
  confidence: low

must_haves:
  truths:
    - "`npm --prefix web run check:e2e` exits 0 — the CI job that currently fails with `TS2305: Module '\"node:fs\"' has no exported member 'unlinkSync'` at `e2e/specs/12-filesystem-recursion.spec.ts(14,42)` is green."
    - "`web/e2e/specs/12-filesystem-recursion.spec.ts` compiles UNCHANGED — the fix lands entirely in the ambient shim, never by editing the spec's import list or by rewriting its `unlinkSync(nestedFilePath)` call into an `rmSync` call."
    - "The e2e tree still type-checks with NO `@types/node` package installed — `web/package.json` and `web/package-lock.json` are byte-identical after this change."
    - "The new declaration is narrowed to exactly the one-string-argument, discarded-return call shape the single call site uses (`web/e2e/specs/12-filesystem-recursion.spec.ts:126`) — no options bag, no overloads, no URL/file-descriptor variants."
  artifacts:
    - "web/e2e/e2e-builtins.d.ts — one `unlinkSync` declaration inside the existing `declare module 'node:fs'` block, carrying a provenance comment in the file's established style"
  key_links:
    - "`web/e2e/tsconfig.json` is the compilation root that pulls in `e2e-builtins.d.ts`; `check:e2e` runs `tsc --noEmit -p e2e/tsconfig.json` against exactly that root. This shim is scoped to the e2e tree only — the SvelteKit app has its own separate `web/src/lib/node-builtins.d.ts` and must not be touched."
    - "The `declare module 'node:fs'` block (currently `web/e2e/e2e-builtins.d.ts:57-82`) is a MODULE AUGMENTATION with no real `node:fs` types behind it — it is the complete definition of that module for this compilation unit. Any member absent from the block is a hard TS2305 for every importer, which is exactly the failure being fixed."
  prohibitions:
    - "Do NOT install `@types/node` (or any other package) to resolve this. The whole reason this shim file exists is that the package-legitimacy gate in 07.1-01-SUMMARY.md scoped approval to exactly @playwright/test, playwright and smol-toml — adding a fourth package here bypasses that gate. The file's own header comment states this."
    - "Do NOT edit `web/e2e/specs/12-filesystem-recursion.spec.ts`. The spec is correct; the shim is what is incomplete. Swapping its `unlinkSync` for `rmSync` to dodge the declaration would silently change what the spec proves about single-leaf-file removal."
    - "Do NOT bulk-add other `node:fs` members 'while in here' (e.g. statSync, renameSync, copyFileSync, appendFileSync). The file's stated discipline is 'exactly what's imported, nothing more'; speculative members are unfalsifiable dead declarations."
    - "Do NOT widen the signature beyond the single call shape — no `PathLike`, no `URL` overload, no callback/promise variant."
    - "Do NOT reformat, re-sort, or re-wrap the surrounding declarations or their comments. The diff is an addition, not a tidy-up."
---

<objective>
Add the missing `unlinkSync` declaration to the `declare module 'node:fs'` block in
`web/e2e/e2e-builtins.d.ts` so the e2e tree type-checks again.

Purpose: CI's `check:e2e` job is red. Phase 12's `12-filesystem-recursion.spec.ts`
(from 12-03-PLAN.md Task 2) imports `unlinkSync` to delete a nested file and prove
the item leaves the stream on the next sync, but nobody extended the ambient shim
that stands in for `@types/node` in this tree. The spec is right; the shim is
incomplete.

Output: one declaration line plus a provenance comment in `web/e2e/e2e-builtins.d.ts`.
</objective>

<execution_context>
@$HOME/.claude/gsd-core/workflows/execute-plan.md
@$HOME/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/STATE.md

@web/e2e/e2e-builtins.d.ts
</context>

<tasks>

<task type="auto">
  <name>Task 1: Declare unlinkSync in the e2e node:fs ambient shim</name>

  <files>web/e2e/e2e-builtins.d.ts</files>

  <precondition>
    `npm --prefix web run check:e2e` currently FAILS with exactly one error:
    `e2e/specs/12-filesystem-recursion.spec.ts(14,42): error TS2305: Module '"node:fs"'
    has no exported member 'unlinkSync'.` Run it first and confirm this is the only
    error reported. If other errors appear, or if it already passes, halt and report —
    the diagnosis this plan is built on no longer holds.
  </precondition>

  <action>
    Add a single member declaration to the EXISTING `declare module 'node:fs'` block in
    `web/e2e/e2e-builtins.d.ts` (the block opens at line 57 and closes at line 82).

    Append the new member at the END of that block, after the existing `readdirSync`
    line and before the block's closing brace. Appending is what the block's current
    ordering already reflects: the base set first, then each later phase's additions
    tacked on as they were needed.

    Declare it with exactly this signature — one required string parameter, void return:

      export function unlinkSync(path: string): void;

    That is the precise shape of the only call site, `unlinkSync(nestedFilePath)` at
    `web/e2e/specs/12-filesystem-recursion.spec.ts:126`, whose return value is discarded.
    Do not add a second overload, an options parameter, or a `PathLike`/`URL` union — the
    surrounding declarations are all narrowed the same way, and the file's header states
    the rule explicitly.

    Precede the declaration with a short tab-indented `//` comment matching the style the
    block already uses for phase-scoped additions — compare the existing `chmodSync` entry,
    which names its originating plan (11-06-PLAN.md Task 3) and then says in one or two
    sentences why that plan needed it, and the `basename` entry in the `node:path` block
    below, which does the same for 12-01-PLAN.md Task 2. Follow that pattern: name this
    one's origin as 12-03-PLAN.md Task 2's filesystem-recursion spec, and state that it
    removes the single nested file so the next sync can prove the item drops out of the
    stream. Worth recording in that comment: the block's existing `rmSync` is a
    directory-shaped recursive removal and is not the right tool for deleting one leaf
    file, which is why a distinct member is needed rather than reusing what is already
    declared.

    Wrap comment prose at the file's prevailing width (roughly 72 columns inside the
    indent) and indent with a tab, as every other line in the block does.

    Touch nothing else in the file. No reordering, no reflowing of neighbouring comments,
    no additional members.
  </action>

  <verify>
    <automated>cd /home/darren/projects/davison/topos && npm --prefix web run check:e2e</automated>
    <automated>cd /home/darren/projects/davison/topos && grep -q 'export function unlinkSync(path: string): void;' web/e2e/e2e-builtins.d.ts && echo DECL_OK</automated>
    <automated>cd /home/darren/projects/davison/topos && git diff --stat -- web/package.json web/package-lock.json web/e2e/specs/12-filesystem-recursion.spec.ts | wc -l | grep -qx '0' && echo NO_COLLATERAL</automated>
    <automated>cd /home/darren/projects/davison/topos && test -z "$(git diff --name-only -- web/ | grep -vx 'web/e2e/e2e-builtins.d.ts')" && echo SCOPED_DIFF_OK</automated>
  </verify>

  <done>
    `npm --prefix web run check:e2e` exits 0 with no diagnostics. The `unlinkSync`
    declaration is present in the `node:fs` block with a provenance comment.
    Within `web/`, `web/e2e/e2e-builtins.d.ts` is the only modified path.
    `web/package.json`, `web/package-lock.json` and the phase-12 spec are untouched.

    Note: the working tree already carries unrelated pre-existing changes OUTSIDE
    `web/` (`go.work.sum`, a deleted `kernel/webui/build/.gitkeep`, and untracked
    `plugins/filesystem/filesystem`). Leave them alone and do not stage them — the
    diff checks above are deliberately scoped to `web/` for exactly this reason.
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| (none crossed) | This change adds a compile-time-only ambient type declaration to the e2e test tree. TypeScript erases it entirely; no runtime code, no shipped artifact, and no kernel/plugin/browser surface is affected. The file is scoped to `web/e2e/tsconfig.json` and is not part of the SvelteKit app build or the `go:embed`ed SPA. |

## STRIDE Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation Plan |
|-----------|----------|-----------|----------|-------------|-----------------|
| T-260814-mkl-01 | Tampering | supply chain (npm) | high | mitigate | The obvious wrong fix is `npm i -D @types/node`, which would silently bypass the package-legitimacy gate that scoped this project's e2e dependencies to exactly three approved packages (07.1-01-SUMMARY.md). Installing a package is prohibited by this plan, and the verify block asserts `web/package.json` and `web/package-lock.json` are unmodified. No package-manager install task exists in this plan, so no legitimacy audit table is required. |
| T-260814-mkl-02 | Tampering | e2e assertion integrity | medium | mitigate | An over-broad or wrongly-typed shim entry can make a spec compile while it no longer proves what it claims (e.g. rewriting the deletion call to `rmSync` to dodge the declaration). Mitigated by narrowing the signature to the single observed call shape and by asserting the spec file itself is unmodified in the diff. |
</threat_model>

<verification>
Run from the repo root:

1. `npm --prefix web run check:e2e` — exits 0, no diagnostics. This is the exact CI
   job that is currently red and is the primary gate.
2. `git diff --name-only -- web/` — returns exactly `web/e2e/e2e-builtins.d.ts`.
   (Scoped to `web/` on purpose: the tree carries unrelated pre-existing changes
   outside it, which this task must not touch or stage.)

No Playwright run is required: this change is type-declaration-only and cannot alter
runtime behaviour of any spec.
</verification>

<success_criteria>
- `npm --prefix web run check:e2e` exits 0.
- `web/e2e/e2e-builtins.d.ts` is the only modified file under `web/`.
- `unlinkSync` is declared once, with signature `(path: string): void`, inside the
  `declare module 'node:fs'` block, preceded by a provenance comment naming
  12-03-PLAN.md Task 2.
- No package was installed; no spec was edited.
</success_criteria>

<output>
Create `.planning/quick/260814-mkl-fix-ci-check-e2e-failure-declare-unlinks/260814-mkl-SUMMARY.md` when done
</output>
