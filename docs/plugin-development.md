# Writing a topos plugin out of tree

This is the path for a stranger: from an empty directory on your own
machine to a plugin the kernel launches, syncs, and installs from a URL
— without cloning this repository into your project, without a seat in
its workspace, and without anyone here having heard of you. It is
deliberately a *path*, not a reference: every semantic (what `Match`
must return, what an `Item` field means, how config reaches you) lives
in [`docs/plugin-contract.md`](plugin-contract.md), and this guide links
there rather than restating it. When the two disagree, the contract
wins and this guide has a bug — please report it.

(Links in this guide are relative to the topos repository; if you were
handed this file on its own, every linked document is under `docs/` of
the checkout step 1 makes — `topos/docs/plugin-contract.md` and so on.)

The four published inputs a plugin is built from:

1. [`docs/plugin-contract.md`](plugin-contract.md) — the contract.
2. [`proto/topos/v1/plugin.proto`](../proto/topos/v1/plugin.proto) — the
   wire contract the sdk's generated stubs come from.
3. The `sdk` Go module — `github.com/davison/topos/sdk`.
4. The reference plugins: [`plugins/mock`](../plugins/mock) (the
   contract's worked example) and
   [`testdata/external-plugin`](../testdata/external-plugin) (the same
   thing built the way *you* will build it — its own module path,
   outside this repository, from the published inputs alone).

## 0. What you are building

A topos plugin is a **separate executable** the kernel launches as a
subprocess and talks to over gRPC through
[hashicorp/go-plugin](https://github.com/hashicorp/go-plugin). It
implements four RPCs — `Describe`, `Match`, `Fetch`, `Health` — and is
**read-only by construction** toward the system it reads (contract: "A
plugin is read-only by construction"). The kernel decides where your
binary may run from and how much it trusts it (contract: "Trust tiers");
your binary declares who it is and which contract generation it speaks.

Three names to get right from the start:

- Your **binary** must be named `topos-plugin-<name>`, where `<name>` is
  lowercase letters, digits and hyphens. The kernel discovers plugins by
  that prefix, and `topos plugin pull` refuses any other shape.
- Your **`source_type`** (returned from `Describe`) is your plugin's kind
  — short, lowercase, permanent once chosen (contract: "Describe").
- Your **contract generation** is `sdk.ContractVersion`. Declare exactly
  that constant in `Describe`; the kernel refuses, by name, any plugin
  whose declaration is outside the set it supports — including a
  missing one (contract: "Describe", and the `contract_incompatible`
  launch failure in [`docs/api.md`](api.md)).

## 1. Prerequisites

- **Go** (the version in this repository's `go.work`, currently 1.25 —
  any current Go toolchain will do).
- **A kernel to run against.** Until the first kernel release that
  ships `topos plugin pull` (v1.3.0), build one from a checkout:

  ```sh
  git clone https://github.com/davison/topos && cd topos
  go build -o bin/topos ./cmd/topos        # headless: the HTTP API, no web UI
  ```

  That single `go build` needs no Node and no browser toolchain; the
  binary serves the API (which is all this guide needs) with an empty
  UI. For the full app, `make build` (needs Node) or, once released,
  `make install` ([`docs/install.md`](install.md)).
- **`sha256sum` and `curl`** for the shipping and installing steps —
  standard on any Linux.

You do **not** need this repository's workspace, its Makefile, or
`protoc` — the sdk module ships the generated stubs.

## 2. Scaffold your module

Anywhere on disk, under any module path you own:

```sh
mkdir topos-plugin-hello && cd topos-plugin-hello
go mod init example.com/you/topos-plugin-hello
go get github.com/davison/topos/sdk@latest
go get github.com/hashicorp/go-plugin@latest
```

`go get …/sdk@latest` resolves to a pseudo-version of this repository's
`main` (the sdk module is versioned by commit, not by its own tags);
`go.mod` pins it. **Track the kernel you target**: `sdk.ContractVersion`
in the sdk you build against must be a generation the kernel you run
under supports (today there is exactly one, `topos.v2`). Rebuilding
against a newer sdk after a kernel upgrade is the normal cadence.

## 3. The smallest complete plugin

Two files. Every line below is contract-conformant; the comments name
the contract section each rule comes from so you can look past this
guide's simplifications.

`main.go`:

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"

	goplugin "github.com/hashicorp/go-plugin" // NOT the stdlib "plugin" package

	"github.com/davison/topos/sdk"
)

// sourceConfig is decoded from WEBSPACES_SOURCE_CONFIG, the JSON
// envelope the kernel sets on every launch (contract: "Configuration").
// Read only the keys you need; Extras is your own [sources.<id>.extras]
// table, passed through verbatim.
type sourceConfig struct {
	BaseURL string            `json:"base_url"`
	Extras  map[string]string `json:"extras"`
}

func main() {
	var cfg sourceConfig
	if raw := os.Getenv("WEBSPACES_SOURCE_CONFIG"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
			fmt.Fprintln(os.Stderr, "topos-plugin-hello: parse WEBSPACES_SOURCE_CONFIG:", err)
			os.Exit(1) // fail startup loudly, never mid-Match (contract: "Configuration")
		}
	}

	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: sdk.Handshake,
		Plugins: map[string]goplugin.Plugin{
			"source": &sdk.SourcePluginGRPCPlugin{Impl: newHello(cfg)},
		},
		GRPCServer: sdk.GRPCServer, // the contract's message-size ceiling
	})
}
```

`plugin.go`:

```go
package main

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/davison/topos/sdk"
	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

const (
	sourceType  = "hello"
	displayName = "Hello"
	binaryName  = "topos-plugin-hello"
)

// One declared match field: the kernel sends a webspace's keywords under
// every name you list here (contract: "Match"; "keywords" fallback).
var matchVocabulary = []string{"labels"}

type hello struct{ cfg sourceConfig }

func newHello(cfg sourceConfig) *hello { return &hello{cfg: cfg} }

// Describe runs once per launch, right after the handshake.
func (h *hello) Describe(context.Context, *toposv1.DescribeRequest) (*toposv1.DescribeResponse, error) {
	return &toposv1.DescribeResponse{
		SourceType:      sourceType,
		DisplayName:     displayName,
		ContractVersion: sdk.ContractVersion, // exactly this — never a literal
		MatchVocabulary: matchVocabulary,
	}, nil
}

// Match runs at sync time. Rule: exact, case-insensitive label match,
// zero items when nothing matches (contract: "Match").
func (h *hello) Match(_ context.Context, req *toposv1.MatchRequest) (*toposv1.MatchResponse, error) {
	labels := []string{"hello"}
	var hit bool
	for _, kw := range req.GetMatchFields()["labels"].GetValues() {
		for _, l := range labels {
			if strings.EqualFold(l, kw) {
				hit = true
			}
		}
	}
	if !hit {
		return &toposv1.MatchResponse{}, nil
	}
	return &toposv1.MatchResponse{Items: []*toposv1.Item{{
		SourceId:      "greeting-1",                               // stable within your plugin
		SourceType:    sourceType,                                 // must equal Describe's
		Title:         "Hello from an out-of-tree plugin",
		TimestampUnix: 1704067200,                                 // real event time, seconds
		Fidelity:      toposv1.LinkFidelity_LINK_FIDELITY_EXACT,   // never UNSPECIFIED
		DeepLink:      "https://example.com/hello/greeting-1",     // absolute, never ""
		Labels:        labels,
		Provenance: map[string]string{ // the five plugin-owned keys (contract: "Provenance")
			"source_type":      sourceType,
			"source_system":    "hello://" + h.cfg.BaseURL,
			"source_id":        "greeting-1",
			"plugin":           binaryName,
			"contract_version": sdk.ContractVersion,
		},
	}}}, nil
}

// Fetch runs at request time, when someone opens the item.
func (h *hello) Fetch(_ context.Context, req *toposv1.FetchRequest) (*toposv1.FetchResponse, error) {
	if req.GetSourceId() != "greeting-1" {
		return nil, status.Errorf(codes.NotFound, "hello: item %q not found", req.GetSourceId())
	}
	return &toposv1.FetchResponse{Available: true, Text: "Hello from an out-of-tree plugin."}, nil
}

// Health is a lightweight reachability probe.
func (h *hello) Health(context.Context, *toposv1.HealthRequest) (*toposv1.HealthResponse, error) {
	return &toposv1.HealthResponse{Reachable: true}, nil
}
```

```sh
go mod tidy
CGO_ENABLED=0 go build -o topos-plugin-hello .
```

If `go mod tidy` or `go build` fails with `read-only file system` under
`~/.cache/go-build`, Go's build cache is not writable where you are (a
sandbox, a read-only home): point `GOCACHE` at any writable directory —
`GOCACHE="$PWD/.gocache" go build …` — and add `.gocache/` to your
`.gitignore`.

If `go build` stops with `error obtaining VCS status … Use
-buildvcs=false to disable VCS stamping`, your plugin directory sits
inside another repository's tree without being one itself (a scratch
directory under some checkout, say): add `-buildvcs=false` — the kernel
never reads VCS stamps — or `git init` the plugin directory, which is
what you will do for a real project anyway.

`CGO_ENABLED=0` gives a static binary that runs on any Linux of the
same architecture — the shape every first-party plugin ships. A plugin
that genuinely needs cgo (the Signal plugin links the system SQLCipher)
is built per machine instead; see `topos-plugins`' `plugins/signal`.

Running the binary by hand prints go-plugin's "this binary is a plugin"
notice and exits — that is correct; only the kernel launches it.

## 4. Run it under a kernel

Your binary carries no evidence the kernel's trust model recognises
(contract: "Trust tiers"), so it runs in the **external tier**: from the
external plugin directory, content-pinned, badged untrusted. That is not
a lesser mode — it is the supported way to run code topos did not sign
([`docs/plugin-trust.md`](plugin-trust.md)). Three ways to get there:

### 4a. Headless, in one shot (the fastest loop)

A scratch config in your plugin directory — `dev.toml`. **Use absolute
paths for the two plugin directories**: the kernel resolves a relative
`[plugins] dir`/`external_dir` against its own *executable's* directory,
not against the config file (contract: "Discovery and launch" — the
rule that lets a built `bin/topos` find `bin/plugins/` beside it), so a
`./external` here would be looked for beside the `topos` binary. A
relative `[index] path`, by contrast, resolves against your working
directory; the example keeps it absolute too so nothing depends on
where you run from.

```toml
[server]
listen = "127.0.0.1:7799"

[index]
path = "/home/you/topos-plugin-hello/dev-index.db"

[plugins]
dir = "/home/you/topos-plugin-hello/no-trusted-plugins"   # may not exist — an empty trusted tier is fine
external_dir = "/home/you/topos-plugin-hello/external"    # where your binary goes

[plugins.pins]
"topos-plugin-hello" = "<sha256 of your binary>"   # sha256sum topos-plugin-hello | cut -d' ' -f1

[sync]
interval = "1h"

[sources.hello]
plugin = "topos-plugin-hello"
base_url = "unused"                   # the kernel requires base_url+token OR path on every
token = "unused"                      # source, even for a plugin that reads neither

# [sources.hello.extras]               # your own keys, passed through verbatim as the
# greeting = "good evening"            # envelope's "extras" map (contract: "Configuration")

[webspaces.demo]
keywords = ["hello"]                  # must match a label your Match returns
```

Then:

```sh
mkdir -p external && install -m 0755 topos-plugin-hello external/
/path/to/topos/bin/topos sync  --config dev.toml    # one-shot: launch, Describe, Match, index
/path/to/topos/bin/topos serve --config dev.toml &  # the API on :7799
curl -s http://127.0.0.1:7799/api/sources | python3 -m json.tool
```

`GET /api/sources` shows your instance with `"tier": "external"`,
`"reachable": true`, and no `launch_failure`; `sync` prints
`hello: 1 items`, and `GET /api/webspaces/demo/stream` lists the item.
A `launch_failed` entry whose message says the binary was "not found in
the trusted or external plugin directory" names the directories the
kernel actually looked in — almost always the relative-path trap above. The pin line is what stands in for the app's
consent click in a headless run (contract: "Pinning" — a hand-recorded
pin is honoured exactly like one the UI wrote). Rebuilt the binary?
Update the pin, or the next launch refuses it by name as
`pin_mismatch` — that refusal is the design working.

If instead you see `launch_failure: "contract_incompatible"`, your sdk
and the kernel disagree on the contract generation — the message names
both; rebuild against the sdk the kernel was built with (or the other
way round).

### 4b. Inside the kernel's dev loop (the full app, hot reload)

From a checkout of this repository with Node installed:

```sh
make dev DEV_PLUGINS_DIR=/path/to/dir-containing-your-binary
```

Every `topos-plugin-*` binary in that directory is copied beside the
mock and hashed into the dev kernel's link-time manifest, so your plugin
runs at the **trusted** tier in the dev instance (a build-time input,
never a runtime one — see the Makefile's `DEV_PLUGINS_DIR` comment). The
dev kernel listens on port 7778 with its own config
(`config.dev.toml`, generated on first run — add your `[sources.<id>]`
and `[webspaces.<name>]` blocks there; hand edits survive every
`make dev`).

### 4c. On an installed kernel, through the app

Copy the binary into the installed instance's external directory —
`$XDG_DATA_HOME/topos/plugins-external` (`~/.local/share/topos/plugins-external`
by default), or the `[plugins] external_dir` its config names — restart
the kernel, and add the source from the app's "+ New source" picker: the
untrusted-add interstitial shows the binary's hash, you consent, the
kernel records the pin, and the chip runs badged. This is the same flow
step 6 reaches with a URL instead of a copy.

## 5. Test it

Test against the contract's behaviour list, not your implementation.
The list is the contract's "RPC semantics" section (one subsection per
RPC) plus the `Item` table's required-field rules; the worked example
that asserts them is `plugins/mock/plugin_test.go`, and the contract's
"Build your first plugin" step 8 walks through what it covers. Adapt
the same assertions to your own plugin's real behaviour. If your plugin
reads a real system with required connection keys, `plugins/mockstrict`
shows the fail-loudly-at-startup discipline the contract's
"Configuration" section requires.

## 6. Ship it

A release is **flat files in one directory**: your binary under its
exact name, plus a `checksums.txt` over every file — the one convention
`topos plugin pull` discovers evidence by:

```sh
sha256sum topos-plugin-hello > checksums.txt
```

Publish that directory anywhere static HTTPS can serve it — a GitHub
Release's assets (`https://github.com/<you>/<repo>/releases/download/<tag>/…`),
an object-store bucket, a plain web root. Two rules that follow from how
the pull command works:

- **One platform per directory.** The pull command derives the plugin's
  name from the URL's basename, and that name must be exactly
  `topos-plugin-<name>` — so per-platform builds cannot be distinguished
  by suffix (`…-linux-amd64` is not a plugin name). Serve each platform
  from its own directory (or its own release), each with its own
  `checksums.txt`.
- **The checksums must be true.** A `checksums.txt` that omits the
  binary or disagrees with its bytes is a failed verification — the pull
  aborts and places nothing. Regenerate it whenever you rebuild.

`checksums.txt` is integrity, not authenticity. You may additionally
sign a provenance manifest with your own key using the kernel's
`topos-provenance sign` — but read the trust section below before you
expect that to change anything.

## 7. Install it — the same command everyone uses

```sh
topos plugin pull https://example.com/hello/linux-amd64/topos-plugin-hello
```

The kernel downloads the binary into a staging area, reads the
`checksums.txt` beside it, verifies, and places the binary in the
**external** directory with the consent-and-pin steps printed
([`docs/install.md`](install.md), "Installing a single plugin from a
URL"). Restart the kernel and add the source from the picker (4c) — or,
headless, record the pin in config as in 4a. To know *which* copy is
running: the pull's own report names the path it wrote; `sync`'s log
line `plugin process exited: plugin=<path>` names the binary each
launch actually executed; and `sha256sum` of that path equals the
release's `checksums.txt` line — three independent answers that must
agree. Updating is re-pulling: new bytes, a new hash, the chip's
re-pin ("Trust updated binary") action once.

## 8. The trust story, truthfully

The kernel's `trusted` tier is earned by **a signed release manifest
whose key is in the kernel's embedded key set** — the `topos-plugins`
repository's release key. There is a second word since v1.4.0
(§8a): a manifest signed by a key the *operator* has chosen to trust
earns `operator_trusted`, which launches exactly the same way. Without
either, your plugin is external-tier: content-pinned, badged untrusted,
consented to once per machine. Signing your release with your own key
does not make it trusted on its own — it makes it *offerable*: the
signature carries your public key, and an operator who checks your
published fingerprint can trust the key once, for every release you
sign. If you would rather not sign, publish `checksums.txt` alone and
let the external tier do its job; the consent interstitial shows your
operator the hash they are accepting, which is exactly the informed
decision the model wants them to make.

### 8a. Sign with your own key

The kernel has a second trust word, the operator's
([#49](https://github.com/davison/topos/issues/49); the design and what
is shipped are in [`plugin-trust.md`](plugin-trust.md),
"Operator-trusted keys"). You sign your releases with your own ed25519
key using the same tooling the fleet uses — `topos-provenance keygen`,
`sign`, `verify`, unchanged — and the signature file `sign` writes
carries your **key id** and **public key**; publish the key's
fingerprint (the SHA-256 of the raw key, which `verify` and
`topos plugin pull` print) somewhere your operators can check it. An
operator who installs your plugin is offered that key once — *trust
this key for future releases*, after which every release you sign runs
at *trusted by you* on their instance without a per-binary pin; or *pin
this binary only*, the external path. Rotate by shipping a new key id;
they are offered it the same way. Never reuse a key id for a different
key: the kernel treats that as an impersonation and warns.

Today (the kernel half, [#56](https://github.com/davison/topos/issues/56))
`topos plugin pull` places a release signed by an unknown key into the
external tier with its manifest and signature, prints your key id,
fingerprint and the `[[plugins.trusted_keys]]` entry that trusts it, and
the kernel reports the same offer on `GET /api/sources`; the operator
adds the entry and restarts. The in-app consent —
[#57](https://github.com/davison/topos/issues/57) — makes that one
click. A kernel older than v1.4.0 still aborts `pull` on an unknown key
(and runs the plugin external if placed by hand), so say which kernel
your signed releases need.

## 9. Before you call it done

The questions the last clean-room build against this contract could not
answer are all answered in the contract now — before you ship, read the
section behind each:

| Your question | Contract section |
|---|---|
| Where do tokens, caches and cursors live between launches? | "Plugin-private state" |
| How do my provider-specific config keys get a labelled form input? | "Configuration: `WEBSPACES_SOURCE_CONFIG`" (the extras subsection) |
| Which environment variables does my process actually receive? | "The launch environment" |
| What may my plugin never do to the system it reads? | "A plugin is read-only by construction" |
| What may I write to stderr, and what never? | "Logging" |
| What should `Health` report, and when? | "RPC semantics: Health" |
| Which `Item` fields are required, and what do they mean? | "The `Item` message" |

## How the contract evolves

The compatibility rules — what `sdk.Handshake.ProtocolVersion` and
`sdk.ContractVersion` each govern, and what the kernel does when a
plugin is on the wrong side of one — are the contract's "Handshake and
the plugin-map key" and "Describe" sections, with the resulting
launch-failure vocabulary in [`docs/api.md`](api.md). The action for
you is one line: track this repository's `sdk` module against the
kernel you target, and rebuild when it moves.
