# TRUST-02 Proof: A Real `topos-plugins` Release Verifies as First-Party Trusted

**Recorded:** 2026-08-20
**Plan:** 16-04-PLAN.md (Task 2)

This is the evidence success criterion 2 of TRUST-02 asks for: that the
verifier introduced in 16-01 is proven against an artifact produced by the
real signing pipeline this plan stood up — never a hand-crafted test
double. Every command below was run against real, downloaded release
assets and a real installed `topos` instance built from this worktree.

## The published artifacts

- **Repository:** https://github.com/davison/topos-plugins (public)
- **Release:** https://github.com/davison/topos-plugins/releases/tag/v0.0.1
- **Workflow run:** https://github.com/davison/topos-plugins/actions/runs/32325806543 — conclusion `success`
- **Signing key id:** `topos-plugins-2026a`
- **Public key spec (embedded in the kernel):**
  `topos-plugins-2026a=hHDvrubE9EkNOl81FY2obRUaeeeBgmZcdWNXy93xntQ=`
- **Tag commit:** `d2cc6837df07741c83967a1f807a0101d8de7494` (topos-plugins `main`)

### Release assets and their SHA-256 digests

(From `checksums.txt`, published by the release itself, and independently
re-verified with `sha256sum -c checksums.txt` after download — all three
report `OK`.)

```
d8a1857758674827a79a09507e2a1c71176431c3796b1c14d2c15040f9b57572  topos-plugin-demo
f0406e155fc0d23a98c90cd8da49442dfd07dad795b3a432c3312acd11f8217d  topos-plugins-v0.0.1.provenance.json
3d259c48887a1e314d14cd34799f084fd7bd4df9464b425d3c478b1e28d8f26b  topos-plugins-v0.0.1.provenance.sig
```

### The signed manifest itself

```json
{
  "schema": "topos.provenance.v1",
  "release": {
    "repo": "davison/topos-plugins",
    "tag": "v0.0.1",
    "os": "linux",
    "arch": "amd64"
  },
  "plugins": [
    {
      "name": "topos-plugin-demo",
      "sha256": "d8a1857758674827a79a09507e2a1c71176431c3796b1c14d2c15040f9b57572",
      "version": "v0.0.1",
      "contract": "topos.v1"
    }
  ]
}
```

```json
{
  "schema": "topos.provenance.sig.v1",
  "key_id": "topos-plugins-2026a",
  "algorithm": "ed25519",
  "signature": "zyU6QrEs4MlwSDZG31wxB1/opW8vR2M/vTd7TJ583OLN1+BIRvVml12EOv3nv4Y3kcu2FFpe90OWMsxBikhgCw=="
}
```

### The signing key never appeared in the workflow log

The GitHub Actions run's "Sign the release manifest" step log shows the
secret masked as `***` at its one point of reference (the `env:` block);
grepping the full run log for the base64 public-key spec above (a
different, non-secret string) also finds no match anywhere the private
key could have leaked. The private key file was deleted from local disk
immediately after `gh secret set` uploaded it (see 16-04-SUMMARY.md for
the deletion record); an encrypted offline backup exists, described in
that SUMMARY.

## Pinning note

The release workflow invokes `go run
github.com/davison/topos/cmd/topos-provenance@<commit>` pinned to the
exact topos kernel commit
(`e0290110026c5717d8fadebcb15f681314b9f2c1`) that introduced
`cmd/topos-provenance` (16-01-PLAN.md) — no topos kernel release TAG
carries this CLI yet, since this proof runs before that tag exists. This
satisfies "a pinned version rather than a floating one" literally (an
exact commit SHA is a strictly narrower pin than a tag) but should move to
a real semver tag the next time the topos kernel repository cuts a
release that includes `cmd/topos-provenance`.

## The installed instance

Built from this worktree with the newly embedded `topos-plugins-2026a`
public key (`kernel/pluginhost/provenance.go`), via `make build-portable`,
plus `go build ./cmd/topos-provenance`, both placed into a throwaway
prefix — never a `sudo make install` on the operator's real machine.

```
$ make build-portable    # produces bin/topos (embeds ONLY the 6 in-repo
                          # portable plugin binaries' build manifest —
                          # see below)
$ go build -o <prefix>/bin/topos-provenance ./cmd/topos-provenance
```

**Structural proof the link-time build manifest has no entry for
`topos-plugin-demo`:** `build-portable`'s own manifest generator is
invoked with exactly six binaries (`Makefile`'s
`MANIFEST_PLUGIN_BINARIES_PORTABLE`) — paperless, silverbullet, proton,
mock, whatsapp, filesystem:

```
topos-plugin-filesystem=6afc9d5de5fc651bcb86ef84061b13e727d69bdd784e2af436ccf839619d0094,topos-plugin-mock=15d48e36655e35a4bf3279a83bd0e451c39a350f5a0fa018afefccef0830246a,topos-plugin-paperless=27d13a2052544362ea63a41b8ecbc600dbcaa52e533d62fdf62988af83840337,topos-plugin-proton=3ec74921a2059525fb7df38e8430476c9c8b59e0342144df9bda2d9a2e95b99b,topos-plugin-silverbullet=3bdf1bef4edbab8ea5b1ca566f812ab6126a1c038b917cc5e1882c3d6538c496,topos-plugin-whatsapp=a0f198c68e96ede43736c11c70ea091e2d7072850c4e99aa4497587874aa1437
```

`topos-plugin-demo` names no entry here — its ONLY path to `TierTrusted`
in this installed kernel is the signed release-manifest arm.

The downloaded release binary, `.provenance.json`, and `.provenance.sig`
were placed into the prefix's `[plugins] dir`
(`<prefix>/lib/topos/plugins/`), and a `[sources.demo]` instance
(`plugin = "topos-plugin-demo"`) was configured pointing at it.

## 1. `topos-provenance verify` — exits 0, names the manifest

```
$ topos-provenance verify --dir <prefix>/lib/topos/plugins
topos-plugin-demo: OK (topos-plugins-v0.0.1.provenance.json)
```

Exit code `0`.

## 2. `GET /api/sources` reports the demo source as `trusted`

```json
{
  "schema_version": 1,
  "sources": [
    {
      "name": "demo",
      "source_type": "demo",
      "display_name": "topos-plugins Demo",
      "plugin": "topos-plugin-demo",
      "tier": "trusted",
      "reachable": true,
      "syncing": false,
      "last_status": "ok",
      "last_sync_unix": 1787194125,
      "last_error": "",
      "last_notice": ""
    }
  ]
}
```

`tier: "trusted"`, `last_error: ""` — no `launch_failure`, no
`launch_advisory` field present at all. This is TRUST-02's whole claim:
first-party trust with **no** entry in the kernel's link-time build
manifest, granted purely by the signed release-manifest arm.

## 3. Negative case — `.provenance.sig` removed: external, not trusted

The `.provenance.sig` file was moved out of the plugins directory and the
kernel restarted. The kernel log names the binary and the specific cause
before any HTTP call:

```
[WARN]  topos: provenance evaluation diagnostic (D-11/T-16-07): binary=topos-plugin-demo ... detail="topos-plugins-v0.0.1.provenance.json: missing or unreadable sibling signature file topos-plugins-v0.0.1.provenance.sig: stat ...: no such file or directory"
[ERROR] topos: plugin launch refused: pinned binary hash mismatch (T-11-07): instance=demo plugin=topos-plugin-demo pinned_hash="" current_hash=d8a1857758674827a79a09507e2a1c71176431c3796b1c14d2c15040f9b57572
```

`GET /api/sources`:

```json
{
  "schema_version": 1,
  "sources": [
    {
      "name": "demo",
      "tier": "external",
      "current_hash": "d8a1857758674827a79a09507e2a1c71176431c3796b1c14d2c15040f9b57572",
      "launch_failure": "pin_mismatch",
      "reachable": false,
      "last_error": "pluginhost: instance \"demo\" binary \"topos-plugin-demo\" hash mismatch: pinned=not pinned current=d8a1857758674827a79a09507e2a1c71176431c3796b1c14d2c15040f9b57572"
    }
  ]
}
```

Without a valid signature, the binary falls to the external tier's
consent/pin path exactly like any unsigned binary — never a silent
trusted launch. The `.provenance.sig` file was restored afterward.

## 4. Negative case — tampered binary: launch refuses, names the binary

One byte was appended to the downloaded `topos-plugin-demo` binary
(digest changes from `d8a18577...` to `16e6fa3a...`). Both `verify` and a
real launch refuse:

```
$ topos-provenance verify --dir <prefix>/lib/topos/plugins
topos-plugin-demo: FAIL: pluginhost: plugin binary is not verified by any signed release manifest: "topos-plugin-demo" — manifest topos-plugins-v0.0.1.provenance.json names a digest that does not match the on-disk binary (signed digest d8a1857758674827a79a09507e2a1c71176431c3796b1c14d2c15040f9b57572, on-disk 16e6fa3a81538b4f49d9054ccf4595119413a69ab531da31925be3a7a8c0bc90)
```

Exit code `1`.

Kernel log at launch time:

```
[WARN]  topos: provenance evaluation diagnostic (D-11/T-16-07): binary=topos-plugin-demo ... detail="topos-plugins-v0.0.1.provenance.json: names \"topos-plugin-demo\" with digest d8a1857758674827a79a09507e2a1c71176431c3796b1c14d2c15040f9b57572, on-disk digest is 16e6fa3a81538b4f49d9054ccf4595119413a69ab531da31925be3a7a8c0bc90"
[ERROR] topos: plugin launch refused: trusted binary not verified by the build manifest (D-12/D-13): instance=demo plugin=topos-plugin-demo current_hash=16e6fa3a81538b4f49d9054ccf4595119413a69ab531da31925be3a7a8c0bc90
```

`GET /api/sources`:

```json
{
  "schema_version": 1,
  "sources": [
    {
      "name": "demo",
      "tier": "",
      "current_hash": "16e6fa3a81538b4f49d9054ccf4595119413a69ab531da31925be3a7a8c0bc90",
      "launch_failure": "manifest_unverified",
      "reachable": false,
      "last_error": "pluginhost: instance \"demo\" binary \"topos-plugin-demo\" is not verified by the kernel's build manifest or any signed release manifest (current=16e6fa3a81538b4f49d9054ccf4595119413a69ab531da31925be3a7a8c0bc90): pluginhost: plugin binary is not verified by any signed release manifest: \"topos-plugin-demo\" — manifest topos-plugins-v0.0.1.provenance.json names a digest that does not match the on-disk binary (signed digest d8a1857758674827a79a09507e2a1c71176431c3796b1c14d2c15040f9b57572, on-disk 16e6fa3a81538b4f49d9054ccf4595119413a69ab531da31925be3a7a8c0bc90)"
    }
  ]
}
```

The binary was restored to its pristine, digest-matching bytes afterward.

## 5. Fully offline verification (D-01)

Steps 1 and 2 were re-run inside a fresh Linux network namespace
(`unshare --net`) with only the loopback interface brought up — no route
to any external host exists in that namespace at all:

```
$ unshare --net --map-root-user -- sh -c '
    ip link set lo up
    <prefix>/bin/topos serve --config <prefix>/etc/config.toml &
    sleep 2
    curl -s http://127.0.0.1:7791/api/sources
    curl -s -m 3 https://github.com -o /dev/null -w "external_http_code=%{http_code}\n"
  '
{"schema_version":1,"sources":[{"name":"demo", ... "tier":"trusted", ...}]}
external_http_code=000
```

`external_http_code=000` (curl could not even resolve/connect) confirms
the namespace had no external network reachability, while the kernel
still launched `topos-plugin-demo` as `tier: "trusted"` — verification
performed entirely from local files (the manifest, its signature, and the
binary), exactly as D-01 requires.

## Summary

| Check | Result |
|---|---|
| Real release, real tag-triggered workflow, `success` | ✅ |
| Four assets published (binary, manifest, signature, checksums) | ✅ |
| `topos-provenance verify` exits 0, names the manifest | ✅ |
| Installed kernel reports `tier: trusted`, no link-time manifest entry | ✅ |
| Missing signature → external/untrusted, never silently trusted | ✅ |
| Tampered binary → launch refuses, names the binary | ✅ |
| Fully offline verification | ✅ |
| Signing key never appeared in any log; deleted after upload | ✅ |
