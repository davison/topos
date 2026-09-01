# Plugin trust

This is the one canonical statement of how topos decides what to trust.
Every other document that touches trust — [`docs/plugin-contract.md`](plugin-contract.md),
[`docs/api.md`](api.md), [`docs/install.md`](install.md) — links back here
rather than restating the model.

## The model, in one paragraph

Trust derives from **verifiable provenance carried by the artifact
itself**, never from where the binary sits on disk. A plugin binary is
first-party if, and only if, it is **signed by a key in the kernel's
embedded key set**; that key lives in the `topos-plugins` repository's own
CI. Everything else — any binary that is not, or is no longer, vouched
for by that key — is external-tier by construction: the existing
consent-and-pin path, described below, is how it runs anyway.

## The two evidence sources today (D-10)

Two independent arms grant `TierTrusted`, evaluated as **peers with no
precedence ordering** — a binary wins the moment either arm vouches for
it:

- **The kernel's link-time build manifest.** At kernel build time,
  `cmd/topos-manifest` hashes the exact plugin binaries the build just
  produced and links the resulting name → SHA-256 table into the kernel
  binary itself via `-ldflags -X`. This is how `make build`/
  `make build-portable`/`make dev` have always worked, and it keeps
  working unchanged during this transition.
- **A validly-signed release manifest** (this document's main subject) —
  a `.provenance.json`/`.provenance.sig` pair, described below, carried
  alongside the plugin binaries themselves.

**The split narrowed the link-time arm.** With the plugins moved out to
[`topos-plugins`](https://github.com/davison/topos-plugins)
(davison/topos#13), signed provenance is the only trusted path for
released, installed plugins. The link-time arm remains for what this
repository still builds itself — the mock reference plugin — and for
`make dev`'s adopted sibling binaries, whose hashes the dev recipe feeds
into the manifest at build time (`DEV_PLUGINS_DIR`): still a build-time
input, never a runtime-readable one. Both evidence sources stay; the
manifest arm's remit is simply the kernel's own builds.

## The on-disk format

A release publishes one **release manifest**, `<basename>.provenance.json`,
signed by exactly one **signature file**, `<basename>.provenance.sig`, sitting
beside it. **Multiple versioned manifests may coexist** in one plugin
directory — a binary is trusted if ANY validly-signed manifest present
names it with a matching digest, which is what lets plugins upgrade
independently without re-placing the whole set.

The manifest (`schema: "topos.provenance.v1"`) carries:

- `release` — the repo and tag this release was cut from, plus the
  `os`/`arch` its binaries were built for (a manifest built for a
  different platform grants no trust, even correctly signed).
- `plugins` — every binary this release vouches for, each entry naming
  `name`, `sha256`, `version`, and `contract` (the gRPC contract
  generation the binary was built against, e.g. `topos.v1` — the field
  Phase 17's plugin-vs-kernel contract mismatch check will consume).

The signature file (`schema: "topos.provenance.sig.v1"`) names `key_id`,
`algorithm` (only `"ed25519"` is ever accepted), and `signature` — the
standard-base64-encoded signature over the manifest file's **raw bytes,
verbatim**. The signature never covers a re-serialized or re-parsed form,
so verification never depends on JSON field ordering or whitespace.

## What earns trust, and what does not

| Outcome | Cause |
|---|---|
| Trusted | A validly-signed manifest, from a key the kernel author embedded, names this exact binary with a digest matching what's on disk. |
| Operator-trusted (`operator_trusted`) | The same, from a key in the operator's own `[[plugins.trusted_keys]]` — launches exactly as trusted; the chip and the API name the key. |
| Untrusted (external tier) | No manifest present names this binary at all — a legitimate "no evidence" state, not an error. **Every defective candidate is the same state at launch**: a signature that does not verify, a malformed manifest, a manifest built for another platform, or a signature naming a key id the kernel does not know — each is recorded as a diagnostic (`VerifySignedProvenance` keeps going) and contributes no evidence, so the binary is external and consent-and-pin runs it. `topos plugin pull`, however, aborts and places nothing whenever provenance was published but none of it vouches for the binary — an installer refuses what the launch gate merely declines to vouch for. The unknown-key case, when the signature file carries the signer's public key and verifies against it, also carries an **offer** — see "Operator-trusted keys" below; the rest are exactly as described. |
| Refused — never demote-and-run | Tamper: a manifest that verifies against an accepted key names this binary with a digest that no longer matches what's on disk. This is the only refusal the launch gate makes from signed provenance. |

A refusal is never silently downgraded to "run it at a lower tier
anyway" — verification never demotes-and-runs. Every refusal names the
binary and the cause, both in the kernel's own log and on the source's
chip.

## Key rotation (D-03)

The kernel embeds an **accepted-key SET**, not a single key. Rotating in
a new key is purely additive: ship a new kernel release adding an entry
to the embedded set. Releases signed by the OLD key stay verifiable
throughout the overlap — nothing an operator already trusts stops working
the moment a new key exists. A retired key's entry can be dropped in a
later release, once every operator has upgraded past the last release
that trusted it.

## Why an unsigned plugin is not a second-class citizen

Provenance is how a binary EARNS the trusted tier automatically. It is
not the only way to run a plugin. The external tier's consent-and-pin
flow — the interstitial the first time you add an unrecognized binary,
the content pin recorded in `[plugins.pins]`, the untrusted badge on its
chip, and the two-click re-pin path when its bytes change — is
**unchanged by this phase** and is the fully supported way to run code
topos did not sign. Nothing about provenance narrows what you can run;
it only widens what can run without that extra click.

## Operator-trusted keys

**Status:** decided at [davison/topos#49](https://github.com/davison/topos/issues/49)
(M2-R4 of [#40](https://github.com/davison/topos/issues/40)), from the
operator's captures [#1](https://github.com/davison/topos/issues/1) and
[#38](https://github.com/davison/topos/issues/38). **The kernel half is
shipped** ([#56](https://github.com/davison/topos/issues/56)): the
config table, the tier, the signature-carried key, the offer on
`GET /api/sources` and in `topos plugin pull`. The app's consent flow —
the badge, the two-choice interstitial, *stop trusting this key* — is
[#57](https://github.com/davison/topos/issues/57); until it lands, an
operator trusts a key by adding the table entry `pull` prints and
restarting the kernel.

Before this, trust was one word: the kernel author's, spoken by the
embedded key set. A third-party plugin was external for every operator
forever, and every release of it needed a fresh consent-and-pin, because
a pin is per binary. A developer who signed their release with their
*own* key gained nothing: at launch the kernel recorded "signature names
unknown key id" as a diagnostic and treated the manifest as no evidence
(external, consent-and-pin), and `topos plugin pull` went further and
refused to place the download at all, because provenance was present
but none of it accepted. The honest developer was, for the installer,
worse off than one who shipped unsigned.

The second word — the operator's:

- **Where it lives.** Keys the operator has chosen to trust are runtime
  configuration in the operator's own config, beside the pins, which are
  the same kind of decision:

  ```toml
  [[plugins.trusted_keys]]
  id         = "acme-2026a"          # the manifest's key_id
  public_key = "<base64 ed25519 public key>"
  trusted_at = 2026-09-02T10:00:00Z
  note       = "Acme's plugin signing key, from their release page"
  ```

  The accepted key set becomes embedded ∪ link-time extras ∪ operator
  keys, and each key remembers whose word it is.
- **A tier named for whose word it is.** A binary whose manifest verifies
  against an operator key runs at `operator_trusted` — *trusted by you
  (key acme-2026a)* on the chip — beside `trusted` (*signed by
  topos-plugins*) and `external`. It launches like trusted; the
  difference is honesty on the chip and in `GET /api/sources`, and a
  place for "stop trusting this key" to hang.
- **An unknown key is an offer.** A manifest whose signature names a
  key the kernel does not know stays what it always was at launch — *no
  evidence*, `external` — but carries an offer: `GET /api/sources`
  exposes the offered key's id, fingerprint and public key on that
  source (and on a `pin_mismatch` failure), and the app ([#57](https://github.com/davison/topos/issues/57))
  offers two consents — **trust this key for future releases** (writes
  the table entry; the plugin becomes operator-trusted, no pin needed)
  or **pin this binary only** (today's path). `topos plugin pull` no
  longer aborts on such a key: it places the binary *and* its manifest
  and signature into the external tier — so the kernel re-derives the
  same offer at launch — and prints the key, its fingerprint and the
  config entry that trusts it. **Launch behaviour
  for every other case is preserved**: tamper stays the one refusal; a
  signature that does not verify, a malformed manifest or a manifest for
  another platform stay no-evidence-at-launch (external) exactly as
  today, and pull keeps aborting on them — only the unknown key, which
  now carries a verifiable public key, is treated differently.
- **The key travels with the signature.** The signature file carries
  one more field: `public_key`, the signer's ed25519 public key in
  standard base64 — `topos-provenance sign` writes it with every
  signature (it holds the pair). The kernel verifies the signature
  against *that* key when the id is unknown, so an offer is only ever
  made for a key that demonstrably signed this manifest; the
  fingerprint is the SHA-256 of the raw key bytes. Trusting stores the
  bytes. A signature whose `key_id` matches an accepted key but whose
  `public_key` differs is an unknown key wearing a trusted name: no
  evidence, offered with `reused` set and a warning, because a reused
  id is exactly the impersonation this guards against. An older
  signature file with no `public_key` from an unknown key is what it
  always was — no evidence, no offer, and `pull` still aborts on it.
  The schema stays `topos.provenance.sig.v1`: an added field an older
  kernel ignores.
- **Removal and rotation.** Removing a key demotes its plugins to
  `external` at next launch, by name, into the existing consent path —
  no new failure vocabulary. A developer rotating keys ships a new key
  id; the operator is offered it like any unknown key; D-03's additive
  overlap applies to the operator's set as to the embedded one. No
  expiry — pins have none either.
- **Threat model, stated.** Editing the config already decides what runs
  (pins, plugin directories), so operator keys widen nothing. A key
  admits every future release by its signer — the same model the
  operator already accepts for the kernel author's key; the difference
  is that this party is chosen by the operator, which is what the
  consent step is for.

**Decision D-12 (revised at #49).** D-12 made the accepted key set
link-time data only — never a file, an environment variable or
configuration at run time. That stays true of the *kernel author's*
keys: the embedded set and the `provenanceKeysExtra` build seam are
untouched. The *operator's* keys are runtime configuration, exactly as
the operator's pins are, because both are the operator's own trust
decisions and the config file is already that surface. Rejected: a
separate keyring file (a second trust surface for the same decision),
and folding operator keys into `trusted` (the chip would then hide
whose word a plugin runs on).

**Decision (key transport, review round 1 of the design PR).** The
public key rides in the signature file. Rejected: a separately
published `<name>.pubkey` asset (a discovery rule to get wrong, and a
second fetch for `topos plugin pull`'s single URL), and carrying the key
in the manifest (the manifest is the thing signed; the signature file
is the signer's own statement and already names the key id).

Out of scope: a certificate authority, a registry, a marketplace
([#3](https://github.com/davison/topos/issues/3)), in-app install
([#2](https://github.com/davison/topos/issues/2)), key expiry.

## Verifying by hand

Anyone can independently verify a downloaded release without installing
it or trusting this document's word for it:

```sh
topos-provenance verify --dir <path to the plugin binaries + manifest>
```

This is the exact verifier `scripts/install.sh`'s own install-time
step, `topos plugin pull`'s tier decision (docs/install.md, "Installing
a single plugin from a URL"), and the kernel's own launch gate all call
— never a second, re-implemented check. It prints one line per binary naming either the
manifest that vouched for it or the specific failure cause, and exits
non-zero if any named binary failed to verify (or if the directory holds
no plugin binaries at all — a verify pass over nothing is never a
success). Pass `--name <binary>` (repeatable) to check only specific
binaries instead of every `topos-plugin-*` file present.

## See also

- [`docs/plugin-contract.md`](plugin-contract.md) — the plugin contract
  and directory layout this trust model sits on top of.
- [`docs/api.md`](api.md) — the `tier`/`launch_failure` fields
  `GET /api/sources` reports, and the `manifest_unverified` vocabulary
  entry that now covers both evidence arms.
- [`docs/install.md`](install.md) — the install-time provenance-
  verification step and its troubleshooting entry.
