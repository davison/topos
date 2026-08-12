# {Plugin Display Name}

{One or two sentences: name the source system this plugin reads from, what
it reads out of it, and what it turns that into (e.g. "conversation-day
digests", "matched documents with deep links back to the original").}

## Install Requirements

{System packages, SDKs, and version floors this plugin needs before it
will build or run, with the reason for each floor stated, not just the
version number. If this plugin needs nothing beyond a reachable instance
of its source system, say so explicitly — "none beyond X" — a blank
section reads as an oversight, not as "nothing needed."}

## Configuration

{A minimal, working `[sources.<id>]` TOML block for one instance of this
plugin — only the keys this plugin actually requires, no optional keys
unless the page's Gotchas section depends on them.}

```toml
[sources.example]
plugin = "topos-plugin-example"
# ...
```

Match vocabulary: {the exact field-name list this plugin's `Describe` RPC
declares, e.g. `tags`, or `folders` — copy it verbatim from the plugin's
own `matchVocabulary` declaration in source, never paraphrase it.}

Every credential-bearing value above uses the environment-expansion form
(`${VAR}`) exactly as `config.example.toml` does — never a literal token
and never a real hostname. See `config.example.toml` for the fully-
commented reference block; this page summarises it and does not
reproduce it.

## Gotchas

{The failure modes an operator actually hits configuring or running this
plugin, each stated symptom-first, fix-second — not a list of theoretical
edge cases.}

## Security & Privacy Notes

- **Read-only:** {the read-only guarantee and how it is mechanically
  enforced — name the actual test file(s), not just "it's read-only."}
- **Credentials:** {where credentials live, what they can and cannot do
  if leaked.}
- **Egress:** {what this plugin is allowed to talk to over the network,
  naming the outbound-host allowlist test where one applies. If this
  plugin makes no outbound network connections at all, say so.}

---

Every page under `docs/plugins/` starts from this file. A plugin without
a page here is undocumented — copy `_template.md` to `docs/plugins/<id>.md`
before shipping a new plugin.
