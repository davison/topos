---
phase: 07-webspace-builder-ui
plan: 13
subsystem: ui
tags: [svelte, sveltekit, go-plugin, forms, validation, error-diagnostics]

requires:
  - phase: 07-webspace-builder-ui
    provides: "07-04's ConnectionForm/AddSourceModal/EditSourceModal two-step add and Edit connection… flows; 07-02's DescribePluginType trial-launch and launch()"
provides:
  - "CONNECTION_FIELDS' required flags re-derived from all four plugins' own pre-goplugin.Serve fatal guards (paperless, silverbullet, proton, signal), not just the UAT-reported field"
  - "Signal's Local Path field seeded with a real editable default (~/.config/Signal), the only field in the table permitted one"
  - "defaultConnectionValues/missingRequiredFields/missingRequiredFieldsMessage — pure helpers called by every submit path that can launch a plugin"
  - "ConnectionForm.svelte's DOM required attribute on both the plain-Input and SecretField branches"
  - "AddSourceModal's Connect step, Save anyway, and EditSourceModal's Edit connection… all refuse to submit while a required field is blank, issuing no request"
  - "kernel/pluginhost.launch captures a bounded, mutex-guarded tail of the plugin's stderr and appends its last line to a pre-handshake connect failure — covering trial launches and boot-time/hot-apply launches identically"
affects: [webspace-builder-ui, plugin-contract]

actuals:
  tokens: 10871
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Derived-from-source-of-truth table: a UI-side config table's required flags are re-derived by reading the plugin's own pre-Serve fatal guards, not patched ad hoc against the one field a bug report named"
    - "Bounded mutex-guarded tail buffer (front-discard) wired into a third-party library's default-io.Discard writer hook, read only after the library's own Kill()/wait-group synchronization point"

key-files:
  created:
    - kernel/pluginhost/stderr_test.go
  modified:
    - web/src/lib/plugin-fields.ts
    - web/src/lib/plugin-fields.test.ts
    - web/src/lib/components/ConnectionForm.svelte
    - web/src/lib/components/SecretField.svelte
    - web/src/lib/components/AddSourceModal.svelte
    - web/src/lib/components/EditSourceModal.svelte
    - web/src/lib/components/add-source.test.ts
    - kernel/pluginhost/host.go

key-decisions:
  - "Required flags derived by reading all four plugins/*/main.go pre-Serve guards, not just the two the UAT report happened to hit — closes the whole class rather than patching one report"
  - "Only Signal's path gets a seeded default value; Proton's webmail_base_url stays required with a placeholder only, since its correct value is installation-specific and a wrong default would silently break deep links"
  - "Enforcement lives in three pure helper functions called by each submit handler, not inside ConnectionForm itself — the shared component cannot own the load-bearing check because each of its three consumers issues its own request"
  - "stderrTail's cap set at 4 KiB (comment states the reasoning): large enough for a fatal line or the tail of a stack trace, small enough to be irrelevant to kernel memory even held for a plugin's whole lifetime"
  - "Server-side required-field validation in DescribePluginHandler deliberately NOT added (planning choice 7) — no connection-field schema exists on the wire (07-RESEARCH.md), and the stderr capture already converts a blank field reaching the kernel into a self-explanatory failure rather than a silent one; a wire schema belongs to a future plugin-contract phase"

patterns-established:
  - "A plugin config field is 'required' in the UI if and only if the plugin's own pre-Serve guard fatals on it empty — verified by table-truth tests naming the specific main.go guard each row mirrors, so a future plugin type's row must be derived the same way rather than copied from a sibling"

requirements-completed: [KERN-08, UI-12]

coverage:
  - id: D1
    description: "CONNECTION_FIELDS' required flags re-derived from all four plugins' pre-Serve fatal guards; Signal's path seeded with a real default; three pure helpers (defaultConnectionValues, missingRequiredFields, missingRequiredFieldsMessage) exported"
    requirement: "UI-12"
    verification:
      - kind: unit
        ref: "web/src/lib/plugin-fields.test.ts — table-truth tests per plugin binary, default-value-count test, missingRequiredFields/missingRequiredFieldsMessage behavior tests"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every UI path that can launch a plugin (Connect step, Save anyway, Edit connection…) refuses to submit with a required field blank, and ConnectionForm marks required fields with the DOM required attribute"
    requirement: "UI-12"
    verification:
      - kind: unit
        ref: "web/src/lib/components/add-source.test.ts — 'G-07-5 guard' describe blocks (source-order assertions ahead of describePlugin(/putConfig(, ConnectionForm required-attribute assertions, EditSourceModal submitConnection ordering)"
        status: pass
    human_judgment: true
    rationale: "The plan's own verification section requires a live-kernel human-check (make dev, New Signal…/New Proton… flows, actually clicking Next on a blank field and confirming no network request) that this autonomous continuation did not perform — no browser or live kernel was driven this run. The automated source-scan and unit coverage above is fully proven; the end-to-end browser confirmation is still outstanding."
  - id: D3
    description: "kernel/pluginhost.launch captures a bounded, mutex-guarded tail of the plugin's stderr and appends its last non-empty line to a pre-handshake connect-failure error; silent exits are byte-identical to before; race-clean"
    requirement: "KERN-08"
    verification:
      - kind: unit
        ref: "kernel/pluginhost/stderr_test.go — TestDescribePluginType_PreHandshakeFatalSurfacesPluginStderr, TestDescribePluginType_SilentPreHandshakeExitLeavesErrorUnchanged, TestStderrTail_BoundedRetainsMostRecentOutput, TestStderrTail_LastLineTrimsTrailingWhitespaceAndNewlines, TestStderrTail_EmptyBufferYieldsEmptyString, TestStderrTail_ConcurrentWritersThenRead"
      - kind: integration
        ref: "go test ./kernel/... -count=1 -race (whole kernel suite, all packages ok)"
        status: pass
    human_judgment: false

duration: ~18min (Task 1 in a prior session ending at commit 3e51eb2 13:06:43+01:00; this continuation — Tasks 2 and 3 — ran commit-to-commit in ~13min, 13:06:43 to 13:19:12+01:00)
completed: 2026-08-09
status: complete
---

# Phase 07 Plan 13: Blank-Required-Field Guard + Plugin Stderr Capture Summary

**A blank required connection field (Signal's path, Proton's webmail_base_url, or either plugin's other mandatory fields) can no longer reach a plugin subprocess from any UI submit path, and a plugin that dies before the go-plugin handshake now explains itself through its own captured stderr line instead of go-plugin's generic four-item diagnostic list.**

## Performance

- **Duration:** ~18 min total across the plan (Task 1 completed in a prior session; this continuation executed Tasks 2–3)
- **Started (Task 1 commit):** 2026-08-09T13:06:43+01:00
- **Completed (Task 3 commit):** 2026-08-09T13:19:12+01:00
- **Tasks:** 3/3
- **Files modified:** 9 (8 modified, 1 new)

## Accomplishments

- **Task 1 (prior session, verified present, not redone):** `CONNECTION_FIELDS`' required flags re-derived from every plugin's own pre-`goplugin.Serve` fatal guard — Signal's `path` and Proton's `webmail_base_url` join the fields already marked required. Signal's `path` additionally seeded with the real, editable default `~/.config/Signal`. Three pure helpers exported: `defaultConnectionValues`, `missingRequiredFields`, `missingRequiredFieldsMessage`.
- **Task 2:** `ConnectionForm.svelte` marks required fields with the DOM `required` attribute on both branches (plain `Input` and `SecretField`, the latter now forwarding it to its underlying input). `AddSourceModal.svelte`'s plugin-type selection seeds `connectionValues` from `defaultConnectionValues()`; its `handleConnectNext` and `saveAnyway` both call `missingRequiredFields()` before issuing `describePlugin`/`putConfig`. `EditSourceModal.svelte`'s `submitConnection` gets the identical guard before `putConfig`. All three sites render the same `missingRequiredFieldsMessage()` sentence and issue no request when a required field is blank.
- **Task 3:** `kernel/pluginhost/host.go`'s `launch` wires a new bounded, mutex-guarded `stderrTail` type into `goplugin.ClientConfig.Stderr` (previously unset, defaulting to `io.Discard`). On a connect failure, after `client.Kill()` returns, the tail's last non-empty line is appended to the wrapped error; a silent exit leaves the error byte-identical to before. Since `Discover`, `Reconcile`, and `DescribePluginType`'s trial launch all share `launch`, this covers boot-time and hot-apply launches identically to UI trial launches.

## Task Commits

1. **Task 1: The field table tells the truth about what each plugin cannot start without** — `3e51eb2` (feat) — completed in a prior session; verified present at continuation start, not redone
2. **Task 2: No blank required field ever reaches a plugin process** — `a274ebb` (feat)
3. **Task 3: A plugin that dies before the handshake gets to say why** — `e2ac411` (feat)

**Plan metadata:** committed as part of this SUMMARY's own commit (see below)

## Files Created/Modified

- `web/src/lib/plugin-fields.ts` — required flags re-derived per plugin; `defaultValue` descriptor property; `defaultConnectionValues`/`missingRequiredFields`/`missingRequiredFieldsMessage` helpers (Task 1)
- `web/src/lib/plugin-fields.test.ts` — table-truth tests per plugin binary, default-value-count test, helper behavior tests (Task 1)
- `web/src/lib/components/ConnectionForm.svelte` — DOM `required` attribute on the plain-field `Input`; forwards `required` into `SecretField` (Task 2)
- `web/src/lib/components/SecretField.svelte` — forwards its already-accepted `required` prop to the underlying `Input` (Task 2)
- `web/src/lib/components/AddSourceModal.svelte` — plugin-type selection seeds from `defaultConnectionValues`; `handleConnectNext`/`saveAnyway` guard on `missingRequiredFields` before `describePlugin`/`putConfig` (Task 2)
- `web/src/lib/components/EditSourceModal.svelte` — `submitConnection` guards on `missingRequiredFields` before `putConfig` (Task 2)
- `web/src/lib/components/add-source.test.ts` — source-order assertions proving the guard runs ahead of every request it exists to stop, across all three sites, plus the DOM-attribute and default-seeding assertions (Task 2)
- `kernel/pluginhost/host.go` — new `stderrTail` type (bounded, mutex-guarded, front-discard) wired into `launch`'s `goplugin.ClientConfig.Stderr`; connect-failure branch appends the captured last line after `client.Kill()` (Task 3)
- `kernel/pluginhost/stderr_test.go` (new) — fake-fatal and fake-silent shell-script plugin fixtures; pre-handshake-fatal and silent-exit `DescribePluginType` tests; direct bounds/last-line/concurrency tests for `stderrTail`

## Decisions Made

- Required flags derived by reading all four `plugins/*/main.go` pre-Serve guards, not patched against the two fields the UAT report happened to name — closes the whole class (paperless/silverbullet's existing required fields were already correct and left untouched; Proton and Signal were the two that had drifted).
- Only Signal's `path` gets a seeded default (`~/.config/Signal`, genuinely correct on a standard install); Proton's `webmail_base_url` stays required with its placeholder only, since its correct value is per-user account-index-specific and a wrong default would silently break deep links.
- Enforcement lives in three pure helper functions called from each submit handler's own body, not inside `ConnectionForm` — the shared component is mounted by three independent consumers, each issuing its own request from its own handler, so a guard living only in the component could never stop any of them.
- `EditSourceModal`'s Edit connection… flow gets the identical guard even though the UAT report reached the defect through the add flow — blanking a required field there would persist an instance that fails every subsequent hot-apply reconcile.
- `stderrTail`'s cap is 4 KiB: large enough for a fatal line or the tail of a stack trace, small enough to be irrelevant to kernel memory even if held for a plugin's whole lifetime.
- The buffer discards from the FRONT once the cap is exceeded, never truncates an incoming write — the explanatory line is the LAST thing a dying plugin writes, so front-discard is what keeps it.
- The stderr read happens strictly after `client.Kill()` returns, since `Kill` waits on go-plugin's own client wait group, which the stderr-draining goroutine belongs to — reading any earlier would race that goroutine.
- Server-side required-field validation in `DescribePluginHandler` was deliberately NOT added (planning choice 7): no connection-field schema exists anywhere on the wire (07-RESEARCH.md's load-bearing finding), and the stderr capture already converts a blank field reaching the kernel into a self-explanatory failure rather than a silent one. A real wire schema for this belongs to a future plugin-contract phase, not this gap-closure plan.

## RED/GREEN Evidence

### Task 1 — table-truth tests (reconstructed at continuation start, since Task 1 ran in a prior session that did not persist this output)

RED, running the Task 1 test additions against the unmodified (pre-3e51eb2) `plugin-fields.ts`:

```
FAIL  src/lib/plugin-fields.test.ts > table truth: required flags match each plugin binary's own pre-Serve fatal guards > proton requires exactly base_url, username, token and webmail_base_url (plugins/proton/main.go fatals on all four; ca_cert has no guard)
AssertionError: expected [ 'base_url', 'username', 'token' ] to deeply equal [ 'base_url', 'username', …(2) ]
- Expected:  [ "base_url", "username", "token", "webmail_base_url" ]
+ Received:  [ "base_url", "username", "token" ]

FAIL  src/lib/plugin-fields.test.ts > table truth: required flags match each plugin binary's own pre-Serve fatal guards > signal requires exactly path (plugins/signal/main.go fatals on cfg.Path == "")
AssertionError: expected [] to deeply equal [ 'path' ]
- Expected:  [ "path" ]
+ Received:  []

Tests  2 failed | 2 passed | 22 skipped (26)
```

(13 tests total failed against the unmodified table+missing helpers combination — the two table-truth failures above plus 11 further failures from `defaultConnectionValues`/`missingRequiredFields`/`missingRequiredFieldsMessage` not yet existing.)

GREEN, against the committed `3e51eb2` state: `npm test -- src/lib/plugin-fields.test.ts` — 26/26 passing.

### Task 2 — source-scan assertions

RED, running the new `add-source.test.ts` assertions against the pre-Task-2 components:

```
FAIL  G-07-5 guard: a blank required field never reaches describePlugin or putConfig > saveAnyway calls missingRequiredFields( strictly before putConfig(
FAIL  G-07-5 guard: a blank required field never reaches describePlugin or putConfig > plugin-type selection builds connectionValues from defaultConnectionValues(
FAIL  G-07-5 guard: ConnectionForm.svelte marks required fields with the DOM required attribute, not only an asterisk > the non-secret Input carries a required attribute bound to the field descriptor
FAIL  G-07-5 guard: EditSourceModal.svelte's Edit connection… guards submitConnection the same way > submitConnection calls missingRequiredFields( strictly before putConfig(

Tests  5 failed | 25 passed (30)
```

(The `handleConnectNext` ordering assertion's `guardIndex >= 0` sub-check also failed pre-change but was folded into the same reported failure set; the `SecretField` required-prop-forwarding assertion passed even pre-change since `ConnectionForm` already passed `required` into `SecretField` before this plan — only the DOM-attribute forwarding inside `SecretField` itself, and the `Input` attribute in `ConnectionForm`'s own else-branch, were new.)

GREEN, against the committed `a274ebb` state: `npm test -- src/lib/components/add-source.test.ts` — 30/30 passing.

### Task 3 — pre-handshake stderr capture, RED and GREEN error text side by side

RED (unwired `ClientConfig.Stderr`):

```
TestDescribePluginType_PreHandshakeFatalSurfacesPluginStderr: FAIL
error: pluginhost: trial-launch for describe: connect to plugin subprocess: Unrecognized remote plugin message:

Failed to read any lines from plugin's stdout
This usually means
  the plugin was not compiled for this architecture,
  the plugin is missing dynamic-link libraries necessary to run,
  the plugin is not executable by this process due to file permissions, or
  the plugin failed to negotiate the initial go-plugin protocol handshake

Additional notes about plugin:
  Path: /tmp/.../topos-plugin-fake-fatal
  Mode: -rwxr-xr-x
  Owner: 1000 [darren] (current: 1000 [darren])
  Group: 1000 [darren] (current: 1000 [darren])
```

None of the four listed causes were true — the fake plugin exists, has the right architecture, has execute permission, and is owned by the running user. The actual cause (a config field the plugin itself required) is invisible in this text, exactly matching 07-UAT.md G-07-5's report.

GREEN (wired):

```
TestDescribePluginType_PreHandshakeFatalSurfacesPluginStderr: PASS
error: pluginhost: trial-launch for describe: connect to plugin subprocess: Unrecognized remote plugin message: ... (plugin stderr: topos-plugin-fake-fatal: config.Path is required, got empty string)
```

The plugin's own stderr line now appears in the error the UI renders, alongside the unchanged wrap prefixes.

`TestDescribePluginType_SilentPreHandshakeExitLeavesErrorUnchanged` (a plugin exiting non-zero writing nothing to stderr) passes both before and after — proving the addition is strictly conditional and never appends an empty parenthetical.

## Verification Results

- `CGO_ENABLED=0 go build ./...` — exit 0
- `go vet ./kernel/...` — exit 0
- `go test ./kernel/pluginhost/... -count=1 -race -v` — all tests pass, including all 6 `stderr_test.go` tests
- `go test ./kernel/... -count=1 -race` — every package `ok` (config, correlate, httpapi, index, pluginhost, supervisor, syncer)
- `cd web && npm test` — 32 files, 528/528 tests passing
- `npm run check` — 0 errors, 9 pre-existing warnings unrelated to this plan's files
- `npm run build` — exit 0
- `git diff --stat plugins/ proto/ go.mod go.sum web/package.json web/package-lock.json` — no output (no plugin source, proto, or dependency changed)
- `git diff kernel/pluginhost/describe_test.go` — no output (pre-existing tests untouched)
- `grep -c 'cmd.Stderr' kernel/pluginhost/host.go` — 0
- Ordering checks (all confirmed via `sed`-extracted function bodies): `missingRequiredFields(` precedes `describePlugin(` in `handleConnectNext`; precedes `putConfig(` in `saveAnyway`; precedes `putConfig(` in `EditSourceModal`'s `submitConnection`; the stderr `lastLine()` read follows `client.Kill()` in `launch`'s failure branch
- `git diff --stat` across the whole plan (`7447f64..HEAD`) touches exactly the 9 files named in `07-13-PLAN.md`'s `files_modified` — no scope drift
- No file under `plugins/` was modified by this plan

## Deviations from Plan

None — plan executed exactly as written across all three tasks. Task 1's RED output was reconstructed at the start of this continuation (against a temporary checkout of the pre-Task-1 `plugin-fields.ts` with the committed test additions) since the prior session's own RED run was not persisted anywhere this continuation could read; this is a documentation reconstruction, not a re-execution of Task 1's work, and Task 1's commit (`3e51eb2`) was left untouched.

## Issues Encountered

One self-inflicted acceptance-criterion trip during Task 3: the doc comment explaining why `cmd.Stderr` is left unset on the `exec.Cmd` originally used the literal substring `cmd.Stderr`, which collided with the acceptance criterion's own `grep -c 'cmd.Stderr' kernel/pluginhost/host.go` check (intended to catch an actual `cmd.Stderr = ...` assignment, not a comment mentioning the identifier). Reworded the comment to say "the exec.Cmd's own Stderr field" instead — no functional change, `grep -c` now correctly reports 0.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- 07-UAT.md `G-07-5` is closed for all four plugin types: the required-field mismatch that let a blank Signal path (or Proton webmail_base_url, or either plugin's other mandatory fields) reach a doomed subprocess launch is now caught client-side before any request, and a pre-handshake fatal from any plugin now explains itself in the kernel's own error text.
- **Outstanding: the plan's `<human-check>` block (live `make dev` verification of the two-step "New Signal…"/"New Proton…" flows, the blank-field message with no network request, and the stderr-carrying error on a forced pre-handshake failure) was NOT performed this run** — no browser or live kernel was driven in this autonomous continuation. All automated verification (unit tests, source-scan assertions, `-race` kernel suite, build/check) passes; the end-to-end human confirmation remains for a live verification pass before this plan is considered fully closed.
- 07-14 (the next incomplete plan) is unaffected by anything in this plan's scope and can proceed independently.

---
*Phase: 07-webspace-builder-ui*
*Completed: 2026-08-09*
