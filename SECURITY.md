# Security Policy

## Supported Versions

topos is pre-1.0 and does not yet publish versioned releases with a
support matrix. Security fixes land on `main`; there is no older
maintained branch.

| Version | Supported |
| ------- | --------- |
| main    | ✅        |

## Reporting a Vulnerability

Please use GitHub's private vulnerability reporting for this repository
rather than opening a public issue: [Report a vulnerability](https://github.com/davison/topos/security/advisories/new).

topos is a locally-run, single-user desktop tool. Its kernel binds
`127.0.0.1` (loopback) only and has no authentication on its HTTP API in
v1 — that is the deliberate, documented v1 security boundary, not an
oversight, and is not itself a finding.

The report classes that matter most here:

- Anything that could cause a plugin to mutate a source data store —
  every plugin is read-only by design.
- Anything that could leak a credential or secret out of config,
  environment, or logs.
- Anything that lets content from a source execute in the embedded web
  UI, such as an unsanitised rendition escaping the kernel's content
  boundary.
- Anything that lets an agent read a source it was not granted.

We aim to acknowledge reports within 7 days.
