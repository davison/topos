# Plugin pages

These pages are for operators configuring an existing plugin; if you're
writing a new plugin, start with `docs/plugin-contract.md` instead — that
document is the author-facing contract.

- **[Proton Mail](./proton.md)** — reads mail over IMAP from a running
  Proton Mail Bridge, matching on folders.
- **[Signal](./signal.md)** — reads Signal Desktop's own local SQLCipher
  database, matching on conversations.
- **[WhatsApp](./whatsapp.md)** — runs as a WhatsApp linked device,
  matching on group and contact names.
- **[paperless-ngx](./paperless.md)** — reads documents from a
  paperless-ngx instance over its REST API, matching on tags.
- **[SilverBullet](./silverbullet.md)** — reads pages from a SilverBullet
  space over its HTTP filesystem API, matching on tags and page names.
- **[Filesystem](./filesystem.md)** — reads documents out of a local or
  network-mounted folder, matching on folder names.

A new plugin page is created by copying `_template.md`. The fixture
plugins `mock` and `mockstrict` deliberately have no page here — they are
test fixtures, not installable sources; `mock` is the PLUG-05 reference
plugin documented by `docs/plugin-contract.md` for plugin *authors*, and
`mockstrict` is built only by the `e2e` Makefile target.
