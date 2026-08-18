# Installing topos

`make install` places a published topos release onto this machine from
the project's GitHub Releases — no Go toolchain, no Node, no build step.
It downloads the release's kernel and plugin binaries, verifies every
file against the release's own `checksums.txt`, and installs them under
a single prefix.

## Prerequisites

- `curl` and `sha256sum` on `PATH` (both standard on any mainstream
  Linux install). `mktemp` is also required and is part of coreutils.
- No Go toolchain, no Node, no credentials — public releases download
  anonymously.
- A checkout of this repository (for the `Makefile` and
  `scripts/install.sh`; the install itself never compiles anything).

## Installing a release

```sh
make install VERSION=1.1.0
```

`VERSION` names the release tag, with or without the leading `v`
(`1.1.0` and `v1.1.0` are the same release). It is required — there is
no implicit "latest" yet.

### `PREFIX`

The install root defaults to `/usr/local`. Override it per invocation:

```sh
make install VERSION=1.1.0 PREFIX=$HOME/.local
```

A `PREFIX` your user cannot write fails loudly, naming the unwritable
directory and the `sudo make install` re-run form. The installer never
runs `sudo` (or any other privilege escalation) itself — escalating is
always your own deliberate action.

## What gets written

Exactly two locations, and nothing else:

| Path | Contents |
|------|----------|
| `$PREFIX/bin/topos` | the kernel binary (mode 0755) |
| `$PREFIX/lib/topos/plugins/` | every plugin binary the release publishes (mode 0755 each) |

The installed kernel finds its plugins in that `lib/topos/plugins`
directory automatically, with the stock config value
`[plugins] dir = "plugins"` unchanged: when a relative plugins
directory does not exist beside the executable and the executable's own
directory is named `bin`, the kernel probes `<prefix>/lib/topos/plugins`
as the installed-layout sibling. No config edit is needed to move onto
installed artifacts, and a checkout build (`bin/topos` with
`bin/plugins/` beside it) is unaffected — its executable-relative
directory exists and always wins. An absolute `[plugins] dir` is
honoured verbatim in both layouts.

The installed instance reads its config and writes its state (the
index database) in the same home/XDG locations a checkout-built kernel
uses — `make install` itself never reads, writes, or migrates any
config or index file.

## Checksum verification

Every downloaded file is verified with `sha256sum -c` against the
release's own `checksums.txt` **before** anything is written to
`$PREFIX`. The asset list itself is derived from `checksums.txt` — the
release's published manifest — never from a second list that could
drift. Any mismatch aborts the install with the failing file named, and
leaves `$PREFIX` byte-for-byte unchanged: nothing is placed until every
asset has verified.

## Verifying the install machinery itself

```sh
make install-check
```

runs a hermetic smoke test: it builds a fixture release on local disk,
installs from it through the exact same script (no network, no
credentials), and asserts the installed kernel actually launches a
plugin out of `$PREFIX/lib/topos/plugins`. Useful after changing
`scripts/install.sh` or the kernel's plugin-directory resolution.
