---
phase: 12
slug: filesystem-source
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-08-14
---

# Phase 12 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.
> Register authored at plan time across all 11 plans (12-01..12-11); audited State-B (no prior SECURITY.md) by gsd-security-auditor on 2026-08-14. Verdict: **SECURED — 72/72 closed**.

> **ID hygiene note:** plans 12-04/12-05 and 12-06 independently assigned T-12-20..T-12-24 to different threats. The 12-06 block is renumbered here with an `a` suffix (T-12-20a..T-12-24a) so tooling keying on threat ID cannot collapse them. The two `critical` threats of the phase are in that renumbered block.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| browser → kernel open route | POST /api/items/{id}/open leads to a desktop exec (xdg-open) | item id only; path resolved server-side |
| plugin → kernel renditions | filesystem plugin serves bytes/markdown/plain-text renditions | file content from the configured root |
| source folder → plugin walk | local/NFS/SMB tree walked and indexed | file names, paths, metadata |
| operator config → advisory text | webspace names and match values reach the DOM as interpolated text | operator-authored strings |
| kernel → browser (GET /api/sources) | health, notices, launch-failure state rendered on the chip | kernel-composed status text |
| external plugins dir → kernel | same binary loaded at external tier, pin-verified | binary identity (SHA-256 pin) |

---

## Threat Register

Status: all **closed** (verified 2026-08-14, ASVS L1, block_on high). Evidence cites the mitigation site or pinning test.

| Threat ID | Category | Component | Severity | Disposition | Mitigation (verified evidence) | Status |
|-----------|----------|-----------|----------|-------------|--------------------------------|--------|
| T-12-01 | EoP | fsopen exec surface | critical | mitigate | `fsopen.go:41` fixed-literal `exec.Command("xdg-open", path)`, no shell, server-resolved single arg | closed |
| T-12-02 | Tampering | open-route path resolution | high | mitigate | `fsopen.go:131-135` join from SourceID+root, prefix check → invalid_path; `TestFilesystemOpen_PathEscapeAnswersInvalidPath` | closed |
| T-12-03 | EoP | open route scheme gate | high | mitigate | `fsopen.go:112` `file://` prefix required before resolution; `fsopen_test.go:243` | closed |
| T-12-04 | Info Disclosure | deep-link exposure | low | accept | `stream.go:221` rewrites `file://` to loopback open route; no new disclosure surface | closed |
| T-12-05 | DoS | opener child lifetime | medium | mitigate | `fsopen.go:42-49` Start + background Wait; handler does no other work | closed |
| T-12-06 | Spoofing | plugin-supplied deep_link | high | mitigate | `fsopen.go:131` path recomputed server-side; deep_link used only as scheme marker | closed |
| T-12-SC(01) | Tampering | supply chain (12-01) | high | mitigate | no dependency added by plan | closed |
| T-12-07 | Tampering | rendition MIME surface | medium | mitigate | `item.go:51-58` only text/plain added; nosniff + sandbox CSP on every rendition (`item.go:244,266`) | closed |
| T-12-08 | EoP | markdown HTML rendition | high | mitigate | goldmark defaults (`render.go:24`); kernel bluemonday second layer fails closed (`rendition.go:74,530-533`) | closed |
| T-12-09 | DoS | oversized file fetch | high | mitigate | `fetch.go:179,209` 32 MiB refusal pre-read; 256 KiB text bound with truncation notice | closed |
| T-12-10 | Tampering | malformed operator glob | low | accept | `scope.go:97` named error for malformed pattern | closed |
| T-12-11 | Info Disclosure | glob scope widening | medium | mitigate | `scope.go:59-86` exclude-first, root-relative anchoring; walk never leaves root | closed |
| T-12-SC(02) | Tampering | supply chain (12-02) | high | mitigate | `doublestar/v4 v4.10.0`; blocking-human checkpoint resolved in 12-02-SUMMARY (pkg.go.dev, MIT, ~900 importers) | closed |
| T-12-12 | Info Disclosure | symlink escape during walk | high | mitigate | `walk.go:153-172` symlinked dirs never descended; file targets re-checked vs resolvedRoot | closed |
| T-12-13 | DoS | symlink cycles / unbounded walk | high | mitigate | cycle class removed structurally; `maxWalkItems=25000` named error | closed |
| T-12-14 | Tampering | plugin writes to source | high | mitigate | `readonly_test.go:56 TestPluginIssuesNoWrite` AST scan, 13 write selectors, negative controls | closed |
| T-12-15 | DoS | partial walk results | high | mitigate | ctx cancel/root-read failure abort; `walk.go:213-215` nil on any error, never partial | closed |
| T-12-16 | Repudiation | empty-vs-denied root | medium | mitigate | `plugin.go:222-227` os.ReadDir distinguishes; Match errors on unreadable root | closed |
| T-12-17 | Info Disclosure | dotfile leakage | low | mitigate | dot-segments skipped unless explicit include_glob (`walk.go:140-186`, isHiddenPath) | closed |
| T-12-SC(03) | Tampering | supply chain (12-03) | high | mitigate | no package-manager install in plan commits | closed |
| T-12-18 | Tampering | shadcn registry pull | medium | mitigate | `components.json` registry unchanged; stock checkbox recipe; deps already present | closed |
| T-12-19 | EoP | path field guidance | medium | accept | two-example placeholder, `secret: false` — per acceptance rationale | closed |
| T-12-20 | Info Disclosure | path field masking | low | accept | declared non-secret, never masked (`plugin-fields.ts:237`) | closed |
| T-12-SC(04) | Tampering | supply chain (12-04) | high | mitigate | no package.json dependency added | closed |
| T-12-21 | Spoofing | external-tier fixture honesty | high | mitigate | e2e seeds externalPluginBinaries only; asserts `tier === 'external'` | closed |
| T-12-22 | Tampering | external binary identity | high | mitigate | SHA-256 pin recomputed before exec (`host.go:834-857`); no-pin ≡ mismatch | closed |
| T-12-23 | Info Disclosure | operator paths in docs | medium | mitigate | placeholder `~/Documents/household` everywhere; no machine value committed | closed |
| T-12-24 | Repudiation | docs overstating behavior | medium | mitigate | spot-checked doc claims against source (256 KiB, 25k cap, ReadDir health, symlink root) | closed |
| T-12-SC(05) | Tampering | supply chain (12-05) | high | mitigate | docs-only plan; no install | closed |
| T-12-20a | Tampering (TOCTOU) | Fetch symlink swap | critical | mitigate | `item.go:142-159 resolvePath` EvalSymlinks fail-closed before read; `TestFetch_SymlinkSwappedAfterIndexingIsRefusedBeforeAnyBytesAreServed` | closed |
| T-12-21a | Tampering (TOCTOU) | open-route symlink swap | critical | mitigate | `fsopen.go:146-159` resolve-then-recheck before opener; `TestFilesystemOpen_SymlinkSwappedAfterIndexingAnswersInvalidPathAndNeverOpens` | closed |
| T-12-22a | Repudiation | opener bound to request ctx | high | mitigate | `context.WithoutCancel` + plain exec.Command; `TestNewXDGOpener_ChildIsNotBoundToACallerContext` | closed |
| T-12-23a | Tampering | residual path-based TOCTOU window | medium | accept | residual window documented; no openat/O_NOFOLLOW claimed anywhere | closed |
| T-12-24a | Info Disclosure | path leakage in errors | low | mitigate | fixed string `resolved path escapes source root`, no path interpolated | closed |
| T-12-25 | DoS | opener child accumulation | low | accept | background reap retained; one child per click | closed |
| T-12-26 | DoS | EvalSymlinks failure | low | accept | resolve error is the fail-closed branch | closed |
| T-12-SC(06) | Tampering | supply chain (12-06) | high | mitigate | stdlib only | closed |
| T-12-27 | Info Disclosure | out-of-scope Fetch | high | mitigate | `fetch.go:120-122` !included → NotFound; two pinning tests | closed |
| T-12-28 | Tampering (TOCTOU) | fetch/open path reuse | medium | mitigate | reads via `resolved`; `openForFetch` opens once and stats same handle | closed |
| T-12-29 | Tampering | TOCTOU honesty in docs | medium | accept | "narrows but does not eliminate" present in plugin-contract.md and api.md | closed |
| T-12-30 | Info Disclosure | source_system provenance | low | accept | `source_system: p.root` documented in plugin-contract.md | closed |
| T-12-31 | DoS | unbounded read | low | mitigate | `io.LimitReader(f, maxByteRenditionSize)` | closed |
| T-12-32 | DoS | glob evaluation cost | low | accept | doublestar.Match named error, one scope per call | closed |
| T-12-SC(07) | Tampering | supply chain (12-07) | high | mitigate | no install; doublestar already direct | closed |
| T-12-33 | Info Disclosure | root base-name label | medium | mitigate | label is `filepath.Base(cleanRoot)` only; documented + e2e pinned | closed |
| T-12-34 | Info Disclosure | labels above root | high | mitigate | all labels from cleanRoot/Rel; `TestFolderLabels_NoLabelNamesADirectoryAboveTheConfiguredRoot` | closed |
| T-12-35 | Spoofing | cross-instance label collision | low | accept | per-instance allowlist (ParticipatesIn) unchanged | closed |
| T-12-36 | Tampering | glob semantics creep in labels | medium | mitigate | zero glob/wildcard refs in item.go; single EqualFold at plugin.go:163 | closed |
| T-12-37 | DoS | label dedupe cost | low | accept | path-depth slice under 25k walk cap | closed |
| T-12-SC(08) | Tampering | supply chain (12-08) | high | mitigate | no install | closed |
| T-12-38 | Spoofing | plugin text in kernel notice | high | mitigate | `zeroMatchNotice(webspace, fields)` — no MatchResponse value reachable | closed |
| T-12-39 | Info Disclosure | cross-instance match fields | medium | mitigate | matchFieldsFor returns this instance's fields only (T-05-07 preserved) | closed |
| T-12-40 | DoS | notice growth | medium | mitigate | `maxJoinedNotices = 5`, remainder counted | closed |
| T-12-41 | Tampering | notice vs error conflation | high | mitigate | notice in its own field via FinishSyncRunWithNotice; tests + api.md | closed |
| T-12-42 | DoS | match-value validation rejection | high | mitigate | verified by absence: no glob/metachar validation added to config | closed |
| T-12-43 | DoS | schema migration | low | accept | schemaVersion 3; rebuild transition proved by store test | closed |
| T-12-44 | Info Disclosure | notice rendering in kernel | high | mitigate | JSON field only; browser rule enforced by T-12-45/T-12-53 | closed |
| T-12-SC(09) | Tampering | supply chain (12-09) | high | mitigate | stdlib only | closed |
| T-12-45 | Tampering (XSS) | advisory in chip DOM | high | mitigate | text/title interpolation only; no `{@html}`; structural guard in match-advisory.test.ts | closed |
| T-12-46 | Tampering | advisory outranking errors | high | mitigate | advisory branch last in healthTone chain. Mechanism superseded: original tooltip gate shipped defective (CR-01), replaced by isAdvisoryOnly — see T-12-51 | closed |
| T-12-47 | Repudiation | warning-not-success tone | high | mitigate | tone branch + e2e asserts bg-warning present AND bg-success absent | closed |
| T-12-48 | Spoofing | client steered by notice text | medium | mitigate | keys on emptiness alone; api.md MUST NOT parse rule | closed |
| T-12-49 | DoS | advisory length | low | accept | bounded upstream by maxJoinedNotices | closed |
| T-12-50 | Info Disclosure | notice on loopback UI | low | accept | operator's own config on loopback | closed |
| T-12-SC(10) | Tampering | supply chain (12-10) | high | mitigate | no install | closed |
| T-12-51 | Tampering | tooltip precedence (CR-01) | high | mitigate | gate `advisory !== '' && advisoryOnly`; isAdvisoryOnly re-asks healthTone with notice removed; six-case matrix + e2e Test A | closed |
| T-12-52 | Repudiation | dead advisory branch | high | mitigate | true-case + anti-dead-code coupling test; e2e Test B; trap recorded in docstring | closed |
| T-12-53 | Tampering (XSS) | advisory interpolation (12-11) | medium | mitigate | no new interpolation site; no-raw-HTML assertion retained | closed |
| T-12-54 | Spoofing | notice-content branching | medium | mitigate | branches on emptiness and healthTone structure only | closed |
| T-12-55 | Info Disclosure | fabricated e2e payload | low | accept | synthetic values, browser-side interception only | closed |
| T-12-56 | Tampering | launch-failure last_notice drift | low | mitigate | contract comment at sources.go:218-225; `TestSourcesHandler_LaunchFailedEntryCarriesNoLastNotice` | closed |
| T-12-SC(11) | Tampering | supply chain (12-11) | high | mitigate | no install | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-12-01 | T-12-04 | file:// deep-links rewritten to loopback route; no new disclosure surface | plan 12-01 register | 2026-08-14 |
| AR-12-02 | T-12-10 | malformed operator glob fails with named error; operator-authored input | plan 12-02 register | 2026-08-14 |
| AR-12-03 | T-12-19, T-12-20 | path field is operator-entered, non-secret by declaration | plan 12-04 register | 2026-08-14 |
| AR-12-04 | T-12-23a | residual path-based TOCTOU window documented ("narrows but does not eliminate") | plan 12-06 register | 2026-08-14 |
| AR-12-05 | T-12-25, T-12-26 | opener child bounded per click; resolve failure fails closed | plan 12-06 register | 2026-08-14 |
| AR-12-06 | T-12-29, T-12-30, T-12-32 | TOCTOU honesty documented; provenance root is operator's own config; glob cost bounded | plan 12-07 register | 2026-08-14 |
| AR-12-07 | T-12-35, T-12-37 | per-instance allowlist isolates label collisions; dedupe bounded by walk cap | plan 12-08 register | 2026-08-14 |
| AR-12-08 | T-12-43 | schema rebuild transition proved by test | plan 12-09 register | 2026-08-14 |
| AR-12-09 | T-12-49, T-12-50 | advisory bounded upstream; loopback-only UI | plan 12-10 register | 2026-08-14 |
| AR-12-10 | T-12-55 | e2e payload fully synthetic, never reaches kernel | plan 12-11 register | 2026-08-14 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-14 | 72 | 72 | 0 | gsd-security-auditor (opus, ASVS L1, block_on high) |

**Auditor's non-blocking observations (recorded, no threat open):**
1. Duplicate threat IDs across plans (T-12-20..24 in 12-04/12-05 vs 12-06) — renumbered with `a` suffix in this register.
2. 12-10-SUMMARY's `## Threat Flags` attested a mitigation (the advisory gate) that CR-01 later showed was defective; genuinely closed now via T-12-51/T-12-52. Treat summary self-attestations as claims, not evidence.
3. docs/plugins/filesystem.md omits the 32 MiB rendition cap (documents 256 KiB text bound and 25k walk cap). Understatement only — no threat violated.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-14
