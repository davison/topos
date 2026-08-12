# Pitfalls Research

**Domain:** Adding external-plugin loading, a filesystem source, an out-of-repo Google Drive plugin, per-item include/exclude marks, and PWA installability to an existing Go-kernel + go-plugin(gRPC-subprocess) local-first app
**Researched:** 2026-08-12
**Confidence:** HIGH for pitfalls grounded directly in this repo's code/contract (cited by file); MEDIUM-HIGH for Google Drive OAuth/export specifics (current official docs + multiple corroborating sources); MEDIUM for fsnotify/network-filesystem behavior (well-documented upstream limitation, exact behavior varies by mount type/OS)

This research is scoped entirely to **integration** pitfalls: mistakes specific to bolting these five features onto *this* system's existing architecture (`kernel/pluginhost`, `kernel/index`, `kernel/correlate`, the published `topos.v1`/`topos.v2` contract, the SvelteKit static SPA embedded via `go:embed`), not generic advice about plugin systems, OAuth, or PWAs in the abstract.

## Critical Pitfalls

### Pitfall 1: Trust marking with no integrity pinning is security theater (TOCTOU)

**What goes wrong:**
`docs/plugin-contract.md` already states the honest baseline: "a plugin is a regular native binary launched as a subprocess with the full local OS access of the user who runs the kernel — `hashicorp/go-plugin` is a transport, not a sandbox." Today every plugin is implicitly trusted because it was built from this repo and lives in the one `[plugins] dir` (`kernel/pluginhost/host.go`'s `launch()` does `filepath.Join(pluginsDir, src.Plugin)` and only checks `os.Stat` for existence). If "trusted vs untrusted" is implemented as a boolean recorded once (e.g., "was this binary added via the in-repo catalog vs a user-browsed path") with no re-verification at every subsequent launch, the label is cosmetic: the file at that path can be swapped after the trust decision was made — by a careless `go build` overwriting the binary in place, a package manager update, or, in the worst case, something malicious with local write access — and the kernel will launch whatever is at that path under the old, stale "trusted" badge with no warning.

**Why it happens:**
"Trusted" reads as a property of *where the binary came from*, which is tempting to implement as a one-time provenance check (in-repo build vs external path) rather than a property of *what bytes are actually being executed right now*. The former is cheap to implement and satisfies the literal requirement ("mark untrusted with a warning when adding"); the latter requires storing and re-checking a content hash on every launch, which is easy to skip since v1.1.0 explicitly defers "distribution, dev guide, and certification."

**How to avoid:**
- Record a content hash (e.g. SHA-256) of the binary at the moment it is marked trusted or added, store it alongside the source's config entry, and re-verify it against the on-disk file at every `launch()` call (Discover, Reconcile, and the trial-launch path in `DescribePluginType` all funnel through this one function today — one recheck point covers all three callers for free).
- On mismatch, fail loud and by name (matching this codebase's existing discipline for config-load failures) rather than silently launching the changed binary — never auto-re-trust on a hash change.
- Be explicit in the UI copy that "trusted" means "built from `davison/topos`," not "verified safe" or "sandboxed" — the existing plugin-contract doc's honesty about "not a sandbox" should carry through to the UI warning text, not get softened into implied security the mechanism doesn't provide.

**Warning signs:**
- The trust flag lives only in config (a boolean/enum) with no accompanying hash field.
- The "mark as trusted" UI flow doesn't re-run any check at the next kernel restart or hot-apply reconcile.
- The warning copy for untrusted plugins reads as a one-time speed bump rather than a standing, re-shown fact about the source instance.

**Phase to address:**
The external-plugin-loading phase — this is the phase's core deliverable, not a follow-on hardening pass.

---

### Pitfall 2: The existing sync-replace strategy silently wipes manual include/exclude marks

**What goes wrong:**
`kernel/index/store.go`'s `ReplaceWebspaceSourceItems` is the exact function every source's sync path calls to commit a `Match` result: inside one transaction it upserts `items` rows, then does `DELETE FROM webspace_items WHERE webspace_name = ? AND item_id IN (SELECT id FROM items WHERE source = ?)` followed by a fresh `INSERT` per currently-matched item. This delete-then-reinsert is deliberate and correct for its current job (a `webspace_items` row is a pure derived fact: "this item currently matches this webspace"). If a per-item include/exclude mark is implemented as a column on `webspace_items` (the natural-looking place, since the mark is scoped to one webspace) or is otherwise keyed to a row that this statement deletes, **every mark is destroyed on the very next sync of that source**, with no error, no test failure, and no visible symptom until a user notices their exclusions "un-excluded themselves."

**Why it happens:**
`webspace_items` looks like the right table for a webspace-scoped, item-scoped fact — that's exactly what it is for match membership. But the milestone's requirement explicitly calls include/exclude "the kernel's first user-owned data beyond config" (PROJECT.md) — a signal that it needs its own persistence lifecycle, independent of the derived, resync-driven `webspace_items` table, not a bolt-on column on a table this codebase already treats as fully disposable every sync.

**How to avoid:**
- Store marks in a dedicated table keyed on `(webspace_name, item_id)` (or `(webspace_name, source, source_id)` if marks must survive an item being fully re-synced under an unchanged stable id — see Pitfall 3) that no sync path ever deletes wholesale.
- Join marks into the read path (`StreamItems`, `Search`) rather than writing them into a table `ReplaceWebspaceSourceItems` owns.
- Add a regression test that runs two consecutive syncs of the same source with a mark applied between them and asserts the mark survives — this is the one test this codebase doesn't have today and is exactly the shape of bug the existing delete-then-reinsert design invites.

**Warning signs:**
- Any schema change that adds an `included`/`excluded` column to `webspace_items` or `items` rather than a new table.
- No test exercises "mark, then resync, then read" as a sequence.

**Phase to address:**
The include/exclude phase — should be the very first design question in that phase's plan, before any schema work starts.

---

### Pitfall 3: Manually including an item Match() never returned is architecturally impossible today

**What goes wrong:**
The plugin contract's only way for an item to enter the index is `Match` — the kernel never asks a plugin "what else do you have," only "what matches these fields" (`docs/plugin-contract.md`, "Match" section: "Called only at sync time"). There is no `List`/`Browse` RPC. This means a user cannot "manually include" an item that no configured match rule ever surfaced, because the kernel has never seen it, has no title/preview/deep-link for it, and has no `source_id` to attach a mark to — the feature as commonly imagined ("pull in that one email even though it's not in a matched folder") requires either a new plugin RPC (a contract change touching every existing plugin) or a narrower definition where "include" only ever *reverses a prior exclude* on an item that did match at some point. If this ambiguity isn't resolved before implementation starts, a plan can accidentally scope "include" as symmetric with "exclude" and only discover the missing-RPC problem mid-phase.

**Why it happens:**
"Include/exclude" reads as a symmetric pair in the requirement text ("mark individual stream entries for inclusion/exclusion"), but the underlying data availability is asymmetric: exclude only needs to suppress an item already in the index; include (of something never matched) needs a browse capability the contract doesn't have.

**How to avoid:**
- Resolve scope explicitly at requirements/design time: either (a) "include" is restricted to un-doing a previous "exclude" (symmetric only within items the source already surfaced at least once), or (b) commit to a contract change (a new bounded `List`/`Browse` RPC, itself a real design effort touching the SDK, every in-repo plugin, and the published contract doc) to support pulling in genuinely unmatched items.
- If (a) is chosen (the lower-risk option given the milestone is already carrying four other features), state this limitation plainly in the UI copy so users don't file it as a bug when a truly-unmatched item can't be manually added.

**Warning signs:**
- A plan or spec for this phase describes "include" without naming which RPC supplies the item's metadata for something never matched.
- UAT criteria assume a user can search/browse "everything in a source" from the include/exclude UI — that browse surface doesn't exist anywhere in the current API.

**Phase to address:**
The include/exclude phase's spec/discuss step — this is a scope decision, not an implementation detail, and belongs before `/gsd-plan-phase`, not discovered during it.

---

### Pitfall 4: Manual marks orphaned when an item's stable id changes on re-sync

**What goes wrong:**
The stable id is `"{source}:{source_id}"` (`docs/api.md`, "The stable-ID scheme"), and `source_id` is entirely plugin-defined ("stable within your plugin" per `docs/plugin-contract.md`). For the new filesystem plugin in particular, the obvious `source_id` choice is a file path — but a rename, a move to a different watched subfolder, or a re-mount at a different point can all change that path even though the user experiences it as "the same file." A Google Drive plugin has a more stable option (Drive's own file `id`, not path), but a naive implementation keying on the file's *name* instead would hit the same problem on a Drive-side rename. Any of these changes silently orphans a mark: the exclude/include row for the old stable id sits in the marks table forever, unattached to anything the stream ever shows again, while the "same" file reappears unmarked under its new id.

**Why it happens:**
Plugin authors reach for the most convenient locally-unique identifier (a path, a filename) without checking whether the source system offers a more durable one (an inode-independent Drive file ID, a content hash) — the contract doesn't mandate durability, only stability "within your plugin," which a path technically satisfies until the file moves.

**How to avoid:**
- For the filesystem plugin, prefer a stable identifier that survives a rename/move within the watched root if the filesystem exposes one (e.g., an inode number combined with a fallback to path when inodes aren't comparable across the watched roots, such as across a network mount boundary) — document the actual choice and its failure mode explicitly, since perfect stability across a rename is not achievable on every filesystem.
- For the Google Drive plugin, always key `source_id` on Drive's own immutable file `id`, never on `name`/`path` — this is a one-line decision to get right at plugin-build time and expensive to fix later once marks exist against the wrong key.
- Add a periodic or sync-triggered sweep that reports (not silently deletes) marks whose `item_id` no longer resolves to any row in `items`, so orphaning is at least observable rather than invisible (see Pitfall 5 for why not deleting automatically also matters).

**Warning signs:**
- The filesystem plugin's `source_id` is literally the file path.
- No test moves/renames a watched file and asserts the item's stable id is unchanged (or, if it can't be, that the mark visibly migrates or reports as orphaned rather than vanishing).

**Phase to address:**
Filesystem plugin phase (choice of `source_id`) and the include/exclude phase (orphan detection/handling) — a cross-phase dependency worth flagging explicitly if these two phases run in parallel.

---

### Pitfall 5: Unbounded mark/tombstone growth with no GC story

**What goes wrong:**
Once marks live in their own table (Pitfall 2's fix), that table has no natural cap. A user who excludes items over months, across a filesystem source with thousands of documents or a Drive folder that gets reorganized, accumulates marks referencing items that have since been deleted from the source, renamed past recognition (Pitfall 4), or belong to a webspace the user later deletes. Nothing in this codebase's existing sync path (`DeleteSourceItems`/`DeleteSyncRuns` fire when a *source instance* is removed from config, not per-item) currently has a hook for "this individual item is gone, clean up anything keyed to it."

**Why it happens:**
Marks are new, user-owned, long-lived state in a system whose only prior persistent, non-config data (`items`, `webspace_items`, `sync_runs`) is entirely sync-derived and already gets bulk-replaced or bulk-deleted at the source/webspace granularity — there's no existing per-item deletion path to hang mark cleanup off of, so it's easy to ship the "add a mark" path without also shipping the "the marked item is gone" path.

**How to avoid:**
- Decide explicitly whether marks cascade-delete when their `item_id` foreign key's row is removed (simplest, but loses the mark if the item is only *temporarily* absent from a sync, e.g., a source outage) versus persisting as orphans with a periodic sweep/report (Pitfall 4's mitigation) that a user or a maintenance command can act on.
- Whichever is chosen, write it down as an explicit behavior (with a test), not an implicit consequence of whatever SQL happens to be easiest.

**Warning signs:**
- No `ON DELETE CASCADE` (or explicit equivalent) on the marks table's foreign key, and no compensating sweep either — the "neither" state where marks just grow forever.

**Phase to address:**
Include/exclude phase — should be a stated design decision in that phase's plan, not left implicit in the schema.

---

### Pitfall 6: Rule-excluded vs user-excluded is a real UX distinction this milestone must define, not just render

**What goes wrong:**
A webspace already has three tiers of filtering before this feature: source-instance participation allowlist, per-instance typed match config, and the keyword fallback (`docs/plugin-contract.md`, "Match" section; PROJECT.md's D-01/D-04 decisions). Per-item include/exclude is explicitly "the final tier." Two failure modes are easy to ship by accident: (1) showing a manually-excluded item identically to one that simply never matched any rule (a user can't tell "I turned this off" from "this was never going to show up"), and (2) leaving undefined what happens when a manual **include** is applied to an item whose match configuration would otherwise exclude it — does the mark override the rule (requiring the item to have entered the index at all, which circles back to Pitfall 3), or is "include" only ever available for items already present?

**Why it happens:**
Filter tiers 1–3 are all *configuration*, evaluated once at sync time, invisible to the end user as separate layers — item-level marks are the first tier that's an end-user-facing, per-item toggle, and it's tempting to build the UI as a simple boolean switch without surfacing which of "rule" or "you" is the reason an item is (or isn't) visible.

**How to avoid:**
- Design the UI affordance to show provenance of exclusion/inclusion explicitly (e.g., "excluded by you" vs "not matched" as distinct, differently-styled states), not one generic hidden/shown boolean.
- Resolve the override question (mark beats rule, or mark only applies within already-matched items) as part of the same scope decision as Pitfall 3, since they're the same underlying question from two angles.

**Warning signs:**
- UI mockups/specs show a single toggle with no visual distinction between "off by rule" and "off by mark."

**Phase to address:**
Include/exclude phase — spec/discuss step, alongside Pitfall 3.

---

### Pitfall 7: ServiceWorker install silently fails over LAN/mobile access because it's not a secure context

**What goes wrong:**
`cmd/topos/main.go`'s `isLoopback` helper already exists specifically because the kernel supports binding to a non-loopback address for LAN access, with "no LAN exposure ships without an explicit, logged warning" (T-01-01). The milestone's PWA requirement explicitly targets **both** desktop *and* mobile installability. A phone on the same LAN necessarily reaches the kernel over `http://<lan-ip>:<port>` — and the Service Worker API is only available in a [secure context](https://developer.mozilla.org/en-US/docs/Web/Security/Secure_Contexts): `https://`, or `http://localhost`/`http://127.0.0.1` specifically. `navigator.serviceWorker` is simply `undefined` (or registration silently no-ops) on a plain-HTTP LAN origin from a mobile browser — there is no error thrown, no console warning in most browsers, and no crash: the "Install app" prompt just never appears, and it will look exactly like a bug in the PWA manifest rather than a fundamental transport-security gate.

**Why it happens:**
Desktop-only testing (`http://localhost:PORT`) is already a secure context, so the feature will appear to work completely in the most obvious dev-loop test, and the gap only shows up when someone actually tries "install on my phone" against the LAN-bound kernel — exactly the scenario the requirement calls out by name.

**How to avoid:**
- Decide this explicitly, early, as a design question rather than discovering it in UAT: either (a) scope mobile PWA install to "only when reached via a trusted-cert HTTPS LAN endpoint" (requiring the kernel to terminate TLS with a self-signed or locally-trusted cert — real added scope), or (b) explicitly scope PWA installability to desktop/localhost only for this milestone and document that mobile install requires a secure-context workaround (e.g., a reverse proxy the user sets up themselves) rather than shipping it as a kernel feature.
- Whichever is chosen, state it in the phase's success criteria in unambiguous terms ("installable on desktop via localhost; mobile requires HTTPS, out of scope this phase" or "kernel gains an HTTPS listen mode for LAN PWA install") — "installable on desktop and mobile" as currently phrased in PROJECT.md doesn't yet make this call.

**Warning signs:**
- The PWA phase's plan has no mention of the kernel's existing loopback-vs-LAN bind distinction.
- Manual testing only ever happens against `localhost`.

**Phase to address:**
The PWA phase — must be resolved before implementation, since it changes whether the kernel needs a TLS listen mode at all.

---

### Pitfall 8: Stale ServiceWorker cache serves an old UI against a new kernel API after upgrade

**What goes wrong:**
The SPA is built once (`web/svelte.config.js` uses `adapter-static`) and embedded into the Go binary via `go:embed` (`kernel/webui/embed.go`) — a kernel upgrade ships a brand-new binary with a brand-new UI bundle *and*, potentially, a changed HTTP API shape in the same release (this repo has already made several breaking-ish API additions across phases, e.g. new endpoints, new response fields). If a ServiceWorker precaches `index.html` and hashed JS/CSS assets aggressively (a common, easy default with generated SW tooling) and doesn't have an explicit update/activation strategy, a browser that already installed the PWA can keep serving the *previous* release's UI — built against the *previous* release's API contract — against the *new* kernel binary indefinitely, because the SW itself is what's stale, and by default browsers only re-check a SW script for byte-level changes on navigation, at most roughly once every 24 hours unless `Cache-Control` on `sw.js` forces more frequent checks. The result is a UI that renders but calls stale endpoints/shapes, degrading confusingly rather than failing loudly.

**Why it happens:**
Precaching everything and calling it done is the fastest path to "PWA installable" checkbox-passing; the update lifecycle (`skipWaiting`, `clients.claim`, a visible "update available, refresh" affordance, and serving `sw.js` itself with `Cache-Control: no-cache`) is the part that's easy to skip because it only manifests after a *second* kernel release, past the point where the first release's UAT would catch it.

**How to avoid:**
- Serve `sw.js` (and any manifest referencing it) with `Cache-Control: no-cache` so the browser re-checks it on every load rather than trusting a stale copy for up to 24 hours.
- Use a versioned/hashed precache manifest tied to the actual build (Vite's own content-hashed asset names already help here) so an old SW's cache references become 404s the SW's own activate handler can detect and purge, rather than silently serving from a cache that no longer matches any served asset.
- Prefer a network-first (or stale-while-revalidate with a visible update prompt) strategy for `index.html` specifically — the one file most likely to reference a stale asset manifest if cached cache-first.
- Extend the existing hermetic Playwright e2e suite (already this project's standing regression gate per CLAUDE.md) with a spec that installs a build, "upgrades" the served bundle, and asserts the client observes and applies the update rather than serving stale content — this is exactly the kind of UAT-drivable check the project's own testing conventions call for.

**Warning signs:**
- The SW registration code has no `updatefound`/`controllerchange` handling.
- `sw.js` isn't explicitly served with cache-defeating headers.

**Phase to address:**
The PWA phase — the update-flow test should be part of that phase's definition of done, not deferred.

---

### Pitfall 9: No launch timeout lets a hung external binary freeze the Add-Plugin flow

**What goes wrong:**
`kernel/pluginhost/host.go`'s `launch()` — the single function every launch path (`Discover`, `Reconcile`, and `DescribePluginType`'s trial-launch used by the "+" add-source UI flow) goes through — constructs `goplugin.ClientConfig` with no `StartTimeout` override, so `hashicorp/go-plugin`'s default handshake timeout applies. That default is generous (on the order of a minute) and was a reasonable choice when every launched binary was built in-repo and known to behave. Once the UI accepts an arbitrary external binary path for a trial-launch describe call, a broken, malicious, or simply non-`go-plugin` executable (one that hangs waiting on stdin, or never emits the handshake line at all) can block that request for the full default timeout with the "Add Plugin" UI showing no useful progress — and because `DescribePluginType` is also reused by "Edit match settings…" on every already-configured instance (per its doc comment), a hang here isn't confined to first-add; it can recur on every edit of a misbehaving instance.

**Why it happens:**
The current fleet of plugins is entirely first-party and well-behaved, so a generous or default timeout has simply never mattered; introducing untrusted external binaries changes the threat/failure model for the exact same code path without necessarily changing the code path itself.

**How to avoid:**
- Set an explicit, short `StartTimeout` on trial-launch specifically (the `describeOnly` launches), distinct from — and shorter than — whatever is acceptable for a real boot-time launch of an already-configured, presumably-working instance.
- Surface a clear "this plugin failed to respond in time" UI state distinct from other launch failures, so a hang doesn't read as the UI itself being frozen.

**Warning signs:**
- `goplugin.ClientConfig` in `launch()` has no `StartTimeout` field set.
- Manual testing of "add a plugin" only ever tries binaries that behave correctly.

**Phase to address:**
The external-plugin-loading phase.

---

### Pitfall 10: Google's OAuth model makes "just ship a client secret" a compounding, not one-time, problem

**What goes wrong:**
Google's own guidance for installed/desktop apps states plainly that such apps "cannot keep secrets" — the client secret is expected to be embedded and is not actually confidential (loopback-redirect flow, no PKCE-only substitute for the refresh-token grant on Google's implementation). That much is a known, accepted tradeoff, not a design flaw per se. What compounds it for an open-source, single-maintainer project specifically: **as long as the Google Cloud OAuth consent screen for that client stays in "Testing" publishing status (the default, and the only realistic status without a formal Google verification review — itself requiring a privacy policy, and for the sensitive Drive scopes this plugin needs, a security assessment), every issued refresh token expires after exactly 7 days**, forcing every user of the shipped Google Drive plugin to re-authenticate weekly, forever, unless the project's maintainer completes Google's app-verification process (a real, ongoing burden disproportionate to a personal local-first tool) or each user registers their own Google Cloud project and brings their own client id/secret. Separately, a client secret string committed to a public open-source repo — even one Google itself considers "not really secret" for this app type — routinely trips GitHub/GitGuardian-style automated secret scanners, generating recurring false-alarm noise for the project.

**Why it happens:**
"Add OAuth to the Drive plugin" reads as a single implementation task; the distribution model (one shared client for every user of the OSS repo, vs. bring-your-own-client) is a separate decision with very different maintenance and UX consequences that's easy to leave implicit until users start reporting weekly forced re-logins.

**How to avoid:**
- Decide explicitly, at design time, between (a) shipping one embedded client id/secret and living with either the 7-day forced-testing-mode re-auth cycle or the ongoing burden of Google app verification, or (b) requiring each user to create their own Google Cloud project and paste their own client id/secret into topos's config — more setup friction, but avoids both the verification burden and the shared-quota problem (see Pitfall 11), and matches this project's existing "user supplies their own credentials" pattern for every other source (paperless-ngx token, Proton Bridge password, SilverBullet token).
- If (a) is chosen anyway, document the 7-day re-auth expectation explicitly in the plugin's README so it isn't reported as a bug.
- Store the refresh token using the same discipline this codebase already applies to other plugin secrets (`docs/plugin-contract.md`'s "never log a credential" rule) — log presence/name only, never the token value.

**Warning signs:**
- The Drive plugin ships a single hardcoded client id/secret with no mention of re-auth cadence anywhere in its docs.
- No test/manual-check exercises "what happens after the refresh token expires" (a real, not hypothetical, weekly event in testing-mode).

**Phase to address:**
The Google Drive plugin phase — this is a first design decision (credential distribution model), not an implementation detail to sort out mid-build. Given this plugin is explicitly built *out-of-repo* to dogfood the external mechanism, this decision also can't lean on this repo's own secret-scanning/CI to catch a leaked-looking string — it's the external plugin author's own repo's problem to get right, which is itself worth validating as part of the dogfooding exercise.

---

### Pitfall 11: Fixed single-plugins-directory assumption breaks when a source needs an out-of-repo binary path

**What goes wrong:**
Today, `[sources.<name>].plugin` is a bare filename resolved via `filepath.Join(pluginsDir, src.Plugin)` against one kernel-wide `[plugins] dir` — there's exactly one trusted root, and every plugin's binary is assumed to live there. External-plugin loading needs *some* plugins to load from elsewhere (wherever the user built or downloaded the Google Drive plugin, say). The naive fix — let `src.Plugin` be an absolute path — reintroduces the classic path-traversal/confusable-path class of bug into a field that used to be a safe, bounded filename: a typo'd or copy-pasted path could resolve to a completely different binary than the one the user meant to trust, and there's no existing validation layer (this field has never needed one) to catch it.

**Why it happens:**
The single-directory model was a correct, minimal design for an all-in-repo plugin fleet; extending the same field to also carry "or an arbitrary external path" without adding new validation is the path of least resistance, but silently changes that field's trust properties.

**How to avoid:**
- Keep the existing bare-filename-resolved-in-`pluginsDir` shape for trusted/in-repo plugins unchanged, and add a distinct, explicitly-named config shape for an external plugin's binary (e.g., a separate `external_path` key, absolute, resolved with symlinks followed and re-displayed to the user in full before they confirm "trust this"), rather than overloading the existing `plugin` field's semantics.
- Whatever the shape, resolve and canonicalize the path once (`filepath.EvalSymlinks` + `filepath.Abs`) and show the *resolved* path in the trust-confirmation UI, not the as-typed one — this is the same "don't let a symlink or relative segment hide what's actually being trusted" discipline that generic path-validation advice already covers, applied to this specific new field.

**Warning signs:**
- `src.Plugin` (or its replacement) accepts both a bare filename and an absolute path with no code branch distinguishing "resolve in the trusted plugins dir" from "resolve as given."

**Phase to address:**
The external-plugin-loading phase.

---

### Pitfall 12: Milestone sequencing creates a real circular dependency between the GDrive and external-loading phases

**What goes wrong:**
PROJECT.md states the Google Drive plugin is "built out-of-repo against the published contract (dogfoods the external-plugin mechanism end to end)" — i.e., it is simultaneously (a) a feature that needs somewhere to load into, and (b) the validation exercise *for* that loading mechanism. If a roadmap schedules substantial Google Drive plugin work (OAuth flow, Drive API client, export handling) before the external-loading mechanism has landed and been exercised against *something*, that work has nowhere real to run and no way to prove the loading path actually works until both land together — and any bug found in the loading mechanism at that late point (e.g., the trust-marking UX, the timeout in Pitfall 9, the path-handling in Pitfall 11) now blocks the hardest, most novel feature in the milestone (a full desktop OAuth flow against a cloud API) rather than a cheap, already-understood one.

**Why it happens:**
Both features read as independent line items in a flat requirements list; the "dogfoods X" phrasing is the only signal of a real ordering dependency, and it's easy to schedule by apparent feature independence rather than by what validates what.

**How to avoid:**
- Sequence the filesystem plugin (in-repo/trusted, no OAuth, no network dependency, the milestone's simplest new source) either alongside or immediately before the external-loading mechanism — its binary is a safe, already-understood stand-in that can validate "does the kernel correctly launch and trust-mark a plugin binary" without also debugging OAuth at the same time. A copy of that same binary, or a second trivial mock-shaped plugin, built and pointed at from outside the repo is a cheap, low-risk first real exercise of the external-loading path before the Google Drive phase begins.
- Only start substantial Google Drive plugin work once external-loading has been proven end-to-end against at least one real out-of-repo binary — treat "external loading works, demonstrated live" as a hard gate/checkpoint before the Drive phase, not an assumption carried forward from the requirement text.

**Warning signs:**
- A roadmap phase order that puts the Google Drive plugin's phase before (or concurrent with, sharing no completed checkpoint from) the external-plugin-loading phase.

**Phase to address:**
Roadmap creation itself — this is an ordering decision, not something a single phase's plan can fix after the fact.

---

## Technical Debt Patterns

Shortcuts that seem reasonable but create long-term problems.

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|-----------------|------------------|
| Recording "trusted" as a boolean with no binary hash | Faster to ship the marking UI | TOCTOU gap (Pitfall 1) — a swapped binary silently inherits stale trust | Never for a real release; borderline acceptable only for an explicitly-labeled "advisory only, not enforced" first cut, clearly communicated as such |
| Watching only top-level configured folders, not recursively, for the filesystem plugin's first cut | Simpler `fsnotify` wiring, no manual recursive-watch-add bookkeeping | Contradicts the stated "optionally subfolders" requirement; users will report missing items from nested folders as bugs | Acceptable only if "subfolders" is explicitly deferred, not silently unimplemented |
| Polling the filesystem instead of `fsnotify` for the initial release | Sidesteps every fsnotify network-mount/inotify-limit pitfall (see below) at a stroke | Higher latency for picking up changes, more CPU on large trees, but genuinely more portable across network mounts | A defensible default for a "docs in a folder" source where near-real-time isn't the point — worth seriously considering, not just falling back to as a last resort |
| Skipping refresh-token proactive renewal for the Drive plugin (only refresh on 401) | Less OAuth-flow code to write initially | Every sync silently no-ops with a stale-looking "source unavailable" state for however long a background sync runs before a live request surfaces the 401 | Never — Health/Match are exactly the paths that need to surface an expired-grant state clearly, per this contract's existing `codes.Unavailable` convention |
| Deferring marks-orphan cleanup (Pitfall 5) entirely for v1.1.0 | Less schema/GC design work this milestone | Slow, invisible table growth; harder to retrofit cleanup once real user data exists | Acceptable if explicitly logged as a known gap with a documented manual cleanup path, not silently absent |

## Integration Gotchas

Common mistakes when connecting to external services/mechanisms specific to this milestone.

| Integration | Common Mistake | Correct Approach |
|-------------|-----------------|-------------------|
| Google Drive API `files.export` (Google Docs/Sheets/Slides) | Calling `files.get?alt=media` on a native Google Docs file (it has no binary content — `files.get` with `alt=media` fails for Google Workspace mime types) | Detect the Workspace `mimeType` (`application/vnd.google-apps.document` etc.) and call `files.export` with an explicit target mime type instead; handle the documented 10 MB export size ceiling (`exportSizeLimitExceeded`) as an expected, not exceptional, outcome for large docs — matches this contract's existing "available=false is normal" convention for `Fetch` |
| Google Drive API quota | Assuming default per-project/per-user quotas never matter for "personal use" and polling on a tight interval across a large Drive | Treat quota errors (`userRateLimitExceeded`/`rateLimitExceeded`) the same way this contract already asks plugins to treat any transient source failure — `codes.Unavailable`, not a crash — and back off; don't assume "single user" means "no rate limiting ever matters," since Drive's per-100-second quotas are tight enough that a naive full-tree poll can trip them |
| `fsnotify` on a configured folder that turns out to be a network mount (NFS/SMB/CIFS) | Assuming a missed-events bug report means the watcher code is broken | NFS/SMB/CIFS provide no kernel-level inotify hook on Linux — `fsnotify` will silently never fire for changes made on the remote side of such a mount; detect the mount type (or simply document the limitation) and fall back to periodic polling for non-local filesystems rather than treating this as a bug to debug forever |
| `fsnotify` + a deep/large watched tree | Recursively adding a watch per subdirectory without checking `fs.inotify.max_user_watches` (a modest default on many Linux distros) | Either cap/document the supported tree depth and file count, surface a clear "too many files to watch, falling back to periodic scan" degraded state (matching this project's existing "degrade honestly, never silently" convention for Signal/WhatsApp), or raise the watch limit as a documented precondition the way the Signal plugin already documents its keyring-daemon precondition |
| `fsnotify` + editors that write via temp-file-then-rename (vim, many others) | Reading the file the instant a `Write`/`Create` event fires, catching a half-written temp file or a moment mid-rename | Debounce on a short settle window and/or watch for the specific rename-into-place event sequence rather than reacting to the first raw event; treat a transient read failure as retry-worthy, not fatal |
| go-plugin trial-launch of an external binary | Reusing the existing `launch()` timeout behavior unchanged (Pitfall 9) | Set an explicit, shorter `StartTimeout` for `describeOnly` launches specifically |

## Performance Traps

Patterns that work at small scale but fail as usage grows.

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|-----------------|
| Blocking initial full-directory scan on the filesystem plugin's first `Match` call | Kernel startup/first sync hangs noticeably; other sources' sync appears stalled if sync is serialized per source | Stream/paginate the initial scan, and confirm (this codebase already fixed exactly this class of bug for WhatsApp's supervisor lock, G-08-5) that a slow source's sync cannot block other sources' routes/health probes | A folder with thousands of files, or one on a slow network mount, on first configure |
| Marks table with no index on `(webspace_name, item_id)` (or its chosen key) | `StreamItems`/`Search` read latency degrades as marks accumulate | Index the marks table on its join key from the start — this codebase already treats query-shape discipline seriously (`BuildMatchQuery`, FTS5 triggers) | Once a user has excluded more than a few hundred items across a long-lived webspace |
| SW precache manifest growing unbounded across releases without cache eviction | Storage-quota pressure on mobile devices; slow install/update | Version the precache and explicitly purge caches not matching the current build id in the SW's `activate` handler | After several kernel releases with an installed PWA that never fully re-installs |

## Security Mistakes

Domain-specific security issues beyond general web security.

| Mistake | Risk | Prevention |
|---------|------|------------|
| Treating "trusted" as verified-safe in UI copy | Users make risk decisions based on a badge that only means "built from this repo," not "scanned" or "sandboxed" | Match the plugin contract doc's own honest framing ("not a sandbox") in every user-facing surface, not just the docs |
| No content-hash re-check at launch (Pitfall 1) | A trusted-marked binary can be silently swapped between trust-time and every subsequent launch | Hash-pin at trust time, re-verify at every `launch()` call |
| Logging the Drive OAuth refresh/access token, or the Signal/paperless-style secrets, in the external Drive plugin's own logging | A leaked local log file exposes a live credential; the existing in-repo contract rule ("never log a credential") isn't mechanically enforced on out-of-repo plugins the way the in-repo AST scan enforces read-only HTTP methods | Since the AST guard (`plugins/` scan for non-GET HTTP methods) explicitly does not extend to out-of-repo plugins per the contract doc, the Drive plugin's own test suite needs to carry an equivalent discipline itself — document this expectation in whatever "how to build an external plugin" guidance ships alongside this milestone |
| Accepting an external plugin's absolute path with no canonicalization/display before the trust confirmation (Pitfall 11) | A symlink or relative-path trick could make the user believe they're trusting one binary while actually trusting another | Resolve symlinks and show the fully-resolved path at confirmation time |
| Binding the kernel to a LAN address to support mobile PWA install without also adding TLS (Pitfall 7) | Widens the existing loopback-only default's attack surface for a feature (PWA) that doesn't actually work without HTTPS anyway | Don't expand LAN exposure as a side effect of chasing mobile PWA install unless HTTPS is part of the same design — the two are coupled, not independent asks |

## UX Pitfalls

Common user experience mistakes in this domain.

| Pitfall | User Impact | Better Approach |
|---------|-------------|-------------------|
| Untrusted-plugin warning shown once at add-time only | User forgets/never learns a source they're actively using runs untrusted, unsandboxed code | Keep the untrusted state visibly, persistently indicated wherever that source appears (matching this project's existing per-source health-chip pattern) — not a one-time toast |
| Generic "item hidden" state with no rule-vs-mark distinction (Pitfall 6) | Users can't tell whether to fix a match rule or just re-include the item | Distinct visual state for "excluded by you" vs "not matched" |
| PWA install prompt that simply never appears on mobile with no explanation (Pitfall 7) | Reads as a broken feature rather than an expected secure-context limitation | Detect the insecure-context case and show an explicit in-app message explaining why install isn't available on this connection, rather than silence |
| Drive plugin failing silently for a week once a refresh token expires (Pitfall 10) | Source looks "just stale" rather than "needs you to re-auth" | Surface the specific expired-grant state distinctly (this contract's `Health`/`codes.Unavailable` machinery already supports a named reason string) with a clear re-auth call to action |

## "Looks Done But Isn't" Checklist

Things that appear complete but are missing critical pieces.

- [ ] **External plugin loading:** Often missing a re-verification step at every launch, not just at add-time — verify a binary swapped after being trusted is caught, not silently re-launched.
- [ ] **Filesystem plugin:** Often missing handling for files still being written (editor temp-file/rename patterns) — verify a large file mid-copy/mid-save doesn't get synced as garbage or a truncated read.
- [ ] **Filesystem plugin:** Often missing behavior on a network-mounted watch root — verify changes made from a *different* machine writing to the same network share are actually picked up (they likely won't be via `fsnotify` alone — see Pitfall/Integration Gotcha above).
- [ ] **Google Drive plugin:** Often missing the distinction between a Workspace-native file (needs `export`) and a regular uploaded file (needs `get?alt=media`) — verify both paths are tested, not just one.
- [ ] **Google Drive plugin:** Often missing an explicit test/runbook for "refresh token has expired" — verify the plugin surfaces this as a named, actionable health state rather than a generic unreachable error.
- [ ] **Include/exclude marks:** Often missing survival across a resync — verify a mark set before a sync is still present and correctly applied after that same item resyncs (Pitfall 2).
- [ ] **Include/exclude marks:** Often missing a defined answer for "what happens to a mark when the source item disappears" — verify orphan behavior is a deliberate choice, not silent accumulation (Pitfall 5).
- [ ] **PWA installability:** Often missing an update-flow test — verify that upgrading the kernel binary actually results in an already-installed client picking up the new UI/API contract within a bounded time, not indefinitely serving a stale cached build (Pitfall 8).
- [ ] **PWA installability:** Often missing secure-context handling for the LAN/mobile case explicitly called for in scope — verify install is tested from an actual second device on the LAN, not only from `localhost` on the dev machine (Pitfall 7).

## Recovery Strategies

When pitfalls occur despite prevention, how to recover.

| Pitfall | Recovery Cost | Recovery Steps |
|---------|-----------------|------------------|
| Marks wiped by sync-replace (Pitfall 2) | LOW if caught before release (schema not yet public); MEDIUM after release | Move marks to an independent table, backfill is impossible once wiped (data is gone) — communicate the loss to users if this ships and is later discovered, since there's no recovery of the actual prior mark state |
| Trust marking without hash-pinning shipped (Pitfall 1) | MEDIUM | Add the hash field and re-verification in a follow-up phase; existing "trusted" entries need a one-time re-hash-and-confirm migration step so they don't silently become "trusted with no hash on file" forever |
| PWA ships without secure-context handling and mobile install reports flood in (Pitfall 7) | LOW–MEDIUM | Ship the explicit in-app "install unavailable on this connection" messaging as a fast follow; the harder TLS-for-LAN option can stay a separate, later decision |
| Google Drive plugin ships with one shared, testing-mode OAuth client and users hit 7-day expiry (Pitfall 10) | MEDIUM | Switch to a bring-your-own-client-id model in a follow-up release; existing users need to be walked through creating their own Google Cloud project once |
| fsnotify silently misses network-mount changes and this is discovered post-release (Integration Gotcha) | LOW | Add a documented polling fallback mode, defaulted on for any watched root the plugin detects (or the user declares) as network-mounted |

## Pitfall-to-Phase Mapping

How roadmap phases should address these pitfalls.

| Pitfall | Prevention Phase | Verification |
|---------|-------------------|----------------|
| Trust marking as security theater / TOCTOU (1) | External-plugin-loading phase | A test that swaps a trusted binary's bytes after trust is granted and asserts the kernel refuses/warns on next launch |
| No launch timeout on external trial-launch (9) | External-plugin-loading phase | A test plugin that hangs before handshake, asserting the Add-Plugin flow fails fast with a named timeout, not a multi-second/minute hang |
| Single-pluginsDir / path handling for external binaries (11) | External-plugin-loading phase | A test with a symlinked or relatively-specified external path asserting the resolved, canonical path is what's shown/trusted |
| GDrive/external-loading sequencing (12) | Roadmap creation (phase ordering) | External-loading phase demonstrably validated against a real out-of-repo binary before the Google Drive phase begins substantial work |
| fsnotify inert on network mounts / inotify limits / mid-write reads (Integration Gotchas) | Filesystem-plugin phase | Manual or automated test against an actual network-mounted folder and a large/deep directory tree, plus a test that edits a file via a temp-file-rename pattern mid-watch |
| Google OAuth credential distribution + 7-day testing-mode expiry (10) | Google Drive plugin phase (design step, before implementation) | A documented, explicit decision on shared-client vs bring-your-own-client, recorded the way this project's existing Key Decisions table records tradeoffs |
| Google Drive export/mime-type gotchas (Integration Gotchas) | Google Drive plugin phase | Tests against both a native Google Doc and a regular uploaded file type |
| Marks wiped by sync-replace (2) | Include/exclude phase (schema design step) | Regression test: mark, resync, read — mark still applied |
| Include of a never-matched item is impossible today (3) | Include/exclude phase (spec/discuss step, before planning) | Scope explicitly recorded before any schema/UI work starts |
| Rule-excluded vs user-excluded UX ambiguity (6) | Include/exclude phase (spec/discuss step) | UI spec shows visually distinct states for each reason |
| Mark orphaning on stable-id change (4) | Filesystem-plugin phase (source_id choice) + Include/exclude phase (orphan handling) | Test: rename/move a watched file, assert mark behavior is deliberate, not silent loss |
| Unbounded mark/tombstone growth (5) | Include/exclude phase (schema design step) | Explicit cascade-or-sweep decision recorded and tested |
| ServiceWorker secure-context gap on LAN/mobile (7) | PWA phase (design step, before implementation) | Install tested from a real second device over LAN, not just localhost |
| Stale SW cache after kernel upgrade (8) | PWA phase | Playwright e2e spec simulating an upgrade and asserting the client observes/applies it |

## Sources

- `/home/darren/projects/davison/topos/docs/plugin-contract.md` — the published plugin contract (handshake, Describe/Match/Fetch/Health semantics, "not a sandbox" framing, stderr-capture discipline) — HIGH, primary source
- `/home/darren/projects/davison/topos/kernel/pluginhost/host.go` — actual `launch()`/`Discover`/`Reconcile`/`DescribePluginType` implementation, confirms no `StartTimeout` override and the `filepath.Join(pluginsDir, src.Plugin)` resolution shape — HIGH, primary source
- `/home/darren/projects/davison/topos/kernel/index/store.go` — `ReplaceWebspaceSourceItems`'s delete-then-reinsert transaction, confirming the sync-replace mechanism that would wipe marks stored on the wrong table — HIGH, primary source
- `/home/darren/projects/davison/topos/cmd/topos/main.go` — `isLoopback` and the existing loopback-default/LAN-warning behavior — HIGH, primary source
- `/home/darren/projects/davison/topos/.planning/PROJECT.md` — v1.1.0 milestone scope, requirement phrasing ("dogfoods the external-plugin mechanism," "final tier of the filter hierarchy," "first user-owned data beyond config") — HIGH, primary source
- Google for Developers, "OAuth 2.0 for iOS & Desktop Apps" (developers.google.com/identity/protocols/oauth2/native-app) — installed apps "cannot keep secrets," loopback flow retained for desktop — HIGH
- Google Cloud Platform Console Help / Google Developer forums discussion on desktop app client secrets — MEDIUM-HIGH, corroborating community/official mix
- Multiple corroborating sources (Nango, Unipile, CData, Google's own support threads) on the 7-day refresh-token expiry for "Testing" publishing status apps — MEDIUM-HIGH, consistent across independent sources
- Google Drive API v3 `files.export` reference (developers.google.com/workspace/drive/api/reference/rest/v3/files/export) and Google Issue Tracker entries on `exportSizeLimitExceeded` (10 MB ceiling) — HIGH, official docs plus corroborating issue reports
- `fsnotify/fsnotify` GitHub repo and LWN.net ("Change notifications for network filesystems") — NFS/SMB/CIFS lack Linux-side inotify hooks; fanotify as a partial alternative — MEDIUM-HIGH, well-documented upstream limitation
- MDN, "Secure Contexts" (developer.mozilla.org/en-US/docs/Web/Security/Secure_Contexts) — ServiceWorker's secure-context requirement (HTTPS or localhost only) — HIGH, standard web-platform reference (not separately re-fetched this session; established web-platform knowledge cross-checked against this project's own `isLoopback` code)

---
*Pitfalls research for: topos v1.1.0 "Plugin Ecosystem" milestone*
*Researched: 2026-08-12*
