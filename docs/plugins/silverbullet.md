# SilverBullet

Reads pages from a SilverBullet space over its HTTP filesystem API and
matches on tags and page names, with exact deep links back to the page.

## Install Requirements

None beyond a reachable SilverBullet instance and an auth token.

## Configuration

```toml
[sources.silverbullet]
plugin = "topos-plugin-silverbullet"
display_name = "SilverBullet"
base_url = "${SILVERBULLET_URL}"
token = "${SB_AUTH_TOKEN}"
# ca_cert = "~/.config/topos/silverbullet-ca.pem"

[sources.silverbullet.agent]
read = false
handoff = false
```

Match vocabulary: `tags`, `pages`.

`ca_cert` is the optional, supported path for a self-signed or
private-CA instance: it pins the CA this source trusts, in addition to
the system trust store. `base_url` and `token` use the environment-
expansion form exactly as `config.example.toml` does — never a literal
host or token. See `config.example.toml` for the fully-commented
reference block; this page summarises it and does not reproduce it.

## Gotchas

- A self-signed instance fails to connect until `ca_cert` points at the
  CA certificate that signed it.
- Matching on `tags`/`pages` is exact and case-insensitive, same rule as
  every other source.

## Security & Privacy Notes

- **Read-only:** enforced by this repository's own AST scan —
  `plugins/paperless/readonly_test.go`'s `TestPluginsIssueOnlyGetRequests`
  walks every `.go` file under `plugins/`, this package included, and
  fails the build on any non-GET HTTP reference.
- **Credentials:** `token` is the SilverBullet `SB_AUTH_TOKEN`, sent as a
  Bearer token on every request.
- **Egress:** restricted to the configured `base_url` host by
  `outbound_hosts_test.go`'s `TestAllowHost_PredicateTable` and its
  redirect-following tests. `ca_cert` is the supported way to trust a
  private CA — there is deliberately no option that disables certificate
  verification, and none should be added.
