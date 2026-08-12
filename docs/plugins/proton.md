# Proton Mail

Reads mail over IMAP from a running Proton Mail Bridge, matching on
folders, and never marks mail read.

## Install Requirements

A running Proton Mail Bridge instance — this plugin talks to Bridge, not
to Proton directly.

## Configuration

```toml
[sources.proton]
plugin = "topos-plugin-proton"
display_name = "Proton Mail"
base_url = "imaps://${PROTON_BRIDGE_ADDR}"
username = "${PROTON_BRIDGE_USER}"
token = "${PROTON_BRIDGE_PASS}"
ca_cert = "~/.config/topos/proton-bridge-cert.pem"
webmail_base_url = "${PROTON_WEBMAIL_BASE}"

[sources.proton.agent]
read = false
handoff = false
```

Match vocabulary: `folders`.

Two hard rules: `base_url`'s scheme (`imap`/`imaps`) must match what
Bridge itself reports for this account's own IMAP connection security,
and `ca_cert` is required, not optional, because Bridge presents a
self-signed certificate. `webmail_base_url` is the base the deep links
back into Proton's web client are built from. `username` and `token` use
the environment-expansion form exactly as `config.example.toml` does —
never a literal host, username, or token. See `config.example.toml` for
the fully-commented reference block; this page summarises it and does not
reproduce it.

## Gotchas

- Bridge binds loopback only, so reaching it from another machine needs a
  forwarder running on the Bridge host — this plugin cannot work around
  that.
- A scheme mismatch between `base_url` and Bridge's own reported setting
  fails to connect.
- `token` is Bridge's own generated password (Bridge -> Settings), not
  your real Proton account password — pasting the real account password
  will not work.

## Security & Privacy Notes

- **Read-only:** mail is fetched without ever marking it read; enforced by
  `readonly_test.go`'s `TestPluginIssuesNoIMAPMutatingCommands`.
- **Credentials:** the Bridge password (`token`) is scoped to Bridge alone
  and cannot sign in to the real Proton account, even if leaked.
- **Egress:** restricted to the configured Bridge host by
  `outbound_hosts_test.go`'s `TestAllowHost_PredicateTable`.
