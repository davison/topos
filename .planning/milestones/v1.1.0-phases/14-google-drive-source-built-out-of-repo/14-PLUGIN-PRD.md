# topos-plugin-gdrive — Product Requirements Document

This is the sole hand-off document into the `topos-plugin-gdrive` repository.
It is addressed to a fresh agent context that has never seen the topos
kernel, web UI, or CLI entry points. Every sentence in this document is
justifiable from one of the four published inputs listed below, this
phase's locked decisions, or the design guidance this document carries
forward from the topos-side research that grounded it. If a design question
comes up that none of those three sources answers, that is a contract gap —
log it in `CONTRACT-GAPS.md` and take your own best-guess resolution. It is
never a reason to go looking for more context than you were given.

## Product Statement

`topos-plugin-gdrive` is a topos source plugin that makes a chosen Google
Drive folder appear as a source inside topos. It solves the same problem
every topos source plugin solves — pulling one more personal data silo into
a correlated, cross-source stream — for the specific case of documents an
operator keeps in Google Drive rather than a local folder, a wiki, or a
document-management system.

The plugin must satisfy four success criteria:

1. The operator authorizes once and the source keeps syncing across host
   restarts without ever needing to re-authorize.
2. Documents in the configured folder appear in the stream with previews,
   including Workspace-native documents (Docs/Sheets/Slides) rendered via
   export.
3. Every sync after the first is incremental — from the plugin's own
   perspective, against the Drive API, not necessarily in the shape of any
   single RPC response (see "Design Guidance" below for why this
   distinction matters).
4. The plugin loads through the host's external, untrusted plugin path, and
   every place its author found the published contract silent or unclear
   is written down in `CONTRACT-GAPS.md` as it is encountered.

## Inputs You May Read

Exactly four inputs are in scope for this project, and nothing else:

1. The vendored plugin contract document (`contract/plugin-contract.md` in
   this repository — a snapshot of the published, third-party-facing
   contract for a topos source plugin).
2. The vendored proto file (`contract/plugin.proto`) — the wire truth for
   the four RPCs (`Describe`, `Match`, `Fetch`, `Health`), `ExtrasField`,
   `ContentVariant`, and `LinkFidelity`.
3. The `github.com/davison/topos/sdk` Go module — the published Go-native
   surface a plugin imports to implement the contract.
4. The vendored mock plugin (`contract/mock/`) — a complete, working
   reference plugin built from exactly these same four inputs and nothing
   else, proof that a plugin author with no other access can build a
   working plugin from this document alone.

Reading anything else belonging to the project that publishes this
contract — its kernel, its web UI, its command-line entry points, or any
other source plugin it ships — invalidates this exercise, even if such a
thing happens to be reachable on the same machine this work is done on. A
question these four inputs cannot answer is a gap-log entry, never a reason
to go looking elsewhere for the answer.

## Locked Decisions

These five decisions are locked by the project that owns this contract.
This repository's own planning must not revisit them:

- **Authorization is a standalone CLI subcommand of the same binary.**
  Running the built binary with `auth` as its first argument opens the
  operator's browser, runs an OAuth loopback redirect, and stores the
  resulting token — entirely out-of-band from the plugin's normal
  serve-mode operation. The host that launches this plugin must never see
  or compose an OAuth URL of any kind; an unauthorized source surfaces
  through the plugin's own named health state (see "Health States" below),
  never through a URL rendered by the host's own interface. Authorization
  is something the operator runs directly in a terminal, not something the
  host launches or drives.
- **The refresh token persists in a plugin-owned file.** A JSON token file,
  mode `0600`, lives under the operator's XDG data directory (e.g.
  `~/.local/share/topos-plugin-gdrive/token.json` on Linux) — a location
  this plugin owns and manages entirely itself. This works headlessly,
  with no D-Bus or OS keyring dependency, because a launched plugin
  subprocess receives a deliberately reduced environment with no
  guaranteed desktop-session plumbing beyond what the published contract's
  launch-environment section documents. Where a plugin keeps its own
  private state that must survive process restarts is undefined by the
  published contract — this is the first entry in this repository's own
  gap log, recorded before a line of plugin code exists.
- **Setup documentation must require the operator to publish their own
  Cloud Console OAuth application to Production status.** Publishing status
  and verification status are two different things, and setup
  documentation must say so explicitly: an unverified, personal-use app
  published to Production is fine — the operator clicks through a one-time
  "unverified app" consent warning. A Testing-status app is not fine: it
  silently expires every refresh token after seven days, which breaks
  success criterion 1 exactly one week after the operator thinks they are
  done.
- **The auth subcommand reads the OAuth client id and secret from the
  operator's shell environment, under the names `GDRIVE_CLIENT_ID` and
  `GDRIVE_CLIENT_SECRET`.** These are the same two environment variable
  names the source's declared `extras` fields reference (see "Declared
  Configuration" below) — one environment-variable vocabulary end to end,
  so an operator who has exported these two variables for the source's
  configuration needs no second, differently-named export for the
  standalone auth command.
- **The module path is `github.com/davison/topos-plugin-gdrive`.** This
  repository's public home names its own module path; there is no
  ambiguity or discretion here.

## Declared Configuration

`Describe` must declare exactly these three `ExtrasField` entries, verbatim:

| `key` | `label` | `required` | `secret` | `placeholder` |
|-------|---------|------------|----------|---------------|
| `client_id` | `OAuth Client ID` | `true` | `true` | `GDRIVE_CLIENT_ID` |
| `client_secret` | `OAuth Client Secret` | `true` | `true` | `GDRIVE_CLIENT_SECRET` |
| `folder_id` | `Drive Folder ID` | `true` | `false` | `e.g. 1a2B3cD4EfGhIjKlmNoPQRstuVwxYZ — the id segment of the folder's Drive URL` |

The `secret` flag on `client_id` and `client_secret` is load-bearing, not
cosmetic. It routes both fields through the host's own secret-handling
input control, which stores and displays only the name of the environment
variable a value comes from — never the value itself. This is the
project's established, non-negotiable convention for any credential-shaped
configuration value: secrets are environment-only references, never
literals in a saved configuration file or a rendered form field.

The instance targets its root by Drive folder id, not by a path. A folder
id (the segment inside `https://drive.google.com/drive/folders/<id>`) is
copy-pasteable, is unambiguous across My Drive, Shared Drives, and
shortcuts, and needs no path-resolution logic of any kind — a path is
ambiguous the moment two folders anywhere in a Drive share a name.

No row is added anywhere in the topos project's own source for this
plugin — no hardcoded connection-fields entry, no plugin-name-keyed branch
of any kind. This is the entire point of the mechanism this plugin
exercises: the host renders these three fields, their labels, their
placeholders, and their secret-masking purely from the `Describe` response
above. If this plugin's `Describe` implementation is correct, the host
needs zero code written specifically for it.

## Match Vocabulary

`Describe.match_vocabulary` must declare exactly one entry: `folders`.

Match values are the item's resolved folder-path segments relative to the
configured root folder, expressed as exact literals — never globs, never
prefix or substring matching. "Everything synced by this instance" is
expressed by matching against the root folder's own name, not by any
special-case empty-string or wildcard value. This mirrors the shape
already established for other folder-scoped sources built against this
same published contract: a value like `Reports` or `Reports/2026` names a
resolved path segment, compared exact and case-insensitive, exactly as the
contract's `Match` semantics section specifies for every declared field.

## Health States

The plugin's mechanism for surfacing an unhealthy state must be a
plugin-internal named-state enum — one constant per distinguishable cause,
never a single generic "unhealthy" flag. `Health` returns `reachable:
false` with that state's exact sentence as `last_error`; `Match` fails with
a gRPC `Unavailable` status wrapping the identical cause text as its error
detail. Never invent new remedy language beyond what is specified below —
each sentence already names a real, existing affordance in the host's own
interface, and a plugin author's job is naming the cause honestly, not
composing new UI copy.

The four sentences below are the exact, verbatim text this plugin's health
states must produce:

| Cause | Exact text |
|-------|------------|
| Never authorized — no token file found | `Not authorized — run "topos-plugin-gdrive auth" in a terminal, then use this source's "Refresh now".` |
| Authorization expired or was revoked | `Authorization expired or was revoked — run "topos-plugin-gdrive auth" again, then use this source's "Refresh now".` |
| Rate-limited or over quota (transient) | `Rate limited by Google Drive — retrying automatically. No action needed.` |
| Configured folder no longer exists or access was removed | `The configured Drive folder is no longer accessible — check the folder still exists and is shared with this account.` |

Two of these causes — the second row above, and how finely Google's own
token-refresh error responses actually let a client distinguish "revoked
by the user" from "expired from inactivity" from "the OAuth application
itself was un-published or deleted" — are open questions this repository
must resolve for itself; see "Open Questions" below. Do not assume the
distinction is cleanly observable until you have verified it.

## Drive API Surface

This table names the Drive v3 API surface this plugin touches, each marked
`integrate` or `opt-out` with the reason for that disposition:

| Capability | Disposition | Reason |
|---|---|---|
| `files.list` | integrate | The baseline folder walk — establishing and maintaining the set of files under the configured root. |
| `files.get` | integrate | Per-item metadata retrieval and native-file byte fetch. |
| `files.export` | integrate | Workspace-native document preview (Docs/Sheets/Slides have no native bytes of their own to fetch — they must be exported to a concrete format). |
| `changes.getStartPageToken` | integrate | Establishes the starting point for the incremental delta feed. |
| `changes.list` | integrate | The incremental delta feed itself, polled on every sync after the first. |
| Shared Drive enumeration | left to this repository's own design decision | Not locked by the topos-side decisions this document hands off; decide and document your own choice here. |
| `permissions` | opt-out | Write-capable in general shape and unnecessary for a strictly read-only source. |
| `revisions` | opt-out | Version history has no analog in this plugin's item model. |
| `comments` / `replies` | opt-out | Not part of a document's preview or content in this plugin's scope. |
| Push-notification watch channels | opt-out | They require a publicly reachable webhook endpoint, which a desktop-local host process does not have. |

## Recommended Stack

Two Go modules were verified against the live Go module proxy and are
recommended for this plugin:

| Module | Verified version | Reason |
|---|---|---|
| `google.golang.org/api/drive/v3` | `v0.293.0` | Google's own generated Drive API v3 client — the standard choice whenever an official generated client exists, rather than hand-rolling REST calls against the raw HTTP API. |
| `golang.org/x/oauth2` | `v0.36.0` | The Go project's own OAuth2 client. Its `oauth2.TokenSource` interface is exactly what the Drive API client's own `option.WithTokenSource` expects natively. |

Both modules were checked for package legitimacy against their official
source repositories and approved: `google.golang.org/api` traces to the
official Google-owned generated-API-client project, and
`golang.org/x/oauth2` is maintained by the Go team itself as part of the
extended standard-library set. Both are the lowest-plausible-risk case for
a slopsquatted or hallucinated package name — verify their current versions
again against the module proxy when this repository actually pins its own
`go.mod`, since both are fast-moving, actively-published modules and some
drift since the versions above were confirmed should be expected.

No third dependency is recommended. The OAuth loopback listener itself is a
small handful of standard-library calls — a third-party CLI-OAuth helper
package is optional convenience, not a requirement, and would be a
dependency this repository would need to independently vet on its own,
outside this document's scope.

**Do not reintroduce an OS-keyring dependency for token storage.** An
earlier round of research for the project that owns this contract once
recommended a keyring-backed credential store; that recommendation is
superseded by the plugin-owned-file decision recorded above (this
document's own Locked Decisions section) and must not be resurrected here.
A keyring dependency would reintroduce exactly the D-Bus/session
availability problem the plugin-owned-file choice exists to avoid.

## Design Guidance

The following findings are carried forward from the research that grounded
this hand-off, each labelled with its own confidence:

- **The binary is dual-mode: it dispatches on its first argument.** An
  `auth` first argument runs the standalone authorization flow and exits;
  anything else — including no arguments at all, the way a plugin host
  process launches a plugin subprocess — falls through to the plugin's
  ordinary serve-mode path. **[Unproven]** whether a plugin host's own
  launch mechanism tolerates or is indifferent to this dispatch shape is
  not confirmed by any of the four published inputs; treat this as an
  early smoke test in this repository's own first plan, and log it as a
  gap-log candidate if it turns out not to work cleanly.
- **The OAuth loopback redirect must use the literal address `127.0.0.1`,
  not the name `localhost`.** The out-of-band copy/paste authorization flow
  for installed apps is fully deprecated industry-wide; the loopback
  redirect is the only currently-supported flow for a desktop/CLI OAuth
  client. Google's own guidance for this flow flags the `localhost` name
  form specifically as prone to client-side firewall problems the literal
  IP address avoids.
- **PKCE is a hardening recommendation, not a locked requirement.** It is
  strongly encouraged for installed-app OAuth flows by the identity
  provider this plugin authenticates against, but this document does not
  mandate it — decide for yourself whether to implement it, and record
  that decision.
- **The incremental change feed has no folder-scoping parameter — it
  reports changes across the operator's entire Drive.** A plugin scoped to
  one configured folder must resolve each changed file's current parent
  chain against its own locally-maintained folder-membership state to
  decide whether that change is even relevant to the configured root. A
  plugin that assumes the delta feed is pre-filtered to its own folder will
  either ingest unrelated content from elsewhere in the operator's Drive,
  or never notice when a file has moved out of the configured folder.
- **The host replaces a source's entire item set on every sync — matching
  never returns only a delta.** "Incremental" describes only this plugin's
  own traffic to the Drive API (a token-based delta poll instead of a full
  re-walk), never the shape of what this plugin hands back on any single
  `Match` call. This plugin must materialize its full current item set —
  built incrementally against the Drive API, but delivered in full — on
  every `Match` call, even though the underlying cache backing that
  materialization is itself updated incrementally.
- **`Describe` must be side-effect-free and must never validate credentials
  or call Drive.** The host's add-source flow trial-launches a plugin
  binary and calls `Describe` before the operator has finished typing
  configuration — sometimes before any credential value has even been
  entered. If `Describe` eagerly validates the OAuth client id/secret or
  attempts any Drive API call, the add-source flow breaks on its very
  first step for every operator. `Describe` must return only static
  identity, vocabulary, and extras-declaration data; defer all credential
  validation to `Match` and `Health`.
- **The export ceiling for a Workspace-native document is 10 MB — a lower
  and separate limit from the transport's own message-size ceiling.** A
  document whose export would exceed this cap, or a format this plugin
  declines to export, must return an unavailable result carrying a named
  reason rather than ever silently truncating a document's exported
  content.
- **Rate limiting and quota exhaustion get exponential backoff with
  jitter, and the rate-limited health sentence — never a crash and never a
  silently short item set.** A sync run that hits a transient rate limit
  or quota condition must retry, not fail the whole sync, and must not let
  the operator observe an incomplete item set that looks like an
  intentional result rather than a transient condition.

## Open Questions

Both open questions below are unresolved by the research that grounds this
hand-off. Neither is answered here — resolve each as your own repository's
first research step, and record your resolution (and, if it makes the
published contract look incomplete, a gap-log entry) once you have.

1. **Does a Desktop-app OAuth client type require its exact loopback
   redirect URI to be pre-registered in Cloud Console, or is it exempt from
   exact port matching?** Some guidance states a redirect URI "must exactly
   match one of the authorized redirect URIs" configured for a client; other
   guidance claims Desktop-app clients specifically are not required to
   configure the URI in Cloud Console at all (loopback ports exempted).
   These two claims are in tension and neither is asserted as correct here —
   verify against a real Cloud Console project before finalizing this
   plugin's redirect-handling code.
2. **Which of the plausible auth-failure causes can this plugin actually
   tell apart from Google's token-refresh error responses alone?** The
   mechanism for surfacing a named auth-failure health state is settled
   (see "Health States" above), but whether a single `invalid_grant`-shaped
   error response lets a caller distinguish "the user revoked access" from
   "the token expired from inactivity" from "the OAuth application itself
   was un-published or deleted" is not established by any of the four
   published inputs. This decides how many of the four health sentences
   above are actually reachable in practice — spike it against a real
   revoked or expired token before committing to a final implementation.

## Prohibitions

This plugin must not do any of the following:

1. **Must not specify or permit storing any Google Drive document's full
   content in the host's own local index.** This plugin returns metadata
   and a bounded preview; full content is always fetched live from Drive
   at the moment an operator opens an item, never persisted locally beyond
   that bounded preview.
2. **Must not specify or permit a literal OAuth client secret or refresh
   token value appearing in a configuration file or a log line.**
   Configuration carries only environment references; logs carry only a
   secret's presence or its name, never its value.
3. **Must not request or use any Google Drive OAuth scope capable of
   writing to, moving, renaming, trashing, or deleting the operator's
   Drive data.** This plugin is read-only by construction, as the
   published contract requires of every source plugin — prefer the
   most narrowly read-only scope Google's own Drive API offers.
4. **Must not require any special-casing anywhere in the project that owns
   this contract** — no hardcoded connection-fields entry, no
   plugin-name-keyed copy branch, no name-keyed behavior of any kind. This
   plugin's entire configuration surface must be renderable by the host's
   already-published, fully generic mechanism, with zero code written
   specifically for this plugin's existence.
5. **Must not omit, soften, or retroactively rewrite a gap-log entry.**
   Every question the four published inputs cannot answer gets recorded in
   `CONTRACT-GAPS.md` as it is actually encountered, including any entry
   that makes the published contract look incomplete. The answer to "the
   contract does not say" is never to go and look somewhere else for the
   answer — it is always a gap-log entry.

## Build, Install and Verify

Build a single static binary named for this plugin. Install it into the
host's external plugin directory — the operating-system-appropriate data
path the published contract documents (an XDG-style data directory on
Linux, with platform-appropriate equivalents elsewhere). Consent to and pin
this plugin through the host's own source-picker flow — this plugin never
self-declares its own trust level; the host derives trust purely from
which directory a binary's bytes were found in, plus a content-hash pin an
operator explicitly accepts the first time a source is added using this
binary.

Run this plugin against a non-production host configuration while
developing and testing it. The host project this contract belongs to
provides a `--config` flag and a `TOPOS_CONFIG` environment variable for
exactly this purpose — pointing a development host instance at a
throwaway configuration, index, and plugin directory, so a development run
never touches an operator's real, production configuration or index.

## Deliverables Back to topos

Two things return to the project that owns this published contract:

1. **The built binary.**
2. **The gap log** — `CONTRACT-GAPS.md`, in whatever state it is in when
   this repository's own first working build exists. The project that owns
   this contract pulls this file back for triage: gaps that are
   fixable by documentation alone are republished into the contract
   in-phase; gaps that require a contract or wire-level change become
   backlog items tracked separately, for a future developer-guide and
   certification effort. Every gap-log entry matters to that triage, even
   the ones that make the published contract look incomplete — especially
   those.

## Explicitly Out of Scope

The following are explicitly out of scope for this repository:

- **Pull-by-URL distribution.** This plugin is installed by an operator
  manually placing its built binary into the host's external plugin
  directory — a future pull-by-URL install mechanism is not this
  repository's concern.
- **A OneDrive sibling plugin.** A structurally similar Microsoft Graph
  source is a plausible future project, but it is not part of this
  repository's scope.
- **Any change to the wire contract itself.** This repository consumes the
  four published inputs as given. A gap this repository finds may become a
  future contract change — but that change, if it happens, happens in the
  project that owns the contract, not here.
