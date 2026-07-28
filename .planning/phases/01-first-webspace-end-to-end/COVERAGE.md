# API Coverage — paperless-ngx REST API (v10)

> Full coverage by default. Opt-outs are explicit, reasoned decisions.
>
> Detector: `api-coverage.cjs` returned `detected: false` against the ROADMAP phase section alone,
> but `detected: true` (4 signals, incl. `integration`+`rest` on "paperless-ngx REST integration (SRC-04)")
> against the phase scope including `01-RESEARCH.md` / `01-CONTEXT.md`. This phase does integrate an
> external API; the matrix is authored at plan time so the seal-time gate has a decided surface.
>
> Capability surface enumerated from `paperless-ngx/docs/api.md` and `src/documents/filters.py`
> (`main`, fetched 2026-07-27 during `01-RESEARCH.md`).

## Standing constraint on this matrix

Every **mutating** capability below is `OPT-OUT` for a reason that is *permanent*, not deferred:
REQUIREMENTS.md `## Out of Scope` excludes "Direct writes through the plugin contract — excluded
permanently", and PLUG-02 makes the plugin contract read-only *by construction* (no mutating RPC
exists to carry such a call). A later phase adding a mutating row to this matrix would be a
requirements change, not a coverage gap.

## Matrix

| capability | decision | reason |
|---|---|---|
| `GET /api/tags/?name__iexact=` — resolve webspace keyword to tag ID | INTEGRATE | |
| `GET /api/tags/` — paginated tag list (fallback resolution + label rendering) | INTEGRATE | |
| `GET /api/documents/?tags__id__in=` — OR-matched document fetch | INTEGRATE | |
| `GET /api/documents/` pagination (`page`, `page_size`, `next`) | INTEGRATE | |
| `GET /api/documents/{id}/` — document detail incl. extracted `content` | INTEGRATE | |
| `GET /api/documents/{id}/preview/` — inline rendition | INTEGRATE | |
| `GET /api/documents/{id}/thumb/` — thumbnail | INTEGRATE | |
| `GET /api/documents/{id}/download/` — original file bytes | OPT-OUT | not needed — the detail pane renders the `preview/` rendition inline; the original is what the "Open in paperless-ngx" deep link (UI-04) hands off to. Revisit if a rendition-less file type appears. |
| Token auth (`Authorization: Token <t>` header) | INTEGRATE | |
| `POST /api/token/` — exchange username+password for a token | OPT-OUT | not needed — D-04 requires the token come from the `PAPERLESS_TOKEN` env var; webspaces never handles the user's paperless password. |
| API versioning via `Accept: application/json; version=N` | INTEGRATE | |
| Frontend deep link `/documents/{id}` (PLUG-03 `exact` fidelity) | INTEGRATE | |
| `GET /api/documents/{id}/metadata/` — file checksum, MIME, archive info | OPT-OUT | not needed yet — `has_thumbnail` and the rendition MIME already come back on the detail/preview path. Candidate for a Phase 2 staleness signal (UI-05). |
| `GET /api/documents/{id}/notes/` — user notes on a document | OPT-OUT | not needed yet — notes are a paperless-native annotation surface; correlating them into the stream is additive and belongs with the Phase 2 renderer work, not the skeleton. |
| `GET /api/documents/{id}/history/` — audit trail | OPT-OUT | not needed — webspaces shows current state of a topic, not a per-document edit history. |
| `GET /api/documents/{id}/suggestions/` — ML tag/correspondent suggestions | OPT-OUT | explicitly out of scope — AI-inferred correlation is v2 (PROJECT.md `## Out of Scope`); v1 correlation is the deterministic config keyword map. |
| `GET /api/documents/?more_like_id=` — similar documents | OPT-OUT | explicitly out of scope — same v2 AI-correlation boundary as suggestions. |
| `GET /api/documents/?query=` — paperless full-text search | OPT-OUT | not needed yet — KERN-05 search is Phase 3 and is specified as FTS5 *over the local index* so one query spans every source, not per-source search fan-out. |
| `GET /api/correspondents/` | OPT-OUT | not needed yet — D-02 matches webspace keywords against tags only. Correspondent is a candidate stream-row metadata field once a second document source exists. |
| `GET /api/document_types/` | OPT-OUT | not needed yet — same reason as correspondents; not part of keyword matching or the Phase 1 row spec in `01-UI-SPEC.md`. |
| `GET /api/storage_paths/` | OPT-OUT | not needed — a paperless-internal filing concept with no webspace meaning. |
| `GET /api/custom_fields/` | OPT-OUT | not needed yet — no webspace concept maps to custom fields in v1; would become relevant only if keyword matching were extended beyond tags (a D-02 change). |
| `GET /api/saved_views/` | OPT-OUT | not needed — webspaces defines its own correlation in `config.toml` (KERN-01); importing paperless saved views would be a competing, source-specific definition of a webspace. |
| `GET /api/statistics/` | OPT-OUT | not needed — no aggregate/dashboard surface exists in Phase 1 (UI-01 is stream + detail pane only). |
| `GET /api/tasks/` — consumption task status | OPT-OUT | not needed — webspaces reads settled documents; paperless ingestion progress is the paperless UI's job. |
| `GET /api/remote_version/` | OPT-OUT | not needed yet — API-version pinning is handled by the `Accept` header. A version probe is a candidate for the Phase 2 plugin health report (PLUG-04). |
| `GET /api/users/`, `/api/groups/`, `/api/profile/` | OPT-OUT | not needed — single-user local tool with no permission model against paperless; AGENT-01 permissions (Phase 2) are enforced kernel-side, not by paperless identity. |
| `GET/POST /api/ui_settings/` | OPT-OUT | not needed — paperless UI preferences have no bearing on the webspaces UI. |
| `GET/POST /api/mail_accounts/`, `/api/mail_rules/` | OPT-OUT | not needed — paperless's own mail ingestion; webspaces reads Proton mail directly via IMAP in Phase 3 (SRC-01). |
| `POST /api/documents/post_document/` — upload | OPT-OUT | explicitly out of scope, permanently — plugins must never mutate source data stores (PROJECT.md `## Constraints`); no mutating RPC exists in `plugin.proto` to carry it (PLUG-02). |
| `POST /api/documents/bulk_edit/`, `/api/bulk_edit_objects/` | OPT-OUT | explicitly out of scope, permanently — same read-only-by-construction constraint. |
| `PUT/PATCH/DELETE /api/documents/{id}/` | OPT-OUT | explicitly out of scope, permanently — same read-only-by-construction constraint. |
| `POST/PUT/DELETE` on any taxonomy endpoint (tags, correspondents, etc.) | OPT-OUT | explicitly out of scope, permanently — same read-only-by-construction constraint. |
| `POST/DELETE /api/documents/{id}/notes/` | OPT-OUT | explicitly out of scope, permanently — same read-only-by-construction constraint. |
| `POST /api/share_links/` — create a public share link | OPT-OUT | explicitly out of scope, permanently — mutating, and it would publish personal content outside the machine, contradicting the privacy constraint. |
| `POST /api/acknowledge_tasks/` | OPT-OUT | explicitly out of scope, permanently — same read-only-by-construction constraint. |

## Rollup

- INTEGRATE: 8
- OPT-OUT (not needed / not needed yet): 18
- OPT-OUT (explicitly out of scope, permanent): 9

Every `OPT-OUT` above carries a reason. No capability is un-decided.
