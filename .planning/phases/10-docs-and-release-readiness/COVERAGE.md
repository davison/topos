# Phase 10 — API Coverage Matrix

**Decided:** 2026-08-12 (planner)
**External API:** GitHub REST API v3, consumed exclusively through the `gh` CLI
(`gh api`, `gh release`, `gh run`, `gh workflow`) — no HTTP client, no SDK, no
new dependency is added by this phase.

## Detector result

`node gsd-core/bin/lib/api-coverage.cjs --json` over the Phase 10 ROADMAP scope
returned `{"detected": false}`. This matrix is produced anyway because the
phase's release-engineering half *does* drive GitHub's REST milestone and
release surfaces, and the seal-time gate re-scans PLAN.md bodies (which cite
`gh api` endpoints verbatim). Deciding the rows up front is cheaper than
discovering an undecided matrix at seal time.

## Capability surface

### Milestones (`/repos/{owner}/{repo}/milestones`)

| Capability | Decision | Reason |
|---|---|---|
| List milestones — `GET /repos/{owner}/{repo}/milestones?state=all` | INTEGRATE | `scripts/sync-milestones.sh` must look a milestone up by title before deciding create-vs-update — this is the idempotency mechanism (RESEARCH.md Pitfall 4: milestone `v1.0` #1 already exists). |
| Create milestone — `POST /repos/{owner}/{repo}/milestones` | INTEGRATE | Success criterion 5 — a `.planning/` milestone that has no GitHub mirror gets created. |
| Update milestone state — `PATCH /repos/{owner}/{repo}/milestones/{number}` | INTEGRATE | Success criterion 5 — closing a milestone at `/gsd-complete-milestone` time, and back-filling the description on the already-existing empty `v1.0`. |
| Delete milestone — `DELETE /repos/{owner}/{repo}/milestones/{number}` | OPT-OUT | `.planning/` never deletes a milestone, only closes it; a delete would silently orphan every issue assigned to it. Deliberately outside the script's vocabulary. |
| Milestone due dates — `due_on` field on POST/PATCH | OPT-OUT | `.planning/` tracks no milestone due dates, so there is no source value to mirror. Mirroring an invented date would make GitHub the source of truth for a field `.planning/` does not own. |
| Assign issues to a milestone — `PATCH /repos/{owner}/{repo}/issues/{n}` | OPT-OUT | Issue triage is a human activity (`/gsd-inbox`), not milestone-boundary sync. Out of this phase's success criteria. |

### Releases (`/repos/{owner}/{repo}/releases`)

| Capability | Decision | Reason |
|---|---|---|
| Create a release with assets — `gh release create <tag> <files>` | INTEGRATE | Success criterion 6 — tag-triggered release attaches kernel + plugin binaries. |
| Auto-generate release notes — `--generate-notes` on create | INTEGRATE | Free changelog from commit/PR history; no hand-maintained CHANGELOG to drift. |
| Mark a release as prerelease — `--prerelease` on create | INTEGRATE | Nightly builds publish to a moving `nightly` tag and must not present as a stable release. |
| Delete a release + its tag — `gh release delete --cleanup-tag` | INTEGRATE | The nightly workflow replaces the previous nightly; the release tracer cleans up its own throwaway verification tag. |
| Upload an asset to an existing release — `gh release upload` | OPT-OUT | Both workflows create the release and its assets in one call; there is no second, later-arriving artifact to append. |
| Release discussions / reactions — `discussion_category_name` | OPT-OUT | Single-user local-first tool; no release discussion forum exists or is wanted. |
| Draft releases — `--draft` | OPT-OUT | Nothing in this project reviews a release before publishing it — a draft would just be an extra manual step between the tag push and a usable download. |

### Repository security settings

| Capability | Decision | Reason |
|---|---|---|
| Enable private vulnerability reporting — `PUT .../private-vulnerability-reporting` | INTEGRATE | Success criterion 2 — `SECURITY.md` points at the "Report a vulnerability" button, which does not exist until this is enabled. |
| Read PVR state — `GET /repos/{owner}/{repo}/private-vulnerability-reporting` | INTEGRATE | Used as the verification assertion that the enable actually took effect. |
| Disable PVR — `DELETE` on the same path | OPT-OUT | Nothing in this project ever wants the private disclosure channel turned off. |
| Security advisories CRUD — `/repos/{owner}/{repo}/security-advisories` | OPT-OUT | Drafting an advisory is a response to a real report, not repo setup. Out of scope until there is a report to respond to. |

### Actions

| Capability | Decision | Reason |
|---|---|---|
| Dispatch a workflow — `gh workflow run` | INTEGRATE | The only way to verify `nightly.yml`'s change gate without waiting for cron. |
| Watch a run to completion — `gh run watch --exit-status` | INTEGRATE | Turns "the workflow exists" into "the workflow succeeded" as a runnable verify. |
| Read job-level conclusions — `gh run view --json jobs` | INTEGRATE | The change-gate proof is "the build job was skipped on the second dispatch" — only readable at job granularity. |
| Re-run / cancel runs — `gh run rerun`, `gh run cancel` | OPT-OUT | No retry orchestration is being built; a failed run is a human's signal to fix the workflow. |
| Self-hosted runner management — `/repos/{owner}/{repo}/actions/runners` | OPT-OUT | GitHub-hosted `ubuntu-latest` only, matching the existing `ci.yml`. |

## Auth surface

Two distinct credentials, deliberately not interchangeable:

- **Inside workflows:** the automatic `secrets.GITHUB_TOKEN`, scoped by an
  explicit job-level `permissions: contents: write` block. No repository secret
  is created, referenced, or required by this phase — the same structural
  guarantee `.github/workflows/ci.yml` already documents in its header.
- **On the operator's machine:** the already-authenticated `gh` CLI
  (`repo`/`workflow` scopes) used by `scripts/sync-milestones.sh` and by the
  one-time PVR enable. The script stores no token and reads no secret file.
