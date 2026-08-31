# Plugin pages

The per-plugin operator pages moved with their plugins to
[`topos-plugins`](https://github.com/davison/topos-plugins) — each
plugin's own `README.md` there carries its install requirements,
configuration, and gotchas. (Their last in-repo versions live in this
repository's history, under `docs/plugins/` at tag `v1.2.0`.)

If you're writing a new plugin, start with
[`docs/plugin-development.md`](../plugin-development.md) — the
out-of-tree path from an empty module to an installable release — and
keep [`docs/plugin-contract.md`](../plugin-contract.md) open beside it:
that document is the author-facing contract, and `plugins/mock` is its
complete working reference implementation.
