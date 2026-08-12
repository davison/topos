---
phase: 01
slug: first-webspace-end-to-end
status: verified
threats_open: 0
asvs_level: 1
created: 2026-07-28
---

# Phase 01 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| browser → kernel HTTP API | Loopback-only, unauthenticated; any local process can reach it | Webspace/item metadata, rendition bytes |
| kernel → plugin subprocess | gRPC over go-plugin handshake; kernel injects source config into plugin env | Source config incl. API token |
| plugin → paperless-ngx REST API | LAN hop carrying a bearer token; all response fields untrusted | Token (outbound), documents/tags (inbound) |
| config.toml + environment → kernel | User-authored file plus env-sourced secrets crossing into parsing and SQL | `${VAR}` secret references |
| paperless response → SQLite → JSON → DOM | Attacker-influenceable content (titles, OCR text, tags) flowing to a rendering surface | Untrusted display text |
| paperless response → plugin control flow | Redirect Location headers and pagination `next` URLs steer the next outbound request | Attacker-influenceable URLs |
| npm/Go registries → build | Installed dependency code ships in the kernel binary and SPA | Third-party code |
| build output → go:embed → browser | Vite output embedded verbatim and served | Static assets |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-01-01 | Info Disclosure | kernel HTTP listener | high | mitigate | Default bind `127.0.0.1:7777` (`kernel/config/types.go`); non-loopback listen logs explicit warning (`cmd/webspaces/main.go:172`) | closed |
| T-01-02 | Info Disclosure | token in plugin logs | high | mitigate | No log calls in `plugins/paperless/client.go`; token held in struct field, never re-serialised | closed |
| T-01-03 | Info Disclosure | secrets in config.toml | high | mitigate | `os.Expand` + `LookupEnv` (`kernel/config/config.go:52`); `config.example.toml` ships `${VAR}` references only | closed |
| T-01-04 | Tampering | SQL construction in `kernel/index/store.go` | high | mitigate | All queries parameterized (`database/sql` placeholders); no Sprintf/concat SQL found | closed |
| T-01-05 | Tampering | untrusted fields rendered in SPA | medium | mitigate | Svelte auto-escaping; zero `{@html}` in web/src (grep-confirmed) | closed |
| T-01-06 | EoP | plugin capability beyond contract | high | mitigate | `plugin.proto` declares only Describe/Match/Fetch/Health; `sdk/contract_test.go` allowlist pins the RPC set | closed |
| T-01-07 | Spoofing | arbitrary binary launched as plugin | medium | mitigate | go-plugin MagicCookie handshake (`sdk/shared.go:22`); kernel calls Describe for source_type | closed |
| T-01-08 | DoS | unbounded pagination / hung LAN request | medium | accept | 30s per-request timeout + 4-connection bound; full coordinator protection is KERN-04 (Phase 2) | closed |
| T-01-09 | Repudiation | no sync run record | low | mitigate | `sync_runs` table (`kernel/index/schema.go:53`) records start/finish/status/error/count | closed |
| T-01-10 | EoP | `/api/items/{id}/content` same-origin bytes | high | mitigate | MIME allowlist before body write (415 otherwise); `nosniff`, `Content-Disposition: inline`, sandboxing CSP (`kernel/httpapi/item.go:176-178`) | closed |
| T-01-11 | EoP | source-controlled text in detail pane | high | mitigate | Svelte escaping; no `{@html}` (grep-confirmed) | closed |
| T-01-12 | Info Disclosure | fetch error text leaking internals | medium | mitigate | Kernel maps to fixed `source_unavailable` code (`kernel/httpapi/item.go:194`); contract test asserts envelope | closed |
| T-01-13 | DoS | large rendition buffered in memory | medium | mitigate | Streamed via `io.Copy` (`kernel/httpapi/item.go:181`); no full-rendition buffering | closed |
| T-01-14 | Tampering | empty/attacker-chosen deep link indexed | medium | mitigate | Correlation rejects unspecified fidelity / empty deep link (`kernel/correlate/correlate.go:140-143`) | closed |
| T-01-15 | Info Disclosure | rendition bytes cached | low | mitigate | `Cache-Control: private, no-store` on content/thumbnail responses (asserted in `item_test.go:212`) | closed |
| T-01-16 | EoP | stream row title/snippet/tags | high | mitigate | Svelte escaping; tags render as Badge text content; no `{@html}` | closed |
| T-01-17 | Info Disclosure | `sync.error` rendered in failure state | medium | mitigate | Failure class + host/status only (no token per T-01-02); rendered as escaped text | closed |
| T-01-18 | EoP | thumbnail bytes in `img` | low | mitigate | Served by kernel thumbnail route under the same MIME allowlist/nosniff/CSP; rendered as `img` only | closed |
| T-01-19 | DoS | unbounded stream list in browser | low | accept | Personal-scale document counts; constant row height; virtualization deferred by UI-SPEC | closed |
| T-01-20 | EoP | future mutating RPC in plugin.proto | high | mitigate | `sdk/contract_test.go` RPC allowlist fails build on any addition, names PLUG-02 | closed |
| T-01-21 | EoP | future non-GET HTTP call in a plugin | high | mitigate | `plugins/paperless/readonly_test.go` AST walk fails on non-GET/HEAD request construction | closed |
| T-01-22 | Info Disclosure | literal token in example config/README | high | mitigate | Both files carry `${VAR}` references / placeholder exports only (grep-confirmed) | closed |
| T-01-23 | Spoofing | docs implying LAN exposure is safe | medium | mitigate | `docs/api.md` states loopback-only default explicitly ("Loopback-only default, no auth" section) | closed |
| T-01-24 | Repudiation | contract tests silently skipping | medium | mitigate | All tests run against `httptest` + temp SQLite; no network dependency | closed |
| T-01-05-01 | Tampering | stale listener on :7777 forging verification | medium | mitigate | `scripts/e2e-smoke.sh:37` FAILs when port already occupied | closed |
| T-01-05-02 | Info Disclosure | PAPERLESS_TOKEN in command env | high | mitigate | Env confined to `bash -c` wrappers sourcing gitignored `.env`; no task echoes or prints headers | closed |
| T-01-05-03 | Info Disclosure | emitted stylesheet content | low | accept | app.css holds only design tokens/layout rules — no secrets or user content; shipping to local browser is intended | closed |
| T-01-05-04 | Tampering | embedded build output | medium | mitigate | Build output gitignored, regenerated from source; smoke test asserts served CSS carries `#020617` token (`e2e-smoke.sh:82`) | closed |
| T-01-06-01 | Info Disclosure | redirect handling in paperless client | high | mitigate | `CheckRedirect` refuses non-base, non-loopback hops (`ErrForeignHost`, `client.go:34`); `errors.Is` test passes | closed |
| T-01-06-02 | Spoofing | Authorization token on cross-host hop | high | mitigate | Foreign hop refused before issue; token cannot reach a foreign host; never logged (D-04) | closed |
| T-01-06-03 | Tampering | pagination `next` URL | medium | mitigate | `splitNextURL` re-pins to configured base (committed foreign-next test); `DialContext` guard as backstop | closed |
| T-01-06-04 | EoP | future outbound call site anywhere in repo | medium | mitigate | `internal/audit` AST test fails build on outbound HTTP outside `plugins/paperless/client.go` or foreign URL literals | closed |
| T-01-06-05 | DoS | custom CheckRedirect removing redirect cap | low | mitigate | 10-hop cap re-implemented explicitly; asserted by redirect-loop test | closed |
| T-01-06-06 | Repudiation | audit test passing vacuously | medium | mitigate | `internal/audit/testdata` negative-control fixture fails suite unless both offense kinds reported | closed |
| T-01-SC | Tampering | go get / npm install supply chain (all plans) | high | mitigate | All packages `[VERIFIED]` in 01-RESEARCH.md Package Legitimacy Audit (2026-07-27); versions pinned; plans 01-02…01-06 introduced zero new dependencies | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-01-1 | T-01-08 | Sync stall degrades freshness only; stream still serves from index. 30s timeout + 4-conn bound cap blast radius. Coordinator-level protection scheduled as KERN-04 (Phase 2). | plan 01-01 (plan-time disposition) | 2026-07-28 |
| AR-01-2 | T-01-19 | Personal-scale document counts; constant row height keeps layout cost linear; virtualization deferred until real volume demands it (UI-SPEC). | plan 01-03 (plan-time disposition) | 2026-07-28 |
| AR-01-3 | T-01-05-03 | Stylesheet contains only design tokens and layout rules — no secrets or user content; serving it locally is the product behavior. | plan 01-05 (plan-time disposition) | 2026-07-28 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-07-28 | 33 | 33 | 0 | gsd-secure-phase (L1 grep-depth, plan-time register short-circuit) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-07-28
