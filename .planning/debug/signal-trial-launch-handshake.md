---
status: diagnosed
trigger: "G-07-5: Adding a second Signal source via the UI's two-step \"New signal…\" flow fails at the Connect step — the trial-launch for describe cannot start the signal plugin subprocess."
created: 2026-08-09T12:01:00Z
updated: 2026-08-09T12:10:00Z
---

## Current Focus
<!-- OVERWRITE on each update - reflects NOW -->

bug_class: Bohrbug (deterministic — reproduces 100% of the time on the exact input; no timing, no concurrency)

hypothesis: The Connect step submits `path` as absent/empty because `web/src/lib/plugin-fields.ts` declares Signal's Local Path field `required: false` with a placeholder that reads as a default. `plugins/signal/main.go` treats an empty `path` as a fatal startup error, printing one line to stderr and `os.Exit(1)` BEFORE `goplugin.Serve` — so go-plugin never reads a handshake line from stdout and reports its generic "Failed to read any lines from plugin's stdout" diagnostic, which names none of the real reason.
test: Reproduced end to end against an ephemeral kernel (XDG_CONFIG_HOME override, port 7799, temp index) by POSTing /api/config/describe-plugin three ways: (A) no `path`, (B) real `path`, (C) bogus-but-non-empty `path`.
expecting: A → the reported 502 error verbatim; B and C → HTTP 200 with match_vocabulary. If C had ALSO failed, "user typed a wrong path" would remain live; C passing proves the trigger is emptiness specifically.
next_action: Diagnosis complete (goal: find_root_cause_only) — return ROOT CAUSE FOUND. No fix applied.

reasoning_checkpoint:
  hypothesis: "The trial-launch is made with config.Source.Path == \"\"; plugins/signal/main.go's pre-Serve guard fatals on that, exiting before the go-plugin handshake, and the kernel discards the child's stderr so the real reason is invisible."
  confirming_evidence:
    - "Direct run: WEBSPACES_SOURCE_CONFIG with path:\"\" → stderr 'topos-plugin-signal: WEBSPACES_SOURCE_CONFIG: path is empty', exit 1, zero bytes on stdout"
    - "Ephemeral-kernel POST with no path → HTTP 502 whose message is byte-identical to the UAT report, including the Path/Mode/Owner/Group/ELF diagnostic block"
    - "Same POST with path → HTTP 200 {source_type: signal, match_vocabulary: [conversations]}"
    - "Same POST with a bogus non-empty path (/definitely/not/a/real/dir) → HTTP 200 — emptiness, not wrongness, is the trigger"
    - "plugin-fields.ts marks Signal's path required:false with placeholder '~/.config/Signal'; ConnectionForm.svelte renders a placeholder only, never a default value, and emits no HTML required attribute"
  falsification_test: "If a bogus non-empty path had also produced the same 502, the root cause would be 'path resolution/validation', not 'path empty'. It returned 200 — falsified."
  fix_rationale: "N/A — diagnose-only mode. Fix direction addresses the UI/plugin contract mismatch (path is mandatory to the plugin but optional in the field table) and the stderr-discard that hid it, not the go-plugin symptom."
  blind_spots:
    - "Not directly observed: the operator's own keystrokes. Inferred from the byte-identical error plus experiment C (only an EMPTY path can produce this error for this plugin)."
    - "The live kernel's config.toml no longer contains a [sources.signal] block (deleted during later UAT tests), so the original session state could not be inspected directly; the prior block is preserved in config.toml.pre-07-01-fix.bak with path = '/home/darren/.config/Signal'."
  candidate_causes:
    - "config/contract: plugin-fields.ts declares path required:false while plugins/signal/main.go requires it (CONFIRMED)"
    - "code/observability: pluginhost.launch never captures the child's pre-handshake stderr, so the one-line reason is discarded (CONFIRMED — kernel log holds only 'exit status 1')"
    - "environment: trial-launch omits env/D-Bus vars the boot-time launch passes (ELIMINATED — same launch() function, same os.Environ())"
    - "data/state: the already-running first Signal instance holds the SQLCipher DB or a lock (ELIMINATED — a second instance handshakes fine alongside four live ones)"
  and_gate: "yes — two conditions are simultaneously required to produce the REPORTED symptom. Cause 1 alone produces the failure; cause 2 alone produces nothing; both together produce a failure whose stated reasons (arch/libs/permissions) are all false and whose real reason is unobtainable from any surface the operator has. Cause 2 is why this reached UAT as an unexplained blocker rather than a self-evident 'path is required'."

## Symptoms
<!-- Written during gathering, then IMMUTABLE -->

expected: Two-step "New {plugin type}…" Connect step trial-launches via describePlugin and advances to the vocabulary-driven Match step on success
actual: Clicking "Next" fails with "Couldn't verify this connection. pluginhost: trial-launch for describe: connect to plugin subprocess: Unrecognized remote plugin message: Failed to read any lines from plugin's stdout." go-plugin diagnostics report path bin/plugins/topos-plugin-signal, mode -rwxr-xr-x, owner/group 1000 darren, ELF EM_X86_64 matching amd64 — binary, arch and permissions all fine; the subprocess almost certainly exited before writing the handshake line
errors: "pluginhost: trial-launch for describe: connect to plugin subprocess: Unrecognized remote plugin message: Failed to read any lines from plugin's stdout"
reproduction: Test 5 in 07-UAT.md — live kernel via `make dev`, chip-row '+', "New signal…", fill Connect step, click Next
started: Discovered during UAT 2026-08-09. An existing signal instance is already configured and running in the same kernel.

## Eliminated
<!-- APPEND only - prevents re-investigating -->

- hypothesis: "(a) The trial-launch environment omits env vars / D-Bus session address / HOME-derived paths that the boot-time launch passes, so the plugin exits at startup"
  evidence: "kernel/pluginhost/host.go:339 DescribePluginType calls the SAME unexported launch() (line 222) that Discover/Reconcile use, with no environment divergence whatsoever: cmd.Env = append(os.Environ(), \"WEBSPACES_SOURCE_CONFIG=\"+…) at line 245 is the single env construction site for both paths. There is no trial-launch-specific environment. Confirmed empirically: an identical config launched via the same handler succeeds (HTTP 200) — env cannot be the differentiator."
  timestamp: 2026-08-09T12:05:00Z

- hypothesis: "(b) The already-running first signal instance holds the SQLCipher DB open, or a lock, so a second open fails instantly"
  evidence: "Two independent disproofs. (1) Structural: plugins/signal/plugin.go's NewSourcePlugin only stores configDir and os.Stderr — it opens no database; the DB is opened per-RPC and the type's own doc comment states it caches nothing across calls. Nothing DB-related runs before goplugin.Serve. (2) Empirical: ran bin/plugins/topos-plugin-signal directly with path=/home/darren/.config/Signal while FOUR other topos-plugin-signal processes were alive (PIDs 964888, 980566, 2011626, 2015959) — it printed its handshake line '1|2|unix|/tmp/plugin106633194|grpc|' to stdout and served normally. Also HTTP 200 from the ephemeral kernel under the same conditions."
  timestamp: 2026-08-09T12:06:00Z

- hypothesis: "The operator typed a wrong/typo'd Local Path and the plugin rejected it"
  evidence: "POST /api/config/describe-plugin with path='/definitely/not/a/real/dir' returned HTTP 200 with match_vocabulary ['conversations']. main.go's only path guard is `cfg.Path == \"\"`; expandHome does no existence check and Describe never touches the filesystem. A wrong path cannot produce this error — only an EMPTY one can."
  timestamp: 2026-08-09T12:08:00Z

- hypothesis: "Binary/arch/permission problem, as go-plugin's own diagnostic block suggests"
  evidence: "The same binary, same mode, same user, same architecture handshakes successfully when path is non-empty (HTTP 200). go-plugin prints that four-item diagnostic list unconditionally on any silent child exit — every item in it is false here."
  timestamp: 2026-08-09T12:08:00Z

## Evidence
<!-- APPEND only - facts discovered -->

- timestamp: 2026-08-09T12:02:00Z
  checked: kernel/pluginhost/host.go — DescribePluginType (line 339) and launch (line 222)
  found: DescribePluginType calls launch(ctx, pluginsDir, "__trial__", src, logger) and defers p.Kill(). launch marshals exactly eight known source-config keys into WEBSPACES_SOURCE_CONFIG (base_url, token, api_version, ca_cert, username, webmail_base_url, path) and sets cmd.Env = append(os.Environ(), …). The go-plugin client is constructed with Logger: logger.Named("plugin."+name); goplugin.ClientConfig.Stderr is never set.
  implication: Boot-time launch and trial-launch are the SAME code with the same environment — env divergence is structurally impossible. The only thing that varies between a working boot launch and a failing trial launch is the CONTENT of the config.Source passed in. Also: nothing captures the child's stderr.

- timestamp: 2026-08-09T12:03:00Z
  checked: plugins/signal/main.go lines 37-93
  found: main() has four fatal() paths that run BEFORE goplugin.Serve — WEBSPACES_SOURCE_CONFIG unset, JSON parse failure, `cfg.Path == ""`, and expandHome error. fatal() does `fmt.Fprintln(os.Stderr, "topos-plugin-signal:", err); os.Exit(1)`. Nothing is ever written to stdout on these paths.
  implication: A pre-handshake exit writing only to stderr is exactly the condition go-plugin reports as "Failed to read any lines from plugin's stdout". Of the four, only `path is empty` is reachable from a kernel-driven launch (the kernel always sets the var and always marshals valid JSON).

- timestamp: 2026-08-09T12:03:30Z
  checked: plugins/signal/plugin.go NewSourcePlugin
  found: Constructor stores configDir and os.Stderr only — no sql.Open, no file read, no key resolution. The type's doc comment states it caches nothing across calls and re-derives everything per Match.
  implication: There is no startup-time DB open, so no second-instance lock contention is possible at handshake time. Rules out hypothesis (b) structurally.

- timestamp: 2026-08-09T12:04:00Z
  checked: Direct subprocess run — `WEBSPACES_SOURCE_CONFIG='{…,"path":""}' ./bin/plugins/topos-plugin-signal`
  found: stderr `topos-plugin-signal: WEBSPACES_SOURCE_CONFIG: path is empty`; exit code 1; zero bytes on stdout.
  implication: Confirms the fatal path fires on an empty path and produces exactly the silent-stdout condition go-plugin misreports.

- timestamp: 2026-08-09T12:04:30Z
  checked: Direct subprocess run with the handshake cookie — `TOPOS_PLUGIN=topos-source-plugin-v1 WEBSPACES_SOURCE_CONFIG='{"path":"/home/darren/.config/Signal"}' ./bin/plugins/topos-plugin-signal` (and again with '~/.config/Signal')
  found: Both printed the go-plugin handshake line `1|2|unix|/tmp/plugin106633194|grpc|` to stdout and served until the timeout killed them — while four other topos-plugin-signal processes were live.
  implication: Same binary, same permissions, same architecture, concurrent siblings — all fine. `path` content is the only variable that matters.

- timestamp: 2026-08-09T12:05:00Z
  checked: web/src/lib/plugin-fields.ts CONNECTION_FIELDS['topos-plugin-signal']
  found: `{ key: 'path', label: 'Local Path', required: false, secret: false, advanced: false, placeholder: '~/.config/Signal' }` — declared OPTIONAL, with the plugin's mandatory value shown only as a placeholder.
  implication: The UI's field table and the plugin's own startup contract disagree about whether `path` is mandatory. The placeholder renders as greyed text that reads like a pre-filled default, so leaving it untouched is the natural operator action.

- timestamp: 2026-08-09T12:05:30Z
  checked: web/src/lib/components/ConnectionForm.svelte lines 71-82
  found: A non-secret field renders `<Input value={fieldValue(field)} placeholder={field.placeholder} …>` — the `required` descriptor only appends ' *' to the label text; no HTML `required` attribute, no default seeding from `placeholder`, and no client-side check before submit. AddSourceModal.handleConnectNext (line 207) posts `connectionValues` verbatim.
  found (corollary): `required: true` fields are therefore equally unenforced — a blank Base URL/Token on paperless/silverbullet/proton reaches the same pre-Serve fatal.
  implication: An untouched optional field is submitted absent, and even a starred required field is submittable blank. The form cannot stop this class of request from reaching exec.Command.

- timestamp: 2026-08-09T12:06:00Z
  checked: Ephemeral kernel (XDG_CONFIG_HOME override, 127.0.0.1:7799, temp index, plugins dir = repo bin/plugins, zero configured sources) — POST /api/config/describe-plugin, signal, source WITHOUT a path key
  found: HTTP 502 `plugin_describe_failed`, message byte-identical to the UAT report including the full "This usually means…" block and the Path/Mode/Owner/Group/ELF notes.
  implication: EXACT reproduction of G-07-5 from a clean kernel with no Signal instance configured at all — which additionally proves the failure has nothing to do with an existing Signal instance being present.

- timestamp: 2026-08-09T12:06:30Z
  checked: Same ephemeral kernel — same POST but WITH `"path":"~/.config/Signal"`
  found: HTTP 200 `{"source_type":"signal","plugin_display_name":"Signal","match_vocabulary":["conversations"]}`.
  implication: One field flips the outcome. With it, the Match step would have rendered its Conversations input and the flow would have completed.

- timestamp: 2026-08-09T12:07:00Z
  checked: Same ephemeral kernel — same POST with `"path":"/definitely/not/a/real/dir"`
  found: HTTP 200, same body as above.
  implication: Decisive discriminator. A non-existent path succeeds; only an EMPTY path fails. Therefore the failing request necessarily carried an empty/absent path — i.e. the Local Path field was left blank. Also shows Describe validates nothing about the path, so a typo'd path passes Connect and only fails later at sync time.

- timestamp: 2026-08-09T12:07:30Z
  checked: Ephemeral kernel log across all failing trial launches
  found: Only `[ERROR] topos.plugin.__trial__: plugin process exited: plugin=…/topos-plugin-signal id=3168895 error="exit status 1"` and `[WARN] plugin failed to exit gracefully`. The string "path is empty" appears NOWHERE in the kernel log.
  implication: The child's stderr is discarded entirely. go-plugin only starts draining plugin stderr into the logger after a successful handshake; a pre-handshake fatal's single explanatory line is lost when the client kills the process. The operator has no surface anywhere — UI, kernel log, or hclog — that carries the real reason. This is what converted a one-line input error into an unexplained blocker.

- timestamp: 2026-08-09T12:08:30Z
  checked: Same defect class across the other three plugins — plugins/{paperless,silverbullet,proton}/main.go pre-Serve guards vs plugin-fields.ts
  found: All three fatal() before Serve on missing config. Proton fatals on empty `webmail_base_url` (main.go:56) while plugin-fields.ts declares webmail_base_url `required: false` — the SAME optional-in-UI/mandatory-in-plugin mismatch as Signal's `path`. Verified empirically: POST describe-plugin for proton with base_url+username+token but no webmail_base_url → HTTP 502 with the identical error shape; adding `"webmail_base_url":"https://mail.proton.me/u/0"` → HTTP 200 `{source_type: proton, match_vocabulary: [folders]}`.
  implication: G-07-5 is NOT Signal-specific. "New Proton…" is broken in exactly the same way for anyone who leaves Webmail Base URL blank, and blank starred fields on paperless/silverbullet hit the same wall. Any fix scoped to Signal alone leaves the class open.

- timestamp: 2026-08-09T12:09:00Z
  checked: Live config ~/.config/topos/config.toml and its backups
  found: The live config currently has NO [sources.signal] block (docs/proton/silverbullet only) — it was deleted during a later UAT test. config.toml.pre-07-01-fix.bak preserves the working block: `path = "/home/darren/.config/Signal"`.
  implication: The pre-existing working instance had an explicit path, set by hand-editing the TOML — never through this UI flow. The UI path has therefore never once produced a working Signal instance, consistent with this being the first attempt through it.

## Resolution
<!-- OVERWRITE as understanding evolves -->

root_cause: |
  Two confirmed contributing causes (AND-gate fired):

  (1) UI/plugin contract mismatch — `web/src/lib/plugin-fields.ts` declares Signal's `path`
      field `required: false` and surfaces the mandatory value `~/.config/Signal` only as a
      placeholder. `ConnectionForm.svelte` renders a placeholder as display-only text and emits
      no HTML `required` attribute, so the Connect step submits `path` absent. The kernel passes
      it through untouched (`DescribePluginHandler` validates only that the plugin binary is a
      discovered one) into `pluginhost.launch`, which marshals `"path":""` into
      WEBSPACES_SOURCE_CONFIG. `plugins/signal/main.go:47` treats an empty path as fatal, prints
      one line to stderr and `os.Exit(1)` BEFORE `goplugin.Serve` — so the child never writes a
      handshake line to stdout, and go-plugin reports its generic "Failed to read any lines from
      plugin's stdout" with a four-item diagnostic list every item of which is false here.

  (2) Pre-handshake stderr is discarded — `pluginhost.launch` never sets
      `goplugin.ClientConfig.Stderr`, and go-plugin only begins draining plugin stderr into the
      logger AFTER a successful handshake. The child's one explanatory line
      ("topos-plugin-signal: WEBSPACES_SOURCE_CONFIG: path is empty") reaches no surface at all:
      the kernel log records only `exit status 1`. Cause (1) creates the failure; cause (2) makes
      it undiagnosable, which is why it reached UAT as an unexplained blocker rather than an
      obvious missing-field error.

  Scope: not Signal-specific. Proton reproduces identically (`webmail_base_url` is `required:
  false` in the field table but fatal-guarded in plugins/proton/main.go), and because
  ConnectionForm never enforces `required: true` either, a blank Base URL or Token on
  paperless/silverbullet hits the same wall.

fix: [not applied — diagnose-only mode (goal: find_root_cause_only)]

verification: |
  Root cause verified by controlled experiment against an ephemeral kernel (port 7799, temp
  XDG_CONFIG_HOME and index, zero configured sources), POST /api/config/describe-plugin:
    A. signal, no `path`                        → HTTP 502, error byte-identical to the UAT report
    B. signal, path "~/.config/Signal"          → HTTP 200, match_vocabulary ["conversations"]
    C. signal, path "/definitely/not/a/real/dir"→ HTTP 200 (proves emptiness is the trigger, not wrongness)
    D. proton, no `webmail_base_url`            → HTTP 502, identical error shape
    E. proton, with webmail_base_url            → HTTP 200, match_vocabulary ["folders"]
  Plus direct subprocess runs confirming the empty-path stderr line + exit 1 with an empty stdout,
  and a successful handshake line alongside four concurrently running signal subprocesses.
  Ephemeral kernel shut down afterwards; the live kernel on 7777 and ~/.config/topos were never
  written to.

files_changed: []
