# paperless-ngx

Reads documents from a paperless-ngx instance over its REST API and
matches them on tags, with exact deep links back to the document.

## Install Requirements

None beyond a reachable paperless-ngx instance and an API token.

## Configuration

```toml
[sources.paperless]
plugin = "topos-plugin-paperless"
display_name = "paperless-ngx"
base_url = "${PAPERLESS_URL}"
token = "${PAPERLESS_TOKEN}"
api_version = "10"

[sources.paperless.agent]
read = false
handoff = false
```

Match vocabulary: `tags`.

`api_version` is the REST API version this plugin negotiates via the
Accept header — match it to your own paperless-ngx instance's supported
API version range. `base_url` and `token` use the environment-expansion
form exactly as `config.example.toml` does — never a literal host or
token. See `config.example.toml` for the fully-commented reference block;
this page summarises it and does not reproduce it.

## Gotchas

- An incompatible `api_version` is not validated at config load — it
  surfaces as an HTTP error from paperless-ngx itself at sync time.
- Matching on `tags` is exact and case-insensitive against tag names, same
  rule as every other source; a near-miss tag spelling silently matches
  nothing.

## Security & Privacy Notes

- **Read-only:** enforced by this repository's own AST scan —
  `readonly_test.go`'s `TestPluginsIssueOnlyGetRequests` walks every `.go`
  file under `plugins/` and fails the build on any non-GET HTTP
  reference.
- **Credentials:** `token` is a paperless-ngx API token, scoped to
  whatever access that instance's own token grants it.
- **Egress:** restricted to the configured `base_url` host by
  `outbound_hosts_test.go`'s `TestAllowHost_PredicateTable` and its
  redirect-following tests.
