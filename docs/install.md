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

## What a failed install leaves behind

Every refusal is designed to leave `$PREFIX` in a defined state:

- **Checksum mismatch** — the install aborts naming the failing file,
  and nothing has been written to `$PREFIX`: verification runs to
  completion before placement ever starts. The two installed
  directories may exist (they are created by the early writability
  probe) but contain no new files.
- **Missing asset** — a download that returns an HTTP error aborts
  naming the asset and the release tag, before verification and before
  any placement. `$PREFIX` is untouched.
- **Unwritable prefix** — probed up front, before any download work is
  done. The install fails naming the exact directory it cannot write
  and the `sudo make install` re-run form. The installer never
  escalates itself, so an unwritable prefix is left exactly as it was
  found — byte-for-byte, permission-for-permission.

## Re-running an install

Re-running an install of the same version into the same prefix is safe
and is the supported repair path: every asset is re-downloaded,
re-verified, and re-placed, ending in a byte-identical tree. There is
no version bookkeeping and no skip-if-present shortcut — a damaged or
partially missing install is fixed by simply running it again.

Installing over a **running** instance is also safe: each file is
renamed into place atomically, so the running kernel and its plugin
subprocesses are never truncated mid-execution. The running process
keeps executing its old code until you restart it — restart the
kernel to pick up the new release.

## Signal on an installed instance

The Signal plugin binary is deliberately **never** a published release
artifact: it is this project's only cgo build, dynamically linking the
system SQLCipher library, so a binary built on one distro carries no
promise of running on another. An installed instance gets Signal
support through an explicit opt-in instead:

```sh
make install-signal
```

This is the **only** install-surface command that needs a toolchain —
a Go toolchain, a C compiler, and the system `sqlcipher` package (see
[`docs/plugins/signal.md`](plugins/signal.md) for the per-distro
package names and the SQLite version floor). The base `make install`
stays download-and-copy only, and the hermetic gate proves it by
running the install with failing compiler shims first on `PATH`.

`make install-signal` builds the plugin through the repository's
single `signal` build definition and places the binary in the
installed instance's **external** plugin directory — by default
`$XDG_DATA_HOME/topos/plugins-external` (or
`~/.local/share/topos/plugins-external`), matching the kernel's own
default. If your config's `[plugins] external_dir` names a different
directory, point the installer at it:

```sh
make install-signal TOPOS_EXTERNAL_PLUGINS_DIR=/path/to/external-dir
```

**Why the external directory, and why there is a one-time consent
step.** The installed kernel verifies every binary in its *trusted*
plugins directory against the build manifest linked into the kernel at
release time. A binary you built locally cannot be in that manifest —
it did not exist when the released kernel was linked — so a
trusted-directory placement would be refused at launch. That refusal
is the trust model working as designed, not a limitation to route
around. The supported path is the external tier: after
`make install-signal`, you add the Signal source once through the
app's untrusted-add consent flow, and it runs pinned and badged
untrusted from then on. Re-running `make install-signal` produces new
bytes, which the kernel surfaces as a pin mismatch — re-accept the
changed binary through the chip's re-pin flow.

`make uninstall` never touches this binary (it lives outside
`$PREFIX`); remove it with:

```sh
make uninstall-signal
```

## Migrating from a checkout build to an installed instance

The path this section covers: you have a running instance started from
a checkout build (`make build`, or the `make dev` loop), and you want
it running from installed release artifacts instead — losing nothing.

**What is preserved, and why.** Your config file, your kernel index —
including every mark in it — and every plugin store (your WhatsApp
linked-device session most importantly) all live in the home/XDG
locations (`~/.config/topos/`, `~/.local/share/topos/`), which the
installer never reads and never writes. Installing touches only the
prefix (`$PREFIX/bin` and `$PREFIX/lib/topos/plugins`). There is
nothing to export, copy, or back up first — though nothing stops you
backing up anyway.

**The steps:**

1. **Stop the running instance.** Installing over a live one succeeds
   too (see "Re-running an install") — but the running process keeps
   executing its old code until restarted, so stopping first keeps the
   before/after line clean.
2. **Run the install** — `make install` for the latest stable release,
   or `make install VERSION=<tag>` for a specific one. The printed
   summary shows the resolved tag, the prefix, and every path written.
3. **Check the one config case that needs attention.** If your config's
   `[plugins] dir` is an **absolute** path pointing into your checkout
   (e.g. `/home/you/topos/bin/plugins`), the installed kernel will keep
   resolving your checkout's plugin binaries — an absolute value is
   honoured verbatim, by design. Two ways out: clear the key back to
   the stock relative `"plugins"` (the default), so the installed
   layout resolves `$PREFIX/lib/topos/plugins` automatically; or point
   it at the installed plugins directory explicitly. The installer
   deliberately does not edit your config to do this for you — it
   never touches your config at all.
4. **Start the installed instance** — `topos serve` (with
   `$PREFIX/bin` on your `PATH`). It serves on the same address your
   config's `[server] listen` names — `127.0.0.1:7777` by default.
   Confirm it is the installed binary that is running:
   `command -v topos` should print `$PREFIX/bin/topos`, and
   `ls -l $PREFIX/bin/topos` shows the install time.
5. **Verify nothing was lost.** The webspace list is unchanged; an
   item you had marked is still marked; each source's chip shows the
   same health state it had before. WhatsApp is still linked (the
   session store never moved), and Signal — if you use it — needs the
   one-time `make install-signal` opt-in described in the Signal
   section above.

**Backing out.** `make uninstall` removes exactly what `make install`
placed and leaves everything you own — config, index, marks, plugin
stores — in place. A checkout build still runs afterwards, against the
same config and data, exactly as before.

**Running a dev instance alongside.** The checkout's `make dev` loop
runs beside the installed instance by design — its own port (7778),
its own per-checkout config and state, and a pre-flight guard that
refuses to touch what the installed instance owns. That story lives in
[`docs/testing.md`](testing.md)'s "The real config and the dev config"
section rather than being retold here.

### Troubleshooting a migration

- **Every source reports a launch failure (binary not found).** The
  installed kernel is not finding its plugins — almost always the
  absolute-`[plugins] dir` case in step 3 above: your config still
  points at the checkout. Apply one of that step's two remedies.
- **The Signal source reports `manifest_unverified`.** A locally built
  Signal binary is sitting in the trusted plugins directory, where the
  released kernel's build manifest cannot vouch for it. See
  [Signal on an installed instance](#signal-on-an-installed-instance)
  — `make install-signal` places it in the external directory and the
  one-time consent flow takes it from there.

## Verifying the install machinery itself

```sh
make install-check
```

runs a hermetic smoke test: it builds a fixture release on local disk,
installs from it through the exact same script (no network, no
credentials), and asserts the installed kernel actually launches a
plugin out of `$PREFIX/lib/topos/plugins`. Useful after changing
`scripts/install.sh` or the kernel's plugin-directory resolution.
