---
status: diagnosed
trigger: "G-07-4: With zero webspaces configured, the root page shows the service-unreachable error instead of the \"No webspaces yet\" empty state with a Create webspace CTA."
created: 2026-08-09T00:00:00Z
updated: 2026-08-09T00:05:00Z
---

## Current Focus

hypothesis: CONFIRMED — GET /api/config marshals a nil `webspaces` map to JSON `null` (not `{}`) when config.toml has zero [webspaces.*] blocks; the SPA's root route does `Object.keys(res.config.webspaces)` inside the same try block as the fetch itself, so this throws a TypeError that's caught by the same blanket catch used for a genuinely unreachable kernel — misattributing a successful-but-empty response as "the topos service didn't respond".
test: live ephemeral kernel (XDG_CONFIG_HOME override, port 7799, zero [webspaces.*] blocks) — curl GET /api/config directly; also verified Object.keys(null) throws in node
expecting: response body's config.webspaces field is JSON null, and Object.keys(null) throws TypeError
next_action: root cause confirmed — proceed to return_diagnosis (goal is find_root_cause_only)

## Symptoms

expected: With no webspaces configured, `/` renders 'No webspaces yet' with a working Create webspace CTA and does not navigate; no redirect loop, no blank page
actual: After deleting all webspaces in config.toml and restarting make dev, the root page shows: "Couldn't load this webspace — the topos service didn't respond. Check that it's running, then retry." with no button or link to create one. When webspaces exist, it correctly redirects to the last used
errors: None beyond the on-page copy above
reproduction: Test 4 in UAT — remove every [webspaces.*] block from config, restart make dev, load /
started: Discovered during UAT 2026-08-09

## Eliminated

- hypothesis: kernel config validation rejects a config with zero [webspaces.*] blocks, so the kernel fails to start entirely (matching the "service didn't respond" wording literally)
  evidence: kernel/config/config.go validateWebspaces() (line 310-345) iterates `for _, name := range names` where names is built from `cfg.Webspaces` keys — with zero webspaces this loop body never executes and the function returns nil (no error). Confirmed live: an ephemeral kernel launched against a config.toml with zero [webspaces.*] blocks starts and serves successfully (see Evidence below).
  timestamp: 2026-08-09T00:00:00Z

- hypothesis: a remembered-but-now-deleted webspace name in localStorage (readLastWebspace()) causes resolveRedirectTarget to misbehave or navigate somewhere invalid
  evidence: resolveRedirectTarget (web/src/lib/last-webspace.ts:52-58) is pure: `if (remembered !== null && webspaces.includes(remembered)) return remembered; return webspaces[0] ?? null;` — with an empty webspaces array, `webspaces.includes(remembered)` is always false regardless of what's remembered, so it falls through to `webspaces[0] ?? null` = null. This function is never reached in practice for the zero-webspace case anyway, because the crash (see confirmed hypothesis) happens one line earlier at `Object.keys(res.config.webspaces)`, before resolveRedirectTarget is ever called.
  timestamp: 2026-08-09T00:00:00Z

## Evidence

- timestamp: 2026-08-09T00:00:00Z
  checked: web/src/routes/+page.svelte onMount (lines 27-44) — the root redirect route
  found: The entire redirect-decision logic is wrapped in one try/catch: `try { const res = await getConfig(); configResponse = res; const webspaceNames = Object.keys(res.config.webspaces); const target = resolveRedirectTarget(webspaceNames, readLastWebspace()); if (target !== null) { await goto(...); return; } phase = 'empty'; } catch { phase = 'error'; }`. The catch block has no discrimination between "fetch/network failure" and "any other JS exception thrown while processing an already-successful response" — both produce the identical 'error' phase, which renders the "Couldn't load this webspace — the topos service didn't respond" copy (lines 66-73).
  implication: any runtime exception anywhere in the try block — not just an actual unreachable-kernel fetch failure — reaches the same generic error copy. This is the structural bug that lets a real defect elsewhere present itself with misleading "service unreachable" wording.

- timestamp: 2026-08-09T00:00:00Z
  checked: kernel/config/types.go line 19 — `Webspaces map[string]Webspace \`toml:"webspaces" json:"webspaces"\`` (no `omitempty`); kernel/config/config.go applyDefaults() (lines 150-163) — only sets scalar defaults (Server.Listen, Index.Path, Plugins.Dir, Sync.Interval), never allocates an empty map for Webspaces or Sources when the TOML source declares no [webspaces.*]/[sources.*] blocks
  found: When config.toml carries zero [webspaces.*] blocks, go-toml/v2 leaves cfg.Webspaces as Go's nil map zero value (the key is entirely absent from the source, so the decoder never allocates), and nothing in Load/LoadRaw/applyDefaults initializes it. Go's encoding/json marshals a nil map field without `omitempty` as JSON `null`, not `{}`.
  implication: GET /api/config's response body contains `"webspaces": null` (not `"webspaces": {}`) whenever zero webspaces are configured — this is what the frontend actually receives.

- timestamp: 2026-08-09T00:00:00Z
  checked: live repro — ephemeral kernel instance (bin/topos serve, XDG_CONFIG_HOME override pointing at a scratch dir with a config.toml carrying only [server]/[index]/[plugins]/[sync] blocks — zero [webspaces.*] and zero [sources.*] — listening on 127.0.0.1:7799, a port distinct from the user's live 127.0.0.1:7777 instance). `curl -s http://127.0.0.1:7799/api/config`
  found: Response body: `{"schema_version":1,"hash":"...","config":{...,"sources":null,"webspaces":null},"env_vars":{},"unknown_keys":[]}` — confirms `webspaces` is JSON `null` on the wire, exactly as predicted. Kernel started and served successfully (no crash, no startup failure) — the "service didn't respond" framing is not literally true; the kernel is up and answering 200 OK. Kernel process was cleanly shut down afterward (verified via curl connection-refused on the ephemeral port and `ps aux` showing no lingering bin/topos process) — no mutation made to the user's live 127.0.0.1:7777 instance or ~/.config/topos/config.toml.
  implication: confirms the config Load/Save path itself is fine with zero webspaces (no kernel-side validation error, no startup crash) — the defect is entirely client-side, in how the SPA's root route processes an otherwise-correct, successful response.

- timestamp: 2026-08-09T00:00:00Z
  checked: `Object.keys(null)` behavior (Node.js, matches all browser JS engines per spec — ToObject abstract operation throws on null/undefined)
  found: `node -e "Object.keys({webspaces: null}.webspaces)"` throws `TypeError: Cannot convert undefined or null to object`
  implication: web/src/routes/+page.svelte line 31 (`const webspaceNames = Object.keys(res.config.webspaces);`) throws this exact TypeError when res.config.webspaces is null (the zero-webspace case) — the throw is caught by the surrounding try/catch (line 41-43: `catch { phase = 'error'; }`), landing on phase 'error' and rendering the "service didn't respond" copy. Mechanism fully confirmed end-to-end: kernel returns webspaces:null on the wire (evidence above) -> Object.keys(null) throws -> caught by the generic catch -> phase='error' -> wrong copy rendered, CTA never reached.

## Resolution

root_cause: "web/src/routes/+page.svelte's onMount handler calls `Object.keys(res.config.webspaces)` unguarded, inside the same try block used to catch actual kernel-unreachable fetch failures. When config.toml has zero [webspaces.*] blocks, the Go kernel's GET /api/config response serializes the nil `Webspaces` map (kernel/config/types.go's `Webspaces map[string]Webspace` field, no omitempty, never defaulted to an empty map in kernel/config/config.go's applyDefaults) as JSON `null` rather than `{}`. `Object.keys(null)` throws a TypeError, which the route's blanket `catch { phase = 'error' }` cannot distinguish from a real network/fetch failure, so it renders the generic 'Couldn't load this webspace — the topos service didn't respond' copy instead of ever reaching the 'No webspaces yet' empty-state branch. The kernel itself starts and responds correctly with 200 OK in this scenario — the 'service didn't respond' framing is factually wrong, confirming this is purely a client-side defect, not a kernel-availability one."
fix: "(diagnosis only — not applied). Two independent, non-exclusive fix points: (1) kernel-side — have applyDefaults (or Load/LoadRaw) initialize cfg.Webspaces (and likely cfg.Sources, same nil-map shape) to an empty map when absent, so GET /api/config always serializes {} rather than null for an empty map field, matching unknownKeysOrEmpty's existing null-to-[] normalization convention already used elsewhere in kernel/httpapi/config.go (see unknownKeysOrEmpty, config.go line 105-113) — the more root-cause-aligned fix, and consistent with the codebase's own established pattern of never emitting null over an API boundary for a collection field. (2) frontend-side (defense in depth, recommended regardless of (1)) — web/src/routes/+page.svelte should defensively read `res.config.webspaces ?? {}` before Object.keys, and/or the onMount handler should distinguish a getConfig() fetch failure (real error phase) from any exception thrown while processing an already-successful response (should not silently collapse to the same generic 'service unreachable' copy — a bug in this branch's own downstream code should not lie about why it failed)."
verification: not applicable — find_root_cause_only mode, no fix applied
files_changed: []
