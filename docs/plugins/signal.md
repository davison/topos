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
