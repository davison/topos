---
phase: 09
slug: ui-polish-and-source-management-rework
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-08-11
---

# Phase 09 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| plugin subprocess → kernel | `DescribeResponse.icon`/`icon_mime` are attacker-influenced bytes from the kernel's perspective (plugin binaries are separately built, potentially third-party) | opaque icon bytes + declared MIME |
| kernel → browser | `GET /api/plugins/{plugin}/icon` serves plugin-supplied bytes into the kernel's own origin; reload failure copy embeds kernel error text into the SPA | icon bytes, error strings |
| browser → kernel | `{plugin}` path segment is untrusted URL input; refresh and `POST /api/config/reload` mutate sync/plugin state | URL path input, state-mutating POSTs |
| upstream project → this repo | Third-party SVG assets (paperless-ngx, SilverBullet, Lucide) copied into the repo and compiled into shipped binaries | SVG source, licenses |
| network → kernel HTTP surface | Instance may be LAN/tailnet-reachable (`make dev` exposes Vite dev server) | robots.txt, favicon |
| stored config → browser UI | `base_url`/`path` rendered verbatim in the source picker | operator-supplied config values |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-09-01 | Elevation of Privilege (stored XSS) | `kernel/httpapi/pluginicon.go` | high | mitigate | Icon route sets `Content-Security-Policy: default-src 'none'; style-src 'unsafe-inline'; sandbox`, `X-Content-Type-Options: nosniff`, `Content-Disposition: inline` (pluginicon.go:78-88) | closed |
| T-09-02 | Spoofing / Tampering | `kernel/pluginhost/host.go` icon capture | high | mitigate | Kernel-side MIME allowlist of exactly `image/svg+xml` and `image/png`; anything else drops the icon (host.go:54-55) | closed |
| T-09-03 | Denial of Service | `kernel/pluginhost/host.go` icon capture | medium | mitigate | `MaxIconBytes = 65536` enforced at capture; oversized icons dropped, never truncated (host.go:46,69) | closed |
| T-09-04 | Information Disclosure | `PluginIconHandler` path param | medium | mitigate | Path separators and `..` rejected before lookup; exact-match over in-memory map, no filesystem access (pluginicon.go:49) | closed |
| T-09-05 | Tampering (supply chain) | `plugins/mock/assets/icon.svg` | low | mitigate | Four-key provenance comment (Source-Project/File/Version/License) above embed; Lucide is ISC | closed |
| T-09-06 | Repudiation | icon caching | low | accept | See Accepted Risks Log AR-09-01 | closed |
| T-09-07 | Tampering (supply chain) | `plugins/*/assets/icon.svg` | medium | mitigate | All six assets carry provenance keys; `internal/audit` `plugin_icons_test.go` fails the build if any key is missing (test passing); assets reviewed as plain text in commit diff; no package-manager installs | closed |
| T-09-08 | Elevation of Privilege (stored XSS) | `plugins/*/assets/icon.svg` | high | mitigate | Two layers: kernel CSP/sandbox/nosniff on icon route (T-09-01) and frontend renders icons only via `<img>` — no `{@html}` anywhere in `web/src` | closed |
| T-09-09 | Information Disclosure | `plugins/*/assets/icon.svg` | low | mitigate | Assets copied from upstream (not locally generated) and reviewed as text before commit | closed |
| T-09-10 | Spoofing (brand) | WhatsApp plugin identity | low | mitigate | Generic Lucide message-bubble glyph used (`plugins/whatsapp/assets/icon.svg`, Source-Project: @lucide/svelte); confirmed at UAT test 1 | closed |
| T-09-11 | Information Disclosure | `web/static/robots.txt` | medium | mitigate | Shipped file is `User-agent: * / Disallow: /`; defence in depth alongside the loopback-only default, not a substitute for it | closed |
| T-09-12 | Information Disclosure | favicon asset | low | accept | See Accepted Risks Log AR-09-02 | closed |
| T-09-13 | Tampering | `web/src/app.css` palette | low | accept | See Accepted Risks Log AR-09-03 | closed |
| T-09-14 | Elevation of Privilege | `plugins/mock` rendition fixture | medium | mitigate | Fixture gated behind env var (`renditionFixtureEnabled(os.Getenv)`, `plugins/mock/main.go:64`), unset in normal launches; emits static embedded PNG; `allowedRenditionTypes` not widened | closed |
| T-09-15 | Tampering | `plugins/mock/plugin.go` shared with 09-01 | low | mitigate | Fixture-off byte-identical behaviour covered by `renditionfixture_test.go` | closed |
| T-09-16 | Denial of Service | media box aspect lock | low | accept | See Accepted Risks Log AR-09-04 | closed |
| T-09-17 | Denial of Service | chip refresh trigger | low | mitigate | Menu refresh item `disabled={source.syncing}` (`SourceChip.svelte:205`) — stricter than the retired standalone button | closed |
| T-09-18 | Information Disclosure | chip tooltip | low | accept | See Accepted Risks Log AR-09-05 | closed |
| T-09-19 | Tampering | shared file with 09-01 | low | mitigate | 09-01 identity icon asserted present before 09-05 edits; icons confirmed rendering at UAT test 1 | closed |
| T-09-20 | Denial of Service | `Reload config` menu item | medium | mitigate | Double guard, both citing T-09-20 in-source: menu item `disabled={reloadBusy}` (`WebspaceSwitcher.svelte:93`) and route-level early return `if (reloadBusy) return;` (`w/[webspace]/+page.svelte:309`) | closed |
| T-09-21 | Information Disclosure | reload failure Alert | medium | mitigate | Kernel error rendered through Svelte default text binding only (no `{@html}` in `web/src`); same string previously shown in the modal; loopback-only, single-user | closed |
| T-09-22 | Elevation of Privilege | route owns the reload call | low | accept | See Accepted Risks Log AR-09-06 | closed |
| T-09-23 | Tampering | superseded D-13 assertion | low | mitigate | Count assertion updated in place with comments naming the superseding authority (`webspace-switcher.test.ts:9,105`, `WebspaceSwitcher.svelte:6`) | closed |
| T-09-24 | Information Disclosure | instance row location line | low | accept | See Accepted Risks Log AR-09-07 | closed |
| T-09-25 | Elevation of Privilege | location line rendering | low | mitigate | `base_url`/`path` rendered via Svelte text binding and native `title` attribute — markup escaped by construction | closed |
| T-09-26 | Spoofing | catalog tiles | low | mitigate | Catalog list sourced solely from kernel `GET /api/config/plugin-types` (`api.ts:427`); no client-side list, no duplicated exclusion rule | closed |
| T-09-27 | Denial of Service | picker scroll height | low | accept | See Accepted Risks Log AR-09-08 | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-09-01 | T-09-06 | Stale cached icon after plugin rebuild under immutable `Cache-Control`; icon bytes static per build, ETag allows revalidation, hard reload resolves — no security consequence | plan 09-01 threat model | 2026-08-11 |
| AR-09-02 | T-09-12 | `/app-icon.png` fetchable by anything reaching the instance; project logo, no user data | plan 09-03 threat model | 2026-08-11 |
| AR-09-03 | T-09-13 | Palette value change crosses no trust boundary; luminance-ordering test guards regression | plan 09-03 threat model | 2026-08-11 |
| AR-09-04 | T-09-16 | Aspect-locked box bounds display size only, not transfer; unchanged from shipped behaviour | plan 09-04 threat model | 2026-08-11 |
| AR-09-05 | T-09-18 | Tooltip renders `source.last_error` verbatim — unchanged from shipped chip behaviour | plan 09-05 threat model | 2026-08-11 |
| AR-09-06 | T-09-22 | Reload ownership moves modal → route; same origin, same loopback context, no boundary crossed | plan 09-06 threat model | 2026-08-11 |
| AR-09-07 | T-09-24 | `base_url`/`path` are non-secret operator-owned values; constraint: no secret-marked field may ever join this line (`token` never rendered) | plan 09-07 threat model | 2026-08-11 |
| AR-09-08 | T-09-27 | Two-section picker reduces visible rows; adequate at six plugin types — revisit with per-group max height if catalog grows well past that | plan 09-07 threat model | 2026-08-11 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-11 | 27 | 27 | 0 | gsd-secure-phase (L1 short-circuit — plan-authored register, all mitigations verified in implementation) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-11
