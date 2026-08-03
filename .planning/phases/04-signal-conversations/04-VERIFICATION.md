---
phase: 04-signal-conversations
verified: 2026-08-03T23:52:12Z
status: human_needed
score: 5/5 truths verified (ROADMAP success criteria)
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: "5/5 truths verified (ROADMAP success criteria); 2 items routed to human verification"
  gaps_closed:
    - "G-04-1: Signal contact-form sgnl:// deep link now emits a literal '+' (E.164 allowlist, validate-and-refuse) instead of percent-encoding it to %2B — root cause fixed in plugins/signal/deeplink.go, regression guard rewritten in deeplink_test.go, end-to-end shape guard added to scripts/signal-readonly-smoke.sh, live index re-synced (105/105 rows carry the corrected shape, 0 rows carry the rejected shape), 04-SECURITY.md corrected (T-04-14 superseded, T-04-17/18/19 added)"
    - "Judgment-tier prohibitions sign-off (privacy/safety: log hygiene, no on-disk plaintext copy, profile-name anti-spoofing) — 04-UAT.md test 2 recorded human pass 2026-08-03"
  gaps_remaining: []
  regressions: []
human_verification:
  - test: "Click 'Open in Signal' on a Signal digest for a 1:1 conversation whose contact has a known E.164 number, with the rebuilt server running."
    expected: "Signal Desktop raises and shows that contact's conversation; no 'Something went wrong! Sorry, that sgnl:// link didn't make sense!' error modal appears."
    why_human: "This verifier confirmed the emitted link's byte shape is correct (literal '+', no percent sign — proven by the unit boundary matrix and by direct sqlite3 queries against the developer's live index: 105 rows match the corrected glob, 0 rows match the rejected glob) and confirmed the running server is serving the rebuilt binary (bin/webspaces, built 00:47, server process started 00:49, identical file by cmp). It cannot itself observe Signal Desktop's rendered window state or click the button. 04-UAT.md tracks this as gap G-04-1 (status: diagnosed) — this is the only remaining check needed to close it."
---

# Phase 4: Signal Conversations Verification Report

**Phase Goal:** User's Signal conversations for a topic appear in the webspace stream, read from Signal Desktop's local database without any risk of corrupting or altering it.
**Verified:** 2026-08-03T23:52:12Z
**Status:** human_needed
**Re-verification:** Yes — after gap closure (04-04-PLAN.md, G-04-1)

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Messages from Signal conversations/groups whose names match the webspace keyword appear in the stream chronologically alongside documents, notes and mail | ✓ VERIFIED | Unchanged by this plan. Regression check: `make test-signal` passes; live index holds 111 Signal rows (`sqlite3 ... select count(*) from items where source_type='signal'` = 111); no code in digest.go/plugin.go touched by 04-04. |
| 2 | Detail pane shows the surrounding conversation thread, with an "open in Signal" affordance declared conversation-only fidelity | ✓ VERIFIED (both link forms) | **This is the criterion G-04-1 blocked.** Now closed at the code/data level: `plugins/signal/deeplink.go`'s `conversationDeepLink` emits `sgnl://signal.me/#p/+15551234567` verbatim (no percent-encoding) for a valid E.164, confirmed by direct read of the file and by running `TestDeepLink_PlusSignEmittedLiterally` and the 14-case `TestDeepLink_E164BoundaryMatrix` (all pass). Live index re-sync confirmed independently by this verifier: `sqlite3 -readonly ~/.local/share/webspaces/index.db` — 105 rows glob `sgnl://signal.me/#p/+*` (literal plus), 0 rows glob the contact-form prefix without a literal plus. `web/src/lib/components/OpenInSource.svelte` passes `link.url` through unmodified into `href` — no client-side re-encoding. The one remaining piece — the visual confirmation that Signal Desktop actually raises and navigates correctly — is a human click, routed below as the sole human-verification item. The bare-scheme (group) form was already visually approved during 04-01's checkpoint. |
| 3 | Signal Desktop's database is opened strictly read-only (`mode=ro`) and is byte-identical after a full sync, including while Signal Desktop is running | ✓ VERIFIED | Unchanged by this plan (deeplink.go performs no DB I/O). Regression check: `./scripts/signal-readonly-smoke.sh` ran through its hash-before/sync/hash-after steps before exiting on an unrelated port conflict (see Anti-Patterns/Notes below) and reported `db.sqlite hash unchanged: 3f95392690faafa395ff3adf6a544fc2cc2973551d55286059ae381eab1b2474` — confirms criterion 3 still holds. `dsn.go`/`readonly_test.go`/`byte_identical_test.go` untouched by 04-04 and still pass under `make test-signal`. |
| 4 | The decryption key is obtained from whichever keyring backend the user's install actually uses, detected at runtime rather than assumed | ✓ VERIFIED | Unchanged by this plan. `keyresolve.go`/`secretservice.go` untouched by 04-04; `make test-signal` (which includes `TestResolveKey_*`) passes. |
| 5 | An unrecognised Signal database schema version fails loudly, naming the version it found, instead of silently importing nothing | ✓ VERIFIED | Unchanged by this plan. `schemaguard.go` untouched by 04-04; `make test-signal` (which includes `TestSchemaVersionCeiling`) passes. |

**Score:** 5/5 ROADMAP success criteria verified. Criterion 2's remaining gap (G-04-1) is closed at every automatable layer; only the human window-raise click remains, tracked as the sole human-verification item below.

### Gap Closure Verification (G-04-1)

| Must-have (04-04-PLAN.md frontmatter) | Status | Evidence |
|---|---|---|
| "Clicking 'Open in Signal' on a 1:1 Signal digest whose contact has an E.164 raises Signal Desktop and lands on that contact's conversation — no error modal" | ⚠️ Awaiting human click | Everything an automated check can prove is proven (see below); the visual confirmation is the one item that must come from the developer. |
| "The contact-form link the plugin emits carries a literal '+' immediately after '#p/'" | ✓ VERIFIED | `plugins/signal/deeplink.go` line 71: `return "sgnl://signal.me/#p/" + e164` — direct string concatenation, no escaping function in the call path (confirmed `encodePhoneFragment` and its `net/url` import are deleted: `grep -n 'net/url' plugins/signal/deeplink.go` = no match). `TestDeepLink_PlusSignEmittedLiterally` asserts the exact literal string and asserts no `%` appears. |
| "A private conversation whose stored number is not a valid E.164 gets the bare 'sgnl://' link, never a contact-form link Signal would reject" | ✓ VERIFIED | `TestDeepLink_NonE164FallsBackToBareForm` plus 12 of the 14 `TestDeepLink_E164BoundaryMatrix` cases (metacharacters, leading zero, wrong length, no leading plus, whitespace padding) all pass — all resolve to the bare `sgnl://` form. |
| "Every Signal row already in the developer's live index carries the corrected link — no stale row keeps the rejected form" | ✓ VERIFIED (independently re-run by this verifier, not just trusted from SUMMARY) | `sqlite3 -readonly ~/.local/share/webspaces/index.db "select count(*) from items where source_type='signal' and deep_link glob 'sgnl://signal.me/#p/+*';"` → **105**. `sqlite3 -readonly ~/.local/share/webspaces/index.db "select count(*) from items where source_type='signal' and deep_link glob 'sgnl://signal.me/#p/*' and not deep_link glob 'sgnl://signal.me/#p/+*';"` → **0**. Total Signal rows: 111 (105 contact-form + 6 bare-form, consistent with a mix of groups and E.164-less 1:1s). |
| "No test in the repo asserts the rejected payload shape any more" | ✓ VERIFIED | `grep -c 'func TestEncodePhoneFragment_PlusSignEscaped' plugins/signal/deeplink_test.go` = 0; `grep -c 'func TestDeepLink_UnsafeCharactersAreEscapedNotEmittedRaw' plugins/signal/deeplink_test.go` = 0. |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `plugins/signal/deeplink.go` | E.164 allowlist validation, verbatim emission, no fragment re-encoding | ✓ VERIFIED | Read in full. `e164Pattern = regexp.MustCompile(`^\+[1-9][0-9]{1,14}$`)`, `isValidE164`, `conversationDeepLink` refuses-and-falls-back per the plan. `net/url` import gone; `regexp` import present. |
| `plugins/signal/deeplink_test.go` | Literal-'+' regression guard plus non-E.164 fallback matrix | ✓ VERIFIED | Read in full; 7 test functions, including the 14-case `TestDeepLink_E164BoundaryMatrix`. All pass (`go test -run TestDeepLink ./plugins/signal/...`). |
| `scripts/signal-readonly-smoke.sh` | End-to-end deep-link shape assertion over real stream JSON | ✓ VERIFIED (present + wired; not independently re-run to completion this session — see Notes) | `grep`-confirmed the new block exists after the existing zero-items assertion, iterates `.link.url` for `source_type=="signal"` contact-form links, fails naming offending URLs on a missing literal `+`, and prints a non-vacuity note when the count is 0. |
| `.planning/phases/04-signal-conversations/04-SECURITY.md` | T-04-14 marked superseded, T-04-17/18/19 recorded | ✓ VERIFIED | Read in full. T-04-14 row kept, disposition text names the supersession and points at T-04-17. T-04-17 (mitigate/closed), T-04-18 (accept/closed), T-04-19 (accept/closed) all present with the argued evidence. Audit trail table has a 2026-08-04 row. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `plugins/signal/plugin.go:226` `toItem` | `plugins/signal/deeplink.go` `conversationDeepLink` | Single construction site, unchanged by this plan | ✓ WIRED | Confirmed by reading `deeplink.go`'s doc comment cross-reference and by the fact 04-04 touched no other Signal file besides deeplink.go/deeplink_test.go/SECURITY.md/smoke-script — fixing the builder alone reaches every item, as the plan claimed. |
| `kernel/index` `items.deep_link` | Live index rows | Written at sync time, overwritten only by a later sync | ✓ WIRED | Independently confirmed: 105/111 Signal rows carry the corrected shape (queried directly above), consistent with a completed re-sync after the rebuild. |
| `kernel/pluginhost/host.go` `launch()` | Running `webspaces serve` process | Spawns plugin subprocess once per kernel process | ✓ WIRED, confirmed current build | The one running `webspaces serve` process (PID 3218549) started at 00:49:30, after `bin/webspaces` was last built (00:47). `cmp /proc/3218549/exe bin/webspaces` reports the running binary is byte-identical to the current `bin/webspaces` on disk — the live server is not running a stale pre-fix build. |
| `web/src/lib/components/OpenInSource.svelte` | `href={link.url}` → OS `sgnl://` scheme handler | String passed through unmodified | ✓ WIRED | Read the component in full: `<Button href={link.url} ...>` — no transformation, no re-encoding, no `encodeURIComponent` call anywhere in the file. |

### Behavioral Spot-Checks / Probe Execution

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full workspace test suite (single run) | `make test` | exit 0 — kernel, sdk, every plugin including cgo Signal module | ✓ PASS |
| Signal plugin build (cgo) | `make signal` | exit 0 — confirms `net/url` import removal doesn't break compilation and no new unused import lingers | ✓ PASS |
| Deep-link unit tests, named run | `CGO_ENABLED=1 go test -tags libsqlcipher -run TestDeepLink ./plugins/signal/...` | exit 0, 7 top-level funcs / 14+3 subtests all PASS | ✓ PASS |
| Superseded test names absent | `grep -c 'func TestEncodePhoneFragment_PlusSignEscaped\|func TestDeepLink_UnsafeCharactersAreEscapedNotEmittedRaw' deeplink_test.go` | both 0 | ✓ PASS |
| Live index shape query 1 (literal-plus rows exist) | `sqlite3 -readonly index.db "select count(*) from items where source_type='signal' and deep_link glob 'sgnl://signal.me/#p/+*';"` | 105 | ✓ PASS |
| Live index shape query 2 (no rejected-shape rows) | `sqlite3 -readonly index.db "select count(*) from items where source_type='signal' and deep_link glob 'sgnl://signal.me/#p/*' and not deep_link glob 'sgnl://signal.me/#p/+*';"` | 0 | ✓ PASS |
| `scripts/signal-readonly-smoke.sh` | `./scripts/signal-readonly-smoke.sh` | Ran through hash-before → sync (1467 items) → hash-after (unchanged) → **stopped**: `FAIL: something is already listening on 127.0.0.1:7777` | ? SKIPPED (see Notes) |

**Notes on the smoke-script run:** the script's own port-conflict guard correctly refused to proceed once it detected the developer's live `webspaces serve` (the one intentionally left running, rebuilt, for the pending human click) already bound to :7777 — exactly the vacuity guard the smoke script is designed to enforce (it will not let its checks pass against a server it didn't start itself). Stopping that server to force the smoke script through would have undone the exact state (rebuilt server, ready for the human click) this re-verification run was asked to preserve. The parts of the smoke script that did run before the guard fired (hash-before, sync, hash-after) independently reconfirm criterion 3 (byte-identical). The specific new assertion this task added — the contact-form literal-'+' shape guard over `STREAM_JSON` — was not executed end-to-end in this session, but its logic was read and confirmed correct, and equivalent evidence (the same shape assertion, run directly against the same live index data) was obtained via the two sqlite3 queries above. This is judged sufficient; not re-verified as a gap.

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|-----------------|-------------|--------|----------|
| SRC-02 | 04-01, 04-02, 04-03, 04-04 | Signal plugin reads Signal Desktop DB strictly read-only (`mode=ro`); extracts key via OS keyring (backend-detected); detects schema version and fails loudly on unknown | ✓ SATISFIED | All 5 ROADMAP criteria verified above; REQUIREMENTS.md marks SRC-02 `[x]` / "Phase 4 / Complete". 04-04-PLAN.md declares `requirements: [SRC-02]` in frontmatter — consistent with the other three plans. |

No orphaned requirements — SRC-02 is the only requirement ID mapped to Phase 4 in REQUIREMENTS.md, and all four plans (including the gap-closure plan) declare it.

### Anti-Patterns Found

None. `grep -n -E "TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER" plugins/signal/deeplink.go plugins/signal/deeplink_test.go scripts/signal-readonly-smoke.sh` returns only a `mktemp ... XXXXXX` template string (not a debt marker). No stub patterns, no empty handlers, no hardcoded-empty data flowing to rendering.

### Human Verification Required

1. **UAT test 1 re-run — E.164 contact-form `sgnl://` deep link, visual confirmation**
   - **Test:** With the rebuilt server running (confirmed current at time of this verification), open a webspace containing a 1:1 Signal digest whose contact has a phone number, open its detail pane, and click "Open in Signal".
   - **Expected:** Signal Desktop raises and shows that contact's conversation. No "Something went wrong! Sorry, that sgnl:// link didn't make sense!" dialog appears.
   - **Why human:** This verifier proved the emitted link's byte shape is correct at every layer it can inspect — the builder function, its unit tests, the live index data, and the DOM wiring that passes the URL through unmodified — and confirmed the running server process is the rebuilt binary, not a stale one. It cannot itself click a button or observe Signal Desktop's window state. 04-UAT.md's gap G-04-1 remains in `status: diagnosed` pending exactly this check.

### Gaps Summary

No blocking gaps. G-04-1's root cause (percent-encoded `+` in the contact-form deep link) is fixed in code, proven by a rewritten unit boundary matrix, and independently confirmed against the developer's live index (105 corrected rows, 0 rejected-shape rows) by this verifier directly — not merely trusted from 04-04-SUMMARY.md's claims. `04-SECURITY.md`'s register is corrected to match. The phase's other four ROADMAP criteria are unaffected by this plan and remain verified by regression (`make test` passing, files untouched). The single remaining item — the human click confirming Signal Desktop's actual window-raise and navigation — is exactly the check 04-04-PLAN.md itself designated as the only thing that can close G-04-1, and it is now ready to perform against the current build.

---

_Verified: 2026-08-03T23:52:12Z_
_Verifier: Claude (gsd-verifier)_
