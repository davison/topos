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

**Phase 17 retires the link-time arm.** Once the plugins move out of this
repository, signed provenance becomes the only trusted path; this
document's "two evidence sources" section will shrink to one.

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
| Trusted | A validly-signed manifest, from an accepted key, names this exact binary with a digest matching what's on disk. |
| Untrusted (external tier) | No manifest present names this binary at all — a legitimate "no evidence" state, not an error. |
| Refused — never demote-and-run | A manifest signed by an unknown key id; a signature that does not verify; a manifest built for a different platform; a manifest that DOES verify but names this binary with a digest that no longer matches what's on disk (tamper). |

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

## Verifying by hand

Anyone can independently verify a downloaded release without installing
it or trusting this document's word for it:

```sh
topos-provenance verify --dir <path to the plugin binaries + manifest>
```

This is the exact verifier `scripts/install.sh`'s own install-time step
and the kernel's own launch gate both call — never a second,
re-implemented check. It prints one line per binary naming either the
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
