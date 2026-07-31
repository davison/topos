---
schema_version: 1
open_count: 3
waived_count: 0
fixed_count: 0
total_count: 3
last_updated: 2026-07-31T02:27:44.466Z
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
  }
]
````
