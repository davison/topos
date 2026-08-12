# WhatsApp

Runs as a WhatsApp linked device with its own persistent message store,
matching on group and contact names.

## Install Requirements

None at build time — the binary is pure Go — but one mandatory,
out-of-band linking step before first use (see Configuration, below).

## Configuration

```toml
[sources.whatsapp]
plugin = "topos-plugin-whatsapp"
display_name = "WhatsApp"
path = "~/.local/share/topos/whatsapp"

[sources.whatsapp.agent]
read = false
handoff = false
```

Match vocabulary: `groups`, `contacts`.

This source needs no `base_url`, no `token`, and no environment variable
at all — the linked-device session keys live only inside the session
store under `path`.

Linking is a one-time, out-of-band step run against the plugin binary
directly, never through the running kernel:

```bash
bin/plugins/topos-plugin-whatsapp -link -path ~/.local/share/topos/whatsapp
```

Scan the rendered QR code with your phone, then restart the kernel (or
`make dev`); the linked session survives restarts with no second scan.
See `config.example.toml` for the fully-commented reference block; this
page summarises it and does not reproduce it.

## Gotchas

- `path` holds two plugin-owned databases — the linked-device session
  store and the captured-message store — and must not collide with any
  other configured source's path, Signal's included.
- The linking step is run against the plugin binary directly and will not
  work through the running kernel.
- **Standing operational risk:** the linked device can be de-linked or
  banned by the platform at any time. The plugin degrades to named health
  states, already-captured rows survive, and there is no recovery beyond
  re-linking.

## Security & Privacy Notes

- **Read-only:** no message is ever sent from this plugin; enforced by
  `readonly_test.go`'s `TestReadOnly_NoSendCapableClientSelector`.
- **Credentials:** no credential is stored in topos config — the session
  store under `path` is plugin-owned and never touches another source's
  files. Path isolation is load-bearing, not a suggestion.
- **Egress:** restricted to WhatsApp's own linked-device endpoints by
  `outbound_hosts_test.go`'s
  `TestOutboundHosts_NoSelfConstructedHTTPClientOrUnlistedHostLiteral`.
