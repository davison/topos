# API Coverage — SilverBullet HTTP API (`/.fs`)

> Full coverage by default. Opt-outs are explicit, reasoned decisions.
>
> Phase 2 integrates one external API: the SilverBullet server's HTTP surface,
> consumed by `plugins/silverbullet` (SRC-05). The baseline is full coverage of
> that surface, decided independently of the paperless-ngx integration shipped in
> Phase 1 — no opt-out is carried over from it.
>
> Capability surface established from the official HTTP API documentation
> (`silverbullet.md/HTTP API`), the `FileMeta` struct of the Go server module
> (`pkg.go.dev/github.com/silverbulletmd/silverbullet/server`), and the community
> confirmation that no search/filter endpoint exists — all three cross-checked in
> `02-RESEARCH.md`. The three open shape assumptions (A1/A2/A3) are resolved
> against the user's real instance in plan `02-01`, Task 1, Step 0; if that check
> reveals a capability not listed here, this matrix must gain a row for it.

| capability | decision | reason |
|---|---|---|
| `list-space` — `GET /.fs` (full file listing with per-file metadata) | INTEGRATE | |
| `read-file` — `GET /.fs/{path}` (raw page body) | INTEGRATE | |
| `file-metadata` — per-file `lastModified`/`created`/`contentType`/`size`/`perm` | INTEGRATE | |
| `auth-bearer` — `Authorization: Bearer <SB_AUTH_TOKEN>` | INTEGRATE | |
| `write-file` — `PUT /.fs/{path}` | OPT-OUT | permanently out of scope — the plugin contract is read-only by construction (PLUG-02); no source-mutating capability may ever be integrated. Enforced by `sdk/contract_test.go` and readonly tests |
| `delete-file` — `DELETE /.fs/{path}` | OPT-OUT | permanently out of scope — same read-only guarantee as `write-file` |
| `head-file` — `HEAD /.fs/{path}` (metadata via response headers) | OPT-OUT | not needed yet — the listing already carries this metadata in one request; HEAD adds a round trip with no new info. Useful only for incremental change detection (`02-RESEARCH.md` Pitfall 2) |
| `auth-session-cookie` — the interactive `/.auth` login flow | OPT-OUT | not needed — bearer-token auth is the locked approach for a headless plugin (`02-CONTEXT.md`); plan `02-01` Task 1 Step 0 halts if the instance requires cookie auth rather than silently degrading |
| `server-side-search-or-tag-filter` | OPT-OUT | capability does not exist — no query/filter/search endpoint (`02-RESEARCH.md` Pitfall 2); tag/page-name matching runs client-side over the full listing, an accepted cost |
| `static-client-assets` — `/.client/*` and the SilverBullet web UI's own routes | OPT-OUT | explicitly out of scope — webspaces links out to the SilverBullet UI via deep links (D-01, exact fidelity) and never proxies or re-serves its assets |
| `plug/space-script endpoints` — SilverBullet's plug and space-script surfaces | OPT-OUT | explicitly out of scope — running code in the user's instance is mutation-shaped and contradicts the read-only constraint; leading-underscore plug paths are excluded from matching |

## Not an external API integration

The kernel's own `/api/*` and `/agent/v1/*` routes are first-party surfaces this
phase authors, not external APIs being consumed; they are specified in
`docs/api.md` and covered by plans `02-02` and `02-04` rather than by this matrix.
