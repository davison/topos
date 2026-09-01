# Installing topos

`make install` places a published topos release onto this machine from
the project's GitHub Releases — no Go toolchain, no Node, no build step.
It downloads the release's assets — the kernel and, from v1.3.0, the
provenance verifier (older tags that shipped plugin binaries still
install them byte-for-byte) — verifies every file against the release's
own `checksums.txt`, and installs them under a single prefix. The plugin
fleet itself is installed beside it by
[`topos-plugins`](https://github.com/davison/topos-plugins)' own
`make install` (see "Plugins on an installed instance", below).

## Prerequisites

- `curl` and `sha256sum` on `PATH` (both standard on any mainstream
  Linux install). `mktemp` is also required and is part of coreutils.
- No Go toolchain, no Node, no credentials — public releases download
  anonymously.
- A checkout of this repository (for the `Makefile` and
  `scripts/install.sh`; the install itself never compiles anything).

## Installing a release

```sh
make install                  # latest published stable release
make install VERSION=1.1.0    # a specific release
```

With no `VERSION`, the latest published **stable** release is resolved
by following the releases/`latest` redirect, and the landing URL is
validated before anything downloads: the host must be exactly
`https://github.com`, the path must be this repository's own
release-tag path, and the tag must be bare three-part
`v<major>.<minor>.<patch>` semver — so a prerelease or the moving
`nightly` tag can never be auto-selected. The resolved tag is printed
before the download starts. No credential, token, or GitHub CLI is
involved.

`VERSION` names a specific release tag, with or without the leading
`v` (`1.1.0` and `v1.1.0` are the same release).

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
| `$PREFIX/bin/topos` | the kernel binary (mode 0755) — plus `topos-provenance`, the provenance verifier, on releases that ship it (v1.3.0 onward) |
| `$PREFIX/lib/topos/plugins/` | the plugin fleet and its provenance pair, placed by [`topos-plugins`](https://github.com/davison/topos-plugins)' own `make install` — see below |

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

## Plugins on an installed instance

Kernel releases ship the kernel alone; the source plugins live in
[`topos-plugins`](https://github.com/davison/topos-plugins), which has
its own `make install`-grade flow. From a checkout of that repository:

```sh
make install                  # latest topos-plugins release
make install VERSION=0.3.0    # a specific release
make install PREFIX=$HOME/.local   # must match the kernel's prefix
```

It downloads the release, verifies every asset against the release's
own `checksums.txt` AND every binary against the release's signed
provenance manifest, then places the binaries with the manifest pair
in `$PREFIX/lib/topos/plugins/` — nothing is placed until everything
has verified, and `$PREFIX/bin` is never touched. Re-running with a
newer version is the update path (it also retires binaries the newer
release no longer publishes), and its `make uninstall` removes exactly
what it manages — whichever installer placed it, so fleets from
pre-split kernel releases come out the same way. See that repository's
README ("Installing the fleet") for the full story, including the
verifier resolution order and its bootstrap-trust caveat. For a single
plugin — a third-party one especially — `topos plugin pull <url>`
installs one binary into the tier its provenance earns (see
[Installing a single plugin from a URL](#installing-a-single-plugin-from-a-url)).

The provenance pair must sit beside the binaries: the installed kernel
trusts a plugin binary through its **signed release manifest** (the
phase-16 provenance arm — the kernel's own link-time manifest no longer
lists any functional plugin), verified against the `topos-plugins`
public key embedded in the kernel. `topos-provenance verify --dir` over
the directory shows exactly what the kernel will conclude.

## Installing a single plugin from a URL

```sh
topos plugin pull <url> [--config <path>]
```

takes one plugin binary from a URL to the trust tier its provenance
earns (M1-R8) — the per-plugin alternative to the fleet's
`make install`, and the install step a third-party plugin's release
instructions can name. The URL points at the binary asset itself; the
command downloads it into a staging area, reads the release's own
`checksums.txt` beside it (the one discovery convention — it records
the binary's SHA-256, verified in passing, and names the
`*.provenance.json`/`.sig` pair), verifies any signed manifest with
the kernel's embedded key set — the exact verifier the launch gate
calls — and only then places atomically:

- **A validly-signed manifest naming the binary's exact digest** lands
  the binary and its vouching manifest pair in the trusted plugins
  directory (the config's `[plugins] dir`, installed-layout probe
  included) — the kernel's launch gate then reaches the same verdict
  the pull just did.
- **No provenance evidence** lands the bare binary in the external
  directory with the consent-and-pin steps printed — the unchanged
  untrusted-add flow, exactly as a hand-copied binary would take. This
  is the state whether `checksums.txt` is cleanly absent or present
  and clean but naming no provenance pair — the legitimate shape of an
  unsigned third-party release, which publishes integrity without
  authenticity. A present `checksums.txt` must still name the binary
  with matching bytes, or the pull aborts.
- **Evidence that exists but does not verify** — a `checksums.txt`
  line contradicting the downloaded bytes or omitting the binary, a
  bad signature, an unknown key, a digest or platform mismatch, or
  published provenance that never vouches for this binary — **aborts
  naming the cause before anything is placed**: both plugin
  directories are byte-identical to their pre-attempt state, the
  directories themselves included (proven by committed tests in
  `cmd/topos/pull_test.go`), never a silent demotion to the external
  tier. Placement, once verification has passed, carries the same
  two-pass guarantee the install scripts document: staged copies
  first, then one pass of atomic per-file renames over pre-checked
  destinations — a failure mid-renames is reported for exactly what it
  is and a re-pull repairs it. Redirects are followed (GitHub assets
  redirect cross-origin), but an https→http downgrade anywhere in the
  chain is refused.

**No flag can override the earned tier** — there is no `--trusted`, no
`--external`, no provenance-URL escape hatch. Restart the kernel (or
apply config) after a pull and add the source from the picker.

## Checksum verification

Every downloaded file is verified with `sha256sum -c` against the
release's own `checksums.txt` **before** anything is written to
`$PREFIX`. The asset list itself is derived from `checksums.txt` — the
release's published manifest — never from a second list that could
drift. Any mismatch aborts the install with the failing file named, and
leaves `$PREFIX` byte-for-byte unchanged: nothing is placed until every
asset has verified.

## Provenance verification

Checksum verification (above) proves transport integrity — "these are
the bytes `checksums.txt` named" — never publisher authenticity. A
release that also publishes a signed provenance manifest
(`*.provenance.json`/`*.provenance.sig`, see
[`docs/plugin-trust.md`](plugin-trust.md) for the full format) gets a
second, independent check: still inside the same verify stage, still
before anything is placed, the installer runs the same
`topos-provenance verify` the kernel's own launch gate calls. A release
publishing no signed manifest — every kernel release to date; shipping
the `topos-provenance` verifier binary is tooling, not evidence — makes
this step a documented no-op; the install proceeds exactly as it always
has.

The installer resolves the verifier itself needs, in order: one already
installed at `$PREFIX/bin/topos-provenance` from a prior install
(kernel releases ship it from v1.3.0); one already on `PATH`; and only
then a `topos-provenance` binary published alongside the release in
the staged payload itself. A release that ships
provenance evidence but for which no verifier can be resolved is a loud
abort, never a silent skip — shipped evidence must be checked.

**Bootstrap-trust caveat.** Provenance verification exists specifically
to catch an attacker who can tamper with release artifacts *and*
regenerate `checksums.txt` to match — someone who controls the release
publishing pipeline but does not hold the ed25519 private signing key.
Under exactly that threat model, such an attacker could also publish a
`topos-provenance` binary in the same tampered release that
unconditionally reports success. The resolution order above is chosen to
narrow that window as much as it reasonably can: a previously-installed
or `PATH`-resolved verifier (neither of which comes from the release
payload under test) is always preferred over the staged payload's own
copy, and the staged copy is only ever consulted when no other verifier
exists anywhere on the machine. On a machine with no prior install and
nothing named `topos-provenance` on `PATH`, though, the staged payload's
own verifier is the only option — and this is an inherent limitation of
any "verifier shipped alongside the thing it verifies" design, not fully
eliminated by ordering alone. If you are installing on such a machine and
want the strongest guarantee, obtain `topos-provenance` from a separate,
independently-trusted source (e.g. a prior release, or build it yourself
from this repository) and place it on `PATH` before running the
installer.

## What a failed install leaves behind

Every refusal is designed to leave `$PREFIX` in a defined state:

- **Checksum mismatch** — the install aborts naming the failing file,
  and nothing has been written to `$PREFIX`: verification runs to
  completion before placement ever starts. The two installed
  directories may exist (they are created by the early writability
  probe) but contain no new files.
- **Provenance refusal** — a release carrying signed provenance evidence
  whose signature does not verify, names an unknown key, or names a
  digest that no longer matches the downloaded bytes aborts naming the
  binary, before any placement. `$PREFIX` is untouched — identical in
  shape to a checksum mismatch, just a second, independent check that
  catches what checksum verification alone cannot (an attacker who
  swaps a binary and regenerates `checksums.txt` still cannot forge a
  signature).
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

The Signal plugin is the fleet's one cgo build — it dynamically links
the system SQLCipher library, so no prebuilt binary is published
anywhere. Build it locally from a
[`topos-plugins`](https://github.com/davison/topos-plugins) checkout
(see topos-plugins' `plugins/signal/README.md` for the per-distro `sqlcipher`
package and the SQLite version floor):

```sh
cd topos-plugins && make install-signal   # builds, then places into the external directory
```

topos-plugins' `make install-signal` builds through its
`make build-signal` and places the binary atomically into the installed instance's **external** plugin
directory (by default `$XDG_DATA_HOME/topos/plugins-external`, or the
directory your config's `[plugins] external_dir` names via
`TOPOS_EXTERNAL_PLUGINS_DIR=<dir>`); topos-plugins' `make uninstall-signal` removes exactly that one file.

**Privileges differ by step, and the difference matters.** The kernel's
`make install` and the fleet's `make install` write under `$PREFIX`
(`/usr/local` by default) and need `sudo`; topos-plugins' `make
install-signal` and `make uninstall-signal` must run **without** it, as
the user the kernel runs as — the external plugin directory is that user's own data
directory, and under `sudo` the binary lands in root's, where no kernel
looks, while the script reports success. The script refuses to run as
root unless `TOPOS_EXTERNAL_PLUGINS_DIR` names the directory explicitly
([topos-plugins#21](https://github.com/davison/topos-plugins/issues/21)). Then add the Signal source once through the
app's untrusted-add consent flow — it runs pinned and badged. A locally built binary carries no signed provenance and is not
in any release manifest, so a trusted-directory placement would be
refused at launch; the external tier's consent-and-pin flow is the
supported path, exactly as it was when the plugin lived in this
repository.

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
   session store never moved), and Signal — if you use it — follows
   the external-tier flow described in the Signal section above.

**Backing out.** `make uninstall` removes the kernel's own artifacts
(`$PREFIX/bin/topos`, `$PREFIX/bin/topos-provenance`) and leaves
everything else — config, index, marks, plugin stores, and the plugin
fleet, which topos-plugins' own `make uninstall` removes — in place. A checkout build still runs afterwards, against the
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
- **A source reports `manifest_unverified`.** As of Phase 16 this one
  value covers refusals from either trust arm
  ([`docs/plugin-trust.md`](plugin-trust.md)) — the specific cause is
  named in the source's own `last_error` text, never a separate field:
  - **A locally built Signal binary** sitting in the trusted plugins
    directory, where no signed release manifest can vouch for it. See
    [Signal on an installed instance](#signal-on-an-installed-instance)
    — place it in the external directory and the one-time consent flow
    takes it from there.
  - **A missing signature file, an unknown signing key id, or a digest
    mismatch** against a signed release manifest — run
    `topos-provenance verify --dir <plugins dir>` against the binary
    directly for the exact diagnostic; see
    [`docs/plugin-trust.md`](plugin-trust.md#verifying-by-hand) for the
    invocation.
  In every case there is no re-pin/trust remedial action for this
  reason — the only path to running a binary neither arm vouches for is
  the existing external-tier consent-and-pin flow.
- **A source reports `handshake_incompatible` or
  `contract_incompatible`.** The kernel and the plugin fleet were
  updated independently (which is the design — see "Plugins on an
  installed instance") and their generations no longer agree: the
  source's `last_error` names both sides (go-plugin protocol versions,
  or declared-vs-supported contract generations). Update whichever side
  is behind — `make install` here for the kernel, `make install` in the
  topos-plugins checkout for the fleet — and restart. Every other
  source keeps running while the mismatch is displayed; a mismatch
  never takes the kernel down.
- **A source reports `launch_failed` naming a missing binary.** The
  fleet no longer ships a binary your config still names (or it was
  removed by hand). Remove or repoint the source, or reinstall the
  fleet.

## Verifying the install machinery itself

```sh
make install-check
```

runs a hermetic smoke test: it builds a fixture release on local disk,
installs from it through the exact same script (no network, no
credentials), and asserts the installed kernel actually launches a
plugin out of `$PREFIX/lib/topos/plugins`. Useful after changing
`scripts/install.sh` or the kernel's plugin-directory resolution.
