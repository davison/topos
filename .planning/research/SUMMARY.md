# Project Research Summary

**Project:** topos v1.1.0 "Plugin Ecosystem"  
**Domain:** Local-first personal data aggregator with external plugin loading, filesystem/cloud sources, per-item curation, and PWA installability  
**Researched:** 2026-08-12  
**Confidence:** HIGH (all recommendations grounded in codebase analysis, official API docs, and this project's own architecture precedents)

## Executive Summary

Topos v1.1.0 extends the existing Go-kernel + gRPC-subprocess architecture to support external, untrusted plugins while adding three new sources (filesystem, Google Drive) and per-item manual override. The research confirms the approach is architecturally sound but carries real security and UX precision requirements that must be designed and tested carefully.

**Core finding:** Trust marking must be kernel-derived from file provenance (which directory the binary was found in), never from anything the plugin declares about itself. A boolean "trusted/untrusted" flag is defensible for v1.1.0 only if paired with a content-hash re-verification at every launch to prevent TOCTOU attacks.

**Architectural finding:** Five required features share zero hard-sequencing constraints except: external-loading phase must complete before GDrive can be meaningfully dogfooded. Everything else can run in parallel. Filesystem plugin (in-repo, trusted) should validate the external mechanism before GDrive's larger scope begins.

**Risk finding:** Per-item manual include of an unmatched item is architecturally impossible under the current plugin contract (no "browse unmatched items" RPC). Must be scoped as either "include only reverses prior exclude" or "commit to a new contract RPC." Currently unresolved; must be a hard design decision before Phase 2b starts.

## Key Findings

### Recommended Stack

**No new language/framework additions — all v1.1.0 work is additive.**

Core dependencies:
- **Google Drive API** (`google.golang.org/api/drive/v3@v0.292.0`): Official Google client
- **OAuth2 loopback** (`golang.org/x/oauth2@v0.36.0`): Desktop installed-app flow
- **Keyring storage** (`github.com/zalando/go-keyring@v0.2.8`): Secret Service (same as Signal precedent)
- **Binary provenance** (`debug/buildinfo` stdlib): Reads VCS metadata from any binary
- **PWA tooling** (`vite-plugin-pwa@1.3.0`, `@vite-pwa/sveltekit@1.1.0`): Generates SW via existing go:embed
- **Filesystem watching** (`github.com/fsnotify/fsnotify@v1.10.1`): Local-mount accelerator only; mandatory `filepath.WalkDir` for network mounts

**What NOT to use:** goreleaser/cosign (premature for v1.1.0); fsnotify alone (broken on NFS/SMB); radovskyb/watcher (redundant).

### Expected Features

All five P1 (table stakes):
- **Trust marking:** Warning + persistent badge + kernel-derived provenance + content-hash verification
- **Filesystem plugin:** Subfolders, document types, polling + optional fsnotify
- **Google Drive plugin:** Incremental sync, Workspace-doc export, OAuth refresh-token rotation
- **Per-item marks:** Exclude/include toggle, survives rebuilds, always beats match rules
- **PWA install:** Desktop/localhost true install; mobile/LAN requires user's own HTTPS

**Out of scope:** Sandboxing, full offline caching, marketplace, write-back

**Unresolved:** Include-of-unmatched-items scope (new RPC vs. un-exclude-only).

### Architecture Approach

**Trust:** Directory-tier provenance (`[plugins] dir` = trusted, `[plugins] external_dir` = untrusted) + content-hash re-verification at every launch

**Marks:** Separate `item_marks` table (no FK to items, exempt from rebuild-drop list) — survives both resyncs and schema rebuilds

**Filesystem plugin:** In-repo trusted, fits Signal's local-path pattern. Stat-diff polling (load-bearing) + optional fsnotify acceleration for local paths.

**Google Drive:** Separate repository, dogfoods external-plugin contract. Needs `config.Source.Extra` map prerequisite for arbitrary config keys.

**PWA:** Pure web/ change; kernel adds one MIME-type line. Design decision: desktop-only vs. add kernel HTTPS for mobile LAN.

### Critical Pitfalls (Top 5)

1. **TOCTOU without re-verification** — Hash-pin binary at trust-time; re-verify at every launch or swapped binary inherits stale trust
2. **Sync-replace wipes marks** — Must be separate table joined at read-time, never touched by `ReplaceWebspaceSourceItems`
3. **Include of unmatched item impossible** — Scope decision: "un-exclude only" (no contract change) or new Browse RPC (bigger scope)
4. **Marks orphaned on rename/move** — Use immutable stable IDs (inode/Drive file ID); add orphan-detection sweep
5. **ServiceWorker fails on LAN without HTTPS** — Secure-context requirement blocks mobile install on plain HTTP LAN IP; scope v1.1.0 to desktop only

## Implications for Roadmap

### Phase 1: External Plugin Loading + Trust Marking

**Rationale:** Only hard dependency; blocks GDrive. Validates filesystem plugin mechanism.

**Delivers:** `external_dir` config + discovery, content-hash verification at every launch, trust badge UI, `config.Source.Extra` map (prerequisite for GDrive), launch timeout on trial-launch, path canonicalization

**Avoids:** Pitfalls 1, 9, 11 (TOCTOU, timeout hang, path confusion)

**Research flags:** None — standard security code review

---

### Phase 2a, 2b, 2c: Parallel Independent Phases

**Phase 2a: Filesystem Plugin**
- **Rationale:** Validates local-path pattern (like Signal); independent of external loading
- **Delivers:** Folder watching (polling + optional fsnotify), subfolders, document types, stable-ID choice
- **Research flags:** Medium — if fsnotify acceleration in-scope, needs integration test on actual NFS/SMB mounts

**Phase 2b: Per-Item Marks**
- **Rationale:** Fully independent; must resolve include-scope in spec step
- **Delivers:** `item_marks` schema (rebuild-exempt), exclude/include endpoints, UI toggle, resync-survival test
- **Research flags:** High — Pitfall 3 scope decision must precede schema work

**Phase 2c: PWA Installability**
- **Rationale:** Fully independent; low technical risk but scope decision needed
- **Delivers:** Manifest, ServiceWorker, MIME-type registration, explicit design decision on mobile/LAN scope
- **Research flags:** Medium — secure-context gap must be explicit design decision before implementation

---

### Phase 3: Google Drive Plugin

**Rationale:** Hard-depends on Phase 1. **CRITICAL GATE:** External-loading phase must be validated end-to-end against at least one real out-of-repo binary (filesystem plugin is the validation vehicle) before GDrive's larger OAuth/API work begins.

**Delivers:** Separate repo, OAuth loopback-redirect + zalando/go-keyring token rotation, Drive API (folder-scoped, incremental sync via changes.list, Workspace-doc export), credential-distribution model decision (shared client vs. bring-your-own)

**Avoids:** Pitfalls 10, Integration Gotchas (export/mime-type, quota)

**Research flags:** High — credential distribution (shared verification burden + 7-day testing-mode re-auth vs. bring-your-own setup friction) needs decision spike before phase plan

### Phase Ordering Rationale

- **Hard constraint:** Phase 1 → Phase 3 only
- **Parallelizable:** 2a, 2b, 2c fully independent
- **Checkpoint:** Phase 1 validated against filesystem-plugin binary before Phase 3 substantial work
- **Why:** Avoids Pitfall 12 (circular dependency); filesystem proves external mechanism before GDrive complexity; marks scope explicit before schema; PWA scope explicit before implementation

### Research Flags

**Phases needing research during `/gsd-plan-phase`:**
- **Phase 1:** Medium — binary hash verification + security review
- **Phase 2a:** Medium→High — network-mount integration testing if fsnotify acceleration in-scope
- **Phase 2b:** High — Pitfall 3 scope decision (unmatched include = new RPC?)
- **Phase 2c:** Medium — LAN/mobile secure-context decision must be explicit in phase plan
- **Phase 3:** High — credential distribution model spike before phase plan

**Standard patterns (skip research):**
- Trust marking grounded in Signal precedent; security review sufficient
- Schema pattern mirrors webspace_items exactly
- PWA: standard vite-plugin-pwa integration

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Versions cross-checked against go.mod/package.json; no conflicts |
| Features | HIGH | Drawn from PROJECT.md; reference systems (VS Code, HACS, Gmail) validate |
| Architecture | HIGH | Grounded in direct codebase analysis (pluginhost, index, config, SDK) |
| Pitfalls | HIGH | Either confirmed from repo patterns (trust, sync-replace, rebuilds) or well-documented upstream (fsnotify on NFS, secure-context requirement) |

**Overall: HIGH** — No architectural blockers, no technology surprises. Primary risk is **design precision** (trust hash-pinning, mark scope, GDrive credential distribution, PWA LAN scope), not technology selection.

## Gaps to Address

Must be resolved during phase planning, not implementation:

1. **Per-item include scope** (Pitfall 3) — Phase 2b spec/discuss: "un-exclude only" vs. "pull unmatched items" (needs new Browse RPC)
2. **Filesystem `source_id` choice** (Pitfall 4) — Phase 2a: inode-based vs. path-based identity (tradeoff: rename stability vs. mount-boundary breakage)
3. **GDrive credential distribution** (Pitfall 10) — Phase 3: shared client (Google verification burden, 7-day testing re-auth) vs. bring-your-own (user setup friction, matches project's credential pattern)
4. **PWA mobile scope** (Pitfall 7) — Phase 2c: desktop/localhost only (documented limitation) vs. add kernel HTTPS for mobile LAN install
5. **Mark orphan handling** (Pitfall 5) — Phase 2b: cascade-delete on item removal (simpler, loses transient data) vs. persistent-orphans-with-sweep (preserves data through temporary unavailability)

## Sources

### Primary (HIGH)
- `/home/darren/projects/davison/topos/.planning/PROJECT.md` — v1.1.0 scope
- `/home/darren/projects/davison/topos/docs/plugin-contract.md` — published contract
- Codebase: `kernel/pluginhost/`, `kernel/index/`, `kernel/config/`, `cmd/topos/main.go`, `sdk/`, `internal/audit/`
- Official: Google Drive API v3 docs, OAuth 2.0 installed-app flow, MDN PWA/ServiceWorker/Secure-Contexts

### Secondary (MEDIUM-HIGH)
- pkg.go.dev: fsnotify, zalando/go-keyring, debug/buildinfo
- LWN.net + fsnotify upstream: NFS/SMB/CIFS notification gaps (kernel-level limitation)
- Google Cloud + community (Nango, Unipile, CData): 7-day testing-mode refresh-token expiry

### Tertiary (MEDIUM)
- Reference systems: VS Code Marketplace, Obsidian plugins, HACS, Gmail filters

---

*Research completed: 2026-08-12*  
*Synthesized by: gsd-research-synthesizer*  
*Ready for roadmap: YES*
