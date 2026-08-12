# Feature Research

**Domain:** Plugin trust UX, filesystem/cloud document sources, manual curation over automated rules, and PWA installability — for a local-first, single-user personal data aggregator (topos v1.1.0 "Plugin Ecosystem")
**Researched:** 2026-08-12
**Confidence:** MEDIUM (built-in WebSearch only — the project's `research-plan`/`classify-confidence`/`research-store` seam commands are not registered in this environment's installed `gsd-tools`, same gap noted in the v1.0 research cycle. Every claim below is cross-checked across 2+ independent sources where the topic allowed it; single-source claims are flagged inline.)

**Scope note:** This file covers only the five NEW v1.1.0 features. topos's five source plugins, kernel+plugin architecture, hybrid data model, stream/detail UI, webspace builder, and agent permissions are already built (v1.0) and are explicitly out of scope for re-research — see `.planning/PROJECT.md`.

## Feature Landscape

Four comparable-but-distinct ecosystems were surveyed, one per new feature area:

- **Plugin/extension trust UX** — VS Code Marketplace (publisher verification + first-install-from-third-party dialog), Obsidian community plugins (Restricted Mode + automated safety scorecard), Home Assistant HACS (default vs. custom repositories). All three converge on the same shape: a *binary* trusted/untrusted signal shown at install time, not a sandboxing mechanism — none of them actually contain what an installed extension/plugin can do once running.
- **Folder-watching document sources** — sync-client literature (FreeFileSync, rsync, Dropbox-style clients) for rename/move detection patterns, applied to a *read-only, metadata-only* index rather than full bidirectional file sync (topos's problem is much narrower).
- **Google Drive as a read-only source** — Drive API v3 official docs: `files.export` (Workspace-native docs have no raw bytes — must be exported), `changes.list`/`getStartPageToken` (incremental sync), `supportsAllDrives`/`includeItemsFromAllDrives` (Shared Drives are opt-in, not automatic).
- **Manual override over automatic rules** — Gmail filters vs. Inbox "importance" overrides, iTunes/Plex smart playlists, Google Photos Memories manual hide. No single reference app documents a clean "exception list survives re-evaluation" pattern publicly, but the shape recurs informally (openHAB's "rule exception for manual override" community pattern is the closest explicit analog) and is corroborated by how Gmail explicitly warns users that its own logic can override filter-driven label placement — the anti-pattern topos must avoid.
- **PWA installability for a localhost-served personal tool** — MDN/web.dev secure-context and installability docs. The single most consequential finding: **`localhost`/`127.0.0.1` is treated as a secure context, but a LAN IP address is not** — this directly constrains what "installable on mobile" can mean for topos as currently deployed (see Anti-Features and Dependencies below).

### Table Stakes (Users Expect These)

Features users assume exist. Missing these = product feels incomplete or unsafe.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Trust warning shown at add-time for any non-trusted plugin | VS Code (since 1.97) shows a confirm-you-trust-this-publisher dialog on first install from a third-party publisher; Obsidian ships Restricted Mode ON by default specifically so a user has to make an affirmative "I trust this" decision before any community plugin runs | LOW-MEDIUM | topos's own `docs/plugin-contract.md` already states the intended policy in prose ("only run plugin binaries you built yourself or whose source you trust") — this feature is turning that existing prose warning into an enforced, user-facing UI gate at the point a plugin is added |
| Persistent, visible trust marking (not just a one-time dialog) | VS Code's blue-checkmark publisher badge and HACS's default-vs-custom-repository distinction are both *always visible*, not one-time — a user revisiting the source list later still needs to see which sources are unverified | LOW | Extends the existing per-source chip/health-chip UI (v1.0) with a trust indicator; no new architecture, just a new badge state |
| Trust determination based on build provenance, not self-declaration | VS Code's Marketplace signs every published extension and verifies the signature on install specifically so publisher identity can't be spoofed; a plugin that merely *claims* to be built from `davison/topos` is not evidence | MEDIUM | "Trusted" must be computed from something structural (e.g., checksum/signature against a manifest of official release builds, or binaries placed by the kernel's own build/release pipeline vs. binaries dropped into an external plugins directory by any other means) — this is the one piece of the trust feature that isn't "just UI" |
| Filesystem plugin: common document file types recognized (markdown, plain text, PDF, common office formats) | Every folder-watching tool (Hazel, sync clients, Paperless-ngx's own consumption folder) filters by file type/extension rather than treating every byte in a directory as a document | LOW-MEDIUM | Extension/MIME-sniff allowlist, consistent with the existing plugin contract's declared `ContentShape` model (v1.0) — no new kernel concept needed, just a new plugin's classification logic |
| Filesystem plugin: optional recursive subfolder scanning | PROJECT.md's stated target ("docs in a folder, optional subfolders") matches how every folder-watcher in the space works — flat-only is the surprising, unwanted default | LOW | Config-level toggle per source instance, same shape as existing per-instance config (KERN-06) |
| Filesystem plugin: renames/moves don't orphan index rows or break "why is this here" provenance | Sync-client literature is unanimous that naive path-based re-scan treats a rename as delete+create, which for topos means a real item disappearing and reappearing as "new" with lost per-item state (see per-item include/exclude below) — this becomes a *correctness* bug once per-item marks exist, not just a UX nicety | MEDIUM-HIGH | Full content-hash rename detection (FreeFileSync's approach) is expensive at scale; topos's index is metadata-only so a cheaper identity (stable OS inode/file ID where the filesystem provides one, else path-as-identity with a documented "moving a file = the item is treated as new" limitation) is the pragmatic scope — flag as a design decision, not assume the expensive approach |
| Google Drive plugin: folder-scoped, not whole-Drive | PROJECT.md's stated target mirrors the filesystem plugin ("docs-in-a-folder"); Drive API v3's `files.list` easily supports a `'<folderId>' in parents` scope query, so this is a query-shape decision, not a missing-capability problem | LOW | OAuth consent should request a scope no broader than necessary — see `drive.readonly` vs `drive.file` tradeoff in Dependencies |
| Google Drive plugin: Workspace-native docs (Google Docs/Sheets/Slides) exported to a renderable format | These files have no raw byte content in Drive at all — `files.get`'s `alt=media` download only works for uploaded binary files; native Docs/Sheets/Slides *must* go through `files.export` to a target MIME type (PDF/plain text/DOCX/etc.) | MEDIUM | This is a hard API constraint, not a design choice — the plugin's `Fetch`/preview path needs a native-vs-uploaded branch. Export is also capped (10MB per Google's docs) which matters for large native docs |
| Google Drive plugin: incremental sync via `changes.list`, not a full folder re-list every sync | Every Drive API integration guide treats `changes.list` + `getStartPageToken`/`pageToken` as the standard sync loop; polling `files.list` from scratch each time wastes quota and can't cheaply distinguish "still there" from "removed" | MEDIUM | Requires persisting a `pageToken` per source instance across syncs — new small piece of plugin-owned state, same shape as WhatsApp's plugin-owned session store (v1.0 precedent) |
| Per-item include/exclude visible directly on stream rows | PROJECT.md names this "the final tier of the filter hierarchy" sitting below webspace keyword rules and per-instance match blocks — users expect the finest-grained control to be reachable at the point of the item itself, not buried in a settings screen, mirroring how email clients let you act on a message inline rather than only via a filter-editing screen | LOW-MEDIUM | UI-level addition to the existing stream row/detail pane; the hard part is the backend semantics below, not the affordance itself |
| Manual exclude/include always wins over automatic match rules, deterministically and permanently until changed | This is the load-bearing requirement of the whole feature — every reference system with a curation-over-rules layer (Gmail's own docs on *not* letting Gmail's importance heuristic override a user's explicit filter, iTunes smart playlists allowing a manually-added track to persist even after it stops matching criteria) treats "explicit user action" as higher-precedence and durable, not a one-time nudge that gets silently reverted on the next automatic pass | MEDIUM-HIGH | Needs new persisted per-item state, keyed by a stable (source instance, source item ID) pair — not a DB row ID — so it survives full index rebuilds (the KERN-06 schema-version-gated rebuild precedent already exists and must be extended to preserve this new state across a rebuild) |
| Manual marks visually distinguishable from automatic membership | "Why is this here" provenance already exists in topos v1.0 (surfaces the matched keyword/native field); a manually-included or manually-excluded item needs its own provenance state ("kept despite no match" / "excluded despite matching") so the existing trust-building UI pattern extends cleanly rather than becoming ambiguous | LOW | Reuses the existing provenance UI slot with a new value, not a new UI surface |
| PWA: valid manifest + registered service worker so an install prompt/affordance appears | Baseline Chromium/Chrome-family install criteria (manifest with name/icons/display:standalone + HTTPS-or-localhost + registered SW) — without all three, no `beforeinstallprompt` fires and there's no "Install" affordance at all | LOW-MEDIUM | Piggybacks on the existing kernel static-asset-serving pattern (Phase 9's icon route precedent) |
| PWA: app-shell caching so reload/launch is fast and doesn't flash-of-unstyled-content | Standard PWA installability expectation once a service worker exists at all — cache-first for the built SPA's JS/CSS/icons is the conventional minimum SW strategy | LOW | Cache the static SPA bundle only, not API data — see Anti-Features for why full offline data caching is explicitly wrong for this app |

### Differentiators (Competitive Advantage)

Features that set topos's plugin-ecosystem features apart. Not required, but valuable given the existing product's emphasis on deterministic, inspectable behavior.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Trust badge carries build provenance detail (which commit/release built this binary), not just a trusted/untrusted flag | VS Code and HACS both stop at a binary signal; topos's own architecture (release pipeline already produces attestable static binaries — v1.0 Phase 10 decision to keep every published artifact `CGO_ENABLED=0` and reproducible) makes a *richer* trust signal cheap to add later and consistent with the project's existing "inspectable, not magic" design philosophy (the same reasoning that drove native-taxonomy correlation over black-box matching in v1.0) | MEDIUM | Natural v1.x follow-on once the binary trusted/untrusted flag ships; don't build both at once |
| Explicit UI distinction between "excluded by manual override" and "never matched any rule" | None of the surveyed reference apps (Photos Memories, smart playlists) surface this distinction cleanly — a manually-excluded item and a never-matched item look identical to the user in every system reviewed. Making the distinction visible directly serves topos's existing "why is this here" trust-building pattern | LOW | Cheap once the per-item state model exists (see Table Stakes) — this is a UI label choice, not new plumbing |
| Bulk include/exclude ("exclude everything from this folder/sender/group") | One-by-one curation fatigue is a known failure mode wherever manual override exists without a bulk path (implicit in why smart-playlist/email-filter systems all eventually grow rule-editing UIs); topos already has the underlying match primitives (per-instance match blocks, KERN-07) that a "promote this exclusion to a rule" action could target | MEDIUM-HIGH | Explicitly defer past MVP — this blurs the boundary between the per-item tier and the match-block tier of the filter hierarchy and needs its own design pass, not a v1.1.0 add-on |
| Same PWA install experience usable from a phone over the LAN, not just the desktop machine itself | Nothing in the surveyed PWA ecosystem treats "install a personal server's app from another device on the same network" as a solved, common case — most PWA guidance assumes a public HTTPS-hosted app. topos already crosses this boundary for the *browser* UI (v1.0 "works on mobile widths"); extending true installability to a LAN client is a differentiator specific to a local-first tool | HIGH | Blocked on a non-trivial dependency — see Dependencies and Anti-Features. Realistic v1.1.0 scope is desktop-only true installability; LAN/mobile installability needs local HTTPS (self-signed CA trusted on the phone, or a tool like `mkcert`/Tailscale-style overlay) which is a meaningfully bigger, separate piece of work |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create problems given topos's existing architecture and constraints.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|------------------|-------------|
| Sandboxing/capability restriction for untrusted plugins (process isolation, syscall filtering, network allowlisting per plugin) | "Untrusted" naturally suggests "should be contained," and WASM-style sandboxing is the obvious-sounding fix | topos's v1.0 stack research already rejected Extism/WASM as the plugin mechanism specifically because two existing plugins (Signal's cgo/SQLCipher linkage, WhatsApp's long-lived stateful WebSocket) need capabilities WASM restricts; `hashicorp/go-plugin` is explicitly "a transport, not a sandbox" per `docs/plugin-contract.md`. Building real containment now would be a second, much larger architecture change disguised as a UX feature, and the existing docs already tell users installing a plugin binary is equivalent to installing the kernel itself — a v1.1.0 feature that pretends otherwise sends a false safety signal | Warning + persistent trust marking + (differentiator) build-provenance detail — an honest, VS-Code/Obsidian-shaped signal, not a technical guarantee that doesn't exist |
| Content-hash-based rename/move detection across the whole watched tree (the FreeFileSync/rsync-delta approach) | "Detect renames properly" sounds like the correct, complete solution | Requires hashing every file's content on every sync pass (or maintaining a persistent hash database) purely to preserve identity for a *metadata-only preview index* — expensive for a feature whose downside (a moved file's item briefly looking "new") is cosmetic, not data-lossy, as long as the file's content itself is never duplicated into the index (already true per the v1.0 hybrid data model) | Cheaper identity: OS-provided stable file ID/inode where available, else path-as-identity with a documented limitation that a move outside the plugin's control is treated as remove+add |
| Two-way sync / write-back to Google Drive or the filesystem (edit a doc from topos) | Natural extension once a source is "connected," and users of full sync clients expect bidirectionality | Directly contradicts an existing, explicit out-of-scope decision in PROJECT.md ("Write/edit functionality in any source — view-only by design") — this isn't a new anti-feature discovered in research, it's a restatement of a standing constraint that a Drive/filesystem source makes newly tempting to violate | Read-only fetch + "open in source" deep link (Drive: open the file's `webViewLink`; filesystem: open the OS file manager or default app at the path), same shape as every existing plugin |
| Full offline data caching in the browser (IndexedDB mirror of the stream/search index) so the PWA "works offline" | "Offline-first" is the default mental model for PWAs, and service workers are strongly associated with offline data access | topos's backend *is* the local kernel process — there is no meaningful "offline" state distinct from "the kernel isn't running," and a browser-side data mirror would duplicate the kernel's own local index for zero benefit while adding real staleness/sync-conflict surface area topos's hybrid data model was specifically designed to avoid (v1.0 architecture decision: index is metadata/preview only, full content fetched live) | Service worker caches the static SPA shell (JS/CSS/icons) only, for fast reload; API calls stay network-first with no offline fallback — matches the "offline-first vs. local-first are not the same thing" distinction found in research |
| A generalized plugin marketplace/registry (search, ratings, one-click install by name) | The natural end-state once "third-party plugins" exists, and VS Code Marketplace/HACS are the visible role models | Explicitly deferred in PROJECT.md's own scope note for this milestone ("distribution, dev guide, and certification deferred") — this milestone is load + trust marking only, for a single user manually placing binaries they already know about (their own Google Drive plugin, built to dogfood the mechanism) | Plugins directory + trust marking only; a registry is a plausible, separate future milestone once there's more than one real third-party plugin author |
| PWA installable from a phone over the LAN in this milestone, treated as equivalent to desktop install | It's in the stated milestone target ("installs on desktop and mobile") and looks like a checkbox feature | Research finding: service workers (and therefore PWA installability) require a secure context, and **a LAN IP address does not qualify** — only `localhost`/`127.0.0.1` get a browser exception to the HTTPS requirement. A phone reaching the desktop kernel over the LAN is, by definition, not hitting `localhost`, so out-of-the-box mobile install will silently fail service-worker registration even with a perfect manifest | Scope "mobile" realistically: either (a) desktop-only true installability for v1.1.0 with mobile tracked as a differentiator needing local HTTPS, or (b) budget the local-HTTPS work (self-signed cert trusted on the phone, or equivalent) as part of this feature rather than discovering the gap during UAT |

## Feature Dependencies

```
[Untrusted plugin warning + persistent marking]
    └──requires──> [Trust-determination mechanism]
                       (checksum/signature against known-official builds,
                        or origin-of-binary tracking — not self-declaration)

[Google Drive plugin, built out-of-repo]
    └──requires──> [External plugin loading feature]
                       (the milestone's own stated purpose: Drive dogfoods
                        the external-plugin path — build it against the
                        already-published topos.v1 contract, no kernel change needed)
    └──requires──> [OAuth read-only Drive scope decision]
                       (drive.readonly vs. narrower drive.file — affects
                        what consent screen the user sees and what folders
                        are visible to the plugin)

[Filesystem plugin: rename/move handling]
    └──requires──> [Per-item include/exclude persistence]
                       (a naive rename = delete+create would silently drop
                        a user's manual mark on a moved file — the identity
                        scheme chosen for renames directly determines
                        whether per-item state survives a move)

[Per-item include/exclude]
    └──requires──> [New kernel-owned persisted item state]
                       (PROJECT.md already flags this as "the kernel's
                        first user-owned data beyond config" — this is not
                        a config-file change like KERN-06/07, it needs its
                        own durable store keyed by (source instance, source
                        item ID), independent of the index's DB row IDs so
                        it survives the existing schema-version-gated
                        index rebuild)
    └──interacts-with──> [Per-instance match blocks (KERN-07) + webspace
                           keyword fallback]
                       (precedence must be defined: manual mark beats both
                        automatic tiers, in both directions — include
                        overrides a non-match, exclude overrides a match)

[PWA installability, full desktop+mobile scope]
    └──requires──> [Local HTTPS for LAN access]
                       (mobile install over LAN IP fails the secure-context
                        check that desktop's localhost access passes for
                        free — this is the one feature in this milestone
                        with a hard, unavoidable platform-level dependency
                        not solvable by application code alone)

[PWA app-shell caching] ──reuses──> [Existing static-asset-serving pattern]
                       (same kernel route shape as the v1.0 plugin-icon
                        MIME-allowlisted serving route)
```

### Dependency Notes

- **Untrusted plugin warning requires a trust-determination mechanism:** a warning dialog is trivial UI; deciding *what makes a plugin trusted* is the real work, and it must be structural (build provenance) rather than a checkbox the plugin author ticks, or the whole feature is theater.
- **Google Drive plugin requires the external plugin loading feature to exist first (or in lockstep):** this is explicit in PROJECT.md's own framing — Drive isn't just "another source," it's the mechanism's proof case. Sequencing them in the same phase, or Drive strictly after external loading, both work; Drive before external loading does not (there'd be nothing to dogfood).
- **Filesystem rename/move handling requires per-item include/exclude to already have a durable identity scheme, or vice versa:** whichever ships first constrains the other's design. Recommend deciding the stable-identity approach (source instance + source item ID) once, and having both features consume it, rather than each inventing its own key.
- **Per-item state must survive the existing index rebuild mechanism:** KERN-06 already established a precedent (schema-version-gated rebuild preserves index data across plugin/schema changes) — the new per-item store needs the same durability discipline or a rebuild silently wipes user curation.
- **PWA's mobile-install ambition requires local HTTPS, which requires its own design decision** (self-signed CA the user trusts once on their phone vs. an overlay-network approach vs. scoping mobile out for this milestone) — this is the one dependency in this feature set that isn't solvable by application code alone and should be explicitly resolved during phase discussion, not discovered mid-build.

## MVP Definition

### Launch With (v1.1.0)

Minimum viable slice of each of the five target features — matches PROJECT.md's stated scope exactly.

- [ ] Untrusted plugin warning at add-time, with a structural (not self-declared) trust determination, and a persistent trust badge — no sandboxing
- [ ] Filesystem plugin: single configured folder, optional recursive subfolders, common document types, path-as-identity rename handling (documented limitation accepted)
- [ ] Google Drive plugin, built out-of-repo: single folder scope, OAuth read-only, Workspace-doc export to PDF/text, `changes.list` incremental sync — no Shared Drives
- [ ] Per-item include/exclude: visible per-row action, persisted keyed by stable (source instance, source item ID), always wins over automatic match/keyword rules, visually distinguished from automatic membership
- [ ] PWA installability: manifest + service worker + app-shell caching, true install on desktop (`localhost`); mobile/LAN install explicitly scoped as best-effort or deferred pending the local-HTTPS decision

### Add After Validation (v1.x)

- [ ] Trust badge carries build-provenance detail (commit/release, not just trusted/untrusted) — once the binary trust flag is proven useful
- [ ] Bulk include/exclude actions — once single-item curation shows the one-by-one fatigue pattern in real use
- [ ] Google Drive Shared Drives support (`supportsAllDrives`/`includeItemsFromAllDrives`) — once single-folder-in-My-Drive is validated
- [ ] Mobile/LAN PWA install via local HTTPS — once the desktop-only install is shipped and the local-HTTPS approach is chosen deliberately, not rushed into this milestone

### Future Consideration (v2+)

- [ ] Plugin marketplace/registry (search, distribution, dev guide, certification) — explicitly deferred in PROJECT.md
- [ ] Plugin sandboxing/process isolation — would require revisiting the `hashicorp/go-plugin` architecture decision itself, not a v1.1.0-scale change
- [ ] OneDrive plugin — explicitly deferred in PROJECT.md
- [ ] "Promote a manual exclusion to a rule" bulk-to-match-block escalation — genuinely new filter-hierarchy design work, not a curation feature

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Untrusted plugin warning + trust marking | HIGH (safety-critical once external loading exists) | MEDIUM | P1 |
| Filesystem plugin (folder + subfolders, common types) | HIGH (explicitly the MVP-deferred source) | MEDIUM | P1 |
| Google Drive plugin, out-of-repo | HIGH (proves the external-plugin mechanism end to end) | MEDIUM-HIGH | P1 |
| Per-item include/exclude | HIGH (closes the filter hierarchy; first user-owned data) | MEDIUM-HIGH | P1 |
| PWA installability (desktop) | MEDIUM | LOW-MEDIUM | P1 |
| PWA installability (mobile/LAN) | MEDIUM | HIGH (external HTTPS dependency) | P2 |
| Trust provenance detail (commit/build info) | MEDIUM | MEDIUM | P2 |
| Bulk include/exclude | MEDIUM | MEDIUM-HIGH | P2 |
| Google Drive Shared Drives | LOW-MEDIUM | MEDIUM | P3 |
| Plugin marketplace/registry | LOW (no third-party authors yet) | HIGH | P3 |

**Priority key:**
- P1: Must have for v1.1.0 launch (matches PROJECT.md's Active requirements)
- P2: Should have, add when a v1.x window opens
- P3: Nice to have, future consideration (v2+ / explicitly deferred)

## Competitor / Reference-System Feature Analysis

| Feature Area | Reference System A | Reference System B | topos's Approach |
|---------------|---------------------|---------------------|-------------------|
| Plugin trust signal | VS Code: blue-checkmark verified-publisher badge + first-install confirm dialog + Marketplace-signed package verification | Obsidian: Restricted Mode default-on + automated safety scorecard per plugin version | Binary trusted (built from `davison/topos`) / untrusted marking + add-time warning; no automated scanning (out of scope — no marketplace to scan against yet) |
| Third-party repo handling | Home Assistant HACS: default (vetted) repositories vs. user-added custom repositories, both listed together with no special containment | — | Same shape: configured plugins directory holds both trusted and untrusted binaries, distinguished only by badge/warning, not by capability |
| Folder source rename detection | FreeFileSync/rsync-delta: content-hash based, requires a persisted hash database, expensive but exact | Simple path-watchers (Hazel-style): path is identity, a move is a new event | topos: path-as-identity (Hazel-style) — matches the metadata-only index's tolerance for occasional "looks new" cosmetic drift, avoids the hash-database cost FreeFileSync accepts for full bidirectional sync correctness topos doesn't need |
| Cloud doc export | Google Drive API `files.export`: server-side conversion to PDF/DOCX/etc., required for all native Workspace docs, ~10MB cap | — | Same mechanism, no alternative exists — this is a hard API constraint, not a design choice |
| Manual override durability | Gmail: explicit "Don't override filters" setting fights the mail client's own importance heuristic reverting filter placement (an anti-pattern to avoid) | iTunes smart playlists: a manually-added non-matching track persists until explicitly removed | topos: manual include/exclude is unconditionally durable and always wins, matching iTunes's model rather than Gmail's overridable-by-the-system model |
| PWA install scope | Standard web.dev/MDN guidance: HTTPS-or-localhost + manifest + SW = installable, no distinction made for "personal LAN server" apps | — | topos must explicitly design around the localhost-vs-LAN-IP secure-context gap that generic PWA guidance doesn't address, because its deployment model (desktop-hosted, LAN-reachable) is exactly the edge case that guidance glosses over |

## Sources

- [Extension Marketplace — VS Code docs](https://code.visualstudio.com/docs/configure/extensions/extension-marketplace) — MEDIUM (official docs, cross-checked)
- [Security and Trust in Visual Studio Marketplace — Microsoft for Developers blog](https://developer.microsoft.com/blog/security-and-trust-in-visual-studio-marketplace/) — MEDIUM (official vendor blog)
- [Extension runtime security — VS Code docs](https://code.visualstudio.com/docs/configure/extensions/extension-runtime-security) — MEDIUM (official docs)
- [Plugin security — obsidianmd/obsidian-help](https://github.com/obsidianmd/obsidian-help/blob/master/en/Extending%20Obsidian/Plugin%20security.md) — MEDIUM (official docs repo)
- [Community plugins — Obsidian Help](https://obsidian.md/help/community-plugins) — MEDIUM (official docs)
- [HACS Custom Repositories FAQ](https://www.hacs.xyz/docs/faq/custom_repositories/) — MEDIUM (official project docs)
- [Detection of moved and renamed files — FreeFileSync Forum](https://freefilesync.org/forum/viewtopic.php?t=1822) — LOW-MEDIUM (community forum, single source for the hash-database mechanism claim)
- [Detecting File Moves & Renames with Rsync — Lincoln Loop](https://lincolnloop.com/blog/detecting-file-moves-renames-rsync/) — LOW-MEDIUM (independent blog, corroborates rename-detection cost tradeoff)
- [Export MIME types for Google Workspace documents — Google for Developers](https://developers.google.com/workspace/drive/api/guides/ref-export-formats) — HIGH (official API docs)
- [Method: files.export — Google Drive API v3 Reference](https://developers.google.com/workspace/drive/api/reference/rest/v3/files/export) — HIGH (official API reference)
- [Implement shared drive support — Google Drive API guide](https://developers.google.com/workspace/drive/api/guides/enable-shareddrives) — HIGH (official API docs)
- [Retrieve changes — Google Drive API guide](https://developers.google.com/workspace/drive/api/guides/manage-changes) — HIGH (official API docs)
- [Gmail Override Filters — SysTools blog](https://www.systoolsgroup.com/updates/override-filters-in-gmail/) — LOW-MEDIUM (third-party explainer, not official Google docs, but consistent with Gmail's own documented "Don't override filters" setting)
- [Create rules to filter your emails — Gmail Help](https://support.google.com/mail/answer/6579?hl=en) — HIGH (official docs)
- [Plex Smart Playlists community/forum discussion](https://forums.plex.tv/t/ability-to-edit-change-smart-playlists/588757) — LOW (forum, weak single-source corroboration for smart-playlist manual-override framing — treat the iTunes/Plex claim as MEDIUM-confidence pattern inference, not a directly documented feature)
- [Remove a photo from a featured memory — Google Photos Community](https://support.google.com/photos/community-guide/283381301/remove-a-photo-from-a-featured-memory?hl=en) — MEDIUM (official Google support community guide)
- [Rule exception for manual override? — openHAB Community](https://community.openhab.org/t/rule-exception-for-manual-override/37206) — LOW-MEDIUM (community forum, best available explicit analog for "exception list overrides automatic rule re-evaluation")
- [Web app manifest installability requirements — Chrome for Developers / Lighthouse](https://developer.chrome.com/docs/lighthouse/pwa/installable-manifest) — HIGH (official vendor docs)
- [Installation prompt — web.dev](https://web.dev/learn/pwa/installation-prompt) — HIGH (official Google web.dev docs)
- [Making PWAs installable — MDN](https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/Guides/Making_PWAs_installable) — HIGH (MDN)
- [Secure contexts — MDN Web Docs](https://developer.mozilla.org/en-US/docs/Web/Security/Defenses/Secure_Contexts) — HIGH (MDN, source for the localhost-only secure-context exception finding)
- [Local-First Architecture for Progressive Web Apps — OpenReplay blog](https://blog.openreplay.com/local-first-pwa-architecture/) — MEDIUM (independent technical blog, source for the offline-first vs. local-first distinction)
- topos internal: `docs/plugin-contract.md` (existing published plugin contract — HIGH, primary source, already establishes "installing a plugin is the same trust decision as installing the kernel" and "go-plugin is a transport, not a sandbox")
- topos internal: `.planning/PROJECT.md` (milestone scope, existing architecture decisions, explicit out-of-scope items) — HIGH, primary source

---
*Feature research for: topos v1.1.0 "Plugin Ecosystem"*
*Researched: 2026-08-12*
