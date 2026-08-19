# Phase 16: Provenance-Based Plugin Trust - Context

**Gathered:** 2026-08-19
**Status:** Ready for planning

<domain>
## Phase Boundary

The kernel decides a plugin's trust tier from verifiable provenance carried by
the artifact itself — an ed25519 signature over a release manifest — so a
first-party `topos-plugins` binary earns trust wherever it lives on disk, and
no config edit, file drop, or shadow binary can forge it. The unsigned
external path (consent interstitial, content pin, untrusted badge, two-click
re-pin) is explicitly unchanged (TRUST-03). This phase lands **before** the
Phase 17 repo split, while the in-repo plugins and their link-time build
manifest still exist and must keep working.

Out of scope: moving any plugin out of the repo (Phase 17), pull-by-URL
install (Phase 18), any new UI surface (existing chip/badge/interstitial
surfaces only — but any UI-visible change to tier surfacing extends the
Playwright e2e suite per the standing 07.1 D-11 rule).

</domain>

<decisions>
## Implementation Decisions

### Provenance mechanism
- **D-01:** Trust derives from an **ed25519 signature over a release manifest**, verified against a public key embedded in the kernel at build time. Pure in-kernel verification via `golang.org/x/crypto` — no sigstore/cosign, no GitHub attestation machinery, no network at verify time. Chosen for fully-offline verification, near-zero new dependencies, and fit with the local-first ethos. — **Reversibility:** costly — the signing workflow, release layout, and kernel verifier all encode the scheme; swapping to sigstore later means re-cutting releases and a kernel release, though the tier model above it would survive.
- **D-02:** The private key is generated once and held as a **GitHub Actions secret in the `topos-plugins` repo**; the tag-triggered release workflow signs the manifest automatically. Trust boundary = "whoever can push a tag to topos-plugins + GitHub's secret store". No manual signing step in the release flow.
- **D-03:** The kernel embeds an **accepted-key SET** (initially one key), each with a key ID; signatures name the key that made them. Rotation = ship a new kernel release adding the new key; old releases stay verifiable during overlap; retired keys can be dropped later.
- **D-04:** TRUST-02's proving artifact comes from **standing up a minimal `topos-plugins` sibling repo in this phase** — just the release/signing workflow and a trivial/mock plugin — and cutting one real signed release that an installed kernel verifies as trusted with no link-time manifest entry. Phase 17 then fills the repo it already trusts; the signing path is proven once, in its final home. — **Reversibility:** one-way — the repo, its key, and its first release tag become public artifacts Phase 17 builds on; abandoning them means re-keying and re-proving the path.

### Provenance format & travel
- **D-05:** The signed unit is **one release manifest** listing every plugin's name→SHA-256, with a single ed25519 signature over the manifest. Names are bound to hashes, so a validly-signed binary renamed to shadow another plugin name does not verify — signature proves origin AND identity. No per-binary signatures.
- **D-06:** Manifest entries carry **plugin version and the gRPC contract generation** the binary was built against (e.g. `topos.v2`); the manifest carries release tag and platform/arch. This directly feeds Phase 17's DIST-03 (mismatch fails by name) — the exact field set is Claude's to settle in planning with this as the baseline.
- **D-07:** The signed manifest and its signature **live alongside the binaries in the plugins directory**. Provenance is self-contained — copying a plugin dir copies its evidence; directory writability is irrelevant because forging requires the private key. No separate kernel-owned provenance store.
- **D-08:** **Multiple release manifests coexist** (versioned filenames): a binary is trusted if ANY validly-signed manifest names it with a matching hash. This is what lets plugins upgrade independently (DIST-02) without re-placing the whole set. Verify logic scans all manifests present.

### Verification timing & manifest coexistence
- **D-09:** Verification runs at **install time AND every launch**. Install verifies before placing (Phase 15's stage→verify→place discipline); the kernel re-verifies at each plugin launch exactly like today's link-time gate, catching post-install tampering. The per-launch hashing cost is already paid today.
- **D-10:** During the transition, **either arm grants TierTrusted**: a binary verifies via the link-time build manifest OR via a valid signed release manifest. In-repo `make build`/`make dev` flows keep working untouched. Phase 17 deletes the link-time arm, leaving signed provenance as the only trusted path. No precedence ordering — both arms are peers.
- **D-11:** The trusted/external directories become **pure search paths**. Tier is computed per binary from provenance alone; anything unverified gets external-tier semantics (consent, pin, badge) regardless of which directory it sits in. Editing `plugins.dir` escalates nothing; a file drop earns nothing; name-shadowing dies because location no longer confers trust. This is success criterion 1 verbatim, and TRUST-04's committed tests assert each closed path. — **Reversibility:** costly — discover/resolve logic, tier_test/manifestgate_test suites, and e2e fixtures all currently encode directory-derived tiers; re-introducing location semantics later would reopen the exact attack paths this phase closes by test.
- **D-12:** The post-split dev loop (unsigned local builds loading into the dev kernel) is **noted as a design constraint, decided in Phase 17** (REPO-05). Phase 16 must keep the manifest-injection seam cleanly factored so "dev kernel trusts its own local builds" remains possible — but builds no dev mechanism now.

### Claude's Discretion
- **Failure behavior & operator visibility** (the unselected fourth area): how a verification failure surfaces on the source chip and in logs, within the locked conventions — fail loudly by name, verification never demotes-and-runs (D-13 in `kernel/pluginhost/manifest.go`), trust-nothing on missing provenance (PD-04), no silent downgrade of a plugin the operator believes is trusted (success criterion 5). Any UI-visible change extends the e2e suite.
- Exact manifest file format (JSON vs extending the existing `name=hexdigest` text form), manifest filename/versioning scheme, and stale-manifest cleanup.
- Whether coexistence (D-10) is structured as two verifiers behind one interface or one verifier with two evidence sources.
- TRUST-04 test placement and shape, provided each escalation path (config edit, file drop, shadowing) has a committed test that fails if its gate is removed.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### The standing security todo (this phase's origin)
- `.planning/todos/completed/2026-08-13-plugin-trust-tier-is-directory-location-not-provenance.md` — Names the three escalation paths TRUST-04 must close by test (config edit, file drop, shadowing/D-11), the severity framing (consent layer, not privilege boundary), and the candidate direction this phase now supersedes with signatures.

### Requirements & roadmap
- `.planning/ROADMAP.md` — Phase 16 section: goal, five success criteria, notes (discuss-mandated trust model — now settled here; lands before the move; link-time path keeps working until Phase 17).
- `.planning/REQUIREMENTS.md` — TRUST-01..04 exact wording.

### Existing trust machinery (the code being extended/superseded)
- `kernel/pluginhost/manifest.go` — The link-time build manifest (D-12 link-time-only discipline, PD-04 trust-nothing-on-empty, D-13 never-demote-and-run, `VerifyTrustedBinary`, `FormatManifest`/`ParseManifest`, the TEST-ONLY override seam). The new verifier coexists with this until Phase 17.
- `kernel/pluginhost/discover_binaries.go` — Tier derivation, `DiscoverAllTiered`, `ResolveBinary`, the D-11 shadowing handling that D-11 (this phase) makes obsolete.
- `kernel/config/types.go` — `PluginsConfig` (Dir/ExternalDir/Pins) and the "trust is derived purely from WHICH directory" comment this phase rewrites.
- `kernel/pluginhost/binaryhash.go` — `HashBinary`, the ONE hashing convention; the new verifier reuses it, never adds a second implementation.
- `Makefile` — `MANIFEST_GEN_*` recipes and `cmd/topos-manifest`: how the link-time manifest is generated today; the release-side manifest generator should share shape with this.

### Install discipline (Phase 15)
- `scripts/install.sh` / `scripts/install-smoke.sh` — the stage→verify→place flow install-time verification (D-09) extends.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `HashBinary` (`kernel/pluginhost/binaryhash.go`): the single hashing convention — manifest verification reuses it.
- `FormatManifest`/`ParseManifest` (`kernel/pluginhost/manifest.go`): the name→hexdigest manifest shape; the signed release manifest can extend this vocabulary (plus version/contract/platform fields per D-06).
- `OverrideBuildManifest` test seam: pattern to mirror for injecting signed-manifest fixtures in tests; keeping this seam clean is D-12's constraint.
- `cmd/topos-manifest`: existing generator that hashes binaries into manifest form — candidate to grow a signing/release mode or to pattern the topos-plugins release tooling after.
- External-tier machinery (pins, consent, badge, `manifestgate_test.go`/`tier_test.go`/`pin_test.go`): TRUST-03 requires this path unchanged; its tests are the regression net.

### Established Patterns
- Fail-loudly-by-name: every verification failure names the binary and the cause (see `ErrManifestUnverified` wrapping); no silent fallback, no demote-and-run.
- Link-time-only trust data (D-12): trust inputs are never read from files the user can edit at runtime — the signed manifest is the deliberate, cryptographically-gated exception, safe because forging needs the private key.
- e2e fixture symlinking (`web/e2e/fixtures/plugin-binaries.ts`) verifies through the same manifest path as real builds — signed-manifest fixtures should follow suit.

### Integration Points
- `host.go` launch path calls `VerifyTrustedBinary` — the "either arm grants trusted" (D-10) decision lands here.
- Tier computation in `discover_binaries.go` becomes provenance-driven (D-11) — the largest structural change.
- Chip health / badge surfaces in the web UI display tier and failure causes — asserted by existing Playwright specs; extend, don't redesign.

</code_context>

<specifics>
## Specific Ideas

- The minimal `topos-plugins` repo stood up for TRUST-02 (D-04) is deliberately skeletal: signing workflow + one trivial/mock plugin + one tagged release. It is the seed Phase 17 fills, not a parallel effort.
- Trust boundary statement worth preserving in docs: "first-party = signed by a key in the kernel's embedded key set; the key lives in topos-plugins CI; everything else is external-tier by construction."

</specifics>

<deferred>
## Deferred Ideas

- **Post-split dev-loop trust mechanism** (unsigned local builds into the dev kernel) — Phase 17, REPO-05. Phase 16 only keeps the seam factored (D-12).
- **Retiring the link-time manifest arm** — Phase 17 deletes it after the move.

### Reviewed Todos (not folded)
- *Abstract OAuth connectivity and secrets management into the kernel for all plugins* (2026-08-17) — kernel plugin-services feature, unrelated to provenance verification; stays in backlog.
- *Signal schema-version verify-and-accept tooling* (2026-08-05) — Signal-plugin operational tooling, out of scope.
- *Popover clone tooltip intercepts clicks* (2026-08-21) — web UI bug, unrelated; stays in backlog.

</deferred>

---

*Phase: 16-Provenance-Based Plugin Trust*
*Context gathered: 2026-08-19*
