# Phase 14 Live UAT — Google Drive Plugin Against a Real Google Account

This is the recorded, runnable script for the parts of Phase 14 (SRC-05,
SRC-06) that need a real Google account — nothing here can be driven by
the hermetic Playwright harness. See
`web/e2e/specs/14-gdrive-external-rehearsal.spec.ts` and
`docs/testing.md`'s own section on that spec for everything a browser
*can* prove; this document covers exactly what's left: authorizing once
and surviving a restart, real documents with real previews (including
Workspace-native export), incremental sync, and the health states that
need a real revoked/inaccessible condition to reach honestly.

**No credential value is ever written into this file or any file in this
repository or the plugin repository.** Every step below names only the two
environment variable names, `GDRIVE_CLIENT_ID` and `GDRIVE_CLIENT_SECRET`
— you export their values into your own shell, never into a file.

Run this end to end against your own Google account, then fill in the
results table at the end.

---

## Preparation

1. **Build (or confirm) the plugin binary.** The binary comes from the
   separate `topos-plugin-gdrive` repository (D-08), never this one — see
   `.planning/phases/14-google-drive-source-built-out-of-repo/
   14-03-SUMMARY.md` for the hand-off evidence. Confirm it exists and is
   executable:
   ```bash
   file /home/darren/projects/davison/topos-plugin-gdrive/topos-plugin-gdrive
   ```
   Expect a static Go ELF executable. Running it directly with no
   arguments should print the standard `hashicorp/go-plugin`
   subprocess-safety message and exit 1 — that is correct, expected
   behaviour for a plugin binary run outside a plugin host, not a bug.

2. **Copy the built binary into the host's external plugin directory —
   never a trusted directory, never this repository's own plugin output.**
   Per `docs/plugin-contract.md`'s "Trust tiers" section, the external
   directory defaults (with no config required) to
   `$XDG_DATA_HOME/topos/plugins-external` (falling back to
   `~/.local/share/topos/plugins-external` when `XDG_DATA_HOME` is unset)
   on Linux:
   ```bash
   mkdir -p ~/.local/share/topos/plugins-external
   cp /home/darren/projects/davison/topos-plugin-gdrive/topos-plugin-gdrive \
     ~/.local/share/topos/plugins-external/topos-plugin-gdrive
   ```
   Do **not** copy it into `bin/plugins/` (this repository's own trusted
   directory) or set `[plugins] dir` to point at it — every proof in this
   phase runs the binary on the external, untrusted path deliberately (see
   this plan's own threat register, T-14.4-04).

3. **Export the two credential environment variables in the shell that
   will start the kernel** (and, separately, in the shell you run the auth
   command from — see "Authorization" below; the auth CLI and the running
   plugin instance read the identical two variable names, per
   `14-PLUGIN-PRD.md`'s Locked Decisions):
   ```bash
   export GDRIVE_CLIENT_ID=<your OAuth client id>
   export GDRIVE_CLIENT_SECRET=<your OAuth client secret>
   ```
   These values must never be written into `config.toml`, this file, or
   any file in either repository — only the two variable *names* ever
   appear in configuration or documentation.

4. **Start the kernel against the development config, not the production
   one.** `topos serve` accepts a `--config <path>` flag and a
   `TOPOS_CONFIG` environment variable (14-01-PLAN.md), taking precedence
   over the default `~/.config/topos/config.toml`:
   ```bash
   topos serve --config ./config.dev.toml
   ```
   or, equivalently:
   ```bash
   TOPOS_CONFIG=./config.dev.toml topos serve
   ```
   Running this UAT against the development config means a mistake here —
   a bad extras value, a source that fails to sync, a config write gone
   wrong — never touches your real, production config, index, or plugin
   set. Use `make dev-config` first if `config.dev.toml` does not exist
   yet in your checkout.

---

## Authorization

5. **Run the plugin's own standalone auth subcommand in a terminal** —
   never something the kernel launches or drives (`14-PLUGIN-PRD.md`'s
   Locked Decisions: "the host that launches this plugin must never see
   or compose an OAuth URL of any kind"):
   ```bash
   ~/.local/share/topos/plugins-external/topos-plugin-gdrive auth
   ```
   Expect it to open your browser to a Google sign-in / consent screen.

6. **Follow the browser flow, and click through the "unverified app"
   warning.** This is expected and correct for a personal-use OAuth
   client that has been published to Production status without
   completing Google's full verification process (see the D-03 note
   below) — it is not a sign anything is misconfigured.

7. **D-03 check — confirm your Cloud Console consent screen reads as
   Production, not Testing, before you rely on this UAT's result.** A
   Testing-status app's refresh tokens silently expire after **seven
   days**; a Production-status app's tokens live until you revoke them.
   Publishing to Production and being "Verified" are two different
   things — an unverified, personal-use app published to Production is
   fine and does not need Google's verification/CASA process. Record
   **today's date** in the results table below as the authorization date:
   if this source silently goes unreachable roughly a week from now with
   no config change in between, that is the Testing-status symptom, not a
   plugin bug — re-check the consent screen's publishing status first.

8. **Confirm a token file was written with owner-only permissions**
   under the XDG data directory (`14-PLUGIN-PRD.md`'s Locked Decisions:
   `~/.local/share/topos-plugin-gdrive/token.json` on Linux, mode
   `0600`):
   ```bash
   ls -l ~/.local/share/topos-plugin-gdrive/token.json
   ```
   Expect `-rw-------` (owner read/write only, nothing for group/other).

---

## Install

9. **Add the source through the add-source picker.** In the SPA, open
   "Add source" and choose the Google Drive plugin type.

10. **Confirm the plugin appears in the untrusted group** of the picker
    (never mixed in with the trusted, in-repo plugin types).

11. **Read the untrusted-confirm interstitial and confirm its dynamic
    disclosure line names both credential variables by name.** Per
    `14-UI-SPEC.md`'s "Explicit Reuse" table, this text is generated
    generically from the source's own configured `${VAR}` references —
    once the two credential fields are filled in (next step), the
    interstitial's disclosure line should read, verbatim in shape:
    `"...plus the values behind these variables referenced in this
    source's own configuration: GDRIVE_CLIENT_ID, GDRIVE_CLIENT_SECRET."`
    If either variable name is missing from that sentence, stop and
    record it as a defect — this is the one place an operator is told,
    in plain language, exactly what credential values are being handed
    to an unsandboxed third-party binary.

12. **Fill in the three declared fields** — `OAuth Client ID`, `OAuth
    Client Secret`, `Drive Folder ID` (the id segment of the target
    folder's Drive URL, per this plan's `user_setup`). Confirm the two
    credential fields show only the **set/unset badge for the variable
    name**, never the value itself — `SecretField`'s existing behaviour,
    unmodified by this phase.

13. **Set the folder match values** in the match-vocabulary editor
    (`Folders`, per `14-UI-SPEC.md` G3) to whatever folder-path segment
    you want a webspace to pick this source up by.

14. **Save, and confirm the chip carries the untrusted badge and a
    pinned hash** in its dropdown footer — the standard external-tier
    consent-and-pin flow (`docs/plugin-contract.md`'s "Pinning" section),
    unchanged by this phase.

---

## Criterion 1 — Authorize once, survive a restart

**Proves:** SRC-05's "the operator authorizes once and the source keeps
syncing across host restarts without ever needing to re-authorize."

- Confirm the source syncs successfully once (chip shows a healthy tone,
  the configured webspace's stream is populated).
- Restart the kernel (`Ctrl-C` the `topos serve` process and start it
  again against the same `--config`).
- **Passing observation:** the source syncs again with **no** further
  authorization step — no browser opens, no health message asking for
  `auth` to be re-run.
- **Disproving observation:** the source reports the "Not authorized"
  health sentence after a restart, even though `auth` was run
  successfully before the restart.
- **Record:** the exact date/time you completed step 6 above (first
  authorization) — this is what lets a later, silent seven-day expiry
  (Pitfall 1, `14-RESEARCH.md`) be dated and distinguished from a genuine
  restart-survival failure.

## Criterion 2 — Documents, previews, deep links

**Proves:** SRC-05's "documents in the configured folder appear in the
stream with previews, including Workspace-native documents rendered via
export."

- Confirm every document in the configured Drive folder appears as a
  stream item in the attached webspace.
- Open a native PDF item and a native image item — both should render
  inline in the detail pane (the same Fetch-bytes+MIME pipeline Phase 12
  proved for the filesystem plugin).
- Open a Google Doc, a Google Sheet, and a Google Slides deck — each
  should render a preview via export (per `14-PLUGIN-PRD.md`'s Drive API
  Surface table: `files.export`, capped at 10 MB).
- Find or create a document whose export would exceed the 10 MB cap, or
  a format the plugin declines to export — confirm it shows the
  existing "No preview available" fallback (`DetailPane.svelte`'s final
  `{:else}` branch), **never** a truncated or blank render. This is one
  of the two specific failure modes this criterion watches for: a
  silently truncated export is a worse failure than an honest "no
  preview," because it looks like a complete, correct result.
- Open every item's "Open in {displayName}" action — confirm it opens
  the Drive web UI in a new tab (`https://drive.google.com/...`), not a
  broken link or a `file://` URL.
- **Passing observation:** every document type above renders correctly,
  and the capped/declined case shows the honest fallback.
- **Disproving observation:** any document silently missing, a
  Workspace-native document with no preview at all (export never
  attempted), or a capped export rendering as truncated/corrupted
  content instead of the no-preview fallback.

## Criterion 3 — Incremental sync after the first

**Proves:** SRC-05's "every sync after the first is incremental" and the
roadmap's own success criterion that the second and subsequent syncs
pull only changed items from Google, not a full folder re-listing.

- Trigger a second sync (via "Refresh now" on the chip, or waiting for
  the configured sync interval).
- Confirm from the plugin's own logs (stdout/stderr, captured by the
  kernel process that launched it) that the second sync used the Drive
  Changes API delta path (`changes.list` against a persisted page token)
  rather than a full `files.list` walk of the folder again.
- **Confirm the stream still shows the full folder's contents after the
  second sync — not only the items that changed.** This is the second
  specific failure mode this criterion watches for (`14-RESEARCH.md`
  Pitfall 3): the host's own `Match` contract replaces a source's whole
  item set on every sync, so a plugin that mistakenly returns only the
  delta from its own second `Match` call will make the stream visibly
  *shrink* to just the recently-changed items — a shrunken stream after
  a "successful" second sync is exactly this bug, not a sign of a small
  folder.
- **Passing observation:** plugin logs show a delta-shaped Drive API
  call on the second sync, and the stream still shows every document
  from the configured folder (unchanged items included).
- **Disproving observation:** plugin logs show a full folder re-listing
  on the second sync, or the stream shrinks to only recently-changed
  items.

## Criterion 4 — Trust surface and gap log

**Proves:** SRC-06's "the plugin loads through the host's external,
untrusted plugin path" and its own gap-logging discipline.

- Confirm the chip carries the untrusted badge (from Install step 14).
- Confirm the chip's dropdown footer shows a pinned hash, and that the
  hash matches the SHA-256 of the binary you actually copied into the
  external plugins directory:
  ```bash
  sha256sum ~/.local/share/topos/plugins-external/topos-plugin-gdrive
  ```
- Note, in the results table below, anything the published contract or
  the vendored mock plugin failed to answer during this run — this goes
  into `CONTRACT-GAPS.md` in the plugin repository (D-07), for 14-05's
  own gap-triage plan to pull back into `docs/plugin-contract.md`, never
  fixed silently here.
- **Passing observation:** badge present, pin present and matching, and
  any gap encountered during this run is written down rather than
  worked around silently.
- **Disproving observation:** the source loaded without a trust badge or
  pin (would mean it launched from the trusted directory by mistake —
  stop and re-check Preparation step 2), or a real gap was hit and not
  recorded.

---

## Health states

For each of the four named sentences (`14-UI-SPEC.md` G4 /
`14-PLUGIN-PRD.md`'s Health States table), reach it deliberately where
you can, and confirm it surfaces through the existing, unmodified chip
tone/tooltip and `StreamSyncDegraded.svelte` banner — never a new
surface.

| Cause | Exact sentence | How to reach it deliberately | Reached this run? |
|---|---|---|---|
| Never authorized | `Not authorized — run "topos-plugin-gdrive auth" in a terminal, then use this source's "Refresh now".` | Add the source before running Authorization step 5 — this is exactly the state the hermetic e2e spec proves without a Google account. | |
| Expired or revoked | `Authorization expired or was revoked — run "topos-plugin-gdrive auth" again, then use this source's "Refresh now".` | In your Google Account's own "Third-party apps & services" settings, revoke access for this OAuth client, then trigger a sync. | |
| Rate limited | `Rate limited by Google Drive — retrying automatically. No action needed.` | Genuinely difficult to provoke deliberately against a personal-use quota (`14-RESEARCH.md` Pitfall 6) — note honestly if this state could not be reached this run rather than marking it passed. | |
| Folder inaccessible | `The configured Drive folder is no longer accessible — check the folder still exists and is shared with this account.` | Remove or unshare the configured folder from the authorized account, then trigger a sync. | |

For every state you *do* reach, confirm:
- `GET /api/sources`'s `reachable` is `false` and `last_error` is the
  exact sentence above (not a paraphrase).
- The chip renders the `destructive` tone (never `warning`, since none
  of these states is "reachable but erroring" — see `14-UI-SPEC.md`'s
  Color section).
- If the affected webspace's stream goes empty because of it,
  `StreamSyncDegraded.svelte`'s banner shows the same sentence, wrapped
  naturally, not truncated.
- The "Refresh now" action the sentence names is the chip's existing
  dropdown menu item — unmodified, not a new affordance.

**Note which of the four states you could not reach this run, rather
than marking them passed.** An unreached state is not a failure of this
UAT — it is an honest gap this record should carry forward, per this
plan's own threat register (T-14.4-06: "a health state marked passed
that was never reached").

---

## Results Table

Fill in during the run. `Observed` is yes/no/not-reached; `Evidence` is a
short pointer (screenshot filename, log excerpt, timestamp) — no
credential value, ever.

| # | Item | Observed | Evidence | Notes |
|---|------|----------|----------|-------|
| 1 | Criterion 1 — restart survives with no re-auth | | | First authorization date: __________ |
| 2 | Criterion 2 — documents, previews (native + Workspace export), deep links | | | |
| 3 | Criterion 3 — second sync is incremental, stream stays full | | | |
| 4 | Criterion 4 — untrusted badge, matching pin, gap log entries recorded | | | |
| 5 | Health — never authorized | | | |
| 6 | Health — expired/revoked | | | |
| 7 | Health — rate limited | | | |
| 8 | Health — folder inaccessible | | | |

**Anything to carry into the gap triage (14-05):**

_(fill in — reference `CONTRACT-GAPS.md` entries by id where applicable)_

---

*Phase: 14-google-drive-source-built-out-of-repo*
*Plan: 14-04*
