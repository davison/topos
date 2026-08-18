# Signal

Reads Signal Desktop's own local SQLCipher database strictly read-only and
turns matched conversations into conversation-day digests.

## Install Requirements

This is the repository's only cgo plugin — every other plugin here builds
`CGO_ENABLED=0`. Before running `make signal` or `make test-signal`,
install the system `sqlcipher` package:

```bash
# Arch
sudo pacman -S sqlcipher

# Debian / Ubuntu
sudo apt-get install libsqlcipher-dev
```

`make signal` and `make test-signal` both build with `CGO_ENABLED=1` and
the `libsqlcipher` build tag, and both fail to link without this package.

**SQLite version floor: 3.51.3.** This plugin reads `sqlite_version()`
immediately after opening the database and refuses to run if the linked
SQLite core is below that floor, naming the version it found. SQLite
3.51.3 fixed a critical WAL-reset database-corruption bug — exactly the
failure class a read-only reader of a live, actively-written WAL database
(Signal Desktop sets `journal_mode=WAL`) is directly exposed to.

## Configuration

Unlike every other source in this project, Signal has no `base_url` or
`token` — it reads a local file, not a network endpoint. The only
required key is `path`, Signal Desktop's own config directory:

```toml
[sources.signal]
plugin = "topos-plugin-signal"
path = "~/.config/Signal"

[sources.signal.agent]
read = false
handoff = false
```

Match vocabulary: `conversations`.

There is no key, token, or secret to configure here at all — see Security
& Privacy Notes, below. See `config.example.toml` for the fully-commented
reference block; this page summarises it and does not reproduce it.

## Gotchas

- A distro `sqlcipher` package older than the 3.51.3 SQLite floor fails
  loudly at startup, naming the version it found. The fix is to upgrade
  the system package, not to work around the check.
- This plugin binary is not published as a prebuilt artifact; `make
  signal` is the local build path (see Install Requirements, above).

## Local builds and the build manifest

**A locally-built Signal binary dropped next to a release kernel refuses
to launch — this is the trust system working correctly, not a bug and
not a special case for Signal.** See
[`docs/plugin-contract.md`'s "Trust tiers"](../plugin-contract.md#trust-tiers)
for the full mechanism; this section is the worked example from the one
plugin whose local build path routinely hits it.

**Why this plugin, specifically, hits the refusal.** `topos-plugin-signal`
is this repository's only cgo build (`docs/releasing.md`'s "The Signal
plugin binary") — every other published binary is a static
`CGO_ENABLED=0` build that runs anywhere, so it ships in every release
artifact. This one dynamically links the system `sqlcipher` library and
carries no such portability guarantee, so it is deliberately excluded
from every published artifact; a Signal user builds it locally with `make
signal` instead. It follows directly that a release kernel's link-time
build manifest can never contain a locally-built Signal binary's hash —
that binary did not exist yet when the release kernel was linked. Drop it
into the release kernel's trusted plugin directory and the kernel refuses
to launch it (`launch_failure: "manifest_unverified"`), by name, on
`GET /api/sources`, exactly as it would for any other trusted-directory
binary the manifest doesn't recognize.

**The fix, step by step:**

1. Build and place in one command: `make install-signal` — the
   supported form for an operator running an installed instance. It
   builds through the same `signal` definition (see Install
   Requirements, above) and places the binary in the external plugin
   directory atomically; see
   [`docs/install.md`'s Signal section](../install.md#signal-on-an-installed-instance)
   for the destination default and its override. Working from a
   checkout, the equivalent longhand is: `make signal`, then place the
   built `topos-plugin-signal` binary in the **external** plugin
   directory (`[plugins] external_dir`) yourself — not the trusted one.
2. Add it through the app's untrusted-add consent flow — the same
   explicit consent-and-pin path any other external, unverified binary
   goes through (`docs/plugin-contract.md`'s "Pinning").
3. It now runs pinned, badged untrusted. Re-running `make
   install-signal` (or `make signal`) later produces new bytes and
   requires re-accepting the changed binary through the chip's own
   re-pin flow — exactly like any other pinned external plugin whose
   bytes changed.

**Whoever builds the whole repository together is unaffected.** `make
build` and `make dev` both build the Signal plugin as part of their own
`plugins` target (which requires the system `sqlcipher` package — see
Install Requirements, above) and generate the trust manifest from that
exact build's own binaries in the same invocation — the Signal binary's
hash is already in that kernel's own manifest, so it launches trusted, no
consent flow needed. (`make build-portable`, by contrast, never builds
Signal at all — see `docs/releasing.md`'s "The Signal plugin binary" —
so this refusal never applies to it either.) The refusal above is
specific to a locally-built Signal binary paired with a DIFFERENT,
already-linked kernel — a downloaded release, most commonly — never to a
from-source build of the whole project with `make build`/`make dev`.

## Security & Privacy Notes

- **Read-only:** this plugin never writes to Signal Desktop's database. It
  opens `db.sqlite` with a `mode=ro` DSN and never `INSERT`s, `UPDATE`s,
  `DELETE`s, `VACUUM`s, or checkpoints the WAL. `readonly_test.go` walks
  this package's own AST and fails the build on any write-shaped SQL
  reference; `byte_identical_test.go` proves a full Match+Fetch cycle
  leaves the database byte-identical.
- **Credentials:** the SQLCipher decryption key is never stored in this
  project's config — it is resolved entirely at runtime from Signal
  Desktop's own `config.json` under `path`, branching automatically
  between the legacy plaintext-key shape and the modern Electron
  `safeStorage`-wrapped shape. There is nothing secret to put in topos
  config.
- **Egress:** none — this plugin talks only to a local file on disk, never
  the network.
