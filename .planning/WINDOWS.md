---
schema_version: 1
open_count: 5
waived_count: 0
fixed_count: 0
total_count: 5
last_updated: 2026-08-08T19:09:03.587Z
---

# Broken Windows Ledger

> Cross-phase defect register. `/gsd-ship` blocks while `open_count > 0`.
> Waive with `gsd-tools windows waive <id> "<reason>"` (reason required).
> Mark fixed with `gsd-tools windows fixed <id>`.

| id | phase | kind | file | line | description | status | reason | recorded_at | resolved_at |
|----|-------|------|------|------|-------------|--------|--------|-------------|-------------|
| 1 | 03 | unrun-verify | plugins/proton/plugin.go |  | Task 1 human-check not run live: HTML email rendering in the detail pane iframe and plain-text fallback were not visually confirmed against a running kernel/webUI (no live Bridge email available this session). | open |  | 2026-07-31T02:27:34.866Z |  |
| 2 | 03 | unrun-verify | plugins/proton/live_bridge_test.go |  | TestSeenFlagUnchanged_LiveBridge (Proof 4) implemented but not executed against the real Bridge this session — live LOGIN is a known-broken credential issue (03-01-SUMMARY.md) and the Bridge rate-limits repeated failed logins, so the live run was deliberately not attempted to avoid extending the lockout. | open |  | 2026-07-31T02:27:39.001Z |  |
| 3 | 03 | unrun-verify | plugins/proton/plugin.go |  | Task 2 human-check not run: confirming in the real Proton web/mobile client that an email opened via webspaces still shows as unread was not performed live (blocked on the same Bridge credential issue as Proof 4). | open |  | 2026-07-31T02:27:44.466Z |  |
| 4 | 05 | unrun-verify | .planning/phases/05-source-instances-per-type-matching/05-05-PLAN.md |  | Task 3's <human-check> (visual confirmation of two named instances in the UI and post-D-11 rendition pixel parity for email/markdown/chat) was substituted with equivalent curl/API checks against a live ephemeral kernel instance, not an actual human eyeballing the running web UI — a human should open make dev and confirm visually. | open |  | 2026-08-06T14:13:27.733Z |  |
| 5 | 07 | deviation | kernel/supervisor/supervisor_test.go | 224 | TestApply_MidFlightSyncLeavesNoStrandedRunningRow has a pre-existing, intermittent (~1-in-6 under load) race between Coordinator.syncOne's detached sync_runs finalize write and the test's own read, reproducing identically on unmodified pre-07-09 code — unrelated to gaps[0], discovered during 07-09 verification, not fixed here because the plan prohibits editing this test's body. | open |  | 2026-08-08T19:09:03.587Z |  |

````json
[
  {
    "id": 1,
    "kind": "unrun-verify",
    "phase": "03",
    "file": "plugins/proton/plugin.go",
    "line": null,
    "description": "Task 1 human-check not run live: HTML email rendering in the detail pane iframe and plain-text fallback were not visually confirmed against a running kernel/webUI (no live Bridge email available this session).",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-07-31T02:27:34.866Z",
    "resolved_at": null
  },
  {
    "id": 2,
    "kind": "unrun-verify",
    "phase": "03",
    "file": "plugins/proton/live_bridge_test.go",
    "line": null,
    "description": "TestSeenFlagUnchanged_LiveBridge (Proof 4) implemented but not executed against the real Bridge this session — live LOGIN is a known-broken credential issue (03-01-SUMMARY.md) and the Bridge rate-limits repeated failed logins, so the live run was deliberately not attempted to avoid extending the lockout.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-07-31T02:27:39.001Z",
    "resolved_at": null
  },
  {
    "id": 3,
    "kind": "unrun-verify",
    "phase": "03",
    "file": "plugins/proton/plugin.go",
    "line": null,
    "description": "Task 2 human-check not run: confirming in the real Proton web/mobile client that an email opened via webspaces still shows as unread was not performed live (blocked on the same Bridge credential issue as Proof 4).",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-07-31T02:27:44.466Z",
    "resolved_at": null
  },
  {
    "id": 4,
    "kind": "unrun-verify",
    "phase": "05",
    "file": ".planning/phases/05-source-instances-per-type-matching/05-05-PLAN.md",
    "line": null,
    "description": "Task 3's <human-check> (visual confirmation of two named instances in the UI and post-D-11 rendition pixel parity for email/markdown/chat) was substituted with equivalent curl/API checks against a live ephemeral kernel instance, not an actual human eyeballing the running web UI — a human should open make dev and confirm visually.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-06T14:13:27.733Z",
    "resolved_at": null
  },
  {
    "id": 5,
    "kind": "deviation",
    "phase": "07",
    "file": "kernel/supervisor/supervisor_test.go",
    "line": 224,
    "description": "TestApply_MidFlightSyncLeavesNoStrandedRunningRow has a pre-existing, intermittent (~1-in-6 under load) race between Coordinator.syncOne's detached sync_runs finalize write and the test's own read, reproducing identically on unmodified pre-07-09 code — unrelated to gaps[0], discovered during 07-09 verification, not fixed here because the plan prohibits editing this test's body.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-08T19:09:03.587Z",
    "resolved_at": null
  }
]
````
