---
created: 2026-08-05T10:33:17.038Z
title: Signal schema-version verify-and-accept tooling
area: tooling
severity: minor
files:
  - plugins/signal/schemaguard.go
  - plugins/signal/schema_version_fixture_test.go
---

## Problem

Every Signal Desktop update that bumps `PRAGMA user_version` (e.g. 1730 → 1740, hit 2026-08-05) makes the Signal plugin fail loudly by name — which is the designed behavior (Phase 4 success criterion 5), but turns each Signal update into a manual investigation: introspect the live schema, check that the tables/columns the plugin reads (`messages`, `message_attachments`, `reactions`, conversation-name sources) are unchanged, then bump the pinned ceiling and rebuild. A plugin rebuild alone does not fix it because the ceiling is a hardcoded constant.

## Solution

Add a verify-and-accept procedure/tool so this recurring maintenance is a five-minute check instead of an investigation: a script or plugin subcommand that (1) opens the live DB read-only, (2) diffs the schema of exactly the tables/columns the plugin reads against a committed expectation fixture, (3) on a clean diff, reports "safe to accept" and points at the single ceiling constant to bump (or bumps it in a checked way). Never auto-accept on a dirty diff — the loud failure stays the default.

Note: the immediate 1740 acceptance itself is a separate quick task (see STATE.md session context 2026-08-05); this todo is the durable tooling so the next bump is cheap.
